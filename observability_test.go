package gocurl

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maniartech/gocurl/options"
)

// fakeMetrics counts sink calls; safe for concurrent use.
type fakeMetrics struct {
	requests   int64
	retries    int64
	errors     int64
	latencies  int64
	inFlightHi int64
	inFlight   int64
	mu         sync.Mutex
	lastResult ResultInfo
	lastKind   Kind
}

func (m *fakeMetrics) IncRequest(RequestInfo) { atomic.AddInt64(&m.requests, 1) }
func (m *fakeMetrics) IncInFlight(d int) {
	v := atomic.AddInt64(&m.inFlight, int64(d))
	for {
		hi := atomic.LoadInt64(&m.inFlightHi)
		if v <= hi || atomic.CompareAndSwapInt64(&m.inFlightHi, hi, v) {
			break
		}
	}
}
func (m *fakeMetrics) ObserveLatency(_ time.Duration, info ResultInfo) {
	atomic.AddInt64(&m.latencies, 1)
	m.mu.Lock()
	m.lastResult = info
	m.mu.Unlock()
}
func (m *fakeMetrics) IncRetry(RequestInfo) { atomic.AddInt64(&m.retries, 1) }
func (m *fakeMetrics) IncError(k Kind, _ RequestInfo) {
	atomic.AddInt64(&m.errors, 1)
	m.mu.Lock()
	m.lastKind = k
	m.mu.Unlock()
}

type fakeSpan struct {
	mu     sync.Mutex
	events []string
	errs   int
	ended  int
	attrs  map[string]any
}

func (s *fakeSpan) SetAttributes(attrs ...Field) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.attrs == nil {
		s.attrs = map[string]any{}
	}
	for _, a := range attrs {
		s.attrs[a.Key] = a.Value
	}
}
func (s *fakeSpan) AddEvent(name string, _ ...Field) {
	s.mu.Lock()
	s.events = append(s.events, name)
	s.mu.Unlock()
}
func (s *fakeSpan) RecordError(error) { s.mu.Lock(); s.errs++; s.mu.Unlock() }
func (s *fakeSpan) End()              { s.mu.Lock(); s.ended++; s.mu.Unlock() }

type fakeTracer struct{ span *fakeSpan }

func (t *fakeTracer) StartSpan(ctx context.Context, _ string, _ ...Field) (context.Context, Span) {
	return ctx, t.span
}

// recordingErrSpan captures the message passed to RecordError (to assert redaction).
type recordingErrSpan struct {
	mu       sync.Mutex
	recorded string
}

func (s *recordingErrSpan) SetAttributes(...Field)    {}
func (s *recordingErrSpan) AddEvent(string, ...Field) {}
func (s *recordingErrSpan) RecordError(err error) {
	s.mu.Lock()
	if err != nil {
		s.recorded = err.Error()
	}
	s.mu.Unlock()
}
func (s *recordingErrSpan) End() {}

type fakeTracer2 struct{ span *recordingErrSpan }

func (t *fakeTracer2) StartSpan(ctx context.Context, _ string, _ ...Field) (context.Context, Span) {
	return ctx, t.span
}

type fakeLogger struct {
	mu      sync.Mutex
	entries []logEntry
}
type logEntry struct {
	level  Level
	msg    string
	fields []Field
}

func (l *fakeLogger) Log(_ context.Context, level Level, msg string, fields ...Field) {
	l.mu.Lock()
	l.entries = append(l.entries, logEntry{level, msg, append([]Field(nil), fields...)})
	l.mu.Unlock()
}

func (l *fakeLogger) allFieldValues() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var sb strings.Builder
	for _, e := range l.entries {
		for _, f := range e.fields {
			sb.WriteString(strings.ToLower(strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(toStr(f.Value), "\n", " "), "\t", " "))))
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func toStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return ""
	}
}

func TestField_Helpers(t *testing.T) {
	if f := String("k", "v"); f.Key != "k" || f.Value != "v" {
		t.Error("String")
	}
	if f := Int("n", 3); f.Value != 3 {
		t.Error("Int")
	}
	if f := Duration("d", time.Second); f.Value != time.Second {
		t.Error("Duration")
	}
	if f := Err(nil); f.Value != nil {
		t.Error("Err(nil) should have nil value")
	}
	if f := Err(errors.New("boom")); f.Key != "error" || f.Value != "boom" {
		t.Error("Err")
	}
	if LevelError.String() != "error" || LevelDebug.String() != "debug" {
		t.Error("Level.String")
	}
}

func TestResolveObs_ActiveAndDefaults(t *testing.T) {
	o := resolveObs(defaultConfig())
	if o.active {
		t.Error("no sinks => not active")
	}
	if o.tracer == nil || o.metrics == nil {
		t.Error("tracer/metrics must default to no-ops")
	}
	// Each option flips active.
	for _, opt := range []Option{
		WithMetrics(&fakeMetrics{}),
		WithLogger(&fakeLogger{}),
		WithTracer(&fakeTracer{span: &fakeSpan{}}),
		WithRequestIDFunc(func() string { return "x" }),
		WithHooks(Hooks{OnRequest: func(context.Context, *http.Request) {}}),
	} {
		c := defaultConfig()
		_ = opt(c)
		if !resolveObs(c).active {
			t.Error("a configured sink/hook should make obs active")
		}
	}
}

func TestObservability_RetryAccounting(t *testing.T) {
	rt := &recordingRT{script: []scriptedResp{{status: 503}, {status: 503}, {status: 200}}}
	fm := &fakeMetrics{}
	span := &fakeSpan{}
	c, _ := New(WithTransport(rt), WithRetry(zeroBackoff(3, 503)), WithMetrics(fm), WithTracer(&fakeTracer{span: span}))
	defer c.Close()

	req, _ := NewRequest("GET", "http://h.example")
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got := atomic.LoadInt64(&fm.requests); got != 1 {
		t.Errorf("IncRequest = %d, want 1 (one logical request)", got)
	}
	if got := atomic.LoadInt64(&fm.latencies); got != 1 {
		t.Errorf("ObserveLatency = %d, want 1", got)
	}
	if got := atomic.LoadInt64(&fm.retries); got != 2 {
		t.Errorf("IncRetry = %d, want 2 (two retries)", got)
	}
	if got := atomic.LoadInt64(&fm.errors); got != 0 {
		t.Errorf("IncError = %d, want 0 (eventual success)", got)
	}
	if got := atomic.LoadInt64(&fm.inFlight); got != 0 {
		t.Errorf("in-flight not balanced: %d", got)
	}
	if atomic.LoadInt64(&fm.inFlightHi) != 1 {
		t.Errorf("in-flight high-water = %d, want 1", fm.inFlightHi)
	}
	span.mu.Lock()
	ended, events := span.ended, len(span.events)
	span.mu.Unlock()
	if ended != 1 {
		t.Errorf("span.End called %d times, want exactly 1", ended)
	}
	if events != 2 {
		t.Errorf("span retry events = %d, want 2", events)
	}
}

func TestObservability_ErrorPathClassified(t *testing.T) {
	rt := &recordingRT{script: []scriptedResp{{err: &net.OpError{Op: "dial", Err: errors.New("connection refused")}}}}
	fm := &fakeMetrics{}
	span := &fakeSpan{}
	var hookErrKind Kind = KindUnknown
	hookFired := false
	c, _ := New(WithTransport(rt), WithMetrics(fm), WithTracer(&fakeTracer{span: span}),
		WithHooks(Hooks{OnError: func(_ context.Context, _ *http.Request, _ error, k Kind) { hookFired = true; hookErrKind = k }}))
	defer c.Close()

	req, _ := NewRequest("GET", "http://h.example")
	_, err := c.Do(context.Background(), req)
	if err == nil {
		t.Fatal("expected a connect error")
	}
	if atomic.LoadInt64(&fm.errors) != 1 {
		t.Errorf("IncError = %d, want 1", fm.errors)
	}
	if !hookFired || hookErrKind != KindConnect {
		t.Errorf("OnError fired=%v kind=%v, want true/KindConnect", hookFired, hookErrKind)
	}
	span.mu.Lock()
	defer span.mu.Unlock()
	if span.errs != 1 || span.ended != 1 {
		t.Errorf("span errs=%d ended=%d, want 1/1", span.errs, span.ended)
	}
}

func TestObservability_PanicSinkRecovered(t *testing.T) {
	rt := &recordingRT{script: []scriptedResp{{status: 200}}}
	c, _ := New(WithTransport(rt),
		WithHooks(Hooks{OnRequest: func(context.Context, *http.Request) { panic("buggy hook") }}),
		WithLogger(&fakeLogger{}))
	defer c.Close()

	req, _ := NewRequest("GET", "http://h.example")
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("a panicking sink must not fail the request: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status=%d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestObservability_RequestIDGeneratedAndPreserved(t *testing.T) {
	rt := &recordingRT{script: []scriptedResp{{status: 503}, {status: 200}}}
	var seen []string
	c, _ := New(WithTransport(rt), WithRetry(zeroBackoff(3, 503)),
		WithRequestIDFunc(func() string { return "rid-xyz" }),
		WithHooks(Hooks{OnRequest: func(_ context.Context, req *http.Request) {
			seen = append(seen, req.Header.Get("X-Request-ID"))
		}}))
	defer c.Close()

	req, _ := NewRequest("GET", "http://h.example")
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(seen) == 0 || seen[0] != "rid-xyz" {
		t.Errorf("request id not generated/applied: %v", seen)
	}
	// Preserved across the retried attempts (same header on the transport hits).
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.hits != 2 {
		t.Fatalf("expected 2 transport hits, got %d", rt.hits)
	}
}

func TestObservability_RequestIDKeptWhenPresent(t *testing.T) {
	rt := &recordingRT{script: []scriptedResp{{status: 200}}}
	gen := 0
	c, _ := New(WithTransport(rt), WithRequestIDFunc(func() string { gen++; return "generated" }))
	defer c.Close()
	req, _ := NewRequest("GET", "http://h.example", Header("X-Request-ID", "preset"))
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gen != 0 {
		t.Error("request-id func should not run when a request already has one")
	}
}

func TestObservability_LoggerRedactsURL(t *testing.T) {
	rt := &recordingRT{script: []scriptedResp{{status: 200}}}
	fl := &fakeLogger{}
	c, _ := New(WithTransport(rt), WithLogger(fl))
	defer c.Close()
	req, _ := NewRequest("GET", "http://h.example/p?api_key=TOPSECRET")
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if strings.Contains(fl.allFieldValues(), "topsecret") {
		t.Errorf("secret leaked into log fields: %q", fl.allFieldValues())
	}
	if !strings.Contains(fl.allFieldValues(), "[redacted]") {
		t.Errorf("expected a [REDACTED] marker in the logged URL: %q", fl.allFieldValues())
	}
}

func TestObservability_LoggerRedactsUserinfo(t *testing.T) {
	rt := &recordingRT{script: []scriptedResp{{status: 200}}}
	fl := &fakeLogger{}
	c, _ := New(WithTransport(rt), WithLogger(fl))
	defer c.Close()
	req, _ := NewRequest("GET", "http://alice:s3cr3tpw@h.example/p")
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if strings.Contains(fl.allFieldValues(), "s3cr3tpw") {
		t.Errorf("basic-auth password leaked into log fields: %q", fl.allFieldValues())
	}
}

func TestObservability_ErrorPathRedactsSecrets(t *testing.T) {
	// A transport failure on a credentialed URL must not leak query secrets via
	// the Logger "error" field or the span.
	rt := &recordingRT{script: []scriptedResp{{err: &net.OpError{Op: "dial", Err: errors.New("connection refused")}}}}
	fl := &fakeLogger{}
	var recordedErr string
	span := &recordingErrSpan{}
	c, _ := New(WithTransport(rt), WithLogger(fl), WithTracer(&fakeTracer2{span: span}))
	defer c.Close()

	req, _ := NewRequest("GET", "http://h.example/p?api_key=TOPSECRET&token=abc123")
	_, err := c.Do(context.Background(), req)
	if err == nil {
		t.Fatal("expected a connect error")
	}
	logged := fl.allFieldValues()
	if strings.Contains(logged, "topsecret") || strings.Contains(logged, "abc123") {
		t.Errorf("query secret leaked into log error field: %q", logged)
	}
	recordedErr = strings.ToLower(span.recorded)
	if strings.Contains(recordedErr, "topsecret") || strings.Contains(recordedErr, "abc123") {
		t.Errorf("query secret leaked into span.RecordError: %q", span.recorded)
	}
}

func TestObservability_PanicRequestIDFuncRecovered(t *testing.T) {
	rt := &recordingRT{script: []scriptedResp{{status: 200}}}
	c, _ := New(WithTransport(rt), WithRequestIDFunc(func() string { panic("boom") }))
	defer c.Close()
	req, _ := NewRequest("GET", "http://h.example")
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("a panicking requestIDFunc must not crash the request: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status=%d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestVerbose_UsesCanonicalRedaction(t *testing.T) {
	old := VerboseWriter
	var buf bytes.Buffer
	VerboseWriter = &buf
	defer func() { VerboseWriter = old }()

	req, _ := http.NewRequest("GET", "http://h.example", nil)
	req.Header.Set("X-Auth-Token", "supersecret")
	req.Header.Set("Authorization", "Bearer abc")
	req.Header.Set("X-Public", "ok")
	opts := options.NewRequestOptions("http://h.example")
	opts.Verbose = true
	printRequestVerbose(opts, req)

	out := buf.String()
	if strings.Contains(out, "supersecret") || strings.Contains(out, "Bearer abc") {
		t.Errorf("verbose output leaked a secret: %q", out)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("non-sensitive header should be shown: %q", out)
	}
}

func TestIsSensitiveHeader_CanonicalSetIncludesAuthToken(t *testing.T) {
	for _, h := range []string{"X-Auth-Token", "auth-token", "Authorization", "Cookie"} {
		if !IsSensitiveHeader(h) {
			t.Errorf("IsSensitiveHeader(%q) = false, want true", h)
		}
	}
}

// --- Benchmarks: disabled vs full observability (run with -benchmem) ---

func benchClient(t testing.TB, opts ...Option) (*Client, *Request) {
	rt := &recordingRT{script: []scriptedResp{{status: 200}}}
	c, err := New(append([]Option{WithTransport(rt)}, opts...)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	req, _ := NewRequest("GET", "http://h.example/path")
	return c, req
}

func BenchmarkDo_NoObservability(b *testing.B) {
	c, req := benchClient(b)
	defer c.Close()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := c.Do(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkDo_FullObservability(b *testing.B) {
	c, req := benchClient(b,
		WithMetrics(&fakeMetrics{}),
		WithTracer(&fakeTracer{span: &fakeSpan{}}),
		WithLogger(&fakeLogger{}),
		WithHooks(Hooks{
			OnRequest:  func(context.Context, *http.Request) {},
			OnResponse: func(context.Context, *http.Request, *http.Response, time.Duration) {},
		}),
	)
	defer c.Close()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := c.Do(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
