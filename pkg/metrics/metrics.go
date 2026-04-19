// Package metrics provides HTTP instrumentation using OpenTelemetry metrics.
// Metrics are exported via OTLP to the configured collector (e.g. SigNoz).
package metrics

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds the OTEL meter instruments for HTTP request tracking.
type Metrics struct {
	httpDuration metric.Float64Histogram
	httpTotal    metric.Int64Counter
}

// New creates a Metrics instance with OTEL HTTP instruments.
func New() *Metrics {
	meter := otel.Meter("restful-boilerplate")

	dur, err := meter.Float64Histogram("http.server.duration",
		metric.WithDescription("HTTP request duration in seconds."),
		metric.WithUnit("s"),
	)
	if err != nil {
		slog.Error("create http duration histogram", "err", err)
	}

	total, err := meter.Int64Counter("http.server.requests",
		metric.WithDescription("Total number of HTTP requests."),
	)
	if err != nil {
		slog.Error("create http requests counter", "err", err)
	}

	return &Metrics{httpDuration: dur, httpTotal: total}
}

// Middleware returns HTTP middleware that records request duration and count.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		attrs := metric.WithAttributes(
			attribute.String("http.request.method", r.Method),
			attribute.String("url.path", r.URL.Path),
			attribute.Int("http.response.status_code", sw.status),
		)

		m.httpDuration.Record(r.Context(), time.Since(start).Seconds(), attrs)
		m.httpTotal.Add(r.Context(), 1, attrs)
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
