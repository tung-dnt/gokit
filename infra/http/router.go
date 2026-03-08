package router

import "net/http"

// Router wraps http.ServeMux with middleware chaining and route grouping.
type Router struct {
	Router  *http.ServeMux
	Handler http.Handler
	prefix  string
	mws     []func(http.Handler) http.Handler
}

// NewRouter creates a Router backed by a new http.ServeMux.
func NewRouter() *Router {
	mux := http.NewServeMux()
	return &Router{Router: mux, Handler: mux}
}

// GlobalPrefix sets a global URL prefix applied to all subsequent routes.
func (r *Router) GlobalPrefix(globalPrefix string) *error {
	r.prefix = globalPrefix
	return nil
}

// Use wraps the handler chain with the given middleware.
func (r *Router) Use(mw func(http.Handler) http.Handler) *error {
	r.mws = append(r.mws, mw)
	return nil
}

// Group registers routes under a shared prefix via the callback fn.
func (r *Router) Group(prefix string, fn func(*Group)) *error {
	fn(&Group{mux: r.Router, prefix: r.prefix + prefix})
	return nil
}

// ANY registers a single handler for the given pattern.
func (r *Router) ANY(pattern string, h http.Handler) *error {
	r.Router.Handle(r.prefix+pattern, r.wrap(h))
	return nil
}

// GET registers a handler for GET requests at the given path.
func (r *Router) GET(path string, h http.Handler) *error {
	r.Router.Handle("GET "+r.prefix+path, r.wrap(h))
	return nil
}

// POST registers a handler for GET requests at the given path.
func (r *Router) POST(path string, h http.Handler) *error {
	r.Router.Handle("POST "+r.prefix+path, r.wrap(h))
	return nil
}

// PUT registers a handler for GET requests at the given path.
func (r *Router) PUT(path string, h http.Handler) *error {
	r.Router.Handle("PUT "+r.prefix+path, r.wrap(h))
	return nil
}

// PATCH registers a handler for GET requests at the given path.
func (r *Router) PATCH(path string, h http.Handler) *error {
	r.Router.Handle("PATCH "+r.prefix+path, r.wrap(h))
	return nil
}

// DELETE registers a handler for GET requests at the given path.
func (r *Router) DELETE(path string, h http.Handler) *error {
	r.Router.Handle("DELETE "+r.prefix+path, r.wrap(h))
	return nil
}

// wrap applies the group's middleware chain to a handler.
// First middleware added via Use is the outermost wrapper.
func (r *Router) wrap(h http.Handler) http.Handler {
	for i := len(r.mws) - 1; i >= 0; i-- {
		h = r.mws[i](h)
	}
	return h
}
