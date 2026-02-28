// Package logger provides structured JSON logging with OpenTelemetry trace correlation.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// Setup initialises slog writing to both stdout and the given file path.
// Sets slog.Default so callers can use slog package-level functions.
// Returns a close function for the log file.
func Setup(logPath string) (func() error, error) {
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644) //nolint:gosec // path is internal, not user-supplied
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	w := io.MultiWriter(os.Stdout, f)
	slog.SetDefault(slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})))
	return f.Close, nil
}

// FromContext returns slog.Default pre-populated with trace_id and span_id from ctx.
// Falls back to slog.Default with no extra fields when no valid span is present.
func FromContext(ctx context.Context) *slog.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return slog.Default()
	}
	return slog.Default().With(
		slog.String("trace_id", span.SpanContext().TraceID().String()),
		slog.String("span_id", span.SpanContext().SpanID().String()),
	)
}
