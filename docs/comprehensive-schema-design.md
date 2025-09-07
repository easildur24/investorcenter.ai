# Comprehensive Database Schema Design for All Asset Types

## Overview
Single unified `stocks` table to store all asset types (stocks, ETFs, crypto, indices) with type-specific fields.

## Schema Design

### Core Fields (All Asset Types)
```sql
-- These fields apply to ALL asset types
symbol VARCHAR(20) UNIQUE NOT NULL    -- e.g., "AAPL", "SPY", "X:BTCUSD", "I:SPX"
name VARCHAR(255) NOT NULL
asset_type VARCHAR(20) NOT NULL       -- 'stock', 'etf', 'crypto', 'index'
market VARCHAR(20)                    -- 'stocks', 'crypto', 'indices'
locale VARCHAR(10)                    -- 'us', 'global'
active BOOLEAN DEFAULT true
currency VARCHAR(10)                  -- Trading currency (USD, EUR, etc.)
description TEXT
logo_url TEXT
icon_url TEXT
website VARCHAR(255)
last_updated TIMESTAMP
created_at TIMESTAMP DEFAULT NOW()
updated_at TIMESTAMP DEFAULT NOW()
```

### Stock & ETF Specific Fields
```sql
-- Traditional equity fields (stocks & ETFs)
exchange VARCHAR(50)                  -- NYSE, NASDAQ, etc.
primary_exchange_code VARCHAR(10)     -- XNAS, XNYS, ARCX
cik VARCHAR(20)                       -- SEC filing identifier
composite_figi VARCHAR(20)            -- Financial instrument identifier
share_class_figi VARCHAR(20)
cusip VARCHAR(12)                     -- CUSIP identifier
isin VARCHAR(12)                      -- International Securities ID
lei VARCHAR(20)                       -- Legal Entity Identifier

-- Company information
sector VARCHAR(100)
industry VARCHAR(100)
sic_code VARCHAR(10)
sic_description VARCHAR(255)
country VARCHAR(50) DEFAULT 'US'
employees INTEGER
phone_number VARCHAR(50)
address_line1 VARCHAR(255)
address_city VARCHAR(100)
address_state VARCHAR(50)
address_postal VARCHAR(20)
address_country VARCHAR(50)

-- Financial metrics
market_cap DECIMAL(20,2)
shares_outstanding BIGINT
shares_float BIGINT
weighted_shares_outstanding BIGINT
ipo_date DATE
fiscal_year_end VARCHAR(10)

-- ETF specific
etf_type VARCHAR(50)                  -- 'equity', 'bond', 'commodity', 'currency'
expense_ratio DECIMAL(6,4)            -- e.g., 0.0003 = 0.03%
inception_date DATE
issuer VARCHAR(100)                   -- e.g., 'Vanguard', 'BlackRock'
index_tracked VARCHAR(255)            -- e.g., 'S&P 500'
aum DECIMAL(20,2)                     -- Assets Under Management
nav DECIMAL(15,4)                     -- Net Asset Value
holdings_count INTEGER                -- Number of holdings
```

### Crypto Specific Fields
```sql
-- Cryptocurrency fields
base_currency_symbol VARCHAR(20)      -- e.g., 'BTC'
base_currency_name VARCHAR(100)       -- e.g., 'Bitcoin'
quote_currency_symbol VARCHAR(20)     -- e.g., 'USD'
quote_currency_name VARCHAR(100)      -- e.g., 'US Dollar'
crypto_type VARCHAR(50)               -- 'coin', 'token', 'stablecoin', 'defi'
blockchain VARCHAR(50)                -- 'bitcoin', 'ethereum', 'solana'
consensus_mechanism VARCHAR(50)       -- 'proof-of-work', 'proof-of-stake'
max_supply DECIMAL(30,8)              -- Maximum supply (if applicable)
circulating_supply DECIMAL(30,8)      -- Current circulating supply
total_supply DECIMAL(30,8)            -- Total supply
market_cap_rank INTEGER               -- Ranking by market cap
volume_24h DECIMAL(20,2)              -- 24-hour trading volume
high_24h DECIMAL(20,8)
low_24h DECIMAL(20,8)
price_change_24h DECIMAL(20,8)
price_change_percentage_24h DECIMAL(10,4)
ath DECIMAL(20,8)                     -- All-time high
ath_date TIMESTAMP
atl DECIMAL(20,8)                     -- All-time low
atl_date TIMESTAMP
```

### Index Specific Fields
```sql
-- Market index fields
index_type VARCHAR(50)                -- 'broad', 'sector', 'international'
source_feed VARCHAR(100)              -- Data provider
index_level DECIMAL(20,4)             -- Current index value
constituent_count INTEGER             -- Number of constituents
weighting_type VARCHAR(50)            -- 'market-cap', 'equal', 'price'
rebalance_frequency VARCHAR(50)       -- 'quarterly', 'annual'
base_value DECIMAL(20,4)              -- Base index value
base_date DATE                        -- Base date for index
divisor DECIMAL(30,10)                -- Index divisor
methodology_url TEXT                  -- Link to index methodology
```

### Trading & Performance Metrics
```sql
-- Common trading metrics
last_price DECIMAL(20,8)
open_price DECIMAL(20,8)
high_price DECIMAL(20,8)
low_price DECIMAL(20,8)
close_price DECIMAL(20,8)
volume BIGINT
dollar_volume DECIMAL(20,2)
vwap DECIMAL(20,8)                    -- Volume-weighted average price
bid DECIMAL(20,8)
ask DECIMAL(20,8)
bid_size INTEGER
ask_size INTEGER
spread DECIMAL(20,8)

-- Performance metrics
change_1d DECIMAL(20,8)
change_percent_1d DECIMAL(10,4)
change_1w DECIMAL(20,8)
change_percent_1w DECIMAL(10,4)
change_1m DECIMAL(20,8)
change_percent_1m DECIMAL(10,4)
change_3m DECIMAL(20,8)
change_percent_3m DECIMAL(10,4)
change_6m DECIMAL(20,8)
change_percent_6m DECIMAL(10,4)
change_1y DECIMAL(20,8)
change_percent_1y DECIMAL(10,4)
change_ytd DECIMAL(20,8)
change_percent_ytd DECIMAL(10,4)

-- Technical indicators
rsi_14 DECIMAL(6,2)                   -- 14-day RSI
ma_50 DECIMAL(20,8)                   -- 50-day moving average
ma_200 DECIMAL(20,8)                  -- 200-day moving average
beta DECIMAL(6,3)                     -- Beta coefficient
volatility_30d DECIMAL(10,4)          -- 30-day volatility
```

### Status & Metadata
```sql
-- Status tracking
delisted_date DATE
delisted_reason VARCHAR(255)
halt_status VARCHAR(50)               -- 'trading', 'halted', 'suspended'
halt_reason VARCHAR(255)
last_trade_date TIMESTAMP
data_quality_score INTEGER            -- 0-100 quality score
data_source VARCHAR(50)               -- 'polygon', 'manual', etc.
```

## Proposed Table Structure

```sql
CREATE TABLE IF NOT EXISTS stocks_unified (
    -- Primary Key
    id SERIAL PRIMARY KEY,
    
    -- Core Identifiers (All Types)
    symbol VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    asset_type VARCHAR(20) NOT NULL,
    market VARCHAR(20),
    locale VARCHAR(10) DEFAULT 'us',
    active BOOLEAN DEFAULT true,
    currency VARCHAR(10) DEFAULT 'USD',
    
    -- Stock/ETF Fields
    exchange VARCHAR(50),
    primary_exchange_code VARCHAR(10),
    cik VARCHAR(20),
    composite_figi VARCHAR(20),
    share_class_figi VARCHAR(20),
    cusip VARCHAR(12),
    isin VARCHAR(12),
    lei VARCHAR(20),
    
    -- Company Info
    sector VARCHAR(100),
    industry VARCHAR(100),
    sic_code VARCHAR(10),
    sic_description VARCHAR(255),
    country VARCHAR(50) DEFAULT 'US',
    employees INTEGER,
    phone_number VARCHAR(50),
    address_line1 VARCHAR(255),
    address_city VARCHAR(100),
    address_state VARCHAR(50),
    address_postal VARCHAR(20),
    
    -- Financial Metrics
    market_cap DECIMAL(20,2),
    shares_outstanding BIGINT,
    shares_float BIGINT,
    ipo_date DATE,
    
    -- ETF Specific
    etf_type VARCHAR(50),
    expense_ratio DECIMAL(6,4),
    inception_date DATE,
    issuer VARCHAR(100),
    index_tracked VARCHAR(255),
    aum DECIMAL(20,2),
    nav DECIMAL(15,4),
    holdings_count INTEGER,
    
    -- Crypto Specific
    base_currency_symbol VARCHAR(20),
    base_currency_name VARCHAR(100),
    quote_currency_symbol VARCHAR(20),
    quote_currency_name VARCHAR(100),
    crypto_type VARCHAR(50),
    blockchain VARCHAR(50),
    max_supply DECIMAL(30,8),
    circulating_supply DECIMAL(30,8),
    total_supply DECIMAL(30,8),
    market_cap_rank INTEGER,
    
    -- Index Specific
    index_type VARCHAR(50),
    source_feed VARCHAR(100),
    index_level DECIMAL(20,4),
    constituent_count INTEGER,
    weighting_type VARCHAR(50),
    rebalance_frequency VARCHAR(50),
    
    -- Trading Metrics
    last_price DECIMAL(20,8),
    volume BIGINT,
    change_1d DECIMAL(20,8),
    change_percent_1d DECIMAL(10,4),
    
    -- Metadata
    description TEXT,
    logo_url TEXT,
    website VARCHAR(255),
    delisted_date DATE,
    last_updated TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT check_asset_type CHECK (
        asset_type IN ('stock', 'etf', 'etn', 'fund', 'crypto', 'index', 'other')
    )
);

-- Indexes for performance
CREATE INDEX idx_unified_asset_type ON stocks_unified(asset_type);
CREATE INDEX idx_unified_symbol_type ON stocks_unified(symbol, asset_type);
CREATE INDEX idx_unified_active ON stocks_unified(active) WHERE active = true;
CREATE INDEX idx_unified_market_cap ON stocks_unified(market_cap) WHERE market_cap IS NOT NULL;
CREATE INDEX idx_unified_crypto ON stocks_unified(base_currency_symbol) WHERE asset_type = 'crypto';
CREATE INDEX idx_unified_etf ON stocks_unified(expense_ratio) WHERE asset_type = 'etf';
```

## Advantages of Single Table Approach

1. **Simplified Queries**: Easy to search across all asset types
2. **Unified API**: Single endpoint can return mixed results
3. **Easier Maintenance**: One table to manage
4. **Flexible Filtering**: Use `asset_type` to filter by type
5. **Better for Portfolios**: Mixed asset portfolios in one query

## Handling NULL Values

Since different asset types use different fields, many will be NULL:
- Stocks won't have `base_currency_symbol`
- Crypto won't have `cik` or `pe_ratio`
- Indices won't have `shares_outstanding`

This is acceptable and can be handled in application logic.

## Alternative: Separate Tables (Not Recommended)

If we used separate tables:
```sql
stocks_equity (symbol, cik, shares_outstanding, ...)
stocks_etf (symbol, expense_ratio, nav, ...)
stocks_crypto (symbol, base_currency, blockchain, ...)
stocks_index (symbol, index_level, constituent_count, ...)
```

Disadvantages:
- Complex JOINs for mixed queries
- Duplicate code in application
- Harder to add new asset types
- More complex API endpoints

## Migration Strategy

1. **Phase 1**: Add new columns (current approach)
2. **Phase 2**: Migrate existing data
3. **Phase 3**: Populate from Polygon API
4. **Phase 4**: Add validation constraints

## Sample Queries

```sql
-- Get all ETFs
SELECT * FROM stocks WHERE asset_type = 'etf';

-- Get top crypto by market cap
SELECT * FROM stocks 
WHERE asset_type = 'crypto' 
ORDER BY market_cap DESC LIMIT 10;

-- Mixed portfolio search
SELECT symbol, name, asset_type, last_price, change_percent_1d
FROM stocks 
WHERE symbol IN ('AAPL', 'SPY', 'X:BTCUSD', 'I:SPX');

-- Find all tech stocks and ETFs
SELECT * FROM stocks 
WHERE asset_type IN ('stock', 'etf') 
AND sector = 'Technology';
```

## Recommendation

✅ **Keep single table approach** with `asset_type` field for differentiation
- Simpler to implement and maintain
- Better for mixed-asset queries
- Easier API development
- NULL fields are acceptable trade-off

The schema above provides comprehensive support for all asset types while maintaining simplicity.