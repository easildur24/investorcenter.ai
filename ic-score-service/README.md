# InvestorCenter.ai - IC Score Service

## Overview

The IC Score Service is a Python-based microservice that provides proprietary stock scoring using 10 financial factors. It uses PostgreSQL 15+ with TimescaleDB for efficient time-series data storage.

## Features

- **IC Score Calculation**: Proprietary 10-factor scoring system (1-100 scale)
- **Financial Data Storage**: Quarterly and annual financial statements from SEC filings
- **Insider & Institutional Tracking**: Form 4 and Form 13F data
- **Analyst Ratings**: Wall Street analyst consensus tracking
- **News Sentiment**: AI-powered sentiment analysis of financial news
- **Time-Series Data**: Efficient storage with TimescaleDB hypertables
- **User Management**: Subscription tiers and preferences
- **Watchlists & Portfolios**: User-created stock lists and portfolio tracking
- **Alerts**: Price, score, and event-based notifications

## Technology Stack

- **Python 3.11+**: Modern async/await patterns
- **SQLAlchemy 2.0**: Modern ORM with full async support
- **PostgreSQL 15+**: Primary database
- **TimescaleDB**: Time-series extension for stock prices and indicators
- **Alembic**: Database migrations
- **asyncpg**: High-performance PostgreSQL driver

## Project Structure

```
ic-score-service/
├── alembic.ini                    # Alembic configuration
├── models.py                      # SQLAlchemy ORM models
├── database/
│   ├── __init__.py
│   ├── database.py                # Database connection and session management
│   └── schema.sql                 # Complete SQL schema
├── migrations/
│   ├── env.py                     # Alembic environment config
│   ├── script.py.mako             # Migration template
│   └── versions/
│       └── 001_initial_schema.py  # Initial migration
├── calculators/                   # IC Score calculation logic (TODO)
├── scripts/
│   └── seed.py                    # Database seeding script
└── tests/                         # Unit tests (TODO)
```

## Prerequisites

1. **Python 3.11+**
   ```bash
   python --version  # Should be 3.11 or higher
   ```

2. **PostgreSQL 15+**
   ```bash
   # macOS (Homebrew)
   brew install postgresql@15
   brew services start postgresql@15

   # Ubuntu/Debian
   sudo apt-get install postgresql-15 postgresql-contrib

   # Verify installation
   psql --version
   ```

3. **TimescaleDB Extension**
   ```bash
   # macOS (Homebrew)
   brew install timescaledb

   # Ubuntu/Debian
   sudo add-apt-repository ppa:timescale/timescaledb-ppa
   sudo apt update
   sudo apt install timescaledb-postgresql-15

   # Enable TimescaleDB
   sudo timescaledb-tune
   ```

## Installation

### 1. Create Virtual Environment

```bash
cd ic-score-service
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
```

### 2. Install Dependencies

```bash
pip install --upgrade pip
pip install sqlalchemy[asyncio] alembic asyncpg psycopg2-binary python-dateutil
```

**Full requirements.txt:**
```txt
sqlalchemy[asyncio]>=2.0.0
alembic>=1.12.0
asyncpg>=0.29.0
psycopg2-binary>=2.9.9
python-dateutil>=2.8.2
```

### 3. Configure Environment Variables

Create a `.env` file in the `ic-score-service/` directory:

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=investorcenter_db
DB_SSLMODE=prefer

# Connection Pool Settings
DB_POOL_SIZE=20
DB_MAX_OVERFLOW=10
DB_POOL_TIMEOUT=30
DB_POOL_RECYCLE=3600

# Debug Settings (development only)
DB_ECHO=false
DB_ECHO_POOL=false
```

Load environment variables:
```bash
export $(cat .env | xargs)
```

### 4. Create Database

```bash
# Connect to PostgreSQL
psql -U postgres

# Create database
CREATE DATABASE investorcenter_db;

# Connect to the database
\c investorcenter_db

# Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "timescaledb";

# Verify TimescaleDB
SELECT default_version FROM pg_available_extensions WHERE name = 'timescaledb';

# Exit psql
\q
```

## Database Setup

### Option 1: Using Alembic Migrations (Recommended)

```bash
# Initialize Alembic (already done, but shown for reference)
alembic init migrations

# Run migrations to create all tables
alembic upgrade head

# Verify migrations
alembic current
alembic history
```

### Option 2: Using SQL Schema Directly

```bash
# Apply schema directly
psql -U postgres -d investorcenter_db -f database/schema.sql

# Verify tables
psql -U postgres -d investorcenter_db -c "\dt"
```

## Seed Sample Data

```bash
# Run seed script to populate with sample data
python scripts/seed.py
```

This creates:
- 2 sample users (demo@investorcenter.ai, free@investorcenter.ai)
- 5 major tech stocks (AAPL, MSFT, GOOGL, AMZN, TSLA)
- 4 quarters of financial data per stock
- 30 days of IC scores per stock
- Analyst ratings, insider trades, institutional holdings
- Sample watchlists, portfolios, and alerts

## Database Health Check

```bash
# Test database connection
python -c "
import asyncio
from database import get_database

async def test():
    db = get_database()
    health = await db.health_check()
    print(health)
    await db.close()

asyncio.run(test())
"
```

Or use the built-in test:
```bash
python database/database.py
```

Expected output:
```
Testing database connection...
Health check: {
    'status': 'healthy',
    'connected': True,
    'database': 'investorcenter_db',
    'host': 'localhost',
    'postgresql_version': '15.3',
    'timescaledb_version': '2.12.0'
}
✓ Database connection successful
  PostgreSQL version: 15.3
  TimescaleDB version: 2.12.0
```

## Database Schema Overview

### Core Tables

- **users**: User accounts with authentication and subscription info
- **companies**: Company master data (ticker, name, sector, etc.)
- **ic_scores**: Proprietary 10-factor stock scores (1-100)

### Financial Data

- **financials**: Quarterly/annual financial statements from SEC
- **insider_trades**: Form 4 insider trading data
- **institutional_holdings**: Form 13F institutional ownership
- **analyst_ratings**: Wall Street analyst ratings and price targets
- **news_articles**: News with AI sentiment analysis

### User Features

- **watchlists** / **watchlist_stocks**: User watchlists
- **portfolios** / **portfolio_positions** / **portfolio_transactions**: Portfolio tracking
- **alerts**: Price, score, and event notifications

### Time-Series (TimescaleDB Hypertables)

- **stock_prices**: OHLCV price data with multiple intervals
- **technical_indicators**: RSI, MACD, and other technical indicators

## Usage Examples

### Query Latest IC Scores

```python
from database import get_session
from models import ICScore
from sqlalchemy import select

async def get_latest_scores():
    async with get_session() as session:
        # Get latest IC scores
        result = await session.execute(
            select(ICScore)
            .order_by(ICScore.date.desc(), ICScore.overall_score.desc())
            .limit(10)
        )
        scores = result.scalars().all()

        for score in scores:
            print(f"{score.ticker}: {score.overall_score}/100 ({score.rating})")
```

### Get Company Financials

```python
from database import get_session
from models import Financial
from sqlalchemy import select

async def get_company_financials(ticker: str):
    async with get_session() as session:
        result = await session.execute(
            select(Financial)
            .where(Financial.ticker == ticker)
            .order_by(Financial.period_end_date.desc())
            .limit(4)
        )
        financials = result.scalars().all()

        for f in financials:
            print(f"Q{f.fiscal_quarter} {f.fiscal_year}: "
                  f"Revenue ${f.revenue:,}, EPS ${f.eps_diluted}")
```

### Create User Watchlist

```python
from database import get_session
from models import Watchlist, WatchlistStock

async def create_watchlist(user_id: str, name: str, tickers: list):
    async with get_session() as session:
        # Create watchlist
        watchlist = Watchlist(
            user_id=user_id,
            name=name,
            description="My custom watchlist"
        )
        session.add(watchlist)
        await session.flush()

        # Add stocks
        for i, ticker in enumerate(tickers):
            stock = WatchlistStock(
                watchlist_id=watchlist.id,
                ticker=ticker,
                position=i + 1
            )
            session.add(stock)

        await session.commit()
        print(f"Created watchlist '{name}' with {len(tickers)} stocks")
```

## Alembic Commands

```bash
# Create a new migration
alembic revision -m "description of changes"

# Apply all pending migrations
alembic upgrade head

# Rollback last migration
alembic downgrade -1

# Show current revision
alembic current

# Show migration history
alembic history --verbose

# Generate migration from model changes (autogenerate)
alembic revision --autogenerate -m "auto-detected changes"
```

## Database Maintenance

### Backup Database

```bash
# Full backup
pg_dump -U postgres investorcenter_db > backup_$(date +%Y%m%d).sql

# Schema only
pg_dump -U postgres --schema-only investorcenter_db > schema_backup.sql

# Data only
pg_dump -U postgres --data-only investorcenter_db > data_backup.sql
```

### Restore Database

```bash
psql -U postgres investorcenter_db < backup_20251112.sql
```

### Vacuum and Analyze

```bash
# Optimize database performance
psql -U postgres -d investorcenter_db -c "VACUUM ANALYZE;"
```

### View Table Sizes

```bash
psql -U postgres -d investorcenter_db -c "
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
"
```

## TimescaleDB Specific Operations

### Hypertable Information

```sql
-- View hypertable info
SELECT * FROM timescaledb_information.hypertables;

-- View chunk info
SELECT * FROM timescaledb_information.chunks;
```

### Compression (Optional)

```sql
-- Enable compression on stock_prices
ALTER TABLE stock_prices SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'ticker'
);

-- Add compression policy (compress data older than 7 days)
SELECT add_compression_policy('stock_prices', INTERVAL '7 days');
```

### Retention Policies

```sql
-- Drop data older than 1 year
SELECT add_retention_policy('stock_prices', INTERVAL '1 year');
```

## Troubleshooting

### Connection Refused

```bash
# Check if PostgreSQL is running
brew services list  # macOS
systemctl status postgresql  # Linux

# Start PostgreSQL
brew services start postgresql@15  # macOS
sudo systemctl start postgresql  # Linux
```

### TimescaleDB Not Found

```bash
# Check if extension is installed
psql -U postgres -c "SELECT * FROM pg_available_extensions WHERE name = 'timescaledb';"

# If not found, reinstall TimescaleDB
brew reinstall timescaledb  # macOS
```

### Permission Denied

```bash
# Grant permissions to user
psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE investorcenter_db TO postgres;"
```

### Migration Conflicts

```bash
# Reset to a specific revision
alembic downgrade <revision>

# Re-run from clean state
alembic downgrade base
alembic upgrade head
```

## Next Steps

1. **Implement IC Score Calculator**: Create the calculation engine in `calculators/`
2. **Add API Layer**: Build FastAPI REST API for accessing IC scores
3. **Implement Data Ingestion**: Create scripts to fetch data from SEC, financial APIs
4. **Add Caching**: Implement Redis caching for frequently accessed scores
5. **Create Tests**: Add unit and integration tests in `tests/`
6. **Deploy**: Package as Docker container and deploy to AWS/GCP

## References

- [SQLAlchemy 2.0 Documentation](https://docs.sqlalchemy.org/en/20/)
- [Alembic Tutorial](https://alembic.sqlalchemy.org/en/latest/tutorial.html)
- [TimescaleDB Documentation](https://docs.timescale.com/)
- [PostgreSQL 15 Documentation](https://www.postgresql.org/docs/15/)

## Support

For issues and questions, please refer to the main InvestorCenter.ai documentation or create an issue in the project repository.
