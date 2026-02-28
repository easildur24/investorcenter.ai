# Product Requirements Document: Enhanced Fundamentals Data Experience

**Document Version:** 1.0
**Author:** Product Management
**Date:** February 28, 2026
**Status:** Draft — Pending Stakeholder Review

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Goals & Success Metrics](#2-goals--success-metrics)
3. [User Research (UXR)](#3-user-research-uxr)
4. [User Personas & Jobs-to-Be-Done](#4-user-personas--jobs-to-be-done)
5. [Current State Audit](#5-current-state-audit)
6. [Feature Requirements](#6-feature-requirements)
7. [User Flows](#7-user-flows)
8. [UX & Design Requirements](#8-ux--design-requirements)
9. [Technical Considerations](#9-technical-considerations)
10. [Monetization & Freemium Strategy](#10-monetization--freemium-strategy)
11. [Phased Rollout](#11-phased-rollout)
12. [Risks & Mitigations](#12-risks--mitigations)
13. [Open Questions](#13-open-questions)

---

## 1. Executive Summary

### Problem Statement

InvestorCenter computes and stores 50+ fundamental metrics per ticker — valuation multiples, profitability ratios, growth rates, quality scores, risk measures, and analyst estimates — yet presents them as raw, context-free numbers scattered across four disconnected tabs (TickerFundamentals sidebar, KeyStats, Metrics, Financials). Users see "P/E: 15.2" with no indication of whether that's cheap or expensive for the stock's sector. They see "Debt/Equity: 1.8" with no warning that this exceeds peer norms. They see 50+ metrics in KeyStats with equal visual weight, no prioritization, and no narrative connecting them.

The result: users who need fundamental analysis — the core value proposition for a stock research platform — must leave InvestorCenter to get context on InvestorCenter's own data. This is a retention-critical UX failure.

### Business Opportunity

Fundamental data is the single most-used surface on competing platforms (Koyfin, Simply Wall St, Stock Analysis, Finviz). The differentiator is never the data itself — every platform has P/E ratios — it's the **interpretation layer**: benchmarks, trend visualization, quality synthesis, and actionable signals. InvestorCenter already has unique assets that competitors lack:

- **IC Score sub-factors** that decompose quality into 10 dimensions
- **Sector percentile materialized views** already computed in the database
- **Peer comparison algorithm** (market cap, growth, margins, P/E, beta similarity scoring)
- **Lifecycle classification** (hypergrowth, growth, mature, value, turnaround)
- **Fair value models** (DCF, Graham Number, EPV) computed but not surfaced

Surfacing these existing backend capabilities in the fundamentals UX transforms raw numbers into an analytical narrative — the gap between a data dump and a research platform.

### Vision

Make InvestorCenter the only platform where a user can open a ticker page and immediately understand: **Is this stock fundamentally strong? Is it cheap or expensive relative to peers? Are the fundamentals improving or deteriorating? What are the red flags?** — all without leaving the page or doing mental math.

### Strategic Positioning vs. Competitors

| Dimension | Yahoo Finance | Stock Analysis | Simply Wall St | Koyfin | **InvestorCenter (Target)** |
|-----------|--------------|----------------|----------------|--------|----------------------------|
| Peer benchmarking | None inline | Basic percentile | Snowflake visual | Peer table | **Inline sector percentile bars on every metric** |
| Trend visualization | Separate charts | Sparklines (paid) | 10Y visual | Full charts | **5Y sparklines on key metrics, free tier** |
| Quality synthesis | None | Score card | Snowflake | None | **IC Score + F-Score + Z-Score unified health card** |
| Red flag detection | None | None | Warnings | None | **Automated red flag alerts with explanations** |
| Growth-vs-value context | None | None | Partial | None | **Lifecycle-aware metric weighting & interpretation** |
| Fair value integration | Analyst targets only | DCF + targets | DCF visual | None | **DCF, Graham, EPV with margin-of-safety gauge** |

### Expected Impact

| Metric | Current | 3-Month Target | 6-Month Target |
|--------|---------|----------------|----------------|
| Avg. time on ticker page | ~45s | 2.5 min | 4+ min |
| Fundamentals tab engagement (% of ticker page views) | ~18% | 40% | 55% |
| Ticker page → Screener conversion | ~3% | 8% | 12% |
| Free → Premium conversion from fundamentals paywall | N/A | 4% | 7% |
| User-reported "data quality" NPS score | Unknown (no baseline) | 35 | 50+ |

---

## 2. Goals & Success Metrics

### Primary KPIs

| KPI | Definition | Target | Measurement |
|-----|-----------|--------|-------------|
| **Fundamentals Engagement Rate** | % of ticker page sessions where user interacts with fundamentals data (scroll, tab click, tooltip hover) | 55% within 90 days | Event tracking on fundamentals components |
| **Time on Fundamentals** | Median time spent on Metrics/KeyStats/Financials tabs per session | 90 seconds (up from ~20s) | Session timing events |
| **Peer Comparison Usage** | % of ticker views where user views peer comparison panel | 30% | Click/expand events on peer panel |
| **Premium Conversion from Fundamentals** | % of free users who upgrade after encountering a fundamentals paywall (sparklines, peer deep-dive, red flags) | 5% of gated users | Conversion funnel tracking |

### Secondary KPIs

| KPI | Definition | Target |
|-----|-----------|--------|
| Fundamentals-to-Watchlist conversion | % of fundamentals sessions resulting in watchlist addition | 12% |
| Red flag interaction rate | % of users who hover/click red flag indicators | 25% |
| Fair value engagement | % of ticker views where user views fair value card | 20% |
| Sparkline hover rate | % of users who hover on trend sparklines for detail | 35% |
| Metric tooltip usage | Avg. tooltip hovers per fundamentals session | 3+ |
| Data freshness satisfaction | % of users rating data freshness "good" or "excellent" in survey | 75% |

### Anti-Goals (Explicitly Not Building)

- **Full financial modeling workspace**: InvestorCenter is a research platform, not a DCF modeling tool. We surface computed fair values; we don't let users build custom models.
- **Real-time fundamental data**: Financial statements update quarterly. Attempting real-time fundamentals creates false precision. We update on filing cadence.
- **Custom metric builder**: Users cannot create custom formulas. We curate the metrics that matter. Custom metrics add complexity without proportional value at our scale.
- **PDF/report generation**: No "download research report" in this phase. Export is limited to CSV for data portability.
- **Backtesting on fundamentals**: No "what would screening by low P/E + high ROE have returned?" functionality.

---

## 3. User Research (UXR)

### 3.1 Research Objectives

Validate assumptions about how retail investors consume fundamental data and identify the highest-leverage UX improvements before committing to development.

### 3.2 Research Plan

#### Study 1: Competitive Benchmarking Audit (Week 1-2)

**Method:** Heuristic evaluation + competitive teardown
**Sample:** 6 competing platforms (Yahoo Finance, Stock Analysis, Simply Wall St, Koyfin, Finviz, Morningstar)
**Deliverable:** Feature comparison matrix with screenshots, interaction pattern catalog, gap analysis

**Key Questions:**
- How do competitors present context for raw metrics (percentiles, color coding, peer comparison)?
- What is the typical information hierarchy on a fundamentals page?
- How do competitors handle data freshness and source attribution?
- What interpretation or synthesis features exist (health scores, red flags)?

#### Study 2: Moderated Usability Testing — Current State (Week 2-3)

**Method:** Remote moderated usability test (60 min per participant)
**Sample:** 8-10 participants across persona segments (2-3 per persona: Dividend Builder, Growth Hunter, Value Analyst, Momentum Trader)
**Recruitment:** Existing InvestorCenter users with ≥5 ticker page views in last 30 days
**Tool:** Zoom + screen recording

**Task Scenarios:**
1. "You're considering investing in [AAPL]. Using InvestorCenter, assess whether it's a fundamentally strong company."
2. "You want to know if [AAPL] is expensive or cheap compared to similar companies. Find that information."
3. "Identify any financial concerns or red flags about [MSFT] using the data available."
4. "Compare the profitability of [GOOGL] vs. its peers."

**Metrics:**
- Task completion rate (can they answer the question?)
- Time on task
- Number of tabs/pages navigated
- Errors/dead ends encountered
- SUS (System Usability Scale) score
- Post-task confidence rating ("How confident are you in the answer you found?")

**Key Questions:**
- Can users synthesize raw metrics into an investment thesis?
- Do users understand what metrics mean without tooltips?
- What metrics do users look at first? Which do they skip?
- Do users notice the absence of benchmarking/context?
- Where do users get stuck or abandon the task?

#### Study 3: Card Sort — Metric Organization (Week 3-4)

**Method:** Hybrid card sort (semi-open)
**Sample:** 15-20 participants
**Tool:** Optimal Workshop or similar
**Cards:** 40 metric labels (P/E, ROE, Debt/Equity, Free Cash Flow, etc.)
**Pre-defined categories available:** Valuation, Profitability, Growth, Financial Health, Risk, Quality, Dividends, Analyst Opinion
**Participants can:** Rename categories, create new ones, mark cards as "don't understand" or "not important"

**Key Questions:**
- How do retail investors mentally categorize financial metrics?
- Which metrics are considered "must-have" vs. "nice-to-have"?
- Are our current category groupings (Valuation, Profitability, etc.) aligned with user mental models?
- What metrics do users not understand?

#### Study 4: Prototype Testing — Enhanced Fundamentals (Week 5-7)

**Method:** Remote unmoderated A/B preference test + moderated deep dives
**Sample:** 20 unmoderated (quantitative) + 6 moderated (qualitative)
**Tool:** Maze (unmoderated) + Zoom (moderated)

**Prototypes (Figma):**
- **Variant A:** Current layout + inline sector percentile bars added to each metric
- **Variant B:** Redesigned "Fundamental Health Dashboard" with grouped cards, sparklines, peer comparison panel, and red flag alerts
- **Variant C:** Simply Wall St-inspired visual "snowflake" approach with narrative summary

**Metrics:**
- Preference ranking (A vs. B vs. C)
- Task completion rate on same scenarios as Study 2
- Time on task comparison
- Confidence rating comparison
- Qualitative feedback on what's most/least useful

#### Study 5: Survey — Metric Importance & Willingness to Pay (Week 4-5)

**Method:** Online survey (10-15 min)
**Sample:** 200+ InvestorCenter registered users
**Distribution:** In-app banner + email

**Survey Sections:**
1. **Investor profile:** Experience level, primary strategy (income, growth, value, momentum), portfolio size
2. **Metric importance:** Rate importance of 20 key metrics on 5-point scale
3. **Feature desirability:** Rank desired features (peer comparison, sparklines, red flags, fair value, quality scores)
4. **Current pain points:** "What's the hardest part about evaluating a stock's fundamentals?" (open text)
5. **Willingness to pay:** "Which of these features would make you consider upgrading to Premium?" (select all that apply)

### 3.3 UXR Success Criteria

| Research Output | Decision It Informs |
|----------------|---------------------|
| Metric importance ranking | Which metrics appear in the "at-a-glance" summary vs. detail view |
| Card sort results | Final metric grouping/tab structure |
| Usability test: task completion rates | Baseline for measuring improvement post-launch |
| Usability test: user confidence scores | Whether interpretation features (percentiles, color coding) increase decision confidence |
| Prototype preference test | Which visual approach to build (incremental enhancement vs. dashboard redesign) |
| WTP survey results | Which features to gate behind Premium |
| Mental model gaps | Where to add tooltips, education, or guided experiences |

### 3.4 UXR Timeline

| Week | Activity | Deliverable |
|------|----------|-------------|
| 1-2 | Competitive audit | Feature matrix + gap report |
| 2-3 | Usability testing (current state) | Findings report with video clips |
| 3-4 | Card sort | Dendrogram + category recommendations |
| 4-5 | Survey deployment + analysis | Metric priority list + WTP analysis |
| 5-7 | Prototype testing | Design recommendation with confidence levels |
| 8 | Synthesis & recommendation | Final UXR report → feeds Phase 1 design |

---

## 4. User Personas & Jobs-to-Be-Done

### Persona 1: The Dividend Income Builder

**Who:** Semi-retired investor, 55-65, managing 30-50 stock portfolio for income.
**Primary JTBD:** "Quickly assess whether a stock pays a sustainable, growing dividend without excessive risk."

**Fundamental metrics that matter most:**
- Dividend yield, payout ratio, FCF payout ratio, consecutive dividend years, dividend growth (5Y CAGR)
- Debt/Equity, Current Ratio, Interest Coverage (financial health context)
- IC Score (overall quality validation)

**Current pain:** Dividend yield is shown but payout sustainability requires cross-referencing 3 tabs. No warning when payout ratio exceeds earnings.

**Enhanced experience need:** A single "Dividend Health" card showing yield + sustainability + safety in one view. Red flag if payout ratio > 80% or FCF payout > 100%.

### Persona 2: The Growth Stock Hunter

**Who:** Active retail investor, 25-40, seeking high-growth companies pre-mainstream.
**Primary JTBD:** "Find fast-growing companies with accelerating fundamentals and know if I'm overpaying for growth."

**Fundamental metrics that matter most:**
- Revenue growth (YoY, 3Y CAGR), EPS growth, FCF growth
- Gross margin expansion/compression trend
- PEG ratio, P/S relative to growth rate
- Analyst estimate revisions (upward = positive signal)

**Current pain:** Revenue growth shown as single number. No trend. No PEG context. Can't tell if growth is accelerating or decelerating without going to Financials tab and doing mental math.

**Enhanced experience need:** Growth trend sparklines, PEG ratio benchmarked to sector, analyst revision direction indicator, lifecycle classification ("Hypergrowth" badge).

### Persona 3: The Value-Oriented Analyst

**Who:** Experienced investor, 30-50, seeking undervalued stocks with margin of safety.
**Primary JTBD:** "Determine if a stock is trading below intrinsic value with strong enough fundamentals to justify a position."

**Fundamental metrics that matter most:**
- P/E, P/B, P/S with sector percentile (is it actually cheap?)
- DCF fair value, Graham Number, margin of safety %
- ROE, ROIC (quality of the business)
- Piotroski F-Score, Altman Z-Score (quality validation)
- Debt/Equity, Interest Coverage (risk check)

**Current pain:** P/E shown without sector context. Fair value models computed but not surfaced on ticker page. F-Score and Z-Score exist in Metrics tab but disconnected from valuation assessment.

**Enhanced experience need:** Valuation vs. peers percentile bars, fair value gauge with margin of safety, integrated quality + valuation assessment ("Cheap and high quality" vs. "Cheap but deteriorating").

### Persona 4: The Momentum Swing Trader

**Who:** Active trader, 25-45, technically-driven but fundamentals-aware.
**Primary JTBD:** "Confirm that a technical setup is backed by improving fundamentals (not a value trap)."

**Fundamental metrics that matter most:**
- Earnings surprise direction, revenue beat/miss
- Analyst estimate revisions (consensus moving up = confirmation)
- Short interest ratio (contrarian signal)
- ROE trend (improving fundamentals = momentum confirmation)

**Current pain:** Earnings and analyst data exist in separate tabs. No quick "fundamentals confirm" or "fundamentals diverge" signal for a momentum trader who wants to spend 10 seconds on fundamentals.

**Enhanced experience need:** A "Fundamental Momentum" indicator showing whether key metrics are improving/stable/deteriorating QoQ. Earnings surprise badges.

### Jobs-to-Be-Done Map

| Job | Currently Served? | Enhancement Priority |
|-----|-------------------|---------------------|
| See if a metric is good/bad relative to sector | No — raw numbers only | **P0 — Highest** |
| Get a quick fundamental health assessment | No — must synthesize 50+ metrics manually | **P0** |
| Identify red flags or deteriorating fundamentals | No — no alerts or warnings | **P0** |
| See how metrics are trending over time | Partially — YoY change in Financials only | **P1** |
| Compare to peer companies on key metrics | No — peer algorithm exists but not surfaced | **P1** |
| Assess fair value / margin of safety | No — computed but not displayed | **P1** |
| Understand what a metric means | Partially — tooltips in Metrics tab only | **P2** |
| Know how fresh the data is | Poorly — timestamps inconsistent | **P2** |

---

## 5. Current State Audit

### Architecture of Fundamentals Data (What Already Exists)

| Capability | Backend Status | Frontend Status | Gap |
|------------|---------------|-----------------|-----|
| 50+ fundamental metrics | Computed & stored | Displayed as raw numbers | **No interpretation layer** |
| Sector percentile rankings | Materialized view (`mv_latest_sector_percentiles`) with min/p10/p25/p50/p75/p90/max | Not surfaced anywhere | **Backend ready, frontend missing** |
| Peer comparison | Python algorithm (5-factor similarity scoring, top 5 peers) | Not surfaced on ticker page | **Backend ready, frontend missing** |
| Lifecycle classification | Computed (hypergrowth/growth/mature/value/turnaround) | Not surfaced | **Backend ready, frontend missing** |
| Fair value (DCF, Graham, EPV) | Computed & stored | Not on ticker page | **Backend ready, frontend missing** |
| IC Score with 10 sub-factors | Full pipeline, v2.1 | Shown in sidebar card only | **Partially surfaced** |
| Piotroski F-Score | Computed | Shown in Metrics > Quality tab | **Disconnected from health narrative** |
| Altman Z-Score | Computed with interpretation | Shown in Metrics > Quality tab | **Disconnected from health narrative** |
| Analyst estimates & revisions | Stored | Shown in Metrics > Analyst tab | **No revision trend or confidence** |
| 5Y historical metrics | Available via financial statements | Only YoY change in Financials table | **No sparklines or multi-year trends** |

### Current Component Layout Issues

**TickerFundamentals (Sidebar):**
- 15 metrics as plain key-value pairs
- No color coding, no benchmarking, no tooltips
- Data source priority logic (manual > IC Score > Polygon) is opaque to user

**KeyStatsTab:**
- 50+ metrics with equal visual weight
- 15 GroupCard sections — overwhelming
- No prioritization or "what matters most" guidance
- No benchmarking against peers or sector

**MetricsTab:**
- Best-organized of all tabs (category sub-tabs, MetricCard with tooltips)
- Some color coding (Z-Score, F-Score, payout ratio)
- Still missing: peer comparison, trend sparklines, red flags

**FinancialsTab:**
- Excellent raw data table with YoY changes
- Missing: common-size analysis, trend visualization, quality indicators

---

## 6. Feature Requirements

### P0 — Fundamentals Interpretation Layer (Phase 1)

#### 6.1 Sector Percentile Bars on Key Metrics

**Description:** Every key metric displayed in TickerFundamentals, Metrics, and KeyStats tabs shows an inline horizontal percentile bar indicating where the stock falls within its sector.

**Behavior:**
- Bar shows sector distribution (p10, p25, p50, p75, p90) as gradient zones
- Stock's position marked with a dot/marker on the bar
- Color zones: Green (favorable), Yellow (neutral), Red (unfavorable) — direction-aware (low P/E = green, low ROE = red)
- Hover shows: "Ranks at 23rd percentile in Technology sector (lower is better for P/E)"
- Data source: `mv_latest_sector_percentiles` materialized view (already computed)

**Metrics to include percentile bars (Phase 1):**
- Valuation: P/E, P/B, P/S, EV/EBITDA
- Profitability: ROE, ROA, Gross Margin, Net Margin, Operating Margin
- Financial Health: Debt/Equity, Current Ratio, Interest Coverage
- Growth: Revenue Growth YoY, EPS Growth YoY

**Acceptance criteria:**
- Percentile bar renders on all 14 metrics listed above
- "Lower is better" metrics (P/E, P/B, P/S, D/E) have inverted color scales
- Hover tooltip shows exact percentile, sector name, and sample size
- Graceful degradation: if sector percentile data unavailable, metric renders without bar (no error)

#### 6.2 Fundamental Health Summary Card

**Description:** A new card at the top of the fundamentals area (above current metric lists) that synthesizes multiple signals into a quick-read health assessment.

**Components:**
- **Overall Health Badge:** "Strong" / "Healthy" / "Fair" / "Weak" / "Distressed" — derived from weighted combination of:
  - Piotroski F-Score (0-9)
  - Altman Z-Score (distress/grey/safe zones)
  - IC Score financial health sub-factor
  - Debt/Equity percentile
- **Key Strengths** (up to 3): auto-generated from metrics in top quartile of sector (e.g., "Gross margin ranks in top 10% of Technology sector")
- **Key Concerns** (up to 3): auto-generated from metrics in bottom quartile or crossing thresholds (e.g., "Debt/Equity exceeds sector median by 2.1x")
- **Lifecycle Badge:** Shows lifecycle classification (Hypergrowth / Growth / Mature / Value / Turnaround) with brief explanation

**Acceptance criteria:**
- Card renders on all stock ticker pages
- Health badge accurately reflects F-Score + Z-Score + IC Score health
- Strengths/concerns auto-generated from sector percentile data
- Lifecycle badge matches backend classification

#### 6.3 Red Flag Detection & Display

**Description:** Automated detection and prominent display of concerning metric combinations.

**Red Flag Rules (Phase 1):**

| Red Flag | Trigger Condition | Severity | Display |
|----------|-------------------|----------|---------|
| Unsustainable dividend | Payout ratio > 100% OR FCF payout > 120% | High | Badge on dividend metrics |
| High leverage risk | D/E > sector p90 AND Interest Coverage < 2x | High | Badge on financial health section |
| Declining profitability | Operating margin declined 3+ consecutive quarters | Medium | Trend indicator on margin metrics |
| Cash burn concern | Negative FCF for 2+ consecutive quarters (non-hypergrowth) | High | Badge on cash flow metrics |
| Earnings quality warning | Operating Cash Flow / Net Income < 0.5 (for 2+ quarters) | Medium | Badge on profitability section |
| Valuation outlier | P/E > sector p95 AND PEG > 3 | Low | Badge on valuation metrics |
| Altman distress | Z-Score < 1.81 | High | Prominent warning on health card |
| Weak Piotroski | F-Score ≤ 3 | Medium | Badge on quality section |

**Display:**
- Red/orange warning badges adjacent to relevant metrics
- Expandable explanation: "Payout ratio of 115% means the company is paying more in dividends than it earns. This may not be sustainable."
- Aggregated in Fundamental Health Summary Card under "Key Concerns"

### P1 — Contextual Enrichment (Phase 2)

#### 6.4 Peer Comparison Panel

**Description:** Expandable panel on the ticker page showing the stock's key metrics side-by-side with its top 5 peers (already computed by similarity algorithm).

**Layout:**
- Collapsed by default: "Compare to 5 similar companies →"
- Expanded: horizontal table with stock + 5 peers as columns
- Rows: IC Score, P/E, ROE, Revenue Growth, Net Margin, D/E, Market Cap
- Each cell color-coded relative to the group (best = green, worst = red)
- Peer tickers are clickable (navigate to their ticker page)
- "How peers are selected" info tooltip explaining similarity algorithm

**Data source:** `PeerComparisonResult` from `ic-score-service/pipelines/utils/peer_comparison.py`

#### 6.5 Trend Sparklines

**Description:** Mini inline charts (sparklines) showing 5-year quarterly trends for key metrics.

**Metrics with sparklines:**
- Revenue, Net Income, Free Cash Flow (absolute values)
- Gross Margin, Operating Margin, Net Margin (percentages)
- ROE, ROA (returns)
- Debt/Equity (leverage trend)
- EPS (per-share earnings)

**Behavior:**
- 20px tall inline sparkline adjacent to the current value
- Color: green if trending up (for metrics where up is good), red if trending down
- Hover shows quarterly values in tooltip
- Click expands to full chart overlay

**Data source:** Historical financial statements (already stored from SEC EDGAR via Polygon)

#### 6.6 Fair Value Integration

**Description:** Display computed fair value estimates on the ticker page with margin of safety visualization.

**Components:**
- **Fair Value Card:** Shows DCF fair value, Graham Number, and EPV (Earnings Power Value)
- **Margin of Safety Gauge:** Visual showing current price vs. fair value range
  - Green zone: price < fair value (potential upside)
  - Yellow zone: price ≈ fair value (fairly valued)
  - Red zone: price > fair value (potential overvaluation)
- **Analyst Target Overlay:** Show analyst consensus price target on same gauge for comparison
- **Confidence Indicator:** Based on model input quality and analyst coverage breadth

**Data source:** Already computed in backend — DCF, Graham Number, EPV models

### P2 — Advanced Features (Phase 3)

#### 6.7 Fundamentals Narrative Summary

**Description:** AI-generated 2-3 sentence summary of the stock's fundamental profile.

**Example:** "Apple is a mature, highly profitable technology company trading at a 15% premium to sector median P/E. Margins are stable at best-in-class levels (30% net margin, sector p98), and the company returns significant capital via buybacks and dividends. Key consideration: revenue growth has decelerated to 5% YoY, below the sector median of 12%."

**Generation:** Template-based with metric insertion (not LLM-generated in real-time) to ensure consistency, speed, and cost control.

#### 6.8 Historical Metric Tracker

**Description:** Dedicated view showing how any metric has changed over 5 years with quarterly granularity.

**Behavior:**
- User clicks any metric → opens full-width chart showing 20-quarter history
- Overlay options: sector median trend, peer average trend
- Annotations: earnings dates, significant events

#### 6.9 Custom Metric Dashboard

**Description:** User can pin their most-important metrics (up to 12) to a personalized dashboard at the top of the fundamentals area.

**Behavior:**
- Drag-and-drop metric selection from full metric catalog
- Pinned metrics show: current value, percentile bar, sparkline, YoY change
- Dashboard layout persisted per user (Premium feature)

---

## 7. User Flows

### Flow 1: Quick Fundamental Assessment (Target: < 30 seconds)

```
User lands on ticker page
  → Sees Fundamental Health Summary Card at top of sidebar
  → Reads health badge ("Strong") + lifecycle ("Mature")
  → Scans 3 key strengths, 2 key concerns
  → If red flags present → sees warning badges → clicks to expand explanation
  → Decision: "Fundamentals look solid, let me check valuation"
  → Scrolls to valuation metrics with sector percentile bars
  → Sees P/E at 23rd percentile (cheaper than 77% of sector)
  → Confidence: HIGH — user answered "Is this fundamentally strong?" in < 30 seconds
```

### Flow 2: Peer Comparison Deep Dive (Target: < 2 minutes)

```
User on ticker page, wants to compare vs. peers
  → Clicks "Compare to 5 similar companies" panel
  → Sees stock vs. 5 peers in side-by-side table
  → Notices stock has highest ROE but also highest D/E
  → Clicks peer ticker "MSFT" to compare in new tab
  → Returns and checks fair value card
  → Sees stock trading 12% below DCF fair value
  → Decision: "Cheaper than peers with higher returns, but more leveraged — worth investigating further"
```

### Flow 3: Dividend Safety Check (Dividend Builder Persona)

```
User researching high-yield stock
  → Sees Fundamental Health Card: "Fair" health, 1 red flag
  → Red flag badge: "Unsustainable dividend — payout ratio 115%"
  → Expands: "Company is paying more in dividends than it earns..."
  → Checks dividend section: yield 6.2%, FCF payout 130%
  → Sees sector percentile bar: D/E at 85th percentile (high)
  → Trend sparkline: net margin declining 4 consecutive quarters
  → Decision: "Yield is attractive but dividend is at risk — pass"
```

### Flow 4: Growth Validation (Growth Hunter Persona)

```
User found stock via screener, checking fundamentals
  → Health Card shows "Hypergrowth" lifecycle badge
  → Key strengths: "Revenue growth 45% YoY (sector p95)", "Gross margin expanding"
  → Checks growth sparklines: revenue accelerating, margins expanding
  → PEG ratio: 1.8 — percentile bar shows 60th in sector (slightly expensive but not extreme)
  → Peer comparison: fastest growing among peers, cheapest on P/S basis
  → Analyst estimates: consensus revenue revised up 12% in last 90 days
  → Decision: "Growth is real and accelerating, valuation reasonable for growth rate — add to watchlist"
```

---

## 8. UX & Design Requirements

### 8.1 Information Hierarchy

The fundamentals experience should follow a progressive disclosure pattern:

**Layer 1 — Glanceable (0-10 seconds):** Fundamental Health Summary Card visible without scrolling. Health badge, lifecycle tag, top strengths/concerns. User gets the "headline" instantly.

**Layer 2 — Scannable (10-60 seconds):** Key metrics with sector percentile bars in sidebar (TickerFundamentals). Red flag badges visible. User scans the numbers with context.

**Layer 3 — Explorable (1-5 minutes):** Metrics tab with full category breakdown, sparklines, peer comparison panel. User goes deep on areas of interest.

**Layer 4 — Comprehensive (5+ minutes):** Financial statements, historical metric tracker, full peer deep dive. Power user territory.

### 8.2 Sector Percentile Bar Design

```
Metric Label                    Value     [=========|====•=====|=========]
                                          p10  p25  p50  stock  p75  p90

Color zones (for "higher is better" metrics like ROE):
  [  Red  |  Orange  |  Yellow  |  Light Green  |  Green  ]
  p0-p25     p25-p40    p40-p60     p60-p75       p75-p100

Inverted for "lower is better" (P/E, D/E):
  [  Green  |  Light Green  |  Yellow  |  Orange  |  Red  ]
  p0-p25       p25-p40         p40-p60    p60-p75    p75-p100
```

### 8.3 Color System

| Signal | Color | Usage |
|--------|-------|-------|
| Strong positive | `green-600` | Top quartile metric, healthy range |
| Moderate positive | `green-400` | Above median, acceptable |
| Neutral | `yellow-500` | Near median, unremarkable |
| Moderate concern | `orange-500` | Below median, worth noting |
| Negative / Red flag | `red-600` | Bottom quartile, threshold breach |
| Lifecycle: Hypergrowth | `purple-500` | Badge color |
| Lifecycle: Growth | `blue-500` | Badge color |
| Lifecycle: Mature | `slate-500` | Badge color |
| Lifecycle: Value | `amber-500` | Badge color |
| Lifecycle: Turnaround | `orange-500` | Badge color |

### 8.4 Responsive Behavior

- **Desktop (≥1024px):** Full sidebar with Health Card + percentile bars. Peer panel inline.
- **Tablet (768-1023px):** Health Card collapses to badge + expand. Percentile bars simplified (no labels).
- **Mobile (<768px):** Health Card as top-level accordion. Metrics in single-column with percentile dots (not full bars). Peer comparison as horizontal scroll cards.

### 8.5 Accessibility

- All color indicators must have text/icon alternatives (not color-only)
- Percentile bars must be screen-reader accessible ("P/E ratio: 15.2, 23rd percentile in Technology sector, below median — favorable")
- Red flag badges must use aria-live for dynamic content
- Sparklines must have alt text summarizing the trend direction
- Minimum contrast ratios per WCAG 2.1 AA

### 8.6 Performance Requirements

- Fundamental Health Card must render within 200ms of page load (server-side data)
- Percentile bars must not add >50ms to metric rendering
- Sparklines lazy-loaded on scroll (not blocking initial paint)
- Peer comparison panel data fetched on expand (not on page load)

---

## 9. Technical Considerations

### 9.1 Backend — Already Built (Minimal Work)

| Component | Status | Location | Work Needed |
|-----------|--------|----------|-------------|
| Sector percentiles | Materialized view exists | `backend/database/sector_percentiles.go` | Expose via API endpoint |
| Peer comparison | Algorithm complete | `ic-score-service/pipelines/utils/peer_comparison.py` | Expose via API endpoint |
| Lifecycle classification | Computed | `backend/models/ic_score_phase3.go` | Include in ticker API response |
| Fair value models | Computed | Backend IC Score pipeline | Expose via API endpoint |
| F-Score, Z-Score | Computed | FMP metrics pipeline | Already in Metrics API |

### 9.2 New API Endpoints Needed

| Endpoint | Purpose | Data Source |
|----------|---------|-------------|
| `GET /api/v1/stocks/{ticker}/sector-percentiles` | Return sector percentile data for all metrics | `mv_latest_sector_percentiles` view |
| `GET /api/v1/stocks/{ticker}/peers` | Return top 5 peers with comparison metrics | Peer comparison Python service |
| `GET /api/v1/stocks/{ticker}/fair-value` | Return DCF, Graham, EPV fair values | IC Score pipeline |
| `GET /api/v1/stocks/{ticker}/health-summary` | Return synthesized health assessment | Composite of existing data |
| `GET /api/v1/stocks/{ticker}/metric-history/{metric}` | Return 5Y quarterly history for a metric | Financial statements DB |

### 9.3 Frontend Components to Build

| Component | Description | Priority |
|-----------|-------------|----------|
| `SectorPercentileBar` | Reusable inline percentile visualization | P0 |
| `FundamentalHealthCard` | Summary card with badge, strengths, concerns | P0 |
| `RedFlagBadge` | Warning indicator with expandable explanation | P0 |
| `LifecycleBadge` | Colored badge showing lifecycle stage | P0 |
| `PeerComparisonPanel` | Expandable side-by-side peer table | P1 |
| `TrendSparkline` | 20px inline mini chart | P1 |
| `FairValueGauge` | Visual margin-of-safety indicator | P1 |
| `MetricHistoryChart` | Full-width expandable metric history | P2 |
| `CustomMetricDashboard` | User-pinnable metric grid | P2 |

### 9.4 Data Freshness Strategy

| Data Type | Update Frequency | Staleness Threshold | User Display |
|-----------|-----------------|---------------------|--------------|
| Sector percentiles | Daily (materialized view refresh) | >2 days | "Sector data as of [date]" |
| Peer comparisons | Daily | >2 days | "Peers updated [date]" |
| Financial statements | On SEC filing | >100 days since last quarter | "Latest filing: Q3 2025" |
| Fair value models | Daily (price-dependent) | >1 day | "Fair value as of [date]" |
| Analyst estimates | Daily | >1 day | "Consensus as of [date]" |

---

## 10. Monetization & Freemium Strategy

### Free Tier Includes

- Fundamental Health Summary Card (badge + lifecycle, limited to 2 strengths/concerns)
- Sector percentile bars on 6 core metrics (P/E, ROE, Gross Margin, D/E, Revenue Growth, Current Ratio)
- Red flag alerts (severity: High only)
- Basic metric display (current layout)

### Premium Tier Unlocks

| Feature | Rationale for Gating |
|---------|---------------------|
| Sector percentile bars on all 14+ metrics | Power users want comprehensive benchmarking |
| Full peer comparison panel (5 peers, all metrics) | High-value differentiated feature |
| Trend sparklines (all metrics) | Trend analysis is a power-user need |
| Fair value gauge with DCF, Graham, EPV | High-value computed insight |
| Full red flag suite (all severities + explanations) | Drives urgency for upgrade |
| Fundamental narrative summary | AI-generated content as premium value |
| Historical metric tracker (5Y charts) | Deep analysis feature |
| Custom metric dashboard | Personalization as premium |
| All strengths/concerns (unlimited) | Free tier teases, Premium delivers |

### Paywall UX

- Gated features show blurred/placeholder content with lock icon
- "Upgrade to see peer comparison for AAPL" CTA with one-click upgrade
- Consistent with existing watchlist paywall patterns

---

## 11. Phased Rollout

### Phase 1: Interpretation Layer (Sprints 1-3, ~6 weeks)

**Scope:** Sector percentile bars, Fundamental Health Summary Card, Red Flag Detection, Lifecycle Badge

**Dependencies:**
- API endpoint for sector percentile data
- Frontend: `SectorPercentileBar`, `FundamentalHealthCard`, `RedFlagBadge`, `LifecycleBadge` components
- UXR Study 1-3 complete (competitive audit, usability baseline, card sort)

**Exit criteria:**
- 14 metrics display percentile bars with correct direction-awareness
- Health card renders on all stock ticker pages with accurate health/lifecycle assessment
- 8 red flag rules implemented and triggering correctly
- Mobile responsive per spec

### Phase 2: Contextual Enrichment (Sprints 4-6, ~6 weeks)

**Scope:** Peer Comparison Panel, Trend Sparklines, Fair Value Integration

**Dependencies:**
- API endpoints for peers, fair value, metric history
- Frontend: `PeerComparisonPanel`, `TrendSparkline`, `FairValueGauge` components
- UXR Study 4 complete (prototype testing results inform final design)
- Phase 1 deployed and baseline metrics collected

**Exit criteria:**
- Peer panel shows top 5 similar companies with comparison table
- Sparklines render for 10+ metrics with correct trend coloring
- Fair value gauge shows margin of safety with 3 valuation models
- Premium gating in place for Phase 2 features

### Phase 3: Advanced & Personalization (Sprints 7-9, ~6 weeks)

**Scope:** Narrative Summary, Historical Metric Tracker, Custom Metric Dashboard

**Dependencies:**
- Template-based narrative generation system
- Full metric history API
- User preference storage for custom dashboard
- Phase 2 metrics showing engagement improvement

**Exit criteria:**
- Narrative generates accurate 2-3 sentence summaries
- Historical charts render 5Y quarterly data for any metric
- Custom dashboard supports pin/unpin for 12 metrics
- Premium tier adoption shows measurable lift

---

## 12. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Sector percentile data stale or missing for small sectors | Medium | Medium | Fallback to broader industry grouping; show "Insufficient sector data" when sample < 10 |
| Red flags generate false positives (e.g., REITs with high D/E is normal) | High | High | Lifecycle-aware and sector-aware thresholds; REIT/financial sector has adjusted rules |
| Information overload — adding context makes it MORE overwhelming | Medium | High | Progressive disclosure; Health Card is the "TLDR"; details only on drill-down; validate with UXR Study 4 |
| Peer comparison shows irrelevant peers | Medium | Medium | Show similarity score; "How peers are selected" tooltip; user override in Phase 3 |
| Fair value models give misleading signals for high-growth or pre-profit companies | High | High | Confidence indicator; suppress fair value for pre-revenue/pre-profit companies; "Fair value models are less reliable for hypergrowth companies" disclaimer |
| Performance regression from additional API calls and rendering | Low | Medium | Lazy loading for sparklines and peer panel; server-side health card; CDN caching for percentile data |
| Premium feature gating frustrates free users | Medium | Medium | Generous free tier (6 percentile bars, high-severity red flags, health badge); paywall shows value, not dead ends |
| Data source inconsistencies between FMP, Polygon, IC Score pipeline | Medium | High | Single source of truth per metric documented; API layer normalizes before frontend; data source badge for transparency |

---

## 13. Open Questions

| # | Question | Owner | Decision Needed By |
|---|----------|-------|-------------------|
| 1 | Should the Fundamental Health Card replace the existing IC Score sidebar card, or live alongside it? | Product + Design | Phase 1 design start |
| 2 | How should sector percentiles handle companies in multiple sectors (e.g., Amazon: Tech + Retail)? | Data Engineering | Phase 1 API development |
| 3 | Should red flag thresholds be absolute (D/E > 2) or relative (D/E > sector p90)? Recommendation: relative for most, absolute for critical (Z-Score distress). | Product | Phase 1 rule engine |
| 4 | What is the right number of free-tier percentile bars (currently proposed: 6)? Too few = no value, too many = no upgrade incentive. | Product + Growth | Phase 1 launch |
| 5 | Should peer comparison use the existing IC Score peer algorithm or a simplified version for the frontend? | Engineering | Phase 2 API design |
| 6 | How do we handle tickers with incomplete fundamental data (SPACs, recent IPOs, foreign ADRs)? | Product + Engineering | Phase 1 edge cases |
| 7 | Should the narrative summary (Phase 3) use LLM generation or template-based? LLM is richer but costly and unpredictable. | Product + Engineering | Phase 3 planning |
| 8 | Do we need a separate mobile-specific design review or is responsive sufficient? | Design + UXR | After UXR Study 2 |

---

*This PRD is a living document. It will be updated based on UXR findings (Section 3) and stakeholder feedback. Next step: Kick off UXR Study 1 (Competitive Audit) and begin Phase 1 API development for sector percentile endpoint.*
