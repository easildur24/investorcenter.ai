-- Migration 012: Fix fiscal year mislabeling and cleanup NULL records
-- This fixes two major data quality issues:
-- 1. Removes garbage records with no financial data (NULL revenue, EPS, net_income)
-- 2. Documents the fiscal year correction that happens in the SEC client

-- Step 1: Delete garbage records (no revenue AND no EPS AND no net_income)
-- These are incomplete XBRL records that shouldn't have been ingested
DELETE FROM financials
WHERE statement_type = '10-Q'
  AND revenue IS NULL
  AND eps_diluted IS NULL
  AND eps_basic IS NULL
  AND net_income IS NULL;

-- Step 2: Get count of remaining records
DO $$
DECLARE
    remaining_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO remaining_count FROM financials WHERE statement_type = '10-Q';
    RAISE NOTICE 'Remaining 10-Q records after cleanup: %', remaining_count;
END $$;

-- Step 3: Add validation constraint to prevent future NULL pollution
-- Require at least ONE of: revenue, net_income, or EPS
ALTER TABLE financials
ADD CONSTRAINT check_has_financial_data
CHECK (
    revenue IS NOT NULL
    OR net_income IS NOT NULL
    OR eps_basic IS NOT NULL
    OR eps_diluted IS NOT NULL
    OR statement_type = '10-K'  -- Allow 10-K to have NULLs (balance sheet only)
);

-- Step 4: Create index to speed up TTM calculator queries
CREATE INDEX IF NOT EXISTS idx_financials_ticker_fiscal_year_quarter
ON financials(ticker, fiscal_year DESC, fiscal_quarter, period_end_date DESC);

-- Note: Fiscal year correction happens in the SEC client (sec_client.py:_correct_fiscal_years)
-- The correction logic:
--   - Determines fiscal year-end month from annual filings
--   - For quarters ending after FYE month: fiscal_year = period_end_year + 1
--   - For quarters ending before/on FYE month: fiscal_year = period_end_year
--
-- Example (Home Depot with Feb fiscal year-end):
--   - Q2 ending Jul 28, 2024: 7 > 2, so FY = 2024 + 1 = 2025 ✓
--   - Q3 ending Oct 27, 2024: 10 > 2, so FY = 2024 + 1 = 2025 ✓
