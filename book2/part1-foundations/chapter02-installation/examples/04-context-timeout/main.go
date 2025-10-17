// Example 4: Context Timeout
// Demonstrates using context for timeout control and graceful error handling.

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/maniartech/gocurl"
)

func main() {
	fmt.Println("⏱️  Context Timeout Example\n")

	// Example 1: Successful request with reasonable timeout
	fmt.Println("1️⃣  Request with 10-second timeout (should succeed)...")
	successfulRequest()

	// Example 2: Request that times out
	fmt.Println("\n2️⃣  Request with 1-millisecond timeout (will timeout)...")
	timeoutRequest()

	// Example 3: Using context cancellation
	fmt.Println("\n3️⃣  Request with manual cancellation...")
	cancellableRequest()
}

func successfulRequest() {
	// Create context with reasonable timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	body, resp, err := gocurl.CurlString(ctx, "https://httpbin.org/delay/1")

	if err != nil {
		log.Printf("❌ Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("✅ Request succeeded (took ~1 second)\n")
	fmt.Printf("📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("📏 Response size: %d bytes\n", len(body))
}

func timeoutRequest() {
	// Intentionally very short timeout to demonstrate timeout behavior
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, _, err := gocurl.CurlString(ctx, "https://httpbin.org/delay/5")
	elapsed := time.Since(start)

	if err != nil {
		// Check if it's a context timeout error
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Printf("⏰ Request timed out as expected after %v\n", elapsed)
			fmt.Printf("💡 Error: %v\n", err)
		} else {
			fmt.Printf("❌ Different error: %v\n", err)
		}
		return
	}

	fmt.Println("⚠️  Request completed (unexpected)")
}

func cancellableRequest() {
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		fmt.Println("🛑 Cancelling request...")
		cancel()
	}()

	start := time.Now()
	_, _, err := gocurl.CurlString(ctx, "https://httpbin.org/delay/5")
	elapsed := time.Since(start)

	if err != nil {
		if ctx.Err() == context.Canceled {
			fmt.Printf("🚫 Request cancelled after %v\n", elapsed)
		} else {
			fmt.Printf("❌ Error: %v\n", err)
		}
		return
	}

	fmt.Println("⚠️  Request completed (unexpected)")
}
