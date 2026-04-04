package router

import "net/http"

// base holds the shared state and route-registration logic
// used by both Router and Group.
type base struct {
	mux    *http.ServeMux
	prefix string
	mws    []func(http.Handler) http.Handler
}

// Use appends middleware that wraps all handlers registered on this base.
// Middleware must be added before registering routes.
func (b *base) Use(mw func(http.Handler) http.Handler) {
	b.mws = append(b.mws, mw)
}

// Prefix adds a prefix affecting all future routes.
func (b *base) Prefix(prefix string) {
	b.prefix += prefix
}

// Group creates a sub-group with an additional prefix.
// The sub-group inherits the current middleware chain.
func (b *base) Group(prefix string, fn func(*Group)) {
	sub := &Group{base: base{
		mux:    b.mux,
		prefix: b.prefix + prefix,
		mws:    make([]func(http.Handler) http.Handler, len(b.mws)),
	}}
	copy(sub.mws, b.mws)
	fn(sub)
}

// ANY registers a handler for all HTTP methods at the given pattern.
func (b *base) ANY(pattern string, h http.HandlerFunc) {
	b.mux.Handle(b.prefix+pattern, b.wrap(h))
}

// GET registers a handler for GET requests at the given path.
func (b *base) GET(path string, h http.HandlerFunc) {
	b.mux.Handle("GET "+b.prefix+path, b.wrap(h))
}

// POST registers a handler for POST requests at the given path.
func (b *base) POST(path string, h http.HandlerFunc) {
	b.mux.Handle("POST "+b.prefix+path, b.wrap(h))
}

// PUT registers a handler for PUT requests at the given path.
func (b *base) PUT(path string, h http.HandlerFunc) {
	b.mux.Handle("PUT "+b.prefix+path, b.wrap(h))
}

// PATCH registers a handler for PATCH requests at the given path.
func (b *base) PATCH(path string, h http.HandlerFunc) {
	b.mux.Handle("PATCH "+b.prefix+path, b.wrap(h))
}

// DELETE registers a handler for DELETE requests at the given path.
func (b *base) DELETE(path string, h http.HandlerFunc) {
	b.mux.Handle("DELETE "+b.prefix+path, b.wrap(h))
}

// wrap applies the middleware chain to a handler.
// First middleware added via Use is the outermost wrapper.
func (b *base) wrap(h http.Handler) http.Handler {
	for i := len(b.mws) - 1; i >= 0; i-- {
		h = b.mws[i](h)
	}
	return h
}

// Group scopes route registration under a shared URL prefix
// with optional group-level middleware.
type Group struct {
	base
}
