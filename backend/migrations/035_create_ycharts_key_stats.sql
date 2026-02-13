-- Migration 035: Create ycharts_key_stats table for storing YCharts Key Stats data

CREATE TABLE IF NOT EXISTS ycharts_key_stats (
    id BIGSERIAL PRIMARY KEY,
    ticker VARCHAR(20) NOT NULL,
    collected_at TIMESTAMPTZ NOT NULL,
    source_url TEXT,
    s3_key TEXT NOT NULL,
    s3_bucket VARCHAR(100) NOT NULL,
    data_size BIGINT,
    
    -- Price info (top of page)
    price DECIMAL(20,4),
    price_currency VARCHAR(3),
    price_exchange VARCHAR(20),
    price_timestamp TIMESTAMPTZ,
    
    -- Income Statement (most commonly queried fields)
    revenue_ttm BIGINT,
    revenue_quarterly BIGINT,
    net_income_ttm BIGINT,
    net_income_quarterly BIGINT,
    ebit_ttm BIGINT,
    ebit_quarterly BIGINT,
    ebitda_ttm BIGINT,
    ebitda_quarterly BIGINT,
    revenue_growth_yoy DECIMAL(10,6),
    eps_growth_yoy DECIMAL(10,6),
    ebitda_growth_yoy DECIMAL(10,6),
    eps_diluted_ttm DECIMAL(10,4),
    eps_basic_ttm DECIMAL(10,4),
    shares_outstanding BIGINT,
    
    -- Store full payload as JSONB for flexibility
    -- This allows querying any field without schema changes
    data_json JSONB NOT NULL,
    
    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Unique constraint: one record per ticker per collection time
    UNIQUE(ticker, collected_at)
);

-- Index for querying by ticker (most common query)
CREATE INDEX idx_ycharts_keystats_ticker ON ycharts_key_stats(ticker, collected_at DESC);

-- Index for time-range queries
CREATE INDEX idx_ycharts_keystats_collected ON ycharts_key_stats(collected_at DESC);

-- GIN index for querying JSONB fields
CREATE INDEX idx_ycharts_keystats_json ON ycharts_key_stats USING GIN(data_json);

-- Partial index for latest data per ticker
CREATE INDEX idx_ycharts_keystats_latest ON ycharts_key_stats(ticker, collected_at DESC)
    WHERE collected_at > NOW() - INTERVAL '7 days';

COMMENT ON TABLE ycharts_key_stats IS 'YCharts Key Stats data (100+ financial metrics per ticker)';
COMMENT ON COLUMN ycharts_key_stats.data_json IS 'Full JSON payload with all sections: income_statement, balance_sheet, cash_flow, valuation, performance, etc.';
COMMENT ON COLUMN ycharts_key_stats.s3_key IS 'S3 object key where raw JSON is stored';
