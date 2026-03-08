---
name: new-domain
description: Scaffold a new domain following the Clean Architecture pattern (domain/ → adapter/)
---

Scaffold a new domain across all layers. Use `user` as the reference implementation.

## Rules

- **Domain layer** (`domain/<domain>/`): Pure Go types, zero framework imports. Exported entity, input types, sentinel errors, Repository interface, Service.
- **Adapter layer** (`adapter/<domain>/`): HTTP handler (net/http), DTOs with `validate` tags, SQLite repository adapter.
- **Infra layer** (`infra/sqlite/`): Migrations, sqlc queries, generated code.
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

### `domain/<domain>/service.go`

```go
package <domain>

import (
    "context"
    "fmt"

    "go.opentelemetry.io/otel/trace"
)

type <Domain>Svc struct {
    repo   Repository
    tracer trace.Tracer
}

func NewService(repo Repository, tracer trace.Tracer) *<Domain>Svc {
    return &<Domain>Svc{repo: repo, tracer: tracer}
}

func (s *<Domain>Svc) Create<Domain>(ctx context.Context, in Create<Domain>Input) (*<Domain>, error) {
    ctx, span := s.tracer.Start(ctx, "<domain>Svc.Create<Domain>")
    defer span.End()
    // ... ID generation, repo.Create, return
}
```

---

## Step 3 — Adapter layer

### `adapter/<domain>/repository.go`

```go
package <domain>

import (
    "context"
    "database/sql"
    "errors"
    "fmt"

    domain<domain> "restful-boilerplate/domain/<domain>"
    sqlitedb "restful-boilerplate/infra/sqlite/db"
)

type SQLite struct {
    q *sqlitedb.Queries
}

func NewSQLite(db *sql.DB) *SQLite {
    return &SQLite{q: sqlitedb.New(db)}
}

// Create inserts a new <domain> into SQLite.
func (r *SQLite) Create(ctx context.Context, u *domain<domain>.<Domain>) error {
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
// GetByID: map sql.ErrNoRows → domain<domain>.ErrNotFound
// Delete: check RowsAffected == 0 → domain<domain>.ErrNotFound
```

### `adapter/<domain>/dto.go`

```go
package <domain>

type Create<Domain>Request struct {
    Name string `json:"name" validate:"required,min=1,max=100" example:"Alice"`
}

type Update<Domain>Request struct {
    Name string `json:"name" validate:"omitempty,min=1,max=100" example:"Alice"`
}
```

### `adapter/<domain>/module.go`

```go
package <domain>

import (
    domain<domain> "restful-boilerplate/domain/<domain>"
    router "restful-boilerplate/infra/http"
)

type Validator interface {
    Validate(i any) error
}

type Module struct {
    svc *domain<domain>.<Domain>Svc
    val Validator
}

func NewModule(svc *domain<domain>.<Domain>Svc, v Validator) *Module {
    return &Module{svc: svc, val: v}
}

func (m *Module) RegisterRoutes(g *router.Group) {
    g.GET("/", m.list<Domain>sHandler)
    g.POST("/", m.create<Domain>Handler)
    g.GET("/{id}", m.get<Domain>ByIDHandler)
    g.PUT("/{id}", m.update<Domain>Handler)
    g.DELETE("/{id}", m.delete<Domain>Handler)
}
```

### `adapter/<domain>/handler.go`

Handlers with swag annotations. Map `<domain>.ErrNotFound` → 404:

```go
func (m *Module) get<Domain>ByIDHandler(w http.ResponseWriter, r *http.Request) {
    u, err := m.svc.Get<Domain>ByID(r.Context(), r.PathValue("id"))
    if err != nil {
        if errors.Is(err, domain<domain>.ErrNotFound) {
            http.Error(w, `{"error":"<domain> not found"}`, http.StatusNotFound)
            return
        }
        http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
        return
    }
    router.WriteJSON(w, http.StatusOK, u)
}
```

---

## Step 4 — Wire into cmd/http/main.go

Inside the versioned group in `main()`:

```go
r.Group("/v1", func(g *router.Group) {
    g.Prefix("/api")

    // <Domain> domain register
    <domain>Repo := <domain>adapter.NewSQLite(db)
    <domain>Svc := <domain>.NewService(<domain>Repo, otel.Tracer("<domain>"))
    g.Group("/<domain>s", <domain>adapter.NewModule(<domain>Svc, v).RegisterRoutes)
})
```

---

## Step 5 — Regenerate Swagger docs

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
- [ ] `domain/<domain>/service.go` — Service with OTEL tracing
- [ ] `adapter/<domain>/repository.go` — SQLite Repository adapter
- [ ] `adapter/<domain>/dto.go` — request DTOs with validate + example tags
- [ ] `adapter/<domain>/module.go` — Module + NewHandler + RegisterRoutes
- [ ] `adapter/<domain>/handler.go` — handlers with swag annotations
- [ ] `cmd/http/main.go` — wire repo → service → handler
- [ ] `make swagger` — regenerate Swagger docs
- [ ] `make check` — fmt + vet + lint + test all pass
