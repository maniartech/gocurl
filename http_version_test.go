package gocurl

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/maniartech/gocurl/options"
)

// TestHTTPVersion_Parser verifies --http1.0/--http1.1/-0 set the right field and
// that the version flags are mutually exclusive with curl's last-flag-wins rule.
func TestHTTPVersion_Parser(t *testing.T) {
	const url = "https://example.com"
	cases := []struct {
		name                 string
		args                 []string
		h2, h2only, h11, h10 bool
	}{
		{"http1.1", []string{"--http1.1", url}, false, false, true, false},
		{"http1.0", []string{"--http1.0", url}, false, false, false, true},
		{"-0 alias", []string{"-0", url}, false, false, false, true},
		{"http2", []string{"--http2", url}, true, false, false, false},
		{"last-wins http2 then http1.1", []string{"--http2", "--http1.1", url}, false, false, true, false},
		{"last-wins http1.1 then http2", []string{"--http1.1", "--http2", url}, true, false, false, false},
		{"last-wins http2-only then http1.0", []string{"--http2-only", "--http1.0", url}, false, false, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o, err := argsToOptions(c.args)
			if err != nil {
				t.Fatalf("argsToOptions: %v", err)
			}
			if o.HTTP2 != c.h2 || o.HTTP2Only != c.h2only || o.HTTP11 != c.h11 || o.HTTP10 != c.h10 {
				t.Fatalf("got HTTP2=%t HTTP2Only=%t HTTP11=%t HTTP10=%t; want %t/%t/%t/%t",
					o.HTTP2, o.HTTP2Only, o.HTTP11, o.HTTP10, c.h2, c.h2only, c.h11, c.h10)
			}
		})
	}
}

// h2CapableServer starts an httptest server that negotiates HTTP/2 over TLS and
// reflects the protocol it actually served plus whether the client asked to close.
func h2CapableServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "proto=%s close=%t", r.Proto, r.Close)
	}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv
}

// TestHTTPVersion_ForceHTTP11 proves --http1.1 suppresses h2 even against an
// h2-capable server: the negotiated protocol must be HTTP/1.1, not HTTP/2.0.
func TestHTTPVersion_ForceHTTP11(t *testing.T) {
	srv := h2CapableServer(t)

	// Baseline: with HTTP2 the same server serves h2 — confirms the server CAN do h2.
	baseline := &options.RequestOptions{URL: srv.URL, Insecure: true,
		TLSConfig: &tls.Config{InsecureSkipVerify: true}, HTTP2: true}
	if _, body, err := processForTest(context.Background(), baseline); err != nil {
		t.Fatalf("h2 baseline: %v", err)
	} else if !strings.Contains(body, "HTTP/2.0") {
		t.Fatalf("baseline did not negotiate h2 (got %q) — test server misconfigured", body)
	}

	opts := &options.RequestOptions{URL: srv.URL, Insecure: true,
		TLSConfig: &tls.Config{InsecureSkipVerify: true}, HTTP11: true}
	_, body, err := processForTest(context.Background(), opts)
	if err != nil {
		t.Fatalf("http1.1 request: %v", err)
	}
	if !strings.Contains(body, "proto=HTTP/1.1") {
		t.Fatalf("--http1.1 did not force HTTP/1.1; server saw %q", body)
	}
}

// TestHTTPVersion_HTTP10 proves --http1.0 pins HTTP/1.1 on the wire AND sends
// Connection: close (the reachable part of curl's 1.0 semantics).
func TestHTTPVersion_HTTP10(t *testing.T) {
	srv := h2CapableServer(t)
	opts := &options.RequestOptions{URL: srv.URL, Insecure: true, Silent: true,
		TLSConfig: &tls.Config{InsecureSkipVerify: true}, HTTP10: true}
	_, body, err := processForTest(context.Background(), opts)
	if err != nil {
		t.Fatalf("http1.0 request: %v", err)
	}
	if !strings.Contains(body, "proto=HTTP/1.1") {
		t.Fatalf("--http1.0 wire version should be HTTP/1.1; server saw %q", body)
	}
	if !strings.Contains(body, "close=true") {
		t.Fatalf("--http1.0 should send Connection: close; server saw %q", body)
	}
}

// TestHTTPVersion_HTTP10Warning verifies the caveat warning is printed to stderr
// unless --silent.
func TestHTTPVersion_HTTP10Warning(t *testing.T) {
	capture := func(silent bool) string {
		old := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		warnHTTP10Limitation(&options.RequestOptions{HTTP10: true, Silent: silent})
		w.Close()
		os.Stderr = old
		out, _ := io.ReadAll(r)
		return string(out)
	}
	if got := capture(false); !strings.Contains(got, "HTTP/1.0 is not supported") {
		t.Errorf("expected warning on stderr, got %q", got)
	}
	if got := capture(true); got != "" {
		t.Errorf("expected no warning when --silent, got %q", got)
	}
}

// TestHTTPVersion_ClientWithHTTP11 proves the reusable Client's WithHTTP11 option
// forces HTTP/1.1 through the Client.Do path.
func TestHTTPVersion_ClientWithHTTP11(t *testing.T) {
	srv := h2CapableServer(t)
	client, err := New(WithHTTP11(), WithInsecure(true))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close()

	resp, err := client.Curl(context.Background(), "curl "+srv.URL)
	if err != nil {
		t.Fatalf("client.Curl: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.Proto != "HTTP/1.1" || !strings.Contains(string(body), "proto=HTTP/1.1") {
		t.Fatalf("WithHTTP11 did not force HTTP/1.1: resp.Proto=%s body=%q", resp.Proto, body)
	}
}

// TestHTTPVersion_TransportKeyAndForce proves the version pin is in the cache key
// (so a 1.1-pinned transport never collides with an h2-capable one) and that the
// forced transport is shaped correctly.
func TestHTTPVersion_TransportKeyAndForce(t *testing.T) {
	def := &options.RequestOptions{URL: "https://x"}
	h2 := &options.RequestOptions{URL: "https://x", HTTP2: true}
	h11 := &options.RequestOptions{URL: "https://x", HTTP11: true}

	ks := map[string]string{"def": transportKey(def), "h2": transportKey(h2), "h11": transportKey(h11)}
	if ks["def"] == ks["h11"] || ks["h2"] == ks["h11"] || ks["def"] == ks["h2"] {
		t.Fatalf("transport keys must be distinct per HTTP version: %+v", ks)
	}

	rt, err := getRoundTripper(h11)
	if err != nil {
		t.Fatalf("getRoundTripper: %v", err)
	}
	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", rt)
	}
	if tr.ForceAttemptHTTP2 {
		t.Error("forced HTTP/1.1 transport must have ForceAttemptHTTP2=false")
	}
	if tr.TLSNextProto == nil || len(tr.TLSNextProto) != 0 {
		t.Errorf("forced HTTP/1.1 transport must have a non-nil EMPTY TLSNextProto, got %v", tr.TLSNextProto)
	}
}
