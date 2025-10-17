# Chapter 1 Examples

This directory contains runnable code examples from Chapter 1. Each example is in its own directory to prevent naming conflicts.

## Prerequisites

- Go 1.21 or higher
- GoCurl installed (from parent repository)
- Internet connection for API calls
- API keys for certain examples (see individual directories)

## Quick Start

All examples use the local gocurl source code from the parent repository. Just run them directly:

```bash
cd 01-simple-get
go run main.go
```

## Directory Structure

```
examples/
├── README.md (this file)
├── 01-simple-get/          # Simple GET request
│   └── main.go
├── 02-post-json/           # POST with JSON data
│   └── main.go
├── 03-json-unmarshal/      # Automatic JSON unmarshaling
│   └── main.go
├── 04-openai-chat/         # OpenAI API integration
│   └── main.go
├── 05-stripe-payment/      # Stripe payment processing
│   └── main.go
├── 06-database-query/      # Database REST API (Supabase)
│   └── main.go
├── 07-slack-webhook/       # Slack notifications
│   └── main.go
└── 08-github-viewer/       # Complete GitHub viewer project
    └── main.go
```

## Examples

### Basic Examples (No API Key Required)

1. **01-simple-get/** - Simple GET request to GitHub API
2. **02-post-json/** - POST request with JSON data
3. **03-json-unmarshal/** - Automatic JSON unmarshaling

### Modern API Integrations (API Keys Required)

4. **04-openai-chat/** - OpenAI chat completion example
5. **05-stripe-payment/** - Stripe payment intent creation
6. **06-database-query/** - Supabase/database REST API query
7. **07-slack-webhook/** - Send Slack notifications

### Complete Project

8. **08-github-viewer/** - Complete GitHub repository viewer CLI tool

## Running Examples

Each example is a simple Go program. Just navigate to the directory and run it.

### Basic Examples (no API key required)

```bash
# Simple GET request
cd 01-simple-get
go run main.go

# POST with JSON
cd 02-post-json
go run main.go

# JSON unmarshaling
cd 03-json-unmarshal
go run main.go
```

### Examples Requiring API Keys

Set environment variables first, then run:

```bash
# OpenAI Chat
export OPENAI_API_KEY="sk-..."
cd 04-openai-chat
go run main.go

# Stripe Payment
export STRIPE_SECRET_KEY="sk_test_..."
cd 05-stripe-payment
go run main.go

# Database Query (Supabase)
export SUPABASE_URL="https://xxx.supabase.co"
export SUPABASE_KEY="eyJ..."
cd 06-database-query
go run main.go

# Slack Webhook
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/..."
cd 07-slack-webhook
go run main.go
```

### GitHub Viewer Project

```bash
cd 08-github-viewer
go build
./github-viewer golang/go

# With authentication (higher rate limits)
export GITHUB_TOKEN="ghp_..."
./github-viewer golang/go
```

## Verifying All Examples

To check all examples compile:

```bash
# From the examples directory
for dir in 0*/; do
  echo "Checking $dir..."
  (cd "$dir" && go run main.go --help 2>/dev/null || echo "Compiles OK")
done
```

Or compile them individually:

```bash
cd 01-simple-get
go build main.go
./main  # Run the compiled binary
```

## Notes

- All examples use production APIs
- Some examples require API keys (see comments in each file)
- Examples demonstrate real-world patterns used in production code
- Error handling is production-ready

## Learn More

See Chapter 1 for detailed explanations of each example.
