package tests

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
)

// flakyServer fails the first failN requests with failStatus, then returns 200.
type flakyServer struct {
	*httptest.Server
	hits        int32
	remoteAddrs []string
	mu          sync.Mutex
}

func newFlakyServer(t *testing.T, failN int, failStatus int) *flakyServer {
	t.Helper()
	s := &flakyServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&s.hits, 1)
		s.mu.Lock()
		s.remoteAddrs = append(s.remoteAddrs, r.RemoteAddr)
		s.mu.Unlock()
		io.Copy(io.Discard, r.Body)
		if int(n) <= failN {
			w.WriteHeader(failStatus)
			return
		}
		w.WriteHeader(200)
		io.WriteString(w, "ok")
	}))
	t.Cleanup(s.Close)
	return s
}

func zeroBackoffPolicy(maxAttempts int, statuses ...int) gocurl.RetryPolicy {
	return gocurl.RetryPolicy{MaxAttempts: maxAttempts, Backoff: gocurl.ConstantBackoff(0), RetryOnStatus: statuses}
}

func TestResilience_GET503Retried(t *testing.T) {
	srv := newFlakyServer(t, 1, 503)
	c, _ := gocurl.New(gocurl.WithRetry(zeroBackoffPolicy(3, 503)))
	defer c.Close()

	resp, err := c.Curl(context.Background(), "curl "+srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || atomic.LoadInt32(&srv.hits) != 2 {
		t.Errorf("status=%d hits=%d, want 200 and 2", resp.StatusCode, srv.hits)
	}
}

func TestResilience_POSTNotRetried(t *testing.T) {
	srv := newFlakyServer(t, 1, 503)
	c, _ := gocurl.New(gocurl.WithRetry(zeroBackoffPolicy(3, 503)))
	defer c.Close()

	req, err := gocurl.NewRequest("POST", srv.URL, gocurl.Body([]byte("payload")))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 503 || atomic.LoadInt32(&srv.hits) != 1 {
		t.Errorf("status=%d hits=%d, want 503 and 1 (POST not retried by default)", resp.StatusCode, srv.hits)
	}
}

func TestResilience_POSTWithIdempotencyKeyRetried(t *testing.T) {
	srv := newFlakyServer(t, 1, 503)
	c, _ := gocurl.New(gocurl.WithRetry(zeroBackoffPolicy(3, 503)))
	defer c.Close()

	req, _ := gocurl.NewRequest("POST", srv.URL, gocurl.Body([]byte("payload")), gocurl.Header("Idempotency-Key", "abc-123"))
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || atomic.LoadInt32(&srv.hits) != 2 {
		t.Errorf("status=%d hits=%d, want 200 and 2 (Idempotency-Key opts POST into retries)", resp.StatusCode, srv.hits)
	}
}

func TestResilience_LegacyRetryFlagPOSTRetried(t *testing.T) {
	// The --retry flag drives the legacy (method-agnostic) path via the one-shot
	// CurlString API, so a POST is retried.
	srv := newFlakyServer(t, 1, 502)
	c, _ := gocurl.New()
	defer c.Close()

	_, resp, err := c.CurlString(context.Background(), "curl --retry 2 -d payload "+srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || atomic.LoadInt32(&srv.hits) != 2 {
		t.Errorf("status=%d hits=%d, want 200 and 2 (legacy --retry POST retried)", resp.StatusCode, srv.hits)
	}
}

func TestResilience_KeepAliveReuseAfterDrain(t *testing.T) {
	// A discarded retry attempt must drain+close its body so the keep-alive
	// connection is reused: both attempts share a RemoteAddr.
	srv := newFlakyServer(t, 1, 503)
	c, _ := gocurl.New(gocurl.WithRetry(zeroBackoffPolicy(2, 503)))
	defer c.Close()

	resp, err := c.Curl(context.Background(), "curl "+srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if len(srv.remoteAddrs) != 2 {
		t.Fatalf("expected 2 server hits, got %d", len(srv.remoteAddrs))
	}
	if srv.remoteAddrs[0] != srv.remoteAddrs[1] {
		t.Errorf("connection not reused across retry: %q vs %q (drain+close regression)", srv.remoteAddrs[0], srv.remoteAddrs[1])
	}
}

func TestResilience_CircuitBreakerFastFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c, _ := gocurl.New(gocurl.WithCircuitBreaker(gocurl.BreakerConfig{
		MinRequests: 3, FailureThreshold: 0.5, Window: time.Minute, OpenTimeout: time.Minute,
	}))
	defer c.Close()
	ctx := context.Background()

	// 3 failing requests trip the breaker (no retry, so each is one final 500).
	for i := 0; i < 3; i++ {
		resp, err := c.Curl(ctx, "curl "+srv.URL)
		if err != nil {
			t.Fatalf("request %d unexpectedly errored before tripping: %v", i, err)
		}
		resp.Body.Close()
	}
	// Next request fast-fails with ErrCircuitOpen.
	_, err := c.Curl(ctx, "curl "+srv.URL)
	if !errors.Is(err, gocurl.ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
	if gocurl.IsRetryable(err) {
		t.Error("ErrCircuitOpen must be non-retryable")
	}
}

func TestResilience_RateLimiterPaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	c, _ := gocurl.New(gocurl.WithRateLimit(50, 1)) // 1 burst, then ~20ms/token
	defer c.Close()
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 3; i++ {
		resp, err := c.Curl(ctx, "curl "+srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}
	// 3 requests, burst 1 => 2 waits of ~20ms each.
	if elapsed := time.Since(start); elapsed < 25*time.Millisecond {
		t.Errorf("rate limiter did not pace requests: 3 requests took only %v", elapsed)
	}
}

func TestResilience_PerAttemptDoesNotTruncateBody(t *testing.T) {
	// Headers arrive immediately, then the body streams slower than PerAttempt.
	// The per-attempt deadline must bound time-to-response, NOT the body read, so
	// the full body is delivered.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.WriteHeader(200)
		if fl != nil {
			fl.Flush()
		}
		for i := 0; i < 5; i++ {
			io.WriteString(w, "chunk")
			if fl != nil {
				fl.Flush()
			}
			time.Sleep(20 * time.Millisecond) // total ~100ms >> PerAttempt
		}
	}))
	defer srv.Close()

	c, _ := gocurl.New(gocurl.WithRetry(gocurl.RetryPolicy{
		MaxAttempts: 3, Backoff: gocurl.ConstantBackoff(0), PerAttempt: 40 * time.Millisecond, RetryOnStatus: []int{503},
	}))
	defer c.Close()

	body, _, err := c.CurlString(context.Background(), "curl "+srv.URL)
	if err != nil {
		t.Fatalf("unexpected error (per-attempt deadline truncated the body?): %v", err)
	}
	if body != "chunkchunkchunkchunkchunk" {
		t.Errorf("body truncated: got %q (len %d), want the full stream", body, len(body))
	}
}

func TestResilience_ContextCancelDuringRetry(t *testing.T) {
	srv := newFlakyServer(t, 100, 503) // always 503
	c, _ := gocurl.New(gocurl.WithRetry(gocurl.RetryPolicy{
		MaxAttempts: 10, Backoff: gocurl.ConstantBackoff(50 * time.Millisecond), RetryOnStatus: []int{503},
	}))
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := c.Curl(ctx, "curl "+srv.URL)
	if err == nil {
		t.Fatal("expected a context error from cancellation during retry backoff")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error should resolve context.DeadlineExceeded, got %v", err)
	}
}
