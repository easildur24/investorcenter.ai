-- Migration 021: Add Sentiment Columns to Reddit Heatmap Daily
-- Purpose: Extend existing heatmap table with sentiment breakdown metrics
-- Date: 2025-11-30

-- Add sentiment columns to existing table
ALTER TABLE reddit_heatmap_daily
ADD COLUMN IF NOT EXISTS sentiment_score DECIMAL(5,2),  -- -100 to +100
ADD COLUMN IF NOT EXISTS bullish_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS bearish_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS neutral_count INTEGER DEFAULT 0;

-- Add index for sentiment queries
CREATE INDEX IF NOT EXISTS idx_reddit_heatmap_sentiment
    ON reddit_heatmap_daily(date, sentiment_score DESC);

-- Add comments for documentation
COMMENT ON COLUMN reddit_heatmap_daily.sentiment_score IS 'Aggregate sentiment score from -100 (very bearish) to +100 (very bullish)';
COMMENT ON COLUMN reddit_heatmap_daily.bullish_count IS 'Number of posts classified as bullish';
COMMENT ON COLUMN reddit_heatmap_daily.bearish_count IS 'Number of posts classified as bearish';
COMMENT ON COLUMN reddit_heatmap_daily.neutral_count IS 'Number of posts classified as neutral';
