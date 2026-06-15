package tests

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
)

// statusServer responds with a fixed status code and body.
func statusServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(s.Close)
	return s
}

// TestStatusPolicy_DefaultNoError pins the default contract: a 4xx/5xx response
// is NOT an error; the live response is returned for the caller to inspect.
func TestStatusPolicy_DefaultNoError(t *testing.T) {
	for _, status := range []int{404, 500} {
		srv := statusServer(t, status, "nope")
		resp, err := gocurl.Curl(context.Background(), "curl "+srv.URL)
		if err != nil {
			t.Fatalf("status %d: unexpected error: %v", status, err)
		}
		if resp == nil || resp.StatusCode != status {
			t.Fatalf("status %d: got resp=%v", status, resp)
		}
		resp.Body.Close()
	}
}

// TestStatusPolicy_FailFlag verifies -f/--fail turns a >=400 response into a
// typed ServerStatusError while still returning the response/body.
func TestStatusPolicy_FailFlag(t *testing.T) {
	srv := statusServer(t, 404, "missing")

	body, resp, err := gocurl.CurlString(context.Background(), "curl -f "+srv.URL)
	if err == nil {
		t.Fatal("expected an error with -f on a 404")
	}
	if !errors.Is(err, gocurl.ErrServerStatus) {
		t.Errorf("error should match ErrServerStatus: %v", err)
	}
	if gocurl.KindOf(err) != gocurl.KindServerStatus {
		t.Errorf("KindOf = %v, want KindServerStatus", gocurl.KindOf(err))
	}
	var gerr *gocurl.GocurlError
	if !errors.As(err, &gerr) || gerr.Status != 404 {
		t.Errorf("expected Status=404, got %+v", gerr)
	}
	// The response is returned alongside the error so the body is still readable.
	if resp == nil {
		t.Fatal("expected the response to be returned alongside the -f error")
	}
	resp.Body.Close()
	_ = body
}

// TestStatusPolicy_FailFlagSuccess verifies -f does not error on a 2xx.
func TestStatusPolicy_FailFlagSuccess(t *testing.T) {
	srv := statusServer(t, 200, "ok")
	resp, err := gocurl.Curl(context.Background(), "curl -f "+srv.URL)
	if err != nil {
		t.Fatalf("unexpected error on 200 with -f: %v", err)
	}
	resp.Body.Close()
}

// TestStatusPolicy_WithFailOnStatusClient verifies the Client option path.
func TestStatusPolicy_WithFailOnStatusClient(t *testing.T) {
	srv := statusServer(t, 503, "down")

	c, err := gocurl.New(gocurl.WithFailOnStatus(true))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	resp, err := c.Curl(context.Background(), "curl "+srv.URL)
	if err == nil {
		t.Fatal("expected ServerStatusError from a fail-on-status Client")
	}
	if gocurl.KindOf(err) != gocurl.KindServerStatus {
		t.Errorf("KindOf = %v, want KindServerStatus", gocurl.KindOf(err))
	}
	// 503 is a retryable status.
	if !gocurl.IsRetryable(err) {
		t.Error("503 server-status error should be retryable")
	}
	if resp != nil {
		resp.Body.Close()
	}
}

// TestErrorKind_Connect verifies a refused connection classifies as KindConnect.
func TestErrorKind_Connect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing is listening now

	_, err := gocurl.Curl(context.Background(), "curl "+url)
	if err == nil {
		t.Fatal("expected a connection error")
	}
	if !errors.Is(err, gocurl.ErrConnect) {
		t.Errorf("error should match ErrConnect: %v (kind=%v)", err, gocurl.KindOf(err))
	}
	if !gocurl.IsRetryable(err) {
		t.Error("a connect error should be retryable")
	}
}

// TestErrorKind_Timeout verifies an exceeded deadline classifies as KindTimeout.
func TestErrorKind_Timeout(t *testing.T) {
	srv := statusServer(t, 200, "ok")

	// An already-expired deadline fails deterministically without sleeping.
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
	defer cancel()

	_, err := gocurl.Curl(ctx, "curl "+srv.URL)
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if !errors.Is(err, gocurl.ErrTimeout) {
		t.Errorf("error should match ErrTimeout: %v (kind=%v)", err, gocurl.KindOf(err))
	}
	if !gocurl.IsTimeout(err) {
		t.Error("IsTimeout should be true for an exceeded deadline")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("errors.Is should still resolve context.DeadlineExceeded")
	}
}

// TestErrorKind_Canceled verifies a canceled context classifies as KindCanceled.
func TestErrorKind_Canceled(t *testing.T) {
	srv := statusServer(t, 200, "ok")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the request runs

	_, err := gocurl.Curl(ctx, "curl "+srv.URL)
	if err == nil {
		t.Fatal("expected a cancellation error")
	}
	if gocurl.KindOf(err) != gocurl.KindCanceled {
		t.Errorf("KindOf = %v, want KindCanceled (%v)", gocurl.KindOf(err), err)
	}
	if gocurl.IsTimeout(err) {
		t.Error("a canceled context is not a timeout")
	}
	if gocurl.IsRetryable(err) {
		t.Error("a canceled request is not retryable")
	}
	if !errors.Is(err, context.Canceled) {
		t.Error("errors.Is should resolve context.Canceled")
	}
}

// TestError_NoSecretLeak verifies credentials embedded in a URL never appear in
// the error string (scrub backstop), even on a connect failure.
func TestError_NoSecretLeak(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	host := strings.TrimPrefix(srv.URL, "http://")
	srv.Close()

	cmd := "curl http://user:s3cr3t@" + host + "/p?api_key=TOPSECRET"
	_, err := gocurl.Curl(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected a connection error")
	}
	s := err.Error()
	if strings.Contains(s, "s3cr3t") || strings.Contains(s, "TOPSECRET") {
		t.Errorf("secret leaked in error string: %q", s)
	}
}
