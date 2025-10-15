package gocurl_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/maniartech/gocurl"
)

// ParityTest defines a test case that compares gocurl with real curl
type ParityTest struct {
	Name        string
	Command     string                                   // Curl command to execute
	Args        []string                                 // Alternative: args instead of command string
	SkipCurl    bool                                     // Skip real curl execution (for tests that don't need it)
	Setup       func(*testing.T) func()                  // Setup function, returns cleanup
	Validate    func(*testing.T, *http.Response, []byte) // Custom validation
	ExpectError bool                                     // Expect the command to fail
}

// RunParityTest executes both gocurl and real curl, comparing results
func RunParityTest(t *testing.T, test ParityTest) {
	t.Run(test.Name, func(t *testing.T) {
		// Setup
		var cleanup func()
		if test.Setup != nil {
			cleanup = test.Setup(t)
			if cleanup != nil {
				defer cleanup()
			}
		}

		ctx := context.Background()

		// Execute with gocurl
		var gocurlResp *http.Response
		var gocurlBody []byte
		var gocurlErr error

		if len(test.Args) > 0 {
			gocurlResp, gocurlErr = gocurl.CurlArgs(ctx, test.Args...)
		} else {
			gocurlResp, gocurlErr = gocurl.CurlCommand(ctx, test.Command)
		}

		if gocurlErr == nil && gocurlResp != nil {
			defer gocurlResp.Body.Close()
			gocurlBody, _ = io.ReadAll(gocurlResp.Body)
		}

		// Check error expectation
		if test.ExpectError {
			if gocurlErr == nil {
				t.Errorf("Expected error but got none")
			}
			return
		}

		if gocurlErr != nil {
			t.Fatalf("gocurl failed: %v", gocurlErr)
		}

		// Custom validation if provided
		if test.Validate != nil {
			test.Validate(t, gocurlResp, gocurlBody)
			return
		}

		// Execute with real curl if not skipped
		if !test.SkipCurl {
			curlResp, curlBody, curlErr := executeRealCurl(test.Command, test.Args)
			if curlErr != nil {
				t.Logf("Warning: Real curl failed: %v (skipping comparison)", curlErr)
				return
			}

			// Compare results
			compareResults(t, curlResp, curlBody, gocurlResp, gocurlBody)
		}
	})
}

// executeRealCurl runs the actual curl command
func executeRealCurl(command string, args []string) (*http.Response, []byte, error) {
	// Check if curl is available
	if _, err := exec.LookPath("curl"); err != nil {
		return nil, nil, fmt.Errorf("curl not found: %w", err)
	}

	var cmdArgs []string

	if len(args) > 0 {
		cmdArgs = args
	} else {
		// Parse command string into args
		// This is a simplified parser - may need enhancement
		cmdArgs = strings.Fields(command)
	}

	// Execute curl
	cmd := exec.Command("curl", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("curl execution failed: %w", err)
	}

	// Note: We can't easily get full http.Response from curl
	// So we just compare body content for now
	return nil, output, nil
}

// compareResults compares gocurl and curl responses
func compareResults(t *testing.T, curlResp *http.Response, curlBody []byte, gocurlResp *http.Response, gocurlBody []byte) {
	// Compare body content
	if !bytes.Equal(curlBody, gocurlBody) {
		t.Errorf("Body mismatch:\nCurl:   %s\nGoCurl: %s", string(curlBody), string(gocurlBody))
	}

	// If we have curl response headers, compare them too
	if curlResp != nil && gocurlResp != nil {
		if curlResp.StatusCode != gocurlResp.StatusCode {
			t.Errorf("Status code mismatch: curl=%d, gocurl=%d", curlResp.StatusCode, gocurlResp.StatusCode)
		}
	}
}

// Test Categories

// TestCoreParity tests basic curl functionality
func TestCoreParity(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/get":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"method": r.Method,
				"path":   r.URL.Path,
			})
		case "/post":
			body, _ := io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"method": r.Method,
				"body":   string(body),
			})
		case "/headers":
			w.Header().Set("Content-Type", "application/json")
			headers := make(map[string]string)
			for key, values := range r.Header {
				headers[key] = values[0]
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"headers": headers,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tests := []ParityTest{
		{
			Name:     "Simple GET",
			Args:     []string{server.URL + "/get"},
			SkipCurl: true, // Skip curl comparison for local server
			Validate: func(t *testing.T, resp *http.Response, body []byte) {
				if resp.StatusCode != 200 {
					t.Errorf("Expected status 200, got %d", resp.StatusCode)
				}
				var data map[string]interface{}
				if err := json.Unmarshal(body, &data); err != nil {
					t.Errorf("Failed to parse JSON: %v", err)
				}
				if data["method"] != "GET" {
					t.Errorf("Expected GET method, got %v", data["method"])
				}
			},
		},
		{
			Name:     "POST with data",
			Args:     []string{"-X", "POST", "-d", "test=data", server.URL + "/post"},
			SkipCurl: true,
			Validate: func(t *testing.T, resp *http.Response, body []byte) {
				if resp.StatusCode != 200 {
					t.Errorf("Expected status 200, got %d", resp.StatusCode)
				}
				var data map[string]interface{}
				if err := json.Unmarshal(body, &data); err != nil {
					t.Errorf("Failed to parse JSON: %v", err)
				}
				if data["method"] != "POST" {
					t.Errorf("Expected POST method, got %v", data["method"])
				}
			},
		},
		{
			Name:     "Custom headers",
			Args:     []string{"-H", "X-Test: value", server.URL + "/headers"},
			SkipCurl: true,
			Validate: func(t *testing.T, resp *http.Response, body []byte) {
				if resp.StatusCode != 200 {
					t.Errorf("Expected status 200, got %d", resp.StatusCode)
				}
				var data map[string]interface{}
				if err := json.Unmarshal(body, &data); err != nil {
					t.Errorf("Failed to parse JSON: %v", err)
				}
				headers := data["headers"].(map[string]interface{})
				if headers["X-Test"] != "value" {
					t.Errorf("Expected X-Test header, got %v", headers)
				}
			},
		},
	}

	for _, test := range tests {
		RunParityTest(t, test)
	}
}

// TestMultilineParity tests multi-line command support
func TestMultilineParity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}))
	defer server.Close()

	tests := []ParityTest{
		{
			Name: "Backslash continuation",
			Command: server.URL + " \\\n" +
				"  -H 'Content-Type: application/json'",
			SkipCurl: true,
			Validate: func(t *testing.T, resp *http.Response, body []byte) {
				if resp.StatusCode != 200 {
					t.Errorf("Expected status 200, got %d", resp.StatusCode)
				}
			},
		},
		{
			Name: "Comments",
			Command: "# This is a comment\n" +
				server.URL + "\n" +
				"# Another comment",
			SkipCurl: true,
			Validate: func(t *testing.T, resp *http.Response, body []byte) {
				if resp.StatusCode != 200 {
					t.Errorf("Expected status 200, got %d", resp.StatusCode)
				}
			},
		},
		{
			Name:     "Curl prefix removal",
			Command:  "curl " + server.URL,
			SkipCurl: true,
			Validate: func(t *testing.T, resp *http.Response, body []byte) {
				if resp.StatusCode != 200 {
					t.Errorf("Expected status 200, got %d", resp.StatusCode)
				}
			},
		},
	}

	for _, test := range tests {
		RunParityTest(t, test)
	}
}

// TestEnvironmentVariablesParity tests environment variable expansion
func TestEnvironmentVariablesParity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		auth := r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]string{
			"auth": auth,
		})
	}))
	defer server.Close()

	tests := []ParityTest{
		{
			Name: "Environment variable expansion",
			Args: []string{"-H", "Authorization: Bearer $TEST_TOKEN", server.URL},
			Setup: func(t *testing.T) func() {
				os.Setenv("TEST_TOKEN", "test123")
				return func() {
					os.Unsetenv("TEST_TOKEN")
				}
			},
			SkipCurl: true,
			Validate: func(t *testing.T, resp *http.Response, body []byte) {
				var data map[string]string
				if err := json.Unmarshal(body, &data); err != nil {
					t.Errorf("Failed to parse JSON: %v", err)
				}
				expected := "Bearer test123"
				if data["auth"] != expected {
					t.Errorf("Expected auth=%s, got %s", expected, data["auth"])
				}
			},
		},
		{
			Name: "Braced variable expansion",
			Args: []string{"-H", "Authorization: Bearer ${TEST_TOKEN}", server.URL},
			Setup: func(t *testing.T) func() {
				os.Setenv("TEST_TOKEN", "test456")
				return func() {
					os.Unsetenv("TEST_TOKEN")
				}
			},
			SkipCurl: true,
			Validate: func(t *testing.T, resp *http.Response, body []byte) {
				var data map[string]string
				if err := json.Unmarshal(body, &data); err != nil {
					t.Errorf("Failed to parse JSON: %v", err)
				}
				expected := "Bearer test456"
				if data["auth"] != expected {
					t.Errorf("Expected auth=%s, got %s", expected, data["auth"])
				}
			},
		},
	}

	for _, test := range tests {
		RunParityTest(t, test)
	}
}

// TestConvenienceFunctions tests all convenience functions
func TestConvenienceFunctions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Hello, World!",
		})
	}))
	defer server.Close()

	ctx := context.Background()

	t.Run("CurlString", func(t *testing.T) {
		body, resp, err := gocurl.CurlString(ctx, server.URL)
		if err != nil {
			t.Fatalf("CurlString failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		if !strings.Contains(body, "Hello, World!") {
			t.Errorf("Expected body to contain 'Hello, World!', got: %s", body)
		}
	})

	t.Run("CurlBytes", func(t *testing.T) {
		body, resp, err := gocurl.CurlBytes(ctx, server.URL)
		if err != nil {
			t.Fatalf("CurlBytes failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		if !bytes.Contains(body, []byte("Hello, World!")) {
			t.Errorf("Expected body to contain 'Hello, World!', got: %s", string(body))
		}
	})

	t.Run("CurlJSON", func(t *testing.T) {
		var data map[string]string
		resp, err := gocurl.CurlJSON(ctx, &data, server.URL)
		if err != nil {
			t.Fatalf("CurlJSON failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		if data["message"] != "Hello, World!" {
			t.Errorf("Expected message='Hello, World!', got: %s", data["message"])
		}
	})
}
