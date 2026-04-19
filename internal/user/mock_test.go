package user

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/mock"

	pgdb "restful-boilerplate/pkg/postgres/db"
)

type mockQuerier struct {
	mock.Mock
}

var _ pgdb.Querier = (*mockQuerier)(nil)

func (m *mockQuerier) CreateUser(ctx context.Context, arg pgdb.CreateUserParams) (pgdb.User, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(pgdb.User), args.Error(1)
}

func (m *mockQuerier) UpdateUser(ctx context.Context, arg pgdb.UpdateUserParams) (pgdb.User, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(pgdb.User), args.Error(1)
}

func (m *mockQuerier) DeleteUser(ctx context.Context, id string) (pgconn.CommandTag, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *mockQuerier) ListUsers(ctx context.Context) ([]pgdb.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]pgdb.User), args.Error(1)
}

func (m *mockQuerier) GetUserByID(ctx context.Context, id string) (pgdb.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(pgdb.User), args.Error(1)
}

type mockUserSvc struct {
	mock.Mock
}

var _ userSvc = (*mockUserSvc)(nil)

func (m *mockUserSvc) createUser(ctx context.Context, in CreateUserRequest) (*pgdb.User, error) {
	args := m.Called(ctx, in)
	u, _ := args.Get(0).(*pgdb.User)
	return u, args.Error(1)
}

func (m *mockUserSvc) updateUser(ctx context.Context, id string, in UpdateUserRequest) (*pgdb.User, error) {
	args := m.Called(ctx, id, in)
	u, _ := args.Get(0).(*pgdb.User)
	return u, args.Error(1)
}

func (m *mockUserSvc) deleteUser(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockUserSvc) listUsers(ctx context.Context) ([]*pgdb.User, error) {
	args := m.Called(ctx)
	u, _ := args.Get(0).([]*pgdb.User)
	return u, args.Error(1)
}

func (m *mockUserSvc) getUserByID(ctx context.Context, id string) (*pgdb.User, error) {
	args := m.Called(ctx, id)
	u, _ := args.Get(0).(*pgdb.User)
	return u, args.Error(1)
}
