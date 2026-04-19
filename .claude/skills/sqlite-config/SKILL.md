---
name: sqlite-config
description: Legacy SQLite reference — the project uses PostgreSQL now. Kept for historical context only.
user_invocable: false
---

This project has migrated to PostgreSQL. All new features, queries, and migrations must use the PostgreSQL layer — see the `postgres-config` skill for current patterns (pgx/v5, sqlc codegen, `$1` params, `TIMESTAMPTZ` schema, `pgxpool` connection).

The `pkg/sqlite/` directory is retained in the tree and `modernc.org/sqlite` is still in `go.mod`, but nothing in the main server wires it up. Reference the SQLite code only when researching history — not when adding new features.
