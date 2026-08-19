package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVerifyConnectResponse(t *testing.T) {
	cases := []struct {
		name     string
		response string
		wantErr  string
	}{
		{name: "success", response: "HTTP/1.1 200 Connection Established\r\n\r\n"},
		{name: "rejected", response: "HTTP/1.1 407 Proxy Authentication Required\r\nContent-Length: 0\r\n\r\n", wantErr: "proxy CONNECT failed"},
		{name: "malformed", response: "not-http\r\n", wantErr: "failed to read CONNECT response"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, server := net.Pipe()
			defer client.Close()
			go func() {
				defer server.Close()
				_, _ = server.Write([]byte(tc.response))
			}()
			hp := &HTTPProxy{}
			err := hp.verifyConnectResponse(client, hp.createConnectRequest("target.example:443"))
			if tc.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("error=%v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestSendConnectRequestWritesAuthenticatedTunnelRequest(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	requestSeen := make(chan *http.Request, 1)
	go func() {
		defer server.Close()
		req, err := http.ReadRequest(bufio.NewReader(server))
		if err == nil {
			requestSeen <- req
			_, _ = server.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
		}
	}()
	hp := &HTTPProxy{Username: "alice", Password: "secret"}
	if err := hp.sendConnectRequest(client, "target.example:443"); err != nil {
		t.Fatal(err)
	}
	select {
	case req := <-requestSeen:
		if req.Method != http.MethodConnect || req.Host != "target.example:443" || req.Header.Get("Proxy-Authorization") == "" {
			t.Fatalf("unexpected CONNECT request: %+v", req)
		}
	case <-time.After(time.Second):
		t.Fatal("proxy did not receive CONNECT request")
	}
}

func TestPrepareTLSConfigPreservesCallerAndSetsSNI(t *testing.T) {
	original := &tls.Config{MinVersion: tls.VersionTLS12}
	hp := &HTTPProxy{TLSConfig: original}
	got := hp.prepareTLSConfig("api.example.com:443")
	if got == original {
		t.Fatal("TLS config was not cloned")
	}
	if original.ServerName != "" || got.ServerName != "api.example.com" || got.MinVersion != tls.VersionTLS12 {
		t.Fatalf("original=%+v prepared=%+v", original, got)
	}

	explicit := &HTTPProxy{TLSConfig: &tls.Config{ServerName: "override.example"}}
	if got := explicit.prepareTLSConfig("api.example.com:443"); got.ServerName != "override.example" {
		t.Fatalf("explicit ServerName overwritten: %q", got.ServerName)
	}
}

func TestPerformTLSHandshakeHonorsCancellation(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	tlsClient := tls.Client(client, &tls.Config{InsecureSkipVerify: true})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := (&HTTPProxy{}).performTLSHandshake(ctx, tlsClient)
	_ = tlsClient.Close()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v, want context.Canceled", err)
	}
}

func TestCreateProxyTLSConfigFilesAndErrors(t *testing.T) {
	base := &tls.Config{MinVersion: tls.VersionTLS12}
	config := ProxyConfig{
		TLSConfig:  base,
		Insecure:   true,
		ClientCert: filepath.Join("..", "fixtures", "certs", "client.crt"),
		ClientKey:  filepath.Join("..", "fixtures", "certs", "client.key"),
		CACert:     filepath.Join("..", "fixtures", "certs", "ca.crt"),
	}
	got, err := createProxyTLSConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if got == base || !got.InsecureSkipVerify || len(got.Certificates) != 1 || got.RootCAs == nil {
		t.Fatalf("unexpected proxy TLS config: %+v", got)
	}
	if base.InsecureSkipVerify {
		t.Fatal("caller TLS config was mutated")
	}

	if _, err := createProxyTLSConfig(ProxyConfig{ClientCert: "missing.crt", ClientKey: "missing.key"}); err == nil {
		t.Fatal("missing client certificate accepted")
	}
	badCA := filepath.Join(t.TempDir(), "bad-ca.pem")
	if err := os.WriteFile(badCA, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := createProxyTLSConfig(ProxyConfig{CACert: badCA}); err == nil {
		t.Fatal("invalid CA accepted")
	}
	if _, err := loadCACert("missing-ca.pem"); err == nil {
		t.Fatal("missing CA accepted")
	}
}

func TestSOCKS5ApplyConfiguration(t *testing.T) {
	if err := (&SOCKS5Proxy{}).Apply(&http.Transport{}); err == nil {
		t.Fatal("empty SOCKS5 address accepted")
	}
	transport := &http.Transport{}
	sp := &SOCKS5Proxy{Address: "127.0.0.1:1080", Timeout: time.Second, NoProxy: []string{"localhost"}}
	if err := sp.Apply(transport); err != nil {
		t.Fatal(err)
	}
	if transport.DialContext == nil || transport.Proxy == nil {
		t.Fatal("SOCKS5 transport hooks were not installed")
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	proxyURL, err := transport.Proxy(req)
	if err != nil || proxyURL.String() != "socks5://127.0.0.1:1080" {
		t.Fatalf("proxy URL=%v err=%v", proxyURL, err)
	}
}
