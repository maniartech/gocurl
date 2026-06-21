package gocurl_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
)

// benchFairTransport returns an *http.Transport tuned IDENTICALLY to gocurl's
// default Client transport, so the net/http (and benchvendor resty/req) arms
// compare client overhead, not connection-pool configuration. These values are
// locked against gocurl's real defaults by TestBenchFairness_DefaultTransportTuning
// (bench_fairness_test.go) — if they drift, that guard fails. DO NOT change one
// without the other.
func benchFairTransport() *http.Transport {
	return &http.Transport{
		ForceAttemptHTTP2:      true,
		MaxResponseHeaderBytes: 1 << 20,
		MaxIdleConns:           100,
		MaxIdleConnsPerHost:    10, // matches gocurl (was 100 — the rigged value)
		MaxConnsPerHost:        0,
		IdleConnTimeout:        90 * time.Second,
		TLSHandshakeTimeout:    10 * time.Second,
		ExpectContinueTimeout:  1 * time.Second,
		DisableCompression:     true, // gocurl decompresses manually (curl semantics)
	}
}

// benchServer returns a shared, in-process httptest server with a fixed small
// JSON body. One server per benchmark function (created before b.ResetTimer) so
// server cost is constant across arms and never lands in the timed loop.
func benchServer(b *testing.B) *httptest.Server {
	b.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	b.Cleanup(srv.Close)
	return srv
}

// drain fully reads and closes the body so the keep-alive connection is returned
// to the pool. Skipping this breaks connection reuse and silently inflates the
// baseline gap.
func drain(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// --- Round-trip: gocurl vs net/http over one shared server -------------------
//
// Arm 1 (NetHTTP) is the parity bar. Arm 2 (Gocurl_Prepared) is the
// "parse once, execute many" hot path — its gap above Arm 1 is gocurl's honest
// execution overhead. Arm 3 (Gocurl_PerCallParse) parses every iteration; its gap
// above Arm 2 is the per-call parse tax that the prepared-Request API avoids.
// We claim PARITY + ergonomics, never "faster than net/http".

// BenchmarkRoundTrip_NetHTTP is the baseline: a well-tuned, reused net/http client.
func BenchmarkRoundTrip_NetHTTP(b *testing.B) {
	srv := benchServer(b)
	client := &http.Client{Transport: benchFairTransport()}

	resp, err := client.Get(srv.URL) // warm up the connection pool
	if err != nil {
		b.Fatal(err)
	}
	drain(resp)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(srv.URL)
		if err != nil {
			b.Fatal(err)
		}
		drain(resp)
	}
}

// BenchmarkRoundTrip_Gocurl_Prepared parses ONCE outside the loop and executes the
// prepared request MANY times over the pooled Client — the measured hot path.
func BenchmarkRoundTrip_Gocurl_Prepared(b *testing.B) {
	srv := benchServer(b)
	c, err := gocurl.New()
	if err != nil {
		b.Fatal(err)
	}
	defer c.Close()
	req, err := c.Prepare("curl " + srv.URL)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()

	resp, err := c.Do(ctx, req) // warm up the cached transport
	if err != nil {
		b.Fatal(err)
	}
	drain(resp)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := c.Do(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
		drain(resp)
	}
}

// BenchmarkRoundTrip_Gocurl_PerCallParse re-parses the command every iteration —
// the cost the prepared-Request API exists to avoid. Expected to be slower; that
// is the point.
func BenchmarkRoundTrip_Gocurl_PerCallParse(b *testing.B) {
	srv := benchServer(b)
	ctx := context.Background()
	cmd := "curl " + srv.URL

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := gocurl.Curl(ctx, cmd)
		if err != nil {
			b.Fatal(err)
		}
		drain(resp)
	}
}

// --- Concurrent throughput (connection reuse via the pooled transport) -------

func BenchmarkRoundTrip_Concurrent_NetHTTP(b *testing.B) {
	srv := benchServer(b)
	client := &http.Client{Transport: benchFairTransport()}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(srv.URL)
			if err != nil {
				b.Error(err)
				return
			}
			drain(resp)
		}
	})
}

func BenchmarkRoundTrip_Concurrent_Gocurl_Prepared(b *testing.B) {
	srv := benchServer(b)
	c, err := gocurl.New()
	if err != nil {
		b.Fatal(err)
	}
	defer c.Close()
	req, err := c.Prepare("curl " + srv.URL)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := c.Do(ctx, req)
			if err != nil {
				b.Error(err)
				return
			}
			drain(resp)
		}
	})
}
