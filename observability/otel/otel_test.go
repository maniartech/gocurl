package otel

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/maniartech/gocurl"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func newRecordingTracer() (*Tracer, *tracetest.SpanRecorder) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	return NewTracer(tp), sr
}

func TestTracer_SpanLifecycleAndAttributes(t *testing.T) {
	tr, sr := newRecordingTracer()

	_, span := tr.StartSpan(context.Background(), "gocurl.request",
		gocurl.String("http.method", "GET"),
		gocurl.String("http.host", "api.example.com"),
		gocurl.Err(nil), // nil value must be skipped, not panic
	)
	span.SetAttributes(gocurl.Int("http.status", 200))
	span.AddEvent("retry", gocurl.Int("attempt", 2))
	span.End()

	ended := sr.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected 1 ended span, got %d", len(ended))
	}
	s := ended[0]
	if s.Name() != "gocurl.request" {
		t.Errorf("span name = %q", s.Name())
	}
	attrs := map[string]string{}
	ints := map[string]int64{}
	for _, kv := range s.Attributes() {
		switch kv.Value.Type().String() {
		case "STRING":
			attrs[string(kv.Key)] = kv.Value.AsString()
		case "INT64":
			ints[string(kv.Key)] = kv.Value.AsInt64()
		}
	}
	if attrs["http.method"] != "GET" || attrs["http.host"] != "api.example.com" {
		t.Errorf("start attributes missing: %v", attrs)
	}
	if ints["http.status"] != 200 {
		t.Errorf("SetAttributes int not recorded: %v", ints)
	}
	if len(s.Events()) != 1 || s.Events()[0].Name != "retry" {
		t.Errorf("retry event missing: %+v", s.Events())
	}
}

func TestTracer_RecordError(t *testing.T) {
	tr, sr := newRecordingTracer()
	_, span := tr.StartSpan(context.Background(), "gocurl.request")
	span.RecordError(errors.New("boom"))
	span.End()

	s := sr.Ended()[0]
	if s.Status().Code != codes.Error {
		t.Errorf("status = %v, want Error", s.Status().Code)
	}
	if len(s.Events()) == 0 {
		t.Error("RecordError should add an exception event")
	}

	// RecordError(nil) is a no-op (no panic, no status change).
	tr2, sr2 := newRecordingTracer()
	_, span2 := tr2.StartSpan(context.Background(), "x")
	span2.RecordError(nil)
	span2.End()
	if sr2.Ended()[0].Status().Code == codes.Error {
		t.Error("RecordError(nil) must not set error status")
	}
}

func TestPropagationMiddleware_InjectsTraceparent(t *testing.T) {
	tr, _ := newRecordingTracer()
	// Start a span and put it in the context, as gocurl's instrumentation does.
	ctx, span := tr.StartSpan(context.Background(), "gocurl.request")
	defer span.End()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://api.example.com", nil)
	called := false
	h := PropagationMiddleware()(func(r *http.Request) (*http.Response, error) {
		called = true
		if r.Header.Get("traceparent") == "" {
			t.Error("traceparent header was not injected")
		}
		return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
	})
	if _, err := h(req); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("middleware did not call next")
	}
}

func TestPropagationMiddleware_CustomPropagator(t *testing.T) {
	// A nil propagator falls back to the default; a custom one is honored.
	mw := PropagationMiddleware(propagation.TraceContext{})
	if mw == nil {
		t.Fatal("middleware should not be nil")
	}
}

func TestTracer_SatisfiesInterface(t *testing.T) {
	tr, _ := newRecordingTracer()
	var _ gocurl.Tracer = tr
}
