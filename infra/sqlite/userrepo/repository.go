// Package userrepo implements domain/user.Repository backed by SQLite.
package userrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"restful-boilerplate/domain/user"
	sqlitedb "restful-boilerplate/infra/sqlite/db"
)

// SQLite implements user.Repository using sqlc-generated queries.
type SQLite struct {
	q *sqlitedb.Queries
}

// NewSQLite returns a Repository adapter backed by db.
func NewSQLite(db *sql.DB) *SQLite {
	return &SQLite{q: sqlitedb.New(db)}
}

// Create inserts a new user into SQLite and updates u with the stored values.
func (r *SQLite) Create(ctx context.Context, u *user.User) error {
	row, err := r.q.CreateUser(ctx, sqlitedb.CreateUserParams{
		ID: u.ID, Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	*u = *toUser(row)
	return nil
}

// List returns all users ordered by creation time.
func (r *SQLite) List(ctx context.Context) ([]*user.User, error) {
	rows, err := r.q.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	users := make([]*user.User, len(rows))
	for i, row := range rows {
		users[i] = toUser(row)
	}
	return users, nil
}

// GetByID returns a user by ID or user.ErrNotFound.
func (r *SQLite) GetByID(ctx context.Context, id string) (*user.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return toUser(row), nil
}

// Update persists the changes in u to SQLite and refreshes u with stored values.
func (r *SQLite) Update(ctx context.Context, u *user.User) error {
	row, err := r.q.UpdateUser(ctx, sqlitedb.UpdateUserParams{
		ID: u.ID, Name: u.Name, Email: u.Email,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return user.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	*u = *toUser(row)
	return nil
}

// Delete removes a user by ID or returns user.ErrNotFound.
func (r *SQLite) Delete(ctx context.Context, id string) error {
	result, err := r.q.DeleteUser(ctx, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return fmt.Errorf("delete user rows affected: %w", rowsErr)
	}
	if n == 0 {
		return user.ErrNotFound
	}
	return nil
}

func toUser(u sqlitedb.User) *user.User {
	return &user.User{ID: u.ID, Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt}
}
