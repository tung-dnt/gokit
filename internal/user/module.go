// Package usermodule to register DI-able user module to app
package user

import (
	"restful-boilerplate/internal/app"
	useradapter "restful-boilerplate/internal/user/adapter"
	usercore "restful-boilerplate/internal/user/core"
	router "restful-boilerplate/pkg/http"
)

// Module exposes user endpoints over HTTP.
type Module struct {
	httpAdapter *useradapter.HTTPAdapter
}

// NewModule wires the user service from the shared App container.
// Main never needs to import this package's service constructor directly.
func NewModule(a *app.App) *Module {
	svc := usercore.NewService(a.Queries, a.Tracer.Tracer("user"))
	userHTTPAdapter := useradapter.NewHTTPAdapter(svc, a.Validator)
	return &Module{httpAdapter: userHTTPAdapter}
}

// RegisterRoutes mounts all user endpoints onto g.
func (m *Module) RegisterRoutes(g *router.Group) {
	g.GET("/", m.httpAdapter.ListUsersHandler)
	g.POST("/", m.httpAdapter.CreateUserHandler)
	g.GET("/{id}", m.httpAdapter.GetUserByIDHandler)
	g.PUT("/{id}", m.httpAdapter.UpdateUserHandler)
	g.DELETE("/{id}", m.httpAdapter.DeleteUserHandler)
}
