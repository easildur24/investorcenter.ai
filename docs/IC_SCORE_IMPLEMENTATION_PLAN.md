# IC Score v2.1 Implementation Plan

> **Version**: 1.0
> **Date**: February 2026
> **Status**: Ready for Review
> **Timeline**: 20 weeks (6 phases)

---

## Table of Contents

1. [Overview](#overview)
2. [Phase 1: Foundation (Weeks 1-3)](#phase-1-foundation-weeks-1-3)
3. [Phase 2: New Factors (Weeks 4-6)](#phase-2-new-factors-weeks-4-6)
4. [Phase 3: Enhanced Features (Weeks 7-9)](#phase-3-enhanced-features-weeks-7-9)
5. [Phase 4: UI Enhancement (Weeks 10-12)](#phase-4-ui-enhancement-weeks-10-12)
6. [Phase 5: Validation & Launch (Weeks 13-16)](#phase-5-validation--launch-weeks-13-16)
7. [Phase 6: Personalization (Weeks 17-20)](#phase-6-personalization-weeks-17-20)
8. [Database Migrations](#database-migrations)
9. [API Changes](#api-changes)
10. [Testing Strategy](#testing-strategy)
11. [Rollout Strategy](#rollout-strategy)
12. [Risk Mitigation](#risk-mitigation)

---

## 1. Overview

### 1.1 Goals

Transform IC Score from absolute-benchmark scoring to a sector-relative, lifecycle-aware system with enhanced transparency features.

### 1.2 Key Deliverables

| Deliverable | Phase | Priority |
|-------------|-------|----------|
| Sector-relative scoring | 1 | P0 |
| Earnings Revisions factor | 2 | P0 |
| Historical Valuation factor | 2 | P0 |
| Score Stability mechanism | 3 | P0 |
| Peer Comparison | 3 | P0 |
| Catalyst Indicators | 3 | P1 |
| Backtest validation | 5 | P0 |
| Dividend Quality (optional) | 2 | P2 |

### 1.3 Team Requirements

| Role | Phase 1-3 | Phase 4-6 |
|------|-----------|-----------|
| Backend Engineer | 2 | 1 |
| IC Score Service Engineer | 2 | 1 |
| Frontend Engineer | 0 | 2 |
| Data Scientist | 0.5 | 1 |
| QA Engineer | 0.5 | 1 |

---

## 2. Phase 1: Foundation (Weeks 1-3)

### 2.1 Sprint 1 (Week 1): Sector Percentile Infrastructure

#### Database Changes

```sql
-- Migration: 001_add_sector_percentiles.sql

CREATE TABLE sector_percentiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sector VARCHAR(50) NOT NULL,
    metric_name VARCHAR(50) NOT NULL,
    calculated_at DATE NOT NULL DEFAULT CURRENT_DATE,

    -- Distribution statistics
    min_value NUMERIC(20,4),
    p10_value NUMERIC(20,4),
    p25_value NUMERIC(20,4),
    p50_value NUMERIC(20,4),  -- median
    p75_value NUMERIC(20,4),
    p90_value NUMERIC(20,4),
    max_value NUMERIC(20,4),
    mean_value NUMERIC(20,4),
    std_dev NUMERIC(20,4),
    sample_count INTEGER,

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (sector, metric_name, calculated_at)
);

CREATE INDEX idx_sector_percentiles_lookup
ON sector_percentiles(sector, metric_name, calculated_at DESC);

-- Materialized view for quick sector lookups
CREATE MATERIALIZED VIEW mv_latest_sector_percentiles AS
SELECT DISTINCT ON (sector, metric_name)
    sector,
    metric_name,
    p10_value,
    p25_value,
    p50_value,
    p75_value,
    p90_value,
    sample_count
FROM sector_percentiles
ORDER BY sector, metric_name, calculated_at DESC;

CREATE UNIQUE INDEX idx_mv_sector_percentiles
ON mv_latest_sector_percentiles(sector, metric_name);
```

#### IC Score Service Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Create `SectorPercentileCalculator` class | IC-1 | 8 | None |
| Implement metric extraction for all sectors | IC-1 | 16 | Above |
| Add percentile calculation logic with outlier handling | IC-1 | 8 | Above |
| Create K8s CronJob for daily sector stats refresh | IC-1 | 4 | Above |
| Unit tests for percentile calculations | IC-1 | 8 | Above |

#### Backend Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Add sector_percentiles table migration | BE-1 | 2 | None |
| Create SectorPercentile model in Go | BE-1 | 4 | Migration |
| Add repository methods for sector percentiles | BE-1 | 4 | Model |
| Add caching layer for sector percentiles (Redis) | BE-1 | 6 | Repository |

**Sprint 1 Definition of Done**:
- [ ] Sector percentiles calculated daily for all 11 sectors
- [ ] 20+ metrics per sector (valuation, growth, profitability, etc.)
- [ ] Percentile lookup API available internally
- [ ] Unit test coverage >80%

---

### 2.2 Sprint 2 (Week 2): Factor Refactoring

#### IC Score Service Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Refactor `calculate_value_score` to use sector percentiles | IC-1 | 8 | Sprint 1 |
| Refactor `calculate_growth_score` to use sector percentiles | IC-1 | 6 | Sprint 1 |
| Refactor `calculate_profitability_score` to use sector percentiles | IC-1 | 6 | Sprint 1 |
| Update `calculate_momentum_score` for sector-relative | IC-2 | 6 | Sprint 1 |
| Add `sector_percentile()` utility function | IC-1 | 4 | Sprint 1 |
| Integration tests for refactored factors | IC-2 | 8 | All above |

#### Code Changes

```python
# ic-score-service/utils/sector_percentile.py

from typing import Optional
import numpy as np

class SectorPercentileCalculator:
    """Calculate sector-relative percentiles for stock metrics."""

    LOWER_IS_BETTER = {
        'pe_ratio', 'ps_ratio', 'pb_ratio', 'ev_ebitda', 'peg_ratio',
        'debt_to_equity'
    }

    def __init__(self, db):
        self.db = db
        self._cache = {}

    async def get_percentile(
        self,
        sector: str,
        metric: str,
        value: float
    ) -> Optional[float]:
        """
        Get percentile score (0-100) for a value within its sector.

        Returns higher scores for "better" values:
        - For most metrics: higher value = higher percentile
        - For valuation metrics: lower value = higher percentile (inverted)
        """
        stats = await self._get_sector_stats(sector, metric)
        if not stats:
            return None

        # Calculate raw percentile
        raw_pct = self._calculate_percentile(value, stats)

        # Invert for "lower is better" metrics
        if metric in self.LOWER_IS_BETTER:
            return 100 - raw_pct

        return raw_pct

    def _calculate_percentile(self, value: float, stats: dict) -> float:
        """Interpolate percentile based on distribution stats."""
        if value <= stats['p10']:
            return 10 * (value - stats['min']) / (stats['p10'] - stats['min'] + 0.001)
        elif value <= stats['p25']:
            return 10 + 15 * (value - stats['p10']) / (stats['p25'] - stats['p10'] + 0.001)
        elif value <= stats['p50']:
            return 25 + 25 * (value - stats['p25']) / (stats['p50'] - stats['p25'] + 0.001)
        elif value <= stats['p75']:
            return 50 + 25 * (value - stats['p50']) / (stats['p75'] - stats['p50'] + 0.001)
        elif value <= stats['p90']:
            return 75 + 15 * (value - stats['p75']) / (stats['p90'] - stats['p75'] + 0.001)
        else:
            return min(100, 90 + 10 * (value - stats['p90']) / (stats['max'] - stats['p90'] + 0.001))
```

**Sprint 2 Definition of Done**:
- [ ] All 4 fundamental factors use sector percentiles
- [ ] Scores for same stock differ by sector context
- [ ] Backward compatibility maintained (old API still works)
- [ ] Performance: <50ms per factor calculation

---

### 2.3 Sprint 3 (Week 3): Lifecycle Classification

#### IC Score Service Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Create `LifecycleClassifier` class | IC-2 | 8 | None |
| Implement classification logic (5 stages) | IC-2 | 8 | Above |
| Create lifecycle weight adjustment logic | IC-2 | 6 | Above |
| Add lifecycle to IC Score calculation pipeline | IC-2 | 4 | Above |
| Integration tests for lifecycle classification | IC-2 | 6 | All above |

#### Code Implementation

```python
# ic-score-service/classifiers/lifecycle.py

from enum import Enum
from typing import Dict

class LifecycleStage(Enum):
    HYPERGROWTH = "hypergrowth"
    GROWTH = "growth"
    MATURE = "mature"
    VALUE = "value"
    TURNAROUND = "turnaround"

class LifecycleClassifier:
    """Classify companies into lifecycle stages for weight adjustment."""

    # Weight multipliers for each lifecycle stage
    WEIGHT_ADJUSTMENTS = {
        LifecycleStage.HYPERGROWTH: {
            'growth': 1.4,
            'profitability': 0.6,
            'value': 0.5,
            'intrinsic_value': 0.5,
            'historical_value': 0.5,
        },
        LifecycleStage.GROWTH: {
            'growth': 1.2,
            'profitability': 0.9,
            'value': 0.8,
        },
        LifecycleStage.MATURE: {
            'profitability': 1.1,
            'value': 1.1,
            'financial_health': 1.1,
            'historical_value': 1.1,
        },
        LifecycleStage.VALUE: {
            'value': 1.3,
            'intrinsic_value': 1.2,
            'historical_value': 1.2,
            'profitability': 1.1,
            'growth': 0.7,
        },
        LifecycleStage.TURNAROUND: {
            'financial_health': 1.3,
            'momentum': 1.2,
            'value': 1.2,
            'growth': 0.8,
        },
    }

    def classify(self, data: Dict) -> LifecycleStage:
        """Classify company based on financial metrics."""
        revenue_growth = data.get('revenue_growth_yoy', 0) or 0
        net_margin = data.get('net_margin', 0) or 0
        pe_ratio = data.get('pe_ratio', 20) or 20

        if revenue_growth > 50:
            return LifecycleStage.HYPERGROWTH
        elif revenue_growth > 20:
            return LifecycleStage.GROWTH
        elif revenue_growth < -5:
            return LifecycleStage.TURNAROUND
        elif pe_ratio < 12 and net_margin > 5:
            return LifecycleStage.VALUE
        else:
            return LifecycleStage.MATURE

    def adjust_weights(
        self,
        base_weights: Dict[str, float],
        lifecycle: LifecycleStage
    ) -> Dict[str, float]:
        """Adjust factor weights based on lifecycle stage."""
        adjustments = self.WEIGHT_ADJUSTMENTS.get(lifecycle, {})

        adjusted = {}
        for factor, weight in base_weights.items():
            multiplier = adjustments.get(factor, 1.0)
            adjusted[factor] = weight * multiplier

        # Normalize to sum to 1.0
        total = sum(adjusted.values())
        return {k: v / total for k, v in adjusted.items()}
```

#### Backend API Changes

```go
// backend/handlers/ic_score_handlers.go - Updated response

type ICScoreResponse struct {
    Ticker          string  `json:"ticker"`
    OverallScore    float64 `json:"overall_score"`
    Rating          string  `json:"rating"`

    // NEW: Lifecycle and sector context
    LifecycleStage  string  `json:"lifecycle_stage"`
    Sector          string  `json:"sector"`
    SectorRank      int     `json:"sector_rank"`
    SectorPercentile float64 `json:"sector_percentile"`

    // Existing fields...
    Categories      map[string]CategoryScore `json:"categories"`
    Factors         []FactorScore            `json:"factors"`
    Confidence      ConfidenceInfo           `json:"confidence"`
}
```

**Sprint 3 Definition of Done**:
- [ ] Lifecycle classification working for all stocks
- [ ] Weight adjustments applied correctly
- [ ] API returns lifecycle_stage field
- [ ] API returns sector_rank and sector_percentile
- [ ] End-to-end test with sample stocks

---

## 3. Phase 2: New Factors (Weeks 4-6)

### 3.1 Sprint 4 (Week 4): Earnings Revisions Factor

#### Data Pipeline Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Research EPS estimate data sources (FMP, Polygon) | IC-1 | 4 | None |
| Create `eps_estimates` table migration | BE-1 | 2 | None |
| Build EPS estimates ingestion pipeline | IC-1 | 12 | Table |
| Schedule daily EPS estimate updates | IC-1 | 4 | Pipeline |

#### Database Migration

```sql
-- Migration: 002_add_eps_estimates.sql

CREATE TABLE eps_estimates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    fiscal_year INTEGER NOT NULL,
    fiscal_quarter INTEGER,  -- NULL for annual estimates

    -- Estimate data
    consensus_eps NUMERIC(10,4),
    num_analysts INTEGER,
    high_estimate NUMERIC(10,4),
    low_estimate NUMERIC(10,4),

    -- Historical tracking
    estimate_30d_ago NUMERIC(10,4),
    estimate_60d_ago NUMERIC(10,4),
    estimate_90d_ago NUMERIC(10,4),

    -- Revision counts
    upgrades_30d INTEGER DEFAULT 0,
    downgrades_30d INTEGER DEFAULT 0,

    fetched_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (ticker, fiscal_year, fiscal_quarter)
);

CREATE INDEX idx_eps_estimates_ticker ON eps_estimates(ticker);
```

#### IC Score Service Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Create `EarningsRevisionsCalculator` class | IC-2 | 8 | Data pipeline |
| Implement magnitude score (estimate change %) | IC-2 | 4 | Above |
| Implement breadth score (upgrades vs downgrades) | IC-2 | 4 | Above |
| Implement recency score (acceleration detection) | IC-2 | 4 | Above |
| Integrate into main IC Score pipeline | IC-2 | 4 | Above |
| Unit and integration tests | IC-2 | 8 | All above |

```python
# ic-score-service/factors/earnings_revisions.py

class EarningsRevisionsCalculator:
    """Calculate Earnings Revisions factor score."""

    WEIGHT = 0.08  # 8% of total score

    async def calculate(self, ticker: str) -> Optional[FactorResult]:
        data = await self.fetch_eps_estimates(ticker)

        if not data or not data.get('consensus_eps'):
            return None

        # Magnitude: How much have estimates changed?
        magnitude_score = self._calculate_magnitude(data)

        # Breadth: What % of analysts are raising estimates?
        breadth_score = self._calculate_breadth(data)

        # Recency: Are recent revisions more positive?
        recency_score = self._calculate_recency(data)

        overall = (
            magnitude_score * 0.50 +
            breadth_score * 0.30 +
            recency_score * 0.20
        )

        return FactorResult(
            name='earnings_revisions',
            score=overall,
            weight=self.WEIGHT,
            metrics={
                'magnitude_score': magnitude_score,
                'breadth_score': breadth_score,
                'recency_score': recency_score,
                'consensus_eps': data['consensus_eps'],
                'revision_pct_90d': data.get('revision_pct_90d'),
                'num_analysts': data.get('num_analysts'),
            }
        )

    def _calculate_magnitude(self, data: dict) -> float:
        """Score based on % change in consensus EPS."""
        current = data.get('consensus_eps')
        prior = data.get('estimate_90d_ago')

        if not current or not prior or prior == 0:
            return 50  # Neutral

        change_pct = (current - prior) / abs(prior)
        # Scale: -15% = 0, 0% = 50, +15% = 100
        score = 50 + (change_pct / 0.15) * 50
        return max(0, min(100, score))
```

**Sprint 4 Definition of Done**:
- [ ] EPS estimates ingested for all S&P 500 stocks
- [ ] Earnings Revisions factor calculating correctly
- [ ] Factor included in overall IC Score
- [ ] Historical revision data available (30/60/90 day)

---

### 3.2 Sprint 5 (Week 5): Historical Valuation Factor

#### Database Changes

```sql
-- Migration: 003_add_valuation_history.sql

-- Store monthly valuation snapshots for historical comparison
CREATE TABLE valuation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    snapshot_date DATE NOT NULL,

    pe_ratio NUMERIC(10,2),
    ps_ratio NUMERIC(10,2),
    pb_ratio NUMERIC(10,2),
    ev_ebitda NUMERIC(10,2),

    stock_price NUMERIC(10,2),

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (ticker, snapshot_date)
);

CREATE INDEX idx_valuation_history_ticker_date
ON valuation_history(ticker, snapshot_date DESC);

-- Populate from existing valuation_ratios (backfill)
INSERT INTO valuation_history (ticker, snapshot_date, pe_ratio, ps_ratio, pb_ratio, stock_price)
SELECT
    ticker,
    DATE_TRUNC('month', calculation_date)::DATE,
    AVG(ttm_pe_ratio),
    AVG(ttm_ps_ratio),
    AVG(ttm_pb_ratio),
    AVG(stock_price)
FROM valuation_ratios
WHERE calculation_date >= NOW() - INTERVAL '5 years'
GROUP BY ticker, DATE_TRUNC('month', calculation_date)::DATE;
```

#### IC Score Service Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Create `HistoricalValuationCalculator` class | IC-1 | 8 | Migration |
| Implement P/E historical percentile calculation | IC-1 | 4 | Above |
| Implement P/S historical percentile (for growth) | IC-1 | 4 | Above |
| Add 5-year range data to response | IC-1 | 4 | Above |
| Create monthly valuation snapshot job | IC-1 | 4 | Above |
| Integration tests | IC-1 | 6 | All above |

```python
# ic-score-service/factors/historical_valuation.py

class HistoricalValuationCalculator:
    """Calculate Historical Valuation factor score."""

    WEIGHT = 0.08  # 8% of total score
    HISTORY_YEARS = 5

    async def calculate(self, ticker: str) -> Optional[FactorResult]:
        current_pe = await self.get_current_pe(ticker)
        pe_history = await self.get_pe_history(ticker, years=self.HISTORY_YEARS)

        if not pe_history or current_pe is None:
            return None

        # Calculate where current P/E sits in 5-year history
        pe_percentile = self._percentile_in_history(current_pe, pe_history)

        # Lower percentile = cheaper = higher score
        pe_score = 100 - pe_percentile

        # For growth companies, also check P/S
        net_margin = await self.get_net_margin(ticker)

        if net_margin and net_margin < 5:
            current_ps = await self.get_current_ps(ticker)
            ps_history = await self.get_ps_history(ticker, years=self.HISTORY_YEARS)

            if ps_history and current_ps:
                ps_percentile = self._percentile_in_history(current_ps, ps_history)
                ps_score = 100 - ps_percentile

                # Weight P/S more heavily for growth companies
                overall = pe_score * 0.3 + ps_score * 0.7
            else:
                overall = pe_score
        else:
            overall = pe_score

        return FactorResult(
            name='historical_value',
            score=overall,
            weight=self.WEIGHT,
            metrics={
                'current_pe': current_pe,
                'pe_5y_low': min(pe_history),
                'pe_5y_high': max(pe_history),
                'pe_5y_median': np.median(pe_history),
                'pe_percentile': pe_percentile,
            }
        )
```

**Sprint 5 Definition of Done**:
- [ ] 5-year valuation history available for all stocks
- [ ] Historical percentile calculating correctly
- [ ] P/E range data included in API response
- [ ] Monthly snapshot job scheduled

---

### 3.3 Sprint 6 (Week 6): Smart Money Consolidation & Dividend Quality

#### IC Score Service Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Consolidate analyst + insider + institutional → Smart Money | IC-2 | 8 | None |
| Implement Dividend Quality factor (optional) | IC-2 | 8 | None |
| Add user preference handling for income mode | IC-2 | 4 | Above |
| Update weight distribution for new factor structure | IC-2 | 4 | All above |
| Integration tests | IC-2 | 8 | All above |

```python
# ic-score-service/factors/dividend_quality.py

class DividendQualityCalculator:
    """Calculate optional Dividend Quality factor for income investors."""

    WEIGHT = 0.05  # +5% when enabled

    async def calculate(self, ticker: str) -> Optional[FactorResult]:
        dividend_yield = await self.get_dividend_yield(ticker)

        # Only calculate for dividend-paying stocks
        if not dividend_yield or dividend_yield < 0.5:
            return None

        sector = await self.get_sector(ticker)

        # Yield score (sector-relative)
        yield_score = await self.sector_percentile(
            sector, 'dividend_yield', dividend_yield
        )

        # Payout ratio (optimal range 30-60%)
        payout = await self.get_payout_ratio(ticker)
        payout_score = self._score_payout_ratio(payout)

        # Dividend growth (5-year CAGR)
        div_growth = await self.get_dividend_growth_5y(ticker)
        growth_score = max(0, min(100, 50 + (div_growth or 0) * 5))

        # Dividend streak
        streak = await self.get_dividend_streak(ticker)
        streak_score = self._score_streak(streak)

        overall = (
            yield_score * 0.25 +
            payout_score * 0.25 +
            growth_score * 0.25 +
            streak_score * 0.25
        )

        return FactorResult(
            name='dividend_quality',
            score=overall,
            weight=self.WEIGHT,
            optional=True,
            metrics={
                'dividend_yield': dividend_yield,
                'payout_ratio': payout,
                'dividend_growth_5y': div_growth,
                'consecutive_years': streak,
            }
        )

    def _score_streak(self, streak: int) -> float:
        if streak >= 25:
            return 100  # Dividend Aristocrat
        elif streak >= 10:
            return 80
        elif streak >= 5:
            return 60
        else:
            return streak * 10
```

**Sprint 6 Definition of Done**:
- [ ] Smart Money factor combines 3 previous factors
- [ ] Dividend Quality available as optional factor
- [ ] Income mode toggle working in API
- [ ] All factors properly weighted

---

## 4. Phase 3: Enhanced Features (Weeks 7-9)

### 4.1 Sprint 7 (Week 7): Score Stability

#### IC Score Service Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Create `ScoreStabilizer` class | IC-1 | 6 | None |
| Implement exponential smoothing | IC-1 | 4 | Above |
| Create event detection for reset triggers | IC-1 | 8 | Above |
| Add previous_score tracking to pipeline | IC-1 | 4 | Above |
| Create score change threshold logic | IC-1 | 4 | Above |
| Integration tests | IC-1 | 6 | All above |

#### Database Changes

```sql
-- Migration: 004_add_score_events.sql

-- Track events that trigger score recalculation
CREATE TABLE ic_score_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    event_type VARCHAR(50) NOT NULL,  -- 'earnings_release', 'analyst_upgrade', etc.
    event_date DATE NOT NULL,
    description TEXT,
    impact_direction VARCHAR(10),  -- 'positive', 'negative', 'neutral'
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_score_events_ticker_date
ON ic_score_events(ticker, event_date DESC);

-- Add smoothed score column to ic_scores
ALTER TABLE ic_scores ADD COLUMN raw_score NUMERIC(5,2);
ALTER TABLE ic_scores ADD COLUMN smoothing_applied BOOLEAN DEFAULT FALSE;
```

```python
# ic-score-service/stability/score_stabilizer.py

class ScoreStabilizer:
    """Apply score smoothing to prevent daily whipsaw."""

    ALPHA = 0.7  # 70% new, 30% previous
    MIN_CHANGE_THRESHOLD = 0.5

    RESET_EVENTS = {
        'earnings_release',
        'analyst_rating_change',
        'insider_trade_large',
        'dividend_announcement',
        'acquisition_news',
        'guidance_update',
    }

    async def stabilize(
        self,
        ticker: str,
        new_score: float,
        previous_score: Optional[float],
        events: List[str]
    ) -> Tuple[float, bool]:
        """
        Apply smoothing unless a significant event occurred.

        Returns: (stabilized_score, smoothing_applied)
        """
        # No previous score - use new score directly
        if previous_score is None:
            return new_score, False

        # Check for reset events
        if any(event in self.RESET_EVENTS for event in events):
            return new_score, False

        # Apply exponential smoothing
        smoothed = self.ALPHA * new_score + (1 - self.ALPHA) * previous_score

        # Only update if change exceeds threshold
        if abs(smoothed - previous_score) < self.MIN_CHANGE_THRESHOLD:
            return previous_score, True

        return round(smoothed, 1), True

    async def detect_events(self, ticker: str, since_date: date) -> List[str]:
        """Detect significant events since last calculation."""
        events = []

        # Check earnings calendar
        if await self.has_recent_earnings(ticker, since_date):
            events.append('earnings_release')

        # Check analyst ratings
        if await self.has_rating_change(ticker, since_date):
            events.append('analyst_rating_change')

        # Check insider trades (>$100k)
        if await self.has_large_insider_trade(ticker, since_date):
            events.append('insider_trade_large')

        return events
```

**Sprint 7 Definition of Done**:
- [ ] Score smoothing working correctly
- [ ] Event detection identifying reset triggers
- [ ] API shows both raw_score and smoothed score
- [ ] Score changes <0.5 points not displayed

---

### 4.2 Sprint 8 (Week 8): Peer Comparison & Catalysts

#### Database Changes

```sql
-- Migration: 005_add_stock_peers.sql

CREATE TABLE stock_peers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    peer_ticker VARCHAR(10) NOT NULL,
    similarity_score NUMERIC(5,4),  -- 0-1 similarity
    calculated_at DATE NOT NULL DEFAULT CURRENT_DATE,

    UNIQUE (ticker, peer_ticker, calculated_at)
);

CREATE INDEX idx_stock_peers_ticker ON stock_peers(ticker, calculated_at DESC);

-- Catalyst events table (also used for sprint 7)
CREATE TABLE catalyst_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    title VARCHAR(200) NOT NULL,
    event_date DATE,
    icon VARCHAR(10),
    impact VARCHAR(20),  -- 'bullish', 'bearish', 'neutral'
    days_until INTEGER,  -- Calculated field
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_catalysts_ticker_date ON catalyst_events(ticker, event_date);
```

#### Backend Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Create peer comparison endpoint | BE-1 | 8 | Migration |
| Implement peer selection algorithm | BE-1 | 8 | Above |
| Create catalyst detection service | BE-2 | 8 | Migration |
| Add catalysts to IC Score response | BE-2 | 4 | Above |
| Add peer_comparison to IC Score response | BE-1 | 4 | Above |

```go
// backend/services/peer_service.go

type PeerService struct {
    db *gorm.DB
}

func (s *PeerService) GetPeers(ticker string, limit int) ([]PeerComparison, error) {
    stock, err := s.getStock(ticker)
    if err != nil {
        return nil, err
    }

    // Get candidates from same sector
    var candidates []Stock
    s.db.Where("sector = ? AND symbol != ?", stock.Sector, ticker).
        Where("market_cap BETWEEN ? AND ?",
            stock.MarketCap * 0.25,
            stock.MarketCap * 4).
        Find(&candidates)

    // Score by similarity
    scored := make([]ScoredPeer, 0)
    for _, c := range candidates {
        similarity := s.calculateSimilarity(stock, c)
        scored = append(scored, ScoredPeer{
            Ticker: c.Symbol,
            Similarity: similarity,
        })
    }

    // Sort and limit
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].Similarity > scored[j].Similarity
    })

    if len(scored) > limit {
        scored = scored[:limit]
    }

    // Get IC Scores for peers
    result := make([]PeerComparison, len(scored))
    for i, peer := range scored {
        peerScore, _ := s.getICScore(peer.Ticker)
        result[i] = PeerComparison{
            Ticker: peer.Ticker,
            Score:  peerScore.OverallScore,
            Delta:  peerScore.OverallScore - stock.ICScore,
        }
    }

    return result, nil
}
```

```go
// backend/services/catalyst_service.go

type CatalystService struct {
    db *gorm.DB
}

var CatalystTypes = []CatalystDetector{
    &EarningsDetector{},
    &InsiderTradeDetector{},
    &AnalystRatingDetector{},
    &TechnicalBreakoutDetector{},
    &DividendDateDetector{},
    &FiftyTwoWeekDetector{},
}

func (s *CatalystService) GetCatalysts(ticker string) ([]Catalyst, error) {
    catalysts := make([]Catalyst, 0)

    for _, detector := range CatalystTypes {
        if catalyst, found := detector.Detect(s.db, ticker); found {
            catalysts = append(catalysts, catalyst)
        }
    }

    // Sort by date (upcoming first)
    sort.Slice(catalysts, func(i, j int) bool {
        return catalysts[i].DaysUntil < catalysts[j].DaysUntil
    })

    return catalysts, nil
}
```

**Sprint 8 Definition of Done**:
- [ ] Peer comparison returning top 5 similar stocks
- [ ] Catalyst detection for 6 event types
- [ ] Both features included in IC Score API response
- [ ] Peers cached and refreshed daily

---

### 4.3 Sprint 9 (Week 9): Score Change Explanations & Granular Confidence

#### Backend Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Create score change tracking table | BE-1 | 4 | None |
| Implement score diff calculation | BE-1 | 8 | Above |
| Generate human-readable explanations | BE-1 | 8 | Above |
| Build granular confidence response | BE-2 | 8 | None |
| Add data freshness tracking | BE-2 | 6 | Above |

```sql
-- Migration: 006_add_score_changes.sql

CREATE TABLE ic_score_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    calculated_at DATE NOT NULL,

    previous_score NUMERIC(5,2),
    current_score NUMERIC(5,2),
    delta NUMERIC(5,2),

    -- Factor-level changes
    factor_changes JSONB,  -- [{"factor": "momentum", "delta": -15, "reason": "..."}]

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_score_changes_ticker ON ic_score_changes(ticker, calculated_at DESC);
```

```go
// backend/models/score_change.go

type ScoreChangeReason struct {
    Factor        string  `json:"factor"`
    PreviousScore float64 `json:"previous_score"`
    CurrentScore  float64 `json:"current_score"`
    Delta         float64 `json:"delta"`
    Weight        float64 `json:"weight"`
    Contribution  float64 `json:"contribution"`
    Explanation   string  `json:"explanation"`
}

type GranularConfidence struct {
    Level      string                     `json:"level"`
    Percentage float64                    `json:"percentage"`
    Factors    map[string]FactorDataStatus `json:"factors"`
}

type FactorDataStatus struct {
    Available     bool    `json:"available"`
    Freshness     string  `json:"freshness,omitempty"`
    FreshnessDays int     `json:"freshness_days,omitempty"`
    Count         int     `json:"count,omitempty"`
    Warning       string  `json:"warning,omitempty"`
    Reason        string  `json:"reason,omitempty"`
}
```

```go
// backend/services/score_explainer.go

func (s *ScoreExplainerService) ExplainChange(
    ticker string,
    current *ICScore,
    previous *ICScore,
) []ScoreChangeReason {
    if previous == nil {
        return nil
    }

    reasons := make([]ScoreChangeReason, 0)

    factorPairs := []struct {
        name    string
        current *float64
        previous *float64
        weight  float64
    }{
        {"growth", current.GrowthScore, previous.GrowthScore, 0.12},
        {"profitability", current.ProfitabilityScore, previous.ProfitabilityScore, 0.12},
        {"value", current.ValueScore, previous.ValueScore, 0.12},
        {"momentum", current.MomentumScore, previous.MomentumScore, 0.10},
        // ... other factors
    }

    for _, pair := range factorPairs {
        if pair.current == nil || pair.previous == nil {
            continue
        }

        delta := *pair.current - *pair.previous

        // Only report significant changes
        if math.Abs(delta) >= 3 {
            explanation := s.generateExplanation(ticker, pair.name, delta)
            reasons = append(reasons, ScoreChangeReason{
                Factor:        pair.name,
                PreviousScore: *pair.previous,
                CurrentScore:  *pair.current,
                Delta:         delta,
                Weight:        pair.weight,
                Contribution:  delta * pair.weight,
                Explanation:   explanation,
            })
        }
    }

    // Sort by absolute contribution
    sort.Slice(reasons, func(i, j int) bool {
        return math.Abs(reasons[i].Contribution) > math.Abs(reasons[j].Contribution)
    })

    return reasons
}
```

**Sprint 9 Definition of Done**:
- [ ] Score changes tracked with factor-level breakdown
- [ ] Human-readable explanations generated
- [ ] Granular confidence showing data availability
- [ ] API returns score_changes and confidence objects

---

## 5. Phase 4: UI Enhancement (Weeks 10-12)

### 5.1 Sprint 10 (Week 10): Core UI Components

#### Frontend Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Redesign ICScoreCard with 3 categories | FE-1 | 16 | API ready |
| Create factor expansion component | FE-1 | 12 | Above |
| Build ICScoreGauge with new color scheme | FE-2 | 8 | None |
| Create category score badges | FE-2 | 8 | None |

```typescript
// components/ic-score/ICScoreCardV2.tsx

interface ICScoreCardV2Props {
    ticker: string;
    score: ICScoreData;
    showPeers?: boolean;
    showCatalysts?: boolean;
}

export function ICScoreCardV2({ ticker, score, showPeers, showCatalysts }: ICScoreCardV2Props) {
    const [expandedFactors, setExpandedFactors] = useState<string[]>([]);

    return (
        <Card className="ic-score-card">
            {/* Header with gauge and rating */}
            <div className="flex items-center justify-between p-6">
                <ICScoreGauge
                    score={score.overall_score}
                    rating={score.rating}
                    change={score.score_change_7d}
                />
                <div className="text-right">
                    <LifecycleBadge stage={score.lifecycle_stage} />
                    <ConfidenceBadge level={score.confidence.level} />
                </div>
            </div>

            {/* Category breakdown */}
            <div className="grid grid-cols-3 gap-4 p-4 border-t">
                <CategoryScore
                    name="Quality"
                    score={score.categories.quality.score}
                    grade={score.categories.quality.grade}
                />
                <CategoryScore
                    name="Valuation"
                    score={score.categories.valuation.score}
                    grade={score.categories.valuation.grade}
                />
                <CategoryScore
                    name="Signals"
                    score={score.categories.signals.score}
                    grade={score.categories.signals.grade}
                />
            </div>

            {/* Factor details */}
            <FactorBreakdown
                factors={score.factors}
                expandedFactors={expandedFactors}
                onToggle={(factor) => toggleExpand(factor)}
            />

            {/* Peer comparison */}
            {showPeers && score.peer_comparison && (
                <PeerComparisonPanel peers={score.peer_comparison} />
            )}

            {/* Catalysts */}
            {showCatalysts && score.catalysts?.length > 0 && (
                <CatalystTimeline catalysts={score.catalysts} />
            )}

            {/* Score change explanation */}
            {score.score_changes?.length > 0 && (
                <ScoreChangeExplainer changes={score.score_changes} />
            )}
        </Card>
    );
}
```

---

### 5.2 Sprint 11 (Week 11): Advanced Components

#### Frontend Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Build PeerComparisonPanel | FE-1 | 12 | API ready |
| Build CatalystTimeline | FE-1 | 8 | API ready |
| Build ScoreChangeExplainer | FE-2 | 12 | API ready |
| Build GranularConfidenceDisplay | FE-2 | 8 | API ready |

```typescript
// components/ic-score/PeerComparisonPanel.tsx

interface PeerComparisonPanelProps {
    ticker: string;
    peers: PeerComparison[];
    currentScore: number;
}

export function PeerComparisonPanel({ ticker, peers, currentScore }: PeerComparisonPanelProps) {
    return (
        <div className="border-t p-4">
            <h3 className="font-medium mb-3">Peer Comparison</h3>
            <div className="space-y-2">
                {peers.map((peer) => (
                    <div key={peer.ticker} className="flex items-center justify-between">
                        <Link href={`/stock/${peer.ticker}`} className="font-mono">
                            {peer.ticker}
                        </Link>
                        <div className="flex items-center gap-2">
                            <span className="text-lg font-medium">{peer.score}</span>
                            <DeltaBadge delta={currentScore - peer.score} />
                        </div>
                    </div>
                ))}
            </div>
            <div className="mt-3 pt-3 border-t text-sm text-muted-foreground">
                Sector Rank: #{currentScore.sector_rank} of {currentScore.sector_total}
            </div>
        </div>
    );
}

// components/ic-score/CatalystTimeline.tsx

export function CatalystTimeline({ catalysts }: { catalysts: Catalyst[] }) {
    return (
        <div className="border-t p-4">
            <h3 className="font-medium mb-3">Upcoming Catalysts</h3>
            <div className="space-y-2">
                {catalysts.map((catalyst, i) => (
                    <div key={i} className="flex items-center gap-3">
                        <span className="text-xl">{catalyst.icon}</span>
                        <div className="flex-1">
                            <div className="font-medium">{catalyst.title}</div>
                            <div className="text-sm text-muted-foreground">
                                {catalyst.days_until > 0
                                    ? `In ${catalyst.days_until} days`
                                    : formatDate(catalyst.date)
                                }
                            </div>
                        </div>
                        {catalyst.impact && (
                            <ImpactBadge impact={catalyst.impact} />
                        )}
                    </div>
                ))}
            </div>
        </div>
    );
}
```

---

### 5.3 Sprint 12 (Week 12): Score History & Polish

#### Frontend Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Build ICScoreHistoryChart | FE-1 | 16 | API ready |
| Add event annotations to chart | FE-1 | 8 | Above |
| Build ICScoreExplainerModal | FE-2 | 8 | Content |
| Responsive design & polish | FE-2 | 8 | All above |
| Accessibility audit | FE-2 | 4 | All above |

```typescript
// components/ic-score/ICScoreHistoryChart.tsx

import { LineChart, Line, XAxis, YAxis, Tooltip, ReferenceLine } from 'recharts';

interface ICScoreHistoryChartProps {
    ticker: string;
    history: ICScoreHistoryPoint[];
    events?: ScoreEvent[];
}

export function ICScoreHistoryChart({ ticker, history, events }: ICScoreHistoryChartProps) {
    return (
        <div className="p-4">
            <h3 className="font-medium mb-4">IC Score History (90 Days)</h3>

            <LineChart width={600} height={300} data={history}>
                <XAxis dataKey="date" tickFormatter={formatDate} />
                <YAxis domain={[0, 100]} />

                {/* Rating threshold lines */}
                <ReferenceLine y={80} stroke="#10b981" strokeDasharray="3 3" />
                <ReferenceLine y={65} stroke="#84cc16" strokeDasharray="3 3" />
                <ReferenceLine y={50} stroke="#eab308" strokeDasharray="3 3" />
                <ReferenceLine y={35} stroke="#f97316" strokeDasharray="3 3" />

                {/* Score lines */}
                <Line
                    type="monotone"
                    dataKey="overall_score"
                    stroke="#2563eb"
                    strokeWidth={2}
                    dot={false}
                />
                <Line
                    type="monotone"
                    dataKey="quality_score"
                    stroke="#10b981"
                    strokeWidth={1}
                    strokeDasharray="5 5"
                    dot={false}
                />

                <Tooltip content={<CustomTooltip />} />
            </LineChart>

            {/* Key events */}
            {events && events.length > 0 && (
                <div className="mt-4 space-y-1">
                    <h4 className="text-sm font-medium">Key Events</h4>
                    {events.map((event, i) => (
                        <div key={i} className="flex items-center gap-2 text-sm">
                            <span>{event.icon}</span>
                            <span className="text-muted-foreground">{formatDate(event.date)}:</span>
                            <span>{event.title}</span>
                            <DeltaBadge delta={event.score_impact} />
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
```

**Phase 4 Definition of Done**:
- [ ] All UI components implemented and polished
- [ ] Mobile responsive
- [ ] Dark mode support
- [ ] Accessibility (WCAG 2.1 AA)
- [ ] Performance: <100ms render time

---

## 6. Phase 5: Validation & Launch (Weeks 13-16)

### 6.1 Sprint 13-14 (Weeks 13-14): Backtesting Infrastructure

#### Data Science Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Build backtesting framework | DS-1 | 24 | Historical data |
| Implement point-in-time score calculation | DS-1 | 16 | Above |
| Create decile portfolio builder | DS-1 | 8 | Above |
| Implement return calculation | DS-1 | 8 | Above |
| Run 5-year backtest | DS-1 | 8 | All above |
| Generate performance reports | DS-1 | 8 | Backtest results |

```python
# ic-score-service/backtesting/backtester.py

class ICScoreBacktester:
    """Backtest IC Score methodology against historical returns."""

    def __init__(self, start_date: date, end_date: date):
        self.start_date = start_date
        self.end_date = end_date
        self.results = []

    async def run_backtest(
        self,
        rebalance_frequency: str = "monthly"
    ) -> BacktestResults:
        """Run full historical backtest."""

        for period_start in self._generate_periods(rebalance_frequency):
            # Calculate scores using only data available at that time
            scores = await self._calculate_point_in_time_scores(period_start)

            # Create decile portfolios
            deciles = self._create_decile_portfolios(scores)

            # Calculate forward returns
            period_end = self._get_period_end(period_start, rebalance_frequency)

            for decile, tickers in deciles.items():
                returns = await self._calculate_portfolio_return(
                    tickers, period_start, period_end
                )

                self.results.append({
                    'period': period_start,
                    'decile': decile,
                    'return': returns,
                    'holdings': len(tickers),
                })

        return self._aggregate_results()

    async def _calculate_point_in_time_scores(
        self,
        as_of_date: date
    ) -> Dict[str, float]:
        """Calculate IC Scores using only data available at as_of_date."""
        scores = {}

        # Get all stocks that existed at that date
        stocks = await self._get_stocks_as_of(as_of_date)

        for stock in stocks:
            try:
                # Use historical data only
                score = await self.calculator.calculate_historical(
                    stock.ticker,
                    as_of_date
                )
                if score:
                    scores[stock.ticker] = score.overall_score
            except Exception as e:
                logger.warning(f"Could not calculate score for {stock.ticker}: {e}")

        return scores
```

---

### 6.2 Sprint 15-16 (Weeks 15-16): Launch Preparation

#### Backend Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Performance optimization | BE-1 | 16 | All features |
| Implement caching strategy | BE-1 | 12 | Above |
| Load testing | BE-1 | 8 | Above |
| Create feature flags for gradual rollout | BE-1 | 8 | None |

#### Frontend Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| Build backtest results dashboard | FE-1 | 16 | Backtest data |
| Create methodology explainer page | FE-1 | 8 | Content |
| Performance optimization | FE-2 | 8 | All components |

#### QA Tasks

| Task | Engineer | Hours | Dependencies |
|------|----------|-------|--------------|
| End-to-end test suite | QA-1 | 16 | All features |
| Regression testing | QA-1 | 8 | E2E suite |
| User acceptance testing | QA-1 | 8 | Regression |
| Documentation review | QA-1 | 4 | All docs |

---

## 7. Database Migrations Summary

| Migration | Phase | Description |
|-----------|-------|-------------|
| 001_add_sector_percentiles | 1 | Sector percentile statistics |
| 002_add_eps_estimates | 2 | EPS estimate data for revisions |
| 003_add_valuation_history | 2 | Historical valuation snapshots |
| 004_add_score_events | 3 | Event tracking for stability |
| 005_add_stock_peers | 3 | Peer comparison data |
| 006_add_score_changes | 3 | Score change tracking |
| 007_update_ic_scores | 3 | New columns for v2.1 |

### Migration 007: IC Scores Table Update

```sql
-- Migration: 007_update_ic_scores.sql

-- Add new factor columns
ALTER TABLE ic_scores
    ADD COLUMN earnings_revisions_score NUMERIC(5,2),
    ADD COLUMN historical_value_score NUMERIC(5,2),
    ADD COLUMN dividend_quality_score NUMERIC(5,2);

-- Add new metadata columns
ALTER TABLE ic_scores
    ADD COLUMN raw_score NUMERIC(5,2),
    ADD COLUMN smoothing_applied BOOLEAN DEFAULT FALSE,
    ADD COLUMN lifecycle_stage VARCHAR(20),
    ADD COLUMN weights_used JSONB;

-- Add peer and catalyst data (denormalized for performance)
ALTER TABLE ic_scores
    ADD COLUMN peer_scores JSONB,  -- [{"ticker": "MSFT", "score": 72}]
    ADD COLUMN catalysts JSONB;     -- [{"type": "earnings", "date": "..."}]

-- Add score change tracking
ALTER TABLE ic_scores
    ADD COLUMN previous_score NUMERIC(5,2),
    ADD COLUMN score_changes JSONB;  -- Factor-level changes

-- Add granular confidence
ALTER TABLE ic_scores
    ADD COLUMN confidence_details JSONB;
```

---

## 8. API Changes Summary

### Updated Endpoints

| Endpoint | Change | Phase |
|----------|--------|-------|
| `GET /api/v1/stocks/{ticker}/ic-score` | Extended response | 1-3 |
| `GET /api/v1/stocks/{ticker}/ic-score/history` | Add events | 4 |
| `GET /api/v1/stocks/{ticker}/ic-score/peers` | New endpoint | 3 |
| `GET /api/v1/ic-scores/backtest` | New endpoint | 5 |

### Response Schema Changes

```typescript
// Updated IC Score Response (v2.1)
interface ICScoreResponseV2 {
    ticker: string;
    calculated_at: string;

    // Scores
    overall_score: number;
    raw_score?: number;  // Before smoothing
    rating: string;
    rating_change?: string;

    // Context (NEW)
    lifecycle_stage: string;
    sector: string;
    sector_rank: number;
    sector_percentile: number;

    // Categories
    categories: {
        quality: CategoryScore;
        valuation: CategoryScore;
        signals: CategoryScore;
    };

    // Factors (updated weights)
    factors: FactorScore[];

    // NEW: Peer comparison
    peer_comparison?: {
        peers: PeerScore[];
        sector_rank: number;
        sector_total: number;
    };

    // NEW: Catalysts
    catalysts?: Catalyst[];

    // NEW: Score changes
    score_changes?: ScoreChangeReason[];

    // NEW: Granular confidence
    confidence: {
        level: string;
        percentage: number;
        factors: Record<string, FactorDataStatus>;
    };

    // Metadata
    smoothing_applied: boolean;
    weights_used: Record<string, number>;
    next_update: string;
}
```

---

## 9. Testing Strategy

### Unit Tests

| Component | Coverage Target | Owner |
|-----------|-----------------|-------|
| Sector Percentile Calculator | 90% | IC Score Service |
| Lifecycle Classifier | 95% | IC Score Service |
| Factor Calculators (each) | 90% | IC Score Service |
| Score Stabilizer | 95% | IC Score Service |
| Peer Selection | 85% | Backend |
| Catalyst Detection | 85% | Backend |

### Integration Tests

| Test Suite | Scope | Owner |
|------------|-------|-------|
| IC Score Pipeline | End-to-end calculation | IC Score Service |
| API Response Validation | Schema compliance | Backend |
| Data Pipeline | Ingestion to scoring | IC Score Service |

### E2E Tests

| Test Suite | Scope | Owner |
|------------|-------|-------|
| IC Score Card UI | Component rendering | Frontend |
| Score History Chart | Interactive features | Frontend |
| Peer Comparison | Click-through | Frontend |

### Performance Tests

| Test | Target | Owner |
|------|--------|-------|
| Single score calculation | <100ms | IC Score Service |
| API response time | <50ms (cached), <200ms (uncached) | Backend |
| UI render time | <100ms | Frontend |
| Backtest full run | <1 hour (5 years) | Data Science |

---

## 10. Rollout Strategy

### Feature Flags

```go
// Feature flags for gradual rollout
var ICScoreV2Features = map[string]bool{
    "sector_relative_scoring": false,
    "lifecycle_classification": false,
    "earnings_revisions_factor": false,
    "historical_valuation_factor": false,
    "dividend_quality_factor": false,
    "score_stability": false,
    "peer_comparison": false,
    "catalysts": false,
    "score_change_explanations": false,
    "granular_confidence": false,
}
```

### Rollout Phases

| Phase | % Users | Features Enabled | Duration |
|-------|---------|------------------|----------|
| Alpha | 1% (internal) | All | 1 week |
| Beta | 5% | All | 2 weeks |
| GA Phase 1 | 25% | All | 1 week |
| GA Phase 2 | 50% | All | 1 week |
| GA Phase 3 | 100% | All | - |

### Rollback Plan

1. **Automatic rollback triggers**:
   - API error rate >5%
   - P95 latency >500ms
   - Score calculation failures >10%

2. **Manual rollback**:
   - Disable feature flags
   - Revert to v2.0 scoring logic
   - Keep new data tables (no data loss)

---

## 11. Risk Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| EPS estimate data quality issues | Medium | High | Validate against multiple sources |
| Backtest shows poor performance | Medium | Critical | A/B test before full rollout |
| Score stability causes stale scores | Low | Medium | Tune smoothing parameters |
| Peer selection produces poor matches | Medium | Low | Add user feedback mechanism |
| Performance degradation | Low | High | Aggressive caching strategy |

---

## 12. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Backtest: Top decile vs bottom | >15% spread | Annual returns |
| API latency P95 | <200ms | DataDog |
| Score coverage | >95% S&P 500 | Daily monitoring |
| User engagement with IC Score | +20% vs v2.0 | Analytics |
| Support tickets on methodology | <5% of total | Zendesk |

---

## Appendix: Sprint Calendar

| Sprint | Dates | Focus |
|--------|-------|-------|
| Sprint 1 | Week 1 | Sector percentile infrastructure |
| Sprint 2 | Week 2 | Factor refactoring |
| Sprint 3 | Week 3 | Lifecycle classification |
| Sprint 4 | Week 4 | Earnings Revisions factor |
| Sprint 5 | Week 5 | Historical Valuation factor |
| Sprint 6 | Week 6 | Smart Money & Dividend Quality |
| Sprint 7 | Week 7 | Score Stability |
| Sprint 8 | Week 8 | Peer Comparison & Catalysts |
| Sprint 9 | Week 9 | Score Changes & Confidence |
| Sprint 10 | Week 10 | Core UI components |
| Sprint 11 | Week 11 | Advanced UI components |
| Sprint 12 | Week 12 | History chart & polish |
| Sprint 13-14 | Weeks 13-14 | Backtesting |
| Sprint 15-16 | Weeks 15-16 | Launch prep |
| Sprint 17-20 | Weeks 17-20 | Personalization |

---

*Document maintained by IC Score Team*
