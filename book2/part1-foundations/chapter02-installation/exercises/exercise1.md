// Exercise 1: Environment Setup Checker
// Difficulty: ⭐ Beginner
// Time: 20-30 minutes
//
// Build a comprehensive environment verification tool that checks:
// - Go version
// - GoCurl installation
// - Network connectivity
// - HTTPS support
// - JSON parsing
// - Environment variable expansion
//
// This exercise reinforces Chapter 2's installation and verification concepts.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
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

// TODO: Implement checkGoVersion
// Hint: Use runtime.Version()
func checkGoVersion() TestResult {
	// YOUR CODE HERE
	return TestResult{
		Name:    "Go Version Check",
		Passed:  false,
		Message: "Not implemented",
	}
}

// TODO: Implement checkGoCurlImport
// Hint: Try a simple GoCurl function call
func checkGoCurlImport() TestResult {
	// YOUR CODE HERE
	return TestResult{
		Name:    "GoCurl Import",
		Passed:  false,
		Message: "Not implemented",
	}
}

// TODO: Implement checkNetworkConnectivity
// Hint: Make a simple GET request to httpbin.org
func checkNetworkConnectivity() TestResult {
	// YOUR CODE HERE
	return TestResult{
		Name:    "Network Connectivity",
		Passed:  false,
		Message: "Not implemented",
	}
}

// TODO: Implement checkHTTPSSupport
// Hint: Make a request to https://api.github.com/zen
func checkHTTPSSupport() TestResult {
	// YOUR CODE HERE
	return TestResult{
		Name:    "HTTPS Support",
		Passed:  false,
		Message: "Not implemented",
	}
}

// TODO: Implement checkJSONParsing
// Hint: Use CurlJSON with httpbin.org/json
func checkJSONParsing() TestResult {
	// YOUR CODE HERE
	return TestResult{
		Name:    "JSON Parsing",
		Passed:  false,
		Message: "Not implemented",
	}
}

// TODO: Implement checkEnvironmentVariables
// Hint: Set a test variable, then use it in a curl command
func checkEnvironmentVariables() TestResult {
	// YOUR CODE HERE
	return TestResult{
		Name:    "Environment Variables",
		Passed:  false,
		Message: "Not implemented",
	}
}

// TODO: Implement checkContextSupport
// Hint: Create a context with timeout and verify it works
func checkContextSupport() TestResult {
	// YOUR CODE HERE
	return TestResult{
		Name:    "Context Support",
		Passed:  false,
		Message: "Not implemented",
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
