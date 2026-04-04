package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed migrations/user.sql
var userSchema string

// Migrate applies all DDL migrations to the database.
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, userSchema); err != nil {
		return fmt.Errorf("apply user schema: %w", err)
	}
	return nil
}
