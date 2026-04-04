---
name: sqlite-config
description: SQLite connection setup, migration system, and configuration details for this project
user_invocable: false
---

Reference for the SQLite database layer in `pkg/sqlite/`.

## Connection Setup

**`pkg/sqlite/connection.go` — `OpenDB(ctx, path)`**

- Always use `OpenDB()` — never `sql.Open` directly
- Single connection mode: `MaxOpenConns(1)` — serializes access, prevents `SQLITE_BUSY`
- `PRAGMA busy_timeout=5000` set on every connection open
- WAL mode enabled for better read concurrency
- Driver: `modernc.org/sqlite` — pure Go, no CGO required

## Migration System

**`pkg/sqlite/migrate.go` — `Migrate(ctx, db)`**

- Uses `//go:embed migrations/*.sql` to bundle migration files
- Migration SQL files live in `pkg/sqlite/migrations/` (e.g., `user.sql`)
- Migrations called explicitly in `main()` after `OpenDB()`

## sqlc Code Generation

- Config: `sqlc.yaml` (v2 format)
- Query files: `pkg/sqlite/queries/<domain>.sql` — sqlc-annotated SQL
- Generated code: `pkg/sqlite/db/` (package `sqlitedb`, gitignored regenerated source)
- Run codegen: `make sqlc` or `go tool sqlc generate`

## Import Paths

```go
import (
    sqlitedb "restful-boilerplate/pkg/sqlite/db"  // sqlc-generated Queries
    pkgdb "restful-boilerplate/pkg/sqlite"         // OpenDB(), Migrate()
)
```

## Wiring Pattern

`*sqlitedb.Queries` is held by `internal/app/app.App` and passed to every module via `NewModule(a)`. Main never calls domain service constructors directly:

```go
// In cmd/http/main.go:
db, err := pkgdb.OpenDB(ctx, "./data.db")
// ...
if err := pkgdb.Migrate(ctx, db); err != nil { ... }

a := &app.App{
    Queries:   sqlitedb.New(db),
    Validator: v,
    Tracer:    otel.GetTracerProvider(),
}

r.Group("/v1", func(g *router.Group) {
    g.Prefix("/api")
    g.Group("/users", usermodule.NewModule(a).RegisterRoutes) // internal/user
})
```
