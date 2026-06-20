# Dev / CI helper scripts

These live in their own module (`scripts/go.mod`) and are **not** part of the
published `gocurl` library. They are conveniences for development and CI.

## `verify-examples.sh` — compile-check the book examples

```bash
./scripts/verify-examples.sh          # build-check every book2 example (default)
./scripts/verify-examples.sh --vet    # also run `go vet ./...`
./scripts/verify-examples.sh --help
```

Builds the whole `book2/` module in one shot against the **local, in-tree** library
(book2 pins it via `replace => ../`). This is the canonical guard that a public-API
change does not silently break the documented examples — exactly the regression that
orphaned the `options` builder when `Execute` was removed.

Default is build-only (the deterministic API-compatibility check). `--vet` is opt-in
because the example sources carry intentional trailing-newline `Println` spacing the
book prose relies on, which `go vet` flags. It is non-interactive and exits non-zero
on any failure, so it is ready to drop into CI when desired (not wired in yet).

It does **not** run the examples — that would make live network calls
(httpbin.org, api.github.com, …); compilation is what proves API compatibility.

## `build.sh` — build the gocurl CLI

Builds the `cmd/gocurl` binary.

## `test-runner.sh` — coverage report

Runs `go test ./...` with coverage, writes `.coverage/coverage.html`, prints the
total, and opens the HTML report.

## `create-test-certificates.sh` — TLS test fixtures

Generates the self-signed certificates used by the TLS tests.
