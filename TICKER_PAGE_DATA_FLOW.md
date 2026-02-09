# TICKER PAGE DATA FLOW ANALYSIS
## For: https://investorcenter.ai/ticker/HIMS

---

## 📊 COMPLETE DATA ARCHITECTURE

### Frontend Page: `app/ticker/[symbol]/page.tsx`

**Server-Side Data Fetching (on page load):**
```typescript
1. GET /api/v1/tickers/HIMS
   → Fetches comprehensive ticker summary (stock info + price + fundamentals)
   
2. GET /api/v1/tickers/HIMS/chart?period=1Y
   → Fetches historical chart data
```

**Client-Side Components (after page loads):**
```typescript
TickerFundamentals component fetches:
  1. GET /api/v1/tickers/HIMS          → Polygon.io data
  2. GET /api/v1/stocks/HIMS/financials → IC Score service (SEC filings)
  3. GET /api/v1/stocks/HIMS/risk       → IC Score service (risk metrics)
  
Then MERGES all three data sources (prefers IC Score data when available)
```

---

## 🗺️ DATA FLOW DIAGRAM

```
┌─────────────────────────────────────────────────────────────────┐
│                    USER VISITS PAGE                             │
│            https://investorcenter.ai/ticker/HIMS                │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│  SERVER-SIDE RENDER (Next.js)                                   │
│  app/ticker/[symbol]/page.tsx                                   │
├─────────────────────────────────────────────────────────────────┤
│  1. Fetch ticker data (backend)                                 │
│  2. Fetch chart data (backend)                                  │
│  3. Render initial HTML with data                               │
└────────────────────────┬────────────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┐
         ▼               ▼               ▼
    ┌────────┐     ┌─────────┐    ┌──────────┐
    │ HEADER │     │  CHART  │    │ SIDEBAR  │
    │ (Price)│     │  (1Y)   │    │ (Metrics)│
    └────────┘     └─────────┘    └──────────┘
         │               │               │
         │               │               ▼
         │               │    ┌───────────────────────┐
         │               │    │ TickerFundamentals    │
         │               │    │ Component (CLIENT)    │
         │               │    └───────┬───────────────┘
         │               │            │
         │               │    Fetches 3 APIs in parallel:
         │               │    ├─ /api/v1/tickers/HIMS
         │               │    ├─ /api/v1/stocks/HIMS/financials
         │               │    └─ /api/v1/stocks/HIMS/risk
         │               │            │
         │               │            ▼
         │               │    ┌───────────────────┐
         │               │    │ DATA MERGE LOGIC  │
         │               │    │ Prefers IC Score  │
         │               │    │ Falls back to     │
         │               │    │ Polygon.io        │
         │               │    └───────────────────┘
         │               │
         └───────────────┴──── DISPLAYS TO USER
```

---

## 🔄 DATA SOURCES & PRIORITY

### 1. **Price Data** (Real-time)
**Source:** Polygon.io API
**Handler:** `backend/handlers/ticker_comprehensive.go` → `GetTicker()`
**Flow:**
```
Polygon.io API
    ↓
polygonClient.GetQuote(symbol)
    ↓
models.StockPrice {
    Price, Open, High, Low, Close,
    Volume, Change, ChangePercent
}
```

### 2. **Fundamental Metrics** (Quarterly/Annual)
**Source Priority:** IC Score Service (SEC EDGAR) → Polygon.io (fallback)

#### From IC Score Service (PostgreSQL `financials` table):
**Handler:** `backend/handlers/ic_score_handlers.go` → `GetFinancialMetrics()`
**Database:** `financials` table
**Data Origin:** SEC EDGAR filings parsed by Python pipeline
**Metrics:**
- ✅ ROE, ROA (more accurate from SEC)
- ✅ Gross Margin, Operating Margin, Net Margin
- ✅ Debt to Equity, Current Ratio, Quick Ratio
- ✅ P/E, P/B, P/S ratios
- ✅ Shares Outstanding
- ✅ Revenue Growth YoY, Earnings Growth YoY

```sql
SELECT * FROM financials 
WHERE ticker = 'HIMS' 
ORDER BY period_end_date DESC 
LIMIT 1
```

#### From Polygon.io (Fallback):
**Handler:** `backend/services/polygon.go` → `GetFundamentals()`
**API:** Polygon.io Financials API
**Metrics:**
- P/E, P/B, P/S ratios
- Revenue, Net Income, EPS
- Margins (less accurate, may be stale)

### 3. **Risk Metrics** (Calculated)
**Source:** IC Score Service (PostgreSQL `risk_metrics` table)
**Handler:** `backend/handlers/ic_score_handlers.go` → `GetRiskMetrics()`
**Metrics:**
- ✅ Beta (stock volatility vs market)
- Alpha (excess returns)
- Sharpe Ratio
- Volatility (standard deviation)

```sql
SELECT * FROM risk_metrics 
WHERE ticker = 'HIMS' 
AND period = '1Y'
```

### 4. **Market Data** (Real-time & Historical)
**Source:** Polygon.io
**Handler:** `backend/handlers/ticker_comprehensive.go` → `buildKeyMetrics()`
**Metrics:**
- 52-week High/Low
- YTD Change %
- Average Volume
- Market Cap

---

## 📦 DATA MERGE STRATEGY

**In `TickerFundamentals.tsx` (lines 206-235):**

```typescript
// MERGE LOGIC (Prefers IC Score data):
const mappedFundamentals = {
  // 1. Valuation: Prefer Polygon (has current price)
  pe: polygonFundamentals?.pe || icScoreFinancials?.pe_ratio || 'N/A',
  pb: polygonFundamentals?.pb || icScoreFinancials?.pb_ratio || 'N/A',
  ps: polygonFundamentals?.ps || icScoreFinancials?.ps_ratio || 'N/A',
  
  // 2. Profitability: PREFER IC SCORE (more accurate from SEC)
  roe: icScoreFinancials?.roe || polygonFundamentals?.roe || 'N/A',
  roa: icScoreFinancials?.roa || polygonFundamentals?.roa || 'N/A',
  
  // 3. Margins: PREFER IC SCORE (from SEC filings)
  grossMargin: icScoreFinancials?.gross_margin || polygonFundamentals?.grossMargin,
  operatingMargin: icScoreFinancials?.operating_margin || polygonFundamentals?.operatingMargin,
  netMargin: icScoreFinancials?.net_margin || polygonFundamentals?.netMargin,
  
  // 4. Financial Health: PREFER IC SCORE
  debtToEquity: icScoreFinancials?.debt_to_equity || polygonKeyMetrics?.debtToEquity,
  currentRatio: icScoreFinancials?.current_ratio || polygonKeyMetrics?.currentRatio,
  
  // 5. Growth: PREFER IC SCORE
  revenueGrowth1Y: icScoreFinancials?.revenue_growth_yoy || polygonKeyMetrics?.revenueGrowth1Y,
  earningsGrowth1Y: icScoreFinancials?.earnings_growth_yoy || polygonKeyMetrics?.earningsGrowth1Y,
  
  // 6. Market Data: PREFER IC SCORE for shares, Polygon for price-based
  beta: icScoreRisk?.beta || polygonKeyMetrics?.beta,
  sharesOutstanding: icScoreFinancials?.shares_outstanding || polygonKeyMetrics?.sharesOutstanding,
};
```

**Why This Strategy?**
- ✅ IC Score data comes from **SEC filings** (official, accurate)
- ✅ Polygon.io has **real-time prices** (needed for P/E calculation)
- ✅ IC Score may lag by 1 quarter (wait for SEC filing)
- ✅ Polygon.io fills gaps when IC Score data not available

---

## 🗄️ DATABASE TABLES USED

### 1. `tickers` table
- Basic stock info: symbol, name, exchange, sector
- Populated by: Polygon.io ticker import script

### 2. `financials` table (IC Score Service)
**Location:** IC Score PostgreSQL database
**Populated by:** Python pipeline `ic-score-service/pipelines/sec_financials_ingestion.py`
**Update Frequency:** Quarterly (after SEC 10-Q/10-K filings)
**Columns:**
```sql
ticker, period_end_date, fiscal_year, fiscal_quarter,
revenue, eps_diluted, gross_margin, operating_margin, net_margin,
roe, roa, debt_to_equity, current_ratio, quick_ratio,
pe_ratio, pb_ratio, ps_ratio, shares_outstanding
```

### 3. `risk_metrics` table (IC Score Service)
**Location:** IC Score PostgreSQL database
**Populated by:** Python script `ic-score-service/scripts/calculate_risk_metrics.py`
**Update Frequency:** Daily
**Columns:**
```sql
ticker, calculation_date, period,
beta, alpha, sharpe_ratio, volatility,
max_drawdown, correlation_with_spy
```

---

## 🔌 API ENDPOINTS SUMMARY

| Endpoint | Data Source | Update Frequency | Used For |
|----------|-------------|------------------|----------|
| `/api/v1/tickers/:symbol` | Polygon.io | Real-time | Price, basic info |
| `/api/v1/stocks/:symbol/financials` | SEC EDGAR (via IC Score) | Quarterly | ROE, ROA, Margins |
| `/api/v1/stocks/:symbol/risk` | Calculated | Daily | Beta, Sharpe Ratio |
| `/api/v1/tickers/:symbol/chart` | Database + Polygon | Real-time/Daily | Price history |

---

## 🏗️ CURRENT ARCHITECTURE vs FMP INTEGRATION

### Current State:
```
Price Data:         Polygon.io ✅
Fundamentals:       Polygon.io (partial) + SEC EDGAR (via IC Score) ✅
Ratios:             Mix of Polygon + SEC calculations ⚠️
Historical Charts:  Database (3 years) + Polygon ✅
```

### **PROBLEM:** 
- Polygon.io fundamentals are **incomplete/stale** for many tickers
- SEC parser (IC Score) is **not 100% accurate** (user's concern)
- Missing: P/E, P/B, ROE for many tickers (shows "N/A")

### **PROPOSED FMP INTEGRATION:**
```
Price Data:         Polygon.io (keep) ✅
Fundamentals:       FMP API (replace) ✅ NEW
  → Income Statement:  FMP /stable/income-statement
  → Balance Sheet:     FMP /stable/balance-sheet-statement
  → Cash Flow:         FMP /stable/cash-flow-statement
  → Company Profile:   FMP /stable/profile (for current price)
Ratios:             Calculate ourselves ✅ NEW
  → P/E = Price / EPS (from FMP TTM data)
  → ROE = Net Income / Shareholders Equity
  → P/B = Price / Book Value per Share
Historical Charts:  Keep current (Database + Polygon) ✅
```

### **WHY FMP IS BETTER:**
1. ✅ Comprehensive data for ALL tickers (not just large caps)
2. ✅ Covers balance sheet items you need (Total Long Term Assets, etc.)
3. ✅ Free tier has financial statements (Income, Balance, Cash Flow)
4. ✅ We calculate ratios ourselves → **Full transparency for tooltips!**
5. ✅ More reliable than SEC parser (official data from SEC via FMP)

---

## 🎯 RECOMMENDATION: INTEGRATE FMP NOW

**Replace:**
- `backend/services/polygon.go` → `GetFundamentals()` (unreliable)
- `ic-score-service/pipelines/sec_financials_ingestion.py` (inaccurate)

**With:**
- `backend/services/fmp_fundamentals.go` (new file)
  - GetIncomeStatement()
  - GetBalanceSheet()
  - GetCashFlow()
  - CalculateRatios() → with metadata for tooltips!

**Keep:**
- Polygon.io for real-time prices ✅
- Polygon.io for historical charts ✅
- Risk metrics calculation ✅

**Timeline:**
1. Create FMP service (2 hours)
2. Add metadata layer for formulas (3 hours)
3. Update frontend tooltips (2 hours)
4. Test with AAPL, HIMS, GOOG (1 hour)
5. Deploy (1 hour)

**Total: ~1 day of work for 100% accurate fundamentals** 🎉
