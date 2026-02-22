---
name: new-domain
description: Scaffold a new internal/<domain>/ module following the 5-file modular monolith pattern
---

Scaffold a new domain module under `internal/<domain>/` following the project's modular monolith pattern. Use `internal/user/` as the reference implementation.

## Rules

- Only `Controller` is exported — all other types (service, repository, handler, model) are unexported (lowercase).
- The repository must be a private interface inside the domain package.
- All handler methods live on `*Controller` but are unexported.
- IDs use `crypto/rand` 8-byte hex (see `internal/user/service.go` `generateID()`).
- All service/repository methods accept `ctx context.Context` as first parameter.
- Wrap errors: `fmt.Errorf("domainService.op: %w", err)`.
- **Validation split:** presence/format checks in `model.go` (`Valid()` method); business-rule checks in `service.go` only.

## Files to create

### `internal/<domain>/model.go`

Domain entity struct + unexported request/response structs. JSON tags on entity fields.

Implement the `Validator` interface on every request struct:

```go
type Validator interface {
    Valid(ctx context.Context) map[string]string
}

func (r createRequest) Valid(ctx context.Context) map[string]string {
    errs := make(map[string]string)
    if r.Name == "" { errs["name"] = "name is required" }
    // add field-level checks here
    return errs
}

func (r updateRequest) Valid(ctx context.Context) map[string]string {
    // partial update — return nil if no required fields
    return nil
}
```

### `internal/<domain>/repository.go`

Unexported interface with CRUD methods. `inMemoryRepository` impl with `sync.RWMutex`.
Include a package-level sentinel: `var errNotFound = errors.New("<domain> not found")`.

### `internal/<domain>/service.go`

Unexported `<domain>Service` struct holding a `repo <domain>Repository`.
Constructor: `func new<Domain>Service(repo <domain>Repository) *<domain>Service`.

**Service does NOT repeat field-presence validation** — that lives in `model.go`. Only add business-rule validation here (e.g., uniqueness, state transitions).

Include `generateID()` helper using `crypto/rand`.

### `internal/<domain>/handler.go`

HTTP handler methods on `*Controller`. Use the `decode[T]()` generic helper and `writeJSON`/`writeError` helpers.

Handler pipeline — follow this exact order:

```go
func (c *Controller) create(w http.ResponseWriter, r *http.Request) {
    req, err := decode[createRequest](r)
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    if errs := req.Valid(r.Context()); len(errs) > 0 {
        writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": errs})
        return
    }
    result, err := c.svc.create(r.Context(), req)
    if err != nil {
        writeError(w, http.StatusInternalServerError, "internal error")
        return
    }
    writeJSON(w, http.StatusCreated, result)
}
```

Include the `decode[T]()` generic helper in this file:

```go
func decode[T any](r *http.Request) (T, error) {
    var v T
    if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
        return v, fmt.Errorf("decode json: %w", err)
    }
    return v, nil
}
```

Error response shape:
- Malformed JSON → 400 `{"error": "invalid request body"}`
- Validation failure → 422 `{"errors": {"field": "message"}}`
- Not found → 404 `{"error": "not found"}`
- Unexpected → 500 `{"error": "internal error"}`

### `internal/<domain>/controller.go`

```go
type Controller struct { svc *<domain>Service }

func NewController() *Controller {
    repo := newInMemoryRepository()
    svc  := new<Domain>Service(repo)
    return &Controller{svc: svc}
}

func (c *Controller) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("GET /<domain>s",        c.list)
    mux.HandleFunc("POST /<domain>s",       c.create)
    mux.HandleFunc("GET /<domain>s/{id}",   c.getByID)
    mux.HandleFunc("PUT /<domain>s/{id}",   c.update)
    mux.HandleFunc("DELETE /<domain>s/{id}", c.delete)
}
```

## Wiring

Add to the `run()` function in `cmd/api/main.go` (inside `mux` setup):
```go
<domain>.NewController().RegisterRoutes(mux)
```

If background jobs are needed, add to `cmd/worker/main.go`:
```go
<domain>.NewController().StartScheduler(ctx)
```
