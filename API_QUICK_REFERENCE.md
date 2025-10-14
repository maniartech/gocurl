# GoCurl API Quick Reference

## Context-Aware HTTP Methods

All HTTP convenience methods now require `context.Context` as the first parameter for better control over request lifecycle.

### GET Request
```go
import "context"

ctx := context.Background()
resp, err := gocurl.Get(ctx, "https://api.example.com/users", nil)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()
```

### POST Request with JSON
```go
ctx := context.Background()

// Struct will be automatically marshaled to JSON
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

user := User{Name: "John Doe", Email: "john@example.com"}
resp, err := gocurl.Post(ctx, "https://api.example.com/users", user, nil)
```

### POST Request with String Body
```go
ctx := context.Background()
resp, err := gocurl.Post(ctx, "https://api.example.com/data",
    `{"key":"value"}`, nil)
```

### PUT Request
```go
ctx := context.Background()
data := map[string]interface{}{"name": "Updated Name"}
resp, err := gocurl.Put(ctx, "https://api.example.com/users/1", data, nil)
```

### DELETE Request
```go
ctx := context.Background()
resp, err := gocurl.Delete(ctx, "https://api.example.com/users/1", nil)
```

### PATCH Request
```go
ctx := context.Background()
patch := map[string]string{"status": "active"}
resp, err := gocurl.Patch(ctx, "https://api.example.com/users/1", patch, nil)
```

### HEAD Request
```go
ctx := context.Background()
resp, err := gocurl.Head(ctx, "https://api.example.com/users/1", nil)
```

## Context Features

### Request with Timeout
```go
import "time"

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := gocurl.Get(ctx, "https://api.example.com/slow-endpoint", nil)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Println("Request timed out")
    }
}
```

### Cancellable Request
```go
ctx, cancel := context.WithCancel(context.Background())

go func() {
    time.Sleep(2 * time.Second)
    cancel() // Cancel the request after 2 seconds
}()

resp, err := gocurl.Get(ctx, "https://api.example.com/long-running", nil)
if err != nil {
    if ctx.Err() == context.Canceled {
        log.Println("Request was cancelled")
    }
}
```

### Request with Values
```go
type contextKey string

const requestIDKey contextKey = "request-id"

ctx := context.WithValue(context.Background(), requestIDKey, "abc123")
resp, err := gocurl.Get(ctx, "https://api.example.com/data", nil)

// Middleware can access: ctx.Value(requestIDKey)
```

## Using RequestOptions Builder

### Basic Builder Pattern (Updated Naming)
```go
import "github.com/maniartech/gocurl/options"

opts := options.NewRequestOptionsBuilder().
    Get("https://api.example.com/users", http.Header{
        "Accept": []string{"application/json"},
    }).
    SetTimeout(30 * time.Second).
    Build()

resp, err := gocurl.Execute(opts)
```

### POST with Builder (Note: Use `Post()` not `POST()`)
```go
opts := options.NewRequestOptionsBuilder().
    Post("https://api.example.com/users",
        `{"name":"John"}`,
        http.Header{"Content-Type": []string{"application/json"}}).
    SetTimeout(10 * time.Second).
    Build()

resp, err := gocurl.Execute(opts)
```

### JSON Helper Method
```go
data := map[string]interface{}{
    "name": "John Doe",
    "age": 30,
}

opts := options.NewRequestOptionsBuilder().
    SetMethod("POST").
    SetURL("https://api.example.com/users").
    JSON(data). // Automatically marshals and sets Content-Type
    Build()

resp, err := gocurl.Execute(opts)
```

### With Context
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

opts := options.NewRequestOptionsBuilder().
    Get("https://api.example.com/users", nil).
    WithContext(ctx).
    Build()

resp, err := gocurl.Execute(opts)
```

### With Retry
```go
opts := options.NewRequestOptionsBuilder().
    Get("https://api.example.com/unstable", nil).
    WithRetry(3, 1*time.Second). // 3 retries, 1 second delay
    Build()

resp, err := gocurl.Execute(opts)
```

### With Default Retry
```go
opts := options.NewRequestOptionsBuilder().
    Get("https://api.example.com/data", nil).
    WithDefaultRetry(). // 3 retries, 1s delay, retries on 429, 5xx
    Build()

resp, err := gocurl.Execute(opts)
```

### Complete Example
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

opts := options.NewRequestOptionsBuilder().
    Post("https://api.example.com/data", "", nil).
    JSON(map[string]string{"key": "value"}).
    WithContext(ctx).
    WithRetry(3, time.Second).
    AddHeader("Authorization", "Bearer token123").
    SetUserAgent("MyApp/1.0").
    Build()

resp, err := gocurl.Execute(opts)
```

## Variable Substitution

Works with all methods:

```go
vars := gocurl.Variables{
    "API_URL": "https://api.example.com",
    "USER_ID": "123",
}

// Variables in URL
ctx := context.Background()
resp, err := gocurl.Get(ctx, "${API_URL}/users/${USER_ID}", vars)
```

## Traditional curl Command Style

Still supported:

```go
ctx := context.Background()
resp, err := gocurl.RequestWithContext(ctx,
    "curl -X POST https://api.example.com/data -H 'Content-Type: application/json'",
    nil)
```

Or use the legacy `Request()` (no context support):

```go
resp, err := gocurl.Request("curl https://api.example.com/data", nil)
```

## Mocking for Tests

```go
import "testing"

type MockHTTPClient struct{}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
    // Return mock response
    return &http.Response{
        StatusCode: 200,
        Header:     http.Header{"Content-Type": []string{"application/json"}},
        Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
    }, nil
}

func TestMyFunction(t *testing.T) {
    opts := options.NewRequestOptionsBuilder().
        Get("https://api.example.com/test", nil).
        SetHTTPClient(&MockHTTPClient()). // Inject mock
        Build()

    resp, err := gocurl.Execute(opts)
    // Assertions...
}
```

## Migration from Old API

| Old (Before) | New (After) |
|--------------|-------------|
| `gocurl.Request(cmd, vars)` | `gocurl.RequestWithContext(ctx, cmd, vars)` |
| No direct HTTP methods | `gocurl.Get(ctx, url, vars)` |
| `builder.POST(url, body, headers)` | `builder.Post(url, body, headers)` |
| `builder.GET(url, headers)` | `builder.Get(url, headers)` |
| `builder.PUT(url, body, headers)` | `builder.Put(url, body, headers)` |
| `builder.DELETE(url, headers)` | `builder.Delete(url, headers)` |
| `builder.PATCH(url, body, headers)` | `builder.Patch(url, body, headers)` |

## Key Changes

1. **All new HTTP methods require `context.Context`** - Better control over request lifecycle
2. **Builder methods use ProperCase naming** - Follows Go conventions (Post not POST)
3. **HTTPClient interface available** - Better testability with mock clients
4. **Automatic JSON marshaling** - POST/PUT/PATCH methods handle structs automatically

## Best Practices

1. **Always use context with timeout for external API calls:**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
   defer cancel()
   ```

2. **Prefer specific HTTP methods over generic Request:**
   ```go
   // Good
   resp, err := gocurl.Get(ctx, url, vars)

   // Less ideal
   resp, err := gocurl.Request("curl "+url, vars)
   ```

3. **Use builder for complex requests:**
   ```go
   opts := options.NewRequestOptionsBuilder().
       Post(url, "", nil).
       JSON(data).
       WithContext(ctx).
       WithRetry(3, time.Second).
       AddHeader("Authorization", "Bearer "+token).
       Build()
   ```

4. **Always defer cancel for contexts:**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel() // Important!
   ```

5. **Close response bodies:**
   ```go
   resp, err := gocurl.Get(ctx, url, nil)
   if err != nil {
       return err
   }
   defer resp.Body.Close() // Prevent resource leaks
   ```
