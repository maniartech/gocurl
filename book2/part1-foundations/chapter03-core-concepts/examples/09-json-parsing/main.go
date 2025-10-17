// Example 9: Advanced JSON Parsing
// Demonstrates various JSON parsing techniques and patterns.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/maniartech/gocurl"
)

// Strongly-typed structures
type User struct {
	Login    string `json:"login"`
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Company  string `json:"company"`
	Blog     string `json:"blog"`
	Location string `json:"location"`
	Email    string `json:"email"`
	Bio      string `json:"bio"`
}

type HTTPBinPost struct {
	Args    map[string]string      `json:"args"`
	Data    string                 `json:"data"`
	Files   map[string]string      `json:"files"`
	Form    map[string]string      `json:"form"`
	Headers map[string]string      `json:"headers"`
	JSON    map[string]interface{} `json:"json"`
	Origin  string                 `json:"origin"`
	URL     string                 `json:"url"`
}

func main() {
	fmt.Println("🔍 Advanced JSON Parsing Demonstration\n")

	// Pattern 1: Automatic unmarshaling (CurlJSON)
	pattern1_AutomaticUnmarshaling()

	// Pattern 2: Manual unmarshaling
	fmt.Println()
	pattern2_ManualUnmarshaling()

	// Pattern 3: Dynamic JSON (map[string]interface{})
	fmt.Println()
	pattern3_DynamicJSON()

	// Pattern 4: Nested structures
	fmt.Println()
	pattern4_NestedStructures()

	// Pattern 5: Error handling with JSON
	fmt.Println()
	pattern5_JSONErrorHandling()
}

func pattern1_AutomaticUnmarshaling() {
	fmt.Println("1️⃣  Automatic Unmarshaling (CurlJSON)")

	ctx := context.Background()
	var user User

	resp, err := gocurl.CurlJSON(ctx, &user, "https://api.github.com/users/golang")

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ JSON parsed automatically into struct\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("\n   👤 User: %s\n", user.Login)
	fmt.Printf("      Name: %s\n", user.Name)
	fmt.Printf("      Location: %s\n", user.Location)
	fmt.Printf("      Bio: %s\n", truncate(user.Bio, 60))
	fmt.Printf("\n   💡 CurlJSON handles unmarshaling automatically\n")
}

func pattern2_ManualUnmarshaling() {
	fmt.Println("2️⃣  Manual Unmarshaling")

	ctx := context.Background()

	// Get response body as string
	body, resp, err := gocurl.CurlString(ctx, "https://api.github.com/users/rust-lang")

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	// Manual unmarshaling
	var user User
	if err := json.Unmarshal([]byte(body), &user); err != nil {
		log.Printf("Unmarshal error: %v", err)
		return
	}

	fmt.Printf("   ✅ Manual JSON unmarshaling\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("   📏 Body length: %d bytes\n", len(body))
	fmt.Printf("\n   👤 User: %s\n", user.Login)
	fmt.Printf("      Name: %s\n", user.Name)
	fmt.Printf("\n   💡 Use manual unmarshaling for custom processing\n")
}

func pattern3_DynamicJSON() {
	fmt.Println("3️⃣  Dynamic JSON (Unknown Structure)")

	ctx := context.Background()

	// Use map for unknown structure
	var result map[string]interface{}

	resp, err := gocurl.CurlJSON(ctx, &result, "https://httpbin.org/json")

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Parsed into dynamic map\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("\n   🗂️  Top-level keys: %v\n", getKeys(result))

	// Access nested data
	if slideshow, ok := result["slideshow"].(map[string]interface{}); ok {
		fmt.Printf("\n   📖 Slideshow:\n")
		fmt.Printf("      Author: %v\n", slideshow["author"])
		fmt.Printf("      Title: %v\n", slideshow["title"])

		if slides, ok := slideshow["slides"].([]interface{}); ok {
			fmt.Printf("      Slides: %d\n", len(slides))
		}
	}

	fmt.Printf("\n   💡 map[string]interface{} handles any JSON structure\n")
}

func pattern4_NestedStructures() {
	fmt.Println("4️⃣  Nested Structures (POST with JSON)")

	ctx := context.Background()

	// POST JSON data
	jsonData := `{
        "user": {
            "name": "Alice",
            "email": "alice@example.com"
        },
        "message": "Hello World"
    }`

	var result HTTPBinPost

	resp, err := gocurl.CurlJSON(ctx, &result,
		`curl -X POST https://httpbin.org/post`,
		`-H "Content-Type: application/json"`,
		`-d '`+jsonData+`'`)

	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("   ✅ Nested JSON posted and parsed\n")
	fmt.Printf("   📊 Status: %d\n", resp.StatusCode)
	fmt.Printf("\n   📦 Echo Response:\n")
	fmt.Printf("      URL: %s\n", result.URL)

	if result.JSON != nil {
		fmt.Printf("      JSON keys: %v\n", getKeys(result.JSON))

		// Access nested data
		if user, ok := result.JSON["user"].(map[string]interface{}); ok {
			fmt.Printf("      User name: %v\n", user["name"])
			fmt.Printf("      User email: %v\n", user["email"])
		}
	}

	fmt.Printf("\n   💡 Handle nested JSON with proper struct definitions\n")
}

func pattern5_JSONErrorHandling() {
	fmt.Println("5️⃣  JSON Error Handling")

	ctx := context.Background()

	// Try to parse non-JSON response
	var user User

	resp, err := gocurl.CurlJSON(ctx, &user, "https://httpbin.org/html")

	if err != nil {
		fmt.Printf("   ❌ Expected error (non-JSON response): %v\n", err)
		fmt.Printf("   💡 CurlJSON will fail on non-JSON content\n")
		return
	}
	defer resp.Body.Close()

	// Alternative: Manual approach with error handling
	fmt.Println("\n   ✅ Better Approach: Manual Parsing with Validation")

	resp2, err := gocurl.Curl(ctx, "https://api.github.com/users/nonexistent-user-xyz")
	if err != nil {
		log.Printf("Request error: %v", err)
		return
	}
	defer resp2.Body.Close()

	contentType := resp2.Header.Get("Content-Type")
	fmt.Printf("      Content-Type: %s\n", contentType)

	bodyBytes, err := io.ReadAll(resp2.Body)
	if err != nil {
		log.Printf("Read error: %v", err)
		return
	}

	// Check status before parsing
	if resp2.StatusCode >= 400 {
		fmt.Printf("      ⚠️  HTTP error: %d\n", resp2.StatusCode)
		fmt.Printf("      📦 Error body: %s\n", string(bodyBytes))
		return
	}

	// Validate it's JSON
	var testJSON interface{}
	if err := json.Unmarshal(bodyBytes, &testJSON); err != nil {
		fmt.Printf("      ❌ Not valid JSON: %v\n", err)
		return
	}

	fmt.Println("\n   💡 Best Practice:")
	fmt.Println("      1. Check HTTP status code")
	fmt.Println("      2. Validate Content-Type")
	fmt.Println("      3. Handle unmarshal errors")
}

// Helper functions
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}
