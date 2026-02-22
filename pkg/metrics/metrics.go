package metrics

import (
	"github.com/labstack/echo-contrib/v5/echoprometheus"
	"github.com/labstack/echo/v5"
)

// Register attaches Prometheus middleware and exposes /metrics endpoint.
// Must be called before other middleware so all requests — including error paths — are instrumented.
func Register(e *echo.Echo) {
	cfg := echoprometheus.MiddlewareConfig{
		Skipper: func(c *echo.Context) bool {
			return c.Path() == "/metrics"
		},
		DoNotUseRequestPathFor404: true, // prevents cardinality explosion from unknown paths
	}
	e.Use(echoprometheus.NewMiddlewareWithConfig(cfg))
	e.GET("/metrics", echoprometheus.NewHandler())
}
