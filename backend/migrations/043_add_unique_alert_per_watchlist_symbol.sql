-- Migration 043: Add unique constraint on alert_rules(watch_list_id, symbol)
-- Date: 2026-02-25
-- Description: Prevents duplicate alerts for the same symbol within a watchlist.
--   The unique index is the authoritative guard; the application relies solely on the
--   DB constraint + ErrAlertAlreadyExists error mapping (no app-layer pre-check).
--
-- OPERATOR NOTE: Before running in production, preview affected rows with:
--   SELECT id, watch_list_id, symbol, created_at FROM alert_rules
--   WHERE (watch_list_id, symbol) IN (
--     SELECT watch_list_id, symbol FROM alert_rules
--     GROUP BY watch_list_id, symbol HAVING COUNT(*) > 1
--   ) ORDER BY watch_list_id, symbol, created_at DESC;
--
-- Consider backing up the alert_rules table first:
--   CREATE TABLE alert_rules_backup_043 AS SELECT * FROM alert_rules;

BEGIN;

-- Remove any legacy duplicates (keep the newest alert per watch_list_id + symbol)
DELETE FROM alert_rules
WHERE id NOT IN (
    SELECT DISTINCT ON (watch_list_id, symbol) id
    FROM alert_rules
    ORDER BY watch_list_id, symbol, created_at DESC
);

-- Add the unique index
CREATE UNIQUE INDEX IF NOT EXISTS idx_alert_rules_watchlist_symbol_unique
    ON alert_rules (watch_list_id, symbol);

COMMIT;
