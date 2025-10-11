package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// HTTPProxy represents an HTTP proxy configuration.
type HTTPProxy struct {
	Address   string
	Username  string
	Password  string
	Timeout   time.Duration
	TLSConfig *tls.Config // For HTTPS proxies
	NoProxy   []string    // Domains to exclude from proxying
}

// Apply configures the transport to use an HTTP proxy with CONNECT support for HTTPS.
func (hp *HTTPProxy) Apply(transport *http.Transport) error {
	proxyURLStr := fmt.Sprintf("http://%s", hp.Address)
	if hp.Username != "" && hp.Password != "" {
		proxyURLStr = fmt.Sprintf("http://%s:%s@%s", url.QueryEscape(hp.Username), url.QueryEscape(hp.Password), hp.Address)
	}

	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		return fmt.Errorf("failed to parse HTTP proxy URL: %v", err)
	}

	// Set the proxy with no-proxy support
	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		// Check if this URL should bypass proxy
		if ShouldBypassProxy(req.URL.String(), hp.NoProxy) {
			return nil, nil
		}
		return proxyURL, nil
	}

	// Set the dialer with timeout
	dialer := &net.Dialer{
		Timeout:   hp.Timeout,
		KeepAlive: 30 * time.Second,
	}

	// For HTTPS requests through HTTP proxy, we need custom CONNECT handling
	transport.DialContext = dialer.DialContext

	// Enable CONNECT for HTTPS tunneling
	// The standard library handles CONNECT automatically when using http.ProxyURL
	// but we provide custom DialTLS for additional control
	if hp.TLSConfig != nil {
		transport.TLSClientConfig = hp.TLSConfig
	}

	// Custom DialTLS for HTTPS through HTTP proxy using CONNECT
	transport.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		// First, connect to the proxy
		proxyConn, err := dialer.DialContext(ctx, "tcp", hp.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to proxy: %v", err)
		}

		// Send CONNECT request
		connectReq := &http.Request{
			Method: "CONNECT",
			URL:    &url.URL{Opaque: addr},
			Host:   addr,
			Header: make(http.Header),
		}

		// Add proxy authentication if needed
		if hp.Username != "" && hp.Password != "" {
			auth := hp.Username + ":" + hp.Password
			basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
			connectReq.Header.Set("Proxy-Authorization", basicAuth)
		}

		// Write CONNECT request
		if err := connectReq.Write(proxyConn); err != nil {
			proxyConn.Close()
			return nil, fmt.Errorf("failed to write CONNECT request: %v", err)
		}

		// Read CONNECT response
		br := bufio.NewReader(proxyConn)
		resp, err := http.ReadResponse(br, connectReq)
		if err != nil {
			proxyConn.Close()
			return nil, fmt.Errorf("failed to read CONNECT response: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			proxyConn.Close()
			return nil, fmt.Errorf("proxy CONNECT failed: %s", resp.Status)
		}

		// Now establish TLS over the tunneled connection
		tlsConfig := hp.TLSConfig
		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}

		// Clone to avoid modifying the original
		tlsConfig = tlsConfig.Clone()

		// Set ServerName for SNI if not already set
		if tlsConfig.ServerName == "" {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				host = addr
			}
			tlsConfig.ServerName = host
		}

		tlsConn := tls.Client(proxyConn, tlsConfig)

		// Perform TLS handshake with context timeout
		errChan := make(chan error, 1)
		go func() {
			errChan <- tlsConn.Handshake()
		}()

		select {
		case err := <-errChan:
			if err != nil {
				tlsConn.Close()
				return nil, fmt.Errorf("TLS handshake failed: %v", err)
			}
			return tlsConn, nil
		case <-ctx.Done():
			tlsConn.Close()
			return nil, ctx.Err()
		}
	}

	return nil
}
