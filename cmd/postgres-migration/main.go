package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"gokit/pkg/config"
	"gokit/pkg/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg := config.Load(os.Getenv)

	pool, err := postgres.OpenDB(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer pool.Close()

	if err = postgres.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	slog.Info("migrations applied successfully")
	return nil
}
