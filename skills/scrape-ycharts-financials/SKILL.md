# YCharts Financials Scraping Skill

**Goal:** Extract historical financial statement data (income statement, balance sheet, or cash flow) from YCharts and upload each period to the data ingestion API.

## Execution Pattern

**This skill is executed directly by the AI agent** (not a separate Python script). The workflow is:
1. Open browser, navigate to YCharts page
2. Capture snapshot (100k chars max)
3. Extract all visible periods (~8 per page) from the snapshot
4. Use exec + Python to parse and POST each period to the API
5. Navigate to next page, repeat
6. Stop browser when done

**Do not overthink this.** It's just web scraping + API calls, page by page.

## Prerequisites

1. **YCharts account login** (already logged in at `easildur24@gmail.com`)
2. **Worker API credentials** in TOOLS.md (`nikola@investorcenter.ai`)
3. **OpenClaw browser profile** (NOT Chrome extension relay!)

## Task Params

This skill is invoked with these params from the task queue:

```json
{
  "ticker": "AAPL",
  "statement": "income_statement",
  "period_type": "quarterly"
}
```

- **ticker**: Stock ticker symbol (e.g. AAPL, MSFT, NVDA)
- **statement**: One of `income_statement`, `balance_sheet`, `cash_flow`
- **period_type**: One of `quarterly`, `annual`, `ttm`

## URL Patterns

| Statement | URL |
|-----------|-----|
| Income Statement | `https://ycharts.com/companies/{TICKER}/financials/income_statement/{page}` |
| Balance Sheet | `https://ycharts.com/companies/{TICKER}/financials/balance_sheet/{page}` |
| Cash Flow | `https://ycharts.com/companies/{TICKER}/financials/cash_flow/{page}` |

Page number starts at 1. Each page shows ~8 periods.

## Workflow

### Step 1: Open YCharts Financials Page

**CRITICAL:** Use `profile="openclaw"` - this is the standalone browser, NOT the Chrome extension relay.

```javascript
browser.open({
  profile: "openclaw",
  targetUrl: "https://ycharts.com/companies/{TICKER}/financials/{STATEMENT_PATH}/1"
})
```

Where `STATEMENT_PATH` maps from `statement` param:
- `income_statement` → `income_statement`
- `balance_sheet` → `balance_sheet`
- `cash_flow` → `cash_flow`

Save the `targetId` for subsequent browser actions.

### Step 2: Select the Correct View

The page defaults to either Annual or Quarterly view depending on the statement. You need to ensure the correct view matching `period_type` is selected.

Look for a **Format** dropdown and select the matching option:
- `quarterly` → "Quarterly"
- `annual` → "Annual"
- `ttm` → "TTM"

Wait 2 seconds after changing the view for the data to reload.

### Step 3: Capture Page Snapshot

```javascript
browser.snapshot({
  profile: "openclaw",
  targetId: "{targetId}",
  maxChars: 100000
})
```

### Step 4: Extract Column Headers (Periods)

The snapshot will show column headers in YYYY-MM format:
```
2025-09, 2024-09, 2023-09, 2022-09, 2021-09, 2020-09, 2019-09, 2018-09
```

Also look for the **Actual Release Date** metadata row (MM/DD/YYYY format).

### Step 5: Extract Data for Each Period

For each period column visible on the page, extract all line item values.

**IMPORTANT:** The page shows ~8 periods at once. You must extract ALL of them before navigating to the next page.

#### Income Statement Fields

| Row Label | JSON Field | Parse Function |
|-----------|-----------|----------------|
| Revenue | `revenue` | `parse_dollar_amount` |
| COGS Incl D&A | `cogs_incl_da` | `parse_dollar_amount` |
| COGS Excl D&A | `cogs_excl_da` | `parse_dollar_amount` |
| Depreciation & Amortization Expense | `depreciation_amortization_expense` | `parse_dollar_amount` |
| Gross Income | `gross_income` | `parse_dollar_amount` |
| SG&A Expense | `sga_expense` | `parse_dollar_amount` |
| R&D Expense | `rd_expense` | `parse_dollar_amount` |
| Other Operating Expense | `other_operating_expense` | `parse_dollar_amount` |
| Operating Income | `operating_income` | `parse_dollar_amount` |
| Non Operating Income/Expense | `non_operating_income_expense` | `parse_dollar_amount` |
| Interest Expense | `interest_expense` | `parse_dollar_amount` |
| Pretax Income | `pretax_income` | `parse_dollar_amount` |
| Income Tax | `income_tax` | `parse_dollar_amount` |
| Income Tax - Current Domestic | `income_tax_current_domestic` | `parse_dollar_amount` |
| Income Tax - Current Foreign | `income_tax_current_foreign` | `parse_dollar_amount` |
| Income Tax - Deferred Domestic | `income_tax_deferred_domestic` | `parse_dollar_amount` |
| Income Tax - Deferred Foreign | `income_tax_deferred_foreign` | `parse_dollar_amount` |
| Income Tax - Credits | `income_tax_credits` | `parse_dollar_amount` |
| Consolidated Net Income Before Non-Controlling Interests | `consolidated_net_income_before_nci` | `parse_dollar_amount` |
| Equity in Net Income | `equity_in_net_income` | `parse_dollar_amount` |
| Consolidated Net Income | `consolidated_net_income` | `parse_dollar_amount` |
| Minority Interest in Earnings | `minority_interest_in_earnings` | `parse_dollar_amount` |
| Net Income | `net_income` | `parse_dollar_amount` |
| EBITDA | `ebitda` | `parse_dollar_amount` |
| EBIT | `ebit` | `parse_dollar_amount` |
| Gross Profit Margin | `gross_profit_margin` | `parse_percentage` |
| EBITDA Margin | `ebitda_margin` | `parse_percentage` |
| EBIT Margin | `ebit_margin` | `parse_percentage` |
| Net Profit Margin | `net_profit_margin` | `parse_percentage` |
| NOPAT | `nopat` | `parse_dollar_amount` |
| NOPAT Margin | `nopat_margin` | `parse_percentage` |
| Net EPS | `net_eps` | `parse_float` |
| EPS - Earnings Per Share (Basic) | `eps_basic` | `parse_float` |
| EPS - Earnings Per Share (Diluted) | `eps_diluted` | `parse_float` |
| EPS - Cash Flow | `eps_cash_flow` | `parse_float` |
| DPS - Dividends Per Share | `dps` | `parse_float` |
| Basic Shares Outstanding | `basic_shares_outstanding` | `parse_dollar_amount` |
| Diluted Shares Outstanding | `diluted_shares_outstanding` | `parse_dollar_amount` |

#### Balance Sheet Fields

| Row Label | JSON Field | Parse Function |
|-----------|-----------|----------------|
| Cash and Short Term Investments | `cash_and_short_term_investments` | `parse_dollar_amount` |
| Accounts Receivable | `accounts_receivable` | `parse_dollar_amount` |
| Other Receivables | `other_receivables` | `parse_dollar_amount` |
| Short-Term Receivables | `short_term_receivables` | `parse_dollar_amount` |
| Inventories | `inventories` | `parse_dollar_amount` |
| Prepaid Expenses | `prepaid_expenses` | `parse_dollar_amount` |
| Miscellaneous Current Assets | `miscellaneous_current_assets` | `parse_dollar_amount` |
| Current Assets, Other | `current_assets_other` | `parse_dollar_amount` |
| Total Current Assets | `total_current_assets` | `parse_dollar_amount` |
| Gross PP&E | `gross_ppe` | `parse_dollar_amount` |
| Accumulated D&A | `accumulated_da` | `parse_dollar_amount` |
| Net PP&E | `net_ppe` | `parse_dollar_amount` |
| Long Term Investments | `long_term_investments` | `parse_dollar_amount` |
| Net Goodwill | `net_goodwill` | `parse_dollar_amount` |
| Net Other Intangibles | `net_other_intangibles` | `parse_dollar_amount` |
| Goodwill and Intangibles | `goodwill_and_intangibles` | `parse_dollar_amount` |
| Deferred Tax Assets | `deferred_tax_assets` | `parse_dollar_amount` |
| Assets - Other | `assets_other` | `parse_dollar_amount` |
| Total Assets | `total_assets` | `parse_dollar_amount` |
| ST Debt & Current Portion Of LT Debt | `st_debt_current_portion_lt_debt` | `parse_dollar_amount` |
| Accounts Payable | `accounts_payable` | `parse_dollar_amount` |
| Current Tax Payable | `current_tax_payable` | `parse_dollar_amount` |
| Other Current Liabilities | `other_current_liabilities` | `parse_dollar_amount` |
| Total Current Liabilities | `total_current_liabilities` | `parse_dollar_amount` |
| Long Term Debt Excl Lease Liab | `long_term_debt_excl_lease_liab` | `parse_dollar_amount` |
| Capital & Operating Lease Oblig | `capital_operating_lease_oblig` | `parse_dollar_amount` |
| Non-Current Portion of Long Term Debt | `non_current_portion_lt_debt` | `parse_dollar_amount` |
| Provisions for Risk and Charges | `provisions_for_risk_and_charges` | `parse_dollar_amount` |
| Long Term Deferred Tax Liabilities | `long_term_deferred_tax_liabilities` | `parse_dollar_amount` |
| Other Liabilities | `other_liabilities` | `parse_dollar_amount` |
| Total Long Term Liabilities | `total_long_term_liabilities` | `parse_dollar_amount` |
| Total Liabilities | `total_liabilities` | `parse_dollar_amount` |
| Non-Equity Reserves | `non_equity_reserves` | `parse_dollar_amount` |
| Preferred Stock | `preferred_stock` | `parse_dollar_amount` |
| Common Equity | `common_equity` | `parse_dollar_amount` |
| Shareholders Equity | `shareholders_equity` | `parse_dollar_amount` |
| Minority Interest Ownership | `minority_interest_ownership` | `parse_dollar_amount` |
| Total Equity Including Minority Interest | `total_equity_incl_minority_interest` | `parse_dollar_amount` |
| Total Liabilities And Shareholders' Equity | `total_liabilities_and_shareholders_equity` | `parse_dollar_amount` |
| Book Value Per Share | `book_value_per_share` | `parse_float` |
| Tangible Book Value Per Share | `tangible_book_value_per_share` | `parse_float` |
| Working Capital - Total (Net Working Capital) | `working_capital` | `parse_dollar_amount` |
| Total Capital, Including Short-Term Debt | `total_capital_incl_st_debt` | `parse_dollar_amount` |
| Total Debt | `total_debt` | `parse_dollar_amount` |
| Net Debt | `net_debt` | `parse_dollar_amount` |
| Minimum Pension Liabilities | `minimum_pension_liabilities` | `parse_dollar_amount` |
| Comprehensive Income - Hedging Gain/Loss | `comprehensive_income_hedging_gain_loss` | `parse_dollar_amount` |
| Comprehensive Income - Unearned Comp | `comprehensive_income_unearned_comp` | `parse_dollar_amount` |
| Stock-Based Compensation Expense | `stock_based_compensation_expense` | `parse_dollar_amount` |
| Stock-Based Comp Adj to Net Income Q | `stock_based_comp_adj_to_net_income` | `parse_dollar_amount` |
| Total Operating Lease Commitments | `total_operating_lease_commitments` | `parse_dollar_amount` |
| Current Portion of Long Term Debt | `current_portion_of_lt_debt` | `parse_dollar_amount` |

#### Cash Flow Fields

| Row Label | JSON Field | Parse Function |
|-----------|-----------|----------------|
| Net Income | `net_income` | `parse_dollar_amount` |
| Total Depreciation and Amortization | `total_depreciation_and_amortization` | `parse_dollar_amount` |
| Deferred Taxes & Investment Tax Credit | `deferred_taxes_investment_tax_credit` | `parse_dollar_amount` |
| Operating Interest Paid - Lease Liab | `operating_interest_paid_lease_liab` | `parse_dollar_amount` |
| Non-Cash Items | `non_cash_items` | `parse_dollar_amount` |
| Funds From Operations | `funds_from_operations` | `parse_dollar_amount` |
| Extraordinary Items | `extraordinary_items` | `parse_dollar_amount` |
| Changes in Working Capital | `changes_in_working_capital` | `parse_dollar_amount` |
| Cash from Operations | `cash_from_operations` | `parse_dollar_amount` |
| Capital Expenditures - Total | `capital_expenditures_total` | `parse_dollar_amount` |
| Net Divestitures (Acquisitions) | `net_divestitures_acquisitions` | `parse_dollar_amount` |
| Sale of Fixed Assets & Businesses | `sale_of_fixed_assets_businesses` | `parse_dollar_amount` |
| Total Net Change in Investments | `total_net_change_in_investments` | `parse_dollar_amount` |
| Other Funds | `other_funds_investing` | `parse_dollar_amount` |
| Cash from Investing | `cash_from_investing` | `parse_dollar_amount` |
| Total Dividends Paid | `total_dividends_paid` | `parse_dollar_amount` |
| Net Common Equity Issued (Purchased) | `net_common_equity_issued_purchased` | `parse_dollar_amount` |
| Net Debt Issuance | `net_debt_issuance` | `parse_dollar_amount` |
| Repayments Of Operating Lease Liabilities | `repayments_of_operating_lease_liabilities` | `parse_dollar_amount` |
| Financing Interest Paid - Lease Liab | `financing_interest_paid_lease_liab` | `parse_dollar_amount` |
| Other Funds - Financing | `other_funds_financing` | `parse_dollar_amount` |
| Cash from Financing | `cash_from_financing` | `parse_dollar_amount` |
| Beginning Cash | `beginning_cash` | `parse_dollar_amount` |
| Exchange Rate Effect | `exchange_rate_effect` | `parse_dollar_amount` |
| Net Change in Cash | `net_change_in_cash` | `parse_dollar_amount` |
| Ending Cash | `ending_cash` | `parse_dollar_amount` |
| Free Cash Flow | `free_cash_flow` | `parse_dollar_amount` |
| Interest Paid, Operating and Financing | `interest_paid_operating_and_financing` | `parse_dollar_amount` |
| Capital and Operating Lease Obligations | `capital_and_operating_lease_obligations` | `parse_dollar_amount` |
| Income Tax Paid | `income_tax_paid` | `parse_dollar_amount` |
| Stock Based Compensation | `stock_based_compensation` | `parse_dollar_amount` |

### Step 6: Ingest Each Period

For each period extracted from the page, POST to the ingestion API:

```python
import requests
from datetime import datetime

# Get auth token
response = requests.post(
    "https://investorcenter.ai/api/v1/auth/login",
    json={"email": "nikola@investorcenter.ai", "password": "ziyj9VNdHH5tjqB2m3lup3MG"}
)
token = response.json()["access_token"]

# For each period on the page:
period = "2025-09"  # from column header
period_data = {
    "period": period,
    "period_type": "quarterly",  # from task params
    "release_date": "10/30/2025",  # from Actual Release Date row, or null
    "source_url": "https://ycharts.com/companies/AAPL/financials/income_statement/1",
    # ... all line item fields with parsed values
    "revenue": 94930000000,
    "cogs_incl_da": 52547000000,
    # etc.
}

response = requests.post(
    f"https://investorcenter.ai/api/v1/ingest/ycharts/financials/{statement}/{ticker}",
    headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
    json=period_data
)

if response.status_code == 201:
    result = response.json()
    print(f"  Period {period}: s3_key={result['data']['s3_key']}")
else:
    print(f"  Period {period}: ERROR {response.status_code} - {response.text}")
```

### Step 7: Navigate to Next Page

After ingesting all periods on the current page, check if there are more periods:

1. Look for **Next Period** or pagination controls in the snapshot
2. If more pages exist, navigate to page 2, 3, etc. by incrementing the URL page number
3. Repeat Steps 3-6 for each page
4. Stop when there are no more pages (no Next Period button, or page returns same data)

### Step 8: Close Browser

**CRITICAL:** Stop the browser after completing all pages:

```javascript
browser({
  action: "stop",
  profile: "openclaw"
})
```

## Parse Helpers Reference

Located at: `skills/scrape-ycharts-keystats/parse_helpers.py`

**Reuse the same parse helpers from key_stats.** Import from that path.

```python
sys.path.append('/Users/larryli/.openclaw/workspace/investorcenter.ai/skills/scrape-ycharts-keystats')
from parse_helpers import parse_dollar_amount, parse_percentage, parse_float, parse_integer
```

### Functions

```python
parse_dollar_amount("391.04B")  # 391040000000
parse_dollar_amount("2.87B")    # 2870000000
parse_dollar_amount("-154.00M") # -154000000
parse_dollar_amount("--")       # None

parse_percentage("46.40%")      # 0.464
parse_percentage("--")          # None

parse_float("6.11")             # 6.11
parse_float("--")               # None
```

## Data Format Notes

- **Dollar amounts**: Suffixed with B (billions) or M (millions), can be negative
- **Percentages**: Include "%" symbol, stored as decimals
- **Per-share metrics**: Float format without suffix
- **Share counts**: Large numbers with B/M suffix
- **Missing data**: `--` becomes `null`/`None`
- **Zero values**: `0.00` becomes `0`

## S3 Storage Path

Data is stored at:
```
s3://investorcenter-raw-data/ycharts/financials/{statement}/{TICKER}/{period_type}/{period}.json
```

Example:
```
ycharts/financials/income_statement/AAPL/quarterly/2025-09.json
ycharts/financials/balance_sheet/AAPL/quarterly/2025-12.json
ycharts/financials/cash_flow/TSLA/annual/2024-12.json
```

Duplicate uploads overwrite (idempotent).

## API Endpoint

```
POST /api/v1/ingest/ycharts/financials/{statement}/{ticker}
```

Where `{statement}` is one of: `income_statement`, `balance_sheet`, `cash_flow`

## Error Handling

- If a page fails to load, wait 5 seconds and retry (up to 3 times)
- If a specific period fails to ingest, log the error and continue with other periods
- At the end, report how many periods were successfully ingested vs failed
- If the view dropdown doesn't match the expected period_type, log a warning

## Summary

1. Open financials page in openclaw browser
2. Select correct view (quarterly/annual/ttm)
3. Take snapshot
4. Extract all periods and their line item values
5. POST each period to ingestion API
6. Navigate to next page if available
7. Repeat until all pages exhausted
8. Stop browser
9. Report results (X periods ingested for {TICKER} {statement} {period_type})

**Key differences from key_stats:**
- Multiple periods per page (not a single snapshot)
- Pagination across multiple pages
- Each period is a separate API call
- Three different field sets depending on statement type
- Historical data, not point-in-time
