// Example 2: Variable Expansion
// Demonstrates environment variable expansion and explicit variable maps.

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/maniartech/gocurl"
)

func main() {
	fmt.Println("🔐 Variable Expansion Demonstration\n")

	// Example 1: Automatic environment expansion
	example1_EnvironmentExpansion()

	// Example 2: Explicit variable maps
	fmt.Println()
	example2_ExplicitVariables()

	// Example 3: Security best practices
	fmt.Println()
	example3_SecureSecrets()

	// Example 4: Multi-environment configuration
	fmt.Println()
	example4_MultiEnvironment()
}

func example1_EnvironmentExpansion() {
	fmt.Println("1️⃣  Automatic Environment Variable Expansion")

	// Set environment variables
	os.Setenv("API_BASE", "https://httpbin.org")
	os.Setenv("API_ENDPOINT", "/get")
	os.Setenv("USER_AGENT", "GoCurl-Example/1.0")

	ctx := context.Background()

	// Both $VAR and ${VAR} syntax work
	_, resp, err := gocurl.CurlStringCommand(ctx,
		`curl -H "User-Agent: ${USER_AGENT}" $API_BASE$API_ENDPOINT`)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Request successful\n")
	fmt.Printf("   📍 URL: %s%s\n", os.Getenv("API_BASE"), os.Getenv("API_ENDPOINT"))
	fmt.Printf("   👤 User-Agent: %s\n", os.Getenv("USER_AGENT"))
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("   💡 Variables automatically expanded from environment\n")
}

func example2_ExplicitVariables() {
	fmt.Println("2️⃣  Explicit Variable Maps (WithVars)")

	// Set an environment variable
	os.Setenv("ENV_TOKEN", "this-will-be-ignored")

	// Define explicit variables
	vars := gocurl.Variables{
		"base_url": "https://httpbin.org",
		"endpoint": "/headers",
		"token":    "explicit-map-value",
	}

	ctx := context.Background()

	// WithVars functions ONLY use the provided map
	resp, err := gocurl.CurlCommandWithVars(ctx, vars,
		`curl -H "X-Token: ${token}" ${base_url}${endpoint}`)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Request successful\n")
	fmt.Printf("   📍 URL: %s%s\n", vars["base_url"], vars["endpoint"])
	fmt.Printf("   🔑 Token used: %s (from map)\n", vars["token"])
	fmt.Printf("   🚫 Environment variable ENV_TOKEN was IGNORED\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
}

func example3_SecureSecrets() {
	fmt.Println("3️⃣  Secure Secret Management")

	// Simulate loading from secure source
	apiKey := loadAPIKeyFromVault()
	os.Setenv("API_KEY", apiKey)

	ctx := context.Background()

	// Use environment variable for secret
	_, resp, err := gocurl.CurlStringCommand(ctx,
		`curl -H "X-API-Key: ${API_KEY}" https://httpbin.org/headers`)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Request authenticated\n")
	fmt.Printf("   🔒 API key loaded from secure vault\n")
	fmt.Printf("   🔑 API key in environment (hidden): %s...\n", maskSecret(apiKey))
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("\n   ✅ Best Practices Demonstrated:\n")
	fmt.Printf("      • Never hard-code secrets\n")
	fmt.Printf("      • Load from environment\n")
	fmt.Printf("      • Use secure vault/manager\n")
	fmt.Printf("      • Mask secrets in logs\n")
}

func example4_MultiEnvironment() {
	fmt.Println("4️⃣  Multi-Environment Configuration")

	// Setup different environments
	environments := map[string]map[string]string{
		"DEV": {
			"API_URL": "https://api.dev.example.com",
			"API_KEY": "dev-key-123",
		},
		"STAGING": {
			"API_URL": "https://api.staging.example.com",
			"API_KEY": "staging-key-456",
		},
		"PROD": {
			"API_URL": "https://api.prod.example.com",
			"API_KEY": "prod-key-789",
		},
	}

	// Select environment
	env := "DEV"
	config := environments[env]

	// Use explicit variables for clarity
	vars := gocurl.Variables{
		"api_url": config["API_URL"],
		"api_key": config["API_KEY"],
	}

	fmt.Printf("   🌍 Environment: %s\n", env)
	fmt.Printf("   📍 URL: %s\n", vars["api_url"])
	fmt.Printf("   🔑 API Key: %s...\n", maskSecret(vars["api_key"]))
	fmt.Printf("\n   💡 This pattern allows easy environment switching\n")
	fmt.Printf("   💡 Each environment has its own configuration\n")
}

// Simulated vault loader
func loadAPIKeyFromVault() string {
	// In production, this would connect to a secure vault
	// (HashiCorp Vault, AWS Secrets Manager, etc.)
	return "secure-api-key-from-vault-xyz123"
}

// Mask secrets for logging
func maskSecret(secret string) string {
	if len(secret) < 4 {
		return "***"
	}
	return secret[:4] + "***"
}
