# CLI (`cmd/gocurl`)

> Status: Draft for review · Spec 13

This spec owns the behavior of the `gocurl` command-line tool. The CLI is a thin shell over
the library: it parses presentation flags, delegates execution to the default `Client`
(Spec 01), and formats output. Specs 08 (errors) and 02 (flags) defer concrete CLI
behavior here.

## Goals

- **Library/CLI parity:** the same curl syntax works in `gocurl <cmd>` and in
  `gocurl.Curl(ctx, <cmd>)`. The CLI must route through the same parser and Client.
- Stable, curl-compatible **exit codes** and **stream discipline** (stdout = body, stderr =
  diagnostics).
- **Never leak secrets** in any output mode (build on `gocurl.IsSensitiveHeader` /
  `RedactHeaders`).
- A single source of truth for execution: the CLI must not re-implement request logic.

## Non-goals

- Not a full curl CLI clone (HTTP/HTTPS only; the flag matrix is Spec 02's).
- Not an interactive shell, REPL, or config-file system (env vars + flags only).

## Design

The CLI splits arguments into **presentation flags** (consumed by the CLI: `-v`, `-i`,
`-s`, `-o`, `-w`) and **curl args** (passed verbatim to the library), then executes via the
default Client and formats the result. This is the current `cmd/gocurl/main.go`
`separateFlags` + `executeCLI` shape, kept and hardened.

```go
func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// run is the testable entry point (Spec 09 unit-tests it directly with buffers).
func run(args []string, stdout, stderr io.Writer) int
```

Refactoring `main` into `run(args, stdout, stderr) int` is required so the CLI is unit
testable without a subprocess (today only the black-box subprocess tests exercise it; Spec
09 wants in-process coverage too).

### Stream discipline

- **stdout**: response body only (or the `-w`/`-o` formatted output). Because the library
  no longer writes to stdout (fixed this cycle), the CLI is the *sole* writer — no
  double-printing.
- **stderr**: verbose trace (`-v`), warnings (e.g. `--insecure`), and error messages.

### Output modes

| Flag | Behavior |
|------|----------|
| (default) | Body to stdout; pretty-print when `Content-Type` is JSON. |
| `-i` / `--include` | Response status line + headers, then body. Response headers are **data**, shown verbatim. |
| `-v` / `--verbose` | Request + response metadata to **stderr** with sensitive headers redacted; body to stdout. |
| `-s` / `--silent` | Suppress body and non-error diagnostics. |
| `-o FILE` / `--output` | Write body to FILE; nothing to stdout. |
| `-w FMT` / `--write-out` | Append curl-style `%{http_code}`, `%{content_type}`, `%{size_download}` … expansion. |

Redaction (Spec 07) applies to `-v` request/response header dumps. `-i` shows response
headers verbatim (they are the server's data, not the caller's secrets), matching curl.

### Exit codes (curl-compatible subset)

Map the typed error `Kind` (Spec 08) to curl's exit codes rather than string-matching
(the current `getExitCode` matches substrings — replace it):

| Condition | Kind (Spec 08) | Exit |
|-----------|----------------|------|
| Success | — | 0 |
| URL malformed / missing | `KindParse` / `KindValidation` (URL) | 3 |
| Unsupported/unknown flag | `KindParse` | 2 |
| Failed to connect | `KindConnect` | 7 |
| Operation timeout | `KindTimeout` | 28 |
| TLS/SSL problem | `KindTLS` | 35 |
| Too many redirects | `KindServerStatus` (redirect cap) | 47 |
| Generic error | other | 1 |

`--fail` (curl `-f`) opt-in: exit 22 on HTTP ≥ 400 (the library does not error on 4xx/5xx
by default — Spec 08).

## Behavior & edge cases

- Unknown **presentation** flag vs unknown **curl** flag must produce distinct, clear
  messages; a single unsupported curl flag should fail fast with exit 2 and name the flag.
- `-o` and `-s` together: file written, stdout silent.
- `-v` to stderr must still allow piping the clean body on stdout (`gocurl -v url | jq`).
- Binary bodies: do not pretty-print or corrupt; only attempt JSON formatting when
  `Content-Type` is JSON and the body parses.
- Environment variables (`$VAR`) are expanded by the library's default (env-expanding) path
  for the CLI, matching today's behavior.

## Acceptance criteria / Definition of Done

- [x] `main` refactored to `run(args, stdout, stderr) int`; unit tests drive it in-process
      (Spec 09) raising `cmd/gocurl` line coverage from ~45% to 97.3%.
- [x] Exit codes derived from `Kind` (Spec 08), not string matching; table test per Kind.
      Parse/tokenize/convert failures are now typed `ParseError` (KindParse), so an unknown flag
      exits 2 by classification.
- [x] `-v` redaction test asserts `Authorization`/`Cookie` never appear in cleartext on any
      stream (black-box + in-process assertions).
- [x] Body printed exactly once across all flag combinations (regression test kept + in-process
      matrix).
- [x] `-o`/`-s`/`-i`/`-w` behaviors covered; `--fail` opt-in implemented and tested (exit 22).
- [x] Library/CLI parity test: the same command produces the same request on the wire via
      `gocurl.Curl` and the CLI argv.

## Dependencies

- Spec 01 (default Client), Spec 02 (flag matrix + `separateFlags` presentation flags),
  Spec 07 (redaction), Spec 08 (error `Kind` → exit codes), Spec 09 (in-process tests).

## Open questions / decisions to confirm in review

- **`--fail` / `-f`:** implement curl's fail-on-HTTP-error in v1, or defer? *Proposed:
  implement; it is commonly used in CI scripts.*
- **`-w` format coverage:** which `%{…}` variables for v1 (http_code, content_type,
  size_download, time_total, num_redirects)? *Proposed: the first three now, time/redirect
  counts when the observability hooks (Spec 06) expose them.*
- **Config/rc file:** out of scope for v1? *Proposed: yes, out of scope.*
