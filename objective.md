# GoCurl Project Objectives

## Executive Summary

GoCurl is a **zero-allocation, military-grade performance** HTTP/HTTP2 client library for Go that revolutionizes how developers interact with REST APIs. By supporting HTTP-specific curl command syntax from third-party API documentation, GoCurl eliminates the translation overhead between API examples and Go implementation. The name "GoCurl" reflects its curl-compatible syntax for HTTP operations, though it focuses exclusively on HTTP/HTTPS features rather than implementing curl's full feature set (FTP, SMTP, etc.). Designed to replace `net/http` and other HTTP clients entirely, GoCurl delivers superior performance while maintaining developer-friendly syntax that allows copy-pasting HTTP curl commands directly into production Go code.

## Primary Objective

**Deliver a zero-allocation, ultra-high-performance HTTP/HTTP2 client that allows Go developers to use HTTP-specific curl command syntax from third-party API documentation directly in their Go code, enabling seamless API integration and the creation of type-safe Go wrappers without the need for SDK-specific learning curves or manual request translation.**

## Core Goals

### 1. Performance Excellence
- **Zero-allocation architecture** for minimal garbage collection overhead
- **Military-grade robustness** with battle-tested reliability for production systems
- **Superior performance** to `net/http` and existing HTTP client libraries
- Efficient connection pooling and keep-alive management
- Optimized for both HTTP/1.1 and HTTP/2 protocols
- Sub-microsecond overhead for request construction

### 2. Direct Curl-to-Go Execution

- Enable **copy-paste of HTTP-specific curl commands** from API documentation directly into Go code
- **Accept curl commands as strings** - no need to convert to string slices
- Built-in parser handles curl command string parsing automatically
- Execute HTTP curl syntax **without translation or modification**
- Support HTTP-related curl flags and options used in real-world API examples
- Seamless variable substitution for dynamic values (API keys, tokens, parameters)
- Preserve the exact semantics of curl's HTTP behavior (not FTP, SMTP, or other protocols)
- Alternative string slice input supported for programmatic construction

### 3. Universal API Integration
- **Replace net/http** as the primary HTTP client for Go projects
- Provide a single, unified interface for all HTTP/HTTP2 interactions
- Support consumption of APIs without needing official SDKs
- Enable rapid integration with third-party services (Stripe, GitHub, AWS, etc.)
- Support all HTTP methods and modern web standards

### 4. Developer Experience

- Eliminate the learning curve for new API integrations
- Allow developers to use API documentation examples directly
- **Test APIs quickly using the gocurl CLI** before integrating into projects
- **Seamless CLI-to-code workflow**: test with CLI, then copy the same command into Go code
- Minimize cognitive overhead when switching between different APIs

### 5. Security and Reliability

- Implement comprehensive input validation and sanitization
- Provide secure handling of sensitive data (API keys, tokens)
- Support modern authentication mechanisms
- Include timeout and retry mechanisms for resilient applications
- Enforce TLS security best practices

## Key Features to Deliver

### Performance & Architecture

- [ ] **Zero-allocation request/response handling**
- [ ] **Memory pooling** for reusable buffers and objects
- [ ] **Lock-free data structures** where applicable
- [ ] Benchmark suite proving superiority over `net/http`
- [ ] Sub-microsecond request construction overhead
- [ ] Efficient connection pooling and reuse
- [ ] HTTP/2 multiplexing optimization
- [ ] Streaming support for large request/response bodies

### HTTP Protocol Support

- [ ] All HTTP methods (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)
- [ ] HTTP/1.1 and HTTP/2 protocol support
- [ ] Custom header management
- [ ] Cookie handling and persistence
- [ ] Compression support (gzip, deflate, brotli)

### Request Construction

- [ ] **Direct curl command execution** (copy-paste from documentation)
- [ ] Variable substitution in curl commands
- [ ] File upload capabilities (multipart/form-data)
- [ ] Form data encoding
- [ ] JSON request body support
- [ ] Query parameter handling
- [ ] Support for all curl flags and options

### Curl Command Compatibility

- [ ] Complete HTTP-specific curl syntax parser
- [ ] Support for shorthand and long-form curl flags (HTTP operations only)
- [ ] Curl command string parsing (single string input)
- [ ] Curl command array parsing ([]string input)
- [ ] Preserve curl's HTTP semantics and behavior exactly
- [ ] Error messages that match curl's conventions for HTTP errors

### Authentication

- [ ] Basic authentication
- [ ] Bearer token authentication
- [ ] Custom authentication header support

### Network Features

- [ ] HTTP and SOCKS5 proxy support
- [ ] TLS/SSL configuration
- [ ] Certificate verification options
- [ ] Redirect handling

### Response Processing

- [ ] JSON response parsing
- [ ] Response body streaming
- [ ] Error handling and status code management
- [ ] Response header access

### Developer Tools

- [ ] **Command-line interface (CLI)** for rapid API testing and exploration
- [ ] **CLI-to-code workflow**: exact same syntax works in both CLI and Go code
- [ ] Library API for programmatic use
- [ ] Debugging and verbose output modes
- [ ] Response formatting and prettification in CLI

## Target Use Cases

1. **API Exploration and Testing**
   - **Quick CLI testing**: Run `gocurl` command to test API endpoints instantly
   - **Verify API behavior** before writing Go code
   - **CLI-to-Go migration**: Once tested, copy the exact command into Go code
   - Documentation verification

2. **Rapid Development Workflow**
   - **Test with CLI first**: `gocurl -H "Authorization: Bearer $TOKEN" https://api.example.com/data`
   - **Copy to Go second**: Same command works in `gocurl.Request()`
   - **No syntax translation needed**: CLI and library use identical syntax
   - Immediate feedback loop for API integration

3. **Production Applications**
   - Service-to-service communication in Go microservices
   - Integration with third-party SaaS platforms (Stripe, GitHub, AWS, etc.)
   - Internal API consumption
   - CI/CD pipeline API interactions
   - Automated testing of external APIs

## Success Metrics

- **Performance**: Zero-allocation execution with benchmarks proving superiority over `net/http`
- **Adoption**: Number of projects replacing `net/http` with GoCurl
- **Curl Compatibility**: 100% compatibility with HTTP curl commands from major API providers
- **Developer Satisfaction**: Time-to-integration metrics (minutes vs. hours/days)
- **Reliability**: API success rate and error handling effectiveness

## Non-Goals

- Creating a GraphQL client (focus remains on REST/HTTP APIs)
- Building a full-featured API gateway
- Implementing server-side functionality
- Supporting non-HTTP curl protocols (FTP, SMTP, TFTP, LDAP, etc.)
- Providing a GUI or web interface
- Full curl compatibility beyond HTTP/HTTPS protocols

## Real-World Usage Example

```bash
# Step 1: Test with CLI first (exact same syntax as in API docs)
$ gocurl -X POST https://api.stripe.com/v1/charges \
  -u sk_test_YOUR_STRIPE_TEST_KEY: \
  -d amount=2000 \
  -d currency=usd

# Step 2: Once it works, use the EXACT same command in Go code
```

```go
// Example 1: Direct curl command from Stripe documentation (as string)
// Documentation shows: curl https://api.stripe.com/v1/charges \
//   -u sk_test_YOUR_STRIPE_TEST_KEY: \
//   -d amount=2000 \
//   -d currency=usd

// Copy-paste the exact curl command as a string - parser handles everything
curlCommand := `curl https://api.stripe.com/v1/charges \
  -u sk_test_YOUR_STRIPE_TEST_KEY: \
  -d amount=2000 \
  -d currency=usd`
response, err := gocurl.Request(curlCommand, nil)

// Example 2: Same command as single line
response, err := gocurl.Request(
    "curl https://api.stripe.com/v1/charges -u sk_test_YOUR_STRIPE_TEST_KEY: -d amount=2000 -d currency=usd",
    nil,
)

// Example 3: With variable substitution
variables := gocurl.Variables{
    "api_key": os.Getenv("STRIPE_API_KEY"),
    "amount": "2000",
}

response, err := gocurl.Request(
    "curl https://api.stripe.com/v1/charges -u ${api_key}: -d amount=${amount} -d currency=usd",
    variables,
)

// Example 4: Alternative - using string slice (if preferred)
response, err := gocurl.Request([]string{
    "curl", "https://api.stripe.com/v1/charges",
    "-u", "sk_test_YOUR_STRIPE_TEST_KEY:",
    "-d", "amount=2000",
    "-d", "currency=usd",
}, nil)

// Example 5: Building type-safe Go API
func CreateCharge(apiKey string, amount int, currency string) (*Charge, error) {
    // Use the exact curl command syntax - no manual parsing needed
    response, err := gocurl.Request(
        fmt.Sprintf("curl https://api.stripe.com/v1/charges -u %s: -d amount=%d -d currency=%s",
            apiKey, amount, currency),
        nil,
    )

    if err != nil {
        return nil, err
    }

    var charge Charge
    if err := json.Unmarshal([]byte(response), &charge); err != nil {
        return nil, err
    }

    return &charge, nil
}

// Example 6: GitHub API (from their docs) - exact copy-paste
// curl -L \
//   -H "Accept: application/vnd.github+json" \
//   -H "Authorization: Bearer <YOUR-TOKEN>" \
//   https://api.github.com/repos/OWNER/REPO/issues

response, err := gocurl.Request(`curl -L \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${token}" \
  https://api.github.com/repos/maniartech/gocurl/issues`,
  gocurl.Variables{"token": os.Getenv("GITHUB_TOKEN")})
```

## Workflow Example

```bash
# Developer workflow - Test first, code second

# 1. See API documentation with curl example
# 2. Test immediately with gocurl CLI:
$ gocurl -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  https://api.github.com/repos/maniartech/gocurl/issues

# 3. Works! Now use EXACT same syntax in Go:
```

```go
response, err := gocurl.Request(
    `gocurl -H "Authorization: Bearer ` + token + `" \
     -H "Accept: application/vnd.github+json" \
     https://api.github.com/repos/maniartech/gocurl/issues`,
    nil,
)

// Or even simpler - single line
response, err := gocurl.Request(
    "gocurl -H \"Authorization: Bearer " + token + "\" -H \"Accept: application/vnd.github+json\" https://api.github.com/repos/maniartech/gocurl/issues",
    nil,
)
```## Project Phases

### Phase 1: Foundation (MVP)

- Zero-allocation architecture implementation
- Core curl command parsing (complete syntax support)
- Basic HTTP methods support with zero-alloc execution
- Simple authentication mechanisms
- JSON response handling
- Performance benchmarking framework

### Phase 2: Enhancement

- Advanced authentication (OAuth)
- Proxy support (HTTP and SOCKS5)
- File upload capabilities
- Connection pooling optimization
- HTTP/2 support

### Phase 3: Developer Experience

- **CLI tool development** with identical syntax to library
- **CLI testing capabilities** for rapid API exploration
- Comprehensive documentation with real API examples
- **CLI-to-code workflow documentation** and tutorials
- Example library for common APIs (Stripe, GitHub, AWS, etc.)

### Phase 4: Performance & Production Readiness

- Advanced performance optimization (lock-free structures)
- Advanced security features
- Performance comparison dashboard
- Production-grade error handling and recovery

## Long-term Vision

Establish GoCurl as the **primary HTTP client** for Go developers, replacing `net/http` and other HTTP libraries in production systems. Create a paradigm shift where developers can:

1. **Test with CLI first**: `gocurl [command]` for instant API verification
2. **Copy HTTP curl commands** from any API documentation
3. **Paste them directly** into Go code with identical syntax
4. **Run them with zero modification**
5. **Build type-safe wrappers** when needed
6. **Achieve better performance** than hand-written `net/http` code

Make GoCurl the universal standard that eliminates the "SDK gap" problem—where APIs exist but Go SDKs don't—by making HTTP curl commands themselves the universal SDK. The CLI-to-code workflow ensures developers can validate API behavior instantly before committing code. The name "GoCurl" signifies curl-compatible syntax for HTTP operations, not full curl feature parity. Build a thriving ecosystem of:

- Pre-built Go wrappers for popular APIs (generated from HTTP curl examples)
- Performance benchmarks proving GoCurl's superiority
- Integration examples for every major HTTP-based API platform
- CLI testing guides and best practices

**Ultimate Goal**: When someone asks "How do I call this API in Go?", the workflow is:
1. Test with `gocurl [command]` CLI
2. Once working, use the exact same command in Go with `gocurl.Request()`
3. No translation, no guesswork, just copy-paste from CLI to code
