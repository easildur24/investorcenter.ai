# Tech Spec: Project 1 — Fundamentals API Layer

**Parent PRD:** `docs/prd-enhanced-fundamentals-experience.md`
**Priority:** P0 — Enables all other projects
**Estimated Effort:** 2-3 sprints
**Dependencies:** None (foundational project)

---

## 1. Overview

This project exposes existing backend-computed data — sector percentiles, peer comparisons, fair values, lifecycle classifications, health summaries, and metric histories — through new REST API endpoints on the Go backend. No new computation is required; this is plumbing that unlocks all downstream frontend work.

## 2. Architecture Context

### Current Stack
- **Backend:** Go + Gin Gonic (`backend/main.go`)
- **Database:** PostgreSQL with SQLx (`backend/database/db.go`)
- **ORM Pattern:** Raw SQL with `DB.Get()` / `DB.Select()` via `jmoiron/sqlx`
- **Models:** `backend/models/` with `ToResponse()` conversion pattern
- **Route Registration:** Inline in `main.go` under `/api/v1/stocks` group
- **Response Envelope:** `gin.H{"data": ..., "meta": {...}}` or direct struct serialization

### Existing Backend Capabilities (Already Computed)

| Capability | Database Table / View | Go Model | DB Access Function |
|---|---|---|---|
| Sector percentiles (min→p90→max) | `mv_latest_sector_percentiles` | `models.SectorPercentile` | `database.GetSectorPercentiles(sector)` |
| Single metric percentile | Same view | Same | `database.GetSectorPercentile(sector, metric)` |
| Percentile calculation (interpolated) | N/A (in-memory) | N/A | `database.CalculatePercentile(sector, metric, value)` |
| Lifecycle classification | `lifecycle_classifications` | `models.LifecycleClassification` | `database.GetLifecycleClassification(ticker)` |
| Stock peers | `stock_peers` | `models.StockPeer` | `database.GetStockPeers(ticker, limit)` |
| Peers with IC Scores | `stock_peers` JOIN `ic_scores` | `models.StockPeerResponse` | `database.GetStockPeersWithScores(ticker, limit)` |
| Peer comparison summary | Composite | `models.PeerComparisonResponse` | `database.GetPeerComparisonSummary(ticker)` |
| IC Score (with sub-factors) | `ic_scores` | `models.ICScore` | Existing handler `GetICScore` |
| F-Score / Z-Score | FMP API | `FMPScore` (in `services.FMPClient`) | `fmpClient.GetScore(ticker)` |
| Fair value (Graham, DCF) | FMP API + IC Score pipeline | `FMPRatiosTTM`, `fundamental_metrics_extended` | `fmpClient.GetRatiosTTM(ticker)` |
| Financial statements (quarterly) | `financial_statements` | `models.FinancialStatement` | `FinancialsService` methods |
| Sector rank | `ic_scores` + `tickers` | Computed | `database.GetSectorRank(sector, ticker, score)` |
| Lower-is-better metric map | In-memory | `models.LowerIsBetterMetrics` | Direct access |
| Tracked metric list | In-memory | `models.TrackedMetrics` | Direct access |

---

## 3. New API Endpoints

All endpoints are **public** (no auth required) and registered under the existing `stocks` group in `main.go`:

```go
stocks := v1.Group("/stocks")
```

### 3.1 `GET /api/v1/stocks/:ticker/sector-percentiles`

**Purpose:** Return sector percentile distribution data for all tracked metrics for the stock's sector, plus the stock's computed percentile for each metric.

**Handler:** `handlers.GetSectorPercentiles`

**Query Params:**
- `metrics` (optional): Comma-separated metric names to filter (e.g., `pe_ratio,roe,debt_to_equity`). If omitted, return all tracked metrics.

**Implementation:**

```go
// 1. Look up stock's sector from tickers table
stock, err := database.GetStockBySymbol(ticker)
sector := stock.Sector

// 2. Get all sector percentile data
percentiles, err := database.GetSectorPercentiles(sector)

// 3. Get stock's current metric values from FMP/IC Score
metrics, err := fmpClient.GetComprehensiveMetrics(ticker)

// 4. For each metric: calculate stock's percentile position
for _, sp := range percentiles {
    value := extractMetricValue(metrics, sp.MetricName)
    stockPercentile := database.CalculatePercentile(sector, sp.MetricName, value)
}
```

**Response Schema:**

```json
{
  "ticker": "AAPL",
  "sector": "Technology",
  "calculated_at": "2026-02-27",
  "sample_count": 245,
  "metrics": {
    "pe_ratio": {
      "value": 28.5,
      "percentile": 65.3,
      "lower_is_better": true,
      "distribution": {
        "min": 5.2,
        "p10": 12.1,
        "p25": 18.3,
        "p50": 25.7,
        "p75": 35.2,
        "p90": 52.8,
        "max": 312.0
      },
      "sample_count": 245
    },
    "roe": {
      "value": 147.2,
      "percentile": 95.1,
      "lower_is_better": false,
      "distribution": { ... },
      "sample_count": 230
    }
  },
  "meta": {
    "source": "mv_latest_sector_percentiles",
    "metric_count": 25,
    "timestamp": "2026-02-28T12:00:00Z"
  }
}
```

**New Files:**
- `backend/handlers/fundamentals_handlers.go` — New handler file for all fundamentals endpoints
- `backend/database/fundamentals.go` — New DB queries (composites of existing functions)

**Estimated LOC:** ~150 handler, ~80 DB queries

---

### 3.2 `GET /api/v1/stocks/:ticker/peers`

**Purpose:** Return top 5 peers with comparison metrics for side-by-side display.

**Handler:** `handlers.GetStockPeers`

**Query Params:**
- `limit` (optional, default 5, max 10)

**Implementation:**

```go
// 1. Get peers with IC Scores (existing function)
peers, err := database.GetStockPeersWithScores(ticker, limit)

// 2. For each peer, fetch key comparison metrics
//    (P/E, ROE, Revenue Growth, Net Margin, D/E, Market Cap)
//    from fmpClient or comprehensive metrics cache

// 3. Get stock's own metrics for comparison
// 4. Return enriched response
```

**Response Schema:**

```json
{
  "ticker": "AAPL",
  "ic_score": 78.5,
  "peers": [
    {
      "ticker": "MSFT",
      "company_name": "Microsoft Corporation",
      "ic_score": 82.1,
      "similarity_score": 0.87,
      "metrics": {
        "pe_ratio": 33.2,
        "roe": 38.5,
        "revenue_growth_yoy": 12.3,
        "net_margin": 35.1,
        "debt_to_equity": 0.42,
        "market_cap": 3100000000000
      }
    }
  ],
  "stock_metrics": {
    "pe_ratio": 28.5,
    "roe": 147.2,
    "revenue_growth_yoy": 5.1,
    "net_margin": 25.3,
    "debt_to_equity": 1.87,
    "market_cap": 2800000000000
  },
  "avg_peer_score": 75.3,
  "vs_peers_delta": 3.2,
  "meta": {
    "similarity_algorithm": "5-factor (market_cap, revenue_growth, net_margin, pe_ratio, beta)",
    "timestamp": "2026-02-28T12:00:00Z"
  }
}
```

**Existing DB Functions Used:**
- `database.GetStockPeersWithScores(ticker, limit)` (already exists in `database/ic_score_phase3.go`)
- `database.GetPeerComparisonSummary(ticker)` (already exists)

**New Code:** ~120 LOC handler enrichment (fetching comparison metrics per peer)

---

### 3.3 `GET /api/v1/stocks/:ticker/fair-value`

**Purpose:** Return computed fair value estimates (DCF, Graham Number, EPV) with margin of safety calculation.

**Handler:** `handlers.GetFairValue`

**Implementation:**

```go
// 1. Get FMP ratios TTM (contains Graham Number)
ratios, err := fmpClient.GetRatiosTTM(ticker)

// 2. Get IC Score pipeline fair values from fundamental_metrics_extended
//    (dcf_fair_value, epv_fair_value, graham_number, wacc)
fairValues, err := database.GetFairValueMetrics(ticker)

// 3. Get current stock price
price, err := polygonClient.GetQuote(ticker)

// 4. Get analyst consensus target price
estimates, err := fmpClient.GetAnalystEstimates(ticker)

// 5. Calculate margin of safety for each model
```

**Response Schema:**

```json
{
  "ticker": "AAPL",
  "current_price": 178.50,
  "models": {
    "dcf": {
      "fair_value": 195.30,
      "upside_percent": 9.4,
      "confidence": "medium",
      "inputs": {
        "wacc": 9.2,
        "terminal_growth": 2.5,
        "fcf_ttm": 110500000000
      }
    },
    "graham_number": {
      "fair_value": 142.80,
      "upside_percent": -20.0,
      "confidence": "high"
    },
    "epv": {
      "fair_value": 168.40,
      "upside_percent": -5.7,
      "confidence": "medium"
    }
  },
  "analyst_consensus": {
    "target_price": 210.50,
    "upside_percent": 17.9,
    "num_analysts": 38,
    "consensus": "Buy"
  },
  "margin_of_safety": {
    "avg_fair_value": 168.83,
    "zone": "fairly_valued",
    "description": "Stock is trading within 10% of average fair value estimate"
  },
  "meta": {
    "suppressed": false,
    "suppression_reason": null,
    "timestamp": "2026-02-28T12:00:00Z"
  }
}
```

**New DB Function:**
- `database.GetFairValueMetrics(ticker)` — query `fundamental_metrics_extended` for DCF, Graham, EPV columns

**Estimated LOC:** ~180 handler + ~40 DB

---

### 3.4 `GET /api/v1/stocks/:ticker/health-summary`

**Purpose:** Return a synthesized fundamental health assessment combining multiple quality signals.

**Handler:** `handlers.GetHealthSummary`

**Implementation:**

```go
// 1. Get IC Score (includes financial_health_score, lifecycle_stage)
icScore, err := getLatestICScore(ticker)

// 2. Get F-Score and Z-Score from FMP
fmpScore, err := fmpClient.GetScore(ticker)

// 3. Get lifecycle classification
lifecycle, err := database.GetLifecycleClassification(ticker)

// 4. Get sector percentile data for this stock
percentiles, err := getSectorPercentilesForStock(ticker)

// 5. Compute health badge
healthBadge := computeHealthBadge(
    fmpScore.PiotroskiFScore,
    fmpScore.AltmanZScore,
    icScore.FinancialHealthScore,
    debtToEquityPercentile,
)

// 6. Generate strengths (top quartile metrics)
strengths := generateStrengths(percentiles, 3)

// 7. Generate concerns (bottom quartile metrics + red flags)
concerns := generateConcerns(percentiles, metrics, 3)

// 8. Detect red flags
redFlags := detectRedFlags(metrics, percentiles, lifecycle)
```

**Response Schema:**

```json
{
  "ticker": "AAPL",
  "health": {
    "badge": "Strong",
    "score": 82,
    "components": {
      "piotroski_f_score": { "value": 7, "max": 9, "interpretation": "Strong" },
      "altman_z_score": { "value": 5.8, "zone": "safe", "interpretation": "Healthy" },
      "ic_financial_health": { "value": 85.2, "max": 100 },
      "debt_percentile": { "value": 62, "interpretation": "Moderate leverage" }
    }
  },
  "lifecycle": {
    "stage": "mature",
    "description": "Mature company with stable operations. Focus on profitability, cash flow, and capital efficiency.",
    "classified_at": "2026-02-27"
  },
  "strengths": [
    {
      "metric": "net_margin",
      "value": 25.3,
      "percentile": 92,
      "message": "Net margin ranks in top 10% of Technology sector"
    },
    {
      "metric": "roe",
      "value": 147.2,
      "percentile": 95,
      "message": "Return on equity is exceptional vs. sector peers"
    }
  ],
  "concerns": [
    {
      "metric": "debt_to_equity",
      "value": 1.87,
      "percentile": 78,
      "message": "Debt/Equity is above sector median (0.65)"
    }
  ],
  "red_flags": [
    {
      "id": "high_leverage",
      "severity": "medium",
      "title": "Above-average leverage",
      "description": "Debt/Equity of 1.87 exceeds 78% of Technology sector peers",
      "related_metrics": ["debt_to_equity", "interest_coverage"]
    }
  ],
  "meta": {
    "timestamp": "2026-02-28T12:00:00Z"
  }
}
```

**Health Badge Algorithm:**

```go
func computeHealthBadge(fScore int, zScore float64, icHealth float64, dePercentile float64) string {
    score := 0.0

    // F-Score contribution (0-9 → 0-30 points)
    score += float64(fScore) / 9.0 * 30.0

    // Z-Score contribution (0-30 points)
    switch {
    case zScore > 2.99: score += 30.0   // Safe zone
    case zScore > 1.81: score += 15.0   // Grey zone
    default: score += 0.0               // Distress zone
    }

    // IC Health contribution (0-100 → 0-25 points)
    score += icHealth / 100.0 * 25.0

    // D/E percentile contribution (inverted, 0-15 points)
    score += (100 - dePercentile) / 100.0 * 15.0

    switch {
    case score >= 80: return "Strong"
    case score >= 65: return "Healthy"
    case score >= 45: return "Fair"
    case score >= 25: return "Weak"
    default: return "Distressed"
    }
}
```

**Red Flag Rules (implemented server-side):**

| Rule ID | Condition | Severity |
|---|---|---|
| `unsustainable_dividend` | Payout ratio > 100% OR FCF payout > 120% | high |
| `high_leverage` | D/E > sector p90 AND Interest Coverage < 2x | high |
| `declining_profitability` | Op margin declined 3+ consecutive Qs | medium |
| `cash_burn` | Negative FCF 2+ consecutive Qs (non-hypergrowth) | high |
| `earnings_quality` | OCF/NI < 0.5 for 2+ Qs | medium |
| `valuation_outlier` | P/E > sector p95 AND PEG > 3 | low |
| `altman_distress` | Z-Score < 1.81 | high |
| `weak_piotroski` | F-Score ≤ 3 | medium |

**Estimated LOC:** ~300 handler + ~60 helper functions + ~50 DB

---

### 3.5 `GET /api/v1/stocks/:ticker/metric-history/:metric`

**Purpose:** Return 5-year quarterly history for a single metric (for sparklines and full charts).

**Handler:** `handlers.GetMetricHistory`

**Query Params:**
- `timeframe` (optional, default `quarterly`): `quarterly` | `annual`
- `limit` (optional, default 20): Number of periods

**Implementation:**

```go
// 1. Map metric name to financial statement field
//    e.g., "revenue" → income_statement → "revenues"
//    e.g., "gross_margin" → ratios → "gross_margin"
//    e.g., "debt_to_equity" → ratios → "debt_to_equity"

// 2. Query financial_statements table for the appropriate statement type
//    Extract the specific field from the JSONB `data` column

// 3. For ratio metrics, query from ratios statement type
// 4. Return time series
```

**Response Schema:**

```json
{
  "ticker": "AAPL",
  "metric": "revenue",
  "timeframe": "quarterly",
  "unit": "USD",
  "data_points": [
    {
      "period_end": "2026-01-31",
      "fiscal_year": 2026,
      "fiscal_quarter": 1,
      "value": 124300000000,
      "yoy_change": 0.051
    },
    {
      "period_end": "2025-10-31",
      "fiscal_year": 2025,
      "fiscal_quarter": 4,
      "value": 119800000000,
      "yoy_change": 0.062
    }
  ],
  "trend": {
    "direction": "up",
    "slope": 0.043,
    "consecutive_growth_quarters": 6
  },
  "meta": {
    "available_periods": 20,
    "source": "sec_filings",
    "timestamp": "2026-02-28T12:00:00Z"
  }
}
```

**Metric-to-Statement Mapping:**

```go
var metricStatementMap = map[string]struct {
    StatementType string
    FieldName     string
    Unit          string
}{
    "revenue":          {"income", "revenues", "USD"},
    "net_income":       {"income", "net_income_loss", "USD"},
    "gross_profit":     {"income", "gross_profit", "USD"},
    "operating_income": {"income", "operating_income_loss", "USD"},
    "eps":              {"income", "diluted_earnings_per_share", "USD"},
    "free_cash_flow":   {"cash_flow", "computed_fcf", "USD"},
    "gross_margin":     {"ratios", "gross_margin", "percent"},
    "operating_margin": {"ratios", "operating_margin", "percent"},
    "net_margin":       {"ratios", "net_profit_margin", "percent"},
    "roe":              {"ratios", "return_on_equity", "percent"},
    "roa":              {"ratios", "return_on_assets", "percent"},
    "debt_to_equity":   {"ratios", "debt_to_equity", "ratio"},
    "current_ratio":    {"ratios", "current_ratio", "ratio"},
}
```

**Existing Infrastructure Used:**
- `FinancialsService` already queries `financial_statements` with JSONB data extraction
- `models.FinancialStatement.Data` is `map[string]interface{}` — can extract any field

**Estimated LOC:** ~120 handler + ~60 DB query + ~30 metric mapping

---

## 4. Route Registration

Add to `backend/main.go` inside the existing `stocks` group:

```go
stocks := v1.Group("/stocks")
{
    // ... existing routes ...

    // Fundamentals enhancement endpoints (Project 1)
    fundamentalsHandler := handlers.NewFundamentalsHandler()
    stocks.GET("/:ticker/sector-percentiles", fundamentalsHandler.GetSectorPercentiles)
    stocks.GET("/:ticker/peers", fundamentalsHandler.GetStockPeers)
    stocks.GET("/:ticker/fair-value", fundamentalsHandler.GetFairValue)
    stocks.GET("/:ticker/health-summary", fundamentalsHandler.GetHealthSummary)
    stocks.GET("/:ticker/metric-history/:metric", fundamentalsHandler.GetMetricHistory)
}
```

## 5. Frontend Route Registration

Add to `lib/api/routes.ts` in the `stocks` object:

```typescript
export const stocks = {
    // ... existing routes ...

    // Fundamentals enhancement endpoints
    sectorPercentiles: (ticker: string) => `/stocks/${ticker}/sector-percentiles`,
    peers: (ticker: string) => `/stocks/${ticker}/peers`,
    fairValue: (ticker: string) => `/stocks/${ticker}/fair-value`,
    healthSummary: (ticker: string) => `/stocks/${ticker}/health-summary`,
    metricHistory: (ticker: string, metric: string) => `/stocks/${ticker}/metric-history/${metric}`,
} as const;
```

## 6. New Files Summary

| File | Type | Purpose | Est. LOC |
|---|---|---|---|
| `backend/handlers/fundamentals_handlers.go` | Handler | All 5 endpoint handlers | ~600 |
| `backend/database/fundamentals.go` | DB layer | Composite queries for health summary, fair value, metric history | ~200 |
| `backend/models/fundamentals.go` | Models | Response structs for new endpoints | ~250 |

## 7. Performance Considerations

- **Sector percentiles endpoint:** Single DB query to materialized view + metric extraction. Target: <100ms.
- **Peers endpoint:** 1 DB query (existing) + N FMP calls for metrics (where N = peer count). Consider caching FMP results or querying from `fundamental_metrics_extended` instead. Target: <300ms.
- **Fair value endpoint:** 2-3 FMP calls + 1 DB query. Consider aggregating in IC Score pipeline and reading from DB. Target: <250ms.
- **Health summary endpoint:** Most complex — 4-5 data source queries. Must use `goroutines` + `errgroup` for parallel fetching. Target: <400ms.
- **Metric history endpoint:** Single DB query with JSONB extraction. Target: <50ms.

**Caching Strategy:**
- Sector percentile data changes daily → cache with 1-hour TTL
- Peer comparison changes daily → cache with 1-hour TTL
- Fair value changes daily (price-dependent) → cache with 15-min TTL
- Health summary depends on multiple sources → cache with 30-min TTL
- Metric history changes quarterly → cache with 24-hour TTL

## 8. Error Handling

Follow existing patterns from `ic_score_handlers.go`:

```go
// DB not initialized
if database.DB == nil {
    c.JSON(http.StatusServiceUnavailable, gin.H{
        "error":   "Database not available",
        "message": "[Feature] is temporarily unavailable",
    })
    return
}

// Ticker not found
if err == sql.ErrNoRows {
    c.JSON(http.StatusNotFound, gin.H{
        "error":   "[Data] not found",
        "message": fmt.Sprintf("No [data] available for %s", ticker),
        "ticker":  ticker,
    })
    return
}

// Graceful degradation for partial data
// If sector percentiles unavailable but other data exists, return what we have
```

## 9. Testing Strategy

- **Unit tests:** Mock database layer, test handler response shapes
- **Follow existing patterns:** See `backend/handlers/ic_score_handlers_test.go`, `backend/handlers/ic_score_mock_test.go`
- **Integration tests:** Test against test database with known data
- **Key test cases:**
  - Stock with full data → all fields populated
  - Stock with missing sector → graceful degradation
  - Stock with no IC Score → health summary still works (degraded)
  - Invalid ticker → 404
  - Metric history for unknown metric name → 400

## 10. Acceptance Criteria

- [ ] All 5 endpoints return correct data for stocks with complete data (AAPL, MSFT)
- [ ] All 5 endpoints handle missing data gracefully (no 500 errors)
- [ ] Health summary badge algorithm matches specification
- [ ] Red flag detection triggers correctly for known test cases
- [ ] Sector percentile calculation matches existing `CalculatePercentile()` output
- [ ] Frontend route definitions added to `lib/api/routes.ts`
- [ ] All endpoints respond in <500ms for typical stocks
- [ ] Unit tests covering happy path and edge cases for each handler
