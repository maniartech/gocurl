// Package main demonstrates a simple GET request using GoCurl.
// This example queries the GitHub API to fetch user information.
//
// Run: go run simple_get.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/gocurl"
)

func main() {
	ctx := context.Background()

	// Simple GET request - no authentication needed
	body, resp, err := gocurl.CurlString(ctx,
		"https://api.github.com/users/octocat")

	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("\nBody:\n%s\n", body)
}
