package sqlite

import (
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed migrations/user.sql
var userSchema string

// Migrate applies all DDL migrations to the database.
func Migrate(db *sql.DB) error {
	if _, err := db.Exec(userSchema); err != nil {
		return fmt.Errorf("apply user schema: %w", err)
	}
	return nil
}
