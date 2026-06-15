# gocurl Production Specs

> Status: **Draft for review.** No implementation has started. These specs define the
> evolution of gocurl from a curl-ergonomic convenience client into a production-grade,
> mission-critical HTTP client — **without abandoning the curl ergonomics**. Read [Spec 00]
> first, then ratify the decisions below, then work the [ROADMAP](../ROADMAP.md).

The organizing principle is **parse once, execute many**: a curl command is an *authoring*
convenience, parsed once into an immutable `Request`, then executed many times over a
reusable, pooled `Client` built on `net/http`. We make **no** "zero-allocation / faster
than net/http" claims; the target is parity with a well-tuned `net/http` client plus better
ergonomics and reliability.

## Canonical spec index

This index is **authoritative** for numbering. (Some inline `Spec NN` references inside the
drafts predate reconciliation — trust this table; fixing them is ROADMAP Milestone 0.)

| # | Spec | Owns |
|---|------|------|
| 00 | [Overview & Architecture](00-overview-and-architecture.md) | layering & guiding principles |
| 01 | [Client API & Lifecycle](01-client-api.md) | the `Client` & `Option` surface, lifecycle |
| 02 | [Request Model & Curl Compatibility](02-request-model-and-curl-compat.md) | **the `Request` surface** |
| 03 | [Transport & Connection Management](03-transport-and-connections.md) | transport, pooling, redirect policy |
| 04 | [Resilience](04-resilience.md) | retry / circuit breaker / rate limiter |
| 05 | [Streaming & Bodies](05-streaming-and-bodies.md) | **the body model (`BodySource`)** |
| 06 | [Observability](06-observability.md) | tracer / metrics / logger interfaces |
| 07 | [Security](07-security.md) | TLS, SSRF guard, redaction, validation |
| 08 | [Error Model](08-error-model.md) | **error classification (`Kind`)** |
| 09 | [Testing & Quality](09-testing-and-quality.md) | test layering, fuzz, soak, fault injection |
| 10 | [Benchmarking](10-benchmarking.md) | honest perf methodology |
| 11 | [API Stability & Migration](11-api-stability-and-migration.md) | SemVer, v1 surface, migration |
| 12 | [Middleware & Handler Model](12-middleware.md) | **the middleware contract** |
| 13 | [CLI (`cmd/gocurl`)](13-cli.md) | CLI behavior & exit codes |

### Single-owner rule

To stop the same type being defined two ways, each concept has exactly one owning spec;
others **reference**, never redefine:

- **`Request` surface** (constructors, builder methods) → **Spec 02**.
- **Body model** (`BodySource`, replay, multipart, streaming) → **Spec 05**.
- **Error classification** (`Kind`, `Timeout/Temporary/Retryable`) → **Spec 08**.
- **Middleware contract** (`Handler`/`Middleware`) → **Spec 12**.
- **Transport defaults & redirect policy** → **Spec 03**.

## Decisions to ratify before implementation (Milestone 0)

These come from the adversarial consistency/completeness review of the drafts. Each has a
**recommendation** — approve, or tell me to change it. ROADMAP Milestone 0 applies the
ratified set back into the specs so they're internally consistent before code.

### A. `Request` surface (Spec 02 is sole owner)
- **A1 — `Prepare` env default:** `Prepare(cmd)` **expands env vars** (mirrors today's
  `CurlCommand`, back-compat). Add `PrepareNoEnv(cmd)` for the no-env case and
  `PrepareWithVars(vars, command)` (**vars first**, matching `CurlCommandWithVars`).
- **A2 — programmatic constructor:** keep two, distinctly named — `Client.Prepare(curl)`
  for the curl path and package `gocurl.NewRequest(method, url string, ...RequestOption)`
  for hand-built requests. (Resolves the `NewRequest` name collision.)
- **A3 — builder methods** (immutable; each returns a clone): `WithHeader`, `WithQuery`
  (not `WithQueryParam`), `WithBody`, `WithVars` (not `Rebind`), `WithContext`, `Clone`.
- **A4 — `WithBody` type:** takes a **`BodySource`** (Spec 05), with sugar
  `WithBodyBytes([]byte)` and `WithBodyReader(io.Reader)`. (Resolves the `io.Reader` vs
  `[]byte` conflict; unifies with the body model.)

### B. Body & replay model (Spec 05 is sole owner)
- **B1:** `BodySource{ Open() (io.ReadCloser,error); Len() (int64,bool); Rewindable() bool }`
  is the one abstraction. Spec 01 `WithBody` and Spec 04 replay reference it.
- **B2 — retry replay:** never silently `io.ReadAll`. Rewindable sources (BytesBody,
  FileBody) replay freely; a non-rewindable `ReaderBody` is buffered only up to
  `WithMaxReplayBytes` (opt-in, default off) and otherwise is **not retried**.

### C. Error model (Spec 08 is sole owner)
- **C1:** one taxonomy — `Kind` + `KindOf(err)` + `Timeout()/Temporary()/Retryable()`.
  Delete Spec 06's separate `ErrorClass`/`Classify`; observability consumes `Kind`.
- **C2 — retryable set:** `KindConnect` + `KindTimeout` + select `KindServerStatus`
  (429/502/503/504); **`KindTLS` is not retryable**. `Retryable()` reflects the **new**
  idempotency-aware engine (Spec 04), not legacy `retry.go`.
- **C3 — status codes:** 4xx/5xx do **not** error by default (the response is returned);
  CLI `--fail` and a library opt-in change that.

### D. Options & types
- **D1 — `WithTLS`:** `WithTLS(TLSOptions)` (curl-knobs struct feeding `LoadTLSConfig`, so
  it stays the single TLS source) **plus** `WithTLSConfig(*tls.Config)` as a documented
  advanced escape hatch. (Resolves the `TLSOptions` vs `*tls.Config` conflict.)
- **D2 — SSRF:** config struct is `SSRFPolicy`; option is `WithSSRFGuard(SSRFPolicy)`;
  the middleware constructor is `SSRFGuard(policy)`. Fix Specs 01/11 to use `SSRFPolicy`.
- **D3 — redirects:** define `RedirectPolicy{ Follow bool; Max int; Allow func(req, via) error }`
  owned by **Spec 03**; the SSRF per-redirect check composes via `Allow`. Keep
  `WithRedirectPolicy` in the v1 surface only once this type is defined.
- **D4 — connection defaults (Spec 03 table):** `MaxIdleConns=100`,
  `MaxIdleConnsPerHost=10`, `MaxConnsPerHost=0` (**0 = unlimited**, net/http semantics).
- **D5 — retry types:** `RetryPolicy` (new, public, idempotency-aware) and legacy
  `options.RetryConfig` (internal carrier) are distinct; Spec 04's mapping table is
  normative. Confirm `WithRetryAttempts(n)` sugar is wanted (not in the original brief).
- **D6 — `gocurl.Client` vs `HTTPClient`:** the existing `HTTPClient` interface stays the
  injection seam (`WithTransport`, replacing `CustomClient`); `Client` is the new
  high-level type. Document the relationship; no rename.

### E. Client lifecycle & concurrency (Spec 01) — fills production gaps
- **E1 — shutdown:** add `Shutdown(ctx) error` that drains in-flight requests (WaitGroup)
  and `Close()` that closes idle connections.
- **E2 — transport ownership:** each `Client` **owns its transport** (built at `New`), so
  `Close()` never degrades other clients. The process-global transport cache
  (`clientpool.go`) is used **only** by the default one-shot `Curl*` path. (Resolves the
  shared-transport-on-Close gap.)
- **E3 — cookie jar:** per-Client, created at `New`; relies on the concurrency-safe stdlib
  jar; persisted on `Shutdown`/explicit save. `-b`/`-c` semantics under a reused Client
  documented in Spec 02/13.
- **E4 — context precedence:** one normative section: `Do(ctx)` is the base; a `Request`'s
  `WithContext` merges; per-attempt timeout (Spec 04) derives a child; overall `Timeout`
  (Spec 03) is the ceiling. Define behavior when ctx is already cancelled.
- **E5 — default-Client test reset:** expose an internal `export_test.go` reset hook (no
  public API) so Spec 09 can reset pooled state between tests.
- **E6 — default config table:** enumerate all defaults in Spec 01.

### F. Streaming & security cross-cuts
- **F1 — multipart cancellation (Spec 05):** the `io.Pipe` writer goroutine selects on
  `ctx.Done()` and `pw.CloseWithError` on reader close — no goroutine leak. Tested by Spec 09.
- **F2 — validation vs streaming (Specs 05↔07):** `validateBody` size cap applies only to
  in-memory `BytesBody`; streaming sources are exempt (checked against `Content-Length`
  when known). Resolves the "07 rejects >10MB that 05 streams" contradiction.
- **F3 — backpressure (Spec 04):** document the aggregate `N × MaxReplayBytes` memory risk
  under concurrent reuse; optional `WithReplayBudget` is a future, not v1.
- **F4 — HTTP/2 (Specs 03↔04):** treat `RST_STREAM`/`GOAWAY` as retryable connection-level
  errors; note h2 timeout differences (`ReadIdleTimeout`/`PingTimeout`).

### G. Testing (Spec 09)
- **G1 — fault injection:** add a failure-injection harness (connection resets, partial/
  slow responses, mid-stream EOF, TLS handshake failure) to validate retry classification,
  breaker tripping, and per-attempt deadlines.

## How this set was produced

12 grounded spec-writers (one per concern) drafted in parallel, then an adversarial
consistency/completeness critic reviewed all drafts; its findings are distilled into the
ratification list above. The two specs the critic flagged as missing — Middleware (12) and
CLI (13) — have been added.
