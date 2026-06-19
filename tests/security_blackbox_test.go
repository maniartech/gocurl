package tests

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maniartech/gocurl"
)

func TestSecurity_SSRFBlocksLoopback(t *testing.T) {
	c, _ := gocurl.New(gocurl.WithSSRFGuard(gocurl.DefaultSSRFPolicy()))
	defer c.Close()
	_, err := c.Curl(context.Background(), "curl http://127.0.0.1:9/x")
	if !errors.Is(err, gocurl.ErrSSRFBlocked) {
		t.Fatalf("expected ErrSSRFBlocked for loopback, got %v", err)
	}
}

func TestSecurity_SSRFAllowListPasses(t *testing.T) {
	// Allow-list the test server's loopback host so the request proceeds.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	policy := gocurl.DefaultSSRFPolicy()
	policy.AllowHosts = []string{"127.0.0.1"}
	c, _ := gocurl.New(gocurl.WithSSRFGuard(policy))
	defer c.Close()
	resp, err := c.Curl(context.Background(), "curl "+srv.URL)
	if err != nil {
		t.Fatalf("allow-listed host should pass: %v", err)
	}
	resp.Body.Close()
}

func TestSecurity_SSRFRedirectToMetadataBlocked(t *testing.T) {
	// A public (loopback, allow-listed) URL that 302-redirects to the cloud
	// metadata endpoint must be blocked by the per-redirect pre-flight.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
	}))
	defer srv.Close()

	policy := gocurl.SSRFPolicy{BlockCloudMetadata: true, BlockLinkLocal: true, AllowHosts: []string{"127.0.0.1"}}
	c, _ := gocurl.New(gocurl.WithSSRFGuard(policy), gocurl.WithFollowRedirects(true))
	defer c.Close()

	_, err := c.Curl(context.Background(), "curl "+srv.URL)
	if !errors.Is(err, gocurl.ErrSSRFBlocked) {
		t.Fatalf("redirect to metadata should be blocked, got %v", err)
	}
}

func TestSecurity_PlaintextAuthFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	cmd := "curl -u alice:secret " + srv.URL

	t.Run("default errors", func(t *testing.T) {
		c, _ := gocurl.New()
		defer c.Close()
		_, _, err := c.CurlString(context.Background(), cmd)
		if err == nil || !strings.Contains(err.Error(), "insecure") {
			t.Fatalf("basic auth over http should fail closed, got %v", err)
		}
	})

	t.Run("WithAllowInsecureAuth overrides", func(t *testing.T) {
		c, _ := gocurl.New(gocurl.WithAllowInsecureAuth(true))
		defer c.Close()
		_, resp, err := c.CurlString(context.Background(), cmd)
		if err != nil {
			t.Fatalf("WithAllowInsecureAuth should permit it: %v", err)
		}
		resp.Body.Close()
	})

	t.Run("env var overrides", func(t *testing.T) {
		t.Setenv("GOCURL_ALLOW_INSECURE_AUTH", "1")
		c, _ := gocurl.New()
		defer c.Close()
		_, resp, err := c.CurlString(context.Background(), cmd)
		if err != nil {
			t.Fatalf("GOCURL_ALLOW_INSECURE_AUTH=1 should permit it: %v", err)
		}
		resp.Body.Close()
	})
}

func TestSecurity_ForbiddenHeaderRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c, _ := gocurl.New()
	defer c.Close()
	_, _, err := c.CurlString(context.Background(), `curl -H "Transfer-Encoding: chunked" `+srv.URL)
	if err == nil || !strings.Contains(err.Error(), "forbidden header") {
		t.Fatalf("setting a forbidden header should be rejected, got %v", err)
	}
}
