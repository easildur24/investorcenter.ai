-- Migration 038: Create ticker_sentiment_snapshots table
-- Purpose: Pre-computed, read-optimized aggregation of sentiment per ticker per time range.
--          Refreshed every ~15 minutes by a pipeline. The API reads from this table directly
--          instead of aggregating reddit_heatmap_daily at query time.
-- Date: 2026-02-23
-- Phase: 1b (Database Schema Evolution)

CREATE TABLE IF NOT EXISTS ticker_sentiment_snapshots (
    id                      BIGSERIAL PRIMARY KEY,
    ticker                  VARCHAR(10) NOT NULL,
    snapshot_time           TIMESTAMPTZ NOT NULL,
    time_range              VARCHAR(10) NOT NULL,       -- '1d', '7d', '14d', '30d'

    -- Mention metrics
    mention_count           INTEGER NOT NULL DEFAULT 0,
    total_upvotes           INTEGER NOT NULL DEFAULT 0,
    total_comments          INTEGER NOT NULL DEFAULT 0,
    unique_posts            INTEGER NOT NULL DEFAULT 0,

    -- Sentiment breakdown
    bullish_count           INTEGER NOT NULL DEFAULT 0,
    neutral_count           INTEGER NOT NULL DEFAULT 0,
    bearish_count           INTEGER NOT NULL DEFAULT 0,
    bullish_pct             FLOAT NOT NULL DEFAULT 0,
    neutral_pct             FLOAT NOT NULL DEFAULT 0,
    bearish_pct             FLOAT NOT NULL DEFAULT 0,
    sentiment_score         FLOAT NOT NULL DEFAULT 0,   -- weighted avg: -1.0 to +1.0
    sentiment_label         VARCHAR(10) NOT NULL DEFAULT 'neutral',  -- derived: 'bullish'/'bearish'/'neutral'

    -- Velocity metrics (how fast sentiment/mentions are changing)
    mention_velocity_1h     FLOAT,                      -- mentions per hour (last 1h vs prior 1h)
    sentiment_velocity_24h  FLOAT,                      -- sentiment delta over past 24h

    -- Composite score: non-circular formula incorporating sentiment
    -- composite_score = w1*mentions + w2*upvotes + w3*comments + w4*sentiment_score
    composite_score         FLOAT NOT NULL DEFAULT 0,

    -- Subreddit distribution (JSONB for flexible schema across different subreddit combos)
    -- Example: {"wallstreetbets": 45, "stocks": 30, "investing": 20, "options": 5}
    subreddit_distribution  JSONB,

    -- Ranking (pre-computed — avoids floating-point rank issues from BUG-001)
    rank                    INTEGER,
    previous_rank           INTEGER,
    rank_change             INTEGER,                    -- always integer: previous_rank - rank

    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_sentiment_snapshot UNIQUE (ticker, snapshot_time, time_range),
    CONSTRAINT chk_time_range CHECK (time_range IN ('1d', '7d', '14d', '30d')),
    CONSTRAINT chk_sentiment_label CHECK (sentiment_label IN ('bullish', 'bearish', 'neutral'))
);

-- Primary read path: "give me the latest snapshot for ticker X in time range Y"
CREATE INDEX IF NOT EXISTS idx_sentiment_snapshots_ticker
    ON ticker_sentiment_snapshots (ticker, time_range, snapshot_time DESC);

-- Trending list: "give me the top N tickers by rank for the 7d time range"
CREATE INDEX IF NOT EXISTS idx_sentiment_snapshots_rank
    ON ticker_sentiment_snapshots (time_range, snapshot_time DESC, rank ASC);

-- Composite score sorting (alternative to rank-based sorting)
CREATE INDEX IF NOT EXISTS idx_sentiment_snapshots_composite
    ON ticker_sentiment_snapshots (time_range, snapshot_time DESC, composite_score DESC);

-- Comments
COMMENT ON TABLE ticker_sentiment_snapshots IS 'Pre-computed sentiment aggregation per ticker per time range. Refreshed every ~15 min by pipeline. Primary read table for trending/sentiment APIs.';
COMMENT ON COLUMN ticker_sentiment_snapshots.composite_score IS 'Non-circular score: weighted(mentions, upvotes, comments, sentiment). Replaces popularity_score which had circular rank dependency.';
COMMENT ON COLUMN ticker_sentiment_snapshots.rank_change IS 'Integer rank change (previous_rank - rank). Positive = improved. Stored as INTEGER to prevent floating-point display bugs.';
COMMENT ON COLUMN ticker_sentiment_snapshots.subreddit_distribution IS 'JSONB map of subreddit_name → mention_count. Flexible schema allows adding subreddits without migration.';
COMMENT ON COLUMN ticker_sentiment_snapshots.mention_velocity_1h IS 'Rate of change in mentions: (mentions_last_1h - mentions_prior_1h) / mentions_prior_1h. NULL if no prior data.';
COMMENT ON COLUMN ticker_sentiment_snapshots.sentiment_velocity_24h IS 'Sentiment score delta over the past 24 hours. Positive = trending more bullish.';
