# Style Guide
## The Definitive Guide to the GoCurl Library

**Version:** 1.0
**Last Updated:** 2025
**Status:** Active

---

## 1. Purpose & Scope

This style guide ensures consistency, clarity, and quality across all chapters of "The Definitive Guide to the GoCurl Library." All content must adhere to these standards to maintain professional O'Reilly-quality publication standards.

### 1.1 Guiding Principles

1. **Clarity First** - Every sentence must serve the reader's understanding
2. **Real Examples Only** - All code examples must compile and run
3. **Progressive Disclosure** - Introduce concepts in logical order
4. **Practical Focus** - Always connect theory to practice
5. **Respect Reader's Time** - Be concise without sacrificing completeness

---

## 2. Writing Voice & Tone

### 2.1 Voice Characteristics

**Professional but Approachable:**
- ✅ "Let's examine how gocurl handles retries..."
- ❌ "Hey, retries are pretty cool, right?"
- ❌ "The retry mechanism shall be examined herein..."

**Clear and Direct:**
- ✅ "The `CurlString` function returns three values: the response body as a string, the response object, and an error."
- ❌ "So basically, `CurlString` gives you back a bunch of stuff..."
- ❌ "The aforementioned function provides a triumvirate of return values..."

**Confident but Humble:**
- ✅ "This approach works well for most API clients."
- ❌ "This is THE ONLY way to build API clients!"
- ❌ "Maybe this might possibly work sometimes..."

### 2.2 Pronouns

- Use **"we"** when guiding the reader through examples
  - ✅ "We'll start by creating a simple GET request..."
- Use **"you"** when describing actions the reader takes
  - ✅ "You can configure retries using the `RetryConfig` struct..."
- Avoid **"I"** - this is not a personal blog
  - ❌ "I think you should use the Builder pattern..."

### 2.3 Tense

- **Present tense** for describing how things work
  - ✅ "The `Process()` function handles all HTTP execution..."
- **Future tense** when previewing upcoming content
  - ✅ "In Chapter 8, we'll explore certificate pinning..."
- **Past tense** only when referencing earlier content
  - ✅ "As we saw in Chapter 3, context cancellation stops..."

---

## 3. Content Structure

### 3.1 Chapter Organization

Every chapter must follow this structure:

```markdown
# Chapter N: Title

## Learning Objectives
[3-5 bullet points stating what readers will learn]

## [Content Section 1]
[3-6 pages of content]

## [Content Section 2]
[3-6 pages of content]

... [4-8 sections total]

## Hands-On Project: [Project Name]
[Complete working implementation with:
 - Project overview
 - Requirements
 - Complete code
 - Testing instructions
 - Extension ideas]

## Summary
[Recap of key points, 1-2 pages]
```

### 3.2 Section Organization

**Introduction Paragraph:**
- State what the section covers
- Explain why it matters
- Preview the subsections

**Body:**
- Present concepts in logical order
- Use examples to illustrate each concept
- Include warnings/notes/tips as needed

**Conclusion:**
- Summarize key takeaways
- Point to related sections/chapters

### 3.3 Learning Objectives Format

Must be:
- **Actionable** - Start with action verbs (Understand, Master, Learn, Build, etc.)
- **Measurable** - Reader can verify they achieved it
- **Specific** - Not vague or generic

✅ Good:
- "Understand the difference between Curl-syntax and Builder pattern approaches"
- "Master context-based timeout and cancellation patterns"
- "Build a production-ready API client with retries and error handling"

❌ Bad:
- "Learn about timeouts" (too vague)
- "Become a gocurl expert" (not measurable)
- "See some examples" (not actionable)

---

## 4. Code Examples

### 4.1 Code Quality Standards

**ALL code examples must:**
1. **Compile** - No syntax errors, no placeholders
2. **Run** - Execute without runtime errors
3. **Demonstrate** - Show the concept being discussed
4. **Self-Contain** - Include all necessary imports/setup
5. **Follow Go conventions** - Use `gofmt`, proper naming

### 4.2 Code Example Format

**Minimal Example (< 10 lines):**
```go
resp, err := gocurl.Curl(ctx, "https://api.github.com/users/octocat")
if err != nil {
    return err
}
defer resp.Body.Close()
```

**Complete Example (10-50 lines):**
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/stackql/gocurl"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    resp, err := gocurl.Curl(ctx, "https://api.github.com/users/octocat")
    if err != nil {
        log.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        log.Fatalf("Unexpected status: %d", resp.StatusCode)
    }

    fmt.Println("Request successful!")
}
```

**Listing Example (> 50 lines):**
```go
// Listing 3-1: Complete API Client Example
package apiclient

import (
    "context"
    "fmt"
    "time"

    "github.com/stackql/gocurl"
)

// GitHubClient provides access to GitHub API
type GitHubClient struct {
    baseURL string
    token   string
}

// NewGitHubClient creates a new GitHub API client
func NewGitHubClient(token string) *GitHubClient {
    return &GitHubClient{
        baseURL: "https://api.github.com",
        token:   token,
    }
}

// GetUser retrieves user information
func (c *GitHubClient) GetUser(ctx context.Context, username string) (*User, error) {
    url := fmt.Sprintf("%s/users/%s", c.baseURL, username)

    var user User
    resp, err := gocurl.CurlJSON(ctx, &user, url,
        "-H", fmt.Sprintf("Authorization: Bearer %s", c.token),
        "-H", "Accept: application/vnd.github+json",
    )
    if err != nil {
        return nil, fmt.Errorf("failed to fetch user: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    return &user, nil
}

type User struct {
    Login     string `json:"login"`
    Name      string `json:"name"`
    Email     string `json:"email"`
    PublicRepos int  `json:"public_repos"`
}
```

### 4.3 Code Example Captions

- **Inline code** (< 10 lines): No caption needed
- **Complete examples** (10-50 lines): Add descriptive comment at top
- **Listings** (> 50 lines): Number and title (e.g., "Listing 3-1: Complete API Client")

### 4.4 Comments in Code

**Use comments to:**
- Explain WHY, not WHAT (code shows what)
- Highlight important patterns
- Point out common mistakes
- Reference other chapters/sections

**Don't over-comment:**
```go
// ❌ BAD - Comments state the obvious
ctx := context.Background() // Create a context
resp, err := gocurl.Curl(ctx, url) // Make the request
if err != nil { // Check for errors
    return err // Return the error
}

// ✅ GOOD - Comments add value
// Use background context since this operation has no deadline
ctx := context.Background()

// CurlString returns: (body, response, error)
// We need the body as a string for parsing
body, resp, err := gocurl.CurlString(ctx, url)
if err != nil {
    return fmt.Errorf("failed to fetch data: %w", err)
}
```

### 4.5 Error Handling

**Always show proper error handling:**

✅ Good:
```go
resp, err := gocurl.Curl(ctx, url)
if err != nil {
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()
```

❌ Bad:
```go
resp, _ := gocurl.Curl(ctx, url) // Never ignore errors!
```

❌ Bad:
```go
// TODO: Add error handling
resp := gocurl.Curl(ctx, url) // No placeholders!
```

### 4.6 Complete vs. Snippet

**Complete examples** must include:
- Package declaration
- All imports
- Full function/struct definitions
- Error handling
- Resource cleanup (defer)

**Snippets** (for brevity) can omit:
- Imports (if obvious: context, fmt, log)
- Package declaration (if clearly in context)
- Boilerplate setup (if shown earlier)

**Indicate when using snippets:**
```go
// Continuing from the previous example...
resp, err := gocurl.CurlJSON(ctx, &data, url)
```

---

## 5. Technical Accuracy

### 5.1 API Signatures

**CRITICAL:** All function signatures must match the actual gocurl API.

**Verified Signatures:**

```go
// Basic (2 returns)
func Curl(ctx context.Context, command string, args ...string) (*http.Response, error)
func CurlCommand(ctx context.Context, command string) (*http.Response, error)
func CurlArgs(ctx context.Context, args ...string) (*http.Response, error)

// String (3 returns - body FIRST)
func CurlString(ctx context.Context, command string, args ...string) (string, *http.Response, error)
func CurlStringCommand(ctx context.Context, command string) (string, *http.Response, error)
func CurlStringArgs(ctx context.Context, args ...string) (string, *http.Response, error)

// Bytes (3 returns - body FIRST)
func CurlBytes(ctx context.Context, command string, args ...string) ([]byte, *http.Response, error)
func CurlBytesCommand(ctx context.Context, command string) ([]byte, *http.Response, error)
func CurlBytesArgs(ctx context.Context, args ...string) ([]byte, *http.Response, error)

// JSON (2 returns - unmarshals into v)
func CurlJSON(ctx context.Context, v interface{}, command string, args ...string) (*http.Response, error)
func CurlJSONCommand(ctx context.Context, v interface{}, command string) (*http.Response, error)
func CurlJSONArgs(ctx context.Context, v interface{}, args ...string) (*http.Response, error)

// Download (3 returns - bytes written, response, error)
func CurlDownload(ctx context.Context, outputPath, command string, args ...string) (int64, *http.Response, error)
func CurlDownloadCommand(ctx context.Context, outputPath, command string) (int64, *http.Response, error)
func CurlDownloadArgs(ctx context.Context, outputPath string, args ...string) (int64, *http.Response, error)

// WithVars (2 returns - explicit variables)
func CurlWithVars(ctx context.Context, vars map[string]string, command string, args ...string) (*http.Response, error)
func CurlCommandWithVars(ctx context.Context, vars map[string]string, command string) (*http.Response, error)
func CurlArgsWithVars(ctx context.Context, vars map[string]string, args ...string) (*http.Response, error)

// Core execution
func Process(ctx context.Context, opts *RequestOptions) (*http.Response, error)
```

**NEVER use incorrect signatures:**
- ❌ `Curl() → (response, body, error)` - WRONG ORDER
- ❌ `CurlString() → (response, error)` - WRONG COUNT
- ❌ `CurlJSON() → (data, response, error)` - WRONG SIGNATURE

### 5.2 Terminology

**Use official terminology:**

| Correct | Incorrect |
|---------|-----------|
| Curl-syntax functions | Curl-style, curl-like |
| Builder pattern | Builder style, fluent API only |
| RequestOptions | Request options, Options struct |
| Process() function | Execute function, Process method |
| Middleware | Middleware function, interceptor |
| Variable expansion | Variable substitution, templating |
| Context cancellation | Context timeout (when specifically about cancel) |

### 5.3 Cross-References

**When referencing other chapters:**
- ✅ "As we'll see in Chapter 8..."
- ✅ "Recall from Chapter 3 that..."
- ✅ "For more on retries, see Chapter 10."

**When referencing sections:**
- ✅ "See 'Context Cancellation' in Chapter 3"
- ✅ "Refer to the Builder Pattern (Chapter 5)"

**When referencing appendices:**
- ✅ "See Appendix A for the complete API reference"
- ✅ "Appendix B provides migration examples"

---

## 6. Formatting Conventions

### 6.1 Text Formatting

**Code Elements:**
- Function names: `` `Curl()` ``, `` `CurlString()` ``
- Parameters: `` `ctx` ``, `` `command` ``
- Types: `` `*http.Response` ``, `` `RequestOptions` ``
- Package names: `` `gocurl` ``, `` `context` ``
- File names: `` `main.go` ``, `` `api_client.go` ``
- Commands: `` `go get` ``, `` `go test` ``
- URLs: `` `https://api.github.com` ``

**Emphasis:**
- Important concepts: **bold** (e.g., **always close response bodies**)
- New terms on first use: *italic* (e.g., *variable expansion*)
- Do NOT use emphasis for regular text

**Lists:**
- Use numbered lists for sequential steps
- Use bullet points for unordered items
- Use consistent punctuation (all items end with period, or none do)

### 6.2 Code Blocks

**Language specification:**
```go
// Always specify language
func Example() {
    // ...
}
```

```bash
# For shell commands
go get github.com/stackql/gocurl
```

```json
{
  "comment": "For JSON examples",
  "language": "json"
}
```

**NO language specification for:**
- Output/results (use plain text or omit)
- Configuration files (use specific language: yaml, toml, etc.)

### 6.3 Callout Boxes

**Note:** General information
> **Note:** The `CurlString` functions return the body as the first value, unlike the basic `Curl` functions.

**Warning:** Common mistakes or pitfalls
> **Warning:** Always close the response body with `defer resp.Body.Close()` to prevent resource leaks.

**Tip:** Helpful advice or best practices
> **Tip:** Use `CurlJSON` to automatically unmarshal JSON responses into your structs.

**Important:** Critical information
> **Important:** The `WithVars` functions do NOT expand environment variables—only the explicit variables you provide.

### 6.4 Diagrams

**Use diagrams for:**
- Architecture/flow (text-based ASCII art acceptable)
- Decision trees (when to use which function)
- Component relationships

**Example ASCII diagram:**
```
User Code
    │
    ├─→ Curl-syntax Functions (CurlString, CurlJSON, etc.)
    │       │
    │       └─→ Tokenization
    │               │
    │               └─→ RequestOptions
    │                       │
    └─→ Builder Pattern ────┘
            │
            └─→ Process() ──→ http.Client ──→ HTTP Request
```

---

## 7. Hands-On Projects

### 7.1 Project Requirements

Every chapter must include ONE hands-on project that:
1. **Synthesizes** multiple concepts from the chapter
2. **Compiles and runs** without modification
3. **Demonstrates** real-world usage
4. **Extends easily** with suggested enhancements
5. **Provides value** - solves a realistic problem

### 7.2 Project Structure

```markdown
## Hands-On Project: [Name]

### Project Overview
[2-3 paragraphs describing what we'll build and why]

### Requirements
[Bulleted list of what the project needs to do]

### Implementation
[Complete, runnable code with comments]

### Testing
[How to run and verify the project works]

### Extension Ideas
[3-5 ways readers can extend the project]
```

### 7.3 Project Complexity

- **Early chapters (1-5):** 50-100 lines
- **Middle chapters (6-12):** 100-200 lines
- **Advanced chapters (13-19):** 200-300 lines

### 7.4 Project Topics

Projects should be **practical and realistic:**
- ✅ GitHub repository viewer
- ✅ Weather data aggregator
- ✅ API health monitor
- ✅ Webhook receiver
- ✅ Multi-service orchestrator

Projects should NOT be **trivial or contrived:**
- ❌ Hello World variations
- ❌ Toy examples with no real use
- ❌ Overly academic exercises

---

## 8. Summary Sections

### 8.1 Summary Format

Every chapter ends with a summary that:
1. **Recaps** key concepts (bulleted list)
2. **Highlights** important code patterns
3. **Previews** next chapter
4. **Length:** 1-2 pages

### 8.2 Summary Template

```markdown
## Summary

In this chapter, we explored [main topic]:

- **[Key Concept 1]:** [One sentence summary]
- **[Key Concept 2]:** [One sentence summary]
- **[Key Concept 3]:** [One sentence summary]
- **[Key Concept 4]:** [One sentence summary]

You learned how to:
- [Action 1]
- [Action 2]
- [Action 3]

The key patterns to remember are:
```go
// Critical code pattern 1
```

```go
// Critical code pattern 2
```

In the next chapter, we'll [preview next topic], building on these concepts to [benefit].
```

---

## 9. Common Patterns

### 9.1 Introducing New Concepts

**Three-step pattern:**

1. **Why it matters** (motivation)
   > "When building production API clients, you need reliable retry logic to handle transient failures..."

2. **What it is** (definition)
   > "GoCurl provides the `RetryConfig` struct to configure automatic retries with exponential backoff..."

3. **How to use it** (example)
   ```go
   opts := gocurl.NewRequestOptions(url)
   opts.RetryConfig = &gocurl.RetryConfig{
       MaxRetries: 3,
       RetryDelay: time.Second,
       RetryOnHTTP: []int{500, 502, 503, 504},
   }
   ```

### 9.2 Comparing Alternatives

**Use comparison tables:**

| Approach | Best For | Limitations |
|----------|----------|-------------|
| `Curl()` | Quick requests, CLI conversion | Manual body reading |
| `CurlString()` | Text responses (HTML, XML) | Loads entire body into memory |
| `CurlJSON()` | Structured API data | Requires known struct type |

### 9.3 Progressive Examples

**Build complexity gradually:**

```go
// Step 1: Basic request
resp, err := gocurl.Curl(ctx, "https://api.example.com/data")

// Step 2: Add authentication
resp, err := gocurl.Curl(ctx, "https://api.example.com/data",
    "-H", "Authorization: Bearer token",
)

// Step 3: Add error handling and JSON parsing
var data Response
resp, err := gocurl.CurlJSON(ctx, &data, "https://api.example.com/data",
    "-H", "Authorization: Bearer token",
)
if err != nil {
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()

// Step 4: Full production version with retries
opts := gocurl.NewRequestOptionsBuilder().
    SetURL("https://api.example.com/data").
    AddHeader("Authorization", "Bearer token").
    SetRetryConfig(&gocurl.RetryConfig{
        MaxRetries: 3,
        RetryDelay: time.Second,
    }).
    Build()

resp, err := gocurl.Process(ctx, opts)
// ... error handling
```

---

## 10. Quality Checklist

Before submitting any chapter content, verify:

### 10.1 Code Quality
- [ ] All code examples compile
- [ ] All code examples run without errors
- [ ] Error handling is complete
- [ ] Resources are properly cleaned up (defer)
- [ ] API signatures match actual gocurl API
- [ ] No placeholders or TODOs

### 10.2 Content Quality
- [ ] Learning objectives are clear and measurable
- [ ] Concepts are introduced in logical order
- [ ] Examples illustrate each concept
- [ ] Hands-on project synthesizes chapter content
- [ ] Summary recaps key points
- [ ] Cross-references are accurate

### 10.3 Writing Quality
- [ ] Voice is professional but approachable
- [ ] Tense is consistent
- [ ] Terminology is official
- [ ] Code elements use backticks
- [ ] Callout boxes are used appropriately
- [ ] No typos or grammar errors

### 10.4 Structure Quality
- [ ] Chapter follows standard structure
- [ ] Sections are 3-6 pages each
- [ ] Hands-on project is complete and runnable
- [ ] Summary is 1-2 pages
- [ ] Formatting is consistent

---

## 11. Review Process

### 11.1 Self-Review

Before submitting content:
1. Run all code examples
2. Check API signatures against source
3. Verify cross-references
4. Read aloud (catches awkward phrasing)
5. Run spell-check

### 11.2 Peer Review

Reviewers check for:
1. Technical accuracy
2. Code quality
3. Clarity of explanation
4. Consistency with style guide

### 11.3 Technical Review

Final review verifies:
1. All code compiles with current gocurl version
2. All examples produce expected output
3. API signatures are correct
4. No deprecated features

---

## 12. Revision History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2025 | Initial style guide created |

---

## Appendix: Example Chapter Excerpt

**Following all style guide principles:**

---

## Understanding the Curl Functions

GoCurl provides six categories of functions, each designed for specific use cases. Understanding when to use each category is crucial for writing clean, efficient code.

### The Basic Functions

The basic `Curl`, `CurlCommand`, and `CurlArgs` functions return two values: the HTTP response and an error. These functions are ideal when you need full control over response body handling.

```go
resp, err := gocurl.Curl(ctx, "https://api.github.com/users/octocat")
if err != nil {
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()

// Read and process body manually
body, err := io.ReadAll(resp.Body)
if err != nil {
    return fmt.Errorf("failed to read body: %w", err)
}
```

> **Note:** Always close the response body with `defer resp.Body.Close()` to prevent resource leaks.

**When to use basic functions:**
- You need streaming or chunked reading
- Response size is very large
- You want to process headers before reading body
- You're implementing custom body processing

### The String Functions

The `CurlString*` functions return three values: the response body as a string (first), the HTTP response (second), and an error (third).

```go
// Note: body is the FIRST return value
body, resp, err := gocurl.CurlString(ctx, "https://api.github.com/users/octocat")
if err != nil {
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()

// Body is already loaded as a string
fmt.Println("User data:", body)
```

> **Warning:** The `CurlString*` functions load the entire response body into memory. Don't use them for large responses (> 10MB).

**When to use string functions:**
- Response is text-based (JSON, XML, HTML)
- Response size is manageable (< 10MB)
- You need the body as a string for parsing

**Decision tree:**

```
Need response body as string?
│
├─ Yes → Response size < 10MB?
│        │
│        ├─ Yes → Use CurlString()
│        └─ No → Use Curl() + streaming
│
└─ No → Use appropriate function for data type
```

This excerpt demonstrates:
- ✅ Clear section structure
- ✅ Professional but approachable voice
- ✅ Correct API signatures
- ✅ Proper code examples with error handling
- ✅ Helpful callout boxes
- ✅ Decision tree for choosing functions
- ✅ Consistent formatting

---

**END OF STYLE GUIDE**
