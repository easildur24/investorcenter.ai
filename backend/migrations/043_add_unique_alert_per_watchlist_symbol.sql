-- Migration 043: Add unique constraint on alert_rules(watch_list_id, symbol)
-- Date: 2026-02-25
-- Description: Prevents duplicate alerts for the same symbol within a watchlist.
--   The application pre-check (AlertExistsForSymbol) is not atomic — two concurrent
--   POSTs can both pass the existence check before either inserts. This unique index
--   is the real guard; the pre-check remains as an early-out for the common case.

-- Remove any legacy duplicates first (keep the newest alert per watch_list_id + symbol)
DELETE FROM alert_rules
WHERE id NOT IN (
    SELECT DISTINCT ON (watch_list_id, symbol) id
    FROM alert_rules
    ORDER BY watch_list_id, symbol, created_at DESC
);

-- Add the unique index
CREATE UNIQUE INDEX IF NOT EXISTS idx_alert_rules_watchlist_symbol_unique
    ON alert_rules (watch_list_id, symbol);
