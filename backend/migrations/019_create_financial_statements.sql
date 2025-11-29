-- Migration: create_financial_statements_table
-- Description: Creates table for storing financial statements from Polygon.io SEC EDGAR data

-- Create financial_statements table
CREATE TABLE IF NOT EXISTS financial_statements (
    id SERIAL PRIMARY KEY,
    ticker_id INTEGER NOT NULL REFERENCES tickers(id) ON DELETE CASCADE,
    cik VARCHAR(10),
    statement_type VARCHAR(20) NOT NULL CHECK (statement_type IN ('income', 'balance_sheet', 'cash_flow', 'ratios')),
    timeframe VARCHAR(30) NOT NULL CHECK (timeframe IN ('quarterly', 'annual', 'trailing_twelve_months')),
    fiscal_year INTEGER NOT NULL,
    fiscal_quarter INTEGER CHECK (fiscal_quarter >= 1 AND fiscal_quarter <= 4),
    period_start DATE,
    period_end DATE NOT NULL,
    filed_date DATE,
    source_filing_url TEXT,
    source_filing_type VARCHAR(20), -- '10-K', '10-Q', '8-K', etc.
    data JSONB NOT NULL, -- Full statement data with all line items
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Ensure unique statements per ticker/type/period combination
    UNIQUE(ticker_id, statement_type, timeframe, fiscal_year, fiscal_quarter)
);

-- Create indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_financial_statements_ticker ON financial_statements(ticker_id);
CREATE INDEX IF NOT EXISTS idx_financial_statements_period ON financial_statements(period_end DESC);
CREATE INDEX IF NOT EXISTS idx_financial_statements_type ON financial_statements(statement_type, timeframe);
CREATE INDEX IF NOT EXISTS idx_financial_statements_fiscal ON financial_statements(fiscal_year DESC, fiscal_quarter DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS idx_financial_statements_filed ON financial_statements(filed_date DESC);

-- Create GIN index for JSONB data queries
CREATE INDEX IF NOT EXISTS idx_financial_statements_data ON financial_statements USING GIN (data);

-- Create symbol lookup index (join with tickers table)
CREATE INDEX IF NOT EXISTS idx_financial_statements_ticker_type_period ON financial_statements(ticker_id, statement_type, period_end DESC);

-- Add comments for documentation
COMMENT ON TABLE financial_statements IS 'Stores financial statement data from SEC EDGAR filings via Polygon.io API';
COMMENT ON COLUMN financial_statements.ticker_id IS 'Foreign key reference to tickers table';
COMMENT ON COLUMN financial_statements.cik IS 'SEC Central Index Key for the company';
COMMENT ON COLUMN financial_statements.statement_type IS 'Type of financial statement: income, balance_sheet, cash_flow, or ratios';
COMMENT ON COLUMN financial_statements.timeframe IS 'Reporting period: quarterly, annual, or trailing_twelve_months';
COMMENT ON COLUMN financial_statements.fiscal_year IS 'Fiscal year of the statement';
COMMENT ON COLUMN financial_statements.fiscal_quarter IS 'Fiscal quarter (1-4), NULL for annual statements';
COMMENT ON COLUMN financial_statements.period_start IS 'Start date of the reporting period';
COMMENT ON COLUMN financial_statements.period_end IS 'End date of the reporting period';
COMMENT ON COLUMN financial_statements.filed_date IS 'Date the filing was submitted to SEC';
COMMENT ON COLUMN financial_statements.source_filing_url IS 'URL to the original SEC filing';
COMMENT ON COLUMN financial_statements.source_filing_type IS 'Type of SEC filing (10-K, 10-Q, etc.)';
COMMENT ON COLUMN financial_statements.data IS 'JSONB containing all financial line items from the statement';

-- Create function to auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_financial_statements_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for auto-updating updated_at
DROP TRIGGER IF EXISTS trigger_financial_statements_updated_at ON financial_statements;
CREATE TRIGGER trigger_financial_statements_updated_at
    BEFORE UPDATE ON financial_statements
    FOR EACH ROW
    EXECUTE FUNCTION update_financial_statements_updated_at();
