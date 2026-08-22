// Command production-composition demonstrates the managed gocurl pipeline against a
// hermetic server. It intentionally combines a redirect, a retryable status, hooks, and
// SSRF enforcement because composition defects usually appear at feature boundaries.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/maniartech/gocurl"
)

func main() {
	if err := run(context.Background()); err != nil {
		panic(err)
	}
}

func run(ctx context.Context) error {
	var serverAttempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/start":
			http.Redirect(w, req, "/unstable", http.StatusFound)
		case "/unstable":
			if serverAttempts.Add(1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = io.WriteString(w, "ready")
		default:
			http.NotFound(w, req)
		}
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		return fmt.Errorf("parse test server URL: %w", err)
	}

	policy := gocurl.DefaultSSRFPolicy()
	// DefaultSSRFPolicy correctly blocks loopback. This exact host is allow-listed only
	// because the example owns the hermetic server; production allow-lists should be
	// reviewed like firewall rules and should not broadly permit private networks.
	policy.AllowHosts = []string{serverURL.Hostname()}

	var retries atomic.Int32
	client, err := gocurl.New(
		gocurl.WithSSRFGuard(policy),
		gocurl.WithRetry(gocurl.RetryPolicy{
			MaxAttempts: 3,
			Backoff:     gocurl.ConstantBackoff(0),
			MaxElapsed:  2 * time.Second,
		}),
		gocurl.WithHooks(gocurl.Hooks{
			OnRetry: func(context.Context, *http.Request, int, error, *http.Response) {
				retries.Add(1)
			},
		}),
	)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	defer client.Close()

	request, err := client.Prepare("curl --location " + server.URL + "/start")
	if err != nil {
		return fmt.Errorf("prepare request: %w", err)
	}
	response, err := client.Do(ctx, request)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	fmt.Printf("status=%d body=%q retries=%d\n", response.StatusCode, body, retries.Load())

	// CheckSSRF is useful for validation, but only WithSSRFGuard also carries the
	// approved addresses to the dial and rechecks redirects. Never replace the managed
	// guard with this validation-only call when DNS rebinding is in scope.
	blocked := gocurl.DefaultSSRFPolicy().CheckSSRF(ctx, "127.0.0.1")
	if !errors.Is(blocked, gocurl.ErrSSRFBlocked) {
		return fmt.Errorf("expected loopback to be blocked, got %v", blocked)
	}

	return nil
}
