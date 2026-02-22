package telemetry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"restful-boilerplate/pkg/logger"
)

// SetupAll initialises OpenTelemetry tracing and structured log file output.
// Returns a single cleanup func that flushes spans and closes the log file.
func SetupAll(ctx context.Context, logPath string) (func(), error) {
	shutdownTracer, err := Setup(ctx)
	if err != nil {
		return nil, err
	}

	if dir := filepath.Dir(logPath); dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil { //nolint:gosec // 0750 intentional
			_ = shutdownTracer(ctx)
			return nil, fmt.Errorf("create logs dir: %w", err)
		}
	}

	closeLog, err := logger.Setup(logPath)
	if err != nil {
		_ = shutdownTracer(ctx)
		return nil, fmt.Errorf("setup logger: %w", err)
	}

	return func() {
		_ = closeLog()
		_ = shutdownTracer(ctx)
	}, nil
}
