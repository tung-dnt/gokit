// Package otelecho provides an Echo v5 middleware for OpenTelemetry tracing.
package otelecho

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Middleware returns an Echo v5 middleware that creates a server span per request.
// The global TracerProvider and propagator must be initialised before use
// (e.g. via telemetry.Setup).
func Middleware(serviceName string) echo.MiddlewareFunc {
	tracer := otel.Tracer(serviceName)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			ctx := otel.GetTextMapPropagator().Extract(req.Context(), propagation.HeaderCarrier(req.Header))

			spanName := c.Path()
			if spanName == "" {
				spanName = req.URL.Path
			}

			ctx, span := tracer.Start(ctx,
				fmt.Sprintf("%s %s", req.Method, spanName),
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(req.Method),
					semconv.URLPath(req.URL.Path),
					attribute.String("http.host", req.Host),
				),
			)
			defer span.End()

			c.SetRequest(req.WithContext(ctx))
			err := next(c)

			status := http.StatusOK
			if resp, uErr := echo.UnwrapResponse(c.Response()); uErr == nil {
				status = resp.Status
			}
			span.SetAttributes(semconv.HTTPResponseStatusCode(status))
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else if status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(status))
			}

			return err
		}
	}
}
