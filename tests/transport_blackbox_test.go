package tests

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
)

// Application use-case: a redirect-authorization hook (the seam an SSRF guard
// will use) blocks following a redirect to a disallowed host.
func TestClient_RedirectPolicyAllowHookBlocks(t *testing.T) {
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "SHOULD NOT REACH")
	}))
	defer internal.Close()
	entry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, internal.URL, http.StatusFound)
	}))
	defer entry.Close()

	internalHost := mustHost(t, internal.URL)
	c, err := gocurl.New(gocurl.WithRedirectPolicy(gocurl.RedirectPolicy{
		Follow: true,
		Max:    10,
		Allow: func(req *http.Request, via []*http.Request) error {
			if req.URL.Host == internalHost {
				return fmt.Errorf("redirect to %s blocked by policy", internalHost)
			}
			return nil
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if _, err := c.Curl(context.Background(), "curl "+entry.URL); err == nil {
		t.Error("expected redirect to the disallowed host to be blocked")
	}
}

// Application use-case: HTTP/2 round-trip over TLS.
func TestClient_HTTP2RoundTrip(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Proto)
	}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()

	// Trust the test server's self-signed cert via -k; HTTP/2 is on by default.
	c, err := gocurl.New(gocurl.WithInsecure(true))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	body, resp, err := c.CurlString(context.Background(), "curl "+srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.ProtoMajor != 2 {
		t.Errorf("ProtoMajor = %d, want 2 (HTTP/2)", resp.ProtoMajor)
	}
	if !strings.Contains(body, "HTTP/2") {
		t.Errorf("server saw proto %q, want HTTP/2", body)
	}
}

// Application use-case: a tuned, reusable client works end-to-end.
func TestClient_TunedTransport(t *testing.T) {
	es := newEchoServer(t)
	c, err := gocurl.New(
		gocurl.WithMaxIdleConnsPerHost(4),
		gocurl.WithMaxConnsPerHost(8),
		gocurl.WithIdleConnTimeout(30*time.Second),
		gocurl.WithResponseHeaderTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	resp, err := c.Curl(context.Background(), "curl "+es.URL)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status %d", resp.StatusCode)
	}
}

func mustHost(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u.Host
}
