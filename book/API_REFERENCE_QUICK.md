# GoCurl API Quick Reference

**CRITICAL:** Use this reference for all book examples!

## Primary API Functions

### Basic Request Functions (Return Response Only)

```go
// Returns: (*http.Response, error)
func Curl(ctx context.Context, command ...string) (*http.Response, error)
func CurlCommand(ctx context.Context, command string) (*http.Response, error)
func CurlArgs(ctx context.Context, args ...string) (*http.Response, error)
func CurlWithVars(ctx context.Context, vars Variables, command ...string) (*http.Response, error)
func CurlCommandWithVars(ctx context.Context, vars Variables, command string) (*http.Response, error)
func CurlArgsWithVars(ctx context.Context, vars Variables, args ...string) (*http.Response, error)
```

**Example Usage:**
```go
resp, err := gocurl.Curl(ctx, "https://api.github.com/zen")
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

body, err := io.ReadAll(resp.Body)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(body))
```

### Convenience Functions (Auto-Read Body)

```go
// Returns: (string, *http.Response, error) - body as string
func CurlString(ctx context.Context, command ...string) (string, *http.Response, error)
func CurlStringCommand(ctx context.Context, command string) (string, *http.Response, error)
func CurlStringArgs(ctx context.Context, args ...string) (string, *http.Response, error)

// Returns: ([]byte, *http.Response, error) - body as bytes
func CurlBytes(ctx context.Context, command ...string) ([]byte, *http.Response, error)
func CurlBytesCommand(ctx context.Context, command string) ([]byte, *http.Response, error)
func CurlBytesArgs(ctx context.Context, args ...string) ([]byte, *http.Response, error)

// Returns: (*http.Response, error) - decodes JSON into provided struct
func CurlJSON(ctx context.Context, v interface{}, command ...string) (*http.Response, error)
func CurlJSONCommand(ctx context.Context, v interface{}, command string) (*http.Response, error)
func CurlJSONArgs(ctx context.Context, v interface{}, args ...string) (*http.Response, error)

// Returns: (int64, *http.Response, error) - bytes written
func CurlDownload(ctx context.Context, filepath string, command ...string) (int64, *http.Response, error)
func CurlDownloadCommand(ctx context.Context, filepath string, command string) (int64, *http.Response, error)
func CurlDownloadArgs(ctx context.Context, filepath string, args ...string) (int64, *http.Response, error)
```

**Example Usage:**
```go
// String body
body, resp, err := gocurl.CurlString(ctx, "https://api.github.com/zen")
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()
fmt.Println("Body:", body)

// JSON decoding
var user User
resp, err := gocurl.CurlJSON(ctx, &user, "https://api.github.com/users/octocat")
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()
fmt.Printf("User: %+v\n", user)
```

## COMMON MISTAKES TO AVOID

### ❌ WRONG - Using three return values with Curl()
```go
// This is WRONG - Curl() only returns 2 values!
resp, body, err := gocurl.Curl(ctx, url)  // ❌ COMPILE ERROR
```

### ✅ CORRECT - Option 1: Use Curl() and read body manually
```go
resp, err := gocurl.Curl(ctx, url)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

body, err := io.ReadAll(resp.Body)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(body))
```

### ✅ CORRECT - Option 2: Use CurlString() for auto-read
```go
body, resp, err := gocurl.CurlString(ctx, url)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()
fmt.Println(body)
```

## Decision Tree: Which Function to Use?

```
Do you need the response body?
│
├─ No (just status/headers)
│  └─ Use: Curl() → (*http.Response, error)
│
├─ Yes, as string
│  └─ Use: CurlString() → (string, *http.Response, error)
│
├─ Yes, as []byte
│  └─ Use: CurlBytes() → ([]byte, *http.Response, error)
│
├─ Yes, as JSON struct
│  └─ Use: CurlJSON() → (*http.Response, error)
│
└─ Yes, save to file
   └─ Use: CurlDownload() → (int64, *http.Response, error)
```

## Best Practices for Book Examples

### ✅ For Simple Examples - Use CurlString()
```go
body, resp, err := gocurl.CurlString(ctx, "https://api.github.com/zen")
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()
fmt.Println(body)
```

### ✅ For JSON APIs - Use CurlJSON()
```go
var user User
resp, err := gocurl.CurlJSON(ctx, &user, url)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()
fmt.Printf("User: %s\n", user.Login)
```

### ✅ For Advanced/Custom Handling - Use Curl()
```go
resp, err := gocurl.Curl(ctx, url)
if err != nil {
    return err
}
defer resp.Body.Close()

// Custom body processing
scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
    processLine(scanner.Text())
}
```

## Internal Function (Don't Use in Book)

```go
// This is internal - use the Curl* functions instead
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error)
```

---

**Last Updated:** October 17, 2025
**Version:** 1.0
**Always verify against this reference before writing examples!**
