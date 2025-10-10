package gocurl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	return Execute(opts)
}

// Execute runs a request with pre-built RequestOptions
func Execute(opts *options.RequestOptions) (*Response, error) {
	// Use context.Background if not provided
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

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
func (r *Response) Bytes() ([]byte, error) {
	if r.bodyRead {
		return r.bodyBytes, nil
	}

	if r.Response == nil || r.Response.Body == nil {
		return nil, fmt.Errorf("response body is nil")
	}

	defer r.Response.Body.Close()

	data, err := io.ReadAll(r.Response.Body)
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
