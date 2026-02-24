# CLAUDE.md — restful-boilerplate

## Project Overview

A Go RESTful API boilerplate built on **Echo v5 + SQLite + sqlc** with full observability (OpenTelemetry, Tempo, Loki, Grafana). Demonstrates idiomatic Go patterns for a modular monolith.

**Module:** `restful-boilerplate` | **Go:** 1.26.0 | **Deps:** Echo v5, modernc/sqlite, go-playground/validator, swaggo/swag

**Dev tools (via `go tool`):** `sqlc generate`, `swag init`, `air`, `golangci-lint`

---

## Directory Structure

```
biz/
  <domain>/              → One folder per business domain
    route.go             → Controller struct + NewController(db) + RegisterRoutes
    controller.go        → HTTP handler methods (unexported) + swag annotations
    model.go             → Domain entity + input types + errNotFound sentinel
    service.go           → Business logic + CRUD + generateID()
    dto/dto.go           → Request DTOs with validate + example tags
cmd/
  http/main.go           → Echo server entrypoint + registerRouters()
pkg/
  config/config.go       → Env-var config loader
  logger/logger.go       → slog MultiWriter (stdout + ./logs/app.log)
  metrics/metrics.go     → Prometheus metrics
  middleware/            → Request logger middleware
  otelecho/middleware.go → Custom Echo v5 OTEL middleware
  telemetry/             → OTLP TracerProvider setup
  validator/validator.go → Echo Validator adapter (go-playground/validator)
repo/sqlite/
  db/                    → sqlc-generated Go code (gitignored source)
  migrations/            → SQL CREATE TABLE files + migration runner (main.go)
  queries/               → sqlc-annotated SQL query files
dx/
  deploy/                → Docker Compose for observability stack (Tempo, Loki, Alloy, Grafana)
  docs/                  → swag-generated OpenAPI docs (gitignored)
  scripts/               → k6 performance test script
  test/                  → Integration test helpers
sqlc.yaml                → sqlc v2 config
.golangci.yml            → golangci-lint config (23 linters, 5m timeout)
Makefile                 → All dev tasks
```

---

## Architecture: Modular Monolith

Each domain under `biz/` is fully self-contained. **`Controller` is the only exported symbol** per domain package — all other types are package-private.

```
cmd/http/main.go
    └── registerRouters(g *echo.Group, db *sql.DB)
            └── user.NewController(db).RegisterRoutes(g.Group("/users"))
                    └── &userService{q: sqlitedb.New(db)}
```

To add a new domain: run `/new-domain` — it creates all 5 files and wires into `cmd/http/main.go`.

---

## Key Patterns

- **Single export rule:** `Controller` is the only exported type per domain. Everything else uses lowercase names (unexported).
- **DI via NewController:** `NewController(db *sql.DB) *Controller` wires the sqlc querier internally. No global state.
- **Handler pipeline:** `c.Bind` → `c.Validate` → service call → `c.JSON`. Return 400 for bind errors, 422 for validation (auto-handled by `pkg/validator`), 404 for not-found, 500 for unexpected.
- **Validation:** go-playground/validator tags on DTOs (`validate:"required,min=1,max=100"`). NOT manual `Valid()` method. `c.Validate(&req)` triggers automatic 422 response via `pkg/validator`.
- **Service errors:** Wrap with `fmt.Errorf("opName: %w", err)`. Map `sql.ErrNoRows` → `errNotFound` sentinel in model.go. Use `errors.Is()` — never `==`.
- **ID generation:** `generateID()` helper in service.go — `crypto/rand` 8-byte hex.
- **Context:** All service/repo methods accept `ctx context.Context` as first parameter.
- **Structured logging:** `pkg/logger` — slog with JSON output to stdout + `./logs/app.log`. Inject via `logger.FromContext(ctx)` to get trace-correlated logger.
- **OTEL tracing:** `pkg/telemetry/otel.go` — OTLP HTTP. Store tracer as `tracer trace.Tracer` struct field (not global). Use `pkg/otelecho` middleware for Echo v5.

---

## Development Commands

```bash
# Hot-reload dev server
make dev

# Run server (no hot-reload)
make run

# Apply DB migrations (runner is repo/sqlite/migrations/main.go)
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
make swagger   # go tool swag init -g cmd/http/main.go -o dx/docs/

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
