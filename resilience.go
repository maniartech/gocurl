package gocurl

import (
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/maniartech/gocurl/options"
)

// DefaultMaxReplayBytes is the default ceiling for buffering a request body for
// retries when the body cannot be re-obtained via GetBody. Bodies larger than
// this are sent once and the request becomes non-retryable on attempt 2+.
const DefaultMaxReplayBytes int64 = 1 << 20 // 1 MiB

// idempotentMethods is the default set of HTTP methods eligible for retry. It is
// the HTTP-safe/idempotent set; POST, PATCH, and CONNECT are intentionally
// excluded (see RetryPolicy and the Idempotency-Key escape hatch).
var idempotentMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
	http.MethodPut:     true,
	http.MethodDelete:  true,
}

// RetryPolicy is an immutable value describing how a request is retried. Attach
// it to a Client with WithRetry / WithRetryAttempts, or to a single prepared
// Request with Request.WithRetryPolicy. The zero value disables retries
// (MaxAttempts <= 1 means a single attempt).
//
// Retries are idempotency-aware by default: only methods in the idempotent set
// (GET, HEAD, OPTIONS, TRACE, PUT, DELETE) — or a request carrying a non-empty
// Idempotency-Key header — are retried. POST/PATCH/CONNECT are NOT retried
// unless AllowMethods opts them in. (The legacy options.RetryConfig / --retry
// path remains method-agnostic for backward compatibility.)
//
// See specs/04-resilience.md.
type RetryPolicy struct {
	MaxAttempts       int                 // total attempts incl. the first; <=1 disables retries
	Backoff           Backoff             // delay schedule (default: ExponentialJitter(100ms,5s))
	MaxElapsed        time.Duration       // 0 = unbounded; overall wall-clock budget across attempts
	PerAttempt        time.Duration       // 0 = none; per-attempt deadline via context.WithTimeout
	RetryOnStatus     []int               // HTTP status codes to retry (nil => 429,500,502,503,504)
	RespectRetryAfter bool                // honor Retry-After on 429/503
	AllowMethods      []string            // methods eligible for retry; nil => idempotent set
	Retryable         func(*Attempt) bool // optional override; when set it FULLY governs the retry decision, bypassing the idempotency gate (a non-replayable body still cannot be retried)
	Budget            *RetryBudget        // optional token-bucket cap on retry fraction (nil = unlimited)

	// fromLegacy marks a policy translated from options.RetryConfig / --retry. Such
	// policies are method-agnostic (they retry any method) to preserve historical
	// behavior. Unexported so only the legacy mapper can set it.
	fromLegacy bool
	// maxReplayBytes is the body-buffer ceiling, filled in at policy resolution
	// time from the Client config (or DefaultMaxReplayBytes for the one-shot path).
	// 0 means unlimited.
	maxReplayBytes int64
}

// Attempt is the decision input passed to a custom Retryable function.
type Attempt struct {
	Method   string         // request method
	Number   int            // 1-based attempt index
	Response *http.Response // nil on transport error
	Err      error          // nil on HTTP response
	Started  time.Time      // when this attempt began
}

// Backoff computes the delay before the retry that follows a given attempt
// (1-based). Implementations are pure except for jitter, which draws from the
// supplied *rand.Rand so tests can inject a seeded source for determinism.
type Backoff interface {
	Delay(attempt int, rnd *rand.Rand) time.Duration
}

// constantBackoff returns the same delay for every attempt.
type constantBackoff struct{ d time.Duration }

func (c constantBackoff) Delay(attempt int, rnd *rand.Rand) time.Duration { return c.d }

// ConstantBackoff returns a Backoff that always waits d (mirrors the legacy
// RetryDelay-when-set behavior).
func ConstantBackoff(d time.Duration) Backoff { return constantBackoff{d} }

// exponentialBackoff implements base*2^(attempt-1) capped at max. When jitter is
// true it applies "equal jitter" (d/2 + rand[0,d/2]); when false it is fully
// deterministic (used by the legacy mapping so historical timing tests stay
// stable).
type exponentialBackoff struct {
	base   time.Duration
	max    time.Duration
	jitter bool
}

func (e exponentialBackoff) Delay(attempt int, rnd *rand.Rand) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := e.base
	for i := 1; i < attempt && d < e.max; i++ {
		d *= 2
	}
	if e.max > 0 && d > e.max {
		d = e.max
	}
	if !e.jitter || d <= 0 {
		return d
	}
	half := d / 2
	return half + time.Duration(rnd.Int63n(int64(half)+1))
}

// ExponentialJitter returns an exponential backoff: base*2^(attempt-1) capped at
// max, with equal jitter (each delay is in [d/2, d]). Equal jitter (rather than
// AWS full jitter) preserves a usable lower bound on the delay.
func ExponentialJitter(base, max time.Duration) Backoff {
	return exponentialBackoff{base: base, max: max, jitter: true}
}

// DefaultRetryPolicy returns the recommended idempotency-aware policy for the
// given attempt count: exponential jitter (100ms base, 5s cap), retrying
// 429/500/502/503/504, honoring Retry-After.
func DefaultRetryPolicy(maxAttempts int) RetryPolicy {
	return RetryPolicy{
		MaxAttempts:       maxAttempts,
		Backoff:           ExponentialJitter(100*time.Millisecond, 5*time.Second),
		RetryOnStatus:     []int{429, 500, 502, 503, 504},
		RespectRetryAfter: true,
	}
}

// maxAttempts returns the effective attempt count (at least 1).
func (p *RetryPolicy) maxAttempts() int {
	if p == nil || p.MaxAttempts < 1 {
		return 1
	}
	return p.MaxAttempts
}

// backoff returns the configured Backoff or the default exponential jitter.
func (p *RetryPolicy) backoff() Backoff {
	if p.Backoff != nil {
		return p.Backoff
	}
	return ExponentialJitter(100*time.Millisecond, 5*time.Second)
}

// replayCap returns the body-buffer ceiling (0 = unlimited).
func (p *RetryPolicy) replayCap() int64 { return p.maxReplayBytes }

// methodEligible reports whether method is in the policy's eligible set.
func (p *RetryPolicy) methodEligible(method string) bool {
	if len(p.AllowMethods) > 0 {
		for _, m := range p.AllowMethods {
			if strings.EqualFold(m, method) {
				return true
			}
		}
		return false
	}
	return idempotentMethods[strings.ToUpper(method)]
}

// eligibleForRetry reports whether a request is allowed to be retried at all,
// independent of the per-attempt outcome. Legacy policies are always eligible
// (method-agnostic); otherwise the method must be idempotent or the request must
// carry a non-empty Idempotency-Key header.
func (p *RetryPolicy) eligibleForRetry(req *http.Request) bool {
	if p.fromLegacy {
		return true
	}
	if p.methodEligible(req.Method) {
		return true
	}
	return hasIdempotencyKey(req)
}

// hasIdempotencyKey reports whether the request carries a non-empty
// Idempotency-Key header (case-insensitive via canonical header lookup).
func hasIdempotencyKey(req *http.Request) bool {
	return strings.TrimSpace(req.Header.Get("Idempotency-Key")) != ""
}

// wantRetry decides whether an attempt's outcome warrants a retry. A custom
// Retryable override, when set, FULLY governs the decision — it bypasses the
// idempotency eligibility gate, so a caller can opt a POST into retries by
// returning true (the caller takes responsibility). A non-replayable body still
// prevents the retry physically; that is enforced by the caller, not here.
//
// Without an override, the request must be eligible (idempotent method,
// AllowMethods, or an Idempotency-Key) AND the outcome must be a retryable
// transport error or status code.
func (p *RetryPolicy) wantRetry(att *Attempt, eligible bool) bool {
	if p.Retryable != nil {
		return p.Retryable(att)
	}
	if !eligible {
		return false
	}
	return p.outcomeRetryable(att)
}

// outcomeRetryable reports whether an attempt's transport error or HTTP status is
// retryable under the built-in policy.
func (p *RetryPolicy) outcomeRetryable(att *Attempt) bool {
	if att.Err != nil {
		k := classifyTransportError(att.Err)
		return k == KindConnect || k == KindTimeout
	}
	if att.Response != nil {
		return shouldRetry(att.Response.StatusCode, p.RetryOnStatus)
	}
	return false
}

// legacyPolicyFromRetryConfig translates the legacy options.RetryConfig (set
// directly or via the --retry flag) into a method-agnostic RetryPolicy, so the
// historical behavior — retrying ANY method that has a (re-sendable) body — is
// preserved. Returns nil when no retries are configured.
//
// Mapping (normative):
//
//	MaxAttempts       = MaxRetries + 1
//	Backoff           = ConstantBackoff(RetryDelay)      if RetryDelay > 0
//	                    deterministic 100ms*2^(n-1) (cap 5s) otherwise
//	RetryOnStatus     = RetryOnHTTP (nil => default 429/500/502/503/504)
//	RespectRetryAfter = true
//	fromLegacy        = true (method-agnostic)
func legacyPolicyFromRetryConfig(rc *options.RetryConfig) *RetryPolicy {
	if rc == nil || rc.MaxRetries <= 0 {
		return nil
	}
	var bo Backoff
	if rc.RetryDelay > 0 {
		bo = ConstantBackoff(rc.RetryDelay)
	} else {
		// Deterministic (no jitter) to match the historical schedule exactly.
		bo = exponentialBackoff{base: 100 * time.Millisecond, max: 5 * time.Second}
	}
	return &RetryPolicy{
		MaxAttempts:       rc.MaxRetries + 1,
		Backoff:           bo,
		RetryOnStatus:     rc.RetryOnHTTP,
		RespectRetryAfter: true,
		fromLegacy:        true,
		maxReplayBytes:    DefaultMaxReplayBytes,
	}
}

// withReplayCap returns a shallow copy of the policy with its body-buffer ceiling
// set. Copying avoids mutating a policy that may be shared across requests.
func (p *RetryPolicy) withReplayCap(n int64) *RetryPolicy {
	eff := *p
	eff.maxReplayBytes = n
	return &eff
}

// newRand returns a per-request *rand.Rand. math/rand.Rand is not safe for
// concurrent use, so each request owns its own source (never shared between
// goroutines); this keeps jitter race-free while remaining injectable in tests.
func newRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

// parseRetryAfter parses an HTTP Retry-After header value, supporting both
// delta-seconds and an HTTP-date. It never errors: an absent or unparseable
// value yields 0 (no enforced wait).
func parseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	v := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs <= 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}
