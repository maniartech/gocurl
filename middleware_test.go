package gocurl

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/gocurl/middlewares"
)

func okHandler() Handler {
	return func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
	}
}

func TestRetryMiddleware_RetriesTransientResponse(t *testing.T) {
	attempts := 0
	base := Handler(func(*http.Request) (*http.Response, error) {
		attempts++
		status := http.StatusServiceUnavailable
		if attempts == 2 {
			status = http.StatusOK
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: http.NoBody}, nil
	})
	h := Retry(RetryPolicy{MaxAttempts: 2, Backoff: ConstantBackoff(0)})(base)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	resp, err := h(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || attempts != 2 {
		t.Fatalf("status=%d attempts=%d, want 200 and 2", resp.StatusCode, attempts)
	}
}

func TestRetryMiddleware_DoesNotRetryNonIdempotentPost(t *testing.T) {
	attempts := 0
	base := Handler(func(req *http.Request) (*http.Response, error) {
		attempts++
		_, _ = io.Copy(io.Discard, req.Body)
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: http.NoBody}, nil
	})
	h := Retry(RetryPolicy{MaxAttempts: 3, Backoff: ConstantBackoff(0)})(base)
	req, _ := http.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("payload"))

	if _, err := h(req); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("attempts=%d, want 1", attempts)
	}
}

func TestObserveMiddleware_InstrumentsStandaloneRetryChain(t *testing.T) {
	metrics := &fakeMetrics{}
	span := &fakeSpan{}
	tracer := &fakeTracer{span: span}
	logger := &fakeLogger{}
	requests, retries, responses := 0, 0, 0
	hooks := Hooks{
		OnRequest:  func(context.Context, *http.Request) { requests++ },
		OnRetry:    func(context.Context, *http.Request, int, error, *http.Response) { retries++ },
		OnResponse: func(context.Context, *http.Request, *http.Response, time.Duration) { responses++ },
	}

	attempts := 0
	base := Handler(func(*http.Request) (*http.Response, error) {
		attempts++
		status := http.StatusServiceUnavailable
		if attempts == 2 {
			status = http.StatusOK
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: http.NoBody}, nil
	})
	h := Observe(hooks, metrics, tracer, logger)(
		Retry(RetryPolicy{MaxAttempts: 2, Backoff: ConstantBackoff(0)})(base),
	)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	if _, err := h(req); err != nil {
		t.Fatal(err)
	}
	if requests != 1 || retries != 1 || responses != 1 {
		t.Fatalf("hooks request/retry/response = %d/%d/%d, want 1/1/1", requests, retries, responses)
	}
	if metrics.requests != 1 || metrics.retries != 1 || metrics.latencies != 1 || metrics.inFlight != 0 {
		t.Fatalf("metrics request/retry/latency/inflight = %d/%d/%d/%d", metrics.requests, metrics.retries, metrics.latencies, metrics.inFlight)
	}
	if span.ended != 1 || len(span.events) != 1 {
		t.Fatalf("span ended/events = %d/%d, want 1/1", span.ended, len(span.events))
	}
	if len(logger.entries) != 1 {
		t.Fatalf("log entries=%d, want 1", len(logger.entries))
	}
}

func TestChain_OutermostFirstOrder(t *testing.T) {
	var order []string
	mk := func(name string) Middleware {
		return func(next Handler) Handler {
			return func(req *http.Request) (*http.Response, error) {
				order = append(order, "in:"+name)
				resp, err := next(req)
				order = append(order, "out:"+name)
				return resp, err
			}
		}
	}
	base := Handler(func(req *http.Request) (*http.Response, error) {
		order = append(order, "base")
		return okHandler()(req)
	})

	h := chain(base, mk("a"), mk("b"), nil) // nil middleware must be skipped
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	if _, err := h(req); err != nil {
		t.Fatal(err)
	}
	want := []string{"in:a", "in:b", "base", "out:b", "out:a"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestChain_NoMiddleware(t *testing.T) {
	called := false
	base := Handler(func(req *http.Request) (*http.Response, error) {
		called = true
		return okHandler()(req)
	})
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	if _, err := chain(base)(req); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("base not called")
	}
}

func TestFromMiddlewareFunc_MutatesRequest(t *testing.T) {
	mf := middlewares.MiddlewareFunc(func(r *http.Request) (*http.Request, error) {
		r.Header.Set("X-Added", "yes")
		return r, nil
	})
	var seen string
	base := Handler(func(r *http.Request) (*http.Response, error) {
		seen = r.Header.Get("X-Added")
		return okHandler()(r)
	})
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	if _, err := chain(base, FromMiddlewareFunc(mf))(req); err != nil {
		t.Fatal(err)
	}
	if seen != "yes" {
		t.Errorf("middleware mutation not seen: %q", seen)
	}
}

func TestFromMiddlewareFunc_ErrorShortCircuits(t *testing.T) {
	mf := middlewares.MiddlewareFunc(func(r *http.Request) (*http.Request, error) {
		return nil, fmt.Errorf("boom")
	})
	baseCalled := false
	base := Handler(func(r *http.Request) (*http.Response, error) {
		baseCalled = true
		return okHandler()(r)
	})
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	_, err := chain(base, FromMiddlewareFunc(mf))(req)
	if err == nil || err.Error() != "boom" {
		t.Errorf("err = %v, want boom", err)
	}
	if baseCalled {
		t.Error("base must not be called after middleware error")
	}
}

func TestRoundTripperHandlerAdapters(t *testing.T) {
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 204, Body: http.NoBody}, nil
	})
	h := HandlerFromRoundTripper(rt)
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := h(req)
	if err != nil || resp.StatusCode != 204 {
		t.Fatalf("HandlerFromRoundTripper: resp=%v err=%v", resp, err)
	}

	back := RoundTripperFromHandler(h)
	resp2, err := back.RoundTrip(req)
	if err != nil || resp2.StatusCode != 204 {
		t.Fatalf("RoundTripperFromHandler: resp=%v err=%v", resp2, err)
	}
}
