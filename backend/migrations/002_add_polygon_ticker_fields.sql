-- Migration to add Polygon API ticker fields
-- This migration adds support for ETFs, crypto, indices, and additional metadata

-- Add new columns to stocks table for Polygon data
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS asset_type VARCHAR(20) DEFAULT 'stock';
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS cik VARCHAR(20);
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS ipo_date DATE;
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS logo_url TEXT;
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS icon_url TEXT;
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS primary_exchange_code VARCHAR(10);
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS composite_figi VARCHAR(20);
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS share_class_figi VARCHAR(20);
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS sic_code VARCHAR(10);
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS sic_description VARCHAR(255);
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS employees INTEGER;
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS phone_number VARCHAR(50);
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS address_city VARCHAR(100);
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS address_state VARCHAR(50);
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS address_postal VARCHAR(20);
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS weighted_shares_outstanding BIGINT;

-- For crypto pairs
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS base_currency_symbol VARCHAR(20);
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS base_currency_name VARCHAR(100);
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS currency_symbol VARCHAR(20);

-- For indices
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS source_feed VARCHAR(100);

-- Add check constraint for asset types
ALTER TABLE stocks DROP CONSTRAINT IF EXISTS check_asset_type;
ALTER TABLE stocks ADD CONSTRAINT check_asset_type 
  CHECK (asset_type IN ('stock', 'etf', 'etn', 'fund', 'preferred', 'warrant', 
                        'right', 'bond', 'adr', 'crypto', 'index', 'other'));

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_stocks_asset_type ON stocks(asset_type);
CREATE INDEX IF NOT EXISTS idx_stocks_cik ON stocks(cik);
CREATE INDEX IF NOT EXISTS idx_stocks_active ON stocks(asset_type, symbol) WHERE asset_type IS NOT NULL;

-- Add a column to track if ticker is active (still trading)
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS active BOOLEAN DEFAULT true;
ALTER TABLE stocks ADD COLUMN IF NOT EXISTS delisted_date DATE;

-- Create index for active tickers
CREATE INDEX IF NOT EXISTS idx_stocks_active_status ON stocks(active);

-- Update existing records to have correct asset_type
-- This is a best-effort attempt based on name patterns
UPDATE stocks 
SET asset_type = 'etf'
WHERE (name ILIKE '%ETF%' OR name ILIKE '%Fund%' OR name ILIKE '%Trust%')
  AND asset_type = 'stock';

-- Add comment to table
COMMENT ON TABLE stocks IS 'Unified table for all tradeable assets: stocks, ETFs, crypto, indices via Polygon.io API';

-- Add comments to new columns
COMMENT ON COLUMN stocks.asset_type IS 'Type of asset: stock, etf, crypto, index, etc.';
COMMENT ON COLUMN stocks.cik IS 'SEC Central Index Key for regulatory filings';
COMMENT ON COLUMN stocks.ipo_date IS 'Initial public offering or listing date';
COMMENT ON COLUMN stocks.primary_exchange_code IS 'Polygon exchange code (e.g., XNAS, XNYS)';
COMMENT ON COLUMN stocks.sic_code IS 'Standard Industrial Classification code';
COMMENT ON COLUMN stocks.base_currency_symbol IS 'For crypto: base currency (e.g., BTC)';
COMMENT ON COLUMN stocks.currency_symbol IS 'For crypto: quote currency (e.g., USD)';
COMMENT ON COLUMN stocks.source_feed IS 'For indices: data source provider';
COMMENT ON COLUMN stocks.active IS 'Whether the ticker is currently actively trading';
COMMENT ON COLUMN stocks.delisted_date IS 'Date when ticker was delisted (if applicable)';