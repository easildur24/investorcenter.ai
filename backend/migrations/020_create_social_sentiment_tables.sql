-- Migration 020: Create Social Sentiment Tables
-- Purpose: Support sentiment analysis and representative posts for social media data
-- Date: 2025-11-30

-- Table: social_posts
-- Stores representative posts for the sentiment feature
-- NO author field - privacy decision
CREATE TABLE IF NOT EXISTS social_posts (
    id BIGSERIAL PRIMARY KEY,
    external_post_id VARCHAR(50) UNIQUE NOT NULL,
    source VARCHAR(20) NOT NULL DEFAULT 'reddit',
    ticker VARCHAR(10) NOT NULL,
    subreddit VARCHAR(50) NOT NULL,
    title TEXT NOT NULL,
    body_preview VARCHAR(500),
    url TEXT NOT NULL,
    -- NO author field - privacy decision
    upvotes INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    award_count INTEGER DEFAULT 0,
    sentiment VARCHAR(10) CHECK (sentiment IN ('bullish', 'bearish', 'neutral')),
    sentiment_confidence DECIMAL(3,2),
    flair VARCHAR(50),
    posted_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for user-selectable sort options
CREATE INDEX IF NOT EXISTS idx_social_posts_ticker_recent
    ON social_posts(ticker, posted_at DESC);
CREATE INDEX IF NOT EXISTS idx_social_posts_ticker_engagement
    ON social_posts(ticker, (upvotes + comment_count * 2 + award_count * 5) DESC);
CREATE INDEX IF NOT EXISTS idx_social_posts_ticker_sentiment
    ON social_posts(ticker, sentiment, sentiment_confidence DESC);
CREATE INDEX IF NOT EXISTS idx_social_posts_subreddit
    ON social_posts(subreddit, posted_at DESC);
CREATE INDEX IF NOT EXISTS idx_social_posts_source
    ON social_posts(source, posted_at DESC);

-- Table: sentiment_lexicon
-- WSB/Reddit financial slang for sentiment analysis
CREATE TABLE IF NOT EXISTS sentiment_lexicon (
    id SERIAL PRIMARY KEY,
    term VARCHAR(100) UNIQUE NOT NULL,
    sentiment VARCHAR(10) CHECK (sentiment IN ('bullish', 'bearish', 'modifier')),
    weight DECIMAL(3,2) DEFAULT 1.00,
    category VARCHAR(50),  -- 'slang', 'options', 'emoji', 'modifier', 'action', 'position', 'analysis', 'direct'
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for term lookups
CREATE INDEX IF NOT EXISTS idx_sentiment_lexicon_term
    ON sentiment_lexicon(LOWER(term));
CREATE INDEX IF NOT EXISTS idx_sentiment_lexicon_sentiment
    ON sentiment_lexicon(sentiment);

-- Table: social_data_sources (for future platform extension)
CREATE TABLE IF NOT EXISTS social_data_sources (
    id SERIAL PRIMARY KEY,
    source_name VARCHAR(50) UNIQUE NOT NULL,
    is_enabled BOOLEAN DEFAULT true,
    config JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default Reddit source
INSERT INTO social_data_sources (source_name, is_enabled, config)
VALUES ('reddit', true, '{"subreddits": ["wallstreetbets", "stocks", "options", "investing", "Daytrading"]}')
ON CONFLICT (source_name) DO NOTHING;

-- Add comments for documentation
COMMENT ON TABLE social_posts IS 'Representative social media posts for sentiment analysis';
COMMENT ON TABLE sentiment_lexicon IS 'WSB/Reddit financial slang terms for sentiment scoring';
COMMENT ON TABLE social_data_sources IS 'Configurable social media data sources (extensible for future platforms)';
COMMENT ON COLUMN social_posts.sentiment IS 'Computed sentiment: bullish, bearish, or neutral';
COMMENT ON COLUMN social_posts.sentiment_confidence IS 'Confidence score 0.00-1.00 for the sentiment classification';
COMMENT ON COLUMN sentiment_lexicon.weight IS 'Multiplier for sentiment scoring (higher = stronger signal)';
COMMENT ON COLUMN sentiment_lexicon.category IS 'Term category: slang, options, emoji, modifier, action, position, analysis, direct';
