package user

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

	pgdb "restful-boilerplate/pkg/postgres/db"
)

func newTestService(q pgdb.Querier) *userService {
	return newUserService(q, noop.NewTracerProvider().Tracer("user"))
}

func TestUserService_CreateUser_Success(t *testing.T) {
	q := &mockQuerier{}
	created := pgdb.User{ID: "abc", Name: "Alice", Email: "a@b.c", CreatedAt: time.Now()}
	q.On("CreateUser", mock.Anything, mock.MatchedBy(func(p pgdb.CreateUserParams) bool {
		return p.Name == "Alice" && p.Email == "a@b.c" && p.ID != "" && !p.CreatedAt.IsZero()
	})).Return(created, nil)

	svc := newTestService(q)
	got, err := svc.createUser(context.Background(), CreateUserRequest{Name: "Alice", Email: "a@b.c"})

	require.NoError(t, err)
	require.Equal(t, created.Name, got.Name)
	q.AssertExpectations(t)
}

func TestUserService_CreateUser_DBError(t *testing.T) {
	q := &mockQuerier{}
	boom := errors.New("boom")
	q.On("CreateUser", mock.Anything, mock.Anything).Return(pgdb.User{}, boom)

	svc := newTestService(q)
	_, err := svc.createUser(context.Background(), CreateUserRequest{Name: "A", Email: "a@b.c"})

	require.Error(t, err)
	require.False(t, errors.Is(err, ErrNotFound))
}

func TestUserService_GetUserByID(t *testing.T) {
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
			q.On("GetUserByID", mock.Anything, "id-1").Return(pgdb.User{ID: "id-1"}, tt.dbErr)

			_, err := newTestService(q).getUserByID(context.Background(), "id-1")

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Equal(t, tt.wantNotFnd, errors.Is(err, ErrNotFound))
		})
	}
}

func TestUserService_UpdateUser_NotFound(t *testing.T) {
	q := &mockQuerier{}
	q.On("UpdateUser", mock.Anything, mock.Anything).Return(pgdb.User{}, pgx.ErrNoRows)

	_, err := newTestService(q).updateUser(context.Background(), "id-1", UpdateUserRequest{Name: "A"})

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNotFound))
}

func TestUserService_DeleteUser(t *testing.T) {
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
			q.On("DeleteUser", mock.Anything, "id-1").Return(tt.tag, tt.dbErr)

			err := newTestService(q).deleteUser(context.Background(), "id-1")

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Equal(t, tt.wantNotFnd, errors.Is(err, ErrNotFound))
		})
	}
}

func TestUserService_ListUsers(t *testing.T) {
	q := &mockQuerier{}
	rows := []pgdb.User{{ID: "1"}, {ID: "2"}}
	q.On("ListUsers", mock.Anything).Return(rows, nil)

	got, err := newTestService(q).listUsers(context.Background())

	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "1", got[0].ID)
}
