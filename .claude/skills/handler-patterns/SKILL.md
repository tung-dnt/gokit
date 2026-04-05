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

All handlers are **unexported** methods on `*httpAdapter` (in `internal/<domain>/adapter.http.go`), unless they need to be exported for route registration from the module.

## Handler Pipeline

Every handler follows this exact flow:

```
router.Bind(m.val, w, r, &req) → service call → ToResponse(...) → router.WriteJSON(w, status, response)
```

`router.Bind` combines decode + validate. It returns `false` and writes the error response automatically — just `return` after it.

## Error Response Patterns

| Step | HTTP Status | Response |
|------|-------------|----------|
| `router.Bind` decode fails | **400** | `{"error": "invalid request body"}` |
| `router.Bind` validate fails | **422** | validation error object |
| `ErrNotFound` from service | **404** | `{"error": "<domain> not found"}` |
| Unexpected error | **500** | `{"error": err.Error()}` |

Use a shared `writeErr` helper on the adapter:

```go
func (m *httpAdapter) writeErr(w http.ResponseWriter, err error) {
    if errors.Is(err, ErrNotFound) {
        router.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "<domain> not found"})
        return
    }
    router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}
```

### Code Examples

```go
// Imports in adapter.http.go:
// "net/http"                          → http.ResponseWriter, http.StatusXxx
// router "restful-boilerplate/pkg/http" → router.WriteJSON, router.Bind

// Bind (decode + validate) → 400 on decode fail, 422 on validation fail
var req CreateXxxRequest
if !router.Bind(m.val, w, r, &req) {
    return
}

// ErrNotFound → 404 (ErrNotFound is in same package, no prefix)
if errors.Is(err, ErrNotFound) {
    router.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "xxx not found"})
    return
}

// Internal error → 500
router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})

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

`Group` provides `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `ANY` methods accepting `http.HandlerFunc`.
`Router` provides the same methods but accepting `http.Handler`.

## Group-Level Middleware

Groups support scoped middleware via `Use()`. Middleware added to a group wraps only that group's handlers (first added = outermost):

```go
func (m *Module) RegisterRoutes(g *router.Group) {
    g.Use(someAuthMiddleware)  // only applies to routes in this group
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

## JSON Helpers

```go
router.WriteJSON(w, http.StatusOK, data)
router.Bind(m.val, w, r, &req)  // returns false + writes error on failure
```

## Validation

- Use **go-playground/validator** tags on DTOs: `validate:"required,min=1,max=100"`
- **NOT** manual `Valid()` method
- DTOs live in `internal/<domain>/domain.dto.go` (same package) with `validate`, `json`, and `example` tags
- `router.Bind` calls `val.Validate` internally — no manual two-step in handlers

## Middleware

Standard `func(http.Handler) http.Handler` signature.

**Global** (all routes): via `Router.Use()`:
```go
r.Use(metric.Middleware)
r.Use(otelhttp.Middleware("restful-boilerplate"))
r.Use(logger.Middleware)
r.Use(recovery.Middleware)
```

**Group-scoped** (single group's routes): via `Group.Use()`:
```go
g.Use(authMiddleware) // must be added before registering routes
```

## Rules

- Never expose `err.Error()` directly in production 500 responses
- Always use `errors.Is()` / `errors.As()` — never compare errors with `==`
- Swagger annotations (`@Summary`, `@Tags`, `@Router`, status codes) on every handler — see `/gen-swagger` skill
