---
name: gen-test
description: Generate Go unit and integration tests for handlers, service, or DTOs using table-driven tests, net/http httptest, and in-memory SQLite
---

Generate idiomatic Go tests for the 3-folder structure. Uses stdlib only. No external test libraries (no testify, etc.).

## Step 0 — Invoke test-master first

Before writing any test code, use the Skill tool to invoke `fullstack-dev-skills:test-master`. Provide it:
- Language: Go (stdlib `testing` only — no testify, no mocks)
- The source file(s) being tested and what they do
- Project constraints: in-memory SQLite, net/http httptest, table-driven tests, no external test libraries

Let test-master apply its full skill set — unit testing strategy, integration testing, TDD iron laws, testing anti-patterns, QA methodology, automation frameworks, security testing, and test reporting. Use its complete output to build a test plan before writing code. The project-specific patterns below then govern the exact implementation.

---

## Rules

- Service tests live in `internal/<domain>/core/service_test.go` (external test package `package <domain>core_test`).
- DTO/input tests live in `internal/<domain>/model/dto_test.go` (internal test package `package <domain>model`).
- Handler/module tests live in `internal/<domain>/module.test.go` (internal test package `package <domain>`).
- Use **table-driven tests**: `tests := []struct{ name string; ... }{ {...}, ... }` with `t.Run(tt.name, ...)`.
- Use `t.Fatal` / `t.Errorf` — never `panic`.
- Test both happy path and error cases.
- For service tests, use the shared `pkg/testutil/testdb.go` helper with in-memory SQLite.

---

## File argument

When invoked with a file path, generate the corresponding test file:

| Source file | Test file | What to test |
|-------------|-----------|--------------|
| `internal/<domain>/model/dto.go` | `internal/<domain>/model/dto_test.go` | Validator tag behaviour |
| `internal/<domain>/core/service.go` | `internal/<domain>/core/service_test.go` | CRUD ops with in-memory SQLite |
| `internal/<domain>/adapter/http.go` | `internal/<domain>/module.test.go` | HTTP status codes via httptest |

---

## Test helper — service setup (`core/service_test.go`)

Service tests are in the **external** test package (`package <domain>core_test`). They import `<domain>core` directly:

```go
package usercore_test

import (
    "testing"

    "go.opentelemetry.io/otel/trace/noop"

    usercore "restful-boilerplate/internal/user/core"
    sqlitedb "restful-boilerplate/pkg/sqlite/db"
    "restful-boilerplate/pkg/testutil"
)

func newTestService(t *testing.T) *usercore.Service {
    t.Helper()
    db := testutil.SetupTestDB(t)
    q := sqlitedb.New(db)
    return usercore.NewService(q, noop.NewTracerProvider().Tracer("test"))
}

func TestCreateUser(t *testing.T) {
    t.Parallel()
    svc := newTestService(t)
    ctx := context.Background()

    u, err := svc.CreateUser(ctx, usercore.CreateUserInput{Name: "Alice", Email: "alice@example.com"})
    if err != nil {
        t.Fatalf("CreateUser() error = %v", err)
    }
    if u.ID == "" {
        t.Error("expected non-empty ID")
    }
}

func TestGetUserByID_NotFound(t *testing.T) {
    t.Parallel()
    svc := newTestService(t)
    _, err := svc.GetUserByID(context.Background(), "nonexistent")
    if !errors.Is(err, usercore.ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}
```

---

## DTO test pattern (`model/dto_test.go`)

DTOs live in `internal/<domain>/model/dto.go`. Tests use the **internal** `<domain>model` package:

```go
package usermodel

import (
    "testing"

    "github.com/go-playground/validator/v10"
)

func TestCreateUserRequest(t *testing.T) {
    t.Parallel()
    v := validator.New()
    tests := []struct {
        name    string
        input   CreateUserRequest
        wantErr bool
    }{
        {name: "valid", input: CreateUserRequest{Name: "Alice", Email: "alice@example.com"}},
        {name: "missing name", input: CreateUserRequest{Email: "alice@example.com"}, wantErr: true},
        {name: "invalid email", input: CreateUserRequest{Name: "Alice", Email: "bad"}, wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            err := v.Struct(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
            }
        })
    }
}
```

---

## Handler/module test pattern (`module.test.go`)

Module tests are in the **internal** `package <domain>`. They construct an `app.App` and call `NewModule(a)` — same DI path as production:

```go
package user

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "go.opentelemetry.io/otel/trace/noop"

    "restful-boilerplate/internal/app"
    router "restful-boilerplate/pkg/http"
    sqlitedb "restful-boilerplate/pkg/sqlite/db"
    "restful-boilerplate/pkg/testutil"
    cv "restful-boilerplate/pkg/validator"
)

func newTestHandler(t *testing.T) http.Handler {
    t.Helper()
    db := testutil.SetupTestDB(t)
    a := &app.App{
        Queries:   sqlitedb.New(db),
        Validator: cv.New(),
        Tracer:    noop.NewTracerProvider(),
    }
    srv := router.NewRouter()
    srv.Group("/users", func(g *router.Group) {
        NewModule(a).RegisterRoutes(g)
    })
    return srv.Handler
}

func TestCreateUser_HTTP_Success(t *testing.T) {
    h := newTestHandler(t)
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
go test ./internal/<domain>/...
go test ./... -count=1   # all packages, no cache
```
