package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

// OpenDB opens and configures a SQLite database at the given path.
// Single connection serialises access at the Go level, preventing SQLITE_BUSY
// between goroutines. busy_timeout is per-connection so must be set on every open.
func OpenDB(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1)
	if _, execErr := db.ExecContext(ctx, `PRAGMA busy_timeout=5000;`); execErr != nil {
		_ = db.Close() //nolint:errcheck // best-effort close on error path
		return nil, fmt.Errorf("configure db: %w", execErr)
	}

	return db, nil
}
