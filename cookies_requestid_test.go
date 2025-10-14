package gocurl

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/maniartech/gocurl/options"
)

// TestCookies_SingleCookie verifies that a single cookie is properly added to the request
func TestCookies_SingleCookie(t *testing.T) {
	// Create a test server that echoes back cookies
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		if len(cookies) != 1 {
			t.Errorf("Expected 1 cookie, got %d", len(cookies))
			return
		}
		if cookies[0].Name != "session" || cookies[0].Value != "abc123" {
			t.Errorf("Expected cookie session=abc123, got %s=%s", cookies[0].Name, cookies[0].Value)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create request with single cookie
	opts := options.NewRequestOptions(server.URL)
	opts.Cookies = []*http.Cookie{
		{Name: "session", Value: "abc123"},
	}

	// Execute request
	_, err := Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

// TestCookies_MultipleCookies verifies that multiple cookies are properly added
func TestCookies_MultipleCookies(t *testing.T) {
	expectedCookies := map[string]string{
		"session": "abc123",
		"user":    "john",
		"theme":   "dark",
		"lang":    "en",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		if len(cookies) != len(expectedCookies) {
			t.Errorf("Expected %d cookies, got %d", len(expectedCookies), len(cookies))
		}

		receivedCookies := make(map[string]string)
		for _, cookie := range cookies {
			receivedCookies[cookie.Name] = cookie.Value
		}

		for name, value := range expectedCookies {
			if receivedCookies[name] != value {
				t.Errorf("Cookie %s: expected %s, got %s", name, value, receivedCookies[name])
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.Cookies = []*http.Cookie{
		{Name: "session", Value: "abc123"},
		{Name: "user", Value: "john"},
		{Name: "theme", Value: "dark"},
		{Name: "lang", Value: "en"},
	}

	_, err := Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

// TestCookies_EmptyArray verifies that empty cookie array doesn't cause issues
func TestCookies_EmptyArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		if len(cookies) != 0 {
			t.Errorf("Expected 0 cookies, got %d", len(cookies))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.Cookies = []*http.Cookie{} // Empty array

	_, err := Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

// TestCookies_NilArray verifies that nil cookie array doesn't cause panic
func TestCookies_NilArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.Cookies = nil // Nil array

	_, err := Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

// TestCookies_WithCookieJar verifies Cookies field works alongside CookieJar
func TestCookies_WithCookieJar(t *testing.T) {
	// First request sets a cookie via Set-Cookie header
	// Second request should send both the jar cookie AND the Cookies field cookie

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		if callCount == 1 {
			// First request: set a cookie via Set-Cookie
			http.SetCookie(w, &http.Cookie{Name: "jar_cookie", Value: "from_jar"})
			w.WriteHeader(http.StatusOK)
		} else {
			// Second request: verify both cookies present
			cookies := r.Cookies()

			foundJarCookie := false
			foundManualCookie := false

			for _, cookie := range cookies {
				if cookie.Name == "jar_cookie" && cookie.Value == "from_jar" {
					foundJarCookie = true
				}
				if cookie.Name == "manual_cookie" && cookie.Value == "from_opts" {
					foundManualCookie = true
				}
			}

			if !foundJarCookie {
				t.Error("Cookie from jar not found in second request")
			}
			if !foundManualCookie {
				t.Error("Cookie from Cookies field not found in second request")
			}

			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	// Create jar
	jar, _ := cookiejar.New(nil)

	// First request - populates jar
	opts1 := options.NewRequestOptions(server.URL)
	opts1.CookieJar = jar
	_, err := Execute(context.Background(), opts1)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}

	// Second request - uses jar + manual cookie
	opts2 := options.NewRequestOptions(server.URL)
	opts2.CookieJar = jar
	opts2.Cookies = []*http.Cookie{
		{Name: "manual_cookie", Value: "from_opts"},
	}
	_, err = Execute(context.Background(), opts2)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
}

// TestCookies_ConcurrentSafe verifies concurrent requests with cookies are safe
func TestCookies_ConcurrentSafe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			opts := options.NewRequestOptions(server.URL)
			opts.Cookies = []*http.Cookie{
				{Name: "id", Value: fmt.Sprintf("%d", id)}, // Use valid cookie value
			}

			_, err := Execute(context.Background(), opts)
			if err != nil {
				t.Errorf("Goroutine %d failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
}

// TestRequestID_Added verifies X-Request-ID header is added when set
func TestRequestID_Added(t *testing.T) {
	expectedID := "req-12345-abcde"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID != expectedID {
			t.Errorf("Expected X-Request-ID: %s, got: %s", expectedID, requestID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.RequestID = expectedID

	_, err := Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

// TestRequestID_Empty verifies header is not added when RequestID is empty
func TestRequestID_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID != "" {
			t.Errorf("Expected no X-Request-ID header, got: %s", requestID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.RequestID = "" // Empty string

	_, err := Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

// TestRequestID_UUIDFormat verifies UUID format works correctly
func TestRequestID_UUIDFormat(t *testing.T) {
	// Common UUID v4 format
	expectedID := "550e8400-e29b-41d4-a716-446655440000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID != expectedID {
			t.Errorf("Expected X-Request-ID: %s, got: %s", expectedID, requestID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.RequestID = expectedID

	_, err := Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}

// TestRequestID_ConcurrentSafe verifies concurrent requests with different IDs
func TestRequestID_ConcurrentSafe(t *testing.T) {
	receivedIDs := sync.Map{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID != "" {
			receivedIDs.Store(requestID, true)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			opts := options.NewRequestOptions(server.URL)
			opts.RequestID = fmt.Sprintf("req-%d", id) // Use valid string format

			_, err := Execute(context.Background(), opts)
			if err != nil {
				t.Errorf("Goroutine %d failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify we received unique IDs
	count := 0
	receivedIDs.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	if count != numGoroutines {
		t.Errorf("Expected %d unique request IDs, got %d", numGoroutines, count)
	}
}

// TestRequestID_OverridesExisting verifies RequestID field takes precedence
func TestRequestID_OverridesExisting(t *testing.T) {
	expectedID := "override-id"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID != expectedID {
			t.Errorf("Expected X-Request-ID: %s, got: %s", expectedID, requestID)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	opts := options.NewRequestOptions(server.URL)
	opts.Headers = make(http.Header)
	opts.Headers.Set("X-Request-ID", "will-be-overridden")
	opts.RequestID = expectedID // Should override the header

	_, err := Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}
