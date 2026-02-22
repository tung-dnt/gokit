// Package main is the background worker entrypoint.
// It runs independently of the HTTP API and can import the same domain controllers.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"restful-boilerplate/pkg/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load(os.Getenv)
	_ = cfg

	// Domain controllers can be imported here the same way as in cmd/api.
	// Example: if a domain exposes a job scheduler or event consumer,
	// it would be wired here instead of (or alongside) RegisterRoutes.
	//
	// user.NewController().StartScheduler(ctx)
	// notification.NewController().ConsumeEvents(ctx)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger.Info("worker starting")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopped")
			return
		case t := <-ticker.C:
			logger.Info("worker tick", "time", t)
		}
	}
}
