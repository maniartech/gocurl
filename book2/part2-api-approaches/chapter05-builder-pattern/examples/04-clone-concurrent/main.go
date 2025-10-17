package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/options"
)

// Example 4: Clone for Concurrent Requests
// Demonstrates safe concurrent request execution using Clone()

func main() {
	fmt.Println("Example 4: Clone for Concurrent Requests")
	fmt.Println("========================================")
	fmt.Println()

	// Create base options with common configuration
	baseOpts := options.NewRequestOptions("https://api.github.com/users/golang")
	baseOpts.Method = "GET"
	baseOpts.SetHeader("User-Agent", "GoCurl-Example/1.0")
	baseOpts.SetHeader("Accept", "application/json")

	fmt.Println("Base configuration:")
	fmt.Println("  URL: https://api.github.com/users/<username>")
	fmt.Println("  Method: GET")
	fmt.Println("  Headers: User-Agent, Accept")
	fmt.Println()

	// GitHub users to fetch
	users := []string{"golang", "microsoft", "google", "facebook", "apple"}

	fmt.Println("Fetching", len(users), "users concurrently...")
	fmt.Println()

	// Execute concurrent requests
	ctx := context.Background()
	var wg sync.WaitGroup

	for _, username := range users {
		wg.Add(1)

		// Launch goroutine for each user
		go func(user string) {
			defer wg.Done()

			// ✅ SAFE: Clone before modification
			opts := baseOpts.Clone()
			opts.URL = "https://api.github.com/users/" + user
			opts.AddHeader("X-Request-ID", "req-"+user)

			// Execute request
			resp, err := gocurl.Execute(ctx, opts)
			if err != nil {
				fmt.Printf("❌ %s: Error - %v\n", user, err)
				return
			}
			defer resp.Body.Close()

			fmt.Printf("✅ %s: HTTP %d\n", user, resp.StatusCode)
		}(username)
	}

	// Wait for all requests to complete
	wg.Wait()

	fmt.Println()
	fmt.Println("All requests completed!")
	fmt.Println()

	// Demonstrate UNSAFE pattern (commented out)
	fmt.Println("⚠️  UNSAFE Pattern (DO NOT USE):")
	fmt.Println()
	fmt.Println("// ❌ RACE CONDITION: Concurrent map writes")
	fmt.Println("opts := options.NewRequestOptions(\"https://api.example.com\")")
	fmt.Println("go opts.AddHeader(\"X-ID\", \"1\")  // UNSAFE!")
	fmt.Println("go opts.AddHeader(\"X-ID\", \"2\")  // UNSAFE!")
	fmt.Println()

	fmt.Println("✅ SAFE Pattern:")
	fmt.Println()
	fmt.Println("opts := baseOpts.Clone()  // Clone before modification")
	fmt.Println("opts.AddHeader(\"X-ID\", \"unique\")")
	fmt.Println()

	fmt.Println("💡 Key Learnings:")
	fmt.Println("   • Clone() creates independent copy")
	fmt.Println("   • Safe for concurrent modifications")
	fmt.Println("   • Deep copies Headers, Form, QueryParams")
	fmt.Println("   • Use for request templates")
	fmt.Println()

	fmt.Println("🔍 Testing for Race Conditions:")
	fmt.Println("   Run with: go run -race main.go")
	fmt.Println("   This example will pass race detection ✅")
}
