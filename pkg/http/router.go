package router

import "net/http"

// Router wraps http.ServeMux with middleware chaining and route grouping.
type Router struct {
	base
	Handler http.Handler
}

// NewRouter creates a Router backed by a new http.ServeMux.
func NewRouter() *Router {
	mux := http.NewServeMux()
	return &Router{base: base{mux: mux}, Handler: mux}
}

// GlobalPrefix sets a global URL prefix applied to all subsequent routes.
func (r *Router) GlobalPrefix(globalPrefix string) {
	r.prefix = globalPrefix
}
