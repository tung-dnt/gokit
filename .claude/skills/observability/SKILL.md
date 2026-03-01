---
name: observability
description: OpenTelemetry tracing, structured logging, metrics, and the Grafana observability stack
user_invocable: false
---

Reference for the observability stack: tracing, logging, metrics, and the Docker Compose infrastructure.

## Component Table

| Component | Role | Access |
|-----------|------|--------|
| Tempo | Trace backend (OTLP HTTP) | Port 4318 |
| Loki | Log aggregation | Internal only |
| Alloy | Log shipper — tails `./logs/app.log` | Internal only |
| Prometheus | Metrics scraper | Internal only |
| Grafana | Visualization | `http://localhost:3000` |

Docker Compose lives in `deploy/`. Manage with `make obs/up` and `make obs/down`.
Grafana datasource UIDs are fixed: `tempo` and `loki` (enables cross-correlation in provisioning).

## Tracing — `infra/telemetry/`

- `infra/telemetry/otel.go` — OTLP HTTP TracerProvider
- `infra/telemetry/setup.go` — `SetupAll(ctx, logPath)` initializes tracer + log file
- Reads `OTEL_EXPORTER_OTLP_ENDPOINT` env var (default: `http://localhost:4318` in dev)
- **Tracer as struct field** — store `tracer trace.Tracer` on service structs to avoid `gochecknoglobals` linter

```go
type xxxService struct {
    q      *sqlitedb.Queries
    tracer trace.Tracer
}
```

## net/http OTEL Middleware — `infra/otelhttp/`

- `infra/otelhttp/middleware.go` — custom middleware for Go stdlib net/http
- `Middleware(serviceName string) func(http.Handler) http.Handler`
- Uses `statusWriter` wrapper to capture response status code for span attributes
- Injects `trace_id` + `span_id` into request context

## Structured Logging — `infra/logger/`

- `infra/logger/logger.go` — slog MultiWriter (stdout + `./logs/app.log`)
- `logger.FromContext(ctx)` returns trace-correlated `slog.Logger` (includes `trace_id` + `span_id`)
- JSON output format for machine parsing

## Metrics — `infra/metrics/`

- `infra/metrics/metrics.go` — Prometheus registry with `promhttp.HandlerFor()`
- Exposes via `r.Route("GET /metrics", metrics.Handler())` in main
