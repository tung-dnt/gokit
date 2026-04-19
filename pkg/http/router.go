package router

import "net/http"

// Router wraps http.ServeMux with middleware chaining and route grouping.
type Router struct {
	base
	Handler http.Handler
}

// NewRouter creates a Router backed by a new http.ServeMux. Options run in
// declaration order and may wrap Router.Handler — e.g. WithInstrumentation
// to attach OpenTelemetry HTTP middleware.
func NewRouter(opts ...Option) *Router {
	mux := http.NewServeMux()
	r := &Router{base: base{mux: mux}, Handler: mux}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// GlobalPrefix sets a global URL prefix applied to all subsequent routes.
func (r *Router) GlobalPrefix(globalPrefix string) {
	r.prefix = globalPrefix
}
