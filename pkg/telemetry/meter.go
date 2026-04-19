package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
)

// SetupMeter initialises an OTLP MeterProvider with Go runtime metrics and returns a shutdown function.
func SetupMeter(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	exp, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(Endpoint()))
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
