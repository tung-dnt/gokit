// Package userhdl is the HTTP adapter for user management.
package userhdl

import (
	"github.com/labstack/echo/v5"

	"restful-boilerplate/app/user"
)

// Handler exposes user endpoints over HTTP.
type Handler struct {
	svc *user.Service
}

// NewHandler creates a Handler backed by svc.
func NewHandler(svc *user.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts all user endpoints onto g.
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("", h.listUsersHandler)
	g.POST("", h.createUserHandler)
	g.GET("/:id", h.getUserByIDHandler)
	g.PUT("/:id", h.updateUserHandler)
	g.DELETE("/:id", h.deleteUserHandler)
}
