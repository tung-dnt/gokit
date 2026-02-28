// Package testutil provides shared test helpers for integration tests.
package testutil

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	sqlite "restful-boilerplate/infra/sqlite"
)

// SetupTestDB opens an in-memory SQLite database, applies all migrations,
// and registers a cleanup function that closes the database when the test ends.
func SetupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	ctx := context.Background()

	db, err := sqlite.OpenDB(ctx, ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	if migrateErr := sqlite.Migrate(ctx, db); migrateErr != nil {
		db.Close() //nolint:errcheck,gosec // best-effort close on error path
		t.Fatalf("migrate test db: %v", migrateErr)
	}

	t.Cleanup(func() { db.Close() }) //nolint:errcheck,gosec // best-effort close in test cleanup

	return db
}
