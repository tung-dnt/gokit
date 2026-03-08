package router

import "net/http"

// Group scopes route registration under a shared URL prefix
// with optional group-level middleware.
type Group struct {
	mux    *http.ServeMux
	prefix string
	mws    []func(http.Handler) http.Handler
}

// Use appends middleware that wraps all handlers registered in this group.
// Middleware must be added before registering routes.
func (r *Group) Use(mw func(http.Handler) http.Handler) {
	r.mws = append(r.mws, mw)
}

// Group creates a nested sub-group with an additional prefix.
// The sub-group inherits this group's middleware chain.
func (r *Group) Group(prefix string, fn func(*Group)) {
	sub := &Group{
		mux:    r.mux,
		prefix: r.prefix + prefix,
		mws:    make([]func(http.Handler) http.Handler, len(r.mws)),
	}
	copy(sub.mws, r.mws)
	fn(sub)
}

// Prefix adds a prefix to this group, affecting all existing and future routes.
func (r *Group) Prefix(prefix string) {
	r.prefix += prefix
}

// ANY registers a single handler for the given pattern.
func (r *Group) ANY(pattern string, h http.HandlerFunc) *error {
	r.mux.Handle(r.prefix+pattern, r.wrap(h))
	return nil
}

// GET registers a handler for GET requests at the given path.
func (r *Group) GET(path string, h http.HandlerFunc) *error {
	r.mux.Handle("GET "+r.prefix+path, r.wrap(h))
	return nil
}

// POST registers a handler for GET requests at the given path.
func (r *Group) POST(path string, h http.HandlerFunc) *error {
	r.mux.Handle("POST "+r.prefix+path, r.wrap(h))
	return nil
}

// PUT registers a handler for GET requests at the given path.
func (r *Group) PUT(path string, h http.HandlerFunc) *error {
	r.mux.Handle("PUT "+r.prefix+path, r.wrap(h))
	return nil
}

// PATCH registers a handler for GET requests at the given path.
func (r *Group) PATCH(path string, h http.HandlerFunc) *error {
	r.mux.Handle("PATCH "+r.prefix+path, r.wrap(h))
	return nil
}

// DELETE registers a handler for GET requests at the given path.
func (r *Group) DELETE(path string, h http.HandlerFunc) *error {
	r.mux.Handle("DELETE "+r.prefix+path, r.wrap(h))
	return nil
}

// wrap applies the group's middleware chain to a handler.
// First middleware added via Use is the outermost wrapper.
func (r *Group) wrap(h http.Handler) http.Handler {
	for i := len(r.mws) - 1; i >= 0; i-- {
		h = r.mws[i](h)
	}
	return h
}

// prefixPattern inserts prefix between method and path in Go 1.22+ mux patterns.
//
//	"GET /users" + "/api" → "GET /api/users"
//	"/users"     + "/api" → "/api/users"
// func prefixPattern(prefix, pattern string) string {
// 	if prefix == "" {
// 		return pattern
// 	}
// 	if i := strings.IndexByte(pattern, ' '); i >= 0 {
// 		return pattern[:i] + " " + prefix + pattern[i+1:]
// 	}
// 	return prefix + pattern
// }
