---
name: sqlite-config
description: SQLite connection setup, migration system, and configuration details for this project
user_invocable: false
---

Reference for the SQLite database layer in `infra/sqlite/`.

## Connection Setup

**`infra/sqlite/connection.go` — `OpenDB(ctx, path)`**

- Always use `OpenDB()` — never `sql.Open` directly
- Single connection mode: `MaxOpenConns(1)` — serializes access, prevents `SQLITE_BUSY`
- `PRAGMA busy_timeout=5000` set on every connection open
- WAL mode enabled for better read concurrency
- Driver: `modernc.org/sqlite` — pure Go, no CGO required

## Migration System

**`infra/sqlite/migrate.go` — `Migrate(ctx, db)`**

- Uses `//go:embed migrations/*.sql` to bundle migration files
- Migration SQL files live in `infra/sqlite/migrations/` (e.g., `user.sql`)
- Migrations run automatically in `OpenDB()`

## sqlc Code Generation

- Config: `sqlc.yaml` (v2 format)
- Query files: `infra/sqlite/queries/<domain>.sql` — sqlc-annotated SQL
- Generated code: `infra/sqlite/db/` (package `sqlitedb`, gitignored regenerated source)
- Run codegen: `make sqlc` or `go tool sqlc generate`

## Import Paths

```go
import (
    sqlitedb "restful-boilerplate/infra/sqlite/db"  // sqlc-generated Queries
    infradb "restful-boilerplate/infra/sqlite"       // OpenDB(), Migrate()
)
```

## Wiring Pattern (Clean Architecture)

```go
// In cmd/http/main.go:
userRepo := useradapter.NewSQLite(db)                            // adapter/user
userSvc := user.NewService(userRepo, otel.Tracer("user"))        // domain/user
r.Group("/users", useradapter.NewHandler(userSvc, v).RegisterRoutes) // adapter/user
```
