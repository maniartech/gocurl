package proxy_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maniartech/gocurl/proxy"
)

// Mock HTTP Proxy Server
func startHTTPProxy() *httptest.Server {
	proxyHandler := func(w http.ResponseWriter, r *http.Request) {
		// Simulate proxy behavior: Check the URL being requested
		if r.URL.String() == "http://example.com/" {
			w.Header().Set("X-Proxied", "true")
			fmt.Fprintf(w, "Request to example.com was proxied")
		} else {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(w, "Unexpected request: %s", r.URL.String())
		}
	}
	proxyServer := httptest.NewServer(http.HandlerFunc(proxyHandler))
	return proxyServer
}

func TestHTTPProxyApplyWithRealProxy(t *testing.T) {
	// Start a mock HTTP proxy server
	httpProxyServer := startHTTPProxy()
	defer httpProxyServer.Close()

	// Extract the proxy address (remove "http://")
	proxyAddress := httpProxyServer.URL[7:]

	// Create the HTTPProxy with the mock server's address
	transport := &http.Transport{}
	httpProxy := &proxy.HTTPProxy{
		Address: proxyAddress,
	}

	// Apply the HTTPProxy
	err := httpProxy.Apply(transport)
	if err != nil {
		t.Fatalf("Failed to apply HTTPProxy: %v", err)
	}

	// Create an HTTP client using the configured transport
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	// Send a request to example.com through the proxy
	// Note: Using HTTP (not HTTPS) because httptest.Server doesn't support CONNECT method
	resp, err := client.Get("http://example.com")
	if err != nil {
		t.Fatalf("Failed to make request through HTTP proxy: %v", err)
	}
	defer resp.Body.Close()

	// Check if the request was proxied by verifying the "X-Proxied" header
	if resp.Header.Get("X-Proxied") != "true" {
		t.Error("Expected request to be proxied to example.com, but it was not")
	}
}
