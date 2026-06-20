package gocurl

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"
)

// goroutinesAtMost polls runtime.NumGoroutine, giving transport/dialer goroutines
// time to wind down, and returns as soon as the count is <= target (or the
// deadline passes). It GCs between samples so finalizers/idle conns are reclaimed.
func goroutinesAtMost(target int, d time.Duration) int {
	deadline := time.Now().Add(d)
	for {
		runtime.GC()
		n := runtime.NumGoroutine()
		if n <= target || time.Now().After(deadline) {
			return n
		}
		runtime.Gosched()
		time.Sleep(10 * time.Millisecond)
	}
}

// TestClient_Do_NoGoroutineLeak drives a batch of Client.Do calls and asserts the
// goroutine count returns near baseline once every body is closed and the client
// is closed — a per-request leak (dangling dialer/redirect/read goroutines) would
// scale with the request count and blow past the small margin.
func TestClient_Do_NoGoroutineLeak(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c, err := New()
	if err != nil {
		t.Fatal(err)
	}

	do := func(n int) {
		for i := 0; i < n; i++ {
			req, err := NewRequest("GET", srv.URL)
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
	}

	do(20) // warm up persistent transport goroutines
	runtime.GC()
	base := goroutinesAtMost(0, 200*time.Millisecond) // brief settle; target 0 forces the full window

	do(200)
	if err := c.Close(); err != nil { // close idle keep-alive conns (client side)
		t.Fatal(err)
	}
	srv.CloseClientConnections() // and server side

	final := goroutinesAtMost(base+15, 3*time.Second)
	if final > base+15 {
		t.Errorf("goroutine leak: base=%d final=%d (delta %d after 200 requests)", base, final, final-base)
	}
}

// TestClient_Do_ReusesConnections asserts the pooled transport reuses one
// keep-alive connection across many sequential requests rather than dialing a
// fresh TCP connection each time.
func TestClient_Do_ReusesConnections(t *testing.T) {
	var mu sync.Mutex
	newConns := 0

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	// ConnState must be installed before Start.
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			mu.Lock()
			newConns++
			mu.Unlock()
		}
	}
	srv.Start()
	defer srv.Close()

	c, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	const N = 50
	for i := 0; i < N; i++ {
		req, err := NewRequest("GET", srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := c.Do(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body) // drain so the conn is returned to the pool
		resp.Body.Close()
	}

	mu.Lock()
	got := newConns
	mu.Unlock()
	if got >= 5 {
		t.Errorf("opened %d new connections for %d sequential requests; expected keep-alive reuse (~1)", got, N)
	}
}
