DATABASE_URL := postgres://loyalty:loyalty@localhost:5433/loyalty?sslmode=disable
MIGRATIONS_PATH := migrations

## Infrastructure

.PHONY: infra-up
infra-up: ## Start Postgres + Redis + MinIO
	docker compose --profile infra up -d

.PHONY: infra-down
infra-down: ## Stop infrastructure
	docker compose --profile infra down

.PHONY: full-up
full-up: ## Start all services including Go app
	docker compose --profile full up -d --build

.PHONY: full-down
full-down: ## Stop all services
	docker compose --profile full down

## Migrations (requires: brew install golang-migrate)

.PHONY: migrate-up
migrate-up: ## Run all migrations
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down: ## Rollback last migration
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

.PHONY: migrate-reset
migrate-reset: ## Rollback all migrations
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down -all

.PHONY: migrate-create
migrate-create: ## Create new migration (usage: make migrate-create NAME=create_customers)
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)

## Quick Start

.PHONY: start
start: ## One command: infra + wait + migrate + run app
	@echo "Starting infrastructure..."
	@docker compose --profile infra up -d
	@echo "Waiting for Postgres..."
	@until docker compose exec postgres pg_isready -U loyalty -q 2>/dev/null; do sleep 1; done
	@echo "Running migrations..."
	@migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up
	@echo "Starting app..."
	@go run main.go

.PHONY: stop
stop: infra-down ## Stop everything

## Development

.PHONY: dev
dev: ## Run the Go app locally (infra must be running)
	go run main.go

.PHONY: dev-admin
dev-admin: ## Run the admin dashboard in dev mode
	cd admin && npm run dev

.PHONY: dev-all
dev-all: ## Run Go backend + admin frontend concurrently (infra must be running)
	@trap 'kill 0' EXIT; \
	go run main.go & \
	cd admin && npm run dev & \
	wait

.PHONY: build
build: ## Build the Go binary
	CGO_ENABLED=0 go build -o fidel-quick .

.PHONY: build-admin
build-admin: ## Build the admin dashboard for production
	cd admin && npm run build

.PHONY: build-all
build-all: build build-admin ## Build Go binary + admin dashboard

## Admin Setup

.PHONY: admin-install
admin-install: ## Install admin dashboard dependencies
	cd admin && npm install

.PHONY: create-admin
create-admin: ## Create admin account (usage: make create-admin EMAIL=tu@email.com PASSWORD=secret CUSTOMER_ID=uuid)
	@if [ -z "$(EMAIL)" ] || [ -z "$(PASSWORD)" ] || [ -z "$(CUSTOMER_ID)" ]; then \
		echo "Uso: make create-admin EMAIL=tu@email.com PASSWORD=secret CUSTOMER_ID=uuid"; \
		exit 1; \
	fi
	@curl -s -X POST http://localhost:8080/api/v1/auth/register \
		-H "Content-Type: application/json" \
		-d '{"email":"$(EMAIL)","password":"$(PASSWORD)","customer_id":"$(CUSTOMER_ID)"}' | python3 -m json.tool

## Full Stack

.PHONY: start-all
start-all: ## One command: infra + migrate + backend + admin frontend
	@echo "Starting infrastructure..."
	@docker compose --profile infra up -d
	@echo "Waiting for Postgres..."
	@until docker compose exec postgres pg_isready -U loyalty -q 2>/dev/null; do sleep 1; done
	@echo "Running migrations..."
	@migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up
	@echo "Starting backend + admin..."
	@trap 'kill 0' EXIT; \
	go run main.go & \
	cd admin && npm run dev & \
	wait

## Help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
