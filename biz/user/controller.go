package user

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"restful-boilerplate/biz/user/dto"
)

// listUsersHandler returns all users.
//
//	@Summary      List users
//	@Tags         users
//	@Produce      json
//	@Success      200  {array}   User
//	@Failure      500  {object}  map[string]string
//	@Router       /users [get]
func (ctrl *Controller) listUsersHandler(c *echo.Context) error {
	users, err := ctrl.svc.listUsers(c.Request().Context())
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
//	@Param        body  body      dto.CreateUserRequest  true  "User data"
//	@Success      201   {object}  User
//	@Failure      400   {object}  map[string]string
//	@Failure      422   {object}  map[string]string
//	@Failure      500   {object}  map[string]string
//	@Router       /users [post]
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

// getUserByIDHandler gets a user by ID.
//
//	@Summary      Get user by ID
//	@Tags         users
//	@Produce      json
//	@Param        id   path      string  true  "User ID"
//	@Success      200  {object}  User
//	@Failure      404  {object}  map[string]string
//	@Failure      500  {object}  map[string]string
//	@Router       /users/{id} [get]
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

// updateUserHandler updates a user.
//
//	@Summary      Update user
//	@Tags         users
//	@Accept       json
//	@Produce      json
//	@Param        id    path      string                 true  "User ID"
//	@Param        body  body      dto.UpdateUserRequest  true  "User data"
//	@Success      200   {object}  User
//	@Failure      400   {object}  map[string]string
//	@Failure      404   {object}  map[string]string
//	@Failure      500   {object}  map[string]string
//	@Router       /users/{id} [put]
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
func (ctrl *Controller) deleteUserHandler(c *echo.Context) error {
	if err := ctrl.svc.deleteUser(c.Request().Context(), c.Param("id")); err != nil {
		if errors.Is(err, errNotFound) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}
