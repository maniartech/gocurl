package gocurl_test

import (
	"testing"

	"github.com/maniartech/gocurl"
)

// BenchmarkRequestConstruction measures the overhead of parsing curl commands
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

// BenchmarkVariableExpansion measures variable substitution performance
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

// BenchmarkRequestAPI measures the high-level API performance
func BenchmarkRequestAPI(b *testing.B) {
	// Note: This will make real HTTP requests - commented out for now
	// Uncomment when you want to benchmark with a local test server
	
	b.Skip("Skipping HTTP request benchmark - requires test server")
	
	vars := gocurl.Variables{
		"url": "http://localhost:8080/test",
	}
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		resp, err := gocurl.Request("curl ${url}", vars)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

// BenchmarkConcurrentRequests measures concurrent request handling
func BenchmarkConcurrentRequests(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := gocurl.ArgsToOptions([]string{
				"curl", "-X", "GET",
				"https://example.com",
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
