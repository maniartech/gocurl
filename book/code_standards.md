# GoCurl Book: Code Example Standards

**Version:** 1.0
**Last Updated:** October 17, 2025
**Purpose:** Ensure all code examples are production-quality, runnable, and educational

---

## Core Principles

### 1. Every Example Must Compile

**No pseudocode. No shortcuts. Every example must run.**

```go
// ✅ GOOD - Complete and runnable
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

func main() {
    // CurlString returns (body, resp, err)
    body, resp, err := gocurl.CurlString(context.Background(), "https://api.github.com/zen")
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    fmt.Println(body)
}

// ❌ BAD - Incomplete
body, resp := gocurl.CurlString("https://api.github.com") // Missing imports, ctx, error handling
fmt.Println(body)
```

### 2. Always Handle Errors

**Never use `_` to ignore errors in book examples.**

```go
// ✅ GOOD - Proper error handling
body, resp, err := gocurl.CurlString(ctx, url)
if err != nil {
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()

// ❌ BAD - Ignored errors
body, resp, _ := gocurl.CurlString(ctx, url) // ❌ Never do this in book

// ❌ BAD - Generic error handling
if err != nil {
    panic(err) // ❌ Don't use panic in examples
}
```

### 3. Production-Ready Patterns

**Show how to write real code, not toy examples.**

```go
// ✅ GOOD - Production pattern
type GitHubClient struct {
    token string
    baseURL string
}

func (c *GitHubClient) GetUser(ctx context.Context, username string) (*User, error) {
    url := fmt.Sprintf("%s/users/%s", c.baseURL, username)

    // CurlStringCommand returns (body, resp, err)
    body, resp, err := gocurl.CurlStringCommand(ctx,
        fmt.Sprintf(`curl -H "Authorization: Bearer %s" \
                          -H "Accept: application/vnd.github+json" \
                          %s`, c.token, url))

    if err != nil {
        return nil, fmt.Errorf("failed to fetch user: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
    }

    var user User
    if err := json.Unmarshal([]byte(body), &user); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }

    return &user, nil
}// ❌ BAD - Toy example
func getUser(username string) {
    body, _, _ := gocurl.CurlString(context.Background(), "https://api.github.com/users/"+username)
    fmt.Println(body)
}
```

---

## Code Structure Standards

### Package and Imports

**Always include full package declaration and imports:**

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/maniartech/gocurl"
)

func main() {
    // Code here
}
```

**Group imports (stdlib → external):**

```go
import (
    // Standard library
    "context"
    "fmt"
    "log"

    // External packages
    "github.com/maniartech/gocurl"
    "github.com/joho/godotenv"
)
```

### Function Structure

**Follow this pattern:**

```go
// FunctionName does X and returns Y.
//
// It handles Z edge cases and returns an error if W occurs.
// Example:
//
//     client := NewClient(token)
//     result, err := client.FunctionName(ctx, arg)
//
func FunctionName(ctx context.Context, arg string) (*Result, error) {
    // 1. Validation
    if arg == "" {
        return nil, fmt.Errorf("arg cannot be empty")
    }

    // 2. Setup
    url := buildURL(arg)

    // 3. Request
    resp, body, err := gocurl.Curl(ctx, url)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    // 4. Validation
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
    }

    // 5. Parsing
    var result Result
    if err := json.Unmarshal([]byte(body), &result); err != nil {
        return nil, fmt.Errorf("parse failed: %w", err)
    }

    // 6. Return
    return &result, nil
}
```

### Main Function

**Include complete setup:**

```go
func main() {
    // Parse flags/args if needed
    flag.Parse()

    // Load environment
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }

    // Setup
    token := os.Getenv("GITHUB_TOKEN")
    if token == "" {
        log.Fatal("GITHUB_TOKEN required")
    }

    // Create context
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Run program
    if err := run(ctx, token); err != nil {
        log.Fatalf("Error: %v", err)
    }
}

func run(ctx context.Context, token string) error {
    // Actual logic here
    return nil
}
```

---

## Error Handling Patterns

### Basic Error Handling

```go
body, resp, err := gocurl.CurlString(ctx, url)
if err != nil {
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()
```

### Context Error Handling

```go
body, resp, err := gocurl.CurlString(ctx, url)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        return fmt.Errorf("request timed out: %w", err)
    }
    if errors.Is(err, context.Canceled) {
        return fmt.Errorf("request canceled: %w", err)
    }
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()
```

### HTTP Status Error Handling

```go
body, resp, err := gocurl.CurlString(ctx, url)
if err != nil {
    return nil, fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()

switch {
case resp.StatusCode == 200:
    // Success
case resp.StatusCode == 404:
    return nil, fmt.Errorf("resource not found")
case resp.StatusCode >= 500:
    return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, body)
case resp.StatusCode >= 400:
    return nil, fmt.Errorf("client error (%d): %s", resp.StatusCode, body)
default:
    return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
}
```

### JSON Parsing Errors

```go
var result Result
if err := json.Unmarshal([]byte(body), &result); err != nil {
    return nil, fmt.Errorf("failed to parse JSON: %w\nBody: %s", err, body)
}
```

---

## Testing Standards

### Every Example Gets a Test

```go
// Example code
func GetUser(ctx context.Context, username string) (*User, error) {
    // ... implementation
}

// Test for the example
func TestGetUser(t *testing.T) {
    tests := []struct {
        name     string
        username string
        want     *User
        wantErr  bool
    }{
        {
            name:     "valid user",
            username: "octocat",
            want:     &User{Login: "octocat"},
            wantErr:  false,
        },
        {
            name:     "empty username",
            username: "",
            want:     nil,
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.Background()
            got, err := GetUser(ctx, tt.username)

            if (err != nil) != tt.wantErr {
                t.Errorf("GetUser() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("GetUser() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

---

## Documentation Standards

### Function Comments

```go
// GetUser fetches a GitHub user by username.
//
// It makes an authenticated request to the GitHub API and returns
// the user's profile information. Returns an error if the user
// is not found or if the request fails.
//
// Example:
//
//     user, err := GetUser(ctx, "octocat")
//     if err != nil {
//         log.Fatal(err)
//     }
//     fmt.Printf("Name: %s\n", user.Name)
//
func GetUser(ctx context.Context, username string) (*User, error) {
    // Implementation
}
```

### Type Comments

```go
// User represents a GitHub user profile.
type User struct {
    // Login is the GitHub username.
    Login string `json:"login"`

    // Name is the user's full name.
    Name string `json:"name"`

    // Email is the user's public email address.
    // May be empty if the user has not set a public email.
    Email string `json:"email"`
}
```

### Inline Comments

```go
// 🎯 Always use context with timeout for production code
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// ⚠️ Don't forget to close the response body to prevent resource leaks
defer resp.Body.Close()

// 💡 Use variables for better readability and debugging
url := fmt.Sprintf("https://api.github.com/users/%s", username)
```

---

## Security Standards

### Never Hardcode Credentials

```go
// ✅ GOOD - Environment variables
token := os.Getenv("GITHUB_TOKEN")
if token == "" {
    return fmt.Errorf("GITHUB_TOKEN environment variable required")
}

// ❌ BAD - Hardcoded
token := "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxx" // ❌ Never do this
```

### Sanitize Logging

```go
// ✅ GOOD - Redacted sensitive data
log.Printf("Making request to %s with token=[REDACTED]", url)

// ❌ BAD - Logging credentials
log.Printf("Making request with token=%s", token) // ❌ Security risk
```

### Validate Input

```go
// ✅ GOOD - Input validation
func GetUser(ctx context.Context, username string) (*User, error) {
    if username == "" {
        return nil, fmt.Errorf("username cannot be empty")
    }

    if len(username) > 39 {
        return nil, fmt.Errorf("username too long (max 39 characters)")
    }

    // Validate username format
    if !isValidUsername(username) {
        return nil, fmt.Errorf("invalid username format")
    }

    // ... rest of implementation
}
```

---

## Performance Standards

### Use Context Timeouts

```go
// ✅ GOOD - With timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

body, resp, err := gocurl.CurlString(ctx, url)

// ❌ BAD - No timeout (can hang forever)
body, resp, err := gocurl.CurlString(context.Background(), url)
```

### Close Response Bodies

```go
// ✅ GOOD - Always defer close
body, resp, err := gocurl.CurlString(ctx, url)
if err != nil {
    return err
}
defer resp.Body.Close() // 🎯 Critical for preventing leaks

// ❌ BAD - Forgot to close
body, resp, err := gocurl.CurlString(ctx, url)
if err != nil {
    return err
}
// Missing defer resp.Body.Close() - resource leak!
```

### Reuse Clients

```go
// ✅ GOOD - Reusable client
type APIClient struct {
    baseURL string
    token   string
}

func (c *APIClient) request(ctx context.Context, endpoint string) (string, error) {
    url := c.baseURL + endpoint
    // ... make request
}

// ❌ BAD - Creating new client each time (unnecessary overhead)
func makeRequest(endpoint string) (string, error) {
    client := &APIClient{baseURL: "https://api.example.com"}
    return client.request(context.Background(), endpoint)
}
```

---

## Example Organization

### File Structure for Examples

```
chapter01/
├── 01-first-request/
│   ├── main.go
│   ├── main_test.go
│   └── README.md
├── 02-with-headers/
│   ├── main.go
│   ├── main_test.go
│   └── README.md
└── 03-complete-client/
    ├── main.go
    ├── client.go
    ├── client_test.go
    └── README.md
```

### Example README Template

```markdown
# Example: First Request

## What You'll Learn

- How to make a simple GET request
- How to handle responses
- Basic error handling

## Prerequisites

- GoCurl installed
- Go 1.21+

## Running the Example

```bash
go run main.go
```

## Expected Output

```
Status: 200
Body: Hello, World!
```

## Code Explanation

[Explain what the code does step by step]

## Common Issues

- **Error: context deadline exceeded** - Increase timeout
- **Error: connection refused** - Check URL

## Next Steps

- Try with different URLs
- Add headers
- Handle different status codes
```

---

## Quality Checklist

Before including any code example in the book:

### Compilation
- [ ] Code compiles without errors
- [ ] All imports are correct
- [ ] Package declaration is present
- [ ] No unused variables or imports

### Functionality
- [ ] Code runs successfully
- [ ] Produces expected output
- [ ] Handles errors properly
- [ ] Resource cleanup (defer close)

### Style
- [ ] Follows Go conventions
- [ ] Proper formatting (gofmt)
- [ ] Clear variable names
- [ ] Appropriate comments

### Documentation
- [ ] Function comments present
- [ ] Inline comments explain non-obvious parts
- [ ] Example usage shown
- [ ] README explains example

### Security
- [ ] No hardcoded credentials
- [ ] Input validation present
- [ ] Sensitive data not logged
- [ ] Uses environment variables

### Testing
- [ ] Test file exists
- [ ] Tests pass
- [ ] Edge cases covered
- [ ] Table-driven tests used

### Performance
- [ ] Context with timeout
- [ ] Response body closed
- [ ] No unnecessary allocations
- [ ] Efficient patterns used

---

## Common Patterns Library

### Pattern 1: Simple GET Request

```go
func getJSON(ctx context.Context, url string, result interface{}) error {
    body, resp, err := gocurl.CurlString(ctx, url)
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
    }

    if err := json.Unmarshal([]byte(body), result); err != nil {
        return fmt.Errorf("parse failed: %w", err)
    }

    return nil
}
```

### Pattern 2: POST with JSON

```go
func postJSON(ctx context.Context, url string, payload interface{}) (*Response, error) {
    data, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("marshal failed: %w", err)
    }

    body, resp, err := gocurl.CurlStringCommand(ctx,
        fmt.Sprintf(`curl -X POST \
            -H "Content-Type: application/json" \
            -d '%s' \
            %s`, string(data), url))

    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    var response Response
    if err := json.Unmarshal([]byte(body), &response); err != nil {
        return nil, fmt.Errorf("parse failed: %w", err)
    }

    return &response, nil
}
```

### Pattern 3: With Authentication

```go
func authenticatedRequest(ctx context.Context, token, url string) (string, error) {
    body, resp, err := gocurl.CurlStringCommand(ctx,
        fmt.Sprintf(`curl -H "Authorization: Bearer %s" %s`, token, url))

    if err != nil {
        return "", fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == 401 {
        return "", fmt.Errorf("authentication failed: invalid token")
    }

    if resp.StatusCode != 200 {
        return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
    }

    return body, nil
}
```

### Pattern 4: With Retry

```go
func requestWithRetry(ctx context.Context, url string, maxRetries int) (string, error) {
    var lastErr error

    for attempt := 0; attempt <= maxRetries; attempt++ {
        if attempt > 0 {
            // Exponential backoff
            backoff := time.Duration(1<<uint(attempt-1)) * time.Second
            select {
            case <-time.After(backoff):
            case <-ctx.Done():
                return "", ctx.Err()
            }
        }

        body, resp, err := gocurl.CurlString(ctx, url)
        if err != nil {
            lastErr = err
            continue
        }
        defer resp.Body.Close()

        if resp.StatusCode == 200 {
            return body, nil
        }

        if resp.StatusCode < 500 {
            // Client error, don't retry
            return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
        }

        // Server error, will retry
        lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
    }

    return "", fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}
```---

## Final Notes

**Remember:**

1. Every example must run
2. Every example must handle errors
3. Every example must be production-ready
4. Every example must teach something
5. Every example must be tested

**Code is teaching material. Make it perfect.**

---

**Last Updated:** October 17, 2025
**Version:** 1.0
**Next Review:** Before each chapter
