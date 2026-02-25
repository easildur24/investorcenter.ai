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
    volume BIGINT,
    avg_volume_30d BIGINT,
    avg_volume_90d BIGINT,
    vwap DECIMAL(15,4),
    current_price DECIMAL(15,2),
    day_open DECIMAL(15,2),
    day_high DECIMAL(15,2),
    day_low DECIMAL(15,2),
    previous_close DECIMAL(15,2),
    week_52_high DECIMAL(15,2),
    week_52_low DECIMAL(15,2),
    last_trade_timestamp TIMESTAMP WITH TIME ZONE,
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
-- Used by GetWatchListItemsWithData LEFT JOIN for IC Score, fundamentals, etc.
-- NOTE: prod screener_data is a materialized view refreshed daily at 23:45 UTC.
-- The UNIQUE constraint on symbol exists here for test ON CONFLICT usage, but
-- prod relies on the view definition to guarantee one row per symbol. If the
-- refresh query ever produces duplicates, the LEFT JOIN could multiply rows.
CREATE TABLE IF NOT EXISTS screener_data (
    symbol VARCHAR(10) NOT NULL UNIQUE,
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

-- =====================
-- Phase 4.1 additions
-- =====================

-- financial_statements (Batch 1: financials)
CREATE TABLE IF NOT EXISTS financial_statements (
    id SERIAL PRIMARY KEY,
    ticker_id INTEGER NOT NULL REFERENCES tickers(id),
    cik VARCHAR(10),
    statement_type VARCHAR(20) NOT NULL,
    timeframe VARCHAR(30) NOT NULL,
    fiscal_year INTEGER NOT NULL,
    fiscal_quarter INTEGER,
    period_start DATE,
    period_end DATE NOT NULL,
    filed_date DATE,
    source_filing_url TEXT,
    source_filing_type VARCHAR(20),
    data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(ticker_id, statement_type, timeframe, fiscal_year, fiscal_quarter)
);

-- eps_estimates (Batch 1: IC Score Phase 2)
CREATE TABLE IF NOT EXISTS eps_estimates (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    fiscal_year INTEGER NOT NULL,
    fiscal_quarter INTEGER,
    consensus_eps NUMERIC,
    num_analysts INTEGER,
    high_estimate NUMERIC,
    low_estimate NUMERIC,
    estimate_30d_ago NUMERIC,
    estimate_60d_ago NUMERIC,
    estimate_90d_ago NUMERIC,
    upgrades_30d INTEGER DEFAULT 0,
    downgrades_30d INTEGER DEFAULT 0,
    upgrades_60d INTEGER DEFAULT 0,
    downgrades_60d INTEGER DEFAULT 0,
    upgrades_90d INTEGER DEFAULT 0,
    downgrades_90d INTEGER DEFAULT 0,
    revision_pct_30d NUMERIC,
    revision_pct_60d NUMERIC,
    revision_pct_90d NUMERIC,
    fetched_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(ticker, fiscal_year, fiscal_quarter)
);

-- valuation_ratios (Batch 1: financials/IC Score)
CREATE TABLE IF NOT EXISTS valuation_ratios (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    calculation_date DATE NOT NULL,
    stock_price NUMERIC,
    ttm_pe_ratio NUMERIC,
    ttm_pb_ratio NUMERIC,
    ttm_ps_ratio NUMERIC,
    ttm_market_cap BIGINT,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(ticker, calculation_date)
);

-- fundamental_metrics_extended (Batch 1: financials/IC Score)
CREATE TABLE IF NOT EXISTS fundamental_metrics_extended (
    id SERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    calculation_date DATE NOT NULL,
    gross_margin NUMERIC,
    operating_margin NUMERIC,
    net_margin NUMERIC,
    ebitda_margin NUMERIC,
    roe NUMERIC,
    roa NUMERIC,
    roic NUMERIC,
    current_ratio NUMERIC,
    quick_ratio NUMERIC,
    debt_to_equity NUMERIC,
    interest_coverage NUMERIC,
    enterprise_value NUMERIC,
    ev_to_revenue NUMERIC,
    ev_to_ebitda NUMERIC,
    revenue_growth_yoy NUMERIC,
    eps_growth_yoy NUMERIC,
    dividend_yield NUMERIC,
    payout_ratio NUMERIC,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(ticker, calculation_date)
);

-- mv_latest_sector_percentiles (regular table in tests; materialized view in prod)
CREATE TABLE IF NOT EXISTS mv_latest_sector_percentiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sector VARCHAR(50) NOT NULL,
    metric_name VARCHAR(50) NOT NULL,
    calculated_at DATE,
    min_value NUMERIC(20,4),
    p10_value NUMERIC(20,4),
    p25_value NUMERIC(20,4),
    p50_value NUMERIC(20,4),
    p75_value NUMERIC(20,4),
    p90_value NUMERIC(20,4),
    max_value NUMERIC(20,4),
    mean_value NUMERIC(20,4),
    std_dev NUMERIC(20,4),
    sample_count INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);

-- alert_rules (Batch 2: alerts)
CREATE TABLE IF NOT EXISTS alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    watch_list_id UUID REFERENCES watch_lists(id) ON DELETE CASCADE,
    watch_list_item_id UUID REFERENCES watch_list_items(id) ON DELETE SET NULL,
    symbol VARCHAR(20),
    alert_type VARCHAR(50) NOT NULL,
    conditions JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    frequency VARCHAR(20) DEFAULT 'once',
    notify_email BOOLEAN DEFAULT TRUE,
    notify_in_app BOOLEAN DEFAULT TRUE,
    name VARCHAR(255),
    description TEXT,
    last_triggered_at TIMESTAMP WITH TIME ZONE,
    trigger_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- alert_logs (Batch 2: alerts)
CREATE TABLE IF NOT EXISTS alert_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_rule_id UUID REFERENCES alert_rules(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symbol VARCHAR(20),
    triggered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    alert_type VARCHAR(50),
    condition_met JSONB,
    market_data JSONB,
    notification_sent BOOLEAN DEFAULT FALSE,
    notification_sent_at TIMESTAMP WITH TIME ZONE,
    notification_error TEXT,
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP WITH TIME ZONE,
    is_dismissed BOOLEAN DEFAULT FALSE,
    dismissed_at TIMESTAMP WITH TIME ZONE
);

-- sessions (Batch 2: session management)
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    user_agent TEXT,
    ip_address VARCHAR(45)
);

-- password_reset_tokens (Batch 2: password reset)
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- notification_preferences (Batch 2: notifications)
CREATE TABLE IF NOT EXISTS notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    email_enabled BOOLEAN DEFAULT TRUE,
    email_address VARCHAR(255),
    email_verified BOOLEAN DEFAULT FALSE,
    price_alerts_enabled BOOLEAN DEFAULT TRUE,
    volume_alerts_enabled BOOLEAN DEFAULT TRUE,
    news_alerts_enabled BOOLEAN DEFAULT TRUE,
    earnings_alerts_enabled BOOLEAN DEFAULT TRUE,
    sec_filing_alerts_enabled BOOLEAN DEFAULT TRUE,
    daily_digest_enabled BOOLEAN DEFAULT FALSE,
    daily_digest_time TIME DEFAULT '09:00:00',
    weekly_digest_enabled BOOLEAN DEFAULT FALSE,
    weekly_digest_day INTEGER DEFAULT 1,
    weekly_digest_time TIME DEFAULT '09:00:00',
    digest_include_portfolio_summary BOOLEAN DEFAULT TRUE,
    digest_include_top_movers BOOLEAN DEFAULT TRUE,
    digest_include_recent_alerts BOOLEAN DEFAULT TRUE,
    digest_include_news_highlights BOOLEAN DEFAULT TRUE,
    quiet_hours_enabled BOOLEAN DEFAULT FALSE,
    quiet_hours_start TIME DEFAULT '22:00:00',
    quiet_hours_end TIME DEFAULT '08:00:00',
    quiet_hours_timezone VARCHAR(50) DEFAULT 'America/Los_Angeles',
    max_alerts_per_day INTEGER DEFAULT 50,
    max_emails_per_day INTEGER DEFAULT 20,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- notification_queue (Batch 2: in-app notifications)
CREATE TABLE IF NOT EXISTS notification_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    alert_log_id UUID REFERENCES alert_logs(id) ON DELETE SET NULL,
    type VARCHAR(50),
    title VARCHAR(255),
    message TEXT,
    metadata JSONB,
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP WITH TIME ZONE,
    is_dismissed BOOLEAN DEFAULT FALSE,
    dismissed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT (CURRENT_TIMESTAMP + INTERVAL '30 days')
);

-- sentiment_lexicon (Batch 3: social/sentiment)
CREATE TABLE IF NOT EXISTS sentiment_lexicon (
    id SERIAL PRIMARY KEY,
    term VARCHAR(100) UNIQUE NOT NULL,
    sentiment VARCHAR(10),
    weight DECIMAL(3,2) DEFAULT 1.00,
    category VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- reddit_posts_raw (Batch 3: raw Reddit posts from Arctic Shift API)
CREATE TABLE IF NOT EXISTS reddit_posts_raw (
    id BIGSERIAL PRIMARY KEY,
    external_id VARCHAR(20) UNIQUE NOT NULL,
    subreddit VARCHAR(50) NOT NULL,
    author VARCHAR(50),
    title TEXT NOT NULL,
    body TEXT,
    url TEXT NOT NULL,
    upvotes INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    award_count INTEGER DEFAULT 0,
    flair VARCHAR(200),
    posted_at TIMESTAMPTZ NOT NULL,
    fetched_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    processing_skipped BOOLEAN DEFAULT FALSE,
    llm_response JSONB,
    llm_model VARCHAR(50),
    is_finance_related BOOLEAN,
    spam_score DECIMAL(3,2),
    quality_score DECIMAL(3,2),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- reddit_post_tickers (Batch 3: AI-extracted ticker mentions)
CREATE TABLE IF NOT EXISTS reddit_post_tickers (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT REFERENCES reddit_posts_raw(id) ON DELETE CASCADE,
    ticker VARCHAR(10) NOT NULL,
    sentiment VARCHAR(10),
    confidence DECIMAL(3,2),
    is_primary BOOLEAN DEFAULT FALSE,
    mention_type VARCHAR(20),
    extracted_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(post_id, ticker)
);

-- reddit_ticker_rankings (Batch 3: hourly reddit ranking snapshots)
-- Required by GetWatchListItemsWithData LATERAL JOIN for rank_24h_ago data
CREATE TABLE IF NOT EXISTS reddit_ticker_rankings (
    id BIGSERIAL PRIMARY KEY,
    ticker_symbol VARCHAR(10),
    rank INT NOT NULL,
    mentions INT NOT NULL,
    upvotes INT DEFAULT 0,
    rank_24h_ago INT,
    mentions_24h_ago INT,
    snapshot_date DATE NOT NULL,
    snapshot_time TIMESTAMP NOT NULL,
    data_source VARCHAR(20) DEFAULT 'apewisdom',
    created_at TIMESTAMP DEFAULT NOW()
);

-- reddit_heatmap_daily (Batch 3: reddit heatmap)
CREATE TABLE IF NOT EXISTS reddit_heatmap_daily (
    id SERIAL PRIMARY KEY,
    ticker_symbol VARCHAR(10),
    date DATE,
    avg_rank DECIMAL(5,2),
    min_rank INT,
    max_rank INT,
    total_mentions INT,
    total_upvotes INT,
    rank_volatility DECIMAL(5,2),
    trend_direction VARCHAR(10),
    popularity_score DECIMAL(8,2),
    data_source VARCHAR(20) DEFAULT 'apewisdom'
);

-- heatmap_configs (Batch 4: admin)
CREATE TABLE IF NOT EXISTS heatmap_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    watch_list_id UUID REFERENCES watch_lists(id) ON DELETE CASCADE,
    name VARCHAR(255),
    size_metric VARCHAR(50) DEFAULT 'market_cap',
    color_metric VARCHAR(50) DEFAULT 'price_change_pct',
    time_period VARCHAR(10) DEFAULT '1D',
    color_scheme VARCHAR(50) DEFAULT 'red_green',
    label_display VARCHAR(50) DEFAULT 'symbol_change',
    layout_type VARCHAR(50) DEFAULT 'treemap',
    filters_json JSONB DEFAULT '{}'::jsonb,
    color_gradient_json JSONB,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- subscription_plans (Batch 4: subscriptions)
CREATE TABLE IF NOT EXISTS subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100),
    description TEXT,
    price_monthly DECIMAL(10,2) DEFAULT 0.00,
    price_yearly DECIMAL(10,2) DEFAULT 0.00,
    max_watch_lists INTEGER DEFAULT 3,
    max_items_per_watch_list INTEGER DEFAULT 10,
    max_alert_rules INTEGER DEFAULT 10,
    max_heatmap_configs INTEGER DEFAULT 3,
    features JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- user_subscriptions (Batch 4: subscriptions)
CREATE TABLE IF NOT EXISTS user_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    status VARCHAR(20) DEFAULT 'active',
    billing_period VARCHAR(20) DEFAULT 'monthly',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    current_period_start TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    current_period_end TIMESTAMP WITH TIME ZONE,
    canceled_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    stripe_subscription_id VARCHAR(255) UNIQUE,
    stripe_customer_id VARCHAR(255),
    payment_method VARCHAR(50),
    last_payment_date TIMESTAMP WITH TIME ZONE,
    next_payment_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
