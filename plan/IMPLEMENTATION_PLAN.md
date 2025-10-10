# GoCurl Implementation Plan - SSR Approach

## Executive Summary

This document outlines the implementation strategy for GoCurl using the **Sweet, Simple, Robust (SSR)** philosophy. The goal is to deliver a zero-allocation, military-grade HTTP client library without over-engineering### Week 5: Load Testing & Release (P2/P3)

**Goal**: Battle-test and release production-ready v1.0

**Tasks**:

1. **Update documentation**
   - Rewrite README with working examples
   - Document all flags and options
   - CLI help and usage guide
   - Migration guide from net/http
   - Thread-safety guarantees documented

2. **Performance benchmarks**
   - Comparison vs net/http
   - Comparison vs other HTTP libraries
   - Memory usage charts
   - Throughput benchmarks
   - Concurrent benchmarks
   - Regression tests in CI

3. **Load testing suite** (`load_test.go`)
   ```go
   // Sustained throughput test
   func TestSustainedLoad(t *testing.T) {
       // 10k req/s for 1 hour
   }
   
   // Burst handling test
   func TestBurstLoad(t *testing.T) {
       // 100k req/s for 10 seconds
   }
   
   // Soak test
   func TestSoakTest(t *testing.T) {
       // 24-hour continuous operation
       // Monitor memory growth
   }
   
   // Concurrent clients
   func TestConcurrentClients(t *testing.T) {
       // 100k simultaneous requests
   }
   ```

4. **Stress testing** (`stress_test.go`)
   ```go
   // Find breaking point
   func TestBreakingPoint(t *testing.T) {
       // Incrementally increase load until failure
       // Document maximum concurrent requests
   }
   
   // Resource exhaustion
   func TestResourceExhaustion(t *testing.T) {
       // Test file descriptor limits
       // Test memory limits
       // Verify graceful degradation
   }
   ```

5. **Chaos testing** (`chaos_test.go`)
   ```go
   // Network failures
   func TestRandomFailures(t *testing.T) {
       // Random connection drops
       // Timeouts
       // Partial responses
   }
   
   // Slow responses
   func TestSlowServer(t *testing.T) {
       // Delayed responses
       // Slow reads/writes
   }
   ```

6. **Fuzz testing**
   ```go
   func FuzzCommandParser(f *testing.F) {
       // 100M+ random inputs
       // Verify no crashes
   }
   
   func FuzzVariableSubstitution(f *testing.F) {
       // Edge cases in ${var} expansion
   }
   ```

7. **Example library**
   - Stripe API integration
   - GitHub API example
   - AWS signature example

**Success Criteria**:
- Professional documentation
- Benchmark results published
- **10k req/s sustained for 24 hours**
- **100k concurrent requests handled**
- **Zero race conditions in all tests**
- **Fuzz tests pass 100M+ iterations**
- **Breaking point documented**
- Load/stress/chaos test reports published
- Ready for v1.0 releaseosophy

### Sweet (Developer Experience)
- Copy-paste curl commands from API docs
- Identical CLI-to-code syntax
- Clear, actionable error messages
- Convenience methods for common tasks
- No surprises, predictable behavior

### Simple (Implementation)
- No over-engineering or premature optimization
- Clear data flow: Parse → Convert → Execute → Respond
- Each component has one clear purpose
- Minimal dependencies, leverage stdlib
- Maintainable for new contributors

### Robust (Performance & Reliability)
- Zero-allocation on critical paths
- Military-grade error handling
- Smart resource management
- Secure by default
- Production-ready quality

## Current State Analysis

### Critical Issues (P0 Blockers)

1. **Broken curl string conversion** (`convert.go`)
   - Token iteration reads flag instead of consuming next value
   - Never initializes Headers/Form maps (nil pointer panics)
   - Variable expansion happens but uses wrong tokens
   - **Impact**: Copy-paste workflow completely broken

2. **No working CLI**
   - `cmd/` contains unrelated USDA tool
   - No `cmd/gocurl` binary despite documentation
   - **Impact**: "Test with CLI, then copy to Go" is impossible

3. **Missing high-level API**
   - README references `gocurl.Request()`, `Variables`, `ParseJSON()`
   - Only low-level `Process()` and broken conversion exists
   - **Impact**: Users can't follow documented examples

4. **Documentation oversells zero-allocation**
   - Claims "zero-allocation, net/http replacement"
   - Code uses `ioutil.ReadAll`, `bytes.Buffer`, no pooling
   - **Impact**: Marketing promises erode trust

### High Priority Issues (P1)

1. **Variable substitution limited to environment only**
   - Uses `os.ExpandEnv` - can't inject runtime values securely
   - Need map-based substitution with escaping

2. **Retry logic broken**
   - Reuses exhausted request bodies
   - POST/PUT retries fail silently

3. **Compression handling inverted**
   - `DisableCompression` set opposite of expectation

## Implementation Strategy

### Week 1: Foundation (P0 Blockers)

**Goal**: Make documented examples actually work

**Tasks**:

1. **Fix `convert.go` token iteration**
   ```go
   // BEFORE (broken)
   o.Method = token  // Uses flag, not value!
   
   // AFTER (correct)
   case "-X", "--request":
       i++ // Move to next token
       if i >= len(expandedTokens) {
           return nil, fmt.Errorf("missing value for %s", token.Value)
       }
       o.Method = expandedTokens[i].Value // Use NEXT token
   ```

2. **Initialize all maps**
   ```go
   o := options.NewRequestOptions("")
   o.Headers = make(http.Header)      // FIX: Initialize
   o.Form = make(url.Values)          // FIX: Initialize
   o.QueryParams = make(url.Values)   // FIX: Initialize
   ```

3. **Create high-level API** (`api.go`)
   ```go
   type Variables map[string]string
   
   func Request(command interface{}, vars Variables) (*Response, error)
   func Execute(opts *RequestOptions) (*Response, error)
   
   type Response struct {
       *http.Response
       bodyBytes []byte
   }
   
   func (r *Response) String() (string, error)
   func (r *Response) JSON(v interface{}) error
   func (r *Response) Bytes() ([]byte, error)
   ```

4. **Implement map-based variables** (`variables.go`)
   ```go
   // Replace os.ExpandEnv with controlled substitution
   func ExpandVariables(text string, vars Variables) (string, error)
   // Support ${var} and $var
   // Handle escaping: \${var} becomes literal
   // Error on undefined variables (security)
   ```

5. **Build working CLI** (`cmd/gocurl/main.go`)
   ```go
   func main() {
       command := strings.Join(os.Args[1:], " ")
       vars := envToVariables() // Auto-populate from env
       resp, err := gocurl.Request(command, vars)
       // Format and print
   }
   ```

**Success Criteria**:
- All `convert_test.go` tests pass
- CLI works: `gocurl -X POST -d "key=value" https://example.com`
- README examples execute without errors
- CLI and library use identical syntax

### Week 2: Zero-Allocation Core (P1)

**Goal**: Achieve true zero-allocation on critical path

**Tasks**:

1. **Buffer pools** (`pools.go`)
   ```go
   var (
       tokenArrayPool   = sync.Pool{New: func() interface{} {
           return make([]Token, 0, 32)
       }}
       stringBuilderPool = sync.Pool{New: func() interface{} {
           return &strings.Builder{}
       }}
       bufferPool = sync.Pool{New: func() interface{} {
           return bytes.NewBuffer(make([]byte, 0, 4096))
       }}
   )
   ```

2. **Zero-alloc request builder** (`request.go`)
   - Reuse pooled string builders for URL construction
   - Pre-allocate header maps
   - Stream large bodies instead of buffering

3. **Client pooling** (enhance `client.go`)
   ```go
   var clientPool = &sync.Map{} // key: config hash
   
   func GetClient(opts *RequestOptions) (*http.Client, error) {
       hash := hashConfig(opts)
       if client, ok := clientPool.Load(hash); ok {
           return client.(*http.Client), nil
       }
       // Create new and cache
   }
   ```

4. **Benchmark verification**
   ```go
   BenchmarkRequestConstruction
   BenchmarkVsNetHTTP
   BenchmarkConcurrent
   ```

**Success Criteria**:
- 0 allocs/op on request construction path
- Performance matches or beats net/http
- Memory stays flat under load

### Week 3: Reliability Features (P1/P2)

**Goal**: Production-ready error handling and concurrent safety

**Tasks**:

1. **Fix retry logic** (`retry.go`)
   ```go
   func cloneRequest(req *http.Request) (*http.Request, error) {
       // Buffer body or use io.Seeker
       // Clone headers
       // Return rewindable request
   }
   ```

2. **Smart response handling** (`response.go`)
   - Read body once, cache bytes
   - Pool buffers for small responses (<1MB)
   - Stream large responses

3. **Structured errors** (`errors.go`)
   ```go
   type GocurlError struct {
       Op      string // "parse", "request", "response"
       Command string // Command snippet
       Err     error
   }
   ```

4. **Security hardening**
   - Redact sensitive headers in logs
   - Validate TLS configurations
   - Sanitize variable substitution

5. **Thread-safety verification** (`race_test.go`)
   ```go
   func TestConcurrentRequests(t *testing.T) {
       const numGoroutines = 10000
       var wg sync.WaitGroup
       
       for i := 0; i < numGoroutines; i++ {
           wg.Add(1)
           go func() {
               defer wg.Done()
               _, err := gocurl.Request("curl https://example.com", nil)
               // Verify no races
           }()
       }
       wg.Wait()
   }
   
   func TestConcurrentClientPool(t *testing.T) {
       // Test simultaneous client pool access
       // Verify sync.Map operations are race-free
   }
   
   func TestConcurrentBufferPool(t *testing.T) {
       // Stress test buffer pool Get/Put
       // Ensure no data races in pooling
   }
   ```

6. **Race detection in CI**
   ```yaml
   # .github/workflows/test.yml
   - name: Race Detection
     run: go test -race -v ./...
   
   - name: Concurrent Stress Test
     run: go test -race -run=Concurrent -v ./...
   ```

**Success Criteria**:
- Retries work for POST/PUT with bodies
- Large files handled efficiently
- Clear error messages
- Security audit passes
- **All tests pass with `go test -race ./...`**
- **10k concurrent requests execute without data races**
- **Client pool handles concurrent access correctly**

### Week 4: Complete Feature Set (P2)

**Goal**: Full HTTP curl feature parity

**Tasks**:

1. Proxy support (HTTP/HTTPS/SOCKS5)
2. Fix compression handling
3. Complete TLS support
4. Cookie jar persistence
5. All HTTP-relevant curl flags

**Success Criteria**:
- All curl HTTP flags implemented
- Comprehensive test coverage
- Feature parity with objectives

### Week 5: Polish & Release (P2/P3)

**Goal**: Professional documentation and v1.0

**Tasks**:

1. Rewrite README with working examples
2. Performance benchmarks published
3. Example library (Stripe, GitHub, AWS)
4. Migration guide from net/http
5. v1.0 release

**Success Criteria**:
- Professional documentation
- Published benchmarks
- Ready for production use

## File Organization

### New Files
```
api.go                  - High-level public API
response.go             - Response wrapper with helpers
variables.go            - Map-based variable substitution
errors.go               - Structured error types
pools.go                - Buffer/object pools
retry.go                - Retry logic with body rewinding
cmd/gocurl/main.go      - CLI tool
```

### Enhanced Files
```
convert.go              - FIX token iteration, initialize maps
process.go              - Refactor to use new components
client.go               - Add client pooling
```

### Keep Unchanged
```
tokenizer/              - Already works well
middlewares/            - Good design
options/                - Solid foundation
```

## What We're NOT Building

❌ Custom HTTP protocol implementation
❌ Complex state machines
❌ Plugin architecture (middleware is enough)
❌ Auto-generated SDKs
❌ Non-HTTP curl features (FTP, SMTP, etc.)
❌ Perfect curl parity (only HTTP-relevant flags)
❌ GUI/web interface

## Key Design Decisions

### Zero-Allocation Strategy

**Critical Path (must be zero-alloc):**
- Request parsing - reuse token arrays
- Header building - pre-allocated maps
- URL construction - pooled string builders

**Acceptable Allocations (off critical path):**
- Response bodies - read once
- Error messages - exceptional cases
- TLS handshakes - infrequent

**Pooling Strategy:**
- Pool small, frequently-used objects only
- Don't pool large buffers (memory waste)
- Don't pool complex structs (maintenance burden)

### Variable Substitution

**Map-based, not environment-based:**
- Explicit, testable, secure
- Support ${var} and $var
- Handle escaping
- Error on undefined (fail-fast)

### Error Handling

**Philosophy:**
1. Fail fast - invalid input = immediate error
2. Clear messages - include context
3. No silent failures
4. Structured errors with operation context

### CLI-to-Code Workflow

**Same syntax works in both:**

```bash
# CLI
gocurl -H "Authorization: Bearer $TOKEN" https://api.github.com/repos/owner/repo
```

```go
// Go code (same syntax)
gocurl.Request(
    `gocurl -H "Authorization: Bearer ${token}" https://api.github.com/repos/owner/repo`,
    gocurl.Variables{"token": token},
)
```

## Success Metrics

### Performance
- 0 allocs/op on request construction
- < 1μs overhead vs net/http
- 10,000+ req/s sustained throughput
- 100,000+ concurrent requests without degradation
- < 100MB for 10k concurrent requests
- Linear scaling up to CPU core count

### Reliability
- 100% test coverage on core paths
- Zero panics in production
- All errors handled clearly
- Memory leak-free
- **Zero race conditions** (proven with -race)
- **Thread-safe**: All public APIs safe for concurrent use
- **No data races** under concurrent load

### Robustness (Military-Grade)
- **Fuzz tested**: 100M+ iterations without crashes
- **Load tested**: 10k req/s for 24 hours minimum
- **Stress tested**: Graceful at 10x normal load
- **Chaos tested**: Handles network failures, timeouts
- **Race-free**: All tests pass with -race flag
- **Benchmark regression**: CI fails on >5% degradation
- **Resource limits**: Handles exhaustion gracefully
- **Breaking point**: Documented maximum capacity

### Developer Experience
- < 5 minutes to first request
- Single-line curl execution
- CLI-to-code copy-paste works
- Clear examples for 10+ APIs

## Next Steps

1. **Start with Week 1** (P0 blockers)
2. **Get all tests passing**
3. **Build working CLI**
4. **Verify documented examples work**
5. **Move to zero-allocation work**

## Questions & Decisions

**Q: Why not use existing curl parsers?**
A: Simple, focused implementation is more maintainable and tailored to our needs.

**Q: Why pool buffers but not requests?**
A: Buffers are uniform and frequently reused. Requests are complex and varied.

**Q: Why separate API layer?**
A: Testability, clarity, and backward compatibility.

**Q: Can we achieve true zero-allocation?**
A: Yes, on the critical path (request construction). Response handling will have unavoidable allocations.

---

**Remember**: Every line of code should make the API sweeter, the implementation simpler, or the system more robust. If it doesn't, it doesn't belong.
