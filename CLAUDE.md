# CLAUDE.md — restful-boilerplate

## Project Overview

A Go RESTful API boilerplate built on **net/http (stdlib) + PostgreSQL + sqlc** with full observability (OpenTelemetry, Tempo, Loki, Grafana). Uses a 3-folder structure: `internal/` for business modules, `pkg/` for pure utilities, `infra/` for Docker/observability configs only. Each domain is split into four sub-packages: `adapter/`, `core/`, `mapping/`, `model/`.

**Module:** `restful-boilerplate` | **Go:** 1.26.0 | **Deps:** jackc/pgx/v5, go-playground/validator, swaggo/swag

**Legacy:** `pkg/sqlite/` — SQLite code retained for reference; `modernc.org/sqlite` still in go.mod

**Dev tools (via `go tool`):** `sqlc generate`, `swag init`, `air`, `golangci-lint`

---

## Directory Structure

```
internal/
  app/                     → Shared DI container (package app)
    app.go                 → App struct + Validator interface
  <domain>/                → Domain root — package <domain>
    module.go              → Module{httpAdapter *<domain>adapter.HTTPAdapter} + NewModule(a *app.App) + RegisterRoutes
    module.test.go         → Internal test package (package <domain>)
    adapter/               → HTTP adapter layer — package <domain>adapter
      http.go              → HTTPAdapter{svc *core.Service, val app.Validator} + exported handler methods
    core/                  → Business logic layer — package <domain>core
      <domain>.go          → ErrNotFound sentinel + type aliases (CreateXInput = model.CreateXRequest)
      service.go           → Service struct + NewService(q, tracer) + all CRUD methods (consolidated)
      service_test.go      → External test package (package <domain>core_test)
    mapping/               → Response conversion layer — package <domain>mapping
      mapping.go           → <Domain>Response struct + ToResponse(pgdb.<Domain>)
    model/                 → Request/response types — package <domain>model
      dto.go               → Create/UpdateXRequest with validate + json + example tags
      dto_test.go          → Validator tag tests (internal package <domain>model)
      errors.go            → ErrNotFound sentinel
  shared/                  → Cross-domain utilities — package shared
    generate-id.go         → GenerateID() — crypto/rand 8-byte hex

pkg/                       → Pure utilities — NO business knowledge
  http/                    → Router wrapper for net/http ServeMux
    router.go              → Router struct + NewRouter + Prefix + Use + Group + Route
    group.go               → Group struct + Use (group middleware) + GET/POST/PUT/PATCH/DELETE/ANY + nested Group
    util.go                → WriteJSON helper
    server.go              → GracefulServe (graceful shutdown)
  sqlite/                  → SQLite connection + migration infra
    connection.go          → OpenDB() — single conn + WAL + busy_timeout
    migrate.go             → Migrate() with //go:embed
    db/                    → sqlc-generated Go code (package sqlitedb)
    queries/               → sqlc-annotated SQL query files
    migrations/            → SQL CREATE TABLE files
  config/                  → Env-var config loader
  logger/                  → slog MultiWriter (stdout + ./logs/app.log)
  metrics/                 → Prometheus metrics (promhttp)
  recovery/                → Recovery middleware (net/http)
  otelhttp/                → Custom net/http OTEL middleware
  telemetry/               → OTLP TracerProvider setup
  validator/               → Validator adapter (go-playground/validator)
  testutil/                → Shared test helpers (SetupPgTestDB for PostgreSQL, SetupTestDB for legacy SQLite)
  postgres/                → PostgreSQL connection + migration infra
    connection.go          → OpenDB() — pgxpool.Pool
    migrate.go             → Migrate() with //go:embed
    db/                    → sqlc-generated Go code (package pgdb)
    queries/               → sqlc-annotated SQL query files
    migrations/            → SQL CREATE TABLE files (TIMESTAMPTZ, $1 params)
  sqlite/                  → Legacy SQLite layer (kept for reference)

infra/                     → Docker/observability configs ONLY (no Go code)
  docker-compose.yml
  alloy/                   → Grafana Alloy log shipper config
  grafana/                 → Grafana provisioning (dashboards + datasources)
  loki/                    → Loki config
  prometheus/              → Prometheus config
  tempo/                   → Tempo config

cmd/
  http/main.go             → net/http server entrypoint (explicit DI, inline wiring)
docs/                      → swag-generated OpenAPI docs (gitignored)
scripts/                   → k6 performance test script
sqlc.yaml                  → sqlc v2 config
.golangci.yml              → golangci-lint config
Makefile                   → All dev tasks
```

---

## Architecture: 3-Folder Structure

Simple layered structure with clear dependency rules:

```
internal/<domain>/  → internal/app (App container) + pkg/postgres/db + pkg/logger
internal/app/       → pkg/postgres/db (Queries) + pkg/validator (Validator interface)
pkg/                → stdlib + external libs only (no business knowledge)
infra/              → Docker/observability configs only (no Go code)
```

No repository interface — the service uses `*pgdb.Queries` directly. Each domain has four sub-packages: `adapter/` (HTTP), `core/` (business logic), `mapping/` (response conversion), `model/` (DTOs). Dependency rule: `adapter/` imports `core/`, `mapping/`, `model/`; `core/` imports `model/`; `model/` imports nothing from the domain. No circular deps.

**Wiring in main.go:**
```go
pool, err := postgres.OpenDB(ctx, cfg.DatabaseURL)
postgres.Migrate(ctx, pool)
defer pool.Close()

a := &app.App{
    Queries:   pgdb.New(pool),
    Validator: v,
    Tracer:    otel.GetTracerProvider(),
}

r.Group("/v1", func(g *router.Group) {
    g.Prefix("/api")
    g.ANY("/swagger/", httpSwagger.WrapHandler)
    g.Group("/users", user.NewModule(a).RegisterRoutes)
})
```

**Environment:** `DATABASE_URL` — defaults to `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`

To add a new domain: run `/new-domain` — it creates `internal/<domain>/` with all sub-packages and adds `g.Group("/<domain>s", <domain>.NewModule(a).RegisterRoutes)` to the v1 group in `main.go`.

---

## Key Patterns

- **Model layer** (`internal/<domain>/model/`, package `<domain>model`): Request DTOs (`Create<Domain>Request`, `Update<Domain>Request`) with `validate` + `json` + `example` tags in `dto.go`. `ErrNotFound` sentinel in `errors.go`. Validator tag tests in `dto_test.go` (package `<domain>model`).
- **Core layer** (`internal/<domain>/core/`, package `<domain>core`): `ErrNotFound` + type aliases (`CreateXInput = model.CreateXRequest`) in `<domain>.go`. Consolidated `Service` struct + `NewService(q, tracer)` + all CRUD methods in `service.go`. All return `*pgdb.<Entity>` directly. OTEL tracing on every method. Service tests in `service_test.go` (package `<domain>core_test`).
- **Mapping layer** (`internal/<domain>/mapping/`, package `<domain>mapping`): `<Domain>Response` struct + `ToResponse(pgdb.<Domain>)` in `mapping.go`. Called by adapter before `WriteJSON`.
- **Adapter layer** (`internal/<domain>/adapter/`, package `<domain>adapter`): `HTTPAdapter{svc *<domain>core.Service, val app.Validator}` + exported handler methods (`ListXxxHandler`, `CreateXxxHandler`, etc.). Imports `core/`, `mapping/`, `model/`.
- **Module** (`internal/<domain>/module.go`, package `<domain>`): `Module{httpAdapter *<domain>adapter.HTTPAdapter}` + `NewModule(a *app.App)` + `RegisterRoutes`. Constructor calls `core.NewService` then `adapter.NewHTTPAdapter`. Main imports only the domain root package.
- **No repository adapter:** Service calls `s.q.<QueryName>(ctx, ...)` on `*pgdb.Queries` directly. No intermediate interface.
- **App container:** `internal/app/app.App` holds `*pgdb.Queries`, `Validator`, and `trace.TracerProvider`. Created once in `main`. Every module receives it via `NewModule(a)`.
- **Tracer:** `a.Tracer.Tracer("<domain>")` called inside `NewModule`. Tests inject `noop.NewTracerProvider()`.
- **Handler pipeline:** `json.NewDecoder(r.Body).Decode(&req)` → `m.val.Validate(&req)` → service call → `router.WriteJSON(w, status, v)`. Return 400 for decode errors, 422 for validation, 404 for not-found, 500 for unexpected.
- **JSON helper:** `router.WriteJSON(w, status, v)` — `pkg/http` package name is `router` (no alias needed). `net/http` imported as `http`.
- **Validation:** go-playground/validator tags on DTOs (`validate:"required,min=1,max=100"`). NOT manual `Valid()` method.
- **Service errors:** Wrap with `fmt.Errorf("opName: %w", err)`. Use `errors.Is()` — never `==`.
- **ID generation:** `shared.GenerateID()` from `internal/shared` — `crypto/rand` 8-byte hex.
- **Context:** All service/db methods accept `ctx context.Context` as first parameter.
- **Structured logging:** `pkg/logger` — slog with JSON output. `logger.FromContext(ctx)` for trace-correlated logger.
- **OTEL tracing:** Store tracer as struct field (not global). Use `pkg/otelhttp` middleware for net/http.
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
- **`postgres-config`** — PostgreSQL connection setup, migrations, sqlc codegen, pgx/v5 patterns, env vars
- **`sqlite-config`** — Legacy SQLite reference (kept for historical context)
- **`observability`** — OTEL tracing, structured logging, metrics, Grafana stack
