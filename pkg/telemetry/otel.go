// Package telemetry initialises the OpenTelemetry providers (traces, metrics, logs).
package telemetry

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"restful-boilerplate/pkg/version"
)

// Endpoint returns the OTLP gRPC endpoint (host:port) from
// OTEL_EXPORTER_OTLP_ENDPOINT or the default. SigNoz best practice is gRPC
// on :4317 for better batching and compression than HTTP.
func Endpoint() string {
	if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" {
		return v
	}
	return "localhost:4317"
}

// EndpointHTTP returns the OTLP HTTP endpoint URL for components that only
// support HTTP transport (e.g. Genkit's OTEL plugin). Reads
// OTEL_EXPORTER_OTLP_HTTP_ENDPOINT or falls back to http://localhost:4318.
func EndpointHTTP() string {
	if v := os.Getenv("OTEL_EXPORTER_OTLP_HTTP_ENDPOINT"); v != "" {
		return v
	}
	return "http://localhost:4318"
}

// insecure reports whether OTLP exporters should skip TLS. Defaults to true
// for self-hosted SigNoz on plaintext :4317; set OTEL_EXPORTER_OTLP_INSECURE=false
// for SigNoz Cloud or any TLS-fronted collector.
func insecure() bool {
	v := os.Getenv("OTEL_EXPORTER_OTLP_INSECURE")
	if v == "" {
		return true
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return b
}

// sampleRatio returns the head-based trace sample ratio in [0, 1].
// OTEL_TRACES_SAMPLER_ARG=0.1 keeps 10% of traces; defaults to 1.0 (sample all).
func sampleRatio() float64 {
	v := os.Getenv("OTEL_TRACES_SAMPLER_ARG")
	if v == "" {
		return 1.0
	}
	r, err := strconv.ParseFloat(v, 64)
	if err != nil || r < 0 || r > 1 {
		return 1.0
	}
	return r
}

// NewResource creates the shared OTEL resource used by all providers.
// service.name comes from OTEL_SERVICE_NAME (or OTEL_RESOURCE_ATTRIBUTES) via
// WithFromEnv so deployments can override it without a rebuild.
// service.version is baked in from build-time -ldflags via the version package.
func NewResource(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
		resource.WithAttributes(
			semconv.ServiceVersion(version.Version),
		),
	)
}

// SetupTraces initialises a global OTLP TracerProvider and returns a shutdown function.
func SetupTraces(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(Endpoint())}
	if insecure() {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	exp, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("otlptrace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio()))),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}
