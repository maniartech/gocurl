// Package otel provides OpenTelemetry adapters for gocurl: a gocurl.Tracer
// backed by an OTel TracerProvider, and a propagation Middleware that injects
// W3C traceparent/tracestate headers into outgoing requests. It lives in its own
// module so the core gocurl package stays free of third-party dependencies.
//
// Usage:
//
//	tp := otelsdk.NewTracerProvider(...)
//	client, _ := gocurl.New(
//	    gocurl.WithTracer(gcotel.NewTracer(tp)),
//	    gocurl.WithMiddleware(gcotel.PropagationMiddleware()),
//	)
//
// The Tracer creates one span per logical request (covering all retries). The
// PropagationMiddleware reads the active span from the request context — which
// gocurl's instrumentation places there — and injects the trace headers once, so
// they are preserved across retry clones.
package otel

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/maniartech/gocurl"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/maniartech/gocurl/observability/otel"

// Tracer adapts an OTel TracerProvider to gocurl.Tracer.
type Tracer struct {
	tracer oteltrace.Tracer
}

var _ gocurl.Tracer = (*Tracer)(nil)

// NewTracer builds a gocurl.Tracer from an OTel TracerProvider. A nil provider
// is invalid; pass otel.GetTracerProvider() for the global one.
func NewTracer(tp oteltrace.TracerProvider) *Tracer {
	return &Tracer{tracer: tp.Tracer(instrumentationName)}
}

// StartSpan begins a client span and returns the span-carrying context.
func (t *Tracer) StartSpan(ctx context.Context, name string, attrs ...gocurl.Field) (context.Context, gocurl.Span) {
	ctx, span := t.tracer.Start(ctx, name,
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(toAttrs(attrs)...),
	)
	return ctx, &span0{span: span}
}

// span0 adapts an OTel span to gocurl.Span.
type span0 struct{ span oteltrace.Span }

func (s *span0) SetAttributes(attrs ...gocurl.Field) { s.span.SetAttributes(toAttrs(attrs)...) }

func (s *span0) AddEvent(name string, attrs ...gocurl.Field) {
	s.span.AddEvent(name, oteltrace.WithAttributes(toAttrs(attrs)...))
}

func (s *span0) RecordError(err error) {
	if err == nil {
		return
	}
	s.span.RecordError(err)
	s.span.SetStatus(codes.Error, err.Error())
}

func (s *span0) End() { s.span.End() }

// toAttrs converts gocurl Fields to OTel attributes.
func toAttrs(fields []gocurl.Field) []attribute.KeyValue {
	if len(fields) == 0 {
		return nil
	}
	out := make([]attribute.KeyValue, 0, len(fields))
	for _, f := range fields {
		switch v := f.Value.(type) {
		case nil:
			// skip nil values (e.g. Err(nil))
		case string:
			out = append(out, attribute.String(f.Key, v))
		case bool:
			out = append(out, attribute.Bool(f.Key, v))
		case int:
			out = append(out, attribute.Int(f.Key, v))
		case int64:
			out = append(out, attribute.Int64(f.Key, v))
		case float64:
			out = append(out, attribute.Float64(f.Key, v))
		case time.Duration:
			out = append(out, attribute.String(f.Key, v.String()))
		default:
			out = append(out, attribute.String(f.Key, fmt.Sprint(v)))
		}
	}
	return out
}

// PropagationMiddleware returns a gocurl.Middleware that injects trace context
// (W3C traceparent/tracestate by default) into the outgoing request headers from
// the active span in the request context. Pass a custom propagator to override.
func PropagationMiddleware(propagators ...propagation.TextMapPropagator) gocurl.Middleware {
	var prop propagation.TextMapPropagator = propagation.TraceContext{}
	if len(propagators) > 0 && propagators[0] != nil {
		prop = propagators[0]
	}
	return func(next gocurl.Handler) gocurl.Handler {
		return func(req *http.Request) (*http.Response, error) {
			prop.Inject(req.Context(), propagation.HeaderCarrier(req.Header))
			return next(req)
		}
	}
}
