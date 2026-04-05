# CLAUDE.md — restful-boilerplate

## Project Overview

A Go RESTful API boilerplate built on **net/http (stdlib) + PostgreSQL + sqlc** with full observability (OpenTelemetry, Tempo, Loki, Grafana). Uses a 3-folder structure: `internal/` for business modules, `pkg/` for pure utilities, `infra/` for Docker/observability configs only. Each domain is a **flat single package** — all layers (adapter, service, mapping, DTOs, errors) live in `internal/<domain>/` with file-name prefixes instead of sub-packages.

**Module:** `restful-boilerplate` | **Go:** 1.26.0 | **Deps:** jackc/pgx/v5, go-playground/validator, swaggo/swag

**Legacy:** `pkg/sqlite/` — SQLite code retained for reference; `modernc.org/sqlite` still in go.mod

**Dev tools (via `go tool`):** `sqlc generate`, `swag init`, `air`, `golangci-lint`

---

## Directory Structure

```
internal/
  app/                     → Shared DI container (package app)
    app.go                 → App struct + Validator interface
  <domain>/                → Flat domain package — package <domain>
    module.<domain>.go     → Module{httpAdapter *httpAdapter} + NewModule(a *app.App) + RegisterRoutes
    adapter.http.go        → httpAdapter{svc *<domain>Service, val app.Validator} + handler methods (unexported)
    service.<domain>.go    → <domain>Service struct + new<Domain>Service(q, tracer) + all CRUD methods
    domain.dto.go          → Create/UpdateXRequest with validate + json + example tags
    domain.error.go        → ErrNotFound sentinel
    mapping.response.go    → <Domain>Response struct + ToResponse(pgdb.<Domain>)

pkg/                       → Pure utilities — NO business knowledge
  http/                    → Router wrapper for net/http ServeMux
    router.go              → Router struct + NewRouter + Prefix + Use + Group + Route
    group.go               → Group struct + Use (group middleware) + GET/POST/PUT/PATCH/DELETE/ANY + nested Group
    util.go                → WriteJSON + Bind helpers
    server.go              → GracefulServe (graceful shutdown)
  util/                    → Shared utilities — package util
    generate-id.go         → GenerateID() — crypto/rand 8-byte hex
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
internal/<domain>/  → internal/app (App container) + pkg/postgres/db + pkg/logger + pkg/util
internal/app/       → pkg/postgres/db (Queries) + pkg/validator (Validator interface)
pkg/                → stdlib + external libs only (no business knowledge)
infra/              → Docker/observability configs only (no Go code)
```

No repository interface — the service uses `*pgdb.Queries` directly. Each domain is a **flat single package** — all concerns live in one `internal/<domain>/` directory using file-name prefixes (`adapter.http.go`, `service.<domain>.go`, `domain.dto.go`, `domain.error.go`, `mapping.response.go`). No sub-packages, no cross-package imports within a domain.

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

To add a new domain: run `/new-domain` — it creates `internal/<domain>/` as a flat package with all domain files and adds `g.Group("/<domain>s", <domain>.NewModule(a).RegisterRoutes)` to the v1 group in `main.go`.

---

## Key Patterns

- **Flat domain package** (`internal/<domain>/`, package `<domain>`): All layers in one package, separated by file-name prefix. No sub-packages, no cross-domain imports within a domain.
- **File naming convention:** `adapter.http.go`, `service.<domain>.go`, `domain.dto.go`, `domain.error.go`, `mapping.response.go`, `module.<domain>.go`.
- **DTOs** (`domain.dto.go`): `Create<Domain>Request`, `Update<Domain>Request` with `validate` + `json` + `example` tags. Used directly — no type aliases.
- **Error sentinel** (`domain.error.go`): Single `ErrNotFound` declaration in the domain package. No duplication.
- **Service** (`service.<domain>.go`): Unexported `<domain>Service` struct + unexported constructor `new<Domain>Service(q, tracer)` + all CRUD methods (unexported, e.g. `createUser`, `listUsers`). All return `*pgdb.<Entity>` directly. OTEL tracing on every method.
- **Mapping** (`mapping.response.go`): `<Domain>Response` struct + `ToResponse(pgdb.<Domain>)`. Called by adapter before `WriteJSON`.
- **Adapter** (`adapter.http.go`): Unexported `httpAdapter{svc *<domain>Service, val app.Validator}` + unexported constructor `newHTTPAdapter`. Handler methods may be exported (needed for route registration) or unexported.
- **Module** (`module.<domain>.go`, package `<domain>`): `Module{httpAdapter *httpAdapter}` + `NewModule(a *app.App)` + `RegisterRoutes`. Constructor calls `new<Domain>Service` then `newHTTPAdapter`. Main imports only the domain root package.
- **No repository adapter:** Service calls `s.q.<QueryName>(ctx, ...)` on `*pgdb.Queries` directly. No intermediate interface.
- **App container:** `internal/app/app.App` holds `*pgdb.Queries`, `Validator`, and `trace.TracerProvider`. Created once in `main`. Every module receives it via `NewModule(a)`.
- **Tracer:** `a.Tracer.Tracer("<domain>")` called inside `NewModule`. Tests inject `noop.NewTracerProvider()`.
- **Handler pipeline:** `router.Bind(m.val, w, r, &req)` → service call → `ToResponse(...)` → `router.WriteJSON(w, status, v)`. `router.Bind` returns false and writes the error response on decode (400) or validation (422) failure — just `return` after it.
- **JSON helpers:** `router.WriteJSON(w, status, v)` + `router.Bind(val, w, r, &v)` — `pkg/http` package name is `router`. `net/http` imported as `http`.
- **Validation:** go-playground/validator tags on DTOs (`validate:"required,min=1,max=100"`). NOT manual `Valid()` method.
- **Service errors:** Wrap with `fmt.Errorf("opName: %w", err)`. Use `errors.Is()` — never `==`.
- **ID generation:** `shared.GenerateID()` — import as `shared "restful-boilerplate/pkg/util"` — `crypto/rand` 8-byte hex.
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
