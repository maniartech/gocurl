package gocurl

import (
	"sync"
	"time"
)

// RetryBudget caps retries as a fraction of overall request volume so that a
// partial outage cannot trigger a retry storm. It is a token bucket: each
// completed request refills it a little, and each retry spends a token. When the
// bucket is empty, further retries are suppressed (the first attempt of any
// request is never budget-gated).
//
// A RetryBudget is safe for concurrent use and is shared across all requests of
// a Client. See specs/04-resilience.md.
type RetryBudget struct {
	// Ratio is the maximum retries expressed as a fraction of requests (e.g. 0.1
	// allows retries up to 10% of request volume). Each successful request adds
	// Ratio tokens.
	Ratio float64
	// MinPerSec is a floor on token accrual (tokens/sec) so a low-traffic client
	// can still retry occasionally even with little baseline traffic.
	MinPerSec float64

	mu         sync.Mutex
	tokens     float64
	max        float64
	lastRefill time.Time
	nowFn      func() time.Time // injectable clock for tests; nil => time.Now
}

// NewRetryBudget returns a retry budget that allows retries up to `ratio` of
// request volume, with a floor of `minPerSec` tokens/sec for low-traffic
// clients. The bucket starts full.
func NewRetryBudget(ratio, minPerSec float64) *RetryBudget {
	if ratio < 0 {
		ratio = 0
	}
	if minPerSec < 0 {
		minPerSec = 0
	}
	// Cap the bucket at ~10 seconds of floor accrual (at least 1 token) so it can
	// absorb a small burst without permitting an unbounded backlog.
	max := minPerSec * 10
	if max < 1 {
		max = 1
	}
	return &RetryBudget{
		Ratio:     ratio,
		MinPerSec: minPerSec,
		tokens:    max,
		max:       max,
	}
}

func (b *RetryBudget) now() time.Time {
	if b.nowFn != nil {
		return b.nowFn()
	}
	return time.Now()
}

// refillLocked adds floor-rate tokens for the elapsed time. Caller must hold mu.
func (b *RetryBudget) refillLocked(now time.Time) {
	if b.lastRefill.IsZero() {
		b.lastRefill = now
		return
	}
	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}
	b.lastRefill = now
	b.tokens += elapsed * b.MinPerSec
	if b.tokens > b.max {
		b.tokens = b.max
	}
}

// Consume attempts to spend one retry token. It returns true (and decrements) if
// a token was available, false otherwise. A nil budget always permits the retry.
func (b *RetryBudget) Consume() bool {
	if b == nil {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refillLocked(b.now())
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// Refund credits the budget for a completed request (Ratio tokens), raising the
// allowance for subsequent retries. A nil budget is a no-op.
func (b *RetryBudget) Refund() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refillLocked(b.now())
	b.tokens += b.Ratio
	if b.tokens > b.max {
		b.tokens = b.max
	}
}
