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

// GlobalPrefix sets a global URL prefix applied to all subsequent routes.
func (s *Router) GlobalPrefix(globalPrefix string) *error {
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

// ANY registers a single handler for the given pattern.
func (s *Router) ANY(pattern string, h http.Handler) *error {
	s.Router.Handle(prefixPattern(s.prefix, pattern), h)
	return nil
}

// GET registers a handler for GET requests at the given path.
func (s *Router) GET(path string, h http.Handler) *error {
	s.Router.Handle(prefixPattern("GET "+s.prefix, path), h)
	return nil
}

// POST registers a handler for GET requests at the given path.
func (s *Router) POST(path string, h http.Handler) *error {
	s.Router.Handle(prefixPattern("POST "+s.prefix, path), h)
	return nil
}

// PUT registers a handler for GET requests at the given path.
func (s *Router) PUT(path string, h http.Handler) *error {
	s.Router.Handle(prefixPattern("PUT "+s.prefix, path), h)
	return nil
}

// PATCH registers a handler for GET requests at the given path.
func (s *Router) PATCH(path string, h http.Handler) *error {
	s.Router.Handle(prefixPattern("PATCH "+s.prefix, path), h)
	return nil
}

// DELETE registers a handler for GET requests at the given path.
func (s *Router) DELETE(path string, h http.Handler) *error {
	s.Router.Handle(prefixPattern("DELETE "+s.prefix, path), h)
	return nil
}
