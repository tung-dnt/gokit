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

Docker Compose lives in `dx/deploy/`. Manage with `make obs/up` and `make obs/down`.
Grafana datasource UIDs are fixed: `tempo` and `loki` (enables cross-correlation in provisioning).

## Tracing — `pkg/telemetry/`

- `pkg/telemetry/otel.go` — OTLP HTTP TracerProvider
- `pkg/telemetry/setup.go` — `SetupAll(ctx, logPath)` initializes tracer + log file
- Reads `OTEL_EXPORTER_OTLP_ENDPOINT` env var (default: `http://localhost:4318` in dev)
- **Tracer as struct field** — store `tracer trace.Tracer` on service structs to avoid `gochecknoglobals` linter

```go
type xxxService struct {
    q      *sqlitedb.Queries
    tracer trace.Tracer
}
```

## Echo v5 OTEL Middleware — `pkg/otelecho/`

- `pkg/otelecho/middleware.go` — custom middleware (official `otelecho` only supports Echo v4)
- Injects `trace_id` + `span_id` into request context
- **Echo v5 quirk:** use `echo.UnwrapResponse(c.Response())` to get `*echo.Response` with `.Status` field — `c.Response()` returns plain `http.ResponseWriter`

## Structured Logging — `pkg/logger/`

- `pkg/logger/logger.go` — slog MultiWriter (stdout + `./logs/app.log`)
- `logger.FromContext(ctx)` returns trace-correlated `slog.Logger` (includes `trace_id` + `span_id`)
- JSON output format for machine parsing

## Metrics — `pkg/metrics/`

- `pkg/metrics/metrics.go` — Prometheus registry
- Exposes `/metrics` endpoint for Prometheus scraping
