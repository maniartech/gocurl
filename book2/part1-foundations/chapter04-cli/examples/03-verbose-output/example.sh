#!/bin/bash
# Example 3: Verbose Output
# Demonstrates debugging with verbose mode

echo "🔍 Example 3: Verbose Output"
echo "============================="
echo ""

# Example 1: Basic verbose output
echo "1️⃣  Basic Verbose Output (-v)"
echo "Command: gocurl -v https://httpbin.org/get"
echo "Shows: Request headers, response headers, and body"
echo ""
gocurl -v https://httpbin.org/get
echo ""

# Example 2: Include headers in output
echo "2️⃣  Include Headers in Output (-i)"
echo "Command: gocurl -i https://httpbin.org/get"
echo "Shows: Response headers followed by body"
echo ""
gocurl -i https://httpbin.org/get
echo ""

# Example 3: Silent mode (errors only)
echo "3️⃣  Silent Mode (-s)"
echo "Command: gocurl -s https://httpbin.org/get > /dev/null && echo 'Success'"
gocurl -s https://httpbin.org/get > /dev/null && echo "✅ Success (output suppressed)"
echo ""

# Example 4: Verbose with POST request
echo "4️⃣  Verbose POST Request"
echo "Command: gocurl -v -X POST -d '{\"test\":\"data\"}' https://httpbin.org/post"
echo ""
gocurl -v -X POST -H "Content-Type: application/json" -d '{"test":"data"}' https://httpbin.org/post
echo ""

# Example 5: Debug SSL/TLS
echo "5️⃣  Debug SSL/TLS Connection"
echo "Command: gocurl -v https://api.github.com/zen"
echo "Shows: TLS handshake details"
echo ""
gocurl -v https://api.github.com/zen
echo ""

# Example 6: Verbose with custom output
echo "6️⃣  Verbose with Custom Output Format"
echo "Command: gocurl -v -w 'Status: %{http_code}\\nTime: %{time_total}s\\n' https://httpbin.org/get"
echo ""
gocurl -v -w "Status: %{http_code}\nTime: %{time_total}s\n" https://httpbin.org/get
echo ""

echo "✅ Verbose output examples complete!"
echo ""
echo "💡 When to Use Each:"
echo "   -v (verbose)  : Full debugging, see all details"
echo "   -i (include)  : Need headers with response"
echo "   -s (silent)   : Scripts where only success/failure matters"
