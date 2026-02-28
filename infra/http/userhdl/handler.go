package userhdl

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	"restful-boilerplate/domain/user"
)

// listUsersHandler returns all users.
//
//	@Summary      List users
//	@Tags         users
//	@Produce      json
//	@Success      200  {array}   user.User
//	@Failure      500  {object}  map[string]string
//	@Router       /users [get]
func (h *Handler) listUsersHandler(c *echo.Context) error {
	users, err := h.svc.ListUsers(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, users)
}

// createUserHandler creates a new user.
//
//	@Summary      Create user
//	@Tags         users
//	@Accept       json
//	@Produce      json
//	@Param        body  body      CreateUserRequest  true  "User data"
//	@Success      201   {object}  user.User
//	@Failure      400   {object}  map[string]string
//	@Failure      422   {object}  map[string]string
//	@Failure      500   {object}  map[string]string
//	@Router       /users [post]
func (h *Handler) createUserHandler(c *echo.Context) error {
	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	u, err := h.svc.CreateUser(c.Request().Context(), user.CreateUserInput{Name: req.Name, Email: req.Email})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, u)
}

// getUserByIDHandler gets a user by ID.
//
//	@Summary      Get user by ID
//	@Tags         users
//	@Produce      json
//	@Param        id   path      string  true  "User ID"
//	@Success      200  {object}  user.User
//	@Failure      404  {object}  map[string]string
//	@Failure      500  {object}  map[string]string
//	@Router       /users/{id} [get]
func (h *Handler) getUserByIDHandler(c *echo.Context) error {
	u, err := h.svc.GetUserByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, u)
}

// updateUserHandler updates a user.
//
//	@Summary      Update user
//	@Tags         users
//	@Accept       json
//	@Produce      json
//	@Param        id    path      string             true  "User ID"
//	@Param        body  body      UpdateUserRequest  true  "User data"
//	@Success      200   {object}  user.User
//	@Failure      400   {object}  map[string]string
//	@Failure      404   {object}  map[string]string
//	@Failure      500   {object}  map[string]string
//	@Router       /users/{id} [put]
func (h *Handler) updateUserHandler(c *echo.Context) error {
	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	u, err := h.svc.UpdateUser(c.Request().Context(), c.Param("id"), user.UpdateUserInput{Name: req.Name, Email: req.Email})
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, u)
}

// deleteUserHandler deletes a user.
//
//	@Summary      Delete user
//	@Tags         users
//	@Produce      json
//	@Param        id   path      string  true  "User ID"
//	@Success      204
//	@Failure      404  {object}  map[string]string
//	@Failure      500  {object}  map[string]string
//	@Router       /users/{id} [delete]
func (h *Handler) deleteUserHandler(c *echo.Context) error {
	if err := h.svc.DeleteUser(c.Request().Context(), c.Param("id")); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}
