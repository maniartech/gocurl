package proxy

import (
	"fmt"
	"net/http"
	"time"
)

// NewProxy creates a Proxy implementation based on the provided ProxyConfig.
func NewProxy(config ProxyConfig) (Proxy, error) {
	switch config.Type {
	case ProxyTypeNone:
		return &NoProxy{}, nil
	case ProxyTypeHTTP:
		return &HTTPProxy{
			Address:   config.Address,
			Username:  config.Username,
			Password:  config.Password,
			Timeout:   config.Timeout,
			TLSConfig: config.TLSConfig,
			NoProxy:   config.NoProxy,
		}, nil
	case ProxyTypeSOCKS5:
		return &SOCKS5Proxy{
			Address:  config.Address,
			Username: config.Username,
			Password: config.Password,
			Dialer:   config.CustomDialer, // Allows injecting a custom dialer if needed
			Timeout:  config.Timeout,
			NoProxy:  config.NoProxy,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", config.Type)
	}
}

// NewTransport creates a new http.Transport based on the provided ProxyConfig.
func NewTransport(config ProxyConfig) (*http.Transport, error) {
	transport := &http.Transport{
		TLSClientConfig:       config.TLSConfig,
		DisableKeepAlives:     config.DisableKeepAlives,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Create the appropriate Proxy implementation
	proxyImpl, err := NewProxy(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy: %v", err)
	}

	// Apply the proxy settings to the transport
	if err := proxyImpl.Apply(transport); err != nil {
		return nil, fmt.Errorf("failed to apply proxy settings: %v", err)
	}

	return transport, nil
}
