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

// Handler exposes user endpoints over HTTP.
type Handler struct {
	svc *user.UserSvc
	val Validator
}

// NewHandler creates a Handler backed by svc with request validation via v.
func NewHandler(svc *user.UserSvc, v Validator) *Handler {
	return &Handler{svc: svc, val: v}
}

// RegisterRoutes mounts all user endpoints onto g.
func (h *Handler) RegisterRoutes(g *router.Group) {
	g.HandleFunc("GET /", h.listUsersHandler)
	g.HandleFunc("POST /", h.createUserHandler)
	g.HandleFunc("GET /{id}", h.getUserByIDHandler)
	g.HandleFunc("PUT /{id}", h.updateUserHandler)
	g.HandleFunc("DELETE /{id}", h.deleteUserHandler)
}
