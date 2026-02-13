# Scrape YCharts Key Stats Skill

Extract 100+ financial metrics from YCharts Key Stats page and upload to the data ingestion API.

## Task Format

When assigned a task like:
- "Scrape key stats for NVDA from YCharts"
- "Get YCharts key stats for AAPL, MSFT, GOOGL"

## Workflow

1. **Authenticate with API**
   ```bash
   TOKEN=$(curl -s -X POST https://investorcenter.ai/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d "{\"email\": \"$CLAWDBOT_EMAIL\", \"password\": \"$CLAWDBOT_PASSWORD\"}" \
     | jq -r .access_token)
   ```

2. **Navigate to Key Stats page**
   - URL: `https://ycharts.com/companies/{TICKER}/key_stats/stats`
   - Wait for page to fully load (JavaScript-rendered content)

3. **Extract data** from page sections (see extraction guide below)

4. **Upload to API**
   ```bash
   POST https://investorcenter.ai/api/v1/ingest/ycharts/key_stats/{TICKER}
   Authorization: Bearer $TOKEN
   Content-Type: application/json
   
   {
     "collected_at": "2026-02-12T20:30:00Z",
     "source_url": "https://ycharts.com/companies/NVDA/key_stats/stats",
     "price": { ... },
     "income_statement": { ... },
     ...
   }
   ```

5. **Report completion**
   - Log the ingestion ID from response
   - Report any missing/null fields

## Data Extraction Guide

### Page Structure

YCharts Key Stats page has **3 main sections**:

1. **Financials** (left column)
   - Income Statement
   - Common Size Statements
   - Balance Sheet
   - Earnings Quality
   - Cash Flow
   - Profitability

2. **Performance, Risk and Estimates** (middle column)
   - Stock Price Performance
   - Estimates
   - Dividends and Shares
   - Management Effectiveness
   - Current Valuation
   - Risk Metrics

3. **Other Metrics** (right column)
   - Advanced Metrics
   - Liquidity And Solvency
   - Employee Count Metrics

### Extraction Methods

**Option A: Browser snapshot + manual parsing**
```javascript
// Use browser.snapshot() to get page structure
// Then parse values from the refs
```

**Option B: Scrape HTML directly**
```python
# Extract table rows with metric names and values
# Map to JSON schema fields
```

**Option C: Use browser evaluate**
```javascript
// Inject JavaScript to extract all metrics
const metrics = {};
document.querySelectorAll('table tr').forEach(row => {
  const label = row.querySelector('td:first-child')?.innerText;
  const value = row.querySelector('td:last-child')?.innerText;
  metrics[label] = parseValue(value);
});
return metrics;
```

## Field Mapping

### Price Section (top of page)

| YCharts Display | JSON Field | Parse Rule |
|----------------|------------|------------|
| "186.96" | `price.current` | Float |
| "USD \| NASDAQ \| Feb 12, 16:00" | `price.currency`, `price.exchange`, `price.timestamp` | Split on "\|" |

### Income Statement Section

| YCharts Label | JSON Field | Parse Rule |
|--------------|------------|------------|
| Revenue (TTM) | `income_statement.revenue_ttm` | Parse "187.14B" → 187140000000 |
| Net Income (TTM) | `income_statement.net_income_ttm` | Parse "99.20B" → 99200000000 |
| EBIT (TTM) | `income_statement.ebit_ttm` | Parse number |
| EBITDA (TTM) | `income_statement.ebitda_ttm` | Parse number |
| Revenue (Quarterly) | `income_statement.revenue_quarterly` | Parse number |
| Net Income (Quarterly) | `income_statement.net_income_quarterly` | Parse number |
| EBIT (Quarterly) | `income_statement.ebit_quarterly` | Parse number |
| EBITDA (Quarterly) | `income_statement.ebitda_quarterly` | Parse number |
| Revenue (Quarterly YoY Growth) | `income_statement.revenue_growth_yoy` | Parse "62.49%" → 0.6249 |
| EPS Diluted (Quarterly YoY Growth) | `income_statement.eps_growth_yoy` | Parse percentage |
| EBITDA (Quarterly YoY Growth) | `income_statement.ebitda_growth_yoy` | Parse percentage |
| EPS Diluted (TTM) | `income_statement.eps_diluted_ttm` | Float |
| EPS Basic (TTM) | `income_statement.eps_basic_ttm` | Float |
| Shares Outstanding | `income_statement.shares_outstanding` | Parse "24.30B" → 24300000000 |

### Balance Sheet Section

| YCharts Label | JSON Field |
|--------------|------------|
| Total Assets (Quarterly) | `balance_sheet.total_assets` |
| Total Liabilities (Quarterly) | `balance_sheet.total_liabilities` |
| Shareholders Equity (Quarterly) | `balance_sheet.shareholders_equity` |
| Cash and Short Term Investments (Quarterly) | `balance_sheet.cash_and_short_term_investments` |
| Total Long Term Assets (Quarterly) | `balance_sheet.total_long_term_assets` |
| Total Long Term Debt (Quarterly) | `balance_sheet.total_long_term_debt` |
| Book Value (Quarterly) | `balance_sheet.book_value` |

### Cash Flow Section

| YCharts Label | JSON Field |
|--------------|------------|
| Cash from Operations (TTM) | `cash_flow.cash_from_operations_ttm` |
| Cash from Investing (TTM) | `cash_flow.cash_from_investing_ttm` |
| Cash from Financing (TTM) | `cash_flow.cash_from_financing_ttm` |
| Change in Receivables (TTM) | `cash_flow.change_in_receivables_ttm` |
| Changes in Working Capital (TTM) | `cash_flow.changes_in_working_capital_ttm` |
| Capital Expenditures (TTM) | `cash_flow.capital_expenditures_ttm` |
| Ending Cash (Quarterly) | `cash_flow.ending_cash_quarterly` |
| Free Cash Flow | `cash_flow.free_cash_flow_ttm` |

### Profitability Section

| YCharts Label | JSON Field |
|--------------|------------|
| Operating Margin (TTM) | `profitability.operating_margin_ttm` |
| Gross Profit Margin | `profitability.gross_profit_margin_ttm` |

### Earnings Quality Section

| YCharts Label | JSON Field |
|--------------|------------|
| Return on Assets | `profitability.return_on_assets` |
| Return on Equity | `profitability.return_on_equity` |
| Return on Invested Capital | `profitability.return_on_invested_capital` |

### Stock Price Performance Section

| YCharts Label | JSON Field | Parse Rule |
|--------------|------------|------------|
| 1 Month Total Returns (Daily) | `performance.return_1m` | Parse "2.81%" → 0.0281 |
| 3 Month Total Returns (Daily) | `performance.return_3m` | Parse percentage |
| 6 Month Total Returns (Daily) | `performance.return_6m` | Parse percentage |
| Year to Date Total Returns (Daily) | `performance.return_ytd` | Parse percentage |
| 1 Year Total Returns (Daily) | `performance.return_1y` | Parse percentage |
| Annualized 3 Year Total Returns (Daily) | `performance.return_3y_annualized` | Parse percentage |
| Annualized 5 Year Total Returns (Daily) | `performance.return_5y_annualized` | Parse percentage |
| Annualized 10 Year Total Returns (Daily) | `performance.return_10y_annualized` | Parse percentage |
| Annualized 15 Year Total Returns (Daily) | `performance.return_15y_annualized` | Parse percentage |
| Annualized Total Returns Since Inception (Daily) | `performance.return_since_inception_annualized` | Parse percentage |
| 52 Week High (Daily) | `performance.high_52w` | Float |
| 52 Week Low (Daily) | `performance.low_52w` | Float |
| 52-Week High Date | `performance.high_52w_date` | Parse "Oct. 29, 2025" → "2025-10-29" |
| 52-Week Low Date | `performance.low_52w_date` | Parse date |

### Valuation Section

| YCharts Label | JSON Field |
|--------------|------------|
| Market Cap | `valuation.market_cap` |
| Enterprise Value | `valuation.enterprise_value` |
| Price | `valuation.price` (also in `price.current`) |
| PE Ratio | `valuation.pe_ratio` |
| PE Ratio (Forward) | `valuation.pe_ratio_forward` |
| PE Ratio (Forward 1y) | `valuation.pe_ratio_forward_1y` |
| PS Ratio | `valuation.ps_ratio` |
| PS Ratio (Forward) | `valuation.ps_ratio_forward` |
| PS Ratio (Forward 1y) | `valuation.ps_ratio_forward_1y` |
| Price to Book Value | `valuation.price_to_book` |
| Price to Free Cash Flow | `valuation.price_to_free_cash_flow` |
| PEG Ratio | `valuation.peg_ratio` |
| EV to EBITDA | `valuation.ev_to_ebitda` |
| EV to EBITDA (Forward) | `valuation.ev_to_ebitda_forward` |
| EV to EBIT | `valuation.ev_to_ebit` |
| EBIT Margin (TTM) | `valuation.ebit_margin_ttm` |

### Estimates Section

| YCharts Label | JSON Field |
|--------------|------------|
| Revenue Estimates for Current Quarter | `estimates.revenue_current_quarter` |
| Revenue Estimates for Next Quarter | `estimates.revenue_next_quarter` |
| Revenue Estimates for Current Fiscal Year | `estimates.revenue_current_fiscal_year` |
| Revenue Estimates for Next Fiscal Year | `estimates.revenue_next_fiscal_year` |
| EPS Estimates for Current Quarter | `estimates.eps_current_quarter` |
| EPS Estimates for Next Quarter | `estimates.eps_next_quarter` |
| EPS Estimates for Current Fiscal Year | `estimates.eps_current_fiscal_year` |
| EPS Estimates for Next Fiscal Year | `estimates.eps_next_fiscal_year` |

### Dividends Section

| YCharts Label | JSON Field |
|--------------|------------|
| Dividend Yield | `dividends.dividend_yield` |
| Dividend Yield (Forward) | `dividends.dividend_yield_forward` |
| Payout Ratio (TTM) | `dividends.payout_ratio_ttm` |
| Cash Dividend Payout Ratio | `dividends.cash_dividend_payout_ratio` |
| Last Dividend Amount | `dividends.last_dividend_amount` |
| Last Ex-Dividend Date | `dividends.last_ex_dividend_date` |

### Risk Metrics Section

| YCharts Label | JSON Field |
|--------------|------------|
| Alpha (5Y) | `risk_metrics.alpha_5y` |
| Beta (5Y) | `risk_metrics.beta_5y` |
| Annualized Standard Deviation of Monthly Returns (5Y Lookback) | `risk_metrics.standard_deviation_monthly_5y` |
| Historical Sharpe Ratio (5Y) | `risk_metrics.sharpe_ratio_5y` |
| Historical Sortino (5Y) | `risk_metrics.sortino_ratio_5y` |
| Max Drawdown (5Y) | `risk_metrics.max_drawdown_5y` |
| Monthly Value at Risk (VaR) 5% (5Y Lookback) | `risk_metrics.var_monthly_5pct_5y` |

### Management Effectiveness Section

| YCharts Label | JSON Field |
|--------------|------------|
| Asset Utilization (TTM) | `management_effectiveness.asset_utilization_ttm` |
| Days Sales Outstanding (Quarterly) | `management_effectiveness.days_sales_outstanding` |
| Days Inventory Outstanding (Quarterly) | `management_effectiveness.days_inventory_outstanding` |
| Days Payable Outstanding (Quarterly) | `management_effectiveness.days_payable_outstanding` |
| Total Receivables (Quarterly) | `management_effectiveness.total_receivables` |

### Advanced Metrics Section

| YCharts Label | JSON Field |
|--------------|------------|
| Piotroski F Score (TTM) | `advanced_metrics.piotroski_f_score_ttm` |
| Sustainable Growth Rate (TTM) | `advanced_metrics.sustainable_growth_rate_ttm` |
| Tobin's Q (Approximate) (Quarterly) | `advanced_metrics.tobin_q_quarterly` |
| Momentum Score | `advanced_metrics.momentum_score` |
| Market Cap Score | `advanced_metrics.market_cap_score` |
| Quality Ratio Score | `advanced_metrics.quality_ratio_score` |

### Liquidity & Solvency Section

| YCharts Label | JSON Field |
|--------------|------------|
| Debt to Equity Ratio | `liquidity_solvency.debt_to_equity_ratio` |
| Free Cash Flow (Quarterly) | `cash_flow.free_cash_flow_quarterly` |
| Current Ratio | `liquidity_solvency.current_ratio` |
| Quick Ratio (Quarterly) | `liquidity_solvency.quick_ratio` |
| Altman Z-Score (TTM) | `liquidity_solvency.altman_z_score_ttm` |
| Times Interest Earned (TTM) | `liquidity_solvency.times_interest_earned_ttm` |

### Employee Count Metrics Section

| YCharts Label | JSON Field |
|--------------|------------|
| Total Employees (Annual) | `employees.total_employees_annual` |
| Revenue Per Employee (Annual) | `employees.revenue_per_employee_annual` |
| Net Income Per Employee (Annual) | `employees.net_income_per_employee_annual` |

## Number Parsing Rules

### Dollar Amounts
- "187.14B" → 187140000000 (multiply by 1,000,000,000)
- "99.20B" → 99200000000
- "57.01B" → 57010000000
- "3.979M" → 3979000 (multiply by 1,000,000)
- "11.33B" → 11330000000
- "-28.56B" → -28560000000 (handle negatives)

### Percentages
- "62.49%" → 0.6249 (divide by 100)
- "0.02%" → 0.0002
- "-1.60%" → -0.0160 (handle negatives)

### Plain Numbers
- "4.038" → 4.038 (keep as-is)
- "24.30B" → 24300000000 (shares)
- "36000" → 36000 (employees)

### Dates
- "Oct. 29, 2025" → "2025-10-29"
- "Dec. 04, 2025" → "2025-12-04"
- "Apr. 07, 2025" → "2025-04-07"

## Error Handling

### Missing Fields
If a metric isn't displayed on the page, send `null`:
```json
{
  "dividends": {
    "dividend_yield": null,
    "last_dividend_amount": null
  }
}
```

### Parse Failures
If you can't parse a value:
1. Log a warning
2. Send `null` for that field
3. Continue with other fields
4. Report parsing issues in task result

### API Errors
| Status | Action |
|--------|--------|
| 400 | Check JSON structure, re-read schema |
| 401 | Re-authenticate |
| 409 | Data already exists (skip or update timestamp) |
| 500 | Retry up to 3 times with exponential backoff |

## Testing

Before scraping production tickers, test with NVDA:
1. Extract all metrics
2. Validate against JSON schema
3. POST to API
4. Verify response
5. Check data in database

## Rate Limiting

- Max 1 ticker per 2 seconds (YCharts may rate limit)
- If scraping multiple tickers, add delays between requests
- Respect YCharts' robots.txt and terms of service

## Schema Reference

Full JSON schema: `/Users/larryli/.openclaw/workspace/memory/2026-02-12-ycharts-keystats-schema.md`

API endpoint will validate against this schema.
