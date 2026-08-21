# Postchi - local development
# Usage: make help

.DEFAULT_GOAL := help

ROOT_DIR    := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
BACKEND_DIR := $(ROOT_DIR)/backend
FRONTEND_DIR := $(ROOT_DIR)/frontend
MIGRATIONS_DIR := $(ROOT_DIR)/migrations

# Load .env when present (export all vars for subprocesses)
ifneq (,$(wildcard $(ROOT_DIR)/.env))
include $(ROOT_DIR)/.env
export
endif

# Defaults (override via .env or environment)
DATABASE_URL       ?= postgres://postchi:postchi@localhost:5432/postchi?sslmode=disable
JWT_SECRET         ?= dev-secret-change-in-production
JWT_ISSUER         ?= postchi
ENCRYPTION_KEY     ?= postchi-dev-encryption-key-32b!!
HTTP_PORT          ?= 8080
CORS_ORIGINS       ?= http://localhost:3000
NUXT_PUBLIC_API_URL ?=
NUXT_TELEMETRY_DISABLED ?= 1
PUBLIC_APP_URL     ?= http://localhost:3000
APP_PUBLIC_URL     ?= http://localhost:3000
MIGRATIONS_PATH    ?= file://$(MIGRATIONS_DIR)
MIGRATE            ?= migrate

export DATABASE_URL JWT_SECRET JWT_ISSUER ENCRYPTION_KEY HTTP_PORT CORS_ORIGINS
export MIGRATIONS_PATH NUXT_PUBLIC_API_URL NUXT_TELEMETRY_DISABLED

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z0-9_.-]+:.*##/ {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------

.PHONY: setup
setup: ## Copy .env.example to .env if missing
	@if [ ! -f "$(ROOT_DIR)/.env" ]; then \
		cp "$(ROOT_DIR)/.env.example" "$(ROOT_DIR)/.env"; \
		echo "Created .env from .env.example"; \
	else \
		echo ".env already exists"; \
	fi

.PHONY: install
install: setup ## Install Go and npm dependencies
	cd "$(BACKEND_DIR)" && go mod download
	cd "$(FRONTEND_DIR)" && npm install

.PHONY: tools
tools: ## Install golang-migrate CLI (v4.18.3)
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3
	@echo "Installed migrate to $$(go env GOPATH)/bin/migrate"

# ---------------------------------------------------------------------------
# Database (Docker)
# ---------------------------------------------------------------------------

.PHONY: db-up
db-up: ## Start Postgres via docker compose
	docker compose -f "$(ROOT_DIR)/docker-compose.yml" up database -d

.PHONY: db-down
db-down: ## Stop Postgres container
	docker compose -f "$(ROOT_DIR)/docker-compose.yml" stop database

.PHONY: db-logs
db-logs: ## Tail Postgres logs
	docker compose -f "$(ROOT_DIR)/docker-compose.yml" logs -f database

.PHONY: db-reset
db-reset: ## Stop Postgres and remove the data volume (destructive)
	docker compose -f "$(ROOT_DIR)/docker-compose.yml" down -v

.PHONY: db-wait
db-wait: ## Wait until Postgres accepts connections
	@echo "Waiting for database..."
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do \
		docker compose -f "$(ROOT_DIR)/docker-compose.yml" exec -T database \
			pg_isready -U postchi -d postchi >/dev/null 2>&1 && exit 0; \
		sleep 1; \
	done; \
	echo "Database not ready after 20s"; exit 1

# ---------------------------------------------------------------------------
# Migrations (golang-migrate)
# ---------------------------------------------------------------------------

.PHONY: migrate-up
migrate-up: ## Apply all pending migrations
	@$(MIGRATE) -path "$(MIGRATIONS_DIR)" -database "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down: ## Roll back the last migration
	@$(MIGRATE) -path "$(MIGRATIONS_DIR)" -database "$(DATABASE_URL)" down 1

.PHONY: migrate-status
migrate-status: ## Show current migration version
	@$(MIGRATE) -path "$(MIGRATIONS_DIR)" -database "$(DATABASE_URL)" version

.PHONY: migrate-force
migrate-force: ## Force migration version (use: make migrate-force VERSION=2)
	@test -n "$(VERSION)" || (echo "VERSION is required, e.g. make migrate-force VERSION=2" && exit 1)
	@$(MIGRATE) -path "$(MIGRATIONS_DIR)" -database "$(DATABASE_URL)" force $(VERSION)

.PHONY: migrate-create
migrate-create: ## Create a new migration (use: make migrate-create NAME=add_foo)
	@test -n "$(NAME)" || (echo "NAME is required, e.g. make migrate-create NAME=add_foo" && exit 1)
	@$(MIGRATE) create -ext sql -dir "$(MIGRATIONS_DIR)" -seq $(NAME)

.PHONY: migrate
migrate: db-up db-wait migrate-up ## Start DB and run migrations

# ---------------------------------------------------------------------------
# Run locally (outside Docker)
# ---------------------------------------------------------------------------

.PHONY: backend
backend: ## Run API server locally (applies migrations on startup)
	cd "$(BACKEND_DIR)" && \
		DATABASE_URL="$(DATABASE_URL)" \
		JWT_SECRET="$(JWT_SECRET)" \
		JWT_ISSUER="$(JWT_ISSUER)" \
		ENCRYPTION_KEY="$(ENCRYPTION_KEY)" \
		HTTP_PORT="$(HTTP_PORT)" \
		CORS_ORIGINS="$(CORS_ORIGINS)" \
		MIGRATIONS_PATH="$(MIGRATIONS_PATH)" \
		go run ./cmd/api

.PHONY: frontend
frontend: ## Run Nuxt dev server
	cd "$(FRONTEND_DIR)" && \
		NUXT_PUBLIC_API_URL="$(NUXT_PUBLIC_API_URL)" \
		NUXT_TELEMETRY_DISABLED="$(NUXT_TELEMETRY_DISABLED)" \
		npm run dev

.PHONY: dev
dev: db-up db-wait migrate-up ## Start DB, migrate, then backend + frontend
	@echo "Starting backend (http://localhost:$(HTTP_PORT)) and frontend (http://localhost:3000)..."
	@$(MAKE) -j2 backend frontend

# ---------------------------------------------------------------------------
# Docker (full stack)
# ---------------------------------------------------------------------------

.PHONY: docker-up
docker-up: setup ## Build and start all services via docker compose
	docker compose -f "$(ROOT_DIR)/docker-compose.yml" up --build

.PHONY: docker-up-d
docker-up-d: setup ## Build and start all services in the background
	docker compose -f "$(ROOT_DIR)/docker-compose.yml" up --build -d

.PHONY: docker-down
docker-down: ## Stop all docker compose services
	docker compose -f "$(ROOT_DIR)/docker-compose.yml" down

.PHONY: docker-logs
docker-logs: ## Follow logs for all services
	docker compose -f "$(ROOT_DIR)/docker-compose.yml" logs -f

# ---------------------------------------------------------------------------
# Test & build
# ---------------------------------------------------------------------------

.PHONY: test
test: ## Run backend and frontend tests
	cd "$(BACKEND_DIR)" && go test ./...
	cd "$(FRONTEND_DIR)" && npm run test && npm run build

.PHONY: test-backend
test-backend: ## Run Go tests only
	cd "$(BACKEND_DIR)" && go test ./...

.PHONY: build
build: ## Build backend binaries and frontend production bundle
	@mkdir -p "$(ROOT_DIR)/bin"
	cd "$(BACKEND_DIR)" && go build -o "$(ROOT_DIR)/bin/postchi-api" ./cmd/api
	cd "$(BACKEND_DIR)" && go build -o "$(ROOT_DIR)/bin/postchi-migrate" ./cmd/migrate
	cd "$(FRONTEND_DIR)" && npm run build

.PHONY: build-backend
build-backend: ## Build API + migrate binaries to bin/
	@mkdir -p "$(ROOT_DIR)/bin"
	cd "$(BACKEND_DIR)" && go build -o "$(ROOT_DIR)/bin/postchi-api" ./cmd/api
	cd "$(BACKEND_DIR)" && go build -o "$(ROOT_DIR)/bin/postchi-migrate" ./cmd/migrate
