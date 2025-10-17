# Chapter 3: Core Concepts - Examples

Working examples demonstrating gocurl's core concepts and function categories.

## Examples Overview

1. **01-function-categories** - All six function categories demonstrated
2. **02-variable-expansion** - Environment and explicit variable usage
3. **03-context-patterns** - Timeout, cancellation, deadline examples
4. **04-response-handling** - Different response parsing techniques
5. **05-error-handling** - Status codes and error response patterns
6. **06-process-direct** - Using Process() function directly
7. **07-curl-vs-builder** - Comparing curl-syntax vs builder approaches
8. **08-streaming** - Streaming large responses
9. **09-json-parsing** - Advanced JSON unmarshaling patterns
10. **10-practical-client** - Complete API client implementation

## Running Examples

```bash
# Run any example
cd 01-function-categories
go run main.go

# Or from this directory
go run 01-function-categories/main.go
```

## Prerequisites

- Go 1.21+
- GoCurl installed
- Internet connection
- Some examples need environment variables (see example code)

## Key Concepts Demonstrated

- **Function Categories**: When to use Curl, CurlString, CurlBytes, CurlJSON, CurlDownload
- **Variable Expansion**: Automatic vs explicit, security best practices
- **Context Management**: Timeouts, cancellation, deadline control
- **Response Handling**: String, bytes, JSON, streaming, error responses
- **Process() Function**: Core execution engine usage

## Learning Path

**Beginner:**
1. Start with 01-function-categories
2. Then 02-variable-expansion
3. Then 03-context-patterns

**Intermediate:**
4. Study 04-response-handling
5. Practice with 05-error-handling
6. Understand 06-process-direct

**Advanced:**
7. Compare 07-curl-vs-builder
8. Master 08-streaming
9. Advanced 09-json-parsing
10. Build complete 10-practical-client

## Next Steps

After completing examples:
1. Try exercises in `../exercises/`
2. Read Chapter 4: CLI
3. Build your own API client
