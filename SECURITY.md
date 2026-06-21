# Security Policy

GoCurl is an HTTP client that handles credentials, tokens, TLS configuration, and
proxies, so we take security reports seriously.

## Reporting a vulnerability

Please **do not** open a public issue for security vulnerabilities.

Instead, report privately via GitHub's
[private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability)
on this repository, or email the maintainers.

Please include:

- A description of the issue and its impact.
- Steps to reproduce (a minimal curl command or code snippet is ideal).
- The GoCurl version / commit and Go version.

We aim to acknowledge reports within a few business days.

## Scope and known considerations

GoCurl executes curl-syntax commands, which has security-relevant behavior worth
understanding:

- **Variable expansion.** The default `Curl`/`CurlCommand` entry points expand
  `$VAR`/`${VAR}` from the process environment. When executing curl strings from
  untrusted sources, use the `*WithVars` entry points with an explicit
  `Variables` map so the process environment is not consulted.
- **SSRF / redirects.** An opt-in SSRF guard (`WithSSRFGuard(DefaultSSRFPolicy())`)
  blocks requests — and every redirect hop — whose host resolves to a loopback,
  link-local, RFC1918/ULA private, the unspecified address (`0.0.0.0`/`::`), or a
  cloud-metadata (`169.254.169.254`, `fd00:ec2::254`, `metadata.google.internal`,
  trailing FQDN dot normalized) address, unless explicitly allow-listed
  (`AllowHosts`/`AllowIPs`). The guard is opt-in to preserve the paste-any-curl
  promise; enable it for untrusted input. Residual risk: resolution happens at
  check time, so DNS rebinding between the check and the dial is not fully closed
  in this version (dial-IP pinning is a documented future addition).
- **Plaintext auth.** Sending BasicAuth or a bearer token over cleartext `http://`
  fails closed by default. Override deliberately with `WithAllowInsecureAuth(true)`
  or `GOCURL_ALLOW_INSECURE_AUTH=1`.
- **Redaction.** Verbose output, structured logs, span attributes, and error
  messages all route through one redaction path: sensitive headers
  (Authorization, Cookie, Proxy-Authorization, API keys, `x-auth-token`,
  `x-amz-security-token`, `x-csrf-token`, …), basic-auth userinfo in URLs, curl
  credential flags (`-u`/`--user`, `-b`/`--cookie`), and sensitive query parameters
  (`api_key`/`token`/`secret`/`key`/`password`, every occurrence) are stripped.
  Report any path that leaks a credential in cleartext.
- **TLS.** Secure defaults (TLS 1.2 floor) apply unless the caller downgrades via
  `--tlsv1.x`. Certificate pinning (`CertPinFingerprints`) is checked **in addition
  to** chain verification — a wrong pin against a valid chain fails closed — unless
  the caller also passes `-k`/`--insecure`, which disables certificate verification
  and should only be used intentionally.
- **Input validation.** The live request path enforces method-token validity,
  header count/size caps, a forbidden-header rule (`Host`/`Content-Length`/
  `Transfer-Encoding` are managed automatically and rejected if set), and
  body/form/query-count caps. Streaming bodies are exempt from the in-memory body
  cap.
- **Parse-time file reads.** Curl flags that read a file into memory at parse time
  — `-d @file` / `--data[-binary] @file`, `--data-urlencode name@file`, and
  `-b <cookiefile>` — are bounded (data files at the 10 MiB in-memory body cap,
  cookie files at 256 KiB) so an untrusted command string pointing at a huge or
  endless path (e.g. `-b /dev/zero`) fails closed instead of exhausting memory.
  Streaming `-T`/`-F @file` uploads are read at execution and not buffered, so they
  are unaffected.

Reports that improve any of the above are welcome.

## Threat model

This makes the trust boundaries explicit so you can reason about gocurl in a
mission-critical deployment. Each control below is backed by an un-skipped test (cited),
enforced by the honesty doc-lint.

### Assets

- **Credentials** — bearer tokens, basic-auth userinfo, cookies, API keys, proxy creds.
- **Process integrity & availability** — memory, file descriptors, goroutines.
- **The request target** — the service must not be coerced into reaching an unintended host.

### Trust boundaries & attacker models

1. **Untrusted *server*** (you call an endpoint you do not control). The server may stall,
   reset, send a premature EOF, a decompression bomb, or a flood of response headers.
   - *Defended:* response-header cap (1 MiB), buffered-read cap (64 MiB) against bombs
     (`TestFault_BufferingHelpersBoundedAgainstBomb`), overall timeout bounds a stall
     (`TestFaultT2_ResponseHeaderTimeout`), premature EOF surfaces as `KindBodyRead`
     (`TestFaultT2_PrematureBodyEOF`), and credentials never leak in the resulting error
     (`TestFault_NoSecretLeakOnFailurePaths`).

2. **Untrusted *command string / URL*** (a curl string or URL influenced by user input).
   - *Defended:* the opt-in SSRF guard blocks internal/metadata targets on the request
     **and every redirect hop**; plaintext auth fails closed; `*WithVars` avoids the
     process environment; parse-time file reads are bounded. *Residual:* DNS rebinding
     between SSRF check and dial is not fully closed (dial-IP pinning is future work);
     the guard is **opt-in**, so you must enable it for untrusted input.

3. **Untrusted *network path*** (MITM between client and server).
   - *Defended:* TLS 1.2 floor, optional certificate pinning checked in addition to chain
     verification. *Residual:* `-k`/`--insecure` disables verification by explicit caller
     choice.

4. **Buggy *caller code* under load** (panicking middleware, undrained bodies, abandoned
   requests).
   - *Defended:* a panicking middleware still releases the in-flight count so `Shutdown`
     drains (`TestFaultT2_PanicMiddlewareDoesNotWedgeShutdown`); a graceful `Shutdown`
     waits for in-flight streams instead of truncating them
     (`TestFault_ShutdownWaitsForOpenBody`); no goroutine/connection leak under a fault
     storm or sustained soak (`TestFault_NoGoroutineLeakUnderStorm`, `TestClient_Soak`).

### Explicitly out of scope

- gocurl is an HTTP/HTTPS client; non-HTTP curl protocols (FTP, SMTP, …) are not implemented.
- It does not sandbox the *content* of responses you choose to execute or deserialize.
- It is not a WAF or an egress firewall; the SSRF guard is a defense-in-depth control, not a
  replacement for network-level egress policy.
