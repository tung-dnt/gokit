package telemetry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"restful-boilerplate/infra/logger"
)

// SetupAll initialises OpenTelemetry tracing and structured log file output.
// Returns a single cleanup func that flushes spans and closes the log file.
func SetupAll(ctx context.Context, logPath string) (func(), error) {
	shutdownTracer, err := Setup(ctx)
	if err != nil {
		return nil, err
	}

	if dir := filepath.Dir(logPath); dir != "." {
		if mkdirErr := os.MkdirAll(dir, 0o750); mkdirErr != nil {
			_ = shutdownTracer(ctx) //nolint:errcheck // best-effort cleanup on error path
			return nil, fmt.Errorf("create logs dir: %w", mkdirErr)
		}
	}

	closeLog, err := logger.Setup(logPath)
	if err != nil {
		_ = shutdownTracer(ctx) //nolint:errcheck // best-effort cleanup on error path
		return nil, fmt.Errorf("setup logger: %w", err)
	}

	return func() {
		_ = closeLog()          //nolint:errcheck // best-effort cleanup on shutdown
		_ = shutdownTracer(ctx) //nolint:errcheck // best-effort cleanup on shutdown
	}, nil
}
