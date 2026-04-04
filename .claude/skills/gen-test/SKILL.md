---
name: gen-test
description: Generate Go unit and integration tests for handlers, service, or DTOs using table-driven tests, net/http httptest, and PostgreSQL
---

Generate idiomatic Go tests for the 3-folder structure. Uses stdlib only. No external test libraries (no testify, etc.).

## Step 0 — Invoke test-master first

Before writing any test code, use the Skill tool to invoke `fullstack-dev-skills:test-master`. Provide it:
- Language: Go (stdlib `testing` only — no testify, no mocks)
- The source file(s) being tested and what they do
- Project constraints: PostgreSQL integration tests, net/http httptest, table-driven tests, no external test libraries

Let test-master apply its full skill set. Use its complete output to build a test plan before writing code.

---

## Rules

- Service tests live in `internal/<domain>/core/service_test.go` (external test package `package <domain>core_test`).
- DTO/input tests live in `internal/<domain>/model/dto_test.go` (internal test package `package <domain>model`).
- Handler/module tests live in `internal/<domain>/module.test.go` (internal test package `package <domain>`).
- Use **table-driven tests**: `tests := []struct{ name string; ... }{ {...}, ... }` with `t.Run(tt.name, ...)`.
- Use `t.Fatal` / `t.Errorf` — never `panic`.
- Test both happy path and error cases.
- Service tests use `pkg/testutil/pgdb.go` helper — requires `TEST_DATABASE_URL` env var (tests skip when absent).
- Do NOT use `t.Parallel()` in service tests — they share a PostgreSQL instance and run sequentially.

---

## File argument

When invoked with a file path, generate the corresponding test file:

| Source file | Test file | What to test |
|-------------|-----------|--------------|
| `internal/<domain>/model/dto.go` | `internal/<domain>/model/dto_test.go` | Validator tag behaviour |
| `internal/<domain>/core/service.go` | `internal/<domain>/core/service_test.go` | CRUD ops with PostgreSQL |
| `internal/<domain>/adapter/http.go` | `internal/<domain>/module.test.go` | HTTP status codes via httptest |

---

## Test helper — service setup (`core/service_test.go`)

Service tests are in the **external** test package (`package <domain>core_test`). They require PostgreSQL via `TEST_DATABASE_URL`:

```go
package usercore_test

import (
    "context"
    "errors"
    "testing"

    "go.opentelemetry.io/otel/trace/noop"

    usercore "restful-boilerplate/internal/user/core"
    pgdb    "restful-boilerplate/pkg/postgres/db"
    "restful-boilerplate/pkg/testutil"
)

func newTestService(t *testing.T) *usercore.Service {
    t.Helper()
    pool := testutil.SetupPgTestDB(t)  // skips if TEST_DATABASE_URL unset
    q := pgdb.New(pool)
    return usercore.NewService(q, noop.NewTracerProvider().Tracer("test"))
}

func TestCreateUser(t *testing.T) {
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
    svc := newTestService(t)
    _, err := svc.GetUserByID(context.Background(), "nonexistent")
    if !errors.Is(err, usercore.ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}
```

To run service tests locally:
```bash
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" go test ./internal/user/core/...
```

Start PostgreSQL via docker-compose first:
```bash
cd infra && docker-compose up -d postgres
```

---

## DTO test pattern (`model/dto_test.go`)

DTOs live in `internal/<domain>/model/dto.go`. Tests use the **internal** `<domain>model` package — no database needed:

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

Module tests construct an `app.App` and call `NewModule(a)` — same DI path as production. Requires PostgreSQL:

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
    pgdb   "restful-boilerplate/pkg/postgres/db"
    "restful-boilerplate/pkg/testutil"
    cv "restful-boilerplate/pkg/validator"
)

func newTestHandler(t *testing.T) http.Handler {
    t.Helper()
    pool := testutil.SetupPgTestDB(t)
    a := &app.App{
        Queries:   pgdb.New(pool),
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
# DTO tests (no DB needed):
go test ./internal/<domain>/model/...

# Service + handler tests (requires PostgreSQL):
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" go test ./internal/<domain>/...

# All packages:
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" go test ./... -count=1
```
