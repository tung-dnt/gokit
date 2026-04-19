---
name: observability
description: OpenTelemetry tracing, structured logging, metrics, and the SigNoz observability stack
user_invocable: false
---

The stack is **SigNoz** (ClickHouse + OTEL Collector + UI), replacing the older Grafana/Tempo/Loki/Prometheus setup. The app exports traces, metrics, and logs via OTLP. Default transport is **gRPC on :4317**; HTTP on :4318 is used only by components that require HTTP (Genkit). Compose file lives at `infra/docker-compose.yaml`; SigNoz UI at `http://localhost:8080`.

## Components

| Component | Role | Access |
|-----------|------|--------|
| SigNoz UI + query service | Traces, metrics, logs, dashboards | `http://localhost:8080` |
| OTEL Collector | OTLP receiver → ClickHouse | `:4317` (gRPC), `:4318` (HTTP) |
| ClickHouse | Telemetry datastore | internal |
| Zookeeper | ClickHouse coordination | internal |

Manage with `make obs/up`, `make obs/down`, `make obs/logs`, `make obs/ps`.

## Tracing — `pkg/telemetry/otel.go`

- Exporter: `otlptracegrpc` to `Endpoint()` (env `OTEL_EXPORTER_OTLP_ENDPOINT`, default `localhost:4317`).
- TLS: off when `OTEL_EXPORTER_OTLP_INSECURE=true` (the default — SigNoz self-hosted is plaintext).
- Sampler: `ParentBased(TraceIDRatioBased(OTEL_TRACES_SAMPLER_ARG))`. Default `1.0` (sample all); set to `0.1` in prod for 10%.
- Resource: `service.name` from `OTEL_SERVICE_NAME` / `OTEL_RESOURCE_ATTRIBUTES`; `service.version` injected from `pkg/version.Version` via `-ldflags -X`.
- Propagators: `tracecontext` + `baggage`.

TracerProvider lives in `app.App.Tracer` (`otel.GetTracerProvider()`). Each domain grabs a named tracer inside `NewModule` and stores it on the service struct (no globals, no `gochecknoglobals` noise):

```go
func NewModule(a *app.App) *Module {
    svc := newUserService(a.Queries, a.Tracer.Tracer("user"))
    return &Module{httpAdapter: newHTTPAdapter(svc, a.Validator)}
}
```

Tests inject `noop.NewTracerProvider()` to disable export.

## HTTP server instrumentation — `pkg/http/options.go`

Wrap the router at construction with `router.WithInstrumentation("http.server")`:

```go
r := router.NewRouter(router.WithInstrumentation("http.server"))
```

This uses `otelhttp.NewHandler` with a span-name formatter emitting `"{METHOD} {matched-pattern}"` (e.g. `GET /v1/api/users/{id}`) and falling back to the raw path. Route-template names keep span cardinality bounded — essential for SigNoz trace search and histograms.

## PostgreSQL instrumentation — `pkg/postgres/connection.go`

`postgres.OpenDB` wires `otelpgx` into pgxpool: every query becomes a span, and `otelpgx.RecordStats(pool)` emits pool metrics (`acquired`, `idle`, `wait_duration`). Nothing to do per-domain.

## Metrics — `pkg/telemetry/meter.go`

- Exporter: `otlpmetricgrpc` to the same `Endpoint()`.
- **Temporality: DELTA** — forced via a custom `TemporalitySelector`. SigNoz's ClickHouse pipeline expects delta-temporality counters and histograms; the OTel Go SDK defaults to cumulative, which breaks rate math.
- Reader: `PeriodicReader` at 10s.
- Go runtime metrics started via `runtime.Start(runtime.WithMeterProvider(mp))` — GC, goroutine count, heap, etc.

There is no `pkg/metrics/` package and no `/metrics` HTTP endpoint — everything ships via OTLP.

## Logs — `pkg/logger/logger.go`

`logger.Setup(logFormat, logProvider)` installs a `fanoutHandler` on `slog.Default`:

1. **stdout** — pretty (colorized) when `LOG_FORMAT=pretty`, JSON otherwise. Dev uses `pretty`; prod uses `json`.
2. **OTLP** — `otelslog.NewHandler("gokit", ...)` bridges slog records to the OTLP log exporter from `pkg/telemetry/logs.go`.

For trace-correlated logs inside a handler:

```go
logger.FromContext(ctx).Info("created user", "id", u.ID)
// emits trace_id + span_id attributes, linked to the active span in SigNoz
```

## Error helpers — `pkg/telemetry/span.go`

Two helpers keep SLI math clean. Both wrap with `fmt.Errorf("%s: %w", op, err)` and attach a low-cardinality `error.type` attribute.

- **`SpanExpectedErr(span, err, op, kind)`** — records the error but **leaves span status Unset**. Use for handled 4xx outcomes (404, validation, conflict). Per OTel HTTP semconv, client errors must not pollute error-rate SLIs.
- **`SpanUnexpectedErr(span, err, op)`** — records, classifies via `ClassifyErr`, and sets span status `Error`. Use for DB failures, upstream 5xx, corrupt state.

Error kinds (stable — referenced in SigNoz dashboards):
`ErrKindNotFound`, `ErrKindValidation`, `ErrKindInvalidJSON`, `ErrKindConflict`, `ErrKindUnauthorized`, `ErrKindTimeout`, `ErrKindCanceled`, `ErrKindPanic`, `ErrKindUnexpected`.

Real example from `internal/user/service.user.go`:

```go
row, err := s.db.GetUser(ctx, id)
if err != nil {
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, telemetry.SpanExpectedErr(span, ErrNotFound, "user.getUserByID", telemetry.ErrKindNotFound)
    }
    return nil, telemetry.SpanUnexpectedErr(span, err, "user.getUserByID")
}
```

Never pass `err.Error()` as the `error.type` value — that's high-cardinality and wrecks metric dimensions.

## LLM spans — `pkg/telemetry/llm.go`

`StartLLMSpan` + `RecordLLMAttrs` annotate LLM calls with provider, model, token usage. Emits both Traceloop and OTel GenAI semconv attributes for dashboard compatibility. See the `llm-observability` skill for usage details.

## Genkit OTEL plugin

`cmd/http/main.go` wires `genkit-opentelemetry-go` alongside the app's own telemetry:

```go
otelPlugin := opentelemetry.New(opentelemetry.Config{
    ServiceName:    "gokit",
    ForceExport:    true,
    OTLPEndpoint:   telemetry.EndpointHTTP(),
    OTLPUseHTTP:    true,
    MetricInterval: 15 * time.Second,
})
```

The plugin uses `EndpointHTTP()` (env `OTEL_EXPORTER_OTLP_HTTP_ENDPOINT`, default `http://localhost:4318`) because Genkit's plugin only supports HTTP transport — the app proper still uses gRPC. This gets Genkit-internal spans (flow executions, LLM calls, embeddings, retrieval) into the same SigNoz trace alongside the HTTP/DB spans.

## Environment variables

From `.env.example`:

- `OTEL_SERVICE_NAME` — service identity.
- `OTEL_EXPORTER_OTLP_ENDPOINT` — gRPC, default `localhost:4317`.
- `OTEL_EXPORTER_OTLP_HTTP_ENDPOINT` — HTTP (Genkit only), default `http://localhost:4318`.
- `OTEL_EXPORTER_OTLP_INSECURE` — default `true` for dev plaintext; set `false` for TLS.
- `OTEL_TRACES_SAMPLER_ARG` — head-based sample ratio in `[0, 1]`.
- `OTEL_RESOURCE_ATTRIBUTES` — e.g. `deployment.environment=dev`.
- `LOG_FORMAT` — `pretty` or `json`.

## Dev stack

```bash
make obs/up      # start SigNoz + Postgres
make obs/ps
make obs/logs
make obs/down
```

Configuration for the Collector and ClickHouse lives under `infra/signoz/`.
