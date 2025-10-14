package options

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// TestValidateMethod tests HTTP method validation
func TestValidateMethod_Valid(t *testing.T) {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE"}

	for _, method := range validMethods {
		t.Run(method, func(t *testing.T) {
			err := validateMethod(method)
			if err != nil {
				t.Errorf("validateMethod(%s) returned error: %v", method, err)
			}
		})
	}
}

func TestValidateMethod_Empty(t *testing.T) {
	err := validateMethod("")
	if err != nil {
		t.Errorf("validateMethod(\"\") should allow empty (defaults to GET), got error: %v", err)
	}
}

func TestValidateMethod_Invalid(t *testing.T) {
	invalidMethods := []string{"INVALID", "HACK", "hack", "get", "GeT", "CUSTOM"}

	for _, method := range invalidMethods {
		t.Run(method, func(t *testing.T) {
			err := validateMethod(method)
			if err == nil {
				t.Errorf("validateMethod(%s) should return error, got nil", method)
			}
			if !strings.Contains(err.Error(), "invalid HTTP method") {
				t.Errorf("validateMethod(%s) error should mention 'invalid HTTP method', got: %v", method, err)
			}
		})
	}
}

// TestValidateURL tests URL validation
func TestValidateURL_Valid(t *testing.T) {
	validURLs := []string{
		"https://example.com",
		"http://localhost:8080/path",
		"https://api.example.com/v1/users?id=123",
	}

	for _, url := range validURLs {
		t.Run(url, func(t *testing.T) {
			err := validateURL(url)
			if err != nil {
				t.Errorf("validateURL(%s) returned error: %v", url, err)
			}
		})
	}
}

func TestValidateURL_Empty(t *testing.T) {
	err := validateURL("")
	if err == nil {
		t.Error("validateURL(\"\") should return error for empty URL")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("validateURL(\"\") error should mention 'cannot be empty', got: %v", err)
	}
}

func TestValidateURL_TooLong(t *testing.T) {
	longURL := "https://example.com/" + strings.Repeat("a", MaxURLLength)
	err := validateURL(longURL)
	if err == nil {
		t.Error("validateURL should return error for URL exceeding max length")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Errorf("validateURL error should mention 'too long', got: %v", err)
	}
}

// TestValidateHeaders tests header validation
func TestValidateHeaders_Valid(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("Authorization", "Bearer token")
	headers.Set("X-Custom-Header", "value")

	err := validateHeaders(headers)
	if err != nil {
		t.Errorf("validateHeaders returned error: %v", err)
	}
}

func TestValidateHeaders_Nil(t *testing.T) {
	err := validateHeaders(nil)
	if err != nil {
		t.Errorf("validateHeaders(nil) should not return error, got: %v", err)
	}
}

func TestValidateHeaders_TooMany(t *testing.T) {
	headers := http.Header{}
	for i := 0; i <= MaxHeaders; i++ {
		headers.Set(fmt.Sprintf("X-Header-%d", i), "value")
	}

	err := validateHeaders(headers)
	if err == nil {
		t.Error("validateHeaders should return error when exceeding max headers")
	}
	if !strings.Contains(err.Error(), "too many headers") {
		t.Errorf("validateHeaders error should mention 'too many headers', got: %v", err)
	}
}

func TestValidateHeaders_TooLarge(t *testing.T) {
	headers := http.Header{}
	largeValue := strings.Repeat("a", MaxHeaderSize)
	headers.Set("X-Large-Header", largeValue)

	err := validateHeaders(headers)
	if err == nil {
		t.Error("validateHeaders should return error for too large header")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("validateHeaders error should mention 'too large', got: %v", err)
	}
}

func TestValidateHeaders_Forbidden(t *testing.T) {
	forbiddenHeaders := []string{"Host", "Content-Length", "Transfer-Encoding"}

	for _, headerName := range forbiddenHeaders {
		t.Run(headerName, func(t *testing.T) {
			headers := http.Header{}
			headers.Set(headerName, "value")

			err := validateHeaders(headers)
			if err == nil {
				t.Errorf("validateHeaders should return error for forbidden header: %s", headerName)
			}
			if !strings.Contains(err.Error(), "forbidden header") {
				t.Errorf("validateHeaders error should mention 'forbidden header', got: %v", err)
			}
		})
	}
}

// TestValidateBody tests body validation
func TestValidateBody_Valid(t *testing.T) {
	validBodies := []string{
		"",
		"short body",
		`{"key": "value"}`,
		strings.Repeat("a", 1000),
	}

	for i, body := range validBodies {
		t.Run(fmt.Sprintf("body_%d", i), func(t *testing.T) {
			err := validateBody(body, MaxBodySize)
			if err != nil {
				t.Errorf("validateBody returned error: %v", err)
			}
		})
	}
}

func TestValidateBody_TooLarge(t *testing.T) {
	largeBody := strings.Repeat("a", int(MaxBodySize)+1)
	err := validateBody(largeBody, MaxBodySize)
	if err == nil {
		t.Error("validateBody should return error for body exceeding limit")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("validateBody error should mention 'too large', got: %v", err)
	}
}

func TestValidateBody_CustomLimit(t *testing.T) {
	body := strings.Repeat("a", 1000)
	customLimit := int64(500)

	err := validateBody(body, customLimit)
	if err == nil {
		t.Error("validateBody should return error when exceeding custom limit")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("validateBody error should mention 'too large', got: %v", err)
	}
}

// TestValidateForm tests form validation
func TestValidateForm_Valid(t *testing.T) {
	form := map[string][]string{
		"field1": {"value1"},
		"field2": {"value2", "value3"},
	}

	err := validateForm(form)
	if err != nil {
		t.Errorf("validateForm returned error: %v", err)
	}
}

func TestValidateForm_Nil(t *testing.T) {
	err := validateForm(nil)
	if err != nil {
		t.Errorf("validateForm(nil) should not return error, got: %v", err)
	}
}

func TestValidateForm_TooManyFields(t *testing.T) {
	form := make(map[string][]string)
	for i := 0; i <= MaxFormFields; i++ {
		form[fmt.Sprintf("field%d", i)] = []string{"value"}
	}

	err := validateForm(form)
	if err == nil {
		t.Error("validateForm should return error when exceeding max fields")
	}
	if !strings.Contains(err.Error(), "too many form fields") {
		t.Errorf("validateForm error should mention 'too many form fields', got: %v", err)
	}
}

// TestValidateQueryParams tests query parameter validation
func TestValidateQueryParams_Valid(t *testing.T) {
	params := map[string][]string{
		"param1": {"value1"},
		"param2": {"value2", "value3"},
	}

	err := validateQueryParams(params)
	if err != nil {
		t.Errorf("validateQueryParams returned error: %v", err)
	}
}

func TestValidateQueryParams_Nil(t *testing.T) {
	err := validateQueryParams(nil)
	if err != nil {
		t.Errorf("validateQueryParams(nil) should not return error, got: %v", err)
	}
}

func TestValidateQueryParams_TooMany(t *testing.T) {
	params := make(map[string][]string)
	for i := 0; i <= MaxQueryParams; i++ {
		params[fmt.Sprintf("param%d", i)] = []string{"value"}
	}

	err := validateQueryParams(params)
	if err == nil {
		t.Error("validateQueryParams should return error when exceeding max params")
	}
	if !strings.Contains(err.Error(), "too many query parameters") {
		t.Errorf("validateQueryParams error should mention 'too many query parameters', got: %v", err)
	}
}

// TestValidateSecureAuth tests security validation
func TestValidateSecureAuth_HTTPS(t *testing.T) {
	err := validateSecureAuth("https://example.com", true, false)
	if err != nil {
		t.Errorf("validateSecureAuth should not error for HTTPS with BasicAuth, got: %v", err)
	}

	err = validateSecureAuth("https://example.com", false, true)
	if err != nil {
		t.Errorf("validateSecureAuth should not error for HTTPS with BearerToken, got: %v", err)
	}
}

func TestValidateSecureAuth_HTTP_NoAuth(t *testing.T) {
	err := validateSecureAuth("http://example.com", false, false)
	if err != nil {
		t.Errorf("validateSecureAuth should not error for HTTP without auth, got: %v", err)
	}
}

func TestValidateSecureAuth_HTTP_BasicAuth(t *testing.T) {
	err := validateSecureAuth("http://example.com", true, false)
	if err == nil {
		t.Error("validateSecureAuth should error for BasicAuth over HTTP")
	}
	if !strings.Contains(err.Error(), "insecure") {
		t.Errorf("validateSecureAuth error should mention 'insecure', got: %v", err)
	}
}

func TestValidateSecureAuth_HTTP_BearerToken(t *testing.T) {
	err := validateSecureAuth("http://example.com", false, true)
	if err == nil {
		t.Error("validateSecureAuth should error for Bearer token over HTTP")
	}
	if !strings.Contains(err.Error(), "insecure") {
		t.Errorf("validateSecureAuth error should mention 'insecure', got: %v", err)
	}
}

// TestValidation_ConcurrentSafe tests that validation is safe for concurrent use
func TestValidation_ConcurrentSafe(t *testing.T) {
	const goroutines = 50

	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			// Test all validators concurrently
			_ = validateMethod("GET")
			_ = validateURL("https://example.com")

			headers := http.Header{}
			headers.Set("Content-Type", "application/json")
			_ = validateHeaders(headers)

			_ = validateBody("test body", MaxBodySize)

			form := map[string][]string{"field": {"value"}}
			_ = validateForm(form)

			params := map[string][]string{"param": {"value"}}
			_ = validateQueryParams(params)

			_ = validateSecureAuth("https://example.com", true, false)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}
}
