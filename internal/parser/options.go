// internal/parser/parser.go

package parser

import (
	"net/http"
	"net/url"
	"time"
)

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
	TLSClientConfig *TLSClientConfig

	// Proxy settings
	Proxy *url.URL

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
	Cookies []*http.Cookie

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
}

type BasicAuth struct {
	Username string
	Password string
}

type TLSClientConfig struct {
	InsecureSkipVerify bool
	CertFile           string
	KeyFile            string
	CAFile             string
}

type FileUpload struct {
	FieldName string
	FileName  string
	FilePath  string
}

type RetryConfig struct {
	MaxRetries  int
	RetryDelay  time.Duration
	RetryOnHTTP []int // HTTP status codes to retry on
}
