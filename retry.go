package gocurl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/maniartech/gocurl/options"
)

// executeWithRetries handles HTTP request execution with retry logic.
// It properly clones requests with bodies to enable retries for POST/PUT requests.
// It respects context cancellation and will stop retrying if context is cancelled or deadline exceeded.
// Accepts HTTPClient interface to support custom client implementations including mocks and test clients.
func executeWithRetries(client options.HTTPClient, req *http.Request, opts *options.RequestOptions) (*http.Response, error) {
	// Check if context is already cancelled before starting
	if err := checkContextCancelled(req); err != nil {
		return nil, err
	}

	// Buffer request body for retries if needed
	bodyBytes, err := bufferRequestBody(req)
	if err != nil {
		return nil, err
	}

	retries := getMaxRetries(opts)

	return retryLoop(client, req, opts, bodyBytes, retries)
}

// checkContextCancelled checks if the request context is already cancelled
func checkContextCancelled(req *http.Request) error {
	if req.Context() == nil {
		return nil
	}

	select {
	case <-req.Context().Done():
		return fmt.Errorf("request context cancelled before execution: %w", req.Context().Err())
	default:
		return nil
	}
}

// bufferRequestBody reads and buffers the request body for retries
func bufferRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.ContentLength = int64(len(bodyBytes))

	return bodyBytes, nil
}

// getMaxRetries extracts the max retry count from options
func getMaxRetries(opts *options.RequestOptions) int {
	if opts.RetryConfig != nil {
		return opts.RetryConfig.MaxRetries
	}
	return 0
}

// retryLoop executes the retry loop
func retryLoop(client options.HTTPClient, req *http.Request, opts *options.RequestOptions, bodyBytes []byte, retries int) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt <= retries; attempt++ {
		// Check context before each retry attempt
		if attempt > 0 {
			if err := checkContextDuringRetry(req, resp, attempt, retries); err != nil {
				return nil, err
			}
		}

		// Execute the request attempt
		resp, err = executeAttempt(client, req, bodyBytes, attempt)

		// Handle context errors
		if err != nil && isContextError(err) {
			return nil, fmt.Errorf("request failed due to context error (attempt %d/%d): %w",
				attempt, retries, err)
		}

		// Check if retry is needed
		if !needsRetry(resp, err, opts) {
			if err != nil && attempt == retries {
				return nil, fmt.Errorf("request failed after %d retries: %w", retries, err)
			}
			return resp, err
		}

		// Close response body before retry
		if resp != nil {
			resp.Body.Close()
		}

		// Sleep before next attempt (unless this is the last attempt)
		if attempt < retries {
			if err := sleepWithContext(req, opts, attempt); err != nil {
				return nil, err
			}
		}
	}

	return resp, nil
}

// checkContextDuringRetry checks if context was cancelled during retry loop
func checkContextDuringRetry(req *http.Request, resp *http.Response, attempt, retries int) error {
	if req.Context() == nil {
		return nil
	}

	select {
	case <-req.Context().Done():
		if resp != nil {
			resp.Body.Close()
		}
		return fmt.Errorf("request context cancelled during retries (attempt %d/%d): %w",
			attempt, retries, req.Context().Err())
	default:
		return nil
	}
}

// executeAttempt executes a single request attempt
func executeAttempt(client options.HTTPClient, req *http.Request, bodyBytes []byte, attempt int) (*http.Response, error) {
	attemptReq := req

	// Clone request for retry attempts
	if attempt > 0 && bodyBytes != nil {
		var err error
		attemptReq, err = cloneRequest(req, bodyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to clone request for retry: %w", err)
		}
	}

	return client.Do(attemptReq)
}

// isContextError checks if error is due to context cancellation
func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// needsRetry determines if another retry attempt is needed
func needsRetry(resp *http.Response, err error, opts *options.RequestOptions) bool {
	if err != nil {
		return true
	}

	if resp == nil {
		return true
	}

	if opts.RetryConfig == nil {
		return false
	}

	return shouldRetry(resp.StatusCode, opts.RetryConfig.RetryOnHTTP)
}

// sleepWithContext sleeps before next retry attempt with context awareness
func sleepWithContext(req *http.Request, opts *options.RequestOptions, attempt int) error {
	sleepDuration := calculateSleepDuration(opts, attempt)

	if req.Context() != nil {
		select {
		case <-req.Context().Done():
			return fmt.Errorf("request context cancelled during retry delay: %w", req.Context().Err())
		case <-time.After(sleepDuration):
			return nil
		}
	}

	time.Sleep(sleepDuration)
	return nil
}

// calculateSleepDuration calculates the sleep duration for retry
func calculateSleepDuration(opts *options.RequestOptions, attempt int) time.Duration {
	if opts.RetryConfig != nil && opts.RetryConfig.RetryDelay > 0 {
		return opts.RetryConfig.RetryDelay
	}

	// Default exponential backoff: 100ms, 200ms, 400ms, etc.
	backoff := time.Duration(100*(1<<attempt)) * time.Millisecond
	if backoff > 5*time.Second {
		backoff = 5 * time.Second
	}
	return backoff
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
