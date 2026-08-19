package gocurl

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/maniartech/gocurl/options"
)

func TestExecutionPathFeatureMatrix(t *testing.T) {
	var mu sync.Mutex
	attempts := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		attempts[req.URL.Path]++
		n := attempts[req.URL.Path]
		mu.Unlock()
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	type result struct {
		path         string
		hookRequests int
		hookRetries  int
	}
	results := make([]result, 0, 3)

	native := result{path: "/native"}
	nativeClient, err := New(
		WithRetry(RetryPolicy{MaxAttempts: 2, Backoff: ConstantBackoff(0)}),
		WithHooks(Hooks{
			OnRequest: func(context.Context, *http.Request) { native.hookRequests++ },
			OnRetry:   func(context.Context, *http.Request, int, error, *http.Response) { native.hookRetries++ },
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer nativeClient.Close()
	nativeReq, err := NewRequest(http.MethodGet, srv.URL+native.path)
	if err != nil {
		t.Fatal(err)
	}
	nativeResp, err := nativeClient.Do(context.Background(), nativeReq)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, nativeResp.Body)
	_ = nativeResp.Body.Close()
	results = append(results, native)

	injected := result{path: "/injected"}
	base := HandlerFromRoundTripper(http.DefaultTransport)
	h := Observe(Hooks{
		OnRequest: func(context.Context, *http.Request) { injected.hookRequests++ },
		OnRetry:   func(context.Context, *http.Request, int, error, *http.Response) { injected.hookRetries++ },
	}, nil, nil, nil)(Retry(RetryPolicy{MaxAttempts: 2, Backoff: ConstantBackoff(0)})(base))
	injectedClient := &http.Client{Transport: RoundTripperFromHandler(h)}
	injectedResp, err := injectedClient.Get(srv.URL + injected.path)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, injectedResp.Body)
	_ = injectedResp.Body.Close()
	results = append(results, injected)

	legacy := result{path: "/legacy"}
	legacyOpts := options.NewRequestOptions(srv.URL + legacy.path)
	legacyOpts.RetryConfig = &options.RetryConfig{
		MaxRetries: 1,
		RetryDelay: time.Nanosecond,
	}
	legacyResp, err := Execute(context.Background(), legacyOpts)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, legacyResp.Body)
	_ = legacyResp.Body.Close()
	results = append(results, legacy)

	for _, got := range results {
		mu.Lock()
		gotAttempts := attempts[got.path]
		mu.Unlock()
		if gotAttempts != 2 {
			t.Errorf("%s attempts=%d, want 2", got.path, gotAttempts)
		}
		wantHooks := 1
		if got.path == "/legacy" {
			wantHooks = 0
		}
		if got.hookRequests != wantHooks || got.hookRetries != wantHooks {
			t.Errorf("%s request/retry hooks=%d/%d, want %d/%d", got.path, got.hookRequests, got.hookRetries, wantHooks, wantHooks)
		}
	}
}
