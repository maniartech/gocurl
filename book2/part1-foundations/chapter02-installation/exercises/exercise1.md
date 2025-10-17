# Exercise 1: Environment Setup Checker

**Difficulty:** ⭐ Beginner
**Time:** 20-30 minutes

Build a comprehensive environment verification tool that checks:
- Go version compatibility
- GoCurl installation
- Network connectivity
- HTTPS support
- JSON parsing
- Environment variable expansion
- Context timeout support

This exercise reinforces Chapter 2's installation and verification concepts.

## Objectives

- Verify Go installation (version 1.21+)
- Confirm GoCurl is properly installed
- Test network connectivity
- Validate HTTPS/TLS support
- Test JSON unmarshaling
- Verify environment variable expansion
- Check context timeout functionality

## Starter Code

Create a file `main.go` with the following structure:

```go
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/maniartech/gocurl"
)

// TestResult holds the result of a verification test
type TestResult struct {
	Name    string
	Passed  bool
	Message string
	Details string
}

func main() {
	fmt.Println("🔍 GoCurl Environment Setup Checker")
	fmt.Println("====================================\n")

	// Run all verification tests
	results := []TestResult{
		checkGoVersion(),
		checkGoCurlImport(),
		checkNetworkConnectivity(),
		checkHTTPSSupport(),
		checkJSONParsing(),
		checkEnvironmentVariables(),
		checkContextSupport(),
	}

	// Display results
	displayResults(results)

	// Summary
	displaySummary(results)
}

// TODO 1: Implement checkGoVersion
// Check if Go version is 1.21 or higher
// Hint: Use runtime.Version() and parse the version string
func checkGoVersion() TestResult {
	version := runtime.Version()

	// TODO: Parse version and check >= 1.21
	// Return TestResult with appropriate Passed, Message, and Details

	return TestResult{
		Name:    "Go Version Check",
		Passed:  false,
		Message: "TODO: Implement version check",
		Details: fmt.Sprintf("Current: %s", version),
	}
}

// TODO 2: Implement checkGoCurlImport
// Verify GoCurl can be imported and used
// Hint: Make a simple HEAD request to verify library works
func checkGoCurlImport() TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// TODO: Try a simple gocurl.Curl() call
	// If successful, library is working

	return TestResult{
		Name:    "GoCurl Import",
		Passed:  false,
		Message: "TODO: Test GoCurl import",
	}
}

// TODO 3: Implement checkNetworkConnectivity
// Test basic network connectivity
// Hint: GET request to https://httpbin.org/status/200
func checkNetworkConnectivity() TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TODO: Make request to httpbin.org
	// Check for successful response (status 200)

	return TestResult{
		Name:    "Network Connectivity",
		Passed:  false,
		Message: "TODO: Test network",
	}
}

// TODO 4: Implement checkHTTPSSupport
// Verify HTTPS/TLS works correctly
// Hint: Request https://api.github.com/zen
func checkHTTPSSupport() TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TODO: Make HTTPS request
	// Verify TLS handshake succeeds

	return TestResult{
		Name:    "HTTPS/TLS Support",
		Passed:  false,
		Message: "TODO: Test HTTPS",
	}
}

// TODO 5: Implement checkJSONParsing
// Test JSON unmarshaling capability
// Hint: Use CurlJSON with https://httpbin.org/json
func checkJSONParsing() TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TODO: Define a struct for the response
	// Use gocurl.CurlJSON() to parse

	return TestResult{
		Name:    "JSON Parsing",
		Passed:  false,
		Message: "TODO: Test JSON",
	}
}

// TODO 6: Implement checkEnvironmentVariables
// Test environment variable expansion
// Hint: Set TEST_VAR, then use ${TEST_VAR} in curl command
func checkEnvironmentVariables() TestResult {
	// Set test environment variable
	os.Setenv("GOCURL_TEST_VAR", "test_value_123")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TODO: Make request using ${GOCURL_TEST_VAR}
	// Verify variable was expanded (check response)

	return TestResult{
		Name:    "Environment Variables",
		Passed:  false,
		Message: "TODO: Test env vars",
	}
}

// TODO 7: Implement checkContextSupport
// Verify context timeout works
// Hint: Short timeout (1s) with slow endpoint (delay/5)
func checkContextSupport() TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// TODO: Request endpoint with 5 second delay
	// Expect timeout error
	// If timeout occurs correctly, test passes

	return TestResult{
		Name:    "Context Timeout",
		Passed:  false,
		Message: "TODO: Test context",
	}
}

// Display individual test results
func displayResults(results []TestResult) {
	for i, result := range results {
		status := "❌"
		if result.Passed {
			status = "✅"
		}

		fmt.Printf("%d. %s %s\n", i+1, status, result.Name)
		fmt.Printf("   %s\n", result.Message)

		if result.Details != "" {
			fmt.Printf("   Details: %s\n", result.Details)
		}
		fmt.Println()
	}
}

// Display summary of all tests
func displaySummary(results []TestResult) {
	passed := 0
	total := len(results)

	for _, result := range results {
		if result.Passed {
			passed++
		}
	}

	fmt.Println("====================================")
	fmt.Printf("Summary: %d/%d tests passed\n", passed, total)

	if passed == total {
		fmt.Println("🎉 All tests passed! Your environment is ready.")
	} else {
		fmt.Printf("⚠️  %d test(s) failed. Review and fix issues above.\n", total-passed)
	}
}
```

## Implementation Guide

### Test 1: Go Version

```go
version := runtime.Version()
// Parse "go1.21.0" -> major=1, minor=21
// Check if version >= 1.21
```

### Test 2: GoCurl Import

```go
resp, err := gocurl.Curl(ctx, "-I", "https://httpbin.org/status/200")
if err == nil {
    // Import successful
}
```

### Test 3: Network

```go
_, resp, err := gocurl.CurlString(ctx, "https://httpbin.org/status/200")
if err == nil && resp.StatusCode == 200 {
    // Network OK
}
```

### Test 4: HTTPS

```go
body, resp, err := gocurl.CurlString(ctx, "https://api.github.com/zen")
if err == nil && len(body) > 0 {
    // HTTPS works
}
```

### Test 5: JSON

```go
var result map[string]interface{}
resp, err := gocurl.CurlJSON(ctx, &result, "https://httpbin.org/json")
if err == nil && len(result) > 0 {
    // JSON parsing works
}
```

### Test 6: Environment Variables

```go
body, resp, err := gocurl.CurlString(ctx,
    "https://httpbin.org/anything?test=${GOCURL_TEST_VAR}")
if err == nil && strings.Contains(body, "test_value_123") {
    // Variable expanded
}
```

### Test 7: Context

```go
_, _, err := gocurl.CurlString(ctx, "https://httpbin.org/delay/5")
if ctx.Err() == context.DeadlineExceeded {
    // Timeout works correctly
}
```

## Expected Output

```
🔍 GoCurl Environment Setup Checker
====================================

1. ✅ Go Version Check
   Go version 1.21.5 (meets requirement >= 1.21)
   Details: Current: go1.21.5

2. ✅ GoCurl Import
   GoCurl library imported and functional

3. ✅ Network Connectivity
   Successfully connected to httpbin.org

4. ✅ HTTPS/TLS Support
   HTTPS requests working correctly

5. ✅ JSON Parsing
   JSON unmarshaling successful

6. ✅ Environment Variables
   Variable expansion working (found: test_value_123)

7. ✅ Context Timeout
   Context timeout handled correctly

====================================
Summary: 7/7 tests passed
🎉 All tests passed! Your environment is ready.
```

## Self-Check Criteria

- ✅ All 7 tests pass
- ✅ Proper error handling in each test
- ✅ Clear pass/fail messages
- ✅ Detailed information in results
- ✅ Correct timeout values used

## Bonus Challenges

1. Add color output (green/red for pass/fail)
2. Add timing for each test
3. Export results to JSON file
4. Add retry logic for flaky network tests
5. Create a web dashboard for results

## Troubleshooting

**Issue:** Network tests fail
**Solution:** Check firewall, proxy settings, internet connection

**Issue:** JSON test fails
**Solution:** Verify httpbin.org is accessible

**Issue:** Go version check fails
**Solution:** Update Go to 1.21 or higher

**Issue:** Context timeout doesn't work
**Solution:** Check that delay endpoint is used (delay/5)
