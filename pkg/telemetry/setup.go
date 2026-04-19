package telemetry

import (
	"context"
	"fmt"

	"gokit/pkg/logger"
)

// SetupAll initialises OpenTelemetry tracing, metrics, and log export.
// logFormat controls stdout output: "pretty" for colorized, "json" (default) for JSON.
// Returns a single cleanup func that flushes all providers.
func SetupAll(ctx context.Context, logFormat string) (func(), error) {
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

	logger.Setup(logFormat, logProvider)

	return func() {
		_ = shutdownLogs(ctx)   //nolint:errcheck // best-effort cleanup on shutdown
		_ = shutdownMeter(ctx)  //nolint:errcheck // best-effort cleanup on shutdown
		_ = shutdownTracer(ctx) //nolint:errcheck // best-effort cleanup on shutdown
	}, nil
}
