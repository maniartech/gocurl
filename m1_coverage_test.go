package gocurl

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/maniartech/gocurl/options"
)

func TestNew_WithProxyBuildsTransport(t *testing.T) {
	// Exercises buildOwnedTransport's proxy branch (no dial happens at New).
	c, err := New(WithProxy("http://127.0.0.1:9"))
	if err != nil {
		t.Fatalf("New with proxy: %v", err)
	}
	if c.httpClient.Transport == nil {
		t.Error("proxy transport not built")
	}
}

func TestDo_DecompressErrorBubbles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = io.WriteString(w, "this is not gzip data")
	}))
	defer srv.Close()

	c, _ := New()
	r, _ := c.Prepare("curl --compressed " + srv.URL)
	if _, err := c.Do(context.Background(), r); err == nil {
		t.Error("expected a decompression error to bubble up from Do")
	}
}

func TestConvenience_EarlyErrorReturns(t *testing.T) {
	c, _ := New()
	ctx := context.Background()
	bad := "curl --bogus-flag https://x"
	if _, _, err := c.CurlString(ctx, bad); err == nil {
		t.Error("CurlString should return the parse error")
	}
	if _, _, err := c.CurlBytes(ctx, bad); err == nil {
		t.Error("CurlBytes should return the parse error")
	}
	if _, _, err := c.CurlDownload(ctx, filepath.Join(t.TempDir(), "f"), bad); err == nil {
		t.Error("CurlDownload should return the parse error")
	}
}

func TestRequest_BuilderInitsNilMaps(t *testing.T) {
	// A Request whose options have nil Headers/QueryParams must still accept
	// builder methods (defensive initialization).
	r := &Request{opts: &options.RequestOptions{Method: "GET", URL: "https://x"}}
	if got := r.WithHeader("X", "1").opts.Headers.Get("X"); got != "1" {
		t.Errorf("WithHeader on nil headers: %q", got)
	}
	if got := r.AddHeader("Y", "2").opts.Headers.Get("Y"); got != "2" {
		t.Errorf("AddHeader on nil headers: %q", got)
	}
	if got := r.WithQuery("q", "1").opts.QueryParams.Get("q"); got != "1" {
		t.Errorf("WithQuery on nil query: %q", got)
	}
}

func TestNewRequest_OptionError(t *testing.T) {
	// A RequestOption that errors must fail NewRequest.
	if _, err := NewRequest("GET", "https://x", Query("", "v"), badOption()); err == nil {
		t.Error("expected NewRequest to surface a RequestOption error")
	}
}

func badOption() RequestOption {
	return func(*options.RequestOptions) error { return io.ErrUnexpectedEOF }
}

func TestPrepareVariants_ParseErrors(t *testing.T) {
	c := newTestClient(t)
	if _, err := c.PrepareNoEnv("curl --bogus https://x"); err == nil {
		t.Error("PrepareNoEnv should surface a parse error")
	}
	if _, err := c.PrepareWithVars(Variables{}, "curl --bogus https://x"); err == nil {
		t.Error("PrepareWithVars should surface a parse error")
	}
}

func TestPrepare_TokenizeError(t *testing.T) {
	c := newTestClient(t)
	if _, err := c.Prepare("curl -H 'unterminated https://x"); err == nil {
		t.Error("unmatched quote should produce a tokenize error")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestBodyReader_ReadError(t *testing.T) {
	if _, err := NewRequest("POST", "https://x", BodyReader(errReader{})); err == nil {
		t.Error("BodyReader should surface a read error")
	}
}

func TestNewRequest_DefaultMethod(t *testing.T) {
	r, err := NewRequest("", "https://x")
	if err != nil {
		t.Fatal(err)
	}
	if r.Method() != "GET" {
		t.Errorf("default method = %q, want GET", r.Method())
	}
}

func TestDo_ValidationError(t *testing.T) {
	// --cert without --key parses fine but fails validation at execution time.
	c, _ := New()
	r, err := c.Prepare("curl --cert only.pem https://example.com")
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if _, err := c.Do(context.Background(), r); err == nil {
		t.Error("Do should fail validation for cert without key")
	}
}

func TestDo_MaxTimeSuccessCancels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "fast")
	}))
	defer srv.Close()
	c, _ := New()
	r, _ := c.Prepare("curl --max-time 5 " + srv.URL)
	resp, err := c.Do(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	if string(b) != "fast" {
		t.Errorf("body = %q", b)
	}
	if err := resp.Body.Close(); err != nil { // triggers the cancel-on-close path
		t.Errorf("close: %v", err)
	}
}
