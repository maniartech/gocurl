# RequestOptions Analysis - Alignment with Project Objectives

## Executive Summary

Based on the project objectives, this document analyzes the current `RequestOptions` structure to identify:
1. ✅ **Essential options** aligned with HTTP/HTTPS curl operations
2. ⚠️ **Missing options** needed for industry standards
3. ❌ **Unnecessary options** that should be removed
4. 🔧 **Options requiring better implementation**

## Project Scope (from objective.md)

**Primary Focus**: HTTP/HTTPS curl operations with zero-allocation, military-grade performance

**Explicit Non-Goals**:
- ❌ Non-HTTP protocols (FTP, SMTP, TFTP, LDAP)
- ❌ GraphQL support
- ❌ Server-side functionality
- ❌ GUI/web interface
- ❌ Full curl feature parity beyond HTTP/HTTPS

## Current RequestOptions Analysis

### ✅ Essential Options (Keep - Aligned with Objectives)

#### HTTP Request Basics
```go
Method      string      // ✅ Essential - all HTTP methods
URL         string      // ✅ Essential - target endpoint
Headers     http.Header // ✅ Essential - custom headers
Body        string      // ✅ Essential - request payload
Form        url.Values  // ✅ Essential - form data
QueryParams url.Values  // ✅ Essential - URL parameters
```
**Rationale**: Core HTTP request construction, directly maps to curl flags.

#### Authentication
```go
BasicAuth   *BasicAuth // ✅ Essential - curl -u flag
BearerToken string     // ✅ Essential - common in modern APIs
```
**Rationale**: Industry standard authentication methods, frequently used in API docs.

#### TLS/SSL Options
```go
CertFile            string      // ✅ Essential - curl --cert
KeyFile             string      // ✅ Essential - curl --key
CAFile              string      // ✅ Essential - curl --cacert
Insecure            bool        // ✅ Essential - curl -k/--insecure
TLSConfig           *tls.Config // ✅ Essential - advanced TLS control
CertPinFingerprints []string    // ✅ Essential - security feature
SNIServerName       string      // ✅ Essential - curl --resolve SNI
```
**Rationale**: Security and TLS configuration critical for HTTPS, maps to curl flags.

#### Proxy Settings
```go
Proxy        string   // ✅ Essential - curl -x/--proxy
ProxyNoProxy []string // ✅ Essential - curl --noproxy
```
**Rationale**: Common in enterprise environments, direct curl mapping.

#### Timeout Settings
```go
Timeout        time.Duration // ✅ Essential - curl -m/--max-time
ConnectTimeout time.Duration // ✅ Essential - curl --connect-timeout
```
**Rationale**: Critical for reliability, prevents hanging requests.

#### Redirect Behavior
```go
FollowRedirects bool // ✅ Essential - curl -L/--location
MaxRedirects    int  // ✅ Essential - curl --max-redirs
```
**Rationale**: Standard HTTP behavior control, direct curl mapping.

#### Compression
```go
Compress           bool     // ✅ Essential - curl --compressed
CompressionMethods []string // ✅ Keep - specific compression control
```
**Rationale**: Performance optimization, modern web standard.

#### HTTP Version
```go
HTTP2     bool // ✅ Essential - curl --http2
HTTP2Only bool // ✅ Essential - curl --http2-prior-knowledge
```
**Rationale**: Protocol version control, performance optimization.

#### Cookie Handling
```go
Cookies    []*http.Cookie // ✅ Essential - curl -b/--cookie
CookieJar  http.CookieJar // ✅ Essential - session management
CookieFile string         // ✅ Essential - curl -c/--cookie-jar
```
**Rationale**: Session management, stateful API interactions.

#### Custom Options
```go
UserAgent string // ✅ Essential - curl -A/--user-agent
Referer   string // ✅ Essential - curl -e/--referer
```
**Rationale**: Common HTTP headers, API requirements.

#### File Upload
```go
FileUpload *FileUpload // ✅ Essential - curl -F/--form
```
**Rationale**: Multipart form data, common API operation.

#### Retry Configuration
```go
RetryConfig *RetryConfig // ✅ Essential - reliability feature
```
**Rationale**: Production reliability, not in curl but essential for robust clients.

#### Context & Lifecycle
```go
Context       context.Context    // ✅ Essential - Go standard, timeout control
ContextCancel context.CancelFunc // ✅ Essential - memory leak prevention
```
**Rationale**: Go best practices, context-aware operations, industry standard.

#### Advanced Features
```go
RequestID         string                       // ✅ Keep - debugging/tracing
Middleware        []middlewares.MiddlewareFunc // ✅ Keep - extensibility
ResponseBodyLimit int64                        // ✅ Keep - DoS protection
```
**Rationale**: Production features for debugging, extensibility, security.

### ⚠️ Questionable Options (Review)

```go
OutputFile string // ⚠️ Review - curl -o/--output
Silent     bool   // ⚠️ Review - curl -s/--silent
Verbose    bool   // ⚠️ Review - curl -v/--verbose
```
**Issue**: These are CLI-specific features, not library features.

**Recommendation**:
- **Keep for CLI tool** - These make sense for the `gocurl` CLI command
- **Remove from library RequestOptions** - Library users handle output themselves
- **Alternative**: Move to CLI-specific options structure

### ❌ Unnecessary/Problematic Options (Remove or Refactor)

```go
ResponseDecoder func(*http.Response) (interface{}, error) // ❌ Remove
Metrics         *RequestMetrics                            // ❌ Remove
CustomClient    interface{}                                // ❌ Remove
```

#### 1. ResponseDecoder ❌
**Problems**:
- Not used anywhere in codebase
- Unclear purpose - users can decode responses themselves
- Adds complexity without clear benefit
- Not curl-related

**Recommendation**: **REMOVE** - Users can handle response decoding:
```go
// Instead of built-in decoder:
resp, err := gocurl.Get(ctx, url, nil)
defer resp.Body.Close()
json.NewDecoder(resp.Body).Decode(&result)
```

#### 2. Metrics ❌
**Problems**:
- Partially implemented but not consistently used
- Adds allocation overhead (against zero-alloc goal)
- Not curl-related
- Better handled by external observability tools

**Recommendation**: **REMOVE** from RequestOptions
- Users can track metrics externally if needed
- Reduces struct size and allocation
- Focus on core HTTP functionality

#### 3. CustomClient ❌
**Problems**:
- Type `interface{}` is code smell
- Unclear purpose and usage
- No documentation or tests
- Adds confusion

**Recommendation**: **REMOVE**
- If users need custom client, they can use their own wrapper
- Library should own the client creation

### 🔧 Missing Options (Industry Standards from NOT_COVERED.md)

Based on industry standards and the NOT_COVERED.md analysis:

#### Transport-Level Timeouts (CRITICAL)
```go
// MISSING - Industry standard transport timeouts
TLSHandshakeTimeout  time.Duration // Go stdlib default: 10s
ResponseHeaderTimeout time.Duration // Go stdlib default: none
IdleConnTimeout      time.Duration // Go stdlib default: 90s
ExpectContinueTimeout time.Duration // Go stdlib default: 1s
```
**Rationale**: Fine-grained timeout control, production reliability
**curl mapping**: Not direct, but essential for HTTP/2 and production use

#### Connection Pool Control (CRITICAL)
```go
// MISSING - Connection pool management
MaxIdleConns        int  // Go stdlib default: 100
MaxIdleConnsPerHost int  // Go stdlib default: 2
MaxConnsPerHost     int  // Go stdlib default: 0 (unlimited)
DisableKeepAlives   bool // curl --no-keepalive
```
**Rationale**: Performance tuning, resource management
**curl mapping**: `--no-keepalive`, essential for high-performance scenarios

#### TLS Configuration (IMPORTANT)
```go
// MISSING - TLS version control
MinTLSVersion uint16 // curl --tlsv1.2
MaxTLSVersion uint16 // curl --tls-max
```
**Rationale**: Security compliance, some APIs require specific TLS versions

#### HTTP/2 Configuration (NICE TO HAVE)
```go
// MISSING - HTTP/2 specific
HTTP2PriorKnowledge bool // curl --http2-prior-knowledge (HTTP/2 over cleartext)
```
**Rationale**: HTTP/2 without TLS for internal services

## Recommended Changes

### Phase 1: Remove Unnecessary Options (Immediate)

```go
// REMOVE these fields from RequestOptions:
// ❌ ResponseDecoder ResponseDecoder
// ❌ Metrics         *RequestMetrics
// ❌ CustomClient    interface{}
```

### Phase 2: Move CLI-Specific Options (Immediate)

Create separate `CLIOptions` structure:

```go
// NEW: CLI-specific options (separate from RequestOptions)
type CLIOptions struct {
    OutputFile string // curl -o
    Silent     bool   // curl -s
    Verbose    bool   // curl -v
    ShowHeaders bool  // curl -i
    HeadOnly    bool  // curl -I
}

// RequestOptions should NOT have these CLI-specific fields
```

### Phase 3: Add Missing Industry Standard Options (High Priority)

```go
// ADD to RequestOptions:
type RequestOptions struct {
    // ... existing fields ...

    // Transport-level timeouts (CRITICAL for production)
    TLSHandshakeTimeout   time.Duration
    ResponseHeaderTimeout time.Duration
    IdleConnTimeout       time.Duration
    ExpectContinueTimeout time.Duration

    // Connection pool control (CRITICAL for performance)
    MaxIdleConns        int
    MaxIdleConnsPerHost int
    MaxConnsPerHost     int
    DisableKeepAlives   bool

    // TLS version control (IMPORTANT for security)
    MinTLSVersion uint16
    MaxTLSVersion uint16

    // HTTP/2 specific (NICE TO HAVE)
    HTTP2PriorKnowledge bool
}
```

### Phase 4: Update CreateHTTPClient to Use New Options

Ensure all new options are properly applied to `http.Transport`:

```go
func CreateHTTPClient(ctx context.Context, opts *options.RequestOptions) (*http.Client, error) {
    transport := &http.Transport{
        TLSClientConfig: tlsConfig,

        // Apply new timeout options
        TLSHandshakeTimeout:   opts.TLSHandshakeTimeout,
        ResponseHeaderTimeout: opts.ResponseHeaderTimeout,
        IdleConnTimeout:       opts.IdleConnTimeout,
        ExpectContinueTimeout: opts.ExpectContinueTimeout,

        // Apply connection pool options
        MaxIdleConns:        opts.MaxIdleConns,
        MaxIdleConnsPerHost: opts.MaxIdleConnsPerHost,
        MaxConnsPerHost:     opts.MaxConnsPerHost,
        DisableKeepAlives:   opts.DisableKeepAlives,
    }

    // Apply TLS version constraints
    if opts.MinTLSVersion != 0 {
        tlsConfig.MinVersion = opts.MinTLSVersion
    }
    if opts.MaxTLSVersion != 0 {
        tlsConfig.MaxVersion = opts.MaxTLSVersion
    }

    // ... rest of implementation
}
```

## Impact Analysis

### Breaking Changes

1. **Removed Fields**:
   - `ResponseDecoder` - Unlikely to affect anyone (unused)
   - `Metrics` - May affect some users (provide migration path)
   - `CustomClient` - Unlikely to affect anyone (unclear usage)

2. **Moved Fields**:
   - `OutputFile`, `Silent`, `Verbose` - Only affects CLI code (internal)

### Benefits

1. **Smaller Struct**: Reduced memory footprint
2. **Clearer Purpose**: RequestOptions focused on HTTP request configuration
3. **Better Performance**: Removed allocation-heavy fields (Metrics)
4. **Industry Standard**: Added missing transport-level controls
5. **Production Ready**: Fine-grained timeout and connection pool control

## Implementation Priority

### Priority 1 (Immediate - This PR)
- ✅ Remove `ResponseDecoder`, `Metrics`, `CustomClient`
- ✅ Move CLI-specific fields to separate structure
- ✅ Update builder to remove deprecated fields
- ✅ Update tests

### Priority 2 (Next PR - High Value)
- ⬜ Add transport-level timeout options
- ⬜ Add connection pool control options
- ⬜ Add TLS version control
- ⬜ Update CreateHTTPClient to apply new options
- ⬜ Add builder methods for new options
- ⬜ Add tests for new options

### Priority 3 (Future - Nice to Have)
- ⬜ HTTP/2 prior knowledge option
- ⬜ Custom DNS resolution
- ⬜ Interface binding

## Testing Strategy

### Remove Unnecessary Options
1. Search codebase for usage of removed fields
2. Confirm no external dependencies
3. Remove from struct
4. Remove from builder
5. Run all tests

### Add New Options
1. Add fields to struct with sensible defaults
2. Add builder methods
3. Apply in CreateHTTPClient
4. Add unit tests
5. Add integration tests
6. Document in examples

## Migration Guide (for removed options)

### ResponseDecoder
```go
// Before (NOT SUPPORTED):
opts.ResponseDecoder = func(resp *http.Response) (interface{}, error) {
    // ...
}

// After (USER HANDLES):
resp, err := gocurl.Execute(opts)
if err != nil {
    return nil, err
}
defer resp.Body.Close()
json.NewDecoder(resp.Body).Decode(&result)
```

### Metrics
```go
// Before (REMOVED):
opts.Metrics = &RequestMetrics{}
// ... make request ...
fmt.Println(opts.Metrics.Duration)

// After (EXTERNAL TRACKING):
start := time.Now()
resp, err := gocurl.Execute(opts)
duration := time.Since(start)
fmt.Println(duration)
```

### CustomClient
```go
// Before (REMOVED):
opts.CustomClient = myClient

// After (USE WRAPPER):
// Create your own wrapper if you need custom client logic
type MyClient struct {
    customClient *http.Client
}

func (c *MyClient) Do(req *http.Request) (*http.Response, error) {
    return c.customClient.Do(req)
}
```

## Conclusion

**Recommended Actions**:
1. ✅ **Remove** unused/problematic options (ResponseDecoder, Metrics, CustomClient)
2. ✅ **Move** CLI-specific options to separate structure
3. ✅ **Add** industry-standard transport and TLS options
4. ✅ **Implement** new options in CreateHTTPClient
5. ✅ **Document** changes and migration path

This cleanup aligns RequestOptions with project objectives:
- Focus on HTTP/HTTPS curl operations
- Remove bloat for zero-allocation goals
- Add missing industry standards
- Clear separation between library and CLI concerns
