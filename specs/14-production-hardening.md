# Production Hardening & Mission-Critical Readiness

> Status: Draft for review · Spec 14 (Milestone 12) · revised after a 13-agent adversarial review

This spec defines how gocurl earns the **"production-grade, mission-critical"** claim it now
makes — by *proving* it: deliberate failure injection, a correctly-targeted execution-path
optimization, statistically sound benchmarks measured **against the competition**, extended soak,
and an operations contract. It is the validation layer on top of M1–M11.

The bar is **best-in-class, honestly demonstrated**. Every claim in README/VISION/docs must trace
to an un-skipped test or benchmark in this milestone, or it is not made — and that rule is enforced
by a doc-lint (§D), not left to discipline.

An adversarial review of the first draft surfaced real correctness bugs in shipped code (no overall
retry budget; h2 `GOAWAY`/`RST_STREAM` not retried; graceful-shutdown truncates streams; unbounded
decompression by default) and a mis-targeted performance thesis (the COW-shared-header design was
unsafe and aimed at a cost that barely exists). Those fixes are now **in scope** below.

## Goals

- **Prove reliability under failure, and fix the correctness bugs the review found.** Build the
  two-tier fault harness, and land the four behavior fixes in §A.3 (overall retry budget, h2 retry
  classification, shutdown stream-accounting, redirect-cap classification) plus the response-side
  memory bounds in §A.4.
- **Optimize the real execution-path cost, correctly.** Replace the rejected COW design with
  "clone-the-small" targeting `req.opts.Clone()`; prove it on the hard concurrent cases.
- **Benchmark with rigor and fairness, against competitors.** Identical transport tuning across all
  arms, p50/p99/p999 + throughput, `benchstat` gates that actually fail on regression, honest results.
- **Prove resource stability over time** with an instrumented soak arm matching the production config.
- **Ship an operations contract** plus a v1.0-readiness checklist.

## Non-goals

- **No public API change.** Validation + internal/test + docs + internal behavior fixes only. The
  `api.txt`/`api_options.txt` guards MUST stay green. Behavior fixes (retry budget, h2 classification,
  shutdown accounting, default caps) change *runtime behavior*, not the exported surface.
- **No regression of the "easy as curl" on-ramp** — tested on both the happy path AND the fault paths
  (DNS/TLS/redirect failures through `Curl*`, not only `Client.Do`).
- **No HTTP/3.** Deferred; `WithTransport` is the seam.
- **No superiority claims.** Target = parity with a well-tuned `net/http` client + proven reliability.
  Competitor results reported fairly, wins and losses.

## Design

### Phase A — Reliability: fault injection + correctness fixes

#### A.1 Two-tier fault harness

Errors injected at the `RoundTripper` layer cannot exercise transport-level behavior, so the matrix
is split by harness tier; every row is tagged with the tier that can actually reach it.

- **Tier 1 — `RoundTripper`-injected** (`faultyRT` via `WithTransport`, **landed** in
  `faultinject_test.go`): error *classification*, retry honoring, circuit breaker, rate limiter,
  per-attempt deadline. Deterministic, hermetic.
- **Tier 2 — real transport** (over `httptest` / a custom `net.Listener`; reuse `leak_test.go`'s
  `ConnState` + `httptest.NewUnstartedServer`; `x/net/http2` is a direct dep): slow-loris /
  `ResponseHeaderTimeout`, premature-EOF / `KindBodyRead`, **real h2 `GOAWAY`/`RST_STREAM`**,
  idle-conn-drop-then-reuse, **DNS failure** (`.invalid` host), **connection-pool exhaustion**
  (`WithMaxConnsPerHost(1)`), **proxy-CONNECT failure** (407 / refused / stalled tunnel),
  **panicking user middleware**, **context-cancel mid-stream**, `Expect: 100-continue` withheld,
  oversized response-header block, and **shutdown-mid-stream**.

#### A.2 Cross-cutting assertions (on every scenario)

- **No goroutine leak** via `goroutinesAtMost(target, deadline)` poll-with-deadline (promote the
  existing `leak_test.go` helper to shared test code) — **not** a fixed-sleep settle + single snapshot.
- **No connection/fd leak** via `ConnState` counting (`leak_test.go` pattern).
- **Secrets stay redacted** on the error path — asserted on every scenario, and **mandatory on every
  new `fmt.Errorf` wrap** added by §A.3 (overall-budget, h2, redirect-cap), or the redaction guarantee
  regresses exactly where new code lands.
- `-race` clean; seeded; hermetic.

#### A.3 Correctness fixes surfaced by the review (behavior-only, no API change)

1. **Overall retry budget.** `WithTimeout` maps to `http.Client.Timeout`, enforced *per attempt*;
   `MaxElapsed` defaults to `0` (unbounded) and `sleepWithContext` waits the full backoff. So
   `WithTimeout(2s)` under a retryable storm permits N×2s of attempts plus unbounded backoff — a
   retry amplifier, the opposite of mission-critical. **Fix:** derive each attempt's context deadline
   from the **remaining overall budget** (layered *under* `Client.Timeout`, so a caller who sets only
   `WithTimeout` still gets a true overall bound); clamp `sleepWithContext` backoff and `Retry-After`
   to `min(value, remaining)`; never produce a negative/huge sleep (clock-skew clamp). **Test:** total
   wall-clock ≤ deadline + slack under a retryable storm.
2. **h2 `GOAWAY`/`RST_STREAM` retry.** `classifyTransportError` returns `KindUnknown` for
   `http2.GoAwayError`/`http2.StreamError` (neither implements `net.Error` nor wraps `*net.OpError`),
   so they are **not** retried — yet h2 is the default TLS path. **Fix:** map them to `KindConnect`
   (gated on the existing idempotency path). **Test:** real h2 server, idempotent GET recovers
   (fails today).
3. **Graceful-shutdown stream accounting.** `Do` does `inflight.Add(1)` / `defer inflight.Done()`,
   which fires on `Do` *return* — but `Do` returns a **live streamed body**, so `Shutdown` can tear
   down a connection mid-stream. **Fix:** move `inflight.Done()` into the returned body's `Close`
   (compose with `cancelOnCloseBody`); the ctx deadline stays the escape hatch. **Test:**
   shutdown-mid-stream blocks until the body closes or ctx fires — no truncation, no leak — **tied to
   the panicking-middleware scenario** so a leaked in-flight cannot wedge `Shutdown` forever.
4. **Redirect-cap classification parity.** `Client.Do` returns a bare `fmt.Errorf("stopped after %d
   redirects")` with no `%w` of `ErrTooManyRedirects`, while the CLI path wraps it — so library
   callers get a `KindUnknown` inconsistent with the CLI's exit-47. **Fix:** `%w`-wrap. **Test:**
   `Client.Do` over-cap classifies as `ErrTooManyRedirects`.

#### A.5 Curl wire-parity (differential) — *partly landed*

"Behavior must match curl" is a correctness property, not an aspiration, so it is proven by
**differential testing against a real `curl` binary**: fire the same command at `curl` and at
gocurl against one server and byte-compare the wire request (method, query, body, headers). The
M9 corpus only checks parse→`options`; this checks the actual bytes. *Landed
(`curl_parity_test.go`): hermetic parity locks + `TestCurlParity_DifferentialVsRealCurl` (self-skips
where curl is absent/sandboxed). It already proved — and fixed — one systematic divergence: gocurl
omitted the `Accept: */*` curl always sends.* **Remaining:** broaden the case matrix
(`--data-urlencode` `+` vs `%20`, `--compressed` `Accept-Encoding` set, multipart `-F`, redirects,
`-I`), and run the differential job in CI where a curl binary is available.

**The one deliberate divergence (security > parity):** gocurl fail-closes basic/bearer auth over
plaintext HTTP (curl leaks credentials in the clear). This is intentional, documented
(`TestCurlParity_BasicAuthFailClosed`), and opt-out via `WithAllowInsecureAuth` /
`GOCURL_ALLOW_INSECURE_AUTH`. Any future divergence must be either fixed or explicitly justified
here as a security/spec decision — never silent.

#### A.4 Response-side memory bounds (untrusted server)

- **Decompression bomb (unbounded today) — *landed*.** `DisableCompression=true` (gocurl inflates
  manually) bypasses net/http's own guard, and the decompressed cap applied **only** when
  `opts.ResponseBodyLimit > 0` (opt-in) — so `CurlString`/`CurlBytes`/`CurlJSON` on a `--compressed`
  response with no limit would buffer GBs into memory (OOM). **Design refinement (implementation
  found a spec gap):** a *transport-level* default cap was rejected because it would break **all**
  streaming (a raw `Curl` + `io.Copy` proxy, not just `CurlDownload`) and need opt-out plumbing
  everywhere. Instead, bound exactly what gocurl buffers on the caller's behalf — the convenience
  helpers — via a shared `readBounded` / a `LimitReader`-wrapped decoder with a `defaultBufferedResponseLimit`
  (64 MiB, a `var` for tests). STREAMING (`Curl`, `CurlDownload`) is deliberately **unbounded**: the
  caller controls its own memory, so legitimate large downloads keep working. To buffer more, stream.
  *Tests: `TestFault_BufferingHelpersBoundedAgainstBomb` (helpers cap; streaming does not).*
- **Oversized response headers — *landed*.** `MaxResponseHeaderBytes` was unset (10 MB default);
  tightened to 1 MiB on every gocurl-built transport (`newTransport` + `buildTransport`).
- One coherent **"untrusted-server memory bounds"** story (ops doc + the above), framed honestly:
  *tighten* existing defaults; the buffered-helper bound closes the one genuinely unbounded path.

### Phase B — Execution performance: clone-the-small + competitive proof

**Scope (user, 2026-06-21):** parser-perf benchmarking is **deferred** (one-time cost, amortized by
parse-once). The focus is the **execution** path.

**The real cost (corrected).** The per-`Do` overhead vs `net/http` is the unconditional
`req.opts.Clone()` at `client.go:128` — a field-by-field deep copy of `Headers` + `Form` +
`QueryParams` + `Cookies` + auth + `RetryConfig` (`net/http.Header.Clone` shows up only as its
*child*). The first draft's "eliminate `Header.Clone`-per-`Do`" thesis was mis-targeted and is removed.

**The COW-shared-header design is REJECTED.** It cannot be enforced: the cookie jar calls
`req.AddCookie`, observability calls `req.Header.Set` for the request-ID, and retry rewrites
`req.Body`/`ContentLength` in place — all from packages with zero COW awareness.

**Clone-the-small (the edge, done safely).** Because gocurl holds the full parsed recipe at `Prepare`,
compile an **immutable small header template** from the `Request`'s **own** opts at `Prepare`. Per-`Do`:
`http.NewRequestWithContext` + `template.Clone()` onto an **owned** map, then apply the dynamic bits
(Client-default headers/UA, request-ID, cookies/jar) to that owned map. **Never** memoize a
Client-merged plan on the shared `Request` (Client defaults must not bleed across two `Client`s using
one prepared `Request`). Re-target the optimization at **avoiding the unconditional `opts.Clone()`**
per `Do`. This is correct with **zero** changes to `process.go`/`observability.go`/the jar/redirects.

**Validation matrix (the breaking cases — replaces the single trivial GET), all `-race`:** jar +
concurrent (distinct cookies, template untouched); `WithRequestIDFunc` + concurrent (distinct IDs);
POST with a buffered (non-`GetBody`) body retried (correct `Content-Length` per attempt);
two `Client`s + one `Request` (no default bleed); cross-host redirect carrying `Authorization` +
`Cookie` (parity with the pre-optimization path); non-rewindable `ReaderBody` executed twice. Each
asserts **wire correctness AND** that the per-`Do` request's header/body are owned. Plus a
`Request.Options()`-equals-the-wire-request invariant test so the plan cannot drift the public read
path past the signature-only api guard.

**Competitive proof.**

- **Module:** use Spec 10's already-ratified `//go:build benchvendor` mechanism (or a real
  `benchcmp/go.mod` with a `replace`) — **not** the nonexistent "bench/scripts module" the first draft
  cited; reconcile with Spec 10's deferral of these arms; add a CI diff-guard asserting `go mod tidy`
  introduces **no** new root `require`/`go.sum` entry.
- **Fairness (mandatory before any concurrent/competitor number ships):** identical transport tuning
  across all arms — idle pool, `MaxConnsPerHost`, `DisableCompression` — enforced by a guard test that
  fails if they differ. (Today the `net/http` arm runs `MaxIdleConnsPerHost:100` vs gocurl's `10`.)
- **Arms** over one shared server: `net/http` (parity bar), gocurl prepared, gocurl per-call-parse,
  and ≥1 popular client (`go-resty/resty` / `imroc/req`).
- **Metrics:** p50/p99/**p999** (raise N ≥ 1e5 + `runtime.GC()` + median-of-5, or drop p999 rather than
  publish a noisy number) + throughput under bounded concurrency at varying pool sizes.
- **Gates:** **ratchet** the `Do` alloc budget from 100 to baseline + small headroom *after* the plan
  lands (today's 100-vs-76 = 24-alloc slack would let the win silently regress); commit
  `bench/baseline.txt` at `-count ≥ 10`; a scheduled `benchstat` job that **fails** on `allocs/op` or
  `B/op` regression (`ns/op` advisory, since wall-clock is noisy); and a clone-path alloc test proving
  the static header is **not** deep-cloned per `Do`.
- Honest results table in `docs/benchmarking.md`, **including where gocurl loses**.

### Phase C — Soak + resource stability

- **Extend `TestClient_Soak`** (`soak_test.go`) by name — keep `GOCURL_PROFILE`, select duration with a
  single env var `GOCURL_SOAK=<duration>` consistent with the shipped convention; the repo has **no**
  build tags and M12 will not introduce its first.
- **Two arms:** uninstrumented AND **instrumented** (a recording `Tracer`+`Metrics`+`Logger`+`Hooks`,
  the config operators actually run). Both assert no goroutine/heap/fd growth trend (linear-fit slope
  ≈ 0); publish the alloc/op delta between arms so the honesty story reflects the production config.
- Pool-churn / backpressure at `MaxConnsPerHost` (requests block then drain; no deadlock), idle-eviction.
- Scheduled (cron) CI; never the PR path.

### Phase D — Operability + v1 contract

- `docs/operations.md`: timeout taxonomy + pool/retry-budget tuning, capacity planning, a failure-mode
  playbook (each `Kind` operationally, incl. the unlimited-`MaxConnsPerHost`/EMFILE default), an
  observability runbook, the **untrusted-server memory-bounds** section, and an **"easy start →
  production checklist"**.
- Threat model (extend `SECURITY.md`).
- **Honesty doc-lint:** a dependency-free test (mirror `api_guard_test.go`) that fails if a claim keyword
  ("production-grade", "mission-critical", "parity", …) in README/VISION/docs lacks a citation token
  naming an un-skipped test/benchmark. Gate the README's "production-grade" wording on **M12-T1 being
  `[x]`** (today `README:12` already claims it while M12-T1 is `[~]` — the claim precedes its evidence).
- **v1.0-readiness checklist:** surface locked · §A matrix green · §B plan landed + benchmarks published
  · soak green · ops docs done → a tag is then a near-trivial step (timing is the user's call, Spec 11).

## Behavior & edge cases

- **Honesty guardrail (now enforced).** A performance/reliability sentence must cite a named, un-skipped
  test/benchmark; the §D doc-lint enforces it. Competitor results reported as-measured, wins and losses.
- **"Easy as curl" invariant, on failure too.** A test asserts the zero-config `Curl*` one-liner still
  works on success AND surfaces a correct classified error/exit code on the fault paths (DNS/TLS/redirect),
  since newcomers hit failures through the on-ramp.
- **Determinism over flakiness.** Fault scenarios seeded/hermetic; timing assertions use poll-with-deadline,
  not fixed sleeps. Latency gates advisory; alloc/byte gates hard.
- **No silent caps.** Any bounded coverage (sampling, top-N, fixed duration) is logged.

## Acceptance criteria / Definition of Done

- [ ] **A.1/A.2** — two-tier harness; every matrix row tagged + reachable by its tier; each scenario
      asserts no goroutine leak (`goroutinesAtMost`), no conn/fd leak (`ConnState`), and secret redaction;
      `-race` clean. Fast Tier-1 subset in CI `-short`; full matrix in a non-short job.
- [ ] **A.3** — overall retry budget honored (wall-clock ≤ deadline+slack test); h2 `GOAWAY`/`RST_STREAM`
      retried for idempotent GET (real-h2 test); shutdown-mid-stream does not truncate/leak (tied to
      panicking-middleware); `Client.Do` redirect-cap classifies as `ErrTooManyRedirects`.
- [ ] **A.4** — default decompressed-bytes cap + opt-out sentinel; `MaxResponseHeaderBytes` tightened;
      §A rows proving each bound.
- [ ] **B (clone-the-small)** — `opts.Clone()` per-`Do` eliminated via a Prepare-time template; the full
      breaking-case validation matrix green under `-race`; a test proves the static header is not
      deep-cloned per `Do`; before/after `benchstat` shows the per-`Do` alloc/byte gap vs `net/http`
      shrink; no surface change; §A matrix + suite stay green.
- [ ] **B (competitive)** — benchvendor (or benchcmp module) arms with **identical** transport tuning
      across arms (guard test); p50/p99/p999 + throughput; `benchstat` regression gate (allocs/B hard,
      ns advisory); ratcheted `Do` alloc budget; honest results table incl. losses; `go mod tidy`
      diff-guard.
- [ ] **C** — extended `TestClient_Soak` with uninstrumented + instrumented arms, no growth trend, pprof
      artifacts, alloc/op delta published; pool-churn/backpressure test.
- [ ] **D** — `docs/operations.md`, threat model, honesty doc-lint (gating the "production-grade" wording),
      and the v1.0-readiness checklist.
- [ ] **Surface unchanged** — `api.txt`/`api_options.txt` guards green; the "easy as curl" one-liner
      (happy + fault path) test passes.
- [ ] All new tests hermetic and `-race` clean; coverage stays ≥ floor; no claim without a backing
      benchmark/test (doc-lint enforced).

## Dependencies

- **Spec 03** (transport/timeouts/h2/`MaxConnsPerHost`/redirect policy), **Spec 04** (retry/breaker/
  limiter, the overall-budget fix), **Spec 06** (observability — instrumented soak, request-ID write),
  **Spec 07** (security/redaction, decompression/header bounds), **Spec 08** (`Kind` — h2 + redirect
  classification) — the behavior under test/fix.
- **Spec 09** (testing layering, soak, the fault-injection decision G1, `leak_test.go` helpers).
- **Spec 10** (benchmark methodology + the ratified `benchvendor` mechanism this milestone reuses).
- **Spec 11** (API stability — the surface guard M12 must not disturb; feeds the v1.0-readiness checklist).
