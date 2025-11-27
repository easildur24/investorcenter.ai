-- Migration: Create sector comparison materialized views
-- Description: Materialized views for sector and industry metric averages
-- Author: IC Score Pipeline
-- Date: 2024-11-27

-- ============================================================================
-- Materialized View: sector_metric_averages
-- Purpose: Aggregate fundamental metrics by sector for benchmarking
-- Refresh: Daily via CronJob after fundamental_metrics_extended is updated
-- ============================================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS sector_metric_averages AS
WITH latest_metrics AS (
    -- Get most recent metrics for each ticker
    SELECT DISTINCT ON (fme.ticker)
        fme.*,
        c.sector,
        c.industry
    FROM fundamental_metrics_extended fme
    JOIN companies c ON fme.ticker = c.ticker
    WHERE fme.calculation_date >= CURRENT_DATE - INTERVAL '7 days'
      AND c.sector IS NOT NULL
      AND c.sector != ''
    ORDER BY fme.ticker, fme.calculation_date DESC
)
SELECT
    sector,

    -- Count
    COUNT(*) as company_count,

    -- Valuation Averages (filter outliers)
    AVG(ev_to_revenue) FILTER (WHERE ev_to_revenue BETWEEN 0 AND 100) as avg_ev_revenue,
    AVG(ev_to_ebitda) FILTER (WHERE ev_to_ebitda BETWEEN 0 AND 100) as avg_ev_ebitda,
    AVG(ev_to_fcf) FILTER (WHERE ev_to_fcf BETWEEN 0 AND 100) as avg_ev_fcf,

    -- Profitability Averages
    AVG(gross_margin) FILTER (WHERE gross_margin BETWEEN -100 AND 100) as avg_gross_margin,
    AVG(operating_margin) FILTER (WHERE operating_margin BETWEEN -100 AND 100) as avg_operating_margin,
    AVG(net_margin) FILTER (WHERE net_margin BETWEEN -100 AND 100) as avg_net_margin,
    AVG(ebitda_margin) FILTER (WHERE ebitda_margin BETWEEN -100 AND 100) as avg_ebitda_margin,

    -- Return Averages
    AVG(roe) FILTER (WHERE roe BETWEEN -100 AND 200) as avg_roe,
    AVG(roa) FILTER (WHERE roa BETWEEN -100 AND 100) as avg_roa,
    AVG(roic) FILTER (WHERE roic BETWEEN -100 AND 100) as avg_roic,

    -- Growth Averages
    AVG(revenue_growth_yoy) FILTER (WHERE revenue_growth_yoy BETWEEN -100 AND 500) as avg_revenue_growth_yoy,
    AVG(eps_growth_yoy) FILTER (WHERE eps_growth_yoy BETWEEN -100 AND 500) as avg_eps_growth_yoy,
    AVG(revenue_growth_5y_cagr) FILTER (WHERE revenue_growth_5y_cagr BETWEEN -50 AND 100) as avg_revenue_growth_5y,

    -- Leverage Averages
    AVG(debt_to_equity) FILTER (WHERE debt_to_equity BETWEEN 0 AND 10) as avg_debt_to_equity,
    AVG(interest_coverage) FILTER (WHERE interest_coverage BETWEEN 0 AND 100) as avg_interest_coverage,
    AVG(net_debt_to_ebitda) FILTER (WHERE net_debt_to_ebitda BETWEEN -5 AND 20) as avg_net_debt_to_ebitda,

    -- Liquidity Averages
    AVG(current_ratio) FILTER (WHERE current_ratio BETWEEN 0 AND 10) as avg_current_ratio,
    AVG(quick_ratio) FILTER (WHERE quick_ratio BETWEEN 0 AND 10) as avg_quick_ratio,

    -- Dividend Averages
    AVG(dividend_yield) FILTER (WHERE dividend_yield BETWEEN 0 AND 20) as avg_dividend_yield,
    AVG(payout_ratio) FILTER (WHERE payout_ratio BETWEEN 0 AND 200) as avg_payout_ratio,

    -- Medians for key metrics (more robust than averages)
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY ev_to_revenue)
        FILTER (WHERE ev_to_revenue BETWEEN 0 AND 100) as median_ev_revenue,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY net_margin)
        FILTER (WHERE net_margin BETWEEN -100 AND 100) as median_net_margin,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY roe)
        FILTER (WHERE roe BETWEEN -100 AND 200) as median_roe,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY debt_to_equity)
        FILTER (WHERE debt_to_equity BETWEEN 0 AND 10) as median_debt_to_equity,

    -- Percentiles for ranking (25th, 75th)
    PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY roe)
        FILTER (WHERE roe BETWEEN -100 AND 200) as roe_25th_percentile,
    PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY roe)
        FILTER (WHERE roe BETWEEN -100 AND 200) as roe_75th_percentile,
    PERCENTILE_CONT(0.25) WITHIN GROUP (ORDER BY net_margin)
        FILTER (WHERE net_margin BETWEEN -100 AND 100) as margin_25th_percentile,
    PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY net_margin)
        FILTER (WHERE net_margin BETWEEN -100 AND 100) as margin_75th_percentile,

    -- Timestamp
    NOW() as calculated_at

FROM latest_metrics
WHERE sector IS NOT NULL
GROUP BY sector;

-- Index for fast lookups
CREATE UNIQUE INDEX IF NOT EXISTS idx_sector_averages_sector
ON sector_metric_averages(sector);

-- ============================================================================
-- Materialized View: industry_metric_averages
-- Purpose: More granular aggregation by industry for peer comparison
-- ============================================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS industry_metric_averages AS
WITH latest_metrics AS (
    SELECT DISTINCT ON (fme.ticker)
        fme.*,
        c.sector,
        c.industry
    FROM fundamental_metrics_extended fme
    JOIN companies c ON fme.ticker = c.ticker
    WHERE fme.calculation_date >= CURRENT_DATE - INTERVAL '7 days'
      AND c.industry IS NOT NULL
      AND c.industry != ''
    ORDER BY fme.ticker, fme.calculation_date DESC
)
SELECT
    sector,
    industry,
    COUNT(*) as company_count,

    -- Same metrics as sector view
    AVG(ev_to_revenue) FILTER (WHERE ev_to_revenue BETWEEN 0 AND 100) as avg_ev_revenue,
    AVG(gross_margin) FILTER (WHERE gross_margin BETWEEN -100 AND 100) as avg_gross_margin,
    AVG(operating_margin) FILTER (WHERE operating_margin BETWEEN -100 AND 100) as avg_operating_margin,
    AVG(net_margin) FILTER (WHERE net_margin BETWEEN -100 AND 100) as avg_net_margin,
    AVG(roe) FILTER (WHERE roe BETWEEN -100 AND 200) as avg_roe,
    AVG(roa) FILTER (WHERE roa BETWEEN -100 AND 100) as avg_roa,
    AVG(revenue_growth_yoy) FILTER (WHERE revenue_growth_yoy BETWEEN -100 AND 500) as avg_revenue_growth_yoy,
    AVG(debt_to_equity) FILTER (WHERE debt_to_equity BETWEEN 0 AND 10) as avg_debt_to_equity,
    AVG(dividend_yield) FILTER (WHERE dividend_yield BETWEEN 0 AND 20) as avg_dividend_yield,

    -- Medians
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY net_margin)
        FILTER (WHERE net_margin BETWEEN -100 AND 100) as median_net_margin,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY roe)
        FILTER (WHERE roe BETWEEN -100 AND 200) as median_roe,

    NOW() as calculated_at
FROM latest_metrics
WHERE industry IS NOT NULL
GROUP BY sector, industry
HAVING COUNT(*) >= 5;  -- Only industries with 5+ companies

CREATE UNIQUE INDEX IF NOT EXISTS idx_industry_averages_industry
ON industry_metric_averages(sector, industry);

-- ============================================================================
-- Helper Function: Calculate Percentile Rank
-- ============================================================================

CREATE OR REPLACE FUNCTION calculate_sector_percentile(
    p_ticker VARCHAR(10),
    p_metric_value DECIMAL,
    p_metric_name VARCHAR(50),
    p_higher_is_better BOOLEAN DEFAULT TRUE
) RETURNS INTEGER AS $$
DECLARE
    v_sector VARCHAR(100);
    v_count_below INTEGER;
    v_count_equal INTEGER;
    v_total_count INTEGER;
    v_percentile DECIMAL;
BEGIN
    -- Get company's sector
    SELECT sector INTO v_sector
    FROM companies
    WHERE ticker = p_ticker;

    IF v_sector IS NULL THEN
        RETURN NULL;
    END IF;

    IF p_metric_value IS NULL THEN
        RETURN NULL;
    END IF;

    -- Count companies with values below and equal to target
    EXECUTE format(
        'SELECT
            COUNT(*) FILTER (WHERE %I < $1),
            COUNT(*) FILTER (WHERE %I = $1),
            COUNT(*) FILTER (WHERE %I IS NOT NULL)
         FROM fundamental_metrics_extended fme
         JOIN companies c ON fme.ticker = c.ticker
         WHERE c.sector = $2
           AND fme.calculation_date >= CURRENT_DATE - INTERVAL ''7 days''',
        p_metric_name, p_metric_name, p_metric_name
    )
    INTO v_count_below, v_count_equal, v_total_count
    USING p_metric_value, v_sector;

    IF v_total_count = 0 THEN
        RETURN NULL;
    END IF;

    -- Standard percentile rank formula: (B + 0.5E) / N * 100
    v_percentile := ((v_count_below + 0.5 * v_count_equal) / v_total_count) * 100;

    -- Invert if lower is better (e.g., P/E, debt ratios)
    IF NOT p_higher_is_better THEN
        v_percentile := 100 - v_percentile;
    END IF;

    RETURN ROUND(v_percentile);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION calculate_sector_percentile IS 'Calculate percentile rank of a metric within a sector. Returns 0-100 where higher means better performance.';

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON MATERIALIZED VIEW sector_metric_averages IS 'Sector-level aggregate metrics for benchmarking. Refresh daily after fundamental_metrics_extended update.';
COMMENT ON MATERIALIZED VIEW industry_metric_averages IS 'Industry-level aggregate metrics for peer comparison. Only industries with 5+ companies included.';

-- ============================================================================
-- Sample Queries
-- ============================================================================

-- Refresh materialized views (run via CronJob)
-- REFRESH MATERIALIZED VIEW CONCURRENTLY sector_metric_averages;
-- REFRESH MATERIALIZED VIEW CONCURRENTLY industry_metric_averages;

-- Get sector averages
-- SELECT * FROM sector_metric_averages WHERE sector = 'Technology';

-- Compare company to sector average
-- SELECT
--     fme.ticker,
--     fme.roe as company_roe,
--     sma.avg_roe as sector_avg_roe,
--     fme.roe - sma.avg_roe as roe_vs_sector,
--     fme.net_margin as company_margin,
--     sma.avg_net_margin as sector_avg_margin
-- FROM fundamental_metrics_extended fme
-- JOIN companies c ON fme.ticker = c.ticker
-- JOIN sector_metric_averages sma ON c.sector = sma.sector
-- WHERE fme.ticker = 'AAPL'
--   AND fme.calculation_date = CURRENT_DATE;

-- Find companies outperforming sector averages
-- SELECT
--     fme.ticker,
--     c.sector,
--     fme.roe,
--     sma.avg_roe as sector_avg,
--     fme.roe - sma.avg_roe as outperformance
-- FROM fundamental_metrics_extended fme
-- JOIN companies c ON fme.ticker = c.ticker
-- JOIN sector_metric_averages sma ON c.sector = sma.sector
-- WHERE fme.calculation_date = CURRENT_DATE
--   AND fme.roe > sma.avg_roe
-- ORDER BY fme.roe - sma.avg_roe DESC
-- LIMIT 20;
