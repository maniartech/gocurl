package gocurl

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maniartech/gocurl/options"
)

// TestExecute_BuilderRoundTrip is the end-to-end guard for the typed/builder API
// approach: a request assembled with options.NewRequestOptionsBuilder() must be
// runnable via the public Execute, returning a live, streamable response. This is
// the path the book2 "builder pattern" chapter documents.
func TestExecute_BuilderRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("X-Demo"); got != "1" {
			t.Errorf("X-Demo = %q, want 1", got)
		}
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		w.Write(append([]byte("echo:"), body...))
	}))
	defer server.Close()

	opts := options.NewRequestOptionsBuilder().
		SetURL(server.URL).
		SetMethod(http.MethodPost).
		AddHeader("X-Demo", "1").
		SetBody("ping").
		Build()

	resp, err := Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("Execute(builder opts) failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "echo:ping" {
		t.Errorf("body = %q, want %q", body, "echo:ping")
	}
}

// TestExecute_NilOptions verifies the public entry point rejects nil with a
// classifiable validation error rather than panicking.
func TestExecute_NilOptions(t *testing.T) {
	_, err := Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("Execute(nil) must return an error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("Execute(nil) error = %v, want ErrValidation", err)
	}
}

// TestExecute_HonorsResponseBodyLimit verifies Execute applies the same streaming
// ResponseBodyLimit the Curl* path does (it shares executeOpts), so the typed API
// is not a second-class path that skips the DoS guard.
func TestExecute_HonorsResponseBodyLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("A", 4096)))
	}))
	defer server.Close()

	opts := options.NewRequestOptionsBuilder().SetURL(server.URL).Build()
	opts.ResponseBodyLimit = 1024

	resp, err := Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	defer resp.Body.Close()

	_, rerr := io.ReadAll(resp.Body)
	if rerr == nil || !errors.Is(rerr, ErrBodyRead) {
		t.Fatalf("body read past the cap must fail with ErrBodyRead, got: %v", rerr)
	}
}
