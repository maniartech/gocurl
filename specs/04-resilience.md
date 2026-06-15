# Resilience: Retries, Backoff, Circuit Breaking

> Status: Draft for review · Spec 04

## Goals

- Replace the ad-hoc retry loop in `retry.go` (`executeWithRetries` → `retryLoop`) with an
  **idempotency-aware `RetryPolicy`** that is safe by default: it retries only idempotent methods and
  safe failure conditions, and **never silently retries a `POST`** (or any non-idempotent method) unless
  the caller explicitly opts in.
- Provide correct, production-grade backoff: exponential backoff **with jitter**, a configurable cap,
  bounded max attempts, optional **per-attempt deadlines**, and **`Retry-After` honoring** for `429`/`503`.
- Add a **retry budget** so retries cannot amplify load during a partial outage (the classic retry storm).
- Make request **body replay** explicit and bounded, building on `bufferRequestBody`/`cloneRequest` in
  `retry.go`, with documented tradeoffs for large and streaming bodies.
- Offer an optional **circuit breaker** and **client-side rate limiter**, expressed as `Middleware` in the
  round-trip middleware model from Spec 02, so they compose with retries rather than being bolted on.
- Offer optional, **experimental hedging** (parallel speculative attempts) behind an explicit flag.
- Be correct under `context` cancellation and deadlines: a cancelled or expired context stops retries
  immediately and surfaces a classified error, preserving today's behavior in `sleepWithContext` and
  `checkContextDuringRetry`.

## Non-goals

- No claim of being "faster" than a plain `net/http` client. Resilience here means *predictability under
  load* (no retry storms, no surprise replays, correct deadlines), not raw throughput.
- Not building a distributed/coordinated rate limiter or breaker (no shared state across processes). These
  are per-`Client`, in-process primitives. A distributed limiter can be supplied via the `Limiter` interface.
- Not retrying responses whose body the caller has already started streaming (`client.Do` returns a live
  body). Retry decisions happen **before** the body is handed to the caller; once streaming begins, a
  failure mid-read is the caller's to handle.
- Not implementing server-side anything. Breaker/limiter protect *this* client.

## Design

Resilience is layered. The **`RetryPolicy`** lives inside the execution engine (the successor to
`executeWithRetries`) because it must re-issue the prepared `*Request` and replay the buffered body.
The **circuit breaker** and **rate limiter** are `Middleware` (Spec 02) wrapping the retrying `Handler`,
so a single logical `client.Do` call flows: `rate limit → circuit breaker → [retry loop → transport]`.

```go
// RetryPolicy is an immutable value attached to a Client (WithRetry) or a prepared Request.
type RetryPolicy struct {
    MaxAttempts   int               // total attempts incl. the first; <=1 disables retries
    Backoff       Backoff           // delay schedule (default: ExponentialJitter)
    MaxElapsed    time.Duration     // 0 = unbounded; overall wall-clock budget across attempts
    PerAttempt    time.Duration     // 0 = none; per-attempt deadline derived via context.WithTimeout
    RetryOnStatus []int             // HTTP status codes to retry (default: 429,500,502,503,504)
    RespectRetryAfter bool          // honor Retry-After header on 429/503 (default true)
    AllowMethods  []string          // methods eligible for retry; nil => idempotent set below
    Retryable     func(*Attempt) bool // optional override; sees method, response, err, attempt no.
    Budget        *RetryBudget      // optional token-bucket cap on retry fraction (nil = unlimited)
}

// Attempt is the decision input passed to Retryable / classification.
type Attempt struct {
    Method   string
    Number   int            // 1-based attempt index
    Response *http.Response // nil on transport error
    Err      error          // nil on HTTP response
    Started  time.Time
}

// Backoff computes the delay before attempt n (1-based). Implementations are pure & deterministic
// except for jitter, which takes a *rand.Rand for testability.
type Backoff interface {
    Delay(attempt int, rnd *rand.Rand) time.Duration
}

// ExponentialJitter: base * 2^(attempt-1), capped at Max, with full jitter (AWS-style).
func ExponentialJitter(base, max time.Duration) Backoff
// ConstantBackoff mirrors today's RetryDelay-when-set behavior.
func ConstantBackoff(d time.Duration) Backoff
```

Idempotency is the load-bearing default. The default eligible set is curl/HTTP-safe:
`GET, HEAD, OPTIONS, TRACE, PUT, DELETE`. `POST`, `PATCH`, and `CONNECT` are **excluded** unless the
caller sets `AllowMethods` or an `Idempotency-Key` header is present (see edge cases). This is the single
biggest correctness upgrade over the current `needsRetry`, which retries any method that has a buffered body.

```go
// Default policy used when WithRetry is given attempts but no explicit policy.
func DefaultRetryPolicy(maxAttempts int) RetryPolicy

// Option wiring (Spec 01 naming). Both forms are accepted.
func WithRetry(p RetryPolicy) Option
func WithRetryAttempts(n int) Option  // sugar => DefaultRetryPolicy(n)

// Per-request override on the prepared Request (Spec 03 clone-with-overrides).
func (r *Request) WithRetryPolicy(p RetryPolicy) *Request
```

Circuit breaker and rate limiter as middleware (round-trip `Handler`/`Middleware` from Spec 02):

```go
type Middleware func(next Handler) Handler
type Handler func(*http.Request) (*http.Response, error)

// Circuit breaker: per-host (default) rolling-window breaker. Open => fast-fail with ErrCircuitOpen.
type BreakerConfig struct {
    FailureThreshold float64       // fraction in window that trips (default 0.5)
    MinRequests      int           // min samples before tripping (default 20)
    Window           time.Duration // rolling window (default 10s)
    OpenTimeout      time.Duration // half-open probe delay (default 5s)
    KeyFunc          func(*http.Request) string // default: req.URL.Host
    IsFailure        func(*http.Response, error) bool // default: err != nil || 5xx
}
func CircuitBreaker(cfg BreakerConfig) Middleware
func WithCircuitBreaker(cfg BreakerConfig) Option

// Rate limiter: client-side token bucket; blocks until a token or ctx done.
type Limiter interface { Wait(ctx context.Context) error }      // pluggable (e.g. golang.org/x/time/rate)
func RateLimiter(l Limiter) Middleware
func WithRateLimit(rps float64, burst int) Option               // builds an x/time/rate limiter
```

Retry budget (token bucket that refills with successful traffic; retries spend tokens):

```go
type RetryBudget struct {
    Ratio   float64       // max retries as fraction of requests (e.g. 0.1 => 10%)
    MinPerSec float64     // floor so low-traffic clients can still retry a little
}
func NewRetryBudget(ratio, minPerSec float64) *RetryBudget
```

Body replay is the same mechanism as today, made explicit and bounded. `bufferRequestBody` reads the body
into memory once before the loop; `cloneRequest` hands each attempt a fresh `bytes.Reader`. We add a cap
and a `GetBody` fast-path:

```go
// Buffer ceiling; bodies larger than this are NOT buffered and the request becomes non-retryable
// on attempt 2+ (single-shot). Default 1<<20 (1 MiB); 0 = unlimited (caller accepts the memory cost).
func WithMaxReplayBytes(n int64) Option
```

Hedging (experimental, off by default): after `HedgeDelay`, fire a second attempt in parallel; first
successful response wins, the loser's context is cancelled. Hedging is restricted to idempotent methods,
counts against `MaxAttempts` and the `RetryBudget`, and is incompatible with non-replayable bodies.

```go
type HedgeConfig struct { Delay time.Duration; MaxParallel int } // EXPERIMENTAL
func WithHedging(cfg HedgeConfig) Option
```

## Behavior & edge cases

- **Default safety / POST.** With `MaxAttempts > 1` but default policy, a `GET` that returns `503` is
  retried; a `POST` that returns `503` is **not** — it returns the response unchanged. This is a deliberate
  behavior change from the current `retryLoop`, which would retry the POST because it has a buffered body.
  The change must be called out in the changelog and migration notes.
- **Idempotency-Key escape hatch.** If a non-idempotent request carries a non-empty `Idempotency-Key`
  header, it is treated as retry-eligible (the server is expected to dedupe). This lets users opt a specific
  `POST` into retries without globally enabling unsafe behavior.
- **`Retry-After` honoring.** On `429`/`503` with `RespectRetryAfter`, parse both forms (delta-seconds and
  HTTP-date). The effective delay is `max(Retry-After, Backoff.Delay(n))`, clamped to the remaining
  `MaxElapsed`. A `Retry-After` that exceeds the remaining budget aborts retrying and returns the response.
- **Context & deadlines (preserve current behavior).** Before each attempt and during each backoff sleep,
  honor `ctx.Done()` exactly as `checkContextDuringRetry` and `sleepWithContext` do today (select on
  `ctx.Done()` vs `time.After`). A cancelled/expired context returns immediately; the error is classified
  `Timeout`/`Canceled` (Spec on errors) and is **not** retried. `PerAttempt` wraps each attempt in
  `context.WithTimeout(ctx, PerAttempt)` so one slow attempt cannot consume the whole `MaxElapsed`.
- **MaxElapsed vs MaxAttempts.** Both are ceilings; whichever trips first stops retrying. If the next
  computed delay would exceed remaining `MaxElapsed`, do not sleep — return the last result.
- **Body replay tradeoffs.** Buffering holds the full body in memory for the request's lifetime; for large
  uploads this is real memory pressure, and for truly streaming bodies (an unbounded `io.Reader`) it is
  impossible. Behavior: bodies ≤ `WithMaxReplayBytes` are buffered and replayable; larger or unknown-length
  streaming bodies are sent once and become non-retryable (attempt 2+ short-circuits with a
  `non-replayable body` reason). If the source provides `req.GetBody` (net/http convention), prefer it over
  buffering and skip the in-memory copy. The buffered-body close/reset in `bufferRequestBody` must remain
  race-clean.
- **Response body hygiene.** Every discarded attempt must `io.Copy(io.Discard, resp.Body)` then
  `resp.Body.Close()` so the pooled transport (`clientpool.go`) can reuse the connection — today's loop only
  `Close()`s, which can poison keep-alive on partially-read bodies. This is a fix, not just a port.
- **Circuit breaker ordering.** The breaker sits *outside* the retry loop, so an open circuit fast-fails
  before any attempt and before the limiter blocks. `ErrCircuitOpen` is classified non-retryable. Breaker
  failure accounting counts the *final* outcome of the retry loop, not each internal attempt (otherwise
  retries double-count and trip the breaker prematurely).
- **Rate limiter ordering.** The limiter is outermost; `Wait(ctx)` blocks (respecting ctx) before the
  breaker. Each retry attempt also passes through the limiter so retries are shaped by the same budget — but
  the `RetryBudget` is what prevents retries from monopolizing limiter tokens.
- **Retry budget exhaustion.** When the budget has no tokens, additional retries are suppressed and the
  current response/error is returned with a `retry budget exhausted` annotation; the first attempt is never
  budget-gated.
- **Hedging hazards.** Hedged requests must be idempotent and replayable; if either fails the requirement,
  hedging is silently disabled for that request (documented). Losing attempts are cancelled and their bodies
  drained+closed. Hedging interacts with the breaker: a hedge loss due to cancellation must not count as a
  breaker failure.
- **Error surface.** Exhaustion wraps the last error/response via `RetryError(url, attempts, err)` from
  `errors.go`, preserving `Unwrap` so `errors.Is(err, context.DeadlineExceeded)` and the new
  classification helpers work. Retry metadata (attempt count, last status, total elapsed) is attached for
  observability (Spec on observability) without changing the `GocurlError` shape incompatibly.
- **Back-compat with `RetryConfig`.** The legacy `options.RetryConfig` (`MaxRetries`, `RetryDelay`,
  `RetryOnHTTP`) and `--retry` (convert.go) continue to work and are translated into a `RetryPolicy`:
  `MaxAttempts = MaxRetries+1`, `Backoff = ConstantBackoff(RetryDelay)` when `RetryDelay>0` else
  `ExponentialJitter(100ms, 5s)` (matching `calculateSleepDuration`), `RetryOnStatus = RetryOnHTTP` or the
  current default set in `shouldRetry`. To avoid a silent behavior change for existing users, the legacy
  path keeps method-agnostic retry **only when** `RetryConfig` is set directly; the new `WithRetry`/policy
  path is idempotency-aware. This divergence is an Open question below.

## Acceptance criteria / Definition of Done

- [ ] `RetryPolicy`, `Backoff`, `ExponentialJitter`, `ConstantBackoff`, `RetryBudget`, `Attempt` implemented
      with godoc; `DefaultRetryPolicy` excludes `POST/PATCH/CONNECT`.
- [ ] `executeWithRetries`/`retryLoop` refactored to consume a `RetryPolicy`; existing `retry.go` tests pass
      or are updated with documented rationale.
- [ ] Default policy retries a `503` `GET` and does **not** retry a `503` `POST`; a `POST` with a non-empty
      `Idempotency-Key` *is* retried. Covered by table tests.
- [ ] Backoff is exponential with full jitter, capped; jitter is deterministic under an injected `*rand.Rand`
      so tests are stable.
- [ ] `Retry-After` (both delta-seconds and HTTP-date) is parsed and honored; delay = `max(Retry-After, backoff)`,
      clamped to remaining `MaxElapsed`; over-budget `Retry-After` aborts retrying.
- [ ] `PerAttempt` enforces a per-attempt `context.WithTimeout`; `MaxElapsed` bounds total wall clock;
      whichever ceiling trips first stops retries. Context cancellation/deadline aborts immediately and is
      non-retryable (parity with current `sleepWithContext`/`checkContextDuringRetry`).
- [ ] Discarded attempts drain+close the body (verified by a keep-alive reuse test against `httptest`).
- [ ] Body replay: bodies ≤ `WithMaxReplayBytes` retry; larger/streaming bodies are single-shot and
      short-circuit on attempt 2+ with a clear reason; `req.GetBody` is preferred when present.
- [ ] `CircuitBreaker` middleware trips after `MinRequests`+threshold, fast-fails with `ErrCircuitOpen`
      (non-retryable), half-opens after `OpenTimeout`, and counts only final loop outcomes.
- [ ] `RateLimiter` middleware blocks on `Wait(ctx)` respecting cancellation; `WithRateLimit(rps, burst)`
      wiring works end-to-end via `client.Do`.
- [ ] `RetryBudget` suppresses retries when exhausted; first attempt is never gated.
- [ ] Hedging is behind `WithHedging`, documented EXPERIMENTAL, idempotent+replayable only, drains losers,
      and does not mis-count breaker failures.
- [ ] Legacy `options.RetryConfig` and `--retry` map onto `RetryPolicy` with the documented translation; a
      blackbox CLI test (`tests/`) confirms `curl --retry 3` behavior is unchanged.
- [ ] Exhaustion returns `RetryError(...)` with intact `Unwrap`; `errors.Is(err, context.DeadlineExceeded)`
      holds for deadline cases.
- [ ] All new tests are hermetic and race-clean (`go test -race ./...`); no test reaches the network.

## Dependencies

- **Spec 01** (Client / Option) — `WithRetry`, `WithCircuitBreaker`, `WithRateLimit`, `WithHedging`,
  `WithMaxReplayBytes` are functional options on `*config`.
- **Spec 02** (Middleware model) — round-trip `Handler`/`Middleware`; breaker and limiter are middleware
  composed around the retrying handler. Requires the response-aware middleware type, not the current
  request-only `middlewares.MiddlewareFunc`.
- **Spec 03** (Prepared `Request`) — per-request `WithRetryPolicy`/clone-with-overrides and `req.GetBody`.
- **Errors spec** — classification (`Timeout`/`Temporary`/`Retryable`) consumed by `Retryable`/budget logic;
  builds on `GocurlError`/`RetryError` in `errors.go`.
- **Observability spec** — retry/breaker/limiter emit metrics, spans, and lifecycle hooks (attempt count,
  last status, elapsed, breaker state transitions).

## Open questions / decisions to confirm in review

- **Legacy divergence.** Should setting `options.RetryConfig` directly stay method-agnostic (current
  behavior) for strict back-compat, or should *all* paths become idempotency-aware in v0.x with a loud
  deprecation note? Proposed: keep legacy method-agnostic through v0.x, make idempotency-aware the only
  behavior in v1.
- **`WithMaxReplayBytes` default.** Is 1 MiB the right ceiling, or should the default be "buffer only if
  `Content-Length` is known and small, otherwise rely on `GetBody`"? Confirm the memory-vs-retryability
  tradeoff for typical API payloads.
- **Breaker scope.** Per-host by default — should the default key also include the port/scheme, and should
  we expose a per-route key for path-based APIs?
- **Rate limiter dependency.** Adopt `golang.org/x/time/rate` for the built-in `WithRateLimit`, or ship a
  zero-dependency token bucket and keep `Limiter` pluggable only? Proposed: depend on `x/time/rate` (already
  a common, vetted dep) behind the `Limiter` interface.
- **Hedging in v1?** Confirm hedging stays EXPERIMENTAL and is excluded from the v1 stability guarantee.
- **Retry-After cap.** Should an absurd `Retry-After` (e.g. hours) be capped by a configurable max even when
  under `MaxElapsed`, to avoid pathological waits? Proposed: clamp to `min(Retry-After, MaxElapsed)`.
