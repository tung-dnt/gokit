package postgres

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/user.sql
var userSchema string

// Migrate applies all DDL migrations to the database.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, userSchema); err != nil {
		return fmt.Errorf("apply user schema: %w", err)
	}
	return nil
}
