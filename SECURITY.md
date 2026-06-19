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

Reports that improve any of the above are welcome.
