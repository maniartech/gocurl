package gocurl

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/maniartech/gocurl/options"
)

// TestRequestOptions_ConcurrentCloneIsSafe verifies that Clone() prevents race conditions
// when modifying options concurrently.
func TestRequestOptions_ConcurrentCloneIsSafe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.Headers = make(http.Header)

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Clone before modification - this SHOULD be safe
			cloned := opts.Clone()
			cloned.AddHeader("X-Request-ID", fmt.Sprintf("req-%d", id))
			cloned.AddQueryParam("id", fmt.Sprintf("%d", id))
			cloned.Form.Add("test", fmt.Sprintf("value-%d", id))

			// Execute request with cloned options
			_, _, err := Process(context.Background(), cloned)
			if err != nil {
				t.Errorf("Request failed for goroutine %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
	// Should pass with -race flag: go test -race -run TestRequestOptions_ConcurrentCloneIsSafe
}

// TestRequestOptions_ConcurrentReadsAreSafe verifies concurrent reads are safe.
func TestRequestOptions_ConcurrentReadsAreSafe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.Headers = make(http.Header)
	opts.SetHeader("Authorization", "Bearer token123")
	opts.SetHeader("User-Agent", "GoCurl/1.0")
	opts.Form = make(map[string][]string)
	opts.Form.Add("key", "value")
	opts.Timeout = 5000000000 // 5 seconds

	var wg sync.WaitGroup
	numReaders := 50

	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// All these reads should be safe
			_ = opts.URL
			_ = opts.Method
			_ = opts.Timeout
			_ = opts.Headers.Get("Authorization")
			_ = opts.Headers.Get("User-Agent")
			_ = opts.Form.Get("key")

			// Execute request (reads options)
			_, _, err := Process(context.Background(), opts)
			if err != nil {
				t.Errorf("Request failed for reader %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
	// Should pass with -race flag: go test -race -run TestRequestOptions_ConcurrentReadsAreSafe
}

// TestRequestOptions_ConcurrentHeaderWrites_DetectsRace demonstrates the race condition
// when modifying Headers concurrently without Clone().
// This test is designed to FAIL with the race detector to prove the warning is correct.
//
// Run with: go test -race -run TestRequestOptions_ConcurrentHeaderWrites_DetectsRace
//
// Expected: "WARNING: DATA RACE" when run with -race flag
func TestRequestOptions_ConcurrentHeaderWrites_DetectsRace(t *testing.T) {
	t.Skip("⚠️  Skipping: This test intentionally triggers race conditions. Run manually with: go test -race -run TestRequestOptions_ConcurrentHeaderWrites_DetectsRace")

	t.Log("⚠️  This test demonstrates UNSAFE concurrent map writes")
	t.Log("⚠️  Run with -race flag to see the race detector catch this")

	opts := options.NewRequestOptions("https://example.com")
	opts.Headers = make(http.Header)

	var wg sync.WaitGroup
	numWriters := 10

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// ❌ UNSAFE: Concurrent writes to the same map
			// This SHOULD trigger race detector
			opts.Headers.Add("X-ID", fmt.Sprintf("%d", id))
			opts.Headers.Set("X-Concurrent", fmt.Sprintf("value-%d", id))
		}(i)
	}

	wg.Wait()

	t.Log("✅ If you see 'WARNING: DATA RACE' above, the race detector is working correctly")
	t.Log("✅ This proves Headers requires Clone() before concurrent modification")
}

// TestRequestOptions_ConcurrentFormWrites_DetectsRace demonstrates race in Form field.
func TestRequestOptions_ConcurrentFormWrites_DetectsRace(t *testing.T) {
	t.Skip("⚠️  Skipping: This test intentionally triggers race conditions. Run manually with: go test -race -run TestRequestOptions_ConcurrentFormWrites_DetectsRace")

	t.Log("⚠️  This test demonstrates UNSAFE concurrent Form writes")

	opts := options.NewRequestOptions("https://example.com")
	opts.Form = make(map[string][]string)

	var wg sync.WaitGroup
	numWriters := 10

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// ❌ UNSAFE: Concurrent writes to Form map
			opts.Form.Add("field", fmt.Sprintf("value-%d", id))
			opts.Form.Set("id", fmt.Sprintf("%d", id))
		}(i)
	}

	wg.Wait()

	t.Log("✅ Race detector should catch concurrent Form map writes")
}

// TestRequestOptions_ConcurrentQueryParamWrites_DetectsRace demonstrates race in QueryParams.
func TestRequestOptions_ConcurrentQueryParamWrites_DetectsRace(t *testing.T) {
	t.Skip("⚠️  Skipping: This test intentionally triggers race conditions. Run with GOCURL_RACE_TESTS=1 to enable")

	t.Log("⚠️  This test demonstrates UNSAFE concurrent QueryParams writes")

	opts := options.NewRequestOptions("https://example.com")
	opts.QueryParams = make(map[string][]string)

	var wg sync.WaitGroup
	numWriters := 10

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// ❌ UNSAFE: Concurrent writes to QueryParams map
			opts.QueryParams.Add("param", fmt.Sprintf("value-%d", id))
			opts.QueryParams.Set("id", fmt.Sprintf("%d", id))
		}(i)
	}

	wg.Wait()

	t.Log("✅ Race detector should catch concurrent QueryParams map writes")
}

// TestRequestOptions_BuilderConcurrentContextIsSafe verifies builder pattern is thread-safe.
func TestRequestOptions_BuilderConcurrentContextIsSafe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}))
	defer server.Close()

	var wg sync.WaitGroup
	numBuilders := 50

	for i := 0; i < numBuilders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine creates its own options
			opts := options.NewRequestOptions(server.URL)
			opts.Headers = make(http.Header)
			opts.SetHeader("X-Request-ID", fmt.Sprintf("req-%d", id))

			_, _, err := Process(context.Background(), opts)
			if err != nil {
				t.Errorf("Request failed for goroutine %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
	// Should pass with -race flag
}
