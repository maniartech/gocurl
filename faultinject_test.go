package gocurl

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/http2"
)

// Fault-injection harness (Spec 14, Phase A / M12-T1).
//
// The backbone is faultyRT: a programmable http.RoundTripper that returns a
// scripted sequence of (response, error) outcomes, so the resilience layer
// (retry classification, circuit breaker, per-attempt deadlines) is exercised
// under *deterministic* failures with no real network and no flakiness. It is
// injected via the existing WithTransport option — no new public surface.

type faultStep func(req *http.Request) (*http.Response, error)

type faultyRT struct {
	mu    sync.Mutex
	n     int
	steps []faultStep
}

func newFaultyRT(steps ...faultStep) *faultyRT { return &faultyRT{steps: steps} }

func (f *faultyRT) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.n
}

func (f *faultyRT) RoundTrip(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	i := f.n
	f.n++
	f.mu.Unlock()
	if i >= len(f.steps) {
		i = len(f.steps) - 1 // repeat the last step indefinitely
	}
	resp, err := f.steps[i](req)
	if resp != nil {
		resp.Request = req
	}
	return resp, err
}

// --- step + error constructors (cross-platform: classification keys on the
// error *type*, so we avoid platform-specific syscall errnos) ---

func stepStatus(code int) faultStep {
	return func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: code,
			Proto:      "HTTP/1.1",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("x")),
		}, nil
	}
}

func stepErr(err error) faultStep {
	return func(*http.Request) (*http.Response, error) { return nil, err }
}

// stepBlock waits for the (per-attempt) context to fire, modelling a slow-loris /
// dead peer, then returns the context error.
func stepBlock() faultStep {
	return func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	}
}

func connResetErr() error {
	return &net.OpError{Op: "read", Net: "tcp", Err: errors.New("connection reset by peer")}
}
func dialRefusedErr() error {
	return &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
}
func tlsHandshakeErr() error { return errors.New("tls: handshake failure") } // isTLSError matches "tls:"

func faultClient(t *testing.T, rt http.RoundTripper, opts ...Option) *Client {
	t.Helper()
	c, err := New(append([]Option{WithTransport(rt)}, opts...)...)
	if err != nil {
		t.Fatalf("New(faultClient): %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func drain(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// TestFault_RetryClassification proves each transport failure is classified
// correctly AND that the retry engine honors the classification: connect/timeout
// are retried (until exhausted), TLS is not.
func TestFault_RetryClassification(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		sentinel  error // the underlying classification, reachable via errors.Is even after wrapping
		wantRetry bool
	}{
		{"connection reset", connResetErr(), ErrConnect, true},
		{"dial refused", dialRefusedErr(), ErrConnect, true},
		{"timeout", context.DeadlineExceeded, ErrTimeout, true},
		{"tls handshake", tlsHandshakeErr(), ErrTLS, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rt := newFaultyRT(stepErr(c.err)) // always fails with this error
			const attempts = 3
			client := faultClient(t, rt, WithRetry(RetryPolicy{MaxAttempts: attempts, Backoff: ConstantBackoff(0)}))

			resp, err := client.Curl(context.Background(), "curl http://fault.test/")
			drain(resp)
			if err == nil {
				t.Fatal("expected an error")
			}
			// The underlying classification is reachable through any retry-exhausted wrapper.
			if !errors.Is(err, c.sentinel) {
				t.Errorf("errors.Is(err, %v) = false; err: %v", c.sentinel, err)
			}
			if c.wantRetry {
				// A retryable failure burns every attempt and surfaces as retry-exhausted.
				if !errors.Is(err, ErrRetryExhausted) {
					t.Errorf("retryable failure should exhaust retries; err: %v", err)
				}
				if rt.calls() != attempts {
					t.Errorf("calls = %d, want %d (retryable)", rt.calls(), attempts)
				}
			} else {
				// A non-retryable failure is attempted exactly once, never retried.
				if errors.Is(err, ErrRetryExhausted) {
					t.Errorf("non-retryable failure must not be retried; err: %v", err)
				}
				if rt.calls() != 1 {
					t.Errorf("calls = %d, want 1 (non-retryable)", rt.calls())
				}
			}
		})
	}
}

// TestFault_RetrySucceedsAfterTransient proves a request recovers when transient
// failures are followed by success, replaying the idempotent GET.
func TestFault_RetrySucceedsAfterTransient(t *testing.T) {
	rt := newFaultyRT(stepErr(connResetErr()), stepErr(connResetErr()), stepStatus(200))
	client := faultClient(t, rt, WithRetry(RetryPolicy{MaxAttempts: 5, Backoff: ConstantBackoff(0)}))

	resp, err := client.Curl(context.Background(), "curl http://fault.test/")
	defer drain(resp)
	if err != nil {
		t.Fatalf("expected recovery, got error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if rt.calls() != 3 {
		t.Errorf("calls = %d, want 3 (two failures + success)", rt.calls())
	}
}

// TestFault_OverallRetryBudget proves the Client's WithTimeout bounds the ENTIRE
// operation including retries (curl's --max-time semantics) — not each attempt.
// Without the overall budget, a retryable storm runs MaxAttempts * backoff well past
// the deadline (a retry amplifier); with it, the loop is cut off near the budget.
func TestFault_OverallRetryBudget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable) // always retryable (503)
	}))
	defer srv.Close()

	const budget = 250 * time.Millisecond
	client, err := New(WithTimeout(budget), WithRetry(RetryPolicy{
		MaxAttempts: 20, Backoff: ConstantBackoff(80 * time.Millisecond),
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer client.Close()

	start := time.Now()
	resp, _ := client.Curl(context.Background(), "curl "+srv.URL)
	if resp != nil {
		resp.Body.Close()
	}
	elapsed := time.Since(start)
	// 20 attempts * 80ms backoff would be ~1.6s; the overall budget must cut it off.
	if elapsed > budget*3 {
		t.Fatalf("WithTimeout(%v) did not bound the retry loop: elapsed=%v (retry amplifier)", budget, elapsed)
	}
}

// TestFault_H2ErrorsRetried proves HTTP/2 connection-level failures (GOAWAY and a
// refused RST_STREAM) are classified as retryable and an idempotent GET recovers.
// h2 is the default TLS path, and these errors mean the server did not process the
// request — so they MUST be retried, like a connection drop.
func TestFault_H2ErrorsRetried(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"GOAWAY", http2.GoAwayError{LastStreamID: 1, ErrCode: http2.ErrCodeNo}},
		{"RST_STREAM refused", http2.StreamError{StreamID: 1, Code: http2.ErrCodeRefusedStream}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := classifyTransportError(c.err); got != KindConnect {
				t.Errorf("classifyTransportError(%T) = %v, want KindConnect", c.err, got)
			}
			rt := newFaultyRT(stepErr(c.err), stepStatus(200))
			client := faultClient(t, rt, WithRetry(RetryPolicy{MaxAttempts: 3, Backoff: ConstantBackoff(0)}))
			resp, err := client.Curl(context.Background(), "curl http://fault.test/")
			drain(resp)
			if err != nil {
				t.Fatalf("idempotent GET should recover from h2 %s, got: %v", c.name, err)
			}
			if rt.calls() != 2 {
				t.Errorf("calls = %d, want 2 (h2 error then 200)", rt.calls())
			}
		})
	}
}

// TestFault_OneShotMaxTimeBoundsRetries proves --max-time bounds the WHOLE one-shot
// operation including --retry attempts (curl semantics), not each attempt.
func TestFault_OneShotMaxTimeBoundsRetries(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable) // retryable
	}))
	defer srv.Close()

	start := time.Now()
	resp, _ := CurlArgs(context.Background(), "--max-time", "1", "--retry", "15", srv.URL)
	if resp != nil {
		resp.Body.Close()
	}
	elapsed := time.Since(start)
	if elapsed > 3*time.Second {
		t.Fatalf("--max-time did not bound the one-shot retry loop: elapsed=%v (attempts=%d)",
			elapsed, atomic.LoadInt32(&attempts))
	}
}

// TestFault_RetriesOnRetryableStatus proves a 429 storm (in the default retry set)
// is retried for an idempotent GET and recovers when the server returns 200.
func TestFault_RetriesOnRetryableStatus(t *testing.T) {
	rt := newFaultyRT(stepStatus(429), stepStatus(503), stepStatus(200))
	client := faultClient(t, rt, WithRetry(RetryPolicy{MaxAttempts: 5, Backoff: ConstantBackoff(0)}))

	resp, err := client.Curl(context.Background(), "curl http://fault.test/")
	defer drain(resp)
	if err != nil {
		t.Fatalf("expected recovery after retryable statuses, got: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if rt.calls() != 3 {
		t.Errorf("calls = %d, want 3 (429, 503, then 200)", rt.calls())
	}
}

// TestFault_PerAttemptDeadline proves a stalled peer is bounded by the per-attempt
// deadline and surfaces as a (retryable) timeout, not a hang.
func TestFault_PerAttemptDeadline(t *testing.T) {
	rt := newFaultyRT(stepBlock())
	client := faultClient(t, rt, WithRetry(RetryPolicy{
		MaxAttempts: 2, PerAttempt: 40 * time.Millisecond, Backoff: ConstantBackoff(0),
	}))

	start := time.Now()
	resp, err := client.Curl(context.Background(), "curl http://fault.test/")
	drain(resp)
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if !IsTimeout(err) {
		t.Errorf("expected timeout, got Kind=%v err=%v", KindOf(err), err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("per-attempt deadline did not bound the stall (%v)", elapsed)
	}
}

// TestFault_CircuitBreakerTrips proves a failure storm trips the breaker, which
// then short-circuits with ErrCircuitOpen instead of hammering the dead peer.
func TestFault_CircuitBreakerTrips(t *testing.T) {
	rt := newFaultyRT(stepStatus(503)) // every call is a 5xx failure
	client := faultClient(t, rt, WithCircuitBreaker(BreakerConfig{
		FailureThreshold: 0.5,
		MinRequests:      4,
		Window:           time.Minute,
		OpenTimeout:      time.Minute, // stays open for the test
	}))

	var opened bool
	for i := 0; i < 30; i++ {
		resp, err := client.Curl(context.Background(), "curl http://fault.test/")
		drain(resp)
		if errors.Is(err, ErrCircuitOpen) {
			opened = true
			break
		}
	}
	if !opened {
		t.Fatal("circuit breaker never opened under a sustained failure storm")
	}
	// Once open, the breaker must stop calling the transport.
	callsAtOpen := rt.calls()
	for i := 0; i < 5; i++ {
		resp, err := client.Curl(context.Background(), "curl http://fault.test/")
		drain(resp)
		if !errors.Is(err, ErrCircuitOpen) {
			t.Fatalf("expected ErrCircuitOpen while open, got %v", err)
		}
	}
	if rt.calls() != callsAtOpen {
		t.Errorf("breaker still called transport while open: %d -> %d", callsAtOpen, rt.calls())
	}
}

// TestFault_NoGoroutineLeakUnderStorm runs a storm of mixed failures and successes
// through a reused Client and asserts the goroutine count returns to baseline —
// the mission-critical "no leak under failure" invariant.
func TestFault_NoGoroutineLeakUnderStorm(t *testing.T) {
	rt := newFaultyRT(
		stepErr(connResetErr()), stepStatus(503), stepStatus(200), stepErr(dialRefusedErr()),
	)
	client := faultClient(t, rt, WithRetry(RetryPolicy{MaxAttempts: 3, Backoff: ConstantBackoff(0)}))

	// Warm up, then snapshot.
	for i := 0; i < 20; i++ {
		resp, _ := client.Curl(context.Background(), "curl http://fault.test/")
		drain(resp)
	}
	settle()
	before := runtime.NumGoroutine()

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, _ := client.Curl(context.Background(), "curl http://fault.test/")
			drain(resp)
		}()
	}
	wg.Wait()
	settle()
	after := runtime.NumGoroutine()

	if after > before+5 {
		t.Errorf("goroutine leak under failure storm: before=%d after=%d", before, after)
	}
}

// TestFault_EasyCurlStillWorks is the "easy as curl" invariant: after all the
// production hardening, the zero-config one-liner against a healthy server still
// just works.
func TestFault_EasyCurlStillWorks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	body, resp, err := CurlString(context.Background(), "curl "+srv.URL)
	if err != nil {
		t.Fatalf("the easy one-liner must keep working: %v", err)
	}
	defer resp.Body.Close()
	if body != "hello" {
		t.Errorf("body = %q, want %q", body, "hello")
	}
}

// settle gives spawned goroutines a moment to unwind and runs GC so the
// goroutine snapshot is stable.
func settle() {
	for i := 0; i < 5; i++ {
		runtime.GC()
		time.Sleep(20 * time.Millisecond)
	}
}
