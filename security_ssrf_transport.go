package gocurl

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type ssrfDialPinKey struct{}

type ssrfDialPin struct {
	host string
	ips  []net.IP
}

type unsupportedPinningRoundTripper struct {
	next http.RoundTripper
}

func (rt unsupportedPinningRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if _, ok := req.Context().Value(ssrfDialPinKey{}).(ssrfDialPin); ok {
		// An arbitrary RoundTripper may resolve, proxy, tunnel, or dial internally;
		// there is no general API through which gocurl can force its approved IP.
		// Refusing guarded requests is safer than silently reverting to TOCTOU.
		return nil, ssrfError(req.URL.Hostname(), "transport cannot enforce SSRF dial pinning")
	}
	return rt.next.RoundTrip(req)
}

// withSSRFDialPinning clones a standard transport and makes its dial consume
// the IPs validated by SSRFGuard. The request URL is not rewritten, so net/http
// continues to use the original hostname for Host routing and TLS SNI.
func withSSRFDialPinning(rt http.RoundTripper) http.RoundTripper {
	t, ok := rt.(*http.Transport)
	// Custom TLS dial callbacks own both connection establishment and TLS setup.
	// Wrapping them with an IP address could change SNI or allow an internal second
	// lookup, so treat them like any other opaque transport and fail closed only
	// when an SSRF pin is actually present.
	//lint:ignore SA1019 DialTLS is deprecated for callers but must still be detected; wrapping a legacy custom TLS dialer would bypass the pin.
	if !ok || t.DialTLS != nil || t.DialTLSContext != nil {
		return unsupportedPinningRoundTripper{next: rt}
	}

	// Clone once when the Client/handler chain is constructed. Mutating a transport
	// after it has served requests is unsafe, while a one-time clone preserves
	// connection pooling and all of the caller's tuning without per-request clones.
	clone := t.Clone()
	baseDial := clone.DialContext
	if baseDial == nil {
		baseDial = (&net.Dialer{}).DialContext
	}
	clone.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		pin, ok := ctx.Value(ssrfDialPinKey{}).(ssrfDialPin)
		if !ok {
			return baseDial(ctx, network, address)
		}
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("SSRF dial target %q: %w", address, err)
		}
		if !strings.EqualFold(hostOnly(host), hostOnly(pin.host)) {
			// The common mismatch is a proxy address. Connecting to the proxy would
			// leave origin resolution outside gocurl's control, so guarded proxy paths
			// fail closed rather than claiming rebinding protection they do not have.
			return nil, ssrfError(pin.host, "transport attempted to dial an unvalidated destination")
		}

		// Preserve resolver order and try every already-approved address. This keeps
		// multi-address destinations available without performing another hostname
		// lookup; context cancellation stops fallback promptly.
		var lastErr error
		for _, ip := range pin.ips {
			conn, dialErr := baseDial(ctx, network, net.JoinHostPort(ip.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
			if ctx.Err() != nil {
				break
			}
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("validated destination has no IP addresses")
		}
		return nil, lastErr
	}
	return clone
}
