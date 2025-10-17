package main

import (
	"context"
	"fmt"
	"io"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/options"
)

// Example 3: Authentication
// Demonstrates Bearer Token and Basic Auth

func main() {
	fmt.Println("Example 3: Authentication")
	fmt.Println("=========================")
	fmt.Println()

	// Example 1: Bearer Token Authentication
	fmt.Println("1️⃣  Bearer Token Authentication")
	fmt.Println()

	bearerOpts := options.NewRequestOptionsBuilder().
		SetURL("https://httpbin.org/bearer").
		SetMethod("GET").
		SetBearerToken("your-secret-token-12345").
		Build()

	ctx := context.Background()
	resp, err := gocurl.Execute(ctx, bearerOpts)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("Response:", string(body))
	fmt.Println()

	// Example 2: Basic Authentication
	fmt.Println("2️⃣  Basic Authentication")
	fmt.Println()

	basicOpts := options.NewRequestOptionsBuilder().
		SetURL("https://httpbin.org/basic-auth/alice/secret123").
		SetMethod("GET").
		SetBasicAuth("alice", "secret123").
		Build()

	resp2, err := gocurl.Execute(ctx, basicOpts)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)
	fmt.Println("Status:", resp2.StatusCode)
	fmt.Println("Response:", string(body2))
	fmt.Println()

	// Example 3: Using BearerAuth() convenience method
	fmt.Println("3️⃣  Using BearerAuth() Convenience Method")
	fmt.Println()

	convOpts := options.NewRequestOptionsBuilder().
		SetURL("https://httpbin.org/headers").
		SetMethod("GET").
		BearerAuth("my-token-67890"). // Adds "Authorization: Bearer <token>" header
		Build()

	resp3, err := gocurl.Execute(ctx, convOpts)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp3.Body.Close()

	body3, _ := io.ReadAll(resp3.Body)
	fmt.Println("Status:", resp3.StatusCode)
	fmt.Println("Headers sent (excerpt):", string(body3[:200])+"...")
	fmt.Println()

	// Comparison
	fmt.Println("Curl-syntax equivalents:")
	fmt.Println()
	fmt.Println("Bearer Token:")
	fmt.Println(`resp, err := gocurl.Curl(ctx, "-H", "Authorization: Bearer token", "https://httpbin.org/bearer")`)
	fmt.Println()
	fmt.Println("Basic Auth:")
	fmt.Println(`resp, err := gocurl.Curl(ctx, "-u", "alice:secret123", "https://httpbin.org/basic-auth/alice/secret123")`)
	fmt.Println()

	fmt.Println("💡 Key Learnings:")
	fmt.Println("   • SetBearerToken() for OAuth 2.0 tokens")
	fmt.Println("   • SetBasicAuth() for username/password")
	fmt.Println("   • BearerAuth() convenience method")
	fmt.Println("   • Authentication automatically encoded")
	fmt.Println()

	fmt.Println("⚠️  Security Note:")
	fmt.Println("   • Always use HTTPS for authentication")
	fmt.Println("   • Store tokens securely (not in code)")
	fmt.Println("   • Use environment variables for secrets")
}
