# Tech Spec: Earnings Feature

**Author:** Engineering Team
**Date:** 2026-02-27
**Status:** Draft
**PRD Reference:** `docs/earnings-prd.md`

---

## 1. Architecture Overview

The Earnings feature follows InvestorCenter's existing three-tier pattern:

```
Frontend (Next.js 14)               Backend (Go/Gin)              FMP API
────────────────────────            ─────────────────            ──────────
EarningsTab component      →   GET /stocks/:ticker/earnings  →  /stable/earnings
EarningsCalendarPage       →   GET /earnings-calendar         →  /stable/earnings-calendar
                                        ↕
                                  Redis (cache layer)
                                  TTL: 1hr (per-stock)
                                  TTL: 4hr (calendar)
```

### Files to Create
| File | Purpose |
|------|---------|
| `backend/handlers/earnings.go` | HTTP handlers for earnings endpoints |
| `backend/services/fmp_earnings.go` | FMP client methods for earnings data |
| `components/ticker/tabs/EarningsTab.tsx` | Main Earnings tab component |
| `components/ticker/tabs/earnings/NextEarningsCard.tsx` | Upcoming earnings card |
| `components/ticker/tabs/earnings/EarningsSummaryStats.tsx` | Beat rate stats |
| `components/ticker/tabs/earnings/EarningsTable.tsx` | History table |
| `components/ticker/tabs/earnings/EarningsBarChart.tsx` | EPS/Revenue bar charts |
| `app/earnings-calendar/page.tsx` | Earnings Calendar page |
| `components/earnings-calendar/EarningsCalendarDateStrip.tsx` | Date navigation strip |
| `components/earnings-calendar/EarningsCalendarTable.tsx` | Calendar results table |
| `lib/api/earnings.ts` | Frontend API client for earnings |

### Files to Modify
| File | Change |
|------|--------|
| `backend/main.go` | Add route definitions (~line 157) |
| `lib/api/routes.ts` | Add earnings route paths |
| `app/ticker/[symbol]/page.tsx` | Add Earnings tab to `stockTabs` array and children |
| `components/Header.tsx` | Add "Earnings Calendar" link to main nav |

---

## 2. API Integration Layer

### 2.1 FMP `/stable/earnings` — Per-Stock Earnings History

**Request:**
```
GET https://financialmodelingprep.com/stable/earnings?symbol=AAPL&apikey={FMP_API_KEY}
```

**Response Schema:**
```json
[
  {
    "symbol": "AAPL",
    "date": "2025-10-30",
    "epsActual": 1.64,
    "epsEstimated": 1.60,
    "revenueActual": 94930000000,
    "revenueEstimated": 94500000000,
    "lastUpdated": "2025-11-01T12:00:00.000Z"
  },
  {
    "symbol": "AAPL",
    "date": "2026-01-30",
    "epsActual": null,
    "epsEstimated": 2.35,
    "revenueActual": null,
    "revenueEstimated": 124100000000,
    "lastUpdated": "2026-01-15T08:00:00.000Z"
  }
]
```

**Go struct:**
```go
// backend/services/fmp_earnings.go

type FMPEarningsRecord struct {
    Symbol           string   `json:"symbol"`
    Date             string   `json:"date"`
    EPSActual        *float64 `json:"epsActual"`
    EPSEstimated     *float64 `json:"epsEstimated"`
    RevenueActual    *float64 `json:"revenueActual"`
    RevenueEstimated *float64 `json:"revenueEstimated"`
    LastUpdated      string   `json:"lastUpdated"`
}
```

**Error Handling:**
- HTTP 401/403: Log "FMP API key invalid or expired", return 503 to client
- HTTP 429: Log "FMP rate limit exceeded", return 503 with `Retry-After` header
- HTTP 5xx: Return 502 "Upstream service unavailable"
- Empty array response: Return valid empty `earnings` array (not an error)
- Network timeout (10s): Return 504 "Gateway timeout"

### 2.2 FMP `/stable/earnings-calendar` — Site-Wide Calendar

**Request:**
```
GET https://financialmodelingprep.com/stable/earnings-calendar?from=2026-02-23&to=2026-03-07&apikey={FMP_API_KEY}
```

**Response:** Same schema as above, but returns all tickers in the date range. Max 4000 records, max 90-day window.

**Pagination Strategy:**
- Fetch 2 weeks at a time (current week + next week) per request
- If user navigates to a different week, fetch that week + the following week
- Never request more than 14-day windows to stay well under the 4000-record limit

### 2.3 Caching Strategy (Redis)

| Cache Key Pattern | TTL | Reason |
|-------------------|-----|--------|
| `earnings:stock:{SYMBOL}` | 1 hour | Earnings data changes infrequently; 1hr balances freshness with FMP rate budget |
| `earnings:calendar:{FROM}:{TO}` | 4 hours | Calendar data for a date range changes once per day at most |

**Implementation:**
```go
// In handler, before FMP call:
cacheKey := fmt.Sprintf("earnings:stock:%s", ticker)
cached, err := redisClient.Get(ctx, cacheKey).Result()
if err == nil {
    // Return cached response
    c.Data(http.StatusOK, "application/json", []byte(cached))
    return
}

// After successful FMP call:
redisClient.Set(ctx, cacheKey, responseJSON, 1*time.Hour)
```

**Cache Invalidation:**
- No explicit invalidation needed — TTL expiration is sufficient
- During earnings season (Jan/Apr/Jul/Oct), consider reducing TTL to 30 minutes

### 2.4 Fallback Behavior

When FMP returns null values for specific fields:
| Field | Null Meaning | Frontend Display |
|-------|-------------|------------------|
| `epsActual` | Not yet reported | "-" |
| `epsEstimated` | No analyst coverage | "N/A" |
| `revenueActual` | Not yet reported | "-" |
| `revenueEstimated` | No analyst coverage | "N/A" |
| `date` | Should never be null | Skip record |

---

## 3. Data Transformation Layer

### 3.1 Computed Fields (Backend — Go)

All computations happen in the backend handler before sending to the frontend:

```go
type EarningsResult struct {
    Symbol               string   `json:"symbol"`
    Date                 string   `json:"date"`
    FiscalQuarter        string   `json:"fiscalQuarter"`        // "Q1 '26"
    EPSEstimated         *float64 `json:"epsEstimated"`
    EPSActual            *float64 `json:"epsActual"`
    EPSSurprisePercent   *float64 `json:"epsSurprisePercent"`   // computed
    EPSBeat              *bool    `json:"epsBeat"`              // computed
    RevenueEstimated     *float64 `json:"revenueEstimated"`
    RevenueActual        *float64 `json:"revenueActual"`
    RevenueSurprisePercent *float64 `json:"revenueSurprisePercent"` // computed
    RevenueBeat          *bool    `json:"revenueBeat"`          // computed
    IsUpcoming           bool     `json:"isUpcoming"`           // computed
}

type EarningsResponse struct {
    Earnings     []EarningsResult `json:"earnings"`
    NextEarnings *EarningsResult  `json:"nextEarnings"` // first upcoming or most recent
    BeatRate     *BeatRate        `json:"beatRate"`
    Meta         ResponseMeta     `json:"meta"`
}

type BeatRate struct {
    EPSBeats       int `json:"epsBeats"`
    RevenueBeats   int `json:"revenueBeats"`
    TotalQuarters  int `json:"totalQuarters"`  // quarters with both actual + estimate
}
```

### 3.2 Surprise Calculation

```go
func computeSurprisePercent(actual, estimated *float64) *float64 {
    if actual == nil || estimated == nil {
        return nil
    }
    if *estimated == 0 {
        // Avoid division by zero — return absolute difference
        diff := *actual - *estimated
        return &diff
    }
    surprise := (*actual - *estimated) / math.Abs(*estimated) * 100
    return &surprise
}

func computeBeat(actual, estimated *float64) *bool {
    if actual == nil || estimated == nil {
        return nil
    }
    beat := *actual > *estimated
    return &beat
}
```

### 3.3 Fiscal Quarter Labeling

```go
func toFiscalQuarter(dateStr string) string {
    t, err := time.Parse("2006-01-02", dateStr)
    if err != nil {
        return dateStr
    }
    month := t.Month()
    year := t.Year() % 100 // 2-digit year
    var q int
    switch {
    case month >= 1 && month <= 3:
        q = 1
    case month >= 4 && month <= 6:
        q = 2
    case month >= 7 && month <= 9:
        q = 3
    default:
        q = 4
    }
    return fmt.Sprintf("Q%d '%02d", q, year)
}
```

**Note:** This uses calendar quarters, not fiscal year quarters. This is acceptable because FMP's `date` field represents the earnings report date, not the fiscal period end. The fiscal quarter label is a display convenience — not used for computations.

### 3.4 Handling Future Quarters

Records where `epsActual == nil` AND `date > today`:
- Set `isUpcoming = true`
- Do not compute surprise or beat fields (leave as nil)
- The first such record (sorted by date ascending) becomes `nextEarnings`

If no future record exists, `nextEarnings` is the most recent past record (with a flag `"reported": true`).

---

## 4. Frontend Components

### 4.1 Component Tree

```
<EarningsTab symbol={symbol}>              // components/ticker/tabs/EarningsTab.tsx
  ├── <NextEarningsCard />                  // earnings/NextEarningsCard.tsx
  ├── <EarningsSummaryStats />              // earnings/EarningsSummaryStats.tsx
  ├── <EarningsTable />                     // earnings/EarningsTable.tsx
  ├── <EarningsBarChart type="eps" />       // earnings/EarningsBarChart.tsx
  └── <EarningsBarChart type="revenue" />   // earnings/EarningsBarChart.tsx

<EarningsCalendarPage>                      // app/earnings-calendar/page.tsx
  ├── <EarningsCalendarDateStrip />         // earnings-calendar/EarningsCalendarDateStrip.tsx
  ├── <CalendarFilters />                   // inline in page
  └── <EarningsCalendarTable />             // earnings-calendar/EarningsCalendarTable.tsx
```

### 4.2 `<EarningsTab />`

**Props:** `{ symbol: string }`
**Pattern:** Follows exact pattern of `KeyStatsTab.tsx` — fetches data in `useEffect`, manages loading/error/data state.

```tsx
'use client';
import { useState, useEffect } from 'react';
import NextEarningsCard from './earnings/NextEarningsCard';
import EarningsSummaryStats from './earnings/EarningsSummaryStats';
import EarningsTable from './earnings/EarningsTable';
import EarningsBarChart from './earnings/EarningsBarChart';

interface EarningsTabProps {
  symbol: string;
}

export default function EarningsTab({ symbol }: EarningsTabProps) {
  const [data, setData] = useState<EarningsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchEarnings = async () => {
      try {
        setLoading(true);
        setError(null);
        const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '/api/v1';
        const res = await fetch(`${API_BASE_URL}/stocks/${symbol}/earnings`);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        setData(json.data);
      } catch (err) {
        setError('Failed to load earnings data');
      } finally {
        setLoading(false);
      }
    };
    fetchEarnings();
  }, [symbol]);

  if (loading) return <EarningsTabSkeleton />;
  if (error || !data) return <EarningsTabError message={error} />;

  return (
    <div className="p-6 space-y-6">
      <NextEarningsCard data={data.nextEarnings} />
      <EarningsSummaryStats beatRate={data.beatRate} />
      <EarningsTable earnings={data.earnings} />
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        <EarningsBarChart data={data.earnings} type="eps" />
        <EarningsBarChart data={data.earnings} type="revenue" />
      </div>
    </div>
  );
}
```

### 4.3 `<NextEarningsCard />`

Prominent card at top of the tab.

```tsx
// Props: { data: EarningsResult | null }
// Layout:
//   ┌─────────────────────────────────────────────┐
//   │  📅 Next Earnings: March 15, 2026           │
//   │     Estimated · in 16 days                  │
//   │                                              │
//   │  EPS Estimate: $2.35    Revenue Est: $124.1B │
//   └─────────────────────────────────────────────┘

// Styling: bg-ic-bg-secondary rounded-lg p-6
// Date: text-xl font-semibold text-ic-text-primary
// Countdown: text-ic-blue font-medium
// "Estimated" badge: bg-yellow-500/20 text-yellow-400 text-xs px-2 py-0.5 rounded
// "Confirmed" badge: bg-green-500/20 text-green-400 text-xs px-2 py-0.5 rounded
```

**Countdown logic (frontend):**
```ts
const daysUntil = Math.ceil(
  (new Date(data.date).getTime() - Date.now()) / (1000 * 60 * 60 * 24)
);
const countdownText = daysUntil > 0 ? `in ${daysUntil} days` : 'Today';
```

### 4.4 `<EarningsSummaryStats />`

Row of stat cards using the established `MetricCard` grid pattern.

```tsx
// Props: { beatRate: BeatRate }
// Layout: grid grid-cols-2 md:grid-cols-4 gap-4
// Cards:
//   1. EPS Beat Rate:     "6 / 8 quarters" (75%)
//   2. Revenue Beat Rate: "7 / 8 quarters" (88%)
//   3. Avg EPS Surprise:  "+5.2%"
//   4. Avg Rev Surprise:  "+2.1%"
```

### 4.5 `<EarningsTable />`

Sortable, color-coded history table.

```tsx
// Props: { earnings: EarningsResult[] }
// Features:
//   - Quarterly/Annual toggle (state managed internally)
//   - Default: last 8 quarters, "Show More" loads up to 40
//   - Sortable columns (click header to sort)
//   - Color coding on surprise columns

// Table styling matches FinancialsTab:
//   - <table className="w-full text-sm">
//   - <th className="text-left text-ic-text-muted font-medium px-3 py-2">
//   - <td className="px-3 py-2 border-t border-ic-border/30">
//   - Surprise positive: text-green-400
//   - Surprise negative: text-red-400
//   - Horizontal scroll wrapper: overflow-x-auto
```

### 4.6 `<EarningsBarChart />`

Recharts bar chart for EPS or Revenue comparison.

```tsx
// Props: { data: EarningsResult[], type: 'eps' | 'revenue' }
// Library: recharts (already installed, used in backtest charts)
// Components used:
//   - <ResponsiveContainer width="100%" height={300}>
//   - <BarChart data={chartData}>
//   - <Bar dataKey="estimated" fill="transparent" stroke="#6B7280" />  (ghost bar)
//   - <Bar dataKey="actual" fill={beatColor} />  (green for beat, red for miss)
//   - <XAxis dataKey="fiscalQuarter" tick={{ fill: '#9CA3AF', fontSize: 12 }} />
//   - <YAxis tick={{ fill: '#9CA3AF', fontSize: 12 }} />
//   - <Tooltip content={<CustomTooltip />} />

// Zoom: buttons for 2Y / 5Y / All (filter data array by date range)
// Wrapper: bg-ic-bg-secondary rounded-lg p-4
// Title: text-base font-semibold text-ic-text-primary mb-4
```

**Revenue formatting for Y-axis:**
```ts
const formatRevenue = (value: number): string => {
  if (value >= 1e12) return `$${(value / 1e12).toFixed(1)}T`;
  if (value >= 1e9) return `$${(value / 1e9).toFixed(1)}B`;
  if (value >= 1e6) return `$${(value / 1e6).toFixed(0)}M`;
  return `$${value.toLocaleString()}`;
};
```

### 4.7 `<EarningsCalendarPage />`

**Route:** `app/earnings-calendar/page.tsx`

Server component wrapper that renders client components. Page metadata for SEO:

```tsx
export const metadata: Metadata = {
  title: 'Earnings Calendar | InvestorCenter',
  description: 'Browse upcoming earnings dates, EPS estimates, and revenue estimates for thousands of stocks.',
};
```

Main content is a client component (`'use client'`) that manages date selection, filters, and data fetching.

### 4.8 `<EarningsCalendarDateStrip />`

Horizontal weekday selector.

```tsx
// Props: {
//   selectedDate: Date,
//   onDateSelect: (date: Date) => void,
//   earningsCounts: Record<string, number>  // { "2026-02-27": 42, ... }
// }
// Layout:
//   ← [Mon 23 (12)] [Tue 24 (35)] [Wed 25 (42)] [Thu 26 (28)] [Fri 27 (19)] →
//
// Styling:
//   - Container: flex items-center gap-2 overflow-x-auto
//   - Date chip default: bg-ic-bg-secondary text-ic-text-muted px-4 py-2 rounded-lg
//   - Date chip selected: bg-ic-blue text-white
//   - Count badge: text-xs text-ic-text-dim
//   - Arrows: <button> with ChevronLeft/ChevronRight icons
```

### 4.9 `<EarningsCalendarTable />`

Filterable, sortable table of earnings for selected date(s).

```tsx
// Props: {
//   data: CalendarEarningsRecord[],
//   searchQuery: string,
//   marketCapFilter: 'all' | 'large' | 'mid' | 'small'
// }
// Client-side filtering:
//   - Search: filter by symbol or company name (case-insensitive includes)
//   - Market cap: large (>10B), mid (2B-10B), small (<2B)
// Symbol links to: /ticker/{SYMBOL}?tab=earnings
```

---

## 5. State Management

### Earnings Tab (Per-Stock)
- **Fetch trigger:** `useEffect` on `symbol` change (standard tab pattern)
- **State:** `useState` for `data`, `loading`, `error` — local to `EarningsTab`
- **No global state needed** — earnings data is only used within the Earnings tab
- **Sub-component state:** Chart zoom (2Y/5Y/All), table toggle (Quarterly/Annual), table pagination (Show More) — all local `useState`

### Earnings Calendar Page
- **Fetch trigger:** `useEffect` on `selectedWeekStart` change
- **State:**
  - `selectedDate: Date` — currently selected day in the date strip
  - `weekData: Record<string, CalendarEarningsRecord[]>` — earnings keyed by date string
  - `searchQuery: string` — search filter value
  - `marketCapFilter: string` — market cap filter value
  - `viewMode: 'daily' | 'weekly'` — toggle state
- **Data flow:** Fetch 2 weeks from backend → store in `weekData` → filter/sort client-side

### Data Fetching Pattern
```
Frontend                    Backend                     Redis           FMP
   |                           |                          |               |
   |  GET /stocks/AAPL/earnings|                          |               |
   |-------------------------->|                          |               |
   |                           | GET earnings:stock:AAPL  |               |
   |                           |------------------------->|               |
   |                           |     (cache miss)         |               |
   |                           |                          |               |
   |                           | GET /stable/earnings?symbol=AAPL         |
   |                           |----------------------------------------->|
   |                           |          [earnings data]                 |
   |                           |<-----------------------------------------|
   |                           |                          |               |
   |                           | SET earnings:stock:AAPL  |               |
   |                           |   (TTL: 1hr)             |               |
   |                           |------------------------->|               |
   |                           |                          |               |
   |    { data, meta }         |                          |               |
   |<--------------------------|                          |               |
```

---

## 6. Error States & Loading States

### Loading States

**EarningsTab skeleton:**
```tsx
function EarningsTabSkeleton() {
  return (
    <div className="p-6 animate-pulse space-y-6">
      {/* Next Earnings Card skeleton */}
      <div className="bg-ic-bg-secondary rounded-lg p-6">
        <div className="h-6 bg-ic-bg-tertiary rounded w-64 mb-3" />
        <div className="h-4 bg-ic-bg-tertiary rounded w-40 mb-4" />
        <div className="flex gap-8">
          <div className="h-5 bg-ic-bg-tertiary rounded w-32" />
          <div className="h-5 bg-ic-bg-tertiary rounded w-32" />
        </div>
      </div>
      {/* Beat rate skeleton */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {[1,2,3,4].map(i => (
          <div key={i} className="bg-ic-bg-secondary rounded-lg p-4">
            <div className="h-4 bg-ic-bg-tertiary rounded w-20 mb-2" />
            <div className="h-6 bg-ic-bg-tertiary rounded w-16" />
          </div>
        ))}
      </div>
      {/* Table skeleton */}
      <div className="bg-ic-bg-secondary rounded-lg p-4">
        {[1,2,3,4,5].map(i => (
          <div key={i} className="h-10 bg-ic-bg-tertiary rounded mb-2" />
        ))}
      </div>
    </div>
  );
}
```

### Error States

| Scenario | Display |
|----------|---------|
| FMP API error (5xx, timeout) | "Earnings data temporarily unavailable. Please try again later." with retry button |
| FMP API key misconfigured | "Earnings data is not available at this time." (no retry) |
| No earnings data for stock | "No earnings data available for {SYMBOL}." with muted icon |
| Calendar: no earnings on selected date | "No earnings reports scheduled for {date}." |
| Network error (client offline) | "Unable to connect. Check your internet connection." |

### Empty States
- History table with 0 past records: Show a centered message with a calendar icon
- Calendar with 0 results after filtering: "No results match your filters. Try broadening your search."

---

## 7. Performance Considerations

### Lazy Loading
- The Earnings tab already benefits from the existing `TickerTabs` architecture — only the active tab's content is rendered
- Data is fetched only when the user clicks the Earnings tab (via `useEffect` in the tab component)
- Charts are client-rendered with Recharts — no SSR needed

### Data Size
- Per-stock earnings: ~40 quarters = ~40 records ≈ 4KB JSON — negligible
- Calendar page: 2 weeks = ~200-500 records ≈ 50KB JSON — acceptable
- No pagination needed for per-stock data
- Calendar table: client-side search/filter is fine for <500 records

### Chart Rendering
- Recharts `<BarChart>` with 40 bars renders in <50ms — no optimization needed
- Use `<ResponsiveContainer>` for fluid width
- Memoize chart data transformation with `useMemo` to avoid recalculation on re-renders

```tsx
const chartData = useMemo(() =>
  earnings
    .filter(e => e.epsActual !== null)
    .slice(-zoomQuarters)
    .reverse()
    .map(e => ({
      quarter: e.fiscalQuarter,
      estimated: e.epsEstimated,
      actual: e.epsActual,
      beat: e.epsBeat,
    })),
  [earnings, zoomQuarters]
);
```

### Redis Cache Benefits
- Per-stock cache (1hr TTL): If 100 users view AAPL's earnings tab in an hour, only 1 FMP API call is made
- Calendar cache (4hr TTL): All users share the same calendar data for a given date range
- Estimated FMP API calls saved: ~95% reduction during peak hours

### Bundle Size
- `EarningsTab` and sub-components are dynamically imported only when the tab is active (Next.js code-splitting handles this automatically since each tab is a separate component)
- Recharts is already in the bundle (used by backtest/sentiment) — no additional bundle cost

---

## 8. Testing Plan

### Unit Tests (Go — `backend/services/`)

**File:** `backend/services/fmp_earnings_test.go`

| Test | Description |
|------|-------------|
| `TestComputeSurprisePercent_Beat` | Actual > Estimated → positive surprise |
| `TestComputeSurprisePercent_Miss` | Actual < Estimated → negative surprise |
| `TestComputeSurprisePercent_NilActual` | Null actual → nil result |
| `TestComputeSurprisePercent_NilEstimate` | Null estimate → nil result |
| `TestComputeSurprisePercent_ZeroEstimate` | Zero estimate → return absolute diff |
| `TestComputeBeat` | Various actual/estimate combinations |
| `TestToFiscalQuarter` | Date strings → "Q1 '26", "Q3 '25" etc. |
| `TestToFiscalQuarter_InvalidDate` | Malformed date → returns original string |
| `TestBeatRateCalculation` | 6 beats out of 8 quarters → correct counts |
| `TestTransformEarnings_FutureQuarters` | Future dates marked `isUpcoming: true` |
| `TestTransformEarnings_NextEarnings` | Correct next earnings record selected |

### Integration Tests (Go — `backend/handlers/`)

**File:** `backend/handlers/earnings_test.go`

| Test | Description |
|------|-------------|
| `TestGetEarnings_Success` | Mock FMP → 200 with valid data → verify response shape |
| `TestGetEarnings_FMPDown` | Mock FMP → 500 → verify 502 response |
| `TestGetEarnings_EmptyResult` | Mock FMP → empty array → verify empty earnings array |
| `TestGetEarnings_RedisCache` | Second call hits Redis, not FMP |
| `TestGetEarningsCalendar_DateRange` | Verify from/to params passed to FMP |
| `TestGetEarningsCalendar_Filtering` | Market cap filter applied correctly |

### Frontend Tests (TypeScript — component-level)

| Test | Description |
|------|-------------|
| `EarningsTable.test.tsx` | Renders correct columns, colors surprise values |
| `NextEarningsCard.test.tsx` | Shows countdown, handles null data |
| `EarningsBarChart.test.tsx` | Renders correct number of bars, handles empty data |
| `formatRevenue.test.ts` | "$1.2B", "$450M", "$1.5T" formatting |

### E2E Tests

| Test | Description |
|------|-------------|
| Earnings tab navigation | Click Earnings tab → data loads → table visible |
| Earnings Calendar page | Navigate to `/earnings-calendar` → date strip renders → table populated |
| Calendar search filter | Type "AAPL" → only Apple row shown |
| Deep link | Navigate to `/ticker/AAPL?tab=earnings` → Earnings tab active |

---

## 9. Estimated Implementation Effort

| Component | Effort | Dependencies |
|-----------|--------|-------------|
| **Backend: FMP client methods** (`fmp_earnings.go`) | 3 hours | None |
| **Backend: Earnings handler** (`earnings.go`) | 4 hours | FMP client methods |
| **Backend: Redis caching layer** | 2 hours | Earnings handler |
| **Backend: Route registration** (`main.go`) | 15 min | Handler |
| **Backend: Unit + integration tests** | 3 hours | Handler |
| **Frontend: API routes + client** (`routes.ts`, `earnings.ts`) | 30 min | Backend endpoints |
| **Frontend: EarningsTab + NextEarningsCard** | 3 hours | API client |
| **Frontend: EarningsSummaryStats** | 1 hour | API client |
| **Frontend: EarningsTable** | 4 hours | API client |
| **Frontend: EarningsBarChart** (EPS + Revenue) | 4 hours | API client |
| **Frontend: Ticker page integration** (`page.tsx`) | 30 min | EarningsTab |
| **Frontend: Earnings Calendar page** | 5 hours | Backend calendar endpoint |
| **Frontend: Calendar DateStrip** | 2 hours | Calendar page |
| **Frontend: Calendar Table + filters** | 3 hours | Calendar page |
| **Frontend: Header nav link** | 15 min | Calendar page |
| **Frontend: Responsive design pass** | 2 hours | All components |
| **QA + bug fixes** | 4 hours | All |
| | | |
| **Phase 1 Total** | **~41 hours (~5 dev days)** | |

### Suggested Implementation Order

1. Backend FMP client methods + unit tests
2. Backend handlers + Redis cache + integration tests
3. Frontend API client + routes
4. EarningsTab (NextEarningsCard → SummaryStats → Table → Charts)
5. Ticker page integration (add tab)
6. Earnings Calendar page (DateStrip → Table → Filters)
7. Header nav link
8. Responsive design pass + QA
