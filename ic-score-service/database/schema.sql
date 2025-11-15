-- InvestorCenter.ai Database Schema
-- PostgreSQL 15+ with TimescaleDB Extension
-- Version: 1.0
-- Date: November 12, 2025

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "timescaledb";

-- ============================================================================
-- CORE TABLES
-- ============================================================================

-- Users table: authentication, profiles, and subscriptions
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

COMMENT ON TABLE users IS 'User accounts with authentication and subscription information';
COMMENT ON COLUMN users.subscription_tier IS 'Subscription level: free, basic, professional, or enterprise';
COMMENT ON COLUMN users.preferences IS 'User preferences stored as JSON (theme, notifications, default weights, etc.)';

-- Companies table: stock information and metadata
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
CREATE INDEX idx_companies_active ON companies(is_active) WHERE is_active = true;

COMMENT ON TABLE companies IS 'Company master data for all tracked stocks';
COMMENT ON COLUMN companies.sector IS 'GICS sector classification for relative scoring';

-- ============================================================================
-- IC SCORE SYSTEM
-- ============================================================================

-- IC Scores table: our proprietary scoring system
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
    rating VARCHAR(20), -- 'Strong Buy', 'Buy', 'Hold', 'Underperform', 'Sell'
    sector_percentile DECIMAL(5,2),
    confidence_level VARCHAR(20), -- 'High', 'Medium', 'Low'
    data_completeness DECIMAL(5,2), -- % of factors with data
    calculation_metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(ticker, date)
);

CREATE INDEX idx_ic_scores_ticker ON ic_scores(ticker);
CREATE INDEX idx_ic_scores_date ON ic_scores(date);
CREATE INDEX idx_ic_scores_overall ON ic_scores(overall_score DESC);
CREATE INDEX idx_ic_scores_ticker_date ON ic_scores(ticker, date DESC);
CREATE INDEX idx_ic_scores_rating ON ic_scores(rating);

COMMENT ON TABLE ic_scores IS 'InvestorCenter proprietary 10-factor stock scores (1-100)';
COMMENT ON COLUMN ic_scores.overall_score IS 'Weighted average of all 10 factors (1-100 scale)';
COMMENT ON COLUMN ic_scores.calculation_metadata IS 'JSON with factor weights, data sources, and calculation timestamp';

-- ============================================================================
-- FINANCIAL DATA
-- ============================================================================

-- Financials table: income statement, balance sheet, cash flow
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
CREATE INDEX idx_financials_fiscal_year ON financials(fiscal_year DESC);

COMMENT ON TABLE financials IS 'Quarterly and annual financial statements from SEC filings';
COMMENT ON COLUMN financials.raw_data IS 'Full SEC filing data in JSON format for custom analysis';

-- ============================================================================
-- INSIDER & INSTITUTIONAL DATA
-- ============================================================================

-- Insider Trades table: Form 4 data from SEC
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
CREATE INDEX idx_insider_type ON insider_trades(transaction_type);

COMMENT ON TABLE insider_trades IS 'Insider trading activity from SEC Form 4 filings';
COMMENT ON COLUMN insider_trades.is_derivative IS 'True for options/warrants, false for direct stock purchases';

-- Institutional Holdings table: Form 13F data
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
CREATE INDEX idx_institutional_institution ON institutional_holdings(institution_cik);

COMMENT ON TABLE institutional_holdings IS 'Institutional ownership data from SEC Form 13F filings';

-- ============================================================================
-- ANALYST & NEWS DATA
-- ============================================================================

-- Analyst Ratings table: Wall Street analyst data
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
CREATE INDEX idx_analyst_firm ON analyst_ratings(analyst_firm);

COMMENT ON TABLE analyst_ratings IS 'Wall Street analyst ratings and price targets';

-- News Articles table: news with sentiment scores
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
CREATE INDEX idx_news_source ON news_articles(source);

COMMENT ON TABLE news_articles IS 'News articles with AI-powered sentiment analysis';
COMMENT ON COLUMN news_articles.sentiment_score IS 'Sentiment from -100 (very negative) to +100 (very positive)';

-- ============================================================================
-- WATCHLISTS & PORTFOLIOS
-- ============================================================================

-- Watchlists table: user watchlists
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

COMMENT ON TABLE watchlists IS 'User-created stock watchlists';

-- Watchlist Stocks table: stocks in watchlists
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
CREATE INDEX idx_watchlist_stocks_ticker ON watchlist_stocks(ticker);

COMMENT ON TABLE watchlist_stocks IS 'Stocks added to user watchlists';

-- Portfolios table: user portfolios
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

COMMENT ON TABLE portfolios IS 'User investment portfolios';

-- Portfolio Positions table: current holdings
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
CREATE INDEX idx_positions_ticker ON portfolio_positions(ticker);

COMMENT ON TABLE portfolio_positions IS 'Current stock positions in portfolios';

-- Portfolio Transactions table: transaction history
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
CREATE INDEX idx_transactions_ticker ON portfolio_transactions(ticker);
CREATE INDEX idx_transactions_date ON portfolio_transactions(transaction_date DESC);
CREATE INDEX idx_transactions_type ON portfolio_transactions(transaction_type);

COMMENT ON TABLE portfolio_transactions IS 'Transaction history for portfolio tracking';

-- ============================================================================
-- ALERTS
-- ============================================================================

-- Alerts table: user alerts
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
CREATE INDEX idx_alerts_type ON alerts(alert_type);

COMMENT ON TABLE alerts IS 'User-configured alerts for price, score, and event notifications';
COMMENT ON COLUMN alerts.condition IS 'JSON with alert conditions (e.g., {"threshold": 150, "direction": "above"})';

-- ============================================================================
-- TIME-SERIES DATA (TimescaleDB Hypertables)
-- ============================================================================

-- Stock Prices table: OHLCV price data
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

-- Convert to hypertable (must be done after table creation)
SELECT create_hypertable('stock_prices', 'time');

-- Create indexes
CREATE INDEX idx_prices_ticker_time ON stock_prices(ticker, time DESC);
CREATE INDEX idx_prices_time ON stock_prices(time DESC);
CREATE INDEX idx_prices_interval ON stock_prices(interval);

COMMENT ON TABLE stock_prices IS 'TimescaleDB hypertable for efficient time-series price storage';

-- Continuous aggregate for daily prices from minute data
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

-- Technical Indicators table: calculated indicators
CREATE TABLE technical_indicators (
    time TIMESTAMPTZ NOT NULL,
    ticker VARCHAR(10) NOT NULL,
    indicator_name VARCHAR(50) NOT NULL,
    value DECIMAL(18,6),
    metadata JSONB
);

-- Convert to hypertable
SELECT create_hypertable('technical_indicators', 'time');

-- Create indexes
CREATE INDEX idx_indicators_ticker_time ON technical_indicators(ticker, time DESC);
CREATE INDEX idx_indicators_name ON technical_indicators(indicator_name);
CREATE INDEX idx_indicators_ticker_name ON technical_indicators(ticker, indicator_name);

COMMENT ON TABLE technical_indicators IS 'TimescaleDB hypertable for technical indicators (RSI, MACD, etc.)';

-- ============================================================================
-- TRIGGERS FOR AUTOMATIC TIMESTAMP UPDATES
-- ============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to tables with updated_at column
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_watchlists_updated_at BEFORE UPDATE ON watchlists
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_portfolios_updated_at BEFORE UPDATE ON portfolios
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- VIEWS FOR COMMON QUERIES
-- ============================================================================

-- View: Latest IC Score for each ticker
CREATE VIEW latest_ic_scores AS
SELECT DISTINCT ON (ticker)
    ticker,
    date,
    overall_score,
    rating,
    value_score,
    growth_score,
    profitability_score,
    financial_health_score,
    momentum_score,
    analyst_consensus_score,
    insider_activity_score,
    institutional_score,
    news_sentiment_score,
    technical_score,
    sector_percentile,
    confidence_level,
    data_completeness
FROM ic_scores
ORDER BY ticker, date DESC;

COMMENT ON VIEW latest_ic_scores IS 'Most recent IC Score for each stock';

-- View: Latest financials for each ticker
CREATE VIEW latest_financials AS
SELECT DISTINCT ON (ticker)
    ticker,
    period_end_date,
    fiscal_year,
    fiscal_quarter,
    revenue,
    net_income,
    eps_diluted,
    pe_ratio,
    pb_ratio,
    debt_to_equity,
    roe,
    free_cash_flow
FROM financials
ORDER BY ticker, period_end_date DESC;

COMMENT ON VIEW latest_financials IS 'Most recent financial statements for each stock';

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================
