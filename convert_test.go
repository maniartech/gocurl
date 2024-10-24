// File: gocurl/convert_tokens_test.go
package gocurl_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
)

func TestConvertTokensToRequestOptions(t *testing.T) {
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
			name: "Simple GET request",
			tokens: []string{
				"curl",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method:  "GET",
				URL:     "https://api.example.com/data",
				Headers: http.Header{},
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
				Method:  "POST",
				URL:     "https://api.example.com/data",
				Body:    "key=value",
				Headers: http.Header{},
			},
		},
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
		{
			name: "Request with HTTP/2",
			tokens: []string{
				"curl",
				"--http2",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method:  "GET",
				URL:     "https://api.example.com/data",
				HTTP2:   true,
				Headers: http.Header{},
			},
		},
		{
			name: "Request with User-Agent and Referer",
			tokens: []string{
				"curl",
				"-A", "CustomUserAgent/1.0",
				"-e", "https://referrer.example.com",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method:    "GET",
				URL:       "https://api.example.com/data",
				UserAgent: "CustomUserAgent/1.0",
				Referer:   "https://referrer.example.com",
				Headers: http.Header{
					"User-Agent": []string{"CustomUserAgent/1.0"},
					"Referer":    []string{"https://referrer.example.com"},
				},
			},
		},
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
			name: "Request with multiple data fields",
			tokens: []string{
				"curl",
				"-d", "field1=value1",
				"-d", "field2=value2",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method: "POST",
				URL:    "https://api.example.com/data",
				Body:   "field1=value1&field2=value2",
				Headers: http.Header{
					"Content-Type": []string{"application/x-www-form-urlencoded"},
				},
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
		{
			name: "Request with compressed option",
			tokens: []string{
				"curl",
				"--compressed",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method:   "GET",
				URL:      "https://api.example.com/data",
				Compress: true,
				Headers: http.Header{
					"Accept-Encoding": []string{"deflate, gzip"},
				},
			},
		},
		{
			name: "Request with redirect options",
			tokens: []string{
				"curl",
				"-L",
				"--max-redirs", "5",
				"https://api.example.com/data",
			},
			expected: &gocurl.RequestOptions{
				Method:          "GET",
				URL:             "https://api.example.com/data",
				FollowRedirects: true,
				MaxRedirects:    5,
				Headers:         http.Header{},
			},
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
		{
			name: "Request with multiple unknown tokens",
			tokens: []string{
				"curl",
				"https://api.example.com/data",
				"unexpected_token",
			},
			expectError: true,
		},
	}

	for i, tt := range tests {
		var isErr bool
		t.Run(tt.name, func(t *testing.T) {
			options, err := gocurl.ArgsToOptions(tt.tokens)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error at test case %d: %#v", i, err)
				return
			}

			// Normalize Headers for comparison
			normalizeHeaders(tt.expected.Headers)
			normalizeHeaders(options.Headers)

			// Compare options
			if !compareRequestOptions(tt.expected, options, t) {
				fmt.Printf("# %d: %v#v", i, tt)
				t.Errorf("Expected %+v, got %+v", tt.expected, options)
				isErr = true
			}
		})

		if isErr {
			break
		}
	}
}

// Helper function to compare RequestOptions

func compareRequestOptions(expected, actual *gocurl.RequestOptions, t *testing.T) bool {
	equal := true

	if expected.Method != actual.Method {
		t.Errorf("Method mismatch: expected %s, got %s", expected.Method, actual.Method)
		equal = false
	}
	if expected.URL != actual.URL {
		t.Errorf("URL mismatch: expected %s, got %s", expected.URL, actual.URL)
		equal = false
	}
	if expected.Body != actual.Body {
		t.Errorf("Body mismatch: expected %s, got %s", expected.Body, actual.Body)
		equal = false
	}

	// Normalize comparison for nil and empty maps for Headers, QueryParams, Form
	if !compareMaps(expected.Headers, actual.Headers) {
		t.Errorf("Headers mismatch: expected %v, got %v", expected.Headers, actual.Headers)
		equal = false
	}
	if !compareMaps(expected.QueryParams, actual.QueryParams) {
		t.Errorf("QueryParams mismatch: expected %v, got %v", expected.QueryParams, actual.QueryParams)
		equal = false
	}
	if !compareMaps(expected.Form, actual.Form) {
		t.Errorf("Form mismatch: expected %v, got %v", expected.Form, actual.Form)
		equal = false
	}

	if !compareBasicAuth(expected.BasicAuth, actual.BasicAuth) {
		equal = false
	}
	if expected.Proxy != actual.Proxy {
		t.Errorf("Proxy mismatch: expected %s, got %s", expected.Proxy, actual.Proxy)
		equal = false
	}
	if expected.Timeout != actual.Timeout {
		t.Errorf("Timeout mismatch: expected %v, got %v", expected.Timeout, actual.Timeout)
		equal = false
	}
	if expected.ConnectTimeout != actual.ConnectTimeout {
		t.Errorf("ConnectTimeout mismatch: expected %v, got %v", expected.ConnectTimeout, actual.ConnectTimeout)
		equal = false
	}
	if expected.Insecure != actual.Insecure {
		t.Errorf("Insecure mismatch: expected %v, got %v", expected.Insecure, actual.Insecure)
		equal = false
	}
	if expected.FollowRedirects != actual.FollowRedirects {
		t.Errorf("FollowRedirects mismatch: expected %v, got %v", expected.FollowRedirects, actual.FollowRedirects)
		equal = false
	}
	if expected.MaxRedirects != actual.MaxRedirects {
		t.Errorf("MaxRedirects mismatch: expected %d, got %d", expected.MaxRedirects, actual.MaxRedirects)
		equal = false
	}
	if expected.Compress != actual.Compress {
		t.Errorf("Compress mismatch: expected %v, got %v", expected.Compress, actual.Compress)
		equal = false
	}
	if expected.HTTP2 != actual.HTTP2 {
		t.Errorf("HTTP2 mismatch: expected %v, got %v", expected.HTTP2, actual.HTTP2)
		equal = false
	}
	if expected.HTTP2Only != actual.HTTP2Only {
		t.Errorf("HTTP2Only mismatch: expected %v, got %v", expected.HTTP2Only, actual.HTTP2Only)
		equal = false
	}
	if expected.UserAgent != actual.UserAgent {
		t.Errorf("UserAgent mismatch: expected %s, got %s", expected.UserAgent, actual.UserAgent)
		equal = false
	}
	if expected.Referer != actual.Referer {
		t.Errorf("Referer mismatch: expected %s, got %s", expected.Referer, actual.Referer)
		equal = false
	}
	if !compareCookies(expected.Cookies, actual.Cookies) {
		equal = false
	}
	if !compareFileUpload(expected.FileUpload, actual.FileUpload) {
		equal = false
	}

	// Special handling for Context
	if !compareContext(expected.Context, actual.Context) {
		t.Errorf("Context mismatch: expected %v, got %v", expected.Context, actual.Context)
		equal = false
	}

	return equal
}

// Helper function to compare maps (handling nil and empty maps as equivalent)
func compareMaps(expected, actual map[string][]string) bool {
	if len(expected) == 0 && len(actual) == 0 {
		return true
	}
	return reflect.DeepEqual(expected, actual)
}

// Helper function to compare Contexts
func compareContext(expected, actual context.Context) bool {
	// Since context.Background() is equivalent to context.Background(), we just check
	// if both contexts are equivalent.
	return expected == actual || (expected == nil && actual == context.Background())
}

// Helper function to compare BasicAuth
func compareBasicAuth(expected, actual *gocurl.BasicAuth) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected == nil || actual == nil {
		return false
	}
	return expected.Username == actual.Username && expected.Password == actual.Password
}

// Helper function to compare Cookies
func compareCookies(expected, actual []*http.Cookie) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if expected[i].Name != actual[i].Name || expected[i].Value != actual[i].Value {
			return false
		}
	}
	return true
}

// Helper function to compare FileUpload
func compareFileUpload(expected, actual *gocurl.FileUpload) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected == nil || actual == nil {
		return false
	}
	return expected.FieldName == actual.FieldName &&
		expected.FileName == actual.FileName &&
		expected.FilePath == actual.FilePath
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
