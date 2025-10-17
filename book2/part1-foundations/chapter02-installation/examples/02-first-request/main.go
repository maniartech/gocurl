// Example 2: First Request
// Making your first API request with proper error handling and response parsing.

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/maniartech/gocurl"
)

func main() {
	fmt.Println("🚀 Your First GoCurl Request\n")

	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Make a GET request to httpbin.org (a service for testing HTTP)
	// httpbin.org/get returns information about the request
	body, resp, err := gocurl.CurlString(ctx, "https://httpbin.org/get")

	if err != nil {
		log.Fatalf("❌ Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Examine the response
	fmt.Printf("📊 Response Details:\n")
	fmt.Printf("  Status Code: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("  Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("  Content-Length: %s bytes\n", resp.Header.Get("Content-Length"))

	// Show response body (formatted JSON)
	fmt.Printf("\n📄 Response Body:\n%s\n", body)

	// Success indicators
	if resp.StatusCode == 200 {
		fmt.Println("\n✅ Request completed successfully!")
	}
}
