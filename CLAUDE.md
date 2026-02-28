# CLAUDE.md — restful-boilerplate

## Project Overview

A Go RESTful API boilerplate built on **Echo v5 + SQLite + sqlc** with full observability (OpenTelemetry, Tempo, Loki, Grafana). Uses Clean Architecture with strict dependency rules: domain → nothing, app → domain only, infra → domain + app.

**Module:** `restful-boilerplate` | **Go:** 1.26.0 | **Deps:** Echo v5, modernc/sqlite, go-playground/validator, swaggo/swag

**Dev tools (via `go tool`):** `sqlc generate`, `swag init`, `air`, `golangci-lint`

---

## Directory Structure

```
domain/
  <domain>/                → Pure domain types (zero framework imports)
    entity.go              → Exported entity + input types
    errors.go              → Exported sentinel errors (e.g. ErrNotFound)
    port.go                → Repository interface

app/
  <domain>svc/             → Application/use-case layer
    service.go             → Service + CRUD + generateID() + OTEL tracing
    service_test.go

infra/
  http/<domain>hdl/        → HTTP adapter (Echo handlers)
    handler.go             → HTTP handler methods + swag annotations
    routes.go              → Handler struct + NewHandler + RegisterRoutes
    dto.go                 → Request DTOs with validate + example tags
    handler_test.go
    dto_test.go
  sqlite/                  → SQLite connection + migration infra
    connection.go          → OpenDB() — single conn + WAL + busy_timeout
    migrate.go             → Migrate() with //go:embed
    db/                    → sqlc-generated Go code (package sqlitedb)
    queries/               → sqlc-annotated SQL query files
    migrations/            → SQL CREATE TABLE files
  sqlite/<domain>repo/     → Repository adapter (implements domain port)
    repository.go          → SQLite adapter + mapper + ErrNoRows→ErrNotFound

  config/                  → Env-var config loader
  logger/                  → slog MultiWriter (stdout + ./logs/app.log)
  metrics/                 → Prometheus metrics
  middleware/              → Request logger middleware
  otelecho/                → Custom Echo v5 OTEL middleware
  telemetry/               → OTLP TracerProvider setup
  validator/               → Echo Validator adapter (go-playground/validator)
  testutil/                → Shared test helpers (SetupTestDB)

cmd/
  http/main.go             → Echo server entrypoint + registerRouters() (explicit DI)
  migrate/main.go          → DB migration runner
docs/                      → swag-generated OpenAPI docs (gitignored)
deploy/                    → Docker Compose for observability stack
scripts/                   → k6 performance test script
sqlc.yaml                  → sqlc v2 config
.golangci.yml              → golangci-lint config
Makefile                   → All dev tasks
```

---

## Architecture: Clean Architecture

Strict layered architecture with dependency rule enforcement:

```
domain/user         → nothing (pure Go types + interfaces)
app/usersvc         → domain/user only
infra/http/userhdl  → domain/user + app/usersvc
infra/sqlite/userrepo → domain/user + infra/sqlite/db
```

**Wiring in cmd/http/main.go:**
```
registerRouters(g *echo.Group, db *sql.DB)
    └── userRepo := userrepo.NewSQLite(db)
    └── userSvc  := usersvc.NewService(userRepo, otel.Tracer("user"))
    └── userhdl.NewHandler(userSvc).RegisterRoutes(g.Group("/users"))
```

To add a new domain: run `/new-domain` — it creates all files across the three layers and wires into `cmd/http/main.go`.

---

## Key Patterns

- **Domain layer:** Pure Go types. `User`, `CreateUserInput`, `UpdateUserInput` are exported. `ErrNotFound` sentinel. `Repository` interface defines the port.
- **Service layer:** Depends on `user.Repository` interface (not sqlc types). All methods exported: `CreateUser`, `ListUsers`, etc. OTEL tracing lives here.
- **Handler layer:** `Handler` wraps `*usersvc.Service`. Maps `user.ErrNotFound` to HTTP 404.
- **Repository adapter:** `userrepo.SQLite` implements `user.Repository`. Maps `sql.ErrNoRows` → `user.ErrNotFound`.
- **Handler pipeline:** `c.Bind` → `c.Validate` → service call → `c.JSON`. Return 400 for bind errors, 422 for validation (auto-handled by `infra/validator`), 404 for not-found, 500 for unexpected.
- **Validation:** go-playground/validator tags on DTOs (`validate:"required,min=1,max=100"`). NOT manual `Valid()` method.
- **Service errors:** Wrap with `fmt.Errorf("opName: %w", err)`. Use `errors.Is()` — never `==`.
- **ID generation:** `generateID()` helper in service — `crypto/rand` 8-byte hex.
- **Context:** All service/repo methods accept `ctx context.Context` as first parameter.
- **Structured logging:** `infra/logger` — slog with JSON output. `logger.FromContext(ctx)` for trace-correlated logger.
- **OTEL tracing:** Store tracer as struct field (not global). Use `infra/otelecho` middleware for Echo v5.

---

## Development Commands

```bash
# Hot-reload dev server
make dev

# Run server (no hot-reload)
make run

# Apply DB migrations
make migrate

# Full quality gate (fmt + vet + lint + test) — also the post-edit hook
make check

# Individual quality steps
make fmt       # gofmt -w .
make vet       # go vet ./...
make lint      # golangci-lint run ./...
make test      # go test ./...

# Code generation
make sqlc      # go tool sqlc generate
make swagger   # go tool swag init -g cmd/http/main.go -o docs/

# Build binaries
make build

# Observability stack (Tempo + Loki + Alloy + Grafana + Prometheus)
make obs/up
make obs/down
```

---

## Stack-Specific Reference Skills

These skills provide detailed patterns on demand — invoke when working in these areas:

- **`echo-handler-patterns`** — Handler pipeline, error responses, validation patterns
- **`sqlite-config`** — Connection setup, migrations, sqlc codegen, import paths
- **`observability`** — OTEL tracing, structured logging, metrics, Grafana stack
