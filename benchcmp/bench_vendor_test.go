// Package benchcmp benchmarks gocurl against popular Go HTTP clients (resty, req)
// over one shared in-process httptest server, with IDENTICAL transport tuning on
// every arm so the numbers compare client overhead, not connection-pool config.
//
// These live in a separate module (see go.mod) so resty/req never pollute the
// library's dependency graph. Honesty rules (Spec 14 §B):
//   - Same server, same fair transport, body drained on every arm.
//   - resty and req BUFFER the full response body by default (their model); gocurl
//     streams. That is a real semantic difference, noted in docs/benchmarking.md —
//     it is not hidden by the harness.
//   - Results are reported as-measured, wins AND losses.
package benchcmp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	resty "github.com/go-resty/resty/v2"
	"github.com/imroc/req/v3"
	"github.com/maniartech/gocurl"
)

// benchFairTransport mirrors gocurl's default Client transport tuning, exactly as
// the root module's bench_roundtrip_test.go does (and as locked by gocurl's
// TestBenchFairness_DefaultTransportTuning). Every arm here uses it.
func benchFairTransport() *http.Transport {
	return &http.Transport{
		ForceAttemptHTTP2:      true,
		MaxResponseHeaderBytes: 1 << 20,
		MaxIdleConns:           100,
		MaxIdleConnsPerHost:    10,
		MaxConnsPerHost:        0,
		IdleConnTimeout:        90 * time.Second,
		TLSHandshakeTimeout:    10 * time.Second,
		ExpectContinueTimeout:  1 * time.Second,
		DisableCompression:     true,
	}
}

func benchServer(b *testing.B) *httptest.Server {
	b.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	b.Cleanup(srv.Close)
	return srv
}

func drain(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// BenchmarkCmp_NetHTTP is the parity bar shared with the root suite.
func BenchmarkCmp_NetHTTP(b *testing.B) {
	srv := benchServer(b)
	client := &http.Client{Transport: benchFairTransport()}
	warm, _ := client.Get(srv.URL)
	drain(warm)

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

// BenchmarkCmp_Gocurl_Prepared is gocurl's "parse once, execute many" hot path.
func BenchmarkCmp_Gocurl_Prepared(b *testing.B) {
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
	warm, _ := c.Do(ctx, req)
	drain(warm)

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

// BenchmarkCmp_Resty measures go-resty/resty over the same server + fair transport.
// resty reads the full body into the Response by default.
func BenchmarkCmp_Resty(b *testing.B) {
	srv := benchServer(b)
	client := resty.New().SetTransport(benchFairTransport())

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.R().Get(srv.URL)
		if err != nil {
			b.Fatal(err)
		}
		_ = resp.Body() // resty has already buffered it
	}
}

// BenchmarkCmp_Req measures imroc/req over the same server + fair transport.
// req also buffers the body by default.
func BenchmarkCmp_Req(b *testing.B) {
	srv := benchServer(b)
	// req manages its own transport type; apply the fair tuning to the underlying
	// *http.Client so the connection-pool config matches the other arms.
	client := req.C()
	client.GetClient().Transport = benchFairTransport()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.R().Get(srv.URL)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
