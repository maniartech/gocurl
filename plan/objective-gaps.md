# Objective Gap Analysis

## Design Philosophy: Sweet, Simple, Robust (SSR)

This project follows three core principles to avoid over-engineering while delivering military-grade performance:

- **Sweet**: Developer-friendly API with minimal cognitive load - copy-paste curl commands and they just work
- **Simple**: No over-engineering, clear data flow, maintainable codebase with focused components
- **Robust**: Zero-allocation on critical paths, military-grade reliability, comprehensive error handling, thread-safe by design, race-free concurrent execution, battle-tested under extreme load

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

## Implementation Strategy: SSR Approach

### Zero-Allocation Strategy (Robust Without Over-Engineering)

**Critical Path (Must be zero-alloc):**
- Request option parsing - reuse token arrays from pool
- Header construction - pre-allocated maps, no dynamic growth
- URL building - pooled string builders

**Acceptable Allocations (Off critical path):**
- Response bodies - read once, smart buffering strategy
- Error messages - exceptional cases only
- TLS handshakes - infrequent, unavoidable

**Pooling Strategy (Simple, focused):**
```go
// Pool ONLY small, frequently-used objects
var (
    tokenArrayPool   = sync.Pool{...} // []Token - reused constantly
    stringBuilderPool = sync.Pool{...} // strings.Builder - URL/header building
    bufferPool       = sync.Pool{...} // bytes.Buffer - request bodies
)

// DON'T pool:
// - Large buffers (memory waste)
// - Complex structs (maintenance burden)  
// - Rarely-used objects (pool overhead > benefit)
```

### Architecture Principles (Simple, Clear)

**Component Separation:**
```
User API Layer (api.go)
    ↓
Parser Layer (parser/, tokenizer/)
    ↓  
Converter Layer (convert.go - FIXED)
    ↓
Execution Layer (request.go, response.go, client.go)
    ↓
Transport Layer (net/http with pooling)
```

**Error Handling Philosophy:**
1. Fail fast - invalid input = immediate error
2. Clear messages - "missing value for -X flag" not "nil pointer"
3. No silent failures - all errors propagate up
4. Include context - show command snippet in errors

### File Organization (Keep It Simple)

**New Files (Minimal additions):**
- `api.go` - High-level public API (Request, Execute, Variables)
- `response.go` - Response wrapper with convenience methods
- `variables.go` - Map-based variable substitution (secure)
- `errors.go` - Structured error types
- `pools.go` - Buffer/object pools (focused, not over-engineered)
- `cmd/gocurl/main.go` - CLI tool using same code path

**Enhanced Files:**
- `convert.go` - FIX token iteration logic, initialize maps
- `process.go` - Refactor to use new components
- `client.go` - Add simple client pooling
- `options/options.go` - Add helper methods

**Keep Unchanged:**
- `tokenizer/` - Already works well
- `middlewares/` - Good design
- `options/` - Solid foundation

## Remediation Roadmap

### Phase 1: Foundation (P0 Blockers) - Week 1

**Goal:** Make documented examples actually work

**Tasks:**
1. **Fix `convert.go` token iteration** (P0 - Critical)
   - Initialize all maps/slices in RequestOptions (Headers, Form, QueryParams)
   - Fix flag parsing: consume NEXT token for value, not current token
   - Correct variable expansion before iteration
   - Add validation for required URL

2. **Create high-level API** (`api.go`)
   - `Request(command, vars)` - accepts string or []string
   - `Execute(opts)` - direct execution with RequestOptions
   - `Variables` type - map[string]string for safe substitution
   - `Response` wrapper - convenience methods for String(), JSON(), Bytes()

3. **Implement map-based variable substitution** (`variables.go`)
   - Replace os.ExpandEnv with controlled map lookup
   - Support ${var} and $var syntax
   - Handle escaping: \${var} becomes literal
   - Error on undefined variables (fail-fast security)

4. **Build working CLI** (`cmd/gocurl/main.go`)
   - Use same code path as library (zero duplication)
   - Parse os.Args, call gocurl.Request()
   - Auto-populate variables from environment for CLI context
   - Handle output formatting (pretty JSON, verbose mode)

5. **Restore failing tests**
   - Fix all convert_test.go failures
   - Add regression tests for string input
   - Verify documented examples work

**Success Criteria:** 
- All tests pass
- `gocurl -X POST -d "key=value" https://example.com` works
- Examples from README execute without errors
- CLI and library use identical syntax

### Phase 2: Zero-Allocation Core (P1) - Week 2

**Goal:** Achieve true zero-allocation on critical path

**Tasks:**
1. **Implement buffer pools** (`pools.go`)
   - sync.Pool for []Token arrays
   - sync.Pool for strings.Builder (URL construction)
   - sync.Pool for bytes.Buffer (request bodies < 64KB)

2. **Create zero-alloc request builder** (`request.go`)
   - Reuse pooled buffers for header building
   - Stream large bodies instead of buffering
   - Pre-allocate header maps based on common sizes

3. **Add client pooling** (enhance `client.go`)
   - Cache configured clients by config hash
   - Reuse connections via http.Transport pooling
   - Background cleanup of idle clients

4. **Benchmark and verify**
   - Write benchmarks proving zero-alloc on request path
   - Compare vs raw net/http baseline
   - Memory profiling to verify no leaks
   - Document allocation strategy

**Success Criteria:**
- Benchmarks show 0 allocs/op on request construction
- Performance matches or beats net/http
- Memory usage stays flat under load
- Documentation explains allocation strategy

### Phase 3: Reliability Features (P1/P2) - Week 3

**Goal:** Handle edge cases and production scenarios

**Tasks:**
1. **Fix retry logic** (create `retry.go`)
   - Clone request with rewindable body
   - Buffer body for retries or use io.Seeker
   - Apply retry delays correctly
   - Log attempts in verbose mode

2. **Smart response handling** (`response.go`)
   - Read body once, cache bytes
   - Use pooled buffers for responses < 1MB
   - Stream large responses to avoid memory spikes
   - Provide String(), JSON(), Bytes() helpers

3. **Comprehensive error types** (`errors.go`)
   - Structured errors with operation context
   - Include command snippet in error messages
   - Classify errors (parse, network, server)
   - Clear, actionable error messages

4. **Security hardening**
   - Redact sensitive headers in logs (Authorization, Cookie)
   - Validate TLS configurations
   - Sanitize variable substitution
   - Document security best practices

**Success Criteria:**
- Retries work correctly for POST/PUT with bodies
- Large file uploads/downloads work efficiently
- Clear error messages for all failure modes
- Security audit passes

### Phase 4: Complete Feature Set (P2) - Week 4

**Goal:** Full curl HTTP/HTTPS feature parity

**Tasks:**
1. **Proxy support** (complete `proxy/`)
   - HTTP proxy with CONNECT for HTTPS
   - SOCKS5 proxy implementation
   - Proxy authentication
   - No-proxy domain exclusions

2. **Compression handling**
   - Fix DisableCompression logic (currently inverted)
   - Add brotli support
   - Handle Accept-Encoding correctly
   - Transparent decompression

3. **Complete TLS support**
   - Client certificates
   - Custom CA bundles
   - Certificate pinning options
   - SNI support

4. **Cookie management**
   - Cookie jar persistence
   - Parse cookies from files
   - Send cookies correctly
   - Handle cookie expiry

**Success Criteria:**
- All curl HTTP flags implemented
- Proxy scenarios tested
- TLS configurations validated
- Cookie handling matches curl behavior

### Phase 5: Polish & Release (P2/P3) - Week 5

**Goal:** Professional documentation and v1.0 release

**Tasks:**
1. **Update documentation**
   - Rewrite README with working examples
   - Document all flags and options
   - CLI help and usage guide
   - Migration guide from net/http

2. **Performance benchmarks**
   - Comparison vs net/http
   - Comparison vs other HTTP libraries
   - Memory usage charts
   - Throughput benchmarks

3. **Example library**
   - Stripe API integration example
   - GitHub API example
   - AWS signature example
   - Generic REST API patterns

4. **Testing completeness**
   - 80%+ code coverage
   - Integration tests with real servers
   - Concurrent usage tests
   - Stress tests

**Success Criteria:**
- Professional documentation
- Benchmark results published
- Example library demonstrates value
- Ready for v1.0 release

## Design Decisions (Avoiding Over-Engineering)

### What We're NOT Building

❌ **Custom HTTP protocol implementation** - Use net/http transport
❌ **Complex state machines** - Simple sequential processing
❌ **Plugin architecture** - Middleware is sufficient
❌ **Auto-generated SDKs** - Focus on consumption, not generation
❌ **Non-HTTP curl features** - No FTP, SMTP, etc.
❌ **Perfect curl parity** - Only HTTP-relevant flags
❌ **GUI/web interface** - CLI and library only

### Key Design Choices

**Why map-based variables instead of environment?**
→ Security and control. Map is explicit, testable, and safe. Environment is for CLI only.

**Why pool buffers but not entire requests?**
→ Buffers are uniform and frequently reused. Requests are varied and complex.

**Why separate API layer from execution?**
→ Testability, clarity, backward compatibility, and clean separation of concerns.

**Why not support all 100+ curl flags?**
→ Focus on HTTP/HTTPS excellence. Rarely-used flags add complexity without value.

**Why read response body once and cache?**
→ Simple semantics, predictable behavior, avoids "body already read" errors.

## Ongoing Quality Gates

**Before Each PR:**
- [ ] All tests pass (`go test ./...`)
- [ ] **Race detector clean** (`go test -race ./...`)
- [ ] Benchmarks don't regress (< 5% degradation)
- [ ] Code coverage maintained or improved
- [ ] Documentation updated for new features
- [ ] Examples updated and tested
- [ ] Static analysis passes (`go vet`, `golangci-lint`)

**Before Each Release:**
- [ ] Full integration test suite passes
- [ ] **All tests pass with -race flag**
- [ ] Performance benchmarks meet targets
- [ ] **Load tests pass** (10k req/s sustained)
- [ ] **Stress tests pass** (graceful degradation verified)
- [ ] **Fuzz tests pass** (100M+ iterations)
- [ ] Security review completed
- [ ] Documentation reviewed
- [ ] CHANGELOG updated
- [ ] **Soak test passes** (24-hour stability)

**Continuous Monitoring:**
- Memory leak detection in long-running tests
- Allocation profiling on critical paths
- **Race condition detection** in all CI runs
- Error rate tracking in integration tests
- Performance regression detection
- **Concurrent load testing** in nightly builds
- **Benchmark regression alerts** (>5% degradation)

**Thread-Safety Requirements:**
- All public APIs must be safe for concurrent use
- Shared state must be protected with proper synchronization
- All tests must pass with `-race` flag
- Client pool must handle concurrent access
- Buffer pools must be thread-safe
- No data races allowed (zero tolerance)
- Concurrent stress tests required for critical paths

**Load Testing Requirements:**
- Sustained throughput: 10k req/s for 1 hour minimum
- Burst handling: 100k req/s for 10 seconds
- Concurrent clients: 100k simultaneous requests
- Memory stability: No growth over 24-hour soak test
- Breaking point identified and documented
- Graceful degradation at 10x normal load
- Recovery from overload verified
