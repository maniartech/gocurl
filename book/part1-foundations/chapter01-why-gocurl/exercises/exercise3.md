# Exercise 3: Implement Custom Retry Logic

**Difficulty:** ⭐⭐⭐ Advanced
**Time:** 60-90 minutes
**Concepts:** Retry strategies, exponential backoff, error classification, resilient APIs

## Objective

Build a production-ready API client with sophisticated retry logic, including exponential backoff, jitter, circuit breaker pattern, and detailed observability.

## Requirements

### Functional Requirements

1. Create a retry wrapper around `gocurl` calls
2. Implement exponential backoff with jitter
3. Classify errors (retryable vs non-retryable)
4. Implement circuit breaker pattern
5. Add detailed logging and metrics
6. Test with a flaky API

### Technical Requirements

1. Support configurable retry policies
2. Exponential backoff: wait 1s, 2s, 4s, 8s, etc.
3. Add jitter to prevent thundering herd
4. Circuit breaker states: Closed, Open, Half-Open
5. Log all retry attempts with timestamps
6. Return detailed error information

## Architecture

### Retry Strategy

```
Attempt 1 ──(fail)──> Wait 1s + jitter ──> Attempt 2
                                              │
                                           (fail)
                                              │
                                    Wait 2s + jitter ──> Attempt 3
                                                            │
                                                         (fail)
                                                            │
                                                  Wait 4s + jitter ──> Attempt 4
                                                                          │
                                                                       (success)
                                                                          │
                                                                      Return Result
```

### Circuit Breaker States

```
         ┌──────────────┐
         │   CLOSED     │ (Normal operation)
         │  (Allow all) │
         └───────┬──────┘
                 │
          Failure threshold
          reached (e.g., 5)
                 │
                 v
         ┌──────────────┐
         │    OPEN      │ (Reject all)
         │ (Fast fail)  │
         └───────┬──────┘
                 │
          After timeout
          (e.g., 60s)
                 │
                 v
         ┌──────────────┐
         │  HALF-OPEN   │ (Try one request)
         │(Test recovery)│
         └───────┬──────┘
                 │
          Success ────> Back to CLOSED
          Failure ────> Back to OPEN
```

## Getting Started

### 1. Project Structure

```bash
mkdir exercise3
cd exercise3
touch main.go retry.go circuit_breaker.go
go mod init resilient-client
go get github.com/maniartech/gocurl
```

### 2. Configuration Structure

```go
type RetryConfig struct {
    MaxRetries     int
    InitialBackoff time.Duration
    MaxBackoff     time.Duration
    BackoffFactor  float64
    Jitter         bool
    RetryableStatusCodes []int
    RetryableErrors []string
}

type CircuitBreakerConfig struct {
    FailureThreshold int
    SuccessThreshold int
    Timeout          time.Duration
}
```

### 3. Implementation Checklist

**Retry Logic:**
- [ ] Calculate exponential backoff
- [ ] Add random jitter
- [ ] Classify errors (retryable vs permanent)
- [ ] Respect max retries
- [ ] Log each attempt

**Circuit Breaker:**
- [ ] Track failure/success counts
- [ ] Implement state transitions
- [ ] Fast-fail when circuit is open
- [ ] Test recovery in half-open state
- [ ] Reset counters on success

**Integration:**
- [ ] Combine retry + circuit breaker
- [ ] Create clean API for users
- [ ] Add observability (logs, metrics)
- [ ] Test with simulated failures

## Core Functions to Implement

### 1. Retry Wrapper

```go
func WithRetry(ctx context.Context, config RetryConfig, fn func() (*http.Response, error)) (*http.Response, error) {
    // Implement retry logic with exponential backoff
}
```

### 2. Error Classifier

```go
func isRetryable(err error, statusCode int, config RetryConfig) bool {
    // Classify if error/status is retryable
}
```

### 3. Backoff Calculator

```go
func calculateBackoff(attempt int, config RetryConfig) time.Duration {
    // Calculate exponential backoff with jitter
}
```

### 4. Circuit Breaker

```go
type CircuitBreaker struct {
    state          State
    failures       int
    successes      int
    lastFailure    time.Time
    config         CircuitBreakerConfig
    mu             sync.RWMutex
}

func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
    // Execute with circuit breaker protection
}
```

## Example Usage

```go
func main() {
    config := RetryConfig{
        MaxRetries:     5,
        InitialBackoff: time.Second,
        MaxBackoff:     30 * time.Second,
        BackoffFactor:  2.0,
        Jitter:         true,
        RetryableStatusCodes: []int{500, 502, 503, 504, 429},
    }

    cbConfig := CircuitBreakerConfig{
        FailureThreshold: 5,
        SuccessThreshold: 2,
        Timeout:          60 * time.Second,
    }

    cb := NewCircuitBreaker(cbConfig)
    client := NewResilientClient(config, cb)

    // Make resilient API call
    data, err := client.Get(ctx, "https://api.flaky-service.com/data")
    if err != nil {
        log.Printf("Request failed after retries: %v", err)
        return
    }

    fmt.Printf("Success: %s\n", data)
}
```

## Expected Output

```
[Attempt 1/5] Calling https://api.example.com/data
[Attempt 1/5] Failed with status 503 - retrying in 1.2s
[Attempt 2/5] Calling https://api.example.com/data
[Attempt 2/5] Failed with status 503 - retrying in 2.4s
[Attempt 3/5] Calling https://api.example.com/data
[Attempt 3/5] Failed with status 503 - retrying in 4.8s
[Attempt 4/5] Calling https://api.example.com/data
[Success] Request succeeded on attempt 4
Total time: 8.6s
```

## Bonus Challenges

1. **Metrics Collection**: Track retry counts, success rates, circuit breaker trips
2. **Adaptive Backoff**: Adjust backoff based on server headers (Retry-After)
3. **Per-endpoint Circuit Breakers**: Different circuit breakers for different endpoints
4. **Retry Budgets**: Limit total retry time across all requests
5. **Custom Retry Predicates**: Allow users to define custom retry conditions
6. **Distributed Circuit Breaker**: Share state across multiple instances (Redis)
7. **OpenTelemetry Integration**: Add distributed tracing spans

## Testing Strategy

### Create a Mock Flaky Server

```go
func main() {
    // Start a test server that fails N times then succeeds
    http.HandleFunc("/flaky", func(w http.ResponseWriter, r *http.Request) {
        attemptCount := getAttemptCount(r)

        if attemptCount < 3 {
            w.WriteHeader(503)
            fmt.Fprintf(w, "Service Unavailable (attempt %d)", attemptCount)
            return
        }

        w.WriteHeader(200)
        fmt.Fprintf(w, "Success after %d attempts!", attemptCount)
    })

    log.Println("Flaky server running on :8080")
    http.ListenAndServe(":8080", nil)
}
```

### Test Scenarios

1. **Temporary Failures**: API fails 2-3 times then succeeds
2. **Permanent Failures**: API always returns 404
3. **Timeout**: API is too slow, context timeout triggers
4. **Circuit Breaker**: Multiple failures trigger circuit open
5. **Recovery**: Circuit breaker transitions from open to half-open to closed

## Hints

<details>
<summary>Hint 1: Exponential Backoff with Jitter</summary>

```go
func calculateBackoff(attempt int, config RetryConfig) time.Duration {
    // Exponential backoff: 1s, 2s, 4s, 8s, 16s, 32s
    backoff := config.InitialBackoff * time.Duration(math.Pow(config.BackoffFactor, float64(attempt)))

    // Cap at max backoff
    if backoff > config.MaxBackoff {
        backoff = config.MaxBackoff
    }

    // Add jitter to prevent thundering herd
    if config.Jitter {
        jitter := time.Duration(rand.Float64() * float64(backoff) * 0.3)
        backoff += jitter
    }

    return backoff
}
```
</details>

<details>
<summary>Hint 2: Error Classification</summary>

```go
func isRetryable(err error, statusCode int, config RetryConfig) bool {
    // Network errors are always retryable
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            return false // Don't retry timeouts
        }
        // Check for network errors, DNS errors, etc.
        return true
    }

    // Check if status code is in retryable list
    for _, code := range config.RetryableStatusCodes {
        if statusCode == code {
            return true
        }
    }

    // 4xx errors are usually not retryable (except 429)
    if statusCode >= 400 && statusCode < 500 && statusCode != 429 {
        return false
    }

    // 5xx errors are retryable
    return statusCode >= 500
}
```
</details>

<details>
<summary>Hint 3: Circuit Breaker State Machine</summary>

```go
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
    cb.mu.Lock()

    switch cb.state {
    case StateClosed:
        cb.mu.Unlock()
        err := fn()
        cb.recordResult(err)
        return err

    case StateOpen:
        if time.Since(cb.lastFailure) > cb.config.Timeout {
            cb.state = StateHalfOpen
            cb.mu.Unlock()
            return cb.Call(ctx, fn) // Retry in half-open
        }
        cb.mu.Unlock()
        return ErrCircuitOpen

    case StateHalfOpen:
        cb.mu.Unlock()
        err := fn()
        cb.recordHalfOpenResult(err)
        return err
    }

    cb.mu.Unlock()
    return nil
}
```
</details>

## Common Pitfalls

❌ **Mistake**: Retrying non-idempotent operations (POST, DELETE)
✅ **Solution**: Only retry safe methods (GET) or make operations idempotent

❌ **Mistake**: No maximum backoff limit
✅ **Solution**: Cap backoff at reasonable max (e.g., 30s)

❌ **Mistake**: No jitter leads to thundering herd
✅ **Solution**: Add random jitter to spread out retries

❌ **Mistake**: Circuit breaker never recovers
✅ **Solution**: Implement half-open state to test recovery

❌ **Mistake**: Retrying forever
✅ **Solution**: Respect context deadlines and max retries

## Production Considerations

1. **Observability**: Log every retry, circuit breaker state change
2. **Metrics**: Track retry rates, circuit breaker trips, latency percentiles
3. **Alerts**: Alert on high retry rates or circuit breaker opens
4. **Configuration**: Make retry config tunable without code changes
5. **Testing**: Chaos engineering to test resilience

## Next Steps

After completing this exercise:
1. Apply retry logic to your real projects
2. Study production retry libraries (go-retryablehttp, etc.)
3. Learn about distributed system patterns
4. Move on to Exercise 4 for CLI development!

## Solution

The complete solution is available in `solutions/exercise3/`.
