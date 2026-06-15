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

## Milestone 1 — Client / Request / Middleware foundation *(the keystone)*

Specs 01, 02, 12. Establishes parse-once/execute-many without breaking the current API.

- [ ] **M1-T1 — `Handler`/`Middleware` types + `chain` + legacy adapter** (Spec 12).
  deps: M0. *DoD: types exported; composition-order test; `FromMiddlewareFunc` test proving
  `opts.Middleware` still runs.*
- [ ] **M1-T2 — `config` + functional `Option` set** (Spec 01). deps: M1-T1. *DoD: options
  compile; invalid options return errors; default-config test matches the M0-T4 table.*
- [ ] **M1-T3 — `Client` (`New`, fields, concurrency-safe), `Do(ctx, *Request)`** routing
  through the middleware chain to the existing `doRequest` engine. deps: M1-T2. *DoD:
  `Do` streams the live body; race test with concurrent `Do` on one Client.*
- [ ] **M1-T4 — Immutable `Request` + `Prepare`/`PrepareNoEnv`/`PrepareWithVars` + builder
  methods** (Spec 02, per ratified A1–A4). deps: M1-T3. *DoD: `Prepare` parses once;
  builder methods return clones (no shared-state mutation, race-tested); env semantics
  per A1.*
- [ ] **M1-T5 — Default `Client` + repoint package `Curl*` wrappers** (Spec 01). deps:
  M1-T3. *DoD: existing `Curl*`/`tests/` suite passes unchanged; an `export_test.go` reset
  hook exists (E5).*
- [ ] **M1-T6 — `Shutdown(ctx)`/`Close()` + per-Client transport ownership** (Spec 01/03,
  E1/E2). deps: M1-T3. *DoD: in-flight drain test; closing one Client does not affect
  another; no goroutine/conn leak (Spec 09 leak check).*

---

## Milestone 2 — Transport & connection management

Spec 03. deps: M1.

- [ ] **M2-T1 — Per-Client owned, idle-tuned transport** with the M0-T4 defaults;
  `WithMaxConnsPerHost`/`WithTransport`. *DoD: defaults asserted; `0=unlimited` honored.*
- [ ] **M2-T2 — Timeout taxonomy** (connect, TLS handshake, response-header, idle, overall)
  exposed as options; context-precedence per E4. *DoD: each timeout has a httptest-driven test.*
- [ ] **M2-T3 — `RedirectPolicy` type + `WithRedirectPolicy`** (D3). *DoD: follow/no-follow,
  max-redirs, and custom `Allow` covered; integrates with current `-L` default of 30.*
- [ ] **M2-T4 — HTTP/1.1 + HTTP/2 (h2/h2c) config**; document HTTP/3 as future. *DoD: h2
  round-trip test; h2c opt-in test.*

---

## Milestone 3 — Body model & streaming

Spec 05. deps: M1.

- [ ] **M3-T1 — `BodySource` interface + `BytesBody`/`ReaderBody`/`FileBody`** (B1). *DoD:
  each source has Open/Len/Rewindable tests.*
- [ ] **M3-T2 — Streaming request upload (`-T`/`WithBody`) and live response streaming
  contract** (body ownership rules). *DoD: large-file upload streams (no full buffering),
  memory bounded under test.*
- [ ] **M3-T3 — `MultipartBody` with cancellation-safe `io.Pipe`** (F1). *DoD: goroutine
  leak test passes on early cancel/close.*
- [ ] **M3-T4 — Size limits reconciled with validation** (F2): streaming sources exempt
  from the in-memory cap. *DoD: >10MB streams succeed; oversized in-memory body rejected.*

---

## Milestone 4 — Error model

Spec 08. deps: M1.

- [ ] **M4-T1 — `Kind` taxonomy on `GocurlError` + `KindOf`/`Timeout`/`Temporary`/`Retryable`**
  (C1/C2), `errors.Is/As` friendly. *DoD: table test mapping causes→Kind; redaction in
  error strings.*
- [ ] **M4-T2 — Status-code policy** (C3): no error on 4xx/5xx by default; opt-in helper.
  *DoD: documented + tested.*

---

## Milestone 5 — Resilience

Spec 04. deps: M3 (replay), M4 (classification), M1-T1 (middleware).

- [ ] **M5-T1 — Idempotency-aware retry middleware** (backoff+jitter, max attempts,
  per-attempt deadline, Retry-After). *DoD: never retries non-idempotent by default;
  replays via `BodySource`; fault-injection tests (G1) green.*
- [ ] **M5-T2 — Legacy `RetryConfig` → `RetryPolicy` mapping** (D5). *DoD: existing
  `retry_test.go` behavior preserved or explicitly migrated; mapping table is normative.*
- [ ] **M5-T3 — Circuit breaker (optional middleware), concurrency-safe.** *DoD: trips/half-open/
  reset state-machine test; race-clean.*
- [ ] **M5-T4 — Client-side rate limiter (optional middleware).** *DoD: token-bucket test;
  race-clean.*

---

## Milestone 6 — Observability

Spec 06. deps: M1-T1, M4.

- [ ] **M6-T1 — Vendor-neutral `Tracer`/`Metrics`/`Logger` interfaces + lifecycle hooks**;
  consume `Kind` (C1). *DoD: no vendor import in core; hooks fire in order (test).*
- [ ] **M6-T2 — Request-ID/trace propagation** (build on `RequestID`). *DoD: propagation test.*
- [ ] **M6-T3 — OTel adapter + Prometheus-friendly metrics adapter in subpackages.** *DoD:
  adapters compile behind their own go files; example test.*

---

## Milestone 7 — Security

Spec 07. deps: M1-T1, M2-T3.

- [ ] **M7-T1 — Opt-in SSRF guard** (`SSRFPolicy`/`WithSSRFGuard`, D2): pre-flight + per-redirect
  (via `RedirectPolicy.Allow`). *DoD: blocks link-local/loopback/RFC1918/metadata unless
  allow-listed; tested.*
- [ ] **M7-T2 — Redaction middleware** wired everywhere (logs/verbose/errors). *DoD: secret
  never appears in any stream (test across modes).*
- [ ] **M7-T3 — Runtime input validation on the live path** (wire the dead `options`
  validators), reconciled with streaming (F2). *DoD: plaintext-auth-over-http policy +
  header/body checks active on `Do`.*
- [ ] **M7-T4 — Proxy auth incl. username-only; TLS `WithTLS`/`WithTLSConfig`** (D1). *DoD:
  username-only proxy authenticates; `WithTLS` flows through `LoadTLSConfig`.*

---

## Milestone 8 — CLI

Spec 13. deps: M1, M4.

- [ ] **M8-T1 — Refactor `main` → `run(args, stdout, stderr) int`.** *DoD: in-process unit
  tests raise `cmd/gocurl` coverage well above current ~45%.*
- [ ] **M8-T2 — Exit codes from `Kind`** (replace string matching) + `--fail`. *DoD: table
  test per Kind; `--fail` tested.*
- [ ] **M8-T3 — Output modes + parity test** (body printed once; `-v` redaction; `-i`/`-o`/
  `-w`). *DoD: parity test (same command → same result in lib and CLI).*

---

## Milestone 9 — Testing & quality hardening

Spec 09. deps: ongoing; finalize here.

- [ ] **M9-T1 — Parser fuzz target** (`go test -fuzz`) + seed corpus from real API docs.
- [ ] **M9-T2 — Fault-injection harness** (G1) reused by M5 tests.
- [ ] **M9-T3 — Leak detection** (goroutines/conns) + soak/load test with pprof (gated).
- [ ] **M9-T4 — Coverage gates per package in CI** (raise proxy >70%, core >80%).
  *DoD: CI enforces the gates.*

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
