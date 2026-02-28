---
name: gen-swagger
description: Add or update swag annotations on Echo handlers and regenerate the OpenAPI/Swagger docs in docs/
---

Manage Swagger/OpenAPI documentation for this project. The project uses `swaggo/swag` with Echo v5.

## Regenerate docs

After adding/updating swag annotations, always regenerate:

```bash
go tool swag init -g cmd/http/main.go -o docs/
go build ./...
```

The `-g` flag points to the file with `@title`, `@version`, `@host`, `@BasePath` annotations.
Output goes to `docs/` (swagger.yaml, swagger.json, docs.go — gitignored).

## Handler annotation template

Place annotations immediately above each handler function:

```go
// handlerName does X.
//
//  @Summary      Short description (shown in Swagger UI list)
//  @Description  Optional longer description
//  @Tags         <domain>s
//  @Accept       json
//  @Produce      json
//  @Param        id    path      string                     true   "Resource ID"
//  @Param        body  body      CreateXxxRequest           true   "Request body"
//  @Param        page  query     int                        false  "Page number"
//  @Success      201   {object}  <domain>.<Domain>
//  @Success      200   {array}   <domain>.<Domain>
//  @Failure      400   {object}  map[string]string
//  @Failure      404   {object}  map[string]string
//  @Failure      422   {object}  map[string]string
//  @Failure      500   {object}  map[string]string
//  @Router       /<domain>s [post]
func (h *Handler) createXxxHandler(c *echo.Context) error {
```

## Status code conventions

| Code | When |
|------|------|
| 200 | GET (single or list), PUT |
| 201 | POST (created) |
| 204 | DELETE (no body) |
| 400 | Malformed JSON (`c.Bind` error) |
| 404 | Resource not found (`<domain>.ErrNotFound`) |
| 422 | Validation failure (`c.Validate` error) |
| 500 | Unexpected service/DB error |

## Main file annotations (cmd/http/main.go)

These are already set — only update if the API version or base path changes:

```go
//  @title          Restful Boilerplate API
//  @version        1.0
//  @description    Go RESTful API boilerplate built on Echo v5 + SQLite.
//  @host           localhost:8080
//  @BasePath       /api
//  @schemes        http
```

## `example` tags on structs

Add `example` struct tags on entity fields (in `domain/<domain>/entity.go`) and DTOs so Swagger UI shows realistic payloads:

```go
type User struct {
    ID        string    `json:"id"         example:"a1b2c3d4e5f6g7h8"`
    Name      string    `json:"name"       example:"Alice"`
    Email     string    `json:"email"      example:"alice@example.com"`
    CreatedAt time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
}
```

## View in browser

After regenerating, start the server and open:

```
http://localhost:8080/swagger/index.html
```

## Checklist

- [ ] All handler functions have `@Summary`, `@Tags`, `@Router`
- [ ] Request body handlers have `@Accept json` and `@Param body`
- [ ] All success/error status codes documented
- [ ] Entity structs have `example` tags
- [ ] DTO structs have `example` tags
- [ ] `make swagger` ran successfully
- [ ] `make check` passes
