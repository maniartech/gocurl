# RequestOptions Cleanup Plan

**Date**: October 14, 2025
**Objective**: Fix all compliance gaps identified in REQUESTOPTIONS_AUDIT.md before v1.0 release
**Status**: In Progress

## Executive Summary

Based on comprehensive audit, 9 critical actions required to achieve 100% compliance with objective.md SSR philosophy:
- Remove 1 unimplemented field (ResponseDecoder)
- Add 3 missing implementations (thread-safety docs, race tests, ResponseBodyLimit)
- Verify 2 unclear features (HTTP/2, TLSConfig immutability)
- Update all documentation

**Current Status**: 39/40 fields (after ResponseDecoder removal)
**Target**: 100% compliance (all fields Sweet, Simple, Robust)

---

## Priority 1: Critical Fixes (Immediate)

### 1.1 Remove ResponseDecoder ❌ CRITICAL
**Issue**: Field defined but NEVER implemented in code
**Impact**: Documentation lies about features that don't exist
**Curl Compatibility**: No curl equivalent (violates SSR "Sweet")

**Files to Modify**:
- `options/options.go`: Delete field (line 80), delete type (lines 133-136), update Clone() comment (line 172)
- Search all `.md` files for "ResponseDecoder" and remove references

**Verification**:
```bash
grep -r "ResponseDecoder" --include="*.go" --include="*.md" .
# Should only find comments saying "removed in v1.0"
```

### 1.2 Add Thread-Safety Documentation 🔒 CRITICAL
**Issue**: Headers, Form, QueryParams are maps (NOT thread-safe for concurrent writes)
**Impact**: Users may cause race conditions without warnings
**Compliance Gap**: Violates "Robust" (military-grade requires clear safety guarantees)

**Implementation**:
Add to `options/options.go` above RequestOptions struct:

```go
// RequestOptions represents the configuration for an HTTP request in GoCurl.
//
// THREAD-SAFETY GUARANTEES:
//   - SAFE: All primitive fields (string, bool, int, time.Duration) are safe for concurrent reads
//   - SAFE: Struct pointer fields are safe for concurrent reads if not modified
//   - UNSAFE: Headers, Form, QueryParams map writes are NOT safe for concurrent modification
//   - BEST PRACTICE: Use Clone() before concurrent modification to avoid race conditions
//
// Example of safe concurrent usage:
//   opts := options.NewRequestOptions("https://api.example.com")
//   opts.SetHeader("Authorization", "Bearer token")
//
//   // Safe: Clone before concurrent use
//   opts1 := opts.Clone()
//   opts2 := opts.Clone()
//
//   go makeRequest(ctx, opts1) // Safe
//   go makeRequest(ctx, opts2) // Safe
//
// Example of UNSAFE concurrent usage:
//   opts := options.NewRequestOptions("https://api.example.com")
//
//   // RACE CONDITION: Concurrent map writes
//   go opts.AddHeader("X-Request-ID", "1") // ❌ UNSAFE
//   go opts.AddHeader("X-Request-ID", "2") // ❌ UNSAFE
type RequestOptions struct {
```

### 1.3 Create Race Tests 🧪 CRITICAL
**Issue**: No tests verify thread-safety warnings
**Impact**: Can't prove compliance with military-grade requirements

**Create**: `race_concurrent_test.go`

```go
package gocurl

import (
	"net/http"
	"sync"
	"testing"

	"github.com/maniartech/gocurl/options"
)

// TestRequestOptions_ConcurrentCloneIsSafe verifies Clone() prevents races
func TestRequestOptions_ConcurrentCloneIsSafe(t *testing.T) {
	opts := options.NewRequestOptions("https://example.com")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			cloned := opts.Clone()
			cloned.AddHeader("X-Request-ID", fmt.Sprintf("%d", id))
		}(i)
	}
	wg.Wait()
	// Should pass with -race flag
}

// TestRequestOptions_ConcurrentHeaderWrites_DetectsRace verifies race detection
// Run with: go test -race -run TestRequestOptions_ConcurrentHeaderWrites_DetectsRace
func TestRequestOptions_ConcurrentHeaderWrites_DetectsRace(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	opts := options.NewRequestOptions("https://example.com")
	opts.Headers = make(http.Header)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// This SHOULD trigger race detector
			opts.Headers.Add("X-ID", fmt.Sprintf("%d", id))
		}(i)
	}
	wg.Wait()
	// When run with -race, this should FAIL with race condition detected
}
```

---

## Priority 2: Missing Implementations (High)

### 2.1 Implement ResponseBodyLimit ⚠️ HIGH
**Issue**: Field exists (line 78) but NOT enforced in process.go
**Current Code**: `ioutil.ReadAll(resp.Body)` reads unlimited bytes

**Fix**: Update `process.go` line ~93:

```go
// Read the response body with size limit
var bodyBytes []byte
if opts.ResponseBodyLimit > 0 {
	limitedReader := io.LimitReader(resp.Body, opts.ResponseBodyLimit)
	bodyBytes, err = ioutil.ReadAll(limitedReader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %v", err)
	}
	// Check if we hit the limit
	if int64(len(bodyBytes)) >= opts.ResponseBodyLimit {
		resp.Body.Close()
		return nil, "", fmt.Errorf("response body exceeds limit of %d bytes", opts.ResponseBodyLimit)
	}
} else {
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %v", err)
	}
}
```

**Test**: Create test in `process_test.go`:

```go
func TestResponseBodyLimit_EnforcedCorrectly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write 1MB of data
		w.Write(bytes.Repeat([]byte("a"), 1024*1024))
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.ResponseBodyLimit = 1024 // 1KB limit

	_, _, err := Process(context.Background(), opts)
	if err == nil {
		t.Error("Expected error for exceeding body limit")
	}
	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Errorf("Expected limit error, got: %v", err)
	}
}
```

### 2.2 Verify HTTP/2 Implementation ⚠️ HIGH
**Issue**: HTTP2/HTTP2Only flags exist but implementation unclear
**Code Location**: `process.go` lines 191-206

**Action**: Create `http2_test.go`:

```go
func TestHTTP2Support_EnabledCorrectly(t *testing.T) {
	// Requires HTTPS server with HTTP/2
	// Test that opts.HTTP2 = true enables HTTP/2
}

func TestHTTP2Only_RejectsHTTP1(t *testing.T) {
	// Test that opts.HTTP2Only = true rejects HTTP/1.1 connections
}
```

### 2.3 Document TLSConfig Immutability ⚠️ MEDIUM
**Issue**: TLSConfig should not be modified after use (undefined behavior)

**Fix**: Add comment in `options/options.go` line ~33:

```go
// TLS/SSL options
CertFile            string      `json:"cert_file,omitempty"`
KeyFile             string      `json:"key_file,omitempty"`
CAFile              string      `json:"ca_file,omitempty"`
Insecure            bool        `json:"insecure,omitempty"`
TLSConfig           *tls.Config `json:"-"` // Not exported to JSON. WARNING: Do not modify after setting - not safe for concurrent use
```

---

## Priority 3: Documentation Updates (Medium)

### 3.1 Remove ResponseDecoder from All Docs
**Files to Check**:
- `README.md`
- `objective.md`
- `design.md`
- `MIDDLEWARE_VS_DECODER_PATTERNS.md` (likely primary reference)
- `API_QUALITY_ASSESSMENT.md`
- `CLEANUP_SUMMARY.md`

**Search Command**:
```bash
grep -r "ResponseDecoder" --include="*.md" .
```

**Replace With**:
```
Note: ResponseDecoder was removed in v1.0 as it was never implemented.
Use Middleware for custom response processing instead.
```

### 3.2 Update REQUESTOPTIONS_AUDIT.md
**Action**: Mark ResponseDecoder as "REMOVED" and update compliance stats:
- Total fields: 39 (down from 40)
- Compliance: Target 100%

---

## Implementation Order

### Phase 1: Removal (30 minutes)
1. ✅ Remove ResponseDecoder field from options.go
2. ✅ Remove ResponseDecoder type definition
3. ✅ Update Clone() comment
4. ✅ Search and remove doc references

### Phase 2: Critical Safety (1 hour)
5. ✅ Add thread-safety documentation
6. ✅ Create race_concurrent_test.go
7. ✅ Run `go test -race ./...`

### Phase 3: Missing Features (1-2 hours)
8. ✅ Implement ResponseBodyLimit in process.go
9. ✅ Add ResponseBodyLimit test
10. ✅ Verify HTTP/2 implementation
11. ✅ Add HTTP/2 tests (if needed)

### Phase 4: Documentation (30 minutes)
12. ✅ Document TLSConfig immutability
13. ✅ Update all .md files
14. ✅ Update REQUESTOPTIONS_AUDIT.md

### Phase 5: Verification (30 minutes)
15. ✅ Run full test suite
16. ✅ Run race detector
17. ✅ Re-audit all 39 fields
18. ✅ Create compliance summary

**Total Time Estimate**: 3-4 hours

---

## Success Criteria

- [ ] ResponseDecoder completely removed (0 references in code)
- [ ] Thread-safety documentation added
- [ ] Race tests created and passing
- [ ] ResponseBodyLimit implemented and tested
- [ ] HTTP/2 verified and tested
- [ ] TLSConfig immutability documented
- [ ] All tests passing: `go test ./...`
- [ ] No race conditions: `go test -race ./...`
- [ ] 100% field compliance in re-audit
- [ ] All documentation updated

---

## Risk Assessment

**Low Risk**:
- Removing ResponseDecoder (never used)
- Adding documentation (no code changes)
- Adding tests (no breaking changes)

**Medium Risk**:
- ResponseBodyLimit implementation (must not break existing behavior)
- Solution: Only enforce if explicitly set (> 0)

**High Risk**:
- None identified

---

## Rollback Plan

If issues arise:
1. Git branch created before changes
2. Each phase committed separately
3. Can rollback phase-by-phase
4. All tests must pass before merge

---

## Next Steps

1. User approval of plan
2. Execute Phase 1 (Removal)
3. Execute Phase 2 (Safety)
4. Execute Phase 3 (Features)
5. Execute Phase 4 (Docs)
6. Execute Phase 5 (Verification)
7. Create REQUESTOPTIONS_COMPLIANCE_FINAL.md

**Ready to proceed?**
