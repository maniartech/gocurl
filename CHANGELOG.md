# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project aims to
follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) once it reaches
a tagged release.

## [Unreleased]

### Changed — CLI hardening (Milestone 8)
- `cmd/gocurl`'s `main` is refactored to a testable `run(args []string, stdout, stderr io.Writer) int`
  entry point. It is reentrant — flags are parsed on a local `flag.FlagSet`, never the global
  `flag.CommandLine` (this also fixes a latent "flag redefined" panic on a second invocation). In-process
  tests now cover every output mode, raising `cmd/gocurl` coverage from ~45% to 97.3%.
- **Stream discipline (curl-compatible):** the `-v` verbose trace now goes to **stderr** (sensitive
  headers redacted) so the body on **stdout** stays pipe-clean (`gocurl -v url | jq`). The body is
  printed exactly once across every flag combination. `-o` writes the body to a file **verbatim**
  (no JSON pretty-print/reorder, no added newline — pretty-printing is a stdout-only convenience);
  `-w` expansion is explicit data that always goes to stdout, even under `-s` and alongside `-o`
  (the `-s -o /dev/null -w '%{http_code}'` idiom); `%{size_download}` reports the actual bytes
  downloaded; usage on misuse goes to stderr.
- **Argument splitting** recognizes gocurl presentation flags only in the leading prefix (matching the
  documented `gocurl [gocurl options] [curl options] <URL>`) and honors a `--` separator, so a curl
  flag value that looks like a presentation flag (`-d -s url`) is passed through verbatim rather than
  stolen; a value-taking presentation flag with no argument fails fast.
- **Exit codes are derived from the error `Kind`** (Spec 08), not string matching. Parse/tokenize/
  convert failures from `Curl`/`CurlArgs`/`CurlCommand*` and `Client.Prepare*` are now typed
  (`ParseError`/`KindParse`, with credentials in the failing command redacted), so an unknown flag
  exits `2`; a missing/malformed URL is `KindValidation` → `3`; too-many-redirects carries the new
  `ErrTooManyRedirects` sentinel → `47`; `--fail`/`-f` → `22` on HTTP ≥ 400.
- **Redaction widened:** `IsSensitiveHeader` adds a credential-suffix/content heuristic so the open set
  of vendor auth headers (`X-Goog-Api-Key`, `Private-Token`, `X-Vault-Token`, `X-Functions-Key`, …) is
  redacted in verbose/log output, and the `-u`/`--user`/`-b`/`--cookie` redactors now fire even when
  the flag is the first token.

### Added — security hardening (Milestone 7)
- **Opt-in SSRF guard** (`WithSSRFGuard`, `SSRFPolicy`, `DefaultSSRFPolicy`): blocks the initial
  request and every redirect hop whose host resolves to a loopback, link-local, RFC1918/ULA
  private, the unspecified address (`0.0.0.0`/`::`, loopback-equivalent for routing), or a
  cloud-metadata address (`169.254.169.254`, `fd00:ec2::254`, `metadata.google.internal`, matched
  with a trailing FQDN dot normalized away) unless allow-listed (`AllowHosts`/`AllowIPs`). A block
  is a typed, non-retryable error (`errors.Is(err, ErrSSRFBlocked)`). Opt-in to preserve the
  paste-any-curl promise; DNS-rebinding between check and dial is a documented residual risk.
- **Runtime input validation on the live path**: `Curl*`/`Do` now enforce method-token validity,
  header count/size caps, a forbidden-header rule (`Host`/`Content-Length`/`Transfer-Encoding`),
  and body/form/query-count caps — previously reachable only via the builder. Streaming bodies are
  exempt from the in-memory body cap.
- **Fail-closed plaintext auth**: BasicAuth or a bearer token over cleartext `http://` is now an
  error by default (scheme matched case-insensitively, so `HTTP://` cannot slip past); override with
  `WithAllowInsecureAuth(true)` or `GOCURL_ALLOW_INSECURE_AUTH=1`.
- `WithTLSConfig(*tls.Config)` merges a caller TLS config **over** `SecureDefaults()` via
  `LoadTLSConfig`: the caller's fields win, but any field left zero (e.g. a config that only sets
  `RootCAs`) still inherits the TLS 1.2 floor and the curated cipher list.
- **Redaction coverage widened** (one path: `RedactURL`/`RedactCommand`/`IsSensitiveHeader`, also
  feeding logs, spans, and verbose output): every occurrence of a repeated sensitive query
  parameter is redacted (not just the first); the `-u`/`--user` (and `=`-separated) and
  `-b`/`--cookie` flag forms are redacted, not only the header forms; and `x-amz-security-token`
  (AWS STS) and `x-csrf-token` join the sensitive-header set.

### Changed
- **TLS pinning hardened**: a pinned request now verifies the certificate chain **and** the pin
  (a wrong pin against a valid chain fails closed); pinning no longer silently sets
  `InsecureSkipVerify`. Only `-k`/`--insecure` makes the pin the sole check.
- **Behavior changes (security, may affect existing code):** setting `Host`/`Content-Length`/
  `Transfer-Encoding` via `-H` is now rejected; BasicAuth/bearer over `http://` fails closed
  (see overrides above). `validateMethod` now accepts any valid HTTP method token (custom/WebDAV
  methods are allowed; previously a fixed allow-list).
- Proxy auth is sent whenever a username is present (`-x http://user@proxy`), not only when a
  password is also set. Proxy credentials are now encoded via the `net/url` userinfo helpers
  (spaces and special characters round-trip correctly; the HTTP-proxy and CONNECT paths agree), and
  a malformed proxy address never echoes the credentials in its error.

### Added — observability: tracing, metrics, logging, hooks (Milestone 6)
- Vendor-neutral, dependency-free primitives in the core package: `Tracer`/`Span`, `Metrics`,
  `Logger`/`Level`/`Field`, and lifecycle `Hooks` (`OnRequest`/`OnRetry`/`OnResponse`/`OnError`),
  wired via `WithTracer`/`WithMetrics`/`WithLogger`/`WithHooks`/`WithRequestIDFunc`. Error
  classification reuses the M4 `Kind` taxonomy (no parallel enum).
- An internal instrumentation middleware brackets the whole logical request (one span, one
  latency observation, one request count; one `IncRetry` per retry beyond the first; balanced
  in-flight gauge). It is the outermost middleware (so its span/latency also cover circuit-breaker
  fast-fails and rate-limiter waits) and is installed **only when a sink/hook is configured** —
  the disabled path adds no allocations (`BenchmarkDo_NoObservability` vs `…FullObservability`).
- A panic in any user-supplied sink/hook is recovered and can never take down a request.
- Request-id is kept (existing `X-Request-ID`/`opts.RequestID`) or generated once via
  `WithRequestIDFunc`, preserved across retry clones, and surfaced as a span attribute and log
  field. Redaction is unified on `errors.go`'s `IsSensitiveHeader` (now including
  `x-auth-token`/`auth-token`); `verbose.go`'s duplicate list was removed.
- New adapter modules (kept out of the core's dependency graph): `observability/prometheus`
  (registers `gocurl_requests_total`, `gocurl_in_flight`, `gocurl_request_duration_seconds`,
  `gocurl_retries_total`, `gocurl_errors_total{kind}`) and `observability/otel` (`Tracer`/`Span`
  + a W3C `traceparent` propagation middleware).

### Changed
- `NewRequest` now skips nil `RequestOption`s (matching `New`'s tolerance of nil `Option`s).

### Added — resilience: retries, circuit breaker, rate limiter (Milestone 5)
- Idempotency-aware `RetryPolicy` (`WithRetry` / `WithRetryAttempts`, per-request
  `Request.WithRetryPolicy`): retries only the idempotent method set
  (GET/HEAD/OPTIONS/TRACE/PUT/DELETE) or a request carrying an `Idempotency-Key` header — a
  POST is **not** retried by default. `Backoff` (`ExponentialJitter` with equal jitter,
  `ConstantBackoff`), `MaxElapsed` and per-attempt deadlines, `Retry-After` honoring
  (delta-seconds + HTTP-date), and a `RetryBudget` (`WithRetryBudget`) to prevent retry storms.
- Body replay prefers `GetBody`; otherwise the body is buffered up to `WithMaxReplayBytes`
  (default 1 MiB) — larger bodies are sent once and become non-retryable. Discarded retry
  attempts now drain **and** close their body so pooled keep-alive connections are reused.
- `CircuitBreaker` / `WithCircuitBreaker`: per-host rolling-window breaker that fast-fails with
  `ErrCircuitOpen` (non-retryable), half-opens after a timeout, and counts only the final
  outcome of each request (never individual retry attempts).
- `RateLimiter` / `WithRateLimit`: a zero-dependency client-side token bucket behind a
  pluggable `Limiter` interface. Client middleware composes as
  `circuit breaker → rate limiter → user middleware → retry loop → transport`.

### Changed
- **Behavior change (new API only):** the new `WithRetry` path is idempotency-aware, so a POST
  is not retried unless you set `AllowMethods` or send an `Idempotency-Key`. The legacy
  `options.RetryConfig` and the `--retry` flag remain **method-agnostic** (a POST with `--retry`
  is still retried), preserving existing behavior through v0.x.

### Added — typed error model (Milestone 4)
- `Kind` taxonomy on `GocurlError` (`KindParse`, `KindValidation`, `KindConnect`, `KindTLS`,
  `KindTimeout`, `KindCanceled`, `KindServerStatus`, `KindRetryExhausted`, `KindBodyRead`) plus
  the classification triple `Timeout()`/`Temporary()`/`Retryable()` (all nil-safe).
- `errors.Is`/`errors.As` support: sentinels `ErrParse … ErrBodyRead` (e.g.
  `errors.Is(err, gocurl.ErrTimeout)`), and `errors.Is(err, context.DeadlineExceeded)` still
  resolves through the wrap chain. Package helpers `KindOf`, `IsTimeout`, `IsTemporary`,
  `IsRetryable`.
- New constructors `ServerStatusError`/`BodyReadError`/`ConnectError`/`TLSError`/`TimeoutError`/
  `CanceledError`; `RequestError` now classifies the wrapped transport error
  (`net.OpError`→connect, `*tls.CertificateVerificationError`/pin mismatch→tls,
  `context.DeadlineExceeded`→timeout, `context.Canceled`→canceled). Classification is wired into
  the live request path (`doRequest`, `Client.Do`) and retry exhaustion (`KindRetryExhausted`
  chaining the last attempt's classified error).
- Error strings get a final-pass scrub backstop so credentials embedded in a wrapped stdlib
  error (userinfo passwords, `api_key`/`token`/… query params) never leak.

### Added — status-code policy (Milestone 4)
- Default contract formalized and tested: a non-2xx response is **not** an error; the live
  `*http.Response` is returned. Opt in to curl `-f`/`--fail` behavior with `WithFailOnStatus`
  (Client) or `options.FailOnError` / the `-f`/`--fail` flag — a `>=400` response then yields a
  `ServerStatusError` while the response is still returned so the caller can read the error
  body. The convenience wrappers (`CurlString`/`CurlBytes`/`CurlJSON`/`CurlDownload`) return the
  response alongside the error.
- CLI: exit codes are derived from `Kind` (22 server-status, 28 timeout, 7 connect, 35 TLS,
  2 parse, 3 validation), replacing brittle string matching, with a legacy string fallback;
  `-f`/`--fail` is honored.

### Added — body model & streaming (Milestone 3)
- `BodySource` interface (in `options`) + `BytesBody`/`StringBody`/`FileBody`/`ReaderBody`/
  `MultipartBody` constructors. Requests can now stream bodies instead of buffering them.
- `CreateRequest` honors a streaming `BodyStream` (sets `Content-Length` for known sizes and
  a `GetBody` for rewindable sources); the `-T` upload flag now streams from disk.
- `MultipartBody` streams via a cancellation-safe `io.Pipe` (closing the body unblocks the
  writer goroutine) and supplies its own `multipart/form-data` Content-Type.
- `Request.WithBodySource`/`WithBodyFile` builders and the `Stream(...)` request option.

### Changed
- `executeWithRetries` only buffers the request body when retries are enabled and the body
  has no `GetBody`; `executeAttempt` replays via `GetBody` when available. The default
  (no-retry) client now streams uploads straight through.

### Added — transport & connection management (Milestone 2)
- Config-driven, per-Client owned transport with tunable pooling and the timeout taxonomy:
  `WithMaxIdleConns`, `WithMaxIdleConnsPerHost`, `WithMaxConnsPerHost` (0 = unlimited),
  `WithIdleConnTimeout`, `WithTLSHandshakeTimeout`, `WithResponseHeaderTimeout`,
  `WithExpectContinueTimeout` (connect timeout via `WithConnectTimeout`'s dialer).
- `RedirectPolicy{Follow, Max, Allow}` + `WithRedirectPolicy` — the `Allow` hook authorizes
  each redirect (works on a shared Client via the request-context redirect seam; it is the
  seam the SSRF guard will use).
- `WithHTTP2(bool)` (default on) and `WithHTTP2Only(allowCleartext bool)` for h2/h2c.
  HTTP/3 (QUIC) is intentionally out of scope.

### Fixed
- `retryLoop` no longer drops the error (or closes the returned response body) once retries
  are exhausted — it propagates the last error and preserves the response.

### Added — reusable Client (production foundation, Milestone 1)
- `Client` (`New(opts ...Option)`) — a reusable, concurrency-safe HTTP client that owns its
  pooled transport, for the parse-once/execute-many model. `Prepare`/`PrepareNoEnv`/
  `PrepareWithVars` parse a curl command once into an immutable `Request`; `Do(ctx, *Request)`
  executes it (streaming the live body), with `Curl`/`CurlString`/`CurlBytes`/`CurlJSON`/
  `CurlDownload` convenience methods, `Close()`, and `Shutdown(ctx)` (drains in-flight).
- Functional options: `WithTimeout`, `WithConnectTimeout`, `WithFollowRedirects`,
  `WithMaxRedirects`, `WithProxy`, `WithInsecure`, `WithUserAgent`, `WithDefaultHeader`,
  `WithCookieFile`, `WithMiddleware`, `WithTransport`.
- `Handler`/`Middleware` composition model (`middleware.go`) with a `FromMiddlewareFunc`
  adapter for the legacy `middlewares.MiddlewareFunc`.
- Programmatic `NewRequest(method, url, ...RequestOption)` with `Header`/`Query`/`Body`/
  `BodyReader` option constructors and immutable `Request` builders (`WithHeader`,
  `AddHeader`, `WithQuery`, `WithBody`, `WithVars`, `Clone`).
- Per-request redirect policy works on a shared Client (carried via request context).

### Removed
- Deprecated `Request()`/`RequestWithContext()` helpers (the name `Request` is now the
  prepared-request type). Use `Curl`/`CurlWithVars` or a `Client` + `Prepare`/`Do`.

### Changed
- Repositioned the project around its real value: pasting curl commands from API
  docs directly into Go. Rewrote the README and added `VISION.md`; removed
  unsubstantiated "zero-allocation / net/http replacement" claims.
- The high-level `Curl*` functions now stream the live response body and never
  write to `os.Stdout`. `Process` is retained but deprecated.

### Added
- `doc.go` package documentation, `CONTRIBUTING.md`, `SECURITY.md`, this changelog,
  and a CI workflow (gofmt, vet, build, hermetic `-short -race` tests).
- Curl flags: `-G/--get`, `-I/--head`, `-O`, `-T`, `--data-urlencode`,
  `--connect-timeout`, `--retry`, `--noproxy`, `--url`; `-d @file` now reads files.
- Connection reuse via cached, idle-tuned transports across one-shot calls.
- Black-box test suite (`tests/`) exercising the public API and CLI across
  methods/flags/options, plus whitebox tests for the parser and new features.

### Fixed
- **Parser:** strip surrounding quotes (quoted JSON bodies and header values were
  sent literally); stop fragmenting quoted `$VAR`; accept bare-host URLs
  (default `http://`); preserve userinfo/query in URLs.
- **Security:** `CurlWithVars` no longer leaks environment variables; certificate
  pinning compares the real SHA-256 of the cert; `LoadTLSConfig` is the single
  source of truth and applies `--tlsv1.x`/`--tls-max`/`--ciphers` on every request;
  the CLI redacts secrets in verbose output.
- **Redirects:** `-L` now follows redirects (default max 30) instead of failing on
  the first hop.
- **Compression:** real `deflate` decompression (zlib-wrapped and raw).
- **Proxy:** SOCKS5 no longer fails every dial when no connect timeout is set.

### Hardening
- Isolated the `book2/` example programs and `scripts/` helpers into their own Go
  modules so `go build/vet/test ./...` covers only the library and CLI.
- Removed stale `*.go.old` files; normalized formatting (`gofmt`, LF line endings).
