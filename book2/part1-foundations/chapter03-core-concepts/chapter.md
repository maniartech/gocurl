# Chapter 3: Core Concepts

> **Learning Objectives**
>
> By the end of this chapter, you will:
> - Understand gocurl's dual API approach (Curl-syntax vs Builder pattern)
> - Master all six function categories and know when to use each
> - Use variable expansion securely and effectively
> - Implement proper context handling for timeouts and cancellation
> - Parse responses using multiple techniques
> - Understand the Process() function as gocurl's core execution engine

## Introduction

GoCurl's architecture is built on a simple but powerful principle: **provide multiple ways to accomplish the same task, letting developers choose the approach that best fits their use case**.

This chapter explores gocurl's core concepts in depth. You'll learn how the library transforms curl commands into HTTP requests, how the six function categories serve different needs, and how to leverage context, variables, and response handling effectively.

By mastering these concepts, you'll be able to confidently choose the right tool for any HTTP client scenario.

## The Dual API Approach

GoCurl offers two distinct ways to make HTTP requests:

1. **Curl-Syntax Functions** - String-based commands using curl syntax
2. **Programmatic Builder** - Type-safe RequestOptions and Builder pattern

Both approaches ultimately call the same core `Process()` function, but they serve different use cases.

### Approach 1: Curl-Syntax Functions

The curl-syntax approach lets you use curl commands directly in Go code:

```go
body, resp, err := gocurl.CurlStringCommand(ctx,
    `curl -H "Authorization: Bearer token" \
          -H "Accept: application/json" \
          https://api.example.com/data`)
```

**When to use:**
- Rapid prototyping and exploration
- Converting API documentation examples
- CLI tools and scripts
- One-off requests or simple integrations

**Benefits:**
- Zero translation from API docs to code
- Familiar curl syntax
- Quick to write and iterate
- Test with CLI, copy to code

**Limitations:**
- String-based (no compile-time type checking)
- Less IDE support (no autocompletion for flags)
- Harder to generate requests dynamically

### Approach 2: Programmatic Builder

The builder approach uses structs and methods for type-safe request construction:

```go
opts := options.NewRequestOptionsBuilder().
    SetURL("https://api.example.com/data").
    SetHeader("Authorization", "Bearer token").
    SetHeader("Accept", "application/json").
    Build()

httpResp, _, err := gocurl.Process(ctx, opts)
```

**When to use:**
- Building SDKs and libraries
- Complex, dynamic requests
- Request templates (using Clone())
- Enterprise applications
- Type-safe configurations

**Benefits:**
- Full type safety
- IDE autocompletion
- Easy to generate requests programmatically
- Validation before execution
- Reusable with Clone()

**Limitations:**
- More verbose
- Requires understanding RequestOptions struct
- Less direct mapping from curl commands

### How They Relate

Here's the internal flow:

```
Curl Command String
        ↓
   Tokenization
        ↓
   Parse Tokens
        ↓
 RequestOptions ← Builder Pattern
        ↓
   Process()
        ↓
  http.Client
        ↓
 *http.Response
```

Both paths converge at `Process()`, which is the core execution engine. This means:
- Same performance characteristics
- Same features available
- Same error handling
- Choose based on developer ergonomics, not technical limitations

## Understanding the Six Function Categories

GoCurl provides **six function categories**, each optimized for different response handling needs. Each category has three variants: basic, Command, and Args.

### Category 1: Basic Functions (Curl, CurlCommand, CurlArgs)

**Returns:** `(*http.Response, error)`

These functions return the raw HTTP response. You read the body manually.

```go
// Basic URL
resp, err := gocurl.Curl(ctx, "https://api.example.com/data")

// Command-style (multi-line curl command)
resp, err := gocurl.CurlCommand(ctx,
    `curl -H "Accept: application/json" https://api.example.com/data`)

// Args-style (individual arguments)
resp, err := gocurl.CurlArgs(ctx,
    "-H", "Accept: application/json",
    "https://api.example.com/data")
```

**Use when:**
- You need access to raw response body stream
- Implementing custom parsing logic
- Streaming responses
- HEAD requests (don't need body)

**Example:**
```go
resp, err := gocurl.Curl(ctx, "-I", "https://example.com")
if err != nil {
    return err
}
defer resp.Body.Close()

// Just check headers, don't read body
contentType := resp.Header.Get("Content-Type")
lastModified := resp.Header.Get("Last-Modified")
```

### Category 2: String Functions (CurlString, CurlStringCommand, CurlStringArgs)

**Returns:** `(string, *http.Response, error)`

These functions automatically read the response body as a string.

```go
// Returns body as string
body, resp, err := gocurl.CurlString(ctx, "https://api.example.com/data")

body, resp, err := gocurl.CurlStringCommand(ctx,
    `curl https://api.example.com/data`)

body, resp, err := gocurl.CurlStringArgs(ctx,
    "https://api.example.com/data")
```

**Use when:**
- Response is text (HTML, XML, plain text)
- You want the body as a string immediately
- Simple response processing
- Logging or displaying responses

**Example:**
```go
body, resp, err := gocurl.CurlString(ctx, "https://api.github.com/zen")
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Println("GitHub Zen:", body) // Already a string
```

### Category 3: Bytes Functions (CurlBytes, CurlBytesCommand, CurlBytesArgs)

**Returns:** `([]byte, *http.Response, error)`

These functions automatically read the response body as bytes.

```go
// Returns body as []byte
bodyBytes, resp, err := gocurl.CurlBytes(ctx, "https://example.com/image.png")

bodyBytes, resp, err := gocurl.CurlBytesCommand(ctx,
    `curl https://example.com/image.png`)

bodyBytes, resp, err := gocurl.CurlBytesArgs(ctx,
    "https://example.com/image.png")
```

**Use when:**
- Binary data (images, PDFs, archives)
- Need []byte for further processing
- Writing to disk or other destinations
- Binary protocol responses

**Example:**
```go
imageBytes, resp, err := gocurl.CurlBytes(ctx,
    "https://example.com/logo.png")
if err != nil {
    return err
}
defer resp.Body.Close()

// Save to file
err = os.WriteFile("logo.png", imageBytes, 0644)
```

### Category 4: JSON Functions (CurlJSON, CurlJSONCommand, CurlJSONArgs)

**Returns:** `(*http.Response, error)`

These functions automatically unmarshal JSON responses into your struct.

```go
// Define target struct
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Automatically unmarshals into user
var user User
resp, err := gocurl.CurlJSON(ctx, &user, "https://api.example.com/user/123")

resp, err := gocurl.CurlJSONCommand(ctx, &user,
    `curl -H "Accept: application/json" https://api.example.com/user/123`)

resp, err := gocurl.CurlJSONArgs(ctx, &user,
    "-H", "Accept: application/json",
    "https://api.example.com/user/123")
```

**Use when:**
- API returns JSON
- You have type definitions
- Want type-safe response handling
- Need structured data

**Example:**
```go
type Repository struct {
    Name  string `json:"name"`
    Stars int    `json:"stargazers_count"`
}

var repo Repository
resp, err := gocurl.CurlJSON(ctx, &repo,
    "https://api.github.com/repos/golang/go")
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Printf("Repository: %s\n", repo.Name)
fmt.Printf("Stars: %d\n", repo.Stars)
```

### Category 5: Download Functions (CurlDownload, CurlDownloadCommand, CurlDownloadArgs)

**Returns:** `(int64, *http.Response, error)`

These functions stream the response directly to a file.

```go
// Downloads to file, returns bytes written
written, resp, err := gocurl.CurlDownload(ctx,
    "/tmp/output.json",
    "https://api.example.com/large-dataset")

written, resp, err := gocurl.CurlDownloadCommand(ctx,
    "/tmp/output.json",
    `curl https://api.example.com/large-dataset`)

written, resp, err := gocurl.CurlDownloadArgs(ctx,
    "/tmp/output.json",
    "https://api.example.com/large-dataset")
```

**Use when:**
- Large files (don't want in memory)
- Downloads that need to persist
- Bandwidth-efficient transfers
- Progress tracking (with middleware)

**Example:**
```go
written, resp, err := gocurl.CurlDownload(ctx,
    "/tmp/golang-1.21.tar.gz",
    "https://go.dev/dl/go1.21.linux-amd64.tar.gz")
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Printf("Downloaded %d bytes\n", written)
```

### Category 6: WithVars Functions (Explicit Variable Control)

**Returns:** `(*http.Response, error)` (same as basic functions)

These functions use explicit variable maps instead of environment expansion.

```go
vars := gocurl.Variables{
    "api_key": "secret123",
    "endpoint": "/users",
}

// NO environment variable expansion, only vars map
resp, err := gocurl.CurlWithVars(ctx, vars,
    "https://api.example.com${endpoint}")

resp, err := gocurl.CurlCommandWithVars(ctx, vars,
    `curl -H "X-API-Key: ${api_key}" https://api.example.com${endpoint}`)

resp, err := gocurl.CurlArgsWithVars(ctx, vars,
    "-H", "X-API-Key: ${api_key}",
    "https://api.example.com${endpoint}")
```

**Use when:**
- Testing with mock values
- Don't want environment variable expansion
- Need explicit variable control
- Sandboxed execution

**Important:** WithVars functions do NOT expand environment variables. Only the provided map is used.

### Decision Tree: Which Function to Use?

```
Need response body?
├─ NO → Use Curl() functions
│   └─ Example: HEAD requests, checking existence
│
├─ YES → What format?
    │
    ├─ Text/String → Use CurlString() functions
    │   └─ Example: HTML, XML, plain text
    │
    ├─ Binary → Use CurlBytes() functions
    │   └─ Example: Images, PDFs, archives
    │
    ├─ JSON with struct → Use CurlJSON() functions
    │   └─ Example: REST API responses
    │
    ├─ Large file → Use CurlDownload() functions
    │   └─ Example: Downloads, datasets
    │
    └─ Need variable control? → WithVars variants
        └─ Example: Testing, sandboxed execution
```

## Variable Expansion and Substitution

GoCurl supports two types of variable substitution: automatic environment expansion and explicit variable maps.

### Automatic Environment Variable Expansion

By default, most Curl* functions automatically expand environment variables:

```go
// Set environment variables
os.Setenv("API_KEY", "secret123")
os.Setenv("BASE_URL", "https://api.example.com")

// Both $VAR and ${VAR} syntax work
body, resp, err := gocurl.CurlStringCommand(ctx,
    `curl -H "X-API-Key: $API_KEY" ${BASE_URL}/users`)
// Expands to: curl -H "X-API-Key: secret123" https://api.example.com/users
```

**Supported syntax:**
- `$VARIABLE` - Simple expansion
- `${VARIABLE}` - Brace expansion (safer, recommended)
- `${VARIABLE:-default}` - With default value (if gocurl supports it)

### Explicit Variable Maps (WithVars Functions)

For controlled, explicit variable substitution without environment access:

```go
vars := gocurl.Variables{
    "api_key": "test-key-123",
    "user_id": "42",
}

body, resp, err := gocurl.CurlCommandWithVars(ctx, vars,
    `curl -H "Authorization: Bearer ${api_key}" \
          https://api.example.com/users/${user_id}`)
```

**Key difference:**
```go
// Regular function - expands environment
os.Setenv("TOKEN", "from-env")
body, resp, err := gocurl.CurlString(ctx,
    "https://api.example.com -H 'Authorization: Bearer $TOKEN'")
// Uses: from-env

// WithVars function - ONLY uses provided map, ignores environment
os.Setenv("TOKEN", "from-env")
vars := gocurl.Variables{"token": "from-map"}
body, resp, err := gocurl.CurlCommandWithVars(ctx, vars,
    `curl -H "Authorization: Bearer ${token}" https://api.example.com`)
// Uses: from-map (environment ignored)
```

### Security Best Practices

**❌ DON'T: Hard-code secrets**
```go
// BAD - secret in source code
body, resp, err := gocurl.CurlString(ctx,
    "https://api.example.com -H 'X-API-Key: hardcoded-secret-123'")
```

**✅ DO: Use environment variables**
```go
// GOOD - secret from environment
os.Setenv("API_KEY", loadFromVault()) // Load from secure source
body, resp, err := gocurl.CurlStringCommand(ctx,
    `curl -H "X-API-Key: $API_KEY" https://api.example.com`)
```

**✅ DO: Use explicit maps for testing**
```go
// GOOD - explicit for tests, no environment pollution
vars := gocurl.Variables{
    "api_key": "test-key-for-mocking",
}
body, resp, err := gocurl.CurlCommandWithVars(ctx, vars,
    `curl -H "X-API-Key: ${api_key}" https://api.example.com`)
```

### Variable Expansion Examples

**Example 1: Configuration from environment**
```go
// Load config from environment
os.Setenv("API_BASE_URL", "https://api.production.com")
os.Setenv("API_VERSION", "v2")
os.Setenv("API_KEY", "prod-key-xxx")

// Use in requests
body, resp, err := gocurl.CurlStringCommand(ctx,
    `curl -H "Authorization: Bearer ${API_KEY}" \
          ${API_BASE_URL}/${API_VERSION}/users`)
```

**Example 2: Multi-environment support**
```go
func NewAPIClient(env string) *APIClient {
    prefix := strings.ToUpper(env) + "_"
    return &APIClient{
        baseURL: os.Getenv(prefix + "API_URL"),
        apiKey:  os.Getenv(prefix + "API_KEY"),
    }
}

// Usage:
// PROD_API_URL=https://api.prod.com
// PROD_API_KEY=prod-key
// DEV_API_URL=https://api.dev.com
// DEV_API_KEY=dev-key

prodClient := NewAPIClient("PROD")
devClient := NewAPIClient("DEV")
```

**Example 3: Testing with explicit variables**
```go
func TestAPIClient(t *testing.T) {
    // Don't pollute environment in tests
    vars := gocurl.Variables{
        "base_url": "http://localhost:8080",
        "api_key":  "test-key",
    }

    body, resp, err := gocurl.CurlCommandWithVars(context.Background(), vars,
        `curl -H "X-API-Key: ${api_key}" ${base_url}/test`)

    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}
```

## Context Usage and Management

Context is crucial for controlling request lifecycle: timeouts, cancellation, and deadline management.

### Context Basics

Every gocurl function's first parameter is `context.Context`:

```go
func CurlString(ctx context.Context, command ...string) (string, *http.Response, error)
```

This context controls:
- Request timeout
- Cancellation
- Deadline
- Request-scoped values

### Common Context Patterns

**Pattern 1: Timeout**
```go
// Request must complete within 10 seconds
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

body, resp, err := gocurl.CurlString(ctx, "https://api.example.com/slow")
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        fmt.Println("Request timed out")
    }
    return err
}
defer resp.Body.Close()
```

**Pattern 2: Cancellation**
```go
ctx, cancel := context.WithCancel(context.Background())

// Cancel on signal
go func() {
    <-sigChan
    cancel()
}()

body, resp, err := gocurl.CurlString(ctx, "https://api.example.com/data")
if err != nil {
    if ctx.Err() == context.Canceled {
        fmt.Println("Request cancelled by user")
    }
    return err
}
defer resp.Body.Close()
```

**Pattern 3: Deadline**
```go
// Must complete before specific time
deadline := time.Now().Add(30 * time.Second)
ctx, cancel := context.WithDeadline(context.Background(), deadline)
defer cancel()

body, resp, err := gocurl.CurlString(ctx, "https://api.example.com/data")
```

**Pattern 4: Parent context propagation**
```go
// HTTP handler with request context
func handleRequest(w http.ResponseWriter, r *http.Request) {
    // Use request's context (cancelled if client disconnects)
    ctx := r.Context()

    // Add timeout to existing context
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    body, resp, err := gocurl.CurlString(ctx, "https://api.backend.com/data")
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }
    defer resp.Body.Close()

    w.Write([]byte(body))
}
```

### Context Values for Request Metadata

Use context values for request-scoped data like request IDs:

```go
type contextKey string

const requestIDKey contextKey = "request_id"

// Add request ID to context
requestID := uuid.New().String()
ctx := context.WithValue(context.Background(), requestIDKey, requestID)

// Later, extract in middleware
if reqID := ctx.Value(requestIDKey); reqID != nil {
    // Use request ID for tracing
}
```

### Context Best Practices

**✅ DO:**
- Always pass context as first parameter
- Use `defer cancel()` immediately after creating context
- Check `ctx.Err()` to distinguish timeout vs other errors
- Propagate parent contexts in HTTP handlers
- Use context values for request-scoped data only

**❌ DON'T:**
- Don't use `context.Background()` in production without timeout
- Don't ignore context cancellation
- Don't store context in structs (pass explicitly)
- Don't use context values for required function parameters

## Response Handling Patterns

Different scenarios require different response handling approaches.

### Manual Response Reading

With basic `Curl()` functions, read the body yourself:

```go
resp, err := gocurl.Curl(ctx, "https://api.example.com/data")
if err != nil {
    return err
}
defer resp.Body.Close()

// Read entire body
body, err := io.ReadAll(resp.Body)
if err != nil {
    return err
}

fmt.Println(string(body))
```

### Automatic String Reading

With `CurlString()` functions, body is already a string:

```go
body, resp, err := gocurl.CurlString(ctx, "https://api.example.com/data")
if err != nil {
    return err
}
defer resp.Body.Close() // Still close for cleanup

// Body is already string
fmt.Println(body)
```

### Status Code Handling

Always check status codes:

```go
body, resp, err := gocurl.CurlString(ctx, "https://api.example.com/users/123")
if err != nil {
    return err
}
defer resp.Body.Close()

switch resp.StatusCode {
case 200:
    // Success
    fmt.Println("User found:", body)
case 404:
    return fmt.Errorf("user not found")
case 401:
    return fmt.Errorf("unauthorized")
case 500:
    return fmt.Errorf("server error: %s", body)
default:
    return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
}
```

### Header Access

Access response headers:

```go
body, resp, err := gocurl.CurlString(ctx, "https://api.example.com/data")
if err != nil {
    return err
}
defer resp.Body.Close()

// Get specific headers
contentType := resp.Header.Get("Content-Type")
rateLimit := resp.Header.Get("X-RateLimit-Remaining")

// Iterate all headers
for key, values := range resp.Header {
    for _, value := range values {
        fmt.Printf("%s: %s\n", key, value)
    }
}
```

### JSON Unmarshaling

**Option 1: Use CurlJSON (Recommended)**
```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

var user User
resp, err := gocurl.CurlJSON(ctx, &user, "https://api.example.com/users/123")
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Printf("User: %s (ID: %d)\n", user.Name, user.ID)
```

**Option 2: Manual unmarshaling**
```go
body, resp, err := gocurl.CurlString(ctx, "https://api.example.com/users/123")
if err != nil {
    return err
}
defer resp.Body.Close()

var user User
if err := json.Unmarshal([]byte(body), &user); err != nil {
    return err
}
```

### Error Response Handling

Many APIs return JSON error responses:

```go
type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

body, resp, err := gocurl.CurlString(ctx, "https://api.example.com/action")
if err != nil {
    return err
}
defer resp.Body.Close()

if resp.StatusCode != 200 {
    var apiErr APIError
    if err := json.Unmarshal([]byte(body), &apiErr); err != nil {
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
    }
    return fmt.Errorf("API error %s: %s", apiErr.Code, apiErr.Message)
}
```

### Streaming Responses

For large responses, stream instead of loading into memory:

```go
resp, err := gocurl.Curl(ctx, "https://api.example.com/large-dataset")
if err != nil {
    return err
}
defer resp.Body.Close()

// Process line by line
scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
    line := scanner.Text()
    // Process line
    fmt.Println(line)
}

if err := scanner.Err(); err != nil {
    return err
}
```

## The Process() Function - Core Execution Engine

`Process()` is the heart of gocurl. All Curl* functions eventually call it.

### Process() Signature

```go
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, *http.Response, error)
```

**Parameters:**
- `ctx` - Context for timeout/cancellation
- `opts` - RequestOptions struct with all configuration

**Returns:**
- First `*http.Response` - The actual HTTP response
- Second `*http.Response` - Same response (for convenience)
- `error` - Any error that occurred

### What Process() Does

1. Validates RequestOptions
2. Applies middleware pipeline
3. Builds http.Request
4. Executes request with http.Client
5. Handles retries (if configured)
6. Returns response

### Using Process() Directly

When you build RequestOptions programmatically:

```go
opts := &options.RequestOptions{
    URL:    "https://api.example.com/data",
    Method: "GET",
    Headers: http.Header{
        "Accept":        []string{"application/json"},
        "Authorization": []string{"Bearer token"},
    },
    Timeout: 30 * time.Second,
}

resp, _, err := gocurl.Process(ctx, opts)
if err != nil {
    return err
}
defer resp.Body.Close()
```

### What Curl Functions Do Internally

Here's what happens inside `CurlCommand`:

```go
func CurlCommand(ctx context.Context, command string) (*http.Response, error) {
    // 1. Preprocess multi-line command
    processed := preprocessMultilineCommand(command)

    // 2. Tokenize
    tokens := tokenize(processed)

    // 3. Convert to RequestOptions
    opts, err := convertTokensToRequestOptions(tokens)
    if err != nil {
        return nil, err
    }

    // 4. Call Process()
    resp, _, err := Process(ctx, opts)
    return resp, err
}
```

Similarly for `CurlString`:

```go
func CurlString(ctx context.Context, command ...string) (string, *http.Response, error) {
    // Convert to RequestOptions (same process)
    opts, err := parseToRequestOptions(command...)
    if err != nil {
        return "", nil, err
    }

    // Call Process()
    resp, _, err := Process(ctx, opts)
    if err != nil {
        return "", nil, err
    }

    // Read body as string
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", resp, err
    }

    return string(body), resp, nil
}
```

### When to Use Process() Directly

Use `Process()` when:

1. **Building requests programmatically**
   ```go
   opts := buildRequestOptions(config)
   resp, _, err := gocurl.Process(ctx, opts)
   ```

2. **Reusing RequestOptions**
   ```go
   baseOpts := &options.RequestOptions{...}

   // Clone and modify
   opts1 := baseOpts.Clone()
   opts1.URL = "https://api.example.com/endpoint1"

   opts2 := baseOpts.Clone()
   opts2.URL = "https://api.example.com/endpoint2"
   ```

3. **Testing with custom options**
   ```go
   opts := &options.RequestOptions{
       URL: testServer.URL,
       CustomClient: mockHTTPClient,
   }
   resp, _, err := gocurl.Process(ctx, opts)
   ```

4. **Implementing custom abstractions**
   ```go
   func (c *APIClient) request(endpoint string) (*http.Response, error) {
       opts := c.baseOptions.Clone()
       opts.URL = c.baseURL + endpoint
       return gocurl.Process(c.ctx, opts)
   }
   ```

## Practical Examples

### Example 1: Function Category Selection

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

func main() {
    ctx := context.Background()

    // Scenario 1: Just checking if endpoint exists (HEAD request)
    resp, err := gocurl.Curl(ctx, "-I", "https://api.github.com/zen")
    if err != nil {
        log.Fatal(err)
    }
    resp.Body.Close()
    fmt.Printf("Exists: %t\n", resp.StatusCode == 200)

    // Scenario 2: Need response as string
    body, resp, err := gocurl.CurlString(ctx, "https://api.github.com/zen")
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
    fmt.Printf("Quote: %s\n", body)

    // Scenario 3: Need structured JSON data
    type Repo struct {
        Name  string `json:"name"`
        Stars int    `json:"stargazers_count"`
    }
    var repo Repo
    resp, err = gocurl.CurlJSON(ctx, &repo, "https://api.github.com/repos/golang/go")
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
    fmt.Printf("Repository: %s (%d stars)\n", repo.Name, repo.Stars)
}
```

### Example 2: Context Timeout Handling

```go
func fetchWithTimeout(url string, timeout time.Duration) (string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()

    body, resp, err := gocurl.CurlString(ctx, url)
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return "", fmt.Errorf("request timed out after %v", timeout)
        }
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return "", fmt.Errorf("HTTP %d", resp.StatusCode)
    }

    return body, nil
}
```

### Example 3: Variable Expansion

```go
func makeAPIRequest(endpoint string) (string, error) {
    // Environment variables set elsewhere
    // API_KEY=secret123
    // BASE_URL=https://api.example.com

    ctx := context.Background()

    body, resp, err := gocurl.CurlStringCommand(ctx,
        `curl -H "X-API-Key: ${API_KEY}" ${BASE_URL}`+endpoint)

    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    return body, nil
}
```

## Summary

In this chapter, you learned:

- ✅ **Dual API approach** - Curl-syntax for rapid development, Builder for type-safety
- ✅ **Six function categories** - Each optimized for different response handling needs
- ✅ **Variable expansion** - Automatic environment expansion vs explicit maps
- ✅ **Context management** - Timeouts, cancellation, and deadline control
- ✅ **Response handling** - Multiple patterns for different scenarios
- ✅ **Process() function** - The core execution engine powering all requests

**Key Takeaways:**

1. Choose function category based on response needs (string, bytes, JSON, etc.)
2. Use Curl-syntax for quick prototyping, Builder for production SDKs
3. Always use context with appropriate timeouts
4. Check status codes and handle errors appropriately
5. Leverage variable expansion for secure credential management

**Next Chapter:**

In **Chapter 4: Command-Line Interface**, we'll explore the gocurl CLI tool for testing, debugging, and the powerful CLI-to-code workflow that makes gocurl unique.
