package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"
)

// NewProxy creates a Proxy implementation based on the provided ProxyConfig.
func NewProxy(config ProxyConfig) (Proxy, error) {
	switch config.Type {
	case ProxyTypeNone:
		return &NoProxy{}, nil
	case ProxyTypeHTTP:
		// Create proxy TLS config if HTTPS proxy with client cert
		proxyTLSConfig, err := createProxyTLSConfig(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy TLS config: %v", err)
		}

		return &HTTPProxy{
			Address:   config.Address,
			Username:  config.Username,
			Password:  config.Password,
			Timeout:   config.Timeout,
			TLSConfig: proxyTLSConfig,
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

// createProxyTLSConfig creates a TLS config for HTTPS proxy with client cert support.
// This implements curl's --proxy-cert, --proxy-key, --proxy-cacert, --proxy-insecure flags.
func createProxyTLSConfig(config ProxyConfig) (*tls.Config, error) {
	// Start with base TLS config or create new one
	tlsConfig := config.TLSConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{}
	} else {
		// Clone to avoid modifying original
		tlsConfig = tlsConfig.Clone()
	}

	// Apply --proxy-insecure
	if config.Insecure {
		tlsConfig.InsecureSkipVerify = true
	}

	// Load proxy client certificate (--proxy-cert, --proxy-key)
	if config.ClientCert != "" && config.ClientKey != "" {
		cert, err := tls.LoadX509KeyPair(config.ClientCert, config.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load proxy client certificate: %v", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load proxy CA certificate (--proxy-cacert)
	if config.CACert != "" {
		caCert, err := loadCACert(config.CACert)
		if err != nil {
			return nil, fmt.Errorf("failed to load proxy CA certificate: %v", err)
		}
		if tlsConfig.RootCAs == nil {
			tlsConfig.RootCAs = caCert
		} else {
			// Append to existing pool
			for _, cert := range caCert.Subjects() {
				tlsConfig.RootCAs.AppendCertsFromPEM(cert)
			}
		}
	}

	return tlsConfig, nil
}

// loadCACert loads a CA certificate from file and returns a cert pool.
func loadCACert(caFile string) (*x509.CertPool, error) {
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return caCertPool, nil
}
