package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	pgdb "restful-boilerplate/pkg/postgres/db"
	"restful-boilerplate/pkg/validator"
)

func newTestAdapter(svc userSvc) *httpAdapter {
	return newHTTPAdapter(svc, validator.New())
}

func decodeBody[T any](t *testing.T, r io.Reader) T {
	t.Helper()
	var v T
	require.NoError(t, json.NewDecoder(r).Decode(&v))
	return v
}

func TestCreateUserHandler_Created(t *testing.T) {
	svc := &mockUserSvc{}
	now := time.Now()
	svc.On("createUser", mock.Anything, CreateUserRequest{Name: "Alice", Email: "a@b.c"}).
		Return(&pgdb.User{ID: "id-1", Name: "Alice", Email: "a@b.c", CreatedAt: now}, nil)

	body := bytes.NewBufferString(`{"name":"Alice","email":"a@b.c"}`)
	req := httptest.NewRequest(http.MethodPost, "/users", body)
	rec := httptest.NewRecorder()

	newTestAdapter(svc).createUserHandler(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	got := decodeBody[userResponse](t, rec.Body)
	require.Equal(t, "id-1", got.ID)
	require.Equal(t, "Alice", got.Name)
	svc.AssertExpectations(t)
}

func TestCreateUserHandler_MalformedJSON(t *testing.T) {
	svc := &mockUserSvc{}
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()

	newTestAdapter(svc).createUserHandler(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	svc.AssertNotCalled(t, "createUser")
}

func TestCreateUserHandler_ValidationError(t *testing.T) {
	svc := &mockUserSvc{}
	// Missing required name, bad email
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"","email":"not-email"}`))
	rec := httptest.NewRecorder()

	newTestAdapter(svc).createUserHandler(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	svc.AssertNotCalled(t, "createUser")
}

func TestCreateUserHandler_ServiceError(t *testing.T) {
	svc := &mockUserSvc{}
	svc.On("createUser", mock.Anything, mock.Anything).Return((*pgdb.User)(nil), errors.New("boom"))

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"name":"A","email":"a@b.c"}`))
	rec := httptest.NewRecorder()

	newTestAdapter(svc).createUserHandler(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetUserByIDHandler(t *testing.T) {
	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{"ok", nil, http.StatusOK},
		{"not found", ErrNotFound, http.StatusNotFound},
		{"internal", errors.New("db down"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockUserSvc{}
			if tt.svcErr == nil {
				svc.On("getUserByID", mock.Anything, "id-1").Return(&pgdb.User{ID: "id-1"}, nil)
			} else {
				svc.On("getUserByID", mock.Anything, "id-1").Return((*pgdb.User)(nil), tt.svcErr)
			}

			req := httptest.NewRequest(http.MethodGet, "/users/id-1", nil)
			req.SetPathValue("id", "id-1")
			rec := httptest.NewRecorder()

			newTestAdapter(svc).getUserByIDHandler(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestListUsersHandler(t *testing.T) {
	svc := &mockUserSvc{}
	svc.On("listUsers", mock.Anything).Return(
		[]*pgdb.User{{ID: "1", Name: "A"}, {ID: "2", Name: "B"}}, nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()

	newTestAdapter(svc).listUsersHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	got := decodeBody[[]userResponse](t, rec.Body)
	require.Len(t, got, 2)
	require.Equal(t, "1", got[0].ID)
}

func TestUpdateUserHandler_NotFound(t *testing.T) {
	svc := &mockUserSvc{}
	svc.On("updateUser", mock.Anything, "id-1", mock.Anything).Return((*pgdb.User)(nil), ErrNotFound)

	req := httptest.NewRequest(http.MethodPut, "/users/id-1", strings.NewReader(`{"name":"N"}`))
	req.SetPathValue("id", "id-1")
	rec := httptest.NewRecorder()

	newTestAdapter(svc).updateUserHandler(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteUserHandler(t *testing.T) {
	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{"deleted", nil, http.StatusNoContent},
		{"not found", ErrNotFound, http.StatusNotFound},
		{"internal", errors.New("db"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockUserSvc{}
			svc.On("deleteUser", mock.Anything, "id-1").Return(tt.svcErr)

			req := httptest.NewRequest(http.MethodDelete, "/users/id-1", nil)
			req.SetPathValue("id", "id-1")
			rec := httptest.NewRecorder()

			newTestAdapter(svc).deleteUserHandler(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// Ensure context-based logger lookup does not panic when ctx has no logger.
func TestWriteErr_NoLoggerInContext_DoesNotPanic(t *testing.T) {
	svc := &mockUserSvc{}
	a := newTestAdapter(svc)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(context.Background())
	a.writeErr(req, rec, errors.New("x"))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
