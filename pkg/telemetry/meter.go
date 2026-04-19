package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

// SetupMeter initialises an OTLP MeterProvider with Go runtime metrics and returns a shutdown function.
// Forces delta temporality so SigNoz/ClickHouse aggregates counters and histograms correctly
// (Go SDK defaults to cumulative; SigNoz pipelines expect delta).
func SetupMeter(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(Endpoint()),
		otlpmetricgrpc.WithTemporalitySelector(func(sdkmetric.InstrumentKind) metricdata.Temporality {
			return metricdata.DeltaTemporality
		}),
	}
	if insecure() {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}
	exp, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("otlpmetric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp,
			sdkmetric.WithInterval(10*time.Second),
		)),
	)
	otel.SetMeterProvider(mp)

	if rtErr := runtime.Start(runtime.WithMeterProvider(mp)); rtErr != nil {
		_ = mp.Shutdown(ctx) //nolint:errcheck // best-effort cleanup on error path
		return nil, fmt.Errorf("runtime metrics: %w", rtErr)
	}

	return mp.Shutdown, nil
}
