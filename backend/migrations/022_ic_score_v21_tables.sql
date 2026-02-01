-- Migration: 022_ic_score_v21_tables.sql
-- Description: Add IC Score v2.1 tables and columns for sector-relative scoring
-- Date: 2026-02-01

-- ============================================================================
-- Sector Percentiles Table
-- Stores distribution statistics for financial metrics within each sector
-- ============================================================================

CREATE TABLE IF NOT EXISTS sector_percentiles (
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

CREATE INDEX IF NOT EXISTS idx_sector_percentiles_lookup
ON sector_percentiles(sector, metric_name, calculated_at DESC);

-- Materialized view for quick sector lookups (latest values only)
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_latest_sector_percentiles AS
SELECT DISTINCT ON (sector, metric_name)
    id,
    sector,
    metric_name,
    calculated_at,
    min_value,
    p10_value,
    p25_value,
    p50_value,
    p75_value,
    p90_value,
    max_value,
    mean_value,
    std_dev,
    sample_count
FROM sector_percentiles
ORDER BY sector, metric_name, calculated_at DESC;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_sector_percentiles
ON mv_latest_sector_percentiles(sector, metric_name);

-- ============================================================================
-- Lifecycle Classifications Table
-- Stores company lifecycle stage classifications for weight adjustments
-- ============================================================================

CREATE TABLE IF NOT EXISTS lifecycle_classifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    classified_at DATE NOT NULL DEFAULT CURRENT_DATE,

    -- Classification result
    lifecycle_stage VARCHAR(20) NOT NULL,  -- hypergrowth, growth, mature, value, turnaround

    -- Input metrics used for classification
    revenue_growth_yoy NUMERIC(10,4),
    net_margin NUMERIC(10,4),
    pe_ratio NUMERIC(10,4),
    market_cap BIGINT,

    -- Adjusted weights applied
    weights_applied JSONB,

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (ticker, classified_at)
);

CREATE INDEX IF NOT EXISTS idx_lifecycle_ticker_date
ON lifecycle_classifications(ticker, classified_at DESC);

-- ============================================================================
-- IC Scores Table Updates (v2.1 columns)
-- Add new columns to existing ic_scores table
-- ============================================================================

-- Add lifecycle and sector context columns
ALTER TABLE ic_scores
    ADD COLUMN IF NOT EXISTS lifecycle_stage VARCHAR(20),
    ADD COLUMN IF NOT EXISTS raw_score NUMERIC(5,2),
    ADD COLUMN IF NOT EXISTS smoothing_applied BOOLEAN DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS weights_used JSONB,
    ADD COLUMN IF NOT EXISTS sector_rank INTEGER,
    ADD COLUMN IF NOT EXISTS sector_total INTEGER;

-- Add index for lifecycle-based queries
CREATE INDEX IF NOT EXISTS idx_ic_scores_lifecycle
ON ic_scores(lifecycle_stage, date DESC);

-- ============================================================================
-- Cronjob monitoring for new pipelines
-- ============================================================================

INSERT INTO cronjob_definitions (
    name,
    description,
    schedule,
    is_active,
    expected_duration_minutes,
    alert_on_failure,
    created_at
) VALUES (
    'sector_percentiles_calculator',
    'Calculate sector percentile statistics for IC Score v2.1',
    '0 4 * * *',  -- 4 AM daily
    true,
    30,
    true,
    NOW()
) ON CONFLICT (name) DO NOTHING;

INSERT INTO cronjob_definitions (
    name,
    description,
    schedule,
    is_active,
    expected_duration_minutes,
    alert_on_failure,
    created_at
) VALUES (
    'lifecycle_classifier',
    'Classify company lifecycle stages for IC Score v2.1',
    '0 5 * * *',  -- 5 AM daily (after sector percentiles)
    true,
    15,
    true,
    NOW()
) ON CONFLICT (name) DO NOTHING;
