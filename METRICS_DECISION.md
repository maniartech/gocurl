# Metrics Decision: RequestOptions vs Middleware

**Date**: October 14, 2025
**Issue**: Should `Metrics` be a field in `RequestOptions` or implemented as middleware?
**Decision**: **REMOVE from RequestOptions, implement as middleware**

---

## Executive Summary

**Verdict**: `Metrics` in `RequestOptions` is **overkill and violates GoCurl's "Sweet, Simple, Robust" philosophy**.

### The Problem

1. ✅ Middleware already exists and works
2. ❌ Metrics in RequestOptions is **not implemented** (just structure)
3. ❌ Adds complexity to core library
4. ❌ Violates "Simple" principle from objective.md
5. ❌ Not needed for curl compatibility (core objective)

### The Solution

**Remove `Metrics` field entirely, provide middleware instead.**

---

## Analysis Against GoCurl Objectives

### 1. Primary Objective (from objective.md)

> "Deliver a zero-allocation, ultra-high-performance HTTP/HTTP2 client that allows Go developers to use HTTP-specific curl command syntax"

**Question**: Does Metrics in RequestOptions help achieve this?

❌ **NO** - Metrics:
- Is NOT part of curl syntax
- Does NOT help with curl compatibility
- Is NOT required for core functionality
- ADDS allocation overhead (pointer, 12 fields)
- COMPLICATES the zero-allocation goal

### 2. Sweet, Simple, Robust Philosophy

From objective.md:

#### Sweet (Developer Experience)

> "Minimal cognitive load: Copy-paste curl commands from documentation"

**Impact of Metrics in RequestOptions**:
- ❌ User must understand RequestMetrics struct
- ❌ User must create and pass Metrics pointer
- ❌ Increases API surface area
- ❌ Not curl-related, adds mental overhead

#### Simple (Implementation)

> "No over-engineering: Solve the problem, don't create frameworks"
> "Minimal dependencies: Leverage stdlib, avoid external complexity"

**Impact of Metrics in RequestOptions**:
- ❌ **Over-engineering** - metrics is monitoring, not core HTTP
- ❌ Adds 12 fields to core struct
- ❌ Requires httptrace integration for full implementation
- ❌ Increases maintenance burden
- ❌ Violates single responsibility principle

#### Robust (Performance & Reliability)

> "Zero-allocation on critical path: Pooled buffers, reused objects"

**Impact of Metrics in RequestOptions**:
- ❌ **Metrics pointer allocation** on every opt-in request
- ❌ RequestMetrics struct allocation (12 fields = 96+ bytes)
- ❌ Contradicts zero-allocation goal
- ❌ Optional feature pollutes critical path

### 3. Implementation Guardrails

From objective.md - "What We Don't Build":

> "❌ Complex state machines (sequential processing)"
> "❌ Plugin architecture (middleware is enough)"

**Metrics in RequestOptions violates this**:
- Middleware exists and is sufficient
- Adding metrics to RequestOptions is over-engineering
- Makes RequestOptions a "god object"

---

## Current Middleware System

### What Already Exists

```go
// middlewares/middlewares.go
type MiddlewareFunc func(*http.Request) (*http.Request, error)

// process.go
func ApplyMiddleware(req *http.Request, middleware []middlewares.MiddlewareFunc) (*http.Request, error) {
    for _, mw := range middleware {
        modifiedReq, err := mw(req)
        if err != nil {
            return nil, fmt.Errorf("middleware error: %v", err)
        }
        req = modifiedReq
    }
    return req, nil
}
```

### How It's Used

```go
opts := &options.RequestOptions{
    URL:        "https://api.example.com",
    Middleware: []middlewares.MiddlewareFunc{
        myAuthMiddleware,
        myLoggingMiddleware,
    },
}
```

### Why This Is Perfect for Metrics

✅ **Separation of concerns** - metrics is cross-cutting, not core HTTP
✅ **Optional by design** - users add if needed
✅ **Zero-allocation** - only allocates if user opts in
✅ **Composable** - combine with other middleware
✅ **Testable** - middleware can be unit tested independently
✅ **Extensible** - users can write custom metrics collectors

---

## Proposed Solution

### Remove Metrics from RequestOptions

**Delete these lines from options/options.go**:

```diff
- Metrics           *RequestMetrics              `json:"-"` // Request metrics for observability
```

```diff
- // RequestMetrics represents metrics collected during a request.
- // This is useful for observability, monitoring, and debugging in production.
- type RequestMetrics struct {
-     StartTime     time.Time     `json:"start_time"`
-     EndTime       time.Time     `json:"end_time"`
-     Duration      time.Duration `json:"duration"`
-     DNSLookupTime time.Duration `json:"dns_lookup_time"`
-     ConnectTime   time.Duration `json:"connect_time"`
-     TLSTime       time.Duration `json:"tls_time"`
-     FirstByteTime time.Duration `json:"first_byte_time"`
-     RetryCount    int           `json:"retry_count"`
-     ResponseSize  int64         `json:"response_size"`
-     RequestSize   int64         `json:"request_size"`
-     StatusCode    int           `json:"status_code"`
-     Error         string        `json:"error,omitempty"`
- }
```

```diff
- if ro.Metrics != nil {
-     clonedMetrics := *ro.Metrics
-     clone.Metrics = &clonedMetrics
- }
```

**Impact**:
- Simplifies RequestOptions (already has 30+ fields!)
- Reduces API surface area
- Removes unimplemented feature
- Aligns with "Simple" philosophy

### Provide Metrics Middleware Instead

Create `middlewares/metrics.go`:

```go
package middlewares

import (
    "net/http"
    "time"
)

// MetricsCollector is a simple metrics collection interface.
type MetricsCollector struct {
    StartTime time.Time
    EndTime   time.Time
    Duration  time.Duration
}

// NewMetricsMiddleware creates a middleware that collects basic metrics.
// Users can extend this for custom metrics collection.
func NewMetricsMiddleware(collector *MetricsCollector) MiddlewareFunc {
    return func(req *http.Request) (*http.Request, error) {
        collector.StartTime = time.Now()

        // Attach cleanup to request context if needed
        // (Or handle in response callback - see below)

        return req, nil
    }
}
```

**Problem**: Middleware only modifies requests, not responses!

---

## The REAL Issue: Middleware Can't Access Responses

### Current Middleware Limitation

```go
type MiddlewareFunc func(*http.Request) (*http.Request, error)
```

**This ONLY modifies requests** - can't collect response metrics!

### Two Options

#### Option 1: Response Callbacks (Simple, Aligned with Objectives)

Add **optional** response callback to RequestOptions:

```go
// RequestOptions
type RequestOptions struct {
    // ... existing fields ...

    // Optional callback invoked after response received
    OnResponse func(*http.Response, time.Duration, error)
}
```

**Usage**:

```go
opts := &options.RequestOptions{
    URL: "https://api.example.com",
    OnResponse: func(resp *http.Response, duration time.Duration, err error) {
        // User's custom metrics collection
        log.Printf("Request took %v, status: %d", duration, resp.StatusCode)
        myMetricsSystem.RecordHTTPRequest(duration, resp.StatusCode)
    },
}
```

**Pros**:
✅ Simple - just a callback function
✅ Zero allocation if not used (nil check)
✅ User controls what metrics to collect
✅ No predefined struct (flexibility)
✅ Aligns with "Simple" philosophy
✅ Works with ANY metrics system (Prometheus, DataDog, etc.)

**Cons**:
❌ User must implement their own metrics struct
❌ Doesn't provide "batteries included" metrics

#### Option 2: Response Middleware (Complex, Over-Engineering)

Extend middleware to handle responses:

```go
type MiddlewareFunc func(*http.Request) (*http.Request, error)
type ResponseMiddlewareFunc func(*http.Response, error) (*http.Response, error)

type RequestOptions struct {
    Middleware         []MiddlewareFunc
    ResponseMiddleware []ResponseMiddlewareFunc
}
```

**Pros**:
✅ Symmetrical request/response handling
✅ Composable

**Cons**:
❌ **Over-engineering** - adds complexity
❌ Violates "Simple" principle
❌ Most use cases don't need this
❌ Creates another abstraction layer

---

## Recommendation: Progressive Enhancement

### Phase 1: Remove Metrics Field (Immediate)

**Action**: Delete `Metrics *RequestMetrics` from RequestOptions

**Rationale**:
- Not implemented anyway
- Reduces API complexity
- Aligns with SSR philosophy
- Zero breaking changes (wasn't used)

### Phase 2: Add OnResponse Callback (Beta Release)

**Action**: Add simple callback to RequestOptions

```go
type RequestOptions struct {
    // ... existing fields ...

    // OnResponse is called after request completion with response and duration.
    // This is optional and only called if set. Use for custom metrics collection.
    OnResponse func(resp *http.Response, duration time.Duration, err error)
}
```

**Implementation** (in process.go):

```go
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
    startTime := time.Now()

    // ... existing code ...

    // At the end, before return:
    if opts.OnResponse != nil {
        opts.OnResponse(resp, time.Since(startTime), err)
    }

    return resp, bodyString, err
}
```

**Usage Examples**:

```go
// Example 1: Simple logging
opts := &options.RequestOptions{
    URL: "https://api.example.com",
    OnResponse: func(resp *http.Response, duration time.Duration, err error) {
        log.Printf("[%s] %s - %v - %d",
            opts.Method, opts.URL, duration, resp.StatusCode)
    },
}

// Example 2: Prometheus metrics
opts := &options.RequestOptions{
    URL: "https://api.example.com",
    OnResponse: func(resp *http.Response, duration time.Duration, err error) {
        httpRequestDuration.WithLabelValues(
            opts.Method,
            resp.StatusCode,
        ).Observe(duration.Seconds())
    },
}

// Example 3: Custom struct (user's choice)
type MyMetrics struct {
    Duration   time.Duration
    StatusCode int
    Success    bool
}

metrics := &MyMetrics{}
opts := &options.RequestOptions{
    URL: "https://api.example.com",
    OnResponse: func(resp *http.Response, duration time.Duration, err error) {
        metrics.Duration = duration
        metrics.StatusCode = resp.StatusCode
        metrics.Success = err == nil && resp.StatusCode < 400
    },
}
```

### Phase 3: Optional Helpers Package (Post v1.0)

**If users ask for it**, create `gocurl/metrics` package:

```go
package metrics

import (
    "net/http"
    "time"
)

// Standard metrics struct (optional helper)
type RequestMetrics struct {
    Duration   time.Duration
    StatusCode int
    Error      error
}

// Collector creates an OnResponse callback that populates metrics
func Collector(m *RequestMetrics) func(*http.Response, time.Duration, error) {
    return func(resp *http.Response, duration time.Duration, err error) {
        m.Duration = duration
        m.StatusCode = resp.StatusCode
        m.Error = err
    }
}

// Usage:
// metrics := &metrics.RequestMetrics{}
// opts.OnResponse = metrics.Collector(metrics)
```

---

## Comparison: Current vs Proposed

### Current (Broken)

```go
// Define metrics struct
metrics := &options.RequestMetrics{}

// Pass to options
opts := &options.RequestOptions{
    URL:     "https://api.example.com",
    Metrics: metrics,  // NOT IMPLEMENTED - fields stay empty!
}

resp, body, err := gocurl.Process(ctx, opts)

// Try to use metrics (BROKEN - all zeros)
fmt.Printf("Duration: %v\n", metrics.Duration)  // 0
fmt.Printf("Status: %d\n", metrics.StatusCode)   // 0
```

**Problems**:
❌ Predefined struct (inflexible)
❌ Not implemented
❌ Allocates even if not needed
❌ Adds 12 fields to core library
❌ Pollutes RequestOptions

### Proposed (Clean)

```go
// User defines their own metrics (or uses any monitoring system)
var requestDuration time.Duration
var statusCode int

opts := &options.RequestOptions{
    URL: "https://api.example.com",
    OnResponse: func(resp *http.Response, duration time.Duration, err error) {
        requestDuration = duration
        statusCode = resp.StatusCode

        // Or send to Prometheus, DataDog, etc.
    },
}

resp, body, err := gocurl.Process(ctx, opts)

fmt.Printf("Duration: %v\n", requestDuration)
fmt.Printf("Status: %d\n", statusCode)
```

**Benefits**:
✅ Simple callback (1 field vs 1+ struct)
✅ Zero allocation if not used (nil check)
✅ User controls metrics structure
✅ Works with ANY monitoring system
✅ Follows SSR philosophy
✅ Actually implemented (5 lines of code)

---

## Why Metrics in RequestOptions is Wrong

### 1. Violates Single Responsibility

RequestOptions should handle **request configuration**, not **observability**.

Metrics is a cross-cutting concern, like:
- Logging (not in RequestOptions)
- Tracing (not in RequestOptions)
- Rate limiting (not in RequestOptions)
- Circuit breaking (not in RequestOptions)

**Why should metrics be special?**

### 2. Forces Implementation Decisions

Current Metrics struct assumes users want:
- DNSLookupTime
- ConnectTime
- TLSTime
- FirstByteTime

**But**:
- Some users only want duration
- Some users have Prometheus (different format)
- Some users have DataDog (different format)
- Some users want custom fields

**Predefined struct locks users into OUR choice.**

### 3. Increases Coupling

Adding Metrics to RequestOptions means:
- Process() must understand metrics
- Retry logic must update metrics
- Clone() must copy metrics
- Every change to metrics affects core library

**Callback decouples**: Process() just calls a function, doesn't know what it does.

### 4. Not Curl-Compatible

Curl doesn't have metrics in request options.

Curl has `--write-out` for timing:

```bash
curl -w "%{time_total}" https://api.example.com
```

**This is OUTPUT formatting**, not request configuration!

GoCurl equivalent:
```go
// This is OUTPUT handling, not request option
opts.OnResponse = func(resp, duration, err) {
    fmt.Printf("time_total: %v\n", duration)
}
```

### 5. The "Just Because" Fallacy

> "But other HTTP libraries have metrics!"

**Counterargument**:
- Other libraries aren't zero-allocation
- Other libraries aren't curl-compatible
- GoCurl's goal is **simplicity**, not feature parity

From objective.md:

> "No over-engineering: Solve the problem, don't create frameworks"

---

## Decision Matrix

| Criterion | Metrics in RequestOptions | OnResponse Callback |
|-----------|---------------------------|---------------------|
| **Simplicity** | ❌ Complex (12 fields, struct) | ✅ Simple (1 callback) |
| **Zero-allocation** | ❌ Always allocates pointer + struct | ✅ Zero if nil |
| **Flexibility** | ❌ Locked to predefined struct | ✅ User defines metrics |
| **SSR Philosophy** | ❌ Violates "Simple" | ✅ Aligns perfectly |
| **Curl Compatibility** | ❌ Not curl-related | ✅ Like --write-out |
| **Implementation** | ❌ Not implemented (8+ hours) | ✅ 5 lines of code |
| **Maintenance** | ❌ High (core library change) | ✅ Low (optional callback) |
| **User Freedom** | ❌ Forces our metrics format | ✅ Use any monitoring system |

**Winner**: OnResponse Callback (7-1)

---

## Final Recommendation

### Immediate Action (Today)

1. **Remove `Metrics *RequestMetrics` from RequestOptions**
2. **Remove `RequestMetrics` struct from options.go**
3. **Remove metrics cloning from Clone()**
4. **Update METRICS_STATUS.md to explain decision**

### Beta Release (Oct 18)

1. **Add `OnResponse` callback to RequestOptions**
2. **Implement in Process()** (5 lines)
3. **Document with examples** (Prometheus, logging, custom)

### Post v1.0 (If Requested)

1. **Create `gocurl/metrics` helper package** (optional)
2. **Provide common collectors** (Prometheus, DataDog adapters)

---

## Conclusion

**Metrics in RequestOptions is overkill because**:

1. ❌ **Violates SSR philosophy** - over-engineering
2. ❌ **Not implemented** - waste of effort
3. ❌ **Adds complexity** - 12 fields, allocations
4. ❌ **Locks user choice** - predefined struct
5. ❌ **Not curl-compatible** - metrics is observability, not request config
6. ✅ **Middleware exists** - but can't access responses
7. ✅ **Callback is better** - simple, flexible, zero-allocation

**OnResponse callback is the right solution**:

✅ Aligns with "Sweet, Simple, Robust"
✅ Zero-allocation when not used
✅ User controls metrics format
✅ Works with any monitoring system
✅ Takes 5 minutes to implement
✅ Minimal API surface increase (1 field vs 12)

---

**Decision**: Remove Metrics field, add OnResponse callback.

**Priority**: P1 (Beta blocker - clean up API before release)

**Effort**: 30 minutes (delete code + 5 line callback)

**Impact**: Simplifies library, aligns with objectives, provides better solution
