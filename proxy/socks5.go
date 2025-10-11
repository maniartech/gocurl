package proxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// SOCKS5Proxy represents a SOCKS5 proxy configuration.
type SOCKS5Proxy struct {
	Address  string
	Username string
	Password string
	Dialer   proxy.Dialer  // Allows injecting a custom dialer, useful for testing.
	Timeout  time.Duration // This is your custom timeout for the SOCKS5 connection
	NoProxy  []string      // Domains to exclude from proxying
}

// Apply configures the transport to use a SOCKS5 proxy.
func (sp *SOCKS5Proxy) Apply(transport *http.Transport) error {
	if sp.Address == "" {
		return fmt.Errorf("SOCKS5 proxy address is required")
	}

	var auth *proxy.Auth
	if sp.Username != "" && sp.Password != "" {
		auth = &proxy.Auth{
			User:     sp.Username,
			Password: sp.Password,
		}
	}

	// If Dialer is not provided, use proxy.Direct
	if sp.Dialer == nil {
		sp.Dialer = proxy.Direct
	}

	// Create SOCKS5 proxy dialer
	socksDialer, err := proxy.SOCKS5("tcp", sp.Address, auth, sp.Dialer)
	if err != nil {
		return fmt.Errorf("failed to create SOCKS5 proxy dialer: %v", err)
	}

	// Set the transport's DialContext to use the socksDialer with manual timeout handling
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		// Check if this address should bypass proxy
		// Construct a URL for checking (SOCKS5 is used for any protocol)
		testURL := "http://" + addr
		if ShouldBypassProxy(testURL, sp.NoProxy) {
			// Use direct connection
			dialer := &net.Dialer{
				Timeout:   sp.Timeout,
				KeepAlive: 30 * time.Second,
			}
			return dialer.DialContext(ctx, network, addr)
		}

		// Apply the custom timeout from sp.Timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, sp.Timeout)
		defer cancel()

		dialerChan := make(chan net.Conn, 1)
		errChan := make(chan error, 1)

		// Start the dial operation in a separate goroutine
		go func() {
			conn, err := socksDialer.Dial(network, addr)
			if err != nil {
				errChan <- err
				return
			}
			dialerChan <- conn
		}()

		// Wait for either the dial to complete, the context to timeout, or an error to occur
		select {
		case <-timeoutCtx.Done(): // If the custom context times out or is canceled
			return nil, timeoutCtx.Err() // Return the context's error (timeout or cancellation)
		case conn := <-dialerChan: // Successful connection
			return conn, nil
		case err := <-errChan: // Error during dialing
			return nil, err
		}
	}

	// Also set Proxy function to handle no-proxy for HTTP requests
	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		// For SOCKS5, we handle bypass in DialContext
		// Return the SOCKS5 URL (though it won't be used directly)
		return url.Parse("socks5://" + sp.Address)
	}

	return nil
}
