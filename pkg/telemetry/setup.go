package telemetry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"restful-boilerplate/pkg/logger"
)

// SetupAll initialises OpenTelemetry tracing, metrics, log export, and structured log file output.
// logFormat controls stdout output: "pretty" for colorized, "json" (default) for JSON.
// Returns a single cleanup func that flushes all providers and closes the log file.
func SetupAll(ctx context.Context, logPath string, logFormat string) (func(), error) {
	res, err := NewResource(ctx)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	shutdownTracer, err := SetupTraces(ctx, res)
	if err != nil {
		return nil, err
	}

	shutdownMeter, err := SetupMeter(ctx, res)
	if err != nil {
		_ = shutdownTracer(ctx) //nolint:errcheck // best-effort cleanup on error path
		return nil, err
	}

	logProvider, shutdownLogs, err := SetupLogs(ctx, res)
	if err != nil {
		_ = shutdownMeter(ctx)  //nolint:errcheck // best-effort cleanup on error path
		_ = shutdownTracer(ctx) //nolint:errcheck // best-effort cleanup on error path
		return nil, err
	}

	if dir := filepath.Dir(logPath); dir != "." {
		if mkdirErr := os.MkdirAll(dir, 0o750); mkdirErr != nil {
			_ = shutdownLogs(ctx)   //nolint:errcheck // best-effort cleanup on error path
			_ = shutdownMeter(ctx)  //nolint:errcheck // best-effort cleanup on error path
			_ = shutdownTracer(ctx) //nolint:errcheck // best-effort cleanup on error path
			return nil, fmt.Errorf("create logs dir: %w", mkdirErr)
		}
	}

	closeLog, err := logger.Setup(logPath, logFormat, logProvider)
	if err != nil {
		_ = shutdownLogs(ctx)   //nolint:errcheck // best-effort cleanup on error path
		_ = shutdownMeter(ctx)  //nolint:errcheck // best-effort cleanup on error path
		_ = shutdownTracer(ctx) //nolint:errcheck // best-effort cleanup on error path
		return nil, fmt.Errorf("setup logger: %w", err)
	}

	return func() {
		_ = closeLog()          //nolint:errcheck // best-effort cleanup on shutdown
		_ = shutdownLogs(ctx)   //nolint:errcheck // best-effort cleanup on shutdown
		_ = shutdownMeter(ctx)  //nolint:errcheck // best-effort cleanup on shutdown
		_ = shutdownTracer(ctx) //nolint:errcheck // best-effort cleanup on shutdown
	}, nil
}
