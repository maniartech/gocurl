# gocurl Production Roadmap & Task Tracker

This is the **durable, idempotent** task tracker for evolving gocurl into a production-grade
HTTP client. It is the source of truth for *what is done and what remains*. The specs in
[`specs/`](specs/README.md) define *how*; this file tracks *progress*.

## How to use this tracker (idempotency contract)

- **Every task is self-contained and re-runnable.** A task is "done" only when its
  Definition of Done (DoD) holds. Re-running a done task is a no-op — the DoD is an
  invariant, not a one-time action. If you lose context, the checkboxes + DoD tell you
  exactly what is already true and what is left.
- **Status legend:** `[ ]` not started · `[~]` in progress · `[x]` done (DoD verified).
- **Global DoD (applies to every implementation task):**
  - `gofmt -l` clean, `go vet ./...` clean, `go build ./...` succeeds.
  - `go test -short -race ./...` passes (hermetic — no live network; gate any network test
    behind `testing.Short()`).
  - New/changed behavior has whitebox (sibling `_test.go`) and/or blackbox (`tests/`)
    coverage per Spec 09.
  - **Back-compat preserved** (existing `Curl*` API keeps working) until the v1 milestone.
  - **No new performance claims** unless backed by a benchmark (Spec 10).
- **Sequencing:** do milestones in order; within a milestone, respect task `deps`. M0 must
  complete before any code.
- The harness task list (TaskCreate) is session-scoped scratch; **this file is the
  persistent tracker** — update the checkboxes here as work lands.

---

## Milestone 0 — Ratify & reconcile specs *(no code)*

Goal: a self-consistent, hand-off-ready spec set. Depends on user ratification of the
decisions in [specs/README.md](specs/README.md#decisions-to-ratify-before-implementation-milestone-0).

- [ ] **M0-T1 — Ratify decisions A–G.** Walk the ratification list with the user; record
  the approved choice for each (A1–A4, B1–B2, C1–C3, D1–D6, E1–E6, F1–F4, G1).
  *DoD: every decision marked approved/changed; no "unconfirmed" left in the list.*
- [ ] **M0-T2 — Apply decisions to the owning specs.** Update Spec 02 (Request surface),
  05 (BodySource), 08 (error taxonomy), 12 (middleware), 03 (defaults/redirect) to the
  ratified signatures; make all other specs *reference* them.
  *DoD: each concept defined in exactly one spec; signatures identical across specs.*
- [ ] **M0-T3 — Fix stale cross-references.** Replace every inline `Spec NN` reference and
  the Spec 00 diagram labels with canonical numbers (00–13).
  *DoD: no cross-reference points at the wrong document; `specs/README.md` index matches.*
- [ ] **M0-T4 — Lock the default-config table** (Spec 01/03) and the context-precedence
  section (Spec 01).
  *DoD: concrete default values and ctx precedence written and reviewed.*

---

## Milestone 1 — Client / Request / Middleware foundation *(the keystone)* — **DONE**

Specs 01, 02, 12. Establishes parse-once/execute-many without breaking the current API.
Decisions A–G were adopted on the recommended defaults (user approved "go ahead"); the
implemented code is now the source of truth for those choices.

- [x] **M1-T1 — `Handler`/`Middleware` types + `chain` + legacy adapter** (`middleware.go`).
  *DoD met: outermost-first composition test; `FromMiddlewareFunc` success + error tests.*
- [x] **M1-T2 — `config` + functional `Option` set** (`config.go`). *DoD met: every option
  + error path tested; defaults asserted; options 100% covered.*
- [x] **M1-T3 — `Client` (`New`, concurrency-safe), `Do(ctx, *Request)`** through the
  middleware chain to the existing retry engine. *DoD met: streams live body; concurrent
  `Do` race test green; per-request redirect honored on a SHARED client via request-context.*
- [x] **M1-T4 — Immutable `Request` + `Prepare`/`PrepareNoEnv`/`PrepareWithVars` + builders**
  (`request.go`). *DoD met: parse-once; builder methods return clones (immutability tested);
  `WithVars` re-bind; programmatic `NewRequest` + option ctors.*
- [~] **M1-T5 — Default Client / package wrappers.** **Deviation (intentional):** the
  package-level `Curl*` funcs were NOT repointed onto a single shared `Client`, because a
  configured Client has a fixed transport and that would regress per-command TLS/proxy/
  `--insecure` flags. They keep their existing per-config pooled engine (no regression); the
  new `Client` is the explicit configure-once/reuse surface. *(`CloseDefault`/E5 test hook
  not needed since no shared default Client was introduced.)*
- [x] **M1-T6 — `Shutdown(ctx)`/`Close()` + per-Client transport ownership** (E1/E2). *DoD
  met: `buildOwnedTransport` gives each Client its own transport; Shutdown drains in-flight
  (timeout test); Close marks closed and frees idle conns.*

> Tests added: `middleware_test.go`, `config_test.go`, `request_internal_test.go`,
> `client_internal_test.go`, `m1_coverage_test.go` (whitebox, realistic paths at/near 100%),
> and `tests/client_blackbox_test.go` (use-cases: prepare-once/execute-many, concurrent
> reuse under -race, per-tenant `WithVars`, configured follow-redirects).

---

## Milestone 2 — Transport & connection management — **DONE** *(uncommitted, under review)*

Spec 03. deps: M1.

- [x] **M2-T1 — Per-Client owned, idle-tuned transport** with the M0-T4 defaults
  (MaxIdleConns=100, PerHost=10, MaxConnsPerHost=0). `config.buildTransport()` replaces
  `buildOwnedTransport`; `WithMaxIdleConns`/`WithMaxIdleConnsPerHost`/`WithMaxConnsPerHost`
  added; `WithTransport` override honored. *DoD met: defaults asserted on the built
  `*http.Transport`; `0=unlimited`.*
- [x] **M2-T2 — Timeout taxonomy** — `WithConnectTimeout` (via `DialContext`),
  `WithTLSHandshakeTimeout`, `WithResponseHeaderTimeout`, `WithIdleConnTimeout`,
  `WithExpectContinueTimeout`, plus overall `WithTimeout`. Per-request `--max-time` via
  context; context precedence documented. *DoD met: options tested; transport fields asserted.*
- [x] **M2-T3 — `RedirectPolicy{Follow,Max,Allow}` + `WithRedirectPolicy`** (D3). The
  `Allow` hook composes through the request-context redirect seam, so it works on a shared
  Client and is the seam the SSRF guard (M7) will use. *DoD met: follow/max/allow unit
  tests + blackbox test blocking a redirect to a disallowed host.*
- [x] **M2-T4 — HTTP/2 config** — `WithHTTP2(bool)` (default on, ForceAttemptHTTP2 +
  ConfigureTransport) and `WithHTTP2Only(allowH2C bool)` (→ `*http2.Transport`). HTTP/3
  documented as out-of-scope/future. *DoD met: real HTTP/2 round-trip blackbox test; h2c
  transport-type unit test.*

> Also fixed a pre-existing **`retry.go` bug** surfaced by the redirect Allow hook:
> `retryLoop` dropped the error (and closed the returned body) once retries were exhausted;
> it now propagates the last error and preserves the returned response body.
>
> Tests added: `transport_internal_test.go` (whitebox — options, defaults, `buildTransport`
> variants, redirect hook) and `tests/transport_blackbox_test.go` (redirect-allow block,
> HTTP/2 round-trip, tuned client). Full suite green under `go test -short -race ./...`.

---

## Milestone 3 — Body model & streaming — **DONE** *(uncommitted, under review)*

Spec 05. deps: M1.

- [x] **M3-T1 — `BodySource` interface (`options` pkg) + `BytesBody`/`StringBody`/`FileBody`/
  `ReaderBody`** (B1). *DoD met: each source has Open/Len/Rewindable tests; ReaderBody is
  single-use (non-rewindable).*
- [x] **M3-T2 — Streaming request upload + live response streaming.** `CreateRequest` uses
  `opts.BodyStream` (sets `Content-Length` and a `GetBody` for rewindable sources); `-T`
  now streams via `FileBody`; `executeWithRetries` only buffers when retries are enabled and
  there is no `GetBody`, so the default client streams uploads straight through. Builders:
  `WithBodySource`/`WithBodyFile`, `Stream(...)` option. *DoD met: ~2MB upload streamed;
  chunked response consumed incrementally; non-rewindable ReaderBody upload works.*
- [x] **M3-T3 — `MultipartBody` with cancellation-safe `io.Pipe`** (F1). Closing the body
  reader unblocks the writer goroutine; the boundary is stable so `ContentType()` matches
  across opens; rewindable only when all parts are `Path`-based. *DoD met: 50× open+close
  goroutine-leak test; path and reader round-trips; missing-file error propagates.*
- [x] **M3-T4 — Size limits vs streaming** (F2): documented that the in-memory body cap
  (`validateBody`, currently inactive on the live path — wired in M7) applies only to
  `BytesBody`; streaming sources are exempt. *Reconciliation noted; no active cap to bypass yet.*

> Also: `executeAttempt` now replays via `GetBody` (streaming-friendly) when available,
> falling back to buffered bytes. Tests: `body_internal_test.go` (whitebox, body.go ~100%
> bar one defensive multipart error branch) and `tests/streaming_blackbox_test.go` (file
> upload, `-T`, multipart, non-rewindable reader, chunked response). `-short -race` green.

---

## Milestone 4 — Error model

Spec 08. deps: M1.

- [x] **M4-T1 — `Kind` taxonomy on `GocurlError` + `KindOf`/`Timeout`/`Temporary`/`Retryable`**
  (C1/C2), `errors.Is/As` friendly. *DoD: table test mapping causes→Kind; redaction in
  error strings.* — Kind enum + 9 kinds, sentinels `ErrParse…ErrBodyRead`, `classifyTransportError`,
  new constructors (`ServerStatus/BodyRead/Connect/TLS/Timeout/Canceled`), `scrubErrorString`
  backstop; classification wired into the live boundaries (`doRequest`, `Client.Do`, retry
  exhaustion). Whitebox table tests + blackbox connect/timeout/canceled tests.
- [x] **M4-T2 — Status-code policy** (C3): no error on 4xx/5xx by default; opt-in helper.
  *DoD: documented + tested.* — `WithFailOnStatus` / `options.FailOnError` / `-f`/`--fail`;
  `failOnStatus` returns `ServerStatusError` *with* the response; convenience wrappers return
  the response alongside the error. CLI maps `Kind`→curl exit codes.

---

## Milestone 5 — Resilience

Spec 04. deps: M3 (replay), M4 (classification), M1-T1 (middleware).

- [x] **M5-T1 — Idempotency-aware retry middleware** (backoff+jitter, max attempts,
  per-attempt deadline, Retry-After). *DoD: never retries non-idempotent by default;
  replays via `BodySource`; fault-injection tests (G1) green.* — `RetryPolicy`/`Attempt`/
  `Backoff` (`ExponentialJitter` equal-jitter + `ConstantBackoff`), `DefaultRetryPolicy`
  excludes POST/PATCH/CONNECT, Idempotency-Key escape hatch, GetBody/buffered replay with
  `WithMaxReplayBytes` cap, per-attempt (`context.WithTimeout`, retryable) vs parent-context
  (terminal) split, `MaxElapsed` ceiling, Retry-After (delta + HTTP-date), drain+close discarded
  bodies, `RetryBudget`. Engine shared by one-shot + Client paths via `executeWithRetries(…,
  policy, rnd)`. `WithRetry`/`WithRetryAttempts`/`WithRetryBudget`/`WithMaxReplayBytes` +
  `Request.WithRetryPolicy`.
- [x] **M5-T2 — Legacy `RetryConfig` → `RetryPolicy` mapping** (D5). *DoD: existing
  `retry_test.go` behavior preserved or explicitly migrated; mapping table is normative.* —
  `legacyPolicyFromRetryConfig` (method-agnostic, deterministic no-jitter backoff); `retry_test.go`
  passes UNCHANGED; `--retry` blackbox test confirms POST still retried.
- [x] **M5-T3 — Circuit breaker (optional middleware), concurrency-safe.** *DoD: trips/half-open/
  reset state-machine test; race-clean.* — per-host rolling-window breaker (`CircuitBreaker`/
  `WithCircuitBreaker`/`ErrCircuitOpen`), counts only the final loop outcome, single-probe
  half-open, race-clean.
- [x] **M5-T4 — Client-side rate limiter (optional middleware).** *DoD: token-bucket test;
  race-clean.* — zero-dependency `TokenBucket` behind a pluggable `Limiter`; `RateLimiter`/
  `WithRateLimit`; composed `breaker → limiter → user-mw → retry → transport`.
- Deferred: hedging (`WithHedging`, EXPERIMENTAL) — out of the M5 ROADMAP scope; revisit later.

---

## Milestone 6 — Observability

Spec 06. deps: M1-T1, M4.

- [x] **M6-T1 — Vendor-neutral `Tracer`/`Metrics`/`Logger` interfaces + lifecycle hooks**;
  consume `Kind` (C1). *DoD: no vendor import in core; hooks fire in order (test).* —
  `observability.go`: `Logger`/`Level`/`Field`, `Tracer`/`Span`, `Metrics`, `Hooks`,
  `RequestInfo`/`ResultInfo` (reuse M4 `Kind`); `WithTracer`/`WithMetrics`/`WithLogger`/
  `WithHooks`/`WithRequestIDFunc`; no-op sinks; per-sink panic recovery; instrumentation
  middleware OUTERMOST in `Client.Do`, installed only when a sink/hook is configured
  (zero overhead disabled — benchmarks `BenchmarkDo_No/FullObservability`). `go list -deps`
  confirms no vendor import in core. Redaction unified on `errors.go IsSensitiveHeader`
  (`x-auth-token`/`auth-token` merged); `verbose.go` dedup removed.
- [x] **M6-T2 — Request-ID/trace propagation** (build on `RequestID`). *DoD: propagation test.* —
  request-id kept/generated once, preserved across retry clones, surfaced as span attr +
  log field; per-retry observability (IncRetry/OnRetry/span event) threaded via request
  context so `retry.go` stays decoupled.
- [x] **M6-T3 — OTel adapter + Prometheus-friendly metrics adapter in subpackages.** *DoD:
  adapters compile behind their own go files; example test.* — `observability/prometheus`
  (5 collectors against a `Registerer`, concurrency-safe, end-to-end test through a real
  `Client`) and `observability/otel` (`Tracer`/`Span` + W3C `traceparent` `PropagationMiddleware`)
  as SEPARATE modules so the core stays dependency-free.

---

## Milestone 7 — Security

Spec 07. deps: M1-T1, M2-T3.

- [x] **M7-T1 — Opt-in SSRF guard** (`SSRFPolicy`/`WithSSRFGuard`, D2): pre-flight + per-redirect
  (via `RedirectPolicy.Allow`). *DoD: blocks link-local/loopback/RFC1918/metadata unless
  allow-listed; tested.* — `security_ssrf.go`: `SSRFPolicy`/`DefaultSSRFPolicy`/`CheckSSRF`/
  `SSRFGuard`/`ErrSSRFBlocked` (KindValidation, non-retryable); IPv4/IPv6 + metadata coverage,
  allow-list precedence; enforced as outermost system middleware AND on every redirect hop via
  the request-context redirect seam (public→metadata 302 blocked, hermetic test).
- [x] **M7-T2 — Redaction middleware** wired everywhere (logs/verbose/errors). *DoD: secret
  never appears in any stream (test across modes).* — unified on `errors.go IsSensitiveHeader`
  + `sanitizeURL` (userinfo + query secrets), `RedactURL`/`RedactCommand` exported; M6 already
  routed logs/spans through it. **TLS pinning hardened**: checked in addition to chain
  verification (fail-closed on a bad pin) unless `--insecure` — pinning no longer forces
  `InsecureSkipVerify` (tested: good pin succeeds, bad pin fails closed).
- [x] **M7-T3 — Runtime input validation on the live path** (wire the dead `options`
  validators), reconciled with streaming (F2). *DoD: plaintext-auth-over-http policy +
  header/body checks active on `Do`.* — `options.ValidateRequest` (method-token, header
  count/size + forbidden Host/Content-Length/Transfer-Encoding, body/form/query caps,
  secure-auth) called from `ValidateRequestOptions`; streaming bodies exempt from the body cap.
  Plaintext auth fails closed by default; `WithAllowInsecureAuth`/`GOCURL_ALLOW_INSECURE_AUTH=1`
  override. `validateMethod` relaxed to RFC7230 tokens (curl-compat).
- [x] **M7-T4 — Proxy auth incl. username-only; TLS `WithTLS`/`WithTLSConfig`** (D1). *DoD:
  username-only proxy authenticates; `WithTLS` flows through `LoadTLSConfig`.* — `proxy`
  sends `Proxy-Authorization`/userinfo whenever a username is present (RFC 7617 empty password);
  `WithTLSConfig` flows a `*tls.Config` through `LoadTLSConfig` via `baseOptions`.

---

## Milestone 8 — CLI

Spec 13. deps: M1, M4.

- [x] **M8-T1 — Refactor `main` → `run(args, stdout, stderr) int`.** *DoD met: `run` takes the
  argv slice + explicit writers and is reentrant (local `flag.FlagSet`, never the global
  `flag.CommandLine` — fixed a latent re-registration panic); in-process tests drive every path,
  raising `cmd/gocurl` coverage from ~45% to **97.3%**.*
- [x] **M8-T2 — Exit codes from `Kind`** (replace string matching) + `--fail`. *DoD met:
  `getExitCode` maps `Kind`→curl codes (parse/validation 2/3, connect 7, timeout 28, tls 35,
  server-status 22); parse/tokenize/convert failures now return typed `ParseError` (KindParse)
  from `api.go`/`request.go` so an unknown flag exits 2 by `Kind`, not string-matching; table
  test per Kind + `--fail` exit-22 test.*
- [x] **M8-T3 — Output modes + parity test** (body printed once; `-v` redaction; `-i`/`-o`/
  `-w`). *DoD met: verbose trace → stderr, body → stdout (pipe-clean `gocurl -v url | jq`),
  `-o`/`-s`/`-i`/`-w` covered in-process and black-box; body-printed-once asserted across every
  flag combination; `-v` redaction asserted on both streams; library/CLI parity test sends an
  identical request on the wire for the same command via `gocurl.Curl` and the CLI argv.*

---

## Milestone 9 — Testing & quality hardening

Spec 09. deps: ongoing; finalize here.

- [x] **M9-T1 — Curl-compat corpus + parser fuzz targets.** `internal/corpus` (`//go:embed
  compat.json`, 9 verified cases: 3 each Stripe/GitHub/OpenAI) is run by BOTH a whitebox parse
  assertion (`TestCurlCompatCorpus_Parse`, reuses `parseCmd`) and a blackbox echo-server execution
  (`TestCurlCompatCorpus_Execute`, doc host swapped for the test server); adding a doc command is a
  one-line JSON append. `FuzzTokenize` (tokenizer) and `FuzzParseCommand` (root) with seeds; the
  parse fuzzer stays filesystem-free by skipping `@`-token inputs (all convert-time reads are
  `@`-triggered). Both fuzzed 20s+ (3.8M / 0.8M execs) with zero crashers.
- [x] **M9-T2 — Fault-injection** is already exercised by the M5 retry tests (RoundTripper fault
  injection in `m5_client_test.go`/`retry_test.go`); M9 adds the fuzz/leak/soak layers on top.
- [x] **M9-T3 — Leak + reuse detection.** `TestClient_Do_NoGoroutineLeak` (NumGoroutine settle loop
  around 200 `Client.Do` calls), `TestClient_Do_ReusesConnections` (`Server.Config.ConnState`
  counter asserts one pooled keep-alive conn for 50 sequential requests), and a `-short`-gated
  `TestClient_Soak` (bounded 3000-request loop, stable goroutines, writes cpu/mem pprof when
  `GOCURL_PROFILE` is set).
- [x] **M9-T4 — CI gates + CONTRIBUTING.** `.github/workflows/ci.yml` now enforces a coverage floor
  (75% overall via `-coverpkg=./...`; baseline 76.1%) and a 30s fuzz smoke per target, on top of
  gofmt/vet/build/`-short -race`. CONTRIBUTING.md documents the whitebox/blackbox placement rules,
  the corpus, and the fuzz/leak/soak commands.

---

## Milestone 10 — Benchmarking

Spec 10. deps: M2, M3.

- [ ] **M10-T1 — Comparative benchmark suite vs raw `net/http`** over a shared httptest
  server (construction, round-trip, concurrent throughput, allocs/op). *DoD: reproducible;
  results documented honestly (parity framing, never "faster").*
- [ ] **M10-T2 — Benchmark regression detection in CI.** *DoD: CI flags >X% regressions.*

---

## Milestone 11 — API stability & v1 release

Spec 11. deps: all prior milestones.

- [ ] **M11-T1 — Unexport/move internals to `internal/`** (`Process`, `CreateHTTPClient`,
  `CreateRequest`, `HandleOutput`, `ApplyMiddleware`, `ArgsToOptions`). *DoD: public surface
  matches the Spec 11 v1 keep-list; build green.*
- [ ] **M11-T2 — Deprecations finalized** (`Process`/`Execute`/legacy `MiddlewareFunc`
  adapter) with notices + migration guide. *DoD: MIGRATION.md written; godoc deprecation tags.*
- [ ] **M11-T3 — Cut `v1.0.0`** — annotated tag, CHANGELOG, README/VISION aligned. *DoD:
  tag pushed; `go install ...@v1.0.0` works.*

---

## Working agreement

1. Land work milestone by milestone; keep the suite green at every commit.
2. Update the checkboxes in this file as the durable record.
3. If a task uncovers a spec gap, fix the spec first (Milestone 0 discipline), then code.
4. Honesty over hype: every claim is backed by a test or benchmark, or it isn't made.
