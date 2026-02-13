# Data Ingestion Skill

Upload raw scraped data to InvestorCenter's data ingestion service. This skill is used after scraping data from external sources (ycharts, seekingalpha, SEC EDGAR, etc.) to persist the raw content for later processing.

## Endpoint

```
POST https://investorcenter.ai/api/v1/ingest
Authorization: Bearer <jwt>
Content-Type: application/json
```

## Authentication

Use your worker credentials to get a JWT token:

```bash
TOKEN=$(curl -s -X POST https://investorcenter.ai/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$CLAWDBOT_EMAIL\", \"password\": \"$CLAWDBOT_PASSWORD\"}" \
  | jq -r .access_token)
```

## Request Format

```json
{
  "source": "ycharts",
  "ticker": "NVDA",
  "data_type": "financials",
  "source_url": "https://ycharts.com/companies/NVDA/financials",
  "raw_data": "<the raw scraped content>",
  "collected_at": "2026-02-12T15:30:00Z"
}
```

### Fields

| Field | Type | Required | Max Size | Description |
|-------|------|----------|----------|-------------|
| `source` | string | Yes | 50 chars | Data provider: `ycharts`, `seekingalpha`, `sec_edgar`, `reddit`, etc. |
| `data_type` | string | Yes | 100 chars | Content category within the source (see examples below) |
| `ticker` | string | No | 20 chars | Stock ticker symbol (e.g., `NVDA`, `AAPL`). Omit for non-ticker data. |
| `source_url` | string | No | 2000 chars | The original URL that was scraped |
| `raw_data` | string | Yes | 10 MB | The raw scraped content (HTML, JSON, text, etc.) |
| `collected_at` | string | No | - | ISO 8601 timestamp of when the data was scraped. Defaults to now. |

### Response

```json
{
  "success": true,
  "data": {
    "id": 123,
    "s3_key": "raw/ycharts/NVDA/financials/2026-02-12/20260212T153000Z.json"
  }
}
```

## Source + Data Type Conventions

Use these standard `source` and `data_type` values:

### YCharts (`source: "ycharts"`)

| data_type | Description | Example URL |
|-----------|-------------|-------------|
| `financials` | Income statement, balance sheet, cash flow | `/companies/NVDA/financials` |
| `valuation` | P/E, P/S, P/B, EV/EBITDA ratios | `/companies/NVDA/valuation` |
| `dividends` | Dividend history, yield, payout ratio | `/companies/NVDA/dividends` |
| `performance` | Price returns, total returns | `/companies/NVDA/performance` |
| `profitability` | Margins, ROE, ROA, ROIC | `/companies/NVDA/profitability` |
| `growth` | Revenue growth, earnings growth | `/companies/NVDA/growth_rates` |
| `estimates` | Analyst estimates, EPS forecasts | `/companies/NVDA/estimates` |
| `technicals` | Moving averages, RSI, MACD | `/companies/NVDA/technicals` |
| `peers` | Peer comparison data | `/companies/NVDA/peers` |

### SeekingAlpha (`source: "seekingalpha"`)

| data_type | Description | Example URL |
|-----------|-------------|-------------|
| `ratings` | Quant/Wall Street/SA author ratings | `/symbol/NVDA/ratings` |
| `analysis` | Recent analysis articles | `/symbol/NVDA/analysis` |
| `news` | Recent news articles | `/symbol/NVDA/news` |
| `earnings` | Earnings call transcripts | `/symbol/NVDA/earnings` |
| `dividends` | Dividend data and safety | `/symbol/NVDA/dividends` |
| `momentum` | Momentum grades and metrics | `/symbol/NVDA/momentum` |
| `profitability` | Profitability grades | `/symbol/NVDA/profitability` |
| `valuation` | Valuation grades | `/symbol/NVDA/valuation` |
| `growth` | Growth grades | `/symbol/NVDA/growth` |

### SEC EDGAR (`source: "sec_edgar"`)

| data_type | Description |
|-----------|-------------|
| `10k` | Annual report |
| `10q` | Quarterly report |
| `8k` | Current report |
| `def14a` | Proxy statement |
| `13f` | Institutional holdings |

## Examples

### Scrape YCharts Financials

```bash
# After scraping the page content into $RAW_HTML
curl -s -X POST https://investorcenter.ai/api/v1/ingest \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"source\": \"ycharts\",
    \"ticker\": \"NVDA\",
    \"data_type\": \"financials\",
    \"source_url\": \"https://ycharts.com/companies/NVDA/financials\",
    \"raw_data\": $(echo "$RAW_HTML" | jq -Rs .),
    \"collected_at\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
  }"
```

### Scrape SeekingAlpha Ratings

```bash
curl -s -X POST https://investorcenter.ai/api/v1/ingest \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"source\": \"seekingalpha\",
    \"ticker\": \"AAPL\",
    \"data_type\": \"ratings\",
    \"source_url\": \"https://seekingalpha.com/symbol/AAPL/ratings\",
    \"raw_data\": $(echo "$RAW_CONTENT" | jq -Rs .),
    \"collected_at\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
  }"
```

### Scrape Multiple Tabs for One Ticker

When a task says "scrape all data for NVDA from ycharts", iterate through each tab:

```
1. Scrape financials → POST /ingest with data_type="financials"
2. Scrape valuation  → POST /ingest with data_type="valuation"
3. Scrape dividends  → POST /ingest with data_type="dividends"
4. Scrape performance → POST /ingest with data_type="performance"
... and so on for each tab
```

Each tab is a separate ingestion call. This keeps the data organized in S3.

## Error Handling

| Status | Error | Action |
|--------|-------|--------|
| `400` | Invalid request | Check field names and types |
| `401` | Unauthorized | Re-authenticate (token may be expired) |
| `413` | Payload too large | Split raw_data into smaller chunks |
| `500` | Server error | Retry up to 3 times with exponential backoff |

## Integration with Task Workflow

When executing a scraping task:

1. Pick up task from task queue (`GET /worker/tasks?status=pending`)
2. Mark task as in_progress (`PUT /worker/tasks/:id/status`)
3. Navigate to the target page and scrape the content
4. **Upload raw data via this skill** (`POST /ingest`)
5. Post progress update (`POST /worker/tasks/:id/updates`)
6. Repeat steps 3-5 for each page/tab
7. Post task result summary (`POST /worker/tasks/:id/result`)
8. Mark task as completed (`PUT /worker/tasks/:id/status`)

## Important Notes

- **Send raw data** — do not parse, structure, or clean the data before uploading. The processing pipeline handles that.
- **One call per page/tab** — each scraped page should be a separate ingestion call with the appropriate `data_type`.
- **Always include `source_url`** — this helps with debugging and re-scraping.
- **Always include `ticker`** when the data is about a specific stock.
- **Max 10MB per request** — if a page is larger, consider splitting it or extracting only the relevant section.
- **Retries are safe** — the S3 key includes a timestamp, so re-uploads create new objects (no overwrites unless exact same second).
