-- Migration 041: Add bearish_pct and neutral_pct to ticker_sentiment_history
--
-- The history table only had bullish_pct. The API history endpoint needs
-- all three breakdown percentages to return accurate bullish/bearish/neutral
-- counts. Adding these columns avoids lossy approximations.

ALTER TABLE ticker_sentiment_history
  ADD COLUMN IF NOT EXISTS bearish_pct FLOAT,
  ADD COLUMN IF NOT EXISTS neutral_pct FLOAT;
