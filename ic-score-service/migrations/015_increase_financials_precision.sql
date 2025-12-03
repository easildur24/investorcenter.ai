-- Migration 015: Increase precision for financials columns
-- Issue: NumericValueOutOfRangeError for tickers with extreme values
-- Affected: ADGM, ASH, BHVN, CHD, CTBB, CTDD, EVTV
-- Root cause: DECIMAL(10,4) can only hold values up to ±999,999.9999

-- Increase eps columns to handle extreme EPS values (e.g., pre-split, micro-caps)
ALTER TABLE financials
    ALTER COLUMN eps_basic TYPE DECIMAL(15, 4),
    ALTER COLUMN eps_diluted TYPE DECIMAL(15, 4);

-- Increase ratio/margin columns to handle extreme percentages
-- (e.g., -30,000% margin for pre-revenue companies)
ALTER TABLE financials
    ALTER COLUMN pe_ratio TYPE DECIMAL(15, 4),
    ALTER COLUMN pb_ratio TYPE DECIMAL(15, 4),
    ALTER COLUMN ps_ratio TYPE DECIMAL(15, 4),
    ALTER COLUMN debt_to_equity TYPE DECIMAL(15, 4),
    ALTER COLUMN current_ratio TYPE DECIMAL(15, 4),
    ALTER COLUMN quick_ratio TYPE DECIMAL(15, 4),
    ALTER COLUMN roe TYPE DECIMAL(15, 4),
    ALTER COLUMN roa TYPE DECIMAL(15, 4),
    ALTER COLUMN roic TYPE DECIMAL(15, 4),
    ALTER COLUMN gross_margin TYPE DECIMAL(15, 4),
    ALTER COLUMN operating_margin TYPE DECIMAL(15, 4),
    ALTER COLUMN net_margin TYPE DECIMAL(15, 4);

-- Also update fundamental_metrics_extended table for consistency
ALTER TABLE fundamental_metrics_extended
    ALTER COLUMN gross_margin TYPE DECIMAL(15, 4),
    ALTER COLUMN operating_margin TYPE DECIMAL(15, 4),
    ALTER COLUMN net_margin TYPE DECIMAL(15, 4),
    ALTER COLUMN roe TYPE DECIMAL(15, 4),
    ALTER COLUMN roa TYPE DECIMAL(15, 4),
    ALTER COLUMN roic TYPE DECIMAL(15, 4),
    ALTER COLUMN debt_to_equity TYPE DECIMAL(15, 4),
    ALTER COLUMN current_ratio TYPE DECIMAL(15, 4),
    ALTER COLUMN quick_ratio TYPE DECIMAL(15, 4);

-- Verify changes
DO $$
BEGIN
    RAISE NOTICE 'Migration 015 complete: Increased precision for financials and fundamental_metrics_extended columns';
END $$;
