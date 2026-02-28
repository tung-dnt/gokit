// Package user implements the application use-cases for user management.
package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"restful-boilerplate/infra/logger"
)

// UserSvc orchestrates user use-cases on top of a Repository.
type UserSvc struct {
	repo   Repository
	tracer trace.Tracer
}

// NewService creates a Service backed by repo and traced via tracer.
func NewService(repo Repository, tracer trace.Tracer) *UserSvc {
	return &UserSvc{repo: repo, tracer: tracer}
}

// CreateUser creates a new user from the given input.
func (s *UserSvc) CreateUser(ctx context.Context, in CreateUserInput) (*User, error) {
	ctx, span := s.tracer.Start(ctx, "user.CreateUser")
	defer span.End()

	logger.FromContext(ctx).Info("creating user", slog.String("email", in.Email))

	id, err := generateID()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("generate id: %w", err)
	}

	u := &User{
		ID: id, Name: in.Name, Email: in.Email, CreatedAt: time.Now().UTC(),
	}
	if err = s.repo.Create(ctx, u); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("createUser: %w", err)
	}
	return u, nil
}

// ListUsers returns all users.
func (s *UserSvc) ListUsers(ctx context.Context) ([]*User, error) {
	ctx, span := s.tracer.Start(ctx, "user.ListUsers")
	defer span.End()

	users, err := s.repo.List(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("listUsers: %w", err)
	}
	return users, nil
}

// GetUserByID returns a single user or ErrNotFound.
func (s *UserSvc) GetUserByID(ctx context.Context, id string) (*User, error) {
	ctx, span := s.tracer.Start(ctx, "user.GetUserByID")
	defer span.End()

	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if !isNotFound(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return nil, err
	}
	return u, nil
}

// UpdateUser applies a partial update to the user identified by id.
func (s *UserSvc) UpdateUser(ctx context.Context, id string, in UpdateUserInput) (*User, error) {
	ctx, span := s.tracer.Start(ctx, "user.UpdateUser")
	defer span.End()

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != "" {
		existing.Name = in.Name
	}
	if in.Email != "" {
		existing.Email = in.Email
	}
	if err = s.repo.Update(ctx, existing); err != nil {
		if !isNotFound(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return nil, err
	}
	return existing, nil
}

// DeleteUser removes a user by id or returns ErrNotFound.
func (s *UserSvc) DeleteUser(ctx context.Context, id string) error {
	ctx, span := s.tracer.Start(ctx, "user.DeleteUser")
	defer span.End()

	if err := s.repo.Delete(ctx, id); err != nil {
		if !isNotFound(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
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

func isNotFound(err error) bool {
	return err != nil && err == ErrNotFound
}
