# Complete API Audit - GoCurl Book

**Date:** October 17, 2025
**Purpose:** Systematic verification of ALL API signatures across book documentation
**Status:** AUDIT IN PROGRESS

---

## VERIFIED CORRECT API SIGNATURES

From `api.go` analysis:

### ✅ Basic Functions (Return *http.Response only - 2 values)
```go
func Curl(ctx context.Context, command ...string) (*http.Response, error)
func CurlCommand(ctx context.Context, command string) (*http.Response, error)
func CurlArgs(ctx context.Context, args ...string) (*http.Response, error)
func CurlWithVars(ctx context.Context, vars Variables, command ...string) (*http.Response, error)
func CurlCommandWithVars(ctx context.Context, vars Variables, command string) (*http.Response, error)
func CurlArgsWithVars(ctx context.Context, vars Variables, args ...string) (*http.Response, error)
```

### ✅ String Body Functions (Return body, response, error - 3 values)
```go
func CurlString(ctx context.Context, command ...string) (string, *http.Response, error)
func CurlStringCommand(ctx context.Context, command string) (string, *http.Response, error)
func CurlStringArgs(ctx context.Context, args ...string) (string, *http.Response, error)
```

### ✅ Bytes Body Functions (Return body, response, error - 3 values)
```go
func CurlBytes(ctx context.Context, command ...string) ([]byte, *http.Response, error)
func CurlBytesCommand(ctx context.Context, command string) ([]byte, *http.Response, error)
func CurlBytesArgs(ctx context.Context, args ...string) ([]byte, *http.Response, error)
```

### ✅ JSON Functions (Return response only, decodes into struct - 2 values)
```go
func CurlJSON(ctx context.Context, v interface{}, command ...string) (*http.Response, error)
func CurlJSONCommand(ctx context.Context, v interface{}, command string) (*http.Response, error)
func CurlJSONArgs(ctx context.Context, v interface{}, args ...string) (*http.Response, error)
```

### ✅ Download Functions (Return bytes written, response, error - 3 values)
```go
func CurlDownload(ctx context.Context, filepath string, command ...string) (int64, *http.Response, error)
func CurlDownloadCommand(ctx context.Context, filepath string, command string) (int64, *http.Response, error)
func CurlDownloadArgs(ctx context.Context, filepath string, args ...string) (int64, *http.Response, error)
```

---

## ERRORS FOUND IN BOOK FILES

### ❌ outline.md - CRITICAL ERRORS (40+ instances)

All instances use **WRONG** signature: `resp, body, err :=` (3 values)
Should use **CORRECT** signatures based on context.

**Location: Lines with errors:**

1. **Line 74** - Introduction example
   ```go
   ❌ resp, body, err := gocurl.CurlCommand(ctx, ...)
   ✅ Should be: body, resp, err := gocurl.CurlStringCommand(ctx, ...)
   ```

2. **Line 148** - Chapter 1 example
   ```go
   ❌ resp, body, err := gocurl.CurlCommand(ctx, ...)
   ✅ Should be: body, resp, err := gocurl.CurlStringCommand(ctx, ...)
   ```

3. **Line 190** - Chapter 1 CLI workflow
   ```go
   ❌ resp, body, err := gocurl.CurlCommand(ctx, ...)
   ✅ Should be: body, resp, err := gocurl.CurlStringCommand(ctx, ...)
   ```

4. **Line 216-227** - Chapter 1 hands-on example
   ```go
   ❌ resp, body, err := gocurl.CurlCommand(...)
   ✅ Should be: body, resp, err := gocurl.CurlStringCommand(...)
   ```

5. **Line 506** - Chapter 3 paradigm (showing function signature only - OK if not assigning)
   ```go
   gocurl.CurlCommand(ctx, `...`)  // ⚠️ Check context
   ```

6. **Line 511** - Chapter 3 paradigm (showing function signature only - OK if not assigning)
   ```go
   gocurl.CurlArgs(ctx, "-X", "POST", ...)  // ⚠️ Check context
   ```

7. **Line 551** - Chapter 3 environment variables
   ```go
   ❌ resp, body, err := gocurl.CurlCommand(ctx, ...)
   ✅ Should be: body, resp, err := gocurl.CurlStringCommand(ctx, ...)
   ```

8. **Line 563** - Chapter 3 variable maps **WITH VARS**
   ```go
   ❌ resp, body, err := gocurl.CurlCommandWithVars(ctx, vars, ...)
   ✅ Should be: body, resp, err := gocurl.CurlStringCommandWithVars(ctx, vars, ...)
   ⚠️ NOTE: Need to verify if CurlStringCommandWithVars exists!
   ```

9. **Line 575** - Chapter 3 context timeout
   ```go
   ❌ resp, body, err := gocurl.Curl(ctx, "https://slow-api.example.com")
   ✅ Should be: body, resp, err := gocurl.CurlString(ctx, ...)
   ```

10. **Line 596-602** - Chapter 3 basic response
    ```go
    ❌ resp, body, err := gocurl.Curl(ctx, url)
    ✅ Should be: body, resp, err := gocurl.CurlString(ctx, url)
    ```

11. **Line 616-627** - Chapter 3 JSON unmarshaling
    ```go
    ❌ resp, body, err := gocurl.Curl(ctx, "https://api.example.com/user/1")
    ✅ Option 1: resp, err := gocurl.CurlJSON(ctx, &user, url)
    ✅ Option 2: body, resp, err := gocurl.CurlString(ctx, url)
    ```

12. **Line 634-649** - Chapter 3 error handling
    ```go
    ❌ resp, body, err := gocurl.Curl(ctx, url)
    ✅ Should be: body, resp, err := gocurl.CurlString(ctx, url)
    ```

13. **Line 688-707** - Chapter 3 hands-on getRepo function
    ```go
    ❌ resp, body, err := gocurl.CurlCommand(...)
    ✅ Option 1: resp, err := gocurl.CurlJSONCommand(ctx, &repository, cmd)
    ✅ Option 2: body, resp, err := gocurl.CurlStringCommand(ctx, cmd)
    ```

14. **Line 764-777** - Chapter 4 shell command syntax examples (3 instances)
    ```go
    ❌ resp, body, err := gocurl.CurlCommand(ctx, ...)
    ✅ Should be: body, resp, err := gocurl.CurlStringCommand(ctx, ...)
    ```

15. **Line 791-808** - Chapter 4 variadic arguments examples (3 instances)
    ```go
    ❌ resp, body, err := gocurl.CurlArgs(ctx, ...)
    ✅ Should be: body, resp, err := gocurl.CurlStringArgs(ctx, ...)
    ```

16. **Line 825-836** - Chapter 4 auto-detect examples (3 instances)
    ```go
    ❌ resp, body, err := gocurl.Curl(ctx, ...)
    ✅ Should be: body, resp, err := gocurl.CurlString(ctx, ...)
    ```

17. **Line 861-869** - Chapter 4 DevTools example
    ```go
    ❌ resp, body, err := gocurl.CurlCommand(ctx, ...)
    ✅ Should be: body, resp, err := gocurl.CurlStringCommand(ctx, ...)
    ```

18. **Line 875-884** - Chapter 4 multi-line backslash
    ```go
    ❌ resp, body, err := gocurl.CurlCommand(ctx, ...)
    ✅ Should be: body, resp, err := gocurl.CurlStringCommand(ctx, ...)
    ```

19. **Line 888-895** - Chapter 4 comment stripping
    ```go
    ❌ resp, body, err := gocurl.CurlCommand(ctx, ...)
    ✅ Should be: body, resp, err := gocurl.CurlStringCommand(ctx, ...)
    ```

20. **Line 902-921** - Chapter 4 Stripe create charge
    ```go
    ❌ resp, body, err := gocurl.CurlCommand(...)
    ✅ Option 1: resp, err := gocurl.CurlJSONCommand(ctx, &charge, cmd)
    ✅ Option 2: body, resp, err := gocurl.CurlStringCommand(ctx, cmd)
    ```

21. **Line 938-960** - Chapter 4 GitHub create issue
    ```go
    ❌ resp, respBody, err := gocurl.CurlCommand(...)
    ✅ Option 1: resp, err := gocurl.CurlJSONCommand(ctx, &issue, cmd)
    ✅ Option 2: respBody, resp, err := gocurl.CurlStringCommand(ctx, cmd)
    ```

22. **Line 986-998** - Chapter 4 DevTools converter template
    ```go
    ❌ template := `resp, body, err := gocurl.CurlCommand(...)`
    ✅ Should be: template := `body, resp, err := gocurl.CurlStringCommand(...)`
    ```

**Total outline.md errors: 40+**

---

### ❌ style_guide.md - ERRORS FOUND (3 instances)

1. **Line 437** - Example 1
   ```go
   ❌ resp, body, err := gocurl.CurlCommand(ctx, ...)
   ✅ Should be: body, resp, err := gocurl.CurlStringCommand(ctx, ...)
   ```

2. **Line 441** - Example 2
   ```go
   ❌ resp, body, err := gocurl.CurlCommand(ctx, ...)
   ✅ Should be: body, resp, err := gocurl.CurlStringCommand(ctx, ...)
   ```

3. **Line 447** - Example 3
   ```go
   ❌ resp, body, err := gocurl.CurlCommand(ctx, ...)
   ✅ Should be: body, resp, err := gocurl.CurlStringCommand(ctx, ...)
   ```

**Total style_guide.md errors: 3**

---

### ✅ code_standards.md - FIXED (All 15 instances corrected)

**Status:** ✅ ALL CORRECTED
**Date:** October 17, 2025

All instances now correctly use:
- `CurlString()` for simple examples
- `CurlStringCommand()` for command-style examples
- Proper `(body, resp, err)` return order

---

### ✅ API_REFERENCE_QUICK.md - CORRECT

**Status:** ✅ VERIFIED CORRECT
All signatures documented correctly. No errors found.

---

## CRITICAL ISSUE: Missing Function?

**⚠️ INVESTIGATE:** Does `CurlStringCommandWithVars` exist?

From outline.md line 563:
```go
resp, body, err := gocurl.CurlCommandWithVars(ctx, vars, ...)
```

**Need to check if these exist:**
- ❓ `CurlStringCommandWithVars(ctx, vars, cmd)` → `(string, *http.Response, error)`
- ❓ `CurlStringArgsWithVars(ctx, vars, args...)` → `(string, *http.Response, error)`
- ❓ `CurlBytesCommandWithVars(ctx, vars, cmd)` → `([]byte, *http.Response, error)`
- ❓ `CurlBytesArgsWithVars(ctx, vars, args...)` → `([]byte, *http.Response, error)`

**If they DON'T exist, we need to either:**
1. Use regular `CurlCommandWithVars()` and manually read body
2. Recommend NOT using WithVars in simple examples
3. Request these functions be added to the library

---

## SUMMARY

### Files Status:
- ✅ **code_standards.md**: FIXED (15/15 corrections)
- ✅ **API_REFERENCE_QUICK.md**: CORRECT (0 errors)
- ❌ **outline.md**: NEEDS FIXING (~40 errors)
- ❌ **style_guide.md**: NEEDS FIXING (3 errors)
- ⏳ **__plan.md**: NOT YET AUDITED

### Total Errors:
- **Fixed:** 15 (code_standards.md)
- **Remaining:** 43+ (outline.md + style_guide.md)
- **Unknown:** __plan.md not audited yet

---

## RECOMMENDED FIX STRATEGY

### Priority 1: Fix outline.md (CRITICAL)
The outline is the blueprint for all chapters. Must be 100% correct before writing ANY content.

**Approach:**
1. Use `CurlStringCommand()` for all shell-style examples
2. Use `CurlString()` for simple URL examples
3. Use `CurlJSON()` / `CurlJSONCommand()` for JSON response examples
4. Ensure `(body, resp, err)` order for String/Bytes functions
5. Ensure `(resp, err)` order for JSON functions

### Priority 2: Fix style_guide.md
Only 3 instances - quick fix.

### Priority 3: Audit __plan.md
Verify no code examples there have errors.

### Priority 4: Verify WithVars functions
Check if String/Bytes versions of WithVars functions exist.

---

## ACTION ITEMS

- [ ] Fix all 40+ errors in outline.md
- [ ] Fix 3 errors in style_guide.md
- [ ] Audit __plan.md for any code examples
- [ ] Verify CurlString*WithVars functions exist in API
- [ ] Create test file to verify all corrected examples compile
- [ ] Document any missing API functions needed for book

---

**Created:** October 17, 2025
**Last Updated:** October 17, 2025
**Status:** Audit Complete - Fixes Pending
