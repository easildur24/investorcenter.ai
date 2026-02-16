-- Migration: Expand screener_data materialized view with additional metrics
--
-- Adds columns from fundamental_metrics_extended and ic_scores that are
-- already computed by existing pipelines but were not included in the
-- original view (016). This enables filtering/sorting by:
--   - Profitability: ROA, gross margin, net margin, operating margin
--   - Financial health: debt/equity, current ratio
--   - Growth: EPS growth YoY
--   - Dividends: payout ratio, consecutive dividend years
--   - Fair value: DCF upside percent
--   - IC Score sub-factors: all 10 factor scores
--   - IC Score metadata: rating, sector percentile, lifecycle stage
--
-- The join structure is identical to 016 (same 4 LATERAL joins), just
-- selecting more columns from the same subqueries. Refresh time should
-- remain under 2 minutes.

BEGIN;

-- Drop existing view and indexes
DROP MATERIALIZED VIEW IF EXISTS screener_data;

-- Recreate with expanded columns
CREATE MATERIALIZED VIEW screener_data AS
SELECT
    -- Core identity
    t.symbol,
    t.name,
    COALESCE(t.sector, '') as sector,
    COALESCE(t.industry, '') as industry,
    t.market_cap,
    t.asset_type,
    t.active,

    -- Price (latest close from stock_prices)
    lp.price,

    -- Valuation ratios (from valuation_ratios)
    lv.ttm_pe_ratio as pe_ratio,
    lv.ttm_pb_ratio as pb_ratio,
    lv.ttm_ps_ratio as ps_ratio,

    -- Profitability (from fundamental_metrics_extended)
    lm.roe,
    lm.roa,
    lm.gross_margin,
    lm.operating_margin,
    lm.net_margin,

    -- Financial health
    lm.debt_to_equity,
    lm.current_ratio,

    -- Growth
    lm.revenue_growth_yoy as revenue_growth,
    lm.eps_growth_yoy,

    -- Dividends
    lm.dividend_yield,
    lm.payout_ratio,
    lm.consecutive_dividend_years,

    -- Risk
    lm.beta,

    -- Fair value
    lm.dcf_upside_percent,

    -- IC Score: overall
    lic.overall_score as ic_score,
    lic.rating as ic_rating,

    -- IC Score: 10 sub-factor scores
    lic.value_score,
    lic.growth_score,
    lic.profitability_score,
    lic.financial_health_score,
    lic.momentum_score,
    lic.analyst_consensus_score,
    lic.insider_activity_score,
    lic.institutional_score,
    lic.news_sentiment_score,
    lic.technical_score,

    -- IC Score: metadata
    lic.sector_percentile as ic_sector_percentile,
    lic.lifecycle_stage,
    lic.data_completeness as ic_data_completeness,

    -- View metadata
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
    SELECT
        -- Profitability
        roe, roa, gross_margin, operating_margin, net_margin,
        -- Financial health
        debt_to_equity, current_ratio,
        -- Growth
        revenue_growth_yoy, eps_growth_yoy,
        -- Dividends
        dividend_yield, payout_ratio, consecutive_dividend_years,
        -- Risk
        beta,
        -- Fair value
        dcf_upside_percent
    FROM fundamental_metrics_extended
    WHERE ticker = t.symbol
    ORDER BY calculation_date DESC
    LIMIT 1
) lm ON true
LEFT JOIN LATERAL (
    SELECT
        -- Overall
        overall_score, rating,
        -- Sub-factors
        value_score, growth_score, profitability_score,
        financial_health_score, momentum_score,
        analyst_consensus_score, insider_activity_score,
        institutional_score, news_sentiment_score, technical_score,
        -- Metadata
        sector_percentile, lifecycle_stage, data_completeness
    FROM ic_scores
    WHERE ticker = t.symbol
    ORDER BY date DESC
    LIMIT 1
) lic ON true
WHERE t.asset_type = 'CS' AND t.active = true;

-- ============================================================================
-- INDEXES
-- ============================================================================

-- Primary key equivalent (required for CONCURRENTLY refresh)
CREATE UNIQUE INDEX idx_screener_data_symbol
    ON screener_data(symbol);

-- Sorting/filtering: core columns
CREATE INDEX idx_screener_data_market_cap
    ON screener_data(market_cap DESC NULLS LAST);
CREATE INDEX idx_screener_data_sector
    ON screener_data(sector);
CREATE INDEX idx_screener_data_industry
    ON screener_data(industry);

-- Sorting/filtering: valuation
CREATE INDEX idx_screener_data_pe_ratio
    ON screener_data(pe_ratio)
    WHERE pe_ratio IS NOT NULL;
CREATE INDEX idx_screener_data_pb_ratio
    ON screener_data(pb_ratio)
    WHERE pb_ratio IS NOT NULL;

-- Sorting/filtering: profitability
CREATE INDEX idx_screener_data_roe
    ON screener_data(roe)
    WHERE roe IS NOT NULL;
CREATE INDEX idx_screener_data_gross_margin
    ON screener_data(gross_margin)
    WHERE gross_margin IS NOT NULL;

-- Sorting/filtering: financial health
CREATE INDEX idx_screener_data_debt_to_equity
    ON screener_data(debt_to_equity)
    WHERE debt_to_equity IS NOT NULL;

-- Sorting/filtering: growth
CREATE INDEX idx_screener_data_eps_growth
    ON screener_data(eps_growth_yoy)
    WHERE eps_growth_yoy IS NOT NULL;

-- Sorting/filtering: dividends
CREATE INDEX idx_screener_data_dividend_yield
    ON screener_data(dividend_yield DESC NULLS LAST);
CREATE INDEX idx_screener_data_payout_ratio
    ON screener_data(payout_ratio)
    WHERE payout_ratio IS NOT NULL;

-- Sorting/filtering: risk
CREATE INDEX idx_screener_data_beta
    ON screener_data(beta)
    WHERE beta IS NOT NULL;

-- Sorting/filtering: IC Score
CREATE INDEX idx_screener_data_ic_score
    ON screener_data(ic_score DESC NULLS LAST);

-- Sorting/filtering: IC Score sub-factors
CREATE INDEX idx_screener_data_value_score
    ON screener_data(value_score DESC NULLS LAST);
CREATE INDEX idx_screener_data_growth_score
    ON screener_data(growth_score DESC NULLS LAST);
CREATE INDEX idx_screener_data_momentum_score
    ON screener_data(momentum_score DESC NULLS LAST);
CREATE INDEX idx_screener_data_insider_score
    ON screener_data(insider_activity_score DESC NULLS LAST);

-- Sorting/filtering: fair value
CREATE INDEX idx_screener_data_dcf_upside
    ON screener_data(dcf_upside_percent DESC NULLS LAST);

-- ============================================================================
-- REFRESH FUNCTION (replace to keep in sync)
-- ============================================================================

CREATE OR REPLACE FUNCTION refresh_screener_data()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY screener_data;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- PERMISSIONS
-- ============================================================================

GRANT SELECT ON screener_data TO investorcenter;

-- Log completion
DO $$
BEGIN
    RAISE NOTICE 'Materialized view screener_data expanded successfully (migration 019)';
    RAISE NOTICE 'New columns: roa, gross_margin, operating_margin, net_margin, debt_to_equity, current_ratio, eps_growth_yoy, payout_ratio, consecutive_dividend_years, dcf_upside_percent, ic_rating, 10 IC sub-factor scores, lifecycle_stage';
    RAISE NOTICE 'Run: SELECT refresh_screener_data(); to refresh after data updates';
END $$;

COMMIT;
