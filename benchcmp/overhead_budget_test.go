package benchcmp

import "testing"

// TestCompetitiveOverheadBudgets turns the deterministic allocation side of the
// competitive benchmark into a scheduled CI gate. Wall-clock time remains advisory:
// even a stub benchmark is affected by scheduler and CPU-frequency noise, whereas
// bytes/op and allocs/op are stable properties of the code path.
func TestCompetitiveOverheadBudgets(t *testing.T) {
	results := map[string]testing.BenchmarkResult{
		"net_http": testing.Benchmark(BenchmarkOverhead_NetHTTP),
		"gocurl":   testing.Benchmark(BenchmarkOverhead_Gocurl_Prepared),
		"resty":    testing.Benchmark(BenchmarkOverhead_Resty),
		"req":      testing.Benchmark(BenchmarkOverhead_Req),
	}
	for name, result := range results {
		t.Logf("%s: %d ns/op, %d B/op, %d allocs/op", name, result.NsPerOp(), result.AllocedBytesPerOp(), result.AllocsPerOp())
	}

	gocurl := results["gocurl"]
	// Go 1.23 historical measurements were 3,132 B/op and 25 allocs/op; Go 1.26
	// measures 2,490 B/op and 24 allocs/op. These ceilings retain cross-version
	// headroom while catching loss of the lazy-RNG and clone-the-small wins.
	if got := gocurl.AllocedBytesPerOp(); got > 3400 {
		t.Errorf("gocurl bytes/op=%d exceeds cross-version budget=3400", got)
	}
	if got := gocurl.AllocsPerOp(); got > 26 {
		t.Errorf("gocurl allocs/op=%d exceeds cross-version budget=26", got)
	}

	// The scoped competitive claim is allocation efficiency versus the two pinned
	// full-featured clients in benchcmp/go.mod. It is not a market-wide latency
	// claim, and raw net/http remains the lower-overhead parity floor.
	for _, competitor := range []string{"resty", "req"} {
		other := results[competitor]
		if gocurl.AllocedBytesPerOp() >= other.AllocedBytesPerOp() {
			t.Errorf("gocurl bytes/op=%d is not below %s=%d", gocurl.AllocedBytesPerOp(), competitor, other.AllocedBytesPerOp())
		}
		if gocurl.AllocsPerOp() >= other.AllocsPerOp() {
			t.Errorf("gocurl allocs/op=%d is not below %s=%d", gocurl.AllocsPerOp(), competitor, other.AllocsPerOp())
		}
	}
}
