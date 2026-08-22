# Exercise 3: Context Management

Master context patterns including timeouts, cancellation, and context propagation.

## Objectives

- Implement proper timeout handling
- Handle graceful cancellation
- Understand context propagation
- Build resilient request patterns

## Scenario

You're building a monitoring service that makes health check requests to multiple services. The system needs to:

1. Enforce timeouts to prevent hanging
2. Allow manual cancellation of checks
3. Propagate context across function calls
4. Handle deadline exceeded gracefully

## Requirements

### Part 1: Timeout Management (25 points)

Implement `checkServiceWithTimeout()` that:
- Accepts custom timeout duration
- Handles fast and slow endpoints
- Returns appropriate error for timeout

### Part 2: Cancellation Handling (25 points)

Implement `cancellableHealthCheck()` that:
- Starts health check in goroutine
- Accepts cancel signal
- Cleans up properly on cancellation

### Part 3: Context Propagation (25 points)

Implement `ServiceChecker` with hierarchical context:
- Parent context with overall timeout
- Child contexts per service
- Proper cleanup and cancellation

### Part 4: Deadline Management (25 points)

Implement `checkBeforeDeadline()` that:
- Uses absolute deadline (not relative timeout)
- Checks multiple services within deadline
- Reports services checked before deadline expires

## Starter Code

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/maniartech/gocurl"
)

// TODO 1: Implement checkServiceWithTimeout
// Return: true if service responds within timeout, false if timeout/error
func checkServiceWithTimeout(url string, timeout time.Duration) (bool, error) {
    // TODO: Create context with timeout
    // TODO: Make request with context
    // TODO: Check for timeout error

    return false, nil
}

// TODO 2: Implement cancellableHealthCheck
// Return: error channel that receives result
func cancellableHealthCheck(url string, cancel <-chan struct{}) <-chan error {
    errCh := make(chan error, 1)

    go func() {
        // TODO: Create cancellable context
        // TODO: Make request in goroutine
        // TODO: Handle cancel signal
        // TODO: Send result to channel
    }()

    return errCh
}

// ServiceChecker checks multiple services with context propagation
type ServiceChecker struct {
    services []string
    timeout  time.Duration
}

// TODO 3: Implement NewServiceChecker
func NewServiceChecker(timeout time.Duration) *ServiceChecker {
    return &ServiceChecker{
        services: []string{},
        timeout:  timeout,
    }
}

// TODO 4: Implement AddService
func (sc *ServiceChecker) AddService(url string) {
    // TODO: Add service to list
}

// TODO 5: Implement CheckAll with context propagation
// Uses parent context and creates child context per service
func (sc *ServiceChecker) CheckAll(ctx context.Context) map[string]error {
    results := make(map[string]error)

    // TODO: For each service:
    // TODO:   Create child context with timeout
    // TODO:   Make request
    // TODO:   Store result
    // TODO: Handle parent context cancellation

    return results
}

// TODO 6: Implement checkBeforeDeadline
// Check services until deadline, return list of checked services
func checkBeforeDeadline(services []string, deadline time.Time) ([]string, error) {
    // TODO: Create context with deadline
    // TODO: Check services while time remains
    // TODO: Stop when deadline approaches

    return nil, nil
}

func main() {
    fmt.Println("⏱️  Exercise 3: Context Management\n")

    // Test 1: Timeout management
    fmt.Println("Test 1: Timeout Management")

    // Fast endpoint (should succeed)
    ok, err := checkServiceWithTimeout("https://httpbin.org/delay/1", 3*time.Second)
    if err != nil {
        log.Printf("   ❌ Error: %v\n", err)
    } else if ok {
        fmt.Println("   ✅ Fast endpoint: Success")
    }

    // Slow endpoint (should timeout)
    ok, err = checkServiceWithTimeout("https://httpbin.org/delay/5", 2*time.Second)
    if err != nil {
        fmt.Printf("   ✅ Slow endpoint: Timeout (expected) - %v\n", err)
    } else if !ok {
        fmt.Println("   ✅ Slow endpoint: Timed out correctly")
    }

    // Test 2: Cancellation
    fmt.Println("\nTest 2: Cancellation Handling")

    cancelCh := make(chan struct{})
    errCh := cancellableHealthCheck("https://httpbin.org/delay/10", cancelCh)

    // Cancel after 500ms
    time.Sleep(500 * time.Millisecond)
    close(cancelCh)

    err = <-errCh
    if err != nil && err.Error() == "cancelled" {
        fmt.Println("   ✅ Health check cancelled successfully")
    } else {
        fmt.Printf("   ⚠️  Expected cancellation, got: %v\n", err)
    }

    // Test 3: Context propagation
    fmt.Println("\nTest 3: Context Propagation")

    checker := NewServiceChecker(3 * time.Second)
    checker.AddService("https://httpbin.org/delay/1")
    checker.AddService("https://httpbin.org/delay/2")
    checker.AddService("https://httpbin.org/delay/5") // Will timeout

    parentCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    results := checker.CheckAll(parentCtx)
    for service, err := range results {
        if err != nil {
            fmt.Printf("   ⚠️  %s: %v\n", service, err)
        } else {
            fmt.Printf("   ✅ %s: OK\n", service)
        }
    }

    // Test 4: Deadline management
    fmt.Println("\nTest 4: Deadline Management")

    services := []string{
        "https://httpbin.org/delay/1",
        "https://httpbin.org/delay/1",
        "https://httpbin.org/delay/1",
        "https://httpbin.org/delay/1",
        "https://httpbin.org/delay/1",
    }

    deadline := time.Now().Add(4 * time.Second)
    checked, err := checkBeforeDeadline(services, deadline)
    if err != nil {
        log.Printf("   ❌ Error: %v\n", err)
    } else {
        fmt.Printf("   ✅ Checked %d/%d services before deadline\n", len(checked), len(services))
    }

    fmt.Println("\n" + repeatString("=", 60))
    fmt.Println("Exercise complete! Review your implementation.")
    fmt.Println(repeatString("=", 60))
}

func repeatString(s string, count int) string {
    result := ""
    for i := 0; i < count; i++ {
        result += s
    }
    return result
}
```

## Self-Check Criteria

### Part 1: Timeouts
- ✅ Uses `context.WithTimeout()`
- ✅ Checks for `context.DeadlineExceeded`
- ✅ Returns appropriate error

### Part 2: Cancellation
- ✅ Uses `context.WithCancel()`
- ✅ Handles cancel channel properly
- ✅ Cleans up goroutines

### Part 3: Propagation
- ✅ Parent context created first
- ✅ Child contexts inherit from parent
- ✅ All contexts cleaned up with `defer cancel()`

### Part 4: Deadlines
- ✅ Uses `context.WithDeadline()`
- ✅ Checks `time.Until(deadline)`
- ✅ Stops before deadline exceeded

## Expected Output

```
Test 1: Timeout Management
   ✅ Fast endpoint: Success
   ✅ Slow endpoint: Timeout (expected)

Test 2: Cancellation Handling
   ✅ Health check cancelled successfully

Test 3: Context Propagation
   ✅ https://httpbin.org/delay/1: OK
   ✅ https://httpbin.org/delay/2: OK
   ⚠️  https://httpbin.org/delay/5: context deadline exceeded

Test 4: Deadline Management
   ✅ Checked 3/5 services before deadline
```

## Bonus Challenges

1. Add retry logic with exponential backoff
2. Implement parallel checking with sync.WaitGroup
3. Add progress reporting during checks
4. Implement circuit breaker pattern
5. Add metrics collection (success/failure counts)

## Common Pitfalls

⚠️ **Pitfall 1:** Forgetting `defer cancel()`
- Always defer cancel() after creating context
- Prevents resource leaks

⚠️ **Pitfall 2:** Not checking context errors
- Check if error is `context.DeadlineExceeded` or `context.Canceled`
- Use `errors.Is()` or `ctx.Err()`

⚠️ **Pitfall 3:** Blocking on channels
- Use select with context.Done()
- Don't block indefinitely

## Resources

- Chapter 3 section on context patterns
- Example 03-context-patterns
- Example 05-error-handling
