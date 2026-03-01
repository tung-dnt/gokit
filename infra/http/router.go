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
func (s *Router) Prefix(globalPrefix string) *error {
	s.prefix = globalPrefix
	return nil
}

// Use wraps the handler chain with the given middleware.
func (s *Router) Use(mw func(http.Handler) http.Handler) *error {
	s.Handler = mw(s.Handler)
	return nil
}

// Group registers routes under a shared prefix via the callback fn.
func (s *Router) Group(prefix string, fn func(*Group)) *error {
	fn(&Group{mux: s.Router, prefix: s.prefix + prefix})
	return nil
}

// Route registers a single handler for the given pattern.
func (s *Router) Route(pattern string, h http.Handler) *error {
	s.Router.Handle(prefixPattern(s.prefix, pattern), h)
	return nil
}
