package gocurl

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

// RequestOptions represents the configuration for an HTTP request in GoCurl.
type RequestOptions struct {
	// HTTP request basics
	Method      string
	URL         string
	Headers     http.Header
	Body        string
	Form        url.Values
	QueryParams url.Values

	// Authentication
	BasicAuth   *BasicAuth
	BearerToken string

	// TLS/SSL options
	CertFile  string
	KeyFile   string
	CAFile    string
	Insecure  bool
	TLSConfig *tls.Config

	// Proxy settings
	Proxy string

	// Timeout settings
	Timeout        time.Duration
	ConnectTimeout time.Duration

	// Redirect behavior
	FollowRedirects bool
	MaxRedirects    int

	// Compression
	Compress bool

	// HTTP version specific
	HTTP2     bool
	HTTP2Only bool

	// Cookie handling
	Cookies   []*http.Cookie
	CookieJar http.CookieJar

	// Custom options
	UserAgent string
	Referer   string

	// File upload
	FileUpload *FileUpload

	// Retry configuration
	RetryConfig *RetryConfig

	// Output options
	OutputFile string
	Silent     bool
	Verbose    bool

	// Advanced options
	Context           context.Context
	RequestID         string
	Middleware        []MiddlewareFunc
	ResponseBodyLimit int64
	ResponseDecoder   ResponseDecoder
	Metrics           *RequestMetrics
}

// BasicAuth represents HTTP Basic Authentication credentials.
type BasicAuth struct {
	Username string
	Password string
}

// FileUpload represents a file to be uploaded in a multipart form.
type FileUpload struct {
	FieldName string
	FileName  string
	FilePath  string
}

// RetryConfig represents the configuration for request retries.
type RetryConfig struct {
	MaxRetries  int
	RetryDelay  time.Duration
	RetryOnHTTP []int // HTTP status codes to retry on
}

// MiddlewareFunc is a function type for request middleware.
type MiddlewareFunc func(*http.Request) (*http.Request, error)

// ResponseDecoder is a function type for custom response decoding.
type ResponseDecoder func(*http.Response) (interface{}, error)

// RequestMetrics represents metrics collected during a request.
type RequestMetrics struct {
	StartTime    time.Time
	Duration     time.Duration
	RetryCount   int
	ResponseSize int64
}

// NewRequestOptions creates a new RequestOptions with default values.
// NewRequestOptions creates a new RequestOptions with default values aligned to cURL's defaults.
func NewRequestOptions() *RequestOptions {
	return &RequestOptions{
		Headers:         make(http.Header),
		Form:            make(url.Values),
		QueryParams:     make(url.Values),
		FollowRedirects: false, // cURL does not follow redirects by default
		MaxRedirects:    0,     // No redirects followed unless -L is used
		Compress:        false, // Compression not enabled by default
	}
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
