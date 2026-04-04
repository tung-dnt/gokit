package usercore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"restful-boilerplate/internal/shared"
	"restful-boilerplate/pkg/logger"
	sqlitedb "restful-boilerplate/pkg/sqlite/db"
)

// Service orchestrates user use-cases on top of a sqlitedb.Queries.
type Service struct {
	q      *sqlitedb.Queries
	tracer trace.Tracer
}

// NewService creates a Svc backed by q and traced via tracer.
func NewService(q *sqlitedb.Queries, tracer trace.Tracer) *Service {
	return &Service{q: q, tracer: tracer}
}

// CreateUser creates a new user from the given input.
func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (*sqlitedb.User, error) {
	ctx, span := s.tracer.Start(ctx, "user.CreateUser")
	defer span.End()

	logger.FromContext(ctx).Info("creating user", slog.String("email", in.Email))

	id, err := shared.GenerateID()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("generate id: %w", err)
	}

	row, err := s.q.CreateUser(ctx, sqlitedb.CreateUserParams{
		ID:        id,
		Name:      in.Name,
		Email:     in.Email,
		CreatedAt: time.Now(),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("createUser: %w", err)
	}
	return &row, nil
}

// UpdateUser applies a partial update to the user identified by id.
func (s *Service) UpdateUser(ctx context.Context, id string, in UpdateUserInput) (*sqlitedb.User, error) {
	ctx, span := s.tracer.Start(ctx, "user.UpdateUser")
	defer span.End()

	existing, err := s.GetUserByID(ctx, id)
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
		ID:    id,
		Name:  existing.Name,
		Email: existing.Email,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("updateUser: %w", err)
	}
	return &row, nil
}

// DeleteUser removes a user by id or returns ErrNotFound.
func (s *Service) DeleteUser(ctx context.Context, id string) error {
	ctx, span := s.tracer.Start(ctx, "user.DeleteUser")
	defer span.End()

	result, err := s.q.DeleteUser(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("deleteUser: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleteUser rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListUsers returns all users.
func (s *Service) ListUsers(ctx context.Context) ([]*sqlitedb.User, error) {
	ctx, span := s.tracer.Start(ctx, "user.ListUsers")
	defer span.End()

	rows, err := s.q.ListUsers(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("listUsers: %w", err)
	}

	users := make([]*sqlitedb.User, 0, len(rows))
	for i := range rows {
		users = append(users, &rows[i])
	}
	return users, nil
}

// GetUserByID returns a single user or ErrNotFound.
func (s *Service) GetUserByID(ctx context.Context, id string) (*sqlitedb.User, error) {
	ctx, span := s.tracer.Start(ctx, "user.GetUserByID")
	defer span.End()

	row, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("getUserByID: %w", err)
	}
	return &row, nil
}
