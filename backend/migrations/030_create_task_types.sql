-- Migration 030: Create task_types table and extend worker_tasks

-- Task types table with SOPs
CREATE TABLE IF NOT EXISTS task_types (
    id           SERIAL PRIMARY KEY,
    name         VARCHAR(100) NOT NULL UNIQUE,
    label        VARCHAR(200) NOT NULL,
    sop          TEXT NOT NULL,
    param_schema JSONB,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Update trigger for task_types
CREATE OR REPLACE FUNCTION update_task_types_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_task_types_updated_at
    BEFORE UPDATE ON task_types
    FOR EACH ROW
    EXECUTE FUNCTION update_task_types_updated_at();

-- Extend worker_tasks with task type, params, result, and timestamps
ALTER TABLE worker_tasks
    ADD COLUMN IF NOT EXISTS task_type_id INTEGER REFERENCES task_types(id),
    ADD COLUMN IF NOT EXISTS params       JSONB,
    ADD COLUMN IF NOT EXISTS result       JSONB,
    ADD COLUMN IF NOT EXISTS started_at   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_worker_tasks_task_type ON worker_tasks(task_type_id);

-- Seed initial task types
INSERT INTO task_types (name, label, sop, param_schema) VALUES
('reddit_crawl', 'Reddit Crawl', E'## Reddit Crawl SOP\n\n### Data Source\nUse the Arctic Shift API: https://arctic-shift.photon-reddit.com/api\n- Endpoint: GET /api/posts/search\n- Query params: subreddit, q (search term), after (epoch), before (epoch), limit (max 100)\n- Rate limit: Be respectful — add 1s delay between requests\n\n### What to Collect\nFor each post:\n- external_id (Reddit post ID)\n- subreddit\n- title\n- body (selftext)\n- score (upvotes - downvotes)\n- num_comments\n- author\n- created_utc (ISO 8601)\n- url (permalink)\n\n### How to Store Results\nPOST batches of up to 100 posts to:\n  {api_base}/api/v1/reddit/posts\n\nUse your worker JWT in the Authorization header.\n\n### Task Params You''ll Receive\n- ticker (string) — e.g., "NVDA"\n- subreddits (string[]) — e.g., ["wallstreetbets", "stocks"]\n- days (number) — how many days back to search\n\n### Completion\nReport back with:\n- Total posts collected and stored\n- Actual date range of posts found\n- Any errors or rate limit issues', '{"ticker": "string", "subreddits": "string[]", "days": "number"}'),

('ai_sentiment', 'AI Sentiment Analysis', E'## AI Sentiment Analysis SOP\n\n### What You''re Doing\nAnalyze social media posts about a specific ticker and classify each post''s sentiment as bullish, bearish, or neutral.\n\n### Data Source\nFetch posts from InvestorCenter API:\n  GET {api_base}/api/v1/reddit/posts?ticker={ticker}&limit={limit}\n\n### Analysis\nFor each post, determine:\n- sentiment: "bullish" | "bearish" | "neutral"\n- confidence: 0.0 to 1.0\n- key_phrases: array of phrases that indicate sentiment\n- reasoning: one sentence explaining the classification\n\n### Storing Results\nPOST results to:\n  {api_base}/api/v1/sentiment/batch\n\n### Task Params\n- ticker (string)\n- source (string) — "reddit" or "twitter"\n- limit (number) — max posts to analyze\n\n### Completion\nReport: total posts analyzed, sentiment breakdown, average confidence score.', '{"ticker": "string", "source": "string", "limit": "number"}'),

('sec_filing', 'SEC Filing Crawl', E'## SEC Filing Crawl SOP\n\n### What You''re Doing\nSearch SEC EDGAR for specific filings for a given ticker.\n\n### Data Source\nUse SEC EDGAR full-text search: https://efts.sec.gov/LATEST/search-index?q=\n\n### Task Params\n- ticker (string)\n- filing_type (string) — e.g., "10-K", "10-Q", "8-K"\n\n### Completion\nReport: filings found, date range, key excerpts.', '{"ticker": "string", "filing_type": "string"}'),

('custom', 'Custom Task', 'Follow the task description directly. The description field contains your instructions.', NULL)
ON CONFLICT (name) DO NOTHING;
