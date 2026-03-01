// Package user is the HTTP adapter for user management.
package user

import (
	"restful-boilerplate/domain/user"
	router "restful-boilerplate/infra/http"
)

// Validator validates struct fields.
type Validator interface {
	Validate(i any) error
}

// Module exposes user endpoints over HTTP.
type Module struct {
	svc *user.Svc
	val Validator
}

// NewHandler creates a Handler backed by svc with request validation via v.
func NewHandler(svc *user.Svc, v Validator) *Module {
	return &Module{svc: svc, val: v}
}

// RegisterRoutes mounts all user endpoints onto g.
func (m *Module) RegisterRoutes(g *router.Group) {
	g.HandleFunc("GET /", m.listUsersHandler)
	g.HandleFunc("POST /", m.createUserHandler)
	g.HandleFunc("GET /{id}", m.getUserByIDHandler)
	g.HandleFunc("PUT /{id}", m.updateUserHandler)
	g.HandleFunc("DELETE /{id}", m.deleteUserHandler)
}
