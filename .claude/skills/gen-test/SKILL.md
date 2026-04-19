---
name: gen-test
description: Generate Go unit and integration tests for handlers, service, or DTOs using table-driven tests, net/http httptest, and PostgreSQL
---

Generate idiomatic Go unit tests for the flat domain package structure. The default is **mock-based unit tests** — no live database. Uses `github.com/stretchr/testify/mock` + `require` for assertions.

## Test Layers

The project uses two complementary test layers per domain:

| Layer | File | Mocks | What it proves |
|-------|------|-------|----------------|
| Service | `internal/<domain>/service.<domain>_test.go` | `mockQuerier` (satisfies `pgdb.Querier`) | Business logic, `pgx.ErrNoRows` → `ErrNotFound`, `RowsAffected() == 0` → `ErrNotFound`, span error classification |
| Adapter | `internal/<domain>/adapter.http_test.go` | `mock<Domain>Svc` (satisfies `<domain>Svc` interface) | HTTP status codes, `router.Bind` decode/validate failures, `writeErr` mapping |
| DTO | `internal/<domain>/domain.dto_test.go` | none | `validator` tag behaviour |

Both mocks live in a single `internal/<domain>/mock_test.go` file.

---

## Rules

- All tests use `package <domain>` (internal test package — they exercise unexported types like `userSvc`, `userService`, `httpAdapter`).
- Use `github.com/stretchr/testify/mock` + `github.com/stretchr/testify/require`.
- Use **table-driven tests**: `tests := []struct{ name string; ... }{ ... }` with `t.Run(tt.name, ...)`.
- Service tests inject `noop.NewTracerProvider().Tracer("<domain>")` via a small `newTestService` helper.
- Adapter tests inject a real `validator.New()` (bind logic is what's being tested).
- Tests ARE parallel-safe — no shared DB. Feel free to `t.Parallel()` on each subtest.
- For `DeleteXxx` results, build a fake tag with `pgconn.NewCommandTag("DELETE 1")` / `pgconn.NewCommandTag("DELETE 0")`.
- For `GetXxxByID` not-found paths, return `pgx.ErrNoRows` from the mock.
- Legacy `pkg/testutil/SetupPgTestDB` is still available for the rare case where you need a real database — but the default for new tests is mocks.

---

## File argument

When invoked with a file path, generate the corresponding test file:

| Source file | Test file | What to test |
|-------------|-----------|--------------|
| `internal/<domain>/domain.dto.go` | `internal/<domain>/domain.dto_test.go` | Validator tag behaviour |
| `internal/<domain>/service.<domain>.go` | `internal/<domain>/service.<domain>_test.go` | CRUD methods + error mapping, with `mockQuerier` |
| `internal/<domain>/adapter.http.go` | `internal/<domain>/adapter.http_test.go` | HTTP status codes + `writeErr`, with `mock<Domain>Svc` |

If `mock_test.go` doesn't exist yet, create it first (see next section).

---

## `mock_test.go` — shared mocks

One file per domain. Defines `mockQuerier` (for service tests) and `mock<Domain>Svc` (for adapter tests). Each mock has a compile-time interface assertion so a missing method is a build error.

```go
package <domain>

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"

	pgdb "gokit/pkg/postgres/db"
)

// ----- mockQuerier: satisfies pgdb.Querier -----

type mockQuerier struct {
	mock.Mock
}

var _ pgdb.Querier = (*mockQuerier)(nil)

func (m *mockQuerier) Create<Domain>(ctx context.Context, arg pgdb.Create<Domain>Params) (pgdb.<Domain>, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(pgdb.<Domain>), args.Error(1)
}

func (m *mockQuerier) Update<Domain>(ctx context.Context, arg pgdb.Update<Domain>Params) (pgdb.<Domain>, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(pgdb.<Domain>), args.Error(1)
}

func (m *mockQuerier) Delete<Domain>(ctx context.Context, id string) (pgconn.CommandTag, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *mockQuerier) List<Domain>s(ctx context.Context) ([]pgdb.<Domain>, error) {
	args := m.Called(ctx)
	return args.Get(0).([]pgdb.<Domain>), args.Error(1)
}

func (m *mockQuerier) Get<Domain>ByID(ctx context.Context, id string) (pgdb.<Domain>, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(pgdb.<Domain>), args.Error(1)
}

// ----- mock<Domain>Svc: satisfies <domain>Svc -----

type mock<Domain>Svc struct {
	mock.Mock
}

var _ <domain>Svc = (*mock<Domain>Svc)(nil)

func (m *mock<Domain>Svc) create<Domain>(ctx context.Context, in Create<Domain>Request) (*pgdb.<Domain>, error) {
	args := m.Called(ctx, in)
	d, _ := args.Get(0).(*pgdb.<Domain>)
	return d, args.Error(1)
}

func (m *mock<Domain>Svc) update<Domain>(ctx context.Context, id string, in Update<Domain>Request) (*pgdb.<Domain>, error) {
	args := m.Called(ctx, id, in)
	d, _ := args.Get(0).(*pgdb.<Domain>)
	return d, args.Error(1)
}

func (m *mock<Domain>Svc) delete<Domain>(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mock<Domain>Svc) list<Domain>s(ctx context.Context) ([]*pgdb.<Domain>, error) {
	args := m.Called(ctx)
	d, _ := args.Get(0).([]*pgdb.<Domain>)
	return d, args.Error(1)
}

func (m *mock<Domain>Svc) get<Domain>ByID(ctx context.Context, id string) (*pgdb.<Domain>, error) {
	args := m.Called(ctx, id)
	d, _ := args.Get(0).(*pgdb.<Domain>)
	return d, args.Error(1)
}
```

The `var _ pgdb.Querier = (*mockQuerier)(nil)` line is the **interface assertion** — if sqlc adds a new query, the build breaks here until you add the new method.

---

## Service tests (`service.<domain>_test.go`)

Inject `noop.NewTracerProvider().Tracer("<domain>")` via a tiny helper. Use `mockQuerier`. Cover happy path + DB error + `pgx.ErrNoRows` → `ErrNotFound` + `RowsAffected() == 0` → `ErrNotFound`.

```go
package <domain>

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	pgdb "gokit/pkg/postgres/db"
)

func newTestService(q pgdb.Querier) *<domain>Service {
	return new<Domain>Service(q, noop.NewTracerProvider().Tracer("<domain>"))
}

func TestService_Create_Success(t *testing.T) {
	q := &mockQuerier{}
	created := pgdb.<Domain>{ID: "abc", Name: "Alice", CreatedAt: time.Now()}
	q.On("Create<Domain>", mock.Anything, mock.MatchedBy(func(p pgdb.Create<Domain>Params) bool {
		return p.Name == "Alice" && p.ID != "" && !p.CreatedAt.IsZero()
	})).Return(created, nil)

	got, err := newTestService(q).create<Domain>(context.Background(), Create<Domain>Request{Name: "Alice"})

	require.NoError(t, err)
	require.Equal(t, created.Name, got.Name)
	q.AssertExpectations(t)
}

func TestService_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		dbErr      error
		wantNotFnd bool
		wantErr    bool
	}{
		{"found", nil, false, false},
		{"not found maps ErrNoRows", pgx.ErrNoRows, true, true},
		{"generic db error wraps", errors.New("conn reset"), false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &mockQuerier{}
			q.On("Get<Domain>ByID", mock.Anything, "id-1").Return(pgdb.<Domain>{ID: "id-1"}, tt.dbErr)

			_, err := newTestService(q).get<Domain>ByID(context.Background(), "id-1")

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Equal(t, tt.wantNotFnd, errors.Is(err, ErrNotFound))
		})
	}
}

func TestService_Delete(t *testing.T) {
	tests := []struct {
		name       string
		tag        pgconn.CommandTag
		dbErr      error
		wantNotFnd bool
		wantErr    bool
	}{
		{"deleted", pgconn.NewCommandTag("DELETE 1"), nil, false, false},
		{"zero rows maps ErrNotFound", pgconn.NewCommandTag("DELETE 0"), nil, true, true},
		{"db error wraps", pgconn.CommandTag{}, errors.New("conn"), false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &mockQuerier{}
			q.On("Delete<Domain>", mock.Anything, "id-1").Return(tt.tag, tt.dbErr)

			err := newTestService(q).delete<Domain>(context.Background(), "id-1")

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Equal(t, tt.wantNotFnd, errors.Is(err, ErrNotFound))
		})
	}
}
```

---

## Adapter tests (`adapter.http_test.go`)

Construct a `*httpAdapter` with `mock<Domain>Svc` + real `validator.New()`. Exercise handlers via `net/http/httptest`. Set path params with `req.SetPathValue("id", "id-1")`.

```go
package <domain>

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	pgdb "gokit/pkg/postgres/db"
	"gokit/pkg/validator"
)

func newTestAdapter(svc <domain>Svc) *httpAdapter {
	return newHTTPAdapter(svc, validator.New())
}

func decodeBody[T any](t *testing.T, r io.Reader) T {
	t.Helper()
	var v T
	require.NoError(t, json.NewDecoder(r).Decode(&v))
	return v
}

func TestCreate<Domain>Handler_Created(t *testing.T) {
	svc := &mock<Domain>Svc{}
	now := time.Now()
	svc.On("create<Domain>", mock.Anything, Create<Domain>Request{Name: "Alice"}).
		Return(&pgdb.<Domain>{ID: "id-1", Name: "Alice", CreatedAt: now}, nil)

	body := bytes.NewBufferString(`{"name":"Alice"}`)
	req := httptest.NewRequest(http.MethodPost, "/<domain>s", body)
	rec := httptest.NewRecorder()

	newTestAdapter(svc).create<Domain>Handler(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	got := decodeBody[<domain>Response](t, rec.Body)
	require.Equal(t, "id-1", got.ID)
	svc.AssertExpectations(t)
}

func TestCreate<Domain>Handler_MalformedJSON(t *testing.T) {
	svc := &mock<Domain>Svc{}
	req := httptest.NewRequest(http.MethodPost, "/<domain>s", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()

	newTestAdapter(svc).create<Domain>Handler(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	svc.AssertNotCalled(t, "create<Domain>")
}

func TestCreate<Domain>Handler_ValidationError(t *testing.T) {
	svc := &mock<Domain>Svc{}
	req := httptest.NewRequest(http.MethodPost, "/<domain>s", strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()

	newTestAdapter(svc).create<Domain>Handler(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	svc.AssertNotCalled(t, "create<Domain>")
}

func TestGet<Domain>ByIDHandler(t *testing.T) {
	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{"ok", nil, http.StatusOK},
		{"not found", ErrNotFound, http.StatusNotFound},
		{"internal", errors.New("db down"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mock<Domain>Svc{}
			if tt.svcErr == nil {
				svc.On("get<Domain>ByID", mock.Anything, "id-1").Return(&pgdb.<Domain>{ID: "id-1"}, nil)
			} else {
				svc.On("get<Domain>ByID", mock.Anything, "id-1").Return((*pgdb.<Domain>)(nil), tt.svcErr)
			}

			req := httptest.NewRequest(http.MethodGet, "/<domain>s/id-1", nil)
			req.SetPathValue("id", "id-1")
			rec := httptest.NewRecorder()

			newTestAdapter(svc).get<Domain>ByIDHandler(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// Ensure context-based logger lookup does not panic when ctx has no logger.
func TestWriteErr_NoLoggerInContext_DoesNotPanic(t *testing.T) {
	a := newTestAdapter(&mock<Domain>Svc{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(context.Background())
	a.writeErr(req, rec, errors.New("x"))
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
```

---

## DTO tests (`domain.dto_test.go`)

No DB, no mocks — just exercise the validator tags.

```go
package <domain>

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestCreate<Domain>Request(t *testing.T) {
	t.Parallel()
	v := validator.New()
	tests := []struct {
		name    string
		input   Create<Domain>Request
		wantErr bool
	}{
		{name: "valid", input: Create<Domain>Request{Name: "Alice"}},
		{name: "missing name", input: Create<Domain>Request{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := v.Struct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}
```

---

## Run tests

```bash
# All domain tests — no DB required (mocks only)
go test ./internal/<domain>/...

# Full suite
go test ./... -count=1
```

---

## Legacy / rare case: live-DB tests

If you genuinely need a real PostgreSQL (e.g. verifying a sqlc-generated query against the real engine), `pkg/testutil/SetupPgTestDB(t)` still exists. It skips when `TEST_DATABASE_URL` is unset. Prefer mocks for everything else — they're faster, deterministic, and parallel-safe.

```bash
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" go test ./... -count=1
```
