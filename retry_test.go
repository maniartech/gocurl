package gocurl_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/options"
)

// TestRetryLogic_GET verifies retry works for GET requests
func TestRetryLogic_GET(t *testing.T) {
	attempts := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Service unavailable"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	}))
	defer server.Close()

	opts := &options.RequestOptions{
		URL:    server.URL,
		Method: "GET",
		RetryConfig: &options.RetryConfig{
			MaxRetries:  3,
			RetryDelay:  10 * time.Millisecond,
			RetryOnHTTP: []int{503},
		},
	}

	resp, _, err := gocurl.Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestRetryLogic_POST verifies retry works for POST requests with bodies
func TestRetryLogic_POST(t *testing.T) {
	attempts := int32(0)
	receivedBodies := []string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBodies = append(receivedBodies, string(body))

		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("Bad gateway"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	}))
	defer server.Close()

	testBody := "key=value&data=test"

	opts := &options.RequestOptions{
		URL:    server.URL,
		Method: "POST",
		Body:   testBody,
		RetryConfig: &options.RetryConfig{
			MaxRetries:  3,
			RetryDelay:  10 * time.Millisecond,
			RetryOnHTTP: []int{502},
		},
	}

	resp, _, err := gocurl.Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	// Verify body was sent correctly on all attempts
	for i, body := range receivedBodies {
		if body != testBody {
			t.Errorf("Attempt %d: expected body %q, got %q", i+1, testBody, body)
		}
	}
}

// TestRetryLogic_PUT verifies retry works for PUT requests with bodies
func TestRetryLogic_PUT(t *testing.T) {
	attempts := int32(0)
	receivedBodies := []string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBodies = append(receivedBodies, string(body))

		count := atomic.AddInt32(&attempts, 1)
		if count < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal error"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Updated"))
	}))
	defer server.Close()

	testBody := `{"id": 123, "name": "updated"}`

	opts := &options.RequestOptions{
		URL:    server.URL,
		Method: "PUT",
		Body:   testBody,
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
		RetryConfig: &options.RetryConfig{
			MaxRetries:  2,
			RetryDelay:  10 * time.Millisecond,
			RetryOnHTTP: []int{500},
		},
	}

	resp, _, err := gocurl.Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}

	// Verify body was sent correctly on all attempts
	for i, body := range receivedBodies {
		if body != testBody {
			t.Errorf("Attempt %d: expected body %q, got %q", i+1, testBody, body)
		}
	}
}

// TestRetryLogic_NoRetryNeeded verifies single successful request doesn't retry
func TestRetryLogic_NoRetryNeeded(t *testing.T) {
	attempts := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	}))
	defer server.Close()

	opts := &options.RequestOptions{
		URL:    server.URL,
		Method: "GET",
		RetryConfig: &options.RetryConfig{
			MaxRetries:  3,
			RetryDelay:  10 * time.Millisecond,
			RetryOnHTTP: []int{500, 502, 503},
		},
	}

	resp, _, err := gocurl.Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("Expected 1 attempt (no retries), got %d", attempts)
	}
}

// TestRetryLogic_DefaultRetryBehavior verifies default retry on common transient errors
func TestRetryLogic_DefaultRetryBehavior(t *testing.T) {
	testCases := []struct {
		name        string
		statusCode  int
		shouldRetry bool
	}{
		{"429 Too Many Requests", http.StatusTooManyRequests, true},
		{"500 Internal Server Error", http.StatusInternalServerError, true},
		{"502 Bad Gateway", http.StatusBadGateway, true},
		{"503 Service Unavailable", http.StatusServiceUnavailable, true},
		{"504 Gateway Timeout", http.StatusGatewayTimeout, true},
		{"404 Not Found", http.StatusNotFound, false},
		{"400 Bad Request", http.StatusBadRequest, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attempts := int32(0)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&attempts, 1)
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			opts := &options.RequestOptions{
				URL:    server.URL,
				Method: "GET",
				RetryConfig: &options.RetryConfig{
					MaxRetries: 2,
					RetryDelay: 10 * time.Millisecond,
					// Empty RetryOnHTTP means use defaults
				},
			}

			resp, _, err := gocurl.Process(context.Background(), opts)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			expectedAttempts := int32(1)
			if tc.shouldRetry {
				expectedAttempts = 3 // 1 initial + 2 retries
			}

			if atomic.LoadInt32(&attempts) != expectedAttempts {
				t.Errorf("Expected %d attempts, got %d", expectedAttempts, attempts)
			}
		})
	}
}

// TestRetryLogic_ExponentialBackoff verifies exponential backoff when no delay specified
func TestRetryLogic_ExponentialBackoff(t *testing.T) {
	attempts := int32(0)
	timestamps := []time.Time{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamps = append(timestamps, time.Now())
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := &options.RequestOptions{
		URL:    server.URL,
		Method: "GET",
		RetryConfig: &options.RetryConfig{
			MaxRetries:  2,
			RetryDelay:  0, // Use default exponential backoff
			RetryOnHTTP: []int{503},
		},
	}

	start := time.Now()
	resp, _, err := gocurl.Process(context.Background(), opts)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should have at least some delay between retries (100ms + 200ms = 300ms minimum)
	if elapsed < 200*time.Millisecond {
		t.Errorf("Expected at least 200ms elapsed with backoff, got %v", elapsed)
	}

	if len(timestamps) == 3 {
		// First retry should be ~100ms after initial
		delay1 := timestamps[1].Sub(timestamps[0])
		if delay1 < 80*time.Millisecond || delay1 > 150*time.Millisecond {
			t.Logf("First retry delay: %v (expected ~100ms)", delay1)
		}

		// Second retry should be ~200ms after first retry
		delay2 := timestamps[2].Sub(timestamps[1])
		if delay2 < 180*time.Millisecond || delay2 > 250*time.Millisecond {
			t.Logf("Second retry delay: %v (expected ~200ms)", delay2)
		}
	}
}

// TestRetryLogic_LargeBody verifies retry works with larger request bodies
func TestRetryLogic_LargeBody(t *testing.T) {
	attempts := int32(0)
	largeBody := strings.Repeat("abcdefghij", 10000) // 100KB body

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		if string(body) != largeBody {
			t.Errorf("Body mismatch on attempt %d", atomic.LoadInt32(&attempts)+1)
		}

		count := atomic.AddInt32(&attempts, 1)
		if count < 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := &options.RequestOptions{
		URL:    server.URL,
		Method: "POST",
		Body:   largeBody,
		RetryConfig: &options.RetryConfig{
			MaxRetries:  2,
			RetryDelay:  10 * time.Millisecond,
			RetryOnHTTP: []int{502},
		},
	}

	resp, _, err := gocurl.Process(context.Background(), opts)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}
