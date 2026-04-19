// Package telemetry initialises the OpenTelemetry providers (traces, metrics, logs).
package telemetry

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Endpoint returns the OTLP endpoint from OTEL_EXPORTER_OTLP_ENDPOINT or the default.
func Endpoint() string {
	if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" {
		return v
	}
	return "http://localhost:4318"
}

// NewResource creates the shared OTEL resource used by all providers.
func NewResource(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName("restful-boilerplate")),
	)
}

// SetupTraces initialises a global OTLP TracerProvider and returns a shutdown function.
func SetupTraces(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(Endpoint()))
	if err != nil {
		return nil, fmt.Errorf("otlptrace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp.Shutdown, nil
}

// parseEndpoint extracts host:port and scheme from an OTLP endpoint URL.
// Returns the host:port string and whether TLS should be disabled.
func parseEndpoint(raw string) (host string, insecure bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "localhost:4318", true
	}
	return u.Host, u.Scheme == "http"
}
