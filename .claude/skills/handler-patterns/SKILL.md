---
name: handler-patterns
description: net/http handler pipeline, error responses, and validation patterns for this project
user_invocable: false
---

Reference for net/http handler patterns used across all `internal/<domain>/` packages.

## Handler Signature

```go
func (m *httpAdapter) xxxHandler(w http.ResponseWriter, r *http.Request)
```

All handlers are methods on `*httpAdapter` (in `internal/<domain>/adapter.http.go`). The adapter depends on the `<domain>Svc` **interface** (service seam), not the concrete `*<domain>Service`. This lets tests inject a `mock<Domain>Svc` without touching a database.

```go
type <domain>Svc interface {
    create<Domain>(ctx context.Context, in Create<Domain>Request) (*pgdb.<Domain>, error)
    // ... every CRUD method the adapter calls
}

type httpAdapter struct {
    svc <domain>Svc
    val app.Validator
}
```

## Handler Pipeline

Every handler follows this exact flow:

```
router.Bind(m.val, w, r, &req)  →  svc call  →  (on err: writeErr)  →  router.WriteJSON(w, status, ToResponse(...))
```

`router.Bind` combines decode + validate. It returns `false` and writes the error response automatically — just `return` after it.

## Error Response Patterns

| Step | HTTP Status | Response body |
|------|-------------|---------------|
| `router.Bind` decode fails | **400** | `{"error": "invalid request body"}` |
| `router.Bind` validate fails | **422** | validation error object |
| `ErrNotFound` from service | **404** | `{"error": "<domain> not found"}` |
| Any other error | **500** | `{"error": "internal server error"}` |

Response bodies **never** leak raw `err.Error()`. Always a stable string. The real error is logged via the trace-correlated logger.

### `writeErr` — shared helper on the adapter

`writeErr` takes `*http.Request` so it can resolve `logger.FromContext(r.Context())` for trace correlation. Multiple error sentinels per domain are supported — switch via `errors.Is`.

```go
func (m *httpAdapter) writeErr(r *http.Request, w http.ResponseWriter, err error) {
    ctx := r.Context()
    log := logger.FromContext(ctx)

    if errors.Is(err, ErrNotFound) {
        log.DebugContext(ctx, "<domain> not found", "error", err)
        router.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "<domain> not found"})
        return
    }
    // Add more sentinels here as the domain grows, e.g.:
    // if errors.Is(err, ErrConflict) { ... 409 ... }

    log.ErrorContext(ctx, "<domain> request failed", "error", err)
    router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}
```

### Full handler example

```go
// createUserHandler creates a new user.
//
//	@Summary      Create user
//	@Tags         users
//	@Accept       json
//	@Produce      json
//	@Param        body  body      CreateUserRequest  true  "User data"
//	@Success      201   {object}  userResponse
//	@Failure      400   {object}  map[string]string
//	@Failure      422   {object}  map[string]string
//	@Failure      500   {object}  map[string]string
//	@Router       /users [post]
func (m *httpAdapter) createUserHandler(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if !router.Bind(m.val, w, r, &req) {
        return
    }
    u, err := m.svc.createUser(r.Context(), req)
    if err != nil {
        m.writeErr(r, w, err)
        return
    }
    router.WriteJSON(w, http.StatusCreated, ToResponse(*u))
}
```

### Code snippets

```go
// Imports in adapter.http.go:
// "net/http"                              → http.ResponseWriter, http.StatusXxx
// router "gokit/pkg/http"   → router.WriteJSON, router.Bind
// "gokit/pkg/logger"        → logger.FromContext

// Bind (decode + validate) → 400 on decode fail, 422 on validation fail
var req CreateXxxRequest
if !router.Bind(m.val, w, r, &req) {
    return
}

// Error dispatch
if err != nil {
    m.writeErr(r, w, err)
    return
}

// Success with mapping → 200/201
router.WriteJSON(w, http.StatusOK, ToResponse(*entity))

// No-content response → 204
w.WriteHeader(http.StatusNoContent)
```

## Route Registration

Routes registered in `internal/<domain>/module.<domain>.go`:

```go
func (m *Module) RegisterRoutes(g *router.Group) {
    g.GET("/", m.httpAdapter.listXxxHandler)
    g.POST("/", m.httpAdapter.createXxxHandler)
    g.GET("/{id}", m.httpAdapter.getXxxByIDHandler)
    g.PUT("/{id}", m.httpAdapter.updateXxxHandler)
    g.DELETE("/{id}", m.httpAdapter.deleteXxxHandler)
}
```

`Group` methods accept `http.HandlerFunc`; `Router` methods accept `http.Handler`.

## Group-Level Middleware

```go
func (m *Module) RegisterRoutes(g *router.Group) {
    g.Use(someAuthMiddleware)  // applies only to this group's routes
    g.GET("/", m.httpAdapter.listXxxHandler)
}
```

## Nested Groups

`Group.Group(prefix, fn)` creates sub-groups that inherit the parent's middleware chain:

```go
func (m *Module) RegisterRoutes(g *router.Group) {
    g.GET("/", m.httpAdapter.listXxxHandler)

    g.Group("/admin", func(sub *router.Group) {
        sub.Use(adminOnlyMiddleware)
        sub.DELETE("/{id}", m.httpAdapter.deleteXxxHandler)
    })
}
```

## Path Parameters

```go
id := r.PathValue("id")
```

In tests, set path values explicitly: `req.SetPathValue("id", "id-1")`.

## JSON Helpers

```go
router.WriteJSON(w, http.StatusOK, data)
router.Bind(m.val, w, r, &req)  // returns false + writes error on failure
```

## Validation

- Use **go-playground/validator** tags on DTOs: `validate:"required,min=1,max=100"`
- **NOT** manual `Valid()` methods
- DTOs live in `internal/<domain>/domain.dto.go` (same package) with `validate`, `json`, and `example` tags
- `router.Bind` calls `val.Validate` internally — no manual two-step in handlers

## Middleware

Standard `func(http.Handler) http.Handler` signature.

**Global** (all routes) via `Router.Use()`:
```go
r.Use(metric.Middleware)
r.Use(otelhttp.Middleware("gokit"))
r.Use(logger.Middleware)
r.Use(recovery.Middleware)
```

**Group-scoped** (single group's routes) via `Group.Use()`:
```go
g.Use(authMiddleware) // must be added before registering routes
```

## Rules

- **Never** expose `err.Error()` directly in production 5xx responses — always a stable string.
- Always use `errors.Is()` / `errors.As()` — never compare errors with `==`.
- Adapter depends on the `<domain>Svc` interface, not the concrete service type.
- `writeErr` takes `*http.Request` so it can log via `logger.FromContext(r.Context())` for trace correlation.
- Swagger annotations (`@Summary`, `@Tags`, `@Router`, status codes) on every handler — see `/gen-swagger` skill.
