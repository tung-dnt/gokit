package router

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Option configures a Router at construction time. Options run in order and
// may wrap or replace the underlying http.Handler exposed via Router.Handler.
type Option func(*Router)

// WithInstrumentation wraps the router with OpenTelemetry HTTP instrumentation
// (server spans + http.server.request.duration histogram). The span name uses
// the matched route template (Go 1.22+ ServeMux pattern) so cardinality stays
// bounded — falls back to the raw path if no pattern matched.
//
// Apply this last among options so the OTEL handler sits at the outermost
// position and captures every request, including ones that 404 inside the mux.
func WithInstrumentation(serviceName string) Option {
	return func(r *Router) {
		r.Handler = otelhttp.NewHandler(r.Handler, serviceName,
			otelhttp.WithSpanNameFormatter(func(_ string, req *http.Request) string {
				if p := req.Pattern; p != "" {
					return req.Method + " " + p
				}
				return req.Method + " " + req.URL.Path
			}),
		)
	}
}
