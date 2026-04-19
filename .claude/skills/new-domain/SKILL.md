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
| `adapter.http.go` | HTTP handlers + `<domain>Svc` service seam interface |
| `service.<domain>.go` | Business logic (unexported `<domain>Service`, depends on `pgdb.Querier`) |
| `domain.dto.go` | Request DTOs with validate + json + example tags |
| `domain.error.go` | Domain error sentinels (one or many) |
| `mapping.response.go` | DB model → HTTP response conversion |

All types in the same `package <domain>`. No sub-package imports within a domain.

**Shared ID utility:** `shared "gokit/pkg/util"` → `shared.GenerateID() (string, error)`

## Rules

- Error sentinels in `domain.error.go` — one or many per domain (e.g. `ErrNotFound`, `ErrConflict`). No duplication.
- No type aliases — use `Create<Domain>Request` directly in service method signatures
- `httpAdapter`, `<domain>Service`, `<domain>Svc` interface, constructors, and CRUD methods are **unexported**
- `Module`, `NewModule`, `RegisterRoutes` are exported (consumed by `main.go`)
- Service depends on `pgdb.Querier` (interface) — **not** `*pgdb.Queries`. This enables mockable unit tests.
- Adapter depends on `<domain>Svc` interface — **not** the concrete `*<domain>Service`. The concrete type satisfies the seam; `mock<Domain>Svc` satisfies it in tests.
- Handler decode+validate: `router.Bind(m.val, w, r, &req)` — returns false + writes error; just `return` after
- `writeErr(r, w, err)` takes `*http.Request` so it can pull `logger.FromContext(r.Context())` for trace-correlated logging. Response bodies never leak raw `err.Error()` — always a stable string.
- All service/db methods accept `ctx context.Context` as first parameter
- OTEL tracing on every service method via `pkg/telemetry`:
  - `telemetry.SpanExpectedErr(span, ErrNotFound, "<op>", telemetry.ErrKindNotFound)` for handled/4xx errors (span status stays Unset per OTel HTTP semconv).
  - `telemetry.SpanUnexpectedErr(span, err, "<op>")` for unhandled/5xx errors (span status Error + classified `error.type`).
  - Op string = span name = `"<domain>.<domain>Service.<methodName>"` (e.g. `"user.userService.createUser"`).
- `pgx.ErrNoRows` → `ErrNotFound` via `SpanExpectedErr`. For `DELETE`, `RowsAffected() == 0` → `ErrNotFound`.

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

// Add more sentinels as the domain grows, e.g.:
// var ErrConflict = errors.New("<domain>: conflict")
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

	pgdb "gokit/pkg/postgres/db"
)

// <domain>Response is the HTTP JSON response shape for a <domain>.
type <domain>Response struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts a pgdb.<Domain> DB model to a <domain>Response.
func ToResponse(d pgdb.<Domain>) <domain>Response {
	return <domain>Response{ID: d.ID, Name: d.Name, CreatedAt: d.CreatedAt}
}
```

---

## Step 5 — service.\<domain\>.go

Service depends on `pgdb.Querier` (interface) so tests can inject a mock. Use `telemetry.SpanExpectedErr` / `SpanUnexpectedErr` — not the legacy `SpanErr`.

```go
package <domain>

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"

	"gokit/pkg/logger"
	pgdb "gokit/pkg/postgres/db"
	"gokit/pkg/telemetry"
	shared "gokit/pkg/util"
)

type <domain>Service struct {
	db     pgdb.Querier
	tracer trace.Tracer
}

func new<Domain>Service(q pgdb.Querier, tracer trace.Tracer) *<domain>Service {
	return &<domain>Service{db: q, tracer: tracer}
}

func (s *<domain>Service) create<Domain>(ctx context.Context, in Create<Domain>Request) (*pgdb.<Domain>, error) {
	ctx, span := s.tracer.Start(ctx, "<domain>.<domain>Service.create<Domain>")
	defer span.End()

	logger.FromContext(ctx).InfoContext(ctx, "creating <domain>", slog.String("name", in.Name))

	id, err := shared.GenerateID()
	if err != nil {
		return nil, telemetry.SpanUnexpectedErr(span, err, "<domain>.<domain>Service.create<Domain>: generate id")
	}

	row, err := s.db.Create<Domain>(ctx, pgdb.Create<Domain>Params{
		ID: id, Name: in.Name, CreatedAt: time.Now(),
	})
	if err != nil {
		return nil, telemetry.SpanUnexpectedErr(span, err, "<domain>.<domain>Service.create<Domain>")
	}
	return &row, nil
}

func (s *<domain>Service) get<Domain>ByID(ctx context.Context, id string) (*pgdb.<Domain>, error) {
	ctx, span := s.tracer.Start(ctx, "<domain>.<domain>Service.get<Domain>ByID")
	defer span.End()

	row, err := s.db.Get<Domain>ByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, telemetry.SpanExpectedErr(span, ErrNotFound, "<domain>.<domain>Service.get<Domain>ByID", telemetry.ErrKindNotFound)
		}
		return nil, telemetry.SpanUnexpectedErr(span, err, "<domain>.<domain>Service.get<Domain>ByID")
	}
	return &row, nil
}

func (s *<domain>Service) list<Domain>s(ctx context.Context) ([]*pgdb.<Domain>, error) {
	ctx, span := s.tracer.Start(ctx, "<domain>.<domain>Service.list<Domain>s")
	defer span.End()

	rows, err := s.db.List<Domain>s(ctx)
	if err != nil {
		return nil, telemetry.SpanUnexpectedErr(span, err, "<domain>.<domain>Service.list<Domain>s")
	}
	out := make([]*pgdb.<Domain>, 0, len(rows))
	for i := range rows {
		out = append(out, &rows[i])
	}
	return out, nil
}

func (s *<domain>Service) update<Domain>(ctx context.Context, id string, in Update<Domain>Request) (*pgdb.<Domain>, error) {
	ctx, span := s.tracer.Start(ctx, "<domain>.<domain>Service.update<Domain>")
	defer span.End()

	row, err := s.db.Update<Domain>(ctx, pgdb.Update<Domain>Params{ID: id, Name: in.Name})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, telemetry.SpanExpectedErr(span, ErrNotFound, "<domain>.<domain>Service.update<Domain>", telemetry.ErrKindNotFound)
		}
		return nil, telemetry.SpanUnexpectedErr(span, err, "<domain>.<domain>Service.update<Domain>")
	}
	return &row, nil
}

func (s *<domain>Service) delete<Domain>(ctx context.Context, id string) error {
	ctx, span := s.tracer.Start(ctx, "<domain>.<domain>Service.delete<Domain>")
	defer span.End()

	result, err := s.db.Delete<Domain>(ctx, id)
	if err != nil {
		return telemetry.SpanUnexpectedErr(span, err, "<domain>.<domain>Service.delete<Domain>")
	}
	if result.RowsAffected() == 0 {
		return telemetry.SpanExpectedErr(span, ErrNotFound, "<domain>.<domain>Service.delete<Domain>", telemetry.ErrKindNotFound)
	}
	return nil
}
```

---

## Step 6 — adapter.http.go

Define the `<domain>Svc` interface (service seam) and depend on it — not the concrete service type. `writeErr` takes `*http.Request` to pull a trace-correlated logger via `logger.FromContext`. Multiple error sentinels can be switched via `errors.Is` inside `writeErr`.

```go
package <domain>

import (
	"context"
	"errors"
	"net/http"

	"gokit/internal/app"
	router "gokit/pkg/http"
	"gokit/pkg/logger"
	pgdb "gokit/pkg/postgres/db"
)

// <domain>Svc is the service seam consumed by httpAdapter. Concrete *<domain>Service
// satisfies it; tests inject a mock implementation.
type <domain>Svc interface {
	create<Domain>(ctx context.Context, in Create<Domain>Request) (*pgdb.<Domain>, error)
	update<Domain>(ctx context.Context, id string, in Update<Domain>Request) (*pgdb.<Domain>, error)
	delete<Domain>(ctx context.Context, id string) error
	list<Domain>s(ctx context.Context) ([]*pgdb.<Domain>, error)
	get<Domain>ByID(ctx context.Context, id string) (*pgdb.<Domain>, error)
}

type httpAdapter struct {
	svc <domain>Svc
	val app.Validator
}

func newHTTPAdapter(svc <domain>Svc, val app.Validator) *httpAdapter {
	return &httpAdapter{svc: svc, val: val}
}

// writeErr maps domain errors to HTTP responses and logs unexpected failures
// exactly once. Expected errors (e.g. ErrNotFound) log at debug; 5xx errors
// log at error with trace correlation from the request context.
func (m *httpAdapter) writeErr(r *http.Request, w http.ResponseWriter, err error) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	if errors.Is(err, ErrNotFound) {
		log.DebugContext(ctx, "<domain> not found", "error", err)
		router.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "<domain> not found"})
		return
	}
	log.ErrorContext(ctx, "<domain> request failed", "error", err)
	router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

// list<Domain>sHandler returns all <domain>s.
//
//	@Summary      List <domain>s
//	@Tags         <domain>s
//	@Produce      json
//	@Success      200  {array}   <domain>Response
//	@Failure      500  {object}  map[string]string
//	@Router       /<domain>s [get]
func (m *httpAdapter) list<Domain>sHandler(w http.ResponseWriter, r *http.Request) {
	items, err := m.svc.list<Domain>s(r.Context())
	if err != nil {
		m.writeErr(r, w, err)
		return
	}
	resp := make([]<domain>Response, 0, len(items))
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
//	@Success      201   {object}  <domain>Response
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
		m.writeErr(r, w, err)
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
//	@Success      200  {object}  <domain>Response
//	@Failure      404  {object}  map[string]string
//	@Failure      500  {object}  map[string]string
//	@Router       /<domain>s/{id} [get]
func (m *httpAdapter) get<Domain>ByIDHandler(w http.ResponseWriter, r *http.Request) {
	d, err := m.svc.get<Domain>ByID(r.Context(), r.PathValue("id"))
	if err != nil {
		m.writeErr(r, w, err)
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
//	@Success      200   {object}  <domain>Response
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
		m.writeErr(r, w, err)
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
		m.writeErr(r, w, err)
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
	"gokit/internal/app"
	router "gokit/pkg/http"
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
    <domain> "gokit/internal/<domain>"
)

// Inside r.Group("/v1", ...):
g.Group("/<domain>s", <domain>.NewModule(a).RegisterRoutes)
```

---

## Step 9 — Regenerate Swagger docs

```bash
go tool swag init -g cmd/http/main.go -o docs/
make check
```

---

## Checklist

- [ ] `pkg/postgres/migrations/<domain>.sql` — CREATE TABLE (TIMESTAMPTZ for timestamps)
- [ ] `pkg/postgres/queries/<domain>.sql` — CRUD queries with `$1, $2, ...` params
- [ ] `go tool sqlc generate` + `go build ./...`
- [ ] `internal/<domain>/domain.error.go` — one or more error sentinels
- [ ] `internal/<domain>/domain.dto.go` — Create/UpdateXRequest with validate + example tags
- [ ] `internal/<domain>/mapping.response.go` — `<domain>Response` + `ToResponse(pgdb.<Domain>)`
- [ ] `internal/<domain>/service.<domain>.go` — unexported service on `pgdb.Querier` + `SpanExpected/UnexpectedErr`
- [ ] `internal/<domain>/adapter.http.go` — `<domain>Svc` interface + `httpAdapter` + `writeErr(r, w, err)`
- [ ] `internal/<domain>/module.<domain>.go` — `Module` + `NewModule(a *app.App)` + `RegisterRoutes`
- [ ] `main.go` — add group wiring (import domain root package)
- [ ] `make swagger` + `make check`
