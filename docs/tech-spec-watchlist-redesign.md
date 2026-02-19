# Technical Specification: Watch Lists Feature Redesign

**Document Version:** 1.0
**Author:** Engineering
**Date:** February 18, 2026
**Status:** Draft — Pending Engineering Review
**Companion Document:** [PRD: Watch Lists Feature Redesign](./prd-watchlist-redesign.md)

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Phase 1: Diagnose & Fix Production Failures](#2-phase-1-diagnose--fix-production-failures)
3. [Phase 2: Backend — Enriched Data Layer](#3-phase-2-backend--enriched-data-layer)
4. [Phase 3: Backend — Batch Price Optimization](#4-phase-3-backend--batch-price-optimization)
5. [Phase 4: Backend — Premium Tier Enforcement](#5-phase-4-backend--premium-tier-enforcement)
6. [Phase 5: Frontend — Table Views & Sorting](#6-phase-5-frontend--table-views--sorting)
7. [Phase 6: Frontend — Polling, Error States, Mobile](#7-phase-6-frontend--polling-error-states-mobile)
8. [Phase 7: Cross-Feature Integration](#8-phase-7-cross-feature-integration)
9. [Database Migrations](#9-database-migrations)
10. [API Contract Changes](#10-api-contract-changes)
11. [Performance Budget](#11-performance-budget)
12. [Testing Strategy](#12-testing-strategy)
13. [Deployment Plan](#13-deployment-plan)
14. [Appendix: Current Implementation Reference](#appendix-current-implementation-reference)

---

## 1. System Overview

### Current Architecture

```
┌──────────────────┐      ┌─────────────────┐      ┌────────────────────┐
│   Next.js 14     │      │    Go / Gin      │      │   PostgreSQL       │
│   Frontend       │─────▶│    Backend       │─────▶│   (shared DB)      │
│   Port 3000      │ REST │    Port 8080     │      │                    │
│                  │      │                  │      │ watch_lists        │
│ app/watchlist/   │      │ handlers/        │      │ watch_list_items   │
│ components/      │      │   watchlist_     │      │ alert_rules        │
│   watchlist/     │      │   handlers.go    │      │ alert_trigger_logs │
│ lib/api/         │      │ services/        │      │ tickers (~25K)     │
│   watchlist.ts   │      │   watchlist_     │      │ screener_data (MV) │
│                  │      │   service.go     │      │ reddit_heatmap_    │
│                  │      │ services/        │      │   daily            │
│                  │      │   polygon.go ────┼──────│ stock_prices (TS)  │
│                  │      │                  │      │                    │
└──────────────────┘      └────────┬────────┘      └────────────────────┘
                                   │
                          ┌────────▼────────┐
                          │   Polygon.io    │
                          │   REST API      │
                          │ (real-time      │
                          │  stock quotes)  │
                          └─────────────────┘
```

### Data Flow for `GET /api/v1/watchlists/:id`

Current (broken in production, functional in code):

```
1. Handler: auth check → extract watchListID, userID
2. Service: GetWatchListWithItems(watchListID, userID)
   a. database.GetWatchListByID(watchListID, userID)          → watch_lists row
   b. database.GetWatchListItemsWithData(watchListID)          → JOIN: watch_list_items + tickers + reddit_heatmap_daily
   c. FOR EACH item: polygonClient.GetQuote(item.Symbol)       → sequential Polygon API calls (N+1 problem)
   d. Enrich each item with: CurrentPrice, PriceChange, PriceChangePct, Volume, PrevClose
3. Handler: return JSON { watchlist metadata + enriched items[] }
```

Target state after this spec:

```
1. Handler: auth check → extract watchListID, userID
2. Service: GetWatchListWithItems(watchListID, userID)
   a. database.GetWatchListByID(watchListID, userID)                          → watch_lists row
   b. database.GetWatchListItemsWithEnrichedData(watchListID)                 → JOIN: watch_list_items + tickers + reddit_heatmap_daily + screener_data
   c. polygonClient.GetBatchSnapshots(allSymbols)                             → SINGLE batch Polygon API call
   d. Merge batch prices into items
   e. If Polygon fails: return items WITHOUT prices (graceful degradation)    → NEW behavior
3. Handler: return JSON { watchlist metadata + enriched items[] + summary metrics }
```

---

## 2. Phase 1: Diagnose & Fix Production Failures

**Priority:** P0 — Block all other work until resolved
**Estimated effort:** 2-3 days
**Owner:** Backend engineer

### 2.1 Root Cause Investigation

The watchlist feature returns "Failed to fetch watch lists" in production. The code is structurally sound — the likely causes are infrastructure/deployment issues.

**Investigation runbook (execute in order):**

#### Step 1: Verify database tables exist

```bash
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  psql $DATABASE_URL -c "
    SELECT table_name FROM information_schema.tables
    WHERE table_schema = 'public'
    AND table_name IN ('watch_lists', 'watch_list_items', 'alert_rules', 'alert_trigger_logs');
  "
```

**Expected output:** 4 rows. If 0 rows: migration 010 and 012 were never applied.

**Fix if missing:**
```bash
# Apply watchlist migration
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  psql $DATABASE_URL -f /app/migrations/010_watchlist_tables.sql

# Apply alert migration
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  psql $DATABASE_URL -f /app/migrations/012_alert_system.sql
```

#### Step 2: Verify trigger functions exist

```bash
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  psql $DATABASE_URL -c "
    SELECT routine_name FROM information_schema.routines
    WHERE routine_schema = 'public'
    AND routine_name IN (
      'update_watch_lists_updated_at',
      'check_watch_list_item_limit',
      'create_default_watch_list',
      'update_alert_rules_updated_at'
    );
  "
```

**Expected output:** 4 rows.

#### Step 3: Verify tickers table exists (not stocks)

```bash
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- \
  psql $DATABASE_URL -c "
    SELECT table_name FROM information_schema.tables
    WHERE table_name IN ('tickers', 'stocks');
  "
```

**Expected:** `tickers` present, `stocks` absent (renamed in migration 003).

#### Step 4: Test API endpoint directly

```bash
# Get a JWT token
TOKEN=$(curl -s -X POST https://investorcenter.ai/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"..."}' | jq -r '.access_token')

# Test watchlist list endpoint
curl -v -H "Authorization: Bearer $TOKEN" \
  https://investorcenter.ai/api/v1/watchlists

# Test watchlist create endpoint
curl -v -X POST -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Test List"}' \
  https://investorcenter.ai/api/v1/watchlists
```

**Examine:** HTTP status code, response body, error message. Check for:
- 500 → database connection or query failure (check pod logs)
- 401 → auth middleware issue
- 404 → route not registered (check `main.go` initialization)
- CORS error in browser → missing CORS headers for `/watchlists` path

#### Step 5: Check pod logs

```bash
kubectl logs -n investorcenter deploy/investorcenter-backend --tail=100 | grep -i "watchlist\|watch_list\|error\|panic"
```

#### Step 6: Verify service initialization

In `backend/main.go`, the `watchListService` is initialized at package level:

```go
var watchListService = services.NewWatchListService()
```

If `NewPolygonClient()` (called inside `GetWatchListWithItems`) fails due to missing `POLYGON_API_KEY` env var, the entire request will fail. Check:

```bash
kubectl exec -it deploy/investorcenter-backend -n investorcenter -- env | grep POLYGON
```

### 2.2 Fix Test Suite Table Name Bug

**File:** `backend/handlers/watchlist_handlers_test.go`

The test file references `INSERT INTO stocks` but the table was renamed to `tickers` in migration 003. This causes test failures and blocks CI.

**Changes required:**

```go
// BEFORE (broken):
_, err = db.Exec(`INSERT INTO stocks (symbol, name, ...) VALUES ($1, $2, ...)`, ...)

// AFTER (fixed):
_, err = db.Exec(`INSERT INTO tickers (symbol, name, ...) VALUES ($1, $2, ...)`, ...)
```

**Locations to update:** All `INSERT INTO stocks` references in the test file. Search with:

```bash
grep -n "INSERT INTO stocks" backend/handlers/watchlist_handlers_test.go
```

### 2.3 Graceful Degradation for Polygon Failures

Currently, if `polygonClient.GetQuote()` fails for any ticker, the error is silently swallowed and the item is returned without price data. This is already the correct behavior — but we should make it explicit and add logging.

**Change in `backend/services/watchlist_service.go`:**

```go
// Current: silent swallow
price, err := polygonClient.GetQuote(item.Symbol)
if err == nil && price != nil {
    // ... enrich item
}

// Target: explicit logging + partial failure tracking
price, err := polygonClient.GetQuote(item.Symbol)
if err != nil {
    log.Printf("Warning: Failed to fetch price for %s: %v", item.Symbol, err)
    priceFailures++
    continue
}
if price != nil {
    // ... enrich item
}

// After loop:
if priceFailures > 0 {
    log.Printf("Price fetch: %d/%d tickers failed", priceFailures, len(items))
}
```

This is a no-op for users (behavior is identical) but gives us observability into Polygon failures.

---

## 3. Phase 2: Backend — Enriched Data Layer

**Priority:** P1 — Foundation for column views
**Estimated effort:** 3-4 days
**Owner:** Backend engineer
**Depends on:** Phase 1 complete

### 3.1 Extend `GetWatchListItemsWithData` Query

The current query joins `watch_list_items` → `tickers` → `reddit_heatmap_daily`. We need to add a LEFT JOIN to `screener_data` to pull IC Score, fundamentals, valuation ratios, and dividend data.

**File:** `backend/database/watchlists.go`

**New function: `GetWatchListItemsWithEnrichedData`**

We create a new function rather than modifying the existing one to avoid breaking the current API contract until the frontend is ready.

```go
func GetWatchListItemsWithEnrichedData(watchListID string) ([]models.WatchListItemEnriched, error) {
    query := `
        SELECT
            -- watch_list_items (9 cols)
            wli.id, wli.watch_list_id, wli.symbol, wli.notes, wli.tags,
            wli.target_buy_price, wli.target_sell_price, wli.added_at, wli.display_order,

            -- tickers (4 cols)
            t.name, t.exchange, t.asset_type, t.logo_url,

            -- reddit_heatmap_daily via LATERAL (5 cols)
            rhd.avg_rank, rhd.total_mentions, rhd.popularity_score,
            rhd.trend_direction, (rhd.avg_rank - rhd.rank_24h_ago),

            -- screener_data (28 cols)
            sd.ic_score, sd.ic_rating,
            sd.value_score, sd.growth_score, sd.profitability_score,
            sd.financial_health_score, sd.momentum_score,
            sd.analyst_consensus_score, sd.insider_activity_score,
            sd.institutional_score, sd.news_sentiment_score, sd.technical_score,
            sd.ic_sector_percentile, sd.lifecycle_stage,
            sd.pe_ratio, sd.pb_ratio, sd.ps_ratio,
            sd.roe, sd.roa, sd.gross_margin, sd.operating_margin, sd.net_margin,
            sd.debt_to_equity, sd.current_ratio,
            sd.revenue_growth, sd.eps_growth_yoy,
            sd.dividend_yield, sd.payout_ratio,

            -- alert count (1 col)
            COALESCE(ac.alert_count, 0)
        FROM watch_list_items wli
        JOIN tickers t ON wli.symbol = t.symbol
        LEFT JOIN LATERAL (
            SELECT avg_rank, total_mentions, popularity_score,
                   trend_direction, rank_24h_ago
            FROM reddit_heatmap_daily
            WHERE ticker_symbol = wli.symbol
            ORDER BY date DESC
            LIMIT 1
        ) rhd ON true
        LEFT JOIN screener_data sd ON wli.symbol = sd.symbol
        LEFT JOIN LATERAL (
            SELECT COUNT(*) as alert_count
            FROM alert_rules
            WHERE watch_list_id = wli.watch_list_id
              AND symbol = wli.symbol
              AND is_active = true
        ) ac ON true
        WHERE wli.watch_list_id = $1
        ORDER BY wli.display_order ASC, wli.added_at DESC
    `
    // ... scan implementation
}
```

**JOIN breakdown:**

| JOIN | Source | Type | Indexed? | Expected Performance |
|------|--------|------|----------|---------------------|
| `tickers t` | INNER JOIN | `tickers.symbol` | PK (unique index) | O(1) lookup per item |
| `reddit_heatmap_daily` | LEFT JOIN LATERAL | `idx_reddit_heatmap_ticker` on (ticker_symbol, date DESC) | Yes | O(1) per item via index + LIMIT 1 |
| `screener_data sd` | LEFT JOIN | `idx_screener_data_symbol` (unique) | Yes | O(1) per item |
| `alert_rules` | LEFT JOIN LATERAL (COUNT) | `idx_alert_rules_watch_list_id` + `idx_alert_rules_symbol` | Yes | O(small N) per item |

**Total query columns:** 9 + 4 + 5 + 28 + 1 = **47 columns**

For a 100-item watchlist, this is a single SQL query with 4 JOINs, all index-backed. Expected execution time: <50ms.

### 3.2 New Go Model: `WatchListItemEnriched`

**File:** `backend/models/watchlist.go`

```go
type WatchListItemEnriched struct {
    WatchListItem

    // Ticker metadata (from tickers table)
    Name      string  `json:"name"`
    Exchange  string  `json:"exchange"`
    AssetType string  `json:"asset_type"`
    LogoURL   *string `json:"logo_url"`

    // Real-time prices (populated by service layer, not DB)
    CurrentPrice   *float64 `json:"current_price"`
    PriceChange    *float64 `json:"price_change"`
    PriceChangePct *float64 `json:"price_change_pct"`
    Volume         *int64   `json:"volume"`
    MarketCap      *float64 `json:"market_cap"`
    PrevClose      *float64 `json:"prev_close"`

    // Reddit sentiment (from reddit_heatmap_daily)
    RedditRank       *int     `json:"reddit_rank,omitempty"`
    RedditMentions   *int     `json:"reddit_mentions,omitempty"`
    RedditPopularity *float64 `json:"reddit_popularity,omitempty"`
    RedditTrend      *string  `json:"reddit_trend,omitempty"`
    RedditRankChange *int     `json:"reddit_rank_change,omitempty"`

    // IC Score (from screener_data materialized view)
    ICScore               *float64 `json:"ic_score,omitempty"`
    ICRating              *string  `json:"ic_rating,omitempty"`
    ValueScore            *float64 `json:"value_score,omitempty"`
    GrowthScore           *float64 `json:"growth_score,omitempty"`
    ProfitabilityScore    *float64 `json:"profitability_score,omitempty"`
    FinancialHealthScore  *float64 `json:"financial_health_score,omitempty"`
    MomentumScore         *float64 `json:"momentum_score,omitempty"`
    AnalystConsensusScore *float64 `json:"analyst_consensus_score,omitempty"`
    InsiderActivityScore  *float64 `json:"insider_activity_score,omitempty"`
    InstitutionalScore    *float64 `json:"institutional_score,omitempty"`
    NewsSentimentScore    *float64 `json:"news_sentiment_score,omitempty"`
    TechnicalScore        *float64 `json:"technical_score,omitempty"`
    SectorPercentile      *float64 `json:"sector_percentile,omitempty"`
    LifecycleStage        *string  `json:"lifecycle_stage,omitempty"`

    // Fundamentals (from screener_data)
    PERatio        *float64 `json:"pe_ratio,omitempty"`
    PBRatio        *float64 `json:"pb_ratio,omitempty"`
    PSRatio        *float64 `json:"ps_ratio,omitempty"`
    ROE            *float64 `json:"roe,omitempty"`
    ROA            *float64 `json:"roa,omitempty"`
    GrossMargin    *float64 `json:"gross_margin,omitempty"`
    OperatingMargin *float64 `json:"operating_margin,omitempty"`
    NetMargin      *float64 `json:"net_margin,omitempty"`
    DebtToEquity   *float64 `json:"debt_to_equity,omitempty"`
    CurrentRatio   *float64 `json:"current_ratio,omitempty"`
    RevenueGrowth  *float64 `json:"revenue_growth,omitempty"`
    EPSGrowth      *float64 `json:"eps_growth,omitempty"`
    DividendYield  *float64 `json:"dividend_yield,omitempty"`
    PayoutRatio    *float64 `json:"payout_ratio,omitempty"`

    // Alert metadata
    AlertCount int `json:"alert_count"`
}
```

### 3.3 Updated Response Model

**File:** `backend/models/watchlist.go`

```go
type WatchListWithItemsEnriched struct {
    WatchList
    ItemCount int                      `json:"item_count"`
    Items     []WatchListItemEnriched  `json:"items"`
    Summary   *WatchListSummaryMetrics `json:"summary,omitempty"`
}

type WatchListSummaryMetrics struct {
    TotalTickers       int      `json:"total_tickers"`
    AvgICScore         *float64 `json:"avg_ic_score,omitempty"`
    AvgDayChangePct    *float64 `json:"avg_day_change_pct,omitempty"`
    AvgDividendYield   *float64 `json:"avg_dividend_yield,omitempty"`
    MedianMarketCap    *float64 `json:"median_market_cap,omitempty"`
    RedditTrendingCount int     `json:"reddit_trending_count"`
}
```

The `Summary` field is computed in the service layer after items are enriched with prices. It is always returned (even for free-tier users).

### 3.4 Add Performance Columns to `screener_data`

The Performance view (1W%, 1M%, 3M%, 6M%, YTD%, 1Y%) needs historical price comparisons. Rather than per-ticker API calls, we pre-compute these in the `screener_data` materialized view.

**New migration:** `ic-score-service/migrations/XXX_add_performance_to_screener.sql`

```sql
-- Drop and recreate the materialized view with performance columns
DROP MATERIALIZED VIEW IF EXISTS screener_data;

CREATE MATERIALIZED VIEW screener_data AS
SELECT
    -- [all existing columns from migration 019] ...

    -- Performance: computed from stock_prices historical data
    CASE WHEN lp.price > 0 AND lp_1w.price > 0
         THEN ((lp.price - lp_1w.price) / lp_1w.price * 100)
         ELSE NULL END AS perf_1w,
    CASE WHEN lp.price > 0 AND lp_1m.price > 0
         THEN ((lp.price - lp_1m.price) / lp_1m.price * 100)
         ELSE NULL END AS perf_1m,
    CASE WHEN lp.price > 0 AND lp_3m.price > 0
         THEN ((lp.price - lp_3m.price) / lp_3m.price * 100)
         ELSE NULL END AS perf_3m,
    CASE WHEN lp.price > 0 AND lp_6m.price > 0
         THEN ((lp.price - lp_6m.price) / lp_6m.price * 100)
         ELSE NULL END AS perf_6m,
    CASE WHEN lp.price > 0 AND lp_ytd.price > 0
         THEN ((lp.price - lp_ytd.price) / lp_ytd.price * 100)
         ELSE NULL END AS perf_ytd,
    CASE WHEN lp.price > 0 AND lp_1y.price > 0
         THEN ((lp.price - lp_1y.price) / lp_1y.price * 100)
         ELSE NULL END AS perf_1y,

    CURRENT_TIMESTAMP as refreshed_at

FROM tickers t
-- [existing LATERAL joins: lp, lv, lm, lic] ...

-- Performance lookback prices
LEFT JOIN LATERAL (
    SELECT close AS price FROM stock_prices
    WHERE ticker = t.symbol AND time <= (CURRENT_DATE - INTERVAL '7 days')
    ORDER BY time DESC LIMIT 1
) lp_1w ON true
LEFT JOIN LATERAL (
    SELECT close AS price FROM stock_prices
    WHERE ticker = t.symbol AND time <= (CURRENT_DATE - INTERVAL '1 month')
    ORDER BY time DESC LIMIT 1
) lp_1m ON true
LEFT JOIN LATERAL (
    SELECT close AS price FROM stock_prices
    WHERE ticker = t.symbol AND time <= (CURRENT_DATE - INTERVAL '3 months')
    ORDER BY time DESC LIMIT 1
) lp_3m ON true
LEFT JOIN LATERAL (
    SELECT close AS price FROM stock_prices
    WHERE ticker = t.symbol AND time <= (CURRENT_DATE - INTERVAL '6 months')
    ORDER BY time DESC LIMIT 1
) lp_6m ON true
LEFT JOIN LATERAL (
    SELECT close AS price FROM stock_prices
    WHERE ticker = t.symbol AND time <= (DATE_TRUNC('year', CURRENT_DATE))
    ORDER BY time DESC LIMIT 1
) lp_ytd ON true
LEFT JOIN LATERAL (
    SELECT close AS price FROM stock_prices
    WHERE ticker = t.symbol AND time <= (CURRENT_DATE - INTERVAL '1 year')
    ORDER BY time DESC LIMIT 1
) lp_1y ON true

WHERE t.asset_type = 'CS' AND t.active = true;

-- Recreate all indexes (required after DROP)
CREATE UNIQUE INDEX idx_screener_data_symbol ON screener_data(symbol);
-- [all other existing indexes] ...

-- New performance indexes
CREATE INDEX idx_screener_data_perf_1w ON screener_data(perf_1w DESC NULLS LAST);
CREATE INDEX idx_screener_data_perf_1m ON screener_data(perf_1m DESC NULLS LAST);
CREATE INDEX idx_screener_data_perf_ytd ON screener_data(perf_ytd DESC NULLS LAST);
```

**Trade-off acknowledged:** Adding 6 more LATERAL subqueries to an already complex view will increase refresh time. The `stock_prices` table is a TimescaleDB hypertable with time-based chunking — these lookback queries are efficient because TimescaleDB indexes by time. Current refresh takes ~30 seconds; expect ~60-90 seconds with the addition. This runs once daily at 23:45 UTC and uses `REFRESH MATERIALIZED VIEW CONCURRENTLY`, so no read blocking.

**Impact on screener:** The screener handler reads from `screener_data`. Adding columns doesn't affect existing queries — they select by name, not `*`. No screener code changes needed.

---

## 4. Phase 3: Backend — Batch Price Optimization

**Priority:** P1 — Critical for performance at scale
**Estimated effort:** 2 days
**Owner:** Backend engineer
**Depends on:** Phase 1 complete

### 4.1 Problem

`GetWatchListWithItems` calls `polygonClient.GetQuote()` sequentially for each ticker:

```go
for i := range items {
    price, err := polygonClient.GetQuote(item.Symbol)  // N sequential HTTP calls
}
```

For a 50-ticker watchlist, this means 50 serial HTTP round-trips to Polygon.io. At ~100ms each, that's **5 seconds** of latency added to the API response — completely unacceptable.

### 4.2 Solution: Batch Snapshot API

Polygon already provides bulk snapshot endpoints that return all requested tickers in a single response. The Go client already has `GetBulkStockSnapshots()` and `GetBulkCryptoSnapshots()` — but these fetch **all** tickers, which is wasteful.

**New method in `backend/services/polygon.go`:**

```go
// GetBatchQuotes fetches real-time prices for a list of symbols in minimal API calls.
// Stocks: Uses Polygon Snapshot filtered by tickers param.
// Crypto: Iterates cached Redis values (already O(1) per symbol).
func (p *PolygonClient) GetBatchQuotes(symbols []string) (map[string]*models.StockPrice, error) {
    results := make(map[string]*models.StockPrice)

    var stockSymbols, cryptoSymbols []string
    for _, s := range symbols {
        if strings.HasPrefix(s, "X:") {
            cryptoSymbols = append(cryptoSymbols, s)
        } else {
            stockSymbols = append(stockSymbols, s)
        }
    }

    // Batch fetch stocks via snapshot API
    // Polygon supports: GET /v2/snapshot/locale/us/markets/stocks/tickers?tickers=AAPL,MSFT,GOOGL
    // Max ~50 tickers per request (URL length limit)
    if len(stockSymbols) > 0 {
        for _, batch := range chunkSlice(stockSymbols, 50) {
            tickerParam := strings.Join(batch, ",")
            url := fmt.Sprintf("%s/v2/snapshot/locale/us/markets/stocks/tickers?tickers=%s&apikey=%s",
                PolygonBaseURL, tickerParam, p.APIKey)

            resp, err := p.Client.Get(url)
            if err != nil {
                log.Printf("Batch stock snapshot failed: %v", err)
                continue // graceful degradation: skip failed batch
            }
            defer resp.Body.Close()

            var snapshotResp BulkStockSnapshotResponse
            json.NewDecoder(resp.Body).Decode(&snapshotResp)

            for _, ticker := range snapshotResp.Tickers {
                results[ticker.Ticker] = convertSnapshotToStockPrice(ticker)
            }
        }
    }

    // Fetch crypto from cache (already fast, no batch API needed)
    cryptoCache := GetCryptoCache()
    for _, symbol := range cryptoSymbols {
        if cachedPrice, exists := cryptoCache.GetPrice(symbol); exists {
            results[symbol] = cachedPrice
        } else {
            // Fallback to individual API call
            price, err := p.GetCryptoRealTimePrice(symbol)
            if err == nil {
                results[symbol] = price
            }
        }
    }

    return results, nil
}

func chunkSlice(slice []string, chunkSize int) [][]string {
    var chunks [][]string
    for i := 0; i < len(slice); i += chunkSize {
        end := i + chunkSize
        if end > len(slice) {
            end = len(slice)
        }
        chunks = append(chunks, slice[i:end])
    }
    return chunks
}
```

### 4.3 Updated Service Method

**File:** `backend/services/watchlist_service.go`

```go
func (s *WatchListService) GetWatchListWithItems(watchListID string, userID string) (*models.WatchListWithItemsEnriched, error) {
    watchList, err := database.GetWatchListByID(watchListID, userID)
    if err != nil {
        return nil, err
    }

    items, err := database.GetWatchListItemsWithEnrichedData(watchListID)
    if err != nil {
        return nil, err
    }

    // Collect all symbols for batch price fetch
    symbols := make([]string, len(items))
    for i, item := range items {
        symbols[i] = item.Symbol
    }

    // SINGLE batch call replaces N sequential calls
    polygonClient := NewPolygonClient()
    prices, err := polygonClient.GetBatchQuotes(symbols)
    if err != nil {
        log.Printf("Warning: Batch price fetch failed: %v", err)
        // Continue without prices — graceful degradation
    }

    // Merge prices into items
    priceHits, priceMisses := 0, 0
    for i := range items {
        if price, ok := prices[items[i].Symbol]; ok && price != nil {
            p := price.Price.InexactFloat64()
            items[i].CurrentPrice = &p
            c := price.Change.InexactFloat64()
            items[i].PriceChange = &c
            cp := price.ChangePercent.InexactFloat64()
            items[i].PriceChangePct = &cp
            if price.Volume > 0 {
                v := int64(price.Volume)
                items[i].Volume = &v
            }
            if price.Change.IsPositive() || price.Change.IsNegative() {
                pc := price.Price.Sub(price.Change).InexactFloat64()
                items[i].PrevClose = &pc
            }
            priceHits++
        } else {
            priceMisses++
        }
    }

    if priceMisses > 0 {
        log.Printf("Batch price: %d hits, %d misses for watchlist %s", priceHits, priceMisses, watchListID)
    }

    // Compute summary metrics
    summary := computeSummaryMetrics(items)

    return &models.WatchListWithItemsEnriched{
        WatchList: *watchList,
        ItemCount: len(items),
        Items:     items,
        Summary:   summary,
    }, nil
}
```

**Performance improvement:**
- Before: 50 tickers × ~100ms/call = **~5,000ms**
- After: 1 batch call (~200ms) + DB query (~50ms) = **~250ms**
- **20x improvement**

---

## 5. Phase 4: Backend — Premium Tier Enforcement

**Priority:** P1 — Required before public launch
**Estimated effort:** 2 days
**Owner:** Backend engineer

### 5.1 Current Tier Enforcement

Tier limits are currently enforced in two places:

1. **Database trigger** (`check_watch_list_item_limit`): Checks `users.is_premium` and enforces 10-item limit per watchlist at INSERT time.
2. **Alert handler** (`CreateAlert`): Calls `database.CountAlertRulesByUserID()` and checks against tier limit.

**Subscription plans** (from migration 012):

| Plan | Max Watchlists | Max Items/List | Max Alerts | Max Heatmap Configs |
|------|---------------|----------------|------------|---------------------|
| Free | 3 | 10 | 10 | 3 |
| Premium | 20 | 100 | 100 | 20 |
| Enterprise | Unlimited | Unlimited | Unlimited | Unlimited |

### 5.2 Missing Enforcement: Watchlist Count Limit

The free tier allows 3 watchlists, but `CreateWatchList` handler doesn't check this. Add enforcement:

**File:** `backend/handlers/watchlist_handlers.go` — `CreateWatchList` handler

```go
func CreateWatchList(c *gin.Context) {
    userID, _ := auth.GetUserIDFromContext(c)

    // NEW: Check watchlist count limit
    existingLists, err := database.GetWatchListsByUserID(userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check watchlist count"})
        return
    }

    userPlan, err := database.GetUserSubscriptionPlan(userID)
    if err != nil {
        // Default to free tier limits if no subscription found
        userPlan = &models.SubscriptionPlan{MaxWatchLists: 3}
    }

    if userPlan.MaxWatchLists > 0 && len(existingLists) >= userPlan.MaxWatchLists {
        c.JSON(http.StatusForbidden, gin.H{
            "error":   "Watchlist limit reached",
            "message": fmt.Sprintf("Free accounts can create up to %d watchlists. Upgrade to create more.", userPlan.MaxWatchLists),
            "limit":   userPlan.MaxWatchLists,
            "current": len(existingLists),
        })
        return
    }

    // ... existing create logic
}
```

### 5.3 Column-Level Gating (API Response Filtering)

Premium columns should be nulled out in API responses for free users. This is enforced **server-side** — the frontend renders whatever it receives.

**New middleware function in `backend/services/watchlist_service.go`:**

```go
func (s *WatchListService) applyTierGating(items []models.WatchListItemEnriched, isPremium bool) {
    if isPremium {
        return // Premium users get everything
    }

    for i := range items {
        // Gate IC Score sub-factors
        items[i].ValueScore = nil
        items[i].GrowthScore = nil
        items[i].ProfitabilityScore = nil
        items[i].FinancialHealthScore = nil
        items[i].MomentumScore = nil
        items[i].AnalystConsensusScore = nil
        items[i].InsiderActivityScore = nil
        items[i].InstitutionalScore = nil
        items[i].NewsSentimentScore = nil
        items[i].TechnicalScore = nil
        items[i].SectorPercentile = nil
        items[i].LifecycleStage = nil

        // Gate extended performance (3M+)
        // perf_3m, perf_6m, perf_ytd, perf_1y — set to nil
        // (These fields will be added when performance columns are integrated)

        // IC Score overall: return as "gated" marker, not nil
        // This allows the frontend to show the blurred teaser
        // The actual score is NOT sent — just a flag
    }
}
```

**Design decision:** IC Score overall (`ic_score`) is **not** gated. Free users see the number. Sub-factor scores are gated. This matches the PRD's "blurred teaser" strategy — showing the score creates curiosity, gating the breakdown creates upgrade intent.

**Alternative considered:** Sending a `"gated": true` flag instead of `nil`. Rejected — nulling fields is simpler, and the frontend already handles null/undefined gracefully. The frontend can check `isPremium` from the auth context to decide whether to show a blurred overlay vs. real data.

### 5.4 View-Level Gating

The PRD specifies that certain preset views (Fundamentals, Dividends, IC Score, Performance) are premium-only. This is enforced **client-side only** — the backend returns all data that the user's tier permits, and the frontend controls which views are accessible.

Rationale: Server-side view gating would require the client to pass a `?view=performance` parameter, adding complexity. Since column gating already ensures premium data is nulled, the frontend just needs to show an upgrade prompt when switching to a gated view.

---

## 6. Phase 5: Frontend — Table Views & Sorting

**Priority:** P1 — Core UX improvement
**Estimated effort:** 5-7 days
**Owner:** Frontend engineer
**Depends on:** Phase 2 (enriched API response)

### 6.1 Column Definition System

Create a centralized column registry that drives all table rendering. This avoids hardcoding column configurations across multiple components.

**New file:** `lib/watchlist/columns.ts`

```typescript
export interface ColumnDefinition {
  id: string;
  label: string;
  shortLabel?: string;      // For mobile / compact display
  type: 'string' | 'currency' | 'percent' | 'number' | 'badge' | 'image' | 'date' | 'range';
  align: 'left' | 'center' | 'right';
  sortable: boolean;
  premium: boolean;          // If true, show blurred/locked for free users
  width?: number;            // Default column width in px
  format?: (value: any, item: WatchListItem) => string;  // Custom formatter
  sortKey?: string;          // JSON field name for sorting (defaults to id)
}

export type ViewPreset = {
  id: string;
  label: string;
  columns: string[];         // Column IDs in display order
  premium: boolean;          // If true, view is premium-only
};

export const COLUMN_REGISTRY: Record<string, ColumnDefinition> = {
  symbol: { id: 'symbol', label: 'Symbol', type: 'string', align: 'left', sortable: true, premium: false },
  name: { id: 'name', label: 'Name', type: 'string', align: 'left', sortable: true, premium: false },
  current_price: { id: 'current_price', label: 'Price', type: 'currency', align: 'right', sortable: true, premium: false },
  price_change_pct: { id: 'price_change_pct', label: 'Change %', type: 'percent', align: 'right', sortable: true, premium: false },
  // ... all 50+ columns defined here
  ic_score: { id: 'ic_score', label: 'IC Score', type: 'number', align: 'center', sortable: true, premium: false },
  value_score: { id: 'value_score', label: 'Value', type: 'number', align: 'center', sortable: true, premium: true },
  // ... etc.
};

export const VIEW_PRESETS: ViewPreset[] = [
  {
    id: 'general',
    label: 'General',
    columns: ['symbol', 'name', 'current_price', 'price_change', 'price_change_pct', 'volume', 'market_cap', 'ic_score'],
    premium: false,
  },
  {
    id: 'performance',
    label: 'Performance',
    columns: ['symbol', 'name', 'current_price', 'perf_1d', 'perf_1w', 'perf_1m', 'perf_3m', 'perf_6m', 'perf_ytd', 'perf_1y'],
    premium: true,
  },
  {
    id: 'fundamentals',
    label: 'Fundamentals',
    columns: ['symbol', 'name', 'current_price', 'pe_ratio', 'pb_ratio', 'ps_ratio', 'roe', 'roa', 'debt_to_equity', 'current_ratio', 'revenue_growth'],
    premium: true,
  },
  {
    id: 'dividends',
    label: 'Dividends',
    columns: ['symbol', 'name', 'current_price', 'dividend_yield', 'payout_ratio'],
    premium: true,
  },
  {
    id: 'ic_score',
    label: 'IC Score',
    columns: ['symbol', 'name', 'current_price', 'ic_score', 'ic_rating', 'value_score', 'growth_score', 'profitability_score', 'financial_health_score', 'momentum_score'],
    premium: true,
  },
  {
    id: 'social',
    label: 'Social',
    columns: ['symbol', 'name', 'current_price', 'price_change_pct', 'reddit_rank', 'reddit_mentions', 'reddit_popularity', 'reddit_trend', 'reddit_rank_change'],
    premium: false,
  },
  {
    id: 'compact',
    label: 'Compact',
    columns: ['symbol', 'current_price', 'price_change_pct', 'ic_score', 'market_cap'],
    premium: false,
  },
];
```

### 6.2 Table Component Rewrite

Replace the current `WatchListTable.tsx` with a data-driven table that renders any column configuration.

**File:** `components/watchlist/WatchListTable.tsx`

Key changes:
- Accept `activeView: ViewPreset` prop instead of hardcoded columns
- Use `COLUMN_REGISTRY` to resolve column definitions
- Render column headers from definition (label, sortable indicator)
- Render cells using type-based formatters (currency, percent, badge, etc.)
- Client-side sorting with `useMemo` for stable sort:

```typescript
const sortedItems = useMemo(() => {
  if (!sortConfig) return items;

  return [...items].sort((a, b) => {
    const aVal = a[sortConfig.key];
    const bVal = b[sortConfig.key];
    if (aVal == null && bVal == null) return 0;
    if (aVal == null) return 1;   // nulls last
    if (bVal == null) return -1;
    const cmp = aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
    return sortConfig.direction === 'asc' ? cmp : -cmp;
  });
}, [items, sortConfig]);
```

### 6.3 View Switcher Component

**New file:** `components/watchlist/ViewSwitcher.tsx`

Dropdown selector that:
1. Lists all `VIEW_PRESETS`
2. Shows lock icon on premium-only views for free users
3. On select: updates `activeView` state, persists to `localStorage`
4. For premium-gated views on free accounts: shows upgrade overlay instead of data

### 6.4 Sorting State

```typescript
type SortConfig = {
  key: string;           // Column ID
  direction: 'asc' | 'desc';
} | null;

// Sort cycling: null → asc → desc → null
function cycleSortDirection(current: SortConfig, columnId: string): SortConfig {
  if (!current || current.key !== columnId) return { key: columnId, direction: 'asc' };
  if (current.direction === 'asc') return { key: columnId, direction: 'desc' };
  return null;
}
```

### 6.5 Filter Implementation

Client-side filtering — no API changes needed.

```typescript
type FilterState = {
  search: string;            // Text filter on symbol + name
  assetType: 'all' | 'CS' | 'ETF' | 'crypto';  // Asset type filter
};

const filteredItems = useMemo(() => {
  let result = items;

  if (filter.search) {
    const q = filter.search.toUpperCase();
    result = result.filter(item =>
      item.symbol.toUpperCase().includes(q) ||
      item.name.toUpperCase().includes(q)
    );
  }

  if (filter.assetType !== 'all') {
    if (filter.assetType === 'crypto') {
      result = result.filter(item => item.symbol.startsWith('X:'));
    } else {
      result = result.filter(item => item.asset_type === filter.assetType);
    }
  }

  return result;
}, [items, filter]);
```

---

## 7. Phase 6: Frontend — Polling, Error States, Mobile

**Priority:** P1
**Estimated effort:** 5-6 days
**Owner:** Frontend engineer

### 7.1 Adopt React Query (TanStack Query)

Replace the current `useEffect` + `setInterval` polling pattern with React Query. This provides:
- Stale-while-revalidate (show cached data immediately, refresh in background)
- Automatic retry with backoff
- Window focus refetching
- Deduplication (multiple components requesting same data = 1 API call)
- Devtools for debugging

**Install:**
```bash
npm install @tanstack/react-query
```

**Query Provider (wrap app):**

```typescript
// app/providers.tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 15_000,           // 15s before data is considered stale
      retry: 3,
      retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 10000),
    },
  },
});
```

**Watchlist detail query hook:**

```typescript
// lib/hooks/useWatchList.ts
import { useQuery } from '@tanstack/react-query';

export function useWatchList(watchListId: string) {
  return useQuery({
    queryKey: ['watchlist', watchListId],
    queryFn: () => watchListAPI.getWatchList(watchListId),
    refetchInterval: (query) => {
      // Adaptive polling based on market state
      const hasStocks = query.data?.items?.some(i => !i.symbol.startsWith('X:'));
      const hasCrypto = query.data?.items?.some(i => i.symbol.startsWith('X:'));
      if (isMarketOpen()) return hasStocks ? 15_000 : hasCrypto ? 5_000 : 60_000;
      return hasCrypto ? 15_000 : 60_000;
    },
    refetchIntervalInBackground: false,  // Pause when tab not focused
  });
}

export function useWatchLists() {
  return useQuery({
    queryKey: ['watchlists'],
    queryFn: () => watchListAPI.getWatchLists(),
    staleTime: 30_000,
  });
}
```

### 7.2 Error State Components

**New file:** `components/watchlist/ErrorState.tsx`

```typescript
type ErrorType = 'network' | 'auth' | 'server' | 'rate_limit' | 'not_found';

function classifyError(error: Error): ErrorType {
  const msg = error.message.toLowerCase();
  if (msg.includes('session expired') || msg.includes('401')) return 'auth';
  if (msg.includes('429') || msg.includes('too many')) return 'rate_limit';
  if (msg.includes('404') || msg.includes('not found')) return 'not_found';
  if (msg.includes('failed to fetch') || msg.includes('network')) return 'network';
  return 'server';
}

const ERROR_CONFIGS: Record<ErrorType, { title: string; message: string; action: string; actionType: 'retry' | 'login' | 'wait' }> = {
  network: {
    title: 'Unable to connect',
    message: 'Check your internet connection and try again.',
    action: 'Retry',
    actionType: 'retry',
  },
  auth: {
    title: 'Session expired',
    message: 'Please sign in again to view your watchlists.',
    action: 'Sign in',
    actionType: 'login',
  },
  server: {
    title: 'Something went wrong',
    message: "We're looking into it. Please try again in a moment.",
    action: 'Retry',
    actionType: 'retry',
  },
  rate_limit: {
    title: 'Too many requests',
    message: 'Please wait a moment before trying again.',
    action: 'Retry',
    actionType: 'wait', // auto-retry after 5 seconds
  },
  not_found: {
    title: 'Watchlist not found',
    message: 'This watchlist may have been deleted.',
    action: 'Go to Dashboard',
    actionType: 'retry',
  },
};
```

### 7.3 Price Flash Animation

When a price changes on refresh, briefly flash the cell green (up) or red (down).

```typescript
// Track previous prices to detect changes
const prevPricesRef = useRef<Record<string, number>>({});

useEffect(() => {
  if (!data?.items) return;
  const newPrices: Record<string, number> = {};
  const flashes: Record<string, 'up' | 'down'> = {};

  for (const item of data.items) {
    if (item.current_price != null) {
      const prev = prevPricesRef.current[item.symbol];
      if (prev != null && prev !== item.current_price) {
        flashes[item.symbol] = item.current_price > prev ? 'up' : 'down';
      }
      newPrices[item.symbol] = item.current_price;
    }
  }

  prevPricesRef.current = newPrices;
  setPriceFlashes(flashes);

  // Clear flashes after animation
  const timer = setTimeout(() => setPriceFlashes({}), 1000);
  return () => clearTimeout(timer);
}, [data?.items]);
```

CSS animation:
```css
.price-flash-up { animation: flashGreen 1s ease-out; }
.price-flash-down { animation: flashRed 1s ease-out; }

@keyframes flashGreen {
  0% { background-color: rgba(34, 197, 94, 0.3); }
  100% { background-color: transparent; }
}
@keyframes flashRed {
  0% { background-color: rgba(239, 68, 68, 0.3); }
  100% { background-color: transparent; }
}
```

### 7.4 Mobile Card Layout

Below 768px breakpoint, replace the table with expandable cards.

**New file:** `components/watchlist/WatchListCard.tsx`

```typescript
interface WatchListCardProps {
  item: WatchListItemEnriched;
  onRemove: (symbol: string) => void;
  onEdit: (symbol: string) => void;
}

// Collapsed state: Symbol + Logo, Name, Price, Change %, IC Score badge
// Expanded state: All columns from active view in stacked key-value layout
```

Detection in parent:
```typescript
const isMobile = useMediaQuery('(max-width: 768px)');

return isMobile
  ? <WatchListCardList items={items} ... />
  : <WatchListTable items={items} activeView={activeView} ... />;
```

---

## 8. Phase 7: Cross-Feature Integration

**Priority:** P2 — Differentiating features
**Estimated effort:** 8-10 days total
**Owner:** Full stack

### 8.1 Screener → Watchlist Integration

**Backend change:** None. The `POST /api/v1/watchlists/:id/bulk` endpoint already accepts an array of symbols.

**Frontend changes:**

1. **Screener results table** (`app/screener/page.tsx` or equivalent):
   - Add checkbox column as first column
   - Add "+" icon in each row → opens `WatchListPicker` dropdown
   - Add floating action bar when ≥1 row selected

2. **New component:** `components/shared/WatchListPicker.tsx`
   - Fetches user's watchlists via `useWatchLists()` React Query hook
   - Dropdown with watchlist names and item counts
   - "Create New Watchlist" option at bottom
   - On select: calls `watchListAPI.bulkAddTickers(watchListId, selectedSymbols)`
   - Shows toast with result: "Added {N} to {watchlist name}"

### 8.2 Reddit Trends → Watchlist Integration

**Backend change:** None. Same `POST /api/v1/watchlists/:id/items` endpoint.

**Frontend changes:** Add `WatchListPicker` trigger button to each row in the Reddit Trends table. Same component as Screener integration.

### 8.3 Inline Alert Creation

**New component:** `components/watchlist/InlineAlertPanel.tsx`

This panel expands below a watchlist row when the user clicks the bell icon.

**API used:** `POST /api/v1/alerts` (existing endpoint)

**Request body mapping:**
```typescript
{
  watch_list_id: currentWatchListId,    // From page context
  symbol: item.symbol,                   // From row
  alert_type: selectedAlertType,         // User selection
  conditions: { threshold: inputValue }, // User input
  frequency: selectedFrequency,          // User selection
  notify_email: emailToggle,             // User toggle
  notify_in_app: inAppToggle,            // User toggle
  name: `${item.symbol} ${alertTypeLabel} ${inputValue}`,  // Auto-generated
}
```

**Alert count display:** The `alert_count` field is already returned by the enriched query (Phase 2). Render as a badge on the bell icon.

### 8.4 CSV Import/Export

**Export (client-side):**
```typescript
function exportToCSV(items: WatchListItemEnriched[], columns: ColumnDefinition[], filename: string) {
  const header = columns.map(c => c.label).join(',');
  const rows = items.map(item =>
    columns.map(c => {
      const val = item[c.id as keyof WatchListItemEnriched];
      if (val == null) return '';
      if (typeof val === 'string' && val.includes(',')) return `"${val}"`;
      return String(val);
    }).join(',')
  );
  const csv = [header, ...rows].join('\n');
  const blob = new Blob([csv], { type: 'text/csv' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
```

**Import (client-side parsing + bulk API):**
```typescript
async function importCSV(file: File, watchListId: string): Promise<{ added: string[]; failed: string[] }> {
  const text = await file.text();
  const lines = text.split('\n').filter(l => l.trim());
  const header = lines[0].split(',').map(h => h.trim().toLowerCase());
  const symbolIdx = header.findIndex(h => h === 'symbol' || h === 'ticker');
  if (symbolIdx === -1) throw new Error('CSV must have a "Symbol" or "Ticker" column');

  const symbols = lines.slice(1)
    .map(line => line.split(',')[symbolIdx]?.trim().toUpperCase())
    .filter(Boolean);

  // Deduplicate
  const uniqueSymbols = [...new Set(symbols)].slice(0, 500);

  const result = await watchListAPI.bulkAddTickers(watchListId, uniqueSymbols);
  return result;
}
```

---

## 9. Database Migrations

### Migration Checklist

| # | Migration | Phase | Description | Reversible? |
|---|-----------|-------|-------------|-------------|
| 1 | Verify 010 applied | Phase 1 | `watch_lists`, `watch_list_items` tables + triggers | N/A (check only) |
| 2 | Verify 012 applied | Phase 1 | `alert_rules`, `alert_trigger_logs` tables | N/A (check only) |
| 3 | `XXX_add_performance_to_screener.sql` | Phase 2 | Add `perf_1w` through `perf_1y` columns to `screener_data` materialized view | Yes (recreate view without columns) |
| 4 | Backfill default watchlists | Phase 1 | For existing users missing a default watchlist | Yes (DELETE WHERE is_default=true AND item_count=0) |

### Migration 4: Backfill Default Watchlists

```sql
-- One-time backfill: create default watchlists for existing users who don't have one
INSERT INTO watch_lists (user_id, name, description, is_default, display_order)
SELECT u.id, 'My Watch List', 'Default watch list', TRUE, 0
FROM users u
WHERE NOT EXISTS (
    SELECT 1 FROM watch_lists wl WHERE wl.user_id = u.id AND wl.is_default = TRUE
);
```

---

## 10. API Contract Changes

### Existing Endpoints (No Breaking Changes)

All existing watchlist and alert endpoints remain unchanged in behavior. The `GET /api/v1/watchlists/:id` response is **extended** with additional fields (all nullable with `omitempty`), which is backward-compatible.

### Response Schema: `GET /api/v1/watchlists/:id`

**Current response (maintained for backward compat):**
```json
{
  "id": "uuid",
  "name": "AI Stocks 2026",
  "description": "...",
  "is_default": false,
  "item_count": 12,
  "items": [
    {
      "id": "uuid",
      "symbol": "NVDA",
      "name": "NVIDIA Corporation",
      "exchange": "NASDAQ",
      "asset_type": "CS",
      "logo_url": "https://...",
      "current_price": 892.45,
      "price_change": 12.30,
      "price_change_pct": 1.40,
      "volume": 42500000,
      "prev_close": 880.15,
      "reddit_rank": 5,
      "reddit_mentions": 847,
      "reddit_popularity": 72.3,
      "reddit_trend": "rising",
      "reddit_rank_change": 3,
      "notes": "Watching for pullback",
      "tags": ["AI", "Semis"],
      "target_buy_price": 800.00,
      "target_sell_price": 1000.00,
      "added_at": "2026-02-12T...",
      "display_order": 0
    }
  ],
  "created_at": "2026-02-10T...",
  "updated_at": "2026-02-18T..."
}
```

**New fields added (Phase 2+):**
```json
{
  "...existing fields...",
  "summary": {
    "total_tickers": 12,
    "avg_ic_score": 72.5,
    "avg_day_change_pct": 1.23,
    "avg_dividend_yield": 0.85,
    "median_market_cap": 45200000000,
    "reddit_trending_count": 3
  },
  "items": [
    {
      "...existing fields...",
      "ic_score": 78,
      "ic_rating": "Strong Buy",
      "value_score": 65,
      "growth_score": 82,
      "profitability_score": 71,
      "financial_health_score": 55,
      "momentum_score": 90,
      "analyst_consensus_score": 73,
      "insider_activity_score": 40,
      "institutional_score": 68,
      "sector_percentile": 88.5,
      "lifecycle_stage": "growth",
      "pe_ratio": 32.5,
      "pb_ratio": 18.2,
      "ps_ratio": 28.1,
      "roe": 35.2,
      "roa": 18.7,
      "gross_margin": 72.1,
      "operating_margin": 54.3,
      "net_margin": 48.9,
      "debt_to_equity": 0.41,
      "current_ratio": 4.17,
      "revenue_growth": 122.4,
      "eps_growth": 88.1,
      "dividend_yield": 0.03,
      "payout_ratio": 1.2,
      "perf_1w": 5.23,
      "perf_1m": 12.8,
      "perf_3m": 28.4,
      "perf_6m": 45.1,
      "perf_ytd": 18.9,
      "perf_1y": 185.2,
      "alert_count": 2
    }
  ]
}
```

All new fields are nullable (`omitempty` in Go). Free-tier users receive `null` for premium-gated fields. No existing field is removed or renamed.

### New Endpoint: Watchlist Summary (Optional)

If computing summary metrics on every detail request adds unacceptable latency, we can split it into a separate endpoint:

```
GET /api/v1/watchlists/:id/summary
```

Response:
```json
{
  "total_tickers": 12,
  "avg_ic_score": 72.5,
  "avg_day_change_pct": 1.23,
  "avg_dividend_yield": 0.85,
  "median_market_cap": 45200000000,
  "reddit_trending_count": 3
}
```

**Recommendation:** Include summary in the main response for now. Split only if profiling shows it adds >100ms.

---

## 11. Performance Budget

### API Response Time Targets

| Endpoint | Watchlist Size | Target p50 | Target p95 | Current (est.) |
|----------|---------------|-----------|-----------|----------------|
| `GET /watchlists` (list) | N/A | 50ms | 200ms | Unknown (broken) |
| `GET /watchlists/:id` (detail) | 10 items | 200ms | 500ms | ~1.5s (sequential Polygon) |
| `GET /watchlists/:id` (detail) | 50 items | 300ms | 800ms | ~6s (sequential Polygon) |
| `GET /watchlists/:id` (detail) | 100 items | 400ms | 1200ms | ~12s (sequential Polygon) |
| `POST /watchlists/:id/items` (add) | N/A | 100ms | 300ms | Unknown |
| `POST /watchlists/:id/bulk` (import) | 100 symbols | 500ms | 2000ms | Unknown |

After batch optimization (Phase 3), the detail endpoint should be ~250ms for 50 items (DB query ~50ms + 1 batch Polygon call ~200ms).

### Frontend Rendering Targets

| Interaction | Target | Notes |
|-------------|--------|-------|
| Initial table render (50 rows) | <100ms | From data available to pixels on screen |
| View switch (re-render columns) | <50ms | Client-side column swap, no API call |
| Sort click (50 rows) | <30ms | Client-side `Array.sort()` |
| Filter typing (debounced) | <50ms | Client-side filter on each keystroke |
| Price flash animation | 60fps | CSS animation, no JS per frame |

### Bundle Size Budget

| Addition | Estimated Size | Justification |
|----------|---------------|---------------|
| React Query | ~13KB gzipped | Replaces custom polling code (~2KB); net +11KB |
| Column registry + view presets | ~3KB | Static config |
| New table component | ~5KB | Replaces existing ~3KB; net +2KB |
| Error state components | ~2KB | New |
| Mobile card component | ~3KB | New |
| **Total delta** | **~21KB gzipped** | Acceptable for the feature value |

---

## 12. Testing Strategy

### Backend Tests

#### Unit Tests (Go)

**File:** `backend/handlers/watchlist_handlers_test.go` (fix + extend)

| Test | What it validates |
|------|-------------------|
| `TestListWatchLists` | Returns user's watchlists with item counts |
| `TestCreateWatchList` | Creates watchlist, returns UUID |
| `TestCreateWatchList_LimitExceeded` | **NEW**: Free user with 3 watchlists gets 403 |
| `TestGetWatchList_WithEnrichedData` | **NEW**: Response includes IC Score, fundamentals, Reddit data |
| `TestGetWatchList_PolygonFailure` | **NEW**: Returns items without prices when Polygon is down |
| `TestGetWatchList_PremiumGating` | **NEW**: Free user gets null for gated fields |
| `TestAddTicker_DuplicateSymbol` | Returns 409 Conflict |
| `TestAddTicker_InvalidSymbol` | Returns 404 (ticker not in DB) |
| `TestAddTicker_FreeTierLimit` | Returns 403 at 10 items |
| `TestBulkAddTickers` | Returns added/failed arrays |
| `TestDeleteWatchList_DefaultProtected` | **NEW**: Default watchlist returns 400 |

**File:** `backend/services/polygon_test.go` (extend)

| Test | What it validates |
|------|-------------------|
| `TestGetBatchQuotes_StocksOnly` | Batch fetches stock prices |
| `TestGetBatchQuotes_CryptoOnly` | Uses cache, falls back to API |
| `TestGetBatchQuotes_Mixed` | Handles stock + crypto in single call |
| `TestGetBatchQuotes_ChunkSplitting` | Splits >50 symbols into batches |
| `TestGetBatchQuotes_PartialFailure` | Returns partial results on API error |

#### Integration Tests

```bash
# Run with real database (test DB)
cd backend && go test ./handlers -v -run TestWatchList -tags=integration
cd backend && go test ./services -v -run TestWatchListService -tags=integration
```

### Frontend Tests

#### Component Tests (React Testing Library)

| Test | Component | What it validates |
|------|-----------|-------------------|
| Renders all column types | `WatchListTable` | Currency, percent, badge, range formatting |
| Sorts on column click | `WatchListTable` | Ascending → descending → clear cycle |
| Filters by search text | `WatchListTable` | Symbol and name matching |
| Filters by asset type | `WatchListTable` | "Stocks", "ETFs", "Crypto" chips |
| Shows premium gate | `ViewSwitcher` | Lock icon on premium views for free user |
| Shows error state | `ErrorState` | Correct message for each error type |
| Renders mobile cards | `WatchListCard` | Below 768px breakpoint |
| Flashes on price change | `WatchListTable` | Green/red flash class applied |

#### E2E Tests (Playwright)

| Flow | What it validates |
|------|-------------------|
| Create watchlist → add ticker → see price | Full happy path |
| Switch view → sort → filter | Table interaction flow |
| Hit item limit → see upgrade prompt | Freemium gate |
| Import CSV → see results toast | Bulk add flow |
| Click bell → create alert → see badge | Alert integration |

### Database Tests

```sql
-- Verify screener_data view has performance columns
SELECT column_name FROM information_schema.columns
WHERE table_name = 'screener_data'
AND column_name LIKE 'perf_%';
-- Expected: perf_1w, perf_1m, perf_3m, perf_6m, perf_ytd, perf_1y

-- Verify enriched query returns all columns
EXPLAIN ANALYZE
SELECT wli.*, t.name, t.exchange, sd.ic_score, sd.pe_ratio
FROM watch_list_items wli
JOIN tickers t ON wli.symbol = t.symbol
LEFT JOIN screener_data sd ON wli.symbol = sd.symbol
WHERE wli.watch_list_id = 'test-uuid'
ORDER BY wli.display_order;
-- Verify: all JOINs use index scans, execution time < 50ms
```

---

## 13. Deployment Plan

### Phase 1 Deployment (Fix)

```
1. Apply migrations 010 + 012 in production (if missing)
2. Run backfill migration for default watchlists
3. Fix test suite table name → merge to main
4. Deploy backend with graceful degradation logging
5. Verify: curl all 12 endpoints → 200/201/204
6. Verify: frontend loads /watchlist without error
7. Monitor: 0 5xx errors over 48 hours
```

### Phase 2-3 Deployment (Enriched Data + Batch Prices)

```
1. Apply screener_data performance columns migration (IC Score service DB)
2. Refresh materialized view: SELECT refresh_screener_data();
3. Deploy backend with:
   - GetWatchListItemsWithEnrichedData query
   - GetBatchQuotes method
   - Updated GetWatchListWithItems service
4. Verify: GET /watchlists/:id returns ic_score, pe_ratio, perf_1w etc.
5. Monitor: API response time p95 < 800ms for 50-item watchlist
```

### Phase 5-6 Deployment (Frontend)

```
1. Install @tanstack/react-query
2. Build frontend: NEXT_PUBLIC_API_URL=/api/v1 npm run build
3. Deploy frontend with:
   - Column views + ViewSwitcher
   - React Query polling
   - Error states
   - Mobile card layout
4. Feature flag: 10% rollout → monitor task completion + error rates
5. Full rollout after 1 week at 10% with no regressions
```

### Phase 7 Deployment (Integrations)

```
1. Deploy Screener → Watchlist integration
2. Deploy Reddit Trends → Watchlist button
3. Deploy inline alert panel
4. Deploy CSV import/export
5. Deploy premium gating UI
6. No feature flag — these are additive features, not modifications
```

### Rollback Plan

Each phase is independently deployable and rollback-safe:

- **Backend**: Go binary rollback via `kubectl rollout undo deployment/investorcenter-backend`
- **Frontend**: Previous Docker image rollback via `kubectl rollout undo deployment/investorcenter-frontend`
- **Database**: Performance columns migration is additive (new columns in materialized view). Rollback = recreate view without those columns. No data loss.
- **React Query**: If React Query causes issues, the polling logic is self-contained in `useWatchList` hook. Reverting = swap hook implementation back to `useEffect` + `setInterval`.

---

## Appendix: Current Implementation Reference

### File Inventory

| File | Language | Lines | Modification Scope |
|------|----------|-------|--------------------|
| `backend/database/watchlists.go` | Go | ~250 | Add `GetWatchListItemsWithEnrichedData` |
| `backend/models/watchlist.go` | Go | ~120 | Add `WatchListItemEnriched`, `WatchListWithItemsEnriched`, `WatchListSummaryMetrics` |
| `backend/services/watchlist_service.go` | Go | ~80 | Rewrite `GetWatchListWithItems` for batch prices + enriched data |
| `backend/services/polygon.go` | Go | ~1400 | Add `GetBatchQuotes`, `chunkSlice` |
| `backend/handlers/watchlist_handlers.go` | Go | ~330 | Add watchlist count limit check, default delete protection |
| `backend/handlers/watchlist_handlers_test.go` | Go | ~500 | Fix `stocks` → `tickers`, add new test cases |
| `components/watchlist/WatchListTable.tsx` | TSX | ~170 | Rewrite to data-driven column rendering |
| `app/watchlist/[id]/page.tsx` | TSX | ~200 | Replace polling with React Query |
| `lib/api/watchlist.ts` | TS | ~130 | Update `WatchListItem` interface with new fields |
| **New:** `lib/watchlist/columns.ts` | TS | ~200 | Column registry + view presets |
| **New:** `lib/hooks/useWatchList.ts` | TS | ~50 | React Query hooks |
| **New:** `components/watchlist/ViewSwitcher.tsx` | TSX | ~80 | View dropdown |
| **New:** `components/watchlist/ErrorState.tsx` | TSX | ~60 | Error handling |
| **New:** `components/watchlist/WatchListCard.tsx` | TSX | ~100 | Mobile card layout |
| **New:** `components/shared/WatchListPicker.tsx` | TSX | ~80 | Screener/Reddit integration |
| **New:** `components/watchlist/InlineAlertPanel.tsx` | TSX | ~120 | Alert creation from row |

### Existing Database Schema (For Reference)

```
watch_lists
├── id UUID PK
├── user_id UUID FK → users(id) CASCADE
├── name VARCHAR(255) NOT NULL
├── description TEXT
├── is_default BOOLEAN
├── display_order INTEGER
├── is_public BOOLEAN
├── public_slug VARCHAR(100) UNIQUE
├── created_at TIMESTAMP
└── updated_at TIMESTAMP

watch_list_items
├── id UUID PK
├── watch_list_id UUID FK → watch_lists(id) CASCADE
├── symbol VARCHAR(20) NOT NULL
├── notes TEXT
├── tags TEXT[]
├── target_buy_price DECIMAL(20,4)
├── target_sell_price DECIMAL(20,4)
├── added_at TIMESTAMP
├── display_order INTEGER
└── UNIQUE(watch_list_id, symbol)

alert_rules
├── id UUID PK
├── user_id UUID FK → users(id) CASCADE
├── watch_list_id UUID FK → watch_lists(id) CASCADE
├── watch_list_item_id UUID FK → watch_list_items(id) CASCADE (nullable)
├── symbol VARCHAR(20) NOT NULL
├── alert_type VARCHAR(50) CHECK(...)
├── conditions JSONB NOT NULL
├── is_active BOOLEAN
├── frequency VARCHAR(20) CHECK(once|daily|always)
├── notify_email BOOLEAN
├── notify_in_app BOOLEAN
├── name VARCHAR(255) NOT NULL
├── description TEXT
├── last_triggered_at TIMESTAMPTZ
├── trigger_count INTEGER
├── created_at TIMESTAMPTZ
└── updated_at TIMESTAMPTZ

screener_data (MATERIALIZED VIEW — 48+ columns)
├── symbol (UNIQUE INDEX)
├── name, sector, industry, market_cap, asset_type, active
├── price (latest close)
├── pe_ratio, pb_ratio, ps_ratio
├── roe, roa, gross_margin, operating_margin, net_margin
├── debt_to_equity, current_ratio
├── revenue_growth, eps_growth_yoy
├── dividend_yield, payout_ratio
├── ic_score, ic_rating
├── value_score, growth_score, profitability_score, ...  (10 sub-factors)
├── ic_sector_percentile, lifecycle_stage
├── perf_1w, perf_1m, perf_3m, perf_6m, perf_ytd, perf_1y  (NEW)
└── refreshed_at
```

---

*End of Technical Specification*

*This document should be reviewed alongside the [PRD](./prd-watchlist-redesign.md) by the engineering team. Phase 1 (Fix) can begin immediately. Phases 2-3 (backend enrichment) should be implemented before frontend phases to ensure the API contract is stable.*
