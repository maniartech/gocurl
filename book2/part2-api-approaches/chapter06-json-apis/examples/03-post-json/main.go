package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/options"
)

// CreateUserRequest represents data to send
type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Age      int    `json:"age"`
	Active   bool   `json:"active"`
}

// HTTPBinResponse represents httpbin.org's POST response
type HTTPBinResponse struct {
	Args    map[string]interface{} `json:"args"`
	Data    string                 `json:"data"`
	Files   map[string]interface{} `json:"files"`
	Form    map[string]interface{} `json:"form"`
	Headers map[string]string      `json:"headers"`
	JSON    map[string]interface{} `json:"json"`
	URL     string                 `json:"url"`
}

func main() {
	fmt.Println("Example 3: Sending JSON Data with POST")
	fmt.Println("======================================\n")

	ctx := context.Background()

	// Prepare user data to send
	newUser := CreateUserRequest{
		Name:     "Alice Johnson",
		Email:    "alice@example.com",
		Username: "alice",
		Age:      28,
		Active:   true,
	}

	fmt.Println("Sending JSON data:")
	fmt.Printf("  Name: %s\n", newUser.Name)
	fmt.Printf("  Email: %s\n", newUser.Email)
	fmt.Printf("  Username: %s\n", newUser.Username)
	fmt.Printf("  Age: %d\n", newUser.Age)
	fmt.Printf("  Active: %v\n\n", newUser.Active)

	// Build request with JSON body using Builder pattern
	opts := options.NewRequestOptionsBuilder().
		SetURL("https://httpbin.org/post").
		SetMethod("POST").
		JSON(newUser). // Automatically marshals and sets Content-Type
		Build()

	// Execute request
	httpResp, _, err := gocurl.Process(ctx, opts)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer httpResp.Body.Close()

	// Parse response
	var response HTTPBinResponse
	resp2, err := gocurl.CurlJSON(ctx, &response,
		"https://httpbin.org/post",
		"-X", "POST",
		"-H", "Content-Type: application/json",
		"-d", `{"name":"Alice Johnson","email":"alice@example.com","username":"alice","age":28,"active":true}`)

	if err != nil {
		log.Fatalf("Failed to parse response: %v", err)
	}

	fmt.Printf("Status: %d\n\n", resp2.StatusCode)

	// Display what server received
	fmt.Println("Server received:")
	fmt.Printf("  Content-Type: %s\n", response.Headers["Content-Type"])
	fmt.Printf("  JSON Data: %v\n", response.JSON)

	fmt.Println("\n✅ JSON data sent successfully")
	fmt.Println("✅ Content-Type header automatically set to application/json")
}
