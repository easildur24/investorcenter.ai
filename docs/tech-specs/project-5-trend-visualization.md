# Tech Spec: Project 5 — Trend Visualization (Sparklines + History)

**Parent PRD:** `docs/prd-enhanced-fundamentals-experience.md`
**Priority:** P1
**Estimated Effort:** 2-3 sprints
**Dependencies:** Project 1 (metric-history API endpoint)

---

## 1. Overview

Build inline `TrendSparkline` components (20px tall mini-charts) for 10+ key financial metrics and a `MetricHistoryChart` full-width expandable chart for detailed 5-year quarterly analysis. Sparklines show at a glance whether metrics are improving or deteriorating. This is a data visualization project with a focus on performance (lazy loading, efficient rendering).

## 2. Architecture Context

### Chart Libraries Available (from `package.json`)

| Library | Usage Pattern | Best For |
|---|---|---|
| **Recharts** | Primary chart lib across the app (IC Score charts, earnings, sentiment) | Full charts with axes, tooltips, legends |
| **Chart.js + react-chartjs-2** | Secondary (some chart types) | Canvas-based charts |
| **D3** | Installed (`@types/d3`), not heavily used | Custom SVG visualizations |

**Recommendation for sparklines:** Use **SVG path rendering** (no library) for sparklines — Recharts/Chart.js add too much overhead for a 20px-tall inline element rendered 10+ times on a page. For the full `MetricHistoryChart`, use **Recharts** (consistent with existing chart patterns like `ICScoreHistoryChart`, `EarningsBarChart`).

### Data Source

**API:** `GET /api/v1/stocks/:ticker/metric-history/:metric` (Project 1)

**Response shape:**
```typescript
interface MetricHistoryResponse {
  ticker: string;
  metric: string;
  timeframe: 'quarterly' | 'annual';
  unit: 'USD' | 'percent' | 'ratio';
  data_points: Array<{
    period_end: string;
    fiscal_year: number;
    fiscal_quarter: number;
    value: number;
    yoy_change: number | null;
  }>;
  trend: {
    direction: 'up' | 'down' | 'flat';
    slope: number;
    consecutive_growth_quarters: number;
  };
}
```

### Existing Sparkline-Adjacent Patterns

The codebase doesn't have sparklines yet, but has chart patterns in:
- `components/ic-score/ICScoreHistoryChart.tsx` — Recharts line chart
- `components/ticker/HybridChart.tsx` — Custom canvas chart for prices
- `components/backtest/CumulativeReturnsChart.tsx` — Recharts area chart

---

## 3. Component Design

### 3.1 `TrendSparkline` — Inline Mini Chart

**File:** `components/ui/TrendSparkline.tsx`

**Props:**

```typescript
interface TrendSparklineProps {
  /** Array of values (oldest first) */
  values: number[];
  /** Trend direction for color */
  trend: 'up' | 'down' | 'flat';
  /** Whether higher values are better (ROE: true, D/E: false) */
  higherIsBetter: boolean;
  /** Width in pixels */
  width?: number;
  /** Height in pixels */
  height?: number;
  /** Hover data for tooltip */
  hoverData?: Array<{ label: string; value: number }>;
  /** Click handler (expand to full chart) */
  onClick?: () => void;
  /** Premium gating */
  visible?: boolean;
}
```

**Visual Design:**

```
Normal (up, higher is better): Green line
───────╱──╱╱───╱╱──     (20px tall, 80px wide)

Normal (down, higher is better): Red line
──╲──╲───╲╲──╲╲────    (20px tall, 80px wide)

Flat (within ±5% band): Gray line
────────────────────    (20px tall, 80px wide)
```

**SVG Implementation (no library dependency):**

```typescript
function TrendSparkline({ values, trend, higherIsBetter, width = 80, height = 20, ...props }: TrendSparklineProps) {
  if (values.length < 2) return null;

  // Normalize values to 0-height range
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;

  const points = values.map((v, i) => ({
    x: (i / (values.length - 1)) * width,
    y: height - ((v - min) / range) * (height - 4) - 2, // 2px padding top/bottom
  }));

  // Build SVG path
  const pathD = points
    .map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x.toFixed(1)} ${p.y.toFixed(1)}`)
    .join(' ');

  // Determine color
  const isPositive = (trend === 'up' && higherIsBetter) || (trend === 'down' && !higherIsBetter);
  const isNegative = (trend === 'down' && higherIsBetter) || (trend === 'up' && !higherIsBetter);
  const strokeColor = isPositive ? '#4ade80' : isNegative ? '#f87171' : '#94a3b8';

  return (
    <svg
      width={width}
      height={height}
      className={cn('inline-block cursor-pointer', !props.visible && 'blur-sm')}
      onClick={props.onClick}
      role="img"
      aria-label={`Trend: ${trend} over ${values.length} quarters`}
    >
      <path d={pathD} fill="none" stroke={strokeColor} strokeWidth={1.5} strokeLinecap="round" />
      {/* End dot */}
      <circle cx={points[points.length - 1].x} cy={points[points.length - 1].y} r={2} fill={strokeColor} />
    </svg>
  );
}
```

**Hover Tooltip:**

When the user hovers over the sparkline, show a small tooltip with quarterly values:

```
Revenue (5Y Quarterly)
Q1'22: $97.3B
Q2'22: $83.0B
Q3'22: $90.1B
...
Q1'26: $124.3B (latest)
Trend: ▲ Up, 6 consecutive growth quarters
```

Implementation: Wrap in the existing `Tooltip` component. On hover, display the full data list. The sparkline itself doesn't handle hover positioning (too small) — the tooltip wraps the entire sparkline element.

### 3.2 `MetricHistoryChart` — Full Expandable Chart

**File:** `components/ticker/MetricHistoryChart.tsx`

**Props:**

```typescript
interface MetricHistoryChartProps {
  ticker: string;
  metric: string;
  metricLabel: string;
  unit: 'USD' | 'percent' | 'ratio';
  /** Initial data from sparkline (if available, avoids refetch) */
  initialData?: MetricHistoryResponse;
  onClose: () => void;
}
```

**Layout (overlay/modal):**

```
┌──────────────────────────────────────────────────────────────┐
│  Revenue — AAPL                                         [✕]  │
│  5-Year Quarterly History                                    │
│                                                              │
│  $130B ┤                                          ╱         │
│  $120B ┤                                     ╱╱╱╱           │
│  $110B ┤                              ╱╱╱╱╱╱               │
│  $100B ┤                    ╱╱╱╱╱╱╱╱╱                      │
│   $90B ┤         ╱╱╱╱╱╱╱╱╱                                 │
│   $80B ┤ ╱╱╱╱╱╱╱                                           │
│   $70B ┼─────────┼──────────┼──────────┼──────────┼─────────│
│        2022      2023       2024       2025       2026       │
│                                                              │
│  [Quarterly ▼]   [Overlay: Sector Median ☐] [Peer Avg ☐]   │
│                                                              │
│  Latest: $124.3B (Q1'26) │ YoY: +5.1% │ 5Y CAGR: 8.3%     │
└──────────────────────────────────────────────────────────────┘
```

**Implementation:** Use Recharts `AreaChart` or `LineChart`:

```tsx
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';

function MetricHistoryChart({ ticker, metric, metricLabel, unit, initialData, onClose }: MetricHistoryChartProps) {
  const [data, setData] = useState(initialData);
  const [timeframe, setTimeframe] = useState<'quarterly' | 'annual'>('quarterly');

  useEffect(() => {
    if (!data) {
      fetchMetricHistory(ticker, metric, timeframe).then(setData);
    }
  }, [ticker, metric, timeframe]);

  return (
    <div className="bg-ic-surface rounded-lg border border-ic-border p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-medium">{metricLabel} — {ticker}</h3>
        <button onClick={onClose}>
          <XMarkIcon className="h-5 w-5" />
        </button>
      </div>

      <ResponsiveContainer width="100%" height={300}>
        <AreaChart data={data.data_points}>
          <XAxis dataKey="period_end" tickFormatter={formatQuarterLabel} />
          <YAxis tickFormatter={(v) => formatMetricValue(v, unit)} />
          <Tooltip content={<CustomTooltip unit={unit} />} />
          <Area
            type="monotone"
            dataKey="value"
            stroke="#3b82f6"
            fill="#3b82f6"
            fillOpacity={0.1}
          />
        </AreaChart>
      </ResponsiveContainer>

      {/* Summary stats */}
      <div className="mt-3 flex gap-4 text-sm text-ic-text-secondary">
        <span>Latest: {formatMetricValue(latestValue, unit)}</span>
        <span>YoY: {formatPercent(latestYoY)}</span>
        <span>5Y CAGR: {formatPercent(cagr5Y)}</span>
      </div>
    </div>
  );
}
```

### 3.3 Sparkline-to-Chart Interaction

When a user clicks a sparkline, the full `MetricHistoryChart` appears. Two options:

**Option A (Recommended): Inline expansion** — the sparkline row expands to show the full chart directly below the metric. Uses CSS transition for smooth expansion. Chart appears in the same column/area.

**Option B: Modal overlay** — the chart appears as a centered modal. Better for mobile but disconnects from the metric context.

```tsx
// Option A implementation:
const [expandedMetric, setExpandedMetric] = useState<string | null>(null);

// In the metric row:
<div className="space-y-1">
  <div className="flex justify-between items-center">
    <span>Revenue</span>
    <div className="flex items-center gap-2">
      <span>$124.3B</span>
      <TrendSparkline
        values={revenueHistory}
        trend="up"
        higherIsBetter={true}
        onClick={() => setExpandedMetric('revenue')}
      />
    </div>
  </div>
  {expandedMetric === 'revenue' && (
    <MetricHistoryChart
      ticker={ticker}
      metric="revenue"
      metricLabel="Revenue"
      unit="USD"
      onClose={() => setExpandedMetric(null)}
    />
  )}
</div>
```

---

## 4. Metrics with Sparklines

| Metric | API Key | Unit | Higher is Better | Surface |
|---|---|---|---|---|
| Revenue | `revenue` | USD | Yes | Sidebar, MetricsTab |
| Net Income | `net_income` | USD | Yes | MetricsTab |
| Free Cash Flow | `free_cash_flow` | USD | Yes | MetricsTab |
| Gross Margin | `gross_margin` | percent | Yes | Sidebar, MetricsTab |
| Operating Margin | `operating_margin` | percent | Yes | MetricsTab |
| Net Margin | `net_margin` | percent | Yes | Sidebar, MetricsTab |
| ROE | `roe` | percent | Yes | Sidebar, MetricsTab |
| ROA | `roa` | percent | Yes | MetricsTab |
| Debt/Equity | `debt_to_equity` | ratio | No | Sidebar, MetricsTab |
| EPS | `eps` | USD | Yes | Sidebar, MetricsTab |
| Current Ratio | `current_ratio` | ratio | Yes (generally) | MetricsTab |

---

## 5. Data Fetching Strategy

### 5.1 Batch Fetch for Sparklines

Fetching metric history one-by-one for 10+ sparklines would be 10+ API calls on page load. Instead:

**Option A (Recommended): New batch endpoint**

Add a batch endpoint to Project 1:

`GET /api/v1/stocks/:ticker/metric-history?metrics=revenue,net_income,gross_margin,...&limit=20`

Returns all requested metrics in one response:

```json
{
  "ticker": "AAPL",
  "timeframe": "quarterly",
  "metrics": {
    "revenue": { "values": [...], "trend": {...} },
    "net_income": { "values": [...], "trend": {...} },
    ...
  }
}
```

This reduces 10+ API calls to 1 call. **Add this to Project 1 scope.**

**Option B: Client-side parallel fetch with concurrency limit**

```typescript
async function fetchSparklineData(ticker: string, metrics: string[]) {
  const results = await Promise.allSettled(
    metrics.map(metric =>
      fetch(`${API_BASE_URL}${stocks.metricHistory(ticker, metric)}?limit=20`)
        .then(r => r.json())
    )
  );
  // Process settled results
}
```

### 5.2 `useSparklineData` Hook

**File:** `lib/hooks/useSparklineData.ts`

```typescript
interface SparklineDataMap {
  [metricKey: string]: {
    values: number[];
    trend: 'up' | 'down' | 'flat';
    latestValue: number;
    yoyChange: number | null;
  };
}

function useSparklineData(ticker: string): {
  data: SparklineDataMap | null;
  loading: boolean;
  error: string | null;
}
```

### 5.3 Lazy Loading

Sparklines are **lazy-loaded on scroll** using Intersection Observer:

```typescript
function LazySparkline({ ticker, metric, ...props }: LazySparklineProps) {
  const ref = useRef<HTMLDivElement>(null);
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true);
          observer.disconnect();
        }
      },
      { rootMargin: '100px' } // Start loading 100px before visible
    );

    if (ref.current) observer.observe(ref.current);
    return () => observer.disconnect();
  }, []);

  return (
    <div ref={ref} style={{ width: 80, height: 20 }}>
      {isVisible ? (
        <TrendSparkline {...props} />
      ) : (
        <div className="w-full h-full bg-ic-border/30 rounded animate-pulse" />
      )}
    </div>
  );
}
```

**However**, if using the batch endpoint (Option A), all sparkline data arrives in one call. In that case, lazy loading the *data fetch* isn't needed — only lazy loading the *render* matters. Since SVG sparklines are very lightweight to render (<2ms each), we can render them all immediately once data arrives. Skip Intersection Observer in this case.

---

## 6. Integration Points

### 6.1 `TickerFundamentals` Sidebar

Add sparklines next to metric values for the 6 metrics that have sparklines in the sidebar:

```tsx
<div className="flex justify-between items-center">
  <span className="text-ic-text-dim">Revenue</span>
  <div className="flex items-center gap-2">
    <span className="text-ic-text-primary font-medium">$124.3B</span>
    <TrendSparkline
      values={sparklineData?.revenue?.values ?? []}
      trend={sparklineData?.revenue?.trend ?? 'flat'}
      higherIsBetter={true}
      visible={isPremium}
      onClick={() => setExpandedMetric('revenue')}
    />
  </div>
</div>
```

### 6.2 `MetricsTab`

Add sparklines to metric rows in each category sub-tab. The MetricsTab already has a structured layout with metric config objects — extend each config to include sparkline metadata:

```typescript
// In types/metrics.ts, extend MetricDisplayConfig:
interface MetricDisplayConfig {
  // ... existing fields ...
  sparklineKey?: string;        // API key for metric history (if sparkline available)
  higherIsBetter?: boolean;     // Direction for sparkline coloring
}
```

---

## 7. Responsive Behavior

| Breakpoint | Sparkline | Full Chart |
|---|---|---|
| Desktop (≥1024px) | 80px wide × 20px tall, inline | Full-width inline expansion (300px tall) |
| Tablet (768-1023px) | 60px wide × 16px tall | Full-width inline expansion (250px tall) |
| Mobile (<768px) | 50px wide × 14px tall | Full-width modal (fills screen width, 200px tall) |

## 8. Accessibility

- **Sparkline `aria-label`:** `"Revenue trend: up over 20 quarters, latest value $124.3 billion"`
- **Click action:** `role="button"` with `aria-label="Expand revenue history chart"`
- **Sparkline alt text summarizes:** direction, time span, latest value
- **Full chart:** Recharts has built-in accessibility; add `aria-label` to container
- **Reduced motion:** When `prefers-reduced-motion` is set, render static sparklines without any entrance animation

## 9. Freemium Gating

| Feature | Free | Premium |
|---|---|---|
| Sparklines | Blurred (placeholder) | Yes |
| Full history chart | Not accessible | Yes |

Free tier shows sparkline placeholders (gray blurred line) with lock icon to indicate premium content.

## 10. New Files

| File | Purpose | Est. LOC |
|---|---|---|
| `components/ui/TrendSparkline.tsx` | Inline SVG sparkline component | ~120 |
| `components/ticker/MetricHistoryChart.tsx` | Full Recharts history chart | ~200 |
| `lib/hooks/useSparklineData.ts` | Batch sparkline data hook | ~70 |

## 11. Modified Files

| File | Change |
|---|---|
| `components/ticker/TickerFundamentals.tsx` | Add sparklines to 6 sidebar metrics |
| `components/ticker/tabs/MetricsTab.tsx` | Add sparklines to metric rows |
| `types/metrics.ts` | Extend `MetricDisplayConfig` with sparkline fields |
| `lib/types/fundamentals.ts` | Add `MetricHistoryResponse` type |
| `lib/api/routes.ts` | Add `metricHistory` route |

## 12. Performance Budget

| Metric | Target |
|---|---|
| `TrendSparkline` render time (single) | <1ms |
| Total sparkline render time (10 instances) | <10ms |
| SVG sparkline file size (component) | <2 KB gzipped |
| Batch API call for all sparkline data | <200ms |
| `MetricHistoryChart` render time (Recharts) | <100ms |
| No LCP impact (sparklines not in viewport on load) | 0ms added to LCP |
| CLS prevention | Fixed-size placeholder (80×20px) always present |

## 13. Acceptance Criteria

- [ ] `TrendSparkline` renders as inline SVG (20px tall) next to metric values
- [ ] Sparkline color is direction-aware: green for improving (considering higher/lower-is-better), red for deteriorating
- [ ] Hover tooltip shows quarterly values for the full time range
- [ ] Click on sparkline expands to full `MetricHistoryChart` with Recharts
- [ ] Full chart shows 5-year quarterly data with proper axis formatting
- [ ] Full chart displays summary stats (latest, YoY, 5Y CAGR)
- [ ] Sparklines appear on 10+ metrics across sidebar and MetricsTab
- [ ] Batch API endpoint fetches all sparkline data in single call
- [ ] Free tier: blurred sparkline placeholders with lock indicator
- [ ] Premium tier: full sparklines + expandable charts
- [ ] Mobile: smaller sparklines (50×14px), charts fill screen width
- [ ] Screen reader describes trend direction and latest value
- [ ] No layout shift from sparkline loading (fixed-size placeholder)
