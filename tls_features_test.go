package gocurl

import (
	"crypto/tls"
	"testing"

	"github.com/maniartech/gocurl/options"
)

// These are whitebox tests for the internal TLS-flag parsers. The former
// live-network checks against www.howsmyssl.com were removed: gocurl's TLS
// version/cipher application is verified hermetically by tls_enhancements_test.go
// (LoadTLSConfig asserts MinVersion/MaxVersion/CipherSuites on the built config),
// and a real TLS handshake is exercised by security_pinning_test.go via
// httptest.NewTLSServer.

func TestCipherSuiteSelection(t *testing.T) {
	cipherSuites, err := ParseCipherSuites("ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256")
	if err != nil {
		t.Fatalf("Failed to parse cipher suites: %v", err)
	}
	if len(cipherSuites) != 2 {
		t.Errorf("Expected 2 cipher suites, got %d", len(cipherSuites))
	}
}

func TestTLS13CipherSuites(t *testing.T) {
	tls13Suites, err := ParseTLS13CipherSuites("TLS_AES_256_GCM_SHA384:TLS_AES_128_GCM_SHA256")
	if err != nil {
		t.Fatalf("Failed to parse TLS 1.3 cipher suites: %v", err)
	}
	if len(tls13Suites) != 2 {
		t.Errorf("Expected 2 TLS 1.3 cipher suites, got %d", len(tls13Suites))
	}
}

func TestProxyTLSConfiguration(t *testing.T) {
	opts := options.NewRequestOptions("https://api.github.com")
	opts.Proxy = "https://proxy.example.com:8080"
	opts.ProxyCert = "/path/to/client.pem"
	opts.ProxyKey = "/path/to/client-key.pem"
	opts.ProxyCACert = "/path/to/ca.pem"
	opts.ProxyInsecure = false

	if opts.ProxyCert != "/path/to/client.pem" {
		t.Errorf("ProxyCert not set correctly")
	}
	if opts.ProxyKey != "/path/to/client-key.pem" {
		t.Errorf("ProxyKey not set correctly")
	}
	if opts.ProxyCACert != "/path/to/ca.pem" {
		t.Errorf("ProxyCACert not set correctly")
	}
	if opts.ProxyInsecure != false {
		t.Errorf("ProxyInsecure not set correctly")
	}
}

func TestTLSVersionParsing(t *testing.T) {
	tests := []struct {
		version string
		want    uint16
	}{
		{"1.0", tls.VersionTLS10},
		{"1.1", tls.VersionTLS11},
		{"1.2", tls.VersionTLS12},
		{"1.3", tls.VersionTLS13},
	}
	for _, tt := range tests {
		t.Run("TLS_"+tt.version, func(t *testing.T) {
			got, err := ParseTLSVersion(tt.version)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseTLSVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
