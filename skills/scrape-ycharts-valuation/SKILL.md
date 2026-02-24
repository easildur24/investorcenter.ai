# YCharts Valuation Scraping Skill

**Goal:** Extract current and historical valuation ratios from YCharts valuation page and upload to data ingestion API.

## Execution Pattern

**This skill is executed directly by the AI agent.** The workflow is:
1. Open browser, navigate to YCharts valuation page
2. Capture snapshot (100k chars max)
3. Extract all data sections from the snapshot
4. Use exec + Python to parse and POST to the API
5. Stop browser when done

**Do not overthink this.** It's just web scraping + API calls for a single page.

## Prerequisites

1. **YCharts account login** (already logged in at `easildur24@gmail.com`)
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
https://ycharts.com/companies/{TICKER}/valuation
```

## Workflow

### Step 1: Open YCharts Valuation Page

**CRITICAL:** Use `profile="openclaw"` - this is the standalone browser, NOT the Chrome extension relay.

```javascript
browser.open({
  profile: "openclaw",
  targetUrl: "https://ycharts.com/companies/{TICKER}/valuation"
})
```

Save the `targetId` for subsequent browser actions.

### Step 2: Wait for Page Load

```bash
sleep 2
```

### Step 3: Capture Page Snapshot

```javascript
browser.snapshot({
  profile: "openclaw",
  targetId: "{targetId}",
  maxChars: 100000
})
```

### Step 4: Extract Data

The page has 4 main tables. Extract all of them.

#### Table 1: Price Ratios (Current + Comparisons)

Table columns: TTM, Industry Avg, 3Y Median, 5Y Median, 10Y Median, Fwd 1Y

| Row Label | JSON Field Prefix | Parse Function |
|-----------|-------------------|----------------|
| PE Ratio | `price_ratios.pe_ratio_*` | `parse_float` |
| PS Ratio | `price_ratios.ps_ratio_*` | `parse_float` |
| Price to Book Value | `price_ratios.price_to_book_*` | `parse_float` |
| Price to Free Cash Flow | `price_ratios.price_to_fcf_*` | `parse_float` |
| PEG Ratio | `price_ratios.peg_ratio_*` | `parse_float` |

For each row, extract values for each column:
- `_ttm` (TTM column)
- `_industry_avg` (Industry Avg column)
- `_3y_median` (3Y Median column)
- `_5y_median` (5Y Median column)
- `_10y_median` (10Y Median column)
- `_forward_1y` (Fwd 1Y column, only for PE and PS)

#### Table 2: Enterprise Ratios (Current + Comparisons)

Table columns: TTM, Industry Avg, 3Y Median, 5Y Median, 10Y Median

| Row Label | JSON Field Prefix | Parse Function |
|-----------|-------------------|----------------|
| EV to EBITDA | `enterprise_ratios.ev_to_ebitda_*` | `parse_float` |
| EV to EBIT | `enterprise_ratios.ev_to_ebit_*` | `parse_float` |
| EV to Revenues | `enterprise_ratios.ev_to_revenue_*` | `parse_float` |
| EV to Free Cash Flow | `enterprise_ratios.ev_to_fcf_*` | `parse_float` |
| EV to Assets | `enterprise_ratios.ev_to_assets_*` | `parse_float` |

Suffixes: `_ttm`, `_industry_avg`, `_3y_median`, `_5y_median`, `_10y_median`

#### Table 3: Historical Price Ratios by Year

Table columns: Latest, 2025, 2024, 2023, ... back to ~2016

Extract as array `historical_price_ratios`:
```json
[
  {"year": 2025, "pe_ratio": 33.68, "ps_ratio": 9.52, "price_to_book": 62.11, "price_to_fcf": 31.24, "peg_ratio": null},
  {"year": 2024, "pe_ratio": 38.79, ...},
  ...
]
```

Skip the "Latest" column (it duplicates the TTM values above).

| Row | JSON Field | Parse Function |
|-----|-----------|----------------|
| PE Ratio | `pe_ratio` | `parse_float` |
| PS Ratio | `ps_ratio` | `parse_float` |
| Price to Book Value | `price_to_book` | `parse_float` |
| Price to Free Cash Flow | `price_to_fcf` | `parse_float` |
| PEG Ratio | `peg_ratio` | `parse_float` |

#### Table 4: Historical Enterprise Ratios by Year

Same layout as Table 3. Extract as array `historical_enterprise_ratios`:
```json
[
  {"year": 2025, "ev_to_ebitda": 26.89, "ev_to_ebit": 32.56, "ev_to_revenue": 9.80, "ev_to_fcf": 32.10, "ev_to_assets": 10.54},
  ...
]
```

| Row | JSON Field | Parse Function |
|-----|-----------|----------------|
| EV to EBITDA | `ev_to_ebitda` | `parse_float` |
| EV to EBIT | `ev_to_ebit` | `parse_float` |
| EV to Revenues | `ev_to_revenue` | `parse_float` |
| EV to Free Cash Flow | `ev_to_fcf` | `parse_float` |
| EV to Assets | `ev_to_assets` | `parse_float` |

### Step 5: Determine as_of_date

Use today's date in YYYY-MM-DD format as `as_of_date`.

### Step 6: Ingest Data

```python
import requests, json

# Get auth token
response = requests.post(
    "https://investorcenter.ai/api/v1/auth/login",
    json={"email": "nikola@investorcenter.ai", "password": "ziyj9VNdHH5tjqB2m3lup3MG"}
)
token = response.json()["access_token"]

data = {
    "as_of_date": "2026-02-24",
    "source_url": "https://ycharts.com/companies/AAPL/valuation",
    "price_ratios": { ... },
    "enterprise_ratios": { ... },
    "historical_price_ratios": [ ... ],
    "historical_enterprise_ratios": [ ... ]
}

response = requests.post(
    f"https://investorcenter.ai/api/v1/ingest/ycharts/valuation/{ticker}",
    headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
    json=data
)
```

### Step 7: Close Browser

```javascript
browser({
  action: "stop",
  profile: "openclaw"
})
```

## Parse Helpers

Reuse from key stats skill:
```python
sys.path.append('/Users/larryli/.openclaw/workspace/investorcenter.ai/skills/scrape-ycharts-keystats')
from parse_helpers import parse_dollar_amount, parse_percentage, parse_float, parse_integer
```

## S3 Storage Path

```
ycharts/valuation/{TICKER}/{YYYY-MM-DD}/{timestamp}.json
```

## API Endpoint

```
POST /api/v1/ingest/ycharts/valuation/{TICKER}
```

## Summary

1. Open valuation page in openclaw browser
2. Wait 2 seconds, take snapshot
3. Extract 4 tables: price ratios, enterprise ratios, historical price ratios, historical enterprise ratios
4. POST to ingestion API with as_of_date
5. Stop browser
6. Report result

**This is a single-page scrape** — no pagination needed. All data is on one page.
