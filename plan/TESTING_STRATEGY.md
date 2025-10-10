# GoCurl Testing Strategy - Military-Grade Robustness

## Overview

This document outlines the comprehensive testing strategy to ensure GoCurl achieves military-grade robustness with zero race conditions, proven performance, and battle-tested reliability.

## Testing Principles

1. **Zero Tolerance for Races** - All tests must pass with `-race` flag
2. **Continuous Load Testing** - Regular high-load scenarios in CI
3. **Chaos Engineering** - Test failure modes and recovery
4. **Fuzz Everything** - 100M+ iterations on all parsers
5. **Benchmark Regression** - CI fails on >5% performance degradation
6. **Real-World Scenarios** - Test actual API integration patterns

## Test Categories

### 1. Unit Tests (Foundation)

**Coverage Target**: 90%+ on core paths

**Files**:
- `convert_test.go` - Token conversion logic
- `api_test.go` - High-level API
- `parser_test.go` - Command parsing
- `variables_test.go` - Variable substitution
- `pools_test.go` - Buffer pool behavior
- `errors_test.go` - Error handling

**Key Tests**:
```go
func TestTokenConversion(t *testing.T) {
    // Test every curl flag combination
    // Verify correct option extraction
    // Test edge cases (empty values, special chars)
}

func TestVariableSubstitution(t *testing.T) {
    // Test ${var} and $var syntax
    // Test escaping \${var}
    // Test undefined variable errors
    // Test nested/complex cases
}

func TestBufferPooling(t *testing.T) {
    // Verify Get/Put semantics
    // Test pool cleanup
    // Ensure no buffer reuse corruption
}
```

### 2. Race Condition Tests (Critical)

**Coverage Target**: All concurrent code paths

**Files**:
- `race_test.go` - Dedicated race detection tests

**Required Tests**:
```go
func TestConcurrentRequests(t *testing.T) {
    const numGoroutines = 10000
    var wg sync.WaitGroup
    var errors atomic.Int64
    
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            cmd := fmt.Sprintf("curl https://httpbin.org/get?id=%d", id)
            resp, err := gocurl.Request(cmd, nil)
            if err != nil {
                errors.Add(1)
                return
            }
            defer resp.Body.Close()
            
            // Verify response
            if resp.StatusCode != 200 {
                errors.Add(1)
            }
        }(i)
    }
    
    wg.Wait()
    
    if errors.Load() > 0 {
        t.Errorf("Failed requests: %d", errors.Load())
    }
}

func TestConcurrentClientPool(t *testing.T) {
    // Hammer the client pool with concurrent access
    const concurrency = 1000
    var wg sync.WaitGroup
    
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            // Different configs to stress pool
            opts := &options.RequestOptions{
                URL:     "https://example.com",
                Timeout: time.Duration(rand.Intn(10)) * time.Second,
            }
            
            client, err := GetClient(opts)
            if err != nil {
                t.Error(err)
                return
            }
            
            // Use client
            _, _ = client.Get("https://example.com")
        }()
    }
    
    wg.Wait()
}

func TestConcurrentBufferPool(t *testing.T) {
    // Stress test sync.Pool operations
    const iterations = 100000
    var wg sync.WaitGroup
    
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            for j := 0; j < iterations/100; j++ {
                buf := bufferPool.Get().(*bytes.Buffer)
                buf.WriteString("test data")
                buf.Reset()
                bufferPool.Put(buf)
            }
        }()
    }
    
    wg.Wait()
}

func TestConcurrentVariableSubstitution(t *testing.T) {
    // Parallel variable substitution
    const concurrency = 1000
    var wg sync.WaitGroup
    
    vars := gocurl.Variables{
        "token": "secret-token-" + strconv.Itoa(rand.Int()),
        "url":   "https://api.example.com",
    }
    
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            cmd := fmt.Sprintf(`curl -H "Authorization: Bearer ${token}" ${url}/user/%d`, id)
            _, err := gocurl.Request(cmd, vars)
            if err != nil {
                t.Error(err)
            }
        }(i)
    }
    
    wg.Wait()
}
```

**CI Integration**:
```yaml
# .github/workflows/race.yml
name: Race Detection

on: [push, pull_request]

jobs:
  race-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run tests with race detector
        run: go test -race -v -timeout 30m ./...
      
      - name: Run concurrent stress tests
        run: go test -race -run=Concurrent -v -timeout 1h ./...
```

### 3. Benchmark Tests (Performance)

**Coverage Target**: All critical paths

**Files**:
- `benchmark_test.go` - Performance benchmarks
- `benchmark_regression_test.go` - Regression detection

**Required Benchmarks**:
```go
func BenchmarkRequestConstruction(b *testing.B) {
    cmd := "curl -X POST -H 'Content-Type: application/json' -d '{\"key\":\"value\"}' https://api.example.com"
    vars := gocurl.Variables{}
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        _, _ = gocurl.Request(cmd, vars)
    }
}

func BenchmarkVsNetHTTP(b *testing.B) {
    b.Run("gocurl", func(b *testing.B) {
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            _, _ = gocurl.Request("curl https://example.com", nil)
        }
    })
    
    b.Run("net/http", func(b *testing.B) {
        client := &http.Client{}
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            resp, _ := client.Get("https://example.com")
            if resp != nil {
                resp.Body.Close()
            }
        }
    })
}

func BenchmarkConcurrent(b *testing.B) {
    benchmarks := []int{1, 10, 100, 1000, 10000}
    
    for _, concurrency := range benchmarks {
        b.Run(fmt.Sprintf("Concurrent-%d", concurrency), func(b *testing.B) {
            b.ReportAllocs()
            b.SetParallelism(concurrency)
            
            b.RunParallel(func(pb *testing.PB) {
                for pb.Next() {
                    _, _ = gocurl.Request("curl https://example.com", nil)
                }
            })
        })
    }
}

func BenchmarkZeroAlloc(b *testing.B) {
    // Verify zero allocations on request construction
    cmd := "curl https://example.com"
    vars := gocurl.Variables{}
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        _, _ = gocurl.Request(cmd, vars)
    }
    
    // Verify 0 allocs/op
    if testing.AllocsPerRun(100, func() {
        _, _ = gocurl.Request(cmd, vars)
    }) > 0 {
        b.Error("Expected zero allocations on critical path")
    }
}
```

**Regression Detection**:
```go
// benchmark_regression_test.go
func TestBenchmarkRegression(t *testing.T) {
    // Load baseline benchmarks from file
    baseline := loadBaseline("benchmarks.json")
    
    // Run current benchmarks
    current := runBenchmarks()
    
    // Compare
    for name, curr := range current {
        base := baseline[name]
        degradation := (curr.NsPerOp - base.NsPerOp) / base.NsPerOp * 100
        
        if degradation > 5.0 {
            t.Errorf("Benchmark %s regressed by %.2f%%", name, degradation)
        }
    }
}
```

### 4. Load Tests (Sustained Throughput)

**Coverage Target**: Real-world production scenarios

**Files**:
- `load_test.go` - Load testing suite

**Required Tests**:
```go
func TestSustainedLoad(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }
    
    const (
        targetRPS  = 10000
        duration   = 1 * time.Hour
        maxMemoryMB = 500
    )
    
    var (
        requestCount atomic.Int64
        errorCount   atomic.Int64
        startMem     runtime.MemStats
    )
    
    runtime.ReadMemStats(&startMem)
    
    ctx, cancel := context.WithTimeout(context.Background(), duration)
    defer cancel()
    
    // Spawn workers
    numWorkers := runtime.NumCPU() * 10
    for i := 0; i < numWorkers; i++ {
        go func(id int) {
            ticker := time.NewTicker(time.Second / time.Duration(targetRPS/numWorkers))
            defer ticker.Stop()
            
            for {
                select {
                case <-ctx.Done():
                    return
                case <-ticker.C:
                    resp, err := gocurl.Request("curl https://httpbin.org/get", nil)
                    requestCount.Add(1)
                    
                    if err != nil {
                        errorCount.Add(1)
                        continue
                    }
                    
                    resp.Body.Close()
                }
            }
        }(i)
    }
    
    // Monitor progress
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            // Final stats
            var endMem runtime.MemStats
            runtime.ReadMemStats(&endMem)
            
            memGrowthMB := float64(endMem.Alloc-startMem.Alloc) / 1024 / 1024
            totalReqs := requestCount.Load()
            totalErrs := errorCount.Load()
            errorRate := float64(totalErrs) / float64(totalReqs) * 100
            
            t.Logf("Load test complete:")
            t.Logf("  Total requests: %d", totalReqs)
            t.Logf("  Total errors: %d (%.2f%%)", totalErrs, errorRate)
            t.Logf("  Memory growth: %.2f MB", memGrowthMB)
            
            if memGrowthMB > float64(maxMemoryMB) {
                t.Errorf("Memory growth exceeded limit: %.2f MB > %d MB", memGrowthMB, maxMemoryMB)
            }
            
            if errorRate > 1.0 {
                t.Errorf("Error rate too high: %.2f%%", errorRate)
            }
            
            return
            
        case <-ticker.C:
            reqs := requestCount.Load()
            errs := errorCount.Load()
            t.Logf("Progress: %d requests, %d errors", reqs, errs)
        }
    }
}

func TestBurstLoad(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping burst test in short mode")
    }
    
    const (
        burstSize = 100000
        burstDuration = 10 * time.Second
    )
    
    var wg sync.WaitGroup
    var errorCount atomic.Int64
    
    start := time.Now()
    
    for i := 0; i < burstSize; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            resp, err := gocurl.Request("curl https://httpbin.org/get", nil)
            if err != nil {
                errorCount.Add(1)
                return
            }
            defer resp.Body.Close()
        }(i)
        
        // Pace the requests
        if i%10000 == 0 {
            time.Sleep(time.Millisecond)
        }
    }
    
    wg.Wait()
    duration := time.Since(start)
    
    rps := float64(burstSize) / duration.Seconds()
    errors := errorCount.Load()
    
    t.Logf("Burst test results:")
    t.Logf("  Requests: %d in %v", burstSize, duration)
    t.Logf("  RPS: %.0f", rps)
    t.Logf("  Errors: %d", errors)
    
    if duration > burstDuration*2 {
        t.Errorf("Burst took too long: %v", duration)
    }
}

func TestSoakTest(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping soak test in short mode")
    }
    
    const duration = 24 * time.Hour
    
    ctx, cancel := context.WithTimeout(context.Background(), duration)
    defer cancel()
    
    var memSamples []uint64
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    // Background request generator
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            default:
                _, _ = gocurl.Request("curl https://httpbin.org/get", nil)
                time.Sleep(100 * time.Millisecond)
            }
        }
    }()
    
    // Monitor memory
    for {
        select {
        case <-ctx.Done():
            // Analyze memory growth
            if len(memSamples) < 2 {
                t.Error("Not enough memory samples")
                return
            }
            
            // Calculate linear regression to detect growth
            growth := calculateMemoryGrowth(memSamples)
            if growth > 1.0 { // 1 MB/hour
                t.Errorf("Memory leak detected: %.2f MB/hour", growth)
            }
            
            return
            
        case <-ticker.C:
            var mem runtime.MemStats
            runtime.ReadMemStats(&mem)
            memSamples = append(memSamples, mem.Alloc)
            
            t.Logf("Memory: %d MB", mem.Alloc/1024/1024)
        }
    }
}
```

### 5. Stress Tests (Breaking Point)

**Coverage Target**: Find limits

**Files**:
- `stress_test.go` - Stress testing

**Required Tests**:
```go
func TestFindBreakingPoint(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping stress test in short mode")
    }
    
    concurrency := 1000
    maxConcurrency := 1000000
    step := 10000
    
    for concurrency <= maxConcurrency {
        t.Logf("Testing %d concurrent requests...", concurrency)
        
        var wg sync.WaitGroup
        var successCount atomic.Int64
        var errorCount atomic.Int64
        
        start := time.Now()
        
        for i := 0; i < concurrency; i++ {
            wg.Add(1)
            go func() {
                defer wg.Done()
                
                resp, err := gocurl.Request("curl https://httpbin.org/get", nil)
                if err != nil {
                    errorCount.Add(1)
                    return
                }
                defer resp.Body.Close()
                successCount.Add(1)
            }()
        }
        
        wg.Wait()
        duration := time.Since(start)
        
        success := successCount.Load()
        errors := errorCount.Load()
        successRate := float64(success) / float64(concurrency) * 100
        
        t.Logf("  Duration: %v", duration)
        t.Logf("  Success: %d (%.2f%%)", success, successRate)
        t.Logf("  Errors: %d", errors)
        
        // If success rate drops below 95%, we've found breaking point
        if successRate < 95.0 {
            t.Logf("Breaking point found at ~%d concurrent requests", concurrency)
            break
        }
        
        concurrency += step
    }
}

func TestResourceExhaustion(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping resource test in short mode")
    }
    
    // Test file descriptor exhaustion
    t.Run("FileDescriptors", func(t *testing.T) {
        var openConns []*http.Response
        defer func() {
            for _, conn := range openConns {
                conn.Body.Close()
            }
        }()
        
        for i := 0; i < 10000; i++ {
            resp, err := gocurl.Request("curl https://httpbin.org/get", nil)
            if err != nil {
                t.Logf("FD limit reached at ~%d connections", i)
                break
            }
            openConns = append(openConns, resp)
        }
    })
    
    // Test memory exhaustion
    t.Run("Memory", func(t *testing.T) {
        const largeBodySize = 100 * 1024 * 1024 // 100 MB
        
        for i := 0; i < 100; i++ {
            resp, err := gocurl.Request("curl https://httpbin.org/bytes/"+strconv.Itoa(largeBodySize), nil)
            if err != nil {
                t.Logf("Memory limit reached at iteration %d", i)
                break
            }
            defer resp.Body.Close()
            
            // Read response
            _, _ = io.ReadAll(resp.Body)
        }
    })
}
```

### 6. Chaos Tests (Failure Modes)

**Coverage Target**: All failure scenarios

**Files**:
- `chaos_test.go` - Chaos engineering tests

**Required Tests**:
```go
func TestRandomFailures(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Randomly fail
        if rand.Float64() < 0.3 {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        
        // Random delays
        delay := time.Duration(rand.Intn(1000)) * time.Millisecond
        time.Sleep(delay)
        
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok"}`))
    }))
    defer server.Close()
    
    const requests = 1000
    var successCount atomic.Int64
    var wg sync.WaitGroup
    
    for i := 0; i < requests; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            resp, err := gocurl.Request("curl "+server.URL, nil)
            if err == nil && resp.StatusCode == 200 {
                successCount.Add(1)
                resp.Body.Close()
            }
        }()
    }
    
    wg.Wait()
    
    successRate := float64(successCount.Load()) / float64(requests) * 100
    t.Logf("Success rate under chaos: %.2f%%", successRate)
    
    // Should handle failures gracefully
    if successRate < 60.0 {
        t.Errorf("Too many failures: %.2f%%", 100.0-successRate)
    }
}

func TestSlowResponses(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Simulate slow response
        time.Sleep(5 * time.Second)
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()
    
    cmd := fmt.Sprintf("curl --max-time 10 %s", server.URL)
    
    start := time.Now()
    resp, err := gocurl.Request(cmd, nil)
    duration := time.Since(start)
    
    if err != nil {
        t.Error("Should handle slow responses")
    }
    
    if resp != nil {
        resp.Body.Close()
    }
    
    t.Logf("Slow response handled in %v", duration)
}

func TestPartialResponses(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"incomplete":`))
        // Simulate connection drop - don't close properly
    }))
    defer server.Close()
    
    resp, err := gocurl.Request("curl "+server.URL, nil)
    if err == nil && resp != nil {
        body, readErr := io.ReadAll(resp.Body)
        t.Logf("Partial response: %s", body)
        t.Logf("Read error: %v", readErr)
        resp.Body.Close()
    }
}
```

### 7. Fuzz Tests (Input Validation)

**Coverage Target**: All parsers

**Files**:
- `fuzz_test.go` - Fuzzing tests

**Required Tests**:
```go
func FuzzCommandParser(f *testing.F) {
    // Seed corpus
    seeds := []string{
        "curl https://example.com",
        "curl -X POST -d 'data' https://example.com",
        `curl -H "Auth: Bearer token" https://api.example.com`,
        "curl --invalid-flag https://example.com",
        "",
        "   ",
        "\n\n\n",
    }
    
    for _, seed := range seeds {
        f.Add(seed)
    }
    
    f.Fuzz(func(t *testing.T, input string) {
        // Should not crash
        _, _ = gocurl.Request(input, nil)
    })
}

func FuzzVariableSubstitution(f *testing.F) {
    seeds := []string{
        "${var}",
        "$var",
        "\\${var}",
        "${",
        "$",
        "${var${nested}}",
    }
    
    for _, seed := range seeds {
        f.Add(seed)
    }
    
    f.Fuzz(func(t *testing.T, input string) {
        vars := gocurl.Variables{"var": "value", "nested": "test"}
        _, _ = ExpandVariables(input, vars)
    })
}
```

## CI/CD Integration

### GitHub Actions Workflow

```yaml
name: Comprehensive Testing

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Unit Tests
        run: go test -v -coverprofile=coverage.txt ./...
      
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.txt
  
  race-detection:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      
      - name: Race Tests
        run: go test -race -v -timeout 30m ./...
      
      - name: Concurrent Stress
        run: go test -race -run=Concurrent -v ./...
  
  benchmarks:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      
      - name: Run Benchmarks
        run: go test -bench=. -benchmem -run=^$ ./...
      
      - name: Check Regression
        run: go test -run=TestBenchmarkRegression ./...
  
  load-tests:
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      
      - name: Short Load Test
        run: go test -run=TestSustainedLoad -timeout 10m ./...
  
  fuzz-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      
      - name: Fuzz Tests
        run: go test -fuzz=. -fuzztime=1m ./...

  nightly-soak:
    runs-on: ubuntu-latest
    if: github.event.schedule == '0 0 * * *'
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      
      - name: 24-Hour Soak Test
        run: go test -run=TestSoakTest -timeout 25h ./...
      
      - name: Stress Test
        run: go test -run=TestFindBreakingPoint -timeout 2h ./...
```

## Testing Checklist

### Before Each PR

- [ ] All unit tests pass
- [ ] All tests pass with `-race` flag
- [ ] Benchmarks don't regress >5%
- [ ] Code coverage maintained/improved
- [ ] New features have tests
- [ ] Fuzz tests pass (1 minute)

### Before Each Release

- [ ] Full test suite passes (including -race)
- [ ] Load tests pass (10k req/s for 1 hour)
- [ ] Stress tests pass (breaking point found)
- [ ] Chaos tests pass (failure handling verified)
- [ ] Fuzz tests pass (100M+ iterations)
- [ ] Benchmark regression check passes
- [ ] Soak test passes (24 hours)
- [ ] Zero race conditions detected
- [ ] Memory leak analysis clean

### Continuous Monitoring

- [ ] Daily race detection runs
- [ ] Weekly load tests
- [ ] Monthly soak tests
- [ ] Continuous benchmark tracking
- [ ] Performance regression alerts
- [ ] Memory profile monitoring

## Success Criteria

### Performance

- **Zero allocations** on request construction (verified via benchmarks)
- **< 1μs overhead** vs raw net/http
- **10k+ req/s** sustained throughput
- **100k+ concurrent** requests without degradation

### Reliability

- **Zero race conditions** (all tests pass with -race)
- **Zero panics** in production scenarios
- **< 1% error rate** under load
- **Graceful degradation** at 10x normal load

### Robustness

- **100M+ fuzz iterations** without crashes
- **24-hour soak test** with no memory growth
- **Breaking point** documented and tested
- **Chaos scenarios** handled correctly

---

**Testing Motto**: "If it's not tested with -race under concurrent load, it's not production-ready."
