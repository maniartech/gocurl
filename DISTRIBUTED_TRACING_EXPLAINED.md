# Distributed Tracing Explained

**Date**: October 14, 2025
**For**: Understanding why Context is essential for enterprise/military systems

---

## What is Distributed Tracing?

### The Problem: Modern Systems are Complex

**Traditional monolith:**
```
User → Web Server → Database
       (single request, easy to debug)
```

**Modern microservices:**
```
User → API Gateway → Auth Service → User Service → Database
                   ↓                ↓
                   Payment Service → Inventory Service → Database
                   ↓
                   Email Service
                   ↓
                   Notification Service
```

**One user request touches 7+ services!**

**Question**: When a request fails or is slow, which service is the problem?

### The Solution: Distributed Tracing

**Distributed tracing** tracks a single request across multiple services by:
1. Generating a unique **Trace ID** for each user request
2. Propagating that ID through all services
3. Each service logs its actions with the same Trace ID
4. Tools like Jaeger/Zipkin visualize the full request flow

---

## Example: E-commerce Checkout

### Scenario

User clicks "Buy Now" → Request fails after 5 seconds

**Without tracing:**
- ❓ Which service is slow? Auth? Payment? Inventory?
- ❓ Check logs in 7 different services
- ❓ No correlation between service logs
- 😫 **Hours to debug**

**With distributed tracing:**
```
Trace ID: abc123-def456-ghi789

API Gateway    [0ms]     ─┐
Auth Service   [50ms]     │
User Service   [100ms]    ├─ All share same Trace ID
Payment Svc    [4800ms] ◄─┘ SLOW! Found the problem!
Inventory Svc  [timeout]
```

✅ **5 minutes to debug** - immediately see Payment Service took 4.8 seconds!

---

## How Context Enables Distributed Tracing

### The Magic of context.Context

In Go, `context.Context` carries **trace metadata** across function calls and service boundaries:

```go
// Service 1: API Gateway
ctx := context.Background()
ctx, span := tracer.Start(ctx, "checkout-request")
span.SetAttributes(attribute.String("trace.id", "abc123"))

// Call next service, passing Context
makePaymentRequest(ctx, paymentData)
span.End()

// Service 2: Payment Service (different process/server)
func makePaymentRequest(ctx context.Context, data PaymentData) {
    // Extract trace ID from context
    span := trace.SpanFromContext(ctx)
    // This span has SAME trace ID: "abc123"

    // Make HTTP request to Payment API
    opts := &options.RequestOptions{
        URL:     "https://payment-api.internal/charge",
        Context: ctx, // ✅ Propagates trace ID automatically!
    }

    gocurl.Process(ctx, opts)
}
```

### How It Works

1. **Generate Trace ID** in first service (API Gateway)
2. **Inject into Context** using OpenTelemetry
3. **Pass Context** to all function calls
4. **HTTP headers propagate trace** between services:
   ```
   Traceparent: 00-abc123def456-789ghi-01
   ```
5. **Next service extracts trace** from headers
6. **Continues same trace** in new service

---

## Real-World Example: GoCurl with OpenTelemetry

### Without Context (Broken Tracing)

```go
// Service A: User Service
func GetUserProfile(userID string) (*User, error) {
    // Start trace
    ctx, span := tracer.Start(context.Background(), "get-user")
    defer span.End()

    // Call external API
    opts := &options.RequestOptions{
        URL: "https://api.internal/users/" + userID,
        // ❌ No Context! Trace is broken here!
    }

    resp, _, err := gocurl.Process(context.Background(), opts)
    // External API has NO IDEA this is part of trace "abc123"
    // Distributed tracing is BROKEN
}
```

**Result:** Each service has disconnected traces, can't see full request flow.

### With Context (Working Tracing)

```go
// Service A: User Service
func GetUserProfile(ctx context.Context, userID string) (*User, error) {
    // Start trace (uses parent context with trace ID)
    ctx, span := tracer.Start(ctx, "get-user")
    defer span.End()

    // Call external API
    opts := &options.RequestOptions{
        URL:     "https://api.internal/users/" + userID,
        Context: ctx, // ✅ Propagates trace ID!
    }

    resp, _, err := gocurl.Process(ctx, opts)
    // External API receives trace ID in headers
    // Distributed tracing WORKS
}
```

**Result:** Full request flow visible across all services!

---

## Visualization: Distributed Trace in Jaeger

### What You See in Jaeger UI

```
Trace ID: abc123-def456-ghi789
Total Duration: 523ms

┌─ API Gateway (10ms) ────────────────────────────┐
│  ├─ Auth Service (45ms)                         │
│  ├─ User Service (120ms)                        │
│  │  └─ GET /users/123 (gocurl) (100ms) ◄───────┼─── GoCurl request
│  ├─ Payment Service (310ms)                     │
│  │  ├─ Validate Card (50ms)                     │
│  │  └─ Charge Card (250ms)                      │
│  │     └─ POST /charges (gocurl) (240ms) ◄──────┼─── GoCurl request
│  └─ Inventory Service (38ms)                    │
│     └─ GET /inventory/sku123 (gocurl) (30ms) ◄──┼─── GoCurl request
└──────────────────────────────────────────────────┘

All 3 GoCurl requests show in SAME trace because Context propagated!
```

**Benefits:**
- See which service is slow (Payment: 310ms)
- See which HTTP calls are slow (Charge Card: 240ms)
- Full request timeline across all services
- Click any span to see details (headers, errors, etc.)

---

## Why Military/Enterprise Need This

### Military Use Case: Classified System

**Scenario:** Missile defense system processes threat assessment

```
Radar Data → Threat Analysis → Target Classification → Launch Decision
    ↓              ↓                    ↓                    ↓
 Sensor API    ML Service         Database            Command System
```

**Without distributed tracing:**
- Threat assessment takes 3 seconds (too slow!)
- Which component is slow? Unknown
- Can't optimize without visibility
- Mission failure risk

**With distributed tracing:**
```
Trace ID: mission-alpha-001

Radar Data       [10ms]
Threat Analysis  [2800ms] ◄─── BOTTLENECK! ML model is slow
Target Class.    [50ms]
Launch Decision  [140ms]
```

- Immediately identified: ML Service is bottleneck (2.8s)
- Optimize ML model or add more servers
- Reduced latency to 500ms
- ✅ Mission success

### Enterprise Use Case: E-commerce Peak Traffic

**Black Friday: 100,000 requests/second**

**Without distributed tracing:**
- Checkout failures increase to 15%
- Users complain about slowness
- Dev team checks logs in 20 services
- Takes 4 hours to find issue
- Lost revenue: $500,000

**With distributed tracing:**
- Identify payment service timeout (5s) in 2 minutes
- See it's calling external fraud detection API (slow)
- Increase timeout, add caching
- Checkout failures drop to 0.1%
- ✅ Saved $500,000

---

## How GoCurl Context Integration Works

### The Flow

```go
// 1. User's application starts trace
func HandleCheckout(w http.ResponseWriter, r *http.Request) {
    // Extract trace from incoming HTTP request
    ctx := otel.GetTextMapPropagator().Extract(r.Context(),
        propagation.HeaderCarrier(r.Header))

    // Start span for this operation
    ctx, span := tracer.Start(ctx, "checkout")
    defer span.End()

    // Call payment API using GoCurl
    processPayment(ctx, paymentData)
}

// 2. GoCurl propagates trace to next service
func processPayment(ctx context.Context, data PaymentData) error {
    opts := &options.RequestOptions{
        URL:     "https://payment-api.internal/charge",
        Method:  "POST",
        Body:    jsonBody,
        Context: ctx, // ✅ Contains trace ID from step 1
    }

    // GoCurl creates HTTP request
    resp, _, err := gocurl.Process(ctx, opts)

    // Behind the scenes, GoCurl does:
    // req, _ := http.NewRequestWithContext(ctx, ...)
    // This automatically injects trace headers:
    //   Traceparent: 00-abc123def456-span789-01

    return err
}

// 3. Payment API receives request with trace headers
func PaymentAPIHandler(w http.ResponseWriter, r *http.Request) {
    // Extract trace from headers
    ctx := otel.GetTextMapPropagator().Extract(r.Context(),
        propagation.HeaderCarrier(r.Header))

    // Continue same trace (same trace ID: abc123)
    ctx, span := tracer.Start(ctx, "process-payment")
    defer span.End()

    // All logging/metrics share same trace ID
    span.SetAttributes(attribute.String("payment.amount", "99.99"))
}
```

### The Magic Headers

When GoCurl uses Context, it automatically injects headers:

```http
POST /charge HTTP/1.1
Host: payment-api.internal
Traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
Tracestate: congo=t61rcWkgMzE
```

**These headers:**
- `Traceparent`: Trace ID + Span ID + flags
- `Tracestate`: Vendor-specific trace data

**Payment API extracts these** and continues the trace!

---

## GoCurl's Role in Distributed Tracing

### What GoCurl Already Does ✅

```go
// options/options.go (line 78)
Context context.Context `json:"-"`
```

```go
// process.go
func Process(ctx context.Context, opts *options.RequestOptions) {
    if opts.Context == nil {
        opts.Context = ctx
    }

    // Create request WITH context
    req, err := CreateRequest(opts.Context, opts)
    // This calls: http.NewRequestWithContext(opts.Context, ...)
    // Which automatically propagates trace headers!
}
```

**GoCurl already supports distributed tracing!**

### What OpenTelemetry Does

OpenTelemetry (the tracing library) uses Context to:

1. **Store trace metadata** (trace ID, span ID)
2. **Inject HTTP headers** automatically when you use Context
3. **Extract HTTP headers** when receiving requests
4. **Send trace data** to backends (Jaeger, Zipkin, etc.)

**GoCurl doesn't need to do anything special** - just use the Context!

---

## Example: Full Distributed Tracing Setup

### Setup (One Time)

```go
package main

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/trace"
)

func initTracer() {
    // Export traces to Jaeger
    exporter, _ := jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint("http://localhost:14268/api/traces"),
    ))

    // Create trace provider
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
    )

    otel.SetTracerProvider(tp)
}

func main() {
    initTracer()

    // Your application code
    handleUserRequest()
}
```

### Usage in Your Code

```go
import (
    "go.opentelemetry.io/otel"
    "github.com/maniartech/gocurl"
)

func handleUserRequest() {
    tracer := otel.Tracer("my-service")

    // Start trace for user request
    ctx, span := tracer.Start(context.Background(), "user-checkout")
    defer span.End()

    // Call multiple services - all share same trace!
    getUserProfile(ctx, userID)
    processPayment(ctx, paymentData)
    updateInventory(ctx, items)
    sendConfirmationEmail(ctx, email)
}

func getUserProfile(ctx context.Context, userID string) (*User, error) {
    // Start sub-span (child of user-checkout span)
    ctx, span := tracer.Start(ctx, "get-user-profile")
    defer span.End()

    // Call external API
    opts := &options.RequestOptions{
        URL:     "https://user-api.internal/users/" + userID,
        Context: ctx, // ✅ Trace propagates automatically!
    }

    resp, body, err := gocurl.Process(ctx, opts)

    // Trace automatically includes:
    // - This HTTP request duration
    // - Status code
    // - Any errors

    return parseUser(body), err
}

func processPayment(ctx context.Context, data PaymentData) error {
    ctx, span := tracer.Start(ctx, "process-payment")
    defer span.End()

    opts := &options.RequestOptions{
        URL:     "https://payment-api.internal/charge",
        Method:  "POST",
        Context: ctx, // ✅ Same trace ID continues!
        Hooks: &options.RequestHooks{
            OnRequestEnd: func(ctx, resp, duration, err) {
                // Add custom attributes to trace
                span.SetAttributes(
                    attribute.String("payment.gateway", "stripe"),
                    attribute.Float64("payment.amount", data.Amount),
                    attribute.Int("http.status", resp.StatusCode),
                )
            },
        },
    }

    _, _, err := gocurl.Process(ctx, opts)
    return err
}
```

### What You See in Jaeger

```
Trace: user-checkout
Duration: 850ms

├─ user-checkout (850ms) ─────────────────────────┐
│  ├─ get-user-profile (120ms)                    │
│  │  └─ HTTP GET /users/123 (100ms)              │
│  │     Status: 200                               │
│  │                                               │
│  ├─ process-payment (680ms)                     │
│  │  └─ HTTP POST /charge (650ms)                │
│  │     Status: 200                               │
│  │     Attributes:                               │
│  │       payment.gateway: stripe                 │
│  │       payment.amount: 99.99                   │
│  │                                               │
│  ├─ update-inventory (30ms)                     │
│  │  └─ HTTP PUT /inventory (25ms)               │
│  │                                               │
│  └─ send-confirmation (20ms)                    │
│     └─ HTTP POST /emails (15ms)                 │
└──────────────────────────────────────────────────┘
```

**Click any span** to see:
- Request/response headers
- Request/response bodies (if logged)
- Errors and stack traces
- Custom attributes
- Timing breakdown

---

## Why This Matters for GoCurl

### Enterprise/Military Requirements

✅ **Visibility** - See request flow across all services
✅ **Performance** - Identify bottlenecks instantly
✅ **Debugging** - Correlate logs across services
✅ **Compliance** - Audit trail with trace IDs
✅ **SLAs** - Measure end-to-end latency

### Without Context Support

```go
// ❌ GoCurl breaks distributed tracing
opts := &options.RequestOptions{
    URL: "https://api.example.com",
}
gocurl.Process(context.Background(), opts)
// Trace is broken - can't correlate this HTTP call with parent trace
```

**Enterprise can't use GoCurl!** (Breaks their tracing infrastructure)

### With Context Support (Already Have!)

```go
// ✅ GoCurl integrates seamlessly with distributed tracing
opts := &options.RequestOptions{
    URL:     "https://api.example.com",
    Context: ctx, // Trace propagates automatically
}
gocurl.Process(ctx, opts)
```

**Enterprise loves GoCurl!** (Works with their existing OpenTelemetry setup)

---

## Summary

### What is Distributed Tracing?

**Tracking a single user request across multiple services** with a unique Trace ID.

### Why Context?

**Go's `context.Context` carries trace metadata** (trace ID, span ID) across function calls and HTTP requests.

### What GoCurl Already Does

```go
Context context.Context `json:"-"` // ✅ Already in RequestOptions!
```

**This single field enables:**
- OpenTelemetry integration ✅
- Jaeger/Zipkin tracing ✅
- Request correlation across services ✅
- Performance monitoring ✅
- Debug visibility ✅

### Why It's Critical

**Without Context:** Enterprise/military can't use GoCurl (breaks tracing)
**With Context:** GoCurl integrates seamlessly with their infrastructure

---

## Bottom Line

**You already have Context support!** This makes GoCurl enterprise/military-grade because it integrates with distributed tracing systems that are **mandatory** in production environments.

**Next step:** Add Hooks for observability metrics (duration, status codes, retries) while Context handles distributed tracing automatically.

**The combination:**
- Context = Distributed tracing (where did request go?)
- Hooks = Metrics/logging (how long did it take? did it fail?)

Both are essential for production systems!
