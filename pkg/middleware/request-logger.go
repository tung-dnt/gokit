// Package middlewares wires all application-level Echo middlewares in priority order.
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"

	"restful-boilerplate/pkg/logger"
)

// requestLog emits a structured JSON log line per request with method, path,
// status code, latency, and — when a span is active — trace_id and span_id.
func RequestLog(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		start := time.Now()
		err := next(c)
		status := http.StatusOK
		if resp, uErr := echo.UnwrapResponse(c.Response()); uErr == nil {
			status = resp.Status
		}
		logger.FromContext(c.Request().Context()).Info("request",
			slog.String("method", c.Request().Method),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("latency", time.Since(start)),
		)
		return err
	}
}
