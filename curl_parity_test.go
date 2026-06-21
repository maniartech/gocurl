package gocurl

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"
)

// wire is what a server actually received — the bytes a client put on the wire.
type wire struct {
	method, query, body string
	header              http.Header
}

// gocurlWire runs a curl command through gocurl's one-shot engine against a local
// server and returns the request the server saw. This is how we lock gocurl's
// wire behavior to curl's, hermetically, with no external curl needed.
func gocurlWire(t *testing.T, args ...string) wire {
	t.Helper()
	ch := make(chan wire, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		ch <- wire{r.Method, r.URL.RawQuery, string(b), r.Header.Clone()}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()
	resp, err := CurlArgs(context.Background(), append(args, srv.URL+"/p")...)
	if err != nil {
		t.Fatalf("CurlArgs(%v): %v", args, err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	select {
	case w := <-ch:
		return w
	case <-time.After(5 * time.Second):
		t.Fatal("gocurl made no request")
		return wire{}
	}
}

// TestCurlParity_Headers locks the header behavior that must match curl.
func TestCurlParity_Headers(t *testing.T) {
	// curl ALWAYS sends Accept: */* unless overridden.
	if got := gocurlWire(t).header.Get("Accept"); got != "*/*" {
		t.Errorf("default Accept = %q, want %q (curl always sends */*)", got, "*/*")
	}
	// A user -H Accept overrides it, exactly like curl.
	if got := gocurlWire(t, "-H", "Accept: application/json").header.Get("Accept"); got != "application/json" {
		t.Errorf("overridden Accept = %q, want application/json", got)
	}
	// User-Agent is present (gocurl/VERSION — the one intentional, documented deviation).
	if got := gocurlWire(t).header.Get("User-Agent"); got == "" {
		t.Error("User-Agent must always be sent (like curl)")
	}
}

// TestCurlParity_FormData locks curl's -d semantics: POST, fields joined with &,
// application/x-www-form-urlencoded content type, and Accept: */* still present.
func TestCurlParity_FormData(t *testing.T) {
	w := gocurlWire(t, "-d", "a=1", "-d", "b=2")
	if w.method != http.MethodPost {
		t.Errorf("method = %s, want POST", w.method)
	}
	if w.body != "a=1&b=2" {
		t.Errorf("body = %q, want %q", w.body, "a=1&b=2")
	}
	if ct := w.header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
	}
	if w.header.Get("Accept") != "*/*" {
		t.Error("Accept: */* must be sent on POST too")
	}
}

// TestCurlParity_AuthRefererCookie locks -u / -e / -b to curl's header output.
//
// NOTE on -u: gocurl INTENTIONALLY diverges from curl here — it fail-closes basic
// auth over plaintext HTTP (curl sends credentials in the clear). That is security
// winning over curl-parity, by design (see TestCurlParity_BasicAuthFailClosed). We
// opt in to insecure auth only to verify the Authorization header *format* matches
// curl once credentials ARE sent (over HTTPS, this happens with no opt-in).
func TestCurlParity_AuthRefererCookie(t *testing.T) {
	t.Setenv("GOCURL_ALLOW_INSECURE_AUTH", "1")
	if a := gocurlWire(t, "-u", "user:pass").header.Get("Authorization"); a != "Basic dXNlcjpwYXNz" {
		t.Errorf("Authorization = %q, want Basic dXNlcjpwYXNz", a)
	}
	if r := gocurlWire(t, "-e", "https://ref.example").header.Get("Referer"); r != "https://ref.example" {
		t.Errorf("Referer = %q, want https://ref.example", r)
	}
	if c := gocurlWire(t, "-b", "session=abc").header.Get("Cookie"); c != "session=abc" {
		t.Errorf("Cookie = %q, want session=abc", c)
	}
}

// TestCurlParity_BasicAuthFailClosed documents the one deliberate, security-driven
// divergence from curl: gocurl REFUSES to send basic-auth credentials over plaintext
// HTTP by default (curl sends them in the clear). "Top-notch security" wins over
// blind curl-parity here; the caller can opt back into curl's behavior explicitly.
func TestCurlParity_BasicAuthFailClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()
	_, err := CurlArgs(context.Background(), "-u", "user:pass", srv.URL)
	if err == nil {
		t.Fatal("gocurl must fail-close basic auth over plaintext HTTP (security divergence from curl)")
	}
}

// TestCurlParity_GMovesDataToQuery locks curl's -G: data moves to the query string
// and the method stays GET with no body.
func TestCurlParity_GMovesDataToQuery(t *testing.T) {
	w := gocurlWire(t, "-G", "-d", "q=hello")
	if w.method != http.MethodGet {
		t.Errorf("-G method = %s, want GET", w.method)
	}
	if w.body != "" {
		t.Errorf("-G body = %q, want empty (moved to query)", w.body)
	}
	if w.query != "q=hello" {
		t.Errorf("-G query = %q, want q=hello", w.query)
	}
}

// TestCurlParity_DifferentialVsRealCurl is the gold-standard guarantee: when a real
// curl binary is available, it fires the SAME command at curl and at gocurl against
// one server and asserts the wire requests match (modulo the documented User-Agent
// and the Accept-Encoding set). It self-skips where curl is absent or cannot reach
// the test server (sandboxed CI), so it never flakes the build.
func TestCurlParity_DifferentialVsRealCurl(t *testing.T) {
	curlBin, err := exec.LookPath("curl")
	if err != nil {
		t.Skip("real curl not available")
	}
	// gocurl fail-closes basic auth over plaintext HTTP (intentional security
	// divergence); allow it here so the -u case compares the header curl produces.
	t.Setenv("GOCURL_ALLOW_INSECURE_AUTH", "1")
	ch := make(chan wire, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		ch <- wire{r.Method, r.URL.RawQuery, string(b), r.Header.Clone()}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()
	url := srv.URL + "/p"

	cases := [][]string{
		{},
		{"-d", "a=1", "-d", "b=2"},
		{"-u", "user:pass"},
		{"-e", "https://ref.example"},
		{"-b", "session=abc"},
		{"-G", "-d", "q=hello"},
	}
	skipHdr := map[string]bool{"User-Agent": true, "Accept-Encoding": true, "Content-Length": true}
	for _, args := range cases {
		// real curl
		cmd := exec.Command(curlBin, append(append([]string{"-s", "-o", "/dev/null"}, args...), url)...)
		_ = cmd.Run()
		var cu wire
		select {
		case cu = <-ch:
		case <-time.After(5 * time.Second):
			t.Skipf("curl could not reach the test server here (sandbox); args=%v", args)
		}
		// gocurl
		resp, err := CurlArgs(context.Background(), append(append([]string{}, args...), url)...)
		if err != nil {
			t.Fatalf("gocurl CurlArgs(%v): %v", args, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		var go_ wire
		select {
		case go_ = <-ch:
		case <-time.After(5 * time.Second):
			t.Fatalf("gocurl made no request for args=%v", args)
		}

		if cu.method != go_.method {
			t.Errorf("args=%v METHOD curl=%s gocurl=%s", args, cu.method, go_.method)
		}
		if cu.query != go_.query {
			t.Errorf("args=%v QUERY curl=%q gocurl=%q", args, cu.query, go_.query)
		}
		if cu.body != go_.body {
			t.Errorf("args=%v BODY curl=%q gocurl=%q", args, cu.body, go_.body)
		}
		for k := range cu.header {
			if skipHdr[k] {
				continue
			}
			if cu.header.Get(k) != go_.header.Get(k) {
				t.Errorf("args=%v HEADER %s curl=%q gocurl=%q", args, k, cu.header.Get(k), go_.header.Get(k))
			}
		}
	}
}
