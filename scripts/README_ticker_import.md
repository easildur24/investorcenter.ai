# Ticker Database Import

This directory contains scripts for downloading ticker data from exchanges and importing it directly into the PostgreSQL database.

## 🚀 Quick Start

### Prerequisites
```bash
# Install Python dependencies
pip install psycopg2-binary python-dotenv pandas requests

# Set up environment variables in .env or shell:
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=investorcenter
export DB_PASSWORD=your_password
export DB_NAME=investorcenter_db
```

### Import Tickers to Database

**Recommended: Direct Database Import**
```bash
# Preview what will be imported (dry run)
python scripts/ticker_import_to_db.py --dry-run

# Import Nasdaq + NYSE stocks to database
python scripts/ticker_import_to_db.py

# Import only Nasdaq stocks
python scripts/ticker_import_to_db.py --exchanges Q

# Import with detailed logging
python scripts/ticker_import_to_db.py --verbose
```

**For Periodic Updates (Cron)**
```bash
# Add to crontab for daily updates at 6 AM:
0 6 * * * cd /path/to/investorcenter.ai && python scripts/update_tickers_cron.py >> logs/ticker_updates.log 2>&1
```

## 📁 Files Overview

### Main Scripts
- **`ticker_import_to_db.py`** - Interactive script for direct database import
- **`update_tickers_cron.py`** - Cron-friendly script for periodic updates
- **`ticker_db_importer.py`** - Legacy CSV transformation script (still useful)

### Test Scripts
- **`test_direct_import.py`** - Test transformation logic without database
- **`test_ticker_db_importer.py`** - Comprehensive test suite

### Data Files
- **`demo_tickers.csv`** - Original downloaded ticker data (6,916 records)
- **`sample_tickers.csv`** - Small test dataset
- **`transformed_tickers.csv`** - Processed data ready for database (4,642 records)

## 🔄 Import Behavior

### Incremental Updates (ON CONFLICT DO NOTHING)
- ✅ **New tickers**: Inserted into database
- ⏭️ **Existing tickers**: Skipped (preserves existing data)
- 🔒 **No overwrites**: Existing stock data remains unchanged

### Filtering Applied
- ❌ **Warrants**: `AAPL.W`, `MSFT.WS` (contains `.W`, `.WS`)
- ❌ **Preferred Stocks**: `BAC$A`, `JPM$D` (contains `$` or "PREFERRED")
- ❌ **Rights**: `COMPANY.R` (contains `.R` or "RIGHTS")
- ❌ **Notes/Bonds**: Securities containing "NOTES", "SUBORDINATED"
- ❌ **Trust Securities**: Securities containing "TRUST", "DEPOSITARY"
- ✅ **Common Stocks**: Regular equity securities only

## 📊 Data Transformation

### Exchange Code Mapping
- `Q` → `Nasdaq`
- `N` → `NYSE`
- `A` → `NYSE American`
- `P` → `NYSE Arca`
- `Z` → `Cboe`

### Name Cleaning
```
"APPLE INC. - COMMON STOCK" → "APPLE INC."
"TESLA, INC. - CLASS A COMMON STOCK" → "TESLA, INC."
"MICROSOFT CORP" → "MICROSOFT CORP."
```

### Database Schema Mapping
```sql
symbol VARCHAR(10) UNIQUE NOT NULL,    -- Ticker symbol (AAPL)
name VARCHAR(255) NOT NULL,            -- Cleaned company name
exchange VARCHAR(50),                  -- Full exchange name (Nasdaq)
country VARCHAR(50) DEFAULT 'US',      -- Always 'US' for US tickers
currency VARCHAR(3) DEFAULT 'USD',     -- Always 'USD' for US tickers
-- sector, industry, market_cap, etc.  -- Set to NULL (populate later)
```

## 📈 Usage Examples

### One-time Import
```bash
# Import all major exchanges
python scripts/ticker_import_to_db.py --exchanges Q,N,A

# Preview before importing
python scripts/ticker_import_to_db.py --dry-run --verbose
```

### Periodic Updates
```bash
# Weekly update (recommended)
python scripts/update_tickers_cron.py

# Add to crontab for automation:
# Weekly on Sundays at 2 AM:
0 2 * * 0 cd /opt/investorcenter.ai && python scripts/update_tickers_cron.py
```

### Development/Testing
```bash
# Test transformation logic
python scripts/test_direct_import.py

# Test with sample data
python scripts/ticker_db_importer.py --csv scripts/sample_tickers.csv --preview-only
```

## 🚨 Error Handling

The scripts handle common issues gracefully:

- **Database unavailable**: Exits with clear error message
- **Network issues**: Retries with backoff
- **Duplicate data**: Skipped automatically
- **Invalid data**: Filtered out during transformation
- **Missing columns**: Validation with helpful error messages

## 📝 Logging

All scripts support verbose logging:
```bash
python scripts/ticker_import_to_db.py --verbose
```

For cron jobs, redirect output to log files:
```bash
python scripts/update_tickers_cron.py >> logs/ticker_updates.log 2>&1
```

## 🔧 Configuration

### Environment Variables
```bash
# Database connection
DB_HOST=localhost
DB_PORT=5432
DB_USER=investorcenter
DB_PASSWORD=your_password_here
DB_NAME=investorcenter_db
DB_SSLMODE=require

# Optional: Customize behavior
BATCH_SIZE=100
CACHE_TTL_HOURS=24
```

### Performance Tuning
- **Batch size**: Adjust `--batch-size` for large imports
- **Connection pooling**: Configured in Go backend
- **Caching**: Built-in 24-hour cache for downloaded data

## 🎯 Next Steps

1. **Set up PostgreSQL** and run migrations
2. **Test with dry run**: `python scripts/ticker_import_to_db.py --dry-run`
3. **Import initial data**: `python scripts/ticker_import_to_db.py`
4. **Schedule periodic updates**: Add cron job for weekly updates
5. **Monitor logs**: Check for any import issues

The database will contain **4,642+ clean US stock tickers** ready for your InvestorCenter application!
