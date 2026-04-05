---
name: new-domain
description: Scaffold a new domain as a flat single package (no sub-packages)
---

Scaffold a new domain under `internal/<domain>/`. Use `user` as the reference implementation.

## Architecture

Each domain is a **flat single package** — all concerns in `internal/<domain>/` separated by file-name prefix:

| File | Purpose |
|---|---|
| `module.<domain>.go` | DI wiring + RegisterRoutes |
| `adapter.http.go` | HTTP handlers (unexported httpAdapter) |
| `service.<domain>.go` | Business logic (unexported <domain>Service) |
| `domain.dto.go` | Request DTOs with validate + json + example tags |
| `domain.error.go` | ErrNotFound sentinel |
| `mapping.response.go` | DB model → HTTP response conversion |

All types in the same `package <domain>`. No sub-package imports within a domain.

**Shared ID utility:** `shared "restful-boilerplate/pkg/util"` → `shared.GenerateID() (string, error)`

## Rules

- Single `ErrNotFound` in `domain.error.go` — no duplication
- No type aliases — use `Create<Domain>Request` directly in service method signatures
- `httpAdapter`, `<domain>Service`, constructors, and CRUD methods are **unexported**
- `Module`, `NewModule`, `RegisterRoutes` are exported (consumed by `main.go`)
- `ErrNotFound` is in the same package — handlers check via `errors.Is(err, ErrNotFound)`
- Handler decode+validate: `router.Bind(m.val, w, r, &req)` — returns false + writes error; just `return` after
- All service/db methods accept `ctx context.Context` as first parameter
- Wrap errors: `fmt.Errorf("opName: %w", err)`
- OTEL tracing on every service method via `telemetry.SpanErr`

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

## Step 2 — domain.error.go

```go
package <domain>

import "errors"

// ErrNotFound indicates that the requested <domain> does not exist.
var ErrNotFound = errors.New("<domain>: not found")
```

---

## Step 3 — domain.dto.go

```go
package <domain>

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

## Step 4 — mapping.response.go

```go
package <domain>

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

## Step 5 — service.\<domain\>.go

```go
package <domain>

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"

	shared "restful-boilerplate/pkg/util"
	pgdb "restful-boilerplate/pkg/postgres/db"
	"restful-boilerplate/pkg/telemetry"
)

type <domain>Service struct {
	q      *pgdb.Queries
	tracer trace.Tracer
}

func new<Domain>Service(q *pgdb.Queries, tracer trace.Tracer) *<domain>Service {
	return &<domain>Service{q: q, tracer: tracer}
}

func (s *<domain>Service) create<Domain>(ctx context.Context, in Create<Domain>Request) (*pgdb.<Domain>, error) {
	ctx, span := s.tracer.Start(ctx, "<domain>.create<Domain>")
	defer span.End()

	id, err := shared.GenerateID()
	if err != nil {
		return nil, telemetry.SpanErr(span, err, "generate id")
	}

	row, err := s.q.Create<Domain>(ctx, pgdb.Create<Domain>Params{
		ID: id, Name: in.Name, CreatedAt: time.Now(),
	})
	if err != nil {
		return nil, telemetry.SpanErr(span, err, "create<Domain>")
	}
	return &row, nil
}

func (s *<domain>Service) get<Domain>ByID(ctx context.Context, id string) (*pgdb.<Domain>, error) {
	ctx, span := s.tracer.Start(ctx, "<domain>.get<Domain>ByID")
	defer span.End()

	row, err := s.q.Get<Domain>ByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, telemetry.SpanErr(span, err, "get<Domain>ByID")
	}
	return &row, nil
}

func (s *<domain>Service) list<Domain>s(ctx context.Context) ([]*pgdb.<Domain>, error) {
	ctx, span := s.tracer.Start(ctx, "<domain>.list<Domain>s")
	defer span.End()

	rows, err := s.q.List<Domain>s(ctx)
	if err != nil {
		return nil, telemetry.SpanErr(span, err, "list<Domain>s")
	}
	out := make([]*pgdb.<Domain>, 0, len(rows))
	for i := range rows {
		out = append(out, &rows[i])
	}
	return out, nil
}

func (s *<domain>Service) update<Domain>(ctx context.Context, id string, in Update<Domain>Request) (*pgdb.<Domain>, error) {
	ctx, span := s.tracer.Start(ctx, "<domain>.update<Domain>")
	defer span.End()

	existing, err := s.get<Domain>ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != "" {
		existing.Name = in.Name
	}
	row, err := s.q.Update<Domain>(ctx, pgdb.Update<Domain>Params{ID: id, Name: existing.Name})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, telemetry.SpanErr(span, err, "update<Domain>")
	}
	return &row, nil
}

func (s *<domain>Service) delete<Domain>(ctx context.Context, id string) error {
	ctx, span := s.tracer.Start(ctx, "<domain>.delete<Domain>")
	defer span.End()

	result, err := s.q.Delete<Domain>(ctx, id)
	if err != nil {
		return telemetry.SpanErr(span, err, "delete<Domain>")
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
```

---

## Step 6 — adapter.http.go

```go
package <domain>

import (
	"errors"
	"net/http"

	"restful-boilerplate/internal/app"
	router "restful-boilerplate/pkg/http"
)

type httpAdapter struct {
	svc *<domain>Service
	val app.Validator
}

func newHTTPAdapter(svc *<domain>Service, val app.Validator) *httpAdapter {
	return &httpAdapter{svc: svc, val: val}
}

func (m *httpAdapter) writeErr(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		router.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "<domain> not found"})
		return
	}
	router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

// List<Domain>sHandler returns all <domain>s.
//
//	@Summary      List <domain>s
//	@Tags         <domain>s
//	@Produce      json
//	@Success      200  {array}   <Domain>Response
//	@Failure      500  {object}  map[string]string
//	@Router       /<domain>s [get]
func (m *httpAdapter) list<Domain>sHandler(w http.ResponseWriter, r *http.Request) {
	items, err := m.svc.list<Domain>s(r.Context())
	if err != nil {
		m.writeErr(w, err)
		return
	}
	resp := make([]<Domain>Response, 0, len(items))
	for _, d := range items {
		resp = append(resp, ToResponse(*d))
	}
	router.WriteJSON(w, http.StatusOK, resp)
}

// create<Domain>Handler creates a new <domain>.
//
//	@Summary      Create <domain>
//	@Tags         <domain>s
//	@Accept       json
//	@Produce      json
//	@Param        body  body      Create<Domain>Request  true  "<Domain> data"
//	@Success      201   {object}  <Domain>Response
//	@Failure      400   {object}  map[string]string
//	@Failure      422   {object}  map[string]string
//	@Failure      500   {object}  map[string]string
//	@Router       /<domain>s [post]
func (m *httpAdapter) create<Domain>Handler(w http.ResponseWriter, r *http.Request) {
	var req Create<Domain>Request
	if !router.Bind(m.val, w, r, &req) {
		return
	}
	d, err := m.svc.create<Domain>(r.Context(), req)
	if err != nil {
		m.writeErr(w, err)
		return
	}
	router.WriteJSON(w, http.StatusCreated, ToResponse(*d))
}

// get<Domain>ByIDHandler gets a <domain> by ID.
//
//	@Summary      Get <domain> by ID
//	@Tags         <domain>s
//	@Produce      json
//	@Param        id   path      string  true  "<Domain> ID"
//	@Success      200  {object}  <Domain>Response
//	@Failure      404  {object}  map[string]string
//	@Failure      500  {object}  map[string]string
//	@Router       /<domain>s/{id} [get]
func (m *httpAdapter) get<Domain>ByIDHandler(w http.ResponseWriter, r *http.Request) {
	d, err := m.svc.get<Domain>ByID(r.Context(), r.PathValue("id"))
	if err != nil {
		m.writeErr(w, err)
		return
	}
	router.WriteJSON(w, http.StatusOK, ToResponse(*d))
}

// update<Domain>Handler updates a <domain>.
//
//	@Summary      Update <domain>
//	@Tags         <domain>s
//	@Accept       json
//	@Produce      json
//	@Param        id    path      string                 true  "<Domain> ID"
//	@Param        body  body      Update<Domain>Request  true  "<Domain> data"
//	@Success      200   {object}  <Domain>Response
//	@Failure      400   {object}  map[string]string
//	@Failure      404   {object}  map[string]string
//	@Failure      500   {object}  map[string]string
//	@Router       /<domain>s/{id} [put]
func (m *httpAdapter) update<Domain>Handler(w http.ResponseWriter, r *http.Request) {
	var req Update<Domain>Request
	if !router.Bind(m.val, w, r, &req) {
		return
	}
	d, err := m.svc.update<Domain>(r.Context(), r.PathValue("id"), req)
	if err != nil {
		m.writeErr(w, err)
		return
	}
	router.WriteJSON(w, http.StatusOK, ToResponse(*d))
}

// delete<Domain>Handler deletes a <domain>.
//
//	@Summary      Delete <domain>
//	@Tags         <domain>s
//	@Produce      json
//	@Param        id   path      string  true  "<Domain> ID"
//	@Success      204
//	@Failure      404  {object}  map[string]string
//	@Failure      500  {object}  map[string]string
//	@Router       /<domain>s/{id} [delete]
func (m *httpAdapter) delete<Domain>Handler(w http.ResponseWriter, r *http.Request) {
	if err := m.svc.delete<Domain>(r.Context(), r.PathValue("id")); err != nil {
		m.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

---

## Step 7 — module.\<domain\>.go

```go
// Package <domain> registers the <domain> module.
package <domain>

import (
	"restful-boilerplate/internal/app"
	router "restful-boilerplate/pkg/http"
)

// Module exposes <domain> endpoints over HTTP.
type Module struct {
	httpAdapter *httpAdapter
}

// NewModule wires the <domain> service from the shared App container.
func NewModule(a *app.App) *Module {
	svc := new<Domain>Service(a.Queries, a.Tracer.Tracer("<domain>"))
	return &Module{httpAdapter: newHTTPAdapter(svc, a.Validator)}
}

// RegisterRoutes mounts all <domain> endpoints onto g.
func (m *Module) RegisterRoutes(g *router.Group) {
	g.GET("/", m.httpAdapter.list<Domain>sHandler)
	g.POST("/", m.httpAdapter.create<Domain>Handler)
	g.GET("/{id}", m.httpAdapter.get<Domain>ByIDHandler)
	g.PUT("/{id}", m.httpAdapter.update<Domain>Handler)
	g.DELETE("/{id}", m.httpAdapter.delete<Domain>Handler)
}
```

---

## Step 8 — Wire into main.go

```go
import (
    <domain> "restful-boilerplate/internal/<domain>"
)

// Inside r.Group("/v1", ...):
g.Group("/<domain>s", <domain>.NewModule(a).RegisterRoutes)
```

---

## Step 9 — Regenerate Swagger docs

```bash
go tool swag init -g main.go -o docs/
make check
```

---

## Checklist

- [ ] `pkg/postgres/migrations/<domain>.sql` — CREATE TABLE (TIMESTAMPTZ for timestamps)
- [ ] `pkg/postgres/queries/<domain>.sql` — CRUD queries with `$1, $2, ...` params
- [ ] `go tool sqlc generate` + `go build ./...`
- [ ] `internal/<domain>/domain.error.go` — ErrNotFound sentinel
- [ ] `internal/<domain>/domain.dto.go` — Create/UpdateXRequest with validate + example tags
- [ ] `internal/<domain>/mapping.response.go` — <Domain>Response + ToResponse (pgdb.<Domain>)
- [ ] `internal/<domain>/service.<domain>.go` — unexported service + all CRUD methods
- [ ] `internal/<domain>/adapter.http.go` — unexported httpAdapter + handler methods
- [ ] `internal/<domain>/module.<domain>.go` — Module + NewModule(a *app.App) + RegisterRoutes
- [ ] `main.go` — add group wiring (import domain root package)
- [ ] `make swagger` + `make check`
