-- Migration 030: Create task_types table and extend worker_tasks

-- Task types table with SOPs
CREATE TABLE IF NOT EXISTS task_types (
    id           SERIAL PRIMARY KEY,
    name         VARCHAR(100) NOT NULL UNIQUE,
    label        VARCHAR(200) NOT NULL,
    sop          TEXT NOT NULL DEFAULT '',
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
('reddit_crawl', 'Reddit Crawl', E'## Reddit Crawl SOP\n\n### Data Source\nUse the Arctic Shift API: https://arctic-shift.photon-reddit.com/api\n- Endpoint: GET /api/posts/search\n- Query params: subreddit, q (search term), after (epoch), before (epoch), limit (max 100)\n- Rate limit: Be respectful — add 1s delay between requests\n\n### What to Collect\nFor each post:\n- external_id (Reddit post ID)\n- subreddit\n- title\n- body (selftext)\n- score (upvotes - downvotes)\n- num_comments\n- author\n- created_utc (ISO 8601)\n- url (permalink)\n\n### How to Store Results\nAs you collect posts, POST batches of up to 100 items to:\n  POST {api_base}/api/v1/worker/tasks/{task_id}/data\n\nRequest body:\n```json\n{\n  "data_type": "reddit_post",\n  "items": [\n    {\n      "ticker": "NVDA",\n      "external_id": "reddit_post_id_here",\n      "collected_at": "2026-01-15T14:30:00Z",\n      "data": {\n        "subreddit": "stocks",\n        "title": "...",\n        "body": "...",\n        "score": 42,\n        "num_comments": 15,\n        "author": "user123",\n        "url": "/r/stocks/comments/...",\n        "created_utc": "2026-01-15T14:30:00Z"\n      }\n    }\n  ]\n}\n```\n\nUse your worker JWT in the Authorization header.\nDuplicates (same data_type + external_id) are automatically skipped.\nPost data during execution, not just at the end — partial progress survives timeouts.\n\n### Task Params You''ll Receive\n- ticker (string) — e.g., "NVDA"\n- subreddits (string[]) — e.g., ["wallstreetbets", "stocks"]\n- days (number) — how many days back to search\n\n### Completion\nPost your final result, then mark the task as completed:\n1. POST {api_base}/api/v1/worker/tasks/{task_id}/result with summary:\n   {"posts_collected": 42, "date_range": "2026-01-10 to 2026-01-15", "subreddits_searched": ["stocks", "wallstreetbets"]}\n2. PUT {api_base}/api/v1/worker/tasks/{task_id}/status with {"status": "completed"}', '{"ticker": "string", "subreddits": "string[]", "days": "number"}'),

('ai_sentiment', 'AI Sentiment Analysis', E'## AI Sentiment Analysis SOP\n\n### What You''re Doing\nAnalyze social media posts about a specific ticker and classify each post''s sentiment as bullish, bearish, or neutral.\n\n### Data Source\nFetch collected posts from this task''s data:\n  GET {api_base}/api/v1/worker/tasks/{task_id}/data?data_type=reddit_post&ticker={ticker}\n\nOr if analyzing posts from a different source, fetch from the task params.\n\n### Analysis\nFor each post, determine:\n- sentiment: "bullish" | "bearish" | "neutral"\n- confidence: 0.0 to 1.0\n- key_phrases: array of phrases that indicate sentiment\n- reasoning: one sentence explaining the classification\n\n### Storing Results\nPOST sentiment results in batches to:\n  POST {api_base}/api/v1/worker/tasks/{task_id}/data\n\nRequest body:\n```json\n{\n  "data_type": "sentiment_analysis",\n  "items": [\n    {\n      "ticker": "NVDA",\n      "external_id": "sentiment_reddit_post_id",\n      "data": {\n        "source_post_id": "reddit_post_id",\n        "sentiment": "bullish",\n        "confidence": 0.85,\n        "key_phrases": ["strong earnings", "beat expectations"],\n        "reasoning": "Post discusses positive earnings surprise"\n      }\n    }\n  ]\n}\n```\n\nDuplicates (same data_type + external_id) are automatically skipped.\n\n### Task Params\n- ticker (string)\n- source (string) — "reddit" or "twitter"\n- limit (number) — max posts to analyze\n\n### Completion\nPost your final result, then mark the task as completed:\n1. POST {api_base}/api/v1/worker/tasks/{task_id}/result with summary:\n   {"posts_analyzed": 50, "bullish": 25, "bearish": 15, "neutral": 10, "avg_confidence": 0.78}\n2. PUT {api_base}/api/v1/worker/tasks/{task_id}/status with {"status": "completed"}', '{"ticker": "string", "source": "string", "limit": "number"}'),

('sec_filing', 'SEC Filing Crawl', E'## SEC Filing Crawl SOP\n\n### What You''re Doing\nSearch SEC EDGAR for specific filings for a given ticker.\n\n### Data Source\nUse SEC EDGAR full-text search: https://efts.sec.gov/LATEST/search-index?q=\nAlso use the EDGAR company search API: https://www.sec.gov/cgi-bin/browse-edgar\n\n### How to Store Results\nPOST collected filings in batches to:\n  POST {api_base}/api/v1/worker/tasks/{task_id}/data\n\nRequest body:\n```json\n{\n  "data_type": "sec_filing",\n  "items": [\n    {\n      "ticker": "AAPL",\n      "external_id": "0000320193-24-000081",\n      "collected_at": "2024-11-01T00:00:00Z",\n      "data": {\n        "filing_type": "10-K",\n        "filed_date": "2024-11-01",\n        "accession_number": "0000320193-24-000081",\n        "document_url": "https://www.sec.gov/Archives/...",\n        "description": "Annual report",\n        "key_excerpts": ["..."]\n      }\n    }\n  ]\n}\n```\n\nDuplicates (same data_type + external_id) are automatically skipped.\n\n### Task Params\n- ticker (string)\n- filing_type (string) — e.g., "10-K", "10-Q", "8-K"\n\n### Completion\nPost your final result, then mark the task as completed:\n1. POST {api_base}/api/v1/worker/tasks/{task_id}/result with summary:\n   {"filings_found": 5, "date_range": "2023-01 to 2024-11", "filing_types": ["10-K", "10-Q"]}\n2. PUT {api_base}/api/v1/worker/tasks/{task_id}/status with {"status": "completed"}', '{"ticker": "string", "filing_type": "string"}'),

('custom', 'Custom Task', E'## Custom Task SOP\n\nFollow the task description directly. The description field contains your instructions.\n\n### Storing Collected Data\nIf you collect any data during this task, POST it in batches to:\n  POST {api_base}/api/v1/worker/tasks/{task_id}/data\n\nRequest body:\n```json\n{\n  "data_type": "custom_data",\n  "items": [\n    {\n      "external_id": "unique_id_for_dedup",\n      "data": { "your": "data here" }\n    }\n  ]\n}\n```\n\n### Completion\n1. POST {api_base}/api/v1/worker/tasks/{task_id}/result with a summary of what you accomplished\n2. PUT {api_base}/api/v1/worker/tasks/{task_id}/status with {"status": "completed"}', NULL)
ON CONFLICT (name) DO NOTHING;
