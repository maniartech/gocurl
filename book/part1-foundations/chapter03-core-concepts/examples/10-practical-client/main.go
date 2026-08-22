// Example 10: Complete Practical API Client
// Demonstrates building a production-ready API client with GoCurl.

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/maniartech/gocurl"
)

// GitHub API client
type GitHubClient struct {
	baseURL string
	token   string
	timeout time.Duration
}

// API response structures
type GitHubUser struct {
	Login       string `json:"login"`
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Company     string `json:"company"`
	Blog        string `json:"blog"`
	Location    string `json:"location"`
	Email       string `json:"email"`
	Bio         string `json:"bio"`
	PublicRepos int    `json:"public_repos"`
	Followers   int    `json:"followers"`
}

type GitHubRepo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Stars       int    `json:"stargazers_count"`
	Forks       int    `json:"forks_count"`
	OpenIssues  int    `json:"open_issues_count"`
}

// NewGitHubClient creates a new GitHub API client
func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		baseURL: "https://api.github.com",
		token:   token,
		timeout: 10 * time.Second,
	}
}

// GetUser retrieves user information
func (c *GitHubClient) GetUser(username string) (*GitHubUser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	url := fmt.Sprintf("%s/users/%s", c.baseURL, username)

	var user GitHubUser
	resp, err := gocurl.CurlJSON(ctx, &user,
		fmt.Sprintf(`curl %s`, url),
		`-H "Accept: application/vnd.github.v3+json"`,
		fmt.Sprintf(`-H "Authorization: token %s"`, c.token))

	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %d %s", resp.StatusCode, resp.Status)
	}

	return &user, nil
}

// ListUserRepos retrieves user's repositories
func (c *GitHubClient) ListUserRepos(username string, limit int) ([]GitHubRepo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	url := fmt.Sprintf("%s/users/%s/repos?sort=stars&per_page=%d", c.baseURL, username, limit)

	var repos []GitHubRepo
	resp, err := gocurl.CurlJSON(ctx, &repos,
		fmt.Sprintf(`curl %s`, url),
		`-H "Accept: application/vnd.github.v3+json"`,
		fmt.Sprintf(`-H "Authorization: token %s"`, c.token))

	if err != nil {
		return nil, fmt.Errorf("failed to list repos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %d %s", resp.StatusCode, resp.Status)
	}

	return repos, nil
}

// SearchRepositories searches for repositories
func (c *GitHubClient) SearchRepositories(query string, limit int) ([]GitHubRepo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	url := fmt.Sprintf("%s/search/repositories?q=%s&sort=stars&per_page=%d", c.baseURL, query, limit)

	var result struct {
		TotalCount int          `json:"total_count"`
		Items      []GitHubRepo `json:"items"`
	}

	resp, err := gocurl.CurlJSON(ctx, &result,
		fmt.Sprintf(`curl %s`, url),
		`-H "Accept: application/vnd.github.v3+json"`,
		fmt.Sprintf(`-H "Authorization: token %s"`, c.token))

	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %d %s", resp.StatusCode, resp.Status)
	}

	return result.Items, nil
}

func main() {
	fmt.Print("🚀 Complete Practical API Client Demonstration\n\n")

	// Create client (token optional for public API)
	client := NewGitHubClient("")

	// Example 1: Get user information
	example1_GetUser(client)

	// Example 2: List user repositories
	fmt.Println()
	example2_ListRepos(client)

	// Example 3: Search repositories
	fmt.Println()
	example3_SearchRepos(client)

	// Summary
	fmt.Println()
	printSummary()
}

func example1_GetUser(client *GitHubClient) {
	fmt.Println("1️⃣  Get User Information")

	user, err := client.GetUser("golang")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("   ✅ User retrieved successfully\n\n")
	fmt.Printf("   👤 %s (@%s)\n", user.Name, user.Login)
	fmt.Printf("   🏢 %s\n", user.Company)
	fmt.Printf("   📍 %s\n", user.Location)
	fmt.Printf("   📖 %s\n", truncate(user.Bio, 60))
	fmt.Printf("   📊 %d repos, %d followers\n", user.PublicRepos, user.Followers)
}

func example2_ListRepos(client *GitHubClient) {
	fmt.Println("2️⃣  List User Repositories")

	repos, err := client.ListUserRepos("golang", 5)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("   ✅ Retrieved %d repositories\n\n", len(repos))

	for i, repo := range repos {
		fmt.Printf("   %d. %s\n", i+1, repo.FullName)
		fmt.Printf("      %s\n", truncate(repo.Description, 60))
		fmt.Printf("      ⭐ %d stars, 🍴 %d forks, 📝 %d issues\n\n",
			repo.Stars, repo.Forks, repo.OpenIssues)
	}
}

func example3_SearchRepos(client *GitHubClient) {
	fmt.Println("3️⃣  Search Repositories")

	repos, err := client.SearchRepositories("http+language:go", 3)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("   ✅ Found %d repositories\n\n", len(repos))

	for i, repo := range repos {
		fmt.Printf("   %d. %s (%s)\n", i+1, repo.FullName, repo.Language)
		fmt.Printf("      %s\n", truncate(repo.Description, 60))
		fmt.Printf("      ⭐ %d stars\n\n", repo.Stars)
	}
}

func printSummary() {
	fmt.Println("📚 Client Design Patterns Summary")
	fmt.Println()
	fmt.Println("   ✅ Encapsulation:")
	fmt.Println("      • Client struct with configuration")
	fmt.Println("      • Methods for API operations")
	fmt.Println("      • Hide implementation details")
	fmt.Println()
	fmt.Println("   ✅ Error Handling:")
	fmt.Println("      • Check request errors")
	fmt.Println("      • Validate HTTP status codes")
	fmt.Println("      • Return wrapped errors")
	fmt.Println()
	fmt.Println("   ✅ Timeouts:")
	fmt.Println("      • Context with timeout for each request")
	fmt.Println("      • Configurable timeout duration")
	fmt.Println("      • Prevent hanging requests")
	fmt.Println()
	fmt.Println("   ✅ Type Safety:")
	fmt.Println("      • Strongly-typed request/response structures")
	fmt.Println("      • Automatic JSON marshaling")
	fmt.Println("      • Compile-time validation")
	fmt.Println()
	fmt.Println("   ✅ Reusability:")
	fmt.Println("      • Single client instance")
	fmt.Println("      • Consistent API across methods")
	fmt.Println("      • Easy to test and mock")
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}
