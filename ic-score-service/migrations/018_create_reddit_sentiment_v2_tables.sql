-- Migration: 018_create_reddit_sentiment_v2_tables.sql
-- Description: Create tables for Reddit Sentiment V2 pipeline with AI-powered extraction
-- Date: 2024-12-04

-- Raw Reddit posts (before AI processing)
-- This stores ALL posts from Arctic Shift API without any extraction
CREATE TABLE IF NOT EXISTS reddit_posts_raw (
    id BIGSERIAL PRIMARY KEY,
    external_id VARCHAR(20) UNIQUE NOT NULL,  -- Reddit post ID (e.g., "abc123")
    subreddit VARCHAR(50) NOT NULL,
    author VARCHAR(50),
    title TEXT NOT NULL,
    body TEXT,
    url TEXT NOT NULL,
    upvotes INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    award_count INTEGER DEFAULT 0,
    flair VARCHAR(200),
    posted_at TIMESTAMPTZ NOT NULL,
    fetched_at TIMESTAMPTZ DEFAULT NOW(),

    -- Processing status
    processed_at TIMESTAMPTZ,  -- NULL = not yet processed by AI
    processing_skipped BOOLEAN DEFAULT FALSE,  -- TRUE if skipped due to low engagement

    -- AI extraction results (stored for debugging/reprocessing)
    llm_response JSONB,
    llm_model VARCHAR(50),  -- e.g., "claude-3-haiku-20240307"
    is_finance_related BOOLEAN,  -- From LLM extraction
    spam_score DECIMAL(3,2),  -- 0.00 to 1.00 (pump/dump detection)

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_reddit_posts_raw_unprocessed
    ON reddit_posts_raw(fetched_at)
    WHERE processed_at IS NULL AND processing_skipped = FALSE;

CREATE INDEX IF NOT EXISTS idx_reddit_posts_raw_subreddit
    ON reddit_posts_raw(subreddit, posted_at DESC);

CREATE INDEX IF NOT EXISTS idx_reddit_posts_raw_posted
    ON reddit_posts_raw(posted_at DESC);

CREATE INDEX IF NOT EXISTS idx_reddit_posts_raw_external_id
    ON reddit_posts_raw(external_id);

-- Junction table: posts <-> tickers (many-to-many)
-- One post can mention multiple tickers with different sentiments
CREATE TABLE IF NOT EXISTS reddit_post_tickers (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT REFERENCES reddit_posts_raw(id) ON DELETE CASCADE,
    ticker VARCHAR(10) NOT NULL,

    -- Sentiment per ticker (can be different for each ticker in same post)
    sentiment VARCHAR(10) CHECK (sentiment IN ('bullish', 'bearish', 'neutral')),
    confidence DECIMAL(3,2),  -- 0.00 to 1.00

    -- Context
    is_primary BOOLEAN DEFAULT FALSE,  -- Main ticker being discussed
    mention_type VARCHAR(20),  -- 'ticker' ($AAPL), 'company_name' (Apple), 'abbreviation' (NVDA)

    extracted_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(post_id, ticker)
);

-- Indexes for sentiment queries
CREATE INDEX IF NOT EXISTS idx_reddit_post_tickers_ticker
    ON reddit_post_tickers(ticker, extracted_at DESC);

CREATE INDEX IF NOT EXISTS idx_reddit_post_tickers_sentiment
    ON reddit_post_tickers(ticker, sentiment);

CREATE INDEX IF NOT EXISTS idx_reddit_post_tickers_post_id
    ON reddit_post_tickers(post_id);

CREATE INDEX IF NOT EXISTS idx_reddit_post_tickers_primary
    ON reddit_post_tickers(ticker, is_primary)
    WHERE is_primary = TRUE;

-- Company name cache for reducing LLM calls
-- Stores learned company name -> ticker mappings
CREATE TABLE IF NOT EXISTS company_ticker_cache (
    id SERIAL PRIMARY KEY,
    company_name VARCHAR(100) NOT NULL,  -- Lowercase normalized name
    ticker VARCHAR(10) NOT NULL,
    confidence DECIMAL(3,2) DEFAULT 1.00,
    source VARCHAR(20) DEFAULT 'llm',  -- 'llm', 'manual', 'sec'
    hit_count INTEGER DEFAULT 1,  -- How many times this mapping was used
    last_used_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(company_name, ticker)
);

CREATE INDEX IF NOT EXISTS idx_company_ticker_cache_name
    ON company_ticker_cache(company_name);

-- View to get aggregated sentiment per ticker (for API compatibility)
CREATE OR REPLACE VIEW reddit_ticker_sentiment AS
SELECT
    rpt.ticker,
    COUNT(*) as post_count,
    COUNT(*) FILTER (WHERE rpt.sentiment = 'bullish') as bullish_count,
    COUNT(*) FILTER (WHERE rpt.sentiment = 'bearish') as bearish_count,
    COUNT(*) FILTER (WHERE rpt.sentiment = 'neutral') as neutral_count,
    AVG(rpt.confidence) as avg_confidence,
    MAX(rpr.posted_at) as latest_post_at,
    -- Calculate sentiment score (-100 to +100)
    ROUND(
        (COUNT(*) FILTER (WHERE rpt.sentiment = 'bullish')::DECIMAL -
         COUNT(*) FILTER (WHERE rpt.sentiment = 'bearish')::DECIMAL) /
        NULLIF(COUNT(*), 0) * 100
    , 2) as sentiment_score
FROM reddit_post_tickers rpt
JOIN reddit_posts_raw rpr ON rpt.post_id = rpr.id
WHERE rpr.posted_at > NOW() - INTERVAL '30 days'
GROUP BY rpt.ticker;

-- Grant permissions (adjust as needed for your setup)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON reddit_posts_raw TO investorcenter;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON reddit_post_tickers TO investorcenter;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON company_ticker_cache TO investorcenter;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO investorcenter;

COMMENT ON TABLE reddit_posts_raw IS 'Raw Reddit posts from Arctic Shift API before AI processing';
COMMENT ON TABLE reddit_post_tickers IS 'AI-extracted ticker mentions from Reddit posts (many-to-many)';
COMMENT ON TABLE company_ticker_cache IS 'Cached company name to ticker mappings to reduce LLM calls';
COMMENT ON VIEW reddit_ticker_sentiment IS 'Aggregated sentiment per ticker for API consumption';
