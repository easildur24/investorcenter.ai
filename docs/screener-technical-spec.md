# InvestorCenter Stock Screener: Technical Specification

**Author:** Engineering Architecture Review
**Date:** February 2026
**Status:** Draft — Pending Review
**Companion Document:** [Product Improvement Plan](./screener-product-plan.md)

---

## Table of Contents

1. [Codebase Audit & Architecture Assessment](#1-codebase-audit--architecture-assessment)
2. [Architecture Decision: Client-Side vs. Server-Side Filtering](#2-architecture-decision-client-side-vs-server-side-filtering)
3. [URL State Management Spec](#3-url-state-management-spec)
4. [New Filters — Data & Component Spec](#4-new-filters--data--component-spec)
5. [Table Enhancements Spec](#5-table-enhancements-spec)
6. [Saved Screeners & Presets System](#6-saved-screeners--presets-system)
7. [Watchlist & Alerts Integration](#7-watchlist--alerts-integration)
8. [Performance Budget & Optimization Strategy](#8-performance-budget--optimization-strategy)
9. [Component Architecture](#9-component-architecture)
10. [Migration & Rollout Plan](#10-migration--rollout-plan)

---

## 1. Codebase Audit & Architecture Assessment

### 1.1 Project Directory Structure

```
investorcenter.ai/
├── app/
│   ├── layout.tsx                          # Root layout: ThemeProvider → AuthProvider → ToastProvider
│   ├── middleware.ts                        # Cache-busting for /ticker/* routes
│   ├── screener/
│   │   └── page.tsx                        # Basic screener (599 lines, client-side filtering)
│   ├── ic-score/
│   │   └── page.tsx                        # IC Score screener (server-side filtering)
│   └── ticker/[symbol]/
│       └── page.tsx                        # Ticker detail page (server-side data fetch)
├── components/
│   ├── ic-score/
│   │   └── ICScoreScreener.tsx             # IC Score screener component (482 lines)
│   ├── ticker/
│   │   ├── RealTimePriceHeader.tsx
│   │   ├── HybridChart.tsx
│   │   ├── TickerFundamentals.tsx
│   │   └── TickerTabs.tsx
│   └── Header.tsx
├── lib/
│   ├── api.ts                              # Main API client (351 lines) — apiClient + icScoreApi
│   ├── api/
│   │   ├── client.ts                       # Authenticated client with token refresh
│   │   ├── ic-score.ts                     # IC Score modular client
│   │   ├── financials.ts                   # Financials modular client
│   │   ├── metrics.ts                      # Metrics modular client
│   │   └── sentiment.ts                    # Sentiment modular client
│   ├── types/
│   │   ├── ic-score.ts                     # IC Score type definitions (223 lines)
│   │   └── sentiment.ts
│   ├── hooks/
│   │   ├── useRealTimePrice.ts             # Real-time price polling hook
│   │   └── useApiWithRetry.ts              # Retry with exponential backoff
│   ├── auth/
│   │   └── AuthContext.tsx                  # JWT auth + auto-refresh
│   ├── contexts/
│   │   └── ThemeContext.tsx                 # Light/dark/system theme
│   └── utils.ts                            # Formatters: cn, formatNumber, formatCurrency, etc.
├── backend/
│   ├── main.go                             # Route definitions (screener at line 189)
│   ├── handlers/
│   │   └── screener.go                     # GET /api/v1/screener/stocks handler (160 lines)
│   ├── database/
│   │   └── screener.go                     # SQL query builder (185 lines)
│   ├── models/
│   │   └── stock.go                        # ScreenerStock, ScreenerParams structs (lines 260-304)
│   └── migrations/
├── ic-score-service/
│   ├── models.py                           # SQLAlchemy models (all tables)
│   ├── database/
│   │   └── schema.sql                      # Full SQL schema
│   ├── migrations/
│   │   └── 016_create_screener_materialized_view.sql  # screener_data view
│   └── pipelines/                          # 16 data pipelines
└── package.json                            # Dependencies (Next.js 14, React 18, Tailwind, D3, Recharts)
```

### 1.2 How Data Flows to the Screener

**Two completely separate screener implementations exist:**

#### A) Basic Screener (`/screener` → `app/screener/page.tsx`)
```
Page Load
  → fetch('/api/v1/screener/stocks')  // No filter params, limit=20000
  → Go handler parses params, default limit=20000
  → SQL: SELECT ... FROM screener_data ORDER BY market_cap DESC LIMIT 20000
  → Returns ~5,000 rows as JSON (all CS stocks)
  → Client stores in state: stocks[] + filteredStocks[]
  → On filter change: client-side Array.filter() + Array.sort()
  → On sort change: client-side Array.sort()
  → Client-side pagination: .slice((page-1)*25, page*25)
```

**Key issue:** 20,000-row limit, all data loaded upfront, filtered in browser. Works today because only ~5,000 CS stocks have data, but the approach won't scale with more filter dimensions.

#### B) IC Score Screener (`/ic-score` → `components/ic-score/ICScoreScreener.tsx`)
```
Page Load
  → icScoreApi.runScreener(filters)
  → GET {IC_SCORE_API_BASE}/api/scores/screener?limit=50&offset=0&sortBy=score&sortOrder=desc
  → Python/FastAPI microservice queries ic_scores + joins
  → Returns 50 rows + total count
  → On filter change: new API call with updated params
  → Server-side filtering, sorting, and pagination
```

**Key difference:** Server-side filtering is the right pattern. Each filter change triggers a new API call with all filter params.

### 1.3 Current Data Model (screener_data Materialized View)

**Source:** `ic-score-service/migrations/016_create_screener_materialized_view.sql`

The view currently selects these columns:

| Column | Source Table | Source Column | Exposed in API? | Bug? |
|---|---|---|---|---|
| `symbol` | tickers | symbol | Yes | — |
| `name` | tickers | name | Yes | — |
| `sector` | tickers | sector | Yes (filter + display) | — |
| `industry` | tickers | industry | **No** (in view, not exposed) | — |
| `market_cap` | tickers | market_cap | Yes (filter + sort + display) | — |
| `asset_type` | tickers | asset_type | No (hardcoded WHERE CS) | — |
| `active` | tickers | active | No (hardcoded WHERE true) | — |
| `price` | stock_prices | close (latest) | Yes (display) | Aliased as `price` in view but query uses `current_price`* |
| `pe_ratio` | valuation_ratios | ttm_pe_ratio | Yes (filter + sort + display) | — |
| `pb_ratio` | valuation_ratios | ttm_pb_ratio | In API response, **no frontend filter** | — |
| `ps_ratio` | valuation_ratios | ttm_ps_ratio | In API response, **no frontend filter** | — |
| `revenue_growth` | fundamental_metrics_extended | revenue_growth_yoy | Yes (filter + display) | — |
| `dividend_yield` | fundamental_metrics_extended | dividend_yield | Yes (filter + display) | — |
| `roe` | fundamental_metrics_extended | roe | In API response, **no frontend filter** | — |
| `beta` | fundamental_metrics_extended | beta | **Hardcoded to 0.0** | **BUG** |
| `ic_score` | ic_scores | overall_score | **Hardcoded to 0.0** | **BUG** |

*The `price` column alias mismatch: the view selects `lp.close as price` but the Go query references `current_price as price`. This likely works because there's no column named `current_price` — it just reads `price` and aliases it. Need to verify.

### 1.4 The Bug: IC Score and Beta Hardcoded to Zero

**File:** `backend/database/screener.go`, lines 166-167

```go
// Current (broken):
SELECT
    ...
    0.0 as beta,
    0.0 as ic_score
FROM screener_data
```

The view has `beta` and `ic_score` columns, but the Go query ignores them and returns literal `0.0`. The sort column map compounds the issue:

```go
// Line 23-24 — placeholders that map to wrong columns:
"beta":           "roe",        // should be "beta"
"ic_score":       "market_cap", // should be "ic_score"
```

### 1.5 State Management

**Pattern:** Pure React hooks (`useState`/`useCallback`). No external state library (no Zustand, Redux, Jotai). No URL param sync.

**Basic screener state:**
```typescript
const [stocks, setStocks] = useState<Stock[]>([]);           // Full dataset
const [filteredStocks, setFilteredStocks] = useState<Stock[]>([]); // After client-side filter
const [loading, setLoading] = useState(true);
const [filters, setFilters] = useState<FilterState>({});      // Untyped: { [key: string]: any }
const [sortField, setSortField] = useState<SortField>('market_cap');
const [sortDirection, setSortDirection] = useState<SortDirection>('desc');
const [showFilters, setShowFilters] = useState(true);
const [currentPage, setCurrentPage] = useState(1);
```

### 1.6 Database Technology

- **PostgreSQL** with **TimescaleDB** extension for time-series hypertables
- **Go backend** uses `jmoiron/sqlx` with `lib/pq` driver
- **IC Score service** uses SQLAlchemy async with `asyncpg`
- **Redis** for crypto price caching only
- Both services share the same PostgreSQL database

### 1.7 Ticker Detail Page Data Flow (for Comparison)

The ticker detail page fetches from 5+ endpoints and merges data from 3 priority sources:
1. **Manual Fundamentals** (highest) — `/api/v1/tickers/:symbol/keystats`
2. **IC Score Financials** (medium) — `/api/v1/stocks/:symbol/financials`
3. **Polygon.io** (lowest) — `/api/v1/tickers/:symbol`

The detail page has access to P/B, P/S, ROE, ROA, margins, D/E, current ratio, beta, earnings growth, and more. All of this data exists in the database — it's just not wired to the screener.

---

## 2. Architecture Decision: Client-Side vs. Server-Side Filtering

### 2.1 Current Payload Analysis

The basic screener fetches all ~5,000 CS stocks in one request. With the current 15 columns, the JSON payload is approximately:

```
~5,000 rows × ~15 fields × ~30 bytes avg per field = ~2.25 MB uncompressed
With gzip: ~400-600 KB
```

If we expand to 40+ columns (all metrics), the payload grows to:
```
~5,000 rows × ~40 fields × ~30 bytes = ~6 MB uncompressed
With gzip: ~1-1.5 MB
```

### 2.2 Analysis of Approaches

| Factor | Client-Side (Current) | Server-Side (Recommended) | Hybrid |
|---|---|---|---|
| Initial load time | ~1-2s (400KB gzip) | ~200ms (50 rows) | ~1-2s |
| Filter response time | Instant (<16ms) | 100-300ms per change | Mix |
| Memory usage | ~10-15 MB (5K objects) | Minimal | ~10-15 MB |
| Scalability (20K+ stocks) | Degrades significantly | Constant | Degrades |
| URL sharing | Easy (filters are local) | Natural (params in API) | Complex |
| Offline filtering | Yes | No | Partial |
| Mobile performance | Poor on low-end devices | Good | Poor |
| Backend complexity | Low | Medium | High |
| Adding new filters | Frontend-only change | Frontend + backend | Both |

### 2.3 Recommendation: Server-Side Filtering

**Move to server-side filtering, modeled after the IC Score screener pattern.**

Rationale:
1. **The basic screener already sends filter params to the API** — they're just ignored because limit=20000 fetches everything. The API already supports `pe_min`, `pe_max`, `sectors`, etc.
2. **Adding 30+ columns to a 5K-row client-side dataset** creates memory pressure on mobile and a slow initial load.
3. **The IC Score screener already does server-side filtering correctly** via the Python microservice. The Go backend just needs the same treatment.
4. **Materialized view makes server-side fast** — sub-100ms queries with indexes.
5. **URL persistence maps naturally** to API query params.

**The tradeoff — filter response latency — is mitigated** by:
- Debouncing filter changes (150ms) before sending API requests
- Showing a subtle loading indicator (skeleton rows, not full-page spinner)
- Optimistic UI: keep current results visible while new results load
- Response caching: if user toggles a filter off and on, cache the previous response

### 2.4 Proposed API Contract

**Endpoint:** `GET /api/v1/screener/stocks`

```
# Pagination
?page=1&limit=50

# Sorting (Phase 1: single, Phase 2: multi)
?sort=ic_score&order=desc
?sort=ic_score,dividend_yield&order=desc,desc    # Phase 2

# Categorical filters
?sectors=Technology,Healthcare
?industries=Biotechnology,Software              # Phase 1 (dependent on sector)
?mcap=large,mega                                # Enum: mega,large,mid,small,micro

# Range filters (pattern: {metric}_min, {metric}_max)
?pe_min=5&pe_max=20
?pb_min=0&pb_max=3
?ps_min=0&ps_max=10
?roe_min=15
?roa_min=10
?gross_margin_min=40
?net_margin_min=10
?de_min=0&de_max=1.5                            # debt/equity
?current_ratio_min=1.5
?div_yield_min=2&div_yield_max=8
?payout_ratio_max=75
?consec_div_years_min=10
?rev_growth_min=20
?eps_growth_min=15
?beta_min=0.5&beta_max=1.5
?ic_score_min=60&ic_score_max=100

# IC Score sub-factor filters (Phase 2)
?value_score_min=60
?growth_score_min=50
?insider_score_min=70
?sentiment_score_min=60

# Fair value filters (Phase 2)
?dcf_upside_min=20                              # Stocks >20% below DCF fair value

# Technical filters (Phase 2)
?rsi_max=30                                     # Oversold stocks
?above_sma50=true                               # Trading above 50-day MA
?above_sma200=true

# Preset shorthand
?preset=value                                    # Expands to underlying filters

# Asset type
?asset_type=CS                                   # Default: CS (common stock)
```

**Response:**
```typescript
interface ScreenerResponse {
  data: ScreenerStock[];
  meta: {
    total: number;
    page: number;
    limit: number;
    total_pages: number;
    timestamp: string;
    active_filters: number;  // NEW: count of active filters
  };
}

interface ScreenerStock {
  symbol: string;
  name: string;
  sector: string | null;
  industry: string | null;
  market_cap: number | null;
  price: number | null;
  change_percent: number | null;

  // Valuation
  pe_ratio: number | null;
  pb_ratio: number | null;
  ps_ratio: number | null;

  // Profitability
  roe: number | null;
  roa: number | null;
  gross_margin: number | null;
  net_margin: number | null;

  // Financial Health
  debt_to_equity: number | null;
  current_ratio: number | null;

  // Growth
  revenue_growth: number | null;
  eps_growth_yoy: number | null;

  // Dividends
  dividend_yield: number | null;
  payout_ratio: number | null;
  consecutive_dividend_years: number | null;

  // Risk
  beta: number | null;

  // Score
  ic_score: number | null;
  ic_rating: string | null;           // Phase 2

  // IC Score sub-factors (Phase 2 — only included if requested)
  value_score: number | null;
  growth_score: number | null;
  profitability_score: number | null;
  financial_health_score: number | null;
  momentum_score: number | null;
  analyst_consensus_score: number | null;
  insider_activity_score: number | null;
  institutional_score: number | null;
  news_sentiment_score: number | null;
  technical_score: number | null;

  // Fair value (Phase 2)
  dcf_upside_percent: number | null;
}
```

---

## 3. URL State Management Spec

### 3.1 Library Choice: `nuqs` vs. Native `useSearchParams`

**Recommendation: Use `nuqs` (Next.js URL State).**

Rationale:
- Type-safe URL param parsing with built-in validators
- Handles serialization/deserialization automatically
- `shallow: true` option avoids server re-renders (critical for performance)
- Throttling built-in (prevents excessive URL updates)
- Handles history correctly (`replaceState` by default)
- 3KB gzipped, zero dependencies beyond Next.js
- Already battle-tested with Next.js App Router

**Installation:** `npm install nuqs`

### 3.2 Query Parameter Schema

```typescript
// lib/screener/url-params.ts
import { parseAsArrayOf, parseAsFloat, parseAsInteger, parseAsString, createSearchParamsCache } from 'nuqs/server';

export const screenerParamsParsers = {
  // Categorical
  sectors: parseAsArrayOf(parseAsString).withDefault([]),
  industries: parseAsArrayOf(parseAsString).withDefault([]),
  mcap: parseAsArrayOf(parseAsString).withDefault([]),     // 'mega','large','mid','small','micro'

  // Range: Valuation
  pe_min: parseAsFloat,
  pe_max: parseAsFloat,
  pb_min: parseAsFloat,
  pb_max: parseAsFloat,
  ps_min: parseAsFloat,
  ps_max: parseAsFloat,

  // Range: Profitability
  roe_min: parseAsFloat,
  roe_max: parseAsFloat,
  roa_min: parseAsFloat,
  roa_max: parseAsFloat,
  gross_margin_min: parseAsFloat,
  gross_margin_max: parseAsFloat,
  net_margin_min: parseAsFloat,
  net_margin_max: parseAsFloat,

  // Range: Financial Health
  de_min: parseAsFloat,
  de_max: parseAsFloat,
  current_ratio_min: parseAsFloat,
  current_ratio_max: parseAsFloat,

  // Range: Growth
  rev_growth_min: parseAsFloat,
  rev_growth_max: parseAsFloat,
  eps_growth_min: parseAsFloat,
  eps_growth_max: parseAsFloat,

  // Range: Dividends
  div_yield_min: parseAsFloat,
  div_yield_max: parseAsFloat,
  payout_ratio_min: parseAsFloat,
  payout_ratio_max: parseAsFloat,
  consec_div_years_min: parseAsInteger,

  // Range: Risk & Score
  beta_min: parseAsFloat,
  beta_max: parseAsFloat,
  ic_score_min: parseAsFloat,
  ic_score_max: parseAsFloat,

  // Sort & Pagination
  sort: parseAsString.withDefault('market_cap'),
  order: parseAsString.withDefault('desc'),
  page: parseAsInteger.withDefault(1),
  limit: parseAsInteger.withDefault(50),

  // Preset (mutually exclusive with individual filters)
  preset: parseAsString,
};

// Server-side cache for SSR
export const screenerParamsCache = createSearchParamsCache(screenerParamsParsers);
```

### 3.3 URL Encoding/Decoding Strategy

**Arrays:** Comma-separated values (no brackets)
```
?sectors=Technology,Healthcare,Energy
```

**Ranges:** Separate min/max params
```
?pe_min=5&pe_max=20
```

**Open-ended ranges:** Omit the unbounded side
```
?div_yield_min=3     # Yield >= 3%, no max
?pe_max=15           # P/E <= 15, no min
```

**Market cap tiers:** Named values (mapped to numeric ranges server-side)
```
?mcap=large,mega     # Maps to market_cap >= 10e9
```

**Booleans (Phase 2):**
```
?above_sma50=true    # Price > 50-day SMA
```

### 3.4 Bidirectional Sync Hook

```typescript
// lib/hooks/useScreenerParams.ts
'use client';

import { useQueryStates } from 'nuqs';
import { screenerParamsParsers } from '@/lib/screener/url-params';

export function useScreenerParams() {
  const [params, setParams] = useQueryStates(screenerParamsParsers, {
    shallow: true,         // Don't trigger server re-render
    throttleMs: 150,       // Debounce URL updates
    history: 'replace',    // Don't pollute browser history
  });

  // Helper: apply a preset (clears existing filters, sets preset params)
  const applyPreset = (presetId: string) => {
    const presetFilters = PRESET_DEFINITIONS[presetId];
    if (presetFilters) {
      // Clear all params, then set preset filters
      setParams({ ...EMPTY_PARAMS, ...presetFilters, preset: presetId });
    }
  };

  // Helper: clear all filters
  const clearAll = () => {
    setParams(EMPTY_PARAMS);
  };

  // Helper: count active filters
  const activeFilterCount = Object.entries(params).filter(([key, value]) => {
    if (['sort', 'order', 'page', 'limit', 'preset'].includes(key)) return false;
    if (value === null || value === undefined) return false;
    if (Array.isArray(value) && value.length === 0) return false;
    return true;
  }).length;

  return { params, setParams, applyPreset, clearAll, activeFilterCount };
}
```

### 3.5 Quick Screen Presets via URL

```typescript
// lib/screener/presets.ts
export const PRESET_DEFINITIONS: Record<string, Partial<ScreenerParams>> = {
  value: {
    pe_max: 15,
    pb_max: 2,
    div_yield_min: 2,
    de_max: 1.5,
    sort: 'ic_score',
    order: 'desc',
  },
  growth: {
    rev_growth_min: 20,
    eps_growth_min: 15,
    gross_margin_min: 40,
    sort: 'rev_growth',
    order: 'desc',
  },
  quality: {
    ic_score_min: 70,
    roe_min: 15,
    net_margin_min: 10,
    current_ratio_min: 1.5,
    sort: 'ic_score',
    order: 'desc',
  },
  dividend: {
    div_yield_min: 3,
    consec_div_years_min: 10,
    payout_ratio_max: 75,
    de_max: 1,
    sort: 'div_yield',
    order: 'desc',
  },
  undervalued: {
    pe_max: 15,
    pb_max: 1.5,
    roe_min: 12,
    ic_score_min: 50,
    sort: 'ic_score',
    order: 'desc',
  },
};
```

A URL like `/screener?preset=value` expands to the individual filter params on load. If the user then modifies a filter, the `preset` param is removed and the individual params are shown.

### 3.6 Browser Navigation

- **Back/forward:** `nuqs` handles this natively when using `history: 'push'` for preset changes and `history: 'replace'` for filter tweaks.
- **Strategy:** Use `push` only when applying a preset or clearing all. Use `replace` for individual filter changes. This means Back goes to the previous preset/clear action, not every keystroke.

### 3.7 Shareable URLs

Example shareable URLs:
```
/screener?sectors=Technology&pe_max=25&roe_min=15&sort=ic_score&order=desc
/screener?preset=dividend
/screener?div_yield_min=4&de_max=0.5&consec_div_years_min=20&sort=div_yield&order=desc
```

**"Copy Link" button:** Copies `window.location.href` to clipboard. Shows toast: "Link copied! Share this screen with anyone."

---

## 4. New Filters — Data & Component Spec

### 4.1 Filter Registry (Source of Truth)

Both the Go backend and Next.js frontend share a conceptual filter registry. The Go side maps param names to SQL columns; the frontend side defines UI components and groupings.

#### Go Backend Filter Registry

```go
// backend/database/filter_registry.go

type FilterType int

const (
    RangeFilter FilterType = iota
    MultiSelectFilter
    BooleanFilter
)

type FilterDef struct {
    ParamMin   string     // URL param for min (e.g., "pe_min")
    ParamMax   string     // URL param for max (e.g., "pe_max")
    Column     string     // SQL column in screener_data
    Type       FilterType
    NullPolicy string     // "exclude" (default) or "include"
}

var FilterRegistry = map[string]FilterDef{
    // Valuation
    "pe":             {ParamMin: "pe_min", ParamMax: "pe_max", Column: "pe_ratio", Type: RangeFilter},
    "pb":             {ParamMin: "pb_min", ParamMax: "pb_max", Column: "pb_ratio", Type: RangeFilter},
    "ps":             {ParamMin: "ps_min", ParamMax: "ps_max", Column: "ps_ratio", Type: RangeFilter},

    // Profitability
    "roe":            {ParamMin: "roe_min", ParamMax: "roe_max", Column: "roe", Type: RangeFilter},
    "roa":            {ParamMin: "roa_min", ParamMax: "roa_max", Column: "roa", Type: RangeFilter},
    "gross_margin":   {ParamMin: "gross_margin_min", ParamMax: "gross_margin_max", Column: "gross_margin", Type: RangeFilter},
    "net_margin":     {ParamMin: "net_margin_min", ParamMax: "net_margin_max", Column: "net_margin", Type: RangeFilter},

    // Financial Health
    "de":             {ParamMin: "de_min", ParamMax: "de_max", Column: "debt_to_equity", Type: RangeFilter},
    "current_ratio":  {ParamMin: "current_ratio_min", ParamMax: "current_ratio_max", Column: "current_ratio", Type: RangeFilter},

    // Growth
    "rev_growth":     {ParamMin: "rev_growth_min", ParamMax: "rev_growth_max", Column: "revenue_growth", Type: RangeFilter},
    "eps_growth":     {ParamMin: "eps_growth_min", ParamMax: "eps_growth_max", Column: "eps_growth_yoy", Type: RangeFilter},

    // Dividends
    "div_yield":      {ParamMin: "div_yield_min", ParamMax: "div_yield_max", Column: "dividend_yield", Type: RangeFilter},
    "payout_ratio":   {ParamMin: "payout_ratio_min", ParamMax: "payout_ratio_max", Column: "payout_ratio", Type: RangeFilter},
    "consec_div":     {ParamMin: "consec_div_years_min", ParamMax: "", Column: "consecutive_dividend_years", Type: RangeFilter},

    // Risk & Score
    "beta":           {ParamMin: "beta_min", ParamMax: "beta_max", Column: "beta", Type: RangeFilter},
    "ic_score":       {ParamMin: "ic_score_min", ParamMax: "ic_score_max", Column: "ic_score", Type: RangeFilter},

    // Categorical
    "sectors":        {Column: "sector", Type: MultiSelectFilter},
    "industries":     {Column: "industry", Type: MultiSelectFilter},
}
```

The `GetScreenerStocks` function iterates the registry to build WHERE clauses, replacing the current hardcoded approach.

#### Frontend Filter Configuration

```typescript
// lib/screener/filter-config.ts

export type FilterUIType = 'range' | 'multiselect' | 'checkbox-group' | 'boolean-toggle';

export interface FilterConfig {
  id: string;                    // Unique key (matches registry)
  label: string;                 // Display name
  section: FilterSection;        // Which collapsible group
  type: FilterUIType;
  paramMin?: string;             // URL param name for min
  paramMax?: string;             // URL param name for max
  paramKey?: string;             // URL param name for categorical
  step?: number;                 // Step increment for range inputs
  suffix?: string;               // Display suffix (e.g., "%", "x")
  tooltip?: string;              // Help text
  min?: number;                  // Suggested min bound for UI
  max?: number;                  // Suggested max bound for UI
  options?: { value: string; label: string }[];  // For multiselect
  phase: 1 | 2 | 3;             // Implementation phase
}

export type FilterSection =
  | 'classification'   // Sector, Industry, Market Cap
  | 'valuation'        // P/E, P/B, P/S, EV/EBITDA
  | 'profitability'    // ROE, ROA, Margins
  | 'financial_health' // D/E, Current Ratio, Quick Ratio
  | 'growth'           // Rev Growth, EPS Growth
  | 'dividends'        // Yield, Payout, Consecutive Years
  | 'risk'             // Beta, Sharpe, Max Drawdown
  | 'score'            // IC Score, IC Sub-factors
  | 'technical'        // RSI, SMAs, MACD
  | 'sentiment';       // News Sentiment, Reddit

export const FILTER_SECTIONS: { id: FilterSection; label: string; defaultOpen: boolean }[] = [
  { id: 'classification', label: 'Classification', defaultOpen: true },
  { id: 'score', label: 'IC Score', defaultOpen: true },
  { id: 'valuation', label: 'Valuation', defaultOpen: false },
  { id: 'profitability', label: 'Profitability', defaultOpen: false },
  { id: 'financial_health', label: 'Financial Health', defaultOpen: false },
  { id: 'growth', label: 'Growth', defaultOpen: false },
  { id: 'dividends', label: 'Dividends', defaultOpen: false },
  { id: 'risk', label: 'Risk', defaultOpen: false },
  { id: 'technical', label: 'Technical', defaultOpen: false },   // Phase 2
  { id: 'sentiment', label: 'Sentiment', defaultOpen: false },   // Phase 2
];
```

### 4.2 Phase 1 Filters — Detailed Spec

For each filter, here is the data source, component type, and null handling:

#### Classification Filters (Already Exist — Enhance)

| Filter | Column | Component | Null Handling | Notes |
|---|---|---|---|---|
| Sector | `sector` | Checkbox group (11 options) | Empty string = "Unknown" | Already exists; change from hardcoded checkboxes to dynamic from API |
| Industry | `industry` | Checkbox group (dependent on sector) | Empty string = "Unknown" | **NEW:** Show industries only for selected sectors |
| Market Cap | derived from `market_cap` | Checkbox group (5 tiers) | Null = exclude | Already exists; keep tier-based approach |

**Industry filter implementation:**
- Frontend: When sector(s) selected, fetch available industries via `GET /api/v1/screener/industries?sectors=Technology,Healthcare`
- Backend: Simple `SELECT DISTINCT industry FROM screener_data WHERE sector IN (...) ORDER BY industry`
- Cache the industry list client-side per sector combination

#### Valuation Filters

| Filter | View Column | Add to View? | Component | Step | Suffix | Min/Max Hint |
|---|---|---|---|---|---|---|
| P/E Ratio | `pe_ratio` | No (already exists) | Range (min/max) | 1 | — | 0 – 200 |
| P/B Ratio | `pb_ratio` | No (already exists) | Range (min/max) | 0.1 | x | 0 – 50 |
| P/S Ratio | `ps_ratio` | No (already exists) | Range (min/max) | 0.1 | x | 0 – 50 |

**Null handling:** Stocks with null valuation ratios (pre-revenue companies, negative earnings) are excluded when a valuation filter is active. The SQL should include `AND column IS NOT NULL` when filtering.

#### Profitability Filters

| Filter | View Column | Add to View? | Component | Step | Suffix | Min/Max Hint |
|---|---|---|---|---|---|---|
| ROE | `roe` | No (already exists) | Range | 1 | % | -50 – 100 |
| ROA | `roa` | **Yes** (from `fundamental_metrics_extended.roa`) | Range | 1 | % | -30 – 50 |
| Gross Margin | `gross_margin` | **Yes** (from `fundamental_metrics_extended.gross_margin`) | Range | 1 | % | -20 – 100 |
| Net Margin | `net_margin` | **Yes** (from `fundamental_metrics_extended.net_margin`) | Range | 1 | % | -100 – 60 |

**Negative margins:** Allow negative min values. Net margin can be deeply negative for growth companies. Display with +/- formatting.

#### Financial Health Filters

| Filter | View Column | Add to View? | Component | Step | Suffix | Min/Max Hint |
|---|---|---|---|---|---|---|
| Debt/Equity | `debt_to_equity` | **Yes** (from `fundamental_metrics_extended.debt_to_equity`) | Range | 0.1 | x | 0 – 10 |
| Current Ratio | `current_ratio` | **Yes** (from `fundamental_metrics_extended.current_ratio`) | Range | 0.1 | x | 0 – 10 |

**Sector context warning:** Financial companies (banks, insurance) naturally have high D/E ratios. Consider adding a tooltip: "Note: Financial sector companies typically have high leverage ratios by design."

#### Growth Filters

| Filter | View Column | Add to View? | Component | Step | Suffix | Min/Max Hint |
|---|---|---|---|---|---|---|
| Revenue Growth YoY | `revenue_growth` | No (already exists) | Range | 5 | % | -50 – 200 |
| EPS Growth YoY | `eps_growth_yoy` | **Yes** (from `fundamental_metrics_extended.eps_growth_yoy`) | Range | 5 | % | -100 – 500 |

**Extreme growth:** EPS growth can be >1000% for turnaround companies. Cap the UI hint at 500% but allow any numeric input.

#### Dividend Filters

| Filter | View Column | Add to View? | Component | Step | Suffix | Min/Max Hint |
|---|---|---|---|---|---|---|
| Dividend Yield | `dividend_yield` | No (already exists) | Range | 0.1 | % | 0 – 15 |
| Payout Ratio | `payout_ratio` | **Yes** (from `fundamental_metrics_extended.payout_ratio`) | Range | 5 | % | 0 – 200 |
| Consecutive Div Years | `consecutive_dividend_years` | **Yes** (from `fundamental_metrics_extended.consecutive_dividend_years`) | Range (min only) | 1 | years | 0 – 50 |

**Payout ratio >100%:** This happens when companies pay more than they earn (unsustainable). Allow filtering for values >100 to find these.

#### Risk Filters

| Filter | View Column | Add to View? | Component | Step | Suffix | Min/Max Hint |
|---|---|---|---|---|---|---|
| Beta | `beta` | No (already exists — **fix bug**) | Range | 0.1 | — | 0 – 3 |

#### Score Filters

| Filter | View Column | Add to View? | Component | Step | Suffix | Min/Max Hint |
|---|---|---|---|---|---|---|
| IC Score | `ic_score` | No (already exists — **fix bug**) | Range | 1 | — | 0 – 100 |

### 4.3 Materialized View Migration (Phase 1)

New columns to add to `screener_data`:

```sql
-- Migration: 017_expand_screener_materialized_view.sql

DROP MATERIALIZED VIEW IF EXISTS screener_data;

CREATE MATERIALIZED VIEW screener_data AS
SELECT
    t.symbol,
    t.name,
    COALESCE(t.sector, '') as sector,
    COALESCE(t.industry, '') as industry,
    t.market_cap,
    t.asset_type,
    t.active,
    -- Price (latest)
    lp.close as price,
    lp.change_percent,
    -- Valuation (from valuation_ratios)
    lv.ttm_pe_ratio as pe_ratio,
    lv.ttm_pb_ratio as pb_ratio,
    lv.ttm_ps_ratio as ps_ratio,
    -- Profitability (from fundamental_metrics_extended)
    lm.roe,
    lm.roa,
    lm.gross_margin,
    lm.net_margin,
    -- Financial Health
    lm.debt_to_equity,
    lm.current_ratio,
    -- Growth
    lm.revenue_growth_yoy as revenue_growth,
    lm.eps_growth_yoy,
    -- Dividends
    lm.dividend_yield,
    lm.payout_ratio,
    lm.consecutive_dividend_years,
    -- Risk
    lm.beta,
    -- Fair Value (Phase 2)
    lm.dcf_upside_percent,
    -- IC Score
    lic.overall_score as ic_score,
    lic.rating as ic_rating,
    -- IC Score sub-factors (Phase 2)
    lic.value_score,
    lic.growth_score,
    lic.profitability_score,
    lic.financial_health_score,
    lic.momentum_score,
    lic.analyst_consensus_score,
    lic.insider_activity_score,
    lic.institutional_score,
    lic.news_sentiment_score,
    lic.technical_score,
    lic.sector_percentile as ic_sector_percentile,
    lic.lifecycle_stage,
    -- Metadata
    CURRENT_TIMESTAMP as refreshed_at
FROM tickers t
LEFT JOIN LATERAL (
    SELECT close,
           CASE WHEN LAG(close) OVER (ORDER BY time) > 0
                THEN ((close - LAG(close) OVER (ORDER BY time)) / LAG(close) OVER (ORDER BY time)) * 100
                ELSE NULL END as change_percent
    FROM stock_prices
    WHERE ticker = t.symbol
    ORDER BY time DESC
    LIMIT 1
) lp ON true
LEFT JOIN LATERAL (
    SELECT ttm_pe_ratio, ttm_pb_ratio, ttm_ps_ratio
    FROM valuation_ratios
    WHERE ticker = t.symbol
    ORDER BY calculation_date DESC
    LIMIT 1
) lv ON true
LEFT JOIN LATERAL (
    SELECT revenue_growth_yoy, dividend_yield, roe, roa, beta,
           gross_margin, net_margin, debt_to_equity, current_ratio,
           eps_growth_yoy, payout_ratio, consecutive_dividend_years,
           dcf_upside_percent
    FROM fundamental_metrics_extended
    WHERE ticker = t.symbol
    ORDER BY calculation_date DESC
    LIMIT 1
) lm ON true
LEFT JOIN LATERAL (
    SELECT overall_score, rating,
           value_score, growth_score, profitability_score,
           financial_health_score, momentum_score,
           analyst_consensus_score, insider_activity_score,
           institutional_score, news_sentiment_score, technical_score,
           sector_percentile, lifecycle_stage
    FROM ic_scores
    WHERE ticker = t.symbol
    ORDER BY date DESC
    LIMIT 1
) lic ON true
WHERE t.asset_type = 'CS' AND t.active = true;

-- Indexes
CREATE UNIQUE INDEX idx_screener_data_symbol ON screener_data(symbol);
CREATE INDEX idx_screener_data_market_cap ON screener_data(market_cap DESC NULLS LAST);
CREATE INDEX idx_screener_data_sector ON screener_data(sector);
CREATE INDEX idx_screener_data_industry ON screener_data(industry);
CREATE INDEX idx_screener_data_pe_ratio ON screener_data(pe_ratio);
CREATE INDEX idx_screener_data_ic_score ON screener_data(ic_score DESC NULLS LAST);
CREATE INDEX idx_screener_data_dividend_yield ON screener_data(dividend_yield DESC NULLS LAST);
CREATE INDEX idx_screener_data_roe ON screener_data(roe) WHERE roe IS NOT NULL;
CREATE INDEX idx_screener_data_beta ON screener_data(beta) WHERE beta IS NOT NULL;
CREATE INDEX idx_screener_data_de ON screener_data(debt_to_equity) WHERE debt_to_equity IS NOT NULL;
CREATE INDEX idx_screener_data_gross_margin ON screener_data(gross_margin) WHERE gross_margin IS NOT NULL;
CREATE INDEX idx_screener_data_eps_growth ON screener_data(eps_growth_yoy) WHERE eps_growth_yoy IS NOT NULL;
CREATE INDEX idx_screener_data_dcf_upside ON screener_data(dcf_upside_percent) WHERE dcf_upside_percent IS NOT NULL;
CREATE INDEX idx_screener_data_value_score ON screener_data(value_score) WHERE value_score IS NOT NULL;

-- Refresh function (unchanged)
CREATE OR REPLACE FUNCTION refresh_screener_data()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY screener_data;
END;
$$ LANGUAGE plpgsql;

GRANT SELECT ON screener_data TO investorcenter;
```

**Assumption:** The `change_percent` column in the current view is not actually computed (the `stock_prices` LATERAL subquery only selects `close as price`). The migration above adds a window function for `change_percent`, but this may need simplification — the window function won't work inside LATERAL with LIMIT 1. **Alternative:** Store `previous_close` in `tickers` table (already exists as `tickers.previous_close`) and compute change percent as `(price - previous_close) / previous_close * 100` at query time, or add it to the view as a simple formula.

### 4.4 Phase 2 Filters

#### Technical Indicators (Phase 2)

**Data source:** `technical_indicators` hypertable. This table stores indicators in a key-value format (`indicator_name`, `value`), not as separate columns. To include in the materialized view, we need to pivot:

```sql
LEFT JOIN LATERAL (
    SELECT
        MAX(CASE WHEN indicator_name = 'rsi_14' THEN value END) as rsi_14,
        MAX(CASE WHEN indicator_name = 'sma_50' THEN value END) as sma_50,
        MAX(CASE WHEN indicator_name = 'sma_200' THEN value END) as sma_200,
        MAX(CASE WHEN indicator_name = 'macd_histogram' THEN value END) as macd_histogram
    FROM technical_indicators
    WHERE ticker = t.symbol
    AND time = (SELECT MAX(time) FROM technical_indicators WHERE ticker = t.symbol)
) lt ON true
```

**Derived boolean filters:**
- `above_sma50`: `price > sma_50` (computed at query time or in view)
- `above_sma200`: `price > sma_200`
- `golden_cross`: `sma_50 > sma_200`

#### IC Score Sub-Factor Filters (Phase 2)

Already included in the expanded materialized view above. 10 new range filters, each 0-100 with step 1.

#### Fair Value Filters (Phase 2)

`dcf_upside_percent` is already in `fundamental_metrics_extended` and included in the expanded view. Additional derived filters:
- `below_graham`: `price < graham_number` (boolean)
- `below_epv`: `price < epv_fair_value` (boolean)

#### Risk Metrics (Phase 2)

```sql
LEFT JOIN LATERAL (
    SELECT alpha, sharpe_ratio, sortino_ratio, max_drawdown
    FROM risk_metrics
    WHERE ticker = t.symbol AND period = '1Y'
    ORDER BY time DESC
    LIMIT 1
) lr ON true
```

---

## 5. Table Enhancements Spec

### 5.1 Column Registry

```typescript
// lib/screener/column-config.ts

export interface ColumnConfig {
  id: string;                  // Unique key, matches API response field
  label: string;               // Table header text
  shortLabel?: string;         // Abbreviated header for narrow screens
  section: FilterSection;      // Grouping for column picker
  type: 'text' | 'number' | 'percent' | 'currency' | 'score' | 'badge';
  align: 'left' | 'right';
  sortable: boolean;
  sortKey?: string;            // API sort param (if different from id)
  defaultVisible: boolean;
  width?: number;              // Min width in px
  format: (value: any, row?: ScreenerStock) => string | React.ReactNode;
  phase: 1 | 2 | 3;
}

export const COLUMNS: ColumnConfig[] = [
  // Always visible (not hideable)
  { id: 'symbol', label: 'Symbol', section: 'classification', type: 'text', align: 'left', sortable: true, defaultVisible: true, width: 80, format: (v) => v, phase: 1 },
  { id: 'name', label: 'Name', section: 'classification', type: 'text', align: 'left', sortable: true, defaultVisible: true, width: 180, format: (v) => v, phase: 1 },

  // Default visible
  { id: 'market_cap', label: 'Market Cap', shortLabel: 'Mkt Cap', section: 'classification', type: 'currency', align: 'right', sortable: true, defaultVisible: true, width: 100, format: formatLargeNumber, phase: 1 },
  { id: 'price', label: 'Price', section: 'classification', type: 'currency', align: 'right', sortable: true, defaultVisible: true, width: 80, format: (v) => `$${safeToFixed(v, 2)}`, phase: 1 },
  { id: 'change_percent', label: 'Change', section: 'classification', type: 'percent', align: 'right', sortable: true, defaultVisible: true, width: 80, format: formatChangePercent, phase: 1 },
  { id: 'pe_ratio', label: 'P/E', section: 'valuation', type: 'number', align: 'right', sortable: true, defaultVisible: true, width: 70, format: (v) => safeToFixed(v, 1), phase: 1 },
  { id: 'dividend_yield', label: 'Div Yield', section: 'dividends', type: 'percent', align: 'right', sortable: true, defaultVisible: true, width: 80, format: (v) => v ? `${safeToFixed(v, 2)}%` : '—', phase: 1 },
  { id: 'revenue_growth', label: 'Rev Growth', section: 'growth', type: 'percent', align: 'right', sortable: true, defaultVisible: true, width: 90, format: formatGrowthPercent, phase: 1 },
  { id: 'ic_score', label: 'IC Score', section: 'score', type: 'score', align: 'right', sortable: true, defaultVisible: true, width: 90, format: formatICScore, phase: 1 },

  // Available but hidden by default
  { id: 'sector', label: 'Sector', section: 'classification', type: 'text', align: 'left', sortable: true, defaultVisible: false, width: 130, format: (v) => v || '—', phase: 1 },
  { id: 'industry', label: 'Industry', section: 'classification', type: 'text', align: 'left', sortable: true, defaultVisible: false, width: 150, format: (v) => v || '—', phase: 1 },
  { id: 'pb_ratio', label: 'P/B', section: 'valuation', type: 'number', align: 'right', sortable: true, defaultVisible: false, width: 70, format: (v) => safeToFixed(v, 1), phase: 1 },
  { id: 'ps_ratio', label: 'P/S', section: 'valuation', type: 'number', align: 'right', sortable: true, defaultVisible: false, width: 70, format: (v) => safeToFixed(v, 1), phase: 1 },
  { id: 'roe', label: 'ROE', section: 'profitability', type: 'percent', align: 'right', sortable: true, defaultVisible: false, width: 70, format: (v) => v != null ? `${safeToFixed(v, 1)}%` : '—', phase: 1 },
  { id: 'roa', label: 'ROA', section: 'profitability', type: 'percent', align: 'right', sortable: true, defaultVisible: false, width: 70, format: (v) => v != null ? `${safeToFixed(v, 1)}%` : '—', phase: 1 },
  { id: 'gross_margin', label: 'Gross Mgn', section: 'profitability', type: 'percent', align: 'right', sortable: true, defaultVisible: false, width: 80, format: (v) => v != null ? `${safeToFixed(v, 1)}%` : '—', phase: 1 },
  { id: 'net_margin', label: 'Net Mgn', section: 'profitability', type: 'percent', align: 'right', sortable: true, defaultVisible: false, width: 80, format: (v) => v != null ? `${safeToFixed(v, 1)}%` : '—', phase: 1 },
  { id: 'debt_to_equity', label: 'D/E', section: 'financial_health', type: 'number', align: 'right', sortable: true, defaultVisible: false, width: 70, format: (v) => safeToFixed(v, 2), phase: 1 },
  { id: 'current_ratio', label: 'Curr Ratio', section: 'financial_health', type: 'number', align: 'right', sortable: true, defaultVisible: false, width: 80, format: (v) => safeToFixed(v, 2), phase: 1 },
  { id: 'eps_growth_yoy', label: 'EPS Growth', section: 'growth', type: 'percent', align: 'right', sortable: true, defaultVisible: false, width: 90, format: formatGrowthPercent, phase: 1 },
  { id: 'payout_ratio', label: 'Payout', section: 'dividends', type: 'percent', align: 'right', sortable: true, defaultVisible: false, width: 80, format: (v) => v != null ? `${safeToFixed(v, 0)}%` : '—', phase: 1 },
  { id: 'consecutive_dividend_years', label: 'Div Years', section: 'dividends', type: 'number', align: 'right', sortable: true, defaultVisible: false, width: 80, format: (v) => v != null ? `${v}yr` : '—', phase: 1 },
  { id: 'beta', label: 'Beta', section: 'risk', type: 'number', align: 'right', sortable: true, defaultVisible: false, width: 70, format: (v) => safeToFixed(v, 2), phase: 1 },

  // Phase 2 columns
  { id: 'ic_rating', label: 'Rating', section: 'score', type: 'badge', align: 'left', sortable: false, defaultVisible: false, width: 100, format: formatICRating, phase: 2 },
  { id: 'value_score', label: 'Value', section: 'score', type: 'score', align: 'right', sortable: true, defaultVisible: false, width: 70, format: formatSubScore, phase: 2 },
  { id: 'growth_score', label: 'Growth', section: 'score', type: 'score', align: 'right', sortable: true, defaultVisible: false, width: 70, format: formatSubScore, phase: 2 },
  { id: 'momentum_score', label: 'Momentum', section: 'score', type: 'score', align: 'right', sortable: true, defaultVisible: false, width: 80, format: formatSubScore, phase: 2 },
  { id: 'insider_activity_score', label: 'Insider', section: 'score', type: 'score', align: 'right', sortable: true, defaultVisible: false, width: 70, format: formatSubScore, phase: 2 },
  { id: 'dcf_upside_percent', label: 'DCF Upside', section: 'valuation', type: 'percent', align: 'right', sortable: true, defaultVisible: false, width: 90, format: formatGrowthPercent, phase: 2 },
];
```

### 5.2 Column Customization UI

**Component:** `ColumnPicker` — dropdown/popover triggered by a gear icon in the table header.

**Behavior:**
- Grouped by section (Valuation, Profitability, etc.) — matching filter sections
- Checkbox per column; `symbol` and `name` always visible (not toggleable)
- "Reset to Default" button
- Persist to `localStorage` key: `ic_screener_columns`

**State shape:**
```typescript
type VisibleColumns = string[];  // Array of column IDs
// Persisted: ['symbol', 'name', 'market_cap', 'price', 'change_percent', 'pe_ratio', 'dividend_yield', 'revenue_growth', 'ic_score']
```

### 5.3 Multi-Sort (Phase 2)

**UX:**
- Click column header = set as primary sort
- Shift+click = set as secondary sort (shown as smaller sort indicator)
- Maximum 2 sort levels
- Sort indicator: Primary shows `↑`/`↓`, secondary shows `²↑`/`²↓`

**State:**
```typescript
interface SortState {
  primary: { field: string; direction: 'asc' | 'desc' };
  secondary?: { field: string; direction: 'asc' | 'desc' };
}
```

**API mapping:**
```
?sort=ic_score,dividend_yield&order=desc,desc
```

**Backend change:** In `database/screener.go`, parse comma-separated sort/order params:
```go
ORDER BY "ic_score" DESC NULLS LAST, "dividend_yield" DESC NULLS LAST
```

### 5.4 Virtual Scrolling vs. Pagination

**Recommendation: Pagination with page size options.**

Rationale:
- With server-side filtering, we already have paged API responses
- Virtual scrolling is complex to implement correctly with dynamic row heights
- Pagination is the standard UX for data-heavy financial screeners (Finviz, TradingView, Stock Analysis all paginate)
- Page size options: 25, 50, 100 rows per page

**No virtual scrolling needed** — with server-side pagination returning 25-100 rows, there's nothing to virtualize.

### 5.5 Batch Actions (Phase 2)

**Selection state:**
```typescript
const [selectedStocks, setSelectedStocks] = useState<Set<string>>(new Set());  // Set of symbols
```

**UI:**
- Checkbox column (first column) on each row
- "Select all on this page" checkbox in header
- Floating action bar appears when 1+ stocks selected:
  - "Add to Watchlist" — opens watchlist picker dropdown
  - "Compare (max 5)" — opens comparison view
  - "Export Selected" — exports selected rows as CSV
  - Selection count badge: "3 selected"
- Clear selection when filters change

### 5.6 CSV Export

```typescript
// lib/screener/export.ts

export function exportScreenerCSV(
  stocks: ScreenerStock[],
  visibleColumns: string[],
  filename?: string
): void {
  const columns = COLUMNS.filter(c => visibleColumns.includes(c.id));

  // UTF-8 BOM for Excel compatibility
  let csv = '\ufeff';

  // Header row
  csv += columns.map(c => `"${c.label}"`).join(',') + '\n';

  // Data rows
  for (const stock of stocks) {
    csv += columns.map(col => {
      const value = stock[col.id as keyof ScreenerStock];
      if (value === null || value === undefined) return '';
      if (typeof value === 'string') return `"${value.replace(/"/g, '""')}"`;
      return String(value);
    }).join(',') + '\n';
  }

  // Download
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename || `investorcenter-screener-${new Date().toISOString().split('T')[0]}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}
```

**For exports >100 rows:** Add a server-side export endpoint that streams results:
```
GET /api/v1/screener/export?format=csv&{all_filter_params}&limit=10000
```
Returns `Content-Type: text/csv` with `Content-Disposition: attachment; filename="..."`.

---

## 6. Saved Screeners & Presets System

### 6.1 Database Schema

```sql
-- Migration: 018_create_saved_screens.sql

CREATE TABLE saved_screens (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    filters JSONB NOT NULL DEFAULT '{}',      -- URL params as JSON object
    columns JSONB NOT NULL DEFAULT '[]',      -- Array of visible column IDs
    sort JSONB NOT NULL DEFAULT '{}',         -- { primary: { field, direction }, secondary?: ... }
    is_public BOOLEAN DEFAULT false,
    usage_count INTEGER DEFAULT 0,            -- For community rankings
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_saved_screens_user ON saved_screens(user_id);
CREATE INDEX idx_saved_screens_public ON saved_screens(is_public, usage_count DESC)
    WHERE is_public = true;
```

### 6.2 API Routes

```go
// backend/main.go — new routes under auth middleware

screens := v1.Group("/screens")
screens.Use(auth.AuthMiddleware())
{
    screens.GET("", handlers.ListSavedScreens)       // GET /api/v1/screens
    screens.POST("", handlers.CreateSavedScreen)      // POST /api/v1/screens
    screens.PUT("/:id", handlers.UpdateSavedScreen)   // PUT /api/v1/screens/:id
    screens.DELETE("/:id", handlers.DeleteSavedScreen) // DELETE /api/v1/screens/:id
}
```

**Request/Response types:**

```typescript
// Create
POST /api/v1/screens
{
  "name": "My Value Screen",
  "description": "Undervalued large caps with strong dividends",
  "filters": { "pe_max": 15, "div_yield_min": 3, "mcap": ["large", "mega"] },
  "columns": ["symbol", "name", "market_cap", "pe_ratio", "dividend_yield", "ic_score"],
  "sort": { "primary": { "field": "ic_score", "direction": "desc" } }
}

// Response
{
  "id": "uuid-here",
  "name": "My Value Screen",
  "created_at": "2026-02-15T...",
  ...
}

// List
GET /api/v1/screens
[
  { "id": "uuid-1", "name": "My Value Screen", "description": "...", "filters": {...}, "updated_at": "..." },
  { "id": "uuid-2", "name": "Growth Picks", ... }
]
```

### 6.3 UI

- **Save button** in ScreenerToolbar: opens a modal with name + optional description
- **"My Screens" dropdown** next to Quick Screens: lists user's saved screens
- **Load a screen:** Click from dropdown → applies filters, columns, and sort from saved state. URL updates to reflect the loaded filters.
- **Rename/Delete:** Context menu (three-dot icon) on each saved screen in the dropdown
- **Duplicate:** "Save as new screen" copies current screen to a new name
- **Limit:** 20 saved screens per user. Show "Upgrade to Premium for unlimited screens" at limit.

### 6.4 Presets Migration

Quick Screen presets remain hardcoded in the frontend (they're global, not user-specific). They coexist with saved screens:

```
[Quick Screens: Value | Growth | Quality | Dividend | Undervalued]
[My Screens: dropdown ▼]
[Save Current ⚙]
```

---

## 7. Watchlist & Alerts Integration

### 7.1 Add to Watchlist

**Existing infrastructure:** The site already has `/watchlist` with a `watchlists` table and CRUD API.

**Screener integration:**
- Star icon (☆) on each row, filled (★) if stock is in any watchlist
- Click star → if user has one watchlist, add immediately. If multiple, show dropdown picker.
- Batch: select multiple rows → "Add to Watchlist" in floating action bar

**API:** Use existing watchlist endpoints:
```
POST /api/v1/watchlists/:id/stocks
{ "symbol": "AAPL" }
```

**Optimistic update:** Immediately fill the star, revert on error.

**Hydration:** On screener load, fetch user's watchlist stock symbols (`GET /api/v1/watchlists`) and build a `Set<string>` for O(1) membership checks.

### 7.2 Screener-Based Alerts (Phase 2)

**Concept:** "Notify me when a new stock matches these filter criteria."

**Data model:**
```sql
-- Extend existing alerts table or create screener_alerts
CREATE TABLE screener_alerts (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    filters JSONB NOT NULL,                   -- Same format as saved_screens.filters
    last_matched_symbols TEXT[] DEFAULT '{}',  -- Symbols that matched last check
    is_active BOOLEAN DEFAULT true,
    notify_email BOOLEAN DEFAULT true,
    notify_push BOOLEAN DEFAULT false,
    frequency VARCHAR(20) DEFAULT 'daily',    -- 'daily', 'weekly'
    last_checked_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, name)
);
```

**Evaluation mechanism:**
- CronJob runs daily after materialized view refresh (23:50 UTC)
- For each active screener alert:
  1. Run the filter criteria against `screener_data`
  2. Compare resulting symbols with `last_matched_symbols`
  3. If new symbols found: send notification, update `last_matched_symbols`
- Implementation: Go background job or Kubernetes CronJob

**UI:**
- "Create Alert" button in ScreenerToolbar (next to Save)
- Opens modal: "Alert me when new stocks match these filters"
- Name + frequency selection
- Manage alerts at `/alerts`

---

## 8. Performance Budget & Optimization Strategy

### 8.1 Target Metrics

| Metric | Target | Measurement |
|---|---|---|
| Initial page load (LCP) | < 1.5s | Lighthouse, Core Web Vitals |
| Time to interactive | < 2s | First meaningful data visible |
| Filter response (server round-trip) | < 300ms | API p95 latency |
| Perceived filter response | < 100ms | Optimistic UI (keep stale data while loading) |
| Bundle size (screener page JS) | < 150 KB gzipped | `next build` analysis |
| Materialized view refresh | < 2 minutes | Database monitoring |
| API payload (50 rows) | < 15 KB gzipped | Network tab |

### 8.2 Server-Side Optimizations

**Materialized view query performance:**
- Current: 4 LATERAL joins, ~5K stocks, ~30-60s refresh
- Expanded: 4 LATERAL joins (same count, just more columns per join), ~5K stocks
- Expected: ~45-90s refresh (more columns but same join count)
- **If >2 minutes:** Split into `screener_data` (core) + `screener_data_extended` (technical + risk). Join at query time on indexed `symbol`.

**Database indexes:** Partial indexes (WHERE col IS NOT NULL) on nullable filter columns reduce index size and speed up scans.

**Query optimization for multi-filter:**
- PostgreSQL uses bitmap index scans for multi-column WHERE clauses — this is efficient
- Worst case: full scan of ~5K rows in the materialized view, which is still fast (<50ms)
- No query optimizer hints needed at this scale

### 8.3 Client-Side Optimizations

**Debounced API calls:**
```typescript
// In useScreenerParams hook
const debouncedParams = useDebouncedValue(params, 150);  // 150ms debounce

useEffect(() => {
  fetchScreenerData(debouncedParams);
}, [debouncedParams]);
```

**Optimistic UI pattern:**
```typescript
// Keep previous results visible with opacity reduction while loading
<div className={cn('transition-opacity', loading && 'opacity-60')}>
  <ResultsTable stocks={stocks} />
</div>
{loading && <ProgressBar />}  // Subtle top-bar loading indicator
```

**Data caching with SWR:**
```typescript
import useSWR from 'swr';

// Cache screener results by params hash
const paramKey = JSON.stringify(debouncedParams);
const { data, error, isLoading } = useSWR(
  `/api/v1/screener/stocks?${buildQueryString(debouncedParams)}`,
  fetcher,
  {
    keepPreviousData: true,  // Show stale data while revalidating
    revalidateOnFocus: false,
    dedupingInterval: 5000,  // Dedup requests within 5s
  }
);
```

**Note:** SWR is not currently installed. Add `swr` as a dependency (~4KB gzipped). Alternative: React Query (`@tanstack/react-query`, ~12KB) offers more features but is heavier. SWR is sufficient for this use case.

### 8.4 Bundle Size Budget

Current screener page (`app/screener/page.tsx`) is a single 599-line file with no external dependencies beyond React + Next.js utilities. Breaking it into components and adding `nuqs` + `swr`:

| Addition | Size (gzipped) |
|---|---|
| nuqs | ~3 KB |
| swr | ~4 KB |
| Filter config + column config | ~2 KB |
| New components (FilterPanel, ColumnPicker, etc.) | ~8 KB |
| **Total new JS** | **~17 KB** |
| **Budget remaining** | **~133 KB** |

This is well within the 150 KB budget.

---

## 9. Component Architecture

### 9.1 Component Tree

```
ScreenerPage (app/screener/page.tsx — Server Component)
│
│  // Server-side: read URL params, prefetch initial data (optional)
│
└── ScreenerClient (components/screener/ScreenerClient.tsx — Client Component)
    │  // State: URL params ↔ filter state (via useScreenerParams hook)
    │  // Data: SWR query keyed by filter state
    │
    ├── ScreenerToolbar
    │   │  // Props: activeFilterCount, resultCount, loading
    │   ├── PresetSelector         // Quick Screen buttons + My Screens dropdown
    │   ├── SaveScreenButton       // Save current filters (opens modal)
    │   ├── ShareButton            // Copy URL to clipboard
    │   ├── ExportButton           // CSV export
    │   └── ViewToggle             // Table / Card view (Phase 3)
    │
    ├── FilterPanel (collapsible sidebar — sticky on scroll)
    │   │  // Props: params, setParams, activeFilterCount
    │   │  // State: which sections are expanded
    │   │
    │   ├── FilterSection (collapsible group with badge count)
    │   │   ├── CheckboxFilter     // Sector, Industry, Market Cap
    │   │   ├── RangeFilter        // Min/max numeric inputs
    │   │   └── BooleanFilter      // above_sma50, golden_cross (Phase 2)
    │   │
    │   └── FilterFooter
    │       ├── ActiveFilterCount   // "8 filters active"
    │       └── ClearAllButton
    │
    ├── ResultsPanel
    │   │  // Props: stocks, loading, error, columns, sort, selectedStocks
    │   │
    │   ├── ResultsHeader
    │   │   ├── ResultCount         // "2,847 stocks found"
    │   │   └── ColumnPicker        // Gear icon → popover with column checkboxes
    │   │
    │   ├── ResultsTable
    │   │   ├── TableHeader         // Sortable column headers
    │   │   ├── TableRow[]          // Stock data rows
    │   │   │   ├── SelectionCheckbox
    │   │   │   ├── TickerLink      // Clickable symbol → /ticker/{symbol}
    │   │   │   ├── DataCells[]     // Dynamic based on visible columns
    │   │   │   ├── WatchlistStar   // ☆/★ toggle
    │   │   │   └── RowActions      // Three-dot menu (Phase 2)
    │   │   └── TableEmpty          // "No stocks match your filters"
    │   │
    │   ├── Pagination
    │   │   ├── PageSizeSelector    // 25 / 50 / 100
    │   │   ├── PageInfo            // "Showing 1-50 of 2,847"
    │   │   └── PageControls        // Previous / Page X of Y / Next
    │   │
    │   └── BatchActionBar          // Floating bar when stocks selected (Phase 2)
    │       ├── SelectionCount
    │       ├── AddToWatchlistBtn
    │       ├── CompareBtn
    │       └── ExportSelectedBtn
    │
    └── SaveScreenModal (Phase 2)
        ├── NameInput
        ├── DescriptionInput
        └── SaveButton
```

### 9.2 Component Props & Interfaces

```typescript
// components/screener/ScreenerClient.tsx
interface ScreenerClientProps {
  initialData?: ScreenerResponse;  // Optional server-prefetched data
}

// components/screener/FilterPanel.tsx
interface FilterPanelProps {
  params: ScreenerParams;
  setParams: (params: Partial<ScreenerParams>) => void;
  activeFilterCount: number;
  show: boolean;
  onToggle: () => void;
}

// components/screener/FilterSection.tsx
interface FilterSectionProps {
  label: string;
  defaultOpen: boolean;
  activeCount: number;     // Badge count for this section
  children: React.ReactNode;
}

// components/screener/RangeFilter.tsx
interface RangeFilterProps {
  label: string;
  paramMin: string;
  paramMax: string;
  value: { min?: number; max?: number };
  onChange: (value: { min?: number; max?: number }) => void;
  step?: number;
  suffix?: string;
  tooltip?: string;
  min?: number;           // Suggested minimum
  max?: number;           // Suggested maximum
}

// components/screener/CheckboxFilter.tsx
interface CheckboxFilterProps {
  label: string;
  options: { value: string; label: string }[];
  selected: string[];
  onChange: (selected: string[]) => void;
  maxHeight?: number;     // For scrollable lists
}

// components/screener/ResultsTable.tsx
interface ResultsTableProps {
  stocks: ScreenerStock[];
  columns: ColumnConfig[];
  sort: SortState;
  onSort: (field: string, shift: boolean) => void;
  selectedStocks: Set<string>;
  onSelect: (symbol: string) => void;
  onSelectAll: () => void;
  watchlistSymbols: Set<string>;
  onWatchlistToggle: (symbol: string) => void;
  loading: boolean;
}

// components/screener/Pagination.tsx
interface PaginationProps {
  page: number;
  limit: number;
  total: number;
  onPageChange: (page: number) => void;
  onLimitChange: (limit: number) => void;
}
```

### 9.3 Accessibility Requirements

| Component | Requirement |
|---|---|
| FilterPanel | `role="form"`, `aria-label="Stock screener filters"` |
| FilterSection | `<details>` + `<summary>` for native expand/collapse, or `aria-expanded` |
| RangeFilter | `<input type="number">` with `aria-label="Minimum {label}"` |
| CheckboxFilter | `<fieldset>` + `<legend>` wrapping checkboxes |
| ResultsTable | `<table>` with `<thead>`, `<tbody>`, `aria-sort` on sorted columns |
| SortableHeader | `role="columnheader"`, `aria-sort="ascending|descending|none"`, keyboard-triggerable |
| Pagination | `nav` element with `aria-label="Pagination"` |
| ColumnPicker | `role="dialog"`, focus trap, Escape to close |
| BatchActionBar | `role="toolbar"`, `aria-label="Actions for selected stocks"` |

**Keyboard navigation:**
- Tab through filters, Enter to toggle checkboxes
- Table headers focusable, Enter/Space to sort
- Escape closes modals and popovers

---

## 10. Migration & Rollout Plan

### 10.1 Implementation Phases

#### PR 1: Fix IC Score + Beta Bug (Day 1)
**Files changed:**
- `backend/database/screener.go` — Replace `0.0 as beta, 0.0 as ic_score` with actual column reads. Fix sort mappings.

**Changes:**
```go
// Line 166-167: Replace
//   0.0 as beta,
//   0.0 as ic_score
// With:
    beta,
    ic_score
```

```go
// Line 23-24: Fix sort mappings
"beta":     "beta",      // was: "roe"
"ic_score": "ic_score",  // was: "market_cap"
```

**Testing:** `cd backend && go test ./database -v -run TestScreener`
**Risk:** Low. Only changes 4 lines. Rollback: revert the 4 lines.

#### PR 2: Expand Materialized View (Week 1)
**Files changed:**
- `ic-score-service/migrations/017_expand_screener_materialized_view.sql` — New migration

**Testing:** Run migration on staging DB. Verify refresh time <2 minutes. Spot-check data for a few tickers.
**Risk:** Medium. If refresh time exceeds 2 minutes, fall back to the split-view approach.
**Rollback:** Re-run migration 016 to restore original view.

#### PR 3: Backend Filter Registry + New Filter Params (Week 1-2)
**Files changed:**
- `backend/database/screener.go` — Refactor to use filter registry pattern
- `backend/database/filter_registry.go` — New file
- `backend/handlers/screener.go` — Parse new query params
- `backend/models/stock.go` — Expand `ScreenerStock` and `ScreenerParams` structs

**Testing:** `cd backend && go test ./... -v`
**Risk:** Medium. Changes the query builder. Thorough testing needed.
**Rollback:** Revert to hardcoded filter approach.

#### PR 4: URL State Management (Week 2)
**Files changed:**
- `package.json` — Add `nuqs` dependency
- `lib/screener/url-params.ts` — New file: param parsers
- `lib/screener/presets.ts` — New file: preset definitions
- `lib/hooks/useScreenerParams.ts` — New hook
- `app/screener/page.tsx` — Refactor to use URL params

**Testing:** Manual QA: apply filters, refresh page, verify persistence. Test browser back/forward. Test shared URLs.
**Risk:** Low. Additive change. Falls back to default state if params parsing fails.
**Rollback:** Remove `nuqs`, revert to `useState`.

#### PR 5: New Screener UI (Consolidated) (Week 2-3)
**Files changed:**
- `app/screener/page.tsx` — Replace monolithic component with component tree
- `components/screener/ScreenerClient.tsx` — New
- `components/screener/FilterPanel.tsx` — New
- `components/screener/FilterSection.tsx` — New
- `components/screener/RangeFilter.tsx` — New
- `components/screener/CheckboxFilter.tsx` — New
- `components/screener/ResultsTable.tsx` — New
- `components/screener/ColumnPicker.tsx` — New
- `components/screener/Pagination.tsx` — New
- `components/screener/ScreenerToolbar.tsx` — New
- `lib/screener/filter-config.ts` — New
- `lib/screener/column-config.ts` — New
- `lib/screener/export.ts` — New

**Testing:** Visual QA against current screener. Verify all existing functionality works. Test new filters.
**Risk:** High (large UI change). Use feature flag for gradual rollout.
**Rollback:** Feature flag → switch back to old screener.

#### PR 6: CSV Export (Week 3)
**Files changed:**
- `lib/screener/export.ts` — New file
- `components/screener/ScreenerToolbar.tsx` — Add export button

**Testing:** Export 50 rows, 1000 rows, 10000 rows. Open in Excel, Google Sheets. Verify encoding.
**Risk:** Low. Client-side only, no backend changes.

### 10.2 Feature Flag Strategy

```typescript
// lib/feature-flags.ts
export const FEATURE_FLAGS = {
  SCREENER_V2: process.env.NEXT_PUBLIC_SCREENER_V2 === 'true',
  SCREENER_SAVED_SCREENS: process.env.NEXT_PUBLIC_SCREENER_SAVED === 'true',
  SCREENER_TECHNICAL_FILTERS: process.env.NEXT_PUBLIC_SCREENER_TECHNICAL === 'true',
};
```

```typescript
// app/screener/page.tsx
import { FEATURE_FLAGS } from '@/lib/feature-flags';
import ScreenerV1 from './ScreenerV1';  // Renamed current page
import { ScreenerClient } from '@/components/screener/ScreenerClient';

export default function ScreenerPage() {
  if (FEATURE_FLAGS.SCREENER_V2) {
    return <ScreenerClient />;
  }
  return <ScreenerV1 />;
}
```

**Rollout sequence:**
1. Deploy with `SCREENER_V2=false` — old screener still serves
2. Enable for internal team: `SCREENER_V2=true` on staging
3. Enable for 10% of users (cookie-based routing)
4. Monitor error rates, performance metrics
5. Ramp to 50%, then 100%
6. Remove flag and delete ScreenerV1

### 10.3 IC Score Screener Consolidation

**Current state:** Two screener pages exist: `/screener` (basic) and `/ic-score` (IC Score).

**Plan:** After Phase 1 is stable:
1. The new `/screener` has IC Score as a first-class filter (it already includes it)
2. Add IC Score sub-factor filters in Phase 2
3. Redirect `/ic-score` to `/screener?sort=ic_score&order=desc` with a 301
4. Eventually remove `components/ic-score/ICScoreScreener.tsx`

### 10.4 Data Migration Steps

1. **Migration 017:** Expand `screener_data` materialized view (PR 2)
2. **Migration 018:** Create `saved_screens` table (Phase 2)
3. **Migration 019:** Create `screener_alerts` table (Phase 2)
4. **No data migration needed** — the materialized view recreates from source tables
5. **Index creation** is part of the view migration — runs once during deployment

### 10.5 A/B Testing

**Phase 1 A/B test: Server-side vs. client-side filtering response time.**

Not recommended for Phase 1 — the server-side approach is architecturally correct and the IC Score screener already validates it. Skip A/B testing and commit to server-side.

**Phase 2 A/B test candidates:**
- Filter sidebar position (left vs. top)
- Default columns (current 9 vs. proposed 11 with P/B + ROE)
- Quick Screen presets (4 current vs. 6 proposed)
- Pagination sizes (25 default vs. 50 default)

### 10.6 Rollback Plan

| Change | Rollback Mechanism | Time to Rollback |
|---|---|---|
| IC Score/Beta fix | Revert 4 lines in screener.go | 5 minutes (redeploy) |
| Materialized view | Run migration 016 to restore original view | 10 minutes |
| Backend filter params | Revert PR, redeploy | 15 minutes |
| Frontend V2 | Set `SCREENER_V2=false` env var | 2 minutes (no redeploy) |
| Saved screens | Drop table (no other tables depend on it) | 5 minutes |

---

## Appendix A: Dependencies to Add

```bash
npm install nuqs swr
```

| Package | Size (gzip) | Purpose |
|---|---|---|
| `nuqs` | ~3 KB | Type-safe URL state management for Next.js |
| `swr` | ~4 KB | Data fetching with caching and revalidation |

No other new dependencies. The existing stack (React 18, Next.js 14, Tailwind, Lucide icons) provides everything else needed.

## Appendix B: Files Created / Modified Summary

### New Files
```
lib/screener/url-params.ts          # URL param parsers
lib/screener/filter-config.ts       # Filter registry (frontend)
lib/screener/column-config.ts       # Column registry
lib/screener/presets.ts             # Preset definitions
lib/screener/export.ts              # CSV export utility
lib/hooks/useScreenerParams.ts      # URL ↔ state sync hook
components/screener/ScreenerClient.tsx
components/screener/FilterPanel.tsx
components/screener/FilterSection.tsx
components/screener/RangeFilter.tsx
components/screener/CheckboxFilter.tsx
components/screener/ResultsTable.tsx
components/screener/ColumnPicker.tsx
components/screener/Pagination.tsx
components/screener/ScreenerToolbar.tsx
components/screener/BatchActionBar.tsx    # Phase 2
components/screener/SaveScreenModal.tsx   # Phase 2
backend/database/filter_registry.go       # Filter registry (backend)
ic-score-service/migrations/017_expand_screener_materialized_view.sql
```

### Modified Files
```
backend/database/screener.go        # Fix bugs, refactor to use registry
backend/handlers/screener.go        # Parse new filter params
backend/models/stock.go             # Expand ScreenerStock, ScreenerParams
backend/main.go                     # Add /screens routes (Phase 2)
app/screener/page.tsx               # Feature flag → new component
package.json                        # Add nuqs, swr
```
