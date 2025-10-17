# Exercise 1: Builder Basics

**Difficulty:** Beginner
**Duration:** 30-40 minutes
**Prerequisites:** Chapter 5 main content, examples 01-04

## Objective

Master the fundamentals of the RequestOptionsBuilder by implementing various HTTP requests using the fluent API. Learn to configure methods, URLs, headers, body, and query parameters.

## Tasks

### Task 1: Simple GET Request

Build a GET request to `https://api.github.com/zen` using the Builder pattern.

**Requirements:**
- Use `NewRequestOptionsBuilder()`
- Set URL and Method
- Build and execute
- Print the response

**Expected Output:**
```
Status: 200
Response: [GitHub Zen quote]
```

<details>
<summary>💡 Hint</summary>

```go
builder := options.NewRequestOptionsBuilder()
opts := builder.
    SetURL("https://api.github.com/zen").
    SetMethod("GET").
    Build()
```
</details>

---

### Task 2: POST with Headers

Create a POST request to `https://httpbin.org/post` with:
- JSON body: `{"message": "Hello from Builder"}`
- Content-Type header: `application/json`
- Custom header: `X-Custom-Header: MyValue`

**Requirements:**
- Use `SetMethod("POST")`
- Add multiple headers with `AddHeader()`
- Set body with `SetBody()`
- Execute and verify headers appear in response

**Expected Output:**
```
Status: 200
Response includes your headers and JSON body
```

<details>
<summary>💡 Hint</summary>

```go
opts := options.NewRequestOptionsBuilder().
    SetURL("https://httpbin.org/post").
    SetMethod("POST").
    AddHeader("Content-Type", "application/json").
    AddHeader("X-Custom-Header", "MyValue").
    SetBody(`{"message": "Hello from Builder"}`).
    Build()
```
</details>

---

### Task 3: Query Parameters

Build a GET request to `https://httpbin.org/get` with query parameters:
- `name=Alice`
- `age=25`
- `city=NewYork`

**Requirements:**
- Use `AddQueryParam()` for each parameter
- Verify parameters appear in response `args` field

**Expected Output:**
```
Status: 200
Query params in response: name=Alice, age=25, city=NewYork
```

<details>
<summary>💡 Hint</summary>

```go
opts := options.NewRequestOptionsBuilder().
    SetURL("https://httpbin.org/get").
    AddQueryParam("name", "Alice").
    AddQueryParam("age", "25").
    AddQueryParam("city", "NewYork").
    Build()
```
</details>

---

### Task 4: HTTP Method Shortcuts

Use the `Post()` shortcut method to create a POST request to `https://httpbin.org/post`:
- Body: `{"status": "testing shortcuts"}`
- Headers: Content-Type as `application/json`

**Requirements:**
- Use `Post(url, body, headers)` method
- Don't use `SetMethod()` or `SetURL()` separately

**Expected Output:**
```
Status: 200
Response shows your body and headers
```

<details>
<summary>💡 Hint</summary>

```go
headers := http.Header{}
headers.Add("Content-Type", "application/json")

opts := options.NewRequestOptionsBuilder().
    Post("https://httpbin.org/post",
         `{"status": "testing shortcuts"}`,
         headers).
    Build()
```
</details>

---

### Task 5: Multiple Configurations

Create a single builder and use it to make three different requests:
1. GET to `https://httpbin.org/get`
2. POST to `https://httpbin.org/post` with body `{"test": "data"}`
3. DELETE to `https://httpbin.org/delete`

**Requirements:**
- Reuse the same builder instance
- Change configuration between builds
- Execute all three requests
- Print status codes

**Expected Output:**
```
GET: 200
POST: 200
DELETE: 200
```

<details>
<summary>💡 Hint</summary>

```go
builder := options.NewRequestOptionsBuilder()

// First request
opts1 := builder.SetURL("https://httpbin.org/get").SetMethod("GET").Build()
// Execute opts1

// Second request (reuse builder)
opts2 := builder.SetURL("https://httpbin.org/post").SetMethod("POST").
    SetBody(`{"test": "data"}`).Build()
// Execute opts2

// Third request
opts3 := builder.SetURL("https://httpbin.org/delete").SetMethod("DELETE").Build()
// Execute opts3
```
</details>

---

### Task 6: Complete Request Builder

Build a comprehensive POST request with all these features:
- URL: `https://httpbin.org/post`
- Method: POST
- Headers:
  - `Content-Type: application/json`
  - `Authorization: Bearer test-token-12345`
  - `User-Agent: MyApp/1.0`
- Body: `{"user": "alice", "action": "login"}`
- Timeout: 30 seconds

**Requirements:**
- Use fluent API (method chaining)
- Set all configurations
- Execute and verify all settings in response

**Expected Output:**
```
Status: 200
All headers present in response
Body: {"user": "alice", "action": "login"}
Request completed within 30s
```

<details>
<summary>💡 Hint</summary>

```go
opts := options.NewRequestOptionsBuilder().
    SetURL("https://httpbin.org/post").
    SetMethod("POST").
    AddHeader("Content-Type", "application/json").
    AddHeader("Authorization", "Bearer test-token-12345").
    AddHeader("User-Agent", "MyApp/1.0").
    SetBody(`{"user": "alice", "action": "login"}`).
    SetTimeout(30 * time.Second).
    Build()
```
</details>

---

## Validation Script

Create `exercise1_test.go` to validate your solutions:

```go
package main

import (
    "context"
    "net/http"
    "testing"
    "time"

    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/options"
)

func TestTask1_SimpleGET(t *testing.T) {
    opts := options.NewRequestOptionsBuilder().
        SetURL("https://api.github.com/zen").
        SetMethod("GET").
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

func TestTask2_POSTWithHeaders(t *testing.T) {
    opts := options.NewRequestOptionsBuilder().
        SetURL("https://httpbin.org/post").
        SetMethod("POST").
        AddHeader("Content-Type", "application/json").
        AddHeader("X-Custom-Header", "MyValue").
        SetBody(`{"message": "Hello from Builder"}`).
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

func TestTask3_QueryParameters(t *testing.T) {
    opts := options.NewRequestOptionsBuilder().
        SetURL("https://httpbin.org/get").
        AddQueryParam("name", "Alice").
        AddQueryParam("age", "25").
        AddQueryParam("city", "NewYork").
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

func TestTask4_HTTPShortcuts(t *testing.T) {
    headers := http.Header{}
    headers.Add("Content-Type", "application/json")

    opts := options.NewRequestOptionsBuilder().
        Post("https://httpbin.org/post",
            `{"status": "testing shortcuts"}`,
            headers).
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

func TestTask6_CompleteRequest(t *testing.T) {
    opts := options.NewRequestOptionsBuilder().
        SetURL("https://httpbin.org/post").
        SetMethod("POST").
        AddHeader("Content-Type", "application/json").
        AddHeader("Authorization", "Bearer test-token-12345").
        AddHeader("User-Agent", "MyApp/1.0").
        SetBody(`{"user": "alice", "action": "login"}`).
        SetTimeout(30 * time.Second).
        Build()

    ctx := context.Background()
    start := time.Now()
    resp, err := gocurl.Execute(ctx, opts)
    duration := time.Since(start)

    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }

    if duration > 30*time.Second {
        t.Errorf("Request took too long: %v", duration)
    }
}
```

Run tests: `go test -v`

---

## Self-Check Criteria

- ✅ All 6 tasks completed successfully
- ✅ Proper use of Builder pattern methods
- ✅ Method chaining works correctly
- ✅ All requests return expected status codes
- ✅ Headers and query parameters appear in responses
- ✅ Code is clean and readable

## Common Mistakes to Avoid

1. **Forgetting to call Build()**
   ```go
   // ❌ Wrong
   opts := builder.SetURL("...").SetMethod("GET")

   // ✅ Correct
   opts := builder.SetURL("...").SetMethod("GET").Build()
   ```

2. **Not handling errors**
   ```go
   // ❌ Wrong
   resp, _ := gocurl.Execute(ctx, opts)

   // ✅ Correct
   resp, err := gocurl.Execute(ctx, opts)
   if err != nil {
       return err
   }
   ```

3. **Not closing response body**
   ```go
   // ✅ Always defer Close()
   defer resp.Body.Close()
   ```

## Bonus Challenges

1. **Form Data:** Create a POST request with form data instead of JSON
2. **User Agent:** Set a custom User-Agent header
3. **Timeout Test:** Create a request with 1-second timeout to `https://httpbin.org/delay/5` and verify it times out
4. **Builder Reuse:** Create a base builder with common headers, then build 5 different requests from it

## Learning Outcomes

After completing this exercise, you should be able to:
- ✅ Create RequestOptionsBuilder instances
- ✅ Use fluent API for configuration
- ✅ Set methods, URLs, headers, and body
- ✅ Add query parameters
- ✅ Use HTTP method shortcuts
- ✅ Chain multiple configuration calls
- ✅ Build and execute requests
- ✅ Reuse builders for multiple requests

## Next Steps

Once you've completed all tasks and tests pass:
1. Review your code for improvements
2. Try the bonus challenges
3. Proceed to Exercise 2 (Advanced Configuration)
