# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Setup & Development
```bash
make setup        # Complete dev environment setup (deps + DB + data)
make dev          # Start backend (port 8080) + frontend (port 3000)
make build        # Build backend and frontend
```

### Testing & Code Quality
```bash
make test         # Run all tests (Python + Go)
make check        # Complete validation (format + lint + test) before push
make format       # Auto-format Python (black/isort) and Go code
make lint         # Run linting (flake8, mypy, go vet)
```

### Database Operations
```bash
make db-setup     # Setup local PostgreSQL + migrations + import tickers
make db-import    # Import/update ticker data from exchanges
make db-status    # Check database connection and data status
```

### Production Deployment (AWS EKS)
```bash
make prod-deploy-cron    # Deploy ticker update CronJob to AWS EKS
make prod-cron-status    # Check production CronJob status
make prod-cron-logs      # View production CronJob logs
```

## Architecture Overview

### Three-Tier Architecture
- **Frontend**: Next.js 14 with TypeScript, React 18, and Tailwind CSS (port 3000)
- **Backend API**: Go with Gin framework, handles all business logic (port 8080)
- **Database**: PostgreSQL 15 with 9 financial tables and 4,600+ US stocks

### Key Directories
- `app/` - Next.js frontend application with App Router
- `backend/` - Go API server with Gin framework
- `scripts/` - Python data processing and import scripts
- `scripts/us_tickers/` - Stock ticker fetching and validation library
- `k8s/` - Kubernetes manifests for AWS EKS deployment
- `terraform/` - Infrastructure as code for AWS resources

### Database Schema
Nine comprehensive financial tables:
- `stocks` - Company information and metadata
- `stock_prices` - Historical OHLCV price data
- `fundamentals` - Financial metrics (PE, ROE, revenue)
- `earnings` - Quarterly earnings data
- `dividends` - Dividend payment history
- `insider_trades` - Insider trading activity
- `analyst_ratings` - Analyst recommendations
- `financial_statements` - Income statements, balance sheets
- `market_data` - Real-time market information

### API Endpoints
- `GET /api/v1/tickers` - List stocks with pagination and search
- `GET /api/v1/tickers/:symbol` - Stock overview and fundamentals
- `GET /api/v1/tickers/:symbol/chart` - Historical price data
- `GET /api/v1/market/indices` - Market indices overview
- `GET /health` - Application health status

## Development Guidelines

### Python Code Style
- Follow PEP 8 strictly with 79-character line limit
- Use type hints for all functions
- Format with `black` and `isort`
- Test with `pytest` in `scripts/us_tickers/tests/`

### Go Code Style
- Use standard Go formatting with `go fmt`
- Follow Go idioms and best practices
- Database connections via `sqlx` and `pq` driver
- Error handling with proper logging

### Frontend Code Style
- TypeScript with strict type checking
- React functional components with hooks
- Tailwind CSS for styling
- Next.js App Router for routing

### Database Conventions
- Environment-based connection (local vs production)
- Migrations in `backend/migrations/`
- Use prepared statements for all queries
- Proper indexing on frequently queried columns

## Testing Approach
- **Python**: `pytest` with coverage reporting (`scripts/us_tickers/tests/`)
- **Go**: Standard `go test` for backend services
- **CI/CD**: GitHub Actions workflow in `.github/workflows/ci.yml`

## Production Notes
- Kubernetes deployment on AWS EKS
- Terraform manages AWS infrastructure
- CronJob runs ticker updates daily at 1 AM UTC
- Database credentials in Kubernetes secrets
- Container images stored in AWS ECR