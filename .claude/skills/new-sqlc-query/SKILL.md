---
name: new-sqlc-query
description: Add a new SQL query to an existing domain's PostgreSQL query file and regenerate sqlc Go code
---

Add a new named SQL query to `pkg/postgres/queries/<domain>.sql` and run `go tool sqlc generate` to produce the corresponding Go method in `pkg/postgres/db/<domain>.sql.go`.

## sqlc annotation syntax

Every query needs a name and return type annotation:

| Annotation | Return type | Use case |
|------------|-------------|----------|
| `:one` | single row struct | SELECT by ID, INSERT RETURNING, UPDATE RETURNING |
| `:many` | `[]Row` slice | SELECT list/filtered results |
| `:exec` | `error` | DELETE/UPDATE with no return |
| `:execresult` | `pgconn.CommandTag, error` | DELETE/UPDATE when you need RowsAffected |

**PostgreSQL parameters use `$1, $2, ...` (not `?`).**

## Query template

```sql
-- name: <QueryName> :<annotation>
SELECT ...
FROM <table>
WHERE ...;
```

## Common patterns

### Filter/search
```sql
-- name: ListUsersByName :many
SELECT * FROM users
WHERE name ILIKE '%' || $1 || '%'
ORDER BY created_at ASC;
```

### Paginated list
```sql
-- name: ListUsersPaginated :many
SELECT * FROM users
ORDER BY created_at ASC
LIMIT $1 OFFSET $2;
```

### Count
```sql
-- name: CountUsers :one
SELECT COUNT(*) FROM users;
```

### Soft delete (add `deleted_at` column first)
```sql
-- name: SoftDeleteUser :exec
UPDATE users SET deleted_at = $1 WHERE id = $2;
```

### Upsert
```sql
-- name: UpsertUser :one
INSERT INTO users (id, name, email, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name
RETURNING *;
```

## Workflow

1. Add the query to `pkg/postgres/queries/<domain>.sql`
2. Run:
   ```bash
   go tool sqlc generate
   make check
   ```
3. The generated method appears in `pkg/postgres/db/<domain>.sql.go` AND is added to the `pgdb.Querier` interface in `pkg/postgres/db/querier.go` (because `emit_interface: true` in `sqlc.yaml`).
4. Call it from the service `internal/<domain>/service.<domain>.go` via `s.db.<QueryName>(ctx, ...)` — services depend on `pgdb.Querier`, not `*pgdb.Queries`.
5. **Every mock `Querier` in tests (e.g. `internal/user/mock_test.go`) must implement the new method** or the compile-time check `var _ pgdb.Querier = (*mockQuerier)(nil)` will fail. Keep return types plain (`pgdb.<Type>`, `[]pgdb.<Type>`, `pgconn.CommandTag`, `error`) so mocks stay trivial — see the `gen-test` skill.

## If adding a new column (migration required)

1. Add migration to `pkg/postgres/migrations/<domain>.sql`:
   ```sql
   ALTER TABLE <domain>s ADD COLUMN IF NOT EXISTS <field> TEXT;
   ```
   Or for a fresh project, edit the `CREATE TABLE` directly.
2. Re-run `go tool sqlc generate` to pick up schema changes.
3. Migrations run automatically on app startup (`postgres.Migrate` in main).

## sqlc.yaml reference (project PostgreSQL block)

```yaml
- engine: "postgresql"
  queries: "pkg/postgres/queries"
  schema: "pkg/postgres/migrations"
  gen:
    go:
      package: "pgdb"
      out: "pkg/postgres/db"
      sql_package: "pgx/v5"
      overrides:
        - db_type: "timestamptz"
          go_type: "time.Time"
```

## pgx vs database/sql differences for service code

```go
// DeleteUser with pgconn.CommandTag (no error from RowsAffected):
result, err := s.q.DeleteUser(ctx, id)
if err != nil { ... }
if result.RowsAffected() == 0 {
    return ErrNotFound
}

// GetByID — use pgx.ErrNoRows not sql.ErrNoRows:
import "github.com/jackc/pgx/v5"
if errors.Is(err, pgx.ErrNoRows) {
    return nil, ErrNotFound
}
```
