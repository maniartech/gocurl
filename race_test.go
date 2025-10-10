package gocurl_test

import (
	"sync"
	"testing"

	"github.com/maniartech/gocurl"
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
