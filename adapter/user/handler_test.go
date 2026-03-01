package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"

	"restful-boilerplate/domain/user"
	router "restful-boilerplate/infra/http"
	"restful-boilerplate/infra/testutil"
	cv "restful-boilerplate/infra/validator"
)

// newTestHandler sets up an http.Handler with user routes backed by in-memory SQLite.
func newTestHandler(t *testing.T) (http.Handler, *user.Svc) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	repo := NewSQLite(db)
	svc := user.NewService(repo, noop.NewTracerProvider().Tracer("test"))

	srv := router.NewRouter()
	srv.Group("/users", func(g *router.Group) {
		NewHandler(svc, cv.New()).RegisterRoutes(g)
	})
	return srv.Handler, svc
}

// seedUser creates a user via the service and returns it.
func seedUser(t *testing.T, svc *user.Svc, name, email string) *user.User {
	t.Helper()
	u, err := svc.CreateUser(context.Background(), user.CreateUserInput{Name: name, Email: email})
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

// --- POST /users/ ---

func TestCreateUser_HTTP_Success(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)

	body := `{"name":"Alice","email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/users/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var u user.User
	if err := json.Unmarshal(rec.Body.Bytes(), &u); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if u.ID == "" {
		t.Error("expected non-empty ID in response")
	}
	if u.Name != "Alice" {
		t.Errorf("Name = %q, want %q", u.Name, "Alice")
	}
	if u.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", u.Email, "alice@example.com")
	}
}

func TestCreateUser_HTTP_MalformedJSON(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/users/", strings.NewReader(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateUser_HTTP_MissingName(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)

	body := `{"email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/users/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestCreateUser_HTTP_MissingEmail(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)

	body := `{"name":"Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/users/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestCreateUser_HTTP_InvalidEmail(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)

	body := `{"name":"Alice","email":"not-an-email"}`
	req := httptest.NewRequest(http.MethodPost, "/users/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestCreateUser_HTTP_NameTooLong(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)

	body := `{"name":"` + strings.Repeat("a", 101) + `","email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/users/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

// --- GET /users/ ---

func TestListUsers_HTTP_Empty(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/users/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var users []user.User
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestListUsers_HTTP_WithData(t *testing.T) {
	t.Parallel()
	h, svc := newTestHandler(t)

	seedUser(t, svc, "Alice", "alice@example.com")
	seedUser(t, svc, "Bob", "bob@example.com")

	req := httptest.NewRequest(http.MethodGet, "/users/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var users []user.User
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

// --- GET /users/{id} ---

func TestGetUserByID_HTTP_Found(t *testing.T) {
	t.Parallel()
	h, svc := newTestHandler(t)

	u := seedUser(t, svc, "Alice", "alice@example.com")

	req := httptest.NewRequest(http.MethodGet, "/users/"+u.ID, nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got user.User
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ID != u.ID {
		t.Errorf("ID = %q, want %q", got.ID, u.ID)
	}
}

func TestGetUserByID_HTTP_NotFound(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/users/nonexistent", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// --- PUT /users/{id} ---

func TestUpdateUser_HTTP_BothFields(t *testing.T) {
	t.Parallel()
	h, svc := newTestHandler(t)

	u := seedUser(t, svc, "Alice", "alice@example.com")

	body := `{"name":"Bob","email":"bob@example.com"}`
	req := httptest.NewRequest(http.MethodPut, "/users/"+u.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got user.User
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != "Bob" {
		t.Errorf("Name = %q, want %q", got.Name, "Bob")
	}
	if got.Email != "bob@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "bob@example.com")
	}
}

func TestUpdateUser_HTTP_NameOnly(t *testing.T) {
	t.Parallel()
	h, svc := newTestHandler(t)

	u := seedUser(t, svc, "Alice", "alice@example.com")

	body := `{"name":"Bob"}`
	req := httptest.NewRequest(http.MethodPut, "/users/"+u.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got user.User
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != "Bob" {
		t.Errorf("Name = %q, want %q", got.Name, "Bob")
	}
	if got.Email != "alice@example.com" {
		t.Errorf("Email should be preserved, got %q", got.Email)
	}
}

func TestUpdateUser_HTTP_MalformedJSON(t *testing.T) {
	t.Parallel()
	h, svc := newTestHandler(t)

	u := seedUser(t, svc, "Alice", "alice@example.com")

	req := httptest.NewRequest(http.MethodPut, "/users/"+u.ID, strings.NewReader(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateUser_HTTP_InvalidEmail(t *testing.T) {
	t.Parallel()
	h, svc := newTestHandler(t)

	u := seedUser(t, svc, "Alice", "alice@example.com")

	body := `{"email":"not-an-email"}`
	req := httptest.NewRequest(http.MethodPut, "/users/"+u.ID, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d; body: %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}

func TestUpdateUser_HTTP_NotFound(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)

	body := `{"name":"Bob"}`
	req := httptest.NewRequest(http.MethodPut, "/users/nonexistent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// --- DELETE /users/{id} ---

func TestDeleteUser_HTTP_Success(t *testing.T) {
	t.Parallel()
	h, svc := newTestHandler(t)

	u := seedUser(t, svc, "Alice", "alice@example.com")

	req := httptest.NewRequest(http.MethodDelete, "/users/"+u.ID, nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestDeleteUser_HTTP_NotFound(t *testing.T) {
	t.Parallel()
	h, _ := newTestHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/users/nonexistent", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
