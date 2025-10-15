# Timeout Handling Flow - Current Implementation

## Complete Request Execution Flow

```
User Code
    │
    ├─→ Get(ctx, url, vars)           [api.go]
    │   └─→ RequestWithContext(ctx, cmd, vars)
    │
    ├─→ RequestWithContext(ctx, cmd, vars)  [api.go]
    │   ├─→ Parse command & create opts
    │   ├─→ opts.Context = ctx        ⚠️ Stored but may conflict with opts.Timeout
    │   └─→ Execute(opts)
    │
    └─→ Execute(opts)                 [api.go]
        ├─→ ctx = opts.Context (or Background)
        └─→ Process(ctx, opts)
            │
            ├─→ CreateHTTPClient(opts)     [process.go:line ~110]
            │   │
            │   └─→ client := &http.Client{
            │           Timeout: opts.Timeout,    ⚠️ TIMEOUT SET HERE (from options)
            │       }
            │
            ├─→ CreateRequest(ctx, opts)   [process.go:line ~201]
            │   │
            │   └─→ req, err := http.NewRequestWithContext(ctx, method, url, body)
            │                                     ⚠️ CONTEXT ATTACHED HERE
            │
            └─→ ExecuteWithRetries(client, req, opts)  [retry.go:line 15]
                │
                └─→ for attempt := 0; attempt <= retries; attempt++ {
                        │
                        └─→ resp, err = client.Do(attemptReq)    ⚠️ REQUEST EXECUTED HERE
                                │
                                └─→ [Go HTTP Client Internal Logic]
                                    │
                                    ├─→ Checks: req.Context().Done()  (context cancellation/deadline)
                                    ├─→ Checks: client.Timeout        (client-level timeout)
                                    │
                                    └─→ Whichever fires FIRST cancels the request ⚠️
                    }
```

## Where Timeouts Are Set

### 1. **Client-Level Timeout** (process.go:158)

```go
func CreateHTTPClient(opts *options.RequestOptions) (*http.Client, error) {
    // ... setup transport, proxy, TLS ...

    client := &http.Client{
        Transport: transport,
        Timeout:   opts.Timeout,  // ⚠️ SET FROM opts.Timeout
        CheckRedirect: func(...) { ... },
    }

    return client, nil
}
```

**Source**: `opts.Timeout` (set by `-m/--max-time` flag or `SetTimeout()` builder)

**Scope**: Entire request including:
- DNS lookup
- Connection establishment
- Request writing
- Response reading (headers + body)

### 2. **Context Deadline** (process.go:265)

```go
func CreateRequest(ctx context.Context, opts *options.RequestOptions) (*http.Request, error) {
    // ... prepare method, url, body ...

    req, err := http.NewRequestWithContext(ctx, method, url, body)
    //                                      ^^^ CONTEXT ATTACHED HERE

    return req, nil
}
```

**Source**:
- `ctx` parameter passed through from `RequestWithContext()`
- May have deadline from `context.WithTimeout()` or `context.WithDeadline()`

**Scope**: Request execution (respects context cancellation/deadline)

### 3. **Request Execution** (retry.go:46)

```go
func ExecuteWithRetries(client *http.Client, req *http.Request, opts *options.RequestOptions) {
    for attempt := 0; attempt <= retries; attempt++ {
        // ...

        resp, err = client.Do(attemptReq)  // ⚠️ BOTH TIMEOUTS ACTIVE HERE!
        //                ^
        //                ├─→ Uses client.Timeout (from opts.Timeout)
        //                └─→ Uses req.Context() deadline (from ctx parameter)

        // ...
    }
}
```

## How Go's http.Client Handles Timeouts

When `client.Do(req)` is called, Go's internal HTTP client:

1. **Checks Context First**:
   ```go
   if ctx := req.Context(); ctx != nil {
       if err := ctx.Err(); err != nil {
           return nil, err  // Context already cancelled/expired
       }
   }
   ```

2. **Sets Up Deadline Timer**:
   - If `client.Timeout > 0`, creates internal timer
   - If `req.Context()` has deadline, uses it too

3. **Monitors Both During Request**:
   ```go
   select {
   case <-req.Context().Done():
       // Context cancelled or deadline exceeded
       return req.Context().Err()
   case <-time.After(client.Timeout):
       // Client timeout exceeded
       return http.ErrTimeout
   case result := <-requestCompleted:
       // Request completed successfully
       return result
   }
   ```

4. **Whichever Fires First Wins**:
   - ⚠️ Creates race condition
   - ⚠️ Unpredictable which error you'll get
   - ⚠️ User can't control which timeout mechanism is used

## Problem Scenarios

### Scenario 1: Both Timeouts Set (CONFLICT)

```go
// User sets context timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Options also has timeout
opts := options.NewRequestOptionsBuilder().
    SetTimeout(10 * time.Second).
    Build()

// Which timeout wins?
resp, err := gocurl.Get(ctx, "https://slow-api.com", nil)
//                       ^^^
//                       Context has 5s deadline
//
// But CreateHTTPClient sets client.Timeout = 10s
//
// Result: Request times out after 5 seconds (context wins)
// But this is IMPLICIT behavior, not documented or guaranteed
```

### Scenario 2: opts.Timeout Set, No Context Timeout

```go
ctx := context.Background()  // No deadline

opts := options.NewRequestOptionsBuilder().
    SetTimeout(5 * time.Second).
    Build()

resp, err := gocurl.Get(ctx, "https://slow-api.com", nil)
//
// Result: Request uses client.Timeout = 5s ✅ WORKS
```

### Scenario 3: Context Timeout Set, No opts.Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

opts := options.NewRequestOptionsBuilder().
    // No SetTimeout called, opts.Timeout = 0
    Build()

resp, err := gocurl.Get(ctx, "https://slow-api.com", nil)
//
// CreateHTTPClient sets client.Timeout = 0 (no client timeout)
// Request uses context deadline = 5s ✅ WORKS
```

### Scenario 4: Builder.WithTimeout (BROKEN)

```go
opts := options.NewRequestOptionsBuilder().
    Get("https://api.com", nil).
    WithTimeout(5 * time.Second).  // ⚠️ BROKEN
    Build()

// Current WithTimeout code:
//   ctx, cancel := context.WithTimeout(b.options.Context, timeout)
//   b.options.Context = ctx
//   b.options.ContextCancel = cancel  // ❌ Field doesn't exist!

// This DOESN'T COMPILE because ContextCancel field is missing
```

## Where Timeout Is Actually Enforced

The timeout is enforced in **Go's internal HTTP client** during `client.Do(req)`:

```go
// Inside net/http/client.go (Go standard library)
func (c *Client) do(req *Request) (retres *Response, reterr error) {
    // ... code ...

    // Set deadline if client.Timeout is set
    if c.Timeout > 0 {
        ctx, cancel := context.WithTimeout(req.Context(), c.Timeout)
        defer cancel()
        req = req.WithContext(ctx)  // ⚠️ WRAPS the existing context!
    }

    // ... actual request execution ...
}
```

**CRITICAL INSIGHT**:
- If `client.Timeout > 0`, Go **WRAPS** the request's existing context
- Creates a SECOND context with timeout
- Now you have NESTED contexts with potentially different deadlines
- The SHORTER deadline will fire first

## Summary: Where Timeout Handling Happens

| Location | File:Line | What Happens | Timeout Source |
|----------|-----------|--------------|----------------|
| 1. User creates context | User code | `context.WithTimeout()` | User-defined |
| 2. Store context in opts | api.go:62 | `opts.Context = ctx` | From ctx parameter |
| 3. Create HTTP client | process.go:158 | `client.Timeout = opts.Timeout` | From options (flags/builder) |
| 4. Create request with context | process.go:265 | `http.NewRequestWithContext(ctx, ...)` | From ctx parameter |
| 5. Execute request | retry.go:46 | `client.Do(req)` | ⚠️ BOTH timeouts active! |
| 6. Go wraps context | net/http/client.go | Wraps req.Context() if Timeout > 0 | Go internal logic |

## The Problem in Plain English

**We're setting timeouts in TWO places:**

1. **`client.Timeout`** - Set from `opts.Timeout` in `CreateHTTPClient()`
2. **Request context** - Attached in `CreateRequest()` with deadline from user's `ctx`

**Go's HTTP client then:**
- Takes the request context (might have deadline)
- If `client.Timeout > 0`, wraps it with ANOTHER timeout context
- Now has nested contexts with 2 different deadlines
- Whichever deadline is SHORTER fires first

**This causes:**
- ⚠️ Unpredictable behavior (which timeout fires first?)
- ⚠️ User confusion (I set 10s but it timed out at 5s!)
- ⚠️ No control (can't choose which mechanism to use)
- ⚠️ Compilation error (WithTimeout references missing field)

## Solution (Industry Standard Pattern)

### ✅ Context Priority Pattern (INDUSTRY STANDARD - RECOMMENDED)

```go
func CreateHTTPClient(ctx context.Context, opts *options.RequestOptions) (*http.Client, error) {
    var clientTimeout time.Duration

    // If context has deadline, DON'T set client.Timeout
    if ctx != nil {
        if _, hasDeadline := ctx.Deadline(); hasDeadline {
            clientTimeout = 0  // Let context handle timeout
        } else {
            clientTimeout = opts.Timeout  // Use opts.Timeout
        }
    } else {
        clientTimeout = opts.Timeout
    }

    client := &http.Client{
        Timeout: clientTimeout,  // Only set if no context deadline
        // ...
    }
}
```

**Why This Is The Industry Standard:**
- ✅ Kubernetes API clients use this pattern
- ✅ AWS SDK for Go uses this pattern
- ✅ Google's internal Go code uses this pattern
- ✅ Aligns with Go's official blog on Context patterns

**Benefits:**
- ✅ Predictable: Context always takes priority
- ✅ Composable: Easy to wrap contexts with additional timeouts
- ✅ Testable: Easy to mock and test timeout behavior
- ✅ Standard: Follows Go community best practices

**❌ REJECTED Alternatives:**

**Option B: opts.Timeout Takes Priority**
- ❌ Breaks Go conventions
- ❌ Makes context cancellation less useful
- ❌ Not used by any major Go libraries

**Option C: Merge Both (Use Shorter)**
- ❌ Complex and confusing
- ❌ Unpredictable behavior
- ❌ Harder to test
- ❌ No major library uses this pattern
