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

### Authentication Tiers

Endpoints are split across two authentication levels to balance discoverability (SEO, free-tier value) with premium gating:

| Endpoint | Auth Level | Rationale |
|---|---|---|
| `sector-percentiles` | **Optional auth** (`OptionalAuthMiddleware`) | Free tier gets 6 core metrics; premium gets all. Auth needed to determine tier. Unauthenticated users get the free set. |
| `health-summary` | **Optional auth** | Free tier gets badge + lifecycle + 2 strengths/concerns + high-severity flags. Premium gets full details. Publicly visible for SEO. |
| `peers` | **Auth required** (`AuthMiddleware`) | Premium-only feature. Requires user context for subscription check. |
| `fair-value` | **Auth required** | Premium-only feature. Contains investment-grade analysis that drives conversion. |
| `metric-history` | **Auth required** | Premium-only feature. Powers sparklines and history charts. |

**Why not all public?** Three reasons: (1) Premium endpoints are the core conversion drivers — gating them behind auth lets us enforce subscription limits server-side rather than relying on client-side checks that can be bypassed. (2) The `peers` and `fair-value` endpoints make external API calls (FMP) per request — auth prevents anonymous abuse that would burn through our API quota. (3) User context enables server-side feature flag evaluation, analytics attribution, and A/B testing.

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

**Purpose:** Return top 5 industry peers with comparison metrics for side-by-side display.

**Handler:** `handlers.GetStockPeers`

**Query Params:**
- `limit` (optional, default 5, max 10)

#### Peer Selection Algorithm — Industry-Standard Approach

> **Review note:** The previous approach used our proprietary 5-factor IC Score similarity algorithm (`stock_peers` table). Per feedback, we should follow the industry standard used by Yahoo Finance, Bloomberg, Finviz, and Morningstar.

**Industry-standard peer selection works in 3 tiers:**

1. **Primary: Sub-industry classification (GICS / SIC codes)**
   - Match stocks within the same GICS sub-industry (e.g., "Semiconductors" not just "Technology")
   - This is how Bloomberg and FactSet define peer groups
   - Data source: FMP's `stock_peers` endpoint or our `tickers.sub_industry` column

2. **Secondary: Market cap proximity filter**
   - Filter to companies between 0.25x and 4x the target stock's market cap
   - Prevents comparing Apple ($2.8T) with a $500M small-cap in the same sub-industry
   - This is how Morningstar and S&P Capital IQ narrow peer groups

3. **Tertiary: Sort by market cap proximity**
   - Among remaining candidates, rank by closeness in market cap to the target
   - Return the top N (default 5)

**Implementation:**

```go
// 1. Get the stock's sub-industry and market cap
stock, err := database.GetStockBySymbol(ticker)
subIndustry := stock.SubIndustry  // e.g., "Consumer Electronics"
marketCap := stock.MarketCap

// 2. Query stocks in the same sub-industry within market cap range
peers, err := database.GetIndustryPeers(subIndustry, marketCap, limit)
// SQL: SELECT * FROM tickers
//      WHERE sub_industry = $1
//        AND symbol != $2
//        AND market_cap BETWEEN $3 * 0.25 AND $3 * 4.0
//      ORDER BY ABS(market_cap - $3) ASC
//      LIMIT $4

// 3. Fallback: if sub-industry yields < 3 peers, widen to industry level
if len(peers) < 3 {
    peers, err = database.GetIndustryPeers(stock.Industry, marketCap, limit)
}

// 4. For each peer, fetch key comparison metrics
//    (P/E, ROE, Revenue Growth, Net Margin, D/E, Market Cap)

// 5. Optionally include IC Score if available (as supplementary data, not selection criteria)

// 6. Get stock's own metrics for comparison
// 7. Return enriched response
```

**Why not use IC Score similarity for peer selection?** IC Score is our proprietary quality ranking — two stocks can have similar IC Scores but be in completely different industries (e.g., a high-scoring REIT and a high-scoring tech company). Users expect peers to be **business competitors**, not quality-score neighbors. IC Scores can still be displayed alongside peers as supplementary data.

**FMP Peer Data as Alternative Source:** FMP provides a `GET /v4/stock_peers?symbol=AAPL` endpoint that returns industry peers. We can use this as a data source if our sub-industry classification coverage is incomplete, and cache the results in a `stock_industry_peers` table.

**Response Schema:**

```json
{
  "ticker": "AAPL",
  "ic_score": 78.5,
  "sub_industry": "Consumer Electronics",
  "peers": [
    {
      "ticker": "MSFT",
      "company_name": "Microsoft Corporation",
      "ic_score": 82.1,
      "sub_industry": "Systems Software",
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
    "peer_selection": "sub-industry (GICS) + market cap proximity",
    "peer_source": "internal_classification",
    "timestamp": "2026-02-28T12:00:00Z"
  }
}
```

**New DB Functions Required:**
- `database.GetIndustryPeers(subIndustry, marketCap, limit)` — query `tickers` by sub-industry + market cap range
- May need a new `stock_industry_peers` cache table if using FMP's peer endpoint

**Migration:** The existing `stock_peers` table (IC Score similarity) remains for IC Score pipeline use. The new peer endpoint uses industry classification instead.

**New Code:** ~150 LOC handler + ~60 LOC DB queries

---

### 3.3 `GET /api/v1/stocks/:ticker/fair-value`

**Purpose:** Return computed fair value estimates (DCF, Graham Number, EPV) with margin of safety calculation.

**Handler:** `handlers.GetFairValue`

#### Valuation Model Algorithms

Each model uses a different approach to estimate intrinsic value. These are well-established academic/industry methods — not proprietary formulas — used by analysts at Goldman Sachs, Morningstar, and similar firms.

**1. Discounted Cash Flow (DCF) — "What are the future cash flows worth today?"**
- **Formula:** Fair Value = Σ(FCFₜ / (1 + WACC)ᵗ) + Terminal Value / (1 + WACC)ⁿ
- **Inputs:** Free Cash Flow TTM, WACC (weighted average cost of capital), terminal growth rate (typically 2-3%), projection period (10 years)
- **Source:** Our IC Score pipeline computes this in `fair_value_calculator.py` and stores in `fundamental_metrics_extended.dcf_fair_value`
- **Accuracy:** Most sensitive to WACC and terminal growth assumptions. ±1% change in WACC can swing fair value 20-30%. Best for mature, cash-flow-positive companies. **Not reliable for pre-profit or hypergrowth companies** (suppressed in those cases).
- **Confidence:** High for stable FCF companies (utilities, consumer staples), Medium for cyclicals, Low/suppressed for pre-profit

**2. Graham Number — "What would Benjamin Graham pay?"**
- **Formula:** √(22.5 × EPS_TTM × Book_Value_Per_Share)
- **Inputs:** Trailing 12-month EPS, Book Value Per Share (from balance sheet)
- **Source:** FMP API `ratios-ttm` endpoint provides this directly as `grahamNumberTTM`
- **Accuracy:** Conservative by design — Graham intended this as a ceiling price for defensive investors. Tends to undervalue growth companies significantly (ignores future growth entirely). Works well for value/income stocks.
- **Confidence:** High (simple, few inputs), but systematically conservative

**3. Earnings Power Value (EPV) — "What is the company worth at current earnings, no growth?"**
- **Formula:** EPV = Adjusted_Earnings / WACC
- **Inputs:** Normalized operating earnings (adjusted for one-time items), WACC
- **Source:** IC Score pipeline computes this in `fair_value_calculator.py`, stored in `fundamental_metrics_extended.epv_fair_value`
- **Accuracy:** Assumes zero growth, which makes it a conservative baseline. The gap between EPV and DCF represents the market's implied growth premium. Useful for identifying how much of the price is "paying for growth."
- **Confidence:** Medium (depends on earnings normalization quality)

**Important caveats surfaced to the user:**
- Fair value models are **estimates, not price targets**. They provide directional signal, not precision.
- Models disagree with each other by design — they capture different aspects of value.
- The `margin_of_safety.zone` field is directional: undervalued/fairly_valued/overvalued within ±15% bands.
- Models are **suppressed** (not shown) for companies where inputs are unreliable: pre-revenue, negative earnings, negative book value.

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
// 6. Determine suppression: suppress if EPS < 0, BVPS < 0, or FCF < 0 for 4+ Qs
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

#### What Does the Health Summary Indicate?

The health summary answers a single question: **"Is this company financially stable and well-managed?"**

It is **not** a buy/sell signal or a price prediction. Instead, it assesses the underlying financial quality of the business across four dimensions:

| Dimension | Signal | What It Tells the User |
|---|---|---|
| **Piotroski F-Score** (0-9) | Accounting-based quality | Are profitability, leverage, and operating efficiency improving or deteriorating? A score of 7+ means fundamentals are strengthening. |
| **Altman Z-Score** | Bankruptcy risk | How likely is this company to face financial distress in the next 2 years? >2.99 = safe, <1.81 = distress zone. |
| **IC Financial Health Score** (0-100) | Composite balance sheet quality | How strong is the balance sheet relative to sector peers? Considers current ratio, D/E, interest coverage, cash flow. |
| **Debt Percentile** | Leverage vs. sector | How leveraged is this company compared to its sector? High leverage isn't always bad (REITs, utilities) — context matters. |

**The badge (Strong/Healthy/Fair/Weak/Distressed)** is a composite of these four signals designed to be immediately actionable:
- **Strong/Healthy:** User can focus on valuation and growth — financial quality is not a concern
- **Fair:** Some areas need attention — user should look at specific concerns
- **Weak/Distressed:** Financial quality is a risk — user should investigate red flags before investing

**Strengths and concerns** are auto-generated from sector percentile rankings. A "strength" is any metric where the stock ranks in the top quartile (≥75th percentile) of its sector. A "concern" is bottom quartile (≤25th percentile).

**Red flags** are rule-based alerts for specific dangerous patterns (unsustainable dividends, cash burn, earnings quality issues) that percentile bars alone wouldn't surface.

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

**Health Badge Algorithm — Mathematical vs. Agent Approach:**

> **Design decision:** Should the health badge and narrative be computed mathematically (deterministic formula) or generated by an LLM agent that interprets the raw data?

| Criteria | Mathematical (Deterministic) | LLM Agent |
|---|---|---|
| **Consistency** | Same inputs always → same output. Users see stable badge across refreshes. | May vary between calls. "Strong" could become "Healthy" on retry. |
| **Latency** | <5ms computation | 500-2000ms per LLM call |
| **Cost** | Zero marginal cost | ~$0.002-0.01 per call (at scale: significant) |
| **Auditability** | Formula is transparent, testable, debuggable | Black box — hard to explain why badge changed |
| **Nuance** | Rigid — can't weigh context like "REITs typically have high D/E" | Can incorporate qualitative reasoning and sector-specific context |
| **Narrative quality** | Template-based: "Net margin ranks in top 10% of Technology sector" | Natural language: "Apple's profitability stands out even among tech giants, with margins that suggest strong pricing power" |

**Recommendation: Hybrid approach — mathematical badge + agent-generated narrative (Phase 2)**

- **Phase 1 (this project):** Use the deterministic formula below for the badge, score, strengths, concerns, and red flags. This gives us a fast, testable, consistent foundation.
- **Phase 2 (future):** Add an optional `narrative` field generated by an LLM agent that produces a 2-3 sentence natural language summary (e.g., "Apple is in strong financial health with industry-leading margins, though its leverage is worth monitoring..."). This would be cached (regenerated daily, not per-request) to control cost and latency. The badge remains mathematical — the agent only produces the prose.

**Phase 1 Deterministic Algorithm:**

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

**Phase 2 Agent Narrative (future — not in this project's scope):**

```go
// Cached daily per ticker, not computed per-request
type HealthNarrative struct {
    Ticker     string    `json:"ticker"`
    Summary    string    `json:"summary"`     // 2-3 sentence LLM-generated prose
    GeneratedAt time.Time `json:"generated_at"`
}

// Response would include:
// "narrative": "Apple is in strong financial health with industry-leading margins..."
// Only regenerated when underlying data changes (daily IC Score pipeline run)
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

Add to `backend/main.go`. Split across two groups based on auth requirements:

```go
stocks := v1.Group("/stocks")
{
    // ... existing routes ...

    // Fundamentals: public endpoints (optional auth for tier detection)
    fundamentalsHandler := handlers.NewFundamentalsHandler()
    stocks.GET("/:ticker/sector-percentiles", auth.OptionalAuthMiddleware(), fundamentalsHandler.GetSectorPercentiles)
    stocks.GET("/:ticker/health-summary", auth.OptionalAuthMiddleware(), fundamentalsHandler.GetHealthSummary)
}

// Fundamentals: premium endpoints (auth required)
fundamentalsPremium := v1.Group("/stocks")
fundamentalsPremium.Use(auth.AuthMiddleware())
{
    fundamentalsHandler := handlers.NewFundamentalsHandler()
    fundamentalsPremium.GET("/:ticker/peers", fundamentalsHandler.GetStockPeers)
    fundamentalsPremium.GET("/:ticker/fair-value", fundamentalsHandler.GetFairValue)
    fundamentalsPremium.GET("/:ticker/metric-history/:metric", fundamentalsHandler.GetMetricHistory)
}
```

**Note:** `OptionalAuthMiddleware` extracts user info from the JWT if present but does not reject unauthenticated requests. The handler checks `c.GetString("user_id")` to determine tier. If empty → free tier defaults.

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
