// Example 3: JSON Request
// Demonstrates automatic JSON unmarshaling with CurlJSON function.

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/gocurl"
)

// HTTPBinResponse represents the JSON structure returned by httpbin.org
type HTTPBinResponse struct {
	Args    map[string]string `json:"args"`
	Headers map[string]string `json:"headers"`
	Origin  string            `json:"origin"`
	URL     string            `json:"url"`
}

func main() {
	fmt.Println("📦 JSON Request Example\n")

	ctx := context.Background()

	// Use CurlJSON to automatically unmarshal the response
	var result HTTPBinResponse
	resp, err := gocurl.CurlJSON(ctx, &result, "https://httpbin.org/get")

	if err != nil {
		log.Fatalf("❌ JSON request failed: %v", err)
	}
	defer resp.Body.Close()

	// The response is automatically unmarshaled into our struct
	fmt.Printf("✅ JSON Response Parsed Successfully!\n\n")
	fmt.Printf("📍 Your IP Address: %s\n", result.Origin)
	fmt.Printf("🔗 Request URL: %s\n", result.URL)

	fmt.Printf("\n📋 Request Headers:\n")
	for key, value := range result.Headers {
		fmt.Printf("  %s: %s\n", key, value)
	}

	// Demonstrate type safety - we have structured data
	fmt.Printf("\n🎯 Type-Safe Access:\n")
	fmt.Printf("  Origin type: %T\n", result.Origin)
	fmt.Printf("  URL type: %T\n", result.URL)
	fmt.Printf("  Headers type: %T\n", result.Headers)

	fmt.Println("\n💡 Notice: No manual JSON parsing needed!")
}
