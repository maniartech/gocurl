# CRITICAL FINDINGS: GoCurl Book API Audit

**Date:** October 17, 2025
**Auditor:** AI Assistant
**Status:** 🔴 CRITICAL ISSUES FOUND

---

## 🚨 CRITICAL DISCOVERY

### Missing API Functions

The book uses `CurlCommandWithVars` expecting it to return `(resp, body, err)`, but:

1. ❌ **`CurlCommandWithVars` only returns 2 values**: `(*http.Response, error)`
2. ❌ **No `CurlStringCommandWithVars` exists** in the codebase
3. ❌ **No `CurlBytesCommandWithVars` exists** in the codebase
4. ❌ **No `CurlStringArgsWithVars` exists** in the codebase
5. ❌ **No `CurlBytesArgsWithVars` exists** in the codebase

### What EXISTS in the codebase:

```go
// ✅ These exist (return response only - 2 values)
func CurlWithVars(ctx, vars, command...) (*http.Response, error)
func CurlCommandWithVars(ctx, vars, command) (*http.Response, error)
func CurlArgsWithVars(ctx, vars, args...) (*http.Response, error)
```

### What DOES NOT EXIST:

```go
// ❌ These DO NOT exist
func CurlStringWithVars(...) (string, *http.Response, error)          // MISSING
func CurlStringCommandWithVars(...) (string, *http.Response, error)   // MISSING
func CurlStringArgsWithVars(...) (string, *http.Response, error)      // MISSING
func CurlBytesWithVars(...) ([]byte, *http.Response, error)           // MISSING
func CurlBytesCommandWithVars(...) ([]byte, *http.Response, error)    // MISSING
func CurlBytesArgsWithVars(...) ([]byte, *http.Response, error)       // MISSING
```

---

## 📊 COMPLETE ERROR COUNT

### Files Audited:

| File | Total Errors | Status | Priority |
|------|-------------|--------|----------|
| **code_standards.md** | 15 | ✅ FIXED | - |
| **outline.md** | 40+ | ❌ UNFIXED | 🔴 CRITICAL |
| **style_guide.md** | 3 | ❌ UNFIXED | 🟡 HIGH |
| **API_REFERENCE_QUICK.md** | 0 | ✅ CORRECT | - |
| **__plan.md** | ? | ⏳ NOT AUDITED | 🟢 LOW |

**Total Remaining Errors:** 43+ (not including __plan.md)

---

## 🔧 RECOMMENDED SOLUTIONS

### For WithVars Examples:

Since `CurlString*WithVars` functions don't exist, we have **3 options**:

#### Option 1: Use Basic WithVars + Manual Body Read (RECOMMENDED)
```go
resp, err := gocurl.CurlCommandWithVars(ctx, vars, cmd)
if err != nil {
    return err
}
defer resp.Body.Close()

body, err := io.ReadAll(resp.Body)
if err != nil {
    return err
}
// Use string(body)
```

#### Option 2: Avoid WithVars in Simple Examples
```go
// Instead of WithVars, use string formatting
token := vars["api_key"]
cmd := fmt.Sprintf(`curl -H "X-API-Key: %s" https://api.example.com`, token)
body, resp, err := gocurl.CurlStringCommand(ctx, cmd)
```

#### Option 3: Request New Functions (Long-term)
File feature request to add:
- `CurlStringCommandWithVars`
- `CurlBytesCommandWithVars`
- And corresponding Args versions

---

## 📝 DETAILED FIX LIST FOR outline.md

### Introduction Section (Lines 60-76)

**Line 74:**
```go
// ❌ WRONG
resp, body, err := gocurl.CurlCommand(ctx,
    `curl https://api.stripe.com/v1/charges \
      -u sk_test_xyz: \
      -d amount=2000 \
      -d currency=usd`)

// ✅ CORRECT
body, resp, err := gocurl.CurlStringCommand(ctx,
    `curl https://api.stripe.com/v1/charges \
      -u sk_test_xyz: \
      -d amount=2000 \
      -d currency=usd`)
```

### Chapter 1: Why GoCurl? (Lines 104-245)

**Line 148:**
```go
// ❌ WRONG
resp, body, err := gocurl.CurlCommand(ctx,
    `curl -H "Accept: application/vnd.github+json" \
          -H "Authorization: Bearer ` + token + `" \
          https://api.github.com/repos/golang/go/issues`)

// ✅ CORRECT
body, resp, err := gocurl.CurlStringCommand(ctx,
    `curl -H "Accept: application/vnd.github+json" \
          -H "Authorization: Bearer ` + token + `" \
          https://api.github.com/repos/golang/go/issues`)
```

**Line 190:**
```go
// ❌ WRONG
resp, body, err := gocurl.CurlCommand(ctx,
    `gocurl -H "Authorization: Bearer ` + token + `" \
            https://api.github.com/repos/golang/go`)

// ✅ CORRECT
body, resp, err := gocurl.CurlStringCommand(ctx,
    `gocurl -H "Authorization: Bearer ` + token + `" \
            https://api.github.com/repos/golang/go`)
```

**Lines 216-227** (Hands-on example):
```go
// ❌ WRONG
resp, body, err := gocurl.CurlCommand(
    context.Background(),
    `curl -H "Authorization: Bearer ` + token + `" \
          https://api.github.com/user`,
)

if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

fmt.Println("Status:", resp.StatusCode)
fmt.Println("Body:", body)

// ✅ CORRECT
body, resp, err := gocurl.CurlStringCommand(
    context.Background(),
    `curl -H "Authorization: Bearer ` + token + `" \
          https://api.github.com/user`,
)

if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

fmt.Println("Status:", resp.StatusCode)
fmt.Println("Body:", body)
```

### Chapter 2: Installation & Setup (Lines 449-454)

**Lines 449-454:**
```go
// ❌ WRONG
func main() {
    resp, body, _ := gocurl.Curl(
        context.Background(),
        "https://api.github.com/zen",
    )
    fmt.Println("Status:", resp.StatusCode)
    fmt.Println("Body:", body)
}

// ✅ CORRECT - Option 1 (Simple)
func main() {
    body, resp, err := gocurl.CurlString(
        context.Background(),
        "https://api.github.com/zen",
    )
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    fmt.Println("Status:", resp.StatusCode)
    fmt.Println("Body:", body)
}

// ✅ CORRECT - Option 2 (Manual read)
func main() {
    resp, err := gocurl.Curl(
        context.Background(),
        "https://api.github.com/zen",
    )
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Status:", resp.StatusCode)
    fmt.Println("Body:", string(body))
}
```

### Chapter 3: Core Concepts (Lines 497-750)

**Line 551** (Environment Variables):
```go
// ❌ WRONG
resp, body, err := gocurl.CurlCommand(ctx,
    `curl -H "Authorization: Bearer $API_KEY" https://api.example.com`)

// ✅ CORRECT
body, resp, err := gocurl.CurlStringCommand(ctx,
    `curl -H "Authorization: Bearer $API_KEY" https://api.example.com`)
```

**Line 563** (Variable Maps - CRITICAL):
```go
// ❌ WRONG (function doesn't return 3 values AND there's no String version)
resp, body, err := gocurl.CurlCommandWithVars(ctx, vars,
    `curl https://api.example.com${endpoint} -H "X-API-Key: ${api_key}"`)

// ✅ CORRECT (use manual read)
resp, err := gocurl.CurlCommandWithVars(ctx, vars,
    `curl https://api.example.com${endpoint} -H "X-API-Key: ${api_key}"`)
if err != nil {
    return err
}
defer resp.Body.Close()

bodyBytes, err := io.ReadAll(resp.Body)
if err != nil {
    return err
}
body := string(bodyBytes)
```

**Line 575** (Context Timeout):
```go
// ❌ WRONG
resp, body, err := gocurl.Curl(ctx, "https://slow-api.example.com")

// ✅ CORRECT
body, resp, err := gocurl.CurlString(ctx, "https://slow-api.example.com")
```

**Lines 596-602** (Basic Response):
```go
// ❌ WRONG
resp, body, err := gocurl.Curl(ctx, url)
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Println("Status:", resp.StatusCode)
fmt.Println("Headers:", resp.Header)
fmt.Println("Body:", body)

// ✅ CORRECT
body, resp, err := gocurl.CurlString(ctx, url)
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Println("Status:", resp.StatusCode)
fmt.Println("Headers:", resp.Header)
fmt.Println("Body:", body)
```

**Lines 616-627** (JSON Unmarshaling - USE CurlJSON):
```go
// ❌ WRONG
resp, body, err := gocurl.Curl(ctx, "https://api.example.com/user/1")
if err != nil {
    return err
}

var user User
if err := json.Unmarshal([]byte(body), &user); err != nil {
    return err
}

// ✅ CORRECT - Best approach
var user User
resp, err := gocurl.CurlJSON(ctx, &user, "https://api.example.com/user/1")
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Printf("User: %+v\n", user)
```

**Lines 688-707** (GitHub Hands-on - USE CurlJSON):
```go
// ❌ WRONG
resp, body, err := gocurl.CurlCommand(
    context.Background(),
    `curl -H "Accept: application/vnd.github+json" \
          -H "Authorization: Bearer `+token+`" \
          `+url,
)

if err != nil {
    return nil, fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != 200 {
    return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
}

var repository Repository
if err := json.Unmarshal([]byte(body), &repository); err != nil {
    return nil, fmt.Errorf("failed to parse response: %w", err)
}

return &repository, nil

// ✅ CORRECT - Best approach
var repository Repository
resp, err := gocurl.CurlJSONCommand(
    context.Background(),
    &repository,
    `curl -H "Accept: application/vnd.github+json" \
          -H "Authorization: Bearer `+token+`" \
          `+url,
)

if err != nil {
    return nil, fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != 200 {
    return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
}

return &repository, nil
```

---

## ⚡ NEXT STEPS

### Immediate Actions:

1. **🔴 PRIORITY 1:** Fix all 40+ errors in `outline.md`
   - Use `CurlStringCommand()` for examples that need body
   - Use `CurlJSON()` / `CurlJSONCommand()` for JSON responses
   - Use manual `io.ReadAll()` for WithVars examples
   - Ensure correct return value order: `(body, resp, err)` or `(resp, err)`

2. **🟡 PRIORITY 2:** Fix 3 errors in `style_guide.md`
   - Same patterns as outline.md

3. **🟢 PRIORITY 3:** Audit `__plan.md`
   - Check for any code examples
   - Verify no wrong API usage

4. **📝 PRIORITY 4:** Create decision document
   - Document why we use CurlString vs Curl vs CurlJSON
   - Create examples showing all 3 patterns
   - Add to book as reference

### Quality Assurance:

- [ ] Create test file with all corrected examples
- [ ] Run `go build` on test file to verify compilation
- [ ] Test at least 5 examples against real APIs
- [ ] Document any edge cases found

---

## 📚 DECISION MATRIX FOR BOOK AUTHORS

### When writing examples, use this decision tree:

```
Is the response JSON that needs parsing into a struct?
├─ YES → Use CurlJSON() or CurlJSONCommand()
│         Example: resp, err := gocurl.CurlJSON(ctx, &user, url)
│
└─ NO → Does the example need the response body as string?
   ├─ YES → Use CurlString() or CurlStringCommand()
   │         Example: body, resp, err := gocurl.CurlString(ctx, url)
   │
   └─ NO → Does the example show advanced/streaming/custom processing?
      └─ YES → Use Curl() or CurlCommand() with manual body read
                Example: resp, err := gocurl.Curl(ctx, url)
                         body, err := io.ReadAll(resp.Body)
```

---

**Status:** AUDIT COMPLETE
**Total Errors Found:** 43+
**Total Errors Fixed:** 15
**Remaining Work:** 43+ fixes needed

**Last Updated:** October 17, 2025
