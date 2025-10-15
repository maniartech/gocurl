package proxy

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NoProxy represents a direct connection without any proxy.
type NoProxy struct {
}

// Apply configures the transport for a direct connection.
func (np *NoProxy) Apply(transport *http.Transport) error {
	transport.Proxy = nil

	// Configure DialContext with a timeout.
	dialer := &net.Dialer{
		KeepAlive: 30 * time.Second,
	}
	transport.DialContext = dialer.DialContext

	return nil
}

// ShouldBypassProxy determines if a given URL should bypass the proxy based on no-proxy rules.
// This function implements curl-compatible no-proxy matching logic:
// - Exact domain match: "example.com" matches "example.com"
// - Subdomain match: ".example.com" matches "*.example.com"
// - IP address match: "192.168.1.1" matches exactly
// - CIDR match: "192.168.1.0/24" matches IP range
// - Port-specific: "example.com:8080" matches only that port
func ShouldBypassProxy(targetURL string, noProxyList []string) bool {
	if len(noProxyList) == 0 {
		return false
	}

	host, port, err := parseTargetURL(targetURL)
	if err != nil {
		return false
	}

	for _, pattern := range noProxyList {
		if shouldBypassForPattern(host, port, pattern) {
			return true
		}
	}

	return false
}

// parseTargetURL extracts host and port from target URL
func parseTargetURL(targetURL string) (host, port string, err error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return "", "", err
	}
	return parsedURL.Hostname(), parsedURL.Port(), nil
}

// shouldBypassForPattern checks if host/port matches a single no-proxy pattern
func shouldBypassForPattern(host, port, pattern string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}

	// Handle wildcard "*" - bypass all
	if pattern == "*" {
		return true
	}

	// Extract port from pattern if present
	patternHost, patternPort, hasPort := splitHostPort(pattern)
	if !hasPort {
		patternHost = pattern
	}

	// If pattern has a port, it must match
	if hasPort && patternPort != port {
		return false
	}

	// Check different matching strategies
	return matchesCIDR(host, patternHost) ||
		matchesDomainPattern(host, patternHost)
}

// matchesCIDR checks if host matches CIDR pattern
func matchesCIDR(host, patternHost string) bool {
	if !strings.Contains(patternHost, "/") {
		return false
	}
	return matchCIDR(host, patternHost)
}

// matchesDomainPattern checks if host matches domain pattern
func matchesDomainPattern(host, patternHost string) bool {
	// Leading dot means "this domain and all subdomains"
	if strings.HasPrefix(patternHost, ".") {
		domain := patternHost[1:] // Remove leading dot
		return host == domain || strings.HasSuffix(host, patternHost)
	}

	// Exact match or subdomain match
	return host == patternHost || strings.HasSuffix(host, "."+patternHost)
}

// splitHostPort splits a host:port string, handling IPv6 addresses correctly
func splitHostPort(hostPort string) (host, port string, hasPort bool) {
	// Handle IPv6 addresses
	if strings.HasPrefix(hostPort, "[") {
		// IPv6 with port: [::1]:8080
		if closeBracket := strings.Index(hostPort, "]"); closeBracket != -1 {
			host = hostPort[1:closeBracket]
			if len(hostPort) > closeBracket+1 && hostPort[closeBracket+1] == ':' {
				port = hostPort[closeBracket+2:]
				hasPort = true
			}
			return
		}
	}

	// Regular host:port or just host
	if colonIndex := strings.LastIndex(hostPort, ":"); colonIndex != -1 {
		// Check if this might be an IPv6 address without brackets
		if strings.Count(hostPort, ":") > 1 {
			// Likely IPv6 without port
			host = hostPort
			hasPort = false
			return
		}
		host = hostPort[:colonIndex]
		port = hostPort[colonIndex+1:]
		hasPort = true
		return
	}

	host = hostPort
	hasPort = false
	return
}

// matchCIDR checks if an IP address or hostname matches a CIDR pattern
func matchCIDR(host, cidr string) bool {
	// Parse the CIDR
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}

	// Try to parse host as IP
	ip := net.ParseIP(host)
	if ip == nil {
		// If host is a domain name, try to resolve it
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return false
		}
		// Check first resolved IP
		ip = ips[0]
	}

	return ipNet.Contains(ip)
}
