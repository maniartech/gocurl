# User-Agent Behavior - Curl Compatibility

## Overview

gocurl follows curl's behavior by **always sending a User-Agent header** in every request, matching curl's standard practice.

## Default User-Agent

### Format

```
gocurl/<VERSION>
```

This mirrors curl's format: `curl/<VERSION>`

### Examples

**curl:**
```bash
$ curl https://httpbin.org/user-agent
{
  "user-agent": "curl/7.85.0"
}
```

**gocurl (default):**
```bash
$ gocurl https://httpbin.org/user-agent
{
  "user-agent": "gocurl/dev"
}
```

## Version Information

The version is set via build-time variables:

```bash
go build -ldflags "-X github.com/maniartech/gocurl.Version=1.0.0"
```

If not set during build, it defaults to `"dev"`.

## Customizing User-Agent

### 1. Builder Pattern (Recommended)

```go
opts := options.NewRequestOptionsBuilder().
    SetURL("https://api.example.com").
    SetUserAgent("MyApp/1.0").
    Build()

resp, _, err := gocurl.Process(ctx, opts)
```

### 2. CLI Usage

```bash
gocurl -A "MyApp/1.0" https://api.example.com
gocurl --user-agent "MyApp/1.0" https://api.example.com
```

### 3. Curl-Style

```go
resp, err := gocurl.Curl(ctx, "-A", "MyApp/1.0", "https://api.example.com")
```

## Implementation Details

### Location

- **Default behavior**: `process.go` in `applyHeaders()` function
- **Version variable**: `version.go`
- **Builder method**: `options/builder.go` - `SetUserAgent()`
- **Field definition**: `options/options.go` - `UserAgent string`

### Code

```go
// Set user agent (curl always sends a User-Agent header)
if opts.UserAgent != "" {
    req.Header.Set("User-Agent", opts.UserAgent)
} else {
    // Default to "gocurl/VERSION" to match curl's behavior (curl/VERSION)
    req.Header.Set("User-Agent", "gocurl/"+Version)
}
```

## Why This Matters

1. **Curl Compatibility**: Matches curl's well-established behavior
2. **Server Identification**: Servers can identify and log the client type
3. **API Analytics**: Many APIs track User-Agent for usage statistics
4. **Rate Limiting**: Some APIs use User-Agent for rate limiting policies
5. **Best Practice**: HTTP best practice to always send User-Agent

## Testing

### Test Coverage

The implementation includes comprehensive tests:

```go
// Test 1: Default User-Agent follows curl behavior
func TestCustomUserAgent/Default_User-Agent_follows_curl_behavior

// Test 2: Custom User-Agent can be set
func TestCustomUserAgent/Custom_User-Agent_string
```

### Running Tests

```bash
go test -run=TestCustomUserAgent -v
```

## Comparison with Other Libraries

| Library | Default User-Agent | Format |
|---------|-------------------|--------|
| curl | ✅ Yes | `curl/7.85.0` |
| gocurl | ✅ Yes | `gocurl/dev` |
| net/http | ❌ No default | `Go-http-client/1.1` (when header set) |
| requests (Python) | ✅ Yes | `python-requests/2.28.0` |
| axios (Node.js) | ❌ No default | Uses Node.js default |

## Migration Notes

If you were relying on the previous behavior (no default User-Agent), you can:

1. **Option A**: Accept the new curl-compatible behavior (recommended)
2. **Option B**: Explicitly set an empty User-Agent (though not recommended):
   ```go
   // Not recommended - violates HTTP best practices
   req.Header.Del("User-Agent")
   ```

## See Also

- [curl User-Agent documentation](https://curl.se/docs/manpage.html#-A)
- [HTTP User-Agent header spec](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/User-Agent)
- Builder Pattern examples in `book2/part2-api-approaches/chapter05-builder-pattern/`
