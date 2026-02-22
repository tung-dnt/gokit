---
name: gen-test
description: Generate Go unit tests for a domain's service, handler, or model using table-driven tests and interface mocks
---

Generate idiomatic Go unit tests for a file within `internal/<domain>/`. No external test libraries — stdlib only (`testing`, `errors`, `net/http/httptest`, `strings`).

## Rules

- Tests live in the **same package** as the code under test (e.g., `package user`) to access unexported types.
- Use **table-driven tests**: `tests := []struct{ name string; ... }{ {...}, ... }` with `t.Run(tt.name, ...)`.
- Mock the repository interface **inline** — define a `mock<Domain>Repository` struct in the test file that implements the private interface.
- Use `t.Fatal` / `t.Errorf` — never `panic`.
- Test both happy path and error cases.
- Name test functions `Test<Type>_<method>` (e.g., `TestUserService_create`).

## Validation split — where tests live

Validation is split between two layers; test each in its own file:

| What | Where to test | File |
|------|--------------|------|
| Field presence / format | `createRequest.Valid()` | `model_test.go` |
| Business rules (uniqueness etc.) | `<domain>Service.*` | `service_test.go` |
| HTTP decode + Valid() + status codes | `Controller.*` | `handler_test.go` |

## Model test pattern (`model_test.go`)

```go
func TestCreateRequest_Valid(t *testing.T) {
    tests := []struct {
        name     string
        input    createRequest
        wantKeys []string // field names expected in error map
    }{
        {name: "valid", input: createRequest{Name: "Alice", Email: "a@b.com"}},
        {name: "missing name", input: createRequest{Email: "a@b.com"}, wantKeys: []string{"name"}},
        {name: "missing email", input: createRequest{Name: "Alice"}, wantKeys: []string{"email"}},
        {name: "missing both", input: createRequest{}, wantKeys: []string{"name", "email"}},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            errs := tt.input.Valid(context.Background())
            for _, k := range tt.wantKeys {
                if _, ok := errs[k]; !ok {
                    t.Errorf("expected error for field %q, got errors: %v", k, errs)
                }
            }
            if len(tt.wantKeys) == 0 && len(errs) != 0 {
                t.Errorf("expected no errors, got: %v", errs)
            }
        })
    }
}
```

## Service test pattern (`service_test.go`)

Service receives pre-validated requests; test business rules and repo interactions only.

```go
type mockUserRepository struct {
    createErr error
    getErr    error
    users     map[string]*User
}

func (m *mockUserRepository) create(_ context.Context, u *User) error  { return m.createErr }
func (m *mockUserRepository) getByID(_ context.Context, id string) (*User, error) {
    if m.getErr != nil { return nil, m.getErr }
    u, ok := m.users[id]
    if !ok { return nil, errNotFound }
    return u, nil
}
// ... implement remaining interface methods

func TestUserService_create(t *testing.T) {
    tests := []struct {
        name    string
        input   createRequest
        repoErr error
        wantErr bool
    }{
        {name: "success", input: createRequest{Name: "Alice", Email: "a@b.com"}},
        {name: "repo error", input: createRequest{Name: "A", Email: "a@b.com"}, repoErr: errors.New("db down"), wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            svc := newUserService(&mockUserRepository{createErr: tt.repoErr})
            _, err := svc.create(context.Background(), tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
            }
        })
    }
}
```

## Handler test pattern (`handler_test.go`)

Use `net/http/httptest` — covers the full decode → Valid() → service → encode pipeline.

```go
func TestController_create(t *testing.T) {
    tests := []struct {
        name       string
        body       string
        wantStatus int
    }{
        {name: "valid", body: `{"name":"Alice","email":"a@b.com"}`, wantStatus: http.StatusCreated},
        {name: "missing fields", body: `{"name":""}`, wantStatus: http.StatusUnprocessableEntity},
        {name: "malformed json", body: `{bad}`, wantStatus: http.StatusBadRequest},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mux := http.NewServeMux()
            NewController().RegisterRoutes(mux)

            req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(tt.body))
            req.Header.Set("Content-Type", "application/json")
            w := httptest.NewRecorder()
            mux.ServeHTTP(w, req)

            if w.Code != tt.wantStatus {
                t.Errorf("got %d, want %d — body: %s", w.Code, tt.wantStatus, w.Body.String())
            }
        })
    }
}
```

## Integration test pattern (`run_test.go` in `cmd/api/`)

Use the testable `run()` function to start a real server on a random port:

```go
func TestRun_healthz(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    getenv := func(key string) string {
        return map[string]string{
            "SERVER_PORT": "0", // OS picks a free port
        }[key]
    }

    errCh := make(chan error, 1)
    go func() { errCh <- run(ctx, io.Discard, getenv) }()

    // TODO: get actual port from server (requires addr to be returned from run())
    resp, err := http.Get("http://localhost:8080/healthz")
    if err != nil { t.Fatal(err) }
    if resp.StatusCode != http.StatusOK {
        t.Errorf("healthz: got %d, want 200", resp.StatusCode)
    }

    cancel()
    if err := <-errCh; err != nil {
        t.Errorf("run() error: %v", err)
    }
}
```

## What to generate

When invoked with a file argument (e.g., `/gen-test internal/user/service.go`), generate the corresponding test file:

- `model.go` → `model_test.go` (test all `Valid()` methods)
- `service.go` → `service_test.go` (test all service methods with mock repo)
- `handler.go` or `controller.go` → `handler_test.go` (test HTTP status codes via httptest)
