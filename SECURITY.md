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
- **Redirects.** Following redirects (`-L`) to attacker-controlled targets can
  enable SSRF. If you build requests from untrusted input, restrict redirect
  targets at your application layer.
- **Verbose output.** `-v` redacts sensitive headers (Authorization, Cookie, API
  keys). Report any path that leaks credentials in cleartext.
- **TLS.** `-k`/`--insecure` disables certificate verification and should only be
  used intentionally.

Reports that improve any of the above are welcome.
