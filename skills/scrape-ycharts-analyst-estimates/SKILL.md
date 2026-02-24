# YCharts Analyst Estimates Scraping Skill

**Goal:** Extract analyst estimates, price targets, and recommendations from YCharts estimates page and upload to data ingestion API.

## Execution Pattern

**This skill is executed directly by the AI agent.** The workflow is:
1. Open browser, navigate to YCharts estimates page
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
https://ycharts.com/companies/{TICKER}/estimates
```

## Workflow

### Step 1: Open YCharts Estimates Page

**CRITICAL:** Use `profile="openclaw"` - this is the standalone browser, NOT the Chrome extension relay.

```javascript
browser.open({
  profile: "openclaw",
  targetUrl: "https://ycharts.com/companies/{TICKER}/estimates"
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

The page has multiple sections. Extract all of them into a single JSON object.

#### Section 1: Current Period Estimates

Table with columns: Actual, Estimated, Surprise, Current Est., 30 Day Chg., YoY Growth

| Row Label | JSON Path | Parse Function |
|-----------|-----------|----------------|
| EPS | `current_period.eps_actual`, `current_period.eps_estimated`, `current_period.eps_surprise_pct` | `parse_float`, `parse_float`, `parse_percentage` |
| EPS GAAP | `current_period.eps_gaap_actual`, `current_period.eps_gaap_estimated`, `current_period.eps_gaap_surprise_pct` | `parse_float`, `parse_float`, `parse_percentage` |
| Revenue | `current_period.revenue_actual`, `current_period.revenue_estimated`, `current_period.revenue_surprise_pct` | `parse_dollar_amount`, `parse_dollar_amount`, `parse_percentage` |
| EBITDA | `current_period.ebitda_actual`, `current_period.ebitda_estimated`, `current_period.ebitda_surprise_pct` | `parse_dollar_amount`, `parse_dollar_amount`, `parse_percentage` |
| EBITDA Margin | `current_period.ebitda_margin_actual`, `current_period.ebitda_margin_estimated`, `current_period.ebitda_margin_surprise_pct` | `parse_percentage`, `parse_percentage`, `parse_percentage` |
| 30 Day Chg. | `current_period.estimate_30d_change_pct` | `parse_percentage` |
| YoY Growth | `current_period.yoy_growth_pct` | `parse_percentage` |

Also extract the period identifier (e.g. "2026") into `current_period.period`.

#### Section 2: Future Periods (2027, 2028, etc.)

Array of future period objects. For each period:

| Field | JSON Path | Parse Function |
|-------|-----------|----------------|
| Period | `future_periods[].period` | string (e.g. "2027") |
| EPS Mean | `future_periods[].eps_mean` | `parse_float` |
| EPS Std Dev | `future_periods[].eps_std_dev` | `parse_float` |
| EPS # Analysts | `future_periods[].eps_num_analysts` | `parse_integer` |
| EPS GAAP Mean | `future_periods[].eps_gaap_mean` | `parse_float` |
| EPS GAAP Std Dev | `future_periods[].eps_gaap_std_dev` | `parse_float` |
| EPS GAAP # Analysts | `future_periods[].eps_gaap_num_analysts` | `parse_integer` |
| Revenue Mean | `future_periods[].revenue_mean` | `parse_dollar_amount` |
| Revenue Std Dev | `future_periods[].revenue_std_dev` | `parse_dollar_amount` |
| Revenue # Analysts | `future_periods[].revenue_num_analysts` | `parse_integer` |
| EBITDA Mean | `future_periods[].ebitda_mean` | `parse_dollar_amount` |
| EBITDA Std Dev | `future_periods[].ebitda_std_dev` | `parse_dollar_amount` |
| EBITDA # Analysts | `future_periods[].ebitda_num_analysts` | `parse_integer` |
| EBITDA Margin Mean | `future_periods[].ebitda_margin_mean` | `parse_percentage` |

#### Section 3: Price Targets

| Field | JSON Path | Parse Function |
|-------|-----------|----------------|
| Current Price | `price_targets.current_price` | `parse_float` |
| Mean Target | `price_targets.target_mean` | `parse_float` |
| Low Target | `price_targets.target_low` | `parse_float` |
| High Target | `price_targets.target_high` | `parse_float` |
| Std Dev | `price_targets.target_std_dev` | `parse_float` |
| Upside % | `price_targets.target_upside_pct` | `parse_percentage` |
| # Estimates | `price_targets.num_estimates` | `parse_integer` |

#### Section 4: Analyst Recommendations

| Field | JSON Path | Parse Function |
|-------|-----------|----------------|
| Buy | `recommendations.buy` | `parse_integer` |
| Outperform | `recommendations.outperform` | `parse_integer` |
| Hold | `recommendations.hold` | `parse_integer` |
| Underperform | `recommendations.underperform` | `parse_integer` |
| Sell | `recommendations.sell` | `parse_integer` |
| Consensus Score | `recommendations.consensus_score` | `parse_float` |
| Consensus Rating | `recommendations.consensus_rating` | string |
| Total Analysts | `recommendations.total_analysts` | `parse_integer` |

#### Section 5: Valuation Multiples (from estimates page)

| Field | JSON Path | Parse Function |
|-------|-----------|----------------|
| PE Ratio | `valuation_multiples.pe_ratio` | `parse_float` |
| PE Ratio (Forward) | `valuation_multiples.pe_ratio_forward` | `parse_float` |
| PE Ratio (Forward 1y) | `valuation_multiples.pe_ratio_forward_1y` | `parse_float` |
| PS Ratio | `valuation_multiples.ps_ratio` | `parse_float` |
| PS Ratio (Forward) | `valuation_multiples.ps_ratio_forward` | `parse_float` |
| PS Ratio (Forward 1y) | `valuation_multiples.ps_ratio_forward_1y` | `parse_float` |
| PEG Ratio | `valuation_multiples.peg_ratio` | `parse_float` |

#### Section 6: Calendar

| Field | JSON Path | Parse Function |
|-------|-----------|----------------|
| Next Announcement | `next_announcement` | string (MM/DD/YYYY) |
| Last Announcement | `last_announcement` | string (MM/DD/YYYY) |

### Step 5: Determine as_of_date

Look for an "As of" date at the bottom of the tables (e.g. "As of Feb. 20, 2026"). Convert to YYYY-MM-DD format and use as `as_of_date`. If not found, use today's date.

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
    "source_url": "https://ycharts.com/companies/AAPL/estimates",
    # ... all extracted sections
}

response = requests.post(
    f"https://investorcenter.ai/api/v1/ingest/ycharts/analyst_estimates/{ticker}",
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
ycharts/analyst_estimates/{TICKER}/{YYYY-MM-DD}/{timestamp}.json
```

## API Endpoint

```
POST /api/v1/ingest/ycharts/analyst_estimates/{TICKER}
```

## Summary

1. Open estimates page in openclaw browser
2. Wait 2 seconds, take snapshot
3. Extract all sections (current period, future periods, price targets, recommendations, multiples, calendar)
4. POST to ingestion API with as_of_date
5. Stop browser
6. Report result

**This is a single-page scrape** — no pagination needed. All data is on one page.
