// Package useradapter for user controller
package useradapter

import (
	"encoding/json"
	"errors"
	"net/http"

	"restful-boilerplate/internal/app"
	"restful-boilerplate/internal/user/core"
	"restful-boilerplate/internal/user/mapping"
	usermodel "restful-boilerplate/internal/user/model"
	"restful-boilerplate/pkg/http"
)

// HTTPAdapter handles HTTP requests for the user domain.
type HTTPAdapter struct {
	svc *usercore.Service
	val app.Validator
}

// NewHTTPAdapter creates a new HTTPAdapter with the given service and validator.
func NewHTTPAdapter(svc *usercore.Service, val app.Validator) *HTTPAdapter {
	return &HTTPAdapter{svc: svc, val: val}
}

// ListUsersHandler returns all users.
//
//	@Summary      List users
//	@Tags         users
//	@Produce      json
//	@Success      200  {array}   usermapping.UserResponse
//	@Failure      500  {object}  map[string]string
//	@Router       /users [get]
func (m *HTTPAdapter) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := m.svc.ListUsers(r.Context())
	if err != nil {
		router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	resp := make([]usermapping.UserResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, usermapping.ToResponse(*u))
	}
	router.WriteJSON(w, http.StatusOK, resp)
}

// CreateUserHandler creates a new user.
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
func (m *HTTPAdapter) CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req usermodel.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		router.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := m.val.Validate(&req); err != nil {
		router.WriteJSON(w, http.StatusUnprocessableEntity, err)
		return
	}
	u, err := m.svc.CreateUser(r.Context(), req)
	if err != nil {
		router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	router.WriteJSON(w, http.StatusCreated, usermapping.ToResponse(*u))
}

// GetUserByIDHandler gets a user by ID.
//
//	@Summary      Get user by ID
//	@Tags         users
//	@Produce      json
//	@Param        id   path      string  true  "User ID"
//	@Success      200  {object}  usermapping.UserResponse
//	@Failure      404  {object}  map[string]string
//	@Failure      500  {object}  map[string]string
//	@Router       /users/{id} [get]
func (m *HTTPAdapter) GetUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	u, err := m.svc.GetUserByID(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, usercore.ErrNotFound) {
			router.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	router.WriteJSON(w, http.StatusOK, usermapping.ToResponse(*u))
}

// UpdateUserHandler updates a user.
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
func (m *HTTPAdapter) UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req usermodel.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		router.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := m.val.Validate(&req); err != nil {
		router.WriteJSON(w, http.StatusUnprocessableEntity, err)
		return
	}
	u, err := m.svc.UpdateUser(r.Context(), r.PathValue("id"), req)
	if err != nil {
		if errors.Is(err, usercore.ErrNotFound) {
			router.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	router.WriteJSON(w, http.StatusOK, usermapping.ToResponse(*u))
}

// DeleteUserHandler deletes a user.
//
//	@Summary      Delete user
//	@Tags         users
//	@Produce      json
//	@Param        id   path      string  true  "User ID"
//	@Success      204
//	@Failure      404  {object}  map[string]string
//	@Failure      500  {object}  map[string]string
//	@Router       /users/{id} [delete]
func (m *HTTPAdapter) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if err := m.svc.DeleteUser(r.Context(), r.PathValue("id")); err != nil {
		if errors.Is(err, usercore.ErrNotFound) {
			router.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
