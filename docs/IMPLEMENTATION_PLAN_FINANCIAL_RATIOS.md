# Implementation Plan: Financial Ratios Parity with YCharts

**Document Version:** 1.0
**Created:** January 26, 2026
**Data Source:** Financial Modeling Prep (FMP) API

---

## Executive Summary

This document outlines a phased implementation plan to achieve YCharts-level data coverage for financial ratios using the Financial Modeling Prep (FMP) API. The plan adds **~50 new metrics** across 6 phases, prioritized by user value and implementation complexity.

### Current State vs Target State

| Category | Current Metrics | Target Metrics | Gap |
|----------|-----------------|----------------|-----|
| Valuation Ratios | 6 | 15 | +9 |
| Profitability Ratios | 5 | 12 | +7 |
| Growth Metrics | 3 | 12 | +9 |
| Liquidity/Solvency | 4 | 12 | +8 |
| Efficiency Ratios | 0 | 8 | +8 |
| Per Share Metrics | 2 | 10 | +8 |
| Risk/Quality Scores | 0 | 6 | +6 |
| **Total** | **20** | **~75** | **+55** |

---

## FMP API Endpoints to Leverage

### Currently Used
| Endpoint | Metrics | Status |
|----------|---------|--------|
| `/stable/ratios-ttm` | 17 TTM ratios | ✅ In use |

### To Add
| Endpoint | Metrics | Priority |
|----------|---------|----------|
| `/stable/key-metrics-ttm` | ~30 key metrics | 🔴 High |
| `/stable/ratios` | Historical ratios (quarterly/annual) | 🟡 Medium |
| `/stable/key-metrics` | Historical key metrics | 🟡 Medium |
| `/stable/financial-growth` | Growth rates | 🟡 Medium |
| `/stable/analyst-estimates` | Forward estimates | 🟡 Medium |
| `/stable/dividends` | Dividend history | 🔴 High |
| `/stable/score` | Altman Z, Piotroski F scores | 🟢 Low |

---

## Phase 1: Expand Ratios-TTM Endpoint (Week 1)

**Goal:** Capture ALL fields from the existing FMP ratios-ttm endpoint (currently only using 17 of 60+ available fields).

### 1.1 New Fields to Add from Existing Endpoint

The FMP `/stable/ratios-ttm` endpoint provides **60+ metrics**. We're only using 17. Add these:

#### Profitability (Add 4)
| Field Name | JSON Key | Description | YCharts Parity |
|------------|----------|-------------|----------------|
| EBITDA Margin | `ebitdaMarginTTM` | EBITDA / Revenue | ✅ |
| EBIT Margin | `ebitMarginTTM` | EBIT / Revenue | ✅ |
| FCF Margin | `freeCashFlowMarginTTM` | FCF / Revenue | ✅ |
| Pretax Margin | `pretaxProfitMarginTTM` | Pretax Income / Revenue | ✅ |

#### Returns (Add 2)
| Field Name | JSON Key | Description | YCharts Parity |
|------------|----------|-------------|----------------|
| ROIC | `returnOnInvestedCapitalTTM` | Return on Invested Capital | ✅ |
| ROCE | `returnOnCapitalEmployedTTM` | Return on Capital Employed | ✅ |

#### Valuation (Add 6)
| Field Name | JSON Key | Description | YCharts Parity |
|------------|----------|-------------|----------------|
| P/E to Growth (PEG) | `pegRatioTTM` | P/E / EPS Growth | ✅ |
| P/Operating CF | `priceToOperatingCashFlowRatioTTM` | Price / Operating CF per share | ✅ |
| EV/EBIT | `evToEbitTTM` | EV / EBIT | ✅ |
| EV/FCF | `evToFreeCashFlowTTM` | EV / Free Cash Flow | ✅ |
| Earnings Yield | `earningsYieldTTM` | EPS / Price (inverse of P/E) | ✅ |
| FCF Yield | `freeCashFlowYieldTTM` | FCF per share / Price | ✅ |

#### Efficiency (Add 8)
| Field Name | JSON Key | Description | YCharts Parity |
|------------|----------|-------------|----------------|
| Asset Turnover | `assetTurnoverTTM` | Revenue / Total Assets | ✅ |
| Inventory Turnover | `inventoryTurnoverTTM` | COGS / Avg Inventory | ✅ |
| Receivables Turnover | `receivablesTurnoverTTM` | Revenue / Avg Receivables | ✅ |
| Payables Turnover | `payablesTurnoverTTM` | COGS / Avg Payables | ✅ |
| Fixed Asset Turnover | `fixedAssetTurnoverTTM` | Revenue / Fixed Assets | ✅ |
| Days Sales Outstanding | `daysOfSalesOutstandingTTM` | 365 / Receivables Turnover | ✅ |
| Days Inventory Outstanding | `daysOfInventoryOutstandingTTM` | 365 / Inventory Turnover | ✅ |
| Days Payables Outstanding | `daysOfPayablesOutstandingTTM` | 365 / Payables Turnover | ✅ |

#### Leverage (Add 4)
| Field Name | JSON Key | Description | YCharts Parity |
|------------|----------|-------------|----------------|
| Debt/EBITDA | `debtToEbitdaTTM` | Total Debt / EBITDA | ✅ |
| Debt/Capital | `debtToCapitalTTM` | Total Debt / (Debt + Equity) | ✅ |
| Cash Ratio | `cashRatioTTM` | Cash / Current Liabilities | ✅ |
| Cash Conversion Cycle | `cashConversionCycleTTM` | DSO + DIO - DPO | ✅ |

#### Dividend (Add 2)
| Field Name | JSON Key | Description | YCharts Parity |
|------------|----------|-------------|----------------|
| Payout Ratio | `payoutRatioTTM` | Dividends / Net Income | ✅ |
| Dividend Per Share | `dividendPerShareTTM` | Annual DPS | ✅ |

### 1.2 Implementation Tasks

```
□ Task 1.1: Update FMPRatiosTTM struct in fmp_client.go
  - Add all 26 new fields with proper JSON tags
  - Estimated: 30 minutes

□ Task 1.2: Update MergedFinancialMetrics struct
  - Add new fields to merged response
  - Add source tracking for new fields
  - Estimated: 45 minutes

□ Task 1.3: Update MergeWithDBData function
  - Handle new fields in merge logic
  - Add fallback to database where applicable
  - Estimated: 1 hour

□ Task 1.4: Update API response handler
  - Expose new fields in /api/v1/stocks/:ticker/financials
  - Estimated: 30 minutes

□ Task 1.5: Update frontend types and display
  - Add TypeScript types for new fields
  - Add display components for new metrics
  - Estimated: 2 hours

□ Task 1.6: Add unit tests
  - Test FMP response parsing
  - Test merge logic with new fields
  - Estimated: 1 hour
```

### 1.3 Code Changes Required

#### Update `backend/services/fmp_client.go`:

```go
// FMPRatiosTTM - EXPANDED to capture all available fields
type FMPRatiosTTM struct {
    Symbol string `json:"symbol"`

    // === PROFITABILITY MARGINS ===
    GrossProfitMarginTTM     *float64 `json:"grossProfitMarginTTM"`
    OperatingProfitMarginTTM *float64 `json:"operatingProfitMarginTTM"`
    NetProfitMarginTTM       *float64 `json:"netProfitMarginTTM"`
    EBITDAMarginTTM          *float64 `json:"ebitdaMarginTTM"`          // NEW
    EBITMarginTTM            *float64 `json:"ebitMarginTTM"`            // NEW
    FCFMarginTTM             *float64 `json:"freeCashFlowMarginTTM"`    // NEW
    PretaxMarginTTM          *float64 `json:"pretaxProfitMarginTTM"`    // NEW

    // === RETURNS ===
    ReturnOnEquityTTM           *float64 `json:"returnOnEquityTTM"`
    ReturnOnAssetsTTM           *float64 `json:"returnOnAssetsTTM"`
    ReturnOnInvestedCapitalTTM  *float64 `json:"returnOnInvestedCapitalTTM"`  // NEW
    ReturnOnCapitalEmployedTTM  *float64 `json:"returnOnCapitalEmployedTTM"`  // NEW

    // === LIQUIDITY ===
    CurrentRatioTTM *float64 `json:"currentRatioTTM"`
    QuickRatioTTM   *float64 `json:"quickRatioTTM"`
    CashRatioTTM    *float64 `json:"cashRatioTTM"`  // NEW

    // === LEVERAGE ===
    DebtEquityRatioTTM   *float64 `json:"debtEquityRatioTTM"`
    DebtToAssetsRatioTTM *float64 `json:"debtToAssetsRatioTTM"`
    DebtToEBITDATTM      *float64 `json:"debtToEbitdaTTM"`       // NEW
    DebtToCapitalTTM     *float64 `json:"debtToCapitalTTM"`      // NEW
    InterestCoverageTTM  *float64 `json:"interestCoverageTTM"`   // NEW

    // === VALUATION ===
    PriceToEarningsRatioTTM    *float64 `json:"priceToEarningsRatioTTM"`
    PriceToBookRatioTTM        *float64 `json:"priceToBookRatioTTM"`
    PriceToSalesRatioTTM       *float64 `json:"priceToSalesRatioTTM"`
    PriceToFreeCashFlowTTM     *float64 `json:"priceToFreeCashFlowRatioTTM"`
    PriceToOperatingCFTTM      *float64 `json:"priceToOperatingCashFlowRatioTTM"` // NEW
    PEGRatioTTM                *float64 `json:"pegRatioTTM"`                      // NEW
    EarningsYieldTTM           *float64 `json:"earningsYieldTTM"`                 // NEW
    FCFYieldTTM                *float64 `json:"freeCashFlowYieldTTM"`             // NEW

    // === ENTERPRISE VALUE ===
    EnterpriseValueTTM *float64 `json:"enterpriseValueTTM"`
    EVToSalesTTM       *float64 `json:"evToSalesTTM"`
    EVToEBITDATTM      *float64 `json:"evToEbitdaTTM"`
    EVToEBITTTM        *float64 `json:"evToEbitTTM"`          // NEW
    EVToFCFTTM         *float64 `json:"evToFreeCashFlowTTM"`  // NEW

    // === EFFICIENCY ===
    AssetTurnoverTTM            *float64 `json:"assetTurnoverTTM"`              // NEW
    InventoryTurnoverTTM        *float64 `json:"inventoryTurnoverTTM"`          // NEW
    ReceivablesTurnoverTTM      *float64 `json:"receivablesTurnoverTTM"`        // NEW
    PayablesTurnoverTTM         *float64 `json:"payablesTurnoverTTM"`           // NEW
    FixedAssetTurnoverTTM       *float64 `json:"fixedAssetTurnoverTTM"`         // NEW
    DaysOfSalesOutstandingTTM   *float64 `json:"daysOfSalesOutstandingTTM"`     // NEW
    DaysOfInventoryOutstandingTTM *float64 `json:"daysOfInventoryOutstandingTTM"` // NEW
    DaysOfPayablesOutstandingTTM  *float64 `json:"daysOfPayablesOutstandingTTM"`  // NEW
    CashConversionCycleTTM      *float64 `json:"cashConversionCycleTTM"`        // NEW

    // === DIVIDENDS ===
    DividendYieldTTM     *float64 `json:"dividendYieldTTM"`
    PayoutRatioTTM       *float64 `json:"payoutRatioTTM"`       // NEW
    DividendPerShareTTM  *float64 `json:"dividendPerShareTTM"`  // NEW
}
```

---

## Phase 2: Add Key Metrics TTM Endpoint (Week 2)

**Goal:** Add the FMP `/stable/key-metrics-ttm` endpoint for per-share metrics and additional KPIs.

### 2.1 New Endpoint: Key Metrics TTM

**Endpoint:** `GET https://financialmodelingprep.com/stable/key-metrics-ttm?symbol={ticker}&apikey={key}`

### 2.2 Metrics to Add

#### Per Share Metrics (Add 8)
| Field Name | JSON Key | Description | YCharts Parity |
|------------|----------|-------------|----------------|
| Revenue Per Share | `revenuePerShareTTM` | Revenue / Shares Outstanding | ✅ |
| Net Income Per Share | `netIncomePerShareTTM` | Net Income / Shares Outstanding | ✅ |
| Operating CF Per Share | `operatingCashFlowPerShareTTM` | OCF / Shares Outstanding | ✅ |
| FCF Per Share | `freeCashFlowPerShareTTM` | FCF / Shares Outstanding | ✅ |
| Cash Per Share | `cashPerShareTTM` | Cash / Shares Outstanding | ✅ |
| Book Value Per Share | `bookValuePerShareTTM` | Book Value / Shares Outstanding | ✅ |
| Tangible Book Per Share | `tangibleBookValuePerShareTTM` | Tangible BV / Shares | ✅ |
| Shareholders Equity Per Share | `shareholdersEquityPerShareTTM` | Equity / Shares | ✅ |

#### Market Metrics (Add 4)
| Field Name | JSON Key | Description | YCharts Parity |
|------------|----------|-------------|----------------|
| Market Cap | `marketCapTTM` | Current market cap | ✅ |
| Enterprise Value | `enterpriseValueTTM` | EV calculation | ✅ |
| Net Debt | `netDebtTTM` | Total Debt - Cash | ✅ |
| Working Capital | `workingCapitalTTM` | Current Assets - Current Liab | ✅ |

#### Additional Returns (Add 2)
| Field Name | JSON Key | Description | YCharts Parity |
|------------|----------|-------------|----------------|
| ROE (5Y Avg) | `roeTTM` | Can compare to historical | ✅ |
| Graham Number | `grahamNumberTTM` | √(22.5 × EPS × BVPS) | ✅ |

### 2.3 Implementation Tasks

```
□ Task 2.1: Create FMPKeyMetricsTTM struct
  - Define all key metrics fields
  - Estimated: 30 minutes

□ Task 2.2: Add GetKeyMetricsTTM function to FMPClient
  - Similar pattern to GetRatiosTTM
  - Estimated: 30 minutes

□ Task 2.3: Create combined FMP data fetch
  - Fetch both ratios-ttm and key-metrics-ttm in parallel
  - Merge results into unified response
  - Estimated: 1 hour

□ Task 2.4: Update database schema
  - Add new columns to fundamental_metrics_extended
  - Create migration file
  - Estimated: 45 minutes

□ Task 2.5: Update frontend
  - Add per-share metrics display
  - Estimated: 2 hours
```

---

## Phase 3: Add Growth Metrics (Week 3)

**Goal:** Add historical growth rate calculations from FMP `/stable/financial-growth` endpoint.

### 3.1 New Endpoint: Financial Growth

**Endpoint:** `GET https://financialmodelingprep.com/stable/financial-growth?symbol={ticker}&period=annual&limit=5&apikey={key}`

### 3.2 Metrics to Add

| Field Name | JSON Key | Description | YCharts Parity |
|------------|----------|-------------|----------------|
| Revenue Growth (YoY) | `revenueGrowth` | Year-over-year revenue growth | ✅ |
| Revenue Growth (3Y CAGR) | Calculated | 3-year compound growth | ✅ |
| Revenue Growth (5Y CAGR) | Calculated | 5-year compound growth | ✅ |
| Gross Profit Growth | `grossProfitGrowth` | YoY gross profit growth | ✅ |
| Operating Income Growth | `operatingIncomeGrowth` | YoY operating income growth | ✅ |
| Net Income Growth | `netIncomeGrowth` | YoY net income growth | ✅ |
| EPS Growth (YoY) | `epsgrowth` | YoY EPS growth | ✅ |
| EPS Growth (5Y CAGR) | Calculated | 5-year EPS compound growth | ✅ |
| FCF Growth | `freeCashFlowGrowth` | YoY FCF growth | ✅ |
| Book Value Growth | `bookValuePerShareGrowth` | YoY BVPS growth | ✅ |
| Dividend Growth | `dividendsPerShareGrowth` | YoY DPS growth | ✅ |
| Asset Growth | `assetGrowth` | YoY asset growth | ✅ |

### 3.3 CAGR Calculation Logic

```go
// Calculate Compound Annual Growth Rate
func calculateCAGR(startValue, endValue float64, years int) *float64 {
    if startValue <= 0 || years <= 0 {
        return nil
    }
    cagr := math.Pow(endValue/startValue, 1.0/float64(years)) - 1
    result := cagr * 100 // Convert to percentage
    return &result
}
```

### 3.4 Implementation Tasks

```
□ Task 3.1: Create FMPFinancialGrowth struct
  - Define growth rate fields
  - Estimated: 30 minutes

□ Task 3.2: Add GetFinancialGrowth function
  - Fetch 5 years of annual data
  - Calculate CAGRs
  - Estimated: 1 hour

□ Task 3.3: Create growth metrics caching
  - Cache growth metrics (changes less frequently)
  - Refresh weekly or on SEC filing
  - Estimated: 1.5 hours

□ Task 3.4: Update fundamental_metrics_extended table
  - Add growth columns if not present
  - Estimated: 30 minutes

□ Task 3.5: Update frontend growth display
  - Add growth metrics section
  - Add trend indicators
  - Estimated: 2 hours
```

---

## Phase 4: Add Forward Estimates (Week 4)

**Goal:** Add analyst forward estimates from FMP `/stable/analyst-estimates` endpoint.

### 4.1 New Endpoint: Analyst Estimates

**Endpoint:** `GET https://financialmodelingprep.com/stable/analyst-estimates?symbol={ticker}&limit=4&apikey={key}`

### 4.2 Metrics to Add

| Field Name | JSON Key | Description | YCharts Parity |
|------------|----------|-------------|----------------|
| Forward Revenue | `estimatedRevenueAvg` | Consensus revenue estimate | ✅ |
| Forward EPS | `estimatedEpsAvg` | Consensus EPS estimate | ✅ |
| Forward EBITDA | `estimatedEbitdaAvg` | Consensus EBITDA estimate | ✅ |
| Forward Net Income | `estimatedNetIncomeAvg` | Consensus net income | ✅ |
| Forward P/E | Calculated | Price / Forward EPS | ✅ |
| EPS Estimate High | `estimatedEpsHigh` | Highest EPS estimate | ✅ |
| EPS Estimate Low | `estimatedEpsLow` | Lowest EPS estimate | ✅ |
| Number of Analysts | `numberAnalystsEstimatedEps` | Analyst coverage count | ✅ |

### 4.3 Forward P/E Calculation

```go
func calculateForwardPE(currentPrice float64, forwardEPS *float64) *float64 {
    if forwardEPS == nil || *forwardEPS <= 0 {
        return nil
    }
    forwardPE := currentPrice / *forwardEPS
    return &forwardPE
}
```

### 4.4 Implementation Tasks

```
□ Task 4.1: Create FMPAnalystEstimates struct
  - Define estimate fields
  - Estimated: 30 minutes

□ Task 4.2: Add GetAnalystEstimates function
  - Fetch forward estimates
  - Estimated: 30 minutes

□ Task 4.3: Create estimates caching strategy
  - Cache estimates (changes quarterly)
  - Refresh on earnings or estimate revision
  - Estimated: 1 hour

□ Task 4.4: Create new database table for estimates
  - analyst_estimates table
  - Store historical estimates for tracking revisions
  - Estimated: 1 hour

□ Task 4.5: Add Forward P/E to valuation section
  - Calculate and display Forward P/E
  - Compare to TTM P/E
  - Estimated: 1 hour

□ Task 4.6: Update frontend
  - Add estimates section
  - Show estimate ranges
  - Estimated: 2 hours
```

---

## Phase 5: Add Dividend Data (Week 5)

**Goal:** Add comprehensive dividend data from FMP dividend endpoints.

### 5.1 New Endpoints

| Endpoint | Purpose |
|----------|---------|
| `/stable/historical-price-full/stock_dividend/{symbol}` | Dividend history |
| `/stable/dividends` | Current dividend info |

### 5.2 Metrics to Add

| Field Name | Description | YCharts Parity |
|------------|-------------|----------------|
| Dividend Yield (Forward) | Forward annual dividend / Price | ✅ |
| Dividend Per Share (Annual) | Total annual dividend | ✅ |
| Dividend Payout Ratio | Dividends / Net Income | ✅ |
| FCF Payout Ratio | Dividends / FCF | ✅ |
| Dividend Growth (5Y CAGR) | 5-year dividend growth rate | ✅ |
| Consecutive Years of Growth | Dividend aristocrat tracking | ✅ |
| Ex-Dividend Date | Next ex-date | ✅ |
| Payment Date | Next payment date | ✅ |
| Dividend Frequency | Quarterly, monthly, etc. | ✅ |

### 5.3 Implementation Tasks

```
□ Task 5.1: Create FMPDividendHistory struct
  - Define dividend history fields
  - Estimated: 30 minutes

□ Task 5.2: Add GetDividendHistory function
  - Fetch dividend history
  - Calculate consecutive growth years
  - Estimated: 1 hour

□ Task 5.3: Calculate dividend metrics
  - 5Y CAGR calculation
  - FCF payout ratio
  - Estimated: 1 hour

□ Task 5.4: Update dividends table
  - Use existing migration 017
  - Add any missing columns
  - Estimated: 30 minutes

□ Task 5.5: Create dividend calendar feature
  - Show upcoming ex-dates
  - Estimated: 2 hours

□ Task 5.6: Update frontend dividend section
  - Display all dividend metrics
  - Add dividend history chart
  - Estimated: 3 hours
```

---

## Phase 6: Add Quality Scores (Week 6)

**Goal:** Add financial health scoring from FMP score endpoint.

### 6.1 New Endpoint: Financial Score

**Endpoint:** `GET https://financialmodelingprep.com/stable/score?symbol={ticker}&apikey={key}`

### 6.2 Metrics to Add

| Field Name | JSON Key | Description | YCharts Parity |
|------------|----------|-------------|----------------|
| Altman Z-Score | `altmanZScore` | Bankruptcy prediction | ✅ |
| Piotroski F-Score | `piotroskiScore` | Financial strength (0-9) | ✅ |
| Working Capital/TA | Used in Z-Score | Liquidity measure | ✅ |
| Retained Earnings/TA | Used in Z-Score | Profitability measure | ✅ |
| EBIT/TA | Used in Z-Score | Operating efficiency | ✅ |
| Market Value/TL | Used in Z-Score | Solvency measure | ✅ |

### 6.3 Score Interpretation

```
Altman Z-Score:
- Z > 2.99: Safe zone (low bankruptcy risk)
- 1.81 < Z < 2.99: Grey zone (moderate risk)
- Z < 1.81: Distress zone (high bankruptcy risk)

Piotroski F-Score:
- 8-9: Strong financial position
- 5-7: Average
- 0-4: Weak financial position
```

### 6.4 Implementation Tasks

```
□ Task 6.1: Create FMPFinancialScore struct
  - Define score fields
  - Estimated: 20 minutes

□ Task 6.2: Add GetFinancialScore function
  - Fetch scores
  - Estimated: 30 minutes

□ Task 6.3: Add score interpretation logic
  - Map scores to risk levels
  - Generate human-readable labels
  - Estimated: 30 minutes

□ Task 6.4: Update database schema
  - Add score columns to fundamental_metrics_extended
  - Estimated: 20 minutes

□ Task 6.5: Add scores to screener filters
  - Allow filtering by Z-Score, F-Score
  - Estimated: 1 hour

□ Task 6.6: Update frontend
  - Display scores with visual indicators
  - Add score explanations
  - Estimated: 2 hours
```

---

## Database Schema Changes

### New Migration: `018_add_extended_financial_metrics.sql`

```sql
-- Migration: Add extended financial metrics columns
-- Description: Support for additional FMP data fields

-- Add efficiency ratios
ALTER TABLE fundamental_metrics_extended
ADD COLUMN IF NOT EXISTS asset_turnover DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS inventory_turnover DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS receivables_turnover DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS payables_turnover DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS fixed_asset_turnover DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS days_sales_outstanding DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS days_inventory_outstanding DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS days_payables_outstanding DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS cash_conversion_cycle DECIMAL(10, 4);

-- Add additional valuation metrics
ALTER TABLE fundamental_metrics_extended
ADD COLUMN IF NOT EXISTS peg_ratio DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS price_to_operating_cf DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS price_to_fcf DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS ev_to_ebit DECIMAL(12, 4),
ADD COLUMN IF NOT EXISTS earnings_yield DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS fcf_yield DECIMAL(10, 4);

-- Add additional margin metrics
ALTER TABLE fundamental_metrics_extended
ADD COLUMN IF NOT EXISTS ebit_margin DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS fcf_margin DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS pretax_margin DECIMAL(10, 4);

-- Add additional leverage metrics
ALTER TABLE fundamental_metrics_extended
ADD COLUMN IF NOT EXISTS cash_ratio DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS debt_to_capital DECIMAL(10, 4);

-- Add quality scores
ALTER TABLE fundamental_metrics_extended
ADD COLUMN IF NOT EXISTS altman_z_score DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS piotroski_f_score INTEGER;

-- Add per share metrics
ALTER TABLE fundamental_metrics_extended
ADD COLUMN IF NOT EXISTS revenue_per_share DECIMAL(14, 4),
ADD COLUMN IF NOT EXISTS operating_cf_per_share DECIMAL(14, 4),
ADD COLUMN IF NOT EXISTS fcf_per_share DECIMAL(14, 4),
ADD COLUMN IF NOT EXISTS cash_per_share DECIMAL(14, 4),
ADD COLUMN IF NOT EXISTS tangible_book_per_share DECIMAL(14, 4);

-- Add forward estimates
ALTER TABLE fundamental_metrics_extended
ADD COLUMN IF NOT EXISTS forward_pe DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS forward_eps DECIMAL(10, 4),
ADD COLUMN IF NOT EXISTS forward_revenue DECIMAL(20, 2),
ADD COLUMN IF NOT EXISTS num_analysts INTEGER;

-- Add return metrics
ALTER TABLE fundamental_metrics_extended
ADD COLUMN IF NOT EXISTS roce DECIMAL(10, 4);

-- Comments
COMMENT ON COLUMN fundamental_metrics_extended.asset_turnover IS 'Revenue / Average Total Assets';
COMMENT ON COLUMN fundamental_metrics_extended.cash_conversion_cycle IS 'DSO + DIO - DPO (days)';
COMMENT ON COLUMN fundamental_metrics_extended.peg_ratio IS 'P/E Ratio / EPS Growth Rate';
COMMENT ON COLUMN fundamental_metrics_extended.altman_z_score IS 'Bankruptcy prediction score (>2.99 safe, <1.81 distress)';
COMMENT ON COLUMN fundamental_metrics_extended.piotroski_f_score IS 'Financial strength score (0-9, higher is better)';
COMMENT ON COLUMN fundamental_metrics_extended.forward_pe IS 'Price / Forward EPS Estimate';
```

### New Table: `analyst_estimates`

```sql
CREATE TABLE IF NOT EXISTS analyst_estimates (
    id BIGSERIAL PRIMARY KEY,
    ticker VARCHAR(10) NOT NULL,
    estimate_date DATE NOT NULL,
    fiscal_year INTEGER NOT NULL,
    fiscal_quarter INTEGER,  -- NULL for annual

    -- Revenue estimates
    estimated_revenue_low DECIMAL(20, 2),
    estimated_revenue_avg DECIMAL(20, 2),
    estimated_revenue_high DECIMAL(20, 2),

    -- EPS estimates
    estimated_eps_low DECIMAL(10, 4),
    estimated_eps_avg DECIMAL(10, 4),
    estimated_eps_high DECIMAL(10, 4),

    -- EBITDA estimates
    estimated_ebitda_low DECIMAL(20, 2),
    estimated_ebitda_avg DECIMAL(20, 2),
    estimated_ebitda_high DECIMAL(20, 2),

    -- Net Income estimates
    estimated_net_income_low DECIMAL(20, 2),
    estimated_net_income_avg DECIMAL(20, 2),
    estimated_net_income_high DECIMAL(20, 2),

    -- Analyst count
    num_analysts_revenue INTEGER,
    num_analysts_eps INTEGER,

    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),

    CONSTRAINT uq_analyst_estimates UNIQUE (ticker, estimate_date, fiscal_year, fiscal_quarter)
);

CREATE INDEX idx_analyst_estimates_ticker ON analyst_estimates(ticker);
CREATE INDEX idx_analyst_estimates_date ON analyst_estimates(estimate_date DESC);
```

---

## API Response Structure Updates

### Updated `/api/v1/stocks/:ticker/financials` Response

```typescript
interface FinancialMetricsResponse {
  data: {
    // === VALUATION (15 metrics) ===
    pe_ratio: number | null;
    forward_pe: number | null;           // NEW
    pb_ratio: number | null;
    ps_ratio: number | null;
    price_to_fcf: number | null;         // NEW
    price_to_operating_cf: number | null; // NEW
    peg_ratio: number | null;            // NEW
    ev_to_sales: number | null;
    ev_to_ebitda: number | null;
    ev_to_ebit: number | null;           // NEW
    ev_to_fcf: number | null;            // NEW
    earnings_yield: number | null;        // NEW
    fcf_yield: number | null;            // NEW
    enterprise_value: number | null;
    market_cap: number | null;

    // === PROFITABILITY (12 metrics) ===
    gross_margin: number | null;
    operating_margin: number | null;
    net_margin: number | null;
    ebitda_margin: number | null;
    ebit_margin: number | null;          // NEW
    fcf_margin: number | null;           // NEW
    pretax_margin: number | null;        // NEW
    roe: number | null;
    roa: number | null;
    roic: number | null;
    roce: number | null;                 // NEW

    // === LIQUIDITY (4 metrics) ===
    current_ratio: number | null;
    quick_ratio: number | null;
    cash_ratio: number | null;           // NEW
    working_capital: number | null;      // NEW

    // === LEVERAGE (6 metrics) ===
    debt_to_equity: number | null;
    debt_to_assets: number | null;
    debt_to_ebitda: number | null;       // NEW
    debt_to_capital: number | null;      // NEW
    interest_coverage: number | null;
    net_debt_to_ebitda: number | null;

    // === EFFICIENCY (9 metrics) ===
    asset_turnover: number | null;       // NEW
    inventory_turnover: number | null;   // NEW
    receivables_turnover: number | null; // NEW
    payables_turnover: number | null;    // NEW
    fixed_asset_turnover: number | null; // NEW
    days_sales_outstanding: number | null; // NEW
    days_inventory_outstanding: number | null; // NEW
    days_payables_outstanding: number | null; // NEW
    cash_conversion_cycle: number | null; // NEW

    // === GROWTH (12 metrics) ===
    revenue_growth_yoy: number | null;
    revenue_growth_3y_cagr: number | null;
    revenue_growth_5y_cagr: number | null;
    gross_profit_growth_yoy: number | null; // NEW
    operating_income_growth_yoy: number | null; // NEW
    net_income_growth_yoy: number | null; // NEW
    eps_growth_yoy: number | null;
    eps_growth_3y_cagr: number | null;
    eps_growth_5y_cagr: number | null;
    fcf_growth_yoy: number | null;
    book_value_growth_yoy: number | null; // NEW
    dividend_growth_5y_cagr: number | null;

    // === PER SHARE (10 metrics) ===
    eps_diluted: number | null;
    book_value_per_share: number | null;
    tangible_book_per_share: number | null; // NEW
    revenue_per_share: number | null;    // NEW
    operating_cf_per_share: number | null; // NEW
    fcf_per_share: number | null;        // NEW
    cash_per_share: number | null;       // NEW
    dividend_per_share: number | null;   // NEW

    // === DIVIDENDS (9 metrics) ===
    dividend_yield: number | null;
    forward_dividend_yield: number | null; // NEW
    payout_ratio: number | null;
    fcf_payout_ratio: number | null;     // NEW
    consecutive_dividend_years: number | null;
    ex_dividend_date: string | null;     // NEW
    payment_date: string | null;         // NEW
    dividend_frequency: string | null;   // NEW

    // === QUALITY SCORES (6 metrics) ===
    altman_z_score: number | null;       // NEW
    altman_z_interpretation: string | null; // NEW
    piotroski_f_score: number | null;    // NEW
    piotroski_f_interpretation: string | null; // NEW
    data_quality_score: number | null;

    // === FORWARD ESTIMATES (6 metrics) ===
    forward_eps: number | null;          // NEW
    forward_eps_high: number | null;     // NEW
    forward_eps_low: number | null;      // NEW
    forward_revenue: number | null;      // NEW
    num_analysts: number | null;         // NEW
  };

  debug?: {
    sources: Record<string, 'fmp' | 'database' | 'calculated'>;
  };

  meta: {
    ticker: string;
    last_updated: string;
    data_source: string;
  };
}
```

---

## Frontend Component Updates

### New Components to Create

```
components/
├── financials/
│   ├── ValuationMetrics.tsx      // P/E, P/B, P/S, EV metrics
│   ├── ProfitabilityMetrics.tsx  // Margins, ROE, ROA, ROIC
│   ├── GrowthMetrics.tsx         // YoY, CAGR growth rates
│   ├── EfficiencyMetrics.tsx     // Turnover, DSO, DIO, CCC
│   ├── LeverageMetrics.tsx       // Debt ratios
│   ├── DividendMetrics.tsx       // Dividend data
│   ├── QualityScores.tsx         // Z-Score, F-Score
│   ├── PerShareMetrics.tsx       // Per share data
│   ├── ForwardEstimates.tsx      // Analyst estimates
│   └── MetricCard.tsx            // Reusable metric display
```

### TypeScript Types to Add

```typescript
// lib/types/financial-metrics.ts

export interface EfficiencyRatios {
  assetTurnover: number | null;
  inventoryTurnover: number | null;
  receivablesTurnover: number | null;
  payablesTurnover: number | null;
  fixedAssetTurnover: number | null;
  daysOfSalesOutstanding: number | null;
  daysOfInventoryOutstanding: number | null;
  daysOfPayablesOutstanding: number | null;
  cashConversionCycle: number | null;
}

export interface GrowthRates {
  revenueGrowthYoY: number | null;
  revenueGrowth3YCAGR: number | null;
  revenueGrowth5YCAGR: number | null;
  epsGrowthYoY: number | null;
  epsGrowth3YCAGR: number | null;
  epsGrowth5YCAGR: number | null;
  fcfGrowthYoY: number | null;
  dividendGrowth5YCAGR: number | null;
}

export interface QualityScores {
  altmanZScore: number | null;
  altmanZInterpretation: 'safe' | 'grey' | 'distress' | null;
  piotroskiFScore: number | null;
  piotroskiFInterpretation: 'strong' | 'average' | 'weak' | null;
}

export interface ForwardEstimates {
  forwardPE: number | null;
  forwardEPS: number | null;
  forwardEPSHigh: number | null;
  forwardEPSLow: number | null;
  forwardRevenue: number | null;
  numAnalysts: number | null;
}
```

---

## Caching Strategy

### Real-time vs Cached Data

| Data Type | Refresh Frequency | Cache TTL | Storage |
|-----------|-------------------|-----------|---------|
| TTM Ratios (FMP) | Per request | None (real-time) | Memory |
| Key Metrics (FMP) | Per request | None (real-time) | Memory |
| Growth Rates | Daily | 24 hours | Database |
| Forward Estimates | Daily | 24 hours | Database |
| Dividend History | Weekly | 7 days | Database |
| Quality Scores | Weekly | 7 days | Database |

### Caching Implementation

```go
// Use existing patterns from fmp_client.go
// Consider adding Redis caching for frequently accessed data

type CachedMetrics struct {
    Data      interface{}
    CachedAt  time.Time
    ExpiresAt time.Time
}

func (c *FMPClient) GetCachedOrFetch(key string, ttl time.Duration, fetcher func() (interface{}, error)) (interface{}, error) {
    // Check cache first
    if cached, ok := c.cache.Get(key); ok {
        return cached, nil
    }

    // Fetch fresh data
    data, err := fetcher()
    if err != nil {
        return nil, err
    }

    // Store in cache
    c.cache.Set(key, data, ttl)
    return data, nil
}
```

---

## Testing Plan

### Unit Tests

```
tests/
├── fmp_client_test.go
│   ├── TestGetRatiosTTM_AllFields
│   ├── TestGetKeyMetricsTTM
│   ├── TestGetFinancialGrowth
│   ├── TestGetAnalystEstimates
│   ├── TestMergeWithDBData_AllFields
│   ├── TestConvertToPercentage
│   └── TestCalculateCAGR
├── financial_metrics_test.go
│   ├── TestForwardPECalculation
│   ├── TestZScoreInterpretation
│   └── TestFScoreInterpretation
```

### Integration Tests

```
□ Test FMP API connectivity with all new endpoints
□ Test data merge logic with various scenarios
□ Test frontend display of all new metrics
□ Test screener filters with new metrics
□ Test export functionality with new fields
```

---

## Rollout Plan

### Week 1: Phase 1 (Expand ratios-ttm)
- Deploy backend changes
- Deploy frontend changes
- Monitor for errors
- Validate data accuracy

### Week 2: Phase 2 (Key metrics)
- Add key-metrics-ttm endpoint
- Deploy per-share metrics
- Monitor API rate limits

### Week 3: Phase 3 (Growth metrics)
- Add financial-growth endpoint
- Deploy growth calculations
- Add growth to screener

### Week 4: Phase 4 (Forward estimates)
- Add analyst-estimates endpoint
- Deploy Forward P/E
- Add estimates display

### Week 5: Phase 5 (Dividends)
- Add dividend endpoints
- Deploy dividend calendar
- Add dividend to screener

### Week 6: Phase 6 (Quality scores)
- Add score endpoint
- Deploy Z-Score, F-Score
- Add scores to screener

---

## Risk Mitigation

### API Rate Limits
- FMP has rate limits based on subscription tier
- Implement request queuing and backoff
- Cache aggressively to reduce API calls

### Data Quality
- Some FMP fields may be null for certain stocks
- Maintain database fallback for all metrics
- Log and monitor null rates

### Breaking Changes
- Version API responses
- Maintain backward compatibility
- Communicate changes to frontend team

---

## Success Metrics

### Data Coverage
- **Target:** 90%+ data coverage for S&P 500 stocks
- **Measurement:** % of stocks with complete data

### User Engagement
- **Target:** 20% increase in time on financials tab
- **Measurement:** Analytics tracking

### Screener Usage
- **Target:** 50% of screener queries use new filters
- **Measurement:** Query logs

---

## Appendix: FMP API Reference Links

- [FMP Developer Docs](https://site.financialmodelingprep.com/developer/docs)
- [TTM Ratios API](https://site.financialmodelingprep.com/developer/docs/stable/metrics-ratios-ttm)
- [Key Metrics API](https://site.financialmodelingprep.com/developer/docs/stable/key-metrics)
- [Financial Growth API](https://site.financialmodelingprep.com/developer/docs/stable/financial-growth)
- [Analyst Estimates API](https://site.financialmodelingprep.com/developer/docs/stable/analyst-estimates)
- [FMP Pricing](https://site.financialmodelingprep.com/pricing-plans)
