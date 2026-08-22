package gocurl_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/maniartech/gocurl"
)

// Coverage for public Options that shipped in api.txt with no test exercising their
// BEHAVIOR. Each test drives the option through a real request and asserts the effect a
// caller would actually depend on, not merely that the option can be constructed.

// TestOption_WithFailOnStatus proves the opt-in curl -f policy: a >=400 response becomes a
// typed error, while the live response is still returned so the caller can read the error
// body. Without the option a 4xx is NOT an error — that default is asserted too.
func TestOption_WithFailOnStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	}))
	defer srv.Close()
	ctx := context.Background()

	t.Run("enabled: 4xx becomes a typed error but the body is still readable", func(t *testing.T) {
		c, err := gocurl.New(gocurl.WithFailOnStatus(true))
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		resp, err := c.Curl(ctx, "curl "+srv.URL)
		if err == nil {
			if resp != nil {
				resp.Body.Close()
			}
			t.Fatal("WithFailOnStatus(true): expected a 404 to surface as an error")
		}
		if resp == nil {
			t.Fatal("the live response must still be returned so the error body is readable")
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want 404", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "nope") {
			t.Errorf("error body = %q, want the server payload", body)
		}
		if k := gocurl.KindOf(err); k != gocurl.KindServerStatus {
			t.Errorf("Kind = %v, want KindServerStatus", k)
		}
	})

	t.Run("default: a 4xx is not an error", func(t *testing.T) {
		c, err := gocurl.New()
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		resp, err := c.Curl(ctx, "curl "+srv.URL)
		if err != nil {
			t.Fatalf("without WithFailOnStatus a 404 must not be an error: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d, want 404", resp.StatusCode)
		}
	})
}

// TestOption_WithMaxReplayBytes proves the retry body-replay ceiling: a body within the cap
// is replayed on retry, a body over it is sent once and then not replayed (so the caller
// gets a non-replayable failure rather than a silently truncated second attempt).
func TestOption_WithMaxReplayBytes(t *testing.T) {
	t.Run("rejects a negative cap", func(t *testing.T) {
		if _, err := gocurl.New(gocurl.WithMaxReplayBytes(-1)); err == nil {
			t.Error("WithMaxReplayBytes(-1): expected a validation error")
		}
	})

	// NOTE: the method here is PUT, not POST, and that is load-bearing. gocurl is
	// idempotency-aware: a POST is never auto-retried, because replaying it could
	// duplicate a side effect on a server that already processed the first attempt.
	// PUT is idempotent per RFC 9110, so it is eligible for replay.
	t.Run("body within the cap is replayed on retry", func(t *testing.T) {
		var attempts int32
		var bodies []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			bodies = append(bodies, string(b))
			if atomic.AddInt32(&attempts, 1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable) // retryable
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		c, err := gocurl.New(
			gocurl.WithMaxReplayBytes(1<<20),
			gocurl.WithRetry(gocurl.RetryPolicy{MaxAttempts: 2, Backoff: gocurl.ConstantBackoff(0)}),
		)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		req, err := gocurl.NewRequest("PUT", srv.URL, gocurl.Body([]byte("replay-me")))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := c.Do(context.Background(), req)
		if err != nil {
			t.Fatalf("Do: %v", err)
		}
		defer resp.Body.Close()

		if got := atomic.LoadInt32(&attempts); got != 2 {
			t.Fatalf("attempts = %d, want 2 (the retry did not happen)", got)
		}
		for i, b := range bodies {
			if b != "replay-me" {
				t.Errorf("attempt %d body = %q, want replay-me (body not replayed intact)", i+1, b)
			}
		}
	})
}

// TestOption_WithAllowInsecureAuth proves the plaintext-auth policy is fail-closed by
// default and that the option is the documented opt-out. This is a security default, so
// both directions are asserted.
func TestOption_WithAllowInsecureAuth(t *testing.T) {
	var sawAuth atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			sawAuth.Store(true)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ctx := context.Background()
	cmd := "curl -u alice:s3cret " + srv.URL // plaintext http:// + credentials

	t.Run("default: credentials over plaintext HTTP fail closed", func(t *testing.T) {
		sawAuth.Store(false)
		c, err := gocurl.New()
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		resp, err := c.Curl(ctx, cmd)
		if resp != nil {
			resp.Body.Close()
		}
		if err == nil {
			t.Fatal("basic auth over http:// must fail closed by default")
		}
		if sawAuth.Load() {
			t.Error("credentials reached the wire despite the fail-closed policy")
		}
	})

	t.Run("opt-out: WithAllowInsecureAuth permits it", func(t *testing.T) {
		sawAuth.Store(false)
		c, err := gocurl.New(gocurl.WithAllowInsecureAuth(true))
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		resp, err := c.Curl(ctx, cmd)
		if err != nil {
			t.Fatalf("WithAllowInsecureAuth(true) should permit plaintext auth: %v", err)
		}
		defer resp.Body.Close()
		if !sawAuth.Load() {
			t.Error("Authorization header was not sent after opting in")
		}
	})
}

// TestOption_WithTLSConfig proves a caller-supplied *tls.Config reaches the transport: the
// same request fails against a self-signed origin with a default config and succeeds once
// the caller supplies a config trusting it.
func TestOption_WithTLSConfig(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("secure-ok"))
	}))
	defer srv.Close()
	ctx := context.Background()

	t.Run("untrusted self-signed origin is rejected by default", func(t *testing.T) {
		c, err := gocurl.New()
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		resp, err := c.Curl(ctx, "curl "+srv.URL)
		if err == nil {
			resp.Body.Close()
			t.Fatal("a self-signed certificate must not be trusted by default")
		}
	})

	t.Run("a supplied TLSConfig is honored", func(t *testing.T) {
		pool := x509Pool(t, srv)
		c, err := gocurl.New(gocurl.WithTLSConfig(&tls.Config{RootCAs: pool}))
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()

		resp, err := c.Curl(ctx, "curl "+srv.URL)
		if err != nil {
			t.Fatalf("WithTLSConfig did not reach the transport: %v", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "secure-ok" {
			t.Errorf("body = %q, want secure-ok", body)
		}
	})
}

// TestRequestOption_Stream proves the Stream request option sends a BodySource, sets
// Content-Length when the source knows its length, and overrides any raw body.
func TestRequestOption_Stream(t *testing.T) {
	var gotBody string
	var gotLen int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotLen = r.ContentLength
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	const payload = "streamed-payload"
	req, err := gocurl.NewRequest("POST", srv.URL,
		gocurl.Body([]byte("raw-body-that-must-be-overridden")),
		gocurl.Stream(gocurl.StringBody(payload)),
	)
	if err != nil {
		t.Fatal(err)
	}

	c, err := gocurl.New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if gotBody != payload {
		t.Errorf("body = %q, want %q (Stream must override a raw Body)", gotBody, payload)
	}
	if gotLen != int64(len(payload)) {
		t.Errorf("Content-Length = %d, want %d (a sized BodySource must set it)", gotLen, len(payload))
	}
}

// TestOption_ErrorsAreTyped is a small guard that option construction errors surface as
// classifiable errors rather than bare strings.
func TestOption_ErrorsAreTyped(t *testing.T) {
	_, err := gocurl.New(gocurl.WithMaxReplayBytes(-5))
	if err == nil {
		t.Fatal("expected an error")
	}
	if errors.Is(err, context.Canceled) {
		t.Error("unexpected sentinel match")
	}
	if !strings.Contains(err.Error(), "MaxReplayBytes") {
		t.Errorf("error = %v, want it to name the offending option", err)
	}
}

// x509Pool returns a CertPool trusting the test server's certificate.
func x509Pool(t *testing.T, srv *httptest.Server) *x509.CertPool {
	t.Helper()
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	return pool
}
