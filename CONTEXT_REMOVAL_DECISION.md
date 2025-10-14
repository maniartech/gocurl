# Context Decision: Remove opts.Context, Keep Parameter

**Date**: October 14, 2025
**Decision**: REMOVE `Context` field from `RequestOptions`
**Rationale**: Follow Go conventions, avoid confusion, prevent context storage

---

## The Go Philosophy on Context

### Official Go Guidelines

From [Go Blog: Context](https://go.dev/blog/context):

> **"Do not store Contexts inside a struct type"**
>
> "Instead, pass a Context explicitly to each function that needs it. The Context should be the first parameter, typically named ctx."

### Why Go Prohibits Context in Structs

1. **Lifetime confusion** - Context tied to request, not config
2. **Cancellation issues** - Stored context might be cancelled
3. **Memory leaks** - Long-lived struct holds references
4. **Anti-pattern** - Violates explicit context passing

### Go Standard Library Examples

```go
// ✅ CORRECT - Context as parameter
http.NewRequestWithContext(ctx context.Context, method, url string, body io.Reader)

// ❌ WRONG - Context in struct
type Request struct {
    Context context.Context // Go team would reject this!
}
```

---

## Current GoCurl Violates This

### What We Have (WRONG)

```go
// options/options.go - VIOLATES Go conventions!
type RequestOptions struct {
    // ... 30+ fields ...
    Context context.Context `json:"-"` // ❌ Storing context in struct
}

// process.go
func Process(ctx context.Context, opts *options.RequestOptions) {
    // Which context to use? Confusing!
}
```

### Why This is Problematic

**Problem 1: Confusion**
```go
// User writes this - which context wins?
ctx1, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

opts := &options.RequestOptions{
    URL:     "https://api.com",
    Context: ctx2, // From distributed tracing
}

Process(ctx1, opts) // ctx1 or ctx2? 🤔
```

**Problem 2: Long-lived RequestOptions**
```go
// DANGEROUS - Context stored in long-lived struct
var defaultOpts = &options.RequestOptions{
    Context: context.Background(), // ❌ Stored for program lifetime!
    Timeout: 10 * time.Second,
    Headers: commonHeaders,
}

// Later...
func MakeRequest(url string) {
    defaultOpts.URL = url
    Process(???, defaultOpts) // Old context still there!
}
```

**Problem 3: Cancellation Issues**
```go
// Context cancelled but stored in opts
ctx, cancel := context.WithCancel(context.Background())
opts := &options.RequestOptions{
    URL:     "https://api.com",
    Context: ctx,
}
cancel() // Context cancelled!

// Later... (context already cancelled!)
Process(ctx, opts) // Will fail immediately!
```

---

## Decision: Follow Go Conventions

### ✅ REMOVE Context from RequestOptions

**Align with Go best practices:**
- Context is always a function parameter
- Never stored in structs
- Explicit and clear

### Implementation

#### 1. Remove from options.go

```go
// options/options.go
type RequestOptions struct {
    // HTTP request basics
    Method      string      `json:"method"`
    URL         string      `json:"url"`
    Headers     http.Header `json:"headers,omitempty"`
    Body        string      `json:"body"`
    // ... other fields ...

    // Advanced options
    // Context           context.Context              `json:"-"` // ❌ REMOVE THIS
    ContextCancel     context.CancelFunc           `json:"-"` // Keep for cleanup
    RequestID         string                       `json:"request_id,omitempty"`
    Middleware        []middlewares.MiddlewareFunc `json:"-"`
    // ... rest of fields ...
}
```

#### 2. Remove from Clone()

```go
// Clone() method - remove Context handling
func (ro *RequestOptions) Clone() *RequestOptions {
    clone := *ro
    clone.Headers = ro.Headers.Clone()
    // ... other cloning ...

    // Remove this:
    // if ro.Context != nil {
    //     clone.Context = ro.Context
    // }

    return &clone
}
```

#### 3. Keep Process() signature

```go
// process.go - ALREADY CORRECT
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
    // Always use ctx parameter
    // No confusion!
}
```

---

## How to Handle Distributed Tracing

### Question: If we remove opts.Context, how does distributed tracing work?

**Answer: Same way the Go standard library does it!**

### Go Standard Library Pattern

```go
// http package
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
//                                   ^^^ Context is parameter, not in Request struct

client.Do(req)
```

### GoCurl Pattern (After Removing opts.Context)

```go
// User code with distributed tracing
func getUserProfile(ctx context.Context, userID string) (*User, error) {
    // ctx contains trace ID from OpenTelemetry
    ctx, span := tracer.Start(ctx, "get-user")
    defer span.End()

    opts := &options.RequestOptions{
        URL: "https://user-api/users/" + userID,
        // No Context field - pass as parameter!
    }

    resp, body, err := gocurl.Process(ctx, opts)
    //                                ^^^ Context passed here
    return parseUser(body), err
}
```

### How GoCurl Handles It

```go
// process.go
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
    // Create request with context
    req, err := http.NewRequestWithContext(ctx, opts.Method, opts.URL, body)
    //                                     ^^^ OpenTelemetry auto-injects trace headers

    // Execute request
    resp, err := client.Do(req)

    return resp, body, err
}

// That's it! OpenTelemetry handles the rest automatically
```

---

## OpenTelemetry Doesn't Need opts.Context

### How OpenTelemetry Works

**Context carries trace metadata:**

```go
// 1. User starts trace
ctx, span := tracer.Start(context.Background(), "api-call")
// ctx now contains: trace ID, span ID, baggage

// 2. Pass ctx to Process
gocurl.Process(ctx, opts)

// 3. Inside Process, create request with context
req, _ := http.NewRequestWithContext(ctx, ...)
//                                   ^^^ OpenTelemetry intercepts this!

// 4. OpenTelemetry automatically injects headers:
// Traceparent: 00-trace-id-span-id-01

// 5. Remote service receives headers, extracts trace ID
// No opts.Context needed!
```

### The Magic

**OpenTelemetry's HTTP instrumentation hooks into `http.NewRequestWithContext`:**

```go
// When you do this:
req, _ := http.NewRequestWithContext(ctx, method, url, body)

// OpenTelemetry (if configured) automatically:
// 1. Extracts trace ID from ctx
// 2. Injects Traceparent header
// 3. Injects Tracestate header
// 4. Propagates baggage

// You don't need to do ANYTHING!
```

---

## Updated Examples

### Before (With opts.Context - CONFUSING)

```go
// ❌ OLD WAY - Confusing!
ctx, span := tracer.Start(context.Background(), "api-call")
defer span.End()

opts := &options.RequestOptions{
    URL:     "https://api.com",
    Context: ctx, // Storing context in struct - BAD!
}

Process(ctx, opts) // Redundant! ctx passed twice
```

### After (ctx Parameter Only - CLEAR)

```go
// ✅ NEW WAY - Clear and idiomatic!
ctx, span := tracer.Start(context.Background(), "api-call")
defer span.End()

opts := &options.RequestOptions{
    URL: "https://api.com",
    // No Context field!
}

Process(ctx, opts) // Context passed once, as parameter
```

### Complete Example: Distributed Tracing

```go
import (
    "context"
    "go.opentelemetry.io/otel"
    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/options"
)

func handleCheckout(ctx context.Context, order Order) error {
    tracer := otel.Tracer("my-service")

    // Start trace
    ctx, span := tracer.Start(ctx, "checkout")
    defer span.End()

    // Call user service
    user, err := getUserProfile(ctx, order.UserID)
    if err != nil {
        return err
    }

    // Call payment service
    err = processPayment(ctx, order.Payment)
    if err != nil {
        return err
    }

    return nil
}

func getUserProfile(ctx context.Context, userID string) (*User, error) {
    ctx, span := tracer.Start(ctx, "get-user")
    defer span.End()

    opts := &options.RequestOptions{
        URL: "https://user-api/users/" + userID,
        // No Context field - just pass ctx parameter!
    }

    resp, body, err := gocurl.Process(ctx, opts)
    // OpenTelemetry automatically:
    // 1. Extracts trace ID from ctx
    // 2. Injects Traceparent header into HTTP request
    // 3. User API receives request with trace headers
    // 4. Continues same trace!

    return parseUser(body), err
}

func processPayment(ctx context.Context, payment Payment) error {
    ctx, span := tracer.Start(ctx, "process-payment")
    defer span.End()

    opts := &options.RequestOptions{
        URL:    "https://payment-api/charge",
        Method: "POST",
        Body:   toJSON(payment),
        // No Context field!
    }

    _, _, err := gocurl.Process(ctx, opts)
    // Same trace ID propagated automatically!

    return err
}
```

**Result in Jaeger:**
```
Trace: checkout
├─ checkout (500ms)
│  ├─ get-user (100ms)
│  │  └─ HTTP GET /users/123 ← GoCurl request
│  └─ process-payment (350ms)
│     └─ HTTP POST /charge ← GoCurl request

All in same trace, no opts.Context needed!
```

---

## Benefits of Removing opts.Context

### ✅ Follows Go Best Practices

- Context always as function parameter ✅
- Never stored in structs ✅
- Explicit, not implicit ✅

### ✅ Eliminates Confusion

```go
// Before: Which context?
Process(ctx1, &options.RequestOptions{Context: ctx2})

// After: Clear!
Process(ctx, &options.RequestOptions{...})
```

### ✅ Prevents Bugs

```go
// Before: Long-lived context (DANGEROUS)
var defaultOpts = &options.RequestOptions{
    Context: context.Background(), // Lives forever!
}

// After: Fresh context every request (SAFE)
var defaultOpts = &options.RequestOptions{
    // No Context field
}
Process(ctx, defaultOpts) // ctx is request-scoped
```

### ✅ Simpler API

- One way to pass context, not two
- Less fields in RequestOptions
- Easier to understand

### ✅ Same Distributed Tracing Capability

- OpenTelemetry works exactly the same
- Context parameter is all you need
- No functionality lost

---

## Migration Path

### Breaking Change

Yes, this is a **breaking change**:

```go
// Users currently doing this:
opts := &options.RequestOptions{
    URL:     url,
    Context: ctx, // ❌ Won't compile after change
}

// Must change to:
opts := &options.RequestOptions{
    URL: url,
}
Process(ctx, opts) // ✅ Pass ctx as parameter
```

### But It's Worth It

**Why accept breaking change:**
1. Library is in beta (pre-v1.0)
2. Aligns with Go best practices
3. Prevents future bugs
4. Simpler, clearer API
5. Better for long-term maintenance

### Migration Guide for Users

```go
// OLD CODE (won't work):
opts := &options.RequestOptions{
    URL:     "https://api.com",
    Context: ctx,
}
Process(nil, opts)

// NEW CODE (simple fix):
opts := &options.RequestOptions{
    URL: "https://api.com",
}
Process(ctx, opts)

// Or if you were doing both (redundant):
Process(ctx1, &options.RequestOptions{Context: ctx2})

// Now just:
Process(ctx, &options.RequestOptions{...})
```

---

## Final Recommendation

### ✅ REMOVE Context from RequestOptions

**Changes needed:**

1. **Delete from options.go:**
   - Remove `Context context.Context` field (line 78)
   - Remove Context handling from `Clone()` method

2. **Update process.go:**
   - Already correct! Just use `ctx` parameter everywhere

3. **Update documentation:**
   - Remove examples showing `opts.Context = ctx`
   - Show only `Process(ctx, opts)` pattern
   - Emphasize alignment with Go best practices

4. **Update all .md files:**
   - DISTRIBUTED_TRACING_EXPLAINED.md
   - FINAL_DECISION.md
   - ENTERPRISE_OBSERVABILITY.md
   - Remove references to opts.Context

### Timeline

**For Beta (Oct 18):**
- ✅ Remove Context field
- ✅ Update documentation
- ✅ Add migration guide

**Why Now:**
- Pre-v1.0 (breaking changes acceptable)
- Aligns with Go conventions
- Prevents technical debt

---

## Summary

**Question:** Should we keep both ctx parameter and opts.Context?

**Answer:** NO - Remove opts.Context

**Reasoning:**
1. ❌ Go explicitly forbids storing Context in structs
2. ❌ Causes confusion (two ways to pass context)
3. ❌ Enables anti-patterns (long-lived context)
4. ✅ OpenTelemetry works perfectly with ctx parameter
5. ✅ Simpler, clearer, more idiomatic

**Decision:** Remove `Context` field from `RequestOptions`, keep only `ctx` parameter in `Process()`

**Impact:** Breaking change, but worth it for long-term API quality

**Action:** Remove before v1.0 while still in beta

---

**The Go way is the right way: Context as parameter, always.**
