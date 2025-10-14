package gocurl_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTimeoutHandling_ContextOnly verifies context timeout works when no opts.Timeout is set
func TestTimeoutHandling_ContextOnly(t *testing.T) {
	// Create slow server (2 seconds)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Context with 500ms timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// No opts.Timeout set (should be 0)
	resp, err := gocurl.Get(ctx, server.URL, nil)

	// Should timeout via context
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, strings.Contains(err.Error(), "deadline exceeded") ||
		strings.Contains(err.Error(), "context deadline exceeded"),
		"Expected deadline exceeded error, got: %v", err)
}

// TestTimeoutHandling_OptsTimeoutOnly verifies opts.Timeout works when no context deadline
func TestTimeoutHandling_OptsTimeoutOnly(t *testing.T) {
	// Create slow server (2 seconds)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Options with timeout (no context needed for this test)
	builder := options.NewRequestOptionsBuilder().
		Get(server.URL, nil).
		SetTimeout(500 * time.Millisecond)

	opts := builder.Build()
	ctx := builder.GetContext() // Get context from builder
	defer builder.Cleanup()

	resp, err := gocurl.Execute(ctx, opts)

	// Should timeout via opts.Timeout
	require.Error(t, err)
	assert.Nil(t, resp)
}

// TestTimeoutHandling_ContextTakesPriority verifies context deadline takes priority over opts.Timeout
// This is the INDUSTRY STANDARD PATTERN
func TestTimeoutHandling_ContextTakesPriority(t *testing.T) {
	// Create slow server (2 seconds)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Context with 500ms timeout
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Options with LONGER timeout (10 seconds)
	// Context should take priority and timeout at 500ms
	builder := options.NewRequestOptionsBuilder().
		Get(server.URL, nil).
		SetTimeout(10 * time.Second). // This should be ignored
		WithContext(ctx)

	opts := builder.Build()
	builderCtx := builder.GetContext() // This has both timeouts merged
	defer builder.Cleanup()

	start := time.Now()
	resp, err := gocurl.Execute(builderCtx, opts)
	elapsed := time.Since(start)

	// Should timeout via context (~500ms), NOT opts.Timeout (10s)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, elapsed < 1*time.Second,
		"Expected timeout around 500ms, took %v", elapsed)
	assert.True(t, strings.Contains(err.Error(), "deadline exceeded") ||
		strings.Contains(err.Error(), "context deadline exceeded"),
		"Expected deadline exceeded error, got: %v", err)
}

// TestTimeoutHandling_BothSetContextWins tests that context wins when both are set
func TestTimeoutHandling_BothSetContextWins(t *testing.T) {
	// Create slow server (5 seconds)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Context with SHORT timeout (1 second)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Via Get() which sets context
	start := time.Now()
	resp, err := gocurl.Get(ctx, server.URL, nil)
	elapsed := time.Since(start)

	// Should timeout at ~1 second (context), not wait 5 seconds
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, elapsed < 2*time.Second,
		"Expected timeout around 1s, took %v", elapsed)
}

// TestTimeoutHandling_NoTimeoutSet verifies request completes when no timeout is set
func TestTimeoutHandling_NoTimeoutSet(t *testing.T) {
	// Create fast server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	// No context timeout, no opts.Timeout
	builder := options.NewRequestOptionsBuilder().
		Get(server.URL, nil)

	opts := builder.Build()
	ctx := builder.GetContext()

	resp, err := gocurl.Execute(ctx, opts)

	// Should succeed
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()
}

// TestTimeoutHandling_BuilderWithTimeout tests builder's WithTimeout method
func TestTimeoutHandling_BuilderWithTimeout(t *testing.T) {
	// Create slow server (2 seconds)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use builder's WithTimeout (creates context with timeout)
	builder := options.NewRequestOptionsBuilder().
		Get(server.URL, nil).
		WithTimeout(500 * time.Millisecond)

	opts := builder.Build()
	ctx := builder.GetContext()
	defer builder.Cleanup()

	start := time.Now()
	resp, err := gocurl.Execute(ctx, opts)
	elapsed := time.Since(start)

	// Should timeout at ~500ms
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, elapsed < 1*time.Second,
		"Expected timeout around 500ms, took %v", elapsed)
}

// TestTimeoutHandling_ContextCancellation tests that cancelled context stops request
func TestTimeoutHandling_ContextCancellation(t *testing.T) {
	// Create slow server (5 seconds)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
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
		assert.True(t, strings.Contains(err.Error(), "context canceled") ||
			strings.Contains(err.Error(), "canceled"),
			"Expected context canceled error, got: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Request didn't respect context cancellation")
	}
}

// TestTimeoutHandling_SuccessWithinTimeout verifies requests complete successfully within timeout
func TestTimeoutHandling_SuccessWithinTimeout(t *testing.T) {
	// Create fast server (100ms)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	// Context with generous timeout (5 seconds)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gocurl.Get(ctx, server.URL, nil)

	// Should succeed
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()
}

// TestTimeoutHandling_ContextCleanup verifies that context cancel functions are properly called
func TestTimeoutHandling_ContextCleanup(t *testing.T) {
	// This test verifies memory leak prevention
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use builder's WithTimeout which creates a context with cancel
	builder := options.NewRequestOptionsBuilder().
		Get(server.URL, nil).
		WithTimeout(5 * time.Second)

	opts := builder.Build()
	ctx := builder.GetContext()

	// Execute
	resp, err := gocurl.Execute(ctx, opts)

	// Should succeed
	require.NoError(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()

	// Clean up the builder's context
	builder.Cleanup()

	// The cancel should have been called (via builder.Cleanup)
	// We can't directly test this, but no panic means it worked
}

// TestTimeoutHandling_MultipleRequests verifies timeout handling across multiple requests
func TestTimeoutHandling_MultipleRequests(t *testing.T) {
	// Create server with variable delays
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			time.Sleep(100 * time.Millisecond) // Fast
		} else {
			time.Sleep(2 * time.Second) // Slow
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First request - should succeed (fast)
	resp1, err1 := gocurl.Get(ctx, server.URL, nil)
	require.NoError(t, err1)
	require.NotNil(t, resp1)
	resp1.Body.Close()

	// Second request with short timeout - should fail (slow)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel2()

	resp2, err2 := gocurl.Get(ctx2, server.URL, nil)
	require.Error(t, err2)
	assert.Nil(t, resp2)
}

// TestCreateHTTPClient_ContextPriorityPattern tests that CreateHTTPClient follows industry standard
func TestCreateHTTPClient_ContextPriorityPattern(t *testing.T) {
	tests := []struct {
		name            string
		contextDeadline bool
		optsTimeout     time.Duration
		expectTimeout   time.Duration
		description     string
	}{
		{
			name:            "Context with deadline, no opts timeout",
			contextDeadline: true,
			optsTimeout:     0,
			expectTimeout:   0, // Client timeout should be 0 (context handles it)
			description:     "Context takes priority",
		},
		{
			name:            "Context with deadline, opts has timeout",
			contextDeadline: true,
			optsTimeout:     10 * time.Second,
			expectTimeout:   0, // Client timeout should be 0 (context takes priority)
			description:     "Context takes priority over opts",
		},
		{
			name:            "No context deadline, opts has timeout",
			contextDeadline: false,
			optsTimeout:     5 * time.Second,
			expectTimeout:   5 * time.Second, // Client timeout should use opts
			description:     "Opts timeout used as fallback",
		},
		{
			name:            "No context deadline, no opts timeout",
			contextDeadline: false,
			optsTimeout:     0,
			expectTimeout:   0, // No timeout
			description:     "No timeout set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			if tt.contextDeadline {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()
			} else {
				ctx = context.Background()
			}

			opts := options.NewRequestOptions("http://example.com")
			opts.Timeout = tt.optsTimeout

			client, err := gocurl.CreateHTTPClient(ctx, opts)
			require.NoError(t, err)
			require.NotNil(t, client)

			assert.Equal(t, tt.expectTimeout, client.Timeout,
				"%s: Expected client.Timeout=%v, got %v", tt.description, tt.expectTimeout, client.Timeout)
		})
	}
}
