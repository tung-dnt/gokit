# CLAUDE.md — gokit

## Project Overview

A Go RESTful API boilerplate built on **net/http (stdlib) + PostgreSQL + sqlc** with full observability (OpenTelemetry + SigNoz). Uses a 3-folder structure: `internal/` for business modules, `pkg/` for pure utilities, `infra/` for Docker/observability configs only. Each domain is a **flat single package** — all layers (adapter, service, mapping, DTOs, errors) live in `internal/<domain>/` with file-name prefixes instead of sub-packages.

It also hosts AI-agent domains that use Firebase Genkit for LLM-backed endpoints (RAG, generative flows). See the `recipe` domain as reference.

**Module:** `gokit` | **Go:** 1.26.0 | **Deps:** jackc/pgx/v5, exaring/otelpgx, go-playground/validator, swaggo/swag, firebase/genkit, xavidop/genkit-opentelemetry-go

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
                           → (or adapter.agent.go for Genkit AI-agent domains)
    service.<domain>.go    → <domain>Service struct + new<Domain>Service(q, tracer) + all CRUD methods
    domain.dto.go          → Create/UpdateXRequest with validate + json + example tags
    domain.error.go        → ErrNotFound sentinel
    mapping.response.go    → <Domain>Response struct + ToResponse(pgdb.<Domain>)
    observability.<domain>.go → (AI-agent domains only) maps framework LLM response to telemetry.LLMInfo
  recipe/                  → AI-agent module — package recipe (example of Genkit domain)
    module.recipe.go       → Module + NewModule(a) (*Module, error) + RegisterRoutes
    adapter.agent.go       → Agent adapter (calls Genkit flows)
    service.recipe.go      → recipeService — Genkit flows, LLM calls
    domain.dto.go          → request/response DTOs
    domain.error.go        → ErrNotFound sentinel
    mapping.response.go    → response mapping
    observability.recipe.go → framework → telemetry.LLMInfo mapping

pkg/                       → Pure utilities — NO business knowledge
  http/                    → Router wrapper for net/http ServeMux
    router.go              → Router struct + NewRouter + Prefix + Use + Group + Route
    group.go               → Group struct + Use (group middleware) + GET/POST/PUT/PATCH/DELETE/ANY + nested Group
    util.go                → WriteJSON + Bind helpers
    server.go              → GracefulServe (graceful shutdown)
  util/                    → Shared utilities — package util
    generate-id.go         → GenerateID() — crypto/rand 8-byte hex
  chunker/                 → Text chunker for RAG
    chunker.go             → Chunk(text, opts) — splits text for embedding/retrieval
  version/                 → Build-time version string
    version.go             → Version populated via ldflags
  config/                  → Env-var config loader
  logger/                  → slog MultiWriter (stdout + ./logs/app.log) + OTLP bridge
  recovery/                → Recovery middleware (net/http)
  telemetry/               → OTLP providers + span helpers + LLM observability
    otel.go                → TracerProvider + Endpoint() + EndpointHTTP()
    meter.go               → MeterProvider (delta temporality)
    logs.go                → LoggerProvider
    setup.go               → SetupAll() — wires all three providers
    span.go                → SpanExpectedErr / SpanUnexpectedErr + ErrKind constants
    llm.go                 → StartLLMSpan + RecordLLMAttrs (Traceloop + GenAI semconv)
  validator/               → Validator adapter (go-playground/validator)
  testutil/                → Shared test helpers (SetupPgTestDB for PostgreSQL, SetupTestDB for legacy SQLite)
  postgres/                → PostgreSQL connection + migration infra
    connection.go          → OpenDB() — pgxpool.Pool (instrumented via otelpgx)
    migrate.go             → Migrate() with //go:embed
    db/                    → sqlc-generated Go code (package pgdb) — includes Querier interface
    queries/               → sqlc-annotated SQL query files
    migrations/            → SQL CREATE TABLE files (TIMESTAMPTZ, $1 params)
  sqlite/                  → Legacy SQLite layer (kept for reference)

prompts/                   → Dotprompt templates for Genkit
  <domain>_<op>.prompt     → e.g. recipe_generate.prompt

infra/                     → Docker/observability configs ONLY (no Go code)
  docker-compose.yaml      → PostgreSQL + SigNoz stack (ClickHouse, OTEL Collector, UI)
  Dockerfile               → Prod image for the HTTP server
  signoz/                  → ClickHouse + OTEL Collector + SigNoz configs

cmd/
  http/main.go             → net/http server entrypoint (explicit DI, inline wiring)
  postgres-migration/      → standalone DB migration tool
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
internal/app/       → pkg/postgres/db (Queries) + pkg/validator (Validator interface) + firebase/genkit (Agent) + go.opentelemetry.io/otel/trace (TracerProvider)
pkg/                → stdlib + external libs only (no business knowledge)
infra/              → Docker/observability configs only (no Go code)
```

No repository interface — the service uses `*pgdb.Queries` directly. Each domain is a **flat single package** — all concerns live in one `internal/<domain>/` directory using file-name prefixes (`adapter.http.go`, `service.<domain>.go`, `domain.dto.go`, `domain.error.go`, `mapping.response.go`). No sub-packages, no cross-package imports within a domain.

**Wiring in main.go:**
```go
otelPlugin := opentelemetry.New(opentelemetry.Config{
    ServiceName:    "gokit",
    ForceExport:    true,
    OTLPEndpoint:   telemetry.EndpointHTTP(),
    OTLPUseHTTP:    true,
    MetricInterval: 15 * time.Second,
})
ai := genkit.Init(ctx,
    genkit.WithPlugins(&googlegenai.GoogleAI{}, otelPlugin),
    genkit.WithDefaultModel("googleai/gemini-2.5-flash"),
    genkit.WithPromptDir("./prompts"),
)

pool, err := postgres.OpenDB(ctx, cfg.DatabaseURL)
postgres.Migrate(ctx, pool)
defer pool.Close()

a := &app.App{
    Queries:   pgdb.New(pool),
    Validator: v,
    Tracer:    otel.GetTracerProvider(),
    Agent:     ai,
}

r := router.NewRouter(router.WithInstrumentation("http.server"))
r.Use(logger.Middleware)
r.Use(recovery.Middleware)

r.Group("/v1", func(g *router.Group) {
    g.Prefix("/api")
    g.ANY("/swagger/", httpSwagger.WrapHandler)
    g.Group("/users", user.NewModule(a).RegisterRoutes)
    g.Group("/agents", func(g *router.Group) {
        g.Group("/recipes", recipeMod.RegisterRoutes)  // recipe module's NewModule returns (*, error)
    })
})
```

**Environment:** `DATABASE_URL` — defaults to `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`

To add a new domain: run `/new-domain` — it creates `internal/<domain>/` as a flat package with all domain files and adds `g.Group("/<domain>s", <domain>.NewModule(a).RegisterRoutes)` to the v1 group in `main.go`.

To add a new AI-agent domain backed by Genkit: run `/new-genkit-agent` — it scaffolds the flat package, a Dotprompt file under `prompts/`, and wires the route under `/agents/<domain>`.

---

## Key Patterns

- **Flat domain package** (`internal/<domain>/`, package `<domain>`): All layers in one package, separated by file-name prefix. No sub-packages, no cross-domain imports within a domain.
- **File naming convention:** `adapter.http.go`, `service.<domain>.go`, `domain.dto.go`, `domain.error.go`, `mapping.response.go`, `module.<domain>.go`.
- **DTOs** (`domain.dto.go`): `Create<Domain>Request`, `Update<Domain>Request` with `validate` + `json` + `example` tags. Used directly — no type aliases.
- **Error sentinel** (`domain.error.go`): Single `ErrNotFound` declaration in the domain package. No duplication.
- **Service** (`service.<domain>.go`): Unexported `<domain>Service` struct holding `db pgdb.Querier` (the sqlc-generated **interface**, NOT the concrete `*pgdb.Queries`) + unexported constructor `new<Domain>Service(q, tracer)` + all CRUD methods (unexported, e.g. `createUser`, `listUsers`). All return `*pgdb.<Entity>` directly. OTEL tracing on every method. Services are unit-testable by mocking the `Querier` interface.
- **Mapping** (`mapping.response.go`): `<Domain>Response` struct + `ToResponse(pgdb.<Domain>)`. Called by adapter before `WriteJSON`.
- **Adapter** (`adapter.http.go` or `adapter.agent.go`): Unexported `httpAdapter{svc userSvc, val app.Validator}` + unexported constructor `newHTTPAdapter`. The adapter depends on a service **seam interface** (e.g. `userSvc`), NOT the concrete `*userService`. Tests inject a `mockUserSvc`. Handler methods may be exported (needed for route registration) or unexported.
- **Module** (`module.<domain>.go`, package `<domain>`): `Module{httpAdapter *httpAdapter}` + `NewModule(a *app.App)` + `RegisterRoutes`. For Genkit domains the signature is `NewModule(a) (*Module, error)`. Constructor calls `new<Domain>Service` then `newHTTPAdapter`. Main imports only the domain root package.
- **No repository adapter:** Service calls `s.db.<QueryName>(ctx, ...)` on a `pgdb.Querier` interface directly. No intermediate repository interface.
- **App container:** `internal/app/app.App` holds `*pgdb.Queries`, `Validator`, `trace.TracerProvider`, and `*genkit.Genkit`. Created once in `main`. Every module receives it via `NewModule(a)`.
- **Tracer:** `a.Tracer.Tracer("<domain>")` called inside `NewModule`. Tests inject `noop.NewTracerProvider()`.
- **Handler pipeline:** `router.Bind(m.val, w, r, &req)` → service call → `ToResponse(...)` → `router.WriteJSON(w, status, v)`. `router.Bind` returns false and writes the error response on decode (400) or validation (422) failure — just `return` after it.
- **Handler error helper (`writeErr`):** `writeErr(r, w, err)` takes the request so it can pull `logger.FromContext(r.Context())` for trace-correlated logging. The response body is a stable string (e.g. `"user not found"`) — never raw `err.Error()`.
- **JSON helpers:** `router.WriteJSON(w, status, v)` + `router.Bind(val, w, r, &v)` — `pkg/http` package name is `router`. `net/http` imported as `http`.
- **Validation:** go-playground/validator tags on DTOs (`validate:"required,min=1,max=100"`). NOT manual `Valid()` method.
- **Service errors:** Wrap with `fmt.Errorf("opName: %w", err)`. Use `errors.Is()` — never `==`. For span status, use `telemetry.SpanExpectedErr(span, ErrX, op, telemetry.ErrKindX)` for expected 4xx outcomes (span status stays Unset) and `telemetry.SpanUnexpectedErr(span, err, op)` for 5xx outcomes (span status Error, `error.type` classified). The old `telemetry.SpanErr` is gone.
- **ID generation:** `shared.GenerateID()` — import as `shared "gokit/pkg/util"` — `crypto/rand` 8-byte hex.
- **Context:** All service/db methods accept `ctx context.Context` as first parameter.
- **Structured logging:** `pkg/logger` — slog with JSON output. `logger.FromContext(ctx)` for trace-correlated logger.
- **OTEL tracing:** Store tracer as struct field (not global). HTTP instrumentation is wired via `router.WithInstrumentation("http.server")` as a `NewRouter` option. Span names use matched-route patterns (bounded cardinality). `pgxpool` is instrumented via `otelpgx` inside `postgres.OpenDB`.
- **LLM observability:** `pkg/telemetry/llm.go` — `StartLLMSpan(ctx, tracer, op)` names spans `"llm.<op>"`; `RecordLLMAttrs(span, info)` dual-emits Traceloop + OTel GenAI semconv. Call-site lives in `service.<domain>.go`; per-domain framework mapper in `observability.<domain>.go` converts the framework's response to `telemetry.LLMInfo`.
- **Genkit:** `App.Agent *genkit.Genkit` is shared with every domain. Dotprompt files live under `prompts/`. `NewModule(a) (*Module, error)` — returning `error` is allowed (and required for Genkit domains that init embedders/retrievers).
- **Middleware:** Standard `func(http.Handler) http.Handler` signature. `Router.Use()` stores middleware in a slice; applied per-handler via `wrap()`. `Group.Use()` for group-scoped middleware.
- **Middleware order:** `r.Use(logger.Middleware)` then `r.Use(recovery.Middleware)` in main.
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

# Apply DB migrations (runs cmd/postgres-migration against DATABASE_URL)
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

# Build prod Docker image (uses infra/Dockerfile)
make docker/build

# Observability stack (SigNoz — ClickHouse + OTEL Collector + UI on :8080)
make obs/up
make obs/down
```

---

## Stack-Specific Reference Skills

These skills provide detailed patterns on demand — invoke when working in these areas:

- **`handler-patterns`** — Handler pipeline, error responses, validation patterns (net/http)
- **`postgres-config`** — PostgreSQL connection setup, migrations, sqlc codegen, pgx/v5 patterns, env vars
- **`sqlite-config`** — Legacy SQLite reference — stub only, kept for historical context (`pkg/sqlite/`)
- **`observability`** — OTEL tracing, structured logging, metrics, SigNoz stack
- **`new-genkit-agent`** — Scaffold an AI-agent domain (Dotprompt + Genkit + LLM tracing)
- **`llm-observability`** — LLM span helpers, `LLMInfo` schema, SigNoz GenAI dashboards
