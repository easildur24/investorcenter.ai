# InvestorCenter.ai Data Schema and Update Cadency

This document provides a comprehensive overview of all data stored in the IC Score Service database, including schemas, data sources, and update schedules.

## Table of Contents
1. [Time-Series Data](#time-series-data)
2. [Periodic Financial Data](#periodic-financial-data)
3. [Calculated Metrics](#calculated-metrics)
4. [User Data](#user-data)
5. [Update Schedule Summary](#update-schedule-summary)
6. [Data Sources](#data-sources)

---

## Time-Series Data

These tables store data that changes daily and are optimized using TimescaleDB hypertables for efficient time-series queries.

### 1. Stock Prices (`stock_prices`)
Daily OHLCV (Open, High, Low, Close, Volume) price data for all tracked stocks.

| Column | Type | Description |
|--------|------|-------------|
| time | TIMESTAMPTZ | Trading date/time |
| ticker | VARCHAR(10) | Stock symbol |
| open | DECIMAL(10,2) | Opening price |
| high | DECIMAL(10,2) | Day high |
| low | DECIMAL(10,2) | Day low |
| close | DECIMAL(10,2) | Closing price |
| volume | BIGINT | Trading volume |
| vwap | DECIMAL(10,2) | Volume-weighted average price |
| interval | VARCHAR(10) | Time interval ('1day') |

**Data Source:** Polygon.io API
**Update Cadency:** Daily at 10:30 PM UTC (Mon-Fri, after market close)
**CronJob:** `ic-score-daily-price-update`
**History:** 10 years of historical data (backfilled)

#### Historical Price Backfill

For new deployments or when adding new tickers, use the historical price backfill script:

```bash
# Backfill all tickers with 10 years of data
python -m pipelines.historical_price_backfill --all --resume

# Backfill single ticker
python -m pipelines.historical_price_backfill --ticker AAPL

# Test on limited set first
python -m pipelines.historical_price_backfill --limit 100 --resume
```

**Backfill Performance:**
- Uses `tickers` table filtered for stocks + ETFs + ADRs + funds (~10,600 active tickers)
- Expected time: ~2-4 hours with Polygon Business tier (unlimited req/sec)
- `--resume` flag skips tickers that already have sufficient historical data

**Database Indexes for Performance:**
```sql
-- Critical for backfill resume check
CREATE INDEX idx_stock_prices_ticker_time ON stock_prices(ticker, time);
```

---

### 2. Technical Indicators (`technical_indicators`)
Calculated technical analysis indicators for each stock.

| Column | Type | Description |
|--------|------|-------------|
| time | TIMESTAMPTZ | Calculation date |
| ticker | VARCHAR(10) | Stock symbol |
| indicator_name | VARCHAR(50) | Indicator type |
| value | DECIMAL(18,6) | Calculated value |
| metadata | JSONB | Additional parameters |

**Indicators Calculated:**
- RSI (Relative Strength Index) - 14-day
- MACD (Moving Average Convergence Divergence)
- MACD Signal
- MACD Histogram
- SMA_20 (20-day Simple Moving Average)
- SMA_50 (50-day Simple Moving Average)
- SMA_200 (200-day Simple Moving Average)
- EMA_12 (12-day Exponential Moving Average)
- EMA_26 (26-day Exponential Moving Average)
- Bollinger Bands (Upper, Middle, Lower)
- ATR (Average True Range)
- OBV (On-Balance Volume)
- ADX (Average Directional Index)
- Stochastic %K and %D

**Data Source:** Calculated from stock_prices
**Update Cadency:** Daily at 11:00 PM UTC (Mon-Fri)
**CronJob:** `ic-score-technical-indicators`
**Dependencies:** Requires stock_prices to be updated first

---

### 3. Risk Metrics (`risk_metrics`)
Risk and performance metrics calculated relative to benchmarks.

| Column | Type | Description |
|--------|------|-------------|
| id | BIGSERIAL | Primary key |
| ticker | VARCHAR(10) | Stock symbol |
| date | DATE | Calculation date |
| alpha | DECIMAL(10,6) | Jensen's Alpha |
| beta | DECIMAL(10,6) | Market Beta |
| sharpe_ratio | DECIMAL(10,6) | Sharpe Ratio |
| sortino_ratio | DECIMAL(10,6) | Sortino Ratio |
| max_drawdown | DECIMAL(10,6) | Maximum Drawdown |
| volatility | DECIMAL(10,6) | Annual Volatility |
| var_95 | DECIMAL(10,6) | Value at Risk (95%) |
| cvar_95 | DECIMAL(10,6) | Conditional VaR (95%) |
| tracking_error | DECIMAL(10,6) | Tracking Error |
| information_ratio | DECIMAL(10,6) | Information Ratio |
| treynor_ratio | DECIMAL(10,6) | Treynor Ratio |
| calmar_ratio | DECIMAL(10,6) | Calmar Ratio |
| r_squared | DECIMAL(10,6) | R-Squared (correlation) |
| benchmark | VARCHAR(20) | Benchmark used (SPY) |
| lookback_period | INTEGER | Days used (252 = 1 year) |

**Data Source:** Calculated from stock_prices + benchmark_returns + treasury_rates
**Update Cadency:** Daily at 12:00 AM UTC (Tue-Sat, after previous day's prices)
**CronJob:** `ic-score-risk-metrics`
**Dependencies:** Requires stock_prices, benchmark_returns, treasury_rates

---

### 4. Benchmark Returns (`benchmark_returns`)
Daily returns for benchmark indices (SPY, QQQ, etc.).

| Column | Type | Description |
|--------|------|-------------|
| date | DATE | Trading date |
| benchmark | VARCHAR(20) | Benchmark symbol |
| close | DECIMAL(10,4) | Closing price |
| daily_return | DECIMAL(10,6) | Daily return percentage |

**Data Source:** Polygon.io API
**Update Cadency:** Daily at 1:00 AM UTC
**CronJob:** `ic-score-benchmark-data`

---

### 5. Treasury Rates (`treasury_rates`)
Daily risk-free rates from US Treasury.

| Column | Type | Description |
|--------|------|-------------|
| date | DATE | Observation date |
| rate_1m | DECIMAL(10,4) | 1-month T-bill rate |
| rate_3m | DECIMAL(10,4) | 3-month T-bill rate |
| rate_6m | DECIMAL(10,4) | 6-month T-bill rate |
| rate_1y | DECIMAL(10,4) | 1-year T-bill rate |
| rate_2y | DECIMAL(10,4) | 2-year Treasury rate |
| rate_5y | DECIMAL(10,4) | 5-year Treasury rate |
| rate_10y | DECIMAL(10,4) | 10-year Treasury rate |
| rate_30y | DECIMAL(10,4) | 30-year Treasury rate |

**Data Source:** FRED (Federal Reserve Economic Data)
**Update Cadency:** Daily at 2:00 AM UTC
**CronJob:** `ic-score-treasury-rates`

---

## Periodic Financial Data

These tables store data that updates on fixed schedules (quarterly, weekly, etc.).

### 6. Financials (`financials`)
Quarterly and annual financial statements from SEC filings.

| Column | Type | Description |
|--------|------|-------------|
| id | BIGSERIAL | Primary key |
| ticker | VARCHAR(10) | Stock symbol |
| filing_date | DATE | SEC filing date |
| period_end_date | DATE | Fiscal period end |
| fiscal_year | INT | Fiscal year |
| fiscal_quarter | INT | Quarter (NULL for annual) |
| statement_type | VARCHAR(20) | '10-K' or '10-Q' |
| **Income Statement** | | |
| revenue | BIGINT | Total revenue |
| cost_of_revenue | BIGINT | Cost of goods sold |
| gross_profit | BIGINT | Gross profit |
| operating_income | BIGINT | Operating income |
| net_income | BIGINT | Net income |
| eps_basic | DECIMAL(10,4) | Basic EPS |
| eps_diluted | DECIMAL(10,4) | Diluted EPS |
| **Balance Sheet** | | |
| total_assets | BIGINT | Total assets |
| total_liabilities | BIGINT | Total liabilities |
| shareholders_equity | BIGINT | Shareholders' equity |
| cash_and_equivalents | BIGINT | Cash and equivalents |
| long_term_debt | BIGINT | Long-term debt |
| **Cash Flow** | | |
| operating_cash_flow | BIGINT | Cash from operations |
| free_cash_flow | BIGINT | Free cash flow |
| **Calculated Ratios** | | |
| pe_ratio | DECIMAL(10,2) | Price-to-Earnings |
| debt_to_equity | DECIMAL(10,2) | Debt/Equity ratio |
| roe | DECIMAL(10,2) | Return on Equity |

**Data Source:** SEC EDGAR (10-K, 10-Q filings)
**Update Cadency:** Weekly on Sundays at 2:00 AM UTC
**CronJob:** `ic-score-sec-financials`

---

### 7. TTM Financials (`ttm_financials`)
Trailing Twelve Months (TTM) financial metrics calculated from quarterly data.

| Column | Type | Description |
|--------|------|-------------|
| ticker | VARCHAR(10) | Stock symbol |
| calculation_date | DATE | When calculated |
| period_end_date | DATE | Latest quarter end |
| ttm_revenue | BIGINT | Sum of last 4 quarters |
| ttm_net_income | BIGINT | Sum of last 4 quarters |
| ttm_eps_basic | DECIMAL(10,4) | TTM Basic EPS |
| ttm_eps_diluted | DECIMAL(10,4) | TTM Diluted EPS |
| ttm_free_cash_flow | BIGINT | TTM FCF |
| ttm_operating_cash_flow | BIGINT | TTM Operating CF |
| quarters_used | INTEGER | Number of quarters (4) |
| quarters_data | JSONB | Quarter breakdown |

**Data Source:** Calculated from financials table
**Update Cadency:** Weekly on Sundays at 3:00 AM UTC
**CronJob:** `ic-score-sec-financials` (runs after financials)

---

### 8. Insider Trades (`insider_trades`)
SEC Form 4 insider trading activity.

| Column | Type | Description |
|--------|------|-------------|
| ticker | VARCHAR(10) | Stock symbol |
| filing_date | DATE | SEC filing date |
| transaction_date | DATE | Trade date |
| insider_name | VARCHAR(255) | Insider's name |
| insider_title | VARCHAR(255) | Position/title |
| transaction_type | VARCHAR(50) | 'Buy', 'Sell', 'Option Exercise' |
| shares | BIGINT | Number of shares |
| price_per_share | DECIMAL(10,2) | Transaction price |
| total_value | BIGINT | Total transaction value |
| shares_owned_after | BIGINT | Shares after transaction |

**Data Source:** SEC EDGAR (Form 4)
**Update Cadency:** Hourly during market hours (2-9 PM UTC, Mon-Fri)
**CronJob:** `ic-score-insider-trades`

---

### 9. Institutional Holdings (`institutional_holdings`)
SEC Form 13F institutional ownership data.

| Column | Type | Description |
|--------|------|-------------|
| ticker | VARCHAR(10) | Stock symbol |
| filing_date | DATE | SEC filing date |
| quarter_end_date | DATE | Quarter end |
| institution_name | VARCHAR(255) | Institution name |
| institution_cik | VARCHAR(20) | SEC CIK |
| shares | BIGINT | Shares held |
| market_value | BIGINT | Position value |
| percent_of_portfolio | DECIMAL(10,4) | % of portfolio |
| position_change | VARCHAR(50) | 'New', 'Increased', etc. |

**Data Source:** SEC EDGAR (Form 13F)
**Update Cadency:** Quarterly (15th of Jan, Apr, Jul, Oct at 3:00 AM UTC)
**CronJob:** `ic-score-13f-holdings`

---

### 10. Analyst Ratings (`analyst_ratings`)
Wall Street analyst ratings and price targets.

| Column | Type | Description |
|--------|------|-------------|
| ticker | VARCHAR(10) | Stock symbol |
| rating_date | DATE | Rating date |
| analyst_name | VARCHAR(255) | Analyst name |
| analyst_firm | VARCHAR(255) | Firm name |
| rating | VARCHAR(50) | 'Strong Buy' to 'Strong Sell' |
| rating_numeric | DECIMAL(3,1) | 1.0 to 5.0 |
| price_target | DECIMAL(10,2) | Target price |
| action | VARCHAR(50) | 'Initiated', 'Upgraded', etc. |

**Data Source:** Polygon.io (Analyst Data)
**Update Cadency:** Daily at 4:00 AM UTC
**CronJob:** `ic-score-analyst-ratings`

---

### 11. News Articles (`news_articles`)
News with AI-powered sentiment analysis.

| Column | Type | Description |
|--------|------|-------------|
| title | VARCHAR(500) | Article title |
| url | VARCHAR(1000) | Article URL (unique) |
| source | VARCHAR(255) | News source |
| published_at | TIMESTAMP | Publication time |
| tickers | VARCHAR(50)[] | Related tickers |
| sentiment_score | DECIMAL(5,2) | -100 to +100 |
| sentiment_label | VARCHAR(20) | 'Positive', 'Neutral', 'Negative' |
| relevance_score | DECIMAL(5,2) | 0 to 100 |

**Data Source:** Polygon.io (News API) + OpenAI (Sentiment)
**Update Cadency:** Every 4 hours
**CronJob:** `ic-score-news-sentiment`

---

## Calculated Metrics

### 12. Valuation Ratios (`valuation_ratios`)
Real-time valuation metrics using TTM financials and current prices.

| Column | Type | Description |
|--------|------|-------------|
| ticker | VARCHAR(10) | Stock symbol |
| calculation_date | DATE | When calculated |
| pe_ratio | DECIMAL(10,2) | Price/TTM EPS |
| forward_pe | DECIMAL(10,2) | Price/Forward EPS |
| peg_ratio | DECIMAL(10,2) | PE/Growth |
| ps_ratio | DECIMAL(10,2) | Price/Sales |
| pb_ratio | DECIMAL(10,2) | Price/Book |
| ev_ebitda | DECIMAL(10,2) | EV/EBITDA |
| pcf_ratio | DECIMAL(10,2) | Price/Cash Flow |
| dividend_yield | DECIMAL(10,4) | Annual dividend yield |

**Data Source:** Calculated from stock_prices + ttm_financials
**Update Cadency:** Daily at 11:30 PM UTC (Mon-Fri)
**CronJob:** `ic-score-valuation-ratios`

---

### 13. IC Scores (`ic_scores`)
Proprietary 10-factor stock scores.

| Column | Type | Description |
|--------|------|-------------|
| ticker | VARCHAR(10) | Stock symbol |
| date | DATE | Score date |
| overall_score | DECIMAL(5,2) | 1-100 composite score |
| value_score | DECIMAL(5,2) | Value factor |
| growth_score | DECIMAL(5,2) | Growth factor |
| profitability_score | DECIMAL(5,2) | Profitability factor |
| financial_health_score | DECIMAL(5,2) | Balance sheet strength |
| momentum_score | DECIMAL(5,2) | Price momentum |
| analyst_consensus_score | DECIMAL(5,2) | Analyst ratings |
| insider_activity_score | DECIMAL(5,2) | Insider trading |
| institutional_score | DECIMAL(5,2) | Institutional ownership |
| news_sentiment_score | DECIMAL(5,2) | News sentiment |
| technical_score | DECIMAL(5,2) | Technical indicators |
| rating | VARCHAR(20) | 'Strong Buy' to 'Sell' |
| sector_percentile | DECIMAL(5,2) | Percentile in sector |
| confidence_level | VARCHAR(20) | 'High', 'Medium', 'Low' |
| data_completeness | DECIMAL(5,2) | % factors with data |

**Data Source:** Calculated from all other data sources
**Update Cadency:** Daily at 12:00 AM UTC
**CronJob:** `ic-score-calculator`
**Dependencies:** All other pipelines must complete first

---

## Master Data Tables

### 14. Tickers (`tickers`) - Primary Master Table
Master list of ALL tradeable assets from Polygon.io. This is the **source of truth** for the Go backend.

| Column | Type | Description |
|--------|------|-------------|
| id | SERIAL | Primary key |
| symbol | VARCHAR(50) | Ticker symbol (unique with asset_type) |
| name | VARCHAR(255) | Company/asset name |
| asset_type | VARCHAR(20) | 'stock', 'etf', 'crypto', 'index', etc. |
| exchange | VARCHAR(50) | Exchange (NYSE, NASDAQ, etc.) |
| cik | VARCHAR(20) | SEC CIK number |
| active | BOOLEAN | Currently tradeable |
| market | VARCHAR(20) | Market type ('stocks', 'crypto') |
| locale | VARCHAR(10) | Region ('us') |

**Data Source:** Polygon.io Tickers API
**Update Cadency:** Daily via `polygon-ticker-update` CronJob
**Row Count:** ~25,000 (includes stocks, ETFs, crypto, indexes)
**Used By:** Go backend for ticker lookups, search, and display

---

### 15. Companies (`companies`) - IC Score Service Table
**Subset of tickers** used by IC Score pipelines. Contains only securities that need IC Score calculations (stocks + ETFs).

| Column | Type | Description |
|--------|------|-------------|
| id | SERIAL | Primary key |
| ticker | VARCHAR(10) | Stock symbol (unique) |
| name | VARCHAR(255) | Company name |
| sector | VARCHAR(100) | GICS sector |
| industry | VARCHAR(100) | Industry |
| market_cap | BIGINT | Market capitalization |
| exchange | VARCHAR(50) | Exchange (NYSE, NASDAQ) |
| cik | VARCHAR(20) | SEC CIK number |
| is_active | BOOLEAN | Currently tracked |

**Data Source:** Synced from `tickers` table via `sync_tickers_to_companies.py`
**Update Cadency:** Weekly on Sundays at 2:00 AM UTC
**CronJob:** `ic-score-ticker-sync`
**Row Count:** ~10,000 (stocks + ETFs only, excludes indexes)
**Used By:** All IC Score pipelines (price backfill, technical indicators, financials, etc.)

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

**Why Two Tables?**
- `tickers`: Complete market data from Polygon (includes indexes which are calculated values with no price data)
- `companies`: Curated subset for IC Score calculations (only securities with tradeable prices and SEC filings)

**Sync Script:** `ic-score-service/scripts/sync_tickers_to_companies.py`
```bash
# Sync all missing tickers
python scripts/sync_tickers_to_companies.py

# Sync only priority stocks (active US stocks with CIK)
python scripts/sync_tickers_to_companies.py --priority-only
```

---

## User Data

---

### 15-20. User Tables

| Table | Description | Update |
|-------|-------------|--------|
| users | User accounts | On user action |
| watchlists | User watchlists | On user action |
| watchlist_stocks | Stocks in watchlists | On user action |
| portfolios | User portfolios | On user action |
| portfolio_positions | Current holdings | On user action |
| portfolio_transactions | Transaction history | On user action |
| alerts | User alerts | On user action |

---

## Update Schedule Summary

### Daily Pipeline Execution Order (UTC)

| Time | CronJob | Description |
|------|---------|-------------|
| 01:00 | `ic-score-benchmark-data` | Fetch benchmark returns (SPY, QQQ) |
| 02:00 | `ic-score-treasury-rates` | Fetch risk-free rates from FRED |
| 02:00 | `ic-score-sec-financials` | Weekly: SEC financials (Sundays only) |
| 02:00 | `ic-score-ticker-sync` | Weekly: Sync company list (Sundays only) |
| 03:00 | `ic-score-13f-holdings` | Quarterly: Form 13F (15th of quarter months) |
| 04:00 | `ic-score-analyst-ratings` | Analyst ratings updates |
| 08:00 | `ic-score-coverage-monitor` | Data coverage monitoring |
| 14:00-21:00 | `ic-score-insider-trades` | Hourly: Form 4 insider trades (market hours) |
| 22:30 | `ic-score-daily-price-update` | **Phase 1**: Daily stock prices (Mon-Fri) |
| 23:00 | `ic-score-technical-indicators` | **Phase 2**: Technical indicators (Mon-Fri) |
| 23:30 | `ic-score-valuation-ratios` | Valuation ratios (Mon-Fri) |
| 00:00 | `ic-score-risk-metrics` | **Phase 3**: Risk metrics (Tue-Sat) |
| 00:00 | `ic-score-calculator` | Final IC Score calculation |
| */4 hrs | `ic-score-news-sentiment` | News sentiment (every 4 hours) |

### Pipeline Dependencies

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
           │    (Phase 1)    │     │      │   (Sundays)     │
           └────────┬────────┘     │      └────────┬────────┘
                    │               │               │
           ┌────────▼────────┐     │      ┌────────▼────────┐
           │   technical     │     │      │  ttm_financials │
           │  (Phase 2)      │     │      └────────┬────────┘
           └────────┬────────┘     │               │
                    │               │      ┌────────▼────────┐
           ┌────────▼────────┐     │      │valuation_ratios │
           │  risk_metrics   │◄────┘      └────────┬────────┘
           │   (Phase 3)     │                     │
           └────────┬────────┘                     │
                    │                              │
           ┌────────┴──────────────────────────────┘
           │
           │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
           │  │ insider_trades  │  │ 13f_holdings    │  │ news_sentiment  │
           │  │  (hourly)       │  │ (quarterly)     │  │ (4-hourly)      │
           │  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘
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
| **Polygon.io** | Stock prices, benchmarks, analyst ratings, news | 5 req/sec (free tier) |
| **SEC EDGAR** | 10-K, 10-Q, Form 4, Form 13F | 10 req/sec |
| **FRED** | Treasury rates | 120 req/min |
| **OpenAI** | News sentiment analysis | Based on plan |

### Data Coverage

| Metric | Coverage |
|--------|----------|
| Stocks tracked | ~4,600 US equities |
| Price history | 10 years (2015-present) |
| Financials history | 5+ years |
| News retention | 90 days |
| Technical indicators | All active stocks |
| Risk metrics | All stocks with 252+ days history |

### Environment Variables Required

```bash
# Polygon.io
POLYGON_API_KEY=your_polygon_api_key

# Database
DB_HOST=postgres-simple-service
DB_PORT=5432
DB_NAME=investorcenter_db
DB_USER=investorcenter
DB_PASSWORD=your_password

# OpenAI (for sentiment)
OPENAI_API_KEY=your_openai_key

# SEC EDGAR
SEC_USER_AGENT="YourName your.email@example.com"
```

---

## Quick Reference: CronJob Commands

```bash
# View all CronJobs
kubectl get cronjobs -n investorcenter

# Check CronJob status
kubectl get jobs -n investorcenter -l component=ic-score-pipeline

# View logs for a specific pipeline
kubectl logs -n investorcenter -l app=ic-score-daily-price-update --tail=100

# Manually trigger a CronJob
kubectl create job --from=cronjob/ic-score-daily-price-update manual-price-update -n investorcenter

# Check data freshness
kubectl exec -n investorcenter deployment/postgres-simple -- psql -U investorcenter -d investorcenter_db -c "
SELECT
  'stock_prices' as table_name, MAX(time)::date as latest_data FROM stock_prices
UNION ALL
SELECT 'technical_indicators', MAX(time)::date FROM technical_indicators
UNION ALL
SELECT 'risk_metrics', MAX(date) FROM risk_metrics
UNION ALL
SELECT 'ic_scores', MAX(date) FROM ic_scores;
"
```

---

*Last Updated: November 25, 2025*
