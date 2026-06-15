package tests

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maniartech/gocurl"
)

// Application use-case: configure a Client once, prepare a request once, execute
// it many times (the parse-once/execute-many model).
func TestClient_PrepareOnceExecuteMany(t *testing.T) {
	es := newEchoServer(t)
	c, err := gocurl.New(gocurl.WithUserAgent("svc/1.0"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Shutdown(context.Background())

	req, err := c.Prepare("curl " + es.URL + "/resource")
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 25; i++ {
		resp, err := c.Do(context.Background(), req)
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("iteration %d: status %d", i, resp.StatusCode)
		}
	}
	if es.lastHeader.Get("User-Agent") != "svc/1.0" {
		t.Errorf("configured User-Agent not applied: %q", es.lastHeader.Get("User-Agent"))
	}
}

// Application use-case: one shared Client serving many concurrent requests
// (race-checked under `go test -race`).
func TestClient_ConcurrentReuse(t *testing.T) {
	// Plain server (no shared recording state) so the test exercises gocurl's
	// concurrency, not the helper's.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	c, _ := gocurl.New()
	defer c.Close()

	req, err := c.Prepare("curl -X POST -d ping=1 " + srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	const n = 40
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := c.Do(context.Background(), req)
			if err != nil {
				errs <- err
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent Do failed: %v", err)
	}
}

// Application use-case: a single prepared template re-bound with per-call
// variables (e.g. a per-tenant token) without re-writing the command.
func TestClient_TemplateWithVars(t *testing.T) {
	es := newEchoServer(t)
	c, _ := gocurl.New()
	defer c.Close()

	tmpl, err := c.PrepareNoEnv("curl -H 'Authorization: Bearer $TOKEN' " + es.URL)
	if err != nil {
		t.Fatal(err)
	}

	for _, tok := range []string{"tenant-A", "tenant-B"} {
		resp, err := c.Do(context.Background(), tmpl.WithVars(gocurl.Variables{"TOKEN": tok}))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if got := es.lastHeader.Get("Authorization"); got != "Bearer "+tok {
			t.Errorf("Authorization = %q, want Bearer %s", got, tok)
		}
	}
}

// Application use-case: configured client follows redirects for all requests.
func TestClient_ConfiguredFollowRedirects(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	defer final.Close()
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer redir.Close()

	c, _ := gocurl.New(gocurl.WithFollowRedirects(true))
	defer c.Close()

	body, resp, err := c.CurlString(context.Background(), "curl "+redir.URL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || !strings.Contains(body, "ok") {
		t.Errorf("configured follow-redirects failed: status=%d body=%q", resp.StatusCode, body)
	}
}
