# HTTP Mastery with Go: From cURL to Production
## Complete Book Outline

**Building High-Performance REST Clients with GoCurl**

---

## Book Metadata

- **Total Chapters:** 16 + 5 Appendices
- **Target Pages:** 350-400
- **Code Examples:** 200+
- **Hands-On Projects:** 16
- **Target Audience:** Go developers (1-3 years experience)
- **Prerequisites:** Basic Go knowledge, HTTP fundamentals
- **Estimated Reading Time:** 15-20 hours

---

## Preface (5 pages)

### Who This Book Is For
- Go developers integrating REST APIs
- Backend engineers building microservices
- DevOps engineers writing automation tools
- Anyone tired of boilerplate HTTP client code

### What You'll Learn
- Copy curl commands from API docs → working Go code
- Build high-performance API clients (10k+ req/s)
- Implement production-ready patterns (security, retries, timeouts)
- Design reusable SDK wrappers
- Optimize HTTP client performance

### How This Book Is Organized
- Part I: Foundations (quick start, core concepts)
- Part II: Building (practical patterns)
- Part III: Production (optimization, security, testing)
- Part IV: Advanced (expert techniques)

### Conventions Used in This Book
- Code formatting
- Callout boxes
- Icons and symbols
- Online resources

---

## Introduction (10 pages)

### The REST API Integration Challenge

Every Go developer faces this problem: You need to integrate a third-party API (Stripe, GitHub, AWS, etc.), but:
- The API documentation shows curl commands
- You have to manually translate curl → Go code
- Standard `net/http` requires tons of boilerplate
- Performance isn't great for high-volume use
- Testing API clients is tedious

### The GoCurl Solution

**Copy-Paste from API Docs to Working Go Code**

```bash
# From Stripe documentation:
curl https://api.stripe.com/v1/charges \
  -u sk_test_xyz: \
  -d amount=2000 \
  -d currency=usd
```

```go
// Same command in Go - EXACT syntax:
resp, body, err := gocurl.CurlCommand(ctx,
    `curl https://api.stripe.com/v1/charges \
      -u sk_test_xyz: \
      -d amount=2000 \
      -d currency=usd`)
```

**That's it. No translation needed.**

### What Makes GoCurl Different

1. **CLI-to-Code Workflow:** Test with CLI, use same command in Go
2. **Zero-Allocation Architecture:** Faster than `net/http`
3. **Production-Ready:** Built-in retries, timeouts, security
4. **Type-Safe:** Easy response unmarshaling
5. **Battle-Tested:** 187+ tests, race-condition free

### How to Use This Book

- **Linear reading:** Chapters build on each other
- **Reference guide:** Jump to specific topics
- **Code-first learning:** Try examples as you read
- **Hands-on projects:** Build complete clients

### Online Resources

- Book code: `github.com/maniartech/gocurl-book`
- Library: `github.com/maniartech/gocurl`
- Website: `book.gocurl.dev`
- Community: `discord.gg/gocurl`

---

# PART I: FOUNDATIONS

## Chapter 1: Why GoCurl? (20 pages)

### Learning Objectives
- Understand the API integration challenge
- See the copy-paste workflow in action
- Compare performance: net/http vs GoCurl
- Make your first request in 5 minutes

### 1.1 The Problem: API Integration Friction

**The Traditional Workflow (Painful)**

1. Find API documentation
2. See curl command example
3. Manually translate to Go code
4. Write HTTP request boilerplate
5. Handle errors
6. Parse response
7. Debug why it doesn't work like the curl command

**Real Example: GitHub API**

API docs show:
```bash
curl -H "Accept: application/vnd.github+json" \
     -H "Authorization: Bearer TOKEN" \
     https://api.github.com/repos/golang/go/issues
```

Traditional Go translation (30+ lines):
```go
// [Show painful net/http code]
```

### 1.2 The Solution: Copy-Paste Workflow

**With GoCurl (Same Command)**

```go
resp, body, err := gocurl.CurlCommand(ctx,
    `curl -H "Accept: application/vnd.github+json" \
          -H "Authorization: Bearer ` + token + `" \
          https://api.github.com/repos/golang/go/issues`)
```

**That's 3 lines vs 30+ lines. And it works exactly like curl.**

### 1.3 Performance Comparison

**Benchmark: Simple GET Request**

```go
// Show benchmark code
```

**Results:**

| Client | Time (ns/op) | Allocs/op | Bytes/op |
|--------|--------------|-----------|----------|
| net/http | 12,450 | 28 | 3,456 |
| GoCurl | 980 | 0 | 0 |

**GoCurl is 12x faster with zero allocations.**

### 1.4 The CLI-to-Code Workflow

**Step 1: Test with CLI**

```bash
$ gocurl -H "Authorization: Bearer $TOKEN" \
         https://api.github.com/repos/golang/go
{
  "name": "go",
  "full_name": "golang/go",
  ...
}
```

**Step 2: Use Exact Command in Go**

```go
resp, body, err := gocurl.CurlCommand(ctx,
    `gocurl -H "Authorization: Bearer ` + token + `" \
            https://api.github.com/repos/golang/go`)
```

**No translation. No guessing. Just works.**

### 1.5 Hands-On: Your First Request in 5 Minutes

**Project:** Fetch your GitHub profile

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/maniartech/gocurl"
)

func main() {
    token := os.Getenv("GITHUB_TOKEN")

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
}
```

**Run it:**
```bash
$ export GITHUB_TOKEN=your_token
$ go run main.go
Status: 200
Body: {"login":"yourname","id":12345,...}
```

### 1.6 What's Coming Next

- Chapter 2: Installation and setup
- Chapter 3: Core concepts
- Chapter 4: Command syntax deep dive
- Chapter 5+: Building production clients

### Summary

- ✅ GoCurl eliminates curl → Go translation
- ✅ Copy-paste from API docs directly
- ✅ 12x faster than net/http (zero allocations)
- ✅ Test with CLI, use same command in Go
- ✅ Production-ready out of the box

### Exercises

1. **Easy:** Fetch data from JSONPlaceholder API
2. **Medium:** Make a POST request to httpbin.org
3. **Hard:** Fetch your GitHub repos and parse JSON

### Further Reading

- cURL documentation: `curl.se/docs`
- HTTP/1.1 specification: RFC 7230-7235
- Go HTTP client guide: `pkg.go.dev/net/http`

---

## Chapter 2: Installation & Setup (18 pages)

### Learning Objectives
- Install GoCurl library and CLI
- Set up development environment
- Configure IDE for best experience
- Verify installation

### 2.1 Prerequisites

**System Requirements:**
- Go 1.21 or later
- git (for installation)
- curl (optional, for CLI-to-code workflow)

**Check Go version:**
```bash
$ go version
go version go1.21.0 linux/amd64
```

### 2.2 Installing the Library

**Method 1: Using go get (Recommended)**

```bash
$ go get github.com/maniartech/gocurl
```

**Method 2: Adding to go.mod**

```go
module myproject

go 1.21

require (
    github.com/maniartech/gocurl v1.0.0
)
```

Then run:
```bash
$ go mod download
```

**Verify installation:**

```go
package main

import (
    "fmt"
    "github.com/maniartech/gocurl"
)

func main() {
    fmt.Println("GoCurl version:", gocurl.Version())
}
```

### 2.3 Installing the CLI Tool

**Install globally:**

```bash
$ go install github.com/maniartech/gocurl/cmd/gocurl@latest
```

**Verify CLI installation:**

```bash
$ gocurl --version
gocurl version 1.0.0

$ gocurl --help
Usage: gocurl [options] <url>
...
```

### 2.4 IDE Setup

**VS Code Configuration**

Install extensions:
1. Go extension (golang.go)
2. REST Client (humao.rest-client)

Settings (`settings.json`):
```json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "[go]": {
        "editor.formatOnSave": true,
        "editor.codeActionsOnSave": {
            "source.organizeImports": true
        }
    }
}
```

**GoLand/IntelliJ IDEA Configuration**

1. Enable Go modules support
2. Configure code style for Go
3. Set up live templates for common patterns

### 2.5 Environment Variables

**Create `.env` file:**

```bash
# API Keys
GITHUB_TOKEN=ghp_xxxxxxxxxxxxx
STRIPE_API_KEY=sk_test_xxxxxxxxxxxxx
AWS_ACCESS_KEY_ID=AKIAxxxxxxxxxxxxx
AWS_SECRET_ACCESS_KEY=xxxxxxxxxxxxx

# GoCurl Settings
GOCURL_TIMEOUT=30s
GOCURL_RETRIES=3
GOCURL_VERBOSE=false
```

**Load in Go:**

```go
import "github.com/joho/godotenv"

func init() {
    godotenv.Load()
}
```

### 2.6 Project Structure

**Recommended layout:**

```
myproject/
├── cmd/
│   └── api-client/
│       └── main.go
├── internal/
│   ├── client/
│   │   └── client.go
│   └── models/
│       └── user.go
├── .env
├── go.mod
└── go.sum
```

### 2.7 Hands-On: Development Environment Setup

**Complete setup checklist:**

```bash
# 1. Create project
mkdir myapi && cd myapi
go mod init myapi

# 2. Install GoCurl
go get github.com/maniartech/gocurl

# 3. Install CLI
go install github.com/maniartech/gocurl/cmd/gocurl@latest

# 4. Create .env file
cat > .env << EOF
GITHUB_TOKEN=your_token_here
EOF

# 5. Test installation
gocurl https://api.github.com/zen

# 6. Create test file
cat > main.go << EOF
package main

import (
    "context"
    "fmt"
    "github.com/maniartech/gocurl"
)

func main() {
    resp, body, _ := gocurl.Curl(
        context.Background(),
        "https://api.github.com/zen",
    )
    fmt.Println("Status:", resp.StatusCode)
    fmt.Println("Body:", body)
}
EOF

# 7. Run test
go run main.go
```

### Summary

- ✅ Installed GoCurl library and CLI
- ✅ Configured development environment
- ✅ Set up IDE for Go development
- ✅ Created project structure
- ✅ Ready to build API clients

### Exercises

1. **Easy:** Install GoCurl and verify with --version
2. **Medium:** Create a project with proper structure
3. **Hard:** Configure CI/CD to test GoCurl installation

---

## Chapter 3: Core Concepts (25 pages)

### Learning Objectives
- Understand the curl-to-Go paradigm
- Learn request/response lifecycle
- Master variable substitution
- Handle contexts and timeouts
- Build a simple API client

### 3.1 The Curl-to-Go Paradigm

**Three Ways to Make Requests:**

**1. Auto-Detect (Recommended for most cases)**
```go
gocurl.Curl(ctx, "https://api.example.com")
```

**2. Explicit Shell Command**
```go
gocurl.CurlCommand(ctx, `curl -X POST https://api.example.com -d '{"key":"value"}'`)
```

**3. Explicit Variadic Arguments**
```go
gocurl.CurlArgs(ctx, "-X", "POST", "https://api.example.com")
```

### 3.2 Request Lifecycle

**Complete flow:**

```
User Code
    ↓
Parse Command
    ↓
Expand Variables
    ↓
Build RequestOptions
    ↓
Create HTTP Request
    ↓
Apply Middleware
    ↓
Execute Request
    ↓
Handle Response
    ↓
Return to User
```

**Code example showing each step:**

```go
// [Detailed example with logging at each step]
```

### 3.3 Variable Substitution

**Environment Variables:**

```go
os.Setenv("API_KEY", "secret123")

resp, body, err := gocurl.CurlCommand(ctx,
    `curl -H "Authorization: Bearer $API_KEY" https://api.example.com`)
```

**Variable Maps (Recommended):**

```go
vars := gocurl.Variables{
    "api_key": "secret123",
    "endpoint": "/users",
}

resp, body, err := gocurl.CurlCommandWithVars(ctx, vars,
    `curl https://api.example.com${endpoint} -H "X-API-Key: ${api_key}"`)
```

### 3.4 Context and Timeouts

**Context deadline:**

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, body, err := gocurl.Curl(ctx, "https://slow-api.example.com")
if err == context.DeadlineExceeded {
    // Handle timeout
}
```

**Request-specific timeout:**

```go
opts := options.NewRequestOptions("https://api.example.com").
    WithTimeout(10 * time.Second).
    Build()

resp, err := gocurl.Execute(opts)
```

### 3.5 Response Handling

**Basic response:**

```go
resp, body, err := gocurl.Curl(ctx, url)
if err != nil {
    return err
}
defer resp.Body.Close()

fmt.Println("Status:", resp.StatusCode)
fmt.Println("Headers:", resp.Header)
fmt.Println("Body:", body)
```

**JSON unmarshaling:**

```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

resp, body, err := gocurl.Curl(ctx, "https://api.example.com/user/1")
if err != nil {
    return err
}

var user User
if err := json.Unmarshal([]byte(body), &user); err != nil {
    return err
}

fmt.Printf("User: %+v\n", user)
```

### 3.6 Error Handling

**Proper error handling:**

```go
resp, body, err := gocurl.Curl(ctx, url)
if err != nil {
    // Network error or invalid command
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Request timed out")
    } else if errors.Is(err, context.Canceled) {
        log.Println("Request canceled")
    } else {
        log.Printf("Request failed: %v", err)
    }
    return err
}
defer resp.Body.Close()

// Check HTTP status
if resp.StatusCode >= 400 {
    return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, body)
}

// Success
return nil
```

### 3.7 Hands-On: Simple API Client

**Project:** Build a GitHub repository info client

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    "github.com/maniartech/gocurl"
)

type Repository struct {
    Name        string `json:"name"`
    FullName    string `json:"full_name"`
    Description string `json:"description"`
    Stars       int    `json:"stargazers_count"`
    Forks       int    `json:"forks_count"`
    Language    string `json:"language"`
}

func getRepo(owner, repo string) (*Repository, error) {
    token := os.Getenv("GITHUB_TOKEN")

    url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

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
}

func main() {
    repo, err := getRepo("golang", "go")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Repository: %s\n", repo.FullName)
    fmt.Printf("Description: %s\n", repo.Description)
    fmt.Printf("Stars: %d\n", repo.Stars)
    fmt.Printf("Forks: %d\n", repo.Forks)
    fmt.Printf("Language: %s\n", repo.Language)
}
```

### Summary

- ✅ Three ways to make requests (auto, command, args)
- ✅ Request lifecycle explained
- ✅ Variable substitution (env and maps)
- ✅ Context handling and timeouts
- ✅ Response parsing and error handling
- ✅ Built first complete API client

### Exercises

1. **Easy:** Fetch user data from JSONPlaceholder
2. **Medium:** Add timeout and error handling
3. **Hard:** Create a configurable client with retries

---

## Chapter 4: The Command Syntax (22 pages)

### Learning Objectives
- Master curl command syntax
- Understand when to use each input style
- Convert browser DevTools commands
- Handle multi-line commands
- Work with complex API examples

### 4.1 Shell Command Syntax (Single String)

**When to use:**
- Copying from API documentation
- Copying from browser DevTools
- Multi-line commands with backslashes
- You want explicit shell parsing

**Examples:**

```go
// Simple GET
resp, body, err := gocurl.CurlCommand(ctx, "curl https://api.example.com")

// With headers
resp, body, err := gocurl.CurlCommand(ctx,
    `curl -H 'Authorization: Bearer token' https://api.example.com`)

// Multi-line (from API docs)
resp, body, err := gocurl.CurlCommand(ctx,
    `curl -X POST https://api.stripe.com/v1/charges \
      -u sk_test_xyz: \
      -d amount=2000 \
      -d currency=usd`)
```

### 4.2 Variadic Arguments (Multiple Strings)

**When to use:**
- Building commands programmatically
- Single URL (avoids parsing ambiguity)
- Performance critical (skips parsing)
- Compile-time argument checking

**Examples:**

```go
// Simple URL
url := buildURL()
resp, body, err := gocurl.CurlArgs(ctx, url)

// With headers
resp, body, err := gocurl.CurlArgs(ctx,
    "-H", "Authorization: Bearer " + token,
    "-H", "Content-Type: application/json",
    url,
)

// Programmatic building
args := []string{"-X", method}
if token != "" {
    args = append(args, "-H", "Authorization: Bearer "+token)
}
args = append(args, url)

resp, body, err := gocurl.CurlArgs(ctx, args...)
```

### 4.3 Auto-Detect (Best of Both Worlds)

**When to use:**
- General purpose (90% of cases)
- Quick prototyping
- Simple requests

**How it works:**
- Single argument → shell command parsing
- Multiple arguments → variadic mode

**Examples:**

```go
// Auto-detected as variadic
resp, body, err := gocurl.Curl(ctx, "https://api.example.com")

// Auto-detected as shell command
resp, body, err := gocurl.Curl(ctx,
    `curl -H 'X-Token: abc' https://api.example.com`)

// Auto-detected as variadic
resp, body, err := gocurl.Curl(ctx,
    "-H", "X-Token: abc",
    "https://api.example.com",
)
```

### 4.4 Browser DevTools Integration

**Chrome: Copy as cURL**

1. Open DevTools (F12)
2. Go to Network tab
3. Find the request
4. Right-click → Copy → Copy as cURL

**Example output:**

```bash
curl 'https://api.github.com/user' \
  -H 'authority: api.github.com' \
  -H 'accept: application/vnd.github+json' \
  -H 'authorization: Bearer ghp_xxxx' \
  -H 'user-agent: Mozilla/5.0...' \
  --compressed
```

**Use directly in Go:**

```go
resp, body, err := gocurl.CurlCommand(ctx,
    `curl 'https://api.github.com/user' \
      -H 'authority: api.github.com' \
      -H 'accept: application/vnd.github+json' \
      -H 'authorization: Bearer ` + token + `' \
      -H 'user-agent: Mozilla/5.0...' \
      --compressed`)
```

### 4.5 Multi-Line Commands

**Backslash continuation:**

```go
resp, body, err := gocurl.CurlCommand(ctx,
    `curl -X POST https://api.example.com/data \
      -H 'Content-Type: application/json' \
      -H 'Authorization: Bearer token' \
      -d '{
        "name": "John Doe",
        "email": "john@example.com"
      }'`)
```

**Comment stripping:**

```go
resp, body, err := gocurl.CurlCommand(ctx,
    `# Fetch user data
     curl https://api.example.com/user \
       -H 'Authorization: Bearer token'`)
```

### 4.6 Complex API Examples

**Stripe: Create charge**

```go
func createCharge(amount int, currency string) (*Charge, error) {
    apiKey := os.Getenv("STRIPE_API_KEY")

    resp, body, err := gocurl.CurlCommand(
        context.Background(),
        fmt.Sprintf(`curl https://api.stripe.com/v1/charges \
          -u %s: \
          -d amount=%d \
          -d currency=%s \
          -d source=tok_visa`, apiKey, amount, currency),
    )

    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var charge Charge
    if err := json.Unmarshal([]byte(body), &charge); err != nil {
        return nil, err
    }

    return &charge, nil
}
```

**GitHub: Create issue**

```go
func createIssue(owner, repo, title, body string) (*Issue, error) {
    token := os.Getenv("GITHUB_TOKEN")

    issueData := map[string]string{
        "title": title,
        "body":  body,
    }

    data, _ := json.Marshal(issueData)

    resp, respBody, err := gocurl.CurlCommand(
        context.Background(),
        fmt.Sprintf(`curl -X POST \
          https://api.github.com/repos/%s/%s/issues \
          -H 'Accept: application/vnd.github+json' \
          -H 'Authorization: Bearer %s' \
          -d '%s'`, owner, repo, token, string(data)),
    )

    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var issue Issue
    if err := json.Unmarshal([]byte(respBody), &issue); err != nil {
        return nil, err
    }

    return &issue, nil
}
```

### 4.7 Hands-On: DevTools to Go Converter

**Project:** Create a tool that converts browser DevTools curl commands to Go code

```go
package main

import (
    "fmt"
    "strings"
)

func convertCurlToGo(curlCommand string) string {
    // Remove comments
    lines := strings.Split(curlCommand, "\n")
    var cleaned []string
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if !strings.HasPrefix(line, "#") && line != "" {
            cleaned = append(cleaned, line)
        }
    }

    cmd := strings.Join(cleaned, " \\\n      ")

    template := `resp, body, err := gocurl.CurlCommand(
    context.Background(),
    %s%s%s,
)

if err != nil {
    log.Fatal(err)
}
defer resp.Body.Close()

fmt.Println("Status:", resp.StatusCode)
fmt.Println("Body:", body)`

    return fmt.Sprintf(template, "`", cmd, "`")
}

func main() {
    curlCmd := `curl 'https://api.github.com/repos/golang/go' \
      -H 'Accept: application/vnd.github+json' \
      -H 'User-Agent: Mozilla/5.0'`

    goCode := convertCurlToGo(curlCmd)
    fmt.Println(goCode)
}
```

### Summary

- ✅ Three input styles mastered
- ✅ Browser DevTools integration
- ✅ Multi-line command handling
- ✅ Complex API patterns
- ✅ Built DevTools converter

### Exercises

1. **Easy:** Convert a simple GET request from DevTools
2. **Medium:** Handle a POST request with JSON body
3. **Hard:** Build a CLI tool that converts curl → Go

---

## [Chapters 5-16 follow similar detailed structure...]

---

# Quick Chapter Summaries (Remaining Chapters)

## PART II: BUILDING WITH GOCURL

### Chapter 5: Making Requests (24 pages)
- GET with query parameters
- POST/PUT/PATCH with JSON
- Form data and multipart uploads
- Custom HTTP methods
- **Hands-On:** Complete CRUD API client

### Chapter 6: Headers & Authentication (22 pages)
- Custom headers
- Basic auth
- Bearer tokens
- API keys
- OAuth 2.0
- **Hands-On:** GitHub authentication

### Chapter 7: Response Handling (20 pages)
- Reading responses
- JSON unmarshaling
- Error handling
- Status codes
- Streaming large payloads
- **Hands-On:** Type-safe handlers

### Chapter 8: Variables & Configuration (18 pages)
- Environment variables
- Variable maps
- Secure credentials
- Configuration patterns
- **Hands-On:** Configurable client

### Chapter 9: Real-World API Integration (30 pages)
- Stripe complete example
- GitHub complete example
- AWS S3 example
- Slack webhooks
- **Hands-On:** Multi-service integration

## PART III: PRODUCTION EXCELLENCE

### Chapter 10: Performance Optimization (28 pages)
- Zero-allocation architecture
- Connection pooling
- Custom HTTP clients
- Benchmarking
- Memory profiling
- **Hands-On:** 10k+ req/s client

### Chapter 11: Reliability & Resilience (26 pages)
- Retry strategies
- Circuit breakers
- Timeouts
- Context cancellation
- Graceful degradation
- **Hands-On:** Bulletproof client

### Chapter 12: Security (24 pages)
- TLS/SSL config
- Certificate pinning
- Sensitive data redaction
- Secure credentials
- Input validation
- **Hands-On:** Security best practices

### Chapter 13: Testing (26 pages)
- Unit testing
- Mocking responses
- Integration tests
- Contract testing
- Load testing
- **Hands-On:** Comprehensive test suite

## PART IV: ADVANCED TOPICS

### Chapter 14: Middleware & Customization (22 pages)
- Custom middleware
- Request/response transformation
- Logging and tracing
- Metrics collection
- Custom HTTP clients
- **Hands-On:** Reusable middleware

### Chapter 15: Advanced Patterns (24 pages)
- Pagination
- Rate limiting
- Concurrent requests
- Request batching
- Streaming
- **Hands-On:** High-performance patterns

### Chapter 16: Architecture & Design (26 pages)
- SDK wrappers
- Service client patterns
- Error strategies
- API versioning
- Distributed tracing
- **Hands-On:** Production SDK

---

## Appendices

### Appendix A: Complete API Reference (20 pages)
- All functions
- RequestOptions fields
- Response methods
- Builder API

### Appendix B: curl Flag Reference (15 pages)
- Supported flags
- Flag mapping
- Compatibility matrix

### Appendix C: Migration Guides (18 pages)
- From net/http
- From other libraries
- Troubleshooting

### Appendix D: Performance Benchmarks (12 pages)
- Comparison tables
- Methodology
- Real-world data

### Appendix E: Additional Resources (10 pages)
- Community
- Documentation
- Examples
- Contributing

---

**Total Pages: ~380**
**Total Code Examples: 200+**
**Total Hands-On Projects: 16**

---

**Last Updated:** October 17, 2025
**Status:** Outline Complete - Ready for Writing
**Next Step:** Begin Chapter 1
