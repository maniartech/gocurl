# Multi-line Curl Command Support - October 15, 2025

## Decision: Full Multi-line Command Support

**Date**: October 15, 2025
**Status**: Approved ✅
**Scope**: CLI and Library

## Overview

GoCurl will support **all** common curl command formats:

1. ✅ **Variadic arguments** (Go-style)
2. ✅ **Single-line strings** (shell copy/paste)
3. ✅ **Multi-line with backslashes** (API docs style)
4. ✅ **Multi-line without backslashes** (simple newlines)
5. ✅ **With comments** (documentation style)
6. ✅ **Browser DevTools** (Chrome/Firefox "Copy as cURL")

## Use Cases

### Use Case 1: API Documentation

**Most API docs show multi-line curl commands:**

```bash
# Stripe API documentation
curl https://api.stripe.com/v1/charges \
  -u sk_test_xyz: \
  -d amount=2000 \
  -d currency=usd \
  -d source=tok_visa \
  -d description="Charge for demo"
```

**Users can copy/paste directly:**

```go
resp, body, err := gocurl.Curl(ctx, `
curl https://api.stripe.com/v1/charges \
  -u sk_test_xyz: \
  -d amount=2000 \
  -d currency=usd \
  -d source=tok_visa \
  -d description="Charge for demo"
`)
```

### Use Case 2: Browser DevTools

**Chrome/Firefox: Network → Right-click → Copy as cURL**

```bash
curl 'https://api.example.com/graphql' \
  -X POST \
  -H 'authority: api.example.com' \
  -H 'accept: application/json' \
  -H 'authorization: Bearer eyJhbGciOiJI...' \
  -H 'content-type: application/json' \
  --data-raw '{"query":"{ user { name } }"}'
```

**Direct paste into Go:**

```go
resp, body, err := gocurl.Curl(ctx, `
curl 'https://api.example.com/graphql' \
  -X POST \
  -H 'authority: api.example.com' \
  -H 'accept: application/json' \
  -H 'authorization: Bearer eyJhbGciOiJI...' \
  -H 'content-type: application/json' \
  --data-raw '{"query":"{ user { name } }"}'
`)
```

### Use Case 3: System Admin Scripts

**Shell scripts often use readable multi-line format:**

```bash
#!/bin/bash
# Deploy new version

curl -X POST https://deploy.example.com/api/deploy \
  -H "Authorization: Bearer $DEPLOY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "version": "v2.0.0",
    "environment": "production",
    "rollback_on_error": true
  }'
```

**Same command works in Go:**

```go
resp, body, err := gocurl.Curl(ctx, `
curl -X POST https://deploy.example.com/api/deploy \
  -H "Authorization: Bearer $DEPLOY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "version": "v2.0.0",
    "environment": "production",
    "rollback_on_error": true
  }'
`)
```

### Use Case 4: README/Tutorial Examples

**Documentation with comments:**

```bash
# Example: Create a GitHub issue
curl -X POST \
  https://api.github.com/repos/OWNER/REPO/issues \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -d '{
    "title": "Bug report",
    "body": "Steps to reproduce..."
  }'
```

**Copy directly to code:**

```go
resp, body, err := gocurl.Curl(ctx, `
# Example: Create a GitHub issue
curl -X POST \
  https://api.github.com/repos/OWNER/REPO/issues \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -d '{
    "title": "Bug report",
    "body": "Steps to reproduce..."
  }'
`)
```

## Implementation

### Preprocessing Function

```go
// preprocessMultilineCommand handles all multi-line formats
func preprocessMultilineCommand(cmd string) string {
    lines := strings.Split(cmd, "\n")
    var processed []string

    for _, line := range lines {
        // Trim whitespace
        line = strings.TrimSpace(line)

        // Skip empty lines
        if line == "" {
            continue
        }

        // Skip comment lines
        if strings.HasPrefix(line, "#") {
            continue
        }

        // Remove 'curl' prefix from first line
        if len(processed) == 0 && strings.HasPrefix(line, "curl ") {
            line = strings.TrimPrefix(line, "curl ")
            line = strings.TrimSpace(line)
        }

        // Handle backslash continuation
        if strings.HasSuffix(line, "\\") {
            line = strings.TrimSuffix(line, "\\")
            line = strings.TrimSpace(line)
        }

        processed = append(processed, line)
    }

    // Join with spaces
    return strings.Join(processed, " ")
}
```

### Integration with Curl()

```go
func Curl(ctx context.Context, command ...string) (*Response, string, error) {
    var tokens []tokenizer.Token
    var err error

    if len(command) == 1 {
        // Single string - preprocess multi-line
        cmdStr := preprocessMultilineCommand(command[0])
        tokens, err = tokenizer.Tokenize(cmdStr)
    } else {
        // Variadic - direct tokenization
        tokens, err = TokenizeArgs(command)
    }

    // ... rest of implementation
}
```

## Processing Rules

### 1. Line Continuations

**Input:**
```
curl -X POST \
  -H 'X-Token: abc' \
  https://example.com
```

**Processed:**
```
-X POST -H 'X-Token: abc' https://example.com
```

### 2. Newlines as Spaces

**Input:**
```
curl -X POST
  -H 'X-Token: abc'
  https://example.com
```

**Processed:**
```
-X POST -H 'X-Token: abc' https://example.com
```

### 3. Comment Lines

**Input:**
```
# This is a comment
curl -X POST
  # Another comment
  -H 'X-Token: abc'
  https://example.com
```

**Processed:**
```
-X POST -H 'X-Token: abc' https://example.com
```

### 4. curl Prefix Removal

**Input:**
```
curl -X POST https://example.com
```

**Processed:**
```
-X POST https://example.com
```

### 5. Mixed Indentation

**Input:**
```
curl -X POST \
    -H 'X-Token: abc' \
  https://example.com
```

**Processed:**
```
-X POST -H 'X-Token: abc' https://example.com
```

### 6. Empty Lines

**Input:**
```
curl -X POST

  -H 'X-Token: abc'

  https://example.com
```

**Processed:**
```
-X POST -H 'X-Token: abc' https://example.com
```

## Testing Strategy

### Unit Tests

```go
func TestPreprocessMultilineCommand(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            "Backslash continuations",
            "curl -X POST \\\n  -H 'X-Token: abc' \\\n  url",
            "-X POST -H 'X-Token: abc' url",
        },
        {
            "Without backslashes",
            "curl -X POST\n  -H 'X-Token: abc'\n  url",
            "-X POST -H 'X-Token: abc' url",
        },
        {
            "With comments",
            "# Comment\ncurl -X POST\n  url",
            "-X POST url",
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := preprocessMultilineCommand(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Tests

```go
func TestCurl_MultilineCommand(t *testing.T) {
    server := httptest.NewServer(/* ... */)
    defer server.Close()

    // Test multi-line with backslashes
    resp, body, err := gocurl.Curl(ctx, fmt.Sprintf(`
curl -X POST %s \
  -H 'Content-Type: application/json' \
  -d '{"key":"value"}'
`, server.URL))

    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}
```

### Real-world Tests

```go
func TestCurl_BrowserDevToolsCopy(t *testing.T) {
    // Use actual output from Chrome DevTools
    command := `
curl 'https://api.example.com/graphql' \
  -X POST \
  -H 'authority: api.example.com' \
  -H 'accept: application/json' \
  --data-raw '{"query":"..."}'
`

    resp, body, err := gocurl.Curl(ctx, command)

    assert.NoError(t, err)
    // Verify correct behavior
}
```

## Benefits

### For Developers

1. **Copy/Paste Workflow**: Copy from API docs → paste to code → works immediately
2. **Zero Translation**: No manual conversion of multi-line to single-line
3. **Readable Code**: Multi-line commands in code are readable, just like in shell
4. **Documentation**: Can paste examples directly from README/tutorials

### For System Administrators

1. **Script Reuse**: Shell scripts convert to Go with minimal changes
2. **Familiar Syntax**: Same format they use daily
3. **Environment Variables**: `$VAR` expansion works identically
4. **Comments**: Can keep documentation comments from scripts

### For API Users

1. **Browser Tools**: DevTools "Copy as cURL" works directly
2. **API Docs**: Official documentation examples paste directly
3. **Tutorials**: Follow along with tutorials without adaptation
4. **Debugging**: Same command in shell and code for testing

## Edge Cases Handled

1. ✅ **Windows line endings** (`\r\n`)
2. ✅ **Tab characters** in indentation
3. ✅ **Mixed quote styles** (single, double, backticks)
4. ✅ **Escaped quotes** within strings
5. ✅ **URLs with query params** and special chars
6. ✅ **JSON with newlines** in `-d` parameter
7. ✅ **Comments mid-command** (line-by-line)
8. ✅ **Empty/whitespace-only lines**

## Security Considerations

### Safe Processing

- ✅ **No shell execution**: Only string manipulation
- ✅ **No eval/exec**: Pure parsing
- ✅ **Comment stripping**: `#` comments removed before parsing
- ✅ **Whitespace normalization**: Consistent processing

### Environment Variables

- ✅ **$VAR expansion**: From environment (CLI) or map (library)
- ✅ **No command substitution**: `$(cmd)` treated as literal string
- ✅ **Escaping supported**: `$$VAR` → `$VAR` (literal)

## Documentation Requirements

### README Examples

Add section showing:

1. Multi-line command with backslashes
2. Browser DevTools copy/paste
3. API documentation copy/paste
4. Environment variable usage
5. Comment preservation

### API Documentation

```go
// Curl executes an HTTP request using curl-compatible syntax.
//
// Supports multiple input formats:
// 1. Variadic: Curl(ctx, "-X", "POST", "-H", "...", url)
// 2. Single-line: Curl(ctx, "-X POST -H '...' " + url)
// 3. Multi-line: Curl(ctx, `curl -X POST \n  -H '...' \n  ` + url)
//
// Multi-line commands are preprocessed:
// - Backslash continuations (\) are handled
// - 'curl' prefix is automatically removed
// - Comment lines (#) are stripped
// - Newlines are treated as spaces
// - Environment variables ($VAR) are expanded
//
// Example - API documentation copy/paste:
//   resp, body, err := gocurl.Curl(ctx, `
//     curl https://api.stripe.com/v1/charges \
//       -u sk_test_xyz: \
//       -d amount=2000 \
//       -d currency=usd
//   `)
//
// Example - Browser DevTools:
//   resp, body, err := gocurl.Curl(ctx, `
//     curl 'https://api.example.com/search' \
//       -X POST \
//       -H 'authorization: Bearer ...' \
//       --data-raw '{"query":"test"}'
//   `)
func Curl(ctx context.Context, command ...string) (*Response, string, error)
```

## Success Criteria

- [ ] Multi-line commands with `\` work correctly
- [ ] Multi-line commands without `\` work correctly
- [ ] Comment lines are stripped
- [ ] `curl` prefix is removed automatically
- [ ] Empty lines are handled
- [ ] Mixed indentation works
- [ ] Browser DevTools copy/paste works
- [ ] API documentation examples work
- [ ] Environment variables expand correctly
- [ ] 50+ tests covering all scenarios
- [ ] Documentation with examples
- [ ] Zero regressions in existing functionality

## Related Documents

- `CLI_IMPLEMENTATION_PLAN.md` - Full CLI implementation plan
- `ENVIRONMENT_VARIABLES_DECISION.md` - Environment variable strategy
- `OBJECTIVES_STATUS_REVIEW.md` - Overall project status

---

**Next Steps**: Implement `preprocessMultilineCommand()` and integrate with `Curl()` API.
