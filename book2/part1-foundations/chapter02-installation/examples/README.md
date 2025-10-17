# Chapter 2: Installation & Setup - Examples

This directory contains working examples demonstrating installation verification and basic setup patterns.

## Examples Overview

1. **01-verify-install** - Basic installation verification
2. **02-first-request** - Your first gocurl request
3. **03-json-request** - JSON API interaction
4. **04-context-timeout** - Using context with timeouts
5. **05-environment-vars** - Environment variable configuration
6. **06-github-client** - Structured API client example

## Prerequisites

- Go 1.21 or later installed
- GoCurl library installed (`go get github.com/maniartech/gocurl`)
- Internet connection for API requests

## Running Examples

Each example is in its own directory with a `main.go` file:

```bash
# Run any example
cd 01-verify-install
go run main.go

# Or from this directory
go run 01-verify-install/main.go
```

## Example Descriptions

### 01-verify-install
**Purpose:** Verify gocurl is installed correctly
**Learns:** Basic imports and first request
**Time:** 2 minutes

### 02-first-request
**Purpose:** Make your first API call
**Learns:** CurlString function, error handling
**Time:** 3 minutes

### 03-json-request
**Purpose:** Parse JSON responses
**Learns:** CurlJSON function, struct unmarshaling
**Time:** 5 minutes

### 04-context-timeout
**Purpose:** Handle timeouts gracefully
**Learns:** Context usage, timeout patterns
**Time:** 5 minutes

### 05-environment-vars
**Purpose:** Use environment variables securely
**Learns:** Variable expansion, secret management
**Time:** 5 minutes

### 06-github-client
**Purpose:** Build a structured API client
**Learns:** Project organization, reusable clients
**Time:** 10 minutes

## Environment Setup

Some examples require environment variables:

```bash
# For GitHub examples
export GITHUB_TOKEN=your_github_token_here

# For general API testing
export API_KEY=your_api_key_here
```

## Troubleshooting

**"package github.com/maniartech/gocurl is not in GOROOT"**
```bash
go mod tidy
```

**"cannot find main module"**
```bash
go mod init example.com/test
go get github.com/maniartech/gocurl
```

**Network errors**
- Check internet connection
- Verify firewall settings
- Try with different API (e.g., httpbin.org)

## Next Steps

After completing these examples:
1. Read Chapter 3: Core Concepts
2. Try the exercises in `../exercises/`
3. Build your own API client
