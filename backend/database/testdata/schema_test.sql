-- Minimal test schema for integration tests.
-- Subset of production tables, derived from backend/migrations/.
-- Uses regular tables (no materialized views, no TimescaleDB).

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- tickers table (core stock lookup)
CREATE TABLE IF NOT EXISTS tickers (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL,
    name VARCHAR(255) NOT NULL,
    exchange VARCHAR(50),
    sector VARCHAR(100),
    industry VARCHAR(100),
    country VARCHAR(50) DEFAULT 'US',
    currency VARCHAR(3) DEFAULT 'USD',
    market_cap DECIMAL(20,2),
    description TEXT,
    website VARCHAR(255),
    asset_type VARCHAR(20) DEFAULT 'stock',
    cik VARCHAR(10),
    logo_url VARCHAR(255),
    active BOOLEAN DEFAULT TRUE,
    locale VARCHAR(10),
    market VARCHAR(20),
    currency_name VARCHAR(50),
    composite_figi VARCHAR(20),
    share_class_figi VARCHAR(20),
    primary_exchange_code VARCHAR(20),
    polygon_type VARCHAR(20),
    last_updated_utc VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(symbol)
);

-- users table (auth critical path)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    full_name VARCHAR(255),
    timezone VARCHAR(50) DEFAULT 'UTC',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP,
    email_verified BOOLEAN DEFAULT FALSE,
    email_verification_token VARCHAR(255),
    email_verification_expires_at TIMESTAMP,
    password_reset_token VARCHAR(255),
    password_reset_expires_at TIMESTAMP,
    is_premium BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    is_admin BOOLEAN DEFAULT FALSE,
    is_worker BOOLEAN DEFAULT FALSE,
    last_activity_at TIMESTAMP
);

-- watch_lists table (FK relationships, JOINs)
CREATE TABLE IF NOT EXISTS watch_lists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    display_order INTEGER,
    is_public BOOLEAN DEFAULT FALSE,
    public_slug VARCHAR(100) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- watch_list_items table (unique constraints, FK validation)
CREATE TABLE IF NOT EXISTS watch_list_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    watch_list_id UUID REFERENCES watch_lists(id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL,
    notes TEXT,
    tags TEXT[],
    target_buy_price DECIMAL(20, 4),
    target_sell_price DECIMAL(20, 4),
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    display_order INTEGER,
    UNIQUE(watch_list_id, symbol)
);

-- screener_data (regular table; prod uses materialized view)
CREATE TABLE IF NOT EXISTS screener_data (
    symbol VARCHAR(10) NOT NULL,
    name VARCHAR(255),
    sector VARCHAR(100),
    industry VARCHAR(100),
    market_cap FLOAT8,
    price FLOAT8,
    pe_ratio FLOAT8,
    pb_ratio FLOAT8,
    ps_ratio FLOAT8,
    roe FLOAT8,
    roa FLOAT8,
    gross_margin FLOAT8,
    operating_margin FLOAT8,
    net_margin FLOAT8,
    debt_to_equity FLOAT8,
    current_ratio FLOAT8,
    revenue_growth FLOAT8,
    eps_growth_yoy FLOAT8,
    dividend_yield FLOAT8,
    payout_ratio FLOAT8,
    consecutive_dividend_years INT,
    beta FLOAT8,
    dcf_upside_percent FLOAT8,
    ic_score FLOAT8,
    ic_rating VARCHAR(20),
    value_score FLOAT8,
    growth_score FLOAT8,
    profitability_score FLOAT8,
    financial_health_score FLOAT8,
    momentum_score FLOAT8,
    analyst_consensus_score FLOAT8,
    insider_activity_score FLOAT8,
    institutional_score FLOAT8,
    news_sentiment_score FLOAT8,
    technical_score FLOAT8,
    ic_sector_percentile FLOAT8,
    lifecycle_stage VARCHAR(20)
);
