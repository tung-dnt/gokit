# CLAUDE.md — restful-boilerplate

## Project Overview

A Go RESTful API boilerplate built on **net/http (stdlib) + SQLite + sqlc** with full observability (OpenTelemetry, Tempo, Loki, Grafana). Uses Clean Architecture with strict dependency rules: domain → nothing, adapter → domain only, infra → domain + adapter.

**Module:** `restful-boilerplate` | **Go:** 1.26.0 | **Deps:** modernc/sqlite, go-playground/validator, swaggo/swag

**Dev tools (via `go tool`):** `sqlc generate`, `swag init`, `air`, `golangci-lint`

---

## Directory Structure

```
domain/
  <domain>/                → Pure domain types + service (zero framework imports)
    entity.go              → Exported entity + input types
    errors.go              → Exported sentinel errors (e.g. ErrNotFound)
    port.go                → Repository interface
    service.go             → Service (UserSvc) + CRUD + generateID() + OTEL tracing
    service_test.go        → External test package (package <domain>_test)
    internal_test.go       → Same-package tests for unexported helpers

adapter/
  <domain>/                → HTTP + persistence adapters
    handler.go             → HTTP handler methods (net/http HandlerFunc)
    module.go              → Module struct + NewModule + RegisterRoutes
    dto.go                 → Request DTOs with validate + example tags
    dto_test.go
    handler_test.go
    repository.go          → SQLite adapter (implements domain port)

infra/
  http/                    → Router wrapper for net/http ServeMux
    router.go              → Router struct + NewRouter + Prefix + Use + Group + Route
    group.go               → Group struct + Use (group middleware) + GET/POST/PUT/PATCH/DELETE/ANY + nested Group
    util.go                → WriteJSON helper
    serve.go               → GracefulServe (graceful shutdown)
  sqlite/                  → SQLite connection + migration infra
    connection.go          → OpenDB() — single conn + WAL + busy_timeout
    migrate.go             → Migrate() with //go:embed
    db/                    → sqlc-generated Go code (package sqlitedb)
    queries/               → sqlc-annotated SQL query files
    migrations/            → SQL CREATE TABLE files
  config/                  → Env-var config loader
  logger/                  → slog MultiWriter (stdout + ./logs/app.log)
  metrics/                 → Prometheus metrics (promhttp)
  middleware/              → Request logger + recovery middleware (net/http)
  otelhttp/                → Custom net/http OTEL middleware
  telemetry/               → OTLP TracerProvider setup
  validator/               → Validator adapter (go-playground/validator)
  testutil/                → Shared test helpers (SetupTestDB)

cmd/
  http/main.go             → net/http server entrypoint (explicit DI, inline wiring)
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
domain/user         → nothing (pure Go types + interfaces + service)
adapter/user        → domain/user + infra/sqlite/db + infra/http + infra/validator
infra/http          → net/http stdlib only
infra/sqlite        → database/sql + modernc/sqlite
```

**Wiring in cmd/http/main.go:**
```go
r := router.NewRouter()
r.Use(otelhttp.Middleware("restful-boilerplate"))
r.Use(logger.Middleware)
r.Use(recovery.Middleware)
r.GET("/metrics", metric.Handler())

r.Group("/v1", func(g *router.Group) {
    g.Prefix("/api")
    g.ANY("/swagger/", httpSwagger.WrapHandler)

    // User domain: db → repo → service → module → group
    userRepo := useradapter.NewSQLite(db)
    userSvc  := user.NewService(userRepo, otel.Tracer("user"))
    g.Group("/users", useradapter.NewModule(userSvc, v).RegisterRoutes)
})
```

To add a new domain: run `/new-domain` — it creates all files across the layers and wires into `cmd/http/main.go`.

---

## Key Patterns

- **Domain layer:** Pure Go types. `User`, `CreateUserInput`, `UpdateUserInput` are exported. `ErrNotFound` sentinel. `Repository` interface defines the port. `UserSvc` service type is exported.
- **Service layer:** Lives in `domain/<domain>/service.go`. Depends on `Repository` interface (not sqlc types). All methods exported: `CreateUser`, `ListUsers`, etc. OTEL tracing lives here.
- **Handler layer:** `Module` struct in `adapter/<domain>/module.go` wraps `*user.Svc` + `Validator`. `NewModule(svc, v)` constructor. Maps `user.ErrNotFound` to HTTP 404.
- **Repository adapter:** `adapter/<domain>/repository.go` — `SQLite` struct implements `user.Repository`. Maps `sql.ErrNoRows` → `user.ErrNotFound`.
- **Handler pipeline:** `json.NewDecoder(r.Body).Decode(&req)` → `h.val.Validate(&req)` → service call → `router.WriteJSON(w, status, v)`. Return 400 for decode errors, 422 for validation, 404 for not-found, 500 for unexpected.
- **Validation:** go-playground/validator tags on DTOs (`validate:"required,min=1,max=100"`). NOT manual `Valid()` method.
- **Service errors:** Wrap with `fmt.Errorf("opName: %w", err)`. Use `errors.Is()` — never `==`.
- **ID generation:** `generateID()` helper in service — `crypto/rand` 8-byte hex.
- **Context:** All service/repo methods accept `ctx context.Context` as first parameter.
- **Structured logging:** `infra/logger` — slog with JSON output. `logger.FromContext(ctx)` for trace-correlated logger.
- **OTEL tracing:** Store tracer as struct field (not global). Use `infra/otelhttp` middleware for net/http.
- **Middleware:** Standard `func(http.Handler) http.Handler` signature. `Router.Use()` stores middleware in a slice; applied per-handler via `wrap()`. `Group.Use()` for group-scoped middleware.
- **Route registration:** Typed HTTP method helpers: `g.GET("/", h)`, `g.POST("/", h)`, `g.PUT("/{id}", h)`, `g.DELETE("/{id}", h)`, `g.ANY("/path", h)`. Group methods accept `http.HandlerFunc`; Router methods accept `http.Handler`.
- **Nested groups:** `g.Group(prefix, fn)` creates sub-groups inheriting parent middleware. Add sub-group-specific middleware via `Use()` before registering routes.
- **Path params:** `r.PathValue("id")` (stdlib).
- **Graceful shutdown:** `router.GracefulServe(ctx, httpServer, timeout)`.

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

- **`handler-patterns`** — Handler pipeline, error responses, validation patterns (net/http)
- **`sqlite-config`** — Connection setup, migrations, sqlc codegen, import paths
- **`observability`** — OTEL tracing, structured logging, metrics, Grafana stack
