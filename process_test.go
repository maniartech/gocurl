package gocurl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/maniartech/gocurl/middlewares"
	"github.com/maniartech/gocurl/options"
	"github.com/stretchr/testify/assert"
)

func TestValidateOptions(t *testing.T) {
	t.Run("Valid options", func(t *testing.T) {
		opts := &options.RequestOptions{
			URL: "https://example.com",
		}
		err := validateOptions(opts)
		assert.NoError(t, err)
	})

	t.Run("Missing URL", func(t *testing.T) {
		opts := &options.RequestOptions{}
		err := validateOptions(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "URL is required")
	})

	// Add more validation test cases
}

func TestCreateHTTPClient(t *testing.T) {
	t.Run("Default client", func(t *testing.T) {
		ctx := context.Background()
		opts := &options.RequestOptions{}
		client, err := createHTTPClient(ctx, opts)
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("Custom timeout without context deadline", func(t *testing.T) {
		ctx := context.Background()
		opts := &options.RequestOptions{
			Timeout: 5 * time.Second,
		}
		client, err := createHTTPClient(ctx, opts)
		assert.NoError(t, err)
		assert.Equal(t, 5*time.Second, client.Timeout)
	})

	t.Run("Context with deadline takes priority", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		opts := &options.RequestOptions{
			Timeout: 5 * time.Second, // Should be ignored
		}
		client, err := createHTTPClient(ctx, opts)
		assert.NoError(t, err)
		// When context has deadline, client.Timeout should be 0 (context controls)
		assert.Equal(t, time.Duration(0), client.Timeout)
	})

	// Add more client creation test cases
}

func TestCreateRequest(t *testing.T) {
	t.Run("GET request", func(t *testing.T) {
		opts := &options.RequestOptions{
			URL: "https://example.com",
		}
		req, err := createRequest(context.Background(), opts)
		assert.NoError(t, err)
		assert.Equal(t, "GET", req.Method)
		assert.Equal(t, "https://example.com", req.URL.String())
	})

	t.Run("POST request with body", func(t *testing.T) {
		opts := &options.RequestOptions{
			Method: "POST",
			URL:    "https://example.com",
			Body:   "test data",
		}
		req, err := createRequest(context.Background(), opts)
		assert.NoError(t, err)
		assert.Equal(t, "POST", req.Method)
		body, _ := io.ReadAll(req.Body)
		assert.Equal(t, "test data", string(body))
	})

	t.Run("Request with query parameters", func(t *testing.T) {
		opts := &options.RequestOptions{
			URL: "https://example.com",
			QueryParams: url.Values{
				"key1": []string{"value1"},
				"key2": []string{"value2"},
			},
		}
		req, err := createRequest(context.Background(), opts)
		assert.NoError(t, err)
		assert.Contains(t, req.URL.String(), "key1=value1")
		assert.Contains(t, req.URL.String(), "key2=value2")
	})

	t.Run("Request with custom headers", func(t *testing.T) {
		opts := &options.RequestOptions{
			URL: "https://example.com",
			Headers: http.Header{
				"X-Custom-Header": []string{"CustomValue"},
			},
		}
		req, err := createRequest(context.Background(), opts)
		assert.NoError(t, err)
		assert.Equal(t, "CustomValue", req.Header.Get("X-Custom-Header"))
	})

	t.Run("Request with basic auth", func(t *testing.T) {
		opts := &options.RequestOptions{
			URL: "https://example.com",
			BasicAuth: &options.BasicAuth{
				Username: "user",
				Password: "pass",
			},
		}
		req, err := createRequest(context.Background(), opts)
		assert.NoError(t, err)
		username, password, ok := req.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "user", username)
		assert.Equal(t, "pass", password)
	})

	t.Run("Request with bearer token", func(t *testing.T) {
		opts := &options.RequestOptions{
			URL:         "https://example.com",
			BearerToken: "token123",
		}
		req, err := createRequest(context.Background(), opts)
		assert.NoError(t, err)
		assert.Equal(t, "Bearer token123", req.Header.Get("Authorization"))
	})
}

func TestApplyMiddleware(t *testing.T) {
	t.Run("Single middleware", func(t *testing.T) {
		middleware := func(req *http.Request) (*http.Request, error) {
			req.Header.Set("X-Middleware", "Applied")
			return req, nil
		}

		req, _ := http.NewRequest("GET", "https://example.com", nil)
		modifiedReq, err := applyMiddleware(req, []middlewares.MiddlewareFunc{middleware})

		assert.NoError(t, err)
		assert.Equal(t, "Applied", modifiedReq.Header.Get("X-Middleware"))
	})

	t.Run("Multiple middleware", func(t *testing.T) {
		middleware1 := func(req *http.Request) (*http.Request, error) {
			req.Header.Set("X-Middleware-1", "Applied1")
			return req, nil
		}
		middleware2 := func(req *http.Request) (*http.Request, error) {
			req.Header.Set("X-Middleware-2", "Applied2")
			return req, nil
		}

		req, _ := http.NewRequest("GET", "https://example.com", nil)
		modifiedReq, err := applyMiddleware(req, []middlewares.MiddlewareFunc{middleware1, middleware2})

		assert.NoError(t, err)
		assert.Equal(t, "Applied1", modifiedReq.Header.Get("X-Middleware-1"))
		assert.Equal(t, "Applied2", modifiedReq.Header.Get("X-Middleware-2"))
	})

	t.Run("Middleware with error", func(t *testing.T) {
		middleware := func(req *http.Request) (*http.Request, error) {
			return nil, fmt.Errorf("middleware error")
		}

		req, _ := http.NewRequest("GET", "https://example.com", nil)
		_, err := applyMiddleware(req, []middlewares.MiddlewareFunc{middleware})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "middleware error")
	})
}

// TestExecuteRequestWithRetries is deprecated - ExecuteRequestWithRetries function was removed
// Retry logic is now tested in retry_test.go
/*
func TestExecuteRequestWithRetries(t *testing.T) {
	t.Run("Successful request without retries", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &http.Client{}
		req, _ := http.NewRequest("GET", server.URL, nil)
		opts := &options.RequestOptions{}

		resp, err := ExecuteRequestWithRetries(client, req, opts)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Successful request after retries", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		client := &http.Client{}
		req, _ := http.NewRequest("GET", server.URL, nil)
		opts := &options.RequestOptions{
			RetryConfig: &options.RetryConfig{
				MaxRetries:  3,
				RetryDelay:  time.Millisecond,
				RetryOnHTTP: []int{500},
			},
		}

		resp, err := ExecuteRequestWithRetries(client, req, opts)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, 3, attempts)
	})

	t.Run("Request fails after max retries", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := &http.Client{}
		req, _ := http.NewRequest("GET", server.URL, nil)
		opts := &options.RequestOptions{
			RetryConfig: &options.RetryConfig{
				MaxRetries:  2,
				RetryDelay:  time.Millisecond,
				RetryOnHTTP: []int{500},
			},
		}

		resp, err := ExecuteRequestWithRetries(client, req, opts)

		assert.NoError(t, err) // The function doesn't return an error for HTTP errors
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}
*/
