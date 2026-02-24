---
name: sqlite-config
description: SQLite connection setup, migration system, and configuration details for this project
user_invocable: false
---

Reference for the SQLite database layer in `repo/sqlite/`.

## Connection Setup

**`repo/sqlite/db.go` — `OpenDB(ctx, path)`**

- Always use `OpenDB()` — never `sql.Open` directly
- Single connection mode: `MaxOpenConns(1)` — serializes access, prevents `SQLITE_BUSY`
- `PRAGMA busy_timeout=5000` set on every connection open
- WAL mode enabled for better read concurrency
- Driver: `modernc.org/sqlite` — pure Go, no CGO required

## Migration System

**`repo/sqlite/migrate.go` — `Migrate(ctx, db)`**

- Uses `//go:embed migrations/*.sql` to bundle migration files
- Migration SQL files live in `repo/sqlite/migrations/` (e.g., `user.sql`)
- Run migrations: `make migrate`

## sqlc Code Generation

- Config: `sqlc.yaml` (v2 format)
- Query files: `repo/sqlite/queries/<domain>.sql` — sqlc-annotated SQL
- Generated code: `repo/sqlite/db/` (gitignored regenerated source)
- Run codegen: `make sqlc` or `go tool sqlc generate`

## Import Paths

```go
import (
    sqlitedb "restful-boilerplate/repo/sqlite/db"  // sqlc-generated Queries
    "restful-boilerplate/repo/sqlite"               // OpenDB(), Migrate()
)
```

## Wiring Pattern

```go
// In NewController:
func NewController(db *sql.DB) *Controller {
    return &Controller{
        svc: &xxxService{q: sqlitedb.New(db)},
    }
}
```
