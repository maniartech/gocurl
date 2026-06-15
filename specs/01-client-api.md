# Client API & Lifecycle

> Status: Draft for review · Spec 01

## Goals

- Introduce a reusable, concurrency-safe `Client` as the primary execution surface, created
  with `New(opts ...Option) (*Client, error)`, holding immutable config, a pooled transport,
  and a middleware chain.
- Realize **PARSE ONCE, EXECUTE MANY**: `client.Prepare(command string) (*Request, error)`
  parses a curl command exactly once into an immutable prepared `Request`; `client.Do(ctx, req)`
  executes it any number of times over reused connections.
- Define the full v0 API surface: functional `Option`s, `Prepare`, `Do`, convenience methods
  (`Curl`/`CurlString`/`CurlJSON`/`CurlBytes`/`CurlDownload`), and `Close()`.
- Specify how the existing package-level `Curl*` functions in `api.go` become thin wrappers
  over a lazily-initialized default `Client`, preserving 100% back-compat.
- Specify concurrency-safety guarantees, safe `Client` sharing, and how per-call overrides
  (context, variables, header tweaks) apply to an immutable `Request` **without data races**.

## Non-goals

- Replacing `net/http`. The engine remains `net/http` (+ `golang.org/x/net/http2`); the `Client`
  wraps it. HTTP/3 is out of scope (future add-on).
- Specifying the resilience engine (retries, circuit breaker, rate limiter), the observability
  interfaces (tracer/metrics/logger), the SSRF guard, the redirect policy internals, or the
  error-classification taxonomy. Those are owned by later specs; this spec only fixes the
  `Option` names and the seams (middleware chain, lifecycle hooks) they plug into.
- Changing curl parsing semantics. `Prepare` reuses the existing parse pipeline
  (`preprocessMultilineCommand` → `tokenizer` → `expandEnvInTokens` →
  `convertTokensToRequestOptions`) verbatim.
- No performance claims. The target is parity with a well-tuned `net/http` client; the win is
  ergonomics, connection reuse, and predictability.

## Design

### Surface overview

Today, every package-level helper in `api.go` (e.g. `CurlCommand`) re-runs the full parse and
calls `executeOpts` → `doRequest` (`process.go`), which builds an `*http.Client` per call via
`CreateHTTPClient` and reuses transports through the global `transportCache` in `clientpool.go`.
Parsing is cheap but repeated; the connection pool is shared but anonymous (keyed by config
string). The `Client` makes both explicit: parse is hoisted into `Prepare`, and the pool is
owned by a named, closeable `Client` instead of a process-global map.

```go
package gocurl

// Client is a reusable, concurrency-safe HTTP client. A zero Client is NOT valid;
// always construct via New. Safe for use by many goroutines once constructed.
// Construct one per logical upstream (or one per process) and share it.
type Client struct {
    // unexported: immutable resolved config, owned *http.Transport (pool),
    // composed middleware chain, observability hooks. No exported fields.
}

// New constructs a Client from functional options. Options are applied in order;
// the first option to return an error aborts construction (fail fast, no partial Client).
// With no options, New returns a Client whose defaults match the current package-level
// behavior (see "Default configuration").
func New(opts ...Option) (*Client, error)

// Option mutates an internal, in-construction config. Never exposed after New returns.
type Option func(*config) error

// Request is an immutable, prepared request template — the "parse once" artifact.
// All fields are unexported; a Request is produced only by Prepare or NewRequest and
// is safe to share across goroutines and reuse across Do calls. Per-call changes go
// through With* methods, which return a shallow clone and never mutate the receiver.
type Request struct {
    // unexported: resolved *options.RequestOptions snapshot, source command (sanitized),
    // a body factory (func() (io.ReadCloser, error)) so each Do/attempt gets a fresh body.
}
```

### Construction options

These names are LOCKED by the project brief; each maps onto an existing `options.RequestOptions`
field or a later-spec seam. Options set `Client`-wide defaults; a `Request` may narrow them.

```go
func WithTimeout(d time.Duration) Option          // → opts.Timeout (overall)
func WithConnectTimeout(d time.Duration) Option    // → opts.ConnectTimeout
func WithRetry(p RetryPolicy) Option               // resilience engine (Spec 04)
func WithProxy(url string) Option                  // → opts.Proxy (+ proxy/ transport)
func WithTLS(cfg TLSOptions) Option                // → LoadTLSConfig inputs (security.go)
func WithTransport(rt http.RoundTripper) Option    // BYO transport; disables internal pool
func WithMiddleware(mw ...Middleware) Option       // appends to the chain (Spec 03)
func WithTracer(t Tracer) Option                   // observability (Spec 05)
func WithMetrics(m Metrics) Option                 // observability (Spec 05)
func WithLogger(l Logger) Option                   // structured logging (Spec 05)
func WithRedirectPolicy(p RedirectPolicy) Option   // → redirectPolicy seam (process.go)
func WithSSRFGuard(g SSRFGuard) Option             // security middleware (Spec 06)
func WithMaxConnsPerHost(n int) Option             // → Transport.MaxConnsPerHost
```

The types named in later-spec options (`RetryPolicy`, `TLSOptions`, `Middleware`, `Tracer`,
`Metrics`, `Logger`, `RedirectPolicy`, `SSRFGuard`) are declared in their owning specs; this
spec only fixes the constructor names and that each returns `Option`.

### Prepare, NewRequest, and per-call overrides

```go
// Prepare parses a curl command ONCE into an immutable Request bound to no client state
// beyond the parse. Env vars ($VAR/${VAR}) are expanded at Prepare time, mirroring
// CurlCommand. Returns a typed parse error (ParseError, see errors.go) on failure.
func (c *Client) Prepare(command string) (*Request, error)

// PrepareWithVars parses with an explicit variable map and NO environment expansion,
// mirroring CurlCommandWithVars (security-critical / hermetic).
func (c *Client) PrepareWithVars(vars Variables, command string) (*Request, error)

// PrepareArgs treats each arg as a pre-tokenized value (like os.Args), mirroring CurlArgs.
func (c *Client) PrepareArgs(args ...string) (*Request, error)

// NewRequest builds a Request programmatically (no curl string) from pre-resolved options.
func (c *Client) NewRequest(opts *options.RequestOptions) (*Request, error)

// --- immutable per-call overrides: each returns a NEW *Request, never mutates receiver ---
func (r *Request) WithContext(ctx context.Context) *Request        // also implicit in Do(ctx,...)
func (r *Request) WithHeader(key, value string) *Request           // Set (replace)
func (r *Request) AddHeader(key, value string) *Request            // Add (append)
func (r *Request) WithQueryParam(key, value string) *Request
func (r *Request) WithVars(vars Variables) *Request                // re-expand $VARs on a re-parse snapshot
func (r *Request) WithBody(body io.Reader) *Request                // replaces the body factory
func (r *Request) Clone() *Request                                  // explicit deep clone
```

`Prepare` does the same work the package functions do inline today, but stores the resulting
`*options.RequestOptions` (deep-cloned via `options.Clone()`) inside the `Request` so it is never
re-parsed. `Do` consumes that snapshot.

### Do and convenience methods

```go
// Do executes a prepared Request and returns the LIVE, streamed *http.Response. The caller
// owns resp.Body and MUST Close it. ctx overrides any context carried by With* overrides
// and drives cancellation/deadline (per the existing determineClientTimeout contract).
func (c *Client) Do(ctx context.Context, req *Request) (*http.Response, error)

// Convenience methods mirror the package funcs but accept either a *Request (preferred,
// parse-once) or a command string (parses each call, like the package funcs).
func (c *Client) Curl(ctx context.Context, command ...string) (*http.Response, error)
func (c *Client) CurlString(ctx context.Context, command ...string) (string, *http.Response, error)
func (c *Client) CurlBytes(ctx context.Context, command ...string) ([]byte, *http.Response, error)
func (c *Client) CurlJSON(ctx context.Context, v any, command ...string) (*http.Response, error)
func (c *Client) CurlDownload(ctx context.Context, filepath string, command ...string) (int64, *http.Response, error)

// DoString / DoBytes / DoJSON / DoDownload are the *Request-typed equivalents that read/close
// the body for you (the parse-once hot path).
func (c *Client) DoString(ctx context.Context, req *Request) (string, *http.Response, error)
func (c *Client) DoJSON(ctx context.Context, v any, req *Request) (*http.Response, error)
// ... DoBytes, DoDownload analogous.
```

`Do` reuses the existing execution pipeline (`doRequest` body: validate → build client/transport →
`CreateRequest` → middleware → `executeWithRetries` → verbose → decompress) but sourced from the
`Client`'s owned transport rather than the global cache, and from the `Request`'s pre-parsed
snapshot rather than a fresh parse. The library-correct streaming contract from `executeOpts`
(never buffer, never write to stdout, honor `ResponseBodyLimit` via `newLimitedBody`) is preserved.

### Lifecycle: Close and the default client

```go
// Close releases the Client's idle connections (Transport.CloseIdleConnections) and marks the
// Client unusable. Idempotent. After Close, Do/Curl* return ErrClientClosed. In-flight requests
// are NOT aborted; cancel them via context. Close does NOT touch a BYO WithTransport transport's
// shared connections beyond CloseIdleConnections.
func (c *Client) Close() error

var ErrClientClosed = errors.New("gocurl: client is closed")
```

The package-level `Curl*` functions become thin wrappers over a lazily-initialized, never-closed
default `Client` (a `sync.Once`-guarded singleton with default options), so existing callers keep
working unchanged:

```go
func Curl(ctx context.Context, command ...string) (*http.Response, error) {
    return defaultClient().Curl(ctx, command...)
}
```

The current process-global `transportCache` in `clientpool.go` is retained ONLY as the default
client's pool implementation (it is the default client's transport), so package-level callers keep
connection reuse; explicitly constructed clients own their own transport and do not share that map.

## Behavior & edge cases

- **Immutability / no data races.** A `*Request` is read-only after `Prepare`/`NewRequest`. `With*`
  methods deep-clone the underlying `*options.RequestOptions` (using `options.Clone()`, which clones
  `Headers`, `Form`, `QueryParams`, `BasicAuth`, `FileUpload`, `RetryConfig`) and return a new
  `*Request`; the receiver is never mutated. This fixes the documented hazard in
  `options.RequestOptions` ("UNSAFE for concurrent writes" to map fields) by making the public path
  copy-on-write. Sharing one `*Request` across goroutines that only call `Do` is safe.
- **Body re-use across attempts/calls.** Bodies are produced by a stored factory so each `Do` (and
  each retry attempt) gets a fresh reader. This generalizes the current `bufferRequestBody`/
  `cloneRequest` logic in `retry.go`: a string/`-d` body becomes a buffered factory; a `WithBody`
  stream is buffered on first `Do` so the `Request` stays replayable (documented memory cost), or
  the caller may pass a `func() io.ReadCloser` to avoid buffering.
- **Context precedence.** `Do(ctx, ...)` ctx wins over any `WithContext` ctx. The existing
  `determineClientTimeout` rule holds: if ctx has a deadline, `client.Timeout` is left 0 to avoid
  nested timeouts; otherwise `opts.Timeout` (from `WithTimeout`) applies. Cancellation through the
  retry loop (`checkContextDuringRetry`, `sleepWithContext`) is unchanged.
- **Concurrency safety of `Client`.** After `New` returns, the resolved config and transport are
  immutable; `Client` holds no per-request mutable state. The underlying `*http.Transport` is
  safe for concurrent use (as `clientpool.go` already documents). Therefore one `Client` serves
  unbounded concurrent `Do` calls. There is no `Client`-level mutex on the hot path.
- **`Close` semantics.** Double `Close` is a no-op returning nil. `Do` after `Close` returns
  `ErrClientClosed` and performs no network I/O. `WithTransport` (BYO) transports are not owned;
  `Close` calls `CloseIdleConnections` if implemented but never assumes ownership.
- **Convenience methods with a command string** parse on every call (same cost as today's package
  funcs). Callers wanting parse-once use `Prepare` + `Do*`. Both are first-class.
- **Error typing.** `Prepare` returns `ParseError(command, err)`; `Do` returns `RequestError`/
  `ResponseError`/`RetryError` from `errors.go`, all `errors.Is/As`-friendly via `GocurlError.Unwrap`.
  Secret redaction in those errors (`sanitizeCommand`/`sanitizeURL`) is preserved.
- **Empty / nil inputs.** `New(nil...)` is valid (defaults). `Prepare("")` returns a parse error.
  `Do(ctx, nil)` returns a non-nil error, not a panic. A nil ctx is treated as
  `context.Background()` for parity with the current package funcs.
- **CustomClient / `WithTransport` interplay.** `WithTransport(rt)` sets the round tripper for the
  client (analogous to `opts.CustomClient`/`CreateHTTPClient`). The transport cache is bypassed; the
  client does not own connection lifetime beyond `CloseIdleConnections`.

## Acceptance criteria / Definition of Done

- [ ] `New(opts ...Option) (*Client, error)` exists; applies options in order; aborts on first error
      with no leaked/partial `Client`; `New()` with zero options matches current default behavior.
- [ ] All twelve locked `With*` option constructors exist with the exact signatures above and each
      returns `Option`; options whose types are owned by later specs compile against placeholder
      types and are documented as such.
- [ ] `client.Prepare`, `PrepareWithVars`, `PrepareArgs`, and `NewRequest` produce an immutable
      `*Request`; parsing happens exactly once (verified by a test asserting `Do` does not re-tokenize,
      e.g. via a parse-counter hook or a body-factory call count).
- [ ] `Request.With*`/`AddHeader`/`Clone` never mutate the receiver; a `-race` test runs N goroutines
      doing `Do` on the same base `*Request` while others derive `With*` copies, with zero races.
- [ ] `client.Do` returns the live streamed body, never writes to stdout, honors `ResponseBodyLimit`,
      and respects ctx cancellation/deadline identically to `executeOpts`/`doRequest` today.
- [ ] Convenience methods (`Curl`, `CurlString`, `CurlBytes`, `CurlJSON`, `CurlDownload`) and the
      `Do*` variants exist on `*Client` and read+close the body where they own it.
- [ ] `Close()` is idempotent, releases idle connections, makes subsequent `Do`/`Curl*` return
      `ErrClientClosed`, and does not abort in-flight requests.
- [ ] Package-level `Curl*` functions are reimplemented as wrappers over a lazily-initialized default
      `Client`; the existing blackbox `tests/` API + CLI subprocess suites pass unchanged.
- [ ] Connection reuse across calls is demonstrated by a test (same `Client`, multiple `Do`, asserting
      a single TCP/TLS handshake via an `httptest` server connection counter).
- [ ] `-race` clean across the full suite; no new global mutable state beyond the retained default
      client and its transport.

## Dependencies

- None of the other numbered specs are required to land this keystone; this spec DEFINES the seams
  the others plug into. Specifically it pins names consumed by: **03** (Middleware:
  `Middleware`/`Handler` chain, `WithMiddleware`), **04** (Resilience: `RetryPolicy`, `WithRetry`),
  **05** (Observability: `Tracer`/`Metrics`/`Logger` + `WithTracer`/`WithMetrics`/`WithLogger`),
  **06** (Security/SSRF + TLS/redirect: `SSRFGuard`/`TLSOptions`/`RedirectPolicy`).

## Open questions / decisions to confirm in review

- **Middleware shape vs. current code.** The brief locks `Handler func(*http.Request) (*http.Response, error)`
  and `Middleware func(next Handler) Handler` (round-trip wrapping). The existing
  `middlewares.MiddlewareFunc func(*http.Request) (*http.Request, error)` (request-only, applied in
  `ApplyMiddleware`) cannot observe responses. Proposal: keep `MiddlewareFunc` for back-compat by
  adapting it into the new chain, and make the new `Middleware` the v1 surface. Confirm here so Spec 03
  can build on it.
- **`Request` body factory type.** Should `WithBody` accept `io.Reader` (buffered for replay) only, or
  also a `func() (io.ReadCloser, error)` for large/streaming non-replayable bodies? Buffering matches
  today's retry behavior but costs memory for big uploads.
- **Default client lifetime.** The lazily-initialized default `Client` is never `Close`d (process
  lifetime). Acceptable, mirroring `http.DefaultClient`? Or expose `CloseDefault()` for tests?
- **Does `Close` belong on the default client at all?** Calling `Close()` on a user-built client is
  clear; confirm package funcs never expose the default client so it cannot be closed out from under
  callers.
- **`WithMaxConnsPerHost(0)` meaning.** Treat 0 as "unlimited" (net/http default) vs. "use gocurl
  default of 10 per host from `newTransport`"? Recommend: 0 = net/http unlimited; gocurl's 10 applies
  only when the option is unset.
- **Should `Do` accept a command string overload** (for symmetry) or stay strictly `*Request`-typed,
  pushing string callers to the `Curl*` convenience methods? Recommend the latter to keep `Do` the
  single parse-once execute path.
