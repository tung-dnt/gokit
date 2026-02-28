---
name: echo-handler-patterns
description: Echo v5 handler pipeline, error responses, and validation patterns for this project
user_invocable: false
---

Reference for Echo v5 HTTP handler patterns used across all `infra/http/<domain>hdl/` packages.

## Handler Signature

```go
func (h *Handler) xxxHandler(c *echo.Context) error
```

All handlers are **unexported** methods on `*Handler`.

## Handler Pipeline

Every handler follows this exact flow:

```
c.Bind(&req) → c.Validate(&req) → service call → c.JSON(status, response)
```

## Error Response Patterns

| Step | HTTP Status | Response |
|------|-------------|----------|
| `c.Bind` fails | **400** | `{"error": "invalid request body"}` |
| `c.Validate` fails | **422** | Auto-handled by `infra/validator` — returns field-level details |
| `<domain>.ErrNotFound` from service | **404** | `{"error": "<domain> not found"}` |
| Unexpected error | **500** | `{"error": "internal error"}` — log full error with slog |

### Code Examples

```go
// Bind error → 400
var req CreateXxxRequest
if err := c.Bind(&req); err != nil {
    return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
}

// Validate error → 422 (just return err, infra/validator handles it)
if err := c.Validate(&req); err != nil {
    return err
}

// Not found → 404 (using exported domain sentinel)
if errors.Is(err, <domain>.ErrNotFound) {
    return c.JSON(http.StatusNotFound, map[string]string{"error": "xxx not found"})
}

// Internal error → 500 (never expose err.Error())
logger.FromContext(ctx).Error("failed to create xxx", "error", err)
return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
```

## Validation

- Use **go-playground/validator** tags on DTOs: `validate:"required,min=1,max=100"`
- **NOT** manual `Valid()` method
- `c.Validate(&req)` triggers automatic 422 response via `infra/validator/validator.go`
- DTOs live in `infra/http/<domain>hdl/dto.go` with both `validate` and `example` tags

## Rules

- Never expose `err.Error()` directly in production 500 responses
- Always use `errors.Is()` / `errors.As()` — never compare errors with `==`
- Swagger annotations (`@Summary`, `@Tags`, `@Router`, status codes) on every handler — see `/gen-swagger` skill
