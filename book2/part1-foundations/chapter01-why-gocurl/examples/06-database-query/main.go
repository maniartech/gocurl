// Package main demonstrates database REST API integration using GoCurl.
// This example queries a Supabase table using its auto-generated REST API.
//
// Prerequisites:
//   - Supabase project (https://supabase.com)
//   - Create a table named "users" with columns: id, email, created_at
//   - Get API URL and anon key from project settings
//   - Set environment variables:
//     export SUPABASE_URL="https://xxx.supabase.co"
//     export SUPABASE_KEY="eyJ..."
//
// Run: go run database_query.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/maniartech/gocurl"
)

// User represents a database user record
type User struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

func main() {
	ctx := context.Background()

	// Get credentials from environment
	supabaseURL := os.Getenv("SUPABASE_URL")
	apiKey := os.Getenv("SUPABASE_KEY")

	if supabaseURL == "" || apiKey == "" {
		log.Fatal("SUPABASE_URL and SUPABASE_KEY environment variables must be set")
	}

	// Query users table
	var users []User
	resp, err := gocurl.CurlJSON(ctx, &users,
		"-H", "apikey: "+apiKey,
		"-H", "Authorization: Bearer "+apiKey,
		supabaseURL+"/rest/v1/users?select=*&limit=10")

	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer resp.Body.Close()

	// Check for API errors
	if resp.StatusCode != 200 {
		log.Fatalf("API returned status %d", resp.StatusCode)
	}

	// Display results
	fmt.Printf("Found %d users:\n", len(users))
	for i, user := range users {
		fmt.Printf("%d. %s (ID: %d, Created: %s)\n",
			i+1, user.Email, user.ID, user.CreatedAt)
	}

	// Alternative: Create a new user (POST example)
	createUser(ctx, supabaseURL, apiKey)
}

func createUser(ctx context.Context, baseURL, apiKey string) {
	payload := `{"email": "newuser@example.com"}`

	var newUser User
	resp, err := gocurl.CurlJSON(ctx, &newUser,
		"-X", "POST",
		"-H", "apikey: "+apiKey,
		"-H", "Authorization: Bearer "+apiKey,
		"-H", "Content-Type: application/json",
		"-d", payload,
		baseURL+"/rest/v1/users")

	if err != nil {
		log.Printf("Create user failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		fmt.Printf("\n✅ Created new user: %s (ID: %d)\n", newUser.Email, newUser.ID)
	}
}
