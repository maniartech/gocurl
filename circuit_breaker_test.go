package gocurl

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func breakerReq(t *testing.T, rawurl string) *http.Request {
	t.Helper()
	req, err := http.NewRequest("GET", rawurl, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func breakerOK() Handler {
	return func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	}
}

func breakerFail(status int) Handler {
	return func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(""))}, nil
	}
}

func TestBreaker_TripsThenFastFails(t *testing.T) {
	b := newBreaker(BreakerConfig{MinRequests: 4, FailureThreshold: 0.5, Window: time.Second, OpenTimeout: time.Second})
	now := time.Unix(0, 0)
	b.nowFn = func() time.Time { return now }
	h := b.middleware(breakerFail(500))
	req := breakerReq(t, "http://host-a/x")

	for i := 0; i < 4; i++ {
		if _, err := h(req); err != nil {
			t.Fatalf("call %d should pass through while closed, got %v", i, err)
		}
	}
	// 5th call: circuit open, fast-fail.
	if _, err := h(req); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
	if IsRetryable(ErrCircuitOpen) {
		t.Error("ErrCircuitOpen must be non-retryable")
	}
}

func TestBreaker_HalfOpenRecovery(t *testing.T) {
	b := newBreaker(BreakerConfig{MinRequests: 2, FailureThreshold: 0.5, Window: time.Second, OpenTimeout: time.Second})
	now := time.Unix(100, 0)
	b.nowFn = func() time.Time { return now }
	req := breakerReq(t, "http://host/x")

	// Trip it (2 failures).
	hf := b.middleware(breakerFail(503))
	hf(req)
	hf(req)
	if _, err := hf(req); !errors.Is(err, ErrCircuitOpen) {
		t.Fatal("expected open after trip")
	}

	// Before OpenTimeout: still open.
	now = now.Add(500 * time.Millisecond)
	if _, err := hf(req); !errors.Is(err, ErrCircuitOpen) {
		t.Fatal("should still be open before OpenTimeout")
	}

	// After OpenTimeout: a probe is allowed; a success closes the circuit.
	now = now.Add(time.Second)
	ho := b.middleware(breakerOK())
	if _, err := ho(req); err != nil {
		t.Fatalf("half-open probe should be allowed, got %v", err)
	}
	// Closed again: subsequent successes pass.
	if _, err := ho(req); err != nil {
		t.Fatalf("circuit should be closed after a successful probe, got %v", err)
	}
}

func TestBreaker_HalfOpenFailureReopens(t *testing.T) {
	b := newBreaker(BreakerConfig{MinRequests: 2, FailureThreshold: 0.5, Window: time.Second, OpenTimeout: time.Second})
	now := time.Unix(0, 0)
	b.nowFn = func() time.Time { return now }
	req := breakerReq(t, "http://host/x")
	hf := b.middleware(breakerFail(500))
	hf(req)
	hf(req)

	now = now.Add(2 * time.Second) // past OpenTimeout
	// Probe fails -> reopen.
	if _, err := hf(req); err != nil {
		t.Fatalf("probe should be allowed, got %v", err)
	}
	// Immediately after a failed probe, circuit is open again.
	if _, err := hf(req); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("failed probe should reopen the circuit, got %v", err)
	}
}

func TestBreaker_OnlyFirstHalfOpenProbes(t *testing.T) {
	// Direct state-machine test: after transitioning to half-open, a second
	// allow() in flight is rejected until the probe is recorded.
	st := &breakerState{bucketDur: time.Millisecond, minRequests: 2, failureThreshold: 0.5, openTimeout: time.Second}
	now := time.Unix(0, 0)
	st.state = circuitOpen
	st.openedAt = now

	now = now.Add(2 * time.Second)
	if probe, allowed, _ := st.allow(now); !probe || !allowed {
		t.Fatal("first request after OpenTimeout should be the probe")
	}
	if _, allowed, _ := st.allow(now); allowed {
		t.Fatal("second concurrent request must fast-fail while probe is in flight")
	}
	st.record(now, false, true, 0) // probe succeeds -> closed
	if probe, allowed, _ := st.allow(now); probe || !allowed {
		t.Fatal("after a successful probe the circuit should be closed (non-probe, allowed)")
	}
}

func TestBreaker_PanicDoesNotWedgeHalfOpen(t *testing.T) {
	b := newBreaker(BreakerConfig{MinRequests: 2, FailureThreshold: 0.5, Window: time.Second, OpenTimeout: time.Second})
	now := time.Unix(0, 0)
	b.nowFn = func() time.Time { return now }
	req := breakerReq(t, "http://host/x")

	// Trip the circuit.
	hf := b.middleware(breakerFail(500))
	hf(req)
	hf(req)

	// Past OpenTimeout: the half-open probe panics. The breaker must record a
	// failure (not leak the probing flag) and re-raise.
	now = now.Add(2 * time.Second)
	hp := b.middleware(func(*http.Request) (*http.Response, error) { panic("boom") })
	func() {
		defer func() { _ = recover() }()
		hp(req)
		t.Fatal("expected the panic to propagate")
	}()

	// The circuit must not be permanently wedged: after another OpenTimeout it
	// half-opens again and a success closes it.
	now = now.Add(2 * time.Second)
	ho := b.middleware(breakerOK())
	if _, err := ho(req); err != nil {
		t.Fatalf("circuit wedged after a panicking probe: %v", err)
	}
}

func TestBreaker_StaleSampleAfterResetIgnored(t *testing.T) {
	// A closed-state sample captured at an old generation (its window since reset
	// by a successful probe) must be dropped, not re-trip the recovered circuit.
	st := &breakerState{bucketDur: time.Millisecond, minRequests: 1, failureThreshold: 0.5, openTimeout: time.Second}
	now := time.Unix(0, 0)

	_, allowed, gen := st.allow(now) // admitted while closed -> generation 0
	if !allowed {
		t.Fatal("closed circuit should admit")
	}
	st.mu.Lock()
	st.resetWindowLocked() // generation -> 1 (as a half-open probe success would do)
	st.mu.Unlock()

	st.record(now, true, false, gen) // straggler records a failure at the OLD generation

	st.mu.Lock()
	state := st.state
	st.mu.Unlock()
	if state == circuitOpen {
		t.Error("stale post-reset sample must not re-trip the circuit")
	}
}

func TestBreaker_PerHostIsolation(t *testing.T) {
	b := newBreaker(BreakerConfig{MinRequests: 2, FailureThreshold: 0.5, Window: time.Second, OpenTimeout: time.Second})
	now := time.Unix(0, 0)
	b.nowFn = func() time.Time { return now }
	hf := b.middleware(breakerFail(500))

	a := breakerReq(t, "http://host-a/x")
	bn := breakerReq(t, "http://host-b/x")
	hf(a)
	hf(a) // trip host-a
	if _, err := hf(a); !errors.Is(err, ErrCircuitOpen) {
		t.Fatal("host-a should be open")
	}
	// host-b is unaffected.
	if _, err := hf(bn); err != nil {
		t.Fatalf("host-b should still be closed, got %v", err)
	}
}

func TestBreaker_RollingWindowEviction(t *testing.T) {
	// Failures spread beyond the window are evicted and must not trip.
	b := newBreaker(BreakerConfig{MinRequests: 4, FailureThreshold: 0.5, Window: 100 * time.Millisecond, OpenTimeout: time.Second})
	now := time.Unix(0, 0)
	b.nowFn = func() time.Time { return now }
	hf := b.middleware(breakerFail(500))
	req := breakerReq(t, "http://host/x")

	hf(req)
	hf(req) // 2 failures at t0
	now = now.Add(500 * time.Millisecond)
	hf(req)
	hf(req) // 2 more failures, but the first 2 have aged out of the 100ms window
	if _, err := hf(req); errors.Is(err, ErrCircuitOpen) {
		t.Fatal("circuit should NOT trip: old failures evicted, window total < MinRequests")
	}
}

func TestBreaker_CustomIsFailure(t *testing.T) {
	// Treat 404 as a failure via a custom predicate.
	b := newBreaker(BreakerConfig{
		MinRequests:      2,
		FailureThreshold: 0.5,
		Window:           time.Second,
		OpenTimeout:      time.Second,
		IsFailure: func(resp *http.Response, err error) bool {
			return err != nil || (resp != nil && resp.StatusCode == 404)
		},
	})
	now := time.Unix(0, 0)
	b.nowFn = func() time.Time { return now }
	h := b.middleware(breakerFail(404))
	req := breakerReq(t, "http://host/x")
	h(req)
	h(req)
	if _, err := h(req); !errors.Is(err, ErrCircuitOpen) {
		t.Fatal("custom IsFailure should count 404 and trip the circuit")
	}
}

func TestBreaker_ConcurrentRaceClean(t *testing.T) {
	b := newBreaker(BreakerConfig{MinRequests: 50, FailureThreshold: 0.9, Window: time.Second, OpenTimeout: 10 * time.Millisecond})
	// A persistently failing handler exercises closed (record), open (fast-fail),
	// and half-open (probe) paths under concurrency.
	h := b.middleware(breakerFail(500))
	req := breakerReq(t, "http://host/x")
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				resp, err := h(req)
				if resp != nil {
					resp.Body.Close()
				}
				_ = err
			}
		}()
	}
	wg.Wait()
}

func TestBreaker_DefaultsApplied(t *testing.T) {
	b := newBreaker(BreakerConfig{})
	if b.cfg.FailureThreshold != 0.5 || b.cfg.MinRequests != 20 || b.cfg.Window != 10*time.Second || b.cfg.OpenTimeout != 5*time.Second {
		t.Errorf("defaults not applied: %+v", b.cfg)
	}
	if b.cfg.KeyFunc == nil || b.cfg.IsFailure == nil {
		t.Error("default KeyFunc/IsFailure should be set")
	}
	// Default key is the request host.
	if got := b.cfg.KeyFunc(breakerReq(t, "http://example.com:8080/path")); got != "example.com:8080" {
		t.Errorf("default KeyFunc = %q, want host", got)
	}
}
