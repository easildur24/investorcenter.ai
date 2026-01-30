-- Migration: Update analyst_ratings table for FMP ingestion
-- This migration:
-- 1. Makes rating column nullable (price targets may not have ratings)
-- 2. Adds unique constraint for upsert operations
-- 3. Adds index for source filtering

-- Make rating nullable (price targets from FMP may not have explicit ratings)
ALTER TABLE analyst_ratings ALTER COLUMN rating DROP NOT NULL;

-- Add unique constraint for upsert operations (ON CONFLICT requires a constraint, not partial index)
-- This prevents duplicate entries for same ticker/firm/date combination
ALTER TABLE analyst_ratings ADD CONSTRAINT analyst_ratings_unique_key
UNIQUE (ticker, rating_date, analyst_firm);

-- Add index for filtering by source
CREATE INDEX IF NOT EXISTS idx_analyst_source ON analyst_ratings(source);

-- Add comment
COMMENT ON COLUMN analyst_ratings.rating IS 'Analyst rating: Strong Buy, Buy, Hold, Sell, Strong Sell (nullable for price-target-only entries)';
COMMENT ON COLUMN analyst_ratings.source IS 'Data source: FMP, Benzinga, etc.';
