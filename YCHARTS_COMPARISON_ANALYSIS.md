# Investor Center vs YCharts: Comprehensive Gap Analysis

**Analysis Date:** January 26, 2026
**Purpose:** Identify feature and data gaps between Investor Center and YCharts

---

## Executive Summary

YCharts is a mature, enterprise-grade financial research platform primarily targeting financial advisors and institutional users. Investor Center is a modern, retail-focused platform with unique social sentiment features. This analysis identifies **67 feature gaps** and **numerous data coverage gaps** that could be addressed to achieve feature parity.

### Key Findings

| Category | Investor Center | YCharts | Gap Severity |
|----------|-----------------|---------|--------------|
| Stock Coverage | ~5,000+ US stocks | 29,000+ global stocks | 🔴 High |
| Fund Coverage | None | 75,000+ mutual funds/ETFs | 🔴 Critical |
| Bond Coverage | None | 6,000,000+ bonds | 🔴 Critical |
| Economic Indicators | None | 500,000+ indicators | 🔴 Critical |
| Financial Metrics | ~50 ratios | 4,500+ metrics | 🔴 High |
| Unique Strengths | Reddit sentiment, IC Score | Client reporting, Excel Add-in | - |

---

## Part 1: Feature-Level Comparison

### 1.1 Asset Class Coverage

| Feature | Investor Center | YCharts | Gap |
|---------|-----------------|---------|-----|
| US Stocks | ✅ Yes | ✅ Yes | - |
| International Stocks | ❌ No | ✅ 29,000+ global | 🔴 Missing |
| ADRs | ❌ No | ✅ Yes | 🔴 Missing |
| Mutual Funds | ❌ No | ✅ 45,000+ | 🔴 Missing |
| ETFs | ❌ Limited | ✅ 30,000+ | 🔴 Missing |
| CEFs (Closed-End Funds) | ❌ No | ✅ Yes | 🔴 Missing |
| UITs | ❌ No | ✅ 12,000+ | 🔴 Missing |
| Individual Bonds | ❌ No | ✅ 6M+ bonds | 🔴 Missing |
| Municipal Bonds | ❌ No | ✅ Yes | 🔴 Missing |
| Corporate Bonds | ❌ No | ✅ Yes | 🔴 Missing |
| Government Bonds | ❌ No | ✅ Yes | 🔴 Missing |
| SMAs (Separately Managed Accounts) | ❌ No | ✅ 10,000+ | 🔴 Missing |
| Indices | ❌ Limited | ✅ 26,000+ | 🔴 Missing |
| Cryptocurrency | ✅ Yes (CoinGecko) | ✅ Limited (25 coins) | ✅ Advantage |

### 1.2 Screening & Research Tools

| Feature | Investor Center | YCharts | Gap |
|---------|-----------------|---------|-----|
| Stock Screener | ✅ Basic (6 filters) | ✅ Advanced (4,500+ metrics) | 🟡 Limited |
| Fund Screener | ❌ No | ✅ Yes | 🔴 Missing |
| Bond Screener | ❌ No | ✅ Yes | 🔴 Missing |
| Sector Filtering | ✅ 11 sectors | ✅ Full GICS | ✅ Similar |
| Custom Scoring Models | ❌ No | ✅ Yes (blend metrics) | 🔴 Missing |
| Preset Screens | ✅ Basic | ✅ Advanced | 🟡 Limited |
| Screen Alerts | ❌ No | ✅ Yes | 🔴 Missing |
| Save/Share Screens | ❌ No | ✅ Yes | 🔴 Missing |

### 1.3 Portfolio Management

| Feature | Investor Center | YCharts | Gap |
|---------|-----------------|---------|-----|
| Watch Lists | ✅ Yes | ✅ Yes | ✅ Similar |
| Model Portfolios | ❌ No | ✅ Yes | 🔴 Missing |
| Benchmark Portfolios | ❌ No | ✅ Custom blends | 🔴 Missing |
| Client Portfolios | ❌ No | ✅ Yes | 🔴 Missing |
| Household Portfolios | ❌ No | ✅ Yes (new 2025) | 🔴 Missing |
| Portfolio Backtesting | ❌ No | ✅ Yes | 🔴 Missing |
| Portfolio Optimizer | ❌ No | ✅ Sharpe/Volatility optimization | 🔴 Missing |
| Stress Testing | ❌ No | ✅ GFC 2008, COVID scenarios | 🔴 Missing |
| Return Attribution | ❌ No | ✅ Yes | 🔴 Missing |
| Holdings Overlap Analysis | ❌ No | ✅ Yes | 🔴 Missing |
| Transition Analysis | ❌ No | ✅ Yes | 🔴 Missing |
| Correlation Matrix | ❌ No | ✅ Yes | 🔴 Missing |
| Efficient Frontier Plotting | ❌ No | ✅ Yes | 🔴 Missing |
| Performance Proxy | ❌ No | ✅ Fill historical gaps | 🔴 Missing |
| Exposure Proxy | ❌ No | ✅ Asset allocation calc | 🔴 Missing |
| Cost Basis Tracking | ❌ No | ✅ Yes | 🔴 Missing |
| Gain/Loss Calculations | ❌ No | ✅ Yes | 🔴 Missing |
| Portfolio Rebalancing | ❌ No | ✅ Yes | 🔴 Missing |

### 1.4 Charting & Visualization

| Feature | Investor Center | YCharts | Gap |
|---------|-----------------|---------|-----|
| Price Charts | ✅ Candlestick, Line | ✅ Yes | ✅ Similar |
| Fundamental Charts | ❌ Limited | ✅ 4,000+ metrics | 🔴 Missing |
| Comparison Charts | ❌ Limited | ✅ Up to 12 securities | 🔴 Missing |
| Earnings Beat/Miss Charts | ❌ No | ✅ Yes | 🔴 Missing |
| Valuation vs Market Charts | ❌ No | ✅ Yes | 🔴 Missing |
| Return vs Benchmark Charts | ❌ No | ✅ Yes | 🔴 Missing |
| Scatter Plots | ❌ No | ✅ Risk vs Reward | 🔴 Missing |
| Economic Overlay | ❌ No | ✅ Yes | 🔴 Missing |
| Multiple Timeframes | ✅ Yes | ✅ Yes | ✅ Similar |
| Chart Export (PNG/PDF) | ❌ No | ✅ Yes | 🔴 Missing |
| Heatmaps | ✅ Reddit heatmap | ✅ General heatmaps | ✅ Similar |
| Gauge Charts | ✅ IC Score gauge | ❌ No | ✅ Advantage |

### 1.5 Comparison & Analysis Tools

| Feature | Investor Center | YCharts | Gap |
|---------|-----------------|---------|-----|
| Quickflows (1-click comparisons) | ❌ No | ✅ Yes | 🔴 Missing |
| Comp Tables | ❌ No | ✅ 4,000+ metrics | 🔴 Missing |
| AI Smart Compare | ❌ No | ✅ Yes (2025) | 🔴 Missing |
| AI Chat | ❌ No | ✅ Yes (March 2025) | 🔴 Missing |
| Timeseries Analysis | ❌ No | ✅ Yes | 🔴 Missing |
| Peer Comparison | ❌ No | ✅ Industry/Sector | 🔴 Missing |
| Risk Metrics Comparison | ❌ Limited | ✅ Alpha, Beta, Std Dev, etc. | 🟡 Limited |
| ESG Comparison | ❌ No | ✅ 30+ ESG metrics | 🔴 Missing |
| Holdings Comparison | ❌ No | ✅ Yes | 🔴 Missing |

### 1.6 Reporting & Export

| Feature | Investor Center | YCharts | Gap |
|---------|-----------------|---------|-----|
| PDF Report Builder | ❌ No | ✅ Drag-and-drop | 🔴 Missing |
| Pre-built Report Templates | ❌ No | ✅ 30+ modules | 🔴 Missing |
| Custom Branding | ❌ No | ✅ Logo, colors, disclosures | 🔴 Missing |
| FINRA-reviewed Reports | ❌ No | ✅ Yes | 🔴 Missing |
| Tearsheet Generation | ❌ No | ✅ Yes | 🔴 Missing |
| Client-ready PDFs | ❌ No | ✅ Yes | 🔴 Missing |
| Email Report Scheduling | ❌ No | ✅ Yes | 🔴 Missing |
| Excel Add-in | ❌ No | ✅ Dynamic data pull | 🔴 Missing |
| CSV Export | ❌ No | ✅ Yes | 🔴 Missing |
| API Access | ❌ No | ✅ Yes | 🔴 Missing |
| Data Export | ❌ Limited | ✅ Full | 🔴 Missing |

### 1.7 Alerts & Notifications

| Feature | Investor Center | YCharts | Gap |
|---------|-----------------|---------|-----|
| Price Alerts | ✅ Yes | ✅ Yes | ✅ Similar |
| Volume Alerts | ✅ Yes | ✅ Yes | ✅ Similar |
| News Alerts | ✅ Yes | ✅ Yes | ✅ Similar |
| Screen-based Alerts | ❌ No | ✅ Yes | 🔴 Missing |
| Economic Event Alerts | ❌ No | ✅ Yes | 🔴 Missing |
| Earnings Alerts | ✅ Yes | ✅ Yes | ✅ Similar |
| Email Notifications | ✅ Yes | ✅ Yes | ✅ Similar |
| In-app Notifications | ✅ Yes | ✅ Yes | ✅ Similar |

### 1.8 Unique Investor Center Features (Advantages)

| Feature | Investor Center | YCharts | Advantage |
|---------|-----------------|---------|-----------|
| IC Score (Proprietary) | ✅ 10-factor scoring | ❌ No | ✅ **Unique** |
| Reddit Sentiment Analysis | ✅ Real-time | ❌ No | ✅ **Unique** |
| Reddit Heatmap | ✅ Daily trending | ❌ No | ✅ **Unique** |
| Social Media Mentions | ✅ Multiple subreddits | ❌ No | ✅ **Unique** |
| FinBERT News Sentiment | ✅ AI-powered | ❌ Limited | ✅ **Advantage** |
| Cryptocurrency Coverage | ✅ Full CoinGecko | ⚠️ Limited (25) | ✅ **Advantage** |
| Representative Reddit Posts | ✅ Bullish/Bearish | ❌ No | ✅ **Unique** |
| Engagement Metrics | ✅ Upvotes, comments | ❌ No | ✅ **Unique** |

---

## Part 2: Data-Level Comparison

### 2.1 Financial Ratios & Metrics

#### Valuation Metrics

| Metric | Investor Center | YCharts | Gap |
|--------|-----------------|---------|-----|
| P/E Ratio | ✅ | ✅ | - |
| Forward P/E | ❌ | ✅ | 🔴 Missing |
| P/E (NTM) | ❌ | ✅ | 🔴 Missing |
| P/E to Growth (PEG) | ✅ | ✅ | - |
| P/B Ratio | ✅ | ✅ | - |
| P/S Ratio | ✅ | ✅ | - |
| P/FCF Ratio | ❌ | ✅ | 🔴 Missing |
| P/OCF Ratio | ❌ | ✅ | 🔴 Missing |
| P/Tangible Book | ❌ | ✅ | 🔴 Missing |
| EV/EBITDA | ✅ | ✅ | - |
| EV/EBIT | ❌ | ✅ | 🔴 Missing |
| EV/Sales | ✅ | ✅ | - |
| EV/FCF | ❌ | ✅ | 🔴 Missing |
| EV/Gross Profit | ❌ | ✅ | 🔴 Missing |
| Dividend Yield | ✅ | ✅ | - |
| Dividend Yield (Forward) | ❌ | ✅ | 🔴 Missing |
| Earnings Yield | ❌ | ✅ | 🔴 Missing |
| FCF Yield | ❌ | ✅ | 🔴 Missing |
| Shareholder Yield | ❌ | ✅ | 🔴 Missing |
| CAPE Ratio (Shiller P/E) | ❌ | ✅ | 🔴 Missing |

#### Profitability Metrics

| Metric | Investor Center | YCharts | Gap |
|--------|-----------------|---------|-----|
| Gross Margin | ✅ | ✅ | - |
| Operating Margin | ✅ | ✅ | - |
| Net Profit Margin | ✅ | ✅ | - |
| EBITDA Margin | ❌ | ✅ | 🔴 Missing |
| EBIT Margin | ❌ | ✅ | 🔴 Missing |
| FCF Margin | ❌ | ✅ | 🔴 Missing |
| ROE | ✅ | ✅ | - |
| ROA | ✅ | ✅ | - |
| ROIC | ✅ | ✅ | - |
| ROCE (Return on Capital Employed) | ❌ | ✅ | 🔴 Missing |
| ROE (5-year average) | ❌ | ✅ | 🔴 Missing |
| Cash Return on Invested Capital | ❌ | ✅ | 🔴 Missing |

#### Growth Metrics

| Metric | Investor Center | YCharts | Gap |
|--------|-----------------|---------|-----|
| Revenue Growth (YoY) | ✅ | ✅ | - |
| Revenue Growth (QoQ) | ❌ | ✅ | 🔴 Missing |
| Revenue Growth (3Y CAGR) | ❌ | ✅ | 🔴 Missing |
| Revenue Growth (5Y CAGR) | ❌ | ✅ | 🔴 Missing |
| EPS Growth (YoY) | ❌ | ✅ | 🔴 Missing |
| EPS Growth (5Y CAGR) | ❌ | ✅ | 🔴 Missing |
| EBITDA Growth | ❌ | ✅ | 🔴 Missing |
| FCF Growth | ❌ | ✅ | 🔴 Missing |
| Book Value Growth | ❌ | ✅ | 🔴 Missing |
| Dividend Growth (5Y CAGR) | ❌ | ✅ | 🔴 Missing |
| Dividend Growth (10Y CAGR) | ❌ | ✅ | 🔴 Missing |

#### Liquidity & Solvency Metrics

| Metric | Investor Center | YCharts | Gap |
|--------|-----------------|---------|-----|
| Current Ratio | ✅ | ✅ | - |
| Quick Ratio | ✅ | ✅ | - |
| Cash Ratio | ❌ | ✅ | 🔴 Missing |
| Debt/Equity | ✅ | ✅ | - |
| Debt/Assets | ✅ | ✅ | - |
| Debt/EBITDA | ❌ | ✅ | 🔴 Missing |
| Debt/Capital | ❌ | ✅ | 🔴 Missing |
| Net Debt/EBITDA | ❌ | ✅ | 🔴 Missing |
| Interest Coverage | ✅ | ✅ | - |
| Fixed Charge Coverage | ❌ | ✅ | 🔴 Missing |
| Altman Z-Score | ❌ | ✅ | 🔴 Missing |
| Piotroski F-Score | ❌ | ✅ | 🔴 Missing |
| Beneish M-Score | ❌ | ✅ | 🔴 Missing |

#### Efficiency Metrics

| Metric | Investor Center | YCharts | Gap |
|--------|-----------------|---------|-----|
| Asset Turnover | ✅ | ✅ | - |
| Inventory Turnover | ✅ | ✅ | - |
| Receivables Turnover | ✅ | ✅ | - |
| Payables Turnover | ❌ | ✅ | 🔴 Missing |
| Fixed Asset Turnover | ❌ | ✅ | 🔴 Missing |
| Working Capital Turnover | ❌ | ✅ | 🔴 Missing |
| Days Sales Outstanding | ❌ | ✅ | 🔴 Missing |
| Days Inventory Outstanding | ❌ | ✅ | 🔴 Missing |
| Days Payables Outstanding | ❌ | ✅ | 🔴 Missing |
| Cash Conversion Cycle | ❌ | ✅ | 🔴 Missing |

#### Risk Metrics

| Metric | Investor Center | YCharts | Gap |
|--------|-----------------|---------|-----|
| Beta | ✅ | ✅ | - |
| Alpha | ✅ | ✅ | - |
| Sharpe Ratio | ✅ | ✅ | - |
| Sortino Ratio | ❌ | ✅ | 🔴 Missing |
| Treynor Ratio | ❌ | ✅ | 🔴 Missing |
| Calmar Ratio | ❌ | ✅ | 🔴 Missing |
| Information Ratio | ❌ | ✅ | 🔴 Missing |
| Standard Deviation | ❌ | ✅ | 🔴 Missing |
| Max Drawdown | ❌ | ✅ | 🔴 Missing |
| Value at Risk (VaR) | ❌ | ✅ | 🔴 Missing |
| Upside/Downside Capture | ❌ | ✅ | 🔴 Missing |
| R-Squared | ❌ | ✅ | 🔴 Missing |
| Tracking Error | ❌ | ✅ | 🔴 Missing |

#### Per Share Metrics

| Metric | Investor Center | YCharts | Gap |
|--------|-----------------|---------|-----|
| EPS (Basic) | ✅ | ✅ | - |
| EPS (Diluted) | ✅ | ✅ | - |
| EPS (Forward) | ❌ | ✅ | 🔴 Missing |
| EPS (NTM) | ❌ | ✅ | 🔴 Missing |
| Book Value Per Share | ✅ | ✅ | - |
| Tangible BVPS | ✅ | ✅ | - |
| Cash Per Share | ❌ | ✅ | 🔴 Missing |
| FCF Per Share | ❌ | ✅ | 🔴 Missing |
| Revenue Per Share | ❌ | ✅ | 🔴 Missing |
| Dividend Per Share | ❌ | ✅ | 🔴 Missing |
| Sales Per Employee | ❌ | ✅ | 🔴 Missing |
| Net Income Per Employee | ❌ | ✅ | 🔴 Missing |

#### Dividend Metrics

| Metric | Investor Center | YCharts | Gap |
|--------|-----------------|---------|-----|
| Dividend Yield | ✅ | ✅ | - |
| Forward Dividend Yield | ❌ | ✅ | 🔴 Missing |
| Dividend Payout Ratio | ❌ | ✅ | 🔴 Missing |
| FCF Payout Ratio | ❌ | ✅ | 🔴 Missing |
| Dividend Coverage Ratio | ❌ | ✅ | 🔴 Missing |
| Consecutive Dividend Increases | ❌ | ✅ | 🔴 Missing |
| Ex-Dividend Date | ❌ | ✅ | 🔴 Missing |
| Payment Date | ❌ | ✅ | 🔴 Missing |
| Dividend Frequency | ❌ | ✅ | 🔴 Missing |

### 2.2 Fund-Specific Metrics (All Missing in Investor Center)

| Metric | Investor Center | YCharts |
|--------|-----------------|---------|
| Expense Ratio | ❌ | ✅ |
| Gross Expense Ratio | ❌ | ✅ |
| Actual Management Fee | ❌ | ✅ |
| 12b-1 Fee | ❌ | ✅ |
| Load (Front/Back) | ❌ | ✅ |
| Tax Cost Ratio | ❌ | ✅ |
| Turnover Ratio | ❌ | ✅ |
| Fund Size (AUM) | ❌ | ✅ |
| Fund Flows | ❌ | ✅ |
| Manager Tenure | ❌ | ✅ |
| Morningstar Rating | ❌ | ✅ |
| Morningstar Category | ❌ | ✅ |
| NAV Discount/Premium | ❌ | ✅ |
| Sector Exposure % | ❌ | ✅ |
| Geographic Exposure % | ❌ | ✅ |
| Top 10 Holdings % | ❌ | ✅ |
| Active Share | ❌ | ✅ |
| Tracking Difference | ❌ | ✅ |

### 2.3 Bond-Specific Metrics (All Missing in Investor Center)

| Metric | Investor Center | YCharts |
|--------|-----------------|---------|
| Yield to Maturity | ❌ | ✅ |
| Yield to Worst | ❌ | ✅ |
| Current Yield | ❌ | ✅ |
| Coupon Rate | ❌ | ✅ |
| Duration (Modified) | ❌ | ✅ |
| Duration (Effective) | ❌ | ✅ |
| Convexity | ❌ | ✅ |
| Credit Rating | ❌ | ✅ |
| Maturity Date | ❌ | ✅ |
| Call Date | ❌ | ✅ |
| Spread to Treasury | ❌ | ✅ |
| OAS (Option-Adjusted Spread) | ❌ | ✅ |

### 2.4 ESG Metrics (All Missing in Investor Center)

| Metric | Investor Center | YCharts |
|--------|-----------------|---------|
| ESG Score (Overall) | ❌ | ✅ |
| Environmental Score | ❌ | ✅ |
| Social Score | ❌ | ✅ |
| Governance Score | ❌ | ✅ |
| Carbon Intensity | ❌ | ✅ |
| GHG Emissions (Scope 1/2/3) | ❌ | ✅ |
| ESG Controversies | ❌ | ✅ |
| ESG Momentum | ❌ | ✅ |
| Portfolio ESG Aggregate | ❌ | ✅ |
| UN SDG Alignment | ❌ | ✅ |

### 2.5 Analyst Data

| Data Point | Investor Center | YCharts | Gap |
|------------|-----------------|---------|-----|
| Analyst Ratings | ✅ Basic | ✅ Full | 🟡 Limited |
| Target Price Consensus | ❌ | ✅ | 🔴 Missing |
| Target Price (High/Low/Mean) | ❌ | ✅ | 🔴 Missing |
| Revenue Estimates | ❌ | ✅ | 🔴 Missing |
| EPS Estimates | ❌ | ✅ | 🔴 Missing |
| EBITDA Estimates | ❌ | ✅ | 🔴 Missing |
| Estimate Revisions | ❌ | ✅ | 🔴 Missing |
| Analyst Count | ❌ | ✅ | 🔴 Missing |
| Buy/Hold/Sell Breakdown | ❌ | ✅ | 🔴 Missing |

### 2.6 Economic Indicators (All Missing in Investor Center)

| Category | Investor Center | YCharts |
|----------|-----------------|---------|
| GDP Data | ❌ | ✅ 500,000+ |
| Inflation Metrics (CPI, PPI, PCE) | ❌ | ✅ |
| Employment Data | ❌ | ✅ |
| Interest Rates | ❌ | ✅ |
| Federal Reserve Data | ❌ | ✅ |
| Treasury Yields | ❌ | ✅ |
| Housing Data | ❌ | ✅ |
| Consumer Sentiment | ❌ | ✅ |
| Manufacturing PMI | ❌ | ✅ |
| Trade Data | ❌ | ✅ |
| Economic Calendar | ❌ | ✅ |

### 2.7 Company KPIs (Missing in Investor Center)

YCharts provides granular, company-specific KPIs that Investor Center lacks:

| KPI Examples | Investor Center | YCharts |
|--------------|-----------------|---------|
| Amazon Advertising Revenue | ❌ | ✅ |
| Netflix Paid Subscribers | ❌ | ✅ |
| Disney+ Subscribers | ❌ | ✅ |
| Apple iPhone Revenue | ❌ | ✅ |
| Tesla Deliveries | ❌ | ✅ |
| Google Search Revenue | ❌ | ✅ |
| Meta DAU/MAU | ❌ | ✅ |
| Industry-specific metrics | ❌ | ✅ |

---

## Part 3: Priority Gap Analysis & Recommendations

### 3.1 Critical Gaps (Must Have)

These gaps significantly limit the platform's competitiveness:

| Priority | Gap | Impact | Effort |
|----------|-----|--------|--------|
| 1 | **Mutual Fund/ETF Data** | Cannot serve fund investors | High |
| 2 | **Forward Estimates (P/E, EPS)** | Limits valuation analysis | Medium |
| 3 | **Portfolio Backtesting** | Cannot test strategies | High |
| 4 | **Data Export (CSV/Excel)** | Users can't use data externally | Low |
| 5 | **More Screener Filters** | Limited discovery capability | Medium |
| 6 | **Comparison Tables** | Hard to compare securities | Medium |

### 3.2 High-Value Gaps (Should Have)

| Priority | Gap | Impact | Effort |
|----------|-----|--------|--------|
| 7 | **Economic Indicators** | Cannot correlate macro to stocks | High |
| 8 | **Model Portfolios** | Cannot build strategies | Medium |
| 9 | **Risk Metrics (Sortino, Max DD)** | Limited risk analysis | Medium |
| 10 | **PDF Report Generation** | Cannot share professional reports | High |
| 11 | **Growth Rate CAGRs** | Limited growth analysis | Low |
| 12 | **Dividend Metrics** | Cannot serve income investors | Low |

### 3.3 Differentiating Gaps (Nice to Have)

| Priority | Gap | Impact | Effort |
|----------|-----|--------|--------|
| 13 | **ESG Data** | Missing for ESG investors | Medium |
| 14 | **Bond Data** | Cannot serve fixed income | High |
| 15 | **AI Chat** | Modern UX expectation | High |
| 16 | **Quickflows-like Tool** | Faster analysis | Medium |
| 17 | **International Stocks** | Limited to US only | High |

### 3.4 Investor Center Competitive Advantages (Protect & Enhance)

| Advantage | Recommendation |
|-----------|----------------|
| **IC Score** | Add historical trends, factor attribution, peer percentile |
| **Reddit Sentiment** | Expand to Twitter/X, StockTwits, Discord |
| **Real-time Crypto** | Add DeFi metrics, whale tracking, on-chain data |
| **FinBERT Sentiment** | Add sentiment scores to screener filters |
| **Social Heatmap** | Add customizable date ranges, sector filters |

---

## Part 4: Detailed Feature Recommendations

### 4.1 Quick Wins (Low Effort, High Impact)

1. **Add CSV Export** - Allow users to export screener results, watchlists
2. **Add More Financial Ratios** - P/FCF, Forward P/E, Dividend Payout
3. **Add CAGR Calculations** - 3Y, 5Y, 10Y growth rates
4. **Add More Screener Filters** - Revenue growth ranges, debt ratios
5. **Add Target Price** - Show analyst consensus target
6. **Add Dividend Data** - Ex-date, payment date, frequency

### 4.2 Medium-Term Initiatives

1. **Build Fund Screener** - Partner with Morningstar or use free sources
2. **Add Comparison Tables** - Side-by-side security comparison
3. **Build Portfolio Backtesting** - Historical performance simulation
4. **Add Risk Metrics Dashboard** - Sharpe, Sortino, Max Drawdown
5. **Add Chart Export** - PNG/PDF chart download
6. **Build PDF Report Builder** - Customizable reports

### 4.3 Long-Term Strategic Initiatives

1. **Economic Indicators Database** - Partner with FRED API (free)
2. **Individual Bond Data** - Partner with S&P Global or similar
3. **International Stock Coverage** - Expand data sources
4. **AI Chat Interface** - Natural language querying
5. **Excel Add-in** - Power user data access
6. **API Access** - Developer/institutional use

---

## Part 5: Data Source Recommendations

### 5.1 Free/Low-Cost Data Sources to Add

| Data Type | Source | Coverage |
|-----------|--------|----------|
| Economic Indicators | FRED API (St. Louis Fed) | 800,000+ series, free |
| ETF Holdings | ETF sponsors (iShares, Vanguard) | Free from provider sites |
| Mutual Fund Data | SEC EDGAR N-PORT filings | Free, quarterly |
| Analyst Estimates | Zacks, Yahoo Finance | Free/low cost |
| Dividend Data | Nasdaq, Polygon | Available in current sources |
| ESG Data | Sustainalytics, MSCI (paid) | Requires partnership |

### 5.2 Current Data Source Optimization

| Source | Currently Used For | Expansion Opportunity |
|--------|-------------------|----------------------|
| Polygon.io | Prices, fundamentals | Add options data, more metrics |
| FMP | TTM ratios | Add estimates, forward metrics |
| SEC EDGAR | Filings | Add fund holdings (N-PORT) |
| CoinGecko | Crypto prices | Add DeFi, NFT data |

---

## Appendix A: Complete Feature Checklist

### Stock Analysis Features

- [x] Real-time quotes
- [x] Historical prices
- [x] Basic fundamentals
- [x] Income statements
- [x] Balance sheets
- [x] Cash flow statements
- [x] Insider trading
- [x] Institutional ownership
- [x] News feed
- [x] Sentiment analysis
- [ ] Forward estimates
- [ ] Analyst target prices
- [ ] Earnings estimates
- [ ] Revision trends
- [ ] Peer comparison
- [ ] Sector ranking
- [ ] Industry metrics
- [ ] Company-specific KPIs

### Screening Features

- [x] Market cap filter
- [x] Sector filter
- [x] P/E filter
- [x] Dividend yield filter
- [x] Revenue growth filter
- [x] IC Score filter
- [ ] P/FCF filter
- [ ] Forward P/E filter
- [ ] PEG filter
- [ ] ROE filter
- [ ] Debt/Equity filter
- [ ] Beta filter
- [ ] 52-week high/low filter
- [ ] Volume filter
- [ ] Custom scoring model
- [ ] Save/load screens
- [ ] Screen alerts

### Portfolio Features

- [x] Watchlists
- [x] Target prices
- [x] Notes/tags
- [ ] Model portfolios
- [ ] Backtesting
- [ ] Stress testing
- [ ] Optimization
- [ ] Rebalancing
- [ ] Attribution
- [ ] Correlation matrix
- [ ] Holdings overlap
- [ ] Efficient frontier

### Reporting Features

- [ ] PDF export
- [ ] CSV export
- [ ] Excel export
- [ ] Custom templates
- [ ] Branded reports
- [ ] Scheduled reports
- [ ] Email delivery
- [ ] API access

---

## Appendix B: YCharts Pricing Context

For reference, YCharts pricing (as of 2025):

| Tier | Annual Price | Key Features |
|------|--------------|--------------|
| Starter | ~$5,000/year | Basic access |
| Professional | ~$10,000/year | Full platform + Excel Add-in |
| Enterprise | Custom | API + white-labeling |

Investor Center can compete effectively in the retail/prosumer segment by:
1. Offering core YCharts-like features at lower/no cost
2. Differentiating with social sentiment (Reddit, etc.)
3. Focusing on individual investors vs. financial advisors

---

*This analysis was compiled on January 26, 2026. YCharts features are based on publicly available information and may have been updated since.*
