# GoCurl - Curl-Compatible HTTP Client for Go

[![Go Reference](https://pkg.go.dev/badge/github.com/maniartech/gocurl.svg)](https://pkg.go.dev/github.com/maniartech/gocurl)
[![Go Report Card](https://goreportcard.com/badge/github.com/maniartech/gocurl)](https://goreportcard.com/report/github.com/maniartech/gocurl)

**"Test with CLI first" - This is the ABSOLUTE CORE objective of this library.**

GoCurl is a zero-allocation, high-performance HTTP/HTTP2 client library for Go that revolutionizes API integration by supporting curl-compatible syntax. Copy curl commands from API documentation, test them with the CLI, and use them directly in your Go code - no translation needed.

## The Workflow: CLI → Code

```bash
# Step 1: Test with CLI (instant feedback)
$ export API_TOKEN=sk_live_123456
$ gocurl -X POST \
    -H "Authorization: Bearer $API_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"query": "test"}' \
    https://api.example.com/search

# ✅ Works! Returns: {"results": [...]}
```

```go
// Step 2: Use EXACT same command in Go code
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/maniartech/gocurl"
)

func main() {
    ctx := context.Background()
    os.Setenv("API_TOKEN", "sk_live_123456")

    // Full control with *http.Response
    resp, err := gocurl.Curl(ctx,
        "-X", "POST",
        "-H", "Authorization: Bearer $API_TOKEN",
        "-H", "Content-Type: application/json",
        "-d", `{"query": "test"}`,
        "https://api.example.com/search",
    )
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    // Access response details
    fmt.Println("Status:", resp.StatusCode)
    fmt.Println("Headers:", resp.Header)

    // ✅ Identical behavior to CLI - zero translation!
}
```

## Installation

### CLI Tool

```bash
go install github.com/maniartech/gocurl/cmd/gocurl@latest
```

### Library

```bash
go get github.com/maniartech/gocurl
```

## Key Features

✅ **CLI-to-Code Workflow**: Test with CLI → copy exact command → use in Go
✅ **Zero Translation**: Same syntax, same behavior, everywhere
✅ **Zero Allocation**: Military-grade performance on critical path
✅ **Multi-line Support**: Backslashes, comments, browser DevTools format
✅ **Environment Variables**: `$VAR` and `${VAR}` auto-expanded
✅ **Returns `*http.Response`**: Full access to headers, status, cookies
✅ **Convenience Functions**: Auto-read body, decode JSON, download to file
✅ **Curl Parity**: Every feature matches real curl behavior

## Quick Start

### Three Core Functions

```go
// 1. Curl() - Auto-detect (use this 90% of the time)
resp, err := gocurl.Curl(ctx, "-H", "X-Token: abc", "https://example.com")
resp, err := gocurl.Curl(ctx, `curl -H 'X-Token: abc' https://example.com`)

// 2. CurlCommand() - Explicit shell parsing
resp, err := gocurl.CurlCommand(ctx, `curl -H 'X-Token: abc' https://example.com`)

// 3. CurlArgs() - Explicit variadic
resp, err := gocurl.CurlArgs(ctx, "-H", "X-Token: abc", "https://example.com")
```

### Convenience Functions

```go
// CurlString - Auto-read body as string
body, resp, err := gocurl.CurlString(ctx, "https://api.example.com/data")

// CurlJSON - Auto-decode JSON
var data MyStruct
resp, err := gocurl.CurlJSON(ctx, &data, "https://api.example.com/data")

// CurlDownload - Save to file
bytesWritten, resp, err := gocurl.CurlDownload(ctx, "output.json", "https://api.example.com/data")

// CurlBytes - Get raw bytes
bytes, resp, err := gocurl.CurlBytes(ctx, "https://api.example.com/data")
```

## Real-World Examples

### Example 1: Copy from Browser DevTools

```bash
# In browser: Network tab → Right-click → Copy as cURL
curl 'https://api.example.com/search' \
  -X POST \
  -H 'Authorization: Bearer sk_live_123456' \
  -H 'Content-Type: application/json' \
  --data-raw '{"query": "test"}'
```

```go
// Just paste into Go (remove 'curl' prefix or keep it - works either way!)
resp, err := gocurl.Curl(ctx, `
    'https://api.example.com/search'
    -X POST
    -H 'Authorization: Bearer sk_live_123456'
    -H 'Content-Type: application/json'
    --data-raw '{"query": "test"}'
`)
defer resp.Body.Close()
```

### Example 2: From API Documentation

```bash
# Stripe API example
curl -X POST https://api.stripe.com/v1/charges \
  -u sk_test_xyz: \
  -d amount=2000 \
  -d currency=usd
```

```go
// Direct copy - works in Go!
resp, err := gocurl.Curl(ctx, `
    -X POST https://api.stripe.com/v1/charges
    -u sk_test_xyz:
    -d amount=2000
    -d currency=usd
`)

// Or use convenience function to decode JSON
var charge StripeCharge
resp, err := gocurl.CurlJSON(ctx, &charge, `
    -X POST https://api.stripe.com/v1/charges
    -u sk_test_xyz:
    -d amount=2000
    -d currency=usd
`)
fmt.Printf("Charge ID: %s\n", charge.ID)
```

### Example 3: Environment Variables

```bash
# Test with CLI first
$ export GITHUB_TOKEN=ghp_xyz123
$ gocurl -H "Authorization: Bearer $GITHUB_TOKEN" \
         https://api.github.com/user
```

```go
// Use exact same command in Go
os.Setenv("GITHUB_TOKEN", "ghp_xyz123")

var user GitHubUser
resp, err := gocurl.CurlJSON(ctx, &user,
    "-H", "Authorization: Bearer $GITHUB_TOKEN",
    "https://api.github.com/user",
)

fmt.Printf("User: %s\n", user.Login)
```

### Example 4: Multi-line Commands

```go
// All these formats work identically:

// With backslash continuations (API docs style)
resp, err := gocurl.Curl(ctx, `
    curl -X POST https://api.example.com \
      -H "Content-Type: application/json" \
      -d '{"key":"value"}'
`)

// Without backslashes (gocurl handles newlines)
resp, err := gocurl.Curl(ctx, `
    curl -X POST https://api.example.com
      -H "Content-Type: application/json"
      -d '{"key":"value"}'
`)

// With comments
resp, err := gocurl.Curl(ctx, `
    # API endpoint
    https://api.example.com
    # Headers
    -H "Content-Type: application/json"
    # Body
    -d '{"key":"value"}'
`)
```

## CLI Usage

### Basic Commands

```bash
# Simple GET
gocurl https://api.github.com/zen

# POST with data
gocurl -X POST -d "name=John" https://httpbin.org/post

# Custom headers
gocurl -H "Authorization: Bearer $TOKEN" https://api.example.com/data

# Verbose output
gocurl -v https://api.github.com/zen

# Save to file
gocurl -o response.json https://api.github.com/data
```

### CLI Options

```
GoCurl Options:
  -v, --verbose        Verbose output (show headers and request)
  -i, --include        Include response headers in output
  -s, --silent         Silent mode (no output)
  -o, --output FILE    Write output to file
  -w, --write-out FMT  Custom output format

Curl Options: (All standard curl options supported)
  -X, --request        HTTP method (GET, POST, PUT, DELETE, etc.)
  -H, --header         Add header
  -d, --data           Request body data
  -u, --user           Basic auth credentials
  -b, --cookie         Cookies
  -L, --location       Follow redirects
  -k, --insecure       Skip SSL verification
  --proxy              Use proxy
  And many more...
```

## Advanced Features

### Variable Control

```go
// Auto-expand from environment (default)
os.Setenv("API_KEY", "secret")
resp, err := gocurl.Curl(ctx, "-H", "X-API-Key: $API_KEY", "https://api.example.com")

// Explicit variable map (for testing/security)
vars := gocurl.Variables{"API_KEY": "secret"}
resp, err := gocurl.CurlWithVars(ctx, vars, "-H", "X-API-Key: $API_KEY", "https://api.example.com")
```

### Full Response Access

```go
resp, err := gocurl.Curl(ctx, "https://api.example.com")
defer resp.Body.Close()

// Access all response details
fmt.Println("Status:", resp.StatusCode)
fmt.Println("Headers:", resp.Header)
fmt.Println("Cookies:", resp.Cookies())
fmt.Println("Content-Length:", resp.ContentLength)

// Read body when needed
body, _ := io.ReadAll(resp.Body)
```

### Supported Curl Options

- **HTTP Methods**: `-X GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS`
- **Headers**: `-H "Key: Value"`
- **Data**: `-d "data"`, `--data`, `--data-raw`, `--data-binary`
- **Forms**: `-F "field=value"`, `--form`
- **Auth**: `-u user:pass`, Basic authentication
- **Cookies**: `-b cookie`, `-c cookiejar`
- **Redirects**: `-L`, `--location`, `--max-redirs N`
- **Proxy**: `-x proxy`, `--proxy`
- **TLS**: `--cert`, `--key`, `--cacert`, `-k`/`--insecure`
- **HTTP/2**: `--http2`, `--http2-only`
- **Output**: `-o file`, `--output`
- **Compression**: `--compressed`
- **User Agent**: `-A "agent"`, `--user-agent`
- **Timeout**: `--max-time N`
- **Verbose**: `-v`, `--verbose`
- **Silent**: `-s`, `--silent`

## Testing

### Parity Tests

GoCurl includes comprehensive parity tests that compare output with real curl:

```bash
go test -v -run TestCoreParity
go test -v -run TestMultilineParity
go test -v -run TestEnvironmentVariablesParity
```

### Your Own Tests

```go
func TestMyAPI(t *testing.T) {
    ctx := context.Background()

    var data APIResponse
    resp, err := gocurl.CurlJSON(ctx, &data,
        "-H", "Authorization: Bearer test",
        "https://api.example.com/endpoint",
    )

    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
    assert.Equal(t, "expected", data.Field)
}
```

## Performance

GoCurl is designed for **zero-allocation** performance on the critical path:

- Sub-microsecond request construction overhead
- Memory pooling for frequently-used objects
- Efficient connection reuse
- HTTP/2 multiplexing support

```bash
# Run benchmarks
go test -bench=. -benchmem
```

## Comparison with Other Libraries

| Feature | GoCurl | net/http | resty | req |
|---------|--------|----------|-------|-----|
| Curl syntax | ✅ | ❌ | ❌ | ❌ |
| CLI tool | ✅ | ❌ | ❌ | ❌ |
| CLI-to-code workflow | ✅ | ❌ | ❌ | ❌ |
| Zero allocation | ✅ | ⚠️ | ❌ | ❌ |
| Multi-line commands | ✅ | ❌ | ❌ | ❌ |
| Environment variables | ✅ | ❌ | ❌ | ❌ |
| Returns `*http.Response` | ✅ | ✅ | ⚠️ | ⚠️ |
| Curl parity tests | ✅ | ❌ | ❌ | ❌ |

## Why GoCurl?

### Problem: API Integration is Tedious

1. See curl command in API docs
2. Manually translate to Go code
3. Debug differences between curl and your code
4. Repeat for every endpoint

### Solution: Copy-Paste from CLI to Code

1. **Test with CLI**: `gocurl [curl-command]` - instant feedback
2. **Copy to Go code**: Exact same command - zero translation
3. **Same behavior**: Guaranteed by curl parity tests

### Real-World Impact

**Before GoCurl:**
```go
// Manual translation from curl command
req, _ := http.NewRequest("POST", "https://api.stripe.com/v1/charges",
    strings.NewReader("amount=2000&currency=usd"))
req.SetBasicAuth("sk_test_xyz", "")
req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
client := &http.Client{}
resp, _ := client.Do(req)
// Did I translate correctly? Who knows...
```

**With GoCurl:**
```go
// Direct copy from Stripe docs - guaranteed to work
resp, err := gocurl.Curl(ctx, `
    -X POST https://api.stripe.com/v1/charges
    -u sk_test_xyz:
    -d amount=2000
    -d currency=usd
`)
// ✅ Identical to curl - tested with CLI first!
```

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) and [Code of Conduct](CODE_OF_CONDUCT.md).

### Development

```bash
# Clone repository
git clone https://github.com/maniartech/gocurl.git
cd gocurl

# Run tests
go test -v ./...

# Run parity tests
go test -v -run Parity

# Build CLI
go build -o gocurl ./cmd/gocurl

# Run benchmarks
go test -bench=. -benchmem
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Credits

Created by [ManiarTech](https://maniartech.com) - Building tools that developers love.

## Roadmap

- [ ] Complete curl flag compatibility
- [ ] GraphQL support
- [ ] Request/response middleware
- [ ] Distributed tracing integration
- [ ] Performance dashboard
- [ ] Browser extension for one-click code generation

## Support

- 📖 [Documentation](https://pkg.go.dev/github.com/maniartech/gocurl)
- 💬 [Discussions](https://github.com/maniartech/gocurl/discussions)
- 🐛 [Issue Tracker](https://github.com/maniartech/gocurl/issues)
- 📧 Email: support@maniartech.com

---

**"Test with CLI first"** - Made with ❤️ by ManiarTech
