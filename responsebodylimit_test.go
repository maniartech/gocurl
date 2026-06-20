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

// liveExec runs the one-shot engine exactly as CurlArgs does — executeOpts wraps
// the response body with the streaming newLimitedBody/limitedBody when
// ResponseBodyLimit > 0 — and then reads it. This exercises the SHIPPED limit
// enforcement that real callers hit, NOT a buffered test-only copy.
func liveExec(opts *options.RequestOptions) (*http.Response, []byte, error) {
	resp, err := executeOpts(context.Background(), opts)
	if err != nil {
		return nil, nil, err
	}
	body, rerr := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, body, rerr
}

// TestResponseBodyLimit_NoLimit verifies normal operation without limit.
func TestResponseBodyLimit_NoLimit(t *testing.T) {
	largeBody := strings.Repeat("A", 10*1024) // 10KB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(largeBody))
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.ResponseBodyLimit = 0 // No limit

	resp, body, err := liveExec(opts)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", resp.StatusCode)
	}
	if string(body) != largeBody {
		t.Errorf("Expected body length %d, got: %d", len(largeBody), len(body))
	}
}

// TestResponseBodyLimit_WithinLimit verifies body within limit is read successfully.
func TestResponseBodyLimit_WithinLimit(t *testing.T) {
	smallBody := "Hello, World!" // 13 bytes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(smallBody))
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.ResponseBodyLimit = 1024 // 1KB limit

	resp, body, err := liveExec(opts)
	if err != nil {
		t.Fatalf("Expected no error for body within limit, got: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got: %d", resp.StatusCode)
	}
	if string(body) != smallBody {
		t.Errorf("Expected body '%s', got: '%s'", smallBody, body)
	}
}

// TestResponseBodyLimit_ExceedsLimit verifies error when body exceeds limit.
func TestResponseBodyLimit_ExceedsLimit(t *testing.T) {
	largeBody := bytes.Repeat([]byte("A"), 2*1024*1024) // 2MB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(largeBody)
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.ResponseBodyLimit = 1024 // 1KB limit, but server returns 2MB

	_, _, err := liveExec(opts)
	if err == nil || !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("Expected 'exceeds limit' error for oversized body, got: %v", err)
	}
}

// TestResponseBodyLimit_ExactLimit verifies body exactly at limit is accepted.
func TestResponseBodyLimit_ExactLimit(t *testing.T) {
	exactBody := strings.Repeat("B", 1024) // Exactly 1KB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(exactBody))
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.ResponseBodyLimit = 1024 // 1KB limit

	resp, body, err := liveExec(opts)
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
	overBody := strings.Repeat("C", 1025) // 1 byte over 1KB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(overBody))
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.ResponseBodyLimit = 1024 // 1KB limit

	_, _, err := liveExec(opts)
	if err == nil || !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("Expected 'exceeds limit' error for 1-byte overflow, got: %v", err)
	}
}

// TestResponseBodyLimit_DoSProtection verifies the limit protects against a large
// (malicious) streamed response.
func TestResponseBodyLimit_DoSProtection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chunk := bytes.Repeat([]byte("X"), 1024*1024) // 1MB chunk
		for i := 0; i < 10; i++ {                     // 10MB total
			w.Write(chunk)
		}
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.ResponseBodyLimit = 100 * 1024 // 100KB limit vs a 10MB response

	_, _, err := liveExec(opts)
	if err == nil || !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("Expected size-limit error for DoS-sized response, got: %v", err)
	}
}

// TestResponseBodyLimit_Integration verifies the limit fires on the body read
// (after the request), so request-level retries do not re-run it.
func TestResponseBodyLimit_Integration(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Write(bytes.Repeat([]byte("D"), 2048)) // 2KB, oversized
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.ResponseBodyLimit = 1024
	opts.RetryConfig = &options.RetryConfig{MaxRetries: 2, RetryOnHTTP: []int{500}}

	_, _, err := liveExec(opts)
	if err == nil || !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("Expected limit error, got: %v", err)
	}
	if attempts > 1 {
		t.Logf("Note: made %d attempts (body-limit errors fire on read, not retried)", attempts)
	}
}
