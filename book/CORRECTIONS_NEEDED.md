# Corrections Needed for GoCurl Book

## Issue

All book examples incorrectly use:
```go
resp, body, err := gocurl.Curl(ctx, url)  // ❌ WRONG - Curl() returns 2 values, not 3!
```

## Correct Patterns

### Option 1: Use CurlString() for simple examples (RECOMMENDED)
```go
body, resp, err := gocurl.CurlString(ctx, url)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()
fmt.Println(body)
```

### Option 2: Use Curl() and manually read body
```go
resp, err := gocurl.Curl(ctx, url)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

body, err := io.ReadAll(resp.Body)
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(body))
```

### Option 3: Use CurlJSON() for JSON responses
```go
var result MyStruct
resp, err := gocurl.CurlJSON(ctx, &result, url)
if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()
```

## Files Needing Correction

1. **outline.md** - ~40+ instances
2. **code_standards.md** - ~15+ instances
3. **API_REFERENCE_QUICK.md** - Fixed (intentional wrong example with warning)

## Correction Strategy

For each file, decide which pattern to use based on context:

- **Simple GET examples** → Use `CurlString()` (clearest for beginners)
- **JSON API examples** → Use `CurlJSON()` (most appropriate)
- **Advanced/custom** → Use `Curl()` + manual read (shows full control)

## Status

- [ ] Fix outline.md
- [ ] Fix code_standards.md
- [x] API_REFERENCE_QUICK.md (intentionally shows wrong pattern as warning)

---

**Created:** October 17, 2025
**Priority:** CRITICAL - Must fix before writing any chapters
