# Implementation Prompts: Phase 1 (MVP)
## InvestorCenter.ai - Months 1-6

**Version:** 1.0
**Date:** November 12, 2025

---

## Overview

This document contains detailed prompts for implementing all Phase 1 (MVP) features. Each prompt can be used independently to guide implementation of a specific feature or subsystem.

---

## Prompt 1: Database Schema Setup

```
I need you to set up the complete database schema for InvestorCenter.ai,
a stock analysis platform.

Requirements:
1. Use PostgreSQL 15+ as the primary database
2. Add TimescaleDB extension for time-series data
3. Create all tables defined in tech-spec-02-implementation-details.md
4. Include proper indexes for performance
5. Add foreign key constraints
6. Set up database migrations using Alembic
7. Create initial seed data for testing

Tables to create:
- users (authentication, profiles, subscriptions)
- companies (stock information, sectors)
- ic_scores (our proprietary scoring system with 10 factors)
- financials (income statement, balance sheet, cash flow)
- insider_trades (Form 4 data from SEC)
- institutional_holdings (Form 13F data)
- analyst_ratings (Wall Street analyst data)
- news_articles (news with sentiment scores)
- watchlists & watchlist_stocks
- portfolios, portfolio_positions, portfolio_transactions
- alerts (user notifications)
- stock_prices (TimescaleDB hypertable for time-series)
- technical_indicators (TimescaleDB hypertable)

Deliverables:
1. Complete SQL schema file (schema.sql)
2. Alembic migration files
3. Database connection module (database.py)
4. SQLAlchemy models (models.py)
5. Seed data script (seed.py) with sample stocks (AAPL, MSFT, GOOGL, AMZN, TSLA)
6. README with setup instructions

Technical details:
- Use UUIDs for user-facing IDs (users, portfolios, watchlists)
- Use SERIAL/BIGSERIAL for internal IDs
- Store JSON data in JSONB columns where flexible schema needed
- Use proper data types (DECIMAL for money, BIGINT for large numbers)
- Add created_at/updated_at timestamps to all tables
- Use ON DELETE CASCADE where appropriate
- Create composite indexes for frequently queried column combinations

Please implement this database schema with production-ready code, comprehensive
error handling, and detailed comments.
```

---

## Prompt 2: SEC EDGAR Data Scraper

```
I need you to build a data scraper that fetches financial data from the SEC EDGAR
database for InvestorCenter.ai.

Requirements:
1. Scrape financial statements (10-K annual, 10-Q quarterly)
2. Parse XBRL data to extract key metrics
3. Fetch insider trading data (Form 4)
4. Fetch institutional holdings (Form 13F)
5. Store all data in PostgreSQL database
6. Handle rate limiting (SEC allows 10 requests/second)
7. Retry logic for failed requests
8. Comprehensive logging

Data to Extract from Financial Statements:
- Revenue, Cost of Revenue, Gross Profit
- Operating Expenses, Operating Income, Net Income
- EPS (basic and diluted), Shares Outstanding
- Total Assets, Total Liabilities, Shareholders' Equity
- Cash, Short-term Debt, Long-term Debt
- Operating Cash Flow, Investing Cash Flow, Financing Cash Flow
- Free Cash Flow, CapEx

Calculated Metrics:
- P/E, P/B, P/S ratios
- Debt-to-Equity, Current Ratio, Quick Ratio
- ROE, ROA, ROIC
- Gross Margin, Operating Margin, Net Margin

Implementation Details:
- Use Python 3.11+ with asyncio for concurrent requests
- Use httpx for HTTP requests
- Parse XBRL using lxml or similar library
- Store raw filings in S3/MinIO for audit trail
- Create backfill script for historical data (10+ years)
- Schedule daily updates using Celery

Files to Create:
1. sec_scraper.py (main scraper class)
2. xbrl_parser.py (XBRL parsing utilities)
3. models.py (data models for parsed data)
4. tasks.py (Celery tasks for scheduled scraping)
5. config.py (configuration, SEC API endpoints)
6. tests/ (comprehensive test suite)

Error Handling:
- Network errors: Retry with exponential backoff
- Parsing errors: Log and skip, don't crash
- Missing data: Handle gracefully, store NULL
- Rate limit errors: Pause and resume

Please implement a production-ready SEC scraper with comprehensive error handling,
logging, and test coverage.
```

---

## Prompt 3: IC Score Calculation Engine

```
I need you to implement the InvestorCenter Score (IC Score) calculation engine,
our proprietary stock scoring system that evaluates stocks on 10 factors.

Context:
The IC Score is the core differentiator of our platform. It scores stocks 1-100
based on 10 factors: Value, Growth, Profitability, Financial Health, Momentum,
Analyst Consensus, Insider Activity, Institutional Ownership, News Sentiment,
and Technical Indicators.

Requirements:
1. Implement all 10 factor calculation functions
2. Calculate weighted overall score (1-100)
3. Assign rating labels (Strong Buy, Buy, Hold, Sell, Strong Sell)
4. Calculate sector-relative percentiles
5. Track historical scores (store daily snapshots)
6. Handle missing data gracefully
7. Performance: <500ms per stock calculation
8. Batch processing: Calculate 1000+ stocks efficiently

Factor Weights (default):
- Value: 12%
- Growth: 15%
- Profitability: 12%
- Financial Health: 10%
- Momentum: 8%
- Analyst Consensus: 10%
- Insider Activity: 8%
- Institutional: 10%
- News Sentiment: 7%
- Technical: 8%

Implementation Details:

1. Value Factor:
   - Metrics: P/E, P/B, P/S, PEG, EV/EBITDA
   - Lower multiples = higher score
   - Compare to sector median (percentile ranking)

2. Growth Factor:
   - Revenue growth (1Y, 3Y, 5Y CAGR)
   - Earnings growth (1Y, 3Y, 5Y CAGR)
   - FCF growth (1Y, 3Y, 5Y CAGR)
   - Forward estimates (30% weight)
   - Consistency bonus (penalize volatility)

3. Profitability Factor:
   - Margins: Gross, Operating, Net
   - Returns: ROE, ROA, ROIC
   - Sector-relative comparison

4. Financial Health Factor:
   - Liquidity: Current ratio, Quick ratio
   - Leverage: Debt/Equity, Interest coverage
   - Altman Z-Score
   - Sector-adjusted (utilities OK with higher debt)

5. Momentum Factor:
   - Price performance: 1M, 3M, 6M, 12M
   - Relative strength vs. S&P 500
   - Volume analysis
   - MA position (50-day vs 200-day)

6. Analyst Consensus Factor:
   - Buy/Hold/Sell distribution
   - Consensus rating (1-5 scale)
   - Recent changes (upgrades/downgrades)
   - Price target upside

7. Insider Activity Factor:
   - Net buying/selling (3M, 6M, 12M)
   - Transaction size significance
   - Role weighting (CEO > Director)
   - Clustered activity detection

8. Institutional Factor:
   - Total ownership %
   - Change in ownership (QoQ)
   - Notable investors (Buffett, etc.)
   - Optimal range: 60-80%

9. News Sentiment Factor:
   - NLP sentiment score (30 days)
   - Positive/negative ratio
   - Sentiment trend
   - Source credibility weighting

10. Technical Factor:
    - RSI, MACD, Bollinger Bands
    - Support/resistance levels
    - Pattern signals
    - Volume confirmation

Files to Create:
1. ic_score_calculator.py (main calculator class)
2. factor_calculators/ (directory with each factor)
   - value_factor.py
   - growth_factor.py
   - profitability_factor.py
   - financial_health_factor.py
   - momentum_factor.py
   - analyst_factor.py
   - insider_factor.py
   - institutional_factor.py
   - sentiment_factor.py
   - technical_factor.py
3. sector_utils.py (sector comparison utilities)
4. scoring_utils.py (percentile calculation, etc.)
5. batch_calculator.py (efficient batch processing)
6. tasks.py (Celery task for daily recalculation)
7. tests/ (comprehensive test suite)

Performance Requirements:
- Single stock: <500ms
- Batch of 1000 stocks: <5 minutes
- Use database connection pooling
- Cache sector medians (24h TTL)
- Parallel processing where possible

Error Handling:
- Missing data: Skip factor, redistribute weights
- Minimum 6/10 factors needed for score
- Display data completeness %
- Confidence level (High/Medium/Low)

Please implement the IC Score calculation engine with production-ready code,
comprehensive test coverage, and detailed documentation explaining the methodology.
```

---

## Prompt 4: REST API - Stock Endpoints

```
I need you to implement the REST API endpoints for stock data in InvestorCenter.ai.

Requirements:
1. Use FastAPI framework (Python)
2. Implement all stock-related endpoints
3. JWT authentication for protected routes
4. Rate limiting (100 req/day free, 10k req/day premium)
5. Response caching with Redis
6. Comprehensive error handling
7. OpenAPI documentation
8. Input validation with Pydantic

Endpoints to Implement:

GET /api/v1/stocks
- List all stocks (paginated)
- Filters: sector, market_cap_min, market_cap_max, ic_score_min
- Sort: ticker, name, ic_score, market_cap
- Response: List of stock summaries

GET /api/v1/stocks/{ticker}
- Get stock overview
- Response: Company info, current price, IC Score, key metrics

GET /api/v1/stocks/{ticker}/score
- Get detailed IC Score breakdown
- Response: Overall score, all 10 factor scores, trends, percentiles

GET /api/v1/stocks/{ticker}/factors
- Get individual factor details
- Query params: factor (optional, specific factor)
- Response: Factor score, contributing metrics, sector comparison

GET /api/v1/stocks/{ticker}/history
- Get historical IC Scores
- Query params: start_date, end_date, interval (daily/weekly/monthly)
- Response: Time series of scores

GET /api/v1/stocks/{ticker}/financials
- Get financial statements
- Query params: statement_type (income/balance/cash), period (quarter/annual)
- Response: Financial data with multi-year comparison

GET /api/v1/stocks/{ticker}/earnings
- Get earnings data
- Response: Upcoming earnings, historical surprises

GET /api/v1/stocks/{ticker}/analysts
- Get analyst ratings
- Response: Consensus, individual ratings, price targets

GET /api/v1/stocks/{ticker}/insider
- Get insider trading activity
- Query params: start_date, end_date, min_value
- Response: Recent insider transactions, net activity

GET /api/v1/stocks/{ticker}/institutional
- Get institutional ownership
- Response: Top holders, recent changes, ownership %

GET /api/v1/stocks/{ticker}/news
- Get news articles
- Query params: limit, offset, start_date, sentiment (positive/negative/neutral)
- Response: News articles with sentiment scores

GET /api/v1/stocks/{ticker}/sentiment
- Get aggregated sentiment
- Query params: timeframe (7d/30d/90d)
- Response: Overall sentiment score, breakdown, trend

GET /api/v1/stocks/{ticker}/chart
- Get chart data (OHLCV)
- Query params: interval (1min/5min/1hour/1day), start_date, end_date
- Response: Price data, volume

POST /api/v1/stocks/{ticker}/compare
- Compare multiple stocks
- Body: {tickers: ['AAPL', 'MSFT', 'GOOGL']}
- Response: Side-by-side comparison of key metrics

GET /api/v1/search
- Search stocks
- Query params: q (search query), limit
- Response: Matching stocks

Response Format (success):
{
  "success": true,
  "data": { ... },
  "meta": {
    "timestamp": "2025-11-12T14:30:00Z",
    "cached": false,
    "cache_ttl": 3600
  }
}

Response Format (error):
{
  "success": false,
  "error": {
    "code": "INVALID_TICKER",
    "message": "Stock ticker not found",
    "details": {"ticker": "XYZ"}
  },
  "meta": {
    "timestamp": "2025-11-12T14:30:00Z"
  }
}

Technical Details:
1. Use dependency injection for database session
2. Implement caching decorator for GET endpoints
3. Add rate limiting middleware
4. Validate ticker symbols (uppercase, 1-5 chars)
5. Handle pagination with cursor-based approach for large datasets
6. Add CORS middleware for web app
7. Implement request ID tracking for debugging
8. Add performance logging (response times)

Files to Create:
1. main.py (FastAPI app initialization)
2. routers/stocks.py (stock endpoints)
3. dependencies.py (database session, auth, etc.)
4. schemas/ (Pydantic request/response models)
   - stock_schemas.py
   - common_schemas.py
5. middleware/ (rate limiting, CORS, logging)
6. utils/ (helpers, formatters)
7. tests/ (endpoint tests)

Cache Strategy:
- Stock overview: 1 hour
- IC Score: 24 hours
- Price data: 1 minute (market hours), 1 hour (off hours)
- News: 15 minutes
- Financials: 24 hours

Please implement production-ready API endpoints with comprehensive error handling,
input validation, caching, and test coverage.
```

---

## Prompt 5: Frontend - Stock Page with IC Score Display

```
I need you to implement the stock detail page for InvestorCenter.ai that prominently
displays our IC Score and factor breakdown.

Requirements:
1. Use Next.js 14+ (App Router) with TypeScript
2. Use Tailwind CSS + shadcn/ui for styling
3. Implement responsive design (mobile-first)
4. Fetch data from REST API
5. Use TanStack Query for data fetching and caching
6. Implement loading states and error handling
7. SEO optimization (meta tags, structured data)

Page Structure:

1. Header Section:
   - Stock ticker and company name
   - Current price with real-time updates
   - Change ($ and %)
   - Market cap, volume, P/E ratio

2. IC Score Card (Prominent):
   - Large, eye-catching display
   - Overall score (1-100) with progress bar
   - Star rating (1-5 stars)
   - Rating label (Strong Buy, Buy, Hold, Sell, Strong Sell)
   - Color-coded: Green (70+), Yellow (50-69), Red (<50)
   - "View Breakdown" button to expand factors

3. Factor Breakdown (Expandable):
   - All 10 factors displayed
   - Each factor shows:
     * Score (1-100)
     * Letter grade (A+ to F)
     * Sector percentile
     * Trend indicator (↑ improving, → stable, ↓ declining)
     * Progress bar visualization
   - Click any factor to drill down into detailed metrics
   - Responsive: Stack vertically on mobile, 2 columns on tablet, 3 on desktop

4. Tabs:
   - Overview (IC Score, key metrics, description)
   - Chart (price chart with indicators)
   - Financials (income statement, balance sheet, cash flow)
   - News (news feed with sentiment)
   - Analysis (detailed factor analysis)

5. Sidebar (Desktop):
   - Add to Watchlist button
   - Quick stats
   - Analyst consensus
   - Insider activity summary
   - Related stocks

UI Components to Create:

1. ICScoreCard.tsx
   - Props: ticker, score, rating, factors (optional for breakdown)
   - Displays overall score prominently
   - Expandable factor breakdown
   - Animated progress bars
   - Responsive design

2. FactorBreakdown.tsx
   - Props: factors (array of 10 factors with scores)
   - Grid layout
   - Each factor as FactorCard component
   - Click to expand details

3. FactorCard.tsx
   - Props: name, score, percentile, trend
   - Visual progress bar
   - Grade display (A+ to F)
   - Trend indicator arrow
   - Hover state with tooltip

4. FactorDetailModal.tsx
   - Props: factor (detailed factor data)
   - Modal overlay
   - Show all contributing metrics
   - Sector comparison table
   - Historical trend chart

5. StockHeader.tsx
   - Stock symbol, name, price
   - Real-time price updates (WebSocket)
   - Add to watchlist button

6. PriceChart.tsx
   - Use TradingView Lightweight Charts or ECharts
   - OHLC candlestick default
   - Timeframe selector (1D, 1W, 1M, 3M, 6M, 1Y, ALL)
   - Technical indicators dropdown

Data Fetching:
```typescript
// API client
async function getStockScore(ticker: string) {
  const response = await fetch(`/api/v1/stocks/${ticker}/score`)
  if (!response.ok) throw new Error('Failed to fetch')
  return response.json()
}

// React Query hook
function useStockScore(ticker: string) {
  return useQuery({
    queryKey: ['stock', ticker, 'score'],
    queryFn: () => getStockScore(ticker),
    staleTime: 1000 * 60 * 60, // 1 hour
  })
}
```

Example Component Structure:
```tsx
// app/stocks/[ticker]/page.tsx
export default function StockPage({ params }: { params: { ticker: string } }) {
  const { data: stock, isLoading, error } = useStock(params.ticker)
  const { data: score } = useStockScore(params.ticker)

  if (isLoading) return <LoadingSpinner />
  if (error) return <ErrorMessage error={error} />

  return (
    <div className="container mx-auto px-4 py-8">
      <StockHeader stock={stock} />

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mt-8">
        <div className="lg:col-span-2">
          <ICScoreCard ticker={stock.ticker} score={score} />

          <Tabs defaultValue="chart" className="mt-6">
            <TabsList>
              <TabsTrigger value="chart">Chart</TabsTrigger>
              <TabsTrigger value="financials">Financials</TabsTrigger>
              <TabsTrigger value="news">News</TabsTrigger>
              <TabsTrigger value="analysis">Analysis</TabsTrigger>
            </TabsList>
            <TabsContent value="chart">
              <PriceChart ticker={stock.ticker} />
            </TabsContent>
            {/* ... other tabs ... */}
          </Tabs>
        </div>

        <div className="lg:col-span-1">
          <Sidebar stock={stock} />
        </div>
      </div>
    </div>
  )
}
```

Styling Guidelines:
- Use Tailwind CSS utilities
- Color scheme: Green for positive, Red for negative, Blue for neutral
- Typography: Inter font (sans-serif)
- Spacing: Consistent 4px grid (space-4, space-6, space-8)
- Rounded corners: rounded-lg for cards
- Shadows: shadow-md for cards, shadow-lg for modals
- Dark mode support: Use dark: prefix classes

Animations:
- Progress bars: Smooth fill animation on mount
- Cards: Subtle hover effect (scale-105)
- Modals: Fade in backdrop, slide in content
- Transitions: transition-all duration-200

Accessibility:
- Semantic HTML (header, main, aside)
- ARIA labels for icons and buttons
- Keyboard navigation (Tab, Enter, Escape)
- Screen reader friendly
- Color contrast compliance (WCAG AA)

Performance:
- Code splitting: Dynamic imports for heavy components
- Image optimization: next/image for logos
- Lazy loading: Intersection Observer for below-fold content
- Memoization: useMemo for expensive calculations
- Virtual scrolling: For long lists (news feed)

Files to Create:
1. app/stocks/[ticker]/page.tsx (main stock page)
2. components/stock/
   - ICScoreCard.tsx
   - FactorBreakdown.tsx
   - FactorCard.tsx
   - FactorDetailModal.tsx
   - StockHeader.tsx
   - PriceChart.tsx
   - QuickStats.tsx
   - Sidebar.tsx
3. hooks/
   - useStock.ts
   - useStockScore.ts
   - useStockPrice.ts (WebSocket)
4. lib/
   - api-client.ts
   - utils.ts (formatters, helpers)
5. types/
   - stock.ts (TypeScript interfaces)

Please implement a production-ready stock page with comprehensive IC Score display,
responsive design, accessibility, and excellent UX.
```

---

## Prompt 6: Advanced Stock Screener

```
I need you to implement an advanced stock screener for InvestorCenter.ai that allows
users to filter stocks by 500+ criteria including our IC Score.

Requirements:
1. Use Next.js 14+ with TypeScript for frontend
2. FastAPI for backend screening engine
3. Support 500+ filter criteria
4. Real-time results as filters change
5. Save and share custom screens
6. Pre-built screen templates
7. Export results to CSV/Excel
8. Performance: <5 seconds for 1000+ stocks

Filter Categories:

1. IC Score Filters:
   - Overall score (min/max range)
   - Individual factor scores (10 factors)
   - Rating (Strong Buy, Buy, Hold, Sell, Strong Sell)
   - Sector percentile
   - Score trend (improving/declining)

2. Fundamental Filters:
   - Valuation: P/E, P/B, P/S, PEG, EV/EBITDA, Price/FCF
   - Profitability: Gross margin, Operating margin, Net margin, ROE, ROA, ROIC
   - Growth: Revenue growth, EPS growth, FCF growth (1Y, 3Y, 5Y)
   - Financial Health: Current ratio, Quick ratio, Debt/Equity, Interest coverage
   - Dividends: Yield, Payout ratio, Growth rate, Years of increases
   - Size: Market cap, Revenue, Enterprise Value

3. Technical Filters:
   - Price: Above/Below MA (20/50/100/200), 52-week high/low, % off highs/lows
   - Volume: Average volume, Volume spike %, Volume trend
   - Momentum: RSI, MACD, ROC, Stochastic
   - Volatility: Beta, ATR, Historical volatility
   - Patterns: Breakouts, Reversals, specific patterns

4. Quality Filters:
   - Analyst consensus: Rating, # analysts, Recent changes
   - Insider activity: Net buying/selling, Transaction value
   - Institutional: Ownership %, Change in ownership
   - News sentiment: Score, Trend, Article count
   - Earnings: Surprise streak, Beat rate

5. Classification Filters:
   - Sector: 11 GICS sectors
   - Industry: Detailed industries
   - Country: US, International
   - Exchange: NYSE, NASDAQ, etc.
   - Market cap: Mega, Large, Mid, Small, Micro

Backend Implementation (FastAPI):

```python
# routers/screener.py
from fastapi import APIRouter, Depends, Query
from typing import List, Optional
from pydantic import BaseModel

router = APIRouter(prefix="/api/v1/screener", tags=["screener"])

class ScreenerFilter(BaseModel):
    field: str
    operator: str  # 'gt', 'lt', 'gte', 'lte', 'eq', 'between', 'in'
    value: Any

class ScreenerRequest(BaseModel):
    filters: List[ScreenerFilter]
    sort_by: Optional[str] = "ic_score"
    sort_order: Optional[str] = "desc"
    limit: Optional[int] = 100
    offset: Optional[int] = 0

class ScreenerResult(BaseModel):
    ticker: str
    name: str
    sector: str
    market_cap: int
    ic_score: float
    price: float
    # ... other key fields ...

@router.post("/run")
async def run_screener(
    request: ScreenerRequest,
    db: Session = Depends(get_db)
) -> Dict:
    """Run stock screener with filters."""
    # Build SQL query dynamically based on filters
    query = build_screener_query(request.filters)

    # Execute query
    results = db.execute(query).fetchall()

    # Format results
    stocks = [ScreenerResult.from_orm(r) for r in results]

    return {
        "success": True,
        "data": {
            "stocks": stocks,
            "total_count": len(stocks),
            "filters_applied": len(request.filters)
        }
    }

def build_screener_query(filters: List[ScreenerFilter]) -> str:
    """Build SQL query from filters."""
    base_query = """
        SELECT
            c.ticker,
            c.name,
            c.sector,
            c.market_cap,
            s.overall_score as ic_score,
            p.close as price,
            -- ... other fields ...
        FROM companies c
        LEFT JOIN ic_scores s ON c.ticker = s.ticker
            AND s.date = (SELECT MAX(date) FROM ic_scores WHERE ticker = c.ticker)
        LEFT JOIN stock_prices p ON c.ticker = p.ticker
            AND p.time = (SELECT MAX(time) FROM stock_prices WHERE ticker = c.ticker)
        WHERE c.is_active = true
    """

    # Add filter conditions
    conditions = []
    for f in filters:
        condition = build_filter_condition(f)
        conditions.append(condition)

    if conditions:
        base_query += " AND " + " AND ".join(conditions)

    return base_query

def build_filter_condition(filter: ScreenerFilter) -> str:
    """Convert filter to SQL condition."""
    field_map = {
        'ic_score': 's.overall_score',
        'market_cap': 'c.market_cap',
        'pe_ratio': 'f.pe_ratio',
        # ... map all fields ...
    }

    field = field_map.get(filter.field, filter.field)

    if filter.operator == 'gte':
        return f"{field} >= {filter.value}"
    elif filter.operator == 'lte':
        return f"{field} <= {filter.value}"
    elif filter.operator == 'between':
        return f"{field} BETWEEN {filter.value[0]} AND {filter.value[1]}"
    elif filter.operator == 'in':
        values = ",".join([f"'{v}'" for v in filter.value])
        return f"{field} IN ({values})"
    # ... handle other operators ...
```

Frontend Implementation (Next.js):

```tsx
// app/screener/page.tsx
'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import FilterPanel from '@/components/screener/FilterPanel'
import ResultsTable from '@/components/screener/ResultsTable'
import PresetScreens from '@/components/screener/PresetScreens'

export default function ScreenerPage() {
  const [filters, setFilters] = useState<Filter[]>([])
  const [sortBy, setSortBy] = useState('ic_score')
  const [sortOrder, setSortOrder] = useState('desc')

  const { data, isLoading } = useQuery({
    queryKey: ['screener', filters, sortBy, sortOrder],
    queryFn: () => runScreener({ filters, sortBy, sortOrder }),
    enabled: filters.length > 0,
  })

  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-8">Stock Screener</h1>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Left sidebar - Filters */}
        <div className="lg:col-span-1">
          <PresetScreens onSelect={setFilters} />

          <FilterPanel
            filters={filters}
            onFiltersChange={setFilters}
          />
        </div>

        {/* Main content - Results */}
        <div className="lg:col-span-3">
          {filters.length === 0 ? (
            <EmptyState />
          ) : (
            <ResultsTable
              data={data?.stocks}
              isLoading={isLoading}
              sortBy={sortBy}
              sortOrder={sortOrder}
              onSort={(field, order) => {
                setSortBy(field)
                setSortOrder(order)
              }}
            />
          )}
        </div>
      </div>
    </div>
  )
}

// components/screener/FilterPanel.tsx
export default function FilterPanel({ filters, onFiltersChange }) {
  const [activeCategory, setActiveCategory] = useState('ic_score')

  const filterCategories = {
    ic_score: 'IC Score',
    fundamentals: 'Fundamentals',
    technical: 'Technical',
    quality: 'Quality',
    classification: 'Classification',
  }

  return (
    <div className="bg-white rounded-lg shadow p-4">
      <h2 className="text-lg font-semibold mb-4">Filters</h2>

      {/* Category tabs */}
      <div className="flex flex-col space-y-2 mb-4">
        {Object.entries(filterCategories).map(([key, label]) => (
          <button
            key={key}
            onClick={() => setActiveCategory(key)}
            className={`text-left px-3 py-2 rounded ${
              activeCategory === key
                ? 'bg-blue-100 text-blue-700'
                : 'hover:bg-gray-100'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {/* Active filters */}
      {filters.length > 0 && (
        <div className="mb-4">
          <h3 className="text-sm font-medium mb-2">Active Filters ({filters.length})</h3>
          <div className="space-y-1">
            {filters.map((filter, i) => (
              <FilterChip
                key={i}
                filter={filter}
                onRemove={() => {
                  const newFilters = filters.filter((_, idx) => idx !== i)
                  onFiltersChange(newFilters)
                }}
              />
            ))}
          </div>
        </div>
      )}

      {/* Filter inputs based on active category */}
      <div className="space-y-4">
        {activeCategory === 'ic_score' && (
          <ICScoreFilters filters={filters} onChange={onFiltersChange} />
        )}
        {activeCategory === 'fundamentals' && (
          <FundamentalFilters filters={filters} onChange={onFiltersChange} />
        )}
        {/* ... other filter categories ... */}
      </div>
    </div>
  )
}

// components/screener/ICScoreFilters.tsx
function ICScoreFilters({ filters, onChange }) {
  return (
    <div className="space-y-4">
      <div>
        <label className="block text-sm font-medium mb-2">
          IC Score Range
        </label>
        <div className="flex gap-2">
          <input
            type="number"
            placeholder="Min"
            min="1"
            max="100"
            className="flex-1 px-3 py-2 border rounded"
            onChange={(e) => {
              const value = parseInt(e.target.value)
              if (value) {
                onChange([...filters, {
                  field: 'ic_score',
                  operator: 'gte',
                  value: value
                }])
              }
            }}
          />
          <input
            type="number"
            placeholder="Max"
            min="1"
            max="100"
            className="flex-1 px-3 py-2 border rounded"
            onChange={(e) => {
              const value = parseInt(e.target.value)
              if (value) {
                onChange([...filters, {
                  field: 'ic_score',
                  operator: 'lte',
                  value: value
                }])
              }
            }}
          />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium mb-2">
          Rating
        </label>
        <select
          className="w-full px-3 py-2 border rounded"
          onChange={(e) => {
            if (e.target.value) {
              onChange([...filters, {
                field: 'rating',
                operator: 'eq',
                value: e.target.value
              }])
            }
          }}
        >
          <option value="">Any</option>
          <option value="Strong Buy">Strong Buy</option>
          <option value="Buy">Buy</option>
          <option value="Hold">Hold</option>
          <option value="Sell">Sell</option>
        </select>
      </div>

      {/* Individual factor filters */}
      {factors.map(factor => (
        <div key={factor}>
          <label className="block text-sm font-medium mb-2">
            {factor} Score
          </label>
          <input
            type="range"
            min="1"
            max="100"
            className="w-full"
            onChange={(e) => {
              const value = parseInt(e.target.value)
              onChange([...filters, {
                field: `${factor}_score`,
                operator: 'gte',
                value: value
              }])
            }}
          />
        </div>
      ))}
    </div>
  )
}
```

Pre-built Screen Templates:
1. Value Stocks: Low P/E (<20), Low P/B (<3), High IC Score (>70)
2. Growth Stocks: Revenue growth >20%, EPS growth >15%, IC Score >65
3. Dividend Aristocrats: Yield >2%, 25+ years increases, IC Score >60
4. Momentum Leaders: 6M return >20%, RSI 50-70, IC Score >70
5. High Quality: IC Score >80, ROE >20%, Debt/Equity <0.5
6. Insider Buying: Net insider buying >$1M, IC Score >60
7. Smart Money: Institutional ownership increase, notable investors
8. Beaten Down Quality: Price off 52W high >20%, IC Score >75

Please implement a production-ready stock screener with fast performance,
intuitive UI, and comprehensive filtering capabilities.
```

---

## Prompt 7: Real-Time Price Updates with WebSocket

```
I need you to implement real-time price updates for InvestorCenter.ai using WebSocket
connections.

Requirements:
1. WebSocket server in Python (FastAPI WebSockets or standalone)
2. Subscribe to price feed (Polygon.io or similar)
3. Broadcast price updates to connected clients
4. React hook for consuming WebSocket data
5. Automatic reconnection on disconnect
6. Efficient bandwidth usage (only send changes)
7. Market hours awareness (9:30 AM - 4:00 PM ET)

Backend Implementation:

```python
# ws_server.py
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from typing import Set, Dict
import asyncio
import json
from datetime import datetime, time

app = FastAPI()

class ConnectionManager:
    def __init__(self):
        # Map of ticker -> set of WebSocket connections
        self.active_connections: Dict[str, Set[WebSocket]] = {}
        # Store latest prices to send to new subscribers
        self.latest_prices: Dict[str, dict] = {}

    async def connect(self, websocket: WebSocket, ticker: str):
        """Accept new WebSocket connection and subscribe to ticker."""
        await websocket.accept()
        if ticker not in self.active_connections:
            self.active_connections[ticker] = set()
        self.active_connections[ticker].add(websocket)

        # Send latest price immediately
        if ticker in self.latest_prices:
            await websocket.send_json(self.latest_prices[ticker])

    def disconnect(self, websocket: WebSocket, ticker: str):
        """Remove WebSocket connection."""
        if ticker in self.active_connections:
            self.active_connections[ticker].discard(websocket)
            if not self.active_connections[ticker]:
                del self.active_connections[ticker]

    async def broadcast(self, ticker: str, message: dict):
        """Broadcast price update to all subscribers of a ticker."""
        # Store latest price
        self.latest_prices[ticker] = message

        # Send to all connected clients
        if ticker in self.active_connections:
            dead_connections = set()
            for connection in self.active_connections[ticker]:
                try:
                    await connection.send_json(message)
                except:
                    dead_connections.add(connection)

            # Clean up dead connections
            for connection in dead_connections:
                self.disconnect(connection, ticker)

manager = ConnectionManager()

@app.websocket("/ws/prices/{ticker}")
async def price_websocket(websocket: WebSocket, ticker: str):
    """WebSocket endpoint for real-time price updates."""
    ticker = ticker.upper()
    await manager.connect(websocket, ticker)

    try:
        # Keep connection alive and handle client messages
        while True:
            # Wait for client messages (ping/pong, subscribe to more tickers)
            data = await websocket.receive_text()
            # Echo back or handle special commands
            if data == "ping":
                await websocket.send_text("pong")

    except WebSocketDisconnect:
        manager.disconnect(websocket, ticker)

# Background task to fetch and broadcast prices
async def price_updater():
    """Background task that fetches prices and broadcasts updates."""
    import polygon  # or your price data source

    client = polygon.StocksClient(API_KEY)

    while True:
        # Only run during market hours (+ extended hours if needed)
        if is_market_hours():
            # Get all tickers that have active subscribers
            active_tickers = list(manager.active_connections.keys())

            if active_tickers:
                # Fetch latest prices (batch request)
                prices = await fetch_latest_prices(active_tickers)

                # Broadcast updates
                for ticker, price_data in prices.items():
                    await manager.broadcast(ticker, {
                        "type": "price_update",
                        "ticker": ticker,
                        "price": price_data["price"],
                        "change": price_data["change"],
                        "change_percent": price_data["change_percent"],
                        "volume": price_data["volume"],
                        "timestamp": datetime.now().isoformat()
                    })

            # Wait before next update (1 second during market hours)
            await asyncio.sleep(1)
        else:
            # Off hours: slower updates (1 minute)
            await asyncio.sleep(60)

def is_market_hours() -> bool:
    """Check if current time is during market hours."""
    now = datetime.now()
    if now.weekday() >= 5:  # Weekend
        return False

    market_open = time(9, 30)
    market_close = time(16, 0)
    current_time = now.time()

    return market_open <= current_time <= market_close

async def fetch_latest_prices(tickers: list) -> dict:
    """Fetch latest prices from data source."""
    # Implement actual price fetching
    # This would call Polygon.io or similar API
    pass

# Start background task on app startup
@app.on_event("startup")
async def startup_event():
    asyncio.create_task(price_updater())
```

Frontend Implementation (React Hook):

```typescript
// hooks/useRealtimePrice.ts
import { useEffect, useState, useCallback, useRef } from 'react'

interface PriceUpdate {
  ticker: string
  price: number
  change: number
  change_percent: number
  volume: number
  timestamp: string
}

export function useRealtimePrice(ticker: string) {
  const [price, setPrice] = useState<PriceUpdate | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout>()
  const reconnectAttemptsRef = useRef(0)

  const connect = useCallback(() => {
    if (!ticker) return

    const wsUrl = `${process.env.NEXT_PUBLIC_WS_URL}/ws/prices/${ticker}`
    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      console.log(`WebSocket connected for ${ticker}`)
      setIsConnected(true)
      reconnectAttemptsRef.current = 0
    }

    ws.onmessage = (event) => {
      const data: PriceUpdate = JSON.parse(event.data)
      setPrice(data)
    }

    ws.onerror = (error) => {
      console.error(`WebSocket error for ${ticker}:`, error)
    }

    ws.onclose = () => {
      console.log(`WebSocket disconnected for ${ticker}`)
      setIsConnected(false)

      // Attempt to reconnect with exponential backoff
      const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000)
      reconnectTimeoutRef.current = setTimeout(() => {
        reconnectAttemptsRef.current += 1
        connect()
      }, delay)
    }

    wsRef.current = ws

    // Ping every 30 seconds to keep connection alive
    const pingInterval = setInterval(() => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send('ping')
      }
    }, 30000)

    return () => {
      clearInterval(pingInterval)
    }
  }, [ticker])

  useEffect(() => {
    const cleanup = connect()

    return () => {
      cleanup?.()
      if (wsRef.current) {
        wsRef.current.close()
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
    }
  }, [connect])

  return { price, isConnected }
}

// Usage in component
function StockPrice({ ticker }: { ticker: string }) {
  const { price, isConnected } = useRealtimePrice(ticker)

  if (!price) {
    return <Skeleton className="h-8 w-32" />
  }

  return (
    <div className="flex items-center gap-2">
      <span className="text-2xl font-bold">
        ${price.price.toFixed(2)}
      </span>
      <span className={`text-sm ${price.change >= 0 ? 'text-green-600' : 'text-red-600'}`}>
        {price.change >= 0 ? '+' : ''}{price.change.toFixed(2)}
        ({price.change_percent.toFixed(2)}%)
      </span>
      {isConnected && (
        <span className="text-xs text-gray-500">● Live</span>
      )}
    </div>
  )
}
```

Performance Optimizations:
1. Only broadcast when price changes (don't send duplicate data)
2. Batch updates if multiple tickers change simultaneously
3. Throttle updates to max 1 per second per ticker
4. Use binary protocol (MessagePack) instead of JSON for lower bandwidth
5. Implement Redis pub/sub for horizontal scaling

Please implement production-ready WebSocket server and client with automatic
reconnection, efficient updates, and comprehensive error handling.
```

---

(Additional Phase 1 prompts would continue for remaining features:
- News aggregation & sentiment analysis
- Insider trading & institutional data
- Watchlist & portfolio management
- Dividend features
- Mobile responsive design
- Authentication & user management
- etc.)

**End of Implementation Prompts - Phase 1**
