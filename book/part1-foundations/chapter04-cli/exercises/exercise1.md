# Exercise 1: Basic CLI Commands

**Difficulty:** Beginner
**Duration:** 20-30 minutes
**Prerequisites:** GoCurl CLI installed

## Objective

Master fundamental gocurl commands including different HTTP methods, custom headers, query parameters, and response handling.

## Tasks

### Task 1: Simple GET Request
Make a GET request to `https://httpbin.org/get` and examine the response.

**Requirements:**
- Use gocurl to make the request
- Save output to understand the response structure

**Expected Output:** JSON response showing request details

<details>
<summary>💡 Hint</summary>

```bash
gocurl https://httpbin.org/get
```
</details>

---

### Task 2: GET with Query Parameters
Fetch user information with query parameters: `name=John`, `age=25`, `city=NewYork`

**Requirements:**
- Construct URL with query parameters
- Verify parameters are sent correctly

**Expected Output:** JSON response with `args` field showing your parameters

<details>
<summary>💡 Hint</summary>

Query parameters can be added directly to the URL after `?`
</details>

---

### Task 3: POST JSON Data
Send a POST request to `https://httpbin.org/post` with JSON data:
```json
{
  "username": "testuser",
  "email": "test@example.com",
  "active": true
}
```

**Requirements:**
- Use POST method
- Set Content-Type header to `application/json`
- Send the JSON data

**Expected Output:** Response showing your JSON in the `json` field

<details>
<summary>💡 Hint</summary>

Use `-X POST`, `-H "Content-Type: application/json"`, and `-d` flag
</details>

---

### Task 4: Custom Headers
Make a request to `https://httpbin.org/headers` with these custom headers:
- `X-API-Key: my-secret-key-12345`
- `X-Request-ID: req-001`
- `User-Agent: MyApp/1.0`

**Requirements:**
- Send all three custom headers
- Verify they appear in the response

**Expected Output:** Response showing your custom headers

<details>
<summary>💡 Hint</summary>

Use multiple `-H` flags, one for each header
</details>

---

### Task 5: Save Response to File
Fetch JSON data from `https://httpbin.org/json` and save it to `data.json`

**Requirements:**
- Save response to file using gocurl
- Verify file was created
- Print first 5 lines of the file

**Expected Output:** File `data.json` created with JSON content

<details>
<summary>💡 Hint</summary>

Use the `-o` flag to specify output file
</details>

---

### Task 6: Verbose Mode
Make a request to `https://api.github.com/zen` with verbose output enabled

**Requirements:**
- Enable verbose mode to see request/response headers
- Observe the connection details

**Expected Output:** Detailed output showing connection info, headers, and response

<details>
<summary>💡 Hint</summary>

Use the `-v` flag for verbose output
</details>

---

### Task 7: Environment Variables
Set an environment variable `API_TOKEN=secret123` and use it in a request to `https://httpbin.org/headers`

**Requirements:**
- Set the environment variable
- Use it in an Authorization header: `Authorization: Bearer $API_TOKEN`
- Verify the token appears in the response

**Expected Output:** Response showing your token in Authorization header

<details>
<summary>💡 Hint</summary>

```bash
export API_TOKEN=secret123
gocurl -H "Authorization: Bearer $API_TOKEN" https://httpbin.org/headers
```
</details>

---

### Task 8: HTTP Status Code Only
Get only the HTTP status code for `https://httpbin.org/status/404` without the response body

**Requirements:**
- Request the URL
- Output only the status code (404)
- Suppress response body

**Expected Output:** Just the number `404`

<details>
<summary>💡 Hint</summary>

Use `-w "%{http_code}\n"`, `-o /dev/null`, and `-s` flags
</details>

---

## Validation

Create a script `exercise1-validation.sh` that runs all 8 tasks:

```bash
#!/bin/bash
echo "Exercise 1 Validation"
echo "====================="

# Task 1
echo "Task 1: Simple GET"
gocurl https://httpbin.org/get > /dev/null && echo "✅ Pass" || echo "❌ Fail"

# Task 2
echo "Task 2: GET with query params"
gocurl "https://httpbin.org/get?name=John&age=25&city=NewYork" | grep -q "John" && echo "✅ Pass" || echo "❌ Fail"

# Task 3
echo "Task 3: POST JSON"
gocurl -X POST -H "Content-Type: application/json" -d '{"username":"testuser","email":"test@example.com","active":true}' https://httpbin.org/post | grep -q "testuser" && echo "✅ Pass" || echo "❌ Fail"

# Task 4
echo "Task 4: Custom headers"
gocurl -H "X-API-Key: my-secret-key-12345" -H "X-Request-ID: req-001" -H "User-Agent: MyApp/1.0" https://httpbin.org/headers | grep -q "X-Api-Key" && echo "✅ Pass" || echo "❌ Fail"

# Task 5
echo "Task 5: Save to file"
gocurl -o data.json https://httpbin.org/json && [ -f data.json ] && echo "✅ Pass" || echo "❌ Fail"
rm -f data.json

# Task 6
echo "Task 6: Verbose mode"
gocurl -v https://api.github.com/zen 2>&1 | grep -q "HTTP" && echo "✅ Pass" || echo "❌ Fail"

# Task 7
echo "Task 7: Environment variables"
export API_TOKEN=secret123
gocurl -H "Authorization: Bearer $API_TOKEN" https://httpbin.org/headers | grep -q "secret123" && echo "✅ Pass" || echo "❌ Fail"

# Task 8
echo "Task 8: Status code only"
status=$(gocurl -w "%{http_code}\n" -o /dev/null -s https://httpbin.org/status/404)
[ "$status" == "404" ] && echo "✅ Pass" || echo "❌ Fail"

echo ""
echo "Validation complete!"
```

Run: `bash exercise1-validation.sh`

## Bonus Challenges

1. **Timeout Challenge:** Make a request that times out after 2 seconds to `https://httpbin.org/delay/5`
2. **Redirect Challenge:** Make a request to `https://httpbin.org/redirect/3` and follow redirects
3. **Authentication Challenge:** Make a request with Basic Auth to `https://httpbin.org/basic-auth/user/pass`

## Learning Outcomes

After completing this exercise, you should be able to:
- ✅ Execute basic HTTP requests (GET, POST, PUT, DELETE)
- ✅ Work with query parameters and request data
- ✅ Add custom headers to requests
- ✅ Use environment variables in commands
- ✅ Control output (verbose, silent, formatted)
- ✅ Save responses to files
- ✅ Extract specific information (status codes)

## Next Steps

Once you've completed this exercise:
1. Review your solutions
2. Run the validation script
3. Try the bonus challenges
4. Proceed to Exercise 2 (Shell Scripting)
