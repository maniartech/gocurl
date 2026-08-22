// Exercise 1: Function Category Selection
// Learn to choose the right gocurl function for different scenarios.
//
// OBJECTIVES:
// - Understand when to use each function category
// - Practice selecting appropriate APIs
// - Handle different response types correctly
//
// REQUIREMENTS:
// Implement the TODO functions below to make the correct API calls.
// Each function should use the most appropriate gocurl function category.

package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/maniartech/gocurl"
)

// TODO 1: Implement checkServerHealth
// Use the appropriate function to check if the server is responding.
// Requirement: Only need to check HTTP status, don't need body.
// Return: true if server responds with 200, false otherwise.
func checkServerHealth(url string) bool {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // TODO: Use the appropriate gocurl function
    // Hint: We only need headers, not the full response body

    return false
}

// TODO 2: Implement fetchTextContent
// Fetch a text response and return it as a string.
// Requirement: Return the body as a string.
func fetchTextContent(url string) (string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // TODO: Use the appropriate gocurl function
    // Hint: We want the response body as a string

    return "", nil
}

// TODO 3: Implement downloadBinaryData
// Download binary data and return as bytes.
// Requirement: Handle binary content (like images).
func downloadBinaryData(url string) ([]byte, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // TODO: Use the appropriate gocurl function
    // Hint: Binary data should be []byte

    return nil, nil
}

// TODO 4: Implement fetchUserData
// Fetch JSON data and unmarshal into GitHubUser struct.
// Requirement: Automatic JSON parsing.
type GitHubUser struct {
    Login    string `json:"login"`
    Name     string `json:"name"`
    Company  string `json:"company"`
    Location string `json:"location"`
}

func fetchUserData(username string) (*GitHubUser, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    url := fmt.Sprintf("https://api.github.com/users/%s", username)

    // TODO: Use the appropriate gocurl function
    // Hint: We want automatic JSON unmarshaling

    return nil, nil
}

// TODO 5: Implement downloadFileToDiscover
// Download a large file directly to disk.
// Requirement: Stream to file without loading full content in memory.
func downloadFileToPath(url, filepath string) (int64, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // TODO: Use the appropriate gocurl function
    // Hint: We want to write directly to a file

    return 0, nil
}

// TODO 6: Implement fetchWithCustomVariables
// Make a request using explicit variable substitution (not environment vars).
// Requirement: Use a variable map, not environment variables.
func fetchWithCustomVariables() (string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Define variables
    vars := map[string]string{
        "endpoint": "/get",
        "param":    "value123",
    }

    // TODO: Use the appropriate gocurl WithVars function
    // Hint: We need to use explicit variable map

    return "", nil
}

// Main function to test your implementations
func main() {
    fmt.Println("🧪 Exercise 1: Function Category Selection\n")

    // Test 1: Health check
    fmt.Println("Test 1: Server Health Check")
    healthy := checkServerHealth("https://httpbin.org/status/200")
    if healthy {
        fmt.Println("   ✅ Server is healthy")
    } else {
        fmt.Println("   ❌ Server check failed")
    }

    // Test 2: Text content
    fmt.Println("\nTest 2: Fetch Text Content")
    text, err := fetchTextContent("https://httpbin.org/uuid")
    if err != nil {
        fmt.Printf("   ❌ Error: %v\n", err)
    } else {
        fmt.Printf("   ✅ Fetched: %s\n", text[:50]) // Print first 50 chars
    }

    // Test 3: Binary data
    fmt.Println("\nTest 3: Download Binary Data")
    data, err := downloadBinaryData("https://httpbin.org/bytes/100")
    if err != nil {
        fmt.Printf("   ❌ Error: %v\n", err)
    } else {
        fmt.Printf("   ✅ Downloaded %d bytes\n", len(data))
    }

    // Test 4: JSON data
    fmt.Println("\nTest 4: Fetch User Data (JSON)")
    user, err := fetchUserData("golang")
    if err != nil {
        fmt.Printf("   ❌ Error: %v\n", err)
    } else {
        fmt.Printf("   ✅ User: %s (%s)\n", user.Name, user.Location)
    }

    // Test 5: File download
    fmt.Println("\nTest 5: Download File")
    bytes, err := downloadFileToPath("https://httpbin.org/bytes/1024", "/tmp/test-download.bin")
    if err != nil {
        fmt.Printf("   ❌ Error: %v\n", err)
    } else {
        fmt.Printf("   ✅ Downloaded %d bytes to /tmp/test-download.bin\n", bytes)
    }

    // Test 6: Custom variables
    fmt.Println("\nTest 6: Fetch with Custom Variables")
    result, err := fetchWithCustomVariables()
    if err != nil {
        fmt.Printf("   ❌ Error: %v\n", err)
    } else {
        fmt.Printf("   ✅ Result: %s\n", result[:50])
    }

    fmt.Println("\n" + "=".Repeat(60))
    fmt.Println("Exercise complete! Review your implementations.")
    fmt.Println("=".Repeat(60))
}

// SELF-CHECK CRITERIA:
//
// ✅ checkServerHealth uses Curl() with -I flag (HEAD request)
// ✅ fetchTextContent uses CurlString() or CurlStringCommand()
// ✅ downloadBinaryData uses CurlBytes() or CurlBytesCommand()
// ✅ fetchUserData uses CurlJSON() with &user parameter
// ✅ downloadFileToPath uses CurlDownload() with file path
// ✅ fetchWithCustomVariables uses CurlCommandWithVars() or similar
//
// EXPECTED OUTPUT:
//
// Test 1: ✅ Server is healthy
// Test 2: ✅ Fetched: {...}
// Test 3: ✅ Downloaded 100 bytes
// Test 4: ✅ User: Go (...)
// Test 5: ✅ Downloaded 1024 bytes to /tmp/test-download.bin
// Test 6: ✅ Result: {...}
//
// BONUS CHALLENGES:
//
// 1. Add proper error messages for each test
// 2. Implement retry logic for flaky network calls
// 3. Add timeout handling for slow responses
// 4. Validate response status codes
// 5. Add progress reporting for downloads
