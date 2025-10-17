# Chapter 2: Installation & Setup

> **Learning Objectives**
>
> By the end of this chapter, you will:
> - Install the gocurl library in your Go projects
> - Set up the gocurl CLI tool for testing
> - Configure your IDE for optimal productivity
> - Verify your installation with working examples
> - Organize your workspace for gocurl projects
> - Understand Go modules and dependency management

## Introduction

Getting started with gocurl is straightforward, but proper setup ensures smooth development. This chapter walks you through installing both the library and CLI tool, configuring your development environment, and verifying everything works correctly.

Unlike some libraries that require complex configuration, gocurl follows Go's philosophy of simplicity. You'll be making API calls within minutes of installation.

## Prerequisites

Before installing gocurl, ensure you have:

- **Go 1.21 or later** - gocurl uses modern Go features
- **Git** - Required for `go get` to download dependencies
- **Internet connection** - For downloading the library and testing
- **Terminal/command line** - Basic familiarity with command-line tools
- **Text editor or IDE** - VS Code, GoLand, or any Go-compatible editor

**Check your Go version:**

```bash
go version
# Output should be: go version go1.21 or higher
```

If you need to install or upgrade Go, visit [golang.org/dl](https://golang.org/dl).

## Installing the GoCurl Library

The gocurl library is installed as a standard Go module dependency. There are two common scenarios: adding it to an existing project or starting a new project.

### Option 1: Adding to an Existing Project

If you already have a Go project with a `go.mod` file:

```bash
# Navigate to your project directory
cd myproject

# Add gocurl dependency
go get github.com/maniartech/gocurl

# Verify it's added to go.mod
grep gocurl go.mod
```

**Output:**
```
github.com/maniartech/gocurl v1.0.0
```

### Option 2: Starting a New Project

For a brand new project:

```bash
# Create project directory
mkdir my-api-client
cd my-api-client

# Initialize Go module
go mod init example.com/my-api-client

# Add gocurl dependency
go get github.com/maniartech/gocurl
```

This creates two files:
- `go.mod` - Module definition and dependencies
- `go.sum` - Cryptographic checksums for dependencies

### Verifying the Installation

Create a simple test file to verify gocurl is installed correctly:

```go
// main.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/maniartech/gocurl"
)

func main() {
	// Simple GET request
	body, resp, err := gocurl.CurlString(
		context.Background(),
		"https://api.github.com/zen",
	)

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer resp.Body.Close()

	fmt.Println("✅ GoCurl is working!")
	fmt.Println("Status:", resp.StatusCode)
	fmt.Println("GitHub Zen:", body)
}
```

**Run the test:**

```bash
go run main.go

# Expected output:
# ✅ GoCurl is working!
# Status: 200
# GitHub Zen: Design for failure.
```

If you see the output above, gocurl is successfully installed! 🎉

### Common Installation Issues

**Issue: "package github.com/maniartech/gocurl is not in GOROOT"**

**Solution:** Run `go mod tidy` to download dependencies:
```bash
go mod tidy
```

**Issue: "go: cannot find main module"**

**Solution:** You're not in a Go module directory. Run `go mod init`:
```bash
go mod init your-module-name
```

**Issue: Version conflicts**

**Solution:** Update to latest version:
```bash
go get -u github.com/maniartech/gocurl
go mod tidy
```

## Installing the GoCurl CLI Tool

The gocurl CLI tool is invaluable for testing API endpoints before integrating them into your code. It provides the CLI-to-code workflow that makes gocurl unique.

### Installing the CLI

Install the CLI globally using `go install`:

```bash
# Install latest version
go install github.com/maniartech/gocurl/cmd/gocurl@latest
```

This installs the `gocurl` binary in your `$GOPATH/bin` directory (usually `~/go/bin` on Unix-like systems or `%USERPROFILE%\go\bin` on Windows).

### Verify CLI Installation

```bash
# Check if gocurl is in your PATH
gocurl --version

# Expected output:
# gocurl version 1.0.0
```

**If command not found:**

The `gocurl` binary is installed but not in your PATH. Add Go's bin directory:

**Linux/macOS:**
```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH=$PATH:$(go env GOPATH)/bin

# Reload shell
source ~/.bashrc  # or ~/.zshrc
```

**Windows (PowerShell):**
```powershell
# Add to PowerShell profile
$env:Path += ";$(go env GOPATH)\bin"

# Or permanently through System Properties > Environment Variables
```

### Basic CLI Usage

Test the CLI with a simple request:

```bash
# Simple GET request
gocurl https://api.github.com/zen

# Output: Design for failure.
```

**With headers:**
```bash
gocurl -H "Accept: application/json" https://httpbin.org/get
```

**POST request:**
```bash
gocurl -X POST -d "name=John" https://httpbin.org/post
```

**Save to file:**
```bash
gocurl -o output.json https://api.github.com/repos/golang/go
```

We'll explore the CLI in depth in Chapter 4. For now, knowing it's installed and working is sufficient.

## IDE Setup and Configuration

A well-configured IDE significantly improves productivity. Let's set up the two most popular Go IDEs: Visual Studio Code and GoLand.

### Visual Studio Code Setup

VS Code is a free, lightweight editor with excellent Go support.

**1. Install VS Code Extensions:**

Install the official Go extension:
- Open VS Code
- Press `Ctrl+Shift+X` (or `Cmd+Shift+X` on macOS)
- Search for "Go"
- Install the extension by Go Team at Google

**2. Install Go Tools:**

The Go extension needs supporting tools. Open the command palette (`Ctrl+Shift+P`) and run:
```
Go: Install/Update Tools
```

Select all tools and click OK.

**3. Create Code Snippets:**

Add gocurl-specific snippets for faster development. Create `.vscode/gocurl.code-snippets` in your project:

```json
{
  "GoCurl String Request": {
    "prefix": "gocurlstr",
    "body": [
      "body, resp, err := gocurl.CurlString(${1:ctx}, \"${2:url}\")",
      "if err != nil {",
      "\treturn ${3:err}",
      "}",
      "defer resp.Body.Close()",
      "$0"
    ],
    "description": "GoCurl string request with error handling"
  },
  "GoCurl JSON Request": {
    "prefix": "gocurljson",
    "body": [
      "var ${1:result} ${2:Type}",
      "resp, err := gocurl.CurlJSON(${3:ctx}, &${1:result}, \"${4:url}\")",
      "if err != nil {",
      "\treturn ${5:err}",
      "}",
      "defer resp.Body.Close()",
      "$0"
    ],
    "description": "GoCurl JSON request with unmarshaling"
  },
  "GoCurl Command": {
    "prefix": "gocurlcmd",
    "body": [
      "body, resp, err := gocurl.CurlStringCommand(${1:ctx},",
      "\t`curl ${2:command}`)",
      "if err != nil {",
      "\treturn ${3:err}",
      "}",
      "defer resp.Body.Close()",
      "$0"
    ],
    "description": "GoCurl command-style request"
  },
  "GoCurl Context with Timeout": {
    "prefix": "gocurlctx",
    "body": [
      "ctx, cancel := context.WithTimeout(context.Background(), ${1:10}*time.Second)",
      "defer cancel()",
      "",
      "body, resp, err := gocurl.CurlString(ctx, \"${2:url}\")",
      "if err != nil {",
      "\treturn ${3:err}",
      "}",
      "defer resp.Body.Close()",
      "$0"
    ],
    "description": "GoCurl request with timeout context"
  }
}
```

**Usage:** Type the prefix (e.g., `gocurlstr`) and press Tab to expand the snippet.

**4. Configure Settings:**

Add to `.vscode/settings.json`:

```json
{
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "editor.formatOnSave": true,
  "go.useLanguageServer": true,
  "[go]": {
    "editor.codeActionsOnSave": {
      "source.organizeImports": true
    }
  }
}
```

### GoLand/IntelliJ IDEA Setup

GoLand is JetBrains' commercial Go IDE with advanced features.

**1. Enable Go Modules:**
- Settings → Go → Go Modules
- Check "Enable Go modules integration"

**2. Create Live Templates:**

Settings → Editor → Live Templates → Go:

**Template 1: gocurlstr**
```go
body, resp, err := gocurl.CurlString($CTX$, "$URL$")
if err != nil {
    return $ERR$
}
defer resp.Body.Close()
$END$
```

**Template 2: gocurljson**
```go
var $VAR$ $TYPE$
resp, err := gocurl.CurlJSON($CTX$, &$VAR$, "$URL$")
if err != nil {
    return $ERR$
}
defer resp.Body.Close()
$END$
```

**3. Configure External Tools:**

Add gocurl CLI as an external tool:
- Settings → Tools → External Tools → Add
- Name: `Test with GoCurl CLI`
- Program: `gocurl`
- Arguments: `$Prompt$`
- Working directory: `$ProjectFileDir$`

Now you can run `Tools → External Tools → Test with GoCurl CLI` to quickly test endpoints.

## Workspace Organization

Organizing your project structure from the start saves headaches later. Here's a recommended structure for gocurl-based projects.

### Recommended Project Structure

```
my-api-client/
├── go.mod                    # Module definition
├── go.sum                    # Dependency checksums
├── README.md                 # Project documentation
├── .gitignore               # Git ignore file
├── cmd/                     # Command-line applications
│   └── myapp/
│       └── main.go          # Main application entry
├── internal/                # Private application code
│   ├── api/                 # API clients
│   │   ├── github.go        # GitHub API client
│   │   ├── stripe.go        # Stripe API client
│   │   └── slack.go         # Slack API client
│   ├── config/              # Configuration
│   │   └── config.go
│   └── models/              # Data models
│       └── user.go
├── pkg/                     # Public, reusable packages
│   └── httpclient/
│       └── client.go        # Reusable HTTP utilities
└── test/                    # Test files
    ├── integration/
    │   └── api_test.go
    └── fixtures/
        └── test_data.json
```

### Why This Structure?

**`cmd/`** - Application entry points
- Multiple command-line tools can coexist
- Each gets its own subdirectory
- Clear separation of concerns

**`internal/`** - Private code
- Cannot be imported by external projects
- Safe for implementation details
- API clients belong here

**`pkg/`** - Public, reusable code
- Can be imported by other projects
- Generic utilities and helpers
- Stable, well-documented interfaces

**`test/`** - Test code and fixtures
- Integration tests separate from unit tests
- Test data and fixtures organized
- Easy to run specific test suites

### Example: GitHub API Client

Let's create a properly structured API client:

**Project structure:**
```
github-client/
├── go.mod
├── cmd/
│   └── ghcli/
│       └── main.go
└── internal/
    └── api/
        └── github.go
```

**internal/api/github.go:**
```go
package api

import (
	"context"
	"fmt"

	"github.com/maniartech/gocurl"
)

// GitHubClient handles GitHub API interactions
type GitHubClient struct {
	token   string
	baseURL string
}

// NewGitHubClient creates a new GitHub API client
func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		token:   token,
		baseURL: "https://api.github.com",
	}
}

// User represents a GitHub user
type User struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Company   string `json:"company"`
	Location  string `json:"location"`
	Bio       string `json:"bio"`
	PublicRepos int  `json:"public_repos"`
}

// GetAuthenticatedUser fetches the authenticated user's profile
func (c *GitHubClient) GetAuthenticatedUser(ctx context.Context) (*User, error) {
	var user User

	resp, err := gocurl.CurlJSONCommand(ctx, &user,
		`curl -H "Accept: application/vnd.github+json" \
		      -H "Authorization: Bearer `+c.token+`" \
		      `+c.baseURL+`/user`)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	return &user, nil
}

// Repository represents a GitHub repository
type Repository struct {
	Name            string `json:"name"`
	FullName        string `json:"full_name"`
	Description     string `json:"description"`
	Private         bool   `json:"private"`
	StargazersCount int    `json:"stargazers_count"`
	ForksCount      int    `json:"forks_count"`
	Language        string `json:"language"`
}

// ListUserRepos fetches repositories for a user
func (c *GitHubClient) ListUserRepos(ctx context.Context, username string) ([]Repository, error) {
	var repos []Repository

	url := fmt.Sprintf("%s/users/%s/repos", c.baseURL, username)
	resp, err := gocurl.CurlJSON(ctx, &repos,
		`-H "Accept: application/vnd.github+json"`,
		url)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch repos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	return repos, nil
}
```

**cmd/ghcli/main.go:**
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"example.com/github-client/internal/api"
)

func main() {
	// Get token from environment
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN environment variable required")
	}

	// Create client
	client := api.NewGitHubClient(token)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Fetch authenticated user
	user, err := client.GetAuthenticatedUser(ctx)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("\n👤 %s (@%s)\n", user.Name, user.Login)
	fmt.Printf("📧 %s\n", user.Email)
	fmt.Printf("🏢 %s\n", user.Company)
	fmt.Printf("📍 %s\n", user.Location)
	fmt.Printf("📝 %s\n", user.Bio)
	fmt.Printf("📦 %d public repos\n\n", user.PublicRepos)

	// List user's repositories
	repos, err := client.ListUserRepos(ctx, user.Login)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Top repositories:\n")
	for i, repo := range repos {
		if i >= 5 {
			break
		}
		fmt.Printf("  %d. ⭐ %d - %s\n", i+1, repo.StargazersCount, repo.FullName)
	}
}
```

**Usage:**
```bash
export GITHUB_TOKEN=your_token_here
go run cmd/ghcli/main.go
```

This structure scales well as your project grows, with clear separation between application logic, API clients, and entry points.

### .gitignore Configuration

Create a `.gitignore` file to exclude unnecessary files:

```gitignore
# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
/bin/
/cmd/*/bin/

# Test binary
*.test

# Output of the go coverage tool
*.out

# Dependency directories
/vendor/

# Go workspace file
go.work

# IDE specific
.vscode/
.idea/
*.swp
*.swo
*~

# OS specific
.DS_Store
Thumbs.db

# Environment files (contains secrets)
.env
.env.local
*.key
*.pem

# Output files
/output/
/tmp/
```

## Environment Configuration

Proper environment configuration keeps secrets secure and configurations manageable.

### Using Environment Variables

**Create .env file (NOT committed to git):**
```bash
# .env
GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
STRIPE_API_KEY=sk_test_xxxxxxxxxxxxxxxxx
API_BASE_URL=https://api.example.com
REQUEST_TIMEOUT=30s
```

**Load in your application:**

```go
package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	GitHubToken    string
	StripeAPIKey   string
	APIBaseURL     string
	RequestTimeout time.Duration
}

func Load() *Config {
	// Load .env file (development only)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	timeout, _ := time.ParseDuration(os.Getenv("REQUEST_TIMEOUT"))
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Config{
		GitHubToken:    os.Getenv("GITHUB_TOKEN"),
		StripeAPIKey:   os.Getenv("STRIPE_API_KEY"),
		APIBaseURL:     os.Getenv("API_BASE_URL"),
		RequestTimeout: timeout,
	}
}
```

**Install godotenv:**
```bash
go get github.com/joho/godotenv
```

### Configuration Best Practices

1. **Never commit secrets** - Use `.env` files (gitignored)
2. **Provide defaults** - Application works without environment variables
3. **Validate on startup** - Fail fast if required config missing
4. **Use typed configuration** - Structs over raw string access
5. **Document required variables** - In README.md

## Verification Test Suite

Let's create a comprehensive test suite to verify your installation.

**Create test/verification_test.go:**

```go
package test

import (
	"context"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
)

func TestGoCurlBasicInstallation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body, resp, err := gocurl.CurlString(ctx, "https://httpbin.org/get")
	if err != nil {
		t.Fatalf("Basic request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if len(body) == 0 {
		t.Error("Expected non-empty response body")
	}

	t.Log("✅ Basic installation verified")
}

func TestGoCurlJSONUnmarshaling(t *testing.T) {
	ctx := context.Background()

	var data map[string]interface{}
	resp, err := gocurl.CurlJSON(ctx, &data, "https://httpbin.org/json")
	if err != nil {
		t.Fatalf("JSON request failed: %v", err)
	}
	defer resp.Body.Close()

	if data == nil {
		t.Error("Expected non-nil JSON data")
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	t.Log("✅ JSON unmarshaling verified")
}

func TestGoCurlContextTimeout(t *testing.T) {
	// Very short timeout to test context handling
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// This should timeout
	_, _, err := gocurl.CurlString(ctx, "https://httpbin.org/delay/5")
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	t.Log("✅ Context timeout verified")
}

func TestGoCurlCommandStyle(t *testing.T) {
	ctx := context.Background()

	body, resp, err := gocurl.CurlStringCommand(ctx,
		`curl -H "Accept: application/json" https://httpbin.org/headers`)

	if err != nil {
		t.Fatalf("Command-style request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if len(body) == 0 {
		t.Error("Expected non-empty response")
	}

	t.Log("✅ Command-style syntax verified")
}

func TestGoCurlHTTPS(t *testing.T) {
	ctx := context.Background()

	// Test HTTPS connection
	_, resp, err := gocurl.CurlString(ctx, "https://api.github.com/zen")
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	t.Log("✅ HTTPS support verified")
}
```

**Run the verification suite:**

```bash
go test ./test/... -v

# Expected output:
# === RUN   TestGoCurlBasicInstallation
#     verification_test.go:25: ✅ Basic installation verified
# --- PASS: TestGoCurlBasicInstallation (0.45s)
# === RUN   TestGoCurlJSONUnmarshaling
#     verification_test.go:44: ✅ JSON unmarshaling verified
# --- PASS: TestGoCurlJSONUnmarshaling (0.38s)
# === RUN   TestGoCurlContextTimeout
#     verification_test.go:60: ✅ Context timeout verified
# --- PASS: TestGoCurlContextTimeout (0.01s)
# === RUN   TestGoCurlCommandStyle
#     verification_test.go:76: ✅ Command-style syntax verified
# --- PASS: TestGoCurlCommandStyle (0.42s)
# === RUN   TestGoCurlHTTPS
#     verification_test.go:91: ✅ HTTPS support verified
# --- PASS: TestGoCurlHTTPS (0.35s)
# PASS
```

If all tests pass, your installation is complete and working correctly! 🎉

## Troubleshooting Common Issues

### Issue: Import Cycle Detected

**Error:**
```
import cycle not allowed
package example.com/myapp
	imports example.com/myapp/internal/api
	imports example.com/myapp
```

**Solution:** Don't import `main` package from internal packages. Restructure to avoid circular dependencies.

### Issue: Module Not Found

**Error:**
```
cannot find module providing package github.com/maniartech/gocurl
```

**Solution:**
```bash
go mod tidy
go get github.com/maniartech/gocurl
```

### Issue: GOPATH Not Set

**Error:**
```
go: cannot install to ... which is listed in GOROOT
```

**Solution:** Set GOPATH:
```bash
# Linux/macOS
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# Windows
setx GOPATH %USERPROFILE%\go
setx PATH "%PATH%;%GOPATH%\bin"
```

### Issue: Version Conflicts

**Error:**
```
found packages with conflicting versions
```

**Solution:**
```bash
# Clean module cache
go clean -modcache

# Reinstall dependencies
go mod tidy
```

### Getting Help

- **Documentation:** https://pkg.go.dev/github.com/maniartech/gocurl
- **Issues:** https://github.com/maniartech/gocurl/issues
- **Discussions:** https://github.com/maniartech/gocurl/discussions
- **Go Community:** https://gophers.slack.com

## Summary

In this chapter, you:

- ✅ Installed the gocurl library using Go modules
- ✅ Installed the gocurl CLI tool globally
- ✅ Configured your IDE (VS Code or GoLand) with snippets and settings
- ✅ Learned recommended project structure for gocurl applications
- ✅ Set up environment configuration for secure secret management
- ✅ Created and ran a verification test suite
- ✅ Know how to troubleshoot common installation issues

Your development environment is now ready for building production-grade API clients with gocurl!

## What's Next?

In **Chapter 3: Core Concepts**, we'll dive deep into:
- The dual API approach (Curl-syntax vs Builder pattern)
- Understanding all six function categories
- Variable expansion and substitution
- Context usage for timeouts and cancellation
- Response handling patterns
- The `Process()` function as the core execution engine

You'll gain a complete understanding of gocurl's architecture and how to choose the right function for every use case.
