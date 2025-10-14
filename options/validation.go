package options

import (
	"fmt"
	"net/http"
	"strings"
)

// Validation constants (configurable for testing)
const (
	MaxURLLength   = 8192             // 8KB
	MaxHeaders     = 100              // Maximum number of headers
	MaxHeaderSize  = 8192             // 8KB per header
	MaxBodySize    = 10 * 1024 * 1024 // 10MB default
	MaxFormFields  = 1000             // Maximum form fields
	MaxQueryParams = 1000             // Maximum query parameters
)

// validHTTPMethods defines allowed HTTP methods
var validHTTPMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodPost:    true,
	http.MethodPut:     true,
	http.MethodDelete:  true,
	http.MethodPatch:   true,
	http.MethodHead:    true,
	http.MethodOptions: true,
	http.MethodConnect: true,
	http.MethodTrace:   true,
}

// validateMethod checks if HTTP method is valid
func validateMethod(method string) error {
	if method == "" {
		return nil // Empty is OK, defaults to GET
	}

	if !validHTTPMethods[method] {
		return fmt.Errorf("invalid HTTP method: %s (allowed: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, CONNECT, TRACE)", method)
	}

	return nil
}

// validateURL checks URL length
func validateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	if len(url) > MaxURLLength {
		return fmt.Errorf("URL too long: %d bytes (max: %d)", len(url), MaxURLLength)
	}

	return nil
}

// validateHeaders checks header count and sizes
func validateHeaders(headers http.Header) error {
	if headers == nil {
		return nil
	}

	if len(headers) > MaxHeaders {
		return fmt.Errorf("too many headers: %d (max: %d)", len(headers), MaxHeaders)
	}

	for key, values := range headers {
		// Check forbidden headers
		if isForbiddenHeader(key) {
			return fmt.Errorf("cannot set forbidden header: %s (managed automatically)", key)
		}

		// Check header size
		for _, value := range values {
			if len(key)+len(value) > MaxHeaderSize {
				return fmt.Errorf("header too large: %s (max: %d bytes)", key, MaxHeaderSize)
			}
		}
	}

	return nil
}

// isForbiddenHeader checks if header should be managed automatically
func isForbiddenHeader(key string) bool {
	forbidden := []string{
		"Host",              // Managed by http.Request
		"Content-Length",    // Calculated automatically
		"Transfer-Encoding", // Managed by transport
	}

	canonicalKey := http.CanonicalHeaderKey(key)
	for _, f := range forbidden {
		if canonicalKey == f {
			return true
		}
	}

	return false
}

// validateBody checks body size
func validateBody(body string, limit int64) error {
	if body == "" {
		return nil
	}

	if limit <= 0 {
		limit = MaxBodySize
	}

	if int64(len(body)) > limit {
		return fmt.Errorf("body too large: %d bytes (max: %d)", len(body), limit)
	}

	return nil
}

// validateForm checks form field count
func validateForm(form map[string][]string) error {
	if form == nil {
		return nil
	}

	count := 0
	for _, values := range form {
		count += len(values)
	}

	if count > MaxFormFields {
		return fmt.Errorf("too many form fields: %d (max: %d)", count, MaxFormFields)
	}

	return nil
}

// validateQueryParams checks query parameter count
func validateQueryParams(params map[string][]string) error {
	if params == nil {
		return nil
	}

	count := 0
	for _, values := range params {
		count += len(values)
	}

	if count > MaxQueryParams {
		return fmt.Errorf("too many query parameters: %d (max: %d)", count, MaxQueryParams)
	}

	return nil
}

// validateSecureAuth warns about insecure authentication
func validateSecureAuth(url string, hasBasicAuth bool, hasBearerToken bool) error {
	// Check if using HTTP (not HTTPS)
	isHTTP := strings.HasPrefix(url, "http://")

	if !isHTTP {
		return nil // HTTPS is secure
	}

	// Check for override environment variable
	if allowInsecureAuth() {
		return nil
	}

	// Warn about BasicAuth over HTTP
	if hasBasicAuth {
		return fmt.Errorf("security warning: BasicAuth over HTTP is insecure (credentials sent in plaintext). Use HTTPS or set GOCURL_ALLOW_INSECURE_AUTH=1")
	}

	// Warn about BearerToken over HTTP
	if hasBearerToken {
		return fmt.Errorf("security warning: Bearer token over HTTP is insecure. Use HTTPS or set GOCURL_ALLOW_INSECURE_AUTH=1")
	}

	return nil
}

// allowInsecureAuth checks if insecure auth is explicitly allowed
func allowInsecureAuth() bool {
	// Check environment variable
	return false // For now, always enforce. Can add os.Getenv check later
}
