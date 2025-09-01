# Local Development Setup

This guide covers setting up InvestorCenter for local development with PostgreSQL.

## Quick Start

### Prerequisites
- macOS with Homebrew
- Python 3.8+ with venv
- Go 1.19+
- Git

### Complete Setup (One Command)
```bash
git clone <repository-url> investorcenter.ai
cd investorcenter.ai

# Install dependencies, setup PostgreSQL, run migrations, import data
make setup-local

# Start development environment
make dev
```

## Manual Setup Steps

### 1. Install Dependencies
```bash
make install-deps
```

### 2. Setup Local Database
```bash
make db-setup-local
make db-migrate
```

### 3. Import Ticker Data
```bash
# Import 4,600+ US stocks from exchanges
make db-import
```

### 4. Start Development
```bash
# Start both backend and frontend
make dev

# Or start individually:
make dev-backend    # Go API server on :8080
make dev-frontend   # Next.js on :3000
```

## Database Management

### Import and Update Data
```bash
# Import fresh ticker data
make db-import

# Incremental updates (only new stocks)
make db-update
```

### Direct Database Access
```bash
# Access PostgreSQL directly
export PATH="/opt/homebrew/opt/postgresql@15/bin:$PATH"
psql investorcenter_db

# Useful queries:
SELECT COUNT(*) FROM stocks;
SELECT exchange, COUNT(*) FROM stocks GROUP BY exchange;
SELECT * FROM stocks WHERE symbol = 'AAPL';
```

### Backup and Restore
```bash
# Create backup
pg_dump investorcenter_db > backup_$(date +%Y%m%d).sql

# Restore from backup
psql investorcenter_db < backup_20250831.sql
```

## Development Workflow

### Daily Development
```bash
# 1. Start PostgreSQL (if not running)
brew services start postgresql@15

# 2. Start development environment
make dev

# 3. Develop at:
#    Backend:  http://localhost:8080
#    Frontend: http://localhost:3000
```

### Weekly Data Updates
```bash
# Update with latest stock listings
make db-update
```

### Testing
```bash
# Run all tests
make test

# Verify setup
make verify

# Check environment status
make status
```

## Environment Configuration

The setup creates:
- PostgreSQL 15 database with 4,600+ US stocks
- Go API backend with database connectivity
- Environment variables automatically configured
- Smart environment detection and switching

### Database Details
- Host: localhost:5432
- Database: investorcenter_db
- User: investorcenter
- Password: investorcenter123
- SSL: disabled (local development)

### Database Schema
- stocks - Company information (4,643+ US stocks)
- stock_prices - Price history
- fundamentals - Financial metrics
- earnings - Quarterly earnings data
- analyst_ratings - Analyst recommendations
- news_articles - News and sentiment
- dividends - Dividend history
- insider_trading - Insider activity
- technical_indicators - Technical analysis data

## Troubleshooting

### PostgreSQL Issues
```bash
# Restart PostgreSQL
brew services restart postgresql@15

# Check service status
brew services list | grep postgresql

# Test connection
make verify
```

### Build Issues
```bash
# Rebuild everything
make clean
make build
```

### Database Issues
```bash
# Reset database
dropdb investorcenter_db
make db-setup-local
make db-import
```

## Advanced Usage

### Environment Management
```bash
# Check current environment status
python scripts/env-manager.py status

# Test database connections
python scripts/env-manager.py test local
python scripts/env-manager.py test prod
```

### Custom Targets
```bash
# Build specific components
make build-backend
make build-frontend

# Run specific tests
make test-backend
make test-python
```

Ready for InvestorCenter development with enterprise-grade data infrastructure.
