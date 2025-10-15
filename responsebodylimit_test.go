package gocurl

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maniartech/gocurl/options"
)

// TestResponseBodyLimit_NoLimit verifies normal operation without limit.
func TestResponseBodyLimit_NoLimit(t *testing.T) {
	largeBody := strings.Repeat("A", 10*1024) // 10KB

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeBody))
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.Silent = true // Prevent large bodies from spamming stdout
	opts.ResponseBodyLimit = 0 // No limit

	resp, body, err := Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", resp.StatusCode)
	}

	if body != largeBody {
		t.Errorf("Expected body length %d, got: %d", len(largeBody), len(body))
	}
}

// TestResponseBodyLimit_WithinLimit verifies body within limit is read successfully.
func TestResponseBodyLimit_WithinLimit(t *testing.T) {
	smallBody := "Hello, World!" // 13 bytes

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(smallBody))
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.Silent = true // Prevent large bodies from spamming stdout
	opts.ResponseBodyLimit = 1024 // 1KB limit

	resp, body, err := Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Expected no error for body within limit, got: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", resp.StatusCode)
	}

	if body != smallBody {
		t.Errorf("Expected body '%s', got: '%s'", smallBody, body)
	}
}

// TestResponseBodyLimit_ExceedsLimit verifies error when body exceeds limit.
func TestResponseBodyLimit_ExceedsLimit(t *testing.T) {
	largeBody := bytes.Repeat([]byte("A"), 2*1024*1024) // 2MB

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", string(rune(len(largeBody))))
		w.WriteHeader(http.StatusOK)
		w.Write(largeBody)
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.Silent = true // Prevent large bodies from spamming stdout
	opts.ResponseBodyLimit = 1024 // 1KB limit, but server returns 2MB

	_, _, err := Process(context.Background(), opts)
	if err == nil {
		t.Fatal("Expected error for body exceeding limit, got nil")
	}

	expectedErrMsg := "exceeds limit"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error containing '%s', got: %v", expectedErrMsg, err)
	}

	t.Logf("✅ Correctly rejected oversized response: %v", err)
}

// TestResponseBodyLimit_ExactLimit verifies body exactly at limit is accepted.
func TestResponseBodyLimit_ExactLimit(t *testing.T) {
	exactBody := strings.Repeat("B", 1024) // Exactly 1KB

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(exactBody))
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.Silent = true // Prevent large bodies from spamming stdout
	opts.ResponseBodyLimit = 1024 // 1KB limit
	opts.Silent = true            // Don't print 1KB of 'B' characters to stdout

	resp, body, err := Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Expected no error for body at exact limit, got: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", resp.StatusCode)
	}

	if len(body) != 1024 {
		t.Errorf("Expected body length 1024, got: %d", len(body))
	}
}

// TestResponseBodyLimit_OneByteOver verifies one byte over limit triggers error.
func TestResponseBodyLimit_OneByteOver(t *testing.T) {
	overBody := strings.Repeat("C", 1025) // 1025 bytes (1 byte over 1KB)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(overBody))
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.Silent = true // Prevent large bodies from spamming stdout
	opts.ResponseBodyLimit = 1024 // 1KB limit

	_, _, err := Process(context.Background(), opts)
	if err == nil {
		t.Fatal("Expected error for body 1 byte over limit, got nil")
	}

	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Errorf("Expected 'exceeds limit' error, got: %v", err)
	}

	t.Logf("✅ Correctly rejected 1-byte overflow: %v", err)
}

// TestResponseBodyLimit_DoSProtection verifies limit protects against large responses.
func TestResponseBodyLimit_DoSProtection(t *testing.T) {
	// Redirect verbose output to prevent overwhelming terminal with 10MB of 'X' characters
	oldVerboseWriter := VerboseWriter
	VerboseWriter = io.Discard
	defer func() { VerboseWriter = oldVerboseWriter }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		// Simulate malicious server sending huge response
		// Write 10MB in chunks
		chunk := bytes.Repeat([]byte("X"), 1024*1024) // 1MB chunk
		for i := 0; i < 10; i++ {
			w.Write(chunk)
		}
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.Silent = true // Prevent large bodies from spamming stdout
	opts.ResponseBodyLimit = 100 * 1024 // 100KB limit (protect against 10MB response)

	_, _, err := Process(context.Background(), opts)
	if err == nil {
		t.Fatal("Expected error for DoS-sized response, got nil")
	}

	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Errorf("Expected size limit error, got: %v", err)
	}

	t.Logf("✅ DoS protection working: %v", err)
}

// TestResponseBodyLimit_Integration verifies limit works with retries.
func TestResponseBodyLimit_Integration(t *testing.T) {
	// Redirect verbose output to prevent terminal spam
	oldVerboseWriter := VerboseWriter
	VerboseWriter = io.Discard
	defer func() { VerboseWriter = oldVerboseWriter }()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		// Always return oversized response
		w.Write(bytes.Repeat([]byte("D"), 2048)) // 2KB
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.Silent = true // Prevent large bodies from spamming stdout
	opts.ResponseBodyLimit = 1024 // 1KB limit
	opts.RetryConfig = &options.RetryConfig{
		MaxRetries:  2,
		RetryOnHTTP: []int{500}, // Don't retry on our error
	}

	_, _, err := Process(context.Background(), opts)
	if err == nil {
		t.Fatal("Expected error for oversized response, got nil")
	}

	// Should fail on first attempt (body size error not retried)
	if attempts > 1 {
		t.Logf("Note: Made %d attempts (body limit errors may not trigger retry)", attempts)
	}

	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Errorf("Expected limit error, got: %v", err)
	}
}
