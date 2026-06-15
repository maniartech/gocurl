# Security Model

> Status: Draft for review · Spec 07

## Goals

- Make `security.go`'s `LoadTLSConfig` the single, hardened source of TLS configuration for every
  request, and document exactly which knobs it honors (`SecureDefaults`, SHA-256 pinning, SNI,
  min/max version, cipher suites).
- Add an **opt-in SSRF / redirect-to-internal guard** (link-local `169.254.0.0/16`, loopback,
  RFC1918, ULA `fc00::/7`, and cloud metadata endpoints) implemented both as request-time
  middleware and as a redirect pre-flight check, with a configurable allow-list.
- Guarantee **secret redaction everywhere** a credential can surface: verbose output, structured
  logs, and error messages — closing the gap that `verbose.go` and `errors.go` currently maintain
  two divergent sensitive-key lists.
- Fix **proxy auth for the username-only case** (curl `-x http://user@proxy`), which is silently
  dropped today.
- **Wire the runtime input validation that is currently dead code**: the `options` package
  validators (`validateHeaders`, `validateBody`, `validateForm`, `validateQueryParams`,
  forbidden-header check, `validateSecureAuth`) run only through `RequestOptionsBuilder.Validate()`,
  never on the live `Curl*` → `doRequest` path.
- Define the **plaintext-auth-over-HTTP warning policy** (currently `allowInsecureAuth()` is
  hardcoded to `false` and unreachable from the live path).
- Provide a concise, written **threat model** so reviewers know what gocurl defends against and what
  it explicitly delegates to the caller.

## Non-goals

- No HTTP/3 / QUIC security surface (out of scope per the v1 plan).
- No request signing, secret-store integration (Vault/KMS), or credential rotation — gocurl consumes
  credentials the caller supplies.
- No attempt to enforce TLS 1.3 cipher selection: Go's `crypto/tls` does not support it, so
  `opts.TLS13CipherSuites` remains accepted-but-unenforced (documented, not fixed here).
- No mandatory SSRF blocking. The guard is **opt-in** — turning it on by default would break the
  "paste any curl from the docs and it runs" promise. Defaults stay permissive; security is a choice
  the caller makes explicitly.
- Not a sandbox: `-d @file`, `-T file`, `-O`, cert/CA/cookie file paths are read from the local
  filesystem with the caller's privileges. Path confinement is out of scope.

## Design

### 1. TLS configuration — keep one source of truth

`LoadTLSConfig(opts)` in `security.go` already merges `SecureDefaults()` (TLS 1.2 floor, four ECDHE
GCM suites, client cipher preference) with the curl-derived knobs: `--cacert` (`CAFile`),
`--cert`/`--key` (`CertFile`/`KeyFile`), `--pinnedpubkey` (`CertPinFingerprints` via
`VerifyCertificatePin`), SNI (`SNIServerName`), and `--tlsv1.x`/`--tls-max`/`--ciphers`
(`TLSMinVersion`/`TLSMaxVersion`/`CipherSuites`). This spec ratifies that surface and adds three
hardening fixes rather than new TLS API.

```go
// security.go — additions/changes (signatures only)

// SecureDefaults gains an explicit modern profile name so callers and the
// Client (Spec 02) can request it by intent rather than re-deriving suites.
func SecureDefaults() *tls.Config // unchanged default: TLS 1.2 floor

// New: when pinning is configured we currently set InsecureSkipVerify=true and
// rely solely on the pin. Harden so the pin is checked IN ADDITION to chain
// verification unless the caller also passed --insecure.
func VerifyCertificatePin(rawCerts [][]byte, fingerprints []string) error // unchanged
```

Hardening fixes to `LoadTLSConfig`:

- **Pinning must not silently disable chain verification.** Today the pinning branch sets
  `tlsConfig.InsecureSkipVerify = true`. Change it to keep normal verification on and run the pin
  check inside `VerifyConnectionState`/`VerifyPeerCertificate` *after* the standard chain validates;
  only fall back to skip-verify-plus-pin when the caller explicitly set `opts.Insecure`. This makes
  a typo'd pin fail closed rather than trusting any chain.
- **`ValidateTLSConfig` runs on the live path** (see §4). Its existing checks (reject TLS < 1.2,
  reject `InsecureSkipVerify` without `--insecure` or a pin) are good; today they are reachable only
  when `opts.TLSConfig != nil`.
- The TLS 1.3 cipher limitation stays documented in code (the existing `NOTE` comment) and in
  `SECURITY.md`.

### 2. SSRF / redirect-to-internal guard (opt-in)

There is **no SSRF protection in the codebase today** — only prose in `SECURITY.md` telling callers
to "restrict redirect targets at your application layer." We make that enforceable.

```go
// security.go (new) — SSRF guard

type SSRFPolicy struct {
    BlockLoopback     bool     // 127.0.0.0/8, ::1
    BlockLinkLocal    bool     // 169.254.0.0/16, fe80::/10
    BlockPrivate      bool     // RFC1918, fc00::/7 (ULA)
    BlockCloudMetadata bool    // 169.254.169.254, fd00:ec2::254, metadata.google.internal
    AllowHosts        []string // explicit allow-list (host or host:port), checked first
    AllowIPs          []string // explicit allow-list of resolved IPs/CIDRs
}

// DefaultSSRFPolicy returns a policy that blocks loopback, link-local, private,
// and cloud-metadata destinations — the recommended setting for untrusted curl.
func DefaultSSRFPolicy() SSRFPolicy

// CheckSSRF resolves host and rejects the request if any resolved IP is blocked
// by policy and not on the allow-list. Used for pre-flight and per-redirect.
func (p SSRFPolicy) CheckSSRF(ctx context.Context, host string) error
```

Two enforcement points, because a single check is insufficient against DNS rebinding and redirects:

1. **Middleware** (`Middleware`/`Handler` chain from Spec 02/04): a `SSRFGuard(policy)` middleware
   that validates the *initial* target before the request leaves the chain. Wired via the locked
   `WithSSRFGuard(policy)` option.
2. **Redirect pre-flight**: extend the existing `redirectPolicy` closure in `process.go` so each
   hop is re-checked. `redirectPolicy` already enforces `MaxRedirects` and `FollowRedirects`; it
   gains a `policy.CheckSSRF(req.Context(), req.URL.Hostname())` call that returns an error (aborting
   the redirect chain) when the next hop resolves to a blocked address. This catches the classic
   "public URL 302s to 169.254.169.254" attack that the middleware alone cannot see.

Resolution happens against the addresses the dialer will actually use; to close the rebinding gap we
also expose the option (open question below) of pinning the dial to the vetted IP via a custom
`DialContext`. The guard returns a typed, classifiable error (Spec 08) so callers can distinguish a
policy block from a network failure.

### 3. Secret handling & redaction (everywhere)

Today redaction lives in two places with **two different key lists**:

- `errors.go`: `sensitiveHeaders` map + `IsSensitiveHeader` + `RedactHeaders`, plus
  `sanitizeCommand`/`sanitizeURL`/`redactURLParams` (used for error messages and command logging).
- `verbose.go`: a *separate* `isSensitiveHeader` with a *different, shorter* list (e.g. it omits
  `proxy-authorization`, `token`, `secret`, `password`).

```go
// Consolidate: verbose.go drops its private isSensitiveHeader and calls the
// exported errors.go helper, so one list governs logs, verbose, and errors.
func IsSensitiveHeader(name string) bool          // errors.go — the single source
func RedactHeaders(h map[string][]string) ...     // errors.go — unchanged
func RedactURL(raw string) string                 // wraps sanitizeURL for reuse
func RedactCommand(cmd string) string             // wraps sanitizeCommand
```

Redaction coverage required on every sink:

- **Verbose** (`printRequestVerbose`/`printResponseVerbose`): redact request *and* response sensitive
  headers using the shared list. Already redacts request/response headers; switch to the shared list.
- **Errors**: any error string that embeds a URL or command must pass through `RedactURL` /
  `RedactCommand`. The existing `GocurlError` (Spec 08) carries the URL — its `Error()` must redact
  query-string secrets (`api_key`, `token`, `secret`, `key`, `password`).
- **Logs / tracer / metrics** (Spec 03): the logging and OTel adapters receive only
  pre-redacted headers and URLs; raw `*http.Request` headers are never logged directly.
- **Basic-auth userinfo in URLs** (`https://user:pass@host`) must be stripped before logging.

### 4. Wire runtime input validation onto the live path

The live path is `Curl* → doRequest → ValidateOptions → ValidateRequestOptions` (security.go).
`ValidateRequestOptions` validates URL presence, TLS files, timeouts, and redirect/retry counts.
The **richer validators in the `options` package run only through `RequestOptionsBuilder.Validate()`
(`options/builder.go:264`)** and are therefore dead on the path users actually call:
`validateMethod`, `validateHeaders` (count, size, forbidden Host/Content-Length/Transfer-Encoding),
`validateBody`, `validateForm`, `validateQueryParams`, and `validateSecureAuth`.

```go
// options package: expose the bundle so the live path can call it.
func ValidateRequest(opts *RequestOptions) error // runs method/header/body/form/query/secure-auth checks

// security.go: ValidateRequestOptions calls into it after its own checks.
func ValidateRequestOptions(opts *options.RequestOptions) error {
    // ... existing URL/TLS/timeout/redirect checks ...
    return options.ValidateRequest(opts) // NEW: header/body/form/query/forbidden-header/secure-auth
}
```

This makes header-count/size caps (`MaxHeaders`, `MaxHeaderSize`), body/form/query caps, and the
forbidden-header rule enforced for `Curl*` callers and the CLI — not just builder users.

### 5. Plaintext-auth-over-HTTP warning policy

`validateSecureAuth` returns an error when `BasicAuth`/`BearerToken` is sent over `http://`, gated by
`allowInsecureAuth()` — which is **hardcoded `false`** ("Can add os.Getenv check later") and, because
the validator is dead on the live path, never fires for `Curl*` callers.

Policy:

- Wire `validateSecureAuth` onto the live path via §4.
- Implement `allowInsecureAuth()` to honor `GOCURL_ALLOW_INSECURE_AUTH=1` (the env var the error
  message already advertises) **and** a `WithAllowInsecureAuth(true)` option, so the override is real.
- Default behavior is a hard error (fail closed) for credentials over cleartext, matching the
  existing message. The CLI surfaces it as a clear stderr message, consistent with the `--insecure`
  warnings `LoadTLSConfig` already prints.

### 6. Proxy auth, including username-only

`proxy/httpproxy.go` builds the proxy URL and the `Proxy-Authorization` CONNECT header only when
`Username != "" && Password != ""` (`buildProxyURL`, `createConnectRequest`). A curl
`-x http://user@proxy` (username, empty password — valid for many proxies) is **silently dropped**.

```go
// proxy/httpproxy.go — send credentials whenever a username is present.
func (hp *HTTPProxy) buildProxyURL() string // include userinfo if Username != "" (password optional)
func (hp *HTTPProxy) createConnectRequest(addr string) *http.Request // set Proxy-Authorization if Username != ""
```

Change the guard from `Username != "" && Password != ""` to `Username != ""`, encoding
`user:` (empty password) per RFC 7617. `Proxy-Authorization` is already in the shared
`sensitiveHeaders` list, so it is redacted in verbose/logs/errors once §3 lands.

## Behavior & edge cases

- **Pinning + insecure**: `--pinnedpubkey` alone keeps chain verification on (fixed in §1);
  combining with `--insecure` keeps the pin as the sole check and emits the existing insecure
  warning.
- **SSRF allow-list precedence**: `AllowHosts`/`AllowIPs` are checked first; a match short-circuits
  the block. An explicit allow of `127.0.0.1` lets local testing work with the guard enabled.
- **SSRF and redirects**: a 3xx whose `Location` resolves to a blocked IP aborts the redirect chain
  with a classifiable SSRF error, even if the original host was public.
- **DNS rebinding**: middleware-time resolution can differ from dial-time resolution; mitigated only
  if dial-IP pinning is adopted (open question). Documented as a residual risk otherwise.
- **IPv6 forms**: guard normalizes and matches `::1`, `fe80::/10`, `fc00::/7`, and bracketed
  `[::1]:port` host forms; `host:port` is split before resolution.
- **Cloud metadata**: blocked set includes `169.254.169.254` (AWS/Azure/GCP/OpenStack),
  `fd00:ec2::254` (AWS IMDSv6), and `metadata.google.internal`.
- **Redaction false-positives**: a non-secret header literally named `Token`/`Secret`/`Password` is
  redacted by design (fail safe); documented.
- **Validation caps vs. real APIs**: `MaxHeaders=100`, `MaxHeaderSize=8192`, `MaxBodySize=10MB`,
  `MaxURLLength=8192`. Bodies use `ResponseBodyLimit` when set, else the 10MB default. These must not
  reject realistic API docs; if they do, raise the constant rather than special-casing.
- **Forbidden headers**: setting `Host`, `Content-Length`, or `Transfer-Encoding` via `-H` errors on
  the live path once §4 lands — a behavior change from today's silent acceptance; called out in
  CHANGELOG.
- **`--insecure` precedence**: an explicit `opts.Insecure` always wins and prints the existing
  stderr warnings; the SSRF guard and plaintext-auth policy are independent of it.

## Acceptance criteria / Definition of Done

- [ ] Pinning no longer forces `InsecureSkipVerify=true`; a wrong pin against a valid chain fails
      closed. Test: valid cert + bad pin → handshake error; valid cert + good pin → success.
- [ ] `ValidateTLSConfig` and the `options` validators run on the `Curl*`/`doRequest` live path
      (verified by a blackbox test in `tests/` that triggers each rejection through `CurlString`).
- [ ] `WithSSRFGuard(DefaultSSRFPolicy())` blocks a request to `http://127.0.0.1`,
      `http://169.254.169.254/...`, and an RFC1918 host; an allow-listed host passes.
- [ ] A public URL that 302-redirects to `169.254.169.254` is blocked by the redirect pre-flight
      (httptest-based test, hermetic).
- [ ] `verbose.go` no longer defines its own `isSensitiveHeader`; one list (`errors.go`) governs
      verbose, logs, and errors. Test: `Proxy-Authorization`, `Set-Cookie`, `token`, `secret`,
      `password` all redacted in `-v` output.
- [ ] Error messages and logs redact userinfo and `api_key`/`token`/`secret`/`key`/`password`
      query params (unit test on `RedactURL`/`GocurlError.Error()`).
- [ ] `-x http://user@proxy` sends `Proxy-Authorization` / userinfo (test asserts the header is
      present and base64-encodes `user:`).
- [ ] Plaintext auth over `http://` errors by default; `GOCURL_ALLOW_INSECURE_AUTH=1` and
      `WithAllowInsecureAuth(true)` both override it (tests for all three).
- [ ] `SECURITY.md` "Scope and known considerations" updated to describe the opt-in SSRF guard, the
      plaintext-auth policy, and the redaction guarantee, replacing the "handle it at your layer"
      prose for redirects.
- [ ] All new tests pass under `go test -short -race ./...`; no live-network dependency.

## Dependencies

- **Spec 02** (Client / options): `WithSSRFGuard`, `WithAllowInsecureAuth`, `WithTLS`,
  `WithProxy` carry the new config; `New(...)` validates it.
- **Spec 04** (Middleware): `SSRFGuard` is a `Middleware`; redaction-aware logging composes here.
- **Spec 03** (Observability): logger/tracer/metrics consume only pre-redacted headers/URLs.
- **Spec 08** (Errors): SSRF block, TLS, plaintext-auth, and validation failures are typed,
  classifiable `GocurlError`s; `Error()` redaction lives here.

## Open questions / decisions to confirm in review

- **Dial-IP pinning for SSRF**: do we pin the dial to the vetted IP (full rebinding protection,
  custom `DialContext`, interacts with the pooled transport in `clientpool.go`) or accept the
  resolve-then-dial race as a documented residual risk for v1? *Proposed: document the residual risk
  in v1; add dial pinning behind the option later.*
- **SSRF default**: confirm the guard stays fully opt-in (no default-on), to preserve the
  paste-and-run promise. *Proposed: opt-in, with `DefaultSSRFPolicy()` as the recommended setting.*
- **Forbidden-header enforcement**: erroring on `-H Host:` / `-H Content-Length:` is a behavior
  change. Confirm we hard-error (proposed) vs. warn-and-strip.
- **Plaintext-auth default**: confirm hard error (fail closed, proposed) rather than a stderr warning
  that still proceeds. curl itself sends the credentials; we are intentionally stricter.
- **Pin format**: `VerifyCertificatePin` pins the **certificate** SHA-256 (DER of `cert.Raw`), while
  curl's `--pinnedpubkey` pins the **SubjectPublicKeyInfo**. Confirm whether v1 keeps cert pinning,
  switches to SPKI for curl parity, or supports both prefixes.
