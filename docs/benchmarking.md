# Benchmarking & Performance Methodology

This document defines how gocurl is benchmarked, what we measure, and — critically —
**what we will and will not claim**. It implements [Spec 10](../specs/10-benchmarking.md).

> **Motto: persuasion by example, not by marketing.** Every number here is reproducible and
> every claim cites an un-skipped benchmark. Where gocurl loses, we publish the loss.

## Claim policy: parity + ergonomics, never superiority

gocurl is a thin, curl-ergonomic layer over `net/http`. It **cannot** be faster than
the `net/http` client it wraps, and we never say it is.

- We claim **parity**: a prepared request executed over a pooled `Client` adds only a
  thin, constant overhead above a bare `net/http` round-trip.
- We claim **ergonomics**: "paste a curl command from the docs and it runs."
- We do **not** claim "faster than net/http" or "zero-allocation". Any sentence in the
  README/VISION/docs that states a performance property must cite a named, un-skipped
  benchmark below. If the benchmark is deleted or skipped, the claim must be deleted too.

## What we measure

Four benchmark families, each mapped to a layer of the architecture:

| Family | Measures | Where |
|---|---|---|
| **Construction** (one-time) | Parse a curl command into `*RequestOptions` | `BenchmarkRequestConstruction`, `BenchmarkVariableExpansion` (benchmark_test.go) |
| **Round-trip** | Full `Do` over httptest: build request, transport, read body | `BenchmarkRoundTrip_*` (bench_roundtrip_test.go) |
| **Concurrent throughput** | Sustained parallel `Do` with connection reuse | `BenchmarkRoundTrip_Concurrent_*` |
| **Allocation budgets** | Hard ceilings on allocs/op for hot paths | `TestAllocBudget_*` (alloc_budget_test.go) |

Construction/expansion are the **one-time, authoring-time** cost paid once by `Prepare`,
deliberately *not* tuned toward 0 allocs/op — parsing runs once, not per request.

## The three round-trip arms

All three run against the **same** in-process `httptest` server, drain the body, and
report allocs:

- `BenchmarkRoundTrip_NetHTTP` — a well-tuned, reused `net/http` client. **The parity bar.**
- `BenchmarkRoundTrip_Gocurl_Prepared` — `c.Prepare()` once, `c.Do()` many. The hot path.
- `BenchmarkRoundTrip_Gocurl_PerCallParse` — re-parses every call. The cost the prepared
  API avoids.

The gap **Prepared − NetHTTP** is gocurl's honest execution overhead. The gap
**PerCallParse − Prepared** is the per-call parse tax — the empirical reason the prepared
`*Request` API exists.

## Recorded baseline

Provenance: **AMD Ryzen 7 5700G, Windows, Go 1.26.0, GOMAXPROCS=16**, `-benchtime=2000x -count=6`, medians.
These are one-machine numbers for orientation; reproduce locally with `benchstat` (below)
before drawing conclusions. CI runs the benchmarks as a once-through **smoke** (no timing
gate) on Linux/Go 1.23.x.

| Arm | ns/op (median) | B/op | allocs/op |
|---|---|---|---|
| `RoundTrip_NetHTTP` | ~113,000 | 5,940 | 69 |
| `RoundTrip_Gocurl_Prepared` | ~151,000 | 12,940 | 82 |
| `RoundTrip_Gocurl_PerCallParse` | ~137,000 | 13,800 | 92 |

**Honest reading:**

- **Execution overhead (Prepared vs NetHTTP):** ~+38µs and **+13 allocs/op**. The ~30% on a
  near-zero-latency localhost server is an artifact of the trivial "network": the overhead is
  roughly *constant* (~40µs), so as a percentage it shrinks with real latency — a few percent
  at single-digit-millisecond endpoints, and well under 1% once latency reaches the tens of
  milliseconds. That is the parity claim: a thin, constant overhead, never "faster".
- **Parse tax (PerCallParse vs Prepared):** clearest in **allocations** — +10 allocs/op
  (92 vs 82) and +850 B/op. In ns/op it is lost in localhost round-trip + timer noise,
  which is itself the honest point: parsing is cheap relative to I/O, but the prepared API
  still removes those allocations and the repeated work, and the gap widens when the
  endpoint is fast or when you build many requests.

Note the `RoundTrip_Concurrent_*` arms (via `b.RunParallel`) demonstrate connection reuse
through the cached transport; absolute concurrent ns/op depends heavily on core count.

## Allocation budgets (the regression teeth)

`TestAllocBudget_*` (alloc_budget_test.go) use `testing.AllocsPerRun` to fail the build
when a path exceeds a documented ceiling. Current baselines → budgets:

| Path | baseline allocs/op | budget |
|---|---|---|
| `ExpandVariables` | 2 | 6 |
| `Prepare` | 30 | 45 |
| `Do` (round-trip) | 76 | 100 |

Budgets are **ceilings from the measured baseline + headroom**, not zeros. Lowering a
budget is a deliberate, reviewed change; raising one needs a one-line justification in the
commit. `ExpandVariables`/`Prepare` budgets run in the normal (`-short`) test job; the
I/O-bound `Do` budget skips under `-short`.

## Profiling workflow

```bash
# CPU + memory profile of the prepared hot path
go test -run='^$' -bench=BenchmarkRoundTrip_Gocurl_Prepared -benchmem \
        -cpuprofile=cpu.out -memprofile=mem.out -benchtime=3s .
go tool pprof -top cpu.out          # where time goes
go tool pprof -alloc_space mem.out  # where allocations come from

# Latency distribution (p50/p90/p99/p999) — testing.B reports only mean ns/op.
go test -run TestLatencyDistribution -v .
```

`TestLatencyDistribution` issues N sequential `Do` calls, sorts per-request wall time, and
logs percentiles. It is `-short`-gated and informational, not pass/fail. (On Windows the
sub-millisecond timer granularity can report p50≈0; run on Linux for fine-grained
percentiles.)

## Reproducible comparison with benchstat

A single run is never authoritative — use `-count` and `benchstat`:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
go test -run='^$' -bench='RoundTrip_(NetHTTP|Gocurl_Prepared)$' -benchmem \
        -benchtime=2000x -count=10 . | tee bench.txt
benchstat bench.txt   # mean ± variance per arm; compare the two arms' deltas
```

Record the machine, Go version, and OS alongside any numbers you publish.

## Fairness: identical transport tuning on every arm

A cross-client comparison is only honest if every arm uses the **same** connection-pool
configuration — otherwise you are benchmarking pool sizes, not client overhead. The
`net/http` arm uses `benchFairTransport()` (bench_roundtrip_test.go), whose tuning is
**locked to gocurl's real default Client transport** by
`TestBenchFairness_DefaultTransportTuning`: `MaxIdleConns=100`, `MaxIdleConnsPerHost=10`,
`MaxConnsPerHost=0`, `DisableCompression=true`, `ForceAttemptHTTP2=true`. If gocurl's
defaults ever drift, that guard fails and forces the bench helper (and any published
numbers) to be updated in lockstep. (An earlier version of this suite ran the `net/http`
arm at `MaxIdleConnsPerHost=100` against gocurl's `10` — a rigged comparison; the guard
exists so that can't recur.)

## Competitive comparison (gocurl vs net/http vs resty vs req)

The competitive arms live in a **separate module**, [`benchcmp/`](../benchcmp), so the
heavy vendor graph (resty, req, quic-go, utls, …) never enters the library's own
dependency graph — enforced by `TestNoVendorDepsInRootModule`. Every arm runs over one
shared server with the identical fair transport.

```bash
cd benchcmp
go test -run='^$' -bench='BenchmarkCmp' -benchmem -count=10 .
benchstat ...   # compare arms; a single run is never authoritative
```

Representative medians (`-count=6 -benchtime=5000x`, Windows/amd64, Go 1.23, small JSON
body — **reproduce on your own hardware before quoting**):

| Arm | ns/op | B/op | allocs/op |
|---|---|---|---|
| net/http (parity bar) | ~88,500 | 5,484 | 65 |
| **gocurl prepared** | **~91,000** | **7,393** | 81 |
| resty | ~91,500 | 8,300 | 79 |
| req | ~93,000 | 7,765 | 82 |

**Reading this honestly:**

- **ns/op:** all four are within noise on an in-process server (the wire is free, so this
  mostly measures per-request CPU). gocurl is at parity with `net/http` and marginally ahead
  of resty/req here — *parity is the claim; we do not claim to beat `net/http`*.
- **B/op — gocurl is the lightest of the three full-featured clients** (7,393, below req's
  7,765 and resty's 8,300), behind only raw `net/http`. This was *not* always true: an
  earlier version of this suite measured gocurl at ~12,900 B/op (the heaviest arm), and we
  published that loss. Profiling found the cause — a per-`Do` `newRand()` allocated a
  ~4.9 KiB `[607]int64` RNG state on **every** request even when no retry ran. Making the
  jitter RNG lazy (created only when a retry actually needs it) removed it; the win is
  guarded by `TestByteBudget_Do` so it cannot regress. Combined with `clone-the-small`
  (`TestCloneSmall_NoDeepClonePerDo`), gocurl now carries its full resilience pipeline for a
  smaller per-`Do` footprint than the thinner wrappers.
- **allocs/op:** gocurl (81) sits between resty (79) and req (82) — effectively tied.
- **Caveat (not perfectly apples-to-apples):** resty and req **buffer** the full response
  body by default; gocurl **streams** it. For a tiny body this slightly favors the bufferers;
  for large responses gocurl's streaming is the safer default. The harness drains every arm,
  but the body-handling models differ by design.

The takeaway matches our claim policy: **parity with `net/http` on latency, and a lighter
per-request footprint than resty/req** — measured, reproducible, and regression-guarded,
never a marketing claim.

## The soak/leak tests are elsewhere

Goroutine-leak, connection-reuse, and the bounded soak loop live with the quality suite
(see [CONTRIBUTING.md](../CONTRIBUTING.md)), not here — this document is about throughput
and allocation methodology.
