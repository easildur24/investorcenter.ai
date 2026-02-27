# PRD: Earnings Feature

**Author:** Product Team
**Date:** 2026-02-27
**Status:** Draft
**Version:** 1.0

---

## 1. Executive Summary

Add earnings data to InvestorCenter via two surfaces: (1) a new **"Earnings" tab** on each stock ticker page showing historical earnings results, EPS/revenue surprise analysis, and upcoming earnings dates, and (2) a site-wide **Earnings Calendar** page at `/earnings-calendar` where users can browse upcoming earnings across all tracked companies, filter by date, market cap, and report timing.

Data is sourced exclusively from **Financial Modeling Prep (FMP)**, which already powers InvestorCenter's Key Metrics and Financial Statements tabs. No new data provider is needed.

---

## 2. Goals & Success Metrics

### Goals
- Increase engagement on ticker pages by adding a high-demand data tab
- Provide a standalone earnings calendar to drive organic traffic and return visits
- Surface earnings surprise data (beat/miss) that directly supports investment decision-making
- Reduce user dependency on external sites (Yahoo Finance, Stock Analysis) for earnings info

### Success Metrics (KPIs)
| Metric | Target | Measurement |
|--------|--------|-------------|
| Earnings tab views as % of total tab views | >8% within 30 days | Analytics |
| Earnings Calendar unique page views/week | >500 within 60 days | Analytics |
| Avg time on Earnings tab | >45 seconds | Analytics |
| Earnings Calendar bounce rate | <50% | Analytics |
| Organic search impressions for "[ticker] earnings" | Measurable within 90 days | Search Console |

---

## 3. User Stories

1. **As an** investor researching a stock, **I want to** see its recent earnings history (EPS actual vs estimate, revenue actual vs estimate) **so that** I can assess whether the company consistently beats or misses expectations.

2. **As a** trader planning my week, **I want to** view a calendar of upcoming earnings dates across all stocks **so that** I can position trades ahead of volatile earnings events.

3. **As an** investor evaluating a stock, **I want to** see the next earnings date and countdown **so that** I know when the next catalyst event will occur.

4. **As a** long-term investor, **I want to** see a visual chart of EPS and revenue trends over multiple years **so that** I can identify growth trajectories and turning points.

5. **As a** screener user on the earnings calendar, **I want to** filter by market cap and report timing (before open / after close) **so that** I can focus on the most relevant earnings events for my strategy.

6. **As a** mobile user, **I want** the earnings table and charts to be responsive and readable on my phone **so that** I can check earnings data on the go.

---

## 4. Feature Scope

### In Scope (Phase 1 — MVP)
- "Earnings" tab on stock ticker page with:
  - Next Earnings Card (date, countdown, consensus estimates)
  - Earnings History Table (last 8 quarters, quarterly/annual toggle)
  - Beat Rate summary stat
  - EPS Bar Chart (estimated vs actual)
  - Revenue Bar Chart (estimated vs actual)
- Earnings Calendar page at `/earnings-calendar` with:
  - Date-strip weekday navigation with earnings counts
  - Daily/weekly toggle
  - Search by ticker or company name
  - Market cap filter
  - Sortable table of upcoming earnings

### In Scope (Phase 2 — Enhancements)
- Revenue YoY growth line overlay on bar chart
- "Earnings Time" filter (Before Market Open / After Market Close) — blocked until FMP re-enables BMO/AMC field
- Earnings call transcript link (when FMP Earnings Transcript endpoint is stable)
- Earnings notification/alert integration with existing alerts system
- SEO-optimized meta tags and structured data (Schema.org) for earnings pages

### Out of Scope
- Analyst estimate revisions / consensus trend over time
- Earnings whisper numbers
- Options-implied earnings move
- Earnings call audio playback
- Custom date range export / CSV download
- Earnings data for ETFs or crypto (stocks only)

---

## 5. Functional Requirements

### 5.1 Earnings Tab — Next Earnings Card

| # | Requirement |
|---|-------------|
| FR-1.1 | Display the estimated next earnings date sourced from FMP earnings calendar data |
| FR-1.2 | Label the date as "Confirmed" if the date matches a known filing pattern (within 7 days of historical average), or "Estimated" otherwise |
| FR-1.3 | Show a countdown in days ("in X days") calculated from the current date |
| FR-1.4 | Display the consensus EPS estimate for the upcoming quarter (from FMP `epsEstimated` field) |
| FR-1.5 | Display the consensus Revenue estimate for the upcoming quarter (from FMP `revenueEstimated` field) |
| FR-1.6 | If the next earnings date is in the past (earnings already reported this quarter), show the most recent result instead with a "Reported on [date]" label |
| FR-1.7 | If no upcoming earnings data exists, display "Earnings date not available" with a muted style |

### 5.2 Earnings Tab — Earnings History Table

| # | Requirement |
|---|-------------|
| FR-2.1 | Display a table with columns: Quarter, Period Ending, EPS Estimate, EPS Actual, EPS Surprise, Revenue Estimate, Revenue Actual, Revenue Surprise |
| FR-2.2 | Compute EPS Surprise % as `(epsActual - epsEstimated) / |epsEstimated| * 100` |
| FR-2.3 | Compute Revenue Surprise % as `(revenueActual - revenueEstimated) / |revenueEstimated| * 100` |
| FR-2.4 | Color-code surprise values: green for beat (positive surprise), red for miss (negative surprise), neutral/gray for within +/-0.5% |
| FR-2.5 | Show beat/miss icon inline with surprise value |
| FR-2.6 | Support Quarterly / Annual toggle. Default: Quarterly |
| FR-2.7 | Default display: last 8 quarters. Provide "Show More" to load up to 40 quarters |
| FR-2.8 | Rows should be sorted newest-first (most recent quarter at top) |
| FR-2.9 | When `epsActual` is null (future quarter), show "-" in actual and surprise columns |

### 5.3 Earnings Tab — Beat Rate Summary

| # | Requirement |
|---|-------------|
| FR-3.1 | Display "Beat EPS in X of last Y quarters" above the history table |
| FR-3.2 | A "beat" is defined as `epsActual > epsEstimated` |
| FR-3.3 | Also display "Beat Revenue in X of last Y quarters" alongside EPS beat rate |
| FR-3.4 | Y = number of quarters with both actual and estimate data (up to 8) |

### 5.4 Earnings Tab — EPS Chart

| # | Requirement |
|---|-------------|
| FR-4.1 | Render a bar chart showing EPS Estimated (outlined/ghost bar) vs EPS Actual (filled bar) per quarter |
| FR-4.2 | Support zoom presets: 2Y (default), 5Y, All |
| FR-4.3 | On hover, show tooltip with: Quarter label, EPS Estimated, EPS Actual, Surprise %, Beat/Miss |
| FR-4.4 | Bars that represent a beat should use green fill; miss should use red fill; estimate bar is always gray outline |
| FR-4.5 | X-axis: fiscal quarter labels (e.g., "Q3 '25"). Y-axis: EPS value ($) |

### 5.5 Earnings Tab — Revenue Chart

| # | Requirement |
|---|-------------|
| FR-5.1 | Same structure as EPS chart but for Revenue values |
| FR-5.2 | Y-axis should use abbreviated format (e.g., "$1.2B", "$450M") |
| FR-5.3 | Phase 2: Add YoY revenue growth % as a line overlay on the bar chart |

### 5.6 Earnings Calendar Page

| # | Requirement |
|---|-------------|
| FR-6.1 | Page accessible at `/earnings-calendar`, linked from main navigation |
| FR-6.2 | Date-strip navigation: show 5 weekdays (Mon-Fri) at a time with prev/next week arrows |
| FR-6.3 | Each date chip shows the date and count of earnings reporting that day |
| FR-6.4 | Clicking a date chip filters the table to that day. Default: today (or next business day if weekend) |
| FR-6.5 | Daily / Weekly toggle: Daily shows one day, Weekly shows Mon-Fri of selected week |
| FR-6.6 | Search bar: filter by company name or ticker symbol (client-side filtering) |
| FR-6.7 | Market Cap filter: All / Large Cap (>$10B) / Mid Cap ($2B–$10B) / Small Cap (<$2B) |
| FR-6.8 | Table columns: Symbol, Company Name, Market Cap, EPS Estimate, Revenue Estimate, Last EPS Actual, Last Revenue Actual |
| FR-6.9 | Table should be sortable by any column (default: Market Cap descending) |
| FR-6.10 | Symbol should link to the stock's ticker page Earnings tab (`/ticker/[symbol]?tab=earnings`) |
| FR-6.11 | Pre-fetch current week + next week of data on page load |
| FR-6.12 | Phase 2: Add "Report Time" column when FMP re-enables BMO/AMC data |

---

## 6. Non-Functional Requirements

| # | Requirement |
|---|-------------|
| NFR-1 | Earnings tab data must load within 2 seconds (P95) |
| NFR-2 | Earnings Calendar page must load within 3 seconds (P95) for a full week of data |
| NFR-3 | Backend must cache FMP earnings responses in Redis with 1-hour TTL for per-stock data |
| NFR-4 | Backend must cache FMP calendar responses in Redis with 4-hour TTL |
| NFR-5 | Graceful degradation: if FMP is unavailable, show "Earnings data temporarily unavailable" rather than breaking the tab |
| NFR-6 | FMP API rate limit: respect FMP's rate limits (300 req/min on Premium plan). Implement request throttling if needed |
| NFR-7 | Earnings tab should be lazy-loaded (only fetch data when user navigates to the tab) |
| NFR-8 | All monetary values must respect locale formatting (comma separators, 2 decimal places for EPS, abbreviated for revenue) |

---

## 7. UX/Design Requirements

### Visual Design
- Follow InvestorCenter's existing dark theme with `ic-bg-primary`, `ic-bg-secondary`, `ic-surface` background tokens
- Use `ic-text-primary`, `ic-text-muted`, `ic-text-dim` text tokens
- Green accent (`text-green-400` / `text-ic-positive`) for beats; red (`text-red-400` / `text-ic-negative`) for misses
- Cards use `bg-ic-bg-secondary rounded-lg p-4` pattern (matching KeyStatsTab GroupCard)
- Charts use teal/green palette consistent with existing Recharts usage

### Component Patterns
- Next Earnings Card: full-width card at top of tab, prominent styling with larger font for date and countdown
- Beat Rate: inline stat row using `MetricCard` pattern (grid of small stat cards)
- History Table: use existing table styling patterns from FinancialsTab (horizontal scroll on mobile, sticky first column)
- Charts: Recharts `<BarChart>` with responsive container, dark theme axis labels

### Responsive Behavior
- Earnings tab: single column on mobile, table scrolls horizontally
- Charts: `<ResponsiveContainer>` fills width, minimum height 300px
- Earnings Calendar: date strip scrolls horizontally on mobile, table stacks key columns
- Filters collapse into a dropdown/drawer on mobile

### Navigation
- Earnings tab positioned after "Financials" in the tab bar: `{ id: 'earnings', label: 'Earnings' }`
- Deep-link support: `/ticker/AAPL?tab=earnings` (existing TickerTabs behavior)
- Earnings Calendar linked from main site navigation header

---

## 8. Data Requirements

### FMP Endpoints Used

**Endpoint 1: Earnings History (per-stock)**
- URL: `GET /stable/earnings?symbol={SYMBOL}&apikey={KEY}`
- Returns: Array of `{ symbol, date, epsActual, epsEstimated, revenueActual, revenueEstimated, lastUpdated }`
- Max 1000 records per request
- Contains both historical (actuals filled) and future (actuals null) quarters
- Refresh: On each request, cached 1 hour in Redis

**Endpoint 2: Earnings Calendar (site-wide)**
- URL: `GET /stable/earnings-calendar?from={YYYY-MM-DD}&to={YYYY-MM-DD}&apikey={KEY}`
- Returns: Array of same schema as Endpoint 1, for all tickers in date range
- Max 4000 records, max 90-day window per request
- Refresh: On each request, cached 4 hours in Redis

### Computed Fields (Backend)
| Field | Formula |
|-------|---------|
| `epsSurprisePercent` | `(epsActual - epsEstimated) / abs(epsEstimated) * 100` |
| `revenueSurprisePercent` | `(revenueActual - revenueEstimated) / abs(revenueEstimated) * 100` |
| `epsBeat` | `true` if `epsActual > epsEstimated` |
| `revenueBeat` | `true` if `revenueActual > revenueEstimated` |
| `fiscalQuarter` | Derived from `date` field: map month to Q1-Q4 based on fiscal year end |
| `isUpcoming` | `true` if `epsActual` is null and `date` is in the future |
| `daysUntil` | `earningsDate - today` (only for upcoming) |

### Edge Cases
- **Null estimates**: Some small-cap stocks have no analyst estimates. Show "N/A" for estimate and surprise columns.
- **Null actuals (future)**: Future quarters have null actuals. Show estimates only, no surprise calculation.
- **Zero estimate**: If `epsEstimated` = 0, surprise % is undefined. Display absolute surprise value instead of percentage.
- **FMP downtime**: Return cached data if available; otherwise return error state.
- **No data**: Some newly-listed stocks have no earnings history. Show empty state: "No earnings data available for [SYMBOL]."
- **Fiscal year mismatch**: Some companies have non-calendar fiscal years (e.g., AAPL ends in September). Use FMP's `date` field as-is — it represents period-ending date.

---

## 9. Phasing / Rollout Plan

### Phase 1 — MVP (Target: 2 weeks)
- Backend: FMP client methods for earnings + calendar endpoints
- Backend: `/stocks/:ticker/earnings` and `/earnings-calendar` API endpoints with Redis caching
- Frontend: EarningsTab component (Next Earnings Card, History Table, Beat Rate stats, EPS/Revenue charts)
- Frontend: EarningsCalendarPage with date strip, table, search, and market cap filter
- Add "Earnings" to ticker page tab bar
- Add "Earnings Calendar" to site navigation
- Basic responsive design for mobile

### Phase 2 — Enhancements (Target: 2 weeks after Phase 1)
- Revenue YoY growth line overlay on charts
- BMO/AMC report timing column (when FMP re-enables)
- Earnings call transcript links
- SEO meta tags and Schema.org structured data
- Integration with alerts system ("Alert me before AAPL earnings")
- Performance optimization: virtual scrolling for calendar with 100+ results

---

## 10. Open Questions

| # | Question | Impact | Owner |
|---|----------|--------|-------|
| 1 | FMP's BMO/AMC (Before Market Open / After Market Close) field was removed — when will it return? | Affects Report Time column in calendar | Data Team |
| 2 | Should we show earnings data for ETFs or only individual stocks? | Scope definition | Product |
| 3 | Do we want to link to earnings call transcripts from FMP's transcript endpoint (separate API call)? | Phase 2 scope | Product |
| 4 | Should the Earnings Calendar page require authentication or be publicly accessible? | SEO and traffic implications | Product |
| 5 | Do we need to store earnings data in PostgreSQL, or is live FMP + Redis cache sufficient? | Architecture decision | Engineering |
| 6 | What is our FMP rate limit on the current plan? Need to confirm 300 req/min for Premium. | Caching TTL strategy | Engineering |
