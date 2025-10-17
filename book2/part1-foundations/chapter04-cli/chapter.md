# Chapter 4: Command Line Interface

## Introduction

The GoCurl CLI tool brings the power of GoCurl to your terminal, providing a drop-in replacement for curl that leverages all the features you've learned in previous chapters. Whether you're testing APIs, debugging HTTP issues, or automating workflows, the CLI offers a familiar interface with enhanced capabilities.

### What You'll Learn

- Installing and using the `gocurl` CLI tool
- Understanding CLI-specific options and flags
- Converting curl commands to gocurl seamlessly
- Testing and debugging with verbose output
- Integrating gocurl into shell scripts and CI/CD pipelines
- Advanced CLI workflows and patterns

### Why Use the CLI?

While the Go library is powerful for programmatic use, the CLI excels in:

- **Quick API testing**: Test endpoints without writing code
- **Curl command validation**: Verify curl commands work before using in code
- **Shell scripting**: Integrate into bash/zsh/PowerShell scripts
- **CI/CD pipelines**: Use in automated testing and deployment
- **Interactive debugging**: Get immediate feedback on API behavior
- **Documentation**: Generate curl examples for API documentation

## Installation

### From Go Install

The simplest way to install gocurl CLI is using `go install`:

```bash
go install github.com/maniartech/gocurl/cmd/gocurl@latest
```

This installs the `gocurl` binary to your `$GOPATH/bin` (usually `~/go/bin`).

### From Source

For development or customization:

```bash
git clone https://github.com/maniartech/gocurl.git
cd gocurl/cmd/gocurl
go build -o gocurl .
sudo mv gocurl /usr/local/bin/  # or add to PATH
```

### Verify Installation

```bash
gocurl --version  # Check version
gocurl --help     # Show usage
```

### Shell Completion (Optional)

Add completion for your shell:

```bash
# Bash
gocurl completion bash > /etc/bash_completion.d/gocurl

# Zsh
gocurl completion zsh > ~/.zsh/completion/_gocurl

# PowerShell
gocurl completion powershell > gocurl.ps1
```

## Basic Usage

### Simple GET Request

The most basic usage is identical to curl:

```bash
gocurl https://httpbin.org/get
```

Output:
```json
{
  "args": {},
  "headers": {
    "Host": "httpbin.org",
    "User-Agent": "Go-http-client/2.0"
  },
  "origin": "203.0.113.1",
  "url": "https://httpbin.org/get"
}
```

### POST Request with Data

```bash
gocurl -X POST -d '{"name":"Alice"}' \\
  -H "Content-Type: application/json" \\
  https://httpbin.org/post
```

### The 'curl' Prefix (Optional)

GoCurl accepts commands with or without the `curl` prefix:

```bash
# Both are equivalent:
gocurl https://example.com
gocurl curl https://example.com
```

This makes it easy to copy/paste curl commands from documentation.

## CLI-Specific Options

GoCurl adds several options beyond standard curl:

### Verbose Output (-v, --verbose)

Shows request details, headers, and timing:

```bash
gocurl -v https://httpbin.org/get
```

Output:
```
> GET /get HTTP/1.1
> Host: httpbin.org
> User-Agent: gocurl/1.0.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Content-Type: application/json
< Content-Length: 245
< Date: Mon, 01 Jan 2024 12:00:00 GMT
<
{
  "args": {},
  ...
}
```

### Include Headers (-i, --include)

Include response headers in output:

```bash
gocurl -i https://httpbin.org/get
```

Output:
```
HTTP/1.1 200 OK
Content-Type: application/json
Date: Mon, 01 Jan 2024 12:00:00 GMT

{
  "args": {},
  ...
}
```

### Silent Mode (-s, --silent)

Suppress all output except errors:

```bash
gocurl -s https://httpbin.org/get > response.json
```

Useful in scripts where you only want the response body.

### Output to File (-o, --output)

Write response to file:

```bash
gocurl -o response.json https://httpbin.org/get
```

### Custom Output Format (-w, --write-out)

Format output with variables:

```bash
gocurl -w "Status: %{http_code}\\nSize: %{size_download}\\n" https://httpbin.org/get
```

Variables:
- `%{http_code}` - HTTP status code
- `%{size_download}` - Downloaded size
- `%{time_total}` - Total time
- `%{url_effective}` - Final URL (after redirects)

## Standard Curl Options

All standard curl options are supported:

### HTTP Methods (-X, --request)

```bash
gocurl -X GET https://httpbin.org/get
gocurl -X POST https://httpbin.org/post
gocurl -X PUT https://httpbin.org/put
gocurl -X DELETE https://httpbin.org/delete
gocurl -X PATCH https://httpbin.org/patch
```

### Headers (-H, --header)

```bash
# Single header
gocurl -H "Authorization: Bearer $TOKEN" https://api.example.com

# Multiple headers
gocurl \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: secret" \\
  -H "User-Agent: MyApp/1.0" \\
  https://api.example.com
```

### Request Body (-d, --data)

```bash
# JSON data
gocurl -d '{"name":"Alice","email":"alice@example.com"}' \\
  -H "Content-Type: application/json" \\
  https://httpbin.org/post

# Form data
gocurl -d "name=Alice&email=alice@example.com" \\
  https://httpbin.org/post

# From file
gocurl -d @data.json https://httpbin.org/post
```

### Basic Authentication (-u, --user)

```bash
gocurl -u username:password https://httpbin.org/basic-auth/username/password
```

### Follow Redirects (-L, --location)

```bash
gocurl -L https://httpbin.org/redirect/3
```

### Custom User-Agent (-A, --user-agent)

```bash
gocurl -A "MyApp/1.0" https://httpbin.org/user-agent
```

### Timeout (--max-time)

```bash
gocurl --max-time 10 https://httpbin.org/delay/5
```

## Environment Variable Expansion

GoCurl automatically expands environment variables:

### Basic Expansion

```bash
export API_URL="https://api.example.com"
export API_KEY="secret-key-123"

gocurl -H "Authorization: Bearer $API_KEY" $API_URL/users
```

### Braces Syntax

```bash
gocurl -H "X-API-Key: ${API_KEY}" ${API_URL}/users
```

### Configuration File Pattern

```bash
# .env file
API_URL=https://api.example.com
API_KEY=secret-key
API_VERSION=v1

# Load and use
source .env
gocurl ${API_URL}/${API_VERSION}/users
```

## Multi-Line Commands

For complex commands, use line continuation:

### Backslash Continuation

```bash
gocurl \\
  -X POST \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer $TOKEN" \\
  -d '{"name":"Alice","email":"alice@example.com"}' \\
  https://api.example.com/users
```

### Comments

```bash
gocurl \\
  # POST request to create user
  -X POST \\
  # Content type
  -H "Content-Type: application/json" \\
  # Authentication
  -H "Authorization: Bearer $TOKEN" \\
  # Request body
  -d '{"name":"Alice"}' \\
  # API endpoint
  https://api.example.com/users
```

## Shell Scripting

### Basic Script

```bash
#!/bin/bash
# check-api.sh - Check API health

API_URL="https://api.example.com"

# Health check
response=$(gocurl -s ${API_URL}/health)
if [ $? -eq 0 ]; then
  echo "✅ API is healthy"
else
  echo "❌ API is down"
  exit 1
fi
```

### Extract Status Code

```bash
#!/bin/bash
# check-status.sh

status=$(gocurl -s -o /dev/null -w "%{http_code}" https://api.example.com)
if [ "$status" == "200" ]; then
  echo "✅ API returned 200"
else
  echo "❌ API returned $status"
  exit 1
fi
```

### Loop Through Endpoints

```bash
#!/bin/bash
# check-endpoints.sh

endpoints=(
  "/users"
  "/products"
  "/orders"
)

for endpoint in "${endpoints[@]}"; do
  echo "Checking $endpoint..."
  gocurl -s -o /dev/null -w "Status: %{http_code}\\n" \\
    https://api.example.com$endpoint
done
```

### Error Handling

```bash
#!/bin/bash
# api-call.sh - Robust API call with error handling

URL="https://api.example.com/data"

# Make request
response=$(gocurl -s --max-time 10 "$URL" 2>&1)
exit_code=$?

if [ $exit_code -eq 0 ]; then
  echo "Success: $response"
else
  case $exit_code in
    7)
      echo "Error: Failed to connect"
      ;;
    28)
      echo "Error: Operation timeout"
      ;;
    *)
      echo "Error: Unknown error (code: $exit_code)"
      ;;
  esac
  exit $exit_code
fi
```

## CI/CD Integration

### GitHub Actions

```yaml
name: API Tests

on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Install gocurl
        run: go install github.com/maniartech/gocurl/cmd/gocurl@latest

      - name: Test API
        run: |
          gocurl -s -o /dev/null -w "Status: %{http_code}\\n" \\
            https://api.example.com/health

      - name: Run Integration Tests
        run: |
          ./scripts/run-api-tests.sh
```

### GitLab CI

```yaml
stages:
  - test

api-tests:
  stage: test
  image: golang:1.21
  script:
    - go install github.com/maniartech/gocurl/cmd/gocurl@latest
    - export PATH=$PATH:$(go env GOPATH)/bin
    - gocurl https://api.example.com/health
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any

    stages {
        stage('Install') {
            steps {
                sh 'go install github.com/maniartech/gocurl/cmd/gocurl@latest'
            }
        }

        stage('Test API') {
            steps {
                sh '''
                    export PATH=$PATH:$(go env GOPATH)/bin
                    gocurl https://api.example.com/health
                '''
            }
        }
    }
}
```

## Testing and Debugging

### Test API Endpoints

Before using an API in code, test with CLI:

```bash
# Test authentication
gocurl -H "Authorization: Bearer $TOKEN" https://api.example.com/me

# Test POST request
gocurl -X POST \\
  -H "Content-Type: application/json" \\
  -d '{"test":"data"}' \\
  https://api.example.com/test

# Test with verbose output
gocurl -v https://api.example.com/debug
```

### Debug SSL Issues

```bash
# Show SSL handshake
gocurl -v https://api.example.com

# Ignore SSL errors (development only!)
gocurl --insecure https://localhost:8443
```

### Test Timeouts

```bash
# Test with short timeout
gocurl --max-time 1 https://httpbin.org/delay/5

# Expected: timeout error after 1 second
```

### Inspect Headers

```bash
# Show only headers
gocurl -I https://api.example.com

# Include headers with body
gocurl -i https://api.example.com
```

## Advanced Workflows

### API Documentation Generation

Generate curl examples for documentation:

```bash
# Save command to file
echo "gocurl -X POST -d '{\"name\":\"Alice\"}' https://api.example.com/users" \\
  > docs/examples/create-user.sh

# Add to markdown
cat >> docs/API.md << 'EOF'
## Create User

\`\`\`bash
gocurl -X POST -d '{"name":"Alice"}' https://api.example.com/users
\`\`\`
EOF
```

### Request/Response Logging

Log all requests and responses:

```bash
#!/bin/bash
# api-logger.sh

LOG_DIR="logs"
mkdir -p "$LOG_DIR"

timestamp=$(date +%Y%m%d_%H%M%S)
request_log="$LOG_DIR/request_$timestamp.log"
response_log="$LOG_DIR/response_$timestamp.log"

# Log request
echo "gocurl $@" > "$request_log"

# Make request and log response
gocurl "$@" | tee "$response_log"
```

### Environment Switcher

Switch between dev/staging/prod:

```bash
#!/bin/bash
# api-switch.sh

ENV=${1:-dev}

case $ENV in
  dev)
    export API_URL="https://api.dev.example.com"
    export API_KEY="dev-key"
    ;;
  staging)
    export API_URL="https://api.staging.example.com"
    export API_KEY="staging-key"
    ;;
  prod)
    export API_URL="https://api.prod.example.com"
    export API_KEY="prod-key"
    ;;
  *)
    echo "Unknown environment: $ENV"
    exit 1
    ;;
esac

echo "Using $ENV environment: $API_URL"

# Make request
gocurl -H "X-API-Key: $API_KEY" ${API_URL}/health
```

## CLI to Code Workflow

One of the most powerful features is the seamless CLI-to-code workflow:

### Step 1: Test in CLI

```bash
gocurl -X POST \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer secret" \\
  -d '{"name":"Alice"}' \\
  https://api.example.com/users
```

### Step 2: Convert to Go Code

Once the curl command works, convert to Go:

```go
package main

import (
    "context"
    "github.com/maniartech/gocurl"
)

func main() {
    ctx := context.Background()

    var user User
    resp, err := gocurl.CurlJSON(ctx, &user,
        `curl -X POST`,
        `-H "Content-Type: application/json"`,
        `-H "Authorization: Bearer secret"`,
        `-d '{"name":"Alice"}'`,
        `https://api.example.com/users`)

    // Use result...
}
```

### Step 3: Refactor

Refactor to production code:

```go
func createUser(ctx context.Context, name string) (*User, error) {
    token := os.Getenv("API_TOKEN")

    var user User
    resp, err := gocurl.CurlJSON(ctx, &user,
        `curl -X POST`,
        `-H "Content-Type: application/json"`,
        fmt.Sprintf(`-H "Authorization: Bearer %s"`, token),
        fmt.Sprintf(`-d '{"name":"%s"}'`, name),
        `https://api.example.com/users`)

    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    return &user, nil
}
```

## Exit Codes

GoCurl follows curl's exit code conventions:

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Generic error |
| 3 | URL malformed |
| 7 | Failed to connect to host |
| 28 | Operation timeout |
| 52 | Server returned nothing |
| 56 | Error receiving network data |

Use in scripts:

```bash
gocurl https://api.example.com/health
if [ $? -eq 0 ]; then
  echo "Success"
else
  echo "Failed with code: $?"
  exit 1
fi
```

## Best Practices

### 1. Use Environment Variables for Secrets

❌ **Bad:**
```bash
gocurl -H "Authorization: Bearer secret-token-123" https://api.example.com
```

✅ **Good:**
```bash
export API_TOKEN="secret-token-123"
gocurl -H "Authorization: Bearer $API_TOKEN" https://api.example.com
```

### 2. Add Timeouts

❌ **Bad:**
```bash
gocurl https://slow-api.example.com
```

✅ **Good:**
```bash
gocurl --max-time 30 https://slow-api.example.com
```

### 3. Use Verbose Mode for Debugging

❌ **Bad:**
```bash
gocurl https://api.example.com  # Silent failure
```

✅ **Good:**
```bash
gocurl -v https://api.example.com  # See what's happening
```

### 4. Save to File in Production Scripts

❌ **Bad:**
```bash
response=$(gocurl https://api.example.com/large-data)
```

✅ **Good:**
```bash
gocurl -o response.json https://api.example.com/large-data
```

### 5. Check Exit Codes

❌ **Bad:**
```bash
gocurl https://api.example.com
echo "Done"  # Runs even if failed
```

✅ **Good:**
```bash
if gocurl https://api.example.com; then
  echo "Success"
else
  echo "Failed"
  exit 1
fi
```

## Common Pitfalls

### Pitfall 1: Shell Quoting Issues

```bash
# ❌ Wrong - variable not expanded
gocurl -H 'Authorization: Bearer $TOKEN' https://api.example.com

# ✅ Correct - use double quotes
gocurl -H "Authorization: Bearer $TOKEN" https://api.example.com
```

### Pitfall 2: JSON Escaping

```bash
# ❌ Wrong - shell interprets special characters
gocurl -d {"name":"Alice"} https://api.example.com

# ✅ Correct - quote the JSON
gocurl -d '{"name":"Alice"}' https://api.example.com
```

### Pitfall 3: Forgetting Content-Type

```bash
# ❌ Wrong - no content type
gocurl -X POST -d '{"name":"Alice"}' https://api.example.com

# ✅ Correct - specify content type
gocurl -X POST \\
  -H "Content-Type: application/json" \\
  -d '{"name":"Alice"}' \\
  https://api.example.com
```

## Summary

The GoCurl CLI provides:

✅ **Familiar Interface**: Drop-in curl replacement
✅ **Enhanced Features**: Verbose output, custom formatting
✅ **Environment Variables**: Automatic expansion
✅ **Shell Integration**: Perfect for scripts and automation
✅ **CI/CD Ready**: Easy integration into pipelines
✅ **Debugging Tools**: Comprehensive error reporting
✅ **CLI-to-Code**: Seamless transition to library usage

## What's Next

In the next chapter, we'll explore real-world integrations with popular APIs, building complete applications that leverage both the CLI and library features.

### Key Takeaways

1. **Install once, use everywhere**: `go install` puts gocurl in your path
2. **Test before coding**: Validate API calls in CLI first
3. **Environment variables**: Keep secrets out of commands
4. **Verbose mode**: Your best friend for debugging
5. **Exit codes**: Enable robust script error handling
6. **Seamless transition**: CLI commands become library calls

The CLI is your gateway to productive API development—use it to experiment, test, and validate before committing code.
