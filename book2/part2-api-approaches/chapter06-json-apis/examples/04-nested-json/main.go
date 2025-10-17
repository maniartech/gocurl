package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/gocurl"
)

// GitHubIssue represents a GitHub issue with nested structures
type GitHubIssue struct {
	ID        int        `json:"id"`
	Number    int        `json:"number"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	Body      string     `json:"body"`
	User      User       `json:"user"`      // Nested struct
	Labels    []Label    `json:"labels"`    // Array of nested structs
	Milestone *Milestone `json:"milestone"` // Optional nested struct
	Comments  int        `json:"comments"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
}

// User represents a GitHub user (nested in Issue)
type User struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	AvatarURL string `json:"avatar_url"`
	Type      string `json:"type"`
}

// Label represents an issue label
type Label struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Milestone represents an optional milestone
type Milestone struct {
	ID          int    `json:"id"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
}

func main() {
	fmt.Println("Example 4: Working with Nested JSON Structures")
	fmt.Println("===============================================\n")

	ctx := context.Background()

	// Fetch open issues from golang/go repository
	var issues []GitHubIssue
	resp, err := gocurl.CurlJSON(ctx, &issues,
		"https://api.github.com/repos/golang/go/issues?state=open&per_page=5")

	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	fmt.Printf("Status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("Found %d open issues\n\n", len(issues))

	// Display issues with nested data
	for i, issue := range issues {
		fmt.Printf("%d. Issue #%d: %s\n", i+1, issue.Number, issue.Title)
		fmt.Printf("   State: %s\n", issue.State)

		// Access nested User struct
		fmt.Printf("   Author: %s (ID: %d, Type: %s)\n",
			issue.User.Login, issue.User.ID, issue.User.Type)

		// Handle array of nested Label structs
		if len(issue.Labels) > 0 {
			fmt.Printf("   Labels: ")
			for j, label := range issue.Labels {
				if j > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("%s (#%s)", label.Name, label.Color)
			}
			fmt.Println()
		} else {
			fmt.Println("   Labels: none")
		}

		// Handle optional nested Milestone struct
		if issue.Milestone != nil {
			fmt.Printf("   Milestone: %s (State: %s)\n",
				issue.Milestone.Title, issue.Milestone.State)
		} else {
			fmt.Println("   Milestone: none")
		}

		fmt.Printf("   Comments: %d\n", issue.Comments)
		fmt.Printf("   Created: %s\n", issue.CreatedAt)

		// Truncate body if too long
		if len(issue.Body) > 100 {
			fmt.Printf("   Body: %s...\n", issue.Body[:100])
		} else if issue.Body != "" {
			fmt.Printf("   Body: %s\n", issue.Body)
		}

		fmt.Println()
	}

	fmt.Println("✅ Successfully handled nested JSON structures:")
	fmt.Println("   • Nested objects (User)")
	fmt.Println("   • Arrays of nested objects (Labels)")
	fmt.Println("   • Optional nested objects (Milestone)")
}
