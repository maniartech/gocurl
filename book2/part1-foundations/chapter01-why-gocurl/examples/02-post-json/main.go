// Package main demonstrates a POST request with JSON data using GoCurl.
// This example uses JSONPlaceholder, a free fake API for testing.
//
// Run: go run main.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/gocurl"
)

func main() {
	ctx := context.Background()

	// POST request with JSON data
	body, resp, err := gocurl.CurlString(ctx,
		"-X", "POST",
		"-H", "Content-Type: application/json",
		"-d", `{"title": "My Post", "body": "This is the content", "userId": 1}`,
		"https://jsonplaceholder.typicode.com/posts")

	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("\nResponse:\n%s\n", body)
}
