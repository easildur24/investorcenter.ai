-- Migration: Create screener_data materialized view for fast screener queries
-- This pre-computes the latest price, valuation, metrics, and IC score for each ticker
-- Reduces screener query time from 2-3 minutes to <100ms

-- Drop if exists (for re-running)
DROP MATERIALIZED VIEW IF EXISTS screener_data;

-- Create the materialized view
CREATE MATERIALIZED VIEW screener_data AS
SELECT
    t.symbol,
    t.name,
    COALESCE(t.sector, '') as sector,
    COALESCE(t.industry, '') as industry,
    t.market_cap,
    t.asset_type,
    t.active,
    lp.price,
    lv.ttm_pe_ratio as pe_ratio,
    lv.ttm_pb_ratio as pb_ratio,
    lv.ttm_ps_ratio as ps_ratio,
    lm.revenue_growth_yoy as revenue_growth,
    lm.dividend_yield,
    lm.roe,
    lm.beta,
    lic.overall_score as ic_score,
    CURRENT_TIMESTAMP as refreshed_at
FROM tickers t
LEFT JOIN LATERAL (
    SELECT close as price
    FROM stock_prices
    WHERE ticker = t.symbol
    ORDER BY time DESC
    LIMIT 1
) lp ON true
LEFT JOIN LATERAL (
    SELECT ttm_pe_ratio, ttm_pb_ratio, ttm_ps_ratio
    FROM valuation_ratios
    WHERE ticker = t.symbol
    ORDER BY calculation_date DESC
    LIMIT 1
) lv ON true
LEFT JOIN LATERAL (
    SELECT revenue_growth_yoy, dividend_yield, roe, beta
    FROM fundamental_metrics_extended
    WHERE ticker = t.symbol
    ORDER BY calculation_date DESC
    LIMIT 1
) lm ON true
LEFT JOIN LATERAL (
    SELECT overall_score
    FROM ic_scores
    WHERE ticker = t.symbol
    ORDER BY date DESC
    LIMIT 1
) lic ON true
WHERE t.asset_type = 'CS' AND t.active = true;

-- Create indexes for fast filtering and sorting
CREATE UNIQUE INDEX idx_screener_data_symbol ON screener_data(symbol);
CREATE INDEX idx_screener_data_market_cap ON screener_data(market_cap DESC NULLS LAST);
CREATE INDEX idx_screener_data_sector ON screener_data(sector);
CREATE INDEX idx_screener_data_pe_ratio ON screener_data(pe_ratio);
CREATE INDEX idx_screener_data_ic_score ON screener_data(ic_score DESC NULLS LAST);
CREATE INDEX idx_screener_data_dividend_yield ON screener_data(dividend_yield DESC NULLS LAST);

-- Create a function to refresh the materialized view
CREATE OR REPLACE FUNCTION refresh_screener_data()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY screener_data;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions
GRANT SELECT ON screener_data TO investorcenter;

-- Log completion
DO $$
BEGIN
    RAISE NOTICE 'Materialized view screener_data created successfully';
    RAISE NOTICE 'Run: SELECT refresh_screener_data(); to refresh after data updates';
END $$;
