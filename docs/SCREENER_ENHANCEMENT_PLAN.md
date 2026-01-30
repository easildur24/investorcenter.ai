# Stock Screener Enhancement Plan

## Executive Summary

This plan enhances the InvestorCenter.ai stock screener based on competitive analysis of Finviz, TradingView, Yahoo Finance, Seeking Alpha, Morningstar, Stock Rover, and Zacks. The current implementation has significant gaps that this plan addresses in three phases.

---

## Current State Analysis

### What We Have (6 Filters)
- Sector, Market Cap, P/E Ratio, Dividend Yield, Revenue Growth, IC Score

### Critical Issues
| Issue | Location | Impact |
|-------|----------|--------|
| Beta is placeholder | `backend/database/screener.go:23` | Shows ROE instead of Beta |
| IC Score is placeholder | `backend/database/screener.go:24` | Shows market_cap instead |
| 24h Change missing | Materialized view | Always null |
| Two separate screeners | `app/screener/` + `components/ic-score/` | Inconsistent UX |
| Client-side filtering | `app/screener/page.tsx` | Fetches 20K stocks |

### Competitors Comparison
| Feature | Us | Finviz | TradingView | Stock Rover |
|---------|-----|--------|-------------|-------------|
| Total Filters | 6 | 67 | 168+ | 700+ |
| Preset Screens | 4 | 50+ | Many | 140+ |
| Heat Maps | ❌ | ✅ | ❌ | ❌ |
| Saved Screens | ❌ | ✅ | ✅ | ✅ |
| Export | ❌ | ✅ | ✅ | ✅ |
| Proprietary Score | ✅ IC Score | ❌ | ❌ | ❌ |

---

## Phase 1: Fix Foundation (7-9 days)

### 1.1 Fix Placeholder Data (2-3 days)

**Problem**: Beta, IC Score, and 24h Change are hardcoded/missing.

**Database Changes** - New migration `022_update_screener_materialized_view.sql`:

```sql
-- Update screener_data materialized view
CREATE MATERIALIZED VIEW screener_data AS
SELECT
    t.symbol,
    t.name,
    COALESCE(t.sector, '') as sector,
    COALESCE(t.industry, '') as industry,
    t.market_cap,
    lp.price as current_price,
    -- FIX: Calculate change_percent from previous close
    CASE
        WHEN lp.prev_close > 0 THEN ((lp.price - lp.prev_close) / lp.prev_close) * 100
        ELSE NULL
    END as change_percent,
    lv.ttm_pe_ratio as pe_ratio,
    lv.ttm_pb_ratio as pb_ratio,
    lv.ttm_ps_ratio as ps_ratio,
    lm.revenue_growth_yoy as revenue_growth,
    lm.dividend_yield,
    lm.roe,
    -- FIX: Real beta from risk_metrics
    COALESCE(lr.beta_1y, lm.beta) as beta,
    -- FIX: Real IC Score from ic_scores
    lic.overall_score as ic_score,
    lm.debt_to_equity,
    lm.earnings_growth_yoy
FROM tickers t
LEFT JOIN LATERAL (...) lp ON true  -- Price with prev_close
LEFT JOIN LATERAL (...) lr ON true  -- Risk metrics for beta
LEFT JOIN LATERAL (...) lic ON true -- IC Scores
-- ... rest of joins
```

**Backend Changes** - `backend/database/screener.go`:

```go
// Replace lines 23-24
var ValidScreenerSortColumns = map[string]string{
    "beta":           "beta",           // Was: "roe" (placeholder)
    "ic_score":       "ic_score",       // Was: "market_cap" (placeholder)
    "change_percent": "change_percent", // New column
}
```

**Files to Modify**:
- `ic-score-service/migrations/022_update_screener_materialized_view.sql` (new)
- `backend/database/screener.go` (lines 23-24, 141-162)

---

### 1.2 Add Missing Fundamentals (1 day)

**Add filters for**: P/B Ratio, P/S Ratio, ROE, Debt/Equity

**Frontend** - `app/screener/page.tsx`:

```typescript
// Add to FILTERS array (after line 117)
{
  id: 'pbRatio',
  label: 'P/B Ratio',
  field: 'pb_ratio',
  type: 'range',
  min: 0,
  max: 20,
  step: 0.1,
},
{
  id: 'psRatio',
  label: 'P/S Ratio',
  field: 'ps_ratio',
  type: 'range',
  min: 0,
  max: 20,
  step: 0.1,
},
{
  id: 'roe',
  label: 'ROE (%)',
  field: 'roe',
  type: 'range',
  min: -50,
  max: 100,
  step: 1,
},
{
  id: 'debtEquity',
  label: 'Debt/Equity',
  field: 'debt_to_equity',
  type: 'range',
  min: 0,
  max: 5,
  step: 0.1,
},
```

**Files to Modify**:
- `app/screener/page.tsx` (FILTERS array, Stock interface, table columns)
- `backend/database/screener.go` (SELECT query)

---

### 1.3 Consolidate Two Screeners (3-4 days)

**Problem**: Two separate implementations with different APIs and UX.

**Solution**: Merge into unified screener with tabs.

**New Component** - `components/screener/UnifiedScreener.tsx`:

```typescript
interface UnifiedScreenerProps {
  defaultView?: 'all' | 'ic-score';
}

export function UnifiedScreener({ defaultView = 'all' }: UnifiedScreenerProps) {
  const [view, setView] = useState(defaultView);

  return (
    <div>
      <Tabs value={view} onChange={setView}>
        <Tab value="all">All Stocks</Tab>
        <Tab value="ic-score">IC Score Focus</Tab>
      </Tabs>

      <FilterSidebar filters={view === 'ic-score' ? IC_SCORE_FILTERS : ALL_FILTERS} />
      <ResultsTable view={view} />
    </div>
  );
}
```

**Files to Modify**:
- `components/screener/UnifiedScreener.tsx` (new)
- `app/screener/page.tsx` (use unified component)
- `app/ic-score/page.tsx` (redirect to `/screener?view=ic-score`)
- `backend/handlers/screener.go` (add IC Score rating filter)

---

### 1.4 Add Company Name/Ticker Search (0.5 days)

**Frontend** - `app/screener/page.tsx`:

```typescript
const [searchQuery, setSearchQuery] = useState('');

// In filter logic
if (searchQuery) {
  const query = searchQuery.toLowerCase();
  result = result.filter(stock =>
    stock.symbol.toLowerCase().includes(query) ||
    stock.name.toLowerCase().includes(query)
  );
}

// Add search input above filters
<Input
  placeholder="Search by symbol or name..."
  value={searchQuery}
  onChange={(e) => setSearchQuery(e.target.value)}
  leftIcon={<SearchIcon />}
/>
```

---

## Phase 2: Table Stakes (7-9 days)

### 2.1 Saved Screeners (2-3 days)

**Database** - New migration `023_saved_screeners.sql`:

```sql
CREATE TABLE saved_screeners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    filters JSONB NOT NULL,
    sort_config JSONB,
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_screener_name_per_user UNIQUE (user_id, name)
);

CREATE INDEX idx_saved_screeners_user_id ON saved_screeners(user_id);
```

**New API Endpoints**:
```
GET    /api/v1/screener/saved      - List user's saved screeners
POST   /api/v1/screener/saved      - Save new screener
PUT    /api/v1/screener/saved/:id  - Update screener
DELETE /api/v1/screener/saved/:id  - Delete screener
```

**New Files**:
- `backend/models/saved_screener.go`
- `backend/database/saved_screeners.go`
- `backend/handlers/saved_screener_handlers.go`
- `ic-score-service/migrations/023_saved_screeners.sql`

---

### 2.2 CSV/Excel Export (1 day)

**Frontend** - `app/screener/page.tsx`:

```typescript
import { utils, writeFile } from 'xlsx';

const handleExport = (format: 'csv' | 'xlsx') => {
  const exportData = filteredStocks.map(stock => ({
    Symbol: stock.symbol,
    Name: stock.name,
    Sector: stock.sector,
    'Market Cap': formatMarketCap(stock.market_cap),
    Price: stock.price,
    'Change %': stock.change_percent,
    'P/E': stock.pe_ratio,
    'Dividend Yield': stock.dividend_yield,
    'IC Score': stock.ic_score,
  }));

  const ws = utils.json_to_sheet(exportData);
  const wb = utils.book_new();
  utils.book_append_sheet(wb, ws, 'Screener Results');
  writeFile(wb, `screener_${new Date().toISOString().split('T')[0]}.${format}`);
};

// Add export button
<DropdownMenu>
  <DropdownMenuTrigger asChild>
    <Button variant="outline"><Download /> Export</Button>
  </DropdownMenuTrigger>
  <DropdownMenuContent>
    <DropdownMenuItem onClick={() => handleExport('csv')}>CSV</DropdownMenuItem>
    <DropdownMenuItem onClick={() => handleExport('xlsx')}>Excel</DropdownMenuItem>
  </DropdownMenuContent>
</DropdownMenu>
```

**Dependencies**: `npm install xlsx`

---

### 2.3 More Preset Screens (0.5 days)

**Expand to 15 Presets** - `app/screener/page.tsx`:

```typescript
const PRESET_SCREENS = [
  // Existing (4)
  { id: 'value', name: 'Value Stocks', filters: { peRatio: { max: 15 }, dividendYield: { min: 2 } } },
  { id: 'growth', name: 'Growth Stocks', filters: { revenueGrowth: { min: 20 } } },
  { id: 'quality', name: 'Quality Stocks', filters: { icScore: { min: 70 }, marketCap: ['mega', 'large'] } },
  { id: 'dividend', name: 'Dividend Champions', filters: { dividendYield: { min: 3 }, marketCap: ['mega', 'large'] } },

  // New (11)
  { id: 'deep_value', name: 'Deep Value', filters: { peRatio: { max: 10 }, pbRatio: { max: 1 } } },
  { id: 'small_cap_growth', name: 'Small Cap Growth', filters: { marketCap: ['small', 'micro'], revenueGrowth: { min: 25 } } },
  { id: 'low_debt', name: 'Low Debt', filters: { debtEquity: { max: 0.3 } } },
  { id: 'high_roe', name: 'High ROE', filters: { roe: { min: 20 } } },
  { id: 'ic_strong_buy', name: 'IC Score: Strong Buy', filters: { icScore: { min: 80 } } },
  { id: 'ic_buy', name: 'IC Score: Buy', filters: { icScore: { min: 60, max: 80 } } },
  { id: 'beaten_down', name: 'Beaten Down', filters: { changePercent: { max: -20 }, peRatio: { max: 15 } } },
  { id: 'momentum', name: 'Momentum', filters: { changePercent: { min: 10 }, revenueGrowth: { min: 10 } } },
  { id: 'tech_leaders', name: 'Tech Leaders', filters: { sector: ['Technology'], marketCap: ['mega', 'large'] } },
  { id: 'healthcare', name: 'Healthcare Quality', filters: { sector: ['Healthcare'], icScore: { min: 60 } } },
  { id: 'energy_value', name: 'Energy Value', filters: { sector: ['Energy'], peRatio: { max: 12 } } },
];
```

---

### 2.4 Technical Filters (3-4 days)

**Database** - Update materialized view to include technical indicators:

```sql
-- Add to screener_data view
LEFT JOIN LATERAL (
    SELECT value as rsi_14
    FROM technical_indicators
    WHERE ticker = t.symbol AND indicator_name = 'RSI_14'
    ORDER BY time DESC LIMIT 1
) lrsi ON true

LEFT JOIN LATERAL (
    SELECT value as sma_50
    FROM technical_indicators
    WHERE ticker = t.symbol AND indicator_name = 'SMA_50'
    ORDER BY time DESC LIMIT 1
) lsma50 ON true

LEFT JOIN LATERAL (
    SELECT value as sma_200
    FROM technical_indicators
    WHERE ticker = t.symbol AND indicator_name = 'SMA_200'
    ORDER BY time DESC LIMIT 1
) lsma200 ON true

-- Calculate derived fields
CASE WHEN lp.price > lsma50.sma_50 THEN 'above' ELSE 'below' END as price_vs_sma50,
CASE WHEN lp.price > lsma200.sma_200 THEN 'above' ELSE 'below' END as price_vs_sma200,
-- 52-week position (0-100%)
((lp.price - l52.low_52w) / NULLIF(l52.high_52w - l52.low_52w, 0)) * 100 as week52_position
```

**Frontend Filters**:

```typescript
// Technical filter section
{
  id: 'rsi',
  label: 'RSI (14)',
  field: 'rsi_14',
  type: 'range',
  min: 0,
  max: 100,
  step: 5,
},
{
  id: 'priceVsSma50',
  label: 'Price vs 50-Day MA',
  field: 'price_vs_sma50',
  type: 'select',
  options: [
    { value: 'above', label: 'Above 50 MA' },
    { value: 'below', label: 'Below 50 MA' },
  ],
},
{
  id: 'priceVsSma200',
  label: 'Price vs 200-Day MA',
  field: 'price_vs_sma200',
  type: 'select',
  options: [
    { value: 'above', label: 'Above 200 MA' },
    { value: 'below', label: 'Below 200 MA' },
  ],
},
{
  id: 'week52Position',
  label: '52-Week Range Position',
  field: 'week52_position',
  type: 'range',
  min: 0,
  max: 100,
  step: 10,
  description: '0% = at 52-week low, 100% = at 52-week high',
},
```

---

## Phase 3: Differentiators (15-19 days)

### 3.1 Heat Map Visualization (4-5 days)

Finviz-style sector/industry treemap. Reuse existing D3 treemap from `components/watchlist/WatchListHeatmap.tsx`.

**New Component** - `components/screener/ScreenerHeatmap.tsx`:

```typescript
interface HeatmapConfig {
  groupBy: 'sector' | 'industry';
  sizeBy: 'market_cap' | 'volume';
  colorBy: 'change_percent' | 'ic_score';
}

export function ScreenerHeatmap({ stocks, config }: Props) {
  // Build hierarchical data structure
  const data = useMemo(() => buildTreemapData(stocks, config), [stocks, config]);

  // Render D3 treemap with:
  // - Tile size = market cap
  // - Tile color = performance (green/red) or IC Score gradient
  // - Hover tooltip with stock details
  // - Click to filter table to that sector/industry
}
```

**New API Endpoint**:
```
GET /api/v1/screener/heatmap?group_by=sector&color_by=change_percent
```

---

### 3.2 IC Score Factor Breakdown (2-3 days)

Show factor scores when expanding a row or hovering IC Score.

**Reuse existing** `components/ic-score/FactorBreakdown.tsx`.

**Backend** - Add `include_factors=true` parameter:

```go
// In screener handler
if c.Query("include_factors") == "true" {
    // Join ic_score_factors table
    // Include all 10 factor scores in response
}
```

**Frontend** - Expandable row:

```typescript
<TableRow onClick={() => toggleExpanded(stock.symbol)}>
  {/* Normal columns */}
</TableRow>
{expanded === stock.symbol && (
  <TableRow className="bg-muted/50">
    <TableCell colSpan={columns.length}>
      <FactorBreakdown
        factors={stock.factors}
        symbol={stock.symbol}
      />
    </TableCell>
  </TableRow>
)}
```

---

### 3.3 Screener Alerts (4-5 days)

Alert users when new stocks match their saved screener criteria.

**Extend existing alert system** (`backend/migrations/012_alert_system.sql`):

```sql
-- Add screener_criteria alert type
ALTER TABLE alert_rules DROP CONSTRAINT valid_alert_type;
ALTER TABLE alert_rules ADD CONSTRAINT valid_alert_type CHECK (alert_type IN (
    'price_above', 'price_below', 'price_change_pct',
    'volume_spike', 'ic_score_change',
    'screener_criteria'  -- NEW
));
```

**Alert Processor** - `backend/services/alert_processor.go`:

```go
func (p *AlertProcessor) evaluateScreenerCriteria(rule AlertRule) ([]Stock, error) {
    // Parse rule.Conditions as screener filters
    // Query screener_data with those filters
    // Compare to previously matched stocks
    // Return newly matched stocks
}
```

**Frontend**:

```typescript
// "Create Alert" button in screener
<Button onClick={() => openAlertModal(currentFilters)}>
  <Bell /> Create Alert from Filters
</Button>
```

---

### 3.4 Real-Time Data Option (5-6 days)

WebSocket-based live price updates for premium users.

**Backend** - `backend/handlers/screener_realtime.go`:

```go
func ScreenerRealtimeHandler(c *gin.Context) {
    // Check premium subscription
    if !user.HasPremium() {
        c.JSON(403, gin.H{"error": "Real-time requires Premium"})
        return
    }

    // Upgrade to WebSocket
    conn, _ := upgrader.Upgrade(c.Writer, c.Request, nil)

    // Subscribe to Polygon WebSocket
    // Filter updates to user's screener criteria
    // Push matching updates to client
}
```

**Frontend**:

```typescript
const [isRealtime, setIsRealtime] = useState(false);

useEffect(() => {
  if (!isRealtime) return;

  const ws = new WebSocket(`${WS_URL}/api/v1/screener/realtime`);
  ws.onmessage = (event) => {
    const update = JSON.parse(event.data);
    setStocks(prev => prev.map(s =>
      s.symbol === update.symbol ? { ...s, ...update } : s
    ));
  };

  return () => ws.close();
}, [isRealtime]);
```

---

## Implementation Timeline

```
Week 1-2: Phase 1 - Foundation
├── Day 1-3:  Fix placeholder data (Beta, IC Score, Change %)
├── Day 4:    Add missing fundamentals (P/B, P/S, ROE, D/E)
├── Day 5-8:  Consolidate screeners
└── Day 9:    Add search

Week 3-4: Phase 2 - Table Stakes
├── Day 10-12: Saved screeners
├── Day 13:    CSV/Excel export
├── Day 14:    More preset screens
└── Day 15-18: Technical filters

Week 5-8: Phase 3 - Differentiators
├── Day 19-23: Heat map visualization
├── Day 24-26: IC Score factor breakdown
├── Day 27-31: Screener alerts
└── Day 32-37: Real-time data
```

---

## Dependency Graph

```
Phase 1:
1.1 Fix Placeholders ──┬──> 1.2 Add Fundamentals
                       │
                       └──> 1.3 Consolidate Screeners ──> 1.4 Add Search

Phase 2:
1.3 Consolidate ──┬──> 2.1 Saved Screeners ──> 2.3 Preset Screens
                  │
                  ├──> 2.2 Export (independent)
                  │
                  └──> 2.4 Technical Filters (needs indicator pipeline)

Phase 3:
2.1 Saved Screeners ──> 3.3 Screener Alerts
1.3 Consolidate ──────> 3.1 Heat Map
1.1 Fix IC Score ─────> 3.2 Factor Breakdown
All Phase 2 ──────────> 3.4 Real-Time Data
```

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Missing Beta data for some stocks | Show "N/A", don't break filter |
| Technical indicators not calculated | Run indicator pipeline first, or exclude from initial release |
| Performance with 20K stocks | Move to server-side filtering in Phase 2 |
| Real-time data costs | Gate behind Premium tier, implement rate limiting |

---

## Success Metrics

| Metric | Current | Target (Phase 1) | Target (Phase 3) |
|--------|---------|------------------|------------------|
| Total Filters | 6 | 12 | 20+ |
| Preset Screens | 4 | 10 | 15 |
| Data Accuracy | ~65% | 95% | 99% |
| User Saved Screens | 0 | Enabled | + Alerts |
| Export Options | 0 | CSV/Excel | + API |
