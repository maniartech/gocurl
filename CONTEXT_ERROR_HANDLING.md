# Context Error Handling - Implementation Summary

## Overview

This document details the enhancements made to context error handling and cancellation monitoring throughout the request lifecycle, ensuring proper error propagation and early termination when context is cancelled or times out.

## Problem Statement

**Original Issue**: While the HTTP client respected context cancellation (via `http.NewRequestWithContext`), the retry logic wasn't explicitly checking for context errors, which could lead to:

1. Unnecessary retry attempts even after context cancellation
2. Sleeping during retry delays when context was already cancelled
3. Not checking context state before initiating retry attempts
4. Unclear error messages that didn't indicate context cancellation as the root cause

## Solution: Context-Aware Retry Logic

### Key Enhancements to `retry.go`

#### 1. Pre-Execution Context Check

```go
// Check if context is already cancelled/expired before starting
if req.Context() != nil {
    select {
    case <-req.Context().Done():
        return nil, fmt.Errorf("request context cancelled before execution: %w", req.Context().Err())
    default:
        // Context is still active, proceed
    }
}
```

**Purpose**: Prevent attempting requests with an already-cancelled context
**Benefit**: Immediate failure with clear error message, no wasted resources

#### 2. Pre-Retry Context Check

```go
// Check context before each retry attempt
if attempt > 0 && req.Context() != nil {
    select {
    case <-req.Context().Done():
        if resp != nil {
            resp.Body.Close()
        }
        return nil, fmt.Errorf("request context cancelled during retries (attempt %d/%d): %w",
            attempt, retries, req.Context().Err())
    default:
        // Context still active
    }
}
```

**Purpose**: Stop retry loop immediately when context is cancelled
**Benefit**: Prevents unnecessary retry attempts, faster failure response

#### 3. Context Error Detection

```go
if err != nil {
    // Unwrap and check for context errors
    if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
        // Context error - don't retry, return immediately
        return nil, fmt.Errorf("request failed due to context error (attempt %d/%d): %w",
            attempt, retries, err)
    }
    // Other error - will retry if attempts remain
}
```

**Purpose**: Detect context-related failures and stop retrying
**Benefit**: Distinguishes between retriable errors and cancellation

#### 4. Context-Aware Sleep

```go
// Sleep with context awareness - cancel sleep if context is cancelled
if req.Context() != nil {
    select {
    case <-req.Context().Done():
        return nil, fmt.Errorf("request context cancelled during retry delay: %w", req.Context().Err())
    case <-time.After(sleepDuration):
        // Sleep completed normally
    }
} else {
    // No context, just sleep
    time.Sleep(sleepDuration)
}
```

**Purpose**: Interrupt retry delays if context is cancelled
**Benefit**: Faster response to cancellation, no waiting for full delay

## Test Coverage

### New Test File: `context_error_test.go`

**8 Comprehensive Tests** validating context error handling:

#### 1. TestContextError_DeadlineExceeded ✅
- **Purpose**: Verify `context.DeadlineExceeded` error is properly reported
- **Scenario**: Slow server (2s), context timeout (100ms)
- **Verification**: Error is `context.DeadlineExceeded` or contains "deadline exceeded"

#### 2. TestContextError_Cancelled ✅
- **Purpose**: Verify `context.Canceled` error is properly reported
- **Scenario**: Manual cancellation during request
- **Verification**: Error is `context.Canceled`, request stops quickly

#### 3. TestContextError_WithRetries ✅ **[CRITICAL]**
- **Purpose**: Verify context errors stop retry loop
- **Scenario**: Server always fails (5 retries configured), context timeout at 800ms
- **Verification**:
  - Max 2 attempts before timeout (not all 5)
  - Times out around 800ms (not full retry duration)
  - Error indicates context deadline

#### 4. TestContextError_CancelDuringRetry ✅ **[CRITICAL]**
- **Purpose**: Verify context cancellation during retries stops execution
- **Scenario**: 10 retries configured, context cancelled after 400ms
- **Verification**:
  - Max 3 attempts (early stop)
  - Error indicates context cancellation
  - No race conditions (uses `sync/atomic`)

#### 5. TestContextError_PropagationThroughLayers ✅
- **Purpose**: Verify context errors propagate through all API layers
- **Scenario**: Call `Process()` directly with expired context
- **Verification**: Context error reaches top level

#### 6. TestContextError_MultipleRequests_Independent ✅
- **Purpose**: Verify contexts don't interfere between requests
- **Scenario**: First request times out, second succeeds
- **Verification**: Independent context handling per request

#### 7. TestContextError_CheckBeforeRetry ✅
- **Purpose**: Verify context state checked before starting
- **Scenario**: Already-expired context
- **Verification**: Zero attempts made, immediate failure

#### 8. TestContextError_HTTPClientRespect ✅
- **Purpose**: Verify HTTP client respects context during request/response
- **Scenario**: Slow body read with context timeout
- **Verification**: Quick timeout, no hanging

## Error Message Improvements

### Before Enhancement
```
request failed after 5 retries: Post "http://...": context deadline exceeded
```
- Unclear if all 5 retries actually happened
- Doesn't indicate which attempt hit the deadline

### After Enhancement
```
request failed due to context error (attempt 2/5): context deadline exceeded
```
or
```
request context cancelled during retries (attempt 3/5): context canceled
```
or
```
request context cancelled during retry delay: context canceled
```

**Benefits**:
- Clear indication of retry attempt when error occurred
- Explicit context error detection
- Different messages for different cancellation points

## Performance Improvements

### 1. Faster Failure Response

**Before**: Could wait through full retry delay even if context cancelled
**After**: Immediate termination on context cancellation

**Example**:
- Retry delay: 5 seconds
- Context cancelled after 1 second
- **Before**: Wait 4 more seconds before checking
- **After**: Terminate immediately ✅

### 2. Reduced Resource Usage

**Before**: Could attempt all retries even with cancelled context
**After**: Stop immediately on first context error

**Example**:
- 10 retries configured
- Context cancelled after attempt 2
- **Before**: Could attempt all 10 (context races with retry logic)
- **After**: Stop at attempt 2 ✅

### 3. Network Resource Conservation

**Before**: Retry attempts might start even with cancelled context
**After**: Check before each attempt

**Benefit**: No wasted network calls, server load reduction

## Race Condition Prevention

### Thread-Safe Test Implementation

Tests use `sync/atomic` for shared counters accessed from multiple goroutines:

```go
var attemptCount int32 // Atomic access

// In HTTP handler (separate goroutine):
atomic.AddInt32(&attemptCount, 1)

// In test verification:
count := atomic.LoadInt32(&attemptCount)
```

**Verified**: All tests pass with `-race` flag ✅

## Integration with Timeout Handling

This context error handling works seamlessly with the Context Priority Pattern:

1. **Context with deadline** → `client.Timeout = 0` → Context controls timeout
2. **Context cancellation** → Detected in retry logic → Immediate termination
3. **Error propagation** → Context errors bubble up with clear messages

### Combined Example

```go
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

opts := options.NewRequestOptionsBuilder().
    Get(url, nil).
    SetRetryConfig(&options.RetryConfig{
        MaxRetries: 5,
        RetryDelay: 500 * time.Millisecond,
    }).
    WithContext(ctx).
    Build()

resp, err := gocurl.Execute(opts)
// If times out:
// - Client.Timeout = 0 (context controls)
// - Retry logic checks context before each attempt
// - Error: "request failed due to context error (attempt X/5): context deadline exceeded"
```

## Best Practices Enforced

### ✅ Check Context Before Expensive Operations
- Before starting request execution
- Before each retry attempt
- Before sleeping during retry delays

### ✅ Use `errors.Is()` for Error Detection
```go
if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
    // Don't retry, return immediately
}
```

### ✅ Provide Context in Error Messages
```go
fmt.Errorf("request context cancelled during retries (attempt %d/%d): %w",
    attempt, retries, req.Context().Err())
```

### ✅ Clean Up Resources on Context Cancellation
```go
if resp != nil {
    resp.Body.Close() // Clean up before returning
}
```

## Test Results

### All Context Error Tests ✅
```bash
$ go test -v -run TestContextError
=== RUN   TestContextError_DeadlineExceeded
--- PASS: TestContextError_DeadlineExceeded (2.00s)
=== RUN   TestContextError_Cancelled
--- PASS: TestContextError_Cancelled (2.00s)
=== RUN   TestContextError_WithRetries
--- PASS: TestContextError_WithRetries (1.10s)
=== RUN   TestContextError_CancelDuringRetry
--- PASS: TestContextError_CancelDuringRetry (0.50s)
=== RUN   TestContextError_PropagationThroughLayers
--- PASS: TestContextError_PropagationThroughLayers (2.00s)
=== RUN   TestContextError_MultipleRequests_Independent
--- PASS: TestContextError_MultipleRequests_Independent (0.25s)
=== RUN   TestContextError_CheckBeforeRetry
--- PASS: TestContextError_CheckBeforeRetry (0.00s)
=== RUN   TestContextError_HTTPClientRespect
--- PASS: TestContextError_HTTPClientRespect (0.90s)
PASS
ok      github.com/maniartech/gocurl    10.450s
```

### With Race Detector ✅
```bash
$ go test -race -run TestContextError
PASS
ok      github.com/maniartech/gocurl    11.855s
```

**No race conditions detected** ✅

### Combined with Timeout Tests (24 total tests)
```bash
$ go test -run "TestTimeout|TestContextError|TestCreateHTTPClient"
PASS
ok      github.com/maniartech/gocurl    31.956s
```

## Files Modified

### `retry.go`
**Changes**:
- Added `context` import
- Added `errors` import for `errors.Is()`
- Pre-execution context check
- Pre-retry context check
- Context error detection
- Context-aware sleep implementation
- Enhanced error messages with attempt information

**Lines Added**: ~40 lines of context checking logic

### `context_error_test.go` (NEW)
**Content**:
- 8 comprehensive context error tests
- Thread-safe test implementation
- Tests for all cancellation scenarios
- Race condition free (verified with `-race`)

**Lines**: 311 lines

## Backward Compatibility

### ✅ Fully Backward Compatible

**No Breaking Changes**:
- Function signatures unchanged
- Error interface unchanged (still returns `error`)
- Behavior improved (more responsive to cancellation)
- Only enhancement: better error messages

**Users Benefit Automatically**:
- Existing code gets improved context handling
- No code changes required
- Tests will pass faster (early termination)

## Performance Metrics

### Context Cancellation Response Time

| Scenario | Before | After | Improvement |
|----------|--------|-------|-------------|
| Cancel during request | ~0ms | ~0ms | Same (HTTP client handles) |
| Cancel before retry | 5000ms | ~0ms | **99.9% faster** |
| Cancel during retry sleep | 5000ms | ~0ms | **99.9% faster** |
| Expired context at start | Varies | ~0ms | **Immediate** |

### Retry Attempt Reduction

| Scenario | Before | After | Reduction |
|----------|--------|-------|-----------|
| 10 retries, cancel at attempt 2 | Up to 10 | 2 | **80% fewer** |
| 5 retries, context timeout 800ms | Up to 5 | ~2 | **60% fewer** |
| Already-expired context | 1+ | 0 | **100% fewer** |

## Conclusion

The context error handling enhancements provide:

1. ✅ **Faster failure response** - Immediate termination on context cancellation
2. ✅ **Resource efficiency** - No wasted retry attempts or network calls
3. ✅ **Better error messages** - Clear indication of context errors with attempt info
4. ✅ **Race condition free** - Thread-safe implementation verified with `-race`
5. ✅ **Comprehensive testing** - 8 new tests covering all scenarios
6. ✅ **Full backward compatibility** - No breaking changes
7. ✅ **Industry best practices** - Proper use of `errors.Is()`, context checks, resource cleanup

The implementation follows Go best practices for context handling and provides robust, predictable behavior for all cancellation scenarios.
