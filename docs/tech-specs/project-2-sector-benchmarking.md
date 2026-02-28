# Tech Spec: Project 2 — Sector Benchmarking (Percentile Bars)

**Parent PRD:** `docs/prd-enhanced-fundamentals-experience.md`
**Priority:** P0
**Estimated Effort:** 2 sprints
**Dependencies:** Project 1 (sector-percentiles API endpoint)

---

## 1. Overview

Build a reusable `SectorPercentileBar` React component and integrate it alongside key financial metrics across three existing surfaces: `TickerFundamentals` sidebar, `MetricsTab`, and `KeyStatsTab`. Each bar shows where a stock's metric falls within its sector distribution, with direction-aware color coding (low P/E = green, low ROE = red).

## 2. Architecture Context

### Existing Frontend Stack
- **Framework:** Next.js 14 (App Router) with TypeScript
- **Styling:** Tailwind CSS with custom IC theme variables (`ic-positive`, `ic-negative`, `ic-warning`, etc.)
- **Charts:** Recharts (primary), Chart.js, D3 available
- **Data Fetching:** Direct `fetch()` with `useEffect` in client components; `lib/api/routes.ts` as route registry
- **Tooltips:** Custom `Tooltip` and `CalculationTooltip` components (`components/ui/Tooltip.tsx`)

### Target Components (Existing Files)

| Component | File | Current Behavior | Integration Point |
|---|---|---|---|
| `TickerFundamentals` | `components/ticker/TickerFundamentals.tsx` | 15 metrics as key-value pairs, no benchmarking | Add percentile bar after each metric value |
| `MetricsTab` | `components/ticker/tabs/MetricsTab.tsx` | Category sub-tabs with `MetricCard`-style rows | Add percentile bar to each metric row |
| `KeyStatsTab` | `components/ticker/tabs/KeyStatsTab.tsx` | 50+ metrics in 15 `GroupCard` sections | Add percentile bar to key metrics |

### Data Source

**API:** `GET /api/v1/stocks/:ticker/sector-percentiles` (Project 1)

**Response shape (relevant fields):**
```typescript
interface SectorPercentilesResponse {
  ticker: string;
  sector: string;
  calculated_at: string;
  sample_count: number;
  metrics: Record<string, {
    value: number;
    percentile: number;          // 0-100, direction-adjusted
    lower_is_better: boolean;
    distribution: {
      min: number;
      p10: number;
      p25: number;
      p50: number;
      p75: number;
      p90: number;
      max: number;
    };
    sample_count: number;
  }>;
}
```

---

## 3. Component Design

### 3.1 `SectorPercentileBar` — Core Reusable Component

**File:** `components/ui/SectorPercentileBar.tsx`

**Props Interface:**

```typescript
interface SectorPercentileBarProps {
  /** Stock's percentile (0-100, already direction-adjusted by API) */
  percentile: number;
  /** Sector distribution breakpoints */
  distribution: {
    p10: number;
    p25: number;
    p50: number;
    p75: number;
    p90: number;
  };
  /** Stock's actual metric value */
  value: number;
  /** Whether lower raw values are better (P/E, D/E) */
  lowerIsBetter: boolean;
  /** Metric display name for tooltip */
  metricName: string;
  /** Sector name for tooltip */
  sector: string;
  /** Sample size for tooltip */
  sampleCount: number;
  /** Visual variant */
  size?: 'sm' | 'md';
  /** Whether to show the bar (false = hidden, for free-tier gating) */
  visible?: boolean;
}
```

**Visual Rendering:**

```
Size 'md' (MetricsTab, KeyStatsTab):
┌──────────────────────────────────────────────┐
│  [▓▓▓▓░░░░░░░░░░●░░░░░░░░░░░░░░░░░░░░░░░░] │  ← 8px tall bar
│   red   orange  yel  l.grn    green          │  ← color zones
└──────────────────────────────────────────────┘
  200px wide

Size 'sm' (TickerFundamentals sidebar):
┌──────────────────────────────┐
│  [▓▓▓░░░░●░░░░░░░░░░░░░░░░] │  ← 6px tall bar
└──────────────────────────────┘
  140px wide
```

**Color Zones (for "higher is better" metrics — ROE, margins):**

| Zone | Percentile Range | Color (Tailwind) | Meaning |
|---|---|---|---|
| Bottom | 0-25 | `bg-red-500/30` | Below average |
| Below median | 25-40 | `bg-orange-500/30` | Slightly below average |
| Median | 40-60 | `bg-yellow-500/30` | Average |
| Above median | 60-75 | `bg-green-400/30` | Above average |
| Top | 75-100 | `bg-green-500/30` | Well above average |

**For "lower is better" metrics (P/E, D/E): Colors reverse** — the API already inverts the percentile, so low percentile = bad. The component always treats higher percentile as greener.

**Marker:** A solid dot (`bg-ic-blue`, 10px circle) positioned at the stock's percentile.

**Hover Tooltip:**
```
P/E Ratio: 28.5
Percentile: 35th in Technology (n=245)
Below sector median of 25.7
Lower is better for P/E — this stock is cheaper than 65% of peers
```

**Implementation Notes:**
- Render as a single `<div>` with 5 colored segments using CSS `linear-gradient` or 5 adjacent `<div>` children
- Marker positioned via `left: ${percentile}%` with absolute positioning
- Distribution markers (p25, p50, p75) shown as subtle tick marks on the bar
- When `visible === false`: render a blurred placeholder with lock icon (freemium gating)

### 3.2 `useSectorPercentiles` — Data Fetching Hook

**File:** `lib/hooks/useSectorPercentiles.ts`

```typescript
interface UseSectorPercentilesResult {
  data: SectorPercentilesResponse | null;
  loading: boolean;
  error: string | null;
  getMetricPercentile: (metricName: string) => MetricPercentileData | null;
}

function useSectorPercentiles(ticker: string): UseSectorPercentilesResult {
  // 1. Fetch from /api/v1/stocks/:ticker/sector-percentiles
  // 2. Cache in component state (no global store needed — re-fetches per ticker page)
  // 3. Provide helper to look up individual metric data
}
```

**Caching:** Since `TickerFundamentals`, `MetricsTab`, and `KeyStatsTab` all need the same data, lift the fetch to the ticker page level and pass down via props or context. Options:

**Option A (Required):** Create a `SectorPercentilesProvider` context that wraps the ticker page content. All child components consume via `useSectorPercentiles()` context hook. Single API call, shared data. This is required (not optional) because all three surfaces (`TickerFundamentals`, `MetricsTab`, `KeyStatsTab`) render concurrently on the same page — Option B would triple API calls with no benefit.

```typescript
// In app/ticker/[symbol]/page.tsx (or a client wrapper)
<SectorPercentilesProvider ticker={symbol}>
  <TickerFundamentals symbol={symbol} />
  <TickerTabs symbol={symbol} />
</SectorPercentilesProvider>
```

**Option B:** Fetch in each component independently (simpler but 3x API calls). Acceptable if the endpoint is fast (<100ms) and cached on the backend.

### 3.3 TypeScript Types

**File:** `lib/types/fundamentals.ts` (new file)

```typescript
export interface SectorPercentilesResponse {
  ticker: string;
  sector: string;
  calculated_at: string;
  sample_count: number;
  metrics: Record<string, MetricPercentileData>;
  meta: {
    source: string;
    metric_count: number;
    timestamp: string;
  };
}

export interface MetricPercentileData {
  value: number;
  percentile: number;
  lower_is_better: boolean;
  distribution: PercentileDistribution;
  sample_count: number;
}

export interface PercentileDistribution {
  min: number;
  p10: number;
  p25: number;
  p50: number;
  p75: number;
  p90: number;
  max: number;
}
```

---

## 4. Integration Points

### 4.1 `TickerFundamentals` Sidebar Integration

**File:** `components/ticker/TickerFundamentals.tsx`

**Current structure (per metric):**
```tsx
<div className="flex justify-between">
  <span className="text-ic-text-dim">P/E Ratio</span>
  <span className="text-ic-text-primary font-medium">28.5</span>
</div>
```

**Enhanced structure:**
```tsx
<div className="space-y-1">
  <div className="flex justify-between">
    <span className="text-ic-text-dim">P/E Ratio</span>
    <span className="text-ic-text-primary font-medium">28.5</span>
  </div>
  {percentileData && (
    <SectorPercentileBar
      percentile={percentileData.percentile}
      distribution={percentileData.distribution}
      value={percentileData.value}
      lowerIsBetter={percentileData.lower_is_better}
      metricName="P/E Ratio"
      sector={sectorData.sector}
      sampleCount={percentileData.sample_count}
      size="sm"
      visible={isFreeMetric('pe_ratio') || isPremium}
    />
  )}
</div>
```

**Metrics to display in sidebar (6 free, 8 premium):**

| Metric | API Key | Free? |
|---|---|---|
| P/E Ratio | `pe_ratio` | Yes |
| ROE | `roe` | Yes |
| Gross Margin | `gross_margin` | Yes |
| Debt/Equity | `debt_to_equity` | Yes |
| Revenue Growth | `revenue_growth_yoy` | Yes |
| Current Ratio | `current_ratio` | Yes |
| P/B Ratio | `pb_ratio` | Premium |
| P/S Ratio | `ps_ratio` | Premium |
| ROA | `roa` | Premium |
| Operating Margin | `operating_margin` | Premium |
| Net Margin | `net_margin` | Premium |
| EV/EBITDA | `ev_ebitda` | Premium |
| Interest Coverage | `interest_coverage` | Premium |
| EPS Growth | `eps_growth_yoy` | Premium |

**Metric name to API key mapping:**

```typescript
const METRIC_API_KEYS: Record<string, string> = {
  'P/E Ratio': 'pe_ratio',
  'P/B Ratio': 'pb_ratio',
  'P/S Ratio': 'ps_ratio',
  'EV/EBITDA': 'ev_ebitda',
  'ROE': 'roe',
  'ROA': 'roa',
  'Gross Margin': 'gross_margin',
  'Operating Margin': 'operating_margin',
  'Net Margin': 'net_margin',
  'Debt/Equity': 'debt_to_equity',
  'Current Ratio': 'current_ratio',
  'Interest Coverage': 'interest_coverage',
  'Revenue Growth (YoY)': 'revenue_growth_yoy',
  'EPS Growth (YoY)': 'eps_growth_yoy',
};

const FREE_TIER_METRICS = new Set([
  'pe_ratio', 'roe', 'gross_margin', 'debt_to_equity', 'revenue_growth_yoy', 'current_ratio',
]);
```

### 4.2 `MetricsTab` Integration

**File:** `components/ticker/tabs/MetricsTab.tsx`

The MetricsTab already has structured category sub-tabs (`valuation`, `profitability`, `financial_health`, etc.) and renders metrics with `formatMetricValue()` from `types/metrics.ts`.

**Integration approach:** After each `MetricCard`-style row, add the percentile bar. The MetricsTab already fetches `ComprehensiveMetricsResponse` — the sector percentile data can be passed alongside.

### 4.3 `KeyStatsTab` Integration

**File:** `components/ticker/tabs/KeyStatsTab.tsx`

KeyStatsTab displays 50+ metrics in `GroupCard` sections. Add percentile bars to key metrics within each group. Not all 50 metrics need bars — only those in `TrackedMetrics` (the list of metrics for which sector percentile data exists).

---

## 5. Responsive Behavior

| Breakpoint | Bar Behavior |
|---|---|
| Desktop (≥1024px) | Full bar with color zones, marker, distribution ticks. 200px wide. |
| Tablet (768-1023px) | Simplified bar: no distribution ticks, no zone labels. 160px wide. |
| Mobile (<768px) | Dot indicator only: colored circle positioned in a thin line. 100px wide. |

**Implementation:** Use Tailwind responsive classes:
```tsx
<div className="hidden md:block">
  {/* Full bar */}
</div>
<div className="block md:hidden">
  {/* Simplified dot indicator */}
</div>
```

## 6. Accessibility

- **Screen reader text:** Each bar includes `aria-label`:
  ```
  "P/E Ratio: 28.5, 35th percentile in Technology sector. Below sector median of 25.7. Lower is better — this stock is cheaper than 65% of peers."
  ```
- **Not color-only:** The marker dot is always visible; tooltip provides full context. Users with color blindness can rely on percentile number.
- **Keyboard accessible:** Tooltip triggered on focus (tab navigation) in addition to hover.
- **Reduced motion:** Disable any bar entrance animations when `prefers-reduced-motion` is set.

## 7. New Files

| File | Purpose | Est. LOC |
|---|---|---|
| `components/ui/SectorPercentileBar.tsx` | Core reusable bar component | ~180 |
| `lib/hooks/useSectorPercentiles.ts` | Data fetching hook | ~60 |
| `lib/types/fundamentals.ts` | TypeScript interfaces for all fundamentals API responses | ~120 |
| `lib/contexts/SectorPercentilesContext.tsx` | Context provider for shared data | ~50 |

## 8. Modified Files

| File | Change |
|---|---|
| `components/ticker/TickerFundamentals.tsx` | Add SectorPercentileBar to 14 metrics |
| `components/ticker/tabs/MetricsTab.tsx` | Add SectorPercentileBar to metric rows |
| `components/ticker/tabs/KeyStatsTab.tsx` | Add SectorPercentileBar to key metrics |
| `app/ticker/[symbol]/page.tsx` | Wrap content in SectorPercentilesProvider |
| `lib/api/routes.ts` | Add `sectorPercentiles` route (if not done in Project 1) |

## 9. Performance Budget

| Metric | Target |
|---|---|
| Additional JS bundle size for `SectorPercentileBar` | <3 KB gzipped |
| Render time per bar instance | <2ms |
| Total added render time (14 bars in sidebar) | <30ms |
| API call latency (sector-percentiles endpoint) | <100ms |
| No layout shift on bar load | CLS = 0 (reserve space with fixed height) |

**Layout shift prevention:** Always render a placeholder `<div>` with the bar's exact height (8px or 6px) even during loading, using a subtle skeleton animation (`animate-pulse bg-ic-border`).

## 10. Acceptance Criteria

- [ ] `SectorPercentileBar` renders correctly for all 14 Phase 1 metrics
- [ ] Color coding is direction-aware (low P/E = green zone, low ROE = red zone)
- [ ] Hover tooltip shows: value, percentile, sector name, sample size, median comparison
- [ ] Bar renders on TickerFundamentals sidebar (6 free, 8 premium with blur)
- [ ] Bar renders on MetricsTab for all available metrics
- [ ] Bar renders on KeyStatsTab for key metrics
- [ ] Graceful fallback when sector percentile data is unavailable (metric renders without bar)
- [ ] Mobile responsive: dot-only indicator on screens <768px
- [ ] Screen reader accessible with descriptive aria-label
- [ ] No layout shift during loading (skeleton placeholder)
- [ ] Single API call shared across all three surfaces via context provider
