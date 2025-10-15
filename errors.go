package gocurl

import (
	"fmt"
	"strings"
)

// GocurlError provides structured error information with context
type GocurlError struct {
	Op      string // Operation: "parse", "request", "response", "retry", etc.
	Command string // Command snippet (sanitized)
	URL     string // Request URL (sanitized)
	Err     error  // Underlying error
}

// Error implements the error interface
func (e *GocurlError) Error() string {
	var parts []string

	if e.Op != "" {
		parts = append(parts, e.Op)
	}

	if e.URL != "" {
		parts = append(parts, fmt.Sprintf("url=%s", e.URL))
	}

	if e.Command != "" && e.Command != e.URL {
		// Truncate long commands
		cmd := e.Command
		if len(cmd) > 100 {
			cmd = cmd[:97] + "..."
		}
		parts = append(parts, fmt.Sprintf("cmd=%q", cmd))
	}

	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}

	return strings.Join(parts, ": ")
}

// Unwrap allows errors.Is and errors.As to work
func (e *GocurlError) Unwrap() error {
	return e.Err
}

// ParseError creates a parsing error
func ParseError(command string, err error) error {
	return &GocurlError{
		Op:      "parse",
		Command: sanitizeCommand(command),
		Err:     err,
	}
}

// RequestError creates a request execution error
func RequestError(url string, err error) error {
	return &GocurlError{
		Op:  "request",
		URL: sanitizeURL(url),
		Err: err,
	}
}

// ResponseError creates a response reading error
func ResponseError(url string, err error) error {
	return &GocurlError{
		Op:  "response",
		URL: sanitizeURL(url),
		Err: err,
	}
}

// RetryError creates a retry exhaustion error
func RetryError(url string, attempts int, err error) error {
	return &GocurlError{
		Op:  fmt.Sprintf("retry (after %d attempts)", attempts),
		URL: sanitizeURL(url),
		Err: err,
	}
}

// ValidationError creates an input validation error
func ValidationError(field string, err error) error {
	return &GocurlError{
		Op:      "validate",
		Command: field,
		Err:     err,
	}
}

// sensitiveHeaders are headers that should be redacted in logs/errors
var sensitiveHeaders = map[string]bool{
	"authorization":       true,
	"cookie":              true,
	"set-cookie":          true,
	"x-api-key":           true,
	"api-key":             true,
	"token":               true,
	"secret":              true,
	"password":            true,
	"proxy-authorization": true,
}

// sanitizeCommand removes sensitive information from commands
func sanitizeCommand(command string) string {
	// Redact common sensitive patterns
	lower := strings.ToLower(command)

	// Redact -u user:password (basic auth flag)
	if strings.Contains(lower, " -u ") {
		command = redactUserPassword(command)
	}

	// Redact Authorization headers (must be done before bearer/basic to catch the whole header value)
	if strings.Contains(lower, "authorization:") {
		command = redactPattern(command, "authorization:", "Authorization: [REDACTED]")
	}

	// Redact standalone Authorization keyword (for -H Authorization syntax without colon)
	if strings.Contains(lower, "authorization ") && !strings.Contains(command, "Authorization: [REDACTED]") {
		command = redactPattern(command, "authorization ", "Authorization [REDACTED]")
	}

	// Redact bearer tokens (only if not already redacted by Authorization header)
	if strings.Contains(lower, "bearer ") && !strings.Contains(command, "[REDACTED]") {
		command = redactPattern(command, "bearer ", "Bearer [REDACTED]")
	}

	// Redact basic auth (only if not already redacted)
	if strings.Contains(lower, "basic ") && !strings.Contains(command, "[REDACTED]") {
		command = redactPattern(command, "basic ", "Basic [REDACTED]")
	}

	// Redact cookies - redact everything after Cookie:
	if strings.Contains(lower, "cookie:") {
		command = redactPattern(command, "cookie:", "Cookie: [REDACTED]")
	}

	// Redact API keys in URLs
	if strings.Contains(lower, "api_key=") || strings.Contains(lower, "apikey=") || strings.Contains(lower, "key=") {
		command = redactURLParams(command, []string{"api_key", "apikey", "key", "token", "secret"})
	}

	return command
}

// redactUserPassword redacts password from -u user:password format
func redactUserPassword(command string) string {
	lower := strings.ToLower(command)
	idx := strings.Index(lower, " -u ")
	if idx == -1 {
		return command
	}

	// Start after " -u "
	start := idx + 4
	end := start

	// Find the end of the user:password string (space or end)
	for end < len(command) && command[end] != ' ' {
		end++
	}

	userPass := command[start:end]

	// Find the colon separating user and password
	colonIdx := strings.Index(userPass, ":")
	if colonIdx == -1 {
		return command // No password, just username
	}

	// Redact the password part
	user := userPass[:colonIdx]
	redacted := user + ":[REDACTED]"

	return command[:start] + redacted + command[end:]
}

// sanitizeURL removes sensitive query parameters
func sanitizeURL(url string) string {
	return redactURLParams(url, []string{"api_key", "apikey", "key", "token", "secret", "password"})
}

// redactPattern redacts content after a pattern until the next space or quote
func redactPattern(text, pattern, replacement string) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, strings.ToLower(pattern))

	if idx == -1 {
		return text
	}

	inQuotes, quoteChar := isPatternInQuotes(text, idx)

	if inQuotes {
		return redactQuotedPattern(text, idx, len(pattern), replacement, quoteChar)
	}

	return redactUnquotedPattern(text, idx, len(pattern), replacement)
}

// isPatternInQuotes checks if the pattern at idx is inside quotes
func isPatternInQuotes(text string, idx int) (bool, byte) {
	// Look backwards from pattern start to find if we're in quotes
	for i := idx - 1; i >= 0; i-- {
		if text[i] == '"' || text[i] == '\'' {
			return true, text[i]
		}
		// If we hit a space before a quote, we're not in quotes
		if text[i] == ' ' {
			break
		}
	}
	return false, 0
}

// redactQuotedPattern redacts a pattern that is inside quotes
func redactQuotedPattern(text string, idx, patternLen int, replacement string, quoteChar byte) string {
	// Find the closing quote
	closeIdx := idx + patternLen
	for closeIdx < len(text) && text[closeIdx] != quoteChar {
		closeIdx++
	}

	if closeIdx < len(text) {
		closeIdx++ // Include the closing quote
	}

	// Find the opening quote
	openIdx := idx - 1
	for openIdx >= 0 && text[openIdx] != quoteChar {
		openIdx--
	}

	if openIdx >= 0 {
		// Replace the entire quoted string
		return text[:openIdx] + string(quoteChar) + replacement + string(quoteChar) + text[closeIdx:]
	}

	return text
}

// redactUnquotedPattern redacts a pattern that is not inside quotes
func redactUnquotedPattern(text string, idx, patternLen int, replacement string) string {
	// Start right after the pattern
	start := idx + patternLen

	// Skip any whitespace after the pattern
	for start < len(text) && (text[start] == ' ' || text[start] == '\t') {
		start++
	}

	// Find the end of the value (space or special char)
	end := start
	for end < len(text) {
		ch := text[end]
		if ch == ' ' || ch == '"' || ch == '\'' || ch == '\n' || ch == '\r' {
			break
		}
		end++
	}

	// Replace the value
	return text[:idx] + replacement + text[end:]
}

// redactURLParams redacts specified query parameters from URLs
func redactURLParams(url string, params []string) string {
	for _, param := range params {
		// Look for param=value
		pattern := param + "="
		idx := strings.Index(strings.ToLower(url), strings.ToLower(pattern))

		if idx == -1 {
			continue
		}

		start := idx + len(pattern)
		end := start

		// Find end of parameter value (& or end of string)
		for end < len(url) {
			ch := url[end]
			if ch == '&' || ch == ' ' || ch == '"' || ch == '\'' {
				break
			}
			end++
		}

		// Replace value with [REDACTED]
		url = url[:start] + "[REDACTED]" + url[end:]
	}

	return url
}

// IsSensitiveHeader checks if a header should be redacted
func IsSensitiveHeader(headerName string) bool {
	return sensitiveHeaders[strings.ToLower(headerName)]
}

// RedactHeaders creates a copy of headers with sensitive values redacted
func RedactHeaders(headers map[string][]string) map[string][]string {
	if headers == nil {
		return nil
	}

	redacted := make(map[string][]string, len(headers))

	for name, values := range headers {
		if IsSensitiveHeader(name) {
			redacted[name] = []string{"[REDACTED]"}
		} else {
			redacted[name] = values
		}
	}

	return redacted
}
