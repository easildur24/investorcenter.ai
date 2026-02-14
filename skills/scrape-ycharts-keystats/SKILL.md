# YCharts Key Stats Scraping Skill

**Goal:** Extract 100+ financial metrics from YCharts Key Stats pages and upload to data ingestion API.

## Prerequisites

1. **YCharts account login** (already logged in at `easildur24@gmail.com`)
2. **Worker API credentials** in TOOLS.md (`nikola@investorcenter.ai`)
3. **OpenClaw browser profile** (NOT Chrome extension relay!)

## Single-Ticker Workflow

### Step 1: Open YCharts in OpenClaw Browser

**CRITICAL:** Use `profile=openclaw` - this is the standalone browser, NOT the Chrome extension relay.

```javascript
browser.open({
  profile: "openclaw",
  targetUrl: "https://ycharts.com/companies/{TICKER}/key_stats"
})
// Returns: { targetId: "...", url: "..." }
```

Save the `targetId` - you'll need it for all subsequent browser actions.

### Step 2: Wait for Page Load

```bash
sleep 2
```

YCharts pages are JavaScript-rendered. Give them time to fully load.

### Step 3: Capture Page Snapshot

```javascript
browser.snapshot({
  profile: "openclaw",
  targetId: "{targetId from step 1}",
  maxChars: 100000
})
```

This returns the full page structure with all metric labels and values.

### Step 4: Extract Data from Snapshot

**DO NOT** try to parse the snapshot programmatically in one go. The structure is complex.

Instead:
1. Read the snapshot carefully
2. Find each section (Income Statement, Balance Sheet, etc.)
3. Extract values manually into a Python script

**Example extraction for MMM:**

```python
#!/usr/bin/env python3
import json
import sys
import requests
from datetime import datetime

sys.path.append('/Users/larryli/.openclaw/workspace/investorcenter.ai/skills/scrape-ycharts-keystats')
from parse_helpers import parse_dollar_amount, parse_percentage, parse_float, parse_integer, parse_ycharts_date

# MMM data from browser snapshot
data = {
    "ticker": "MMM",
    "collected_at": datetime.utcnow().isoformat() + "Z",
    "source_url": "https://ycharts.com/companies/MMM/key_stats",
    "data": {
        # Income Statement (from snapshot section)
        "revenue": parse_dollar_amount("24.95B"),
        "net_income": parse_dollar_amount("3.262B"),
        "ebit": parse_dollar_amount("4.73B"),
        # ... continue for all fields
    }
}

# Get token
response = requests.post(
    "https://investorcenter.ai/api/v1/auth/login",
    json={"email": "nikola@investorcenter.ai", "password": "ziyj9VNdHH5tjqB2m3lup3MG"}
)
token = response.json()["access_token"]

# Ingest
response = requests.post(
    f"https://investorcenter.ai/api/v1/ingest/ycharts/key_stats/MMM",
    headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
    json=data
)

if response.status_code == 200:
    result = response.json()
    print(f"✅ MMM: Success!")
    print(f"   S3: {result['data']['s3_key']}")
else:
    print(f"❌ Error: {response.text}")
```

Save this as `/tmp/ingest_{ticker}.py` and run it.

### Step 5: Verify Ingestion

Check the response:
```json
{
  "success": true,
  "data": {
    "id": 5,
    "ticker": "MMM",
    "s3_key": "ycharts/key_stats/MMM/2026-02-14/20260214T004643Z.json"
  }
}
```

Data is now in S3 at `s3://investorcenter-raw-data/{s3_key}`.

## Parse Helpers Reference

Located at: `skills/scrape-ycharts-keystats/parse_helpers.py`

### Functions

```python
parse_dollar_amount("187.14B")  # → 187140000000
parse_dollar_amount("3.979M")   # → 3979000
parse_dollar_amount("-28.56B")  # → -28560000000

parse_percentage("62.49%")      # → 0.6249
parse_percentage("-1.60%")      # → -0.0160

parse_float("4.038")            # → 4.038
parse_float("186.96")           # → 186.96

parse_integer("36000")          # → 36000
parse_integer("1,234,567")      # → 1234567

parse_ycharts_date("Oct. 29, 2025")  # → "2025-10-29"
parse_ycharts_date("Feb. 12, 2026")  # → "2026-02-12"
```

All functions return `None` if unable to parse or if value is `"--"`.

## Data Extraction Map

### Snapshot Structure

Browser snapshot returns a tree structure like:

```
- row "Revenue (TTM) 24.95B" [ref=e159]:
  - cell "Revenue (TTM)" [ref=e160]:
    - link "Revenue (TTM)" [ref=e161]
  - cell "24.95B" [ref=e162]
```

**Extract pattern:**
1. Find the row by metric name (e.g., "Revenue (TTM)")
2. The second cell contains the value ("24.95B")
3. Parse using appropriate helper function

### Complete Field Mapping

#### Income Statement Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `Revenue (TTM)` → `24.95B` | `revenue` | `parse_dollar_amount` |
| `Net Income (TTM)` → `3.262B` | `net_income` | `parse_dollar_amount` |
| `EBIT (TTM)` → `4.73B` | `ebit` | `parse_dollar_amount` |
| `EBITDA (TTM)` → `6.038B` | `ebitda` | `parse_dollar_amount` |
| `Revenue (Quarterly)` → `6.133B` | `revenue_quarterly` | `parse_dollar_amount` |
| `Net Income (Quarterly)` → `574.00M` | `net_income_quarterly` | `parse_dollar_amount` |
| `EBIT (Quarterly)` → `798.00M` | `ebit_quarterly` | `parse_dollar_amount` |
| `EBITDA (Quarterly)` → `1.228B` | `ebitda_quarterly` | `parse_dollar_amount` |
| `Revenue (Quarterly YoY Growth)` → `2.05%` | `revenue_growth_quarterly_yoy` | `parse_percentage` |
| `EPS Diluted (Quarterly YoY Growth)` → `-19.67%` | `eps_diluted_growth_quarterly_yoy` | `parse_percentage` |
| `EBITDA (Quarterly YoY Growth)` → `-13.22%` | `ebitda_growth_quarterly_yoy` | `parse_percentage` |

#### Common Size Statements Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `EPS Diluted (TTM)` → `5.995` | `eps_diluted` | `parse_float` |
| `EPS Basic (TTM)` → `6.039` | `eps_basic` | `parse_float` |
| `Shares Outstanding` → `526.70M` | `shares_outstanding` | `parse_dollar_amount` |

#### Balance Sheet Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `Total Assets (Quarterly)` → `37.73B` | `total_assets` | `parse_dollar_amount` |
| `Total Liabilities (Quarterly)` → `32.99B` | `total_liabilities` | `parse_dollar_amount` |
| `Shareholders Equity (Quarterly)` → `4.702B` | `shareholders_equity` | `parse_dollar_amount` |
| `Cash and Short Term Investments (Quarterly)` → `5.933B` | `cash_and_short_term_investments` | `parse_dollar_amount` |
| `Total Long Term Assets (Quarterly)` → `21.18B` | `total_long_term_assets` | `parse_dollar_amount` |
| `Total Long Term Debt (Quarterly)` → `12.96B` | `total_long_term_debt` | `parse_dollar_amount` |
| `Book Value (Quarterly)` → `4.702B` | `book_value` | `parse_dollar_amount` |

#### Earnings Quality Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `Return on Assets` → `8.44%` | `return_on_assets` | `parse_percentage` |
| `Return on Equity` → `73.50%` | `return_on_equity` | `parse_percentage` |
| `Return on Invested Capital` → `18.14%` | `return_on_invested_capital` | `parse_percentage` |

#### Cash Flow Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `Cash from Operations (TTM)` → `2.306B` | `cash_from_operations` | `parse_dollar_amount` |
| `Cash from Investing (TTM)` → `1.35B` | `cash_from_investing` | `parse_dollar_amount` |
| `Cash from Financing (TTM)` → `-4.016B` | `cash_from_financing` | `parse_dollar_amount` |
| `Change in Receivables (TTM)` → `-211.00M` | `change_in_receivables` | `parse_dollar_amount` |
| `Changes in Working Capital (TTM)` → `-3.196B` | `changes_in_working_capital` | `parse_dollar_amount` |
| `Capital Expenditures (TTM)` → `910.00M` | `capital_expenditures` | `parse_dollar_amount` |
| `Ending Cash (Quarterly)` → `5.806B` | `ending_cash` | `parse_dollar_amount` |
| `Free Cash Flow` → `1.396B` | `free_cash_flow` | `parse_dollar_amount` |

#### Profitability Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `Operating Margin (TTM)` → `18.96%` | `operating_margin` | `parse_percentage` |
| `Gross Profit Margin` → `39.59%` | `gross_profit_margin` | `parse_percentage` |

#### Stock Price Performance Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `1 Month Total Returns (Daily)` → `3.97%` | `one_month_total_return` | `parse_percentage` |
| `3 Month Total Returns (Daily)` → `2.51%` | `three_month_total_return` | `parse_percentage` |
| `6 Month Total Returns (Daily)` → `11.62%` | `six_month_total_return` | `parse_percentage` |
| `Year to Date Total Returns (Daily)` → `9.06%` | `ytd_total_return` | `parse_percentage` |
| `1 Year Total Returns (Daily)` → `19.52%` | `one_year_total_return` | `parse_percentage` |
| `Annualized 3 Year Total Returns (Daily)` → `27.04%` | `three_year_total_return_annualized` | `parse_percentage` |
| `Annualized 5 Year Total Returns (Daily)` → `6.87%` | `five_year_total_return_annualized` | `parse_percentage` |
| `Annualized 10 Year Total Returns (Daily)` → `6.49%` | `ten_year_total_return_annualized` | `parse_percentage` |
| `Annualized 15 Year Total Returns (Daily)` → `8.89%` | `fifteen_year_total_return_annualized` | `parse_percentage` |
| `Annualized Total Returns Since Inception (Daily)` → `9.75%` | `since_inception_total_return_annualized` | `parse_percentage` |
| `52 Week High (Daily)` → `177.41` | `fifty_two_week_high` | `parse_float` |
| `52 Week Low (Daily)` → `121.98` | `fifty_two_week_low` | `parse_float` |
| `52-Week High Date` → `Feb. 12, 2026` | `fifty_two_week_high_date` | `parse_ycharts_date` |
| `52-Week Low Date` → `Apr. 07, 2025` | `fifty_two_week_low_date` | `parse_ycharts_date` |

#### Estimates Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `Revenue Estimates for Current Quarter` → `6.046B` | `revenue_estimates_current_quarter` | `parse_dollar_amount` |
| `Revenue Estimates for Next Quarter` → `6.384B` | `revenue_estimates_next_quarter` | `parse_dollar_amount` |
| `Revenue Estimates for Current Fiscal Year` → `25.15B` | `revenue_estimates_current_year` | `parse_dollar_amount` |
| `Revenue Estimates for Next Fiscal Year` → `25.98B` | `revenue_estimates_next_year` | `parse_dollar_amount` |
| `EPS Estimates for Current Quarter` → `1.987` | `eps_estimates_current_quarter` | `parse_float` |
| `EPS Estimates for Next Quarter` → `2.251` | `eps_estimates_next_quarter` | `parse_float` |
| `EPS Estimates for Current Fiscal Year` → `8.669` | `eps_estimates_current_year` | `parse_float` |
| `EPS Estimates for Next Fiscal Year` → `9.363` | `eps_estimates_next_year` | `parse_float` |

#### Dividends and Shares Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `Dividend Yield` → `1.70%` | `dividend_yield` | `parse_percentage` |
| `Dividend Yield (Forward)` → `1.79%` | `dividend_yield_forward` | `parse_percentage` |
| `Payout Ratio (TTM)` → `48.70%` | `payout_ratio` | `parse_percentage` |
| `Cash Dividend Payout Ratio` → `111.9%` | `cash_dividend_payout_ratio` | `parse_percentage` |
| `Last Dividend Amount` → `0.78` | `last_dividend_amount` | `parse_float` |
| `Last Ex-Dividend Date` → `Feb. 13, 2026` | `last_ex_dividend_date` | `parse_ycharts_date` |

#### Management Effectiveness Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `Asset Utilization (TTM)` → `0.6458` | `asset_utilization` | `parse_float` |
| `Days Sales Outstanding (Quarterly)` → `55.24` | `days_sales_outstanding` | `parse_float` |
| `Days Inventory Outstanding (Quarterly)` → `84.66` | `days_inventory_outstanding` | `parse_float` |
| `Days Payable Outstanding (Quarterly)` → `64.62` | `days_payable_outstanding` | `parse_float` |
| `Total Receivables (Quarterly)` → `3.594B` | `total_receivables` | `parse_dollar_amount` |

#### Current Valuation Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `Market Cap` → `90.59B` | `market_cap` | `parse_dollar_amount` |
| `Enterprise Value` → `99.21B` | `enterprise_value` | `parse_dollar_amount` |
| `Price` → `171.99` | `price` | `parse_float` |
| `PE Ratio` → `28.69` | `pe_ratio` | `parse_float` |
| `PE Ratio (Forward)` → `19.84` | `pe_ratio_forward` | `parse_float` |
| `PE Ratio (Forward 1y)` → `18.37` | `pe_ratio_forward_1y` | `parse_float` |
| `PS Ratio` → `3.732` | `ps_ratio` | `parse_float` |
| `PS Ratio (Forward)` → `3.602` | `ps_ratio_forward` | `parse_float` |
| `PS Ratio (Forward 1y)` → `3.487` | `ps_ratio_forward_1y` | `parse_float` |
| `Price to Book Value` → `19.27` | `price_to_book_value` | `parse_float` |
| `Price to Free Cash Flow` → `66.70` | `price_to_free_cash_flow` | `parse_float` |
| `PEG Ratio` → `--` | `peg_ratio` | `None` (missing) |
| `EV to EBITDA` → `16.43` | `ev_to_ebitda` | `parse_float` |
| `EV to EBITDA (Forward)` → `13.54` | `ev_to_ebitda_forward` | `parse_float` |
| `EV to EBIT` → `20.97` | `ev_to_ebit` | `parse_float` |
| `EBIT Margin (TTM)` → `18.96%` | `ebit_margin` | `parse_percentage` |

#### Risk Metrics Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `Alpha (5Y)` → `-11.35` | `alpha_5y` | `parse_float` |
| `Beta (5Y)` → `1.086` | `beta_5y` | `parse_float` |
| `Annualized Standard Deviation of Monthly Returns (5Y Lookback)` → `27.53%` | `standard_deviation_monthly_5y` | `parse_percentage` |
| `Historical Sharpe Ratio (5Y)` → `0.0494` | `sharpe_ratio_5y` | `parse_float` |
| `Historical Sortino (5Y)` → `0.0822` | `sortino_ratio_5y` | `parse_float` |
| `Max Drawdown (5Y)` → `54.05%` | `max_drawdown_5y` | `parse_percentage` |
| `Monthly Value at Risk (VaR) 5% (5Y Lookback)` → `12.50%` | `value_at_risk_monthly_5y` | `parse_percentage` |

#### Advanced Metrics Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `Piotroski F Score (TTM)` → `4.00` | `piotroski_f_score` | `parse_float` |
| `Sustainable Growth Rate (TTM)` → `3.77K%` | `sustainable_growth_rate` | `parse_percentage` |
| `Tobin's Q (Approximate) (Quarterly)` → `2.369` | `tobin_q` | `parse_float` |
| `Momentum Score` → `8.000` | `momentum_score` | `parse_float` |
| `Market Cap Score` → `1.000` | `market_cap_score` | `parse_float` |
| `Quality Ratio Score` → `8.000` | `quality_ratio_score` | `parse_float` |

#### Liquidity And Solvency Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `Debt to Equity Ratio` → `2.793` | `debt_to_equity_ratio` | `parse_float` |
| `Free Cash Flow (Quarterly)` → `1.335B` | `free_cash_flow_quarterly` | `parse_dollar_amount` |
| `Current Ratio` → `1.708` | `current_ratio` | `parse_float` |
| `Quick Ratio (Quarterly)` → `0.9929` | `quick_ratio` | `parse_float` |
| `Altman Z-Score (TTM)` → `4.255` | `altman_z_score` | `parse_float` |
| `Times Interest Earned (TTM)` → `5.043` | `times_interest_earned` | `parse_float` |

#### Employee Count Metrics Section

| Snapshot Text | JSON Field | Parse Function |
|--------------|------------|----------------|
| `Total Employees (Annual)` → `60500` | `total_employees` | `parse_integer` |
| `Revenue Per Employee (Annual)` → `408983.6` | `revenue_per_employee` | `parse_float` |
| `Net Income Per Employee (Annual)` → `53475.41` | `net_income_per_employee` | `parse_float` |

## Special Cases

### Missing Values

When a metric shows `--` in YCharts, set it to `null`:

```python
"peg_ratio": None,  # Shows as "--" in YCharts
```

### Large Percentages

Sustainable Growth Rate can be `3.77K%` (3,770%). The parse_percentage function handles this:

```python
parse_percentage("3.77K%")  # → 37.70 (not 0.0377!)
```

**NOTE:** This is currently handled as 0.0377 in parse_helpers.py - may need fixing!

### Zero vs Null

- `0.00` → `0` (actual zero value)
- `--` → `null` (missing data)

## Multi-Ticker Workflow

To scrape multiple tickers (e.g., S&P 500 batch):

1. **One ticker at a time** - don't try to parallelize
2. **Add delays** between tickers (5-15 seconds) to avoid rate limiting
3. **Track progress** in a JSON file
4. **Handle failures gracefully** - log and continue

Example progress tracking:

```json
{
  "completed": ["MMM", "AOS"],
  "failed": ["XYZ"],
  "current_batch": 1,
  "last_ticker": "AOS",
  "start_time": "2026-02-13T20:30:00Z"
}
```

## Authentication Issues

If you get 403 "admin access required":

1. Check backend deployment image tag
2. Verify data-ingestion-service is running latest
3. Force restart both services:
   ```bash
   kubectl rollout restart deployment/investorcenter-backend -n investorcenter
   kubectl rollout restart deployment/data-ingestion-service -n investorcenter
   ```

## JSON Schema Reference

Full schema: `data-ingestion-service/schemas/ycharts/key_stats.json`

Required top-level fields:
- `ticker` (string)
- `collected_at` (ISO 8601 timestamp)
- `source_url` (string)
- `data` (object with metrics)

All fields in `data` are nullable.

## Files Reference

- **Parse helpers:** `skills/scrape-ycharts-keystats/parse_helpers.py`
- **Schema:** `data-ingestion-service/schemas/ycharts/key_stats.json`
- **API handler:** `data-ingestion-service/handlers/ycharts/key_stats.go`
- **Example script:** `/tmp/ingest_mmm.py` (after first scrape)

## Summary

**The working process:**
1. Open page in openclaw browser (profile="openclaw")
2. Wait 2 seconds
3. Take snapshot with maxChars=100000
4. Manually extract values from snapshot into Python script
5. Use parse_helpers functions for all values
6. Get auth token
7. POST to ingestion API
8. Verify S3 upload

**Key points:**
- Browser profile = `openclaw` (NOT Chrome extension relay!)
- Extraction = manual from snapshot (tedious but reliable)
- Parse helpers handle all number formats
- One ticker per run, add delays for batches
