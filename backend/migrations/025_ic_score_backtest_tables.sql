-- IC Score Backtest & Feature Flag Tables
-- Migration: 025_ic_score_backtest_tables.sql

-- Backtest Jobs table for tracking async backtest runs
CREATE TABLE IF NOT EXISTS backtest_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,

    -- Configuration (stored as JSON)
    config JSONB NOT NULL,

    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error TEXT,

    -- Timestamps
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Results (stored as JSON when completed)
    result JSONB,

    CONSTRAINT valid_status CHECK (status IN ('pending', 'running', 'completed', 'failed'))
);

-- Indexes for backtest_jobs
CREATE INDEX IF NOT EXISTS idx_backtest_jobs_user_id ON backtest_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_backtest_jobs_status ON backtest_jobs(status);
CREATE INDEX IF NOT EXISTS idx_backtest_jobs_created_at ON backtest_jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_backtest_jobs_config ON backtest_jobs USING GIN (config);

-- Feature flags table for gradual rollout
CREATE TABLE IF NOT EXISTS feature_flags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    percentage NUMERIC(5,2) NOT NULL DEFAULT 0,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for feature flags lookup
CREATE INDEX IF NOT EXISTS idx_feature_flags_name ON feature_flags(name);

-- Insert default IC Score v2.1 feature flags
INSERT INTO feature_flags (name, enabled, percentage, description) VALUES
    ('ic_score_sector_relative_scoring', false, 0, 'Enable sector-relative scoring in IC Score calculation'),
    ('ic_score_lifecycle_classification', false, 0, 'Enable lifecycle stage classification for weight adjustments'),
    ('ic_score_earnings_revisions_factor', false, 0, 'Include earnings revisions in IC Score calculation'),
    ('ic_score_historical_valuation_factor', false, 0, 'Include historical valuation factor in IC Score'),
    ('ic_score_dividend_quality_factor', false, 0, 'Include optional dividend quality factor'),
    ('ic_score_score_stability', false, 0, 'Apply score smoothing to reduce daily volatility'),
    ('ic_score_peer_comparison', false, 0, 'Show peer comparison in IC Score display'),
    ('ic_score_catalysts', false, 0, 'Show upcoming catalysts in IC Score display'),
    ('ic_score_score_change_explanations', false, 0, 'Show explanations for score changes'),
    ('ic_score_granular_confidence', false, 0, 'Show per-factor data availability in confidence'),
    ('ic_score_backtest_dashboard', false, 0, 'Enable backtest results dashboard')
ON CONFLICT (name) DO NOTHING;

-- Backtest results cache table (for storing computed backtest summaries)
CREATE TABLE IF NOT EXISTS backtest_results_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_hash VARCHAR(64) NOT NULL,  -- SHA256 hash of config for lookup

    -- Summary data
    summary JSONB NOT NULL,

    -- Detailed data (optional, for full reports)
    detailed_report JSONB,

    -- Timestamps
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '7 days',

    UNIQUE(config_hash)
);

-- Index for cache lookup
CREATE INDEX IF NOT EXISTS idx_backtest_cache_hash ON backtest_results_cache(config_hash);
CREATE INDEX IF NOT EXISTS idx_backtest_cache_expires ON backtest_results_cache(expires_at);

-- Function to clean up expired cache entries
CREATE OR REPLACE FUNCTION cleanup_backtest_cache()
RETURNS void AS $$
BEGIN
    DELETE FROM backtest_results_cache WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to backtest_jobs
DROP TRIGGER IF EXISTS update_backtest_jobs_updated_at ON backtest_jobs;
CREATE TRIGGER update_backtest_jobs_updated_at
    BEFORE UPDATE ON backtest_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Apply trigger to feature_flags
DROP TRIGGER IF EXISTS update_feature_flags_updated_at ON feature_flags;
CREATE TRIGGER update_feature_flags_updated_at
    BEFORE UPDATE ON feature_flags
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
