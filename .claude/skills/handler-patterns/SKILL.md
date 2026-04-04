---
name: handler-patterns
description: net/http handler pipeline, error responses, and validation patterns for this project
user_invocable: false
---

Reference for net/http handler patterns used across all `internal/<domain>/adapter/` packages.

## Handler Signature

```go
func (m *HTTPAdapter) XxxHandler(w http.ResponseWriter, r *http.Request)
```

All handlers are **exported** methods on `*HTTPAdapter` (in `internal/<domain>/adapter/http.go`).

## Handler Pipeline

Every handler follows this exact flow:

```
json.NewDecoder(r.Body).Decode(&req) → m.val.Validate(&req) → service call → mapping.ToResponse → router.WriteJSON(w, status, response)
```

## Error Response Patterns

| Step | HTTP Status | Response |
|------|-------------|----------|
| `json.Decode` fails | **400** | `{"error": "invalid request body"}` |
| `m.val.Validate` fails | **422** | validation error object |
| `ErrNotFound` from service | **404** | `{"error": "<domain> not found"}` |
| Unexpected error | **500** | `{"error": err.Error()}` |

### Code Examples

```go
// Imports in adapter/http.go:
// "net/http"                                    → http.ResponseWriter, http.StatusXxx
// "restful-boilerplate/pkg/http"                → router.WriteJSON (pkg is named "router")

// Decode error → 400
var req <domain>model.CreateXxxRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    router.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
    return
}

// Validate error → 422
if err := m.val.Validate(&req); err != nil {
    router.WriteJSON(w, http.StatusUnprocessableEntity, err)
    return
}

// Not found → 404 (ErrNotFound lives in <domain>core)
if errors.Is(err, <domain>core.ErrNotFound) {
    router.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "xxx not found"})
    return
}

// Internal error → 500
router.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})

// Success with mapping → 200/201
router.WriteJSON(w, http.StatusOK, <domain>mapping.ToResponse(*entity))

// No-content response → 204
w.WriteHeader(http.StatusNoContent)
```

## Route Registration

Routes registered in `internal/<domain>/module.go` via the HTTPAdapter:

```go
// in internal/<domain>/module.go
func (m *Module) RegisterRoutes(g *router.Group) {
    g.GET("/", m.httpAdapter.ListXxxHandler)
    g.POST("/", m.httpAdapter.CreateXxxHandler)
    g.GET("/{id}", m.httpAdapter.GetXxxByIDHandler)
    g.PUT("/{id}", m.httpAdapter.UpdateXxxHandler)
    g.DELETE("/{id}", m.httpAdapter.DeleteXxxHandler)
}
```

`Group` provides `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `ANY` methods accepting `http.HandlerFunc`.
`Router` provides the same methods but accepting `http.Handler`.

## Group-Level Middleware

Groups support scoped middleware via `Use()`. Middleware added to a group wraps only that group's handlers (first added = outermost):

```go
func (m *Module) RegisterRoutes(g *router.Group) {
    g.Use(someAuthMiddleware)  // only applies to routes in this group
    g.GET("/", m.listXxxHandler)
}
```

## Nested Groups

`Group.Group(prefix, fn)` creates sub-groups that inherit the parent's middleware chain:

```go
func (m *Module) RegisterRoutes(g *router.Group) {
    g.GET("/", m.listXxxHandler)       // /xxx — no extra middleware

    g.Group("/admin", func(sub *router.Group) {
        sub.Use(adminOnlyMiddleware)               // inherits parent mw + adds its own
        sub.DELETE("/{id}", m.deleteXxxHandler)     // /xxx/admin/{id}
    })
}
```

## Path Parameters

```go
id := r.PathValue("id")
```

## JSON Response Helper

```go
import rt "restful-boilerplate/pkg/http"

rt.WriteJSON(w, http.StatusOK, data)
rt.WriteJSON(w, http.StatusCreated, created)
```

## Validation

- Use **go-playground/validator** tags on DTOs: `validate:"required,min=1,max=100"`
- **NOT** manual `Valid()` method
- `m.val.Validate(&req)` triggers validation via `pkg/validator/validator.go`
- DTOs live in `internal/<domain>/model/dto.go` (`package <domain>model`) with `validate`, `json`, and `example` tags
- Handlers decode into `<domain>model.Create<Domain>Request` directly (no separate adapter DTO)

## Middleware

Standard `func(http.Handler) http.Handler` signature.

**Global** (all routes): via `Router.Use()` — middleware is stored and applied per-handler via `wrap()`:
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
