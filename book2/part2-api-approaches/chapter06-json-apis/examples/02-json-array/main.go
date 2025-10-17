package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/gocurl"
)

// Repository represents a GitHub repository
type Repository struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Stars       int    `json:"stargazers_count"`
	Forks       int    `json:"forks_count"`
	Language    string `json:"language"`
	Private     bool   `json:"private"`
	Fork        bool   `json:"fork"`
	CreatedAt   string `json:"created_at"`
}

func main() {
	fmt.Println("Example 2: Fetching JSON Arrays")
	fmt.Println("================================\n")

	ctx := context.Background()

	// Fetch Linus Torvalds' public repositories
	var repos []Repository
	resp, err := gocurl.CurlJSON(ctx, &repos,
		"https://api.github.com/users/torvalds/repos?per_page=10&sort=updated")

	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	fmt.Printf("Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("Found %d repositories\n\n", len(repos))

	// Display repository information
	for i, repo := range repos {
		fmt.Printf("%d. %s\n", i+1, repo.Name)
		fmt.Printf("   Full Name: %s\n", repo.FullName)
		if repo.Description != "" {
			fmt.Printf("   Description: %s\n", repo.Description)
		}
		fmt.Printf("   Language: %s\n", repo.Language)
		fmt.Printf("   ⭐ %d  🍴 %d\n", repo.Stars, repo.Forks)
		fmt.Printf("   Created: %s\n", repo.CreatedAt)

		if repo.Fork {
			fmt.Println("   [FORK]")
		}
		fmt.Println()
	}

	fmt.Println("✅ JSON array automatically unmarshaled into slice")
}
