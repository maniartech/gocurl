# Benchmarking & Performance Methodology

> Status: Draft for review · Spec 10

## Goals

- Establish an **honest, reproducible** benchmark suite that measures gocurl against a
  well-tuned `net/http` client (and, optionally, `resty`/`req`) over a **shared, in-process
  `httptest` server**, so numbers reflect gocurl's own overhead and not network noise.
- Define exactly **what we measure** (request construction, full round-trip, concurrent
  throughput, allocs/op) and **what we will and will not claim** (parity + ergonomics;
  never "faster than net/http", never "zero-allocation").
- Make the **"parse once, execute many"** architectural truth visible in the benchmarks:
  separate the one-time `Prepare`/parse cost from the per-request `Do` hot path, and prove
  that a prepared `*Request` executed over a pooled `Client` adds only thin overhead above
  raw `net/http`.
- Add **benchmark regression detection in CI** so a performance claim, once made, is
  guarded by a green, un-skipped benchmark and an allocation budget.
- Document a **profiling workflow** (pprof CPU/heap, latency p50/p99) that any contributor
  can run to investigate a regression.
- Replace the stale, dishonest performance narrative ("Week 2 target: 0 allocs/op", "10k
  req/s target") — since removed with the legacy planning docs — with this methodology.

## Non-goals

- **No raw-throughput superiority claim.** We do not try to beat `net/http`; we cannot, and
  saying so would violate the honesty constraints. The target is parity.
- **No microbenchmark theater.** We will not tune toward `0 allocs/op` on the parse path by
  introducing buffer pools that complicate the code; parsing is an authoring-time cost paid
  once, not a per-request hot path.
- **No real-network or third-party-host benchmarks** in the suite (no httpbin.org, no
  Stripe). Those are non-hermetic and unreproducible. Round-trip benchmarks use only
  `net/http/httptest`.
- **No HTTP/2 / HTTP/3 throughput shootouts** in v1. h2 is enabled by the same transport
  config; h3 (quic-go) is a future add-on (per project scope) and out of scope here.
- **No published vendor leaderboard.** `resty`/`req` comparisons are diagnostic context for
  us, run behind a build tag, not marketing material we publish as rankings.

## Design

### What we benchmark, and where

Four benchmark families, each mapped to a layer of the architecture and existing code:

| Family | Measures | Built on (real code) |
|---|---|---|
| **Construction** | One-time parse of a curl command into a prepared request | `ArgsToOptions` (`convert.go:24`), `convertTokensToRequestOptions`, `tokenizer`, and the planned `client.Prepare` |
| **Variable expansion** | `${VAR}` substitution cost | `ExpandVariables` (`variables.go:10`), `expandEnvInTokens` (`api.go:557`) |
| **Round-trip** | Full `Do` over `httptest`: build `*http.Request`, transport, read body | `doRequest` (`process.go:57`), `CreateRequest` (`process.go:263`), `getRoundTripper` (`clientpool.go:30`) |
| **Concurrent throughput** | Sustained parallel `Do` with connection reuse | `RunParallel` + the cached transport in `clientpool.go` |

All benchmarks live in package-external test files (`package gocurl_test`), matching the
existing `benchmark_test.go` and the blackbox `tests/` package, so they exercise only the
public surface.

### Comparison harness: gocurl vs net/http over one server

Every round-trip and throughput benchmark runs the **same `httptest` server** through three
clients, sharing one server per benchmark function so server cost is constant across arms:

```go
// bench_roundtrip_test.go  (package gocurl_test)

// benchServer returns a shared, in-process server with a fixed small JSON body.
func benchServer(b *testing.B) *httptest.Server { /* httptest.NewServer, b.Cleanup(close) */ }

// Arm 1: baseline — a well-tuned, reused net/http client (the bar we target parity with).
func BenchmarkRoundTrip_NetHTTP(b *testing.B) { /* one *http.Client, pooled transport, reused */ }

// Arm 2: gocurl prepared+pooled — the "parse once, execute many" hot path.
func BenchmarkRoundTrip_Gocurl_Prepared(b *testing.B) {
    c, _ := gocurl.New()                 // pooled Client (Spec: Client/New)
    req, _ := c.Prepare("curl " + srv.URL) // parse ONCE, outside the timed loop
    b.ReportAllocs(); b.ResetTimer()
    for i := 0; i < b.N; i++ {
        resp, err := c.Do(ctx, req)      // execute MANY; the measured hot path
        // io.Copy(io.Discard, resp.Body); resp.Body.Close()
    }
}

// Arm 3: gocurl naive one-shot — parses every iteration (the cost we warn against).
func BenchmarkRoundTrip_Gocurl_PerCallParse(b *testing.B) {
    for i := 0; i < b.N; i++ { gocurl.CurlString(ctx, "curl "+srv.URL) }
}
```

The gap between Arm 2 and Arm 1 is gocurl's **execution overhead** (the honest number we
care about). The gap between Arm 3 and Arm 2 is the **per-call parse tax** — the empirical
justification for the prepared-`Request` API. Arm 3 is expected to be slower and that is the
point: the benchmark *demonstrates* why "parse once" matters rather than hiding it.

Third-party arms (`resty`, `req`) live behind a build tag so they are not a default
dependency:

```go
//go:build benchvendor
// bench_vendor_test.go — opt-in: go test -tags benchvendor -bench=RoundTrip_Resty
```

### Allocation guards (the regression teeth)

Two complementary mechanisms:

1. `b.ReportAllocs()` on every benchmark (already present in `BenchmarkRequestConstruction`,
   `benchmark_test.go:18`) so `allocs/op` and `B/op` appear in output and in CI diffs.
2. `testing.AllocsPerRun` assertions in **regular `Test*` functions** (so they run in the
   normal `go test` job, including `-short`) that fail when a path's allocations exceed a
   documented budget:

```go
// alloc_budget_test.go
func TestAllocBudget_Prepare(t *testing.T) {
    c, _ := gocurl.New()
    avg := testing.AllocsPerRun(100, func() { _, _ = c.Prepare("curl https://example.com") })
    const budget = 40 // documented ceiling, NOT a "zero-alloc" claim
    if avg > budget {
        t.Fatalf("Prepare allocs/op = %.0f, budget %d (update budget intentionally if justified)", avg, budget)
    }
}
```

Budgets are **ceilings chosen from the current measured baseline plus headroom**, not
aspirational zeros. Lowering a budget is a deliberate, reviewed change; raising one requires
a one-line justification in the commit.

### Profiling workflow

Documented in `docs/benchmarking.md` and reproducible by any contributor:

```bash
# CPU + memory profile of the prepared hot path
go test -run=^$ -bench=BenchmarkRoundTrip_Gocurl_Prepared -benchmem \
        -cpuprofile=cpu.out -memprofile=mem.out -benchtime=3s
go tool pprof -top cpu.out          # where time goes
go tool pprof -alloc_space mem.out  # where allocations come from

# Latency distribution (p50/p99) — a small custom harness, not go test -bench,
# since testing.B reports mean ns/op, not percentiles.
go test -run=TestLatencyDistribution -v   # prints p50/p90/p99/p999 over N closed-loop requests
```

A `TestLatencyDistribution` helper issues a fixed number of sequential `Do` calls against
`benchServer`, records per-request wall time, sorts, and logs percentiles. It is `-short`
gated (skipped in the fast CI test job) and informational, not pass/fail.

## Behavior & edge cases

- **Body must be drained.** Every round-trip arm must `io.Copy(io.Discard, resp.Body)` then
  `resp.Body.Close()`; skipping this breaks connection reuse and silently inflates the
  baseline gap. The harness provides a `drain(resp)` helper to make this uniform.
- **Server cost is shared, not measured per-arm.** One `httptest.Server` per benchmark
  function, created before `b.ResetTimer()`, so server allocation never lands in the loop.
- **`-short` mode.** Round-trip/throughput/latency benchmarks and the vendor arms are
  unaffected by `-short` (benchmarks ignore it), but the `TestLatencyDistribution` and any
  long `TestAllocBudget_*` over real I/O must call `t.Skip` under `testing.Short()` so the
  existing CI `go test -short -race` job (`ci.yml:38`) stays fast and hermetic.
- **`-race` and benchmarks don't mix for numbers.** The race detector perturbs timing and
  allocations; CI runs benchmarks **without** `-race` in a dedicated job. Race correctness of
  concurrent access is already covered by `race_test.go` / `race_concurrent_test.go`.
- **Existing skipped benchmark is replaced.** `BenchmarkRequestAPI` (`benchmark_test.go:54`)
  is `b.Skip("requires test server")`. That skip is removed: it becomes a real round-trip
  benchmark against `benchServer`. A skipped benchmark can never back a performance claim.
- **Variance & noise.** Reported numbers use `-count=10` and are compared with `benchstat`;
  a single run is never authoritative. Machine, Go version, and OS are recorded with results
  (the old doc's "AMD Ryzen 7 5700G, Windows, Go 1.21+" footer is the right instinct, kept).
- **Transport caching is part of the measurement.** Because `getRoundTripper` caches
  transports by `transportKey` (`clientpool.go:89`), the first iteration may pay transport
  construction; `b.ResetTimer()` after a warm-up `Do` ensures the steady-state hot path is
  what's timed.
- **No claim without a benchmark.** Any sentence in README/VISION/docs that states a
  performance property must cite a named, un-skipped benchmark in this suite. If the
  benchmark is deleted or skipped, the claim must be deleted too.

## Acceptance criteria / Definition of Done

- [x] `bench_roundtrip_test.go` exists with `BenchmarkRoundTrip_NetHTTP`,
      `BenchmarkRoundTrip_Gocurl_Prepared`, and `BenchmarkRoundTrip_Gocurl_PerCallParse`, all
      against a shared `httptest` server, all draining the body, all calling `b.ReportAllocs()`.
- [x] `BenchmarkRequestAPI`'s `b.Skip` is removed (the function is deleted, superseded by the
      real `BenchmarkRoundTrip_*` arms); no benchmark used to support a claim is skipped.
- [x] Construction and expansion benchmarks (`BenchmarkRequestConstruction`,
      `BenchmarkVariableExpansion`, `BenchmarkConcurrentRequests`) are retained and rephrased
      as **"one-time" cost**, explicitly separated from the round-trip hot path.
- [x] `BenchmarkRoundTrip_Concurrent_*` arms use `b.RunParallel` and demonstrate connection
      reuse via the cached transport.
- [x] `alloc_budget_test.go` uses `testing.AllocsPerRun` with documented ceiling budgets for
      `Prepare` (45), `ExpandVariables` (6), and `Do` over httptest (100); budgets are baselined
      (2/30/76), not zero.
- [~] Optional `resty`/`req` arms — **deferred**: omitted to keep the module's require graph free
      of vendor deps. Documented in `docs/benchmarking.md` as opt-in/future; not required for v1.
- [x] CI gains a non-`-race` `benchmarks` job that runs `go test -run=^$ -bench=. -benchmem
      -benchtime=1x` (smoke: every benchmark executes once without error); the `TestAllocBudget_*`
      guards run in the normal test job.
- [x] `docs/benchmarking.md` documents the methodology, the pprof workflow, the p50/p99
      latency harness, and the "parity, not superiority" claim policy.
- [x] The legacy planning docs that carried the zero-allocation / 10k-req/s / faster-than-net-http
      narrative (`PERFORMANCE_TESTING.md`, `design.md`, `objective*.md`, `README_NEW.md`, `plan/`,
      `book/`) are **removed**; `docs/benchmarking.md` is the authoritative methodology, and no
      active doc retains a "faster than net/http" or "zero-allocation" claim.
- [x] A comparison of `Gocurl_Prepared` vs `NetHTTP` is recorded in `docs/benchmarking.md` with
      machine/Go/OS provenance and the overhead delta (the honest parity number); `benchstat`
      with `-count=10` is documented as the reproducible method.

## Dependencies

- Spec **01 (Client / New / Option)** — the pooled, reusable `Client` and functional options
  the prepared-vs-pooled arm benchmarks.
- Spec **02 (Request / Prepare / Do)** — the immutable prepared `*Request` and
  `client.Prepare`/`client.Do` that make "parse once, execute many" measurable; until those
  land, Arm 2 is approximated with package-level `Curl*` over a cached transport.
- Spec **resilience/RetryPolicy** (retry middleware) — round-trip benchmarks run with retries
  disabled by default; a separate arm may measure middleware-chain overhead once defined.
- Existing code this builds on directly: `convert.go` (`ArgsToOptions`), `variables.go`
  (`ExpandVariables`), `process.go` (`doRequest`/`CreateRequest`), `clientpool.go`
  (`getRoundTripper`/`transportKey`), `benchmark_test.go`, `parity_test.go`, `tests/`.

## Open questions / decisions to confirm in review

- **Vendor arms in CI?** Proposed: keep `resty`/`req` behind `-tags benchvendor` and run them
  only locally/on-demand, never in the default CI graph. Confirm we don't want a periodic
  (manual-dispatch) CI job that runs them for our own diagnostics.
- **Regression gating strength.** `testing.AllocsPerRun` budgets are deterministic and CI-safe;
  time-based regression gating (`benchstat` with a delta threshold) is noisy on shared CI
  runners. Proposed: gate on **allocs only** in CI, treat ns/op as informational. Confirm.
- **Where docs live.** Proposed `docs/benchmarking.md`. Confirm vs `book2/` chapter vs a
  top-level `BENCHMARKS.md`.
- **Go version matrix for benchmarks.** CI tests on 1.22.x and 1.23.x (`ci.yml:13`). Proposed:
  run the benchmark smoke job on a single pinned version (1.23.x) to keep numbers comparable;
  confirm we don't need cross-version benchmark provenance.
- **Latency harness scope.** Proposed: closed-loop sequential p50/p99 only (no open-loop /
  Little's-law load generator). Confirm an in-repo load test is out of scope for v1.
