package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/options"
)

// Example 2: POST with JSON
// Demonstrates POST request with JSON body and headers

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func main() {
	fmt.Println("Example 2: POST with JSON")
	fmt.Println("=========================")
	fmt.Println()

	// JSON data to send
	jsonBody := `{
		"name": "Alice Johnson",
		"email": "alice@example.com",
		"age": 28
	}`

	// Build POST request
	opts := options.NewRequestOptionsBuilder().
		SetURL("https://httpbin.org/post").
		SetMethod("POST").
		AddHeader("Content-Type", "application/json").
		AddHeader("User-Agent", "GoCurl-Example/1.0").
		SetBody(jsonBody).
		SetTimeout(30 * time.Second).
		Build()

	// Execute request
	ctx := context.Background()
	resp, err := gocurl.Execute(ctx, opts)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading body:", err)
		return
	}

	fmt.Println("✅ Request successful!")
	fmt.Println("Status Code:", resp.StatusCode)
	fmt.Println()
	fmt.Println("Response (first 500 characters):")
	if len(body) > 500 {
		fmt.Println(string(body[:500]) + "...")
	} else {
		fmt.Println(string(body))
	}
	fmt.Println()

	// Alternative: Using convenience methods
	fmt.Println("📌 Alternative: Using JSON() convenience method")
	fmt.Println()

	user := User{
		Name:  "Bob Smith",
		Email: "bob@example.com",
		Age:   35,
	}

	opts2 := options.NewRequestOptionsBuilder().
		SetURL("https://httpbin.org/post").
		SetMethod("POST").
		JSON(user). // Convenience method: marshals to JSON and sets Content-Type
		Build()

	resp2, err := gocurl.Execute(ctx, opts2)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp2.Body.Close()

	fmt.Println("✅ JSON() method request successful!")
	fmt.Println("Status Code:", resp2.StatusCode)
	fmt.Println()

	// Comparison
	fmt.Println("Curl-syntax equivalent:")
	fmt.Println(`resp, err := gocurl.Curl(ctx, "-X", "POST", "-H", "Content-Type: application/json", "-d", jsonBody, "https://httpbin.org/post")`)
	fmt.Println()

	fmt.Println("💡 Key Learnings:")
	fmt.Println("   • SetMethod() for HTTP method")
	fmt.Println("   • AddHeader() for multiple headers")
	fmt.Println("   • SetBody() for request body")
	fmt.Println("   • JSON() convenience method")
	fmt.Println("   • SetTimeout() for request timeout")
}
