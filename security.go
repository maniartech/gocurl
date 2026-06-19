package gocurl

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"

	"github.com/maniartech/gocurl/options"
)

// LoadTLSConfig creates a TLS configuration from RequestOptions.
// Supports client certificates, custom CA bundles, and SNI.
func LoadTLSConfig(opts *options.RequestOptions) (*tls.Config, error) {
	if opts == nil {
		return nil, fmt.Errorf("request options cannot be nil")
	}

	// Start with secure defaults
	tlsConfig := SecureDefaults()

	// If user provided a custom TLS config, merge it OVER the secure defaults
	// rather than replacing them. Cloning preserves the caller's RootCAs,
	// Certificates, ServerName, etc.; we then restore the secure floor and curated
	// cipher list for any field the caller left at its zero value, so a
	// WithTLSConfig(&tls.Config{RootCAs: pool}) still gets the TLS 1.2 floor.
	if opts.TLSConfig != nil {
		base := tlsConfig
		tlsConfig = opts.TLSConfig.Clone()
		if tlsConfig.MinVersion == 0 {
			tlsConfig.MinVersion = base.MinVersion
		}
		if len(tlsConfig.CipherSuites) == 0 {
			tlsConfig.CipherSuites = base.CipherSuites
		}
	}

	// Handle insecure mode
	if opts.Insecure {
		tlsConfig.InsecureSkipVerify = true
		// Print security warning to stderr (like curl does)
		if opts.Verbose || !opts.Silent {
			fmt.Fprintf(os.Stderr, "WARNING: Using --insecure mode. Certificate verification is disabled.\n")
			fmt.Fprintf(os.Stderr, "WARNING: This is NOT secure and should only be used for testing.\n")
		}
	}

	// Load client certificate and key if provided
	if opts.CertFile != "" && opts.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(opts.CertFile, opts.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load custom CA bundle if provided
	if opts.CAFile != "" {
		caCert, err := os.ReadFile(opts.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.RootCAs = caCertPool
	}

	// Certificate pinning if provided.
	if len(opts.CertPinFingerprints) > 0 {
		// Run the pin check via VerifyPeerCertificate. We deliberately do NOT set
		// InsecureSkipVerify here: with verification ON, the stdlib validates the
		// chain first and then calls this callback, so the pin is checked IN
		// ADDITION to chain verification — a typo'd pin against a valid chain fails
		// CLOSED. Only when the caller ALSO passed --insecure (handled above, which
		// sets InsecureSkipVerify) does the pin become the sole check.
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			return VerifyCertificatePin(rawCerts, opts.CertPinFingerprints)
		}
	}

	// SNI (Server Name Indication) - set the server name if provided
	if opts.SNIServerName != "" {
		tlsConfig.ServerName = opts.SNIServerName
	}

	// Apply explicit TLS version and cipher constraints from curl flags. This
	// makes LoadTLSConfig the single source of truth so --tlsv1.x / --tls-max /
	// --ciphers take effect on every request, not only when a cert is also given.
	if opts.TLSMinVersion != 0 {
		tlsConfig.MinVersion = opts.TLSMinVersion
	}
	if opts.TLSMaxVersion != 0 {
		tlsConfig.MaxVersion = opts.TLSMaxVersion
	}
	if len(opts.CipherSuites) > 0 {
		tlsConfig.CipherSuites = opts.CipherSuites
	}
	// NOTE: Go's crypto/tls does not support configuring TLS 1.3 cipher suites;
	// opts.TLS13CipherSuites is accepted for curl compatibility but cannot be
	// enforced by the standard library.

	return tlsConfig, nil
}

// VerifyCertificatePin checks if any certificate in the chain matches the provided fingerprints.
// This implements certificate pinning for enhanced security.
func VerifyCertificatePin(rawCerts [][]byte, fingerprints []string) error {
	if len(fingerprints) == 0 {
		return nil
	}

	for _, rawCert := range rawCerts {
		cert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			continue
		}

		// Calculate the SHA-256 fingerprint of the certificate's DER bytes.
		sum := sha256.Sum256(cert.Raw)
		certFingerprint := fmt.Sprintf("%x", sum[:])

		for _, pin := range fingerprints {
			// Normalize pin (strip sha256/ prefix, colons, spaces; lowercase).
			normalizedPin := strings.ToLower(pin)
			normalizedPin = strings.TrimPrefix(normalizedPin, "sha256//")
			normalizedPin = strings.TrimPrefix(normalizedPin, "sha256:")
			normalizedPin = strings.ReplaceAll(normalizedPin, ":", "")
			normalizedPin = strings.ReplaceAll(normalizedPin, " ", "")
			if certFingerprint == normalizedPin {
				return nil // Pin matched
			}
		}
	}

	return fmt.Errorf("certificate pin verification failed: no matching fingerprint found")
}

// ValidateTLSConfig checks TLS configuration for security issues
func ValidateTLSConfig(tlsConfig *tls.Config, opts *options.RequestOptions) error {
	if tlsConfig == nil {
		return nil
	}

	// Warn if InsecureSkipVerify is enabled
	if tlsConfig.InsecureSkipVerify {
		// This is allowed but should be explicit via opts.Insecure
		if !opts.Insecure && (opts.CertPinFingerprints == nil || len(opts.CertPinFingerprints) == 0) {
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
	if err := validateURL(opts); err != nil {
		return err
	}

	// Validate TLS configuration
	if err := validateTLSOptions(opts); err != nil {
		return err
	}

	// Validate timeout values
	if err := validateTimeouts(opts); err != nil {
		return err
	}

	// Validate redirects and retries
	if err := validateRedirectsAndRetries(opts); err != nil {
		return err
	}

	// Runtime input validation on the live path (Spec 07 §4): method, header
	// count/size + forbidden headers, in-memory body size, form/query counts, and
	// the plaintext-auth policy. Streaming bodies are exempt from the body cap.
	if err := options.ValidateRequest(opts); err != nil {
		return ValidationError("request", err)
	}

	return nil
}

// validateURL validates the URL field
func validateURL(opts *options.RequestOptions) error {
	if opts.URL == "" {
		return ValidationError("URL", fmt.Errorf("URL is required"))
	}

	// HTTP is allowed, just not recommended for sensitive data
	// Don't error for http:// URLs, but could log warning in verbose mode

	return nil
}

// validateTLSOptions validates TLS-related options
func validateTLSOptions(opts *options.RequestOptions) error {
	// Validate TLS config if provided
	if opts.TLSConfig != nil {
		if err := ValidateTLSConfig(opts.TLSConfig, opts); err != nil {
			return ValidationError("TLS", err)
		}
	}

	// Validate cert and key are both provided or neither
	if err := validateCertKeyPair(opts); err != nil {
		return err
	}

	// Validate file existence
	if err := validateTLSFiles(opts); err != nil {
		return err
	}

	return nil
}

// validateCertKeyPair ensures cert and key are provided together
func validateCertKeyPair(opts *options.RequestOptions) error {
	hasCert := opts.CertFile != ""
	hasKey := opts.KeyFile != ""

	if hasCert != hasKey {
		return ValidationError("TLS", fmt.Errorf("both cert-file and key-file must be provided together"))
	}

	return nil
}

// validateTLSFiles validates that TLS files exist
func validateTLSFiles(opts *options.RequestOptions) error {
	files := map[string]string{
		"CertFile": opts.CertFile,
		"KeyFile":  opts.KeyFile,
		"CAFile":   opts.CAFile,
	}

	for name, path := range files {
		if path != "" {
			if _, err := os.Stat(path); err != nil {
				return ValidationError(name, fmt.Errorf("%s file not found: %w", strings.ToLower(name), err))
			}
		}
	}

	return nil
}

// validateTimeouts validates timeout values
func validateTimeouts(opts *options.RequestOptions) error {
	if opts.Timeout < 0 {
		return ValidationError("Timeout", fmt.Errorf("timeout cannot be negative"))
	}

	if opts.ConnectTimeout < 0 {
		return ValidationError("ConnectTimeout", fmt.Errorf("connect timeout cannot be negative"))
	}

	return nil
}

// validateRedirectsAndRetries validates redirect and retry configuration
func validateRedirectsAndRetries(opts *options.RequestOptions) error {
	if opts.MaxRedirects < 0 {
		return ValidationError("MaxRedirects", fmt.Errorf("max redirects cannot be negative"))
	}

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
