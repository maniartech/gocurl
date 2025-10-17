# GoCurl Book: Writing Style Guide

**Document Version:** 1.0
**Last Updated:** October 17, 2025
**Purpose:** Maintain consistent voice, tone, and quality across all chapters

---

## Core Writing Principles

### 1. Sweet, Simple, Robust (SSR)

**Sweet (Reader Experience)**
- Keep cognitive load minimal
- Use clear, direct language
- Make examples copy-paste ready
- Celebrate small wins ("You just made your first request!")

**Simple (Implementation)**
- Avoid over-engineering explanations
- Focus on what works
- Skip unnecessary theory
- Practical > Academic

**Robust (Production Quality)**
- All code examples must work
- Include error handling always
- Show production patterns
- Explain security implications

---

## Voice & Tone

### Voice: "Experienced Colleague"

Write as if you're a senior developer explaining to a colleague:

**✅ Good:**
> "Let's copy this curl command from the Stripe docs and paste it directly into our Go code. No translation needed—it just works."

**❌ Avoid (Too Academic):**
> "The architectural paradigm of GoCurl leverages polymorphic abstractions to facilitate isomorphic code reusability across disparate API implementations."

**❌ Avoid (Too Casual):**
> "Dude, just slam that curl command in there and boom! You're done. Ez pz lemon squeezy."

### Tone Guidelines

**Be:**
- ✅ Conversational but professional
- ✅ Encouraging and supportive
- ✅ Direct and clear
- ✅ Respectful of reader's time
- ✅ Honest about trade-offs

**Avoid:**
- ❌ Condescending ("Obviously..." / "Clearly...")
- ❌ Hyperbolic ("revolutionary" / "game-changing")
- ❌ Jargon without explanation
- ❌ Assumptions about prior knowledge
- ❌ Filler words and fluff

---

## Person & Tense

### Person

**Use second person ("you"):**
```markdown
You'll learn how to copy curl commands from API documentation and use them
directly in your Go code.
```

**Not first person plural ("we"):**
```markdown
❌ We'll learn how to copy curl commands...
```

**Exception:** Use "we" when working through code together:
```markdown
✅ Let's build a GitHub client together. We'll start by...
```

### Tense

**Present tense for explanations:**
```markdown
✅ GoCurl parses the curl command and creates an HTTP request.
❌ GoCurl will parse the curl command...
```

**Future tense for what readers will do:**
```markdown
✅ You'll build a complete API client in this chapter.
```

---

## Code Examples

### Every Example Must:

1. **Be Complete and Runnable**
   - No pseudocode
   - No `// ...rest of code` omissions
   - Include all imports
   - Include `package main` when appropriate

2. **Handle Errors**
   - Never ignore errors with `_`
   - Show proper error handling patterns
   - Explain what can go wrong

3. **Include Comments**
   - Explain non-obvious parts
   - Call out important patterns
   - Highlight common mistakes

4. **Follow Go Conventions**
   - gofmt formatted
   - golint clean
   - Idiomatic Go style

### Code Example Template

```go
// Title: What this example demonstrates
//
// Description: Longer explanation of what's happening and why.
// Include any prerequisites or setup needed.

package main

import (
    "context"
    "fmt"
    "log"

    "github.com/maniartech/gocurl"
)

func main() {
    // Step 1: Setup/preparation
    ctx := context.Background()
    url := "https://api.example.com/data"

    // Step 2: Make request
    resp, body, err := gocurl.Curl(ctx, url)
    if err != nil {
        // Explain what errors might occur
        log.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    // Step 3: Process response
    fmt.Println("Status:", resp.StatusCode)
    fmt.Println("Body:", body)
}

// Expected output:
// Status: 200
// Body: {"message":"success"}
```

### Code Annotations

Use emoji sparingly for callouts:

- 🎯 **Key Concept:** Important pattern or technique
- ⚠️ **Warning:** Common mistake or pitfall
- 💡 **Tip:** Helpful optimization or trick
- 🔒 **Security:** Security-related note
- 📊 **Performance:** Performance consideration

```go
// 🎯 Always defer response body close to prevent leaks
defer resp.Body.Close()

// ⚠️ Don't forget to check errors before using response
if err != nil {
    return err
}

// 💡 Use context with timeout for better control
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

---

## Structure Standards

### Chapter Structure

Every chapter follows this structure:

```markdown
# Chapter X: Title

## Learning Objectives
- Bullet point 1
- Bullet point 2
- Bullet point 3

## Prerequisites
What the reader should know/have before starting.

## Introduction (2-3 paragraphs)
Why this chapter matters. Real-world context.

## Section 1: Topic Name
### Explanation
### Code Example
### Common Pitfalls
### Best Practices

## Section 2: Topic Name
[Same structure]

## Hands-On Project
Complete working project with:
- Clear objective
- Step-by-step instructions
- Complete code
- Expected output
- Explanation of what was learned

## Summary
- Key takeaway 1
- Key takeaway 2
- Key takeaway 3

## Exercises
1. Easy exercise
2. Medium exercise
3. Hard exercise

## Further Reading
- Links to documentation
- Related articles
- Additional resources

## Next Chapter Preview
Teaser for next chapter
```

### Page Count Guidelines

- **Introduction/Preface:** 10-15 pages
- **Foundation chapters (1-4):** 18-25 pages
- **Core chapters (5-9):** 20-28 pages
- **Production chapters (10-13):** 24-30 pages
- **Advanced chapters (14-16):** 22-28 pages
- **Appendices:** 10-20 pages each

### Paragraph Guidelines

- **Length:** 3-5 sentences maximum
- **One idea per paragraph**
- **Use transitions** between paragraphs
- **Break up long explanations** with subheadings

---

## Formatting Standards

### Headings

```markdown
# Chapter Title (H1 - only once per chapter)

## Major Section (H2)

### Subsection (H3)

#### Detail Section (H4 - use sparingly)
```

### Lists

**Bullet lists** for unordered items:
```markdown
- Item 1
- Item 2
- Item 3
```

**Numbered lists** for sequential steps:
```markdown
1. First step
2. Second step
3. Third step
```

**Use checkboxes** for task lists:
```markdown
- [ ] Task to complete
- [x] Completed task
```

### Emphasis

- **Bold** for important terms, UI elements: `**important**`
- *Italic* for emphasis, new terms: `*emphasis*`
- `Code` for inline code, commands: `` `code` ``
- ~~Strikethrough~~ for deprecated: `~~old way~~`

### Links

```markdown
[Link text](https://example.com)

[Documentation](https://gocurl.dev/docs)

[Chapter 3](#chapter-3-core-concepts)
```

### Images & Diagrams

```markdown
![Alt text for accessibility](path/to/image.png)

**Figure 1.1:** Clear caption explaining the diagram
```

---

## Callout Boxes

### Quick Tip

```markdown
> 🎯 **Quick Tip**
>
> Short, actionable advice that helps immediately.
```

### Warning

```markdown
> ⚠️ **Warning**
>
> Common mistake or pitfall to avoid.
```

### Best Practice

```markdown
> 💡 **Best Practice**
>
> Production-ready pattern or recommendation.
```

### Deep Dive

```markdown
> 🔬 **Deep Dive** (Optional)
>
> Advanced explanation for curious readers. Can be skipped.
```

### Performance Note

```markdown
> 📊 **Performance Note**
>
> Performance consideration or optimization technique.
```

### Security Alert

```markdown
> 🔒 **Security Alert**
>
> Security implication or best practice.
```

---

## Terminology

### Consistent Terms

Use these terms consistently:

| ✅ Use This | ❌ Not This |
|------------|------------|
| GoCurl | gocurl, Gocurl, go-curl |
| curl command | cURL command, CURL |
| API | api, Api |
| HTTP | Http, http |
| JSON | Json, json |
| REST API | RESTful API, Rest API |
| request/response | req/resp (except in code) |

### Capitalization

- **GoCurl**: Always capitalize both G and C
- **HTTP, API, REST, JSON, TLS, SSL**: Always uppercase
- **Go**: Capitalize when referring to language
- **curl**: Lowercase when referring to curl tool

---

## Examples Standards

### Example Naming

```go
// ✅ Descriptive names
func getUserProfile(userID int) (*User, error)
func createGitHubIssue(title, body string) (*Issue, error)

// ❌ Generic names
func doStuff() error
func process() error
```

### Example Complexity

**Progressive disclosure:**

1. **First example:** Simplest possible (GET request)
2. **Second example:** Add one complexity (headers)
3. **Third example:** Add another (POST with JSON)
4. **Final example:** Complete, production-ready

**Example progression:**

```go
// Example 1: Simplest
resp, body, err := gocurl.Curl(ctx, "https://api.example.com")

// Example 2: Add authentication
resp, body, err := gocurl.CurlCommand(ctx,
    `curl -H "Authorization: Bearer token" https://api.example.com`)

// Example 3: Add POST
resp, body, err := gocurl.CurlCommand(ctx,
    `curl -X POST -H "Authorization: Bearer token" \
         -d '{"key":"value"}' https://api.example.com`)

// Example 4: Production-ready
func makeRequest(ctx context.Context, token, data string) (*Response, error) {
    resp, body, err := gocurl.CurlCommand(ctx,
        fmt.Sprintf(`curl -X POST \
            -H "Authorization: Bearer %s" \
            -H "Content-Type: application/json" \
            -d '%s' \
            https://api.example.com`, token, data))

    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
    }

    return parseResponse(body)
}
```

---

## Real-World Examples

### Use Real APIs

**✅ Do use:**
- GitHub API (free, well-documented)
- JSONPlaceholder (free test API)
- httpbin.org (HTTP testing)
- Stripe API (with test keys)

**❌ Don't use:**
- Fictional APIs (example.com/api)
- Oversimplified toy examples
- APIs without documentation

### Include API Keys Pattern

```go
// ✅ Show secure pattern
token := os.Getenv("GITHUB_TOKEN")
if token == "" {
    log.Fatal("GITHUB_TOKEN environment variable required")
}

// ❌ Don't hardcode
token := "ghp_xxxxxxxxxxxxx" // ❌ Never do this
```

---

## Error Handling

### Always Show Errors

```go
// ✅ Proper error handling
resp, body, err := gocurl.Curl(ctx, url)
if err != nil {
    // Explain possible errors
    if errors.Is(err, context.DeadlineExceeded) {
        return fmt.Errorf("request timed out: %w", err)
    }
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Body.Close()

// Check HTTP status
if resp.StatusCode >= 400 {
    return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, body)
}

// ❌ Never ignore errors
resp, body, _ := gocurl.Curl(ctx, url) // ❌
```

### Explain What Can Fail

```go
// 🎯 Common error scenarios:
// - Network timeout (context deadline exceeded)
// - DNS resolution failure
// - Connection refused
// - HTTP error status (4xx, 5xx)
// - Invalid response body
```

---

## Performance Discussion

### Include Benchmarks

```go
// Example benchmark
func BenchmarkRequest(b *testing.B) {
    ctx := context.Background()
    url := "https://httpbin.org/get"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        resp, _, err := gocurl.Curl(ctx, url)
        if err != nil {
            b.Fatal(err)
        }
        resp.Body.Close()
    }
}
```

### Show Comparisons

```markdown
| Client | Time (ns/op) | Allocs/op | Bytes/op |
|--------|--------------|-----------|----------|
| net/http | 12,450 | 28 | 3,456 |
| GoCurl | 980 | 0 | 0 |

**GoCurl is 12.7x faster with zero allocations.**
```

---

## Review Checklist

Before submitting a chapter:

### Content Quality
- [ ] Learning objectives are clear
- [ ] Introduction explains "why"
- [ ] Sections build progressively
- [ ] Summary captures key points
- [ ] Exercises test understanding

### Code Quality
- [ ] All code compiles
- [ ] All code runs successfully
- [ ] Error handling is shown
- [ ] Comments explain non-obvious parts
- [ ] Examples are production-ready

### Writing Quality
- [ ] Voice is consistent (experienced colleague)
- [ ] Tone is encouraging
- [ ] Paragraphs are short (3-5 sentences)
- [ ] No jargon without explanation
- [ ] No assumed knowledge gaps

### Technical Accuracy
- [ ] All claims are verified
- [ ] Benchmarks are real
- [ ] Security advice is current
- [ ] Best practices are followed
- [ ] Links work

### Formatting
- [ ] Headings follow hierarchy
- [ ] Code blocks have language tags
- [ ] Lists are properly formatted
- [ ] Callouts use correct emoji
- [ ] Images have alt text

---

## Common Mistakes to Avoid

### Don't

1. **Assume knowledge**
   - ❌ "As you know, HTTP/2 multiplexing..."
   - ✅ "HTTP/2 supports multiplexing, which means..."

2. **Use unnecessarily complex examples**
   - ❌ 100-line example for simple concept
   - ✅ Minimal example that demonstrates concept

3. **Skip error handling**
   - ❌ `resp, body, _ := gocurl.Curl(...)`
   - ✅ Always check and handle errors

4. **Use toy examples**
   - ❌ `curl https://example.com/api`
   - ✅ `curl https://api.github.com/user`

5. **Ignore security**
   - ❌ Hardcode API keys
   - ✅ Use environment variables

6. **Make unverified claims**
   - ❌ "GoCurl is the fastest HTTP client"
   - ✅ "Benchmarks show GoCurl is 12x faster than net/http for simple requests"

---

## Words to Avoid

| Instead of... | Use... |
|--------------|--------|
| "Obviously" | Explain it |
| "Simply" | Show the steps |
| "Just" | Be specific |
| "Easy" | "Straightforward" |
| "Clearly" | Demonstrate it |
| "Basically" | Skip it |
| "Actually" | Skip it |
| "Literally" | Skip it |

---

## Accessibility

### Alt Text for Images

```markdown
![Architecture diagram showing request flow from user code through GoCurl to HTTP server](./diagrams/request-flow.png)
```

### Code Contrast

- Use syntax highlighting
- Ensure code is readable
- Avoid light gray comments

### Clear Language

- Short sentences
- Active voice
- Simple words where possible
- Define technical terms

---

## Final Polish

### Before Publication

1. **Read aloud** - If it sounds awkward, rewrite
2. **Run code** - Every example must work
3. **Check links** - All URLs must be valid
4. **Spell check** - No typos
5. **Format check** - Consistent formatting
6. **Peer review** - Get feedback
7. **Technical review** - Expert validation

---

**Remember:** You're teaching someone to be a better developer. Be the mentor you wish you had.

---

**Last Updated:** October 17, 2025
**Version:** 1.0
**Next Review:** Before each chapter publication
