# IC Score Intelligent Design Proposal

**Date**: January 31, 2026
**Purpose**: Propose smarter, more adaptive approaches to IC Score calculation

---

## The Problem with Hardcoded Values

The current IC Score system has fundamental design issues:

| Problem | Example | Impact |
|---------|---------|--------|
| **Static benchmarks** | P/E benchmark of 15 | Outdated within months |
| **One-size-fits-all** | Same formula for NVDA and JNJ | Apples-to-oranges comparison |
| **Linear assumptions** | `score = 100 - (pe - 15) * 2` | Reality is non-linear |
| **No context** | Ignores market conditions | Bull/bear markets treated same |
| **Arbitrary weights** | Growth: 15%, Value: 12% | Why these numbers? |

**Core Insight**: The best investors don't use fixed formulas. They adapt to context, compare against relevant peers, and weight factors based on what matters now.

---

## Proposed Smart Approaches

### Approach 1: Percentile-Based Relative Scoring

**Concept**: Don't ask "Is P/E 30 good?" Ask "Is P/E 30 good for a high-growth software company?"

**Implementation**:
```
Instead of: pe_score = 100 - (pe - 15) * 2

Use: pe_score = 100 - sector_percentile_rank(pe)
     where lower percentile = better value
```

**How it works**:
1. Group stocks by sector (or sub-industry)
2. Calculate percentile rank for each metric within group
3. P/E of 30 might be:
   - 20th percentile in Software (good!) → Score: 80
   - 80th percentile in Utilities (bad!) → Score: 20

**Benefits**:
- No hardcoded benchmarks
- Automatically adapts to market conditions
- Sector-appropriate comparisons
- Easy to understand ("Top 20% in sector")

**Example Calculation for NVDA**:
```
Sector: Technology (500 stocks)
NVDA P/E: 46.21
Sector P/E range: 10 - 200
NVDA percentile: 35th (better than 65% of tech stocks)
P/E Score: 65 (not 37.6 with hardcoded formula)
```

---

### Approach 2: Z-Score Normalization

**Concept**: Measure how many standard deviations from the sector mean.

**Implementation**:
```python
def zscore_to_score(value, sector_mean, sector_std, lower_is_better=True):
    z = (value - sector_mean) / sector_std
    if lower_is_better:
        z = -z  # Flip so lower values get positive z
    # Convert z-score to 0-100 scale
    # z = -2 → 0, z = 0 → 50, z = +2 → 100
    return max(0, min(100, 50 + z * 25))
```

**Benefits**:
- Statistically rigorous
- Handles outliers naturally
- No arbitrary cutoffs
- Adapts to sector characteristics

**Example**:
```
Tech Sector P/E: mean=35, std=20
NVDA P/E: 46.21
Z-score: (46.21 - 35) / 20 = 0.56
Since lower P/E is better, flip: -0.56
Score: 50 + (-0.56) * 25 = 36

BUT if we also consider:
Tech Sector PEG: mean=1.5, std=0.8
NVDA PEG: 0.73
Z-score: (0.73 - 1.5) / 0.8 = -0.96 (lower is better, so good!)
Score: 50 + 0.96 * 25 = 74
```

---

### Approach 3: Dynamic Peer Groups

**Concept**: Don't compare NVDA to all tech stocks. Compare to companies with similar characteristics.

**Implementation**:
```python
def find_peer_group(ticker):
    """Find 20-50 most similar companies for comparison."""
    similarity_factors = [
        'market_cap_range',      # Within 0.5x - 2x market cap
        'revenue_growth_range',  # Within 10% of growth rate
        'gross_margin_range',    # Within 10% of margin
        'business_model',        # Same sub-industry
    ]

    # Use cosine similarity or clustering
    peers = find_similar_stocks(ticker, factors=similarity_factors, n=30)
    return peers
```

**NVDA Peer Group Example**:
Instead of comparing to all 500 tech stocks:
- AMD, AVGO, QCOM (semiconductors)
- MSFT, GOOGL (high-margin tech)
- Similar market cap range
- Similar growth profiles

**Benefits**:
- True apples-to-apples comparison
- NVDA's P/E of 46 might be lowest in its peer group
- More meaningful rankings

---

### Approach 4: Machine Learning Ranking Model

**Concept**: Let the data tell us what predicts good investments.

**Implementation Options**:

#### Option A: Learning to Rank (LTR)
```python
# Train a model to rank stocks by future returns
from lightgbm import LGBMRanker

features = [
    'pe_ratio', 'pb_ratio', 'ps_ratio', 'peg_ratio',
    'roe', 'roa', 'net_margin', 'gross_margin',
    'revenue_growth', 'eps_growth',
    'debt_to_equity', 'current_ratio',
    'analyst_buy_pct', 'insider_net_buying_pct',
    'momentum_3m', 'momentum_12m',
    'sector_encoded', 'market_cap_log',
]

# Target: Future 6-month return quintile (1-5)
model = LGBMRanker(objective='lambdarank')
model.fit(X_train, y_train, group=sector_groups)

# Prediction gives ranking score
ic_score = model.predict(current_features)
```

#### Option B: Classification Model
```python
# Predict probability of outperforming market
from sklearn.ensemble import GradientBoostingClassifier

# Target: 1 if stock beat S&P 500 by >5% in next 6 months
model = GradientBoostingClassifier()
model.fit(X_train, y_train)

# IC Score = probability of outperformance * 100
ic_score = model.predict_proba(current_features)[:, 1] * 100
```

**Benefits**:
- Learns optimal feature combinations
- Captures non-linear relationships
- Weights emerge from data, not intuition
- Can incorporate many more features

**Challenges**:
- Requires historical data (5+ years)
- Risk of overfitting
- Needs regular retraining
- Less interpretable

---

### Approach 5: Dynamic Factor Weights

**Concept**: Factor importance changes with market conditions.

**Implementation**:
```python
def get_dynamic_weights(market_regime):
    """Adjust factor weights based on market environment."""

    # Base weights
    weights = {
        'value': 0.12,
        'growth': 0.15,
        'profitability': 0.12,
        'financial_health': 0.10,
        'momentum': 0.08,
        'analyst': 0.10,
        'insider': 0.08,
        'institutional': 0.10,
        'sentiment': 0.07,
        'technical': 0.08,
    }

    if market_regime == 'high_volatility':  # VIX > 25
        # Safety matters more
        weights['financial_health'] *= 1.5
        weights['value'] *= 1.3
        weights['momentum'] *= 0.5
        weights['growth'] *= 0.7

    elif market_regime == 'bull_market':  # SPY up >15% YTD
        # Growth and momentum matter more
        weights['growth'] *= 1.4
        weights['momentum'] *= 1.5
        weights['value'] *= 0.7

    elif market_regime == 'rising_rates':  # 10Y yield rising
        # Value and profitability matter more
        weights['value'] *= 1.3
        weights['profitability'] *= 1.3
        weights['growth'] *= 0.8

    # Normalize to sum to 1.0
    total = sum(weights.values())
    return {k: v/total for k, v in weights.items()}
```

**Market Regime Detection**:
```python
def detect_market_regime():
    vix = get_current_vix()
    spy_ytd = get_spy_ytd_return()
    yield_10y_change = get_10y_yield_3m_change()

    if vix > 30:
        return 'crisis'
    elif vix > 25:
        return 'high_volatility'
    elif spy_ytd > 15:
        return 'bull_market'
    elif yield_10y_change > 0.5:
        return 'rising_rates'
    else:
        return 'normal'
```

---

### Approach 6: LLM-Enhanced Qualitative Analysis

**Concept**: Use AI to analyze qualitative factors that numbers miss.

**Implementation**:
```python
async def get_llm_analysis(ticker: str) -> dict:
    """Use LLM to analyze qualitative factors."""

    # Gather context
    earnings_transcript = await fetch_latest_earnings_call(ticker)
    recent_news = await fetch_recent_news(ticker, days=30)
    sec_filings = await fetch_recent_8k_filings(ticker)

    prompt = f"""
    Analyze {ticker} and provide scores (0-100) for:

    1. Management Quality: Based on earnings call clarity,
       guidance accuracy, capital allocation decisions

    2. Competitive Position: Moat strength, market share trends,
       pricing power, threat from competitors

    3. Business Model Quality: Recurring revenue, customer
       concentration, switching costs, scalability

    4. Risk Factors: Regulatory, litigation, concentration,
       macro sensitivity

    5. Growth Catalysts: New products, market expansion,
       M&A potential, secular tailwinds

    Earnings Call Excerpts:
    {earnings_transcript[:5000]}

    Recent News:
    {recent_news[:3000]}

    Provide JSON with scores and brief reasoning.
    """

    response = await llm.analyze(prompt)
    return parse_llm_scores(response)
```

**Integration with IC Score**:
```python
def calculate_enhanced_ic_score(ticker):
    # Quantitative scores (existing)
    quant_scores = calculate_quantitative_scores(ticker)

    # Qualitative scores (LLM)
    qual_scores = get_llm_analysis(ticker)

    # Blend: 70% quantitative, 30% qualitative
    final_score = (
        0.70 * weighted_average(quant_scores) +
        0.30 * weighted_average(qual_scores)
    )

    return final_score
```

**What LLM Can Capture**:
- Management credibility and communication style
- Competitive dynamics not in financials
- Emerging risks mentioned in filings
- Market sentiment and narrative shifts
- Quality of growth (organic vs acquired)

---

### Approach 7: Multi-Timeframe Composite Score

**Concept**: Different metrics matter for different investment horizons.

**Implementation**:
```python
def calculate_timeframe_scores(ticker):
    """Calculate scores optimized for different holding periods."""

    # Short-term (1-3 months): Technical + Momentum + Sentiment
    short_term = (
        0.40 * technical_score +
        0.35 * momentum_score +
        0.25 * sentiment_score
    )

    # Medium-term (3-12 months): Growth + Analyst + Value
    medium_term = (
        0.30 * growth_score +
        0.30 * analyst_score +
        0.25 * value_score +
        0.15 * momentum_score
    )

    # Long-term (1-5 years): Fundamentals + Quality + Value
    long_term = (
        0.30 * profitability_score +
        0.25 * financial_health_score +
        0.25 * value_score +
        0.20 * growth_score
    )

    return {
        'short_term': short_term,
        'medium_term': medium_term,
        'long_term': long_term,
        'composite': (short_term + medium_term + long_term) / 3
    }
```

**User Interface**:
```
NVDA IC Scores by Horizon:
├── Short-term (Trader):    78/100 ⭐⭐⭐⭐
├── Medium-term (Investor): 85/100 ⭐⭐⭐⭐⭐
├── Long-term (Holder):     82/100 ⭐⭐⭐⭐
└── Composite:              82/100 ⭐⭐⭐⭐
```

---

### Approach 8: Bayesian Scoring with Uncertainty

**Concept**: Express confidence in scores, not just point estimates.

**Implementation**:
```python
def bayesian_score(ticker, metric, observed_value):
    """Calculate score with uncertainty bounds."""

    # Prior: Sector average and std
    sector_mean, sector_std = get_sector_stats(ticker, metric)

    # Likelihood: Company's historical consistency
    company_mean, company_std = get_company_history(ticker, metric)

    # Posterior: Combine prior and likelihood
    # More historical data = more weight on company-specific
    data_points = get_data_point_count(ticker, metric)

    if data_points < 4:
        # Limited data: rely more on sector prior
        weight_company = 0.3
    elif data_points < 12:
        weight_company = 0.6
    else:
        weight_company = 0.85

    posterior_mean = (
        weight_company * company_mean +
        (1 - weight_company) * sector_mean
    )
    posterior_std = (
        weight_company * company_std +
        (1 - weight_company) * sector_std
    )

    # Score with confidence interval
    score = calculate_score(observed_value, posterior_mean, posterior_std)
    confidence_low = score - 1.96 * (posterior_std / sector_std) * 10
    confidence_high = score + 1.96 * (posterior_std / sector_std) * 10

    return {
        'score': score,
        'confidence_interval': (confidence_low, confidence_high),
        'data_quality': 'high' if data_points >= 12 else 'medium' if data_points >= 4 else 'low'
    }
```

**User Interface**:
```
NVDA Value Score: 65 (±8)
├── Confidence: High (5 years of data)
├── Range: 57 - 73
└── Sector Rank: Top 35%

JNJ Value Score: 72 (±5)
├── Confidence: Very High (10+ years of data)
├── Range: 67 - 77
└── Sector Rank: Top 28%
```

---

## Recommended Architecture

### Hybrid System Design

```
┌─────────────────────────────────────────────────────────────┐
│                    IC SCORE ENGINE v2.0                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │   Layer 1   │    │   Layer 2   │    │   Layer 3   │     │
│  │  Relative   │    │  ML-Based   │    │    LLM      │     │
│  │  Scoring    │    │  Ranking    │    │  Analysis   │     │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘     │
│         │                  │                  │             │
│         └────────────┬─────┴──────────────────┘             │
│                      ▼                                      │
│              ┌──────────────┐                               │
│              │   Ensemble   │                               │
│              │   Combiner   │                               │
│              └──────┬───────┘                               │
│                     │                                       │
│         ┌───────────┼───────────┐                          │
│         ▼           ▼           ▼                          │
│    ┌─────────┐ ┌─────────┐ ┌─────────┐                     │
│    │ Short   │ │ Medium  │ │  Long   │                     │
│    │  Term   │ │  Term   │ │  Term   │                     │
│    │ Score   │ │ Score   │ │ Score   │                     │
│    └─────────┘ └─────────┘ └─────────┘                     │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Implementation Priority

| Phase | Component | Effort | Impact |
|-------|-----------|--------|--------|
| **Phase 1** | Percentile-based scoring | 1 week | High |
| **Phase 1** | Dynamic peer groups | 1 week | High |
| **Phase 2** | Dynamic factor weights | 3 days | Medium |
| **Phase 2** | Multi-timeframe scores | 3 days | Medium |
| **Phase 3** | ML ranking model | 2-3 weeks | High |
| **Phase 3** | Bayesian uncertainty | 1 week | Medium |
| **Phase 4** | LLM qualitative analysis | 2-3 weeks | High |

---

## Concrete Example: NVDA Scoring

### Current System (Hardcoded)
```
P/E: 46.21 vs benchmark 15 → Score: 37.6
P/B: 38.29 vs benchmark 2 → Score: 0
P/S: 24.92 vs benchmark 2 → Score: 0
Value Score: 12.5 (POOR)

Problem: NVDA looks "expensive" by outdated standards
```

### Proposed System (Intelligent)

**Layer 1: Percentile Scoring**
```
Peer Group: High-growth semiconductors (AMD, AVGO, MRVL, etc.)
P/E percentile: 35th (cheaper than 65% of peers) → Score: 65
P/B percentile: 45th → Score: 55
P/S percentile: 40th → Score: 60
PEG percentile: 15th (much cheaper!) → Score: 85
Value Score: 66.25 (GOOD)
```

**Layer 2: ML Ranking**
```
Features: [pe, pb, ps, peg, roe, growth, momentum, ...]
Model prediction: 82nd percentile expected return
ML Score: 82 (EXCELLENT)
```

**Layer 3: LLM Analysis**
```
Management Quality: 88 (Jensen Huang highly regarded)
Competitive Position: 92 (dominant AI/GPU moat)
Growth Catalysts: 95 (AI supercycle, datacenter expansion)
Risk Factors: 65 (China exposure, customer concentration)
Qualitative Score: 85 (EXCELLENT)
```

**Ensemble Combination**
```
Layer 1 (Percentile): 66.25 × 0.40 = 26.5
Layer 2 (ML):         82.00 × 0.35 = 28.7
Layer 3 (LLM):        85.00 × 0.25 = 21.25

Final IC Score: 76.5 → Adjusted for regime: 84
Rating: STRONG BUY
Confidence: High (all layers agree)
```

### Comparison

| Approach | NVDA Score | Rating | Accuracy |
|----------|------------|--------|----------|
| Current (hardcoded) | 72.7 | BUY | Poor |
| Proposed (intelligent) | 84.0 | STRONG BUY | Good |
| Claude Analysis | 85.0 | STRONG BUY | Reference |

---

## Data Requirements

### For Percentile Scoring
- Daily updated sector/industry classifications
- Rolling statistics (mean, std, percentiles) per sector
- 500+ stocks minimum per major sector

### For ML Model
- 5+ years historical data
- Quarterly fundamental snapshots
- Forward returns (6-12 months)
- ~10,000+ training examples

### For LLM Analysis
- Earnings call transcripts (API or scraping)
- SEC filings (EDGAR API)
- News feed (financial news API)
- ~$0.05-0.10 per stock analysis (API costs)

---

## Summary: From Dumb to Smart

| Aspect | Current (Dumb) | Proposed (Smart) |
|--------|----------------|------------------|
| **Benchmarks** | Hardcoded (P/E=15) | Dynamic (sector percentiles) |
| **Comparison** | All stocks same | Peer group specific |
| **Weights** | Fixed (Growth=15%) | Regime-adaptive |
| **Metrics** | Only quantitative | Quant + Qualitative (LLM) |
| **Output** | Point estimate | Score + Confidence |
| **Timeframe** | One-size-fits-all | Short/Medium/Long term |
| **Learning** | None | Continuous (ML) |

**Key Principle**: The best IC Score system should think like a skilled analyst, not calculate like a spreadsheet.

---

## Next Steps

1. **Immediate**: Implement percentile-based scoring (biggest bang for buck)
2. **Short-term**: Build peer group matching algorithm
3. **Medium-term**: Train and validate ML ranking model
4. **Long-term**: Integrate LLM qualitative analysis

Would you like me to elaborate on any specific approach or create a more detailed implementation plan for any of these?
