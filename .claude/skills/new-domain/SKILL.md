---
name: new-domain
description: Scaffold a new domain following the sub-package structure (model/, core/, mapping/, adapter/)
---

Scaffold a new domain under `internal/<domain>/`. Use `user` as the reference implementation.

## Architecture

Each domain has **four sub-packages** plus a root module file:

| Location | Go package | Purpose |
|---|---|---|
| `internal/<domain>/module.go` | `<domain>` | DI wiring + RegisterRoutes |
| `internal/<domain>/adapter/http.go` | `<domain>adapter` | HTTP handlers (HTTPAdapter) |
| `internal/<domain>/core/` | `<domain>core` | Business logic — Service, ErrNotFound, type aliases |
| `internal/<domain>/mapping/mapping.go` | `<domain>mapping` | DB model → HTTP response conversion |
| `internal/<domain>/model/` | `<domain>model` | Request DTOs + ErrNotFound sentinel |

Dependency rule: `adapter/` imports `core/`, `mapping/`, `model/`. `core/` imports `model/`. `model/` and `mapping/` import nothing from the domain. **No circular deps.**

Shared ID utility: `restful-boilerplate/internal/shared` → `shared.GenerateID() (string, error)`

## Rules

- `model/` is `package <domain>model` — request DTOs + `ErrNotFound` sentinel
- `core/` is `package <domain>core` — re-exports type aliases + `ErrNotFound` + consolidated `Service`
- `mapping/` is `package <domain>mapping` — `<Domain>Response` + `ToResponse(pgdb.<Domain>)`
- `adapter/` is `package <domain>adapter` — `HTTPAdapter` with exported handler methods
- `module.go` is `package <domain>` — `Module{httpAdapter}` + `NewModule(a)` + `RegisterRoutes`
- Handlers call `mapping.ToResponse(...)` before `router.WriteJSON`
- `ErrNotFound` defined in `model/errors.go`; re-declared in `core/<domain>.go`; handlers check via `errors.Is(err, <domain>core.ErrNotFound)`
- All service/db methods accept `ctx context.Context` as first parameter
- Wrap errors: `fmt.Errorf("opName: %w", err)`
- IDs: `shared.GenerateID()` from `restful-boilerplate/internal/shared`

---

## Step 1 — SQL migration + queries

### `pkg/postgres/migrations/<domain>.sql`

```sql
CREATE TABLE IF NOT EXISTS <domain>s (
    id         TEXT        PRIMARY KEY NOT NULL,
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
```

### `pkg/postgres/queries/<domain>.sql`

```sql
-- name: Create<Domain> :one
INSERT INTO <domain>s (id, name, created_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: List<Domain>s :many
SELECT * FROM <domain>s ORDER BY created_at ASC;

-- name: Get<Domain>ByID :one
SELECT * FROM <domain>s WHERE id = $1 LIMIT 1;

-- name: Update<Domain> :one
UPDATE <domain>s SET name = $1 WHERE id = $2
RETURNING *;

-- name: Delete<Domain> :execresult
DELETE FROM <domain>s WHERE id = $1;
```

Then run:
```bash
go tool sqlc generate
go build ./...
```

---

## Step 2 — model/ layer

### `internal/<domain>/model/errors.go`

```go
package <domain>model

import "errors"

// ErrNotFound indicates that the requested <domain> does not exist.
var ErrNotFound = errors.New("<domain>: not found")
```

### `internal/<domain>/model/dto.go`

```go
package <domain>model

// Create<Domain>Request is the validated input for creating a new <domain>.
type Create<Domain>Request struct {
    Name string `json:"name" validate:"required,min=1,max=100" example:"Alice"`
}

// Update<Domain>Request is the validated input for updating an existing <domain>.
type Update<Domain>Request struct {
    Name string `json:"name" validate:"omitempty,min=1,max=100" example:"Alice"`
}
```

---

## Step 3 — core/ layer

### `internal/<domain>/core/<domain>.go`

```go
package <domain>core

import (
    "errors"
    <domain>model "restful-boilerplate/internal/<domain>/model"
)

// ErrNotFound is returned when a requested <domain> does not exist.
var ErrNotFound = errors.New("<domain>: not found")

// Create<Domain>Input is the input for creating a <domain>.
type Create<Domain>Input = <domain>model.Create<Domain>Request

// Update<Domain>Input is the input for updating a <domain>.
type Update<Domain>Input = <domain>model.Update<Domain>Request
```

### `internal/<domain>/core/service.go`

```go
package <domain>core

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"

    "restful-boilerplate/internal/shared"
    pgdb "restful-boilerplate/pkg/postgres/db"
)

// Service orchestrates <domain> use-cases on top of pgdb.Queries.
type Service struct {
    q      *pgdb.Queries
    tracer trace.Tracer
}

// NewService creates a Service backed by q and traced via tracer.
func NewService(q *pgdb.Queries, tracer trace.Tracer) *Service {
    return &Service{q: q, tracer: tracer}
}

func (s *Service) Create<Domain>(ctx context.Context, in Create<Domain>Input) (*pgdb.<Domain>, error) {
    ctx, span := s.tracer.Start(ctx, "<domain>.Create<Domain>")
    defer span.End()

    id, err := shared.GenerateID()
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, fmt.Errorf("generate id: %w", err)
    }

    row, err := s.q.Create<Domain>(ctx, pgdb.Create<Domain>Params{
        ID: id, Name: in.Name, CreatedAt: time.Now(),
    })
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, fmt.Errorf("create<Domain>: %w", err)
    }
    return &row, nil
}

func (s *Service) Get<Domain>ByID(ctx context.Context, id string) (*pgdb.<Domain>, error) {
    ctx, span := s.tracer.Start(ctx, "<domain>.Get<Domain>ByID")
    defer span.End()

    row, err := s.q.Get<Domain>ByID(ctx, id)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, ErrNotFound
        }
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, fmt.Errorf("get<Domain>ByID: %w", err)
    }
    return &row, nil
}

func (s *Service) List<Domain>s(ctx context.Context) ([]*pgdb.<Domain>, error) {
    ctx, span := s.tracer.Start(ctx, "<domain>.List<Domain>s")
    defer span.End()

    rows, err := s.q.List<Domain>s(ctx)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, fmt.Errorf("list<Domain>s: %w", err)
    }
    out := make([]*pgdb.<Domain>, 0, len(rows))
    for i := range rows {
        out = append(out, &rows[i])
    }
    return out, nil
}

func (s *Service) Update<Domain>(ctx context.Context, id string, in Update<Domain>Input) (*pgdb.<Domain>, error) {
    ctx, span := s.tracer.Start(ctx, "<domain>.Update<Domain>")
    defer span.End()

    existing, err := s.Get<Domain>ByID(ctx, id)
    if err != nil {
        return nil, err
    }
    if in.Name != "" {
        existing.Name = in.Name
    }
    row, err := s.q.Update<Domain>(ctx, pgdb.Update<Domain>Params{
        ID: id, Name: existing.Name,
    })
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, ErrNotFound
        }
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return nil, fmt.Errorf("update<Domain>: %w", err)
    }
    return &row, nil
}

func (s *Service) Delete<Domain>(ctx context.Context, id string) error {
    ctx, span := s.tracer.Start(ctx, "<domain>.Delete<Domain>")
    defer span.End()

    result, err := s.q.Delete<Domain>(ctx, id)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return fmt.Errorf("delete<Domain>: %w", err)
    }
    if result.RowsAffected() == 0 {
        return ErrNotFound
    }
    return nil
}
```

---

## Step 4 — mapping/ layer

### `internal/<domain>/mapping/mapping.go`

```go
// Package <domain>mapping provides type conversions for the <domain> domain.
package <domain>mapping

import (
    "time"
    pgdb "restful-boilerplate/pkg/postgres/db"
)

// <Domain>Response is the HTTP JSON response shape for a <domain>.
type <Domain>Response struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts a pgdb.<Domain> DB model to a <Domain>Response.
func ToResponse(d pgdb.<Domain>) <Domain>Response {
    return <Domain>Response{ID: d.ID, Name: d.Name, CreatedAt: d.CreatedAt}
}
```

---

## Step 5 — adapter/ layer

### `internal/<domain>/adapter/http.go`

```go
// Package <domain>adapter handles HTTP requests for the <domain> domain.
package <domain>adapter

import (
    "encoding/json"
    "errors"
    "net/http"

    "restful-boilerplate/internal/app"
    <domain>core "restful-boilerplate/internal/<domain>/core"
    <domain>mapping "restful-boilerplate/internal/<domain>/mapping"
    <domain>model "restful-boilerplate/internal/<domain>/model"
    "restful-boilerplate/pkg/http"
)

// HTTPAdapter handles HTTP requests for the <domain> domain.
type HTTPAdapter struct {
    svc *<domain>core.Service
    val app.Validator
}

// NewHTTPAdapter creates a new HTTPAdapter.
func NewHTTPAdapter(svc *<domain>core.Service, val app.Validator) *HTTPAdapter {
    return &HTTPAdapter{svc: svc, val: val}
}

// List<Domain>sHandler returns all <domain>s.
func (m *HTTPAdapter) List<Domain>sHandler(w http.ResponseWriter, r *http.Request) {
    items, err := m.svc.List<Domain>s(r.Context())
    if err != nil {
        router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }
    resp := make([]<domain>mapping.<Domain>Response, 0, len(items))
    for _, d := range items {
        resp = append(resp, <domain>mapping.ToResponse(*d))
    }
    router.WriteJSON(w, http.StatusOK, resp)
}

// Create<Domain>Handler creates a new <domain>.
func (m *HTTPAdapter) Create<Domain>Handler(w http.ResponseWriter, r *http.Request) {
    var req <domain>model.Create<Domain>Request
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        router.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
        return
    }
    if err := m.val.Validate(&req); err != nil {
        router.WriteJSON(w, http.StatusUnprocessableEntity, err)
        return
    }
    d, err := m.svc.Create<Domain>(r.Context(), req)
    if err != nil {
        router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }
    router.WriteJSON(w, http.StatusCreated, <domain>mapping.ToResponse(*d))
}

// Get<Domain>ByIDHandler gets a <domain> by ID.
func (m *HTTPAdapter) Get<Domain>ByIDHandler(w http.ResponseWriter, r *http.Request) {
    d, err := m.svc.Get<Domain>ByID(r.Context(), r.PathValue("id"))
    if err != nil {
        if errors.Is(err, <domain>core.ErrNotFound) {
            router.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "<domain> not found"})
            return
        }
        router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }
    router.WriteJSON(w, http.StatusOK, <domain>mapping.ToResponse(*d))
}

// Update<Domain>Handler updates a <domain>.
func (m *HTTPAdapter) Update<Domain>Handler(w http.ResponseWriter, r *http.Request) {
    var req <domain>model.Update<Domain>Request
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        router.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
        return
    }
    if err := m.val.Validate(&req); err != nil {
        router.WriteJSON(w, http.StatusUnprocessableEntity, err)
        return
    }
    d, err := m.svc.Update<Domain>(r.Context(), r.PathValue("id"), req)
    if err != nil {
        if errors.Is(err, <domain>core.ErrNotFound) {
            router.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "<domain> not found"})
            return
        }
        router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }
    router.WriteJSON(w, http.StatusOK, <domain>mapping.ToResponse(*d))
}

// Delete<Domain>Handler deletes a <domain>.
func (m *HTTPAdapter) Delete<Domain>Handler(w http.ResponseWriter, r *http.Request) {
    if err := m.svc.Delete<Domain>(r.Context(), r.PathValue("id")); err != nil {
        if errors.Is(err, <domain>core.ErrNotFound) {
            router.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "<domain> not found"})
            return
        }
        router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

---

## Step 6 — module.go (domain root)

### `internal/<domain>/module.go`

```go
// Package <domain> registers the <domain> module.
package <domain>

import (
    "restful-boilerplate/internal/app"
    <domain>adapter "restful-boilerplate/internal/<domain>/adapter"
    <domain>core "restful-boilerplate/internal/<domain>/core"
    router "restful-boilerplate/pkg/http"
)

// Module exposes <domain> endpoints over HTTP.
type Module struct {
    httpAdapter *<domain>adapter.HTTPAdapter
}

// NewModule wires the <domain> service from the shared App container.
func NewModule(a *app.App) *Module {
    svc := <domain>core.NewService(a.Queries, a.Tracer.Tracer("<domain>"))
    return &Module{httpAdapter: <domain>adapter.NewHTTPAdapter(svc, a.Validator)}
}

// RegisterRoutes mounts all <domain> endpoints onto g.
func (m *Module) RegisterRoutes(g *router.Group) {
    g.GET("/", m.httpAdapter.List<Domain>sHandler)
    g.POST("/", m.httpAdapter.Create<Domain>Handler)
    g.GET("/{id}", m.httpAdapter.Get<Domain>ByIDHandler)
    g.PUT("/{id}", m.httpAdapter.Update<Domain>Handler)
    g.DELETE("/{id}", m.httpAdapter.Delete<Domain>Handler)
}
```

---

## Step 7 — Wire into main.go

```go
import (
    "<domain>" "restful-boilerplate/internal/<domain>"
)

// Inside r.Group("/v1", ...):
g.Group("/<domain>s", <domain>.NewModule(a).RegisterRoutes)
```

---

## Step 8 — Regenerate Swagger docs

```bash
go tool swag init -g main.go -o docs/
make check
```

---

## Checklist

- [ ] `pkg/postgres/migrations/<domain>.sql` — CREATE TABLE (TIMESTAMPTZ for timestamps)
- [ ] `pkg/postgres/queries/<domain>.sql` — CRUD queries with `$1, $2, ...` params
- [ ] `go tool sqlc generate` + `go build ./...`
- [ ] `internal/<domain>/model/errors.go` — ErrNotFound sentinel
- [ ] `internal/<domain>/model/dto.go` — Create/UpdateXRequest with validate + example tags
- [ ] `internal/<domain>/core/<domain>.go` — ErrNotFound + type aliases from model
- [ ] `internal/<domain>/core/service.go` — Service + NewService + all CRUD methods (pgdb, pgx.ErrNoRows)
- [ ] `internal/<domain>/mapping/mapping.go` — <Domain>Response + ToResponse (pgdb.<Domain>)
- [ ] `internal/<domain>/adapter/http.go` — HTTPAdapter + exported handler methods
- [ ] `internal/<domain>/module.go` — Module + NewModule(a *app.App) + RegisterRoutes
- [ ] `main.go` — add group wiring (import domain root package)
- [ ] `make swagger` + `make check`
