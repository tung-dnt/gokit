// Package postgres provides PostgreSQL connection management.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OpenDB creates and validates a pgxpool connection pool for the given DSN.
func OpenDB(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil
}
