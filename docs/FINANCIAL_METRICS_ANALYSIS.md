# Financial Metrics Analysis Report

**Date**: November 27, 2025
**Purpose**: Evaluate data availability for expanded Fundamentals metrics

---

## Executive Summary

This report analyzes your requested financial metrics against InvestorCenter.ai's current data infrastructure. The good news: **you already have ~70% of the required data** through your existing Polygon.io API subscription and SEC data pipelines. The remaining 30% requires either calculation logic (using existing data) or new data sources.

---

## Metrics Analysis by Category

### 1. Profitability Margins

| Metric | Status | Source | Notes |
|--------|--------|--------|-------|
| **Gross Margin** | ✅ AVAILABLE | Polygon API + SEC | Stored in `fundamentals.gross_margin` |
| **Operating Margin** | ✅ AVAILABLE | Polygon API + SEC | Stored in `fundamentals.operating_margin` |
| **Net Margin** | ✅ AVAILABLE | Polygon API + SEC | Stored in `fundamentals.net_margin` |
| **EBITDA Margin** | ⚠️ CALCULABLE | Polygon API | Have EBITDA and Revenue; need to add calculation: `EBITDA / Revenue * 100` |

**Implementation**:
- Gross/Operating/Net margins already returned by `GetFundamentals()` in `backend/services/polygon.go`
- EBITDA Margin: Add calculation in `convertPolygonFinancials()` function

---

### 2. Returns

| Metric | Status | Source | Notes |
|--------|--------|--------|-------|
| **ROA** | ✅ AVAILABLE | Polygon API + SEC | Stored in `fundamentals.roa` and `ic-score-service.financials.roa` |
| **ROE** | ✅ AVAILABLE | Polygon API + SEC | Stored in `fundamentals.roe` and `ic-score-service.financials.roe` |
| **ROIC** | ✅ AVAILABLE | Polygon API + SEC | Stored in `fundamentals.roic` and `ic-score-service.financials.roic` |

**Implementation**: All return metrics are already calculated and stored. Polygon.io's `/vX/reference/financials` endpoint provides these directly.

---

### 3. Growth Rates

| Metric | Status | Source | Notes |
|--------|--------|--------|-------|
| **Revenue Growth YoY** | ⚠️ CALCULABLE | Historical Financials | Have quarterly/annual revenue data; need YoY calc |
| **Revenue Growth 3Y** | ⚠️ CALCULABLE | Historical Financials | Need CAGR calculation over 3 years |
| **Revenue Growth 5Y** | ⚠️ CALCULABLE | Historical Financials | Need CAGR calculation over 5 years |
| **EPS Growth YoY** | ⚠️ CALCULABLE | Historical Financials | Have quarterly/annual EPS data |
| **EPS Growth 3Y** | ⚠️ CALCULABLE | Historical Financials | Need CAGR calculation |
| **EPS Growth 5Y** | ⚠️ CALCULABLE | Historical Financials | Need CAGR calculation |
| **FCF Growth** | ⚠️ CALCULABLE | Historical Financials | Have FCF data; need growth calc |

**Current Data**:
- SEC pipeline fetches **5 years of quarterly data** (`sec_financials_ingestion.py`)
- TTM financials calculated daily (`ttm_financials_calculator.py`)
- Revenue, EPS, and FCF stored for each period

**Implementation Required**:
```python
# Add to ic-score-service/pipelines/growth_rates_calculator.py
def calculate_cagr(start_value, end_value, years):
    if start_value <= 0 or end_value <= 0:
        return None
    return ((end_value / start_value) ** (1 / years) - 1) * 100

def calculate_yoy_growth(current, prior):
    if prior == 0 or prior is None:
        return None
    return ((current - prior) / abs(prior)) * 100
```

---

### 4. Valuation

| Metric | Status | Source | Notes |
|--------|--------|--------|-------|
| **Enterprise Value** | ✅ AVAILABLE | Polygon API | Returned in fundamentals: `fundamentals.enterprise_value` |
| **EV/Revenue** | ✅ AVAILABLE | Polygon API | Returned as `fundamentals.ev_to_revenue` |
| **EV/EBITDA** | ✅ AVAILABLE | Polygon API | Returned as `fundamentals.ev_to_ebitda` |
| **EV/FCF** | ⚠️ CALCULABLE | Existing Data | Have EV and FCF; need calculation: `EV / FCF` |

**Current Schema** (from `backend/migrations/001_create_stock_tables.sql`):
```sql
CREATE TABLE fundamentals (
    enterprise_value DECIMAL(20, 2),
    ev_to_revenue DECIMAL(12, 4),
    ev_to_ebitda DECIMAL(12, 4),
    free_cash_flow DECIMAL(20, 2),
    -- EV/FCF not stored but easily calculable
);
```

**Implementation**: Add `ev_to_fcf` calculation in `convertPolygonFinancials()`:
```go
if fundamentals.FreeCashFlow > 0 {
    fundamentals.EVToFCF = fundamentals.EnterpriseValue / fundamentals.FreeCashFlow
}
```

---

### 5. Liquidity

| Metric | Status | Source | Notes |
|--------|--------|--------|-------|
| **Current Ratio** | ✅ AVAILABLE | Polygon API + SEC | `fundamentals.current_ratio` |
| **Quick Ratio** | ✅ AVAILABLE | Polygon API + SEC | `fundamentals.quick_ratio` |

**Implementation**: Both already available from Polygon.io financials API.

---

### 6. Debt/Leverage

| Metric | Status | Source | Notes |
|--------|--------|--------|-------|
| **Debt-to-Equity** | ✅ AVAILABLE | Polygon API + SEC | `fundamentals.debt_to_equity` |
| **Interest Coverage** | ⚠️ CALCULABLE | SEC Filings | Need Operating Income / Interest Expense |
| **Net Debt/EBITDA** | ⚠️ CALCULABLE | Existing Data | Have Total Debt, Cash, EBITDA |

**Data Availability**:
- Total Debt: ✅ Available (`total_debt`)
- Cash: ✅ Available (`cash_and_equivalents`)
- EBITDA: ✅ Available
- Interest Expense: ⚠️ **Need to extract from SEC filings**

**Implementation**:
```python
# Net Debt / EBITDA
net_debt = total_debt - cash_and_equivalents
if ebitda > 0:
    net_debt_to_ebitda = net_debt / ebitda

# Interest Coverage (requires Interest Expense from SEC)
if interest_expense > 0:
    interest_coverage = operating_income / interest_expense
```

**Note**: Interest Expense extraction needs enhancement in `sec_financials_ingestion.py`. It's available in SEC filings but not currently parsed. Look for XBRL tags:
- `us-gaap:InterestExpense`
- `us-gaap:InterestAndDebtExpense`
- `us-gaap:InterestExpenseDebt`

---

### 7. Dividends

| Metric | Status | Source | Notes |
|--------|--------|--------|-------|
| **Dividend Yield** | ✅ AVAILABLE | `dividends` table | `yield_percent` column |
| **Payout Ratio** | ⚠️ CALCULABLE | Existing Data | Dividend / EPS |
| **Dividend Growth Rate** | ⚠️ CALCULABLE | Historical Dividends | Compare YoY dividend amounts |
| **Years of Consecutive Growth** | ⚠️ CALCULABLE | Historical Dividends | Count consecutive years of increases |

**Current Schema** (from `backend/migrations/001_create_stock_tables.sql`):
```sql
CREATE TABLE dividends (
    symbol VARCHAR(10),
    ex_date DATE,
    pay_date DATE,
    amount DECIMAL(12, 6),
    frequency VARCHAR(20),  -- Quarterly, Monthly, Annual
    type VARCHAR(50),       -- Regular, Special
    yield_percent DECIMAL(8, 4)
);
```

**Implementation Required**:
```python
# Payout Ratio
annual_dividend = sum(dividends for year)
if eps > 0:
    payout_ratio = (annual_dividend / eps) * 100

# Dividend Growth Rate (YoY)
current_annual = sum(current_year_dividends)
prior_annual = sum(prior_year_dividends)
if prior_annual > 0:
    dividend_growth = ((current_annual - prior_annual) / prior_annual) * 100

# Years of Consecutive Growth
def count_dividend_growth_years(dividend_history):
    years = 0
    for i in range(1, len(annual_dividends)):
        if annual_dividends[i] > annual_dividends[i-1]:
            years += 1
        else:
            break
    return years
```

---

### 8. Sector/Industry Comparisons

| Metric | Status | Source | Notes |
|--------|--------|--------|-------|
| **Sector Averages** | ⚠️ NEEDS BUILD | Aggregate Existing Data | Have sector classification |
| **Industry Averages** | ⚠️ NEEDS BUILD | Aggregate Existing Data | Have industry classification |
| **Percentile Rankings** | ⚠️ PARTIALLY BUILT | IC Score Service | IC Score includes sector percentile |

**Current Data**:
- `stocks.sector` and `stocks.industry` columns populated
- `companies.sector` and `companies.industry` in IC Score DB
- IC Score calculator already computes sector percentile

**Implementation Required**:
```sql
-- Create materialized view for sector averages (refresh daily)
CREATE MATERIALIZED VIEW sector_averages AS
SELECT
    sector,
    AVG(pe) as avg_pe,
    AVG(pb) as avg_pb,
    AVG(ps) as avg_ps,
    AVG(roe) as avg_roe,
    AVG(roa) as avg_roa,
    AVG(gross_margin) as avg_gross_margin,
    AVG(operating_margin) as avg_operating_margin,
    AVG(net_margin) as avg_net_margin,
    AVG(debt_to_equity) as avg_debt_to_equity,
    AVG(current_ratio) as avg_current_ratio,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY pe) as median_pe,
    COUNT(*) as company_count
FROM fundamentals f
JOIN stocks s ON f.symbol = s.symbol
WHERE f.period = 'FY' AND f.year = EXTRACT(YEAR FROM CURRENT_DATE) - 1
GROUP BY sector;
```

---

### 9. Fair Value (DCF)

| Metric | Status | Source | Notes |
|--------|--------|--------|-------|
| **DCF Fair Value** | ❌ NEEDS BUILD | Calculation Required | Requires assumptions + FCF projections |
| **Intrinsic Value** | ❌ NEEDS BUILD | Multiple Models | Graham Number, Earnings Power Value |

**Required Data (All Available)**:
- ✅ Free Cash Flow (historical 5 years)
- ✅ Revenue Growth (calculable)
- ✅ Shares Outstanding
- ⚠️ WACC (needs calculation from beta + risk-free rate)
- ✅ Treasury Rates (in `treasury_rates` table)
- ✅ Beta (in `risk_metrics` table)

**DCF Model Implementation**:
```python
def calculate_dcf_fair_value(
    fcf_ttm: float,
    growth_rate_5y: float,
    terminal_growth: float = 0.025,  # 2.5% perpetuity
    discount_rate: float = 0.10,      # 10% WACC default
    shares_outstanding: float,
    years: int = 10
) -> float:
    """
    Two-stage DCF model:
    - Stage 1: Project FCF at growth_rate for 5 years
    - Stage 2: Terminal value using Gordon Growth Model
    """
    projected_fcf = []
    current_fcf = fcf_ttm

    # Stage 1: High growth period (5 years)
    for year in range(1, 6):
        current_fcf *= (1 + growth_rate_5y)
        projected_fcf.append(current_fcf / (1 + discount_rate) ** year)

    # Stage 2: Mature growth period (years 6-10)
    mature_growth = (growth_rate_5y + terminal_growth) / 2  # Decay to terminal
    for year in range(6, years + 1):
        current_fcf *= (1 + mature_growth)
        projected_fcf.append(current_fcf / (1 + discount_rate) ** year)

    # Terminal Value (Gordon Growth Model)
    terminal_fcf = current_fcf * (1 + terminal_growth)
    terminal_value = terminal_fcf / (discount_rate - terminal_growth)
    present_terminal = terminal_value / (1 + discount_rate) ** years

    # Enterprise Value
    enterprise_value = sum(projected_fcf) + present_terminal

    # Equity Value (simplified - should subtract net debt)
    fair_value_per_share = enterprise_value / shares_outstanding

    return fair_value_per_share

def calculate_wacc(
    beta: float,
    risk_free_rate: float,      # 10Y Treasury
    market_risk_premium: float = 0.055,  # ~5.5% historical
    cost_of_debt: float = 0.05,  # Estimated
    debt_ratio: float = 0.3,     # D/(D+E)
    tax_rate: float = 0.21       # Corporate tax rate
) -> float:
    """
    WACC = E/(D+E) * Re + D/(D+E) * Rd * (1-T)
    Where Re = Rf + Beta * (Rm - Rf)
    """
    equity_ratio = 1 - debt_ratio
    cost_of_equity = risk_free_rate + beta * market_risk_premium
    wacc = (equity_ratio * cost_of_equity) + (debt_ratio * cost_of_debt * (1 - tax_rate))
    return wacc
```

---

## Data Source Summary

### Currently Paid APIs

| API | Cost | Data Provided | Coverage |
|-----|------|---------------|----------|
| **Polygon.io** | $199/mo (Stocks Starter) | Real-time prices, Fundamentals, News, Historical OHLCV | 4,600+ US stocks |
| **SEC EDGAR** | FREE | 10-K, 10-Q, Form 4, 13F filings | All US public companies |
| **CoinGecko** | FREE tier | Crypto prices, market data | 10,000+ cryptocurrencies |
| **ApeWisdom** | FREE | Reddit mentions, sentiment | WSB/investing subreddits |
| **FRED API** | FREE | Treasury rates, macro indicators | US economic data |

### Polygon.io Financials Endpoint

Your current Polygon subscription includes the Financials API (`/vX/reference/financials`), which provides:

**Income Statement**:
- Revenues, Cost of Revenue, Gross Profit
- Operating Expenses, Operating Income
- Interest Income/Expense, Other Income
- Income Before Tax, Tax Expense
- Net Income, EPS (Basic & Diluted)

**Balance Sheet**:
- Total Assets, Current Assets
- Cash & Equivalents, Inventory
- Total Liabilities, Current Liabilities
- Long-term Debt, Total Debt
- Shareholders' Equity, Retained Earnings

**Cash Flow**:
- Operating Cash Flow
- Capital Expenditure
- Free Cash Flow
- Investing Activities
- Financing Activities

**Pre-Calculated Ratios** (from Polygon):
- P/E, P/B, P/S
- ROE, ROA, ROIC
- Debt-to-Equity
- Current Ratio, Quick Ratio
- Gross/Operating/Net Margins

---

## Implementation Roadmap

---

### Phase 1: Quick Wins (1-2 days)

**Goal**: Add simple calculations using existing data

#### 1.1 Add EBITDA Margin

**File**: `ic-score-service/pipelines/fundamental_metrics_calculator.py` (new file)

```python
"""
Fundamental Metrics Calculator Pipeline
Calculates extended financial metrics from existing data.
"""
import logging
from datetime import date
from decimal import Decimal
from typing import Optional
from sqlalchemy import select, and_
from sqlalchemy.ext.asyncio import AsyncSession

from database.database import async_session_maker
from models import Financials, TTMFinancials, FundamentalMetricsExtended

logger = logging.getLogger(__name__)


def calculate_ebitda_margin(ebitda: Optional[Decimal], revenue: Optional[Decimal]) -> Optional[Decimal]:
    """EBITDA Margin = EBITDA / Revenue × 100"""
    if not ebitda or not revenue or revenue == 0:
        return None
    return (ebitda / revenue) * 100


def calculate_ev_to_fcf(enterprise_value: Optional[Decimal], fcf: Optional[Decimal]) -> Optional[Decimal]:
    """EV/FCF = Enterprise Value / Free Cash Flow"""
    if not enterprise_value or not fcf or fcf <= 0:
        return None
    return enterprise_value / fcf
```

#### 1.2 Add Growth Rate Calculations

**File**: `ic-score-service/pipelines/fundamental_metrics_calculator.py` (continued)

```python
def calculate_yoy_growth(
    current: Optional[Decimal],
    prior: Optional[Decimal],
    metric_type: str = "revenue"  # "revenue", "eps", "fcf"
) -> Optional[Decimal]:
    """
    Year-over-Year Growth = (Current - Prior) / Prior × 100

    Special handling for metrics that can be negative (EPS, FCF):
    - If signs change (loss to profit or vice versa), return None (not meaningful)
    - Revenue should always be positive, so no special handling needed

    Examples:
    - Revenue: $100M → $120M = 20% growth ✓
    - EPS: $2.00 → $2.50 = 25% growth ✓
    - EPS: -$1.00 → $1.00 = N/M (not meaningful, turnaround)
    - EPS: $1.00 → -$0.50 = N/M (not meaningful, turned to loss)
    """
    if current is None or prior is None or prior == 0:
        return None

    # For metrics that can be negative, check for sign changes
    if metric_type in ("eps", "fcf"):
        # Sign change = not meaningful for growth calculation
        if (current > 0 and prior < 0) or (current < 0 and prior > 0):
            return None
        # Both negative: use absolute values but preserve direction
        # e.g., -$2 → -$1 is actually improvement (50% improvement)
        if current < 0 and prior < 0:
            # Loss narrowed = positive "growth", loss widened = negative
            return ((abs(prior) - abs(current)) / abs(prior)) * 100

    return ((current - prior) / prior) * 100


def calculate_cagr(start_value: Optional[Decimal], end_value: Optional[Decimal], years: int) -> Optional[Decimal]:
    """
    Compound Annual Growth Rate = ((End/Start)^(1/years) - 1) × 100

    Example: Revenue grew from $100M to $150M over 3 years
    CAGR = ((150/100)^(1/3) - 1) × 100 = 14.47%
    """
    if not start_value or not end_value or start_value <= 0 or end_value <= 0 or years <= 0:
        return None
    return (((end_value / start_value) ** (Decimal(1) / years)) - 1) * 100


async def get_historical_annual_values(
    session: AsyncSession,
    ticker: str,
    field: str,
    years_back: int = 5
) -> dict[int, Decimal]:
    """
    Fetch annual values for a specific field going back N years.
    Returns dict mapping fiscal_year -> value
    """
    result = await session.execute(
        select(Financials.fiscal_year, getattr(Financials, field))
        .where(
            and_(
                Financials.ticker == ticker,
                Financials.fiscal_quarter.is_(None),  # Annual filings only
                Financials.fiscal_year >= date.today().year - years_back
            )
        )
        .order_by(Financials.fiscal_year.desc())
    )
    return {row[0]: row[1] for row in result.fetchall() if row[1] is not None}


async def calculate_growth_rates(
    session: AsyncSession,
    ticker: str
) -> dict:
    """Calculate all growth rate metrics for a ticker."""

    # Get historical revenue data
    revenue_history = await get_historical_annual_values(session, ticker, 'revenue', 5)
    eps_history = await get_historical_annual_values(session, ticker, 'diluted_eps', 5)
    fcf_history = await get_historical_annual_values(session, ticker, 'free_cash_flow', 5)

    current_year = date.today().year - 1  # Most recent complete fiscal year

    growth_rates = {}

    # Revenue Growth
    if current_year in revenue_history and (current_year - 1) in revenue_history:
        growth_rates['revenue_growth_yoy'] = calculate_yoy_growth(
            revenue_history[current_year],
            revenue_history[current_year - 1],
            metric_type="revenue"
        )

    if current_year in revenue_history and (current_year - 3) in revenue_history:
        growth_rates['revenue_growth_3y_cagr'] = calculate_cagr(
            revenue_history[current_year - 3],
            revenue_history[current_year],
            3
        )

    if current_year in revenue_history and (current_year - 5) in revenue_history:
        growth_rates['revenue_growth_5y_cagr'] = calculate_cagr(
            revenue_history[current_year - 5],
            revenue_history[current_year],
            5
        )

    # EPS Growth (handles negative EPS transitions)
    if current_year in eps_history and (current_year - 1) in eps_history:
        growth_rates['eps_growth_yoy'] = calculate_yoy_growth(
            eps_history[current_year],
            eps_history[current_year - 1],
            metric_type="eps"  # Returns None if sign changes (loss to profit)
        )

    if current_year in eps_history and (current_year - 3) in eps_history:
        growth_rates['eps_growth_3y_cagr'] = calculate_cagr(
            eps_history[current_year - 3],
            eps_history[current_year],
            3
        )

    if current_year in eps_history and (current_year - 5) in eps_history:
        growth_rates['eps_growth_5y_cagr'] = calculate_cagr(
            eps_history[current_year - 5],
            eps_history[current_year],
            5
        )

    # FCF Growth (handles negative FCF transitions)
    if current_year in fcf_history and (current_year - 1) in fcf_history:
        growth_rates['fcf_growth_yoy'] = calculate_yoy_growth(
            fcf_history[current_year],
            fcf_history[current_year - 1],
            metric_type="fcf"  # Returns None if sign changes
        )

    return growth_rates
```

#### 1.3 Database Migration

**File**: `ic-score-service/migrations/XXX_create_fundamental_metrics_extended.sql`

```sql
-- Migration: Create fundamental_metrics_extended table
-- Run after: All existing financial tables

CREATE TABLE IF NOT EXISTS fundamental_metrics_extended (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    calculation_date DATE NOT NULL,

    -- Profitability Margins (as percentages)
    gross_margin DECIMAL(10, 4),
    operating_margin DECIMAL(10, 4),
    net_margin DECIMAL(10, 4),
    ebitda_margin DECIMAL(10, 4),

    -- Returns (as percentages)
    roe DECIMAL(10, 4),
    roa DECIMAL(10, 4),
    roic DECIMAL(10, 4),

    -- Growth Rates (as percentages)
    revenue_growth_yoy DECIMAL(10, 4),
    revenue_growth_3y_cagr DECIMAL(10, 4),
    revenue_growth_5y_cagr DECIMAL(10, 4),
    eps_growth_yoy DECIMAL(10, 4),
    eps_growth_3y_cagr DECIMAL(10, 4),
    eps_growth_5y_cagr DECIMAL(10, 4),
    fcf_growth_yoy DECIMAL(10, 4),

    -- Valuation
    enterprise_value DECIMAL(20, 2),
    ev_to_revenue DECIMAL(12, 4),
    ev_to_ebitda DECIMAL(12, 4),
    ev_to_fcf DECIMAL(12, 4),

    -- Liquidity
    current_ratio DECIMAL(10, 4),
    quick_ratio DECIMAL(10, 4),

    -- Debt/Leverage
    debt_to_equity DECIMAL(10, 4),
    interest_coverage DECIMAL(10, 4),
    net_debt_to_ebitda DECIMAL(10, 4),

    -- Dividends
    dividend_yield DECIMAL(10, 4),
    payout_ratio DECIMAL(10, 4),
    dividend_growth_rate DECIMAL(10, 4),
    consecutive_dividend_years INTEGER,

    -- Fair Value
    dcf_fair_value DECIMAL(14, 2),
    dcf_upside_percent DECIMAL(10, 4),
    graham_number DECIMAL(14, 2),

    -- Sector Comparisons (percentile ranks 0-100)
    pe_sector_percentile INTEGER,
    pb_sector_percentile INTEGER,
    roe_sector_percentile INTEGER,
    margin_sector_percentile INTEGER,

    -- Metadata
    data_quality_score DECIMAL(5, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_ticker_date UNIQUE(ticker, calculation_date)
);

-- Indexes for common queries
CREATE INDEX idx_fund_metrics_ticker ON fundamental_metrics_extended(ticker);
CREATE INDEX idx_fund_metrics_date ON fundamental_metrics_extended(calculation_date DESC);
CREATE INDEX idx_fund_metrics_ticker_date ON fundamental_metrics_extended(ticker, calculation_date DESC);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_fundamental_metrics_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_fundamental_metrics_updated
    BEFORE UPDATE ON fundamental_metrics_extended
    FOR EACH ROW
    EXECUTE FUNCTION update_fundamental_metrics_timestamp();
```

#### 1.4 Add SQLAlchemy Model

**File**: `ic-score-service/models.py` (add to existing file)

```python
class FundamentalMetricsExtended(Base):
    __tablename__ = "fundamental_metrics_extended"

    id = Column(Integer, primary_key=True)
    ticker = Column(String(10), nullable=False)
    calculation_date = Column(Date, nullable=False)

    # Profitability Margins
    gross_margin = Column(Numeric(10, 4))
    operating_margin = Column(Numeric(10, 4))
    net_margin = Column(Numeric(10, 4))
    ebitda_margin = Column(Numeric(10, 4))

    # Returns
    roe = Column(Numeric(10, 4))
    roa = Column(Numeric(10, 4))
    roic = Column(Numeric(10, 4))

    # Growth Rates
    revenue_growth_yoy = Column(Numeric(10, 4))
    revenue_growth_3y_cagr = Column(Numeric(10, 4))
    revenue_growth_5y_cagr = Column(Numeric(10, 4))
    eps_growth_yoy = Column(Numeric(10, 4))
    eps_growth_3y_cagr = Column(Numeric(10, 4))
    eps_growth_5y_cagr = Column(Numeric(10, 4))
    fcf_growth_yoy = Column(Numeric(10, 4))

    # Valuation
    enterprise_value = Column(Numeric(20, 2))
    ev_to_revenue = Column(Numeric(12, 4))
    ev_to_ebitda = Column(Numeric(12, 4))
    ev_to_fcf = Column(Numeric(12, 4))

    # Liquidity
    current_ratio = Column(Numeric(10, 4))
    quick_ratio = Column(Numeric(10, 4))

    # Debt/Leverage
    debt_to_equity = Column(Numeric(10, 4))
    interest_coverage = Column(Numeric(10, 4))
    net_debt_to_ebitda = Column(Numeric(10, 4))

    # Dividends
    dividend_yield = Column(Numeric(10, 4))
    payout_ratio = Column(Numeric(10, 4))
    dividend_growth_rate = Column(Numeric(10, 4))
    consecutive_dividend_years = Column(Integer)

    # Fair Value
    dcf_fair_value = Column(Numeric(14, 2))
    dcf_upside_percent = Column(Numeric(10, 4))
    graham_number = Column(Numeric(14, 2))

    # Sector Comparisons
    pe_sector_percentile = Column(Integer)
    pb_sector_percentile = Column(Integer)
    roe_sector_percentile = Column(Integer)
    margin_sector_percentile = Column(Integer)

    # Metadata
    data_quality_score = Column(Numeric(5, 2))
    created_at = Column(DateTime, default=func.now())
    updated_at = Column(DateTime, default=func.now(), onupdate=func.now())

    __table_args__ = (
        UniqueConstraint('ticker', 'calculation_date', name='unique_ticker_date'),
    )
```

---

### Phase 2: Dividend Metrics (2-3 days)

**Goal**: Calculate dividend-related metrics from existing dividend history

#### 2.1 Dividend Calculations

**File**: `ic-score-service/pipelines/fundamental_metrics_calculator.py` (continued)

```python
from collections import defaultdict
from datetime import date, timedelta


async def get_dividend_history(
    session: AsyncSession,
    ticker: str,
    years_back: int = 10
) -> list[dict]:
    """
    Fetch dividend payment history for a ticker.
    Returns list of {year, total_amount} sorted by year desc.
    """
    # Query dividends table and aggregate by year
    result = await session.execute(
        text("""
            SELECT
                EXTRACT(YEAR FROM ex_date)::int as year,
                SUM(amount) as annual_dividend,
                COUNT(*) as payment_count
            FROM dividends
            WHERE symbol = :ticker
              AND ex_date >= :start_date
              AND type = 'Regular'  -- Exclude special dividends
            GROUP BY EXTRACT(YEAR FROM ex_date)
            ORDER BY year DESC
        """),
        {
            "ticker": ticker,
            "start_date": date.today() - timedelta(days=365 * years_back)
        }
    )
    return [
        {"year": row[0], "amount": row[1], "payments": row[2]}
        for row in result.fetchall()
    ]


def calculate_payout_ratio(
    annual_dividend: Optional[Decimal],
    eps: Optional[Decimal]
) -> Optional[Decimal]:
    """
    Payout Ratio = (Annual Dividend per Share / EPS) × 100

    Interpretation:
    - < 30%: Very conservative, room for dividend growth
    - 30-50%: Healthy, sustainable
    - 50-75%: Moderate risk
    - > 75%: High risk, may not be sustainable
    - > 100%: Paying more than earnings (using reserves/debt)
    """
    if annual_dividend is None or eps is None or eps <= 0:
        return None
    return (annual_dividend / eps) * 100


def calculate_dividend_growth_rate(dividend_history: list[dict]) -> Optional[Decimal]:
    """
    Calculate 5-year CAGR of dividend payments.

    Example: Dividend grew from $1.00 to $1.50 over 5 years
    Growth Rate = ((1.50/1.00)^(1/5) - 1) × 100 = 8.45%
    """
    if len(dividend_history) < 2:
        return None

    # Get most recent and 5-year-ago dividends
    current_year = dividend_history[0]["year"]
    current_amount = dividend_history[0]["amount"]

    # Find dividend from ~5 years ago
    target_year = current_year - 5
    prior_amount = None
    years_diff = 0

    for record in dividend_history:
        if record["year"] <= target_year:
            prior_amount = record["amount"]
            years_diff = current_year - record["year"]
            break

    if prior_amount is None or prior_amount <= 0 or years_diff == 0:
        return None

    return calculate_cagr(prior_amount, current_amount, years_diff)


def count_consecutive_dividend_growth_years(dividend_history: list[dict]) -> int:
    """
    Count how many consecutive years the dividend has increased.

    Dividend Aristocrats: 25+ years of consecutive increases
    Dividend Champions: 25+ years
    Dividend Contenders: 10-24 years
    Dividend Challengers: 5-9 years

    Returns 0 if no dividend history or dividend was cut/flat in most recent year.

    Note: If a company has raised dividends every year from 2019-2024 (5 raises),
    we report "5 years of consecutive growth" (the number of increases, not spans).
    """
    if len(dividend_history) < 2:
        return 0

    # Sort by year descending (most recent first)
    sorted_history = sorted(dividend_history, key=lambda x: x["year"], reverse=True)

    consecutive_years = 0
    for i in range(len(sorted_history) - 1):
        current = sorted_history[i]["amount"]
        prior = sorted_history[i + 1]["amount"]

        # Check if years are consecutive (no gaps)
        if sorted_history[i]["year"] - sorted_history[i + 1]["year"] != 1:
            break

        # Check if dividend increased (strictly greater, not equal)
        if current > prior:
            consecutive_years += 1
        else:
            # Dividend was cut or held flat - stop counting
            break

    return consecutive_years


async def calculate_dividend_metrics(
    session: AsyncSession,
    ticker: str,
    ttm_eps: Optional[Decimal],
    current_price: Optional[Decimal]
) -> dict:
    """Calculate all dividend-related metrics for a ticker."""

    dividend_history = await get_dividend_history(session, ticker, years_back=10)

    if not dividend_history:
        return {
            "dividend_yield": None,
            "payout_ratio": None,
            "dividend_growth_rate": None,
            "consecutive_dividend_years": 0
        }

    # Most recent annual dividend
    current_year_dividend = dividend_history[0]["amount"] if dividend_history else None

    # Dividend Yield = Annual Dividend / Current Price × 100
    dividend_yield = None
    if current_year_dividend and current_price and current_price > 0:
        dividend_yield = (current_year_dividend / current_price) * 100

    return {
        "dividend_yield": dividend_yield,
        "payout_ratio": calculate_payout_ratio(current_year_dividend, ttm_eps),
        "dividend_growth_rate": calculate_dividend_growth_rate(dividend_history),
        "consecutive_dividend_years": count_consecutive_dividend_growth_years(dividend_history)
    }
```

#### 2.2 Data Source: Polygon Dividends API

**Note**: Dividend data is already being fetched. Verify the pipeline populates the `dividends` table:

```python
# In existing pipeline or new dividend_ingestion.py
async def fetch_dividends_from_polygon(ticker: str, api_key: str) -> list[dict]:
    """
    Polygon.io Dividends endpoint: /v3/reference/dividends
    Returns historical dividend payments.
    """
    url = f"https://api.polygon.io/v3/reference/dividends"
    params = {
        "ticker": ticker,
        "limit": 100,  # Get all available history
        "apiKey": api_key
    }

    async with aiohttp.ClientSession() as session:
        async with session.get(url, params=params) as response:
            data = await response.json()

    return [
        {
            "symbol": ticker,
            "ex_date": d["ex_dividend_date"],
            "pay_date": d.get("pay_date"),
            "amount": d["cash_amount"],
            "frequency": d.get("frequency"),  # 1=Annual, 2=Semi, 4=Quarterly, 12=Monthly
            "type": d.get("dividend_type", "Regular"),
        }
        for d in data.get("results", [])
    ]
```

---

### Phase 3: Leverage Metrics (3-4 days)

**Goal**: Extract Interest Expense from SEC filings and calculate leverage ratios

#### 3.1 Update SEC Financials Extraction

**File**: `ic-score-service/pipelines/sec_financials_ingestion.py` (modify existing)

```python
# Add to XBRL_TAGS dictionary
INTEREST_EXPENSE_TAGS = [
    "us-gaap:InterestExpense",
    "us-gaap:InterestAndDebtExpense",
    "us-gaap:InterestExpenseDebt",
    "us-gaap:InterestExpenseBorrowings",
    "us-gaap:InterestExpenseRelatedParty",
    "us-gaap:InterestIncomeExpenseNet",  # May need sign flip
]

INCOME_STATEMENT_TAGS = {
    # ... existing tags ...

    # Add Interest Expense
    "interest_expense": INTEREST_EXPENSE_TAGS,

    # Add EBIT for additional calculations
    "ebit": [
        "us-gaap:OperatingIncomeLoss",
        "us-gaap:IncomeLossFromContinuingOperationsBeforeIncomeTaxes",
    ],
}


async def extract_interest_expense(xbrl_data: dict, filing_date: date) -> Optional[Decimal]:
    """
    Extract Interest Expense from SEC filing XBRL data.

    Note: Interest Expense is typically reported as a positive number
    representing the cost, not as a negative cash flow.
    """
    for tag in INTEREST_EXPENSE_TAGS:
        if tag in xbrl_data:
            facts = xbrl_data[tag]
            # Get the value closest to the filing date
            for fact in sorted(facts, key=lambda x: x.get("end", ""), reverse=True):
                if fact.get("form") in ["10-K", "10-Q"]:
                    value = fact.get("val")
                    if value is not None:
                        return Decimal(str(value))
    return None
```

#### 3.2 Database Migration for Interest Expense

**File**: `ic-score-service/migrations/XXX_add_interest_expense.sql`

```sql
-- Add interest_expense column to financials table
ALTER TABLE financials
ADD COLUMN IF NOT EXISTS interest_expense DECIMAL(20, 2);

-- Add interest_expense to ttm_financials table
ALTER TABLE ttm_financials
ADD COLUMN IF NOT EXISTS interest_expense DECIMAL(20, 2);

-- Comment for documentation
COMMENT ON COLUMN financials.interest_expense IS 'Interest expense from income statement, used for Interest Coverage ratio';
```

#### 3.3 Leverage Ratio Calculations

**File**: `ic-score-service/pipelines/fundamental_metrics_calculator.py` (continued)

```python
def calculate_interest_coverage(
    operating_income: Optional[Decimal],
    interest_expense: Optional[Decimal]
) -> Optional[Decimal]:
    """
    Interest Coverage Ratio = Operating Income (EBIT) / Interest Expense

    Also known as Times Interest Earned (TIE) ratio.

    Interpretation:
    - > 5: Excellent, very safe
    - 3-5: Good, healthy
    - 2-3: Adequate, some risk
    - 1-2: Poor, high risk
    - < 1: Cannot cover interest payments (distressed)

    Note: Some analysts use EBITDA instead of EBIT for a more conservative measure.
    """
    if operating_income is None or interest_expense is None:
        return None
    if interest_expense <= 0:
        return None  # No interest expense = N/A (not infinite)

    return operating_income / interest_expense


def calculate_net_debt_to_ebitda(
    total_debt: Optional[Decimal],
    cash: Optional[Decimal],
    ebitda: Optional[Decimal]
) -> Optional[Decimal]:
    """
    Net Debt / EBITDA = (Total Debt - Cash) / EBITDA

    Measures how many years of EBITDA needed to pay off debt.

    Interpretation:
    - < 1: Very low leverage, strong balance sheet
    - 1-2: Conservative leverage
    - 2-3: Moderate leverage
    - 3-4: Elevated leverage
    - > 4: High leverage, potential distress
    - Negative: More cash than debt (net cash position)

    Note: Industry context matters - utilities/REITs typically run higher leverage.
    """
    if total_debt is None or cash is None or ebitda is None:
        return None
    if ebitda <= 0:
        return None  # Negative EBITDA makes ratio meaningless

    net_debt = total_debt - cash
    return net_debt / ebitda


async def calculate_leverage_metrics(
    session: AsyncSession,
    ticker: str
) -> dict:
    """Calculate all leverage-related metrics for a ticker."""

    # Get most recent TTM financials
    ttm = await session.execute(
        select(TTMFinancials)
        .where(TTMFinancials.ticker == ticker)
        .order_by(TTMFinancials.calculation_date.desc())
        .limit(1)
    )
    ttm_data = ttm.scalar_one_or_none()

    if not ttm_data:
        return {
            "debt_to_equity": None,
            "interest_coverage": None,
            "net_debt_to_ebitda": None
        }

    # Calculate EBITDA if not stored directly
    # EBITDA = Operating Income + Depreciation & Amortization
    ebitda = ttm_data.ebitda if hasattr(ttm_data, 'ebitda') else None
    if ebitda is None and ttm_data.operating_income:
        # Approximate: Operating Income as proxy (conservative)
        ebitda = ttm_data.operating_income

    return {
        "debt_to_equity": (
            ttm_data.total_debt / ttm_data.shareholders_equity
            if ttm_data.total_debt and ttm_data.shareholders_equity and ttm_data.shareholders_equity > 0
            else None
        ),
        "interest_coverage": calculate_interest_coverage(
            ttm_data.operating_income,
            ttm_data.interest_expense
        ),
        "net_debt_to_ebitda": calculate_net_debt_to_ebitda(
            ttm_data.total_debt,
            ttm_data.cash,
            ebitda
        )
    }
```

---

### Phase 4: Sector Comparisons (1 week)

**Goal**: Build infrastructure for sector/industry benchmarking

#### 4.1 Create Materialized Views

**File**: `ic-score-service/migrations/XXX_create_sector_averages_views.sql`

```sql
-- Materialized view for sector-level metric averages
-- Refresh daily via CronJob after fundamental_metrics_extended is updated

CREATE MATERIALIZED VIEW IF NOT EXISTS sector_metric_averages AS
WITH latest_metrics AS (
    -- Get most recent metrics for each ticker
    SELECT DISTINCT ON (fme.ticker)
        fme.*,
        c.sector,
        c.industry
    FROM fundamental_metrics_extended fme
    JOIN companies c ON fme.ticker = c.ticker
    WHERE fme.calculation_date >= CURRENT_DATE - INTERVAL '7 days'
      AND c.sector IS NOT NULL
    ORDER BY fme.ticker, fme.calculation_date DESC
)
SELECT
    sector,

    -- Count
    COUNT(*) as company_count,

    -- Valuation Averages
    AVG(ev_to_revenue) FILTER (WHERE ev_to_revenue BETWEEN 0 AND 100) as avg_ev_revenue,
    AVG(ev_to_ebitda) FILTER (WHERE ev_to_ebitda BETWEEN 0 AND 100) as avg_ev_ebitda,
    AVG(ev_to_fcf) FILTER (WHERE ev_to_fcf BETWEEN 0 AND 100) as avg_ev_fcf,

    -- Profitability Averages
    AVG(gross_margin) FILTER (WHERE gross_margin BETWEEN -100 AND 100) as avg_gross_margin,
    AVG(operating_margin) FILTER (WHERE operating_margin BETWEEN -100 AND 100) as avg_operating_margin,
    AVG(net_margin) FILTER (WHERE net_margin BETWEEN -100 AND 100) as avg_net_margin,
    AVG(ebitda_margin) FILTER (WHERE ebitda_margin BETWEEN -100 AND 100) as avg_ebitda_margin,

    -- Return Averages
    AVG(roe) FILTER (WHERE roe BETWEEN -100 AND 200) as avg_roe,
    AVG(roa) FILTER (WHERE roa BETWEEN -100 AND 100) as avg_roa,
    AVG(roic) FILTER (WHERE roic BETWEEN -100 AND 100) as avg_roic,

    -- Growth Averages
    AVG(revenue_growth_yoy) FILTER (WHERE revenue_growth_yoy BETWEEN -100 AND 500) as avg_revenue_growth_yoy,
    AVG(eps_growth_yoy) FILTER (WHERE eps_growth_yoy BETWEEN -100 AND 500) as avg_eps_growth_yoy,

    -- Leverage Averages
    AVG(debt_to_equity) FILTER (WHERE debt_to_equity BETWEEN 0 AND 10) as avg_debt_to_equity,
    AVG(interest_coverage) FILTER (WHERE interest_coverage BETWEEN 0 AND 100) as avg_interest_coverage,
    AVG(net_debt_to_ebitda) FILTER (WHERE net_debt_to_ebitda BETWEEN -5 AND 20) as avg_net_debt_to_ebitda,

    -- Liquidity Averages
    AVG(current_ratio) FILTER (WHERE current_ratio BETWEEN 0 AND 10) as avg_current_ratio,
    AVG(quick_ratio) FILTER (WHERE quick_ratio BETWEEN 0 AND 10) as avg_quick_ratio,

    -- Dividend Averages
    AVG(dividend_yield) FILTER (WHERE dividend_yield BETWEEN 0 AND 20) as avg_dividend_yield,
    AVG(payout_ratio) FILTER (WHERE payout_ratio BETWEEN 0 AND 200) as avg_payout_ratio,

    -- Medians for key metrics (more robust than averages)
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY ev_to_revenue)
        FILTER (WHERE ev_to_revenue BETWEEN 0 AND 100) as median_ev_revenue,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY net_margin)
        FILTER (WHERE net_margin BETWEEN -100 AND 100) as median_net_margin,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY roe)
        FILTER (WHERE roe BETWEEN -100 AND 200) as median_roe,

    -- Timestamp
    CURRENT_TIMESTAMP as calculated_at

FROM latest_metrics
GROUP BY sector;

-- Index for fast lookups
CREATE UNIQUE INDEX IF NOT EXISTS idx_sector_averages_sector
ON sector_metric_averages(sector);

-- Similar view for industry-level (more granular)
CREATE MATERIALIZED VIEW IF NOT EXISTS industry_metric_averages AS
WITH latest_metrics AS (
    SELECT DISTINCT ON (fme.ticker)
        fme.*,
        c.sector,
        c.industry
    FROM fundamental_metrics_extended fme
    JOIN companies c ON fme.ticker = c.ticker
    WHERE fme.calculation_date >= CURRENT_DATE - INTERVAL '7 days'
      AND c.industry IS NOT NULL
    ORDER BY fme.ticker, fme.calculation_date DESC
)
SELECT
    sector,
    industry,
    COUNT(*) as company_count,

    -- Same metrics as sector view
    AVG(ev_to_revenue) FILTER (WHERE ev_to_revenue BETWEEN 0 AND 100) as avg_ev_revenue,
    AVG(gross_margin) FILTER (WHERE gross_margin BETWEEN -100 AND 100) as avg_gross_margin,
    AVG(operating_margin) FILTER (WHERE operating_margin BETWEEN -100 AND 100) as avg_operating_margin,
    AVG(net_margin) FILTER (WHERE net_margin BETWEEN -100 AND 100) as avg_net_margin,
    AVG(roe) FILTER (WHERE roe BETWEEN -100 AND 200) as avg_roe,
    AVG(roa) FILTER (WHERE roa BETWEEN -100 AND 100) as avg_roa,
    AVG(revenue_growth_yoy) FILTER (WHERE revenue_growth_yoy BETWEEN -100 AND 500) as avg_revenue_growth_yoy,
    AVG(debt_to_equity) FILTER (WHERE debt_to_equity BETWEEN 0 AND 10) as avg_debt_to_equity,
    AVG(dividend_yield) FILTER (WHERE dividend_yield BETWEEN 0 AND 20) as avg_dividend_yield,

    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY net_margin)
        FILTER (WHERE net_margin BETWEEN -100 AND 100) as median_net_margin,

    CURRENT_TIMESTAMP as calculated_at
FROM latest_metrics
WHERE industry IS NOT NULL
GROUP BY sector, industry
HAVING COUNT(*) >= 5;  -- Only industries with 5+ companies

CREATE UNIQUE INDEX IF NOT EXISTS idx_industry_averages_industry
ON industry_metric_averages(sector, industry);
```

#### 4.2 Percentile Ranking Functions

**File**: `ic-score-service/pipelines/fundamental_metrics_calculator.py` (continued)

```python
async def calculate_sector_percentiles(
    session: AsyncSession,
    ticker: str,
    metrics: dict
) -> dict:
    """
    Calculate percentile rank within sector for key metrics.

    Percentile 90 = better than 90% of sector peers
    Percentile 50 = median
    Percentile 10 = worse than 90% of sector peers

    Note: For metrics where LOWER is better (e.g., P/E, Debt/Equity),
    we invert the percentile so higher = better.
    """

    # Get company's sector
    company = await session.execute(
        select(Company.sector).where(Company.ticker == ticker)
    )
    sector = company.scalar_one_or_none()

    if not sector:
        return {}

    # Get all companies in same sector with recent metrics
    sector_metrics = await session.execute(
        text("""
            SELECT
                fme.ticker,
                fme.roe,
                fme.net_margin,
                fme.ev_to_revenue,
                fme.debt_to_equity
            FROM fundamental_metrics_extended fme
            JOIN companies c ON fme.ticker = c.ticker
            WHERE c.sector = :sector
              AND fme.calculation_date >= CURRENT_DATE - INTERVAL '7 days'
            ORDER BY fme.ticker, fme.calculation_date DESC
        """),
        {"sector": sector}
    )

    sector_data = sector_metrics.fetchall()

    if len(sector_data) < 5:
        return {}  # Not enough peers for meaningful comparison

    def percentile_rank(values: list, target: float, higher_is_better: bool = True) -> int:
        """
        Calculate percentile rank of target within values.

        Uses the standard percentile rank formula:
        PR = (B + 0.5E) / N × 100

        Where:
        - B = number of values below target
        - E = number of values equal to target
        - N = total number of values

        This gives a more accurate percentile that accounts for ties.
        """
        if target is None:
            return None
        valid_values = [v for v in values if v is not None]
        if not valid_values:
            return None

        count_below = sum(1 for v in valid_values if v < target)
        count_equal = sum(1 for v in valid_values if v == target)

        # Standard percentile rank formula with tie handling
        percentile = ((count_below + 0.5 * count_equal) / len(valid_values)) * 100

        if not higher_is_better:
            percentile = 100 - percentile  # Invert for "lower is better" metrics

        return int(round(percentile))

    # Extract values for each metric
    roe_values = [row[1] for row in sector_data]
    margin_values = [row[2] for row in sector_data]
    ev_rev_values = [row[3] for row in sector_data]
    de_values = [row[4] for row in sector_data]

    return {
        "roe_sector_percentile": percentile_rank(roe_values, metrics.get("roe"), higher_is_better=True),
        "margin_sector_percentile": percentile_rank(margin_values, metrics.get("net_margin"), higher_is_better=True),
        "pe_sector_percentile": percentile_rank(ev_rev_values, metrics.get("ev_to_revenue"), higher_is_better=False),
        "pb_sector_percentile": percentile_rank(de_values, metrics.get("debt_to_equity"), higher_is_better=False),
    }
```

#### 4.3 CronJob to Refresh Materialized Views

**File**: `ic-score-service/k8s/ic-score-refresh-sector-views-cronjob.yaml`

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: ic-score-refresh-sector-views
  namespace: investorcenter
spec:
  schedule: "30 6 * * *"  # 6:30 AM UTC daily (after metrics calculation)
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: refresh-views
            image: postgres:15
            command:
            - /bin/sh
            - -c
            - |
              PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "
                REFRESH MATERIALIZED VIEW CONCURRENTLY sector_metric_averages;
                REFRESH MATERIALIZED VIEW CONCURRENTLY industry_metric_averages;
              "
            envFrom:
            - secretRef:
                name: postgres-secret
          restartPolicy: OnFailure
```

#### 4.4 API Endpoint for Sector Comparison

**File**: `backend/handlers/sector_comparison_handlers.go` (new file)

```go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

type SectorComparison struct {
    Ticker              string   `json:"ticker"`
    Sector              string   `json:"sector"`
    Industry            string   `json:"industry"`

    // Company Metrics
    Metrics             map[string]float64 `json:"metrics"`

    // Sector Averages
    SectorAverages      map[string]float64 `json:"sector_averages"`

    // Percentile Rankings (0-100)
    Percentiles         map[string]int     `json:"percentiles"`

    // Peer Companies (top 5 similar)
    Peers               []PeerCompany      `json:"peers"`
}

type PeerCompany struct {
    Ticker      string  `json:"ticker"`
    Name        string  `json:"name"`
    MarketCap   float64 `json:"market_cap"`
    Similarity  float64 `json:"similarity_score"`
}

// GetSectorComparison returns sector benchmarking for a ticker
func GetSectorComparison(c *gin.Context) {
    symbol := c.Param("symbol")

    // Fetch from ic-score-service or database
    comparison, err := fetchSectorComparison(symbol)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, comparison)
}

// Add route in main.go:
// v1.GET("/tickers/:symbol/sector-comparison", handlers.GetSectorComparison)
```

---

### Phase 5: DCF Fair Value (1-2 weeks)

**Goal**: Build discounted cash flow valuation model

#### 5.1 WACC Calculator

**File**: `ic-score-service/pipelines/fair_value_calculator.py` (new file)

```python
"""
Fair Value Calculator Pipeline
Calculates DCF-based intrinsic value estimates.
"""
import logging
from datetime import date
from decimal import Decimal
from typing import Optional, Tuple
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from database.database import async_session_maker
from models import (
    TTMFinancials, RiskMetrics, TreasuryRates,
    FundamentalMetricsExtended, Company
)

logger = logging.getLogger(__name__)


async def get_risk_free_rate(session: AsyncSession) -> Decimal:
    """
    Get current 10-Year Treasury rate as risk-free rate.
    Falls back to 4.5% if not available.
    """
    result = await session.execute(
        select(TreasuryRates.rate_10y)
        .order_by(TreasuryRates.date.desc())
        .limit(1)
    )
    rate = result.scalar_one_or_none()
    return Decimal(str(rate / 100)) if rate else Decimal("0.045")


async def get_beta(session: AsyncSession, ticker: str) -> Optional[Decimal]:
    """Get stock's beta from risk_metrics table."""
    result = await session.execute(
        select(RiskMetrics.beta)
        .where(RiskMetrics.ticker == ticker)
        .order_by(RiskMetrics.calculation_date.desc())
        .limit(1)
    )
    return result.scalar_one_or_none()


def calculate_cost_of_equity(
    beta: Decimal,
    risk_free_rate: Decimal,
    market_risk_premium: Decimal = Decimal("0.055")  # Historical ~5.5%
) -> Decimal:
    """
    Cost of Equity using CAPM:
    Re = Rf + β × (Rm - Rf)

    Where:
    - Re = Cost of Equity
    - Rf = Risk-free rate (10Y Treasury)
    - β = Beta (stock volatility vs market)
    - Rm - Rf = Market Risk Premium (~5.5% historical)
    """
    return risk_free_rate + (beta * market_risk_premium)


def calculate_wacc(
    cost_of_equity: Decimal,
    cost_of_debt: Decimal,
    total_debt: Decimal,
    market_cap: Decimal,
    tax_rate: Decimal = Decimal("0.21")  # US corporate tax rate
) -> Decimal:
    """
    Weighted Average Cost of Capital:
    WACC = (E/V × Re) + (D/V × Rd × (1-T))

    Where:
    - E = Market value of equity (market cap)
    - D = Market value of debt
    - V = E + D (total firm value)
    - Re = Cost of equity
    - Rd = Cost of debt (interest rate on debt)
    - T = Corporate tax rate
    """
    total_value = market_cap + total_debt
    if total_value == 0:
        return Decimal("0.10")  # Default 10%

    equity_weight = market_cap / total_value
    debt_weight = total_debt / total_value

    after_tax_cost_of_debt = cost_of_debt * (1 - tax_rate)

    wacc = (equity_weight * cost_of_equity) + (debt_weight * after_tax_cost_of_debt)

    # Sanity bounds: WACC typically between 5% and 20%
    return max(Decimal("0.05"), min(Decimal("0.20"), wacc))


def estimate_cost_of_debt(
    interest_expense: Optional[Decimal],
    total_debt: Optional[Decimal]
) -> Decimal:
    """
    Estimate cost of debt from financial statements.
    Cost of Debt ≈ Interest Expense / Total Debt

    Falls back to 5% if data unavailable.
    """
    if interest_expense and total_debt and total_debt > 0:
        implied_rate = interest_expense / total_debt
        # Sanity bounds: 2% to 15%
        return max(Decimal("0.02"), min(Decimal("0.15"), implied_rate))
    return Decimal("0.05")  # Default 5%
```

#### 5.2 DCF Model

**File**: `ic-score-service/pipelines/fair_value_calculator.py` (continued)

```python
def calculate_dcf_fair_value(
    fcf_ttm: Decimal,
    growth_rate_high: Decimal,      # First 5 years growth
    growth_rate_terminal: Decimal,  # Perpetuity growth (typically 2-3%)
    wacc: Decimal,
    shares_outstanding: Decimal,
    net_debt: Decimal,              # Total Debt - Cash
    projection_years: int = 10
) -> Tuple[Decimal, dict]:
    """
    Two-Stage DCF Model:

    Stage 1 (Years 1-5): High growth period at growth_rate_high
    Stage 2 (Years 6-10): Fade to terminal growth rate
    Terminal Value: Gordon Growth Model

    Returns:
        fair_value_per_share: Intrinsic value estimate
        details: Dict with calculation breakdown
    """
    if fcf_ttm <= 0:
        return None, {"error": "Negative FCF - DCF not applicable"}

    if wacc <= growth_rate_terminal:
        return None, {"error": "WACC must be greater than terminal growth"}

    projected_fcf = []
    current_fcf = fcf_ttm

    # Stage 1: High growth (Years 1-5)
    for year in range(1, 6):
        current_fcf = current_fcf * (1 + growth_rate_high)
        pv_factor = Decimal(1) / ((1 + wacc) ** year)
        pv_fcf = current_fcf * pv_factor
        projected_fcf.append({
            "year": year,
            "fcf": float(current_fcf),
            "pv_fcf": float(pv_fcf),
            "growth_rate": float(growth_rate_high * 100)
        })

    # Stage 2: Transition to terminal growth (Years 6-10)
    # Linear fade from high growth to terminal growth
    for year in range(6, projection_years + 1):
        # Fade factor: at year 6 = 100% of gap, at year 10 = 0%
        fade = Decimal(projection_years - year) / Decimal(projection_years - 5)
        blended_growth = growth_rate_terminal + (growth_rate_high - growth_rate_terminal) * fade

        current_fcf = current_fcf * (1 + blended_growth)
        pv_factor = Decimal(1) / ((1 + wacc) ** year)
        pv_fcf = current_fcf * pv_factor
        projected_fcf.append({
            "year": year,
            "fcf": float(current_fcf),
            "pv_fcf": float(pv_fcf),
            "growth_rate": float(blended_growth * 100)
        })

    # Sum of discounted FCFs
    sum_pv_fcf = sum(Decimal(str(p["pv_fcf"])) for p in projected_fcf)

    # Terminal Value (Gordon Growth Model)
    # TV = FCF_n+1 / (WACC - g)
    terminal_fcf = current_fcf * (1 + growth_rate_terminal)
    terminal_value = terminal_fcf / (wacc - growth_rate_terminal)
    pv_terminal = terminal_value / ((1 + wacc) ** projection_years)

    # Enterprise Value
    enterprise_value = sum_pv_fcf + pv_terminal

    # Equity Value = Enterprise Value - Net Debt
    equity_value = enterprise_value - net_debt

    # Per Share Value
    if shares_outstanding <= 0:
        return None, {"error": "Invalid shares outstanding"}

    fair_value_per_share = equity_value / shares_outstanding

    details = {
        "inputs": {
            "fcf_ttm": float(fcf_ttm),
            "growth_rate_high": float(growth_rate_high * 100),
            "growth_rate_terminal": float(growth_rate_terminal * 100),
            "wacc": float(wacc * 100),
            "shares_outstanding": float(shares_outstanding),
            "net_debt": float(net_debt)
        },
        "projected_fcf": projected_fcf,
        "terminal_value": float(terminal_value),
        "pv_terminal_value": float(pv_terminal),
        "sum_pv_fcf": float(sum_pv_fcf),
        "enterprise_value": float(enterprise_value),
        "equity_value": float(equity_value),
        "fair_value_per_share": float(fair_value_per_share)
    }

    return fair_value_per_share, details


def calculate_graham_number(
    eps: Optional[Decimal],
    book_value_per_share: Optional[Decimal]
) -> Optional[Decimal]:
    """
    Graham Number = √(22.5 × EPS × Book Value)

    Benjamin Graham's formula for maximum price a defensive investor
    should pay for a stock. Combines earnings and book value.

    The 22.5 multiplier comes from:
    - Maximum P/E of 15
    - Maximum P/B of 1.5
    - 15 × 1.5 = 22.5
    """
    if not eps or not book_value_per_share:
        return None
    if eps <= 0 or book_value_per_share <= 0:
        return None

    import math
    value = Decimal("22.5") * eps * book_value_per_share
    return Decimal(str(math.sqrt(float(value))))


def calculate_earnings_power_value(
    normalized_ebit: Decimal,
    wacc: Decimal,
    net_debt: Decimal,
    shares_outstanding: Decimal,
    tax_rate: Decimal = Decimal("0.21")  # US corporate tax rate
) -> Optional[Decimal]:
    """
    Earnings Power Value (EPV) - Bruce Greenwald's approach

    EPV = (Normalized EBIT × (1 - Tax Rate)) / WACC - Net Debt

    Assumes no growth - values the company's current sustainable earning power
    as a perpetuity. More conservative than DCF because it ignores growth.

    Key principles:
    1. Use EBIT (not Net Income) to exclude financing decisions
    2. Apply tax rate to get after-tax operating earnings
    3. Capitalize at WACC to get Enterprise Value
    4. Subtract Net Debt to get Equity Value

    Normalized EBIT should be adjusted for:
    - One-time charges/gains (restructuring, litigation, etc.)
    - Cyclical peaks/troughs (use average across cycle)
    - Accounting distortions (excessive depreciation, stock comp)
    - Maintenance capex vs growth capex distinction
    """
    if not normalized_ebit or normalized_ebit <= 0:
        return None
    if not wacc or wacc <= 0:
        return None
    if shares_outstanding <= 0:
        return None

    # After-tax operating earnings (NOPAT)
    after_tax_earnings = normalized_ebit * (1 - tax_rate)

    # Enterprise Value = perpetuity of after-tax earnings
    enterprise_value = after_tax_earnings / wacc

    # Equity Value = Enterprise Value - Net Debt
    equity_value = enterprise_value - net_debt

    # Handle negative equity value (overleveraged companies)
    if equity_value <= 0:
        return None

    return equity_value / shares_outstanding
```

#### 5.3 Main Fair Value Pipeline

**File**: `ic-score-service/pipelines/fair_value_calculator.py` (continued)

```python
async def calculate_fair_value_for_ticker(
    session: AsyncSession,
    ticker: str,
    current_price: Decimal
) -> dict:
    """
    Calculate all fair value estimates for a single ticker.
    """

    # Get TTM financials
    ttm_result = await session.execute(
        select(TTMFinancials)
        .where(TTMFinancials.ticker == ticker)
        .order_by(TTMFinancials.calculation_date.desc())
        .limit(1)
    )
    ttm = ttm_result.scalar_one_or_none()

    if not ttm:
        return {"error": "No TTM financials available"}

    # Get company info
    company_result = await session.execute(
        select(Company).where(Company.ticker == ticker)
    )
    company = company_result.scalar_one_or_none()

    # Get risk-free rate and beta
    risk_free_rate = await get_risk_free_rate(session)
    beta = await get_beta(session, ticker) or Decimal("1.0")  # Default beta = 1

    # Calculate WACC
    cost_of_equity = calculate_cost_of_equity(beta, risk_free_rate)
    cost_of_debt = estimate_cost_of_debt(ttm.interest_expense, ttm.total_debt)

    market_cap = current_price * ttm.shares_outstanding if ttm.shares_outstanding else Decimal(0)

    wacc = calculate_wacc(
        cost_of_equity=cost_of_equity,
        cost_of_debt=cost_of_debt,
        total_debt=ttm.total_debt or Decimal(0),
        market_cap=market_cap
    )

    # Estimate growth rate from historical data
    # Use 5-year revenue CAGR or default to 5%
    growth_rate = Decimal("0.05")  # Default
    # TODO: Pull from fundamental_metrics_extended once populated

    # Calculate DCF Fair Value
    net_debt = (ttm.total_debt or Decimal(0)) - (ttm.cash or Decimal(0))

    dcf_value, dcf_details = calculate_dcf_fair_value(
        fcf_ttm=ttm.free_cash_flow or Decimal(0),
        growth_rate_high=growth_rate,
        growth_rate_terminal=Decimal("0.025"),  # 2.5% perpetuity
        wacc=wacc,
        shares_outstanding=ttm.shares_outstanding or Decimal(0),
        net_debt=net_debt
    )

    # Calculate Graham Number
    graham = calculate_graham_number(
        eps=ttm.diluted_eps,
        book_value_per_share=ttm.book_value_per_share if hasattr(ttm, 'book_value_per_share') else None
    )

    # Calculate upside/downside
    dcf_upside = None
    if dcf_value and current_price and current_price > 0:
        dcf_upside = ((dcf_value - current_price) / current_price) * 100

    graham_upside = None
    if graham and current_price and current_price > 0:
        graham_upside = ((graham - current_price) / current_price) * 100

    return {
        "ticker": ticker,
        "current_price": float(current_price) if current_price else None,
        "dcf_fair_value": float(dcf_value) if dcf_value else None,
        "dcf_upside_percent": float(dcf_upside) if dcf_upside else None,
        "graham_number": float(graham) if graham else None,
        "graham_upside_percent": float(graham_upside) if graham_upside else None,
        "wacc": float(wacc * 100),
        "beta": float(beta),
        "cost_of_equity": float(cost_of_equity * 100),
        "risk_free_rate": float(risk_free_rate * 100),
        "dcf_details": dcf_details,
        "valuation_summary": get_valuation_summary(dcf_upside, graham_upside)
    }


def get_valuation_summary(dcf_upside: Optional[Decimal], graham_upside: Optional[Decimal]) -> str:
    """
    Provide a human-readable valuation summary.
    """
    if dcf_upside is None and graham_upside is None:
        return "Insufficient data for valuation"

    # Average the available upsides
    upsides = [u for u in [dcf_upside, graham_upside] if u is not None]
    avg_upside = sum(upsides) / len(upsides)

    if avg_upside > 50:
        return "Significantly Undervalued"
    elif avg_upside > 20:
        return "Undervalued"
    elif avg_upside > -10:
        return "Fairly Valued"
    elif avg_upside > -30:
        return "Overvalued"
    else:
        return "Significantly Overvalued"


async def run_fair_value_pipeline(batch_size: int = 100):
    """
    Main entry point - calculate fair values for all tickers.
    """
    async with async_session_maker() as session:
        # Get all tickers with TTM financials
        result = await session.execute(
            select(TTMFinancials.ticker).distinct()
        )
        tickers = [row[0] for row in result.fetchall()]

        logger.info(f"Calculating fair values for {len(tickers)} tickers")

        for i, ticker in enumerate(tickers):
            try:
                # Get current price from Polygon or cache
                current_price = await get_current_price(ticker)

                # Calculate fair value
                fair_value = await calculate_fair_value_for_ticker(
                    session, ticker, current_price
                )

                # Update fundamental_metrics_extended table
                await update_fair_value_metrics(session, ticker, fair_value)

                if (i + 1) % 100 == 0:
                    logger.info(f"Processed {i + 1}/{len(tickers)} tickers")
                    await session.commit()

            except Exception as e:
                logger.error(f"Error calculating fair value for {ticker}: {e}")
                continue

        await session.commit()
        logger.info("Fair value calculation complete")


if __name__ == "__main__":
    import asyncio
    asyncio.run(run_fair_value_pipeline())
```

#### 5.4 CronJob for Fair Value Calculation

**File**: `ic-score-service/k8s/ic-score-fair-value-cronjob.yaml`

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: ic-score-fair-value
  namespace: investorcenter
spec:
  schedule: "0 7 * * *"  # 7:00 AM UTC daily (after all other pipelines)
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      backoffLimit: 2
      template:
        spec:
          containers:
          - name: fair-value-calculator
            image: 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/ic-score-service:latest
            command: ["python", "-m", "pipelines.fair_value_calculator"]
            envFrom:
            - secretRef:
                name: postgres-secret
            - secretRef:
                name: polygon-api-secret
            resources:
              requests:
                memory: "512Mi"
                cpu: "250m"
              limits:
                memory: "1Gi"
                cpu: "500m"
          restartPolicy: OnFailure
```

#### 5.5 API Endpoint for Fair Value

**File**: `backend/handlers/fair_value_handlers.go` (new file)

```go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

type FairValueResponse struct {
    Ticker              string  `json:"ticker"`
    CurrentPrice        float64 `json:"current_price"`

    // DCF Valuation
    DCFFairValue        *float64 `json:"dcf_fair_value"`
    DCFUpsidePercent    *float64 `json:"dcf_upside_percent"`

    // Graham Valuation
    GrahamNumber        *float64 `json:"graham_number"`
    GrahamUpsidePercent *float64 `json:"graham_upside_percent"`

    // WACC Components
    WACC                float64 `json:"wacc"`
    Beta                float64 `json:"beta"`
    CostOfEquity        float64 `json:"cost_of_equity"`
    RiskFreeRate        float64 `json:"risk_free_rate"`

    // Summary
    ValuationSummary    string  `json:"valuation_summary"`

    // Calculation timestamp
    CalculatedAt        string  `json:"calculated_at"`
}

// GetFairValue returns DCF and Graham valuations for a ticker
func GetFairValue(c *gin.Context) {
    symbol := c.Param("symbol")

    // Query from fundamental_metrics_extended or ic-score-service
    fairValue, err := fetchFairValue(symbol)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Fair value not available"})
        return
    }

    c.JSON(http.StatusOK, fairValue)
}

// Add route in main.go:
// v1.GET("/tickers/:symbol/fair-value", handlers.GetFairValue)
```

---

## Pipeline Execution Order

After implementation, CronJobs should run in this order:

| Order | CronJob | Schedule (UTC) | Dependencies |
|-------|---------|----------------|--------------|
| 1 | `ic-score-sec-financials` | 2:00 AM | None |
| 2 | `ic-score-ttm-financials` | 3:00 AM | SEC Financials |
| 3 | `ic-score-technical-indicators` | 3:30 AM | Stock Prices |
| 4 | `ic-score-risk-metrics` | 4:00 AM | Stock Prices |
| 5 | `ic-score-fundamental-metrics` | 5:00 AM | TTM Financials, Dividends |
| 6 | `ic-score-refresh-sector-views` | 6:30 AM | Fundamental Metrics |
| 7 | `ic-score-fair-value` | 7:00 AM | All above |
| 8 | `ic-score-calculator` | 8:00 AM | All above |

---

## Summary Table

| Category | Metrics | Status |
|----------|---------|--------|
| **Profitability Margins** | Gross, Operating, Net, EBITDA | 75% Ready |
| **Returns** | ROA, ROE, ROIC | 100% Ready |
| **Growth Rates** | Revenue/EPS/FCF YoY, 3Y, 5Y | Data Ready, Calc Needed |
| **Valuation** | EV, EV/Revenue, EV/EBITDA, EV/FCF | 75% Ready |
| **Liquidity** | Current Ratio, Quick Ratio | 100% Ready |
| **Debt/Leverage** | D/E, Interest Coverage, Net Debt/EBITDA | 33% Ready |
| **Dividends** | Yield, Payout, Growth, Years | 25% Ready |
| **Sector Comparisons** | All metric averages by sector | Schema Ready, Build Needed |
| **Fair Value** | DCF-based estimate | Data Ready, Model Needed |

---

## Recommendations

### No Additional API Cost Required

All requested metrics can be derived from:
1. **Polygon.io** (current subscription) - fundamentals, prices
2. **SEC EDGAR** (free) - enhanced financial extraction
3. **Internal calculations** - growth rates, ratios, DCF

### Optional Future Enhancements

If you want premium features later:

| Feature | API Option | Cost |
|---------|------------|------|
| Consensus Estimates | Polygon Premium / Refinitiv | $500+/mo |
| Options Data | Polygon Options | $79/mo addon |
| Short Interest | FINRA / Ortex | $50-200/mo |
| ESG Scores | MSCI / Sustainalytics | Enterprise pricing |
| Earnings Transcripts | AlphaSense / Sentieo | Enterprise pricing |

### Immediate Action Items

1. **Create new pipeline**: `ic-score-service/pipelines/fundamental_metrics_calculator.py`
2. **Add database table**: `fundamental_metrics_extended` for calculated metrics
3. **Create CronJob**: `ic-score-fundamental-metrics-cronjob.yaml` (run daily after financials update)
4. **Update API**: Add new endpoint `/api/v1/tickers/:symbol/metrics` for extended fundamentals

---

## Appendix: Database Schema Additions

```sql
-- New table for extended fundamental metrics
CREATE TABLE fundamental_metrics_extended (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    calculation_date DATE NOT NULL,

    -- Profitability Margins
    gross_margin DECIMAL(10, 4),
    operating_margin DECIMAL(10, 4),
    net_margin DECIMAL(10, 4),
    ebitda_margin DECIMAL(10, 4),

    -- Returns
    roe DECIMAL(10, 4),
    roa DECIMAL(10, 4),
    roic DECIMAL(10, 4),

    -- Growth Rates (as percentages)
    revenue_growth_yoy DECIMAL(10, 4),
    revenue_growth_3y_cagr DECIMAL(10, 4),
    revenue_growth_5y_cagr DECIMAL(10, 4),
    eps_growth_yoy DECIMAL(10, 4),
    eps_growth_3y_cagr DECIMAL(10, 4),
    eps_growth_5y_cagr DECIMAL(10, 4),
    fcf_growth_yoy DECIMAL(10, 4),

    -- Valuation
    enterprise_value DECIMAL(20, 2),
    ev_to_revenue DECIMAL(12, 4),
    ev_to_ebitda DECIMAL(12, 4),
    ev_to_fcf DECIMAL(12, 4),

    -- Liquidity
    current_ratio DECIMAL(10, 4),
    quick_ratio DECIMAL(10, 4),

    -- Debt/Leverage
    debt_to_equity DECIMAL(10, 4),
    interest_coverage DECIMAL(10, 4),
    net_debt_to_ebitda DECIMAL(10, 4),

    -- Dividends
    dividend_yield DECIMAL(10, 4),
    payout_ratio DECIMAL(10, 4),
    dividend_growth_rate DECIMAL(10, 4),
    consecutive_dividend_years INTEGER,

    -- Fair Value
    dcf_fair_value DECIMAL(14, 2),
    dcf_upside_percent DECIMAL(10, 4),
    graham_number DECIMAL(14, 2),

    -- Sector Comparisons (percentile ranks 0-100)
    pe_sector_percentile INTEGER,
    pb_sector_percentile INTEGER,
    roe_sector_percentile INTEGER,
    margin_sector_percentile INTEGER,

    -- Metadata
    data_quality_score DECIMAL(5, 2),  -- 0-100% completeness
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(symbol, calculation_date)
);

CREATE INDEX idx_fund_metrics_symbol ON fundamental_metrics_extended(symbol);
CREATE INDEX idx_fund_metrics_date ON fundamental_metrics_extended(calculation_date DESC);
```

---

## Appendix B: Financial Calculation Review Notes

*Reviewed: November 27, 2025*

### Formulas Verified ✅

| Calculation | Formula | Status |
|-------------|---------|--------|
| EBITDA Margin | EBITDA / Revenue × 100 | ✅ Correct |
| CAGR | ((End/Start)^(1/n) - 1) × 100 | ✅ Correct |
| Payout Ratio | DPS / EPS × 100 | ✅ Correct |
| Interest Coverage | EBIT / Interest Expense | ✅ Correct |
| Net Debt/EBITDA | (Total Debt - Cash) / EBITDA | ✅ Correct |
| CAPM Cost of Equity | Rf + β × (Rm - Rf) | ✅ Correct |
| WACC | (E/V × Re) + (D/V × Rd × (1-T)) | ✅ Correct |
| DCF Terminal Value | FCF × (1+g) / (WACC - g) | ✅ Correct |
| Graham Number | √(22.5 × EPS × BVPS) | ✅ Correct |

### Formulas Corrected ⚠️

| Calculation | Issue | Resolution |
|-------------|-------|------------|
| **YoY Growth** | Using `abs(prior)` caused misleading results when EPS/FCF crosses zero | Added `metric_type` parameter; returns `None` for sign changes (turnarounds) |
| **Consecutive Dividend Years** | Documentation clarified | Function counts number of **increases**, which is standard industry practice |
| **Earnings Power Value** | Missing tax adjustment and net debt | Added tax rate parameter and proper equity value calculation |
| **Percentile Rank** | Only counted values below, not equal | Added standard formula: `(B + 0.5E) / N × 100` for proper tie handling |

### Acceptable Simplifications

| Item | Simplification | Impact |
|------|----------------|--------|
| **Cost of Debt** | Uses Interest Expense / Total Debt (historical average) | For screening purposes, acceptable; YTM would be more accurate |
| **DCF Equity Value** | Does not subtract minority interests, preferred stock, pension liabilities | Acceptable for screening; detailed models should include these |
| **WACC Tax Rate** | Fixed at 21% US corporate rate | Could be enhanced to use company-specific effective tax rate |
| **Growth Rate** | Default 5% if historical CAGR unavailable | Conservative default; should pull from `fundamental_metrics_extended` once populated |

### Edge Case Handling

| Scenario | Handling |
|----------|----------|
| Negative EPS → Positive EPS | YoY Growth returns `None` (not meaningful) |
| Both EPS negative but improving | Returns positive growth (loss narrowing) |
| Negative FCF | DCF returns `None` with error message |
| Zero Interest Expense | Interest Coverage returns `None` (not infinity) |
| Negative EBITDA | Net Debt/EBITDA returns `None` |
| Company has more cash than debt | Net Debt/EBITDA can be negative (net cash position) |

### Industry Standard Interpretations

**Interest Coverage Ratio:**
- \> 5.0: Excellent, very low risk
- 3.0 - 5.0: Good, healthy
- 2.0 - 3.0: Adequate, some risk
- 1.0 - 2.0: Poor, high default risk
- < 1.0: Cannot cover interest (distressed)

**Net Debt / EBITDA:**
- < 1.0: Very low leverage
- 1.0 - 2.0: Conservative
- 2.0 - 3.0: Moderate
- 3.0 - 4.0: Elevated (watch closely)
- \> 4.0: High leverage (potential distress)
- *Note: Utilities, REITs typically run 4-6x*

**Payout Ratio:**
- < 30%: Very conservative, high growth reinvestment
- 30% - 50%: Healthy, sustainable
- 50% - 75%: Moderate risk
- \> 75%: High risk, may not be sustainable
- \> 100%: Paying more than earning (using reserves)

**Dividend Growth Streaks:**
- 5-9 years: Dividend Challenger
- 10-24 years: Dividend Contender
- 25+ years: Dividend Champion / Aristocrat
