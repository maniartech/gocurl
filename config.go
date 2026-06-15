package gocurl

import (
	"fmt"
	"net/http"
	"time"

	"github.com/maniartech/gocurl/options"
)

// config holds the connection-level and default settings of a Client. It is
// assembled from functional Options at New() time and is never mutated
// afterwards, so a Client is safe for concurrent use.
//
// See specs/01-client-api.md.
type config struct {
	timeout        time.Duration // overall per-request ceiling (0 = none; ctx still applies)
	connectTimeout time.Duration

	followRedirects bool
	maxRedirects    int

	proxy    string
	insecure bool

	userAgent      string
	defaultHeaders http.Header
	cookieFile     string

	// transport, when set via WithTransport, overrides the Client's owned
	// transport (advanced / testing — e.g. an httptest or mock RoundTripper).
	transport http.RoundTripper

	// middleware are user middlewares appended to the chain via WithMiddleware.
	middleware []Middleware
}

// defaultConfig returns the baseline configuration. These defaults intentionally
// mirror curl/NewRequestOptions behavior: no client timeout, redirects NOT
// followed unless enabled, no proxy, verification on.
func defaultConfig() *config {
	return &config{
		followRedirects: false,
		maxRedirects:    0,
		defaultHeaders:  http.Header{},
	}
}

// baseOptions projects the connection-relevant config onto a RequestOptions used
// to build the Client's owned transport (TLS/proxy/HTTP2).
func (c *config) baseOptions() *options.RequestOptions {
	o := options.NewRequestOptions("")
	o.Proxy = c.proxy
	o.Insecure = c.insecure
	o.ConnectTimeout = c.connectTimeout
	return o
}

// Option configures a Client. Options are applied in order by New().
type Option func(*config) error

// WithTimeout sets the overall per-request timeout ceiling (http.Client.Timeout).
// A per-request --max-time or context deadline may impose a shorter bound.
func WithTimeout(d time.Duration) Option {
	return func(c *config) error {
		if d < 0 {
			return fmt.Errorf("WithTimeout: duration cannot be negative")
		}
		c.timeout = d
		return nil
	}
}

// WithConnectTimeout sets the connection (dial) timeout.
func WithConnectTimeout(d time.Duration) Option {
	return func(c *config) error {
		if d < 0 {
			return fmt.Errorf("WithConnectTimeout: duration cannot be negative")
		}
		c.connectTimeout = d
		return nil
	}
}

// WithFollowRedirects enables/disables following redirects for requests that do
// not specify their own policy (e.g. a curl command without -L).
func WithFollowRedirects(follow bool) Option {
	return func(c *config) error {
		c.followRedirects = follow
		if follow && c.maxRedirects == 0 {
			c.maxRedirects = 30
		}
		return nil
	}
}

// WithMaxRedirects sets the maximum number of redirects to follow.
func WithMaxRedirects(n int) Option {
	return func(c *config) error {
		if n < 0 {
			return fmt.Errorf("WithMaxRedirects: count cannot be negative")
		}
		c.maxRedirects = n
		return nil
	}
}

// WithProxy sets a proxy URL (http://, https://, or socks5://) for all requests.
func WithProxy(proxyURL string) Option {
	return func(c *config) error {
		c.proxy = proxyURL
		return nil
	}
}

// WithInsecure disables TLS certificate verification (curl -k). Use only for
// testing; it is unsafe in production.
func WithInsecure(insecure bool) Option {
	return func(c *config) error {
		c.insecure = insecure
		return nil
	}
}

// WithUserAgent sets a default User-Agent for requests that do not set their own.
func WithUserAgent(ua string) Option {
	return func(c *config) error {
		c.userAgent = ua
		return nil
	}
}

// WithDefaultHeader adds a default header applied to every request that does not
// already set that header.
func WithDefaultHeader(key, value string) Option {
	return func(c *config) error {
		if key == "" {
			return fmt.Errorf("WithDefaultHeader: key cannot be empty")
		}
		if c.defaultHeaders == nil {
			c.defaultHeaders = http.Header{}
		}
		c.defaultHeaders.Add(key, value)
		return nil
	}
}

// WithCookieFile sets a cookie jar file used by the Client (curl -b/-c).
func WithCookieFile(path string) Option {
	return func(c *config) error {
		c.cookieFile = path
		return nil
	}
}

// WithMiddleware appends user middleware to the Client's chain. The first
// middleware passed is the outermost.
func WithMiddleware(mw ...Middleware) Option {
	return func(c *config) error {
		c.middleware = append(c.middleware, mw...)
		return nil
	}
}

// WithTransport overrides the Client's owned transport with a custom
// http.RoundTripper (advanced; useful for tests, mocks, or a hand-tuned
// transport). When set, TLS/proxy options that build a transport are ignored.
func WithTransport(rt http.RoundTripper) Option {
	return func(c *config) error {
		if rt == nil {
			return fmt.Errorf("WithTransport: round tripper cannot be nil")
		}
		c.transport = rt
		return nil
	}
}
