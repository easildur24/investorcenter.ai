# InvestorCenter.ai Quick Start Guide

Get InvestorCenter running in under 5 minutes with automated setup.

## Prerequisites

- macOS with Homebrew
- Git

## One-Command Setup

```bash
# Clone repository
git clone <repository-url> investorcenter.ai
cd investorcenter.ai

# Complete local development setup
make setup-local
```

This single command will:
1. Install all dependencies (Node.js, Go, Python, PostgreSQL)
2. Setup local PostgreSQL database
3. Run database migrations (9 financial tables)
4. Import 4,600+ US stock tickers from exchanges
5. Configure environment variables

## Start Development

```bash
# Start both backend and frontend
make dev

# Access your application:
# - Backend API: http://localhost:8080
# - Frontend: http://localhost:3000
# - Health check: http://localhost:8080/health
```

## Verify Setup

```bash
# Check everything is working
make verify

# Expected output:
# Local PostgreSQL: 4643 stocks
# Production PostgreSQL: Available
# All tests: PASS
```

## Key Features Ready

After setup, you have:
- PostgreSQL database with 4,643 US stocks (AAPL, MSFT, GOOGL, etc.)
- Go API backend with database connectivity
- Next.js frontend framework
- Environment management for local/production
- Automated ticker data updates

## Common Commands

```bash
make help           # Show all available commands
make status         # Check environment status
make test           # Run all tests
make db-update      # Update ticker data
make clean          # Clean build artifacts
```

## Production Deployment

```bash
# Deploy to Kubernetes (requires AWS setup)
make setup-prod

# Switch between environments
make local db-import    # Use local database
make prod db-import     # Use production database
```

## Troubleshooting

If setup fails:
```bash
# Install missing dependencies
make install-deps

# Reset and retry
make clean
make setup-local
```

For detailed setup instructions, see:
- LOCAL_DEVELOPMENT_SETUP.md - Local development guide
- PRODUCTION_SETUP.md - Kubernetes deployment guide
- README_DATABASE_SETUP.md - Database configuration details

Your professional financial platform is ready for development!
