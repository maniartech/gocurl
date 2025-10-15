# FINAL DECISION: Metrics Architecture

**Date**: October 14, 2025
**Status**: ✅ **DECISION MADE**

---

## Current State

### Already Have ✅

```go
// options/options.go (line 78)
Context           context.Context              `json:"-"` // Not exported to JSON
ContextCancel     context.CancelFunc           `json:"-"` // Cancel function for context cleanup
```

**Perfect!** Context is already there for distributed tracing.

### Currently Have (Need to Remove) ❌

```go
// options/options.go (line 84)
Metrics           *RequestMetrics              `json:"-"` // Request metrics for observability
```

**Problem:** Not implemented, overkill for curl-compatible library.

---

## Final Architecture Decision

### What We Have (Keep)

✅ `Context context.Context` - For distributed tracing, cancellation
✅ `ContextCancel context.CancelFunc` - For cleanup
✅ `Middleware []middlewares.MiddlewareFunc` - For request modification

### What to Remove

❌ `Metrics *RequestMetrics` - Wrong abstraction, not implemented

### What to Add

✅ `Hooks *RequestHooks` - For observability integration

---

## Simplified Solution: Just Add Hooks

Since Context already exists, we only need to add hooks!

### Add to options.go

```go
// After line 81 (after Middleware field)
type RequestOptions struct {
    // ... existing fields ...

    Context           context.Context              `json:"-"` // Already exists ✅
    ContextCancel     context.CancelFunc           `json:"-"` // Already exists ✅
    RequestID         string                       `json:"request_id,omitempty"`
    Middleware        []middlewares.MiddlewareFunc `json:"-"`
    ResponseBodyLimit int64                        `json:"response_body_limit,omitempty"`
    ResponseDecoder   ResponseDecoder              `json:"-"`
    Hooks             *RequestHooks                `json:"-"` // ✅ ADD THIS
    // Metrics        *RequestMetrics              `json:"-"` // ❌ REMOVE THIS
    CustomClient      HTTPClient                   `json:"-"`
}

// RequestHooks provides lifecycle callbacks for observability.
// All hooks are optional (nil-safe).
type RequestHooks struct {
    // OnRequestStart is called after request is created, before sending.
    // Use for: start timers, inject trace headers, audit logging.
    OnRequestStart func(ctx context.Context, req *http.Request)

    // OnRequestEnd is called after request completes (success or failure).
    // Use for: metrics collection, distributed tracing, error logging.
    // Always receives duration even if request failed.
    OnRequestEnd func(ctx context.Context, resp *http.Response, duration time.Duration, err error)

    // OnRetry is called before each retry attempt (not called on first attempt).
    // Use for: retry metrics, backoff logging, circuit breaker logic.
    OnRetry func(ctx context.Context, attempt int, lastErr error)
}
```

### Remove from options.go

```go
// DELETE these lines (144-157):
// RequestMetrics represents metrics collected during a request.
type RequestMetrics struct {
    StartTime     time.Time     `json:"start_time"`
    EndTime       time.Time     `json:"end_time"`
    Duration      time.Duration `json:"duration"`
    DNSLookupTime time.Duration `json:"dns_lookup_time"`
    ConnectTime   time.Duration `json:"connect_time"`
    TLSTime       time.Duration `json:"tls_time"`
    FirstByteTime time.Duration `json:"first_byte_time"`
    RetryCount    int           `json:"retry_count"`
    ResponseSize  int64         `json:"response_size"`
    RequestSize   int64         `json:"request_size"`
    StatusCode    int           `json:"status_code"`
    Error         string        `json:"error,omitempty"`
}
```

```go
// DELETE from Clone() method (lines 192-195):
if ro.Metrics != nil {
    clonedMetrics := *ro.Metrics
    clone.Metrics = &clonedMetrics
}
```

---

## Implementation in process.go

### Modify Process() function

```go
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
    // Use provided context or create new one
    if opts.Context == nil {
        opts.Context = ctx
    }

    // Validate options
    if err := ValidateOptions(opts); err != nil {
        return nil, "", err
    }

    // ... existing client creation code ...

    // Create request
    req, err := CreateRequest(opts.Context, opts)
    if err != nil {
        return nil, "", err
    }

    // Apply middleware
    req, err = ApplyMiddleware(req, opts.Middleware)
    if err != nil {
        return nil, "", err
    }

    // 🆕 Hook: Request start
    if opts.Hooks != nil && opts.Hooks.OnRequestStart != nil {
        opts.Hooks.OnRequestStart(opts.Context, req)
    }

    // Execute request with retries
    startTime := time.Now()
    resp, err := executeWithRetries(httpClient, req, opts)
    duration := time.Since(startTime)

    // 🆕 Hook: Request end (always called, even on error)
    if opts.Hooks != nil && opts.Hooks.OnRequestEnd != nil {
        opts.Hooks.OnRequestEnd(opts.Context, resp, duration, err)
    }

    if err != nil {
        return nil, "", err
    }

    // ... rest of existing code ...

    return resp, bodyString, nil
}
```

### Modify executeWithRetries() in retry.go

```go
func executeWithRetries(client options.HTTPClient, req *http.Request, opts *options.RequestOptions) (*http.Response, error) {
    // ... existing code ...

    for attempt := 0; attempt <= retries; attempt++ {
        // 🆕 Hook: Retry (not called on first attempt)
        if attempt > 0 && opts.Hooks != nil && opts.Hooks.OnRetry != nil {
            opts.Hooks.OnRetry(opts.Context, attempt, lastErr)
        }

        // ... existing retry logic ...

        resp, err = client.Do(attemptReq)

        // ... existing retry decision code ...
    }

    return resp, nil
}
```

---

## Usage Examples

### Example 1: Simple Duration Logging

```go
opts := &options.RequestOptions{
    URL: "https://api.example.com",
    Hooks: &options.RequestHooks{
        OnRequestEnd: func(ctx context.Context, resp *http.Response, duration time.Duration, err error) {
            log.Printf("Request took %v, status: %d", duration, resp.StatusCode)
        },
    },
}
```

### Example 2: OpenTelemetry Distributed Tracing

```go
ctx, span := tracer.Start(context.Background(), "api-call")
defer span.End()

opts := &options.RequestOptions{
    URL:     "https://api.example.com",
    Context: ctx, // ✅ Already supported!
    Hooks: &options.RequestHooks{
        OnRequestStart: func(ctx context.Context, req *http.Request) {
            span.SetAttributes(
                attribute.String("http.method", req.Method),
                attribute.String("http.url", req.URL.String()),
            )
        },
        OnRequestEnd: func(ctx context.Context, resp *http.Response, duration time.Duration, err error) {
            if resp != nil {
                span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
            }
            if err != nil {
                span.RecordError(err)
            }
        },
    },
}
```

### Example 3: Prometheus Metrics

```go
var (
    httpDuration = prometheus.NewHistogramVec(...)
    httpTotal    = prometheus.NewCounterVec(...)
)

opts := &options.RequestOptions{
    URL: "https://api.example.com",
    Hooks: &options.RequestHooks{
        OnRequestEnd: func(ctx context.Context, resp *http.Response, duration time.Duration, err error) {
            status := "error"
            if resp != nil {
                status = strconv.Itoa(resp.StatusCode)
            }

            httpDuration.WithLabelValues(opts.Method, status).Observe(duration.Seconds())
            httpTotal.WithLabelValues(opts.Method, status).Inc()
        },
        OnRetry: func(ctx context.Context, attempt int, lastErr error) {
            retryTotal.WithLabelValues(opts.URL).Inc()
        },
    },
}
```

### Example 4: Custom Metrics Struct (User Controlled)

```go
type MyMetrics struct {
    Duration   time.Duration
    StatusCode int
    Success    bool
    Retries    int
}

metrics := &MyMetrics{}

opts := &options.RequestOptions{
    URL: "https://api.example.com",
    Hooks: &options.RequestHooks{
        OnRequestEnd: func(ctx context.Context, resp *http.Response, duration time.Duration, err error) {
            metrics.Duration = duration
            if resp != nil {
                metrics.StatusCode = resp.StatusCode
            }
            metrics.Success = err == nil && resp != nil && resp.StatusCode < 400
        },
        OnRetry: func(ctx context.Context, attempt int, lastErr error) {
            metrics.Retries = attempt
        },
    },
}

// After request
fmt.Printf("Duration: %v, Status: %d, Retries: %d\n",
    metrics.Duration, metrics.StatusCode, metrics.Retries)
```

---

## Why This is the Right Solution

### ✅ Leverages Existing Context

- Already have `Context context.Context` for distributed tracing
- No need to add it again
- Just need hooks to tap into lifecycle

### ✅ Enterprise/Military-Grade

- **OpenTelemetry compatible** - Context propagation already works
- **Pluggable backends** - Works with Prometheus, DataDog, any system
- **Zero overhead** - Nil checks, no allocation when not used
- **Compliance ready** - User controls what's logged (PII redaction)
- **Security** - User can audit log in OnRequestStart

### ✅ Follows SSR Philosophy

**Sweet:**
- Simple hook API
- Clear lifecycle events
- Obvious how to use

**Simple:**
- 1 new field (`Hooks`)
- 1 new type (`RequestHooks` with 3 callbacks)
- Remove 1 complex type (`RequestMetrics` with 12 fields)
- Net: -11 fields, +3 callbacks

**Robust:**
- Zero-allocation when `Hooks` is nil
- Can't break request flow (hooks are observers)
- Thread-safe (no shared state in hooks)

### ✅ Minimal Changes

**Remove:**
- `Metrics *RequestMetrics` field (line 84)
- `RequestMetrics` type definition (lines 144-157)
- Metrics cloning in `Clone()` (lines 192-195)

**Add:**
- `Hooks *RequestHooks` field (1 line)
- `RequestHooks` type definition (3 callbacks)
- Hook calls in `Process()` (4 lines)
- Hook call in `executeWithRetries()` (3 lines)

**Total:** ~12 lines of implementation

---

## Implementation Checklist

### Phase 1: Remove Metrics (5 minutes)

- [ ] Delete `Metrics *RequestMetrics` from RequestOptions (line 84)
- [ ] Delete `RequestMetrics` type (lines 144-157)
- [ ] Delete metrics cloning from `Clone()` (lines 192-195)

### Phase 2: Add Hooks (10 minutes)

- [ ] Add `Hooks *RequestHooks` to RequestOptions
- [ ] Define `RequestHooks` type with 3 callbacks
- [ ] Add OnRequestStart hook in Process() (before executeWithRetries)
- [ ] Add OnRequestEnd hook in Process() (after executeWithRetries)
- [ ] Add OnRetry hook in executeWithRetries() (in retry loop)

### Phase 3: Documentation (15 minutes)

- [ ] Add hook examples to README
- [ ] Document OpenTelemetry integration
- [ ] Document Prometheus integration
- [ ] Document custom metrics patterns

### Phase 4: Testing (20 minutes)

- [ ] Test hooks are called correctly
- [ ] Test nil safety (no hooks = no crash)
- [ ] Test Context propagation works
- [ ] Test retry hook is called on retries only

**Total Time:** ~50 minutes

---

## Comparison: Before vs After

### Before (Current - BROKEN)

```go
type RequestOptions struct {
    // ... 30+ fields ...
    Metrics *RequestMetrics `json:"-"` // ❌ Not implemented
}

type RequestMetrics struct {
    // 12 fields that are never populated
    StartTime     time.Time
    EndTime       time.Time
    // ... 10 more unused fields
}

// Usage (BROKEN - fields stay zero)
metrics := &options.RequestMetrics{}
opts := &options.RequestOptions{
    Metrics: metrics,
}
// After request: metrics.Duration == 0 (NOT POPULATED!)
```

### After (Proposed - WORKING)

```go
type RequestOptions struct {
    // ... 30+ fields ...
    Context context.Context   `json:"-"` // ✅ Already exists
    Hooks   *RequestHooks     `json:"-"` // ✅ New, simple
}

type RequestHooks struct {
    OnRequestStart func(ctx context.Context, req *http.Request)
    OnRequestEnd   func(ctx context.Context, resp *http.Response, duration time.Duration, err error)
    OnRetry        func(ctx context.Context, attempt int, lastErr error)
}

// Usage (WORKS - user controls format)
var duration time.Duration
var statusCode int

opts := &options.RequestOptions{
    Hooks: &options.RequestHooks{
        OnRequestEnd: func(ctx, resp, d, err) {
            duration = d
            statusCode = resp.StatusCode
        },
    },
}
// After request: duration and statusCode are populated ✅
```

**Net Change:**
- Remove: 12-field struct that doesn't work
- Add: 3-callback hook that does work
- Result: Simpler, more flexible, actually functional

---

## Final Decision

### ✅ APPROVED: Hooks Architecture

**Remove:**
1. `Metrics *RequestMetrics` field
2. `RequestMetrics` type definition
3. Metrics cloning logic

**Add:**
1. `Hooks *RequestHooks` field
2. `RequestHooks` type with 3 callbacks
3. Hook invocations in Process() and executeWithRetries()

**Rationale:**
- Context already exists ✅
- Hooks are simpler than predefined metrics
- Works with any monitoring system
- Zero overhead when not used
- Enterprise/military-grade
- Follows SSR philosophy
- ~50 minutes to implement

**Priority:** P0 for Beta (Oct 18)

**Status:** Ready to implement

---

**Decision made:** Remove broken Metrics field, add working Hooks field.

**Next step:** Implement the changes.
