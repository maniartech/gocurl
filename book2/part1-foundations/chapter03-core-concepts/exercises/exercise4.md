# Exercise 4: Response Processing Pipeline

Create a complete response processing pipeline with error handling, retries, and status code validation.

## Objectives

- Build production-ready response handling
- Implement retry logic with backoff
- Handle various HTTP status codes
- Process different response types
- Implement circuit breaker pattern

## Scenario

You're building an API gateway that needs to:

1. Make requests with automatic retries
2. Handle different HTTP status codes appropriately
3. Parse various response formats
4. Implement circuit breaker for failing services
5. Log and report errors properly

## Requirements

### Part 1: Retry Logic (20 points)

Implement `retryableRequest()` that:
- Retries on 5xx errors and timeouts
- Uses exponential backoff
- Has configurable max retries
- Stops on 4xx errors (don't retry)

### Part 2: Status Code Handling (20 points)

Implement `handleResponse()` that:
- Returns different errors for different status codes
- Extracts error messages from response body
- Provides helpful error context

### Part 3: Response Parser (20 points)

Implement `ResponseParser` that:
- Detects content type
- Parses JSON, text, and binary
- Validates response structure
- Handles malformed responses

### Part 4: Circuit Breaker (20 points)

Implement `CircuitBreaker` that:
- Opens after N consecutive failures
- Allows test requests after timeout
- Closes when requests succeed again
- Tracks failure statistics

### Part 5: Complete Pipeline (20 points)

Combine all components into `RequestPipeline`:
- Uses retryable requests
- Implements circuit breaker
- Parses responses
- Handles all error cases

## Starter Code

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log"
    "net/http"
    "time"

    "github.com/maniartech/gocurl"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
    MaxRetries     int
    InitialBackoff time.Duration
    MaxBackoff     time.Duration
}

// TODO 1: Implement retryableRequest
// Retries on 5xx and timeouts, stops on 4xx
func retryableRequest(ctx context.Context, url string, config RetryConfig) (*http.Response, error) {
    var lastErr error
    backoff := config.InitialBackoff

    for attempt := 0; attempt <= config.MaxRetries; attempt++ {
        // TODO: Make request
        // TODO: Check status code
        // TODO: Implement exponential backoff
        // TODO: Return on success or non-retryable error
    }

    return nil, lastErr
}

// TODO 2: Implement handleResponse
// Returns appropriate error based on status code
func handleResponse(resp *http.Response) error {
    // TODO: Handle different status codes:
    // - 2xx: success
    // - 4xx: client error (return specific error)
    // - 5xx: server error (return specific error)

    return nil
}

// ResponseParser parses different response types
type ResponseParser struct{}

// TODO 3: Implement NewResponseParser
func NewResponseParser() *ResponseParser {
    return &ResponseParser{}
}

// TODO 4: Implement ParseJSON
func (rp *ResponseParser) ParseJSON(resp *http.Response, target interface{}) error {
    // TODO: Validate content type
    // TODO: Read and parse JSON
    // TODO: Handle errors

    return nil
}

// TODO 5: Implement ParseText
func (rp *ResponseParser) ParseText(resp *http.Response) (string, error) {
    // TODO: Read body as string

    return "", nil
}

// TODO 6: Implement ParseBinary
func (rp *ResponseParser) ParseBinary(resp *http.Response) ([]byte, error) {
    // TODO: Read body as bytes

    return nil, nil
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
    maxFailures     int
    resetTimeout    time.Duration
    consecutiveFails int
    lastFailTime     time.Time
    state           string // "closed", "open", "half-open"
}

// TODO 7: Implement NewCircuitBreaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        maxFailures:  maxFailures,
        resetTimeout: resetTimeout,
        state:        "closed",
    }
}

// TODO 8: Implement Call
// Wraps request with circuit breaker logic
func (cb *CircuitBreaker) Call(fn func() error) error {
    // TODO: Check if circuit is open
    // TODO: Execute function
    // TODO: Record success/failure
    // TODO: Update circuit state

    return nil
}

// TODO 9: Implement recordSuccess
func (cb *CircuitBreaker) recordSuccess() {
    // TODO: Reset failure count, close circuit
}

// TODO 10: Implement recordFailure
func (cb *CircuitBreaker) recordFailure() {
    // TODO: Increment failure count
    // TODO: Open circuit if threshold reached
}

// RequestPipeline combines all components
type RequestPipeline struct {
    retryConfig    RetryConfig
    circuitBreaker *CircuitBreaker
    parser         *ResponseParser
}

// TODO 11: Implement NewRequestPipeline
func NewRequestPipeline(config RetryConfig, cb *CircuitBreaker) *RequestPipeline {
    return &RequestPipeline{
        retryConfig:    config,
        circuitBreaker: cb,
        parser:         NewResponseParser(),
    }
}

// TODO 12: Implement Execute
// Complete request pipeline: circuit breaker → retry → parse
func (rp *RequestPipeline) Execute(ctx context.Context, url string) (interface{}, error) {
    var result interface{}

    // TODO: Wrap in circuit breaker
    // TODO: Make retryable request
    // TODO: Handle response
    // TODO: Parse response

    return result, nil
}

func main() {
    fmt.Println("🔄 Exercise 4: Response Processing Pipeline\n")

    // Test 1: Retry logic
    fmt.Println("Test 1: Retry Logic")
    config := RetryConfig{
        MaxRetries:     3,
        InitialBackoff: 100 * time.Millisecond,
        MaxBackoff:     2 * time.Second,
    }

    ctx := context.Background()

    // Test with 500 error (should retry)
    resp, err := retryableRequest(ctx, "https://httpbin.org/status/500", config)
    if err != nil {
        fmt.Printf("   ✅ Retried and failed (expected): %v\n", err)
    }
    if resp != nil {
        resp.Body.Close()
    }

    // Test with 404 (should NOT retry)
    resp, err = retryableRequest(ctx, "https://httpbin.org/status/404", config)
    if err != nil {
        fmt.Printf("   ✅ No retry on 404 (expected): %v\n", err)
    }
    if resp != nil {
        resp.Body.Close()
    }

    // Test 2: Status code handling
    fmt.Println("\nTest 2: Status Code Handling")

    testResp, _ := gocurl.Curl(ctx, "https://httpbin.org/status/403")
    if err := handleResponse(testResp); err != nil {
        fmt.Printf("   ✅ Handled 403: %v\n", err)
    }
    testResp.Body.Close()

    // Test 3: Response parsing
    fmt.Println("\nTest 3: Response Parsing")

    parser := NewResponseParser()

    // JSON parsing
    jsonResp, _ := gocurl.Curl(ctx, "https://api.github.com/users/golang")
    var user map[string]interface{}
    if err := parser.ParseJSON(jsonResp, &user); err != nil {
        fmt.Printf("   ❌ JSON parse error: %v\n", err)
    } else {
        fmt.Printf("   ✅ Parsed JSON: %v\n", user["login"])
    }
    jsonResp.Body.Close()

    // Test 4: Circuit breaker
    fmt.Println("\nTest 4: Circuit Breaker")

    cb := NewCircuitBreaker(3, 5*time.Second)

    // Make 3 failing requests (should open circuit)
    for i := 0; i < 5; i++ {
        err := cb.Call(func() error {
            resp, err := gocurl.Curl(ctx, "https://httpbin.org/status/500")
            if resp != nil {
                resp.Body.Close()
            }
            if err != nil {
                return err
            }
            if resp.StatusCode >= 500 {
                return errors.New("server error")
            }
            return nil
        })

        if err != nil && err.Error() == "circuit breaker open" {
            fmt.Printf("   ✅ Circuit opened after %d failures\n", i)
            break
        }
    }

    // Test 5: Complete pipeline
    fmt.Println("\nTest 5: Complete Request Pipeline")

    pipeline := NewRequestPipeline(config, cb)

    result, err := pipeline.Execute(ctx, "https://api.github.com/users/golang")
    if err != nil {
        fmt.Printf("   ⚠️  Pipeline error: %v\n", err)
    } else {
        fmt.Printf("   ✅ Pipeline executed successfully\n")
        fmt.Printf("   📦 Result: %v\n", result)
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

### Part 1: Retry Logic
- ✅ Implements exponential backoff
- ✅ Retries on 5xx, timeouts
- ✅ Stops on 4xx errors
- ✅ Respects max retries

### Part 2: Status Handling
- ✅ Different errors for different codes
- ✅ Extracts error from body
- ✅ Provides context

### Part 3: Response Parser
- ✅ Validates content type
- ✅ Handles JSON, text, binary
- ✅ Error handling for malformed

### Part 4: Circuit Breaker
- ✅ Opens after N failures
- ✅ Half-open for testing
- ✅ Closes on success
- ✅ Tracks statistics

### Part 5: Pipeline
- ✅ Combines all components
- ✅ Proper error propagation
- ✅ Clean code structure

## Expected Output

```
Test 1: Retry Logic
   ✅ Retried and failed (expected): ...
   ✅ No retry on 404 (expected): ...

Test 2: Status Code Handling
   ✅ Handled 403: forbidden

Test 3: Response Parsing
   ✅ Parsed JSON: golang

Test 4: Circuit Breaker
   ✅ Circuit opened after 3 failures

Test 5: Complete Request Pipeline
   ✅ Pipeline executed successfully
   📦 Result: map[...]
```

## Bonus Challenges

1. Add request/response logging middleware
2. Implement rate limiting
3. Add metrics (success rate, latency)
4. Implement bulkhead pattern
5. Add distributed tracing

## Resources

- Chapter 3 section on response handling
- Example 04-response-handling
- Example 05-error-handling
- Example 10-practical-client
