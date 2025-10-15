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
	// Set up proxy URL and function
	if err := hp.configureProxyURL(transport); err != nil {
		return err
	}

	// Set up dialer
	dialer := hp.createDialer()
	transport.DialContext = dialer.DialContext

	// Configure TLS if needed
	hp.configureTLS(transport)

	// Set up custom TLS dialing for HTTPS through HTTP proxy
	transport.DialTLSContext = hp.createDialTLSContext(dialer)

	return nil
}

// configureProxyURL sets up the proxy URL and proxy function
func (hp *HTTPProxy) configureProxyURL(transport *http.Transport) error {
	proxyURLStr := hp.buildProxyURL()
	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		return fmt.Errorf("failed to parse HTTP proxy URL: %v", err)
	}

	// Set the proxy with no-proxy support
	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		if ShouldBypassProxy(req.URL.String(), hp.NoProxy) {
			return nil, nil
		}
		return proxyURL, nil
	}

	return nil
}

// buildProxyURL constructs the proxy URL string with credentials
func (hp *HTTPProxy) buildProxyURL() string {
	if hp.Username != "" && hp.Password != "" {
		return fmt.Sprintf("http://%s:%s@%s",
			url.QueryEscape(hp.Username),
			url.QueryEscape(hp.Password),
			hp.Address)
	}
	return fmt.Sprintf("http://%s", hp.Address)
}

// createDialer creates a network dialer with timeout
func (hp *HTTPProxy) createDialer() *net.Dialer {
	return &net.Dialer{
		Timeout:   hp.Timeout,
		KeepAlive: 30 * time.Second,
	}
}

// configureTLS sets TLS configuration if available
func (hp *HTTPProxy) configureTLS(transport *http.Transport) {
	if hp.TLSConfig != nil {
		transport.TLSClientConfig = hp.TLSConfig
	}
}

// createDialTLSContext creates a custom DialTLSContext for HTTPS through HTTP proxy
func (hp *HTTPProxy) createDialTLSContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		// Connect to proxy
		proxyConn, err := dialer.DialContext(ctx, "tcp", hp.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to proxy: %v", err)
		}

		// Send CONNECT request
		if err := hp.sendConnectRequest(proxyConn, addr); err != nil {
			proxyConn.Close()
			return nil, err
		}

		// Establish TLS over the tunnel
		return hp.establishTLS(ctx, proxyConn, addr)
	}
}

// sendConnectRequest sends HTTP CONNECT request to proxy
func (hp *HTTPProxy) sendConnectRequest(proxyConn net.Conn, addr string) error {
	connectReq := hp.createConnectRequest(addr)

	// Write CONNECT request
	if err := connectReq.Write(proxyConn); err != nil {
		return fmt.Errorf("failed to write CONNECT request: %v", err)
	}

	// Read and verify response
	return hp.verifyConnectResponse(proxyConn, connectReq)
}

// createConnectRequest creates HTTP CONNECT request
func (hp *HTTPProxy) createConnectRequest(addr string) *http.Request {
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

	return connectReq
}

// verifyConnectResponse reads and verifies CONNECT response
func (hp *HTTPProxy) verifyConnectResponse(proxyConn net.Conn, connectReq *http.Request) error {
	br := bufio.NewReader(proxyConn)
	resp, err := http.ReadResponse(br, connectReq)
	if err != nil {
		return fmt.Errorf("failed to read CONNECT response: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("proxy CONNECT failed: %s", resp.Status)
	}

	return nil
}

// establishTLS establishes TLS connection over proxy tunnel
func (hp *HTTPProxy) establishTLS(ctx context.Context, proxyConn net.Conn, addr string) (net.Conn, error) {
	tlsConfig := hp.prepareTLSConfig(addr)
	tlsConn := tls.Client(proxyConn, tlsConfig)

	// Perform TLS handshake with context
	if err := hp.performTLSHandshake(ctx, tlsConn); err != nil {
		tlsConn.Close()
		return nil, err
	}

	return tlsConn, nil
}

// prepareTLSConfig prepares TLS configuration for the connection
func (hp *HTTPProxy) prepareTLSConfig(addr string) *tls.Config {
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

	return tlsConfig
}

// performTLSHandshake performs TLS handshake with context timeout
func (hp *HTTPProxy) performTLSHandshake(ctx context.Context, tlsConn *tls.Conn) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- tlsConn.Handshake()
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("TLS handshake failed: %v", err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
