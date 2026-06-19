package gocurl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/maniartech/gocurl/options"
)

// executeWithRetries runs req under the given RetryPolicy. A nil policy (or one
// with MaxAttempts <= 1) executes the request exactly once. The retry engine is
// shared by the one-shot path (process.go) and the Client path (client.go); the
// caller resolves the policy and supplies a per-request *rand.Rand for jitter
// (math/rand.Rand is not safe for concurrent use, so it must never be shared).
func executeWithRetries(client options.HTTPClient, req *http.Request, opts *options.RequestOptions, policy *RetryPolicy, rnd *rand.Rand) (*http.Response, error) {
	if err := checkContextCancelled(req); err != nil {
		return nil, err
	}

	// No retries: execute once, no body buffering.
	if policy == nil || policy.maxAttempts() <= 1 {
		return client.Do(req)
	}
	if rnd == nil {
		rnd = newRand()
	}

	// Body replay setup: prefer GetBody (net/http convention, set by CreateRequest
	// for rewindable BodySource). Otherwise buffer once, capped — a body that
	// exceeds the cap is sent on attempt 1 but is not replayable afterwards.
	bodyBytes, replayable, err := setupReplay(req, policy.replayCap())
	if err != nil {
		return nil, err
	}

	return retryLoop(client, req, opts, policy, rnd, bodyBytes, replayable)
}

// checkContextCancelled checks if the request context is already cancelled.
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

// setupReplay arranges for the request body to be re-sendable across attempts.
// It returns the buffered bytes (nil when GetBody is used or there is no body)
// and whether the body is replayable on attempt 2+.
//
//   - No body / http.NoBody  => replayable (nothing to replay).
//   - req.GetBody set        => replayable via GetBody, no buffering.
//   - cap <= 0 (unlimited)   => buffer the whole body, replayable.
//   - body <= cap            => buffer, replayable.
//   - body > cap, no GetBody => send once (head+rest), NOT replayable.
func setupReplay(req *http.Request, cap int64) ([]byte, bool, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, true, nil
	}
	if req.GetBody != nil {
		return nil, true, nil
	}

	if cap <= 0 {
		b, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, false, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(b))
		req.ContentLength = int64(len(b))
		return b, true, nil
	}

	head, err := io.ReadAll(io.LimitReader(req.Body, cap+1))
	if err != nil {
		return nil, false, fmt.Errorf("failed to read request body: %w", err)
	}
	if int64(len(head)) <= cap {
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(head))
		req.ContentLength = int64(len(head))
		return head, true, nil
	}

	// Body exceeds the replay cap: send what we have plus the remainder for this
	// one attempt, but mark it non-replayable so attempts 2+ short-circuit.
	rest := req.Body
	req.Body = &readMultiCloser{
		Reader: io.MultiReader(bytes.NewReader(head), rest),
		closer: rest,
	}
	return nil, false, nil
}

// readMultiCloser adapts a combined io.Reader plus the original Closer into an
// io.ReadCloser (used when a body is too large to buffer for replay).
type readMultiCloser struct {
	io.Reader
	closer io.Closer
}

func (r *readMultiCloser) Close() error { return r.closer.Close() }

// retryLoop drives the per-attempt loop for a multi-attempt policy.
func retryLoop(client options.HTTPClient, req *http.Request, opts *options.RequestOptions, policy *RetryPolicy, rnd *rand.Rand, bodyBytes []byte, replayable bool) (*http.Response, error) {
	maxAttempts := policy.maxAttempts()
	eligible := policy.eligibleForRetry(req)
	loopStart := time.Now()

	var resp *http.Response
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		started := time.Now()

		// Fire the per-retry observability hook (if installed) at the top of each
		// retry, carrying the previous attempt's outcome. Decoupled from the
		// observability types: retry.go only knows about an optional context func.
		if attempt > 1 {
			if h := retryHookFromContext(req.Context()); h != nil {
				h(attempt, err, resp)
			}
		}

		attemptReq, berr := buildAttempt(req, attempt, bodyBytes, replayable)
		if berr != nil {
			return nil, berr // body rewind failure — non-retryable
		}
		attemptReq, pa := applyPerAttempt(attemptReq, policy)

		resp, err = client.Do(attemptReq)
		// The per-attempt deadline bounds time-to-RESPONSE, not the time the caller
		// spends reading the body, so stop the timer as soon as Do returns.
		pa.stop()

		// A context error from the PARENT request context (caller cancelled, or the
		// overall --max-time/deadline expired) is terminal and non-retryable. A
		// PerAttempt deadline (the timer fired while the parent is still alive) is a
		// retryable timeout — re-cast it so classification yields KindTimeout.
		if err != nil && isContextError(err) {
			if perr := parentContextErr(req); perr != nil {
				drainAndCloseBody(resp)
				pa.release()
				return nil, fmt.Errorf("request failed due to context error (attempt %d/%d): %w", attempt, maxAttempts, perr)
			}
			if pa.didFire() {
				err = fmt.Errorf("per-attempt deadline exceeded: %w", context.DeadlineExceeded)
			}
		}

		att := &Attempt{Method: req.Method, Number: attempt, Response: resp, Err: err, Started: started}
		willRetry := replayable && attempt < maxAttempts && policy.wantRetry(att, eligible)

		if !willRetry {
			return finalizeResult(resp, err, pa.cancelFunc(), policy, attempt, opts)
		}

		// We intend to retry. Compute the delay and check the wall-clock budget
		// BEFORE discarding the body, so a "can't wait" outcome still returns the
		// live response to the caller.
		delay := nextDelay(policy, attempt, resp, rnd)
		if policy.MaxElapsed > 0 && time.Since(loopStart)+delay > policy.MaxElapsed {
			return finalizeResult(resp, err, pa.cancelFunc(), policy, attempt, opts)
		}
		if !policy.Budget.Consume() { // nil-safe: a nil budget always permits
			return finalizeResult(resp, err, pa.cancelFunc(), policy, attempt, opts)
		}

		// Commit to retrying: discard this attempt's body (drain for keep-alive
		// reuse) and wait.
		drainAndCloseBody(resp)
		pa.release()
		if serr := sleepWithContext(req, delay); serr != nil {
			return nil, serr
		}
	}

	return resp, err
}

// buildAttempt returns the *http.Request to use for the given 1-based attempt,
// rewinding the body for attempts after the first via GetBody or buffered bytes.
func buildAttempt(req *http.Request, attempt int, bodyBytes []byte, replayable bool) (*http.Request, error) {
	if attempt == 1 {
		return req, nil
	}
	if !replayable {
		return nil, errors.New("cannot retry: request body is not replayable")
	}
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("failed to rewind body for retry: %w", err)
		}
		clone := req.Clone(req.Context())
		clone.Body = body
		return clone, nil
	}
	if bodyBytes != nil {
		return cloneRequest(req, bodyBytes)
	}
	return req.Clone(req.Context()), nil
}

// perAttempt holds the per-attempt deadline machinery. It uses a cancellable
// child context plus a timer (rather than context.WithTimeout) so the deadline
// can be DISARMED — via stop — once the response is in hand, ensuring it bounds
// only time-to-response and never truncates the caller's body read.
type perAttempt struct {
	cancel context.CancelFunc
	timer  *time.Timer
	fired  atomic.Bool
}

// applyPerAttempt wraps req in a per-attempt deadline context when PerAttempt > 0,
// returning the per-attempt handle (nil when no per-attempt deadline applies).
func applyPerAttempt(req *http.Request, policy *RetryPolicy) (*http.Request, *perAttempt) {
	if policy.PerAttempt <= 0 {
		return req, nil
	}
	ctx, cancel := context.WithCancel(req.Context())
	pa := &perAttempt{cancel: cancel}
	pa.timer = time.AfterFunc(policy.PerAttempt, func() {
		pa.fired.Store(true)
		cancel()
	})
	return req.WithContext(ctx), pa
}

// stop disarms the deadline timer (so it cannot fire during a body read).
func (pa *perAttempt) stop() {
	if pa != nil && pa.timer != nil {
		pa.timer.Stop()
	}
}

// didFire reports whether the per-attempt deadline elapsed.
func (pa *perAttempt) didFire() bool { return pa != nil && pa.fired.Load() }

// cancelFunc returns the per-attempt cancel (nil when none), to bind to the
// returned response body's close.
func (pa *perAttempt) cancelFunc() context.CancelFunc {
	if pa == nil {
		return nil
	}
	return pa.cancel
}

// release cancels the per-attempt context (used when the attempt is discarded).
func (pa *perAttempt) release() {
	if pa != nil && pa.cancel != nil {
		pa.cancel()
	}
}

// nextDelay computes the delay before the next attempt, honoring Retry-After on
// 429/503 when configured (effective delay = max(backoff, Retry-After)).
func nextDelay(policy *RetryPolicy, attempt int, resp *http.Response, rnd *rand.Rand) time.Duration {
	delay := policy.backoff().Delay(attempt, rnd)
	if policy.RespectRetryAfter && resp != nil && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) {
		if ra := parseRetryAfter(resp); ra > delay {
			delay = ra
		}
	}
	return delay
}

// finalizeResult returns the final (resp, err) for the loop, wrapping a
// retry-exhausted transport error, refunding the retry budget on a completed
// request, and binding any per-attempt cancel to the response body's close.
func finalizeResult(resp *http.Response, err error, cancel context.CancelFunc, policy *RetryPolicy, attempt int, opts *options.RequestOptions) (*http.Response, error) {
	if err != nil && policy.maxAttempts() > 1 && attempt == policy.maxAttempts() {
		err = RetryError(opts.URL, attempt, classifyToError(err))
	}
	if err == nil {
		policy.Budget.Refund() // nil-safe
	}
	if cancel != nil {
		if resp != nil && resp.Body != nil {
			resp.Body = &cancelOnCloseBody{ReadCloser: resp.Body, cancel: cancel}
		} else {
			cancel()
		}
	}
	return resp, err
}

// drainAndCloseBody fully drains then closes a response body so the pooled
// transport can reuse the keep-alive connection (a bare Close on a partially
// read body poisons reuse).
func drainAndCloseBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// isContextError reports whether err is due to context cancellation/deadline.
func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// parentContextErr returns the request's (parent) context error, if any. A
// PerAttempt deadline lives on a child context, so a non-nil result here means
// the caller's context — not just one attempt — is done.
func parentContextErr(req *http.Request) error {
	if req.Context() == nil {
		return nil
	}
	return req.Context().Err()
}

// sleepWithContext waits delay, aborting early if the request context is done.
func sleepWithContext(req *http.Request, delay time.Duration) error {
	if delay <= 0 {
		delay = 0
	}
	if req.Context() != nil {
		timer := time.NewTimer(delay)
		select {
		case <-req.Context().Done():
			timer.Stop()
			return fmt.Errorf("request context cancelled during retry delay: %w", req.Context().Err())
		case <-timer.C:
			return nil
		}
	}
	time.Sleep(delay)
	return nil
}

// cloneRequest creates a copy of req with a fresh body reader from bodyBytes.
func cloneRequest(req *http.Request, bodyBytes []byte) (*http.Request, error) {
	cloned := req.Clone(req.Context())
	if bodyBytes != nil {
		cloned.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		cloned.ContentLength = int64(len(bodyBytes))
	}
	return cloned, nil
}

// shouldRetry reports whether a status code is in the retry set (or the default
// transient set when retryOnHTTP is empty). It is shared by the retry policy and
// the error classifier (errors.go / error_classify.go).
func shouldRetry(statusCode int, retryOnHTTP []int) bool {
	if len(retryOnHTTP) == 0 {
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
	for _, code := range retryOnHTTP {
		if statusCode == code {
			return true
		}
	}
	return false
}
