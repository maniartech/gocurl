# Request Model & Curl Compatibility

> Status: Draft for review · Spec 02

## Goals

- Define the immutable, prepared **`Request`** template — the concrete artifact of
  "parse once, execute many." A curl string (or programmatic builder) is parsed
  *once* into a `Request`; the same `Request` is then executed many times over a
  reusable, pooled `Client` (see Spec 01) with no repeated parsing cost.
- Specify `client.Prepare(command string) (*Request, error)`, which reuses the
  existing `tokenizer` + `convert.go` pipeline (`Tokenize` →
  `convertTokensToRequestOptions`) so curl-compat correctness is shared with the
  legacy `Curl*` functions, not forked.
- Pin down a precise, versioned **supported curl flag matrix** (currently
  implemented + planned) and an explicit, predictable **policy for unsupported
  flags** (strict vs. lenient).
- Define the two **variable substitution models** — environment (`$VAR`/`${VAR}`
  via `os.ExpandEnv`) vs. an explicit `Variables` map (via `ExpandVariables`) —
  and *when* substitution happens relative to preparing a `Request`.
- Define **cloning / templating**: cheap per-call overrides (URL, headers, body,
  query, vars) that never mutate the shared prepared `Request`.
- State the **curl-compat scope and non-goals** (HTTP/HTTPS only; no FTP/SMTP/
  other protocols; only flags that appear in real API docs).

## Non-goals

- Re-specifying the execution engine, transport pooling, middleware chain, or
  the `Client`/`Option` surface — that is Spec 01. This spec stops at producing
  and overriding a `Request`; `client.Do` consumes it.
- Resilience/retry semantics, observability, SSRF guard (later specs). We only
  note where a prepared `Request` *carries* their config.
- Full curl parity. We are not a curl clone; see Non-goals matrix below.
- Replacing `options.RequestOptions`. `Request` *wraps and owns* a finalized
  `RequestOptions` as its internal immutable state; we are not redesigning that
  struct here.

## Design

### The prepared `Request`

`Request` is an immutable, concurrency-safe template built once. Internally it
owns a fully-finalized `*options.RequestOptions` (the output of today's
`convertTokensToRequestOptions` + `finalizeRequestOptions`) plus the provenance
needed for safe cloning and templating. Callers never see the internal options
directly in v1; they interact through accessors and override methods that return
a *new* `Request`.

```go
// Request is an immutable, prepared request template — the "parse once" artifact.
// All exported methods that "change" a Request return a new Request; the receiver
// is never mutated, so a single Request is safe to share across goroutines and
// reuse across many client.Do calls.
type Request struct {
    opts *options.RequestOptions // finalized, treated as immutable after Prepare
    // template metadata (unexpanded source, declared placeholders) for re-binding vars
    raw          string            // original command, "" if built programmatically
    unexpanded   *options.RequestOptions // pre-substitution snapshot for templating (optional, see Open questions)
    boundVars    Variables         // vars already applied, for diagnostics
}

// Read-only accessors (no setters; mutation is via override constructors).
func (r *Request) Method() string
func (r *Request) URL() string
func (r *Request) Header() http.Header // returns a clone; caller cannot mutate internal state
func (r *Request) Body() []byte        // returns a copy
```

### Preparing from a curl command

`Prepare` is the single parse-once entry point. It is a thin orchestration over
the existing pipeline so there is exactly one curl grammar in the codebase:

```go
// Prepare parses a curl command ONCE into a reusable Request.
// It does NOT execute anything and performs NO environment expansion by default
// (see Variable substitution). Multi-line, backslash-continued, commented
// commands and an optional leading "curl" word are accepted.
func (c *Client) Prepare(command string) (*Request, error)

// PrepareArgs builds a Request from pre-split args (like os.Args), bypassing the
// shell tokenizer. Mirrors today's CurlArgs token path.
func (c *Client) PrepareArgs(args ...string) (*Request, error)
```

Internally `Prepare` runs the established steps, unchanged in behavior:

1. `preprocessMultilineCommand` (api.go) — strip comments, join `\` lines, drop a
   leading `curl`.
2. `tokenizer.NewTokenizer().Tokenize` (tokenizer/tokenizer.go) — shell-aware
   split honoring single/double quotes and backslash escapes, stripping quote
   delimiters. **No variable expansion happens here.**
3. `convertTokensToRequestOptions` (convert.go) — `newParseState` →
   `parseTokens` (dispatching through `processSimpleFlag`, `processDataFlags`,
   `processHeaderAuthFlags`, `processTLSFlags`, `processNetworkFlags`) →
   `finalizeRequestOptions` (URL normalization via `normalizeURL`, `applyData`,
   `-O` remote filename, `-L` default `MaxRedirects=30`).

The result is wrapped in an immutable `Request`. Crucially, `convertTokensToReq‑
uestOptions` already performs **no expansion of its own** (see api.go comments) —
expansion is a separate, explicit token-rewrite step. That property is what lets
`Prepare` defer or skip env expansion safely.

### Programmatic construction

For callers who do not start from a curl string, a builder produces the same
`Request` artifact, reusing `options.RequestOptions` + its `Clone()`:

```go
func NewRequest(method, rawURL string, opts ...RequestOption) (*Request, error)

type RequestOption func(*options.RequestOptions) error

func WithHeader(key, value string) RequestOption
func WithBody(body []byte) RequestOption
func WithQuery(key, value string) RequestOption
func WithBasicAuth(user, pass string) RequestOption
func WithBearerToken(token string) RequestOption
// ... mirrors the curl flag matrix below, 1:1 where it makes sense.
```

`NewRequest` runs the same `finalizeRequestOptions`/`normalizeURL` and
`ValidateRequestOptions` (security.go) as the curl path, so both routes converge
on one validated, finalized state.

### Variable substitution model

Two mutually exclusive models, decided **at prepare/clone time**, never silently
mixed:

```go
// Environment model: $VAR / ${VAR} expanded from os.Environ via os.ExpandEnv.
func (c *Client) Prepare(command string) (*Request, error)            // NO env expansion (explicit by default)
func (c *Client) PrepareEnv(command string) (*Request, error)         // expands from environment

// Explicit-map model: $VAR / ${VAR} resolved ONLY from the provided map via
// ExpandVariables (variables.go); undefined placeholders are an error.
func (c *Client) PrepareWithVars(command string, vars Variables) (*Request, error)
```

Substitution reuses the existing token-rewrite helpers and their security rule:
**only `TokenValue` tokens that do not begin with `-` are expanded** — flag names
are never expanded, preventing flag injection from a variable value
(`expandEnvInTokens`, `expandVarsInTokens` in api.go).

- Environment model → `os.ExpandEnv` per value token. Undefined → empty string
  (curl/shell behavior).
- Explicit model → `ExpandVariables(value, vars)`; undefined placeholder →
  `"undefined variable: NAME"` error (variables.go `lookupVariable`). Escape with
  `\$` to emit a literal `$`.

> Behavioral note vs. legacy: package-level `Curl(...)` auto-expands the
> environment; the new `Client.Prepare` does **not** by default. This is a
> deliberate, safer default for the prepared-request API (no implicit reads of
> process env). The legacy auto-env behavior is preserved only through the
> back-compat `Curl*` wrappers. Flagged under Open questions.

### Templating for "parse once, execute many"

A prepared `Request` is reused directly for the common case (identical request,
many executions). When per-call variation is needed, `With*` override
constructors return a cheap clone built on `options.RequestOptions.Clone()`
(which deep-copies `Headers`, `Form`, `QueryParams`, `BasicAuth`, `FileUpload`,
`RetryConfig`):

```go
// Each returns a NEW Request; the receiver is unchanged.
func (r *Request) WithURL(rawURL string) (*Request, error)
func (r *Request) WithHeader(key, value string) *Request      // adds/sets on a clone
func (r *Request) WithQuery(key, value string) *Request
func (r *Request) WithBody(body []byte) *Request
func (r *Request) WithContextOverrides(...) *Request           // see Open questions
```

Two distinct templating notions:

1. **Override templating (post-finalize):** mutate the cloned `RequestOptions`
   directly (headers, body, URL). Re-runs `normalizeURL`/validation only for the
   fields touched. Cheapest; covers most cases.
2. **Placeholder re-binding (pre-finalize):** for `Prepare` results that retain
   the *unexpanded* command/snapshot, re-apply a fresh `Variables` map to produce
   a new `Request` (e.g., one prepared template, executed per-tenant with
   different `$API_KEY`). Implemented by re-running the expansion + convert step
   from the retained `raw`/`unexpanded` source.

```go
// Re-bind placeholders against a new variable map (placeholder re-binding).
func (r *Request) Rebind(vars Variables) (*Request, error)
```

## Behavior & edge cases

- **Parse-once cost:** tokenize + convert + validate run exactly once per
  `Prepare`/`NewRequest`. `Do` must not re-parse. Override clones do not
  re-tokenize.
- **Immutability:** `Header()`/`Body()` accessors return copies; `With*` never
  mutate the receiver. A shared `*Request` is safe for concurrent `Do`.
- **URL normalization:** scheme-less hosts default to `http://`; userinfo, port,
  path, query, fragment preserved losslessly (`normalizeURL`). Empty/host-less
  URLs error at prepare time, not execute time.
- **Method inference (must match curl, already in convert.go):** `-d`/`--data*`
  and `--data-urlencode` flip default GET→POST (`setPostIfDefault`); `-F` flips
  GET→POST; `-T` flips GET→PUT; `-I` sets HEAD; `-G` forces GET and moves
  accumulated data to the query string (`applyData`). Explicit `-X` always wins.
- **`-d` data joining:** multiple `-d` joined with `&`; `@file` read (CR/LF
  stripped except `--data-binary`/`--data-raw`); default `Content-Type:
  application/x-www-form-urlencoded` set when unset.
- **Variable expansion ordering:** expansion is a token-rewrite step *before*
  `convertTokensToRequestOptions`, so an expanded value can supply a header
  value or URL but can never turn into a new flag.
- **Unsupported flag policy (default = strict):** an unrecognized flag yields
  `unknown flag: <flag>` from `processFlag` and fails `Prepare`. A lenient mode
  (skip-and-warn) is opt-in via `WithUnsupportedFlagPolicy` so pasting a doc
  command containing one cosmetic flag (e.g. `--progress-bar`) need not hard-fail.
- **Recognized-but-ignored flags:** a small allow-list of no-op-at-library-level
  flags (e.g. `--progress-bar`, `-#`, `-i`/`--include` which is a CLI output
  concern, `-w`/`--write-out`) are accepted and dropped with a structured warning
  rather than erroring. These are presentation flags handled by `cmd/gocurl`
  (main.go `separateFlags`), not request semantics.
- **Missing flag argument:** `nextArg` returns `missing value for <flag>` —
  surfaced at prepare time.
- **Forbidden headers:** `Host`, `Content-Length`, `Transfer-Encoding` rejected
  by `validateHeaders` (options/validation.go) at prepare time.
- **Insecure auth:** BasicAuth/Bearer over `http://` flagged by
  `validateSecureAuth` at prepare time (before any network call).

### Supported curl flag matrix

Implemented today (verified in convert.go); these MUST keep working through the
`Request` path:

| Category | Flags |
|---|---|
| Method/body | `-X/--request`, `-d/--data`, `--data-raw`, `--data-binary`, `--data-urlencode`, `-G/--get`, `-T/--upload-file`, `-F/--form` (incl. `@file`) |
| Headers/auth/identity | `-H/--header`, `-u/--user`, `-b/--cookie`, `-c/--cookie-jar`, `-A/--user-agent`, `-e/--referer` |
| TLS | `--cert`, `--key`, `--cacert`, `--tlsv1`/`.0`/`.1`/`.2`/`.3`, `--tls-max`, `--ciphers`, `--tls13-ciphers`, `-k/--insecure` |
| Proxy | `-x/--proxy`, `--proxy-cert`, `--proxy-key`, `--proxy-cacert`, `--proxy-insecure`, `--noproxy` |
| Network/redirect/retry | `--max-time`, `--connect-timeout`, `-L/--location`, `--max-redirs`, `--retry` |
| Compression/HTTP | `--compressed`, `--http2`, `--http2-only`/`--http2-prior-knowledge` |
| Output (lib-level) | `-o/--output`, `-O/--remote-name` |
| Behavior/diagnostic | `-v/--verbose`, `-s/--silent`, `-I/--head`, `--url` |

Planned (not yet implemented; add behind the same dispatch functions):

| Flag(s) | Maps to |
|---|---|
| `--retry-delay`, `--retry-max-time`, `--retry-all-errors` | `RetryConfig` (Spec on resilience) |
| `--bearer` / `--oauth2-bearer` | `BearerToken` |
| `--header @file` / `-H @file` | header file loading |
| `--compressed-ssh` n/a; `--raw` | disable auto-decompress |
| `--resolve`, `--connect-to` | custom host→addr mapping (transport) |
| `--http1.0`, `--http1.1`, `--http3` | version pinning (HTTP/3 future) |
| `-:` / `--next` | multiple requests in one command (deferred; see Non-goals) |

### Non-goals / explicitly unsupported

| Area | Decision |
|---|---|
| Protocols | HTTP/HTTPS only. `ftp://`, `smtp://`, `tftp://`, `scp://`, `file://`, `dict://`, `imap://`, etc. → unsupported-scheme error |
| Output formatting | curl's `--write-out`, `-#`/`--progress-bar`, `-i`/`--include`, `-D`/`--dump-header` are CLI presentation concerns; library treats them as recognized-but-ignored |
| Multi-request | `--next` chaining deferred (a `Request` is a single request) |
| Telnet/interactive, `--telnet-option`, `--mail-*` | unsupported |
| `--trace`/`--trace-ascii` | superseded by Spec on observability/verbose |

## Acceptance criteria / Definition of Done

- [ ] `Client.Prepare`, `PrepareArgs`, `PrepareEnv`, `PrepareWithVars` exist and
      route through `tokenizer` + `convertTokensToRequestOptions` (single grammar;
      no duplicated parsing logic).
- [ ] A `Request` is immutable: `Header()`/`Body()` return copies; all `With*`
      return a new `Request`; `go test -race` passes with one `Request` shared
      across concurrent `Do` calls.
- [ ] Parse cost is incurred once: a test asserts `Do` does not re-tokenize/re-
      convert (e.g. via a counting hook or benchmark showing flat parse cost as
      `Do` count grows).
- [ ] Every flag in the "Implemented today" table is covered by a prepare-level
      test asserting the resulting `Request` fields (method, headers, body, URL,
      TLS/proxy/timeout options).
- [ ] Method inference matches curl for `-d`, `-F`, `-T`, `-I`, `-G`, with
      explicit `-X` overriding.
- [ ] Env model (`PrepareEnv`) and explicit-map model (`PrepareWithVars`) are
      isolated: a test proves `PrepareWithVars` never reads `os.Environ`, and
      flag tokens are never expanded (no flag injection).
- [ ] Undefined variable in explicit mode errors; `\$` escapes to literal `$`.
- [ ] Unsupported flag → strict error by default; `WithUnsupportedFlagPolicy`
      switches to skip-and-warn; recognized-but-ignored flags never hard-fail.
- [ ] Non-HTTP scheme is rejected at prepare time with a clear error.
- [ ] `Rebind(vars)` produces a new `Request` with re-applied placeholders without
      mutating the original.
- [ ] Back-compat: existing package-level `Curl*` functions still auto-expand env
      and behave identically (regression tests in `tests/` stay green).
- [ ] Programmatic `NewRequest` + `With*RequestOption` produce a `Request`
      indistinguishable from the equivalent curl command for the same intent.

## Dependencies

- **Spec 01** (Client / Option / `Do` / middleware) — consumes the `Request` and
  owns execution, transport pooling, and the `WithUnsupportedFlagPolicy` plumbing
  on `config`.
- Builds directly on existing code: `tokenizer/tokenizer.go`, `convert.go`
  (`convertTokensToRequestOptions`, `finalizeRequestOptions`, `normalizeURL`,
  `processSimpleFlag`/`processDataFlags`/`processHeaderAuthFlags`/`processTLSFlags`/
  `processNetworkFlags`), `api.go` (`preprocessMultilineCommand`,
  `expandEnvInTokens`, `expandVarsInTokens`), `variables.go` (`ExpandVariables`),
  `options/options.go` (`RequestOptions`, `Clone`), `options/validation.go`
  (`ValidateRequestOptions`, `validateSecureAuth`).

## Open questions / decisions to confirm in review

1. **Default env expansion in `Prepare`.** Proposed: `Prepare` does *not* expand
   env (explicit, safer); `PrepareEnv` opts in. This diverges from package-level
   `Curl`'s implicit env expansion. Confirm the safer default, or make `Prepare`
   env-expanding to match `Curl` and add `PrepareNoEnv`.
2. **Templating retention cost.** Placeholder re-binding (`Rebind`) requires
   retaining the unexpanded source (`raw` string and/or pre-finalize snapshot).
   Confirm whether every `Request` keeps this, only `Prepare`-built ones, or only
   when an opt-in (`WithTemplating()`) is set — to avoid memory overhead for
   one-shot requests.
3. **`Request` immutability vs. body streams.** Body is currently a `string`
   (`opts.Body`). For large uploads (`-T`, `-F @file`) we copy bytes today. Do we
   keep the byte-copy model in `Request`, or allow a re-openable body provider
   (`func() (io.ReadCloser, error)`) so a `Request` can be safely re-executed and
   retried without buffering huge files? (Interacts with retry spec.)
4. **Strict-by-default unsupported flags.** Confirm strict default. Some pasted
   doc commands include benign flags (`--progress-bar`, `--compressed-ssh`);
   strict-by-default forces users to opt into lenient. Alternative: lenient
   default with a strict opt-in for production.
5. **Per-call vars on `Do` vs. `Rebind`.** Should `client.Do(ctx, req, ...vars)`
   accept ad-hoc vars, or must callers always `Rebind` first? Prefer `Rebind`
   for a clean immutable model, but confirm ergonomics.
6. **`-O` remote filename in library context.** `-O` sets `OutputFile` from the
   URL (`remoteFilename`); for a library `Request` that streams the body, is
   `OutputFile` honored by a convenience like `CurlDownload`, or treated as a
   CLI-only concern? Confirm where output-to-file lives in the prepared model.
