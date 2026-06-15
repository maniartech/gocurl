# Observability: Tracing, Metrics, Logging

> Status: Draft for review · Spec 06

## Goals

- Make every request observable through three vendor-neutral, dependency-free
  primitives in the core package: a **Tracer**, a **Metrics** collector, and a
  **Logger**, plus a set of **request lifecycle hooks**
  (`OnRequest`/`OnRetry`/`OnResponse`/`OnError`).
- Wire these primitives through the **single execution pipeline** (`doRequest`
  in `process.go` and the retry loop in `retry.go`) so they fire correctly for
  the "parse once, execute many" model — one prepared `Request`, many `Do`
  calls over a pooled `Client`.
- Provide first-class **OpenTelemetry** and **Prometheus** adapters as
  *separate subpackages* (`observability/otel`, `observability/prometheus`) so
  the core module pulls in **zero** new third-party dependencies.
- Standardize **trace/span propagation** and **request-id** headers, building on
  the existing `applyRequestID` / `X-Request-ID` handling and on the
  `Middleware` chain.
- Make **secret redaction mandatory and centralized**: all observability output
  (logs, span attributes, metric labels) must route through one redaction path
  built on `errors.go`'s `RedactHeaders` / `IsSensitiveHeader`, unifying the
  currently-duplicated redaction logic in `verbose.go`.
- Keep all instrumentation **cheap when disabled** (nil interfaces ⇒ no-op, no
  allocation on the hot path) and **safe under concurrency** (the `Client` and
  its tracer/metrics/logger are shared across goroutines).

## Non-goals

- No bundled OTLP exporter, collector config, or metrics HTTP endpoint — the
  adapters emit into the user's existing OTel SDK / Prometheus registry; wiring
  exporters stays in user code.
- No log-level filtering framework, log rotation, or sampling policy. We expose
  a minimal structured `Logger` interface; routing/sampling is the adapter's job.
- No replacement of curl-style `-v` verbose output (`verbose.go` stays as the
  human-facing CLI debug stream). This spec only makes verbose *share* the
  redaction path; the two are complementary.
- No distributed-context store beyond standard `context.Context` propagation.
- No new perf claims. Instrumentation targets near-zero overhead when disabled
  and small, bounded overhead when enabled; any number we cite must come from a
  reproducible benchmark.

## Design

### Core interfaces (package `gocurl`, dependency-free)

All three are interfaces so users can plug adapters without the core importing
OTel or Prometheus. A nil value means "disabled" and is never called.

```go
// Logger is a minimal structured logger. Implementations must be safe for
// concurrent use. Fields carry already-redacted values (see Redaction).
type Logger interface {
    Log(ctx context.Context, level Level, msg string, fields ...Field)
}

type Level int8
const (
    LevelDebug Level = iota
    LevelInfo
    LevelWarn
    LevelError
)

// Field is a single structured key/value log/attr pair.
type Field struct {
    Key   string
    Value any
}
func String(k, v string) Field      // helpers
func Int(k string, v int) Field
func Duration(k string, v time.Duration) Field
func Err(err error) Field           // key "error"
```

```go
// Tracer creates a span around one logical request (covering all retry
// attempts). A nil Tracer disables tracing entirely.
type Tracer interface {
    // StartSpan returns a child context carrying the span and the span itself.
    StartSpan(ctx context.Context, name string, attrs ...Field) (context.Context, Span)
}

// Span is the active span for an in-flight request. End is always called once.
type Span interface {
    SetAttributes(attrs ...Field)
    AddEvent(name string, attrs ...Field) // e.g. "retry", "redirect"
    RecordError(err error)
    End()
}
```

```go
// Metrics collects request-level measurements. Implementations must be safe
// for concurrent use and must not block. A nil Metrics disables metrics.
type Metrics interface {
    IncRequest(info RequestInfo)                 // total requests started
    IncInFlight(delta int)                       // +1 at start, -1 at end
    ObserveLatency(d time.Duration, info ResultInfo) // wall time of the logical request
    IncRetry(info RequestInfo)                   // one per retry attempt beyond the first
    IncError(class ErrorClass, info RequestInfo) // one per failed logical request
}

// RequestInfo / ResultInfo carry low-cardinality, pre-redacted labels.
type RequestInfo struct {
    Method string
    Host   string // host only — never full URL (cardinality + secrets)
}
type ResultInfo struct {
    RequestInfo
    StatusCode int        // 0 if no response
    Class      ErrorClass // ClassNone on success
}
```

`ErrorClass` is defined in **Spec 05 (Errors & classification)**; this spec
consumes it (`ClassNone`, `ClassTimeout`, `ClassTemporary`, `ClassRetryable`,
`ClassPermanent`). If 05 is not yet merged, this spec ships a minimal enum and
05 absorbs it.

### Lifecycle hooks (lightweight, synchronous)

Hooks are for users who want callbacks without implementing a full Tracer.
They are plain function fields on a `Hooks` struct; each is optional (nil-safe).

```go
type Hooks struct {
    OnRequest  func(ctx context.Context, req *http.Request)
    OnRetry    func(ctx context.Context, req *http.Request, attempt int, lastErr error, lastResp *http.Response)
    OnResponse func(ctx context.Context, req *http.Request, resp *http.Response, elapsed time.Duration)
    OnError    func(ctx context.Context, req *http.Request, err error, class ErrorClass)
}
```

Hooks receive the live `*http.Request`/`*http.Response`. The pipeline NEVER
hands raw headers to logs/spans/metrics without redaction, but hooks get the
real objects (the user already owns them); documentation warns that anything a
hook forwards to a sink must be redacted by the user.

### Options (package `gocurl`, matches locked naming)

```go
func WithTracer(t Tracer) Option
func WithMetrics(m Metrics) Option
func WithLogger(l Logger) Option
func WithHooks(h Hooks) Option
func WithRequestIDFunc(fn func() string) Option // generates an ID when none present
```

These set fields on the unexported `config`; the `Client` holds the resolved
`Tracer`/`Metrics`/`Logger`/`Hooks`. Package-level `Curl*` wrappers use the
default client (no observability unless configured), preserving back-compat.

### Where it plugs in (build on real code)

Observability is implemented as a single internal **instrumentation
middleware** composed into the chain described by the locked
`Middleware`/`Handler` model, wrapping the existing pipeline:

- **Span + timing + in-flight + request-count** start in `doRequest`
  (`process.go`) immediately after `ValidateOptions`, around the call to
  `executeWithRetries`. `IncInFlight(+1)` / `IncInFlight(-1)` and
  `ObserveLatency` bracket the whole logical request (all attempts).
- **`OnRequest`** fires after `CreateRequest` + `ApplyMiddleware`, just before
  execution, so it sees the final outgoing request.
- **`OnRetry` + `IncRetry` + span event "retry"** fire inside `retryLoop`
  (`retry.go`) at the top of each `attempt > 0` iteration, reusing the existing
  `attempt`/`retries` counters and `needsRetry` decision. This is the only new
  call site added to `retry.go`.
- **`OnResponse`** + `printResponseVerbose` coexist after a successful return
  from `executeWithRetries`.
- **`OnError` + `IncError` + `span.RecordError`** fire on the error return
  paths, classifying via Spec 05's `Classify(err, resp)`.
- **`span.End()`** is deferred so it always runs exactly once.

```go
// instrumentation is the internal middleware that ties the pipeline to the
// configured sinks. Conceptually:
func (c *Client) instrument(next Handler) Handler {
    return func(req *http.Request) (*http.Response, error) {
        ctx := req.Context()
        info := RequestInfo{Method: req.Method, Host: req.URL.Hostname()}

        ctx, span := c.tracer.StartSpan(ctx, "gocurl.request",
            String("http.method", info.Method), String("http.host", info.Host))
        defer span.End()
        req = req.WithContext(ctx)

        c.metrics.IncRequest(info)
        c.metrics.IncInFlight(1)
        defer c.metrics.IncInFlight(-1)
        c.hooks.onRequest(ctx, req)

        start := time.Now()
        resp, err := next(req) // -> executeWithRetries
        elapsed := time.Since(start)

        if err != nil {
            class := Classify(err, resp)        // Spec 05
            span.RecordError(err)
            c.metrics.IncError(class, info)
            c.metrics.ObserveLatency(elapsed, ResultInfo{info, 0, class})
            c.hooks.onError(ctx, req, err, class)
            c.log(ctx, LevelError, req, resp, elapsed, err)
            return nil, err
        }
        c.metrics.ObserveLatency(elapsed, ResultInfo{info, resp.StatusCode, ClassNone})
        c.hooks.onResponse(ctx, req, resp, elapsed)
        c.log(ctx, LevelInfo, req, resp, elapsed, nil)
        return resp, nil
    }
}
```

The `c.tracer`/`c.metrics`/`c.hooks` fields resolve to **no-op singletons**
when the user did not configure them, so every call above is a cheap method
call on an empty struct (no branch-per-sink on the hot path, no allocation).

### Request-ID & trace propagation

Build on `applyRequestID` in `process.go` (currently sets `X-Request-ID` from
`opts.RequestID`). New behavior, applied inside instrumentation BEFORE
`next(req)`:

1. If `req` already carries `X-Request-ID`, keep it (idempotent across retries).
2. Else if `opts.RequestID` is set, use it (unchanged behavior).
3. Else if `WithRequestIDFunc` is configured, generate one and set the header.
4. The resolved request id is added as a span attribute (`request.id`) and a
   log field, and passed to `RecordError`/hooks via context.
5. When a `Tracer` adapter implements propagation (OTel), it injects
   `traceparent`/`tracestate` into `req.Header` from the active span context.
   The core never hard-codes W3C headers; injection lives in the OTel adapter
   via an optional `Propagator` it owns.

Request id and trace headers are set once on the prepared request and survive
`cloneRequest` in `retry.go` (which copies headers), so all attempts share them.

### Redaction (mandatory, unified)

A single internal helper governs every value that leaves the process via an
observability sink:

```go
// redactedHeaders returns a copy safe for logs/spans, delegating to errors.go.
func redactedHeaders(h http.Header) map[string][]string {
    return RedactHeaders(h) // errors.go — IsSensitiveHeader drives it
}
```

- Logs emit headers only through `redactedHeaders`; the URL is logged via
  `sanitizeURL` (errors.go) so query secrets (`api_key`, `token`, …) are
  stripped.
- Metric labels are restricted to **method**, **host**, **status code**, and
  **error class** — never full URL or header values — both for secret-safety and
  to bound cardinality.
- Span attributes follow the same rule; any header attribute is redacted first.
- **Cleanup:** `verbose.go` currently has its own lowercase `isSensitiveHeader`
  with a *different* list than `errors.go`'s `sensitiveHeaders`/
  `IsSensitiveHeader`. This spec consolidates verbose to call the exported
  `IsSensitiveHeader`, so there is exactly one source of truth. The union of the
  two lists (adds `x-auth-token`, `auth-token` from verbose) becomes the
  canonical set in `errors.go`.

## Behavior & edge cases

- **All sinks disabled:** `WithTracer`/`WithMetrics`/`WithLogger`/`WithHooks`
  unset ⇒ instrumentation resolves to no-op structs; behavior is byte-identical
  to today and adds no measurable overhead (verified by benchmark).
- **Retries:** the logical request = one span, one latency observation, one
  `IncRequest`. `IncRetry` fires once per extra attempt; `OnRetry` carries the
  attempt index and the previous error/response. A retried request reuses the
  same request id and trace headers (set once, preserved by `cloneRequest`).
- **Context cancellation / timeout:** `retryLoop`'s existing context checks
  return an error; instrumentation classifies it (`ClassTimeout` for
  `DeadlineExceeded`, etc.), records it on the span, increments `IncError`, and
  still calls `span.End()` and `IncInFlight(-1)` via defer.
- **Streaming body:** latency is measured to **response-headers-received** (the
  return of `executeWithRetries`), NOT to body EOF, because `Do` returns the
  live streamed body. This is documented; a separate `OnBodyClose`-style hook is
  an open question, not in scope here.
- **Panicking sinks:** a panic in a user Logger/Tracer/Metrics/Hook must not
  corrupt the request. Each invocation is wrapped so a panic is recovered,
  logged once at `LevelError` (if a logger exists), and swallowed — observability
  must never take down a request.
- **Concurrency:** the `Client`'s tracer/metrics/logger are shared and called
  from many goroutines; the contract requires them to be concurrency-safe. The
  no-op defaults and the Prometheus adapter (atomic counters / `sync`-safe
  collectors) satisfy this.
- **High cardinality guard:** host is taken from `req.URL.Hostname()` (no port,
  no path, no userinfo). Adapters may further bucket hosts; the core does not.
- **Nil response on error:** `ResultInfo.StatusCode == 0` and the latency
  observation still fires so error latency is visible.
- **CLI:** `cmd/gocurl` can opt into a stderr `Logger` adapter; `-v` verbose
  output is unaffected and continues through `VerboseWriter`.

## Acceptance criteria / Definition of Done

- [ ] `Tracer`, `Span`, `Metrics`, `Logger`, `Level`, `Field`, `Hooks`,
      `RequestInfo`, `ResultInfo` defined in core `gocurl` package with **no new
      third-party imports** (`go list -deps` shows no otel/prometheus in core).
- [ ] `WithTracer`, `WithMetrics`, `WithLogger`, `WithHooks`,
      `WithRequestIDFunc` options implemented as `Option func(*config) error`.
- [ ] Instrumentation middleware fires `IncRequest`, `IncInFlight(±1)`,
      `ObserveLatency`, `IncRetry`, `IncError`, span start/end, and all four
      hooks at the documented call sites in `process.go` / `retry.go`.
- [ ] One logical request (with N retries) ⇒ exactly 1 span, 1 `IncRequest`,
      1 `ObserveLatency`, N `IncRetry`; verified by a test with a fake `Metrics`
      and a server returning retryable 503s.
- [ ] Request id is set once, preserved across retry clones, surfaced as span
      attribute `request.id` and a log field; existing `X-Request-ID` /
      `opts.RequestID` behavior unchanged (existing `cookies_requestid_test.go`
      tests still pass).
- [ ] `verbose.go` redaction uses the exported `IsSensitiveHeader`; the
      duplicate lowercase `isSensitiveHeader` list is removed and the canonical
      `sensitiveHeaders` set in `errors.go` includes `x-auth-token`/`auth-token`.
      A test asserts no sensitive header value appears in any log/span/metric.
- [ ] Nil sinks ⇒ no-op; a benchmark (`BenchmarkDo_NoObservability` vs
      `BenchmarkDo_FullObservability`) demonstrates the disabled path adds no
      allocations.
- [ ] Panic in a user-supplied sink is recovered and does not fail the request
      (test with a panicking Logger/Hook).
- [ ] `observability/otel` subpackage: adapters implementing `Tracer`/`Logger`
      from an `otel` `TracerProvider`, with `traceparent` propagation; lives in
      its own go file set, depends on `go.opentelemetry.io/otel` only there.
- [ ] `observability/prometheus` subpackage: a `Metrics` adapter registering
      `gocurl_requests_total`, `gocurl_in_flight`, `gocurl_request_duration_seconds`
      (histogram), `gocurl_retries_total`, `gocurl_errors_total{class}` against a
      `prometheus.Registerer`; concurrency-safe; example in package doc.
- [ ] `go test -race ./...` passes including new concurrent instrumentation tests.
- [ ] Package docs / examples show: enabling Prometheus, enabling OTel tracing,
      and a custom `Logger` with redaction guaranteed by the core.

## Dependencies

- **Spec 01 — Client/Option/Request model**: provides `New`, `Client`, `Option`,
  the `config` struct, and `Prepare`/`Do`.
- **Spec 02 — Middleware/Handler chain**: provides `Handler`/`Middleware` and the
  composition point where instrumentation is inserted.
- **Spec 05 — Errors & classification**: provides `ErrorClass` and
  `Classify(err, resp)` used by `IncError`/`OnError`/`span.RecordError`.
- Builds directly on existing code: `process.go` (`doRequest`, `CreateRequest`,
  `ApplyMiddleware`, `applyRequestID`), `retry.go` (`retryLoop`, `needsRetry`),
  `errors.go` (`RedactHeaders`, `IsSensitiveHeader`, `sanitizeURL`),
  `verbose.go` (`VerboseWriter`, redaction unification),
  `options/options.go` (`RequestID`).

## Open questions / decisions to confirm in review

- **Latency boundary**: measure to response-headers (proposed) vs. to body
  EOF. Headers-received is correct for the streaming `Do` contract, but some
  users want full-transfer latency — do we add an optional `OnBodyClose` hook /
  `gocurl_body_transfer_duration_seconds` metric in a later spec?
- **Metric naming/units**: `gocurl_request_duration_seconds` (Prometheus
  convention, base-unit seconds) — confirm prefix `gocurl_` and the default
  histogram buckets (proposed: `prometheus.DefBuckets`).
- **Logger vs slog**: should the core `Logger` be a thin shim over
  `log/slog` (Go 1.21+) instead of a bespoke interface, with a `slog` adapter as
  the default? Affects the minimum Go version and `Field` design.
- **`ErrorClass` ownership**: if Spec 05 lands after this, we ship a minimal
  enum here and 05 must adopt the exact names — confirm the name set now.
- **Trace propagation default**: should the OTel adapter inject W3C
  `traceparent` by default, or only when the user opts in via a `Propagator`?
- **Hook execution model**: synchronous (proposed, simplest, ordering-clear) vs.
  fire-and-forget goroutine. Synchronous means a slow hook slows the request —
  acceptable given the panic-recovery guard, or do we time-box hooks?
- **Canonical sensitive-header list**: confirm merging verbose's
  `x-auth-token`/`auth-token` into `errors.go` and whether to also add
  `x-amz-security-token`, `x-csrf-token`, and `www-authenticate`.
