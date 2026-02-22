package user

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	sqlitedb "restful-boilerplate/repo/sqlite/db"
)

type userService struct {
	q *sqlitedb.Queries
}

func (s *userService) createUser(ctx context.Context, in createUserInput) (*User, error) {
	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate id: %w", err)
	}
	row, err := s.q.CreateUser(ctx, sqlitedb.CreateUserParams{
		ID: id, Name: in.Name, Email: in.Email, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("createUser: %w", err)
	}
	return toUser(row), nil
}

func (s *userService) listUsers(ctx context.Context) ([]*User, error) {
	rows, err := s.q.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("listUsers: %w", err)
	}
	users := make([]*User, len(rows))
	for i, r := range rows {
		users[i] = toUser(r)
	}
	return users, nil
}

func (s *userService) getUserByID(ctx context.Context, id string) (*User, error) {
	row, err := s.q.GetUserByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getUserByID: %w", err)
	}
	return toUser(row), nil
}

func (s *userService) updateUser(ctx context.Context, id string, in updateUserInput) (*User, error) {
	existing, err := s.getUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != "" {
		existing.Name = in.Name
	}
	if in.Email != "" {
		existing.Email = in.Email
	}
	row, err := s.q.UpdateUser(ctx, sqlitedb.UpdateUserParams{
		ID: id, Name: existing.Name, Email: existing.Email,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("updateUser: %w", err)
	}
	return toUser(row), nil
}

func (s *userService) deleteUser(ctx context.Context, id string) error {
	result, err := s.q.DeleteUser(ctx, id)
	if err != nil {
		return fmt.Errorf("deleteUser: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return errNotFound
	}
	return nil
}

func generateID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func toUser(u sqlitedb.User) *User {
	return &User{ID: u.ID, Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt}
}
