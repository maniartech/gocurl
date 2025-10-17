# Chapter 6 Examples: Working with JSON APIs

This directory contains 10 working examples demonstrating JSON request/response patterns with GoCurl.

## Examples Overview

### Basic JSON Operations
1. **[01-basic-json](01-basic-json/)** - Simple JSON GET with CurlJSON
2. **[02-json-array](02-json-array/)** - Fetching JSON arrays
3. **[03-post-json](03-post-json/)** - Sending JSON data with POST

### Advanced Patterns
4. **[04-nested-json](04-nested-json/)** - Working with nested structures
5. **[05-optional-fields](05-optional-fields/)** - Handling optional/null fields
6. **[06-error-handling](06-error-handling/)** - Parsing JSON error responses

### Real-World Integration
7. **[07-github-client](07-github-client/)** - Complete GitHub API client
8. **[08-pagination](08-pagination/)** - Handling paginated responses
9. **[09-generic-fetcher](09-generic-fetcher/)** - Type-safe generic JSON fetcher
10. **[10-caching](10-caching/)** - JSON response caching layer

## Running Examples

Each example is self-contained and can be run independently:

```bash
cd 01-basic-json
go run main.go
```

## Prerequisites

- Go 1.18+ (for generics in examples 09 and 10)
- GitHub token (for example 07 - set `GITHUB_TOKEN` environment variable)
- Internet connection (examples use real APIs)

## APIs Used

- **httpbin.org** - Testing HTTP requests
- **api.github.com** - GitHub REST API (requires token for higher rate limits)
- **jsonplaceholder.typicode.com** - Fake REST API for testing

## Key Learning Points

- ✅ Using CurlJSON for automatic unmarshaling
- ✅ Sending JSON with POST/PUT requests
- ✅ Handling nested and complex JSON structures
- ✅ Working with optional fields and null values
- ✅ Parsing JSON error responses
- ✅ Building type-safe API clients
- ✅ Implementing pagination patterns
- ✅ Using Go generics for reusable code
- ✅ Adding caching layers

## Example Difficulty

| Example | Difficulty | Time |
|---------|-----------|------|
| 01-03 | Beginner | 5-10 min each |
| 04-06 | Intermediate | 10-15 min each |
| 07-10 | Advanced | 20-30 min each |

## Next Steps

After completing these examples:
1. Try modifying examples to use different APIs
2. Combine patterns from multiple examples
3. Complete the exercises in `../exercises/`
4. Build your own API client for a service you use
