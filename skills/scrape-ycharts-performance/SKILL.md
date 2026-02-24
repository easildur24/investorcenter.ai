# YCharts Performance Scraping Skill

**Goal:** Extract stock performance, returns, risk metrics, and peer comparisons from YCharts performance page and upload to data ingestion API.

## Execution Pattern

**This skill is executed directly by the AI agent.** The workflow is:
1. Open browser, navigate to YCharts performance page
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
https://ycharts.com/companies/{TICKER}/performance/price
```

## Workflow

### Step 1: Open YCharts Performance Page

**CRITICAL:** Use `profile="openclaw"` - this is the standalone browser, NOT the Chrome extension relay.

```javascript
browser.open({
  profile: "openclaw",
  targetUrl: "https://ycharts.com/companies/{TICKER}/performance/price"
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

The page has multiple sections. Extract all of them.

#### Section 1: Periodic Total Returns

| Time Period | JSON Field | Parse Function |
|-------------|-----------|----------------|
| 1 Month Total Returns | `periodic_returns.return_1m` | `parse_percentage` |
| 3 Month Total Returns | `periodic_returns.return_3m` | `parse_percentage` |
| 6 Month Total Returns | `periodic_returns.return_6m` | `parse_percentage` |
| 1 Year Total Returns | `periodic_returns.return_1y` | `parse_percentage` |
| 3 Year Annualized Total Returns | `periodic_returns.return_3y` | `parse_percentage` |
| 5 Year Annualized Total Returns | `periodic_returns.return_5y` | `parse_percentage` |
| 10 Year Annualized Total Returns | `periodic_returns.return_10y` | `parse_percentage` |
| 15 Year Annualized Total Returns | `periodic_returns.return_15y` | `parse_percentage` |
| 20 Year Annualized Total Returns | `periodic_returns.return_20y` | `parse_percentage` |
| All-Time Annualized Total Returns | `periodic_returns.return_all_time` | `parse_percentage` |

#### Section 2: Annual Returns

Extract as array `annual_returns`. Each year shows the total return for that calendar year.

```json
[
  {"year": 2025, "total_return": 0.3040},
  {"year": 2024, "total_return": 0.3402},
  {"year": 2023, "total_return": 0.4906},
  ...
]
```

#### Section 3: Benchmark Comparison

The page compares the stock vs S&P 500 Total Return (or similar benchmark).

| Field | JSON Path | Parse Function |
|-------|-----------|----------------|
| Benchmark Name | `benchmark_comparison.benchmark_name` | string |
| 1M Return | `benchmark_comparison.benchmark_return_1m` | `parse_percentage` |
| 3M Return | `benchmark_comparison.benchmark_return_3m` | `parse_percentage` |
| 6M Return | `benchmark_comparison.benchmark_return_6m` | `parse_percentage` |
| 1Y Return | `benchmark_comparison.benchmark_return_1y` | `parse_percentage` |
| 3Y Return | `benchmark_comparison.benchmark_return_3y` | `parse_percentage` |
| 5Y Return | `benchmark_comparison.benchmark_return_5y` | `parse_percentage` |
| 10Y Return | `benchmark_comparison.benchmark_return_10y` | `parse_percentage` |
| 15Y Return | `benchmark_comparison.benchmark_return_15y` | `parse_percentage` |
| 20Y Return | `benchmark_comparison.benchmark_return_20y` | `parse_percentage` |
| All-Time | `benchmark_comparison.benchmark_return_all_time` | `parse_percentage` |

#### Section 4: Benchmark Annual Comparison

Year-by-year returns for both the stock and benchmark. Extract as array `benchmark_annual_comparison`:

```json
[
  {"year": 2025, "stock_return": 0.3040, "benchmark_return": 0.2302, "excess_return": 0.0738},
  {"year": 2024, "stock_return": 0.3402, "benchmark_return": 0.2502, "excess_return": 0.0900},
  ...
]
```

#### Section 5: Risk Metrics

| Field | JSON Path | Parse Function |
|-------|-----------|----------------|
| Alpha (3Y) | `risk_metrics.alpha_3y` | `parse_float` |
| Beta (3Y) | `risk_metrics.beta_3y` | `parse_float` |
| Beta (5Y) | `risk_metrics.beta_5y` | `parse_float` |
| Sharpe Ratio (3Y) | `risk_metrics.sharpe_ratio_3y` | `parse_float` |
| Sortino Ratio (3Y) | `risk_metrics.sortino_ratio_3y` | `parse_float` |
| Max Drawdown (3Y) | `risk_metrics.max_drawdown_3y` | `parse_percentage` |
| Max Drawdown (5Y) | `risk_metrics.max_drawdown_5y` | `parse_percentage` |
| Std Dev Monthly (3Y) | `risk_metrics.std_dev_monthly_3y` | `parse_percentage` |
| VaR 5% Monthly (3Y) | `risk_metrics.var_5pct_monthly_3y` | `parse_percentage` |

#### Section 6: Peer Comparison (if available)

The page may show peer companies with their returns. Extract as array `peer_comparison`:

```json
[
  {"ticker": "MSFT", "name": "Microsoft", "return_1m": 0.05, "return_3m": 0.12, ...},
  {"ticker": "GOOGL", "name": "Alphabet", "return_1m": 0.03, ...},
  ...
]
```

Fields per peer: `ticker`, `name`, `return_1m`, `return_3m`, `return_6m`, `return_1y`, `return_3y`, `return_5y`, `return_10y`, `return_15y`, `return_20y`, `return_all_time`

#### Section 7: Peer Annual Comparison (if available)

Year-by-year returns for each peer. Extract as array `peer_annual_comparison`:

```json
[
  {"ticker": "AAPL", "name": "Apple", "year_returns": {"2025": 0.3040, "2024": 0.3402, ...}},
  {"ticker": "MSFT", "name": "Microsoft", "year_returns": {"2025": 0.1234, ...}},
  ...
]
```

### Step 5: Determine as_of_date

Look for a date reference on the page. If not found, use today's date in YYYY-MM-DD format.

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
    "source_url": "https://ycharts.com/companies/AAPL/performance/price",
    "periodic_returns": { ... },
    "annual_returns": [ ... ],
    "benchmark_comparison": { ... },
    "benchmark_annual_comparison": [ ... ],
    "risk_metrics": { ... },
    "peer_comparison": [ ... ],
    "peer_annual_comparison": [ ... ]
}

response = requests.post(
    f"https://investorcenter.ai/api/v1/ingest/ycharts/performance/{ticker}",
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
ycharts/performance/{TICKER}/{YYYY-MM-DD}/{timestamp}.json
```

## API Endpoint

```
POST /api/v1/ingest/ycharts/performance/{TICKER}
```

## Summary

1. Open performance page in openclaw browser
2. Wait 2 seconds, take snapshot
3. Extract all sections (periodic returns, annual returns, benchmark, risk metrics, peers)
4. POST to ingestion API with as_of_date
5. Stop browser
6. Report result

**This is a single-page scrape** — no pagination needed. All data is on one page.
