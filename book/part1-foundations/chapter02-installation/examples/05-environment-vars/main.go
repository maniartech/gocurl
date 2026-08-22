// Example 5: Environment Variables
// Demonstrates using environment variables for configuration and secrets.

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/maniartech/gocurl"
)

func main() {
	fmt.Println("🔐 Environment Variables Example")

	// Example 1: Using environment variables with automatic expansion
	example1_AutomaticExpansion()

	// Example 2: Loading from environment for configuration
	fmt.Println()
	example2_ConfigurationFromEnv()

	// Example 3: Best practices for secret management
	fmt.Println()
	example3_SecretManagement()
}

func example1_AutomaticExpansion() {
	fmt.Println("1️⃣  Automatic environment variable expansion:")

	// Set an environment variable
	os.Setenv("API_BASE_URL", "https://httpbin.org")
	os.Setenv("API_ENDPOINT", "/get")

	ctx := context.Background()

	// GoCurl automatically expands $VARIABLE and ${VARIABLE}
	body, resp, err := gocurl.CurlStringCommand(ctx,
		`curl ${API_BASE_URL}${API_ENDPOINT}`)

	if err != nil {
		log.Printf("❌ Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("✅ Request to: %s%s\n", os.Getenv("API_BASE_URL"), os.Getenv("API_ENDPOINT"))
	fmt.Printf("📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("📏 Response size: %d bytes\n", len(body))
}

func example2_ConfigurationFromEnv() {
	fmt.Println("2️⃣  Loading configuration from environment:")

	// Simulating environment setup
	os.Setenv("REQUEST_TIMEOUT", "30")
	os.Setenv("USER_AGENT", "MyApp/1.0")
	os.Setenv("MAX_RETRIES", "3")

	// Load configuration
	config := struct {
		Timeout   string
		UserAgent string
		Retries   string
	}{
		Timeout:   os.Getenv("REQUEST_TIMEOUT"),
		UserAgent: os.Getenv("USER_AGENT"),
		Retries:   os.Getenv("MAX_RETRIES"),
	}

	fmt.Printf("📋 Configuration loaded:\n")
	fmt.Printf("  Timeout: %s seconds\n", config.Timeout)
	fmt.Printf("  User-Agent: %s\n", config.UserAgent)
	fmt.Printf("  Max Retries: %s\n", config.Retries)

	// Use in request
	ctx := context.Background()
	_, resp, err := gocurl.CurlStringCommand(ctx,
		`curl -H "User-Agent: ${USER_AGENT}" https://httpbin.org/user-agent`)

	if err != nil {
		log.Printf("❌ Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("✅ Request completed with custom User-Agent\n")
}

func example3_SecretManagement() {
	fmt.Println("3️⃣  Best practices for secret management:")

	// Check if API key is set
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  API_KEY not set - using demo mode")
		apiKey = "demo-key-for-testing"
		os.Setenv("API_KEY", apiKey)
	}

	fmt.Println("\n✅ Best Practices Demonstrated:")
	fmt.Println("  1. Check for required environment variables")
	fmt.Println("  2. Provide helpful error messages if missing")
	fmt.Println("  3. Never hard-code secrets in source code")
	fmt.Println("  4. Use .env files (gitignored) for development")
	fmt.Println("  5. Use secret managers in production")

	// Example request with secret
	ctx := context.Background()
	_, resp, err := gocurl.CurlStringCommand(ctx,
		`curl -H "X-API-Key: ${API_KEY}" https://httpbin.org/headers`)

	if err != nil {
		log.Printf("❌ Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("\n✅ Request authenticated with API key from environment\n")
	fmt.Printf("📊 Status: %d\n", resp.StatusCode)
	fmt.Println("\n💡 API key was loaded from environment, not hard-coded!")
}
