# GoCurl API Design - Final Recommendation
## Date: October 15, 2025

## Recommended API: Hybrid Approach

### Three Functions (Best of All Worlds)

```go
// Curl - Primary API with auto-detection
// Automatically detects syntax based on argument count:
// - Single argument: Treated as shell command (parsed, multi-line supported)
// - Multiple arguments: Treated as variadic args (direct tokenization)
//
// Use this for 90% of cases - it "just works"
func Curl(ctx context.Context, command ...string) (*Response, string, error)

// CurlCommand - Explicit shell command parsing
// ALWAYS treats input as a shell command string.
// Use when:
// - Copying from API documentation
// - Copying from browser DevTools
// - Multi-line commands with backslashes
// - You want to be explicit about parsing
//
// Example:
//   cmd := `curl -X POST https://api.example.com \
//             -H 'Authorization: Bearer $TOKEN' \
//             -d '{"key":"value"}'`
//   resp, body, err := gocurl.CurlCommand(ctx, cmd)
func CurlCommand(ctx context.Context, shellCommand string) (*Response, string, error)

// CurlArgs - Explicit variadic arguments
// ALWAYS treats inputs as separate arguments (no parsing).
// Use when:
// - You want compile-time safety
// - Building commands programmatically
// - You have a single URL (avoids ambiguity)
// - Performance critical (skips parsing)
//
// Example:
//   url := "https://example.com"
//   resp, body, err := gocurl.CurlArgs(ctx, url)
func CurlArgs(ctx context.Context, args ...string) (*Response, string, error)
```

## Usage Examples

### Example 1: Simple GET (Auto-detect)

```go
// Auto-detect: Multiple args → variadic mode
resp, body, err := gocurl.Curl(ctx, "https://example.com")

// Auto-detect: Single arg → shell mode (but works fine!)
url := "https://example.com"
resp, body, err := gocurl.Curl(ctx, url)
```

### Example 2: Copy from Browser DevTools (Explicit)

```go
// Chrome: Right-click → Copy as cURL
command := `curl 'https://api.github.com/repos/owner/repo/issues' \
  -H 'Accept: application/vnd.github+json' \
  -H 'Authorization: Bearer ghp_xxxxx' \
  --compressed`

// Explicit - makes intent clear
resp, body, err := gocurl.CurlCommand(ctx, command)
```

### Example 3: Copy from API Docs (Explicit)

```go
// Stripe API documentation
stripeCommand := `curl https://api.stripe.com/v1/charges \
  -u sk_test_xyz: \
  -d amount=2000 \
  -d currency=usd \
  -d source=tok_visa`

resp, body, err := gocurl.CurlCommand(ctx, stripeCommand)
```

### Example 4: Programmatic Building (Explicit)

```go
// Building URL programmatically
baseURL := "https://api.example.com"
endpoint := "/users"
url := baseURL + endpoint

// Explicit - avoids any parsing ambiguity
resp, body, err := gocurl.CurlArgs(ctx, url)

// Or with headers
resp, body, err := gocurl.CurlArgs(ctx,
    "-H", "Authorization: Bearer " + token,
    "-H", "Content-Type: application/json",
    url,
)
```

### Example 5: Auto-detect (Most Common)

```go
// Works for both!

// Variadic style
resp, body, err := gocurl.Curl(ctx,
    "-X", "POST",
    "-H", "Content-Type: application/json",
    "-d", `{"key":"value"}`,
    "https://api.example.com",
)

// Shell style
resp, body, err := gocurl.Curl(ctx,
    `-X POST -H 'Content-Type: application/json' -d '{"key":"value"}' https://api.example.com`,
)
```

## Implementation

### Primary Function (Auto-detect)

```go
// Curl - Auto-detects syntax based on argument count
func Curl(ctx context.Context, command ...string) (*Response, string, error) {
    if len(command) == 0 {
        return nil, "", fmt.Errorf("no command provided")
    }

    if len(command) == 1 {
        // Single string → treat as shell command
        return CurlCommand(ctx, command[0])
    }

    // Multiple args → treat as variadic
    return CurlArgs(ctx, command...)
}
```

### Shell Command Function (Explicit)

```go
// CurlCommand - ALWAYS parses as shell command
func CurlCommand(ctx context.Context, shellCommand string) (*Response, string, error) {
    // 1. Preprocess multi-line (backslashes, comments, curl prefix)
    cmdStr := preprocessMultilineCommand(shellCommand)

    // 2. Tokenize as shell command
    tokens, err := tokenizer.Tokenize(cmdStr)
    if err != nil {
        return nil, "", fmt.Errorf("failed to parse command: %w", err)
    }

    // 3. Expand environment variables
    tokens = expandEnvInTokens(tokens)

    // 4. Convert to options
    opts, err := ConvertTokensToRequestOptions(tokens)
    if err != nil {
        return nil, "", err
    }

    // 5. Execute
    return Process(ctx, opts)
}
```

### Variadic Function (Explicit)

```go
// CurlArgs - ALWAYS treats as separate arguments
func CurlArgs(ctx context.Context, args ...string) (*Response, string, error) {
    if len(args) == 0 {
        return nil, "", fmt.Errorf("no arguments provided")
    }

    // 1. Direct tokenization (no parsing)
    tokens, err := TokenizeArgs(args)
    if err != nil {
        return nil, "", fmt.Errorf("failed to tokenize arguments: %w", err)
    }

    // 2. Expand environment variables
    tokens = expandEnvInTokens(tokens)

    // 3. Convert to options
    opts, err := ConvertTokensToRequestOptions(tokens)
    if err != nil {
        return nil, "", err
    }

    // 4. Execute
    return Process(ctx, opts)
}
```

## With Variable Maps

Same pattern for explicit variable maps:

```go
// Auto-detect
func CurlWithVars(ctx context.Context, vars map[string]string, command ...string) (*Response, string, error)

// Explicit shell command
func CurlCommandWithVars(ctx context.Context, vars map[string]string, shellCommand string) (*Response, string, error)

// Explicit variadic
func CurlArgsWithVars(ctx context.Context, vars map[string]string, args ...string) (*Response, string, error)
```

## Decision Matrix

| Use Case | Recommended Function | Why |
|----------|---------------------|-----|
| Simple GET | `Curl(ctx, url)` | Auto-detect works fine |
| Variadic args | `Curl(ctx, "-H", "X-Token", url)` | Auto-detect (multiple args) |
| Copy from browser | `CurlCommand(ctx, copiedCommand)` | Explicit intent |
| Copy from API docs | `CurlCommand(ctx, docCommand)` | Explicit intent |
| Multi-line command | `CurlCommand(ctx, multilineCmd)` | Explicit intent |
| Programmatic URL | `CurlArgs(ctx, buildURL())` | Avoids ambiguity |
| Performance critical | `CurlArgs(ctx, args...)` | Skips parsing |
| Single URL variable | `CurlArgs(ctx, url)` | Explicit (avoids ambiguity) |

## API Documentation

```go
// Curl executes an HTTP request using curl-compatible syntax.
//
// Automatically detects input format:
// - Single argument: Parsed as shell command (supports multi-line, backslashes, comments)
// - Multiple arguments: Treated as separate arguments (no parsing)
//
// For explicit control, use CurlCommand() or CurlArgs().
//
// Examples:
//   // Variadic (auto-detected)
//   resp, body, err := Curl(ctx, "-H", "X-Token: abc", "https://example.com")
//
//   // Shell command (auto-detected)
//   resp, body, err := Curl(ctx, `curl -H 'X-Token: abc' https://example.com`)
//
//   // Multi-line (auto-detected)
//   resp, body, err := Curl(ctx, `
//     curl -X POST https://api.example.com \
//       -H 'Content-Type: application/json' \
//       -d '{"key":"value"}'
//   `)
func Curl(ctx context.Context, command ...string) (*Response, string, error)

// CurlCommand executes an HTTP request from a shell command string.
//
// ALWAYS parses the input as a shell command, even if it could be
// interpreted as a simple URL. Use this when:
// - Copying commands from API documentation
// - Copying "Copy as cURL" from browser DevTools
// - Working with multi-line commands
// - You want to be explicit about shell parsing
//
// Preprocessing includes:
// - Backslash line continuations (\)
// - Comment stripping (#)
// - 'curl' prefix removal
// - Environment variable expansion ($VAR)
//
// Examples:
//   // API documentation
//   cmd := `curl https://api.stripe.com/v1/charges \
//             -u sk_test_xyz: \
//             -d amount=2000`
//   resp, body, err := CurlCommand(ctx, cmd)
//
//   // Browser DevTools
//   cmd := `curl 'https://api.example.com/graphql' \
//             -X POST \
//             -H 'authorization: Bearer ...' \
//             --data-raw '{"query":"..."}'`
//   resp, body, err := CurlCommand(ctx, cmd)
func CurlCommand(ctx context.Context, shellCommand string) (*Response, string, error)

// CurlArgs executes an HTTP request from separate arguments.
//
// ALWAYS treats inputs as separate arguments (no shell parsing).
// Use this when:
// - Building commands programmatically
// - You have a single URL and want to avoid parsing ambiguity
// - Performance is critical (skips shell parsing)
// - You want compile-time argument checking
//
// Examples:
//   // Simple URL
//   url := buildURL()
//   resp, body, err := CurlArgs(ctx, url)
//
//   // With headers
//   resp, body, err := CurlArgs(ctx,
//       "-H", "Authorization: Bearer " + token,
//       "-H", "Content-Type: application/json",
//       url,
//   )
func CurlArgs(ctx context.Context, args ...string) (*Response, string, error)
```

## Benefits of This Approach

### 1. Gradual Learning Curve

```go
// Beginner: Just use Curl()
gocurl.Curl(ctx, "https://example.com")

// Intermediate: Still use Curl() with copy/paste
gocurl.Curl(ctx, `curl -H 'X-Token' https://example.com`)

// Advanced: Use explicit functions for clarity
gocurl.CurlCommand(ctx, complexCommand)
```

### 2. Clear Intent in Code Review

```go
// This is clear: "I'm using a shell command"
resp, body, err := gocurl.CurlCommand(ctx, copiedFromBrowser)

// This is clear: "I'm building programmatically"
resp, body, err := gocurl.CurlArgs(ctx, url)

// This might be ambiguous (but still works!)
resp, body, err := gocurl.Curl(ctx, someString)
```

### 3. Future-Proof

If we ever need to change auto-detection logic, the explicit functions maintain their behavior.

### 4. Discovery

IDE autocomplete shows three options with clear names:
- `Curl()` - General purpose
- `CurlCommand()` - Shell command
- `CurlArgs()` - Variadic args

## Recommendation Summary

✅ **Implement all three functions**
✅ **Document when to use each**
✅ **Make `Curl()` the primary API** (90% of use cases)
✅ **Provide explicit alternatives** for clarity and edge cases

This gives developers:
- **Flexibility**: Auto-detect "just works"
- **Clarity**: Explicit functions when needed
- **Safety**: Type system helps catch errors
- **Simplicity**: One function for most cases

---

**Ready to implement this API design?**
