# Verbose Output Example

This document demonstrates the verbose output (`-v` flag) in gocurl, which matches curl's behavior.

## What Does curl -v Do?

When you use `curl -v`, it prints:
1. **Connection information** - DNS resolution, connection attempts
2. **TLS/SSL handshake** - Protocol negotiation, certificate verification
3. **Request headers** - All headers sent (prefixed with `>`)
4. **Response headers** - All headers received (prefixed with `<`)
5. **Connection status** - Whether connection is kept alive or closed

## gocurl Verbose Output

### Example 1: HTTP Request

```go
opts := options.NewRequestOptions("http://httpbin.org/get")
opts.Verbose = true

resp, body, err := gocurl.Process(context.Background(), opts)
```

**Output (to stderr):**
```
*   Trying 54.204.39.132:80...
* Connected to httpbin.org (54.204.39.132) port 80 (#0)
*
> GET /get HTTP/1.1
> Host: httpbin.org
> User-Agent: Go-http-client/1.1
> Accept-Encoding: gzip
>
< HTTP/1.1 200 OK
< Date: Tue, 14 Oct 2025 17:51:53 GMT
< Content-Type: application/json
< Content-Length: 314
< Connection: keep-alive
< Server: gunicorn/19.9.0
< Access-Control-Allow-Origin: *
< Access-Control-Allow-Credentials: true
<
* Connection #0 to host left intact
```

### Example 2: HTTPS Request

```go
opts := options.NewRequestOptions("https://api.github.com/users/octocat")
opts.Verbose = true

resp, body, err := gocurl.Process(context.Background(), opts)
```

**Output (to stderr):**
```
*   Trying 140.82.121.6:443...
* Connected to api.github.com (140.82.121.6) port 443 (#0)
* ALPN, offering h2
* ALPN, offering http/1.1
* TLS: Successfully set certificate verify locations
* Using TLS 1.2
*
> GET /users/octocat HTTP/1.1
> Host: api.github.com
> User-Agent: Go-http-client/1.1
> Accept-Encoding: gzip
>
* Using HTTP/2
< HTTP/2 200
< Server: GitHub.com
< Date: Tue, 14 Oct 2025 17:51:53 GMT
< Content-Type: application/json; charset=utf-8
< Cache-Control: public, max-age=60, s-maxage=60
< Etag: "abc123def456"
< X-Github-Request-Id: 1234:5678:9ABC:DEF0:123456
<
* Connection #0 to host left intact
```

### Example 3: Sensitive Headers Redacted

```go
opts := options.NewRequestOptions("https://api.example.com/protected")
opts.Verbose = true
opts.Headers = make(http.Header)
opts.Headers.Set("Authorization", "Bearer secret-token-12345")
opts.Headers.Set("Cookie", "session=abc123xyz789")

resp, body, err := gocurl.Process(context.Background(), opts)
```

**Output (to stderr):**
```
*   Trying 93.184.216.34:443...
* Connected to api.example.com (93.184.216.34) port 443 (#0)
* ALPN, offering h2
* ALPN, offering http/1.1
* TLS: Successfully set certificate verify locations
* Using TLS 1.2
*
> GET /protected HTTP/1.1
> Host: api.example.com
> Authorization: [REDACTED]
> Cookie: [REDACTED]
> User-Agent: Go-http-client/1.1
>
< HTTP/1.1 200 OK
< Content-Type: application/json
< Set-Cookie: [REDACTED]
<
* Connection #0 to host left intact
```

## Sensitive Headers

The following headers are automatically redacted in verbose output for security:
- `Authorization`
- `Cookie`
- `Set-Cookie`
- `X-Api-Key`
- `Api-Key`
- `X-Auth-Token`
- `Auth-Token`

## Line Prefixes

Following curl conventions:
- `*` - Connection/protocol information
- `>` - Request headers sent to server
- `<` - Response headers received from server

## Comparison with curl -v

| Feature | curl -v | gocurl Verbose | Status |
|---------|---------|----------------|--------|
| Connection info | ✅ | ✅ | Matching |
| TLS handshake details | ✅ | ✅ | Matching |
| Request headers | ✅ | ✅ | Matching |
| Response headers | ✅ | ✅ | Matching |
| HTTP/2 detection | ✅ | ✅ | Matching |
| Sensitive data redaction | ❌ | ✅ | Enhanced |
| Output to stderr | ✅ | ✅ | Matching |
| Custom writer support | ❌ | ✅ | Enhanced |

## Usage in Code

### Basic Usage
```go
opts := options.NewRequestOptions("https://example.com")
opts.Verbose = true

resp, body, err := gocurl.Process(context.Background(), opts)
// Verbose output goes to stderr automatically
```

### Custom Writer (for testing)
```go
var buf bytes.Buffer
gocurl.VerboseWriter = &buf

opts := options.NewRequestOptions("https://example.com")
opts.Verbose = true

resp, body, err := gocurl.Process(context.Background(), opts)

// Verbose output captured in buf
output := buf.String()
```

### Disable Verbose Output
```go
opts := options.NewRequestOptions("https://example.com")
opts.Verbose = false  // Default

resp, body, err := gocurl.Process(context.Background(), opts)
// No verbose output
```

## Notes

1. **Output Target**: Verbose output always goes to `stderr` (like curl), not `stdout`
2. **Silent Mode**: When `opts.Silent = true`, verbose warnings are still shown unless `Verbose = false`
3. **Thread-Safe**: Multiple concurrent requests with verbose output are thread-safe
4. **Performance**: Minimal overhead when `Verbose = false` (just a boolean check)

## Benefits Over curl -v

1. **Automatic Redaction**: Sensitive headers are automatically hidden
2. **Custom Writers**: Can redirect output for testing
3. **Structured**: Output is programmatically generated, ensuring consistency
4. **Integration**: Works seamlessly with Go error handling and context
