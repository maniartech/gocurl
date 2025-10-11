package gocurl

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/maniartech/gocurl/options"
)

// ExecuteWithRetries handles HTTP request execution with retry logic.
// It properly clones requests with bodies to enable retries for POST/PUT requests.
func ExecuteWithRetries(client *http.Client, req *http.Request, opts *options.RequestOptions) (*http.Response, error) {
	var resp *http.Response
	var err error
	var bodyBytes []byte

	// If request has a body, buffer it for retries
	if req.Body != nil && req.Body != http.NoBody {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
	}

	retries := 0
	if opts.RetryConfig != nil {
		retries = opts.RetryConfig.MaxRetries
	}

	for attempt := 0; attempt <= retries; attempt++ {
		// Clone the request for retry attempts (rewind body if needed)
		attemptReq := req
		if attempt > 0 && bodyBytes != nil {
			attemptReq, err = cloneRequest(req, bodyBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to clone request for retry: %w", err)
			}
		}

		resp, err = client.Do(attemptReq)

		// Success - no error and no retry needed
		if err == nil {
			if opts.RetryConfig == nil || !shouldRetry(resp.StatusCode, opts.RetryConfig.RetryOnHTTP) {
				return resp, nil
			}
			// Status code indicates retry needed, close response body before retry
			resp.Body.Close()
		}

		// Don't sleep after the last attempt
		if attempt < retries {
			if opts.RetryConfig != nil && opts.RetryConfig.RetryDelay > 0 {
				time.Sleep(opts.RetryConfig.RetryDelay)
			} else {
				// Default exponential backoff: 100ms, 200ms, 400ms, etc.
				backoff := time.Duration(100*(1<<attempt)) * time.Millisecond
				if backoff > 5*time.Second {
					backoff = 5 * time.Second
				}
				time.Sleep(backoff)
			}
		}
	}

	// All retries exhausted
	if err != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", retries, err)
	}
	// Last response had retry-worthy status code
	return resp, nil
}

// cloneRequest creates a copy of the HTTP request with a fresh body reader.
// This is necessary for retrying requests that have bodies (POST, PUT, PATCH).
func cloneRequest(req *http.Request, bodyBytes []byte) (*http.Request, error) {
	cloned := req.Clone(req.Context())

	if bodyBytes != nil {
		cloned.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		cloned.ContentLength = int64(len(bodyBytes))
	}

	return cloned, nil
}

// shouldRetry determines if a request should be retried based on the HTTP status code.
func shouldRetry(statusCode int, retryOnHTTP []int) bool {
	if len(retryOnHTTP) == 0 {
		// Default retry on common transient errors
		switch statusCode {
		case http.StatusTooManyRequests, // 429
			http.StatusInternalServerError, // 500
			http.StatusBadGateway,          // 502
			http.StatusServiceUnavailable,  // 503
			http.StatusGatewayTimeout:      // 504
			return true
		}
		return false
	}

	// Use configured retry codes
	for _, code := range retryOnHTTP {
		if statusCode == code {
			return true
		}
	}
	return false
}
