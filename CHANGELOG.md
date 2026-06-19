# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project aims to
follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) once it reaches
a tagged release.

## [Unreleased]

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
