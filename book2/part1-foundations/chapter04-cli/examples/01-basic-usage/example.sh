#!/bin/bash
# Example 1: Basic CLI Usage
# Demonstrates fundamental gocurl commands

echo "🚀 Example 1: Basic CLI Usage"
echo "=============================="
echo ""

# Example 1: Simple GET request
echo "1️⃣  Simple GET Request"
echo "Command: gocurl https://httpbin.org/get"
gocurl https://httpbin.org/get
echo ""

# Example 2: GET with query parameters
echo "2️⃣  GET with Query Parameters"
echo "Command: gocurl https://httpbin.org/get?name=Alice&age=30"
gocurl https://httpbin.org/get?name=Alice&age=30
echo ""

# Example 3: POST request with data
echo "3️⃣  POST Request with JSON Data"
echo "Command: gocurl -X POST -H 'Content-Type: application/json' -d '{\"message\":\"Hello\"}' https://httpbin.org/post"
gocurl -X POST -H "Content-Type: application/json" -d '{"message":"Hello"}' https://httpbin.org/post
echo ""

# Example 4: Custom headers
echo "4️⃣  Request with Custom Headers"
echo "Command: gocurl -H 'X-Custom-Header: MyValue' -H 'User-Agent: MyApp/1.0' https://httpbin.org/headers"
gocurl -H "X-Custom-Header: MyValue" -H "User-Agent: MyApp/1.0" https://httpbin.org/headers
echo ""

# Example 5: Save to file
echo "5️⃣  Save Response to File"
echo "Command: gocurl -o response.json https://httpbin.org/json"
gocurl -o response.json https://httpbin.org/json
if [ -f response.json ]; then
    echo "✅ Response saved to response.json"
    echo "First 100 characters:"
    head -c 100 response.json
    echo ""
    rm response.json
fi
echo ""

# Example 6: HEAD request (headers only)
echo "6️⃣  HEAD Request (Headers Only)"
echo "Command: gocurl -I https://httpbin.org/get"
gocurl -I https://httpbin.org/get
echo ""

echo "✅ Basic usage examples complete!"
