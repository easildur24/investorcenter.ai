# Implementation Prompt: IC Score Data Ingestion Pipelines

## Project Context

InvestorCenter.ai is a financial data platform similar to YCharts. We're building a proprietary **IC Score system** that rates stocks 1-100 using 10 financial factors. The database schema is already created, and we need to implement data ingestion pipelines to populate it.

## Current State

### ✅ Already Completed
- **Database schema**: 16 tables created in `ic-score-service/database/schema.sql`
- **SQLAlchemy models**: All models defined in `ic-score-service/models.py`
- **Database connection**: Async session management in `ic-score-service/database/database.py`
- **Alembic migrations**: Initial migration in `ic-score-service/migrations/versions/001_initial_schema.py`
- **Seed data**: Sample data script in `ic-score-service/scripts/seed.py`

### 🎯 What Needs Implementation
7 data ingestion pipelines across 5 phases to populate the database with real financial data for 4,600 US stocks.

## Existing Infrastructure (To Leverage)

### Go Backend (Already Working)
- **Location**: `/Users/esun/code/investorcenter.ai/backend/`
- **Polygon.io Integration**: `backend/services/polygon.go` (lines 119-702)
  - Stock prices (OHLCV data)
  - Financial statements API
  - News API with sentiment
- **Database**: PostgreSQL at localhost:5432, database `investorcenter_db`
- **Stocks table**: 4,600 US stocks with CIK numbers in `stocks` table

### Kubernetes Deployment
- **Cluster**: `investorcenter-eks` on AWS EKS
- **Namespace**: `investorcenter`
- **Existing CronJobs**: Ticker updates, crypto updates, volume updates
- **Pattern to follow**: See `k8s/polygon-ticker-updater-cronjob.yaml`

### Environment Variables
```bash
POLYGON_API_KEY=<already exists>
DB_HOST=postgres.investorcenter.svc.cluster.local
DB_PORT=5432
DB_NAME=investorcenter_db
DB_USER=postgres
DB_PASSWORD=<in secrets>
```

## Implementation Plan

### Phase 1: SEC Financial Statements (Priority: CRITICAL)
**Timeline**: Weeks 1-2
**Cost**: FREE (SEC EDGAR API)
**IC Score Coverage**: 40% (4 factors)

#### Task 1.1: Create SEC EDGAR Client
**File**: `ic-score-service/pipelines/utils/sec_client.py`

**Requirements**:
- Fetch financial data from SEC EDGAR API: `https://data.sec.gov/api/xbrl/companyfacts/CIK{cik}.json`
- Rate limit: 10 requests/second with proper backoff
- Parse XBRL financial facts (income statement, balance sheet, cash flow)
- Extract quarterly (10-Q) and annual (10-K) data
- Handle missing data gracefully

**Key Metrics to Extract**:
- Income Statement: revenue, cost_of_revenue, gross_profit, operating_expenses, operating_income, net_income, eps_basic, eps_diluted
- Balance Sheet: total_assets, total_liabilities, shareholders_equity, cash_and_equivalents, short_term_debt, long_term_debt
- Cash Flow: operating_cash_flow, investing_cash_flow, financing_cash_flow, free_cash_flow, capex

**Calculated Ratios**:
- Valuation: pe_ratio, pb_ratio, ps_ratio
- Leverage: debt_to_equity, current_ratio, quick_ratio
- Profitability: roe, roa, roic, gross_margin, operating_margin, net_margin

#### Task 1.2: SEC Financials Ingestion Script
**File**: `ic-score-service/pipelines/sec_financials_ingestion.py`

**Requirements**:
- Read all stocks from main database `stocks` table (use CIK numbers)
- For each stock, fetch last 5 years of quarterly data
- Calculate all financial ratios
- Insert into `financials` table in IC Score database
- Use batch processing (100 stocks per batch)
- Progress logging with estimated completion time
- Resume capability if interrupted
- Target database: `investorcenter_db` (same as main DB, just different schema/tables)

**Database Connection**:
```python
# Connect to main DB to read stocks
from sqlalchemy import create_engine, text
main_engine = create_engine("postgresql://postgres@localhost:5432/investorcenter_db")

# Connect to IC Score tables (same DB, different schema if needed)
# Use existing database.py module
from database import get_session
```

**CLI Interface**:
```bash
python pipelines/sec_financials_ingestion.py --limit 100  # Test on 100 stocks
python pipelines/sec_financials_ingestion.py --all        # All 4,600 stocks
python pipelines/sec_financials_ingestion.py --ticker AAPL # Single stock
python pipelines/sec_financials_ingestion.py --resume     # Resume from last
```

#### Task 1.3: Kubernetes CronJob
**File**: `k8s/ic-score-sec-financials-cronjob.yaml`

**Requirements**:
- Schedule: Weekly on Sunday at 2am UTC (`0 2 * * 0`)
- Container: Python 3.11 with dependencies
- Resource limits: 2 CPU, 4Gi memory
- Timeout: 12 hours
- Restart policy: OnFailure
- Environment variables from secrets
- Follow pattern from `k8s/polygon-ticker-updater-cronjob.yaml`

---

### Phase 2: Technical Indicators (Priority: HIGH)
**Timeline**: Week 3
**Cost**: INCLUDED (Polygon.io)
**IC Score Coverage**: +20% (2 additional factors)

#### Task 2.1: Technical Indicators Calculator
**File**: `ic-score-service/pipelines/technical_indicators_calculator.py`

**Requirements**:
- Fetch 252 days of price data from Polygon.io API
- Calculate indicators using `ta-lib` or `pandas-ta`:
  - RSI (14-day)
  - MACD (12, 26, 9)
  - SMA (50-day, 200-day)
  - EMA (12-day, 26-day)
  - Bollinger Bands (20-day, 2 std dev)
  - Volume moving average (20-day)
- Calculate momentum metrics:
  - 1-month return
  - 3-month return
  - 6-month return
  - 12-month return
- Insert into `technical_indicators` TimescaleDB hypertable
- Insert raw prices into `stock_prices` hypertable

**Polygon.io Endpoint**:
```
GET /v2/aggs/ticker/{ticker}/range/1/day/{from}/{to}
```

**CLI Interface**:
```bash
python pipelines/technical_indicators_calculator.py --limit 100
python pipelines/technical_indicators_calculator.py --all
python pipelines/technical_indicators_calculator.py --ticker AAPL
```

#### Task 2.2: Kubernetes CronJob
**File**: `k8s/ic-score-technical-indicators-cronjob.yaml`

**Requirements**:
- Schedule: Daily at 6pm ET / 11pm UTC on weekdays (`0 23 * * 1-5`)
- After market close + existing price updates
- Resource limits: 4 CPU, 8Gi memory
- Timeout: 2 hours

---

### Phase 3: Insider & Institutional Data (Priority: HIGH)
**Timeline**: Week 4
**Cost**: FREE (SEC EDGAR)
**IC Score Coverage**: +20% (2 additional factors)

#### Task 3.1: Insider Trades Ingestion
**File**: `ic-score-service/pipelines/sec_insider_trades_ingestion.py`

**Requirements**:
- Poll SEC RSS feed: `https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=4&output=atom`
- Download and parse Form 4 XML filings
- Extract: insider_name, insider_title, transaction_type, shares, price_per_share, total_value
- Map CIK to ticker using main DB `stocks` table
- Insert into `insider_trades` table
- Track net buying/selling over 30-day and 90-day windows

**CLI Interface**:
```bash
python pipelines/sec_insider_trades_ingestion.py --hours 24  # Last 24 hours
python pipelines/sec_insider_trades_ingestion.py --backfill 90  # Last 90 days
```

#### Task 3.2: Institutional Holdings Ingestion
**File**: `ic-score-service/pipelines/sec_13f_ingestion.py`

**Requirements**:
- Download quarterly 13F bulk files: `https://www.sec.gov/data-research/sec-markets-data/form-13f-data-sets`
- Parse XML/CSV for institutional holdings
- Extract: institution_name, institution_cik, shares, market_value
- Calculate quarter-over-quarter changes
- Insert into `institutional_holdings` table

**CLI Interface**:
```bash
python pipelines/sec_13f_ingestion.py --quarter 2024Q3
python pipelines/sec_13f_ingestion.py --backfill 4  # Last 4 quarters
```

#### Task 3.3: Kubernetes CronJobs
**Files**:
- `k8s/ic-score-insider-trades-cronjob.yaml` (hourly during market hours)
- `k8s/ic-score-13f-holdings-cronjob.yaml` (quarterly)

---

### Phase 4: Analyst Ratings & News Sentiment (Priority: MEDIUM)
**Timeline**: Weeks 5-6
**Cost**: FREE to start (Polygon.io News API included)
**IC Score Coverage**: +20% (2 final factors)

#### Task 4.1: Analyst Ratings Ingestion
**File**: `ic-score-service/pipelines/analyst_ratings_ingestion.py`

**Requirements**:
- **Option A** (Recommended): Use Benzinga API free tier (500 req/day)
  - Endpoint: `https://api.benzinga.com/api/v2/calendar/ratings`
  - Free tier: 500 requests/day
- **Option B**: Scrape from financial sites (legal/ethical considerations)
- Extract: analyst_firm, analyst_name, rating, rating_numeric, price_target
- Map ratings to numeric scale: Strong Buy=5, Buy=4, Hold=3, Sell=2, Strong Sell=1
- Insert into `analyst_ratings` table
- Prioritize S&P 500 stocks first

**CLI Interface**:
```bash
python pipelines/analyst_ratings_ingestion.py --limit 500  # Free tier daily limit
python pipelines/analyst_ratings_ingestion.py --sp500     # S&P 500 only
```

#### Task 4.2: News Sentiment Ingestion
**File**: `ic-score-service/pipelines/news_sentiment_ingestion.py`

**Requirements**:
- Fetch news from Polygon.io News API (already has sentiment scores)
- Endpoint: `/v2/reference/news?ticker={symbol}`
- Parse: title, summary, source, published_at, sentiment_score
- If sentiment not provided, use FinBERT model:
  - Model: `ProsusAI/finbert` from HuggingFace
  - Classify: positive, negative, neutral
  - Score: -1.0 to +1.0
- Calculate 7-day and 30-day average sentiment
- Insert into `news_articles` table

**CLI Interface**:
```bash
python pipelines/news_sentiment_ingestion.py --hours 24
python pipelines/news_sentiment_ingestion.py --ticker AAPL
python pipelines/news_sentiment_ingestion.py --backfill 30  # Last 30 days
```

#### Task 4.3: Kubernetes CronJobs
**Files**:
- `k8s/ic-score-analyst-ratings-cronjob.yaml` (daily)
- `k8s/ic-score-news-sentiment-cronjob.yaml` (every 4 hours)

---

### Phase 5: IC Score Calculation Engine (Priority: CRITICAL)
**Timeline**: Week 7
**IC Score Coverage**: 100% (all 10 factors)

#### Task 5.1: IC Score Calculator
**File**: `ic-score-service/pipelines/ic_score_calculator.py`

**Requirements**:
Implement the IC Score calculation algorithm from `/Users/esun/code/investorcenter.ai/docs/competitor-features/tech-spec-02-implementation-details.md` (lines 428-766).

**Factor Weights**:
```python
WEIGHTS = {
    'value': 0.12,           # P/E, P/B, P/S vs sector median
    'growth': 0.15,          # Revenue, EPS, FCF 5-year CAGR
    'profitability': 0.12,   # Margins, ROE, ROA
    'financial_health': 0.10, # D/E, current ratio, interest coverage
    'momentum': 0.08,        # 1M, 3M, 6M, 12M price returns
    'analyst_consensus': 0.10, # Average rating, price target vs price
    'insider_activity': 0.08, # Net insider buying (30d, 90d)
    'institutional': 0.10,   # % ownership, QoQ change
    'news_sentiment': 0.07,  # 7d, 30d average sentiment
    'technical': 0.08        # RSI, MACD, trend strength
}
```

**Calculation Steps**:
1. For each stock, fetch data from all tables
2. Calculate each factor score (0-100) using sector-relative percentiles
3. Handle missing data:
   - If factor has no data, exclude from weighted average
   - Track data_completeness percentage
   - Set confidence_level: High (>90%), Medium (70-90%), Low (<70%)
4. Calculate weighted overall score (1-100)
5. Assign rating:
   - 80-100: Strong Buy
   - 65-79: Buy
   - 50-64: Hold
   - 35-49: Underperform
   - 1-34: Sell
6. Calculate sector_percentile (rank within GICS sector)
7. Insert into `ic_scores` table with all factor breakdowns

**CLI Interface**:
```bash
python pipelines/ic_score_calculator.py --ticker AAPL  # Single stock
python pipelines/ic_score_calculator.py --sector Technology  # All tech stocks
python pipelines/ic_score_calculator.py --all        # All 4,600 stocks
python pipelines/ic_score_calculator.py --sp500      # S&P 500 only
```

#### Task 5.2: FastAPI Service
**File**: `ic-score-service/api/main.py`

**Requirements**:
Create REST API to serve IC Scores to frontend.

**Endpoints**:
```python
@app.get("/api/scores/{ticker}")
def get_ic_score(ticker: str):
    # Return latest IC Score with factor breakdown
    # Include: overall_score, rating, 10 factor scores,
    #          sector_percentile, confidence_level, data_completeness

@app.get("/api/scores/{ticker}/history")
def get_score_history(ticker: str, days: int = 90):
    # Return historical scores for charting

@app.get("/api/scores/top")
def get_top_scores(
    sector: Optional[str] = None,
    min_score: Optional[float] = None,
    limit: int = 50
):
    # Return highest-rated stocks
    # Support filtering by sector, minimum score
    # Order by overall_score DESC

@app.get("/api/scores/screener")
def screener(
    min_value: Optional[float] = None,
    min_growth: Optional[float] = None,
    min_profitability: Optional[float] = None,
    # ... other factor filters
    limit: int = 100
):
    # Advanced screener with factor-level filtering

@app.get("/health")
def health_check():
    # Database health check
```

**Run Locally**:
```bash
cd ic-score-service
uvicorn api.main:app --reload --port 8001
```

#### Task 5.3: Kubernetes Deployments
**Files**:
- `k8s/ic-score-calculator-cronjob.yaml` (daily at 7pm ET)
- `k8s/ic-score-api-deployment.yaml` (3 replicas for HA)
- `k8s/ic-score-api-service.yaml` (ClusterIP service)

**API Deployment Requirements**:
- 3 replicas for high availability
- Health check endpoint at `/health`
- Resource limits: 1 CPU, 2Gi memory per pod
- Port 8000
- Expose via existing ingress at `/api/scores/*`

---

## Implementation Guidelines

### Code Quality
- Use type hints throughout (Python 3.11+)
- Add comprehensive docstrings
- Handle errors gracefully with logging
- Use async/await for database operations
- Add progress bars for long-running operations (use `tqdm`)
- Write unit tests for critical functions

### Database Best Practices
- Use transactions for batch inserts
- Add database constraints validation
- Handle duplicate key conflicts (INSERT ... ON CONFLICT)
- Use connection pooling from `database/database.py`
- Log database errors with context

### Logging
```python
import logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)
```

### Error Handling
```python
try:
    # API call
except requests.exceptions.RequestException as e:
    logger.error(f"API error: {e}")
    # Retry logic with exponential backoff
except Exception as e:
    logger.exception(f"Unexpected error: {e}")
    # Alert/metric
```

### CLI Arguments (use argparse)
```python
import argparse
parser = argparse.ArgumentParser(description='Ingest SEC financials')
parser.add_argument('--limit', type=int, help='Limit number of stocks')
parser.add_argument('--ticker', type=str, help='Single ticker symbol')
parser.add_argument('--all', action='store_true', help='Process all stocks')
args = parser.parse_args()
```

### Dependencies to Add to requirements.txt
```txt
# API clients
requests>=2.31.0
aiohttp>=3.9.0

# Technical analysis
ta-lib>=0.4.28  # May need system-level install
pandas>=2.1.0
pandas-ta>=0.3.14b0
numpy>=1.26.0

# NLP/Sentiment (Phase 4)
transformers>=4.35.0
torch>=2.1.0

# FastAPI (Phase 5)
fastapi>=0.104.0
uvicorn[standard]>=0.24.0
pydantic>=2.5.0

# Utilities
python-dotenv>=1.0.0
tqdm>=4.66.0
```

---

## Testing Strategy

### Unit Tests
Create `ic-score-service/tests/` directory:
```python
# tests/test_sec_client.py
def test_fetch_financials_aapl():
    # Test SEC EDGAR API for AAPL

# tests/test_technical_indicators.py
def test_rsi_calculation():
    # Test RSI calculation with known data

# tests/test_ic_score_calculator.py
def test_calculate_value_score():
    # Test factor score calculation
```

Run tests:
```bash
pytest tests/ -v
```

### Integration Tests
Test full pipeline on 10 stocks:
```bash
# Phase 1
python pipelines/sec_financials_ingestion.py --limit 10

# Verify data
psql -d investorcenter_db -c "SELECT ticker, COUNT(*) FROM financials GROUP BY ticker;"

# Phase 2
python pipelines/technical_indicators_calculator.py --limit 10

# Phase 5
python pipelines/ic_score_calculator.py --limit 10

# Verify IC Scores
psql -d investorcenter_db -c "SELECT ticker, overall_score, rating FROM ic_scores ORDER BY overall_score DESC LIMIT 10;"
```

---

## Deployment Instructions

### Local Development
```bash
# 1. Setup Python environment
cd ic-score-service
python -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# 2. Setup environment variables
cp .env.example .env
# Edit .env with database credentials

# 3. Test database connection
python database/database.py

# 4. Run Phase 1 on 10 stocks
python pipelines/sec_financials_ingestion.py --limit 10

# 5. Run Phase 2 on same 10 stocks
python pipelines/technical_indicators_calculator.py --limit 10

# 6. Calculate IC Scores
python pipelines/ic_score_calculator.py --limit 10

# 7. Start API server
uvicorn api.main:app --reload --port 8001

# 8. Test API
curl http://localhost:8001/api/scores/AAPL
```

### Production Deployment (Kubernetes)
```bash
# 1. Build Docker image
docker build -t ic-score-service:v1.0 .

# 2. Push to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 360358043271.dkr.ecr.us-east-1.amazonaws.com
docker tag ic-score-service:v1.0 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/ic-score-service:v1.0
docker push 360358043271.dkr.ecr.us-east-1.amazonaws.com/investorcenter/ic-score-service:v1.0

# 3. Apply Kubernetes manifests
kubectl apply -f k8s/ic-score-sec-financials-cronjob.yaml
kubectl apply -f k8s/ic-score-technical-indicators-cronjob.yaml
kubectl apply -f k8s/ic-score-calculator-cronjob.yaml
kubectl apply -f k8s/ic-score-api-deployment.yaml
kubectl apply -f k8s/ic-score-api-service.yaml

# 4. Update ingress to route /api/scores/* to ic-score-api-service

# 5. Monitor logs
kubectl logs -f deployment/ic-score-api -n investorcenter
kubectl logs -f job/ic-score-sec-financials -n investorcenter
```

---

## Success Criteria

### Phase 1 Complete
- [ ] 4,600 stocks in `financials` table
- [ ] 90%+ S&P 500 data completeness
- [ ] 70%+ all stocks data completeness
- [ ] Can calculate value, growth, profitability, financial health scores

### Phase 2 Complete
- [ ] 4,600 stocks in `technical_indicators` table
- [ ] 252 days of price history in `stock_prices` table
- [ ] Can calculate momentum and technical scores

### Phase 3 Complete
- [ ] 30 days of insider trades in `insider_trades` table
- [ ] 4 quarters of institutional holdings in `institutional_holdings` table
- [ ] Can calculate insider activity and institutional scores

### Phase 4 Complete
- [ ] 500+ stocks in `analyst_ratings` table (S&P 500 minimum)
- [ ] 30 days of news in `news_articles` table
- [ ] Can calculate analyst consensus and news sentiment scores

### Phase 5 Complete
- [ ] 4,600 stocks with daily IC Scores in `ic_scores` table
- [ ] All 10 factors calculable
- [ ] API responds in <100ms
- [ ] Frontend can display IC Scores on ticker pages

---

## Files to Reference

### Existing Files (Read These)
- `/Users/esun/code/investorcenter.ai/ic-score-service/models.py` - Database models
- `/Users/esun/code/investorcenter.ai/ic-score-service/database/database.py` - DB connection
- `/Users/esun/code/investorcenter.ai/ic-score-service/database/schema.sql` - Database schema
- `/Users/esun/code/investorcenter.ai/docs/competitor-features/tech-spec-02-implementation-details.md` - IC Score algorithm (lines 428-766)
- `/Users/esun/code/investorcenter.ai/backend/services/polygon.go` - Polygon.io integration patterns
- `/Users/esun/code/investorcenter.ai/k8s/polygon-ticker-updater-cronjob.yaml` - CronJob pattern

### New Files to Create
All files listed in the phase tasks above.

---

## Important Notes

1. **Database**: Use the SAME database (`investorcenter_db`) but IC Score tables are in a different schema or just coexist with existing tables
2. **Watchlists**: DO NOT use the new watchlist tables - they duplicate existing functionality. The IC Score system should query existing `watch_lists` table from main database
3. **API Key**: POLYGON_API_KEY already exists in environment - reuse it
4. **Rate Limits**: Respect all API rate limits to avoid getting blocked
5. **Error Handling**: Log all errors, don't crash the entire process for one stock
6. **Data Quality**: Track data_completeness and confidence_level - some stocks will have missing data
7. **Sector Classification**: Use GICS sectors from `companies.sector` column for relative scoring

---

## Questions to Resolve Before Starting

1. **Database connection**: Confirm IC Score tables are in same `investorcenter_db` database? (Assumed YES)
2. **Docker base image**: Use Python 3.11 Alpine or standard?
3. **Kubernetes secrets**: How to access DB_PASSWORD and POLYGON_API_KEY? (Assumed from existing secrets)
4. **Logging**: Where to send logs? CloudWatch, Datadog, or stdout?
5. **Metrics**: Do you want Prometheus metrics for pipeline monitoring?

---

## Estimated Timeline

- **Phase 1**: 1.5 weeks (SEC client + ingestion + testing)
- **Phase 2**: 3 days (technical indicators + testing)
- **Phase 3**: 1 week (insider + institutional + testing)
- **Phase 4**: 1 week (analyst + news + testing)
- **Phase 5**: 1 week (calculator + API + testing)
- **Total**: ~6 weeks with testing

---

## Get Started

**Step 1**: Implement Phase 1 (SEC Financials)
```bash
# Create the SEC client
touch ic-score-service/pipelines/utils/sec_client.py

# Create the ingestion script
touch ic-score-service/pipelines/sec_financials_ingestion.py

# Test on 10 stocks
python ic-score-service/pipelines/sec_financials_ingestion.py --limit 10
```

**Step 2**: Verify data in database
```bash
psql -d investorcenter_db -c "SELECT ticker, period_end_date, revenue, net_income FROM financials ORDER BY ticker, period_end_date DESC LIMIT 20;"
```

**Step 3**: Continue to Phase 2...

---

Good luck! This is a comprehensive project but each phase builds on the previous one. Focus on getting Phase 1 working perfectly before moving to Phase 2.
