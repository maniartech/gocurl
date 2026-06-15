package gocurl

import (
	"crypto/tls"
	"fmt"
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
	tlsConfig, err := LoadTLSConfig(opts)
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

	ConfigureCompressionForTransport(t, opts.Compress)

	if opts.HTTP2 {
		if err := http2.ConfigureTransport(t); err != nil {
			return nil, fmt.Errorf("failed to configure HTTP/2: %v", err)
		}
	}
	return t, nil
}

// transportKey builds a cache key from the connection-relevant options.
func transportKey(opts *options.RequestOptions) string {
	return fmt.Sprintf("ins=%t|min=%d|max=%d|ca=%s|cert=%s|key=%s|sni=%s|pins=%v|h2=%t|gz=%t",
		opts.Insecure, opts.TLSMinVersion, opts.TLSMaxVersion,
		opts.CAFile, opts.CertFile, opts.KeyFile, opts.SNIServerName,
		opts.CertPinFingerprints, opts.HTTP2, opts.Compress)
}
