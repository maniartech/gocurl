package options

import (
	"fmt"
	"net/http"
	"os"
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

// validateMethod checks that the HTTP method is a valid RFC 7230 token. It does
// NOT restrict to the standard verbs: curl (and gocurl) must support custom and
// WebDAV methods (PROPFIND, MKCALENDAR, …); only methods containing illegal
// characters (spaces, control chars, separators) are rejected.
func validateMethod(method string) error {
	if method == "" {
		return nil // Empty is OK, defaults to GET
	}
	for _, r := range method {
		if !isMethodTokenChar(r) {
			return fmt.Errorf("invalid HTTP method %q: contains an illegal character", method)
		}
	}
	return nil
}

// isMethodTokenChar reports whether r is a valid RFC 7230 "tchar".
func isMethodTokenChar(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	}
	switch r {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	}
	return false
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

// validateSecureAuth fails closed when credentials would be sent over cleartext
// HTTP. The allowInsecure argument (from WithAllowInsecureAuth) and the
// GOCURL_ALLOW_INSECURE_AUTH=1 environment variable both override it.
func validateSecureAuth(url string, hasBasicAuth bool, hasBearerToken bool, allowInsecure bool) error {
	// Case-insensitive: net/url lowercases the scheme, so "HTTP://" / "Http://"
	// travel over plaintext just like "http://". A case-sensitive prefix check
	// would fail open and send credentials in the clear. We fold only the 7-byte
	// scheme prefix in place rather than strings.ToLower(url), which would allocate a
	// full lowercased copy of the URL on every request just to inspect those 7 bytes.
	if !hasPlaintextHTTPPrefix(url) {
		return nil // HTTPS (or scheme-relative/other) is not plaintext HTTP
	}

	if allowInsecure || allowInsecureAuth() {
		return nil
	}

	if hasBasicAuth {
		return fmt.Errorf("BasicAuth over HTTP is insecure (credentials sent in plaintext); use HTTPS, set GOCURL_ALLOW_INSECURE_AUTH=1, or WithAllowInsecureAuth(true)")
	}
	if hasBearerToken {
		return fmt.Errorf("Bearer token over HTTP is insecure; use HTTPS, set GOCURL_ALLOW_INSECURE_AUTH=1, or WithAllowInsecureAuth(true)")
	}
	return nil
}

// hasPlaintextHTTPPrefix reports whether url begins with the "http://" scheme,
// case-insensitively, WITHOUT allocating. strings.ToLower(url) would allocate a full
// lowercased copy of the URL on every request just to test a 7-byte prefix; URL schemes
// are ASCII (RFC 3986), so an in-place ASCII case-fold of those 7 bytes is sufficient
// and equivalent.
func hasPlaintextHTTPPrefix(url string) bool {
	const p = "http://"
	if len(url) < len(p) {
		return false
	}
	for i := 0; i < len(p); i++ {
		c := url[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != p[i] {
			return false
		}
	}
	return true
}

// allowInsecureAuth reports whether the GOCURL_ALLOW_INSECURE_AUTH environment
// variable opts out of the plaintext-auth check.
func allowInsecureAuth() bool {
	return os.Getenv("GOCURL_ALLOW_INSECURE_AUTH") == "1"
}

// ValidateRequest runs the runtime input validators that protect the live
// request path: method, headers (count/size/forbidden Host/Content-Length/
// Transfer-Encoding), in-memory body size, form/query counts, and the secure-auth
// (plaintext-credential) policy. Streaming bodies (BodyStream) are exempt from the
// in-memory body cap — only opts.Body is size-checked.
func ValidateRequest(opts *RequestOptions) error {
	if err := validateMethod(opts.Method); err != nil {
		return err
	}
	if err := validateURL(opts.URL); err != nil {
		return err
	}
	if err := validateHeaders(opts.Headers); err != nil {
		return err
	}
	if opts.Body != "" {
		limit := opts.ResponseBodyLimit
		if limit <= 0 {
			limit = MaxBodySize
		}
		if err := validateBody(opts.Body, limit); err != nil {
			return err
		}
	}
	if len(opts.Form) > 0 {
		if err := validateForm(opts.Form); err != nil {
			return err
		}
	}
	if len(opts.QueryParams) > 0 {
		if err := validateQueryParams(opts.QueryParams); err != nil {
			return err
		}
	}
	return validateSecureAuth(opts.URL, opts.BasicAuth != nil, opts.BearerToken != "", opts.AllowInsecureAuth)
}
