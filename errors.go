package gocurl

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

// GocurlError provides structured error information with context. It is the
// single concrete error type gocurl returns; callers branch on Kind (the
// machine-readable discriminator) or use errors.Is with the Err* sentinels,
// and errors.As to recover this type for full detail.
//
// See specs/08-error-model.md.
type GocurlError struct {
	Op      string // Operation: "parse", "request", "response", "retry", etc. (legacy/message)
	Kind    Kind   // machine-readable classification
	Command string // Command snippet (sanitized)
	URL     string // Request URL (sanitized)
	Status  int    // HTTP status when Kind == KindServerStatus, else 0
	Attempt int    // attempts made when Kind == KindRetryExhausted, else 0
	Err     error  // Underlying error
}

// Error implements the error interface. The message shape is backward
// compatible (op: url=… cmd=… underlying); the status is appended to the op
// segment for KindServerStatus. The joined message is run through
// scrubErrorString so credentials leaking in from wrapped stdlib errors never
// appear.
func (e *GocurlError) Error() string {
	var parts []string

	op := e.Op
	if op == "" && e.Kind != KindUnknown {
		op = e.Kind.String()
	}
	if e.Kind == KindServerStatus && e.Status != 0 {
		if op == "" {
			op = "server status"
		}
		op = fmt.Sprintf("%s (%d)", op, e.Status)
	}
	if op != "" {
		parts = append(parts, op)
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

	return scrubErrorString(strings.Join(parts, ": "))
}

// Unwrap allows errors.Is and errors.As to work
func (e *GocurlError) Unwrap() error {
	return e.Err
}

// Is reports whether target is the kindSentinel matching this error's Kind.
// errors.Is walks the Unwrap chain for us, so a retry-exhausted error wrapping a
// timeout still matches errors.Is(err, ErrTimeout), and errors.Is(err,
// context.DeadlineExceeded) keeps resolving through the wrapped error.
func (e *GocurlError) Is(target error) bool {
	if ks, ok := target.(*kindSentinel); ok {
		return e.Kind == ks.kind
	}
	return false
}

// Timeout reports whether the error was caused by a deadline/timeout. It is
// nil-Err safe and consults a wrapped net.Error / context.DeadlineExceeded.
func (e *GocurlError) Timeout() bool {
	if e == nil {
		return false
	}
	if e.Kind == KindTimeout {
		return true
	}
	if e.Err == nil {
		return false
	}
	if errors.Is(e.Err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	if errors.As(e.Err, &ne) {
		return ne.Timeout()
	}
	return false
}

// Temporary reports whether the failure is plausibly transient. Advisory and
// conservative; it is NOT a retry decision on its own (idempotency still
// governs retries — see Spec 04).
func (e *GocurlError) Temporary() bool {
	if e == nil {
		return false
	}
	switch e.Kind {
	case KindConnect, KindTimeout:
		return true
	case KindServerStatus:
		return shouldRetry(e.Status, nil)
	case KindRetryExhausted:
		var inner *GocurlError
		if errors.As(e.Err, &inner) && inner != e {
			return inner.Temporary()
		}
	}
	return false
}

// Retryable reports whether gocurl's resilience layer considers this error safe
// to retry for an idempotent request: connect and timeout always, server-status
// only for the retryable status codes. TLS and retry-exhausted are terminal.
func (e *GocurlError) Retryable() bool {
	if e == nil {
		return false
	}
	switch e.Kind {
	case KindConnect, KindTimeout:
		return true
	case KindServerStatus:
		return shouldRetry(e.Status, nil)
	}
	return false
}

// ErrTooManyRedirects is wrapped by the redirect-cap error so callers (and the
// CLI's exit-code mapping) can match it with errors.Is even after net/http wraps
// it in a *url.Error and the engine wraps that in a GocurlError.
var ErrTooManyRedirects = errors.New("too many redirects")

// ParseError creates a parsing error (Kind == KindParse).
func ParseError(command string, err error) error {
	return &GocurlError{
		Op:      "parse",
		Kind:    KindParse,
		Command: sanitizeCommand(command),
		Err:     err,
	}
}

// parseOrPassthrough classifies a parse-stage failure. An error that is ALREADY a
// typed *GocurlError (e.g. the URL ValidationError from normalizeURL) is returned
// unchanged so its Kind — and therefore the CLI exit code — is preserved; any
// other error (an unknown flag, a tokenizer failure) is wrapped as a ParseError.
func parseOrPassthrough(command string, err error) error {
	var ge *GocurlError
	if errors.As(err, &ge) {
		return err
	}
	return ParseError(command, err)
}

// RequestError creates a request execution error. The wrapped transport error
// is classified (KindConnect/KindTLS/KindTimeout/KindCanceled) rather than
// blindly tagged, while the Op stays "request" for message compatibility.
func RequestError(url string, err error) error {
	return &GocurlError{
		Op:   "request",
		Kind: classifyTransportError(err),
		URL:  sanitizeURL(url),
		Err:  err,
	}
}

// ResponseError creates a response reading error (Kind == KindBodyRead).
func ResponseError(url string, err error) error {
	return &GocurlError{
		Op:   "response",
		Kind: KindBodyRead,
		URL:  sanitizeURL(url),
		Err:  err,
	}
}

// RetryError creates a retry exhaustion error (Kind == KindRetryExhausted). It
// chains the last attempt's (already classified) error so errors.Is(err,
// ErrTimeout) and friends still resolve through the wrapper.
func RetryError(url string, attempts int, err error) error {
	return &GocurlError{
		Op:      fmt.Sprintf("retry (after %d attempts)", attempts),
		Kind:    KindRetryExhausted,
		URL:     sanitizeURL(url),
		Attempt: attempts,
		Err:     err,
	}
}

// ValidationError creates an input validation error (Kind == KindValidation).
func ValidationError(field string, err error) error {
	return &GocurlError{
		Op:      "validate",
		Kind:    KindValidation,
		Command: field,
		Err:     err,
	}
}

// ServerStatusError reports a non-2xx HTTP response surfaced as an error. It is
// only produced when fail-on-status is enabled (WithFailOnStatus / -f); the
// engine still returns the *http.Response alongside it so the caller may read
// the error body.
func ServerStatusError(url string, status int) error {
	return &GocurlError{
		Op:     "server status",
		Kind:   KindServerStatus,
		URL:    sanitizeURL(url),
		Status: status,
	}
}

// BodyReadError reports a failure reading/decoding the response body, including
// the over-limit (truncation) case (Kind == KindBodyRead).
func BodyReadError(url string, err error) error {
	return &GocurlError{
		Op:   "body read",
		Kind: KindBodyRead,
		URL:  sanitizeURL(url),
		Err:  err,
	}
}

// ConnectError reports a dial/DNS/proxy-connect failure (Kind == KindConnect).
func ConnectError(url string, err error) error {
	return &GocurlError{
		Op:   "connect",
		Kind: KindConnect,
		URL:  sanitizeURL(url),
		Err:  err,
	}
}

// TLSError reports a TLS handshake/verification/pin failure (Kind == KindTLS).
func TLSError(url string, err error) error {
	return &GocurlError{
		Op:   "tls",
		Kind: KindTLS,
		URL:  sanitizeURL(url),
		Err:  err,
	}
}

// TimeoutError reports a deadline exceeded (Kind == KindTimeout).
func TimeoutError(url string, err error) error {
	return &GocurlError{
		Op:   "timeout",
		Kind: KindTimeout,
		URL:  sanitizeURL(url),
		Err:  err,
	}
}

// CanceledError reports a context cancellation by the caller (Kind == KindCanceled).
func CanceledError(url string, err error) error {
	return &GocurlError{
		Op:   "canceled",
		Kind: KindCanceled,
		URL:  sanitizeURL(url),
		Err:  err,
	}
}

// sensitiveHeaders are headers that should be redacted in logs/errors
var sensitiveHeaders = map[string]bool{
	"authorization":        true,
	"cookie":               true,
	"set-cookie":           true,
	"x-api-key":            true,
	"api-key":              true,
	"x-auth-token":         true, // canonical set merges verbose.go's list
	"auth-token":           true,
	"x-amz-security-token": true, // AWS STS temporary session token (bearer-equivalent)
	"x-csrf-token":         true,
	"token":                true,
	"secret":               true,
	"password":             true,
	"proxy-authorization":  true,
}

// sanitizeCommand removes sensitive information from commands
func sanitizeCommand(command string) string {
	// Prefix a space so the -u/--user and -b/--cookie flag matchers (which key off
	// a leading space) also fire when the flag is the FIRST token — e.g. args
	// joined without a leading space ("-u user:pass …"). The prefix is stripped
	// again immediately after the credential redactions.
	command = " " + command
	// Redact -u/--user user:password (basic auth flag, space- or =-separated).
	command = redactUserPassword(command)
	// Redact -b/--cookie values: session cookies are credentials, and the parser
	// accepts the flag form just like the Cookie: header form below.
	command = redactFlagValue(command, []string{" --cookie=", " --cookie ", " -b=", " -b "})
	command = strings.TrimPrefix(command, " ")

	// Recompute after the credential redactions above so the substring gates below
	// operate on the current command.
	lower := strings.ToLower(command)

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

// redactUserPassword redacts the password from a -u/--user user:password value.
// curl accepts both the short (-u) and long (--user) flags, each either
// space- or =-separated, so all four forms are handled; the username is kept and
// only the password segment after the first colon is replaced.
func redactUserPassword(command string) string {
	lower := strings.ToLower(command)
	// Longest forms first so "--user=" wins over "--user " and "-u=" over "-u ".
	for _, flag := range []string{" --user=", " --user ", " -u=", " -u "} {
		idx := strings.Index(lower, flag)
		if idx == -1 {
			continue
		}

		start := idx + len(flag)
		end := start
		// Find the end of the user:password token (space or end of string).
		for end < len(command) && command[end] != ' ' {
			end++
		}

		userPass := command[start:end]
		colonIdx := strings.Index(userPass, ":")
		if colonIdx == -1 {
			continue // No password segment to redact for this flag form.
		}

		redacted := userPass[:colonIdx] + ":[REDACTED]"
		return command[:start] + redacted + command[end:]
	}

	return command
}

// redactFlagValue replaces the entire value token following the first matching
// flag form with [REDACTED]. Used for flags whose whole argument is sensitive
// (e.g. -b/--cookie). flags should be ordered longest-first.
func redactFlagValue(command string, flags []string) string {
	lower := strings.ToLower(command)
	for _, flag := range flags {
		idx := strings.Index(lower, flag)
		if idx == -1 {
			continue
		}

		start := idx + len(flag)
		end := start
		// The value runs to the next space (a quoted value has no internal space
		// in practice; the surrounding quotes are consumed into the token).
		for end < len(command) && command[end] != ' ' {
			end++
		}
		if end == start {
			continue // No value token present.
		}

		return command[:start] + "[REDACTED]" + command[end:]
	}

	return command
}

// sanitizeURL removes sensitive query parameters AND any basic-auth userinfo
// (user:password@) so a URL is safe to log or surface in an error. It is the
// single redaction path shared by error messages and observability sinks.
func sanitizeURL(url string) string {
	url = redactURLParams(url, []string{"api_key", "apikey", "key", "token", "secret", "password"})
	return userinfoCredRe.ReplaceAllString(url, "://$1:[REDACTED]@")
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

// redactURLParams redacts specified query parameters from URLs. It redacts EVERY
// occurrence of each parameter, not just the first: repeated sensitive keys
// (e.g. ?api_key=A&…&api_key=B) are legitimate in URLs and SDK-built requests,
// and leaving the later values in clear text would leak them to logs/errors.
func redactURLParams(url string, params []string) string {
	for _, param := range params {
		// Look for param=value
		pattern := param + "="
		searchFrom := 0
		for {
			rel := strings.Index(strings.ToLower(url[searchFrom:]), strings.ToLower(pattern))
			if rel == -1 {
				break
			}
			idx := searchFrom + rel
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
			// Advance past the replacement so we keep scanning for more
			// occurrences of the same parameter without re-matching it.
			searchFrom = start + len("[REDACTED]")
		}
	}

	return url
}

// RedactURL returns a copy of raw safe to log or display: sensitive query
// parameters and any basic-auth userinfo are redacted. It is the exported entry
// point to the single URL-redaction path (sanitizeURL).
func RedactURL(raw string) string { return sanitizeURL(raw) }

// RedactCommand returns a copy of a curl command safe to log or display, with
// credentials (`-u`, Authorization/Bearer/Basic, Cookie, sensitive URL params)
// redacted. It is the exported entry point to sanitizeCommand.
func RedactCommand(cmd string) string { return sanitizeCommand(cmd) }

// IsSensitiveHeader reports whether a header value should be redacted in logs,
// verbose output, spans, and errors. It first consults the canonical exact-match
// set, then falls back to a bounded heuristic: no fixed list can enumerate the
// open set of vendor auth headers (X-Goog-Api-Key, Private-Token, X-Vault-Token,
// X-Functions-Key, …), so anything that looks credential-bearing — by content or
// by a credential-style suffix — is redacted. This errs toward redaction, which
// is the intended fail-safe (Spec 07: never leak secrets).
func IsSensitiveHeader(headerName string) bool {
	name := strings.ToLower(headerName)
	if sensitiveHeaders[name] {
		return true
	}
	if strings.Contains(name, "authorization") ||
		strings.Contains(name, "apikey") ||
		strings.Contains(name, "password") ||
		strings.Contains(name, "secret") {
		return true
	}
	for _, suffix := range []string{"-api-key", "-apikey", "-auth-token", "-token", "-secret", "-key"} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
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
