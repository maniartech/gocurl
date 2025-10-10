package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/maniartech/gocurl"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Build command from args (skip program name)
	args := os.Args[1:]

	// Join arguments - gocurl supports both formats:
	// 1. Individual args: gocurl -X POST -d data https://example.com
	// 2. Single string: gocurl "curl -X POST https://example.com"

	// Auto-populate variables from environment
	vars := envToVariables()

	// Execute request
	resp, err := gocurl.Request(args, vars)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	defer resp.Body.Close()

	// Print response
	if err := printResponse(resp); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("gocurl - Execute curl commands in Go")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gocurl [curl options] <URL>")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  gocurl https://api.example.com/data")
	fmt.Println("  gocurl -X POST -d 'key=value' https://api.example.com/data")
	fmt.Println("  gocurl -H 'Authorization: Bearer $TOKEN' https://api.example.com/data")
	fmt.Println()
	fmt.Println("Environment variables can be used with $VAR or ${VAR} syntax")
}

func envToVariables() gocurl.Variables {
	vars := make(gocurl.Variables)

	// Get all environment variables
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			vars[parts[0]] = parts[1]
		}
	}

	return vars
}

func printResponse(resp *gocurl.Response) error {
	// Get response body
	body, err := resp.Bytes()
	if err != nil {
		return err
	}

	// Check if response is JSON
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		// Pretty print JSON
		var data interface{}
		if err := json.Unmarshal(body, &data); err == nil {
			formatted, err := json.MarshalIndent(data, "", "  ")
			if err == nil {
				fmt.Println(string(formatted))
				return nil
			}
		}
	}

	// Print raw body
	fmt.Print(string(body))
	return nil
}
