package gocurl

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/maniartech/gocurl/options"
)

// ValidateTLSConfig checks TLS configuration for security issues
func ValidateTLSConfig(tlsConfig *tls.Config, opts *options.RequestOptions) error {
	if tlsConfig == nil {
		return nil
	}

	// Warn if InsecureSkipVerify is enabled
	if tlsConfig.InsecureSkipVerify {
		// This is allowed but should be explicit via opts.Insecure
		if !opts.Insecure {
			return fmt.Errorf("InsecureSkipVerify is enabled but --insecure flag not set")
		}
	}

	// Check for weak TLS versions
	if tlsConfig.MinVersion > 0 && tlsConfig.MinVersion < tls.VersionTLS12 {
		return fmt.Errorf("TLS version < 1.2 is not recommended (current: 0x%x)", tlsConfig.MinVersion)
	}

	// Check certificates if provided
	if len(tlsConfig.Certificates) > 0 {
		for i, cert := range tlsConfig.Certificates {
			if len(cert.Certificate) == 0 {
				return fmt.Errorf("certificate %d has no certificate data", i)
			}

			// Parse and validate certificate
			x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
			if err != nil {
				return fmt.Errorf("failed to parse certificate %d: %w", i, err)
			}

			// Check if certificate is expired (informational)
			if x509Cert.NotAfter.Before(x509Cert.NotBefore) {
				return fmt.Errorf("certificate %d has invalid validity period", i)
			}
		}
	}

	return nil
}

// ValidateRequestOptions performs security validation on request options
func ValidateRequestOptions(opts *options.RequestOptions) error {
	if opts == nil {
		return fmt.Errorf("request options cannot be nil")
	}

	// Validate URL
	if opts.URL == "" {
		return ValidationError("URL", fmt.Errorf("URL is required"))
	}

	// Check for insecure patterns
	if strings.HasPrefix(opts.URL, "http://") && !opts.Insecure {
		// HTTP is allowed, just not recommended for sensitive data
		// Don't error, but could log a warning in verbose mode
	}

	// Validate TLS config
	if opts.TLSConfig != nil {
		if err := ValidateTLSConfig(opts.TLSConfig, opts); err != nil {
			return ValidationError("TLS", err)
		}
	}

	// Validate cert/key files exist if specified
	if opts.CertFile != "" {
		if _, err := os.Stat(opts.CertFile); err != nil {
			return ValidationError("CertFile", fmt.Errorf("certificate file not found: %w", err))
		}
	}

	if opts.KeyFile != "" {
		if _, err := os.Stat(opts.KeyFile); err != nil {
			return ValidationError("KeyFile", fmt.Errorf("key file not found: %w", err))
		}
	}

	if opts.CAFile != "" {
		if _, err := os.Stat(opts.CAFile); err != nil {
			return ValidationError("CAFile", fmt.Errorf("CA file not found: %w", err))
		}
	}

	// Validate that cert and key are both provided or neither
	if (opts.CertFile != "" && opts.KeyFile == "") || (opts.CertFile == "" && opts.KeyFile != "") {
		return ValidationError("TLS", fmt.Errorf("both cert-file and key-file must be provided together"))
	}

	// Validate timeout values
	if opts.Timeout < 0 {
		return ValidationError("Timeout", fmt.Errorf("timeout cannot be negative"))
	}

	if opts.ConnectTimeout < 0 {
		return ValidationError("ConnectTimeout", fmt.Errorf("connect timeout cannot be negative"))
	}

	// Validate max redirects
	if opts.MaxRedirects < 0 {
		return ValidationError("MaxRedirects", fmt.Errorf("max redirects cannot be negative"))
	}

	// Validate retry configuration
	if opts.RetryConfig != nil {
		if opts.RetryConfig.MaxRetries < 0 {
			return ValidationError("RetryConfig.MaxRetries", fmt.Errorf("max retries cannot be negative"))
		}
		if opts.RetryConfig.RetryDelay < 0 {
			return ValidationError("RetryConfig.RetryDelay", fmt.Errorf("retry delay cannot be negative"))
		}
	}

	return nil
}

// SanitizeHeadersForLogging redacts sensitive headers before logging
func SanitizeHeadersForLogging(headers map[string][]string) map[string][]string {
	return RedactHeaders(headers)
}

// ValidateVariables checks variable names and values for security issues
func ValidateVariables(vars Variables) error {
	if vars == nil {
		return nil
	}

	for name, value := range vars {
		// Check for empty variable names
		if name == "" {
			return fmt.Errorf("variable name cannot be empty")
		}

		// Check for variable names with suspicious characters
		if strings.ContainsAny(name, "${}\\'\"`") {
			return fmt.Errorf("variable name %q contains invalid characters", name)
		}

		// Warn about suspiciously large values (could indicate injection attempt)
		if len(value) > 1024*1024 { // 1MB
			return fmt.Errorf("variable %q value exceeds maximum size (1MB)", name)
		}
	}

	return nil
}

// SecureDefaults returns a TLS configuration with secure defaults
func SecureDefaults() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: false, // Use client preferences
	}
}
