# Writing Guide - Quick Start

**For Authors Working on "The Definitive Guide to the GoCurl Library"**

## Before You Start

**Read these documents first (in order):**
1. `__MASTER_PLAN.md` - Understand the complete book vision
2. `outline.md` - See the chapter structure (Chapters 1-3 are complete templates)
3. `style_guide.md` - Learn the writing standards
4. `API_REFERENCE.md` - Reference for all API signatures
5. `CODE_STANDARDS.md` - Requirements for code examples

## Chapter Writing Workflow

### Step 1: Review Chapter Outline

Open `outline.md` and find your chapter. Review:
- Learning objectives
- Content sections
- Hands-on project requirements
- Page count target

**Example:** Chapter 5 is allocated 30 pages covering Builder Pattern.

### Step 2: Study Completed Examples

Review Chapters 1-3 in `outline.md` to see:
- How learning objectives are written
- Section organization patterns
- Code example quality
- Hands-on project structure
- Summary format

**Pattern:** Learning Objectives → Content Sections → Hands-On Project → Summary

### Step 3: Gather Real Examples

**Find test files for your chapter topic:**

```bash
# Example: For Builder Pattern chapter
cd e:/Projects/go-libs/gocurl
grep -r "NewRequestOptionsBuilder" *_test.go

# Example: For Retry chapter
grep -r "RetryConfig" *_test.go

# Example: For Middleware chapter
grep -r "MiddlewareFunc" *_test.go
```

**Use real, working code from tests** - Never create fictional examples.

### Step 4: Create Chapter Directory

```bash
# Example: Chapter 5 (Builder Pattern)
cd book2/part2-api-approaches/chapter05-builder-pattern

# Create initial files
touch chapter.md           # Main chapter content
touch examples/example1.go # Code examples
touch examples/example1_test.go
touch exercises/exercise1.md
```

### Step 5: Write Content

**Follow this template:**

````markdown
# Chapter 5: RequestOptions & Builder Pattern

*[Opening paragraph: What this chapter covers and why it matters]*

## Learning Objectives

By the end of this chapter, you will:

- Understand the Builder pattern in gocurl
- Master fluent API for request configuration
- Learn when to use Builder vs Curl-syntax
- Build production-ready API clients with builders

## Understanding the Builder Pattern

*[3-6 pages of content]*

### What Is the Builder Pattern?

*[Explanation with examples]*

### Why Use the Builder Pattern?

*[Benefits and use cases]*

### Your First Builder

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
    ctx := context.Background()

    opts := gocurl.NewRequestOptionsBuilder().
        SetMethod("GET").
        SetURL("https://api.github.com/users/octocat").
        AddHeader("Accept", "application/vnd.github+json").
        SetTimeout(10 * time.Second).
        Build()

    resp, err := gocurl.Process(ctx, opts)
    if err != nil {
        log.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()

    fmt.Println("Status:", resp.StatusCode)
}
```

*[Continue with more sections...]*

## Hands-On Project: API Client with Builder Pattern

*[Complete working project with:
  - Project overview
  - Requirements
  - Full implementation
  - Testing instructions
  - Extension ideas]*

## Summary

In this chapter, we explored the Builder pattern:

- **Builder Pattern**: Fluent API for constructing RequestOptions
- **Key Methods**: SetMethod, SetURL, AddHeader, SetTimeout, Build
- **When to Use**: Complex configurations, readable code, IDE autocomplete
- **Production Pattern**: Reusable clients with default configurations

Key pattern to remember:

```go
opts := gocurl.NewRequestOptionsBuilder().
    SetMethod("POST").
    SetURL(url).
    AddHeader("Content-Type", "application/json").
    SetBody(jsonData).
    Build()
```

In the next chapter, we'll explore working with JSON APIs...
````

### Step 6: Create Code Examples

**In `examples/` directory:**

```go
// examples/builder_basic.go
package examples

import (
    "context"
    "time"

    "github.com/stackql/gocurl"
)

func BasicBuilder() error {
    ctx := context.Background()

    opts := gocurl.NewRequestOptionsBuilder().
        SetURL("https://api.github.com").
        SetTimeout(10 * time.Second).
        Build()

    resp, err := gocurl.Process(ctx, opts)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return nil
}
```

```go
// examples/builder_basic_test.go
package examples

import (
    "testing"
)

func TestBasicBuilder(t *testing.T) {
    if err := BasicBuilder(); err != nil {
        t.Fatalf("BasicBuilder failed: %v", err)
    }
}
```

### Step 7: Test Everything

```bash
# Format code
gofmt -w .

# Build examples
cd book2
go build ./...

# Run tests
go test ./...

# Check for race conditions
go test -race ./...

# Run go vet
go vet ./...
```

**All checks must pass before submitting.**

### Step 8: Verify Against Standards

Use the checklist from `CODE_STANDARDS.md`:

**Compilation:**
- [ ] Code compiles with `go build`
- [ ] No `go vet` warnings
- [ ] Formatted with `gofmt`
- [ ] All imports present

**Correctness:**
- [ ] API signatures match `API_REFERENCE.md`
- [ ] Return values in correct order
- [ ] Error handling complete

**Quality:**
- [ ] Follows `style_guide.md` standards
- [ ] Real examples (not fictional)
- [ ] Comments explain WHY, not WHAT
- [ ] Production-ready code

## Common Patterns

### Introducing New API Functions

**Three-step pattern:**

1. **Why it matters** (motivation)
2. **What it is** (definition)
3. **How to use it** (example)

### Showing Alternatives

**Use comparison tables:**

| Approach | Best For | Example |
|----------|----------|---------|
| Curl-syntax | Quick conversion | `Curl(ctx, "curl ...")` |
| Builder | Complex config | `NewRequestOptionsBuilder()...` |

### Progressive Complexity

Start simple, add features gradually:

```go
// Step 1: Basic
opts := gocurl.NewRequestOptionsBuilder().
    SetURL(url).
    Build()

// Step 2: Add auth
opts := gocurl.NewRequestOptionsBuilder().
    SetURL(url).
    SetBearerToken(token).
    Build()

// Step 3: Add retry
opts := gocurl.NewRequestOptionsBuilder().
    SetURL(url).
    SetBearerToken(token).
    SetRetryConfig(&gocurl.RetryConfig{MaxRetries: 3}).
    Build()
```

## Finding Examples in Test Files

**Builder Pattern examples:**
```bash
grep -A 20 "NewRequestOptionsBuilder" options/builder_test.go
```

**Retry examples:**
```bash
grep -A 30 "RetryConfig" retry_test.go
```

**Variable expansion:**
```bash
grep -A 20 "WithVars" variables_refactor_test.go
```

**JSON examples:**
```bash
grep -A 20 "CurlJSON" parity_test.go
```

## Quality Checklist

Before considering chapter complete:

- [ ] All learning objectives addressed
- [ ] 4-8 content sections (3-6 pages each)
- [ ] 10-20 code examples (all working)
- [ ] 1 complete hands-on project
- [ ] Summary section
- [ ] All code tested
- [ ] All examples from real tests
- [ ] Follows style guide
- [ ] Correct API signatures
- [ ] No TODOs or placeholders

## Getting Help

**Reference Documents:**
- Stuck on writing style? → Read `style_guide.md`
- Need API signature? → Check `API_REFERENCE.md`
- Code not compiling? → Review `CODE_STANDARDS.md`
- Need chapter structure? → See Chapters 1-3 in `outline.md`

**Finding Examples:**
1. Search test files: `grep -r "function_name" *_test.go`
2. Read test file directly: `cat parity_test.go`
3. Check source code: `cat api.go`, `cat options/builder.go`

## Timeline Estimates

**Per Chapter:**
- Research & planning: 2-4 hours
- Writing content: 8-12 hours
- Code examples: 4-6 hours
- Testing: 2-3 hours
- Review & polish: 2-3 hours

**Total per chapter: 18-28 hours**

For 19 chapters: ~380-530 hours of work

## Next Chapter Priorities

**Recommended order:**

1. **Chapter 4** (CLI) - Builds on Ch 1-3, relatively simple
2. **Chapter 5** (Builder) - Core API, many examples in tests
3. **Chapter 6** (JSON) - Common use case, lots of examples
4. **Chapter 10** (Retries) - Well-documented in retry_test.go
5. **Chapter 8** (Security) - Important, well-tested
6. Continue with remaining chapters...

## Contact

For questions about the writing process or standards, refer to:
- `__MASTER_PLAN.md` - Overall vision
- `style_guide.md` - Writing standards
- `CODE_STANDARDS.md` - Code requirements

---

**Remember:** Quality over speed. Every example must compile, run, and demonstrate real-world usage.

**Last Updated:** 2025
