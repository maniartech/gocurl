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

	// redirectAllow is an optional per-redirect authorization hook (see
	// RedirectPolicy.Allow); it composes with follow/max.
	redirectAllow func(req *http.Request, via []*http.Request) error

	// Transport tuning (applied to the Client's owned transport).
	maxIdleConns          int
	maxIdleConnsPerHost   int
	maxConnsPerHost       int // 0 = unlimited (net/http semantics)
	idleConnTimeout       time.Duration
	tlsHandshakeTimeout   time.Duration
	responseHeaderTimeout time.Duration
	expectContinueTimeout time.Duration

	// HTTP version selection.
	http2     bool // attempt HTTP/2 upgrade over TLS (ForceAttemptHTTP2)
	http2Only bool // use HTTP/2 exclusively (h2 / h2c)
	h2c       bool // allow HTTP/2 cleartext when http2Only

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

		// Connection-pool and timeout defaults (Spec 03 default table).
		maxIdleConns:          100,
		maxIdleConnsPerHost:   10,
		maxConnsPerHost:       0, // unlimited
		idleConnTimeout:       90 * time.Second,
		tlsHandshakeTimeout:   10 * time.Second,
		expectContinueTimeout: 1 * time.Second,
		http2:                 true,
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

// RedirectPolicy controls how the Client follows HTTP redirects. Allow, if set,
// is an additional per-redirect authorization hook (e.g. an SSRF guard, Spec 07)
// that runs after the follow/max checks.
type RedirectPolicy struct {
	Follow bool
	Max    int
	Allow  func(req *http.Request, via []*http.Request) error
}

// WithRedirectPolicy sets the redirect policy. If Follow is true and Max is 0,
// Max defaults to 30 (curl-like).
func WithRedirectPolicy(p RedirectPolicy) Option {
	return func(c *config) error {
		if p.Max < 0 {
			return fmt.Errorf("WithRedirectPolicy: Max cannot be negative")
		}
		c.followRedirects = p.Follow
		c.maxRedirects = p.Max
		if p.Follow && c.maxRedirects == 0 {
			c.maxRedirects = 30
		}
		c.redirectAllow = p.Allow
		return nil
	}
}

// WithMaxIdleConns sets the maximum number of idle (keep-alive) connections
// across all hosts. 0 means no limit.
func WithMaxIdleConns(n int) Option {
	return func(c *config) error {
		if n < 0 {
			return fmt.Errorf("WithMaxIdleConns: cannot be negative")
		}
		c.maxIdleConns = n
		return nil
	}
}

// WithMaxIdleConnsPerHost sets the maximum idle connections kept per host.
func WithMaxIdleConnsPerHost(n int) Option {
	return func(c *config) error {
		if n < 0 {
			return fmt.Errorf("WithMaxIdleConnsPerHost: cannot be negative")
		}
		c.maxIdleConnsPerHost = n
		return nil
	}
}

// WithMaxConnsPerHost limits the total connections per host (dialing blocks
// when reached). 0 means unlimited (net/http semantics).
func WithMaxConnsPerHost(n int) Option {
	return func(c *config) error {
		if n < 0 {
			return fmt.Errorf("WithMaxConnsPerHost: cannot be negative")
		}
		c.maxConnsPerHost = n
		return nil
	}
}

// WithIdleConnTimeout sets how long an idle connection is kept before closing.
func WithIdleConnTimeout(d time.Duration) Option {
	return func(c *config) error {
		if d < 0 {
			return fmt.Errorf("WithIdleConnTimeout: cannot be negative")
		}
		c.idleConnTimeout = d
		return nil
	}
}

// WithTLSHandshakeTimeout bounds the TLS handshake duration.
func WithTLSHandshakeTimeout(d time.Duration) Option {
	return func(c *config) error {
		if d < 0 {
			return fmt.Errorf("WithTLSHandshakeTimeout: cannot be negative")
		}
		c.tlsHandshakeTimeout = d
		return nil
	}
}

// WithResponseHeaderTimeout bounds the wait for response headers after the
// request is written. 0 means no timeout.
func WithResponseHeaderTimeout(d time.Duration) Option {
	return func(c *config) error {
		if d < 0 {
			return fmt.Errorf("WithResponseHeaderTimeout: cannot be negative")
		}
		c.responseHeaderTimeout = d
		return nil
	}
}

// WithExpectContinueTimeout bounds the wait for a 100-continue response.
func WithExpectContinueTimeout(d time.Duration) Option {
	return func(c *config) error {
		if d < 0 {
			return fmt.Errorf("WithExpectContinueTimeout: cannot be negative")
		}
		c.expectContinueTimeout = d
		return nil
	}
}

// WithHTTP2 enables or disables HTTP/2 upgrade over TLS (default enabled).
func WithHTTP2(enabled bool) Option {
	return func(c *config) error {
		c.http2 = enabled
		return nil
	}
}

// WithHTTP2Only forces HTTP/2 exclusively. If allowCleartext is true, HTTP/2
// cleartext (h2c) is permitted for non-TLS targets.
//
// HTTP/3 (QUIC) is intentionally out of scope; it is a documented future add-on.
func WithHTTP2Only(allowCleartext bool) Option {
	return func(c *config) error {
		c.http2Only = true
		c.h2c = allowCleartext
		return nil
	}
}
