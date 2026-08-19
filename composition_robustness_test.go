package gocurl

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCompositionRobustness_CancelledRequestNeverAttempts(t *testing.T) {
	var calls atomic.Int64
	base := Handler(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})
	h := Retry(DefaultRetryPolicy(3))(base)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)

	_, err := h(req)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v, want context.Canceled", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("attempts=%d, want zero", calls.Load())
	}
}

func TestCompositionRobustness_ReplaysBufferedBody(t *testing.T) {
	const payload = "replay-me"
	var mu sync.Mutex
	var bodies []string
	base := Handler(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		mu.Lock()
		bodies = append(bodies, string(body))
		attempt := len(bodies)
		mu.Unlock()
		status := http.StatusServiceUnavailable
		if attempt == 2 {
			status = http.StatusOK
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: http.NoBody}, nil
	})
	h := Retry(RetryPolicy{MaxAttempts: 2, Backoff: ConstantBackoff(0)})(base)
	req, _ := http.NewRequest(http.MethodPut, "http://example.com", nil)
	// Supplying Body directly keeps GetBody nil and exercises gocurl's bounded
	// buffering path rather than net/http's built-in rewind helper.
	req.Body = io.NopCloser(bytes.NewBufferString(payload))
	req.ContentLength = int64(len(payload))

	resp, err := h(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want 200", resp.StatusCode)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(bodies) != 2 || bodies[0] != payload || bodies[1] != payload {
		t.Fatalf("attempt bodies=%q, want two identical payloads", bodies)
	}
}

func TestCompositionRobustness_OverReplayCapSendsOnce(t *testing.T) {
	payload := bytes.Repeat([]byte{'x'}, int(DefaultMaxReplayBytes)+1)
	var calls atomic.Int64
	base := Handler(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, req.Body)
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: http.NoBody}, nil
	})
	h := Retry(RetryPolicy{
		MaxAttempts:  3,
		Backoff:      ConstantBackoff(0),
		AllowMethods: []string{http.MethodPost},
	})(base)
	req, _ := http.NewRequest(http.MethodPost, "http://example.com", nil)
	req.Body = io.NopCloser(bytes.NewReader(payload))
	req.ContentLength = int64(len(payload))

	if _, err := h(req); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("attempts=%d, want one for an over-cap non-rewindable body", calls.Load())
	}
}

func TestCompositionRobustness_RetryAfterCannotEscapeElapsedBudget(t *testing.T) {
	var calls atomic.Int64
	base := Handler(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Header:     http.Header{"Retry-After": {"60"}},
			Body:       http.NoBody,
		}, nil
	})
	h := Retry(RetryPolicy{
		MaxAttempts:       3,
		Backoff:           ConstantBackoff(0),
		MaxElapsed:        25 * time.Millisecond,
		RespectRetryAfter: true,
	})(base)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	started := time.Now()

	resp, err := h(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable || calls.Load() != 1 {
		t.Fatalf("status=%d attempts=%d, want 503 and one attempt", resp.StatusCode, calls.Load())
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("elapsed=%s; Retry-After escaped the 25ms elapsed budget", elapsed)
	}
}

func TestCompositionRobustness_ConcurrentChainReuse(t *testing.T) {
	const requests = 200
	var requestHooks atomic.Int64
	var responseHooks atomic.Int64
	base := Handler(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: http.NoBody, Request: req}, nil
	})
	h := Observe(Hooks{
		OnRequest:  func(context.Context, *http.Request) { requestHooks.Add(1) },
		OnResponse: func(context.Context, *http.Request, *http.Response, time.Duration) { responseHooks.Add(1) },
	}, performanceMetrics{}, performanceTracer{}, performanceLogger{})(
		Retry(RetryPolicy{MaxAttempts: 2, Backoff: ConstantBackoff(0)})(base),
	)

	var wg sync.WaitGroup
	errs := make(chan error, requests)
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
			resp, err := h(req)
			if err == nil {
				_ = resp.Body.Close()
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if requestHooks.Load() != requests || responseHooks.Load() != requests {
		t.Fatalf("request/response hooks=%d/%d, want %d/%d", requestHooks.Load(), responseHooks.Load(), requests, requests)
	}
}

func TestCompositionRobustness_InjectedRedirectRechecksSSRF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, "http://169.254.169.254/latest/meta-data", http.StatusFound)
	}))
	defer srv.Close()

	policy := DefaultSSRFPolicy()
	policy.AllowHosts = []string{"127.0.0.1"}
	h := SSRFGuard(policy)(HandlerFromRoundTripper(http.DefaultTransport))
	client := &http.Client{Transport: RoundTripperFromHandler(h)}

	_, err := client.Get(srv.URL)
	if !errors.Is(err, ErrSSRFBlocked) {
		t.Fatalf("error=%v, want redirect hop blocked by SSRF policy", err)
	}
}
