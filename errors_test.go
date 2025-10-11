package gocurl_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/maniartech/gocurl"
)

func TestGocurlError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *gocurl.GocurlError
		contains []string
	}{
		{
			name: "Parse error",
			err: &gocurl.GocurlError{
				Op:      "parse",
				Command: "curl https://example.com",
				Err:     errors.New("invalid flag"),
			},
			contains: []string{"parse", "curl https://example.com", "invalid flag"},
		},
		{
			name: "Request error with URL",
			err: &gocurl.GocurlError{
				Op:  "request",
				URL: "https://example.com/api",
				Err: errors.New("connection refused"),
			},
			contains: []string{"request", "https://example.com/api", "connection refused"},
		},
		{
			name: "Long command truncation",
			err: &gocurl.GocurlError{
				Op:      "parse",
				Command: strings.Repeat("a", 150),
				Err:     errors.New("too long"),
			},
			contains: []string{"parse", "...", "too long"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, expected := range tt.contains {
				if !strings.Contains(errStr, expected) {
					t.Errorf("Error string %q does not contain %q", errStr, expected)
				}
			}
		})
	}
}

func TestGocurlError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	err := &gocurl.GocurlError{
		Op:  "test",
		Err: innerErr,
	}

	if !errors.Is(err, innerErr) {
		t.Error("errors.Is should find the wrapped error")
	}

	unwrapped := errors.Unwrap(err)
	if unwrapped != innerErr {
		t.Errorf("Expected unwrapped error to be %v, got %v", innerErr, unwrapped)
	}
}

func TestSanitizeCommand(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		notContains string
		contains    string
	}{
		{
			name:        "Authorization header",
			input:       "curl -H 'Authorization: Bearer secret123' https://api.example.com",
			notContains: "secret123",
			contains:    "[REDACTED]",
		},
		{
			name:        "Basic auth",
			input:       "curl -u user:password https://example.com",
			notContains: "password", // This is in the pattern, but should still work
			contains:    "user",
		},
		{
			name:        "Cookie header",
			input:       "curl -H 'Cookie: session=abc123' https://example.com",
			notContains: "abc123",
			contains:    "[REDACTED]",
		},
		{
			name:        "API key in URL",
			input:       "curl https://api.example.com?api_key=secret123",
			notContains: "secret123",
			contains:    "[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gocurl.ParseError(tt.input, errors.New("test"))
			resultStr := result.Error()

			if tt.notContains != "" && strings.Contains(strings.ToLower(resultStr), strings.ToLower(tt.notContains)) {
				t.Errorf("Sanitized output should not contain %q, but got: %s", tt.notContains, resultStr)
			}

			if tt.contains != "" && !strings.Contains(resultStr, tt.contains) {
				t.Errorf("Sanitized output should contain %q, but got: %s", tt.contains, resultStr)
			}
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		notContains string
	}{
		{
			name:        "API key parameter",
			input:       "https://api.example.com/data?api_key=secret123&format=json",
			notContains: "secret123",
		},
		{
			name:        "Token parameter",
			input:       "https://example.com/auth?token=abc123def456",
			notContains: "abc123def456",
		},
		{
			name:        "Password parameter",
			input:       "https://example.com/login?user=john&password=secret",
			notContains: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gocurl.RequestError(tt.input, errors.New("test"))
			resultStr := result.Error()

			if strings.Contains(strings.ToLower(resultStr), strings.ToLower(tt.notContains)) {
				t.Errorf("Sanitized URL should not contain %q, but got: %s", tt.notContains, resultStr)
			}

			if !strings.Contains(resultStr, "[REDACTED]") {
				t.Errorf("Sanitized URL should contain [REDACTED], but got: %s", resultStr)
			}
		})
	}
}

func TestIsSensitiveHeader(t *testing.T) {
	tests := []struct {
		name        string
		header      string
		isSensitive bool
	}{
		{"Authorization", "Authorization", true},
		{"authorization", "authorization", true},
		{"AUTHORIZATION", "AUTHORIZATION", true},
		{"Cookie", "Cookie", true},
		{"X-API-Key", "X-API-Key", true},
		{"Content-Type", "Content-Type", false},
		{"Accept", "Accept", false},
		{"User-Agent", "User-Agent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gocurl.IsSensitiveHeader(tt.header)
			if result != tt.isSensitive {
				t.Errorf("IsSensitiveHeader(%q) = %v, want %v", tt.header, result, tt.isSensitive)
			}
		})
	}
}

func TestRedactHeaders(t *testing.T) {
	headers := map[string][]string{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer secret123"},
		"Cookie":        {"session=abc123"},
		"User-Agent":    {"gocurl/1.0"},
	}

	redacted := gocurl.RedactHeaders(headers)

	// Non-sensitive headers should be unchanged
	if redacted["Content-Type"][0] != "application/json" {
		t.Error("Content-Type should not be redacted")
	}
	if redacted["User-Agent"][0] != "gocurl/1.0" {
		t.Error("User-Agent should not be redacted")
	}

	// Sensitive headers should be redacted
	if redacted["Authorization"][0] != "[REDACTED]" {
		t.Errorf("Authorization should be redacted, got: %s", redacted["Authorization"][0])
	}
	if redacted["Cookie"][0] != "[REDACTED]" {
		t.Errorf("Cookie should be redacted, got: %s", redacted["Cookie"][0])
	}

	// Original headers should be unchanged
	if headers["Authorization"][0] == "[REDACTED]" {
		t.Error("Original headers should not be modified")
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("ParseError", func(t *testing.T) {
		err := gocurl.ParseError("curl -X GET", errors.New("test"))
		if !strings.Contains(err.Error(), "parse") {
			t.Error("ParseError should contain 'parse'")
		}
	})

	t.Run("RequestError", func(t *testing.T) {
		err := gocurl.RequestError("https://example.com", errors.New("test"))
		if !strings.Contains(err.Error(), "request") {
			t.Error("RequestError should contain 'request'")
		}
	})

	t.Run("ResponseError", func(t *testing.T) {
		err := gocurl.ResponseError("https://example.com", errors.New("test"))
		if !strings.Contains(err.Error(), "response") {
			t.Error("ResponseError should contain 'response'")
		}
	})

	t.Run("RetryError", func(t *testing.T) {
		err := gocurl.RetryError("https://example.com", 3, errors.New("test"))
		errStr := err.Error()
		if !strings.Contains(errStr, "retry") {
			t.Error("RetryError should contain 'retry'")
		}
		if !strings.Contains(errStr, "3") {
			t.Error("RetryError should contain attempt count")
		}
	})

	t.Run("ValidationError", func(t *testing.T) {
		err := gocurl.ValidationError("URL", errors.New("test"))
		if !strings.Contains(err.Error(), "validate") {
			t.Error("ValidationError should contain 'validate'")
		}
	})
}
