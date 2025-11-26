-- Migration: Create TTM Financials and Valuation Ratios tables
-- Description: Add support for Trailing Twelve Months (TTM) financial data and daily valuation ratios
-- Author: IC Score Pipeline
-- Date: 2024-11-18

-- ============================================================================
-- Table: ttm_financials
-- Purpose: Store calculated Trailing Twelve Months (TTM) financial metrics
-- ============================================================================

CREATE TABLE IF NOT EXISTS ttm_financials (
    id BIGSERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    calculation_date DATE NOT NULL,
    ttm_period_start DATE NOT NULL,
    ttm_period_end DATE NOT NULL,

    -- Income Statement Items (summed from 4 quarters)
    revenue BIGINT,
    cost_of_revenue BIGINT,
    gross_profit BIGINT,
    operating_expenses BIGINT,
    operating_income BIGINT,
    net_income BIGINT,
    eps_basic NUMERIC(10,4),
    eps_diluted NUMERIC(10,4),

    -- Balance Sheet Items (from most recent quarter)
    shares_outstanding BIGINT,
    total_assets BIGINT,
    total_liabilities BIGINT,
    shareholders_equity BIGINT,
    cash_and_equivalents BIGINT,
    short_term_debt BIGINT,
    long_term_debt BIGINT,

    -- Cash Flow Items (summed from 4 quarters)
    operating_cash_flow BIGINT,
    investing_cash_flow BIGINT,
    financing_cash_flow BIGINT,
    free_cash_flow BIGINT,
    capex BIGINT,

    -- Metadata
    quarters_included JSONB,  -- Array of quarter details used in calculation
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    CONSTRAINT uq_ttm_financials_ticker_date UNIQUE (ticker, calculation_date)
);

-- Indexes for ttm_financials
CREATE INDEX idx_ttm_financials_ticker ON ttm_financials(ticker);
CREATE INDEX idx_ttm_financials_calculation_date ON ttm_financials(calculation_date);
CREATE INDEX idx_ttm_financials_period_end ON ttm_financials(ttm_period_end);

COMMENT ON TABLE ttm_financials IS 'Trailing Twelve Months (TTM) financial metrics calculated by summing last 4 quarters';
COMMENT ON COLUMN ttm_financials.calculation_date IS 'Date when TTM metrics were calculated';
COMMENT ON COLUMN ttm_financials.ttm_period_start IS 'Start date of the 12-month period (oldest quarter end date)';
COMMENT ON COLUMN ttm_financials.ttm_period_end IS 'End date of the 12-month period (most recent quarter end date)';
COMMENT ON COLUMN ttm_financials.quarters_included IS 'JSON array of quarters used: [{period_end_date, fiscal_year, fiscal_quarter}, ...]';

-- ============================================================================
-- Table: valuation_ratios
-- Purpose: Store daily valuation ratios using both TTM and Annual data
-- ============================================================================

CREATE TABLE IF NOT EXISTS valuation_ratios (
    id BIGSERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    calculation_date DATE NOT NULL,
    stock_price NUMERIC(10,2) NOT NULL,

    -- TTM-based Ratios
    ttm_pe_ratio NUMERIC(10,2),
    ttm_pb_ratio NUMERIC(10,2),
    ttm_ps_ratio NUMERIC(10,2),
    ttm_market_cap BIGINT,
    ttm_financial_id BIGINT REFERENCES ttm_financials(id) ON DELETE SET NULL,
    ttm_period_start DATE,
    ttm_period_end DATE,

    -- Annual-based Ratios
    annual_pe_ratio NUMERIC(10,2),
    annual_pb_ratio NUMERIC(10,2),
    annual_ps_ratio NUMERIC(10,2),
    annual_market_cap BIGINT,
    annual_financial_id BIGINT REFERENCES financials(id) ON DELETE SET NULL,
    annual_period_end DATE,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    CONSTRAINT uq_valuation_ratios_ticker_date UNIQUE (ticker, calculation_date)
);

-- Indexes for valuation_ratios
CREATE INDEX idx_valuation_ratios_ticker ON valuation_ratios(ticker);
CREATE INDEX idx_valuation_ratios_calculation_date ON valuation_ratios(calculation_date);
CREATE INDEX idx_valuation_ratios_ttm_financial_id ON valuation_ratios(ttm_financial_id);
CREATE INDEX idx_valuation_ratios_annual_financial_id ON valuation_ratios(annual_financial_id);

COMMENT ON TABLE valuation_ratios IS 'Daily valuation ratios calculated using current stock price with both TTM and Annual financial data';
COMMENT ON COLUMN valuation_ratios.stock_price IS 'Stock price used for ratio calculation (from Polygon API)';
COMMENT ON COLUMN valuation_ratios.ttm_pe_ratio IS 'Price-to-Earnings ratio using TTM earnings';
COMMENT ON COLUMN valuation_ratios.annual_pe_ratio IS 'Price-to-Earnings ratio using annual (10-K) earnings';

-- ============================================================================
-- Sample Queries
-- ============================================================================

-- Get latest TTM financials for a ticker
-- SELECT * FROM ttm_financials WHERE ticker = 'AAPL' ORDER BY calculation_date DESC LIMIT 1;

-- Get latest valuation ratios for a ticker (both TTM and Annual)
-- SELECT * FROM valuation_ratios WHERE ticker = 'AAPL' ORDER BY calculation_date DESC LIMIT 1;

-- Compare TTM vs Annual PE ratios
-- SELECT ticker, calculation_date, ttm_pe_ratio, annual_pe_ratio,
--        (ttm_pe_ratio - annual_pe_ratio) as pe_difference
-- FROM valuation_ratios
-- WHERE calculation_date = CURRENT_DATE
-- ORDER BY ABS(ttm_pe_ratio - annual_pe_ratio) DESC
-- LIMIT 20;

-- Get historical PE ratio trend for a ticker
-- SELECT calculation_date, stock_price, ttm_pe_ratio, annual_pe_ratio
-- FROM valuation_ratios
-- WHERE ticker = 'AAPL'
-- ORDER BY calculation_date DESC
-- LIMIT 90;  -- Last 90 days
