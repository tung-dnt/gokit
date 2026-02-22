# ============================================================
# restful-boilerplate — local development Makefile
# ============================================================

.DEFAULT_GOAL := help
SHELL         := /bin/bash

BIN_DIR  := bin
API_BIN  := $(BIN_DIR)/api
WORK_BIN := $(BIN_DIR)/worker
DB_PATH  := ./data.db
OBS_DIR  := dx/deploy

# ─── Help ────────────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
	  /^[a-zA-Z_\/-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } \
	  /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0,5) }' $(MAKEFILE_LIST)

# ─── Dev ─────────────────────────────────────────────────────────────────────

##@ Development

.PHONY: run
run: ## Run HTTP server (no hot-reload)
	go run ./cmd/http

.PHONY: dev
dev: ## Run HTTP server with hot-reload (air)
	go tool air

.PHONY: worker
worker: ## Run background worker
	go run ./cmd/worker

.PHONY: migrate
migrate: ## Apply DB migrations (default: ./data.db)
	go run ./cmd/migrate -db $(DB_PATH)

# ─── Build ───────────────────────────────────────────────────────────────────

##@ Build

.PHONY: build
build: build/api build/worker ## Build all binaries

.PHONY: build/api
build/api: ## Build HTTP API binary → bin/api
	@mkdir -p $(BIN_DIR)
	go build -o $(API_BIN) ./cmd/http

.PHONY: build/worker
build/worker: ## Build worker binary → bin/worker
	@mkdir -p $(BIN_DIR)
	go build -o $(WORK_BIN) ./cmd/worker

.PHONY: clean
clean: ## Remove build artifacts and DB file
	rm -rf $(BIN_DIR) tmp $(DB_PATH)

# ─── Quality ─────────────────────────────────────────────────────────────────

##@ Quality

.PHONY: fmt
fmt: ## Format all Go files
	gofmt -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	go tool golangci-lint run ./...

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: test/v
test/v: ## Run all tests (verbose)
	go test -v ./...

.PHONY: check
check: fmt vet lint test ## fmt + vet + lint + test

# ─── Code generation ─────────────────────────────────────────────────────────

##@ Code generation

.PHONY: sqlc
sqlc: ## Regenerate sqlc Go code from SQL queries
	go tool sqlc generate

.PHONY: swagger
swagger: ## Regenerate OpenAPI/Swagger docs
	go tool swag init -g cmd/http/main.go -o dx/docs/

# ─── Performance testing ─────────────────────────────────────────────────────

##@ Performance

PERF_URL ?= http://host.docker.internal:8080

.PHONY: perf
perf: ## Run k6 performance tests via Docker (BASE_URL=http://... to override)
	@bash dx/scripts/perf-test.sh $(PERF_URL)

# ─── Observability stack ─────────────────────────────────────────────────────

##@ Observability

.PHONY: obs/up
obs/up: ## Start Prometheus + Grafana (docker compose)
	docker compose -f $(OBS_DIR)/docker-compose.yml up -d

.PHONY: obs/down
obs/down: ## Stop observability stack
	docker compose -f $(OBS_DIR)/docker-compose.yml down

.PHONY: obs/logs
obs/logs: ## Tail logs from observability stack
	docker compose -f $(OBS_DIR)/docker-compose.yml logs -f

.PHONY: obs/ps
obs/ps: ## Show status of observability containers
	docker compose -f $(OBS_DIR)/docker-compose.yml ps
