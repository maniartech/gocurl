package gocurl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/maniartech/gocurl/options"
	"github.com/maniartech/gocurl/tokenizer"
)

// Variables type for safe variable substitution
type Variables map[string]string

// ============================================================================
// PRIMARY API - Auto-detection (Use this for 90% of cases)
// ============================================================================

// Curl executes an HTTP request using curl-compatible syntax.
//
// Automatically detects input format:
// - Single argument: Parsed as shell command (supports multi-line, backslashes, comments)
// - Multiple arguments: Treated as separate arguments (no parsing)
//
// Environment variables ($VAR and ${VAR}) are automatically expanded from os.Environ.
// For explicit control, use CurlCommand() or CurlArgs().
//
// Examples:
//
//	// Variadic (auto-detected)
//	resp, err := Curl(ctx, "-H", "X-Token: abc", "https://example.com")
//	defer resp.Body.Close()
//
//	// Shell command (auto-detected)
//	resp, err := Curl(ctx, `curl -H 'X-Token: abc' https://example.com`)
//	defer resp.Body.Close()
//
//	// Multi-line (auto-detected)
//	resp, err := Curl(ctx, `
//	  curl -X POST https://api.example.com \
//	    -H 'Content-Type: application/json' \
//	    -d '{"key":"value"}'
//	`)
//	defer resp.Body.Close()
//
//	// Access response details
//	fmt.Println(resp.StatusCode)
//	fmt.Println(resp.Header.Get("Content-Type"))
func Curl(ctx context.Context, command ...string) (*http.Response, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("no command provided")
	}

	// Auto-detect: single string = shell command, multiple = args
	if len(command) == 1 {
		return CurlCommand(ctx, command[0])
	}
	return CurlArgs(ctx, command...)
}

// CurlCommand executes a curl command from a shell-style string.
// Handles multi-line commands, backslash continuations, and comments.
// Environment variables are automatically expanded.
func CurlCommand(ctx context.Context, command string) (*http.Response, error) {
	// Preprocess multi-line command
	processed := preprocessMultilineCommand(command)

	// Tokenize
	tok := tokenizer.NewTokenizer()
	if err := tok.Tokenize(processed); err != nil {
		return nil, parseOrPassthrough(command, err)
	}

	tokens := tok.GetTokens()

	// Auto-expand environment variables
	tokens = expandEnvInTokens(tokens)

	// Convert to request options
	opts, err := convertTokensToRequestOptions(tokens)
	if err != nil {
		return nil, parseOrPassthrough(command, err)
	}

	return executeOpts(ctx, opts)
}

// executeOpts runs the request and returns the live response. The body is
// streamed (never buffered or written to stdout by the library); the caller is
// responsible for reading and closing resp.Body.
func executeOpts(ctx context.Context, opts *options.RequestOptions) (*http.Response, error) {
	resp, err := doRequest(ctx, opts)
	if err != nil {
		return nil, err
	}
	if opts.ResponseBodyLimit > 0 {
		resp.Body = newLimitedBody(resp.Body, opts.ResponseBodyLimit)
	}
	// Fail-on-status (opt-in): a >=400 response becomes a typed error while the
	// live response is still returned for the caller to inspect.
	if ferr := failOnStatus(resp, opts); ferr != nil {
		return resp, ferr
	}
	return resp, nil
}

// Execute runs a request described by a typed *options.RequestOptions and returns
// the live response. It is the programmatic counterpart to the curl-string Curl*
// helpers: use it when you build the request with the options builder
// (options.NewRequestOptionsBuilder().…​.Build()) or by populating
// options.RequestOptions directly, rather than from a pasted curl command.
//
// The response body is streamed and is subject to opts.ResponseBodyLimit; the
// caller owns reading and closing resp.Body. With opts.FailOnError set, a >=400
// status is returned as a typed error alongside the still-inspectable response
// (see failOnStatus). For a reusable, middleware-configured client, prefer
// Client.Prepare/Do; Execute is the one-shot path and shares its engine.
func Execute(ctx context.Context, opts *options.RequestOptions) (*http.Response, error) {
	if opts == nil {
		return nil, ValidationError("options", fmt.Errorf("request options cannot be nil"))
	}
	return executeOpts(ctx, opts)
}

// CurlArgs executes a curl command from variadic arguments.
// Each argument is a separate token (like os.Args).
// Environment variables are automatically expanded.
func CurlArgs(ctx context.Context, args ...string) (*http.Response, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no arguments provided")
	}

	// Convert args to tokens
	tokens := make([]tokenizer.Token, len(args))
	for i, arg := range args {
		tokens[i] = tokenizer.Token{Type: tokenizer.TokenValue, Value: arg}
	}

	// Auto-expand environment variables
	tokens = expandEnvInTokens(tokens)

	// Convert to request options
	opts, err := convertTokensToRequestOptions(tokens)
	if err != nil {
		return nil, parseOrPassthrough(strings.Join(args, " "), err)
	}

	// Execute
	return executeOpts(ctx, opts)
}

// ============================================================================
// EXPLICIT VARIABLE CONTROL
// ============================================================================

// CurlWithVars executes a curl command with explicit variable map.
// Variables are NOT expanded from environment, only from the provided map.
// Useful for testing and security-critical code.
func CurlWithVars(ctx context.Context, vars Variables, command ...string) (*http.Response, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("no command provided")
	}

	if len(command) == 1 {
		return CurlCommandWithVars(ctx, vars, command[0])
	}
	return CurlArgsWithVars(ctx, vars, command...)
}

// CurlCommandWithVars executes a shell-style command with explicit variables
func CurlCommandWithVars(ctx context.Context, vars Variables, command string) (*http.Response, error) {
	processed := preprocessMultilineCommand(command)

	tok := tokenizer.NewTokenizer()
	if err := tok.Tokenize(processed); err != nil {
		return nil, parseOrPassthrough(command, err)
	}

	tokens := tok.GetTokens()
	tokens = expandVarsInTokens(tokens, vars)

	opts, err := convertTokensToRequestOptions(tokens)
	if err != nil {
		return nil, parseOrPassthrough(command, err)
	}

	return executeOpts(ctx, opts)
}

// CurlArgsWithVars executes variadic arguments with explicit variables
func CurlArgsWithVars(ctx context.Context, vars Variables, args ...string) (*http.Response, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no arguments provided")
	}

	tokens := make([]tokenizer.Token, len(args))
	for i, arg := range args {
		tokens[i] = tokenizer.Token{Type: tokenizer.TokenValue, Value: arg}
	}

	tokens = expandVarsInTokens(tokens, vars)

	opts, err := convertTokensToRequestOptions(tokens)
	if err != nil {
		return nil, parseOrPassthrough(strings.Join(args, " "), err)
	}

	return executeOpts(ctx, opts)
}

// ============================================================================
// CONVENIENCE FUNCTIONS - Auto-read body
// ============================================================================

// CurlString executes request and returns body as string + response
func CurlString(ctx context.Context, command ...string) (string, *http.Response, error) {
	resp, err := Curl(ctx, command...)
	if err != nil {
		return "", resp, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp, fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), resp, nil
}

// CurlStringCommand executes shell command and returns body as string + response
func CurlStringCommand(ctx context.Context, command string) (string, *http.Response, error) {
	resp, err := CurlCommand(ctx, command)
	if err != nil {
		return "", resp, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp, fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), resp, nil
}

// CurlStringArgs executes variadic args and returns body as string + response
func CurlStringArgs(ctx context.Context, args ...string) (string, *http.Response, error) {
	resp, err := CurlArgs(ctx, args...)
	if err != nil {
		return "", resp, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp, fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), resp, nil
}

// CurlBytes executes request and returns body as []byte + response
func CurlBytes(ctx context.Context, command ...string) ([]byte, *http.Response, error) {
	resp, err := Curl(ctx, command...)
	if err != nil {
		return nil, resp, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, resp, nil
}

// CurlBytesCommand executes shell command and returns body as []byte + response
func CurlBytesCommand(ctx context.Context, command string) ([]byte, *http.Response, error) {
	resp, err := CurlCommand(ctx, command)
	if err != nil {
		return nil, resp, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, resp, nil
}

// CurlBytesArgs executes variadic args and returns body as []byte + response
func CurlBytesArgs(ctx context.Context, args ...string) ([]byte, *http.Response, error) {
	resp, err := CurlArgs(ctx, args...)
	if err != nil {
		return nil, resp, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, resp, nil
}

// CurlJSON executes request and decodes JSON response into provided struct
func CurlJSON(ctx context.Context, v interface{}, command ...string) (*http.Response, error) {
	resp, err := Curl(ctx, command...)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return resp, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return resp, nil
}

// CurlJSONCommand executes shell command and decodes JSON response
func CurlJSONCommand(ctx context.Context, v interface{}, command string) (*http.Response, error) {
	resp, err := CurlCommand(ctx, command)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return resp, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return resp, nil
}

// CurlJSONArgs executes variadic args and decodes JSON response
func CurlJSONArgs(ctx context.Context, v interface{}, args ...string) (*http.Response, error) {
	resp, err := CurlArgs(ctx, args...)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return resp, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return resp, nil
}

// CurlDownload executes request and saves body to file, returns bytes written + response
func CurlDownload(ctx context.Context, filepath string, command ...string) (int64, *http.Response, error) {
	resp, err := Curl(ctx, command...)
	if err != nil {
		return 0, resp, err
	}
	defer resp.Body.Close()

	file, err := os.Create(filepath)
	if err != nil {
		return 0, resp, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return written, resp, fmt.Errorf("failed to write to file: %w", err)
	}

	return written, resp, nil
}

// CurlDownloadCommand executes shell command and saves body to file
func CurlDownloadCommand(ctx context.Context, filepath string, command string) (int64, *http.Response, error) {
	resp, err := CurlCommand(ctx, command)
	if err != nil {
		return 0, resp, err
	}
	defer resp.Body.Close()

	file, err := os.Create(filepath)
	if err != nil {
		return 0, resp, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return written, resp, fmt.Errorf("failed to write to file: %w", err)
	}

	return written, resp, nil
}

// CurlDownloadArgs executes variadic args and saves body to file
func CurlDownloadArgs(ctx context.Context, filepath string, args ...string) (int64, *http.Response, error) {
	resp, err := CurlArgs(ctx, args...)
	if err != nil {
		return 0, resp, err
	}
	defer resp.Body.Close()

	file, err := os.Create(filepath)
	if err != nil {
		return 0, resp, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return written, resp, fmt.Errorf("failed to write to file: %w", err)
	}

	return written, resp, nil
}

// ============================================================================
// BACKWARD COMPATIBILITY (Deprecated - use Curl functions instead)
// ============================================================================

// NOTE: the former Request()/RequestWithContext()/Execute() helpers and the
// Response wrapper were removed — the name Request is now the prepared-request
// type (see request.go). Use Curl/CurlWithVars for the one-shot API (CurlString/
// CurlBytes/CurlJSON for buffered bodies), or a Client + Prepare/Do for the
// reusable API.

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// preprocessMultilineCommand handles multi-line curl commands
// Supports:
// - Backslash line continuations (\)
// - Newlines as whitespace
// - Comment lines (#)
// - Automatic 'curl' prefix removal
// - Leading/trailing whitespace trimming
func preprocessMultilineCommand(cmd string) string {
	lines := strings.Split(cmd, "\n")
	processed := []string{}

	for _, line := range lines {
		// Trim leading/trailing whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip comment lines
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Remove 'curl' prefix from first non-empty line
		if len(processed) == 0 && strings.HasPrefix(line, "curl ") {
			line = strings.TrimPrefix(line, "curl ")
			line = strings.TrimSpace(line)
		} else if len(processed) == 0 && line == "curl" {
			continue
		}

		// Handle backslash continuation
		if strings.HasSuffix(line, "\\") {
			line = strings.TrimSuffix(line, "\\")
			line = strings.TrimSpace(line)
		}

		if line != "" {
			processed = append(processed, line)
		}
	}

	return strings.Join(processed, " ")
}

// expandEnvInTokens expands environment variables in tokens
func expandEnvInTokens(tokens []tokenizer.Token) []tokenizer.Token {
	result := make([]tokenizer.Token, len(tokens))

	for i, token := range tokens {
		// Only expand STRING tokens (not flags - prevents injection)
		if token.Type == tokenizer.TokenValue && !strings.HasPrefix(token.Value, "-") {
			result[i] = tokenizer.Token{
				Type:  token.Type,
				Value: os.ExpandEnv(token.Value),
			}
		} else {
			result[i] = token
		}
	}

	return result
}

// expandVarsInTokens expands variables from explicit map in tokens
func expandVarsInTokens(tokens []tokenizer.Token, vars Variables) []tokenizer.Token {
	result := make([]tokenizer.Token, len(tokens))

	for i, token := range tokens {
		// Only expand STRING tokens (not flags)
		if token.Type == tokenizer.TokenValue && !strings.HasPrefix(token.Value, "-") {
			expanded, err := ExpandVariables(token.Value, vars)
			if err != nil {
				// If expansion fails, keep original value
				result[i] = token
			} else {
				result[i] = tokenizer.Token{
					Type:  token.Type,
					Value: expanded,
				}
			}
		} else {
			result[i] = token
		}
	}

	return result
}
