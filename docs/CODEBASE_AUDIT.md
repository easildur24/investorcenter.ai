# InvestorCenter.AI Codebase Audit

**Date**: March 1, 2026
**Purpose**: Audit existing codebase before home page redesign (PRD v2.0 / Tech Spec v1.0)
**Scope**: Frontend, Backend, Data Layer

---

## 1. Project Configuration

| Layer | Technology | Version |
|-------|-----------|---------|
| Frontend | Next.js (App Router) | 14.0.4 |
| React | React | ^18 |
| TypeScript | TypeScript | ^5 (strict mode) |
| Styling | Tailwind CSS | 3.3.0 |
| Data Fetching | SWR | 2.4.0 |
| State Mgmt | Zustand | 5.0.11 |
| Charts | Recharts (primary), Chart.js, D3 | 3.4.1 / 4.5.0 / 7.9.0 |
| Validation | Zod | 4.3.6 |
| URL Params | nuqs | 1.20.0 |
| Testing | Jest + React Testing Library + Playwright | 30.2.0 / 16.3.2 / 1.58.2 |
| Backend | Go / Gin | — |
| Database | PostgreSQL + TimescaleDB | — |
| Cache | Redis | — |
| Deployment | AWS EKS (Kubernetes) | — |

**Key Config Files**:
- `next.config.js`: Standalone output (Docker), Polygon.io image remotes, dev rewrite proxy to `localhost:8080`
- `tailwind.config.js`: Custom `ic-*` color namespace mapping to CSS variables
- `tsconfig.json`: Strict mode, path alias `@/*`, bundler module resolution
- `.env.example`: `NEXT_PUBLIC_API_URL`, `NEXT_PUBLIC_IC_SCORE_API_URL`

---

## 2. Directory Structure

```
/investorcenter.ai/
├── app/                          # Next.js 14 App Router
│   ├── page.tsx                  # Home page
│   ├── layout.tsx                # Root layout with providers
│   ├── globals.css               # Global styles
│   ├── ticker/[symbol]/          # Dynamic ticker pages
│   ├── screener/                 # Stock screener
│   ├── crypto/                   # Crypto page (broken)
│   ├── earnings-calendar/        # Earnings calendar
│   ├── watchlist/                # Watchlist pages
│   ├── alerts/                   # Alert management
│   ├── ic-score/                 # IC Score page
│   ├── reddit/                   # Reddit trends
│   ├── sentiment/                # Sentiment analysis
│   ├── auth/                     # Login/signup/password
│   └── admin/                    # Admin dashboard
├── components/                   # React components
│   ├── home/TopMovers.tsx        # Top Movers widget
│   ├── MarketOverview.tsx        # Market Overview widget
│   ├── Header.tsx                # Main navigation
│   ├── TickerSearch.tsx          # Search bar
│   ├── ticker/                   # Ticker detail components
│   ├── watchlist/                # Watchlist components
│   ├── ic-score/                 # IC Score components
│   └── ui/                       # Reusable UI primitives
├── lib/                          # Utilities & hooks
│   ├── api.ts                    # Main API client (singleton)
│   ├── api/                      # API client, routes, schemas
│   │   ├── client.ts             # HTTP client with token refresh
│   │   ├── routes.ts             # Centralized route paths
│   │   ├── schemas.ts            # Zod validation schemas
│   │   └── validate.ts           # Response validation
│   ├── auth/AuthContext.tsx       # JWT auth provider
│   ├── contexts/ThemeContext.tsx  # Dark/light mode
│   ├── hooks/                    # Custom hooks
│   │   ├── useRealTimePrice.ts   # Real-time price polling
│   │   ├── useScreener.ts        # SWR-powered screener
│   │   └── useApiWithRetry.ts    # Retry logic
│   ├── types/                    # TypeScript type definitions
│   ├── formatters/               # Number/date formatters
│   ├── stores/                   # Zustand stores
│   ├── theme.ts                  # Chart theme colors
│   └── utils.ts                  # General utilities
├── styles/
│   └── theme.css                 # CSS custom properties (dark/light)
├── backend/                      # Go/Gin API server
│   ├── main.go                   # Router + all route definitions
│   ├── handlers/                 # HTTP handlers
│   ├── services/                 # Business logic (Polygon, FMP)
│   ├── database/                 # SQL queries
│   ├── auth/                     # JWT, middleware, rate limiting
│   └── migrations/               # SQL migration files
├── ic-score-service/             # Python/FastAPI microservice
├── k8s/                          # Kubernetes manifests
├── terraform/                    # Infrastructure as code
└── middleware.ts                  # Cache-busting for /ticker/*
```

---

## 3. Current Home Page (`app/page.tsx`)

**Layout** (262 lines):
1. Hero section with gradient background, blur circles, CTA buttons ("Start Free Trial", "Watch Demo")
2. `<MarketOverview />` component
3. `<TopMovers />` component
4. Features section (4 feature cards)
5. Footer (inline, not a shared component)

**Missing from PRD requirements**:
- No Market Status Banner
- No sparklines anywhere
- No Market News section
- No Daily Market Summary
- No Upcoming Earnings widget
- No Sector Heatmap
- No Watchlist Preview (logged-in)
- No live mini-dashboard in hero
- Footer is inline, not reusable

---

## 4. Market Overview (`components/MarketOverview.tsx`)

**Current State** (137 lines):
- Calls `apiClient.getMarketIndices()` which hits `GET /api/v1/markets/indices`
- Polls every 30 seconds via `setInterval`
- Validates with Zod schema (`MarketIndicesSchema`)

**Backend Reality** (`backend/handlers/market_handlers.go:72-135`):
- **CONFIRMED BUG**: Uses ETF proxies (SPY, DIA, QQQ) for index values
- Maps: SPY -> "S&P 500", DIA -> "Dow Jones", QQQ -> "NASDAQ-100"
- Returns ETF share prices ($685, $489, $605) not index points (6,878 / 48,977 / 22,668)
- Data source: Polygon.io real-time quotes
- Response includes `symbol`, `name`, `price`, `change`, `changePercent`, `lastUpdated`

**No sparklines, no tabs, no config-driven structure.**

---

## 5. Top Movers (`components/home/TopMovers.tsx`)

**Current State** (228 lines):
- Default tab: `'gainers'` (line 30) - **Already correct per PRD**
- Three tabs: Top Gainers, Top Losers, Most Active
- Calls `apiClient.getMarketMovers(5)` -> `GET /api/v1/markets/movers?limit=5`
- Polls every 5 minutes
- Links to individual ticker pages

**Backend** (`backend/handlers/market_handlers.go:138-259`):
- Uses Polygon.io bulk stock snapshots
- 5-minute in-memory cache
- Filters: excludes NaN, >100% moves, <100K volume, <$1 penny stocks
- Returns `gainers`, `losers`, `mostActive` arrays

**Missing from PRD requirements**:
- No company names displayed (only ticker symbols)
- No sector tag pills
- No relative volume ("2.3x avg vol")
- No intraday sparklines per row
- No "Show 5 more" expand
- No % change color bar

---

## 6. Earnings Calendar (`app/earnings-calendar/page.tsx`)

**Current State** (548 lines):
- Date logic uses `getMonday(today)` and `nearestWeekday(today)` with `useMemo`
- Fetches from `GET /api/v1/earnings-calendar?from={date}&to={date}`
- Week navigation (prev/next buttons)
- Daily/weekly view toggle
- Sortable columns, search by symbol

**Backend** (`backend/handlers/earnings.go:108-233`):
- Data source: FMP (Financial Modeling Prep)
- Default range: current Monday through next week's Friday (12-day window)
- Redis cache: key `earnings:v1:calendar:{FROM}:{TO}`, 4-hour TTL
- Max range: 14 days

**Potential Stale Week Cause**:
- `useMemo(() => new Date(), [])` - date is memoized on mount, won't update on day change
- If backend returns stale cached data, the calendar shows old dates
- `nearestWeekday()`: Saturday -> Friday, Sunday -> Monday (could show previous week on weekends)

---

## 7. Crypto Page (`app/crypto/page.tsx`)

**Current State** (405 lines):
- Fetches from `GET /api/v1/crypto/` with pagination
- Expects symbols with `X:` prefix (e.g., `X:BTCUSD`)
- Polls every 5 seconds for price updates
- Flash animation for price changes

**Backend** (`backend/handlers/crypto_realtime_handlers.go`):
- Data source: Redis cache with key `crypto:quote:{SYMBOL}`
- Symbols stored in sorted set `crypto:symbols:ranked`
- **Root Cause of 0 Results**: Redis cache not being populated. The crypto price CronJob that populates Redis is likely not running or misconfigured.

---

## 8. Footer (Inline in `app/page.tsx`)

**Dead Links** (6 total):
| Section | Link Text | Current `href` | Should Be |
|---------|-----------|----------------|-----------|
| Platform | Charts | `#` | `/screener` or `/coming-soon` |
| Platform | Data | `#` | `/screener` or `/coming-soon` |
| Platform | Analytics | `#` | `/ic-score` or `/coming-soon` |
| Company | About | `#` | `/about` or `/coming-soon` |
| Company | Contact | `#` | `/coming-soon` |
| Company | Privacy | `#` | `/privacy` |

---

## 9. Data Fetching Patterns

### Pattern 1: Direct Fetch + Polling (most endpoints)
```typescript
// components/MarketOverview.tsx — typical pattern
const [data, setData] = useState<T[]>([]);
const [loading, setLoading] = useState(true);
const [error, setError] = useState<string | null>(null);
useEffect(() => {
  const fetch = async () => { ... };
  fetch();
  const interval = setInterval(fetch, 30000);
  return () => clearInterval(interval);
}, []);
```

### Pattern 2: SWR (screener, heavy filtering)
```typescript
// lib/hooks/useScreener.ts
const { data, error, isLoading } = useSWR<ScreenerResponse>(url, fetcher, {
  revalidateOnFocus: false, keepPreviousData: true, dedupingInterval: 2000,
});
```

### Pattern 3: Real-Time Price Polling (`useRealTimePrice`)
- Crypto: 5-second interval
- Stocks: 15+ seconds (respects backend `updateInterval`)
- Market session detection (pre_market, regular, after_hours, closed)

### Pattern 4: Raw fetch (earnings, crypto pages)
```typescript
fetch(`${API_BASE_URL}${earningsCalendar.list}?from=${from}&to=${to}`)
```

### API Client (`lib/api/client.ts`)
- Token refresh with promise deduplication
- Reads from localStorage: `access_token`, `refresh_token`
- Auto-retry on 401

### Centralized Routes (`lib/api/routes.ts`)
- Single source of truth for all endpoint paths
- Typed helpers: `tickers.price(symbol)`, `stocks.financials(ticker)`

### Response Validation (`lib/api/schemas.ts`)
- Zod schemas validate all API responses at runtime
- `validateResponse()` wrapper

---

## 10. Styling & Theme System

**Tailwind CSS** (primary, v3.3.0):
- Content: `./app/**/*.tsx`, `./components/**/*.tsx`
- Dark mode: selector-based `[data-theme="dark"]`
- Custom colors via CSS variables: `ic-bg-*`, `ic-text-*`, `ic-border-*`, `ic-surface-*`

**CSS Custom Properties** (`styles/theme.css`):
- Dark mode (default): `--ic-bg-primary: #09090B`
- Light mode: Slate palette (blue-tinted)
- Chart colors via `getChartColors(resolvedTheme)` from `lib/theme.ts`

**Dark Mode Implementation** (`lib/contexts/ThemeContext.tsx`):
- `ThemeProvider` context with `useTheme()` hook
- `ThemeToggle` component (Lucide icons)
- Hydration script injected in `layout.tsx`
- Selector: `[data-theme="dark"]` on `<html>`

**No CSS modules, no styled-components.**

---

## 11. Analytics

**Status: NOT CONFIGURED**

No Google Analytics, Mixpanel, or any analytics provider detected. This is a prerequisite for measuring PRD success metrics.

---

## 12. WebSocket Usage

**Status: NONE**

No WebSocket connections in the codebase. All real-time data uses HTTP polling:
- `useRealTimePrice`: setInterval (5s crypto, 15s+ stocks)
- Market Overview: 30s polling
- Top Movers: 5-minute polling

---

## 13. Backend API Summary (127+ endpoints)

### Market-Related Endpoints (relevant to redesign)
| Method | Route | Handler | Notes |
|--------|-------|---------|-------|
| GET | `/api/v1/markets/indices` | `GetMarketIndices` | **ETF proxies** (SPY/DIA/QQQ) |
| GET | `/api/v1/markets/movers` | `GetMarketMovers` | Gainers/losers/active, 5-min cache |
| GET | `/api/v1/markets/search` | Inline | Ticker search, DB query |
| GET | `/api/v1/earnings-calendar` | `GetEarningsCalendar` | FMP data, 4h Redis cache |
| GET | `/api/v1/crypto/` | `GetAllCryptos` | Redis-backed, paginated |
| GET | `/api/v1/tickers/:symbol` | `GetComprehensiveTicker` | Full ticker data |
| GET | `/api/v1/tickers/:symbol/news` | `GetTickerNews` | Per-ticker news |
| GET | `/api/v1/logos/:symbol` | `ProxyLogo` | Polygon.io logo proxy |

### Standard Response Format
```json
{
  "data": <payload>,
  "meta": { "timestamp": "RFC3339", "source": "polygon.io|database|fmp", "cached": boolean }
}
```

### Pagination Format (screener)
```json
{ "meta": { "total": 5000, "page": 1, "limit": 20, "total_pages": 250 } }
```

### External Data Sources
| Provider | Used For | Client Location |
|----------|----------|-----------------|
| Polygon.io | Stock quotes, bulk snapshots, ticker details, logos | `backend/services/polygon.go` |
| FMP | Earnings calendar, historical earnings | `backend/services/fmp_client.go` |
| Redis | Crypto prices (cache), earnings cache | `backend/handlers/crypto_realtime_handlers.go` |
| PostgreSQL | Tickers, screener, users, watchlists, IC scores | `backend/database/` |

---

## 14. Key Discrepancies: Tech Spec vs Actual Codebase

| Tech Spec Assumes | Actual Codebase | Impact |
|-------------------|-----------------|--------|
| `src/` directory structure | Root `app/`, `components/`, `lib/` | All file paths need adjustment |
| Next.js API routes or Node.js backend | Go/Gin backend on port 8080 | New endpoints go in Go, not Node |
| React Query / TanStack Query | SWR 2.4.0 + direct fetch | Can migrate or keep SWR |
| Vitest for testing | Jest 30.2.0 + React Testing Library | Use Jest, not Vitest |
| Vercel deployment | AWS EKS (Kubernetes) | No Vercel cron/KV; use K8s CronJobs |
| WebSocket support | No WebSocket infrastructure | Need to evaluate: add WS to Go, or use Ably/Pusher |
| `src/adapters/` pattern | Go handlers/services in backend | Data adapters are Go-side, not TS |
| Logo.dev or Brandfetch | Polygon.io logo proxy exists | May keep existing logo proxy |
| Mixpanel or GA4 | Nothing configured | Need to set up from scratch |
| Custom SVG sparklines | Recharts installed (3.4.1) | Can use Recharts or custom SVG |

---

## 15. Existing Infrastructure to Reuse

| Asset | Location | Reusability |
|-------|----------|-------------|
| API client with auth | `lib/api.ts`, `lib/api/client.ts` | High - extend with new endpoints |
| Zod validation schemas | `lib/api/schemas.ts` | High - add new schemas |
| Route definitions | `lib/api/routes.ts` | High - add new routes |
| Theme system (dark/light) | `styles/theme.css`, `lib/contexts/ThemeContext.tsx` | High - all new widgets use it |
| Real-time price hook | `lib/hooks/useRealTimePrice.ts` | Medium - can extend for indices |
| Recharts integration | `components/ticker/HybridChart.tsx` | Medium - for sparklines |
| Skeleton animation pattern | `components/MarketOverview.tsx` | High - extend to all widgets |
| Number formatters | `lib/utils.ts`, `lib/formatters/financial.ts` | High - reuse |
| Auth context | `lib/auth/AuthContext.tsx` | High - for logged-in features |
| Zustand store pattern | `lib/stores/watchlistPageStore.ts` | Medium - for complex state |
| Logo proxy | `GET /api/v1/logos/:symbol` | High - for earnings widget logos |
| Earnings API | `GET /api/v1/earnings-calendar` | High - extend for upcoming filter |
| Polygon service | `backend/services/polygon.go` | High - add index endpoints |
| Redis caching | `backend/handlers/earnings.go` | High - extend for news/summary |
