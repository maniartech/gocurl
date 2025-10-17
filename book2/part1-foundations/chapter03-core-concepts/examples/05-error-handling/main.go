// Example 5: Error Handling
// Demonstrates comprehensive error handling patterns for HTTP requests.

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/maniartech/gocurl"
)

func main() {
	fmt.Println("⚠️  Error Handling Demonstration\n")

	// Pattern 1: HTTP status code errors
	pattern1_StatusCodeErrors()

	// Pattern 2: Timeout errors
	fmt.Println()
	pattern2_TimeoutErrors()

	// Pattern 3: Network errors
	fmt.Println()
	pattern3_NetworkErrors()

	// Pattern 4: Proper error checking
	fmt.Println()
	pattern4_ProperErrorChecking()

	// Pattern 5: Retry logic
	fmt.Println()
	pattern5_RetryLogic()
}

func pattern1_StatusCodeErrors() {
	fmt.Println("1️⃣  HTTP Status Code Errors")

	ctx := context.Background()

	// Test different status codes
	statusCodes := []int{200, 404, 500}

	for _, code := range statusCodes {
		url := fmt.Sprintf("https://httpbin.org/status/%d", code)
		body, resp, err := gocurl.CurlString(ctx, url)

		fmt.Printf("\n   📍 Testing status code: %d\n", code)

		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		// Check status code explicitly
		if resp.StatusCode >= 400 {
			fmt.Printf("   ⚠️  HTTP Error: %d %s\n", resp.StatusCode, resp.Status)
			fmt.Printf("   📦 Response body: %s\n", body)
		} else {
			fmt.Printf("   ✅ Success: %d %s\n", resp.StatusCode, resp.Status)
		}
	}

	fmt.Println("\n   💡 Always check resp.StatusCode for HTTP errors")
}

func pattern2_TimeoutErrors() {
	fmt.Println("2️⃣  Timeout Errors")

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	fmt.Println("   ⏰ Timeout: 500ms, Delay: 2 seconds")
	fmt.Println("   🚀 Making request...")

	start := time.Now()
	_, _, err := gocurl.CurlString(ctx, "https://httpbin.org/delay/2")
	elapsed := time.Since(start)

	if err != nil {
		// Check for context deadline exceeded
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("   ⏰ Timeout after %v (as expected)\n", elapsed)
		} else if ctx.Err() == context.DeadlineExceeded {
			fmt.Printf("   ⏰ Context deadline exceeded after %v\n", elapsed)
		} else {
			fmt.Printf("   ❌ Other error: %v\n", err)
		}
	}

	fmt.Println("\n   💡 Use errors.Is() or ctx.Err() to check timeouts")
}

func pattern3_NetworkErrors() {
	fmt.Println("3️⃣  Network Errors")

	ctx := context.Background()

	// Test invalid hostname
	fmt.Println("   📍 Testing invalid hostname...")
	_, _, err := gocurl.CurlString(ctx, "https://this-domain-does-not-exist-12345.com")

	if err != nil {
		fmt.Printf("   ❌ Network error: %v\n", err)
		fmt.Printf("   💡 Check DNS resolution and connectivity\n")
	}

	// Test connection refused (invalid port)
	fmt.Println("\n   📍 Testing connection refused...")
	_, _, err = gocurl.CurlString(ctx, "https://httpbin.org:9999")

	if err != nil {
		fmt.Printf("   ❌ Connection error: %v\n", err)
		fmt.Printf("   💡 Port may be closed or unreachable\n")
	}

	fmt.Println("\n   💡 Network errors indicate connectivity issues")
}

func pattern4_ProperErrorChecking() {
	fmt.Println("4️⃣  Proper Error Checking Pattern")

	ctx := context.Background()

	body, resp, err := gocurl.CurlString(ctx, "https://httpbin.org/status/403")

	// Step 1: Check for request error
	if err != nil {
		log.Printf("   ❌ Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	// Step 2: Check HTTP status code
	if resp.StatusCode >= 400 {
		fmt.Printf("   ⚠️  HTTP Error: %d %s\n", resp.StatusCode, resp.Status)
		fmt.Printf("   📦 Error body: %s\n", body)

		// Handle specific status codes
		switch resp.StatusCode {
		case http.StatusNotFound:
			fmt.Println("   💡 Resource not found")
		case http.StatusUnauthorized:
			fmt.Println("   💡 Authentication required")
		case http.StatusForbidden:
			fmt.Println("   💡 Access forbidden")
		case http.StatusTooManyRequests:
			fmt.Println("   💡 Rate limit exceeded")
		case http.StatusInternalServerError:
			fmt.Println("   💡 Server error")
		default:
			fmt.Printf("   💡 HTTP error code: %d\n", resp.StatusCode)
		}
		return
	}

	// Step 3: Process successful response
	fmt.Printf("   ✅ Success: %d %s\n", resp.StatusCode, resp.Status)

	fmt.Println("\n   ✅ Best Practice Pattern:")
	fmt.Println("      1. Check err != nil first")
	fmt.Println("      2. Defer resp.Body.Close()")
	fmt.Println("      3. Check resp.StatusCode")
	fmt.Println("      4. Process successful response")
}

func pattern5_RetryLogic() {
	fmt.Println("5️⃣  Retry Logic Pattern")

	ctx := context.Background()
	maxRetries := 3
	retryDelay := 1 * time.Second

	url := "https://httpbin.org/status/500" // Simulating server error

	var lastErr error
	var resp *http.Response

	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("\n   🔄 Attempt %d/%d\n", attempt, maxRetries)

		_, resp, lastErr = gocurl.CurlString(ctx, url)

		if lastErr != nil {
			fmt.Printf("   ❌ Request error: %v\n", lastErr)
			if attempt < maxRetries {
				fmt.Printf("   ⏳ Waiting %v before retry...\n", retryDelay)
				time.Sleep(retryDelay)
			}
			continue
		}

		// Check status code
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			fmt.Printf("   ⚠️  Server error: %d\n", resp.StatusCode)
			if attempt < maxRetries {
				fmt.Printf("   ⏳ Waiting %v before retry...\n", retryDelay)
				time.Sleep(retryDelay)
			}
			continue
		}

		// Success
		defer resp.Body.Close()
		fmt.Printf("   ✅ Success on attempt %d\n", attempt)
		return
	}

	// All retries failed
	if resp != nil {
		resp.Body.Close()
		fmt.Printf("\n   ❌ All %d attempts failed\n", maxRetries)
		fmt.Printf("   📊 Last status: %d\n", resp.StatusCode)
	} else {
		fmt.Printf("\n   ❌ All %d attempts failed\n", maxRetries)
		fmt.Printf("   ❌ Last error: %v\n", lastErr)
	}

	fmt.Println("\n   💡 Retry transient errors (5xx, timeouts)")
	fmt.Println("   💡 Don't retry client errors (4xx)")
}
