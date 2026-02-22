# Product Requirements Document: Social Sentiment & Reddit Trends Redesign

**Product:** InvestorCenter.AI  
**Version:** 1.0  
**Date:** February 2026  
**Status:** Draft for Review  
**Author:** Product Management

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [User Research & Insights](#2-user-research--insights)
3. [User Personas](#3-user-personas)
4. [Current vs. Proposed Experience](#4-current-vs-proposed-experience)
5. [Feature Specifications](#5-feature-specifications)
6. [Information Architecture Changes](#6-information-architecture-changes)
7. [Success Metrics](#7-success-metrics)
8. [Data & Technical Considerations](#8-data--technical-considerations)
9. [Monetization & Tiering](#9-monetization--tiering)
10. [Out of Scope](#10-out-of-scope)
11. [Risks & Mitigations](#11-risks--mitigations)
12. [Phased Rollout Plan](#12-phased-rollout-plan)
13. [Design Decision Log](#13-design-decision-log)
14. [Appendix: Competitor Feature Matrix](#14-appendix-competitor-feature-matrix)
15. [OpenClaw Data Pipeline Strategy](#15-openclaw-data-pipeline-strategy)
16. [Product Evolution: Multi-Source Social Intelligence](#16-product-evolution-multi-source-social-intelligence)

---

## 1. Executive Summary

### 1.1 Problem Statement

InvestorCenter currently offers two socially-driven features that are disconnected from one another: a **Reddit Trends page** (`/reddit`) that ranks stocks by mention volume, and a per-ticker **Social Sentiment detail page** (`/sentiment?ticker=XXX`) that provides sentiment breakdown and post feeds. Users cannot move fluidly between discovery (trending) and analysis (sentiment) because the experiences live in separate navigation silos, surface different data, and share no common visual language.

Additionally, the trending table lacks the financial context (price, percentage change) that would make it immediately actionable, and known data-quality bugs — floating-point rank-change values, inconsistent iconography, and stale timestamps — erode user trust.

### 1.2 Opportunity

Social sentiment is one of the fastest-growing signal categories among retail investors. Platforms such as Stocktwits, Unusual Whales, and Sentiment Investor have demonstrated that aggregating crowd-sourced opinion into a structured, quant-readable layer creates significant user engagement and premium conversion. InvestorCenter is well-positioned to differentiate here: unlike pure social platforms, it already provides institutional-grade financial data (fundamentals, screener, charting). Embedding a high-quality sentiment layer directly alongside that data turns InvestorCenter from a research tool into a **decision-support system**.

### 1.3 Strategic Rationale

- **Unify** the Reddit Trends page and Social Sentiment detail page into a single, cohesive experience that serves discovery, analysis, and action.
- **Surface sentiment signals inline** on the trending table so users can assess conviction without extra clicks.
- **Fix all known data-quality bugs** in the initial release to restore user trust.
- **Lay the foundation** for multi-source sentiment (Twitter/X, StockTwits, financial news) in a later phase.
- **Create a premium-tier gateway:** basic sentiment is free; deep history, alerts, API access, and comparison tools become paid features.

---

## 2. User Research & Insights

### 2.1 Key Pain Points (Inferred from Product Audit)

- **Data disconnection:** Trending table shows volume metrics but no sentiment polarity. Users must navigate to a separate page per ticker to learn whether discussion is bullish or bearish.
- **Missing financial context:** The trending table omits price and percentage change, forcing users to cross-reference another tool to evaluate whether a trending stock is worth investigating.
- **Data quality issues:** Floating-point rank changes (e.g., "↓11.14") and inconsistent score icons undermine credibility.
- **Stale data:** Timestamps showing "Last updated: 7h ago" make the feature feel unreliable for time-sensitive trading decisions.
- **Limited filtering:** Users cannot filter by sector, sentiment direction, or subreddit source, reducing the table's utility for targeted screening.

### 2.2 User Mental Models

Retail investors use Reddit/social data at three distinct points in their workflow:

1. **Discovery** — "What's the market talking about right now?" Users scan for unusual activity or emerging narratives.
2. **Validation** — "Is the crowd bullish or bearish on a stock I'm already watching?" Sentiment acts as a confirmation or contrarian signal.
3. **Timing** — "Is buzz accelerating? Should I enter or exit before the crowd moves?" Velocity of mentions and sentiment shift informs trade timing.

### 2.3 Jobs to Be Done (JTBD)

| Job Statement | Current Satisfaction | Priority |
|---|---|---|
| When I open the trending page, I want to see which stocks have unusual social momentum so I can investigate further. | Partially met (volume only, no sentiment or price) | Critical |
| When I find a trending ticker, I want to understand crowd conviction without leaving the page so I can quickly triage opportunities. | Not met (requires navigation to /sentiment) | Critical |
| When I'm monitoring a position, I want to be alerted when sentiment shifts materially so I can react in a timely manner. | Not met (no alerts exist) | High |
| When I'm doing deep research, I want to overlay historical sentiment on the price chart so I can evaluate the signal's predictive value. | Partially met (chart exists but is disconnected from price) | Medium |

### 2.4 Synthetic User Scenarios

> *"I check the Reddit Trends page every morning before the bell. But when I see a ticker trending, I have no idea if people are excited about it or panicking. I end up opening five tabs just to figure out if the buzz is bullish."* — Maria, casual investor

> *"The sentiment detail page is actually useful, but I only found it by accident. I had no idea it existed because nothing on the trending table links to it or even hints that deeper analysis is available."* — Jake, swing trader

> *"I'd love to backtest whether Reddit sentiment actually predicts price movement. If InvestorCenter showed me a historical accuracy metric, I'd probably pay for a premium subscription."* — Priya, quantitative researcher

---

## 3. User Personas

### 3.1 Maria — The Casual Discoverer

| Attribute | Detail |
|---|---|
| Profile | 28, marketing manager, invests in index funds and a handful of individual stocks |
| Behavior | Checks Reddit Trends 2–3 times per week; reads r/wallstreetbets for entertainment value |
| Goal | Spot stocks that are generating unusual buzz so she can do lightweight research before deciding whether to buy |
| Frustration | No sentiment indicator on the trending page; has to open multiple tabs to understand whether buzz is positive or negative |
| Success metric | Finds 1–2 new tickers per week worth researching, in under 3 minutes |

### 3.2 Jake — The Active Swing Trader

| Attribute | Detail |
|---|---|
| Profile | 35, full-time trader, holds positions for 2–10 days based on technical and sentiment signals |
| Behavior | Monitors sentiment as a timing and confirmation layer alongside chart patterns and options flow |
| Goal | Identify when crowd sentiment is shifting (bullish-to-bearish or vice versa) to time entries and exits |
| Frustration | Data staleness (7h delay) makes the feature unreliable for intraday decisions; no alerts for sentiment shifts |
| Success metric | Receives a timely alert when a watched ticker's sentiment flips, allowing him to act within 30 minutes |

### 3.3 Priya — The Quant Power User

| Attribute | Detail |
|---|---|
| Profile | 31, data scientist turned independent researcher, uses Python notebooks to backtest strategies |
| Behavior | Wants raw sentiment scores via API; overlays historical sentiment on price charts to test predictive value |
| Goal | Quantify whether Reddit sentiment is a statistically significant leading indicator for short-term price movement |
| Frustration | No API access to sentiment data; no historical accuracy metric; cannot compare tickers side-by-side |
| Success metric | Downloads 90 days of sentiment data via API and integrates it into her backtesting framework within one session |

---

## 4. Current vs. Proposed Experience

### 4.1 Current Journey Map

**Maria (Discoverer):** Opens /reddit → Sees ranked table with mentions and score → Sees a ticker she doesn't recognize → Has no sentiment or price context → Opens a new browser tab to look up price → May or may not find /sentiment page → Abandons the workflow if effort is too high.

**Jake (Trader):** Opens /reddit → Scans for tickers he holds → Notices a rank change but the number is a floating-point mess → Clicks through to /sentiment → Finds sentiment data is 7 hours stale → Loses trust and falls back to Stocktwits for real-time reads.

**Priya (Quant):** Opens /sentiment?ticker=NVDA → Reviews 30-day sentiment history chart → Wants to overlay on price but cannot → Wants to export data but no API exists → Manually screenshots the chart and considers building her own scraper.

### 4.2 Proposed Journey Map

**Maria (Discoverer):** Opens /reddit → Sees enriched table with price, % change, sentiment badge, and sparkline → Filters by sector (Technology) and sentiment (Bullish) → Expands a row inline to preview top 3 posts → Clicks "Add to Watchlist" → Completes her morning scan in under 2 minutes.

**Jake (Trader):** Opens /reddit → Sees "Last updated: 12 min ago" indicator → Sorts by Sentiment Score descending → Notices TSLA sentiment shifted from Neutral to Bearish → Clicks through to enhanced /sentiment page with sentiment overlay on price chart → Sets an alert: "Notify me if TSLA sentiment turns Bullish."

**Priya (Quant):** Opens /sentiment?ticker=NVDA → Views sentiment overlaid directly on the price chart → Toggles to 90-day view → Sees "Historical Accuracy: When NVDA sentiment was Bullish, price was up 2.3% on average 3 days later" → Compares NVDA vs. AMD sentiment side-by-side → Accesses API to pull raw data into her notebook.

### 4.3 Before / After Key Screen Comparison

| Element | Before | After |
|---|---|---|
| Trending Table Columns | Rank, Ticker, Prev Rank, Rank Change, Mentions, Upvotes, Score | Rank, Ticker, Company Name, Price, % Change, Mentions, Upvotes, Score, Sentiment Badge, Rank Change (integer), 7D Sparkline |
| Rank Change Display | Raw float: "↓11.142857..." | Rounded integer: "↑11" with consistent arrow icon |
| Score Icon | Chart icon for top 3, chat bubble for rest | Unified numeric score with consistent color-coded badge |
| Sentiment Signal | Not visible on trending table | Inline Bullish/Neutral/Bearish badge with color coding (green/gray/red) |
| Data Freshness | "Last updated: 7h ago", no refresh | "Updated 12 min ago" with manual refresh button; auto-refresh every 15 min |
| Filtering | Time range only | Sector, Sentiment Direction, Time Range, Subreddit Source |
| View Modes | Table only | Table View + Heatmap View toggle |
| Row Interaction | Click navigates to ticker page | Expand inline for top 3 posts; click ticker for full detail; "Add to Watchlist" action |

---

## 5. Feature Specifications

### 5a. Redesigned Reddit Trends Page (`/reddit`)

#### Table Columns

The redesigned table presents the following columns in a single, scannable row:

- **Rank** — numerical position, 1–50/100
- **Ticker** — linked to `/ticker/XXX`
- **Company Name** — full name for discoverability
- **Current Price** — real-time or 15-min delayed
- **% Change** — daily, color-coded green/red
- **Mentions** — count over selected time range
- **Upvotes** — aggregate across tracked subreddits
- **Score** — composite metric combining mentions, upvotes, and comment velocity
- **Sentiment Badge** — Bullish / Neutral / Bearish with green/gray/red coloring
- **Rank Change** — rounded integer, displayed as ↑N or ↓N with directional arrow
- **7-Day Sparkline** — miniature line chart showing mention trend

#### Filter Bar

- **Sector:** All, Technology, Healthcare, Finance, Energy, Consumer, Industrials, etc.
- **Sentiment Direction:** All / Bullish / Bearish / Neutral
- **Time Range:** Today / 7 Days / 14 Days / 30 Days
- **Subreddit Source:** All / r/wallstreetbets / r/stocks / r/investing / r/options

#### Sort Options

Users may sort by: Score (default), Mentions, Upvotes, Sentiment Score, Rank Change (absolute magnitude), Price % Change.

#### View Modes

A toggle allows switching between **Table View** and **Heatmap View**. In heatmap mode, tiles are colored by sentiment direction (green = bullish, red = bearish, gray = neutral) and sized proportionally to mention count. This mirrors the existing watchlist heatmap pattern already familiar to InvestorCenter users.

#### Data Freshness Indicator

A persistent banner at the top of the table displays "Updated X minutes ago" with a manual refresh button. Auto-refresh fires every 15 minutes. If data is older than 30 minutes, the indicator turns amber with a warning.

#### Inline Actions

- **"Add to Watchlist"** icon on each row (star icon, toggles on/off).
- **Row expand** (chevron): reveals a quick-view panel showing the top 3 Reddit posts for that ticker (title, subreddit, upvotes, sentiment label, timestamp). Users can scan posts without navigating away.

#### Pagination

Default view shows 50 results. Users may toggle to 100. Pagination controls at the bottom of the table. Infinite scroll is not recommended due to the filtering/sorting UX — paginated tables are more predictable for data-heavy interfaces.

### 5b. Enhanced Social Sentiment Detail Page (`/sentiment?ticker=XXX`)

#### Price Chart Integration

The sentiment score line is overlaid directly on the price chart as a secondary y-axis. Users can toggle the overlay on/off. Time range options: 7D, 30D, 90D. This eliminates the need to mentally correlate two separate charts.

#### Sentiment Velocity Indicator

A directional arrow and numeric delta show whether sentiment is becoming more bullish or bearish over the past 24 hours. Example: "Sentiment Velocity: +12% more bullish in last 24h."

#### Comparison Mode

Users can add a second ticker to compare sentiment side-by-side on the same chart. This supports relative-value analysis ("Is the crowd more bullish on NVDA or AMD right now?"). Available in premium tier.

#### Post Feed Improvements

- Filterable by sentiment: Bullish / Bearish / Neutral / All
- Sortable by: Most Upvoted (default), Most Recent, Most Comments
- High-scoring posts (>100 upvotes) show a body preview (first 200 characters) in addition to the title
- "View full thread" expansion loads the post body and top comments inline without navigating to Reddit

#### Subreddit Distribution

A horizontal bar chart shows mention distribution across subreddits (e.g., 45% r/wallstreetbets, 30% r/options, 15% r/stocks, 10% r/investing). This helps users assess where the conversation is happening and weight the signal accordingly.

#### Historical Sentiment Accuracy

A backtested metric displayed as a callout: "When [TICKER] sentiment was Bullish, price was up X.X% on average 3 days later (based on 90-day lookback)." This feature ships in Phase 4 and requires a disclaimer (see Section 11).

### 5c. Social Sentiment Widget on Ticker Pages (`/ticker/XXX`)

The existing compact widget is expanded into a more prominent module within the ticker detail page:

- Current sentiment score with Bullish/Neutral/Bearish label
- 7-day trend direction arrow (up/down/flat)
- Mention count (24h) and velocity indicator
- Top 2–3 post titles as a preview feed with subreddit tags
- "See full sentiment analysis" CTA button linking to `/sentiment?ticker=XXX`

This widget serves as the primary bridge between the ticker research flow and the sentiment deep-dive, addressing the current discoverability gap.

### 5d. Sentiment-Based Alerts

Users can configure the following alert types:

| Alert Type | Configuration | Example |
|---|---|---|
| Mention Volume Spike | [TICKER] mentions exceed [N] in [24h/7d] | Alert me when NVDA mentions exceed 200 in 24h |
| Sentiment Shift | [TICKER] sentiment shifts to [Bullish/Bearish] | Alert me when SPY sentiment turns Bearish |
| Trending Entry | [TICKER] enters the Top [N] trending | Alert me when AAPL enters the Top 10 |
| Sentiment Velocity | [TICKER] sentiment changes by [N]% in [24h] | Alert me when TSLA sentiment drops 20% in 24h |

Delivery channels: in-app notification badge and email. Alerts integrate with the existing InvestorCenter Alerts system and are managed from the centralized Alerts dashboard. Free-tier users may create up to 3 sentiment alerts; premium users get unlimited.

### 5e. Bug Fixes (Must-Ship with Phase 1)

| Bug ID | Description | Fix Specification | Acceptance Criteria |
|---|---|---|---|
| BUG-001 | Rank change displays raw floating-point values | Apply `Math.round()` to rank change delta; format display as ↑N or ↓N | All rank change values display as integers with directional arrows; no decimal points visible |
| BUG-002 | Score icon inconsistency (chart vs. chat bubble) | Replace conditional icon logic with a single unified score display: numeric value inside a consistently-styled badge | All rows use the same icon/badge treatment regardless of rank position |
| BUG-003 | No current price or % change on trending table | Add Price and % Change columns pulling from the existing market data API | Every row shows current price and daily % change with appropriate red/green coloring |
| BUG-004 | Data staleness (7h+ without refresh) | Add "Last updated X min ago" indicator; implement 15-min auto-refresh; add manual refresh button | Indicator visible at all times; auto-refresh fires every 15 min; data is never more than 30 min stale during market hours |

---

## 6. Information Architecture Changes

### 6.1 Navigation Recommendation

Reddit Trends should remain a top-level navigation item, but be relabeled as **"Social Trends"** to accommodate future multi-source expansion (Twitter/X, StockTwits). This positions the feature as a platform-agnostic social intelligence layer rather than a Reddit-specific page. The relabeling also creates a natural umbrella for the sentiment detail page, alerts, and heatmap view.

### 6.2 Relationship to Screener and Alerts

Social sentiment becomes a filterable dimension within the Screener: users can add "Sentiment = Bullish" as a screening criterion alongside fundamental and technical filters. Sentiment-based alerts are integrated into the existing Alerts management page as a new alert category alongside price alerts and volume alerts.

### 6.3 Proposed Information Architecture

```
Top Nav: Dashboard | Screener | Social Trends | Watchlist | Alerts | Portfolio

├── Social Trends (/reddit)
│   ├── Row expand → inline post preview
│   ├── Ticker click → /ticker/XXX (with enhanced sentiment widget)
│   └── "Full Sentiment" CTA → /sentiment?ticker=XXX
│
├── Ticker Page (/ticker/XXX)
│   └── Expanded sentiment widget module
│
├── Sentiment Detail (/sentiment?ticker=XXX)
│   └── Deep-dive: price+sentiment overlay, post feed, comparison mode
│
├── Alerts (/alerts)
│   └── Sentiment alert category (alongside price/volume alerts)
│
└── Screener (/screener)
    └── Sentiment added as a filterable dimension
```

---

## 7. Success Metrics

| Category | KPI | Target | Measurement Method |
|---|---|---|---|
| Engagement | Avg. time on Social Trends page | +40% vs. current Reddit Trends | Analytics (Mixpanel / PostHog) |
| Engagement | Pages per session from Social Trends entry | +25% (users navigating to sentiment detail and ticker pages) | Funnel analytics |
| Engagement | 7-day return visit rate for Social Trends | >35% of monthly active users | Cohort retention analysis |
| Feature Adoption | % of Social Trends visitors using filter bar | >30% within 60 days of launch | Feature event tracking |
| Feature Adoption | Sentiment alert creation rate | >500 alerts created in first 30 days | Alert system logs |
| Feature Adoption | Heatmap view toggle usage | >15% of Social Trends sessions | Toggle event tracking |
| Data Quality | Data freshness SLA | <30 min during market hours, 100% of the time | Backend monitoring |
| Data Quality | Floating-point display bugs | 0 recurrence | Automated regression test |
| Business | Premium conversion from Social Trends funnel | >2% of free users upgrade within 90 days | Subscription analytics |
| Business | API access adoption (premium) | >50 API keys provisioned within 90 days of API launch | API key issuance logs |

---

## 8. Data & Technical Considerations

### 8.1 Expanded Data Sources

Phase 1–2 will continue to source from the existing four subreddits (r/wallstreetbets, r/stocks, r/investing, r/options) using the **OpenClaw reddit-scraper skill** as the primary data acquisition layer (see Section 15 for full pipeline architecture). Phase 3–4 should evaluate adding: r/SecurityAnalysis, r/ValueInvesting, r/ETFs, r/Forex, r/SPACs, and r/pennystocks via the same OpenClaw pipeline. Multi-platform expansion (Twitter/X, StockTwits, financial news) is targeted for Phase 4, leveraging OpenClaw's web_fetch and Firecrawl capabilities to scrape public feeds without requiring per-platform API keys.

### 8.2 Refresh Frequency Targets

| Data Type | Current | Target (Phase 1) | Target (Phase 3+) |
|---|---|---|---|
| Trending table (rank, mentions, score) | Unknown (>7h stale) | Every 15 minutes | Every 10 minutes |
| Sentiment score per ticker | Unknown | Every 30 minutes | Every 15 minutes |
| Post feed | Unknown | Every 30 minutes | Near-real-time (5 min) |
| Price and % change | Not shown | 15-min delayed (free) / real-time (premium) | Real-time for all tiers |

### 8.3 Historical Data Retention

Sentiment scores, mention counts, and post metadata should be retained for a minimum of 12 months to support the backtested sentiment accuracy feature (Phase 4). Raw post content may be stored in a compressed cold-storage tier after 90 days. The historical sentiment accuracy metric requires at least 90 days of continuous data collection before it can be displayed with statistical confidence.

### 8.4 API Exposure

Social sentiment data should be exposed through the InvestorCenter public API as a premium-tier endpoint:

- `GET /api/v1/sentiment/{ticker}` — current + historical scores
- `GET /api/v1/trending` — ranked list with all metadata
- `GET /api/v1/sentiment/{ticker}/posts` — paginated post feed

Rate limits and authentication will follow existing API patterns.

---

## 9. Monetization & Tiering

| Feature | Free Tier | Premium Tier |
|---|---|---|
| Trending table (enriched) | ✓ Full access | ✓ Full access |
| Sentiment badge on trending table | ✓ | ✓ |
| Filter bar (sector, sentiment, time range) | ✓ | ✓ |
| Heatmap view | ✓ | ✓ |
| Sentiment detail page: 7D history | ✓ | ✓ |
| Sentiment detail page: 30D / 90D history | ✗ Locked | ✓ |
| Sentiment overlay on price chart | ✗ Locked | ✓ |
| Comparison mode (two tickers) | ✗ Locked | ✓ |
| Full post feed (sortable, filterable, body preview) | Limited to top 10 posts | ✓ Unlimited |
| Sentiment alerts | Up to 3 alerts | ✓ Unlimited |
| Historical sentiment accuracy metric | ✗ Locked | ✓ |
| API access | ✗ Not available | ✓ Rate-limited access |

This tiering strategy ensures that the free experience is genuinely valuable for discovery and casual use (Maria persona), while incentivizing upgrade for power users who need depth, history, and programmatic access (Priya persona). The sentiment alert limit creates a natural conversion trigger for active traders (Jake persona).

---

## 10. Out of Scope

- **Building a native social posting or commenting platform.** InvestorCenter aggregates and analyzes social data; it does not create its own social network.
- **Real-time streaming via WebSocket.** The target architecture uses near-real-time polling (10–15 min intervals) rather than persistent socket connections, which would significantly increase infrastructure cost.
- **Non-English Reddit content.** NLP sentiment analysis will target English-language posts only in all phases.
- **Non-financial subreddits.** Only finance and investing subreddits will be tracked.
- **Mobile-native app features.** This PRD covers the web application. Mobile responsiveness is in scope; native mobile app enhancements are not.
- **Custom sentiment model training.** InvestorCenter will use an existing NLP sentiment model (e.g., FinBERT or a commercial API). Building a proprietary model is deferred.

---

## 11. Risks & Mitigations

| Risk | Severity | Likelihood | Mitigation Strategy |
|---|---|---|---|
| Reddit API rate limits or policy changes reduce data availability | High | Medium | Primary mitigation: **OpenClaw's reddit-scraper skill uses public JSON endpoints, bypassing the official API entirely** (see Section 15). Secondary: implement caching layer; diversify to multiple data sources; maintain a registered Reddit API key as fallback; build additional fallback to Firecrawl for bot-circumvention. |
| Sentiment model inaccuracy leads to misleading signals | High | Medium | Display all sentiment data with a mandatory disclaimer: "Sentiment data is algorithmically derived and should not be used as the sole basis for investment decisions." Include accuracy confidence intervals where available. |
| Pump-and-dump schemes inflate mention volume for manipulated tickers | Medium | High | Implement anomaly detection: flag tickers with sudden, extreme mention spikes from low-karma or new accounts. Display a "⚠ Unusual Activity" badge when bot-like patterns are detected. Weight scores by account age and karma. |
| Data licensing and copyright issues with Reddit post content | Medium | Low | Store post metadata (title, score, timestamp, link) rather than full post bodies. Link to Reddit for full content. Review Reddit's Terms of Service and API licensing agreement on a quarterly basis. |
| Stale data undermines user trust despite technical improvements | Medium | Low | Implement real-time monitoring of data pipeline health. Display pipeline status on the page. Degrade gracefully: if data is >1h stale, show a prominent warning and disable alerts. |

---

## 12. Phased Rollout Plan

### Phase 1: Foundation (Weeks 1–4)

**Theme:** Fix trust, add context, unify the experience.

- Fix all four must-ship bugs (BUG-001 through BUG-004)
- Add Company Name, Price, % Change, Sentiment Badge, and Sparkline columns to the trending table
- Implement the filter bar (Sector, Sentiment Direction, Time Range, Subreddit Source)
- Add sort-by options (Score, Mentions, Upvotes, Sentiment Score, Rank Change)
- Implement 15-minute auto-refresh with visible freshness indicator
- Add inline row expand for top 3 posts preview
- Add "Add to Watchlist" inline action
- Relabel navigation from "Reddit Trends" to "Social Trends"

**Exit criteria:** All bugs resolved, enriched table live, filter/sort functional, data freshness < 30 min.

### Phase 2: Depth (Weeks 5–8)

**Theme:** Make sentiment analysis a deep, compelling experience.

- Redesign the sentiment detail page (`/sentiment?ticker=XXX`)
- Implement sentiment overlay on the price chart (toggle on/off)
- Add sentiment velocity indicator (24h directional change)
- Improve post feed: filtering by sentiment, sorting by upvotes/recency, body preview for high-scoring posts
- Add subreddit distribution bar chart
- Expand the sentiment widget on the ticker detail page (`/ticker/XXX`)

**Exit criteria:** Sentiment detail page redesigned, sentiment+price chart overlay live, post feed enhanced.

### Phase 3: Engagement (Weeks 9–12)

**Theme:** Drive habitual use through alerts, visualization, and comparison.

- Launch sentiment-based alerts (mention volume, sentiment shift, trending entry, velocity)
- Implement heatmap view toggle on the Social Trends page
- Launch comparison mode on the sentiment detail page
- Add sentiment as a filterable dimension in the Screener

**Exit criteria:** Alerts system live, heatmap and comparison features launched, screener integration complete.

### Phase 4: Scale & Intelligence (Weeks 13–20)

**Theme:** Expand data sources, add predictive intelligence, and monetize.

- Expand subreddit coverage via OpenClaw cron jobs (r/SecurityAnalysis, r/ValueInvesting, r/ETFs, r/Forex, etc.)
- Integrate Twitter/X and StockTwits as additional sentiment sources using OpenClaw's web_fetch + Firecrawl pipeline (see Section 15)
- Launch composite multi-source sentiment score with source attribution bar
- Launch historical sentiment accuracy metric (backtested signal quality)
- Launch public API for sentiment data (premium tier)
- Implement anomaly detection for pump-and-dump / bot activity

**Exit criteria:** Multi-source sentiment live via OpenClaw pipeline, API launched, accuracy metric displayed, anomaly detection active. See Section 16 for the longer-term product evolution roadmap beyond Phase 4.

---

## 13. Design Decision Log

This section documents the key design decisions made during the creation of this PRD, along with the rationale for each.

| Decision | Options Considered | Chosen Approach | Rationale |
|---|---|---|---|
| Pagination vs. Infinite Scroll | Infinite scroll for modern feel; Pagination for data-heavy UX | Pagination (50/100 toggle) | Financial data tables require predictable row positions for comparison workflows. Infinite scroll makes re-finding a row difficult after filtering or sorting. |
| Heatmap as separate page vs. toggle | Dedicated /reddit/heatmap URL; Toggle on existing page | Toggle on the same page | Reduces navigation friction. Users can switch views without losing filter state. Matches the existing watchlist heatmap pattern. |
| Score icon standardization | Remove icons entirely; Use numeric badges; Use consistent icon | Numeric value inside color-coded badge | Numeric display is scannable and sortable. Color-coded badge (green/amber/red) adds sentiment signal without relying on icon metaphors that vary by rank. |
| Sentiment alert free-tier limit | 0 alerts free; 3 alerts free; Unlimited free | 3 alerts free, unlimited premium | 3 alerts let casual users experience the value; the limit creates a natural conversion trigger for active traders who want more. |
| Navigation label | Keep "Reddit Trends"; Rename to "Social Sentiment"; Rename to "Social Trends" | "Social Trends" | Future-proofs for multi-source expansion. "Social Sentiment" implies analysis only; "Social Trends" encompasses both discovery (trending) and analysis. |
| Sentiment overlay vs. side-by-side charts | Overlaid dual-axis chart; Two separate charts stacked | Overlaid with toggle | Overlay allows direct visual correlation of sentiment and price movement. Toggle respects users who find dual-axis charts confusing. |
| Inline row expand vs. hover tooltip | Tooltip with post preview; Expandable row panel | Expandable row panel | Tooltips are too transient for post content that users want to read carefully. Expand panels persist until closed and accommodate 3 posts comfortably. |
| API exposure tier | Free API with rate limits; Premium only; No API | Premium only | API access is a high-value feature for quant users (Priya persona). Gating it to premium creates meaningful differentiation and monetization. |

---

## 14. Appendix: Competitor Feature Matrix

| Feature | IC (Current) | IC (Proposed) | Stocktwits | Unusual Whales | Yahoo Finance | Sentiment Investor |
|---|---|---|---|---|---|---|
| Inline sentiment on trending | ✗ | ✓ | ✓ | N/A | ✗ | ✓ |
| Price + % change on trending | ✗ | ✓ | ✗ | N/A | ✓ | ✗ |
| Heatmap view | ✗ | ✓ | ✗ | ✗ | ✓ | ✗ |
| Sentiment overlay on price chart | ✗ | ✓ | ✗ | ✗ | ✗ | ✓ |
| Sentiment alerts | ✗ | ✓ | ✗ | ✓ (options flow) | ✗ | ✓ |
| Multi-source sentiment | ✗ | ✓ (Phase 4) | ✗ (own platform) | ✗ | ✗ | ✓ |
| Historical accuracy metric | ✗ | ✓ (Phase 4) | ✗ | ✗ | ✗ | Partial |
| API access | ✗ | ✓ (Premium) | ✓ (Premium) | ✓ (Premium) | ✓ (Premium) | ✓ (Premium) |
| Comparison mode | ✗ | ✓ | ✗ | ✗ | ✗ | ✓ |

---

## 15. OpenClaw Data Pipeline Strategy

### 15.1 Current State & Context

InvestorCenter is actively integrating [OpenClaw](https://docs.openclaw.ai) (the open-source AI agent framework, formerly Clawdbot) as a data acquisition layer for Reddit and social media content. OpenClaw's Reddit scraper skill uses public JSON endpoints (`old.reddit.com`) to fetch subreddit posts without requiring the official Reddit API — eliminating API key dependency, rate-limit constraints, and the cost associated with Reddit's paid data access tiers.

This positions InvestorCenter to scale its social data collection far beyond the current 4-subreddit scope, while maintaining low operational cost and high resilience against Reddit API policy changes (a key risk identified in Section 11).

### 15.2 OpenClaw Integration Architecture

The proposed data pipeline leverages OpenClaw's capabilities in three layers:

```
┌──────────────────────────────────────────────────────────────────┐
│                     DATA ACQUISITION LAYER                       │
│                                                                  │
│  OpenClaw Agent (scheduled cron jobs)                            │
│  ├── reddit-scraper skill    → r/wallstreetbets, r/stocks, ...  │
│  ├── reddit-scraper skill    → r/SecurityAnalysis, r/ETFs, ...  │
│  ├── web_fetch / Firecrawl   → StockTwits public feeds          │
│  ├── web_fetch / Firecrawl   → Twitter/X public posts           │
│  └── web_fetch / Firecrawl   → Financial news headlines         │
│                                                                  │
├──────────────────────────────────────────────────────────────────┤
│                     PROCESSING LAYER                             │
│                                                                  │
│  ├── Ticker extraction (NER / regex on post titles + bodies)    │
│  ├── Sentiment classification (FinBERT or commercial NLP API)   │
│  ├── Deduplication & bot/spam filtering                         │
│  ├── Mention counting, upvote aggregation, velocity calc        │
│  └── Anomaly detection (pump-and-dump signals)                  │
│                                                                  │
├──────────────────────────────────────────────────────────────────┤
│                     STORAGE & SERVING LAYER                      │
│                                                                  │
│  ├── Time-series DB (sentiment scores, mention counts)          │
│  ├── Post metadata store (title, score, timestamp, link)        │
│  ├── Computed trending rankings (refreshed every 10-15 min)     │
│  └── REST API → InvestorCenter frontend + public API            │
└──────────────────────────────────────────────────────────────────┘
```

### 15.3 OpenClaw Reddit Scraper — Capabilities & Usage

The OpenClaw reddit-scraper skill supports the following operations that map directly to InvestorCenter's data needs:

| Operation | OpenClaw Command | IC Use Case |
|---|---|---|
| Fetch subreddit posts (hot/new/top/rising) | `--subreddit wallstreetbets --sort hot` | Trending list: identify most-discussed tickers in real-time |
| Search within a subreddit | `--subreddit stocks --search "NVDA"` | Per-ticker sentiment: find all mentions of a specific ticker |
| Search all of Reddit | `--search "TSLA earnings"` | Event-driven alerts: detect sentiment spikes around catalysts |
| Time-filtered top posts | `--subreddit investing --sort top --time week` | Historical sentiment: populate 7D/30D/90D lookback data |
| JSON output for processing | `--json --limit 100` | Pipeline ingestion: structured data for NLP and aggregation |

### 15.4 Scaling Beyond Reddit

OpenClaw's architecture extends naturally beyond Reddit through its web_fetch and Firecrawl integrations:

- **StockTwits:** Fetch public message streams per ticker via web_fetch. StockTwits posts include user-tagged Bullish/Bearish labels — an already-classified sentiment signal that requires no NLP processing.
- **Twitter/X:** Fetch public posts containing $TICKER cashtags via web_fetch with Firecrawl for bot-circumvention on rate-limited endpoints. Requires NLP sentiment classification.
- **Financial News:** Scrape headlines from major financial news sites (Reuters, Bloomberg, MarketWatch) via Firecrawl. Headline sentiment provides a professional/institutional counterweight to retail social sentiment.
- **Bluesky:** OpenClaw has a native Bluesky browsing skill (with firehose sampling). As financial communities grow on Bluesky, this becomes a zero-incremental-cost data source.

### 15.5 OpenClaw Scheduling & Freshness

OpenClaw supports cron-job scheduling for recurring data pulls. Proposed schedule:

| Data Source | Frequency | Posts per Pull | Estimated Daily Volume |
|---|---|---|---|
| r/wallstreetbets (hot + new) | Every 10 min | 100 | ~14,400 posts/day |
| r/stocks, r/investing, r/options | Every 15 min | 50 each | ~14,400 posts/day |
| r/SecurityAnalysis, r/ValueInvesting, r/ETFs, r/pennystocks | Every 30 min | 50 each | ~9,600 posts/day |
| StockTwits (top 50 tickers) | Every 15 min | 20 per ticker | ~96,000 messages/day |
| Twitter/X ($cashtag search) | Every 30 min | 50 per search | ~2,400 posts/day |
| Financial news headlines | Every 30 min | 25 | ~1,200 headlines/day |

This achieves the 10–15 minute freshness SLA defined in Section 8 while staying well within OpenClaw's resource constraints for a single-instance deployment.

### 15.6 Design Decisions: Why OpenClaw

| Decision | Alternatives Considered | Rationale |
|---|---|---|
| OpenClaw vs. Reddit Official API | Reddit Data API ($0.24/1000 API calls for enterprise tier) | OpenClaw's public JSON scraping eliminates API costs entirely. Reddit's official API has become increasingly restrictive (pricing changes in 2023, rate limit reductions). OpenClaw provides resilience through multiple access methods. |
| OpenClaw vs. custom scraper | Build a bespoke Python/Node scraper | OpenClaw provides battle-tested scraping with built-in Firecrawl fallback, bot-detection circumvention, caching (15-min TTL), and retry logic. Lower maintenance burden for a small team. |
| OpenClaw vs. third-party data vendor | Purchase social data from Quiver Quantitative, Sentdex, or similar | Third-party vendors cost $500–$5,000/month and introduce data dependency. OpenClaw gives InvestorCenter full control over data freshness, source selection, and processing pipeline. |
| Single OpenClaw instance vs. distributed | Multiple OpenClaw agents across subreddits | Start with single instance. OpenClaw's cron scheduling handles 50+ subreddits at 10-15 min intervals comfortably. Scale to multiple instances only if daily volume exceeds 200K posts. |

### 15.7 Risks Specific to OpenClaw

| Risk | Severity | Mitigation |
|---|---|---|
| Reddit blocks public JSON endpoints | High | OpenClaw's Firecrawl integration provides bot-circumvention fallback. Additionally, maintain a registered Reddit API key as a secondary fallback (free tier: 100 requests/min). |
| OpenClaw project discontinuation or breaking changes | Medium | Pin to a stable release version. The project is open-source (68K+ GitHub stars) with active community, reducing abandonment risk. Fork the repo as a contingency. |
| Scraping volume triggers IP-level blocks | Medium | Use OpenClaw's configurable User-Agent rotation. Implement request throttling below 1 req/sec per domain. Consider a residential proxy for high-volume periods. |
| Data quality degradation from public endpoints vs. official API | Low | Public JSON endpoints return the same structured data as the official API (title, score, num_comments, created_utc, selftext). Validate schema on every pull. |

---

## 16. Product Evolution: Multi-Source Social Intelligence

This section outlines how InvestorCenter's social features should evolve as the OpenClaw data pipeline matures and multi-source data becomes available. This is the long-term vision beyond the 20-week phased rollout in Section 12.

### 16.1 Evolution Stages

```
Stage 1 (Current → Phase 2)     Stage 2 (Phase 3-4)           Stage 3 (6-12 months)          Stage 4 (12-24 months)
─────────────────────────────    ──────────────────────────    ─────────────────────────────   ──────────────────────────
Reddit-only sentiment            + StockTwits, Twitter/X       + News sentiment, Bluesky       Composite Social Score
4 subreddits                     10+ subreddits                20+ subreddits                  Cross-platform consensus
Basic trending table             Enriched table + alerts       Multi-source sentiment page     Predictive sentiment signals
Manual sentiment page            Sentiment overlay on chart    Sector-level sentiment          AI-generated sentiment reports
No alerts                        Basic alerts                  Smart alerts (ML-tuned)         Autonomous alert suggestions
```

### 16.2 Stage 2: Multi-Platform Sentiment (Phase 3–4 Timeframe)

When StockTwits and Twitter/X data comes online via OpenClaw, the product should evolve in these ways:

**Unified Sentiment Score:** Replace the current Reddit-only sentiment badge with a composite score that weights signals from all available platforms. The weighting formula should account for platform characteristics:

| Platform | Signal Type | Weight Factor | Rationale |
|---|---|---|---|
| Reddit (finance subs) | Discussion sentiment (NLP-classified) | 1.0x (baseline) | Longer-form discussion provides richer context |
| StockTwits | Self-tagged Bullish/Bearish | 1.2x | Pre-classified by users; higher signal-to-noise for sentiment direction |
| Twitter/X | Cashtag mention sentiment (NLP) | 0.8x | Higher noise; more bots; shorter posts with less context |
| Financial News | Headline sentiment (NLP) | 1.5x | Professional/institutional signal; lower volume but higher quality |

**Source Attribution Bar:** On the trending table and sentiment detail page, show a horizontal stacked bar indicating how much of the sentiment signal comes from each platform (e.g., 50% Reddit, 30% StockTwits, 15% Twitter/X, 5% News). This gives users transparency into the signal composition.

**Platform-Specific Filters:** Extend the filter bar to allow users to isolate sentiment by source: "Show me only Reddit sentiment for NVDA" or "Show me only StockTwits sentiment." Power users want to compare platforms to identify divergences.

**Cross-Platform Divergence Alerts:** A new alert type: "Alert me when Reddit sentiment for [TICKER] is Bullish but Twitter/X sentiment is Bearish." Cross-platform divergence is often a leading indicator of narrative shifts.

### 16.3 Stage 3: Sector & Market-Level Sentiment (6–12 Months)

As data volume increases, sentiment analysis should expand from individual tickers to higher-order aggregations:

**Sector Sentiment Dashboard:** Aggregate ticker-level sentiment into sector-level views. Example: "Technology sector sentiment is 68% Bullish based on 2,400 posts across 45 tickers in the last 24h." This is directly competitive with Sentiment Investor's sector-level aggregation feature.

**Market Mood Index:** A single composite metric summarizing overall retail market sentiment across all tracked platforms and tickers. Displayed as a gauge or dial on the InvestorCenter dashboard. Historical Market Mood Index charted over time becomes a unique proprietary indicator.

**Sentiment-Driven Screener Enhancements:** New screener criteria enabled by richer data:
- "Stocks where sentiment flipped from Bearish to Bullish in the last 48h"
- "Stocks with >500 mentions this week but negative price performance" (potential contrarian plays)
- "Stocks with rising sentiment velocity across 2+ platforms"

**Earnings Sentiment Spike Analysis:** Detect and surface sentiment spikes around earnings dates. Overlay pre-earnings sentiment on post-earnings price movement to build the "Historical Accuracy" metric with stronger statistical power.

### 16.4 Stage 4: Predictive Intelligence & AI Reports (12–24 Months)

As InvestorCenter accumulates 12+ months of multi-source sentiment history alongside price data, the platform can evolve from descriptive analytics to predictive intelligence:

**Sentiment-Price Correlation Engine:** Backtested models showing the statistical relationship between sentiment shifts and subsequent price movement, per ticker and per sector. This transforms the "Historical Accuracy" metric from a simple average into a confidence-scored predictive signal. Example: "NVDA: When multi-source sentiment shifted to Bullish with >70% consensus across 2+ platforms, the stock was up an average of 3.1% within 5 trading days (72% hit rate, p<0.05)."

**AI-Generated Sentiment Reports:** Use LLM summarization (via Claude or similar) to generate daily/weekly narrative reports: "This week, semiconductor sentiment surged to a 90-day high driven by NVDA and AMD earnings beats. Reddit discussion volume spiked 3x on r/wallstreetbets, while StockTwits sentiment remained more measured at 58% Bullish. The divergence suggests retail enthusiasm is outpacing institutional conviction." These reports become a premium feature and a powerful content marketing asset.

**Autonomous Alert Suggestions:** ML-driven alert recommendations based on user watchlist and portfolio: "Based on your holdings, you might want to set an alert for TSLA — sentiment has been declining for 3 consecutive days and is approaching the Bearish threshold." This moves InvestorCenter from a passive tool to a proactive assistant.

**Sentiment API v2:** Enhanced API endpoints for quant users, including: streaming sentiment updates via webhooks, historical sentiment bulk export (CSV/Parquet), cross-platform consensus scores, and sentiment-price correlation coefficients. This API becomes a meaningful revenue stream for the quant/institutional audience.

### 16.5 Data Volume & Infrastructure Evolution

As the data pipeline scales, infrastructure needs will evolve:

| Stage | Estimated Daily Volume | Storage (Annual) | Compute Requirements |
|---|---|---|---|
| Stage 1 (Reddit only, 4 subs) | ~5K posts | ~2 GB | Single OpenClaw instance, basic cron |
| Stage 2 (Reddit expanded + StockTwits + X) | ~140K posts | ~50 GB | Single OpenClaw instance, 15-min cron, NLP batch processing |
| Stage 3 (+ News, Bluesky, sector agg) | ~200K+ posts | ~120 GB | 2–3 OpenClaw instances, real-time NLP pipeline, time-series DB |
| Stage 4 (Predictive, AI reports) | ~200K+ posts + model training | ~200 GB + model storage | GPU-enabled compute for model training, LLM API costs for reports |

### 16.6 Revised Monetization at Scale

As the product evolves, the tiering structure should expand:

| Tier | Price Point | Features |
|---|---|---|
| Free | $0 | Trending table (all sources), basic sentiment badge, 7D history, 3 alerts |
| Pro | $19/mo | 90D history, sentiment overlay on charts, comparison mode, unlimited alerts, platform-specific filters |
| Quant | $49/mo | API access (v1 + v2), bulk data export, sentiment-price correlation, cross-platform divergence alerts |
| Institutional | $199/mo | AI-generated reports, sector sentiment dashboard, Market Mood Index, custom alert rules, dedicated support |

This tiering progressively gates features by user sophistication and willingness to pay, creating a natural upgrade path from Maria (Free) → Jake (Pro) → Priya (Quant) → hedge fund analysts (Institutional).

---

*— End of Document —*
