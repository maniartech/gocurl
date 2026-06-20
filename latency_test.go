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
// fixed number of sequential Do calls against an httptest server. testing.B
// reports only mean ns/op, so this fills the percentile gap. It is informational
// (pass/fail only on request error) and skipped in -short.
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

	c, err := gocurl.New()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	req, err := c.Prepare("curl " + srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	do := func() time.Duration {
		start := time.Now()
		resp, err := c.Do(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return time.Since(start)
	}
	do() // warm up the cached transport

	const N = 2000
	samples := make([]time.Duration, 0, N)
	for i := 0; i < N; i++ {
		samples = append(samples, do())
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })

	pct := func(q float64) time.Duration {
		idx := int(float64(len(samples)) * q)
		if idx >= len(samples) {
			idx = len(samples) - 1
		}
		return samples[idx]
	}
	t.Logf("closed-loop Do latency over %d requests: p50=%v p90=%v p99=%v p999=%v (min=%v max=%v)",
		N, pct(0.50), pct(0.90), pct(0.99), pct(0.999), samples[0], samples[len(samples)-1])
}
