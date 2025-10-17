# Code Standards
## The Definitive Guide to the GoCurl Library

**Version:** 1.0
**Last Updated:** 2025
**Purpose:** Quality standards for all code examples in the book

---

## 1. Overview

Every code example in this book must meet these standards. No exceptions.

**Guiding Principles:**
1. **Every example must compile** - No syntax errors, no placeholders
2. **Every example must run** - No runtime errors, no panics
3. **Every example must demonstrate** - Show the concept being taught
4. **Every example must be tested** - Verify output matches expectations
5. **Every example must be idiomatic** - Follow Go best practices

---

## 2. Compilation Requirements

### 2.1 All Examples Must Compile

**Mandatory:**
- [ ] Code compiles with `go build` without errors
- [ ] Code compiles with `go vet` without warnings
- [ ] Code formatted with `gofmt` (no manual formatting)
- [ ] Imports are complete and correct
- [ ] Package declarations are appropriate

**Example Verification:**

```bash
# Every code example must pass these checks
go build ./...
go vet ./...
gofmt -l . | wc -l  # Should be 0
```

### 2.2 No Placeholders

**NEVER use placeholders:**

❌ **WRONG:**
```go
// TODO: Add error handling
resp, _ := gocurl.Curl(ctx, url)
```

❌ **WRONG:**
```go
// ... existing code ...
opts.AddHeader("Authorization", token)
// ... more code ...
```

❌ **WRONG:**
```go
// Implementation goes here
func ProcessResponse(resp *http.Response) error {
    return nil
}
```

✅ **CORRECT:**
```go
resp, err := gocurl.Curl(ctx, url)
if err != nil {
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()
```

### 2.3 Import Requirements

**Complete imports for every example:**

✅ **Minimal Example (< 10 lines):**
```go
resp, err := gocurl.Curl(ctx, "https://api.github.com")
if err != nil {
    return err
}
defer resp.Body.Close()
```
*Context: Can omit imports if they're obvious from surrounding text*

✅ **Complete Example (10-50 lines):**
```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/stackql/gocurl"
)

func main() {
    ctx := context.Background()

    resp, err := gocurl.Curl(ctx, "https://api.github.com")
    if err != nil {
        log.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    fmt.Println("Status:", resp.StatusCode)
}
```

✅ **Listing Example (> 50 lines):**
```go
// Listing 5-1: Complete API Client with Builder Pattern
package apiclient

import (
    "context"
    "fmt"
    "time"

    "github.com/stackql/gocurl"
)

type Client struct {
    baseURL string
    token   string
}

func NewClient(baseURL, token string) *Client {
    return &Client{
        baseURL: baseURL,
        token:   token,
    }
}

func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
    url := c.baseURL + path

    opts := gocurl.NewRequestOptionsBuilder().
        SetMethod("GET").
        SetURL(url).
        AddHeader("Authorization", "Bearer "+c.token).
        SetTimeout(10 * time.Second).
        Build()

    return gocurl.Process(ctx, opts)
}
```

---

## 3. Error Handling Standards

### 3.1 Always Handle Errors

**Every function that returns an error must have error handling.**

❌ **WRONG:**
```go
resp, _ := gocurl.Curl(ctx, url)  // NEVER ignore errors
```

❌ **WRONG:**
```go
resp, err := gocurl.Curl(ctx, url)
// Forgot to check error
defer resp.Body.Close()  // Will panic if err != nil
```

✅ **CORRECT:**
```go
resp, err := gocurl.Curl(ctx, url)
if err != nil {
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()
```

### 3.2 Error Wrapping

**Use error wrapping (`%w`) to preserve error chain:**

❌ **WRONG:**
```go
if err != nil {
    return fmt.Errorf("failed: %v", err)  // Loses error chain
}
```

✅ **CORRECT:**
```go
if err != nil {
    return fmt.Errorf("failed: %w", err)  // Preserves error chain
}
```

### 3.3 Context-Specific Error Messages

**Provide context in error messages:**

❌ **WRONG:**
```go
if err != nil {
    return err  // No context
}
```

❌ **WRONG:**
```go
if err != nil {
    return fmt.Errorf("error: %w", err)  // Generic message
}
```

✅ **CORRECT:**
```go
if err != nil {
    return fmt.Errorf("failed to fetch user data: %w", err)
}
```

✅ **CORRECT:**
```go
if resp.StatusCode != 200 {
    return fmt.Errorf("unexpected status %d: expected 200", resp.StatusCode)
}
```

---

## 4. Resource Management

### 4.1 Always Close Response Bodies

**EVERY `http.Response` must be closed with `defer`:**

❌ **WRONG:**
```go
resp, err := gocurl.Curl(ctx, url)
if err != nil {
    return err
}
// Forgot defer - RESOURCE LEAK
```

✅ **CORRECT:**
```go
resp, err := gocurl.Curl(ctx, url)
if err != nil {
    return err
}
defer resp.Body.Close()
```

### 4.2 Check Error Before Defer

**Only defer if response is non-nil:**

❌ **WRONG:**
```go
resp, err := gocurl.Curl(ctx, url)
defer resp.Body.Close()  // Panic if resp is nil!
if err != nil {
    return err
}
```

✅ **CORRECT:**
```go
resp, err := gocurl.Curl(ctx, url)
if err != nil {
    return err
}
defer resp.Body.Close()  // Safe: resp is non-nil
```

### 4.3 Context Cleanup

**Clean up contexts when using builders:**

✅ **CORRECT:**
```go
builder := gocurl.NewRequestOptionsBuilder()
defer builder.Cleanup()  // Always cleanup

opts := builder.
    WithTimeout(10 * time.Second).
    SetURL(url).
    Build()

resp, err := gocurl.Process(ctx, opts)
```

---

## 5. API Signature Correctness

### 5.1 Verified Signatures Only

**ALL function calls must match actual gocurl API signatures.**

**Reference:** See `API_REFERENCE.md` for complete signatures.

### 5.2 Return Value Order

**Critical:** String/Bytes functions return body FIRST.

❌ **WRONG:**
```go
resp, body, err := gocurl.CurlString(ctx, url)  // WRONG ORDER
```

✅ **CORRECT:**
```go
body, resp, err := gocurl.CurlString(ctx, url)  // Body FIRST
```

### 5.3 Return Value Count

**Each function category has specific return count:**

| Category | Returns | Example |
|----------|---------|---------|
| Basic | 2 | `resp, err := Curl(...)` |
| String | 3 | `body, resp, err := CurlString(...)` |
| Bytes | 3 | `data, resp, err := CurlBytes(...)` |
| JSON | 2 | `resp, err := CurlJSON(ctx, &v, ...)` |
| Download | 3 | `n, resp, err := CurlDownload(...)` |
| WithVars | 2 | `resp, err := CurlWithVars(...)` |

---

## 6. Testing Standards

### 6.1 All Examples Must Be Tested

**Before including in book:**
1. Create test file for the example
2. Run the test
3. Verify output matches expectations
4. Check for race conditions (`go test -race`)

### 6.2 Test File Organization

```
book2/
  part1-foundations/
    chapter01-why-gocurl/
      examples/
        github_client.go          # Example code
        github_client_test.go     # Test for example
      exercises/
        solution_1.go
        solution_1_test.go
```

### 6.3 Example Test Template

```go
package examples

import (
    "context"
    "testing"

    "github.com/stackql/gocurl"
)

func TestGitHubClient(t *testing.T) {
    ctx := context.Background()

    resp, err := gocurl.Curl(ctx, "https://api.github.com")
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
}
```

### 6.4 Race Condition Testing

**All concurrent examples must pass race detector:**

```bash
go test -race ./...
```

**Example with concurrency:**
```go
func TestConcurrentRequests(t *testing.T) {
    ctx := context.Background()
    baseOpts := gocurl.NewRequestOptions("https://api.example.com")

    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()

            // MUST clone before modification
            opts := baseOpts.Clone()
            opts.AddQueryParam("id", fmt.Sprintf("%d", id))

            resp, err := gocurl.Process(ctx, opts)
            if err != nil {
                t.Errorf("Request %d failed: %v", id, err)
                return
            }
            defer resp.Body.Close()
        }(i)
    }
    wg.Wait()
}
```

---

## 7. Code Style Standards

### 7.1 Go Formatting

**All code must be formatted with `gofmt`:**

```bash
gofmt -w .
```

**No manual formatting:**
- Use tabs for indentation (automatic with gofmt)
- No trailing whitespace
- Consistent spacing around operators

### 7.2 Naming Conventions

**Follow Go naming conventions:**

✅ **CORRECT:**
```go
// Variables: camelCase
var apiClient *Client
var baseURL string

// Constants: PascalCase or ALL_CAPS for exported
const MaxRetries = 3
const defaultTimeout = 10 * time.Second

// Functions: PascalCase for exported, camelCase for private
func NewClient() *Client
func (c *Client) GetUser() (*User, error)
func processResponse(resp *http.Response) error

// Structs: PascalCase
type GitHubClient struct
type RequestConfig struct

// Interfaces: PascalCase, prefer "-er" suffix
type HTTPClient interface
type Processor interface
```

### 7.3 Variable Declaration

**Use short declarations when possible:**

✅ **CORRECT:**
```go
resp, err := gocurl.Curl(ctx, url)
```

❌ **WRONG (verbose):**
```go
var resp *http.Response
var err error
resp, err = gocurl.Curl(ctx, url)
```

**Use explicit type when clarity is needed:**

✅ **CORRECT:**
```go
var timeout time.Duration = 10 * time.Second
```

### 7.4 Comments

**Write clear, concise comments:**

✅ **CORRECT:**
```go
// FetchUser retrieves user information from the API
func FetchUser(ctx context.Context, id string) (*User, error) {
    // Build request with retry configuration
    opts := gocurl.NewRequestOptionsBuilder().
        SetURL(fmt.Sprintf("https://api.example.com/users/%s", id)).
        SetRetryConfig(&gocurl.RetryConfig{
            MaxRetries: 3,
            RetryDelay: time.Second,
        }).
        Build()

    return executeRequest(ctx, opts)
}
```

❌ **WRONG (obvious comments):**
```go
// Declare context variable
ctx := context.Background()  // Create context

// Call Curl function
resp, err := gocurl.Curl(ctx, url)  // Make HTTP request

// Check if error is not nil
if err != nil {  // Error happened
    return err  // Return the error
}
```

---

## 8. Example Categories

### 8.1 Minimal Examples (< 10 lines)

**Purpose:** Show single concept quickly
**Location:** Within chapter text, inline with explanation
**Requirements:**
- Must compile (with implied context)
- Focus on one concept
- Can omit imports if obvious

**Example:**
```go
// Simple GET request
resp, err := gocurl.Curl(ctx, "https://api.github.com")
if err != nil {
    return err
}
defer resp.Body.Close()
```

### 8.2 Complete Examples (10-50 lines)

**Purpose:** Show concept in realistic context
**Location:** End of sections, demonstrating multiple related concepts
**Requirements:**
- Complete package declaration
- All imports
- Full error handling
- Can be copied and run

**Example:**
```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/stackql/gocurl"
)

func main() {
    ctx := context.Background()

    var user struct {
        Login string `json:"login"`
        Name  string `json:"name"`
    }

    resp, err := gocurl.CurlJSON(ctx, &user, "https://api.github.com/users/octocat")
    if err != nil {
        log.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        log.Fatalf("Unexpected status: %d", resp.StatusCode)
    }

    fmt.Printf("%s (%s)\n", user.Name, user.Login)
}
```

### 8.3 Listings (> 50 lines)

**Purpose:** Complete, production-ready implementations
**Location:** Hands-on projects, case studies
**Requirements:**
- Full package structure
- Complete documentation
- Production error handling
- Tests included
- Can be used in real projects

**Example:**
```go
// Listing 8-1: Production API Client with Retries and Tracing
package apiclient

import (
    "context"
    "fmt"
    "time"

    "github.com/stackql/gocurl"
)

// Client provides a production-ready API client
type Client struct {
    baseURL string
    token   string
}

// NewClient creates a new API client
func NewClient(baseURL, token string) *Client {
    return &Client{
        baseURL: baseURL,
        token:   token,
    }
}

// Get performs a GET request with retries and tracing
func (c *Client) Get(ctx context.Context, path string, requestID string) (*http.Response, error) {
    url := c.baseURL + path

    opts := gocurl.NewRequestOptionsBuilder().
        SetMethod("GET").
        SetURL(url).
        AddHeader("Authorization", "Bearer "+c.token).
        AddHeader("X-Request-ID", requestID).
        SetTimeout(10 * time.Second).
        SetRetryConfig(&gocurl.RetryConfig{
            MaxRetries:  3,
            RetryDelay:  time.Second,
            RetryOnHTTP: []int{500, 502, 503, 504},
        }).
        SetFollowRedirects(true).
        SetMaxRedirects(5).
        Build()

    resp, err := gocurl.Process(ctx, opts)
    if err != nil {
        return nil, fmt.Errorf("GET %s failed: %w", path, err)
    }

    return resp, nil
}

// Post performs a POST request with JSON body
func (c *Client) Post(ctx context.Context, path, body, requestID string) (*http.Response, error) {
    url := c.baseURL + path

    opts := gocurl.NewRequestOptionsBuilder().
        SetMethod("POST").
        SetURL(url).
        AddHeader("Authorization", "Bearer "+c.token).
        AddHeader("Content-Type", "application/json").
        AddHeader("X-Request-ID", requestID).
        SetBody(body).
        SetTimeout(15 * time.Second).
        SetRetryConfig(&gocurl.RetryConfig{
            MaxRetries:  3,
            RetryDelay:  time.Second,
            RetryOnHTTP: []int{500, 502, 503, 504},
        }).
        Build()

    resp, err := gocurl.Process(ctx, opts)
    if err != nil {
        return nil, fmt.Errorf("POST %s failed: %w", path, err)
    }

    return resp, nil
}
```

---

## 9. Real vs. Mock Examples

### 9.1 Real API Examples

**Preferred for demonstrations:**
- Use real, public APIs (GitHub, JSONPlaceholder, etc.)
- Show actual output
- Demonstrate real-world usage

**Approved Public APIs:**
- https://api.github.com - GitHub API (no auth needed for public data)
- https://jsonplaceholder.typicode.com - Fake REST API for testing
- https://httpbin.org - HTTP testing service
- https://api.openweathermap.org - Weather data (free tier)

**Example:**
```go
// Using real GitHub API
var user struct {
    Login       string `json:"login"`
    PublicRepos int    `json:"public_repos"`
}

resp, err := gocurl.CurlJSON(ctx, &user, "https://api.github.com/users/octocat")
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Printf("User %s has %d public repos\n", user.Login, user.PublicRepos)
// Output: User octocat has 8 public repos
```

### 9.2 Test Server Examples

**For testing-specific examples:**
- Use `httptest.NewServer`
- Show testing patterns
- Demonstrate mocking

**Example:**
```go
func TestClientRetry(t *testing.T) {
    attempts := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        attempts++
        if attempts < 3 {
            w.WriteHeader(http.StatusServiceUnavailable)
            return
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok"}`))
    }))
    defer server.Close()

    opts := gocurl.NewRequestOptions(server.URL)
    opts.RetryConfig = &gocurl.RetryConfig{
        MaxRetries:  3,
        RetryDelay:  100 * time.Millisecond,
        RetryOnHTTP: []int{503},
    }

    resp, err := gocurl.Process(context.Background(), opts)
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    if attempts != 3 {
        t.Errorf("Expected 3 attempts, got %d", attempts)
    }
}
```

---

## 10. Verification Checklist

Before including ANY code example in the book, verify:

### 10.1 Compilation
- [ ] Code compiles with `go build`
- [ ] No `go vet` warnings
- [ ] Formatted with `gofmt`
- [ ] All imports present

### 10.2 Correctness
- [ ] API signatures match `API_REFERENCE.md`
- [ ] Return values in correct order
- [ ] Return value count is correct
- [ ] Error handling is complete

### 10.3 Resource Management
- [ ] Response bodies are closed
- [ ] Contexts are cleaned up
- [ ] No resource leaks

### 10.4 Testing
- [ ] Example has been tested
- [ ] Test passes with `go test`
- [ ] No race conditions (`go test -race`)
- [ ] Output matches expectations

### 10.5 Style
- [ ] Follows Go conventions
- [ ] Clear variable names
- [ ] Appropriate comments
- [ ] Consistent formatting

### 10.6 Documentation
- [ ] Example demonstrates stated concept
- [ ] Comments explain WHY, not WHAT
- [ ] Complex logic is explained
- [ ] Real-world applicable

---

## 11. Common Mistakes to Avoid

### 11.1 Ignoring Errors

❌ **NEVER:**
```go
resp, _ := gocurl.Curl(ctx, url)
```

### 11.2 Forgetting defer

❌ **NEVER:**
```go
resp, err := gocurl.Curl(ctx, url)
if err != nil {
    return err
}
// Missing: defer resp.Body.Close()
```

### 11.3 Wrong Return Order

❌ **NEVER:**
```go
resp, body, err := gocurl.CurlString(ctx, url)  // WRONG
```

### 11.4 Using Placeholders

❌ **NEVER:**
```go
// TODO: Implement
// ... code here ...
```

### 11.5 Missing Imports

❌ **NEVER:**
```go
package main

func main() {
    // Missing imports!
    resp, err := gocurl.Curl(ctx, url)
}
```

### 11.6 Ignoring Context

❌ **NEVER:**
```go
// Don't use background context when timeout is needed
ctx := context.Background()  // Will never timeout!
resp, err := gocurl.Curl(ctx, url)
```

✅ **CORRECT:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
resp, err := gocurl.Curl(ctx, url)
```

---

## 12. Quality Metrics

### 12.1 Target Metrics

**Every chapter should achieve:**
- ✅ 100% of examples compile
- ✅ 100% of examples tested
- ✅ 100% of API signatures correct
- ✅ Zero placeholders or TODOs
- ✅ Zero resource leaks
- ✅ Zero race conditions

### 12.2 Automated Checks

```bash
#!/bin/bash
# scripts/verify-examples.sh

echo "Checking compilation..."
go build ./... || exit 1

echo "Checking formatting..."
UNFORMATTED=$(gofmt -l .)
if [ -n "$UNFORMATTED" ]; then
    echo "ERROR: Unformatted files:"
    echo "$UNFORMATTED"
    exit 1
fi

echo "Running tests..."
go test ./... || exit 1

echo "Checking for race conditions..."
go test -race ./... || exit 1

echo "Running go vet..."
go vet ./... || exit 1

echo "✓ All checks passed"
```

---

## 13. Example Repository Structure

```
book2/
  examples/                 # All runnable examples
    chapter01/
      quick_start.go
      quick_start_test.go
    chapter02/
      installation_test.go
    chapter03/
      variable_expansion.go
      context_patterns.go
      response_handling.go
    ...

  listings/                 # Production-quality listings
    chapter01/
      listing_1_1_github_client.go
      listing_1_1_test.go
    ...

  go.mod                    # Module definition
  go.sum                    # Dependencies
  scripts/
    verify-examples.sh      # Automated verification
    test-all.sh             # Run all tests
```

---

## 14. Continuous Integration

### 14.1 Pre-Commit Checks

Before committing any code:

```bash
# 1. Format all code
gofmt -w .

# 2. Build everything
go build ./...

# 3. Run all tests
go test ./...

# 4. Check for race conditions
go test -race ./...

# 5. Run go vet
go vet ./...
```

### 14.2 CI Pipeline

```yaml
# .github/workflows/verify.yml
name: Verify Examples

on: [push, pull_request]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Format check
        run: |
          gofmt -l . | grep . && exit 1 || echo "OK"

      - name: Build
        run: go build ./...

      - name: Test
        run: go test -v ./...

      - name: Race detection
        run: go test -race ./...

      - name: Vet
        run: go vet ./...
```

---

**END OF CODE STANDARDS**
