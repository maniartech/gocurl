package gocurl_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/options"
)

// TestTLSVersionControl demonstrates TLS version control functionality
func TestTLSVersionControl(t *testing.T) {
	// Create options with TLS 1.2 minimum
	opts := options.NewRequestOptions("https://www.howsmyssl.com/a/check")
	opts.TLSMinVersion = tls.VersionTLS12
	opts.TLSMaxVersion = tls.VersionTLS13

	// Execute request
	ctx := context.Background()
	resp, err := gocurl.Execute(ctx, opts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	t.Logf("✅ TLS version control test passed - Status: %d", resp.StatusCode)
}

// TestCipherSuiteSelection demonstrates cipher suite selection
func TestCipherSuiteSelection(t *testing.T) {
	// Test cipher suite parsing
	cipherSuites, err := gocurl.ParseCipherSuites("ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256")
	if err != nil {
		t.Fatalf("Failed to parse cipher suites: %v", err)
	}

	if len(cipherSuites) != 2 {
		t.Errorf("Expected 2 cipher suites, got %d", len(cipherSuites))
	}

	// Create options with specific cipher suites
	opts := options.NewRequestOptions("https://www.howsmyssl.com/a/check")
	opts.CipherSuites = cipherSuites
	opts.TLSMinVersion = tls.VersionTLS12

	// Execute request
	ctx := context.Background()
	resp, err := gocurl.Execute(ctx, opts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	t.Logf("✅ Cipher suite selection test passed - Status: %d", resp.StatusCode)
}

// TestTLS13CipherSuites demonstrates TLS 1.3 cipher selection
func TestTLS13CipherSuites(t *testing.T) {
	// Test TLS 1.3 cipher parsing
	tls13Suites, err := gocurl.ParseTLS13CipherSuites("TLS_AES_256_GCM_SHA384:TLS_AES_128_GCM_SHA256")
	if err != nil {
		t.Fatalf("Failed to parse TLS 1.3 cipher suites: %v", err)
	}

	if len(tls13Suites) != 2 {
		t.Errorf("Expected 2 TLS 1.3 cipher suites, got %d", len(tls13Suites))
	}

	t.Logf("✅ TLS 1.3 cipher suite parsing test passed - Parsed %d suites", len(tls13Suites))
}

// TestProxyTLSConfiguration demonstrates proxy TLS field initialization
func TestProxyTLSConfiguration(t *testing.T) {
	// Create options with proxy TLS settings
	opts := options.NewRequestOptions("https://api.github.com")
	opts.Proxy = "https://proxy.example.com:8080"
	opts.ProxyCert = "/path/to/client.pem"
	opts.ProxyKey = "/path/to/client-key.pem"
	opts.ProxyCACert = "/path/to/ca.pem"
	opts.ProxyInsecure = false

	// Verify fields are set correctly
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

	t.Log("✅ Proxy TLS configuration test passed - All fields set correctly")
}

// TestTLSVersionParsing tests TLS version string parsing
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
			got, err := gocurl.ParseTLSVersion(tt.version)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseTLSVersion(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}

	t.Log("✅ TLS version parsing test passed - All versions parsed correctly")
}
