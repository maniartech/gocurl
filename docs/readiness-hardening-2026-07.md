# Readiness hardening — resumable execution tracker

This file is the durable resume point for the July 2026 readiness audit. Work is
idempotent: every task has a stable identifier, an objective verification command, and
an evidence field. A resumed session starts at the first unchecked task, inspects the
current tree, and reruns the command before changing the checkbox. No task depends on a
commit, generated timestamp, or one-off local state.

## Resume protocol

1. Run `git status --short` and preserve all existing changes.
2. Read this file from the top and select the first unchecked `RH-*` task.
3. Run its verification command before editing. If it already passes and the required
   files are present, mark it complete without duplicating work.
4. Implement only the missing portion, rerun the verification, then record concise
   evidence below the task.
5. Never stage, commit, push, or create a PR unless the maintainer explicitly asks.

## Tasks

- [x] **RH-00 — Durable idempotent tracker.** This document exists and defines the
  resume protocol.
  - Verify: `Test-Path docs/readiness-hardening-2026-07.md`
  - Evidence: tracker created with stable task IDs and command-based completion rules.

- [x] **RH-01 — PR performance enforcement.** Non-short allocation and byte budgets run
  in CI; deterministic composition-path budgets run in the same job.
  - Verify: inspect `.github/workflows/ci.yml`, then run the exact performance-gate
    command declared there locally.
  - Evidence: `.github/workflows/ci.yml` has a non-short `performance-gates` job;
    the exact command passed locally with Do=71 allocs/op and 5,894 B/op.

- [x] **RH-02 — Composition performance budgets.** Plain handler, Retry, Observe, SSRF,
  and full injected-chain overhead have deterministic allocation/byte ceilings.
  - Verify: `go test -run '^TestCompositionPerformance_' -count=1 -v .`
  - Evidence: `composition_performance_test.go` gates all five paths; measured
    bare=5, Retry=7, Observe=23, SSRF=12, full chain=32 allocs/op. The separate
    `benchcmp` gate measured gocurl=2,489 B/24 allocs, below the pinned Resty and Req
    arms, and is enforced by the scheduled workflow.

- [x] **RH-03 — Composition robustness matrix.** Cancellation, replay caps,
  Retry-After, concurrent reuse, redirect pinning, TLS SNI, and opaque-transport
  fail-closed behavior are executable invariants.
  - Verify: `go test -run '^TestCompositionRobustness_' -count=1 .`
  - Evidence: `composition_robustness_test.go` covers all named cases; the focused
    suite passed both normally and with `-race`.

- [x] **RH-04 — Critical-path coverage.** Retry, middleware composition,
  observability, and SSRF enforcement meet explicit per-file/function coverage bars;
  the repository-wide CI floor is raised from 75% without gaming generated/no-op code.
  - Verify: run the coverage command in `.github/workflows/ci.yml` and its critical-file
    coverage checker.
  - Evidence: patched Go 1.26.5 coverage is 82.8% overall (floor raised 75→82),
    root 82.2%, options 91.8% (package floor 90), proxy 69.2% (package floor 65),
    tokenizer 93.9%. Retry/Observe/replay/SSRF enforcement functions have explicit
    85–100% gates in `scripts/check-coverage.sh`.

- [x] **RH-05 — Claim/evidence integrity.** Strong claims require an adjacent named
  test/benchmark citation; readiness, Spec 14, ROADMAP, README, and benchmark numbers
  agree.
  - Verify: `go test -run '^TestDocHonestyLint$' .`
  - Evidence: `TestDocHonestyLint` now requires a citation within six lines of every
    strong-claim occurrence and passes after adding local evidence. Spec 14 explicitly
    distinguishes completed M12 items from stricter partial requirements; v1 readiness
    points to this tracker as the authoritative post-audit gate.

- [x] **RH-06 — Reproducible benchmark baseline.** Current-machine measurements,
  budgets, commands, pinned competitor versions, and semantic caveats are recorded from
  one coherent run.
  - Verify: run the documented root and `benchcmp` commands; allocation/byte values must
    remain within committed gates.
  - Evidence: `docs/benchmarking.md` records one sequential Go 1.26/Windows/5700G run:
    prepared=77 allocs/6,648 B, per-call parse=92/8,378 B, and deterministic competitive
    gocurl=24/2,489 B versus pinned Resty=28/3,684 and Req=39/3,956. CI gates bytes and
    allocations; timing remains advisory.

- [x] **RH-07 — CI breadth.** Minimum and current Go, Linux and Windows, static analysis,
  scheduled long soak, scheduled fuzz, and competitive diagnostics are represented
  without making timing noise a PR gate.
  - Verify: validate `.github/workflows/ci.yml` syntax and run every local command that
    does not require GitHub-hosted infrastructure.
  - Evidence: workflow YAML parses successfully; CI covers Go 1.25/current stable,
    Ubuntu race and Windows smoke, pinned staticcheck v0.7.0 and govulncheck v1.6.0,
    weekly 10-minute-per-arm soak and two five-minute fuzz targets. Staticcheck is
    clean. Govulncheck is clean with Go 1.26.5 and x/net 0.55.0; the dependency fix
    required raising the minimum Go version to 1.25 across runtime modules.

- [x] **RH-08 — Final release audit.** Build, vet, full tests, race, coverage, fault
  matrix, fuzz smoke, soak, API guards, documentation lint, and benchmarks are green.
  - Verify: commands listed in the final evidence section below.
  - Evidence: all commands below passed with Go 1.26.5. The only non-test module,
    `scripts`, correctly reports that it contains no Go packages; its coverage gate
    scripts were exercised directly. No files were staged or committed.

- [x] **RH-09 — Published-book parity.** `book` and `book2` state the Go 1.25
  requirement and accurately document exported Retry/Observe composition and SSRF
  dial pinning without presenting the legacy request-mutator middleware as the modern
  composition API.
  - Verify: `go test -run '^TestBookReleaseParity$' .`, then run the `book2` module
    tests and the root documentation guards.
  - Evidence: `book/README.md` is now an intentional compatibility landing page for the
    maintained `book2` manuscript. Obsolete module ownership, `Process`, root-level
    options symbols, Go 1.18/1.21 prerequisites, and wrong return arity were removed.
    The API reference distinguishes modern execution middleware from legacy request
    mutators and documents retry idempotency, observability ordering, redirect checks,
    DNS-rebinding-safe dial pinning, and fail-closed transports. The hermetic production
    composition example ran with `status=200 body="ready" retries=1`; book2 build, vet,
    and all package tests passed. `TestBookReleaseParity` prevents these facts drifting.

## Final evidence

Populate this section only from actual command output. Keep machine-dependent timing
advisory; allocation counts, byte counts, correctness, race, and coverage are gates.

- Build/vet: `go build ./...` and `go vet ./...` passed; staticcheck v0.7.0 passed.
- Full tests: `go test -count=1 ./...` passed in the root module; `go test ./...`
  also passed in `book2`, `benchcmp`, `observability/otel`, and
  `observability/prometheus`.
- Race: `go test -short -race -count=1 ./...` passed. Performance tests skip their
  byte counters under race because the race runtime intentionally inflates them; the
  same gates run unskipped in the dedicated performance job.
- Coverage: 82.8% overall; root 82.2%, options 91.8%, proxy 69.2%, tokenizer 93.9%.
  All six critical-function floors passed (Retry 90.0%, Observe 90.0%, replay setup
  91.3%, retry loop 89.2%, SSRF RoundTrip 100%, dial pinning 85.7%).
- Fault matrix: `go test -count=1 -run '^(TestFault_|TestFaultT2_)' .` passed.
- Fuzz: 10-second parser and tokenizer fuzz smokes passed after replaying 541 and 90
  baseline inputs respectively; both found additional interesting inputs without a
  failure.
- Soak: both `TestClient_Soak` arms plus backpressure passed with `GOCURL_SOAK=5s`.
- Security/API/docs: govulncheck v1.6.0 reported `No vulnerabilities found`;
  public-API golden and documentation-honesty guards passed.
- Performance gates: composition paths measured bare 5 allocs/704 B, Retry 7/1,392,
  Observe 23/1,711, SSRF 12/912, and full chain 32/2,619. Prepared `Do` measured
  71 allocs/5,894 B; all are below their hard ceilings.
- Root benchmark smoke: all three loopback round-trip arms passed; their timing remains
  advisory because Windows loopback noise can reorder them.
- Competitive benchmarks: the deterministic budget test measured gocurl at 24
  allocs/2,489 B, below pinned Resty at 29/3,737 and Req at 40/4,006. The benchmark
  smoke also passed; timing is advisory.
- Published books: `TestBookReleaseParity`, the root API/documentation guards, the
  hermetic composition example, and `book2` build/vet/tests passed.
