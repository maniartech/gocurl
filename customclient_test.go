package gocurl

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/maniartech/gocurl/middlewares"
	"github.com/maniartech/gocurl/options"
)

// mockHTTPClient is a test implementation of HTTPClient
type mockHTTPClient struct {
	called       bool
	responseBody string
	statusCode   int
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.called = true
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(strings.NewReader(m.responseBody)),
		Header:     make(http.Header),
	}, nil
}

func TestCustomClient_IsUsedWhenSet(t *testing.T) {
	mock := &mockHTTPClient{
		responseBody: `{"custom": "client response"}`,
		statusCode:   200,
	}

	opts := &options.RequestOptions{
		URL:          "https://example.com",
		CustomClient: mock,
	}

	ctx := context.Background()
	resp, body, err := Process(ctx, opts)

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if !mock.called {
		t.Error("CustomClient.Do() was not called - CustomClient is not being used!")
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}

	if !strings.Contains(body, "custom") {
		t.Errorf("Expected body to contain 'custom', got: %s", body)
	}
}

func TestCustomClient_NotUsedWhenNil(t *testing.T) {
	t.Skip("Skipping network test - verifies standard client creation when CustomClient is nil")

	// This test verifies that when CustomClient is nil,
	// the standard CreateHTTPClient is used instead
	opts := &options.RequestOptions{
		URL:          "https://httpbin.org/get",
		CustomClient: nil, // Explicitly nil
	}

	ctx := context.Background()
	resp, _, err := Process(ctx, opts)

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	// If we get a real response from httpbin.org, it means standard client was used
	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}
}

func TestCustomClient_WithRetries(t *testing.T) {
	mock := &mockHTTPClient{
		responseBody: `{"retried": true}`,
		statusCode:   200,
	}

	opts := &options.RequestOptions{
		URL:          "https://example.com",
		CustomClient: mock,
		RetryConfig: &options.RetryConfig{
			MaxRetries: 3,
		},
	}

	ctx := context.Background()
	resp, body, err := Process(ctx, opts)

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if !mock.called {
		t.Error("CustomClient.Do() was not called with retry options")
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}

	if !strings.Contains(body, "retried") {
		t.Errorf("Expected body to contain 'retried', got: %s", body)
	}
}

func TestCustomClient_WithMiddleware(t *testing.T) {
	mock := &mockHTTPClient{
		responseBody: `{"middleware": "test"}`,
		statusCode:   200,
	}

	middlewareCalled := false
	middleware := middlewares.MiddlewareFunc(func(req *http.Request) (*http.Request, error) {
		middlewareCalled = true
		req.Header.Set("X-Custom-Header", "from-middleware")
		return req, nil
	})

	opts := &options.RequestOptions{
		URL:          "https://example.com",
		CustomClient: mock,
		Middleware: []middlewares.MiddlewareFunc{
			middleware,
		},
	}

	ctx := context.Background()
	resp, _, err := Process(ctx, opts)

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if !mock.called {
		t.Error("CustomClient.Do() was not called with middleware")
	}

	if !middlewareCalled {
		t.Error("Middleware was not called when using CustomClient")
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}
}

// TestCustomClient_ClonePreservesReference ensures that cloning RequestOptions
// preserves the CustomClient reference (shallow copy behavior)
func TestCustomClient_ClonePreservesReference(t *testing.T) {
	mock := &mockHTTPClient{
		responseBody: `{"cloned": true}`,
		statusCode:   200,
	}

	opts := &options.RequestOptions{
		URL:          "https://example.com",
		CustomClient: mock,
	}

	cloned := opts.Clone()

	if cloned.CustomClient != opts.CustomClient {
		t.Error("Clone() should preserve CustomClient reference (shallow copy)")
	}

	// Verify cloned options work with the same mock client
	ctx := context.Background()
	resp, _, err := Process(ctx, cloned)

	if err != nil {
		t.Fatalf("Process with cloned options failed: %v", err)
	}

	if !mock.called {
		t.Error("CustomClient was not used from cloned options")
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}
}
