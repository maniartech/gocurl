package proxy

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// HTTPProxy represents an HTTP proxy configuration.
type HTTPProxy struct {
	Address  string
	Username string
	Password string

	Timeout time.Duration
}

// Apply configures the transport to use an HTTP proxy.
func (hp *HTTPProxy) Apply(transport *http.Transport) error {
	proxyURLStr := fmt.Sprintf("http://%s", hp.Address)
	if hp.Username != "" && hp.Password != "" {
		proxyURLStr = fmt.Sprintf("http://%s:%s@%s", url.QueryEscape(hp.Username), url.QueryEscape(hp.Password), hp.Address)
	}

	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		return fmt.Errorf("failed to parse HTTP proxy URL: %v", err)
	}

	// Set the proxy
	transport.Proxy = http.ProxyURL(proxyURL)

	// Set the dialer with timeout
	dialer := &net.Dialer{
		Timeout:   hp.Timeout, // Timeout from config
		KeepAlive: 30 * time.Second,
	}
	transport.DialContext = dialer.DialContext

	return nil
}
