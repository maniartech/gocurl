# Chapter 1 Examples

This directory contains runnable code examples from Chapter 1. Each example is in its own directory to prevent naming conflicts.

## Prerequisites

- Go 1.21 or higher
- Internet connection for API calls
- API keys for certain examples (see individual directories)

## Directory Structure

```
examples/
├── README.md (this file)
├── 01-simple-get/          # Simple GET request
├── 02-post-json/           # POST with JSON data
├── 03-json-unmarshal/      # Automatic JSON unmarshaling
├── 04-openai-chat/         # OpenAI API integration
├── 05-stripe-payment/      # Stripe payment processing
├── 06-database-query/      # Database REST API (Supabase)
├── 07-slack-webhook/       # Slack notifications
└── 08-github-viewer/       # Complete GitHub viewer project
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

Each example is a standalone Go module with its own `go.mod` file.

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

## Building All Examples

To verify all examples compile:

```bash
# From the examples directory
for dir in */; do
  echo "Building $dir..."
  (cd "$dir" && go build)
done
```

## Notes

- All examples use production APIs
- Some examples require API keys (see comments in each file)
- Examples demonstrate real-world patterns used in production code
- Error handling is production-ready

## Learn More

See Chapter 1 for detailed explanations of each example.
