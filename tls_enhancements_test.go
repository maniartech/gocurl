package gocurl

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/maniartech/gocurl/options"
)

// TestTLSConfig_Clone verifies TLSConfig is cloned before use
func TestTLSConfig_Clone(t *testing.T) {
	// Create custom TLS config
	originalTLS := &tls.Config{
		MinVersion: tls.VersionTLS13,
		ServerName: "original.example.com",
	}

	opts := options.NewRequestOptions("https://example.com")
	opts.TLSConfig = originalTLS

	// Load TLS config (should clone it)
	loadedTLS, err := LoadTLSConfig(opts)
	if err != nil {
		t.Fatalf("LoadTLSConfig failed: %v", err)
	}

	// Verify it's a different object (cloned)
	if loadedTLS == originalTLS {
		t.Error("TLSConfig should be cloned, not the same reference")
	}

	// Modify loaded config
	loadedTLS.ServerName = "modified.example.com"

	// Verify original is unchanged
	if originalTLS.ServerName != "original.example.com" {
		t.Errorf("Original TLSConfig was modified after cloning. Expected 'original.example.com', got '%s'",
			originalTLS.ServerName)
	}
}

// TestTLSConfig_ClonePreservesSettings verifies cloned config preserves settings
func TestTLSConfig_ClonePreservesSettings(t *testing.T) {
	// Create custom TLS config with specific settings
	originalTLS := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		ServerName:   "test.example.com",
		CipherSuites: []uint16{tls.TLS_AES_256_GCM_SHA384},
	}

	opts := options.NewRequestOptions("https://example.com")
	opts.TLSConfig = originalTLS

	// Load TLS config
	loadedTLS, err := LoadTLSConfig(opts)
	if err != nil {
		t.Fatalf("LoadTLSConfig failed: %v", err)
	}

	// Verify settings are preserved
	if loadedTLS.MinVersion != originalTLS.MinVersion {
		t.Errorf("MinVersion not preserved. Expected %d, got %d", originalTLS.MinVersion, loadedTLS.MinVersion)
	}

	if loadedTLS.MaxVersion != originalTLS.MaxVersion {
		t.Errorf("MaxVersion not preserved. Expected %d, got %d", originalTLS.MaxVersion, loadedTLS.MaxVersion)
	}

	if loadedTLS.ServerName != originalTLS.ServerName {
		t.Errorf("ServerName not preserved. Expected '%s', got '%s'", originalTLS.ServerName, loadedTLS.ServerName)
	}

	if len(loadedTLS.CipherSuites) != len(originalTLS.CipherSuites) {
		t.Errorf("CipherSuites not preserved. Expected %d, got %d",
			len(originalTLS.CipherSuites), len(loadedTLS.CipherSuites))
	}
}

// TestInsecure_Warning verifies warning is printed when Insecure is enabled
func TestInsecure_Warning(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	opts := options.NewRequestOptions("https://example.com")
	opts.Insecure = true
	opts.Silent = false // Ensure warning is shown

	// Load TLS config (should print warning)
	_, err := LoadTLSConfig(opts)
	if err != nil {
		t.Fatalf("LoadTLSConfig failed: %v", err)
	}

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify warning was printed
	if !strings.Contains(output, "WARNING") {
		t.Errorf("Expected WARNING in output, got: %s", output)
	}

	if !strings.Contains(output, "insecure") || !strings.Contains(output, "Certificate verification is disabled") {
		t.Errorf("Expected insecure warning message, got: %s", output)
	}

	if !strings.Contains(output, "NOT secure") {
		t.Errorf("Expected security warning, got: %s", output)
	}
}

// TestInsecure_NoWarningWhenSilent verifies no warning when Silent is true
func TestInsecure_NoWarningWhenSilent(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	opts := options.NewRequestOptions("https://example.com")
	opts.Insecure = true
	opts.Silent = true // Suppress warning

	// Load TLS config
	_, err := LoadTLSConfig(opts)
	if err != nil {
		t.Fatalf("LoadTLSConfig failed: %v", err)
	}

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify NO warning was printed
	if strings.Contains(output, "WARNING") {
		t.Errorf("Expected no warning when Silent=true, got: %s", output)
	}
}

// TestInsecure_WarningWithVerbose verifies warning is shown with Verbose
func TestInsecure_WarningWithVerbose(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	opts := options.NewRequestOptions("https://example.com")
	opts.Insecure = true
	opts.Verbose = true
	opts.Silent = true // Even with Silent, Verbose should show warning

	// Load TLS config
	_, err := LoadTLSConfig(opts)
	if err != nil {
		t.Fatalf("LoadTLSConfig failed: %v", err)
	}

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify warning was printed (Verbose overrides Silent for warnings)
	if !strings.Contains(output, "WARNING") {
		t.Errorf("Expected WARNING with Verbose=true, got: %s", output)
	}
}

// TestInsecure_IntegrationWithHTTPS verifies insecure mode works end-to-end
func TestInsecure_IntegrationWithHTTPS(t *testing.T) {
	// Create HTTPS test server with self-signed certificate
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Secure response")
	}))
	defer server.Close()

	// Capture stderr for warnings
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	opts := options.NewRequestOptions(server.URL)
	opts.Insecure = true // Required for self-signed cert
	opts.Silent = false

	// Should succeed with Insecure=true
	_, _, err := Process(context.Background(), opts)

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Fatalf("Process failed with Insecure=true: %v", err)
	}

	// Verify warning was printed
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "WARNING") {
		t.Errorf("Expected security warning, got: %s", output)
	}
}

// TestTLSConfig_SecureDefaults verifies secure defaults are applied
func TestTLSConfig_SecureDefaults(t *testing.T) {
	opts := options.NewRequestOptions("https://example.com")
	// Don't set custom TLSConfig - use defaults

	tlsConfig, err := LoadTLSConfig(opts)
	if err != nil {
		t.Fatalf("LoadTLSConfig failed: %v", err)
	}

	// Verify secure defaults
	if tlsConfig.MinVersion < tls.VersionTLS12 {
		t.Errorf("Expected MinVersion >= TLS 1.2, got 0x%x", tlsConfig.MinVersion)
	}

	if tlsConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be false by default")
	}

	if len(tlsConfig.CipherSuites) == 0 {
		t.Error("Expected secure cipher suites to be configured")
	}
}

// TestTLSConfig_NilConfig verifies nil TLSConfig doesn't cause panic
func TestTLSConfig_NilConfig(t *testing.T) {
	opts := options.NewRequestOptions("https://example.com")
	opts.TLSConfig = nil // Explicitly nil

	// Should use secure defaults
	tlsConfig, err := LoadTLSConfig(opts)
	if err != nil {
		t.Fatalf("LoadTLSConfig failed with nil config: %v", err)
	}

	if tlsConfig == nil {
		t.Fatal("Expected non-nil TLS config even when opts.TLSConfig is nil")
	}

	// Should have secure defaults
	if tlsConfig.MinVersion < tls.VersionTLS12 {
		t.Errorf("Expected secure MinVersion, got 0x%x", tlsConfig.MinVersion)
	}
}
