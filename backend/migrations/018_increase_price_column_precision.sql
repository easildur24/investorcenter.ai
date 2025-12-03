-- Migration: Increase stock_prices column precision for high-priced stocks
-- Date: 2025-11-29
-- Reason: BRK.A and other high-priced stocks (>$100,000) overflow NUMERIC(10,2)

ALTER TABLE stock_prices ALTER COLUMN open TYPE NUMERIC(15,2);
ALTER TABLE stock_prices ALTER COLUMN high TYPE NUMERIC(15,2);
ALTER TABLE stock_prices ALTER COLUMN low TYPE NUMERIC(15,2);
ALTER TABLE stock_prices ALTER COLUMN close TYPE NUMERIC(15,2);
ALTER TABLE stock_prices ALTER COLUMN vwap TYPE NUMERIC(15,2);

-- Verify the change
-- \d stock_prices
