package gocurl

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Limiter throttles outbound requests. Wait blocks until the request may proceed
// or the context is done (returning ctx.Err()). It is the pluggable seam for
// rate limiting: the built-in TokenBucket implements it, and an external limiter
// (e.g. golang.org/x/time/rate) can be supplied via RateLimiter.
type Limiter interface {
	Wait(ctx context.Context) error
}

// TokenBucket is a client-side token-bucket rate limiter. It refills at `rps`
// tokens per second up to a maximum of `burst`, and is safe for concurrent use.
// It has zero external dependencies.
type TokenBucket struct {
	mu     sync.Mutex
	tokens float64
	rps    float64
	burst  float64
	last   time.Time
	nowFn  func() time.Time // injectable clock for tests; nil => time.Now
}

// NewTokenBucket returns a token bucket admitting `rps` requests per second with
// a burst capacity of `burst`. The bucket starts full. rps <= 0 is treated as 1;
// burst < 1 is treated as 1.
func NewTokenBucket(rps float64, burst int) *TokenBucket {
	if rps <= 0 {
		rps = 1
	}
	if burst < 1 {
		burst = 1
	}
	return &TokenBucket{
		tokens: float64(burst),
		rps:    rps,
		burst:  float64(burst),
	}
}

func (tb *TokenBucket) now() time.Time {
	if tb.nowFn != nil {
		return tb.nowFn()
	}
	return time.Now()
}

// refillLocked accrues tokens for the elapsed time. Caller must hold mu.
func (tb *TokenBucket) refillLocked(now time.Time) {
	if tb.last.IsZero() {
		tb.last = now
		return
	}
	elapsed := now.Sub(tb.last).Seconds()
	if elapsed <= 0 {
		return
	}
	tb.last = now
	tb.tokens += elapsed * tb.rps
	if tb.tokens > tb.burst {
		tb.tokens = tb.burst
	}
}

// Wait blocks until a token is available or ctx is done. On cancellation it
// returns ctx.Err() without consuming a token.
func (tb *TokenBucket) Wait(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		tb.mu.Lock()
		now := tb.now()
		tb.refillLocked(now)
		if tb.tokens >= 1 {
			tb.tokens--
			tb.mu.Unlock()
			return nil
		}
		// Time until the next whole token, computed under the lock but waited on
		// without holding it.
		deficit := 1 - tb.tokens
		wait := time.Duration(deficit / tb.rps * float64(time.Second))
		tb.mu.Unlock()

		if wait <= 0 {
			wait = time.Millisecond
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// retry the consume
		}
	}
}

// RateLimiter returns a Middleware that blocks each request on l.Wait(ctx) before
// passing it to the next handler. A nil Limiter is a pass-through.
func RateLimiter(l Limiter) Middleware {
	return func(next Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			if l != nil {
				if err := l.Wait(req.Context()); err != nil {
					return nil, err
				}
			}
			return next(req)
		}
	}
}
