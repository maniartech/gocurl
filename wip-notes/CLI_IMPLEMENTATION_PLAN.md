# GoCurl CLI Implementation Plan - October 15, 2025

## Core Objective

**"Test with CLI first" - This is the ABSOLUTE CORE objective of this library.**

The workflow MUST be:

1. **Test API with CLI**: `gocurl [curl-command]` - instant feedback
2. **Copy to Go code**: Exact same command - zero translation
3. **Same behavior**: Guaranteed by curl parity tests

## Key Decisions Summary

✅ **Response API**: Return `*http.Response` (not string) with convenience helpers
✅ **Curl Parity**: Every test MUST compare against real curl command
✅ **Multi-line Support**: All formats (backslashes, comments, browser DevTools)
✅ **Environment Variables**: Unified expansion in both CLI and library
✅ **Three Functions**: Curl() (auto-detect), CurlCommand() (shell), CurlArgs() (variadic)
✅ **Zero Divergence**: CLI and library use EXACT same code path

### Example: The Complete Workflow

**Workflow 1: Copy/Paste from Browser DevTools**

```bash
# Step 1: Open browser DevTools → Network tab → Right-click request → Copy as cURL
# You get something like:
curl 'https://api.example.com/search' \
  -X POST \
  -H 'Authorization: Bearer sk_live_123456' \
  -H 'Content-Type: application/json' \
  --data-raw '{"query": "test"}'
```

```go
// Step 2: Paste into Go code - Just remove 'curl' prefix!
package main

import (
    "context"
    "fmt"
    "io"

    "github.com/maniartech/gocurl"
)

func main() {
    ctx := context.Background()

    // Option A: Full control with *http.Response
    resp, err := gocurl.Curl(ctx, `
        'https://api.example.com/search'
        -X POST
        -H 'Authorization: Bearer sk_live_123456'
        -H 'Content-Type: application/json'
        --data-raw '{"query": "test"}'
    `)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    fmt.Println(string(body))  // ✅ Same result!

    // Option B: Convenience - auto-read body
    // bodyStr, resp, err := gocurl.CurlString(ctx, `...`)
}
```

**Workflow 2: With Environment Variables**

```bash
# Step 1: Test with CLI (using environment variables)
$ export API_TOKEN=sk_live_123456
$ gocurl -X POST \
    -H "Authorization: Bearer $API_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"query": "test"}' \
    https://api.example.com/search

# ✅ Works! Returns: {"results": [...]}
```

```go
// Step 2: Copy to Go code - TWO syntax options!

// Option A: Variadic syntax (clean, Go-style) - Full control
os.Setenv("API_TOKEN", "sk_live_123456")
resp, err := gocurl.Curl(ctx,
    "-X", "POST",
    "-H", "Authorization: Bearer $API_TOKEN",  // ✅ Auto-expands!
    "-H", "Content-Type: application/json",
    "-d", `{"query": "test"}`,
    "https://api.example.com/search",
)
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)

// Option B: Single string (direct copy/paste from shell!)
os.Setenv("API_TOKEN", "sk_live_123456")
resp, err := gocurl.Curl(ctx,
    `-X POST -H "Authorization: Bearer $API_TOKEN" -H "Content-Type: application/json" -d '{"query": "test"}' https://api.example.com/search`,
)
defer resp.Body.Close()
body, _ = io.ReadAll(resp.Body)

// Option C: Convenience - auto-read body to string
bodyStr, resp, err := gocurl.CurlString(ctx,
    "-X", "POST",
    "-H", "Authorization: Bearer $API_TOKEN",
    "-H", "Content-Type: application/json",
    "-d", `{"query": "test"}`,
    "https://api.example.com/search",
)
// bodyStr is ready to use, resp has headers/status

// ✅ ALL options produce identical results!
```

**Key Points:**

- ✅ **Copy from browser**: Copy as cURL → paste into Go (remove 'curl' prefix)
- ✅ **Copy from shell**: Test with gocurl CLI → copy exact command to Go
- ✅ **Environment variables**: `$API_TOKEN` expands in both CLI and library
- ✅ **Two syntaxes**: Variadic (clean) or single string (copy/paste)
- ✅ **Same result**: Guaranteed by roundtrip tests
- ✅ **Zero translation**: No manual conversion needed

**Workflow 3: Multi-line Commands (API Docs / Sysadmin Style)**

```bash
# Common in API documentation and shell scripts:
curl -X POST https://api.stripe.com/v1/charges \
  -u sk_test_xyz: \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d amount=2000 \
  -d currency=usd \
  -d source=tok_visa \
  -d description="Charge for demo"
```

```go
// Option 1: Full control with *http.Response
resp, err := gocurl.Curl(ctx, `
curl -X POST https://api.stripe.com/v1/charges \
  -u sk_test_xyz: \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d amount=2000 \
  -d currency=usd \
  -d source=tok_visa \
  -d description="Charge for demo"
`)
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)

// Option 2: Without 'curl' prefix (cleaner):
resp, err := gocurl.Curl(ctx, `
-X POST https://api.stripe.com/v1/charges \
  -u sk_test_xyz: \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d amount=2000 \
  -d currency=usd \
  -d source=tok_visa \
  -d description="Charge for demo"
`)
defer resp.Body.Close()
body, _ = io.ReadAll(resp.Body)

// Option 3: Without backslashes (gocurl treats newlines as spaces):
resp, err := gocurl.Curl(ctx, `
-X POST https://api.stripe.com/v1/charges
  -u sk_test_xyz:
  -H "Content-Type: application/x-www-form-urlencoded"
  -d amount=2000
  -d currency=usd
  -d source=tok_visa
  -d description="Charge for demo"
`)
defer resp.Body.Close()
body, _ = io.ReadAll(resp.Body)

// Option 4: Convenience - auto-decode JSON response
var charge StripeCharge
resp, err := gocurl.CurlJSON(ctx, &charge, `
-X POST https://api.stripe.com/v1/charges
  -u sk_test_xyz:
  -d amount=2000
  -d currency=usd
`)
// charge struct is populated, resp has status/headers

// ✅ ALL forms produce IDENTICAL results!
```

**Workflow 4: From README/Documentation (Markdown Code Blocks)**

```bash
# API docs often show:
```bash
curl -X GET "https://api.github.com/repos/owner/repo/issues" \
     -H "Accept: application/vnd.github+json" \
     -H "Authorization: Bearer $GITHUB_TOKEN"
```
````

```go
// Just copy the command and paste:
// Option A: Full control with *http.Response
resp, err := gocurl.Curl(ctx, `
curl -X GET "https://api.github.com/repos/owner/repo/issues" \
     -H "Accept: application/vnd.github+json" \
     -H "Authorization: Bearer $GITHUB_TOKEN"
`)
defer resp.Body.Close()

// Option B: Auto-decode JSON to struct
var issues []GitHubIssue
resp, err := gocurl.CurlJSON(ctx, &issues, `
curl -X GET "https://api.github.com/repos/owner/repo/issues" \
     -H "Accept: application/vnd.github+json" \
     -H "Authorization: Bearer $GITHUB_TOKEN"
`)
// issues slice is populated!

// Gocurl automatically:
// ✅ Strips 'curl' prefix if present
// ✅ Handles backslash line continuations
// ✅ Handles both \n and \r\n line endings
// ✅ Preserves quotes and escaping
// ✅ Expands $GITHUB_TOKEN from environment
// ✅ Returns *http.Response for maximum flexibility
```

**Multi-line Processing Rules:**

1. **Line continuations**: `\` at end of line → join with next line
2. **Newlines**: Treated as whitespace (space separator)
3. **Indentation**: Automatically trimmed
4. **curl prefix**: Automatically removed if present
5. **Comments**: Lines starting with `#` are ignored
6. **Empty lines**: Automatically skipped## Core Principle: ZERO DIVERGENCE

**Critical Requirement**: CLI and library MUST use the EXACT same code path.

## Response API Design

**DECISION**: Return `*http.Response` instead of `string` for maximum flexibility.

### Core Functions (Return *http.Response)

```go
// Curl - Auto-detect input format, return response
func Curl(ctx, ...string) (*http.Response, error)

// CurlCommand - Explicit shell parsing, return response
func CurlCommand(ctx, string) (*http.Response, error)

// CurlArgs - Explicit variadic, return response
func CurlArgs(ctx, ...string) (*http.Response, error)
```

### Convenience Functions (Auto-read body)

```go
// CurlString - Return body as string + response
func CurlString(ctx, ...string) (string, *http.Response, error)

// CurlBytes - Return body as []byte + response
func CurlBytes(ctx, ...string) ([]byte, *http.Response, error)

// CurlJSON - Decode JSON to struct + response
func CurlJSON(ctx, interface{}, ...string) (*http.Response, error)

// CurlDownload - Download directly to file + response
func CurlDownload(ctx, filepath, ...string) (int64, *http.Response, error)

// CurlStream - Explicit streaming (body not auto-closed)
func CurlStream(ctx, ...string) (*http.Response, error)
```

**All functions have Command and Args variants**: `CurlStringCommand()`, `CurlBytesArgs()`, etc.

**Why This Design?**

1. ✅ **Efficiency**: No forced buffering, user controls memory
2. ✅ **Flexibility**: Access headers, status, cookies via response
3. ✅ **Streaming**: Can process large files without loading to memory
4. ✅ **Type Safety**: JSON decoding with compile-time checks
5. ✅ **Convenience**: Helpers for common cases (string, JSON, download)

## Curl Parity Testing Strategy

**CRITICAL PRINCIPLE**: Every test MUST compare gocurl output against real curl command.

### Parity Test Framework

```go
// RunParityTest - Execute both gocurl and real curl, compare results
func RunParityTest(t *testing.T, test ParityTest) {
    // 1. Execute with real curl
    curlOutput, curlErr := executeRealCurl(command)

    // 2. Execute with gocurl
    gocurlOutput, gocurlErr := executeGoCurl(ctx, command)

    // 3. Compare results (MUST be identical)
    compareResults(t, curlOutput, gocurlOutput)
}
```

### Test Categories

1. **Core Parity Tests** (~50 tests)
   - Simple GET, POST, headers, auth, redirects
   - All basic curl flags
   - Multi-line commands
   - Environment variables

2. **Browser DevTools Tests** (~20 tests)
   - Chrome "Copy as cURL"
   - Firefox "Copy as cURL"
   - Real API endpoints (GitHub, Stripe, etc.)

3. **API Documentation Tests** (~30 tests)
   - Commands copied from real API docs
   - Stripe, GitHub, AWS, etc.
   - Multi-line with backslashes

4. **Edge Case Tests** (~20 tests)
   - Binary data, unicode, large files
   - Empty responses, chunked encoding
   - Compressed responses

**Success Criteria**: 100% parity for core features, 95%+ for advanced features.

**See**: `wip-notes/CURL_PARITY_TESTING.md` for complete strategy.

## Core Principle: ZERO DIVERGENCE (Implementation)

**Critical Requirement**: CLI and library MUST use the EXACT same code path.

### Architecture: Shared Code Path

```
┌─────────────────┐
│   CLI Input     │
│ os.Args[1:]     │
└────────┬────────┘
         │
         ↓
┌─────────────────┐      ┌──────────────────┐
│  Tokenizer      │ ←──→ │  Library Input   │
│  (shared)       │      │  (same code)     │
└────────┬────────┘      └────────┬─────────┘
         │                        │
         ↓                        ↓
┌──────────────────────────────────┐
│     Parser (shared)              │
│  convertTokensToRequestOptions   │
└────────────────┬─────────────────┘
                 │
                 ↓
┌──────────────────────────────────┐
│     Process (shared)             │
│   HTTP execution engine          │
└────────────────┬─────────────────┘
                 │
                 ↓
         ┌───────┴────────┐
         │                │
         ↓                ↓
    ┌────────┐      ┌─────────┐
    │  CLI   │      │ Library │
    │ stdout │      │ return  │
    └────────┘      └─────────┘
```

**Key Insight**: The ONLY difference is input source (os.Args vs function param) and output destination (stdout vs return value).

### Environment Variable Behavior

| Feature | CLI | Library (Curl) | Library (CurlWithVars) |
|---------|-----|----------------|------------------------|
| **Input** | `os.Args[1:]` | Function params | Function params |
| **$VAR expansion** | ✅ Auto from env | ✅ Auto from env | ✅ From explicit map |
| **Behavior** | Matches curl | **IDENTICAL to CLI!** | Controlled expansion |
| **Testing** | ⚠️ Env pollution | ⚠️ Env pollution | ✅ Isolated |
| **Security** | ✅ No cmd exec | ✅ No cmd exec | ✅ Explicit control |
| **Use Case** | Shell usage | **Production (matches CLI!)** | Unit tests/security |
| **Parity** | Reference | **GUARANTEED MATCH** | Different (by design) |

**Key Decision: UNIFIED Expansion**

✅ **CLI**: Auto-expand from environment (matches curl)
✅ **Curl()**: Auto-expand from environment (**MATCHES CLI - core objective!**)
✅ **CurlWithVars()**: Expand from map (for testing/security-critical code)

**This ensures:**

1. 🎯 **Core Objective Met**: Test with CLI → copy to code → identical behavior
2. 🔒 **Security Preserved**: No shell command execution (only `$VAR` expansion)
3. 🧪 **Testing Flexibility**: Can use `CurlWithVars()` for isolated tests
4. 📝 **Zero Translation**: Same syntax, same behavior, everywhere


## Implementation Strategy

### Phase 1: Core CLI (Military-Grade Foundation)

**Files to Create:**
1. `cmd/gocurl/main.go` - Entry point
2. `cmd/gocurl/cli.go` - CLI logic (shared with library)
3. `cmd/gocurl/output.go` - Output formatting
4. `cmd/gocurl/cli_test.go` - Comprehensive tests

**Design Principles:**
- ✅ Use EXACT same parser/converter/processor as library
- ✅ Environment variables for CLI ($VAR), map for library
- ✅ Exit codes match curl conventions
- ✅ Error messages match curl format
- ✅ No memory leaks (defer cleanup, proper resource management)
- ✅ Thread-safe (though CLI is single-threaded by nature)
- ✅ Comprehensive error handling

### Phase 2: Output Formatting (Curl Parity)

**Output Modes:**
1. **Default**: Response body to stdout (like curl)
2. **Verbose (-v)**: Connection info + headers + body
3. **Include (-i)**: Headers + body
4. **Silent (-s)**: Suppress progress, only output body
5. **Output (-o file)**: Write to file
6. **Write-out (-w format)**: Custom output format

### Phase 3: Testing Strategy

**Test Categories:**
1. **Unit Tests**: Each function isolated
2. **Integration Tests**: Full CLI execution
3. **Roundtrip Tests**: CLI output → Go code → same result
4. **Parity Tests**: gocurl vs curl exact comparison
5. **Race Tests**: Concurrent CLI invocations (if needed)
6. **Memory Tests**: No leaks, efficient allocation
7. **Error Tests**: Every error path covered

## Detailed Implementation

### 1. Main Entry Point

**File**: `cmd/gocurl/main.go`

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/maniartech/gocurl"
)

func main() {
    // Exit with proper code
    os.Exit(run())
}

func run() int {
    // Parse command line args (os.Args[1:])
    ctx := context.Background()

    // Execute using SHARED code path
    result, err := executeCLI(ctx, os.Args[1:])

    if err != nil {
        // Error output to stderr (like curl)
        fmt.Fprintf(os.Stderr, "gocurl: %v\n", err)
        return getExitCode(err)
    }

    // Success output to stdout
    fmt.Print(result)
    return 0
}

// executeCLI - uses EXACT same code as library
func executeCLI(ctx context.Context, args []string) (string, error) {
    // 1. Tokenize (shared)
    tokens, err := gocurl.TokenizeArgs(args)
    if err != nil {
        return "", err
    }

    // 2. Expand environment variables (SHARED WITH LIBRARY!)
    // Note: gocurl.Curl() does the same expansion internally
    tokens = gocurl.ExpandEnvInTokens(tokens)

    // 3. Convert to RequestOptions (shared)
    opts, err := gocurl.ConvertTokensToRequestOptions(tokens)
    if err != nil {
        return "", err
    }

    // 4. Execute HTTP request (shared) - Process returns (*http.Response, error)
    resp, err := gocurl.Process(ctx, opts)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    // 5. Read response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }

    // 6. Format output (CLI-specific)
    return formatOutput(resp, string(body), opts)
}
```

### 2. Environment Variable Expansion

**Strategy: UNIFIED - Both CLI and Library Support $VAR**

#### CLI Behavior (Matches curl)

```bash
# Set environment variable
$ export API_TOKEN=secret123

# Use in command - automatic expansion
$ gocurl -H "Authorization: Bearer $API_TOKEN" https://api.example.com
# ✅ Expands to: "Authorization: Bearer secret123"

# Both $VAR and ${VAR} syntax supported (bash-compatible)
$ gocurl -H "X-Token: ${API_TOKEN}" https://api.example.com
```

#### Library Behavior (UNIFIED - Same as CLI!)

```go
// DEFAULT: Auto-expands from environment (MATCHES CLI!)
os.Setenv("API_TOKEN", "secret123")
resp, err := gocurl.Curl(ctx,
    "-H", "Authorization: Bearer $API_TOKEN",
    "https://api.example.com",
)
defer resp.Body.Close()
// ✅ Expands to: "Authorization: Bearer secret123"
// ✅ IDENTICAL behavior to CLI - test in CLI, copy to code!
// ✅ Returns *http.Response - full access to headers, status, cookies

// Convenience: Auto-read body to string
os.Setenv("API_TOKEN", "secret123")
bodyStr, resp, err := gocurl.CurlString(ctx,
    "-H", "Authorization: Bearer $API_TOKEN",
    "https://api.example.com",
)
// ✅ Body already read, response still available

// Alternative: Explicit variable map (for testing/security)
vars := map[string]string{"API_TOKEN": "secret123"}
resp, err := gocurl.CurlWithVars(ctx, vars,
    "-H", "Authorization: Bearer $API_TOKEN",
    "https://api.example.com",
)
defer resp.Body.Close()
// ✅ Expands from explicit map (testable, no env pollution)

// Alternative: No expansion (escape with $$)
resp, err := gocurl.Curl(ctx,
    "-H", "Authorization: Bearer $$API_TOKEN",  // Literal $API_TOKEN
    "https://api.example.com",
)
defer resp.Body.Close()
// ✅ Double $$ escapes the variable (becomes single $)
```

**Why This Unified Design?**

1. **Core Objective**: "Test with CLI first" - CLI and library MUST behave identically
2. **Zero Translation**: Copy command from CLI to Go code - it just works
3. **User Experience**: Natural, intuitive - same syntax everywhere
4. **Flexibility**: Can opt-in to explicit vars for testing (CurlWithVars)

**Security Considerations:**

- ✅ Only expands `$VAR` and `${VAR}` (no shell command execution)
- ✅ Only expands STRING tokens (not flags - prevents injection)
- ✅ Can escape with `$$` → single `$` (like Makefiles)
- ✅ For security-critical code, use `CurlWithVars()` with explicit map

**Implementation:**

```go
// preprocessMultilineCommand - Handles multi-line curl commands
// Supports:
// - Backslash line continuations (\)
// - Newlines as whitespace
// - Comment lines (#)
// - Automatic 'curl' prefix removal
// - Leading/trailing whitespace trimming
func preprocessMultilineCommand(cmd string) string {
    lines := strings.Split(cmd, "\n")
    var processed []string

    for i, line := range lines {
        // Trim leading/trailing whitespace
        line = strings.TrimSpace(line)

        // Skip empty lines
        if line == "" {
            continue
        }

        // Skip comment lines
        if strings.HasPrefix(line, "#") {
            continue
        }

        // Remove 'curl' prefix from first non-empty line
        if len(processed) == 0 && strings.HasPrefix(line, "curl ") {
            line = strings.TrimPrefix(line, "curl ")
            line = strings.TrimSpace(line)
        }

        // Handle backslash line continuation
        if strings.HasSuffix(line, "\\") {
            // Remove trailing backslash
            line = strings.TrimSuffix(line, "\\")
            line = strings.TrimSpace(line)
            processed = append(processed, line)
        } else {
            processed = append(processed, line)

            // If this is not the last line, add a space separator
            if i < len(lines)-1 {
                // Check if next line is continuation
                if i+1 < len(lines) {
                    nextLine := strings.TrimSpace(lines[i+1])
                    if nextLine != "" && !strings.HasPrefix(nextLine, "#") {
                        // Add space between lines
                    }
                }
            }
        }
    }

    // Join all processed lines with spaces
    return strings.Join(processed, " ")
}

// SHARED: Environment expansion (used by both CLI and Library)
func expandEnvInTokens(tokens []tokenizer.Token) []tokenizer.Token {
    result := make([]tokenizer.Token, len(tokens))

    for i, token := range tokens {
        result[i] = token

        // Only expand STRING tokens (not flags)
        if token.Type == tokenizer.STRING {
            // os.ExpandEnv handles both $VAR and ${VAR}
            result[i].Value = os.ExpandEnv(token.Value)
        }
    }

    return result
}

// Alternative: Expand from explicit map (for testing)
func expandVarsInTokens(tokens []tokenizer.Token, vars map[string]string) []tokenizer.Token {
    result := make([]tokenizer.Token, len(tokens))

    for i, token := range tokens {
        result[i] = token

        if token.Type == tokenizer.STRING {
            // Custom expansion from map (not environment)
            result[i].Value = expandVars(token.Value, vars)
        }
    }

    return result
}
```
            // Custom expansion from map (not environment)
            result[i].Value = expandVars(token.Value, vars)
        }
    }

    return result
}

// Helper: Expand $VAR and ${VAR} from map
func expandVars(s string, vars map[string]string) string {
    return os.Expand(s, func(key string) string {
        if val, ok := vars[key]; ok {
            return val
        }
        return "$" + key  // Keep unexpanded if not in map
    })
}
```

**Updated Library APIs (Three-Function Design):**

```go
// api.go

// ============================================================================
// PRIMARY API - Auto-detection (Use this for 90% of cases)
// ============================================================================

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
//   resp, err := Curl(ctx, "-H", "X-Token: abc", "https://example.com")
//   defer resp.Body.Close()
//
//   // Shell command (auto-detected)
//   resp, err := Curl(ctx, `curl -H 'X-Token: abc' https://example.com`)
//   defer resp.Body.Close()
//
//   // Multi-line (auto-detected)
//   resp, err := Curl(ctx, `
//     curl -X POST https://api.example.com \
//       -H 'Content-Type: application/json' \
//       -d '{"key":"value"}'
//   `)
//   defer resp.Body.Close()
//
//   // Access response details
//   fmt.Println(resp.StatusCode)
//   fmt.Println(resp.Header.Get("Content-Type"))
func Curl(ctx context.Context, command ...string) (*http.Response, error) {
    if len(command) == 0 {
        return nil, fmt.Errorf("no command provided")
    }

    if len(command) == 1 {
        // Single string → treat as shell command
        return CurlCommand(ctx, command[0])
    }

    // Multiple args → treat as variadic
    return CurlArgs(ctx, command...)
}

// ============================================================================
// EXPLICIT SHELL COMMAND API - For copy/paste from curl
// ============================================================================

// CurlCommand executes an HTTP request from a shell command string.
//
// ALWAYS parses the input as a shell command. Use this when:
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
//   resp, err := CurlCommand(ctx, cmd)
//   defer resp.Body.Close()
//
//   // Browser DevTools
//   cmd := `curl 'https://api.example.com/graphql' \
//             -X POST \
//             -H 'authorization: Bearer ...' \
//             --data-raw '{"query":"..."}'`
//   resp, err := CurlCommand(ctx, cmd)
//   defer resp.Body.Close()
//
//   // Read body
//   body, _ := io.ReadAll(resp.Body)
func CurlCommand(ctx context.Context, shellCommand string) (*http.Response, error) {
    // 1. Preprocess multi-line (backslashes, comments, curl prefix)
    cmdStr := preprocessMultilineCommand(shellCommand)

    // 2. Tokenize as shell command
    tokens, err := tokenizer.Tokenize(cmdStr)
    if err != nil {
        return nil, fmt.Errorf("failed to parse command: %w", err)
    }

    // 3. Expand environment variables
    tokens = expandEnvInTokens(tokens)

    // 4. Convert to options
    opts, err := ConvertTokensToRequestOptions(tokens)
    if err != nil {
        return nil, err
    }

    // 5. Execute - Process now returns (*http.Response, error)
    return Process(ctx, opts)
}

// ============================================================================
// EXPLICIT VARIADIC API - For programmatic building
// ============================================================================

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
//   resp, err := CurlArgs(ctx, url)
//   defer resp.Body.Close()
//
//   // With headers
//   resp, err := CurlArgs(ctx,
//       "-H", "Authorization: Bearer " + token,
//       "-H", "Content-Type: application/json",
//       url,
//   )
//   defer resp.Body.Close()
//
//   // Read response
//   body, _ := io.ReadAll(resp.Body)
func CurlArgs(ctx context.Context, args ...string) (*http.Response, error) {
    if len(args) == 0 {
        return nil, fmt.Errorf("no arguments provided")
    }

    // 1. Direct tokenization (no parsing)
    tokens, err := TokenizeArgs(args)
    if err != nil {
        return nil, fmt.Errorf("failed to tokenize arguments: %w", err)
    }

    // 2. Expand environment variables
    tokens = expandEnvInTokens(tokens)

    // 3. Convert to options
    opts, err := ConvertTokensToRequestOptions(tokens)
    if err != nil {
        return nil, err
    }

    // 4. Execute - Process now returns (*http.Response, error)
    return Process(ctx, opts)
}

// ============================================================================
// WITH EXPLICIT VARIABLE MAPS (for testing/security)
// ============================================================================

// CurlWithVars - Auto-detect with explicit variable map
func CurlWithVars(ctx context.Context, vars map[string]string, command ...string) (*http.Response, error) {
    if len(command) == 0 {
        return nil, fmt.Errorf("no command provided")
    }

    if len(command) == 1 {
        return CurlCommandWithVars(ctx, vars, command[0])
    }

    return CurlArgsWithVars(ctx, vars, command...)
}

// CurlCommandWithVars - Shell command with explicit variable map
func CurlCommandWithVars(ctx context.Context, vars map[string]string, shellCommand string) (*http.Response, error) {
    // Preprocess
    cmdStr := preprocessMultilineCommand(shellCommand)

    // Tokenize
    tokens, err := tokenizer.Tokenize(cmdStr)
    if err != nil {
        return nil, fmt.Errorf("failed to parse command: %w", err)
    }

    // Expand from explicit map (NOT environment!)
    tokens = expandVarsInTokens(tokens, vars)

    // Convert and execute
    opts, err := ConvertTokensToRequestOptions(tokens)
    if err != nil {
        return nil, err
    }

    return Process(ctx, opts)
}

// CurlArgsWithVars - Variadic with explicit variable map
func CurlArgsWithVars(ctx context.Context, vars map[string]string, args ...string) (*http.Response, error) {
    if len(args) == 0 {
        return nil, fmt.Errorf("no arguments provided")
    }

    // Tokenize
    tokens, err := TokenizeArgs(args)
    if err != nil {
        return nil, fmt.Errorf("failed to tokenize arguments: %w", err)
    }

    // Expand from explicit map
    tokens = expandVarsInTokens(tokens, vars)

    // Convert and execute
    opts, err := ConvertTokensToRequestOptions(tokens)
    if err != nil {
        return nil, err
    }

    return Process(ctx, opts)
}
```

**API Summary:**

| Function | Returns | When to Use |
|----------|---------|-------------|
| `Curl()` | `(*http.Response, error)` | Full control - 90% of cases |
| `CurlCommand()` | `(*http.Response, error)` | Copy/paste from curl, API docs, browser |
| `CurlArgs()` | `(*http.Response, error)` | Programmatic building, performance |
| `CurlString()` | `(string, *http.Response, error)` | Text response, want body as string |
| `CurlBytes()` | `([]byte, *http.Response, error)` | Binary data, need bytes |
| `CurlJSON()` | `(*http.Response, error)` | JSON response, auto-decode to struct |
| `CurlDownload()` | `(int64, *http.Response, error)` | File download, save to disk |
| `CurlWithVars()` | `(*http.Response, error)` | Testing with explicit variable map |

**All functions have Command, Args, and WithVars variants** for maximum flexibility.

**Usage Examples:**

```go
// Example 1: Variadic syntax (Go-style) - Full control
resp, err := gocurl.Curl(ctx,
    "-X", "POST",
    "-H", "Content-Type: application/json",
    "-H", "Authorization: Bearer $TOKEN",
    "-d", `{"key":"value"}`,
    "https://api.example.com",
)
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)

// Example 2: Single line (copy from shell) - Full control
resp, err := gocurl.Curl(ctx,
    `-X POST -H 'Content-Type: application/json' -d '{"key":"value"}' https://api.example.com`,
)
defer resp.Body.Close()
body, _ = io.ReadAll(resp.Body)

// Example 3: Multi-line with backslashes (API docs style!)
resp, err := gocurl.Curl(ctx, `
curl -X POST https://api.example.com/v1/resource \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d amount=2000 \
  -d currency=usd
`)
defer resp.Body.Close()
body, _ = io.ReadAll(resp.Body)

// Example 4: Multi-line without backslashes (still works!)
resp, err := gocurl.Curl(ctx, `
curl -X POST https://api.example.com/v1/resource
  -H "Authorization: Bearer $TOKEN"
  -H "Content-Type: application/json"
  -d amount=2000
  -d currency=usd
`)
defer resp.Body.Close()
body, _ = io.ReadAll(resp.Body)

// Example 5: Copy from browser DevTools (Chrome/Firefox)
resp, err := gocurl.Curl(ctx, `
curl 'https://api.example.com/search' \
  -X POST \
  -H 'authority: api.example.com' \
  -H 'accept: application/json' \
  -H 'authorization: Bearer eyJhbGc...' \
  --data-raw '{"query":"test"}'
`)
defer resp.Body.Close()
body, _ = io.ReadAll(resp.Body)

// Example 6: With comments (documentation style)
resp, err := gocurl.Curl(ctx, `
# Create a new charge
curl -X POST https://api.stripe.com/v1/charges \
  -u $STRIPE_KEY: \
  -d amount=2000 \  # Amount in cents
  -d currency=usd \
  -d source=tok_visa
`)
defer resp.Body.Close()
body, _ = io.ReadAll(resp.Body)

// Example 7: Convenience - auto-read to string
bodyStr, resp, err := gocurl.CurlString(ctx,
    "-X", "POST",
    "-H", "Content-Type: application/json",
    "-d", `{"key":"value"}`,
    "https://api.example.com",
)
// bodyStr is ready, resp has headers/status

// Example 8: Convenience - auto-decode JSON
var result APIResponse
resp, err := gocurl.CurlJSON(ctx, &result, `
curl -X GET https://api.example.com/data \
  -H "Authorization: Bearer $TOKEN"
`)
// result struct is populated

// ✅ ALL examples work identically for equivalent requests!
```### 3. Output Formatting (Curl-Compatible)

```go
func formatOutput(resp *http.Response, body string, opts *options.RequestOptions) string {
    var out strings.Builder

    // Include headers if requested (-i flag)
    if opts.IncludeHeaders {
        // Format: "HTTP/1.1 200 OK"
        fmt.Fprintf(&out, "%s %s\n", resp.Proto, resp.Status)

        // Write headers
        for name, values := range resp.Header {
            for _, value := range values {
                fmt.Fprintf(&out, "%s: %s\n", name, value)
            }
        }
        out.WriteString("\n")
    }

    // Write body (unless silent and success)
    if !opts.Silent || resp.StatusCode >= 400 {
        out.WriteString(body)
    }

    return out.String()
}
```

### 4. Exit Codes (Curl Parity)

```go
// getExitCode - match curl's exit code conventions
func getExitCode(err error) int {
    if err == nil {
        return 0
    }

    // Match curl exit codes
    // https://everything.curl.dev/usingcurl/returns

    switch {
    case isParseError(err):
        return 3  // URL malformed
    case isConnectionError(err):
        return 7  // Failed to connect to host
    case isAuthError(err):
        return 67 // Access denied
    case isTimeoutError(err):
        return 28 // Operation timeout
    default:
        return 1  // General error
    }
}
```

## Testing Strategy (Military-Grade + Curl Parity)

**CRITICAL**: Every test MUST verify against real curl behavior.

### Test File Structure

```
cmd/gocurl/
├── main.go              # Entry point
├── cli.go               # CLI execution logic
├── output.go            # Output formatting
├── errors.go            # Error handling
├── cli_test.go          # Unit tests
├── integration_test.go  # Full CLI tests
├── parity_test.go       # ⭐ Curl parity tests (CRITICAL!)
├── roundtrip_test.go    # CLI→Go parity tests
└── testdata/
    ├── responses/       # Mock HTTP responses
    ├── expected/        # Expected outputs
    └── curl_commands/   # Real curl commands for parity testing
```

### Test Categories (Priority Order)

#### 1. Curl Parity Tests (parity_test.go) ⭐ HIGHEST PRIORITY

**Every test compares gocurl vs real curl command.**

```go
func TestCurlParity(t *testing.T) {
    tests := []ParityTest{
        {
            Name:    "Simple GET",
            Command: "curl {{URL}}",
        },
        {
            Name:    "POST with JSON",
            Command: `curl -X POST -H 'Content-Type: application/json' -d '{"key":"value"}' {{URL}}`,
        },
        {
            Name:    "Multi-line with backslashes",
            Command: `curl -X POST {{URL}} \
                -H 'Authorization: Bearer token' \
                -d '{"test": true}'`,
        },
        {
            Name:    "Browser DevTools copy",
            Command: `curl 'https://api.example.com/search' \
                -X POST \
                -H 'authorization: Bearer ...' \
                --data-raw '{"query":"test"}'`,
        },
        {
            Name:    "Environment variables",
            Command: "curl -H 'Authorization: Bearer $TOKEN' {{URL}}",
            Setup: func(srv *httptest.Server) error {
                os.Setenv("TOKEN", "test123")
                return nil
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Execute BOTH curl and gocurl
            curlResult := executeRealCurl(tt.Command)
            gocurlResult := executeGoCurl(ctx, tt.Command)

            // MUST match exactly
            assert.Equal(t, curlResult.StatusCode, gocurlResult.StatusCode)
            assert.Equal(t, curlResult.Body, gocurlResult.Body)
        })
    }
}
```

**Parity Test Suites**:

1. **Core Features** (~50 tests): GET, POST, headers, auth, methods
2. **Browser DevTools** (~20 tests): Chrome/Firefox "Copy as cURL"
3. **API Documentation** (~30 tests): Real commands from Stripe, GitHub, AWS docs
4. **Edge Cases** (~20 tests): Binary, unicode, large files, compression
5. **Environment Variables** (~15 tests): $VAR expansion matching curl

**Total**: ~135 parity tests ensuring 100% curl compatibility

#### 2. Unit Tests (cli_test.go)

```go
// Test multi-line command preprocessing
func TestPreprocessMultilineCommand(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            "Backslash continuations",
            "curl -X POST \\\n  -H 'X-Token: abc' \\\n  https://example.com",
            "-X POST -H 'X-Token: abc' https://example.com",
        },
        {
            "Without backslashes",
            "curl -X POST\n  -H 'X-Token: abc'\n  https://example.com",
            "-X POST -H 'X-Token: abc' https://example.com",
        },
        {
            "With comments",
            "# Create resource\ncurl -X POST \\\n  -d key=value \\\n  https://example.com",
            "-X POST -d key=value https://example.com",
        },
        {
            "No curl prefix",
            "-X POST \\\n  -H 'X-Token: abc' \\\n  https://example.com",
            "-X POST -H 'X-Token: abc' https://example.com",
        },
        {
            "Empty lines",
            "curl -X POST\n\n  -H 'X-Token: abc'\n\n  https://example.com",
            "-X POST -H 'X-Token: abc' https://example.com",
        },
        {
            "Mixed indentation",
            "curl -X POST \\\n    -H 'X-Token: abc' \\\n  https://example.com",
            "-X POST -H 'X-Token: abc' https://example.com",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := preprocessMultilineCommand(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}

// Test environment variable expansion
func TestExpandEnvInTokens(t *testing.T) {
    os.Setenv("API_KEY", "secret123")
    defer os.Unsetenv("API_KEY")

    tokens := []tokenizer.Token{
        {Type: tokenizer.STRING, Value: "https://api.example.com"},
        {Type: tokenizer.FLAG, Value: "-H"},
        {Type: tokenizer.STRING, Value: "Authorization: Bearer $API_KEY"},
    }

    result := expandEnvInTokens(tokens)

    assert.Equal(t, "Authorization: Bearer secret123", result[2].Value)
}

// Test output formatting
func TestFormatOutput(t *testing.T) {
    resp := &http.Response{
        Status:     "200 OK",
        Proto:      "HTTP/1.1",
        StatusCode: 200,
        Header:     http.Header{"Content-Type": []string{"text/plain"}},
    }

    opts := &options.RequestOptions{
        IncludeHeaders: true,
    }

    output := formatOutput(resp, "Hello, World!", opts)

    assert.Contains(t, output, "HTTP/1.1 200 OK")
    assert.Contains(t, output, "Content-Type: text/plain")
    assert.Contains(t, output, "Hello, World!")
}

// Test exit codes
func TestGetExitCode(t *testing.T) {
    tests := []struct {
        name     string
        err      error
        expected int
    }{
        {"nil error", nil, 0},
        {"parse error", fmt.Errorf("parse: invalid URL"), 3},
        {"connection error", fmt.Errorf("connection refused"), 7},
        {"timeout", fmt.Errorf("timeout exceeded"), 28},
        {"general error", fmt.Errorf("unknown error"), 1},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            code := getExitCode(tt.err)
            assert.Equal(t, tt.expected, code)
        })
    }
}
```

#### 2. Integration Tests (integration_test.go)

```go
// Test full CLI execution
func TestCLI_BasicGET(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, "Success!")
    }))
    defer server.Close()

    // Run CLI
    args := []string{"gocurl", server.URL}
    output, exitCode := runCLI(args)

    assert.Equal(t, 0, exitCode)
    assert.Equal(t, "Success!", output)
}

// Test multi-line command with backslashes
func TestCLI_MultiLineWithBackslashes(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "POST", r.Method)
        assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
        assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))

        body, _ := io.ReadAll(r.Body)
        assert.Equal(t, `{"key":"value"}`, string(body))

        fmt.Fprint(w, "OK")
    }))
    defer server.Close()

    // Multi-line command (like API documentation)
    command := fmt.Sprintf(`
curl -X POST %s \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer token123' \
  -d '{"key":"value"}'
`, server.URL)

    output, exitCode := runCLIWithString(command)

    assert.Equal(t, 0, exitCode)
    assert.Equal(t, "OK", output)
}

// Test multi-line command without backslashes
func TestCLI_MultiLineWithoutBackslashes(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "POST", r.Method)
        fmt.Fprint(w, "OK")
    }))
    defer server.Close()

    // Multi-line without backslashes (still works!)
    command := fmt.Sprintf(`
curl -X POST %s
  -H 'Content-Type: application/json'
  -H 'Authorization: Bearer token123'
  -d '{"key":"value"}'
`, server.URL)

    output, exitCode := runCLIWithString(command)

    assert.Equal(t, 0, exitCode)
    assert.Equal(t, "OK", output)
}

// Test with comments (documentation style)
func TestCLI_WithComments(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, "OK")
    }))
    defer server.Close()

    command := fmt.Sprintf(`
# Create a new resource
curl -X POST %s \
  # Set content type
  -H 'Content-Type: application/json' \
  # Add authentication
  -H 'Authorization: Bearer token123' \
  # Send payload
  -d '{"key":"value"}'
`, server.URL)

    output, exitCode := runCLIWithString(command)

    assert.Equal(t, 0, exitCode)
    assert.Equal(t, "OK", output)
}

// Test with environment variables
func TestCLI_WithEnvVars(t *testing.T) {
    os.Setenv("TOKEN", "abc123")
    defer os.Unsetenv("TOKEN")

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        auth := r.Header.Get("Authorization")
        assert.Equal(t, "Bearer abc123", auth)
        fmt.Fprint(w, "Authenticated")
    }))
    defer server.Close()

    args := []string{
        "gocurl",
        "-H", "Authorization: Bearer $TOKEN",
        server.URL,
    }

    output, exitCode := runCLI(args)

    assert.Equal(t, 0, exitCode)
    assert.Equal(t, "Authenticated", output)
}

// Test browser DevTools copy/paste (realistic scenario)
func TestCLI_BrowserDevToolsCopy(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify all headers that Chrome includes
        assert.Equal(t, "POST", r.Method)
        assert.Equal(t, "api.example.com", r.Header.Get("authority"))
        assert.Equal(t, "application/json", r.Header.Get("accept"))

        body, _ := io.ReadAll(r.Body)
        assert.Equal(t, `{"query":"test"}`, string(body))

        fmt.Fprint(w, `{"results":[]}`)
    }))
    defer server.Close()

    // Exact format from Chrome DevTools "Copy as cURL"
    command := fmt.Sprintf(`
curl '%s' \
  -X POST \
  -H 'authority: api.example.com' \
  -H 'accept: application/json' \
  -H 'content-type: application/json' \
  --data-raw '{"query":"test"}'
`, server.URL)

    output, exitCode := runCLIWithString(command)

    assert.Equal(t, 0, exitCode)
    assert.Contains(t, output, `"results"`)
}
```

#### 3. Roundtrip Tests (roundtrip_test.go)

**CRITICAL**: Prove CLI and library produce identical results

```go
// TestRoundtrip_CLItoLibrary - Core guarantee test
func TestRoundtrip_CLItoLibrary(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Echo request details
        fmt.Fprintf(w, "Method: %s\n", r.Method)
        fmt.Fprintf(w, "Header: %s\n", r.Header.Get("X-Custom"))
        body, _ := io.ReadAll(r.Body)
        fmt.Fprintf(w, "Body: %s\n", string(body))
    }))
    defer server.Close()

    // Test cases that MUST work identically
    tests := []struct {
        name string
        args []string
        env  map[string]string  // Environment to set
    }{
        {
            "Simple GET",
            []string{"gocurl", server.URL},
            nil,
        },
        {
            "POST with data",
            []string{"gocurl", "-X", "POST", "-d", "key=value", server.URL},
            nil,
        },
        {
            "Environment variable expansion",
            []string{"gocurl", "-H", "X-Token: $API_TOKEN", server.URL},
            map[string]string{"API_TOKEN": "secret123"},
        },
        {
            "Multiple env vars",
            []string{
                "gocurl",
                "-H", "Authorization: Bearer $TOKEN",
                "-H", "X-API-Key: ${API_KEY}",
                server.URL,
            },
            map[string]string{
                "TOKEN":   "jwt123",
                "API_KEY": "key456",
            },
        },
        {
            "Multiple flags",
            []string{
                "gocurl",
                "-X", "POST",
                "-H", "Content-Type: application/json",
                "-H", "X-Custom: test",
                "-d", `{"key":"value"}`,
                server.URL,
            },
            nil,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Set environment variables
            for k, v := range tt.env {
                os.Setenv(k, v)
                defer os.Unsetenv(k)
            }

            // 1. Execute via CLI
            cliOutput, cliExitCode := runCLI(tt.args)

            // 2. Execute via Library (same args, SAME ENVIRONMENT!)
            ctx := context.Background()

            // Test with string convenience function
            libBody, libResp, libErr := gocurl.CurlString(ctx, tt.args[1:]...)

            // 3. MUST match exactly
            assert.Equal(t, 0, cliExitCode, "CLI should succeed")
            assert.NoError(t, libErr, "Library should succeed")
            assert.Equal(t, cliOutput, libBody, "CLI and library MUST produce identical output")
            assert.Equal(t, 200, libResp.StatusCode, "Should get 200 OK")

            // Cleanup environment
            for k := range tt.env {
                os.Unsetenv(k)
            }
        })
    }
}

// TestRoundtrip_ResponseAPI - Verify response API returns full response
func TestRoundtrip_ResponseAPI(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Custom-Header", "test-value")
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(200)
        fmt.Fprint(w, `{"status":"ok"}`)
    }))
    defer server.Close()

    ctx := context.Background()

    // Test core API (returns *http.Response)
    resp, err := gocurl.Curl(ctx, server.URL)
    assert.NoError(t, err)
    assert.NotNil(t, resp)
    assert.Equal(t, 200, resp.StatusCode)
    assert.Equal(t, "test-value", resp.Header.Get("X-Custom-Header"))
    assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

    // Read body manually
    body, _ := io.ReadAll(resp.Body)
    resp.Body.Close()
    assert.Equal(t, `{"status":"ok"}`, string(body))

    // Test convenience API (auto-reads body)
    bodyStr, resp2, err := gocurl.CurlString(ctx, server.URL)
    assert.NoError(t, err)
    assert.Equal(t, `{"status":"ok"}`, bodyStr)
    assert.Equal(t, 200, resp2.StatusCode)
    assert.Equal(t, "test-value", resp2.Header.Get("X-Custom-Header"))

    // Test bytes API
    bodyBytes, resp3, err := gocurl.CurlBytes(ctx, server.URL)
    assert.NoError(t, err)
    assert.Equal(t, []byte(`{"status":"ok"}`), bodyBytes)
    assert.Equal(t, 200, resp3.StatusCode)

    // Test JSON API
    var result struct {
        Status string `json:"status"`
    }
    resp4, err := gocurl.CurlJSON(ctx, &result, server.URL)
    assert.NoError(t, err)
    assert.Equal(t, "ok", result.Status)
    assert.Equal(t, 200, resp4.StatusCode)
}

// TestRoundtrip_WithVars - Verify CurlWithVars for testing
func TestRoundtrip_WithVars(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Token: %s\n", r.Header.Get("X-Token"))
    }))
    defer server.Close()

    // Test that CurlWithVars uses explicit map (not environment)
    os.Setenv("TOKEN", "wrong_value")
    defer os.Unsetenv("TOKEN")

    vars := map[string]string{"TOKEN": "correct_value"}
    ctx := context.Background()

    // Test with string convenience
    body, resp, err := gocurl.CurlStringWithVars(ctx, vars,
        "-H", "X-Token: $TOKEN",
        server.URL,
    )

    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
    assert.Contains(t, body, "Token: correct_value")  // Should use map, not env
    assert.NotContains(t, body, "wrong_value")        // Should NOT use environment
}
```

#### 4. Memory Leak Tests

```go
// Test no memory leaks
func TestCLI_NoMemoryLeaks(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping memory leak test in short mode")
    }

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, strings.Repeat("x", 1024)) // 1KB response
    }))
    defer server.Close()

    // Get baseline memory
    runtime.GC()
    var m1 runtime.MemStats
    runtime.ReadMemStats(&m1)

    // Run CLI 1000 times
    args := []string{"gocurl", server.URL}
    for i := 0; i < 1000; i++ {
        runCLI(args)
    }

    // Force GC and check memory
    runtime.GC()
    var m2 runtime.MemStats
    runtime.ReadMemStats(&m2)

    // Memory growth should be minimal (< 1MB)
    growth := m2.Alloc - m1.Alloc
    assert.Less(t, growth, uint64(1024*1024), "Memory leak detected: %d bytes leaked", growth)
}
```

#### 5. Concurrent Safety Tests

```go
// Test concurrent CLI invocations (if library is used concurrently)
func TestCLI_ConcurrentSafety(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, "OK")
    }))
    defer server.Close()

    // Run 100 concurrent CLI calls
    var wg sync.WaitGroup
    args := []string{"gocurl", server.URL}

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            output, code := runCLI(args)
            assert.Equal(t, 0, code)
            assert.Equal(t, "OK", output)
        }()
    }

    wg.Wait()
}
```

## Build and Installation

### Makefile

```makefile
# Build CLI
.PHONY: build
build:
	@echo "Building gocurl CLI..."
	go build -o bin/gocurl ./cmd/gocurl

# Install globally
.PHONY: install
install:
	@echo "Installing gocurl..."
	go install ./cmd/gocurl

# Test CLI
.PHONY: test-cli
test-cli:
	@echo "Testing CLI..."
	go test -v ./cmd/gocurl/...

# Test with race detector
.PHONY: test-cli-race
test-cli-race:
	@echo "Testing CLI with race detector..."
	go test -race -v ./cmd/gocurl/...

# Test for memory leaks
.PHONY: test-cli-memory
test-cli-memory:
	@echo "Testing CLI for memory leaks..."
	go test -v ./cmd/gocurl -run TestCLI_NoMemoryLeaks

# All CLI tests
.PHONY: test-cli-all
test-cli-all: test-cli test-cli-race test-cli-memory
```

## Success Criteria

### Must Have ✅

1. **✅ Curl Parity**: 100% of core curl features match real curl behavior
2. **✅ Response API**: Return `*http.Response` with convenience helpers
3. **✅ Exact Parity**: CLI and library produce identical results
4. **✅ Zero Divergence**: Same parser, converter, processor
5. **✅ Multi-line Support**: All formats (backslashes, comments, curl prefix, browser DevTools)
6. **✅ Environment Variables**: Unified expansion in CLI and library (auto from env)
7. **✅ Three Functions**: Curl() (auto-detect), CurlCommand() (shell), CurlArgs() (variadic)
8. **✅ Comprehensive Tests**:
   - 135+ curl parity tests (vs real curl)
   - 50+ unit tests
   - 30+ integration tests
   - 20+ roundtrip tests (CLI ↔ Library)
   - 10+ memory/race tests
   - **Total: 245+ tests**
9. **✅ No Memory Leaks**: Proven via repeated execution
10. **✅ Thread-Safe**: Race detector clean
11. **✅ Error Handling**: All error paths tested
12. **✅ Exit Codes**: Match curl conventions

### Should Have ✅

1. **✅ Streaming Support**: Large files don't load to memory
2. **✅ JSON Helpers**: CurlJSON() for auto-decoding
3. **✅ Download Helpers**: CurlDownload() for direct-to-disk
4. **✅ Performance**: CLI overhead < 1ms
5. **✅ Documentation**: README with CLI examples and all API functions
6. **✅ Examples**: Common use cases documented
7. **✅ Benchmarks**: vs raw curl and vs string-based API

### Nice to Have

1. **Auto-completion**: Shell completion scripts
2. **Man pages**: Unix manual pages
3. **Homebrew**: Installation via brew
4. **Windows**: Chocolatey package

## Implementation Timeline

### Phase 1: Response API Migration (Day 1 - Morning, ~4 hours)

- [ ] Update all Curl functions to return `*http.Response` instead of string
- [ ] Implement convenience functions:
  - [ ] CurlString(), CurlStringCommand(), CurlStringArgs()
  - [ ] CurlBytes(), CurlBytesCommand(), CurlBytesArgs()
  - [ ] CurlJSON(), CurlJSONCommand(), CurlJSONArgs()
  - [ ] CurlDownload(), CurlDownloadCommand(), CurlDownloadArgs()
  - [ ] CurlStream(), CurlStreamCommand(), CurlStreamArgs()
- [ ] Update existing tests to use new API
- [ ] Add response API tests (TestRoundtrip_ResponseAPI)

### Phase 2: Core CLI Implementation (Day 1 - Afternoon, ~4 hours)

- [ ] Create `cmd/gocurl/main.go`
- [ ] Implement `executeCLI()` using shared code
- [ ] Add environment variable expansion (unified with library)
- [ ] Implement multi-line preprocessing (backslashes, comments, curl prefix)
- [ ] Basic output formatting
- [ ] Write 20+ unit tests

### Phase 3: Curl Parity Testing (Day 2 - Full Day, ~8 hours)

- [ ] Create parity test framework (`RunParityTest`, `executeRealCurl`, `compareResults`)
- [ ] Implement core parity tests (~50 tests)
- [ ] Implement browser DevTools tests (~20 tests)
- [ ] Implement API documentation tests (~30 tests)
- [ ] Implement edge case tests (~20 tests)
- [ ] Implement environment variable tests (~15 tests)
- [ ] **Total: 135+ parity tests**
- [ ] Fix any discrepancies found
- [ ] Document parity matrix (gocurl vs curl compatibility)

### Phase 4: Integration & Roundtrip Testing (Day 3 - Morning, ~4 hours)

- [ ] Integration tests (full CLI execution)
- [ ] Roundtrip tests (CLI ↔ Library parity, ~20 tests)
- [ ] Memory leak tests
- [ ] Race condition tests
- [ ] Error path coverage
- [ ] Performance benchmarks

### Phase 5: Polish & Documentation (Day 3 - Afternoon, ~4 hours)

- [ ] Update README with:
  - [ ] CLI installation instructions
  - [ ] Response API usage examples
  - [ ] Multi-line command examples
  - [ ] Environment variable usage
  - [ ] All convenience functions
  - [ ] Browser DevTools workflow
  - [ ] API documentation workflow
- [ ] Add CLI help/usage
- [ ] Create CLI-to-code tutorial
- [ ] Write migration guide (string API → response API)
- [ ] Add performance comparison benchmarks
- [ ] Final review and testing

**Total**: ~3 days (24 hours) for complete, military-grade implementation

### Breakdown

- Response API: 4 hours
- CLI Core: 4 hours
- **Curl Parity Tests: 8 hours** (CRITICAL - ensures curl compatibility)
- Integration/Roundtrip: 4 hours
- Documentation: 4 hours

**Total: 24 hours over 3 days**

Before merging:

- [ ] **Curl Parity Tests**: All 135+ parity tests pass (gocurl vs real curl)
- [ ] **Response API**: All functions return `*http.Response` (not string)
- [ ] **Convenience Functions**: CurlString, CurlBytes, CurlJSON, CurlDownload implemented
- [ ] **Three-Function API**: Curl(), CurlCommand(), CurlArgs() all working
- [ ] **Multi-line Support**: Backslashes, comments, curl prefix, browser DevTools
- [ ] **Environment Variables**: Unified expansion (CLI and library from env)
- [ ] **All tests pass**: `go test ./cmd/gocurl/...` (245+ tests)
- [ ] **Race detector clean**: `go test -race ./cmd/gocurl/...`
- [ ] **No memory leaks**: Proven via TestCLI_NoMemoryLeaks
- [ ] **100% roundtrip parity**: CLI ↔ Library identical behavior
- [ ] **Exit codes match curl**: All error codes correct
- [ ] **Error messages are clear**: User-friendly error output
- [ ] **Documentation complete**:
  - [ ] README with CLI examples
  - [ ] All API functions documented
  - [ ] Multi-line support explained
  - [ ] Environment variable usage
  - [ ] Response API usage patterns
- [ ] **Examples tested**: All examples in docs work
- [ ] **Benchmarks**: Performance vs curl and vs old string API
- [ ] **Code reviewed**: Security, edge cases, error handling
- [ ] **All edge cases covered**: Binary, unicode, large files, streaming
- [ ] **Real-world tests**: Browser DevTools, API docs (Stripe, GitHub, AWS)

## Risk Mitigation

### Risk 1: CLI/Library Divergence

**Mitigation**:

- Share ALL code except input/output
- Roundtrip tests enforce parity
- CI fails if parity breaks

### Risk 2: Environment Variable Security

**Risk**: Malicious input like `-H "X-Evil: $(/bin/rm -rf /)"` could execute commands

**Mitigation**:

1. **CLI**: Use `os.ExpandEnv()` NOT `os.Expand()` with shell execution
   - ✅ `os.ExpandEnv()` only reads environment variables
   - ❌ Never use shell expansion (no command execution)
   - ✅ Validate: `$VAR` and `${VAR}` syntax ONLY

2. **Library**: Expand from explicit map, never environment
   - ✅ User controls exact variable mapping
   - ✅ No access to environment at all
   - ✅ Testable without side effects

3. **Both**: Only expand STRING tokens, never FLAG tokens
   - ✅ Prevents flag injection via variables
   - ✅ Flag names must be literal (e.g., `-H` not `$FLAG`)

**Example Security Test:**

```go
func TestEnvironmentExpansion_NoCommandInjection(t *testing.T) {
    // Malicious environment variable
    os.Setenv("EVIL", "$(rm -rf /)")
    defer os.Unsetenv("EVIL")

    tokens := []tokenizer.Token{
        {Type: tokenizer.STRING, Value: "Header: $EVIL"},
    }

    result := expandEnvInTokens(tokens)

    // Should expand to literal string, NOT execute
    assert.Equal(t, "Header: $(rm -rf /)", result[0].Value)
    // os.ExpandEnv does NOT execute shell commands
}

func TestEnvironmentExpansion_OnlyStringTokens(t *testing.T) {
    // Try to inject flag via environment
    os.Setenv("FLAG", "-X")
    defer os.Unsetenv("FLAG")

    tokens := []tokenizer.Token{
        {Type: tokenizer.FLAG, Value: "$FLAG"},  // Should NOT expand
    }

    result := expandEnvInTokens(tokens)

    // Flag tokens should never expand
    assert.Equal(t, "$FLAG", result[0].Value)
}
```

### Risk 3: Memory Leaks

**Mitigation**:

- Explicit defer cleanup
- Memory leak tests in CI
- Regular profiling### Risk 4: Platform Differences

**Mitigation**:
- Test on Windows/Linux/Mac
- Handle path separators correctly
- Use filepath.Join, not hardcoded slashes

## Next Steps

1. **Create basic CLI structure** (1 hour)
2. **Implement shared code path** (2 hours)
3. **Write comprehensive tests** (4 hours)
4. **Verify parity** (2 hours)
5. **Documentation** (2 hours)

**Total**: ~11 hours (1.5 days) for military-grade CLI implementation

---

**Ready to implement?** This plan ensures:
- ✅ Zero divergence between CLI and library
- ✅ Military-grade testing
- ✅ No memory leaks
- ✅ Thread-safe
- ✅ Curl-compatible
- ✅ Production-ready

Let me know when to start implementation!
