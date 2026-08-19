# v1.0 readiness checklist

gocurl is **production-grade and pre-1.0**: the engine is hardened, tested, and proven;
the remaining gate to a `v1.0.0` tag is *API-contract confidence from real usage*, which is
the maintainer's call (per [Spec 11](../specs/11-api-stability-and-migration.md)). This
checklist tracks the objective gates. When all are green, cutting the tag is a near-trivial
step. Reliability and resource evidence includes `TestFault_OverallRetryBudget`,
`TestFault_NoSecretLeakOnFailurePaths`, and `TestClient_Soak`.

Every "proven" item below cites an un-skipped test/benchmark; the honesty doc-lint
(`TestDocHonestyLint`) fails the build if a claim here lacks its citation.

The post-audit hardening gates (`RH-*`) that followed Milestone 12 are all complete; the
evidence for each is summarized below and enforced by the test suite.

## Reliability (M12 Phase A) — ✅ done

- [x] Two-tier fault harness (RoundTripper-injected + real-transport) covering retry
      classification, breaker, per-attempt deadline, DNS, pool exhaustion, premature EOF,
      header timeout, panic-mid-flight, context-cancel mid-stream.
- [x] Overall retry budget bounds the whole operation (`TestFault_OverallRetryBudget`).
- [x] h2 `GOAWAY`/`RST_STREAM` retried (`TestFault_H2ErrorsRetried`).
- [x] Graceful shutdown never truncates a live stream and is never wedged by a panic
      (`TestFault_ShutdownWaitsForOpenBody`, `TestFaultT2_PanicMiddlewareDoesNotWedgeShutdown`).
- [x] Redirect cap classifies as `ErrTooManyRedirects` (`TestFault_ClientRedirectCapClassifiable`).
- [x] Untrusted-server memory bounds (`TestFault_BufferingHelpersBoundedAgainstBomb`).
- [x] Secrets redacted on every failure path (`TestFault_NoSecretLeakOnFailurePaths`).
- [x] Curl wire-parity proven against a real curl binary (`TestCurlParity_DifferentialVsRealCurl`).

## Performance (M12 Phase B) — ✅ done

- [x] Clone-the-small: the immutable recipe is no longer deep-cloned per `Do`
      (`TestCloneSmall_NoDeepClonePerDo`), validated under concurrency
      (`TestCloneSmall_DoesNotMutateSharedRequest`, `TestCloneSmall_NoDefaultBleedAcrossClients`).
- [x] Fair competitive benchmarks (net/http, gocurl, resty, req) with identical transport
      tuning enforced by a guard (`TestBenchFairness_DefaultTransportTuning`); honest results
      table including where gocurl loses, in [benchmarking.md](benchmarking.md).
- [x] Latency p50/p99/p999 harness (`TestLatencyDistribution`); ratcheted `Do` alloc budget
      (`TestAllocBudget_Do`); root dependency graph stays clean (`TestNoVendorDepsInRootModule`).

## Resource stability (M12 Phase C) — ✅ done

- [x] Two-arm soak (uninstrumented + instrumented) shows no goroutine/heap growth trend and
      publishes the instrumentation alloc delta (`TestClient_Soak`).
- [x] Backpressure under `MaxConnsPerHost` drains without deadlock (`TestClient_Soak_Backpressure`).

## Operability (M12 Phase D) — ✅ done

- [x] [operations.md](operations.md): timeout taxonomy, pool/retry tuning, per-`Kind`
      failure-mode playbook, observability runbook, untrusted-server memory bounds,
      easy-start→production checklist.
- [x] Threat model in [SECURITY.md](../SECURITY.md).
- [x] Honesty doc-lint gating performance/reliability claims on cited, un-skipped tests
      (`TestDocHonestyLint`).

## Shipped: v0.5.0 — the public release

- [x] **July 2026 hardening gates.** Performance enforcement in CI, the composition
      robustness matrix, critical-path coverage floors, claim/evidence integrity, a
      reproducible benchmark baseline, and the final release audit are all complete.
- [x] **Tagged and published as `v0.5.0`.** See the [CHANGELOG](../CHANGELOG.md).

**Why v0.5.0 and not v1.0.0.** Every *quality* gate above is green — this is not a
maturity hedge. It is a semver decision. A `v1` major is a promise that the exported API
will not change, and in Go that promise is expensive to revise: `v1 → v2` forces every
importer to rewrite their import path to `/v2`, whereas `v0.x → v1.0` costs them nothing.
Shipping at v0.5.0 lets real integrations pressure-test the API shape while breaking
changes are still cheap for users. It is the conservative choice, not the timid one.

## Remaining before a v1.0 tag — maintainer's call

- [ ] **API-contract confidence.** Validate the public surface against real-world usage; the
      `api.txt` / `api_options.txt` guards lock it, but v1 is a *stability promise*, made only
      once the surface has been exercised by several independent integrations.
- [ ] **curl-flag coverage** continues to expand (see [VISION.md](../VISION.md)); the gaps are
      documented and non-blocking for the engine's production use.
- [ ] **Tag `v1.0.0`.** When the API has survived real usage without a breaking change for a
      meaningful stretch, promoting v0.x → v1.0.0 is a near-trivial, importer-invisible step.
