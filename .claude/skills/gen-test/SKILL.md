---
name: gen-test
description: Generate Go unit and integration tests for handlers, service, or DTOs using table-driven tests, net/http httptest, and in-memory SQLite
---

Generate idiomatic Go tests for the Clean Architecture layers. Uses stdlib only. No external test libraries (no testify, etc.).

## Step 0 — Invoke test-master first

Before writing any test code, use the Skill tool to invoke `fullstack-dev-skills:test-master`. Provide it:
- Language: Go (stdlib `testing` only — no testify, no mocks)
- The source file(s) being tested and what they do
- Project constraints: in-memory SQLite, net/http httptest, table-driven tests, no external test libraries

Let test-master apply its full skill set — unit testing strategy, integration testing, TDD iron laws, testing anti-patterns, QA methodology, automation frameworks, security testing, and test reporting. Use its complete output to build a test plan before writing code. The project-specific patterns below then govern the exact implementation.

---

## Rules

- Service tests live in `domain/<domain>/service_test.go` (external test package `package <domain>_test` to avoid import cycles).
- Handler tests live in `adapter/<domain>/handler_test.go` (same package).
- DTO tests live in `adapter/<domain>/dto_test.go` (same package).
- Use **table-driven tests**: `tests := []struct{ name string; ... }{ {...}, ... }` with `t.Run(tt.name, ...)`.
- Use `t.Fatal` / `t.Errorf` — never `panic`.
- Test both happy path and error cases.
- For service tests, use the shared `infra/testutil/testdb.go` helper with in-memory SQLite.

---

## File argument

When invoked with a file path, generate the corresponding test file:

| Source file | Test file | What to test |
|-------------|-----------|--------------|
| `adapter/<domain>/dto.go` | `adapter/<domain>/dto_test.go` | Validator tag behaviour |
| `domain/<domain>/service.go` | `domain/<domain>/service_test.go` | CRUD ops with in-memory SQLite |
| `adapter/<domain>/handler.go` | `adapter/<domain>/handler_test.go` | HTTP status codes via httptest |

---

## Test helper — service setup

```go
package user_test

import (
    "testing"

    "go.opentelemetry.io/otel/trace/noop"

    useradapter "restful-boilerplate/adapter/user"
    "restful-boilerplate/domain/user"
    "restful-boilerplate/infra/testutil"
)

func newTestService(t *testing.T) *user.UserSvc {
    t.Helper()
    db := testutil.SetupTestDB(t)
    repo := useradapter.NewSQLite(db)
    return user.NewService(repo, noop.NewTracerProvider().Tracer("test"))
}
```

---

## DTO test pattern (`dto_test.go`)

Test that go-playground/validator enforces the struct tags correctly:

```go
package user

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
package user_test

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

Use `net/http/httptest` + router to test the full handler pipeline:

```go
package user

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "go.opentelemetry.io/otel/trace/noop"

    domainuser "restful-boilerplate/domain/user"
    router "restful-boilerplate/infra/http"
    "restful-boilerplate/infra/testutil"
    cv "restful-boilerplate/infra/validator"
)

func newTestHandler(t *testing.T) (http.Handler, *domainuser.UserSvc) {
    t.Helper()
    db := testutil.SetupTestDB(t)
    repo := NewSQLite(db)
    svc := domainuser.NewService(repo, noop.NewTracerProvider().Tracer("test"))

    srv := router.NewRouter()
    srv.Group("/users", func(g *router.Group) {
        NewModule(svc, cv.New()).RegisterRoutes(g)
    })
    return srv.Handler, svc
}

func TestCreateUser_HTTP_Success(t *testing.T) {
    h, _ := newTestHandler(t)
    body := `{"name":"Alice","email":"alice@example.com"}`
    req := httptest.NewRequest(http.MethodPost, "/users/", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    h.ServeHTTP(rec, req)
    if rec.Code != http.StatusCreated {
        t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
    }
}
```

---

## Run tests

```bash
go test ./domain/<domain>/...
go test ./adapter/<domain>/...
go test ./... -count=1   # all packages, no cache
```
