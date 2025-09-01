# InvestorCenter.ai Makefile

.PHONY: help setup install build dev test check clean

# Configuration
VENV_PATH = path/to/venv
DB_NAME = investorcenter_db
DB_USER = investorcenter

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
	@echo "Quality:"
	@echo "  make format          - Format all code"
	@echo "  make lint            - Run linting checks"
	@echo "  make clean           - Clean build artifacts"

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
	psql $(DB_NAME) -c "CREATE USER $(DB_USER) WITH PASSWORD 'investorcenter123';" || true && \
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
	@cd backend && DB_HOST=localhost DB_PORT=5432 DB_USER=$(DB_USER) DB_PASSWORD=investorcenter123 DB_NAME=$(DB_NAME) DB_SSLMODE=disable ./investorcenter-api &
	@npm run dev

# Build everything
build:
	@echo "Building application..."
	cd backend && go build -o investorcenter-api .
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

# Kubernetes operations
k8s-setup:
	kubectl apply -f k8s/namespace.yaml
	kubectl create secret generic postgres-secret --from-literal=username=$(DB_USER) --from-literal=password=prod_investorcenter_456 -n investorcenter || true
	kubectl apply -f k8s/postgres-deployment.yaml

db-import-prod:
	@echo "Setting up production database access..."
	@kubectl port-forward -n investorcenter svc/postgres-service 5433:5432 &
	@sleep 3
	@export DB_HOST=localhost DB_PORT=5433 DB_USER=$(DB_USER) DB_PASSWORD="prod_investorcenter_456" DB_NAME=$(DB_NAME) DB_SSLMODE=disable && \
	. $(VENV_PATH)/bin/activate && python scripts/ticker_import_to_db.py
	@pkill -f "kubectl port-forward.*postgres-service" || true

# Cleanup
clean:
	@echo "Cleaning build artifacts..."
	rm -f backend/investorcenter-api
	rm -rf .next/
	cd backend && go clean

# Verification
verify:
	@./scripts/verify-setup.sh
