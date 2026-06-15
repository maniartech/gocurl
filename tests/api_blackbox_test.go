package tests

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maniartech/gocurl"
)

// echoServer records the last request it received and replies with a fixed body.
type echoServer struct {
	*httptest.Server
	lastMethod string
	lastPath   string
	lastQuery  string
	lastHeader http.Header
	lastBody   string
}

func newEchoServer(t *testing.T) *echoServer {
	t.Helper()
	es := &echoServer{}
	es.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		es.lastMethod = r.Method
		es.lastPath = r.URL.Path
		es.lastQuery = r.URL.RawQuery
		es.lastHeader = r.Header.Clone()
		es.lastBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(es.Close)
	return es
}

func ctx() context.Context { return context.Background() }

func mustString(t *testing.T, args ...string) (string, *http.Response) {
	t.Helper()
	body, resp, err := gocurl.CurlString(ctx(), args...)
	if err != nil {
		t.Fatalf("CurlString(%v): %v", args, err)
	}
	return body, resp
}

func TestBlackbox_DefaultGET(t *testing.T) {
	es := newEchoServer(t)
	_, resp := mustString(t, es.URL)
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if es.lastMethod != "GET" {
		t.Errorf("method = %q, want GET", es.lastMethod)
	}
}

func TestBlackbox_MethodPOSTAndHead(t *testing.T) {
	es := newEchoServer(t)
	mustString(t, "-X", "POST", es.URL)
	if es.lastMethod != "POST" {
		t.Errorf("-X POST -> method %q", es.lastMethod)
	}
	mustString(t, "-I", es.URL)
	if es.lastMethod != "HEAD" {
		t.Errorf("-I -> method %q", es.lastMethod)
	}
}

func TestBlackbox_Headers(t *testing.T) {
	es := newEchoServer(t)
	mustString(t, "-H", "X-One: 1", "-H", "X-Two: two", es.URL)
	if es.lastHeader.Get("X-One") != "1" || es.lastHeader.Get("X-Two") != "two" {
		t.Errorf("headers not sent: %v", es.lastHeader)
	}
}

func TestBlackbox_DataBodyAndContentType(t *testing.T) {
	es := newEchoServer(t)
	mustString(t, "-d", "a=1", "-d", "b=2", es.URL)
	if es.lastMethod != "POST" {
		t.Errorf("data -> method %q, want POST", es.lastMethod)
	}
	if es.lastBody != "a=1&b=2" {
		t.Errorf("body = %q, want a=1&b=2", es.lastBody)
	}
	if ct := es.lastHeader.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
		t.Errorf("content-type = %q", ct)
	}
}

func TestBlackbox_JSONBodyVerbatim(t *testing.T) {
	es := newEchoServer(t)
	// Single-string command with a quoted JSON body, as copied from API docs.
	cmd := `curl -X POST -H 'Content-Type: application/json' -d '{"name":"ada","n":42}' ` + es.URL
	if _, _, err := gocurl.CurlString(ctx(), cmd); err != nil {
		t.Fatal(err)
	}
	if es.lastBody != `{"name":"ada","n":42}` {
		t.Errorf("JSON body corrupted: %q", es.lastBody)
	}
}

func TestBlackbox_GetModeQuery(t *testing.T) {
	es := newEchoServer(t)
	mustString(t, "-G", "-d", "q=go", "-d", "p=2", es.URL)
	if es.lastMethod != "GET" {
		t.Errorf("method = %q, want GET", es.lastMethod)
	}
	if !strings.Contains(es.lastQuery, "q=go") || !strings.Contains(es.lastQuery, "p=2") {
		t.Errorf("query = %q", es.lastQuery)
	}
}

func TestBlackbox_BasicAuth(t *testing.T) {
	es := newEchoServer(t)
	mustString(t, "-u", "alice:secret", es.URL)
	user, pass, ok := parseBasicAuth(es.lastHeader.Get("Authorization"))
	if !ok || user != "alice" || pass != "secret" {
		t.Errorf("basic auth = %q (%s/%s)", es.lastHeader.Get("Authorization"), user, pass)
	}
}

func TestBlackbox_DefaultUserAgent(t *testing.T) {
	es := newEchoServer(t)
	mustString(t, es.URL)
	if ua := es.lastHeader.Get("User-Agent"); !strings.HasPrefix(ua, "gocurl/") {
		t.Errorf("user-agent = %q, want gocurl/ prefix", ua)
	}
	mustString(t, "-A", "custom-agent/9", es.URL)
	if ua := es.lastHeader.Get("User-Agent"); ua != "custom-agent/9" {
		t.Errorf("user-agent = %q, want custom-agent/9", ua)
	}
}

func TestBlackbox_QueryInURLPreserved(t *testing.T) {
	es := newEchoServer(t)
	mustString(t, es.URL+"/search?term=hello+world&lang=go")
	if !strings.Contains(es.lastQuery, "term=hello+world") || !strings.Contains(es.lastQuery, "lang=go") {
		t.Errorf("query not preserved: %q", es.lastQuery)
	}
}

func TestBlackbox_FollowRedirects(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("arrived"))
	}))
	defer final.Close()
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redir.Close()

	// Without -L: not followed (302 returned).
	_, resp := mustString(t, redir.URL)
	if resp.StatusCode != http.StatusFound {
		t.Errorf("without -L: status = %d, want 302", resp.StatusCode)
	}
	// With -L: followed to final.
	body, resp := mustString(t, "-L", redir.URL)
	if resp.StatusCode != 200 || body != "arrived" {
		t.Errorf("with -L: status=%d body=%q, want 200/arrived", resp.StatusCode, body)
	}
}

func TestBlackbox_JSONHelper(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"name":"ada","n":42}`))
	}))
	defer srv.Close()

	var out struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	if _, err := gocurl.CurlJSON(ctx(), &out, srv.URL); err != nil {
		t.Fatal(err)
	}
	if out.Name != "ada" || out.N != 42 {
		t.Errorf("decoded = %+v", out)
	}
}

func TestBlackbox_BytesHelper(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("raw-bytes"))
	}))
	defer srv.Close()
	b, _, err := gocurl.CurlBytes(ctx(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "raw-bytes" {
		t.Errorf("bytes = %q", b)
	}
}

func TestBlackbox_DownloadStreamsToFile(t *testing.T) {
	payload := strings.Repeat("x", 512*1024) // 512KB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, payload)
	}))
	defer srv.Close()

	dst := filepath.Join(t.TempDir(), "out.bin")
	n, _, err := gocurl.CurlDownload(ctx(), dst, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if n != int64(len(payload)) {
		t.Errorf("wrote %d bytes, want %d", n, len(payload))
	}
	got, _ := os.ReadFile(dst)
	if len(got) != len(payload) {
		t.Errorf("file size = %d, want %d", len(got), len(payload))
	}
}

func TestBlackbox_EnvVarExpansion(t *testing.T) {
	es := newEchoServer(t)
	t.Setenv("GC_BB_TOKEN", "tok-123")
	cmd := `curl -H 'Authorization: Bearer $GC_BB_TOKEN' ` + es.URL
	if _, _, err := gocurl.CurlString(ctx(), cmd); err != nil {
		t.Fatal(err)
	}
	if got := es.lastHeader.Get("Authorization"); got != "Bearer tok-123" {
		t.Errorf("Authorization = %q, want Bearer tok-123", got)
	}
}

func TestBlackbox_WithVarsNoEnvLeak(t *testing.T) {
	es := newEchoServer(t)
	t.Setenv("GC_BB_SECRET", "leaked")
	// Explicit vars map (empty): must not pull GC_BB_SECRET from the environment.
	resp, err := gocurl.CurlWithVars(ctx(), gocurl.Variables{},
		"-H", "Authorization: Bearer $GC_BB_SECRET", es.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got := es.lastHeader.Get("Authorization"); strings.Contains(got, "leaked") {
		t.Errorf("env leaked into WithVars path: %q", got)
	}
}

func TestBlackbox_GzipDecompression(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		_, _ = gz.Write([]byte("compressed-payload"))
		_ = gz.Close()
	}))
	defer srv.Close()

	body, _, err := gocurl.CurlString(ctx(), "--compressed", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if body != "compressed-payload" {
		t.Errorf("body = %q, want decompressed payload", body)
	}
}

func TestBlackbox_NoStdoutSideEffect(t *testing.T) {
	es := newEchoServer(t)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	_, _, err := gocurl.CurlString(ctx(), es.URL)
	_ = w.Close()
	os.Stdout = old
	if err != nil {
		t.Fatal(err)
	}
	captured, _ := io.ReadAll(r)
	if len(captured) != 0 {
		t.Errorf("library wrote to stdout: %q", captured)
	}
}

func TestBlackbox_Errors(t *testing.T) {
	if _, err := gocurl.Curl(ctx()); err == nil {
		t.Error("expected error for no command")
	}
	if _, err := gocurl.Curl(ctx(), "--definitely-unknown-flag", "https://example.com"); err == nil {
		t.Error("expected error for unknown flag")
	}
}

// parseBasicAuth decodes a "Basic base64(user:pass)" header.
func parseBasicAuth(h string) (user, pass string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(h, prefix) {
		return "", "", false
	}
	req := &http.Request{Header: http.Header{"Authorization": {h}}}
	return req.BasicAuth()
}
