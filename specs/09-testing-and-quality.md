# Testing & Quality Strategy

> Status: Draft for review · Spec 09

## Goals

- Make the test suite a *contract*, not an afterthought: every behavior promised in
  the curl-ergonomics positioning (VISION.md) and the resilience/observability specs
  has a hermetic, race-clean test that fails loudly if it regresses.
- Formalize the two-layer test architecture already present in the repo — **whitebox**
  sibling `*_test.go` files (`package gocurl`) and **blackbox** consumer tests under
  `tests/` (`package tests`) — into explicit rules about what belongs where.
- Lock a **hermetic, `httptest`-only** policy for the default test run: `go test ./...`
  must never touch the live network, never bind a fixed port, and never depend on
  external services (Stripe/GitHub/OpenAI). Anything that does is gated behind `-short`.
- Build a **curl-compat regression corpus**: real command strings copied verbatim from
  API docs, asserting the parser produces the right `*options.RequestOptions`. This is
  the spec's headline product surface and must be the most heavily covered path.
- Add **fuzzing** of the tokenizer + parser (`go test -fuzz`), **race detection**,
  **goroutine/connection leak detection**, and **soak/load** tests with `pprof`, so
  "production-grade" is demonstrated, not asserted.
- Define **coverage targets per package** and **CI gates** (gofmt, vet, build,
  `-short -race`, coverage floor) that match and extend the existing
  `.github/workflows/ci.yml`.
- Establish **Definition of Done conventions** so development is idempotent: re-running
  the suite, the fuzzers, or the corpus is deterministic and side-effect-free.

## Non-goals

- Not a benchmarking *claim* document. Benchmarks exist to detect regressions and to
  back any perf statement with a reproducible number (per the honesty constraints);
  they do not establish "faster than net/http". Benchmark methodology beyond regression
  gating belongs to a future performance spec.
- Not replacing the execution engine's own tests: we test gocurl's behavior over
  `net/http`/`golang.org/x/net/http2`, not net/http internals.
- Not mandating 100% coverage. Coverage is a floor and a trend signal, not a target to
  game.
- Not specifying the resilience/observability *behaviors* themselves (specs for retries,
  circuit breaker, tracing own those). This spec says how they get tested.

## Design

### Layer 1 — Whitebox (sibling `*_test.go`, `package gocurl` / subpackage)

Lives next to the code it tests, compiled into the package, may touch unexported
identifiers. Existing examples to follow as the canonical style:

- `parser_internal_test.go` — drives the real parsing pipeline via the unexported
  `preprocessMultilineCommand` → `tokenizer.Tokenize` → `expandEnvInTokens` →
  `convertTokensToRequestOptions` chain (helper `parseCmd`), stopping *before*
  execution. This is exactly where parser-correctness assertions belong.
- `tokenizer/tokenizer_test.go` — table-driven assertions on the token stream.
- `features_internal_test.go`, `tls_utils_test.go` — exercise unexported helpers and
  `LoadTLSConfig`/`VerifyCertificatePin`/`ValidateRequestOptions` (security.go).

**Belongs in whitebox:** parser/tokenizer correctness, variable-expansion internals
(`expandVariables`, `expandVarsInTokens`, `expandEnvInTokens`), `RequestOptions`
finalization, retry-loop helpers (`bufferRequestBody`, `checkContextCancelled`,
`retryLoop` in retry.go), TLS config construction, compression internals, error
classification. Anything needing access to internal state.

### Layer 2 — Blackbox (`tests/`, `package tests`)

Imports `github.com/maniartech/gocurl` exactly as an external consumer would, plus
drives the CLI as a subprocess. Canonical examples already present:

- `tests/api_blackbox_test.go` — `newEchoServer` (httptest) records the last request;
  asserts `CurlString`/`CurlJSON`/`CurlBytes`/`CurlDownload` behavior, default
  User-Agent, redirect following, env expansion, `CurlWithVars` env non-leak,
  gzip decode, and the no-stdout-side-effect guarantee.
- `tests/cli_blackbox_test.go` — builds `cmd/gocurl` once in `TestMain`, runs it via
  `os/exec`, asserts body-printed-once, `-o`/`-s`/`-i`/`-v` redaction, exit codes.

**Belongs in blackbox:** the public API surface (package funcs and the new `Client` /
`Prepare` / `Do` from spec set), CLI behavior and exit codes, end-to-end request
semantics against an httptest server, and the curl-compat corpus execution path.

```go
// tests/ blackbox skeleton (already the house style):
func newEchoServer(t *testing.T) *echoServer   // httptest.Server, t.Cleanup(Close)
func mustString(t *testing.T, args ...string) (string, *http.Response)
func runCLI(t *testing.T, args ...string) (stdout, stderr string, exit int) // skips if bin==""
```

### Hermetic policy (default run is offline)

- Every test server is `httptest.NewServer` / `NewTLSServer`, closed via `t.Cleanup`
  or `defer`. No `:8080`, no hard-coded ports, no DNS to real hosts. URLs that *look*
  external (`https://api.example.com`, `https://example.com`) are only ever used as
  parse inputs in whitebox tests — never dialed.
- `TestMain` in `tests/` already degrades gracefully: if `go build` of the CLI fails
  (offline toolchain), it sets `gocurlBin = ""` and CLI tests `t.Skip`. Keep this.
- Temp files use `t.TempDir()`; env mutation uses `t.Setenv` (auto-restored). No test
  writes outside its temp dir or leaves global state mutated — this is what makes the
  suite idempotent.

### `-short` gating

`go test -short ./...` is the canonical fast/CI path and MUST be hermetic and fast.
Reserve the non-short path for: real-network/integration smoke tests, soak/load runs,
and the 10k-goroutine stress tests. Pattern already used in `race_test.go`:

```go
func TestStressTest_10kGoroutines(t *testing.T) {
    if testing.Short() { t.Skip("Skipping stress test in short mode") }
    // ...
}
// Or scale down instead of skipping (TestConcurrentResponseBufferPool):
goroutines := 100; if testing.Short() { goroutines = 10 }
```

Any test that performs real DNS/network must `if testing.Short() { t.Skip(...) }` and
additionally guard on an opt-in env var (e.g. `GOCURL_LIVE=1`) so even
`go test ./...` (no `-short`) stays green offline.

### Curl-compat regression corpus

A data-driven corpus of real command strings copied verbatim from public API docs,
each mapped to expected parse results. Stored as testdata, executed by a whitebox test
that reuses the `parseCmd` helper from `parser_internal_test.go`.

```go
// testdata/corpus/compat.go (or a //go:embed JSON manifest)
type CompatCase struct {
    Name    string            // "stripe_create_charge"
    Source  string            // doc URL the command was copied from
    Command string            // verbatim curl string, multiline allowed
    Want    ExpectedRequest   // method, URL (lossless), headers, body, query, auth
}

// whitebox runner:
func TestCurlCompatCorpus(t *testing.T) {
    for _, c := range loadCorpus(t) {
        t.Run(c.Name, func(t *testing.T) {
            opts := parseCmd(t, c.Command)        // tokenize+expand+convert, no exec
            assertMatches(t, c.Want, opts)         // method/URL/headers/body/query
        })
    }
}
```

Seed coverage (minimum): Stripe (`-u $KEY:`, repeated `-d` form fields),
GitHub (`-H 'Authorization: Bearer ...'`, `-H 'Accept: application/vnd.github+json'`,
`-X PATCH`), OpenAI (`-H 'Content-Type: application/json'` + `-d '{...json...}'`).
Each case is also executed end-to-end against `newEchoServer` in blackbox to confirm
the parsed request reaches the wire intact (URL losslessness, JSON body verbatim,
auth header form) — extending the existing `TestBlackbox_JSONBodyVerbatim` /
`TestBlackbox_BasicAuth` pattern. Adding a new doc command is a one-line corpus
append, the regression guard for the product's core promise.

### Fuzzing the parser

The tokenizer and converter are the riskiest untrusted-input surface. Add native Go
fuzz targets (run with `go test -fuzz`):

```go
// tokenizer/fuzz_test.go
func FuzzTokenize(f *testing.F) {
    f.Add("curl -X POST -d '{\"k\":\"v\"}' https://api.example.com")
    f.Add("curl -H 'Authorization: Bearer $TOK' $URL/data")
    f.Fuzz(func(t *testing.T, cmd string) {
        tok := tokenizer.NewTokenizer()
        _ = tok.Tokenize(cmd)   // must never panic; error is fine
    })
}

// fuzz_parser_test.go (package gocurl)
func FuzzParseCommand(f *testing.F) {
    // seed from the compat corpus + edge cases (unterminated quotes, $VAR, @file)
    f.Fuzz(func(t *testing.T, cmd string) {
        defer func() { if r := recover(); r != nil { t.Fatalf("panic: %v", r) } }()
        tok := tokenizer.NewTokenizer()
        if tok.Tokenize(cmd) != nil { return }
        _, _ = convertTokensToRequestOptions(expandEnvInTokens(tok.GetTokens()))
    })
}
```

Invariant: no panic, no unbounded allocation, no infinite loop for any input. `@file`
and env-expansion paths in fuzz must not read arbitrary host files — fuzz targets call
the pure tokenize/convert path, and any `-d @file` resolution is exercised only with a
`t.TempDir()` path (as `TestParser_DataAtFileReadsContents` already does). Discovered
crashers are committed under `testdata/fuzz/` so they become permanent regression seeds.

### Race, leak, soak/load

- **Race:** the canonical CI command is `go test -short -race ./...`. Concurrency tests
  already exist (`race_test.go`, `race_concurrent_test.go`) covering parse-under-load,
  expansion, retry, and buffer-pool reuse. New `Client.Do` concurrency (shared pooled
  transport from `clientpool.go`) gets a dedicated parallel test hitting one
  `httptest.Server` from N goroutines and asserting connection reuse + no data races.
- **Goroutine leak detection:** add a lightweight check that snapshots
  `runtime.NumGoroutine()` (with a short settle/`runtime.Gosched` retry loop) before
  and after a batch of `Client.Do` calls, asserting it returns to baseline once all
  response bodies are closed. This catches transports/dialers/redirect machinery that
  leak goroutines.
- **Connection leak detection:** wrap an `httptest.Server` handler with a counter of
  open connections (via `Server.Config.ConnState`), drive many requests with body
  close, and assert idle connections are reused (not unbounded) — validates the
  pooled-transport reuse promise.
- **Soak/load + pprof:** a `-short`-gated test runs a sustained request loop for a
  bounded duration/iteration count against an httptest server; when
  `GOCURL_PROFILE=<dir>` is set it writes `cpu.pprof`/`mem.pprof` via `runtime/pprof`
  for offline `go tool pprof` analysis. Asserts zero errors and stable goroutine count
  across the run.

### Coverage targets & CI gates

Per-package floors (measured by `go test -short -coverprofile`):

| Package / area | Floor |
|---|---|
| `tokenizer/` | 90% |
| parser/convert (`convert.go`, expansion) | 85% |
| `options/` | 85% |
| `security.go` (TLS, pinning, validation) | 80% |
| root API + `Client` (`api.go`, `client.go`, `clientpool.go`) | 80% |
| resilience (`retry.go`, future middleware) | 80% |
| overall module | 75% |

CI extends the current `.github/workflows/ci.yml` (gofmt-clean excluding `book2/`,
`go vet ./...`, `go build ./...`, `go test -short -race -timeout 240s ./...`, coverage
summary) with: (1) a coverage-floor gate that fails if overall `go tool cover -func`
drops below 75%; (2) a short fuzz smoke step (`go test -run=^$ -fuzz=Fuzz -fuzztime=30s`
on tokenizer + parser) so fuzz targets are kept compiling and seeds green; (3) the Go
version matrix stays aligned with `go.mod` (`go 1.22.3`).

## Behavior & edge cases

- **Idempotent re-runs:** no test depends on prior-run artifacts. The `.coverage/`
  directory and `coverage.out` are git-ignored build outputs, never committed test
  inputs. `tests/TestMain` cleans its temp build dir.
- **CLI build skip:** if the toolchain can't build `cmd/gocurl`, CLI tests skip rather
  than fail the suite (preserve current `TestMain` behavior).
- **Windows/Unix parity:** `tests/` already branches on `os.PathSeparator` (`isWindows`)
  and appends `.exe`. New file-output and `-O`/`-o` tests must use `filepath.Join`,
  never hard-coded separators.
- **Secret hygiene in tests:** verbose/log assertions confirm redaction
  (`TestCLI_VerboseRedactsSecrets`, `SanitizeHeadersForLogging`). Corpus commands use
  obviously-fake tokens; no real credentials in testdata.
- **Fuzz corpus growth:** crashers are minimized and committed; the suite must stay
  green on the committed `testdata/fuzz` corpus even without `-fuzz`.
- **Flake control:** time-based assertions use generous bounds and context deadlines;
  no `time.Sleep`-based synchronization where a channel/`sync` primitive works. Soak
  durations are bounded so `-short` CI stays fast.
- **Deprecated paths:** `Process` (process.go) is deprecated but still exercised by
  existing tests; new resilience features are tested through `Client.Do`, not `Process`.

## Acceptance criteria / Definition of Done

- [ ] `go test ./...` (no `-short`, default) passes fully offline with no live network.
- [ ] `go test -short -race -timeout 240s ./...` passes locally and in CI, race-clean.
- [ ] Whitebox vs blackbox placement rules documented in `CONTRIBUTING.md`; new tests
      follow them (parser/internal → sibling; public API + CLI → `tests/`).
- [ ] Curl-compat corpus exists with ≥3 verified cases each for Stripe, GitHub, OpenAI;
      each case has a whitebox parse assertion and a blackbox echo-server execution.
- [ ] Adding a new doc command to the corpus requires only a data append (no new Go
      test function).
- [ ] `FuzzTokenize` and `FuzzParseCommand` exist, compile, and have committed seeds;
      `go test -fuzz=Fuzz -fuzztime=30s` runs clean on both targets.
- [ ] Goroutine-leak and connection-reuse tests exist for `Client.Do` over the pooled
      transport and pass.
- [ ] A `-short`-gated soak/load test exists and writes pprof profiles when
      `GOCURL_PROFILE` is set.
- [ ] CI gates: gofmt (excl. `book2/`), `go vet`, build, `-short -race`, coverage floor
      (75% overall), and a 30s fuzz smoke step — all green.
- [ ] Per-package coverage floors met; `go tool cover -func` summary recorded in CI.
- [ ] No test leaves files outside `t.TempDir()`, mutates global env without
      `t.Setenv`, or binds a fixed port.

## Dependencies

- **Spec set core** (Client / `New` / `Prepare` / `Do`, pooled transport, middleware):
  the public-API blackbox tests and `Client.Do` concurrency/leak tests assert that
  surface. Builds on `clientpool.go` transport caching.
- **Resilience spec** (idempotency-aware `RetryPolicy`, backoff, circuit breaker): the
  retry/leak tests here verify its behavior; extends `retry.go` test patterns.
- **Observability spec** (tracer/metrics/logger interfaces): redaction and lifecycle-hook
  assertions extend `verbose_test.go` / `SanitizeHeadersForLogging`.
- **Errors spec** (classification, `errors.Is/As`): coverage of error-helper tests.
- **VISION.md** positioning: the corpus exists to defend the "paste curl from docs"
  promise.

## Open questions / decisions to confirm in review

- **Corpus format:** embedded Go table vs `//go:embed` JSON/YAML manifest under
  `testdata/corpus/`? Proposed: `//go:embed` JSON so non-Go contributors can add cases.
  (Unconfirmed.)
- **Coverage floors:** the per-package numbers above are proposed starting points, not
  measured. Confirm against an actual `go tool cover -func` baseline before gating CI on
  them, to avoid an immediate red build.
- **Leak detection dependency:** roll our own `NumGoroutine` snapshot (zero new deps,
  matches the current no-extra-deps posture) vs adopt `go.uber.org/goleak` (more robust,
  but a new dependency). Proposed: home-grown first; revisit if flaky.
- **Live integration tier:** keep real-network smoke tests in-repo behind
  `GOCURL_LIVE=1` + non-short, or move them to a separate manually-triggered CI job?
  Proposed: in-repo, opt-in env, never run by default CI.
- **Fuzz in CI:** 30s smoke per target is proposed; should we also schedule a longer
  nightly fuzz run (e.g. 10m) on a cron workflow? (Unconfirmed — depends on CI budget.)
- **Live HTTP benchmarks:** `BenchmarkRequestAPI` is currently skipped (needs a server).
  Should we wire it to an `httptest` server so a parse+execute benchmark exists for
  regression tracking, or leave perf to a dedicated future spec? Proposed: add the
  httptest-backed benchmark here for regression only, no comparative claims.
