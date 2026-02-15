# InvestorCenter Stock Screener: Product Improvement Plan

**Author:** Product Strategy Review
**Date:** February 2026
**Status:** Proposal

---

## Executive Summary

InvestorCenter's screener sits on a goldmine of data it doesn't expose. The database contains 50+ metrics across valuation, profitability, growth, risk, technicals, fair value, and sentiment — yet the screener surfaces only 6 filterable fields and 9 table columns. Worse, two fields that _are_ in the materialized view (beta and IC Score) return hardcoded zeros due to placeholder query mappings. The path to a competitive screener is primarily a frontend + API exposure problem, not a data problem. Most of the work is wiring up what already exists.

This plan prioritizes ruthlessly: Phase 1 fixes embarrassing gaps with data that's already computed. Phase 2 builds the workflows that make users stay. Phase 3 leverages InvestorCenter's unique assets (IC Score sub-factors, fair value models, Reddit sentiment) to create features no competitor offers.

---

## 1. Competitive Analysis

### Current Positioning vs. Competitors

| Capability | IC (Now) | Finviz | TradingView | Stock Analysis | Koyfin | Simply Wall St | Yahoo Finance |
|---|---|---|---|---|---|---|---|
| Filter count | 6 | 65+ | 100+ | 40+ | 50+ | 30+ | 25+ |
| Technical filters | 0 | 20+ | 50+ | 10+ | 15+ | 0 | 5+ |
| Custom columns | No | Yes | Yes | Yes | Yes | No | Limited |
| Save screens | No | Free (1) | Yes | Yes | Yes | Yes | Yes |
| Export CSV | No | Pro | Yes | Yes | Yes | No | No |
| URL persistence | No | Yes | Yes | Yes | Yes | No | Yes |
| Visual screening | No | Heat map | Charts | No | Charts | Snowflake | No |
| Proprietary score | IC Score (broken) | No | No | Score | No | Snowflake | No |
| Fair value | In DB, not exposed | No | No | Yes | No | Yes | Yes |
| Sentiment filter | In DB, not exposed | No | No | No | No | No | No |
| Mobile UX | Passable | Poor | Good | Good | Poor | Excellent | Good |

### Critical Gaps (Ranked by User Impact)

1. **IC Score and Beta return 0.0** — The two most differentiating metrics are broken in production. The materialized view has the data; the Go query has placeholder mappings. This is a bug, not a feature gap.
2. **6 filters for 20,000 stocks is unusable** — Competitors offer 25-100+ filters. The database already has 40+ metrics in `fundamental_metrics_extended` alone. Exposing even 15 more would be transformative.
3. **No filter persistence** — Filters reset on page reload. No URL params, no saved screens. Users lose context constantly.
4. **No export** — Every serious competitor offers CSV export. Active investors need to pull data into spreadsheets.
5. **Missing "detail page" metrics** — The ticker detail page shows P/B, P/S, ROE, ROA, margins, D/E, current ratio, beta, and more. Users can see these on individual stocks but can't screen by them.
6. **No technical filters** — RSI, moving averages, MACD, and Bollinger Bands are all computed daily into `technical_indicators`. Zero of them are available for screening.
7. **No fair value screening** — DCF fair value, Graham Number, and EPV are computed and stored. No competitor except Simply Wall St and Stock Analysis offers this. Huge missed opportunity.

---

## 2. User Personas & Jobs-to-Be-Done

### Persona 1: The Dividend Income Builder
**Who:** Semi-retired investor, 55-65, building a portfolio for reliable income. Holds 30-50 stocks.
**Primary JTBD:** "Find stocks that pay a growing, sustainable dividend without excessive risk."
**Key filters needed:** Dividend yield, payout ratio, consecutive dividend years, dividend growth rate, debt/equity, current ratio, IC Score.
**Current screener fit:** POOR — Only yield is filterable. No payout ratio, no dividend growth history, no financial health filters. Can't separate 8% yield traps from 3% yield aristocrats.

### Persona 2: The Growth Stock Hunter
**Who:** Active retail investor, 25-40, seeking high-growth companies before they become mainstream picks.
**Primary JTBD:** "Find fast-growing companies with improving fundamentals before the crowd catches on."
**Key filters needed:** Revenue growth (YoY + 3Y CAGR), EPS growth, FCF growth, gross margin expansion, analyst upgrades, insider buying, IC Score growth sub-score.
**Current screener fit:** POOR — Only YoY revenue growth is available. No EPS growth, no margin trends, no analyst/insider signals. The single growth filter makes it impossible to distinguish quality growth from revenue-buying.

### Persona 3: The Value-Oriented Analyst
**Who:** Experienced investor or financial analyst, 30-50, seeking undervalued stocks with a margin of safety.
**Primary JTBD:** "Find stocks trading below intrinsic value with strong fundamentals and limited downside risk."
**Key filters needed:** P/E, P/B, P/S (with sector percentiles), DCF upside %, Graham Number vs. price, EV/EBITDA, ROE, ROIC, debt/equity, beta, Sharpe ratio.
**Current screener fit:** MODERATE — P/E and dividend yield filters exist. But no P/B, P/S, ROE, or fair value filters despite all being computed. Can't screen for "stocks trading >20% below DCF fair value" even though the data exists.

### Persona 4: The Momentum Swing Trader
**Who:** Active trader, 25-45, making decisions based on technical signals and market momentum.
**Primary JTBD:** "Find stocks showing bullish technical setups with confirming volume and positive sentiment."
**Key filters needed:** RSI, price vs. 50/200 SMA, MACD signal, volume spike detection, 1M/3M/6M momentum returns, news sentiment score.
**Current screener fit:** NOT SERVED — Zero technical filters. Despite computing RSI, MACD, SMAs, and Bollinger Bands daily, none are available for screening.

### Jobs-to-Be-Done Map

| Job | Currently Served? |
|---|---|
| Find stocks by basic valuation (P/E) | Yes |
| Filter by sector and market cap | Yes |
| Find dividend-paying stocks | Partially (yield only) |
| Find undervalued stocks with margin of safety | No |
| Screen by financial health metrics | No |
| Screen by profitability metrics | No |
| Find stocks with technical momentum | No |
| Save and reuse a screen | No |
| Share a screen with someone | No |
| Export results for analysis | No |
| Screen using IC Score sub-factors | No |
| Find stocks below fair value | No |

---

## 3. Prioritized Improvement Roadmap

### Phase 1 — Quick Wins (1-2 Sprints)

These require minimal backend work because the data already exists in the `screener_data` materialized view or in `fundamental_metrics_extended`.

#### 1.1 Fix IC Score and Beta (Bug Fix — 1-2 days)
- **Problem:** `backend/database/screener.go` maps `ic_score` sort to `market_cap` and `beta` sort to `roe` as placeholders. The `screener_data` materialized view already has `ic_score` (from `ic_scores.overall_score`) and `beta` (from `fundamental_metrics_extended.beta`), but the query code hardcodes them to 0.0.
- **Fix:** Update the Go query in `GetScreenerStocks()` to read `ic_score` and `beta` from the view instead of returning defaults. Fix the sort column mapping for both fields.
- **Impact:** IC Score becomes the screener's centerpiece feature overnight.

#### 1.2 Expose Existing Metrics as Filters (3-5 days)
Add 12 high-value filters using data already in `fundamental_metrics_extended` and `valuation_ratios` (already joined in the materialized view or trivially addable):

| Filter | Source Column | Why It Matters |
|---|---|---|
| P/B Ratio | `valuation_ratios.ttm_pb_ratio` | Already in view, not exposed |
| P/S Ratio | `valuation_ratios.ttm_ps_ratio` | Already in view, not exposed |
| ROE | `fundamental_metrics_extended.roe` | Already in view, not exposed |
| ROA | `fundamental_metrics_extended.roa` | Add to view |
| Gross Margin | `fundamental_metrics_extended.gross_margin` | Add to view |
| Net Margin | `fundamental_metrics_extended.net_margin` | Add to view |
| Debt/Equity | `fundamental_metrics_extended.debt_to_equity` | Add to view |
| Current Ratio | `fundamental_metrics_extended.current_ratio` | Add to view |
| Beta | `fundamental_metrics_extended.beta` | Already in view (broken) |
| EPS Growth YoY | `fundamental_metrics_extended.eps_growth_yoy` | Add to view |
| Payout Ratio | `fundamental_metrics_extended.payout_ratio` | Add to view |
| Consecutive Div Years | `fundamental_metrics_extended.consecutive_dividend_years` | Add to view |

**Backend work:** Alter the materialized view migration to add ~8 columns from `fundamental_metrics_extended`. Add corresponding filter params to the Go handler and database query. Add sort mappings.

**Frontend work:** Add filter UI components for each new metric (min/max range inputs). Add columns to the results table.

#### 1.3 URL-Based Filter Persistence (2-3 days)
- Serialize filter state to URL query params (e.g., `?pe_max=15&div_min=3&sectors=Technology,Healthcare`)
- Read params on page load to restore filters
- Users can bookmark and share screener configurations
- **Implementation:** Replace `useState` filter management with `useSearchParams` from Next.js, syncing bidirectionally.

#### 1.4 CSV Export (1-2 days)
- Add "Export CSV" button that downloads current filtered results
- Include all visible columns plus a few extras (industry, beta, IC Score)
- Client-side generation from already-loaded data (no new API endpoint needed since all data is fetched)
- Limit to 10,000 rows per export

#### 1.5 Column Customization (2-3 days)
- Add a column picker dropdown/modal
- Let users show/hide columns from available metrics
- Persist column selection in `localStorage`
- Default column set matches current: Symbol, Name, Market Cap, Price, Change, P/E, Div Yield, Rev Growth, IC Score

#### 1.6 Improve Quick Screen Presets (1 day)
- Update presets to use newly available filters:
  - **Value Stocks:** P/E < 15, P/B < 2, Dividend Yield > 2%, Debt/Equity < 1.5
  - **Growth Stocks:** Revenue Growth > 20%, EPS Growth > 15%, Gross Margin > 40%
  - **Quality Stocks:** IC Score > 70, ROE > 15%, Net Margin > 10%, Current Ratio > 1.5
  - **Dividend Champions:** Div Yield > 3%, Consecutive Years > 10, Payout Ratio < 75%, Debt/Equity < 1
  - **NEW — Undervalued:** P/E Sector Percentile < 30, P/B < 1.5, ROE > 12%
  - **NEW — Momentum:** (placeholder for Phase 2 when technical filters exist)

#### 1.7 Industry Sub-Filtering (1-2 days)
- The `screener_data` view already has an `industry` column (from `tickers.industry`)
- Add industry as a dependent filter under sector (when sector is selected, show its industries)
- This alone massively improves specificity — "Healthcare > Biotechnology" vs. just "Healthcare"

**Phase 1 Total Effort:** ~2 sprints (3-4 weeks)
**Phase 1 Impact:** Takes the screener from 6 filters to 18+, fixes the broken IC Score display, adds persistence/export, and makes presets actually useful.

---

### Phase 2 — Core Enhancements (1-2 Months)

#### 2.1 Saved Screeners (1 week)
- Authenticated users can save filter configurations with a name
- Store in a new `saved_screens` table (user_id, name, filters JSON, created_at, updated_at)
- "My Screens" dropdown alongside Quick Screen presets
- CRUD: create, rename, update, delete saved screens
- Limit: 20 saved screens per user (free), unlimited (future premium)

#### 2.2 Technical Indicator Filters (1-2 weeks)
Add filters from the `technical_indicators` hypertable (data already computed daily):

| Filter | What It Enables |
|---|---|
| RSI (14-day) | "Show me oversold stocks (RSI < 30)" |
| Price vs. SMA 50 | "Stocks trading above their 50-day moving average" |
| Price vs. SMA 200 | "Stocks in long-term uptrend (above 200 SMA)" |
| MACD Signal | "Stocks with bullish MACD crossover" |

**Backend work:** Add a technical indicators lateral join to the materialized view (latest RSI, SMA50, SMA200, MACD per ticker). Or: create a separate lightweight view and join at query time.

**Performance consideration:** Adding another lateral join to 5,000+ tickers may slow the materialized view refresh. Alternative: pre-compute a `screener_technical_data` table that the daily `technical_indicators` pipeline populates, then join.

#### 2.3 Fair Value & Intrinsic Value Filters (1 week)
Expose the computed fair value metrics from `fundamental_metrics_extended`:

| Filter | Source | User Story |
|---|---|---|
| DCF Upside % | `dcf_upside_percent` | "Stocks trading >20% below DCF fair value" |
| Graham Number vs. Price | Derived | "Stocks below Graham Number" |
| EPV Fair Value vs. Price | Derived | "Stocks below earnings power value" |

This is a **massive differentiator** — no free screener offers DCF-based intrinsic value screening. Simply Wall St charges for this.

#### 2.4 Risk Metric Filters (1 week)
From the `risk_metrics` hypertable:

| Filter | Use Case |
|---|---|
| Sharpe Ratio (1Y) | Risk-adjusted return screening |
| Max Drawdown (1Y) | "Stocks that haven't crashed >20% in a year" |
| Alpha (1Y) | "Stocks outperforming the S&P 500" |

#### 2.5 Watchlist Integration (3-5 days)
- Add "Add to Watchlist" button on each screener result row (icon button)
- Batch add: select multiple stocks and add to watchlist in one action
- Show watchlist membership indicator on screener results (dot/star if already in a watchlist)
- Quick link: "View in Watchlist" from screener results

#### 2.6 Alert Creation from Screener (3-5 days)
- "Create Alert" button: when a stock enters/exits the current filter criteria, notify user
- Leverages existing `alerts` table in the database
- Example: user sets up "P/E < 12, Div Yield > 3%, IC Score > 60" — gets email when AAPL drops into that range

#### 2.7 Multi-Sort (2-3 days)
- Allow sorting by primary + secondary column (e.g., sort by IC Score desc, then by Div Yield desc)
- UI: click to set primary sort, shift+click for secondary
- API: extend `sort` param to accept comma-separated values: `?sort=ic_score,dividend_yield&order=desc,desc`

#### 2.8 Comparison Mode (1 week)
- Checkbox column to select 2-5 stocks
- "Compare" button opens side-by-side comparison view
- Shows all available metrics for selected stocks in a comparison table
- Links to existing ticker detail pages for deep dive

#### 2.9 IC Score Sub-Factor Filters (1 week)
Expose the 10 IC Score sub-factors as individual filters. This is unique to InvestorCenter:

| Sub-Factor Filter | Example Use |
|---|---|
| Value Score (0-100) | "Stocks with high value scores" |
| Growth Score (0-100) | "Stocks with strong growth signals" |
| Profitability Score (0-100) | "Highly profitable companies" |
| Financial Health Score (0-100) | "Financially healthy companies" |
| Momentum Score (0-100) | "Stocks with positive momentum" |
| Analyst Consensus Score (0-100) | "Analyst favorites" |
| Insider Activity Score (0-100) | "Stocks insiders are buying" |
| Institutional Score (0-100) | "Stocks institutions are accumulating" |
| News Sentiment Score (0-100) | "Positive news sentiment" |
| Technical Score (0-100) | "Bullish technical signals" |

**Backend:** Add IC Score sub-factors to the materialized view (join `ic_scores` table for all score columns). Add filter params.

**This is the moat.** No other screener lets you filter by "insider activity score > 70 AND analyst consensus score > 60." This turns IC Score from a black-box number into a transparent, actionable screening system.

**Phase 2 Total Effort:** ~6-8 weeks
**Phase 2 Impact:** Saved screens create retention loops. Technical and fair value filters serve two unserved personas. IC Score sub-factor filters create genuine differentiation.

---

### Phase 3 — Differentiators (Quarter+)

#### 3.1 Natural Language Screening (2-3 weeks)
- AI-powered search bar: "Find undervalued healthcare stocks with growing dividends and insider buying"
- Parse intent → map to filters → apply → show results with explanation
- Uses Claude/GPT API to interpret, returns structured filter JSON
- Fallback to regular filters if AI parsing is uncertain
- **Moat:** Combines the IC Score sub-factors + fair value + sentiment data in a way no competitor can match because they don't have the underlying data

#### 3.2 Reddit Trends Integration (2-3 weeks)
Leverage the existing `reddit_posts_raw` pipeline data:

| Filter | Description |
|---|---|
| Reddit Mentions (7d) | Volume of Reddit discussion |
| Reddit Sentiment | AI-analyzed sentiment from r/investing, r/stocks, r/wallstreetbets |
| Reddit Trending | Stocks with rapidly increasing mention volume |

- "Reddit Trending" Quick Screen preset: stocks with spiking social mentions + positive sentiment
- Warning indicator for stocks with WSB hype but poor IC Scores (contrarian signal)
- **Moat:** No screener combines institutional-grade fundamentals with retail social sentiment data

#### 3.3 Visual Screening / Heat Map (2-3 weeks)
- Treemap visualization: size = market cap, color = selected metric (IC Score, change %, etc.)
- Scatter plot mode: plot any two metrics (e.g., P/E vs. ROE) with results as dots
- Clickable: click any dot/cell to see stock detail
- Filter the heat map using all existing screener filters

#### 3.4 Backtesting Screener Criteria (4-6 weeks)
- "What if I applied these filters 1 year ago?"
- Show hypothetical portfolio performance of stocks that matched criteria at that time
- Uses historical `stock_prices`, `fundamental_metrics_extended`, and `ic_scores` data
- Display: returns vs. S&P 500 benchmark, Sharpe ratio, max drawdown
- **Caveat:** Requires historical snapshots of metrics (current pipeline overwrites). Would need to start archiving metric snapshots.

#### 3.5 Community Shared Screens (2-3 weeks)
- Users can publish their saved screens with a name and description
- "Popular Screens" section: community-created screens ranked by usage
- Clone a shared screen to customize it
- Moderation: flag/hide screens with offensive names
- Social proof: "347 investors use this screen"

#### 3.6 Screener-Based Portfolio Construction (3-4 weeks)
- "Build Portfolio" button: take top N screener results and create a model portfolio
- Weighting options: equal weight, market cap weight, IC Score weight
- Show portfolio-level metrics: weighted P/E, expected yield, sector concentration, diversification score
- One-click add to existing portfolio tracking (leverages existing `portfolios` table)

#### 3.7 Earnings Calendar Integration (1-2 weeks)
- Filter: "Reporting earnings in next 7/14/30 days"
- Show next earnings date in results table
- Combine with other filters: "Undervalued stocks reporting earnings next week with bullish analyst sentiment"
- **Data:** Would need a new data source (Polygon.io or Alpha Vantage earnings calendar)

---

## 4. Detailed Feature Specs — Top 5 Priorities

### Priority 1: Fix IC Score and Beta (Bug Fix)

**Problem:** The screener's most differentiating metric (IC Score) displays as 0 for every stock. Beta also shows 0. This undermines trust in the entire platform. Users see "IC Score: 0" in the screener, click through to a ticker page, and see "IC Score: 78." It looks broken because it is.

**Root Cause:** In `backend/database/screener.go`, the query reads columns from `screener_data` but the Go struct mapping for `ic_score` and `beta` either falls through to defaults or the sort mapping points to wrong columns (`ic_score` → `market_cap`, `beta` → `roe`).

**User Story:** "As a user, I want to see accurate IC Scores and Beta values in screener results so I can quickly identify top-rated stocks and assess volatility."

**Proposed Fix:**
1. Verify `screener_data` materialized view contains `ic_score` (from `ic_scores.overall_score`) and `beta` (from `fundamental_metrics_extended.beta`) — confirmed present in migration 016.
2. In `backend/database/screener.go`:
   - Fix the SELECT query to properly read `ic_score` and `beta` columns from the view
   - Fix sort column mapping: `"ic_score" → "ic_score"`, `"beta" → "beta"`
   - Remove placeholder/default values
3. In `backend/handlers/screener.go`: ensure the response struct properly populates `ICScore` and `Beta` fields from query results.
4. Frontend: IC Score column should use the color-coded badge (green ≥70, yellow ≥40, red <40) that already exists in the IC Score screener component.

**Edge Cases:**
- Stocks without IC Score (data_completeness < 40%): show "N/A" not 0
- Stocks without Beta (insufficient price history): show "N/A"
- Null handling: ensure nullable float pointers propagate correctly

**Success Metrics:**
- 0% of stocks showing IC Score = 0 when they have a computed score
- IC Score becomes the #1 used sort column within 2 weeks of fix
- Reduction in user-reported "IC Score broken" support tickets to zero

---

### Priority 2: Expose 12 Existing Metrics as Filters

**Problem:** The detail page for every stock shows P/B, P/S, ROE, ROA, gross margin, net margin, debt/equity, current ratio, and more. But users can't screen by any of them. This forces users to manually check stocks one by one — defeating the purpose of a screener.

**User Story:** "As a value investor, I want to filter stocks by P/B ratio, ROE, and debt/equity so I can find financially sound companies trading at reasonable valuations without clicking through hundreds of detail pages."

**Proposed UX Behavior:**
- Filter sidebar organized into collapsible sections:
  - **Valuation:** P/E, P/B, P/S (existing + new)
  - **Profitability:** ROE, ROA, Gross Margin, Net Margin
  - **Financial Health:** Debt/Equity, Current Ratio
  - **Growth:** Revenue Growth YoY, EPS Growth YoY
  - **Dividends:** Dividend Yield, Payout Ratio, Consecutive Div Years
  - **Score:** IC Score, Beta
- Each numeric filter: min/max range inputs with step increments
- Each filter section shows count of active filters as badge
- "Clear All" button to reset all filters

**Data Requirements:**
- Alter `screener_data` materialized view to add columns: `roa`, `gross_margin`, `net_margin`, `debt_to_equity`, `current_ratio`, `eps_growth_yoy`, `payout_ratio`, `consecutive_dividend_years`
- These all exist in `fundamental_metrics_extended` which is already joined in the view
- Add corresponding filter parameters to Go handler
- Add sort mappings for each new column

**Edge Cases:**
- Negative P/E ratios (unprofitable companies): allow negative min values
- Extreme outliers (P/E > 1000): consider capping display but not filtering
- Null values: stocks missing a metric should not appear when that filter is active (WHERE col IS NOT NULL AND col >= min)
- Debt/Equity for financial companies: often very high by nature; consider flagging sector context

**Success Metrics:**
- Filter adoption: >30% of screener sessions use at least one new filter within 30 days
- Average filters per session increases from 1.2 to 3+
- Time to find target stocks decreases (measure via screener→detail page click patterns)
- Quick Screen preset click-through rate increases 20%+ (because presets now use more meaningful filter combinations)

---

### Priority 3: URL-Based Filter Persistence

**Problem:** Every page refresh, browser back-navigation, or shared link loses all filter state. A user who carefully configured 5 filters and sorted results loses everything on refresh. They can't bookmark a useful screen or share it with a friend. This is a basic usability failure that every mature screener solves.

**User Story:** "As a user, I want my screener filters to persist in the URL so I can bookmark useful screens, share them with friends, and not lose my work when I refresh the page."

**Proposed UX Behavior:**
- All active filters, sort column, sort order, and current page serialize to URL query params
- URL updates on every filter change (using `replaceState`, not `pushState`, to avoid polluting browser history)
- On page load, read URL params and restore filter state
- "Copy Link" button that copies the current URL to clipboard with a toast confirmation
- Quick Screen presets update the URL when applied

**URL Format:**
```
/screener?sectors=Technology,Healthcare&pe_max=15&div_min=3&roe_min=15&sort=ic_score&order=desc&page=1
```

**Parameter Naming Convention:**
- Sectors: `sectors=Technology,Healthcare`
- Ranges: `{metric}_min`, `{metric}_max` (e.g., `pe_min=5&pe_max=20`)
- Market cap tiers: `mcap=large,mega`
- Sort: `sort=market_cap&order=desc`
- Page: `page=2`

**Data Requirements:** None — purely frontend change.

**Implementation:**
- Use Next.js `useSearchParams()` and `useRouter()`
- Create a `useScreenerParams` custom hook that bidirectionally syncs filter state ↔ URL params
- Debounce URL updates (100ms) to avoid excessive history entries during rapid filter changes
- Validate params on load (ignore invalid values, don't crash)

**Edge Cases:**
- Invalid param values (e.g., `pe_min=abc`): silently ignore, use defaults
- Conflicting params (e.g., `pe_min=50&pe_max=10`): swap min/max
- Very long URLs (many sectors selected): URL encode properly, test with 200+ character URLs
- URL shared from a state with more filters than current version supports: ignore unknown params gracefully

**Success Metrics:**
- Page refresh retention: users who refresh maintain their screen (track via analytics)
- Shared link usage: unique URLs with filter params accessed by non-creator (>5% of screener sessions involve a shared URL within 60 days)
- Bookmark creation proxy: returning users arriving with filter params in URL

---

### Priority 4: IC Score Sub-Factor Filters

**Problem:** IC Score is a 1-100 number that tells you the overall quality of a stock, but it's opaque. A stock with IC Score 75 might be there because of stellar growth and poor value, or great value with poor momentum. Users can't filter by what _matters to them_ — they can only filter by the aggregate. Meanwhile, the database stores all 10 sub-factor scores (value_score, growth_score, profitability_score, financial_health_score, momentum_score, analyst_consensus_score, insider_activity_score, institutional_score, news_sentiment_score, technical_score).

**User Story:** "As an investor who follows insider buying signals, I want to filter stocks where insiders are actively buying (insider_activity_score > 70) AND analysts are bullish (analyst_consensus_score > 60) so I can find smart-money conviction picks."

**Proposed UX Behavior:**
- New filter section: "IC Score Factors" (collapsible, below the existing IC Score filter)
- Each sub-factor displayed as a labeled slider or min/max input (0-100 range)
- Show factor name + tooltip explaining what it measures:
  - Value Score: "How cheaply the stock trades vs. its sector peers (P/E, P/B, P/S percentiles)"
  - Growth Score: "Revenue, EPS, and free cash flow growth trajectory"
  - Profitability Score: "Margin quality — gross, operating, and net margins + ROE/ROA"
  - Financial Health Score: "Balance sheet strength — leverage, liquidity, solvency"
  - Momentum Score: "Price momentum across 1M, 3M, 6M, 12M timeframes"
  - Analyst Consensus: "Wall Street analyst ratings and price target upside"
  - Insider Activity: "Recent insider buying/selling from SEC Form 4 filings"
  - Institutional: "Institutional ownership breadth and quarterly position changes"
  - News Sentiment: "AI-analyzed sentiment from recent financial news"
  - Technical: "Technical indicator signals (RSI, MACD, moving averages)"
- Optional: "Smart Money" composite filter (combines analyst + insider + institutional)
- Results table: add optional sub-factor score columns (via column customization)

**Data Requirements:**
- Add 10 sub-factor score columns to the `screener_data` materialized view from `ic_scores` table
- The `ic_scores` table lateral join already exists in the view; just need to select additional columns
- Add 10 filter params to Go handler (e.g., `value_score_min`, `growth_score_min`, etc.)
- Add sort mappings for each sub-factor

**Edge Cases:**
- Stocks without IC Score (data_completeness < 40%): sub-factors will also be null; filter them out when any sub-factor filter is active
- Sub-factor = 0 vs. null: 0 is a valid score; null means not computed. Use null for "N/A"
- Interaction with overall IC Score filter: sub-factor filters AND overall score filter should work independently (a stock with IC Score 60 might have insider_activity_score 90)

**Success Metrics:**
- Adoption: >15% of screener sessions use at least one sub-factor filter within 60 days
- Engagement: sessions using sub-factor filters have 40%+ longer duration than average
- Unique value: track what % of sub-factor filter users don't use equivalent features on competitors (survey)
- Detail page click-through: users who filter by sub-factors have higher click-through to detail pages (they're more engaged)

---

### Priority 5: CSV Export

**Problem:** Active investors and analysts pull screener data into spreadsheets for further analysis, portfolio modeling, or record-keeping. Every competing screener offers export. Without it, users must manually copy-paste, which is unacceptable for anyone screening seriously.

**User Story:** "As an analyst, I want to export my screener results to CSV so I can combine the data with my own models in Excel/Google Sheets."

**Proposed UX Behavior:**
- "Export CSV" button in the screener toolbar (next to results count)
- Exports all rows matching current filters (not just current page)
- Includes all visible columns plus: industry, beta, IC Score, and any sub-factor scores if IC Score sub-factor columns are enabled
- Filename format: `investorcenter-screener-{YYYY-MM-DD}.csv`
- Maximum 10,000 rows per export
- Show toast: "Exported 2,847 stocks to CSV"

**Implementation:**
- Client-side CSV generation using the already-loaded data (the screener fetches up to 20,000 results)
- Use a lightweight CSV library or manual string building
- Trigger browser download via `Blob` + `URL.createObjectURL`
- No new API endpoint needed

**Data Requirements:** None — uses already-fetched client-side data.

**Edge Cases:**
- Large exports (>10K rows): warn user and cap at 10K, sorted by their current sort
- Special characters in stock names (commas, quotes): proper CSV escaping
- Null values: export as empty string, not "null" or "0"
- Unicode company names: ensure UTF-8 BOM for Excel compatibility

**Success Metrics:**
- Export adoption: >10% of screener sessions include an export within 30 days
- Retention: users who export have higher 30-day return rate than those who don't
- Reduced support requests for "how do I get this data out"

---

## 5. Technical Considerations

### What Data Is Already Available (Just Needs Wiring)

The gap between "what exists" and "what the screener shows" is the biggest finding of this analysis. Here's the breakdown:

**Available in `screener_data` view TODAY, not exposed in API results:**
- `pb_ratio` (P/B) — in view, column exists, returned in API response but no frontend filter
- `ps_ratio` (P/S) — same as above
- `roe` — same
- `beta` — in view but Go code returns 0.0 (bug)
- `ic_score` — in view but Go code returns 0.0 (bug)
- `industry` — in view, not exposed as filter

**Available in `fundamental_metrics_extended`, requires adding to materialized view:**
- roa, roic, gross_margin, operating_margin, net_margin, ebitda_margin
- eps_growth_yoy, revenue_growth_3y_cagr, revenue_growth_5y_cagr, fcf_growth_yoy
- debt_to_equity, current_ratio, quick_ratio, interest_coverage
- dividend_yield, payout_ratio, dividend_growth_rate, consecutive_dividend_years
- dcf_fair_value, dcf_upside_percent, graham_number, epv_fair_value
- pe_sector_percentile, pb_sector_percentile, roe_sector_percentile
- enterprise_value, ev_to_revenue, ev_to_ebitda, ev_to_fcf
- wacc, data_quality_score

**Available in `ic_scores`, requires adding to materialized view:**
- value_score, growth_score, profitability_score, financial_health_score
- momentum_score, analyst_consensus_score, insider_activity_score
- institutional_score, news_sentiment_score, technical_score
- rating, sector_percentile, confidence_level, lifecycle_stage

**Available in `technical_indicators`, requires new join or pre-computation:**
- RSI (14), SMA 50, SMA 200, MACD, Bollinger Bands

**Available in `risk_metrics`, requires new join:**
- alpha, sharpe_ratio, sortino_ratio, max_drawdown, var_5

### What Would Require New Data Pipelines

| Feature | New Pipeline Needed | Complexity |
|---|---|---|
| Earnings calendar filter | Yes — new data source (Polygon.io earnings) | Medium |
| Reddit sentiment score filter | Partially — pipeline exists but scoring needs formalization | Low-Medium |
| Historical metric snapshots (for backtesting) | Yes — archive pipeline for daily metric snapshots | High |
| 52-week high/low | Derived from `stock_prices` — add to materialized view refresh | Low |
| YTD return | Derived from `stock_prices` — add to materialized view refresh | Low |
| Average volume (20/50-day) | Derived from `stock_prices` — add to materialized view refresh | Low |

### Performance Considerations

**Current Architecture:** The materialized view is the right pattern. At 5,000+ stocks, live joins across 4+ tables would be slow. The view pre-computes everything and refreshes at 23:45 UTC.

**Concerns with expansion:**
1. **View refresh time:** Currently the view joins `tickers`, `stock_prices`, `valuation_ratios`, `fundamental_metrics_extended`, and `ic_scores` via lateral joins. Adding `technical_indicators` and `risk_metrics` as additional lateral joins may push refresh time beyond acceptable limits (currently ~30-60 seconds for 5K stocks, could grow to 2-5 minutes with 7 lateral joins).

2. **Mitigation strategy:**
   - Option A: Expand the single materialized view with all columns (simplest, test performance)
   - Option B: Create a second materialized view (`screener_data_extended`) with technical + risk data, joined at query time on `symbol` (indexed)
   - Option C: Pre-compute a denormalized `screener_snapshot` table that each pipeline writes to, avoiding the need for a materialized view entirely
   - **Recommendation:** Start with Option A. If refresh exceeds 2 minutes, move to Option B.

3. **Query performance:** Adding more WHERE clauses to the screener query is negligible with proper indexes. Each new filterable column should have an index in the materialized view. Current indexes cover market_cap, sector, pe_ratio, ic_score, dividend_yield. Add indexes for: roe, beta, debt_to_equity, gross_margin, eps_growth_yoy.

4. **Client-side vs. server-side filtering:** The current basic screener loads all 20K results and filters client-side. This works for 6 filters but will degrade with 30+ filters on 20K rows. **Recommendation:** Migrate to server-side filtering for all filters (the IC Score screener already does this correctly). Remove the client-side filtering pattern.

5. **Pagination:** With server-side filtering, implement proper OFFSET/LIMIT pagination. Consider cursor-based pagination if users regularly page deep (>page 50), though this is unlikely for a screener.

### Architecture Recommendations

1. **Consolidate the two screeners.** The codebase has two parallel screener implementations (basic at `/screener` and IC Score at `/ic-score`). These should be merged into a single, unified screener at `/screener` that includes IC Score as a first-class filter/column. The IC Score screener has better architecture (server-side filtering) and should be the base.

2. **Move to server-side filtering entirely.** The basic screener's "fetch 20K rows and filter client-side" pattern won't scale. Migrate to the IC Score screener's pattern of sending filter params to the API.

3. **Single materialized view as source of truth.** Expand `screener_data` to include all needed columns rather than creating multiple views. This simplifies the query layer and avoids join overhead at query time.

4. **Add a filter registry pattern.** Instead of hardcoding each filter in the Go handler, create a filter registry that maps param names to column names and types. This makes adding new filters a config change rather than a code change:
   ```go
   var filterRegistry = map[string]FilterDef{
       "pe":              {Column: "pe_ratio", Type: RangeFilter},
       "pb":              {Column: "pb_ratio", Type: RangeFilter},
       "roe":             {Column: "roe", Type: RangeFilter},
       "sectors":         {Column: "sector", Type: MultiSelectFilter},
       // ... etc
   }
   ```

5. **Frontend filter configuration.** Mirror the registry on the frontend — a JSON config that defines available filters, their types, labels, sections, and min/max bounds. This makes adding a filter a config change on both sides.

---

## 6. Metrics & Success Framework

### Engagement Metrics

| Metric | Current Baseline (Est.) | Phase 1 Target | Phase 2 Target |
|---|---|---|---|
| Avg. filters used per session | ~1.2 | 3+ | 5+ |
| Avg. session duration on screener | ~90 seconds | 3+ minutes | 5+ minutes |
| Screener sessions / user / week | ~1.5 | 3+ | 5+ |
| 7-day return rate (screener users) | ~20% | 35% | 50% |
| Preset usage rate | ~40% of sessions | 50%+ | 30% (as custom filters mature) |

### Conversion Metrics

| Metric | Current (Est.) | Target |
|---|---|---|
| Screener → detail page CTR | ~15% | 30%+ |
| Screener → watchlist add | ~0% (no integration) | 10%+ |
| Screener → alert creation | ~0% (no integration) | 5%+ |
| Screener → CSV export | ~0% (doesn't exist) | 10%+ |
| Screener → saved screen | ~0% (doesn't exist) | 15%+ |
| Shared URL screener visits | ~0% (no persistence) | 5% of sessions |

### Product-Market Fit Signals

| Signal | How to Measure | Threshold |
|---|---|---|
| Screener is the entry point | % of sessions starting at /screener | >25% of all site sessions |
| Power user adoption | Users with 5+ saved screens | >10% of registered users |
| Feature stickiness | % of screener users using 3+ filter categories | >40% |
| Export as value indicator | Users who export and return within 7 days | >60% |
| IC Score differentiation | % of sessions using IC Score or sub-factor filters | >30% |
| Referral signal | Shared screener URLs that drive new signups | Track absolute count |

### Instrumentation Plan

Track these events (via analytics):
- `screener_filter_applied` — {filter_name, value, session_id}
- `screener_sort_changed` — {column, direction}
- `screener_preset_used` — {preset_name}
- `screener_stock_clicked` — {symbol, position_in_list, active_filters}
- `screener_export_csv` — {row_count, active_filters}
- `screener_save_screen` — {screen_name, filter_count}
- `screener_share_url` — {filter_count}
- `screener_add_to_watchlist` — {symbol, source: "screener"}
- `screener_create_alert` — {filter_criteria}
- `screener_column_customized` — {columns_added, columns_removed}

---

## Appendix: Implementation Sequence

Recommended order within Phase 1 (each item can be a PR):

1. **Fix IC Score + Beta bug** (highest impact, lowest effort — ship same day)
2. **URL filter persistence** (foundational for everything else)
3. **Expose 12 new filters** (materialized view migration + Go handler + frontend)
4. **Column customization** (makes new columns useful)
5. **CSV export** (quick win, high perceived value)
6. **Industry sub-filtering** (leverages existing data)
7. **Updated presets** (leverages new filters)

After Phase 1, reassess priorities based on analytics data from the new instrumentation.
