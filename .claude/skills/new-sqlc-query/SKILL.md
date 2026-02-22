---
name: new-sqlc-query
description: Add a new SQL query to an existing domain's query file and regenerate sqlc Go code
---

Add a new named SQL query to `repo/sqlite/queries/<domain>.sql` and run `go tool sqlc generate` to produce the corresponding Go method in `repo/sqlite/db/<domain>.sql.go`.

## sqlc annotation syntax

Every query needs a name and return type annotation:

| Annotation | Return type | Use case |
|------------|-------------|----------|
| `:one` | single row struct | SELECT by ID, INSERT RETURNING, UPDATE RETURNING |
| `:many` | `[]Row` slice | SELECT list/filtered results |
| `:exec` | `error` | DELETE/UPDATE with no return |
| `:execresult` | `sql.Result, error` | DELETE/UPDATE when you need RowsAffected |

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
WHERE name LIKE '%' || ? || '%'
ORDER BY created_at ASC;
```

### Paginated list
```sql
-- name: ListUsersPaginated :many
SELECT * FROM users
ORDER BY created_at ASC
LIMIT ? OFFSET ?;
```

### Count
```sql
-- name: CountUsers :one
SELECT COUNT(*) FROM users;
```

### Soft delete (add `deleted_at` column first)
```sql
-- name: SoftDeleteUser :exec
UPDATE users SET deleted_at = ? WHERE id = ?;
```

### Upsert
```sql
-- name: UpsertUser :one
INSERT INTO users (id, name, email, created_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(email) DO UPDATE SET name = excluded.name
RETURNING *;
```

## Workflow

1. Add the query to `repo/sqlite/queries/<domain>.sql`
2. Run:
   ```bash
   go tool sqlc generate
   make check
   ```
3. The generated method appears in `repo/sqlite/db/<domain>.sql.go`
4. Import and call it from `biz/<domain>/service.go` via `s.q.<QueryName>(ctx, ...)`

## If adding a new column (migration required)

1. Alter `repo/sqlite/migrations/<domain>.sql` — add the column:
   ```sql
   ALTER TABLE <domain>s ADD COLUMN <field> TEXT;
   ```
   Or for a fresh project, edit the `CREATE TABLE` directly.
2. Re-run `go tool sqlc generate` to pick up schema changes.
3. Re-run migrations: `go run ./cmd/migrate`.
4. Update `biz/<domain>/model.go` entity struct and `to<Domain>()` mapper in `service.go`.

## sqlc.yaml reference (project config)

```yaml
version: "2"
sql:
  - engine: sqlite
    queries: repo/sqlite/queries
    schema: repo/sqlite/migrations
    gen:
      go:
        package: sqlite
        out: repo/sqlite/db
        overrides:
          - db_type: DATETIME
            go_type: time.Time
```
