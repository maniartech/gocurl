# Context and Timeout Handling Analysis

## Industry Best Practices (Go Standard)

### ✅ **The Golden Rule: Context Takes Precedence**

According to Go documentation and industry standards:

> **"At Google, we require that Go programmers pass a Context parameter as the first argument to every function on the call path between incoming and outgoing requests."**

**Core Principles:**
1. **Context is king**: Always respect context cancellation/deadlines
2. **Single source of truth**: Don't mix `client.Timeout` with context deadlines
3. **Cleanup is mandatory**: Always `defer cancel()` to prevent leaks
4. **Context propagation**: Pass context down the call chain

### Industry Examples

**Go stdlib (net/http):**
```go
client := &http.Client{} // No Timeout set
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
resp, err := client.Do(req) // Uses ONLY context deadline
```

**Kubernetes API Client:**
```go
client := &http.Client{
    Timeout: 0, // Always 0 - context controls timeout
}
```

**AWS SDK for Go:**
```go
if deadline, ok := ctx.Deadline(); ok {
    transport.ResponseHeaderTimeout = time.Until(deadline)
} else {
    transport.ResponseHeaderTimeout = c.cfg.HTTPTimeout
}
```

## Issues Identified

### 🔴 **CRITICAL ISSUE 1: Conflicting Timeout Mechanisms** ⚠️ VIOLATES GO STANDARDS

Currently, there are **TWO SEPARATE** timeout mechanisms that can conflict:

1. **`client.Timeout`** (set in `CreateHTTPClient` from `opts.Timeout`)
2. **Context deadline** (passed via `ctx` parameter)

**This violates industry best practice of having a single timeout source.**

#### Current Code (process.go:158)
```go
client := &http.Client{
    Transport: transport,
    Timeout:   opts.Timeout,  // ⚠️ Client-level timeout
    // ...
}
```

#### The Problem
When a user does this:
```go
// User sets context timeout to 5 seconds
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// But options also has Timeout set to 10 seconds
opts := options.NewRequestOptionsBuilder().
    SetTimeout(10 * time.Second).
    Build()

resp, err := gocurl.Execute(opts)
```

**What happens?**
- The request will use BOTH timeouts
- The HTTP client has a 10-second timeout
- The context has a 5-second timeout
- Whichever fires first will cancel the request
- This creates **unpredictable behavior**

### 🔴 **CRITICAL ISSUE 2: Context from Options is Ignored in Process()**

#### Current Code Flow
```go
// api.go - RequestWithContext
func RequestWithContext(ctx context.Context, ...) {
    // ...
    opts.Context = ctx  // ✅ Context is stored in options
    return Execute(opts)
}

// api.go - Execute
func Execute(opts *options.RequestOptions) {
    ctx := opts.Context
    if ctx == nil {
        ctx = context.Background()
    }

    httpResp, _, err := Process(ctx, opts)  // ✅ Passes context
}

// process.go - Process
func Process(ctx context.Context, opts *options.RequestOptions) {
    // ...
    req, err := CreateRequest(ctx, opts)  // ✅ Uses ctx
}

// process.go - CreateRequest
func CreateRequest(ctx context.Context, opts *options.RequestOptions) {
    req, err := http.NewRequestWithContext(ctx, method, url, body)  // ✅ Context attached
}

// process.go - CreateHTTPClient
func CreateHTTPClient(opts *options.RequestOptions) {
    client := &http.Client{
        Timeout: opts.Timeout,  // ⚠️ Ignores opts.Context!
    }
}
```

**The Problem:**
- If `opts.Context` has a deadline, it's used in the request
- But `client.Timeout` is ALSO set from `opts.Timeout`
- These can conflict with each other

### 🟡 **ISSUE 3: WithTimeout Builder Creates Context but Might Conflict**

#### Current Code (builder.go)
```go
func (b *Builder) WithTimeout(timeout time.Duration) *Builder {
    if b.opts.Context == nil {
        b.opts.Context = context.Background()
    }
    ctx, cancel := context.WithTimeout(b.opts.Context, timeout)
    b.opts.Context = ctx
    b.opts.ContextCancel = cancel  // ⚠️ This field doesn't exist!
    return b
}
```

**Problems:**
1. References `b.opts.ContextCancel` which doesn't exist in `RequestOptions`
2. Creates a context with timeout, but code won't compile
3. The cancel function is lost (memory leak potential)

### 🟡 **ISSUE 4: No Context Cleanup**

When contexts with timeouts/deadlines are created, the cancel functions should be called to release resources. Currently:
- No mechanism to store cancel functions
- No deferred calls to cancel()
- Potential goroutine leaks

## Correct Behavior (Industry Standard)

### Go HTTP Client Timeout Behavior

From Go documentation:

> **`Client.Timeout`** includes the entire time from connection to reading the body.
>
> **Context deadline** is also respected and will cancel the request if exceeded.

### Industry Best Practice (The Standard Pattern)

**✅ DO: Context Priority Pattern**
```go
func CreateHTTPClient(ctx context.Context, opts *RequestOptions) (*http.Client, error) {
    var timeout time.Duration

    // Context deadline takes absolute priority
    if ctx != nil {
        if _, hasDeadline := ctx.Deadline(); hasDeadline {
            timeout = 0  // Let context handle it
        } else {
            timeout = opts.Timeout  // Fallback to options
        }
    } else {
        timeout = opts.Timeout
    }

    client := &http.Client{Timeout: timeout}
    return client, nil
}
```

**❌ DON'T: Mix Both**
```go
// BAD - creates race condition
client := &http.Client{Timeout: 10*time.Second}
req, _ := http.NewRequestWithContext(ctxWith5sDeadline, ...)
client.Do(req) // Which wins? Unpredictable!
```

**Timeout Priority (Industry Standard):**
1. **Context deadline** (if set) - HIGHEST PRIORITY
2. **opts.Timeout** (if context has no deadline) - FALLBACK
3. **No timeout** (if both are zero/nil) - NO LIMIT

## Recommended Solution (Industry Standard)

### ✅ Solution: Context Priority Pattern (INDUSTRY STANDARD)

```go
// Change signature
func CreateHTTPClient(ctx context.Context, opts *options.RequestOptions) (*http.Client, error) {
    var clientTimeout time.Duration

    // Priority: Context deadline > opts.Timeout
    if ctx != nil {
        if deadline, ok := ctx.Deadline(); ok {
            // Context has deadline, don't set client timeout
            // The context will handle cancellation
            clientTimeout = 0
        } else if opts.Timeout > 0 {
            // No context deadline, use opts.Timeout
            clientTimeout = opts.Timeout
        }
    } else if opts.Timeout > 0 {
        clientTimeout = opts.Timeout
    }

    client := &http.Client{
        Transport: transport,
        Timeout:   clientTimeout,
        // ...
    }

    return client, nil
}

// Update Process to pass context
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
    // ...
    client, err := CreateHTTPClient(ctx, opts)  // Pass context
    // ...
}
```

### Solution 3: Fix ContextCancel Field

**Add the missing field to RequestOptions:**
```go
type RequestOptions struct {
    // ... existing fields ...
    Context       context.Context    `json:"-"`
    ContextCancel context.CancelFunc `json:"-"` // ADD THIS
    // ... rest of fields ...
}
```

**Update Builder.WithTimeout (INDUSTRY STANDARD PATTERN):**
```go
func (b *Builder) WithTimeout(timeout time.Duration) *Builder {
    ctx := b.options.Context
    if ctx == nil {
        ctx = context.Background()
    }

    // Create timeout context
    timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
    b.options.Context = timeoutCtx
    b.options.ContextCancel = cancel  // Store for cleanup

    return b
}
```

**Cleanup in Execute (MANDATORY - prevents leaks):**
```go
func Execute(opts *options.RequestOptions) (*Response, error) {
    // CRITICAL: Always cleanup context
    if opts.ContextCancel != nil {
        defer opts.ContextCancel()  // ✅ Prevents goroutine leaks
    }
    // ... rest of execution ...
}
```

### Solution 4: Add RequestOptions.ContextCancel Field

```go
// In options.go
type RequestOptions struct {
    // ... existing fields ...

    Context       context.Context    `json:"-"`
    ContextCancel context.CancelFunc `json:"-"` // Function to cancel the context

    // ... rest of fields ...
}
```

Then in Execute():
```go
func Execute(opts *options.RequestOptions) (*Response, error) {
    // Cleanup context if we created it
    if opts.ContextCancel != nil {
        defer opts.ContextCancel()
    }

    ctx := opts.Context
    if ctx == nil {
        ctx = context.Background()
    }

    // ...
}
```

## Proposed Changes

### Change 1: Fix CreateHTTPClient to Accept Context

```go
func CreateHTTPClient(ctx context.Context, opts *options.RequestOptions) (*http.Client, error) {
    // Load TLS configuration
    tlsConfig, err := LoadTLSConfig(opts)
    if err != nil {
        return nil, fmt.Errorf("failed to load TLS config: %w", err)
    }

    transport := &http.Transport{
        TLSClientConfig: tlsConfig,
        Proxy:           http.ProxyFromEnvironment,
    }

    // ... proxy and HTTP/2 configuration ...

    // Determine timeout based on context deadline vs opts.Timeout
    var clientTimeout time.Duration

    if ctx != nil {
        // If context has a deadline, prefer it over opts.Timeout
        if deadline, ok := ctx.Deadline(); ok {
            // Don't set client.Timeout, let context handle it
            clientTimeout = 0
        } else {
            // No deadline in context, use opts.Timeout
            clientTimeout = opts.Timeout
        }
    } else {
        // No context provided, use opts.Timeout
        clientTimeout = opts.Timeout
    }

    client := &http.Client{
        Transport: transport,
        Timeout:   clientTimeout,
        // ... redirect handling ...
    }

    return client, nil
}
```

### Change 2: Update Process to Pass Context

```go
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
    // Validate options
    if err := ValidateOptions(opts); err != nil {
        return nil, "", err
    }

    // Create HTTP client - now with context awareness
    client, err := CreateHTTPClient(ctx, opts)
    if err != nil {
        return nil, "", err
    }

    // ... rest remains the same ...
}
```

### Change 3: Add ContextCancel to RequestOptions

```go
// In options/options.go
type RequestOptions struct {
    // ... existing fields ...

    // Advanced options
    Context           context.Context       `json:"-"` // Not exported to JSON
    ContextCancel     context.CancelFunc    `json:"-"` // Cancel function for context

    // ... rest of fields ...
}
```

### Change 4: Fix Builder.WithTimeout

```go
// In options/builder.go
func (b *RequestOptionsBuilder) WithTimeout(timeout time.Duration) *RequestOptionsBuilder {
    // Create context with timeout
    ctx := b.options.Context
    if ctx == nil {
        ctx = context.Background()
    }

    timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
    b.options.Context = timeoutCtx
    b.options.ContextCancel = cancel

    return b
}
```

### Change 5: Cleanup in Execute

```go
// In api.go
func Execute(opts *options.RequestOptions) (*Response, error) {
    // Cleanup context if we created it
    if opts.ContextCancel != nil {
        defer opts.ContextCancel()
    }

    // Use context from options or create one
    ctx := opts.Context
    if ctx == nil {
        ctx = context.Background()
    }

    // Use the existing Process function
    httpResp, _, err := Process(ctx, opts)
    if err != nil {
        return nil, err
    }

    return &Response{
        Response: httpResp,
    }, nil
}
```

## Testing Requirements

After fixes, ensure:

1. ✅ Context timeout alone works
2. ✅ opts.Timeout alone works
3. ✅ Context timeout takes precedence when both are set
4. ✅ No goroutine leaks (use `go test -race`)
5. ✅ Cancel functions are properly called
6. ✅ Builder.WithTimeout doesn't leak contexts

## Summary

### Current Problems (Violates Industry Standards)
1. ❌ **`client.Timeout` and context deadline can conflict** - VIOLATES "single source" principle
2. ❌ **`CreateHTTPClient` doesn't consider context deadline** - VIOLATES "context priority" pattern
3. ❌ **`ContextCancel` field doesn't exist but is referenced** - COMPILATION ERROR
4. ❌ **No cleanup of context cancel functions** - MEMORY LEAKS (violates "cleanup is mandatory")
5. ❌ **Unpredictable behavior when both timeouts are set** - VIOLATES "context is king" principle

### Required Fixes (Align with Industry Standards)
1. ✅ **Pass context to `CreateHTTPClient`** - Required for context-aware timeout logic
2. ✅ **Implement Context Priority Pattern** - Set `client.Timeout = 0` when context has deadline
3. ✅ **Add `ContextCancel` field to `RequestOptions`** - Enables proper cleanup
4. ✅ **Fix `Builder.WithTimeout` to store cancel function** - Prevents leaks
5. ✅ **Defer cancel in `Execute()`** - Mandatory cleanup pattern

### Priority
**CRITICAL** - This affects:
- ❌ Correctness (unpredictable timeout behavior)
- ❌ Reliability (memory leaks from uncancelled contexts)
- ❌ Compilation (ContextCancel field missing)
- ❌ Standards compliance (violates Go best practices)

### Documentation Requirements
After fixes, update docs to specify:
```
Timeout Behavior (Industry Standard Pattern):
  1. Context deadline (if set) - Takes absolute priority
  2. opts.Timeout (if no context deadline) - Fallback
  3. No timeout (if both zero) - Unlimited

Always call defer cancel() when creating timeout contexts.
```
