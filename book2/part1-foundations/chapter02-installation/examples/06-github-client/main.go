// Example 6: GitHub Client
// Demonstrates building a structured, reusable API client.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/maniartech/gocurl"
)

// GitHubClient encapsulates GitHub API interactions
type GitHubClient struct {
	token   string
	baseURL string
}

// NewGitHubClient creates a new GitHub API client
func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		token:   token,
		baseURL: "https://api.github.com",
	}
}

// Repository represents a GitHub repository
type Repository struct {
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	Description     string `json:"description"`
	Private         bool   `json:"private"`
	StargazersCount int    `json:"stargazers_count"`
	ForksCount      int    `json:"forks_count"`
	Language        string `json:"language"`
	URL             string `json:"html_url"`
}

// GetRepository fetches repository information
func (c *GitHubClient) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	var repository Repository

	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, owner, repo)

	resp, err := gocurl.CurlJSONCommand(ctx, &repository,
		`curl -H "Accept: application/vnd.github+json" \
              -H "Authorization: Bearer `+c.token+`" \
              `+url)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	return &repository, nil
}

func main() {
	fmt.Println("🐙 GitHub Client Example\n")

	// Get token from environment (or use empty for public repos)
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("ℹ️  GITHUB_TOKEN not set - using unauthenticated access")
		fmt.Println("   (Rate limited to 60 requests/hour)")
		token = "" // Unauthenticated access works for public repos
	} else {
		fmt.Println("✅ Using authenticated access (5000 requests/hour)")
	}

	// Create GitHub client
	client := NewGitHubClient(token)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Fetch repository information
	fmt.Println("\n📦 Fetching repository: golang/go")
	repo, err := client.GetRepository(ctx, "golang", "go")

	if err != nil {
		log.Fatalf("❌ Error: %v", err)
	}

	// Display repository information
	separator := "============================================================"
	fmt.Println("\n" + separator)
	fmt.Printf("📂 %s\n", repo.FullName)
	fmt.Println(separator)
	fmt.Printf("📝 Description: %s\n", repo.Description)
	fmt.Printf("⭐ Stars: %d\n", repo.StargazersCount)
	fmt.Printf("🍴 Forks: %d\n", repo.ForksCount)
	fmt.Printf("💻 Language: %s\n", repo.Language)
	fmt.Printf("🔒 Private: %v\n", repo.Private)
	fmt.Printf("🔗 URL: %s\n", repo.URL)
	fmt.Println(separator)

	// Demonstrate code organization benefits
	fmt.Println("\n✅ Benefits of this structure:")
	fmt.Println("  1. Encapsulated API logic in GitHubClient struct")
	fmt.Println("  2. Type-safe with Repository struct")
	fmt.Println("  3. Reusable across your application")
	fmt.Println("  4. Easy to test and mock")
	fmt.Println("  5. Clear separation of concerns")

	fmt.Println("\n💡 This pattern scales to complex API integrations!")
}
