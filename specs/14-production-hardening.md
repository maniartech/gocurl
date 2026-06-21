# Production Hardening & Mission-Critical Readiness

> Status: Draft for review · Spec 14 (Milestone 12)

This spec defines how gocurl earns the **"production-grade, mission-critical"** claim it now
makes — not by asserting it, but by *proving* it: deliberate failure injection, statistically
sound benchmarks measured **against the competition**, extended soak, and an operations
contract. It is the validation layer on top of the features shipped in M1–M11.

The bar is **best-in-class, honestly demonstrated**. Every claim in the README/VISION/docs must
trace to an un-skipped test or benchmark in this milestone, or it is not made.

## Goals

- **Prove reliability under failure.** Build the fault-injection harness Spec 09 (G1) called for
  and never delivered, and validate — under real failures — that retries classify correctly, the
  circuit breaker trips/recovers, the rate limiter holds, per-attempt deadlines fire, and nothing
  leaks (goroutines, connections, fds).
- **Benchmark with rigor and against competitors.** Add latency-distribution (p50/p99/p999) and
  throughput-under-load measurement, a documented `benchstat` methodology, competitor arms
  (`net/http` baseline + popular Go HTTP clients), and a regression gate. Publish honest results.
- **Prove resource stability over time.** Extend soak to a long-form, pprof-instrumented run that
  asserts no goroutine/heap/fd growth trend.
- **Ship an operations contract.** Tuning, capacity, and failure-mode docs plus a threat model and
  a concrete v1.0-readiness checklist.

## Non-goals

- **No public API changes.** This milestone is *validation + docs + internal/test code*. The
  `api.txt` / `api_options.txt` surface guards MUST stay green; nothing here adds, removes, or
  re-signatures an exported symbol. If a gap genuinely needs new public surface, it is escalated
  as a separate, explicitly-approved change — not folded into M12.
- **No regression of the "easy as curl" on-ramp.** The getting-started path (`gocurl.Curl*`) stays
  a one-liner. All hardening is either internal, opt-in via existing options, or operational docs.
  A newcomer still pastes a curl command and it runs — that invariant is tested, not assumed.
- **No HTTP/3.** Deferred until the ecosystem is production-ready; `WithTransport` is the seam.
- **No superiority claims.** The target is parity with a well-tuned `net/http` client plus proven
  reliability. Competitor benchmarks are reported fairly, including cases where gocurl is slower.

## Design

### Phase A — Fault-injection harness *(the biggest credibility gap)*

A hermetic, seeded, test-only harness (`faultinject_test.go` + helpers; no production code, no new
public surface) that injects failures at the connection, TLS, HTTP, and h2 layers via a custom
`net.Listener`/`net.Conn`, a misbehaving `http.Handler`, and a controllable dialer:

| Failure mode | Injected how | Asserted behavior |
|---|---|---|
| Connection refused / dial timeout | dialer returns error / black-hole addr | `KindConnect`/`KindTimeout`, retried (idempotent), `--connect-timeout` fires |
| Reset mid-headers / mid-body | conn `Close` during write | classified, retried only if idempotent + not yet streamed; no leak |
| Slow-loris (header / body stall) | handler sleeps past `ResponseHeaderTimeout` | per-phase timeout fires; attempt aborts; ctx respected |
| Partial body + premature EOF | short write then close | `KindBodyRead`; not silently truncated |
| TLS handshake / cert failure | bad cert / wrong host | `KindTLS`, **not** retried |
| h2 `GOAWAY` / `RST_STREAM` | h2 test server | treated as retryable connection-level error (Spec 03 F4) |
| 5xx / 429 storm (+/- `Retry-After`) | handler status sequence | breaker trips after threshold, half-opens, recovers; `Retry-After` honored; retry budget bounds attempts |
| Idle-conn drop then reuse | server closes idle conn | reused-conn error on idempotent request is retried (Spec 03) |
| Body over cap under failure | oversized + mid-stream error | `ResponseBodyLimit` still bounds memory |

Cross-cutting assertions on **every** scenario: no goroutine leak (snapshot before/after), no
connection/fd leak, secrets stay redacted on the error path, and the run is `-race` clean. A
representative, fast subset is gated in CI (`-short`); the full matrix runs in a non-`-short` job.

### Phase B — Execution performance: the compiled plan + competitive proof

**Scope (user, 2026-06-21):** parser *performance* benchmarking is **deferred** — parsing is a
one-time cost amortized by "parse once, execute many", so it is not the production hot path. The
focus is the **execution** path: it must be top-class in performance, security, and robustness.

**The edge — compile the recipe into a reusable execution plan.** A `net/http` developer hand-builds
the request once and reuses it; gocurl currently *re-derives* it on every `Do` (deep-clone
`RequestOptions`, then `applyHeaders`/`applyAuth`/`applyCookies`/`applyCompression`/`applyRequestID`,
then `Header.Clone`). The alloc profile attributes the ~2× memory / ~13 extra allocs/op vs
`net/http` to exactly that re-derivation. Because we hold the **full parsed recipe** at `Prepare`
time, we can compile the plan once and make each `Do` do only the irreducible work:

- **Compiled once (at `Prepare`/first use, memoized, immutable):** method, URL, the static header
  set, auth header, body source, compression/TLS intent.
- **Per-`Do` (irreducible):** fresh `ctx`; per-attempt body rewind (`GetBody`); dynamic headers
  (e.g. request-ID) applied via **copy-on-write** so the static header map is shared read-only
  across executions (eliminating the `Header.Clone`-per-`Do` cost); cookie-jar read.

Hard requirements: **no public surface change** (the plan is internal to `Request`/`Client`);
**concurrency-safe** (concurrent `Do` on one prepared `Request` must never share mutable state —
verified by `-race` + a concurrent benchmark); **behavior-identical** (the fault matrix in §A and
the full suite stay green); and **validated** (a test proves `net/http`'s `RoundTrip` does not
mutate the shared request header, the assumption COW relies on). Each step is benchmarked
before/after with `benchstat`; the goal is to close the gap toward `net/http` parity, never a
"faster than net/http" claim.

**Competitive proof.**

- **Competitor arms** live in the **benchmark module's own `go.mod`** (the existing bench/scripts
  module pattern), so the *library's* require graph stays clean. Arms: `net/http` (the parity
  bar), gocurl prepared (`Prepare`+`Do`), gocurl per-call-parse, and 1–2 popular Go clients
  (e.g. `go-resty/resty`, `imroc/req`) over **one shared httptest server**.
- **Latency distribution + throughput under load**: a bounded-concurrency load generator records
  p50/p99/**p999** and req/s at varying concurrency and pool sizes — extending the existing
  `TestLatencyDistribution` from p50/p99 to p999 and to sustained concurrent load.
- **Methodology**: `-count≥10` + `benchstat`, machine/Go/OS provenance, documented in
  `docs/benchmarking.md` with an honest results table (parity numbers + percentiles + the
  competitor comparison, *including where gocurl loses*).
- **Regression gate**: allocation budgets stay a **hard** gate (deterministic). Latency/throughput
  get a tracked baseline file + a `benchstat`-based comparison surfaced as **advisory** in PRs
  (wall-clock is noisy on shared CI) and a tighter gate in the dedicated benchmark job.

### Phase C — Extended soak + resource stability

- A long-form soak (behind `GOCURL_SOAK=1` / a build tag; minutes-to-hours) drives sustained mixed
  traffic through a reused `Client` and asserts **no upward trend** in goroutines, heap-in-use, or
  fds (linear-fit slope ≈ 0 within tolerance), emitting cpu/heap pprof artifacts. Scheduled
  (cron) CI, **never** the PR path.
- Pool-churn / backpressure validation at `MaxConnsPerHost` (requests block, then drain; no
  deadlock), and idle-eviction correctness.

### Phase D — Operability + v1 contract

- `docs/operations.md`: the timeout taxonomy + pool/retry-budget tuning, capacity planning, a
  failure-mode playbook (what each `Kind` means operationally), an observability runbook, and an
  explicit **"easy start → production checklist"**.
- Threat model: extend `SECURITY.md` (or `docs/threat-model.md`) with the trust boundaries, the
  SSRF/redaction/pinning guarantees, and the unknown-flag policy.
- **v1.0-readiness checklist**: surface locked (guards green) · fault matrix green · soak green ·
  benchmarks published · ops docs done → then a tag is a near-trivial step (timing is the user's
  call, per Spec 11).

## Behavior & edge cases

- **Honesty guardrail (normative).** A performance or reliability sentence in README/VISION/docs
  must cite a named, un-skipped test/benchmark here. Delete the benchmark → delete the claim.
  Competitor results are reported as-measured, both wins and losses.
- **"Easy as curl" invariant.** A test asserts the canonical one-liner
  (`gocurl.CurlString(ctx, "curl <url>")`) still works with zero configuration after all hardening.
  Robustness is opt-in or internal; it never complicates the on-ramp.
- **Determinism over flakiness.** Fault scenarios are seeded and hermetic; timing-sensitive
  assertions use generous bounds. Anything inherently noisy (latency gates) is advisory, not a
  hard PR gate, so CI never flakes on wall-clock.
- **No silent caps.** If a benchmark or soak bounds coverage (sampling, top-N, fixed duration), it
  logs what was bounded so results are not mistaken for exhaustive.

## Acceptance criteria / Definition of Done

- [ ] **A** — Fault-injection harness exists; every row of the failure matrix has a `-race`-clean,
      hermetic test asserting the documented classification/breaker/limiter/deadline behavior and
      zero goroutine/conn/fd leak. A fast subset runs in CI `-short`; the full matrix in a non-short job.
- [ ] **B (compiled plan)** — the recipe is compiled into a reusable execution plan at `Prepare`;
      each `Do` does only the irreducible work (fresh ctx, body rewind, dynamic headers via
      copy-on-write over a shared static header). `-race` + a concurrent benchmark prove no shared
      mutable state; a test proves `net/http` does not mutate the shared request header; before/after
      `benchstat` shows the per-`Do` alloc/byte gap vs `net/http` shrink. No surface change; §A matrix
      + full suite stay green.
- [ ] **B (competitive proof)** — competitor benchmark arms (`net/http` + ≥1 popular client) in the
      bench module; p50/p99/p999 + throughput-under-load harness; `benchstat` methodology + provenance
      documented; honest results table in `docs/benchmarking.md` (incl. where gocurl loses);
      alloc-budget hard gate + latency baseline/advisory gate. Parser-perf benchmarking deferred.
- [ ] **C** — Long-form soak (gated, scheduled) asserts no goroutine/heap/fd growth trend with pprof
      artifacts; pool-churn/backpressure test green.
- [ ] **D** — `docs/operations.md` (tuning + capacity + failure playbook + "easy→prod checklist"),
      threat model, and the v1.0-readiness checklist exist.
- [ ] **Surface unchanged** — `api.txt` and `api_options.txt` guards green throughout (no public
      API change); the "easy as curl" one-liner test passes.
- [ ] All new tests hermetic and `-race` clean; coverage gate stays ≥ floor; no claim without a
      backing benchmark/test.

## Dependencies

- **Spec 03** (transport/timeouts/h2), **Spec 04** (retry/breaker/limiter), **Spec 06**
  (observability), **Spec 07** (security/redaction), **Spec 08** (`Kind` taxonomy) — the behavior
  under test.
- **Spec 09** (testing layering) — extends its fault-injection decision (G1) and soak.
- **Spec 10** (benchmark methodology) — extends it with competition + percentiles + the gate.
- **Spec 11** (API stability) — the surface guard this milestone must not disturb; feeds the
  v1.0-readiness checklist.
