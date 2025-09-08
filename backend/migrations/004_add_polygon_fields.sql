-- Add Polygon.io specific fields to tickers table
-- These fields are needed for the incremental update from Polygon API
-- Note: Some fields may already exist from migration 002, using IF NOT EXISTS to handle both cases

-- Add new columns for Polygon data (only those not in migration 002)
ALTER TABLE tickers 
ADD COLUMN IF NOT EXISTS locale VARCHAR(10) DEFAULT 'us',
ADD COLUMN IF NOT EXISTS market VARCHAR(50) DEFAULT 'stocks',
ADD COLUMN IF NOT EXISTS currency_name VARCHAR(50),
ADD COLUMN IF NOT EXISTS polygon_type VARCHAR(50),
ADD COLUMN IF NOT EXISTS last_updated_utc TIMESTAMP WITH TIME ZONE;

-- Create index on last_updated_utc for efficient incremental updates
CREATE INDEX IF NOT EXISTS idx_tickers_last_updated_utc ON tickers(last_updated_utc);

-- Create index on asset_type for filtering by asset type
CREATE INDEX IF NOT EXISTS idx_tickers_asset_type ON tickers(asset_type);

-- Create index on active status
CREATE INDEX IF NOT EXISTS idx_tickers_active ON tickers(active);

-- Update existing records to have a default last_updated_utc if null
UPDATE tickers 
SET last_updated_utc = COALESCE(last_updated_utc, updated_at, created_at, NOW())
WHERE last_updated_utc IS NULL;