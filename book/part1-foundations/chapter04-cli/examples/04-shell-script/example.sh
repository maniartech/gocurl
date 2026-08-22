#!/bin/bash
# Example 4: Shell Script Integration
# Demonstrates using gocurl in production shell scripts

echo "📜 Example 4: Shell Script Integration"
echo "======================================="
echo ""

# Example 1: Health check script
echo "1️⃣  Health Check Script"
cat > health-check.sh << 'EOF'
#!/bin/bash
API_URL="${1:-https://httpbin.org/status/200}"

echo "Checking health of $API_URL..."
response=$(gocurl -s -o /dev/null -w "%{http_code}" "$API_URL")
exit_code=$?

if [ $exit_code -eq 0 ] && [ "$response" == "200" ]; then
    echo "✅ Healthy (HTTP $response)"
    exit 0
else
    echo "❌ Unhealthy (HTTP $response, exit code: $exit_code)"
    exit 1
fi
EOF

chmod +x health-check.sh
echo "Running: ./health-check.sh"
./health-check.sh
echo ""

# Example 2: API test script
echo "2️⃣  API Test Script"
cat > api-test.sh << 'EOF'
#!/bin/bash
set -e  # Exit on error

API_URL="https://httpbin.org"
TESTS_PASSED=0
TESTS_FAILED=0

test_endpoint() {
    local method=$1
    local endpoint=$2
    local expected_status=$3

    echo "Testing $method $endpoint..."
    status=$(gocurl -s -o /dev/null -w "%{http_code}" -X "$method" "${API_URL}${endpoint}")

    if [ "$status" == "$expected_status" ]; then
        echo "  ✅ Pass (got $status)"
        ((TESTS_PASSED++))
    else
        echo "  ❌ Fail (expected $expected_status, got $status)"
        ((TESTS_FAILED++))
    fi
}

test_endpoint "GET" "/get" "200"
test_endpoint "POST" "/post" "200"
test_endpoint "PUT" "/put" "200"
test_endpoint "DELETE" "/delete" "200"

echo ""
echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed"
[ $TESTS_FAILED -eq 0 ] && echo "✅ All tests passed!"
EOF

chmod +x api-test.sh
echo "Running: ./api-test.sh"
./api-test.sh
echo ""

# Example 3: Retry logic script
echo "3️⃣  Retry Logic Script"
cat > retry-request.sh << 'EOF'
#!/bin/bash
URL="${1:-https://httpbin.org/status/500}"
MAX_RETRIES=3
RETRY_DELAY=1

for attempt in $(seq 1 $MAX_RETRIES); do
    echo "Attempt $attempt/$MAX_RETRIES..."

    if gocurl -s -o /dev/null "$URL"; then
        echo "✅ Success on attempt $attempt"
        exit 0
    else
        echo "❌ Failed on attempt $attempt"
        if [ $attempt -lt $MAX_RETRIES ]; then
            echo "Waiting ${RETRY_DELAY}s before retry..."
            sleep $RETRY_DELAY
        fi
    fi
done

echo "❌ All $MAX_RETRIES attempts failed"
exit 1
EOF

chmod +x retry-request.sh
echo "Running: ./retry-request.sh (testing with 500 error)"
./retry-request.sh || echo "Expected failure demonstrated"
echo ""

# Example 4: Data extraction script
echo "4️⃣  Data Extraction Script"
cat > extract-data.sh << 'EOF'
#!/bin/bash
URL="https://api.github.com/users/golang"

echo "Fetching user data from GitHub..."
response=$(gocurl -s "$URL")

# Extract fields using grep/sed (simple parsing)
login=$(echo "$response" | grep -o '"login":"[^"]*"' | cut -d'"' -f4)
name=$(echo "$response" | grep -o '"name":"[^"]*"' | cut -d'"' -f4)
location=$(echo "$response" | grep -o '"location":"[^"]*"' | cut -d'"' -f4)

echo "Login: $login"
echo "Name: $name"
echo "Location: $location"
EOF

chmod +x extract-data.sh
echo "Running: ./extract-data.sh"
./extract-data.sh
echo ""

# Example 5: Parallel requests
echo "5️⃣  Parallel Requests Script"
cat > parallel-check.sh << 'EOF'
#!/bin/bash
URLS=(
    "https://httpbin.org/delay/1"
    "https://httpbin.org/delay/1"
    "https://httpbin.org/delay/1"
)

echo "Checking ${#URLS[@]} URLs in parallel..."
start_time=$(date +%s)

for url in "${URLS[@]}"; do
    (
        gocurl -s -o /dev/null "$url" && echo "✅ $url"
    ) &
done

wait  # Wait for all background jobs
end_time=$(date +%s)
duration=$((end_time - start_time))

echo "All checks complete in ${duration}s"
EOF

chmod +x parallel-check.sh
echo "Running: ./parallel-check.sh"
./parallel-check.sh
echo ""

# Cleanup
rm -f health-check.sh api-test.sh retry-request.sh extract-data.sh parallel-check.sh

echo "✅ Shell script integration examples complete!"
echo ""
echo "💡 Key Patterns:"
echo "   • Check exit codes: gocurl ... || handle_error"
echo "   • Extract status: -w '%{http_code}'"
echo "   • Silent mode in scripts: -s flag"
echo "   • Use variables for configuration"
echo "   • Implement retry logic for resilience"
