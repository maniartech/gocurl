package gocurl_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
	"github.com/maniartech/gocurl/middlewares"
	"github.com/maniartech/gocurl/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcess(t *testing.T) {
	t.Run("Basic GET request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			fmt.Fprint(w, "Hello, World!")
		}))
		defer server.Close()

		opts := &options.RequestOptions{
			URL: server.URL,
		}

		resp, body, err := gocurl.Process(context.Background(), opts)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "Hello, World!", body)
	})

	t.Run("Output to file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "File content")
		}))
		defer server.Close()

		tempFile, err := ioutil.TempFile("", "gocurl-test-")
		require.NoError(t, err)
		tempFile.Close()
		defer os.Remove(tempFile.Name())

		opts := &options.RequestOptions{
			URL:        server.URL,
			OutputFile: tempFile.Name(),
		}

		_, _, err = gocurl.Process(context.Background(), opts)
		require.NoError(t, err)

		content, err := ioutil.ReadFile(tempFile.Name())
		require.NoError(t, err)
		assert.Equal(t, "File content", string(content))
	})

	t.Run("Custom headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "TestValue", r.Header.Get("X-Test-Header"))
			fmt.Fprint(w, "Header received")
		}))
		defer server.Close()

		opts := &options.RequestOptions{
			URL: server.URL,
			Headers: http.Header{
				"X-Test-Header": []string{"TestValue"},
			},
		}

		resp, body, err := gocurl.Process(context.Background(), opts)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.NoError(t, err)
		assert.Equal(t, "Header received", body)
	})

	t.Run("Basic authentication", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			assert.True(t, ok)

			if username != "testuser" || password != "testpass" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			assert.Equal(t, "testuser", username)
			assert.Equal(t, "testpass", password)
			fmt.Fprint(w, "Authenticated")
		}))
		defer server.Close()

		opts := &options.RequestOptions{
			URL: server.URL,
			BasicAuth: &options.BasicAuth{
				Username: "testuser",
				Password: "testpass",
			},
		}

		resp, body, err := gocurl.Process(context.Background(), opts)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "Authenticated", body)
	})

	t.Run("Output to file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "File content")
		}))
		defer server.Close()

		tempFile, err := os.CreateTemp("", "gocurl-test-")
		require.NoError(t, err)
		tempFile.Close()
		defer os.Remove(tempFile.Name())

		opts := &options.RequestOptions{
			URL:        server.URL,
			OutputFile: tempFile.Name(),
		}

		_, _, err = gocurl.Process(context.Background(), opts)
		require.NoError(t, err)

		content, err := ioutil.ReadFile(tempFile.Name())
		require.NoError(t, err)
		assert.Equal(t, "File content", string(content))
	})

	t.Run("Retry mechanism", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			fmt.Fprint(w, "Success after retries")
		}))
		defer server.Close()

		opts := &options.RequestOptions{
			URL: server.URL,
			RetryConfig: &options.RetryConfig{
				MaxRetries:  3,
				RetryDelay:  time.Millisecond,
				RetryOnHTTP: []int{500},
			},
		}

		resp, body, err := gocurl.Process(context.Background(), opts)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, "Success after retries", body)
		assert.Equal(t, 3, attempts)
	})

	// Add more test cases for other features like file uploads, middleware, etc.
}

func TestValidateOptions(t *testing.T) {
	t.Run("Valid options", func(t *testing.T) {
		opts := &options.RequestOptions{
			URL: "https://example.com",
		}
		err := gocurl.ValidateOptions(opts)
		assert.NoError(t, err)
	})

	t.Run("Missing URL", func(t *testing.T) {
		opts := &options.RequestOptions{}
		err := gocurl.ValidateOptions(opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "URL is required")
	})

	// Add more validation test cases
}

func TestCreateHTTPClient(t *testing.T) {
	t.Run("Default client", func(t *testing.T) {
		ctx := context.Background()
		opts := &options.RequestOptions{}
		client, err := gocurl.CreateHTTPClient(ctx, opts)
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("Custom timeout without context deadline", func(t *testing.T) {
		ctx := context.Background()
		opts := &options.RequestOptions{
			Timeout: 5 * time.Second,
		}
		client, err := gocurl.CreateHTTPClient(ctx, opts)
		assert.NoError(t, err)
		assert.Equal(t, 5*time.Second, client.Timeout)
	})

	t.Run("Context with deadline takes priority", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		opts := &options.RequestOptions{
			Timeout: 5 * time.Second, // Should be ignored
		}
		client, err := gocurl.CreateHTTPClient(ctx, opts)
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
		req, err := gocurl.CreateRequest(context.Background(), opts)
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
		req, err := gocurl.CreateRequest(context.Background(), opts)
		assert.NoError(t, err)
		assert.Equal(t, "POST", req.Method)
		body, _ := ioutil.ReadAll(req.Body)
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
		req, err := gocurl.CreateRequest(context.Background(), opts)
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
		req, err := gocurl.CreateRequest(context.Background(), opts)
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
		req, err := gocurl.CreateRequest(context.Background(), opts)
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
		req, err := gocurl.CreateRequest(context.Background(), opts)
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
		modifiedReq, err := gocurl.ApplyMiddleware(req, []middlewares.MiddlewareFunc{middleware})

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
		modifiedReq, err := gocurl.ApplyMiddleware(req, []middlewares.MiddlewareFunc{middleware1, middleware2})

		assert.NoError(t, err)
		assert.Equal(t, "Applied1", modifiedReq.Header.Get("X-Middleware-1"))
		assert.Equal(t, "Applied2", modifiedReq.Header.Get("X-Middleware-2"))
	})

	t.Run("Middleware with error", func(t *testing.T) {
		middleware := func(req *http.Request) (*http.Request, error) {
			return nil, fmt.Errorf("middleware error")
		}

		req, _ := http.NewRequest("GET", "https://example.com", nil)
		_, err := gocurl.ApplyMiddleware(req, []middlewares.MiddlewareFunc{middleware})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "middleware error")
	})
}

func TestExecuteRequestWithRetries(t *testing.T) {
	t.Run("Successful request without retries", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &http.Client{}
		req, _ := http.NewRequest("GET", server.URL, nil)
		opts := &options.RequestOptions{}

		resp, err := gocurl.ExecuteRequestWithRetries(client, req, opts)

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

		resp, err := gocurl.ExecuteRequestWithRetries(client, req, opts)

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

		resp, err := gocurl.ExecuteRequestWithRetries(client, req, opts)

		assert.NoError(t, err) // The function doesn't return an error for HTTP errors
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestHandleOutput(t *testing.T) {
	t.Run("Output to file", func(t *testing.T) {
		tempFile, err := ioutil.TempFile("", "gocurl-test-")
		require.NoError(t, err)
		tempFile.Close()
		defer os.Remove(tempFile.Name())

		resp := &http.Response{
			Body: ioutil.NopCloser(strings.NewReader("Test output")),
		}
		opts := &options.RequestOptions{
			OutputFile: tempFile.Name(),
		}

		body, err := ioutil.ReadAll(resp.Body)
		assert.NoError(t, err)

		err = gocurl.HandleOutput(string(body), opts)
		assert.NoError(t, err)

		content, err := ioutil.ReadFile(tempFile.Name())
		assert.NoError(t, err)
		assert.Equal(t, "Test output", string(content))
	})

	t.Run("Output to stdout", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		resp := &http.Response{
			Body: ioutil.NopCloser(strings.NewReader("Test output")),
		}
		opts := &options.RequestOptions{}

		body, err := ioutil.ReadAll(resp.Body)
		assert.NoError(t, err)

		err = gocurl.HandleOutput(string(body), opts)
		assert.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		assert.Equal(t, "Test output", buf.String())
	})

	t.Run("Silent output", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		resp := &http.Response{
			Body: ioutil.NopCloser(strings.NewReader("Test output")),
		}
		opts := &options.RequestOptions{
			Silent: true,
		}

		body, err := ioutil.ReadAll(resp.Body)
		assert.NoError(t, err)

		err = gocurl.HandleOutput(string(body), opts)
		assert.NoError(t, err)

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		io.Copy(&buf, r)
		assert.Empty(t, buf.String())
	})
}
