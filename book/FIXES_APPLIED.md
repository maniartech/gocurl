# API Corrections Applied to GoCurl Book

**Date:** October 17, 2025
**Issue:** Incorrect API signatures used throughout book documentation
**Status:** ✅ code_standards.md FIXED | ⏳ outline.md PENDING | ⏳ style_guide.md PENDING

---

## The Problem

All examples incorrectly used:
```go
resp, body, err := gocurl.Curl(ctx, url)  // ❌ WRONG! Curl() returns 2 values, not 3
```

## The Correct API

### Basic Functions (Return Response Only - 2 values)
```go
resp, err := gocurl.Curl(ctx, url)              // (*http.Response, error)
resp, err := gocurl.CurlCommand(ctx, cmd)       // (*http.Response, error)
resp, err := gocurl.CurlArgs(ctx, args...)      // (*http.Response, error)
```

### Convenience Functions (Auto-Read Body - 3 values)
```go
body, resp, err := gocurl.CurlString(ctx, url)         // (string, *http.Response, error)
body, resp, err := gocurl.CurlStringCommand(ctx, cmd)  // (string, *http.Response, error)
body, resp, err := gocurl.CurlStringArgs(ctx, args...) // (string, *http.Response, error)

data, resp, err := gocurl.CurlBytes(ctx, url)          // ([]byte, *http.Response, error)

resp, err := gocurl.CurlJSON(ctx, &result, url)        // (*http.Response, error) - decodes into result
```

---

## Files Fixed

### ✅ code_standards.md (COMPLETE)

**Changes Applied:**

1. **Line 13-38:** First example - Changed to `CurlString()`
   - Before: `resp, body, err := gocurl.Curl(...)`
   - After: `body, resp, err := gocurl.CurlString(...)`

2. **Line 40-56:** Error handling pattern - Changed to `CurlString()`
   - Before: `resp, body, err := gocurl.Curl(ctx, url)`
   - After: `body, resp, err := gocurl.CurlString(ctx, url)`

3. **Line 65-90:** Production pattern (GetUser) - Changed to `CurlStringCommand()`
   - Before: `resp, body, err := gocurl.CurlCommand(ctx, ...)`
   - After: `body, resp, err := gocurl.CurlStringCommand(ctx, ...)`

4. **Line 94-97:** Bad example - Changed to `CurlString()`
   - Before: `resp, body, _ := gocurl.Curl(...)`
   - After: `body, _, _ := gocurl.CurlString(...)`

5. **Line 150-180:** Function structure pattern - Changed to `CurlString()`
   - Before: `resp, body, err := gocurl.Curl(ctx, url)`
   - After: `body, resp, err := gocurl.CurlString(ctx, url)`

6. **Line 234-268:** All error handling patterns - Changed to `CurlString()`
   - Basic, Context, and HTTP Status error handling updated

7. **Line 458-477:** Performance patterns - Changed to `CurlString()`
   - Timeout and close body examples updated

8. **Line 620-750:** All 4 common patterns - Changed to `CurlString()` and `CurlStringCommand()`
   - Pattern 1: Simple GET Request
   - Pattern 2: POST with JSON
   - Pattern 3: With Authentication
   - Pattern 4: With Retry

**Total Changes:** 15 code blocks corrected

---

## Files Still Need Fixing

### ⏳ outline.md (~40 instances)

**High Priority Sections:**
- Introduction examples
- Chapter 1: First request examples
- Chapter 2: Setup verification examples
- Chapter 3: All core concept examples
- Chapter 4: Command syntax examples
- Remaining chapters 5-16

**Strategy for outline.md:**
- Simple GET examples → Use `CurlString()`
- JSON response examples → Use `CurlJSON()`
- Complex/production examples → Use `CurlStringCommand()`

### ⏳ style_guide.md (if any)

**Need to check:**
- Code example sections
- Pattern library sections

---

## Correction Guidelines

### When to Use Each Function

**Use `CurlString()` for:**
- ✅ Simple examples in early chapters
- ✅ Quick demonstrations
- ✅ When you just need the body as string
- ✅ Teaching beginners

**Use `CurlJSON()` for:**
- ✅ JSON API examples
- ✅ Production patterns with structured data
- ✅ When you have a target struct
- ✅ Type-safe response handling

**Use `Curl()` for:**
- ✅ Advanced examples showing manual body handling
- ✅ Streaming responses
- ✅ When you need full control over response processing
- ✅ Performance-critical code where you want zero extra allocation

**Use `CurlBytes()` for:**
- ✅ Binary data
- ✅ When you need []byte specifically
- ✅ Image/file downloads before saving

---

## Next Steps

1. ✅ Fix code_standards.md (DONE)
2. ⏳ Fix outline.md (NEXT - ~40 instances)
3. ⏳ Check style_guide.md for errors
4. ⏳ Check __plan.md for errors
5. ⏳ Create updated examples for Chapter 1
6. ⏳ Verify all files compile

---

## Verification Checklist

Before writing ANY chapter:

- [ ] All code_standards.md examples verified
- [ ] All outline.md examples corrected
- [ ] All style_guide.md examples corrected
- [ ] API_REFERENCE_QUICK.md reviewed
- [ ] Test at least 3 examples by running them
- [ ] Create a "verified examples" directory with working code

---

**Remember:** Every code example MUST compile and run. This is non-negotiable for O'Reilly quality.

---

**Last Updated:** October 17, 2025
**Next Review:** After fixing outline.md
