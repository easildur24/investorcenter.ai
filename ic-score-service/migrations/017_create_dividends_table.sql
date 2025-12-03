-- Migration: Create dividends table for storing stock dividend history
-- This table stores historical dividend data from Polygon API
-- Used to calculate dividend_yield in fundamental_metrics_extended

-- Create dividends table
CREATE TABLE IF NOT EXISTS dividends (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    ex_date DATE NOT NULL,              -- Ex-dividend date
    pay_date DATE,                       -- Payment date
    record_date DATE,                    -- Record date
    declaration_date DATE,               -- Declaration date
    amount DECIMAL(12, 6) NOT NULL,      -- Dividend amount per share
    currency VARCHAR(10) DEFAULT 'USD',  -- Currency
    frequency INTEGER,                   -- Dividend frequency (1=annual, 2=semi-annual, 4=quarterly, 12=monthly)
    type VARCHAR(20) DEFAULT 'CD',       -- Dividend type (CD=cash dividend, SC=special cash, etc.)
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    -- Unique constraint to prevent duplicates
    CONSTRAINT dividends_unique UNIQUE (symbol, ex_date, type)
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_dividends_symbol ON dividends(symbol);
CREATE INDEX IF NOT EXISTS idx_dividends_ex_date ON dividends(ex_date DESC);
CREATE INDEX IF NOT EXISTS idx_dividends_symbol_ex_date ON dividends(symbol, ex_date DESC);

-- Create function to get annual dividend for a ticker (sum of last 12 months)
CREATE OR REPLACE FUNCTION get_annual_dividend(p_symbol VARCHAR)
RETURNS DECIMAL AS $$
DECLARE
    annual_div DECIMAL;
BEGIN
    SELECT COALESCE(SUM(amount), 0)
    INTO annual_div
    FROM dividends
    WHERE symbol = p_symbol
      AND ex_date >= CURRENT_DATE - INTERVAL '12 months'
      AND type IN ('CD', 'SC');  -- Include regular and special cash dividends

    RETURN annual_div;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions
GRANT SELECT, INSERT, UPDATE ON dividends TO investorcenter;
GRANT USAGE, SELECT ON SEQUENCE dividends_id_seq TO investorcenter;

-- Log completion
DO $$
BEGIN
    RAISE NOTICE 'Dividends table created successfully';
    RAISE NOTICE 'Run the dividend backfill job to populate data';
END $$;
