package gocurl

import (
	"net/http"
	"testing"
)

// Benchmark fairness guard (Spec 14 §B).
//
// A competitive benchmark is only honest if every arm runs with IDENTICAL
// transport tuning — otherwise the numbers compare connection-pool configs, not
// client overhead. The review found the net/http arm was running
// MaxIdleConnsPerHost:100 against gocurl's 10, a rigged comparison.
//
// This guard locks gocurl's DEFAULT Client transport tuning to the exact values
// the external benchmark arms hard-code in benchFairTransport (bench_roundtrip_test.go).
// If gocurl's defaults drift, this test fails — forcing the bench helper (and the
// published numbers) to be updated in lockstep, so the arms can never silently diverge.
//
// The canonical values are duplicated as literals here ON PURPOSE: this test is the
// single enforcement point that the two packages agree.
func TestBenchFairness_DefaultTransportTuning(t *testing.T) {
	rt, err := defaultConfig().buildTransport()
	if err != nil {
		t.Fatalf("buildTransport: %v", err)
	}
	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatalf("default transport is %T, want *http.Transport", rt)
	}

	checks := []struct {
		name string
		got  int64
		want int64
	}{
		{"MaxIdleConns", int64(tr.MaxIdleConns), 100},
		{"MaxIdleConnsPerHost", int64(tr.MaxIdleConnsPerHost), 10},
		{"MaxConnsPerHost", int64(tr.MaxConnsPerHost), 0},
		{"MaxResponseHeaderBytes", tr.MaxResponseHeaderBytes, 1 << 20},
		{"IdleConnTimeout", int64(tr.IdleConnTimeout), int64(90e9)},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("default transport %s = %d, want %d — update benchFairTransport "+
				"in bench_roundtrip_test.go to match (the bench arms must mirror gocurl's tuning)",
				c.name, c.got, c.want)
		}
	}
	if !tr.DisableCompression {
		t.Error("default transport DisableCompression = false, want true (gocurl decompresses manually)")
	}
	if !tr.ForceAttemptHTTP2 {
		t.Error("default transport ForceAttemptHTTP2 = false, want true")
	}
}
