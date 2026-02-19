-- Migration: Add composite indexes for watchlist LATERAL JOIN performance
--
-- The GetWatchListItemsWithData query uses LATERAL JOINs that execute per-row.
-- Without proper composite indexes, PostgreSQL falls back to sequential scans
-- on each correlated subquery.

-- alert_rules: composite index for the per-item active alert count LATERAL JOIN.
-- Existing single-column indexes (watch_list_id, symbol, is_active) don't cover
-- the combined WHERE clause efficiently.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alert_rules_watchlist_symbol_active
    ON alert_rules (watch_list_id, symbol, is_active)
    WHERE is_active = true;

-- reddit_ticker_rankings: the LATERAL JOIN orders by snapshot_time DESC, but the
-- existing idx_reddit_rankings_ticker_date uses (ticker_symbol, snapshot_date DESC).
-- This index covers the actual ORDER BY column for optimal index-only scans.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_reddit_rankings_ticker_snapshot_time
    ON reddit_ticker_rankings (ticker_symbol, snapshot_time DESC);
