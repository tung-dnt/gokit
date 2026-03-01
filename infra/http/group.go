package router

import (
	"net/http"
	"strings"
)

// Group scopes route registration under a shared URL prefix.
type Group struct {
	mux    *http.ServeMux
	prefix string
}

// Handle registers handler for the given pattern within this group's prefix.
func (g *Group) Handle(pattern string, handler http.Handler) {
	g.mux.Handle(prefixPattern(g.prefix, pattern), handler)
}

// HandleFunc registers a handler function for the given pattern within this group's prefix.
func (g *Group) HandleFunc(pattern string, handler http.HandlerFunc) {
	g.mux.HandleFunc(prefixPattern(g.prefix, pattern), handler)
}

// Group creates a nested sub-group with an additional prefix.
func (g *Group) Group(prefix string, fn func(*Group)) {
	fn(&Group{mux: g.mux, prefix: g.prefix + prefix})
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
