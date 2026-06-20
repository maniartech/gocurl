# GoCurl

**Paste any curl command from any API doc straight into Go.** Test it in the shell, run
the exact same command in your code — no translation, no guesswork.

```go
resp, err := gocurl.Curl(ctx, `
  curl https://api.github.com/repos/golang/go
`)
```

GoCurl is a curl-ergonomic HTTP client for Go, built on `net/http`, with a CLI that
shares the exact same syntax. It exists to remove the tax every Go developer pays when
integrating a new API: mentally compiling a curl snippet from the docs into
`http.NewRequest`, headers, body encoding, and auth.

> See [VISION.md](VISION.md) for what we're building and why.

## Project status

**Pre-1.0 and under active development.** The public API may still change and curl-flag
coverage is still expanding, so pin a version and check the [CHANGELOG](CHANGELOG.md) when
upgrading. The core is hardened and tested — race-clean, fuzzed, with a coverage gate in
CI — but treat anything pre-1.0 as subject to refinement. Feedback and contributions are
very welcome.

## Why GoCurl

Every REST API documents itself with curl. Almost none ship a Go SDK for their long-tail
endpoints. GoCurl makes the curl command *be* the code, so you can:

- **Integrate a new third-party API** by copy-pasting its documented curl example.
- **Prototype and build internal tooling** in seconds.
- **Write scripts, CI checks, and API smoke tests** with one syntax for shell and code.
- **Drive HTTP from config** by storing curl commands as data and executing them.

It's at its best wherever HTTP is glue rather than the hot path. For a high-throughput
production client, reach for a hand-tuned `net/http` client — GoCurl gets your
integration working first.

## Installation

As a library:

```bash
go get github.com/maniartech/gocurl
```

As a command-line tool:

```bash
go install github.com/maniartech/gocurl/cmd/gocurl@latest
```

Requires Go 1.23+.

## Usage

### As a library

The primary entry points accept a curl command (as a single string or as separate
arguments) and return a standard `*http.Response`:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

func main() {
    ctx := context.Background()

    // Separate arguments (each is one token, like os.Args).
    resp, err := gocurl.Curl(ctx,
        "-H", "Accept: application/vnd.github+json",
        "https://api.github.com/repos/golang/go",
    )
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    fmt.Println(resp.StatusCode)
}
```

Convenience helpers read the body for you:

```go
// Body as a string (plus the response).
body, resp, err := gocurl.CurlString(ctx, "https://api.github.com/repos/golang/go")

// Decode JSON directly into a struct.
var repo struct {
    FullName string `json:"full_name"`
    Stars    int    `json:"stargazers_count"`
}
_, err = gocurl.CurlJSON(ctx, &repo, "https://api.github.com/repos/golang/go")

// Stream a download to a file.
n, resp, err := gocurl.CurlDownload(ctx, "go.tar.gz",
    "https://go.dev/dl/go1.23.0.linux-amd64.tar.gz")
```

`CurlBytes` is also available for raw `[]byte` bodies.

### Reusable client

For repeated calls, create a `Client` once and reuse it — it pools connections and carries
shared configuration (timeouts, retries, auth, TLS, observability). Functional options
configure it:

```go
client, err := gocurl.New(
    gocurl.WithTimeout(10*time.Second),
    gocurl.WithRetryAttempts(3),
    gocurl.WithUserAgent("myapp/1.0"),
)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Run a curl command directly...
resp, err := client.Curl(ctx, "curl https://api.github.com/repos/golang/go")

// ...or prepare once and execute it many times (safe across goroutines).
req, err := client.Prepare("curl -H 'Accept: application/json' https://api.example.com/v1/items")
if err != nil {
    log.Fatal(err)
}
resp, err = client.Do(ctx, req)
```

### Variable substitution

By default, environment variables (`$VAR` and `${VAR}`) are expanded automatically:

```go
resp, err := gocurl.Curl(ctx,
    "-H", "Authorization: Bearer $GITHUB_TOKEN",
    "https://api.github.com/user",
)
```

For explicit, testable control — and to avoid pulling in the process environment — pass a
`Variables` map and use the `WithVars` entry points:

```go
vars := gocurl.Variables{"token": myToken}
resp, err := gocurl.CurlWithVars(ctx, vars,
    "-H", "Authorization: Bearer ${token}",
    "https://api.github.com/user",
)
```

### Command-line interface

The CLI uses the same curl syntax as the library:

```bash
gocurl -H "Authorization: Bearer $GITHUB_TOKEN" https://api.github.com/user
gocurl -X POST -d "name=value" https://httpbin.org/post
gocurl -o repo.json https://api.github.com/repos/golang/go
```

Run `gocurl` with no arguments for usage help.

## Built for real integrations

GoCurl is more than a parser — the reusable `Client` wires in the cross-cutting concerns
real API integrations need. Everything below is opt-in via functional options on
`gocurl.New(...)`:

- **Resilience** — idempotency-aware retries with backoff (`WithRetry`, `WithRetryAttempts`,
  `WithRetryBudget`), a circuit breaker (`WithCircuitBreaker`), and a client-side rate
  limiter (`WithRateLimit`).
- **Observability** — pluggable tracing, metrics, structured logging, and lifecycle hooks
  (`WithTracer`, `WithMetrics`, `WithLogger`, `WithHooks`), with ready-made OpenTelemetry
  and Prometheus adapters in [`observability/otel`](observability/otel) and
  [`observability/prometheus`](observability/prometheus).
- **Security** — an opt-in SSRF guard (`WithSSRFGuard`), automatic redaction of secrets in
  errors and verbose output, a fail-closed policy for plaintext auth, and TLS certificate
  pinning.
- **Typed errors** — every failure is a classifiable `*GocurlError`. Branch with
  `errors.Is(err, gocurl.ErrTimeout)` (also `ErrConnect`, `ErrTLS`, `ErrSSRFBlocked`,
  `ErrCircuitOpen`, …) or `gocurl.KindOf(err)`; `gocurl.IsRetryable(err)` reports whether a
  retry could help.
- **Streaming & limits** — responses stream by default (the library never buffers the full
  body or writes to stdout), with an optional response body-size cap to bound memory.

## Supported curl features

GoCurl targets the HTTP/HTTPS flags that appear in real API documentation, including:
HTTP methods (`-X`), headers (`-H`), data/body (`-d`), form and file upload (`-F`),
basic and bearer auth (`-u`), output to file (`-o`), TLS options (`--cert`, `--key`,
`--cacert`, `-k`), proxies (`-x`, including SOCKS5), and compression (`--compressed`).

It deliberately does **not** implement curl's non-HTTP protocols (FTP, SMTP, etc.) or
flags that don't map to HTTP API usage. Flag coverage is expanding — see [VISION.md](VISION.md).

## Contributing

Contributions are welcome. The most valuable work right now is parser correctness
(making real-world curl commands from API docs run verbatim) and test coverage. Please
open an issue to discuss substantial changes before sending a PR. See
[CONTRIBUTING.md](CONTRIBUTING.md) to get started.

## License

MIT — see [LICENSE](LICENSE).
