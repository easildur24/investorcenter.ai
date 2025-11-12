# Technical Specification: Implementation Details
## InvestorCenter.ai - Database Schemas & Core Algorithms

**Version:** 1.0
**Date:** November 12, 2025
**Status:** Draft

---

## Table of Contents

1. [Database Schema](#database-schema)
2. [IC Score Calculation Algorithm](#ic-score-calculation-algorithm)
3. [Data Models](#data-models)
4. [Caching Strategy](#caching-strategy)
5. [Security Implementation](#security-implementation)
6. [Performance Optimization](#performance-optimization)

---

## Database Schema

### PostgreSQL Tables

**users**
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    subscription_tier VARCHAR(50) DEFAULT 'free',
    stripe_customer_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_login_at TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    email_verified BOOLEAN DEFAULT false,
    preferences JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_subscription ON users(subscription_tier);
```

**companies**
```sql
CREATE TABLE companies (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    sector VARCHAR(100),
    industry VARCHAR(100),
    market_cap BIGINT,
    country VARCHAR(50),
    exchange VARCHAR(50),
    currency VARCHAR(10),
    website VARCHAR(255),
    description TEXT,
    employees INT,
    founded_year INT,
    hq_location VARCHAR(255),
    logo_url VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    last_updated TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_companies_ticker ON companies(ticker);
CREATE INDEX idx_companies_sector ON companies(sector);
CREATE INDEX idx_companies_market_cap ON companies(market_cap);
```

**ic_scores**
```sql
CREATE TABLE ic_scores (
    id BIGSERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    date DATE NOT NULL,
    overall_score DECIMAL(5,2) NOT NULL,
    value_score DECIMAL(5,2),
    growth_score DECIMAL(5,2),
    profitability_score DECIMAL(5,2),
    financial_health_score DECIMAL(5,2),
    momentum_score DECIMAL(5,2),
    analyst_consensus_score DECIMAL(5,2),
    insider_activity_score DECIMAL(5,2),
    institutional_score DECIMAL(5,2),
    news_sentiment_score DECIMAL(5,2),
    technical_score DECIMAL(5,2),
    rating VARCHAR(20), -- Strong Buy, Buy, Hold, etc.
    sector_percentile DECIMAL(5,2),
    confidence_level VARCHAR(20), -- High, Medium, Low
    data_completeness DECIMAL(5,2), -- % of factors with data
    calculation_metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(ticker, date)
);

CREATE INDEX idx_ic_scores_ticker ON ic_scores(ticker);
CREATE INDEX idx_ic_scores_date ON ic_scores(date);
CREATE INDEX idx_ic_scores_overall ON ic_scores(overall_score DESC);
CREATE INDEX idx_ic_scores_ticker_date ON ic_scores(ticker, date DESC);
```

**financials**
```sql
CREATE TABLE financials (
    id BIGSERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    filing_date DATE NOT NULL,
    period_end_date DATE NOT NULL,
    fiscal_year INT NOT NULL,
    fiscal_quarter INT, -- NULL for annual
    statement_type VARCHAR(20), -- '10-K' or '10-Q'
    -- Income Statement
    revenue BIGINT,
    cost_of_revenue BIGINT,
    gross_profit BIGINT,
    operating_expenses BIGINT,
    operating_income BIGINT,
    net_income BIGINT,
    eps_basic DECIMAL(10,4),
    eps_diluted DECIMAL(10,4),
    shares_outstanding BIGINT,
    -- Balance Sheet
    total_assets BIGINT,
    total_liabilities BIGINT,
    shareholders_equity BIGINT,
    cash_and_equivalents BIGINT,
    short_term_debt BIGINT,
    long_term_debt BIGINT,
    -- Cash Flow
    operating_cash_flow BIGINT,
    investing_cash_flow BIGINT,
    financing_cash_flow BIGINT,
    free_cash_flow BIGINT,
    capex BIGINT,
    -- Calculated Metrics
    pe_ratio DECIMAL(10,2),
    pb_ratio DECIMAL(10,2),
    ps_ratio DECIMAL(10,2),
    debt_to_equity DECIMAL(10,2),
    current_ratio DECIMAL(10,2),
    quick_ratio DECIMAL(10,2),
    roe DECIMAL(10,2),
    roa DECIMAL(10,2),
    roic DECIMAL(10,2),
    gross_margin DECIMAL(10,2),
    operating_margin DECIMAL(10,2),
    net_margin DECIMAL(10,2),
    -- Metadata
    sec_filing_url VARCHAR(500),
    raw_data JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(ticker, period_end_date, fiscal_quarter)
);

CREATE INDEX idx_financials_ticker ON financials(ticker);
CREATE INDEX idx_financials_period ON financials(period_end_date DESC);
CREATE INDEX idx_financials_ticker_period ON financials(ticker, period_end_date DESC);
```

**insider_trades**
```sql
CREATE TABLE insider_trades (
    id BIGSERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    filing_date DATE NOT NULL,
    transaction_date DATE NOT NULL,
    insider_name VARCHAR(255) NOT NULL,
    insider_title VARCHAR(255),
    transaction_type VARCHAR(50), -- 'Buy', 'Sell', 'Option Exercise', etc.
    shares BIGINT NOT NULL,
    price_per_share DECIMAL(10,2),
    total_value BIGINT,
    shares_owned_after BIGINT,
    is_derivative BOOLEAN DEFAULT false,
    form_type VARCHAR(10), -- 'Form 4', 'Form 3', etc.
    sec_filing_url VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_insider_ticker ON insider_trades(ticker);
CREATE INDEX idx_insider_date ON insider_trades(transaction_date DESC);
CREATE INDEX idx_insider_ticker_date ON insider_trades(ticker, transaction_date DESC);
```

**institutional_holdings**
```sql
CREATE TABLE institutional_holdings (
    id BIGSERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    filing_date DATE NOT NULL,
    quarter_end_date DATE NOT NULL,
    institution_name VARCHAR(255) NOT NULL,
    institution_cik VARCHAR(20),
    shares BIGINT NOT NULL,
    market_value BIGINT NOT NULL,
    percent_of_portfolio DECIMAL(10,4),
    position_change VARCHAR(50), -- 'New', 'Increased', 'Decreased', 'Sold Out'
    shares_change BIGINT,
    percent_change DECIMAL(10,2),
    sec_filing_url VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(ticker, quarter_end_date, institution_cik)
);

CREATE INDEX idx_institutional_ticker ON institutional_holdings(ticker);
CREATE INDEX idx_institutional_date ON institutional_holdings(quarter_end_date DESC);
CREATE INDEX idx_institutional_ticker_date ON institutional_holdings(ticker, quarter_end_date DESC);
```

**analyst_ratings**
```sql
CREATE TABLE analyst_ratings (
    id BIGSERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    rating_date DATE NOT NULL,
    analyst_name VARCHAR(255) NOT NULL,
    analyst_firm VARCHAR(255),
    rating VARCHAR(50) NOT NULL, -- 'Strong Buy', 'Buy', 'Hold', 'Sell', 'Strong Sell'
    rating_numeric DECIMAL(3,1), -- 1.0 to 5.0
    price_target DECIMAL(10,2),
    prior_rating VARCHAR(50),
    prior_price_target DECIMAL(10,2),
    action VARCHAR(50), -- 'Initiated', 'Upgraded', 'Downgraded', 'Reiterated'
    notes TEXT,
    source VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_analyst_ticker ON analyst_ratings(ticker);
CREATE INDEX idx_analyst_date ON analyst_ratings(rating_date DESC);
CREATE INDEX idx_analyst_ticker_date ON analyst_ratings(ticker, rating_date DESC);
```

**news_articles**
```sql
CREATE TABLE news_articles (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    url VARCHAR(1000) UNIQUE NOT NULL,
    source VARCHAR(255) NOT NULL,
    published_at TIMESTAMP NOT NULL,
    summary TEXT,
    content TEXT,
    author VARCHAR(255),
    tickers VARCHAR(50)[], -- Array of related tickers
    sentiment_score DECIMAL(5,2), -- -100 to +100
    sentiment_label VARCHAR(20), -- 'Positive', 'Neutral', 'Negative'
    relevance_score DECIMAL(5,2), -- 0 to 100
    categories VARCHAR(50)[],
    image_url VARCHAR(500),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_news_published ON news_articles(published_at DESC);
CREATE INDEX idx_news_tickers ON news_articles USING GIN(tickers);
CREATE INDEX idx_news_sentiment ON news_articles(sentiment_score);
```

**watchlists**
```sql
CREATE TABLE watchlists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT false,
    color VARCHAR(20),
    sort_order INT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_watchlists_user ON watchlists(user_id);
```

**watchlist_stocks**
```sql
CREATE TABLE watchlist_stocks (
    id BIGSERIAL PRIMARY KEY,
    watchlist_id UUID NOT NULL REFERENCES watchlists(id) ON DELETE CASCADE,
    ticker VARCHAR(10) NOT NULL,
    notes TEXT,
    position INT, -- Order in list
    added_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(watchlist_id, ticker)
);

CREATE INDEX idx_watchlist_stocks_watchlist ON watchlist_stocks(watchlist_id);
```

**portfolios**
```sql
CREATE TABLE portfolios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    currency VARCHAR(10) DEFAULT 'USD',
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_portfolios_user ON portfolios(user_id);
```

**portfolio_positions**
```sql
CREATE TABLE portfolio_positions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    ticker VARCHAR(10) NOT NULL,
    shares DECIMAL(18,6) NOT NULL, -- Support fractional shares
    average_cost DECIMAL(10,2) NOT NULL,
    first_purchased_at TIMESTAMP,
    last_updated_at TIMESTAMP DEFAULT NOW(),
    notes TEXT,
    UNIQUE(portfolio_id, ticker)
);

CREATE INDEX idx_positions_portfolio ON portfolio_positions(portfolio_id);
```

**portfolio_transactions**
```sql
CREATE TABLE portfolio_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id UUID NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
    ticker VARCHAR(10) NOT NULL,
    transaction_type VARCHAR(20) NOT NULL, -- 'BUY', 'SELL', 'DIVIDEND', 'SPLIT'
    shares DECIMAL(18,6) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    total_amount DECIMAL(18,2) NOT NULL,
    fees DECIMAL(10,2) DEFAULT 0,
    transaction_date DATE NOT NULL,
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_transactions_portfolio ON portfolio_transactions(portfolio_id);
CREATE INDEX idx_transactions_date ON portfolio_transactions(transaction_date DESC);
```

**alerts**
```sql
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ticker VARCHAR(10) NOT NULL,
    alert_type VARCHAR(50) NOT NULL, -- 'PRICE', 'IC_SCORE', 'NEWS', 'EARNINGS', etc.
    condition JSONB NOT NULL, -- Flexible condition storage
    delivery_method VARCHAR(50)[] DEFAULT ARRAY['email'], -- email, push, sms
    is_active BOOLEAN DEFAULT true,
    triggered_count INT DEFAULT 0,
    last_triggered_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP
);

CREATE INDEX idx_alerts_user ON alerts(user_id);
CREATE INDEX idx_alerts_ticker ON alerts(ticker);
CREATE INDEX idx_alerts_active ON alerts(is_active) WHERE is_active = true;
```

### TimescaleDB Tables (Time-Series)

**stock_prices** (Hypertable)
```sql
CREATE TABLE stock_prices (
    time TIMESTAMPTZ NOT NULL,
    ticker VARCHAR(10) NOT NULL,
    open DECIMAL(10,2),
    high DECIMAL(10,2),
    low DECIMAL(10,2),
    close DECIMAL(10,2),
    volume BIGINT,
    vwap DECIMAL(10,2), -- Volume-weighted average price
    interval VARCHAR(10) DEFAULT '1day' -- '1min', '1hour', '1day', etc.
);

-- Convert to hypertable
SELECT create_hypertable('stock_prices', 'time');

-- Create indexes
CREATE INDEX idx_prices_ticker_time ON stock_prices(ticker, time DESC);
CREATE INDEX idx_prices_time ON stock_prices(time DESC);

-- Continuous aggregates for different timeframes
CREATE MATERIALIZED VIEW stock_prices_daily
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 day', time) AS day,
    ticker,
    first(open, time) AS open,
    max(high) AS high,
    min(low) AS low,
    last(close, time) AS close,
    sum(volume) AS volume
FROM stock_prices
WHERE interval = '1min'
GROUP BY day, ticker;
```

**technical_indicators** (Hypertable)
```sql
CREATE TABLE technical_indicators (
    time TIMESTAMPTZ NOT NULL,
    ticker VARCHAR(10) NOT NULL,
    indicator_name VARCHAR(50) NOT NULL,
    value DECIMAL(18,6),
    metadata JSONB
);

SELECT create_hypertable('technical_indicators', 'time');

CREATE INDEX idx_indicators_ticker_time ON technical_indicators(ticker, time DESC);
CREATE INDEX idx_indicators_name ON technical_indicators(indicator_name);
```

---

## IC Score Calculation Algorithm

### Python Implementation

```python
from decimal import Decimal
from typing import Dict, Optional
from dataclasses import dataclass
from datetime import date

@dataclass
class FactorScore:
    value: Decimal  # 0-100
    weight: Decimal  # 0-1, sum to 1.0
    percentile: Optional[Decimal] = None  # 0-100
    trend: Optional[str] = None  # 'up', 'down', 'stable'

@dataclass
class ICScore:
    ticker: str
    date: date
    overall_score: Decimal  # 1-100
    factors: Dict[str, FactorScore]
    rating: str  # 'Strong Buy', 'Buy', 'Hold', 'Sell', 'Strong Sell'
    sector_percentile: Decimal
    confidence_level: str
    data_completeness: Decimal

class ICScoreCalculator:
    """Calculate InvestorCenter Score for a stock."""

    DEFAULT_WEIGHTS = {
        'value': Decimal('0.12'),
        'growth': Decimal('0.15'),
        'profitability': Decimal('0.12'),
        'financial_health': Decimal('0.10'),
        'momentum': Decimal('0.08'),
        'analyst_consensus': Decimal('0.10'),
        'insider_activity': Decimal('0.08'),
        'institutional': Decimal('0.10'),
        'news_sentiment': Decimal('0.07'),
        'technical': Decimal('0.08')
    }

    def __init__(self, db_session, weights: Optional[Dict[str, Decimal]] = None):
        self.db = db_session
        self.weights = weights or self.DEFAULT_WEIGHTS

    def calculate(self, ticker: str, as_of_date: Optional[date] = None) -> ICScore:
        """Calculate IC Score for a stock."""
        if as_of_date is None:
            as_of_date = date.today()

        # Calculate each factor score
        factors = {
            'value': self._calculate_value_factor(ticker, as_of_date),
            'growth': self._calculate_growth_factor(ticker, as_of_date),
            'profitability': self._calculate_profitability_factor(ticker, as_of_date),
            'financial_health': self._calculate_financial_health_factor(ticker, as_of_date),
            'momentum': self._calculate_momentum_factor(ticker, as_of_date),
            'analyst_consensus': self._calculate_analyst_factor(ticker, as_of_date),
            'insider_activity': self._calculate_insider_factor(ticker, as_of_date),
            'institutional': self._calculate_institutional_factor(ticker, as_of_date),
            'news_sentiment': self._calculate_sentiment_factor(ticker, as_of_date),
            'technical': self._calculate_technical_factor(ticker, as_of_date)
        }

        # Calculate weighted overall score
        overall = self._calculate_overall_score(factors)

        # Determine rating
        rating = self._get_rating(overall)

        # Calculate sector percentile
        sector = self._get_company_sector(ticker)
        sector_percentile = self._calculate_sector_percentile(ticker, overall, sector, as_of_date)

        # Confidence and completeness
        confidence = self._calculate_confidence(factors)
        completeness = self._calculate_completeness(factors)

        return ICScore(
            ticker=ticker,
            date=as_of_date,
            overall_score=overall,
            factors=factors,
            rating=rating,
            sector_percentile=sector_percentile,
            confidence_level=confidence,
            data_completeness=completeness
        )

    def _calculate_value_factor(self, ticker: str, as_of_date: date) -> FactorScore:
        """Calculate value score based on valuation multiples."""
        # Get latest financials
        financials = self._get_latest_financials(ticker, as_of_date)
        if not financials:
            return FactorScore(value=Decimal('50'), weight=self.weights['value'])

        # Get sector for comparison
        sector = self._get_company_sector(ticker)
        sector_medians = self._get_sector_medians(sector, as_of_date)

        # Calculate individual metric scores
        metrics = {}

        # P/E Ratio (lower is better)
        if financials.pe_ratio and sector_medians.get('pe_ratio'):
            pe_percentile = self._calculate_percentile(
                ticker,
                'pe_ratio',
                financials.pe_ratio,
                sector,
                as_of_date,
                lower_is_better=True
            )
            metrics['pe'] = pe_percentile

        # P/B Ratio (lower is better)
        if financials.pb_ratio and sector_medians.get('pb_ratio'):
            pb_percentile = self._calculate_percentile(
                ticker,
                'pb_ratio',
                financials.pb_ratio,
                sector,
                as_of_date,
                lower_is_better=True
            )
            metrics['pb'] = pb_percentile

        # P/S Ratio (lower is better)
        if financials.ps_ratio and sector_medians.get('ps_ratio'):
            ps_percentile = self._calculate_percentile(
                ticker,
                'ps_ratio',
                financials.ps_ratio,
                sector,
                as_of_date,
                lower_is_better=True
            )
            metrics['ps'] = ps_percentile

        # PEG Ratio (lower is better, but need growth data)
        peg_ratio = self._calculate_peg_ratio(ticker, as_of_date)
        if peg_ratio:
            peg_percentile = self._calculate_percentile(
                ticker,
                'peg_ratio',
                peg_ratio,
                sector,
                as_of_date,
                lower_is_better=True
            )
            metrics['peg'] = peg_percentile

        # Average all available metrics
        if metrics:
            avg_score = sum(metrics.values()) / len(metrics)
            value_score = Decimal(str(avg_score))
        else:
            value_score = Decimal('50')  # Neutral if no data

        # Calculate trend (compare to 30 days ago)
        trend = self._calculate_factor_trend(ticker, 'value', as_of_date)

        return FactorScore(
            value=value_score,
            weight=self.weights['value'],
            percentile=value_score,
            trend=trend
        )

    def _calculate_growth_factor(self, ticker: str, as_of_date: date) -> FactorScore:
        """Calculate growth score based on revenue, earnings, FCF growth."""
        # Get historical financials
        financials_history = self._get_financials_history(ticker, years=5, as_of_date=as_of_date)
        if len(financials_history) < 2:
            return FactorScore(value=Decimal('50'), weight=self.weights['growth'])

        growth_metrics = {}

        # Revenue growth (1Y, 3Y, 5Y CAGR)
        revenue_growth = self._calculate_cagr(
            [f.revenue for f in financials_history if f.revenue],
            [f.period_end_date for f in financials_history if f.revenue]
        )
        if revenue_growth:
            sector = self._get_company_sector(ticker)
            revenue_percentile = self._calculate_percentile(
                ticker, 'revenue_growth', revenue_growth['5y'], sector, as_of_date
            )
            growth_metrics['revenue'] = revenue_percentile

        # EPS growth
        eps_growth = self._calculate_cagr(
            [f.eps_diluted for f in financials_history if f.eps_diluted],
            [f.period_end_date for f in financials_history if f.eps_diluted]
        )
        if eps_growth:
            eps_percentile = self._calculate_percentile(
                ticker, 'eps_growth', eps_growth['5y'], sector, as_of_date
            )
            growth_metrics['eps'] = eps_percentile

        # FCF growth
        fcf_growth = self._calculate_cagr(
            [f.free_cash_flow for f in financials_history if f.free_cash_flow],
            [f.period_end_date for f in financials_history if f.free_cash_flow]
        )
        if fcf_growth:
            fcf_percentile = self._calculate_percentile(
                ticker, 'fcf_growth', fcf_growth['5y'], sector, as_of_date
            )
            growth_metrics['fcf'] = fcf_percentile

        # Forward growth estimates (from analysts)
        forward_growth = self._get_forward_growth_estimates(ticker, as_of_date)
        if forward_growth:
            # Weight forward estimates at 30% of growth score
            growth_metrics['forward'] = forward_growth

        # Average growth metrics
        if growth_metrics:
            # Weight historical 70%, forward 30%
            if 'forward' in growth_metrics:
                historical_avg = sum(v for k, v in growth_metrics.items() if k != 'forward') / (len(growth_metrics) - 1)
                growth_score = Decimal(str(historical_avg * 0.7 + growth_metrics['forward'] * 0.3))
            else:
                growth_score = Decimal(str(sum(growth_metrics.values()) / len(growth_metrics)))
        else:
            growth_score = Decimal('50')

        # Consistency bonus (penalize volatile growth)
        consistency_factor = self._calculate_growth_consistency(financials_history)
        growth_score = growth_score * consistency_factor
        growth_score = min(growth_score, Decimal('100'))

        trend = self._calculate_factor_trend(ticker, 'growth', as_of_date)

        return FactorScore(
            value=growth_score,
            weight=self.weights['growth'],
            percentile=growth_score,
            trend=trend
        )

    # ... (similar implementations for other factors)

    def _calculate_overall_score(self, factors: Dict[str, FactorScore]) -> Decimal:
        """Calculate weighted overall IC Score."""
        total_score = Decimal('0')
        total_weight = Decimal('0')

        for factor_name, factor in factors.items():
            if factor.value is not None:
                total_score += factor.value * factor.weight
                total_weight += factor.weight

        if total_weight > 0:
            # Normalize if some factors missing
            overall = total_score / total_weight
        else:
            overall = Decimal('50')  # Default neutral

        # Round to 2 decimal places, clamp to 1-100
        overall = max(Decimal('1'), min(Decimal('100'), overall.quantize(Decimal('0.01'))))

        return overall

    def _get_rating(self, score: Decimal) -> str:
        """Convert numerical score to rating label."""
        if score >= 80:
            return 'Strong Buy'
        elif score >= 65:
            return 'Buy'
        elif score >= 50:
            return 'Hold'
        elif score >= 35:
            return 'Underperform'
        else:
            return 'Sell'

    def _calculate_sector_percentile(
        self,
        ticker: str,
        score: Decimal,
        sector: str,
        as_of_date: date
    ) -> Decimal:
        """Calculate where this stock ranks within its sector."""
        # Get all stocks in sector with scores
        sector_scores = self.db.query(ICScores).filter(
            ICScores.date == as_of_date,
            Companies.sector == sector
        ).join(Companies, ICScores.ticker == Companies.ticker).all()

        if not sector_scores:
            return Decimal('50')  # Default

        # Calculate percentile
        scores_list = [s.overall_score for s in sector_scores]
        percentile = (sum(1 for s in scores_list if s < score) / len(scores_list)) * 100

        return Decimal(str(percentile)).quantize(Decimal('0.01'))

    def _calculate_confidence(self, factors: Dict[str, FactorScore]) -> str:
        """Calculate confidence level based on data availability and quality."""
        # Count factors with data
        factors_with_data = sum(1 for f in factors.values() if f.value is not None)
        completeness = factors_with_data / len(factors)

        if completeness >= 0.9:
            return 'High'
        elif completeness >= 0.7:
            return 'Medium'
        else:
            return 'Low'

    def _calculate_completeness(self, factors: Dict[str, FactorScore]) -> Decimal:
        """Calculate data completeness percentage."""
        factors_with_data = sum(1 for f in factors.values() if f.value is not None)
        return Decimal(str(factors_with_data / len(factors) * 100)).quantize(Decimal('0.01'))

# Usage example
if __name__ == "__main__":
    from database import SessionLocal

    db = SessionLocal()
    calculator = ICScoreCalculator(db)

    # Calculate IC Score for AAPL
    score = calculator.calculate('AAPL')

    print(f"{score.ticker} IC Score: {score.overall_score}/100 ({score.rating})")
    print(f"Data Completeness: {score.data_completeness}%")
    print(f"Confidence: {score.confidence_level}")
    print("\nFactor Breakdown:")
    for name, factor in score.factors.items():
        print(f"  {name.capitalize()}: {factor.value}/100 ({factor.weight*100}% weight)")
```

---

## Caching Strategy

### Cache Layers

**Level 1: Browser Cache**
- Static assets (JS, CSS, images): 1 year
- HTML pages: No cache (or 5 min for mostly static pages)

**Level 2: CDN Cache (CloudFlare)**
- Static assets: 1 year
- API responses (public data): 1 hour to 1 day
- Purge on deploy or data update

**Level 3: Application Cache (Redis)**

```python
CACHE_TTLS = {
    # Stock data
    'stock:overview:{ticker}': 3600,  # 1 hour
    'stock:score:{ticker}': 86400,  # 24 hours (recalc daily)
    'stock:price:{ticker}': 60,  # 1 minute (market hours)
    'stock:price:{ticker}:afterhours': 3600,  # 1 hour (off hours)
    'stock:financials:{ticker}': 86400,  # 24 hours
    'stock:news:{ticker}': 900,  # 15 minutes

    # Screener
    'screener:results:{hash}': 3600,  # 1 hour
    'screener:count:{hash}': 1800,  # 30 minutes

    # Lists
    'watchlist:{user_id}:{id}': 300,  # 5 minutes
    'portfolio:{user_id}:{id}': 300,  # 5 minutes

    # User data
    'user:{user_id}:profile': 3600,  # 1 hour
    'user:{user_id}:subscription': 3600,  # 1 hour

    # Aggregates
    'sector:medians:{sector}': 86400,  # 24 hours
    'market:indices': 300,  # 5 minutes
}
```

**Cache Invalidation:**
```python
def invalidate_stock_cache(ticker: str):
    """Invalidate all cache entries for a stock."""
    redis_client.delete(
        f'stock:overview:{ticker}',
        f'stock:score:{ticker}',
        f'stock:price:{ticker}',
        f'stock:financials:{ticker}',
        f'stock:news:{ticker}'
    )

def invalidate_on_score_recalculation(ticker: str):
    """Called after IC Score recalculated."""
    invalidate_stock_cache(ticker)
    # Also invalidate screener caches that might include this stock
    redis_client.delete_pattern('screener:results:*')
```

---

## Performance Optimization

### Database Optimizations

1. **Indexes:** Comprehensive indexes on all foreign keys and frequently queried columns
2. **Partitioning:** Partition large tables by date (stock_prices, ic_scores)
3. **Materialized Views:** Pre-aggregate common queries
4. **Read Replicas:** Separate read/write databases
5. **Connection Pooling:** Use pgBouncer

### Application Optimizations

1. **Async I/O:** Use async/await for all I/O operations
2. **Bulk Operations:** Batch database queries
3. **Lazy Loading:** Load data only when needed
4. **Pagination:** Never load full datasets
5. **Compression:** Gzip API responses

### Frontend Optimizations

1. **Code Splitting:** Load code on demand
2. **Image Optimization:** WebP format, lazy loading
3. **Bundle Size:** Tree shaking, minification
4. **Prefetching:** Prefetch likely next pages
5. **Service Workers:** Offline caching

---

**End of Technical Specification Part 2**
