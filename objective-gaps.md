# Objective Gap Analysis

## Priority Framework

- **P0 – Blockers:** Prevent the project from meeting the published objectives or make the advertised workflow impossible.
- **P1 – High Priority:** Significant capability gaps that undermine the core value proposition but do not immediately break execution.
- **P2 – Medium Priority:** Important enhancements or hard promises in documentation that require delivery to build trust.
- **P3 – Low Priority:** Quality-of-life or stretch goals that can follow once higher priorities are addressed.

## P0 – Blockers

| Gap | Evidence | Impact | Immediate Actions |
| --- | --- | --- | --- |
| Curl string conversion is broken | `convertTokensToRequestOptions` sets method to the flag token, never reads the following value, and operates on `Headers/Form` that are `nil`; `convert_test.go` fails accordingly. | Copy-paste curl workflow fails and panics, invalidating the primary promise. | 1) Fix token iteration to consume values correctly and respect token types. 2) Initialize `RequestOptions` maps before use. 3) Restore failing tests and add regression coverage for string inputs. |
| No working CLI | `cmd/` hosts an unrelated USDA tool; there is no `cmd/gocurl` binary despite README/objectives. | “Test with CLI, then copy to Go” can’t be executed. | 1) Scaffold an actual CLI package invoking shared execution pipeline. 2) Provide parity tests/examples for CLI-to-code roundtrip. |
| Documentation oversells zero-allocation guarantees | Objective/README claim “zero-allocation, net/http replacement”, yet `process.go` uses `ioutil.ReadAll`, `bytes.Buffer`, alloc-heavy structs, and no pooling. | Marketing promises risk eroding trust and set incorrect performance expectations. | 1) Either scale back claims immediately or 2) implement measurable zero-allocation architecture (pooled buffers, streaming, benchmarks) before release. |

## P1 – High Priority

| Gap | Evidence | Impact | Planned Actions |
| --- | --- | --- | --- |
| Missing high-level API (`Request`, `Variables`, JSON helpers) | README references `gocurl.Request`, `ParseJSON`, `Variables`, but repo exposes only low-level `Process` and broken conversion helpers. | Users cannot follow documented usage patterns. | Implement public API surface that wraps tokenization + processing, with typed variable substitution and helper utilities. |
| Variable substitution limited to environment only | Tokenizer labels variables, but `convertTokensToRequestOptions` merely calls `os.ExpandEnv`. | Cannot inject runtime variables or secrets safely as promised. | Introduce explicit substitution map (`map[string]string`) with escaping rules and CLI parity. |
| Retry logic reuses exhausted request bodies | `ExecuteRequestWithRetries` resubmits the same request without rewinding body readers. | Retried POST/PUT calls can fail silently or send empty bodies. | Add request cloning with buffered bodies or rewindable readers; guard with tests. |

## P2 – Medium Priority

- **Compression handling:** Transport sets `DisableCompression` inverse to expectation; no brotli support despite documentation.
- **Proxy support:** Tests reveal HTTPS proxying via CONNECT isn’t implemented; SOCKS5 path is incomplete.
- **Security posture:** No sensitive-data redaction, limited auth mechanisms beyond Basic/Bearer, TLS CA pool handling incomplete.
- **Streaming and large payloads:** All responses buffered into memory; no streaming helpers or limits.
- **Documentation accuracy:** README still contains legacy sections (plugins, struct generation) with no backing code.

## P3 – Low Priority

- Benchmark suite and performance dashboards.
- CLI polish (formatters, verbose/debug UX).
- Advanced auth (OAuth flows) and middleware catalog.
- Generated wrappers for popular APIs.

## Remediation Roadmap

1. **Stabilize Core Workflow (P0)**
   - Repair token conversion and initialize request options correctly.
   - Introduce end-to-end tests (`Curl` string → HTTP roundtrip) covering headers, data, forms, cookies.
   - Stand up the real `cmd/gocurl` entrypoint using shared logic.

2. **Align Public API with Docs (P0/P1)**
   - Ship `Request`, `RequestFromString`, and `Variables` interfaces matching README samples.
   - Harden variable substitution (map-based, test coverage for escaping) and update docs simultaneously.

3. **Clarify Performance Positioning (P0/P2)**
   - Update objectives immediately to state current performance status.
   - In parallel, scope true low-allocation architecture: pooled buffers, streaming, targeted benchmarks.

4. **Strengthen Reliability Features (P1/P2)**
   - Fix retry body rewind, adjust compression defaults, finalize proxy CONNECT/SOCKS5 implementations.
   - Add security-focused validations (sensitive header masking, TLS root CA handling, more auth hooks).

5. **Documentation & Evangelism (P2/P3)**
   - Rewrite README/examples post-implementation with realistic snippets.
   - Publish roadmap notes capturing remaining stretch goals (benchmarks, auto-generated wrappers, advanced CLI UX).

6. **Ongoing Quality Gates**
   - Keep `go test ./...` green and expand coverage alongside each fix.
   - Add benchmarks and integration tests before reasserting performance claims.
