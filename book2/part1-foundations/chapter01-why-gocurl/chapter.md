# Chapter 1: Why GoCurl?

*Making REST API integration as simple as copy-paste*

Every Go developer has experienced the frustration: you find a perfect API that solves your problem, the documentation provides clear curl commands to test it, but then you face the tedious task of translating those commands into verbose Go code. Hours are spent wrestling with `net/http` boilerplate, debugging header formats, and testing edge cases that worked perfectly with curl.

What if you could skip the translation step entirely? What if you could paste curl commands directly into your Go code and have them just work?

This is exactly what GoCurl provides: a bridge between the curl commands in API documentation and production-ready Go code. But GoCurl is more than just a curl wrapper—it's a complete HTTP client library designed for performance, reliability, and developer productivity.

In this chapter, you'll discover why GoCurl exists, when to use it, and how it compares to traditional approaches. By the end, you'll have a clear understanding of GoCurl's value proposition and be ready to start building robust API clients.

---

## Learning Objectives

By the end of this chapter, you will:

- **Understand** the HTTP client problem that GoCurl solves
- **Identify** when to use GoCurl vs traditional `net/http`
- **Make** your first successful API call with GoCurl
- **Compare** GoCurl's performance against standard approaches
- **Recognize** production features that set GoCurl apart
- **Master** the CLI-to-code workflow for rapid development
- **Build** a complete GitHub repository viewer application

---

## The HTTP Client Problem

Let's start with a real scenario. You're building a Go application that needs to integrate with the GitHub API. The GitHub documentation shows this curl command:

```bash
curl -H "Accept: application/vnd.github+json" \
     -H "Authorization: Bearer YOUR_TOKEN" \
     https://api.github.com/repos/golang/go
```

Simple enough to test in your terminal. But translating this to Go requires significantly more code:

```go
package main

import (
    "fmt"
    "io"
    "net/http"
    "os"
)

func main() {
    // Create the HTTP client
    client := &http.Client{}

    // Create the request
    req, err := http.NewRequest("GET",
        "https://api.github.com/repos/golang/go", nil)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
        os.Exit(1)
    }

    // Add headers
    req.Header.Add("Accept", "application/vnd.github+json")
    req.Header.Add("Authorization", "Bearer YOUR_TOKEN")

    // Execute the request
    resp, err := client.Do(req)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error making request: %v\n", err)
        os.Exit(1)
    }
    defer resp.Body.Close()

    // Read the response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
        os.Exit(1)
    }

    // Check status code
    if resp.StatusCode != 200 {
        fmt.Fprintf(os.Stderr, "Unexpected status: %d\n", resp.StatusCode)
        os.Exit(1)
    }

    // Print the result
    fmt.Println(string(body))
}
```

That's **36 lines of code** to replicate a **3-line curl command**. And we haven't even added:
- Timeouts
- Retries for network errors
- Proper error context
- Response validation
- Request tracing

### The Translation Burden

The problem isn't just the code volume—it's the **cognitive load** of translation:

1. **Header syntax differences**: Curl's `-H "Name: Value"` becomes `req.Header.Add("Name", "Value")`
2. **Authentication formats**: `-u user:pass` must be translated to `req.SetBasicAuth()`
3. **Data encoding**: `-d` form data requires `url.Values` and proper encoding
4. **File uploads**: `-F` multipart forms need `multipart.Writer` setup
5. **Error handling**: Every step can fail and needs checking

Each translation introduces opportunities for bugs. Did you encode the form data correctly? Are the headers in the right format? Is the Content-Type set properly?

### The Maintenance Problem

API documentation changes. A new header is required, authentication switches from tokens to OAuth, or a new parameter gets added. With traditional approaches, you must:

1. Test the updated curl command manually
2. Translate the changes to Go code
3. Verify the translation is correct
4. Update tests

This cycle repeats for every API change, consuming valuable development time.

---

## The GoCurl Approach

GoCurl eliminates the translation step. Here's the same GitHub API call with GoCurl:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

func main() {
    ctx := context.Background()

    body, resp, err := gocurl.CurlString(ctx,
        "-H", "Accept: application/vnd.github+json",
        "-H", "Authorization: Bearer YOUR_TOKEN",
        "https://api.github.com/repos/golang/go")

    if err != nil {
        log.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        log.Fatalf("Unexpected status: %d", resp.StatusCode)
    }

    fmt.Println(body)
}
```

**Only 16 lines** compared to 36. But the real benefit isn't just fewer lines—it's **zero translation**. The curl syntax remains intact, meaning:

- Copy curl commands directly from API docs
- Test changes instantly without rewriting Go code
- No translation errors
- API documentation updates map 1:1 to code changes

### Even Simpler: The Command String Format

GoCurl can also parse complete curl command strings:

```go
body, resp, err := gocurl.CurlStringCommand(ctx,
    `curl -H "Accept: application/vnd.github+json" \
          -H "Authorization: Bearer YOUR_TOKEN" \
          https://api.github.com/repos/golang/go`)
```

This is **identical** to the curl command—just wrapped in `gocurl.CurlStringCommand()`. You can literally copy-paste from API documentation.

---

## When to Use GoCurl

GoCurl isn't always the right choice. Understanding when to use it (and when not to) ensures you make effective architectural decisions.

### ✅ Use GoCurl When:

**1. Rapid API Integration**
You need to integrate a third-party API quickly. API docs provide curl examples, and you want working code immediately without translation overhead.

**Example:** Adding Stripe payment processing to your e-commerce site.

**2. API Exploration & Prototyping**
You're experimenting with a new API, testing different endpoints, and iterating quickly. The curl syntax allows fast experimentation.

**Example:** Testing different GitHub API endpoints to find the right data structure.

**3. CLI-to-Code Workflow**
Your workflow involves testing APIs with curl first, then converting working commands to code. GoCurl eliminates the conversion step.

**Example:** Building a monitoring tool that queries multiple REST endpoints.

**4. High-Volume API Clients**
Your application makes thousands of API calls per second. GoCurl's "parse once, execute many" model (`Prepare` a request once, `Do` it many times over a pooled `Client`) keeps per-request overhead thin and constant — **parity** with a well-tuned `net/http` client, never a "faster than net/http" claim (GoCurl is *built on* `net/http`). The performance is measured honestly against the competition in [docs/benchmarking.md](../../../docs/benchmarking.md), losses included.

**Example:** A metrics aggregation service polling hundreds of microservices.

**5. Production-Grade Requirements**
You need built-in retries, timeouts, certificate pinning, distributed tracing, and middleware—all without writing custom infrastructure.

**Example:** Enterprise SaaS platform integrating with customer APIs.

**6. SDK Development**
You're building an SDK wrapper around a REST API and want clean, maintainable code with minimal boilerplate.

**Example:** Official Go SDK for your company's public API.

### ❌ Don't Use GoCurl When:

**1. Simple, One-Off Requests**
For a single GET request in a small script, standard `http.Get()` is simpler:

```go
resp, err := http.Get("https://api.example.com/status")
```

**2. Custom HTTP Client Requirements**
You need very specific `http.Client` configurations that GoCurl doesn't expose directly. (Note: GoCurl supports custom clients via `RequestOptions.CustomClient`, but highly specialized needs might be better served directly.)

**3. GraphQL or gRPC**
GoCurl is designed for REST/HTTP APIs. For GraphQL, use specialized libraries. For gRPC, use the official gRPC-Go library.

**4. WebSocket Connections**
GoCurl handles HTTP request/response cycles. For persistent WebSocket connections, use `gorilla/websocket` or similar.

**5. Team Unfamiliarity**
If your team has never seen curl syntax and learning it would slow development, standard Go approaches might be more familiar.

### Decision Tree

```
Need to make HTTP requests?
│
├─ GraphQL or gRPC?
│  └─ NO to GoCurl → Use specialized libraries
│
├─ WebSocket connection?
│  └─ NO to GoCurl → Use websocket library
│
├─ Single simple GET/POST?
│  └─ Consider http.Get() or http.Post()
│
├─ Integrating REST API with curl examples?
│  └─ YES to GoCurl ✅
│
├─ Need high performance?
│  └─ YES to GoCurl ✅
│
├─ Need production features (retries, tracing)?
│  └─ YES to GoCurl ✅
│
└─ Building SDK or API client library?
   └─ YES to GoCurl ✅
```

---

## Your First API Call: 5-Minute Quick Start

Let's make your first successful API call with GoCurl. We'll query the GitHub API to get repository information.

### Prerequisites

```bash
go version  # Requires Go 1.21+
```

### Install GoCurl

```bash
go get github.com/maniartech/gocurl
```

### Example 1: Simple GET Request

Create `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

func main() {
    ctx := context.Background()

    // Simple GET request - no authentication needed
    body, resp, err := gocurl.CurlString(ctx,
        "https://api.github.com/users/octocat")

    if err != nil {
        log.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    fmt.Printf("Status: %d\n", resp.StatusCode)
    fmt.Printf("Content-Type: %s\n", resp.Header.Get("Content-Type"))
    fmt.Printf("\nBody:\n%s\n", body)
}
```

Run it:

```bash
go run main.go
```

**Output:**
```
Status: 200
Content-Type: application/json; charset=utf-8

Body:
{
  "login": "octocat",
  "id": 583231,
  "name": "The Octocat",
  "public_repos": 8,
  ...
}
```

**Success!** You've made your first GoCurl request.

### Example 2: POST with JSON Data

Let's simulate creating a GitHub issue (using JSONPlaceholder as a test API):

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

func main() {
    ctx := context.Background()

    body, resp, err := gocurl.CurlString(ctx,
        "-X", "POST",
        "-H", "Content-Type: application/json",
        "-d", `{"title":"Test Issue","body":"This is a test"}`,
        "https://jsonplaceholder.typicode.com/posts")

    if err != nil {
        log.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    fmt.Printf("Status: %d\n", resp.StatusCode)
    fmt.Printf("Response:\n%s\n", body)
}
```

**Output:**
```
Status: 201
Response:
{
  "title": "Test Issue",
  "body": "This is a test",
  "id": 101
}
```

### Example 3: Automatic JSON Unmarshaling

Instead of parsing JSON manually, let GoCurl unmarshal it directly:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

type User struct {
    Login       string `json:"login"`
    Name        string `json:"name"`
    PublicRepos int    `json:"public_repos"`
    Followers   int    `json:"followers"`
}

func main() {
    ctx := context.Background()

    var user User
    resp, err := gocurl.CurlJSON(ctx, &user,
        "https://api.github.com/users/octocat")

    if err != nil {
        log.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    fmt.Printf("User: %s (%s)\n", user.Name, user.Login)
    fmt.Printf("Repos: %d | Followers: %d\n",
        user.PublicRepos, user.Followers)
}
```

**Output:**
```
User: The Octocat (octocat)
Repos: 8 | Followers: 9762
```

**Note:** `CurlJSON` returns only `(*http.Response, error)`—the JSON is unmarshaled directly into the `user` variable you provide.

### Example 4: OpenAI Chat Completion

Modern AI applications often integrate with OpenAI's API. Here's how simple it is with GoCurl:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/maniartech/gocurl"
)

type ChatResponse struct {
    Choices []struct {
        Message struct {
            Content string `json:"content"`
        } `json:"message"`
    } `json:"choices"`
}

func main() {
    ctx := context.Background()
    apiKey := os.Getenv("OPENAI_API_KEY")

    payload := `{
        "model": "gpt-4",
        "messages": [{"role": "user", "content": "Explain HTTP in one sentence"}]
    }`

    var response ChatResponse
    resp, err := gocurl.CurlJSON(ctx, &response,
        "-X", "POST",
        "-H", "Authorization: Bearer "+apiKey,
        "-H", "Content-Type: application/json",
        "-d", payload,
        "https://api.openai.com/v1/chat/completions")

    if err != nil {
        log.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    if len(response.Choices) > 0 {
        fmt.Println("AI Response:", response.Choices[0].Message.Content)
    }
}
```

**Output:**
```
AI Response: HTTP is a protocol that enables communication between web clients and servers by defining how messages are formatted and transmitted.
```

### Example 5: Stripe Payment Processing

Integrating with payment APIs is a common requirement. Here's a Stripe example:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/maniartech/gocurl"
)

type PaymentIntent struct {
    ID            string `json:"id"`
    Amount        int    `json:"amount"`
    Currency      string `json:"currency"`
    Status        string `json:"status"`
    ClientSecret  string `json:"client_secret"`
}

func main() {
    ctx := context.Background()
    apiKey := os.Getenv("STRIPE_SECRET_KEY")

    var intent PaymentIntent
    resp, err := gocurl.CurlJSON(ctx, &intent,
        "-X", "POST",
        "-u", apiKey+":",
        "-d", "amount=2000",
        "-d", "currency=usd",
        "-d", "payment_method_types[]=card",
        "https://api.stripe.com/v1/payment_intents")

    if err != nil {
        log.Fatalf("Payment creation failed: %v", err)
    }
    defer resp.Body.Close()

    fmt.Printf("Payment Intent Created: %s\n", intent.ID)
    fmt.Printf("Amount: $%.2f %s\n", float64(intent.Amount)/100, intent.Currency)
    fmt.Printf("Status: %s\n", intent.Status)
}
```

### Example 6: Database REST API (Supabase)

Modern database services provide REST APIs. Here's querying a Supabase table:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/maniartech/gocurl"
)

type User struct {
    ID        int    `json:"id"`
    Email     string `json:"email"`
    CreatedAt string `json:"created_at"`
}

func main() {
    ctx := context.Background()
    supabaseURL := os.Getenv("SUPABASE_URL")
    apiKey := os.Getenv("SUPABASE_KEY")

    var users []User
    resp, err := gocurl.CurlJSON(ctx, &users,
        "-H", "apikey: "+apiKey,
        "-H", "Authorization: Bearer "+apiKey,
        supabaseURL+"/rest/v1/users?select=*&limit=10")

    if err != nil {
        log.Fatalf("Query failed: %v", err)
    }
    defer resp.Body.Close()

    fmt.Printf("Found %d users:\n", len(users))
    for _, user := range users {
        fmt.Printf("- %s (ID: %d)\n", user.Email, user.ID)
    }
}
```

### Example 7: Slack Webhook Integration

Send notifications to Slack channels:

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
    ctx := context.Background()
    webhookURL := os.Getenv("SLACK_WEBHOOK_URL")

    payload := `{
        "text": "🚀 Deployment completed successfully!",
        "blocks": [
            {
                "type": "section",
                "text": {
                    "type": "mrkdwn",
                    "text": "*Deployment Status*\nVersion: v1.2.3\nEnvironment: Production"
                }
            }
        ]
    }`

    resp, err := gocurl.Curl(ctx,
        "-X", "POST",
        "-H", "Content-Type: application/json",
        "-d", payload,
        webhookURL)

    if err != nil {
        log.Fatalf("Slack notification failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == 200 {
        fmt.Println("✅ Slack notification sent successfully")
    }
}
```

**Real-World Applications:**

These examples demonstrate GoCurl's versatility across modern API integrations:

- **AI/ML**: OpenAI, Anthropic, Google AI for intelligent features
- **Payments**: Stripe, PayPal, Square for e-commerce
- **Databases**: Supabase, Firebase, MongoDB Atlas for data persistence
- **Communication**: Slack, Twilio, SendGrid for notifications
- **Cloud Services**: AWS, Azure, GCP REST APIs for infrastructure

**The pattern is identical**: construct the curl-syntax call with proper headers and data, let GoCurl handle the HTTP complexity.

---

## Performance Comparison

Performance matters when building production systems. Let's compare GoCurl against standard `net/http` approaches.

### The honest claim: parity, not superiority

> **GoCurl's motto: persuasion by example, not by marketing.** Throughout this book, every
> performance or reliability claim points at a test or benchmark you can run yourself. If we
> can't show it, we don't say it.

GoCurl is a thin layer **built on** `net/http`. It therefore **cannot** be faster than the
`net/http` client it wraps, and we never claim it is. What we claim — and measure — is:

- **Parity**: a prepared request executed over a pooled `Client` adds only thin, constant
  overhead above a bare `net/http` round-trip.
- **Ergonomics + reliability**: you get the curl-paste workflow and a proven production
  pipeline for that small, honest overhead.

### How it's measured

The suite runs every arm — `net/http` (the parity bar), GoCurl prepared, and popular
clients (`resty`, `req`) — with **identical** transport tuning (enforced by a guard test).
No real-network endpoints (they'd be unreproducible), no cherry-picked numbers.

One subtlety worth teaching: an early version ran everything over a real loopback TCP
connection, and the latency numbers rank-flipped run to run. A CPU profile showed why —
**~60% of the measured time was the OS network syscall plus goroutine scheduling**, work
that's identical across all arms and swings with machine load. The few microseconds of
*library* overhead drowned in it. So the suite adds a second view, `BenchmarkOverhead_*`,
that swaps in a stub transport (no network): it isolates each library's per-request work and
gives a *stable* ranking. There, GoCurl adds the **lowest** overhead of the three
full-featured clients — fewer ns, bytes, and allocations than both resty and req — behind
only raw `net/http`, which carries no resilience machinery. The round-trip view is kept for
end-to-end realism, with its latency treated as advisory.

The results — **wins and losses alike** — live in
[docs/benchmarking.md](../../../docs/benchmarking.md). At the time of writing, GoCurl is at
latency parity with `net/http` and carries a *smaller* per-request byte footprint than the
thinner wrappers resty and req — behind only raw `net/http`. That was not always true: an
earlier run had GoCurl as the heaviest arm, and we published that loss; profiling then found
a per-`Do` RNG allocation (~4.9 KiB) that ran even without retries, and making it lazy closed
the gap. The win is regression-guarded by `TestByteBudget_Do`, and `clone-the-small`
(`TestCloneSmall_NoDeepClonePerDo`) removed the per-`Do` deep clone of the recipe. This is the
motto in action: we found the regression by *measuring*, fixed it, and let the test keep us
honest — not by tuning a marketing number.

### Where the real developer win is

The win is **not** raw speed — it's the **execution pipeline**. Hand-rolled `net/http`
forces you to re-derive, per service, an overall-timeout-across-retries, idempotency-aware
retries, error classification, secret redaction, and memory bounds — and to keep all of it
correct as the code changes. Because GoCurl has the **curl recipe**, it knows your intent and
assembles that pipeline for you, proven correct by a fault-injection harness. That is the
comparative reason to choose GoCurl, and it does not depend on a single nanosecond.

---

## Production Features Overview

GoCurl isn't just about syntax convenience—it provides enterprise-grade features out of the box.

### 1. Automatic Retries with Exponential Backoff

Network failures happen. GoCurl handles them gracefully:

```go
opts := gocurl.NewRequestOptions("https://api.example.com/data")
opts.RetryConfig = &gocurl.RetryConfig{
    MaxRetries:  3,
    RetryDelay:  time.Second,
    RetryOnHTTP: []int{500, 502, 503, 504},
}

resp, err := gocurl.Process(ctx, opts)
```

If the server returns 503 (Service Unavailable), GoCurl automatically retries up to 3 times with increasing delays: 1s, 2s, 4s.

**Covered in detail:** Chapter 10 - Timeouts & Retries

### 2. Certificate Pinning for Security

Protect against man-in-the-middle attacks by pinning expected certificate fingerprints:

```go
opts := gocurl.NewRequestOptions("https://api.bank.com/transfer")
opts.CertPinFingerprints = []string{
    "sha256/X3pGTSOuJeEVw989IJ/oKo9EgZ9GN6wpFevf0tVFJ0=",
}

resp, err := gocurl.Process(ctx, opts)
// Fails if server certificate doesn't match fingerprint
```

**Covered in detail:** Chapter 8 - Security & TLS

### 3. Distributed Tracing Support

Track requests across microservices with request IDs:

```go
opts := gocurl.NewRequestOptions("https://api.service-a.com/data")
opts.RequestID = "req-12345-67890"

resp, err := gocurl.Process(ctx, opts)
// Request ID automatically added to logs and traces
```

Integrates with OpenTelemetry and other tracing systems.

**Covered in detail:** Chapter 12 - Enterprise Patterns

### 4. Middleware Pipeline

Transform requests before execution:

```go
// Add timestamp to all requests
func TimestampMiddleware(req *http.Request) (*http.Request, error) {
    req.Header.Set("X-Timestamp", time.Now().Format(time.RFC3339))
    return req, nil
}

opts := gocurl.NewRequestOptions("https://api.example.com")
opts.Middleware = []gocurl.MiddlewareFunc{
    TimestampMiddleware,
    AuthMiddleware,
    LoggingMiddleware,
}

resp, err := gocurl.Process(ctx, opts)
```

**Covered in detail:** Chapter 11 - Middleware System

### 5. Context-Based Timeouts & Cancellation

Full context support for timeouts and cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := gocurl.Curl(ctx, "https://slow-api.com/data")
// Automatically cancelled after 5 seconds
```

**Covered in detail:** Chapter 3 - Core Concepts & Chapter 10 - Timeouts & Retries

### 6. Mutual TLS (mTLS)

Client certificate authentication for secure services:

```go
opts := gocurl.NewRequestOptions("https://api.enterprise.com/secure")
opts.CertFile = "/path/to/client-cert.pem"
opts.KeyFile = "/path/to/client-key.pem"
opts.CAFile = "/path/to/ca-cert.pem"

resp, err := gocurl.Process(ctx, opts)
```

**Covered in detail:** Chapter 8 - Security & TLS

---

## The CLI-to-Code Workflow

One of GoCurl's most powerful features is the seamless workflow between CLI testing and production code.

### Traditional Workflow

1. **Test API with curl** in terminal
2. **Manually translate** curl command to Go code
3. **Debug** translation errors
4. **Update** when API changes
5. **Repeat** steps 1-4

**Time wasted:** Hours per integration

### GoCurl Workflow

1. **Test API with gocurl CLI**
2. **Copy command directly** into Go code
3. **Done**

**Time saved:** Minutes per integration

### Step-by-Step Example

**Step 1: Install GoCurl CLI**

```bash
go install github.com/maniartech/gocurl/cmd/gocurl@latest
```

**Step 2: Test API with CLI**

```bash
gocurl -H "Accept: application/json" \
       https://api.github.com/repos/golang/go
```

Output shows response headers, status, body, and performance metrics.

**Step 3: Copy to Go Code**

If the command works, wrap it in `gocurl.CurlCommand()`:

```go
body, resp, err := gocurl.CurlStringCommand(ctx,
    `gocurl -H "Accept: application/json" \
            https://api.github.com/repos/golang/go`)
```

Or use the variadic form:

```go
body, resp, err := gocurl.CurlString(ctx,
    "-H", "Accept: application/json",
    "https://api.github.com/repos/golang/go")
```

**Step 4: Add Error Handling & Polish**

```go
func GetRepo(ctx context.Context, owner, repo string) (*Repository, error) {
    url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

    var repoData Repository
    resp, err := gocurl.CurlJSON(ctx, &repoData,
        "-H", "Accept: application/vnd.github+json",
        url)

    if err != nil {
        return nil, fmt.Errorf("failed to fetch repo: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    return &repoData, nil
}
```

**That's it!** No translation, no debugging format errors, no wasted time.

---

## Hands-On Project: GitHub Repository Viewer

Let's build a complete application that fetches and displays GitHub repository information. This project demonstrates:

- Making authenticated API calls
- Parsing JSON responses
- Error handling
- Clean code organization

### Project Requirements

Build a CLI tool that:
1. Accepts a GitHub repository as input (e.g., `golang/go`)
2. Fetches repository information from GitHub API
3. Displays key metrics (stars, forks, issues, language)
4. Handles errors gracefully
5. Supports optional authentication via token

### Implementation

Create `github-viewer/main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    "github.com/maniartech/gocurl"
)

// Repository represents GitHub repository data
type Repository struct {
    Name            string    `json:"name"`
    FullName        string    `json:"full_name"`
    Description     string    `json:"description"`
    Language        string    `json:"language"`
    StargazersCount int       `json:"stargazers_count"`
    ForksCount      int       `json:"forks_count"`
    OpenIssuesCount int       `json:"open_issues_count"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
    HTMLURL         string    `json:"html_url"`
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: github-viewer <owner/repo>")
        fmt.Println("Example: github-viewer golang/go")
        os.Exit(1)
    }

    // Parse repository input
    repoPath := os.Args[1]
    parts := strings.Split(repoPath, "/")
    if len(parts) != 2 {
        log.Fatal("Invalid repository format. Use: owner/repo")
    }

    owner, repo := parts[0], parts[1]

    // Get optional GitHub token from environment
    token := os.Getenv("GITHUB_TOKEN")

    // Fetch repository information
    repoInfo, err := fetchRepository(owner, repo, token)
    if err != nil {
        log.Fatalf("Error: %v", err)
    }

    // Display results
    displayRepository(repoInfo)
}

func fetchRepository(owner, repo, token string) (*Repository, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

    // Build request arguments
    args := []string{
        "-H", "Accept: application/vnd.github+json",
    }

    // Add authentication if token provided
    if token != "" {
        args = append(args, "-H", fmt.Sprintf("Authorization: Bearer %s", token))
    }

    // Add URL
    args = append(args, url)

    // Execute request with JSON unmarshaling
    var repoData Repository
    resp, err := gocurl.CurlJSON(ctx, &repoData, args...)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch repository: %w", err)
    }
    defer resp.Body.Close()

    // Check status code
    if resp.StatusCode == 404 {
        return nil, fmt.Errorf("repository not found: %s/%s", owner, repo)
    }

    if resp.StatusCode == 401 {
        return nil, fmt.Errorf("authentication failed (invalid token)")
    }

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }

    return &repoData, nil
}

func displayRepository(repo *Repository) {
    fmt.Println("\n" + strings.Repeat("=", 60))
    fmt.Printf("  %s\n", repo.FullName)
    fmt.Println(strings.Repeat("=", 60))

    if repo.Description != "" {
        fmt.Printf("\n📝 Description:\n  %s\n", repo.Description)
    }

    fmt.Printf("\n📊 Statistics:\n")
    fmt.Printf("  ⭐ Stars:        %d\n", repo.StargazersCount)
    fmt.Printf("  🔱 Forks:        %d\n", repo.ForksCount)
    fmt.Printf("  ❗ Open Issues:  %d\n", repo.OpenIssuesCount)

    if repo.Language != "" {
        fmt.Printf("\n💻 Language:      %s\n", repo.Language)
    }

    fmt.Printf("\n📅 Dates:\n")
    fmt.Printf("  Created:  %s\n", repo.CreatedAt.Format("2006-01-02"))
    fmt.Printf("  Updated:  %s\n", repo.UpdatedAt.Format("2006-01-02"))

    fmt.Printf("\n🔗 URL: %s\n", repo.HTMLURL)
    fmt.Println()
}
```

### Testing the Application

**1. Build the application:**

```bash
cd github-viewer
go mod init github-viewer
go get github.com/maniartech/gocurl
go build
```

**2. Run without authentication (rate limited):**

```bash
./github-viewer golang/go
```

**Output:**
```
============================================================
  golang/go
============================================================

📝 Description:
  The Go programming language

📊 Statistics:
  ⭐ Stars:        118234
  🔱 Forks:        15876
  ❗ Open Issues:  8543

💻 Language:      Go

📅 Dates:
  Created:  2014-08-19
  Updated:  2024-10-17

🔗 URL: https://github.com/golang/go
```

**3. Run with authentication (higher rate limits):**

```bash
export GITHUB_TOKEN=your_personal_access_token
./github-viewer microsoft/vscode
```

**4. Test error handling:**

```bash
# Non-existent repository
./github-viewer nonexistent/repository
# Output: Error: repository not found: nonexistent/repository

# Invalid format
./github-viewer invalid-format
# Output: Invalid repository format. Use: owner/repo
```

### What You've Learned

This project demonstrates:

✅ **Real API integration** - Fetching live data from GitHub
✅ **JSON unmarshaling** - Using `CurlJSON()` for structured data
✅ **Error handling** - Graceful handling of network and API errors
✅ **Authentication** - Optional token-based auth
✅ **Context usage** - Timeouts for reliability
✅ **Clean code** - Organized, production-ready structure

### Extension Ideas

Enhance this project by adding:

1. **List user repositories**: Fetch all repos for a user
2. **Search functionality**: Search repositories by topic or language
3. **Caching**: Store results to reduce API calls
4. **Output formats**: Add JSON or CSV output options
5. **Batch processing**: Process multiple repositories from a file
6. **Rate limit handling**: Check and display remaining API rate limits

---

## Summary

In this chapter, we explored why GoCurl exists and when to use it:

- **The Problem**: Traditional HTTP clients require verbose boilerplate and manual translation of curl commands to Go code
- **The Solution**: GoCurl provides curl-compatible syntax that eliminates translation overhead
- **Performance**: GoCurl is faster and more memory-efficient than standard `net/http` patterns
- **Production Features**: Built-in retries, certificate pinning, distributed tracing, middleware, and mTLS
- **CLI-to-Code Workflow**: Test with CLI, copy command to code—no translation step
- **Practical Application**: Built a complete GitHub repository viewer with error handling and authentication

### Key Patterns to Remember

**Simple GET request:**
```go
body, resp, err := gocurl.CurlString(ctx, "https://api.example.com/data")
```

**JSON unmarshaling:**
```go
var data MyStruct
resp, err := gocurl.CurlJSON(ctx, &data, "https://api.example.com/data")
```

**POST with data:**
```go
body, resp, err := gocurl.CurlString(ctx,
    "-X", "POST",
    "-H", "Content-Type: application/json",
    "-d", `{"key":"value"}`,
    "https://api.example.com/create")
```

### What's Next?

Now that you understand the value proposition, you're ready to start using GoCurl. In the next chapter, we'll cover:

- Installing the GoCurl library and CLI tool
- Setting up your development environment
- Configuring your IDE for optimal productivity
- Verifying your installation with tests

By the end of Chapter 2, you'll have a fully configured GoCurl development environment and be ready to dive into the core concepts.

---

**Chapter 1 Complete** ✅

Continue to **Chapter 2: Installation & Setup** →
