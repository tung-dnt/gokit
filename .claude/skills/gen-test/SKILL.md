---
name: gen-test
description: Generate Go unit and integration tests for handlers, service, or DTOs using table-driven tests, Echo v5 test utilities, and in-memory SQLite
---

Generate idiomatic Go tests for the Clean Architecture layers. Uses stdlib + Echo test utilities only. No external test libraries (no testify, etc.).

## Step 0 — Invoke test-master first

Before writing any test code, use the Skill tool to invoke `fullstack-dev-skills:test-master`. Provide it:
- Language: Go (stdlib `testing` only — no testify, no mocks)
- The source file(s) being tested and what they do
- Project constraints: in-memory SQLite, Echo v5 httptest, table-driven tests, no external test libraries

Let test-master apply its full skill set — unit testing strategy, integration testing, TDD iron laws, testing anti-patterns, QA methodology, automation frameworks, security testing, and test reporting. Use its complete output to build a test plan before writing code. The project-specific patterns below then govern the exact implementation.

---

## Rules

- Service tests live in `app/<domain>svc/service_test.go` (same package to access unexported helpers like `generateID`).
- Handler tests live in `infra/http/<domain>hdl/handler_test.go` (same package).
- DTO tests live in `infra/http/<domain>hdl/dto_test.go` (same package).
- Use **table-driven tests**: `tests := []struct{ name string; ... }{ {...}, ... }` with `t.Run(tt.name, ...)`.
- Use `t.Fatal` / `t.Errorf` — never `panic`.
- Test both happy path and error cases.
- For service tests, use the shared `infra/testutil/testdb.go` helper with in-memory SQLite.

---

## File argument

When invoked with a file path, generate the corresponding test file:

| Source file | Test file | What to test |
|-------------|-----------|--------------|
| `infra/http/<domain>hdl/dto.go` | `infra/http/<domain>hdl/dto_test.go` | Validator tag behaviour |
| `app/<domain>svc/service.go` | `app/<domain>svc/service_test.go` | CRUD ops with in-memory SQLite |
| `infra/http/<domain>hdl/handler.go` | `infra/http/<domain>hdl/handler_test.go` | HTTP status codes via Echo |

---

## Test helper — service setup

```go
package usersvc

import (
    "testing"

    "go.opentelemetry.io/otel/trace/noop"

    testutil "restful-boilerplate/infra/testutil"
    "restful-boilerplate/infra/sqlite/userrepo"
)

func newTestService(t *testing.T) *Service {
    t.Helper()
    db := testutil.SetupTestDB(t)
    repo := userrepo.NewSQLite(db)
    return NewService(repo, noop.NewTracerProvider().Tracer("test"))
}
```

---

## DTO test pattern (`dto_test.go`)

Test that go-playground/validator enforces the struct tags correctly:

```go
package userhdl

import (
    "testing"

    "github.com/go-playground/validator/v10"
)

func TestCreateUserRequest(t *testing.T) {
    v := validator.New()
    tests := []struct {
        name    string
        input   CreateUserRequest
        wantErr bool
    }{
        {name: "valid", input: CreateUserRequest{Name: "Alice", Email: "alice@example.com"}},
        {name: "missing name", input: CreateUserRequest{Email: "alice@example.com"}, wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := v.Struct(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
            }
        })
    }
}
```

---

## Service test pattern (`service_test.go`)

Uses a real in-memory SQLite DB through the Repository interface:

```go
package usersvc

import (
    "context"
    "errors"
    "testing"

    "restful-boilerplate/domain/user"
)

func TestCreateUser(t *testing.T) {
    svc := newTestService(t)
    ctx := context.Background()

    u, err := svc.CreateUser(ctx, user.CreateUserInput{Name: "Alice", Email: "alice@example.com"})
    if err != nil {
        t.Fatalf("CreateUser() error = %v", err)
    }
    if u.ID == "" {
        t.Error("expected non-empty ID")
    }
}

func TestGetUserByID_NotFound(t *testing.T) {
    svc := newTestService(t)
    _, err := svc.GetUserByID(context.Background(), "nonexistent")
    if !errors.Is(err, user.ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}
```

---

## Handler test pattern (`handler_test.go`)

Use `net/http/httptest` + Echo to test the full handler pipeline:

```go
package userhdl

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/labstack/echo/v5"
    "go.opentelemetry.io/otel/trace/noop"

    "restful-boilerplate/app/usersvc"
    testutil "restful-boilerplate/infra/testutil"
    "restful-boilerplate/infra/sqlite/userrepo"
    cv "restful-boilerplate/infra/validator"
)

func newTestEcho(t *testing.T) (*echo.Echo, *usersvc.Service) {
    t.Helper()
    db := testutil.SetupTestDB(t)
    repo := userrepo.NewSQLite(db)
    svc := usersvc.NewService(repo, noop.NewTracerProvider().Tracer("test"))
    e := echo.New()
    e.Validator = cv.New()
    NewHandler(svc).RegisterRoutes(e.Group("/users"))
    return e, svc
}

func TestCreateUser_HTTP_Success(t *testing.T) {
    e, _ := newTestEcho(t)
    body := `{"name":"Alice","email":"alice@example.com"}`
    req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
    req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
    rec := httptest.NewRecorder()
    e.ServeHTTP(rec, req)
    if rec.Code != http.StatusCreated {
        t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
    }
}
```

---

## Run tests

```bash
go test ./app/<domain>svc/...
go test ./infra/http/<domain>hdl/...
go test ./... -count=1   # all packages, no cache
```
