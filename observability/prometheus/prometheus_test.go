package prometheus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/gocurl"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetrics_RegistersAndRecords(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := New(reg)

	info := gocurl.RequestInfo{Method: "GET", Host: "api.example.com"}
	m.IncRequest(info)
	m.IncRequest(info)
	m.IncInFlight(1)
	m.IncInFlight(1)
	m.IncInFlight(-1)
	m.IncRetry(info)
	m.ObserveLatency(150*time.Millisecond, gocurl.ResultInfo{RequestInfo: info, StatusCode: 200})
	m.IncError(gocurl.KindTimeout, info)

	if got := testutil.ToFloat64(m.requests.WithLabelValues("GET", "api.example.com")); got != 2 {
		t.Errorf("requests_total = %v, want 2", got)
	}
	if got := testutil.ToFloat64(m.inFlight); got != 1 {
		t.Errorf("in_flight = %v, want 1", got)
	}
	if got := testutil.ToFloat64(m.retries.WithLabelValues("GET", "api.example.com")); got != 1 {
		t.Errorf("retries_total = %v, want 1", got)
	}
	if got := testutil.ToFloat64(m.errors.WithLabelValues("GET", "api.example.com", "timeout")); got != 1 {
		t.Errorf("errors_total{kind=timeout} = %v, want 1", got)
	}
	if got := testutil.CollectAndCount(m.duration); got != 1 {
		t.Errorf("duration series count = %d, want 1", got)
	}

	// All five collectors must be registered (gather succeeds, families present).
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"gocurl_requests_total":           false,
		"gocurl_in_flight":                false,
		"gocurl_request_duration_seconds": false,
		"gocurl_retries_total":            false,
		"gocurl_errors_total":             false,
	}
	for _, f := range families {
		if _, ok := want[f.GetName()]; ok {
			want[f.GetName()] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("metric family %q not registered", name)
		}
	}
}

func TestMetrics_NilRegistrySkipsRegistration(t *testing.T) {
	// Must not panic with a nil registerer; the adapter is still usable.
	m := New(nil)
	m.IncRequest(gocurl.RequestInfo{Method: "POST", Host: "h"})
	if got := testutil.ToFloat64(m.requests.WithLabelValues("POST", "h")); got != 1 {
		t.Errorf("requests_total = %v, want 1", got)
	}
}

// TestMetrics_SatisfiesInterface is a compile-time + runtime check that the
// adapter is accepted by gocurl.WithMetrics.
func TestMetrics_SatisfiesInterface(t *testing.T) {
	var _ gocurl.Metrics = New(prometheus.NewRegistry())
}

// TestMetrics_EndToEndThroughClient drives a real gocurl Client (retrying a 503
// to a 200) and asserts the adapter recorded the request, the retry, and one
// latency observation — proving the core instrumentation wires into the adapter.
func TestMetrics_EndToEndThroughClient(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	reg := prometheus.NewRegistry()
	m := New(reg)
	c, err := gocurl.New(
		gocurl.WithMetrics(m),
		gocurl.WithRetry(gocurl.RetryPolicy{MaxAttempts: 3, Backoff: gocurl.ConstantBackoff(0), RetryOnStatus: []int{503}}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	resp, err := c.Curl(context.Background(), "curl "+srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	host := hostOfURL(t, srv.URL)
	if got := testutil.ToFloat64(m.requests.WithLabelValues("GET", host)); got != 1 {
		t.Errorf("requests_total = %v, want 1", got)
	}
	if got := testutil.ToFloat64(m.retries.WithLabelValues("GET", host)); got != 1 {
		t.Errorf("retries_total = %v, want 1 (one retry)", got)
	}
	if got := testutil.CollectAndCount(m.duration); got != 1 {
		t.Errorf("duration series = %d, want 1", got)
	}
}

func hostOfURL(t *testing.T, raw string) string {
	t.Helper()
	req, err := http.NewRequest("GET", raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req.URL.Hostname()
}
