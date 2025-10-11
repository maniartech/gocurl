package proxy

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoProxyShouldBypassProxy(t *testing.T) {
	tests := []struct {
		name         string
		targetURL    string
		noProxyList  []string
		shouldBypass bool
	}{
		{
			name:         "exact domain match",
			targetURL:    "http://example.com/path",
			noProxyList:  []string{"example.com"},
			shouldBypass: true,
		},
		{
			name:         "subdomain match with leading dot",
			targetURL:    "http://api.example.com/path",
			noProxyList:  []string{".example.com"},
			shouldBypass: true,
		},
		{
			name:         "subdomain match without leading dot",
			targetURL:    "http://api.example.com/path",
			noProxyList:  []string{"example.com"},
			shouldBypass: true,
		},
		{
			name:         "wildcard matches all",
			targetURL:    "http://anything.com/path",
			noProxyList:  []string{"*"},
			shouldBypass: true,
		},
		{
			name:         "no match - different domain",
			targetURL:    "http://other.com/path",
			noProxyList:  []string{"example.com"},
			shouldBypass: false,
		},
		{
			name:         "port-specific match",
			targetURL:    "http://example.com:8080/path",
			noProxyList:  []string{"example.com:8080"},
			shouldBypass: true,
		},
		{
			name:         "port-specific no match",
			targetURL:    "http://example.com:8080/path",
			noProxyList:  []string{"example.com:9090"},
			shouldBypass: false,
		},
		{
			name:         "empty no-proxy list",
			targetURL:    "http://example.com/path",
			noProxyList:  []string{},
			shouldBypass: false,
		},
		{
			name:         "localhost match",
			targetURL:    "http://localhost:8080/path",
			noProxyList:  []string{"localhost"},
			shouldBypass: true,
		},
		{
			name:         "IP address match",
			targetURL:    "http://192.168.1.1/path",
			noProxyList:  []string{"192.168.1.1"},
			shouldBypass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldBypassProxy(tt.targetURL, tt.noProxyList)
			assert.Equal(t, tt.shouldBypass, result, "unexpected bypass result")
		})
	}
}

func TestHTTPProxyApply(t *testing.T) {
	proxy := &HTTPProxy{
		Address:  "proxy.example.com:8080",
		Username: "user",
		Password: "pass",
		Timeout:  10 * time.Second,
	}

	transport := &http.Transport{}
	err := proxy.Apply(transport)

	require.NoError(t, err, "Apply should not return error")
	assert.NotNil(t, transport.Proxy, "Proxy function should be set")
	assert.NotNil(t, transport.DialContext, "DialContext should be set")
}

func TestHTTPProxyWithNoProxy(t *testing.T) {
	proxy := &HTTPProxy{
		Address: "proxy.example.com:8080",
		NoProxy: []string{"internal.local", ".corp.com"},
		Timeout: 10 * time.Second,
	}

	transport := &http.Transport{}
	err := proxy.Apply(transport)
	require.NoError(t, err)

	// Test that no-proxy domains are handled
	req1, _ := http.NewRequest("GET", "http://internal.local/api", nil)
	proxyURL1, err := transport.Proxy(req1)
	require.NoError(t, err)
	assert.Nil(t, proxyURL1, "internal.local should bypass proxy")

	req2, _ := http.NewRequest("GET", "http://external.com/api", nil)
	proxyURL2, err := transport.Proxy(req2)
	require.NoError(t, err)
	assert.NotNil(t, proxyURL2, "external.com should use proxy")
}

func TestSOCKS5ProxyApply(t *testing.T) {
	proxy := &SOCKS5Proxy{
		Address:  "127.0.0.1:1080",
		Username: "user",
		Password: "pass",
		Timeout:  10 * time.Second,
	}

	transport := &http.Transport{}
	err := proxy.Apply(transport)

	require.NoError(t, err, "Apply should not return error")
	assert.NotNil(t, transport.DialContext, "DialContext should be set")
}

func TestSOCKS5ProxyRequiresAddress(t *testing.T) {
	proxy := &SOCKS5Proxy{
		Username: "user",
		Password: "pass",
	}

	transport := &http.Transport{}
	err := proxy.Apply(transport)

	assert.Error(t, err, "Apply should return error when address is missing")
	assert.Contains(t, err.Error(), "address is required")
}

func TestNoProxyApply(t *testing.T) {
	proxy := &NoProxy{}
	transport := &http.Transport{}

	err := proxy.Apply(transport)

	require.NoError(t, err)
	assert.Nil(t, transport.Proxy, "Proxy should be nil for direct connection")
	assert.NotNil(t, transport.DialContext, "DialContext should be set")
}

func TestNewProxy(t *testing.T) {
	tests := []struct {
		name         string
		config       ProxyConfig
		expectedType string
		expectError  bool
	}{
		{
			name: "HTTP proxy",
			config: ProxyConfig{
				Type:    ProxyTypeHTTP,
				Address: "proxy.example.com:8080",
			},
			expectedType: "*proxy.HTTPProxy",
			expectError:  false,
		},
		{
			name: "SOCKS5 proxy",
			config: ProxyConfig{
				Type:    ProxyTypeSOCKS5,
				Address: "127.0.0.1:1080",
			},
			expectedType: "*proxy.SOCKS5Proxy",
			expectError:  false,
		},
		{
			name: "No proxy",
			config: ProxyConfig{
				Type: ProxyTypeNone,
			},
			expectedType: "*proxy.NoProxy",
			expectError:  false,
		},
		{
			name: "Invalid proxy type",
			config: ProxyConfig{
				Type: "INVALID",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy, err := NewProxy(tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, proxy)
			}
		})
	}
}

func TestNewTransport(t *testing.T) {
	config := ProxyConfig{
		Type:    ProxyTypeHTTP,
		Address: "proxy.example.com:8080",
		Timeout: 30 * time.Second,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		DisableKeepAlives: false,
		NoProxy:           []string{"localhost", ".internal"},
	}

	transport, err := NewTransport(config)

	require.NoError(t, err)
	assert.NotNil(t, transport)
	assert.Equal(t, config.TLSConfig, transport.TLSClientConfig)
	assert.Equal(t, config.DisableKeepAlives, transport.DisableKeepAlives)
	assert.Equal(t, 100, transport.MaxIdleConns)
}

func TestProxyConcurrentAccess(t *testing.T) {
	// Test thread-safety of proxy configuration
	proxy := &HTTPProxy{
		Address: "proxy.example.com:8080",
		Timeout: 10 * time.Second,
		NoProxy: []string{"internal.local"},
	}

	// Create multiple transports concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			transport := &http.Transport{}
			err := proxy.Apply(transport)
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestHTTPProxyWithHTTPSTarget(t *testing.T) {
	// This tests that HTTPS requests through HTTP proxy use CONNECT
	proxy := &HTTPProxy{
		Address: "proxy.example.com:8080",
		Timeout: 10 * time.Second,
	}

	transport := &http.Transport{}
	err := proxy.Apply(transport)
	require.NoError(t, err)

	// DialTLSContext should be set for HTTPS tunneling
	assert.NotNil(t, transport.DialTLSContext, "DialTLSContext should be set for HTTPS support")
}

func BenchmarkShouldBypassProxy(b *testing.B) {
	noProxyList := []string{"localhost", ".internal", "192.168.1.0/24", "*.local"}
	targetURL := "http://api.internal.company.com/endpoint"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ShouldBypassProxy(targetURL, noProxyList)
	}
}

func BenchmarkHTTPProxyApply(b *testing.B) {
	proxy := &HTTPProxy{
		Address: "proxy.example.com:8080",
		Timeout: 10 * time.Second,
		NoProxy: []string{"localhost", ".internal"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		transport := &http.Transport{}
		proxy.Apply(transport)
	}
}
