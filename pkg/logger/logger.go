// Package logger provides structured logging with OpenTelemetry trace correlation.
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/trace"
)

// Setup initialises slog writing to stdout, a log file, and optionally an OTLP exporter.
// logFormat controls stdout output: "pretty" for colorized human-readable, "json" (default) for JSON.
// The file always receives JSON. When logProvider is non-nil, logs are also exported via OTLP.
// Sets slog.Default so callers can use slog package-level functions.
// Returns a close function for the log file.
func Setup(logPath string, logFormat string, logProvider *sdklog.LoggerProvider) (func() error, error) {
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644) //nolint:gosec // path is internal, not user-supplied
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	opts := &slog.HandlerOptions{Level: slog.LevelInfo}

	var stdoutHandler slog.Handler
	if logFormat == "pretty" {
		stdoutHandler = NewPrettyHandler(os.Stdout, opts)
	} else {
		stdoutHandler = slog.NewJSONHandler(os.Stdout, opts)
	}

	fileHandler := slog.NewJSONHandler(f, opts)

	handlers := []slog.Handler{stdoutHandler, fileHandler}
	if logProvider != nil {
		handlers = append(handlers, otelslog.NewHandler("restful-boilerplate",
			otelslog.WithLoggerProvider(logProvider),
			otelslog.WithSource(true),
		))
	}

	slog.SetDefault(slog.New(&fanoutHandler{handlers: handlers}))
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

// fanoutHandler distributes log records to multiple handlers.
type fanoutHandler struct {
	handlers []slog.Handler
}

// Enabled reports whether any underlying handler handles records at the given level.
func (h *fanoutHandler) Enabled(ctx context.Context, l slog.Level) bool {
	for _, hh := range h.handlers {
		if hh.Enabled(ctx, l) {
			return true
		}
	}
	return false
}

// Handle distributes the record to all enabled underlying handlers.
func (h *fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, hh := range h.handlers {
		if hh.Enabled(ctx, r.Level) {
			if err := hh.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

// WithAttrs returns a new fanoutHandler with the given attributes pre-applied to all handlers.
func (h *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cloned := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		cloned[i] = hh.WithAttrs(attrs)
	}
	return &fanoutHandler{handlers: cloned}
}

// WithGroup returns a new fanoutHandler with the given group name applied to all handlers.
func (h *fanoutHandler) WithGroup(name string) slog.Handler {
	cloned := make([]slog.Handler, len(h.handlers))
	for i, hh := range h.handlers {
		cloned[i] = hh.WithGroup(name)
	}
	return &fanoutHandler{handlers: cloned}
}
