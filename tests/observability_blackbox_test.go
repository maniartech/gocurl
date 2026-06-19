package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
)

// countingMetrics is a minimal gocurl.Metrics for end-to-end assertions.
type countingMetrics struct {
	requests, retries, errors, latencies int64
	mu                                   sync.Mutex
	lastStatus                           int
}

func (m *countingMetrics) IncRequest(gocurl.RequestInfo) { atomic.AddInt64(&m.requests, 1) }
func (m *countingMetrics) IncInFlight(int)               {}
func (m *countingMetrics) ObserveLatency(_ time.Duration, info gocurl.ResultInfo) {
	atomic.AddInt64(&m.latencies, 1)
	m.mu.Lock()
	m.lastStatus = info.StatusCode
	m.mu.Unlock()
}
func (m *countingMetrics) IncRetry(gocurl.RequestInfo) { atomic.AddInt64(&m.retries, 1) }
func (m *countingMetrics) IncError(gocurl.Kind, gocurl.RequestInfo) {
	atomic.AddInt64(&m.errors, 1)
}

func TestObservability_EndToEndRetryAccounting(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) < 3 {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	m := &countingMetrics{}
	var responses int32
	c, _ := gocurl.New(
		gocurl.WithMetrics(m),
		gocurl.WithHooks(gocurl.Hooks{
			OnResponse: func(context.Context, *http.Request, *http.Response, time.Duration) {
				atomic.AddInt32(&responses, 1)
			},
		}),
		gocurl.WithRetry(gocurl.RetryPolicy{MaxAttempts: 5, Backoff: gocurl.ConstantBackoff(0), RetryOnStatus: []int{503}}),
	)
	defer c.Close()

	resp, err := c.Curl(context.Background(), "curl "+srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := atomic.LoadInt64(&m.requests); got != 1 {
		t.Errorf("IncRequest = %d, want 1", got)
	}
	if got := atomic.LoadInt64(&m.retries); got != 2 {
		t.Errorf("IncRetry = %d, want 2", got)
	}
	if got := atomic.LoadInt64(&m.latencies); got != 1 {
		t.Errorf("ObserveLatency = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&responses); got != 1 {
		t.Errorf("OnResponse fired %d times, want 1", got)
	}
	if m.lastStatus != 200 {
		t.Errorf("recorded status = %d, want 200", m.lastStatus)
	}
}

// TestObservability_DisabledByDefault confirms a Client with no sinks behaves
// normally (and does not require any observability configuration).
func TestObservability_DisabledByDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c, _ := gocurl.New()
	defer c.Close()
	resp, err := c.Curl(context.Background(), "curl "+srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status=%d, want 200", resp.StatusCode)
	}
}
