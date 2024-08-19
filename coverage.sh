#!/bin/bash

# Create .coverage directory if it doesn't exist
mkdir -p .coverage

# Run tests with coverage
go test ./... -coverprofile=.coverage/coverage.out

# Generate coverage report
go tool cover -html=.coverage/coverage.out -o .coverage/coverage.html

# Display coverage percentage
go tool cover -func=.coverage/coverage.out | grep total | awk '{print "Total Coverage: " $3}'

echo "Coverage report generated. Open .coverage/coverage.html to view the report."

# Open coverage report in browser based on OS
case "$OSTYPE" in
  darwin*)  open .coverage/coverage.html ;;  # macOS
  linux*)   xdg-open .coverage/coverage.html ;;  # Linux
  msys*)    start .coverage/coverage.html ;;  # Windows
  *)        echo "Unsupported OS: Please open .coverage/coverage.html manually" ;;
esac