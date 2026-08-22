// Example 1: Verify Installation
// This example verifies that gocurl is installed correctly by making a simple request.

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/gocurl"
)

func main() {
	fmt.Println("🔍 Verifying GoCurl Installation...")

	// Create a basic context
	ctx := context.Background()

	// Make a simple GET request to GitHub's Zen API
	// This returns a random quote from GitHub's design principles
	body, resp, err := gocurl.CurlString(ctx, "https://api.github.com/zen")

	if err != nil {
		log.Fatalf("❌ Installation verification failed: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != 200 {
		log.Fatalf("❌ Unexpected status code: %d", resp.StatusCode)
	}

	// Success!
	fmt.Println("✅ GoCurl is installed and working correctly!")
	fmt.Printf("📡 HTTP Status: %d\n", resp.StatusCode)
	fmt.Printf("💬 GitHub Zen Quote: %s\n\n", body)
	fmt.Println("🎉 You're ready to build API clients with GoCurl!")
}
