# API Stability, Versioning & Migration

> Status: Draft for review · Spec 11

## Goals

- Define a **SemVer** policy for `github.com/maniartech/gocurl` and the concrete path from
  the current untagged `v0.x` state (the module `Version` is literally `"dev"`; `git tag`
  shows no tags) to a stable, supportable **v1**.
- Lock the **minimal v1 public surface**: the reusable `Client`, functional `Option`s, the
  prepared `Request`, the `Curl*` helper family, and the small set of supporting types
  (`Variables`, `GocurlError` + classifiers, `RetryPolicy`, observability interfaces).
- Decide, symbol by symbol, what stays exported vs. what moves to `internal/` or gets
  unexported. The current package leaks its execution machinery: `Process`, `Execute`,
  `CreateHTTPClient`, `CreateRequest`, `ApplyMiddleware`, `HandleOutput`, `ValidateOptions`,
  `ArgsToOptions`, and the legacy `Response` wrapper are all exported today (see `api.go`,
  `process.go`, `convert.go`).
- Establish a **deprecation policy and timeline** that keeps existing `Curl*` callers working
  through all of `v0.x` and into `v1`, with mechanical compiler-assisted migration where a
  symbol must move.
- Write a **migration guide** from the one-shot package functions to the reusable `Client`.
- Specify the **release process**: tagging, build-time version stamping (`version.go`'s
  `Version` var set via `-ldflags`), `CHANGELOG.md` discipline (the file already follows
  Keep a Changelog and has an `[Unreleased]` section), and `go vet`/CI gates that protect the
  surface.

## Non-goals

- Designing the `Client`/`Option`/`Request`/middleware APIs themselves — those are owned by
  the resilience, observability, and client-lifecycle specs. This spec only **freezes** their
  signatures and decides their export status.
- Adding new HTTP features, flags, or transports. Stability work ships no behavior changes
  except deprecations and relocations.
- Multi-module splitting (e.g. a separate `gocurl/otel` module). The OpenTelemetry adapter's
  module boundary is an open question, noted below, not a decision here.
- Tooling to auto-rewrite user code beyond standard `// Deprecated:` markers and an optional
  `gofmt -r` recipe in the migration guide.

## Design

### SemVer contract

gocurl is a Go module at major version 0, so `go.mod` import paths carry **no version
suffix**. Under SemVer + the Go module rules:

- **`v0.x.y` (now → v1):** the API is explicitly unstable. We still bump **minor** for additive
  changes and **patch** for fixes, and we *announce* breaking changes in `CHANGELOG.md`, but
  `v0` makes no compatibility promise the toolchain enforces. This is the window in which we
  unexport internals and relocate symbols.
- **`v1.0.0`:** the exported surface defined in this spec becomes a **compatibility promise**.
  After v1, no exported identifier is removed or changed incompatibly without a major bump
  (`v2`, which *would* require the `/v2` path suffix). Additive-only changes ship as minors.
- The **public surface** subject to the promise is everything exported from the root
  `gocurl` package plus the `cmd/gocurl` CLI flags. The `options`, `tokenizer`, `proxy`,
  and `middlewares` sub-packages are **not** part of the v1 promise unless a type from them
  appears in a v1 root signature (see "leakage" below).

### The v1 public surface (the keep-list)

The root `gocurl` package exports exactly this and no more at v1:

```go
// Construction & lifecycle (from the client/options specs).
func New(opts ...Option) (*Client, error)
type Client struct{ /* opaque */ }
type Option func(*config) error            // config is unexported

func WithTimeout(d time.Duration) Option
func WithConnectTimeout(d time.Duration) Option
func WithRetry(p RetryPolicy) Option
func WithProxy(url string) Option
func WithTLS(cfg *tls.Config) Option
func WithTransport(rt http.RoundTripper) Option
func WithMiddleware(mw ...Middleware) Option
func WithTracer(t Tracer) Option
func WithMetrics(m Metrics) Option
func WithLogger(l Logger) Option
func WithRedirectPolicy(p RedirectPolicy) Option
func WithSSRFGuard(g SSRFGuard) Option
func WithMaxConnsPerHost(n int) Option

// Parse once, execute many.
func (c *Client) Prepare(command string) (*Request, error)
func (c *Client) Do(ctx context.Context, req *Request) (*http.Response, error)
type Request struct{ /* immutable, cloneable with per-call overrides */ }
func (r *Request) Clone() *Request

// Curl* helpers — package level (default Client) AND as Client methods.
func Curl(ctx context.Context, command ...string) (*http.Response, error)
func CurlString(ctx context.Context, command ...string) (string, *http.Response, error)
func CurlBytes(ctx context.Context, command ...string) ([]byte, *http.Response, error)
func CurlJSON(ctx context.Context, v any, command ...string) (*http.Response, error)
func CurlDownload(ctx context.Context, path string, command ...string) (int64, *http.Response, error)
// ...and the Command/Args/WithVars variants already in api.go, kept verbatim.

// Supporting types kept stable.
type Variables map[string]string                 // unchanged from api.go
type Middleware func(next Handler) Handler        // from the middleware spec
type Handler func(*http.Request) (*http.Response, error)
type RetryPolicy struct{ /* idempotency-aware */ }
type GocurlError struct{ Op, Command, URL string; Err error } // from errors.go
```

Everything else exported today is **not** in the keep-list and is handled by the table below.

### The unexport / relocate plan

The execution machinery currently lives in the root package as exported functions. They were
exported only because the package grew organically; nothing external needs them once `Client`
exists. Disposition:

| Current symbol | File | v1 disposition |
| --- | --- | --- |
| `Process` (`(resp, body, err)`) | `process.go` | **Removed at v1.** Deprecated now (already marked). Tests pin its `(resp,body,err)` shape and `HandleOutput` stdout behavior — keep through v0. Replaced by `Client.Do` + caller-side body read. |
| `Execute`, `Request`, `RequestWithContext`, `Response` | `api.go` | **Removed at v1.** Already marked Deprecated. The `Response` wrapper (buffering `String/Bytes/JSON`) is superseded by `CurlString/CurlBytes/CurlJSON`. |
| `CreateHTTPClient` | `process.go` | **Unexport → `internal/engine`** (or `createHTTPClient`). Pure machinery; `Client` owns client construction. |
| `CreateRequest` | `process.go` | **Unexport/move to internal.** |
| `ApplyMiddleware` | `process.go` | **Unexport/move to internal.** Public middleware composition is via `WithMiddleware` + the `Middleware`/`Handler` types. |
| `HandleOutput` | `process.go` | **Move to `cmd/gocurl` (unexported).** It writes to `OutputFile`/stdout — a CLI concern, never library behavior (per VISION principle 2). |
| `ValidateOptions` / `ValidateRequestOptions` | `process.go`, `security.go` | **Unexport/move to internal.** Validation is an engine step, not API. |
| `ArgsToOptions` | `convert.go` | **Unexport/move to internal.** `Client.Prepare` is the public parse entry. |
| `options.RequestOptions` & fields | `options/` | **Demote from the public contract.** It currently appears in `Process`/`Execute` signatures; once those go, no v1 root signature references it. It becomes an internal config carrier behind `Request`/`Option`. |
| `CreateProxyTransport`, `getRoundTripper`, transport cache | `clientpool.go`, `process.go` | Already unexported (`getRoundTripper`, `newTransport`, `transportKey`) or trivially so. Move under `internal/engine` with the rest. |
| `DecompressResponse`, `GetAcceptEncodingHeader`, `ConfigureCompressionForTransport` | `compression.go` | **Unexport/move to internal.** Implementation detail of streaming. |
| `IsSensitiveHeader`, `RedactHeaders` | `errors.go` | **Keep exported** — `cmd/gocurl/main.go` consumes `gocurl.IsSensitiveHeader` for verbose redaction, and they are genuinely useful to embedders. In the keep-list as a small "redaction" group. |
| `ParseError/RequestError/ResponseError/RetryError/ValidationError` | `errors.go` | **Keep**, but reframe as constructors behind the classified-error API; the `GocurlError` type and `errors.Is/As` helpers are the stable contract. |
| TLS helpers `LoadTLSConfig`, `ParseTLSVersion`, `ParseCipherSuites`, `ParseTLS13CipherSuites` | `security.go`, `tls_utils.go` | **Unexport/move to internal** unless `WithTLS` exposes a parsing convenience; default is internal. |
| `NewPersistentCookieJar`, `LoadCookiesFromFile`, `SaveCookiesToFile` | `cookie.go` | **Keep exported** (small, useful, no churn risk) but document as a stable utility group. Confirm in review. |
| `Version` var | `version.go` | **Keep exported** (set via `-ldflags`); see release process. |

The mechanism is a new `internal/` tree (Go's compiler-enforced `internal/` rule prevents
external import), e.g. `internal/engine` (request build, client, retries, validation, output)
and `internal/parse` (the current `convert.go`/`tokenizer` plumbing wrapped by `Prepare`). The
root package keeps only the keep-list and delegates inward.

### Backward compatibility for existing `Curl*` users

The package-level `Curl*` functions in `api.go` are the surface most users actually call, and
they are **preserved unchanged** in v1. Per the locked design, they become thin wrappers over a
lazily-initialized default `Client`:

```go
var defaultClient = sync.OnceValue(func() *Client { c, _ := New(); return c })

func Curl(ctx context.Context, command ...string) (*http.Response, error) {
    return defaultClient().Curl(ctx, command...)
}
```

This is a behavior-preserving refactor: same signatures, same auto-detection (single string =
shell command, multiple = args), same env-var expansion, same streamed-body contract. No
existing `Curl*` caller needs to change a line to move from v0.x to v1.

## Behavior & edge cases

- **Deprecated-but-present (v0.x):** `Process`, `Execute`, `Request`, `RequestWithContext`,
  `Response`, and the to-be-unexported helpers stay exported and functional through every
  `v0.x` tag, each carrying a `// Deprecated:` doc comment naming its replacement so `go vet`
  / `staticcheck` (SA1019) and IDEs flag usages. Their existing tests
  (`process_test.go`, `process2_test.go`) keep pinning current behavior until removal.
- **Removal happens only at the v1.0.0 boundary**, never inside v0.x, and never silently —
  each removal is listed in the `[1.0.0]` CHANGELOG `### Removed` section with the v0 minor in
  which it was deprecated.
- **`options.RequestOptions` exposure:** any embedder constructing it directly (via
  `Execute`/`Process`) loses that path at v1. The migration guide shows the `Client.Prepare` +
  `Option` equivalent. Because `RequestOptions` is large and field-mutable, it is explicitly
  *not* promised stable even in v0.
- **Default-Client semantics:** the lazy default `Client` must reproduce today's behavior
  exactly — cached idle-tuned transports (`clientpool.go`), env-var expansion on values only,
  no stdout writes. A regression here is a compatibility break even though signatures match.
- **CLI vs library split:** `HandleOutput`'s stdout/`OutputFile` writes move entirely into
  `cmd/gocurl`. The library must never print, so this relocation is also a correctness fix.
  The CLI's exported `gocurl` dependencies shrink to the redaction helpers + `Curl*`.
- **`go.mod` go directive:** stays at `go 1.22.3` (or bumps only on a minor). The v1 tag does
  not require a Go version bump; if one is needed it is called out as a minor's breaking-ish
  note, not hidden in a patch.
- **Pre-1.0 `pseudo-versions`:** until the first tag, `go get` resolves pseudo-versions
  (`v0.0.0-<date>-<sha>`). The first real tag (`v0.1.0`) immediately improves resolvability.

## Acceptance criteria / Definition of Done

- [ ] `CHANGELOG.md` gains an explicit **versioning policy** paragraph (v0 = unstable, v1 =
      SemVer promise) and the `[Unreleased]` section is split toward a concrete first tag.
- [ ] Every symbol slated for removal/unexport carries a `// Deprecated: use X` comment; a
      `staticcheck`/`go vet` run reports SA1019 on internal usages and the deprecation list
      matches this spec's table.
- [ ] An `internal/` tree exists and the root package imports inward; no external-importable
      package exposes `CreateHTTPClient`, `CreateRequest`, `ApplyMiddleware`, `ArgsToOptions`,
      validation, TLS-parsing, or compression internals.
- [ ] `HandleOutput` no longer exists in the `gocurl` root package; equivalent logic lives
      unexported in `cmd/gocurl`, and a test confirms the library produces zero stdout writes.
- [ ] Package-level `Curl*` functions delegate to a lazily-initialized default `Client` and a
      test asserts byte-identical behavior (status, headers, streamed body, redaction) vs. the
      pre-refactor implementation.
- [ ] A `MIGRATION.md` (or a `# Migration` section in the README) documents the one-shot →
      `Client` path with side-by-side `Process`/`Execute`/`Request` → `Client.Do`/`Curl*`
      examples and a `gofmt -r` recipe for the common rename.
- [ ] An **API-surface guard** exists in CI: a golden `api.txt` (e.g. via `go run
      golang.org/x/exp/cmd/apidiff` or `go doc`-diff) so any change to the exported root
      surface fails the build until the golden file and CHANGELOG are updated.
- [ ] The release process is documented and exercised once on a pre-release tag: annotated
      `git tag v0.1.0`, push, and `Version` stamped via
      `-ldflags "-X github.com/maniartech/gocurl.Version=0.1.0"` in `cmd/gocurl` builds.
- [ ] `go build ./...`, `go vet ./...`, and `go test -short -race ./...` pass on the tagged
      commit (the existing CI workflow already runs these).
- [ ] A documented **deprecation timeline** maps each deprecated symbol to "deprecated in
      v0.N, removed in v1.0.0".

## Dependencies

- **Client lifecycle / `New` + `Option` spec** — defines the surface this spec freezes
  (`Client`, `Option`, `Prepare`, `Do`, the `With*` options).
- **Resilience spec** — defines `RetryPolicy`, `RedirectPolicy`, retry/backoff types kept in
  the v1 surface.
- **Observability spec** — defines `Tracer`, `Metrics`, `Logger` interfaces kept in v1.
- **Errors/classification spec** — defines the classified `GocurlError` + `errors.Is/As`
  helpers referenced in the keep-list.
- **Middleware spec** — defines `Middleware`/`Handler` (the public replacement for
  `ApplyMiddleware`).
- **SSRF/redirect-guard spec** — defines `SSRFGuard` for `WithSSRFGuard`.

## Open questions / decisions to confirm in review

- **First tag: `v0.1.0` or straight to `v1.0.0`?** Proposal: ship `v0.1.0` first to get the
  module out of pseudo-version land and bake the `Client` API for one or two minors, then cut
  `v1.0.0`. Confirm we don't want to commit to the v1 promise immediately.
- **Module split for OTel:** keep the OpenTelemetry adapter in-tree (adds an
  `go.opentelemetry.io/otel` dependency to every importer) or as a separate
  `github.com/maniartech/gocurl/otel` module? Leaning separate module to keep the core
  dependency-light. Out of scope to decide here but blocks the v1 dependency surface.
- **Cookie utilities (`NewPersistentCookieJar` et al.) and TLS parsers:** keep exported as a
  blessed utility group, or hide them too? Proposal: keep cookie jar exported (low churn,
  clearly useful), hide TLS parsers behind `WithTLS`. Confirm.
- **`Variables` + `*WithVars` family:** confirm all `CurlCommandWithVars`/`CurlArgsWithVars`
  variants stay in the v1 surface, or whether we collapse the variadic auto-detect entry
  points and drop the explicit `Command`/`Args` spellings to shrink the surface.
- **`options.RequestOptions` as a *read-only* escape hatch?** Some embedders may want to
  inspect a parsed `Request`. Do we expose a read accessor on `Request` instead of leaking
  `RequestOptions`, and is that needed for v1 or deferrable?
- **apidiff vs. a hand-maintained `api.txt`:** which surface-guard tooling does CI adopt?
- **`Version` source of truth:** keep the build-time `-ldflags` var (current approach), or
  move to `runtime/debug.ReadBuildInfo()` so tagged installs report the right version without
  an explicit `-X`? Proposal: support both — prefer `ReadBuildInfo`, fall back to the
  `-ldflags` var, default `"dev"`.
