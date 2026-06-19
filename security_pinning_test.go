package gocurl

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maniartech/gocurl/options"
)

// TestLoadTLSConfig_PinningFailsClosed verifies the Spec 07 §1 hardening: a
// pinned request keeps chain verification ON, so a WRONG pin against an otherwise
// valid (trusted) chain fails closed, while the correct pin succeeds.
func TestLoadTLSConfig_PinningFailsClosed(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cert := srv.Certificate()
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	goodPin := fmt.Sprintf("%x", sha256.Sum256(cert.Raw))

	get := func(pin string) error {
		opts := options.NewRequestOptions(srv.URL)
		opts.TLSConfig = &tls.Config{RootCAs: pool} // trust the test CA (chain verification on)
		opts.CertPinFingerprints = []string{pin}
		cfg, err := LoadTLSConfig(opts)
		if err != nil {
			return err
		}
		if cfg.InsecureSkipVerify {
			t.Fatal("pinning without --insecure must NOT set InsecureSkipVerify")
		}
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: cfg}}
		resp, err := client.Get(srv.URL)
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	}

	if err := get(goodPin); err != nil {
		t.Errorf("valid chain + correct pin should succeed, got: %v", err)
	}
	if err := get("deadbeefbadpin"); err == nil {
		t.Error("valid chain + WRONG pin must fail closed")
	}
}

// TestLoadTLSConfig_MergesSecureDefaults is the regression for the
// replace-instead-of-merge bug: a caller-supplied TLS config (e.g. only a custom
// RootCAs pool) must still inherit the secure floor (TLS 1.2) and the curated
// cipher list for the fields it left at their zero value.
func TestLoadTLSConfig_MergesSecureDefaults(t *testing.T) {
	pool := x509.NewCertPool()
	opts := options.NewRequestOptions("https://example.com")
	opts.TLSConfig = &tls.Config{RootCAs: pool} // only RootCAs set; MinVersion/ciphers zero

	cfg, err := LoadTLSConfig(opts)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = 0x%x, want the secure floor 0x%x (TLS 1.2)", cfg.MinVersion, tls.VersionTLS12)
	}
	if len(cfg.CipherSuites) == 0 {
		t.Error("CipherSuites was dropped; the curated secure list should be preserved")
	}
	if cfg.RootCAs != pool {
		t.Error("caller-supplied RootCAs must be preserved")
	}
}
