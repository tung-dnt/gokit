# ============================================================
# gokit — local development Makefile
# ============================================================

.DEFAULT_GOAL := help
SHELL         := /bin/bash

BIN_DIR  := bin
API_BIN  := $(BIN_DIR)/api
FE_BIN   := $(BIN_DIR)/frontend
DB_PATH  := ./data.db
OBS_DIR  := infra

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
	LOG_FORMAT=pretty genkit start -- go tool air

.PHONY: migrate
migrate: ## Apply PostgreSQL DB migrations
	go run ./cmd/postgres-migration

# ─── Build ───────────────────────────────────────────────────────────────────

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X 'gokit/pkg/version.Version=$(VERSION)'

##@ Build
.PHONY: build/api
build/api: ## Build HTTP API binary → bin/api
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(API_BIN) ./cmd/http

.PHONY: build/frontend
	@mkdir -p $(BIN_DIR)
	go build -o $(FE_BIN) ./cmd/frontend

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
	go tool swag init -g cmd/http/main.go -o docs/

# ─── Performance testing ─────────────────────────────────────────────────────

##@ Performance

PERF_URL ?= http://host.docker.internal:4040

.PHONY: perf
perf: ## Run k6 performance tests via Docker (BASE_URL=http://... to override)
	@bash scripts/perf-test.sh $(PERF_URL)

# ─── Docker ──────────────────────────────────────────────────────────────────

##@ Docker

IMAGE ?= gokit
TAG   ?= $(VERSION)

.PHONY: docker/build
docker/build: ## Build production Docker image → $(IMAGE):$(TAG)
	docker build -f $(OBS_DIR)/Dockerfile \
		--build-arg VERSION=$(VERSION) \
		-t $(IMAGE):$(TAG) \
		-t $(IMAGE):latest \
		.

.PHONY: docker/run
docker/run: ## Run the built image on :4040 (DATABASE_URL=... to override)
	docker run --rm -p 4040:4040 \
		-e DATABASE_URL="$${DATABASE_URL:-postgres://postgres:postgres@host.docker.internal:5432/postgres?sslmode=disable}" \
		-e LOG_FORMAT="$${LOG_FORMAT:-json}" \
		--name $(IMAGE) \
		$(IMAGE):$(TAG)

.PHONY: docker/push
docker/push: ## Push $(IMAGE):$(TAG) (and :latest) to the configured registry
	docker push $(IMAGE):$(TAG)
	docker push $(IMAGE):latest

# ─── Observability stack ─────────────────────────────────────────────────────

##@ Observability

.PHONY: obs/up
obs/up: ## Start SigNoz observability stack (docker compose)
	docker compose -f $(OBS_DIR)/docker-compose.yaml up -d

.PHONY: obs/down
obs/down: ## Stop observability stack
	docker compose -f $(OBS_DIR)/docker-compose.yaml down

.PHONY: obs/logs
obs/logs: ## Tail logs from observability stack
	docker compose -f $(OBS_DIR)/docker-compose.yaml logs -f

.PHONY: obs/ps
obs/ps: ## Show status of observability containers
	docker compose -f $(OBS_DIR)/docker-compose.yaml ps
