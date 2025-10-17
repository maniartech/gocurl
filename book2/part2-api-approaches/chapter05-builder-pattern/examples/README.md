# Chapter 5: RequestOptions & Builder Pattern - Examples

This directory contains practical examples demonstrating the Builder pattern for constructing HTTP requests.

## Examples Overview

1. **01-basic-builder** - Simple GET request with builder
2. **02-post-json** - POST request with JSON body
3. **03-authentication** - Bearer token and Basic Auth
4. **04-clone-concurrent** - Safe concurrent requests with Clone()
5. **05-context-management** - WithTimeout() and Cleanup()
6. **06-validation** - Request validation before execution
7. **07-convenience-methods** - JSON(), Form(), WithDefaultRetry()
8. **08-request-template** - Reusable request templates

## Running Examples

Each example is a standalone Go program:

```bash
cd 01-basic-builder
go run main.go
```

## Prerequisites

- Go 1.21 or higher
- GoCurl library: `go get github.com/maniartech/gocurl`
- Internet connectivity

## Learning Path

Start with `01-basic-builder` and progress sequentially. Each example builds on concepts from previous ones.

## Key Concepts Demonstrated

- **Builder Pattern:** Fluent API for constructing requests
- **RequestOptions:** 30+ configuration fields
- **Thread Safety:** Using Clone() for concurrent requests
- **Context Management:** WithTimeout(), WithContext(), Cleanup()
- **Validation:** Validate() method for early error detection
- **Convenience Methods:** JSON(), Form(), WithDefaultRetry()
- **HTTP Shortcuts:** Get(), Post(), Put(), Delete(), Patch()

## Example Structure

Each example demonstrates:
1. Creating a RequestOptionsBuilder
2. Configuring the request
3. Building RequestOptions
4. Executing with gocurl.Execute()
5. Handling the response

## Common Patterns

### Basic Request
```go
opts := options.NewRequestOptionsBuilder().
    SetURL("https://api.example.com").
    SetMethod("GET").
    Build()

resp, err := gocurl.Execute(ctx, opts)
```

### With Timeout
```go
builder := options.NewRequestOptionsBuilder().
    SetURL("https://api.example.com").
    WithTimeout(30 * time.Second)
defer builder.Cleanup()

resp, err := gocurl.Execute(builder.GetContext(), builder.Build())
```

### Concurrent with Clone
```go
baseOpts := options.NewRequestOptions("https://api.example.com")

// In goroutine
opts := baseOpts.Clone()
opts.AddHeader("X-ID", "unique-id")
resp, err := gocurl.Execute(ctx, opts)
```

## Comparison with Curl-Syntax

Each builder example shows the equivalent curl-syntax function call for comparison.

## Next Steps

After completing these examples:
1. Practice building your own requests
2. Experiment with different configurations
3. Complete the chapter exercises
4. Proceed to Chapter 6 (JSON APIs)
