package gocurl

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// recordingServer captures the last request and replies with a fixed JSON body.
type recordingServer struct {
	*httptest.Server
	mu     sync.Mutex
	method string
	header http.Header
	body   string
}

func newRecordingServer(t *testing.T) *recordingServer {
	t.Helper()
	s := &recordingServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.method, s.header, s.body = r.Method, r.Header.Clone(), string(b)
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(s.Close)
	return s
}

func TestNew_Default(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if c.httpClient == nil || c.httpClient.Transport == nil {
		t.Fatal("client/transport not initialized")
	}
}

func TestNew_OptionError(t *testing.T) {
	if _, err := New(WithTimeout(-1)); err == nil {
		t.Error("expected option error to propagate from New")
	}
	if _, err := New(nil); err != nil { // nil option is skipped
		t.Errorf("nil option should be skipped, got %v", err)
	}
}

func TestNew_WithTransportAndCookieFile(t *testing.T) {
	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, nil })
	c, err := New(WithTransport(rt), WithCookieFile(filepath.Join(t.TempDir(), "c.txt")))
	if err != nil {
		t.Fatal(err)
	}
	if c.httpClient.Transport == nil {
		t.Error("custom transport not applied")
	}
	if c.httpClient.Jar == nil {
		t.Error("cookie jar not configured")
	}
}

func TestDo_Success(t *testing.T) {
	s := newRecordingServer(t)
	c, _ := New()
	r, err := c.Prepare("curl -X POST -d a=1 " + s.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status %d", resp.StatusCode)
	}
	if s.method != "POST" || s.body != "a=1" {
		t.Errorf("server saw method=%q body=%q", s.method, s.body)
	}
}

func TestDo_NilRequestAndClosed(t *testing.T) {
	c, _ := New()
	if _, err := c.Do(context.Background(), nil); err == nil {
		t.Error("nil request should error")
	}
	s := newRecordingServer(t)
	r, _ := c.Prepare("curl " + s.URL)
	_ = c.Close()
	if _, err := c.Do(context.Background(), r); err == nil {
		t.Error("Do on a closed client should error")
	}
}

func TestDo_PerRequestTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = io.WriteString(w, "late")
	}))
	defer slow.Close()
	c, _ := New()
	r, _ := c.Prepare("curl --max-time 0.05 " + slow.URL)
	if _, err := c.Do(context.Background(), r); err == nil {
		t.Error("expected per-request --max-time to fail the slow request")
	}
}

func TestDo_FollowRedirectsViaContextOnSharedClient(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "arrived")
	}))
	defer final.Close()
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redir.Close()

	c, _ := New() // one shared client for both requests

	noFollow, _ := c.Prepare("curl " + redir.URL)
	resp, err := c.Do(context.Background(), noFollow)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("without -L: status %d, want 302", resp.StatusCode)
	}

	follow, _ := c.Prepare("curl -L " + redir.URL)
	body, resp2, err := func() (string, *http.Response, error) {
		rr, e := c.Do(context.Background(), follow)
		if e != nil {
			return "", nil, e
		}
		defer rr.Body.Close()
		b, _ := io.ReadAll(rr.Body)
		return string(b), rr, nil
	}()
	if err != nil {
		t.Fatal(err)
	}
	if resp2.StatusCode != 200 || body != "arrived" {
		t.Errorf("with -L on shared client: status=%d body=%q", resp2.StatusCode, body)
	}
}

func TestEffectiveOptions_Merge(t *testing.T) {
	c, _ := New(WithUserAgent("cfg-ua"), WithDefaultHeader("X-Default", "d"), WithFollowRedirects(true))
	// Request without UA, without that header, without -L: config defaults apply.
	r, _ := c.Prepare("curl https://x")
	o := c.effectiveOptions(r)
	if o.UserAgent != "cfg-ua" {
		t.Errorf("UA = %q, want cfg-ua", o.UserAgent)
	}
	if o.Headers.Get("X-Default") != "d" {
		t.Errorf("default header not merged")
	}
	if !o.FollowRedirects || o.MaxRedirects != 30 {
		t.Errorf("redirect defaults not applied: %v %d", o.FollowRedirects, o.MaxRedirects)
	}
	// Request that sets its own UA wins over config.
	r2, _ := c.Prepare("curl -A req-ua https://x")
	if c.effectiveOptions(r2).UserAgent != "req-ua" {
		t.Errorf("request UA should win over config")
	}
}

func TestWithMiddleware_Runs(t *testing.T) {
	s := newRecordingServer(t)
	mw := func(next Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			req.Header.Set("X-Mw", "applied")
			return next(req)
		}
	}
	c, _ := New(WithMiddleware(mw))
	r, _ := c.Prepare("curl " + s.URL)
	resp, err := c.Do(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if s.header.Get("X-Mw") != "applied" {
		t.Errorf("middleware did not run; header=%v", s.header)
	}
}

func TestRedirectFromContext(t *testing.T) {
	mk := func(follow bool, max int) *http.Request {
		req, _ := http.NewRequest("GET", "http://x", nil)
		return req.WithContext(withRedirectSettings(context.Background(), redirectSettings{follow: follow, max: max}))
	}
	if err := redirectFromContext(mk(false, 0), nil); err != http.ErrUseLastResponse {
		t.Errorf("no-follow should return ErrUseLastResponse, got %v", err)
	}
	if err := redirectFromContext(mk(true, 5), make([]*http.Request, 1)); err != nil {
		t.Errorf("within max should be nil, got %v", err)
	}
	if err := redirectFromContext(mk(true, 2), make([]*http.Request, 2)); err == nil {
		t.Error("exceeding max should error")
	}
}

func TestConvenienceMethods(t *testing.T) {
	jsonSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"ada","n":7}`)
	}))
	defer jsonSrv.Close()
	c, _ := New()
	ctx := context.Background()

	if str, _, err := c.CurlString(ctx, "curl "+jsonSrv.URL); err != nil || !strings.Contains(str, "ada") {
		t.Errorf("CurlString: %q %v", str, err)
	}
	if b, _, err := c.CurlBytes(ctx, "curl "+jsonSrv.URL); err != nil || len(b) == 0 {
		t.Errorf("CurlBytes: %v", err)
	}
	var out struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	if _, err := c.CurlJSON(ctx, &out, "curl "+jsonSrv.URL); err != nil || out.Name != "ada" || out.N != 7 {
		t.Errorf("CurlJSON: %+v %v", out, err)
	}
	dst := filepath.Join(t.TempDir(), "d.json")
	if n, _, err := c.CurlDownload(ctx, dst, "curl "+jsonSrv.URL); err != nil || n == 0 {
		t.Errorf("CurlDownload: n=%d %v", n, err)
	}
}

func TestConvenience_ErrorPaths(t *testing.T) {
	textSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "not json")
	}))
	defer textSrv.Close()
	c, _ := New()
	ctx := context.Background()

	var v map[string]any
	if _, err := c.CurlJSON(ctx, &v, "curl "+textSrv.URL); err == nil {
		t.Error("CurlJSON on non-JSON should error")
	}
	// Bad parse → Curl returns error before execution.
	if _, err := c.Curl(ctx, "curl --bogus-flag https://x"); err == nil {
		t.Error("bad command should error")
	}
	// Download to an unwritable path.
	if _, _, err := c.CurlDownload(ctx, filepath.Join(t.TempDir(), "no-such-dir", "f"), "curl "+textSrv.URL); err == nil {
		t.Error("download to bad path should error")
	}
}

func TestShutdown_NoInflight(t *testing.T) {
	s := newRecordingServer(t)
	c, _ := New()
	r, _ := c.Prepare("curl " + s.URL)
	resp, err := c.Do(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if err := c.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown with no in-flight should be nil, got %v", err)
	}
}

func TestShutdown_TimesOutWithInflight(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // block until released
		_, _ = io.WriteString(w, "done")
	}))
	defer srv.Close()
	defer close(release)

	c, _ := New()
	r, _ := c.Prepare("curl " + srv.URL)

	started := make(chan struct{})
	go func() {
		close(started)
		resp, err := c.Do(context.Background(), r)
		if err == nil {
			resp.Body.Close()
		}
	}()
	<-started
	time.Sleep(20 * time.Millisecond) // let the request reach the server

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := c.Shutdown(ctx); err == nil {
		t.Error("Shutdown should time out while a request is in flight")
	}
}
