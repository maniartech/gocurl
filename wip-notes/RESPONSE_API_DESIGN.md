# GoCurl Response API Design - October 15, 2025

## Problem: Current String-Based Return

```go
// ❌ Current design - returns string
func Curl(ctx context.Context, command ...string) (*Response, string, error)
```

**Issues:**
- Memory inefficient (loads entire response)
- Can't handle binary data
- No streaming support
- Forces encoding assumptions

## Recommended Solution: Flexible Response Access

### Core API - Returns Response Object

```go
// Curl - Returns http.Response with body ready to read
// User controls how to consume the response body
func Curl(ctx context.Context, command ...string) (*http.Response, error)

// CurlCommand - Explicit shell command
func CurlCommand(ctx context.Context, shellCommand string) (*http.Response, error)

// CurlArgs - Explicit variadic
func CurlArgs(ctx context.Context, args ...string) (*http.Response, error)
```

**Usage:**
```go
// Get response
resp, err := gocurl.Curl(ctx, url)
if err != nil {
    return err
}
defer resp.Body.Close()

// User chooses how to read:
// 1. Read to string
body, _ := io.ReadAll(resp.Body)
bodyStr := string(body)

// 2. Stream to file
file, _ := os.Create("output.bin")
io.Copy(file, resp.Body)

// 3. Stream processing
scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
    processLine(scanner.Text())
}

// 4. JSON decode directly
var result MyStruct
json.NewDecoder(resp.Body).Decode(&result)
```

### Convenience Helpers

```go
// CurlString - Convenience for text responses
// Returns body as string (for backward compatibility and common case)
func CurlString(ctx context.Context, command ...string) (string, *http.Response, error)

// CurlBytes - Convenience for binary responses
// Returns body as bytes
func CurlBytes(ctx context.Context, command ...string) ([]byte, *http.Response, error)

// CurlJSON - Convenience for JSON responses
// Decodes directly to struct
func CurlJSON(ctx context.Context, result interface{}, command ...string) (*http.Response, error)

// CurlStream - Returns response with streaming guaranteed
// Does NOT buffer the body
func CurlStream(ctx context.Context, command ...string) (*http.Response, error)
```

## Complete API Design

### 1. Core Functions (Return http.Response)

```go
// Curl - Core API, returns http.Response
// User has full control over body handling
//
// Example:
//   resp, err := gocurl.Curl(ctx, url)
//   defer resp.Body.Close()
//   body, _ := io.ReadAll(resp.Body)
func Curl(ctx context.Context, command ...string) (*http.Response, error)

// CurlCommand - Explicit shell command parsing
func CurlCommand(ctx context.Context, shellCommand string) (*http.Response, error)

// CurlArgs - Explicit variadic arguments
func CurlArgs(ctx context.Context, args ...string) (*http.Response, error)
```

### 2. String Convenience (Auto-reads to string)

```go
// CurlString - Reads body to string automatically
// Returns: (body string, response, error)
//
// Use for text responses where you need the full body as string.
// Body is automatically closed.
//
// Example:
//   body, resp, err := gocurl.CurlString(ctx, url)
//   if resp.StatusCode != 200 {
//       log.Printf("Error: %s", body)
//   }
func CurlString(ctx context.Context, command ...string) (string, *http.Response, error) {
    resp, err := Curl(ctx, command...)
    if err != nil {
        return "", nil, err
    }
    defer resp.Body.Close()

    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", resp, fmt.Errorf("failed to read response body: %w", err)
    }

    return string(bodyBytes), resp, nil
}

// CurlCommandString - Shell command variant
func CurlCommandString(ctx context.Context, shellCommand string) (string, *http.Response, error)

// CurlArgsString - Variadic variant
func CurlArgsString(ctx context.Context, args ...string) (string, *http.Response, error)
```

### 3. Bytes Convenience (Auto-reads to []byte)

```go
// CurlBytes - Reads body to bytes automatically
// Returns: (body []byte, response, error)
//
// Use for binary responses or when you need []byte.
// Body is automatically closed.
//
// Example:
//   data, resp, err := gocurl.CurlBytes(ctx, imageURL)
//   os.WriteFile("image.png", data, 0644)
func CurlBytes(ctx context.Context, command ...string) ([]byte, *http.Response, error) {
    resp, err := Curl(ctx, command...)
    if err != nil {
        return nil, nil, err
    }
    defer resp.Body.Close()

    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, resp, fmt.Errorf("failed to read response body: %w", err)
    }

    return bodyBytes, resp, nil
}

// CurlCommandBytes - Shell command variant
func CurlCommandBytes(ctx context.Context, shellCommand string) ([]byte, *http.Response, error)

// CurlArgsBytes - Variadic variant
func CurlArgsBytes(ctx context.Context, args ...string) ([]byte, *http.Response, error)
```

### 4. JSON Convenience (Auto-decodes JSON)

```go
// CurlJSON - Decodes JSON response directly to struct
// Returns: (response, error)
//
// Use for JSON APIs where you want automatic decoding.
// Body is automatically closed after decoding.
//
// Example:
//   var user User
//   resp, err := gocurl.CurlJSON(ctx, &user,
//       "-H", "Accept: application/json",
//       "https://api.example.com/user/123",
//   )
func CurlJSON(ctx context.Context, result interface{}, command ...string) (*http.Response, error) {
    resp, err := Curl(ctx, command...)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
        return resp, fmt.Errorf("failed to decode JSON: %w", err)
    }

    return resp, nil
}

// CurlCommandJSON - Shell command variant
func CurlCommandJSON(ctx context.Context, result interface{}, shellCommand string) (*http.Response, error)

// CurlArgsJSON - Variadic variant
func CurlArgsJSON(ctx context.Context, result interface{}, args ...string) (*http.Response, error)
```

### 5. Streaming (Guaranteed no buffering)

```go
// CurlStream - Returns response optimized for streaming
// Ensures body is NOT buffered, suitable for large files.
//
// Use for:
// - Large file downloads
// - Server-sent events
// - Chunked responses
// - When you need line-by-line processing
//
// Example:
//   resp, err := gocurl.CurlStream(ctx, largeFileURL)
//   defer resp.Body.Close()
//
//   scanner := bufio.NewScanner(resp.Body)
//   for scanner.Scan() {
//       processLine(scanner.Text())
//   }
func CurlStream(ctx context.Context, command ...string) (*http.Response, error) {
    // Same as Curl() but documents intent
    return Curl(ctx, command...)
}
```

### 6. File Download (Direct to disk)

```go
// CurlDownload - Downloads response directly to file
// Returns: (bytesWritten int64, response, error)
//
// Use for downloading files efficiently.
// Streams directly to disk without loading into memory.
//
// Example:
//   n, resp, err := gocurl.CurlDownload(ctx, "output.zip",
//       "https://example.com/file.zip",
//   )
//   fmt.Printf("Downloaded %d bytes\n", n)
func CurlDownload(ctx context.Context, filepath string, command ...string) (int64, *http.Response, error) {
    resp, err := Curl(ctx, command...)
    if err != nil {
        return 0, nil, err
    }
    defer resp.Body.Close()

    file, err := os.Create(filepath)
    if err != nil {
        return 0, resp, fmt.Errorf("failed to create file: %w", err)
    }
    defer file.Close()

    n, err := io.Copy(file, resp.Body)
    if err != nil {
        return n, resp, fmt.Errorf("failed to write to file: %w", err)
    }

    return n, resp, nil
}

// CurlCommandDownload - Shell command variant
func CurlCommandDownload(ctx context.Context, filepath string, shellCommand string) (int64, *http.Response, error)

// CurlArgsDownload - Variadic variant
func CurlArgsDownload(ctx context.Context, filepath string, args ...string) (int64, *http.Response, error)
```

## Usage Examples

### Example 1: Simple Text Response

```go
// Option A: Full control
resp, err := gocurl.Curl(ctx, "https://example.com/api/status")
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
fmt.Println(string(body))

// Option B: Convenience
body, resp, err := gocurl.CurlString(ctx, "https://example.com/api/status")
fmt.Println(body)
```

### Example 2: JSON API

```go
// Option A: Manual decode
resp, err := gocurl.Curl(ctx,
    "-H", "Accept: application/json",
    "https://api.github.com/repos/owner/repo",
)
defer resp.Body.Close()
var repo Repository
json.NewDecoder(resp.Body).Decode(&repo)

// Option B: Auto-decode
var repo Repository
resp, err := gocurl.CurlJSON(ctx, &repo,
    "-H", "Accept: application/json",
    "https://api.github.com/repos/owner/repo",
)
```

### Example 3: Binary File Download

```go
// Option A: Manual stream
resp, err := gocurl.Curl(ctx, "https://example.com/image.png")
defer resp.Body.Close()
file, _ := os.Create("image.png")
io.Copy(file, resp.Body)

// Option B: Direct download
n, resp, err := gocurl.CurlDownload(ctx, "image.png",
    "https://example.com/image.png",
)
fmt.Printf("Downloaded %d bytes\n", n)

// Option C: To memory
data, resp, err := gocurl.CurlBytes(ctx, "https://example.com/image.png")
os.WriteFile("image.png", data, 0644)
```

### Example 4: Large File Streaming

```go
// Stream large log file line by line
resp, err := gocurl.CurlStream(ctx, "https://example.com/huge-log.txt")
defer resp.Body.Close()

scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
    line := scanner.Text()
    if strings.Contains(line, "ERROR") {
        log.Println(line)
    }
}
```

### Example 5: Multi-line Command with JSON

```go
var result APIResponse
resp, err := gocurl.CurlCommandJSON(ctx, &result, `
curl -X POST https://api.example.com/search \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer $TOKEN' \
  -d '{"query": "golang"}'
`)

if resp.StatusCode == 200 {
    fmt.Printf("Found %d results\n", len(result.Items))
}
```

## API Summary Table

| Function | Returns | Use When |
|----------|---------|----------|
| `Curl()` | `(*http.Response, error)` | Need full control over body handling |
| `CurlString()` | `(string, *http.Response, error)` | Text response, want full body as string |
| `CurlBytes()` | `([]byte, *http.Response, error)` | Binary response, need bytes in memory |
| `CurlJSON()` | `(*http.Response, error)` | JSON response, want auto-decode to struct |
| `CurlStream()` | `(*http.Response, error)` | Large response, need streaming |
| `CurlDownload()` | `(int64, *http.Response, error)` | File download, save directly to disk |

All functions have three variants:
- Base: Auto-detect syntax
- `Command`: Explicit shell command
- `Args`: Explicit variadic

## Backward Compatibility

If existing code expects string return:

```go
// Old API (deprecated)
func CurlDeprecated(ctx context.Context, command ...string) (*Response, string, error) {
    body, resp, err := CurlString(ctx, command...)
    return resp, body, err
}
```

## With Variable Maps

Same pattern for explicit variable maps:

```go
// Core
func CurlWithVars(ctx context.Context, vars map[string]string, command ...string) (*http.Response, error)

// Convenience
func CurlStringWithVars(ctx context.Context, vars map[string]string, command ...string) (string, *http.Response, error)
func CurlBytesWithVars(ctx context.Context, vars map[string]string, command ...string) ([]byte, *http.Response, error)
func CurlJSONWithVars(ctx context.Context, vars map[string]string, result interface{}, command ...string) (*http.Response, error)
func CurlDownloadWithVars(ctx context.Context, vars map[string]string, filepath string, command ...string) (int64, *http.Response, error)
```

## Benefits

### 1. Efficiency

```go
// ✅ Stream large file without loading into memory
resp, _ := gocurl.CurlStream(ctx, largeFileURL)
io.Copy(file, resp.Body)

// ❌ Old way: Loads entire file into memory
_, body, _ := oldCurl(ctx, largeFileURL)
os.WriteFile("file.bin", []byte(body), 0644)
```

### 2. Flexibility

```go
// User chooses how to handle response
resp, _ := gocurl.Curl(ctx, url)

// String
body, _ := io.ReadAll(resp.Body)
str := string(body)

// Stream
scanner := bufio.NewScanner(resp.Body)

// JSON
var data MyStruct
json.NewDecoder(resp.Body).Decode(&data)

// Direct to file
io.Copy(file, resp.Body)
```

### 3. Type Safety

```go
// JSON responses - compile-time type checking
var user User
resp, err := gocurl.CurlJSON(ctx, &user, userURL)
// user is properly typed, no manual unmarshaling
```

### 4. Performance

- No unnecessary allocations
- Streaming by default
- User controls buffering
- Direct file I/O when needed

## Recommendation

✅ **Implement all variants** for maximum flexibility
✅ **Make `Curl()` return `*http.Response`** (most flexible)
✅ **Provide convenience functions** for common cases
✅ **Document when to use each** (with examples)

This gives developers:
- **Control**: Full access to response stream
- **Convenience**: Helpers for common patterns
- **Efficiency**: No forced buffering
- **Flexibility**: Choose the right tool for the job

---

**Should I update the implementation plan with this response API design?**
