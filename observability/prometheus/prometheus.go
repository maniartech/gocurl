// Package prometheus provides a gocurl.Metrics adapter backed by a Prometheus
// registry. It lives in its own module so the core gocurl package stays free of
// third-party dependencies.
//
// Usage:
//
//	reg := prometheus.NewRegistry()
//	m := gcprom.New(reg)
//	client, _ := gocurl.New(gocurl.WithMetrics(m))
//
// The adapter registers five collectors:
//
//	gocurl_requests_total{method,host}
//	gocurl_in_flight
//	gocurl_request_duration_seconds{method,host,status} (histogram)
//	gocurl_retries_total{method,host}
//	gocurl_errors_total{method,host,kind}
package prometheus

import (
	"strconv"
	"time"

	"github.com/maniartech/gocurl"
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics implements gocurl.Metrics over Prometheus collectors. It is safe for
// concurrent use (Prometheus collectors are).
type Metrics struct {
	requests *prometheus.CounterVec
	inFlight prometheus.Gauge
	duration *prometheus.HistogramVec
	retries  *prometheus.CounterVec
	errors   *prometheus.CounterVec
}

// Compile-time assertion that *Metrics satisfies the core interface.
var _ gocurl.Metrics = (*Metrics)(nil)

// New builds the adapter and registers its collectors with reg. A nil reg skips
// registration (useful when composing into a larger collector). It panics via
// MustRegister if a collector is already registered — pass a fresh registry per
// adapter or handle the panic in caller setup.
func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "gocurl_requests_total",
			Help: "Total number of logical gocurl requests started.",
		}, []string{"method", "host"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "gocurl_in_flight",
			Help: "Number of gocurl requests currently in flight.",
		}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gocurl_request_duration_seconds",
			Help:    "Wall-clock duration of a logical gocurl request to response headers.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "host", "status"}),
		retries: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "gocurl_retries_total",
			Help: "Total number of gocurl retry attempts beyond the first.",
		}, []string{"method", "host"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "gocurl_errors_total",
			Help: "Total number of failed logical gocurl requests, by error kind.",
		}, []string{"method", "host", "kind"}),
	}
	if reg != nil {
		reg.MustRegister(m.requests, m.inFlight, m.duration, m.retries, m.errors)
	}
	return m
}

// IncRequest counts a started logical request.
func (m *Metrics) IncRequest(info gocurl.RequestInfo) {
	m.requests.WithLabelValues(info.Method, info.Host).Inc()
}

// IncInFlight adjusts the in-flight gauge.
func (m *Metrics) IncInFlight(delta int) { m.inFlight.Add(float64(delta)) }

// ObserveLatency records the request duration in seconds.
func (m *Metrics) ObserveLatency(d time.Duration, info gocurl.ResultInfo) {
	m.duration.WithLabelValues(info.Method, info.Host, strconv.Itoa(info.StatusCode)).Observe(d.Seconds())
}

// IncRetry counts a retry attempt.
func (m *Metrics) IncRetry(info gocurl.RequestInfo) {
	m.retries.WithLabelValues(info.Method, info.Host).Inc()
}

// IncError counts a failed request labelled by the gocurl error Kind.
func (m *Metrics) IncError(kind gocurl.Kind, info gocurl.RequestInfo) {
	m.errors.WithLabelValues(info.Method, info.Host, kind.String()).Inc()
}
