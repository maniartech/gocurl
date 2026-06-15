# Overview & Architecture

> Status: Draft for review · Spec 00

This is the umbrella spec for evolving **gocurl** from a curl-ergonomic convenience
layer into a **serious, production-grade HTTP client** — without abandoning the curl
ergonomics that are its reason to exist. Every other spec (01–11) references this one for
shared vocabulary, layering, and guiding principles. It supersedes the framing in the
legacy `design.md` (the "zero-allocation / military-grade / faster than net/http" language
there is dead and must not be reintroduced).

## Goals

- State the **production vision**: reliability and predictability under load, not raw
  req/s superiority. "High performance for mission-critical" means correct timeouts,
  connection reuse, idempotency-aware retries, backpressure, and observability.
- Pin down **THE ONE ARCHITECTURAL TRUTH** — **PARSE ONCE, EXECUTE MANY** — and make it
  the organizing principle of the codebase.
- Define the **layering**: Curl ergonomics (authoring) → `Request` (immutable, parsed
  once) → `Client` (reusable, pooled) → middleware chain → `net/http` transport.
- Show how a curl command flows through these layers at **authoring time** vs
  **request time**.
- Diagram the package/type relationships and map them onto today's real files
  (`api.go`, `convert.go`, `tokenizer/`, `process.go`, `clientpool.go`, `options/`,
  `middlewares/`, `retry.go`, `security.go`, `errors.go`, `verbose.go`).
- Enumerate the full spec set (00–11) and how the pieces relate.
- Fix the **guiding principles** (honesty, hermetic-testability, back-compat,
  reliability-over-microbenchmarks) that every later spec must obey.

## Non-goals

- **Not** replacing `net/http` or adopting `fasthttp`. The execution engine is Go's
  `net/http` plus `golang.org/x/net/http2`.
- **Not** specifying HTTP/3. quic-go is an optional **future** add-on, never v1.
- **Not** re-specifying work already shipped this cycle (honest README/VISION, parser
  fixes, streaming library behavior, transport caching, real deflate, cert pinning,
  unified TLS loading, verbose redaction). We build **on** that, not over it.
- **Not** turning gocurl into a code generator, SDK builder, or full curl (HTTP/HTTPS
  only; only flags that appear in real API docs).
- **Not** making any unbacked performance claim. The target is **parity** with a
  well-tuned `net/http` client, plus better ergonomics and reliability.

## Design

### The one architectural truth: parse once, execute many

A curl command string is an **authoring convenience**, never a per-request hot-path cost.
Today the high-level helpers (`CurlCommand` in `api.go`) re-run the full pipeline —
`preprocessMultilineCommand` → `tokenizer.Tokenize` → `expandEnvInTokens` →
`convertTokensToRequestOptions` → `doRequest` — on **every** call. That is fine for a
one-shot integration, but it is wasteful and unsafe to lean on for a mission-critical hot
path.

The production model splits the two costs cleanly:

- **Authoring time (once):** parse the curl string into an **immutable, prepared
  `Request`**. This is the artifact produced by `client.Prepare(command)`. All tokenizing,
  env expansion, flag conversion, and validation happen here, exactly once.
- **Request time (many):** execute that prepared `Request` over a **reusable, pooled,
  concurrency-safe `Client`** via `client.Do(ctx, req)`. No re-parsing, no per-request
  transport allocation.

This is the lens through which every other spec is written.

### Layering

```
  AUTHORING                                     EXECUTION
  =========                                     =========

  curl command string / []string
        │  (Spec 02: parser)
        │  preprocessMultilineCommand
        │  tokenizer.Tokenize  ──►  []tokenizer.Token
        │  expand env / vars        (env vs explicit Variables map)
        │  convertTokensToRequestOptions
        ▼
  options.RequestOptions  ── (immutable snapshot) ──►  Request   (Spec 01)
        │                                                 │  Clone + per-call overrides
        │                                                 ▼
        └───────────────────────────────────────►   Client.Do(ctx, req)   (Spec 01)
                                                          │
                                                          ▼
                                              ┌────────────────────────┐
                                              │   MIDDLEWARE CHAIN      │  (Spec 03)
                                              │  Handler / Middleware   │
                                              │  ─ request-id / logging │
                                              │  ─ redaction (security) │  (Spec 08)
                                              │  ─ SSRF guard           │  (Spec 08)
                                              │  ─ retry (idempotent)   │  (Spec 04)
                                              │  ─ circuit breaker      │  (Spec 04)
                                              │  ─ rate limiter         │  (Spec 04)
                                              │  ─ tracing / metrics    │  (Spec 05)
                                              └───────────┬────────────┘
                                                          ▼
                                              ┌────────────────────────┐
                                              │  base Handler           │
                                              │  net/http transport     │  (Spec 06)
                                              │  pooled, idle-tuned      │
                                              │  TLS / proxy / HTTP2     │  (Spec 07)
                                              └───────────┬────────────┘
                                                          ▼
                                                   live *http.Response
                                                   (streamed body)
```

The **innermost** Handler is the only place a real `*http.Request` touches the network. It
calls the pooled `http.RoundTripper` that `clientpool.go` builds today (`getRoundTripper` /
`newTransport`). Everything above it is composable behavior.

### Type & package relationships

```
  package gocurl
  ────────────────────────────────────────────────────────────────────
                                 ┌───────────────────────────────────┐
   New(opts ...Option)           │ Client                            │
       │  ────────────────────►  │  cfg     *config                  │
       │                         │  pool    (transport cache)        │
   Option func(*config) error    │  chain   Middleware composition   │
                                 │                                   │
                                 │  Prepare(cmd) (*Request, error)   │  parse once
                                 │  Do(ctx, *Request) (*http.Resp,…) │  execute many
                                 │  Curl / CurlString / CurlJSON /   │  convenience
                                 │  CurlBytes / CurlDownload         │  (mirror pkg funcs)
                                 └───────────────┬───────────────────┘
                                                 │ holds
                  ┌──────────────────────────────┼──────────────────────────────┐
                  ▼                               ▼                              ▼
        ┌──────────────────┐         ┌────────────────────────┐      ┌──────────────────┐
        │ Request          │         │ middleware chain        │      │ pooled transport │
        │ (immutable,      │         │ Handler / Middleware    │      │ clientpool.go    │
        │  cloneable)      │         │ (middlewares pkg, ext.) │      │ http.Transport + │
        │ wraps an opts    │         └────────────────────────┘      │ http2            │
        │ snapshot         │
        └──────────────────┘

  package tokenizer   →  Token, Tokenizer            (parse: lexing)
  package options     →  RequestOptions, RetryConfig, BasicAuth, FileUpload,
                         HTTPClient, Clone()          (parse target / config carrier)
  package middlewares →  MiddlewareFunc (legacy)      (to be superseded by Handler model)
  package proxy       →  ProxyConfig, NewTransport    (HTTP/HTTPS/SOCKS5)
```

### Focused Go API sketch (the anchor surface)

The names below are **locked** and reused verbatim by Specs 01–11. Signatures only; full
behavior lives in the referenced specs.

```go
package gocurl

// Client is reusable and concurrency-safe: created once, used for many requests.
// It holds immutable config, a pooled transport, and a composed middleware chain.
type Client struct { /* cfg *config; pool ...; chain Middleware */ }

// New constructs a Client from functional options (Spec 01).
func New(opts ...Option) (*Client, error)

// Option mutates an internal config during New. Spec 01 enumerates the set:
// WithTimeout, WithConnectTimeout, WithRetry, WithProxy, WithTLS, WithTransport,
// WithMiddleware, WithTracer, WithMetrics, WithLogger, WithRedirectPolicy,
// WithSSRFGuard, WithMaxConnsPerHost.
type Option func(*config) error

// Request is the immutable, prepared "parse once" artifact (Spec 01). It is built from
// a curl command or programmatically, and is cloneable with per-call overrides.
type Request struct { /* immutable snapshot of options.RequestOptions + parsed metadata */ }

// Prepare parses a curl command exactly ONCE into a reusable Request.
func (c *Client) Prepare(command string) (*Request, error)

// Do executes a prepared Request and returns the LIVE streamed body (Spec 01, 06).
// The caller owns resp.Body and must Close it.
func (c *Client) Do(ctx context.Context, req *Request) (*http.Response, error)

// Convenience methods mirror the package-level Curl* funcs in api.go.
func (c *Client) Curl(ctx context.Context, command ...string) (*http.Response, error)
func (c *Client) CurlString(ctx context.Context, command ...string) (string, *http.Response, error)
func (c *Client) CurlJSON(ctx context.Context, v any, command ...string) (*http.Response, error)
func (c *Client) CurlBytes(ctx context.Context, command ...string) ([]byte, *http.Response, error)
func (c *Client) CurlDownload(ctx context.Context, filepath string, command ...string) (int64, *http.Response, error)

// Middleware model (Spec 03). The base Handler wraps the pooled net/http transport.
type Handler func(*http.Request) (*http.Response, error)
type Middleware func(next Handler) Handler
```

### How a curl command flows: authoring vs request time

**Authoring time** (`client.Prepare("curl https://api.example.com -d a=1")`):
1. `preprocessMultilineCommand` strips `curl `, comments, line continuations (`api.go`).
2. `tokenizer.NewTokenizer().Tokenize(...)` lexes into `[]tokenizer.Token`.
3. Env/var expansion runs once — `expandEnvInTokens` for the default Client, or
   `expandVarsInTokens` with an explicit `Variables` map (no env leak, per `CurlWithVars`).
4. `convertTokensToRequestOptions` (`convert.go`) yields an `options.RequestOptions`.
5. Validation runs (`ValidateRequestOptions` via `ValidateOptions`).
6. The result is frozen into an immutable `*Request`.

**Request time** (`client.Do(ctx, req)`, repeatable and concurrent):
1. `Request.Clone()` + any per-call overrides produce the effective config (no re-parse).
2. `CreateRequest` (`process.go`) builds the `*http.Request` from the snapshot.
3. The composed middleware chain wraps the base Handler.
4. The base Handler runs the request over the pooled transport (`getRoundTripper`,
   `clientpool.go`), with retries handled as middleware (replacing today's inline
   `executeWithRetries` in `retry.go`).
5. The **live** `*http.Response` is returned, body streamed and unread — exactly as
   `doRequest` returns today. The library never writes to `os.Stdout` (that side effect
   lives only in the deprecated `Process`).

### Relationship to the existing code

Production gocurl is an additive layer over today's internals, not a rewrite:

- `convertTokensToRequestOptions` + `tokenizer/` remain the parser, now invoked once
  inside `Prepare` (Spec 02).
- `options.RequestOptions` (with its existing `Clone()`) is the snapshot a `Request`
  wraps; new config lives behind `Option`/`config` (Spec 01).
- `doRequest`/`CreateRequest`/`CreateHTTPClient` become the base Handler and the pooled
  execution core (Specs 01, 06).
- `clientpool.go`'s transport cache becomes the `Client`'s owned, explicit pool (Spec 06).
- `retry.go`'s `executeWithRetries`/`shouldRetry` are refactored into an
  idempotency-aware retry **middleware** (Spec 04).
- `middlewares.MiddlewareFunc` (request-only mutation) is superseded by the
  `Handler`/`Middleware` round-trip model (Spec 03), with a back-compat adapter.
- `errors.go`'s `GocurlError` + redaction helpers gain classification (Spec 11) and feed
  the redaction middleware (Spec 08).
- Package-level `Curl*` funcs in `api.go` become thin wrappers over a lazily-initialized
  default `Client` (Spec 01), preserving v0.x behavior.

## Behavior & edge cases

- **Concurrency:** `Client` and `Request` are safe for concurrent use; a prepared
  `Request` is immutable, so `Do` reads it without locking. Per-call mutation goes through
  `Clone()` (mirrors the documented thread-safety contract on `options.RequestOptions`).
- **Streaming preserved:** `Do` returns the live body; nothing buffers or prints unless the
  caller asks (`CurlString`/`CurlBytes`/`CurlJSON`/`CurlDownload`). `Process` stays
  deprecated as the only stdout-writing path.
- **Back-compat:** every existing package-level function keeps working via the default
  Client. No v0.x signature changes; v1 may unexport internals (`Process`,
  `CreateHTTPClient`, `CreateRequest`, etc.).
- **Honest perf:** any benchmark added (e.g. `benchmark_test.go`) measures parity, never
  claims superiority; skipped benchmarks may not back a public claim.
- **Authoring failures fail early:** parse/validation errors surface at `Prepare` time as
  classified `GocurlError`s, not mid-flight at `Do`.
- **No env leak:** the `Variables`-map path must never fall back to `os.Environ`, matching
  current `CurlWithVars` semantics.

## Acceptance criteria / Definition of Done

- [ ] This document is the single referenced anchor; Specs 01–11 cite it for the layering
      diagram and the locked names (`Client`, `New`, `Option`, `Request`, `Prepare`, `Do`,
      `Handler`, `Middleware`).
- [ ] The legacy `design.md` is marked superseded (no "zero-allocation / military-grade /
      faster-than-net-http" language survives in any active spec).
- [ ] The parse-once/execute-many split is described in terms of the real functions it
      replaces (`CurlCommand`, `doRequest`, `executeWithRetries`, `getRoundTripper`).
- [ ] The four guiding principles (honesty, hermetic-testability, back-compat,
      reliability-over-microbenchmarks) appear and are referenced by later specs.
- [ ] The full spec set (00–11) is enumerated with one-line scopes and dependency edges.
- [ ] Every locked type/function name in the API sketch matches the names mandated in the
      project brief.

## Guiding principles (referenced by all specs)

1. **Honesty.** No "zero-allocation", "faster than net/http", or "military-grade". Perf
   claims require a reproducible, un-skipped benchmark. The bar is parity + ergonomics +
   reliability.
2. **Hermetic testability.** Every behavior is testable without the network — whitebox
   siblings plus the blackbox `tests/` package (API + CLI subprocess), race-clean and
   hermetic, as already established.
3. **Backward compatibility.** v0.x public surface keeps working through the default
   Client; deprecations (e.g. `Process`) stay until v1, which may unexport internals.
4. **Reliability over microbenchmarks.** Correct timeouts, connection reuse,
   idempotency-aware retries, backpressure, and observability outrank raw throughput.

## Spec set (00–13) and how they relate

> **Canonical numbering.** This table and [`specs/README.md`](README.md) are the
> authoritative index. Some inline `Spec NN` cross-references inside the individual draft
> documents (and the parenthetical labels in the diagram above) predate this reconciliation
> and may be off by a number — trust this table and the README, not the inline references.
> Reconciling those stale references is the first task in `ROADMAP.md` (Milestone 0).

| # | Title | Scope (one line) | Builds on | Owns |
|---|-------|------------------|-----------|------|
| 00 | Overview & Architecture | This anchor: vision, parse-once truth, layering, principles. | — | the layering & principles |
| 01 | Client API & Lifecycle | `New`/`Option`/`config`, `Client`, `Do`, default-Client wrappers, lifecycle/shutdown. | 00 | the `Client` & `Option` surface |
| 02 | Request Model & Curl Compatibility | Immutable `Request`, parse-once, flag matrix, env vs `Variables`, builder methods. | 00, 01 | **the `Request` surface** |
| 03 | Transport & Connection Management | Owned pooled transport, idle tuning, timeouts, HTTP/1.1+2, `WithMaxConnsPerHost`. | 00, 01 | transport & pooling |
| 04 | Resilience | Idempotency-aware `RetryPolicy`, backoff+jitter, Retry-After, breaker, limiter. | 01, 12 | retry/breaker/limiter |
| 05 | Streaming & Bodies | Response/request streaming, **`BodySource`** body model, multipart, SSE, size limits. | 01, 02 | **the body model** |
| 06 | Observability | Tracer/metrics/logger interfaces, OTel adapter, lifecycle hooks, request-id. | 01, 12 | telemetry interfaces |
| 07 | Security | TLS hardening, SSRF guard, redaction, proxy auth, runtime input validation. | 03, 12 | security middleware |
| 08 | Error Model | Typed `GocurlError` taxonomy + classification (`Kind`, `Timeout/Temporary/Retryable`). | 04 | **error classification** |
| 09 | Testing & Quality | Whitebox/blackbox, fuzz, race, leak, soak, failure-injection, CI gates. | all | quality bar |
| 10 | Benchmarking | Honest comparative bench vs `net/http`, methodology, regression detection. | 03 | perf methodology |
| 11 | API Stability & Migration | SemVer, v1 surface, deprecations (`Process`, `Execute`), migration guide. | 01, 02 | versioning policy |
| 12 | Middleware & Handler Model | `Handler`/`Middleware` types, composition order, legacy `MiddlewareFunc` adapter. | 00, 01 | **the middleware contract** |
| 13 | CLI (`cmd/gocurl`) | Identical curl syntax, verbose redaction, exit codes, library parity. | 01, 02 | CLI behavior |

Dependency reading: 01 is the keystone after 00; **12 (middleware)** is the spine that 04,
06, and 07 hang off; 02 owns the `Request` surface and 05 owns the body model and 08 owns
error classification (other specs reference, never redefine, those); 03 is the transport
floor; 09–11 and 13 are cross-cutting consumers.

## Open questions / decisions to confirm in review

- **`Request` internals:** does `Request` wrap an `options.RequestOptions` snapshot
  directly, or a new reduced struct? Proposed: wrap a cloned `RequestOptions` for v0.x to
  minimize churn, with a path to a leaner type at v1. *(unconfirmed)*
- **Default Client lifetime:** lazily-initialized singleton vs initialized on first
  package call. Proposed: lazy `sync.Once` singleton that owns the existing transport
  cache. *(unconfirmed)*
- **Legacy `MiddlewareFunc` fate:** keep an adapter (`MiddlewareFunc` → `Middleware`)
  through v0.x and remove at v1, or deprecate immediately? Proposed: adapter + deprecation
  notice. *(unconfirmed)*
- **Where retries live:** confirm retries move entirely into middleware (Spec 04) and
  `executeWithRetries` is retired, vs a hybrid. Proposed: fully middleware. *(unconfirmed)*
- **Spec numbering:** the 00–11 titles above are proposed; confirm exact titles/scopes
  before authors start on dependent specs. *(unconfirmed)*
- **`Do` override mechanism:** per-call overrides as variadic `Option`-like args on `Do`
  vs an explicit `req.Clone().With(...)` builder. Proposed: `Request` builder methods to
  keep `Do` signature stable. *(unconfirmed)*
