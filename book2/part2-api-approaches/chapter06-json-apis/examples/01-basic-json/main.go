package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/gocurl"
)

// GitHubUser represents a GitHub user's profile
type GitHubUser struct {
	Login       string `json:"login"`
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Bio         string `json:"bio"`
	PublicRepos int    `json:"public_repos"`
	Followers   int    `json:"followers"`
	Following   int    `json:"following"`
	CreatedAt   string `json:"created_at"`
}

func main() {
	fmt.Println("Example 1: Basic JSON GET with CurlJSON")
	fmt.Println("========================================\n")

	ctx := context.Background()

	// Fetch Linus Torvalds' GitHub profile
	var user GitHubUser
	resp, err := gocurl.CurlJSON(ctx, &user,
		"https://api.github.com/users/torvalds")

	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	fmt.Printf("Status: %d %s\n\n", resp.StatusCode, resp.Status)

	// Display user information
	fmt.Printf("User: %s (%s)\n", user.Name, user.Login)
	fmt.Printf("Bio: %s\n", user.Bio)
	fmt.Printf("Created: %s\n", user.CreatedAt)
	fmt.Printf("Public Repos: %d\n", user.PublicRepos)
	fmt.Printf("Followers: %d\n", user.Followers)
	fmt.Printf("Following: %d\n", user.Following)

	fmt.Println("\n✅ JSON automatically unmarshaled into struct")
}
