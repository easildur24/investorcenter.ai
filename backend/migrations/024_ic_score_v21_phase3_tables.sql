-- IC Score v2.1 Phase 3 Tables: Enhanced Features
-- This migration adds tables for:
-- 1. IC Score Events (for score stabilization/reset triggers)
-- 2. Stock Peers (for peer comparison)
-- 3. Catalyst Events (upcoming market events)
-- 4. IC Score Changes (score change tracking and explanations)

-- ====================
-- IC Score Events Table
-- ====================
-- Stores events that may trigger score recalculation or bypass smoothing
CREATE TABLE IF NOT EXISTS ic_score_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    event_date DATE NOT NULL,
    description TEXT,
    impact_direction VARCHAR(10),  -- 'positive', 'negative', 'neutral'
    impact_magnitude NUMERIC(5,2),  -- Estimated impact on score
    source VARCHAR(50),            -- Data source: 'earnings', 'analyst', 'insider', etc.
    metadata JSONB,                -- Additional event-specific data

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (ticker, event_type, event_date)
);

CREATE INDEX IF NOT EXISTS idx_ic_score_events_ticker ON ic_score_events(ticker);
CREATE INDEX IF NOT EXISTS idx_ic_score_events_date ON ic_score_events(event_date DESC);
CREATE INDEX IF NOT EXISTS idx_ic_score_events_ticker_date ON ic_score_events(ticker, event_date DESC);
CREATE INDEX IF NOT EXISTS idx_ic_score_events_type ON ic_score_events(event_type);

-- ====================
-- Stock Peers Table
-- ====================
-- Stores calculated peer relationships for comparison
CREATE TABLE IF NOT EXISTS stock_peers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    peer_ticker VARCHAR(10) NOT NULL,

    -- Similarity metrics
    similarity_score NUMERIC(5,4) NOT NULL,  -- 0-1 scale
    similarity_factors JSONB,                 -- Breakdown of similarity components

    calculated_at DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (ticker, peer_ticker, calculated_at)
);

CREATE INDEX IF NOT EXISTS idx_stock_peers_ticker ON stock_peers(ticker);
CREATE INDEX IF NOT EXISTS idx_stock_peers_ticker_date ON stock_peers(ticker, calculated_at DESC);
CREATE INDEX IF NOT EXISTS idx_stock_peers_similarity ON stock_peers(ticker, similarity_score DESC);

-- ====================
-- Catalyst Events Table
-- ====================
-- Stores upcoming catalysts/events for stocks
CREATE TABLE IF NOT EXISTS catalyst_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    event_type VARCHAR(50) NOT NULL,  -- 'earnings', 'ex_dividend', 'analyst_day', 'technical', etc.
    title VARCHAR(200) NOT NULL,
    event_date DATE,                  -- NULL if date unknown
    icon VARCHAR(10),                 -- Emoji for UI display
    impact VARCHAR(20),               -- 'Positive', 'Negative', 'Neutral', 'Unknown'
    confidence NUMERIC(3,2),          -- 0-1 confidence in event
    days_until INTEGER,               -- Days until event (can be negative for past)

    metadata JSONB,                   -- Additional event-specific data

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at DATE,                  -- When to remove stale catalyst

    UNIQUE (ticker, event_type, event_date)
);

CREATE INDEX IF NOT EXISTS idx_catalyst_events_ticker ON catalyst_events(ticker);
CREATE INDEX IF NOT EXISTS idx_catalyst_events_date ON catalyst_events(event_date);
CREATE INDEX IF NOT EXISTS idx_catalyst_events_ticker_date ON catalyst_events(ticker, event_date);
CREATE INDEX IF NOT EXISTS idx_catalyst_events_upcoming ON catalyst_events(days_until) WHERE days_until >= 0;

-- ====================
-- IC Score Changes Table
-- ====================
-- Tracks score changes with factor-level breakdown
CREATE TABLE IF NOT EXISTS ic_score_changes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker VARCHAR(10) NOT NULL,
    calculated_at DATE NOT NULL DEFAULT CURRENT_DATE,

    -- Score values
    previous_score NUMERIC(5,2),
    current_score NUMERIC(5,2) NOT NULL,
    delta NUMERIC(5,2),

    -- Factor-level changes
    factor_changes JSONB,     -- Array of {factor, delta, contribution, explanation}

    -- Event triggers
    trigger_events JSONB,     -- Array of event types that triggered recalculation
    smoothing_applied BOOLEAN DEFAULT false,

    -- Explanation
    summary TEXT,             -- Human-readable summary

    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (ticker, calculated_at)
);

CREATE INDEX IF NOT EXISTS idx_ic_score_changes_ticker ON ic_score_changes(ticker);
CREATE INDEX IF NOT EXISTS idx_ic_score_changes_date ON ic_score_changes(calculated_at DESC);
CREATE INDEX IF NOT EXISTS idx_ic_score_changes_ticker_date ON ic_score_changes(ticker, calculated_at DESC);
CREATE INDEX IF NOT EXISTS idx_ic_score_changes_significant ON ic_score_changes(ticker, ABS(delta) DESC);

-- ====================
-- Add Phase 3 columns to ic_scores table
-- ====================
DO $$
BEGIN
    -- Add previous_score column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'ic_scores' AND column_name = 'previous_score') THEN
        ALTER TABLE ic_scores ADD COLUMN previous_score NUMERIC(5,2);
    END IF;

    -- Add raw_score column (score before stabilization)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'ic_scores' AND column_name = 'raw_score') THEN
        ALTER TABLE ic_scores ADD COLUMN raw_score NUMERIC(5,2);
    END IF;

    -- Add smoothing_applied flag
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'ic_scores' AND column_name = 'smoothing_applied') THEN
        ALTER TABLE ic_scores ADD COLUMN smoothing_applied BOOLEAN DEFAULT false;
    END IF;

    -- Add peer comparison fields
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'ic_scores' AND column_name = 'peer_avg_score') THEN
        ALTER TABLE ic_scores ADD COLUMN peer_avg_score NUMERIC(5,2);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'ic_scores' AND column_name = 'vs_peers_delta') THEN
        ALTER TABLE ic_scores ADD COLUMN vs_peers_delta NUMERIC(5,2);
    END IF;
END $$;

-- ====================
-- Score Stability Settings Table (optional, for tuning)
-- ====================
CREATE TABLE IF NOT EXISTS ic_score_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    setting_key VARCHAR(50) UNIQUE NOT NULL,
    setting_value JSONB NOT NULL,
    description TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default settings
INSERT INTO ic_score_settings (setting_key, setting_value, description)
VALUES
    ('stabilization_alpha', '{"value": 0.7}', 'Exponential smoothing factor (0-1). Higher = more responsive to changes'),
    ('min_change_threshold', '{"value": 0.5}', 'Minimum score change to update (prevents micro-fluctuations)'),
    ('reset_event_types', '{"types": ["earnings_release", "analyst_rating_change", "insider_trade_large", "dividend_announcement", "acquisition_news", "guidance_update"]}', 'Event types that bypass smoothing'),
    ('peer_market_cap_ratio', '{"min": 0.25, "max": 4.0}', 'Market cap range for peer selection (0.25x to 4x)'),
    ('catalyst_lookback_days', '{"value": 30}', 'Days to look back for catalyst detection'),
    ('catalyst_lookahead_days', '{"value": 60}', 'Days to look ahead for upcoming catalysts')
ON CONFLICT (setting_key) DO UPDATE SET
    setting_value = EXCLUDED.setting_value,
    description = EXCLUDED.description,
    updated_at = NOW();

-- ====================
-- Views for Common Queries
-- ====================

-- View: Stocks with significant recent changes
CREATE OR REPLACE VIEW v_significant_score_changes AS
SELECT
    c.ticker,
    c.calculated_at,
    c.previous_score,
    c.current_score,
    c.delta,
    c.summary,
    c.trigger_events,
    t.company_name,
    t.sector
FROM ic_score_changes c
JOIN tickers t ON c.ticker = t.symbol
WHERE ABS(c.delta) >= 3
  AND c.calculated_at >= CURRENT_DATE - INTERVAL '7 days'
ORDER BY ABS(c.delta) DESC, c.calculated_at DESC;

-- View: Upcoming catalysts for all stocks
CREATE OR REPLACE VIEW v_upcoming_catalysts AS
SELECT
    ce.ticker,
    ce.event_type,
    ce.title,
    ce.event_date,
    ce.days_until,
    ce.impact,
    ce.confidence,
    t.company_name,
    t.sector,
    i.overall_score as ic_score
FROM catalyst_events ce
JOIN tickers t ON ce.ticker = t.symbol
LEFT JOIN ic_scores i ON ce.ticker = i.ticker AND i.date = CURRENT_DATE
WHERE ce.days_until >= 0
  AND ce.days_until <= 30
ORDER BY ce.days_until ASC, ce.confidence DESC;

-- View: Peer comparison summary
CREATE OR REPLACE VIEW v_peer_comparison_summary AS
SELECT
    sp.ticker,
    t.company_name,
    t.sector,
    i.overall_score as ic_score,
    AVG(pi.overall_score) as avg_peer_score,
    i.overall_score - AVG(pi.overall_score) as vs_peers_delta,
    COUNT(*) as peer_count
FROM stock_peers sp
JOIN tickers t ON sp.ticker = t.symbol
JOIN ic_scores i ON sp.ticker = i.ticker AND i.date = CURRENT_DATE
JOIN ic_scores pi ON sp.peer_ticker = pi.ticker AND pi.date = CURRENT_DATE
WHERE sp.calculated_at = CURRENT_DATE
GROUP BY sp.ticker, t.company_name, t.sector, i.overall_score;

-- ====================
-- Cronjob Definitions for Phase 3 Pipelines
-- ====================
INSERT INTO cronjob_definitions (name, schedule, description, pipeline_name, enabled)
VALUES
    ('ic_score_events_detection', '0 5 * * *', 'Daily event detection for score stabilization', 'ic_score_events_detection', true),
    ('stock_peers_calculation', '0 6 * * 0', 'Weekly peer relationship calculation', 'stock_peers_calculation', true),
    ('catalyst_events_detection', '0 5 * * *', 'Daily catalyst detection', 'catalyst_events_detection', true),
    ('ic_score_changes_tracking', '0 8 * * *', 'Daily score change tracking (after IC Score calculation)', 'ic_score_changes_tracking', true)
ON CONFLICT (name) DO UPDATE SET
    schedule = EXCLUDED.schedule,
    description = EXCLUDED.description,
    pipeline_name = EXCLUDED.pipeline_name;
