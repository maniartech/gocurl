# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project aims to
follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html) once it reaches
a tagged release.

## [Unreleased]

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
