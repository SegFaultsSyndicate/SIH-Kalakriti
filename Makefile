# Makefile
SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

# .env is optional; every variable below has a local-dev default.
-include .env
export
export PATH := $(HOME)/go/bin:$(PATH)


COMPOSE ?= docker compose
BIN_DIR  := bin

GO_MODULES  := pkg services/core-svc services/search-svc services/collab-svc services/bff services/insight-svc services/channel-svc
GO_SERVICES := core-svc search-svc collab-svc bff insight-svc channel-svc

POSTGRES_USER     ?= kalakriti
POSTGRES_PASSWORD ?= kalakriti
POSTGRES_DB       ?= kalakriti
POSTGRES_PORT     ?= 5432
POSTGRES_DSN      ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

# Must match whatever docker-compose.yml's bff/core-svc containers were
# actually started with (their own JWT_SECRET default, or your .env override).
JWT_SECRET        ?= dev-secret-change-in-prod-32bytes-minimum
BFF_BASE_URL       ?= http://localhost:8000

# Prefer a locally installed binary; otherwise pin the version through `go run`.
BUF   ?= $(shell command -v buf   2>/dev/null || echo "go run github.com/bufbuild/buf/cmd/buf@v1.34.0")
GOOSE ?= $(shell command -v goose 2>/dev/null || echo "go run github.com/pressly/goose/v3/cmd/goose@v3.21.1")
SQLC  ?= $(shell command -v sqlc  2>/dev/null || echo "go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0")
LINT  ?= $(shell command -v golangci-lint 2>/dev/null || echo "go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1")

.PHONY: help up down logs ps reset proto proto-go proto-py proto-lint migrate-up migrate-down seed seed-demo seed-data sqlc test test-ml lint tidy build clean check psql services services-stop demo-up demo-reset tags docker-build

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-14s %s\n", $$1, $$2}'

# --- infrastructure ----------------------------------------------------------

up: proto sqlc ## Generate code, then start infrastructure and wait for it to be healthy
	$(COMPOSE) up -d --wait postgres redis kafka minio jaeger prometheus
	$(COMPOSE) up minio-init

down: ## Stop infrastructure, keep volumes
	$(COMPOSE) down --remove-orphans

services: ## Start core-svc and bff in background for local web development
	@bash scripts/run-services.sh

services-stop: ## Stop background core-svc and bff
	@pkill -f "services/core-svc/core-svc" 2>/dev/null || true
	@pkill -f "services/bff/bff" 2>/dev/null || true
	@echo "Services stopped."


logs: ## Tail infrastructure logs
	$(COMPOSE) logs -f --tail=100

ps: ## Show container status
	$(COMPOSE) ps

reset: ## Stop infrastructure and destroy all volumes
	$(COMPOSE) down --volumes --remove-orphans

check: ## Probe every dependency from the host
	./scripts/check-deps.sh

psql: ## Open a psql shell on the dev database
	$(COMPOSE) exec -e PGPASSWORD=$(POSTGRES_PASSWORD) postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

# --- codegen -----------------------------------------------------------------

proto: proto-go proto-py ## Generate Go and Python code from proto/

proto-go: ## Generate Go code from proto/ (buf.gen.yaml only wires Go plugins)
	$(BUF) generate proto --template buf.gen.yaml

proto-py: ## Generate ml-svc's Python protobuf/grpc stubs into services/ml-svc/pb/
	@if command -v uv >/dev/null 2>&1; then \
		cd services/ml-svc && uv run --extra dev ./scripts/gen_proto.sh; \
	else \
		echo "uv not found; skipping ml-svc proto generation"; \
	fi

proto-lint: ## Lint and breaking-change-check the protos
	$(BUF) lint proto
	$(BUF) format -d --exit-code

sqlc: ## Generate type-safe query code from SQL
	$(SQLC) generate

# --- database ----------------------------------------------------------------

migrate-up: ## Apply all pending migrations
	$(GOOSE) -dir migrations postgres "$(POSTGRES_DSN)" up

migrate-down: ## Roll back the most recent migration
	$(GOOSE) -dir migrations postgres "$(POSTGRES_DSN)" down

seed: ## Load the craft ontology from scripts/data/*.csv (idempotent)
	cd services/core-svc && go run ./cmd/seed-ontology \
		-dsn "$(POSTGRES_DSN)" \
		-crafts ../../scripts/data/crafts.csv \
		-aliases ../../scripts/data/aliases.csv

# --- Go ----------------------------------------------------------------------

build: ## Build every Go service into ./bin
	@mkdir -p $(BIN_DIR)
	@set -e; for s in $(GO_SERVICES); do \
		echo ">> building $$s"; \
		(cd services/$$s && go build -trimpath -o ../../$(BIN_DIR)/$$s ./cmd/$$s); \
	done

test: ## Run all Go unit tests (no Docker needed)
	@set -e; for m in $(GO_MODULES); do \
		echo ">> testing $$m"; \
		(cd $$m && go test ./... -race -count=1); \
	done

test-ml: ## Run the ml-svc test suite (mock mode, no weights needed)
	cd services/ml-svc && uv run --extra dev pytest -q

test-integration: ## Run tests that start real containers (needs a Docker daemon)
	@set -e; for m in $(GO_MODULES); do \
		echo ">> integration testing $$m"; \
		(cd $$m && go test ./... -tags=integration -race -count=1); \
	done

lint: ## Run golangci-lint across every module
	@set -e; for m in $(GO_MODULES); do \
		echo ">> linting $$m"; \
		(cd $$m && $(LINT) run ./...); \
	done

tidy: ## go mod tidy every module and sync the workspace
	@set -e; for m in $(GO_MODULES); do \
		echo ">> tidying $$m"; \
		(cd $$m && go mod tidy); \
	done
	go work sync

clean: ## Remove build output
	rm -rf $(BIN_DIR)

# --- demo --------------------------------------------------------------------

demo-up: proto sqlc ## Start full demo: infra + all services + migrate + seed
	@echo "Starting demo environment..."
	$(COMPOSE) up -d
	@echo "Waiting for services..."
	sleep 15
	@echo "Running migrations..."
	$(MAKE) migrate-up POSTGRES_DSN="$(POSTGRES_DSN)"
	@echo "Seeding craft ontology..."
	$(MAKE) seed POSTGRES_DSN="$(POSTGRES_DSN)"
	@echo "Seeding demo artisans/listings/orders through the real API..."
	$(MAKE) seed-demo
	@echo ""
	@echo "✓ Demo environment ready!"
	@echo ""
	@echo "Services:"
	@echo "  Web (NGINX):    http://localhost/  (buyer, /artisan/, /admin/)"
	@echo "  BFF API:        http://localhost:8000  (also proxied at http://localhost/api/v1/*)"
	@echo "  Jaeger UI:      http://localhost:16686"
	@echo "  Prometheus:     http://localhost:9090"
	@echo "  MinIO Console:  http://localhost:9001 (minioadmin/minioadmin)"

demo-reset: ## Reset demo: down + volumes + demo-up
	@echo "Resetting demo environment..."
	$(COMPOSE) down -v
	$(MAKE) demo-up

# scripts/seed/main.go targets a users/orders/order_allocations schema that
# predates the real migrations (no `users` table exists — see CLAUDE.md) and
# will fail against the current database; left in place but unused (`make
# seed` runs services/core-svc/cmd/seed-ontology instead).
seed-demo: ## Seed demo artisans/listings/orders through the real BFF API (needs `make up`/`make demo-up` running, and `make seed` already run)
	go run ./cmd/seed-demo

seed-data: seed seed-demo ## Alias for `make seed seed-demo`

tags: ## Generate QR code sheet PDF
	go run scripts/generate-qr-sheet/main.go

docker-build: proto ## Build all Docker images
	@echo "Building Docker images..."
	docker build -f Dockerfile.core-svc -t kalakriti/core-svc:latest .
	docker build -f Dockerfile.search-svc -t kalakriti/search-svc:latest .
	docker build -f Dockerfile.collab-svc -t kalakriti/collab-svc:latest .
	docker build -f Dockerfile.bff -t kalakriti/bff:latest .
	docker build -f Dockerfile.ml-svc -t kalakriti/ml-svc:latest .
	docker build -f Dockerfile.insight-svc -t kalakriti/insight-svc:latest .
	docker build -f Dockerfile.channel-svc -t kalakriti/channel-svc:latest .
	@echo "✓ All images built"
