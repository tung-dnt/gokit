package user

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"restful-boilerplate/biz/user/dto"
)

func (ctrl *Controller) listUsersHandler(c *echo.Context) error {
	users, err := ctrl.svc.listUsers(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, users)
}

func (ctrl *Controller) createUserHandler(c *echo.Context) error {
	var req dto.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	u, err := ctrl.svc.createUser(c.Request().Context(), createUserInput{Name: req.Name, Email: req.Email})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, u)
}

func (ctrl *Controller) getUserByIDHandler(c *echo.Context) error {
	u, err := ctrl.svc.getUserByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, errNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, u)
}

func (ctrl *Controller) updateUserHandler(c *echo.Context) error {
	var req dto.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	u, err := ctrl.svc.updateUser(c.Request().Context(), c.Param("id"), updateUserInput{Name: req.Name, Email: req.Email})
	if err != nil {
		if errors.Is(err, errNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, u)
}

func (ctrl *Controller) deleteUserHandler(c *echo.Context) error {
	if err := ctrl.svc.deleteUser(c.Request().Context(), c.Param("id")); err != nil {
		if errors.Is(err, errNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}
