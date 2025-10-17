# API Reference
## The Definitive Guide to the GoCurl Library

**Version:** 1.0 (GoCurl v1.x)
**Last Updated:** 2025
**Package:** `github.com/stackql/gocurl`

---

## Table of Contents

1. [Introduction](#introduction)
2. [Core Functions](#core-functions)
   - [Basic Functions](#basic-functions)
   - [String Functions](#string-functions)
   - [Bytes Functions](#bytes-functions)
   - [JSON Functions](#json-functions)
   - [Download Functions](#download-functions)
   - [WithVars Functions](#withvars-functions)
3. [RequestOptions API](#requestoptions-api)
4. [Builder Pattern API](#builder-pattern-api)
5. [Middleware API](#middleware-api)
6. [Process Function](#process-function)
7. [Legacy API (Deprecated)](#legacy-api-deprecated)
8. [Supporting Types](#supporting-types)

---

## Introduction

This reference documents all public APIs in the gocurl library. All examples show correct signatures and include proper error handling.

**Import Statement:**
```go
import "github.com/stackql/gocurl"
```

**Common Imports:**
```go
import (
    "context"
    "net/http"
    "time"

    "github.com/stackql/gocurl"
)
```

---

## Core Functions

The gocurl library provides six categories of functions, each designed for specific use cases.

### Basic Functions

**Purpose:** Return HTTP response for manual body handling
**Returns:** `(*http.Response, error)` - 2 values
**When to use:** Streaming, large responses, custom body processing

#### Curl

```go
func Curl(ctx context.Context, command ...string) (*http.Response, error)
```

Auto-detects input format:
- **Single argument:** Parsed as shell command (supports multi-line, backslashes, comments)
- **Multiple arguments:** Treated as separate arguments

Environment variables (`$VAR` and `${VAR}`) are automatically expanded.

**Examples:**

```go
// Single argument - parsed as shell command
resp, err := gocurl.Curl(ctx, "curl -H 'X-Token: abc' https://example.com")
if err != nil {
    return err
}
defer resp.Body.Close()

// Multiple arguments - treated as separate tokens
resp, err := gocurl.Curl(ctx, "-H", "X-Token: abc", "https://example.com")
if err != nil {
    return err
}
defer resp.Body.Close()

// Multi-line command
resp, err := gocurl.Curl(ctx, `
    curl -X POST https://api.example.com \
      -H 'Content-Type: application/json' \
      -d '{"key":"value"}'
`)
```

#### CurlCommand

```go
func CurlCommand(ctx context.Context, command string) (*http.Response, error)
```

Executes a curl command from a **shell-style string only**.
Handles multi-line, backslash continuations, comments.
Environment variables automatically expanded.

**Example:**

```go
cmd := `curl -X GET https://api.github.com/users/octocat \
    -H "Accept: application/vnd.github+json"`

resp, err := gocurl.CurlCommand(ctx, cmd)
if err != nil {
    return err
}
defer resp.Body.Close()
```

#### CurlArgs

```go
func CurlArgs(ctx context.Context, args ...string) (*http.Response, error)
```

Executes curl command from **variadic arguments** (like `os.Args`).
Each argument is a separate token.
Environment variables automatically expanded.

**Example:**

```go
resp, err := gocurl.CurlArgs(ctx,
    "-X", "POST",
    "-H", "Content-Type: application/json",
    "-d", `{"name":"test"}`,
    "https://api.example.com/items",
)
if err != nil {
    return err
}
defer resp.Body.Close()
```

---

### String Functions

**Purpose:** Return response body as string (automatically read)
**Returns:** `(string, *http.Response, error)` - 3 values (body FIRST)
**When to use:** Text responses (HTML, JSON, XML), small payloads (< 10MB)

> **Warning:** These functions load the entire response body into memory.

#### CurlString

```go
func CurlString(ctx context.Context, command ...string) (string, *http.Response, error)
```

Auto-detects format (same as `Curl`). Returns body as string.

**Example:**

```go
// Body is the FIRST return value
body, resp, err := gocurl.CurlString(ctx, "https://api.github.com/users/octocat")
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Println("Body:", body)
fmt.Println("Status:", resp.StatusCode)
```

#### CurlStringCommand

```go
func CurlStringCommand(ctx context.Context, command string) (string, *http.Response, error)
```

Shell-style command only. Returns body as string.

**Example:**

```go
body, resp, err := gocurl.CurlStringCommand(ctx,
    `curl -H 'Accept: text/html' https://example.com`,
)
if err != nil {
    return err
}
defer resp.Body.Close()

// Process HTML body
fmt.Println(body)
```

#### CurlStringArgs

```go
func CurlStringArgs(ctx context.Context, args ...string) (string, *http.Response, error)
```

Variadic arguments. Returns body as string.

**Example:**

```go
body, resp, err := gocurl.CurlStringArgs(ctx,
    "-H", "Accept: application/xml",
    "https://api.example.com/data.xml",
)
if err != nil {
    return err
}
defer resp.Body.Close()

// Process XML body
fmt.Println(body)
```

---

### Bytes Functions

**Purpose:** Return response body as byte slice (automatically read)
**Returns:** `([]byte, *http.Response, error)` - 3 values (body FIRST)
**When to use:** Binary data, images, files, raw data processing

> **Warning:** These functions load the entire response body into memory.

#### CurlBytes

```go
func CurlBytes(ctx context.Context, command ...string) ([]byte, *http.Response, error)
```

Auto-detects format. Returns body as bytes.

**Example:**

```go
// Download an image
data, resp, err := gocurl.CurlBytes(ctx, "https://example.com/image.png")
if err != nil {
    return err
}
defer resp.Body.Close()

// Write to file
err = os.WriteFile("image.png", data, 0644)
```

#### CurlBytesCommand

```go
func CurlBytesCommand(ctx context.Context, command string) ([]byte, *http.Response, error)
```

Shell-style command. Returns body as bytes.

**Example:**

```go
data, resp, err := gocurl.CurlBytesCommand(ctx, "curl https://example.com/data.bin")
if err != nil {
    return err
}
defer resp.Body.Close()

// Process binary data
fmt.Printf("Received %d bytes\n", len(data))
```

#### CurlBytesArgs

```go
func CurlBytesArgs(ctx context.Context, args ...string) ([]byte, *http.Response, error)
```

Variadic arguments. Returns body as bytes.

**Example:**

```go
data, resp, err := gocurl.CurlBytesArgs(ctx,
    "-H", "Accept: application/octet-stream",
    "https://example.com/download",
)
if err != nil {
    return err
}
defer resp.Body.Close()
```

---

### JSON Functions

**Purpose:** Unmarshal JSON response directly into struct
**Returns:** `(*http.Response, error)` - 2 values
**When to use:** Structured API responses with known schema

> **Note:** JSON is unmarshaled into the provided variable `v`.

#### CurlJSON

```go
func CurlJSON(ctx context.Context, v interface{}, command ...string) (*http.Response, error)
```

Auto-detects format. Unmarshals JSON into `v`.

**Example:**

```go
type User struct {
    Login string `json:"login"`
    Name  string `json:"name"`
}

var user User
resp, err := gocurl.CurlJSON(ctx, &user, "https://api.github.com/users/octocat")
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Printf("User: %s (%s)\n", user.Name, user.Login)
```

#### CurlJSONCommand

```go
func CurlJSONCommand(ctx context.Context, v interface{}, command string) (*http.Response, error)
```

Shell-style command. Unmarshals JSON into `v`.

**Example:**

```go
type Repository struct {
    Name        string `json:"name"`
    Stars       int    `json:"stargazers_count"`
    Description string `json:"description"`
}

var repo Repository
resp, err := gocurl.CurlJSONCommand(ctx, &repo,
    `curl -H "Accept: application/vnd.github+json" https://api.github.com/repos/golang/go`,
)
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Printf("%s: %d stars\n", repo.Name, repo.Stars)
```

#### CurlJSONArgs

```go
func CurlJSONArgs(ctx context.Context, v interface{}, args ...string) (*http.Response, error)
```

Variadic arguments. Unmarshals JSON into `v`.

**Example:**

```go
type ApiResponse struct {
    Status string `json:"status"`
    Data   []Item `json:"data"`
}

var response ApiResponse
resp, err := gocurl.CurlJSONArgs(ctx, &response,
    "-X", "GET",
    "-H", "Authorization: Bearer token",
    "https://api.example.com/items",
)
if err != nil {
    return err
}
defer resp.Body.Close()

for _, item := range response.Data {
    fmt.Println(item)
}
```

---

### Download Functions

**Purpose:** Download response body to file
**Returns:** `(int64, *http.Response, error)` - 3 values (bytes written, response, error)
**When to use:** Large files, downloads, saving responses to disk

#### CurlDownload

```go
func CurlDownload(ctx context.Context, outputPath, command string, args ...string) (int64, *http.Response, error)
```

Auto-detects format. Downloads to `outputPath`.

**Example:**

```go
bytesWritten, resp, err := gocurl.CurlDownload(ctx,
    "image.png",
    "https://example.com/image.png",
)
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Printf("Downloaded %d bytes to image.png\n", bytesWritten)
```

#### CurlDownloadCommand

```go
func CurlDownloadCommand(ctx context.Context, outputPath, command string) (int64, *http.Response, error)
```

Shell-style command. Downloads to `outputPath`.

**Example:**

```go
bytesWritten, resp, err := gocurl.CurlDownloadCommand(ctx,
    "data.json",
    `curl -H "Accept: application/json" https://api.example.com/export`,
)
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Printf("Saved %d bytes\n", bytesWritten)
```

#### CurlDownloadArgs

```go
func CurlDownloadArgs(ctx context.Context, outputPath string, args ...string) (int64, *http.Response, error)
```

Variadic arguments. Downloads to `outputPath`.

**Example:**

```go
bytesWritten, resp, err := gocurl.CurlDownloadArgs(ctx,
    "archive.zip",
    "-L", // Follow redirects
    "https://github.com/user/repo/archive/main.zip",
)
if err != nil {
    return err
}
defer resp.Body.Close()
```

---

### WithVars Functions

**Purpose:** Explicit variable substitution (NO environment expansion)
**Returns:** `(*http.Response, error)` - 2 values
**When to use:** Security-critical code, testing, controlled environments

> **Important:** These functions do NOT expand environment variables.

#### CurlWithVars

```go
func CurlWithVars(ctx context.Context, vars map[string]string, command ...string) (*http.Response, error)
```

Auto-detects format. Uses explicit variables only.

**Example:**

```go
vars := map[string]string{
    "API_KEY": "secret-key-123",
    "USER_ID": "12345",
}

resp, err := gocurl.CurlWithVars(ctx, vars,
    "-H", "Authorization: Bearer $API_KEY",
    "https://api.example.com/users/$USER_ID",
)
if err != nil {
    return err
}
defer resp.Body.Close()

// $API_KEY and $USER_ID are expanded from vars map only
// Environment variables are NOT expanded
```

#### CurlCommandWithVars

```go
func CurlCommandWithVars(ctx context.Context, vars map[string]string, command string) (*http.Response, error)
```

Shell-style command with explicit variables.

**Example:**

```go
vars := map[string]string{
    "ENDPOINT": "https://api.example.com",
    "TOKEN":    "abc123",
}

resp, err := gocurl.CurlCommandWithVars(ctx, vars,
    `curl -H "Authorization: Bearer $TOKEN" $ENDPOINT/data`,
)
if err != nil {
    return err
}
defer resp.Body.Close()
```

#### CurlArgsWithVars

```go
func CurlArgsWithVars(ctx context.Context, vars map[string]string, args ...string) (*http.Response, error)
```

Variadic arguments with explicit variables.

**Example:**

```go
vars := map[string]string{
    "BASE_URL": "https://api.github.com",
    "USERNAME": "octocat",
}

resp, err := gocurl.CurlArgsWithVars(ctx, vars,
    "-H", "Accept: application/json",
    "$BASE_URL/users/$USERNAME",
)
if err != nil {
    return err
}
defer resp.Body.Close()
```

---

## RequestOptions API

The `RequestOptions` struct provides fine-grained control over HTTP requests.

### Constructor

```go
func NewRequestOptions(url string) *RequestOptions
```

Creates a new `RequestOptions` with defaults aligned to curl's behavior.

**Example:**

```go
opts := gocurl.NewRequestOptions("https://api.example.com/data")
opts.Method = "GET"
opts.AddHeader("Accept", "application/json")

resp, err := gocurl.Process(ctx, opts)
```

### RequestOptions Fields

```go
type RequestOptions struct {
    // HTTP request basics
    Method      string      // HTTP method (GET, POST, etc.)
    URL         string      // Request URL
    Headers     http.Header // HTTP headers
    Body        string      // Request body
    Form        url.Values  // Form data (application/x-www-form-urlencoded)
    QueryParams url.Values  // URL query parameters

    // Authentication
    BasicAuth   *BasicAuth  // HTTP Basic Authentication
    BearerToken string      // Bearer token (Authorization: Bearer ...)

    // TLS/SSL options
    CertFile            string      // Client certificate file
    KeyFile             string      // Client private key file
    CAFile              string      // CA certificate file
    Insecure            bool        // Skip TLS verification (insecure!)
    TLSConfig           *tls.Config // Custom TLS configuration
    CertPinFingerprints []string    // SHA256 fingerprints for cert pinning
    SNIServerName       string      // Server name for SNI

    // Proxy settings
    Proxy        string   // Proxy URL (http://proxy:port)
    ProxyNoProxy []string // Domains to exclude from proxying

    // Timeout settings
    Timeout        time.Duration // Total request timeout
    ConnectTimeout time.Duration // Connection timeout

    // Redirect behavior
    FollowRedirects bool // Follow HTTP redirects
    MaxRedirects    int  // Maximum number of redirects

    // Compression
    Compress           bool     // Enable compression
    CompressionMethods []string // Specific methods: gzip, deflate, br

    // HTTP version
    HTTP2     bool // Enable HTTP/2
    HTTP2Only bool // Force HTTP/2 only

    // Cookie handling
    Cookies    []*http.Cookie // Cookies to send
    CookieJar  http.CookieJar // Cookie jar for persistence
    CookieFile string         // File to read/write cookies

    // Custom options
    UserAgent string // User-Agent header
    Referer   string // Referer header

    // File upload
    FileUpload *FileUpload // Multipart file upload config

    // Retry configuration
    RetryConfig *RetryConfig // Automatic retry config

    // Output options
    OutputFile string // Save response to file
    Silent     bool   // Suppress output
    Verbose    bool   // Verbose logging

    // Advanced options
    RequestID         string          // Request ID for tracing
    Middleware        []MiddlewareFunc // Request middleware chain
    ResponseBodyLimit int64           // Limit response body size
    CustomClient      HTTPClient      // Custom HTTP client for testing
}
```

### RequestOptions Methods

#### Clone

```go
func (ro *RequestOptions) Clone() *RequestOptions
```

Creates a deep copy of `RequestOptions`. Essential for concurrent use.

**Example:**

```go
baseOpts := gocurl.NewRequestOptions("https://api.example.com")
baseOpts.AddHeader("Authorization", "Bearer token")

// Safe: Clone before modification in each goroutine
go func() {
    opts1 := baseOpts.Clone()
    opts1.AddQueryParam("id", "1")
    resp, _ := gocurl.Process(ctx, opts1)
    // ...
}()

go func() {
    opts2 := baseOpts.Clone()
    opts2.AddQueryParam("id", "2")
    resp, _ := gocurl.Process(ctx, opts2)
    // ...
}()
```

#### ToJSON

```go
func (ro *RequestOptions) ToJSON() (string, error)
```

Marshals `RequestOptions` to JSON format.

**Example:**

```go
opts := gocurl.NewRequestOptions("https://example.com")
opts.Method = "POST"
opts.AddHeader("Content-Type", "application/json")

jsonStr, err := opts.ToJSON()
if err != nil {
    return err
}

fmt.Println(jsonStr)
// Output: {"method":"POST","url":"https://example.com","headers":{"Content-Type":["application/json"]},...}
```

#### AddHeader

```go
func (ro *RequestOptions) AddHeader(key, value string)
```

Adds a header to the request. Appends to existing values.

**Example:**

```go
opts := gocurl.NewRequestOptions("https://api.example.com")
opts.AddHeader("Accept", "application/json")
opts.AddHeader("X-Custom-Header", "value1")
opts.AddHeader("X-Custom-Header", "value2") // Multiple values
```

#### SetHeader

```go
func (ro *RequestOptions) SetHeader(key, value string)
```

Sets a header, replacing any existing values.

**Example:**

```go
opts := gocurl.NewRequestOptions("https://api.example.com")
opts.SetHeader("Content-Type", "application/json")
opts.SetHeader("Content-Type", "text/plain") // Replaces previous value
```

#### AddQueryParam

```go
func (ro *RequestOptions) AddQueryParam(key, value string)
```

Adds a query parameter to the URL.

**Example:**

```go
opts := gocurl.NewRequestOptions("https://api.example.com/search")
opts.AddQueryParam("q", "golang")
opts.AddQueryParam("limit", "10")
// URL becomes: https://api.example.com/search?q=golang&limit=10
```

---

## Builder Pattern API

The `RequestOptionsBuilder` provides a fluent API for constructing requests.

### Constructor

```go
func NewRequestOptionsBuilder() *RequestOptionsBuilder
```

Creates a new builder instance.

**Example:**

```go
builder := gocurl.NewRequestOptionsBuilder()
```

### Builder Methods

All builder methods return `*RequestOptionsBuilder` for chaining.

#### SetMethod

```go
func (b *RequestOptionsBuilder) SetMethod(method string) *RequestOptionsBuilder
```

Sets the HTTP method.

#### SetURL

```go
func (b *RequestOptionsBuilder) SetURL(url string) *RequestOptionsBuilder
```

Sets the request URL.

#### AddHeader

```go
func (b *RequestOptionsBuilder) AddHeader(key, value string) *RequestOptionsBuilder
```

Adds a header to the request.

#### SetHeaders

```go
func (b *RequestOptionsBuilder) SetHeaders(headers http.Header) *RequestOptionsBuilder
```

Sets multiple headers at once.

#### SetBody

```go
func (b *RequestOptionsBuilder) SetBody(body string) *RequestOptionsBuilder
```

Sets the request body.

#### SetForm

```go
func (b *RequestOptionsBuilder) SetForm(form url.Values) *RequestOptionsBuilder
```

Sets form data (application/x-www-form-urlencoded).

#### SetQueryParams

```go
func (b *RequestOptionsBuilder) SetQueryParams(queryParams url.Values) *RequestOptionsBuilder
```

Sets query parameters.

#### AddQueryParam

```go
func (b *RequestOptionsBuilder) AddQueryParam(key, value string) *RequestOptionsBuilder
```

Adds a single query parameter.

#### SetBasicAuth

```go
func (b *RequestOptionsBuilder) SetBasicAuth(username, password string) *RequestOptionsBuilder
```

Sets HTTP Basic Authentication.

#### SetBearerToken

```go
func (b *RequestOptionsBuilder) SetBearerToken(token string) *RequestOptionsBuilder
```

Sets Bearer token authentication.

#### SetCertFile

```go
func (b *RequestOptionsBuilder) SetCertFile(certFile string) *RequestOptionsBuilder
```

Sets client certificate file path.

#### SetKeyFile

```go
func (b *RequestOptionsBuilder) SetKeyFile(keyFile string) *RequestOptionsBuilder
```

Sets client private key file path.

#### SetCAFile

```go
func (b *RequestOptionsBuilder) SetCAFile(caFile string) *RequestOptionsBuilder
```

Sets CA certificate file path.

#### SetInsecure

```go
func (b *RequestOptionsBuilder) SetInsecure(insecure bool) *RequestOptionsBuilder
```

Skips TLS verification (insecure).

#### SetTLSConfig

```go
func (b *RequestOptionsBuilder) SetTLSConfig(tlsConfig *tls.Config) *RequestOptionsBuilder
```

Sets custom TLS configuration.

#### SetProxy

```go
func (b *RequestOptionsBuilder) SetProxy(proxy string) *RequestOptionsBuilder
```

Sets proxy URL.

#### SetTimeout

```go
func (b *RequestOptionsBuilder) SetTimeout(timeout time.Duration) *RequestOptionsBuilder
```

Sets total request timeout.

#### SetConnectTimeout

```go
func (b *RequestOptionsBuilder) SetConnectTimeout(connectTimeout time.Duration) *RequestOptionsBuilder
```

Sets connection timeout.

#### SetFollowRedirects

```go
func (b *RequestOptionsBuilder) SetFollowRedirects(follow bool) *RequestOptionsBuilder
```

Enables/disables redirect following.

#### SetMaxRedirects

```go
func (b *RequestOptionsBuilder) SetMaxRedirects(maxRedirects int) *RequestOptionsBuilder
```

Sets maximum redirect count.

#### SetCompress

```go
func (b *RequestOptionsBuilder) SetCompress(compress bool) *RequestOptionsBuilder
```

Enables/disables compression.

#### SetHTTP2

```go
func (b *RequestOptionsBuilder) SetHTTP2(http2 bool) *RequestOptionsBuilder
```

Enables/disables HTTP/2.

#### SetHTTP2Only

```go
func (b *RequestOptionsBuilder) SetHTTP2Only(http2Only bool) *RequestOptionsBuilder
```

Forces HTTP/2 only mode.

#### SetCookie

```go
func (b *RequestOptionsBuilder) SetCookie(cookie *http.Cookie) *RequestOptionsBuilder
```

Adds a cookie to the request.

#### SetUserAgent

```go
func (b *RequestOptionsBuilder) SetUserAgent(userAgent string) *RequestOptionsBuilder
```

Sets custom User-Agent header. If not set, defaults to `gocurl/dev` (or `gocurl/VERSION` in releases), following curl's behavior.

#### SetReferer

```go
func (b *RequestOptionsBuilder) SetReferer(referer string) *RequestOptionsBuilder
```

Sets Referer header.

#### SetFileUpload

```go
func (b *RequestOptionsBuilder) SetFileUpload(fileUpload *FileUpload) *RequestOptionsBuilder
```

Sets file upload configuration.

#### SetRetryConfig

```go
func (b *RequestOptionsBuilder) SetRetryConfig(retryConfig *RetryConfig) *RequestOptionsBuilder
```

Sets retry configuration.

#### SetOutputFile

```go
func (b *RequestOptionsBuilder) SetOutputFile(outputFile string) *RequestOptionsBuilder
```

Sets output file path.

#### SetSilent

```go
func (b *RequestOptionsBuilder) SetSilent(silent bool) *RequestOptionsBuilder
```

Enables/disables silent mode.

#### SetVerbose

```go
func (b *RequestOptionsBuilder) SetVerbose(verbose bool) *RequestOptionsBuilder
```

Enables/disables verbose logging.

#### WithContext

```go
func (b *RequestOptionsBuilder) WithContext(ctx context.Context) *RequestOptionsBuilder
```

Sets context for the request.

#### WithTimeout

```go
func (b *RequestOptionsBuilder) WithTimeout(timeout time.Duration) *RequestOptionsBuilder
```

Creates context with timeout.

#### Cleanup

```go
func (b *RequestOptionsBuilder) Cleanup()
```

Cancels context and cleans up resources.

#### Validate

```go
func (b *RequestOptionsBuilder) Validate() error
```

Validates the request configuration.

#### Build

```go
func (b *RequestOptionsBuilder) Build() *RequestOptions
```

Builds and returns the `RequestOptions`.

**Complete Example:**

```go
opts := gocurl.NewRequestOptionsBuilder().
    SetMethod("POST").
    SetURL("https://api.example.com/items").
    AddHeader("Content-Type", "application/json").
    AddHeader("Authorization", "Bearer token123").
    SetBody(`{"name":"test","value":42}`).
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
    return err
}
defer resp.Body.Close()
```

---

## Middleware API

Middleware allows request transformation before execution.

### MiddlewareFunc Type

```go
type MiddlewareFunc func(*http.Request) (*http.Request, error)
```

A middleware function transforms an `http.Request`.

**Example:**

```go
// Add request ID to all requests
func RequestIDMiddleware(req *http.Request) (*http.Request, error) {
    reqID := uuid.New().String()
    req.Header.Set("X-Request-ID", reqID)
    return req, nil
}

// Add timestamp header
func TimestampMiddleware(req *http.Request) (*http.Request, error) {
    req.Header.Set("X-Timestamp", time.Now().Format(time.RFC3339))
    return req, nil
}

// Use middleware
opts := gocurl.NewRequestOptions("https://api.example.com")
opts.Middleware = []gocurl.MiddlewareFunc{
    RequestIDMiddleware,
    TimestampMiddleware,
}

resp, err := gocurl.Process(ctx, opts)
```

---

## Process Function

The core execution function that all Curl functions call internally.

```go
func Process(ctx context.Context, opts *RequestOptions) (*http.Response, error)
```

**When to use:**
- Direct control over request options
- Custom retry logic
- Advanced configuration
- Testing and mocking

**Example:**

```go
opts := gocurl.NewRequestOptions("https://api.example.com/data")
opts.Method = "POST"
opts.AddHeader("Content-Type", "application/json")
opts.Body = `{"key":"value"}`
opts.RetryConfig = &gocurl.RetryConfig{
    MaxRetries:  3,
    RetryDelay:  time.Second,
    RetryOnHTTP: []int{500, 502, 503, 504},
}

resp, err := gocurl.Process(ctx, opts)
if err != nil {
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != 200 {
    return fmt.Errorf("unexpected status: %d", resp.StatusCode)
}
```

---

## Legacy API (Deprecated)

These functions are deprecated and maintained for backward compatibility only.
**Recommendation:** Migrate to modern API (Curl functions or Builder pattern).

### Request (Deprecated)

```go
func Request(command string, vars map[string]string) (*Response, error)
```

Legacy function using background context. Use `Curl` instead.

### RequestWithContext (Deprecated)

```go
func RequestWithContext(ctx context.Context, command string, vars map[string]string) (*Response, error)
```

Legacy function with context. Use `Curl` or `CurlWithVars` instead.

### Execute (Deprecated)

```go
func Execute(ctx context.Context, opts *RequestOptions) (*Response, error)
```

Legacy execution function. Use `Process` instead.

### Response Type (Deprecated)

```go
type Response struct {
    response *http.Response
}
```

Legacy wrapper around `http.Response`. Modern API returns `*http.Response` directly.

**Methods:**

```go
func (r *Response) String() (string, error)  // Use CurlString instead
func (r *Response) Bytes() ([]byte, error)   // Use CurlBytes instead
func (r *Response) JSON(v interface{}) error // Use CurlJSON instead
```

**Migration Example:**

```go
// OLD (Deprecated)
resp, err := gocurl.Request("curl https://example.com", nil)
if err != nil {
    return err
}
body, _ := resp.String()

// NEW (Recommended)
body, resp, err := gocurl.CurlString(ctx, "https://example.com")
if err != nil {
    return err
}
defer resp.Body.Close()
```

See Appendix B in the book for complete migration guide.

---

## Supporting Types

### BasicAuth

```go
type BasicAuth struct {
    Username string
    Password string
}
```

HTTP Basic Authentication credentials.

**Example:**

```go
opts := gocurl.NewRequestOptions("https://api.example.com")
opts.BasicAuth = &gocurl.BasicAuth{
    Username: "user",
    Password: "pass",
}
```

### FileUpload

```go
type FileUpload struct {
    FieldName string // Form field name
    FileName  string // File name in request
    FilePath  string // Local file path
}
```

Multipart file upload configuration.

**Example:**

```go
opts := gocurl.NewRequestOptions("https://api.example.com/upload")
opts.Method = "POST"
opts.FileUpload = &gocurl.FileUpload{
    FieldName: "file",
    FileName:  "document.pdf",
    FilePath:  "/path/to/document.pdf",
}
```

### RetryConfig

```go
type RetryConfig struct {
    MaxRetries  int           // Maximum retry attempts
    RetryDelay  time.Duration // Delay between retries
    RetryOnHTTP []int         // HTTP status codes to retry on
}
```

Automatic retry configuration.

**Example:**

```go
opts := gocurl.NewRequestOptions("https://api.example.com")
opts.RetryConfig = &gocurl.RetryConfig{
    MaxRetries:  3,
    RetryDelay:  time.Second,
    RetryOnHTTP: []int{500, 502, 503, 504},
}
```

### HTTPClient Interface

```go
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}
```

Custom HTTP client interface for testing/mocking.

**Example:**

```go
type MockClient struct{}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
    // Return mock response
    return &http.Response{
        StatusCode: 200,
        Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
    }, nil
}

opts := gocurl.NewRequestOptions("https://api.example.com")
opts.CustomClient = &MockClient{}

resp, err := gocurl.Process(ctx, opts)
```

---

## Quick Reference

### Function Return Signatures

| Category | Functions | Returns |
|----------|-----------|---------|
| Basic | `Curl`, `CurlCommand`, `CurlArgs` | `(*http.Response, error)` |
| String | `CurlString*` | `(string, *http.Response, error)` |
| Bytes | `CurlBytes*` | `([]byte, *http.Response, error)` |
| JSON | `CurlJSON*` | `(*http.Response, error)` |
| Download | `CurlDownload*` | `(int64, *http.Response, error)` |
| WithVars | `Curl*WithVars` | `(*http.Response, error)` |
| Core | `Process` | `(*http.Response, error)` |

### Decision Tree

```
What do you need?

├─ Quick curl conversion → Curl() or CurlCommand()
├─ Variadic arguments → CurlArgs() or CurlStringArgs()
├─ Response as string → CurlString*()
├─ Response as bytes → CurlBytes*()
├─ JSON unmarshaling → CurlJSON*()
├─ Download to file → CurlDownload*()
├─ No env expansion → Curl*WithVars()
├─ Fine control → NewRequestOptions() + Process()
└─ Fluent API → NewRequestOptionsBuilder().Build()
```

---

**END OF API REFERENCE**
