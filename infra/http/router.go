package router

import (
	"net/http"
)

// Router wraps http.ServeMux with middleware chaining and route grouping.
type Router struct {
	Router  *http.ServeMux
	Handler http.Handler
	prefix  string
}

// NewRouter creates a Router backed by a new http.ServeMux.
func NewRouter() *Router {
	mux := http.NewServeMux()
	return &Router{Router: mux, Handler: mux}
}

// Prefix sets a global URL prefix applied to all subsequent routes.
func (s *Router) Prefix(globalPrefix string) *Router {
	s.prefix = globalPrefix
	return s
}

// Use wraps the handler chain with the given middleware.
func (s *Router) Use(mw func(http.Handler) http.Handler) *Router {
	s.Handler = mw(s.Handler)
	return s
}

// Group registers routes under a shared prefix via the callback fn.
func (s *Router) Group(prefix string, fn func(*Group)) *Router {
	fn(&Group{mux: s.Router, prefix: s.prefix + prefix})
	return s
}

// Route registers a single handler for the given pattern.
func (s *Router) Route(pattern string, h http.Handler) *Router {
	s.Router.Handle(prefixPattern(s.prefix, pattern), h)
	return s
}
