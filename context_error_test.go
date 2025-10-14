package gocurl_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContextError_DeadlineExceeded verifies proper error reporting for context deadline
func TestContextError_DeadlineExceeded(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	resp, err := gocurl.Get(ctx, server.URL, nil)

	// Verify error
	require.Error(t, err)
	assert.Nil(t, resp)

	// Verify it's a context deadline error
	assert.True(t, errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(err.Error(), "deadline exceeded") ||
		strings.Contains(err.Error(), "context deadline exceeded"),
		"Expected context.DeadlineExceeded, got: %v", err)
}

// TestContextError_Cancelled verifies proper error reporting for context cancellation
func TestContextError_Cancelled(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Start request in goroutine
	errChan := make(chan error, 1)
	go func() {
		_, err := gocurl.Get(ctx, server.URL, nil)
		errChan <- err
	}()

	// Cancel after 100ms
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for error
	select {
	case err := <-errChan:
		require.Error(t, err)
		// Verify it's a context canceled error
		assert.True(t, errors.Is(err, context.Canceled) ||
			strings.Contains(err.Error(), "context canceled") ||
			strings.Contains(err.Error(), "canceled"),
			"Expected context.Canceled, got: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Request didn't respect context cancellation")
	}
}

// TestContextError_WithRetries verifies context errors propagate through retries
func TestContextError_WithRetries(t *testing.T) {
	// Create server that always fails with 500
	var attemptCount int32 // Use atomic access
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		time.Sleep(500 * time.Millisecond) // Each attempt takes 500ms
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Context with timeout that will hit during retries
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()

	// Configure retries
	builder := options.NewRequestOptionsBuilder().
		Get(server.URL, nil).
		SetRetryConfig(&options.RetryConfig{
			MaxRetries: 5, // Try 5 times
			RetryDelay: 100 * time.Millisecond,
		}).
		WithContext(ctx)
	
	opts := builder.Build()
	builderCtx := builder.GetContext()
	defer builder.Cleanup()

	start := time.Now()
	resp, err := gocurl.Execute(builderCtx, opts)
	elapsed := time.Since(start)

	// Should fail with context deadline, not complete all retries
	require.Error(t, err)
	assert.Nil(t, resp)

	// Should timeout around 800ms, not wait for all 5 retries
	assert.True(t, elapsed < 2*time.Second,
		"Expected timeout around 800ms, took %v", elapsed)

	// Should have attempted 2 times max (first + one retry before timeout)
	count := atomic.LoadInt32(&attemptCount)
	assert.True(t, count <= 2,
		"Expected max 2 attempts before context timeout, got %d", count)

	// Verify it's a context error
	assert.True(t, errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(err.Error(), "deadline exceeded") ||
		strings.Contains(err.Error(), "context deadline exceeded"),
		"Expected context error, got: %v", err)
}

// TestContextError_CancelDuringRetry verifies context cancellation stops retries
func TestContextError_CancelDuringRetry(t *testing.T) {
	var attemptCount int32 // Use atomic access
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError) // Always fail to trigger retry
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Configure retries
	builder := options.NewRequestOptionsBuilder().
		Get(server.URL, nil).
		SetRetryConfig(&options.RetryConfig{
			MaxRetries: 10, // Many retries
			RetryDelay: 100 * time.Millisecond,
		}).
		WithContext(ctx)
	
	opts := builder.Build()
	builderCtx := builder.GetContext()
	defer builder.Cleanup()

	// Start request in goroutine
	errChan := make(chan error, 1)
	go func() {
		_, err := gocurl.Execute(builderCtx, opts)
		errChan <- err
	}()

	// Cancel after first attempt completes
	time.Sleep(400 * time.Millisecond)
	cancel()

	// Wait for error
	select {
	case err := <-errChan:
		require.Error(t, err)

		// Should have stopped early (max 2-3 attempts before cancel kicks in)
		count := atomic.LoadInt32(&attemptCount)
		assert.True(t, count <= 3,
			"Expected early stop due to cancel, got %d attempts", count)

		// Verify it's a context canceled error
		assert.True(t, errors.Is(err, context.Canceled) ||
			strings.Contains(err.Error(), "context canceled") ||
			strings.Contains(err.Error(), "canceled"),
			"Expected context.Canceled, got: %v", err)

	case <-time.After(5 * time.Second):
		t.Fatal("Request didn't stop after context cancellation")
	}
}

// TestContextError_PropagationThroughLayers verifies context errors propagate through all layers
func TestContextError_PropagationThroughLayers(t *testing.T) {
	// Create slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request context is already cancelled
		select {
		case <-r.Context().Done():
			// Context was cancelled, don't even respond
			return
		default:
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Test with Process function directly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	opts := options.NewRequestOptions(server.URL)
	_, _, err := gocurl.Process(ctx, opts)

	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(err.Error(), "deadline exceeded"),
		"Expected context error through Process(), got: %v", err)
}

// TestContextError_MultipleRequests_Independent verifies contexts don't interfere between requests
func TestContextError_MultipleRequests_Independent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// First request with short timeout - should fail
	ctx1, cancel1 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel1()

	resp1, err1 := gocurl.Get(ctx1, server.URL, nil)
	require.Error(t, err1)
	assert.Nil(t, resp1)

	// Second request with long timeout - should succeed
	ctx2, cancel2 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel2()

	resp2, err2 := gocurl.Get(ctx2, server.URL, nil)
	require.NoError(t, err2)
	require.NotNil(t, resp2)
	resp2.Body.Close()
}

// TestContextError_CheckBeforeRetry verifies context is checked before each retry attempt
func TestContextError_CheckBeforeRetry(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError) // Always fail
	}))
	defer server.Close()

	// Already expired context
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
	defer cancel()

	builder := options.NewRequestOptionsBuilder().
		Get(server.URL, nil).
		SetRetryConfig(&options.RetryConfig{
			MaxRetries: 5,
			RetryDelay: 100 * time.Millisecond,
		}).
		WithContext(ctx)
	
	opts := builder.Build()
	builderCtx := builder.GetContext()
	defer builder.Cleanup()

	_, err := gocurl.Execute(builderCtx, opts)

	// Should fail immediately without any attempts (context already expired)
	require.Error(t, err)
	assert.Equal(t, 0, attemptCount,
		"Expected no attempts with expired context, got %d", attemptCount)

	assert.True(t, errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(err.Error(), "deadline exceeded"),
		"Expected context error, got: %v", err)
}

// TestContextError_HTTPClientRespect verifies HTTP client respects context
func TestContextError_HTTPClientRespect(t *testing.T) {
	// Create server that delays in reading request body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to read body slowly
		buf := make([]byte, 1024)
		for {
			_, err := r.Body.Read(buf)
			if err != nil {
				break
			}
			time.Sleep(100 * time.Millisecond) // Slow read
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	// Send POST with body
	builder := options.NewRequestOptionsBuilder().
		SetMethod("POST").
		SetURL(server.URL).
		SetBody(strings.Repeat("x", 10000)). // Large body
		WithContext(ctx)
	
	opts := builder.Build()
	builderCtx := builder.GetContext()
	defer builder.Cleanup()

	start := time.Now()
	_, err := gocurl.Execute(builderCtx, opts)
	elapsed := time.Since(start)

	// Should timeout
	require.Error(t, err)
	assert.True(t, elapsed < 500*time.Millisecond,
		"Expected quick timeout, took %v", elapsed)
}
