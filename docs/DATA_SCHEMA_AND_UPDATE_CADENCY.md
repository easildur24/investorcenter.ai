# Data Schema and Update Cadency

This document describes the database schema, data sources, and update schedules for InvestorCenter.ai's IC Score Service.

**Last Updated:** November 27, 2025

---

## Table of Contents

1. [Time-Series Data](#time-series-data)
2. [Periodic Financial Data](#periodic-financial-data)
3. [Calculated Metrics](#calculated-metrics)
4. [Master Data Tables](#master-data-tables)
5. [User Tables](#user-tables)
6. [Update Schedule Summary](#update-schedule-summary)
7. [Pipeline Dependencies](#pipeline-dependencies)
8. [Data Sources](#data-sources)
9. [Environment Variables](#environment-variables)
10. [Quick Reference Commands](#quick-reference-commands)

---

## Time-Series Data

These tables store data that changes daily and are optimized using TimescaleDB hypertables for efficient time-series queries.

### 1. Stock Prices (`stock_prices`)

Daily OHLCV (Open, High, Low, Close, Volume) price data for all tracked stocks.

| Column | Type | Description |
|--------|------|-------------|
| `time` | TIMESTAMPTZ | Trading date/time (PK) |
| `ticker` | VARCHAR(10) | Stock symbol (PK) |
| `open` | DECIMAL(10,2) | Opening price |
| `high` | DECIMAL(10,2) | Day high |
| `low` | DECIMAL(10,2) | Day low |
| `close` | DECIMAL(10,2) | Closing price |
| `volume` | BIGINT | Trading volume |
| `vwap` | DECIMAL(10,2) | Volume-weighted average price |
| `interval` | VARCHAR(10) | Time interval ('1day') |

- **Data Source:** Polygon.io API
- **Update Cadency:** Daily at 10:30 PM UTC (Mon-Fri, after market close)
- **CronJob:** `ic-score-daily-price-update`
- **History:** 10 years of historical data (backfilled)

---

### 2. Technical Indicators (`technical_indicators`)

Calculated technical analysis indicators for each stock.

| Column | Type | Description |
|--------|------|-------------|
| `time` | TIMESTAMPTZ | Calculation date (PK) |
| `ticker` | VARCHAR(10) | Stock symbol (PK) |
| `indicator_name` | VARCHAR(50) | Indicator type (PK) |
| `value` | DECIMAL(18,6) | Calculated value |
| `metadata` | JSONB | Additional parameters |

**Indicators Calculated:**

| Indicator | Description |
|-----------|-------------|
| RSI | Relative Strength Index (14-day) |
| MACD | Moving Average Convergence Divergence |
| MACD_SIGNAL | MACD Signal Line |
| MACD_HISTOGRAM | MACD Histogram |
| SMA_20 | 20-day Simple Moving Average |
| SMA_50 | 50-day Simple Moving Average |
| SMA_200 | 200-day Simple Moving Average |
| EMA_12 | 12-day Exponential Moving Average |
| EMA_26 | 26-day Exponential Moving Average |
| BB_UPPER | Bollinger Band Upper |
| BB_MIDDLE | Bollinger Band Middle |
| BB_LOWER | Bollinger Band Lower |
| ATR | Average True Range |
| OBV | On-Balance Volume |
| ADX | Average Directional Index |
| STOCH_K | Stochastic %K |
| STOCH_D | Stochastic %D |

- **Data Source:** Calculated from `stock_prices`
- **Update Cadency:** Daily at 11:00 PM UTC (Mon-Fri)
- **CronJob:** `ic-score-technical-indicators`
- **Dependencies:** Requires `stock_prices` to be updated first

---

### 3. Risk Metrics (`risk_metrics`)

Risk and performance metrics calculated relative to benchmarks.

| Column | Type | Description |
|--------|------|-------------|
| `time` | TIMESTAMPTZ | Calculation date (PK) |
| `ticker` | VARCHAR(10) | Stock symbol (PK) |
| `period` | VARCHAR(10) | Lookback period (PK) |
| `alpha` | DECIMAL(10,4) | Jensen's Alpha |
| `beta` | DECIMAL(10,4) | Market Beta |
| `sharpe_ratio` | DECIMAL(10,4) | Sharpe Ratio |
| `sortino_ratio` | DECIMAL(10,4) | Sortino Ratio |
| `std_dev` | DECIMAL(10,4) | Standard Deviation |
| `max_drawdown` | DECIMAL(10,4) | Maximum Drawdown |
| `var_5` | DECIMAL(10,4) | Value at Risk (5%) |
| `annualized_return` | DECIMAL(10,4) | Annualized Return |
| `downside_deviation` | DECIMAL(10,4) | Downside Deviation |
| `data_points` | INTEGER | Number of data points used |
| `calculation_date` | TIMESTAMP | When calculated |

- **Data Source:** Calculated from `stock_prices` + `benchmark_returns` + `treasury_rates`
- **Update Cadency:** Daily at 12:00 AM UTC (Tue-Sat, after previous day's prices)
- **CronJob:** `ic-score-risk-metrics`
- **Dependencies:** Requires `stock_prices`, `benchmark_returns`, `treasury_rates`

---

### 4. Benchmark Returns (`benchmark_returns`)

Daily returns for benchmark indices (SPY, QQQ, etc.).

| Column | Type | Description |
|--------|------|-------------|
| `time` | TIMESTAMPTZ | Trading date (PK) |
| `symbol` | VARCHAR(20) | Benchmark symbol (PK) |
| `close` | DECIMAL(12,4) | Closing price |
| `total_return` | DECIMAL(12,4) | Total return |
| `daily_return` | DECIMAL(10,6) | Daily return percentage |
| `volume` | BIGINT | Trading volume |

- **Data Source:** Polygon.io API
- **Update Cadency:** Daily at 1:00 AM UTC
- **CronJob:** `ic-score-benchmark-data`

---

### 5. Treasury Rates (`treasury_rates`)

Daily risk-free rates from US Treasury.

| Column | Type | Description |
|--------|------|-------------|
| `date` | DATE | Observation date (PK) |
| `rate_1m` | DECIMAL(8,4) | 1-month T-bill rate |
| `rate_3m` | DECIMAL(8,4) | 3-month T-bill rate |
| `rate_6m` | DECIMAL(8,4) | 6-month T-bill rate |
| `rate_1y` | DECIMAL(8,4) | 1-year T-bill rate |
| `rate_2y` | DECIMAL(8,4) | 2-year Treasury rate |
| `rate_10y` | DECIMAL(8,4) | 10-year Treasury rate |
| `created_at` | TIMESTAMP | Record creation time |

- **Data Source:** FRED (Federal Reserve Economic Data)
- **Update Cadency:** Daily at 2:00 AM UTC
- **CronJob:** `ic-score-treasury-rates`

---

## Periodic Financial Data

These tables store data that updates on fixed schedules (quarterly, weekly, etc.).

### 6. Financials (`financials`)

Quarterly and annual financial statements from SEC filings.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `ticker` | VARCHAR(10) | Stock symbol |
| `filing_date` | DATE | SEC filing date |
| `period_end_date` | DATE | Fiscal period end |
| `fiscal_year` | INT | Fiscal year |
| `fiscal_quarter` | INT | Quarter (NULL for annual) |
| `statement_type` | VARCHAR(20) | '10-K' or '10-Q' |
| **Income Statement** | | |
| `revenue` | BIGINT | Total revenue |
| `cost_of_revenue` | BIGINT | Cost of goods sold |
| `gross_profit` | BIGINT | Gross profit |
| `operating_expenses` | BIGINT | Operating expenses |
| `operating_income` | BIGINT | Operating income |
| `net_income` | BIGINT | Net income |
| `eps_basic` | DECIMAL(10,4) | Basic EPS |
| `eps_diluted` | DECIMAL(10,4) | Diluted EPS |
| `shares_outstanding` | BIGINT | Shares outstanding |
| **Balance Sheet** | | |
| `total_assets` | BIGINT | Total assets |
| `total_liabilities` | BIGINT | Total liabilities |
| `shareholders_equity` | BIGINT | Shareholders' equity |
| `cash_and_equivalents` | BIGINT | Cash and equivalents |
| `short_term_debt` | BIGINT | Short-term debt |
| `long_term_debt` | BIGINT | Long-term debt |
| **Cash Flow** | | |
| `operating_cash_flow` | BIGINT | Cash from operations |
| `investing_cash_flow` | BIGINT | Cash from investing |
| `financing_cash_flow` | BIGINT | Cash from financing |
| `free_cash_flow` | BIGINT | Free cash flow |
| `capex` | BIGINT | Capital expenditures |
| **Calculated Ratios** | | |
| `pe_ratio` | DECIMAL(10,2) | Price-to-Earnings |
| `pb_ratio` | DECIMAL(10,2) | Price-to-Book |
| `ps_ratio` | DECIMAL(10,2) | Price-to-Sales |
| `debt_to_equity` | DECIMAL(10,2) | Debt/Equity ratio |
| `roe` | DECIMAL(10,2) | Return on Equity |
| `roa` | DECIMAL(10,2) | Return on Assets |
| `roic` | DECIMAL(10,2) | Return on Invested Capital |
| `gross_margin` | DECIMAL(10,2) | Gross Margin % |
| `operating_margin` | DECIMAL(10,2) | Operating Margin % |
| `net_margin` | DECIMAL(10,2) | Net Margin % |
| **Metadata** | | |
| `sec_filing_url` | VARCHAR(500) | SEC filing URL |
| `raw_data` | JSONB | Raw XBRL data |
| `created_at` | TIMESTAMP | Record creation time |

- **Data Source:** SEC EDGAR (10-K, 10-Q filings)
- **Update Cadency:** Weekly on Sundays at 2:00 AM UTC
- **CronJob:** `ic-score-sec-financials`

---

### 7. TTM Financials (`ttm_financials`)

Trailing Twelve Months (TTM) financial metrics calculated from quarterly data.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `ticker` | VARCHAR(10) | Stock symbol |
| `calculation_date` | DATE | When calculated |
| `ttm_period_start` | DATE | TTM period start |
| `ttm_period_end` | DATE | TTM period end |
| `revenue` | BIGINT | Sum of last 4 quarters |
| `cost_of_revenue` | BIGINT | Sum of last 4 quarters |
| `gross_profit` | BIGINT | Sum of last 4 quarters |
| `operating_expenses` | BIGINT | Sum of last 4 quarters |
| `operating_income` | BIGINT | Sum of last 4 quarters |
| `net_income` | BIGINT | Sum of last 4 quarters |
| `eps_basic` | DECIMAL(10,4) | TTM Basic EPS |
| `eps_diluted` | DECIMAL(10,4) | TTM Diluted EPS |
| `shares_outstanding` | BIGINT | Latest quarter shares |
| `total_assets` | BIGINT | Latest quarter assets |
| `total_liabilities` | BIGINT | Latest quarter liabilities |
| `shareholders_equity` | BIGINT | Latest quarter equity |
| `cash_and_equivalents` | BIGINT | Latest quarter cash |
| `short_term_debt` | BIGINT | Latest quarter short debt |
| `long_term_debt` | BIGINT | Latest quarter long debt |
| `operating_cash_flow` | BIGINT | TTM Operating CF |
| `investing_cash_flow` | BIGINT | TTM Investing CF |
| `financing_cash_flow` | BIGINT | TTM Financing CF |
| `free_cash_flow` | BIGINT | TTM FCF |
| `capex` | BIGINT | TTM CapEx |
| `quarters_included` | JSONB | Quarter breakdown details |
| `created_at` | TIMESTAMP | Record creation time |

- **Data Source:** Calculated from `financials` table
- **Update Cadency:** Daily at 10:00 PM UTC (Mon-Fri)
- **CronJob:** `ic-score-ttm-financials`

---

### 8. Insider Trades (`insider_trades`)

SEC Form 4 insider trading activity.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `ticker` | VARCHAR(10) | Stock symbol |
| `filing_date` | DATE | SEC filing date |
| `transaction_date` | DATE | Trade date |
| `insider_name` | VARCHAR(255) | Insider's name |
| `insider_title` | VARCHAR(255) | Position/title |
| `transaction_type` | VARCHAR(50) | 'Buy', 'Sell', 'Option Exercise' |
| `shares` | BIGINT | Number of shares |
| `price_per_share` | DECIMAL(10,2) | Transaction price |
| `total_value` | BIGINT | Total transaction value |
| `shares_owned_after` | BIGINT | Shares after transaction |
| `is_derivative` | BOOLEAN | Is derivative transaction |
| `form_type` | VARCHAR(10) | Form type (4, 4/A) |
| `sec_filing_url` | VARCHAR(500) | SEC filing URL |
| `created_at` | TIMESTAMP | Record creation time |

- **Data Source:** SEC EDGAR (Form 4)
- **Update Cadency:** Hourly during market hours (2:00 PM - 9:00 PM UTC, Mon-Fri)
- **CronJob:** `ic-score-insider-trades`

---

### 9. Institutional Holdings (`institutional_holdings`)

SEC Form 13F institutional ownership data.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `ticker` | VARCHAR(10) | Stock symbol |
| `filing_date` | DATE | SEC filing date |
| `quarter_end_date` | DATE | Quarter end |
| `institution_name` | VARCHAR(255) | Institution name |
| `institution_cik` | VARCHAR(20) | SEC CIK |
| `shares` | BIGINT | Shares held |
| `market_value` | BIGINT | Position value |
| `percent_of_portfolio` | DECIMAL(10,4) | % of portfolio |
| `position_change` | VARCHAR(50) | 'New', 'Increased', 'Decreased', 'Unchanged' |
| `shares_change` | BIGINT | Change in shares |
| `percent_change` | DECIMAL(10,2) | % change |
| `sec_filing_url` | VARCHAR(500) | SEC filing URL |
| `created_at` | TIMESTAMP | Record creation time |

- **Data Source:** SEC EDGAR (Form 13F)
- **Update Cadency:** Quarterly (15th of Jan, Apr, Jul, Oct at 3:00 AM UTC)
- **CronJob:** `ic-score-13f-holdings`

---

### 10. Analyst Ratings (`analyst_ratings`)

Wall Street analyst ratings and price targets.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `ticker` | VARCHAR(10) | Stock symbol |
| `rating_date` | DATE | Rating date |
| `analyst_name` | VARCHAR(255) | Analyst name |
| `analyst_firm` | VARCHAR(255) | Firm name |
| `rating` | VARCHAR(50) | 'Strong Buy' to 'Strong Sell' |
| `rating_numeric` | DECIMAL(3,1) | 1.0 to 5.0 |
| `price_target` | DECIMAL(10,2) | Target price |
| `prior_rating` | VARCHAR(50) | Previous rating |
| `prior_price_target` | DECIMAL(10,2) | Previous target |
| `action` | VARCHAR(50) | 'Initiated', 'Upgraded', 'Downgraded', 'Reiterated' |
| `notes` | TEXT | Additional notes |
| `source` | VARCHAR(100) | Data source |
| `created_at` | TIMESTAMP | Record creation time |

- **Data Source:** Polygon.io (Analyst Data)
- **Update Cadency:** Daily at 4:00 AM UTC
- **CronJob:** `ic-score-analyst-ratings`

---

### 11. News Articles (`news_articles`)

News with AI-powered sentiment analysis.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `title` | VARCHAR(500) | Article title |
| `url` | VARCHAR(1000) | Article URL (unique) |
| `source` | VARCHAR(255) | News source |
| `published_at` | TIMESTAMP | Publication time |
| `summary` | TEXT | Article summary |
| `content` | TEXT | Full content |
| `author` | VARCHAR(255) | Author name |
| `tickers` | VARCHAR(50)[] | Related tickers |
| `sentiment_score` | DECIMAL(5,2) | -100 to +100 |
| `sentiment_label` | VARCHAR(20) | 'Positive', 'Neutral', 'Negative' |
| `relevance_score` | DECIMAL(5,2) | 0 to 100 |
| `categories` | VARCHAR(50)[] | Article categories |
| `image_url` | VARCHAR(500) | Article image URL |
| `created_at` | TIMESTAMP | Record creation time |

- **Data Source:** Polygon.io (News API) + OpenAI (Sentiment)
- **Update Cadency:** Every 4 hours
- **CronJob:** `ic-score-news-sentiment`

---

## Calculated Metrics

### 12. Valuation Ratios (`valuation_ratios`)

Real-time valuation metrics using TTM financials and current prices.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `ticker` | VARCHAR(10) | Stock symbol |
| `calculation_date` | DATE | When calculated |
| `stock_price` | DECIMAL(10,2) | Current stock price |
| `ttm_pe_ratio` | DECIMAL(10,2) | Price/TTM EPS |
| `ttm_pb_ratio` | DECIMAL(10,2) | Price/Book (TTM) |
| `ttm_ps_ratio` | DECIMAL(10,2) | Price/Sales (TTM) |
| `ttm_market_cap` | BIGINT | Market cap (TTM shares) |
| `ttm_financial_id` | BIGINT | FK to ttm_financials |
| `ttm_period_start` | DATE | TTM period start |
| `ttm_period_end` | DATE | TTM period end |
| `annual_pe_ratio` | DECIMAL(10,2) | Price/Annual EPS |
| `annual_pb_ratio` | DECIMAL(10,2) | Price/Book (Annual) |
| `annual_ps_ratio` | DECIMAL(10,2) | Price/Sales (Annual) |
| `annual_market_cap` | BIGINT | Market cap (Annual shares) |
| `annual_financial_id` | BIGINT | FK to financials |
| `annual_period_end` | DATE | Annual period end |
| `created_at` | TIMESTAMP | Record creation time |

- **Data Source:** Calculated from `stock_prices` + `ttm_financials`
- **Update Cadency:** Daily at 11:30 PM UTC (Mon-Fri)
- **CronJob:** `ic-score-valuation-ratios`

---

### 13. Fundamental Metrics Extended (`fundamental_metrics_extended`)

Extended fundamental metrics including growth rates, leverage, dividends, and fair value.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `ticker` | VARCHAR(10) | Stock symbol |
| `calculation_date` | DATE | When calculated |
| **Profitability Margins** | | |
| `gross_margin` | DECIMAL(10,4) | Gross Margin % |
| `operating_margin` | DECIMAL(10,4) | Operating Margin % |
| `net_margin` | DECIMAL(10,4) | Net Margin % |
| `ebitda_margin` | DECIMAL(10,4) | EBITDA Margin % |
| **Returns** | | |
| `roe` | DECIMAL(10,4) | Return on Equity % |
| `roa` | DECIMAL(10,4) | Return on Assets % |
| `roic` | DECIMAL(10,4) | Return on Invested Capital % |
| **Growth Rates** | | |
| `revenue_growth_yoy` | DECIMAL(10,4) | Revenue Growth YoY % |
| `revenue_growth_3y_cagr` | DECIMAL(10,4) | Revenue 3Y CAGR % |
| `revenue_growth_5y_cagr` | DECIMAL(10,4) | Revenue 5Y CAGR % |
| `eps_growth_yoy` | DECIMAL(10,4) | EPS Growth YoY % |
| `eps_growth_3y_cagr` | DECIMAL(10,4) | EPS 3Y CAGR % |
| `eps_growth_5y_cagr` | DECIMAL(10,4) | EPS 5Y CAGR % |
| `fcf_growth_yoy` | DECIMAL(10,4) | FCF Growth YoY % |
| **Valuation** | | |
| `enterprise_value` | DECIMAL(20,2) | Enterprise Value |
| `ev_to_revenue` | DECIMAL(12,4) | EV/Revenue |
| `ev_to_ebitda` | DECIMAL(12,4) | EV/EBITDA |
| `ev_to_fcf` | DECIMAL(12,4) | EV/FCF |
| **Liquidity** | | |
| `current_ratio` | DECIMAL(10,4) | Current Ratio |
| `quick_ratio` | DECIMAL(10,4) | Quick Ratio |
| **Debt/Leverage** | | |
| `debt_to_equity` | DECIMAL(10,4) | Debt/Equity Ratio |
| `interest_coverage` | DECIMAL(10,4) | Interest Coverage Ratio |
| `net_debt_to_ebitda` | DECIMAL(10,4) | Net Debt/EBITDA |
| **Dividends** | | |
| `dividend_yield` | DECIMAL(10,4) | Dividend Yield % |
| `payout_ratio` | DECIMAL(10,4) | Payout Ratio % |
| `dividend_growth_rate` | DECIMAL(10,4) | Dividend Growth Rate % |
| `consecutive_dividend_years` | INTEGER | Years of consecutive dividends |
| **Fair Value** | | |
| `dcf_fair_value` | DECIMAL(14,2) | DCF Fair Value |
| `dcf_upside_percent` | DECIMAL(10,4) | DCF Upside % |
| `graham_number` | DECIMAL(14,2) | Graham Number |
| `epv_fair_value` | DECIMAL(14,2) | EPV Fair Value |
| **Sector Comparisons** | | |
| `pe_sector_percentile` | INTEGER | P/E Sector Percentile (0-100) |
| `pb_sector_percentile` | INTEGER | P/B Sector Percentile (0-100) |
| `roe_sector_percentile` | INTEGER | ROE Sector Percentile (0-100) |
| `margin_sector_percentile` | INTEGER | Margin Sector Percentile (0-100) |
| **WACC Components** | | |
| `wacc` | DECIMAL(10,4) | Weighted Avg Cost of Capital |
| `beta` | DECIMAL(10,4) | Beta |
| `cost_of_equity` | DECIMAL(10,4) | Cost of Equity |
| `cost_of_debt` | DECIMAL(10,4) | Cost of Debt |
| **Metadata** | | |
| `data_quality_score` | DECIMAL(5,2) | Data Quality Score (0-100) |
| `created_at` | TIMESTAMP | Record creation time |
| `updated_at` | TIMESTAMP | Last update time |

- **Data Source:** Calculated from `ttm_financials` + `stock_prices` + `risk_metrics`
- **Update Cadency:** Daily at 5:00 AM UTC (fundamental metrics), 7:00 AM UTC (fair value)
- **CronJobs:** `ic-score-fundamental-metrics`, `ic-score-fair-value`, `ic-score-refresh-sector-views`

---

### 14. IC Scores (`ic_scores`)

Proprietary 10-factor stock scores.

| Column | Type | Description |
|--------|------|-------------|
| `id` | BIGSERIAL | Primary key |
| `ticker` | VARCHAR(10) | Stock symbol |
| `date` | DATE | Score date |
| `overall_score` | DECIMAL(5,2) | 1-100 composite score |
| `value_score` | DECIMAL(5,2) | Value factor |
| `growth_score` | DECIMAL(5,2) | Growth factor |
| `profitability_score` | DECIMAL(5,2) | Profitability factor |
| `financial_health_score` | DECIMAL(5,2) | Balance sheet strength |
| `momentum_score` | DECIMAL(5,2) | Price momentum |
| `analyst_consensus_score` | DECIMAL(5,2) | Analyst ratings |
| `insider_activity_score` | DECIMAL(5,2) | Insider trading |
| `institutional_score` | DECIMAL(5,2) | Institutional ownership |
| `news_sentiment_score` | DECIMAL(5,2) | News sentiment |
| `technical_score` | DECIMAL(5,2) | Technical indicators |
| `rating` | VARCHAR(20) | 'Strong Buy' to 'Sell' |
| `sector_percentile` | DECIMAL(5,2) | Percentile in sector |
| `confidence_level` | VARCHAR(20) | 'High', 'Medium', 'Low' |
| `data_completeness` | DECIMAL(5,2) | % factors with data |
| `calculation_metadata` | JSONB | Factor weights and details |
| `created_at` | TIMESTAMP | Record creation time |

- **Data Source:** Calculated from all other data sources
- **Update Cadency:** Daily at 12:00 AM UTC
- **CronJob:** `ic-score-calculator`
- **Dependencies:** All other pipelines must complete first

---

## Master Data Tables

### 15. Tickers (`tickers`) - Primary Master Table

Master list of ALL tradeable assets from Polygon.io. This is the source of truth for the Go backend.

| Column | Type | Description |
|--------|------|-------------|
| `id` | SERIAL | Primary key |
| `symbol` | VARCHAR(50) | Ticker symbol |
| `name` | VARCHAR(255) | Company/asset name |
| `asset_type` | VARCHAR(20) | 'stock', 'etf', 'crypto', 'index', etc. |
| `exchange` | VARCHAR(50) | Exchange (NYSE, NASDAQ, etc.) |
| `sector` | VARCHAR(100) | GICS sector |
| `industry` | VARCHAR(100) | Industry |
| `country` | VARCHAR(50) | Country |
| `currency` | VARCHAR(20) | Trading currency |
| `market_cap` | BIGINT | Market capitalization |
| `cik` | VARCHAR(20) | SEC CIK number |
| `ipo_date` | DATE | IPO date |
| `employees` | INTEGER | Number of employees |
| `sic_code` | VARCHAR(10) | SIC code |
| `sic_description` | VARCHAR(255) | SIC description |
| `composite_figi` | VARCHAR(20) | Composite FIGI |
| `share_class_figi` | VARCHAR(20) | Share class FIGI |
| `weighted_shares_outstanding` | BIGINT | Shares outstanding |
| `active` | BOOLEAN | Currently tradeable |
| `market` | VARCHAR(20) | Market type ('stocks', 'crypto') |
| `locale` | VARCHAR(10) | Region ('us') |
| `delisted_date` | DATE | Delisted date (if applicable) |
| `created_at` | TIMESTAMP | Record creation time |
| `updated_at` | TIMESTAMP | Last update time |

**Asset Type Distribution:**

| Asset Type | Count |
|------------|-------|
| stock | 5,615 |
| etf | 4,210 |
| index | 12,608 |
| crypto | 1,000 |
| fund | 421 |
| preferred | 439 |
| adr | 377 |
| warrant | 418 |
| etn | 49 |
| right | 71 |
| **Total** | **~25,208** |

- **Data Source:** Polygon.io Tickers API
- **Update Cadency:** Daily
- **Used By:** Go backend for ticker lookups, search, and display

---

### 16. Companies (`companies`)

**Subset of tickers** used by IC Score pipelines. Contains only securities that need IC Score calculations (stocks + ETFs).

| Column | Type | Description |
|--------|------|-------------|
| `id` | SERIAL | Primary key |
| `ticker` | VARCHAR(10) | Stock symbol (unique) |
| `name` | VARCHAR(255) | Company name |
| `sector` | VARCHAR(100) | GICS sector |
| `industry` | VARCHAR(100) | Industry |
| `market_cap` | BIGINT | Market capitalization |
| `country` | VARCHAR(50) | Country |
| `exchange` | VARCHAR(50) | Exchange (NYSE, NASDAQ) |
| `currency` | VARCHAR(10) | Trading currency |
| `website` | VARCHAR(255) | Company website |
| `description` | TEXT | Company description |
| `employees` | INTEGER | Number of employees |
| `logo_url` | VARCHAR(255) | Company logo URL |
| `is_active` | BOOLEAN | Currently tracked |
| `last_updated` | TIMESTAMP | Last update time |
| `created_at` | TIMESTAMP | Record creation time |

- **Data Source:** Synced from `tickers` table via `sync_tickers_to_companies.py`
- **Update Cadency:** Weekly on Sundays at 2:00 AM UTC
- **CronJob:** `ic-score-ticker-sync`
- **Row Count:** ~10,000 (stocks + ETFs only, excludes indexes)
- **Used By:** All IC Score pipelines

### Relationship Between `tickers` and `companies`

```
┌─────────────────────────────────────────────────────────┐
│                    tickers (~25,000)                     │
│  ┌────────────┬────────────┬───────────┬──────────────┐ │
│  │  stocks    │   ETFs     │  crypto   │   indexes    │ │
│  │  (5,615)   │  (4,210)   │  (1,000)  │  (12,608)    │ │
│  └─────┬──────┴─────┬──────┴───────────┴──────────────┘ │
│        │            │                                    │
│        └──────┬─────┘                                    │
│               │ sync_tickers_to_companies.py             │
│               ▼                                          │
│  ┌────────────────────────┐                             │
│  │   companies (~10,000)   │  ◄─── IC Score pipelines   │
│  │   (stocks + ETFs)       │       use this table       │
│  └────────────────────────┘                             │
└─────────────────────────────────────────────────────────┘
```

---

## User Tables

| Table | Description | Update |
|-------|-------------|--------|
| `users` | User accounts | On user action |
| `watchlists` | User watchlists | On user action |
| `watchlist_stocks` | Stocks in watchlists | On user action |
| `portfolios` | User portfolios | On user action |
| `portfolio_positions` | Current holdings | On user action |
| `portfolio_transactions` | Transaction history | On user action |
| `alerts` | User alerts | On user action |
| `notifications` | User notifications | System generated |
| `sessions` | Active sessions | On login/logout |

---

## Update Schedule Summary

### Daily Pipeline Execution Order (UTC)

| Time (UTC) | CronJob | Description |
|------------|---------|-------------|
| 01:00 | `ic-score-benchmark-data` | Fetch benchmark returns (SPY, QQQ) |
| 02:00 | `ic-score-treasury-rates` | Fetch risk-free rates from FRED |
| 02:00 | `ic-score-sec-financials` | Weekly: SEC financials (Sundays only) |
| 02:00 | `ic-score-ticker-sync` | Weekly: Sync company list (Sundays only) |
| 03:00 | `ic-score-13f-holdings` | Quarterly: Form 13F (15th of quarter months) |
| 04:00 | `ic-score-analyst-ratings` | Analyst ratings updates |
| 05:00 | `ic-score-fundamental-metrics` | Extended fundamental metrics |
| 06:30 | `ic-score-refresh-sector-views` | Refresh sector comparison views |
| 07:00 | `ic-score-fair-value` | DCF and Graham fair values |
| 08:00 | `ic-score-coverage-monitor` | Data coverage monitoring |
| 14:00-21:00 | `ic-score-insider-trades` | Hourly: Form 4 insider trades (market hours) |
| 22:00 | `ic-score-ttm-financials` | TTM financials calculation (Mon-Fri) |
| 22:30 | `ic-score-daily-price-update` | Daily stock prices (Mon-Fri) |
| 23:00 | `ic-score-technical-indicators` | Technical indicators (Mon-Fri) |
| 23:30 | `ic-score-valuation-ratios` | Valuation ratios (Mon-Fri) |
| 00:00 | `ic-score-risk-metrics` | Risk metrics (Tue-Sat) |
| 00:00 | `ic-score-calculator` | Final IC Score calculation |
| */4 hrs | `ic-score-news-sentiment` | News sentiment (every 4 hours) |

---

## Pipeline Dependencies

```
                          ┌─────────────────────┐
                          │   benchmark_data    │
                          │   treasury_rates    │
                          └─────────┬───────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
           ┌────────▼────────┐     │      ┌────────▼────────┐
           │  daily_price    │     │      │  sec_financials │
           │    (22:30)      │     │      │   (Sundays)     │
           └────────┬────────┘     │      └────────┬────────┘
                    │               │               │
           ┌────────▼────────┐     │      ┌────────▼────────┐
           │   technical     │     │      │  ttm_financials │
           │    (23:00)      │     │      │    (22:00)      │
           └────────┬────────┘     │      └────────┬────────┘
                    │               │               │
           ┌────────▼────────┐     │      ┌────────▼────────┐
           │  risk_metrics   │◄────┘      │valuation_ratios │
           │   (00:00)       │            │    (23:30)      │
           └────────┬────────┘            └────────┬────────┘
                    │                              │
           ┌────────┴───────────────┬──────────────┘
           │                        │
           │  ┌─────────────────┐  ┌▼────────────────┐  ┌─────────────────┐
           │  │ insider_trades  │  │ fundamental     │  │ news_sentiment  │
           │  │  (hourly)       │  │ metrics (05:00) │  │ (4-hourly)      │
           │  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘
           │           │                    │                    │
           │  ┌────────▼────────┐  ┌────────▼────────┐          │
           │  │ 13f_holdings    │  │  fair_value     │          │
           │  │ (quarterly)     │  │   (07:00)       │          │
           │  └────────┬────────┘  └────────┬────────┘          │
           │           │                    │                    │
           └───────────┴────────────────────┴────────────────────┘
                                    │
                           ┌────────▼────────┐
                           │  ic_score_calc  │
                           │   (midnight)    │
                           └─────────────────┘
```

---

## Data Sources

### Primary Data Sources

| Source | Data Provided | API Rate Limit |
|--------|---------------|----------------|
| Polygon.io | Stock prices, benchmarks, analyst ratings, news | 5 req/sec (free tier) |
| SEC EDGAR | 10-K, 10-Q, Form 4, Form 13F | 10 req/sec |
| FRED | Treasury rates | 120 req/min |
| OpenAI | News sentiment analysis | Based on plan |

### Data Coverage

| Metric | Coverage |
|--------|----------|
| Stocks tracked | ~5,600 US equities |
| ETFs tracked | ~4,200 ETFs |
| Price history | 10 years (2015-present) |
| Financials history | 5+ years |
| News retention | 90 days |
| Technical indicators | All active stocks |
| Risk metrics | All stocks with 252+ days history |

---

## Environment Variables

```bash
# Polygon.io
POLYGON_API_KEY=your_polygon_api_key

# Database
DB_HOST=postgres-simple-service
DB_PORT=5432
DB_NAME=investorcenter_db
DB_USER=investorcenter
DB_PASSWORD=your_password
DB_SSLMODE=disable

# OpenAI (for sentiment)
OPENAI_API_KEY=your_openai_key

# SEC EDGAR
SEC_USER_AGENT="YourName your.email@example.com"
```

---

## Quick Reference Commands

### View All CronJobs

```bash
kubectl get cronjobs -n investorcenter
```

### Check CronJob Status

```bash
kubectl get jobs -n investorcenter -l component=ic-score-pipeline
```

### View Logs for a Specific Pipeline

```bash
kubectl logs -n investorcenter -l app=ic-score-daily-price-update --tail=100
```

### Manually Trigger a CronJob

```bash
kubectl create job --from=cronjob/ic-score-daily-price-update manual-price-update -n investorcenter
```

### Check Data Freshness

```bash
kubectl exec -n investorcenter deployment/postgres-simple -- psql -U investorcenter -d investorcenter_db -c "
SELECT
  'stock_prices' as table_name, MAX(time)::date as latest_data FROM stock_prices
UNION ALL
SELECT 'technical_indicators', MAX(time)::date FROM technical_indicators
UNION ALL
SELECT 'risk_metrics', MAX(time)::date FROM risk_metrics
UNION ALL
SELECT 'ic_scores', MAX(date) FROM ic_scores
UNION ALL
SELECT 'ttm_financials', MAX(calculation_date) FROM ttm_financials
UNION ALL
SELECT 'valuation_ratios', MAX(calculation_date) FROM valuation_ratios
UNION ALL
SELECT 'fundamental_metrics_extended', MAX(calculation_date) FROM fundamental_metrics_extended;
"
```

### Check Record Counts

```bash
kubectl exec -n investorcenter deployment/postgres-simple -- psql -U investorcenter -d investorcenter_db -c "
SELECT 'tickers' as table_name, COUNT(*) FROM tickers
UNION ALL SELECT 'companies', COUNT(*) FROM companies
UNION ALL SELECT 'stock_prices', COUNT(*) FROM stock_prices
UNION ALL SELECT 'financials', COUNT(*) FROM financials
UNION ALL SELECT 'ic_scores', COUNT(*) FROM ic_scores;
"
```

---

## Materialized Views

### Sector Comparison Views

Created by migration `014_create_sector_comparison_views.sql`:

| View | Description | Refresh |
|------|-------------|---------|
| `sector_metric_averages` | Sector-level average metrics | Daily at 6:30 AM UTC |
| `industry_metric_averages` | Industry-level average metrics | Daily at 6:30 AM UTC |

**Function:**
- `calculate_sector_percentile(ticker, metric_name)` - Returns percentile rank within sector

---

## Database Indexes

Key indexes for query performance:

| Table | Index | Columns |
|-------|-------|---------|
| `stock_prices` | TimescaleDB hypertable | `time`, `ticker` |
| `technical_indicators` | TimescaleDB hypertable | `time`, `ticker`, `indicator_name` |
| `financials` | `idx_financials_ticker_period` | `ticker`, `period_end_date` |
| `ic_scores` | `idx_ic_scores_ticker_date` | `ticker`, `date` |
| `ttm_financials` | `idx_ttm_financials_ticker` | `ticker`, `calculation_date` |
| `fundamental_metrics_extended` | `uq_fundamental_metrics_ticker_date` | `ticker`, `calculation_date` |

---

**Last Updated:** November 27, 2025
