package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"restful-boilerplate/pkg/postgres"
)

// SetupPgTestDB opens a PostgreSQL connection pool for integration tests.
// It requires TEST_DATABASE_URL to be set; tests are skipped when absent.
// Each call truncates the users table so tests start with a clean state.
func SetupPgTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping PostgreSQL integration test")
	}

	ctx := context.Background()

	pool, err := postgres.OpenDB(ctx, dsn)
	if err != nil {
		t.Fatalf("open pg test db: %v", err)
	}

	if err = postgres.Migrate(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("migrate pg test db: %v", err)
	}

	// Clean state before each test so parallel-unsafe tests don't bleed.
	if _, err = pool.Exec(ctx, "DELETE FROM users"); err != nil {
		pool.Close()
		t.Fatalf("truncate users: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}
