-- Migration to rename stocks table to tickers for multi-asset support
-- This makes the table name more generic since it now holds ETFs, crypto, indices, etc.

ALTER TABLE IF EXISTS stocks RENAME TO tickers;

-- Update any indexes that reference the old table name
ALTER INDEX IF EXISTS idx_stocks_asset_type RENAME TO idx_tickers_asset_type;
ALTER INDEX IF EXISTS idx_stocks_cik RENAME TO idx_tickers_cik;
ALTER INDEX IF EXISTS idx_stocks_active RENAME TO idx_tickers_active;
ALTER INDEX IF EXISTS idx_stocks_active_status RENAME TO idx_tickers_active_status;

-- Update the table comment
COMMENT ON TABLE tickers IS 'Unified table for all tradeable assets: stocks, ETFs, crypto, indices via Polygon.io API';