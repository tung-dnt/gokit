// Package recovery provides HTTP middleware that recovers from panics.
package recovery

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"restful-boilerplate/pkg/logger"
	"restful-boilerplate/pkg/telemetry"
)

// Middleware recovers from panics, annotates the active OTel span with the
// panic + stack trace, logs via the trace-correlated slog logger, and responds
// with 500 Internal Server Error. Place this inside the otelhttp middleware so
// a server span already exists in the request context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			ctx := r.Context()
			err := fmt.Errorf("panic: %v", rec)
			stack := debug.Stack()

			span := trace.SpanFromContext(ctx)
			span.RecordError(err, trace.WithStackTrace(true))
			span.SetStatus(codes.Error, "panic")
			span.SetAttributes(attribute.String("error.type", telemetry.ErrKindPanic))

			logger.FromContext(ctx).ErrorContext(ctx, "panic recovered",
				"error", err,
				"stack", string(stack),
			)

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}()
		next.ServeHTTP(w, r)
	})
}
