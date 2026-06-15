# Middleware & Handler Model

> Status: Draft for review · Spec 12

This spec **owns the middleware contract**. Every cross-cutting behavior in the production
client — retries (Spec 04), observability (Spec 06), redaction and the SSRF guard (Spec
07) — is expressed as middleware over a common `Handler` type. Specs that mention
middleware reference the types defined *here*; they do not redefine them.

## Goals

- Define one composition primitive (`Handler` + `Middleware`) that wraps the pooled
  `net/http` transport at the bottom of the chain (see Spec 00 layering, Spec 03 transport).
- Make composition **order explicit and deterministic**, so retries, tracing, and
  redaction interact predictably.
- Provide a back-compat adapter for the existing `middlewares.MiddlewareFunc`
  (`func(*http.Request) (*http.Request, error)`) so current users are not broken.
- Keep the core dependency-free; concrete middlewares live next to their owning spec.

## Non-goals

- Not a plugin system or DI framework (the legacy `objective.md` "plugin" idea stays dead).
- Not request *routing* or server-side middleware — this is a client round-trip chain only.

## Design

A `Handler` is a function that turns a `*http.Request` into a `*http.Response` (the same
shape as `http.RoundTripper.RoundTrip`, but as a func for easy composition). A `Middleware`
wraps a `Handler` to produce a new one.

```go
// Handler executes a single request and returns its response. The innermost
// Handler is backed by the Client's pooled net/http transport (Spec 03).
type Handler func(*http.Request) (*http.Response, error)

// Middleware wraps a Handler to add behavior (retry, tracing, redaction, …).
type Middleware func(next Handler) Handler

// WithMiddleware appends user middleware to the chain (Option, Spec 01).
func WithMiddleware(mw ...Middleware) Option
```

### Composition order

Middlewares are applied so that the **first** middleware in the chain is the **outermost**
(runs first on the way out, last on the way back). The Client builds the chain once at
`New` time:

```go
// chain builds: mw[0]( mw[1]( … ( base ) ) )
func chain(base Handler, mw ...Middleware) Handler {
    for i := len(mw) - 1; i >= 0; i-- {
        base = mw[i](base)
    }
    return base
}
```

The Client installs built-in middleware in a **fixed, documented order**, with user
middleware (from `WithMiddleware`) inserted at a defined point. Outermost → innermost:

1. **Recover/observe boundary** — panic safety + the top span/timer (Spec 06).
2. **Request-ID / trace propagation** (Spec 06).
3. **Redaction context** — marks sensitive headers so logging/verbose never leak (Spec 07).
4. **SSRF guard** — pre-flight + per-redirect host checks (Spec 07).
5. **Circuit breaker** (Spec 04, optional).
6. **Rate limiter** (Spec 04, optional).
7. **Retry** (Spec 04) — must sit *inside* breaker/limiter so each attempt is counted, but
   *outside* the per-attempt timeout it applies.
8. **User middleware** (`WithMiddleware`) — between retry and the metrics-per-attempt layer
   by default; see Open questions for whether users can opt to wrap retries.
9. **Per-attempt metrics/logging** (Spec 06) — observes each individual attempt.
10. **base Handler** → pooled transport (Spec 03).

> Rationale: retry is *inside* the breaker (so a tripped breaker short-circuits all
> attempts) and *outside* per-attempt metrics (so each attempt is measured). This ordering
> is normative and tested (Spec 09).

### Legacy adapter

The current `options.RequestOptions.Middleware` field is `[]middlewares.MiddlewareFunc`
(`func(*http.Request) (*http.Request, error)` — a request *mutator*, applied in
`ApplyMiddleware` in `process.go`). It is adapted, not removed:

```go
// FromMiddlewareFunc adapts a legacy request-mutating MiddlewareFunc into a Middleware.
func FromMiddlewareFunc(f middlewares.MiddlewareFunc) Middleware {
    return func(next Handler) Handler {
        return func(req *http.Request) (*http.Response, error) {
            r, err := f(req)
            if err != nil {
                return nil, err
            }
            return next(r)
        }
    }
}
```

Existing `opts.Middleware` entries are wrapped via this adapter and run at the user
position in the chain, preserving v0.x behavior.

## Behavior & edge cases

- A middleware that returns a non-nil response **and** a non-nil error must let the chain
  close the body; document the ownership rule (mirrors `http.RoundTripper`).
- Middleware must propagate `req.Context()`; cancellation is observed by every layer.
- Retry middleware re-invokes `next` with a **rewound** body via the `BodySource` from Spec
  05 (never `io.ReadAll`); it must not assume the body is replayable for non-idempotent
  methods (Spec 04).
- Built-in middleware is only added when its feature is configured (no breaker unless
  `WithCircuitBreaker`, etc.), keeping the zero-config chain minimal.

## Acceptance criteria / Definition of Done

- [ ] `Handler` and `Middleware` types exported from package `gocurl`, used by Specs 04/06/07.
- [ ] `chain` composition with a table-driven test asserting outermost-first order.
- [ ] Built-in middleware ordering implemented exactly as listed and covered by a test that
      asserts relative order (e.g. retry inside breaker, metrics innermost).
- [ ] `FromMiddlewareFunc` adapter + a test proving existing `opts.Middleware` still runs.
- [ ] `WithMiddleware` option documented with the insertion point.
- [ ] No import of any observability/telemetry vendor in this file.

## Dependencies

- Spec 00 (layering), Spec 01 (`Option`, Client owns the chain).
- Consumed by Spec 04 (retry/breaker/limiter), Spec 06 (tracing/metrics), Spec 07
  (redaction/SSRF). Body rewinding uses Spec 05's `BodySource`.

## Open questions / decisions to confirm in review

- **User middleware position:** default between retry and per-attempt metrics. Should we
  also offer `WithOuterMiddleware`/`WithInnerMiddleware` for users who need to wrap retries
  or sit closest to the wire? *Proposed: ship one `WithMiddleware` at the documented
  position for v1; add explicit outer/inner variants only if demand appears.*
- **Handler vs RoundTripper:** expose `http.RoundTripper` adapters too (so a `Handler` can
  be used as a `Transport` and vice versa)? *Proposed: provide `HandlerFromRoundTripper`
  and `RoundTripperFromHandler` helpers for interop.*
- **Per-attempt vs per-call observation:** confirm metrics middleware is installed twice
  (one outer call-level, one inner attempt-level) vs a single layer that emits both.
