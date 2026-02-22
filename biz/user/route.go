// Package user is the self-contained business domain for user management.
package user

import (
	"database/sql"

	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel"
	sqlitedb "restful-boilerplate/repo/sqlite/db"
)

// Controller is the only exported symbol in this package.
type Controller struct {
	svc *userService
}

// NewController creates a new user Controller backed by db.
func NewController(db *sql.DB) *Controller {
	return &Controller{svc: &userService{
		q:      sqlitedb.New(db),
		tracer: otel.Tracer("user"),
	}}
}

// RegisterRoutes mounts all user endpoints onto g.
func (ctrl *Controller) RegisterRoutes(g *echo.Group) {
	g.GET("", ctrl.listUsersHandler)
	g.POST("", ctrl.createUserHandler)
	g.GET("/:id", ctrl.getUserByIDHandler)
	g.PUT("/:id", ctrl.updateUserHandler)
	g.DELETE("/:id", ctrl.deleteUserHandler)
}
