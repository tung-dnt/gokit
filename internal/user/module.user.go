// Package usermodule to register DI-able user module to app
package user

import (
	"restful-boilerplate/internal/app"
	router "restful-boilerplate/pkg/http"
)

// Module exposes user endpoints over HTTP.
type Module struct {
	httpAdapter *httpAdapter
}

// NewModule wires the user service from the shared App container.
// Main never needs to import this package's service constructor directly.
func NewModule(a *app.App) *Module {
	svc := newUserService(a.Queries, a.Tracer.Tracer("user"))
	userHTTPAdapter := newHTTPAdapter(svc, a.Validator)

	return &Module{httpAdapter: userHTTPAdapter}
}

// RegisterRoutes mounts all user endpoints onto g.
func (m *Module) RegisterRoutes(g *router.Group) {
	g.GET("/", m.httpAdapter.listUsersHandler)
	g.POST("/", m.httpAdapter.createUserHandler)
	g.GET("/{id}", m.httpAdapter.getUserByIDHandler)
	g.PUT("/{id}", m.httpAdapter.updateUserHandler)
	g.DELETE("/{id}", m.httpAdapter.deleteUserHandler)
}
