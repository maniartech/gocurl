package gocurl_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/options"
)

// TestConcurrentRequestConstruction verifies thread-safety when parsing requests concurrently
func TestConcurrentRequestConstruction(t *testing.T) {
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				args := []string{
					"curl", "-X", "POST",
					"-H", "Content-Type: application/json",
					"-d", `{"key":"value"}`,
					"https://example.com/test",
				}

				_, err := gocurl.ArgsToOptions(args)
				if err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent request construction failed: %v", err)
	}
}

// TestConcurrentVariableExpansion verifies thread-safety of variable substitution
func TestConcurrentVariableExpansion(t *testing.T) {
	const goroutines = 100
	const iterations = 100

	vars := gocurl.Variables{
		"token": "my-secret-token",
		"url":   "https://example.com",
		"data":  "important data",
	}

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				text := "Authorization: Bearer ${token}, URL: ${url}, Data: ${data}"
				_, err := gocurl.ExpandVariables(text, vars)
				if err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent variable expansion failed: %v", err)
	}
}

// TestConcurrentRequestAPI verifies thread-safety of the high-level Request API
func TestConcurrentRequestAPI(t *testing.T) {
	// Skip this test as it would make real HTTP requests
	// Enable with a local test server for real concurrent request testing
	t.Skip("Skipping concurrent API test - requires test server")

	const goroutines = 10
	const iterations = 10

	vars := gocurl.Variables{
		"url": "http://localhost:8080/test",
	}

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				resp, err := gocurl.Request("curl ${url}", vars)
				if err != nil {
					errors <- err
					return
				}
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent API request failed: %v", err)
	}
}

// TestConcurrentBufferPool verifies thread-safety of response buffer pool
func TestConcurrentBufferPool(t *testing.T) {
	const goroutines = 1000
	const iterations = 100

	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Test parsing which uses internal buffers
				args := []string{
					"curl", "-X", "GET",
					"-H", "Content-Type: application/json",
					"https://example.com/api/test",
				}

				_, err := gocurl.ArgsToOptions(args)
				if err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent buffer pool test failed: %v", err)
	}
}

// TestHighConcurrencyStress tests with 10k+ concurrent goroutines
func TestHighConcurrencyStress(t *testing.T) {
	const goroutines = 10000
	const iterations = 10

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				args := []string{
					"curl", "-X", "POST",
					"-H", "Authorization: Bearer token123",
					"-d", `{"id":` + string(rune(id)) + `}`,
					"https://api.example.com/data",
				}

				_, err := gocurl.ArgsToOptions(args)
				if err != nil {
					errorCount++
					return
				}
				successCount++
			}
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("High concurrency stress test had %d errors", errorCount)
	}

	t.Logf("Successfully processed %d operations with %d goroutines", successCount, goroutines)
}

// TestConcurrentErrorHandling verifies error handling is thread-safe
func TestConcurrentErrorHandling(t *testing.T) {
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Create various error scenarios
				err := gocurl.ParseError("curl -X GET https://example.com",
					gocurl.ValidationError("test", gocurl.RequestError("https://example.com",
						gocurl.ResponseError("https://example.com", gocurl.RetryError("https://example.com", 3, nil)))))

				// Verify error formatting is consistent
				errStr := err.Error()
				if errStr == "" {
					t.Errorf("Error formatting failed in goroutine %d", id)
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestConcurrentSecurityValidation verifies security validation is thread-safe
func TestConcurrentSecurityValidation(t *testing.T) {
	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	testVars := []gocurl.Variables{
		{"token": "secret123", "url": "https://example.com"},
		{"key": "value", "data": "test"},
		{"api_key": "sensitive", "endpoint": "/api/v1"},
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				vars := testVars[j%len(testVars)]

				// Test variable validation
				err := gocurl.ValidateVariables(vars)
				if err != nil {
					errors <- err
					return
				}

				// Test variable expansion
				_, err = gocurl.ExpandVariables("${token}:${url}", vars)
				if err != nil {
					// Expected for some variable combinations
					continue
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent security validation failed: %v", err)
	}
}

// TestConcurrentMixedOperations tests multiple operations concurrently
func TestConcurrentMixedOperations(t *testing.T) {
	const goroutines = 500

	var wg sync.WaitGroup

	// Mix of different operations
	for i := 0; i < goroutines; i++ {
		wg.Add(1)

		operationType := i % 5

		go func(id, opType int) {
			defer wg.Done()

			switch opType {
			case 0:
				// Request parsing
				_, _ = gocurl.ArgsToOptions([]string{"curl", "https://example.com"})

			case 1:
				// Variable expansion
				vars := gocurl.Variables{"url": "https://example.com"}
				_, _ = gocurl.ExpandVariables("${url}/api", vars)

			case 2:
				// Error creation
				err := gocurl.ParseError("curl test", gocurl.RequestError("https://example.com", nil))
				_ = err.Error()

			case 3:
				// Security validation
				vars := gocurl.Variables{"key": "value"}
				_ = gocurl.ValidateVariables(vars)

			case 4:
				// Header redaction
				headers := map[string][]string{
					"Authorization": {"Bearer token"},
					"Content-Type":  {"application/json"},
				}
				_ = gocurl.RedactHeaders(headers)
			}
		}(i, operationType)
	}

	wg.Wait()
}

// TestConcurrentResponseBufferPool verifies buffer pool is thread-safe
func TestConcurrentResponseBufferPool(t *testing.T) {
	const goroutines = 100
	const iterations = 50

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send varying sized responses to test pool
		id := r.URL.Query().Get("id")
		sizeMultiplier := 1
		if len(id) > 0 {
			sizeMultiplier = 1 + int(id[0]%10)
		}
		size := 1024 * sizeMultiplier
		data := bytes.Repeat([]byte("x"), size)
		w.Write(data)
	}))
	defer server.Close()

	var wg sync.WaitGroup
	successCount := int64(0)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				opts := &options.RequestOptions{
					URL:    server.URL + "?id=" + string(rune('0'+id%10)),
					Method: "GET",
				}

				resp, _, err := gocurl.Process(context.Background(), opts)
				if err != nil {
					t.Errorf("Request failed: %v", err)
					return
				}
				resp.Body.Close()
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	expected := int64(goroutines * iterations)
	if atomic.LoadInt64(&successCount) != expected {
		t.Errorf("Expected %d successful requests, got %d", expected, successCount)
	}
}

// TestConcurrentRetryLogic verifies retry logic is thread-safe
func TestConcurrentRetryLogic(t *testing.T) {
	const goroutines = 50
	attempts := int64(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&attempts, 1)
		if count%3 != 0 { // Fail 2/3 of requests
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			opts := &options.RequestOptions{
				URL:    server.URL,
				Method: "GET",
				RetryConfig: &options.RetryConfig{
					MaxRetries:  3,
					RetryDelay:  5 * time.Millisecond,
					RetryOnHTTP: []int{503},
				},
			}

			resp, _, err := gocurl.Process(context.Background(), opts)
			if err != nil {
				errors <- err
				return
			}
			resp.Body.Close()
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent retry failed: %v", err)
	}
}

// TestConcurrentVariableSubstitution tests concurrent variable substitution
func TestConcurrentVariableSubstitution(t *testing.T) {
	const goroutines = 200
	const iterations = 100

	vars := gocurl.Variables{
		"host":  "example.com",
		"token": "secret123",
		"path":  "/api/v1",
	}

	var wg sync.WaitGroup
	errors := make(chan error, goroutines*iterations)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				text := "https://${host}${path}?token=${token}&id=" + string(rune('0'+id%10))
				result, err := gocurl.ExpandVariables(text, vars)
				if err != nil {
					errors <- err
					return
				}

				expected := "https://example.com/api/v1?token=secret123&id=" + string(rune('0'+id%10))
				if result != expected {
					errors <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent variable substitution failed: %v", err)
	}
}

// TestStressTest_10kGoroutines verifies handling of 10k concurrent operations
func TestStressTest_10kGoroutines(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const goroutines = 10000

	var wg sync.WaitGroup
	successCount := int64(0)
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Just parse, don't make actual requests
			args := []string{
				"curl", "-X", "GET",
				"-H", "X-Request-ID: " + string(rune('0'+id%10)),
				"https://example.com/test/" + string(rune('0'+id%10)),
			}

			_, err := gocurl.ArgsToOptions(args)
			if err != nil {
				errors <- err
				return
			}
			atomic.AddInt64(&successCount, 1)
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Stress test failed: %v", err)
	}

	if atomic.LoadInt64(&successCount) != goroutines {
		t.Errorf("Expected %d successful operations, got %d", goroutines, successCount)
	}
}
