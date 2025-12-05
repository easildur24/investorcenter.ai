# Reddit Sentiment Pipeline V2 - Technical Specification

## Overview

Redesign the Reddit sentiment pipeline to use AI-powered extraction and support multiple tickers per post. This replaces the regex-based approach with an agentic LLM solution for better accuracy.

## Current Architecture (V1)

```
Arctic Shift API → Regex Extraction → Keyword Sentiment → social_posts (1 ticker/post)
```

**Problems:**
- Regex misses company names ("Oracle" → ORCL)
- Only stores 1 ticker per post
- Keyword-based sentiment is context-blind
- Can't handle "bullish AAPL, bearish GOOGL" in same post

## Proposed Architecture (V2)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           INGESTION LAYER                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   Arctic Shift API  ──►  reddit_posts_raw (store all posts)                 │
│   (30 subreddits)        - No processing, just raw storage                  │
│   (hourly)               - Deduplication by external_id                     │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         PROCESSING LAYER (AI-Powered)                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐        │
│   │  Batch Selector │───►│  LLM Processor  │───►│  Result Writer  │        │
│   │  (unprocessed   │    │  (Claude Haiku) │    │  (to DB)        │        │
│   │   posts)        │    │                 │    │                 │        │
│   └─────────────────┘    └─────────────────┘    └─────────────────┘        │
│                                                                              │
│   LLM extracts:                                                              │
│   - All tickers mentioned (with confidence)                                  │
│   - Company name → ticker mapping                                            │
│   - Per-ticker sentiment (bullish/bearish/neutral)                          │
│   - Overall post relevance score                                             │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           STORAGE LAYER                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   reddit_posts_raw          reddit_post_tickers         social_posts        │
│   ├── id                    ├── post_id (FK)           (existing, for API)  │
│   ├── external_id           ├── ticker                                      │
│   ├── subreddit             ├── sentiment                                   │
│   ├── title                 ├── confidence                                  │
│   ├── body                  ├── is_primary                                  │
│   ├── url                   └── extracted_at                                │
│   ├── author                                                                 │
│   ├── upvotes                                                                │
│   ├── comments                                                               │
│   ├── awards                                                                 │
│   ├── flair                                                                  │
│   ├── posted_at                                                              │
│   ├── fetched_at                                                             │
│   ├── processed_at (null = unprocessed)                                      │
│   └── llm_response (JSON, for debugging)                                    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Database Schema

### New Tables

```sql
-- Raw Reddit posts (before AI processing)
CREATE TABLE reddit_posts_raw (
    id BIGSERIAL PRIMARY KEY,
    external_id VARCHAR(20) UNIQUE NOT NULL,  -- Reddit post ID
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
    processed_at TIMESTAMPTZ,  -- NULL = not yet processed
    llm_response JSONB,  -- Store full LLM response for debugging
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient querying
CREATE INDEX idx_reddit_posts_raw_unprocessed
    ON reddit_posts_raw(fetched_at) WHERE processed_at IS NULL;
CREATE INDEX idx_reddit_posts_raw_subreddit
    ON reddit_posts_raw(subreddit, posted_at DESC);
CREATE INDEX idx_reddit_posts_raw_posted
    ON reddit_posts_raw(posted_at DESC);

-- Junction table: posts ↔ tickers (many-to-many)
CREATE TABLE reddit_post_tickers (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT REFERENCES reddit_posts_raw(id) ON DELETE CASCADE,
    ticker VARCHAR(10) NOT NULL,
    sentiment VARCHAR(10) CHECK (sentiment IN ('bullish', 'bearish', 'neutral')),
    confidence DECIMAL(3,2),  -- 0.00 to 1.00
    is_primary BOOLEAN DEFAULT FALSE,  -- Main ticker discussed
    mention_type VARCHAR(20),  -- 'ticker', 'company_name', 'abbreviation'
    extracted_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(post_id, ticker)
);

-- Indexes for sentiment queries
CREATE INDEX idx_reddit_post_tickers_ticker
    ON reddit_post_tickers(ticker, extracted_at DESC);
CREATE INDEX idx_reddit_post_tickers_sentiment
    ON reddit_post_tickers(ticker, sentiment);
```

### Migration for Existing Data

```sql
-- Keep social_posts for backward compatibility
-- New API endpoints will use new tables
-- Eventually deprecate social_posts
```

## LLM Processing Pipeline

### Prompt Template

```python
EXTRACTION_PROMPT = """
Analyze this Reddit post from r/{subreddit} and extract stock/ETF mentions.

Title: {title}
Body: {body}
Flair: {flair}

Return a JSON object with:
{{
  "tickers": [
    {{
      "symbol": "AAPL",           // Ticker symbol (uppercase)
      "sentiment": "bullish",      // bullish, bearish, or neutral
      "confidence": 0.85,          // 0.0 to 1.0
      "is_primary": true,          // Is this the main topic?
      "mention_type": "ticker"     // ticker, company_name, or abbreviation
    }}
  ],
  "is_finance_related": true,      // Is this post about investing/trading?
  "spam_score": 0.1                // 0.0 to 1.0 (pump/dump, spam)
}}

Rules:
1. Map company names to tickers (e.g., "Oracle" → "ORCL", "Tesla" → "TSLA")
2. Handle abbreviations (e.g., "Nvidia" → "NVDA")
3. Sentiment is per-ticker (one post can be bullish on X, bearish on Y)
4. Only include tickers for US-listed stocks and ETFs
5. Ignore crypto unless it's a crypto ETF
6. Mark is_primary=true for the ticker that is the main subject
7. If no valid tickers found, return empty tickers array
8. Set spam_score > 0.5 for pump-and-dump or promotional posts
"""
```

### Processing Logic

```python
class RedditAIProcessor:
    def __init__(self, anthropic_client, batch_size=50):
        self.client = anthropic_client
        self.batch_size = batch_size
        self.model = "claude-3-haiku-20240307"  # Fast & cheap

    async def process_batch(self) -> int:
        """Process a batch of unprocessed posts."""
        # 1. Get unprocessed posts
        posts = await self.get_unprocessed_posts(limit=self.batch_size)

        if not posts:
            return 0

        # 2. Process each post with LLM
        results = []
        for post in posts:
            try:
                extraction = await self.extract_tickers(post)
                results.append((post.id, extraction))
            except Exception as e:
                logger.error(f"Failed to process post {post.id}: {e}")
                continue

        # 3. Save results to database
        await self.save_results(results)

        return len(results)

    async def extract_tickers(self, post) -> dict:
        """Use LLM to extract tickers from post."""
        prompt = EXTRACTION_PROMPT.format(
            subreddit=post.subreddit,
            title=post.title,
            body=post.body[:2000] if post.body else "",
            flair=post.flair or ""
        )

        response = await self.client.messages.create(
            model=self.model,
            max_tokens=500,
            messages=[{"role": "user", "content": prompt}]
        )

        # Parse JSON response
        return json.loads(response.content[0].text)
```

## Cost Analysis

### Claude Haiku Pricing (as of Dec 2024)
- Input: $0.25 / 1M tokens
- Output: $1.25 / 1M tokens

### Estimated Usage (Unoptimized)
- Average post: ~200 tokens input, ~100 tokens output
- Posts per hour: ~3000 (30 subreddits × 100 posts)
- Daily posts: ~72,000
- Unoptimized cost: ~$12.60/day (~$380/month)

### Cost Optimization Strategies

We apply three optimizations to reduce costs by ~90%:

#### 1. Batch Processing (5 posts per API call)
Instead of 1 post per call, batch 5 posts together:
```
72,000 posts / 5 = 14,400 API calls
Cost reduction: 5x → ~$76/month
```

#### 2. Engagement Threshold
Only process posts with meaningful engagement (>5 upvotes OR >3 comments):
```
~60% of posts have meaningful engagement
43,200 posts / 5 = 8,640 API calls
Cost reduction: ~$45/month
```

#### 3. Company Name Cache
Cache extracted company→ticker mappings to skip redundant LLM calls:
```python
# If we've seen "Oracle" → "ORCL" before, use cache
COMPANY_CACHE = {
    "oracle": "ORCL", "tesla": "TSLA", "nvidia": "NVDA",
    "palantir": "PLTR", "gamestop": "GME", ...
}
# ~30% cache hit rate after warmup
6,000 API calls → ~$32/month
```

### Final Cost Estimate

| Strategy | Daily Posts | LLM Calls | Monthly Cost |
|----------|-------------|-----------|--------------|
| Unoptimized | 72,000 | 72,000 | ~$380 |
| + Batch (5x) | 72,000 | 14,400 | ~$76 |
| + Engagement filter | 43,200 | 8,640 | ~$45 |
| + Company cache | 43,200 | 6,000 | **~$32** |

**Final estimate: ~$30-45/month** (91% reduction from unoptimized)

## Pipeline Schedule

```yaml
# CronJob 1: Ingestion (hourly)
reddit-posts-ingestion:
  schedule: "0 * * * *"  # Every hour
  job: Fetch from Arctic Shift → reddit_posts_raw

# CronJob 2: AI Processing (every 15 min)
reddit-posts-ai-processor:
  schedule: "*/15 * * * *"  # Every 15 minutes
  job: Process unprocessed posts → reddit_post_tickers
  batch_size: 200

# CronJob 3: Aggregation (hourly)
reddit-sentiment-aggregator:
  schedule: "5 * * * *"  # 5 min after each hour
  job: Update social_posts from reddit_post_tickers (for API compatibility)
```

## API Changes

### New Endpoints

```
GET /api/v1/sentiment/v2/:ticker
  - Uses new tables
  - Returns multi-ticker data
  - Higher accuracy

GET /api/v1/sentiment/v2/:ticker/posts
  - Shows all posts mentioning ticker
  - Even if not primary ticker
```

### Backward Compatibility

- Keep existing `/api/v1/sentiment/*` endpoints
- Populate `social_posts` table from new pipeline
- Gradual migration

## Implementation Phases

### Phase 1: Schema & Ingestion (1-2 days)
- [ ] Create new database tables
- [ ] Modify ingestion to write to `reddit_posts_raw`
- [ ] Keep existing pipeline running in parallel

### Phase 2: AI Processor (2-3 days)
- [ ] Build LLM extraction service
- [ ] Test with sample posts
- [ ] Deploy as Kubernetes CronJob
- [ ] Monitor costs

### Phase 3: API Integration (1-2 days)
- [ ] Create v2 sentiment endpoints
- [ ] Update aggregation to populate `social_posts`
- [ ] Frontend integration (optional)

### Phase 4: Optimization (ongoing)
- [ ] Add pre-filtering to reduce LLM calls
- [ ] Batch processing optimization
- [ ] Cost monitoring dashboard

## Success Metrics

| Metric | V1 (Current) | V2 (Target) |
|--------|--------------|-------------|
| Tickers extracted per post | 1 | 2-3 avg |
| Company name recognition | 0% | 95%+ |
| Sentiment accuracy | ~60% | ~85% |
| Coverage (posts with tickers) | 35% | 60%+ |
| Cost per post | $0 | $0.0002 |

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| LLM costs spike | High | Pre-filtering, batching, cost alerts |
| LLM rate limits | Medium | Queuing, backoff, multiple API keys |
| Extraction errors | Medium | Fallback to regex, manual review |
| Schema migration | Low | Parallel operation, gradual rollout |

## Open Questions

1. Should we re-process historical posts in `social_posts`?
2. Do we need real-time processing or is batch OK?
3. Should we expose AI confidence scores in the UI?
4. Do we want to detect spam/pump-and-dump posts?
