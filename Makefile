# InvestorCenter.ai Makefile

.PHONY: help setup install build dev test check clean dev-task-service build-task-service test-task-service coverage coverage-frontend coverage-backend coverage-ic-score coverage-task-service coverage-data-ingestion

# Configuration
VENV_PATH = path/to/venv
DB_NAME = investorcenter_db
DB_USER = investorcenter
DB_PASSWORD ?= $(error DB_PASSWORD is not set. Export it or pass via make DB_PASSWORD=...)
PROD_DB_PASSWORD ?= $(error PROD_DB_PASSWORD is not set. Export it or pass via make PROD_DB_PASSWORD=...)

help:
	@echo "InvestorCenter.ai Development Commands"
	@echo "====================================="
	@echo ""
	@echo "Setup:"
	@echo "  make setup           - Complete development environment setup"
	@echo "  make install         - Install all dependencies"
	@echo ""
	@echo "Development:"
	@echo "  make dev             - Start backend and frontend"
	@echo "  make build           - Build all components"
	@echo "  make test            - Run tests and linting"
	@echo "  make check           - Complete validation before push"
	@echo ""
	@echo "Database:"
	@echo "  make db-setup        - Setup database and import data"
	@echo "  make db-import       - Import/update ticker data"
	@echo "  make db-status       - Check database status"
	@echo ""
	@echo "Production:"
	@echo "  make prod-k8s-setup  - Deploy PostgreSQL to PRODUCTION cluster"
	@echo "  make prod-deploy-cron - Deploy ticker CronJob to PRODUCTION"
	@echo "  make prod-cron-status - Check production CronJob status"
	@echo "  make prod-cron-logs   - View production CronJob logs"
	@echo ""
	@echo "Quality:"
	@echo "  make format          - Format all code"
	@echo "  make lint            - Run linting checks"
	@echo "  make clean           - Clean build artifacts"
	@echo ""
	@echo "Coverage:"
	@echo "  make coverage              - Run all coverage reports"
	@echo "  make coverage-frontend     - Frontend (Jest) coverage"
	@echo "  make coverage-backend      - Backend (Go) coverage"
	@echo "  make coverage-ic-score     - IC Score service (Python) coverage"
	@echo "  make coverage-task-service - Task service (Go) coverage"
	@echo "  make coverage-data-ingestion - Data ingestion service (Go) coverage"

# Complete setup
setup: install db-setup
	@echo "✅ Development environment ready!"
	@echo "Start development with: make dev"

# Install all dependencies
install:
	@echo "Installing dependencies..."
	@command -v brew >/dev/null 2>&1 || { echo "Homebrew required on macOS"; exit 1; }
	npm install
	cd backend && go mod tidy
	python3 -m venv $(VENV_PATH) || true
	. $(VENV_PATH)/bin/activate && pip install -r requirements.txt
	. $(VENV_PATH)/bin/activate && pre-commit install

# Database setup and data import
db-setup:
	@echo "Setting up database..."
	@if ! command -v psql >/dev/null 2>&1; then \
		echo "Installing PostgreSQL..." && \
		brew install postgresql@15; \
	fi
	@export PATH="/opt/homebrew/opt/postgresql@15/bin:$$PATH" && \
	brew services start postgresql@15 || true && \
	sleep 2 && \
	createdb $(DB_NAME) || true && \
	psql $(DB_NAME) -c "CREATE USER $(DB_USER) WITH PASSWORD '$(DB_PASSWORD)';" || true && \
	psql $(DB_NAME) -c "GRANT ALL PRIVILEGES ON DATABASE $(DB_NAME) TO $(DB_USER);" && \
	psql $(DB_NAME) -f backend/migrations/001_create_stock_tables.sql && \
	psql $(DB_NAME) -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $(DB_USER);"
	@echo "Importing stock data..."
	@. $(VENV_PATH)/bin/activate && python scripts/ticker_import_to_db.py

# Development environment
dev:
	@echo "Starting development environment..."
	@echo "Backend: http://localhost:8080"
	@echo "Frontend: http://localhost:3000"
	@cd backend && DB_HOST=localhost DB_PORT=5432 DB_USER=$(DB_USER) DB_PASSWORD=$(DB_PASSWORD) DB_NAME=$(DB_NAME) DB_SSLMODE=disable ./investorcenter-api &
	@npm run dev

# Build everything
build:
	@echo "Building application..."
	cd backend && go build -o investorcenter-api .
	cd task-service && go build -o task-service .
	npm run build

# Complete testing and validation
test:
	@echo "Running tests and validation..."
	. $(VENV_PATH)/bin/activate && pytest scripts/us_tickers/tests/ -v
	cd backend && go test ./...

# Code quality checks
check:
	@echo "Running complete validation..."
	@$(MAKE) format
	@$(MAKE) lint
	@$(MAKE) test
	@echo "✅ All checks passed! Safe to push."

# Format all code
format:
	@echo "Formatting code..."
	. $(VENV_PATH)/bin/activate && black scripts/us_tickers/ scripts/test_*.py scripts/ticker_*.py scripts/update_*.py
	. $(VENV_PATH)/bin/activate && isort scripts/us_tickers/ scripts/test_*.py scripts/ticker_*.py scripts/update_*.py
	cd backend && go fmt ./...

# Linting
lint:
	@echo "Running linting..."
	. $(VENV_PATH)/bin/activate && flake8 scripts/us_tickers/ --max-line-length=79
	. $(VENV_PATH)/bin/activate && mypy scripts/us_tickers/
	cd backend && go vet ./...

# Database operations
db-import:
	@echo "Importing ticker data..."
	. $(VENV_PATH)/bin/activate && python scripts/ticker_import_to_db.py

db-status:
	@./scripts/verify-setup.sh

# Production Kubernetes operations (DO NOT RUN LOCALLY)
prod-k8s-setup:
	@echo "⚠️  PRODUCTION DEPLOYMENT - Ensure you're connected to production cluster!"
	@echo "Current context: $$(kubectl config current-context)"
	@read -p "Continue with production deployment? (y/N): " confirm && [ "$$confirm" = "y" ]
	kubectl apply -f k8s/namespace.yaml
	kubectl create secret generic postgres-secret --from-literal=username=$(DB_USER) --from-literal=password=$(PROD_DB_PASSWORD) -n investorcenter || true
	kubectl apply -f k8s/postgres-deployment.yaml

prod-deploy-cron:
	@echo "⚠️  PRODUCTION CRON DEPLOYMENT"
	@echo "This will deploy ticker update automation to AWS EKS"
	@./scripts/deploy-to-production.sh

prod-cron-status:
	@echo "Production Ticker Update CronJob Status:"
	@echo "========================================"
	@echo "Cluster: $$(kubectl config current-context)"
	kubectl get cronjobs -n investorcenter
	@echo ""
	@echo "Recent Jobs:"
	kubectl get jobs -n investorcenter -l app=ticker-update --sort-by=.metadata.creationTimestamp
	@echo ""
	@echo "To trigger manual run: kubectl create job --from=cronjob/ticker-update manual-ticker-update -n investorcenter"

prod-cron-logs:
	@echo "Production ticker update logs:"
	@kubectl logs -n investorcenter -l app=ticker-update --tail=50

db-import-prod:
	@echo "Setting up production database access..."
	@kubectl port-forward -n investorcenter svc/postgres-service 5433:5432 &
	@sleep 3
	@export DB_HOST=localhost DB_PORT=5433 DB_USER=$(DB_USER) DB_PASSWORD="$(PROD_DB_PASSWORD)" DB_NAME=$(DB_NAME) DB_SSLMODE=disable && \
	. $(VENV_PATH)/bin/activate && python scripts/ticker_import_to_db.py
	@pkill -f "kubectl port-forward.*postgres-service" || true

# Task service targets
dev-task-service:
	@echo "Starting task service on port 8001..."
	cd task-service && go run .

build-task-service:
	@echo "Building task service..."
	cd task-service && go build -o task-service .

test-task-service:
	@echo "Running task service tests..."
	cd task-service && go test ./...

# Test coverage
coverage: coverage-frontend coverage-backend coverage-ic-score
	@echo ""
	@echo "====================================="
	@echo "Coverage reports generated:"
	@echo "  Frontend:  coverage/ (open coverage/lcov-report/index.html)"
	@echo "  Backend:   backend/coverage/ (open backend/coverage/coverage.html)"
	@echo "  IC Score:  ic-score-service/htmlcov/ (open ic-score-service/htmlcov/index.html)"
	@echo "====================================="

coverage-frontend:
	@echo "Running frontend (Jest) coverage..."
	npx jest --coverage

coverage-backend:
	@echo "Running backend (Go) coverage..."
	@mkdir -p backend/coverage
	cd backend && go test ./... -coverprofile=coverage/coverage.out -covermode=atomic
	cd backend && go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	cd backend && go tool cover -func=coverage/coverage.out

coverage-ic-score:
	@echo "Running IC Score service (Python) coverage..."
	cd ic-score-service && ./venv/bin/python -m pytest tests/ pipelines/tests/ \
		--cov=api --cov=pipelines --cov=. \
		--cov-report=term-missing \
		--cov-report=html:htmlcov \
		--cov-report=xml:coverage.xml \
		--rootdir=. \
		-v -o "addopts="

test-integration-ic-score:
	@echo "Running IC Score pipeline integration tests..."
	@echo "Requires: INTEGRATION_TEST_DB=true + PostgreSQL with TimescaleDB"
	cd ic-score-service && INTEGRATION_TEST_DB=true ./venv/bin/python -m pytest \
		pipelines/tests/integration/ -v -o "addopts="

coverage-task-service:
	@echo "Running task service (Go) coverage..."
	@mkdir -p task-service/coverage
	cd task-service && go test ./... -coverprofile=coverage/coverage.out -covermode=atomic
	cd task-service && go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	cd task-service && go tool cover -func=coverage/coverage.out

coverage-data-ingestion:
	@echo "Running data ingestion service (Go) coverage..."
	@mkdir -p data-ingestion-service/coverage
	cd data-ingestion-service && go test ./... -coverprofile=coverage/coverage.out -covermode=atomic
	cd data-ingestion-service && go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	cd data-ingestion-service && go tool cover -func=coverage/coverage.out

# Cleanup
clean:
	@echo "Cleaning build artifacts..."
	rm -f backend/investorcenter-api
	rm -f task-service/task-service
	rm -rf .next/
	cd backend && go clean
	cd task-service && go clean

# Verification
verify:
	@./scripts/verify-setup.sh
