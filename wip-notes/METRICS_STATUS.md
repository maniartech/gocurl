# RequestMetrics Status Report

**Date**: October 14, 2025
**Status**: ⚠️ **DEFINED BUT NOT IMPLEMENTED**

---

## Current Status

### ✅ What Exists

**1. Type Definition** (options/options.go:144-157)
```go
type RequestMetrics struct {
    StartTime     time.Time     `json:"start_time"`      // When the request started
    EndTime       time.Time     `json:"end_time"`        // When the request completed
    Duration      time.Duration `json:"duration"`        // Total request duration
    DNSLookupTime time.Duration `json:"dns_lookup_time"` // DNS resolution time
    ConnectTime   time.Duration `json:"connect_time"`    // Connection establishment time
    TLSTime       time.Duration `json:"tls_time"`        // TLS handshake time
    FirstByteTime time.Duration `json:"first_byte_time"` // Time to first response byte
    RetryCount    int           `json:"retry_count"`     // Number of retries attempted
    ResponseSize  int64         `json:"response_size"`   // Size of response body in bytes
    RequestSize   int64         `json:"request_size"`    // Size of request body in bytes
    StatusCode    int           `json:"status_code"`     // HTTP status code
    Error         string        `json:"error,omitempty"` // Error message if request failed
}
```

**2. Field in RequestOptions** (options/options.go:84)
```go
Metrics *RequestMetrics `json:"-"` // Request metrics for observability
```

**3. Clone Support** (options/options.go:192-195)
```go
if ro.Metrics != nil {
    clonedMetrics := *ro.Metrics
    clone.Metrics = &clonedMetrics
}
```

### ❌ What's Missing

**NO ACTUAL IMPLEMENTATION**

Metrics are:
- ❌ **NOT collected** during request execution
- ❌ **NOT populated** in Process()
- ❌ **NOT updated** in executeWithRetries()
- ❌ **NOT set** anywhere in the codebase

**Evidence**:
```bash
$ grep -r "opts\.Metrics" *.go
# Only found in Clone() method - no actual usage!
```

---

## How It SHOULD Work (Not Implemented)

### Intended Usage Pattern

```go
// User provides a Metrics struct
opts := &options.RequestOptions{
    URL: "https://api.example.com",
    Metrics: &options.RequestMetrics{}, // User opts in
}

resp, body, err := gocurl.Process(ctx, opts)

// After request, metrics should be populated:
fmt.Printf("Duration: %v\n", opts.Metrics.Duration)
fmt.Printf("DNS Lookup: %v\n", opts.Metrics.DNSLookupTime)
fmt.Printf("Connect Time: %v\n", opts.Metrics.ConnectTime)
fmt.Printf("TLS Time: %v\n", opts.Metrics.TLSTime)
fmt.Printf("First Byte: %v\n", opts.Metrics.FirstByteTime)
fmt.Printf("Retry Count: %d\n", opts.Metrics.RetryCount)
fmt.Printf("Response Size: %d bytes\n", opts.Metrics.ResponseSize)
fmt.Printf("Status Code: %d\n", opts.Metrics.StatusCode)
```

### Why It's Valuable

1. **Observability**: Track request performance in production
2. **Debugging**: Identify slow DNS, connection, or TLS issues
3. **Monitoring**: Feed metrics to Prometheus, DataDog, etc.
4. **SLAs**: Measure and report on API performance
5. **Optimization**: Identify bottlenecks (DNS vs Connect vs Transfer)

---

## Implementation Required

### Where to Add Metrics Collection

#### 1. In `Process()` - Track Overall Request

```go
// process.go
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
    // Start metrics if user opted in
    if opts.Metrics != nil {
        opts.Metrics.StartTime = time.Now()
    }

    // ... existing validation code ...

    // Execute request with retries
    resp, err := executeWithRetries(httpClient, req, opts)
    if err != nil {
        // Record error in metrics
        if opts.Metrics != nil {
            opts.Metrics.EndTime = time.Now()
            opts.Metrics.Duration = opts.Metrics.EndTime.Sub(opts.Metrics.StartTime)
            opts.Metrics.Error = err.Error()
        }
        return nil, "", err
    }

    // ... existing body reading code ...

    // Finalize metrics
    if opts.Metrics != nil {
        opts.Metrics.EndTime = time.Now()
        opts.Metrics.Duration = opts.Metrics.EndTime.Sub(opts.Metrics.StartTime)
        opts.Metrics.StatusCode = resp.StatusCode
        opts.Metrics.ResponseSize = int64(len(bodyBytes))
    }

    return resp, bodyString, nil
}
```

#### 2. In `executeWithRetries()` - Track Retries

```go
// retry.go
func executeWithRetries(client options.HTTPClient, req *http.Request, opts *options.RequestOptions) (*http.Response, error) {
    // ... existing retry loop code ...

    for attempt := 0; attempt <= retries; attempt++ {
        // Record retry count
        if opts.Metrics != nil && attempt > 0 {
            opts.Metrics.RetryCount = attempt
        }

        resp, err = client.Do(attemptReq)

        // ... existing retry logic ...
    }

    return resp, nil
}
```

#### 3. In `CreateHTTPClient()` - Track Connection Timing

**This is HARD** - requires custom `http.Transport` with tracing:

```go
// client.go - Advanced implementation
func CreateHTTPClient(ctx context.Context, opts *options.RequestOptions) (*http.Client, error) {
    // ... existing TLS config code ...

    transport := &http.Transport{
        TLSClientConfig: tlsConfig,
        Proxy:           http.ProxyFromEnvironment,
    }

    // Add trace hooks for detailed metrics (if Metrics enabled)
    if opts.Metrics != nil {
        // Use httptrace to collect timing metrics
        // This requires per-request context with trace hooks
        // See: https://pkg.go.dev/net/http/httptrace
    }

    // ... rest of client creation ...
}
```

**Note**: Detailed timing (DNS, Connect, TLS, FirstByte) requires using `httptrace.ClientTrace` which must be attached to the request context.

---

## Implementation Complexity

### Easy Metrics (Can implement quickly)

✅ **StartTime** - `time.Now()` at start of Process()
✅ **EndTime** - `time.Now()` at end of Process()
✅ **Duration** - `EndTime.Sub(StartTime)`
✅ **StatusCode** - From `resp.StatusCode`
✅ **ResponseSize** - `int64(len(bodyBytes))`
✅ **RequestSize** - `int64(len(opts.Body))` or from request
✅ **RetryCount** - Track in executeWithRetries() loop
✅ **Error** - From final error if request fails

**Estimated Time**: 1-2 hours

### Hard Metrics (Requires httptrace integration)

⚠️ **DNSLookupTime** - Requires `httptrace.ClientTrace.DNSDone`
⚠️ **ConnectTime** - Requires `httptrace.ClientTrace.ConnectDone`
⚠️ **TLSTime** - Requires `httptrace.ClientTrace.TLSHandshakeDone`
⚠️ **FirstByteTime** - Requires `httptrace.ClientTrace.GotFirstResponseByte`

**Challenge**: httptrace hooks must be set on request context BEFORE calling `client.Do()`, which means:
1. Check if `opts.Metrics != nil` in CreateRequest()
2. Create httptrace.ClientTrace with callbacks
3. Attach to request context
4. Callbacks populate opts.Metrics fields

**Estimated Time**: 4-6 hours (requires deep understanding of httptrace)

---

## Example: Full Implementation (Easy Metrics)

```go
// process.go
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
    // Initialize metrics
    if opts.Metrics != nil {
        opts.Metrics.StartTime = time.Now()
        if opts.Body != "" {
            opts.Metrics.RequestSize = int64(len(opts.Body))
        }
    }

    // Validate options
    if err := ValidateOptions(opts); err != nil {
        if opts.Metrics != nil {
            opts.Metrics.Error = err.Error()
        }
        return nil, "", err
    }

    // Use custom client if provided, otherwise create standard HTTP client
    var httpClient options.HTTPClient
    if opts.CustomClient != nil {
        httpClient = opts.CustomClient
    } else {
        client, err := CreateHTTPClient(ctx, opts)
        if err != nil {
            if opts.Metrics != nil {
                opts.Metrics.EndTime = time.Now()
                opts.Metrics.Duration = opts.Metrics.EndTime.Sub(opts.Metrics.StartTime)
                opts.Metrics.Error = err.Error()
            }
            return nil, "", err
        }
        httpClient = client
    }

    // Create request
    req, err := CreateRequest(ctx, opts)
    if err != nil {
        if opts.Metrics != nil {
            opts.Metrics.EndTime = time.Now()
            opts.Metrics.Duration = opts.Metrics.EndTime.Sub(opts.Metrics.StartTime)
            opts.Metrics.Error = err.Error()
        }
        return nil, "", err
    }

    // Apply middleware
    req, err = ApplyMiddleware(req, opts.Middleware)
    if err != nil {
        if opts.Metrics != nil {
            opts.Metrics.EndTime = time.Now()
            opts.Metrics.Duration = opts.Metrics.EndTime.Sub(opts.Metrics.StartTime)
            opts.Metrics.Error = err.Error()
        }
        return nil, "", err
    }

    // Execute request with retries
    resp, err := executeWithRetries(httpClient, req, opts)
    if err != nil {
        if opts.Metrics != nil {
            opts.Metrics.EndTime = time.Now()
            opts.Metrics.Duration = opts.Metrics.EndTime.Sub(opts.Metrics.StartTime)
            opts.Metrics.Error = err.Error()
        }
        return nil, "", err
    }

    // Decompress response if needed
    if opts.Compress {
        if err := DecompressResponse(resp); err != nil {
            resp.Body.Close()
            if opts.Metrics != nil {
                opts.Metrics.EndTime = time.Now()
                opts.Metrics.Duration = opts.Metrics.EndTime.Sub(opts.Metrics.StartTime)
                opts.Metrics.StatusCode = resp.StatusCode
                opts.Metrics.Error = err.Error()
            }
            return nil, "", fmt.Errorf("failed to decompress response: %w", err)
        }
    }

    // Read the response body
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        if opts.Metrics != nil {
            opts.Metrics.EndTime = time.Now()
            opts.Metrics.Duration = opts.Metrics.EndTime.Sub(opts.Metrics.StartTime)
            opts.Metrics.StatusCode = resp.StatusCode
            opts.Metrics.Error = err.Error()
        }
        return nil, "", fmt.Errorf("failed to read response body: %v", err)
    }
    resp.Body.Close()
    bodyString := string(bodyBytes)

    // Finalize metrics - SUCCESS
    if opts.Metrics != nil {
        opts.Metrics.EndTime = time.Now()
        opts.Metrics.Duration = opts.Metrics.EndTime.Sub(opts.Metrics.StartTime)
        opts.Metrics.StatusCode = resp.StatusCode
        opts.Metrics.ResponseSize = int64(len(bodyBytes))
    }

    // Handle output
    err = HandleOutput(bodyString, opts)
    if err != nil {
        return nil, "", err
    }

    // Recreate the response body for further use
    resp.Body = ioutil.NopCloser(strings.NewReader(bodyString))

    return resp, bodyString, nil
}
```

```go
// retry.go
func executeWithRetries(client options.HTTPClient, req *http.Request, opts *options.RequestOptions) (*http.Response, error) {
    var resp *http.Response
    var err error
    var bodyBytes []byte

    // ... existing context check code ...

    retries := 0
    if opts.RetryConfig != nil {
        retries = opts.RetryConfig.MaxRetries
    }

    for attempt := 0; attempt <= retries; attempt++ {
        // Update retry count in metrics
        if opts.Metrics != nil && attempt > 0 {
            opts.Metrics.RetryCount = attempt
        }

        // ... existing retry logic ...

        resp, err = client.Do(attemptReq)

        // ... rest of retry logic ...
    }

    return resp, nil
}
```

---

## Testing Required

### Unit Tests

```go
// metrics_test.go
func TestMetricsCollection(t *testing.T) {
    metrics := &options.RequestMetrics{}

    opts := &options.RequestOptions{
        URL:     "https://httpbin.org/get",
        Metrics: metrics,
    }

    ctx := context.Background()
    resp, _, err := Process(ctx, opts)

    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }

    // Verify metrics were populated
    if metrics.StartTime.IsZero() {
        t.Error("StartTime not set")
    }
    if metrics.EndTime.IsZero() {
        t.Error("EndTime not set")
    }
    if metrics.Duration == 0 {
        t.Error("Duration not calculated")
    }
    if metrics.StatusCode != 200 {
        t.Errorf("Expected status 200, got %d", metrics.StatusCode)
    }
    if metrics.ResponseSize == 0 {
        t.Error("ResponseSize not set")
    }

    t.Logf("Metrics: Duration=%v, Status=%d, Size=%d bytes",
        metrics.Duration, metrics.StatusCode, metrics.ResponseSize)
}

func TestMetricsWithRetries(t *testing.T) {
    metrics := &options.RequestMetrics{}

    opts := &options.RequestOptions{
        URL: "https://httpbin.org/status/503", // Will fail
        Metrics: metrics,
        RetryConfig: &options.RetryConfig{
            MaxRetries: 3,
        },
    }

    ctx := context.Background()
    _, _, err := Process(ctx, opts)

    // Request should fail after retries
    if err == nil {
        t.Error("Expected error due to 503 status")
    }

    // Verify retry count
    if metrics.RetryCount != 3 {
        t.Errorf("Expected 3 retries, got %d", metrics.RetryCount)
    }

    if metrics.Error == "" {
        t.Error("Error not recorded in metrics")
    }
}

func TestMetricsOptional(t *testing.T) {
    // No metrics provided - should not crash
    opts := &options.RequestOptions{
        URL:     "https://httpbin.org/get",
        Metrics: nil, // Not collecting metrics
    }

    ctx := context.Background()
    _, _, err := Process(ctx, opts)

    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
}
```

---

## Current Impact

### On Users

**Low Impact** - Feature is optional:
- Users who don't set `Metrics` are unaffected
- Users who DO set `Metrics` get an empty struct (silent failure)
- No crashes, just missing functionality

### On Documentation

**Documentation Claims** (from MIDDLEWARE_VS_DECODER_PATTERNS.md):
```go
// METRICS COLLECTION - Observability
opts.Metrics = &options.RequestMetrics{
    StartTime: time.Now(),
}

// Metrics populated after request
logger.Printf("Duration: %v", opts.Metrics.Duration)
logger.Printf("Status: %d", opts.Metrics.StatusCode)
```

**Reality**: ❌ This code won't work - metrics are NOT populated

---

## Recommendation

### Option 1: Implement Basic Metrics (Recommended for Beta)

**Effort**: 2-3 hours
**Scope**: Easy metrics only (StartTime, EndTime, Duration, StatusCode, ResponseSize, RetryCount, Error)
**Value**: High - gives users immediate observability
**Complexity**: Low - just add time tracking

### Option 2: Full Implementation with httptrace

**Effort**: 6-8 hours
**Scope**: All 12 fields including DNSLookupTime, ConnectTime, TLSTime, FirstByteTime
**Value**: Very High - production-grade observability
**Complexity**: Medium - requires httptrace integration

### Option 3: Document as "Not Yet Implemented"

**Effort**: 30 minutes
**Scope**: Add note to documentation
**Value**: Transparency
**Complexity**: None

### Option 4: Remove Feature (Not Recommended)

**Effort**: 1 hour
**Scope**: Remove Metrics field and type
**Value**: Simplification
**Risk**: Removes valuable future feature

---

## Decision

### For Beta Release (October 18):

✅ **Implement Option 1** - Basic Metrics
- Easy to implement (2-3 hours)
- Provides immediate value
- No breaking changes
- Users can start using it right away

### For v1.0 Release (November 8):

✅ **Implement Option 2** - Full httptrace Integration
- Complete feature implementation
- Production-ready observability
- Industry-standard metrics collection

---

## Status Summary

| Aspect | Status | Notes |
|--------|--------|-------|
| Type Definition | ✅ Complete | 12 comprehensive fields |
| Field in RequestOptions | ✅ Complete | Optional pointer |
| Clone Support | ✅ Complete | Deep copy implemented |
| **Collection Logic** | ❌ **MISSING** | **No implementation** |
| **Population Logic** | ❌ **MISSING** | **No implementation** |
| Documentation | ⚠️ Misleading | Claims it works (it doesn't) |
| Tests | ❌ Missing | No tests for metrics |
| User Impact | ⚠️ Low | Silent failure (no crash) |

---

## Next Steps

1. **Immediate** (for Beta):
   - Implement basic metrics collection (2-3 hours)
   - Add tests for metrics functionality
   - Update documentation with working examples

2. **Before v1.0**:
   - Implement httptrace integration for detailed timing
   - Add comprehensive metrics tests
   - Benchmark metrics overhead (should be <1%)

3. **Documentation**:
   - Create METRICS_IMPLEMENTATION.md guide
   - Add examples to README
   - Show integration with monitoring tools

---

**Report Date**: October 14, 2025 15:30 PM
**Priority**: Medium (Not blocking beta, but valuable feature)
**Estimated Implementation**: 2-3 hours (basic) or 6-8 hours (full)
**Recommendation**: Implement basic metrics for beta, full metrics for v1.0
