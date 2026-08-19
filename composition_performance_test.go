package gocurl

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// These sinks intentionally do no work. The budgets below measure gocurl's
// instrumentation machinery, not allocation choices made by a consumer's logger,
// tracer, or metrics backend.
type performanceMetrics struct{}

func (performanceMetrics) IncRequest(RequestInfo)                   {}
func (performanceMetrics) IncInFlight(int)                          {}
func (performanceMetrics) ObserveLatency(time.Duration, ResultInfo) {}
func (performanceMetrics) IncRetry(RequestInfo)                     {}
func (performanceMetrics) IncError(Kind, RequestInfo)               {}

type performanceTracer struct{}

func (performanceTracer) StartSpan(ctx context.Context, _ string, _ ...Field) (context.Context, Span) {
	return ctx, performanceSpan{}
}

type performanceSpan struct{}

func (performanceSpan) SetAttributes(...Field)    {}
func (performanceSpan) AddEvent(string, ...Field) {}
func (performanceSpan) RecordError(error)         {}
func (performanceSpan) End()                      {}

type performanceLogger struct{}

func (performanceLogger) Log(context.Context, Level, string, ...Field) {}

type performanceLimiter struct{}

func (performanceLimiter) Wait(context.Context) error { return nil }

type compositionPerformanceCase struct {
	name      string
	handler   Handler
	maxAllocs int64
	maxBytes  int64
}

func compositionPerformanceCases() []compositionPerformanceCase {
	base := Handler(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       http.NoBody,
			Request:    req,
		}, nil
	})
	observe := Observe(
		Hooks{OnRequest: func(context.Context, *http.Request) {}},
		performanceMetrics{},
		performanceTracer{},
		performanceLogger{},
	)
	retry := Retry(RetryPolicy{MaxAttempts: 1})
	ssrf := SSRFGuard(DefaultSSRFPolicy())
	full := observe(ssrf(CircuitBreaker(BreakerConfig{MinRequests: 1 << 30})(
		RateLimiter(performanceLimiter{})(retry(base)),
	)))

	// Ceilings include request construction because that is part of the actual
	// RoundTripper-facing cost. They are ratcheted from repeatable local/CI
	// measurements with modest cross-Go/OS headroom; timing is deliberately not a
	// gate because scheduler noise is not a library regression.
	return []compositionPerformanceCase{
		{name: "bare", handler: base, maxAllocs: 8, maxBytes: 1024},
		{name: "retry", handler: retry(base), maxAllocs: 10, maxBytes: 1792},
		{name: "observe", handler: observe(base), maxAllocs: 27, maxBytes: 2304},
		{name: "ssrf_literal", handler: ssrf(base), maxAllocs: 16, maxBytes: 1280},
		{name: "full_chain", handler: full, maxAllocs: 38, maxBytes: 3584},
	}
}

func benchmarkCompositionHandler(h Handler) testing.BenchmarkResult {
	return testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://203.0.113.10/resource", nil)
			if err != nil {
				b.Fatal(err)
			}
			resp, err := h(req)
			if err != nil {
				b.Fatal(err)
			}
			_ = resp.Body.Close()
		}
	})
}

func TestCompositionPerformance_Budgets(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("race instrumentation changes allocation sizes; enforced by the non-race performance-gates CI job")
	}
	for _, tc := range compositionPerformanceCases() {
		t.Run(tc.name, func(t *testing.T) {
			result := benchmarkCompositionHandler(tc.handler)
			allocs := result.AllocsPerOp()
			bytes := result.AllocedBytesPerOp()
			t.Logf("%s: %d ns/op, %d B/op, %d allocs/op", tc.name, result.NsPerOp(), bytes, allocs)
			if allocs > tc.maxAllocs {
				t.Errorf("allocs/op=%d exceeds budget=%d", allocs, tc.maxAllocs)
			}
			if bytes > tc.maxBytes {
				t.Errorf("bytes/op=%d exceeds budget=%d", bytes, tc.maxBytes)
			}
		})
	}
}

func BenchmarkCompositionPerformance(b *testing.B) {
	for _, tc := range compositionPerformanceCases() {
		b.Run(tc.name, func(b *testing.B) {
			h := tc.handler
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://203.0.113.10/resource", nil)
				if err != nil {
					b.Fatal(err)
				}
				resp, err := h(req)
				if err != nil {
					b.Fatal(err)
				}
				_ = resp.Body.Close()
			}
		})
	}
}
