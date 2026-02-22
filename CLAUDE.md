# CLAUDE.md — restful-boilerplate

## Project Overview

A Go RESTful API boilerplate demonstrating idiomatic Go patterns with **zero external dependencies**. Built on Go 1.25.0 standard library only (`net/http`, `log/slog`, `encoding/json`, `sync`, `crypto/rand`).

**Module:** `restful-boilerplate`

---

## Directory Structure

```
cmd/
  api/main.go           → HTTP API server entrypoint
  worker/main.go        → Background worker entrypoint
configs/
  config.go             → Env-var config loader (SERVER_HOST, SERVER_PORT, timeouts)
internal/
  <domain>/             → One folder per business domain (modular monolith)
    controller.go       → ONLY exported type — wires DI, registers routes/jobs
    handler.go          → HTTP handlers (unexported)
    model.go            → Domain entity + request/response structs (unexported)
    repository.go       → Repository interface + in-memory impl (all unexported)
    service.go          → Business logic, validation, ID generation (unexported)
deployments/            → Docker, Kubernetes manifests (placeholder)
scripts/                → Build/deploy shell scripts (placeholder)
test/                   → Integration test data and helpers (placeholder)
```

---

## Architecture: Modular Monolith

Each domain under `internal/` is fully self-contained. **`Controller` is the only exported symbol** per domain — all other types (service, repository, handlers, models) are package-private.

```
cmd/api/main.go
    └── user.NewController().RegisterRoutes(mux)   ← only touch point
            └── newUserService(repo)                ← internal wiring
                    └── newInMemoryRepository()
```

To add a new domain, create `internal/<domain>/` with the same 5-file pattern, then call `<domain>.NewController().RegisterRoutes(mux)` in `cmd/api/main.go`.

The `cmd/worker` binary follows the same pattern: `<domain>.NewController().StartScheduler(ctx)`.

---

## Key Patterns

- **Single export rule:** `Controller` is the only exported type per domain package. Everything else is unexported — enforced by package privacy.
- **`run()` delegation:** `main()` only calls `run(ctx, os.Stdout, os.Getenv)` and exits on error. All setup lives in `run()` — making it callable from tests.
- **Injectable env:** `config.Load(getenv func(string) string)` — never calls `os.Getenv` directly. `run()` passes its own `getenv`; tests supply a custom map.
- **Graceful shutdown:** `signal.NotifyContext()` drives shutdown via context cancellation. Call `stop()` after `<-ctx.Done()` to release the OS signal subscription.
- **Generic `decode[T]()`:** Single helper in `handler.go` for all JSON request decoding — `decode[createRequest](r)`. No inline `json.NewDecoder` in handlers.
- **Validator interface:** Request structs implement `Valid(ctx context.Context) map[string]string`. Handlers call it after decode and return 422 + `{"errors": {...}}` on failure. Services only contain business-rule validation.
- **Private repository interface:** `userRepository` interface lives inside the `user` package — service and handler can't escape to callers.
- **Concurrency safety:** `inMemoryRepository` uses `sync.RWMutex` (RLock for reads, Lock for writes).
- **Context propagation:** All repo/service methods accept `ctx context.Context` as first arg.
- **Error handling:** `fmt.Errorf("op: %w", err)` for wrapping; sentinel errors with `errors.Is()`.
- **Crypto IDs:** 8-byte hex via `crypto/rand` (not sequential integers).
- **Structured logging:** `log/slog` with `NewJSONHandler` — JSON to stdout.

---

## HTTP API

Base: `http://localhost:8080`

| Method | Path          | Action      |
|--------|---------------|-------------|
| GET    | /healthz      | health check (200 OK) |
| GET    | /users        | list        |
| POST   | /users        | create      |
| GET    | /users/{id}   | getByID     |
| PUT    | /users/{id}   | update      |
| DELETE | /users/{id}   | delete      |

---

## Development Commands

```bash
# Run API server
go run ./cmd/api

# Run background worker
go run ./cmd/worker

# Build all binaries
go build -o bin/api ./cmd/api
go build -o bin/worker ./cmd/worker

# Run all tests
go test ./...

# Format all Go files
gofmt -w .

# Vet code
go vet ./...

# Compile check (all packages)
go build ./...

# Health check
curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/healthz
# Expected: 200

# Validation error (422 with field map)
curl -s -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":""}' | jq
# Expected: 422 {"errors": {"email": "email is required", "name": "name is required"}}

# Create user
curl -s -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}' | jq

curl -s http://localhost:8080/users | jq
```

---

## Code Conventions

- Follow standard Go formatting (`gofmt`) — enforced by post-edit hook.
- **One export per domain:** Only `Controller` is exported from each `internal/<domain>` package.
- **`main()` stays lean:** All logic lives in `run(ctx, w, getenv)`. `main()` only calls it and exits on error.
- **Handler pipeline:** `decode[T]()` → `Valid()` → service call → `writeJSON`. Return 422 for validation errors (`{"errors": {...}}`), 400 for malformed JSON, 500 for unexpected errors.
- **Validation split:** Format/presence checks belong in `model.go` (`Valid()` method). Business-rule checks (uniqueness, state transitions) belong in `service.go`.
- Use `errors.Is()` / `errors.As()` — never compare errors with `==`.
- Always pass `context.Context` as first parameter in functions that do I/O.
- Wrap errors with context: `fmt.Errorf("userService.create: %w", err)`.
- No global state — all state flows through DI inside `NewController()`.
- Use `sync.RWMutex` for concurrent map access (RLock for reads).
- All internal types use lowercase names (unexported by convention).

---

## Adding a New Domain

1. Create `internal/<domain>/` with these 5 files:
   - `model.go` — domain struct + request/response types (all unexported)
   - `repository.go` — private interface + in-memory impl
   - `service.go` — business logic, validation, ID generation
   - `handler.go` — HTTP handler methods on `*Controller` (unexported)
   - `controller.go` — exported `Controller`, `NewController()`, `RegisterRoutes()`
2. In `cmd/api/main.go`: add `<domain>.NewController().RegisterRoutes(mux)`
3. In `cmd/worker/main.go` (if needed): add scheduler hookup

Copy `internal/user/` as a reference implementation.

---

## Planned / Next Steps

- SQLite repository implementation (swap `inMemoryRepository` — interface already in place)
- Unit tests in `internal/<domain>/` + integration test helpers in `test/`
- `golangci-lint` config (`.golangci.yml`)
- Makefile for common tasks
- Dockerfile + k8s manifests in `deployments/`
- Build/deploy scripts in `scripts/`

