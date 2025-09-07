# Current vs Polygon Data Format Comparison

## Current Database Schema (`stocks` table)

```sql
CREATE TABLE stocks (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    exchange VARCHAR(50),
    sector VARCHAR(100),
    industry VARCHAR(100),
    country VARCHAR(50) DEFAULT 'US',
    currency VARCHAR(3) DEFAULT 'USD',
    market_cap DECIMAL(20,2),
    description TEXT,
    website VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

## Example: Current Data Format (from NASDAQ/NYSE files)

### 1. Common Stock (AAPL)
```json
{
  "symbol": "AAPL",
  "name": "APPLE INC",
  "exchange": "NASDAQ",
  "sector": null,              // Not provided by exchange files
  "industry": null,             // Not provided by exchange files  
  "country": "US",
  "currency": "USD",
  "market_cap": null,           // Not provided by exchange files
  "description": null,          // Not provided by exchange files
  "website": null               // Not provided by exchange files
}
```

### 2. ETF (SPY) - Currently mixed with stocks
```json
{
  "symbol": "SPY",
  "name": "SPDR S&P 500 ETF TRUST",
  "exchange": "NYSE ARCA",
  "sector": null,
  "industry": null,
  "country": "US", 
  "currency": "USD",
  "market_cap": null,
  "description": null,
  "website": null
}
```

## New: Polygon API Data Format

### 1. Common Stock (AAPL)
```json
{
  // From Polygon API
  "ticker": "AAPL",
  "name": "Apple Inc.",
  "market": "stocks",
  "locale": "us",
  "primary_exchange": "XNAS",
  "type": "CS",                    // Common Stock
  "active": true,
  "currency_name": "usd",
  "cik": "0000320193",
  "composite_figi": "BBG000B9XRY4",
  "share_class_figi": "BBG001S5N8V8",
  "market_cap": 2950000000000,
  "phone_number": "(408) 996-1010",
  "address": {
    "address1": "One Apple Park Way",
    "city": "Cupertino",
    "state": "CA",
    "postal_code": "95014"
  },
  "description": "Apple Inc. designs, manufactures...",
  "sic_code": "3571",
  "sic_description": "Electronic Computers",
  "ticker_root": "AAPL",
  "homepage_url": "https://www.apple.com",
  "total_employees": 164000,
  "list_date": "1980-12-12",
  "branding": {
    "logo_url": "https://api.polygon.io/v1/reference/company-branding/YXBwbGUuY29t/images/2024-06-01_logo.svg",
    "icon_url": "https://api.polygon.io/v1/reference/company-branding/YXBwbGUuY29t/images/2024-06-01_icon.jpeg"
  },
  "weighted_shares_outstanding": 15441900000,
  
  // Maps to database as:
  "symbol": "AAPL",
  "name": "Apple Inc.",
  "exchange": "NASDAQ",            // Mapped from XNAS
  "sector": "Technology",          // Mapped from SIC code
  "industry": "Electronic Computers",
  "country": "US",
  "currency": "USD",
  "market_cap": 2950000000000.00,
  "description": "Apple Inc. designs, manufactures...",
  "website": "https://www.apple.com",
  
  // NEW FIELDS TO ADD:
  "asset_type": "stock",           // From type: "CS"
  "cik": "0000320193",
  "ipo_date": "1980-12-12",
  "logo_url": "https://api.polygon.io/...",
  "employees": 164000,
  "sic_code": "3571"
}
```

### 2. ETF (SPY)
```json
{
  // From Polygon API
  "ticker": "SPY",
  "name": "SPDR S&P 500 ETF Trust",
  "market": "stocks",
  "locale": "us",
  "primary_exchange": "ARCX",
  "type": "ETF",                    // Exchange Traded Fund
  "active": true,
  "currency_name": "usd",
  "cik": "0001118190",
  "composite_figi": "BBG000BDTBL9",
  "share_class_figi": "BBG001S9KZ14",
  "list_date": "1993-01-22",
  
  // Maps to database as:
  "symbol": "SPY",
  "name": "SPDR S&P 500 ETF Trust",
  "exchange": "NYSE ARCA",
  "sector": "ETF",                  // Special sector for ETFs
  "industry": "Equity Fund",
  "country": "US",
  "currency": "USD",
  "asset_type": "etf",              // From type: "ETF"
  "cik": "0001118190",
  "ipo_date": "1993-01-22"
}
```

### 3. Crypto (BTC-USD)
```json
{
  // From Polygon API
  "ticker": "X:BTCUSD",
  "name": "Bitcoin - United States dollar",
  "market": "crypto",
  "locale": "global",
  "active": true,
  "currency_symbol": "USD",
  "currency_name": "United States dollar",
  "base_currency_symbol": "BTC",
  "base_currency_name": "Bitcoin",
  
  // Maps to database as:
  "symbol": "X:BTCUSD",
  "name": "Bitcoin - USD",
  "exchange": "CRYPTO",
  "sector": "Cryptocurrency",
  "industry": "Digital Currency",
  "country": "GLOBAL",
  "currency": "USD",
  "asset_type": "crypto"
}
```

### 4. Index (S&P 500)
```json
{
  // From Polygon API
  "ticker": "I:SPX",
  "name": "S&P 500 Index",
  "market": "indices",
  "locale": "us",
  "active": true,
  "source_feed": "S&P",
  
  // Maps to database as:
  "symbol": "I:SPX",
  "name": "S&P 500 Index",
  "exchange": "INDEX",
  "sector": "Index",
  "industry": "Market Index",
  "country": "US",
  "currency": "USD",
  "asset_type": "index"
}
```

## Required Database Schema Changes

```sql
-- Add new columns to support additional asset types and metadata
ALTER TABLE stocks ADD COLUMN asset_type VARCHAR(20) DEFAULT 'stock';
ALTER TABLE stocks ADD COLUMN cik VARCHAR(20);
ALTER TABLE stocks ADD COLUMN ipo_date DATE;
ALTER TABLE stocks ADD COLUMN logo_url TEXT;
ALTER TABLE stocks ADD COLUMN icon_url TEXT;
ALTER TABLE stocks ADD COLUMN primary_exchange_code VARCHAR(10);
ALTER TABLE stocks ADD COLUMN figi VARCHAR(20);
ALTER TABLE stocks ADD COLUMN share_class_figi VARCHAR(20);
ALTER TABLE stocks ADD COLUMN sic_code VARCHAR(10);
ALTER TABLE stocks ADD COLUMN employees INTEGER;
ALTER TABLE stocks ADD COLUMN address_city VARCHAR(100);
ALTER TABLE stocks ADD COLUMN address_state VARCHAR(50);

-- Add index for asset_type for faster filtering
CREATE INDEX idx_stocks_asset_type ON stocks(asset_type);

-- Add check constraint for asset types
ALTER TABLE stocks ADD CONSTRAINT check_asset_type 
  CHECK (asset_type IN ('stock', 'etf', 'crypto', 'index', 'etn', 'fund'));
```

## Data Migration Summary

### Before (Exchange Files)
- **Source**: NASDAQ & NYSE FTP files
- **Coverage**: ~7,000 US stocks only
- **Data Points**: Symbol, Name, Exchange only
- **ETF Handling**: Mixed with stocks, hard to distinguish
- **Updates**: Manual download from FTP

### After (Polygon API)
- **Source**: Polygon.io unified API
- **Coverage**: 
  - ~10,000+ US stocks
  - ~3,000+ ETFs
  - ~20,000+ crypto pairs
  - ~100+ indices
- **Data Points**: 20+ fields including CIK, market cap, employees, logos
- **ETF Handling**: Clearly identified with `type: "ETF"`
- **Updates**: Real-time API with pagination

## Example Query Differences

### Current: Get all ETFs
```sql
-- Difficult - ETFs mixed with stocks, no clear identifier
SELECT * FROM stocks 
WHERE name LIKE '%ETF%' OR name LIKE '%FUND%';
```

### New: Get all ETFs
```sql
-- Easy - clear asset_type field
SELECT * FROM stocks 
WHERE asset_type = 'etf';
```

### New: Get stocks by type
```sql
-- Stocks only (no ETFs)
SELECT * FROM stocks WHERE asset_type = 'stock';

-- All tradeable equities (stocks + ETFs)
SELECT * FROM stocks WHERE asset_type IN ('stock', 'etf');

-- Crypto pairs
SELECT * FROM stocks WHERE asset_type = 'crypto';

-- Market indices
SELECT * FROM stocks WHERE asset_type = 'index';
```