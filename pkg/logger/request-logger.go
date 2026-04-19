package logger

import (
	"log/slog"
	"net/http"
	"time"
)

// Middleware returns a net/http middleware that emits a structured JSON log line
// per request with method, path, status code, latency, and — when a span is
// active — trace_id and span_id.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(sw, r)

		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		FromContext(r.Context()).InfoContext(r.Context(), "request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Duration("latency", time.Since(start)),
		)
	})
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status code and delegates to the wrapped ResponseWriter.
func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Write captures a default 200 status on first write and delegates to the wrapped ResponseWriter.
func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}
