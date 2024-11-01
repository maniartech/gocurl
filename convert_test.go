// File: gocurl/convert_tokens_test.go
package gocurl_test

import (
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
	"github.com/stretchr/testify/assert"
)

func TestBasicRequests(t *testing.T) {
	tests := []struct {
		name        string
		tokens      []string
		expected    *gocurl.RequestOptions
		expectError bool
	}{
		{
			name: "Simple GET request",
			tokens: []string{
				"curl",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method:      "GET",
				URL:         "https://api.example.com/data",
				QueryParams: url.Values{},
			},
		},
		{
			name: "POST request with data",
			tokens: []string{
				"curl",
				"-X", "POST",
				"-d", "key=value",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method:      "POST",
				URL:         "https://api.example.com/data",
				Body:        "key=value",
				QueryParams: url.Values{},
			},
		},
	}
	runTests(t, tests)
}

func TestHeadersAndAuth(t *testing.T) {
	tests := []struct {
		name        string
		tokens      []string
		expected    *gocurl.RequestOptions
		expectError bool
	}{
		{
			name: "Request with headers",
			tokens: []string{
				"curl",
				"-H", "Content-Type: application/json",
				"-H", "Authorization: Bearer dummy_token",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method: "GET",
				URL:    "https://api.example.com/data",
				Headers: http.Header{
					"Content-Type":  []string{"application/json"},
					"Authorization": []string{"Bearer dummy_token"},
				},
			},
		},
		{
			name: "Request with basic auth",
			tokens: []string{
				"curl",
				"-u", "username:password",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method: "GET",
				URL:    "https://api.example.com/data",
				BasicAuth: &gocurl.BasicAuth{
					Username: "username",
					Password: "password",
				},
				Headers: http.Header{},
			},
		},
	}
	runTests(t, tests)
}

func TestFormAndFileUploads(t *testing.T) {
	tests := []struct {
		name        string
		tokens      []string
		expected    *gocurl.RequestOptions
		expectError bool
	}{
		{
			name: "Request with form data",
			tokens: []string{
				"curl",
				"-F", "field1=value1",
				"-F", "field2=value2",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method: "POST",
				URL:    "https://api.example.com/data",
				Form: url.Values{
					"field1": []string{"value1"},
					"field2": []string{"value2"},
				},
				Headers: http.Header{},
			},
		},
		{
			name: "Request with file upload",
			tokens: []string{
				"curl",
				"-F", "file=@/path/to/file.txt",
				"https://api.example.com/upload",
			},
			expected: &gocurl.RequestOptions{
				Method: "POST",
				URL:    "https://api.example.com/upload",
				FileUpload: &gocurl.FileUpload{
					FieldName: "file",
					FileName:  "file.txt",
					FilePath:  "/path/to/file.txt",
				},
				Headers: http.Header{},
			},
		},
	}
	runTests(t, tests)
}

func TestCookiesAndProxy(t *testing.T) {
	tests := []struct {
		name        string
		tokens      []string
		expected    *gocurl.RequestOptions
		expectError bool
	}{
		{
			name: "Request with cookies",
			tokens: []string{
				"curl",
				"-b", "session=abcd1234; theme=light",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method: "GET",
				URL:    "https://api.example.com/data",
				Cookies: []*http.Cookie{
					{Name: "session", Value: "abcd1234"},
					{Name: "theme", Value: "light"},
				},
				Headers: http.Header{},
			},
		},
		{
			name: "Request with proxy",
			tokens: []string{
				"curl",
				"-x", "http://proxy.example.com:8080",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method:  "GET",
				URL:     "https://api.example.com/data",
				Proxy:   "http://proxy.example.com:8080",
				Headers: http.Header{},
			},
		},
	}
	runTests(t, tests)
}

func TestTimeoutAndSSL(t *testing.T) {
	tests := []struct {
		name        string
		tokens      []string
		expected    *gocurl.RequestOptions
		expectError bool
	}{
		{
			name: "Request with timeout",
			tokens: []string{
				"curl",
				"--max-time", "30",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method:  "GET",
				URL:     "https://api.example.com/data",
				Timeout: 30 * time.Second,
				Headers: http.Header{},
			},
		},
		{
			name: "Request with SSL options",
			tokens: []string{
				"curl",
				"--cert", "client.crt",
				"--key", "client.key",
				"--cacert", "ca.crt",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method:   "GET",
				URL:      "https://api.example.com/data",
				CertFile: "client.crt",
				KeyFile:  "client.key",
				CAFile:   "ca.crt",
				Headers:  http.Header{},
			},
		},
	}
	runTests(t, tests)
}

func TestErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		tokens      []string
		expected    *gocurl.RequestOptions
		expectError bool
	}{
		{
			name: "Request with unknown flag",
			tokens: []string{
				"curl",
				"--unknown",
				"https://api.example.com/data",
			},
			expectError: true,
		},
		{
			name: "Request with missing argument",
			tokens: []string{
				"curl",
				"-X",
				"https://api.example.com/data",
			},
			expectError: true,
		},
		{
			name: "Request with invalid URL",
			tokens: []string{
				"curl",
				"ht!tp://invalid-url",
			},
			expectError: true,
		},
		{
			name: "Request with invalid max redirects",
			tokens: []string{
				"curl",
				"--max-redirs", "invalid",
				"https://api.example.com/data",
			},
			expectError: true,
		},
	}
	runTests(t, tests)
}

func TestEnvironmentVariables(t *testing.T) {
	// Set up environment variables for testing
	os.Setenv("API_URL", "https://api.example.com/data")
	os.Setenv("TOKEN", "dummy_token")

	tests := []struct {
		name        string
		tokens      []string
		expected    *gocurl.RequestOptions
		expectError bool
	}{
		{
			name: "Request with environment variables",
			tokens: []string{
				"curl",
				"$API_URL",
			},
			expected: &gocurl.RequestOptions{
				Method:  "GET",
				URL:     "https://api.example.com/data",
				Headers: http.Header{},
			},
		},
		{
			name: "Request with environment variable in data",
			tokens: []string{
				"curl",
				"-d", "token=$TOKEN",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method: "POST",
				URL:    "https://api.example.com/data",
				Body:   "token=dummy_token",
				Headers: http.Header{
					"Content-Type": []string{"application/x-www-form-urlencoded"},
				},
			},
		},
	}
	runTests(t, tests)
}

// Helper function to run the tests
func runTests(t *testing.T, tests []struct {
	name        string
	tokens      []string
	expected    *gocurl.RequestOptions
	expectError bool
}) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, err := gocurl.ArgsToOptions(tt.tokens)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			compareRequestOptions(tt.expected, options, t)
		})
	}
}

func compareRequestOptions(expected, actual *gocurl.RequestOptions, t *testing.T) bool {

	normalizeHeaders(expected.Headers)
	normalizeHeaders(actual.Headers)

	// Convert both expected and actual RequestOptions to json strings for comparison
	expectedJSON, err := expected.ToJSON()
	if err != nil {
		t.Errorf("Error converting expected RequestOptions to JSON: %v", err)
		return false
	}

	actualJSON, err := actual.ToJSON()
	if err != nil {
		t.Errorf("Error converting actual RequestOptions to JSON: %v", err)
		return false
	}

	assert.JSONEq(t, expectedJSON, actualJSON)

	if expectedJSON != actualJSON {
		t.Errorf("RequestOptions mismatch: expected %s, got %s", expectedJSON, actualJSON)
		return false
	}

	return true
}

// Helper function to normalize headers (convert to lowercase keys)
func normalizeHeaders(headers http.Header) {
	for key, values := range headers {
		lowerKey := http.CanonicalHeaderKey(key)
		if lowerKey != key {
			headers[lowerKey] = values
			delete(headers, key)
		}
	}
}
