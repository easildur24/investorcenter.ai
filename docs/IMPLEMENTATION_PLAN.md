# InvestorCenter.AI — Home Page Redesign Implementation Plan

**Date**: March 1, 2026
**PRD**: v2.0 (US Equities & ETFs)
**Tech Spec**: v1.0 (8 Mini-Projects, 3 Phases)
**Companion**: [CODEBASE_AUDIT.md](./CODEBASE_AUDIT.md)

---

## Table of Contents
- [Shared Infrastructure](#0-shared-infrastructure)
- [MP-1: Bug Fixes & Data Accuracy](#mp-1-bug-fixes--data-accuracy)
- [MP-2: Analytics Foundation](#mp-2-analytics-foundation)
- [MP-3: Hero Section Redesign](#mp-3-hero-section-redesign)
- [MP-4: Market Overview Widget Overhaul](#mp-4-market-overview-widget-overhaul)
- [MP-5: Market News & Daily Summary](#mp-5-market-news--daily-summary)
- [MP-6: Upcoming Earnings Widget](#mp-6-upcoming-earnings-widget)
- [MP-7: Top Movers Enhancement](#mp-7-top-movers-enhancement)
- [MP-8: Personalization, Heatmap & Extensibility](#mp-8-personalization-heatmap--extensibility)
- [Environment & Config Requirements](#environment--config-requirements)

---

## 0. Shared Infrastructure

These shared modules are used by multiple mini-projects and should be built first or concurrently with MP-1.

### A. File Manifest

```
CREATE: lib/analytics.ts                          — Provider-agnostic analytics track() + identify()
CREATE: lib/hooks/useWidgetTracking.ts             — Auto-track widget visibility, time, interactions
CREATE: lib/market-status.ts                       — Trading hours, NYSE holidays, market state calc
CREATE: lib/types/market.ts                        — Shared interfaces: MarketAsset, SparklineData, NewsArticle, etc.
CREATE: config/market-config.ts                    — Config-driven tabs, nav items, asset categories
CREATE: config/nyse-holidays.ts                    — NYSE holiday calendar (2026-2027)
CREATE: config/gics-sectors.ts                     — 11 GICS sector definitions with colors
CREATE: components/ui/Sparkline.tsx                — Reusable SVG sparkline (80x24px, draw-on animation)
CREATE: components/ui/SkeletonCard.tsx             — Reusable skeleton card primitives
CREATE: components/ui/WidgetErrorBoundary.tsx       — Per-widget error isolation wrapper
CREATE: components/ui/SectorTag.tsx                — GICS sector pill badge
CREATE: components/ui/MarketStatusBadge.tsx         — Colored dot + label for market state
CREATE: components/Footer.tsx                      — Extract footer from page.tsx into reusable component
```

### B. Dependency Changes

```
ADD:    @tanstack/react-query     — Replace direct fetch+polling pattern (SWR kept for screener)
EVALUATE: Keep SWR for screener (already working well), use React Query for new widgets
KEEP:   recharts (3.4.1)          — For complex charts; sparklines will use custom SVG for bundle size
KEEP:   date-fns (4.1.0)          — Already installed, used for date formatting
KEEP:   zod (4.3.6)               — Runtime validation
ADD:    mixpanel-browser           — Analytics provider (or GA4 via @analytics/google-analytics)
```

### C. Shared TypeScript Interfaces (`lib/types/market.ts`)

```typescript
// Standard market asset (used by Market Overview, Hero, Top Movers)
export interface MarketAsset {
  symbol: string;
  name: string;
  value: number;
  change: number;
  percentChange: number;
  direction: 'up' | 'down' | 'flat';
  sparkline?: SparklinePoint[];
  metadata: {
    assetType: 'index' | 'stock' | 'etf';
    market: string;
    exchange?: string;
    displayFormat: 'points' | 'usd';
  };
}

export interface SparklinePoint {
  t: string;  // ISO8601 timestamp
  v: number;  // value
}

export interface MarketStatus {
  state: 'pre_market' | 'open' | 'after_hours' | 'closed';
  label: string;
  countdown: string;
  nextState: string;
  nextTransition: string;
}

export interface NewsArticle {
  id: string;
  title: string;
  source: string;
  url: string;
  publishedAt: string;
  category: 'earnings' | 'macro' | 'sector' | 'ipo_ma' | 'analysis';
  tickers: string[];
  isLive: boolean;
}

export interface EarningsEntry {
  ticker: string;
  companyName: string;
  logoUrl: string;
  date: string;
  timeOfDay: 'BMO' | 'AMC' | 'TNS';
  consensusEps: number;
  consensusRevenue: number;
  lastQuarter: {
    actualEps: number;
    expectedEps: number;
    beatMiss: 'beat' | 'miss';
    delta: number;
  };
  marketCap: number;
}

export interface MarketSummary {
  scope: string;
  summaryText: string;
  generatedAt: string;
  method: 'template' | 'llm';
  dataPoints: Record<string, number | string>;
}
```

### D. Market Status Utility (`lib/market-status.ts`)

Core logic for determining current market state, used by MarketStatusBanner (MP-3), MarketOverview (MP-4), and Hero (MP-3).

```typescript
// Key functions:
getMarketState(date: Date): MarketState
getCountdown(date: Date): string
getNextTransition(date: Date): { state: MarketState; time: Date }
isNYSEHoliday(date: Date): boolean
isTradingDay(date: Date): boolean
getNearestTradingDay(date: Date): Date
```

### E. Config Files

**`config/market-config.ts`** — Defines tabs, nav items, and asset categories:
```typescript
export const MARKET_OVERVIEW_TABS = [
  { id: 'us_indices', label: 'US Indices', category: 'us_indices', enabled: true },
  { id: 'us_etfs', label: 'Major ETFs', category: 'us_etfs', enabled: true },
  // Future: { id: 'crypto', label: 'Crypto', category: 'crypto', enabled: false },
];

export const NAV_MARKET_ITEMS = [
  { label: 'US Indices', href: '/markets/indices' },
  { label: 'ETFs', href: '/markets/etfs' },
  { label: 'Sectors', href: '/markets/sectors' },
];
```

**`config/nyse-holidays.ts`** — Hardcoded 2026-2027 NYSE holiday calendar.

**`config/gics-sectors.ts`** — 11 GICS sectors with label, light color, dark color.

---

## MP-1: Bug Fixes & Data Accuracy

**Phase 1 | Duration: 1.5 weeks | Team: 2 Frontend, 1 Backend**
Fix all credibility-damaging bugs before any new features ship.

### A. File Manifest

```
MODIFY: backend/handlers/market_handlers.go       — Replace ETF proxy with Polygon indices snapshot
MODIFY: backend/services/polygon.go                — Add GetIndexSnapshots() method for I:SPX, I:DJI, etc.
MODIFY: components/MarketOverview.tsx               — Update display: points format (no $), handle index metadata
MODIFY: app/earnings-calendar/page.tsx              — Fix date initialization to use current trading week
MODIFY: app/crypto/page.tsx                         — Replace with "Coming Soon" placeholder page
MODIFY: app/page.tsx                                — Fix footer links, extract footer component
MODIFY: components/home/TopMovers.tsx               — Add company name display below ticker
MODIFY: lib/api/routes.ts                           — Add new routes if endpoints change
MODIFY: lib/api/schemas.ts                          — Update MarketIndicesSchema for index format
CREATE: components/Footer.tsx                       — Extracted reusable footer with real links
```

### B. Dependency Changes

None required for MP-1.

### C. API Changes

#### C.1 — Modify `GET /api/v1/markets/indices` (backend)

**Current**: Fetches SPY, DIA, QQQ ETF prices via `GetStockRealTimePrice()`
**New**: Fetch actual index values via Polygon.io indices snapshot endpoint

**Polygon.io Indices Endpoint**: `GET /v3/snapshot/indices`
- Requires symbols: `I:SPX`, `I:DJI`, `I:COMP`, `I:RUT`, `I:VIX`
- Note: This endpoint may require a premium Polygon plan. If unavailable, keep ETF proxies but clearly label them as "ETF Proxy" and display in USD format, not points format.

**Updated Response Shape**:
```json
{
  "data": [
    {
      "symbol": "SPX",
      "name": "S&P 500",
      "value": 6878.10,
      "change": 52.34,
      "changePercent": 0.77,
      "displayFormat": "points",
      "lastUpdated": "2026-03-01T15:30:00Z"
    }
  ],
  "meta": {
    "count": 5,
    "timestamp": "2026-03-01T15:30:00Z",
    "source": "polygon.io",
    "dataType": "index"
  }
}
```

**Key Changes in `backend/handlers/market_handlers.go`**:
- Replace `indexSymbols` map from ETF tickers to index symbols (I:SPX, I:DJI, I:COMP, I:RUT, I:VIX)
- Add new method to `polygon.go`: `GetIndexSnapshots(symbols []string)` calling `GET /v3/snapshot/indices`
- Return `displayFormat: "points"` for indices, `"usd"` for ETFs
- Remove `$` prefix from index display values

**Key Changes in `components/MarketOverview.tsx`**:
- Check `displayFormat` field: if `"points"` format as `6,878.10` (comma-separated, no dollar sign); if `"usd"` format as `$685.23`
- Update the value display line (currently hardcodes `$` prefix on line 106)

**Fallback Strategy**: If Polygon.io premium is not available:
1. Keep ETF proxy approach
2. Clearly label as "ETF Proxies" in the meta section
3. Display with `$` prefix
4. Add a "Note: Showing ETF proxy values" tooltip

#### C.2 — Modify Top Movers API response

Add `name` (company name) to the `MoverStock` response. The backend already has this field in the struct (`Name string json:"name,omitempty"`), but it may not be populated. Check if `GetBulkStockSnapshots` returns company names; if not, look up from `tickers` table.

### D. Task Breakdown & Ordering

```
1. [Backend] Add GetIndexSnapshots() to polygon.go — no dependencies
2. [Backend] Update GetMarketIndices handler to use indices — depends on #1
3. [Backend] Ensure MoverStock.Name is populated — no dependencies, parallel with #1-2
4. [Frontend] Fix MarketOverview display format ($ vs points) — depends on #2
5. [Frontend] Fix Earnings Calendar date-sync — no dependencies, parallel with #1-4
6. [Frontend] Replace Crypto page with "Coming Soon" — no dependencies, parallel
7. [Frontend] Fix footer links + extract Footer component — no dependencies, parallel
8. [Frontend] Add company names to TopMovers rows — depends on #3
9. [All] Write tests — depends on #4-8
```

Tasks 1-3 (backend) and 5-7 (frontend) can all run in parallel.

### E. Detailed Task Specs

#### Task 1.1: Replace ETF Proxy Prices (Backend)

**File**: `backend/services/polygon.go`
- Add method `GetIndexSnapshots(symbols []string) ([]IndexSnapshot, error)`
- Call Polygon.io `GET /v3/snapshot/indices?ticker.any_of=I:SPX,I:DJI,I:COMP,I:RUT,I:VIX`
- Parse response into `IndexSnapshot` struct with `Value`, `Change`, `ChangePercent`
- Handle API errors gracefully

**File**: `backend/handlers/market_handlers.go`
- Replace `indexSymbols` array with index tickers (not ETF tickers)
- Call `GetIndexSnapshots()` instead of `GetStockRealTimePrice()`
- Map symbols: I:SPX -> "S&P 500", I:DJI -> "Dow Jones", I:COMP -> "NASDAQ Composite"
- Add `displayFormat: "points"` to response
- Add I:RUT (Russell 2000) and I:VIX to the list

#### Task 1.2: Fix Earnings Calendar Date-Sync (Frontend)

**File**: `app/earnings-calendar/page.tsx`
- **Root cause**: `const today = useMemo(() => new Date(), [])` on line 129 — memoized once on mount
- **Fix**: Use a more robust current-date calculation:
  - Replace `useMemo(() => new Date(), [])` with a direct `new Date()` in the `useState` initializer
  - Add `nearestTradingDay()` helper that accounts for weekends AND NYSE holidays
  - Ensure `getMonday()` snaps to the current week's Monday, not a stale one
- **Additional fix**: If the calendar is opened on a weekend, show the upcoming week (next Monday) rather than the previous week's Friday

#### Task 1.3: Crypto "Coming Soon" Page (Frontend)

**File**: `app/crypto/page.tsx`
- Replace the entire page with a polished placeholder:
  - Heading: "Crypto Markets Coming Soon"
  - Description: Brief text about upcoming crypto support
  - Optional email capture field (submit to backend or just display success toast)
  - CTA: "Return to Home" button
  - Matches existing dark/light mode theme

#### Task 1.4: Fix Footer Links (Frontend)

**File**: `app/page.tsx` (lines 172-259) and new `components/Footer.tsx`
- Extract footer into `components/Footer.tsx`
- Replace dead `href="#"` links with real routes or `/coming-soon` placeholder:
  - Charts -> `/screener`
  - Data -> `/screener`
  - Analytics -> `/ic-score`
  - About -> `/about` (create simple static page) or `/coming-soon`
  - Contact -> `/coming-soon`
  - Privacy -> `/privacy` (create simple static page) or `/coming-soon`
- Create `app/coming-soon/page.tsx` — generic "Coming Soon" placeholder for unbuilt pages

#### Task 1.5: Top Movers Default Tab

**Status**: Already defaults to `'gainers'` (line 30 of `TopMovers.tsx`). Verify in production.
- Tab order in rendering already matches: Gainers | Losers | Most Active
- **No change needed** unless production behavior differs from code

#### Task 1.6: Add Company Names to Top Movers

**Backend**: Ensure `MoverStock.Name` is populated in `GetMarketMovers()`:
- The `MoverStock` struct already has `Name string json:"name,omitempty"` (line 31)
- In the handler (line 138+), check if `GetBulkStockSnapshots()` returns ticker names
- If not, do a DB lookup: `SELECT name FROM tickers WHERE symbol = $1`
- Add `name` field to each mover entry

**Frontend** (`components/home/TopMovers.tsx`):
- Below the ticker symbol (line 185), add: `<span className="text-xs text-ic-text-muted">{stock.name}</span>`
- Truncate to 20 chars with `...` on mobile (Tailwind: `truncate max-w-[120px]`)

### F. Test Plan

```
CREATE: components/__tests__/MarketOverview.test.tsx
  - Unit: Renders index values in points format (no $ prefix) when displayFormat="points"
  - Unit: Renders ETF prices with $ prefix when displayFormat="usd"
  - Unit: Shows skeleton while loading
  - Unit: Shows error state with retry messaging

CREATE: backend/handlers/market_handlers_test.go (extend)
  - Unit: GetMarketIndices returns correct symbols (SPX, not SPY)
  - Unit: Index values > 1000 (not ETF prices)
  - Integration: Full request/response cycle

CREATE: app/earnings-calendar/__tests__/page.test.tsx
  - Unit: getEarningsWeekRange() returns current Monday-Friday
  - Unit: Weekend dates snap to upcoming Monday
  - Unit: NYSE holidays treated as non-trading days

CREATE: app/crypto/__tests__/page.test.tsx
  - Unit: Renders "Coming Soon" heading
  - Unit: No console errors
  - Visual: Matches dark/light mode

CREATE: components/__tests__/Footer.test.tsx
  - Unit: All links resolve to valid routes (no href="#")
  - Unit: External links open in new tab

CREATE: components/home/__tests__/TopMovers.test.tsx
  - Unit: Default tab is "gainers"
  - Unit: Company names display below ticker symbols
  - Unit: Company names truncate on mobile
```

---

## MP-2: Analytics Foundation

**Phase 1 | Duration: 0.5 week | Team: 1 Full-stack**
Establish event tracking before any redesign ships for baseline metrics.

### A. File Manifest

```
CREATE: lib/analytics.ts                           — Provider-agnostic track(), identify(), page()
CREATE: lib/hooks/useWidgetTracking.ts              — Auto-track visibility, time, interactions
MODIFY: app/layout.tsx                              — Initialize analytics provider
MODIFY: components/MarketOverview.tsx                — Wrap with useWidgetTracking
MODIFY: components/home/TopMovers.tsx                — Wrap with useWidgetTracking
MODIFY: app/page.tsx                                — Track page_view, hero_cta_click
```

### B. Dependency Changes

```
ADD: mixpanel-browser (or @analytics/google-analytics for GA4)
```

### C. Analytics Event Schema

```typescript
interface AnalyticsEvent {
  event_name: string;           // 'widget_interaction', 'cta_click', 'page_view'
  widget: string;               // 'market_overview', 'top_movers', 'hero'
  action: string;               // 'tab_switch', 'row_click', 'expand'
  metadata: Record<string, any>;// { tab: 'gainers', ticker: 'AAPL' }
  timestamp: string;            // ISO8601
  session_id: string;
  user_id?: string;             // null for logged-out
}
```

### D. Baseline Events to Track

| Event | Trigger | KPI |
|-------|---------|-----|
| `page_view` | Home page load | Bounce rate, sessions |
| `hero_cta_click` | Click "Start Free Trial" or "Explore Markets" | Trial start rate |
| `widget_visible` | Widget enters viewport (IntersectionObserver) | Scroll depth |
| `widget_interaction` | Any click/hover/tab-switch in a widget | Widget engagement |
| `ticker_click` | Click stock row to navigate to ticker page | CTR |
| `outbound_link` | Click news headline (external) | News engagement |
| `session_end` | Page unload / visibility change | Session duration |

### E. Task Ordering

```
1. Create lib/analytics.ts with provider-agnostic API — no dependencies
2. Create lib/hooks/useWidgetTracking.ts — depends on #1
3. Configure analytics provider in app/layout.tsx — depends on #1
4. Wrap existing MarketOverview + TopMovers with useWidgetTracking — depends on #2
5. Add CTA click tracking to hero section — depends on #1
6. Write tests — depends on #4-5
```

### F. Test Plan

```
CREATE: lib/__tests__/analytics.test.ts
  - Unit: track() calls provider with correct event schema
  - Unit: identify() sets user_id when logged in
  - Unit: Events include session_id and timestamp

CREATE: lib/hooks/__tests__/useWidgetTracking.test.ts
  - Unit: Fires widget_visible when IntersectionObserver triggers
  - Unit: Tracks time_on_widget correctly
  - Unit: Fires widget_interaction on click events
```

---

## MP-3: Hero Section Redesign

**Phase 2 | Duration: 1.5 weeks | Team: 1 Designer, 1 Frontend | Depends on: MP-1**

### A. File Manifest

```
CREATE: components/home/MarketStatusBanner.tsx      — Pre-Market / Open / After-Hours / Closed banner
CREATE: components/home/HeroMiniDashboard.tsx        — 3-5 live index tiles with sparklines
CREATE: components/home/HeroSection.tsx              — Redesigned hero with copy, CTAs, social proof
MODIFY: app/page.tsx                                — Replace inline hero with HeroSection component
MODIFY: lib/market-status.ts                        — Trading hours logic (shared infrastructure)
MODIFY: config/nyse-holidays.ts                     — Holiday data (shared infrastructure)
```

### B. Component Architecture

#### MarketStatusBanner
```typescript
// No props — self-contained, fetches own state
// Uses lib/market-status.ts for state calculation
// Updates countdown every 60 seconds
// Colors: green gradient (open), amber (pre/after), muted grey (closed)
// role="status" aria-live="polite" for accessibility
```

**States**:
- Pre-Market (4:00–9:30 AM ET weekdays): amber, "Pre-Market — Opens in Xh Xm"
- Open (9:30 AM–4:00 PM ET weekdays): green, "Market Open — Closes in Xh Xm"
- After-Hours (4:00–8:00 PM ET weekdays): amber, "After-Hours — Closes in Xh Xm"
- Closed: grey, "Market Closed — Opens Mon 9:30 AM ET" (or next trading day)

#### HeroMiniDashboard
```typescript
interface HeroMiniDashboardProps {
  // No props — fetches from same API as MarketOverview
  // Reuses: GET /api/v1/markets/indices
  // During market hours: polls every 15s for live data
  // During closed: shows last close, no animation
}
// Renders 3-5 tiles: S&P 500, NASDAQ, Dow (+ Russell 2000, VIX on desktop)
// Each tile: index name, value, change%, 5-day sparkline
// Number-rolling animation (300ms ease-out) on price change
// Green/red flash on update
```

#### HeroSection
```typescript
// Combines: MarketStatusBanner + HeroMiniDashboard + Copy + CTAs
// Logged-out: Full hero with trial CTA, social proof, "Start Free Trial" primary
// Logged-in: Compact hero (reduced height), no trial CTA
// Responsive: Desktop 2-col, tablet stacked, mobile horizontal scroll for tiles
```

### C. Task Ordering

```
1. [Shared] Build lib/market-status.ts + config/nyse-holidays.ts — no dependencies
2. [Frontend] Build MarketStatusBanner — depends on #1
3. [Frontend] Build HeroMiniDashboard with sparklines — depends on Sparkline (shared infra)
4. [Frontend] Build HeroSection (copy, CTAs, social proof) — depends on #2, #3
5. [Frontend] Responsive layout (4 breakpoints) — depends on #4
6. [Frontend] Dark mode + accessibility pass — depends on #4
7. [Frontend] Replace inline hero in app/page.tsx — depends on #4-6
8. [All] Write tests — depends on #7
```

### D. Test Plan

```
CREATE: lib/__tests__/market-status.test.ts
  - Unit: getMarketState() returns 'open' at 10:00 AM ET weekday
  - Unit: getMarketState() returns 'pre_market' at 8:00 AM ET weekday
  - Unit: getMarketState() returns 'closed' on Christmas (NYSE holiday)
  - Unit: getMarketState() returns 'closed' on Saturday
  - Unit: getCountdown() formats correctly: "2h 15m", "45m", "<1m"

CREATE: components/home/__tests__/MarketStatusBanner.test.tsx
  - Unit: Renders correct label for each market state
  - Unit: Banner has role="status" and aria-live="polite"
  - Visual: Green gradient when open, amber for pre/after, grey when closed

CREATE: components/home/__tests__/HeroMiniDashboard.test.tsx
  - Unit: Renders 3+ live index tiles
  - Unit: Values > 1000 (not ETF prices)
  - Responsive: Desktop 5 tiles, mobile 3 in horizontal scroll

CREATE: components/home/__tests__/HeroSection.test.tsx
  - Unit: Logged-out shows full hero with trial CTA
  - Unit: Logged-in shows compact hero
  - Visual: Dark mode renders correctly
```

---

## MP-4: Market Overview Widget Overhaul

**Phase 2 | Duration: 2 weeks | Team: 1 Frontend, 1 Backend | Depends on: MP-1**

### A. File Manifest

```
CREATE: backend/handlers/market_overview_handler.go  — New unified market overview endpoint
CREATE: backend/handlers/sparkline_handler.go        — Sparkline data endpoint
MODIFY: backend/services/polygon.go                  — Add GetIndexSnapshots(), GetETFSnapshots()
MODIFY: backend/main.go                              — Register new routes
MODIFY: lib/api/routes.ts                            — Add market overview + sparkline routes
MODIFY: lib/api/schemas.ts                           — Add MarketOverviewSchema, SparklineSchema
MODIFY: lib/api.ts                                   — Add getMarketOverview(), getSparklines() methods
CREATE: components/MarketOverviewV2.tsx               — Config-driven tabs, sparklines, market status
CREATE: components/ui/Sparkline.tsx                   — Reusable SVG sparkline (shared infra)
CREATE: config/market-config.ts                       — Tab configuration (shared infra)
MODIFY: app/page.tsx                                  — Replace old MarketOverview with V2
CREATE: app/markets/page.tsx                          — /markets page scaffold
CREATE: app/markets/layout.tsx                        — Markets page layout
MODIFY: components/Header.tsx                         — Add "Markets" nav dropdown
```

### B. API Endpoints

#### B.1 — New `GET /api/v1/markets/overview`

**Route**: `GET /api/v1/markets/overview?category={us_indices|us_etfs}`

**Backend Handler**: `backend/handlers/market_overview_handler.go`

**Response** (`category=us_indices`):
```json
{
  "data": {
    "category": "us_indices",
    "marketStatus": { "state": "open", "label": "Market Open" },
    "assets": [
      {
        "symbol": "SPX",
        "name": "S&P 500",
        "value": 6878.10,
        "change": 52.34,
        "percentChange": 0.77,
        "direction": "up",
        "sparkline": [
          { "t": "2026-02-24T16:00:00Z", "v": 6810.00 },
          { "t": "2026-02-25T16:00:00Z", "v": 6825.76 }
        ],
        "metadata": {
          "assetType": "index",
          "market": "US",
          "exchange": "NYSE",
          "displayFormat": "points"
        }
      }
    ]
  },
  "meta": { "timestamp": "...", "source": "polygon.io" }
}
```

**Data Source (us_indices)**: Polygon.io `GET /v3/snapshot/indices` for I:SPX, I:DJI, I:COMP, I:RUT, I:VIX
**Data Source (us_etfs)**: Polygon.io `GET /v2/snapshot/locale/us/markets/stocks/tickers` for SPY, QQQ, IWM, DIA, XLF, XLK, XLE, XLV, XLI, GLD, TLT

**Caching**: In-memory 15s TTL for ETFs, no caching for indices (real-time)
**Error**: 503 if Polygon unavailable, partial data if some symbols fail

#### B.2 — New `GET /api/v1/markets/sparklines`

**Route**: `GET /api/v1/markets/sparklines?symbols=SPX,DJI,COMP&period=5d&interval=1d`

**Backend Handler**: `backend/handlers/sparkline_handler.go`

**Response**:
```json
{
  "data": {
    "SPX": [
      { "t": "2026-02-24T16:00:00Z", "v": 6810.00 },
      { "t": "2026-02-25T16:00:00Z", "v": 6825.76 },
      { "t": "2026-02-26T16:00:00Z", "v": 6850.30 },
      { "t": "2026-02-27T16:00:00Z", "v": 6840.15 },
      { "t": "2026-02-28T16:00:00Z", "v": 6878.10 }
    ]
  },
  "meta": { "period": "5d", "interval": "1d" }
}
```

**Data Source**: Polygon.io `GET /v2/aggs/ticker/{symbol}/range/1/day/{from}/{to}`
**Caching**: Redis 5-minute TTL with key `sparklines:{symbols}:{period}:{interval}`

### C. Component Architecture

#### Sparkline (`components/ui/Sparkline.tsx`)
```typescript
interface SparklineProps {
  data: Array<{ t: string; v: number }>;
  width?: number;     // default 80
  height?: number;    // default 24
  direction: 'up' | 'down' | 'flat';
  animate?: boolean;  // draw-on animation (500ms)
  ariaLabel: string;  // "S&P 500: up 1.2% over 5 days"
}
// Pure SVG with <polyline>
// Color: var(--ic-positive) for up, var(--ic-negative) for down
// Animation: CSS stroke-dasharray + stroke-dashoffset transition
// Hover: tooltip with value and date (optional)
```

#### MarketOverviewV2 (`components/MarketOverviewV2.tsx`)
```typescript
// Config-driven tabs from config/market-config.ts
// Each tab lazy-loads data for its category
// Tab state persists in URL via nuqs (already installed)
// Each row: symbol, name, value, change, %change, sparkline
// Market status badge (reuses MarketStatusBadge from shared infra)
// Skeleton screen per tab while loading
// Error: show last-known data + warning icon
// Auto-refresh: 15s polling (ETFs), 30s (indices during market hours)
```

### D. Task Ordering

```
1. [Backend] Add GetIndexSnapshots() + GetETFSnapshots() to polygon.go — no dependencies
2. [Backend] Create market_overview_handler.go with category support — depends on #1
3. [Backend] Create sparkline_handler.go — parallel with #2
4. [Backend] Register routes in main.go — depends on #2, #3
5. [Frontend] Build Sparkline SVG component — no dependencies, parallel with #1-4
6. [Frontend] Build config/market-config.ts — no dependencies, parallel
7. [Frontend] Add API methods + routes + schemas — depends on #4
8. [Frontend] Build MarketOverviewV2 component — depends on #5, #6, #7
9. [Frontend] Add "Markets" nav dropdown to Header — depends on #6
10. [Frontend] Create /markets page scaffold — depends on #8, #9
11. [Frontend] Skeleton screens, error states, dark mode — depends on #8
12. [All] Write tests — depends on #8-11
```

Tasks 1-3 (backend) and 5-6 (frontend) can run in parallel.

### E. Test Plan

```
CREATE: components/ui/__tests__/Sparkline.test.tsx
  - Unit: Renders correct number of points from data array
  - Unit: Uses green color when direction='up', red when direction='down'
  - Unit: aria-label is set correctly on SVG element
  - Visual: Draw-on animation plays on initial render
  - Visual: Dark mode colors correct

CREATE: components/__tests__/MarketOverviewV2.test.tsx
  - Unit: Renders only tabs where enabled === true in config
  - Unit: Tab click loads correct category data
  - Unit: Index values displayed without $ prefix
  - Unit: ETF values displayed with $ prefix
  - Integration: Tab state persists within session
  - Unit: Skeleton screen shows during loading
  - Unit: Error state shows last-known data with warning

CREATE: backend/handlers/market_overview_handler_test.go
  - Unit: Returns correct response for us_indices category
  - Unit: Returns correct response for us_etfs category
  - Unit: Unknown category returns 400
  - Unit: Polygon error returns 503 with detail

CREATE: backend/handlers/sparkline_handler_test.go
  - Unit: Returns sparkline data for multiple symbols
  - Unit: Respects period and interval parameters
  - Unit: Redis cache hit returns cached data
```

---

## MP-5: Market News & Daily Summary

**Phase 2 | Duration: 2 weeks | Team: 1 Frontend, 1 Backend | Depends on: MP-1**

### A. File Manifest

```
CREATE: backend/handlers/news_handler.go             — News feed proxy + caching
CREATE: backend/handlers/summary_handler.go           — Market summary generation (template-based)
CREATE: backend/services/news_client.go               — News API client (Benzinga or Alpha Vantage)
CREATE: backend/services/summary_generator.go          — Template engine for market summaries
MODIFY: backend/main.go                               — Register news + summary routes
MODIFY: lib/api/routes.ts                              — Add news + summary routes
MODIFY: lib/api/schemas.ts                             — Add NewsSchema, SummarySchema
MODIFY: lib/api.ts                                     — Add getMarketNews(), getMarketSummary() methods
CREATE: components/home/MarketSummary.tsx               — Summary card with blue accent bar
CREATE: components/home/NewsFeed.tsx                    — 5-7 headlines with source, timestamp, category tag
MODIFY: app/page.tsx                                    — Add MarketSummary + NewsFeed to home page
```

### B. Dependency Changes

```
EVALUATE: News API provider — Benzinga Pro (paid) vs. Alpha Vantage News (free tier)
           Decision needed before development starts (blocker)
```

### C. API Endpoints

#### C.1 — New `GET /api/v1/markets/summary`

**Route**: `GET /api/v1/markets/summary?scope=us_equities`

**Response**:
```json
{
  "data": {
    "scope": "us_equities",
    "summaryText": "S&P 500 up 0.8%, NASDAQ up 1.2%, led by Technology sector gains of 2.1%.",
    "generatedAt": "2026-03-01T15:15:00Z",
    "method": "template",
    "dataPoints": {
      "sp500Change": 0.8,
      "nasdaqChange": 1.2,
      "topSector": "Technology",
      "sectorChange": 2.1
    }
  },
  "meta": { "timestamp": "...", "source": "generated" }
}
```

**Generation**: Template-based in Phase 2 (see `summary_generator.go`)
- Templates fill variables from: index changes, top sector, optional macro event from news headlines
- Pre-generated on schedule via Go goroutine or K8s CronJob (4x daily: 8:00 AM, 10:00 AM, 12:30 PM, 4:15 PM ET)
- Stored in Redis with key `market:summary:us_equities`
- API reads from cache (no generation latency on request)

#### C.2 — New `GET /api/v1/markets/news`

**Route**: `GET /api/v1/markets/news?categories=all&limit=7`

**Response**:
```json
{
  "data": {
    "articles": [
      {
        "id": "bz-123456",
        "title": "Apple Reports Record Q1 Revenue, Beats Estimates",
        "source": "Benzinga",
        "url": "https://benzinga.com/...",
        "publishedAt": "2026-03-01T14:30:00Z",
        "category": "earnings",
        "tickers": ["AAPL"],
        "isLive": true
      }
    ],
    "categories": ["all", "earnings", "macro", "sector", "ipo_ma"]
  },
  "meta": { "timestamp": "...", "source": "benzinga" }
}
```

**Backend (`news_client.go`)**:
- Proxies from chosen news API (Benzinga Pro or Alpha Vantage)
- Normalizes response into standard article schema
- Redis cache: 5-minute TTL during market hours, 15-minute outside
- Deduplicates articles by title similarity (>80% match)
- Tags articles with categories via keyword matching

### D. Component Architecture

#### MarketSummary (`components/home/MarketSummary.tsx`)
```typescript
// Card with blue accent bar on left
// Displays summary text + timestamp + "Auto-generated" label
// Fade transition (400ms) on text update
// Loading: 3-line skeleton placeholder
// Error: "Market summary is being prepared..." with latest index changes as fallback
// Dark mode: uses existing ic-surface + ic-text-primary
```

#### NewsFeed (`components/home/NewsFeed.tsx`)
```typescript
interface NewsFeedProps {
  maxHeadlines?: number;   // default 7 logged-in, 5 logged-out
  showFilters?: boolean;   // default false (Phase 2), true (Phase 3)
}
// Each headline: title (truncated to 2 lines), source, relative timestamp
// Category pill badge per article
// "LIVE" badge (red dot + text) for articles < 1 hour old
// Auto-refresh every 5 minutes during market hours
// Click opens URL in new tab (target="_blank" rel="noopener")
// Loading: 5 headline-shaped skeleton blocks
// Error: "News temporarily unavailable" + Retry button
```

### E. Task Ordering

```
1. [Backend] Create news_client.go (API proxy) — depends on API provider decision
2. [Backend] Create summary_generator.go (template engine) — no dependencies
3. [Backend] Create news_handler.go + summary_handler.go — depends on #1, #2
4. [Backend] Register routes in main.go — depends on #3
5. [Frontend] Add API routes + schemas + methods — depends on #4
6. [Frontend] Build MarketSummary component — depends on #5
7. [Frontend] Build NewsFeed component — depends on #5
8. [Frontend] Add to home page layout — depends on #6, #7
9. [Frontend] Skeleton screens, error states, dark mode — depends on #6, #7
10. [All] Write tests — depends on #8-9
```

Backend tasks 1-2 can run in parallel. Frontend tasks 6-7 can run in parallel.

### F. Test Plan

```
CREATE: backend/services/summary_generator_test.go
  - Unit: Template produces grammatically correct output for all direction combos
  - Unit: Summary length is 2-5 sentences (50-200 words)
  - Unit: Handles missing macro event gracefully

CREATE: backend/handlers/news_handler_test.go
  - Unit: Returns articles with correct schema
  - Unit: Deduplication filters similar titles
  - Unit: Category tagging works for known keywords
  - Unit: Cache hit returns cached data

CREATE: components/home/__tests__/MarketSummary.test.tsx
  - Unit: Renders summary text with timestamp
  - Unit: Shows skeleton during loading
  - Unit: Shows fallback message on error
  - Visual: Blue accent bar renders in both themes

CREATE: components/home/__tests__/NewsFeed.test.tsx
  - Unit: Renders correct number of headlines
  - Unit: "LIVE" badge appears for articles < 1 hour old
  - Unit: Category tags render with correct labels
  - Unit: Headlines open in new tab
  - Integration: Auto-refresh every 5 minutes
```

---

## MP-6: Upcoming Earnings Widget

**Phase 2 | Duration: 1.5 weeks | Team: 1 Frontend, 1 Backend | Depends on: MP-1**

### A. File Manifest

```
CREATE: backend/handlers/upcoming_earnings_handler.go  — Upcoming earnings with S&P 500 filter
MODIFY: backend/main.go                                — Register new route
MODIFY: lib/api/routes.ts                               — Add upcoming earnings route
MODIFY: lib/api/schemas.ts                              — Add UpcomingEarningsSchema
MODIFY: lib/api.ts                                      — Add getUpcomingEarnings() method
CREATE: components/home/UpcomingEarnings.tsx             — Earnings widget with day grouping
MODIFY: app/page.tsx                                     — Add UpcomingEarnings to home page
```

### B. API Endpoint

#### New `GET /api/v1/earnings/upcoming`

**Route**: `GET /api/v1/earnings/upcoming?days=5&filter=sp500&limit=10`

**Response**:
```json
{
  "data": {
    "earnings": [
      {
        "ticker": "AAPL",
        "companyName": "Apple Inc.",
        "logoUrl": "/api/v1/logos/AAPL",
        "date": "2026-03-03",
        "timeOfDay": "AMC",
        "consensusEps": 2.35,
        "consensusRevenue": 124500000000,
        "lastQuarter": {
          "actualEps": 2.18,
          "expectedEps": 2.10,
          "beatMiss": "beat",
          "delta": 0.08
        },
        "marketCap": 3800000000000
      }
    ],
    "dateRange": { "from": "2026-03-03", "to": "2026-03-07" }
  },
  "meta": { "total": 8, "filter": "sp500", "timestamp": "..." }
}
```

**Backend (`upcoming_earnings_handler.go`)**:
- Extends existing FMP earnings calendar data
- Adds `filter=sp500` parameter: filter by S&P 500 constituents (needs a list of S&P 500 tickers, could use screener_data view with market_cap rank)
- Adds `lastQuarter` beat/miss data from FMP historical earnings
- Logo URL uses existing Polygon.io logo proxy: `/api/v1/logos/{symbol}`
- Sort by market cap descending within each date
- Redis cache: 1-hour TTL with key `earnings:upcoming:{filter}:{days}`

### C. Component Architecture

#### UpcomingEarnings (`components/home/UpcomingEarnings.tsx`)
```typescript
// Groups entries by date (e.g., "Monday, Mar 3" heading)
// Each row: logo (32x32, fallback: colored circle with initial letter), ticker, company name
//           EPS estimate, time of day (Before Open / After Close)
//           Beat/miss badge from last quarter (green checkmark or red X + delta)
// "View Full Calendar" link -> /earnings-calendar with current week selected
// Watchlist stocks pinned to top with badge (for logged-in users)
// Loading: table row skeleton placeholders
// Error: "Earnings data unavailable" with link to /earnings-calendar
// Responsive: Desktop table, tablet hides revenue column, mobile card layout
```

### D. Task Ordering

```
1. [Backend] Create upcoming_earnings_handler.go — depends on existing FMP client
2. [Backend] Add S&P 500 filter logic (market cap-based or ticker list) — parallel with #1
3. [Backend] Register route in main.go — depends on #1
4. [Frontend] Add API route + schema + method — depends on #3
5. [Frontend] Build UpcomingEarnings component — depends on #4
6. [Frontend] Responsive layout + dark mode — depends on #5
7. [Frontend] Add to home page — depends on #5
8. [All] Write tests — depends on #7
```

### E. Test Plan

```
CREATE: backend/handlers/upcoming_earnings_handler_test.go
  - Unit: Returns earnings for next 5 trading days
  - Unit: sp500 filter excludes non-S&P 500 companies
  - Unit: Results sorted by market cap descending
  - Unit: lastQuarter data included with beat/miss

CREATE: components/home/__tests__/UpcomingEarnings.test.tsx
  - Unit: Groups entries by date correctly
  - Unit: Empty dates omitted (no empty headers)
  - Unit: Beat badge shows green checkmark + "Beat by $X.XX"
  - Unit: Miss badge shows red X + "Missed by $X.XX"
  - Unit: "BMO" renders as "Before Open", "AMC" as "After Close"
  - Unit: Logo fallback renders colored circle with initial
  - Unit: "View Full Calendar" links to /earnings-calendar
  - Responsive: Mobile shows compact card layout
```

---

## MP-7: Top Movers Enhancement

**Phase 2 | Duration: 1 week | Team: 1 Frontend | Depends on: MP-1**

### A. File Manifest

```
MODIFY: backend/handlers/market_handlers.go          — Add avg_volume_20d to MoverStock response
MODIFY: backend/services/polygon.go                   — Fetch 20-day avg volume (if not in snapshot)
CREATE: config/gics-sectors.ts                         — Sector definitions with colors (shared infra)
CREATE: components/ui/SectorTag.tsx                    — GICS sector pill badge (shared infra)
MODIFY: components/home/TopMovers.tsx                  — Add sector tags, relative volume, sparklines, expand
MODIFY: lib/api/schemas.ts                             — Update MoverStockSchema with new fields
```

### B. API Changes

**Modify `GET /api/v1/markets/movers` response**:

Add to each `MoverStock`:
```json
{
  "symbol": "NVDA",
  "name": "NVIDIA Corporation",
  "sector": "Technology",
  "price": 875.50,
  "change": 42.10,
  "changePercent": 5.05,
  "volume": 85000000,
  "avgVolume20d": 35000000,
  "sparkline": [
    { "t": "09:30", "v": 833.40 },
    { "t": "10:00", "v": 845.20 },
    ...
  ]
}
```

**Backend Changes**:
- `MoverStock` struct: add `Sector string`, `AvgVolume20d float64`, `Sparkline []SparklinePoint`
- Sector: look up from `tickers` or `screener_data` table
- AvgVolume20d: calculate from Polygon.io historical data or store in DB
- Sparkline: fetch intraday 5-min bars for current day from Polygon.io

### C. Component Changes

**TopMovers enhancements**:
1. **Sector tag pills**: `<SectorTag sector="Technology" />` after company name
2. **Relative volume**: `relativeVolume = volume / avgVolume20d` -> display as "2.3x avg vol", bold if > 2x
3. **Intraday sparklines**: Reuse `<Sparkline />` component from MP-4 (1-day, 5-min intervals, 60px wide)
4. **"Show 5 more" expand**: Initial 5 rows, expand button reveals next 5 (request `limit=10`)
5. **% change color bar**: Horizontal bar proportional to percentage change magnitude

### D. Task Ordering

```
1. [Frontend] Create config/gics-sectors.ts + SectorTag component — no dependencies
2. [Backend] Add sector, avgVolume20d, sparkline to MoverStock — no dependencies
3. [Frontend] Add sector tags to TopMovers rows — depends on #1, #2
4. [Frontend] Add relative volume display — depends on #2
5. [Frontend] Add intraday sparklines to rows — depends on Sparkline component (MP-4)
6. [Frontend] Add "Show 5 more" expand + % change bar — no dependencies
7. [Frontend] Dark mode + responsive + accessibility — depends on #3-6
8. [All] Write tests — depends on #7
```

Tasks 1 and 2 can run in parallel.

### E. Test Plan

```
CREATE: components/ui/__tests__/SectorTag.test.tsx
  - Unit: Renders correct color and label for each GICS sector
  - Visual: Colors distinct and accessible in both themes

CREATE: components/home/__tests__/TopMovers.enhanced.test.tsx
  - Unit: Sector tags render for each stock
  - Unit: Relative volume: 5M current / 2M avg = "2.5x avg vol"
  - Unit: Bold styling when relative volume > 2x
  - Unit: "Show 5 more" expands from 5 to 10 rows
  - Unit: % change bar width proportional to magnitude
  - Unit: Sparklines render per row
  - E2E: Default tab "Top Gainers" with names, sector tags, volume context
```

---

## MP-8: Personalization, Heatmap & Extensibility

**Phase 3 | Duration: 4 weeks | Team: 2 Frontend, 1 Backend, 1 ML | Depends on: MP-4, MP-5**

### A. File Manifest

```
CREATE: backend/handlers/sector_handler.go             — Sector performance data
CREATE: backend/handlers/breadth_handler.go             — Market breadth metrics
CREATE: backend/services/llm_summary.go                 — LLM-powered summary generation
MODIFY: backend/handlers/summary_handler.go             — A/B test template vs LLM
MODIFY: backend/main.go                                 — Register new routes
CREATE: components/home/SectorHeatmap.tsx                — 11 GICS sector heatmap grid
CREATE: components/home/MarketBreadth.tsx                — A/D ratio, 200-DMA %, 52-week H/L strip
CREATE: components/home/WatchlistPreview.tsx              — Top 5 watchlist stocks (logged-in)
MODIFY: components/home/NewsFeed.tsx                     — Add category filter tabs
MODIFY: components/home/HeroSection.tsx                  — A/B test headline variants
CREATE: app/markets/indices/page.tsx                     — Full US Indices page
CREATE: app/markets/etfs/page.tsx                        — Full ETFs page
CREATE: app/markets/sectors/page.tsx                     — Full Sectors page
MODIFY: app/page.tsx                                      — Add new widgets to home page layout
MODIFY: lib/api/routes.ts                                — Add sectors, breadth routes
MODIFY: lib/api.ts                                       — Add getSectorPerformance(), getMarketBreadth()
```

### B. API Endpoints

#### B.1 — New `GET /api/v1/markets/sectors`

**Route**: `GET /api/v1/markets/sectors?period=1d`

**Response**:
```json
{
  "data": {
    "period": "1d",
    "sectors": [
      { "id": "tech", "name": "Technology", "change": 1.85, "direction": "up" },
      { "id": "energy", "name": "Energy", "change": -0.42, "direction": "down" }
    ]
  },
  "meta": { "timestamp": "..." }
}
```

**Data Source**: Calculate from stock prices grouped by GICS sector (from `screener_data` view)
**Caching**: Redis 5-minute TTL

#### B.2 — New `GET /api/v1/markets/breadth`

**Route**: `GET /api/v1/markets/breadth`

**Response**:
```json
{
  "data": {
    "advanceDecline": { "advances": 320, "declines": 180, "ratio": 1.78 },
    "above200DMA": { "count": 285, "total": 500, "percentage": 57.0 },
    "fiftyTwoWeek": { "newHighs": 12, "newLows": 5 }
  },
  "meta": { "timestamp": "..." }
}
```

**Data Source**: Polygon.io bulk snapshots + DB technical indicators
**Caching**: Redis 5-minute TTL

#### B.3 — Upgrade `GET /api/v1/markets/summary` (LLM)

Add `method` query param: `GET /api/v1/markets/summary?scope=us_equities&method=llm`
- LLM call via Anthropic Claude API (or OpenAI)
- System prompt instructs 2-4 sentence summary from structured data
- Max 200 words, no financial advice, no predictions
- Label: "AI-generated" in response
- Fallback: template engine if LLM API fails
- Pre-generate on schedule (not on-demand)
- A/B test: 50% template vs 50% LLM

### C. Component Architecture

#### SectorHeatmap (`components/home/SectorHeatmap.tsx`)
```typescript
// 11 GICS sector tiles in grid
// Color scale: deep green (>+2%) through grey (0%) to deep red (<-2%)
// Each tile: sector name, % change, mini-sparkline (optional)
// Toggle buttons: 1D | 1W | 1M
// Desktop: 4-column grid, Mobile: 2-column with horizontal scroll
// Click-through to /markets/sectors
```

#### MarketBreadth (`components/home/MarketBreadth.tsx`)
```typescript
// Horizontal strip with 3 metrics:
// - Advance/Decline ratio (progress bar + ratio number)
// - % above 200-DMA (progress bar + percentage)
// - 52-week Highs vs Lows (two numbers)
// Compact, single-row layout
```

#### WatchlistPreview (`components/home/WatchlistPreview.tsx`)
```typescript
// Only rendered for logged-in users
// Fetches first watchlist's top 5 items
// Each row: ticker, company name, price, change%, sparkline
// "Edit" and "View All" links -> /watchlist/{id}
// Uses existing watchlist API: GET /api/v1/watchlists (auth required)
```

### D. Task Ordering

```
1. [Backend] Create sector_handler.go + route — no dependencies
2. [Backend] Create breadth_handler.go + route — no dependencies
3. [Backend] Create llm_summary.go (Anthropic/OpenAI client) — no dependencies
4. [Backend] Update summary_handler.go for A/B test — depends on #3
5. [Frontend] Build SectorHeatmap — depends on #1
6. [Frontend] Build MarketBreadth — depends on #2
7. [Frontend] Build WatchlistPreview — no new backend needed (uses existing APIs)
8. [Frontend] Add category filter tabs to NewsFeed — no backend change (client-side filter)
9. [Frontend] A/B test framework for hero headlines — needs analytics (MP-2)
10. [Frontend] Build full /markets pages (indices, etfs, sectors) — depends on #5, MP-4
11. [Frontend] Add new widgets to home page — depends on #5-8
12. [All] Write tests — depends on #11
```

Backend tasks 1-3 can all run in parallel. Frontend tasks 5-8 can run in parallel.

### E. Test Plan

```
CREATE: backend/handlers/sector_handler_test.go
  - Unit: Returns 11 GICS sectors
  - Unit: Supports 1d, 1w, 1m periods
  - Unit: Change values are percentage-based

CREATE: backend/services/llm_summary_test.go
  - Unit: LLM prompt constructed correctly
  - Unit: Fallback to template on LLM timeout
  - Unit: A/B test returns correct method label

CREATE: components/home/__tests__/SectorHeatmap.test.tsx
  - Unit: Renders 11 sector tiles
  - Unit: Colors scale correctly (green > 0, red < 0)
  - Unit: Period toggle switches data
  - Visual: Grid layout at all breakpoints

CREATE: components/home/__tests__/WatchlistPreview.test.tsx
  - Unit: Only renders for logged-in users
  - Unit: Shows top 5 stocks from first watchlist
  - Unit: "View All" links to correct watchlist page
```

---

## Environment & Config Requirements

### Environment Variables

```bash
# Existing (already configured)
POLYGON_API_KEY=xxx              # Market data — stocks, indices, ETFs, logos
NEXT_PUBLIC_API_URL=xxx          # Go backend URL (baked into Next.js build)
NEXT_PUBLIC_IC_SCORE_API_URL=xxx # Python IC Score service URL
REDIS_ADDR=xxx                   # Redis for caching
REDIS_PASSWORD=xxx               # Redis auth (optional)

# New — Required for Phase 2
NEWS_API_KEY=xxx                 # Benzinga Pro or Alpha Vantage news feed (MP-5)
MIXPANEL_TOKEN=xxx               # Analytics provider token (MP-2) — or GA4 measurement ID

# New — Required for Phase 3
ANTHROPIC_API_KEY=xxx            # LLM market summary generation (MP-8)
```

### Kubernetes Secrets (Production)

```bash
# Add news API key to app-secrets
kubectl patch secret app-secrets -n investorcenter \
  --type='merge' -p='{"data":{"news-api-key":"'$(echo -n $NEWS_API_KEY | base64)'"}}'

# Add analytics token (frontend needs it at build time via NEXT_PUBLIC_*)
# Option: bake into Docker build arg, or use config map

# Add Anthropic key for Phase 3
kubectl create secret generic llm-secret \
  --from-literal=anthropic-api-key="$ANTHROPIC_API_KEY" \
  -n investorcenter
```

### Polygon.io Plan Requirements

**Critical blocker for MP-1**: Verify if the current Polygon.io plan supports the indices snapshot endpoint (`GET /v3/snapshot/indices`).

- **Starter plan**: Stocks only, no indices snapshots
- **Developer plan**: Includes indices snapshots (I:SPX, I:DJI, I:COMP, I:RUT, I:VIX)
- **If indices unavailable**: Keep ETF proxy approach with clear labeling ("ETF Proxy Values"), display in USD format

Check: `curl "https://api.polygon.io/v3/snapshot/indices?ticker.any_of=I:SPX&apiKey=$POLYGON_API_KEY"`

---

## Dependency Graph & Execution Order

```
Week 1-2 (Phase 1):
  MP-1: Bug Fixes ─────────────────────── [MUST COMPLETE FIRST]
  MP-2: Analytics Foundation ──────────── [parallel with MP-1]

Week 3-4 (Phase 2, Wave 1):
  MP-3: Hero Redesign ────────────────── [depends on MP-1]
  MP-4: Market Overview ──────────────── [depends on MP-1]
  MP-7: Top Movers Enhancement ───────── [depends on MP-1, reuses MP-4 Sparkline]

Week 5-6 (Phase 2, Wave 2):
  MP-5: News & Summary ──────────────── [depends on MP-1, NEWS_API_KEY blocker]
  MP-6: Upcoming Earnings ────────────── [depends on MP-1]

Week 7-8 (Phase 2, Finalization):
  Integration testing across all Phase 2 widgets
  Performance audit (LCP < 2.5s, CLS < 0.1)
  Accessibility audit (WCAG 2.1 AA)

Week 9-12 (Phase 3):
  MP-8: Personalization & Extensibility ─ [depends on MP-4, MP-5]
```

### Critical Blockers

| Blocker | Blocks | Decision Needed By |
|---------|--------|--------------------|
| Polygon.io indices endpoint availability | MP-1 Task 1.1 | Before Phase 1 starts |
| News API provider selection | MP-5 | Before Week 5 |
| LLM provider for summaries (Anthropic/OpenAI) | MP-8 Task 8.4 | Before Week 9 |
| Analytics provider selection (Mixpanel/GA4) | MP-2 | Before Week 1 |

### Open Technical Questions

1. **Can Polygon.io current plan supply actual index values (I:SPX)?** If not, cost to upgrade? (Blocks MP-1)
2. **Which news API?** Benzinga Pro ($) vs Alpha Vantage News (free) vs NewsAPI? (Blocks MP-5)
3. **WebSocket for real-time indices?** Current Go backend doesn't have WS. Options: Add gorilla/websocket to Go, use Ably/Pusher as managed service, or keep polling at 15s intervals. (Affects MP-4)
4. **S&P 500 constituent list**: How to maintain? Static file? Pull from Polygon.io reference data? (Affects MP-6 filter)
5. **Social proof in hero**: Current registered user count? If < 1,000, use partner/press logos instead. (Affects MP-3)
