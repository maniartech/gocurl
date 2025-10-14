# GoCurl Implementation Fix Plan

**Date**: October 14, 2025
**Philosophy**: Sweet, Simple, Robust (SSR)
**Status**: 🔴 CRITICAL - Production Blocker
**Target**: Military-Grade Quality (95%+ compliance)

---

## Executive Summary

**Current State**: 61.5% implementation quality (24/39 fields properly implemented)
**Target State**: 95%+ military-grade quality
**Estimated Time**: 5-7 days
**Blockers**: 3 fields completely unused, 4 fields with critical security/DoS vectors

**Risk Assessment**:
- **HIGH**: Silent failures (Cookies, Verbose, RequestID never used)
- **CRITICAL**: DoS vectors (Headers, Body, URL no limits)
- **CRITICAL**: Security gaps (Method, Headers no validation)

---

## Guiding Principles (SSR)

### Sweet (Developer Experience)
- ✅ Clear error messages: "invalid HTTP method: HACK" not "request failed"
- ✅ Fail fast: Validate at construction time, not execution time
- ✅ Helpful warnings: Log security warnings for Insecure=true, HTTP+Auth

### Simple (Implementation)
- ✅ No over-engineering: Add validation, don't create validation frameworks
- ✅ Use stdlib: Leverage existing validators where possible
- ✅ Clear flow: Validate → Construct → Execute

### Robust (Military-Grade)
- ✅ Zero allocation preserved: Pool validation buffers
- ✅ Thread-safe: All validation is concurrent-safe
- ✅ Comprehensive tests: Every validation path tested
- ✅ Fuzz tested: Parsers withstand malformed input

---

## Priority 1: CRITICAL FIXES (Must Complete Before v1.0)

**Estimated Time**: 3-4 days
**Goal**: Fix silent failures and security vulnerabilities

### 1.1 Implement Missing Fields (Day 1: 4 hours)

#### Task 1.1.1: Implement Cookies Field
**File**: `process.go`
**Location**: After line 346 (after Referer)
**Estimated Time**: 30 minutes

**Implementation**:
```go
// Apply cookies from Cookies field
for _, cookie := range opts.Cookies {
    req.AddCookie(cookie)
}
```

**Tests Required** (`cookie_test.go`):
- TestCookies_SingleCookie
- TestCookies_MultipleCookies
- TestCookies_EmptyArray
- TestCookies_WithCookieJar (interaction test)
- TestCookies_ConcurrentSafe

**Validation**:
- ✅ Curl compatible: Yes (-b flag)
- ✅ Thread-safe: Read-only operation
- ✅ Zero-alloc: Cookie already allocated

---

#### Task 1.1.2: Implement Verbose Field
**File**: New file `verbose.go`
**Estimated Time**: 2-3 hours

**Design Philosophy (Sweet + Simple)**:
- Match curl -v output format exactly
- Write to stderr by default (like curl)
- Support custom io.Writer for testing
- No buffering - real-time output

**Implementation Structure**:
```go
package gocurl

import (
    "fmt"
    "io"
    "net/http"
    "os"
)

// VerboseWriter controls verbose output
var VerboseWriter io.Writer = os.Stderr

// printVerbose writes verbose output if enabled
func printVerbose(opts *options.RequestOptions, format string, args ...interface{}) {
    if opts.Verbose {
        fmt.Fprintf(VerboseWriter, format, args...)
    }
}

// printRequestVerbose prints request details (curl -v style)
func printRequestVerbose(opts *options.RequestOptions, req *http.Request) {
    if !opts.Verbose {
        return
    }

    // Connection info
    fmt.Fprintf(VerboseWriter, "* Trying %s...\n", req.URL.Host)

    // Request line
    fmt.Fprintf(VerboseWriter, "> %s %s %s\n", req.Method, req.URL.Path, req.Proto)

    // Request headers
    fmt.Fprintf(VerboseWriter, "> Host: %s\n", req.Host)
    for key, values := range req.Header {
        for _, value := range values {
            // Redact sensitive headers
            if key == "Authorization" || key == "Cookie" {
                fmt.Fprintf(VerboseWriter, "> %s: [REDACTED]\n", key)
            } else {
                fmt.Fprintf(VerboseWriter, "> %s: %s\n", key, value)
            }
        }
    }
    fmt.Fprintf(VerboseWriter, ">\n")
}

// printResponseVerbose prints response details
func printResponseVerbose(opts *options.RequestOptions, resp *http.Response) {
    if !opts.Verbose {
        return
    }

    // Status line
    fmt.Fprintf(VerboseWriter, "< %s %s\n", resp.Proto, resp.Status)

    // Response headers
    for key, values := range resp.Header {
        for _, value := range values {
            fmt.Fprintf(VerboseWriter, "< %s: %s\n", key, value)
        }
    }
    fmt.Fprintf(VerboseWriter, "<\n")
}
```

**Integration Points**:
1. `process.go:CreateRequest()` - after request creation, before return
2. `process.go:Execute()` - after receiving response

**Tests Required** (`verbose_test.go`):
- TestVerbose_Disabled (default)
- TestVerbose_RequestHeaders
- TestVerbose_ResponseHeaders
- TestVerbose_SensitiveDataRedacted (Authorization, Cookie)
- TestVerbose_CustomWriter
- TestVerbose_ConcurrentSafe
- TestVerbose_MatchesCurlFormat

**Validation**:
- ✅ Curl compatible: Yes (-v flag)
- ✅ Thread-safe: Only reads opts.Verbose
- ✅ Sweet: Redacts sensitive data
- ✅ Simple: No buffering, direct write

---

#### Task 1.1.3: Implement RequestID Field
**File**: `process.go`
**Location**: After line 346 (after Referer)
**Estimated Time**: 15 minutes

**Implementation**:
```go
// Add Request ID header for distributed tracing
if opts.RequestID != "" {
    req.Header.Set("X-Request-ID", opts.RequestID)
}
```

**Tests Required** (`request_id_test.go`):
- TestRequestID_Added
- TestRequestID_Empty (not added)
- TestRequestID_UUIDFormat
- TestRequestID_ConcurrentSafe

**Validation**:
- ✅ Industry standard: X-Request-ID
- ✅ Thread-safe: String is immutable
- ✅ Zero-alloc: String already allocated

---

### 1.2 Add Input Validation (Day 1-2: 6 hours)

#### Task 1.2.1: Add Method Validation
**File**: New file `validation.go`
**Estimated Time**: 1 hour

**Design Philosophy (Simple + Robust)**:
- Validate at NewRequestOptions() time (fail fast)
- Use map for O(1) lookup
- Clear error messages

**Implementation**:
```go
package gocurl

import (
    "fmt"
    "net/http"
)

// validHTTPMethods defines allowed HTTP methods
var validHTTPMethods = map[string]bool{
    http.MethodGet:     true,
    http.MethodPost:    true,
    http.MethodPut:     true,
    http.MethodDelete:  true,
    http.MethodPatch:   true,
    http.MethodHead:    true,
    http.MethodOptions: true,
    http.MethodConnect: true,
    http.MethodTrace:   true,
}

// Validation constants (configurable for testing)
const (
    MaxURLLength     = 8192  // 8KB
    MaxHeaders       = 100
    MaxHeaderSize    = 8192  // 8KB
    MaxBodySize      = 10 * 1024 * 1024  // 10MB default
    MaxFormFields    = 1000
    MaxQueryParams   = 1000
)

// validateMethod checks if HTTP method is valid
func validateMethod(method string) error {
    if method == "" {
        return nil // Empty is OK, defaults to GET
    }

    if !validHTTPMethods[method] {
        return fmt.Errorf("invalid HTTP method: %s (allowed: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, CONNECT, TRACE)", method)
    }

    return nil
}

// validateURL checks URL length
func validateURL(url string) error {
    if len(url) > MaxURLLength {
        return fmt.Errorf("URL too long: %d bytes (max: %d)", len(url), MaxURLLength)
    }
    return nil
}

// validateHeaders checks header count and sizes
func validateHeaders(headers http.Header) error {
    if len(headers) > MaxHeaders {
        return fmt.Errorf("too many headers: %d (max: %d)", len(headers), MaxHeaders)
    }

    for key, values := range headers {
        // Check forbidden headers
        if isForbiddenHeader(key) {
            return fmt.Errorf("cannot set forbidden header: %s (managed automatically)", key)
        }

        // Check header size
        for _, value := range values {
            if len(key)+len(value) > MaxHeaderSize {
                return fmt.Errorf("header too large: %s (max: %d bytes)", key, MaxHeaderSize)
            }
        }
    }

    return nil
}

// isForbiddenHeader checks if header should be managed automatically
func isForbiddenHeader(key string) bool {
    forbidden := []string{
        "Host",            // Managed by http.Request
        "Content-Length",  // Calculated automatically
        "Transfer-Encoding", // Managed by transport
    }

    for _, f := range forbidden {
        if http.CanonicalHeaderKey(key) == f {
            return true
        }
    }
    return false
}

// validateBody checks body size
func validateBody(body string, limit int64) error {
    if limit <= 0 {
        limit = MaxBodySize
    }

    if int64(len(body)) > limit {
        return fmt.Errorf("body too large: %d bytes (max: %d)", len(body), limit)
    }

    return nil
}

// validateForm checks form field count
func validateForm(form map[string][]string) error {
    count := 0
    for _, values := range form {
        count += len(values)
    }

    if count > MaxFormFields {
        return fmt.Errorf("too many form fields: %d (max: %d)", count, MaxFormFields)
    }

    return nil
}

// validateQueryParams checks query parameter count
func validateQueryParams(params map[string][]string) error {
    count := 0
    for _, values := range params {
        count += len(values)
    }

    if count > MaxQueryParams {
        return fmt.Errorf("too many query parameters: %d (max: %d)", count, MaxQueryParams)
    }

    return nil
}
```

**Integration Point**:
- `options/builder.go` - Add Validate() method called before Execute()
- `process.go:Execute()` - Call opts.Validate() first

**Tests Required** (`validation_test.go`):
- TestValidateMethod_Valid (GET, POST, etc.)
- TestValidateMethod_Invalid ("HACK", "invalid")
- TestValidateMethod_Empty (should pass)
- TestValidateURL_TooLong
- TestValidateHeaders_TooMany
- TestValidateHeaders_TooLarge
- TestValidateHeaders_Forbidden (Host, Content-Length)
- TestValidateBody_TooLarge
- TestValidateForm_TooManyFields
- TestValidateQueryParams_TooMany
- TestValidation_ConcurrentSafe

---

#### Task 1.2.2: Add Validation to Builder
**File**: `options/builder.go`
**Estimated Time**: 30 minutes

**Implementation**:
```go
// Validate checks all fields for validity
func (b *RequestOptionsBuilder) Validate() error {
    opts := b.opts

    // Validate method
    if err := validateMethod(opts.Method); err != nil {
        return err
    }

    // Validate URL
    if err := validateURL(opts.URL); err != nil {
        return err
    }

    // Validate headers
    if err := validateHeaders(opts.Headers); err != nil {
        return err
    }

    // Validate body
    if opts.Body != "" {
        if err := validateBody(opts.Body, opts.ResponseBodyLimit); err != nil {
            return err
        }
    }

    // Validate form
    if len(opts.Form) > 0 {
        if err := validateForm(opts.Form); err != nil {
            return err
        }
    }

    // Validate query params
    if len(opts.QueryParams) > 0 {
        if err := validateQueryParams(opts.QueryParams); err != nil {
            return err
        }
    }

    return nil
}

// Execute validates then executes the request
func (b *RequestOptionsBuilder) Execute() (*Response, string, error) {
    // Validate first (fail fast)
    if err := b.Validate(); err != nil {
        return nil, "", fmt.Errorf("validation failed: %w", err)
    }

    // Execute
    return Execute(b.opts)
}
```

---

### 1.3 Add Security Warnings (Day 2: 2 hours)

#### Task 1.3.1: Add HTTPS Validation for Auth
**File**: `validation.go`
**Estimated Time**: 1 hour

**Implementation**:
```go
// validateSecureAuth warns about insecure authentication
func validateSecureAuth(opts *options.RequestOptions) error {
    url := opts.URL

    // Check if using HTTP (not HTTPS)
    isHTTP := strings.HasPrefix(url, "http://")

    // Warn about BasicAuth over HTTP
    if isHTTP && opts.BasicAuth != nil {
        return fmt.Errorf("security warning: BasicAuth over HTTP is insecure (credentials sent in plaintext). Use HTTPS or set GOCURL_ALLOW_INSECURE_AUTH=1")
    }

    // Warn about BearerToken over HTTP
    if isHTTP && opts.BearerToken != "" {
        return fmt.Errorf("security warning: Bearer token over HTTP is insecure. Use HTTPS or set GOCURL_ALLOW_INSECURE_AUTH=1")
    }

    return nil
}
```

**Tests Required** (`security_validation_test.go`):
- TestSecureAuth_BasicAuthHTTPS (pass)
- TestSecureAuth_BasicAuthHTTP (fail)
- TestSecureAuth_BearerTokenHTTPS (pass)
- TestSecureAuth_BearerTokenHTTP (fail)
- TestSecureAuth_OverrideEnvVar

---

#### Task 1.3.2: Add TLS Insecure Warning
**File**: `process.go`
**Estimated Time**: 30 minutes

**Implementation**:
```go
// In LoadTLSConfig or CreateRequest
if opts.Insecure {
    fmt.Fprintf(os.Stderr, "WARNING: TLS certificate verification disabled (--insecure). This is INSECURE.\n")
}
```

---

#### Task 1.3.3: Clone TLSConfig
**File**: `process.go`
**Estimated Time**: 30 minutes

**Implementation**:
```go
// Clone TLSConfig to prevent user modification after passing
if opts.TLSConfig != nil {
    tlsConfig = opts.TLSConfig.Clone()
}
```

**Tests Required**:
- TestTLSConfig_ClonedNotModified
- TestTLSConfig_UserModificationDoesNotAffect

---

## Priority 2: HIGH PRIORITY (Should Complete)

**Estimated Time**: 2 days
**Goal**: Improve robustness and test coverage

### 2.1 Add Comprehensive Tests (Day 3-4: 1.5 days)

#### Task 2.1.1: DoS Attack Tests
**File**: New file `dos_protection_test.go`
**Estimated Time**: 4 hours

**Tests**:
- TestDoSProtection_10kHeaders
- TestDoSProtection_100kURL
- TestDoSProtection_1GBBody
- TestDoSProtection_10kFormFields
- TestDoSProtection_10kQueryParams

---

#### Task 2.1.2: Malformed Input Tests
**File**: New file `malformed_input_test.go`
**Estimated Time**: 3 hours

**Tests**:
- TestMalformedInput_InvalidMethod
- TestMalformedInput_InvalidHeaders
- TestMalformedInput_NegativeContentLength
- TestMalformedInput_InvalidURL

---

#### Task 2.1.3: Edge Case Tests
**File**: New file `edge_cases_test.go`
**Estimated Time**: 3 hours

**Tests**:
- TestEdgeCase_EmptyBody
- TestEdgeCase_EmptyHeaders
- TestEdgeCase_MaxValidInput
- TestEdgeCase_UnicodeInHeaders
- TestEdgeCase_SpecialCharactersInURL

---

### 2.2 Add Fuzzing (Day 4: 4 hours)

#### Task 2.2.1: Fuzz Method Validation
**File**: `validation_fuzz_test.go`

```go
func FuzzValidateMethod(f *testing.F) {
    f.Add("GET")
    f.Add("INVALID")
    f.Add("")

    f.Fuzz(func(t *testing.T, method string) {
        err := validateMethod(method)
        // Should never panic
        _ = err
    })
}
```

---

## Priority 3: POLISH (Nice to Have)

**Estimated Time**: 1 day
**Goal**: Developer experience improvements

### 3.1 Add BodyReader Field (Day 5: 4 hours)

**File**: `options/options.go`

```go
// Add field
BodyReader io.Reader  // For streaming large bodies

// In process.go, prefer BodyReader over Body
var bodyReader io.Reader
if opts.BodyReader != nil {
    bodyReader = opts.BodyReader
} else if opts.Body != "" {
    bodyReader = strings.NewReader(opts.Body)
}
```

---

### 3.2 Improve Error Messages (Day 5: 2 hours)

**Enhancement**: Add request context to all errors

```go
return fmt.Errorf("request to %s failed: %w", opts.URL, err)
```

---

### 3.3 Add Validation Benchmarks (Day 5: 2 hours)

**File**: `validation_bench_test.go`

Ensure validation is zero-allocation:
- BenchmarkValidateMethod
- BenchmarkValidateHeaders
- BenchmarkValidateAll

---

## Implementation Schedule

### Week 1: Critical Fixes

| Day | Tasks | Hours | Deliverable |
|-----|-------|-------|-------------|
| **Day 1** | Implement Cookies, Verbose, RequestID | 4 | Missing fields work |
| **Day 1-2** | Add validation (Method, URL, Headers, Body, Form, QueryParams) | 6 | DoS protection complete |
| **Day 2** | Add security warnings (Auth HTTPS, Insecure, TLSConfig clone) | 2 | Security hardened |
| **Day 3-4** | Add comprehensive tests (DoS, malformed, edge cases) | 10 | Test coverage 90%+ |
| **Day 4** | Add fuzzing tests | 4 | Fuzz resistant |
| **Day 5** | Polish (BodyReader, error messages, benchmarks) | 8 | Production ready |

**Total**: 5 days (34 hours)

---

## Testing Strategy

### Test Coverage Goals

- **Unit Tests**: 95%+ coverage on validation.go, verbose.go
- **Integration Tests**: All 39 fields tested in realistic scenarios
- **Race Tests**: All concurrent access patterns verified
- **Fuzz Tests**: 1M+ iterations on all parsers/validators
- **DoS Tests**: Verify limits work under attack
- **Load Tests**: 24-hour soak test at 10k req/s

### Test Execution

```bash
# Unit tests
go test ./... -v

# Race detector
go test ./... -race -count=100

# Fuzzing (run for 1 minute each)
go test -fuzz=FuzzValidateMethod -fuzztime=1m
go test -fuzz=FuzzValidateHeaders -fuzztime=1m

# Benchmarks (verify zero-alloc)
go test -bench=. -benchmem

# Coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Acceptance Criteria

### Functionality

- ✅ All 39 fields properly implemented
- ✅ Cookies applied to requests
- ✅ Verbose outputs curl -v style info
- ✅ RequestID added to headers
- ✅ All validation working

### Security

- ✅ No DoS vectors (all limits enforced)
- ✅ No invalid input accepted
- ✅ Auth over HTTPS validated
- ✅ Insecure mode warns
- ✅ TLSConfig cloned

### Performance

- ✅ Zero allocation preserved on critical path
- ✅ Validation adds < 1μs overhead
- ✅ No memory leaks under load
- ✅ Race-free under concurrent access

### Quality

- ✅ 95%+ test coverage
- ✅ All tests passing
- ✅ Race detector clean
- ✅ Fuzz tests pass 1M+ iterations
- ✅ 24-hour soak test stable

---

## Rollout Plan

### Phase 1: Fix and Test (Days 1-5)
- Implement all fixes
- Write all tests
- Run comprehensive test suite

### Phase 2: Review and Validate (Day 6)
- Code review
- Security audit
- Performance benchmarks
- Update documentation

### Phase 3: Integration Testing (Day 7)
- Test with real APIs (GitHub, Stripe, etc.)
- Verify curl compatibility
- CLI testing
- Load testing

### Phase 4: Release Prep
- Update CHANGELOG.md
- Update README.md
- Tag v1.0.0-rc1
- Community testing

---

## Risk Mitigation

### Risk 1: Breaking Changes
**Mitigation**: Add validation behind feature flag initially
```go
// Allow gradual adoption
if os.Getenv("GOCURL_STRICT_VALIDATION") != "" {
    if err := opts.Validate(); err != nil {
        return nil, "", err
    }
}
```

### Risk 2: Performance Regression
**Mitigation**: Benchmark before and after, ensure < 1μs overhead

### Risk 3: False Positives
**Mitigation**: Make limits configurable
```go
// Allow override for edge cases
if opts.MaxHeaders > 0 {
    maxHeaders = opts.MaxHeaders
} else {
    maxHeaders = DefaultMaxHeaders
}
```

---

## Success Metrics

### Before Fix
- ❌ 61.5% implementation quality
- ❌ 3 fields completely unused
- ❌ 4 critical security gaps
- ❌ Multiple DoS vectors
- ❌ NOT production ready

### After Fix
- ✅ 95%+ implementation quality
- ✅ All 39 fields working
- ✅ Zero security vulnerabilities
- ✅ Zero DoS vectors
- ✅ Military-grade production ready
- ✅ Ready for v1.0.0 release

---

## Documentation Updates Required

1. **README.md**: Update with validation rules
2. **SECURITY.md**: Document security features
3. **VALIDATION.md**: New doc explaining all limits
4. **MIGRATION.md**: Guide for upgrading (if breaking changes)
5. **objective.md**: Update compliance status to 95%+

---

## Final Checklist

- [ ] All 39 fields implemented
- [ ] All validation working
- [ ] All security warnings added
- [ ] 95%+ test coverage
- [ ] Race detector clean
- [ ] Fuzz tests passing
- [ ] Benchmarks show zero-alloc
- [ ] DoS tests passing
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Ready for v1.0.0

**Current Status**: 61.5% → **Target**: 95%+
**Timeline**: 5-7 days
**Blocker Status**: CRITICAL → RESOLVED
