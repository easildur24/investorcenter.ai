-- Create table for storing fundamental metrics calculated from SEC filings
CREATE TABLE IF NOT EXISTS fundamental_metrics (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(10) NOT NULL UNIQUE,
    metrics_data JSONB NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on symbol for fast lookups
CREATE INDEX IF NOT EXISTS idx_fundamental_metrics_symbol ON fundamental_metrics(symbol);

-- Create index on updated_at for cache management
CREATE INDEX IF NOT EXISTS idx_fundamental_metrics_updated_at ON fundamental_metrics(updated_at);
