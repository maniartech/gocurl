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

The library uses two test layers:

- **Whitebox tests** live next to the code as `<file>_test.go` in the same package.
  They exercise internal helpers and edge cases.
- **Blackbox tests** live in `tests/` as an external package. They exercise the
  public API and the `gocurl` CLI exactly as a consumer would — across all
  commands, flags, and options.

Run everything:

```bash
go test ./...                  # all packages + blackbox suite
go test -race ./...            # race detector
go test -short ./...           # skip any network-dependent tests
go test -cover ./...           # coverage
```

Tests must be **hermetic**: use `httptest` servers, never live internet endpoints.
Any test that requires the network must be gated behind `testing.Short()`.

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
