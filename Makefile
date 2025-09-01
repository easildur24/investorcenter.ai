# InvestorCenter.ai Makefile

.PHONY: help setup-local setup-prod install-deps build test dev clean

# Configuration
POSTGRES_VERSION = 15
DB_NAME = investorcenter_db
DB_USER = investorcenter
LOCAL_DB_PASSWORD = investorcenter123
VENV_PATH = path/to/venv
BACKEND_BINARY = backend/investorcenter-api

# Environment detection
ENVIRONMENT ?= local

help:
	@echo "InvestorCenter.ai Development & Deployment"
	@echo "=========================================="
	@echo ""
	@echo "Setup Commands:"
	@echo "  make setup-local      - Complete local development setup"
	@echo "  make setup-prod       - Deploy production infrastructure"
	@echo "  make install-deps     - Install all dependencies"
	@echo ""
	@echo "Database Commands:"
	@echo "  make db-setup-local   - Setup local PostgreSQL database"
	@echo "  make db-setup-prod    - Setup production PostgreSQL in K8s"
	@echo "  make db-import        - Import ticker data to current environment"
	@echo "  make db-update        - Update ticker data (incremental)"
	@echo "  make db-migrate       - Run database migrations"
	@echo ""
	@echo "Development Commands:"
	@echo "  make dev             - Start full development environment"
	@echo "  make dev-backend     - Start Go API server only"
	@echo "  make dev-frontend    - Start Next.js frontend only"
	@echo "  make build           - Build all components"
	@echo "  make test            - Run all tests"
	@echo ""
	@echo "Environment Commands:"
	@echo "  make local <target>  - Run target in local environment"
	@echo "  make prod <target>   - Run target in production environment"
	@echo "  make verify          - Verify setup and database status"
	@echo "  make status          - Show environment status"

setup-local: install-deps db-setup-local db-migrate
	@echo "Local development environment ready"
	@echo "Start development with: make dev"

install-deps:
	@echo "Installing dependencies..."
	@command -v brew >/dev/null 2>&1 || { echo "Homebrew required"; exit 1; }
	npm install
	cd backend && go mod tidy
	python3 -m venv $(VENV_PATH) || true
	. $(VENV_PATH)/bin/activate && pip install -r requirements.txt psycopg2-binary python-dotenv

db-setup-local:
	@echo "Setting up local PostgreSQL..."
	@if ! command -v psql >/dev/null 2>&1; then \
		brew install postgresql@$(POSTGRES_VERSION); \
	fi
	@export PATH="/opt/homebrew/opt/postgresql@$(POSTGRES_VERSION)/bin:$$PATH" && \
	brew services start postgresql@$(POSTGRES_VERSION) || true && \
	sleep 2 && \
	createdb $(DB_NAME) || true && \
	psql $(DB_NAME) -c "CREATE USER $(DB_USER) WITH PASSWORD '$(LOCAL_DB_PASSWORD)';" || true && \
	psql $(DB_NAME) -c "GRANT ALL PRIVILEGES ON DATABASE $(DB_NAME) TO $(DB_USER);" && \
	psql $(DB_NAME) -c "GRANT ALL PRIVILEGES ON SCHEMA public TO $(DB_USER);"

db-migrate:
	@echo "Running database migrations..."
	@export PATH="/opt/homebrew/opt/postgresql@$(POSTGRES_VERSION)/bin:$$PATH" && \
	psql $(DB_NAME) -f backend/migrations/001_create_stock_tables.sql && \
	psql $(DB_NAME) -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $(DB_USER);" && \
	psql $(DB_NAME) -c "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $(DB_USER);"

build: build-backend build-frontend

build-backend:
	@echo "Building Go backend..."
	cd backend && go build -o investorcenter-api .

build-frontend:
	@echo "Building Next.js frontend..."
	npm run build

dev:
	@echo "Starting development environment..."
	@$(MAKE) dev-backend &
	@$(MAKE) dev-frontend

dev-backend:
	@echo "Starting Go API server..."
	@export PATH="/opt/homebrew/opt/postgresql@$(POSTGRES_VERSION)/bin:$$PATH" && \
	cd backend && \
	DB_HOST=localhost DB_PORT=5432 DB_USER=$(DB_USER) DB_PASSWORD=$(LOCAL_DB_PASSWORD) \
	DB_NAME=$(DB_NAME) DB_SSLMODE=disable ./investorcenter-api

dev-frontend:
	@echo "Starting Next.js development server..."
	npm run dev

test:
	@echo "Running tests..."
	. $(VENV_PATH)/bin/activate && python scripts/test_ticker_db_importer.py
	cd backend && go test ./...

verify:
	@./scripts/verify-setup.sh

status:
	@python scripts/env-manager.py status

clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BACKEND_BINARY)
	rm -rf .next/
	cd backend && go clean# Database-specific Makefile targets

db-import:
	@echo "Importing ticker data..."
	@. $(VENV_PATH)/bin/activate && python scripts/env-manager.py set $(ENVIRONMENT) >/dev/null && \
	python scripts/ticker_import_to_db.py

db-update:
	@echo "Updating ticker data (incremental)..."
	@. $(VENV_PATH)/bin/activate && python scripts/env-manager.py set $(ENVIRONMENT) >/dev/null && \
	python scripts/update_tickers_cron.py

db-import-prod:
	@$(MAKE) ENVIRONMENT=prod db-import

db-import-local:
	@$(MAKE) ENVIRONMENT=local db-import

db-setup-prod:
	kubectl apply -f k8s/namespace.yaml
	kubectl create secret generic postgres-secret \
		--from-literal=username=$(DB_USER) \
		--from-literal=password=prod_investorcenter_456 \
		-n investorcenter || true
	kubectl apply -f k8s/postgres-deployment.yaml
	kubectl wait --for=condition=available --timeout=300s deployment/postgres -n investorcenter

k8s-deploy:
	kubectl apply -f k8s/
	kubectl wait --for=condition=available --timeout=300s deployment/postgres -n investorcenter

k8s-cleanup:
	kubectl delete namespace investorcenter --ignore-not-found=true
