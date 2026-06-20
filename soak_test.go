package gocurl

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

// TestClient_Soak runs a sustained, bounded request loop against an httptest
// server and asserts zero errors and a stable goroutine count across the run. It
// is skipped in -short mode (CI). When GOCURL_PROFILE=<dir> is set it writes
// cpu.pprof and mem.pprof for offline `go tool pprof` analysis.
//
//	GOCURL_PROFILE=$(mktemp -d) go test -run TestClient_Soak .
func TestClient_Soak(t *testing.T) {
	if testing.Short() {
		t.Skip("soak test skipped in -short mode")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	profileDir := os.Getenv("GOCURL_PROFILE")
	if profileDir != "" {
		f, err := os.Create(filepath.Join(profileDir, "cpu.pprof"))
		if err != nil {
			t.Fatalf("create cpu profile: %v", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			t.Fatalf("start cpu profile: %v", err)
		}
		defer func() {
			pprof.StopCPUProfile()
			f.Close()
		}()
	}

	// Warm up, then snapshot a baseline goroutine count.
	for i := 0; i < 20; i++ {
		drainOnce(t, c, srv.URL)
	}
	runtime.GC()
	base := goroutinesAtMost(0, 200*time.Millisecond)

	const iters = 3000
	for i := 0; i < iters; i++ {
		drainOnce(t, c, srv.URL)
	}

	if profileDir != "" {
		f, err := os.Create(filepath.Join(profileDir, "mem.pprof"))
		if err != nil {
			t.Fatalf("create mem profile: %v", err)
		}
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			t.Fatalf("write heap profile: %v", err)
		}
		f.Close()
	}

	final := goroutinesAtMost(base+15, 3*time.Second)
	if final > base+15 {
		t.Errorf("goroutine growth across soak: base=%d final=%d (delta %d over %d requests)", base, final, final-base, iters)
	}
}

func drainOnce(t *testing.T, c *Client, url string) {
	t.Helper()
	req, err := NewRequest("GET", url)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}
