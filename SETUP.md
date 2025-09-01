# InvestorCenter.ai Setup Guide

Complete setup for local development and database management.

## Quick Start

```bash
# Clone and setup everything
git clone <repository-url> investorcenter.ai
cd investorcenter.ai
make setup

# Start development  
make dev
```

## Prerequisites

- **macOS/Linux** with package manager (Homebrew/apt)
- **Python 3.10+** with pip
- **Node.js 18+** with npm
- **Go 1.19+**

## Setup Commands

### Complete Environment Setup
```bash
make setup          # Install all dependencies + database + data import
make dev            # Start both backend (Go) and frontend (Next.js)
```

### Individual Setup Steps
```bash
make install        # Install all dependencies (Node.js, Go, Python, PostgreSQL)
make db-setup       # Setup PostgreSQL + run migrations + import tickers
make build          # Build backend and frontend
```

## Database Setup

### Local PostgreSQL (Automatic)
```bash
make db-setup       # Installs PostgreSQL, creates database, imports 4,600+ tickers
```

This automatically:
1. Installs PostgreSQL 15 via package manager
2. Creates `investorcenter_db` database  
3. Creates `investorcenter` user with password
4. Runs all migrations (9 financial tables)
5. Imports 4,600+ US stock tickers from exchanges

### Production PostgreSQL (Kubernetes)
```bash
make k8s-setup      # Deploy PostgreSQL to Kubernetes
make db-import-prod # Import data to production database
```

## Development Workflow

### Daily Development
```bash
make dev            # Start full development environment
make test           # Run all tests + linting
make check          # Complete validation before push
```

### Database Operations
```bash
make db-import      # Import/update ticker data
make db-status      # Check database status
```

### Code Quality
```bash
make format         # Auto-format all code
make lint           # Run linting checks
make check          # Complete validation (format + lint + test)
```

## Environment Configuration

The application automatically detects the best available database:
- **Local**: PostgreSQL on localhost:5432 (preferred for development)
- **Production**: Kubernetes PostgreSQL via port-forward

No manual environment switching required.

## Troubleshooting

### Database Issues
```bash
make db-status      # Check database connection and data
make verify         # Complete system verification
```

### Code Quality Issues
```bash
make format         # Fix most formatting issues automatically
make check          # Catch CI issues before push
```

### Build Issues
```bash
make clean          # Clean build artifacts
make install        # Reinstall dependencies
```

## What Gets Installed

### Dependencies
- **PostgreSQL 15**: Database server with 9 financial tables
- **Go modules**: Backend API dependencies
- **Node.js packages**: Frontend dependencies  
- **Python packages**: Data processing + linting tools

### Data
- **4,600+ US stocks** from Nasdaq and NYSE exchanges
- **Smart filtering**: Excludes derivatives, ETFs, test issues
- **Clean data**: Proper company names and exchange mapping

### Development Tools
- **Automatic formatting**: Black + isort + go fmt
- **Linting**: flake8 + mypy + go vet + bandit
- **Testing**: 45+ comprehensive tests
- **Pre-commit hooks**: Automatic code quality enforcement

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Next.js       │    │    Go API        │    │   PostgreSQL    │
│   Frontend      │◄──►│    Backend       │◄──►│   Database      │
│   localhost:3000│    │   localhost:8080 │    │  localhost:5432 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

For production deployment to AWS EKS, see [DEPLOYMENT.md](DEPLOYMENT.md).
