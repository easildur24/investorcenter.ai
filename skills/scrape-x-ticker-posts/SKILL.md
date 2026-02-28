# X (Twitter) Ticker Posts Scraping Skill

**Goal:** Search X for recent posts about a stock ticker using the cashtag (e.g. $AAPL), extract post data, and upload to data ingestion API.

## Execution Pattern

**This skill is executed directly by the AI agent.** The workflow is:
1. Open browser, navigate to X search for the ticker's cashtag
2. Scroll to load posts, capture snapshot
3. Extract post data from the snapshot
4. Use exec + Python to parse and POST to the API
5. Stop browser when done

**Do not overthink this.** It's just web scraping + API calls for a search results page.

## Prerequisites

1. **X account** (already logged in via openclaw browser profile)
2. **Worker API credentials** in TOOLS.md (`nikola@investorcenter.ai`)
3. **OpenClaw browser profile** (NOT Chrome extension relay!)

## Task Params

```json
{
  "ticker": "AAPL"
}
```

## URL Pattern

```
https://x.com/search?q=%24{TICKER}&src=typed_query&f=top
```

The `%24` is the URL-encoded `$` for the cashtag. The `f=top` sorts by Top posts (most relevant). You can also try `f=live` for Latest.

## Workflow

### Step 1: Open X Search Page

**CRITICAL:** Use `profile="openclaw"` - this is the standalone browser, NOT the Chrome extension relay.

```javascript
browser.open({
  profile: "openclaw",
  targetUrl: "https://x.com/search?q=%24{TICKER}&src=typed_query&f=top"
})
```

Save the `targetId` for subsequent browser actions.

### Step 2: Wait for Page Load

```bash
sleep 3
```

X search pages can be slow to render. Wait at least 3 seconds.

### Step 3: Scroll to Load More Posts

Scroll down 2-3 times to load more posts beyond the initial viewport:

```javascript
browser.action({
  profile: "openclaw",
  targetId: "{targetId}",
  action: "scroll",
  coordinate: [512, 400],
  direction: "down",
  amount: 5
})
```

Wait 2 seconds between scrolls to let posts load.

### Step 4: Capture Page Snapshot

```javascript
browser.snapshot({
  profile: "openclaw",
  targetId: "{targetId}",
  maxChars: 100000
})
```

### Step 5: Extract Post Data

From the snapshot, extract each visible post. For each post, capture:

| Field | Where to Find | Notes |
|-------|--------------|-------|
| `author_handle` | @handle next to display name | e.g. "@elonmusk" |
| `author_name` | Display name above handle | e.g. "Elon Musk" |
| `author_verified` | Blue checkmark badge | true/false |
| `content` | Full post text | Include $cashtags and @mentions |
| `timestamp` | Time shown on post | Relative ("2h", "3d") or absolute date |
| `likes` | Heart icon count | Parse "1.2K" → 1200 |
| `reposts` | Repost icon count | Parse "1.2K" → 1200 |
| `replies` | Reply icon count | Parse "1.2K" → 1200 |
| `views` | Views count (if shown) | Parse "1.2M" → 1200000 |
| `bookmarks` | Bookmark icon count | Parse if visible |
| `post_url` | Link to the post | Construct from author handle + post ID if needed |
| `has_media` | Image/video in post | true/false |
| `is_repost` | "Reposted" label | true/false |
| `is_reply` | "Replying to" label | true/false |

#### Parsing engagement numbers

X displays counts in abbreviated form. Convert them:
- "1" → 1
- "15" → 15
- "1.2K" → 1200
- "15K" → 15000
- "1.2M" → 1200000
- "" or missing → null

### Step 6: Determine as_of_date

Use today's date in YYYY-MM-DD format.

### Step 7: Ingest Data

```python
import requests, json
from datetime import date

# Get auth token
response = requests.post(
    "https://investorcenter.ai/api/v1/auth/login",
    json={"email": "nikola@investorcenter.ai", "password": "ziyj9VNdHH5tjqB2m3lup3MG"}
)
token = response.json()["access_token"]

ticker = "AAPL"  # from task params
data = {
    "as_of_date": str(date.today()),
    "source_url": f"https://x.com/search?q=%24{ticker}&src=typed_query&f=top",
    "search_query": f"${ticker}",
    "post_count": len(posts),
    "posts": posts  # array of post objects extracted above
}

response = requests.post(
    f"https://investorcenter.ai/api/v1/ingest/x/ticker_posts/{ticker}",
    headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
    json=data
)
print(response.status_code, response.json())
```

### Step 8: Close Browser

```javascript
browser({
  action: "stop",
  profile: "openclaw"
})
```

## S3 Storage Path

```
x/ticker_posts/{TICKER}/{YYYY-MM-DD}/{timestamp}.json
```

## API Endpoint

```
POST /api/v1/ingest/x/ticker_posts/{TICKER}
```

## Tips

- **Login required:** X search requires being logged in. The openclaw browser profile should already be authenticated.
- **Rate limits:** X may show a rate limit page. If you see "Something went wrong" or rate limit messages, wait 30 seconds and retry.
- **No pagination needed:** Just scroll 2-3 times to get 20-30 top posts. We don't need exhaustive scraping.
- **Skip ads:** X shows promoted posts. Skip any post marked as "Ad" or "Promoted".
- **Engagement nulls:** If engagement counts aren't visible (e.g. views hidden), use null, not 0.

## Summary

1. Open X search for `$TICKER` cashtag in openclaw browser
2. Wait 3 seconds, scroll down 2-3 times to load posts
3. Take snapshot, extract post data (author, content, engagement metrics)
4. POST to ingestion API with as_of_date = today
5. Stop browser
6. Report result

**This is a single-page scrape** — scroll a few times for more posts, no complex pagination needed. Aim for 20-30 posts per run.
