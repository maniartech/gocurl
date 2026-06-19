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

// CheckSSRF resolves host and rejects the request if any resolved IP is blocked
// by the policy and not on the allow-list. A resolution failure is NOT a policy
// block (it surfaces later as a normal dial error). host may be "host" or
// "host:port" (bracketed IPv6 accepted).
func (p SSRFPolicy) CheckSSRF(ctx context.Context, host string) error {
	host = hostOnly(host)
	if host == "" {
		return nil
	}

	// Allow-list by host name takes precedence over every block.
	for _, a := range p.AllowHosts {
		if strings.EqualFold(hostOnly(a), host) {
			return nil
		}
	}

	// Known cloud-metadata hostnames are blocked by name (they may resolve to a
	// non-link-local address, or not resolve at all in the test environment).
	if p.BlockCloudMetadata && blockedMetadataHostnames[strings.ToLower(host)] {
		return ssrfError(host, "cloud metadata host")
	}

	ips, err := resolveIPs(ctx, host)
	if err != nil {
		return nil // not a policy decision; let the dial fail naturally
	}
	for _, ip := range ips {
		if p.ipAllowed(ip) {
			continue
		}
		if reason := p.blockReason(ip); reason != "" {
			return ssrfError(host, reason)
		}
	}
	return nil
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
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
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

// SSRFGuard returns a Middleware that runs the SSRF policy pre-flight check on
// the initial request before it leaves the chain.
func SSRFGuard(policy SSRFPolicy) Middleware {
	return func(next Handler) Handler {
		return func(req *http.Request) (*http.Response, error) {
			if req.URL != nil {
				if err := policy.CheckSSRF(req.Context(), req.URL.Host); err != nil {
					return nil, err
				}
			}
			return next(req)
		}
	}
}
