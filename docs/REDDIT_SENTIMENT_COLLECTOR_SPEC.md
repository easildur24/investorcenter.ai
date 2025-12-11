# Reddit Sentiment Collector - Technical Specification

## Overview

Build a Reddit post collector that fetches actual post content from stock-related subreddits using Reddit's public `.json` endpoint (no API key required), analyzes sentiment using our lexicon, and stores results in the `social_posts` table.

**Goal:** Replace the current ApeWisdom-based collector with one that provides:
- Full post content (title + body) for sentiment analysis
- Ticker extraction from post text
- Sentiment scoring using our 235-term lexicon
- Representative posts for display in the frontend

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Reddit Sentiment Collector                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │ Reddit JSON  │───>│   Ticker     │───>│  Sentiment Analyzer  │  │
│  │   Fetcher    │    │  Extractor   │    │  (Lexicon-based)     │  │
│  └──────────────┘    └──────────────┘    └──────────────────────┘  │
│         │                                          │                 │
│         │                                          │                 │
│         v                                          v                 │
│  ┌──────────────┐                        ┌──────────────────────┐  │
│  │  RSS Feed    │                        │   PostgreSQL         │  │
│  │  Monitor     │                        │   social_posts       │  │
│  │ (Real-time)  │                        │   sentiment_lexicon  │  │
│  └──────────────┘                        └──────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Data Sources

### 1. Reddit JSON Endpoint (Primary)

**URL Format:**
```
https://www.reddit.com/r/{subreddit}/{sort}.json?limit=100&t={time}
```

**Parameters:**
| Parameter | Values | Description |
|-----------|--------|-------------|
| `sort` | `hot`, `new`, `top`, `rising` | Post ordering |
| `limit` | 1-100 | Posts per request |
| `t` | `hour`, `day`, `week` | Time filter (for `top`) |
| `after` | `t3_xxxxx` | Pagination cursor |

**Target Subreddits:**
| Subreddit | Focus | Priority |
|-----------|-------|----------|
| `wallstreetbets` | Options, YOLO plays | High |
| `stocks` | General stock discussion | High |
| `options` | Options strategies | Medium |
| `investing` | Long-term investing | Medium |
| `Daytrading` | Day trading | Medium |
| `stockmarket` | Market news | Low |
| `pennystocks` | Small caps | Low |

**Rate Limits:**
- ~10 requests/minute unauthenticated
- Must set custom User-Agent header
- 6-second delay between requests recommended

**Response Structure:**
```json
{
  "data": {
    "children": [
      {
        "data": {
          "id": "abc123",
          "title": "$AAPL to the moon! 🚀🚀🚀",
          "selftext": "Just bought 100 shares...",
          "author": "username",
          "subreddit": "wallstreetbets",
          "score": 1500,
          "num_comments": 234,
          "created_utc": 1701619200,
          "permalink": "/r/wallstreetbets/comments/...",
          "link_flair_text": "YOLO",
          "url": "https://reddit.com/r/...",
          "is_self": true,
          "over_18": false,
          "stickied": false
        }
      }
    ],
    "after": "t3_xyz789"
  }
}
```

### 2. Reddit RSS Feed (Supplementary - Real-time)

**URL Format:**
```
https://www.reddit.com/r/{subreddit}/new/.rss?limit=25
```

**Use Case:**
- Quick polling for new posts (every 5 minutes)
- Lower overhead than JSON endpoint
- Supplement JSON fetches between full collection cycles

**Limitations:**
- Max 25 items per feed
- Limited metadata (no score, comment count)
- No body text for self-posts

---

## Components

### 1. RedditJSONFetcher

```python
class RedditJSONFetcher:
    """Fetches posts from Reddit's public JSON API"""

    BASE_URL = "https://www.reddit.com/r/{subreddit}/{sort}.json"
    USER_AGENT = "InvestorCenter/1.0 (Stock Sentiment Analysis)"
    RATE_LIMIT_DELAY = 6.0  # seconds between requests

    def fetch_subreddit(self, subreddit: str, sort: str = "hot",
                        limit: int = 100) -> List[RedditPost]:
        """Fetch posts from a subreddit"""

    def fetch_all_subreddits(self, subreddits: List[str]) -> List[RedditPost]:
        """Fetch from multiple subreddits with rate limiting"""

    def paginate(self, subreddit: str, max_posts: int = 500) -> List[RedditPost]:
        """Fetch multiple pages using 'after' cursor"""
```

**Filtering:**
- Skip `stickied` posts (megathreads, rules)
- Skip `over_18` (NSFW) posts
- Skip posts with score < 5 (low engagement)
- Skip posts older than 7 days

### 2. TickerExtractor

Port the existing Go implementation to Python:

```python
class TickerExtractor:
    """Extracts and validates stock tickers from text"""

    def __init__(self, valid_tickers: Set[str]):
        self.valid_tickers = valid_tickers
        self.false_positives = self._load_false_positives()

    def extract(self, title: str, body: str) -> List[TickerMention]:
        """Extract ticker mentions from post content"""
        # 1. Find $TICKER patterns (high confidence)
        # 2. Find standalone TICKER patterns (validate against DB)
        # 3. Rank by: title > count > position

    def get_primary_ticker(self, mentions: List[TickerMention]) -> str:
        """Determine the main ticker being discussed"""
```

**False Positives to Filter:**
```python
FALSE_POSITIVES = {
    # Common words
    "I", "A", "IT", "AT", "ON", "IS", "BE", "DO", "GO", "SO", "TO", "UP",
    # Business terms
    "CEO", "CFO", "DD", "IPO", "EPS", "PE", "ETF", "SEC", "FED",
    # Reddit slang
    "IMO", "TBH", "LOL", "EDIT", "TLDR", "OP", "WSB",
    # Trading terms (not tickers)
    "BUY", "SELL", "HOLD", "CALL", "PUT", "YOLO", "MOON", "BEAR", "BULL",
    # Sentiment lexicon overlap
    "ATH", "ATL", "FUD", "FOMO", "HODL",
}
```

### 3. SentimentAnalyzer

Port the existing Go implementation to Python:

```python
class SentimentAnalyzer:
    """Lexicon-based sentiment analysis"""

    def __init__(self, lexicon: Dict[str, LexiconEntry]):
        self.lexicon = lexicon

    def analyze(self, title: str, body: str) -> SentimentResult:
        """
        Analyze sentiment using lexicon matching.

        Returns:
            SentimentResult with:
            - sentiment: "bullish", "bearish", "neutral"
            - score: -1.0 to 1.0
            - confidence: 0.0 to 1.0
            - matched_terms: list of found terms
        """
        # 1. Combine title (weighted 2x) + body
        # 2. Clean text (remove URLs, normalize)
        # 3. Multi-word phrase matching (up to 4 words)
        # 4. Handle modifiers (negation, amplifiers, reducers)
        # 5. Calculate weighted score
```

**Modifier Handling:**
- **Negation** (not, don't, never): Flip sentiment for next 3 words
- **Amplifiers** (very, extremely): Multiply weight by 1.3-1.5
- **Reducers** (maybe, might): Multiply weight by 0.5-0.7

### 4. Database Layer

Use existing `social_posts` table schema:

```sql
CREATE TABLE social_posts (
    id BIGSERIAL PRIMARY KEY,
    external_post_id VARCHAR(50) UNIQUE NOT NULL,  -- Reddit post ID
    source VARCHAR(20) DEFAULT 'reddit',
    ticker VARCHAR(10) NOT NULL,                    -- Primary ticker
    subreddit VARCHAR(50) NOT NULL,
    title TEXT NOT NULL,
    body_preview VARCHAR(500),                      -- First 500 chars
    url TEXT NOT NULL,
    upvotes INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    award_count INTEGER DEFAULT 0,
    sentiment VARCHAR(10),                          -- bullish/bearish/neutral
    sentiment_confidence DECIMAL(3,2),
    flair VARCHAR(50),
    posted_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## Collection Strategy

### Collection Modes

#### 1. Full Collection (CronJob - Every 4 hours)
```python
def full_collection():
    """Comprehensive collection from all subreddits"""
    for subreddit in SUBREDDITS:
        posts = fetcher.fetch_subreddit(subreddit, sort="hot", limit=100)
        posts += fetcher.fetch_subreddit(subreddit, sort="new", limit=50)
        posts += fetcher.fetch_subreddit(subreddit, sort="top", limit=50, time="day")

        for post in posts:
            tickers = extractor.extract(post.title, post.body)
            if not tickers:
                continue  # Skip posts without stock mentions

            sentiment = analyzer.analyze(post.title, post.body)

            for ticker in tickers:
                db.upsert_post(post, ticker, sentiment)
```

#### 2. Quick Refresh (CronJob - Every 30 minutes)
```python
def quick_refresh():
    """Quick update from RSS feeds for new posts"""
    for subreddit in HIGH_PRIORITY_SUBREDDITS:
        posts = rss_fetcher.fetch(subreddit, limit=25)
        # Process only truly new posts (not in DB)
```

### Deduplication

- Use `external_post_id` (Reddit's post ID) as unique key
- ON CONFLICT: Update engagement metrics (upvotes, comments)
- Don't re-analyze sentiment for existing posts

### Data Retention

- Keep posts for 30 days
- Prune older posts via daily cleanup job
- Archive aggregated metrics before pruning

---

## Kubernetes Deployment

### CronJob: Full Collection

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: reddit-sentiment-collector
  namespace: investorcenter
spec:
  schedule: "0 */4 * * *"  # Every 4 hours
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: collector
            image: investorcenter/reddit-collector:latest
            command: ["python", "reddit_collector.py", "--mode", "full"]
            env:
            - name: DB_HOST
              value: "postgres-service"
            - name: DB_NAME
              value: "investorcenter_db"
            resources:
              requests:
                memory: "256Mi"
                cpu: "100m"
              limits:
                memory: "512Mi"
                cpu: "500m"
```

### CronJob: Quick Refresh

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: reddit-sentiment-refresh
  namespace: investorcenter
spec:
  schedule: "*/30 * * * *"  # Every 30 minutes
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: collector
            image: investorcenter/reddit-collector:latest
            command: ["python", "reddit_collector.py", "--mode", "quick"]
```

---

## Error Handling

### Rate Limiting
```python
def handle_rate_limit(response):
    if response.status_code == 429:
        retry_after = int(response.headers.get('Retry-After', 60))
        logger.warning(f"Rate limited, waiting {retry_after}s")
        time.sleep(retry_after)
        return True  # Retry
    return False
```

### Connection Errors
```python
MAX_RETRIES = 3
RETRY_DELAY = 10

def fetch_with_retry(url):
    for attempt in range(MAX_RETRIES):
        try:
            response = requests.get(url, headers=HEADERS, timeout=15)
            response.raise_for_status()
            return response.json()
        except (requests.RequestException, json.JSONDecodeError) as e:
            if attempt < MAX_RETRIES - 1:
                time.sleep(RETRY_DELAY * (attempt + 1))
            else:
                raise
```

### Missing Data
- Skip posts without valid tickers
- Skip posts with empty title/body
- Log but don't fail on individual post errors

---

## Metrics & Monitoring

### Logging
```python
# Collection metrics
logger.info(f"Collected {total_posts} posts from {subreddit}")
logger.info(f"Extracted {ticker_count} unique tickers")
logger.info(f"Sentiment: {bullish_count} bullish, {bearish_count} bearish")

# Rate limiting
logger.warning(f"Rate limited at {current_time}, waiting {delay}s")

# Errors
logger.error(f"Failed to fetch {subreddit}: {error}")
```

### Database Metrics
```sql
-- Posts collected today
SELECT COUNT(*) FROM social_posts WHERE created_at > NOW() - INTERVAL '1 day';

-- Top tickers by mention count
SELECT ticker, COUNT(*) FROM social_posts
WHERE posted_at > NOW() - INTERVAL '24 hours'
GROUP BY ticker ORDER BY COUNT(*) DESC LIMIT 20;

-- Sentiment distribution
SELECT sentiment, COUNT(*) FROM social_posts
WHERE posted_at > NOW() - INTERVAL '24 hours'
GROUP BY sentiment;
```

---

## File Structure

```
scripts/
├── reddit_sentiment_collector.py   # Main collector script
├── reddit/
│   ├── __init__.py
│   ├── fetcher.py                  # RedditJSONFetcher, RSSFetcher
│   ├── models.py                   # RedditPost, TickerMention, SentimentResult
│   ├── ticker_extractor.py         # TickerExtractor
│   ├── sentiment_analyzer.py       # SentimentAnalyzer
│   └── database.py                 # Database operations
├── Dockerfile.reddit-collector     # Docker image
└── requirements-reddit.txt         # Python dependencies

k8s/
├── reddit-sentiment-collector-cronjob.yaml
└── reddit-sentiment-refresh-cronjob.yaml
```

---

## Implementation Phases

### Phase 1: Core Collector (MVP)
- [ ] Reddit JSON fetcher with rate limiting
- [ ] Basic ticker extraction ($$TICKER patterns)
- [ ] Port sentiment analyzer from Go to Python
- [ ] Store in social_posts table
- [ ] Single subreddit (wallstreetbets) test

### Phase 2: Full Coverage
- [ ] All target subreddits
- [ ] Pagination support (up to 500 posts/subreddit)
- [ ] RSS feed quick refresh
- [ ] Kubernetes CronJob deployment

### Phase 3: Optimization
- [ ] Parallel fetching (per subreddit)
- [ ] Caching for ticker validation
- [ ] Improved ticker extraction (context-aware)
- [ ] Sentiment calibration/tuning

---

## Testing Plan

### Unit Tests
```python
def test_ticker_extraction():
    extractor = TickerExtractor(valid_tickers={"AAPL", "TSLA", "GME"})

    # Test $TICKER pattern
    mentions = extractor.extract("$AAPL to the moon!", "")
    assert mentions[0].ticker == "AAPL"

    # Test false positive filtering
    mentions = extractor.extract("I think CEO said...", "")
    assert len(mentions) == 0

def test_sentiment_analysis():
    analyzer = SentimentAnalyzer(lexicon)

    # Test bullish
    result = analyzer.analyze("GME to the moon! 🚀🚀🚀 Diamond hands!", "")
    assert result.sentiment == "bullish"
    assert result.score > 0.5

    # Test bearish
    result = analyzer.analyze("This is a scam, paper hands selling", "")
    assert result.sentiment == "bearish"

    # Test negation
    result = analyzer.analyze("This is NOT bullish at all", "")
    assert result.sentiment != "bullish"
```

### Integration Tests
- Fetch from real Reddit endpoint
- Store in test database
- Verify deduplication
- Test rate limiting behavior

---

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Reddit blocks our IP | Use rotating User-Agent, respect rate limits |
| Rate limit exceeded | Exponential backoff, reduce frequency |
| False ticker matches | Robust false positive list, context validation |
| Sentiment accuracy | Tune lexicon weights, manual review samples |
| Data staleness | Quick refresh every 30 min, prioritize active posts |

---

## Success Metrics

| Metric | Target |
|--------|--------|
| Posts collected per day | 500+ |
| Unique tickers per day | 100+ |
| Sentiment accuracy (manual review) | >75% |
| Collection success rate | >95% |
| API endpoint latency | <200ms |
