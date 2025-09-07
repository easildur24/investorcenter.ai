# Database Schema Review - Single Table for All Asset Types

## Current Approach: Single `stocks` Table

### ✅ Recommendation: Keep Single Table
Using one table with `asset_type` field to differentiate between stocks, ETFs, crypto, and indices.

## Proposed New Fields to Add

### 🏢 **Stock/ETF Enhanced Fields**
```sql
-- Identifiers
cusip VARCHAR(12)                -- CUSIP number
isin VARCHAR(12)                 -- International Securities ID
lei VARCHAR(20)                  -- Legal Entity Identifier

-- Company details
address_line1 VARCHAR(255)       -- Full address
fiscal_year_end VARCHAR(10)      -- e.g., "12-31"

-- ETF specific
etf_type VARCHAR(50)             -- 'equity', 'bond', 'commodity'
expense_ratio DECIMAL(6,4)       -- 0.0003 = 0.03%
inception_date DATE
issuer VARCHAR(100)              -- 'Vanguard', 'BlackRock'
index_tracked VARCHAR(255)       -- 'S&P 500'
aum DECIMAL(20,2)               -- Assets Under Management
nav DECIMAL(15,4)               -- Net Asset Value
holdings_count INTEGER          -- Number of holdings
```

### 🪙 **Crypto-Specific Fields**
```sql
-- Already have: base_currency_symbol, base_currency_name, currency_symbol
-- New additions:
crypto_type VARCHAR(50)          -- 'coin', 'token', 'stablecoin', 'defi'
blockchain VARCHAR(50)           -- 'bitcoin', 'ethereum', 'solana'
max_supply DECIMAL(30,8)        -- Maximum supply cap
circulating_supply DECIMAL(30,8) -- Current circulating
total_supply DECIMAL(30,8)      -- Total minted
market_cap_rank INTEGER         -- Ranking by market cap
volume_24h DECIMAL(20,2)        -- 24-hour volume
ath DECIMAL(20,8)               -- All-time high price
ath_date TIMESTAMP
atl DECIMAL(20,8)               -- All-time low price
atl_date TIMESTAMP
```

### 📊 **Index-Specific Fields**
```sql
-- Already have: source_feed
-- New additions:
index_type VARCHAR(50)           -- 'broad', 'sector', 'international'
index_level DECIMAL(20,4)        -- Current index value (e.g., 4500.00)
constituent_count INTEGER        -- Number of components
weighting_type VARCHAR(50)       -- 'market-cap', 'equal', 'price'
rebalance_frequency VARCHAR(50)  -- 'quarterly', 'annual'
base_value DECIMAL(20,4)        -- Base index value
base_date DATE                  -- Base date for index
```

### 📈 **Common Trading Metrics (All Types)**
```sql
-- Price & Volume
last_price DECIMAL(20,8)        -- Latest price
volume BIGINT                   -- Trading volume
vwap DECIMAL(20,8)              -- Volume-weighted average

-- Performance tracking
change_1d DECIMAL(20,8)         -- 1-day change
change_percent_1d DECIMAL(10,4)
change_1w DECIMAL(20,8)         -- 1-week change
change_percent_1w DECIMAL(10,4)
change_1m DECIMAL(20,8)         -- 1-month change
change_percent_1m DECIMAL(10,4)
change_1y DECIMAL(20,8)         -- 1-year change
change_percent_1y DECIMAL(10,4)
```

## Key Design Decisions

### 1. **Why Single Table?**
- ✅ Easier to query across asset types
- ✅ Simpler API endpoints
- ✅ Better for mixed portfolios
- ✅ One place to maintain

### 2. **Handling NULL Values**
Each asset type will have NULLs in fields it doesn't use:
- Stocks: NULL in `blockchain`, `expense_ratio`
- Crypto: NULL in `cik`, `employees`, `pe_ratio`
- ETFs: NULL in `blockchain`, `max_supply`
- Indices: NULL in `shares_outstanding`, `employees`

**This is OK!** Application logic handles NULLs appropriately.

### 3. **Asset Type Differentiation**
```sql
asset_type VARCHAR(20) CHECK IN (
    'stock',    -- Common stocks
    'etf',      -- Exchange-traded funds
    'etn',      -- Exchange-traded notes
    'crypto',   -- Cryptocurrencies
    'index',    -- Market indices
    'fund',     -- Mutual funds
    'other'     -- Future types
)
```

## Example Data

### Stock (AAPL)
```sql
symbol: 'AAPL'
asset_type: 'stock'
cik: '0000320193'
shares_outstanding: 15441900000
employees: 164000
-- crypto fields: NULL
-- etf fields: NULL
```

### ETF (SPY)
```sql
symbol: 'SPY'
asset_type: 'etf'
expense_ratio: 0.0945  -- 0.0945%
aum: 400000000000
index_tracked: 'S&P 500'
-- crypto fields: NULL
-- employee fields: NULL
```

### Crypto (X:BTCUSD)
```sql
symbol: 'X:BTCUSD'
asset_type: 'crypto'
base_currency_symbol: 'BTC'
blockchain: 'bitcoin'
max_supply: 21000000
-- stock fields: NULL
-- etf fields: NULL
```

### Index (I:SPX)
```sql
symbol: 'I:SPX'
asset_type: 'index'
index_level: 4500.21
constituent_count: 500
weighting_type: 'market-cap'
-- stock fields: NULL
-- crypto fields: NULL
```

## Migration Impact

### What Changes:
1. Add ~40 new columns
2. Most will be NULL for existing data
3. Polygon API will populate relevant fields

### What Stays Same:
1. Table name remains `stocks`
2. Existing columns unchanged
3. Foreign keys still work
4. No breaking changes

## Questions for Review

1. **Table Name**: Should we rename `stocks` to `assets` or `tickers` to better reflect mixed types?

2. **Performance Fields**: Do we need all performance periods (1d, 1w, 1m, 3m, 6m, 1y)?

3. **Crypto Fields**: Any additional blockchain-specific fields needed?

4. **Index Fields**: Need more index calculation fields?

5. **ETF Fields**: Need tax efficiency or distribution fields?

## Next Steps

If approved:
1. Create migration SQL (003_add_comprehensive_asset_fields.sql)
2. Update Go models
3. Test with sample data
4. Deploy to development first

**Ready to proceed?** Let me know if you want any changes to this schema design!