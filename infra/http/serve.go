package router

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// GracefulServe starts srv in a goroutine and blocks until ctx is cancelled,
// then shuts down gracefully within the given timeout.
func GracefulServe(ctx context.Context, srv *http.Server, shutdownTimeout time.Duration) {
	go func() {
		slog.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
}
