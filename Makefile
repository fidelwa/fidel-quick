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
full-up: ## Start all services (infra + backend + admin) containerized
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
start-all: ## One command: infra + migrate + backend + admin frontend (native)
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

.PHONY: start-docker
start-docker: ## One command: everything containerized (build + run)
	@echo "Building and starting all containers..."
	docker compose --profile full up -d --build
	@echo "Waiting for Postgres..."
	@until docker compose exec postgres pg_isready -U loyalty -q 2>/dev/null; do sleep 1; done
	@echo "Running migrations..."
	@migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up
	@echo ""
	@echo "All services running:"
	@echo "  Admin:  http://localhost:3000"
	@echo "  API:    http://localhost:8080"
	@echo "  MinIO:  http://localhost:9001"

## Production tear-down (DESTRUCTIVE)

# Workdir donde vive el state de terraform (creado por el flujo de despliegue,
# fuera del repo para que no se commitee). Sobrescribir con TF_WORKDIR=/otra/ruta.
TF_WORKDIR ?= $(HOME)/.fidel-deploy/tfwork

.PHONY: destroy
destroy: ## ⚠️  Destruir TODA la infra GCP (DB y datos incluidos). Pide confirmación tipeando BORRAR-TODO.
	@echo ""
	@echo "════════════════════════════════════════════════════════════════════"
	@echo "  ⚠️   make destroy  —  TEAR-DOWN COMPLETO DE INFRA GCP"
	@echo "════════════════════════════════════════════════════════════════════"
	@echo ""
	@echo "  Esto va a BORRAR PERMANENTEMENTE:"
	@echo ""
	@echo "    • Cloud SQL Postgres (fidel-db) Y TODOS LOS DATOS"
	@echo "        - clientes, colaboradores, transacciones, balances,"
	@echo "          tarjetas pushcard, sesiones admin, customer_sisfi, etc."
	@echo "    • Cloud Storage bucket (fidel-mvp-invoice) y TODAS las fotos"
	@echo "        de facturas almacenadas."
	@echo "    • Cloud Run service (fidel-quick) — la URL pública dejará"
	@echo "        de responder. Webhook de WhatsApp empezará a fallar."
	@echo "    • Secret Manager (9 secrets: WhatsApp, JWT, Gemini, etc.)."
	@echo "    • Artifact Registry (imágenes Docker pushed)."
	@echo "    • Service Account fidel-quick-sa y sus IAM bindings."
	@echo ""
	@echo "  Esta operación NO ES REVERSIBLE."
	@echo "  Los backups automáticos de Cloud SQL también se pierden con la instancia."
	@echo ""
	@echo "  Si solo quieres parar costos temporalmente (sin perder datos),"
	@echo "  considera en su lugar:"
	@echo "    gcloud run services update fidel-quick --min-instances=0 --max-instances=0"
	@echo "    gcloud sql instances patch fidel-db --activation-policy=NEVER"
	@echo ""
	@echo "════════════════════════════════════════════════════════════════════"
	@echo ""
	@printf "  Para confirmar, tipea EXACTAMENTE  \033[1;31mBORRAR-TODO\033[0m  y enter: "
	@read CONFIRM; \
	if [ "$$CONFIRM" != "BORRAR-TODO" ]; then \
		echo ""; echo "  ❌ Confirmación no coincide. Abortado."; \
		exit 1; \
	fi
	@echo ""
	@echo "  → Desactivando deletion_protection del Cloud SQL..."
	@-gcloud sql instances patch fidel-db --no-deletion-protection --quiet 2>/dev/null || \
		echo "    (instancia no existe o ya sin protección — continúo)"
	@echo "  → Corriendo terraform destroy en $(TF_WORKDIR)..."
	@if [ ! -d "$(TF_WORKDIR)" ]; then \
		echo "    ⚠️  $(TF_WORKDIR) no existe — saltando terraform destroy."; \
		echo "    Si la infra fue creada con gcloud manual, bórrala desde la consola."; \
	else \
		cd $(TF_WORKDIR) && terraform destroy -auto-approve; \
	fi
	@echo "  → Borrando secrets sobrevivientes (best-effort)..."
	@for s in WHATSAPP_API_TOKEN WHATSAPP_VERIFY_TOKEN WHATSAPP_APP_SECRET \
	          JWT_SECRET BEARER_TOKEN GEMINI_API_KEY GOOGLE_CLIENT_ID \
	          DB_PASSWORD REDIS_URL; do \
		gcloud secrets delete $$s --quiet 2>/dev/null || true; \
	done
	@echo ""
	@echo "  ✓ Tear-down completado."
	@echo "    Verifica en https://console.cloud.google.com/billing que el spend"
	@echo "    se detiene en las próximas 24h. Cloud SQL deja de cobrar inmediato;"
	@echo "    Cloud Run / GCS prorratean."
	@echo ""

## Help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
