package user

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/trace"

	"restful-boilerplate/pkg/logger"
	pgdb "restful-boilerplate/pkg/postgres/db"
	"restful-boilerplate/pkg/telemetry"
	shared "restful-boilerplate/pkg/util"
)

type userService struct {
	db     *pgdb.Queries
	tracer trace.Tracer
}

func newUserService(q *pgdb.Queries, tracer trace.Tracer) *userService {
	return &userService{db: q, tracer: tracer}
}

func (s *userService) createUser(ctx context.Context, in CreateUserRequest) (*pgdb.User, error) {
	ctx, span := s.tracer.Start(ctx, "user.userService.createUser")
	defer span.End()

	logger.FromContext(ctx).InfoContext(ctx, "creating user", slog.String("email", in.Email))

	id, err := shared.GenerateID()
	if err != nil {
		return nil, telemetry.SpanErr(span, err, "generate id")
	}

	row, err := s.db.CreateUser(ctx, pgdb.CreateUserParams{
		ID:        id,
		Name:      in.Name,
		Email:     in.Email,
		CreatedAt: time.Now(),
	})
	if err != nil {
		return nil, telemetry.SpanErr(span, err, "user.userService.createUser")
	}
	return &row, nil
}

func (s *userService) updateUser(ctx context.Context, id string, in UpdateUserRequest) (*pgdb.User, error) {
	ctx, span := s.tracer.Start(ctx, "user.userService.updateUser")
	defer span.End()

	row, err := s.db.UpdateUser(ctx, pgdb.UpdateUserParams{
		ID:    id,
		Name:  in.Name,
		Email: in.Email,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, telemetry.SpanErr(span, err, "user.userService.updateUser")
	}
	return &row, nil
}

func (s *userService) deleteUser(ctx context.Context, id string) error {
	ctx, span := s.tracer.Start(ctx, "user.userService.deleteUser")
	defer span.End()

	result, err := s.db.DeleteUser(ctx, id)
	if err != nil {
		return telemetry.SpanErr(span, err, "user.userService.deleteUser")
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *userService) listUsers(ctx context.Context) ([]*pgdb.User, error) {
	ctx, span := s.tracer.Start(ctx, "user.userService.listUsers")
	defer span.End()

	rows, err := s.db.ListUsers(ctx)
	if err != nil {
		return nil, telemetry.SpanErr(span, err, "user.userService.listUsers")
	}

	users := make([]*pgdb.User, 0, len(rows))
	for i := range rows {
		users = append(users, &rows[i])
	}
	return users, nil
}

func (s *userService) getUserByID(ctx context.Context, id string) (*pgdb.User, error) {
	ctx, span := s.tracer.Start(ctx, "user.userService.getUserByID")
	defer span.End()

	row, err := s.db.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, telemetry.SpanErr(span, err, "user.userService.getUserByID")
	}
	return &row, nil
}
