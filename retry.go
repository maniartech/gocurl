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

	retries := getMaxRetries(opts)

	// Only buffer the body when retries are enabled AND the body cannot be
	// re-obtained via GetBody. Streaming/rewindable bodies set GetBody, and a
	// no-retry request streams straight through without buffering.
	var bodyBytes []byte
	if retries > 0 && req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		var err error
		bodyBytes, err = bufferRequestBody(req)
		if err != nil {
			return nil, err
		}
	}

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
			if cerr := checkContextDuringRetry(req, resp, attempt, retries); cerr != nil {
				return nil, cerr
			}
		}

		// Execute the request attempt
		resp, err = executeAttempt(client, req, bodyBytes, attempt)

		// Handle context errors
		if err != nil && isContextError(err) {
			return nil, fmt.Errorf("request failed due to context error (attempt %d/%d): %w",
				attempt, retries, err)
		}

		// Success or a non-retryable outcome: return as-is.
		if !needsRetry(resp, err, opts) {
			return resp, err
		}

		// Retry is warranted but attempts are exhausted: return the last result,
		// PROPAGATING any error. (Previously the error was dropped here, and the
		// final response body was closed before returning.) When retries were
		// actually configured and the final attempt errored, surface a typed
		// KindRetryExhausted error that chains the last attempt's classified
		// error (so errors.Is(err, ErrTimeout) still resolves).
		if attempt == retries {
			if err != nil && retries > 0 {
				return resp, RetryError(opts.URL, attempt+1, classifyToError(err))
			}
			return resp, err
		}

		// Another attempt remains: discard this response body and back off.
		if resp != nil {
			resp.Body.Close()
		}
		if serr := sleepWithContext(req, opts, attempt); serr != nil {
			return nil, serr
		}
	}

	return resp, err
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

// executeAttempt executes a single request attempt, rewinding the body for
// retries either via GetBody (streaming/rewindable) or the buffered bytes.
func executeAttempt(client options.HTTPClient, req *http.Request, bodyBytes []byte, attempt int) (*http.Response, error) {
	attemptReq := req

	if attempt > 0 {
		switch {
		case req.GetBody != nil:
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("failed to rewind body for retry: %w", err)
			}
			attemptReq = req.Clone(req.Context())
			attemptReq.Body = body
		case bodyBytes != nil:
			var err error
			attemptReq, err = cloneRequest(req, bodyBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to clone request for retry: %w", err)
			}
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
