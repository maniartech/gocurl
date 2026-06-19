package gocurl

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Observability (Spec 06)
//
// Three vendor-neutral, dependency-free primitives — Tracer, Metrics, Logger —
// plus lightweight lifecycle Hooks make every request observable. They are
// wired through a single internal instrumentation middleware that brackets the
// whole logical request (all retry attempts, the breaker and limiter). When no
// sink or hook is configured the middleware is not installed at all, so the
// disabled path is byte-identical to an un-instrumented client.
//
// The error classification consumed here is the M4 Kind taxonomy (see
// errors.go / error_classify.go) rather than a separate enum — one source of
// truth. On the success latency observation, ResultInfo.Kind is KindUnknown
// (i.e. "no error").

// Level is a structured log severity.
type Level int8

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the lowercase level name.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "info"
	}
}

// Field is a single structured key/value pair for logs and span attributes.
// Values placed in a Field that leaves the process via a sink must already be
// redacted; the core only ever passes pre-redacted values.
type Field struct {
	Key   string
	Value any
}

// String builds a string Field.
func String(k, v string) Field { return Field{Key: k, Value: v} }

// Int builds an int Field.
func Int(k string, v int) Field { return Field{Key: k, Value: v} }

// Duration builds a time.Duration Field.
func Duration(k string, v time.Duration) Field { return Field{Key: k, Value: v} }

// Err builds an error Field under the key "error" (nil-safe).
func Err(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Logger is a minimal structured logger. Implementations must be safe for
// concurrent use. A nil Logger disables logging.
type Logger interface {
	Log(ctx context.Context, level Level, msg string, fields ...Field)
}

// Tracer creates a span around one logical request (covering all retry
// attempts). A nil Tracer disables tracing.
type Tracer interface {
	// StartSpan returns a child context carrying the span and the span itself.
	StartSpan(ctx context.Context, name string, attrs ...Field) (context.Context, Span)
}

// Span is the active span for an in-flight request. End is always called once.
type Span interface {
	SetAttributes(attrs ...Field)
	AddEvent(name string, attrs ...Field)
	RecordError(err error)
	End()
}

// Metrics collects request-level measurements. Implementations must be safe for
// concurrent use and must not block. A nil Metrics disables metrics. Labels are
// low-cardinality and pre-redacted (method, host, status, error kind only).
type Metrics interface {
	IncRequest(info RequestInfo)                     // total logical requests started
	IncInFlight(delta int)                           // +1 at start, -1 at end
	ObserveLatency(d time.Duration, info ResultInfo) // wall time to response headers
	IncRetry(info RequestInfo)                       // one per retry attempt beyond the first
	IncError(kind Kind, info RequestInfo)            // one per failed logical request
}

// RequestInfo carries low-cardinality, pre-redacted request labels.
type RequestInfo struct {
	Method string
	Host   string // host only — never the full URL (cardinality + secrets)
}

// ResultInfo carries the outcome labels for a completed logical request. Kind is
// KindUnknown on success.
type ResultInfo struct {
	RequestInfo
	StatusCode int  // 0 if no response was received
	Kind       Kind // KindUnknown on success
}

// Hooks are optional, synchronous lifecycle callbacks. Each is nil-safe. They
// receive the live request/response (which the caller already owns); anything a
// hook forwards to a sink must be redacted by the caller.
type Hooks struct {
	OnRequest  func(ctx context.Context, req *http.Request)
	OnRetry    func(ctx context.Context, req *http.Request, attempt int, lastErr error, lastResp *http.Response)
	OnResponse func(ctx context.Context, req *http.Request, resp *http.Response, elapsed time.Duration)
	OnError    func(ctx context.Context, req *http.Request, err error, kind Kind)
}

func (h Hooks) any() bool {
	return h.OnRequest != nil || h.OnRetry != nil || h.OnResponse != nil || h.OnError != nil
}

// --- Options ---

// WithTracer sets the request Tracer (nil disables tracing).
func WithTracer(t Tracer) Option {
	return func(c *config) error { c.tracer = t; return nil }
}

// WithMetrics sets the request Metrics collector (nil disables metrics).
func WithMetrics(m Metrics) Option {
	return func(c *config) error { c.metrics = m; return nil }
}

// WithLogger sets the structured Logger (nil disables logging).
func WithLogger(l Logger) Option {
	return func(c *config) error { c.logger = l; return nil }
}

// WithHooks sets the lifecycle hooks.
func WithHooks(h Hooks) Option {
	return func(c *config) error { c.hooks = h; return nil }
}

// WithRequestIDFunc sets a generator used to produce an X-Request-ID when a
// request does not already carry one.
func WithRequestIDFunc(fn func() string) Option {
	return func(c *config) error { c.requestIDFunc = fn; return nil }
}

// --- No-op sinks (used for unset interfaces so call sites never branch) ---

type noopTracer struct{}

func (noopTracer) StartSpan(ctx context.Context, name string, attrs ...Field) (context.Context, Span) {
	return ctx, noopSpan{}
}

type noopSpan struct{}

func (noopSpan) SetAttributes(...Field)    {}
func (noopSpan) AddEvent(string, ...Field) {}
func (noopSpan) RecordError(error)         {}
func (noopSpan) End()                      {}

type noopMetrics struct{}

func (noopMetrics) IncRequest(RequestInfo)                   {}
func (noopMetrics) IncInFlight(int)                          {}
func (noopMetrics) ObserveLatency(time.Duration, ResultInfo) {}
func (noopMetrics) IncRetry(RequestInfo)                     {}
func (noopMetrics) IncError(Kind, RequestInfo)               {}

// --- Resolved observability bundle held by a Client ---

// obs is the resolved observability configuration. tracer and metrics are never
// nil (no-op defaults); logger may be nil; hooks fields are individually nil-safe.
type obs struct {
	tracer        Tracer
	metrics       Metrics
	logger        Logger
	hooks         Hooks
	requestIDFunc func() string
	active        bool // any sink/hook configured => install instrumentation
}

func resolveObs(c *config) *obs {
	o := &obs{
		tracer:        c.tracer,
		metrics:       c.metrics,
		logger:        c.logger,
		hooks:         c.hooks,
		requestIDFunc: c.requestIDFunc,
	}
	o.active = c.tracer != nil || c.metrics != nil || c.logger != nil ||
		c.requestIDFunc != nil || c.hooks.any()
	if o.tracer == nil {
		o.tracer = noopTracer{}
	}
	if o.metrics == nil {
		o.metrics = noopMetrics{}
	}
	return o
}

// safe runs fn, recovering any panic from a user-supplied sink so a buggy sink
// can never take down the request. The recovery is itself panic-guarded so a
// panicking logger cannot escape.
func (o *obs) safe(fn func()) {
	defer func() {
		if r := recover(); r != nil && o.logger != nil {
			func() {
				defer func() { _ = recover() }()
				o.logger.Log(context.Background(), LevelError,
					"gocurl: observability sink panicked", String("panic", fmt.Sprint(r)))
			}()
		}
	}()
	fn()
}

// log emits a structured request log line through the configured Logger (no-op
// when unset). Headers are redacted and the URL is sanitized.
func (o *obs) log(ctx context.Context, level Level, req *http.Request, resp *http.Response, elapsed time.Duration, err error, requestID string) {
	if o.logger == nil {
		return
	}
	fields := make([]Field, 0, 6)
	fields = append(fields,
		String("http.method", req.Method),
		String("http.url", sanitizeURL(req.URL.String())),
		Duration("elapsed", elapsed),
	)
	if requestID != "" {
		fields = append(fields, String("request.id", requestID))
	}
	if resp != nil {
		fields = append(fields, Int("http.status", resp.StatusCode))
	}
	if err != nil {
		// Scrub: a raw *url.Error embeds the full URL (query secrets) verbatim.
		fields = append(fields, String("error", scrubErrorString(err.Error())))
	}
	o.safe(func() { o.logger.Log(ctx, level, "gocurl.request", fields...) })
}

// --- Per-retry hook plumbing (context-carried, keeps retry.go decoupled) ---

type retryHookKey struct{}

type retryHookFunc func(attempt int, lastErr error, lastResp *http.Response)

func withRetryHook(ctx context.Context, fn retryHookFunc) context.Context {
	return context.WithValue(ctx, retryHookKey{}, fn)
}

func retryHookFromContext(ctx context.Context) retryHookFunc {
	if ctx == nil {
		return nil
	}
	fn, _ := ctx.Value(retryHookKey{}).(retryHookFunc)
	return fn
}

// --- Instrumentation middleware ---

// hostOf returns the request host without port/userinfo (low cardinality, no
// secrets), the value used for metric/span labels.
func hostOf(req *http.Request) string {
	if req.URL == nil {
		return ""
	}
	return req.URL.Hostname()
}

// resolveRequestID keeps an existing X-Request-ID (from opts.RequestID or a
// prior hop), otherwise generates one via the configured func and sets it. The
// header is set once and preserved across retry clones.
func (o *obs) resolveRequestID(req *http.Request) string {
	if id := req.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	if o.requestIDFunc != nil {
		var id string
		o.safe(func() { id = o.requestIDFunc() }) // a panicking generator must not crash the request
		if id != "" {
			req.Header.Set("X-Request-ID", id)
			return id
		}
	}
	return ""
}

// instrument wraps next with span/timing/metrics/hooks/logging that bracket the
// whole logical request. It is installed only when obs.active.
func (o *obs) instrument(next Handler) Handler {
	return func(req *http.Request) (*http.Response, error) {
		ctx := req.Context()
		info := RequestInfo{Method: req.Method, Host: hostOf(req)}

		var span Span = noopSpan{}
		o.safe(func() {
			ctx, span = o.tracer.StartSpan(ctx, "gocurl.request", String("http.method", info.Method), String("http.host", info.Host))
		})
		if span == nil {
			span = noopSpan{}
		}
		defer o.safe(func() { span.End() })

		requestID := o.resolveRequestID(req)
		if requestID != "" {
			o.safe(func() { span.SetAttributes(String("request.id", requestID)) })
		}

		// Per-retry observability, fired from inside the retry loop.
		ctx = withRetryHook(ctx, func(attempt int, lastErr error, lastResp *http.Response) {
			o.safe(func() { o.metrics.IncRetry(info) })
			o.safe(func() { span.AddEvent("retry", Int("attempt", attempt), Err(lastErr)) })
			if o.hooks.OnRetry != nil {
				rctx := ctx
				o.safe(func() { o.hooks.OnRetry(rctx, req, attempt, lastErr, lastResp) })
			}
		})
		req = req.WithContext(ctx)

		o.safe(func() { o.metrics.IncRequest(info) })
		o.safe(func() { o.metrics.IncInFlight(1) })
		defer o.safe(func() { o.metrics.IncInFlight(-1) })
		if o.hooks.OnRequest != nil {
			o.safe(func() { o.hooks.OnRequest(ctx, req) })
		}

		start := time.Now()
		resp, err := next(req)
		elapsed := time.Since(start)

		if err != nil {
			// Classify the raw transport error: instrumentation runs inside the
			// chain, before wrapTransportError types the error at the Do boundary.
			kind := classifyKind(err)
			// Record a SCRUBBED error on the span: a raw *url.Error embeds the full
			// URL (query secrets) verbatim, and the span is a core sink bound by the
			// redaction guarantee. classifyToError's Error() runs scrubErrorString.
			scrubbed := classifyToError(err)
			o.safe(func() { span.RecordError(scrubbed) })
			o.safe(func() { o.metrics.IncError(kind, info) })
			o.safe(func() { o.metrics.ObserveLatency(elapsed, ResultInfo{RequestInfo: info, StatusCode: 0, Kind: kind}) })
			if o.hooks.OnError != nil {
				o.safe(func() { o.hooks.OnError(ctx, req, err, kind) })
			}
			o.log(ctx, LevelError, req, resp, elapsed, err, requestID)
			return resp, err
		}

		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		o.safe(func() {
			o.metrics.ObserveLatency(elapsed, ResultInfo{RequestInfo: info, StatusCode: status, Kind: KindUnknown})
		})
		o.safe(func() { span.SetAttributes(Int("http.status", status)) })
		if o.hooks.OnResponse != nil {
			o.safe(func() { o.hooks.OnResponse(ctx, req, resp, elapsed) })
		}
		o.log(ctx, LevelInfo, req, resp, elapsed, nil, requestID)
		return resp, nil
	}
}
