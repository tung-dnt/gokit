---
name: postgres-config
description: PostgreSQL connection setup, migration system, and sqlc configuration for this project
user_invocable: false
---

Reference for the PostgreSQL database layer in `pkg/postgres/`.

## Connection Setup

**`pkg/postgres/connection.go` — `OpenDB(ctx, dsn)`**

- Always use `OpenDB()` — never `pgxpool.New` directly from application code
- Returns `*pgxpool.Pool` — connection pool suitable for concurrent HTTP workloads
- Pings on open; fails fast if PostgreSQL is unreachable
- Driver: `github.com/jackc/pgx/v5` — native pgx (no CGO required)

## Migration System

**`pkg/postgres/migrate.go` — `Migrate(ctx, pool)`**

- Uses `//go:embed migrations/*.sql` to bundle migration files
- Migration SQL files live in `pkg/postgres/migrations/` (e.g., `user.sql`)
- Migrations called explicitly in `main()` after `OpenDB()`
- Uses `IF NOT EXISTS` for idempotency

## sqlc Code Generation

- Config: `sqlc.yaml` (v2 format, PostgreSQL block)
- Query files: `pkg/postgres/queries/<domain>.sql` — sqlc-annotated SQL with `$1, $2, ...` params
- Generated code: `pkg/postgres/db/` (package `pgdb`)
- Run codegen: `make sqlc` or `go tool sqlc generate`
- `sql_package: "pgx/v5"` — uses pgx native types (not database/sql)

## Import Paths

```go
import (
    pgdb     "restful-boilerplate/pkg/postgres/db"     // sqlc-generated Queries
    postgres "restful-boilerplate/pkg/postgres"         // OpenDB(), Migrate()
)
```

## Wiring Pattern

`*pgdb.Queries` is held by `internal/app/app.App` and passed to every module via `NewModule(a)`:

```go
// In main.go:
pool, err := postgres.OpenDB(ctx, cfg.DatabaseURL)
// ...
if err := postgres.Migrate(ctx, pool); err != nil { ... }
defer pool.Close()

a := &app.App{
    Queries:   pgdb.New(pool),
    Validator: v,
    Tracer:    otel.GetTracerProvider(),
}
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable` | PostgreSQL DSN |
| `TEST_DATABASE_URL` | (empty — tests skip) | DSN for integration tests |

## pgx vs database/sql differences

| Aspect | database/sql (SQLite) | pgx/v5 (PostgreSQL) |
|--------|-----------------------|----------------------|
| No rows error | `sql.ErrNoRows` | `pgx.ErrNoRows` |
| Delete result | `sql.Result` (RowsAffected returns `int64, error`) | `pgconn.CommandTag` (RowsAffected returns `int64`) |
| Pool type | `*sql.DB` | `*pgxpool.Pool` |
| DBTX interface | database/sql | pgx native |

## Local Development

Start PostgreSQL via docker-compose:
```bash
cd infra && docker-compose up -d postgres
```

Default connection: `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`
