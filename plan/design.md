# GoCurl Implementation Design

## Design Philosophy: Sweet, Simple, Robust (SSR)

This design follows three core principles:
- **Sweet**: Developer-friendly API, minimal cognitive load, copy-paste curl commands
- **Simple**: No over-engineering, clear data flow, maintainable codebase
- **Robust**: Zero-allocation where critical, military-grade reliability, thread-safe, race-free, comprehensive error handling, battle-tested under extreme load

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    User-Facing Layer                        │
├─────────────────────────────────────────────────────────────┤
│  • gocurl.Request(cmd, vars)   - High-level API             │
│  • gocurl.Execute(opts)        - Direct execution           │
│  • CLI tool (cmd/gocurl)       - Command-line interface     │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                    Parsing Layer                            │
├─────────────────────────────────────────────────────────────┤
│  • String parser    - Handle curl command strings           │
│  • Token converter  - Convert tokens to RequestOptions      │
│  • Variable substitution - Safe, escapable, map-based       │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                 Core Execution Layer                        │
├─────────────────────────────────────────────────────────────┤
│  • Request builder  - Zero-alloc HTTP request construction  │
│  • Client factory   - Pooled, reusable HTTP clients         │
│  • Response handler - Streaming, buffered, or pooled        │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                  Transport Layer                            │
├─────────────────────────────────────────────────────────────┤
│  • HTTP/1.1 & HTTP/2 - Protocol support                     │
│  • Connection pooling - Reusable connections                │
│  • Proxy support     - HTTP/HTTPS/SOCKS5                    │
└─────────────────────────────────────────────────────────────┘
```

## Core Components Design

### 1. Public API (High-Level Interface)

**File**: `api.go` (NEW)

```go
// Variables type for safe variable substitution
type Variables map[string]string

// Request executes a curl command with optional variable substitution
// Accepts both string and []string formats
func Request(command interface{}, vars Variables) (*Response, error)

// Execute runs a request with pre-built RequestOptions
func Execute(opts *RequestOptions) (*Response, error)

// Response wraps http.Response with convenience methods
type Response struct {
    *http.Response
    bodyBytes []byte // cached, read once
}

// Convenience methods
func (r *Response) String() (string, error)
func (r *Response) JSON(v interface{}) error
func (r *Response) Bytes() ([]byte, error)
```

**Why**: Matches documented API, provides type safety, simple to use.

### 2. String Parser (Robust Command Parsing)

**File**: `parser/command_parser.go` (NEW - extract from cmd/parser.go)

```go
// ParseCommand handles both string and []string inputs
func ParseCommand(input interface{}) ([]string, error)

// Uses existing tokenizer logic with improvements:
// - Handle line continuations (\ at end of line)
// - Respect quotes (single, double)
// - Handle escaped characters
// - Simple, tested, no regex complexity
```

**Why**: Reuse proven logic, handle both input formats, keep it simple.

### 3. Token Converter (Fixed Core Logic)

**File**: `convert.go` (FIX EXISTING)

**Current Issues**:
- Sets method to flag value instead of consuming next token
- Never initializes Headers/Form maps (nil pointer panics)
- Incorrect token iteration logic

**Fixes**:
```go
func convertTokensToRequestOptions(tokens []Token) (*RequestOptions, error) {
    // INITIALIZE EVERYTHING FIRST
    o := options.NewRequestOptions("")
    o.Headers = make(http.Header)
    o.Form = make(url.Values)
    o.QueryParams = make(url.Values)
    
    // Expand variables in tokens BEFORE iteration
    expandedTokens := expandTokenVariables(tokens)
    
    // CORRECT ITERATION: consume value tokens after flag tokens
    for i := 0; i < len(expandedTokens); i++ {
        token := expandedTokens[i]
        
        // Skip "curl" command
        if i == 0 && token.Value == "curl" {
            continue
        }
        
        // Handle flags
        if token.Type == TokenFlag {
            switch token.Value {
            case "-X", "--request":
                i++ // Move to next token
                if i >= len(expandedTokens) {
                    return nil, fmt.Errorf("missing value for %s", token.Value)
                }
                o.Method = expandedTokens[i].Value // Use NEXT token's value
                
            case "-d", "--data":
                i++ // Move to next token
                if i >= len(expandedTokens) {
                    return nil, fmt.Errorf("missing value for %s", token.Value)
                }
                dataFields = append(dataFields, expandedTokens[i].Value)
                if o.Method == "" || o.Method == "GET" {
                    o.Method = "POST"
                }
                
            // ... similar fixes for all flags
            }
        } else {
            // Handle URL (positional argument)
            if o.URL == "" && isURL(token.Value) {
                o.URL = token.Value
            }
        }
    }
    
    // Validate required fields
    if o.URL == "" {
        return nil, fmt.Errorf("no URL provided")
    }
    
    // Set defaults
    if o.Method == "" {
        o.Method = "GET"
    }
    
    return o, nil
}
```

**Why**: Fixes P0 blocker, maintains simplicity, aligns with curl semantics.

### 4. Variable Substitution (Map-Based, Safe)

**File**: `variables.go` (NEW)

```go
// ExpandVariables replaces ${var} or $var with values from map
// Supports escaping: \${var} becomes literal ${var}
func ExpandVariables(text string, vars Variables) (string, error) {
    // Simple state machine approach (not regex)
    // - Scan character by character
    // - Track escape sequences
    // - Handle ${var} and $var formats
    // - Validate variable names exist in map
    // - Return error for undefined variables (fail-fast, secure)
}

// expandTokenVariables applies to token array
func expandTokenVariables(tokens []Token, vars Variables) []Token
```

**Why**: More secure than os.ExpandEnv, controlled, testable, no surprises.

### 5. Request Builder (Zero-Allocation Path)

**File**: `builder/request_builder.go` (NEW)

```go
// Buffer pools for reusable allocations
var (
    bodyBufferPool = sync.Pool{
        New: func() interface{} {
            return bytes.NewBuffer(make([]byte, 0, 4096))
        },
    }
)

// BuildRequest constructs http.Request with minimal allocations
func BuildRequest(ctx context.Context, opts *RequestOptions) (*http.Request, error) {
    // Reuse buffers from pool where possible
    // Stream large bodies instead of buffering
    // Use pre-allocated headers map
    // Return request ready for execution
}
```

**Why**: Achieves zero-allocation promise for critical path without over-engineering.

### 6. Response Handler (Smart Buffering)

**File**: `response.go` (NEW)

```go
// Response provides convenient access to response data
type Response struct {
    *http.Response
    bodyBytes []byte
    bodyRead  bool
}

// Bytes reads body once, caches result
func (r *Response) Bytes() ([]byte, error) {
    if !r.bodyRead {
        // Use pooled buffer for small responses
        // Stream large responses to disk if needed
        r.bodyBytes, err = readBody(r.Body)
        r.bodyRead = true
    }
    return r.bodyBytes, nil
}

// String is convenience wrapper
func (r *Response) String() (string, error) {
    b, err := r.Bytes()
    return string(b), err
}

// JSON unmarshals into provided struct
func (r *Response) JSON(v interface{}) error {
    b, err := r.Bytes()
    if err != nil {
        return err
    }
    return json.Unmarshal(b, v)
}
```

**Why**: Simple API, read-once semantics, pooled buffers for common case.

### 7. Client Factory (Connection Pooling)

**File**: `client.go` (ENHANCE EXISTING)

```go
// Thread-safe client pool with automatic cleanup
var (
    clientPool   = &sync.Map{} // key: config hash, value: *http.Client
    clientPoolMu sync.RWMutex   // For cleanup operations
)

// GetClient returns cached or creates new client (thread-safe)
func GetClient(opts *RequestOptions) (*http.Client, error) {
    // Hash the config (timeout, proxy, TLS settings)
    hash := hashConfig(opts)
    
    // Check pool (lock-free read)
    if client, ok := clientPool.Load(hash); ok {
        return client.(*http.Client), nil
    }
    
    // Create new client
    client, err := createHTTPClient(opts)
    if err != nil {
        return nil, err
    }
    
    // Store in pool (atomic)
    clientPool.Store(hash, client)
    return client, nil
}

// Background goroutine for cleanup (started once)
func init() {
    go cleanupIdleClients()
}

func cleanupIdleClients() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        // Close idle connections in pooled clients
        clientPool.Range(func(key, value interface{}) bool {
            if client, ok := value.(*http.Client); ok {
                if transport, ok := client.Transport.(*http.Transport); ok {
                    transport.CloseIdleConnections()
                }
            }
            return true
        })
    }
}
```

**Why**: Reuse connections, thread-safe pooling, automatic resource management, no lock contention.

### 8. Retry Logic (Body Rewind Support)

**File**: `retry.go` (NEW)

```go
// executeWithRetries handles retry logic correctly
func executeWithRetries(client *http.Client, req *http.Request, opts *RequestOptions) (*http.Response, error) {
    // Clone request with rewindable body
    // Use io.Seeker or buffer body for retries
    // Apply retry delay
    // Log attempts if verbose
}

// cloneRequest creates copy with rewindable body
func cloneRequest(req *http.Request) (*http.Request, error) {
    // Handle body by buffering or using io.Seeker
    // Clone headers and other fields
    // Return independent request
}
```

**Why**: Fixes P1 issue, handles retries correctly, maintains simplicity.

### 9. CLI Tool (Same Syntax as Library)

**File**: `cmd/gocurl/main.go` (NEW)

```go
func main() {
    // Parse os.Args directly
    command := strings.Join(os.Args[1:], " ")
    
    // Expand environment variables for CLI context
    vars := make(Variables)
    // Auto-populate from environment
    
    // Execute using same code path as library
    resp, err := gocurl.Request(command, vars)
    
    // Format and print output
    // Handle --verbose, --silent flags
    // Pretty-print JSON if detected
}
```

**Why**: Zero duplication, identical syntax, CLI-to-code workflow works seamlessly.

## Memory Management Strategy

### Zero-Allocation Targets

**Critical Path** (must be zero-alloc):
- Request option parsing (reuse token arrays)
- Header construction (pre-allocated maps)
- URL building (string builder pooling)

**Acceptable Allocations** (not on critical path):
- Response bodies (one-time read)
- Error messages (exceptional cases)
- TLS handshakes (infrequent)

### Pooling Strategy

```go
// Pool small, frequently-used objects
var (
    tokenArrayPool   = sync.Pool{...} // []Token
    stringBuilderPool = sync.Pool{...} // strings.Builder
    bufferPool       = sync.Pool{...} // bytes.Buffer
)

// Don't pool:
// - Large buffers (memory waste)
// - Complex structs (maintenance burden)
// - Rarely-used objects (pool overhead)
```

**Why**: Focus zero-alloc efforts where they matter, avoid over-engineering.

## Error Handling Philosophy

1. **Fail Fast**: Invalid input = immediate error
2. **Clear Messages**: "missing value for -X flag" not "nil pointer"
3. **Context**: Include command snippet in errors
4. **No Silent Failures**: All errors propagate up

```go
type GocurlError struct {
    Op      string // "parse", "request", "response"
    Command string // Offending command snippet
    Err     error  // Underlying error
}

func (e *GocurlError) Error() string {
    return fmt.Sprintf("%s failed: %v (command: %s)", e.Op, e.Err, e.Command)
}
```

## Testing Strategy

### Unit Tests
- Parse every curl flag combination
- Variable substitution edge cases
- Error conditions
- Token conversion logic
- Thread-safe operations (concurrent access to shared resources)

### Race Condition Tests
- **All tests run with `-race` flag in CI**
- Concurrent request execution (1000+ goroutines)
- Simultaneous client pool access
- Parallel variable substitution
- Concurrent response reading
- Shared buffer pool stress tests

### Integration Tests
- End-to-end curl command execution
- Real HTTP server roundtrips
- Proxy scenarios
- TLS configurations
- Retry with concurrent requests

### Benchmark Tests
- Zero-allocation verification (`-benchmem`)
- Performance vs net/http
- Memory profiling (`-memprofile`)
- CPU profiling (`-cpuprofile`)
- Concurrent request handling (1k, 10k, 100k goroutines)
- Benchmark regression testing (CI fails on >5% degradation)

### Load Tests
- **Sustained throughput**: 10k req/s for 1 hour minimum
- **Burst handling**: 100k req/s for 10 seconds
- **Memory stability**: No growth over 24-hour soak test
- **Connection pooling**: Verify reuse under load
- **Graceful degradation**: Behavior at 10x normal load

### Stress Tests
- **Breaking point**: Find maximum concurrent requests before failure
- **Resource exhaustion**: File descriptor limits, memory limits
- **Recovery testing**: Graceful recovery from overload
- **Concurrent failures**: Multiple requests failing simultaneously

### Chaos Tests
- **Network failures**: Random connection drops, timeouts
- **Partial responses**: Incomplete body reads
- **Slow responses**: Server delays, slow networks
- **Malformed data**: Invalid headers, corrupt bodies
- **TLS errors**: Certificate failures, handshake errors

### Fuzz Tests
- **Parser fuzzing**: 100M+ random inputs to tokenizer
- **URL fuzzing**: Malformed URLs, edge cases
- **Header fuzzing**: Invalid header formats
- **Variable fuzzing**: Edge cases in substitution

### Test Structure
```
convert_test.go         - Token conversion (FIX EXISTING)
api_test.go             - High-level API (NEW)
parser_test.go          - Command parsing (NEW)
variables_test.go       - Variable substitution (NEW)
race_test.go            - Race condition tests (NEW)
integration_test.go     - End-to-end (NEW)
benchmark_test.go       - Performance (NEW)
load_test.go            - Load and stress tests (NEW)
chaos_test.go           - Chaos engineering (NEW)
fuzz_test.go            - Fuzz testing (NEW)
```

### Continuous Integration Requirements

```yaml
# All PRs must pass:
- go test -race -v ./...           # Race detection
- go test -bench=. -benchmem ./... # Benchmark verification
- go test -fuzz=. -fuzztime=10s    # Fuzz tests
- golangci-lint run                # Static analysis
- go vet ./...                     # Vet checks

# Nightly builds run:
- 24-hour soak tests
- Load tests (100k req/s)
- Stress tests to breaking point
- Full fuzz suite (1 hour)
- Memory leak detection
```

## Implementation Phases

### Phase 1: Foundation (P0 Blockers) - Week 1
- [ ] Fix `convertTokensToRequestOptions` token iteration
- [ ] Initialize all RequestOptions maps/slices
- [ ] Implement high-level `Request()` API
- [ ] Create `Variables` type and map-based substitution
- [ ] Restore all failing tests
- [ ] Build working CLI tool

**Success Criteria**: All existing tests pass, basic CLI works, documented examples run.

### Phase 2: Zero-Allocation Core - Week 2
- [ ] Implement buffer pools for request building
- [ ] Add response body pooling for small responses
- [ ] Create client pool with config hashing
- [ ] Benchmark and verify zero-alloc on critical path
- [ ] Document allocation strategy

**Success Criteria**: Benchmarks show zero alloc on request path, beat net/http baseline.

### Phase 3: Reliability Features - Week 3
- [ ] Fix retry logic with body rewinding
- [ ] Implement request cloning
- [ ] Add streaming support for large bodies
- [ ] Comprehensive error types
- [ ] Security: sensitive data redaction
- [ ] **Thread-safety verification**: All tests pass with `-race`
- [ ] **Concurrent stress tests**: 10k+ parallel requests
- [ ] **Race condition tests**: Shared state protection verified

**Success Criteria**: Retries work correctly, large files handled efficiently, zero races detected.

### Phase 4: Complete Feature Set - Week 4
- [ ] Full proxy support (HTTP/HTTPS/SOCKS5)
- [ ] Complete TLS configuration
- [ ] All curl flags implemented
- [ ] Compression handling (gzip, deflate, brotli)
- [ ] Cookie jar persistence

**Success Criteria**: Feature parity with documented objectives.

### Phase 5: Polish & Documentation - Week 5
- [ ] Update README with real examples
- [ ] CLI help and usage documentation
- [ ] Performance comparison benchmarks
- [ ] **Load test suite**: 10k req/s sustained, 100k burst
- [ ] **Stress test reports**: Breaking point analysis
- [ ] **Chaos test suite**: Network failure handling
- [ ] **Soak test**: 24-hour stability verification
- [ ] Migration guide from net/http
- [ ] Example library (Stripe, GitHub, etc.)

**Success Criteria**: Professional documentation, clear examples, load tested and battle-ready for v1.0.

## File Organization

```
gocurl/
├── api.go                  # NEW - High-level public API
├── client.go               # ENHANCE - Add pooling
├── convert.go              # FIX - Token conversion logic
├── request.go              # NEW - Request builder
├── response.go             # NEW - Response wrapper
├── retry.go                # NEW - Retry logic
├── variables.go            # NEW - Variable substitution
├── errors.go               # NEW - Error types
├── pools.go                # NEW - Buffer/object pools
├── process.go              # REFACTOR - Use new components
├── cmd/
│   └── gocurl/
│       └── main.go         # NEW - CLI tool
├── parser/
│   └── command_parser.go   # NEW - Extract from cmd/
├── tokenizer/
│   └── tokenizer.go        # KEEP - Works well
├── options/
│   └── options.go          # ENHANCE - Add helpers
├── middlewares/
│   └── middlewares.go      # KEEP - Good design
├── proxy/
│   └── *.go                # ENHANCE - Complete impl
└── tests/
    ├── api_test.go         # NEW
    ├── integration_test.go # NEW
    ├── benchmark_test.go   # NEW
    └── ... (existing tests)
```

## Non-Goals (Avoid Over-Engineering)

- ❌ Custom HTTP protocol implementation
- ❌ Complex state machines
- ❌ Plugin architecture (middleware is enough)
- ❌ GUI or web interface
- ❌ Auto-generated SDKs (examples are fine)
- ❌ GraphQL support (HTTP/REST only)
- ❌ Non-HTTP curl features (FTP, SMTP, etc.)

## Success Metrics

### Performance
- Zero allocations on request construction
- < 1μs overhead vs raw net/http
- 10k+ requests/sec sustained throughput
- < 100MB memory for 10k concurrent requests

### Reliability
- 100% test coverage on core paths
- No panics, all errors handled
- Memory leak-free under load
- Graceful degradation on failures

### Developer Experience
- Single-line curl command execution
- CLI-to-code copy-paste works
- Clear error messages
- < 5 minutes to first working request

## Migration from Current Code

1. Keep existing `tokenizer` package - it works
2. Keep existing `options` package - good design
3. Fix `convert.go` - critical path
4. Extract parser from `cmd/parser.go` - reuse logic
5. Refactor `process.go` - use new components
6. Build new API layer on top - backward compatible
7. Add tests incrementally - no big-bang rewrites

## Decision Log

**Why not use reflection for variable substitution?**
→ Performance and security. Map-based is simple, fast, and explicit.

**Why pool buffers but not entire requests?**
→ Buffers are uniform and frequently used. Requests are complex and varied.

**Why separate API layer from execution layer?**
→ Testability, clarity, and ability to evolve internals without breaking users.

**Why not support all curl features?**
→ Focus on HTTP/HTTPS excellence. Full curl is 100+ flags we'll never use.

**Why map-based variable substitution instead of environment?**
→ Security, testability, and control. Environment is for CLI context only.

---

**Design Principle**: Every line of code should serve a clear purpose. If it's not making the API sweeter, the implementation simpler, or the system more robust, it doesn't belong.
