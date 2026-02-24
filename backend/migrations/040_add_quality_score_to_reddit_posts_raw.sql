-- Migration 040: Add quality_score column to reddit_posts_raw
-- Purpose: Store LLM-assessed post quality score (0.0-1.0) for investment
--          sentiment analysis. Used by the aggregation step to compute
--          ticker-level composite scores. High quality = DD/analysis posts,
--          low quality = memes/hype.
-- Date: 2026-02-24
-- Phase: 1c (Combined Reddit Sentiment Pipeline)

ALTER TABLE reddit_posts_raw
    ADD COLUMN IF NOT EXISTS quality_score FLOAT;

COMMENT ON COLUMN reddit_posts_raw.quality_score IS
    'LLM-assessed quality score (0.0-1.0) for investment sentiment analysis usefulness. High (0.7-1.0): original analysis, DD posts. Low (0.0-0.3): memes, hype.';
