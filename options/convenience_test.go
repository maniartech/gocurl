package options_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/maniartech/gocurl/options"
)

func TestRequestOptionsConvenienceMethods(t *testing.T) {
	opts := options.NewRequestOptions("https://example.com")
	if opts.URL != "https://example.com" || opts.FollowRedirects || opts.Compress {
		t.Fatalf("unexpected defaults: %+v", opts)
	}

	opts.SetBasicAuth("user", "pass")
	opts.AddHeader("X-Test", "one")
	opts.AddHeader("X-Test", "two")
	opts.SetHeader("X-Test", "final")
	opts.AddQueryParam("q", "one")
	opts.AddQueryParam("q", "two")
	opts.SetQueryParam("q", "final")
	if opts.BasicAuth == nil || opts.BasicAuth.Username != "user" || opts.Headers.Get("X-Test") != "final" || opts.QueryParams.Get("q") != "final" {
		t.Fatalf("convenience mutation failed: %+v", opts)
	}

	encoded, err := opts.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["url"] != "https://example.com" {
		t.Fatalf("JSON url=%v", decoded["url"])
	}
}

func TestRequestOptionsBuilderConveniencePatterns(t *testing.T) {
	form := url.Values{"name": {"widget"}}
	builder := options.NewRequestOptionsBuilder().
		JSON(map[string]string{"name": "widget"}).
		BearerAuth("token").
		WithDefaultRetry().
		WithExponentialBackoff(4, 25*time.Millisecond).
		QuickTimeout().
		SlowTimeout()

	got := builder.Build()
	if got.Body != `{"name":"widget"}` || got.Headers.Get("Authorization") != "Bearer token" {
		t.Fatalf("JSON/auth result: %+v", got)
	}
	if got.Headers.Get("Content-Type") != "application/json" {
		t.Fatalf("JSON content type=%q", got.Headers.Get("Content-Type"))
	}
	if got.RetryConfig == nil || got.RetryConfig.MaxRetries != 4 || got.RetryConfig.RetryDelay != 25*time.Millisecond {
		t.Fatalf("retry result: %+v", got.RetryConfig)
	}
	if got.Timeout != 2*time.Minute {
		t.Fatalf("timeout=%s, want 2m", got.Timeout)
	}

	formResult := options.NewRequestOptionsBuilder().Form(form).Build()
	if formResult.Headers.Get("Content-Type") != "application/x-www-form-urlencoded" || formResult.Form.Get("name") != "widget" {
		t.Fatalf("form result: %+v", formResult)
	}
}

func TestRequestOptionsBuilderContextLifecycle(t *testing.T) {
	builder := options.NewRequestOptionsBuilder()
	if builder.GetContext() == nil {
		t.Fatal("default context is nil")
	}

	type contextKey struct{}
	parent := context.WithValue(context.Background(), contextKey{}, "value")
	builder.WithContext(parent).WithTimeout(time.Hour)
	ctx := builder.GetContext()
	if ctx.Value(contextKey{}) != "value" {
		t.Fatal("parent context value was not preserved")
	}
	builder.Cleanup()
	if ctx.Err() != context.Canceled {
		t.Fatalf("context error=%v, want canceled after Cleanup", ctx.Err())
	}
}

func TestRequestOptionsBuilderJSONMarshalFailureIsSafe(t *testing.T) {
	builder := options.NewRequestOptionsBuilder().JSON(make(chan int))
	got := builder.Build()
	if got.Body != "" || got.Headers.Get("Content-Type") != "" {
		t.Fatalf("marshal failure should leave an empty JSON body/header: %+v", got)
	}
}

func TestValidateRequestExportedEntryPoint(t *testing.T) {
	valid := options.NewRequestOptions("https://example.com")
	valid.Method = http.MethodGet
	if err := options.ValidateRequest(valid); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	invalid := options.NewRequestOptions("https://example.com")
	invalid.Method = "bad method"
	if err := options.ValidateRequest(invalid); err == nil {
		t.Fatal("invalid method accepted")
	}
}
