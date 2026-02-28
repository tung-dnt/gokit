package router

import (
	"net/http"
)

type Router struct {
	Router  *http.ServeMux
	Handler http.Handler
	prefix  string
}

func NewRouter() *Router {
	mux := http.NewServeMux()
	return &Router{Router: mux, Handler: mux}
}

func (s *Router) Prefix(globalPrefix string) *Router {
	s.prefix = globalPrefix
	return s
}

func (s *Router) Use(mw func(http.Handler) http.Handler) *Router {
	s.Handler = mw(s.Handler)
	return s
}

func (s *Router) Group(prefix string, fn func(*Group)) *Router {
	fn(&Group{mux: s.Router, prefix: s.prefix + prefix})
	return s
}

func (s *Router) Route(pattern string, h http.Handler) *Router {
	s.Router.Handle(prefixPattern(s.prefix, pattern), h)
	return s
}
