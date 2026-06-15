package gocurl

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maniartech/gocurl/options"
)

func TestDeflate_ZlibWrapped(t *testing.T) {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	_, _ = zw.Write([]byte("hello deflate (zlib)"))
	_ = zw.Close()

	resp := &http.Response{
		Header: http.Header{"Content-Encoding": []string{"deflate"}},
		Body:   io.NopCloser(bytes.NewReader(buf.Bytes())),
	}
	if err := DecompressResponse(resp); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "hello deflate (zlib)" {
		t.Errorf("got %q", out)
	}
}

func TestDeflate_RawFlate(t *testing.T) {
	var buf bytes.Buffer
	fw, _ := flate.NewWriter(&buf, flate.DefaultCompression)
	_, _ = fw.Write([]byte("hello deflate (raw)"))
	_ = fw.Close()

	resp := &http.Response{
		Header: http.Header{"Content-Encoding": []string{"deflate"}},
		Body:   io.NopCloser(bytes.NewReader(buf.Bytes())),
	}
	if err := DecompressResponse(resp); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "hello deflate (raw)" {
		t.Errorf("got %q", out)
	}
}

func TestCertPin_SHA256Matches(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	cert := srv.Certificate()
	sum := sha256.Sum256(cert.Raw)
	pin := fmt.Sprintf("%x", sum[:])

	if err := VerifyCertificatePin([][]byte{cert.Raw}, []string{pin}); err != nil {
		t.Errorf("expected matching pin, got error: %v", err)
	}
	// Prefixed / upper-case form should also match.
	if err := VerifyCertificatePin([][]byte{cert.Raw}, []string{"sha256//" + strings.ToUpper(pin)}); err != nil {
		t.Errorf("expected prefixed pin to match, got error: %v", err)
	}
	// Wrong pin must fail.
	if err := VerifyCertificatePin([][]byte{cert.Raw}, []string{"deadbeef"}); err == nil {
		t.Error("expected non-matching pin to fail")
	}
}

func TestLoadTLSConfig_AppliesVersionAndCiphers(t *testing.T) {
	opts := options.NewRequestOptions("https://example.com")
	opts.TLSMinVersion = tls.VersionTLS13
	opts.TLSMaxVersion = tls.VersionTLS13
	opts.CipherSuites = []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}

	cfg, err := LoadTLSConfig(opts)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MinVersion != tls.VersionTLS13 || cfg.MaxVersion != tls.VersionTLS13 {
		t.Errorf("version not applied: min=0x%x max=0x%x", cfg.MinVersion, cfg.MaxVersion)
	}
	if len(cfg.CipherSuites) != 1 || cfg.CipherSuites[0] != tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 {
		t.Errorf("ciphers not applied: %v", cfg.CipherSuites)
	}
}

func TestTransportCache_ReusesTransport(t *testing.T) {
	a := options.NewRequestOptions("https://example.com")
	b := options.NewRequestOptions("https://example.com")

	rtA, err := getRoundTripper(a)
	if err != nil {
		t.Fatal(err)
	}
	rtB, err := getRoundTripper(b)
	if err != nil {
		t.Fatal(err)
	}
	if rtA != rtB {
		t.Error("expected identical config to reuse the same cached transport")
	}

	c := options.NewRequestOptions("https://example.com")
	c.Insecure = true
	c.Silent = true // suppress the insecure warning
	rtC, err := getRoundTripper(c)
	if err != nil {
		t.Fatal(err)
	}
	if rtA == rtC {
		t.Error("different TLS config should not share a transport")
	}
}
