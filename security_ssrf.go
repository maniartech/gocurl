package gocurl

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// ErrSSRFBlocked is wrapped by every SSRF policy rejection so callers can match
// it with errors.Is. The wrapping GocurlError is classified KindValidation
// (non-retryable). See specs/07-security.md.
var ErrSSRFBlocked = errors.New("blocked by SSRF policy")

// SSRFPolicy controls which destinations a Client may reach. It is OPT-IN
// (WithSSRFGuard); the default Client enforces nothing, preserving the
// paste-any-curl promise. Enforcement happens at two points: a pre-flight
// middleware on the initial request, and a per-redirect check on every hop (so a
// public URL that 302s to an internal address is still blocked).
type SSRFPolicy struct {
	BlockLoopback      bool     // 127.0.0.0/8, ::1
	BlockLinkLocal     bool     // 169.254.0.0/16, fe80::/10
	BlockPrivate       bool     // RFC1918, fc00::/7 (ULA)
	BlockCloudMetadata bool     // 169.254.169.254, fd00:ec2::254, metadata.google.internal
	AllowHosts         []string // explicit allow-list (host or host:port), checked first
	AllowIPs           []string // explicit allow-list of IPs or CIDRs, checked first
}

// DefaultSSRFPolicy blocks loopback, link-local, private, and cloud-metadata
// destinations — the recommended setting for untrusted curl input.
func DefaultSSRFPolicy() SSRFPolicy {
	return SSRFPolicy{
		BlockLoopback:      true,
		BlockLinkLocal:     true,
		BlockPrivate:       true,
		BlockCloudMetadata: true,
	}
}

// blockedMetadataHostnames are blocked by name even before resolution.
var blockedMetadataHostnames = map[string]bool{
	"metadata.google.internal": true,
}

// lookupIPAddr is a narrow test seam for deterministic rebinding simulations.
// Production always uses net.DefaultResolver; keeping the seam at this lowest
// level ensures CheckSSRF and dial pinning cannot accidentally use two resolvers.
var lookupIPAddr = net.DefaultResolver.LookupIPAddr

// CheckSSRF resolves host and rejects the request if resolution fails or any
// resolved IP is blocked by the policy and not on the allow-list. host may be
// "host" or "host:port" (bracketed IPv6 accepted).
//
// CheckSSRF performs validation only. To prevent DNS rebinding, execute requests
// through SSRFGuard with HandlerFromRoundTripper, or use WithSSRFGuard on Client;
// those paths pin the subsequent dial to the IPs validated here.
func (p SSRFPolicy) CheckSSRF(ctx context.Context, host string) error {
	_, err := p.resolveAndCheck(ctx, host)
	return err
}

func (p SSRFPolicy) resolveAndCheck(ctx context.Context, host string) ([]net.IP, error) {
	host = hostOnly(host)
	if host == "" {
		return nil, nil
	}

	// Allow-list by host name takes precedence over every block.
	hostAllowed := false
	for _, a := range p.AllowHosts {
		if strings.EqualFold(hostOnly(a), host) {
			hostAllowed = true
			break
		}
	}

	// Known cloud-metadata hostnames are blocked by name (they may resolve to a
	// non-link-local address, or not resolve at all in the test environment).
	if !hostAllowed && p.BlockCloudMetadata && blockedMetadataHostnames[strings.ToLower(host)] {
		return nil, ssrfError(host, "cloud metadata host")
	}

	ips, err := resolveIPs(ctx, host)
	if err != nil {
		// Once pinning is enabled, continuing after a failed validation lookup
		// would necessarily make the transport resolve the hostname independently
		// and recreate the rebinding window. Fail closed instead.
		return nil, ssrfError(host, "destination could not be resolved")
	}
	if len(ips) == 0 {
		return nil, ssrfError(host, "destination resolved to no addresses")
	}
	if hostAllowed {
		return ips, nil
	}
	for _, ip := range ips {
		if p.ipAllowed(ip) {
			continue
		}
		if reason := p.blockReason(ip); reason != "" {
			return nil, ssrfError(host, reason)
		}
	}
	return ips, nil
}

// blockReason returns a non-empty reason when ip is blocked by the policy.
func (p SSRFPolicy) blockReason(ip net.IP) string {
	if p.BlockCloudMetadata && isCloudMetadataIP(ip) {
		return "cloud metadata endpoint"
	}
	// The unspecified address (0.0.0.0, ::, ::ffff:0.0.0.0) is loopback-equivalent
	// for routing: the OS dials a service bound on localhost. Treat it as loopback
	// so it cannot slip past an IsLoopback()-only filter.
	if p.BlockLoopback && ip.IsUnspecified() {
		return "unspecified address"
	}
	if p.BlockLoopback && ip.IsLoopback() {
		return "loopback address"
	}
	if p.BlockLinkLocal && (ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()) {
		return "link-local address"
	}
	if p.BlockPrivate && ip.IsPrivate() {
		return "private address"
	}
	return ""
}

// ipAllowed reports whether ip matches an AllowIPs entry (exact IP or CIDR).
func (p SSRFPolicy) ipAllowed(ip net.IP) bool {
	for _, a := range p.AllowIPs {
		if strings.Contains(a, "/") {
			if _, cidr, err := net.ParseCIDR(a); err == nil && cidr.Contains(ip) {
				return true
			}
			continue
		}
		if aip := net.ParseIP(a); aip != nil && aip.Equal(ip) {
			return true
		}
	}
	return false
}

// isCloudMetadataIP matches the well-known metadata service addresses.
func isCloudMetadataIP(ip net.IP) bool {
	return ip.Equal(net.ParseIP("169.254.169.254")) || ip.Equal(net.ParseIP("fd00:ec2::254"))
}

// hostOnly strips an optional port, IPv6 brackets, and a single trailing dot
// from a host[:port] string. The trailing dot is the FQDN root label: DNS treats
// "metadata.google.internal." identically to "metadata.google.internal", so it is
// stripped here to keep both the by-name block and AllowHosts matching consistent.
func hostOnly(host string) string {
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	return strings.TrimSuffix(host, ".")
}

// resolveIPs returns the literal IP (no DNS) or resolves the hostname using the
// default resolver, honoring ctx.
func resolveIPs(ctx context.Context, host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	addrs, err := lookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		ips = append(ips, a.IP)
	}
	return ips, nil
}

func ssrfError(host, reason string) error {
	return &GocurlError{
		Op:   "ssrf",
		Kind: KindValidation,
		URL:  host,
		Err:  fmt.Errorf("%s (%s): %w", reason, host, ErrSSRFBlocked),
	}
}

// SSRFGuard returns a Middleware that validates the destination and carries the
// approved IPs to the dial. Pair it with HandlerFromRoundTripper (or use
// WithSSRFGuard on Client) so the transport consumes the pin while preserving
// the original hostname for HTTP Host routing and TLS SNI. A raw custom Handler
// is responsible for honoring the pin itself.
func SSRFGuard(policy SSRFPolicy) Middleware {
	return func(next Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			if req == nil {
				return nil, ssrfError("", "nil request")
			}
			if err := policy.pinRequest(req); err != nil {
				return nil, err
			}
			return next(req)
		}
	}
}

func (p SSRFPolicy) pinRequest(req *http.Request) error {
	if req.URL == nil || req.URL.Host == "" {
		return nil
	}
	ips, err := p.resolveAndCheck(req.Context(), req.URL.Host)
	if err != nil {
		return err
	}
	// Preserve the URL and Host exactly as supplied. The transport reads this
	// context value only at dial time, which pins the socket address without
	// changing virtual-host routing or the TLS ServerName derived by net/http.
	pin := ssrfDialPin{host: req.URL.Hostname(), ips: cloneIPs(ips)}
	// Redirect requests are owned by net/http and handed to CheckRedirect as a
	// mutable pointer. Replacing the pointed-to value makes the new hop carry its
	// own pin while retaining all other request fields.
	*req = *req.WithContext(context.WithValue(req.Context(), ssrfDialPinKey{}, pin))
	return nil
}

func cloneIPs(ips []net.IP) []net.IP {
	// net.IP is a byte slice. Deep-copy it so resolver-owned backing arrays cannot
	// be reused or mutated while a concurrent dial is consuming the approved set.
	cloned := make([]net.IP, len(ips))
	for i, ip := range ips {
		cloned[i] = append(net.IP(nil), ip...)
	}
	return cloned
}
