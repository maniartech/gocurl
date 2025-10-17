# Chapter 5: RequestOptions & Builder Pattern

## Introduction

While Part I introduced you to GoCurl's curl-syntax functions (`Curl`, `CurlString`, `CurlJSON`, etc.), there's another powerful way to build HTTP requests in GoCurl: the **RequestOptions struct and Builder pattern**. This programmatic approach gives you type safety, IDE autocompletion, and better validation—making it ideal for complex requests and production codebases.

In this chapter, you'll master:
- When to use Builder pattern vs curl-syntax functions
- The `RequestOptions` struct with all 30+ fields
- Fluent API building with `RequestOptionsBuilder`
- Thread safety and the `Clone()` method
- Context management best practices
- Request validation before execution

### The Two Approaches

GoCurl offers two distinct ways to make HTTP requests:

**1. Curl-Syntax Functions** (Chapters 3-4):
```go
resp, err := gocurl.Curl(ctx, "-X", "POST", "-H", "Content-Type: application/json",
    "-d", `{"name":"Alice"}`, "https://api.example.com/users")
```

**2. Builder Pattern** (This Chapter):
```go
opts := options.NewRequestOptionsBuilder().
    SetMethod("POST").
    SetURL("https://api.example.com/users").
    AddHeader("Content-Type", "application/json").
    SetBody(`{"name":"Alice"}`).
    Build()

resp, err := gocurl.Execute(ctx, opts)
```

Both approaches execute the same request—the difference is in how you construct it.

### When to Use Each Approach

| **Use Curl-Syntax When:**                 | **Use Builder Pattern When:**              |
|-------------------------------------------|---------------------------------------------|
| Converting curl commands to Go            | Building requests programmatically          |
| Quick scripts and prototypes              | Production applications                     |
| Simple requests (GET, POST with headers)  | Complex configurations                      |
| Testing with familiar curl syntax         | Need type safety and validation             |
| Minimal code footprint                    | Want IDE autocompletion                     |
| Environment variable expansion needed     | Building request templates                  |
|                                           | Thread-safe concurrent requests with Clone()|

**Golden Rule:** If you're copying from a curl command, use curl-syntax. If you're building programmatically, use Builder pattern.

---

## Part 1: The RequestOptions Struct

The `RequestOptions` struct is the foundation of the Builder pattern. It contains every possible configuration for an HTTP request—over 30 fields organized into logical groups.

### Struct Overview

```go
type RequestOptions struct {
    // HTTP request basics
    Method      string
    URL         string
    Headers     http.Header
    Body        string
    Form        url.Values
    QueryParams url.Values

    // Authentication
    BasicAuth   *BasicAuth
    BearerToken string

    // TLS/SSL options
    CertFile            string
    KeyFile             string
    CAFile              string
    Insecure            bool
    TLSConfig           *tls.Config
    CertPinFingerprints []string
    SNIServerName       string

    // Proxy settings
    Proxy        string
    ProxyNoProxy []string

    // Timeout settings
    Timeout        time.Duration
    ConnectTimeout time.Duration

    // Redirect behavior
    FollowRedirects bool
    MaxRedirects    int

    // Compression
    Compress           bool
    CompressionMethods []string

    // HTTP version specific
    HTTP2     bool
    HTTP2Only bool

    // Cookie handling
    Cookies    []*http.Cookie
    CookieJar  http.CookieJar
    CookieFile string

    // Custom options
    UserAgent string
    Referer   string

    // File upload
    FileUpload *FileUpload

    // Retry configuration
    RetryConfig *RetryConfig

    // Output options
    OutputFile string
    Silent     bool
    Verbose    bool

    // Advanced options
    RequestID         string
    Middleware        []middlewares.MiddlewareFunc
    ResponseBodyLimit int64
    CustomClient      HTTPClient
}
```

### Field Groups Deep Dive

#### 1. HTTP Request Basics

These are the core fields for any HTTP request:

**Method** (`string`)
- HTTP method: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS
- Default: GET (if not specified)
- Example: `Method: "POST"`

**URL** (`string`)
- Full URL including scheme (http:// or https://)
- Required field
- Example: `URL: "https://api.github.com/user"`

**Headers** (`http.Header`)
- HTTP headers (map[string][]string)
- Use `AddHeader()` or `SetHeader()` methods
- Example:
  ```go
  headers := http.Header{}
  headers.Add("Authorization", "Bearer token")
  headers.Add("Content-Type", "application/json")
  ```

**Body** (`string`)
- Request body content
- Used for POST, PUT, PATCH requests
- Example: `Body: "{\"name\":\"Alice\"}"`

**Form** (`url.Values`)
- Form data for `application/x-www-form-urlencoded`
- Map of string keys to string slice values
- Example:
  ```go
  form := url.Values{}
  form.Add("username", "alice")
  form.Add("email", "alice@example.com")
  ```

**QueryParams** (`url.Values`)
- URL query parameters
- Appended to URL automatically
- Example:
  ```go
  params := url.Values{}
  params.Add("page", "1")
  params.Add("limit", "10")
  // Results in: ?page=1&limit=10
  ```

#### 2. Authentication

**BasicAuth** (`*BasicAuth`)
- HTTP Basic Authentication
- Struct: `{Username string, Password string}`
- Automatically encoded to Base64
- Example:
  ```go
  BasicAuth: &BasicAuth{
      Username: "admin",
      Password: "secret",
  }
  ```

**BearerToken** (`string`)
- OAuth 2.0 Bearer token
- Automatically adds "Authorization: Bearer <token>" header
- Example: `BearerToken: "eyJhbGciOiJI..."`

#### 3. TLS/SSL Options

**CertFile** (`string`)
- Path to client certificate file
- Used for mutual TLS
- Example: `CertFile: "/path/to/client.crt"`

**KeyFile** (`string`)
- Path to private key file
- Must be used with CertFile
- Example: `KeyFile: "/path/to/client.key"`

**CAFile** (`string`)
- Path to custom CA certificate
- For self-signed or internal CAs
- Example: `CAFile: "/path/to/ca.crt"`

**Insecure** (`bool`)
- Skip TLS certificate verification
- **WARNING:** Only use in development!
- Example: `Insecure: true`

**TLSConfig** (`*tls.Config`)
- Custom TLS configuration
- Advanced control over TLS behavior
- Example:
  ```go
  TLSConfig: &tls.Config{
      MinVersion: tls.VersionTLS12,
      MaxVersion: tls.VersionTLS13,
  }
  ```

**CertPinFingerprints** (`[]string`)
- SHA256 fingerprints for certificate pinning
- Enhanced security against MITM attacks
- Example: `CertPinFingerprints: []string{"sha256//AAAA..."}`

**SNIServerName** (`string`)
- Server Name Indication for TLS
- Used when hostname differs from certificate
- Example: `SNIServerName: "api.example.com"`

#### 4. Proxy Settings

**Proxy** (`string`)
- HTTP/HTTPS/SOCKS5 proxy URL
- Format: `protocol://host:port`
- Example: `Proxy: "http://proxy.corp.com:8080"`

**ProxyNoProxy** (`[]string`)
- Domains to exclude from proxying
- Supports wildcards
- Example: `ProxyNoProxy: []string{"localhost", "*.internal.com"}`

#### 5. Timeout Settings

**Timeout** (`time.Duration`)
- Total request timeout (connection + read + write)
- Includes all retries
- Example: `Timeout: 30 * time.Second`

**ConnectTimeout** (`time.Duration`)
- Connection establishment timeout only
- Useful for detecting unreachable hosts quickly
- Example: `ConnectTimeout: 5 * time.Second`

#### 6. Redirect Behavior

**FollowRedirects** (`bool`)
- Whether to follow HTTP redirects (3xx)
- Default: false (matches curl behavior)
- Example: `FollowRedirects: true`

**MaxRedirects** (`int`)
- Maximum number of redirects to follow
- Prevents infinite redirect loops
- Example: `MaxRedirects: 10`

#### 7. Compression

**Compress** (`bool`)
- Enable automatic compression
- Adds "Accept-Encoding" header
- Automatically decompresses response
- Example: `Compress: true`

**CompressionMethods** (`[]string`)
- Specific compression methods to support
- Options: "gzip", "deflate", "br" (Brotli)
- Example: `CompressionMethods: []string{"gzip", "br"}`

#### 8. HTTP Version

**HTTP2** (`bool`)
- Enable HTTP/2 support
- Falls back to HTTP/1.1 if server doesn't support
- Example: `HTTP2: true`

**HTTP2Only** (`bool`)
- Require HTTP/2, fail if not supported
- Use for performance-critical applications
- Example: `HTTP2Only: true`

#### 9. Cookie Handling

**Cookies** (`[]*http.Cookie`)
- Cookies to send with request
- Array of http.Cookie objects
- Example:
  ```go
  Cookies: []*http.Cookie{
      {Name: "session", Value: "abc123"},
      {Name: "theme", Value: "dark"},
  }
  ```

**CookieJar** (`http.CookieJar`)
- Cookie storage for multiple requests
- Automatically manages cookies across requests
- Example:
  ```go
  jar, _ := cookiejar.New(nil)
  CookieJar: jar
  ```

**CookieFile** (`string`)
- Path to file for persistent cookie storage
- Reads cookies on start, writes on completion
- Example: `CookieFile: "/tmp/cookies.txt"`

#### 10. Custom Options

**UserAgent** (`string`)
- Custom User-Agent header
- Overrides default GoCurl user agent
- Example: `UserAgent: "MyApp/1.0"`

**Referer** (`string`)
- HTTP Referer header
- Indicates source page
- Example: `Referer: "https://example.com/page1"`

#### 11. File Upload

**FileUpload** (`*FileUpload`)
- Configuration for multipart file uploads
- Struct: `{FieldName, FileName, FilePath}`
- Example:
  ```go
  FileUpload: &FileUpload{
      FieldName: "document",
      FileName: "report.pdf",
      FilePath: "/path/to/report.pdf",
  }
  ```

#### 12. Retry Configuration

**RetryConfig** (`*RetryConfig`)
- Automatic retry configuration
- Struct: `{MaxRetries, RetryDelay, RetryOnHTTP}`
- Example:
  ```go
  RetryConfig: &RetryConfig{
      MaxRetries:  3,
      RetryDelay:  1 * time.Second,
      RetryOnHTTP: []int{429, 500, 502, 503, 504},
  }
  ```

#### 13. Output Options

**OutputFile** (`string`)
- Save response body to file
- Creates file or overwrites if exists
- Example: `OutputFile: "/tmp/response.json"`

**Silent** (`bool`)
- Suppress progress output
- Useful for scripts
- Example: `Silent: true`

**Verbose** (`bool`)
- Enable detailed logging
- Shows request/response details
- Example: `Verbose: true`

#### 14. Advanced Options

**RequestID** (`string`)
- Custom request identifier for tracing
- Useful for distributed systems
- Example: `RequestID: "req-12345"`

**Middleware** (`[]middlewares.MiddlewareFunc`)
- Chain of middleware functions
- Executed before/after request
- Example: (covered in Chapter 15)

**ResponseBodyLimit** (`int64`)
- Maximum response body size in bytes
- Prevents memory exhaustion
- Example: `ResponseBodyLimit: 10 * 1024 * 1024` (10MB)

**CustomClient** (`HTTPClient`)
- Custom HTTP client implementation
- Useful for testing/mocking
- Example: (covered in Chapter 18)

---

## Part 2: The Builder Pattern

The `RequestOptionsBuilder` provides a fluent API for constructing `RequestOptions` objects. It's the recommended way to build requests programmatically.

### Creating a Builder

```go
import "github.com/maniartech/gocurl/options"

builder := options.NewRequestOptionsBuilder()
```

The builder starts with an empty `RequestOptions` with initialized maps for Headers, Form, and QueryParams.

### Fluent API Methods

Every builder method returns `*RequestOptionsBuilder`, allowing method chaining:

```go
opts := builder.
    SetURL("https://api.example.com/users").
    SetMethod("POST").
    AddHeader("Authorization", "Bearer token").
    SetBody(`{"name":"Alice"}`).
    SetTimeout(30 * time.Second).
    Build()
```

### Core Builder Methods

#### HTTP Basics

```go
// Set HTTP method
SetMethod(method string) *RequestOptionsBuilder

// Set URL
SetURL(url string) *RequestOptionsBuilder

// Add a header (can be called multiple times for same key)
AddHeader(key, value string) *RequestOptionsBuilder

// Set headers (replaces all existing)
SetHeaders(headers http.Header) *RequestOptionsBuilder

// Set request body
SetBody(body string) *RequestOptionsBuilder

// Set form data
SetForm(form url.Values) *RequestOptionsBuilder

// Set query parameters
SetQueryParams(queryParams url.Values) *RequestOptionsBuilder

// Add single query parameter
AddQueryParam(key, value string) *RequestOptionsBuilder
```

#### Authentication

```go
// Set Basic Auth
SetBasicAuth(username, password string) *RequestOptionsBuilder

// Set Bearer token
SetBearerToken(token string) *RequestOptionsBuilder

// Convenience: Add Bearer token as header
BearerAuth(token string) *RequestOptionsBuilder
```

#### TLS/SSL

```go
SetCertFile(certFile string) *RequestOptionsBuilder
SetKeyFile(keyFile string) *RequestOptionsBuilder
SetCAFile(caFile string) *RequestOptionsBuilder
SetInsecure(insecure bool) *RequestOptionsBuilder
SetTLSConfig(tlsConfig *tls.Config) *RequestOptionsBuilder
```

#### Proxy & Timeouts

```go
SetProxy(proxy string) *RequestOptionsBuilder
SetTimeout(timeout time.Duration) *RequestOptionsBuilder
SetConnectTimeout(connectTimeout time.Duration) *RequestOptionsBuilder
```

#### Redirects & Compression

```go
SetFollowRedirects(follow bool) *RequestOptionsBuilder
SetMaxRedirects(maxRedirects int) *RequestOptionsBuilder
SetCompress(compress bool) *RequestOptionsBuilder
```

#### HTTP/2 & Cookies

```go
SetHTTP2(http2 bool) *RequestOptionsBuilder
SetHTTP2Only(http2Only bool) *RequestOptionsBuilder
SetCookie(cookie *http.Cookie) *RequestOptionsBuilder
```

#### Custom & Output

```go
SetUserAgent(userAgent string) *RequestOptionsBuilder
SetReferer(referer string) *RequestOptionsBuilder
SetOutputFile(outputFile string) *RequestOptionsBuilder
SetSilent(silent bool) *RequestOptionsBuilder
SetVerbose(verbose bool) *RequestOptionsBuilder
```

#### Upload & Retry

```go
SetFileUpload(fileUpload *FileUpload) *RequestOptionsBuilder
SetRetryConfig(retryConfig *RetryConfig) *RequestOptionsBuilder
```

### HTTP Method Shortcuts

The builder provides convenience methods for common HTTP methods:

```go
// GET request
Get(url string, headers http.Header) *RequestOptionsBuilder

// POST request
Post(url string, body string, headers http.Header) *RequestOptionsBuilder

// PUT request
Put(url string, body string, headers http.Header) *RequestOptionsBuilder

// DELETE request
Delete(url string, headers http.Header) *RequestOptionsBuilder

// PATCH request
Patch(url string, body string, headers http.Header) *RequestOptionsBuilder
```

Example:
```go
opts := options.NewRequestOptionsBuilder().
    Post("https://api.example.com/users",
        `{"name":"Alice"}`,
        http.Header{"Content-Type": []string{"application/json"}}).
    SetTimeout(30 * time.Second).
    Build()
```

### Convenience Methods

The builder includes several convenience methods for common patterns:

#### JSON()
Marshals an object to JSON and sets Content-Type header:
```go
user := User{Name: "Alice", Email: "alice@example.com"}
builder.JSON(user)
// Equivalent to:
// builder.SetBody(`{"name":"Alice","email":"alice@example.com"}`)
// builder.AddHeader("Content-Type", "application/json")
```

#### Form()
Sets form data and Content-Type header:
```go
formData := url.Values{}
formData.Add("username", "alice")
formData.Add("password", "secret")
builder.Form(formData)
// Adds: Content-Type: application/x-www-form-urlencoded
```

#### WithDefaultRetry()
Adds standard retry configuration:
```go
builder.WithDefaultRetry()
// Equivalent to:
// builder.SetRetryConfig(&RetryConfig{
//     MaxRetries:  3,
//     RetryDelay:  1 * time.Second,
//     RetryOnHTTP: []int{429, 500, 502, 503, 504},
// })
```

#### QuickTimeout() / SlowTimeout()
Pre-configured timeout durations:
```go
builder.QuickTimeout()  // 5 seconds
builder.SlowTimeout()   // 2 minutes
```

### Building the Request

Once configured, call `Build()` to get the `RequestOptions`:

```go
opts := builder.Build()
```

**Important:** `Build()` returns a **clone** of the internal options, so you can safely reuse the builder:

```go
builder := options.NewRequestOptionsBuilder().
    SetURL("https://api.example.com/users").
    AddHeader("Authorization", "Bearer token")

// Build multiple requests from same base
opts1 := builder.SetMethod("GET").Build()
opts2 := builder.SetMethod("POST").SetBody(`{"name":"Bob"}`).Build()
```

---

## Part 3: Thread Safety and Clone()

### Thread Safety Guarantees

`RequestOptions` has specific thread-safety guarantees:

**✅ SAFE for concurrent reads**
- Multiple goroutines can safely read the same `RequestOptions`
- All fields can be accessed concurrently

**✅ SAFE for concurrent execution**
- Multiple goroutines can execute requests with the same options
- Each execution uses its own context and state

**❌ UNSAFE for concurrent writes**
- Map-based fields (`Headers`, `Form`, `QueryParams`) are NOT thread-safe for writes
- Concurrent modifications cause race conditions

**Unsafe map fields:**
- `Headers` (http.Header = map[string][]string)
- `Form` (url.Values = map[string][]string)
- `QueryParams` (url.Values = map[string][]string)

### Safe Concurrent Pattern: Clone()

The `Clone()` method creates a deep copy of `RequestOptions`, making it safe for concurrent modification:

```go
baseOpts := options.NewRequestOptions("https://api.example.com")
baseOpts.SetHeader("Authorization", "Bearer token")

// Safe: Clone before modification in each goroutine
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()

        // Clone creates independent copy
        opts := baseOpts.Clone()
        opts.AddHeader("X-Request-ID", fmt.Sprintf("req-%d", id))

        resp, err := gocurl.Execute(ctx, opts)
        // Process response...
    }(i)
}
wg.Wait()
```

### Unsafe Pattern (Race Condition)

**❌ DON'T DO THIS:**
```go
opts := options.NewRequestOptions("https://api.example.com")

// RACE CONDITION: Concurrent map writes
go opts.AddHeader("X-ID", "1")  // UNSAFE
go opts.AddHeader("X-ID", "2")  // UNSAFE
```

Run tests with `-race` flag to detect race conditions:
```bash
go test -race ./...
```

### What Clone() Copies

The `Clone()` method performs:

**Deep copies:**
- Headers (http.Header)
- Form (url.Values)
- QueryParams (url.Values)
- BasicAuth struct
- FileUpload struct
- RetryConfig struct

**Shallow copies (shared references):**
- TLSConfig (typically shared)
- CookieJar (manages its own concurrency)
- Middleware functions (immutable)
- CustomClient (should be thread-safe)

This design balances safety with performance—objects that are naturally shared or thread-safe aren't unnecessarily copied.

---

## Part 4: Context Management

Go's `context` package is fundamental for timeout management and cancellation. The Builder pattern integrates context management following Go best practices.

### Why Context Isn't in RequestOptions

You might notice `context.Context` is NOT a field in `RequestOptions`. This is intentional and follows Go conventions:

**Go Best Practice:** Context should be **passed explicitly** as a function parameter, not stored in structs.

From Go documentation:
> "Do not store Contexts inside a struct type; instead, pass a Context explicitly to each function that needs it."

### Context in Builder

The `RequestOptionsBuilder` **does** store context, but only temporarily during building:

```go
builder := options.NewRequestOptionsBuilder().
    SetURL("https://api.example.com/users").
    WithContext(ctx)  // Store context in builder

// Context passed explicitly during execution
resp, err := gocurl.Execute(builder.GetContext(), builder.Build())
```

### WithContext() Method

Sets the context on the builder:

```go
ctx := context.Background()
builder.WithContext(ctx)
```

### WithTimeout() Method

Creates a context with timeout and stores it:

```go
builder.WithTimeout(30 * time.Second)
// Creates context.WithTimeout internally
```

**Important:** `WithTimeout()` creates a cancel function that must be cleaned up:

```go
builder := options.NewRequestOptionsBuilder().
    SetURL("https://api.example.com/users").
    WithTimeout(30 * time.Second)
defer builder.Cleanup()  // MUST call to prevent context leak!

resp, err := gocurl.Execute(builder.GetContext(), builder.Build())
```

### GetContext() Method

Retrieves the context from the builder:

```go
ctx := builder.GetContext()
// Returns context.Background() if none was set
```

### Cleanup() Method

Calls the cancel function if one was created:

```go
builder.Cleanup()
```

**When to call Cleanup():**
- After `WithTimeout()` - prevents context leak
- After `WithContext()` with a cancelable context
- In defer statement for safety

### Complete Context Example

```go
func makeRequest(url string) error {
    builder := options.NewRequestOptionsBuilder().
        SetURL(url).
        WithTimeout(30 * time.Second)
    defer builder.Cleanup()  // Cleanup context

    opts := builder.Build()
    resp, err := gocurl.Execute(builder.GetContext(), opts)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Process response...
    return nil
}
```

### Context Best Practices

1. **Always pass context explicitly to Execute()**
   ```go
   resp, err := gocurl.Execute(ctx, opts)
   ```

2. **Use WithTimeout() for automatic timeout**
   ```go
   builder.WithTimeout(30 * time.Second)
   defer builder.Cleanup()
   ```

3. **Call Cleanup() in defer**
   ```go
   defer builder.Cleanup()
   ```

4. **Don't store context in RequestOptions**
   ```go
   // ❌ Wrong: Storing context in struct
   type MyClient struct {
       ctx context.Context
   }

   // ✅ Correct: Pass context to methods
   func (c *MyClient) Get(ctx context.Context, url string) error {
       // ...
   }
   ```

---

## Part 5: Validation

The Builder pattern includes built-in validation to catch errors before execution.

### Validate() Method

Checks all fields for validity:

```go
builder := options.NewRequestOptionsBuilder().
    SetURL("https://api.example.com").
    SetMethod("INVALID")  // Invalid method

if err := builder.Validate(); err != nil {
    fmt.Println("Validation failed:", err)
}
```

### What Gets Validated

**1. HTTP Method**
- Must be valid HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)
- Case-insensitive

**2. URL**
- Must be valid URL format
- Must include scheme (http:// or https://)
- Must have host

**3. Headers**
- Header names must be valid
- No control characters

**4. Body**
- Size must be within `ResponseBodyLimit`
- Default limit: 10MB

**5. Form**
- Form keys and values must be valid
- No empty keys

**6. Query Parameters**
- Parameter names must be valid
- Values must be URL-encodable

**7. Secure Authentication**
- Basic Auth requires HTTPS (unless development mode)
- Bearer Token requires HTTPS (unless development mode)

### Validation Errors

Validation returns descriptive errors:

```go
err := builder.Validate()
// Example errors:
// - "invalid method: INVALID"
// - "url is required"
// - "url must include scheme (http:// or https://)"
// - "basic auth requires HTTPS (use Insecure=true for development)"
```

### When to Validate

**Manual validation:**
```go
if err := builder.Validate(); err != nil {
    return fmt.Errorf("invalid request: %w", err)
}
```

**Automatic validation:**
```go
// Execute() calls Validate() internally
resp, err := gocurl.Execute(ctx, opts)
// err will include validation errors
```

### Validation Best Practices

1. **Validate early in development**
   ```go
   opts := builder.Build()
   if err := opts.Validate(); err != nil {
       log.Fatal("Invalid options:", err)
   }
   ```

2. **Test validation in unit tests**
   ```go
   func TestInvalidMethod(t *testing.T) {
       builder := options.NewRequestOptionsBuilder().
           SetMethod("INVALID")
       err := builder.Validate()
       assert.Error(t, err)
       assert.Contains(t, err.Error(), "invalid method")
   }
   ```

3. **Use validation for request builders**
   ```go
   type APIClient struct {
       baseURL string
   }

   func (c *APIClient) buildRequest(endpoint string) (*RequestOptions, error) {
       builder := options.NewRequestOptionsBuilder().
           SetURL(c.baseURL + endpoint).
           SetMethod("GET")

       if err := builder.Validate(); err != nil {
           return nil, fmt.Errorf("invalid request: %w", err)
       }

       return builder.Build(), nil
   }
   ```

---

## Part 6: Complete Examples

### Example 1: Simple GET Request

```go
package main

import (
    "context"
    "fmt"
    "io"

    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/options"
)

func main() {
    // Create builder
    builder := options.NewRequestOptionsBuilder()

    // Configure request
    opts := builder.
        SetURL("https://api.github.com/zen").
        SetMethod("GET").
        Build()

    // Execute
    ctx := context.Background()
    resp, err := gocurl.Execute(ctx, opts)
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    defer resp.Body.Close()

    // Read response
    body, _ := io.ReadAll(resp.Body)
    fmt.Println("Response:", string(body))
}
```

### Example 2: POST with JSON

```go
package main

import (
    "context"
    "fmt"
    "io"

    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/options"
)

func main() {
    // Create request
    opts := options.NewRequestOptionsBuilder().
        SetURL("https://httpbin.org/post").
        SetMethod("POST").
        AddHeader("Content-Type", "application/json").
        SetBody(`{"name":"Alice","email":"alice@example.com"}`).
        SetTimeout(30 * time.Second).
        Build()

    // Execute
    ctx := context.Background()
    resp, err := gocurl.Execute(ctx, opts)
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    defer resp.Body.Close()

    // Check status
    fmt.Println("Status:", resp.StatusCode)

    // Read response
    body, _ := io.ReadAll(resp.Body)
    fmt.Println("Response:", string(body))
}
```

### Example 3: Authentication with Retry

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/options"
)

func main() {
    // Build request with authentication and retry
    builder := options.NewRequestOptionsBuilder().
        SetURL("https://api.github.com/user").
        SetMethod("GET").
        SetBearerToken("your-github-token").
        WithDefaultRetry().  // 3 retries on 5xx errors
        WithTimeout(30 * time.Second)
    defer builder.Cleanup()

    opts := builder.Build()

    // Execute
    resp, err := gocurl.Execute(builder.GetContext(), opts)
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    defer resp.Body.Close()

    fmt.Println("Status:", resp.StatusCode)
}
```

### Example 4: Clone for Concurrent Requests

```go
package main

import (
    "context"
    "fmt"
    "sync"

    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/options"
)

func main() {
    // Base options
    baseOpts := options.NewRequestOptions("https://api.github.com/users")
    baseOpts.SetHeader("Authorization", "Bearer token")
    baseOpts.Method = "GET"

    // Concurrent requests
    ctx := context.Background()
    users := []string{"golang", "microsoft", "google"}

    var wg sync.WaitGroup
    for _, user := range users {
        wg.Add(1)
        go func(username string) {
            defer wg.Done()

            // Clone and modify
            opts := baseOpts.Clone()
            opts.URL = "https://api.github.com/users/" + username

            resp, err := gocurl.Execute(ctx, opts)
            if err != nil {
                fmt.Printf("Error fetching %s: %v\n", username, err)
                return
            }
            defer resp.Body.Close()

            fmt.Printf("Fetched %s: HTTP %d\n", username, resp.StatusCode)
        }(user)
    }

    wg.Wait()
}
```

---

## Summary

In this chapter, you learned:

✅ **When to use Builder pattern vs curl-syntax functions**
- Builder for programmatic requests, curl-syntax for converting commands

✅ **The RequestOptions struct with 30+ fields**
- Organized into logical groups (HTTP, auth, TLS, proxy, timeouts, etc.)
- Complete control over every aspect of the request

✅ **Fluent API with RequestOptionsBuilder**
- Method chaining for readable code
- HTTP method shortcuts (Get, Post, Put, Delete, Patch)
- Convenience methods (JSON, Form, WithDefaultRetry)

✅ **Thread safety with Clone()**
- Safe for concurrent reads
- Unsafe for concurrent writes
- Clone() for safe concurrent modifications

✅ **Context management best practices**
- Context passed explicitly to Execute()
- WithTimeout() for automatic timeouts
- Cleanup() to prevent context leaks

✅ **Validation before execution**
- Validate() method for early error detection
- Comprehensive checks (method, URL, headers, auth, etc.)
- Clear error messages

### Next Steps

Now that you understand the Builder pattern, you're ready to:

- **Chapter 6:** Master JSON APIs with type-safe request/response handling
- **Chapter 7:** Work with file operations (download, upload, progress tracking)
- **Chapter 8:** Implement production-grade security (TLS, certificate pinning, mutual TLS)

The Builder pattern is your foundation for building robust, type-safe HTTP clients in Go. Combined with the curl-syntax functions from Part I, you now have complete flexibility in how you construct requests.

### Key Takeaways

1. **Choose the right tool:** Curl-syntax for quick scripts, Builder for production code
2. **Use Clone() for concurrency:** Always clone before concurrent modifications
3. **Manage contexts properly:** Use WithTimeout() and Cleanup() to prevent leaks
4. **Validate early:** Catch errors before execution with Validate()
5. **Build request templates:** Reuse builders for consistent requests

You're now equipped to build production-grade HTTP clients with GoCurl's Builder pattern! 🚀
