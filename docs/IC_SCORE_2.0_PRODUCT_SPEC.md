# IC Score 2.0 - Product Design & Technical Specification

**Version**: 2.0
**Date**: January 31, 2026
**Status**: Draft
**Author**: InvestorCenter.ai Team

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Competitive Analysis](#2-competitive-analysis)
3. [Product Vision & Goals](#3-product-vision--goals)
4. [Scoring Methodology](#4-scoring-methodology)
5. [Factor Definitions](#5-factor-definitions)
6. [Technical Architecture](#6-technical-architecture)
7. [Data Requirements](#7-data-requirements)
8. [User Interface Design](#8-user-interface-design)
9. [Implementation Roadmap](#9-implementation-roadmap)
10. [Appendices](#10-appendices)

---

## 1. Executive Summary

### 1.1 The Problem

The current IC Score system (v1.0) has fundamental issues identified through analysis of AAPL, NVDA, and JNJ:

| Issue | Impact |
|-------|--------|
| **Hardcoded benchmarks** (P/E=15, P/B=2) | Tech stocks unfairly penalized (NVDA scores 0 on P/B, P/S) |
| **Absolute value comparisons** | No sector context (healthcare vs tech vs utilities same benchmarks) |
| **Inverted logic** | NVDA's 4.47 current ratio scores 1.2 (should be excellent) |
| **No growth-adjusted valuation** | PEG ratio not used despite being most relevant for growth stocks |
| **Static factor weights** | Same weights in bull/bear markets, across all sectors |

**Result**: NVDA estimated IC Score: 72.7 vs Claude's assessment: 85 (Strong Buy) - a 12+ point underestimate.

### 1.2 The Solution: IC Score 2.0

IC Score 2.0 is a complete redesign using **sector-relative percentile scoring**, **intelligent factor weighting**, and **multi-dimensional analysis** inspired by industry best practices from YCharts, Morningstar, Simply Wall St, Seeking Alpha, TipRanks, and Zacks.

### 1.3 Key Differentiators

| Feature | Current v1.0 | Proposed v2.0 |
|---------|--------------|---------------|
| **Comparison Method** | Absolute benchmarks | Sector-relative percentiles |
| **Factor Weights** | Static 10 factors | Adaptive weights by sector & market regime |
| **Output Format** | Single score (1-100) | Multi-dimensional (overall + 6 sub-scores) |
| **Time Horizons** | One-size-fits-all | Short/Medium/Long-term views |
| **Uncertainty** | Point estimate | Score + confidence interval |
| **Transparency** | Black box | Detailed factor breakdown with explanations |

---

## 2. Competitive Analysis

### 2.1 Industry Landscape Overview

| Platform | Scoring Approach | Key Innovation | Our Opportunity |
|----------|------------------|----------------|-----------------|
| **YCharts Y-Rating** | 3 components (Value, Fundamental, Historical Multiples) | 100% quantitative, no emotion | Add qualitative signals |
| **Morningstar** | DCF-based fair value + Economic Moat | Moat analysis, analyst-driven | Faster updates, more factors |
| **Zacks Rank** | Earnings revisions focus (AMES) | Proven short-term alpha | Broader factor coverage |
| **Simply Wall St** | 30 pass/fail checks, Snowflake visual | Transparent, visual | More nuanced scoring |
| **Seeking Alpha Quant** | 5 factors, sector-relative grades | Sector normalization | More factors, better UX |
| **TipRanks Smart Score** | 8 factors including sentiment | Social/sentiment data | Deeper fundamental analysis |
| **AAII A+ Grades** | 5 factors, percentile-based | Academic rigor | Real-time updates |

### 2.2 YCharts Y-Rating Deep Dive

**Components**:
1. **Value Score** (1-10): How stock is currently valued vs market
   - Strong backtested performance predictor
   - Higher scores → consistent outperformance

2. **Fundamental Score**: Pass/fail tests for financial health
   - Surfaces red flags before investment
   - Low scores → severe underperformance historically

3. **Valuation from Historical Multiples**: Fair value from past data
   - Compares current price to calculated fair value
   - Most quantitative of the three

**Key Insight**: YCharts separates "is it cheap?" (Value) from "is it healthy?" (Fundamental) from "is it fairly priced historically?" (Historical Multiples).

### 2.3 Morningstar Methodology

**Key Features**:
- **DCF-Based Fair Value**: Analyst-driven discounted cash flow models
- **Economic Moat Rating**: Wide/Narrow/None moat assessment
- **Quantitative Ratings**: Machine learning model with 300 tree predictions
- **Uncertainty Score**: Dispersion among model predictions = confidence
- **Daily Updates**: Ratings update daily based on price + estimates

**Key Insight**: Morningstar uses an **ensemble of 300 models** to generate uncertainty scores - we should express confidence, not just point estimates.

### 2.4 Zacks Rank System

**The AMES Framework**:
- **A**greement: Do analysts agree on direction?
- **M**agnitude: How big are the estimate changes?
- **U**pside: How much upside in estimates?
- **S**urprise: Historical earnings surprise track record

**Style Scores (VGM)**:
- **V**alue: A-F grade
- **G**rowth: A-F grade
- **M**omentum: A-F grade

**Key Insight**: Earnings estimate revisions are the #1 predictor of short-term stock performance. We should weight EPS revisions heavily.

### 2.5 Simply Wall St Snowflake

**Structure**: 30 pass/fail checks across 5 categories (6 checks each)

| Category | Checks | Focus |
|----------|--------|-------|
| **Value** | 6 | DCF vs price, P/E vs market/industry, PEG, P/B |
| **Future** | 6 | Earnings growth, revenue growth, ROE projections |
| **Past** | 6 | EPS growth history, ROE, ROCE, ROA |
| **Health** | 6 | Liquidity, leverage, interest coverage, cash runway |
| **Dividends** | 6 | Yield percentile, stability, coverage |

**Scoring**: Each check = 0 or 1. Total possible = 30.

**Visual**: Snowflake graphic with color coding (green = good, red = bad).

**Key Insight**: Binary pass/fail provides clarity but loses nuance. We should combine pass/fail checkpoints with continuous scores.

### 2.6 Seeking Alpha Quant Ratings

**5 Factors** (A+ to F grades):
1. **Value**: P/E, P/B vs sector peers
2. **Growth**: Revenue/earnings growth trends
3. **Profitability**: Margins, ROE efficiency
4. **Momentum**: Recent price movement
5. **EPS Revisions**: Analyst forecast changes

**Methodology**:
- Compare 100+ metrics to sector peers
- Percentile-based grading within sector
- **Disqualification rules**: D+ or worse on Growth/Momentum/Revisions = max "Hold"

**Key Insight**: Sector-relative percentile grading is the gold standard. We must implement this.

### 2.7 TipRanks Smart Score

**8 Factors** (scale 1-10):
1. Wall Street Analysts
2. Corporate Insiders
3. Financial Bloggers
4. Individual Investor Sentiment
5. Hedge Fund Managers
6. News Sentiment
7. Technical Factors
8. Fundamentals

**Performance**: 389% return since 2016 vs S&P 500's 235% (top-rated stocks).

**Key Insight**: TipRanks uniquely includes **social sentiment** (bloggers, retail sentiment) and **smart money** (hedge funds, insiders). We should leverage our Reddit sentiment data more heavily.

### 2.8 AAII A+ Grades

**5 Factors** (percentile-based A-F grades):
1. **Value**: 6 fundamental ratios
2. **Growth**: 5-year sales growth consistency
3. **Momentum**: Price momentum
4. **Revisions**: Earnings estimate changes
5. **Quality**: 8-metric composite (ROA, ROIC, GP/Assets, Buybacks, Leverage, Accruals, Z-Score, F-Score)

**Quality Score Composition**:
- ROA, ROIC (profitability)
- Gross profit to assets (efficiency)
- Buyback yield (capital return)
- Change in total liabilities to assets (leverage)
- Accruals (earnings quality)
- Altman Z double prime (bankruptcy risk)
- Piotroski F-Score (financial strength)

**Key Insight**: AAII's **8-metric Quality Score** is rigorous. We should incorporate Piotroski F-Score and Altman Z-Score.

### 2.9 Competitive Positioning Matrix

```
                    COMPREHENSIVE
                         ^
                         |
          AAII  •    •  IC Score 2.0 (Goal)
                         |
    Seeking Alpha •      |      • TipRanks
                         |
          Zacks  •-------+-------• YCharts
                         |
                         |      • Simply Wall St
        Morningstar •    |
                         |
                         v
                    SPECIALIZED
          <----- QUANT          QUAL ----->
```

**Our Target Position**: Most comprehensive quantitative scoring with qualitative enhancements (sentiment, moat analysis).

---

## 3. Product Vision & Goals

### 3.1 Vision Statement

> **IC Score 2.0** is the most transparent, accurate, and actionable stock scoring system for individual investors, combining sector-relative quantitative analysis with sentiment signals to identify stocks with the best risk-adjusted return potential.

### 3.2 Design Principles

1. **Sector-Relative**: Never compare apples to oranges
2. **Transparent**: Every score explainable with clear reasoning
3. **Adaptive**: Factor weights respond to market conditions
4. **Multi-Dimensional**: Different scores for different investment horizons
5. **Confidence-Aware**: Express uncertainty, not false precision
6. **Actionable**: Clear buy/hold/sell guidance with conviction levels

### 3.3 Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Accuracy vs Analysts** | 80%+ correlation | Compare IC Score rating to consensus |
| **Predictive Power** | Beat S&P 500 | Track top-rated stocks' 6/12-month returns |
| **User Understanding** | 4+ out of 5 | Survey users on score comprehension |
| **Coverage** | 10,000+ stocks | US stocks, ETFs with 1+ year history |
| **Freshness** | <24 hours | Time from data change to score update |

### 3.4 Non-Goals (v2.0)

- Real-time intraday scoring (daily is sufficient)
- International stocks (US-only in v2.0)
- Cryptocurrency scoring (separate product)
- Portfolio optimization recommendations

---

## 4. Scoring Methodology

### 4.1 Overall Framework

```
┌─────────────────────────────────────────────────────────────────────┐
│                        IC SCORE 2.0 ENGINE                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    STEP 1: DATA COLLECTION                   │   │
│  │  Financial Statements • Prices • Estimates • Sentiment       │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│                              ▼                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                 STEP 2: SECTOR CLASSIFICATION               │   │
│  │  GICS Sector → Industry Group → Industry → Sub-Industry      │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│                              ▼                                      │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                STEP 3: PERCENTILE RANKING                    │   │
│  │  Rank each metric within sector/industry peer group          │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              │                                      │
│                              ▼                                      │
│  ┌───────────┬───────────┬───────────┬───────────┬───────────┐   │
│  │   VALUE   │  GROWTH   │  QUALITY  │  MOMENTUM │  SENTIMENT │   │
│  │   Score   │   Score   │   Score   │   Score   │    Score   │   │
│  └─────┬─────┴─────┬─────┴─────┬─────┴─────┬─────┴─────┬─────┘   │
│        │           │           │           │           │         │
│        └───────────┴─────┬─────┴───────────┴───────────┘         │
│                          │                                        │
│                          ▼                                        │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │              STEP 4: WEIGHTED COMBINATION                    │   │
│  │  Apply sector-specific + market-regime weights               │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                          │                                        │
│                          ▼                                        │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │              STEP 5: CONFIDENCE ADJUSTMENT                   │   │
│  │  Factor in data completeness + volatility → confidence band  │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                          │                                        │
│                          ▼                                        │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                  FINAL IC SCORE OUTPUT                       │   │
│  │   Overall: 82 (±5)  |  Rating: STRONG BUY  |  Confidence: HIGH │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.2 Scoring Scale

| Score Range | Rating | Color | Interpretation |
|-------------|--------|-------|----------------|
| 85-100 | **Strong Buy** | Dark Green | Excellent across all factors |
| 70-84 | **Buy** | Light Green | Solid opportunity with minor concerns |
| 50-69 | **Hold** | Yellow | Fair valuation, mixed signals |
| 30-49 | **Sell** | Orange | Concerning fundamentals or valuation |
| 1-29 | **Strong Sell** | Red | Significant red flags |

### 4.3 The 6 Factor Framework

IC Score 2.0 uses **6 core factors** (expanded from 10 in v1.0 for clarity):

| Factor | Weight Range | What It Measures |
|--------|--------------|------------------|
| **Value** | 15-25% | Is the stock cheap relative to fundamentals? |
| **Growth** | 10-20% | Is the company growing faster than peers? |
| **Quality** | 15-25% | Is the company financially healthy and profitable? |
| **Momentum** | 5-15% | Is price/earnings momentum positive? |
| **Sentiment** | 10-20% | What do analysts, insiders, and social say? |
| **Risk** | 10-15% | What are the downside risks? |

### 4.4 Sector-Specific Weight Profiles

Different sectors prioritize different factors:

| Sector | Value | Growth | Quality | Momentum | Sentiment | Risk |
|--------|-------|--------|---------|----------|-----------|------|
| **Technology** | 15% | 20% | 20% | 15% | 15% | 15% |
| **Healthcare** | 15% | 20% | 25% | 10% | 15% | 15% |
| **Financials** | 20% | 10% | 25% | 10% | 15% | 20% |
| **Consumer Discretionary** | 15% | 20% | 20% | 15% | 20% | 10% |
| **Consumer Staples** | 25% | 10% | 25% | 10% | 15% | 15% |
| **Energy** | 20% | 10% | 20% | 15% | 15% | 20% |
| **Utilities** | 25% | 5% | 25% | 10% | 15% | 20% |
| **Real Estate** | 25% | 10% | 20% | 10% | 15% | 20% |
| **Industrials** | 20% | 15% | 20% | 15% | 15% | 15% |
| **Materials** | 20% | 15% | 20% | 15% | 15% | 15% |
| **Communication Services** | 15% | 20% | 20% | 15% | 20% | 10% |

### 4.5 Market Regime Adjustments

Factor weights shift based on market conditions:

| Regime | Trigger | Adjustment |
|--------|---------|------------|
| **High Volatility** | VIX > 25 | Quality +5%, Momentum -5% |
| **Bull Market** | S&P 500 +15% YTD | Growth +5%, Value -5% |
| **Bear Market** | S&P 500 -15% YTD | Value +5%, Quality +5%, Growth -5%, Momentum -5% |
| **Rising Rates** | 10Y yield up >50bps in 3 months | Value +5%, Growth -5% |

### 4.6 Percentile Ranking Methodology

**Why Percentiles?**

Instead of: "P/E of 30 is bad" (arbitrary)
Use: "P/E of 30 is 35th percentile in Tech sector" (contextual)

**Calculation Process**:

```python
def calculate_percentile_score(value, peer_values, lower_is_better=True):
    """
    Convert a raw metric value to a 0-100 score based on percentile rank.

    Args:
        value: The stock's metric value
        peer_values: List of values from sector peers
        lower_is_better: True for metrics like P/E where lower is better

    Returns:
        Score from 0-100 where 100 is best
    """
    percentile = scipy.stats.percentileofscore(peer_values, value)

    if lower_is_better:
        # For P/E, P/B, etc.: Lower percentile = better value = higher score
        return 100 - percentile
    else:
        # For ROE, Growth, etc.: Higher percentile = better = higher score
        return percentile
```

**Example: NVDA P/E Scoring**

| Approach | P/E | Calculation | Score |
|----------|-----|-------------|-------|
| **v1.0 (Absolute)** | 46.21 | `100 - (46.21 - 15) * 2` | 37.6 |
| **v2.0 (Percentile)** | 46.21 | Tech sector 35th percentile → `100 - 35` | 65 |

### 4.7 Confidence Scoring

Every IC Score includes a confidence level based on:

1. **Data Completeness**: How many of the input metrics are available?
2. **Data Freshness**: How recent is the financial data?
3. **Analyst Coverage**: More analysts = more reliable estimates
4. **Historical Consistency**: Stable metrics = more confidence

```python
def calculate_confidence(stock_data, sector_data):
    """Calculate confidence score (0-100) for the IC Score."""

    # Data completeness (0-30 points)
    required_metrics = 20
    available_metrics = count_available_metrics(stock_data)
    completeness_score = (available_metrics / required_metrics) * 30

    # Data freshness (0-25 points)
    days_since_financials = (today - stock_data.last_filing_date).days
    if days_since_financials < 30:
        freshness_score = 25
    elif days_since_financials < 90:
        freshness_score = 20
    elif days_since_financials < 180:
        freshness_score = 10
    else:
        freshness_score = 0

    # Analyst coverage (0-25 points)
    analyst_count = stock_data.analyst_count
    if analyst_count >= 20:
        coverage_score = 25
    elif analyst_count >= 10:
        coverage_score = 20
    elif analyst_count >= 5:
        coverage_score = 15
    elif analyst_count >= 1:
        coverage_score = 10
    else:
        coverage_score = 0

    # Historical consistency (0-20 points)
    metric_volatility = calculate_metric_volatility(stock_data)
    if metric_volatility < 0.1:
        consistency_score = 20
    elif metric_volatility < 0.2:
        consistency_score = 15
    elif metric_volatility < 0.3:
        consistency_score = 10
    else:
        consistency_score = 5

    total_confidence = completeness_score + freshness_score + coverage_score + consistency_score

    return total_confidence  # 0-100
```

**Confidence Display**:

| Confidence Score | Label | Margin of Error |
|------------------|-------|-----------------|
| 80-100 | High | ±3 points |
| 60-79 | Medium | ±5 points |
| 40-59 | Low | ±8 points |
| <40 | Very Low | ±12 points |

---

## 5. Factor Definitions

### 5.1 VALUE Factor (15-25%)

**Purpose**: Identify stocks trading below intrinsic value relative to sector peers.

**Metrics Used**:

| Metric | Weight | Lower is Better | Sector Relative |
|--------|--------|-----------------|-----------------|
| P/E Ratio (TTM) | 20% | Yes | Yes |
| P/E Ratio (Forward) | 15% | Yes | Yes |
| PEG Ratio | 20% | Yes | Yes |
| P/B Ratio | 15% | Yes | Yes |
| P/S Ratio | 10% | Yes | Yes |
| EV/EBITDA | 10% | Yes | Yes |
| P/FCF | 10% | Yes | Yes |

**Calculation**:

```python
def calculate_value_score(stock, sector_peers):
    weights = {
        'pe_ttm': 0.20, 'pe_forward': 0.15, 'peg': 0.20,
        'pb': 0.15, 'ps': 0.10, 'ev_ebitda': 0.10, 'p_fcf': 0.10
    }

    score = 0
    for metric, weight in weights.items():
        value = getattr(stock, metric)
        if value is None or value <= 0:
            continue  # Skip invalid values

        peer_values = [getattr(p, metric) for p in sector_peers if getattr(p, metric)]
        percentile_score = calculate_percentile_score(value, peer_values, lower_is_better=True)
        score += percentile_score * weight

    # Normalize if some metrics were missing
    return normalize_score(score, weights)
```

**Special Handling**:
- Negative P/E (losses): Score 0 for P/E, rely on other metrics
- PEG < 0 or > 5: Exclude from calculation
- REITs: Use P/FFO instead of P/E

### 5.2 GROWTH Factor (10-20%)

**Purpose**: Identify companies with superior growth trajectory vs peers.

**Metrics Used**:

| Metric | Weight | Higher is Better |
|--------|--------|------------------|
| Revenue Growth (YoY) | 25% | Yes |
| Revenue Growth (3-Year CAGR) | 15% | Yes |
| EPS Growth (YoY) | 25% | Yes |
| EPS Growth (3-Year CAGR) | 15% | Yes |
| Analyst EPS Estimate Revisions (30-day) | 20% | Yes |

**Special Adjustments**:
- Companies with 5+ consecutive quarters of revenue growth: +5 bonus points
- EPS turning from loss to profit: Score as top quartile
- Growth deceleration (current YoY < 3-year CAGR): -5 penalty

### 5.3 QUALITY Factor (15-25%)

**Purpose**: Assess financial health, profitability, and earnings quality.

**Metrics Used** (Inspired by AAII's 8-metric Quality Score):

| Metric | Weight | What It Measures |
|--------|--------|------------------|
| ROE (TTM) | 15% | Shareholder returns |
| ROIC (TTM) | 15% | Capital efficiency |
| Gross Margin | 10% | Pricing power |
| Operating Margin | 10% | Operational efficiency |
| Net Margin | 10% | Overall profitability |
| Current Ratio | 10% | Short-term liquidity |
| Debt/Equity | 10% | Leverage risk |
| Interest Coverage | 10% | Debt servicing ability |
| Piotroski F-Score | 5% | Financial strength (1-9) |
| Altman Z-Score | 5% | Bankruptcy risk |

**Piotroski F-Score Components** (1 point each):
1. Positive net income
2. Positive operating cash flow
3. ROA increased YoY
4. Operating cash flow > net income (earnings quality)
5. Long-term debt decreased YoY
6. Current ratio increased YoY
7. No share dilution
8. Gross margin increased YoY
9. Asset turnover increased YoY

**Altman Z-Score** (for non-financial companies):
```
Z = 1.2×(Working Capital/Assets) + 1.4×(Retained Earnings/Assets)
  + 3.3×(EBIT/Assets) + 0.6×(Market Cap/Total Liabilities) + 1.0×(Revenue/Assets)

Z > 2.99: Safe Zone (Score: 100)
Z 1.81-2.99: Grey Zone (Score: 50-80)
Z < 1.81: Distress Zone (Score: 0-50)
```

### 5.4 MOMENTUM Factor (5-15%)

**Purpose**: Capture price and earnings momentum trends.

**Metrics Used**:

| Metric | Weight | Description |
|--------|--------|-------------|
| Price Return (1 month) | 15% | Recent momentum |
| Price Return (3 months) | 20% | Short-term trend |
| Price Return (6 months) | 20% | Medium-term trend |
| Price Return (12 months) | 20% | Long-term trend |
| RSI (14-day) | 15% | Overbought/oversold |
| 50-day MA vs 200-day MA | 10% | Trend direction |

**RSI Scoring**:
- RSI 30-70: Score 50-100 (proportional)
- RSI < 30: Oversold, Score 60-80 (potential bounce)
- RSI > 70: Overbought, Score 30-50 (potential pullback)

**Moving Average Scoring**:
- 50-day > 200-day (Golden Cross): +10 points
- 50-day < 200-day (Death Cross): -10 points

### 5.5 SENTIMENT Factor (10-20%)

**Purpose**: Aggregate professional and crowd sentiment signals.

**Sub-Factors**:

| Signal | Weight | Source |
|--------|--------|--------|
| **Analyst Consensus** | 35% | Wall Street ratings |
| **Analyst Price Target** | 15% | Upside to consensus target |
| **Insider Activity** | 20% | Net insider buying % |
| **Institutional Ownership** | 15% | 13F filings changes |
| **Social Sentiment** | 15% | Reddit + news sentiment |

**Analyst Consensus Scoring**:
```python
def analyst_score(buy_count, hold_count, sell_count):
    total = buy_count + hold_count + sell_count
    if total == 0:
        return 50  # Neutral if no coverage

    # Buy % weighted 2x, Hold % weighted 1x, Sell % weighted 0x
    weighted_score = (buy_count * 100 + hold_count * 50) / total
    return weighted_score
```

**Insider Activity Scoring**:
```python
def insider_score(net_shares_bought, shares_outstanding):
    # Use percentage of shares outstanding, not absolute shares
    pct_bought = (net_shares_bought / shares_outstanding) * 100

    # Scale: ±0.5% = full range
    if pct_bought >= 0.5:
        return 100
    elif pct_bought <= -0.5:
        return 0
    else:
        return 50 + (pct_bought * 100)  # Linear scaling
```

**Social Sentiment Scoring** (Reddit + News):
- Aggregate sentiment from `reddit_post_tickers` table
- Weight by upvotes/engagement
- Combine with news sentiment from `news_articles` table
- Time decay: Last 7 days weighted 2x vs 8-30 days

### 5.6 RISK Factor (10-15%)

**Purpose**: Assess downside risks and volatility.

**Metrics Used**:

| Metric | Weight | Description |
|--------|--------|-------------|
| Beta | 25% | Market correlation |
| Volatility (1-year) | 25% | Price stability |
| Max Drawdown (1-year) | 15% | Worst decline |
| VaR (95%, 1-day) | 15% | Value at Risk |
| Sharpe Ratio | 10% | Risk-adjusted return |
| Sortino Ratio | 10% | Downside risk-adjusted return |

**Risk Score Calculation**:
Lower risk = Higher score (inverted)

```python
def calculate_risk_score(stock, sector_peers):
    # Beta scoring (1.0 is neutral)
    if stock.beta < 0.8:
        beta_score = 90  # Low beta is good
    elif stock.beta <= 1.2:
        beta_score = 70  # Market-like beta
    elif stock.beta <= 1.5:
        beta_score = 50  # Elevated beta
    else:
        beta_score = 30  # High beta

    # Volatility: Compare to sector percentile (lower is better)
    vol_score = calculate_percentile_score(
        stock.volatility_1y,
        [p.volatility_1y for p in sector_peers],
        lower_is_better=True
    )

    # Combine with weights
    return (beta_score * 0.25) + (vol_score * 0.25) + ...
```

---

## 6. Technical Architecture

### 6.1 System Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                         DATA LAYER                                  │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐         │
│   │ Polygon  │  │  SEC     │  │  FMP     │  │  Reddit  │         │
│   │   API    │  │ EDGAR    │  │   API    │  │Sentiment │         │
│   └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘         │
│        │             │             │             │                 │
│        └─────────────┴─────────────┴─────────────┘                 │
│                           │                                        │
│                           ▼                                        │
│   ┌─────────────────────────────────────────────────────────────┐ │
│   │                    PostgreSQL + TimescaleDB                  │ │
│   │                                                              │ │
│   │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │ │
│   │  │   tickers   │ │  companies  │ │stock_prices │           │ │
│   │  │   (25K)     │ │   (10K)     │ │(hypertable) │           │ │
│   │  └─────────────┘ └─────────────┘ └─────────────┘           │ │
│   │                                                              │ │
│   │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │ │
│   │  │ financials  │ │ttm_financials│ │valuation_   │           │ │
│   │  │             │ │             │ │ratios       │           │ │
│   │  └─────────────┘ └─────────────┘ └─────────────┘           │ │
│   │                                                              │ │
│   │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │ │
│   │  │technical_   │ │risk_metrics │ │analyst_     │           │ │
│   │  │indicators   │ │             │ │ratings      │           │ │
│   │  └─────────────┘ └─────────────┘ └─────────────┘           │ │
│   │                                                              │ │
│   │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │ │
│   │  │ ic_scores   │ │sector_stats │ │ic_score_    │  ← NEW    │ │
│   │  │  (v2.0)     │ │(percentiles)│ │breakdowns   │           │ │
│   │  └─────────────┘ └─────────────┘ └─────────────┘           │ │
│   │                                                              │ │
│   └─────────────────────────────────────────────────────────────┘ │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌────────────────────────────────────────────────────────────────────┐
│                       CALCULATION LAYER                            │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│   ┌─────────────────────────────────────────────────────────────┐ │
│   │                IC SCORE 2.0 CALCULATOR                       │ │
│   │                (Python / FastAPI)                            │ │
│   │                                                              │ │
│   │  ┌─────────────────────────────────────────────────────┐    │ │
│   │  │  STEP 1: Sector Statistics Calculator (Daily)        │    │ │
│   │  │  - Calculate percentiles for all metrics per sector  │    │ │
│   │  │  - Store in sector_stats table                        │    │ │
│   │  └─────────────────────────────────────────────────────┘    │ │
│   │                           │                                  │ │
│   │                           ▼                                  │ │
│   │  ┌─────────────────────────────────────────────────────┐    │ │
│   │  │  STEP 2: Individual Stock Scoring (Daily)            │    │ │
│   │  │  - Rank each stock vs sector percentiles             │    │ │
│   │  │  - Calculate 6 factor scores                         │    │ │
│   │  │  - Apply sector weights + market regime              │    │ │
│   │  │  - Generate confidence score                         │    │ │
│   │  └─────────────────────────────────────────────────────┘    │ │
│   │                           │                                  │ │
│   │                           ▼                                  │ │
│   │  ┌─────────────────────────────────────────────────────┐    │ │
│   │  │  STEP 3: Store Results                               │    │ │
│   │  │  - ic_scores table (overall + rating)                │    │ │
│   │  │  - ic_score_breakdowns table (6 factor scores)       │    │ │
│   │  └─────────────────────────────────────────────────────┘    │ │
│   │                                                              │ │
│   └─────────────────────────────────────────────────────────────┘ │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌────────────────────────────────────────────────────────────────────┐
│                         API LAYER                                  │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│   ┌─────────────────────────────────────────────────────────────┐ │
│   │                  IC SCORE API (FastAPI)                      │ │
│   │                                                              │ │
│   │  GET /api/v1/ic-score/{ticker}                              │ │
│   │  → Returns overall score, rating, confidence, 6 factors     │ │
│   │                                                              │ │
│   │  GET /api/v1/ic-score/{ticker}/breakdown                    │ │
│   │  → Returns detailed factor breakdown with all metrics       │ │
│   │                                                              │ │
│   │  GET /api/v1/ic-score/{ticker}/history                      │ │
│   │  → Returns score history over time                          │ │
│   │                                                              │ │
│   │  GET /api/v1/ic-score/screener                              │ │
│   │  → Screen stocks by IC Score with filters                   │ │
│   │                                                              │ │
│   │  GET /api/v1/ic-score/top-rated                             │ │
│   │  → Top 10/25/50 stocks by IC Score                          │ │
│   │                                                              │ │
│   └─────────────────────────────────────────────────────────────┘ │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌────────────────────────────────────────────────────────────────────┐
│                      PRESENTATION LAYER                            │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│   ┌─────────────────────────────────────────────────────────────┐ │
│   │                 Next.js Frontend                             │ │
│   │                                                              │ │
│   │  ┌─────────────────┐  ┌─────────────────┐                   │ │
│   │  │   IC Score      │  │  Factor Radar   │                   │ │
│   │  │   Card          │  │  Chart          │                   │ │
│   │  │  ┌───────────┐  │  │                 │                   │ │
│   │  │  │    82     │  │  │  Value ●────────│                   │ │
│   │  │  │  ━━━━━━━  │  │  │       ╱    ╲    │                   │ │
│   │  │  │STRONG BUY │  │  │ Risk ●      ● Growth                │ │
│   │  │  └───────────┘  │  │      ╲      ╱   │                   │ │
│   │  │  Confidence:    │  │       ●────●    │                   │ │
│   │  │  HIGH (±3)      │  │Sent.     Qual.  │                   │ │
│   │  └─────────────────┘  └─────────────────┘                   │ │
│   │                                                              │ │
│   │  ┌─────────────────────────────────────────────────────┐    │ │
│   │  │               Factor Breakdown Table                 │    │ │
│   │  │                                                      │    │ │
│   │  │  Factor    Score   Grade   Key Drivers               │    │ │
│   │  │  ───────── ─────── ─────── ────────────────────────  │    │ │
│   │  │  Value     65      B       PEG 0.73 (top 15%)        │    │ │
│   │  │  Growth    92      A+      Rev +95% YoY              │    │ │
│   │  │  Quality   95      A+      53% margins, 4.5x CR      │    │ │
│   │  │  Momentum  78      B+      +45% 12-month             │    │ │
│   │  │  Sentiment 88      A       95% analyst Buy           │    │ │
│   │  │  Risk      72      B       Beta 1.8, high vol        │    │ │
│   │  └─────────────────────────────────────────────────────┘    │ │
│   │                                                              │ │
│   └─────────────────────────────────────────────────────────────┘ │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

### 6.2 New Database Tables

#### `sector_stats` - Sector Percentile Statistics

```sql
CREATE TABLE sector_stats (
    id SERIAL PRIMARY KEY,
    calculation_date DATE NOT NULL,
    sector VARCHAR(100) NOT NULL,
    industry VARCHAR(100),
    metric_name VARCHAR(100) NOT NULL,

    -- Percentile values
    p10 DECIMAL(20, 6),
    p25 DECIMAL(20, 6),
    p50 DECIMAL(20, 6),  -- Median
    p75 DECIMAL(20, 6),
    p90 DECIMAL(20, 6),

    -- Statistics
    mean DECIMAL(20, 6),
    std_dev DECIMAL(20, 6),
    min_value DECIMAL(20, 6),
    max_value DECIMAL(20, 6),
    count INTEGER,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(calculation_date, sector, industry, metric_name)
);

-- Index for fast lookups
CREATE INDEX idx_sector_stats_lookup
ON sector_stats(calculation_date, sector, metric_name);
```

#### `ic_scores_v2` - Main Scores Table

```sql
CREATE TABLE ic_scores_v2 (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(20) NOT NULL,
    calculation_date DATE NOT NULL,

    -- Overall score and rating
    overall_score DECIMAL(5, 2) NOT NULL,
    rating VARCHAR(20) NOT NULL,  -- STRONG_BUY, BUY, HOLD, SELL, STRONG_SELL

    -- Confidence
    confidence_score DECIMAL(5, 2),
    confidence_level VARCHAR(20),  -- HIGH, MEDIUM, LOW, VERY_LOW
    margin_of_error DECIMAL(5, 2),

    -- 6 Factor scores
    value_score DECIMAL(5, 2),
    growth_score DECIMAL(5, 2),
    quality_score DECIMAL(5, 2),
    momentum_score DECIMAL(5, 2),
    sentiment_score DECIMAL(5, 2),
    risk_score DECIMAL(5, 2),

    -- Factor weights used
    value_weight DECIMAL(4, 3),
    growth_weight DECIMAL(4, 3),
    quality_weight DECIMAL(4, 3),
    momentum_weight DECIMAL(4, 3),
    sentiment_weight DECIMAL(4, 3),
    risk_weight DECIMAL(4, 3),

    -- Context
    sector VARCHAR(100),
    industry VARCHAR(100),
    market_regime VARCHAR(50),

    -- Previous score for change tracking
    previous_score DECIMAL(5, 2),
    score_change DECIMAL(5, 2),

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(ticker, calculation_date)
);

-- Convert to TimescaleDB hypertable
SELECT create_hypertable('ic_scores_v2', 'calculation_date');

-- Indexes
CREATE INDEX idx_ic_scores_v2_ticker ON ic_scores_v2(ticker, calculation_date DESC);
CREATE INDEX idx_ic_scores_v2_rating ON ic_scores_v2(calculation_date, rating);
CREATE INDEX idx_ic_scores_v2_sector ON ic_scores_v2(calculation_date, sector);
```

#### `ic_score_breakdowns` - Detailed Factor Breakdown

```sql
CREATE TABLE ic_score_breakdowns (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(20) NOT NULL,
    calculation_date DATE NOT NULL,
    factor VARCHAR(50) NOT NULL,

    -- Factor score
    score DECIMAL(5, 2) NOT NULL,
    grade CHAR(2),  -- A+, A, A-, B+, B, B-, C+, C, C-, D, F

    -- Individual metric scores within this factor
    metrics JSONB,  -- {"pe_score": 65, "peg_score": 90, ...}

    -- Key driver explanation
    key_drivers TEXT,  -- "PEG ratio of 0.73 ranks in top 15% of sector"

    -- Sector comparison
    sector_percentile DECIMAL(5, 2),
    sector_rank INTEGER,
    sector_count INTEGER,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(ticker, calculation_date, factor)
);

-- Index for fast lookups
CREATE INDEX idx_ic_breakdowns_ticker
ON ic_score_breakdowns(ticker, calculation_date);
```

### 6.3 API Endpoints

#### GET `/api/v1/ic-score/{ticker}`

**Response**:
```json
{
  "ticker": "NVDA",
  "name": "NVIDIA Corporation",
  "calculation_date": "2026-01-31",

  "score": {
    "overall": 84.3,
    "rating": "STRONG_BUY",
    "previous": 82.1,
    "change": 2.2
  },

  "confidence": {
    "score": 85,
    "level": "HIGH",
    "margin_of_error": 3
  },

  "factors": {
    "value": {"score": 65, "grade": "B", "weight": 0.15},
    "growth": {"score": 92, "grade": "A+", "weight": 0.20},
    "quality": {"score": 95, "grade": "A+", "weight": 0.20},
    "momentum": {"score": 78, "grade": "B+", "weight": 0.15},
    "sentiment": {"score": 88, "grade": "A", "weight": 0.15},
    "risk": {"score": 72, "grade": "B", "weight": 0.15}
  },

  "context": {
    "sector": "Technology",
    "industry": "Semiconductors",
    "market_regime": "NORMAL",
    "sector_rank": 12,
    "sector_count": 487
  },

  "summary": "NVDA earns a STRONG BUY rating with exceptional growth (92) and quality (95) scores. While valuation (65) appears stretched on traditional metrics, the PEG ratio of 0.73 suggests the stock is undervalued relative to its growth rate. Risk (72) reflects elevated volatility typical for high-growth tech."
}
```

#### GET `/api/v1/ic-score/{ticker}/breakdown`

**Response**:
```json
{
  "ticker": "NVDA",
  "calculation_date": "2026-01-31",

  "value": {
    "score": 65,
    "grade": "B",
    "metrics": {
      "pe_ttm": {"value": 46.21, "sector_percentile": 35, "score": 65},
      "pe_forward": {"value": 26.57, "sector_percentile": 25, "score": 75},
      "peg": {"value": 0.73, "sector_percentile": 85, "score": 85},
      "pb": {"value": 38.29, "sector_percentile": 15, "score": 40},
      "ps": {"value": 24.92, "sector_percentile": 10, "score": 35},
      "ev_ebitda": {"value": 35.2, "sector_percentile": 30, "score": 70},
      "p_fcf": {"value": 42.1, "sector_percentile": 25, "score": 55}
    },
    "key_drivers": [
      "PEG ratio of 0.73 ranks in top 15% of Tech sector (excellent value for growth)",
      "Forward P/E of 26.57 suggests reasonable valuation for expected growth",
      "P/B and P/S elevated due to asset-light business model"
    ]
  },

  "growth": {
    "score": 92,
    "grade": "A+",
    "metrics": {
      "revenue_yoy": {"value": 94.8, "sector_percentile": 99, "score": 99},
      "revenue_3y_cagr": {"value": 65.2, "sector_percentile": 98, "score": 98},
      "eps_yoy": {"value": 112.3, "sector_percentile": 99, "score": 99},
      "eps_3y_cagr": {"value": 78.4, "sector_percentile": 99, "score": 99},
      "estimate_revisions": {"value": 5.2, "sector_percentile": 75, "score": 75}
    },
    "key_drivers": [
      "Revenue growth of 95% YoY is #1 in large-cap Tech",
      "5 consecutive quarters of accelerating growth",
      "Analysts revising estimates up 5.2% in last 30 days"
    ]
  }
  // ... other factors
}
```

### 6.4 Calculation Pipeline

New CronJob: `ic-score-v2-calculator`

**Schedule**: Daily at 1:00 AM UTC (after all data pipelines complete)

**Steps**:
1. Calculate sector statistics (percentiles for all metrics)
2. For each stock in `companies` table:
   - Fetch all required metrics
   - Rank vs sector percentiles
   - Calculate 6 factor scores
   - Apply sector-specific weights
   - Apply market regime adjustment
   - Calculate confidence score
   - Generate summary text
   - Store in `ic_scores_v2` and `ic_score_breakdowns`

**Execution Time**: ~30-45 minutes for 10,000 stocks

---

## 7. Data Requirements

### 7.1 Required Data Sources (Already Available)

| Data | Source | Table | Refresh |
|------|--------|-------|---------|
| Price Data | Polygon | `stock_prices` | Daily |
| Financials | SEC EDGAR | `financials`, `ttm_financials` | Weekly |
| Valuation Ratios | Calculated | `valuation_ratios` | Daily |
| Technical Indicators | Calculated | `technical_indicators` | Daily |
| Risk Metrics | Calculated | `risk_metrics` | Daily |
| Analyst Ratings | FMP | `analyst_ratings` | Daily |
| Insider Trades | SEC | `insider_trades` | Hourly |
| News Sentiment | AI | `news_articles` | 4 hours |
| Reddit Sentiment | Arctic Shift + Gemini | `reddit_post_tickers` | Hourly |

### 7.2 New Data Requirements

| Data | Source | Purpose | Priority |
|------|--------|---------|----------|
| Institutional Holdings (13F) | SEC | Sentiment factor | High |
| Sector Classifications (GICS) | Already in `companies` | Sector grouping | High |
| VIX / Market Regime | Polygon | Weight adjustment | Medium |
| Analyst Price Targets | FMP | Sentiment factor | Medium |
| Piotroski F-Score | Calculated | Quality factor | Medium |
| Altman Z-Score | Calculated | Quality factor | Medium |

### 7.3 Data Completeness Requirements

For a stock to receive an IC Score:

| Requirement | Minimum |
|-------------|---------|
| Price history | 252 trading days (1 year) |
| Financial statements | 4 quarters |
| Sector classification | Must have GICS sector |
| Market cap | > $100 million |

Stocks not meeting requirements get `rating: "INSUFFICIENT_DATA"`.

---

## 8. User Interface Design

### 8.1 IC Score Card (Summary View)

```
┌─────────────────────────────────────────────────────────┐
│  IC Score                                               │
│  ┌──────────────────────────────────────────────────┐  │
│  │                                                   │  │
│  │           ┌───────────────┐                      │  │
│  │           │      84       │  STRONG BUY          │  │
│  │           │    ▲ +2.2     │                      │  │
│  │           └───────────────┘                      │  │
│  │                                                   │  │
│  │  Confidence: HIGH (±3 points)                    │  │
│  │  Sector Rank: #12 of 487 in Technology           │  │
│  │                                                   │  │
│  │  ┌─────┬─────┬─────┬─────┬─────┬─────┐          │  │
│  │  │Value│Grwth│Qual.│Mom. │Sent.│Risk │          │  │
│  │  │ 65  │ 92  │ 95  │ 78  │ 88  │ 72  │          │  │
│  │  │  B  │ A+  │ A+  │ B+  │  A  │  B  │          │  │
│  │  └─────┴─────┴─────┴─────┴─────┴─────┘          │  │
│  │                                                   │  │
│  └──────────────────────────────────────────────────┘  │
│                                                         │
│  Key Strengths:                                        │
│  • Exceptional growth (95% revenue YoY) - top 1%       │
│  • Best-in-class profitability (53% net margin)        │
│  • 95% analyst Buy ratings                             │
│                                                         │
│  Key Risks:                                            │
│  • High beta (1.8) means elevated volatility           │
│  • Valuation stretched on P/B, P/S metrics             │
│                                                         │
│  [View Full Analysis →]                                │
└─────────────────────────────────────────────────────────┘
```

### 8.2 Factor Radar Chart

```
                    Value (65)
                        │
                       ╱│╲
                      ╱ │ ╲
            Risk ────●──┼──●──── Growth
            (72)    ╱   │   ╲    (92)
                   ╱    │    ╲
                  ╱     │     ╲
                 ●──────┼──────●
          Sentiment     │     Quality
            (88)        │       (95)
                        │
                    Momentum
                      (78)
```

### 8.3 Detailed Factor Breakdown (Expandable)

```
┌─────────────────────────────────────────────────────────────┐
│  VALUE FACTOR                                    65 │ B     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Metric          Value     Sector %ile    Score    Weight  │
│  ────────────────────────────────────────────────────────  │
│  P/E (TTM)       46.21     35th           65       20%     │
│  P/E (Forward)   26.57     25th           75       15%     │
│  PEG Ratio        0.73     85th           85 ★     20%     │
│  P/B Ratio       38.29     15th           40       15%     │
│  P/S Ratio       24.92     10th           35       10%     │
│  EV/EBITDA       35.20     30th           70       10%     │
│  P/FCF           42.10     25th           55       10%     │
│                                                             │
│  Key Driver: PEG of 0.73 indicates stock is undervalued    │
│  relative to its exceptional growth rate.                   │
│                                                             │
│  ★ = Best metric in factor                                 │
│                                                             │
│  Sector Comparison: Tech sector (n=487)                    │
│  ├── Your Value Score: 65 (58th percentile)                │
│  ├── Sector Median: 52                                     │
│  └── Sector Range: 12 - 94                                 │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 8.4 Score History Chart

```
IC Score History (12 Months)
100 ┤
 90 ┤                                    ╭──╮    ╭───
 80 ┤                      ╭────────────╯  ╰────╯
 70 ┤    ╭────────────────╯
 60 ┤───╯
 50 ┤
    └────┬────┬────┬────┬────┬────┬────┬────┬────┬────┬────┬────
        Feb  Mar  Apr  May  Jun  Jul  Aug  Sep  Oct  Nov  Dec  Jan

    ── IC Score    ┅┅ S&P 500 Avg Score
```

---

## 9. Implementation Roadmap

### Phase 1: Foundation (Weeks 1-2)

| Task | Owner | Effort |
|------|-------|--------|
| Create `sector_stats` table and population script | Backend | 2 days |
| Create `ic_scores_v2` and `ic_score_breakdowns` tables | Backend | 1 day |
| Implement percentile ranking functions | Backend | 2 days |
| Calculate sector statistics for all metrics | Backend | 2 days |
| Unit tests for percentile calculations | Backend | 1 day |

**Milestone**: Sector statistics available for all 11 GICS sectors.

### Phase 2: Factor Calculators (Weeks 3-4)

| Task | Owner | Effort |
|------|-------|--------|
| Implement Value factor calculator | Backend | 2 days |
| Implement Growth factor calculator | Backend | 1 day |
| Implement Quality factor calculator (with F-Score, Z-Score) | Backend | 3 days |
| Implement Momentum factor calculator | Backend | 1 day |
| Implement Sentiment factor calculator | Backend | 2 days |
| Implement Risk factor calculator | Backend | 1 day |
| Integration tests for factor calculators | Backend | 2 days |

**Milestone**: All 6 factor scores calculating correctly.

### Phase 3: Score Aggregation (Week 5)

| Task | Owner | Effort |
|------|-------|--------|
| Implement sector-specific weight profiles | Backend | 1 day |
| Implement market regime detection | Backend | 1 day |
| Implement confidence scoring | Backend | 1 day |
| Implement overall score aggregation | Backend | 1 day |
| Implement score summary text generation | Backend | 1 day |

**Milestone**: Full IC Score 2.0 calculation working.

### Phase 4: API & CronJob (Week 6)

| Task | Owner | Effort |
|------|-------|--------|
| Create FastAPI endpoints for IC Score 2.0 | Backend | 2 days |
| Create Kubernetes CronJob for daily calculation | DevOps | 1 day |
| Performance optimization (parallel processing) | Backend | 2 days |

**Milestone**: IC Score 2.0 API live, daily updates running.

### Phase 5: Frontend (Weeks 7-8)

| Task | Owner | Effort |
|------|-------|--------|
| IC Score Card component | Frontend | 2 days |
| Factor radar chart component | Frontend | 2 days |
| Factor breakdown expandable section | Frontend | 3 days |
| Score history chart | Frontend | 2 days |
| Integration with ticker page | Frontend | 1 day |

**Milestone**: IC Score 2.0 visible on all ticker pages.

### Phase 6: Validation & Launch (Weeks 9-10)

| Task | Owner | Effort |
|------|-------|--------|
| Manual validation: AAPL, NVDA, JNJ, MSFT, JPM, JNJ, XOM | QA | 3 days |
| Compare vs analyst consensus for 100 stocks | QA | 2 days |
| Fix any identified scoring issues | Backend | 3 days |
| Documentation | All | 2 days |
| Production deployment | DevOps | 1 day |

**Milestone**: IC Score 2.0 launched in production.

---

## 10. Appendices

### Appendix A: Validation Test Cases

| Stock | Sector | Expected Score Range | Key Factors |
|-------|--------|---------------------|-------------|
| NVDA | Technology | 80-90 | High growth, high quality |
| AAPL | Technology | 68-78 | Moderate growth, high quality |
| JNJ | Healthcare | 65-75 | Value + defensive |
| MSFT | Technology | 75-85 | Balanced profile |
| JPM | Financials | 65-75 | Value + quality |
| XOM | Energy | 55-70 | Cyclical, value play |
| TSLA | Consumer Disc. | 60-75 | High growth, high risk |

### Appendix B: Sector Weight Profiles (Full)

```python
SECTOR_WEIGHTS = {
    "Technology": {
        "value": 0.15, "growth": 0.20, "quality": 0.20,
        "momentum": 0.15, "sentiment": 0.15, "risk": 0.15
    },
    "Healthcare": {
        "value": 0.15, "growth": 0.20, "quality": 0.25,
        "momentum": 0.10, "sentiment": 0.15, "risk": 0.15
    },
    "Financials": {
        "value": 0.20, "growth": 0.10, "quality": 0.25,
        "momentum": 0.10, "sentiment": 0.15, "risk": 0.20
    },
    "Consumer Discretionary": {
        "value": 0.15, "growth": 0.20, "quality": 0.20,
        "momentum": 0.15, "sentiment": 0.20, "risk": 0.10
    },
    "Consumer Staples": {
        "value": 0.25, "growth": 0.10, "quality": 0.25,
        "momentum": 0.10, "sentiment": 0.15, "risk": 0.15
    },
    "Energy": {
        "value": 0.20, "growth": 0.10, "quality": 0.20,
        "momentum": 0.15, "sentiment": 0.15, "risk": 0.20
    },
    "Utilities": {
        "value": 0.25, "growth": 0.05, "quality": 0.25,
        "momentum": 0.10, "sentiment": 0.15, "risk": 0.20
    },
    "Real Estate": {
        "value": 0.25, "growth": 0.10, "quality": 0.20,
        "momentum": 0.10, "sentiment": 0.15, "risk": 0.20
    },
    "Industrials": {
        "value": 0.20, "growth": 0.15, "quality": 0.20,
        "momentum": 0.15, "sentiment": 0.15, "risk": 0.15
    },
    "Materials": {
        "value": 0.20, "growth": 0.15, "quality": 0.20,
        "momentum": 0.15, "sentiment": 0.15, "risk": 0.15
    },
    "Communication Services": {
        "value": 0.15, "growth": 0.20, "quality": 0.20,
        "momentum": 0.15, "sentiment": 0.20, "risk": 0.10
    }
}
```

### Appendix C: Grade Conversion

| Score Range | Grade |
|-------------|-------|
| 97-100 | A+ |
| 93-96 | A |
| 90-92 | A- |
| 87-89 | B+ |
| 83-86 | B |
| 80-82 | B- |
| 77-79 | C+ |
| 73-76 | C |
| 70-72 | C- |
| 60-69 | D |
| 0-59 | F |

### Appendix D: Competitor Feature Comparison

| Feature | YCharts | Morningstar | Zacks | Simply Wall St | Seeking Alpha | TipRanks | IC Score 2.0 |
|---------|---------|-------------|-------|----------------|---------------|----------|--------------|
| Sector-relative scoring | ✓ | ✓ | ✓ | ✗ | ✓ | ✗ | ✓ |
| Multi-factor breakdown | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Confidence/uncertainty | ✗ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ |
| Social sentiment | ✗ | ✗ | ✗ | ✗ | ✗ | ✓ | ✓ |
| Reddit sentiment | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✓ |
| Insider activity | ✗ | ✗ | ✗ | ✓ | ✗ | ✓ | ✓ |
| Institutional tracking | ✗ | ✗ | ✗ | ✗ | ✗ | ✓ | ✓ |
| Free tier available | ✗ | ✗ | ✓ | ✓ | ✓ | ✓ | ✓ |
| API access | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ |
| Historical scores | ✓ | ✓ | ✗ | ✗ | ✗ | ✗ | ✓ |

---

## References

### Competitor Resources
- [YCharts Scoring Models](https://go.ycharts.com/scoring-models)
- [Morningstar Quantitative Equity Ratings Methodology](https://www.morningstar.com/company/ratings)
- [Zacks Rank Methodology](https://www.zacks.com/stocks/zacks-rank)
- [Simply Wall St Analysis Model (GitHub)](https://github.com/SimplyWallSt/Company-Analysis-Model)
- [Seeking Alpha Quant Ratings FAQ](https://help.seekingalpha.com/premium/quant-ratings-and-factor-grades-faq)
- [TipRanks Smart Score](https://www.tipranks.com/glossary/s/smart-score)
- [AAII A+ Stock Grades](https://www.aaii.com/journal/article/193841-the-eight-metrics-aaii-uses-to-identify-quality-stocks)

### Academic Research
- Piotroski, J. (2000). "Value Investing: The Use of Historical Financial Statement Information"
- Altman, E. (1968). "Financial Ratios, Discriminant Analysis and the Prediction of Corporate Bankruptcy"
- Mohanram, P. (2005). "Separating Winners from Losers among Low Book-to-Market Stocks"

---

*Document Version: 1.0*
*Last Updated: January 31, 2026*
