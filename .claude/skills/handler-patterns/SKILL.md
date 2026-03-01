---
name: handler-patterns
description: net/http handler pipeline, error responses, and validation patterns for this project
user_invocable: false
---

Reference for net/http handler patterns used across all `adapter/<domain>/` packages.

## Handler Signature

```go
func (m *Module) xxxHandler(w http.ResponseWriter, r *http.Request)
```

All handlers are **unexported** methods on `*Module`.

## Handler Pipeline

Every handler follows this exact flow:

```
json.NewDecoder(r.Body).Decode(&req) → m.val.Validate(&req) → service call → router.WriteJSON(w, status, response)
```

## Error Response Patterns

| Step | HTTP Status | Response |
|------|-------------|----------|
| `json.Decode` fails | **400** | `{"error": "invalid request body"}` |
| `m.val.Validate` fails | **422** | `{"error": "<validation details>"}` |
| `<domain>.ErrNotFound` from service | **404** | `{"error": "<domain> not found"}` |
| Unexpected error | **500** | `{"error": "internal error"}` — log full error with slog |

### Code Examples

```go
// Decode error → 400
var req CreateXxxRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
    return
}

// Validate error → 422
if err := m.val.Validate(&req); err != nil {
    http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusUnprocessableEntity)
    return
}

// Not found → 404 (using exported domain sentinel)
if errors.Is(err, <domain>.ErrNotFound) {
    http.Error(w, `{"error":"xxx not found"}`, http.StatusNotFound)
    return
}

// Internal error → 500 (never expose err.Error())
logger.FromContext(ctx).Error("failed to create xxx", "error", err)
http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
```

## Route Registration

Uses Go 1.22+ ServeMux pattern routing:

```go
func (m *Module) RegisterRoutes(g *router.Group) {
    g.HandleFunc("GET /", m.listXxxHandler)
    g.HandleFunc("POST /", m.createXxxHandler)
    g.HandleFunc("GET /{id}", m.getXxxByIDHandler)
    g.HandleFunc("PUT /{id}", m.updateXxxHandler)
    g.HandleFunc("DELETE /{id}", m.deleteXxxHandler)
}
```

`Group` also provides `Handle(pattern, http.Handler)` and `Route(pattern, http.Handler)` for pre-built handlers.

## Group-Level Middleware

Groups support scoped middleware via `Use()`. Middleware added to a group wraps only that group's handlers (first added = outermost):

```go
func (m *Module) RegisterRoutes(g *router.Group) {
    g.Use(someAuthMiddleware)  // only applies to routes in this group
    g.HandleFunc("GET /", m.listXxxHandler)
}
```

## Nested Groups

`Group.Group(prefix, fn)` creates sub-groups that inherit the parent's middleware chain:

```go
func (m *Module) RegisterRoutes(g *router.Group) {
    g.HandleFunc("GET /", m.listXxxHandler)       // /xxx — no extra middleware

    g.Group("/admin", func(sub *router.Group) {
        sub.Use(adminOnlyMiddleware)               // inherits parent mw + adds its own
        sub.HandleFunc("DELETE /{id}", m.deleteXxxHandler)  // /xxx/admin/{id}
    })
}
```

## Path Parameters

```go
id := r.PathValue("id")
```

## JSON Response Helper

```go
router.WriteJSON(w, http.StatusOK, data)
router.WriteJSON(w, http.StatusCreated, created)
```

## Validation

- Use **go-playground/validator** tags on DTOs: `validate:"required,min=1,max=100"`
- **NOT** manual `Valid()` method
- `m.val.Validate(&req)` triggers validation via `infra/validator/validator.go`
- DTOs live in `adapter/<domain>/dto.go` with both `validate` and `example` tags

## Middleware

Standard `func(http.Handler) http.Handler` signature.

**Global** (all routes): via `Router.Use()`:
```go
r.Use(otelhttp.Middleware("restful-boilerplate"))
r.Use(requestlogger.RequestLog)
r.Use(requestlogger.Recovery)
```

**Group-scoped** (single group's routes): via `Group.Use()`:
```go
g.Use(authMiddleware) // must be added before registering routes
```

## Rules

- Never expose `err.Error()` directly in production 500 responses
- Always use `errors.Is()` / `errors.As()` — never compare errors with `==`
- Swagger annotations (`@Summary`, `@Tags`, `@Router`, status codes) on every handler — see `/gen-swagger` skill
