---
name: new-domain
description: Scaffold a new biz/<domain>/ module following the project's Echo v5 + sqlc modular monolith pattern
---

Scaffold a new domain module under `biz/<domain>/`. Use `biz/user/` as the reference implementation.

## Rules

- Only `Controller` is exported — all other types (service, handlers, model, inputs) are unexported (lowercase).
- `NewController(db *sql.DB) *Controller` — receives `*sql.DB`, wires the sqlc querier internally.
- Handlers live in `controller.go` as unexported methods on `*Controller`.
- Route registration lives in `route.go` (`type Controller struct` + `NewController` + `RegisterRoutes(g *echo.Group)`).
- DTOs with `validate` tags (go-playground/validator) live in `biz/<domain>/dto/dto.go`.
- All service/repository methods accept `ctx context.Context` as the first parameter.
- Wrap errors: `fmt.Errorf("<domain>Service.op: %w", err)`.
- IDs: `crypto/rand` 8-byte hex via `generateID()` helper in `service.go`.
- `sql.ErrNoRows` maps to `errNotFound` sentinel in `model.go`.

---

## Step 1 — SQL migration + queries

### `repo/sqlite/migrations/<domain>.sql`

```sql
CREATE TABLE IF NOT EXISTS <domain>s (
    id         TEXT     PRIMARY KEY NOT NULL,
    name       TEXT     NOT NULL,
    created_at DATETIME NOT NULL
);
```

### `repo/sqlite/queries/<domain>.sql`

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
go build ./...   # verify codegen compiles
```

This generates `repo/sqlite/db/<domain>.sql.go` with typed query methods.

---

## Step 2 — Domain files

### `biz/<domain>/model.go`

Domain entity, internal input structs, and sentinel error:

```go
package <domain>

import (
    "errors"
    "time"
)

var errNotFound = errors.New("<domain>: not found")

// <Domain> is the core domain entity.
type <Domain> struct {
    ID        string    `json:"id"         example:"a1b2c3d4"`
    Name      string    `json:"name"       example:"Alice"`
    CreatedAt time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
}

type create<Domain>Input struct {
    Name string
}

type update<Domain>Input struct {
    Name string
}
```

### `biz/<domain>/dto/dto.go`

Request DTOs with go-playground/validator tags and swag `example` tags:

```go
package dto

type Create<Domain>Request struct {
    Name string `json:"name" validate:"required,min=1,max=100" example:"Alice"`
}

type Update<Domain>Request struct {
    Name string `json:"name" validate:"omitempty,min=1,max=100" example:"Alice"`
}
```

### `biz/<domain>/service.go`

Business logic + ID generation using the sqlc `*sqlitedb.Queries` directly:

```go
package <domain>

import (
    "context"
    "crypto/rand"
    "database/sql"
    "encoding/hex"
    "errors"
    "fmt"
    "time"

    sqlitedb "restful-boilerplate/repo/sqlite/db"
)

type <domain>Service struct {
    q *sqlitedb.Queries
}

func (s *<domain>Service) create<Domain>(ctx context.Context, in create<Domain>Input) (*<Domain>, error) {
    id, err := generateID()
    if err != nil {
        return nil, fmt.Errorf("generate id: %w", err)
    }
    row, err := s.q.Create<Domain>(ctx, sqlitedb.Create<Domain>Params{
        ID: id, Name: in.Name, CreatedAt: time.Now().UTC(),
    })
    if err != nil {
        return nil, fmt.Errorf("create<Domain>: %w", err)
    }
    return to<Domain>(row), nil
}

func (s *<domain>Service) list<Domain>s(ctx context.Context) ([]*<Domain>, error) {
    rows, err := s.q.List<Domain>s(ctx)
    if err != nil {
        return nil, fmt.Errorf("list<Domain>s: %w", err)
    }
    out := make([]*<Domain>, len(rows))
    for i, r := range rows {
        out[i] = to<Domain>(r)
    }
    return out, nil
}

func (s *<domain>Service) get<Domain>ByID(ctx context.Context, id string) (*<Domain>, error) {
    row, err := s.q.Get<Domain>ByID(ctx, id)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, errNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("get<Domain>ByID: %w", err)
    }
    return to<Domain>(row), nil
}

func (s *<domain>Service) update<Domain>(ctx context.Context, id string, in update<Domain>Input) (*<Domain>, error) {
    existing, err := s.get<Domain>ByID(ctx, id)
    if err != nil {
        return nil, err
    }
    if in.Name != "" {
        existing.Name = in.Name
    }
    row, err := s.q.Update<Domain>(ctx, sqlitedb.Update<Domain>Params{
        ID: id, Name: existing.Name,
    })
    if errors.Is(err, sql.ErrNoRows) {
        return nil, errNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("update<Domain>: %w", err)
    }
    return to<Domain>(row), nil
}

func (s *<domain>Service) delete<Domain>(ctx context.Context, id string) error {
    result, err := s.q.Delete<Domain>(ctx, id)
    if err != nil {
        return fmt.Errorf("delete<Domain>: %w", err)
    }
    if n, _ := result.RowsAffected(); n == 0 {
        return errNotFound
    }
    return nil
}

func generateID() (string, error) {
    b := make([]byte, 8)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}

func to<Domain>(u sqlitedb.<Domain>) *<Domain> {
    return &<Domain>{ID: u.ID, Name: u.Name, CreatedAt: u.CreatedAt}
}
```

### `biz/<domain>/route.go`

Controller struct + DI constructor + Echo group route registration:

```go
// Package <domain> is the self-contained business domain for <domain> management.
package <domain>

import (
    "database/sql"

    "github.com/labstack/echo/v5"
    sqlitedb "restful-boilerplate/repo/sqlite/db"
)

// Controller is the only exported symbol in this package.
type Controller struct {
    svc *<domain>Service
}

func NewController(db *sql.DB) *Controller {
    return &Controller{svc: &<domain>Service{q: sqlitedb.New(db)}}
}

func (ctrl *Controller) RegisterRoutes(g *echo.Group) {
    g.GET("", ctrl.list<Domain>sHandler)
    g.POST("", ctrl.create<Domain>Handler)
    g.GET("/:id", ctrl.get<Domain>ByIDHandler)
    g.PUT("/:id", ctrl.update<Domain>Handler)
    g.DELETE("/:id", ctrl.delete<Domain>Handler)
}
```

### `biz/<domain>/controller.go`

HTTP handler methods with Swagger annotations:

```go
package <domain>

import (
    "errors"
    "net/http"

    "github.com/labstack/echo/v5"
    "restful-boilerplate/biz/<domain>/dto"
)

// list<Domain>sHandler returns all <domain>s.
//
//  @Summary      List <domain>s
//  @Tags         <domain>s
//  @Produce      json
//  @Success      200  {array}   <Domain>
//  @Failure      500  {object}  map[string]string
//  @Router       /<domain>s [get]
func (ctrl *Controller) list<Domain>sHandler(c *echo.Context) error {
    items, err := ctrl.svc.list<Domain>s(c.Request().Context())
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }
    return c.JSON(http.StatusOK, items)
}

// create<Domain>Handler creates a new <domain>.
//
//  @Summary      Create <domain>
//  @Tags         <domain>s
//  @Accept       json
//  @Produce      json
//  @Param        body  body      dto.Create<Domain>Request  true  "<Domain> data"
//  @Success      201   {object}  <Domain>
//  @Failure      400   {object}  map[string]string
//  @Failure      422   {object}  map[string]string
//  @Failure      500   {object}  map[string]string
//  @Router       /<domain>s [post]
func (ctrl *Controller) create<Domain>Handler(c *echo.Context) error {
    var req dto.Create<Domain>Request
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
    }
    if err := c.Validate(&req); err != nil {
        return err
    }
    item, err := ctrl.svc.create<Domain>(c.Request().Context(), create<Domain>Input{Name: req.Name})
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }
    return c.JSON(http.StatusCreated, item)
}

// get<Domain>ByIDHandler gets a <domain> by ID.
//
//  @Summary      Get <domain> by ID
//  @Tags         <domain>s
//  @Produce      json
//  @Param        id   path      string  true  "<Domain> ID"
//  @Success      200  {object}  <Domain>
//  @Failure      404  {object}  map[string]string
//  @Failure      500  {object}  map[string]string
//  @Router       /<domain>s/{id} [get]
func (ctrl *Controller) get<Domain>ByIDHandler(c *echo.Context) error {
    item, err := ctrl.svc.get<Domain>ByID(c.Request().Context(), c.Param("id"))
    if err != nil {
        if errors.Is(err, errNotFound) {
            return c.JSON(http.StatusNotFound, map[string]string{"error": "<domain> not found"})
        }
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }
    return c.JSON(http.StatusOK, item)
}

// update<Domain>Handler updates a <domain>.
//
//  @Summary      Update <domain>
//  @Tags         <domain>s
//  @Accept       json
//  @Produce      json
//  @Param        id    path      string                     true  "<Domain> ID"
//  @Param        body  body      dto.Update<Domain>Request  true  "<Domain> data"
//  @Success      200   {object}  <Domain>
//  @Failure      400   {object}  map[string]string
//  @Failure      404   {object}  map[string]string
//  @Failure      500   {object}  map[string]string
//  @Router       /<domain>s/{id} [put]
func (ctrl *Controller) update<Domain>Handler(c *echo.Context) error {
    var req dto.Update<Domain>Request
    if err := c.Bind(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
    }
    if err := c.Validate(&req); err != nil {
        return err
    }
    item, err := ctrl.svc.update<Domain>(c.Request().Context(), c.Param("id"), update<Domain>Input{Name: req.Name})
    if err != nil {
        if errors.Is(err, errNotFound) {
            return c.JSON(http.StatusNotFound, map[string]string{"error": "<domain> not found"})
        }
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }
    return c.JSON(http.StatusOK, item)
}

// delete<Domain>Handler deletes a <domain>.
//
//  @Summary      Delete <domain>
//  @Tags         <domain>s
//  @Produce      json
//  @Param        id   path      string  true  "<Domain> ID"
//  @Success      204
//  @Failure      404  {object}  map[string]string
//  @Failure      500  {object}  map[string]string
//  @Router       /<domain>s/{id} [delete]
func (ctrl *Controller) delete<Domain>Handler(c *echo.Context) error {
    if err := ctrl.svc.delete<Domain>(c.Request().Context(), c.Param("id")); err != nil {
        if errors.Is(err, errNotFound) {
            return c.JSON(http.StatusNotFound, map[string]string{"error": "<domain> not found"})
        }
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }
    return c.NoContent(http.StatusNoContent)
}
```

---

## Step 3 — Wire into cmd/http/main.go

In `cmd/http/main.go`, add to `registerRouters`:

```go
func registerRouters(g *echo.Group, db *sql.DB) {
    user.NewController(db).RegisterRoutes(g.Group("/users"))
    <domain>.NewController(db).RegisterRoutes(g.Group("/<domain>s")) // add this
}
```

Also add the import at the top:
```go
"restful-boilerplate/biz/<domain>"
```

---

## Step 4 — Regenerate Swagger docs

```bash
go tool swag init -g cmd/http/main.go -o dx/docs/
go build ./...
```

---

## Checklist

- [ ] `repo/sqlite/migrations/<domain>.sql` — CREATE TABLE
- [ ] `repo/sqlite/queries/<domain>.sql` — CRUD queries with sqlc annotations
- [ ] `go tool sqlc generate` — generates `repo/sqlite/db/<domain>.sql.go`
- [ ] `biz/<domain>/model.go` — entity + input types + errNotFound
- [ ] `biz/<domain>/dto/dto.go` — request DTOs with validate + example tags
- [ ] `biz/<domain>/service.go` — CRUD + generateID() + toXxx() mapper
- [ ] `biz/<domain>/route.go` — Controller struct + NewController(db) + RegisterRoutes
- [ ] `biz/<domain>/controller.go` — handler methods with swag annotations
- [ ] `cmd/http/main.go` — add to registerRouters
- [ ] `make swagger` — regenerate Swagger docs
- [ ] `make check` — fmt + vet + lint + test all pass
