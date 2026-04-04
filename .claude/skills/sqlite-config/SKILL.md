---
name: sqlite-config
description: SQLite connection setup and migration reference — legacy layer kept for reference; PostgreSQL is now the primary database
user_invocable: false
---

> **Note:** SQLite is the legacy database layer. PostgreSQL is now primary. See `postgres-config` skill for current patterns.
> The SQLite code in `pkg/sqlite/` is retained for reference and the in-memory test driver.

Reference for the SQLite database layer in `pkg/sqlite/`.

## Connection Setup

**`pkg/sqlite/connection.go` — `OpenDB(ctx, path)`**

- Single connection mode: `MaxOpenConns(1)` — serializes access, prevents `SQLITE_BUSY`
- `PRAGMA busy_timeout=5000` set on every connection open
- WAL mode enabled for better read concurrency
- Driver: `modernc.org/sqlite` — pure Go, no CGO required

## Migration System

**`pkg/sqlite/migrate.go` — `Migrate(ctx, db)`**

- Uses `//go:embed migrations/*.sql` to bundle migration files
- Migration SQL files live in `pkg/sqlite/migrations/` (e.g., `user.sql`)

## sqlc Code Generation

- Config: `sqlc.yaml` (v2 format, SQLite block)
- Query files: `pkg/sqlite/queries/<domain>.sql` — use `?` placeholders
- Generated code: `pkg/sqlite/db/` (package `sqlitedb`, uses `database/sql`)
- Run codegen: `make sqlc` or `go tool sqlc generate`

## Import Paths

```go
import (
    sqlitedb "restful-boilerplate/pkg/sqlite/db"  // sqlc-generated Queries
    pkgdb    "restful-boilerplate/pkg/sqlite"      // OpenDB(), Migrate()
)
```
