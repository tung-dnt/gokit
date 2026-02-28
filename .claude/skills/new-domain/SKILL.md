---
name: new-domain
description: Scaffold a new domain following the Clean Architecture pattern (domain/ → app/ → infra/)
---

Scaffold a new domain across all three layers. Use `user` as the reference implementation.

## Rules

- **Domain layer** (`domain/<domain>/`): Pure Go types, zero framework imports. Exported entity, input types, sentinel errors, Repository interface.
- **App layer** (`app/<domain>svc/`): Service depends on `<domain>.Repository` interface. All methods exported. OTEL tracing lives here.
- **Infra layer** (`infra/sqlite/<domain>repo/`): Repository adapter maps sqlc types ↔ domain types, maps `sql.ErrNoRows` → `<domain>.ErrNotFound`.
- **HTTP adapter** (`infra/http/<domain>hdl/`): Handler wraps `*<domain>svc.Service`. DTOs with `validate` tags in same package.
- All service/repo methods accept `ctx context.Context` as first parameter.
- Wrap errors: `fmt.Errorf("opName: %w", err)`.
- IDs: `crypto/rand` 8-byte hex via `generateID()` helper in service.

---

## Step 1 — SQL migration + queries

### `infra/sqlite/migrations/<domain>.sql`

```sql
CREATE TABLE IF NOT EXISTS <domain>s (
    id         TEXT     PRIMARY KEY NOT NULL,
    name       TEXT     NOT NULL,
    created_at DATETIME NOT NULL
);
```

### `infra/sqlite/queries/<domain>.sql`

```sql
-- name: Create<Domain> :one
INSERT INTO <domain>s (id, name, created_at)
VALUES (?, ?, ?)
RETURNING *;

-- name: List<Domain>s :many
SELECT * FROM <domain>s ORDER BY created_at ASC;

-- name: Get<Domain>ByID :one
SELECT * FROM <domain>s WHERE id = ? LIMIT 1;

-- name: Update<Domain> :one
UPDATE <domain>s SET name = ? WHERE id = ?
RETURNING *;

-- name: Delete<Domain> :execresult
DELETE FROM <domain>s WHERE id = ?;
```

Then run:
```bash
go tool sqlc generate
go build ./...
```

Also add the embed directive in `infra/sqlite/migrate.go`:
```go
//go:embed migrations/<domain>.sql
var <domain>Schema string
```
And apply it in `Migrate()`.

---

## Step 2 — Domain layer

### `domain/<domain>/entity.go`

```go
package <domain>

import "time"

// <Domain> is the core domain entity.
type <Domain> struct {
    ID        string    `json:"id"         example:"a1b2c3d4"`
    Name      string    `json:"name"       example:"Alice"`
    CreatedAt time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
}

type Create<Domain>Input struct {
    Name string
}

type Update<Domain>Input struct {
    Name string
}
```

### `domain/<domain>/errors.go`

```go
package <domain>

import "errors"

var ErrNotFound = errors.New("<domain>: not found")
```

### `domain/<domain>/port.go`

```go
package <domain>

import "context"

type Repository interface {
    Create(ctx context.Context, u *<Domain>) error
    List(ctx context.Context) ([]*<Domain>, error)
    GetByID(ctx context.Context, id string) (*<Domain>, error)
    Update(ctx context.Context, u *<Domain>) error
    Delete(ctx context.Context, id string) error
}
```

---

## Step 3 — Repository adapter

### `infra/sqlite/<domain>repo/repository.go`

```go
package <domain>repo

import (
    "context"
    "database/sql"
    "errors"
    "fmt"

    "restful-boilerplate/domain/<domain>"
    sqlitedb "restful-boilerplate/infra/sqlite/db"
)

type SQLite struct {
    q *sqlitedb.Queries
}

func NewSQLite(db *sql.DB) *SQLite {
    return &SQLite{q: sqlitedb.New(db)}
}

// Create inserts a new <domain> into SQLite.
func (r *SQLite) Create(ctx context.Context, u *<domain>.<Domain>) error {
    row, err := r.q.Create<Domain>(ctx, sqlitedb.Create<Domain>Params{
        ID: u.ID, Name: u.Name, CreatedAt: u.CreatedAt,
    })
    if err != nil {
        return fmt.Errorf("create <domain>: %w", err)
    }
    *u = *to<Domain>(row)
    return nil
}

// ... other methods follow same pattern
// GetByID: map sql.ErrNoRows → <domain>.ErrNotFound
// Delete: check RowsAffected == 0 → <domain>.ErrNotFound
```

---

## Step 4 — Application service

### `app/<domain>svc/service.go`

```go
package <domain>svc

import (
    "context"
    "fmt"

    "go.opentelemetry.io/otel/trace"
    "restful-boilerplate/domain/<domain>"
)

type Service struct {
    repo   <domain>.Repository
    tracer trace.Tracer
}

func NewService(repo <domain>.Repository, tracer trace.Tracer) *Service {
    return &Service{repo: repo, tracer: tracer}
}

func (s *Service) Create<Domain>(ctx context.Context, in <domain>.Create<Domain>Input) (*<domain>.<Domain>, error) {
    ctx, span := s.tracer.Start(ctx, "<domain>svc.Create<Domain>")
    defer span.End()
    // ... ID generation, repo.Create, return
}
```

---

## Step 5 — HTTP adapter

### `infra/http/<domain>hdl/dto.go`

```go
package <domain>hdl

type Create<Domain>Request struct {
    Name string `json:"name" validate:"required,min=1,max=100" example:"Alice"`
}

type Update<Domain>Request struct {
    Name string `json:"name" validate:"omitempty,min=1,max=100" example:"Alice"`
}
```

### `infra/http/<domain>hdl/routes.go`

```go
package <domain>hdl

import (
    "github.com/labstack/echo/v5"
    "restful-boilerplate/app/<domain>svc"
)

type Handler struct {
    svc *<domain>svc.Service
}

func NewHandler(svc *<domain>svc.Service) *Handler {
    return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(g *echo.Group) {
    g.GET("", h.list<Domain>sHandler)
    g.POST("", h.create<Domain>Handler)
    g.GET("/:id", h.get<Domain>ByIDHandler)
    g.PUT("/:id", h.update<Domain>Handler)
    g.DELETE("/:id", h.delete<Domain>Handler)
}
```

### `infra/http/<domain>hdl/handler.go`

Handlers with swag annotations. Map `<domain>.ErrNotFound` → 404:

```go
func (h *Handler) get<Domain>ByIDHandler(c *echo.Context) error {
    u, err := h.svc.Get<Domain>ByID(c.Request().Context(), c.Param("id"))
    if err != nil {
        if errors.Is(err, <domain>.ErrNotFound) {
            return c.JSON(http.StatusNotFound, map[string]string{"error": "<domain> not found"})
        }
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }
    return c.JSON(http.StatusOK, u)
}
```

---

## Step 6 — Wire into cmd/http/main.go

```go
func registerRouters(g *echo.Group, db *sql.DB) {
    userRepo := userrepo.NewSQLite(db)
    userSvc := usersvc.NewService(userRepo, otel.Tracer("user"))
    userhdl.NewHandler(userSvc).RegisterRoutes(g.Group("/users"))

    // Add new domain:
    <domain>Repo := <domain>repo.NewSQLite(db)
    <domain>Svc := <domain>svc.NewService(<domain>Repo, otel.Tracer("<domain>"))
    <domain>hdl.NewHandler(<domain>Svc).RegisterRoutes(g.Group("/<domain>s"))
}
```

---

## Step 7 — Regenerate Swagger docs

```bash
go tool swag init -g cmd/http/main.go -o docs/
make check
```

---

## Checklist

- [ ] `infra/sqlite/migrations/<domain>.sql` — CREATE TABLE
- [ ] `infra/sqlite/queries/<domain>.sql` — CRUD queries with sqlc annotations
- [ ] `go tool sqlc generate` — generates `infra/sqlite/db/<domain>.sql.go`
- [ ] `infra/sqlite/migrate.go` — embed + apply new migration
- [ ] `domain/<domain>/entity.go` — exported entity + input types
- [ ] `domain/<domain>/errors.go` — ErrNotFound sentinel
- [ ] `domain/<domain>/port.go` — Repository interface
- [ ] `infra/sqlite/<domain>repo/repository.go` — Repository adapter
- [ ] `app/<domain>svc/service.go` — Service with OTEL tracing
- [ ] `infra/http/<domain>hdl/dto.go` — request DTOs with validate + example tags
- [ ] `infra/http/<domain>hdl/routes.go` — Handler + NewHandler + RegisterRoutes
- [ ] `infra/http/<domain>hdl/handler.go` — handlers with swag annotations
- [ ] `cmd/http/main.go` — wire repo → service → handler
- [ ] `make swagger` — regenerate Swagger docs
- [ ] `make check` — fmt + vet + lint + test all pass
