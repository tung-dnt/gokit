// Package user for user controller
package user

import (
	"errors"
	"net/http"

	"restful-boilerplate/internal/app"
	router "restful-boilerplate/pkg/http"
)

// httpAdapter handles HTTP requests for the user domain.
type httpAdapter struct {
	svc *userService
	val app.Validator
}

// newHTTPAdapter creates a new HTTPAdapter with the given service and validator.
func newHTTPAdapter(svc *userService, val app.Validator) *httpAdapter {
	return &httpAdapter{svc: svc, val: val}
}

// writeErr writes a 404 for ErrNotFound, otherwise 500.
func (m *httpAdapter) writeErr(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		router.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

// listUsersHandler returns all users.
//
//	@Summary      List users
//	@Tags         users
//	@Produce      json
//	@Success      200  {array}   usermapping.UserResponse
//	@Failure      500  {object}  map[string]string
//	@Router       /users [get]
func (m *httpAdapter) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := m.svc.listUsers(r.Context())
	if err != nil {
		m.writeErr(w, err)
		return
	}
	resp := make([]userResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, ToResponse(*u))
	}
	router.WriteJSON(w, http.StatusOK, resp)
}

// createUserHandler creates a new user.
//
//	@Summary      Create user
//	@Tags         users
//	@Accept       json
//	@Produce      json
//	@Param        body  body      usercore.CreateUserInput  true  "User data"
//	@Success      201   {object}  usermapping.UserResponse
//	@Failure      400   {object}  map[string]string
//	@Failure      422   {object}  map[string]string
//	@Failure      500   {object}  map[string]string
//	@Router       /users [post]
func (m *httpAdapter) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if !router.Bind(m.val, w, r, &req) {
		return
	}
	u, err := m.svc.createUser(r.Context(), req)
	if err != nil {
		m.writeErr(w, err)
		return
	}
	router.WriteJSON(w, http.StatusCreated, ToResponse(*u))
}

// getUserByIDHandler gets a user by ID.
//
//	@Summary      Get user by ID
//	@Tags         users
//	@Produce      json
//	@Param        id   path      string  true  "User ID"
//	@Success      200  {object}  usermapping.UserResponse
//	@Failure      404  {object}  map[string]string
//	@Failure      500  {object}  map[string]string
//	@Router       /users/{id} [get]
func (m *httpAdapter) getUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	u, err := m.svc.getUserByID(r.Context(), r.PathValue("id"))
	if err != nil {
		m.writeErr(w, err)
		return
	}
	router.WriteJSON(w, http.StatusOK, ToResponse(*u))
}

// updateUserHandler updates a user.
//
//	@Summary      Update user
//	@Tags         users
//	@Accept       json
//	@Produce      json
//	@Param        id    path      string                   true  "User ID"
//	@Param        body  body      usercore.UpdateUserInput true  "User data"
//	@Success      200   {object}  usermapping.UserResponse
//	@Failure      400   {object}  map[string]string
//	@Failure      404   {object}  map[string]string
//	@Failure      500   {object}  map[string]string
//	@Router       /users/{id} [put]
func (m *httpAdapter) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req UpdateUserRequest
	if !router.Bind(m.val, w, r, &req) {
		return
	}
	u, err := m.svc.updateUser(r.Context(), r.PathValue("id"), req)
	if err != nil {
		m.writeErr(w, err)
		return
	}
	router.WriteJSON(w, http.StatusOK, ToResponse(*u))
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
func (m *httpAdapter) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if err := m.svc.deleteUser(r.Context(), r.PathValue("id")); err != nil {
		m.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
