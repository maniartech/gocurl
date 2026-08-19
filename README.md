# GoCurl

**Missing a Go SDK?** Your favorite API provider shipped clients for Python, JavaScript,
and Ruby — and for Go, just a `curl` example and good luck. Don't worry. GoCurl makes that
curl command *be* your Go code:

```go
resp, err := gocurl.Curl(ctx, `
  curl https://api.github.com/repos/golang/go
`)
```

Test it in your shell, paste it into your program — the exact same command, no translation,
no guesswork.

## No SDK? No problem — here's OpenAI

Copy the curl command straight out of the provider's docs. Keep the `$OPENAI_API_KEY`
exactly as written — GoCurl expands it from the environment at execution, just like your
shell does, so the key never touches your source:

```go
var out struct {
    Choices []struct {
        Message struct {
            Content string `json:"content"`
        } `json:"message"`
    } `json:"choices"`
}

resp, err := gocurl.CurlJSON(ctx, &out, `
    curl https://api.openai.com/v1/chat/completions \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $OPENAI_API_KEY" \
      -d '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Say hello."}]}'
`)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

fmt.Println(out.Choices[0].Message.Content)
```

One raw string, multi-line and backslash-continued exactly as the docs print it, decoded
straight into your own struct. That is the whole integration — no SDK, no hand-translation
into `http.NewRequest`. (Verified end-to-end by `TestReadmeExample_OpenAIStyleRawString`;
for keys you'd rather not read from the process environment, see
[Variable substitution](#variable-substitution).)

GoCurl is a **production-grade**, curl-ergonomic HTTP client for Go, built on `net/http`,
with a CLI that shares the exact same syntax. It removes the tax every Go developer pays
when integrating a new API — mentally compiling a curl snippet from the docs into
`http.NewRequest`, headers, body encoding, and auth — and backs it with the retries,
circuit breaking, observability, SSRF protection, secret redaction, and typed errors that
real integrations need in production (`TestFault_OverallRetryBudget`,
`TestFault_NoSecretLeakOnFailurePaths`, `TestExecutionPathFeatureMatrix`).

> **Persuasion by example, not by marketing.** Every performance and reliability claim in
> this repo cites a test or benchmark you can run yourself — enforced by an automated
> doc-lint (`TestDocHonestyLint`), not left to good intentions. **Every Go example on this
> page is executed by a test** (`TestReadmeExample_*`) and every API it names is checked
> against the locked public surface (`TestReadmeExamples_SymbolsExist`) — so what you copy is
> what we run. That is the motto, and it runs through the README, VISION, docs, and book alike.

> See [VISION.md](VISION.md) for what we're building and why.

## Project status

**Production-grade, released at `v0.5.0`.** The engine is hardened and tested — race-clean,
fuzzed, with a coverage gate in CI, streaming bodies, connection pooling, and the
resilience/observability/security stack below. The race and resource claims are exercised by
`TestFault_NoGoroutineLeakUnderStorm` and `TestClient_Soak`.

The `v0` is about the *contract*, not the quality. Under semver a v1 major promises the API
will not change, and in Go that promise is costly to revise — `v1 → v2` forces every importer
to rewrite their import path to `/v2`, while `v0.x → v1.0` costs them nothing. We would rather
take API feedback from real integrations first. So: pin a version, check the
[CHANGELOG](CHANGELOG.md) when upgrading, and expect the API to settle rather than churn.
Feedback and contributions are very welcome — see [docs/v1-readiness.md](docs/v1-readiness.md)
for exactly what stands between here and v1.0.

### Proven, not promised

"Production-grade" and "mission-critical" are claims we back with tests you can run, not
adjectives. A two-tier fault-injection harness deliberately breaks the network and asserts
gocurl does the right thing:

- **Bounded retries** — `WithTimeout` bounds the *whole* operation including backoff, not
  just one attempt (`TestFault_OverallRetryBudget`).
- **Idempotency-aware retries**, including HTTP/2 `GOAWAY`/`RST_STREAM` (`TestFault_H2ErrorsRetried`).
- **Graceful shutdown** never truncates a live stream and is never wedged by a panicking
  middleware (`TestFault_ShutdownWaitsForOpenBody`, `TestFaultT2_PanicMiddlewareDoesNotWedgeShutdown`).
- **Memory bounds** against a decompression bomb from an untrusted server
  (`TestFault_BufferingHelpersBoundedAgainstBomb`).
- **Secrets never leak** on any failure path (`TestFault_NoSecretLeakOnFailurePaths`).
- **No leaks under sustained load** (`TestClient_Soak`); **backpressure without deadlock**
  (`TestClient_Soak_Backpressure`).
- **Wire-parity with real curl**, proven by differential testing (`TestCurlParity_DifferentialVsRealCurl`).

Performance is reported honestly: gocurl targets **parity** with a well-tuned `net/http`
client (proven by `BenchmarkRoundTrip_Gocurl_Prepared` and `TestLatencyDistribution`) and we
publish where it loses. See [docs/operations.md](docs/operations.md),
[docs/benchmarking.md](docs/benchmarking.md), and the
[v1.0-readiness checklist](docs/v1-readiness.md). Every claim above is checked by an automated
honesty doc-lint (`TestDocHonestyLint`): no claim ships without a named, un-skipped test.

### Why gocurl over hand-rolled `net/http`

A `net/http` purist *can* write everything gocurl does — but almost nobody writes it
correctly, per service, and keeps it correct. Because gocurl receives the **curl recipe**, it
knows your intent and wires the right execution pipeline around it:

| Concern | Hand-rolled `net/http` | gocurl |
|---|---|---|
| Overall timeout across retries | `Client.Timeout` is **per-attempt**; easy to ship a retry amplifier | `WithTimeout` bounds the whole loop, backoff clamped to remaining budget |
| Retry safety | Retry-everything (replays a non-idempotent POST) or retry-nothing | Idempotency-aware; only safe methods + retryable outcomes |
| Error decisions | `err != nil`, then guess | Classified `Kind` (`errors.As` into h2/DNS/TLS errors) |
| Secret hygiene | One careless `fmt.Errorf("%v", url)` leaks a token | Redaction on every error/log/span path, build-gated |
| Untrusted-server memory | `DisableCompression` quietly removes net/http's guard | Bounded buffered reads + 1 MiB header cap |

You paste a curl command from the API docs and get this pipeline for free — the
[operations guide](docs/operations.md) shows how to tune it.

## Why GoCurl

Every REST API documents itself with curl. Almost none ship a Go SDK for their long-tail
endpoints. GoCurl makes the curl command *be* the code, so you can:

- **Integrate a third-party API** by copy-pasting its documented curl example — then run it
  in production with retries, timeouts, tracing, and metrics around it.
- **Operate service-to-service HTTP** through a pooled `Client` with circuit breaking, rate
  limiting, and SSRF protection.
- **Write scripts, CI checks, and API smoke tests** with one syntax for shell and code.
- **Drive HTTP from config** by storing curl commands as data and executing them.

Built on `net/http` and organized around **parse once, execute many** — `Prepare` a request
once, then `Do` it repeatedly over a pooled `Client` — so the per-request overhead above
hand-written `net/http` is small and constant. The goal is parity with a well-tuned
`net/http` client plus the ergonomics and reliability above; we make no "faster than
net/http" claims.

## Installation

As a library:

```bash
go get github.com/maniartech/gocurl
```

As a command-line tool:

```bash
go install github.com/maniartech/gocurl/cmd/gocurl@latest
```

Requires Go 1.25+.

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
    "https://go.dev/dl/go1.25.0.linux-amd64.tar.gz")
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

### One pipeline under any SDK

When a vendor SDK accepts an `*http.Client`, compose gocurl's exported handler
middleware and adapt it into the SDK's transport. Compose outermost-first; keep
`Retry` innermost so the breaker records the final logical outcome rather than
each attempt:

```go
base := gocurl.HandlerFromRoundTripper(http.DefaultTransport)
limiter := gocurl.NewTokenBucket(20, 40)

h := gocurl.Observe(hooks, metrics, tracer, logger)(
    gocurl.SSRFGuard(gocurl.DefaultSSRFPolicy())(
        gocurl.CircuitBreaker(breakerConfig)(
            gocurl.RateLimiter(limiter)(
                gocurl.Retry(gocurl.DefaultRetryPolicy(3))(base)))))

sdkHTTPClient := &http.Client{
    Transport: gocurl.RoundTripperFromHandler(h),
    Timeout:   30 * time.Second,
}
```

`HandlerFromRoundTripper` preserves the original hostname while pinning SSRF-validated
IPs at dial time. `RoundTripperFromHandler` deliberately applies only the middleware
shown above: the surrounding `http.Client` owns redirects, cookies, and its timeout.
`WithTimeout`, redirect options, defaults, and `WithTransport` configure a gocurl
`Client`; they are not ambient settings for a standalone chain. Legacy request-mutating
middleware can be included with `FromMiddlewareFunc`.

Backoff policies can use `ConstantBackoff` or `ExponentialJitter`. Programmatic native
requests can be created with `NewRequest` and executed through a reusable `Client`.

| Feature | `Client.Do` | Injected handler chain | `Curl*` / `Execute` |
|---|---:|---:|---:|
| Retry | configured with `WithRetry` | explicit `Retry` | curl/legacy retry options |
| Circuit breaker / rate limiter | configured options | explicit `CircuitBreaker` / `RateLimiter` | unavailable |
| SSRF guard with dial-IP pinning | `WithSSRFGuard` | explicit `SSRFGuard` + adapter | unavailable |
| Hooks / metrics / tracing / logging | `WithHooks` etc. | explicit `Observe` | unavailable |
| Redirects, cookies, defaults | client-managed | surrounding SDK/`http.Client` | curl options |

The one-shot helpers intentionally remain compatibility-oriented and protection-free.
For user-influenced URLs, use `Client.Do` with `WithSSRFGuard`, or build an explicit
injected chain as above.

### Typed request building

Prefer typed, IDE-discoverable configuration over a curl string? Assemble a request with the
options builder and run it with `Execute`:

```go
opts := options.NewRequestOptionsBuilder().
    SetURL("https://api.example.com/v1/items").
    SetMethod("POST").
    AddHeader("Authorization", "Bearer "+token).
    SetBody(`{"name":"widget"}`).
    Build()

resp, err := gocurl.Execute(ctx, opts)
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
`--cacert`, `-k`), proxies (`-x`, including SOCKS5), compression (`--compressed`), and
HTTP version selection (`--http2`, `--http2-only`, `--http1.1`; `--http1.0`/`-0` is
accepted as a best-effort 1.1 pin + `Connection: close`, since Go's `net/http` cannot
emit a true HTTP/1.0 request line).

It deliberately does **not** implement curl's non-HTTP protocols (FTP, SMTP, etc.) or
flags that don't map to HTTP API usage. Flag coverage is expanding — see [VISION.md](VISION.md).

## Contributing

Contributions are welcome. The most valuable work right now is parser correctness
(making real-world curl commands from API docs run verbatim) and test coverage. Please
open an issue to discuss substantial changes before sending a PR. See
[CONTRIBUTING.md](CONTRIBUTING.md) to get started.

## License

MIT — see [LICENSE](LICENSE).
