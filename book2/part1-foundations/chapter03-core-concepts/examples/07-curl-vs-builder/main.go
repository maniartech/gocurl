// Example 7: Authentication Patterns
// Demonstrates various authentication methods (API keys, tokens, Basic auth).

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/maniartech/gocurl"
)

type HTTPBinResponse struct {
	Headers map[string]string `json:"headers"`
}

func main() {
	fmt.Print("� Authentication Patterns Demonstration\n\n")

	// Pattern 1: API Key in header
	pattern1_APIKey()

	// Pattern 2: Bearer token
	fmt.Println()
	pattern2_BearerToken()

	// Pattern 3: Basic authentication
	fmt.Println()
	pattern3_BasicAuth()

	// Pattern 4: Custom authentication
	fmt.Println()
	pattern4_CustomAuth()
}

func pattern1_APIKey() {
	fmt.Println("1️⃣  API Key Authentication")

	ctx := context.Background()

	// Set API key in environment
	os.Setenv("API_KEY", "my-secret-api-key-12345")

	var result HTTPBinResponse
	resp, err := gocurl.CurlJSON(ctx, &result,
		`curl https://httpbin.org/headers`,
		`-H "X-API-Key: ${API_KEY}"`)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Authenticated with API key\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("   🔑 API key sent in header: X-API-Key\n")

	if apiKey, ok := result.Headers["X-Api-Key"]; ok {
		fmt.Printf("   📨 Server received: %s\n", maskSecret(apiKey))
	}

	fmt.Printf("\n   💡 Common header names:\n")
	fmt.Printf("      • X-API-Key\n")
	fmt.Printf("      • X-Api-Key\n")
	fmt.Printf("      • api-key\n")
	fmt.Printf("      • apikey\n")
}

func pattern2_BearerToken() {
	fmt.Println("2️⃣  Bearer Token Authentication")

	ctx := context.Background()

	// Set token in environment
	os.Setenv("ACCESS_TOKEN", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...")

	var result HTTPBinResponse
	resp, err := gocurl.CurlJSON(ctx, &result,
		`curl https://httpbin.org/headers`,
		`-H "Authorization: Bearer ${ACCESS_TOKEN}"`)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Authenticated with Bearer token\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("   🔑 Token sent in Authorization header\n")

	if auth, ok := result.Headers["Authorization"]; ok {
		fmt.Printf("   📨 Server received: %s\n", maskSecret(auth))
	}

	fmt.Printf("\n   💡 Bearer token is standard OAuth 2.0 format\n")
	fmt.Printf("   💡 Format: Authorization: Bearer <token>\n")
}

func pattern3_BasicAuth() {
	fmt.Println("3️⃣  Basic Authentication")

	ctx := context.Background()

	// Method 1: Using -u flag (curl syntax)
	fmt.Println("   🅰️  Method 1: Using -u flag")
	var result1 HTTPBinResponse
	resp1, err := gocurl.CurlJSON(ctx, &result1,
		`curl -u username:password https://httpbin.org/basic-auth/username/password`)

	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		defer resp1.Body.Close()
		fmt.Printf("      ✅ Status: %d\n", resp1.StatusCode)
	}

	// Method 2: Manual Authorization header
	fmt.Println("\n   🅱️  Method 2: Manual header")
	credentials := base64.StdEncoding.EncodeToString([]byte("username:password"))
	authHeader := fmt.Sprintf("Basic %s", credentials)

	var result2 HTTPBinResponse
	resp2, err := gocurl.CurlJSON(ctx, &result2,
		`curl https://httpbin.org/basic-auth/username/password`,
		fmt.Sprintf(`-H "Authorization: %s"`, authHeader))

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp2.Body.Close()

	fmt.Printf("      ✅ Status: %d\n", resp2.StatusCode)
	fmt.Printf("      🔒 Credentials encoded: %s...\n", credentials[:20])

	fmt.Printf("\n   💡 Basic auth encodes 'username:password' in base64\n")
	fmt.Printf("   💡 Format: Authorization: Basic <base64>\n")
}

func pattern4_CustomAuth() {
	fmt.Println("4️⃣  Custom Authentication Headers")

	ctx := context.Background()

	// Some APIs use custom authentication headers
	os.Setenv("CLIENT_ID", "my-client-id")
	os.Setenv("CLIENT_SECRET", "my-client-secret")
	os.Setenv("SESSION_TOKEN", "session-abc-123")

	var result HTTPBinResponse
	resp, err := gocurl.CurlJSON(ctx, &result,
		`curl https://httpbin.org/headers`,
		`-H "X-Client-ID: ${CLIENT_ID}"`,
		`-H "X-Client-Secret: ${CLIENT_SECRET}"`,
		`-H "X-Session-Token: ${SESSION_TOKEN}"`)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Authenticated with custom headers\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("   🔑 Custom headers sent:\n")
	fmt.Printf("      • X-Client-ID\n")
	fmt.Printf("      • X-Client-Secret\n")
	fmt.Printf("      • X-Session-Token\n")

	fmt.Printf("\n   � Custom Authentication Examples:\n")
	fmt.Printf("      • AWS: X-Amz-Date, Authorization\n")
	fmt.Printf("      • Azure: Ocp-Apim-Subscription-Key\n")
	fmt.Printf("      • Stripe: Authorization: Bearer sk_...\n")
	fmt.Printf("      • GitHub: Authorization: token ghp_...\n")

	fmt.Printf("\n   ✅ Security Best Practices:\n")
	fmt.Printf("      1. Never hard-code credentials\n")
	fmt.Printf("      2. Use environment variables\n")
	fmt.Printf("      3. Store secrets in secure vault\n")
	fmt.Printf("      4. Use HTTPS for all API calls\n")
	fmt.Printf("      5. Rotate credentials regularly\n")
	fmt.Printf("      6. Mask secrets in logs\n")
}

func maskSecret(secret string) string {
	if len(secret) < 8 {
		return "***"
	}
	return secret[:8] + "***"
}
