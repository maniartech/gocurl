package proxy

import (
	"crypto/tls"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

// ProxyType represents the type of proxy to use.
type ProxyType string

const (
	ProxyTypeNone   ProxyType = "NONE"
	ProxyTypeHTTP   ProxyType = "HTTP"
	ProxyTypeSOCKS5 ProxyType = "SOCKS5"
)

// ProxyConfig holds the configuration for the proxy.// ProxyConfig holds the configuration for the proxy.
type ProxyConfig struct {
	Type              ProxyType
	Address           string        // e.g., "127.0.0.1:8080" for HTTP or "127.0.0.1:1080" for SOCKS5
	Username          string        // Optional: For proxies that require authentication
	Password          string        // Optional: For proxies that require authentication
	Timeout           time.Duration // Timeout for connections
	TLSConfig         *tls.Config   // Optional: For custom TLS settings
	DisableKeepAlives bool          // Optional: To disable HTTP keep-alives
	CustomDialer      proxy.Dialer  // Optional: Allows injecting a custom dialer (useful for testing)
}

// Proxy defines the interface for different proxy types.
type Proxy interface {
	// Apply configures the http.Transport with proxy settings.
	Apply(*http.Transport) error
}
