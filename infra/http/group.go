package router

import (
	"net/http"
	"strings"
)

// Group scopes route registration under a shared URL prefix
// with optional group-level middleware.
type Group struct {
	mux    *http.ServeMux
	prefix string
	mws    []func(http.Handler) http.Handler
}

// Use appends middleware that wraps all handlers registered in this group.
// Middleware must be added before registering routes.
func (g *Group) Use(mw func(http.Handler) http.Handler) {
	g.mws = append(g.mws, mw)
}

// Handle registers handler for the given pattern within this group's prefix.
func (g *Group) Handle(pattern string, handler http.Handler) {
	g.mux.Handle(prefixPattern(g.prefix, pattern), g.wrap(handler))
}

// HandleFunc registers a handler function for the given pattern within this group's prefix.
func (g *Group) HandleFunc(pattern string, handler http.HandlerFunc) {
	g.mux.Handle(prefixPattern(g.prefix, pattern), g.wrap(handler))
}

// Route registers a single handler for the given pattern within this group's prefix.
func (g *Group) Route(pattern string, h http.Handler) {
	g.mux.Handle(prefixPattern(g.prefix, pattern), g.wrap(h))
}

// Group creates a nested sub-group with an additional prefix.
// The sub-group inherits this group's middleware chain.
func (g *Group) Group(prefix string, fn func(*Group)) {
	sub := &Group{
		mux:    g.mux,
		prefix: g.prefix + prefix,
		mws:    make([]func(http.Handler) http.Handler, len(g.mws)),
	}
	copy(sub.mws, g.mws)
	fn(sub)
}

// wrap applies the group's middleware chain to a handler.
// First middleware added via Use is the outermost wrapper.
func (g *Group) wrap(h http.Handler) http.Handler {
	for i := len(g.mws) - 1; i >= 0; i-- {
		h = g.mws[i](h)
	}
	return h
}

// prefixPattern inserts prefix between method and path in Go 1.22+ mux patterns.
//
//	"GET /users" + "/api" → "GET /api/users"
//	"/users"     + "/api" → "/api/users"
func prefixPattern(prefix, pattern string) string {
	if prefix == "" {
		return pattern
	}
	if i := strings.IndexByte(pattern, ' '); i >= 0 {
		return pattern[:i] + " " + prefix + pattern[i+1:]
	}
	return prefix + pattern
}
