# Benchmarking & Performance Methodology

This document defines how gocurl is benchmarked, what we measure, and — critically —
**what we will and will not claim**. It implements [Spec 10](../specs/10-benchmarking.md).

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

## The soak/leak tests are elsewhere

Goroutine-leak, connection-reuse, and the bounded soak loop live with the quality suite
(see [CONTRIBUTING.md](../CONTRIBUTING.md)), not here — this document is about throughput
and allocation methodology.
