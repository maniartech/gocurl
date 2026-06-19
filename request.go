package gocurl

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/maniartech/gocurl/options"
	"github.com/maniartech/gocurl/tokenizer"
)

// Request is an immutable, prepared request template — the "parse once" artifact
// of the parse-once/execute-many model. Build it from a curl command with
// Client.Prepare* or programmatically with NewRequest, then execute it many
// times with Client.Do. Builder methods (WithHeader, WithQuery, …) return a
// modified COPY, so a Request is safe to share across goroutines.
//
// See specs/02-request-model-and-curl-compat.md.
type Request struct {
	opts       *options.RequestOptions
	rawCommand string // the original curl command if built via Prepare*, else ""

	// retryPolicy, when set via WithRetryPolicy, overrides the Client's default
	// RetryPolicy for this request. It lives on the gocurl Request (not on
	// options.RequestOptions) so the options package never imports gocurl.
	retryPolicy *RetryPolicy
}

// Options returns a deep copy of the underlying request options. The returned
// value may be modified freely without affecting the Request.
func (r *Request) Options() *options.RequestOptions { return r.opts.Clone() }

// Method returns the HTTP method of the prepared request.
func (r *Request) Method() string { return r.opts.Method }

// URL returns the URL of the prepared request.
func (r *Request) URL() string { return r.opts.URL }

// parseCommand runs the shared tokenize -> (optional expand) -> convert pipeline.
func parseCommand(command string, expand func([]tokenizer.Token) []tokenizer.Token) (*options.RequestOptions, error) {
	processed := preprocessMultilineCommand(command)
	tok := tokenizer.NewTokenizer()
	if err := tok.Tokenize(processed); err != nil {
		return nil, fmt.Errorf("failed to tokenize command: %w", err)
	}
	tokens := tok.GetTokens()
	if expand != nil {
		tokens = expand(tokens)
	}
	opts, err := convertTokensToRequestOptions(tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}
	return opts, nil
}

// Prepare parses a curl command ONCE into a reusable Request. Environment
// variables ($VAR/${VAR}) are expanded, mirroring CurlCommand.
func (c *Client) Prepare(command string) (*Request, error) {
	opts, err := parseCommand(command, expandEnvInTokens)
	if err != nil {
		return nil, err
	}
	return &Request{opts: opts, rawCommand: command}, nil
}

// PrepareNoEnv parses a curl command without expanding environment variables.
func (c *Client) PrepareNoEnv(command string) (*Request, error) {
	opts, err := parseCommand(command, nil)
	if err != nil {
		return nil, err
	}
	return &Request{opts: opts, rawCommand: command}, nil
}

// PrepareWithVars parses a curl command, expanding $VAR/${VAR} ONLY from the
// supplied map (never the process environment).
func (c *Client) PrepareWithVars(vars Variables, command string) (*Request, error) {
	opts, err := parseCommand(command, func(t []tokenizer.Token) []tokenizer.Token {
		return expandVarsInTokens(t, vars)
	})
	if err != nil {
		return nil, err
	}
	return &Request{opts: opts, rawCommand: command}, nil
}

// RequestOption configures a Request built by NewRequest.
type RequestOption func(*options.RequestOptions) error

// NewRequest builds a Request programmatically (no curl parsing). The URL is
// normalized (a missing scheme defaults to http://, matching the curl path).
func NewRequest(method, rawURL string, opts ...RequestOption) (*Request, error) {
	o := options.NewRequestOptions(rawURL)
	o.Method = method
	if o.Method == "" {
		o.Method = "GET"
	}
	o.Headers = http.Header{}
	if err := normalizeURL(o); err != nil {
		return nil, err
	}
	for _, opt := range opts {
		if err := opt(o); err != nil {
			return nil, err
		}
	}
	return &Request{opts: o}, nil
}

// Header sets a header on a NewRequest.
func Header(key, value string) RequestOption {
	return func(o *options.RequestOptions) error {
		if key == "" {
			return fmt.Errorf("Header: key cannot be empty")
		}
		o.Headers.Add(key, value)
		return nil
	}
}

// Query sets a query parameter on a NewRequest.
func Query(key, value string) RequestOption {
	return func(o *options.RequestOptions) error {
		if o.QueryParams == nil {
			o.QueryParams = url.Values{}
		}
		o.QueryParams.Add(key, value)
		return nil
	}
}

// Body sets a raw request body on a NewRequest.
func Body(b []byte) RequestOption {
	return func(o *options.RequestOptions) error {
		o.Body = string(b)
		return nil
	}
}

// BodyReader sets a request body from a reader. It is buffered; for true
// streaming (and retry replay) use Stream with a BodySource.
func BodyReader(r io.Reader) RequestOption {
	return func(o *options.RequestOptions) error {
		data, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("BodyReader: %w", err)
		}
		o.Body = string(data)
		return nil
	}
}

// Stream sets a streaming BodySource on a NewRequest (e.g. FileBody, ReaderBody,
// MultipartBody), overriding any raw body.
func Stream(src options.BodySource) RequestOption {
	return func(o *options.RequestOptions) error {
		o.BodyStream = src
		o.Body = ""
		return nil
	}
}

// --- Immutable builder methods (each returns a modified copy) ---

// WithHeader returns a copy of the request with the header set (replacing any
// existing values for that key).
func (r *Request) WithHeader(key, value string) *Request {
	c := r.clone()
	if c.opts.Headers == nil {
		c.opts.Headers = http.Header{}
	}
	c.opts.Headers.Set(key, value)
	return c
}

// AddHeader returns a copy of the request with the header value appended.
func (r *Request) AddHeader(key, value string) *Request {
	c := r.clone()
	if c.opts.Headers == nil {
		c.opts.Headers = http.Header{}
	}
	c.opts.Headers.Add(key, value)
	return c
}

// WithQuery returns a copy of the request with the query parameter set.
func (r *Request) WithQuery(key, value string) *Request {
	c := r.clone()
	if c.opts.QueryParams == nil {
		c.opts.QueryParams = url.Values{}
	}
	c.opts.QueryParams.Set(key, value)
	return c
}

// WithBody returns a copy of the request with the given raw body.
func (r *Request) WithBody(b []byte) *Request {
	c := r.clone()
	c.opts.Body = string(b)
	c.opts.BodyStream = nil
	return c
}

// WithBodySource returns a copy of the request with a streaming body source,
// overriding any raw body.
func (r *Request) WithBodySource(src options.BodySource) *Request {
	c := r.clone()
	c.opts.BodyStream = src
	c.opts.Body = ""
	return c
}

// WithBodyFile returns a copy of the request that streams its body from a file.
func (r *Request) WithBodyFile(path string) *Request {
	return r.WithBodySource(FileBody(path))
}

// WithVars returns a copy of the request re-parsed with the given variables.
// It only applies to requests built from a curl command (Prepare*); for a
// programmatic NewRequest it returns an unchanged copy.
func (r *Request) WithVars(vars Variables) *Request {
	if r.rawCommand == "" {
		return r.clone()
	}
	opts, err := parseCommand(r.rawCommand, func(t []tokenizer.Token) []tokenizer.Token {
		return expandVarsInTokens(t, vars)
	})
	if err != nil {
		// Re-parse should not fail (it parsed once already); keep original on error.
		return r.clone()
	}
	return &Request{opts: opts, rawCommand: r.rawCommand, retryPolicy: r.retryPolicy}
}

// WithRetryPolicy returns a copy of the request with a per-request RetryPolicy
// that overrides the Client's default for this request only.
func (r *Request) WithRetryPolicy(p RetryPolicy) *Request {
	c := r.clone()
	c.retryPolicy = &p
	return c
}

// Clone returns an independent copy of the request.
func (r *Request) Clone() *Request { return r.clone() }

func (r *Request) clone() *Request {
	return &Request{opts: r.opts.Clone(), rawCommand: r.rawCommand, retryPolicy: r.retryPolicy}
}
