package main

import (
	"context"
	"fmt"
	"io"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/options"
)

// Example 1: Basic Builder
// Demonstrates the simplest use of RequestOptionsBuilder

func main() {
	fmt.Println("Example 1: Basic Builder Pattern")
	fmt.Println("=================================")
	fmt.Println()

	// Create a new builder
	builder := options.NewRequestOptionsBuilder()

	// Configure the request using fluent API
	opts := builder.
		SetURL("https://api.github.com/zen").
		SetMethod("GET").
		Build()

	// Execute the request
	ctx := context.Background()
	resp, err := gocurl.Execute(ctx, opts)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	// Read and display response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading body:", err)
		return
	}

	fmt.Println("Status Code:", resp.StatusCode)
	fmt.Println("Response Body:", string(body))
	fmt.Println()

	// Comparison: Curl-syntax equivalent
	fmt.Println("Curl-syntax equivalent:")
	fmt.Println("resp, err := gocurl.Curl(ctx, \"https://api.github.com/zen\")")
	fmt.Println()

	// Advantages of Builder:
	fmt.Println("✅ Builder Pattern Advantages:")
	fmt.Println("   • Type-safe configuration")
	fmt.Println("   • IDE autocompletion")
	fmt.Println("   • Clear, readable code")
	fmt.Println("   • Validation before execution")
	fmt.Println("   • Easy to modify/extend")
}
