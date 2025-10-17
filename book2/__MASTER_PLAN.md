# HTTP Mastery with Go: From cURL to Production
## Master Plan - The Definitive Guide to the GoCurl Library

**Date Created:** October 17, 2025
**Status:** 🚀 Comprehensive & Authoritative
**Approach:** Complete coverage of all gocurl features

---

## 📖 BOOK VISION

**Title:** HTTP Mastery with Go: From cURL to Production
**Subtitle:** The Definitive Guide to the GoCurl Library

**Unique Value Proposition:**
- **The definitive guide** to the gocurl library
- **Dual approach:** CLI-syntax + Programmatic patterns
- **100% API coverage:** Every function, every feature documented
- **Production-ready:** Security, performance, testing, enterprise patterns
- **O'Reilly quality:** Professional, practical, battle-tested

---

## 📊 SCOPE & METRICS

### Coverage Goals:
- ✅ **100% API Coverage** - All public functions documented
- ✅ **Dual Approaches** - Curl syntax AND RequestOptions/Builder
- ✅ **Production Patterns** - Middleware, tracing, custom clients
- ✅ **Security Complete** - TLS, pinning, mutual auth, validation
- ✅ **Legacy Migration** - Deprecated API → Modern API

### Book Metrics:
- **Total Chapters:** 19
- **Appendices:** 6
- **Target Pages:** 510-530
- **Code Examples:** 300+
- **Hands-On Projects:** 20+

### Quality Standards:
- **All code MUST compile** (tested with `go build`)
- **All examples MUST use correct API signatures** (verified from source)
- **No placeholder code** - only working, tested examples
- **Race-condition free** - tested with `-race` flag
- **Each example in own directory** - prevents main/struct conflicts
- **Simple structure** - no go.mod overhead, just runnable code

### Example Structure Convention:
```
chapter-XX-name/
├── chapter.md
├── examples/
│   ├── README.md
│   ├── 01-example-name/
│   │   └── main.go
│   ├── 02-another-example/
│   │   └── main.go
│   └── ...
└── exercises/
    ├── README.md
    ├── exercise1.md
    └── solutions/
        └── exercise1/
            └── main.go
```

**Rationale:**
- Each example in separate directory prevents name conflicts
- No go.mod overhead - examples use parent repository's gocurl
- Simple structure: just `go run main.go` to execute
- Easy to read, understand, and copy
- No dependency management complexity

---

## 📚 COMPLETE BOOK STRUCTURE

### **Part I: Foundations** (4 chapters, ~100 pages)

#### Chapter 1: Why GoCurl? (25 pages)
**Purpose:** Establish the problem and solution

**Key Topics:**
- The REST API integration challenge
- Copy-paste from curl commands to Go
- Performance advantages (zero-allocation)
- Battle-tested quality (187+ tests)
- When to use gocurl vs net/http

**Hands-On:**
- First request (5 minutes to success)
- Performance comparison benchmark
- CLI-to-code workflow

**Learning Outcomes:**
- Understand gocurl's unique value
- Make first API call
- Decide when to use gocurl

---

#### Chapter 2: Installation & Setup (20 pages)
**Purpose:** Get developers productive fast

**Key Topics:**
- Installing gocurl (`go get`)
- IDE setup (VS Code, GoLand)
- CLI tool installation
- Quick verification examples
- Workspace organization

**Hands-On:**
- Install and verify
- First successful request
- Set up sample project structure

**Learning Outcomes:**
- Working gocurl installation
- Development environment ready
- Sample project to build upon

---

#### Chapter 3: Core Concepts (30 pages)
**Purpose:** Master fundamental concepts before diving deep

**Key Topics:**
- **Dual API Approach** (Curl syntax vs Programmatic)
- Variables & environment expansion
- Context & cancellation
- Response handling patterns
- Error handling basics

**Key Coverage:**
- Clear explanation of TWO ways to use gocurl
- When to use Curl* functions vs RequestOptions
- Process() as the core execution function

**Hands-On:**
- Environment variable expansion
- Context timeout examples
- Response parsing patterns

**Learning Outcomes:**
- Understand gocurl architecture
- Choose right approach for use case
- Handle responses correctly

---

#### Chapter 4: Command-Line Interface (25 pages)
**Purpose:** Master CLI for testing and automation

**Key Topics:**
- gocurl CLI tool usage
- Testing API endpoints
- CLI-to-code workflow
- Debugging techniques
- Shell scripting integration

**Hands-On:**
- Test real API with CLI
- Convert curl commands
- Build automation scripts

**Learning Outcomes:**
- Proficient with CLI tool
- Rapid API testing
- Automation capabilities

---

### **Part II: API Approaches** (3 chapters, ~85 pages)

#### Chapter 5: ✨ NEW - RequestOptions & Builder Pattern (30 pages)
**Purpose:** Master programmatic request building

**Key Topics:**
- **RequestOptions struct** - All 30+ fields explained
- **Builder pattern** - Fluent API usage
- Thread-safety & Clone() method
- Validation before execution
- When to use Builder vs Curl syntax

**Fields Covered:**
- HTTP basics (Method, URL, Headers, Body, Form, QueryParams)
- Authentication (BasicAuth, BearerToken)
- TLS/SSL (Certificates, Insecure, TLSConfig, CertPinning, SNI)
- Proxy (Proxy, ProxyNoProxy)
- Timeouts (Timeout, ConnectTimeout)
- Redirects (FollowRedirects, MaxRedirects)
- Compression (Compress, CompressionMethods)
- HTTP/2 (HTTP2, HTTP2Only)
- Cookies (Cookies, CookieJar, CookieFile)
- Custom (UserAgent, Referer)
- Upload (FileUpload)
- Retry (RetryConfig)
- Output (OutputFile, Silent, Verbose)
- Advanced (RequestID, Middleware, ResponseBodyLimit, CustomClient)

**Builder Methods:**
- All Set* methods
- Convenience methods (JSON, BearerAuth, Form, WithDefaultRetry)
- HTTP shortcuts (Post, Get, Put, Delete, Patch)
- Context management (WithContext, WithTimeout, Cleanup)
- Validation (Validate)
- Build & Clone

**Hands-On:**
- Build request with fluent API
- Clone for request templates
- Validate before execution
- Context management patterns

**Learning Outcomes:**
- Choose Builder vs Curl approach
- Build type-safe requests
- Implement request templates
- Prevent context leaks

---

#### Chapter 6: Working with JSON APIs (30 pages)
**Purpose:** Master JSON request/response patterns

**Key Topics:**
- CurlJSON functions (auto-unmarshaling)
- Manual JSON with CurlString
- Type-safe response handling
- Nested JSON structures
- JSON errors and edge cases

**Hands-On:**
- GitHub API client
- Stripe payment integration
- Error handling patterns
- Generic JSON utilities

**Learning Outcomes:**
- Parse JSON responses efficiently
- Handle API errors gracefully
- Build type-safe API clients

---

#### Chapter 7: File Operations (25 pages)
**Purpose:** Master file downloads and uploads

**Key Topics:**
- CurlDownload functions
- Multipart form uploads
- Progress tracking
- Resumable downloads
- Large file handling

**Hands-On:**
- Download with progress bar
- Upload files to S3
- Handle timeouts on large files

**Learning Outcomes:**
- Efficient file transfers
- Progress tracking
- Error recovery

---

### **Part III: Security & Configuration** (3 chapters, ~75 pages)

#### Chapter 8: Security & TLS (30 pages)
**Purpose:** Implement production-grade security

**Key Topics:**
- **TLS/SSL basics** (HTTPS, certificates)
- **Certificate pinning** (CertPinFingerprints)
- **Mutual TLS** (CertFile, KeyFile, CAFile)
- **SNI configuration** (SNIServerName)
- **Custom TLS config** (TLSConfig)
- **Insecure mode** (development only)
- **Secure auth validation** (HTTPS requirement)

**Key Coverage:**
- Complete TLS field coverage
- Certificate pinning tutorial
- Mutual TLS setup guide
- Security validation patterns

**Hands-On:**
- Set up mutual TLS
- Implement certificate pinning
- Validate security configurations

**Learning Outcomes:**
- Secure production API clients
- Implement certificate pinning
- Configure mutual TLS
- Pass security audits

---

#### Chapter 9: ✨ NEW - Advanced Configuration (25 pages)
**Purpose:** Master advanced RequestOptions features

**Key Topics:**
- **Custom clients** (HTTPClient interface, mocking, testing)
- **Response body limits** (ResponseBodyLimit, memory protection)
- **HTTP/2 control** (HTTP2, HTTP2Only)
- **Cookie management** (CookieJar, CookieFile, sessions)
- **Compression** (CompressionMethods, gzip, deflate, br)
- **Process() function** (core execution, direct usage)
- **Proxy bypass** (ProxyNoProxy rules)

**Key Coverage:**
- Complete coverage of advanced fields
- Process() function explained
- Custom client patterns for testing

**Hands-On:**
- Mock HTTP client for testing
- Implement response body limits
- Configure cookie persistence
- Use Process() directly

**Learning Outcomes:**
- Test with custom clients
- Protect against memory issues
- Manage sessions with cookies
- Understand internal execution

---

#### Chapter 10: Timeouts & Retries (20 pages)
**Purpose:** Build resilient clients

**Key Topics:**
- Timeout strategies (request, connect)
- Retry configuration
- Exponential backoff
- Circuit breaker patterns
- Error classification

**Hands-On:**
- Implement smart retries
- Handle flaky APIs
- Build circuit breaker

**Learning Outcomes:**
- Resilient API clients
- Graceful degradation
- Production reliability

---

### **Part IV: Middleware & Enterprise** (3 chapters, ~75 pages)

#### Chapter 11: ✨ NEW - Middleware System (25 pages)
**Purpose:** Master request pipeline and middleware

**Key Topics:**
- **Middleware architecture** (MiddlewareFunc type)
- **Writing middleware** (request transformation)
- **Middleware chaining** (execution order)
- **Common patterns:**
  - Authentication injection
  - Logging middleware
  - Distributed tracing
  - Request ID propagation
  - Rate limiting
  - Retry logic
- **OpenTelemetry integration**
- **Reusable middleware libraries**

**Key Coverage:**
- Complete middleware system coverage
- Production middleware examples
- OpenTelemetry integration guide

**Hands-On:**
- Write authentication middleware
- Implement logging middleware
- Build tracing middleware
- Chain multiple middleware

**Learning Outcomes:**
- Design middleware pipelines
- Implement observability
- Build reusable middleware
- Enterprise request handling

---

#### Chapter 12: Enterprise Patterns (30 pages)
**Purpose:** Build enterprise-grade clients

**Key Topics:**
- **Request ID & distributed tracing** (RequestID field)
- **Custom clients for mocking** (HTTPClient interface)
- **Proxy configuration** (Proxy, ProxyNoProxy)
- SDK design patterns
- Connection pooling
- Rate limiting
- Circuit breakers

**Key Coverage:**
- RequestID patterns explained
- Custom client mocking strategies
- Proxy bypass rules

**Hands-On:**
- Implement distributed tracing
- Build mockable SDK
- Configure proxy rules
- Design SDK wrapper

**Learning Outcomes:**
- Enterprise observability
- Testable architecture
- Production-ready SDKs

---

#### Chapter 13: Variable Substitution (20 pages)
**Purpose:** Master dynamic request building

**Key Topics:**
- Environment variables (automatic)
- Variable maps (WithVars functions)
- Security considerations
- Template patterns

**Hands-On:**
- Dynamic API endpoints
- Secure credential management
- Multi-environment configs

**Learning Outcomes:**
- Flexible request building
- Secure variable handling
- Environment management

---

### **Part V: Optimization & Testing** (3 chapters, ~70 pages)

#### Chapter 14: Performance Optimization (25 pages)
**Purpose:** Build high-performance clients

**Key Topics:**
- Zero-allocation architecture
- Benchmarking techniques
- Memory optimization
- Concurrent requests
- Connection reuse

**Hands-On:**
- Benchmark API client
- Optimize memory usage
- Concurrent request patterns

**Learning Outcomes:**
- High-performance clients
- Profiling techniques
- Optimization strategies

---

#### Chapter 15: Testing API Clients (25 pages)
**Purpose:** Write testable, reliable code

**Key Topics:**
- Mock servers (httptest)
- Custom client mocking
- Table-driven tests
- Integration testing
- Race detection

**Hands-On:**
- Write unit tests
- Build mock server
- Integration test suite

**Learning Outcomes:**
- Comprehensive test coverage
- Reliable API clients
- CI/CD integration

---

#### Chapter 16: Error Handling Patterns (20 pages)
**Purpose:** Handle failures gracefully

**Key Topics:**
- Error types and classification
- Retry strategies
- Fallback patterns
- Error logging
- User-friendly errors

**Hands-On:**
- Error handling middleware
- Retry with backoff
- Error reporting

**Learning Outcomes:**
- Robust error handling
- Production debugging
- User experience

---

### **Part VI: Advanced Topics** (3 chapters, ~65 pages)

#### Chapter 17: CLI Tool Development (20 pages)
**Purpose:** Build command-line tools

**Key Topics:**
- CLI architecture
- Flag parsing
- Output formatting
- Error reporting
- Distribution

**Hands-On:**
- Build CLI tool
- Package for distribution

**Learning Outcomes:**
- Professional CLI tools
- User-friendly interfaces

---

#### Chapter 18: Building SDK Wrappers (25 pages)
**Purpose:** Design production SDKs

**Key Topics:**
- SDK architecture
- Type-safe interfaces
- Documentation
- Versioning
- Distribution

**Hands-On:**
- Build GitHub SDK
- Write comprehensive docs

**Learning Outcomes:**
- Professional SDK design
- Library authoring

---

#### Chapter 19: Real-World Case Studies (20 pages)
**Purpose:** Apply knowledge to real projects

**Key Topics:**
- Payment gateway integration (Stripe)
- Cloud API client (AWS S3)
- Webhook receiver
- API aggregator

**Hands-On:**
- Complete case studies
- Production deployments

**Learning Outcomes:**
- Real-world application
- Best practices
- Production patterns

---

### **APPENDICES** (6 appendices, ~90 pages)

#### Appendix A: Complete API Reference (25 pages)
**ALL Functions Documented:**

**Primary API (Curl Functions):**
- Curl, CurlCommand, CurlArgs (2 returns: resp, err)

**Explicit Variable Control:**
- CurlWithVars, CurlCommandWithVars, CurlArgsWithVars (2 returns)

**String Functions:**
- CurlString, CurlStringCommand, CurlStringArgs (3 returns: body, resp, err)

**Bytes Functions:**
- CurlBytes, CurlBytesCommand, CurlBytesArgs (3 returns: body, resp, err)

**JSON Functions:**
- CurlJSON, CurlJSONCommand, CurlJSONArgs (2 returns: resp, err)

**Download Functions:**
- CurlDownload, CurlDownloadCommand, CurlDownloadArgs (3 returns: written, resp, err)

**Programmatic API:**
- NewRequestOptions()
- NewRequestOptionsBuilder()
- All 40+ builder methods
- Validate()
- Build()
- Clone()

**Core Functions:**
- Process(ctx, opts) - CORE execution function

**Legacy API (DEPRECATED):**
- Request(command, vars)
- RequestWithContext(ctx, command, vars)
- Execute(ctx, opts)
- Response.String(), Response.Bytes(), Response.JSON()

**Middleware:**
- MiddlewareFunc type

---

#### Appendix B: ✨ NEW - Migration from Legacy API (15 pages)
**Complete Migration Guide:**

**Old → New Patterns:**
```go
// OLD (Deprecated)
resp, err := gocurl.Request(cmd, vars)
body, _ := resp.String()

// NEW (Modern)
body, resp, err := gocurl.CurlString(ctx, cmd)
```

**Response Wrapper → Direct:**
```go
// OLD
resp, err := gocurl.Request(cmd, nil)
var user User
resp.JSON(&user)

// NEW
var user User
resp, err := gocurl.CurlJSON(ctx, &user, cmd)
```

**Execute → Process:**
```go
// OLD
resp, err := gocurl.Execute(ctx, opts)

// NEW
httpResp, _, err := gocurl.Process(ctx, opts)
```

**Side-by-side examples for all patterns**

---

#### Appendix C: cURL Command Reference (15 pages)
- Common curl flags
- Mapping to gocurl
- Examples

---

#### Appendix D: HTTP Status Codes (10 pages)
- Complete status code reference
- Handling strategies

---

#### Appendix E: Common Headers (10 pages)
- Authentication headers
- Content negotiation
- Caching

---

#### Appendix F: Performance Benchmarks (15 pages)
- Benchmark suite
- Comparison with net/http
- Optimization guide

---

## 🎯 COMPREHENSIVE COVERAGE

### Complete API Documentation:
All gocurl functions covered with correct signatures and examples:
- **Primary Curl Functions:** Curl, CurlCommand, CurlArgs
- **Variable Control:** CurlWithVars, CurlCommandWithVars, CurlArgsWithVars
- **String Functions:** CurlString, CurlStringCommand, CurlStringArgs
- **Bytes Functions:** CurlBytes, CurlBytesCommand, CurlBytesArgs
- **JSON Functions:** CurlJSON, CurlJSONCommand, CurlJSONArgs
- **Download Functions:** CurlDownload, CurlDownloadCommand, CurlDownloadArgs
- **RequestOptions:** All 30+ fields documented
- **Builder Pattern:** All 40+ methods explained
- **Middleware System:** Complete coverage
- **Legacy API:** Migration guide included

### Key Features:
- ✅ 100% API coverage (no missing functions)
- ✅ Dual approaches (Curl syntax + Builder pattern)
- ✅ ALL code examples compile and run
- ✅ Correct API signatures verified from source
- ✅ Production patterns throughout
- ✅ Security best practices
- ✅ Performance optimization
- ✅ Testing strategies
- ✅ Enterprise patterns

---

## 📅 IMPLEMENTATION TIMELINE

### Week 1-2: Foundation
- [ ] Master plan (THIS FILE)
- [ ] Complete outline with all chapters
- [ ] Style guide
- [ ] API reference documentation
- [ ] Code standards
- [ ] Directory structure

### Week 3-4: Part I (Foundations)
- [ ] Chapter 1: Why GoCurl?
- [ ] Chapter 2: Installation
- [ ] Chapter 3: Core Concepts
- [ ] Chapter 4: CLI

### Week 5-6: Part II (API Approaches)
- [ ] Chapter 5: RequestOptions & Builder Pattern
- [ ] Chapter 6: JSON APIs
- [ ] Chapter 7: File Operations

### Week 7-8: Part III (Security)
- [ ] Chapter 8: Security & TLS
- [ ] Chapter 9: Advanced Configuration
- [ ] Chapter 10: Timeouts & Retries

### Week 9-10: Part IV (Enterprise)
- [ ] Chapter 11: Middleware System
- [ ] Chapter 12: Enterprise Patterns
- [ ] Chapter 13: Variables

### Week 11-12: Part V (Optimization)
- [ ] Chapter 14: Performance
- [ ] Chapter 15: Testing
- [ ] Chapter 16: Error Handling

### Week 13-14: Part VI (Advanced)
- [ ] Chapter 17: CLI Development
- [ ] Chapter 18: SDK Wrappers
- [ ] Chapter 19: Case Studies

### Week 15-16: Appendices
- [ ] Appendix A: Complete API Reference
- [ ] Appendix B: Legacy API Migration Guide
- [ ] Appendix C: cURL Command Reference
- [ ] Appendix D: HTTP Status Codes
- [ ] Appendix E: Common Headers
- [ ] Appendix F: Performance Benchmarks

### Week 17-18: Review & Polish
- [ ] Technical review
- [ ] Code testing (all examples)
- [ ] Copyediting
- [ ] Final formatting

**Total: 18 weeks (4.5 months)**

---

## ✅ QUALITY CHECKLIST

### Code Quality:
- [ ] All examples compile with `go build`
- [ ] All examples tested with real APIs
- [ ] Race condition testing with `-race`
- [ ] No placeholder code
- [ ] Correct API signatures everywhere

### Content Quality:
- [ ] 100% API coverage
- [ ] All struct fields documented
- [ ] All builder methods explained
- [ ] Clear decision trees (when to use what)
- [ ] Real-world examples

### Writing Quality:
- [ ] O'Reilly style guide compliance
- [ ] SSR principles (Sweet, Simple, Robust)
- [ ] Technical accuracy verified
- [ ] Peer reviewed
- [ ] Copyedited

---

## 🎓 LEARNING PATHS

### Path 1: Quick Start (2-3 hours)
- Ch 1, 2, 3 (partial), 6 (partial)
- Goal: First working API client

### Path 2: Production Ready (1-2 days)
- Ch 1-7, 8, 10, 12
- Goal: Production-grade client

### Path 3: Complete Mastery (1-2 weeks)
- All chapters + appendices
- Goal: Expert-level knowledge

### Path 4: Enterprise Developer (3-4 days)
- Ch 1-3, 5, 8-12
- Goal: Enterprise patterns

---

## 📐 BOOK METRICS (Final)

| Metric | Target | Status |
|--------|--------|--------|
| **Total Pages** | 510-530 | Planned |
| **Chapters** | 19 | Planned |
| **Appendices** | 6 | Planned |
| **Code Examples** | 300+ | TBD |
| **Hands-On Projects** | 20+ | Planned |
| **API Coverage** | 100% | Planned |
| **Compilation Rate** | 100% | Target |

---

## 🚀 SUCCESS CRITERIA

### Technical:
- ✅ 100% of public API documented
- ✅ All code examples compile
- ✅ All examples tested
- ✅ Zero API signature errors

### Content:
- ✅ Covers both Curl AND Builder approaches
- ✅ Includes middleware system
- ✅ Includes legacy migration
- ✅ Production patterns throughout

### Quality:
- ✅ O'Reilly publication standards
- ✅ Professional illustrations
- ✅ Comprehensive index
- ✅ Online code repository

---

**STATUS:** Master plan complete ✅
**NEXT STEP:** Create detailed outline with all 19 chapters
**CONFIDENCE:** High - complete coverage with no gaps
