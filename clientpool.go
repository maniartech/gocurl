package gocurl

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/maniartech/gocurl/options"
	"golang.org/x/net/http2"
)

// Connection pooling
//
// A fresh net/http transport per request would discard the idle-connection pool
// after every call, paying a TCP+TLS handshake each time. To reuse connections
// across the one-shot Curl* helpers, transports are cached and keyed by their
// connection-relevant configuration. http.Transport is safe for concurrent use
// by multiple goroutines, and each transport is fully configured once (under the
// cache lock) before being shared, so there is no per-request mutation.
var (
	transportMu    sync.Mutex
	transportCache = map[string]*http.Transport{}
)

// getRoundTripper returns the round tripper for opts, reusing a cached transport
// when the configuration is cacheable (no proxy, no opaque custom *tls.Config,
// not HTTP/2-only).
func getRoundTripper(opts *options.RequestOptions) (http.RoundTripper, error) {
	tlsConfig, err := loadTLSConfig(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}

	if opts.Proxy != "" {
		return createProxyTransport(opts, tlsConfig)
	}

	if opts.HTTP2Only {
		return &http2.Transport{TLSClientConfig: tlsConfig}, nil
	}

	// An opaque, user-supplied *tls.Config cannot be reliably used as a cache
	// key, so build a fresh (still idle-tuned) transport in that case.
	if opts.TLSConfig != nil {
		return newTransport(opts, tlsConfig)
	}

	key := transportKey(opts)
	transportMu.Lock()
	defer transportMu.Unlock()
	if t, ok := transportCache[key]; ok {
		return t, nil
	}
	t, err := newTransport(opts, tlsConfig)
	if err != nil {
		return nil, err
	}
	transportCache[key] = t
	return t, nil
}

// buildTransport builds the Client's OWNED round tripper from its config — so
// closing one Client never affects another. Transport tuning (idle conns,
// timeouts, max conns per host, HTTP version) comes from the config; TLS/proxy
// come from the projected base options.
func (c *config) buildTransport() (http.RoundTripper, error) {
	if c.transport != nil {
		return c.transport, nil // explicit WithTransport override
	}

	base := c.baseOptions()
	tlsConfig, err := loadTLSConfig(base)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}

	if c.proxy != "" {
		return createProxyTransport(base, tlsConfig)
	}

	if c.http2Only {
		return &http2.Transport{
			TLSClientConfig: tlsConfig,
			AllowHTTP:       c.h2c,
		}, nil
	}

	dialer := &net.Dialer{Timeout: c.connectTimeout, KeepAlive: 30 * time.Second}
	t := &http.Transport{
		TLSClientConfig:       tlsConfig,
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     c.http2,
		MaxIdleConns:          c.maxIdleConns,
		MaxIdleConnsPerHost:   c.maxIdleConnsPerHost,
		MaxConnsPerHost:       c.maxConnsPerHost,
		IdleConnTimeout:       c.idleConnTimeout,
		TLSHandshakeTimeout:   c.tlsHandshakeTimeout,
		ResponseHeaderTimeout: c.responseHeaderTimeout,
		ExpectContinueTimeout: c.expectContinueTimeout,
		// Decompress manually (curl semantics: only when --compressed) so the
		// Client's decompression path owns it.
		DisableCompression: true,
	}
	switch {
	case c.http11:
		forceHTTP11(t)
	case c.http2:
		if err := http2.ConfigureTransport(t); err != nil {
			return nil, fmt.Errorf("failed to configure HTTP/2: %w", err)
		}
	}
	return t, nil
}

// newTransport builds an idle-tuned *http.Transport, configuring HTTP/2 upgrade
// when requested.
func newTransport(opts *options.RequestOptions, tlsConfig *tls.Config) (*http.Transport, error) {
	t := &http.Transport{
		TLSClientConfig:       tlsConfig,
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	configureCompressionForTransport(t, opts.Compress)

	switch {
	case opts.HTTP11 || opts.HTTP10:
		forceHTTP11(t)
	case opts.HTTP2:
		if err := http2.ConfigureTransport(t); err != nil {
			return nil, fmt.Errorf("failed to configure HTTP/2: %v", err)
		}
	}
	return t, nil
}

// forceHTTP11 pins t to HTTP/1.1, suppressing HTTP/2 negotiation. The go.mod
// floor is go 1.23, where the http.Transport.Protocols API (Go 1.24+) is not
// available, so this uses the documented TLSNextProto mechanism: a non-nil EMPTY
// map disables the runtime's automatic h2, and we strip "h2" from any ALPN list a
// caller-supplied *tls.Config advertises so it cannot re-enable h2 over ALPN.
func forceHTTP11(t *http.Transport) {
	t.ForceAttemptHTTP2 = false
	t.TLSNextProto = map[string]func(authority string, c *tls.Conn) http.RoundTripper{}
	if t.TLSClientConfig != nil && len(t.TLSClientConfig.NextProtos) > 0 {
		kept := make([]string, 0, len(t.TLSClientConfig.NextProtos))
		for _, proto := range t.TLSClientConfig.NextProtos {
			if proto != "h2" {
				kept = append(kept, proto)
			}
		}
		t.TLSClientConfig.NextProtos = kept
	}
}

// transportKey builds a cache key from the connection-relevant options.
func transportKey(opts *options.RequestOptions) string {
	return fmt.Sprintf("ins=%t|min=%d|max=%d|ca=%s|cert=%s|key=%s|sni=%s|pins=%v|h2=%t|h11=%t|h10=%t|gz=%t",
		opts.Insecure, opts.TLSMinVersion, opts.TLSMaxVersion,
		opts.CAFile, opts.CertFile, opts.KeyFile, opts.SNIServerName,
		opts.CertPinFingerprints, opts.HTTP2, opts.HTTP11, opts.HTTP10, opts.Compress)
}
