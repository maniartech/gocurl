# Enterprise & Military-Grade Observability Architecture

**Date**: October 14, 2025
**Context**: Rethinking metrics/observability for production-grade systems
**Perspective**: Enterprise SRE, DevOps, Military-grade reliability

---

## Enterprise Requirements Analysis

### What Enterprise/Military Systems Actually Need

#### 1. **Observability Pillars**

**Metrics** (quantitative data):
- Request latency (p50, p95, p99)
- Throughput (req/s)
- Error rates
- Resource utilization

**Tracing** (request flow):
- Distributed tracing (OpenTelemetry)
- Span context propagation
- Service dependency mapping

**Logging** (contextual events):
- Structured logging
- Request/response correlation
- Error context capture

#### 2. **Production Requirements**

✅ **Zero performance overhead** when observability disabled
✅ **Pluggable backends** - Prometheus, DataDog, New Relic, Jaeger, etc.
✅ **Context propagation** - trace IDs, span IDs, baggage
✅ **Compliance** - audit logs, data retention, PII redaction
✅ **High availability** - observability failure must not break requests
✅ **Battle-tested** - proven under extreme load

#### 3. **Security Requirements**

✅ **Data redaction** - no secrets in logs/metrics
✅ **Audit trail** - who, what, when, why
✅ **Compliance** - HIPAA, SOC2, FedRAMP
✅ **Least privilege** - minimal data collection

---

## Critical Insight: Observability is NOT Metrics

### The Mistake

Thinking "metrics = observability" is **reductive**.

### Enterprise Reality

**Observability Stack**:
```
┌─────────────────────────────────────────┐
│         Application Code                │
│  (gocurl making HTTP requests)          │
└──────────────┬──────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────┐
│    Observability Instrumentation        │
│  (OpenTelemetry, Prometheus, etc.)      │
└──────────────┬──────────────────────────┘
               │
      ┌────────┴────────┬────────────────┐
      ▼                 ▼                ▼
┌──────────┐    ┌──────────┐    ┌──────────┐
│ Metrics  │    │  Traces  │    │   Logs   │
│(Prometheus)   │(Jaeger)  │    │(ELK)     │
└──────────┘    └──────────┘    └──────────┘
```

**GoCurl should integrate with this stack, not replace it!**

---

## What Enterprise/Military DON'T Want

### ❌ Anti-Pattern 1: Predefined Metrics Struct

```go
// BAD - Forces our metrics format
type RequestMetrics struct {
    StartTime     time.Time
    EndTime       time.Time
    Duration      time.Duration
    DNSLookupTime time.Duration
    // ... 8 more fields
}
```

**Why this fails in enterprise**:

1. **Not compatible with existing systems**
   - Enterprise uses Prometheus (different format)
   - Military uses custom metrics systems (classified)
   - Teams use DataDog, New Relic, Dynatrace (all different)

2. **Can't extend**
   - What if enterprise needs custom labels? (region, environment, team)
   - What if military needs classification tags? (secret, top-secret)
   - What if SaaS needs tenant IDs?

3. **Coupling**
   - Forces users to adapt their systems to our struct
   - Can't swap monitoring vendors without code changes

### ❌ Anti-Pattern 2: Built-in Metrics Collection

```go
// BAD - Library decides what to measure
if opts.Metrics != nil {
    opts.Metrics.DNSLookupTime = measureDNS()
    opts.Metrics.ConnectTime = measureConnect()
}
```

**Why this fails**:

1. **Performance overhead**
   - Measuring DNS/TLS adds httptrace hooks (allocations)
   - Enterprise may not need this granularity
   - Overhead on every request (not acceptable)

2. **Not configurable**
   - Can't disable specific metrics
   - Can't sample (measure 1% of requests)
   - Can't filter by endpoint

3. **Wrong abstraction layer**
   - Observability is cross-cutting concern
   - Should be handled by infrastructure, not library

---

## Enterprise/Military-Grade Solution

### Principle: **Instrumentation Points, Not Metrics Collection**

**Library provides hooks, users implement collection.**

### Architecture: Hooks + Context Propagation

```go
// 1. Context Propagation (for distributed tracing)
type RequestOptions struct {
    // ... existing fields ...

    // Context carries trace IDs, span IDs, and baggage
    Context context.Context

    // OPTIONAL: Hooks for observability
    OnRequestStart  func(ctx context.Context, req *http.Request)
    OnRequestEnd    func(ctx context.Context, resp *http.Response, err error)
    OnRetry         func(ctx context.Context, attempt int, err error)
    OnError         func(ctx context.Context, err error)
}
```

### Why This is Military-Grade

#### 1. **Zero Overhead When Disabled**

```go
// No hooks = zero allocation, zero overhead
opts := &options.RequestOptions{
    URL: "https://api.example.com",
}
// OnRequestStart, OnRequestEnd are nil - no performance impact
```

#### 2. **Pluggable Backend**

```go
// Works with ANY monitoring system
opts.OnRequestEnd = func(ctx context.Context, resp *http.Response, err error) {
    // Prometheus
    httpDuration.Observe(time.Since(ctx.Value("start_time").(time.Time)).Seconds())

    // OR DataDog
    statsd.Timing("http.request", time.Since(startTime))

    // OR custom military system
    classifiedMetrics.Record(ctx, resp)
}
```

#### 3. **Context Propagation for Distributed Tracing**

```go
// OpenTelemetry integration
import "go.opentelemetry.io/otel"

ctx, span := tracer.Start(ctx, "api-call")
defer span.End()

opts := &options.RequestOptions{
    URL:     "https://api.example.com",
    Context: ctx, // Propagates trace ID automatically
    OnRequestEnd: func(ctx context.Context, resp *http.Response, err error) {
        span.SetAttributes(
            attribute.Int("http.status_code", resp.StatusCode),
            attribute.String("http.url", opts.URL),
        )
        if err != nil {
            span.RecordError(err)
        }
    },
}
```

#### 4. **Sampling for High-Throughput Systems**

```go
// Sample 1% of requests for detailed metrics
var sampler = rand.New(rand.NewSource(time.Now().UnixNano()))

opts.OnRequestEnd = func(ctx context.Context, resp *http.Response, err error) {
    if sampler.Float64() < 0.01 { // 1% sampling
        detailedMetrics.Record(ctx, resp, err)
    }
    // Always record basic counters
    requestCounter.Inc()
}
```

#### 5. **Security & Compliance**

```go
opts.OnRequestStart = func(ctx context.Context, req *http.Request) {
    // Audit logging (compliance)
    auditLog.LogRequest(ctx, req, redact(req.Header.Get("Authorization")))
}

opts.OnRequestEnd = func(ctx context.Context, resp *http.Response, err error) {
    // PII redaction
    if resp.StatusCode >= 400 {
        securityLog.LogFailure(ctx, resp, redactPII(resp.Body))
    }
}
```

---

## Proposed Enterprise Architecture

### Core: Context + Lifecycle Hooks

```go
// options/options.go
type RequestOptions struct {
    // ... existing 30+ fields ...

    // Context for distributed tracing, cancellation, and values
    Context context.Context

    // OPTIONAL lifecycle hooks for observability
    // These are called at key points in request lifecycle
    // All hooks are optional (nil-safe)
    Hooks *RequestHooks
}

// RequestHooks provides observability integration points
type RequestHooks struct {
    // OnRequestStart is called after request is created, before sending
    // Use for: start timers, inject headers, audit logging
    OnRequestStart func(ctx context.Context, req *http.Request)

    // OnRequestEnd is called after request completes (success or failure)
    // Use for: metrics collection, distributed tracing, error logging
    OnRequestEnd func(ctx context.Context, resp *http.Response, duration time.Duration, err error)

    // OnRetry is called before each retry attempt
    // Use for: retry metrics, backoff logging
    OnRetry func(ctx context.Context, attempt int, lastErr error)

    // OnDNSStart/OnDNSEnd for detailed DNS metrics (optional)
    OnDNSStart func(ctx context.Context, host string)
    OnDNSEnd   func(ctx context.Context, addrs []net.IP, err error)

    // OnConnectStart/OnConnectEnd for connection metrics (optional)
    OnConnectStart func(ctx context.Context, network, addr string)
    OnConnectEnd   func(ctx context.Context, network, addr string, err error)

    // OnTLSHandshakeStart/OnTLSHandshakeEnd for TLS metrics (optional)
    OnTLSHandshakeStart func(ctx context.Context)
    OnTLSHandshakeEnd   func(ctx context.Context, state tls.ConnectionState, err error)
}
```

### Implementation in process.go

```go
func Process(ctx context.Context, opts *options.RequestOptions) (*http.Response, string, error) {
    // Use provided context or create new one
    if opts.Context == nil {
        opts.Context = ctx
    }

    // ... validation ...

    // Create request
    req, err := CreateRequest(opts.Context, opts)
    if err != nil {
        return nil, "", err
    }

    // Hook: Request start
    if opts.Hooks != nil && opts.Hooks.OnRequestStart != nil {
        opts.Hooks.OnRequestStart(opts.Context, req)
    }

    // Execute with timing
    startTime := time.Now()
    resp, err := executeWithRetries(httpClient, req, opts)
    duration := time.Since(startTime)

    // Hook: Request end (always called, even on error)
    if opts.Hooks != nil && opts.Hooks.OnRequestEnd != nil {
        opts.Hooks.OnRequestEnd(opts.Context, resp, duration, err)
    }

    if err != nil {
        if opts.Hooks != nil && opts.Hooks.OnError != nil {
            opts.Hooks.OnError(opts.Context, err)
        }
        return nil, "", err
    }

    // ... rest of processing ...

    return resp, bodyString, nil
}
```

### Implementation in retry.go

```go
func executeWithRetries(client options.HTTPClient, req *http.Request, opts *options.RequestOptions) (*http.Response, error) {
    // ... existing retry logic ...

    for attempt := 0; attempt <= retries; attempt++ {
        // Hook: Retry attempt
        if attempt > 0 && opts.Hooks != nil && opts.Hooks.OnRetry != nil {
            opts.Hooks.OnRetry(opts.Context, attempt, lastErr)
        }

        resp, err = client.Do(attemptReq)

        // ... retry decision logic ...
    }

    return resp, nil
}
```

---

## Enterprise Integration Examples

### Example 1: OpenTelemetry (Industry Standard)

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func MakeTracedRequest(url string) error {
    ctx := context.Background()
    tracer := otel.Tracer("gocurl")

    ctx, span := tracer.Start(ctx, "http-request")
    defer span.End()

    opts := &options.RequestOptions{
        URL:     url,
        Context: ctx,
        Hooks: &options.RequestHooks{
            OnRequestStart: func(ctx context.Context, req *http.Request) {
                span.SetAttributes(
                    attribute.String("http.method", req.Method),
                    attribute.String("http.url", req.URL.String()),
                )
            },
            OnRequestEnd: func(ctx context.Context, resp *http.Response, duration time.Duration, err error) {
                if resp != nil {
                    span.SetAttributes(
                        attribute.Int("http.status_code", resp.StatusCode),
                        attribute.Int64("http.response_size", resp.ContentLength),
                    )
                }
                if err != nil {
                    span.RecordError(err)
                    span.SetStatus(codes.Error, err.Error())
                }
                span.SetAttributes(attribute.Int64("http.duration_ms", duration.Milliseconds()))
            },
            OnRetry: func(ctx context.Context, attempt int, lastErr error) {
                span.AddEvent("retry", trace.WithAttributes(
                    attribute.Int("retry.attempt", attempt),
                    attribute.String("retry.reason", lastErr.Error()),
                ))
            },
        },
    }

    _, _, err := gocurl.Process(ctx, opts)
    return err
}
```

### Example 2: Prometheus (Metrics Standard)

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "status_code", "endpoint"},
    )

    httpRequestTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "status_code", "endpoint"},
    )

    httpRetryTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_retries_total",
            Help: "Total number of HTTP retries",
        },
        []string{"endpoint"},
    )
)

func init() {
    prometheus.MustRegister(httpRequestDuration, httpRequestTotal, httpRetryTotal)
}

func MakeMonitoredRequest(url string) error {
    opts := &options.RequestOptions{
        URL: url,
        Hooks: &options.RequestHooks{
            OnRequestEnd: func(ctx context.Context, resp *http.Response, duration time.Duration, err error) {
                statusCode := "error"
                if resp != nil {
                    statusCode = strconv.Itoa(resp.StatusCode)
                }

                labels := prometheus.Labels{
                    "method":      opts.Method,
                    "status_code": statusCode,
                    "endpoint":    opts.URL,
                }

                httpRequestDuration.With(labels).Observe(duration.Seconds())
                httpRequestTotal.With(labels).Inc()
            },
            OnRetry: func(ctx context.Context, attempt int, lastErr error) {
                httpRetryTotal.WithLabelValues(opts.URL).Inc()
            },
        },
    }

    _, _, err := gocurl.Process(context.Background(), opts)
    return err
}
```

### Example 3: DataDog (SaaS Monitoring)

```go
import "github.com/DataDog/datadog-go/statsd"

var ddClient *statsd.Client

func init() {
    var err error
    ddClient, err = statsd.New("127.0.0.1:8125")
    if err != nil {
        log.Fatal(err)
    }
}

func MakeDataDogMonitoredRequest(url string) error {
    opts := &options.RequestOptions{
        URL: url,
        Hooks: &options.RequestHooks{
            OnRequestEnd: func(ctx context.Context, resp *http.Response, duration time.Duration, err error) {
                tags := []string{
                    "service:api-client",
                    "endpoint:" + opts.URL,
                }

                if resp != nil {
                    tags = append(tags, "status:"+strconv.Itoa(resp.StatusCode))
                }

                ddClient.Timing("http.request.duration", duration, tags, 1)
                ddClient.Incr("http.request.count", tags, 1)

                if err != nil {
                    ddClient.Incr("http.request.errors", tags, 1)
                }
            },
        },
    }

    _, _, err := gocurl.Process(context.Background(), opts)
    return err
}
```

### Example 4: Military/Custom Classified System

```go
// Hypothetical classified metrics system
type ClassifiedMetricsCollector struct {
    classification string
    project        string
}

func (c *ClassifiedMetricsCollector) RecordRequest(ctx context.Context, resp *http.Response, duration time.Duration, err error) {
    // Custom metrics format with classification tags
    metric := ClassifiedMetric{
        Classification: c.classification,
        Project:        c.project,
        Timestamp:      time.Now(),
        Duration:       duration,
        Success:        err == nil && resp.StatusCode < 400,
        // NO sensitive data - URLs, headers, bodies redacted
    }

    // Send to classified metrics backend
    classifiedBackend.Submit(metric)
}

func MakeMilitaryGradeRequest(url string) error {
    collector := &ClassifiedMetricsCollector{
        classification: "SECRET",
        project:        "PROJECT_PHOENIX",
    }

    opts := &options.RequestOptions{
        URL: url,
        Hooks: &options.RequestHooks{
            OnRequestEnd: func(ctx context.Context, resp *http.Response, duration time.Duration, err error) {
                collector.RecordRequest(ctx, resp, duration, err)
            },
        },
    }

    _, _, err := gocurl.Process(context.Background(), opts)
    return err
}
```

---

## Comparison: Three Approaches

| Aspect | Metrics Field | OnResponse Callback | Hooks + Context |
|--------|--------------|---------------------|-----------------|
| **Enterprise Ready** | ❌ No | ⚠️ Basic | ✅ **Production-grade** |
| **Distributed Tracing** | ❌ No | ❌ No | ✅ **OpenTelemetry** |
| **Pluggable Backends** | ❌ Locked format | ⚠️ Limited | ✅ **Any system** |
| **Zero Overhead** | ❌ Always allocates | ✅ When nil | ✅ **When nil** |
| **Sampling Support** | ❌ No | ⚠️ Manual | ✅ **Built-in** |
| **Retry Metrics** | ❌ Limited | ❌ No | ✅ **OnRetry hook** |
| **Security/Audit** | ❌ No | ⚠️ Manual | ✅ **OnRequestStart** |
| **Compliance** | ❌ No redaction | ❌ No redaction | ✅ **User controls** |
| **Complexity** | Low | Very Low | Medium |
| **Implementation** | 8+ hours | 5 minutes | 2 hours |

**Winner for Enterprise/Military**: **Hooks + Context** (production-grade observability)

---

## Final Recommendation: Three-Tier Strategy

### Tier 1: Context Propagation (Immediate - Beta)

**Add to RequestOptions:**
```go
Context context.Context // For distributed tracing, cancellation
```

**Why**: Foundation for enterprise observability

### Tier 2: Request Lifecycle Hooks (Beta)

**Add to RequestOptions:**
```go
Hooks *RequestHooks // Optional observability hooks
```

**Hooks struct:**
```go
type RequestHooks struct {
    OnRequestStart func(ctx context.Context, req *http.Request)
    OnRequestEnd   func(ctx context.Context, resp *http.Response, duration time.Duration, err error)
    OnRetry        func(ctx context.Context, attempt int, lastErr error)
}
```

**Why**: Covers 95% of enterprise use cases

### Tier 3: Advanced Hooks (Post v1.0, if needed)

**Optional detailed hooks:**
```go
OnDNSStart/OnDNSEnd
OnConnectStart/OnConnectEnd
OnTLSHandshakeStart/OnTLSHandshakeEnd
```

**Why**: Rarely needed, adds complexity

---

## Implementation Priority

### P0 (Beta - Oct 18):
1. ✅ **Remove `Metrics *RequestMetrics`** (overkill, wrong abstraction)
2. ✅ **Add `Context context.Context`** (distributed tracing foundation)
3. ✅ **Add basic `Hooks` struct** (OnRequestStart, OnRequestEnd, OnRetry)

### P1 (v1.0 - Nov 8):
1. Document OpenTelemetry integration examples
2. Document Prometheus integration examples
3. Document security/audit patterns

### P2 (Post v1.0):
1. Advanced hooks (DNS, TLS) only if requested
2. Helper package for common integrations

---

## Conclusion: Military-Grade = Instrumentation, Not Metrics

**Key Insight**: Enterprise/military systems already have observability infrastructure (Prometheus, Jaeger, DataDog, classified systems).

**GoCurl should**:
- ✅ Provide hooks for instrumentation
- ✅ Propagate context for distributed tracing
- ✅ Stay out of the way (zero overhead when not used)

**GoCurl should NOT**:
- ❌ Implement metrics collection
- ❌ Force metrics format
- ❌ Choose monitoring backend

**Decision**: Remove Metrics field, implement Context + Hooks architecture for true enterprise/military-grade observability.
