package router

import (
	"net/http"
	"strings"
)

type Group struct {
	mux    *http.ServeMux
	prefix string
}

func (g *Group) Handle(pattern string, handler http.Handler) {
	g.mux.Handle(prefixPattern(g.prefix, pattern), handler)
}

func (g *Group) HandleFunc(pattern string, handler http.HandlerFunc) {
	g.mux.HandleFunc(prefixPattern(g.prefix, pattern), handler)
}

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
