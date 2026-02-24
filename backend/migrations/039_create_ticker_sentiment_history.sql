-- Migration 039: Create ticker_sentiment_history hypertable (TimescaleDB)
-- Purpose: Time-series storage for historical sentiment charts. One row per ticker
--          per snapshot cycle (~15 min). Uses TimescaleDB for automatic partitioning,
--          compression, and efficient time-range queries.
-- Date: 2026-02-23
-- Phase: 1b (Database Schema Evolution)
--
-- Prerequisite: TimescaleDB extension must be enabled (already done in ic-score-service schema).
-- If running against a fresh DB: CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS ticker_sentiment_history (
    time                TIMESTAMPTZ NOT NULL,
    ticker              VARCHAR(10) NOT NULL,
    sentiment_score     FLOAT NOT NULL,             -- -1.0 to +1.0
    bullish_pct         FLOAT NOT NULL,             -- 0.0 to 1.0
    mention_count       INTEGER NOT NULL,
    composite_score     FLOAT NOT NULL
);

-- Convert to TimescaleDB hypertable (partition by 1-day chunks)
-- Using IF NOT EXISTS so the migration is idempotent
SELECT create_hypertable(
    'ticker_sentiment_history',
    'time',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Primary query: "give me AAPL's sentiment over the last 30 days"
CREATE INDEX IF NOT EXISTS idx_sentiment_history_ticker
    ON ticker_sentiment_history (ticker, time DESC);

-- Unique constraint: one row per (ticker, time). Prevents duplicate inserts
-- when the pipeline retries or runs twice in the same cycle. The Go layer
-- uses ON CONFLICT (ticker, time) DO NOTHING to tolerate duplicates.
CREATE UNIQUE INDEX IF NOT EXISTS idx_sentiment_history_ticker_time_unique
    ON ticker_sentiment_history (ticker, time);

-- Retention policy: automatically drop data older than 12 months
-- TimescaleDB handles this via background worker
SELECT add_retention_policy(
    'ticker_sentiment_history',
    INTERVAL '12 months',
    if_not_exists => TRUE
);

-- Compression policy: compress chunks older than 7 days for storage efficiency.
-- Compressed data is still queryable but uses significantly less disk space.
-- NOTE: ALTER TABLE SET is idempotent — re-running it with the same settings
-- is a no-op in TimescaleDB. The add_compression_policy below uses
-- if_not_exists to avoid errors on re-run.
ALTER TABLE ticker_sentiment_history SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'ticker',
    timescaledb.compress_orderby = 'time DESC'
);

SELECT add_compression_policy(
    'ticker_sentiment_history',
    INTERVAL '7 days',
    if_not_exists => TRUE
);

-- Comments
COMMENT ON TABLE ticker_sentiment_history IS 'Time-series sentiment data (TimescaleDB hypertable). One row per ticker per ~15min snapshot. Powers historical sentiment charts. Auto-compressed after 7 days, retained for 12 months.';
COMMENT ON COLUMN ticker_sentiment_history.sentiment_score IS 'Weighted average sentiment: -1.0 (very bearish) to +1.0 (very bullish)';
COMMENT ON COLUMN ticker_sentiment_history.composite_score IS 'Same composite_score as in ticker_sentiment_snapshots, logged here for historical charting';
