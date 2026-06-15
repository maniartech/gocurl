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

### Added
- `doc.go` package documentation, `CONTRIBUTING.md`, `SECURITY.md`, this changelog,
  and a CI workflow (gofmt, vet, build, race tests).

### Fixed / Hardening (in progress)
- Isolated the `book2/` example programs and `scripts/` helpers into their own Go
  modules so `go build/vet/test ./...` covers only the library and CLI.
- Removed stale `*.go.old` files; normalized formatting (`gofmt`, LF line endings).
