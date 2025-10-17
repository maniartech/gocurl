# Exercise 3: Production API Client

**Difficulty:** Advanced
**Duration:** 90-120 minutes
**Prerequisites:** Exercises 1-2 completed, solid Go concurrency knowledge

## Objective

Build a production-ready API client demonstrating thread-safe concurrency, proper context management, validation, error handling, and real-world patterns. This exercise simulates building a complete GitHub API client.

## Project Overview

You'll build a GitHub API client that:
- Fetches user data and repositories concurrently
- Uses request templates for code reuse
- Implements proper timeout and cancellation
- Handles rate limiting gracefully
- Validates all requests
- Logs operations for debugging

## Setup

Create a new directory for this exercise:

```bash
mkdir exercise3-api-client
cd exercise3-api-client
go mod init example.com/github-client
go get github.com/maniartech/gocurl
```

Set your GitHub token:

```bash
export GITHUB_TOKEN="your-token-here"
```

## Architecture

```
github-client/
├── main.go              # Entry point
├── client.go            # API client with base configuration
├── users.go             # User-related operations
├── repos.go             # Repository operations
└── client_test.go       # Tests
```

---

## Task 1: API Client Foundation

**File:** `client.go`

Build the base client structure with common configuration.

**Requirements:**
- Store base URL and authentication token
- Provide method to create base builder with common headers
- Include logging capability
- Support custom HTTP client

**Starter Code:**

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/options"
)

// GitHubClient is a production-ready GitHub API client
type GitHubClient struct {
    baseURL    string
    token      string
    httpClient *http.Client
    logger     *log.Logger
}

// NewGitHubClient creates a new GitHub API client
func NewGitHubClient(token string) *GitHubClient {
    return &GitHubClient{
        baseURL: "https://api.github.com",
        token:   token,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        logger: log.Default(),
    }
}

// createBaseBuilder returns a builder with common configuration
func (c *GitHubClient) createBaseBuilder() *options.RequestOptionsBuilder {
    // TODO: Create builder with:
    // - Authorization: Bearer <token>
    // - Accept: application/vnd.github.v3+json
    // - User-Agent: gocurl-github-client/1.0
    // - Custom HTTP client
    // - Default retry (3 attempts)
    // - 30 second timeout

    return nil
}

// execute is a helper that validates and executes requests
func (c *GitHubClient) execute(ctx context.Context, builder *options.RequestOptionsBuilder) (*http.Response, error) {
    // TODO: Validate request
    // TODO: Build options
    // TODO: Log request details
    // TODO: Execute request
    // TODO: Log response
    // TODO: Handle rate limiting (check X-RateLimit headers)

    return nil, nil
}
```

**Tests to Pass:**

```go
func TestClientCreation(t *testing.T) {
    client := NewGitHubClient("test-token")
    if client.baseURL != "https://api.github.com" {
        t.Error("Wrong base URL")
    }
}

func TestBaseBuilder(t *testing.T) {
    client := NewGitHubClient("test-token")
    builder := client.createBaseBuilder()
    opts := builder.Build()

    // Check headers are set
    if opts.Headers.Get("Authorization") != "Bearer test-token" {
        t.Error("Authorization header not set")
    }
}
```

---

## Task 2: User Operations

**File:** `users.go`

Implement user-related API calls using the base builder.

**Requirements:**
- GetUser(ctx, username) - fetch single user
- GetCurrentUser(ctx) - fetch authenticated user
- Proper error handling for 404 and 401
- Return structured data (User struct)

**Starter Code:**

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
)

// User represents a GitHub user
type User struct {
    Login     string `json:"login"`
    ID        int    `json:"id"`
    Name      string `json:"name"`
    Email     string `json:"email"`
    Bio       string `json:"bio"`
    PublicRepos int  `json:"public_repos"`
    Followers int    `json:"followers"`
    Following int    `json:"following"`
}

// GetUser fetches a user by username
func (c *GitHubClient) GetUser(ctx context.Context, username string) (*User, error) {
    // TODO: Create builder from base
    // TODO: Set URL to /users/{username}
    // TODO: Execute request
    // TODO: Parse JSON response
    // TODO: Handle 404 error

    return nil, nil
}

// GetCurrentUser fetches the authenticated user
func (c *GitHubClient) GetCurrentUser(ctx context.Context) (*User, error) {
    // TODO: Similar to GetUser but use /user endpoint

    return nil, nil
}

// parseUserResponse is a helper to parse User JSON
func parseUserResponse(body io.Reader) (*User, error) {
    var user User
    if err := json.NewDecoder(body).Decode(&user); err != nil {
        return nil, fmt.Errorf("failed to parse user: %w", err)
    }
    return &user, nil
}
```

**Tests to Pass:**

```go
func TestGetUser(t *testing.T) {
    client := NewGitHubClient(os.Getenv("GITHUB_TOKEN"))
    user, err := client.GetUser(context.Background(), "torvalds")
    if err != nil {
        t.Fatalf("Failed to get user: %v", err)
    }
    if user.Login != "torvalds" {
        t.Errorf("Expected login 'torvalds', got '%s'", user.Login)
    }
}
```

---

## Task 3: Repository Operations

**File:** `repos.go`

Implement repository API calls with pagination support.

**Requirements:**
- ListUserRepos(ctx, username, page, perPage) - list user's repositories
- GetRepo(ctx, owner, name) - get single repository
- Handle pagination parameters
- Return structured data (Repo struct)

**Starter Code:**

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/url"
)

// Repo represents a GitHub repository
type Repo struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    FullName    string `json:"full_name"`
    Description string `json:"description"`
    Private     bool   `json:"private"`
    Fork        bool   `json:"fork"`
    Stars       int    `json:"stargazers_count"`
    Forks       int    `json:"forks_count"`
    Language    string `json:"language"`
}

// ListUserRepos lists repositories for a user
func (c *GitHubClient) ListUserRepos(ctx context.Context, username string, page, perPage int) ([]Repo, error) {
    // TODO: Create builder from base
    // TODO: Set URL to /users/{username}/repos
    // TODO: Add query parameters: page, per_page, sort=updated
    // TODO: Execute request
    // TODO: Parse JSON array response

    return nil, nil
}

// GetRepo fetches a single repository
func (c *GitHubClient) GetRepo(ctx context.Context, owner, name string) (*Repo, error) {
    // TODO: Create builder from base
    // TODO: Set URL to /repos/{owner}/{name}
    // TODO: Execute request
    // TODO: Parse JSON response

    return nil, nil
}

// parseReposResponse parses array of repositories
func parseReposResponse(body io.Reader) ([]Repo, error) {
    var repos []Repo
    if err := json.NewDecoder(body).Decode(&repos); err != nil {
        return nil, fmt.Errorf("failed to parse repos: %w", err)
    }
    return repos, nil
}
```

---

## Task 4: Concurrent Operations with Clone()

**File:** `main.go` (add these functions)

Implement concurrent fetching using Clone() for thread safety.

**Requirements:**
- FetchMultipleUsers(ctx, usernames) - fetch users concurrently
- FetchUserWithRepos(ctx, username) - fetch user and repos in parallel
- Use Clone() to create independent request configurations
- Aggregate results from goroutines
- Handle errors from any goroutine

**Starter Code:**

```go
package main

import (
    "context"
    "fmt"
    "sync"
)

// UserWithRepos combines user and their repositories
type UserWithRepos struct {
    User  *User
    Repos []Repo
    Error error
}

// FetchMultipleUsers fetches multiple users concurrently
func (c *GitHubClient) FetchMultipleUsers(ctx context.Context, usernames []string) ([]*User, error) {
    results := make([]*User, len(usernames))
    errors := make([]error, len(usernames))

    var wg sync.WaitGroup

    for i, username := range usernames {
        wg.Add(1)
        go func(index int, user string) {
            defer wg.Done()

            // TODO: Call GetUser for this username
            // TODO: Store result in results[index]
            // TODO: Store error in errors[index]

        }(i, username)
    }

    wg.Wait()

    // TODO: Check if any errors occurred
    // TODO: Return results

    return results, nil
}

// FetchUserWithRepos fetches user and their repos in parallel
func (c *GitHubClient) FetchUserWithRepos(ctx context.Context, username string) (*UserWithRepos, error) {
    result := &UserWithRepos{}

    var wg sync.WaitGroup
    wg.Add(2)

    // Fetch user
    go func() {
        defer wg.Done()
        // TODO: Fetch user
    }()

    // Fetch repos
    go func() {
        defer wg.Done()
        // TODO: Fetch first page of repos
    }()

    wg.Wait()

    if result.Error != nil {
        return nil, result.Error
    }

    return result, nil
}
```

**Tests to Pass:**

```go
func TestConcurrentFetch(t *testing.T) {
    client := NewGitHubClient(os.Getenv("GITHUB_TOKEN"))
    usernames := []string{"torvalds", "gvanrossum", "mojombo"}

    users, err := client.FetchMultipleUsers(context.Background(), usernames)
    if err != nil {
        t.Fatalf("Failed to fetch users: %v", err)
    }

    if len(users) != 3 {
        t.Errorf("Expected 3 users, got %d", len(users))
    }
}

func TestRaceCondition(t *testing.T) {
    // Run with: go test -race
    client := NewGitHubClient(os.Getenv("GITHUB_TOKEN"))

    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, _ = client.GetUser(context.Background(), "torvalds")
        }()
    }
    wg.Wait()
}
```

---

## Task 5: Context Cancellation

**File:** `main.go` (add these functions)

Implement proper context cancellation and timeout handling.

**Requirements:**
- Support context cancellation mid-request
- Handle timeout gracefully
- Clean up resources on cancellation
- Log cancellation events

**Starter Code:**

```go
package main

import (
    "context"
    "time"
)

// FetchWithTimeout fetches user with specified timeout
func (c *GitHubClient) FetchWithTimeout(username string, timeout time.Duration) (*User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    c.logger.Printf("Fetching user %s with %v timeout", username, timeout)

    user, err := c.GetUser(ctx, username)
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            c.logger.Printf("Request timed out after %v", timeout)
            return nil, fmt.Errorf("timeout fetching user: %w", err)
        }
        return nil, err
    }

    return user, nil
}

// FetchWithCancellation demonstrates cancellation
func (c *GitHubClient) FetchWithCancellation(username string) (*User, context.CancelFunc, error) {
    ctx, cancel := context.WithCancel(context.Background())

    // Return cancel function so caller can cancel
    user, err := c.GetUser(ctx, username)
    return user, cancel, err
}
```

**Test:**

```go
func TestContextCancellation(t *testing.T) {
    client := NewGitHubClient(os.Getenv("GITHUB_TOKEN"))

    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately

    _, err := client.GetUser(ctx, "torvalds")
    if err == nil {
        t.Error("Expected error from cancelled context")
    }
}
```

---

## Task 6: Error Handling and Rate Limiting

**File:** `client.go` (enhance execute method)

Implement production-grade error handling.

**Requirements:**
- Check HTTP status codes
- Parse GitHub error responses
- Handle rate limiting (429, X-RateLimit headers)
- Retry on rate limit with backoff
- Custom error types

**Starter Code:**

```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "time"
)

// GitHubError represents an error from GitHub API
type GitHubError struct {
    Message          string `json:"message"`
    DocumentationURL string `json:"documentation_url"`
    StatusCode       int
}

func (e *GitHubError) Error() string {
    return fmt.Sprintf("GitHub API error (%d): %s", e.StatusCode, e.Message)
}

// checkRateLimit inspects rate limit headers
func (c *GitHubClient) checkRateLimit(resp *http.Response) error {
    remaining := resp.Header.Get("X-RateLimit-Remaining")
    reset := resp.Header.Get("X-RateLimit-Reset")

    if remaining == "0" {
        resetTime := time.Unix(parseResetTime(reset), 0)
        waitDuration := time.Until(resetTime)

        c.logger.Printf("Rate limit exceeded. Resets at %v (in %v)", resetTime, waitDuration)
        return fmt.Errorf("rate limit exceeded, resets in %v", waitDuration)
    }

    c.logger.Printf("Rate limit remaining: %s", remaining)
    return nil
}

func parseResetTime(reset string) int64 {
    timestamp, _ := strconv.ParseInt(reset, 10, 64)
    return timestamp
}

// handleErrorResponse parses GitHub error response
func handleErrorResponse(resp *http.Response) error {
    var ghError GitHubError
    if err := json.NewDecoder(resp.Body).Decode(&ghError); err != nil {
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }

    ghError.StatusCode = resp.StatusCode
    return &ghError
}
```

**Update execute method:**

```go
func (c *GitHubClient) execute(ctx context.Context, builder *options.RequestOptionsBuilder) (*http.Response, error) {
    if err := builder.Validate(); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    opts := builder.Build()

    c.logger.Printf("Request: %s %s", opts.Method, opts.URL)

    resp, err := gocurl.Execute(ctx, opts)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }

    // Check rate limit
    if err := c.checkRateLimit(resp); err != nil {
        resp.Body.Close()
        return nil, err
    }

    // Check status code
    if resp.StatusCode >= 400 {
        defer resp.Body.Close()
        return nil, handleErrorResponse(resp)
    }

    c.logger.Printf("Response: %d %s", resp.StatusCode, resp.Status)

    return resp, nil
}
```

---

## Task 7: Complete Main Program

**File:** `main.go`

Put it all together with a complete demonstration.

**Requirements:**
- Initialize client with token from environment
- Demonstrate all operations
- Show concurrent fetching
- Handle errors gracefully
- Log operations

**Starter Code:**

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"
)

func main() {
    token := os.Getenv("GITHUB_TOKEN")
    if token == "" {
        log.Fatal("GITHUB_TOKEN environment variable required")
    }

    client := NewGitHubClient(token)
    ctx := context.Background()

    // Task 1: Fetch current user
    fmt.Println("\n=== Fetching Current User ===")
    currentUser, err := client.GetCurrentUser(ctx)
    if err != nil {
        log.Printf("Error fetching current user: %v", err)
    } else {
        fmt.Printf("Authenticated as: %s\n", currentUser.Login)
        fmt.Printf("Public repos: %d\n", currentUser.PublicRepos)
    }

    // Task 2: Fetch specific user
    fmt.Println("\n=== Fetching Specific User ===")
    user, err := client.GetUser(ctx, "torvalds")
    if err != nil {
        log.Printf("Error fetching user: %v", err)
    } else {
        fmt.Printf("User: %s (%s)\n", user.Name, user.Login)
        fmt.Printf("Followers: %d\n", user.Followers)
    }

    // Task 3: Fetch user's repositories
    fmt.Println("\n=== Fetching Repositories ===")
    repos, err := client.ListUserRepos(ctx, "torvalds", 1, 5)
    if err != nil {
        log.Printf("Error fetching repos: %v", err)
    } else {
        fmt.Printf("Found %d repositories:\n", len(repos))
        for _, repo := range repos {
            fmt.Printf("  - %s (⭐ %d)\n", repo.Name, repo.Stars)
        }
    }

    // Task 4: Concurrent fetching
    fmt.Println("\n=== Fetching Multiple Users Concurrently ===")
    usernames := []string{"torvalds", "gvanrossum", "mojombo"}

    start := time.Now()
    users, err := client.FetchMultipleUsers(ctx, usernames)
    duration := time.Since(start)

    if err != nil {
        log.Printf("Error fetching multiple users: %v", err)
    } else {
        fmt.Printf("Fetched %d users in %v:\n", len(users), duration)
        for _, u := range users {
            if u != nil {
                fmt.Printf("  - %s\n", u.Login)
            }
        }
    }

    // Task 5: Fetch user with repos in parallel
    fmt.Println("\n=== Fetching User with Repos (Parallel) ===")
    userWithRepos, err := client.FetchUserWithRepos(ctx, "mojombo")
    if err != nil {
        log.Printf("Error fetching user with repos: %v", err)
    } else {
        fmt.Printf("User: %s\n", userWithRepos.User.Login)
        fmt.Printf("Repos: %d\n", len(userWithRepos.Repos))
    }

    // Task 6: Demonstrate timeout
    fmt.Println("\n=== Testing Timeout ===")
    _, err = client.FetchWithTimeout("torvalds", 1*time.Nanosecond)
    if err != nil {
        fmt.Printf("Expected timeout error: %v\n", err)
    }
}
```

---

## Validation Tests

**File:** `client_test.go`

```go
package main

import (
    "context"
    "os"
    "sync"
    "testing"
    "time"
)

func getTestClient() *GitHubClient {
    token := os.Getenv("GITHUB_TOKEN")
    if token == "" {
        panic("GITHUB_TOKEN required for tests")
    }
    return NewGitHubClient(token)
}

func TestGetUser(t *testing.T) {
    client := getTestClient()
    user, err := client.GetUser(context.Background(), "torvalds")

    if err != nil {
        t.Fatalf("GetUser failed: %v", err)
    }

    if user.Login != "torvalds" {
        t.Errorf("Expected login 'torvalds', got '%s'", user.Login)
    }
}

func TestGetCurrentUser(t *testing.T) {
    client := getTestClient()
    user, err := client.GetCurrentUser(context.Background())

    if err != nil {
        t.Fatalf("GetCurrentUser failed: %v", err)
    }

    if user.Login == "" {
        t.Error("Expected non-empty login")
    }
}

func TestListUserRepos(t *testing.T) {
    client := getTestClient()
    repos, err := client.ListUserRepos(context.Background(), "torvalds", 1, 5)

    if err != nil {
        t.Fatalf("ListUserRepos failed: %v", err)
    }

    if len(repos) == 0 {
        t.Error("Expected at least one repo")
    }
}

func TestConcurrentRequests(t *testing.T) {
    client := getTestClient()
    usernames := []string{"torvalds", "gvanrossum", "mojombo"}

    users, err := client.FetchMultipleUsers(context.Background(), usernames)

    if err != nil {
        t.Fatalf("FetchMultipleUsers failed: %v", err)
    }

    if len(users) != 3 {
        t.Errorf("Expected 3 users, got %d", len(users))
    }

    for i, user := range users {
        if user == nil {
            t.Errorf("User %d is nil", i)
        }
    }
}

func TestContextCancellation(t *testing.T) {
    client := getTestClient()

    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately

    _, err := client.GetUser(ctx, "torvalds")
    if err == nil {
        t.Error("Expected error from cancelled context")
    }
}

func TestContextTimeout(t *testing.T) {
    client := getTestClient()

    _, err := client.FetchWithTimeout("torvalds", 1*time.Nanosecond)
    if err == nil {
        t.Error("Expected timeout error")
    }
}

func TestRaceConditions(t *testing.T) {
    // Run with: go test -race
    client := getTestClient()

    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, _ = client.GetUser(context.Background(), "torvalds")
        }()
    }
    wg.Wait()
}

func TestValidation(t *testing.T) {
    client := getTestClient()
    builder := client.createBaseBuilder()

    if err := builder.Validate(); err != nil {
        t.Errorf("Base builder should validate: %v", err)
    }
}
```

**Run Tests:**

```bash
# Normal tests
go test -v

# Race detection
go test -race -v

# With coverage
go test -cover -v
```

---

## Expected Output

```
=== Fetching Current User ===
Authenticated as: your-username
Public repos: 42

=== Fetching Specific User ===
User: Linus Torvalds (torvalds)
Followers: 180000

=== Fetching Repositories ===
Found 5 repositories:
  - linux (⭐ 170000)
  - subsurface (⭐ 2500)
  - ...

=== Fetching Multiple Users Concurrently ===
Fetched 3 users in 856ms:
  - torvalds
  - gvanrossum
  - mojombo

=== Fetching User with Repos (Parallel) ===
User: mojombo
Repos: 5

=== Testing Timeout ===
Expected timeout error: timeout fetching user: context deadline exceeded
```

---

## Self-Check Criteria

- ✅ Client structure with base configuration
- ✅ Request template pattern implemented
- ✅ All CRUD operations work correctly
- ✅ Concurrent operations use Clone() properly
- ✅ Context cancellation handled correctly
- ✅ Rate limiting detected and logged
- ✅ Error handling is comprehensive
- ✅ All tests pass including -race
- ✅ Code is production-ready quality

## Production Checklist

- ✅ **Authentication:** Token from environment, never hardcoded
- ✅ **Validation:** All requests validated before execution
- ✅ **Error Handling:** Custom errors with context
- ✅ **Rate Limiting:** Detected and handled gracefully
- ✅ **Concurrency:** Thread-safe with proper Clone() usage
- ✅ **Context:** Proper timeout and cancellation support
- ✅ **Logging:** All operations logged for debugging
- ✅ **Testing:** Comprehensive tests with race detection
- ✅ **Resource Cleanup:** Contexts cleaned up properly

## Common Mistakes

1. **Not using Clone() for concurrent requests**
   ```go
   // ❌ WRONG - race condition
   for _, username := range usernames {
       go func(user string) {
           builder.SetURL("/users/" + user) // Modifies shared builder!
       }(username)
   }

   // ✅ CORRECT - use Clone() or separate builders
   baseBuilder := client.createBaseBuilder()
   for _, username := range usernames {
       go func(user string) {
           userOpts := baseBuilder.Build() // Create independent copy
           // ... or use Clone() if modifying
       }(username)
   }
   ```

2. **Ignoring context cancellation**
   ```go
   // ✅ Always check context
   if ctx.Err() != nil {
       return ctx.Err()
   }
   ```

3. **Not closing response bodies**
   ```go
   // ✅ Always defer close
   defer resp.Body.Close()
   ```

## Bonus Challenges

1. **Caching:** Add in-memory cache for user data
2. **Pagination:** Implement automatic pagination for ListUserRepos
3. **Batch Operations:** Add batch user fetch with progress reporting
4. **Metrics:** Track request counts, errors, and latency
5. **Retry Strategy:** Implement exponential backoff for failures

## Learning Outcomes

After completing this exercise, you can:
- ✅ Design production-ready API clients
- ✅ Implement request template patterns
- ✅ Handle concurrency safely with Clone()
- ✅ Manage context for timeout and cancellation
- ✅ Implement comprehensive error handling
- ✅ Handle rate limiting gracefully
- ✅ Write testable, maintainable code
- ✅ Use validation to catch errors early
- ✅ Apply Go best practices for APIs

## Next Steps

Congratulations! You've completed all Chapter 5 exercises. You now have:
- ✅ Solid understanding of RequestOptions structure
- ✅ Mastery of Builder pattern usage
- ✅ Production-ready concurrent programming skills
- ✅ Complete API client implementation

**Continue to Chapter 6:** Working with JSON APIs
