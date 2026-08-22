# Exercise 2: Advanced Configuration

**Difficulty:** Intermediate
**Duration:** 45-60 minutes
**Prerequisites:** Exercise 1 completed, Chapter 5 examples

## Objective

Master advanced Builder pattern features including authentication, timeouts, retries, convenience methods, and request templates. Build production-ready request configurations.

## Tasks

### Task 1: Bearer Token Authentication

Build an authenticated request to GitHub API:
- URL: `https://api.github.com/user`
- Use Bearer token authentication
- Add custom User-Agent header

**Requirements:**
- Use `SetBearerToken()` method
- No hardcoded tokens in code (use environment variable)
- Handle 401 Unauthorized errors

**Starter Code:**
```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/options"
)

func main() {
    // Get token from environment
    token := os.Getenv("GITHUB_TOKEN")
    if token == "" {
        fmt.Println("Set GITHUB_TOKEN environment variable")
        return
    }

    // TODO: Build request with Bearer token
    // TODO: Execute and print user info
}
```

**Expected Output:**
```
Status: 200
User: <your-github-username>
```

---

### Task 2: Basic Authentication with Retry

Create a request with Basic Auth and automatic retry:
- URL: `https://httpbin.org/basic-auth/alice/secret123`
- Username: alice
- Password: secret123
- Retry: 3 attempts with 1s delay on 5xx errors

**Requirements:**
- Use `SetBasicAuth()`
- Use `WithDefaultRetry()` or configure custom `RetryConfig`
- Log each retry attempt

**Expected Output:**
```
Attempt 1: Success
Status: 200
Authenticated: true
```

---

### Task 3: Timeout and Context Management

Build a request with proper timeout handling:
- URL: `https://httpbin.org/delay/3`
- Timeout: 5 seconds
- Use `WithTimeout()` method
- Call `Cleanup()` to prevent context leaks

**Requirements:**
- Use builder's `WithTimeout()` method
- Use `defer builder.Cleanup()`
- Pass context from `builder.GetContext()`

**Starter Code:**
```go
func makeRequestWithTimeout(url string, timeout time.Duration) error {
    builder := options.NewRequestOptionsBuilder().
        SetURL(url).
        WithTimeout(timeout)
    defer builder.Cleanup()  // IMPORTANT: Prevent context leak

    // TODO: Build options
    // TODO: Execute with builder.GetContext()
    // TODO: Handle timeout errors

    return nil
}
```

**Expected Output:**
```
✅ Request completed within 5s
Status: 200
```

**Test Timeout:**
```go
// This should timeout
err := makeRequestWithTimeout("https://httpbin.org/delay/10", 2*time.Second)
// Expected: context deadline exceeded
```

---

### Task 4: JSON Convenience Method

Use the `JSON()` convenience method to send structured data:
- Create a User struct
- Marshal to JSON using `JSON()` method
- POST to `https://httpbin.org/post`
- Verify Content-Type header is set automatically

**Starter Code:**
```go
type User struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Age      int    `json:"age"`
    Active   bool   `json:"active"`
}

func main() {
    user := User{
        Name:   "Alice Johnson",
        Email:  "alice@example.com",
        Age:    28,
        Active: true,
    }

    // TODO: Use JSON() convenience method
    // TODO: Verify Content-Type is application/json
}
```

**Expected Output:**
```
Status: 200
Content-Type: application/json (automatic)
Body sent: {"name":"Alice Johnson",...}
```

---

### Task 5: Form Data Submission

Submit form data using the `Form()` convenience method:
- Create form with username, password, remember_me fields
- POST to `https://httpbin.org/post`
- Verify Content-Type is `application/x-www-form-urlencoded`

**Requirements:**
- Use `url.Values` for form data
- Use `Form()` convenience method
- Check Content-Type in response

**Starter Code:**
```go
import "net/url"

func submitForm() error {
    formData := url.Values{}
    formData.Add("username", "alice")
    formData.Add("password", "secret123")
    formData.Add("remember_me", "true")

    // TODO: Use Form() convenience method
    // TODO: Execute and verify

    return nil
}
```

**Expected Output:**
```
Status: 200
Content-Type: application/x-www-form-urlencoded
Form data received: username=alice&password=secret123&remember_me=true
```

---

### Task 6: Request Template Pattern

Build a reusable request template for an API client:
- Create base configuration with common headers
- Build multiple requests from the template
- Each request modifies the template without affecting others

**Requirements:**
- Create base builder with common config
- Reuse builder for multiple endpoints
- Each request should be independent

**Starter Code:**
```go
type APIClient struct {
    baseURL string
    token   string
}

func NewAPIClient(baseURL, token string) *APIClient {
    return &APIClient{baseURL: baseURL, token: token}
}

func (c *APIClient) createBaseBuilder() *options.RequestOptionsBuilder {
    // TODO: Create builder with common configuration
    // - Base URL
    // - Authorization header
    // - User-Agent
    // - Accept: application/json
    return nil
}

func (c *APIClient) GetUser(ctx context.Context, userID string) error {
    // TODO: Use base builder
    // TODO: Modify for this specific endpoint
    return nil
}

func (c *APIClient) CreateUser(ctx context.Context, userData string) error {
    // TODO: Use base builder
    // TODO: Modify for POST request
    return nil
}
```

**Expected Output:**
```
GET /users/123: 200
POST /users: 200
Both requests used same base configuration
```

---

### Task 7: Complete Configuration with Validation

Build a production-ready request with all features and validation:
- URL: `https://httpbin.org/post`
- Method: POST
- Authentication: Bearer token
- Body: JSON object
- Timeout: 30 seconds
- Retry: 3 attempts
- Validation: Call `Validate()` before execution

**Requirements:**
- Use all advanced features
- Validate before execution
- Handle validation errors
- Log request details

**Starter Code:**
```go
func buildProductionRequest() (*options.RequestOptions, error) {
    builder := options.NewRequestOptionsBuilder()

    // TODO: Configure all options
    // - URL, Method
    // - Authentication
    // - JSON body
    // - Timeout
    // - Retry config

    // TODO: Validate
    if err := builder.Validate(); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    return builder.Build(), nil
}
```

**Expected Output:**
```
✅ Validation passed
Request configured with:
  - Bearer authentication
  - 30s timeout
  - 3 retry attempts
  - JSON content type
Status: 200
```

---

## Validation Tests

Create `exercise2_test.go`:

```go
package main

import (
    "context"
    "net/http"
    "net/url"
    "testing"
    "time"

    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/options"
)

func TestTask2_BasicAuthWithRetry(t *testing.T) {
    opts := options.NewRequestOptionsBuilder().
        SetURL("https://httpbin.org/basic-auth/alice/secret123").
        SetBasicAuth("alice", "secret123").
        WithDefaultRetry().
        Build()

    ctx := context.Background()
    resp, err := gocurl.Execute(ctx, opts)
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
}

func TestTask3_TimeoutHandling(t *testing.T) {
    builder := options.NewRequestOptionsBuilder().
        SetURL("https://httpbin.org/delay/2").
        WithTimeout(5 * time.Second)
    defer builder.Cleanup()

    opts := builder.Build()

    ctx := builder.GetContext()
    resp, err := gocurl.Execute(ctx, opts)
    if err != nil {
        t.Fatalf("Request should not timeout: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
}

func TestTask4_JSONMethod(t *testing.T) {
    type TestData struct {
        Name string `json:"name"`
    }

    data := TestData{Name: "Test"}

    opts := options.NewRequestOptionsBuilder().
        SetURL("https://httpbin.org/post").
        SetMethod("POST").
        JSON(data).
        Build()

    ctx := context.Background()
    resp, err := gocurl.Execute(ctx, opts)
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
}

func TestTask5_FormData(t *testing.T) {
    formData := url.Values{}
    formData.Add("username", "testuser")
    formData.Add("password", "testpass")

    opts := options.NewRequestOptionsBuilder().
        SetURL("https://httpbin.org/post").
        SetMethod("POST").
        Form(formData).
        Build()

    ctx := context.Background()
    resp, err := gocurl.Execute(ctx, opts)
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
}

func TestTask7_ValidationPasses(t *testing.T) {
    builder := options.NewRequestOptionsBuilder().
        SetURL("https://httpbin.org/post").
        SetMethod("POST").
        SetBearerToken("test-token").
        JSON(map[string]string{"test": "data"}).
        WithDefaultRetry()

    if err := builder.Validate(); err != nil {
        t.Errorf("Validation should pass: %v", err)
    }
}

func TestValidationFails_InvalidMethod(t *testing.T) {
    builder := options.NewRequestOptionsBuilder().
        SetURL("https://httpbin.org/get").
        SetMethod("INVALID")

    err := builder.Validate()
    if err == nil {
        t.Error("Validation should fail for invalid method")
    }
}
```

Run: `go test -v`

---

## Self-Check Criteria

- ✅ All 7 tasks completed
- ✅ Authentication works correctly
- ✅ Timeouts and retries configured properly
- ✅ Convenience methods used appropriately
- ✅ Request templates are reusable
- ✅ Validation catches errors
- ✅ Context cleanup prevents leaks

## Common Pitfalls

1. **Forgetting Cleanup()**
   ```go
   // ❌ Memory leak
   builder.WithTimeout(5 * time.Second)

   // ✅ Proper cleanup
   builder.WithTimeout(5 * time.Second)
   defer builder.Cleanup()
   ```

2. **Not validating before execution**
   ```go
   // ✅ Always validate complex requests
   if err := builder.Validate(); err != nil {
       return fmt.Errorf("invalid request: %w", err)
   }
   ```

3. **Hardcoding secrets**
   ```go
   // ❌ Don't do this
   SetBearerToken("my-secret-token")

   // ✅ Use environment variables
   SetBearerToken(os.Getenv("API_TOKEN"))
   ```

## Bonus Challenges

1. **Custom Retry Logic:** Implement exponential backoff instead of fixed delay
2. **Multiple Auth Methods:** Support both Bearer and Basic auth in same client
3. **Request Interceptor:** Log all requests before execution
4. **Response Validation:** Check response status and throw custom errors

## Learning Outcomes

After completing this exercise:
- ✅ Configure authentication (Bearer, Basic)
- ✅ Manage timeouts with context
- ✅ Implement retry logic
- ✅ Use convenience methods (JSON, Form)
- ✅ Build reusable request templates
- ✅ Validate requests before execution
- ✅ Handle context cleanup properly

## Next Steps

1. Complete all tasks and run tests
2. Try bonus challenges
3. Proceed to Exercise 3 (Production API Client)
