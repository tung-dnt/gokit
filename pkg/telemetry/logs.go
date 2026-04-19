package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

// SetupLogs initialises an OTLP LoggerProvider and returns the provider and a shutdown function.
// The returned provider is passed to the logger package to bridge slog → OTLP.
func SetupLogs(ctx context.Context, res *resource.Resource) (*sdklog.LoggerProvider, func(context.Context) error, error) {
	opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(Endpoint())}
	if insecure() {
		opts = append(opts, otlploggrpc.WithInsecure())
	}

	exp, err := otlploggrpc.New(ctx, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("otlplog exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)),
	)

	return lp, lp.Shutdown, nil
}
