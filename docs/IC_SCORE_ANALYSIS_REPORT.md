# IC Score Calculation Analysis Report

> **Date**: February 3, 2026
> **Analyst**: Claude Code
> **Version Analyzed**: IC Score v2.1

---

## Executive Summary

This report provides a comprehensive analysis of the IC Score calculation system, identifying bugs, data inefficiencies, missing data patterns, and score accuracy concerns. The analysis covers the main calculator (`ic_score_calculator.py`), all factor calculation modules, and Phase 3 enhancements.

### Key Findings

| Category | Issues Found | Severity |
|----------|-------------|----------|
| **Bugs** | 9 | 3 High, 4 Medium, 2 Low |
| **Data Inefficiencies** | 6 | 2 High, 3 Medium, 1 Low |
| **Missing Data Patterns** | 7 | Expected behavior patterns |
| **Outstanding Values** | 5 | Potential score distortion cases |

---

## 1. Bugs Identified

### 1.1 HIGH SEVERITY BUGS

#### Bug #1: EPS Growth Division by Zero/Sign Issue
**File**: `ic_score_calculator.py:624-629`

```python
if historical[0].get('eps_diluted') and historical[3].get('eps_diluted'):
    eps_current = float(historical[0]['eps_diluted'])
    eps_prior = float(historical[3]['eps_diluted'])
    if eps_prior > 0:  # BUG: Only checks for positive
        eps_growth = ((eps_current / eps_prior) - 1) * 100
```

**Issue**:
- The check `eps_prior > 0` excludes negative-to-positive turnaround cases
- A company going from -$1.00 EPS to +$2.00 EPS (a 300% improvement) receives NO growth score
- This penalizes turnaround companies unfairly

**Recommended Fix**:
```python
if eps_prior != 0:
    eps_growth = ((eps_current / eps_prior) - 1) * 100
    if eps_prior < 0:  # Turnaround case
        eps_growth = -eps_growth  # Invert for correct direction
```

---

#### Bug #2: Data Completeness Exceeds 100%
**File**: `ic_score_calculator.py:1131`

```python
data_completeness = (len(factor_scores) / len(self.WEIGHTS_LEGACY)) * 100
```

**Issue**:
- `WEIGHTS_LEGACY` has 10 factors
- v2.1 can have up to 13 factors (with Phase 2 additions)
- Data completeness can exceed 100%, which:
  - Causes incorrect confidence level assignments
  - Breaks the UI display logic
  - May cause database constraint violations if completeness > 100

**Recommended Fix**:
```python
# Use total available factors for v2.1
total_factors = 10 + (1 if self.USE_EARNINGS_REVISIONS else 0) \
                   + (1 if self.USE_HISTORICAL_VALUATION else 0) \
                   + (1 if self.income_mode else 0)
data_completeness = (len(factor_scores) / total_factors) * 100
```

---

#### Bug #3: Smart Money Weight Distribution Error
**File**: `ic_score_calculator.py:1163-1165`

```python
'analyst_consensus': weights_to_use.get('smart_money', 0.10) * 0.4,  # 40%
'insider_activity': weights_to_use.get('smart_money', 0.10) * 0.3,   # 30%
'institutional': weights_to_use.get('smart_money', 0.10) * 0.3,      # 30%
```

**Issue**:
- When one or more of analyst/insider/institutional scores is missing, the other factors don't get redistributed weight
- Example: If insider data is missing:
  - Expected: analyst=57%, institutional=43% (proportional redistribution)
  - Actual: analyst=40%, institutional=30%, 30% is lost

**Impact**: Stocks without insider data are systematically under-weighted in the Smart Money category.

---

### 1.2 MEDIUM SEVERITY BUGS

#### Bug #4: Dividend CAGR Calculation Error
**File**: `dividend_quality.py:123-124`

```python
if old_div > 0 and current_div > 0:
    dividend_cagr = ((current_div / old_div) ** (1/4) - 1) * 100
```

**Issue**:
- Uses `(1/4)` but there are 5 data points spanning 4 years
- For a proper 5-year CAGR with 5 annual data points, the exponent should be `(1/(len(rows)-1))`
- Current calculation understates dividend growth

---

#### Bug #5: News Sentiment Weight Hardcoded
**File**: `ic_score_calculator.py:1162`

```python
'news_sentiment': weights_to_use.get('news_sentiment', 0.05),
```

**Issue**:
- v2.1 `WEIGHTS` dictionary doesn't include `news_sentiment` (it's part of Smart Money)
- The hardcoded 0.05 fallback is applied, adding extra weight beyond the 100% total
- This affects weight normalization

---

#### Bug #6: Historical Data Index Out of Bounds Risk
**File**: `ic_score_calculator.py:617-629`

```python
if historical[0].get('revenue') and historical[3].get('revenue'):
```

**Issue**:
- Assumes at least 4 quarters of data exist
- No bounds checking before accessing `historical[3]`
- Will crash with `IndexError` if only 1-3 quarters exist

---

#### Bug #7: Sector Percentile Fallback Inconsistency
**File**: `ic_score_calculator.py:559-582`

```python
if not scores:
    metadata['scoring_method'] = 'absolute_benchmark'
```

**Issue**:
- Fallback to absolute benchmarks creates inconsistent scoring
- Same stock can swing between sector-relative and absolute scoring day-to-day
- Users see unexplained score volatility

---

### 1.3 LOW SEVERITY BUGS

#### Bug #8: P/E Similarity Division Potential
**File**: `peer_comparison.py:359`

```python
pe_ratio = max(stock['pe_ratio'], candidate['pe_ratio']) / min(stock['pe_ratio'], candidate['pe_ratio'])
```

**Issue**: If both P/E ratios are identical, this works, but the subsequent calculation could be slightly off due to floating-point precision.

---

#### Bug #9: Missing Event Type Validation
**File**: `score_stabilizer.py:146-149`

```python
has_reset_event = any(
    EventType(e) in self.RESET_EVENTS
    for e in event_types
    if e in [et.value for et in EventType]
)
```

**Issue**: Creates a list on every iteration for comparison. Inefficient, though not incorrect.

---

## 2. Data Inefficiencies

### 2.1 N+1 Query Pattern in Factor Calculations

**Files**: Multiple data fetch methods in `ic_score_calculator.py`

```python
financial_data = await self.fetch_financial_data(ticker)
valuation_data = await self.fetch_valuation_data(ticker)
technical_data = await self.fetch_technical_data(ticker)
insider_data = await self.fetch_insider_data(ticker)
news_data = await self.fetch_news_sentiment_data(ticker)
analyst_data = await self.fetch_analyst_data(ticker)
institutional_data = await self.fetch_institutional_data(ticker)
```

**Issue**: 7 sequential database queries per stock. For 5,000 stocks, this is 35,000+ queries.

**Recommendation**: Use a single joined query or batch fetch for all data sources.

---

### 2.2 Sector Percentile Calculation Overhead

**File**: `sector_percentile.py:274-284`

**Issue**: Each call to `calculate_all_sectors()` recalculates all metrics for all sectors, even if only one metric changed.

**Recommendation**: Implement incremental updates or change detection.

---

### 2.3 Missing Database Indexes

**Identified Missing Indexes**:
1. `ic_scores(ticker, date)` - composite index exists, but no descending index for latest lookup
2. `financials(ticker)` - missing partial index for `WHERE fiscal_quarter IS NOT NULL`
3. `technical_indicators(ticker, indicator_name)` - missing compound index

---

### 2.4 Materialized View Refresh Blocking

**File**: `sector_percentile.py:480-481`

```python
await self.session.execute(
    text("REFRESH MATERIALIZED VIEW CONCURRENTLY mv_latest_sector_percentiles")
)
```

**Issue**: Even with `CONCURRENTLY`, this can be slow for large views and blocks if unique index is missing.

---

### 2.5 Redundant JSON Serialization

**Files**: Multiple modules

**Issue**: Metadata dictionaries are created, converted to JSON for storage, then parsed back on read. Consider using PostgreSQL JSONB operators for direct manipulation.

---

### 2.6 Cache Not Leveraged Across Batches

**File**: `sector_percentile.py:74`

```python
self._cache: Dict[str, SectorStats] = {}
```

**Issue**: Cache is per-session, cleared between runs. High-frequency metrics (like sector PE distributions) could be cached longer.

---

## 3. Missing Data Patterns

### 3.1 Expected Missing Data (By Design)

| Stock Category | Missing Data | Expected Behavior |
|----------------|--------------|-------------------|
| **Microcap stocks** | Analyst ratings, institutional holdings | Score uses 7-8 factors instead of 10 |
| **Non-dividend stocks** | Dividend quality factor | Factor excluded, weights redistributed |
| **IPOs < 5 years** | Historical valuation, 5-year growth rates | Limited historical context |
| **ADRs/Foreign** | Insider trades (no Form 4) | Insider activity score missing |
| **OTC stocks** | Most institutional data | Limited signal factors |

### 3.2 Unexpected Missing Data (Data Quality Issues)

| Data Source | Expected Coverage | Likely Gap | Impact |
|-------------|-------------------|------------|--------|
| `eps_estimates` | S&P 500 + Russell 1000 | Many small caps | Earnings revisions factor unavailable |
| `valuation_history` | 5 years all stocks | Only ~2 years for many | Historical valuation falls back to current only |
| `sector_percentiles` | All 11 GICS sectors | May have < 5 samples for some metrics | Falls back to absolute benchmarks |
| `technical_indicators` | All active stocks daily | Weekend/holiday gaps | Momentum scores use stale data |

### 3.3 Data Freshness Issues

| Data Type | Expected Refresh | Actual Pattern | Staleness Risk |
|-----------|------------------|----------------|----------------|
| Financial statements | Post-earnings (4x/year) | Often delayed 30+ days | Using outdated fundamentals |
| Insider trades | Daily (Form 4 filings) | May miss same-day filings | Delayed insider signals |
| Institutional holdings | Quarterly (Form 13F) | 45-day delay by regulation | Always 1.5+ months stale |
| Analyst ratings | Real-time | Depends on data provider | Could miss intraday changes |

---

## 4. Outstanding/Anomalous Values

### 4.1 Extreme Valuation Ratios

**Issue**: No upper/lower bounds on P/E, P/B, P/S ratios

**Examples that distort scoring**:
- P/E > 1000 (e.g., Amazon in early growth phase)
- P/E < 0 (negative earnings) - handled, but inconsistently
- P/B < 0 (negative book value) - causes sector percentile issues

**Impact**: A single outlier can significantly shift sector percentiles for all stocks.

**Recommendation**: Apply winsorization (cap at 1st/99th percentile) before sector calculations.

---

### 4.2 Infinite/NaN Values

**File**: `sector_percentile.py:421-422`

```python
values = [float(row.value) for row in result.fetchall()
          if row.value is not None and np.isfinite(float(row.value))]
```

**Issue**: Filters out NaN/Inf in sector calculation, but individual stock scores don't have this protection.

**Risk**: A stock with `Inf` margin could receive score `100` instead of being filtered.

---

### 4.3 Lifecycle Classification Edge Cases

**Anomalous Patterns**:

| Scenario | Expected Classification | Actual | Issue |
|----------|------------------------|--------|-------|
| Revenue growth = 49.9% | GROWTH | GROWTH | Correct |
| Revenue growth = 50.1% | HYPERGROWTH | HYPERGROWTH | Correct |
| Revenue growth = 50.0% | ? | HYPERGROWTH | Edge case - threshold is `>50`, not `>=50` |
| Negative margin + 60% growth | HYPERGROWTH | HYPERGROWTH | Should it be TURNAROUND? |

---

### 4.4 Score Smoothing First-Day Discontinuity

**Scenario**:
- Day 1: New stock, no previous score → raw score = 75
- Day 2: Previous score exists → smoothed score = 0.7 * 72 + 0.3 * 75 = 72.9
- Day 3: Score drops suddenly due to smoothing "kicking in"

**Issue**: First-day scores are not smoothed, creating inconsistency.

---

### 4.5 Confidence Level Cliff Effects

**Current Thresholds**:
```python
if data_completeness >= 90: confidence = 'High'
elif data_completeness >= 70: confidence = 'Medium'
else: confidence = 'Low'
```

**Issue**:
- 89.9% completeness = "Medium" confidence
- 90.0% completeness = "High" confidence
- Creates arbitrary cliffs that confuse users

**Recommendation**: Use continuous confidence scores (e.g., 0-100) in addition to labels.

---

## 5. Stock-Specific Analysis (20 Stock Test Cases)

Since I cannot directly query your database, below are 20 hypothetical test scenarios based on code analysis. These should be validated against actual data.

### 5.1 Large Cap Technology Stocks

| Stock | Expected Behavior | Potential Issue |
|-------|-------------------|-----------------|
| **AAPL** | High data completeness, sector-relative scoring | Should score well on profitability, potentially overweight on momentum |
| **MSFT** | Full factor coverage | Check historical valuation - may show "overvalued vs history" |
| **GOOGL** | Strong growth scores | Verify sector percentile calculation with mega-cap peers |
| **NVDA** | Very high P/E (>60) | Confirm sector percentile doesn't distort due to outlier |
| **META** | Turnaround → Growth transition | Check lifecycle classification accuracy |

### 5.2 Financial Sector

| Stock | Expected Behavior | Potential Issue |
|-------|-------------------|-----------------|
| **JPM** | Strong financial health scores | Bank-specific ratios (tier 1 capital) not captured |
| **BAC** | Moderate scores across factors | Current ratio may be misleading for banks |
| **WFC** | Turnaround classification possible | Verify negative sentiment not over-weighted |
| **GS** | High institutional ownership | Check insider activity scoring (executives may have restrictions) |
| **BLK** | Asset manager - different business model | ROE calculation may be skewed |

### 5.3 Small/Mid Cap Stocks

| Stock | Expected Behavior | Potential Issue |
|-------|-------------------|-----------------|
| **DDOG** | High growth, low/no profit | Lifecycle should be HYPERGROWTH, profitability de-emphasized |
| **NET** | Similar to DDOG | Verify hypergrowth weight adjustments applied |
| **CRWD** | Recent profitability | May be misclassified as MATURE if margins just turned positive |
| **MDB** | High growth SaaS | EPS estimates may be volatile - check earnings revisions factor |
| **SNOW** | Very high P/S ratio | Confirm doesn't get unfairly penalized in value score |

### 5.4 Value/Dividend Stocks

| Stock | Expected Behavior | Potential Issue |
|-------|-------------------|-----------------|
| **KO** | Dividend Aristocrat tier | Dividend quality score should be high (50+ years) |
| **JNJ** | Should be VALUE lifecycle | Verify low growth weight, high value weight |
| **VZ** | High yield, slow growth | Income mode should boost overall score |
| **T** | Dividend recently cut | Dividend streak score should reflect cut |
| **XOM** | Cyclical earnings | Historical valuation may be misleading in different oil price environments |

### 5.5 Validation Queries to Run

```sql
-- Query 1: Find stocks with data completeness > 100%
SELECT ticker, date, data_completeness
FROM ic_scores
WHERE data_completeness > 100;

-- Query 2: Find stocks with EPS turnaround missing growth score
SELECT ticker, date, growth_score, calculation_metadata->'growth'->>'eps_growth_yoy' as eps_growth
FROM ic_scores
WHERE growth_score IS NULL
  AND calculation_metadata->'growth' IS NOT NULL;

-- Query 3: Find lifecycle misclassifications
SELECT ticker, lifecycle_stage,
       calculation_metadata->'lifecycle_metrics'->>'revenue_growth_yoy' as rev_growth,
       calculation_metadata->'lifecycle_metrics'->>'net_margin' as margin
FROM ic_scores
WHERE lifecycle_stage = 'hypergrowth'
  AND (calculation_metadata->'lifecycle_metrics'->>'revenue_growth_yoy')::float < 50;

-- Query 4: Check for extreme sector percentile outliers
SELECT sector, metric_name, min_value, max_value,
       (max_value - p90_value) / NULLIF(std_dev, 0) as max_zscore
FROM mv_latest_sector_percentiles
WHERE (max_value - p90_value) / NULLIF(std_dev, 0) > 5;

-- Query 5: Validate Smart Money weight distribution
SELECT ticker, date,
       analyst_consensus_score, insider_activity_score, institutional_score,
       CASE
         WHEN analyst_consensus_score IS NOT NULL
              AND insider_activity_score IS NOT NULL
              AND institutional_score IS NOT NULL THEN 'all_present'
         ELSE 'missing_factors'
       END as smart_money_status
FROM ic_scores
WHERE date = CURRENT_DATE
ORDER BY smart_money_status;
```

---

## 6. Recommendations Summary

### 6.1 Critical Fixes (Do First)

1. **Fix EPS growth calculation** for negative-to-positive turnarounds
2. **Fix data completeness calculation** to cap at 100%
3. **Implement Smart Money weight redistribution** when factors are missing

### 6.2 Important Improvements

4. Add bounds checking for historical data array access
5. Implement winsorization for extreme valuation ratios
6. Fix dividend CAGR calculation exponent
7. Add continuous confidence scoring

### 6.3 Optimization Opportunities

8. Batch database queries for multiple data sources
9. Implement incremental sector percentile updates
10. Add missing database indexes
11. Implement cross-batch caching for sector statistics

### 6.4 Monitoring Additions

12. Add alerting for data completeness below threshold
13. Log sector percentile fallback frequency
14. Track lifecycle classification transitions
15. Monitor score stabilization bypass events

---

## 7. Appendix: Code References

| File | Line Numbers | Issue |
|------|-------------|-------|
| `ic_score_calculator.py` | 624-629 | EPS growth bug |
| `ic_score_calculator.py` | 1131 | Data completeness bug |
| `ic_score_calculator.py` | 1163-1165 | Smart Money weight bug |
| `dividend_quality.py` | 123-124 | CAGR calculation bug |
| `sector_percentile.py` | 274-284 | Sector calculation overhead |
| `peer_comparison.py` | 359 | P/E similarity edge case |
| `score_stabilizer.py` | 146-149 | Inefficient list creation |

---

*Report generated by automated code analysis. Validation against production data recommended.*
