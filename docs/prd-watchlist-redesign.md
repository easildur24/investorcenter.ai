# Product Requirements Document: Watch Lists Feature Redesign

**Document Version:** 1.0
**Author:** Product Management
**Date:** February 18, 2026
**Status:** Draft — Pending Stakeholder Review

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Goals & Success Metrics](#2-goals--success-metrics)
3. [User Personas & Jobs-to-Be-Done](#3-user-personas--jobs-to-be-done)
4. [Feature Requirements](#4-feature-requirements)
5. [User Flows](#5-user-flows)
6. [Data & Columns Specification](#6-data--columns-specification)
7. [UX & Design Requirements](#7-ux--design-requirements)
8. [Technical Considerations](#8-technical-considerations)
9. [Monetization & Freemium Strategy](#9-monetization--freemium-strategy)
10. [Go-to-Market & Rollout](#10-go-to-market--rollout)
11. [Risks & Mitigations](#11-risks--mitigations)
12. [Open Questions](#12-open-questions)

---

## 1. Executive Summary

### Problem Statement

InvestorCenter's Watch Lists feature is currently non-functional in production. Users hitting `/watchlist` encounter "Failed to fetch watch lists" errors, and the feature lacks the data-rich table views, sorting, and workflow integrations that retail investors expect from a financial analytics platform. This represents a critical product gap — watchlists are the single highest-engagement surface on every competing platform (Yahoo Finance, CNBC, StockAnalysis.com) and the primary mechanism through which casual browsers become daily-active retained users.

### Business Opportunity

Watchlists sit at the center of the investor workflow: research → track → act. Without a functional watchlist, InvestorCenter's AI Screener and Reddit Trends features exist as isolated discovery tools with no persistent engagement loop. Fixing and expanding watchlists unlocks:

- **Retention**: Watchlists create "stored value" — every ticker added increases switching cost.
- **Daily active usage**: Users checking portfolio-adjacent watchlists drive 3-5x/day return visits on competing platforms.
- **Premium conversion**: Column gating (IC Score, fair value, analyst consensus) provides a natural, high-intent upgrade prompt that users encounter during their core workflow rather than through interruptive marketing.
- **Data network effects**: More watchlist data improves our understanding of user interests, enabling personalized recommendations and smarter screener defaults.

### Vision

Deliver the most analytically powerful watchlist in the retail fintech space by combining real-time market data, InvestorCenter's proprietary IC Score system, Reddit social sentiment, and integrated alerting into a single, configurable table view — with a seamless "Screener → Watchlist → Alert" pipeline that no competitor offers today.

### Strategic Positioning vs. Competitors

| Dimension | Yahoo Finance | CNBC | StockAnalysis | **InvestorCenter (Target)** |
|-----------|--------------|------|---------------|----------------------------|
| Watchlist + Screener Integration | Manual | None | Basic | **One-click "Add to Watchlist" from screener results** |
| Proprietary Scoring | None | None | None | **IC Score (1-100) with 10 sub-factor breakdown** |
| Social Sentiment | None | None | None | **Reddit rank, mentions, trend direction per ticker** |
| Crypto + Equity Unified | Separate tabs | Equities only | Separate | **Unified watchlist with asset-type indicator** |
| Inline Alerts | Separate feature | Bell icon (basic) | None | **Set price/volume/event alerts from watchlist row** |
| Custom Column Views | Limited | Fixed | 8 preset views | **Preset views + fully custom column builder** |
| Heatmap Visualization | None | None | None | **Interactive D3 treemap with configurable metrics** |

### Expected Impact

| Metric | Current | 3-Month Target | 6-Month Target |
|--------|---------|----------------|----------------|
| Watchlist feature adoption (% of registered users) | 0% (broken) | 40% | 65% |
| DAU/MAU ratio | ~12% | 18% | 25% |
| Avg. session duration | 2.1 min | 4.5 min | 6+ min |
| Free → Premium conversion rate | 1.8% | 3.5% | 5% |

---

## 2. Goals & Success Metrics

### Primary KPIs

| KPI | Definition | Target | Measurement |
|-----|-----------|--------|-------------|
| **Watchlist Adoption Rate** | % of registered users who create at least 1 watchlist with ≥1 ticker | 40% within 90 days of launch | Backend analytics on `watch_lists` + `watch_list_items` table counts |
| **Weekly Active Watchlist Users** | Users who view their watchlist ≥3 days/week | 25% of registered users by month 6 | Page view events on `/watchlist/*` routes |
| **Screener-to-Watchlist Conversion** | % of screener sessions that result in ≥1 ticker added to a watchlist | 15% | Event tracking: `screener_add_to_watchlist` |
| **Premium Upgrade from Watchlist** | % of free users who upgrade after hitting a watchlist paywall (column gate, item limit, watchlist count limit) | 8% of users who encounter a gate | Conversion funnel: `watchlist_gate_shown` → `upgrade_started` → `upgrade_completed` |

### Secondary KPIs

| KPI | Definition | Target |
|-----|-----------|--------|
| Avg. tickers per watchlist | Mean items per non-empty watchlist | ≥8 |
| Watchlist count per active user | Mean watchlists per user with ≥1 watchlist | ≥2.5 |
| Alert creation from watchlist | % of watchlist users who create ≥1 alert via inline watchlist action | 20% |
| Heatmap engagement | % of watchlist views that navigate to heatmap | 10% |
| Time-to-first-ticker | Median time from account creation to first ticker added | <90 seconds |
| API error rate on watchlist endpoints | 5xx responses / total requests | <0.1% |

### Anti-Goals (Explicitly Not Building in This Phase)

- **Portfolio tracking with cost basis / P&L**: This requires brokerage linking or manual transaction entry. It is a fundamentally different product surface. Watchlists track *interest*, portfolios track *ownership*. Conflating them increases complexity without clear value at our scale.
- **Social/community watchlists**: Public sharing via `is_public` / `public_slug` fields exists in the schema but will not be exposed in the UI. Social features require moderation infrastructure we don't have.
- **Mobile native apps**: All work targets responsive web. Native apps are a separate initiative.
- **Real-time WebSocket streaming**: Polling at 15-30 second intervals is sufficient for watchlist views. WebSocket infrastructure is out of scope.
- **Brokerage account linking**: No integration with Plaid, Alpaca, or other trade execution / account aggregation services.

---

## 3. User Personas & Jobs-to-Be-Done

### Persona 1: "The Active Researcher" — Alex, 34

**Profile**: Software engineer, invests $2K/month into individual stocks. Uses screeners, reads SEC filings, follows earnings calendars. Has accounts on 3-4 finance platforms.

**Jobs-to-Be-Done**:
- *When I find an interesting stock through the screener*, I want to save it to a watchlist *so I can track it over days/weeks before deciding to buy*.
- *When I'm reviewing my watchlist*, I want to see fundamental data (P/E, revenue growth, IC Score) inline *so I don't have to click into each ticker individually*.
- *When a stock I'm watching hits my target buy price*, I want to get an alert *so I can act quickly*.

**Pain Points with Current State**:
- Cannot save screener results anywhere — has to manually remember symbols or use a spreadsheet.
- "Failed to fetch" error destroys trust in the platform's reliability.
- No way to set price targets or alerts from a centralized view.

### Persona 2: "The Trend Follower" — Maria, 28

**Profile**: Marketing manager, invests in ETFs and trending stocks. Follows Reddit (r/wallstreetbets, r/stocks), acts on momentum and social signals. Moderate financial literacy.

**Jobs-to-Be-Done**:
- *When I see a stock trending on Reddit*, I want to add it to my watchlist *so I can monitor whether the momentum sustains*.
- *When I check my watchlist*, I want to see Reddit sentiment alongside price data *so I can spot divergences between hype and fundamentals*.
- *When I'm tracking multiple themes* (e.g., AI stocks, meme stocks, dividend plays), I want separate named watchlists *so I can organize by strategy*.

**Pain Points with Current State**:
- Reddit Trends page exists but has no "Add to Watchlist" action — completely disconnected.
- Cannot create multiple themed watchlists.
- No social sentiment data visible in any persistent view.

### Persona 3: "The Diversified Holder" — James, 52

**Profile**: Financial advisor, manages personal portfolio across equities, ETFs, and a small crypto allocation. Values low-maintenance monitoring with alerts on significant moves.

**Jobs-to-Be-Done**:
- *When managing my combined equity + crypto portfolio*, I want a single watchlist that handles both asset types *so I don't have to check separate platforms*.
- *When something significant happens* (earnings surprise, unusual volume, price drop >5%), I want email alerts *so I don't have to check constantly*.
- *When reviewing my watchlist weekly*, I want a performance view showing 1W, 1M, YTD returns *so I can quickly assess momentum across all holdings*.

**Pain Points with Current State**:
- Crypto and equities require different workflows on most platforms.
- Alert system is broken (same "Failed to fetch" pattern).
- No performance view exists.

### Persona 4: "The Casual Browser" — Priya, 24

**Profile**: Recent graduate, just started investing through Robinhood. Uses InvestorCenter for research but hasn't created an account. Needs hand-holding and simplicity.

**Jobs-to-Be-Done**:
- *When I visit InvestorCenter for the first time*, I want to see what watchlists can do *so I'm motivated to create an account*.
- *When I create my first watchlist*, I want suggested tickers *so I'm not staring at an empty page*.
- *When I don't understand a column* (IC Score, P/E ratio), I want inline explanations *so I can learn while tracking*.

**Pain Points with Current State**:
- Feature is completely broken, providing no value and negative impression.
- No onboarding, no empty states, no educational content.
- No guest/preview mode to demonstrate value before signup.

---

## 4. Feature Requirements

### P0 — Must Ship (Critical Fixes + Core Functionality)

These items address the broken state and establish minimum viable watchlist functionality. All P0 items must ship before any P1 work begins.

#### P0-1: Fix API Reliability

**Problem**: All watchlist and alert endpoints return "Failed to fetch" errors.

**Root Cause Investigation Required**: The backend code (handlers, services, database layer) appears structurally sound based on code review. The failure is likely one of:
- Database migration `010_watchlist_tables.sql` not applied in production
- Missing `tickers` table reference (tests still reference old `stocks` table name)
- PostgreSQL connection pool exhaustion under load
- CORS or auth middleware rejecting requests

**Acceptance Criteria**:
- [ ] All 12 watchlist API endpoints return correct responses (200/201/204) for valid requests
- [ ] All 7 alert API endpoints return correct responses for valid requests
- [ ] Error responses include structured JSON with `error` field and appropriate HTTP status codes
- [ ] API response time p95 < 500ms for `GET /watchlists` (listing) and < 1s for `GET /watchlists/:id` (detail with real-time prices)
- [ ] Zero 5xx errors on watchlist endpoints over a 24-hour period in staging before production deploy

#### P0-2: Fix Test Suite

**Problem**: `watchlist_handlers_test.go` references `INSERT INTO stocks` but the table was renamed to `tickers` in migration 003.

**Acceptance Criteria**:
- [ ] All references to `stocks` table in test files updated to `tickers`
- [ ] Test suite passes with `go test ./handlers -v -run TestWatchList`
- [ ] CI pipeline green on watchlist test coverage

#### P0-3: Default Watchlist Auto-Creation

**Current State**: Database trigger `create_default_watch_list()` exists but may not be functioning.

**Acceptance Criteria**:
- [ ] New user registration automatically creates "My Watch List" with `is_default=true`
- [ ] Existing users without a default watchlist get one created on first visit to `/watchlist`
- [ ] Default watchlist cannot be deleted (UI hides delete button; API returns 400)

#### P0-4: Core CRUD Operations (Verify End-to-End)

**Acceptance Criteria**:
- [ ] Create watchlist: name (required), description (optional) → returns new watchlist with UUID
- [ ] List watchlists: returns all user's watchlists with `item_count`, ordered by `display_order`
- [ ] View watchlist detail: returns watchlist with all items enriched with real-time prices from Polygon
- [ ] Update watchlist: modify name and/or description
- [ ] Delete watchlist: cascade deletes items and associated alerts; default watchlist protected
- [ ] Add ticker: search autocomplete → select → add with optional notes/tags/targets
- [ ] Remove ticker: single-click remove with confirmation
- [ ] Bulk add: accept array of symbols (up to 500), skip duplicates, report results
- [ ] Reorder: drag-and-drop reorder persists `display_order` to database

---

### P1 — Core Feature Set

These items transform the watchlist from "functional" to "competitive." P1 should ship within 4-6 weeks of P0 completion.

#### P1-1: Configurable Data Column Views

Provide preset views that surface different data dimensions, similar to StockAnalysis.com's 8 views but enhanced with InvestorCenter-unique data.

**Preset Views**:

| View Name | Columns (beyond Symbol/Name) | Purpose |
|-----------|------------------------------|---------|
| **General** (default) | Price, Change $, Change %, Volume, Market Cap, 52-Week Range, IC Score | Daily monitoring |
| **Performance** | 1D %, 1W %, 1M %, 3M %, 6M %, YTD %, 1Y % | Momentum assessment |
| **Fundamentals** | P/E, P/B, P/S, ROE, ROA, Debt/Equity, Current Ratio | Value analysis |
| **Dividends** | Div Yield, Annual Dividend, Payout Ratio, Ex-Date, Frequency | Income investing |
| **IC Score** | IC Score, IC Rating, Value, Growth, Profitability, Financial Health, Momentum | Proprietary analysis |
| **Compact** | Price, Change %, IC Score, Market Cap | Quick glance |

**Acceptance Criteria**:
- [ ] View switcher (dropdown or tab bar) above the table, persists selection in `localStorage`
- [ ] Each view renders the specified columns with correct data types and formatting
- [ ] Column headers are clickable for sorting (ascending/descending toggle, with third click clearing sort)
- [ ] Current sort state shown with arrow indicator in column header
- [ ] "Custom" view option allows user to select any combination of available columns (P2 for the builder UI, but the data layer must support arbitrary column sets now)

#### P1-2: Watchlist Switcher

**Problem**: Navigating between watchlists requires going back to the dashboard.

**Acceptance Criteria**:
- [ ] Dropdown selector in the watchlist detail page header showing all user's watchlists
- [ ] Switching watchlists updates the URL and table data without full page reload
- [ ] "Create New Watchlist" option at the bottom of the dropdown
- [ ] Current watchlist name displayed prominently; item count shown as badge
- [ ] Keyboard shortcut: `Ctrl/Cmd + K` opens watchlist switcher (stretch goal)

#### P1-3: Table Sorting and Filtering

**Acceptance Criteria**:
- [ ] Click column header to sort ascending; click again for descending; third click clears
- [ ] Multi-column sort via Shift+Click (secondary sort)
- [ ] Quick filter bar: text search filters visible rows by symbol or company name (client-side)
- [ ] Asset type filter chips: "All", "Stocks", "ETFs", "Crypto" — filters rows by `asset_type`
- [ ] Sort and filter state preserved when switching between preset views

#### P1-4: Real-Time Price Display Enhancement

**Current State**: Prices refresh every 30 seconds via polling. No visual feedback on price updates.

**Acceptance Criteria**:
- [ ] Price cells flash green (up) or red (down) briefly when value changes on refresh
- [ ] "Last updated" timestamp shown in table footer (e.g., "Prices as of 2:34:15 PM ET")
- [ ] Market status indicator: green dot "Market Open" or gray dot "Market Closed" with pre/post-market label
- [ ] During market hours, refresh interval: 15 seconds for stocks, 5 seconds for crypto
- [ ] After hours: 60-second refresh for stocks, 15 seconds for crypto
- [ ] If Polygon API fails, show stale price with warning icon and "Delayed" badge

#### P1-5: Empty State and Onboarding

**Acceptance Criteria**:
- [ ] New user's default watchlist shows onboarding empty state:
  - Illustration or icon conveying "track your investments"
  - Headline: "Start tracking stocks you care about"
  - Subtext: "Add tickers to see real-time prices, analytics, and alerts in one place"
  - "Add Your First Ticker" primary CTA button (opens AddTickerModal)
  - "Or try these popular picks:" with 6 clickable chips (AAPL, MSFT, GOOGL, AMZN, TSLA, X:BTCUSD) that add on click
  - "Import from CSV" secondary link
- [ ] Watchlist dashboard (when user has watchlists but no items in any): show "Your watchlists are empty" with similar guidance
- [ ] All empty states are visually polished, not just blank white space

#### P1-6: Error State Handling

**Acceptance Criteria**:
- [ ] Replace "Failed to fetch watch lists" with actionable error states:
  - Network error → "Unable to connect. Check your internet connection and try again." + Retry button
  - Auth error (401) → "Your session has expired." + "Sign in" button
  - Server error (500) → "Something went wrong on our end. We're looking into it." + Retry button + link to status page
  - Rate limited (429) → "Too many requests. Please wait a moment." + auto-retry with backoff
- [ ] Retry button triggers immediate re-fetch
- [ ] Error states are visually distinct from empty states (use warning/error colors, not neutral)
- [ ] Partial failure: if Polygon price fetch fails for some tickers, show available data + "Price unavailable" for failed ones (don't block the entire table)

#### P1-7: Mobile Responsive Table

**Acceptance Criteria**:
- [ ] Breakpoint: 768px
- [ ] Below 768px: table collapses to card layout (one card per ticker)
  - Card shows: Symbol + Logo, Name, Price, Change %, IC Score badge
  - Tap to expand: shows full row data in stacked key-value format
  - Swipe left to reveal Remove action
- [ ] Above 768px: standard table with horizontal scroll if columns exceed viewport
- [ ] Column view switcher works on mobile (dropdown instead of tabs)
- [ ] Add Ticker modal is full-screen on mobile

#### P1-8: CSV Import / Export

**Acceptance Criteria**:
- [ ] **Import**: Accept CSV file with column `Symbol` (or `Ticker`). Parse symbols, validate against `tickers` table, bulk-add valid ones. Report: "Added 45 of 50 tickers. 5 not found: XYZ, ABC, ..."
- [ ] **Export**: Download current watchlist as CSV with all visible columns in current view. Filename: `{watchlist-name}-{YYYY-MM-DD}.csv`
- [ ] Import button in watchlist toolbar; Export button in "..." overflow menu
- [ ] Import limit: 500 tickers per operation (matches `BulkAddTickersRequest` backend limit)

---

### P2 — Differentiating Features (InvestorCenter Unique)

These features are InvestorCenter's competitive moat. P2 should begin development in parallel with P1 polish and ship 4-8 weeks after P1.

#### P2-1: AI Screener → Watchlist Integration

**Problem**: The screener and watchlist are completely disconnected. Users who discover interesting stocks through the screener have no way to persist those discoveries.

**Acceptance Criteria**:
- [ ] **Screener results table**: Add "+" icon button in each row that opens a "Add to Watchlist" dropdown showing user's watchlists
- [ ] **Bulk add from screener**: Checkbox column in screener results. Floating action bar at bottom: "Add {N} selected to watchlist" → watchlist picker dropdown
- [ ] **"Save as Watchlist"**: Button in screener toolbar creates a new watchlist pre-populated with all current screener results (respecting tier item limits)
- [ ] **Smart watchlists (stretch)**: Saved screener filters that auto-update. E.g., "All stocks with IC Score > 70 and P/E < 20" as a dynamic watchlist. Show "Auto-updating" badge. Refreshes daily at 00:00 UTC when `screener_data` view refreshes.

#### P2-2: Reddit Trends Integration in Watchlist

**Current State**: `WatchListItemWithData` already includes `RedditRank`, `RedditMentions`, `RedditPopularity`, `RedditTrend`, `RedditRankChange` from a LATERAL join on `reddit_heatmap_daily`. This data is fetched but not prominently displayed.

**Acceptance Criteria**:
- [ ] **"Social" preset view** added to view switcher: Columns = Price, Change %, Reddit Rank, Reddit Mentions (24h), Popularity Score, Trend Direction (↑↗→↘↓ icons), Rank Change
- [ ] **Trend badge on General view**: Small indicator next to ticker name when `RedditTrend = "rising"` and `RedditMentions > 50` — shows flame icon and "Trending on Reddit"
- [ ] **Reddit Trends page integration**: "Add to Watchlist" button on each ticker row in the Reddit Trends table
- [ ] **Watchlist-level social summary**: Above table, show aggregate: "3 of your tickers are trending on Reddit today" with clickable links

#### P2-3: Crypto + Equity Unified Experience

**Current State**: Backend supports both (crypto symbols prefixed `X:`). Frontend `AddTickerModal` hints at crypto with "X:BTCUSD" example.

**Acceptance Criteria**:
- [ ] Crypto tickers display with crypto icon badge (distinguished from stock icons)
- [ ] Price column shows 24h change for crypto (crypto markets don't close)
- [ ] Volume column shows 24h volume for crypto
- [ ] "Crypto" filter chip in asset type filter
- [ ] Market status indicator handles crypto: "24/7 Market" instead of "Market Closed"
- [ ] Crypto tickers use Redis cache prices (existing infrastructure) with 5-second refresh

#### P2-4: Inline Alert Creation from Watchlist

**Current State**: Alert system exists (`alert_rules` table, `AlertRule` model, full CRUD API) with `WatchListID` as a required field — alerts are already architecturally linked to watchlists. But the UI has no connection between the two features.

**Acceptance Criteria**:
- [ ] **Bell icon** in each watchlist row's Actions column
- [ ] Clicking bell opens a compact inline alert creation panel (not a separate page):
  - Pre-populated with symbol and watchlist ID
  - Alert type selector: Price Above, Price Below, % Change, Volume Spike, Earnings, News
  - Threshold input (number field)
  - Frequency: Once / Daily / Always
  - Notify via: Email toggle, In-App toggle
  - "Create Alert" button
- [ ] **Active alert indicator**: If ticker has active alerts, bell icon shows filled (not outline) with count badge
- [ ] **Quick alert from target prices**: If user sets `target_buy_price` or `target_sell_price` on a ticker, offer "Create alert for this target?" prompt — auto-fills a price_below or price_above alert
- [ ] **Alert status in table**: Optional "Alerts" column showing active alert count per ticker

#### P2-5: IC Score Deep Integration

**Current State**: IC Score data is available in the screener materialized view but not exposed in watchlist item queries.

**Acceptance Criteria**:
- [ ] **IC Score column**: Numeric score (1-100) with color-coded background:
  - 75-100: Green ("Strong Buy")
  - 60-74: Light Green ("Buy")
  - 45-59: Yellow ("Hold")
  - 30-44: Orange ("Sell")
  - 0-29: Red ("Strong Sell")
- [ ] **IC Rating text badge**: "Strong Buy" / "Buy" / "Hold" / "Sell" / "Strong Sell"
- [ ] **IC Score mini-breakdown**: Click/hover on IC Score cell reveals tooltip with radar chart of 10 sub-factors (Value, Growth, Profitability, Financial Health, Momentum, Analyst Consensus, Insider Activity, Institutional, News Sentiment, Technical)
- [ ] **Sector percentile**: "Top 15% in Technology" type label
- [ ] **IC Score preset view** (defined in P1-1): Full sub-factor columns
- [ ] **Watchlist-level IC Score average**: Header metric showing average IC Score across all tickers in the watchlist
- [ ] Requires new backend query: JOIN `watchlist_items` with `screener_data` materialized view on symbol to pull IC Score data

#### P2-6: Heatmap Visualization Enhancement

**Current State**: D3 treemap heatmap exists at `/watchlist/[id]/heatmap` with configurable size/color metrics.

**Acceptance Criteria**:
- [ ] **"Heatmap" tab** alongside table view (not a separate page): toggle between Table and Heatmap without leaving the watchlist
- [ ] **IC Score as color metric option**: Add to existing color metric dropdown
- [ ] **Sector grouping**: Optional treemap grouping by sector (nested rectangles)
- [ ] **Performance tooltip**: On hover, show sparkline (7-day price chart) in addition to existing data

#### P2-7: Watchlist Summary Metrics Bar

**Acceptance Criteria**:
- [ ] Persistent metrics bar above the table showing watchlist-level aggregates:
  - Total tickers count
  - Average IC Score (with color indicator)
  - Average 1D Change %
  - Average Dividend Yield
  - Median Market Cap (formatted: "$45.2B")
  - Number of tickers currently trending on Reddit
- [ ] Metrics update in real-time with price refreshes
- [ ] Clicking a metric sorts the table by that column

---

## 5. User Flows

### Flow 1: Create a New Watchlist

```
[Watchlist Dashboard]
  │
  ├─ User clicks "+ Create Watchlist" button
  │
  ├─ [CreateWatchListModal opens]
  │   ├─ User enters Name (required): "AI Stocks 2026"
  │   ├─ User enters Description (optional): "Tracking AI infrastructure plays"
  │   └─ User clicks "Create"
  │
  ├─ [API: POST /api/v1/watchlists]
  │   └─ Response: { id: "uuid", name: "AI Stocks 2026", ... }
  │
  ├─ [Toast: "Watchlist created"]
  │
  └─ [Redirect to /watchlist/{id}]
      └─ Shows empty state with onboarding CTA
```

### Flow 2: Add Stocks/Crypto to a Watchlist

**Path A: From Watchlist Detail Page**
```
[Watchlist Detail Page]
  │
  ├─ User clicks "+ Add Ticker" button
  │
  ├─ [AddTickerModal opens]
  │   ├─ User types "NV" in search box
  │   ├─ [Debounced search: GET /api/v1/tickers?q=NV]
  │   ├─ Dropdown shows: NVDA (NVIDIA Corp, NASDAQ), NVO (Novo Nordisk, NYSE), ...
  │   ├─ User selects "NVDA"
  │   ├─ Optional: User sets Target Buy Price: $850
  │   ├─ Optional: User adds tag: "AI"
  │   └─ User clicks "Add"
  │
  ├─ [API: POST /api/v1/watchlists/{id}/items]
  │   Body: { symbol: "NVDA", target_buy_price: 850, tags: ["AI"] }
  │
  ├─ [Toast: "NVDA added to AI Stocks 2026"]
  │
  └─ [Table updates with new row, real-time price fetched]
```

**Path B: From AI Screener (P2)**
```
[Screener Results Page]
  │
  ├─ User checks boxes next to NVDA, AMD, MSFT, GOOGL
  │
  ├─ [Floating action bar: "Add 4 selected to watchlist"]
  │   └─ User clicks → Dropdown shows user's watchlists
  │       └─ User selects "AI Stocks 2026"
  │
  ├─ [API: POST /api/v1/watchlists/{id}/bulk]
  │   Body: { symbols: ["NVDA", "AMD", "MSFT", "GOOGL"] }
  │
  └─ [Toast: "4 tickers added to AI Stocks 2026"]
```

**Path C: From Reddit Trends Page (P2)**
```
[Reddit Trends Page]
  │
  ├─ User sees PLTR trending (#3, 1,200 mentions)
  │
  ├─ User clicks "+" button on PLTR row
  │   └─ Dropdown: select target watchlist
  │
  ├─ [API: POST /api/v1/watchlists/{id}/items]
  │
  └─ [Toast: "PLTR added to My Watch List"]
```

### Flow 3: View and Manage Watchlist Data

```
[Watchlist Detail Page — "AI Stocks 2026" with 12 tickers]
  │
  ├─ Default view: "General" (Price, Change, Volume, Market Cap, IC Score)
  │
  ├─ User switches to "Performance" view via dropdown
  │   └─ Table re-renders with 1D%, 1W%, 1M%, 3M%, 6M%, YTD%, 1Y% columns
  │
  ├─ User clicks "1M %" column header → sorts descending
  │   └─ NVDA at top (+15.2%), INTC at bottom (-8.3%)
  │
  ├─ User clicks asset filter chip "Crypto" → filters to X:BTCUSD, X:ETHUSD
  │
  ├─ User clicks "Heatmap" tab → treemap visualization renders
  │   └─ Size: Market Cap, Color: 1D Change %
  │
  └─ User clicks NVDA tile → navigates to /ticker/NVDA detail page
```

### Flow 4: Set Alerts from Within a Watchlist Row

```
[Watchlist Detail Page]
  │
  ├─ User clicks bell icon (🔔) on NVDA row
  │
  ├─ [Inline Alert Panel expands below the row]
  │   ├─ Pre-filled: Symbol = NVDA, Watchlist = "AI Stocks 2026"
  │   ├─ User selects Alert Type: "Price Below"
  │   ├─ User enters Threshold: $800
  │   ├─ Frequency: "Once"
  │   ├─ Notify: ☑ Email  ☑ In-App
  │   └─ User clicks "Create Alert"
  │
  ├─ [API: POST /api/v1/alerts]
  │   Body: {
  │     watch_list_id: "{id}",
  │     symbol: "NVDA",
  │     alert_type: "price_below",
  │     conditions: { "threshold": 800 },
  │     frequency: "once",
  │     notify_email: true,
  │     notify_in_app: true,
  │     name: "NVDA below $800"
  │   }
  │
  ├─ [Toast: "Alert created: NVDA below $800"]
  │
  └─ [Bell icon on NVDA row now shows filled with "1" badge]
```

### Flow 5: Export Watchlist

```
[Watchlist Detail Page — "..." overflow menu]
  │
  ├─ User clicks "..." → menu shows: Export CSV, Import CSV, Delete Watchlist
  │
  ├─ User clicks "Export CSV"
  │
  ├─ [Client-side: generates CSV from current table data + visible columns]
  │   Filename: "AI-Stocks-2026-2026-02-18.csv"
  │
  └─ [Browser downloads file]
```

---

## 6. Data & Columns Specification

### Complete Column Registry

Every column available in the watchlist table, with data source and tier gating:

| Column ID | Display Name | Data Type | Format Example | Data Source | Tier |
|-----------|-------------|-----------|----------------|-------------|------|
| `symbol` | Symbol | string | "NVDA" | `tickers.symbol` | Free |
| `name` | Name | string | "NVIDIA Corporation" | `tickers.name` | Free |
| `logo` | Logo | image | 24x24 favicon | `tickers.logo_url` | Free |
| `asset_type` | Type | badge | "Stock" / "ETF" / "Crypto" | `tickers.asset_type` | Free |
| `exchange` | Exchange | string | "NASDAQ" | `tickers.exchange` | Free |
| `current_price` | Price | currency | "$892.45" | Polygon.io / Redis | Free |
| `price_change` | Change $ | currency | "+$12.30" | Polygon.io | Free |
| `price_change_pct` | Change % | percent | "+1.40%" | Polygon.io | Free |
| `volume` | Volume | number | "42.5M" | Polygon.io | Free |
| `market_cap` | Market Cap | currency | "$2.19T" | Polygon.io / screener_data | Free |
| `prev_close` | Prev Close | currency | "$880.15" | Polygon.io | Free |
| `range_52w` | 52-Week Range | range bar | Low — ● — High | screener_data | Free |
| `ic_score` | IC Score | number (1-100) | "78" (colored) | screener_data | **Premium** |
| `ic_rating` | IC Rating | badge | "Strong Buy" | screener_data | **Premium** |
| `value_score` | Value | number (1-100) | "65" | screener_data | **Premium** |
| `growth_score` | Growth | number (1-100) | "82" | screener_data | **Premium** |
| `profitability_score` | Profitability | number (1-100) | "71" | screener_data | **Premium** |
| `financial_health_score` | Fin. Health | number (1-100) | "55" | screener_data | **Premium** |
| `momentum_score` | Momentum | number (1-100) | "90" | screener_data | **Premium** |
| `analyst_consensus_score` | Analyst | number (1-100) | "73" | screener_data | **Premium** |
| `insider_activity_score` | Insider | number (1-100) | "40" | screener_data | **Premium** |
| `institutional_score` | Institutional | number (1-100) | "68" | screener_data | **Premium** |
| `sector_percentile` | Sector Rank | percentile | "Top 12%" | screener_data | **Premium** |
| `pe_ratio` | P/E | number | "32.5x" | screener_data | Free |
| `pb_ratio` | P/B | number | "18.2x" | screener_data | Free |
| `ps_ratio` | P/S | number | "28.1x" | screener_data | Free |
| `roe` | ROE | percent | "35.2%" | screener_data | Free |
| `roa` | ROA | percent | "18.7%" | screener_data | Free |
| `debt_to_equity` | D/E | number | "0.41" | screener_data | Free |
| `current_ratio` | Current Ratio | number | "4.17" | screener_data | Free |
| `revenue_growth` | Rev Growth | percent | "+122.4%" | screener_data | Free |
| `eps_growth` | EPS Growth | percent | "+88.1%" | screener_data | Free |
| `dividend_yield` | Div Yield | percent | "0.03%" | screener_data | Free |
| `annual_dividend` | Annual Div | currency | "$0.16" | screener_data | Free |
| `payout_ratio` | Payout | percent | "1.2%" | screener_data | Free |
| `perf_1d` | 1D % | percent | "+1.40%" | Polygon.io | Free |
| `perf_1w` | 1W % | percent | "+5.23%" | Derived (price history) | Free |
| `perf_1m` | 1M % | percent | "+12.8%" | Derived (price history) | Free |
| `perf_3m` | 3M % | percent | "+28.4%" | Derived (price history) | **Premium** |
| `perf_6m` | 6M % | percent | "+45.1%" | Derived (price history) | **Premium** |
| `perf_ytd` | YTD % | percent | "+18.9%" | Derived (price history) | **Premium** |
| `perf_1y` | 1Y % | percent | "+185.2%" | Derived (price history) | **Premium** |
| `reddit_rank` | Reddit Rank | number | "#3" | reddit_heatmap_daily | Free |
| `reddit_mentions` | Mentions (24h) | number | "1,247" | reddit_heatmap_daily | Free |
| `reddit_popularity` | Popularity | number | "87.3" | reddit_heatmap_daily | Free |
| `reddit_trend` | Trend | icon | ↑ / → / ↓ | reddit_heatmap_daily | Free |
| `reddit_rank_change` | Rank Chg | number | "+5" / "-2" | reddit_heatmap_daily | Free |
| `target_buy_price` | Target Buy | currency | "$800.00" | watch_list_items | Free |
| `target_sell_price` | Target Sell | currency | "$1,000.00" | watch_list_items | Free |
| `notes` | Notes | text (truncated) | "Watching for..." | watch_list_items | Free |
| `tags` | Tags | badge list | "AI", "Semis" | watch_list_items | Free |
| `alert_count` | Alerts | number badge | "2" | alert_rules (count) | Free |
| `added_at` | Date Added | date | "Feb 12, 2026" | watch_list_items | Free |

### Column-to-View Mapping

| Column | General | Performance | Fundamentals | Dividends | IC Score | Social | Compact |
|--------|:-------:|:-----------:|:------------:|:---------:|:--------:|:------:|:-------:|
| Symbol + Logo | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Name | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | |
| Price | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Change % | ✓ | | | | | ✓ | ✓ |
| Volume | ✓ | | | | | | |
| Market Cap | ✓ | | | | | | ✓ |
| 52-Week Range | ✓ | | | | | | |
| IC Score | ✓ | | | | ✓ | | ✓ |
| IC Rating | | | | | ✓ | | |
| Value Score | | | | | ✓ | | |
| Growth Score | | | | | ✓ | | |
| Profitability Score | | | | | ✓ | | |
| Financial Health Score | | | | | ✓ | | |
| Momentum Score | | | | | ✓ | | |
| 1D % | | ✓ | | | | | |
| 1W % | | ✓ | | | | | |
| 1M % | | ✓ | | | | | |
| 3M % | | ✓ | | | | | |
| 6M % | | ✓ | | | | | |
| YTD % | | ✓ | | | | | |
| 1Y % | | ✓ | | | | | |
| P/E | | | ✓ | | | | |
| P/B | | | ✓ | | | | |
| P/S | | | ✓ | | | | |
| ROE | | | ✓ | | | | |
| ROA | | | ✓ | | | | |
| D/E | | | ✓ | | | | |
| Current Ratio | | | ✓ | | | | |
| Rev Growth | | | ✓ | | | | |
| Div Yield | | | | ✓ | | | |
| Annual Div | | | | ✓ | | | |
| Payout Ratio | | | | ✓ | | | |
| Reddit Rank | | | | | | ✓ | |
| Mentions (24h) | | | | | | ✓ | |
| Popularity | | | | | | ✓ | |
| Trend | | | | | | ✓ | |
| Rank Change | | | | | | ✓ | |

### Data Freshness Requirements

| Data Source | Refresh Strategy | Latency Target |
|-------------|-----------------|----------------|
| Polygon.io (stock prices) | Poll every 15s during market hours, 60s after hours | <2s API response |
| Redis (crypto prices) | Poll every 5s | <500ms |
| screener_data materialized view | Daily refresh at 23:45 UTC | Stale OK (updated once daily) |
| reddit_heatmap_daily | Updated by pipeline every 4 hours | Stale OK |
| watch_list_items user data | On-demand (user action) | <500ms write |

---

## 7. UX & Design Requirements

### Layout and Information Architecture

```
┌─────────────────────────────────────────────────────────┐
│  [Watchlist Switcher ▼]          [+ Add Ticker] [⋮ More] │
│  "AI Stocks 2026" · 12 tickers · Last updated 2:34 PM    │
├─────────────────────────────────────────────────────────┤
│  Summary Bar: Avg IC Score: 72 | Avg Change: +1.2% |...  │
├─────────────────────────────────────────────────────────┤
│  View: [General ▼]  Filter: [All ▼]  Search: [🔍 ___]    │
│  ☐ Table   ☐ Heatmap                                      │
├─────────────────────────────────────────────────────────┤
│  ┌──────┬──────────┬────────┬────────┬───────┬────────┐  │
│  │ Logo │ Symbol   │ Price  │ Chg %  │ Vol   │ IC ▼   │  │
│  │ Name │          │        │        │       │ Score  │  │
│  ├──────┼──────────┼────────┼────────┼───────┼────────┤  │
│  │ 🟢   │ NVDA     │$892.45 │ +1.40% │ 42.5M │  78    │  │
│  │      │ NVIDIA.. │        │ +$12.3 │       │ Buy    │  │
│  ├──────┼──────────┼────────┼────────┼───────┼────────┤  │
│  │ ...  │ ...      │ ...    │ ...    │ ...   │ ...    │  │
│  └──────┴──────────┴────────┴────────┴───────┴────────┘  │
│                                                           │
│  Prices as of 2:34:15 PM ET · 🟢 Market Open             │
└─────────────────────────────────────────────────────────┘
```

### Mobile Layout (< 768px)

```
┌────────────────────────┐
│ [≡] AI Stocks 2026  [+]│
│ 12 tickers              │
├────────────────────────┤
│ View: [General ▼]       │
│ [All] [Stocks] [ETF]    │
├────────────────────────┤
│ ┌────────────────────┐  │
│ │ 🟢 NVDA    $892.45 │  │
│ │ NVIDIA     +1.40%  │  │
│ │ IC Score: 78 (Buy)  │  │
│ └────────────────────┘  │
│ ┌────────────────────┐  │
│ │ 🔴 AMD     $162.30 │  │
│ │ AMD Inc    -0.85%  │  │
│ │ IC Score: 65 (Buy)  │  │
│ └────────────────────┘  │
│ ...                      │
├────────────────────────┤
│ 🟢 Market Open · 2:34 PM│
└────────────────────────┘
```

### Empty States

**Scenario 1: First Visit (No Watchlists)**
- Should not occur — default watchlist auto-created
- Fallback: "Welcome! Let's create your first watchlist." + Create button

**Scenario 2: Watchlist with No Tickers**
- Centered illustration (abstract chart/graph icon)
- Headline: "Start tracking stocks you care about"
- Body: "Add tickers to see real-time prices, IC Score analytics, Reddit trends, and set price alerts — all in one place."
- Primary CTA: "Add Your First Ticker" (opens AddTickerModal)
- Quick-add chips: AAPL, MSFT, GOOGL, AMZN, TSLA, X:BTCUSD
- Secondary: "Import from CSV" link
- Tertiary: "Or explore the Screener to find stocks →"

**Scenario 3: Search with No Results**
- "No tickers match '{query}'. Try searching by symbol (e.g., AAPL) or company name."

### Error State Design Principles

1. **Never show raw error messages** — all errors get human-readable copy
2. **Always provide an action** — Retry button, Sign In link, or "Contact Support"
3. **Preserve user context** — error overlays should not wipe the current view; if data was previously loaded, keep stale data visible with a banner
4. **Distinguish temporary vs. persistent errors** — network blips get auto-retry; auth errors need user action

### Accessibility Requirements

- WCAG 2.1 AA compliance
- All interactive elements keyboard-navigable (Tab, Enter, Escape)
- Screen reader labels for all icons (bell icon: "Set alert for NVDA")
- Color is never the sole indicator — always pair with text/icon (green + "▲" for positive change)
- Focus management: modals trap focus; closing returns to trigger element
- Table headers use `<th scope="col">`; data cells reference headers via `headers` attribute
- Minimum touch target: 44x44px on mobile
- High contrast mode support for IC Score color coding

---

## 8. Technical Considerations

### API Reliability (Immediate Priority)

**Current Failure Investigation Checklist:**

1. Verify migration `010_watchlist_tables.sql` is applied in production:
   ```sql
   SELECT table_name FROM information_schema.tables
   WHERE table_name IN ('watch_lists', 'watch_list_items');
   ```

2. Verify `tickers` table exists (not `stocks`):
   ```sql
   SELECT EXISTS (
     SELECT FROM information_schema.tables WHERE table_name = 'tickers'
   );
   ```

3. Check database trigger functions exist:
   ```sql
   SELECT routine_name FROM information_schema.routines
   WHERE routine_name IN (
     'update_watch_lists_updated_at',
     'check_watch_list_item_limit',
     'create_default_watch_list'
   );
   ```

4. Test endpoints in isolation with curl against staging:
   ```bash
   curl -H "Authorization: Bearer $TOKEN" \
     https://staging.investorcenter.ai/api/v1/watchlists
   ```

5. Check Go service initialization — `watchListService` must be instantiated with a valid `PolygonClient` and database connection.

**Resilience Patterns to Implement:**

| Pattern | Where | Why |
|---------|-------|-----|
| Circuit breaker | Polygon.io API calls in `watchlist_service.go` | Prevent cascade failure when Polygon is down |
| Graceful degradation | `GetWatchListWithItems` | Return ticker data without prices if Polygon fails, rather than 500 |
| Request timeout | All external API calls | 5-second timeout on Polygon calls; don't block response indefinitely |
| Retry with backoff | Frontend API client | 3 retries with exponential backoff (1s, 2s, 4s) on 5xx responses |
| Stale-while-revalidate | Frontend price cache | Show previous prices immediately, update in background |

### Real-Time Data Strategy

**Recommendation: Continue with polling (not WebSockets)**

Rationale:
- Current 15-30 second polling is adequate for watchlist use cases (users are scanning, not trading)
- WebSocket infrastructure adds operational complexity (connection management, reconnection, load balancing) without proportional UX improvement for a watchlist view
- Polygon.io's REST API is already rate-limited; WebSocket would require a different pricing tier

**Polling Intervals:**

| Market State | Stock Prices | Crypto Prices | Screener Data |
|-------------|-------------|---------------|---------------|
| Market Open (9:30-16:00 ET) | 15 seconds | 5 seconds | No refresh |
| Pre/Post Market | 30 seconds | 5 seconds | No refresh |
| Market Closed | 60 seconds | 15 seconds | No refresh |
| Watchlist not in viewport | Pause polling | Pause polling | — |

**Optimization: Batch Price Fetching**

Current implementation calls `polygonClient.GetQuote()` per ticker sequentially. For a 50-ticker watchlist, this is 50 serial API calls.

Recommendation:
- Polygon's Snapshot API supports batch quotes: `GET /v2/snapshot/locale/us/markets/stocks/tickers?tickers=AAPL,MSFT,GOOGL,...`
- Implement `GetBatchQuotes(symbols []string)` in `polygon.go`
- Max 50 symbols per batch request; split larger watchlists into parallel batches
- Expected improvement: 50 serial calls (~5s) → 1 batch call (~200ms)

### Performance Requirements

| Metric | Target | Measurement |
|--------|--------|-------------|
| Watchlist list page load (cold) | < 1.5 seconds | Time from navigation to all cards rendered |
| Watchlist detail page load (cold) | < 2 seconds | Time from navigation to table rendered with prices |
| Watchlist detail page load (warm) | < 500ms | Subsequent navigations (cached data + delta refresh) |
| Add ticker response | < 500ms | Time from click "Add" to row appearing in table |
| View switch | < 200ms | Time from click view tab to table re-rendered |
| Sort/filter | < 100ms | Client-side operation, no API call |
| CSV export (100 tickers) | < 1 second | Client-side generation |

### IC Score Data Integration (New Backend Work)

The current `GetWatchListItemsWithData` query joins `watch_list_items` → `tickers` → `reddit_heatmap_daily`. To add IC Score data, extend this query to also join `screener_data`:

```sql
-- Pseudocode for extended query
SELECT
  wli.*,
  t.name, t.exchange, t.asset_type, t.logo_url,
  rhd.reddit_rank, rhd.reddit_mentions, ...,
  sd.ic_score, sd.ic_rating, sd.value_score, sd.growth_score, ...
  sd.pe_ratio, sd.pb_ratio, sd.ps_ratio, sd.roe, sd.roa, ...
FROM watch_list_items wli
JOIN tickers t ON wli.symbol = t.symbol
LEFT JOIN LATERAL (
  SELECT * FROM reddit_heatmap_daily
  WHERE symbol = wli.symbol ORDER BY date DESC LIMIT 1
) rhd ON true
LEFT JOIN screener_data sd ON wli.symbol = sd.symbol
WHERE wli.watch_list_id = $1
ORDER BY wli.display_order;
```

This is efficient because `screener_data` is a materialized view already indexed by symbol.

### Performance Data (Price History) Backend Work

For the Performance view (1W%, 1M%, 3M%, etc.), historical price data is needed. Options:

1. **Use `stock_prices` TimescaleDB hypertable** (exists in IC Score service): query closing price at T-7d, T-30d, etc. and compute % change. Requires a new API endpoint on the IC Score service or a cross-service query.

2. **Use Polygon.io `previousClose` endpoint**: Only provides 1D. For longer periods, use Polygon's Aggregates endpoint for specific dates.

3. **Pre-compute in `screener_data` view** (recommended): Add `perf_1w`, `perf_1m`, `perf_3m`, `perf_6m`, `perf_ytd`, `perf_1y` columns to the materialized view. Compute during the daily refresh at 23:45 UTC. This keeps the watchlist query simple and avoids per-ticker API calls.

**Recommendation**: Option 3. Add performance columns to `screener_data`. The data is 1-day stale, which is acceptable for a watchlist performance view (intraday performance is shown via the real-time 1D change).

### Data Persistence

| Data | Storage | Sync Strategy |
|------|---------|--------------|
| Watchlist metadata | PostgreSQL `watch_lists` | Server-authoritative |
| Watchlist items | PostgreSQL `watch_list_items` | Server-authoritative |
| User's selected view | `localStorage` | Client-only, no sync |
| User's sort/filter state | `localStorage` | Client-only, no sync |
| Column widths (if resizable) | `localStorage` | Client-only, no sync |

---

## 9. Monetization & Freemium Strategy

### Tier Comparison

| Feature | Free | Premium |
|---------|------|---------|
| Maximum watchlists | 3 | Unlimited |
| Maximum tickers per watchlist | 10 | 100 |
| Data views | General, Compact, Social | All 7 views + Custom |
| Columns: Price, Change, Volume, Market Cap | ✓ | ✓ |
| Columns: P/E, P/B, ROE, Dividend Yield | ✓ | ✓ |
| Columns: IC Score, IC Rating | Blurred (teaser) | ✓ |
| Columns: IC Sub-factors (10 scores) | Locked | ✓ |
| Columns: Performance 3M+ (3M, 6M, YTD, 1Y) | Locked | ✓ |
| Columns: Sector Percentile | Locked | ✓ |
| Reddit data in watchlist | ✓ (rank + trend only) | ✓ (full: mentions, popularity, rank change) |
| Heatmap visualization | 1 config | 10 configs + IC Score metric |
| Alert rules | 10 total | 100 total |
| CSV export | Basic (price data only) | Full (all columns) |
| CSV import | ✓ | ✓ |
| Smart watchlists (auto-updating) | Locked | ✓ |
| Screener → Watchlist bulk add | 5 at a time | 500 at a time |
| IC Score mini-breakdown tooltip | Locked | ✓ |
| Watchlist summary metrics bar | Basic (count, avg change) | Full (IC Score avg, sector rank) |

### Upgrade Prompt Strategy

Prompts should feel informative, not punitive. The goal is to let free users *see* the value of premium data, not to block their workflow.

**Prompt 1: IC Score Column Teaser**
- In the General view, show IC Score column with values blurred/pixelated
- Tooltip on hover: "IC Score rates stocks 1-100 based on 10 fundamental factors. Upgrade to see scores." + "See Plans" button
- Why this works: IC Score is InvestorCenter's unique asset. Showing it exists (but gated) creates curiosity that generic columns cannot.

**Prompt 2: Item Limit Hit**
- When free user tries to add 11th ticker: modal says "Free accounts can track up to 10 tickers per watchlist. Upgrade to track up to 100."
- Include: current count (10/10), ticker they tried to add, "Upgrade" CTA, "Maybe later" dismiss
- Why this works: high-intent moment — user actively wants to track more.

**Prompt 3: View Lock**
- Clicking a Premium-only view (Fundamentals, Dividends, IC Score, Performance): show view preview with blurred data and overlay "Unlock [View Name] with Premium"
- Why this works: shows the shape of the data (columns, layout) so users can evaluate before buying.

**Prompt 4: Smart Watchlist Feature Discovery**
- After user creates 3rd watchlist or adds 15th total ticker: "Tip: Premium users can create Smart Watchlists that auto-update from screener filters. Your 'AI Stocks' list could auto-add new high-scoring AI stocks." + "Learn more"
- Why this works: surfaces a differentiating feature at a natural engagement milestone.

### Benchmarking Against Competitors

| Platform | Free Tier Limits | Premium Price | Gating Strategy |
|----------|-----------------|--------------|-----------------|
| Yahoo Finance | Unlimited watchlists/tickers | $35/month (Yahoo Finance Plus) | Advanced charts, research reports, portfolio analytics |
| StockAnalysis | Unlimited watchlists/tickers | $24.99/month (Pro) | Real-time prices, advanced screener, financial data export |
| CNBC | Unlimited watchlists | $35/month (Pro) | Ad-free, advanced charts, real-time data |
| **InvestorCenter (Recommended)** | 3 watchlists / 10 per list | **$14.99/month** | IC Score, advanced views, smart watchlists, extended performance data |

**Pricing Rationale**: $14.99/month undercuts all competitors meaningfully while the product is still establishing market position. IC Score is a differentiated feature that justifies premium pricing. Raise to $19.99/month once smart watchlists and full IC Score integration are proven.

---

## 10. Go-to-Market & Rollout

### Phase 1: Fix (Weeks 1-2)

**Objective**: Make the feature functional. Zero users should see "Failed to fetch."

| Task | Owner | Duration | Priority |
|------|-------|----------|----------|
| Diagnose and fix API failures (P0-1) | Backend | 2-3 days | Critical |
| Fix test suite table name references (P0-2) | Backend | 1 day | Critical |
| Verify default watchlist auto-creation (P0-3) | Backend | 1 day | High |
| End-to-end CRUD verification in staging (P0-4) | Full stack | 2-3 days | Critical |
| Deploy fixes to production | DevOps | 1 day | Critical |
| Basic error state UI (replace "Failed to fetch") (P1-6 partial) | Frontend | 2 days | High |

**Exit Criteria**: All 12 watchlist endpoints return correct responses. Zero 5xx errors over 48 hours in production. Users can create, view, add tickers, and delete.

**No announcement**. Fix silently. Users who tried the feature before and saw errors will discover it works when they return.

### Phase 2: Core (Weeks 3-8)

**Objective**: Deliver a watchlist that matches or exceeds StockAnalysis.com's core functionality.

| Task | Owner | Duration | Priority |
|------|-------|----------|----------|
| Configurable column views (P1-1) | Frontend | 5 days | High |
| Watchlist switcher (P1-2) | Frontend | 3 days | High |
| Table sorting and filtering (P1-3) | Frontend | 3 days | High |
| Real-time price display enhancement (P1-4) | Frontend + Backend | 4 days | High |
| Empty state and onboarding (P1-5) | Frontend + Design | 3 days | High |
| Complete error state handling (P1-6) | Frontend | 2 days | High |
| Mobile responsive table/cards (P1-7) | Frontend | 5 days | Medium |
| CSV import/export (P1-8) | Frontend + Backend | 3 days | Medium |
| IC Score data join in backend (Tech foundation for P2-5) | Backend | 3 days | High |
| Performance data columns in screener_data (Tech foundation) | Backend + DB | 3 days | Medium |
| Batch price fetching optimization | Backend | 2 days | Medium |

**Beta Release (end of Week 5)**: Feature-flag to 10% of authenticated users. Collect:
- Task completion rates (can users add tickers, switch views, sort?)
- Performance metrics (page load times, API latency)
- Error rates

**Full Release (Week 8)**: Roll to 100%. Announcement:
- In-app banner on dashboard: "Watchlists are here. Track your favorite stocks with real-time data, multiple views, and more."
- Email to registered users who have not visited in >14 days

### Phase 3: Differentiate (Weeks 9-16)

**Objective**: Ship features no competitor has. Make InvestorCenter's watchlist the reason users choose us over Yahoo Finance or StockAnalysis.

| Task | Owner | Duration | Priority |
|------|-------|----------|----------|
| Screener → Watchlist integration (P2-1) | Frontend + Backend | 5 days | High |
| Reddit Trends in watchlist (P2-2) | Frontend | 3 days | High |
| Crypto + Equity unified polish (P2-3) | Frontend | 2 days | Medium |
| Inline alert creation (P2-4) | Frontend + Backend | 5 days | High |
| IC Score deep integration (P2-5) | Frontend | 5 days | High |
| Heatmap as tab (P2-6) | Frontend | 3 days | Medium |
| Watchlist summary metrics bar (P2-7) | Frontend | 3 days | Medium |
| Premium gating implementation | Frontend + Backend | 3 days | High |
| Smart watchlists (stretch) | Full stack | 8 days | Low |

**Announcement Strategy (Phase 3 launch)**:
- Blog post: "The smartest watchlist in retail investing" — positioning IC Score + Reddit + Screener integration
- Social media: Twitter/X thread showing workflow GIFs (Screener → Add to Watchlist → Set Alert → Get notified)
- Product Hunt launch (if timing aligns with overall GTM)
- Email campaign to free users highlighting premium features with 14-day free trial

---

## 11. Risks & Mitigations

### Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **Polygon API rate limiting with large watchlists** | High | High — 50-ticker watchlists with 15s polling = 200 calls/min | Implement batch snapshot endpoint; add Redis cache layer for prices with 10s TTL |
| **screener_data view staleness** | Medium | Low — data is 1-day old by design | Clearly label "Data as of {date}" in Fundamentals/IC Score views; users expect end-of-day for fundamentals |
| **Database migration state unknown in production** | High | Critical — root cause of current failures | Run migration audit as first action in Phase 1; add migration version check to health endpoint |
| **N+1 query in GetWatchListItemsWithData** | Medium | Medium — degrades with watchlist size | Batch the Polygon calls; ensure DB query uses single JOIN (already does); add pagination for >50 items |
| **Frontend bundle size increase from D3 heatmap** | Low | Low | D3 is already loaded for heatmap; column views are lightweight |

### Market Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **Competitors ship similar features** | Medium | Medium — StockAnalysis could add social data | Ship Phase 3 quickly; IC Score is proprietary and cannot be replicated |
| **IC Score accuracy questioned by users** | Medium | High — undermines premium value proposition | Add methodology page; show score history/change over time; invite feedback |
| **Low perceived value of premium tier** | Medium | High — conversion won't hit targets | A/B test gate thresholds (e.g., 5 vs 10 free tickers); test different premium features as gates |

### User Adoption Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **Users don't discover watchlists** | Medium | High | Add "Save to Watchlist" CTAs on ticker pages, screener, Reddit Trends; onboarding tutorial for new signups |
| **Users create watchlist but don't return** | High | High | Implement email digest: "Weekly Watchlist Update — NVDA +5.2%, 2 alerts triggered" |
| **Free tier too restrictive → users churn** | Medium | High | Monitor: if >20% of users who hit the 10-ticker limit don't return within 7 days, raise free limit to 25 |
| **Trust deficit from previous broken state** | Medium | Medium | Silent fix (no "we fixed it" announcement that reminds users it was broken); let quality speak for itself |

---

## 12. Open Questions

The following decisions require stakeholder alignment before or during development:

### Product Decisions

| # | Question | Options | Recommendation | Stakeholder |
|---|----------|---------|----------------|-------------|
| 1 | **Should the default watchlist be deletable?** | A) Protected (never deletable), B) Deletable but a new default auto-recreates | A — Protected. Prevents empty state edge cases and ensures every user always has at least one watchlist. | Product |
| 2 | **Free tier: 10 tickers/list or 25?** | A) 10 (current DB trigger), B) 25 | Start at 10, monitor churn data, raise to 25 if limit-hit → churn correlation exceeds 20%. | Product + Growth |
| 3 | **Should smart watchlists (dynamic screener filters) be in Phase 3 or deferred?** | A) Phase 3, B) Phase 4 / separate initiative | Defer to Phase 4. It requires materialized view re-architecture and adds complexity. Ship the Screener → Watchlist *manual* add flow first. | Engineering + Product |
| 4 | **Should we show IC Score to free users (blurred/teaser) or hide the column entirely?** | A) Blurred teaser, B) Hidden, C) Show for first 3 tickers only | A — Blurred teaser. Increases awareness and creates upgrade curiosity. Hiding it means free users never learn about our differentiation. | Product + Growth |
| 5 | **Public/shared watchlists: include in roadmap?** | A) Add to Phase 4 roadmap, B) Deprioritize indefinitely | B — Deprioritize. Sharing requires moderation, abuse prevention, and privacy controls. Not worth the complexity at our scale. | Product |
| 6 | **Should we support guest watchlists (no login)?** | A) Yes (localStorage-based), B) No (require auth) | B — Require auth. Watchlists are the strongest account-creation incentive we have. Giving them away without signup removes our best conversion lever. | Growth |

### Technical Decisions

| # | Question | Options | Recommendation | Stakeholder |
|---|----------|---------|----------------|-------------|
| 7 | **How to serve performance data (1W%, 1M%, etc.)?** | A) Add to screener_data view, B) New API endpoint querying stock_prices, C) Polygon aggregates API | A — Add to screener_data. Simplest, single JOIN, acceptable staleness. | Engineering |
| 8 | **Batch price API: build our own proxy or use Polygon Snapshot?** | A) Polygon Snapshot endpoint, B) Redis cache populated by a dedicated price CronJob | A for stocks (Polygon Snapshot is designed for this), continue with B for crypto (already implemented). | Engineering |
| 9 | **Should watchlist data refresh use React Query / SWR or continue with custom polling?** | A) Adopt React Query (TanStack Query), B) Keep custom useEffect + setInterval | A — Adopt React Query. It provides stale-while-revalidate, deduplication, window focus refetching, and retry logic out of the box. The custom polling code is fragile and will get worse with more data sources. | Frontend Engineering |
| 10 | **What is the maximum watchlist size we need to support?** | A) 100 (current premium limit), B) 500, C) 1000 | A — 100. No retail investor actively monitors >100 tickers in a single list. Keep the limit to ensure table performance stays fast. | Engineering + Product |

### Design Decisions

| # | Question | Recommendation | Stakeholder |
|---|----------|----------------|-------------|
| 11 | **Should view switching be tabs or a dropdown?** | Dropdown. With 7+ views, tabs create horizontal overflow. Dropdown is cleaner and scales. | Design |
| 12 | **Should the heatmap be a separate page or an inline tab?** | Inline tab in Phase 2 (P2-6). Reduces navigation friction. The existing `/heatmap` route can remain as a permalink. | Design + Frontend |
| 13 | **Light/dark mode: is the heatmap ready for both?** | Yes — `WatchListHeatmap.tsx` already handles theme-aware colors. Verify contrast ratios in both modes during QA. | Design |

---

## Appendix A: Existing Technical Implementation Reference

### Backend Files (Go)

| File | Purpose |
|------|---------|
| `backend/migrations/010_watchlist_tables.sql` | Schema: `watch_lists`, `watch_list_items`, triggers |
| `backend/models/watchlist.go` | Data models and DTOs |
| `backend/handlers/watchlist_handlers.go` | HTTP handlers (12 endpoints) |
| `backend/services/watchlist_service.go` | Business logic, Polygon integration |
| `backend/database/watchlists.go` | SQL queries |
| `backend/handlers/alert_handlers.go` | Alert CRUD handlers (7 endpoints) |
| `backend/services/alert_service.go` | Alert business logic |
| `backend/services/alert_processor.go` | Alert evaluation engine |
| `backend/services/polygon.go` | Real-time price client |

### Frontend Files (Next.js / React)

| File | Purpose |
|------|---------|
| `app/watchlist/page.tsx` | Dashboard (list all watchlists) |
| `app/watchlist/[id]/page.tsx` | Watchlist detail with ticker table |
| `app/watchlist/[id]/heatmap/page.tsx` | Heatmap visualization page |
| `components/watchlist/WatchListTable.tsx` | Table component |
| `components/watchlist/CreateWatchListModal.tsx` | Create watchlist modal |
| `components/watchlist/AddTickerModal.tsx` | Add ticker with search autocomplete |
| `components/watchlist/EditTickerModal.tsx` | Edit ticker metadata |
| `components/watchlist/WatchListHeatmap.tsx` | D3 treemap visualization |
| `components/watchlist/HeatmapConfigPanel.tsx` | Heatmap settings panel |
| `lib/api/watchlist.ts` | Watchlist API types and methods |
| `lib/api/heatmap.ts` | Heatmap API types and methods |
| `lib/hooks/useRealTimePrice.ts` | Real-time price polling hook |

### Database Tables

| Table | Rows (est.) | Notes |
|-------|-------------|-------|
| `watch_lists` | ~0 (feature broken) | UUID PK, user_id FK |
| `watch_list_items` | ~0 (feature broken) | UUID PK, watch_list_id FK, symbol FK to tickers |
| `tickers` | ~25,000 | Master ticker list (stocks + ETFs + crypto) |
| `screener_data` | ~10,000 | Materialized view, refreshed daily |
| `reddit_heatmap_daily` | ~500/day | TimescaleDB, latest per symbol via LATERAL |
| `alert_rules` | ~0 (feature broken) | UUID PK, watch_list_id FK required |
| `stock_prices` | ~millions | TimescaleDB hypertable (IC Score service) |

### API Route Summary

All under `/api/v1/`, authenticated via `auth.AuthMiddleware()`:

```
GET    /watchlists                          → ListWatchLists
POST   /watchlists                          → CreateWatchList
GET    /watchlists/:id                      → GetWatchList (+ real-time prices)
PUT    /watchlists/:id                      → UpdateWatchList
DELETE /watchlists/:id                      → DeleteWatchList
POST   /watchlists/:id/items               → AddTickerToWatchList
DELETE /watchlists/:id/items/:symbol        → RemoveTickerFromWatchList
PUT    /watchlists/:id/items/:symbol        → UpdateWatchListItem
POST   /watchlists/:id/bulk                 → BulkAddTickers
POST   /watchlists/:id/reorder              → ReorderWatchListItems
GET    /watchlists/:id/heatmap              → GetHeatmapData
GET    /alerts                              → ListAlerts
POST   /alerts                              → CreateAlert
GET    /alerts/:id                          → GetAlert
PUT    /alerts/:id                          → UpdateAlert
DELETE /alerts/:id                          → DeleteAlert
GET    /alerts/logs                         → GetAlertLogs
POST   /alerts/logs/:id/read               → MarkAlertRead
POST   /alerts/logs/:id/dismiss             → DismissAlert
```

---

*End of Document*

*This PRD should be reviewed by Engineering, Design, and Growth stakeholders before sprint planning begins. Phase 1 (Fix) can begin immediately — it requires no design work and addresses a production-critical bug.*
