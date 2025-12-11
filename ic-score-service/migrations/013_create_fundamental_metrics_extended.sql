-- Migration: Create fundamental_metrics_extended table
-- Description: Store calculated extended financial metrics for Phase 1-5 implementation
-- Author: IC Score Pipeline
-- Date: 2024-11-27

-- ============================================================================
-- Table: fundamental_metrics_extended
-- Purpose: Store calculated fundamental metrics including growth rates,
--          profitability, leverage, dividends, valuation, and fair value
-- ============================================================================

CREATE TABLE IF NOT EXISTS fundamental_metrics_extended (
    id BIGSERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    calculation_date DATE NOT NULL,

    -- Profitability Margins (as percentages)
    gross_margin DECIMAL(10, 4),
    operating_margin DECIMAL(10, 4),
    net_margin DECIMAL(10, 4),
    ebitda_margin DECIMAL(10, 4),

    -- Returns (as percentages)
    roe DECIMAL(10, 4),
    roa DECIMAL(10, 4),
    roic DECIMAL(10, 4),

    -- Growth Rates (as percentages)
    revenue_growth_yoy DECIMAL(10, 4),
    revenue_growth_3y_cagr DECIMAL(10, 4),
    revenue_growth_5y_cagr DECIMAL(10, 4),
    eps_growth_yoy DECIMAL(10, 4),
    eps_growth_3y_cagr DECIMAL(10, 4),
    eps_growth_5y_cagr DECIMAL(10, 4),
    fcf_growth_yoy DECIMAL(10, 4),

    -- Valuation
    enterprise_value DECIMAL(20, 2),
    ev_to_revenue DECIMAL(12, 4),
    ev_to_ebitda DECIMAL(12, 4),
    ev_to_fcf DECIMAL(12, 4),

    -- Liquidity
    current_ratio DECIMAL(10, 4),
    quick_ratio DECIMAL(10, 4),

    -- Debt/Leverage
    debt_to_equity DECIMAL(10, 4),
    interest_coverage DECIMAL(10, 4),
    net_debt_to_ebitda DECIMAL(10, 4),

    -- Dividends
    dividend_yield DECIMAL(10, 4),
    payout_ratio DECIMAL(10, 4),
    dividend_growth_rate DECIMAL(10, 4),
    consecutive_dividend_years INTEGER DEFAULT 0,

    -- Fair Value (Phase 5)
    dcf_fair_value DECIMAL(14, 2),
    dcf_upside_percent DECIMAL(10, 4),
    graham_number DECIMAL(14, 2),
    epv_fair_value DECIMAL(14, 2),

    -- Sector Comparisons (percentile ranks 0-100)
    pe_sector_percentile INTEGER,
    pb_sector_percentile INTEGER,
    roe_sector_percentile INTEGER,
    margin_sector_percentile INTEGER,

    -- WACC Components (for fair value calculation)
    wacc DECIMAL(10, 4),
    beta DECIMAL(10, 4),
    cost_of_equity DECIMAL(10, 4),
    cost_of_debt DECIMAL(10, 4),

    -- Metadata
    data_quality_score DECIMAL(5, 2),
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    -- Constraints
    CONSTRAINT uq_fundamental_metrics_ticker_date UNIQUE (ticker, calculation_date)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_fund_metrics_ticker ON fundamental_metrics_extended(ticker);
CREATE INDEX IF NOT EXISTS idx_fund_metrics_date ON fundamental_metrics_extended(calculation_date DESC);
CREATE INDEX IF NOT EXISTS idx_fund_metrics_ticker_date ON fundamental_metrics_extended(ticker, calculation_date DESC);

-- Index for sector comparison queries
CREATE INDEX IF NOT EXISTS idx_fund_metrics_roe ON fundamental_metrics_extended(roe) WHERE roe IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_fund_metrics_net_margin ON fundamental_metrics_extended(net_margin) WHERE net_margin IS NOT NULL;

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_fundamental_metrics_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_fundamental_metrics_updated ON fundamental_metrics_extended;
CREATE TRIGGER trigger_fundamental_metrics_updated
    BEFORE UPDATE ON fundamental_metrics_extended
    FOR EACH ROW
    EXECUTE FUNCTION update_fundamental_metrics_timestamp();

-- Comments for documentation
COMMENT ON TABLE fundamental_metrics_extended IS 'Extended fundamental metrics calculated from TTM financials, including growth rates, leverage ratios, dividend metrics, and fair value estimates';

COMMENT ON COLUMN fundamental_metrics_extended.gross_margin IS 'Gross Profit / Revenue * 100';
COMMENT ON COLUMN fundamental_metrics_extended.operating_margin IS 'Operating Income / Revenue * 100';
COMMENT ON COLUMN fundamental_metrics_extended.net_margin IS 'Net Income / Revenue * 100';
COMMENT ON COLUMN fundamental_metrics_extended.ebitda_margin IS 'EBITDA / Revenue * 100';

COMMENT ON COLUMN fundamental_metrics_extended.roe IS 'Return on Equity = Net Income / Shareholders Equity * 100';
COMMENT ON COLUMN fundamental_metrics_extended.roa IS 'Return on Assets = Net Income / Total Assets * 100';
COMMENT ON COLUMN fundamental_metrics_extended.roic IS 'Return on Invested Capital';

COMMENT ON COLUMN fundamental_metrics_extended.revenue_growth_yoy IS 'Year-over-Year revenue growth (%)';
COMMENT ON COLUMN fundamental_metrics_extended.revenue_growth_3y_cagr IS '3-Year Compound Annual Growth Rate for revenue (%)';
COMMENT ON COLUMN fundamental_metrics_extended.revenue_growth_5y_cagr IS '5-Year Compound Annual Growth Rate for revenue (%)';
COMMENT ON COLUMN fundamental_metrics_extended.eps_growth_yoy IS 'Year-over-Year EPS growth (%), NULL if sign change';
COMMENT ON COLUMN fundamental_metrics_extended.fcf_growth_yoy IS 'Year-over-Year FCF growth (%), NULL if sign change';

COMMENT ON COLUMN fundamental_metrics_extended.enterprise_value IS 'Market Cap + Total Debt - Cash';
COMMENT ON COLUMN fundamental_metrics_extended.ev_to_revenue IS 'Enterprise Value / TTM Revenue';
COMMENT ON COLUMN fundamental_metrics_extended.ev_to_ebitda IS 'Enterprise Value / TTM EBITDA';
COMMENT ON COLUMN fundamental_metrics_extended.ev_to_fcf IS 'Enterprise Value / TTM Free Cash Flow';

COMMENT ON COLUMN fundamental_metrics_extended.debt_to_equity IS 'Total Debt / Shareholders Equity';
COMMENT ON COLUMN fundamental_metrics_extended.interest_coverage IS 'EBIT / Interest Expense (Times Interest Earned)';
COMMENT ON COLUMN fundamental_metrics_extended.net_debt_to_ebitda IS '(Total Debt - Cash) / EBITDA';

COMMENT ON COLUMN fundamental_metrics_extended.dividend_yield IS 'Annual Dividend / Stock Price * 100';
COMMENT ON COLUMN fundamental_metrics_extended.payout_ratio IS 'Annual Dividend / EPS * 100';
COMMENT ON COLUMN fundamental_metrics_extended.dividend_growth_rate IS '5-Year CAGR of dividend payments (%)';
COMMENT ON COLUMN fundamental_metrics_extended.consecutive_dividend_years IS 'Number of consecutive years of dividend increases';

COMMENT ON COLUMN fundamental_metrics_extended.dcf_fair_value IS 'DCF model fair value per share';
COMMENT ON COLUMN fundamental_metrics_extended.dcf_upside_percent IS '(DCF Fair Value - Current Price) / Current Price * 100';
COMMENT ON COLUMN fundamental_metrics_extended.graham_number IS 'sqrt(22.5 * EPS * Book Value per Share)';
COMMENT ON COLUMN fundamental_metrics_extended.epv_fair_value IS 'Earnings Power Value per share (Bruce Greenwald)';

COMMENT ON COLUMN fundamental_metrics_extended.wacc IS 'Weighted Average Cost of Capital (%)';
COMMENT ON COLUMN fundamental_metrics_extended.beta IS 'Stock beta vs market';
COMMENT ON COLUMN fundamental_metrics_extended.cost_of_equity IS 'CAPM cost of equity (%)';
COMMENT ON COLUMN fundamental_metrics_extended.cost_of_debt IS 'Interest Expense / Total Debt (%)';

COMMENT ON COLUMN fundamental_metrics_extended.pe_sector_percentile IS 'P/E ratio percentile within sector (0-100, lower is better)';
COMMENT ON COLUMN fundamental_metrics_extended.pb_sector_percentile IS 'P/B ratio percentile within sector (0-100, lower is better)';
COMMENT ON COLUMN fundamental_metrics_extended.roe_sector_percentile IS 'ROE percentile within sector (0-100, higher is better)';
COMMENT ON COLUMN fundamental_metrics_extended.margin_sector_percentile IS 'Net margin percentile within sector (0-100, higher is better)';

COMMENT ON COLUMN fundamental_metrics_extended.data_quality_score IS 'Data completeness score (0-100)';

-- ============================================================================
-- Sample Queries
-- ============================================================================

-- Get latest metrics for a ticker
-- SELECT * FROM fundamental_metrics_extended WHERE ticker = 'AAPL' ORDER BY calculation_date DESC LIMIT 1;

-- Find dividend aristocrats (25+ years of consecutive dividend growth)
-- SELECT ticker, consecutive_dividend_years, dividend_yield, payout_ratio
-- FROM fundamental_metrics_extended
-- WHERE calculation_date = CURRENT_DATE
--   AND consecutive_dividend_years >= 25
-- ORDER BY consecutive_dividend_years DESC;

-- Find undervalued stocks (DCF upside > 20%)
-- SELECT ticker, dcf_fair_value, dcf_upside_percent, wacc
-- FROM fundamental_metrics_extended
-- WHERE calculation_date = CURRENT_DATE
--   AND dcf_upside_percent > 20
--   AND data_quality_score >= 70
-- ORDER BY dcf_upside_percent DESC;

-- Find high ROE, low debt companies
-- SELECT ticker, roe, debt_to_equity, net_margin
-- FROM fundamental_metrics_extended
-- WHERE calculation_date = CURRENT_DATE
--   AND roe > 15
--   AND debt_to_equity < 1
--   AND net_margin > 10
-- ORDER BY roe DESC;
