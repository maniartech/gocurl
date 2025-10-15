# Middleware vs ResponseDecoder - Industry Best Practices

## Date: October 14, 2025

## Overview

GoCurl provides **two distinct patterns** for extending request/response handling:
1. **Middleware** - Request transformation and pre-processing
2. **ResponseDecoder** - Response parsing and unmarshaling

This document clarifies the industry-standard separation of concerns and when to use each pattern.

---

## The Confusion

Both `Middleware` and `ResponseDecoder` can technically access the response, leading to confusion:

```go
// Current implementation
type MiddlewareFunc func(*http.Request) (*http.Request, error)  // Request-only
type ResponseDecoder func(*http.Response) (interface{}, error)   // Response-only
```

**Questions:**
- Why have both if they seem similar?
- When should I use middleware vs decoder?
- Can middleware handle responses too?

---

## Industry Standard: Separation of Concerns

### 📊 Industry Analysis

**Libraries Surveyed:**
- **resty** (Go): Request middleware + Response interceptors (separate)
- **axios** (JavaScript): Request interceptors + Response interceptors
- **OkHttp** (Java/Kotlin): Interceptor chain (request & response)
- **Retrofit** (Java): Converters (response) + Interceptors (both)
- **requests** (Python): Hooks for request/response
- **Go net/http**: RoundTripper (transport layer)

**Common Pattern:** ✅ **Separate request processing from response decoding**

---

## Pattern 1: Middleware (Request Processing)

### Purpose
**Transform, validate, or enhance HTTP requests BEFORE they are sent**

### Signature
```go
type MiddlewareFunc func(*http.Request) (*http.Request, error)
```

### Industry Standard Use Cases

#### ✅ 1. Authentication & Authorization
```go
func BearerTokenMiddleware(token string) middlewares.MiddlewareFunc {
    return func(req *http.Request) (*http.Request, error) {
        req.Header.Set("Authorization", "Bearer "+token)
        return req, nil
    }
}

opts.Middleware = []middlewares.MiddlewareFunc{
    BearerTokenMiddleware(os.Getenv("API_TOKEN")),
}
```

#### ✅ 2. Request Logging & Tracing
```go
func RequestLoggingMiddleware(logger *log.Logger) middlewares.MiddlewareFunc {
    return func(req *http.Request) (*http.Request, error) {
        logger.Printf("→ %s %s", req.Method, req.URL)

        // Add trace ID
        traceID := uuid.New().String()
        req.Header.Set("X-Trace-ID", traceID)

        return req, nil
    }
}
```

#### ✅ 3. Request Validation
```go
func ValidateRequestMiddleware() middlewares.MiddlewareFunc {
    return func(req *http.Request) (*http.Request, error) {
        if req.Header.Get("Content-Type") == "" {
            return nil, fmt.Errorf("Content-Type header required")
        }
        return req, nil
    }
}
```

#### ✅ 4. Custom Headers (API Keys, User-Agent, etc.)
```go
func APIKeyMiddleware(apiKey string) middlewares.MiddlewareFunc {
    return func(req *http.Request) (*http.Request, error) {
        req.Header.Set("X-API-Key", apiKey)
        req.Header.Set("User-Agent", "MyApp/1.0")
        return req, nil
    }
}
```

#### ✅ 5. Request Signing (AWS, OAuth)
```go
func AWSSignatureMiddleware(credentials *AWSCredentials) middlewares.MiddlewareFunc {
    return func(req *http.Request) (*http.Request, error) {
        // Sign request with AWS Signature V4
        signature := signRequest(req, credentials)
        req.Header.Set("Authorization", signature)
        return req, nil
    }
}
```

### Characteristics
- ✅ Runs **before** request is sent
- ✅ Can modify headers, URL, body
- ✅ Can abort request (return error)
- ✅ Chainable (multiple middleware execute in order)
- ✅ No access to response (request-focused)

---

## Pattern 2: ResponseDecoder (Response Parsing)

### Purpose
**Parse and unmarshal response bodies into structured data**

### Signature
```go
type ResponseDecoder func(*http.Response) (interface{}, error)
```

### Industry Standard Use Cases

#### ✅ 1. Custom Format Unmarshaling (XML, YAML, Protobuf)
```go
// XML Decoder
func XMLResponseDecoder(resp *http.Response) (interface{}, error) {
    defer resp.Body.Close()

    var result MyXMLStruct
    decoder := xml.NewDecoder(resp.Body)
    if err := decoder.Decode(&result); err != nil {
        return nil, fmt.Errorf("XML decode failed: %w", err)
    }

    return result, nil
}

opts.ResponseDecoder = XMLResponseDecoder
```

#### ✅ 2. Protocol Buffer Decoding
```go
func ProtobufDecoder(resp *http.Response) (interface{}, error) {
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var message pb.MyMessage
    if err := proto.Unmarshal(body, &message); err != nil {
        return nil, fmt.Errorf("protobuf decode failed: %w", err)
    }

    return &message, nil
}
```

#### ✅ 3. Custom JSON Processing
```go
// JSON with envelope unwrapping
func EnvelopedJSONDecoder(resp *http.Response) (interface{}, error) {
    defer resp.Body.Close()

    var envelope struct {
        Success bool            `json:"success"`
        Data    json.RawMessage `json:"data"`
        Error   string          `json:"error,omitempty"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
        return nil, err
    }

    if !envelope.Success {
        return nil, fmt.Errorf("API error: %s", envelope.Error)
    }

    return envelope.Data, nil
}
```

#### ✅ 4. YAML Decoding
```go
func YAMLDecoder(resp *http.Response) (interface{}, error) {
    defer resp.Body.Close()

    var result map[string]interface{}
    decoder := yaml.NewDecoder(resp.Body)
    if err := decoder.Decode(&result); err != nil {
        return nil, fmt.Errorf("YAML decode failed: %w", err)
    }

    return result, nil
}
```

#### ✅ 5. MessagePack Decoding
```go
func MessagePackDecoder(resp *http.Response) (interface{}, error) {
    defer resp.Body.Close()

    var result interface{}
    decoder := msgpack.NewDecoder(resp.Body)
    if err := decoder.Decode(&result); err != nil {
        return nil, fmt.Errorf("msgpack decode failed: %w", err)
    }

    return result, nil
}
```

### Characteristics
- ✅ Runs **after** response is received
- ✅ Focused on **data transformation** (bytes → structured data)
- ✅ Returns `interface{}` (flexible but requires type assertion)
- ✅ Not chainable (single decoder per request)
- ✅ No access to request (response-focused)

---

## Industry Comparison

### Go Libraries

#### resty (Most Popular Go HTTP Client)
```go
// Request middleware
client.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
    req.SetHeader("X-Custom", "value")
    return nil
})

// Response middleware (separate!)
client.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
    log.Println("Response:", resp.Status())
    return nil
})

// Custom unmarshaler (like our ResponseDecoder)
client.SetJSONUnmarshaler(customJSONUnmarshaler)
```

**Pattern:** ✅ Separate request/response hooks + custom unmarshaling

#### gentleman (Middleware-focused)
```go
// Request middleware
cli.Use(plugin.Headers(map[string]string{
    "User-Agent": "gentleman",
}))

// Response plugin (separate)
cli.Use(plugin.BodyType("json"))
```

**Pattern:** ✅ Separate plugins for request/response

### JavaScript/TypeScript

#### axios (Industry Standard)
```js
// Request interceptor
axios.interceptors.request.use(config => {
    config.headers.Authorization = 'Bearer ' + token;
    return config;
});

// Response interceptor (separate!)
axios.interceptors.response.use(response => {
    // Transform response data
    return response.data;
});

// Custom transformer (like our ResponseDecoder)
axios.get(url, {
    transformResponse: [(data) => {
        return JSON.parse(data);
    }]
});
```

**Pattern:** ✅ Separate request/response interceptors + transformers

### Java/Kotlin

#### Retrofit (Type-safe HTTP client)
```java
// Interceptor (can access request AND response)
OkHttpClient client = new OkHttpClient.Builder()
    .addInterceptor(chain -> {
        Request request = chain.request()
            .newBuilder()
            .addHeader("Authorization", "Bearer " + token)
            .build();
        return chain.proceed(request);
    })
    .build();

// Converter (like our ResponseDecoder)
Retrofit retrofit = new Retrofit.Builder()
    .addConverterFactory(GsonConverterFactory.create())
    .build();
```

**Pattern:** ✅ Interceptor chain + separate converter factory

---

## Why Keep Both Patterns?

### ✅ Separation of Concerns (SOLID Principles)

**Single Responsibility:**
- **Middleware**: Handles request preparation (auth, logging, validation)
- **ResponseDecoder**: Handles data transformation (unmarshaling)

**Open/Closed:**
- Extend functionality without modifying core library
- Add custom formats (Protobuf, YAML) without changing gocurl

**Dependency Inversion:**
- Both depend on abstractions (function types)
- Users can inject custom implementations

### ✅ Performance Optimization

**Middleware:**
- Lightweight (header manipulation)
- Runs sequentially (minimal overhead)
- No allocation for decoding

**ResponseDecoder:**
- Heavy (parsing, unmarshaling)
- Only runs if needed
- Can use pooling for parsers

### ✅ Composability

**Middleware Chaining:**
```go
opts.Middleware = []middlewares.MiddlewareFunc{
    AuthMiddleware(token),
    LoggingMiddleware(logger),
    TracingMiddleware(tracer),
    RateLimitMiddleware(limiter),
}
```

**Decoder Selection:**
```go
// Choose decoder based on Content-Type
if contentType == "application/xml" {
    opts.ResponseDecoder = XMLDecoder
} else if contentType == "application/x-protobuf" {
    opts.ResponseDecoder = ProtobufDecoder
}
```

---

## Current Limitation & Future Enhancement

### Current State ⚠️

**Middleware:** ✅ Request-only (good)
**ResponseDecoder:** ✅ Response parsing (good)

**Missing:** ❌ Response Middleware (observability, logging, metrics)

### Industry Standard Enhancement

Most libraries provide **response middleware** for observability:

```go
// FUTURE ENHANCEMENT (not yet implemented)
type ResponseMiddlewareFunc func(*http.Response) (*http.Response, error)

// Would enable:
func ResponseLoggingMiddleware(logger *log.Logger) ResponseMiddlewareFunc {
    return func(resp *http.Response) (*http.Response, error) {
        logger.Printf("← %d %s", resp.StatusCode, resp.Request.URL)
        return resp, nil
    }
}

func MetricsMiddleware(metrics *Metrics) ResponseMiddlewareFunc {
    return func(resp *http.Response) (*http.Response, error) {
        metrics.RecordStatusCode(resp.StatusCode)
        metrics.RecordDuration(time.Since(startTime))
        return resp, nil
    }
}
```

**Future Addition:**
```go
type RequestOptions struct {
    // ... existing fields ...

    // Request processing (existing)
    Middleware []middlewares.MiddlewareFunc `json:"-"`

    // Response processing (NEW - observability, not decoding)
    ResponseMiddleware []middlewares.ResponseMiddlewareFunc `json:"-"`

    // Response parsing (existing - data transformation)
    ResponseDecoder ResponseDecoder `json:"-"`
}
```

---

## Decision Guide: Which Pattern to Use?

### Use **Middleware** when you need to:
- ✅ Add authentication headers
- ✅ Log requests
- ✅ Add tracing/correlation IDs
- ✅ Validate request before sending
- ✅ Sign requests (OAuth, AWS Signature)
- ✅ Modify URLs or headers dynamically
- ✅ Implement rate limiting
- ✅ Add API keys or tokens
- ✅ Transform request body

### Use **ResponseDecoder** when you need to:
- ✅ Parse XML responses
- ✅ Unmarshal Protocol Buffers
- ✅ Decode YAML
- ✅ Parse MessagePack
- ✅ Custom JSON processing (envelope unwrapping)
- ✅ Convert response to custom types
- ✅ Handle vendor-specific formats

### Use **Metrics Field** when you need to:
- ✅ Track request duration
- ✅ Record retry counts
- ✅ Monitor response sizes
- ✅ Measure connection times
- ✅ Collect observability data
- ✅ Debug performance issues

### Use **CustomClient** when you need to:
- ✅ Mock HTTP calls in tests
- ✅ Inject custom transport logic
- ✅ Implement circuit breakers
- ✅ Add connection pooling logic
- ✅ Test without real HTTP calls

---

## Complete Example: Using All Patterns Together

```go
package main

import (
    "context"
    "encoding/xml"
    "log"
    "time"

    "github.com/maniartech/gocurl"
    "github.com/maniartech/gocurl/middlewares"
    "github.com/maniartech/gocurl/options"
)

// 1. REQUEST MIDDLEWARE - Authentication
func authMiddleware(apiKey string) middlewares.MiddlewareFunc {
    return func(req *http.Request) (*http.Request, error) {
        req.Header.Set("X-API-Key", apiKey)
        return req, nil
    }
}

// 2. REQUEST MIDDLEWARE - Logging
func loggingMiddleware(logger *log.Logger) middlewares.MiddlewareFunc {
    return func(req *http.Request) (*http.Request, error) {
        logger.Printf("→ %s %s", req.Method, req.URL)
        return req, nil
    }
}

// 3. RESPONSE DECODER - XML Parser
type APIResponse struct {
    Status  string `xml:"status"`
    Message string `xml:"message"`
}

func xmlDecoder(resp *http.Response) (interface{}, error) {
    defer resp.Body.Close()

    var result APIResponse
    if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return result, nil
}

func main() {
    logger := log.New(os.Stdout, "[HTTP] ", log.LstdFlags)

    // Build options using all patterns
    opts := options.NewRequestOptions("https://api.example.com/data")
    opts.Method = "GET"

    // REQUEST PROCESSING - Middleware chain
    opts.Middleware = []middlewares.MiddlewareFunc{
        authMiddleware("secret-api-key"),
        loggingMiddleware(logger),
    }

    // RESPONSE PARSING - Custom decoder
    opts.ResponseDecoder = xmlDecoder

    // METRICS COLLECTION - Observability
    opts.Metrics = &options.RequestMetrics{
        StartTime: time.Now(),
    }

    // Execute
    resp, err := gocurl.Execute(context.Background(), opts)
    if err != nil {
        log.Fatal(err)
    }

    // Metrics populated after request
    logger.Printf("Duration: %v", opts.Metrics.Duration)
    logger.Printf("Status: %d", opts.Metrics.StatusCode)
    logger.Printf("Retries: %d", opts.Metrics.RetryCount)

    // Use decoded response
    if opts.ResponseDecoder != nil {
        decoded, err := opts.ResponseDecoder(resp.Response)
        if err != nil {
            log.Fatal(err)
        }

        apiResp := decoded.(APIResponse)
        logger.Printf("API Response: %s - %s", apiResp.Status, apiResp.Message)
    }
}
```

---

## Summary

### Industry Best Practice: ✅ **Keep Both Patterns**

| Pattern | Purpose | When | Example |
|---------|---------|------|---------|
| **Middleware** | Request preprocessing | Before sending | Auth, logging, headers |
| **ResponseDecoder** | Response parsing | After receiving | XML, Protobuf, YAML |
| **Metrics** | Observability | During/after request | Duration, retries, sizes |
| **CustomClient** | Testing/Mocking | Development/Testing | Unit tests, integration tests |

### Key Principles

1. **Separation of Concerns**: Each pattern has distinct responsibility
2. **Composability**: Patterns work together without conflict
3. **Industry Standard**: Matches axios, resty, Retrofit patterns
4. **Performance**: Optimize for common case (middleware) vs special case (decoder)
5. **Extensibility**: Users can add custom behavior without modifying library

### The Confusion is Actually a Feature! 🎯

Having both patterns might seem redundant, but it's an **industry-standard design** that:
- ✅ Follows **Single Responsibility Principle**
- ✅ Enables **clean separation** of concerns
- ✅ Provides **flexibility** for different use cases
- ✅ Matches **user expectations** from other HTTP clients

---

## Next Steps

### Potential Enhancements (Future)

1. **Response Middleware** (observability-focused)
   ```go
   type ResponseMiddlewareFunc func(*http.Response) (*http.Response, error)
   ```

2. **Built-in Decoder Library**
   ```go
   // gocurl/decoders package
   decoders.XML()
   decoders.YAML()
   decoders.Protobuf()
   decoders.MessagePack()
   ```

3. **Built-in Middleware Library**
   ```go
   // gocurl/middleware package
   middleware.Auth.Bearer(token)
   middleware.Logging(logger)
   middleware.Tracing(tracer)
   middleware.RateLimit(limiter)
   ```

4. **Middleware Chaining Helpers**
   ```go
   opts.Use(
       middleware.Chain(
           authMW,
           loggingMW,
           tracingMW,
       ),
   )
   ```

---

## References

**Industry Standards Analyzed:**
- [resty](https://github.com/go-resty/resty) - Go HTTP client
- [axios](https://axios-http.com/docs/interceptors) - JavaScript HTTP client
- [Retrofit](https://square.github.io/retrofit/) - Java HTTP client
- [OkHttp](https://square.github.io/okhttp/interceptors/) - Java/Kotlin HTTP client
- [Go net/http RoundTripper](https://pkg.go.dev/net/http#RoundTripper) - Go standard library

**Design Patterns:**
- Chain of Responsibility (Middleware)
- Strategy Pattern (ResponseDecoder)
- Decorator Pattern (Request/Response enhancement)
