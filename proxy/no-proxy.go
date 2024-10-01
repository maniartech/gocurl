package proxy

import (
	"net"
	"net/http"
	"time"
)

// NoProxy represents a direct connection without any proxy.
type NoProxy struct {
	Timeout time.Duration
}

// Apply configures the transport for a direct connection.
func (np *NoProxy) Apply(transport *http.Transport) error {
	transport.Proxy = nil

	// Configure DialContext with a timeout.
	dialer := &net.Dialer{
		Timeout:   np.Timeout,
		KeepAlive: 30 * time.Second,
	}
	transport.DialContext = dialer.DialContext

	return nil
}
