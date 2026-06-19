package gocurl

import (
	"context"
	"errors"
	"io"
	"net/http"
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
