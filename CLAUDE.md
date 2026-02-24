# CLAUDE.md — restful-boilerplate

## Project Overview

A Go RESTful API boilerplate built on **Echo v5 + SQLite + sqlc** with full observability (OpenTelemetry, Tempo, Loki, Grafana). Demonstrates idiomatic Go patterns for a modular monolith.

**Module:** `restful-boilerplate`
**Go version:** 1.25.0

**Key dependencies:**
- `github.com/labstack/echo/v5` — HTTP framework
- `modernc.org/sqlite` — SQLite driver (pure Go, no CGO)
- `github.com/go-playground/validator/v10` — request validation
- `github.com/swaggo/swag` — OpenAPI/Swagger generation

**Dev tools (via `go tool`):**
- `go tool sqlc generate` — type-safe SQL codegen
- `go tool swag init ...` — Swagger doc generation
- `go tool air` — hot-reload development server
- `go tool golangci-lint run ./...` — linting (via Makefile: `make lint`)

---

## Directory Structure

```
biz/
  <domain>/             → One folder per business domain
    route.go            → Controller struct + NewController(db) + RegisterRoutes
    controller.go       → HTTP handler methods (unexported) + swag annotations
    model.go            → Domain entity + input types + errNotFound sentinel
    service.go          → Business logic + CRUD + generateID()
    dto/dto.go          → Request DTOs with validate + example tags
cmd/
  http/main.go          → Echo server entrypoint + registerRouters()
pkg/
  config/config.go      → Env-var config loader
  logger/logger.go      → slog MultiWriter (stdout + ./logs/app.log)
  metrics/metrics.go    → Prometheus metrics
  middleware/           → Request logger middleware
  otelecho/middleware.go → Custom Echo v5 OTEL middleware
  telemetry/            → OTLP TracerProvider setup
  validator/validator.go → Echo Validator adapter (go-playground/validator)
repo/sqlite/
  db/                   → sqlc-generated Go code (gitignored source)
  migrations/           → SQL CREATE TABLE files + migration runner (main.go)
  queries/              → sqlc-annotated SQL query files
  db.go                 → OpenDB() — single connection + WAL + busy_timeout
  migrate.go            → Migrate() — runs all migration files
dx/
  deploy/               → Docker Compose for observability stack (Tempo, Loki, Alloy, Grafana)
  docs/                 → swag-generated OpenAPI docs (gitignored)
  scripts/              → k6 performance test script
  test/                 → Integration test helpers
sqlc.yaml               → sqlc v2 config
.golangci.yml           → golangci-lint config (23 linters, 5m timeout)
Makefile                → All dev tasks
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

## HTTP API

Base: `http://localhost:8080/api`

| Method | Path               | Action      |
|--------|--------------------|-------------|
| GET    | /healthz           | health check (200 OK) |
| GET    | /api/users         | list all    |
| POST   | /api/users         | create      |
| GET    | /api/users/{id}    | get by ID   |
| PUT    | /api/users/{id}    | update      |
| DELETE | /api/users/{id}    | delete      |
| GET    | /swagger/*         | Swagger UI  |
| GET    | /metrics           | Prometheus  |

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

## Code Conventions

- Follow standard Go formatting (`gofmt`) — enforced by post-edit hook + `make fmt`.
- **One export per domain:** Only `Controller` is exported from each `biz/<domain>` package.
- **Handler signature:** `func (ctrl *Controller) xxxHandler(c *echo.Context) error`
- **Bind errors:** return `c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})`
- **Validate errors:** return `err` (pkg/validator auto-sends 422 with field details)
- **Not found:** return `c.JSON(http.StatusNotFound, map[string]string{"error": "<domain> not found"})`
- **Internal errors:** log full error with slog, return `c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})` — never expose `err.Error()` directly in production 500s.
- All `@Summary`, `@Tags`, `@Router`, status codes in swag annotations — every handler.
- Use `errors.Is()` / `errors.As()` — never compare errors with `==`.
- Always pass `context.Context` as first parameter in functions that do I/O.
- Wrap errors: `fmt.Errorf("opName: %w", err)`.
- No global state — all state flows through constructors.

---

## Adding a New Domain

Use the `/new-domain` skill — it handles the full scaffold automatically:

1. `repo/sqlite/migrations/<domain>.sql` — CREATE TABLE
2. `repo/sqlite/queries/<domain>.sql` — sqlc CRUD queries
3. `go tool sqlc generate` — generates `repo/sqlite/db/<domain>.sql.go`
4. `biz/<domain>/model.go` — entity + input types + errNotFound
5. `biz/<domain>/dto/dto.go` — request DTOs with validate + example tags
6. `biz/<domain>/service.go` — CRUD + generateID() + mapper
7. `biz/<domain>/route.go` — Controller struct + NewController + RegisterRoutes
8. `biz/<domain>/controller.go` — handler methods with swag annotations
9. Wire `<domain>.NewController(db).RegisterRoutes(g.Group("/<domain>s"))` in `cmd/http/main.go`
10. `make swagger` — regenerate OpenAPI docs
11. `make check` — verify compilation, lint, tests pass

Copy `biz/user/` as the reference implementation.

---

## Observability Stack

Start with `make obs/up`. Access Grafana at `http://localhost:3000`.

| Component | Role |
|-----------|------|
| Tempo | Trace backend (OTLP HTTP port 4318) |
| Loki | Log aggregation |
| Alloy | Log shipper — tails `./logs/app.log` |
| Prometheus | Metrics scraper |
| Grafana | Visualization (datasource UIDs: `tempo`, `loki`) |

**Code integration:**
- `pkg/telemetry/setup.go` — `SetupAll(ctx, logPath)` initialises tracer + log file
- `pkg/otelecho/middleware.go` — injects `trace_id` + `span_id` into request context
- `pkg/logger/logger.go` — `logger.FromContext(ctx)` returns trace-correlated slog.Logger
- Set `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318` (default in dev)

---

## SQLite Notes

- Single connection mode (`MaxOpenConns(1)`) — serialises access, prevents `SQLITE_BUSY`
- `PRAGMA busy_timeout=5000` set on every connection open
- WAL mode enabled for better read concurrency
- `repo/sqlite/db.go` — `OpenDB(ctx, path)` — always use this, never `sql.Open` directly

---

## Workflow Orchestration

### 1. Plan Mode Default
- Enter plan mode for ANY non-trivial task (3+ steps or architectural decisions)
- If something goes sideways, STOP and re-plan immediately — don't keep pushing
- Use plan mode for verification steps, not just building
- Write detailed specs upfront to reduce ambiguity

### 2. Subagent Strategy
- Use subagents liberally to keep main context window clean
- Offload research, exploration, and parallel analysis to subagents
- For complex problems, throw more compute at it via subagents
- One task per subagent for focused execution

### 3. Self-Improvement Loop
- After ANY correction from the user: update `tasks/lessons.md` with the pattern
- Write rules for yourself that prevent the same mistake
- Ruthlessly iterate on these lessons until mistake rate drops
- Review lessons at session start for relevant project

### 4. Verification Before Done
- Never mark a task complete without proving it works
- Diff behavior between main and your changes when relevant
- Ask yourself: "Would a staff engineer approve this?"
- Run tests, check logs, demonstrate correctness

### 5. Demand Elegance (Balanced)
- For non-trivial changes: pause and ask "is there a more elegant way?"
- If a fix feels hacky: "Knowing everything I know now, implement the elegant solution"
- Skip this for simple, obvious fixes — don't over-engineer
- Challenge your own work before presenting it

### 6. Autonomous Bug Fixing
- When given a bug report: just fix it. Don't ask for hand-holding
- Point at logs, errors, failing tests — then resolve them
- Zero context switching required from the user
- Go fix failing CI tests without being told how

---

## Task Management

1. **Plan First:** Write plan to `tasks/todo.md` with checkable items
2. **Verify Plan:** Check in before starting implementation
3. **Track Progress:** Mark items complete as you go
4. **Explain Changes:** High-level summary at each step
5. **Document Results:** Add review section to `tasks/todo.md`
6. **Capture Lessons:** Update `tasks/lessons.md` after corrections

---

## Core Principles

- **Simplicity First:** Make every change as simple as possible. Impact minimal code.
- **No Laziness:** Find root causes. No temporary fixes. Senior developer standards.
- **Minimal Impact:** Changes should only touch what's necessary. Avoid introducing bugs.
