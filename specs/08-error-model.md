# Error Model

> Status: Draft for review · Spec 08

## Goals

- Promote the existing `GocurlError` (see `errors.go`) into a **typed error taxonomy** that
  spans the full request lifecycle: parse, validation, connect, TLS, timeout, server-status,
  retry-exhausted, body-read, and cancellation.
- Give every error a **classification surface** so callers can branch on behavior, not on
  string matching: `Timeout() bool`, `Temporary() bool`, `Retryable() bool`.
- Make errors **`errors.Is` / `errors.As` friendly**: stable sentinel values for kinds,
  preserved wrapping chains (the existing `Unwrap()` stays), and a single concrete type
  callers can `errors.As` into.
- Lock in the **status-code policy**: gocurl does **not** turn 4xx/5xx into Go errors by
  default — it returns the live `*http.Response`. Provide an explicit opt-in for callers who
  want curl `-f`/`--fail`-style behavior.
- Guarantee **sensitive-data scrubbing** in every error string, reusing and extending the
  redaction already in `errors.go` (`sanitizeCommand`, `sanitizeURL`, `RedactHeaders`).
- Keep the change **backward compatible through v0.x**: `GocurlError`, its fields, and the
  `ParseError`/`RequestError`/`ResponseError`/`RetryError`/`ValidationError` constructors
  keep working and keep their current `Error()` output shape.

## Non-goals

- Not changing the *control flow* of `doRequest`/`executeWithRetries` (Spec 04/Resilience
  owns retry policy). This spec defines how failures are *typed and classified*, not when a
  retry fires.
- Not introducing localized or i18n error messages.
- Not defining a wire/serialization format for errors (no JSON error envelope). Errors are
  Go values.
- Not adding error codes for transports we do not ship in v1 (e.g. HTTP/3 / QUIC). Those get
  their own kinds when their specs land.
- Not changing the default "return the response, don't error on HTTP status" contract — only
  formalizing it and adding the opt-in.

## Design

### One concrete type, one kind enum

Keep `GocurlError` as the single concrete error type. Add a typed `Kind` and a classification
triple. The existing `Op string` field is retained (and still populated for back-compat
string output), but `Kind` becomes the machine-readable discriminator.

```go
// Kind classifies a GocurlError. It is the machine-readable discriminator;
// the legacy Op string is derived from it for backward-compatible messages.
type Kind int

const (
    KindUnknown        Kind = iota
    KindParse                 // tokenize / convert curl command failed
    KindValidation            // ValidateRequestOptions rejected the prepared request
    KindConnect               // dial / connection refused / DNS / proxy connect
    KindTLS                   // handshake, cert verify, pin mismatch
    KindTimeout               // deadline exceeded (connect, per-attempt, or overall)
    KindCanceled              // context canceled by caller
    KindServerStatus          // 4xx/5xx surfaced as error (opt-in only; see below)
    KindRetryExhausted        // all retry attempts failed
    KindBodyRead              // reading/decoding the response body failed
)

func (k Kind) String() string // "parse", "validation", "connect", ...
```

`GocurlError` gains fields but keeps the originals:

```go
type GocurlError struct {
    Op      string // legacy, derived from Kind (kept for message compatibility)
    Kind    Kind   // NEW: machine-readable classification
    Command string // sanitized command snippet
    URL     string // sanitized URL
    Status  int    // NEW: HTTP status code when Kind == KindServerStatus, else 0
    Attempt int    // NEW: attempts made when Kind == KindRetryExhausted, else 0
    Err     error  // wrapped underlying error (net.Error, tls error, context err, ...)
}

func (e *GocurlError) Error() string  // unchanged shape; see Behavior
func (e *GocurlError) Unwrap() error  // unchanged: returns e.Err
```

### Classification interface

Classification is computed from `Kind` plus the wrapped `Err`, so it stays correct even when
`Err` is a stdlib `net.Error`, `*tls.CertificateVerificationError`, or `context` error.

```go
// Timeout reports whether the error was caused by a deadline/timeout.
func (e *GocurlError) Timeout() bool

// Temporary reports whether the failure is plausibly transient. This is
// advisory and intentionally conservative; it is NOT a retry decision on its
// own (method idempotency still governs retries — see Spec 04).
func (e *GocurlError) Temporary() bool

// Retryable reports whether gocurl's own resilience layer considers this error
// safe to retry for an idempotent request.
func (e *GocurlError) Retryable() bool
```

Package-level helpers mirror `errors.Is`/`errors.As` ergonomics so callers never need to
type-assert:

```go
func IsTimeout(err error) bool    // any GocurlError in the chain reports Timeout()
func IsTemporary(err error) bool
func IsRetryable(err error) bool
func KindOf(err error) Kind       // KindUnknown if no GocurlError in chain
```

### errors.Is / errors.As support

Expose **sentinel kinds** so `errors.Is(err, ErrTimeout)` works without exposing the concrete
type, alongside `errors.As` for full detail.

```go
var (
    ErrParse          = &kindSentinel{KindParse}
    ErrValidation     = &kindSentinel{KindValidation}
    ErrConnect        = &kindSentinel{KindConnect}
    ErrTLS            = &kindSentinel{KindTLS}
    ErrTimeout        = &kindSentinel{KindTimeout}
    ErrCanceled       = &kindSentinel{KindCanceled}
    ErrServerStatus   = &kindSentinel{KindServerStatus}
    ErrRetryExhausted = &kindSentinel{KindRetryExhausted}
    ErrBodyRead       = &kindSentinel{KindBodyRead}
)
```

`GocurlError.Is(target error) bool` returns true when `target` is the matching `kindSentinel`,
and otherwise delegates to the wrapped chain (so `errors.Is(err, context.Canceled)` and
`errors.Is(err, context.DeadlineExceeded)` keep working through `Unwrap`). Usage:

```go
resp, err := gocurl.Curl(ctx, cmd)
switch {
case errors.Is(err, gocurl.ErrTimeout):   // timed out
case errors.Is(err, gocurl.ErrConnect):   // couldn't reach host
case gocurl.IsRetryable(err):             // safe to try again ourselves
}

var gerr *gocurl.GocurlError
if errors.As(err, &gerr) && gerr.Kind == gocurl.KindServerStatus {
    log.Printf("server returned %d", gerr.Status)
}
```

### Constructors

The five existing exported constructors stay (`ParseError`, `RequestError`, `ResponseError`,
`RetryError`, `ValidationError`) and now set the appropriate `Kind`. `RequestError` is
upgraded to *classify* the wrapped error rather than blindly tagging `Op: "request"`:

```go
// classifyTransportError inspects a transport-layer error from client.Do and
// returns the most specific Kind (KindCanceled, KindTimeout, KindTLS, KindConnect).
func classifyTransportError(err error) Kind
```

`RequestError(url, err)` calls `classifyTransportError` so that, e.g., a
`*tls.CertificateVerificationError` produces `Kind == KindTLS` and a `net.OpError{Op:"dial"}`
produces `KindConnect`, while a `context.DeadlineExceeded` produces `KindTimeout`. New
constructors fill the gaps:

```go
func ServerStatusError(url string, status int) error   // Kind=KindServerStatus, Status=status
func BodyReadError(url string, err error) error         // Kind=KindBodyRead
func ConnectError(url string, err error) error          // Kind=KindConnect
func TLSError(url string, err error) error              // Kind=KindTLS
func TimeoutError(url string, err error) error          // Kind=KindTimeout
func CanceledError(url string, err error) error         // Kind=KindCanceled
```

### Status-code policy (the load-bearing decision)

Default behavior is unchanged and explicit: **a non-2xx HTTP response is success at the
transport level**. `executeOpts`/`doRequest` (in `api.go`/`process.go`) return the live
`*http.Response` with a non-nil error only for transport/parse/validation failures. 4xx and
5xx come back as a response the caller inspects — exactly as today, where `CurlJSON`/
`CurlString` read the body regardless of status.

Opt-in fail-on-status mirrors curl `-f`/`--fail`:

```go
// Option (Spec 03) and a RequestOptions field that the parser sets from -f/--fail.
func WithFailOnStatus(fail bool) Option

// options.RequestOptions gains:
//   FailOnError bool `json:"fail_on_error,omitempty"`
```

When enabled, after a successful round-trip with `StatusCode >= 400`, the engine returns
`ServerStatusError(url, resp.StatusCode)` **and still returns the `*http.Response`** so the
caller may read the error body (curl `-f` discards it; we are more useful and let the caller
decide). The convenience wrappers (`CurlString`, `CurlJSON`, `CurlBytes`, `CurlDownload`)
propagate that error but, like today, return the response alongside it.

### Sensitive-data scrubbing

All scrubbing flows through the existing helpers in `errors.go` and is applied **at
construction time**, before the string is ever stored:

- `sanitizeURL` is already applied by `RequestError`/`ResponseError`/`RetryError`; the new
  constructors use it too, so `GocurlError.URL` is always redacted (`api_key`, `token`,
  `secret`, `password`, `key`, `apikey` query params → `[REDACTED]`).
- `sanitizeCommand` redacts `-u user:pass`, `Authorization`/`Bearer`/`Basic`, `Cookie:`, and
  sensitive URL params in `GocurlError.Command`.
- `Error()` never includes header values or the raw response body. If a wrapped stdlib error
  embeds a full URL (Go's `*url.Error` does), `Error()` re-scrubs the *final* string through
  `sanitizeURL` as a backstop so credentials in `userinfo` or query strings never leak.

```go
// scrubErrorString is the final-pass backstop run inside Error() on the joined
// message, covering credentials that leak in from wrapped stdlib errors.
func scrubErrorString(msg string) string
```

## Behavior & edge cases

- **Back-compat message shape.** `Error()` keeps the current `op: url=… cmd=… underlying`
  layout from `errors.go`. `Status` is appended to the `op` segment only for
  `KindServerStatus` (e.g. `server status (503)`); `Attempt` only for `KindRetryExhausted`
  (the existing `retry (after N attempts)` text is preserved).
- **Timeout vs cancellation split.** `context.DeadlineExceeded` ⇒ `KindTimeout` (`Timeout()==true`);
  `context.Canceled` ⇒ `KindCanceled` (`Timeout()==false`, `Retryable()==false`). This
  matches the existing `isContextError` check in `retry.go` but separates the two outcomes,
  which today are merged into one "context error" string.
- **Retry-exhausted wrapping.** When `retryLoop` gives up, the returned error is a
  `KindRetryExhausted` `GocurlError` whose `Err` is the *last* attempt's classified error.
  `errors.Is(err, gocurl.ErrTimeout)` therefore still reports true if the final attempt timed
  out, because `Is` walks the chain. `Attempt` records attempts made.
- **Retryable() must agree with the engine.** `Retryable()` returns true exactly for the
  kinds `executeWithRetries`/`needsRetry` would retry for an idempotent request:
  `KindConnect`, `KindTimeout`, transient `KindTLS` is **not** retryable, and `KindServerStatus`
  is retryable only for the codes in `shouldRetry` (429/500/502/503/504 by default). It is the
  classifier of record; the resilience layer (Spec 04) consults it.
- **Body-read errors.** `io.ReadAll`/`json.Decode` failures in the convenience wrappers and
  the `limitedBody`/`readBodyWithLimit` over-limit error become `KindBodyRead`. The
  over-limit case wraps a sentinel so callers can distinguish truncation from a network read
  error.
- **Validation errors are not retryable and not temporary.** `ValidateRequestOptions`
  failures (`security.go`) keep `Kind == KindValidation`.
- **Parse errors** from `CurlCommand`/`CurlArgs` (tokenize/convert) are `KindParse`; they
  carry the sanitized command, never raw `$VAR`-expanded secrets.
- **Nil `Err`.** All classification methods are nil-safe on `e.Err`; `Timeout()`/`Retryable()`
  fall back to `Kind` alone.
- **`FailOnError` + redirects.** Status is evaluated on the final response after redirect
  handling (`redirectPolicy` in `process.go`), consistent with curl `-f` semantics.
- **CLI parity.** The `cmd/gocurl` CLI maps `KindServerStatus` to a non-zero exit when `-f`
  is set, matching curl's exit code 22 family (exact code table is a CLI concern, deferred to
  the CLI spec; this spec only guarantees the error is distinguishable).

## Acceptance criteria / Definition of Done

- [ ] `Kind`, the nine `Kind*` constants, and `Kind.String()` exist; `GocurlError` gains
      `Kind`, `Status`, `Attempt` without removing `Op`, `Command`, `URL`, or `Err`.
- [ ] `Timeout()`, `Temporary()`, `Retryable()` are implemented on `*GocurlError`, are
      nil-`Err`-safe, and have table-driven tests covering every `Kind`.
- [ ] Sentinels `ErrParse … ErrBodyRead` exist; `errors.Is(err, ErrTimeout)` and
      `errors.As(err, &gerr)` both work, and `errors.Is(err, context.DeadlineExceeded)` still
      resolves through `Unwrap`.
- [ ] `IsTimeout`, `IsTemporary`, `IsRetryable`, `KindOf` package helpers exist and walk the
      chain.
- [ ] `classifyTransportError` maps `net.OpError`(dial)→`KindConnect`,
      `*tls.CertificateVerificationError`/cert-pin mismatch→`KindTLS`,
      `context.DeadlineExceeded`→`KindTimeout`, `context.Canceled`→`KindCanceled`, with tests.
- [ ] Default path: a 404/500 response returns `(resp, nil)` from `Curl`/`doRequest`
      (regression test pins current behavior).
- [ ] `WithFailOnStatus`/`opts.FailOnError` + `-f`/`--fail` parsing wired; when on, a ≥400
      response yields `ServerStatusError` *with* the response still returned; off by default.
- [ ] Every constructor stores only scrubbed `Command`/`URL`; `scrubErrorString` backstop runs
      inside `Error()`. A test asserts a wrapped `*url.Error` containing
      `?api_key=SECRET` and `user:pass@host` shows `[REDACTED]` in the final string.
- [ ] Retry-exhausted error is `KindRetryExhausted`, records `Attempt`, and chains the last
      attempt's classified error such that `IsTimeout`/`errors.Is` still resolve.
- [ ] Existing `errors_test.go` and `race_test.go` pass unchanged (message-shape and
      construction back-compat); race-clean.
- [ ] No raw header values or response bodies appear in any `Error()` output (asserted test).

## Dependencies

- Spec 03 (Client & Options) — `WithFailOnStatus` is a functional `Option`; `FailOnError`
  lives on the prepared `Request`/`RequestOptions`.
- Spec 04 (Resilience / RetryPolicy) — consumes `Retryable()`/`Timeout()` for retry
  decisions; this spec is the classifier of record, Spec 04 is the policy of record.
- Builds directly on current `errors.go` (`GocurlError`, constructors, `sanitizeCommand`,
  `sanitizeURL`, `redactURLParams`, `RedactHeaders`, `IsSensitiveHeader`), `retry.go`
  (`isContextError`, `needsRetry`, `shouldRetry`), `security.go` (`ValidateRequestOptions`,
  `ValidationError` call sites), and `process.go`/`api.go` (`doRequest`, `executeOpts`,
  `readBodyWithLimit`, `limitedBody`).

## Open questions / decisions to confirm in review

1. **`Temporary()` and Go 1.18+ deprecation.** `net.Error.Temporary()` is deprecated upstream.
   Do we keep `Temporary()` for familiarity (documented as advisory) or drop it in favor of
   `Retryable()` only? *Proposed: keep it, advisory, not a retry signal.*
2. **Should `FailOnError` return the response body alongside the error?** Proposed yes (more
   useful than curl, which discards it). Confirm this doesn't surprise users porting `-f`
   scripts.
3. **Is `KindServerStatus` ever produced when `FailOnError` is off?** Proposed no — without
   the opt-in, status is never an error. Confirm no middleware/observability path needs the
   error form regardless.
4. **Granularity of `KindConnect`.** Do we want a distinct `KindDNS` and `KindProxy`, or fold
   both into `KindConnect`? *Proposed: fold into `KindConnect` for v1; split later if a
   benchmark/diagnostic need appears.*
5. **`Kind` zero value.** `KindUnknown` is the zero value; a freshly constructed legacy
   `&GocurlError{Op:"request"}` would report `KindUnknown`. Acceptable, or do constructors
   need to be the only supported way to build one (i.e. document direct struct literals as
   unsupported in v1)?
