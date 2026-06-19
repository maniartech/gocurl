package gocurl

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maniartech/gocurl/options"
)

// recordingRT is a mock RoundTripper returning scripted outcomes and counting hits.
type recordingRT struct {
	mu     sync.Mutex
	hits   int
	script []scriptedResp
}

func (rt *recordingRT) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.mu.Lock()
	i := rt.hits
	rt.hits++
	rt.mu.Unlock()
	sr := rt.script[len(rt.script)-1]
	if i < len(rt.script) {
		sr = rt.script[i]
	}
	if sr.err != nil {
		return nil, sr.err
	}
	h := http.Header{}
	if sr.retryAfter != "" {
		h.Set("Retry-After", sr.retryAfter)
	}
	return &http.Response{StatusCode: sr.status, Header: h, Body: io.NopCloser(strings.NewReader("ok"))}, nil
}

func (rt *recordingRT) count() int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.hits
}

func zeroBackoff(max int, statuses ...int) RetryPolicy {
	return RetryPolicy{MaxAttempts: max, Backoff: ConstantBackoff(0), RetryOnStatus: statuses}
}

func TestClientDo_GET503Retried(t *testing.T) {
	rt := &recordingRT{script: []scriptedResp{{status: 503}, {status: 200}}}
	c, err := New(WithTransport(rt), WithRetry(zeroBackoff(3, 503)))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	req, _ := NewRequest("GET", "http://example.com")
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || rt.count() != 2 {
		t.Errorf("status=%d hits=%d, want 200 and 2", resp.StatusCode, rt.count())
	}
}

func TestClientDo_POST503NotRetried(t *testing.T) {
	rt := &recordingRT{script: []scriptedResp{{status: 503}, {status: 200}}}
	c, _ := New(WithTransport(rt), WithRetry(zeroBackoff(3, 503)))
	defer c.Close()
	req, _ := NewRequest("POST", "http://example.com", Body([]byte("x")))
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 503 || rt.count() != 1 {
		t.Errorf("status=%d hits=%d, want 503 and 1 (POST not retried)", resp.StatusCode, rt.count())
	}
}

func TestClientDo_PerRequestPolicyOverride(t *testing.T) {
	rt := &recordingRT{script: []scriptedResp{{status: 503}, {status: 503}, {status: 503}, {status: 503}, {status: 200}}}
	// Client default allows only 2 attempts; the per-request override allows 5.
	c, _ := New(WithTransport(rt), WithRetry(zeroBackoff(2, 503)))
	defer c.Close()
	req, _ := NewRequest("GET", "http://example.com")
	req = req.WithRetryPolicy(zeroBackoff(5, 503))
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || rt.count() != 5 {
		t.Errorf("status=%d hits=%d, want 200 and 5 (per-request override beats client default)", resp.StatusCode, rt.count())
	}
}

func TestClientDo_UserMiddlewareRunsOncePerDo(t *testing.T) {
	rt := &recordingRT{script: []scriptedResp{{status: 503}, {status: 503}, {status: 200}}}
	entries := 0
	umw := func(next Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			entries++ // user middleware sits OUTSIDE the retry loop
			return next(req)
		}
	}
	c, _ := New(WithTransport(rt), WithRetry(zeroBackoff(3, 503)), WithMiddleware(umw))
	defer c.Close()
	req, _ := NewRequest("GET", "http://example.com")
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if rt.count() != 3 {
		t.Errorf("transport hits=%d, want 3 (retries happen)", rt.count())
	}
	if entries != 1 {
		t.Errorf("user middleware entered %d times, want 1 (outside the retry loop)", entries)
	}
}

func TestClientDo_BreakerCountsFinalOutcomeOnly(t *testing.T) {
	// 503 then 200: the retry loop turns it into a final success. A breaker that
	// counts only the final outcome must NOT trip (it never sees the 503), so the
	// next request still proceeds.
	rt := &recordingRT{script: []scriptedResp{{status: 503}, {status: 200}, {status: 200}}}
	c, _ := New(
		WithTransport(rt),
		WithRetry(zeroBackoff(3, 503)),
		WithCircuitBreaker(BreakerConfig{MinRequests: 1, FailureThreshold: 1.0, Window: time.Minute, OpenTimeout: time.Minute}),
	)
	defer c.Close()

	r1, _ := NewRequest("GET", "http://example.com")
	resp1, err := c.Do(context.Background(), r1)
	if err != nil {
		t.Fatalf("first request: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != 200 {
		t.Fatalf("first request status=%d, want 200", resp1.StatusCode)
	}

	// Second request must NOT be fast-failed by an open circuit.
	r2, _ := NewRequest("GET", "http://example.com")
	resp2, err := c.Do(context.Background(), r2)
	if errors.Is(err, ErrCircuitOpen) {
		t.Fatal("breaker tripped on a retried-then-succeeded request (per-attempt counting bug)")
	}
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
}

func TestClientDo_RateLimitPaces(t *testing.T) {
	rt := &recordingRT{script: []scriptedResp{{status: 200}}}
	// 50 rps, burst 1 => the 2nd request waits ~20ms.
	c, _ := New(WithTransport(rt), WithRateLimit(50, 1))
	defer c.Close()
	ctx := context.Background()

	r1, _ := NewRequest("GET", "http://example.com")
	resp1, _ := c.Do(ctx, r1)
	resp1.Body.Close()

	start := time.Now()
	r2, _ := NewRequest("GET", "http://example.com")
	resp2, _ := c.Do(ctx, r2)
	resp2.Body.Close()
	if elapsed := time.Since(start); elapsed < 5*time.Millisecond {
		t.Errorf("second request should have been paced by the limiter, took %v", elapsed)
	}
}

func TestClientDo_LegacyRetryConfigStillMethodAgnostic(t *testing.T) {
	// A Client with no WithRetry, but the prepared request carries a legacy
	// RetryConfig (as the --retry flag sets it): a POST is retried method-agnostically.
	rt := &recordingRT{script: []scriptedResp{{status: 502}, {status: 200}}}
	c, _ := New(WithTransport(rt))
	defer c.Close()
	req, _ := NewRequest("POST", "http://example.com", Body([]byte("x")))
	req.opts.RetryConfig = &options.RetryConfig{MaxRetries: 2, RetryDelay: time.Millisecond, RetryOnHTTP: []int{502}}
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || rt.count() != 2 {
		t.Errorf("status=%d hits=%d, want 200 and 2 (legacy POST retried)", resp.StatusCode, rt.count())
	}
}

func TestClientDo_LegacyBudgetGatesRetries(t *testing.T) {
	// A Client retry budget must gate the legacy (--retry/RetryConfig) path too.
	// WithRetryBudget(0,0) starts with a single token, so only one retry is
	// permitted before retries are suppressed — even though MaxRetries=5.
	rt := &recordingRT{script: []scriptedResp{{status: 502}, {status: 502}, {status: 502}, {status: 502}, {status: 200}}}
	c, _ := New(WithTransport(rt), WithRetryBudget(0, 0))
	defer c.Close()
	req, _ := NewRequest("POST", "http://example.com", Body([]byte("x")))
	req.opts.RetryConfig = &options.RetryConfig{MaxRetries: 5, RetryDelay: time.Millisecond, RetryOnHTTP: []int{502}}

	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	// 1 initial + 1 budget-permitted retry = 2 hits, then suppressed (still 502).
	if rt.count() != 2 || resp.StatusCode != 502 {
		t.Errorf("hits=%d status=%d, want 2 and 502 (budget gates the legacy path)", rt.count(), resp.StatusCode)
	}
}
