# Exercise 2: Shell Scripting with GoCurl

**Difficulty:** Intermediate
**Duration:** 45-60 minutes
**Prerequisites:** Basic shell scripting, Exercise 1 completed

## Objective

Build production-ready shell scripts that integrate gocurl for API testing, health checking, and automation. Learn error handling, retry logic, and data extraction patterns.

## Tasks

### Task 1: API Health Check Script

Create `health-check.sh` that checks if an API endpoint is healthy.

**Requirements:**
- Accept URL as command-line argument (default: `https://httpbin.org/status/200`)
- Check HTTP status code
- Exit with 0 if status is 200, else exit 1
- Print clear success/failure message

**Starter Code:**
```bash
#!/bin/bash
# TODO: Complete this script

API_URL="${1:-https://httpbin.org/status/200}"

echo "Checking health of $API_URL..."

# TODO: Make request and capture status code

# TODO: Check if status is 200

# TODO: Exit with appropriate code
```

**Expected Behavior:**
```bash
$ ./health-check.sh https://httpbin.org/status/200
Checking health of https://httpbin.org/status/200...
✅ Healthy (HTTP 200)
$ echo $?
0

$ ./health-check.sh https://httpbin.org/status/500
Checking health of https://httpbin.org/status/500...
❌ Unhealthy (HTTP 500)
$ echo $?
1
```

<details>
<summary>💡 Hint</summary>

Use `gocurl -s -o /dev/null -w "%{http_code}" $API_URL` to get only the status code
</details>

---

### Task 2: Retry Logic Script

Create `retry-request.sh` that retries failed requests with exponential backoff.

**Requirements:**
- Accept URL as argument
- Retry up to 3 times
- Wait 1s, 2s, 4s between retries (exponential backoff)
- Exit on success, continue on failure
- Print attempt number and result

**Starter Code:**
```bash
#!/bin/bash
URL="${1:-https://httpbin.org/status/500}"
MAX_RETRIES=3

# TODO: Implement retry loop with exponential backoff

echo "All attempts failed"
exit 1
```

**Expected Behavior:**
```bash
$ ./retry-request.sh https://httpbin.org/status/500
Attempt 1/3... ❌ Failed
Waiting 1s...
Attempt 2/3... ❌ Failed
Waiting 2s...
Attempt 3/3... ❌ Failed
All attempts failed
```

<details>
<summary>💡 Hint</summary>

Use a for loop: `for attempt in $(seq 1 $MAX_RETRIES); do ... done`
Calculate delay: `delay=$((2 ** (attempt - 1)))`
</details>

---

### Task 3: API Test Suite

Create `api-tests.sh` that tests multiple endpoints and generates a report.

**Requirements:**
- Test at least 5 different endpoints
- Check both status codes and response content
- Count passed/failed tests
- Generate summary report
- Use functions for reusability

**Starter Code:**
```bash
#!/bin/bash
set -e

API_URL="https://httpbin.org"
TESTS_PASSED=0
TESTS_FAILED=0

# TODO: Create test_endpoint function
test_endpoint() {
    local method=$1
    local endpoint=$2
    local expected_status=$3
    local check_content=$4  # Optional: string to check in response

    # TODO: Implement test logic
}

# TODO: Add test cases
test_endpoint "GET" "/get" "200"
test_endpoint "POST" "/post" "200"
# Add more tests...

# TODO: Print summary
echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed"
```

**Expected Output:**
```
Testing GET /get...
  ✅ Pass (got 200)
Testing POST /post...
  ✅ Pass (got 200)
Testing PUT /put...
  ✅ Pass (got 200)
Testing DELETE /delete...
  ✅ Pass (got 200)
Testing PATCH /patch...
  ✅ Pass (got 200)

Results: 5 passed, 0 failed
✅ All tests passed!
```

---

### Task 4: Data Extraction Script

Create `extract-user.sh` that fetches GitHub user data and extracts specific fields.

**Requirements:**
- Accept GitHub username as argument
- Fetch user data from GitHub API
- Extract: login, name, location, public_repos
- Print in readable format
- Handle missing/null fields gracefully

**Starter Code:**
```bash
#!/bin/bash
USERNAME="${1:-golang}"
URL="https://api.github.com/users/$USERNAME"

echo "Fetching data for GitHub user: $USERNAME"
echo "========================================="

# TODO: Fetch data
response=$(gocurl -s "$URL")

# TODO: Extract fields (use grep/sed/awk or jq if available)

# TODO: Print formatted output
```

**Expected Output:**
```bash
$ ./extract-user.sh golang
Fetching data for GitHub user: golang
=========================================
Login:        golang
Name:         Go
Location:     null
Public Repos: 0
Bio:          null
```

<details>
<summary>💡 Hint</summary>

Use: `echo "$response" | grep -o '"field":"[^"]*"' | cut -d'"' -f4`
Or install jq: `echo "$response" | jq -r '.field'`
</details>

---

### Task 5: Parallel Request Script

Create `parallel-fetch.sh` that makes multiple requests concurrently.

**Requirements:**
- Accept list of URLs (min 5)
- Make all requests in parallel
- Collect results
- Print summary with timing
- Show which requests succeeded/failed

**Starter Code:**
```bash
#!/bin/bash
URLS=(
    "https://httpbin.org/delay/1"
    "https://httpbin.org/delay/1"
    "https://httpbin.org/delay/1"
    "https://httpbin.org/status/200"
    "https://httpbin.org/status/500"
)

echo "Fetching ${#URLS[@]} URLs in parallel..."
start_time=$(date +%s)

# TODO: Launch parallel requests

wait  # Wait for all background jobs

end_time=$(date +%s)
duration=$((end_time - start_time))

echo "Completed in ${duration}s"
```

**Expected Output:**
```
Fetching 5 URLs in parallel...
✅ https://httpbin.org/delay/1
✅ https://httpbin.org/delay/1
✅ https://httpbin.org/delay/1
✅ https://httpbin.org/status/200
❌ https://httpbin.org/status/500
Completed in 1s
```

<details>
<summary>💡 Hint</summary>

Run commands in background: `(gocurl ... && echo "✅ $url" || echo "❌ $url") &`
</details>

---

### Task 6: API Monitor Script

Create `monitor.sh` that continuously monitors an API endpoint.

**Requirements:**
- Accept URL and check interval (default 5s)
- Run indefinitely until Ctrl+C
- Print status, response time, timestamp
- Alert on status changes
- Log to file (optional)

**Starter Code:**
```bash
#!/bin/bash
URL="${1:-https://httpbin.org/get}"
INTERVAL="${2:-5}"
PREV_STATUS=""

echo "Monitoring $URL every ${INTERVAL}s (Press Ctrl+C to stop)"
echo "Time                Status  Duration  Change"
echo "════════════════════════════════════════════════"

while true; do
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')

    # TODO: Make request and capture status + time

    # TODO: Check for status changes

    # TODO: Print row

    sleep "$INTERVAL"
done
```

**Expected Output:**
```
Monitoring https://httpbin.org/get every 5s (Press Ctrl+C to stop)
Time                Status  Duration  Change
════════════════════════════════════════════════
2024-01-15 10:30:00  200     0.123s
2024-01-15 10:30:05  200     0.118s
2024-01-15 10:30:10  500     0.125s    ⚠️  Status changed!
2024-01-15 10:30:15  500     0.121s
^C
```

---

## Validation

Create `exercise2-validation.sh`:

```bash
#!/bin/bash
echo "Exercise 2 Validation"
echo "====================="
echo ""

# Task 1: Health check
echo "Task 1: Health Check Script"
if [ -f health-check.sh ]; then
    chmod +x health-check.sh
    ./health-check.sh https://httpbin.org/status/200 > /dev/null && \
    echo "✅ Pass" || echo "❌ Fail"
else
    echo "❌ File not found"
fi

# Task 2: Retry logic
echo "Task 2: Retry Logic Script"
if [ -f retry-request.sh ]; then
    chmod +x retry-request.sh
    output=$(./retry-request.sh https://httpbin.org/status/500 2>&1)
    echo "$output" | grep -q "Attempt" && echo "✅ Pass" || echo "❌ Fail"
else
    echo "❌ File not found"
fi

# Task 3: Test suite
echo "Task 3: API Test Suite"
if [ -f api-tests.sh ]; then
    chmod +x api-tests.sh
    ./api-tests.sh | grep -q "passed" && echo "✅ Pass" || echo "❌ Fail"
else
    echo "❌ File not found"
fi

# Task 4: Data extraction
echo "Task 4: Data Extraction"
if [ -f extract-user.sh ]; then
    chmod +x extract-user.sh
    ./extract-user.sh golang | grep -q "Login" && echo "✅ Pass" || echo "❌ Fail"
else
    echo "❌ File not found"
fi

# Task 5: Parallel requests
echo "Task 5: Parallel Requests"
if [ -f parallel-fetch.sh ]; then
    chmod +x parallel-fetch.sh
    start=$(date +%s)
    ./parallel-fetch.sh > /dev/null
    end=$(date +%s)
    duration=$((end - start))
    [ $duration -lt 3 ] && echo "✅ Pass (completed in ${duration}s)" || echo "❌ Fail (too slow: ${duration}s)"
else
    echo "❌ File not found"
fi

echo ""
echo "Validation complete!"
```

## Bonus Challenges

1. **Rate Limiting:** Create a script that respects rate limits (max 10 req/min)
2. **Circuit Breaker:** Implement circuit breaker pattern (stop after 5 consecutive failures)
3. **Caching:** Cache responses for 60s to avoid redundant requests
4. **Authentication Rotation:** Rotate between multiple API keys on each request

## Learning Outcomes

After completing this exercise, you should be able to:
- ✅ Build production-ready shell scripts with gocurl
- ✅ Implement proper error handling
- ✅ Add retry logic with exponential backoff
- ✅ Create automated test suites
- ✅ Extract and parse API responses
- ✅ Make parallel requests efficiently
- ✅ Monitor APIs continuously

## Next Steps

1. Review your scripts for edge cases
2. Add error handling everywhere
3. Test with various failure scenarios
4. Proceed to Exercise 3 (CI/CD Integration)
