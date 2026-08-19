package gocurl

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSSRFPolicy_CheckSSRF(t *testing.T) {
	def := DefaultSSRFPolicy()
	cases := []struct {
		name    string
		policy  SSRFPolicy
		host    string
		blocked bool
	}{
		{"loopback v4", def, "127.0.0.1", true},
		{"loopback v4 with port", def, "127.0.0.1:8080", true},
		{"loopback v6", def, "::1", true},
		{"loopback v6 bracketed port", def, "[::1]:8080", true},
		{"link-local v4", def, "169.254.1.1", true},
		{"cloud metadata", def, "169.254.169.254", true},
		{"private 10", def, "10.0.0.1", true},
		{"private 192.168", def, "192.168.1.1", true},
		{"private 172.16", def, "172.16.5.5", true},
		{"ula v6", def, "fd00::1", true},
		{"link-local v6", def, "fe80::1", true},
		{"public v4 allowed", def, "8.8.8.8", false},
		{"metadata hostname", def, "metadata.google.internal", true},
		{"metadata hostname trailing dot", def, "metadata.google.internal.", true},
		{"metadata hostname trailing dot with port", def, "metadata.google.internal.:443", true},
		{"unspecified v4", def, "0.0.0.0", true},
		{"unspecified v4 with port", def, "0.0.0.0:80", true},
		{"unspecified v6", def, "::", true},
		{"unspecified v6 bracketed port", def, "[::]:80", true},
		{"unspecified v4-mapped v6", def, "::ffff:0.0.0.0", true},
		{"unspecified allowed when loopback block off", SSRFPolicy{BlockPrivate: true}, "0.0.0.0", false},
		{"empty host", def, "", false},
		{"allow-list host wins", SSRFPolicy{BlockLoopback: true, AllowHosts: []string{"127.0.0.1"}}, "127.0.0.1", false},
		{"allow-list CIDR wins", SSRFPolicy{BlockPrivate: true, AllowIPs: []string{"10.0.0.0/8"}}, "10.1.2.3", false},
		{"block disabled", SSRFPolicy{}, "127.0.0.1", false},
		{"only metadata blocked, loopback ok", SSRFPolicy{BlockCloudMetadata: true}, "127.0.0.1", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.policy.CheckSSRF(context.Background(), tc.host)
			if tc.blocked {
				if err == nil {
					t.Fatalf("CheckSSRF(%q) = nil, want blocked", tc.host)
				}
				if !errors.Is(err, ErrSSRFBlocked) {
					t.Errorf("error should match ErrSSRFBlocked: %v", err)
				}
				if KindOf(err) != KindValidation {
					t.Errorf("KindOf = %v, want KindValidation", KindOf(err))
				}
				if IsRetryable(err) {
					t.Error("an SSRF block must not be retryable")
				}
			} else if err != nil {
				t.Errorf("CheckSSRF(%q) = %v, want allowed", tc.host, err)
			}
		})
	}
}

func TestSSRFGuard_HandlerAdapterDialsValidatedIP(t *testing.T) {
	originalLookup := lookupIPAddr
	t.Cleanup(func() { lookupIPAddr = originalLookup })
	lookupCalls := 0
	lookupIPAddr = func(context.Context, string) ([]net.IPAddr, error) {
		lookupCalls++
		// Model attacker-controlled DNS: validation receives a public address,
		// while any accidental second lookup would receive cloud metadata.
		if lookupCalls > 1 {
			return []net.IPAddr{{IP: net.ParseIP("169.254.169.254")}}, nil
		}
		return []net.IPAddr{{IP: net.ParseIP("203.0.113.10")}}, nil
	}

	var dialed string
	stop := errors.New("stop after observing dial target")
	rt := &http.Transport{
		Proxy: nil,
		DialContext: func(_ context.Context, _, address string) (net.Conn, error) {
			dialed = address
			return nil, stop
		},
	}
	h := SSRFGuard(DefaultSSRFPolicy())(HandlerFromRoundTripper(rt))
	req, _ := http.NewRequest(http.MethodGet, "http://rebind.example/resource", nil)

	_, err := h(req)
	if !errors.Is(err, stop) {
		t.Fatalf("error=%v, want dial sentinel", err)
	}
	if lookupCalls != 1 {
		t.Fatalf("DNS lookups=%d, want exactly the validation lookup", lookupCalls)
	}
	if dialed != "203.0.113.10:80" {
		t.Fatalf("dialed %q, want validated IP", dialed)
	}
}

func TestSSRFGuard_ClientDialsValidatedIP(t *testing.T) {
	originalLookup := lookupIPAddr
	t.Cleanup(func() { lookupIPAddr = originalLookup })
	lookupIPAddr = func(context.Context, string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("203.0.113.20")}}, nil
	}

	var dialed string
	stop := errors.New("stop after observing client dial target")
	rt := &http.Transport{
		Proxy: nil,
		DialContext: func(_ context.Context, _, address string) (net.Conn, error) {
			dialed = address
			return nil, stop
		},
	}
	client, err := New(WithTransport(rt), WithSSRFGuard(DefaultSSRFPolicy()))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	req, err := NewRequest(http.MethodGet, "http://native-rebind.example/resource")
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Do(context.Background(), req)
	if !errors.Is(err, stop) {
		t.Fatalf("error=%v, want dial sentinel", err)
	}
	if dialed != "203.0.113.20:80" {
		t.Fatalf("dialed %q, want validated IP", dialed)
	}
}

func TestCompositionRobustness_PinnedTLSPreservesServerName(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	originalLookup := lookupIPAddr
	t.Cleanup(func() { lookupIPAddr = originalLookup })
	lookupIPAddr = func(_ context.Context, host string) ([]net.IPAddr, error) {
		if host != "example.com" {
			return nil, errors.New("unexpected hostname")
		}
		return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}, nil
	}

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	rt := &http.Transport{Proxy: nil, TLSClientConfig: &tls.Config{RootCAs: pool}}
	policy := DefaultSSRFPolicy()
	policy.AllowHosts = []string{"example.com"}
	h := SSRFGuard(policy)(HandlerFromRoundTripper(rt))
	port := strings.TrimPrefix(srv.URL, "https://127.0.0.1:")
	req, _ := http.NewRequest(http.MethodGet, "https://example.com:"+port, nil)

	resp, err := h(req)
	if err != nil {
		t.Fatalf("TLS request with pinned IP and original SNI failed: %v", err)
	}
	_ = resp.Body.Close()
}

func TestCompositionRobustness_OpaqueTransportFailsClosed(t *testing.T) {
	called := false
	opaque := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})
	h := SSRFGuard(DefaultSSRFPolicy())(HandlerFromRoundTripper(opaque))
	req, _ := http.NewRequest(http.MethodGet, "http://203.0.113.30", nil)

	_, err := h(req)
	if !errors.Is(err, ErrSSRFBlocked) {
		t.Fatalf("error=%v, want fail-closed SSRF error", err)
	}
	if called {
		t.Fatal("opaque transport was called despite being unable to enforce the dial pin")
	}
}

func TestSSRFGuard_Middleware(t *testing.T) {
	called := false
	next := Handler(func(*http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	h := SSRFGuard(DefaultSSRFPolicy())(next)

	// Blocked host: next is not called.
	req, _ := http.NewRequest("GET", "http://127.0.0.1:9999/x", nil)
	if _, err := h(req); !errors.Is(err, ErrSSRFBlocked) {
		t.Fatalf("expected ErrSSRFBlocked, got %v", err)
	}
	if called {
		t.Error("next must not run for a blocked host")
	}

	// Allowed host: next runs.
	called = false
	req2, _ := http.NewRequest("GET", "http://8.8.8.8/x", nil)
	if _, err := h(req2); err != nil {
		t.Fatalf("allowed host should pass: %v", err)
	}
	if !called {
		t.Error("next should run for an allowed host")
	}
}

func BenchmarkCheckSSRF_LiteralIP(b *testing.B) {
	p := DefaultSSRFPolicy()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.CheckSSRF(ctx, "10.0.0.1")
	}
}
