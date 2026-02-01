-- IC Score v2.1 Phase 2 Tables: New Factors
-- This migration adds tables for:
-- 1. EPS Estimates (Earnings Revisions factor)
-- 2. Valuation History (Historical Valuation factor)
-- 3. Dividend History (Dividend Quality factor)

-- ====================
-- EPS Estimates Table
-- ====================
CREATE TABLE IF NOT EXISTS eps_estimates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    fiscal_year INTEGER NOT NULL,
    fiscal_quarter INTEGER,  -- NULL for annual estimates

    -- Estimate data
    consensus_eps NUMERIC(10,4),
    num_analysts INTEGER,
    high_estimate NUMERIC(10,4),
    low_estimate NUMERIC(10,4),

    -- Historical tracking for revision calculations
    estimate_30d_ago NUMERIC(10,4),
    estimate_60d_ago NUMERIC(10,4),
    estimate_90d_ago NUMERIC(10,4),

    -- Revision counts
    upgrades_30d INTEGER DEFAULT 0,
    downgrades_30d INTEGER DEFAULT 0,
    upgrades_60d INTEGER DEFAULT 0,
    downgrades_60d INTEGER DEFAULT 0,
    upgrades_90d INTEGER DEFAULT 0,
    downgrades_90d INTEGER DEFAULT 0,

    -- Calculated revision metrics
    revision_pct_30d NUMERIC(10,4),
    revision_pct_60d NUMERIC(10,4),
    revision_pct_90d NUMERIC(10,4),

    fetched_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (ticker, fiscal_year, fiscal_quarter)
);

CREATE INDEX IF NOT EXISTS idx_eps_estimates_ticker ON eps_estimates(ticker);
CREATE INDEX IF NOT EXISTS idx_eps_estimates_ticker_year ON eps_estimates(ticker, fiscal_year DESC);

-- EPS Estimate History for tracking daily snapshots
CREATE TABLE IF NOT EXISTS eps_estimate_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    fiscal_year INTEGER NOT NULL,
    fiscal_quarter INTEGER,
    snapshot_date DATE NOT NULL DEFAULT CURRENT_DATE,

    consensus_eps NUMERIC(10,4),
    num_analysts INTEGER,
    high_estimate NUMERIC(10,4),
    low_estimate NUMERIC(10,4),

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (ticker, fiscal_year, fiscal_quarter, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_eps_estimate_history_ticker_date ON eps_estimate_history(ticker, snapshot_date DESC);

-- ====================
-- Valuation History Table
-- ====================
CREATE TABLE IF NOT EXISTS valuation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    snapshot_date DATE NOT NULL,

    -- Valuation metrics
    pe_ratio NUMERIC(10,2),
    ps_ratio NUMERIC(10,2),
    pb_ratio NUMERIC(10,2),
    ev_ebitda NUMERIC(10,2),
    peg_ratio NUMERIC(10,2),

    -- Price context
    stock_price NUMERIC(10,2),
    market_cap NUMERIC(20,2),

    -- Earnings data at snapshot time
    eps_ttm NUMERIC(10,4),
    revenue_ttm NUMERIC(20,2),

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (ticker, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_valuation_history_ticker_date ON valuation_history(ticker, snapshot_date DESC);
CREATE INDEX IF NOT EXISTS idx_valuation_history_date ON valuation_history(snapshot_date);

-- ====================
-- Dividend History Table
-- ====================
CREATE TABLE IF NOT EXISTS dividend_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    fiscal_year INTEGER NOT NULL,

    -- Dividend metrics
    annual_dividend NUMERIC(10,4),
    dividend_yield NUMERIC(8,4),
    payout_ratio NUMERIC(8,4),
    dividend_growth_yoy NUMERIC(8,4),

    -- Ex-dividend dates (for catalyst tracking)
    ex_dividend_date DATE,
    payment_date DATE,

    -- Dividend streak tracking
    consecutive_years_paid INTEGER DEFAULT 0,
    consecutive_years_increased INTEGER DEFAULT 0,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (ticker, fiscal_year)
);

CREATE INDEX IF NOT EXISTS idx_dividend_history_ticker ON dividend_history(ticker, fiscal_year DESC);

-- ====================
-- Add Phase 2 columns to ic_scores table
-- ====================
DO $$
BEGIN
    -- Add earnings_revisions_score column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'ic_scores' AND column_name = 'earnings_revisions_score') THEN
        ALTER TABLE ic_scores ADD COLUMN earnings_revisions_score NUMERIC(5,2);
    END IF;

    -- Add historical_value_score column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'ic_scores' AND column_name = 'historical_value_score') THEN
        ALTER TABLE ic_scores ADD COLUMN historical_value_score NUMERIC(5,2);
    END IF;

    -- Add dividend_quality_score column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'ic_scores' AND column_name = 'dividend_quality_score') THEN
        ALTER TABLE ic_scores ADD COLUMN dividend_quality_score NUMERIC(5,2);
    END IF;
END $$;

-- ====================
-- Cronjob Definitions for Phase 2 Pipelines
-- ====================
INSERT INTO cronjob_definitions (name, schedule, description, pipeline_name, enabled)
VALUES
    ('eps_estimates_ingestion', '0 6 * * *', 'Daily EPS estimates ingestion for Earnings Revisions factor', 'eps_estimates_ingestion', true),
    ('valuation_history_snapshot', '0 7 * * 1', 'Weekly valuation history snapshot for Historical Valuation factor', 'valuation_history_snapshot', true),
    ('dividend_history_update', '0 8 * * 1', 'Weekly dividend history update for Dividend Quality factor', 'dividend_history_update', true)
ON CONFLICT (name) DO UPDATE SET
    schedule = EXCLUDED.schedule,
    description = EXCLUDED.description,
    pipeline_name = EXCLUDED.pipeline_name;
