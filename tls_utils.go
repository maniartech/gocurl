package gocurl

import (
	"crypto/tls"
	"fmt"
	"strings"
)

// Cipher suite name mapping (OpenSSL/curl names -> Go constants)
// Based on: https://golang.org/pkg/crypto/tls/#pkg-constants
var cipherSuiteMap = map[string]uint16{
	// TLS 1.2 - ECDHE with AES-GCM (recommended)
	"ECDHE-RSA-AES256-GCM-SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	"ECDHE-RSA-AES128-GCM-SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"ECDHE-ECDSA-AES256-GCM-SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	"ECDHE-ECDSA-AES128-GCM-SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,

	// TLS 1.2 - ECDHE with CHACHA20-POLY1305 (modern, mobile-friendly)
	"ECDHE-RSA-CHACHA20-POLY1305":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	"ECDHE-ECDSA-CHACHA20-POLY1305": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,

	// TLS 1.2 - ECDHE with AES-CBC (legacy compatibility)
	"ECDHE-RSA-AES256-CBC-SHA":   tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	"ECDHE-RSA-AES128-CBC-SHA":   tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	"ECDHE-ECDSA-AES256-CBC-SHA": tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	"ECDHE-ECDSA-AES128-CBC-SHA": tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,

	// TLS 1.2 - RSA (legacy, not recommended for new deployments)
	"AES256-GCM-SHA384": tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	"AES128-GCM-SHA256": tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	"AES256-CBC-SHA":    tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	"AES128-CBC-SHA":    tls.TLS_RSA_WITH_AES_128_CBC_SHA,
}

// TLS 1.3 cipher suite mapping
// Note: TLS 1.3 cipher suites cannot be configured in Go < 1.21
var tls13CipherSuiteMap = map[string]uint16{
	"TLS_AES_256_GCM_SHA384":       tls.TLS_AES_256_GCM_SHA384,
	"TLS_AES_128_GCM_SHA256":       tls.TLS_AES_128_GCM_SHA256,
	"TLS_CHACHA20_POLY1305_SHA256": tls.TLS_CHACHA20_POLY1305_SHA256,
}

// ParseCipherSuites parses colon-separated cipher suite names (curl format).
//
// Examples:
//
//	ParseCipherSuites("ECDHE-RSA-AES256-GCM-SHA384")
//	ParseCipherSuites("ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256")
//
// Returns error if any cipher suite name is unknown.
func ParseCipherSuites(cipherStr string) ([]uint16, error) {
	if cipherStr == "" {
		return nil, nil
	}

	names := strings.Split(cipherStr, ":")
	suites := make([]uint16, 0, len(names))

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		suite, ok := cipherSuiteMap[name]
		if !ok {
			return nil, fmt.Errorf("unknown cipher suite: %s (use openssl ciphers -v to see available ciphers)", name)
		}
		suites = append(suites, suite)
	}

	if len(suites) == 0 {
		return nil, fmt.Errorf("no valid cipher suites specified")
	}

	return suites, nil
}

// ParseTLS13CipherSuites parses TLS 1.3 cipher suite names (colon-separated).
//
// Examples:
//
//	ParseTLS13CipherSuites("TLS_AES_256_GCM_SHA384")
//	ParseTLS13CipherSuites("TLS_AES_256_GCM_SHA384:TLS_AES_128_GCM_SHA256")
//
// Returns error if any cipher suite name is unknown.
func ParseTLS13CipherSuites(cipherStr string) ([]uint16, error) {
	if cipherStr == "" {
		return nil, nil
	}

	names := strings.Split(cipherStr, ":")
	suites := make([]uint16, 0, len(names))

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		suite, ok := tls13CipherSuiteMap[name]
		if !ok {
			return nil, fmt.Errorf("unknown TLS 1.3 cipher suite: %s", name)
		}
		suites = append(suites, suite)
	}

	if len(suites) == 0 {
		return nil, fmt.Errorf("no valid TLS 1.3 cipher suites specified")
	}

	return suites, nil
}

// ParseTLSVersion parses a TLS version string to its uint16 constant.
//
// Supported formats:
//   - "1.0" -> tls.VersionTLS10
//   - "1.1" -> tls.VersionTLS11
//   - "1.2" -> tls.VersionTLS12
//   - "1.3" -> tls.VersionTLS13
//
// Returns error if the version string is invalid.
func ParseTLSVersion(version string) (uint16, error) {
	switch version {
	case "1.0":
		return tls.VersionTLS10, nil
	case "1.1":
		return tls.VersionTLS11, nil
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("invalid TLS version: %s (use 1.0, 1.1, 1.2, or 1.3)", version)
	}
}

// GetSupportedCipherSuites returns a list of all supported cipher suite names.
func GetSupportedCipherSuites() []string {
	suites := make([]string, 0, len(cipherSuiteMap))
	for name := range cipherSuiteMap {
		suites = append(suites, name)
	}
	return suites
}

// GetSupportedTLS13CipherSuites returns a list of all supported TLS 1.3 cipher suite names.
func GetSupportedTLS13CipherSuites() []string {
	suites := make([]string, 0, len(tls13CipherSuiteMap))
	for name := range tls13CipherSuiteMap {
		suites = append(suites, name)
	}
	return suites
}
