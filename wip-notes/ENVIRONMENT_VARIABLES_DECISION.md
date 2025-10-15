# Environment Variables Decision - October 15, 2025

## Decision: UNIFIED Expansion (CLI and Library)

**Date**: October 15, 2025
**Status**: Approved ✅
**Impact**: Core workflow alignment

## The Decision

Both CLI and library will support `$VAR` and `${VAR}` expansion by default.

### Rationale

1. **Core Objective**: "Test with CLI first" requires IDENTICAL behavior
2. **Zero Translation**: Copy command from CLI to Go code - it just works
3. **User Experience**: Natural, intuitive - same syntax everywhere
4. **Curl Parity**: curl CLI expands environment variables

## Implementation

### CLI (gocurl command)

```bash
$ export API_TOKEN=secret123
$ gocurl -H "Authorization: Bearer $API_TOKEN" https://api.example.com
# ✅ Expands to: "Authorization: Bearer secret123"
```

### Library (Default API)

```go
import "github.com/maniartech/gocurl"

// Same environment variable
os.Setenv("API_TOKEN", "secret123")

// IDENTICAL syntax to CLI!
resp, body, err := gocurl.Curl(ctx,
    "-H", "Authorization: Bearer $API_TOKEN",  // ✅ Auto-expands!
    "https://api.example.com",
)
// ✅ Expands to: "Authorization: Bearer secret123"
// ✅ SAME behavior as CLI
```

### Library (Alternative - Explicit Map)

For testing or security-critical code, use explicit variable map:

```go
// Use explicit map instead of environment
vars := map[string]string{"API_TOKEN": "secret123"}

resp, body, err := gocurl.CurlWithVars(ctx, vars,
    "-H", "Authorization: Bearer $API_TOKEN",
    "https://api.example.com",
)
// ✅ Expands from map (isolated, testable)
// ✅ Does NOT read environment
```

## API Summary

| Function | Expansion Source | Use Case |
|----------|------------------|----------|
| `Curl()` | Environment | Production (matches CLI) |
| `CurlWithVars()` | Explicit map | Testing, security-critical |

## Security Considerations

### What's Safe ✅

- **Variable expansion only**: `$VAR` and `${VAR}` syntax
- **No command execution**: Uses `os.ExpandEnv()` (not shell)
- **String tokens only**: Flags never expand (prevents injection)
- **Explicit control**: Can use `CurlWithVars()` for strict control

### What's Protected ❌

```go
// ❌ This CANNOT happen - no shell command execution
os.Setenv("EVIL", "$(rm -rf /)")
gocurl.Curl(ctx, "-H", "X-Bad: $EVIL")
// ✅ Expands to literal: "X-Bad: $(rm -rf /)"
// ✅ Shell commands are NOT executed
```

### Preventing Flag Injection

```go
// ❌ This CANNOT happen - flags don't expand
os.Setenv("FLAG", "-X")
tokens := []tokenizer.Token{
    {Type: tokenizer.FLAG, Value: "$FLAG"},
}
// ✅ Stays as: "$FLAG" (literal)
// ✅ Only STRING tokens expand, never FLAG tokens
```

## Testing Strategy

### Roundtrip Tests

Guarantee CLI and library behave identically:

```go
func TestRoundtrip_EnvironmentExpansion(t *testing.T) {
    // Set environment
    os.Setenv("TOKEN", "secret123")
    defer os.Unsetenv("TOKEN")

    // 1. Execute via CLI
    cliOutput := runCLI("gocurl", "-H", "X-Token: $TOKEN", server.URL)

    // 2. Execute via Library
    _, libOutput, _ := gocurl.Curl(ctx, "-H", "X-Token: $TOKEN", server.URL)

    // 3. MUST match exactly
    assert.Equal(t, cliOutput, libOutput)
}
```

### Isolation Tests

Verify `CurlWithVars()` uses explicit map (not environment):

```go
func TestCurlWithVars_IgnoresEnvironment(t *testing.T) {
    // Set wrong value in environment
    os.Setenv("TOKEN", "wrong")
    defer os.Unsetenv("TOKEN")

    // Use explicit map
    vars := map[string]string{"TOKEN": "correct"}

    resp, body, err := gocurl.CurlWithVars(ctx, vars,
        "-H", "X-Token: $TOKEN",
        server.URL,
    )

    // Should use map value, NOT environment
    assert.Contains(t, body, "correct")
    assert.NotContains(t, body, "wrong")
}
```

### Security Tests

Verify no command execution:

```go
func TestEnvironmentExpansion_NoCommandExecution(t *testing.T) {
    // Try to inject shell command
    os.Setenv("EVIL", "$(rm -rf /)")
    defer os.Unsetenv("EVIL")

    resp, body, err := gocurl.Curl(ctx,
        "-H", "X-Test: $EVIL",
        server.URL,
    )

    // Should expand to literal string (not execute)
    // Server will receive: "X-Test: $(rm -rf /)"
    assert.NoError(t, err)
    // Command NOT executed (verified by no side effects)
}
```

## Migration Guide

### For Users Upgrading

**Before** (if we had no expansion):
```go
// Manual string interpolation
token := os.Getenv("API_TOKEN")
gocurl.Curl(ctx, "-H", fmt.Sprintf("Authorization: Bearer %s", token))
```

**After** (with unified expansion):
```go
// Automatic expansion (like CLI!)
gocurl.Curl(ctx, "-H", "Authorization: Bearer $API_TOKEN")
```

### For Security-Critical Code

**Before** (if we had no expansion):
```go
// Explicit values (safest)
gocurl.Curl(ctx, "-H", "Authorization: Bearer secret123")
```

**After** (use CurlWithVars):
```go
// Explicit map (same safety, more flexibility)
vars := map[string]string{"TOKEN": "secret123"}
gocurl.CurlWithVars(ctx, vars, "-H", "Authorization: Bearer $TOKEN")
```

## Documentation Requirements

### README.md

Add section:
- Environment variable expansion (`$VAR` and `${VAR}`)
- Examples showing CLI and library parity
- Security notes (no command execution)
- When to use `CurlWithVars()`

### API Documentation

```go
// Curl executes an HTTP request using curl-compatible syntax.
//
// Environment variables in the form $VAR or ${VAR} are automatically
// expanded in string arguments (not flags). This matches curl CLI behavior.
//
// For testing or security-critical code, use CurlWithVars() with an explicit
// variable map instead of reading from the environment.
//
// Example:
//   os.Setenv("TOKEN", "secret123")
//   resp, body, err := gocurl.Curl(ctx,
//       "-H", "Authorization: Bearer $TOKEN",  // Auto-expands
//       "https://api.example.com",
//   )
func Curl(ctx context.Context, command ...string) (*Response, string, error)

// CurlWithVars executes an HTTP request with explicit variable expansion.
//
// Variables in the form $VAR or ${VAR} are expanded from the provided map,
// NOT from the environment. This provides isolation for testing and explicit
// control for security-critical code.
//
// Example:
//   vars := map[string]string{"TOKEN": "secret123"}
//   resp, body, err := gocurl.CurlWithVars(ctx, vars,
//       "-H", "Authorization: Bearer $TOKEN",  // Expands from map
//       "https://api.example.com",
//   )
func CurlWithVars(ctx context.Context, vars map[string]string, command ...string) (*Response, string, error)
```

## Success Criteria

- [x] Decision documented
- [ ] CLI implementation supports `$VAR` expansion
- [ ] Library `Curl()` supports `$VAR` expansion (from environment)
- [ ] Library `CurlWithVars()` supports `$VAR` expansion (from map)
- [ ] Roundtrip tests verify CLI/library parity
- [ ] Security tests verify no command execution
- [ ] Documentation updated with examples
- [ ] Migration guide provided

## Related Documents

- `CLI_IMPLEMENTATION_PLAN.md` - Full CLI implementation plan
- `OBJECTIVES_STATUS_REVIEW.md` - Overall project status

## Approval

**Decision maker**: User (maniartech)
**Date**: October 15, 2025
**Status**: ✅ Approved

**User quote**: "Yes, we need to support this!" (re: `gocurl.Curl(ctx, "-H", "Authorization: Bearer $TOKEN")`)

---

**Next Steps**: Implement according to CLI_IMPLEMENTATION_PLAN.md with unified expansion strategy.
