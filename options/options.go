package options

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/maniartech/gocurl/middlewares"
)

// RequestOptions represents the configuration for an HTTP request in GoCurl.
type RequestOptions struct {
	// HTTP request basics
	Method      string      `json:"method"`
	URL         string      `json:"url"`
	Headers     http.Header `json:"headers"`
	Body        string      `json:"body"`
	Form        url.Values  `json:"form"`
	QueryParams url.Values  `json:"query_params"`

	// Authentication
	BasicAuth   *BasicAuth `json:"basic_auth,omitempty"`
	BearerToken string     `json:"bearer_token,omitempty"`

	// TLS/SSL options
	CertFile  string      `json:"cert_file,omitempty"`
	KeyFile   string      `json:"key_file,omitempty"`
	CAFile    string      `json:"ca_file,omitempty"`
	Insecure  bool        `json:"insecure,omitempty"`
	TLSConfig *tls.Config `json:"-"` // Not exported to JSON

	// Proxy settings
	Proxy string `json:"proxy,omitempty"`

	// Timeout settings
	Timeout        time.Duration `json:"timeout,omitempty"`
	ConnectTimeout time.Duration `json:"connect_timeout,omitempty"`

	// Redirect behavior
	FollowRedirects bool `json:"follow_redirects,omitempty"`
	MaxRedirects    int  `json:"max_redirects,omitempty"`

	// Compression
	Compress bool `json:"compress,omitempty"`

	// HTTP version specific
	HTTP2     bool `json:"http2,omitempty"`
	HTTP2Only bool `json:"http2_only,omitempty"`

	// Cookie handling
	Cookies   []*http.Cookie `json:"cookies,omitempty"`
	CookieJar http.CookieJar `json:"-"` // Not exported to JSON

	// Custom options
	UserAgent string `json:"user_agent,omitempty"`
	Referer   string `json:"referer,omitempty"`

	// File upload
	FileUpload *FileUpload `json:"file_upload,omitempty"`

	// Retry configuration
	RetryConfig *RetryConfig `json:"retry_config,omitempty"`

	// Output options
	OutputFile string `json:"output_file,omitempty"`
	Silent     bool   `json:"silent,omitempty"`
	Verbose    bool   `json:"verbose,omitempty"`

	// Advanced options
	Context           context.Context              `json:"-"` // Not exported to JSON
	RequestID         string                       `json:"request_id,omitempty"`
	Middleware        []middlewares.MiddlewareFunc `json:"-"`
	ResponseBodyLimit int64                        `json:"response_body_limit,omitempty"`
	ResponseDecoder   ResponseDecoder              `json:"-"`
	Metrics           *RequestMetrics              `json:"metrics,omitempty"`
}

// NewRequestOptions creates a new RequestOptions with default values aligned to cURL's defaults.
func NewRequestOptions(url string) *RequestOptions {
	return &RequestOptions{
		URL: url,
		// Headers:         make(http.Header),
		// Form:            make(url.Values),
		// QueryParams:     make(url.Values),
		QueryParams:     nil,
		FollowRedirects: false, // cURL does not follow redirects by default
		MaxRedirects:    0,     // No redirects followed unless -L is used
		Compress:        false, // Compression not enabled by default
	}
}

// ToJSON marshals the RequestOptions struct to JSON format.
func (ro *RequestOptions) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(ro)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

// BasicAuth represents HTTP Basic Authentication credentials.
type BasicAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// FileUpload represents a file to be uploaded in a multipart form.
type FileUpload struct {
	FieldName string `json:"field_name"`
	FileName  string `json:"file_name"`
	FilePath  string `json:"file_path"`
}

// RetryConfig represents the configuration for request retries.
type RetryConfig struct {
	MaxRetries  int           `json:"max_retries"`
	RetryDelay  time.Duration `json:"retry_delay"`
	RetryOnHTTP []int         `json:"retry_on_http"`
}

// ResponseDecoder is a function type for custom response decoding.
type ResponseDecoder func(*http.Response) (interface{}, error)

// RequestMetrics represents metrics collected during a request.
type RequestMetrics struct {
	StartTime    time.Time     `json:"start_time"`
	Duration     time.Duration `json:"duration"`
	RetryCount   int           `json:"retry_count"`
	ResponseSize int64         `json:"response_size"`
}

// Clone creates a deep copy of RequestOptions.
func (ro *RequestOptions) Clone() *RequestOptions {
	clone := *ro
	clone.Headers = ro.Headers.Clone()

	// Deep copy Form
	clone.Form = make(url.Values)
	for k, v := range ro.Form {
		clone.Form[k] = append([]string(nil), v...)
	}

	// Deep copy QueryParams
	clone.QueryParams = make(url.Values)
	for k, v := range ro.QueryParams {
		clone.QueryParams[k] = append([]string(nil), v...)
	}

	// Deep copy other pointer fields as needed
	if ro.BasicAuth != nil {
		clonedBasicAuth := *ro.BasicAuth
		clone.BasicAuth = &clonedBasicAuth
	}

	if ro.FileUpload != nil {
		clonedFileUpload := *ro.FileUpload
		clone.FileUpload = &clonedFileUpload
	}

	if ro.RetryConfig != nil {
		clonedRetryConfig := *ro.RetryConfig
		clone.RetryConfig = &clonedRetryConfig
	}

	if ro.Metrics != nil {
		clonedMetrics := *ro.Metrics
		clone.Metrics = &clonedMetrics
	}

	// Note: We're not deep copying the Context, TLSConfig, CookieJar,
	// Middleware, or ResponseDecoder as these are typically shared or
	// would require more complex deep copying logic.

	return &clone
}

// SetBasicAuth sets the basic authentication credentials.
func (ro *RequestOptions) SetBasicAuth(username, password string) {
	ro.BasicAuth = &BasicAuth{
		Username: username,
		Password: password,
	}
}

// AddHeader adds a header to the request.
func (ro *RequestOptions) AddHeader(key, value string) {
	ro.Headers.Add(key, value)
}

// SetHeader sets a header in the request, replacing any existing values.
func (ro *RequestOptions) SetHeader(key, value string) {
	ro.Headers.Set(key, value)
}

// AddQueryParam adds a query parameter to the request URL.
func (ro *RequestOptions) AddQueryParam(key, value string) {
	ro.QueryParams.Add(key, value)
}

// SetQueryParam sets a query parameter in the request URL, replacing any existing values.
func (ro *RequestOptions) SetQueryParam(key, value string) {
	ro.QueryParams.Set(key, value)
}
