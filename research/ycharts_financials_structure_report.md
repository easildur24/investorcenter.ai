# YCharts Financials Structure Research Report
**Ticker**: AAPL (Apple, Inc.)  
**Date**: 2026-02-24  
**Purpose**: Capture exact structure for schema design and ingestion pipeline

---

## Summary

All three financial statement pages (Income Statement, Balance Sheet, Cash Flow) use a consistent structure:

- **Column Headers**: Period identifiers in YYYY-MM format
- **Metadata Row**: Actual Release Date in MM/DD/YYYY format
- **Data Organization**: Grouped into logical sections with clickable row labels linking to detailed metric pages
- **Data Formats**: Dollar amounts with B/M suffixes, percentages, floats, and dates
- **View Options**: Quarterly, Annual, TTM (Trailing Twelve Months), and various growth views

---

## 1. Income Statement

**URL Pattern**: `https://ycharts.com/companies/{TICKER}/financials/income_statement/1`

### Column Headers (Annual View)
```
2025-09, 2024-09, 2023-09, 2022-09, 2021-09, 2020-09, 2019-09, 2018-09
```
- Format: YYYY-MM (fiscal year end)
- Metadata: Actual Release Date (MM/DD/YYYY)

### Sections & Line Items

#### Income (Annual)
- Revenue
- COGS Incl D&A
- COGS Excl D&A
- Depreciation & Amortization Expense
- Gross Income
- SG&A Expense
- R&D Expense
- Other Operating Expense
- Operating Income
- Non Operating Income/Expense
- Interest Expense
- Pretax Income
- Income Tax
- Income Tax - Current Domestic
- Income Tax - Current Foreign
- Income Tax - Deferred Domestic
- Income Tax - Deferred Foreign
- Income Tax - Credits
- Consolidated Net Income Before Non-Controlling Interests
- Equity in Net Income
- Consolidated Net Income
- Minority Interest in Earnings
- Net Income

#### Other Income Metrics (Annual)
- EBITDA
- EBIT
- Gross Profit Margin
- EBITDA Margin
- EBIT Margin
- Net Profit Margin
- NOPAT
- NOPAT Margin

#### EPS Metrics (Annual)
- Net EPS
- EPS - Earnings Per Share (Basic)
- EPS - Earnings Per Share (Diluted)
- EPS - Cash Flow

#### Dividends (Annual)
- DPS - Dividends Per Share

#### Shares (Annual)
- Basic Shares Outstanding
- Diluted Shares Outstanding

### Data Formats
- **Dollar amounts**: Formatted with B (billions) or M (millions) suffix (e.g., "391.04B", "2.87B")
- **Percentages**: Decimal format (e.g., "46.40%", "31.94%")
- **Floats**: 2-4 decimal places (e.g., "6.11", "14.02")
- **Missing data**: Represented as "--"

### Available Views
Format dropdown includes:
- Quarterly
- Annual
- TTM (Trailing Twelve Months)
- Quarterly YoY Growth
- Annual YoY Growth
- Quarterly Sequential Growth
- Quarterly % of Revenue
- Annual % of Revenue
- TTM % of Revenue

### Notes
- Default view: **Annual**
- **Quarterly vs Annual**: Same line items, different period granularity
- All metric row labels are clickable links to dedicated metric detail pages

---

## 2. Balance Sheet

**URL Pattern**: `https://ycharts.com/companies/{TICKER}/financials/balance_sheet/1`

### Column Headers (Quarterly View)
```
2025-12, 2025-09, 2025-06, 2025-03, 2024-12, 2024-09, 2024-06, 2024-03
```
- Format: YYYY-MM (fiscal quarter end)
- Metadata: Actual Release Date (MM/DD/YYYY)

### Sections & Line Items

#### Assets (Quarterly)
- Cash and Short Term Investments
- Accounts Receivable
- Other Receivables
- Short-Term Receivables
- Inventories
- Prepaid Expenses
- Miscellaneous Current Assets
- Current Assets, Other
- Total Current Assets
- Gross PP&E
- Accumulated D&A
- Net PP&E
- Long Term Investments
- Net Goodwill
- Net Other Intangibles
- Goodwill and Intangibles
- Deferred Tax Assets
- Assets - Other
- Total Assets

#### Liabilities (Quarterly)
- ST Debt & Current Portion Of LT Debt
- Accounts Payable
- Current Tax Payable
- Other Current Liabilities
- Total Current Liabilities
- Long Term Debt Excl Lease Liab
- Capital & Operating Lease Oblig
- Non-Current Portion of Long Term Debt
- Provisions for Risk and Charges
- Long Term Deferred Tax Liabilities
- Other Liabilities
- Total Long Term Liabilities
- Total Liabilities

#### Shareholder's Equity (Quarterly)
- Non-Equity Reserves
- Preferred Stock
- Common Equity
- Shareholders Equity

#### Other Metrics (Quarterly)
- Minority Interest Ownership
- Total Equity Including Minority Interest
- Total Liabilities And Shareholders' Equity
- Book Value Per Share
- Tangible Book Value Per Share
- Working Capital - Total (Net Working Capital)
- Total Capital, Including Short-Term Debt
- Total Debt
- Net Debt
- Minimum Pension Liabilities
- Comprehensive Income - Hedging Gain/Loss
- Comprehensive Income - Unearned Comp
- Stock-Based Compensation Expense
- Stock-Based Comp Adj to Net Income Q
- Total Operating Lease Commitments
- Current Portion of Long Term Debt

### Data Formats
- **Dollar amounts**: Same format as Income Statement (e.g., "149.02B", "2.94B")
- **Per-share metrics**: Float format (e.g., "3.99", "3.95")
- **Negative values**: Displayed with "-" prefix (e.g., "-117.59B")
- **Missing data**: Represented as "--" or "0.00"

### Available Views
Similar to Income Statement:
- Quarterly (default)
- Annual
- TTM
- Growth views
- % of Total Assets views

### Notes
- Default view: **Quarterly**
- Accumulated D&A is negative (contra-asset)
- Many companies have "--" or "0.00" for items like Goodwill, Intangibles
- All metric row labels are clickable links

---

## 3. Cash Flow Statement

**URL Pattern**: `https://ycharts.com/companies/{TICKER}/financials/cash_flow/1`

### Column Headers (Quarterly View)
```
2025-12, 2025-09, 2025-06, 2025-03, 2024-12, 2024-09, 2024-06, 2024-03
```
- Format: YYYY-MM (fiscal quarter end)
- Metadata: Actual Release Date (MM/DD/YYYY)

### Sections & Line Items

#### Operations (Quarterly)
- Net Income
- Total Depreciation and Amortization
- Deferred Taxes & Investment Tax Credit
- Operating Interest Paid - Lease Liab
- Non-Cash Items
- Funds From Operations
- Extraordinary Items
- Changes in Working Capital
- Cash from Operations

#### Investing (Quarterly)
- Capital Expenditures - Total
- Net Divestitures (Acquisitions)
- Sale of Fixed Assets & Businesses
- Total Net Change in Investments
- Other Funds
- Cash from Investing

#### Financing (Quarterly)
- Total Dividends Paid
- Net Common Equity Issued (Purchased)
- Net Debt Issuance
- Repayments Of Operating Lease Liabilities
- Financing Interest Paid - Lease Liab
- Other Funds - Financing
- Cash from Financing

#### Ending Cash (Quarterly)
- Beginning Cash
- Exchange Rate Effect
- Net Change in Cash
- Ending Cash
- Free Cash Flow

#### Other Metrics (Quarterly)
- Interest Paid, Operating and Financing
- Capital and Operating Lease Obligations
- Income Tax Paid
- Stock Based Compensation

### Data Formats
- **Dollar amounts**: Same format (e.g., "42.10B", "3.214B", "-2.359B", "-154.00M")
- **Negative values**: Common in financing section (dividends, buybacks)
- **Missing data**: "0.00" or "--"

### Available Views
Similar to other statements:
- Quarterly (default)
- Annual
- TTM
- Growth views

### Notes
- Default view: **Quarterly**
- Financing activities typically show large negative values (dividends, buybacks)
- Changes in Working Capital can be positive or negative
- All metric row labels are clickable links

---

## Key Observations for Schema Design

### 1. **Consistent Column Structure**
All three statements share the same period column format (YYYY-MM) and metadata row pattern (Actual Release Date).

### 2. **Hierarchical Organization**
Data is organized into logical sections:
- Income Statement: Income → Other Metrics → EPS → Dividends → Shares
- Balance Sheet: Assets → Liabilities → Equity → Other Metrics
- Cash Flow: Operations → Investing → Financing → Ending Cash → Other Metrics

### 3. **Data Type Patterns**
- **Monetary values**: Always suffixed (B/M), can be negative
- **Percentages**: Always include "%" symbol
- **Per-share metrics**: Float format without suffix
- **Share counts**: Large floats or integers

### 4. **Missing/Zero Data Handling**
- "--" indicates no data available
- "0.00" indicates explicitly zero value
- Not all companies have all line items (e.g., many have no Goodwill)

### 5. **Link Structure**
Every metric row label is a link to a detailed page:
```
/companies/{TICKER}/{metric_slug}
```
Example: `/companies/AAPL/net_income_cf`

### 6. **View Consistency**
While default views differ (Annual for Income Statement, Quarterly for Balance Sheet and Cash Flow), all three support the same view options.

### 7. **Period Navigation**
- Pagination controls: First Period, Prev Period, Last Period, Next Period
- URL pattern includes page number: `/financials/{statement_type}/{page_number}`
- Chronological order toggle available
- View Future Periods toggle available

---

## Recommendations for Ingestion Pipeline

### Phase 1: Structure Parsing
1. **Column headers**: Extract period identifiers (YYYY-MM format)
2. **Metadata row**: Capture Actual Release Date for each period
3. **Section headers**: Parse and preserve hierarchical grouping
4. **Row labels**: Extract metric names and their corresponding metric slugs (from href)

### Phase 2: Data Extraction
1. **Numeric parsing**: Handle B/M suffixes, negative values, percentages
2. **Missing data**: Normalize "--" and null handling
3. **Data validation**: Check that values align with expected data types

### Phase 3: Schema Design
1. **Common structure**: 
   - `ticker` (string)
   - `statement_type` (enum: income_statement, balance_sheet, cash_flow)
   - `view_type` (enum: quarterly, annual, ttm)
   - `period` (YYYY-MM format)
   - `release_date` (MM/DD/YYYY format)
   - `section` (string)
   - `metric_name` (string)
   - `metric_slug` (string, for deep linking)
   - `value` (float, normalized)
   - `value_display` (string, original formatted value)
   - `unit` (enum: dollars, percent, float, shares)

2. **Separate tables or unified structure?**
   - Option A: Three separate tables (income_statement, balance_sheet, cash_flow)
   - Option B: Single `financials` table with `statement_type` discriminator
   - **Recommendation**: Option B for flexibility and unified querying

### Phase 4: Scraping Strategy
1. Start with Quarterly view (more granular, includes recent data)
2. Navigate through pagination to capture historical periods
3. Optional: Also scrape Annual view for long-term trends
4. Store both raw HTML and parsed structured data for validation

---

## Next Steps

1. ✅ Structure research complete
2. ⬜ Design JSON schema for financials ingestion
3. ⬜ Build YCharts financials scraping skill
4. ⬜ Implement data parser and validator
5. ⬜ Create S3 upload logic for financials data
6. ⬜ Add financials ingestion API endpoint
7. ⬜ Test end-to-end pipeline with AAPL data
