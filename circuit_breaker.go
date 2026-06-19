package gocurl

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

// ErrCircuitOpen is returned by the circuit breaker middleware when the circuit
// for a request's key is open (fast-fail). It is classified non-retryable.
var ErrCircuitOpen = errors.New("gocurl: circuit breaker is open")

// BreakerConfig configures a CircuitBreaker. Zero-valued fields take sensible
// defaults (see CircuitBreaker).
type BreakerConfig struct {
	FailureThreshold float64                          // fraction of failures in the window that trips (default 0.5)
	MinRequests      int                              // minimum samples before tripping (default 20)
	Window           time.Duration                    // rolling window (default 10s)
	OpenTimeout      time.Duration                    // delay before a half-open probe (default 5s)
	KeyFunc          func(*http.Request) string       // circuit key (default: request URL host)
	IsFailure        func(*http.Response, error) bool // failure predicate (default: err != nil || status >= 500)
}

const breakerBuckets = 10

type circuitState int

const (
	circuitClosed circuitState = iota
	circuitOpen
	circuitHalfOpen
)

// windowBucket is one time slice of the rolling window. epoch identifies the
// slice; a stale epoch means the bucket has aged out and is ignored when summing.
type windowBucket struct {
	epoch            int64
	success, failure int
}

// breakerState is the per-key circuit. All fields are guarded by mu. The
// config-derived thresholds are copied in at construction so the state is
// self-contained (no back-pointer to the breaker under lock).
type breakerState struct {
	mu       sync.Mutex
	state    circuitState
	openedAt time.Time
	probing  bool // a half-open probe is in flight
	buckets  [breakerBuckets]windowBucket

	bucketDur        time.Duration
	minRequests      int
	failureThreshold float64
	openTimeout      time.Duration

	// generation identifies the current closed-window epoch. It is bumped on every
	// window reset and trip so a slow in-flight request whose window was reset (or
	// superseded) while it ran cannot contaminate the fresh window when it records.
	generation int64
}

// allow decides whether a request may proceed now. It returns probe=true when
// this request is the single half-open probe (so record can interpret the
// outcome as a probe result), and the generation captured at admission (so
// record can drop a sample whose window was reset while the request was in
// flight).
func (s *breakerState) allow(now time.Time) (probe, allowed bool, gen int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch s.state {
	case circuitClosed:
		return false, true, s.generation
	case circuitOpen:
		if now.Sub(s.openedAt) >= s.openTimeout {
			s.state = circuitHalfOpen
			s.probing = true
			return true, true, s.generation
		}
		return false, false, s.generation
	case circuitHalfOpen:
		if s.probing {
			return false, false, s.generation
		}
		s.probing = true
		return true, true, s.generation
	default:
		return false, true, s.generation
	}
}

// record applies the outcome of an allowed request to the state machine. gen is
// the generation captured by allow at admission.
func (s *breakerState) record(now time.Time, failure, probe bool, gen int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if probe {
		s.probing = false
		if failure {
			s.state = circuitOpen
			s.openedAt = now
			s.generation++
		} else {
			s.state = circuitClosed
			s.resetWindowLocked()
		}
		return
	}

	// Drop a stale closed-state sample whose window was reset or superseded while
	// this (slow) request was in flight, so it cannot re-trip a recovered circuit.
	if gen != s.generation {
		return
	}

	// Closed-state sample: record into the rolling window and check the trip
	// condition. (Open/half-open non-probe requests never reach record because
	// allow rejected them.)
	epoch := now.UnixNano() / int64(s.bucketDur)
	idx := ((epoch % breakerBuckets) + breakerBuckets) % breakerBuckets
	if s.buckets[idx].epoch != epoch {
		s.buckets[idx] = windowBucket{epoch: epoch}
	}
	if failure {
		s.buckets[idx].failure++
	} else {
		s.buckets[idx].success++
	}

	if s.state == circuitClosed && failure {
		total, fails := s.windowTotalsLocked(epoch)
		if total >= s.minRequests && float64(fails)/float64(total) >= s.failureThreshold {
			s.state = circuitOpen
			s.openedAt = now
			s.generation++
		}
	}
}

// windowTotalsLocked sums non-stale buckets within the rolling window.
func (s *breakerState) windowTotalsLocked(epoch int64) (total, fails int) {
	minEpoch := epoch - (breakerBuckets - 1)
	for i := range s.buckets {
		if s.buckets[i].epoch >= minEpoch {
			total += s.buckets[i].success + s.buckets[i].failure
			fails += s.buckets[i].failure
		}
	}
	return total, fails
}

func (s *breakerState) resetWindowLocked() {
	for i := range s.buckets {
		s.buckets[i] = windowBucket{}
	}
	s.generation++
}

// breaker holds the per-key circuits and the resolved configuration.
type breaker struct {
	cfg   BreakerConfig
	mu    sync.Mutex
	state map[string]*breakerState
	nowFn func() time.Time // injectable clock for tests; nil => time.Now
}

func newBreaker(cfg BreakerConfig) *breaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 0.5
	}
	if cfg.MinRequests <= 0 {
		cfg.MinRequests = 20
	}
	if cfg.Window <= 0 {
		cfg.Window = 10 * time.Second
	}
	if cfg.OpenTimeout <= 0 {
		cfg.OpenTimeout = 5 * time.Second
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(r *http.Request) string {
			if r.URL != nil {
				return r.URL.Host
			}
			return ""
		}
	}
	if cfg.IsFailure == nil {
		cfg.IsFailure = defaultIsFailure
	}
	return &breaker{cfg: cfg, state: map[string]*breakerState{}}
}

func defaultIsFailure(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	return resp != nil && resp.StatusCode >= 500
}

func (b *breaker) now() time.Time {
	if b.nowFn != nil {
		return b.nowFn()
	}
	return time.Now()
}

func (b *breaker) stateFor(key string) *breakerState {
	b.mu.Lock()
	defer b.mu.Unlock()
	st := b.state[key]
	if st == nil {
		dur := b.cfg.Window / breakerBuckets
		if dur <= 0 {
			dur = time.Millisecond
		}
		st = &breakerState{
			bucketDur:        dur,
			minRequests:      b.cfg.MinRequests,
			failureThreshold: b.cfg.FailureThreshold,
			openTimeout:      b.cfg.OpenTimeout,
		}
		b.state[key] = st
	}
	return st
}

// middleware is the breaker's Middleware implementation: fast-fail when open,
// otherwise run the next handler and record the FINAL outcome once. Recording is
// panic-safe: if next panics, the outcome is recorded as a failure (releasing a
// half-open probe and re-opening the circuit) before the panic is re-raised, so
// a panicking downstream can never permanently wedge a half-open circuit.
func (b *breaker) middleware(next Handler) Handler {
	return func(req *http.Request) (*http.Response, error) {
		st := b.stateFor(b.cfg.KeyFunc(req))
		probe, allowed, gen := st.allow(b.now())
		if !allowed {
			return nil, ErrCircuitOpen
		}
		recorded := false
		defer func() {
			if r := recover(); r != nil {
				if !recorded {
					st.record(b.now(), true, probe, gen)
				}
				panic(r)
			}
		}()
		resp, err := next(req)
		st.record(b.now(), b.cfg.IsFailure(resp, err), probe, gen)
		recorded = true
		return resp, err
	}
}

// CircuitBreaker returns a Middleware implementing a per-key (default: per-host)
// rolling-window circuit breaker. When the failure fraction in the window
// exceeds the threshold (after MinRequests samples), the circuit opens and
// requests fast-fail with ErrCircuitOpen; after OpenTimeout a single probe is
// allowed (half-open), and its outcome closes or re-opens the circuit. The
// breaker counts only the FINAL outcome of a request (e.g. the result after the
// retry loop), never individual retry attempts.
func CircuitBreaker(cfg BreakerConfig) Middleware {
	return newBreaker(cfg).middleware
}
