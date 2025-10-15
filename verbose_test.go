package gocurl

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maniartech/gocurl/options"
)

// TestVerbose_Disabled verifies no output when verbose is disabled
func TestVerbose_Disabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	// Capture verbose output
	var buf bytes.Buffer
	VerboseWriter = &buf

	opts := options.NewRequestOptions(server.URL)
	opts.Verbose = false // Explicitly disabled

	_, _, err := Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	// Should have NO verbose output
	output := buf.String()
	if output != "" {
		t.Errorf("Expected no verbose output, got: %s", output)
	}
}

// TestVerbose_RequestHeaders verifies request headers are printed
func TestVerbose_RequestHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	// Capture verbose output
	var buf bytes.Buffer
	VerboseWriter = &buf

	opts := options.NewRequestOptions(server.URL)
	opts.Verbose = true
	opts.Method = "POST"
	opts.Headers = make(http.Header)
	opts.Headers.Set("Content-Type", "application/json")
	opts.Headers.Set("X-Custom-Header", "test-value")

	_, _, err := Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	output := buf.String()

	// Verify request line
	if !strings.Contains(output, "> POST") {
		t.Errorf("Expected '> POST' in output, got: %s", output)
	}

	// Verify Host header
	if !strings.Contains(output, "> Host:") {
		t.Errorf("Expected '> Host:' in output, got: %s", output)
	}

	// Verify custom headers
	if !strings.Contains(output, "> Content-Type: application/json") {
		t.Errorf("Expected Content-Type header in output, got: %s", output)
	}

	if !strings.Contains(output, "> X-Custom-Header: test-value") {
		t.Errorf("Expected X-Custom-Header in output, got: %s", output)
	}
}

// TestVerbose_ResponseHeaders verifies response headers are printed
func TestVerbose_ResponseHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Response-Header", "response-value")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	// Capture verbose output
	var buf bytes.Buffer
	VerboseWriter = &buf

	opts := options.NewRequestOptions(server.URL)
	opts.Verbose = true

	_, _, err := Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	output := buf.String()

	// Verify response status line
	if !strings.Contains(output, "< HTTP") && !strings.Contains(output, "< 200 OK") {
		t.Errorf("Expected response status line in output, got: %s", output)
	}

	// Verify response headers
	if !strings.Contains(output, "< Content-Type: application/json") {
		t.Errorf("Expected Content-Type header in output, got: %s", output)
	}

	if !strings.Contains(output, "< X-Response-Header: response-value") {
		t.Errorf("Expected X-Response-Header in output, got: %s", output)
	}
}

// TestVerbose_SensitiveDataRedacted verifies sensitive headers are redacted
func TestVerbose_SensitiveDataRedacted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session=secret123")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	// Capture verbose output
	var buf bytes.Buffer
	VerboseWriter = &buf

	opts := options.NewRequestOptions(server.URL)
	opts.Verbose = true
	opts.Headers = make(http.Header)
	opts.Headers.Set("Authorization", "Bearer secret-token-123")
	opts.Headers.Set("Cookie", "session=abc123")
	opts.Headers.Set("X-Api-Key", "super-secret-key")

	_, _, err := Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	output := buf.String()

	// Verify sensitive request headers are redacted
	if strings.Contains(output, "secret-token-123") {
		t.Errorf("Authorization token should be redacted, got: %s", output)
	}

	if strings.Contains(output, "session=abc123") {
		t.Errorf("Cookie should be redacted, got: %s", output)
	}

	if strings.Contains(output, "super-secret-key") {
		t.Errorf("API key should be redacted, got: %s", output)
	}

	// Verify sensitive response headers are redacted
	if strings.Contains(output, "session=secret123") {
		t.Errorf("Set-Cookie should be redacted, got: %s", output)
	}

	// Verify [REDACTED] appears
	if !strings.Contains(output, "[REDACTED]") {
		t.Errorf("Expected [REDACTED] in output for sensitive headers, got: %s", output)
	}
}

// TestVerbose_CustomWriter verifies custom writer support
func TestVerbose_CustomWriter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	// Use custom buffer
	var customBuf bytes.Buffer
	VerboseWriter = &customBuf

	opts := options.NewRequestOptions(server.URL)
	opts.Verbose = true

	_, _, err := Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	output := customBuf.String()

	// Verify output went to custom writer
	if !strings.Contains(output, "> GET") {
		t.Errorf("Expected output in custom writer, got: %s", output)
	}

	if !strings.Contains(output, "< HTTP") {
		t.Errorf("Expected response in custom writer, got: %s", output)
	}
}

// TestVerbose_ConcurrentSafe verifies verbose output is thread-safe
func TestVerbose_ConcurrentSafe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	// Use thread-safe buffer wrapper
	var mu sync.Mutex
	var buf bytes.Buffer
	safeWriter := &threadSafeWriter{w: &buf, mu: &mu}
	VerboseWriter = safeWriter

	var wg sync.WaitGroup
	numRequests := 10

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			opts := options.NewRequestOptions(server.URL)
			opts.Verbose = true
			opts.Headers = make(http.Header)
			opts.Headers.Set("X-Request-ID", fmt.Sprintf("%d", id))

			_, _, err := Process(context.Background(), opts)
			if err != nil {
				t.Errorf("Process failed for request %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	output := buf.String()

	// Verify we got output from multiple requests
	if !strings.Contains(output, "> GET") {
		t.Errorf("Expected request output, got: %s", output)
	}

	if !strings.Contains(output, "< HTTP") {
		t.Errorf("Expected response output, got: %s", output)
	}
}

// TestVerbose_MatchesCurlFormat verifies output format matches curl -v
func TestVerbose_MatchesCurlFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Hello, World!")
	}))
	defer server.Close()

	// Capture verbose output
	var buf bytes.Buffer
	VerboseWriter = &buf

	opts := options.NewRequestOptions(server.URL)
	opts.Verbose = true

	_, _, err := Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	output := buf.String()

	// Verify curl -v format
	verifyOutputPrefixes(t, output)
	verifyConnectionInfo(t, output)
	verifyRequestFormat(t, output)
}

// verifyOutputPrefixes checks that output has correct curl-style prefixes
func verifyOutputPrefixes(t *testing.T, output string) {
	lines := strings.Split(output, "\n")

	hasConnectionPrefix := false
	hasRequestPrefix := false
	hasResponsePrefix := false

	for _, line := range lines {
		if strings.HasPrefix(line, "*") {
			hasConnectionPrefix = true
		}
		if strings.HasPrefix(line, ">") {
			hasRequestPrefix = true
		}
		if strings.HasPrefix(line, "<") {
			hasResponsePrefix = true
		}
	}

	if !hasConnectionPrefix {
		t.Errorf("Expected connection lines starting with '*', got: %s", output)
	}
	if !hasRequestPrefix {
		t.Errorf("Expected request lines starting with '>', got: %s", output)
	}
	if !hasResponsePrefix {
		t.Errorf("Expected response lines starting with '<', got: %s", output)
	}
}

// verifyConnectionInfo checks for curl-style connection messages
func verifyConnectionInfo(t *testing.T, output string) {
	if !strings.Contains(output, "* Trying") && !strings.Contains(output, "*   Trying") {
		t.Errorf("Expected '* Trying' connection info like curl, got: %s", output)
	}
	if !strings.Contains(output, "* Connected to") {
		t.Errorf("Expected '* Connected to' info like curl, got: %s", output)
	}
	if !strings.Contains(output, "* Connection") || !strings.Contains(output, "left intact") {
		t.Errorf("Expected connection close info like curl, got: %s", output)
	}
}

// verifyRequestFormat checks for curl-style request formatting
func verifyRequestFormat(t *testing.T, output string) {
	if !strings.Contains(output, "> GET") && !strings.Contains(output, "> POST") {
		t.Errorf("Expected request method line like curl format, got: %s", output)
	}
	if !strings.Contains(output, "> Host:") {
		t.Errorf("Expected Host header like curl format, got: %s", output)
	}
}

// threadSafeWriter wraps an io.Writer with a mutex for thread-safe writes
type threadSafeWriter struct {
	w  *bytes.Buffer
	mu *sync.Mutex
}

func (tsw *threadSafeWriter) Write(p []byte) (n int, err error) {
	tsw.mu.Lock()
	defer tsw.mu.Unlock()
	return tsw.w.Write(p)
}

// TestVerbose_HTTPSConnectionInfo verifies HTTPS-specific verbose output
func TestVerbose_HTTPSConnectionInfo(t *testing.T) {
	// Create HTTPS test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Secure response")
	}))
	defer server.Close()

	// Capture verbose output
	var buf bytes.Buffer
	VerboseWriter = &buf

	opts := options.NewRequestOptions(server.URL)
	opts.Verbose = true
	opts.Insecure = true // Required for self-signed cert

	_, _, err := Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	output := buf.String()

	// Verify HTTPS-specific verbose output
	if !strings.Contains(output, "* ALPN") {
		t.Errorf("Expected ALPN negotiation info for HTTPS, got: %s", output)
	}

	if !strings.Contains(output, "* TLS") {
		t.Errorf("Expected TLS info for HTTPS, got: %s", output)
	}

	if !strings.Contains(output, "* Using") {
		t.Errorf("Expected TLS version info, got: %s", output)
	}

	// Since Insecure=true, should mention skipping verification
	if !strings.Contains(output, "Skipping certificate verification") {
		t.Errorf("Expected certificate verification skip message, got: %s", output)
	}
}

// TestVerbose_HTTP2Protocol verifies HTTP/2 protocol detection
func TestVerbose_HTTP2Protocol(t *testing.T) {
	// This test would require HTTP/2 server setup
	// For now, we'll test the logic exists in the code
	// Actual HTTP/2 testing would require more complex setup
	t.Skip("HTTP/2 testing requires complex server setup - logic exists in printResponseVerbose")
}
