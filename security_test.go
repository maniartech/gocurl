package gocurl

import (
	"crypto/tls"
	"os"
	"testing"

	"github.com/maniartech/gocurl/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTLSConfig(t *testing.T) {
	tests := []struct {
		name        string
		opts        *options.RequestOptions
		expectError bool
		validate    func(*testing.T, *tls.Config)
	}{
		{
			name: "default secure config",
			opts: &options.RequestOptions{
				URL: "https://example.com",
			},
			expectError: false,
			validate: func(t *testing.T, cfg *tls.Config) {
				assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
				assert.False(t, cfg.InsecureSkipVerify)
			},
		},
		{
			name: "insecure mode",
			opts: &options.RequestOptions{
				URL:      "https://example.com",
				Insecure: true,
			},
			expectError: false,
			validate: func(t *testing.T, cfg *tls.Config) {
				assert.True(t, cfg.InsecureSkipVerify)
			},
		},
		{
			name: "with custom SNI",
			opts: &options.RequestOptions{
				URL:           "https://example.com",
				SNIServerName: "custom.example.com",
			},
			expectError: false,
			validate: func(t *testing.T, cfg *tls.Config) {
				assert.Equal(t, "custom.example.com", cfg.ServerName)
			},
		},
		{
			name: "with certificate pinning",
			opts: &options.RequestOptions{
				URL:                 "https://example.com",
				CertPinFingerprints: []string{"abcdef123456"},
			},
			expectError: false,
			validate: func(t *testing.T, cfg *tls.Config) {
				assert.NotNil(t, cfg.VerifyPeerCertificate)
				assert.True(t, cfg.InsecureSkipVerify, "Must set InsecureSkipVerify when using custom verification")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadTLSConfig(tt.opts)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, cfg)
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestLoadTLSConfigNilOptions(t *testing.T) {
	cfg, err := LoadTLSConfig(nil)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestVerifyCertificatePin(t *testing.T) {
	// For this test, we'll use the fact that when fingerprints is empty, it should return nil
	// And test with invalid cert data to ensure error handling works

	tests := []struct {
		name         string
		rawCerts     [][]byte
		fingerprints []string
		expectError  bool
	}{
		{
			name:         "empty fingerprints",
			rawCerts:     [][]byte{{0x01, 0x02, 0x03}},
			fingerprints: []string{},
			expectError:  false,
		},
		{
			name:         "invalid certificate data",
			rawCerts:     [][]byte{{0x01, 0x02, 0x03}}, // Invalid cert - will be skipped
			fingerprints: []string{"abcdef"},
			expectError:  true, // No valid certs to match against
		},
		{
			name:         "no certificates provided",
			rawCerts:     [][]byte{},
			fingerprints: []string{"abcdef"},
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyCertificatePin(tt.rawCerts, tt.fingerprints)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTLSConfig(t *testing.T) {
	tests := []struct {
		name        string
		tlsConfig   *tls.Config
		opts        *options.RequestOptions
		expectError bool
	}{
		{
			name:        "nil config",
			tlsConfig:   nil,
			opts:        &options.RequestOptions{},
			expectError: false,
		},
		{
			name: "insecure without flag",
			tlsConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			opts:        &options.RequestOptions{Insecure: false},
			expectError: true,
		},
		{
			name: "insecure with flag",
			tlsConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			opts:        &options.RequestOptions{Insecure: true},
			expectError: false,
		},
		{
			name: "weak TLS version",
			tlsConfig: &tls.Config{
				MinVersion: tls.VersionTLS10,
			},
			opts:        &options.RequestOptions{},
			expectError: true,
		},
		{
			name: "acceptable TLS version",
			tlsConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			opts:        &options.RequestOptions{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTLSConfig(tt.tlsConfig, tt.opts)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRequestOptions(t *testing.T) {
	tests := []struct {
		name        string
		opts        *options.RequestOptions
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil options",
			opts:        nil,
			expectError: true,
			errorMsg:    "cannot be nil",
		},
		{
			name: "missing URL",
			opts: &options.RequestOptions{
				Method: "GET",
			},
			expectError: true,
			errorMsg:    "URL is required",
		},
		{
			name: "valid options",
			opts: &options.RequestOptions{
				URL:    "https://example.com",
				Method: "GET",
			},
			expectError: false,
		},
		{
			name: "negative timeout",
			opts: &options.RequestOptions{
				URL:     "https://example.com",
				Timeout: -1,
			},
			expectError: true,
			errorMsg:    "timeout cannot be negative",
		},
		{
			name: "cert without key",
			opts: &options.RequestOptions{
				URL:      "https://example.com",
				CertFile: "/path/to/cert.pem",
			},
			expectError: true,
			errorMsg:    "both cert-file and key-file must be provided together",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequestOptions(tt.opts)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecureDefaults(t *testing.T) {
	cfg := SecureDefaults()

	assert.NotNil(t, cfg)
	assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion, "Should use TLS 1.2 minimum")
	assert.NotEmpty(t, cfg.CipherSuites, "Should have cipher suites configured")
	assert.False(t, cfg.PreferServerCipherSuites, "Should prefer client cipher suites")
}

func TestValidateVariables(t *testing.T) {
	tests := []struct {
		name        string
		vars        Variables
		expectError bool
	}{
		{
			name:        "nil variables",
			vars:        nil,
			expectError: false,
		},
		{
			name: "valid variables",
			vars: Variables{
				"key1": "value1",
				"key2": "value2",
			},
			expectError: false,
		},
		{
			name: "empty variable name",
			vars: Variables{
				"": "value",
			},
			expectError: true,
		},
		{
			name: "variable name with invalid characters",
			vars: Variables{
				"key${}": "value",
			},
			expectError: true,
		},
		{
			name: "oversized variable value",
			vars: Variables{
				"key": string(make([]byte, 2*1024*1024)), // 2MB
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVariables(tt.vars)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTLSConfigWithRealCertificates(t *testing.T) {
	// Create temporary certificate files for testing
	certPEM := `-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`

	keyPEM := `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`

	// Create temp files
	certFile, err := os.CreateTemp("", "cert-*.pem")
	require.NoError(t, err)
	defer os.Remove(certFile.Name())

	keyFile, err := os.CreateTemp("", "key-*.pem")
	require.NoError(t, err)
	defer os.Remove(keyFile.Name())

	os.WriteFile(certFile.Name(), []byte(certPEM), 0644)
	os.WriteFile(keyFile.Name(), []byte(keyPEM), 0644)

	opts := &options.RequestOptions{
		URL:      "https://example.com",
		CertFile: certFile.Name(),
		KeyFile:  keyFile.Name(),
	}

	cfg, err := LoadTLSConfig(opts)
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Certificates, 1, "Should load one certificate")
}

func BenchmarkLoadTLSConfig(b *testing.B) {
	opts := &options.RequestOptions{
		URL:           "https://example.com",
		SNIServerName: "sni.example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LoadTLSConfig(opts)
	}
}

func BenchmarkValidateRequestOptions(b *testing.B) {
	opts := &options.RequestOptions{
		URL:    "https://example.com",
		Method: "GET",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateRequestOptions(opts)
	}
}
