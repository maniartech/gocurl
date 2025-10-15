# Middleware Limitation Analysis

**Date**: October 14, 2025
**Issue**: Why middleware can't handle metrics collection

---

## The Problem with Current Middleware

### Current Definition

```go
// middlewares/middlewares.go
type MiddlewareFunc func(*http.Request) (*http.Request, error)
```

### Critical Limitation

**Middleware can ONLY modify requests, NOT responses!**

```go
// Where it's called (process.go:67-68)
req, err = ApplyMiddleware(req, opts.Middleware)
if err != nil {
    return nil, "", err
}

// Then later...
resp, err := executeWithRetries(httpClient, req, opts)
```

### Why This Breaks Metrics

Metrics needs:
- ✅ `StartTime` - CAN capture in middleware (before request)
- ❌ `EndTime` - **CANNOT** capture (middleware runs before request)
- ❌ `Duration` - **CANNOT** calculate (no end time)
- ❌ `StatusCode` - **CANNOT** access (no response)
- ❌ `ResponseSize` - **CANNOT** access (no response)
- ❌ `RetryCount` - **CANNOT** access (retries happen after middleware)

**Middleware runs BEFORE the request is sent!**

---

## Why Middleware Architecture is Limited

### Request Flow

```
1. CreateRequest()           <- Creates *http.Request
2. ApplyMiddleware()          <- Modifies *http.Request
3. executeWithRetries()       <- Sends request, gets response
4. DecompressResponse()       <- Processes response
5. ioutil.ReadAll()          <- Reads response body
6. HandleOutput()            <- Writes output
7. Return                    <- Returns response
```

**Middleware happens at step 2** - no access to response (steps 3-7)!

### What Middleware CAN Do

✅ Add headers to request
✅ Modify request body
✅ Add authentication
✅ Log request details
✅ Validate request before sending

### What Middleware CANNOT Do

❌ Access response status code
❌ Access response headers
❌ Access response body
❌ Measure request duration
❌ Count retries
❌ Collect timing metrics

---

## Why "Just Use Middleware" Doesn't Work

### Attempted Metrics Middleware (BROKEN)

```go
// This CANNOT work!
func MetricsMiddleware(metrics *RequestMetrics) middlewares.MiddlewareFunc {
    return func(req *http.Request) (*http.Request, error) {
        metrics.StartTime = time.Now()

        // ❌ PROBLEM: We return here, before request is sent!
        // ❌ No way to capture EndTime, StatusCode, ResponseSize, etc.

        return req, nil
    }
}
```

**The middleware returns BEFORE the request is executed!**

### The Closure Attempt (Still Broken)

```go
// Try to use closure to capture end time?
func MetricsMiddleware(metrics *RequestMetrics) middlewares.MiddlewareFunc {
    return func(req *http.Request) (*http.Request, error) {
        metrics.StartTime = time.Now()

        // ❌ Can't defer - we don't have the response yet!
        // ❌ Can't wait - middleware must return immediately

        return req, nil
    }
}
```

**Middleware is synchronous and returns before request execution!**

---

## Three Solutions

### Solution 1: Response Middleware (Extend Architecture)

**Add second middleware type for responses:**

```go
// middlewares/middlewares.go
type MiddlewareFunc func(*http.Request) (*http.Request, error)
type ResponseMiddlewareFunc func(*http.Response, time.Duration, error) error

// options/options.go
type RequestOptions struct {
    // ... existing ...
    Middleware         []middlewares.MiddlewareFunc
    ResponseMiddleware []middlewares.ResponseMiddlewareFunc
}
```

**Implementation in process.go:**

```go
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
    startTime := time.Now()

    // ... existing request handling ...

    resp, err := executeWithRetries(httpClient, req, opts)

    // Apply response middleware
    duration := time.Since(startTime)
    if err2 := ApplyResponseMiddleware(resp, duration, err, opts.ResponseMiddleware); err2 != nil {
        // Log but don't fail request
        log.Printf("response middleware error: %v", err2)
    }

    // ... continue with response processing ...
}

func ApplyResponseMiddleware(resp *http.Response, duration time.Duration, err error, middleware []middlewares.ResponseMiddlewareFunc) error {
    for _, mw := range middleware {
        if err := mw(resp, duration, err); err != nil {
            return err
        }
    }
    return nil
}
```

**Usage:**

```go
metrics := &MyMetrics{}
opts := &options.RequestOptions{
    URL: "https://api.example.com",
    ResponseMiddleware: []middlewares.ResponseMiddlewareFunc{
        func(resp *http.Response, duration time.Duration, err error) error {
            metrics.Duration = duration
            metrics.StatusCode = resp.StatusCode
            metrics.Error = err
            return nil
        },
    },
}
```

**Pros:**
✅ Composable (chain multiple response handlers)
✅ Symmetrical with request middleware
✅ Type-safe
✅ Reusable middleware functions

**Cons:**
❌ Increases complexity (two middleware types)
❌ ResponseMiddleware can't modify response (or can it?)
❌ Need to define middleware signature carefully
❌ Violates "Simple" principle (more abstraction)

---

### Solution 2: OnResponse Callback (Simpler)

**Add single callback field:**

```go
// options/options.go
type RequestOptions struct {
    // ... existing ...

    // OnResponse is called after request completion.
    // Use for metrics collection, logging, etc.
    OnResponse func(*http.Response, time.Duration, error)
}
```

**Implementation:**

```go
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
    startTime := time.Now()

    // ... existing code ...

    resp, err := executeWithRetries(httpClient, req, opts)

    // Call response handler if provided
    if opts.OnResponse != nil {
        opts.OnResponse(resp, time.Since(startTime), err)
    }

    // ... continue processing ...
}
```

**Usage:**

```go
var duration time.Duration
var statusCode int

opts := &options.RequestOptions{
    URL: "https://api.example.com",
    OnResponse: func(resp *http.Response, d time.Duration, err error) {
        duration = d
        statusCode = resp.StatusCode
    },
}
```

**Pros:**
✅ **Simple** - just a callback
✅ Zero-allocation when nil
✅ No new types needed
✅ Flexible - user defines what to capture
✅ Aligns with SSR philosophy

**Cons:**
❌ Only one callback (not composable)
❌ Less structured than middleware
❌ Can't chain handlers easily

---

### Solution 3: Metrics Field + Implementation (Original Plan)

Keep `Metrics *RequestMetrics` in RequestOptions and implement it.

**Pros:**
✅ Batteries-included metrics
✅ Predefined structure

**Cons:**
❌ Forces metrics format on users
❌ Not composable
❌ Violates SSR "Simple" principle
❌ Adds 12 fields to core library
❌ Allocates even when not needed
❌ **Already rejected in METRICS_DECISION.md**

---

## Recommendation: Hybrid Approach

### Phase 1: OnResponse Callback (Beta - Oct 18)

**Add simple callback for most common use case:**

```go
type RequestOptions struct {
    // ... existing fields ...

    // OnResponse is called after request completion with response and duration.
    // Optional callback for metrics collection, logging, etc.
    OnResponse func(resp *http.Response, duration time.Duration, err error)
}
```

**Implementation: 5 lines in process.go**

**Covers 90% of use cases** (simple metrics, logging, monitoring)

### Phase 2: Response Middleware (Post v1.0, if needed)

**If users need composable response handling:**

```go
type ResponseMiddlewareFunc func(*http.Response, time.Duration, error) error

type RequestOptions struct {
    // ... existing ...
    Middleware         []middlewares.MiddlewareFunc         // Request modification
    ResponseMiddleware []middlewares.ResponseMiddlewareFunc // Response observation
}
```

**Only add if users actually request it** (YAGNI principle)

---

## Comparison Table

| Feature | Current Middleware | Response Middleware | OnResponse Callback |
|---------|-------------------|---------------------|---------------------|
| **Access Request** | ✅ Can modify | ✅ Read-only | ✅ Via closure |
| **Access Response** | ❌ No access | ✅ Full access | ✅ Full access |
| **Measure Duration** | ❌ No | ✅ Yes | ✅ Yes |
| **Composable** | ✅ Chain multiple | ✅ Chain multiple | ❌ Single callback |
| **Complexity** | Low | Medium | **Very Low** |
| **Implementation** | Exists | 20 lines | **5 lines** |
| **Zero-allocation** | ✅ When empty | ✅ When empty | ✅ When nil |
| **SSR Alignment** | ✅ Simple | ⚠️ More complex | ✅ **Simplest** |

**Winner for Beta**: OnResponse Callback (lowest complexity, fastest implementation)

**Consider for v1.0+**: Response Middleware (if composability is needed)

---

## Why OnResponse Callback is the Right Choice Now

### 1. Follows "Sweet, Simple, Robust"

**Sweet:**
- ✅ Dead simple API: just set a callback
- ✅ No new concepts to learn
- ✅ Obvious how to use

**Simple:**
- ✅ 1 field vs 2 middleware types
- ✅ 5 lines of implementation
- ✅ No abstraction layers

**Robust:**
- ✅ Zero-allocation when nil
- ✅ Can't break request flow
- ✅ Thread-safe (no shared state)

### 2. Solves 90% of Use Cases

**What users actually want:**
- Measure request duration ✅
- Log status codes ✅
- Send metrics to Prometheus ✅
- Record errors ✅
- Custom monitoring ✅

**What users rarely need:**
- Chaining multiple response handlers ❌ (can add later)
- Modifying responses in middleware ❌ (anti-pattern)

### 3. Can Evolve Later

```go
// Now (Beta):
opts.OnResponse = myHandler

// Future (v1.0+, if needed):
opts.ResponseMiddleware = []middlewares.ResponseMiddlewareFunc{
    handler1,
    handler2,
}

// Both can coexist!
```

### 4. Matches Real-World Usage

**How users actually collect metrics:**

```go
// Prometheus
opts.OnResponse = func(resp *http.Response, duration time.Duration, err error) {
    httpDuration.Observe(duration.Seconds())
    httpStatus.WithLabelValues(strconv.Itoa(resp.StatusCode)).Inc()
}

// DataDog
opts.OnResponse = func(resp *http.Response, duration time.Duration, err error) {
    statsd.Timing("http.request.duration", duration)
    statsd.Incr("http.request.count", []string{
        "status:" + strconv.Itoa(resp.StatusCode),
    }, 1)
}

// Simple logging
opts.OnResponse = func(resp *http.Response, duration time.Duration, err error) {
    log.Printf("[%s] %d - %v", opts.URL, resp.StatusCode, duration)
}
```

**They just need a callback** - not complex middleware chains!

---

## Implementation Plan

### Step 1: Add OnResponse to RequestOptions

```go
// options/options.go
type RequestOptions struct {
    // ... existing 30+ fields ...

    // OnResponse is called after request completion (success or failure).
    // Receives the response, total duration, and any error.
    // Useful for metrics collection, logging, and monitoring.
    // Optional - only called if non-nil.
    OnResponse func(resp *http.Response, duration time.Duration, err error)
}
```

### Step 2: Call in Process()

```go
// process.go
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
    startTime := time.Now()

    // ... existing validation, client creation, request creation ...

    // Execute request with retries
    resp, err := executeWithRetries(httpClient, req, opts)

    // Notify callback (even on error)
    if opts.OnResponse != nil {
        opts.OnResponse(resp, time.Since(startTime), err)
    }

    if err != nil {
        return nil, "", err
    }

    // ... continue with response processing ...

    return resp, bodyString, nil
}
```

### Step 3: Document with Examples

```go
// Example 1: Simple duration logging
opts := &options.RequestOptions{
    URL: "https://api.example.com",
    OnResponse: func(resp *http.Response, duration time.Duration, err error) {
        log.Printf("Request took %v", duration)
    },
}

// Example 2: Custom metrics struct
type Metrics struct {
    Duration   time.Duration
    StatusCode int
    Success    bool
}

metrics := &Metrics{}
opts := &options.RequestOptions{
    URL: "https://api.example.com",
    OnResponse: func(resp *http.Response, duration time.Duration, err error) {
        metrics.Duration = duration
        if resp != nil {
            metrics.StatusCode = resp.StatusCode
        }
        metrics.Success = err == nil && resp.StatusCode < 400
    },
}

// Example 3: Prometheus integration
opts := &options.RequestOptions{
    URL: "https://api.example.com",
    OnResponse: func(resp *http.Response, duration time.Duration, err error) {
        httpRequestDuration.WithLabelValues(
            opts.Method,
            strconv.Itoa(resp.StatusCode),
        ).Observe(duration.Seconds())
    },
}
```

---

## Final Answer: What About Middleware?

**Current middleware CAN'T handle metrics because:**

1. ❌ Middleware runs **BEFORE** request is sent
2. ❌ Returns **BEFORE** response is received
3. ❌ No access to response status, body, or timing
4. ❌ Can't measure duration (no end time)

**Two solutions:**

### Option A: OnResponse Callback (Recommended for Beta)

✅ Simple (1 field, 5 lines)
✅ Zero-allocation when nil
✅ Solves 90% of use cases
✅ Can add later: Response Middleware if needed

### Option B: Response Middleware (Consider for v1.0+)

✅ Composable (chain handlers)
✅ Symmetrical architecture
❌ More complex (2 middleware types)
❌ Overhead for rare use case

**Recommendation**: Start with OnResponse callback, add Response Middleware only if users request it.

---

**Decision**: Remove `Metrics` field, add `OnResponse` callback for beta release.

**Rationale**: Middleware architecture doesn't support response observation, callback is simplest solution.
