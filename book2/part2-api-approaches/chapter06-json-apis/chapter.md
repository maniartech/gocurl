# Chapter 6: Working with JSON APIs

**Purpose:** Master JSON request/response patterns for building type-safe, production-ready API clients.

**What You'll Learn:**
- Use CurlJSON functions for automatic unmarshaling
- Send JSON requests with proper content types
- Handle nested and complex JSON structures
- Implement type-safe response handling
- Parse and handle JSON errors gracefully
- Build robust API clients for real-world services

**Time Investment:** 2-3 hours
**Difficulty:** Intermediate

---

## Introduction

Modern APIs speak JSON. Whether you're integrating with GitHub, Stripe, AWS, or any RESTful service, you'll be sending and receiving JSON data. GoCurl provides both high-level convenience functions and low-level control for working with JSON APIs.

In this chapter, you'll learn:

1. **CurlJSON Functions** - Automatic unmarshaling into Go structs
2. **Manual JSON Handling** - Using CurlString for custom parsing
3. **POST/PUT JSON Data** - Sending JSON request bodies
4. **Type Safety** - Leveraging Go's type system
5. **Error Handling** - Gracefully handling API errors
6. **Real-World Patterns** - GitHub, Stripe, and custom APIs

---

## Part 1: Understanding JSON in GoCurl

### The Three Approaches

GoCurl offers three ways to work with JSON:

1. **CurlJSON** - Automatic unmarshaling (easiest)
2. **CurlString** - Manual JSON parsing (more control)
3. **Builder + JSON()** - Programmatic with type safety

Each has its place. Let's explore when to use each.

### CurlJSON Function Family

```go
// Signature:
func CurlJSON(ctx context.Context, v interface{}, command ...string) (*http.Response, error)
```

**Key Features:**
- Automatically decodes JSON into provided struct
- Returns `(*http.Response, error)` - only 2 values
- Body is already read and closed
- Handles content negotiation
- Type-safe at compile time

**When to Use:**
- ✅ Response structure is known
- ✅ You have a Go struct ready
- ✅ Type safety is important
- ✅ API is well-documented

**When NOT to Use:**
- ❌ Need to inspect raw JSON first
- ❌ Response structure varies
- ❌ Need to parse body multiple times

---

## Part 2: Basic JSON Responses

### Example 1: Simple JSON GET

Let's start with fetching a GitHub user:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

type GitHubUser struct {
    Login     string `json:"login"`
    ID        int    `json:"id"`
    Name      string `json:"name"`
    Bio       string `json:"bio"`
    PublicRepos int  `json:"public_repos"`
    Followers int    `json:"followers"`
}

func main() {
    ctx := context.Background()

    var user GitHubUser
    resp, err := gocurl.CurlJSON(ctx, &user,
        "https://api.github.com/users/torvalds")

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("User: %s\n", user.Name)
    fmt.Printf("Bio: %s\n", user.Bio)
    fmt.Printf("Repos: %d, Followers: %d\n",
        user.PublicRepos, user.Followers)
    fmt.Printf("Status: %d\n", resp.StatusCode)
}
```

**Output:**
```
User: Linus Torvalds
Bio: Creator of Linux
Repos: 5, Followers: 180000
Status: 200
```

**What Happened:**
1. Defined `GitHubUser` struct with JSON tags
2. Passed pointer to `&user` (must be pointer!)
3. GoCurl automatically unmarshaled JSON
4. Response body already closed
5. Type-safe access to fields

---

### Example 2: Handling JSON Arrays

Many APIs return arrays of objects:

```go
type Repository struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    FullName    string `json:"full_name"`
    Description string `json:"description"`
    Stars       int    `json:"stargazers_count"`
    Language    string `json:"language"`
}

func main() {
    ctx := context.Background()

    var repos []Repository
    resp, err := gocurl.CurlJSON(ctx, &repos,
        "https://api.github.com/users/torvalds/repos?per_page=5")

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Found %d repositories:\n", len(repos))
    for _, repo := range repos {
        fmt.Printf("  %s ⭐ %d\n", repo.Name, repo.Stars)
    }
}
```

**Output:**
```
Found 5 repositories:
  linux ⭐ 170000
  subsurface ⭐ 2500
  ...
```

**Key Points:**
- Use slice type `[]Repository` for arrays
- Still pass pointer: `&repos`
- Can iterate immediately after unmarshaling

---

## Part 3: Sending JSON Data

### POST Requests with JSON Body

To send JSON data, use Builder pattern with `JSON()` method:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/options"
)

type CreateUserRequest struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Username string `json:"username"`
}

type CreateUserResponse struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Message  string `json:"message"`
}

func main() {
    ctx := context.Background()

    // Prepare request data
    newUser := CreateUserRequest{
        Name:     "Alice Johnson",
        Email:    "alice@example.com",
        Username: "alice",
    }

    // Build request with JSON body
    opts := options.NewRequestOptionsBuilder().
        SetURL("https://httpbin.org/post").
        SetMethod("POST").
        JSON(newUser).  // Automatically marshals and sets Content-Type
        Build()

    // Execute
    httpResp, _, err := gocurl.Process(ctx, opts)
    if err != nil {
        log.Fatal(err)
    }
    defer httpResp.Body.Close()

    // Parse response
    var response CreateUserResponse
    if err := json.NewDecoder(httpResp.Body).Decode(&response); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Created user ID: %d\n", response.ID)
    fmt.Printf("Response: %s\n", response.Message)
}
```

**What `JSON()` Does:**
1. Marshals struct to JSON string
2. Sets `Content-Type: application/json`
3. Sets request body
4. Returns builder for chaining

---

### Alternative: Manual JSON with CurlString

For more control, marshal manually:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

func main() {
    ctx := context.Background()

    // Manual marshaling
    userData := map[string]interface{}{
        "name":  "Bob Smith",
        "email": "bob@example.com",
        "age":   30,
    }

    jsonData, err := json.Marshal(userData)
    if err != nil {
        log.Fatal(err)
    }

    // Send with curl syntax
    body, resp, err := gocurl.CurlString(ctx,
        "https://httpbin.org/post",
        "-X", "POST",
        "-H", "Content-Type: application/json",
        "-d", string(jsonData))

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Status: %d\n", resp.StatusCode)
    fmt.Printf("Response:\n%s\n", body)
}
```

**When to Use Manual Approach:**
- Need to inspect JSON string before sending
- Dealing with dynamic structures
- Debugging API calls
- Converting from curl commands

---

## Part 4: Nested JSON Structures

### Handling Complex JSON

Real APIs often return deeply nested JSON:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

// Nested structures
type GitHubIssue struct {
    ID     int    `json:"id"`
    Number int    `json:"number"`
    Title  string `json:"title"`
    State  string `json:"state"`
    User   User   `json:"user"`        // Nested struct
    Labels []Label `json:"labels"`      // Array of nested structs
}

type User struct {
    Login string `json:"login"`
    ID    int    `json:"id"`
    Type  string `json:"type"`
}

type Label struct {
    Name  string `json:"name"`
    Color string `json:"color"`
}

func main() {
    ctx := context.Background()

    var issues []GitHubIssue
    resp, err := gocurl.CurlJSON(ctx, &issues,
        "https://api.github.com/repos/golang/go/issues?state=open&per_page=3")

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Status: %d\n", resp.StatusCode)
    fmt.Printf("Found %d issues:\n\n", len(issues))

    for i, issue := range issues {
        fmt.Printf("%d. Issue #%d: %s\n", i+1, issue.Number, issue.Title)
        fmt.Printf("   Author: %s (%s)\n", issue.User.Login, issue.User.Type)

        if len(issue.Labels) > 0 {
            fmt.Printf("   Labels: ")
            for j, label := range issue.Labels {
                if j > 0 {
                    fmt.Printf(", ")
                }
                fmt.Printf("%s", label.Name)
            }
            fmt.Println()
        }
        fmt.Println()
    }
}
```

**Output:**
```
Status: 200
Found 3 issues:

1. Issue #12345: Proposal: Add generics to Go
   Author: alice (User)
   Labels: Proposal, NeedsDecision

2. Issue #12346: Bug: Memory leak in http client
   Author: bob (User)
   Labels: Bug, NeedsInvestigation

...
```

**Best Practices for Nested JSON:**
1. Define clear struct hierarchy
2. Use JSON tags consistently
3. Handle optional fields with pointers
4. Document struct meanings

---

## Part 5: Optional Fields and Null Values

### Handling Optional Fields

Not all JSON fields are always present. Use pointers for optional fields:

```go
type Product struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Description *string `json:"description"` // Optional
    Price       float64 `json:"price"`
    Discount    *float64 `json:"discount"`    // Optional
    Tags        []string `json:"tags"`         // Empty array vs null
}

func main() {
    ctx := context.Background()

    var product Product
    resp, err := gocurl.CurlJSON(ctx, &product,
        "https://api.example.com/products/123")

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Product: %s\n", product.Name)

    // Check optional fields
    if product.Description != nil {
        fmt.Printf("Description: %s\n", *product.Description)
    } else {
        fmt.Println("No description available")
    }

    if product.Discount != nil {
        finalPrice := product.Price * (1 - *product.Discount)
        fmt.Printf("Discounted price: $%.2f\n", finalPrice)
    } else {
        fmt.Printf("Price: $%.2f\n", product.Price)
    }
}
```

**Field Type Guidelines:**

| JSON Type | Go Type | When to Use |
|-----------|---------|-------------|
| Always present | `string`, `int`, `float64` | Required fields |
| Sometimes missing | `*string`, `*int`, `*float64` | Optional fields |
| Can be null | `*Type` | Explicit null values |
| Array (may be empty) | `[]Type` | Arrays (empty vs missing) |
| Object (may be missing) | `*StructType` | Optional nested objects |

---

## Part 6: Error Handling

### JSON Error Responses

APIs often return JSON error structures:

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

// Standard error response
type APIError struct {
    Error   string `json:"error"`
    Message string `json:"message"`
    Code    int    `json:"code"`
}

// Success response
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func fetchUser(ctx context.Context, userID int) (*User, error) {
    url := fmt.Sprintf("https://api.example.com/users/%d", userID)

    // Try to fetch user
    var user User
    resp, err := gocurl.CurlJSON(ctx, &user, url)

    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }

    // Check status code
    if resp.StatusCode != 200 {
        // Try to parse error response
        body, _, _ := gocurl.CurlString(ctx, url)

        var apiErr APIError
        if json.Unmarshal([]byte(body), &apiErr) == nil {
            return nil, fmt.Errorf("API error (%d): %s",
                resp.StatusCode, apiErr.Message)
        }

        return nil, fmt.Errorf("HTTP %d: %s",
            resp.StatusCode, resp.Status)
    }

    return &user, nil
}

func main() {
    ctx := context.Background()

    // Success case
    user, err := fetchUser(ctx, 123)
    if err != nil {
        log.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("User: %s (%s)\n", user.Name, user.Email)
    }

    // Error case
    user, err = fetchUser(ctx, 99999)
    if err != nil {
        log.Printf("Expected error: %v\n", err)
    }
}
```

**Error Handling Pattern:**
1. Make request with CurlJSON
2. Check response status code
3. If error, re-fetch with CurlString
4. Parse error JSON
5. Return descriptive error

---

### Robust Error Handling with Type Switch

For APIs with multiple error formats:

```go
type APIResponse struct {
    Success bool            `json:"success"`
    Data    json.RawMessage `json:"data"`  // Raw JSON for later parsing
    Error   *APIError       `json:"error"`
}

type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details map[string]interface{} `json:"details"`
}

func fetchWithErrorHandling(ctx context.Context, url string, result interface{}) error {
    var response APIResponse
    resp, err := gocurl.CurlJSON(ctx, &response, url)

    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }

    if !response.Success || resp.StatusCode >= 400 {
        if response.Error != nil {
            return fmt.Errorf("API error [%s]: %s",
                response.Error.Code, response.Error.Message)
        }
        return fmt.Errorf("HTTP %d: request failed", resp.StatusCode)
    }

    // Unmarshal the data field into result
    if err := json.Unmarshal(response.Data, result); err != nil {
        return fmt.Errorf("failed to parse data: %w", err)
    }

    return nil
}
```

**Key Techniques:**
- Use `json.RawMessage` for delayed parsing
- Check both success flag and status code
- Provide context in error messages
- Allow caller to handle errors appropriately

---

## Part 7: Advanced Patterns

### Pattern 1: Generic JSON Fetcher

Build a reusable function for any JSON endpoint:

```go
func FetchJSON[T any](ctx context.Context, url string) (*T, error) {
    var result T
    resp, err := gocurl.CurlJSON(ctx, &result, url)

    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("HTTP %d: %s",
            resp.StatusCode, resp.Status)
    }

    return &result, nil
}

// Usage:
func main() {
    ctx := context.Background()

    // Type inference
    user, err := FetchJSON[GitHubUser](ctx,
        "https://api.github.com/users/torvalds")

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("User: %s\n", user.Name)
}
```

**Benefits:**
- Type-safe with generics
- Reusable across project
- Consistent error handling
- Clean API

---

### Pattern 2: Pagination Helper

Many APIs paginate results:

```go
type PaginatedResponse[T any] struct {
    Data       []T    `json:"data"`
    Page       int    `json:"page"`
    PerPage    int    `json:"per_page"`
    TotalPages int    `json:"total_pages"`
    Total      int    `json:"total"`
}

func FetchAllPages[T any](ctx context.Context, baseURL string) ([]T, error) {
    var allResults []T
    page := 1

    for {
        url := fmt.Sprintf("%s?page=%d&per_page=100", baseURL, page)

        var response PaginatedResponse[T]
        resp, err := gocurl.CurlJSON(ctx, &response, url)

        if err != nil {
            return nil, err
        }

        if resp.StatusCode != 200 {
            return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
        }

        allResults = append(allResults, response.Data...)

        // Check if there are more pages
        if page >= response.TotalPages {
            break
        }

        page++
    }

    return allResults, nil
}
```

**Usage:**
```go
repos, err := FetchAllPages[Repository](ctx,
    "https://api.example.com/repos")

fmt.Printf("Fetched %d repositories across all pages\n", len(repos))
```

---

### Pattern 3: Caching JSON Responses

Add caching layer for frequently accessed data:

```go
type CachedClient struct {
    cache map[string]cachedResponse
    mu    sync.RWMutex
    ttl   time.Duration
}

type cachedResponse struct {
    data      interface{}
    timestamp time.Time
}

func NewCachedClient(ttl time.Duration) *CachedClient {
    return &CachedClient{
        cache: make(map[string]cachedResponse),
        ttl:   ttl,
    }
}

func (c *CachedClient) FetchJSON(ctx context.Context, url string, result interface{}) error {
    // Check cache
    c.mu.RLock()
    if cached, ok := c.cache[url]; ok {
        if time.Since(cached.timestamp) < c.ttl {
            c.mu.RUnlock()

            // Copy cached data to result
            data, _ := json.Marshal(cached.data)
            json.Unmarshal(data, result)
            return nil
        }
    }
    c.mu.RUnlock()

    // Fetch from API
    resp, err := gocurl.CurlJSON(ctx, result, url)
    if err != nil {
        return err
    }

    if resp.StatusCode == 200 {
        // Cache response
        c.mu.Lock()
        c.cache[url] = cachedResponse{
            data:      result,
            timestamp: time.Now(),
        }
        c.mu.Unlock()
    }

    return nil
}
```

**Usage:**
```go
client := NewCachedClient(5 * time.Minute)

var user GitHubUser
client.FetchJSON(ctx, "https://api.github.com/users/torvalds", &user)
// Second call uses cache
client.FetchJSON(ctx, "https://api.github.com/users/torvalds", &user)
```

---

## Part 8: Real-World Example - GitHub API Client

Let's build a complete GitHub API client:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/maniartech/gocurl"
)

type GitHubClient struct {
    baseURL string
    token   string
}

func NewGitHubClient(token string) *GitHubClient {
    return &GitHubClient{
        baseURL: "https://api.github.com",
        token:   token,
    }
}

type User struct {
    Login       string `json:"login"`
    Name        string `json:"name"`
    Bio         string `json:"bio"`
    PublicRepos int    `json:"public_repos"`
    Followers   int    `json:"followers"`
}

type Repository struct {
    ID          int    `json:"id"`
    Name        string `json:"name"`
    FullName    string `json:"full_name"`
    Description string `json:"description"`
    Stars       int    `json:"stargazers_count"`
    Language    string `json:"language"`
    Private     bool   `json:"private"`
}

func (c *GitHubClient) GetUser(ctx context.Context, username string) (*User, error) {
    url := fmt.Sprintf("%s/users/%s", c.baseURL, username)

    var user User
    resp, err := gocurl.CurlJSON(ctx, &user, url,
        "-H", fmt.Sprintf("Authorization: Bearer %s", c.token),
        "-H", "Accept: application/vnd.github.v3+json")

    if err != nil {
        return nil, err
    }

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("HTTP %d: failed to fetch user", resp.StatusCode)
    }

    return &user, nil
}

func (c *GitHubClient) ListRepos(ctx context.Context, username string) ([]Repository, error) {
    url := fmt.Sprintf("%s/users/%s/repos?per_page=10", c.baseURL, username)

    var repos []Repository
    resp, err := gocurl.CurlJSON(ctx, &repos, url,
        "-H", fmt.Sprintf("Authorization: Bearer %s", c.token),
        "-H", "Accept: application/vnd.github.v3+json")

    if err != nil {
        return nil, err
    }

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("HTTP %d: failed to fetch repos", resp.StatusCode)
    }

    return repos, nil
}

func main() {
    token := os.Getenv("GITHUB_TOKEN")
    if token == "" {
        log.Fatal("GITHUB_TOKEN environment variable required")
    }

    client := NewGitHubClient(token)
    ctx := context.Background()

    // Fetch user
    user, err := client.GetUser(ctx, "torvalds")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("User: %s (%s)\n", user.Name, user.Login)
    fmt.Printf("Bio: %s\n", user.Bio)
    fmt.Printf("Public Repos: %d\n", user.PublicRepos)
    fmt.Printf("Followers: %d\n\n", user.Followers)

    // Fetch repositories
    repos, err := client.ListRepos(ctx, "torvalds")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Top repositories:\n")
    for i, repo := range repos {
        if i >= 5 {
            break
        }
        fmt.Printf("  %d. %s ⭐ %d\n", i+1, repo.Name, repo.Stars)
    }
}
```

---

## Part 9: Comparison Table

### CurlJSON vs Manual Parsing

| Feature | CurlJSON | CurlString + json.Unmarshal |
|---------|----------|----------------------------|
| **Lines of Code** | ~5 | ~10 |
| **Type Safety** | ✅ Compile-time | ✅ Compile-time |
| **Performance** | Fast | Fast |
| **Error Context** | Less detailed | More detailed |
| **Flexibility** | Limited | High |
| **Body Inspection** | ❌ Already read | ✅ Available |
| **Multiple Parses** | ❌ Body closed | ✅ If saved |
| **Best For** | Known structures | Unknown/dynamic |

---

## Best Practices

### 1. Always Use Struct Tags
```go
// ✅ Good
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

// ❌ Bad - won't unmarshal correctly
type User struct {
    Name  string
    Email string
}
```

### 2. Handle Optional Fields with Pointers
```go
type Product struct {
    Name        string   `json:"name"`
    Description *string  `json:"description"` // May be null
    Price       float64  `json:"price"`
}
```

### 3. Check HTTP Status Codes
```go
resp, err := gocurl.CurlJSON(ctx, &result, url)
if err != nil {
    return err
}

if resp.StatusCode != 200 {
    return fmt.Errorf("unexpected status: %d", resp.StatusCode)
}
```

### 4. Provide Context in Errors
```go
if err != nil {
    return fmt.Errorf("failed to fetch user %s: %w", username, err)
}
```

### 5. Use json.RawMessage for Delayed Parsing
```go
type Response struct {
    Type string          `json:"type"`
    Data json.RawMessage `json:"data"` // Parse later based on type
}
```

---

## Common Pitfalls

### 1. Forgetting Pointer for CurlJSON
```go
// ❌ WRONG
var user User
gocurl.CurlJSON(ctx, user, url) // Won't work!

// ✅ CORRECT
var user User
gocurl.CurlJSON(ctx, &user, url)
```

### 2. Not Handling Non-200 Status
```go
// ❌ Incomplete
resp, err := gocurl.CurlJSON(ctx, &user, url)
if err != nil {
    return err
}
// Missing status check!

// ✅ Complete
resp, err := gocurl.CurlJSON(ctx, &user, url)
if err != nil {
    return err
}
if resp.StatusCode != 200 {
    return fmt.Errorf("HTTP %d", resp.StatusCode)
}
```

### 3. Assuming JSON Structure
```go
// ❌ Risky - might not match API
type User struct {
    name string `json:"name"` // lowercase = unexported!
}

// ✅ Safe - matches API exactly
type User struct {
    Name string `json:"name"` // Exported, has JSON tag
}
```

---

## Summary

In this chapter, you learned:

✅ **CurlJSON Functions** - Automatic unmarshaling into structs
✅ **Sending JSON** - POST/PUT with JSON() method
✅ **Nested Structures** - Handling complex JSON hierarchies
✅ **Optional Fields** - Using pointers for null values
✅ **Error Handling** - Parsing and returning API errors
✅ **Advanced Patterns** - Generics, pagination, caching
✅ **Real-World Client** - Complete GitHub API integration

**Key Takeaways:**
1. Use CurlJSON for known structures
2. Always pass pointers to CurlJSON
3. Check HTTP status codes
4. Use JSON tags on all struct fields
5. Handle optional fields with pointers
6. Provide context in error messages

---

## What's Next?

**Chapter 7: File Operations**
Learn to upload and download files, handle multipart forms, track progress, and work with large files efficiently.

**Practice Exercises:**
Complete the hands-on exercises in the `exercises/` directory to reinforce your learning.
