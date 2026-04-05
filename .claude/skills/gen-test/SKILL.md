---
name: gen-test
description: Generate Go unit and integration tests for handlers, service, or DTOs using table-driven tests, net/http httptest, and PostgreSQL
---

Generate idiomatic Go tests for the flat domain package structure. Uses stdlib only. No external test libraries (no testify, etc.).

## Step 0 — Invoke test-master first

Before writing any test code, use the Skill tool to invoke `fullstack-dev-skills:test-master`. Provide it:
- Language: Go (stdlib `testing` only — no testify, no mocks)
- The source file(s) being tested and what they do
- Project constraints: PostgreSQL integration tests, net/http httptest, table-driven tests, no external test libraries

Let test-master apply its full skill set. Use its complete output to build a test plan before writing code.

---

## Rules

- All domain files are in the same flat package `package <domain>`.
- Service tests live in `internal/<domain>/service_test.go` (external test package `package <domain>_test`).
- DTO tests live in `internal/<domain>/domain_dto_test.go` (internal test package `package <domain>`).
- Handler/module tests live in `internal/<domain>/module_test.go` (internal test package `package <domain>`).
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
| `internal/<domain>/domain.dto.go` | `internal/<domain>/domain_dto_test.go` | Validator tag behaviour |
| `internal/<domain>/service.<domain>.go` | `internal/<domain>/service_test.go` | CRUD ops with PostgreSQL |
| `internal/<domain>/adapter.http.go` | `internal/<domain>/module_test.go` | HTTP status codes via httptest |

---

## Test helper — service setup (`service_test.go`)

Service tests are in the **external** test package (`package <domain>_test`). They require PostgreSQL via `TEST_DATABASE_URL`.

Note: service struct and constructor are unexported — test via the module's `NewModule` or expose a test helper if needed. Alternatively, test service behaviour through the HTTP layer using `module_test.go`.

```go
package user_test

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "go.opentelemetry.io/otel/trace/noop"

    "restful-boilerplate/internal/app"
    "restful-boilerplate/internal/user"
    router "restful-boilerplate/pkg/http"
    pgdb   "restful-boilerplate/pkg/postgres/db"
    "restful-boilerplate/pkg/testutil"
    cv "restful-boilerplate/pkg/validator"
)

func newTestHandler(t *testing.T) http.Handler {
    t.Helper()
    pool := testutil.SetupPgTestDB(t)  // skips if TEST_DATABASE_URL unset
    a := &app.App{
        Queries:   pgdb.New(pool),
        Validator: cv.New(),
        Tracer:    noop.NewTracerProvider(),
    }
    srv := router.NewRouter()
    srv.Group("/users", func(g *router.Group) {
        user.NewModule(a).RegisterRoutes(g)
    })
    return srv.Handler
}
```

---

## DTO test pattern (`domain_dto_test.go`)

Internal package — no database needed:

```go
package user

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

## Handler/module test pattern (`module_test.go`)

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
go test ./internal/<domain>/...

# Service + handler tests (requires PostgreSQL):
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" go test ./internal/<domain>/...

# All packages:
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" go test ./... -count=1
```
