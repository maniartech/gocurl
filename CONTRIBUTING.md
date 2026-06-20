# Contributing to GoCurl

Thanks for your interest in improving GoCurl. This document covers how to build,
test, and submit changes.

## Project scope

GoCurl is a curl-ergonomic HTTP client built on `net/http`. Before contributing,
please read [VISION.md](VISION.md) — features and claims that don't serve the core
promise (paste a curl command from API docs into Go) are out of scope.

The single most valuable area of work right now is **parser correctness**: making
real-world curl commands from popular API docs run verbatim, with regression tests.

## Building

```bash
go build ./...
go install ./cmd/gocurl
```

Requires Go 1.22+.

## Testing

The library uses two test layers. **Put each new test in the right layer:**

- **Whitebox tests** live next to the code as `<file>_test.go` in the same package
  (`package gocurl` or a subpackage) and may touch unexported identifiers. Use them
  for parser/tokenizer correctness (drive the real pipeline via the `parseCmd` helper
  in `parser_internal_test.go`), variable-expansion internals, `RequestOptions`
  finalization, retry-loop helpers, TLS config construction, and error classification
  — anything needing internal state.
- **Blackbox tests** live in `tests/` as `package tests` and import gocurl exactly as
  a consumer would. Use them for the public API (`Curl*`, `Client`/`Prepare`/`Do`),
  CLI behavior and exit codes (`tests/cli_blackbox_test.go` builds `cmd/gocurl` once
  in `TestMain`), and end-to-end request semantics against an `httptest` server.

Run everything:

```bash
go test ./...                  # all packages + blackbox suite (offline, incl. soak)
go test -short -race ./...     # canonical fast/CI path; hermetic
go test -cover ./...           # coverage
```

Tests must be **hermetic**: use `httptest` servers, never live internet endpoints;
temp files via `t.TempDir()`, env via `t.Setenv`. Any test needing real network must
be gated behind `testing.Short()` **and** an opt-in env var.

### The curl-compat corpus

The headline promise — "paste any curl command from the docs and it works" — is
guarded by [`internal/corpus/compat.json`](internal/corpus/compat.json): each entry
is a verbatim doc command plus its expected parse, run by **both** layers
(`TestCurlCompatCorpus_Parse` whitebox + `TestCurlCompatCorpus_Execute` blackbox).
**Adding a documented command is a one-line append to `compat.json`** — no new Go
test function. Use obviously-fake tokens; never commit real credentials.

### Fuzz, leak, and soak

```bash
go test -run='^$' -fuzz='FuzzTokenize$'     -fuzztime=30s ./tokenizer/
go test -run='^$' -fuzz='FuzzParseCommand$' -fuzztime=30s .
go test -run 'TestClient_Do_(NoGoroutineLeak|ReusesConnections)' .
GOCURL_PROFILE=$(mktemp -d) go test -run TestClient_Soak .   # writes cpu/mem pprof
```

Fuzz targets must never panic; minimized crashers are committed under `testdata/fuzz/`
as permanent regression seeds. CI enforces a coverage floor (75% overall, measured
with `-coverpkg=./...`) and a 30s fuzz smoke per target.

## Code style

- `gofmt` is mandatory — CI fails on any unformatted file. Run `gofmt -w .`.
- `go vet ./...` must be clean.
- Keep cyclomatic complexity reasonable; prefer small, focused functions.
- Match the style and naming of surrounding code.

## Submitting changes

1. Open an issue to discuss substantial changes before sending a PR.
2. Add or update tests for any behavior change (whitebox and/or blackbox).
3. Ensure `gofmt -l .` is empty and `go test -race ./...` passes.
4. Keep PRs focused; describe the motivation and link the relevant issue.

## Reporting bugs

For parser bugs, include the exact curl command and what curl does versus what
GoCurl does. For security issues, see [SECURITY.md](SECURITY.md) instead of opening
a public issue.
