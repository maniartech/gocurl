package gocurl_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
)

// TestLatencyDistribution reports closed-loop p50/p90/p99/p999 latency over a
// fixed number of sequential requests against one httptest server, for BOTH the
// net/http baseline and gocurl prepared — so the tail-latency comparison is
// apples-to-apples (both arms use the identical fair transport tuning via
// benchFairTransport). testing.B reports only mean ns/op, so this fills the
// percentile gap. It is informational (pass/fail only on request error; wall-clock
// on a shared runner is noisy) and skipped in -short.
//
//	go test -run TestLatencyDistribution -v .
func TestLatencyDistribution(t *testing.T) {
	if testing.Short() {
		t.Skip("latency harness skipped in -short")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	ctx := context.Background()

	// Arm 1: net/http baseline (the parity bar), fair transport.
	nh := &http.Client{Transport: benchFairTransport()}
	nhRun := func() {
		resp, err := nh.Get(srv.URL)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Arm 2: gocurl prepared, executed over the pooled Client.
	c, err := gocurl.New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	req, err := c.Prepare("curl " + srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	gcRun := func() {
		resp, err := c.Do(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	const N = 5000
	for _, arm := range []struct {
		name string
		run  func()
	}{{"net/http", nhRun}, {"gocurl_prepared", gcRun}} {
		for i := 0; i < 200; i++ { // warm up the pool
			arm.run()
		}
		samples := make([]time.Duration, N)
		for i := 0; i < N; i++ {
			start := time.Now()
			arm.run()
			samples[i] = time.Since(start)
		}
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		pct := func(q float64) time.Duration {
			idx := int(float64(len(samples)) * q)
			if idx >= len(samples) {
				idx = len(samples) - 1
			}
			return samples[idx]
		}
		t.Logf("%-16s closed-loop latency over %d reqs: p50=%v p90=%v p99=%v p999=%v (min=%v max=%v)",
			arm.name, N, pct(0.50), pct(0.90), pct(0.99), pct(0.999), samples[0], samples[len(samples)-1])
	}
}
