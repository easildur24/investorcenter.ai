-- Migration to allow same symbol for different asset types (e.g., META stock and META crypto)
-- Changes unique constraint from (symbol) to (symbol, asset_type)

-- Step 1: Drop foreign key constraints that reference tickers(symbol)
-- Use CASCADE to handle dependent objects

DO $$
BEGIN
    -- Drop FK constraints if they exist (ignore errors for non-existent tables)
    BEGIN ALTER TABLE stock_prices DROP CONSTRAINT IF EXISTS stock_prices_symbol_fkey; EXCEPTION WHEN undefined_table THEN NULL; END;
    BEGIN ALTER TABLE fundamentals DROP CONSTRAINT IF EXISTS fundamentals_symbol_fkey; EXCEPTION WHEN undefined_table THEN NULL; END;
    BEGIN ALTER TABLE dividends DROP CONSTRAINT IF EXISTS dividends_symbol_fkey; EXCEPTION WHEN undefined_table THEN NULL; END;
    BEGIN ALTER TABLE earnings DROP CONSTRAINT IF EXISTS earnings_symbol_fkey; EXCEPTION WHEN undefined_table THEN NULL; END;
    BEGIN ALTER TABLE insider_trades DROP CONSTRAINT IF EXISTS insider_trades_symbol_fkey; EXCEPTION WHEN undefined_table THEN NULL; END;
    BEGIN ALTER TABLE analyst_ratings DROP CONSTRAINT IF EXISTS analyst_ratings_symbol_fkey; EXCEPTION WHEN undefined_table THEN NULL; END;
    BEGIN ALTER TABLE sec_filings DROP CONSTRAINT IF EXISTS sec_filings_symbol_fkey; EXCEPTION WHEN undefined_table THEN NULL; END;
    BEGIN ALTER TABLE reddit_ticker_mentions DROP CONSTRAINT IF EXISTS reddit_ticker_mentions_ticker_symbol_fkey; EXCEPTION WHEN undefined_table THEN NULL; END;
    BEGIN ALTER TABLE reddit_mention_heatmap DROP CONSTRAINT IF EXISTS reddit_mention_heatmap_ticker_symbol_fkey; EXCEPTION WHEN undefined_table THEN NULL; END;
    BEGIN ALTER TABLE reddit_ticker_rankings DROP CONSTRAINT IF EXISTS reddit_ticker_rankings_ticker_symbol_fkey; EXCEPTION WHEN undefined_table THEN NULL; END;
    BEGIN ALTER TABLE reddit_heatmap_daily DROP CONSTRAINT IF EXISTS reddit_heatmap_daily_ticker_symbol_fkey; EXCEPTION WHEN undefined_table THEN NULL; END;
END $$;

-- Step 2: Drop the old unique constraint on symbol (use CASCADE for dependent objects)
ALTER TABLE tickers DROP CONSTRAINT IF EXISTS stocks_symbol_key CASCADE;
ALTER TABLE tickers DROP CONSTRAINT IF EXISTS tickers_symbol_key CASCADE;

-- Step 3: Create new composite unique constraint on (symbol, asset_type)
-- First ensure asset_type has a default value
ALTER TABLE tickers ALTER COLUMN asset_type SET DEFAULT 'stock';
UPDATE tickers SET asset_type = 'stock' WHERE asset_type IS NULL OR asset_type = '';

-- Add the new composite unique constraint (if it doesn't exist)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'tickers_symbol_asset_type_key'
    ) THEN
        ALTER TABLE tickers ADD CONSTRAINT tickers_symbol_asset_type_key UNIQUE (symbol, asset_type);
    END IF;
END $$;

-- Step 4: Create an index on symbol for fast lookups (now non-unique)
CREATE INDEX IF NOT EXISTS idx_tickers_symbol ON tickers(symbol);

-- Step 5: Insert Meta Platforms if it doesn't exist (fix current issue)
INSERT INTO tickers (symbol, name, exchange, asset_type, sector, industry, country, currency)
SELECT 'META', 'Meta Platforms, Inc.', 'NASDAQ', 'stock', 'Technology', 'Internet Content & Information', 'US', 'USD'
WHERE NOT EXISTS (
    SELECT 1 FROM tickers WHERE symbol = 'META' AND asset_type = 'stock'
);

-- Verify both META entries exist
-- SELECT symbol, name, asset_type, exchange FROM tickers WHERE symbol = 'META';