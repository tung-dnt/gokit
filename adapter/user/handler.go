package user

import (
	"encoding/json"
	"errors"
	"net/http"

	"restful-boilerplate/domain/user"
	rt "restful-boilerplate/infra/http"
)

// listUsersHandler returns all users.
//
//	@Summary      List users
//	@Tags         users
//	@Produce      json
//	@Success      200  {array}   user.User
//	@Failure      500  {object}  map[string]string
//	@Router       /users [get]
func (m *Module) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := m.svc.ListUsers(r.Context())
	if err != nil {
		rt.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	rt.WriteJSON(w, http.StatusOK, users)
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
func (m *Module) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := m.val.Validate(&req); err != nil {
		rt.WriteJSON(w, http.StatusUnprocessableEntity, err)
		return
	}
	u, err := m.svc.CreateUser(r.Context(), user.CreateUserInput{Name: req.Name, Email: req.Email})
	if err != nil {
		rt.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	rt.WriteJSON(w, http.StatusCreated, u)
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
func (m *Module) getUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	u, err := m.svc.GetUserByID(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			rt.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		rt.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	rt.WriteJSON(w, http.StatusOK, u)
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
func (m *Module) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rt.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := m.val.Validate(&req); err != nil {
		rt.WriteJSON(w, http.StatusUnprocessableEntity, err)
		return
	}
	u, err := m.svc.UpdateUser(
		r.Context(),
		r.PathValue("id"),
		user.UpdateUserInput{Name: req.Name, Email: req.Email},
	)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			rt.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		rt.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	rt.WriteJSON(w, http.StatusOK, u)
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
func (m *Module) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if err := m.svc.DeleteUser(r.Context(), r.PathValue("id")); err != nil {
		if errors.Is(err, user.ErrNotFound) {
			rt.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		rt.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
