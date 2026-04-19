---
name: postgres-config
description: PostgreSQL connection setup, migration system, and sqlc configuration for this project
user_invocable: false
---

Reference for the PostgreSQL database layer in `pkg/postgres/`.

## Connection Setup

**`pkg/postgres/connection.go` — `OpenDB(ctx, dsn)`**

- Always use `OpenDB()` — never `pgxpool.New` / `pgxpool.NewWithConfig` directly from application code.
- Returns `*pgxpool.Pool` — connection pool suitable for concurrent HTTP workloads.
- Pings on open; fails fast if PostgreSQL is unreachable.
- Driver: `github.com/jackc/pgx/v5` — native pgx (no CGO).

### otelpgx instrumentation (built in)

`OpenDB` wires `github.com/exaring/otelpgx` into the pool. No extra config in callers.

```go
cfg.ConnConfig.Tracer = otelpgx.NewTracer(
    otelpgx.WithTrimSQLInSpanName(),
    otelpgx.WithDisableSQLStatementInAttributes(),
)
pool, err := pgxpool.NewWithConfig(ctx, cfg)
// ...pool.Ping(ctx)
otelpgx.RecordStats(pool)
```

What you get for free:

- One span per query (`pgx.query.*`), child of the active HTTP span.
- SQL statement is stripped from span names (`WithTrimSQLInSpanName`) and not copied into attributes (`WithDisableSQLStatementInAttributes`) — keeps traces readable and avoids leaking query text.
- `otelpgx.RecordStats(pool)` publishes pool gauges as OTEL metrics: `acquired_conns`, `idle_conns`, `wait_duration`, etc. Visible in SigNoz alongside HTTP metrics.

No code inside services or handlers needs to reference `otelpgx`.

## Migration System

**`pkg/postgres/migrate.go` — `Migrate(ctx, pool)`**

- Uses `//go:embed migrations/*.sql` to bundle SQL files at build time.
- Migration files live in `pkg/postgres/migrations/` (e.g., `user.sql`).
- Called explicitly in `main()` after `OpenDB()`.
- Uses `IF NOT EXISTS` for idempotency.
- Run standalone via `make migrate` (which runs `go run ./cmd/postgres-migration`).

## sqlc Code Generation

- Config: `sqlc.yaml` (v2 format, PostgreSQL block).
- Query files: `pkg/postgres/queries/<domain>.sql` — sqlc-annotated SQL with `$1, $2, ...` params.
- Generated code: `pkg/postgres/db/` (package `pgdb`).
- Run codegen: `make sqlc` (wraps `go tool sqlc generate`).
- `sql_package: "pgx/v5"` — uses pgx native types (not `database/sql`).
- `emit_interface: true` — generates the `pgdb.Querier` interface (see below).

For authoring queries see the `new-sqlc-query` skill.

## The `pgdb.Querier` interface (important for testing)

`sqlc.yaml` enables `emit_interface: true`, so `pkg/postgres/db/querier.go` exports a `Querier` interface covering every generated method:

```go
type Querier interface {
    CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
    DeleteUser(ctx context.Context, id string) (pgconn.CommandTag, error)
    GetUserByID(ctx context.Context, id string) (User, error)
    ListUsers(ctx context.Context) ([]User, error)
    UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error)
}

var _ Querier = (*Queries)(nil)
```

Consequences:

- **Services depend on `pgdb.Querier` (interface), NOT `*pgdb.Queries` (concrete).**
  ```go
  type userService struct {
      db     pgdb.Querier
      tracer trace.Tracer
  }
  func newUserService(q pgdb.Querier, tracer trace.Tracer) *userService { ... }
  ```
- `*pgdb.Queries` satisfies the interface, so production wiring is unchanged.
- Unit tests pass a mock (see `internal/user/mock_test.go`):
  ```go
  type mockQuerier struct{ mock.Mock }
  var _ pgdb.Querier = (*mockQuerier)(nil)
  ```
- When adding a new query via sqlc, the interface grows automatically. Every mock must implement the new method or `var _ pgdb.Querier = (*mockQuerier)(nil)` will fail to compile — this is the intended guardrail.
- `internal/app/app.App.Queries` is still `*pgdb.Queries` (concrete). The interface boundary lives at the service constructor, not the app container.

See the `gen-test` skill for the mock pattern.

## Import Paths

```go
import (
    postgres "gokit/pkg/postgres"    // OpenDB(), Migrate()
    pgdb     "gokit/pkg/postgres/db" // Querier, Queries, User, ...
)
```

## Wiring Pattern

`*pgdb.Queries` is held by `internal/app/app.App` and passed to every module via `NewModule(a)`:

```go
// In cmd/http/main.go:
pool, err := postgres.OpenDB(ctx, cfg.DatabaseURL)
// ...
if err := postgres.Migrate(ctx, pool); err != nil { ... }
defer pool.Close()

a := &app.App{
    Queries:   pgdb.New(pool), // *pgdb.Queries (concrete)
    Validator: v,
    Tracer:    otel.GetTracerProvider(),
}
```

Inside `NewModule(a)`, the concrete `a.Queries` is handed to `newXService(q pgdb.Querier, ...)` — Go widens it to the interface at the call site.

## Environment Variables

| Variable            | Default                                                                    | Description                                     |
|---------------------|----------------------------------------------------------------------------|-------------------------------------------------|
| `DATABASE_URL`      | `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`     | PostgreSQL DSN                                  |
| `TEST_DATABASE_URL` | (empty — integration tests skip)                                           | DSN for `pkg/testutil.SetupPgTestDB` tests      |

## pgx vs database/sql differences

| Aspect          | database/sql (SQLite)                         | pgx/v5 (PostgreSQL)                             |
|-----------------|-----------------------------------------------|-------------------------------------------------|
| No rows error   | `sql.ErrNoRows`                               | `pgx.ErrNoRows`                                 |
| Delete result   | `sql.Result` (`RowsAffected` returns `int64, error`) | `pgconn.CommandTag` (`RowsAffected` returns `int64` — no error) |
| Pool type       | `*sql.DB`                                     | `*pgxpool.Pool`                                 |
| Param syntax    | `?`                                           | `$1, $2, ...`                                   |
| Timestamp type  | `DATETIME`                                    | `TIMESTAMPTZ`                                   |

Constructing fake command tags in tests:

```go
import "github.com/jackc/pgx/v5/pgconn"

tag := pgconn.NewCommandTag("DELETE 1")
// tag.RowsAffected() == 1
```

## Local Development

Start PostgreSQL via docker-compose:

```bash
cd infra && docker-compose up -d postgres
```

Default connection: `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`
