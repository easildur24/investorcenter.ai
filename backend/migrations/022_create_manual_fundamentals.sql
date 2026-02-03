-- Create table for manually ingested fundamental data
-- This allows flexible storage of calculated metrics as JSON

CREATE TABLE IF NOT EXISTS manual_fundamentals (
    ticker VARCHAR(10) PRIMARY KEY,
    data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_manual_fundamentals_ticker ON manual_fundamentals(ticker);

-- Index for querying JSON data (optional, but useful)
CREATE INDEX IF NOT EXISTS idx_manual_fundamentals_data ON manual_fundamentals USING GIN (data);

-- Trigger to auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_manual_fundamentals_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_manual_fundamentals_updated_at
    BEFORE UPDATE ON manual_fundamentals
    FOR EACH ROW
    EXECUTE FUNCTION update_manual_fundamentals_updated_at();
