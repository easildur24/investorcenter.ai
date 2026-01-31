# IC Score: Product Design & Technical Specification

> **Version**: 2.0
> **Date**: January 2026
> **Status**: Draft for Review

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Competitive Analysis](#competitive-analysis)
3. [Product Vision](#product-vision)
4. [Scoring Framework Design](#scoring-framework-design)
5. [Factor Definitions & Calculations](#factor-definitions--calculations)
6. [Technical Architecture](#technical-architecture)
7. [Data Pipeline Specification](#data-pipeline-specification)
8. [API Specification](#api-specification)
9. [UI/UX Design](#uiux-design)
10. [Implementation Roadmap](#implementation-roadmap)
11. [Appendix](#appendix)

---

## 1. Executive Summary

### 1.1 What is IC Score?

IC Score is InvestorCenter's proprietary stock rating system that provides investors with a single, comprehensive metric (0-100) to evaluate investment opportunities. Unlike simplistic rating systems, IC Score combines fundamental analysis, technical signals, market sentiment, and alternative data into an intelligent, context-aware scoring framework.

### 1.2 Key Differentiators

| Feature | IC Score | Competitors |
|---------|----------|-------------|
| **Sector-Relative Scoring** | ✅ Context-aware by sector | Most use absolute thresholds |
| **Lifecycle-Aware** | ✅ Adjusts for growth vs value | One-size-fits-all |
| **Transparency** | ✅ Full methodology disclosure | Often black-box |
| **Alternative Data** | ✅ Insider, institutional, sentiment | Limited or premium-only |
| **Real-Time Updates** | ✅ Daily recalculation | Weekly or manual |

### 1.3 Design Principles

1. **Intelligent Context**: Scores are meaningful within sector and company lifecycle context
2. **Data-Driven Transparency**: Every score component is explainable and traceable
3. **Statistical Rigor**: Percentile-based scoring eliminates arbitrary thresholds
4. **Actionable Insights**: Ratings map directly to investment decisions
5. **Graceful Degradation**: System provides value even with incomplete data

---

## 2. Competitive Analysis

### 2.1 YCharts Scoring System

**Approach**: Composite scores using weighted averages of financial metrics

| Score Type | Methodology | Key Features |
|------------|-------------|--------------|
| **Value Score** | Weighted average of P/E, P/B, P/S, EV/EBITDA | Decile ranking (1-10) |
| **Fundamental Score** | 10 fundamental metrics | Sector-relative |
| **Custom Scoring** | User-defined weights on 4,500+ metrics | Highly customizable |

**Strengths**:
- Historical backtest shows Value Score 10 outperforms S&P 500
- Transparency in methodology
- Customization capabilities

**Weaknesses**:
- Separate scores require synthesis by user
- No alternative data integration
- Limited sentiment analysis

**Lessons for IC Score**:
- ✅ Use decile/percentile ranking for relative comparisons
- ✅ Backtest and validate scoring methodology
- ✅ Consider offering customization for power users

---

### 2.2 Simply Wall St Snowflake

**Approach**: Visual 5-axis radar chart with binary checks

| Dimension | # of Checks | Example Checks |
|-----------|-------------|----------------|
| Value | 6 | P/E below sector median, PEG < 1 |
| Future Growth | 6 | Revenue growth > 10%, EPS growth > 15% |
| Past Performance | 6 | 5-year ROE > 15%, consistent margins |
| Financial Health | 6 | D/E < 1, current ratio > 1.5 |
| Dividends | 6 | Yield > 2%, payout sustainable |

**Scoring**: Each passed check = 1 point → Max 30 points total

**Visual Design**:
- Snowflake shape expands with higher scores
- Color gradient: Red (low) → Green (high)
- Intuitive at-a-glance assessment

**Strengths**:
- Highly visual and intuitive
- Open-source methodology (GitHub)
- 5 dimensions cover key investment criteria

**Weaknesses**:
- Binary checks lose nuance (barely passing = fully passing)
- No alternative data (insider, institutional)
- Dividend dimension penalizes growth companies

**Lessons for IC Score**:
- ✅ Visual representation is powerful for quick assessment
- ✅ Multiple dimensions help investors understand strengths/weaknesses
- ✅ Consider open-sourcing methodology for trust
- ⚠️ Avoid binary checks - use continuous scoring

---

### 2.3 Seeking Alpha Quant Ratings

**Approach**: 5-factor model with sector-relative grades

| Factor | Weight | Metrics Used |
|--------|--------|--------------|
| Value | Variable | P/E, P/S, P/B, EV/EBITDA, PEG |
| Growth | Variable | Revenue, EPS, EBITDA growth |
| Profitability | Variable | Margins, ROE, ROA, ROIC |
| Momentum | Variable | Price returns (1m, 3m, 6m, 12m) |
| EPS Revisions | Variable | Analyst estimate changes |

**Key Design Decisions**:
- **Sector-relative**: Compares stocks within same sector
- **Dynamic weighting**: Factors with higher predictability get higher weights
- **Grade system**: A+ to F for each factor, 1.0-5.0 overall

**Performance Claims**:
- Strong Buy recommendations outperform by 3x
- Strong Sell recommendations underperform significantly
- Updated daily for ~6,000 stocks

**Strengths**:
- Sector-relative comparison is statistically sound
- Dynamic weighting based on backtesting
- Professional-grade factor model

**Weaknesses**:
- Black-box weighting methodology
- Requires analyst coverage (no microcaps)
- Premium feature

**Lessons for IC Score**:
- ✅ **Critical**: Sector-relative scoring is essential
- ✅ Dynamic weighting based on predictive power
- ✅ EPS revisions are highly predictive - add this factor
- ✅ Daily updates for timeliness

---

### 2.4 Morningstar Star Rating

**Approach**: Fair value estimation with uncertainty bands

| Component | Methodology |
|-----------|-------------|
| Fair Value | DCF model with analyst assumptions |
| Uncertainty | Historical volatility + business predictability |
| Economic Moat | Competitive advantage assessment (None/Narrow/Wide) |
| Star Rating | Price vs Fair Value adjusted for uncertainty |

**Rating Logic**:
```
5 Stars: Price significantly below fair value (high margin of safety)
4 Stars: Price moderately below fair value
3 Stars: Price approximately at fair value
2 Stars: Price moderately above fair value
1 Star: Price significantly above fair value
```

**Strengths**:
- Analyst-driven with qualitative moat assessment
- Uncertainty rating acknowledges model limitations
- Long track record and brand trust

**Weaknesses**:
- Subjective analyst inputs
- Slow to update (earnings cycles)
- Not available for all stocks

**Lessons for IC Score**:
- ✅ Incorporate fair value estimation
- ✅ Acknowledge data confidence/uncertainty
- ⚠️ Avoid pure analyst subjectivity

---

### 2.5 TipRanks Smart Score

**Approach**: 8-factor composite score (1-10)

| Factor | Description |
|--------|-------------|
| Analyst Consensus | Buy/Hold/Sell recommendations |
| Blogger Sentiment | Financial blogger opinions |
| Hedge Fund Activity | 13F filing analysis |
| Insider Trading | Form 4 analysis |
| Investor Sentiment | Crowd sentiment indicators |
| News Sentiment | NLP on news articles |
| Technical Analysis | Price patterns, indicators |
| Fundamentals | Key financial ratios |

**Strengths**:
- Comprehensive alternative data integration
- Unique blogger/crowd sentiment
- Strong insider/institutional signals

**Weaknesses**:
- Opaque factor weighting
- Some factors have questionable alpha

**Lessons for IC Score**:
- ✅ Alternative data (insider, institutional) adds value
- ✅ News sentiment is increasingly important
- ⚠️ Be selective about which alternative data has predictive power

---

### 2.6 Competitive Analysis Summary

| Feature | YCharts | Simply Wall St | Seeking Alpha | Morningstar | TipRanks | **IC Score (Proposed)** |
|---------|---------|----------------|---------------|-------------|----------|------------------------|
| Sector-Relative | Partial | ❌ | ✅ | ❌ | ❌ | ✅ |
| Lifecycle-Aware | ❌ | ❌ | ❌ | ✅ (via moat) | ❌ | ✅ |
| Alternative Data | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ |
| Transparency | ✅ | ✅ | Partial | Partial | ❌ | ✅ |
| Visual Design | Basic | ✅ | Basic | Basic | Basic | ✅ |
| Confidence Level | ❌ | ❌ | ❌ | ✅ | ❌ | ✅ |
| Free Access | ❌ | Partial | ❌ | Partial | Partial | ✅ |

---

## 3. Product Vision

### 3.1 Mission Statement

> Provide retail investors with institutional-quality stock analysis through a transparent, intelligent scoring system that adapts to each company's context.

### 3.2 Target Users

| Persona | Needs | IC Score Value |
|---------|-------|----------------|
| **Active Retail Investor** | Quick stock screening, validation of ideas | Single metric to filter + factor breakdown |
| **Long-Term Investor** | Quality companies at fair prices | Value + fundamentals + financial health |
| **Momentum Trader** | Trend identification | Momentum + technical + news sentiment |
| **Income Investor** | Stable dividend payers | Financial health + profitability |

### 3.3 Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Predictive Accuracy** | Top decile outperforms bottom by 2x | Backtest on historical data |
| **User Engagement** | 40% of users check IC Score | Analytics tracking |
| **Trust/Transparency** | <5% support tickets on methodology | Support ticket analysis |
| **Data Coverage** | 95% of S&P 500, 80% of Russell 2000 | Coverage dashboard |

---

## 4. Scoring Framework Design

### 4.1 Core Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        IC SCORE (0-100)                         │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │  QUALITY    │  │  VALUATION  │  │   SIGNALS   │             │
│  │  (40%)      │  │  (30%)      │  │   (30%)     │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│        │                │                │                      │
│   ┌────┴────┐      ┌────┴────┐      ┌────┴────┐                │
│   │         │      │         │      │         │                │
│   ▼         ▼      ▼         ▼      ▼         ▼                │
│ Growth  Profit  Value   Fair   Momentum  Smart                 │
│ (15%)   (13%)   (15%)   Value  (10%)     Money                 │
│                         (15%)            (12%)                  │
│ Health  Earnings                  Technical Sentiment          │
│ (12%)   Quality                   (8%)      (8%)               │
│         (Future)                                                │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Factor Categories

#### Category 1: Quality (40% weight)
Measures the fundamental quality of the business

| Factor | Weight | Focus |
|--------|--------|-------|
| **Growth** | 15% | Revenue, earnings, and margin expansion |
| **Profitability** | 13% | Margins, returns on capital, efficiency |
| **Financial Health** | 12% | Balance sheet strength, liquidity |
| **Earnings Quality** | Future | Accruals, cash conversion (Phase 2) |

#### Category 2: Valuation (30% weight)
Determines if the stock is fairly priced

| Factor | Weight | Focus |
|--------|--------|-------|
| **Relative Value** | 15% | P/E, P/S, P/B vs sector peers |
| **Intrinsic Value** | 15% | DCF-based fair value, margin of safety |

#### Category 3: Signals (30% weight)
Market and alternative data signals

| Factor | Weight | Focus |
|--------|--------|-------|
| **Smart Money** | 12% | Analyst + Insider + Institutional activity |
| **Momentum** | 10% | Price trends and relative strength |
| **Technical** | 8% | RSI, MACD, support/resistance |
| **Sentiment** | 8% | News sentiment, earnings revisions |

### 4.3 Sector-Relative Scoring

**Problem**: Absolute thresholds don't work across sectors
- Tech companies have high P/E ratios (growth premium)
- Utilities have low growth but high yields
- Banks have different balance sheet structures

**Solution**: Percentile-based scoring within sector

```python
def calculate_sector_percentile(ticker: str, metric: str, value: float) -> float:
    """
    Calculate where a stock's metric falls within its sector distribution.

    Returns: 0-100 percentile score
    """
    sector = get_sector(ticker)
    sector_values = get_sector_metric_values(sector, metric)

    # Handle metric direction (higher is better vs lower is better)
    if metric in LOWER_IS_BETTER:  # e.g., P/E, Debt/Equity
        percentile = 100 - percentileofscore(sector_values, value)
    else:  # e.g., ROE, Revenue Growth
        percentile = percentileofscore(sector_values, value)

    return percentile
```

**Sector Groups** (GICS-based):
1. Technology
2. Healthcare
3. Financial Services
4. Consumer Cyclical
5. Consumer Defensive
6. Industrials
7. Energy
8. Materials
9. Real Estate
10. Utilities
11. Communication Services

### 4.4 Company Lifecycle Awareness

**Problem**: Growth companies and mature companies have different characteristics

**Solution**: Classify companies and adjust factor weights

| Lifecycle Stage | Characteristics | Weight Adjustments |
|-----------------|-----------------|-------------------|
| **Hypergrowth** | Rev growth >50%, negative earnings | Growth ↑, Profitability ↓, Value ↓ |
| **Growth** | Rev growth 15-50%, low/no dividend | Growth ↑, Profitability neutral |
| **Mature** | Rev growth <15%, profitable, dividend | Profitability ↑, Value ↑ |
| **Turnaround** | Negative growth, improving metrics | Financial Health ↑, Momentum ↑ |
| **Declining** | Negative growth, deteriorating metrics | Financial Health ↑, Value neutral |

```python
def classify_lifecycle(ticker: str) -> str:
    """Classify company into lifecycle stage."""
    revenue_growth = get_revenue_growth(ticker)
    net_margin = get_net_margin(ticker)
    has_dividend = get_dividend_yield(ticker) > 0

    if revenue_growth > 50:
        return "hypergrowth"
    elif revenue_growth > 15:
        return "growth"
    elif revenue_growth > 0 and net_margin > 0:
        return "mature"
    elif revenue_growth < 0 and is_improving(ticker):
        return "turnaround"
    else:
        return "declining"

def get_lifecycle_weights(stage: str) -> Dict[str, float]:
    """Get adjusted factor weights based on lifecycle stage."""
    base_weights = {
        "growth": 0.15,
        "profitability": 0.13,
        "financial_health": 0.12,
        "value": 0.15,
        "fair_value": 0.15,
        "smart_money": 0.12,
        "momentum": 0.10,
        "technical": 0.08,
        "sentiment": 0.08
    }

    adjustments = LIFECYCLE_ADJUSTMENTS[stage]
    adjusted = {k: v * adjustments.get(k, 1.0) for k, v in base_weights.items()}

    # Normalize to sum to 1.0
    total = sum(adjusted.values())
    return {k: v / total for k, v in adjusted.items()}
```

### 4.5 Data Confidence & Score Validity

**Problem**: Incomplete data can lead to misleading scores

**Solution**: Multi-level confidence system

| Confidence Level | Data Completeness | Visual Indicator | User Message |
|------------------|-------------------|------------------|--------------|
| **High** | ≥90% factors available | Solid badge | Full score |
| **Medium** | 70-89% factors | Badge with indicator | "Based on available data" |
| **Low** | 50-69% factors | Muted badge | "Limited data available" |
| **Insufficient** | <50% factors | No score shown | "Insufficient data" |

**Core Factor Requirements**:
- At least 3 of 4 quality factors must have data
- At least 1 of 2 valuation factors must have data
- Score is not displayed if core requirements aren't met

---

## 5. Factor Definitions & Calculations

### 5.1 Growth Score (15% weight)

**Purpose**: Measure business growth trajectory

**Metrics & Weights**:
| Metric | Weight | Calculation | Benchmark |
|--------|--------|-------------|-----------|
| Revenue Growth (YoY) | 35% | (Rev_TTM - Rev_Prior) / Rev_Prior | Sector percentile |
| EPS Growth (YoY) | 30% | (EPS_TTM - EPS_Prior) / EPS_Prior | Sector percentile |
| Revenue Growth (3Y CAGR) | 20% | (Rev_Current / Rev_3YAgo)^(1/3) - 1 | Sector percentile |
| Operating Margin Expansion | 15% | OpMargin_TTM - OpMargin_Prior | Sector percentile |

**Score Calculation**:
```python
def calculate_growth_score(ticker: str) -> float:
    sector = get_sector(ticker)

    # Get metrics
    rev_growth_yoy = get_revenue_growth_yoy(ticker)
    eps_growth_yoy = get_eps_growth_yoy(ticker)
    rev_growth_3y = get_revenue_cagr_3y(ticker)
    margin_expansion = get_margin_expansion(ticker)

    # Convert to sector percentiles
    rev_pct = sector_percentile(sector, "revenue_growth_yoy", rev_growth_yoy)
    eps_pct = sector_percentile(sector, "eps_growth_yoy", eps_growth_yoy)
    rev_3y_pct = sector_percentile(sector, "revenue_cagr_3y", rev_growth_3y)
    margin_pct = sector_percentile(sector, "margin_expansion", margin_expansion)

    # Weighted average
    score = (
        rev_pct * 0.35 +
        eps_pct * 0.30 +
        rev_3y_pct * 0.20 +
        margin_pct * 0.15
    )

    return score
```

**Edge Cases**:
- Negative to positive EPS: Cap at 100 percentile
- Extreme outliers (>3 std dev): Winsorize to 3 std dev
- Missing 3Y data: Redistribute weight to YoY metrics

---

### 5.2 Profitability Score (13% weight)

**Purpose**: Measure operational efficiency and returns

**Metrics & Weights**:
| Metric | Weight | Calculation | Sector Adjustment |
|--------|--------|-------------|-------------------|
| Net Profit Margin | 25% | Net Income / Revenue | Yes |
| Return on Equity (ROE) | 25% | Net Income / Shareholders' Equity | Yes |
| Return on Invested Capital (ROIC) | 25% | NOPAT / Invested Capital | Yes |
| Gross Margin | 15% | Gross Profit / Revenue | Yes |
| Operating Margin | 10% | Operating Income / Revenue | Yes |

**Score Calculation**:
```python
def calculate_profitability_score(ticker: str) -> float:
    sector = get_sector(ticker)

    metrics = {
        "net_margin": (get_net_margin(ticker), 0.25),
        "roe": (get_roe(ticker), 0.25),
        "roic": (get_roic(ticker), 0.25),
        "gross_margin": (get_gross_margin(ticker), 0.15),
        "operating_margin": (get_operating_margin(ticker), 0.10)
    }

    score = 0
    total_weight = 0

    for metric_name, (value, weight) in metrics.items():
        if value is not None:
            pct = sector_percentile(sector, metric_name, value)
            score += pct * weight
            total_weight += weight

    return score / total_weight if total_weight > 0 else None
```

---

### 5.3 Financial Health Score (12% weight)

**Purpose**: Assess balance sheet strength and risk

**Metrics & Weights**:
| Metric | Weight | Ideal Range | Direction |
|--------|--------|-------------|-----------|
| Debt-to-Equity | 25% | 0.3-1.0 (sector-dependent) | Lower is better |
| Current Ratio | 20% | 1.5-2.5 | Optimal range |
| Interest Coverage | 20% | >5x | Higher is better |
| Free Cash Flow Yield | 20% | >5% | Higher is better |
| Altman Z-Score | 15% | >3.0 | Higher is better |

**Score Calculation**:
```python
def calculate_financial_health_score(ticker: str) -> float:
    sector = get_sector(ticker)

    # Debt-to-Equity (lower is better)
    de_ratio = get_debt_to_equity(ticker)
    de_score = 100 - sector_percentile(sector, "debt_to_equity", de_ratio)

    # Current Ratio (optimal range)
    current_ratio = get_current_ratio(ticker)
    cr_score = optimal_range_score(current_ratio, optimal=2.0, min=1.0, max=3.5)

    # Interest Coverage (higher is better, with cap)
    interest_coverage = get_interest_coverage(ticker)
    ic_score = min(100, sector_percentile(sector, "interest_coverage", interest_coverage))

    # FCF Yield (higher is better)
    fcf_yield = get_fcf_yield(ticker)
    fcf_score = sector_percentile(sector, "fcf_yield", fcf_yield)

    # Altman Z-Score
    z_score = calculate_altman_z(ticker)
    z_score_pct = z_score_to_percentile(z_score)

    return (
        de_score * 0.25 +
        cr_score * 0.20 +
        ic_score * 0.20 +
        fcf_score * 0.20 +
        z_score_pct * 0.15
    )

def optimal_range_score(value: float, optimal: float, min: float, max: float) -> float:
    """Score that peaks at optimal and decreases towards min/max."""
    if value < min:
        return max(0, 50 * (value / min))
    elif value <= optimal:
        return 50 + 50 * ((value - min) / (optimal - min))
    elif value <= max:
        return 100 - 50 * ((value - optimal) / (max - optimal))
    else:
        return max(0, 50 * (1 - (value - max) / max))
```

---

### 5.4 Relative Value Score (15% weight)

**Purpose**: Compare current valuation to sector peers

**Metrics & Weights**:
| Metric | Weight | Direction | Notes |
|--------|--------|-----------|-------|
| P/E Ratio (TTM) | 30% | Lower is better | Use forward P/E if available |
| P/S Ratio (TTM) | 25% | Lower is better | Good for unprofitable companies |
| EV/EBITDA | 25% | Lower is better | Better for comparing across capital structures |
| P/B Ratio | 10% | Lower is better | Important for financials |
| PEG Ratio | 10% | Lower is better | Adjusts P/E for growth |

**Score Calculation**:
```python
def calculate_value_score(ticker: str) -> float:
    sector = get_sector(ticker)

    # All value metrics: lower is better (invert percentile)
    metrics = {
        "pe_ratio": (get_pe_ratio(ticker), 0.30),
        "ps_ratio": (get_ps_ratio(ticker), 0.25),
        "ev_ebitda": (get_ev_ebitda(ticker), 0.25),
        "pb_ratio": (get_pb_ratio(ticker), 0.10),
        "peg_ratio": (get_peg_ratio(ticker), 0.10)
    }

    score = 0
    total_weight = 0

    for metric_name, (value, weight) in metrics.items():
        if value is not None and value > 0:  # Negative P/E needs special handling
            # Lower percentile = cheaper = higher score
            pct = 100 - sector_percentile(sector, metric_name, value)
            score += pct * weight
            total_weight += weight

    return score / total_weight if total_weight > 0 else None
```

**Special Cases**:
- Negative P/E: Exclude from calculation, redistribute weight
- Extreme outliers (P/E > 100): Cap at sector 99th percentile

---

### 5.5 Intrinsic Value Score (15% weight)

**Purpose**: Estimate fair value using DCF and compare to current price

**Methodology**:
```python
def calculate_intrinsic_value_score(ticker: str) -> float:
    current_price = get_current_price(ticker)

    # Get fair value estimates from multiple sources
    dcf_value = calculate_dcf_fair_value(ticker)
    analyst_target = get_analyst_price_target(ticker)

    # Combine estimates (weighted by confidence)
    fair_value = weighted_fair_value(dcf_value, analyst_target)

    if fair_value is None:
        return None

    # Calculate margin of safety
    margin_of_safety = (fair_value - current_price) / fair_value

    # Convert to 0-100 score
    # -50% overvalued -> 0
    # 0% fairly valued -> 50
    # +50% undervalued -> 100
    score = 50 + (margin_of_safety * 100)

    return max(0, min(100, score))

def calculate_dcf_fair_value(ticker: str) -> Optional[float]:
    """
    Simplified DCF calculation using:
    - 5-year projected FCF growth
    - Terminal growth rate (2-3%)
    - Discount rate (WACC or 10% default)
    """
    fcf = get_free_cash_flow(ticker)
    growth_rate = estimate_fcf_growth_rate(ticker)
    terminal_growth = 0.025  # 2.5%
    discount_rate = get_wacc(ticker) or 0.10
    shares = get_shares_outstanding(ticker)

    # Project 5 years of FCF
    projected_fcf = [fcf * (1 + growth_rate) ** i for i in range(1, 6)]

    # Terminal value
    terminal_value = projected_fcf[-1] * (1 + terminal_growth) / (discount_rate - terminal_growth)

    # Discount to present
    pv_fcf = sum(cf / (1 + discount_rate) ** i for i, cf in enumerate(projected_fcf, 1))
    pv_terminal = terminal_value / (1 + discount_rate) ** 5

    enterprise_value = pv_fcf + pv_terminal
    equity_value = enterprise_value - get_net_debt(ticker)

    return equity_value / shares if shares > 0 else None
```

---

### 5.6 Smart Money Score (12% weight)

**Purpose**: Track what professional investors are doing

**Components**:
| Signal | Weight | Data Source | Calculation |
|--------|--------|-------------|-------------|
| Analyst Consensus | 40% | analyst_ratings | Buy/Hold/Sell ratio |
| Analyst Revisions | 20% | analyst_ratings | Recent upgrades vs downgrades |
| Insider Activity | 25% | insider_trades | Net buying over 90 days |
| Institutional Flow | 15% | institutional_holdings | Change in institutional ownership |

**Score Calculation**:
```python
def calculate_smart_money_score(ticker: str) -> float:
    # Analyst Consensus (40%)
    ratings = get_analyst_ratings(ticker, days=90)
    if ratings:
        buy_pct = ratings['buy'] / ratings['total']
        hold_pct = ratings['hold'] / ratings['total']
        sell_pct = ratings['sell'] / ratings['total']
        consensus_score = (buy_pct * 100 + hold_pct * 50 + sell_pct * 0)
    else:
        consensus_score = None

    # Analyst Revisions (20%)
    revisions = get_analyst_revisions(ticker, days=90)
    if revisions:
        net_revisions = revisions['upgrades'] - revisions['downgrades']
        revision_score = 50 + (net_revisions / max(revisions['total'], 1)) * 50
    else:
        revision_score = None

    # Insider Activity (25%)
    insider = get_insider_trades(ticker, days=90)
    if insider:
        net_value = insider['buy_value'] - insider['sell_value']
        market_cap = get_market_cap(ticker)
        insider_ratio = net_value / market_cap
        # Scale: -0.1% = 0, 0 = 50, +0.1% = 100
        insider_score = 50 + (insider_ratio * 100 / 0.001) * 50
        insider_score = max(0, min(100, insider_score))
    else:
        insider_score = None

    # Institutional Flow (15%)
    inst = get_institutional_changes(ticker)
    if inst:
        ownership_change = inst['current_pct'] - inst['prior_pct']
        # Scale: -5% = 0, 0 = 50, +5% = 100
        inst_score = 50 + (ownership_change / 5) * 50
        inst_score = max(0, min(100, inst_score))
    else:
        inst_score = None

    # Weighted average with available data
    components = [
        (consensus_score, 0.40),
        (revision_score, 0.20),
        (insider_score, 0.25),
        (inst_score, 0.15)
    ]

    return weighted_average_with_redistribution(components)
```

---

### 5.7 Momentum Score (10% weight)

**Purpose**: Identify price trends and relative strength

**Metrics**:
| Metric | Weight | Calculation |
|--------|--------|-------------|
| 1-Month Return | 20% | Sector-relative percentile |
| 3-Month Return | 30% | Sector-relative percentile |
| 6-Month Return | 30% | Sector-relative percentile |
| 12-Month Return | 20% | Sector-relative percentile |

**Score Calculation**:
```python
def calculate_momentum_score(ticker: str) -> float:
    sector = get_sector(ticker)

    returns = {
        "1m": (get_return(ticker, months=1), 0.20),
        "3m": (get_return(ticker, months=3), 0.30),
        "6m": (get_return(ticker, months=6), 0.30),
        "12m": (get_return(ticker, months=12), 0.20)
    }

    score = 0
    total_weight = 0

    for period, (ret, weight) in returns.items():
        if ret is not None:
            pct = sector_percentile(sector, f"return_{period}", ret)
            score += pct * weight
            total_weight += weight

    return score / total_weight if total_weight > 0 else None
```

---

### 5.8 Technical Score (8% weight)

**Purpose**: Technical analysis signals

**Indicators**:
| Indicator | Weight | Signal |
|-----------|--------|--------|
| RSI (14) | 30% | Oversold (<30) = bullish, Overbought (>70) = bearish |
| MACD | 25% | Histogram positive = bullish |
| Price vs 50-day SMA | 25% | Above = bullish |
| Price vs 200-day SMA | 20% | Above = bullish |

**Score Calculation**:
```python
def calculate_technical_score(ticker: str) -> float:
    # RSI Score (30%)
    rsi = get_rsi(ticker)
    if rsi <= 30:
        rsi_score = 70 + (30 - rsi) / 30 * 30  # 70-100 for oversold
    elif rsi >= 70:
        rsi_score = 30 - (rsi - 70) / 30 * 30  # 0-30 for overbought
    else:
        rsi_score = 50 + (50 - rsi) / 20 * 20  # 30-70 for neutral zone

    # MACD Score (25%)
    macd_hist = get_macd_histogram(ticker)
    macd_score = 50 + min(50, max(-50, macd_hist * 10))

    # SMA Scores
    price = get_current_price(ticker)
    sma50 = get_sma(ticker, 50)
    sma200 = get_sma(ticker, 200)

    sma50_score = 70 if price > sma50 else 30
    sma200_score = 70 if price > sma200 else 30

    return (
        rsi_score * 0.30 +
        macd_score * 0.25 +
        sma50_score * 0.25 +
        sma200_score * 0.20
    )
```

---

### 5.9 Sentiment Score (8% weight)

**Purpose**: Market sentiment from news and earnings revisions

**Components**:
| Component | Weight | Source |
|-----------|--------|--------|
| News Sentiment | 50% | AI-analyzed news articles (30 days) |
| Earnings Revisions | 50% | Analyst EPS estimate changes |

**Score Calculation**:
```python
def calculate_sentiment_score(ticker: str) -> float:
    # News Sentiment (50%)
    news = get_news_articles(ticker, days=30)
    if news:
        avg_sentiment = sum(a['sentiment'] for a in news) / len(news)
        # Sentiment is -100 to +100, convert to 0-100
        news_score = (avg_sentiment + 100) / 2
    else:
        news_score = None

    # Earnings Revisions (50%)
    revisions = get_eps_revisions(ticker, days=90)
    if revisions:
        # Net % change in consensus EPS
        eps_change_pct = (revisions['current'] - revisions['prior']) / abs(revisions['prior'])
        # Scale: -20% = 0, 0 = 50, +20% = 100
        revision_score = 50 + (eps_change_pct / 0.20) * 50
        revision_score = max(0, min(100, revision_score))
    else:
        revision_score = None

    return weighted_average_with_redistribution([
        (news_score, 0.50),
        (revision_score, 0.50)
    ])
```

---

## 6. Technical Architecture

### 6.1 System Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              FRONTEND                                    │
│                         (Next.js + React)                                │
├─────────────────────────────────────────────────────────────────────────┤
│  Pages:                      │  Components:                              │
│  /stock/[ticker]             │  ICScoreCard                             │
│  /screener                   │  ICScoreGauge                            │
│  /portfolio                  │  FactorBreakdown                         │
│                              │  ICScoreHistory                          │
│                              │  ICScoreExplainer                        │
└──────────────────────────────┴──────────────────────────────────────────┘
                                       │
                                       │ REST API
                                       ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                              BACKEND API                                 │
│                            (Go + Chi Router)                             │
├─────────────────────────────────────────────────────────────────────────┤
│  Endpoints:                                                              │
│  GET  /api/v1/stocks/{ticker}/ic-score                                  │
│  GET  /api/v1/stocks/{ticker}/ic-score/history                          │
│  GET  /api/v1/stocks/{ticker}/ic-score/factors/{factor}                 │
│  GET  /api/v1/ic-scores (paginated list)                                │
│  GET  /api/v1/ic-scores/sector/{sector}                                 │
│  GET  /api/v1/ic-scores/top                                             │
│  GET  /api/v1/ic-scores/bottom                                          │
│                                                                          │
│  Services:                                                               │
│  ICScoreService - business logic, caching                               │
│  SectorService - sector percentile calculations                         │
└─────────────────────────────────────────────────────────────────────────┘
                                       │
                                       │ Internal API / Direct DB
                                       ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         IC SCORE SERVICE                                 │
│                       (Python + FastAPI)                                 │
├─────────────────────────────────────────────────────────────────────────┤
│  Pipelines:                                                              │
│  ICScoreCalculator                                                       │
│  ├── DataFetcher (aggregates from all tables)                           │
│  ├── SectorPercentileCalculator                                         │
│  ├── LifecycleClassifier                                                │
│  ├── FactorCalculator (one per factor)                                  │
│  ├── WeightAdjuster (lifecycle-aware)                                   │
│  └── ScoreAggregator                                                    │
│                                                                          │
│  Schedulers:                                                             │
│  - Daily full recalculation (market close + 2 hours)                    │
│  - Hourly price-sensitive updates (momentum, technical)                 │
│  - Event-driven updates (earnings, filings)                             │
└─────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           DATABASE LAYER                                 │
│                    (PostgreSQL + TimescaleDB)                            │
├─────────────────────────────────────────────────────────────────────────┤
│  Core Tables:                │  IC Score Tables:                         │
│  stocks                      │  ic_scores (main score table)             │
│  financials                  │  ic_score_factors (factor details)        │
│  stock_prices (hypertable)   │  ic_score_history (time series)           │
│  technical_indicators        │  sector_percentiles (precomputed)         │
│  analyst_ratings             │                                           │
│  insider_trades              │  Materialized Views:                      │
│  institutional_holdings      │  mv_sector_metrics (daily refresh)        │
│  news_articles               │  mv_ic_score_rankings                     │
│  valuation_ratios            │                                           │
└──────────────────────────────┴──────────────────────────────────────────┘
```

### 6.2 Database Schema

#### ic_scores (Main Table)
```sql
CREATE TABLE ic_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Overall Score
    overall_score NUMERIC(5,2) NOT NULL,
    rating VARCHAR(20) NOT NULL, -- 'Strong Buy', 'Buy', 'Hold', 'Sell', 'Strong Sell'
    rating_change VARCHAR(10), -- 'upgrade', 'downgrade', 'unchanged'

    -- Category Scores (aggregated)
    quality_score NUMERIC(5,2),
    valuation_score NUMERIC(5,2),
    signals_score NUMERIC(5,2),

    -- Individual Factor Scores
    growth_score NUMERIC(5,2),
    profitability_score NUMERIC(5,2),
    financial_health_score NUMERIC(5,2),
    value_score NUMERIC(5,2),
    intrinsic_value_score NUMERIC(5,2),
    smart_money_score NUMERIC(5,2),
    momentum_score NUMERIC(5,2),
    technical_score NUMERIC(5,2),
    sentiment_score NUMERIC(5,2),

    -- Metadata
    lifecycle_stage VARCHAR(20), -- 'hypergrowth', 'growth', 'mature', 'turnaround', 'declining'
    sector VARCHAR(50),
    sector_rank INTEGER,
    sector_percentile NUMERIC(5,2),
    data_completeness NUMERIC(5,2),
    confidence_level VARCHAR(10), -- 'high', 'medium', 'low'

    -- Factor weights used (may differ from default due to lifecycle)
    weights_used JSONB,

    -- Calculation details for debugging/transparency
    calculation_metadata JSONB,

    UNIQUE (ticker, calculated_at::date)
);

CREATE INDEX idx_ic_scores_ticker ON ic_scores(ticker);
CREATE INDEX idx_ic_scores_calculated_at ON ic_scores(calculated_at DESC);
CREATE INDEX idx_ic_scores_sector ON ic_scores(sector);
CREATE INDEX idx_ic_scores_overall ON ic_scores(overall_score DESC);
```

#### ic_score_factor_details (Factor Breakdown)
```sql
CREATE TABLE ic_score_factor_details (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ic_score_id UUID REFERENCES ic_scores(id) ON DELETE CASCADE,
    factor_name VARCHAR(30) NOT NULL,

    -- Scoring
    score NUMERIC(5,2),
    grade VARCHAR(3), -- 'A+', 'A', 'B+', 'B', 'C+', 'C', 'D', 'F'
    weight_applied NUMERIC(5,4),

    -- Component Metrics
    metrics JSONB, -- { "pe_ratio": 15.2, "pe_percentile": 72, ... }

    -- Comparisons
    sector_average NUMERIC(10,2),
    sector_percentile NUMERIC(5,2),

    -- Data Quality
    data_available BOOLEAN DEFAULT TRUE,
    data_freshness_days INTEGER,

    UNIQUE (ic_score_id, factor_name)
);
```

#### sector_percentiles (Precomputed)
```sql
CREATE TABLE sector_percentiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sector VARCHAR(50) NOT NULL,
    metric_name VARCHAR(50) NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Distribution statistics
    min_value NUMERIC,
    p10_value NUMERIC,
    p25_value NUMERIC,
    p50_value NUMERIC, -- median
    p75_value NUMERIC,
    p90_value NUMERIC,
    max_value NUMERIC,
    mean_value NUMERIC,
    std_dev NUMERIC,
    sample_count INTEGER,

    UNIQUE (sector, metric_name, calculated_at::date)
);

CREATE INDEX idx_sector_percentiles_lookup ON sector_percentiles(sector, metric_name, calculated_at DESC);
```

### 6.3 Caching Strategy

| Cache Layer | TTL | Purpose |
|-------------|-----|---------|
| **CDN (CloudFlare)** | 5 min | Static API responses |
| **Redis** | 1 hour | Frequently accessed scores |
| **PostgreSQL Materialized Views** | Daily | Sector percentiles, rankings |

```python
# Cache key patterns
CACHE_KEYS = {
    "ic_score": "ic:score:{ticker}",  # TTL: 1 hour
    "ic_score_list": "ic:list:{page}:{sort}",  # TTL: 5 min
    "sector_percentiles": "ic:sector:{sector}:{metric}",  # TTL: 24 hours
    "factor_details": "ic:factors:{ticker}",  # TTL: 1 hour
}
```

---

## 7. Data Pipeline Specification

### 7.1 Pipeline Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    DAILY CALCULATION PIPELINE                    │
│                  (Runs at 6:00 PM ET, post-market)              │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│ Sector Stats  │    │  Data Fetch   │    │  API Updates  │
│  Calculator   │    │   Pipeline    │    │  (FMP, etc)   │
└───────┬───────┘    └───────┬───────┘    └───────┬───────┘
        │                    │                    │
        └─────────────┬──────┴────────────────────┘
                      ▼
            ┌─────────────────┐
            │  IC Score Calc  │
            │   (per stock)   │
            └────────┬────────┘
                     │
        ┌────────────┼────────────┐
        ▼            ▼            ▼
   ┌─────────┐  ┌─────────┐  ┌─────────┐
   │ Quality │  │Valuation│  │ Signals │
   │ Factors │  │ Factors │  │ Factors │
   └────┬────┘  └────┬────┘  └────┬────┘
        │            │            │
        └────────────┼────────────┘
                     ▼
            ┌─────────────────┐
            │  Aggregation &  │
            │  Weight Adjust  │
            └────────┬────────┘
                     │
                     ▼
            ┌─────────────────┐
            │   Store Score   │
            │  + Invalidate   │
            │     Cache       │
            └─────────────────┘
```

### 7.2 Pipeline Stages

#### Stage 1: Sector Statistics (Pre-calculation)
```python
class SectorStatsCalculator:
    """
    Calculate sector-level statistics for percentile scoring.
    Runs before individual stock calculations.
    """

    METRICS = [
        "pe_ratio", "ps_ratio", "pb_ratio", "ev_ebitda", "peg_ratio",
        "revenue_growth_yoy", "eps_growth_yoy", "revenue_cagr_3y",
        "net_margin", "roe", "roic", "gross_margin", "operating_margin",
        "debt_to_equity", "current_ratio", "interest_coverage", "fcf_yield",
        "return_1m", "return_3m", "return_6m", "return_12m"
    ]

    def calculate_for_sector(self, sector: str):
        for metric in self.METRICS:
            values = self.get_metric_values(sector, metric)
            stats = self.calculate_distribution_stats(values)
            self.store_sector_percentiles(sector, metric, stats)

    def calculate_distribution_stats(self, values: List[float]) -> Dict:
        clean_values = self.remove_outliers(values)  # Winsorize at 3 std dev
        return {
            "min": np.min(clean_values),
            "p10": np.percentile(clean_values, 10),
            "p25": np.percentile(clean_values, 25),
            "p50": np.percentile(clean_values, 50),
            "p75": np.percentile(clean_values, 75),
            "p90": np.percentile(clean_values, 90),
            "max": np.max(clean_values),
            "mean": np.mean(clean_values),
            "std_dev": np.std(clean_values),
            "count": len(clean_values)
        }
```

#### Stage 2: Individual Stock Calculation
```python
class ICScoreCalculator:
    """Main IC Score calculation pipeline."""

    def calculate_score(self, ticker: str) -> ICScore:
        # 1. Fetch all required data
        data = self.data_fetcher.fetch_all(ticker)

        # 2. Classify lifecycle stage
        lifecycle = self.lifecycle_classifier.classify(data)

        # 3. Get adjusted weights
        weights = self.weight_adjuster.get_weights(lifecycle)

        # 4. Calculate each factor
        factors = {}
        for factor_name, calculator in self.factor_calculators.items():
            try:
                factors[factor_name] = calculator.calculate(ticker, data)
            except DataNotAvailableError:
                factors[factor_name] = None

        # 5. Aggregate to overall score
        overall = self.aggregator.aggregate(factors, weights)

        # 6. Determine rating
        rating = self.get_rating(overall)

        # 7. Calculate metadata
        completeness = self.calculate_completeness(factors)
        confidence = self.get_confidence_level(completeness)

        return ICScore(
            ticker=ticker,
            overall_score=overall,
            rating=rating,
            factors=factors,
            weights_used=weights,
            lifecycle_stage=lifecycle,
            data_completeness=completeness,
            confidence_level=confidence
        )
```

### 7.3 Scheduling

| Job | Schedule | Description |
|-----|----------|-------------|
| **Full Recalculation** | Daily 6:00 PM ET | Complete recalc for all stocks |
| **Sector Stats Refresh** | Daily 5:30 PM ET | Recalculate sector percentiles |
| **Price-Sensitive Update** | Hourly 9:30 AM - 4:00 PM | Update momentum, technical only |
| **Event-Driven Update** | On trigger | Earnings, filings, analyst changes |

---

## 8. API Specification

### 8.1 Endpoints

#### GET /api/v1/stocks/{ticker}/ic-score

Returns the latest IC Score for a stock.

**Response**:
```json
{
  "ticker": "AAPL",
  "calculated_at": "2026-01-31T18:00:00Z",
  "overall_score": 78.5,
  "rating": "Buy",
  "rating_change": "unchanged",
  "confidence_level": "high",
  "data_completeness": 95.0,
  "lifecycle_stage": "mature",
  "sector": "Technology",
  "sector_rank": 12,
  "sector_percentile": 88.5,

  "categories": {
    "quality": {
      "score": 82.3,
      "grade": "A"
    },
    "valuation": {
      "score": 71.2,
      "grade": "B+"
    },
    "signals": {
      "score": 79.8,
      "grade": "B+"
    }
  },

  "factors": [
    {
      "name": "growth",
      "display_name": "Growth",
      "score": 85.2,
      "grade": "A",
      "weight": 0.15,
      "category": "quality",
      "description": "Revenue and earnings growth trajectory",
      "data_available": true,
      "metrics": {
        "revenue_growth_yoy": { "value": 8.5, "percentile": 72 },
        "eps_growth_yoy": { "value": 12.3, "percentile": 81 },
        "revenue_cagr_3y": { "value": 11.2, "percentile": 76 }
      }
    },
    // ... other factors
  ],

  "score_change_30d": 2.3,
  "next_update": "2026-02-01T18:00:00Z"
}
```

#### GET /api/v1/stocks/{ticker}/ic-score/history

Returns historical IC Scores.

**Query Params**:
- `days`: Number of days (default: 90, max: 365)
- `interval`: Aggregation interval (daily, weekly, monthly)

**Response**:
```json
{
  "ticker": "AAPL",
  "period": "90d",
  "interval": "daily",
  "data": [
    {
      "date": "2026-01-31",
      "overall_score": 78.5,
      "rating": "Buy",
      "quality_score": 82.3,
      "valuation_score": 71.2,
      "signals_score": 79.8
    },
    // ... more data points
  ],
  "stats": {
    "min": 72.1,
    "max": 81.3,
    "avg": 76.8,
    "trend": "stable"
  }
}
```

#### GET /api/v1/ic-scores

Returns paginated list of all IC Scores.

**Query Params**:
- `limit`: Results per page (default: 20, max: 100)
- `offset`: Pagination offset
- `sector`: Filter by sector
- `min_score`: Minimum overall score
- `max_score`: Maximum overall score
- `rating`: Filter by rating
- `sort`: Sort field (overall_score, sector_percentile, etc.)
- `order`: asc or desc

#### GET /api/v1/ic-scores/top

Returns top-rated stocks.

**Query Params**:
- `limit`: Number of results (default: 10)
- `sector`: Optional sector filter

#### GET /api/v1/ic-scores/sector/{sector}

Returns sector-specific rankings and statistics.

---

## 9. UI/UX Design

### 9.1 Main Score Display (ICScoreCard)

```
┌─────────────────────────────────────────────────────────────────┐
│                        IC SCORE                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│            ┌──────────────────────┐                             │
│            │                      │                             │
│            │         78           │   ← Circular Gauge          │
│            │        ───           │     with color gradient     │
│            │        100           │                             │
│            └──────────────────────┘                             │
│                                                                  │
│                    ┌─────────┐                                  │
│                    │   BUY   │     ← Rating Badge               │
│                    └─────────┘                                  │
│                                                                  │
│           ▲ +2.3 pts (30d)         ← Trend Indicator            │
│                                                                  │
│  Confidence: High  •  Updated: 2h ago                           │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│  CATEGORY BREAKDOWN                                              │
│  ┌────────────────┬────────────────┬────────────────┐           │
│  │   QUALITY      │   VALUATION    │    SIGNALS     │           │
│  │     82.3       │     71.2       │     79.8       │           │
│  │      A         │      B+        │      B+        │           │
│  └────────────────┴────────────────┴────────────────┘           │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│  FACTOR DETAILS                          [Expand All ▼]         │
│                                                                  │
│  Quality (40%)                                                   │
│  ├─ Growth         85.2  A   ████████░░  15%                    │
│  ├─ Profitability  81.5  A-  ████████░░  13%                    │
│  └─ Financial      79.8  B+  ███████░░░  12%                    │
│                                                                  │
│  Valuation (30%)                                                 │
│  ├─ Relative Value 68.3  B   ██████░░░░  15%                    │
│  └─ Intrinsic      74.1  B+  ███████░░░  15%                    │
│                                                                  │
│  Signals (30%)                                                   │
│  ├─ Smart Money    82.4  A   ████████░░  12%                    │
│  ├─ Momentum       78.5  B+  ███████░░░  10%                    │
│  ├─ Technical      71.2  B   ██████░░░░   8%                    │
│  └─ Sentiment      85.0  A   ████████░░   8%                    │
│                                                                  │
├─────────────────────────────────────────────────────────────────┤
│  [📊 View History]  [📖 How It's Calculated]  [⚖️ Compare]      │
└─────────────────────────────────────────────────────────────────┘
```

### 9.2 Color Scheme

| Score Range | Rating | Primary Color | Background |
|-------------|--------|---------------|------------|
| 80-100 | Strong Buy | #10b981 (green-500) | #ecfdf5 |
| 65-79 | Buy | #84cc16 (lime-500) | #f7fee7 |
| 50-64 | Hold | #eab308 (yellow-500) | #fefce8 |
| 35-49 | Sell | #f97316 (orange-500) | #fff7ed |
| 0-34 | Strong Sell | #ef4444 (red-500) | #fef2f2 |

### 9.3 Grade Mapping

| Score Range | Grade |
|-------------|-------|
| 95-100 | A+ |
| 90-94 | A |
| 85-89 | A- |
| 80-84 | B+ |
| 75-79 | B |
| 70-74 | B- |
| 65-69 | C+ |
| 60-64 | C |
| 55-59 | C- |
| 50-54 | D+ |
| 45-49 | D |
| 40-44 | D- |
| 0-39 | F |

### 9.4 Factor Expansion (On Click)

```
┌─────────────────────────────────────────────────────────────────┐
│  ▼ Growth         85.2  A   ████████░░  15%                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  What this measures:                                             │
│  Revenue and earnings growth compared to sector peers            │
│                                                                  │
│  Metrics Used:                                                   │
│  ┌────────────────────┬─────────┬───────────┬──────────────┐    │
│  │ Metric             │ Value   │ Sector Avg│ Percentile   │    │
│  ├────────────────────┼─────────┼───────────┼──────────────┤    │
│  │ Revenue Growth YoY │ 8.5%    │ 5.2%      │ 72nd         │    │
│  │ EPS Growth YoY     │ 12.3%   │ 7.1%      │ 81st         │    │
│  │ 3Y Revenue CAGR    │ 11.2%   │ 8.4%      │ 76th         │    │
│  │ Margin Expansion   │ +1.2%   │ +0.3%     │ 85th         │    │
│  └────────────────────┴─────────┴───────────┴──────────────┘    │
│                                                                  │
│  Data freshness: Updated 2 hours ago (Q4 2025 earnings)         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 9.5 Score History Chart

```
┌─────────────────────────────────────────────────────────────────┐
│  IC SCORE HISTORY (90 Days)                                      │
│                                                                  │
│  82 ┤                                     ╭──╮                   │
│  80 ┤                               ╭────╯  ╰─╮                  │
│  78 ┤      ╭──╮              ╭─────╯          ╰──● 78.5          │
│  76 ┤ ╭───╯  ╰──╮   ╭──────╯                                    │
│  74 ┤╯          ╰──╯                                             │
│  72 ┤                                                            │
│  70 ┼────┬────┬────┬────┬────┬────┬────┬────┬────┬────┬────     │
│     Nov  Dec  Jan                                                │
│                                                                  │
│  ● Overall  ─── Quality  --- Valuation  ··· Signals              │
│                                                                  │
│  Key Events:                                                     │
│  📊 Nov 2: Q4 Earnings Beat (+3.2 pts)                          │
│  📉 Dec 15: Analyst Downgrade (-1.8 pts)                        │
│  📈 Jan 10: Insider Buying Surge (+2.1 pts)                     │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 10. Implementation Roadmap

### Phase 1: Foundation (Weeks 1-3)

| Task | Priority | Owner | Status |
|------|----------|-------|--------|
| Implement sector percentile infrastructure | P0 | Backend | Not Started |
| Add sector_percentiles table and refresh job | P0 | Backend | Not Started |
| Refactor factor calculations to use percentiles | P0 | IC Score Service | Not Started |
| Add lifecycle classification | P1 | IC Score Service | Not Started |
| Update API responses with new fields | P1 | Backend | Not Started |

**Deliverables**:
- Sector-relative scoring for all factors
- Lifecycle classification working
- Updated API with sector rank/percentile

### Phase 2: Factor Improvements (Weeks 4-6)

| Task | Priority | Owner | Status |
|------|----------|-------|--------|
| Implement Intrinsic Value (DCF) factor | P0 | IC Score Service | Not Started |
| Add EPS revisions to Sentiment factor | P0 | IC Score Service | Not Started |
| Consolidate Analyst + Insider + Institutional → Smart Money | P1 | IC Score Service | Not Started |
| Add factor metric breakdowns to API | P1 | Backend | Not Started |
| Implement dynamic weight adjustment | P2 | IC Score Service | Not Started |

**Deliverables**:
- 9 refined factors (from 10)
- Full metric transparency in API
- Lifecycle-aware weight adjustment

### Phase 3: UI Enhancement (Weeks 7-9)

| Task | Priority | Owner | Status |
|------|----------|-------|--------|
| Redesign ICScoreCard with category grouping | P0 | Frontend | Not Started |
| Add factor expansion with metric details | P0 | Frontend | Not Started |
| Implement score history chart | P1 | Frontend | Not Started |
| Add sector comparison view | P1 | Frontend | Not Started |
| Create IC Score explainer modal | P2 | Frontend | Not Started |

**Deliverables**:
- New ICScoreCard component
- Interactive factor breakdown
- Historical trend visualization

### Phase 4: Validation & Polish (Weeks 10-12)

| Task | Priority | Owner | Status |
|------|----------|-------|--------|
| Backtest scoring methodology | P0 | Data Science | Not Started |
| Performance optimization | P1 | Backend | Not Started |
| Caching strategy implementation | P1 | Backend | Not Started |
| Documentation and help content | P2 | Product | Not Started |
| Beta testing with select users | P2 | Product | Not Started |

**Deliverables**:
- Validated scoring model
- Sub-100ms API response times
- User-facing documentation

---

## 11. Appendix

### A. Metric Definitions

| Metric | Formula | Data Source |
|--------|---------|-------------|
| P/E Ratio | Market Cap / Net Income TTM | valuation_ratios |
| P/S Ratio | Market Cap / Revenue TTM | valuation_ratios |
| P/B Ratio | Market Cap / Book Value | valuation_ratios |
| EV/EBITDA | Enterprise Value / EBITDA TTM | valuation_ratios |
| ROE | Net Income / Shareholders Equity | financials |
| ROIC | NOPAT / Invested Capital | calculated |
| FCF Yield | Free Cash Flow / Market Cap | financials |
| Revenue Growth YoY | (Rev TTM - Rev Prior) / Rev Prior | financials |

### B. Sector Classification

Using GICS (Global Industry Classification Standard):
- 10: Energy
- 15: Materials
- 20: Industrials
- 25: Consumer Discretionary
- 30: Consumer Staples
- 35: Health Care
- 40: Financials
- 45: Information Technology
- 50: Communication Services
- 55: Utilities
- 60: Real Estate

### C. Rating Thresholds

| Rating | Score Range | Implied Action |
|--------|-------------|----------------|
| Strong Buy | 80-100 | High conviction buy |
| Buy | 65-79 | Favorable risk/reward |
| Hold | 50-64 | Neutral, hold existing |
| Sell | 35-49 | Consider reducing |
| Strong Sell | 0-34 | High conviction sell |

### D. Confidence Level Criteria

| Level | Data Completeness | Core Factors | Display |
|-------|-------------------|--------------|---------|
| High | ≥90% | All 4 quality | Full score, solid badge |
| Medium | 70-89% | ≥3 quality | Score with indicator |
| Low | 50-69% | ≥2 quality | Muted, with warning |
| Insufficient | <50% | <2 quality | No score shown |

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 2.0 | Jan 2026 | IC Team | Complete redesign with sector-relative scoring |
| 1.0 | Oct 2025 | IC Team | Initial IC Score implementation |

---

*This document is confidential and intended for internal use only.*
