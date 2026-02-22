# restful-boilerplate

A Go RESTful API boilerplate built on **Echo v5 + SQLite + sqlc** with full observability (OpenTelemetry, Tempo, Loki, Grafana).

## Quick Start

```bash
# Install dependencies
go mod download

# Run DB migrations
make migrate

# Start hot-reload dev server
make dev
```

The API is available at `http://localhost:8080/api`. Swagger UI at `http://localhost:8080/swagger/`.

## Stack

| Layer | Technology |
|-------|------------|
| HTTP | Echo v5 |
| Database | SQLite (modernc — pure Go, no CGO) |
| SQL codegen | sqlc |
| Validation | go-playground/validator v10 |
| API docs | swaggo/swag (OpenAPI) |
| Tracing | OpenTelemetry (OTLP HTTP) |
| Metrics | Prometheus |
| Logs | slog + Loki |

## Development Commands

```bash
make dev        # hot-reload server (air)
make run        # run without hot-reload
make migrate    # apply DB migrations
make check      # fmt + vet + lint + test (full quality gate)
make sqlc       # regenerate sqlc Go code
make swagger    # regenerate OpenAPI docs
make build      # build binaries
make obs/up     # start observability stack (Grafana, Tempo, Loki, Prometheus)
make obs/down   # stop observability stack
```

## Project Documentation

Full architecture, patterns, conventions, and developer guide are in **[CLAUDE.md](./CLAUDE.md)**, including:

- Directory structure and module layout
- Modular monolith architecture
- Key coding patterns (handler pipeline, error handling, validation, DI)
- HTTP API reference
- How to add a new domain (`/new-domain` skill)
- Observability stack setup
- SQLite configuration notes
