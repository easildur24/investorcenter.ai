# Makefile Targets Reference

Complete reference for all available make targets in InvestorCenter.ai.

## Setup Targets

### make setup-local
Complete local development environment setup including:
- Install all dependencies (Go, Node.js, Python, PostgreSQL)
- Create and configure local PostgreSQL database
- Run database migrations
- Import 4,600+ US stock tickers
- Configure environment variables

### make setup-prod
Deploy complete production infrastructure:
- Create Kubernetes namespace and secrets
- Deploy PostgreSQL with persistent storage
- Deploy Redis for caching
- Import production data
- Configure production environment

### make install-deps
Install all project dependencies:
- Node.js packages (npm install)
- Go modules (go mod tidy)
- Python packages (requirements.txt)
- PostgreSQL client tools

## Database Targets

### make db-setup-local
Setup local PostgreSQL database:
- Install PostgreSQL 15 via Homebrew
- Create investorcenter_db database
- Create investorcenter user with permissions
- Start PostgreSQL service

### make db-setup-prod
Deploy PostgreSQL to Kubernetes:
- Create namespace and secrets
- Deploy PostgreSQL pod with persistent storage
- Wait for pod to be ready

### make db-migrate
Run database migrations:
- Execute 001_create_stock_tables.sql
- Create 9 financial data tables
- Setup indexes and triggers
- Grant user permissions

### make db-import
Import ticker data to current environment:
- Download latest data from exchanges
- Filter and clean ticker data
- Import 4,600+ US stocks
- Skip existing stocks (incremental safe)

### make db-update
Incremental ticker data update:
- Fetch latest exchange listings
- Add only new companies/IPOs
- Preserve existing data
- Log import statistics

## Development Targets

### make dev
Start complete development environment:
- Start PostgreSQL (if needed)
- Start Go API backend on :8080
- Start Next.js frontend on :3000
- Both services run in parallel

### make dev-backend
Start only the Go API server:
- Connect to local PostgreSQL
- Serve API on http://localhost:8080
- Real database connectivity

### make dev-frontend
Start only the Next.js frontend:
- Development server on :3000
- Hot reloading enabled
- Proxy API requests to backend

### make build
Build all application components:
- Go binary (backend/investorcenter-api)
- Next.js production build
- Optimized for deployment

### make test
Run comprehensive test suite:
- Go unit tests
- Python transformation tests
- Database connection tests
- Integration tests

## Environment Targets

### make local [target]
Run any target in local environment:
- Automatically sets local database variables
- Uses localhost:5432
- Example: make local db-import

### make prod [target]
Run any target in production environment:
- Starts port-forward to K8s database
- Uses localhost:5433 (forwarded)
- Example: make prod db-update

### make status
Show status of all environments:
- Local PostgreSQL connection status
- Production Kubernetes status
- Database stock counts
- Environment configuration

### make verify
Comprehensive setup verification:
- Test all database connections
- Verify data integrity
- Show sample companies
- Environment readiness check

## Utility Targets

### make clean
Clean all build artifacts:
- Remove Go binaries
- Clear Next.js build cache
- Remove Python bytecode
- Clean Docker build cache

### make help
Display help with all available targets and descriptions.

## Production Targets

### make docker-build
Build Docker images:
- Frontend image (Next.js)
- Backend image (Go API)
- Optimized for production

### make docker-push
Push images to container registry:
- Tag with latest version
- Push to ECR or Docker Hub
- Update deployment manifests

### make k8s-deploy
Deploy to Kubernetes:
- Apply all K8s manifests
- Create necessary resources
- Wait for deployments to be ready

### make k8s-cleanup
Remove all Kubernetes resources:
- Delete investorcenter namespace
- Clean up persistent volumes
- Remove secrets and configs

## Environment Management

### Smart Environment Detection
The system automatically detects and configures:
- Local development (localhost:5432)
- Production via port-forward (localhost:5433)
- SSL modes and authentication
- Database credentials from environment or K8s secrets

### Python Environment Manager
```bash
python scripts/env-manager.py status    # Show all environments
python scripts/env-manager.py test      # Test connections
python scripts/env-manager.py set local # Set environment
```

All targets are designed for reliability, automation, and ease of use in both development and production scenarios.
