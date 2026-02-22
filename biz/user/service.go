package user

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"restful-boilerplate/pkg/logger"
	sqlitedb "restful-boilerplate/repo/sqlite/db"
)

type userService struct {
	q      *sqlitedb.Queries
	tracer trace.Tracer
}

func (s *userService) createUser(ctx context.Context, in createUserInput) (*User, error) {
	ctx, span := s.tracer.Start(ctx, "userService.createUser")
	defer span.End()

	logger.FromContext(ctx).Info("creating user", slog.String("email", in.Email))

	id, err := generateID()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("generate id: %w", err)
	}
	row, err := s.q.CreateUser(ctx, sqlitedb.CreateUserParams{
		ID: id, Name: in.Name, Email: in.Email, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("createUser: %w", err)
	}
	return toUser(row), nil
}

func (s *userService) listUsers(ctx context.Context) ([]*User, error) {
	ctx, span := s.tracer.Start(ctx, "userService.listUsers")
	defer span.End()

	rows, err := s.q.ListUsers(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("listUsers: %w", err)
	}
	users := make([]*User, len(rows))
	for i, r := range rows {
		users[i] = toUser(r)
	}
	return users, nil
}

func (s *userService) getUserByID(ctx context.Context, id string) (*User, error) {
	ctx, span := s.tracer.Start(ctx, "userService.getUserByID")
	defer span.End()

	row, err := s.q.GetUserByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errNotFound
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("getUserByID: %w", err)
	}
	return toUser(row), nil
}

func (s *userService) updateUser(ctx context.Context, id string, in updateUserInput) (*User, error) {
	ctx, span := s.tracer.Start(ctx, "userService.updateUser")
	defer span.End()

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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("updateUser: %w", err)
	}
	return toUser(row), nil
}

func (s *userService) deleteUser(ctx context.Context, id string) error {
	ctx, span := s.tracer.Start(ctx, "userService.deleteUser")
	defer span.End()

	result, err := s.q.DeleteUser(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
