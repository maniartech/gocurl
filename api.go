package gocurl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/maniartech/gocurl/options"
	"github.com/maniartech/gocurl/tokenizer"
)

// Variables type for safe variable substitution
type Variables map[string]string

// Response wraps http.Response with convenience methods
type Response struct {
	*http.Response
	bodyBytes []byte
	bodyRead  bool
}

// Request executes a curl command with optional variable substitution
// Accepts both string and []string formats
func Request(command interface{}, vars Variables) (*Response, error) {
	return RequestWithContext(context.Background(), command, vars)
}

// RequestWithContext executes a curl command with context and optional variable substitution
// Accepts both string and []string formats
func RequestWithContext(ctx context.Context, command interface{}, vars Variables) (*Response, error) {
	var opts *options.RequestOptions
	var err error

	switch cmd := command.(type) {
	case string:
		// Parse string command using tokenizer
		tok := tokenizer.NewTokenizer()
		err = tok.Tokenize(cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to tokenize command: %w", err)
		}

		tokens := tok.GetTokens()

		// Apply variable substitution if provided
		if vars != nil {
			tokens, err = substituteVariablesInTokens(tokens, vars)
			if err != nil {
				return nil, fmt.Errorf("variable substitution failed: %w", err)
			}
		}

		opts, err = convertTokensToRequestOptions(tokens)
		if err != nil {
			return nil, fmt.Errorf("failed to parse command: %w", err)
		}

	case []string:
		// Convert string slice to tokens
		tokens := make([]tokenizer.Token, len(cmd))
		for i, arg := range cmd {
			tokens[i] = tokenizer.Token{Type: tokenizer.TokenValue, Value: arg}
		}

		// Apply variable substitution if provided
		if vars != nil {
			tokens, err = substituteVariablesInTokens(tokens, vars)
			if err != nil {
				return nil, fmt.Errorf("variable substitution failed: %w", err)
			}
		}

		opts, err = convertTokensToRequestOptions(tokens)
		if err != nil {
			return nil, fmt.Errorf("failed to convert arguments: %w", err)
		}

	default:
		return nil, fmt.Errorf("command must be string or []string, got %T", command)
	}

	return Execute(ctx, opts)
}

// Execute runs a request with pre-built RequestOptions
func Execute(ctx context.Context, opts *options.RequestOptions) (*Response, error) {
	// Use the existing Process function
	httpResp, _, err := Process(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &Response{
		Response: httpResp,
	}, nil
}

// String returns the response body as a string
func (r *Response) String() (string, error) {
	b, err := r.Bytes()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Bytes reads the response body once and caches it
// Uses pooled buffers for responses <1MB for efficiency
func (r *Response) Bytes() ([]byte, error) {
	if r.bodyRead {
		return r.bodyBytes, nil
	}

	if r.Response == nil || r.Response.Body == nil {
		return nil, fmt.Errorf("response body is nil")
	}

	// Use smart response reading with pooling
	data, err := readResponseBody(r.Response)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	r.bodyBytes = data
	r.bodyRead = true
	return r.bodyBytes, nil
}

// JSON unmarshals the response body into the provided struct
func (r *Response) JSON(v interface{}) error {
	data, err := r.Bytes()
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// substituteVariablesInTokens applies variable substitution to tokens
func substituteVariablesInTokens(tokens []tokenizer.Token, vars Variables) ([]tokenizer.Token, error) {
	result := make([]tokenizer.Token, len(tokens))

	for i, token := range tokens {
		// For now, use simple substitution
		// This will be enhanced in variables.go
		substituted, err := ExpandVariables(token.Value, vars)
		if err != nil {
			return nil, err
		}

		result[i] = tokenizer.Token{
			Type:  token.Type,
			Value: substituted,
		}
	}

	return result, nil
}

// Get is a convenience function for making GET requests
func Get(ctx context.Context, url string, vars Variables) (*Response, error) {
	cmd := fmt.Sprintf("curl -X GET %s", url)
	return RequestWithContext(ctx, cmd, vars)
}

// Post is a convenience function for making POST requests with a body
// The body can be a string, []byte, or any type that can be JSON-marshaled
func Post(ctx context.Context, url string, body interface{}, vars Variables) (*Response, error) {
	var bodyStr string

	switch v := body.(type) {
	case string:
		bodyStr = v
	case []byte:
		bodyStr = string(v)
	default:
		// Try to marshal as JSON
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body as JSON: %w", err)
		}
		bodyStr = string(data)
	}

	cmd := fmt.Sprintf("curl -X POST -d %q %s", bodyStr, url)
	return RequestWithContext(ctx, cmd, vars)
}

// Put is a convenience function for making PUT requests with a body
func Put(ctx context.Context, url string, body interface{}, vars Variables) (*Response, error) {
	var bodyStr string

	switch v := body.(type) {
	case string:
		bodyStr = v
	case []byte:
		bodyStr = string(v)
	default:
		// Try to marshal as JSON
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body as JSON: %w", err)
		}
		bodyStr = string(data)
	}

	cmd := fmt.Sprintf("curl -X PUT -d %q %s", bodyStr, url)
	return RequestWithContext(ctx, cmd, vars)
}

// Delete is a convenience function for making DELETE requests
func Delete(ctx context.Context, url string, vars Variables) (*Response, error) {
	cmd := fmt.Sprintf("curl -X DELETE %s", url)
	return RequestWithContext(ctx, cmd, vars)
}

// Patch is a convenience function for making PATCH requests with a body
func Patch(ctx context.Context, url string, body interface{}, vars Variables) (*Response, error) {
	var bodyStr string

	switch v := body.(type) {
	case string:
		bodyStr = v
	case []byte:
		bodyStr = string(v)
	default:
		// Try to marshal as JSON
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body as JSON: %w", err)
		}
		bodyStr = string(data)
	}

	cmd := fmt.Sprintf("curl -X PATCH -d %q %s", bodyStr, url)
	return RequestWithContext(ctx, cmd, vars)
}

// Head is a convenience function for making HEAD requests
func Head(ctx context.Context, url string, vars Variables) (*Response, error) {
	cmd := fmt.Sprintf("curl -X HEAD %s", url)
	return RequestWithContext(ctx, cmd, vars)
}
