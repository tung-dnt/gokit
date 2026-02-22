---
name: gen-test
description: Generate Go unit and integration tests for a domain's handlers, service, or DTOs using table-driven tests, Echo v5 test utilities, and in-memory SQLite
---

Generate idiomatic Go tests for files within `biz/<domain>/`. Uses stdlib + Echo test utilities only. No external test libraries (no testify, etc.).

## Step 0 — Invoke test-master first

Before writing any test code, use the Skill tool to invoke `fullstack-dev-skills:test-master`. Provide it:
- Language: Go (stdlib `testing` only — no testify, no mocks)
- The source file(s) being tested and what they do
- Project constraints: in-memory SQLite, Echo v5 httptest, table-driven tests, no external test libraries

Let test-master apply its full skill set — unit testing strategy, integration testing, TDD iron laws, testing anti-patterns, QA methodology, automation frameworks, security testing, and test reporting. Use its complete output to build a test plan before writing code. The project-specific patterns below then govern the exact implementation.

---

## Rules

- Tests live in the **same package** as the code under test (e.g., `package user`) to access unexported types.
- Use **table-driven tests**: `tests := []struct{ name string; ... }{ {...}, ... }` with `t.Run(tt.name, ...)`.
- Use `t.Fatal` / `t.Errorf` — never `panic`.
- Test both happy path and error cases.
- Name test functions `Test<Type>_<method>` (e.g., `TestController_createUser`).
- For service tests, use an in-memory SQLite database (`file::memory:?cache=shared&mode=memory`).

---

## File argument

When invoked with a file path (e.g., `/gen-test biz/user/service.go`), generate the corresponding test file:

| Source file | Test file | What to test |
|-------------|-----------|--------------|
| `dto/dto.go` | `dto/dto_test.go` | Validator tag behaviour |
| `service.go` | `service_test.go` | CRUD ops with in-memory SQLite |
| `controller.go` | `controller_test.go` | HTTP status codes via Echo |

---

## Test helper — in-memory SQLite

Put this in a `testmain_test.go` (or inline in `service_test.go`) to create a real DB for every test:

```go
package user

import (
    "database/sql"
    "testing"

    _ "modernc.org/sqlite"
    sqlitedb "restful-boilerplate/repo/sqlite/db"
    sqlitemig "restful-boilerplate/repo/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
    t.Helper()
    db, err := sql.Open("sqlite", "file::memory:?cache=shared&mode=memory")
    if err != nil {
        t.Fatalf("open db: %v", err)
    }
    if err := sqlitemig.Migrate(db); err != nil {
        t.Fatalf("migrate: %v", err)
    }
    t.Cleanup(func() { db.Close() })
    return db
}

func newTestService(t *testing.T) *userService {
    t.Helper()
    db := setupTestDB(t)
    return &userService{q: sqlitedb.New(db)}
}
```

---

## DTO test pattern (`dto/dto_test.go`)

Test that go-playground/validator enforces the struct tags correctly.
Use a real `validator.Validate` instance (same as `pkg/validator`):

```go
package dto

import (
    "testing"

    "github.com/go-playground/validator/v10"
)

func TestCreateUserRequest_Validate(t *testing.T) {
    v := validator.New()
    tests := []struct {
        name    string
        input   CreateUserRequest
        wantErr bool
        errField string
    }{
        {name: "valid", input: CreateUserRequest{Name: "Alice", Email: "alice@example.com"}, wantErr: false},
        {name: "missing name", input: CreateUserRequest{Email: "alice@example.com"}, wantErr: true, errField: "Name"},
        {name: "missing email", input: CreateUserRequest{Name: "Alice"}, wantErr: true, errField: "Email"},
        {name: "invalid email", input: CreateUserRequest{Name: "Alice", Email: "not-an-email"}, wantErr: true, errField: "Email"},
        {name: "name too long", input: CreateUserRequest{Name: string(make([]byte, 101)), Email: "a@b.com"}, wantErr: true, errField: "Name"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := v.Struct(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
            }
            if tt.wantErr && tt.errField != "" {
                var ve validator.ValidationErrors
                if ok := errors.As(err, &ve); !ok {
                    t.Fatalf("expected ValidationErrors, got %T", err)
                }
                found := false
                for _, fe := range ve {
                    if fe.Field() == tt.errField {
                        found = true
                        break
                    }
                }
                if !found {
                    t.Errorf("expected error on field %q, got: %v", tt.errField, ve)
                }
            }
        })
    }
}
```

---

## Service test pattern (`service_test.go`)

Uses a real in-memory SQLite DB — no mocking needed since sqlc generates a concrete `*Queries`:

```go
package user

import (
    "context"
    "errors"
    "testing"
)

func TestUserService_createUser(t *testing.T) {
    tests := []struct {
        name    string
        input   createUserInput
        wantErr bool
    }{
        {name: "success", input: createUserInput{Name: "Alice", Email: "alice@example.com"}},
        {name: "duplicate email", input: createUserInput{Name: "Alice", Email: "alice@example.com"}, wantErr: true},
    }

    svc := newTestService(t)
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := svc.createUser(context.Background(), tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
            }
        })
    }
}

func TestUserService_getUserByID(t *testing.T) {
    svc := newTestService(t)
    ctx := context.Background()

    // seed
    created, err := svc.createUser(ctx, createUserInput{Name: "Alice", Email: "alice@example.com"})
    if err != nil {
        t.Fatalf("seed: %v", err)
    }

    tests := []struct {
        name    string
        id      string
        wantErr bool
        errIs   error
    }{
        {name: "found", id: created.ID},
        {name: "not found", id: "nonexistent", wantErr: true, errIs: errNotFound},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            u, err := svc.getUserByID(ctx, tt.id)
            if (err != nil) != tt.wantErr {
                t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
            }
            if tt.errIs != nil && !errors.Is(err, tt.errIs) {
                t.Errorf("errors.Is: got %v, want %v", err, tt.errIs)
            }
            if err == nil && u.ID != tt.id {
                t.Errorf("got id=%q, want %q", u.ID, tt.id)
            }
        })
    }
}

func TestUserService_deleteUser(t *testing.T) {
    svc := newTestService(t)
    ctx := context.Background()

    created, _ := svc.createUser(ctx, createUserInput{Name: "Bob", Email: "bob@example.com"})

    tests := []struct {
        name    string
        id      string
        wantErr bool
        errIs   error
    }{
        {name: "success", id: created.ID},
        {name: "not found", id: "nonexistent", wantErr: true, errIs: errNotFound},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := svc.deleteUser(ctx, tt.id)
            if (err != nil) != tt.wantErr {
                t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
            }
            if tt.errIs != nil && !errors.Is(err, tt.errIs) {
                t.Errorf("errors.Is: got %v, want %v", err, tt.errIs)
            }
        })
    }
}
```

---

## Handler test pattern (`controller_test.go`)

Use `net/http/httptest` + Echo to test the full handler pipeline (bind → validate → service → response):

```go
package user

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/labstack/echo/v5"
    cv "restful-boilerplate/pkg/validator"
)

func newTestEcho(t *testing.T) (*echo.Echo, *Controller) {
    t.Helper()
    e := echo.New()
    e.Validator = cv.New()
    db := setupTestDB(t)
    ctrl := NewController(db)
    ctrl.RegisterRoutes(e.Group("/users"))
    return e, ctrl
}

func TestController_createUser(t *testing.T) {
    tests := []struct {
        name       string
        body       string
        wantStatus int
    }{
        {name: "valid", body: `{"name":"Alice","email":"alice@example.com"}`, wantStatus: http.StatusCreated},
        {name: "missing name", body: `{"email":"alice@example.com"}`, wantStatus: http.StatusUnprocessableEntity},
        {name: "missing email", body: `{"name":"Alice"}`, wantStatus: http.StatusUnprocessableEntity},
        {name: "invalid email", body: `{"name":"Alice","email":"bad"}`, wantStatus: http.StatusUnprocessableEntity},
        {name: "malformed json", body: `{bad}`, wantStatus: http.StatusBadRequest},
    }

    e, _ := newTestEcho(t)
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(tt.body))
            req.Header.Set("Content-Type", "application/json")
            w := httptest.NewRecorder()
            e.ServeHTTP(w, req)

            if w.Code != tt.wantStatus {
                t.Errorf("got %d, want %d — body: %s", w.Code, tt.wantStatus, w.Body.String())
            }
        })
    }
}

func TestController_getUserByID(t *testing.T) {
    e, ctrl := newTestEcho(t)
    ctx := context.Background()

    // seed via service
    created, _ := ctrl.svc.createUser(ctx, createUserInput{Name: "Alice", Email: "alice@example.com"})

    tests := []struct {
        name       string
        id         string
        wantStatus int
    }{
        {name: "found", id: created.ID, wantStatus: http.StatusOK},
        {name: "not found", id: "nonexistent", wantStatus: http.StatusNotFound},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(http.MethodGet, "/users/"+tt.id, nil)
            w := httptest.NewRecorder()
            e.ServeHTTP(w, req)
            if w.Code != tt.wantStatus {
                t.Errorf("got %d, want %d — body: %s", w.Code, tt.wantStatus, w.Body.String())
            }
        })
    }
}
```

---

## Run tests

```bash
go test ./biz/<domain>/...
go test ./biz/<domain>/... -v -run TestController_
go test ./... -count=1   # all domains, no cache
```
