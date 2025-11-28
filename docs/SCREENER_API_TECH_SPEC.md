# Tech Spec: Stock Screener API (`/api/v1/screener/stocks`)

## Problem Statement

The frontend screener page (`app/screener/page.tsx`) calls `/api/v1/screener/stocks` but this endpoint doesn't exist. Currently, the page falls back to 30 hardcoded mock stocks, providing a poor user experience.

## Goal

Implement a real `/api/v1/screener/stocks` endpoint in the Go backend that:
1. Returns all stocks with fundamental data for screening
2. Supports server-side filtering and sorting
3. Supports pagination for performance
4. Includes IC Score data when available

## Data Sources

### Primary Tables (Backend DB)
- `tickers` - Stock metadata (symbol, name, sector, industry, market_cap, asset_type)
- `fundamentals` - Financial metrics (pe, pb, ps, roe, revenue, etc.)
- `dividends` - Dividend data (yield)
- `stock_prices` - Latest price and change_percent

### Secondary Tables (IC Score DB - same PostgreSQL)
- `ic_scores` - IC Score ratings (overall_score, value_score, growth_score, etc.)
- `companies` - Additional company metadata

## API Design

### Endpoint
```
GET /api/v1/screener/stocks
```

### Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `page` | int | 1 | Page number |
| `limit` | int | 50 | Results per page (max 100) |
| `sort` | string | `market_cap` | Sort field |
| `order` | string | `desc` | Sort order (`asc`/`desc`) |
| `sectors` | string | - | Comma-separated sector filter |
| `market_cap_min` | float | - | Min market cap |
| `market_cap_max` | float | - | Max market cap |
| `pe_min` | float | - | Min P/E ratio |
| `pe_max` | float | - | Max P/E ratio |
| `dividend_yield_min` | float | - | Min dividend yield % |
| `dividend_yield_max` | float | - | Max dividend yield % |
| `revenue_growth_min` | float | - | Min revenue growth % |
| `revenue_growth_max` | float | - | Max revenue growth % |
| `ic_score_min` | float | - | Min IC Score (0-100) |
| `ic_score_max` | float | - | Max IC Score (0-100) |
| `asset_type` | string | `stock` | Filter by asset type |

### Response Schema
```json
{
  "data": [
    {
      "symbol": "AAPL",
      "name": "Apple Inc.",
      "sector": "Technology",
      "industry": "Consumer Electronics",
      "market_cap": 2800000000000,
      "price": 175.43,
      "change_percent": 1.25,
      "pe_ratio": 28.5,
      "pb_ratio": 45.2,
      "ps_ratio": 7.8,
      "roe": 147.5,
      "revenue_growth": 8.1,
      "earnings_growth": 12.3,
      "dividend_yield": 0.55,
      "beta": 1.25,
      "ic_score": 78
    }
  ],
  "meta": {
    "total": 4650,
    "page": 1,
    "limit": 50,
    "total_pages": 93,
    "timestamp": "2025-01-15T10:30:00Z"
  }
}
```

## Implementation Plan

### Phase 1: Backend Implementation

#### 1.1 Add New Handler File
Create `backend/handlers/screener.go`:
- `GetScreenerStocks(c *gin.Context)` - main handler
- Parse and validate query parameters
- Build dynamic SQL query with filters
- Return paginated results

#### 1.2 Add Database Query
Create `backend/database/screener.go`:
- `GetScreenerStocks(params ScreenerParams) ([]ScreenerStock, int, error)`
- Join `tickers` with latest `fundamentals`, `dividends`, `stock_prices`
- LEFT JOIN with `ic_scores` for IC Score data
- Support dynamic WHERE clauses based on filters
- Support dynamic ORDER BY

#### 1.3 Add Model
Add to `backend/models/stock.go`:
```go
type ScreenerStock struct {
    Symbol         string   `json:"symbol" db:"symbol"`
    Name           string   `json:"name" db:"name"`
    Sector         string   `json:"sector" db:"sector"`
    Industry       string   `json:"industry" db:"industry"`
    MarketCap      *float64 `json:"market_cap" db:"market_cap"`
    Price          *float64 `json:"price" db:"price"`
    ChangePercent  *float64 `json:"change_percent" db:"change_percent"`
    PERatio        *float64 `json:"pe_ratio" db:"pe_ratio"`
    PBRatio        *float64 `json:"pb_ratio" db:"pb_ratio"`
    PSRatio        *float64 `json:"ps_ratio" db:"ps_ratio"`
    ROE            *float64 `json:"roe" db:"roe"`
    RevenueGrowth  *float64 `json:"revenue_growth" db:"revenue_growth"`
    EarningsGrowth *float64 `json:"earnings_growth" db:"earnings_growth"`
    DividendYield  *float64 `json:"dividend_yield" db:"dividend_yield"`
    Beta           *float64 `json:"beta" db:"beta"`
    ICScore        *float64 `json:"ic_score" db:"ic_score"`
}
```

#### 1.4 Register Route
Update `backend/main.go`:
```go
// Screener endpoints
screener := v1.Group("/screener")
{
    screener.GET("/stocks", handlers.GetScreenerStocks)
}
```

### Phase 2: Frontend Cleanup

#### 2.1 Remove Mock Data
In `app/screener/page.tsx`:
- Delete `generateMockStocks()` function (lines 582-613)
- Remove mock data fallback in `fetchStocks()` catch block
- Show proper error state instead

#### 2.2 Update API Call
- Add query parameter support for server-side filtering
- Update pagination to use server-side pagination
- Add loading/error states

### Phase 3: Enhancements (Optional)

- Add caching layer for screener results (Redis)
- Add saved screens feature
- Add export to CSV functionality

## SQL Query Design

```sql
WITH latest_prices AS (
    SELECT DISTINCT ON (symbol)
        symbol, price, change_percent
    FROM stock_prices
    ORDER BY symbol, timestamp DESC
),
latest_fundamentals AS (
    SELECT DISTINCT ON (symbol)
        symbol, pe, pb, ps, roe,
        revenue, gross_margin, operating_margin
    FROM fundamentals
    ORDER BY symbol, year DESC, period DESC
),
latest_dividends AS (
    SELECT DISTINCT ON (symbol)
        symbol, yield_percent as dividend_yield
    FROM dividends
    ORDER BY symbol, ex_date DESC
),
latest_ic_scores AS (
    SELECT DISTINCT ON (ticker)
        ticker as symbol, overall_score as ic_score
    FROM ic_scores
    ORDER BY ticker, date DESC
)
SELECT
    t.symbol,
    t.name,
    COALESCE(t.sector, '') as sector,
    COALESCE(t.industry, '') as industry,
    t.market_cap,
    lp.price,
    lp.change_percent,
    lf.pe as pe_ratio,
    lf.pb as pb_ratio,
    lf.ps as ps_ratio,
    lf.roe,
    ld.dividend_yield,
    lic.ic_score
FROM tickers t
LEFT JOIN latest_prices lp ON t.symbol = lp.symbol
LEFT JOIN latest_fundamentals lf ON t.symbol = lf.symbol
LEFT JOIN latest_dividends ld ON t.symbol = ld.symbol
LEFT JOIN latest_ic_scores lic ON t.symbol = lic.symbol
WHERE t.asset_type = 'stock'
  AND t.active = true
  -- Dynamic filters added here
ORDER BY t.market_cap DESC NULLS LAST
LIMIT $1 OFFSET $2;
```

## Files to Modify

| File | Action |
|------|--------|
| `backend/handlers/screener.go` | **CREATE** - New handler |
| `backend/database/screener.go` | **CREATE** - New database queries |
| `backend/models/stock.go` | **MODIFY** - Add ScreenerStock struct |
| `backend/main.go` | **MODIFY** - Register new route |
| `app/screener/page.tsx` | **MODIFY** - Remove mock data, update API call |
| `lib/api.ts` | **MODIFY** - Add screener API method |

## Testing Plan

1. Unit tests for database query building
2. Integration test for `/api/v1/screener/stocks` endpoint
3. Test filter combinations
4. Test pagination edge cases
5. Frontend E2E test for screener page

## Estimated Effort

| Task | Estimate |
|------|----------|
| Backend handler + database | 2-3 hours |
| Frontend cleanup | 1 hour |
| Testing | 1 hour |
| **Total** | **4-5 hours** |

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Large dataset performance | Add LIMIT, use indexes, consider materialized view |
| Missing fundamental data | Use LEFT JOINs, handle NULLs gracefully |
| IC Score table in different schema | Same DB, should work with direct join |

---

## Data Backfill Status (Updated 2025-11-27)

### Data Gaps Addressed

#### 1. Sector/Industry Data ✅ FIXED
- **Problem**: Only 1 stock had sector data
- **Solution**: Created SIC-to-GICS sector mapping and backfilled from `sic_code` column
- **Result**: **3,474 stocks** (61.9%) now have sector/industry data
- **Remaining Gap**: 2,118 stocks missing SIC codes (mostly foreign ADRs)
- **Scripts**: [scripts/backfill_screener_data.py](../scripts/backfill_screener_data.py)

#### 2. Dividend Yield ⚠️ PARTIAL
- **Problem**: `dividends` table didn't exist
- **Solution**: Created table + fetch script from Polygon API
- **Result**: Table created, sample data inserted (6 tickers)
- **Next Step**: Run full backfill via CronJob
- **Scripts**: [scripts/fetch_dividends.py](../scripts/fetch_dividends.py)
- **CronJob**: [k8s/dividend-backfill-cronjob.yaml](../k8s/dividend-backfill-cronjob.yaml)

#### 3. Revenue Growth ✅ FIXED
- **Problem**: Only 28% coverage in `fundamental_metrics_extended`
- **Solution**: Recalculated YoY revenue growth from `financials` table
- **Result**: **2,379 stocks** now have revenue growth data (up from 1,552)

### Current Data Coverage

| Metric | Count | Coverage |
|--------|-------|----------|
| Total stocks | 5,615 | 100% |
| With sector | 3,474 | **61.9%** |
| With IC scores | 6,489 | 100%+ |
| With valuation ratios | 5,528 | 98.5% |
| With revenue growth | 2,379 | **42.4%** |
| With dividend data | 6 | 0.1% (needs backfill) |

### Recommended Next Steps

1. **Deploy dividend backfill CronJob** to populate dividend data for all stocks
2. **Fetch ticker details from Polygon** for remaining 2,118 stocks missing SIC codes
3. **Consider materialized view** for screener query performance

### Updated SQL Query

```sql
WITH latest_prices AS (
    SELECT DISTINCT ON (ticker)
        ticker, close as price
    FROM stock_prices
    ORDER BY ticker, time DESC
),
latest_valuation AS (
    SELECT DISTINCT ON (ticker)
        ticker, ttm_pe_ratio, ttm_pb_ratio, ttm_ps_ratio
    FROM valuation_ratios
    ORDER BY ticker, calculation_date DESC
),
latest_metrics AS (
    SELECT DISTINCT ON (ticker)
        ticker, revenue_growth_yoy, dividend_yield, roe, beta
    FROM fundamental_metrics_extended
    ORDER BY ticker, calculation_date DESC
),
latest_ic_scores AS (
    SELECT DISTINCT ON (ticker)
        ticker, overall_score as ic_score
    FROM ic_scores
    ORDER BY ticker, date DESC
)
SELECT
    t.symbol,
    t.name,
    COALESCE(t.sector, '') as sector,
    COALESCE(t.industry, '') as industry,
    t.market_cap,
    lp.price,
    lv.ttm_pe_ratio as pe_ratio,
    lv.ttm_pb_ratio as pb_ratio,
    lv.ttm_ps_ratio as ps_ratio,
    lm.roe,
    lm.revenue_growth_yoy as revenue_growth,
    lm.dividend_yield,
    lm.beta,
    lic.ic_score
FROM tickers t
LEFT JOIN latest_prices lp ON t.symbol = lp.ticker
LEFT JOIN latest_valuation lv ON t.symbol = lv.ticker
LEFT JOIN latest_metrics lm ON t.symbol = lm.ticker
LEFT JOIN latest_ic_scores lic ON t.symbol = lic.ticker
WHERE t.asset_type = 'stock'
  AND t.active = true
  -- Dynamic filters added here
ORDER BY t.market_cap DESC NULLS LAST
LIMIT $1 OFFSET $2;
```
