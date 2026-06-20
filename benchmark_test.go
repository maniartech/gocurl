package gocurl_test

import (
	"testing"

	"github.com/maniartech/gocurl"
)

// These benchmarks measure the ONE-TIME, authoring-time parse/expand cost — the
// price paid once by Prepare, NOT a per-request hot path. The per-request cost is
// measured by the BenchmarkRoundTrip_* arms in bench_roundtrip_test.go. Parsing is
// deliberately not tuned toward 0 allocs/op (see Spec 10): it runs once.

// BenchmarkRequestConstruction measures the one-time cost of parsing a curl
// command into a *RequestOptions.
func BenchmarkRequestConstruction(b *testing.B) {
	args := []string{
		"curl", "-X", "POST",
		"-H", "Content-Type: application/json",
		"-d", `{"key":"value"}`,
		"https://example.com",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := gocurl.ArgsToOptions(args)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkVariableExpansion measures the one-time cost of ${VAR} substitution.
func BenchmarkVariableExpansion(b *testing.B) {
	vars := gocurl.Variables{
		"token": "my-secret-token",
		"url":   "https://example.com",
		"data":  "important data",
	}
	text := "Authorization: Bearer ${token}, URL: ${url}, Data: ${data}"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := gocurl.ExpandVariables(text, vars)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConcurrentRequests measures concurrent ONE-TIME parsing (no I/O); the
// concurrent round-trip hot path is BenchmarkRoundTrip_Concurrent_* .
func BenchmarkConcurrentRequests(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := gocurl.ArgsToOptions([]string{
				"curl", "-X", "GET",
				"https://example.com",
			})
			if err != nil {
				b.Error(err)
				return
			}
		}
	})
}
