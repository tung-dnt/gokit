// Package metrics provides Prometheus instrumentation for the HTTP server.
// It uses a dedicated registry (not prometheus.DefaultRegisterer) so the /metrics
// endpoint is fully self-contained and does not inherit any stray metrics from
// third-party libraries that call prometheus.MustRegister on import.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds the Prometheus registry and HTTP instrumentation collectors.
type Metrics struct {
	reg          *prometheus.Registry
	httpDuration *prometheus.HistogramVec
	httpTotal    *prometheus.CounterVec
}

// New creates a Metrics instance with process, Go runtime, and HTTP collectors.
func New() *Metrics {
	reg := prometheus.NewRegistry()

	dur := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "status"})

	total := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests.",
	}, []string{"method", "path", "status"})

	reg.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsAll),
		),
		dur,
		total,
	)

	return &Metrics{reg: reg, httpDuration: dur, httpTotal: total}
}

// Handler returns an http.Handler that serves Prometheus metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{})
}

// Middleware returns HTTP middleware that records request duration and count.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		status := strconv.Itoa(sw.status)
		elapsed := time.Since(start).Seconds()

		m.httpDuration.WithLabelValues(r.Method, status).Observe(elapsed)
		m.httpTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
	})
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(code int) { //nolint:revive // implements http.ResponseWriter
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(b)
}
