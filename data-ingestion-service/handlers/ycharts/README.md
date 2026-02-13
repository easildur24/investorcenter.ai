# YCharts Ingestion Handlers

Structured ingestion endpoints for YCharts data with JSON Schema validation.

## Endpoints

### POST /ingest/ycharts/key_stats/:ticker

Upload YCharts Key Stats data (100+ financial metrics).

**Path Parameters:**
- `ticker` - Stock ticker symbol (e.g., NVDA, AAPL)

**Request Body:**
```json
{
  "collected_at": "2026-02-12T20:30:00Z",
  "source_url": "https://ycharts.com/companies/NVDA/key_stats/stats",
  "price": {
    "current": 186.96,
    "currency": "USD",
    "exchange": "NASDAQ",
    "timestamp": "2026-02-12T16:00:00Z"
  },
  "income_statement": {
    "revenue_ttm": 187140000000,
    "net_income_ttm": 99200000000,
    ...
  },
  "balance_sheet": { ... },
  "cash_flow": { ... },
  "valuation": { ... },
  "performance": { ... },
  "estimates": { ... },
  "dividends": { ... },
  "risk_metrics": { ... },
  "management_effectiveness": { ... },
  "advanced_metrics": { ... },
  "liquidity_solvency": { ... },
  "employees": { ... }
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "id": 12345,
    "ticker": "NVDA",
    "s3_key": "ycharts/key_stats/NVDA/2026-02-12/20260212T203000Z.json"
  }
}
```

**Response (400 Bad Request):**
```json
{
  "error": "Request validation failed",
  "validation_errors": [
    "income_statement.revenue_ttm: Invalid type. Expected: integer, given: string"
  ]
}
```

## Validation

Requests are validated against: `schemas/ycharts/key_stats.json`

The schema enforces:
- Required fields: `collected_at`, `source_url`
- Correct data types (integers for dollars, decimals for ratios)
- Nullable fields for missing data
- Date/timestamp formats

## Storage

**S3 Path:**
```
ycharts/key_stats/{TICKER}/{YYYY-MM-DD}/{timestamp}.json
```

Example: `ycharts/key_stats/NVDA/2026-02-12/20260212T203000Z.json`

**Index Table:** `ingestion_log`

The API only writes an index record to track what was uploaded:
- `source`: "ycharts"
- `ticker`: Stock symbol
- `data_type`: "key_stats"
- `s3_key`: Full S3 path
- `collected_at`: When data was scraped

**Processing:** A separate ETL cronjob reads from S3 and inserts into `ycharts_key_stats` table.

## Testing

**Authenticate:**
```bash
TOKEN=$(curl -s -X POST https://investorcenter.ai/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"'$CLAWDBOT_EMAIL'","password":"'$CLAWDBOT_PASSWORD'"}' \
  | jq -r .access_token)
```

**Upload data:**
```bash
curl -X POST https://investorcenter.ai/api/v1/ingest/ycharts/key_stats/NVDA \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @test_data.json
```

## Error Handling

| Status | Meaning |
|--------|---------|
| 201 | Successfully created |
| 400 | Invalid request (validation failed) |
| 401 | Unauthorized (missing or invalid JWT) |
| 409 | Conflict (duplicate ticker + collected_at) |
| 500 | Server error |

## Integration with Scraper

The scraper (Nikola/ClawdBot) will:
1. Navigate to YCharts Key Stats page
2. Extract metrics using the scraping skill
3. Parse values using `parse_helpers.py`
4. POST structured JSON to this endpoint
5. Endpoint validates and uploads to S3
6. Separate ETL job processes S3 files → database

## Adding New Endpoints

To add another YCharts endpoint (e.g., `/ingest/ycharts/financials/:ticker`):

1. Create schema: `schemas/ycharts/financials.json`
2. Create handler: `handlers/ycharts/financials.go`
3. Register route in `main.go`
4. Add migration for database table
5. Update this README
