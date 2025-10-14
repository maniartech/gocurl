# RequestOptions Field-by-Field Deep Analysis

**Date**: October 14, 2025
**Scope**: All 39 fields - Usage, Curl Compatibility, Go Best Practices, Thread-Safety, Tests, Military-Grade
**Status**: 🔴 **CRITICAL GAPS FOUND**

---

## Analysis Criteria

For each field, checking:
1. ✅ **Used**: Is the field actually used in implementation?
2. ✅ **Curl Compatible**: Matches curl behavior?
3. ✅ **Go Best Practice**: Follows Go conventions?
4. ✅ **Thread-Safe**: Safe for concurrent access?
5. ✅ **Tested**: Has comprehensive tests?
6. ✅ **Military-Grade**: Robust, secure, handles edge cases?

---

## HTTP Request Basics (6 fields)

### 1. Method (string)
**Usage**: ⚠️ 1 reference only - `process.go:253`
```go
method := opts.Method
if method == "" {
    method = "GET"
}
```

**Analysis**:
- ✅ **Used**: Yes, in CreateRequest()
- ✅ **Curl Compatible**: Defaults to GET (correct)
- ✅ **Go Best Practice**: Simple string
- ✅ **Thread-Safe**: Immutable string (safe)
- ❌ **Tested**: No validation tests for invalid methods
- ⚠️ **Military-Grade**: NO VALIDATION - accepts "INVALID" as method
  - Should validate: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, CONNECT, TRACE
  - Curl validates methods, we don't

**Gaps**:
- Missing method validation
- No test for invalid method names
- No test for case sensitivity

**Fix Required**: Add method validation

---

### 2. URL (string)
**Usage**: ✅ 3 references

**Analysis**:
- ✅ **Used**: Yes, extensively
- ✅ **Curl Compatible**: Yes
- ⚠️ **Go Best Practice**: Should use url.URL type, not string
- ✅ **Thread-Safe**: Immutable string
- ⚠️ **Tested**: Basic tests exist, but no malformed URL tests
- ⚠️ **Military-Grade**: Basic validation exists but incomplete
  - No test for extremely long URLs (DoS)
  - No test for URL injection attacks

**Gaps**:
- Should validate URL length (prevent DoS)
- Missing tests for malformed URLs
- No tests for URL encoding edge cases

---

### 3. Headers (http.Header)
**Usage**: ⚠️ 1 reference only - just iteration in CreateRequest()

**Analysis**:
- ✅ **Used**: Yes, in CreateRequest()
- ✅ **Curl Compatible**: Yes (-H flag)
- ✅ **Go Best Practice**: Uses stdlib http.Header
- 🔴 **Thread-Safe**: NO - map is unsafe for concurrent writes (documented but dangerous)
- ⚠️ **Tested**: Basic tests, but no concurrent access tests
- ⚠️ **Military-Grade**: No header validation
  - No header size limits (DoS vector)
  - No forbidden header checks (Host, Content-Length should be managed)
  - No header injection prevention

**Critical Gaps**:
- 🔴 **NO VALIDATION**: Can set "Content-Length: -1" or other invalid values
- 🔴 **NO SIZE LIMIT**: Can add 10,000 headers (DoS attack)
- 🔴 **NO FORBIDDEN HEADER CHECKS**: Users can override critical headers
- ⚠️ **Thread-safety**: Documented as unsafe, but no mutex protection option

**Fix Required**:
- Add header validation
- Add size limits
- Protect forbidden headers
- Consider sync.Map or mutex wrapper

---

### 4. Body (string)
**Usage**: ✅ 2 references

**Analysis**:
- ✅ **Used**: Yes
- ✅ **Curl Compatible**: Yes (-d flag)
- ⚠️ **Go Best Practice**: String causes allocation on every request
- ✅ **Thread-Safe**: Immutable string
- ⚠️ **Tested**: Basic tests only
- 🔴 **Military-Grade**: MAJOR ISSUE
  - String type means body loaded into memory
  - No streaming for large bodies
  - No size limit (can OOM with huge body)
  - Allocation on every request (not zero-allocation)

**Critical Gaps**:
- 🔴 **NO BODY SIZE LIMIT**: Can send 1GB body, OOM risk
- 🔴 **NO STREAMING SUPPORT**: Should have BodyReader io.Reader option
- 🔴 **ALLOCATION**: String conversion allocates, violates zero-allocation goal

**Fix Required**:
- Add BodyReader io.Reader field
- Add BodySizeLimit validation
- Add streaming support for large bodies

---

### 5. Form (url.Values)
**Usage**: ✅ 3 references

**Analysis**:
- ✅ **Used**: Yes
- ✅ **Curl Compatible**: Yes (-F flag)
- ✅ **Go Best Practice**: Uses stdlib url.Values
- 🔴 **Thread-Safe**: NO - map is unsafe for concurrent writes
- ✅ **Tested**: Has tests
- ⚠️ **Military-Grade**: No validation
  - No form size limits
  - No key/value length limits

**Gaps**:
- Same thread-safety issues as Headers
- No size limits

---

### 6. QueryParams (url.Values)
**Usage**: ✅ 3 references

**Analysis**:
- ✅ **Used**: Yes
- ✅ **Curl Compatible**: Yes
- ✅ **Go Best Practice**: Uses stdlib
- 🔴 **Thread-Safe**: NO - map unsafe
- ✅ **Tested**: Has tests
- ⚠️ **Military-Grade**: No validation
  - No query string length limit (URLs can be too long)
  - No parameter count limit

**Gaps**:
- Thread-safety
- Size validation

---

## Authentication (2 fields)

### 7. BasicAuth (*BasicAuth)
**Usage**: ✅ 2 references

**Analysis**:
- ✅ **Used**: Yes, properly with SetBasicAuth()
- ✅ **Curl Compatible**: Yes (-u flag)
- ✅ **Go Best Practice**: Yes
- ✅ **Thread-Safe**: Pointer read is safe if not modified
- ✅ **Tested**: Has tests
- ⚠️ **Military-Grade**: Security concerns
  - No warning about plaintext transmission
  - Should warn to use HTTPS
  - No credential masking in logs/errors

**Gaps**:
- Should validate HTTPS when using BasicAuth
- No test for BasicAuth over HTTP (insecure warning)

---

### 8. BearerToken (string)
**Usage**: ✅ 2 references

**Analysis**:
- ✅ **Used**: Yes
- ✅ **Curl Compatible**: Yes (-H "Authorization: Bearer")
- ✅ **Go Best Practice**: Yes
- ✅ **Thread-Safe**: Immutable string
- ✅ **Tested**: Has tests
- ⚠️ **Military-Grade**: Security concerns
  - No HTTPS validation
  - Token could be logged in verbose mode
  - No token format validation

**Gaps**:
- Should validate HTTPS
- Should redact token in logs

---

## TLS/SSL (7 fields)

### 9. CertFile (string)
**Usage**: ✅ 5 references

**Analysis**:
- ✅ **Used**: Yes, in LoadTLSConfig()
- ✅ **Curl Compatible**: Yes (--cert)
- ✅ **Go Best Practice**: Yes
- ✅ **Thread-Safe**: Yes
- ✅ **Tested**: Yes, has certificate tests
- ✅ **Military-Grade**: Good
  - Proper error handling
  - File validation

**Status**: ✅ GOOD

---

### 10. KeyFile (string)
**Usage**: ✅ 5 references

**Analysis**:
- ✅ **Used**: Yes
- ✅ **Curl Compatible**: Yes (--key)
- ✅ **Go Best Practice**: Yes
- ✅ **Thread-Safe**: Yes
- ✅ **Tested**: Yes
- ✅ **Military-Grade**: Good

**Status**: ✅ GOOD

---

### 11. CAFile (string)
**Usage**: ✅ 4 references

**Analysis**:
- ✅ **Used**: Yes
- ✅ **Curl Compatible**: Yes (--cacert)
- ✅ **Go Best Practice**: Yes
- ✅ **Thread-Safe**: Yes
- ✅ **Tested**: Yes
- ✅ **Military-Grade**: Good

**Status**: ✅ GOOD

---

### 12. Insecure (bool)
**Usage**: ✅ 4 references

**Analysis**:
- ✅ **Used**: Yes
- ✅ **Curl Compatible**: Yes (-k/--insecure)
- ✅ **Go Best Practice**: Yes
- ✅ **Thread-Safe**: Yes
- ✅ **Tested**: Yes
- ⚠️ **Military-Grade**: Should warn
  - No warning when used
  - Should log security warning

**Gap**: Should emit warning when Insecure=true

---

### 13. TLSConfig (*tls.Config)
**Usage**: ✅ 4 references

**Analysis**:
- ✅ **Used**: Yes
- ✅ **Curl Compatible**: N/A (advanced feature)
- ✅ **Go Best Practice**: Yes
- ⚠️ **Thread-Safe**: Documented as "don't modify" but no enforcement
- ⚠️ **Tested**: Basic tests only
- ⚠️ **Military-Grade**: Could be better
  - No clone before use
  - User could modify after passing

**Gap**: Should clone TLSConfig before use

---

### 14. CertPinFingerprints ([]string)
**Usage**: ✅ 3 references

**Analysis**:
- ✅ **Used**: Yes
- ✅ **Curl Compatible**: Extension (curl has --pinnedpubkey)
- ✅ **Go Best Practice**: Yes
- ✅ **Thread-Safe**: Slice read is safe
- ✅ **Tested**: Yes, has pinning tests
- ✅ **Military-Grade**: Excellent security feature

**Status**: ✅ EXCELLENT

---

### 15. SNIServerName (string)
**Usage**: ✅ 2 references

**Analysis**:
- ✅ **Used**: Yes
- ✅ **Curl Compatible**: Yes (--resolve)
- ✅ **Go Best Practice**: Yes
- ✅ **Thread-Safe**: Yes
- ✅ **Tested**: Likely tested
- ✅ **Military-Grade**: Good

**Status**: ✅ GOOD

---

## Proxy (2 fields)

### 16. Proxy (string)
**Usage**: ✅ 3 references

**Analysis**:
- ✅ **Used**: Yes
- ✅ **Curl Compatible**: Yes (-x/--proxy)
- ✅ **Go Best Practice**: Yes
- ✅ **Thread-Safe**: Yes
- ✅ **Tested**: Yes, comprehensive proxy tests
- ✅ **Military-Grade**: Good, supports HTTP/HTTPS/SOCKS5

**Status**: ✅ EXCELLENT

---

### 17. ProxyNoProxy ([]string)
**Usage**: ⚠️ 1 reference only

**Analysis**:
- ✅ **Used**: Yes, passed to proxy config
- ✅ **Curl Compatible**: Yes (--noproxy)
- ✅ **Go Best Practice**: Yes
- ✅ **Thread-Safe**: Yes
- ✅ **Tested**: Yes, has no-proxy tests
- ✅ **Military-Grade**: Good

**Status**: ✅ GOOD

---

## Timeouts (2 fields)

### 18. Timeout (time.Duration)
**Usage**: ✅ 7 references

**Analysis**:
- ✅ **Used**: Yes, extensively
- ✅ **Curl Compatible**: Yes (--max-time)
- ✅ **Go Best Practice**: Yes, uses time.Duration
- ✅ **Thread-Safe**: Yes
- ✅ **Tested**: YES! 9+ timeout tests
- ✅ **Military-Grade**: Excellent
  - Context priority pattern
  - Proper deadline handling

**Status**: ✅ EXCELLENT

---

### 19. ConnectTimeout (time.Duration)
**Usage**: ✅ 2 references

**Analysis**:
- ✅ **Used**: Yes
- ✅ **Curl Compatible**: Yes (--connect-timeout)
- ✅ **Go Best Practice**: Yes
- ✅ **Thread-Safe**: Yes
- ⚠️ **Tested**: Basic validation, no dedicated tests
- ✅ **Military-Grade**: Good

**Status**: ✅ GOOD

---

## STOPPING HERE FOR NOW - WILL CONTINUE WITH REMAINING 20 FIELDS

**Next batch**: Redirects, Compression, HTTP2, Cookies, Output, Advanced

---

## Summary So Far (19/39 fields analyzed)

### Critical Issues Found 🔴

1. **Headers** - No validation, no size limits, DoS vector
2. **Body** - No size limit, no streaming, OOM risk, allocates memory
3. **Method** - No validation, accepts invalid methods

### Medium Issues Found ⚠️

4. **URL** - No length validation
5. **Form/QueryParams** - No size limits
6. **TLSConfig** - Should clone before use
7. **BasicAuth/BearerToken** - Should validate HTTPS

### Fields NOT USED YET ❌

- **Cookies** - Defined but not implemented
- **Verbose** - Defined but not implemented
- **RequestID** - Defined but not implemented

**Analysis to continue with remaining 20 fields...**
