"""FastAPI service for IC Score API.

This service provides REST API endpoints for accessing IC Scores and related data.
"""

import logging
import datetime as _dt
from datetime import datetime, timedelta, date
from typing import List, Optional, Dict, Any
from enum import Enum

from fastapi import FastAPI, HTTPException, Query, Depends, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, ConfigDict, Field
from sqlalchemy import text, select, desc
from sqlalchemy.ext.asyncio import AsyncSession

import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from database.database import get_database, get_session
from models import ICScore, Financial, TechnicalIndicator, NewsArticle
from backtesting.backtester import ICScoreBacktester, BacktestConfig as BacktesterConfig, RebalanceFrequency

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Create FastAPI app
app = FastAPI(
    title="InvestorCenter IC Score API",
    description="API for accessing InvestorCenter proprietary IC Scores",
    version="1.0.0"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=[
        "https://investorcenter.ai",
        "https://www.investorcenter.ai",
        "http://localhost:3000",  # Development only
    ],
    allow_credentials=True,
    allow_methods=["GET", "POST", "OPTIONS"],
    allow_headers=["*"],
)


# Pydantic models for API responses
class ICScoreResponse(BaseModel):
    """IC Score response model."""
    ticker: str
    date: date
    overall_score: float
    rating: str
    value_score: Optional[float] = None
    growth_score: Optional[float] = None
    profitability_score: Optional[float] = None
    financial_health_score: Optional[float] = None
    momentum_score: Optional[float] = None
    analyst_consensus_score: Optional[float] = None
    insider_activity_score: Optional[float] = None
    institutional_score: Optional[float] = None
    news_sentiment_score: Optional[float] = None
    technical_score: Optional[float] = None
    sector_percentile: Optional[float] = None
    confidence_level: str
    data_completeness: float

    class Config:
        from_attributes = True


class ICScoreHistoryResponse(BaseModel):
    """IC Score history response model."""
    ticker: str
    scores: List[ICScoreResponse]


class TopStocksResponse(BaseModel):
    """Top stocks response model."""
    stocks: List[ICScoreResponse]
    count: int


class HealthResponse(BaseModel):
    """Health check response model."""
    status: str
    timestamp: datetime
    database_connected: bool
    database_info: dict


class FinancialMetricsResponse(BaseModel):
    """Financial metrics response model for frontend display."""
    ticker: str
    period_end_date: Optional[date] = None
    fiscal_year: Optional[int] = None
    fiscal_quarter: Optional[int] = None

    # Profitability Metrics
    gross_margin: Optional[float] = None
    operating_margin: Optional[float] = None
    net_margin: Optional[float] = None
    roe: Optional[float] = None
    roa: Optional[float] = None

    # Financial Health
    debt_to_equity: Optional[float] = None
    current_ratio: Optional[float] = None
    quick_ratio: Optional[float] = None

    # Valuation
    pe_ratio: Optional[float] = None
    pb_ratio: Optional[float] = None
    ps_ratio: Optional[float] = None

    # Growth (calculated from YoY comparison)
    revenue_growth_yoy: Optional[float] = None
    earnings_growth_yoy: Optional[float] = None

    # Market Data
    shares_outstanding: Optional[int] = None

    # Metadata
    statement_type: Optional[str] = None
    data_as_of: Optional[date] = None

    class Config:
        from_attributes = True


class RiskMetricsResponse(BaseModel):
    """Risk metrics response model for frontend display."""
    ticker: str
    period: str = "1Y"
    calculation_date: Optional[date] = None

    # Risk Metrics
    beta: Optional[float] = None
    alpha: Optional[float] = None
    sharpe_ratio: Optional[float] = None
    sortino_ratio: Optional[float] = None
    volatility: Optional[float] = None  # std_dev
    max_drawdown: Optional[float] = None
    var_95: Optional[float] = None  # var_5

    # Returns
    annualized_return: Optional[float] = None
    downside_deviation: Optional[float] = None

    # Data quality
    data_points: Optional[int] = None

    class Config:
        from_attributes = True


class TechnicalIndicatorsResponse(BaseModel):
    """Technical indicators response model for frontend display."""
    model_config = ConfigDict(from_attributes=True)

    ticker: str
    date: Optional[_dt.date] = None

    # Moving Averages
    sma_20: Optional[float] = None
    sma_50: Optional[float] = None
    sma_200: Optional[float] = None
    ema_12: Optional[float] = None
    ema_26: Optional[float] = None

    # Oscillators
    rsi_14: Optional[float] = None
    macd: Optional[float] = None
    macd_signal: Optional[float] = None
    macd_histogram: Optional[float] = None

    # Bollinger Bands
    bb_upper: Optional[float] = None
    bb_middle: Optional[float] = None
    bb_lower: Optional[float] = None

    # Other
    atr_14: Optional[float] = None
    adx_14: Optional[float] = None
    stoch_k: Optional[float] = None
    stoch_d: Optional[float] = None

    # Current price for context
    close_price: Optional[float] = None


# API Endpoints

@app.get("/", include_in_schema=False)
async def root():
    """Root endpoint."""
    return {
        "service": "IC Score API",
        "version": "2.1.0",
        "endpoints": {
            "health": "/health",
            "score": "/api/scores/{ticker}",
            "history": "/api/scores/{ticker}/history",
            "top": "/api/scores/top",
            "screener": "/api/scores/screener",
            "financials_annual": "/api/financials/{ticker}/annual",
            "financials_ttm": "/api/financials/{ticker}/ttm",
            "metrics": "/api/metrics/{ticker}",
            "risk": "/api/risk/{ticker}",
            "technical": "/api/technical/{ticker}",
            "news": "/api/news/{ticker}",
            "backtest": "/api/backtest",
            "backtest_config": "/api/backtest/config/default",
            "backtest_jobs": "/api/backtest/jobs",
            "backtest_job_status": "/api/backtest/jobs/{job_id}",
            "backtest_job_result": "/api/backtest/jobs/{job_id}/result"
        }
    }


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint.

    Returns:
        Health status and database connection info.
    """
    db = get_database()
    health_info = await db.health_check()

    return {
        "status": health_info.get("status", "unknown"),
        "timestamp": datetime.now(),
        "database_connected": health_info.get("connected", False),
        "database_info": health_info
    }


@app.get("/api/scores/{ticker}", response_model=ICScoreResponse)
async def get_ic_score(
    ticker: str,
    session: AsyncSession = Depends(get_session)
):
    """Get latest IC Score for a ticker.

    Args:
        ticker: Stock ticker symbol.

    Returns:
        Latest IC Score data.

    Raises:
        HTTPException: If ticker not found.
    """
    ticker = ticker.upper()

    query = text("""
        SELECT *
        FROM ic_scores
        WHERE ticker = :ticker
        ORDER BY date DESC
        LIMIT 1
    """)

    result = await session.execute(query, {"ticker": ticker})
    row = result.fetchone()

    if not row:
        raise HTTPException(status_code=404, detail=f"IC Score not found for ticker {ticker}")

    # Convert to dict
    score_data = row._asdict()

    return ICScoreResponse(**score_data)


@app.get("/api/scores/{ticker}/history", response_model=ICScoreHistoryResponse)
async def get_score_history(
    ticker: str,
    days: int = Query(90, ge=1, le=365, description="Number of days of history"),
    session: AsyncSession = Depends(get_session)
):
    """Get historical IC Scores for a ticker.

    Args:
        ticker: Stock ticker symbol.
        days: Number of days of history (1-365).

    Returns:
        Historical IC Score data.
    """
    ticker = ticker.upper()

    query = text("""
        SELECT *
        FROM ic_scores
        WHERE ticker = :ticker
          AND date >= :start_date
        ORDER BY date DESC
    """)

    start_date = date.today() - timedelta(days=days)
    result = await session.execute(query, {"ticker": ticker, "start_date": start_date})
    rows = result.fetchall()

    if not rows:
        raise HTTPException(status_code=404, detail=f"No history found for ticker {ticker}")

    scores = [ICScoreResponse(**row._asdict()) for row in rows]

    return ICScoreHistoryResponse(ticker=ticker, scores=scores)


@app.get("/api/scores/top", response_model=TopStocksResponse)
async def get_top_scores(
    limit: int = Query(50, ge=1, le=500, description="Number of stocks to return"),
    sector: Optional[str] = Query(None, description="Filter by sector"),
    min_score: Optional[float] = Query(None, ge=0, le=100, description="Minimum IC Score"),
    min_confidence: Optional[str] = Query(None, description="Minimum confidence level (High, Medium, Low)"),
    session: AsyncSession = Depends(get_session)
):
    """Get top-rated stocks.

    Args:
        limit: Number of stocks to return (1-500).
        sector: Filter by sector (optional).
        min_score: Minimum overall score (optional).
        min_confidence: Minimum confidence level (optional).

    Returns:
        List of top-rated stocks.
    """
    where_clauses = []
    params = {"limit": limit}

    # Get latest scores (use a window function)
    base_query = """
        WITH latest_scores AS (
            SELECT DISTINCT ON (ticker)
                *
            FROM ic_scores
            ORDER BY ticker, date DESC
        )
        SELECT ls.*
        FROM latest_scores ls
    """

    if sector:
        base_query += " JOIN stocks s ON ls.ticker = s.ticker"
        where_clauses.append("s.sector = :sector")
        params["sector"] = sector

    if min_score is not None:
        where_clauses.append("ls.overall_score >= :min_score")
        params["min_score"] = min_score

    if min_confidence:
        where_clauses.append("ls.confidence_level = :min_confidence")
        params["min_confidence"] = min_confidence

    if where_clauses:
        base_query += " WHERE " + " AND ".join(where_clauses)

    base_query += " ORDER BY ls.overall_score DESC LIMIT :limit"

    query = text(base_query)
    result = await session.execute(query, params)
    rows = result.fetchall()

    stocks = [ICScoreResponse(**row._asdict()) for row in rows]

    return TopStocksResponse(stocks=stocks, count=len(stocks))


@app.get("/api/scores/screener", response_model=TopStocksResponse)
async def screener(
    min_value: Optional[float] = Query(None, ge=0, le=100, description="Minimum value score"),
    min_growth: Optional[float] = Query(None, ge=0, le=100, description="Minimum growth score"),
    min_profitability: Optional[float] = Query(None, ge=0, le=100, description="Minimum profitability score"),
    min_financial_health: Optional[float] = Query(None, ge=0, le=100, description="Minimum financial health score"),
    min_momentum: Optional[float] = Query(None, ge=0, le=100, description="Minimum momentum score"),
    min_technical: Optional[float] = Query(None, ge=0, le=100, description="Minimum technical score"),
    min_overall: Optional[float] = Query(None, ge=0, le=100, description="Minimum overall score"),
    sector: Optional[str] = Query(None, description="Filter by sector"),
    limit: int = Query(100, ge=1, le=1000, description="Number of results"),
    session: AsyncSession = Depends(get_session)
):
    """Advanced stock screener with factor-level filtering.

    Args:
        min_value: Minimum value score (0-100).
        min_growth: Minimum growth score (0-100).
        min_profitability: Minimum profitability score (0-100).
        min_financial_health: Minimum financial health score (0-100).
        min_momentum: Minimum momentum score (0-100).
        min_technical: Minimum technical score (0-100).
        min_overall: Minimum overall score (0-100).
        sector: Filter by sector.
        limit: Number of results (1-1000).

    Returns:
        List of stocks matching criteria.
    """
    where_clauses = []
    params = {"limit": limit}

    # Build query
    base_query = """
        WITH latest_scores AS (
            SELECT DISTINCT ON (ticker)
                *
            FROM ic_scores
            ORDER BY ticker, date DESC
        )
        SELECT ls.*
        FROM latest_scores ls
    """

    if sector:
        base_query += " JOIN stocks s ON ls.ticker = s.ticker"
        where_clauses.append("s.sector = :sector")
        params["sector"] = sector

    if min_value is not None:
        where_clauses.append("ls.value_score >= :min_value")
        params["min_value"] = min_value

    if min_growth is not None:
        where_clauses.append("ls.growth_score >= :min_growth")
        params["min_growth"] = min_growth

    if min_profitability is not None:
        where_clauses.append("ls.profitability_score >= :min_profitability")
        params["min_profitability"] = min_profitability

    if min_financial_health is not None:
        where_clauses.append("ls.financial_health_score >= :min_financial_health")
        params["min_financial_health"] = min_financial_health

    if min_momentum is not None:
        where_clauses.append("ls.momentum_score >= :min_momentum")
        params["min_momentum"] = min_momentum

    if min_technical is not None:
        where_clauses.append("ls.technical_score >= :min_technical")
        params["min_technical"] = min_technical

    if min_overall is not None:
        where_clauses.append("ls.overall_score >= :min_overall")
        params["min_overall"] = min_overall

    if where_clauses:
        base_query += " WHERE " + " AND ".join(where_clauses)

    base_query += " ORDER BY ls.overall_score DESC LIMIT :limit"

    query = text(base_query)
    result = await session.execute(query, params)
    rows = result.fetchall()

    stocks = [ICScoreResponse(**row._asdict()) for row in rows]

    return TopStocksResponse(stocks=stocks, count=len(stocks))


# ============================================================================
# Financial Statements Endpoints (Annual/TTM)
# ============================================================================

class FinancialStatementItem(BaseModel):
    """Single financial statement period."""
    fiscal_year: int
    fiscal_quarter: Optional[int] = None
    period_end_date: date
    filing_date: Optional[date] = None

    # Income Statement
    revenue: Optional[int] = None
    cost_of_revenue: Optional[int] = None
    gross_profit: Optional[int] = None
    operating_expenses: Optional[int] = None
    operating_income: Optional[int] = None
    net_income: Optional[int] = None
    eps_basic: Optional[float] = None
    eps_diluted: Optional[float] = None
    shares_outstanding: Optional[int] = None

    # Balance Sheet
    total_assets: Optional[int] = None
    total_liabilities: Optional[int] = None
    shareholders_equity: Optional[int] = None
    cash_and_equivalents: Optional[int] = None
    short_term_debt: Optional[int] = None
    long_term_debt: Optional[int] = None

    # Cash Flow
    operating_cash_flow: Optional[int] = None
    investing_cash_flow: Optional[int] = None
    financing_cash_flow: Optional[int] = None
    free_cash_flow: Optional[int] = None
    capex: Optional[int] = None

    # Calculated Ratios
    gross_margin: Optional[float] = None
    operating_margin: Optional[float] = None
    net_margin: Optional[float] = None
    roe: Optional[float] = None
    roa: Optional[float] = None
    current_ratio: Optional[float] = None
    debt_to_equity: Optional[float] = None

    # YoY Changes (optional)
    yoy_change: Optional[dict] = None

    class Config:
        from_attributes = True


class FinancialStatementsResponse(BaseModel):
    """Financial statements response with multiple periods."""
    ticker: str
    statement_type: str  # "income", "balance", "cashflow"
    timeframe: str  # "annual", "ttm"
    periods: List[FinancialStatementItem]
    metadata: dict = {}


@app.get("/api/financials/{ticker}/annual")
async def get_annual_financials(
    ticker: str,
    statement_type: str = Query("income", description="Statement type: income, balance, cashflow"),
    limit: int = Query(8, ge=1, le=40, description="Number of periods to return"),
    session: AsyncSession = Depends(get_session)
):
    """Get annual financial statements for a ticker.

    Returns annual (10-K) financial data from SEC filings.

    Args:
        ticker: Stock ticker symbol.
        statement_type: Type of statement (income, balance, cashflow).
        limit: Number of periods to return.

    Returns:
        List of annual financial periods.
    """
    ticker = ticker.upper()

    # Query annual financials (fiscal_quarter IS NULL = annual 10-K data)
    query = text("""
        SELECT
            f.*,
            py.revenue as prior_revenue,
            py.net_income as prior_net_income,
            py.eps_diluted as prior_eps_diluted,
            py.operating_income as prior_operating_income
        FROM financials f
        LEFT JOIN financials py ON f.ticker = py.ticker
            AND py.fiscal_year = f.fiscal_year - 1
            AND py.fiscal_quarter IS NULL
        WHERE f.ticker = :ticker
            AND f.fiscal_quarter IS NULL
        ORDER BY f.period_end_date DESC
        LIMIT :limit
    """)

    result = await session.execute(query, {"ticker": ticker, "limit": limit})
    rows = result.fetchall()

    if not rows:
        raise HTTPException(
            status_code=404,
            detail=f"No annual financial data found for ticker {ticker}"
        )

    periods = []
    for row in rows:
        row_dict = row._asdict()

        # Calculate YoY changes
        yoy_change = {}
        if row_dict.get('prior_revenue') and row_dict.get('revenue'):
            prior_rev = float(row_dict['prior_revenue'])
            curr_rev = float(row_dict['revenue'])
            if prior_rev != 0:
                yoy_change['revenue'] = (curr_rev - prior_rev) / abs(prior_rev)

        if row_dict.get('prior_net_income') and row_dict.get('net_income'):
            prior_ni = float(row_dict['prior_net_income'])
            curr_ni = float(row_dict['net_income'])
            if prior_ni != 0:
                yoy_change['net_income'] = (curr_ni - prior_ni) / abs(prior_ni)

        if row_dict.get('prior_eps_diluted') and row_dict.get('eps_diluted'):
            prior_eps = float(row_dict['prior_eps_diluted'])
            curr_eps = float(row_dict['eps_diluted'])
            if prior_eps != 0:
                yoy_change['eps_diluted'] = (curr_eps - prior_eps) / abs(prior_eps)

        if row_dict.get('prior_operating_income') and row_dict.get('operating_income'):
            prior_oi = float(row_dict['prior_operating_income'])
            curr_oi = float(row_dict['operating_income'])
            if prior_oi != 0:
                yoy_change['operating_income'] = (curr_oi - prior_oi) / abs(prior_oi)

        periods.append(FinancialStatementItem(
            fiscal_year=row_dict['fiscal_year'],
            fiscal_quarter=None,
            period_end_date=row_dict['period_end_date'],
            filing_date=row_dict.get('filing_date'),
            revenue=row_dict.get('revenue'),
            cost_of_revenue=row_dict.get('cost_of_revenue'),
            gross_profit=row_dict.get('gross_profit'),
            operating_expenses=row_dict.get('operating_expenses'),
            operating_income=row_dict.get('operating_income'),
            net_income=row_dict.get('net_income'),
            eps_basic=float(row_dict['eps_basic']) if row_dict.get('eps_basic') else None,
            eps_diluted=float(row_dict['eps_diluted']) if row_dict.get('eps_diluted') else None,
            shares_outstanding=row_dict.get('shares_outstanding'),
            total_assets=row_dict.get('total_assets'),
            total_liabilities=row_dict.get('total_liabilities'),
            shareholders_equity=row_dict.get('shareholders_equity'),
            cash_and_equivalents=row_dict.get('cash_and_equivalents'),
            short_term_debt=row_dict.get('short_term_debt'),
            long_term_debt=row_dict.get('long_term_debt'),
            operating_cash_flow=row_dict.get('operating_cash_flow'),
            investing_cash_flow=row_dict.get('investing_cash_flow'),
            financing_cash_flow=row_dict.get('financing_cash_flow'),
            free_cash_flow=row_dict.get('free_cash_flow'),
            capex=row_dict.get('capex'),
            gross_margin=float(row_dict['gross_margin']) if row_dict.get('gross_margin') else None,
            operating_margin=float(row_dict['operating_margin']) if row_dict.get('operating_margin') else None,
            net_margin=float(row_dict['net_margin']) if row_dict.get('net_margin') else None,
            roe=float(row_dict['roe']) if row_dict.get('roe') else None,
            roa=float(row_dict['roa']) if row_dict.get('roa') else None,
            current_ratio=float(row_dict['current_ratio']) if row_dict.get('current_ratio') else None,
            debt_to_equity=float(row_dict['debt_to_equity']) if row_dict.get('debt_to_equity') else None,
            yoy_change=yoy_change if yoy_change else None
        ))

    # Get company name for metadata
    meta_query = text("SELECT name FROM tickers WHERE UPPER(symbol) = UPPER(:ticker) LIMIT 1")
    meta_result = await session.execute(meta_query, {"ticker": ticker})
    meta_row = meta_result.fetchone()

    return {
        "ticker": ticker,
        "statement_type": statement_type,
        "timeframe": "annual",
        "periods": [p.model_dump() for p in periods],
        "metadata": {
            "company_name": meta_row.name if meta_row else ticker
        }
    }


@app.get("/api/financials/{ticker}/ttm")
async def get_ttm_financials(
    ticker: str,
    statement_type: str = Query("income", description="Statement type: income, balance, cashflow"),
    limit: int = Query(8, ge=1, le=40, description="Number of periods to return"),
    session: AsyncSession = Depends(get_session)
):
    """Get TTM (Trailing Twelve Months) financial data for a ticker.

    Returns TTM financial data calculated by summing the last 4 quarters.

    Args:
        ticker: Stock ticker symbol.
        statement_type: Type of statement (income, balance, cashflow).
        limit: Number of periods to return.

    Returns:
        List of TTM financial periods.
    """
    ticker = ticker.upper()

    # Query TTM financials
    query = text("""
        SELECT
            t.*,
            pt.revenue as prior_revenue,
            pt.net_income as prior_net_income,
            pt.eps_diluted as prior_eps_diluted,
            pt.operating_income as prior_operating_income
        FROM ttm_financials t
        LEFT JOIN ttm_financials pt ON t.ticker = pt.ticker
            AND pt.ttm_period_end <= t.ttm_period_end - INTERVAL '11 months'
            AND pt.ttm_period_end >= t.ttm_period_end - INTERVAL '13 months'
        WHERE t.ticker = :ticker
        ORDER BY t.calculation_date DESC
        LIMIT :limit
    """)

    result = await session.execute(query, {"ticker": ticker, "limit": limit})
    rows = result.fetchall()

    if not rows:
        raise HTTPException(
            status_code=404,
            detail=f"No TTM financial data found for ticker {ticker}"
        )

    periods = []
    for row in rows:
        row_dict = row._asdict()

        # Calculate YoY changes
        yoy_change = {}
        if row_dict.get('prior_revenue') and row_dict.get('revenue'):
            prior_rev = float(row_dict['prior_revenue'])
            curr_rev = float(row_dict['revenue'])
            if prior_rev != 0:
                yoy_change['revenue'] = (curr_rev - prior_rev) / abs(prior_rev)

        if row_dict.get('prior_net_income') and row_dict.get('net_income'):
            prior_ni = float(row_dict['prior_net_income'])
            curr_ni = float(row_dict['net_income'])
            if prior_ni != 0:
                yoy_change['net_income'] = (curr_ni - prior_ni) / abs(prior_ni)

        if row_dict.get('prior_eps_diluted') and row_dict.get('eps_diluted'):
            prior_eps = float(row_dict['prior_eps_diluted'])
            curr_eps = float(row_dict['eps_diluted'])
            if prior_eps != 0:
                yoy_change['eps_diluted'] = (curr_eps - prior_eps) / abs(prior_eps)

        if row_dict.get('prior_operating_income') and row_dict.get('operating_income'):
            prior_oi = float(row_dict['prior_operating_income'])
            curr_oi = float(row_dict['operating_income'])
            if prior_oi != 0:
                yoy_change['operating_income'] = (curr_oi - prior_oi) / abs(prior_oi)

        # Extract fiscal year from the TTM period end date
        fiscal_year = row_dict['ttm_period_end'].year

        # Calculate margins from TTM data if not present
        gross_margin = None
        operating_margin = None
        net_margin = None

        if row_dict.get('revenue') and row_dict['revenue'] > 0:
            if row_dict.get('gross_profit'):
                gross_margin = float(row_dict['gross_profit']) / float(row_dict['revenue'])
            if row_dict.get('operating_income'):
                operating_margin = float(row_dict['operating_income']) / float(row_dict['revenue'])
            if row_dict.get('net_income'):
                net_margin = float(row_dict['net_income']) / float(row_dict['revenue'])

        periods.append(FinancialStatementItem(
            fiscal_year=fiscal_year,
            fiscal_quarter=None,  # TTM doesn't have a quarter
            period_end_date=row_dict['ttm_period_end'],
            filing_date=row_dict.get('calculation_date'),
            revenue=row_dict.get('revenue'),
            cost_of_revenue=row_dict.get('cost_of_revenue'),
            gross_profit=row_dict.get('gross_profit'),
            operating_expenses=row_dict.get('operating_expenses'),
            operating_income=row_dict.get('operating_income'),
            net_income=row_dict.get('net_income'),
            eps_basic=float(row_dict['eps_basic']) if row_dict.get('eps_basic') else None,
            eps_diluted=float(row_dict['eps_diluted']) if row_dict.get('eps_diluted') else None,
            shares_outstanding=row_dict.get('shares_outstanding'),
            total_assets=row_dict.get('total_assets'),
            total_liabilities=row_dict.get('total_liabilities'),
            shareholders_equity=row_dict.get('shareholders_equity'),
            cash_and_equivalents=row_dict.get('cash_and_equivalents'),
            short_term_debt=row_dict.get('short_term_debt'),
            long_term_debt=row_dict.get('long_term_debt'),
            operating_cash_flow=row_dict.get('operating_cash_flow'),
            investing_cash_flow=row_dict.get('investing_cash_flow'),
            financing_cash_flow=row_dict.get('financing_cash_flow'),
            free_cash_flow=row_dict.get('free_cash_flow'),
            capex=row_dict.get('capex'),
            gross_margin=gross_margin,
            operating_margin=operating_margin,
            net_margin=net_margin,
            roe=None,  # TTM table doesn't have ROE
            roa=None,  # TTM table doesn't have ROA
            current_ratio=None,  # TTM table doesn't have current ratio
            debt_to_equity=None,  # TTM table doesn't have debt_to_equity
            yoy_change=yoy_change if yoy_change else None
        ))

    # Get company name for metadata
    meta_query = text("SELECT name FROM tickers WHERE UPPER(symbol) = UPPER(:ticker) LIMIT 1")
    meta_result = await session.execute(meta_query, {"ticker": ticker})
    meta_row = meta_result.fetchone()

    return {
        "ticker": ticker,
        "statement_type": statement_type,
        "timeframe": "ttm",
        "periods": [p.model_dump() for p in periods],
        "metadata": {
            "company_name": meta_row.name if meta_row else ticker
        }
    }


# ============================================================================
# Financial Metrics Endpoints
# ============================================================================

@app.get("/api/metrics/{ticker}", response_model=FinancialMetricsResponse)
async def get_financial_metrics(
    ticker: str,
    session: AsyncSession = Depends(get_session)
):
    """Get latest financial metrics for a ticker.

    Returns profitability ratios, financial health metrics, and growth rates
    calculated from SEC filings data.

    Args:
        ticker: Stock ticker symbol.

    Returns:
        Financial metrics including margins, ratios, and growth rates.

    Raises:
        HTTPException: If ticker not found.
    """
    ticker = ticker.upper()

    # Get latest quarterly or annual financials
    query = text("""
        WITH latest AS (
            SELECT *
            FROM financials
            WHERE ticker = :ticker
            ORDER BY period_end_date DESC
            LIMIT 1
        ),
        prior_year AS (
            SELECT revenue, eps_diluted, period_end_date
            FROM financials
            WHERE ticker = :ticker
              AND period_end_date <= (SELECT period_end_date - INTERVAL '11 months' FROM latest)
            ORDER BY period_end_date DESC
            LIMIT 1
        )
        SELECT
            l.*,
            p.revenue as prior_revenue,
            p.eps_diluted as prior_eps
        FROM latest l
        LEFT JOIN prior_year p ON TRUE
    """)

    result = await session.execute(query, {"ticker": ticker})
    row = result.fetchone()

    if not row:
        raise HTTPException(status_code=404, detail=f"Financial data not found for ticker {ticker}")

    row_dict = row._asdict()

    # Calculate YoY growth rates
    revenue_growth = None
    earnings_growth = None

    if row_dict.get('prior_revenue') and row_dict.get('revenue'):
        prior_rev = float(row_dict['prior_revenue'])
        curr_rev = float(row_dict['revenue'])
        if prior_rev > 0:
            revenue_growth = ((curr_rev - prior_rev) / prior_rev) * 100

    if row_dict.get('prior_eps') and row_dict.get('eps_diluted'):
        prior_eps = float(row_dict['prior_eps'])
        curr_eps = float(row_dict['eps_diluted'])
        if prior_eps != 0:
            earnings_growth = ((curr_eps - prior_eps) / abs(prior_eps)) * 100

    return FinancialMetricsResponse(
        ticker=ticker,
        period_end_date=row_dict.get('period_end_date'),
        fiscal_year=row_dict.get('fiscal_year'),
        fiscal_quarter=row_dict.get('fiscal_quarter'),
        gross_margin=float(row_dict['gross_margin']) if row_dict.get('gross_margin') else None,
        operating_margin=float(row_dict['operating_margin']) if row_dict.get('operating_margin') else None,
        net_margin=float(row_dict['net_margin']) if row_dict.get('net_margin') else None,
        roe=float(row_dict['roe']) if row_dict.get('roe') else None,
        roa=float(row_dict['roa']) if row_dict.get('roa') else None,
        debt_to_equity=float(row_dict['debt_to_equity']) if row_dict.get('debt_to_equity') else None,
        current_ratio=float(row_dict['current_ratio']) if row_dict.get('current_ratio') else None,
        quick_ratio=float(row_dict['quick_ratio']) if row_dict.get('quick_ratio') else None,
        pe_ratio=float(row_dict['pe_ratio']) if row_dict.get('pe_ratio') else None,
        pb_ratio=float(row_dict['pb_ratio']) if row_dict.get('pb_ratio') else None,
        ps_ratio=float(row_dict['ps_ratio']) if row_dict.get('ps_ratio') else None,
        revenue_growth_yoy=revenue_growth,
        earnings_growth_yoy=earnings_growth,
        shares_outstanding=int(row_dict['shares_outstanding']) if row_dict.get('shares_outstanding') else None,
        statement_type=row_dict.get('statement_type'),
        data_as_of=row_dict.get('period_end_date')
    )


# ============================================================================
# Risk Metrics Endpoints
# ============================================================================

@app.get("/api/risk/{ticker}", response_model=RiskMetricsResponse)
async def get_risk_metrics(
    ticker: str,
    period: str = Query("1Y", description="Period for risk metrics (1Y, 3Y, 5Y)"),
    session: AsyncSession = Depends(get_session)
):
    """Get risk metrics for a ticker.

    Returns risk-adjusted performance metrics including Beta, Alpha,
    Sharpe Ratio, and volatility measures.

    Args:
        ticker: Stock ticker symbol.
        period: Time period (1Y, 3Y, 5Y).

    Returns:
        Risk metrics calculated over the specified period.

    Raises:
        HTTPException: If ticker not found.
    """
    ticker = ticker.upper()

    query = text("""
        SELECT
            time,
            ticker,
            period,
            alpha,
            beta,
            sharpe_ratio,
            sortino_ratio,
            std_dev,
            max_drawdown,
            var_5,
            annualized_return,
            downside_deviation,
            data_points,
            calculation_date
        FROM risk_metrics
        WHERE ticker = :ticker
          AND period = :period
        ORDER BY time DESC
        LIMIT 1
    """)

    result = await session.execute(query, {"ticker": ticker, "period": period})
    row = result.fetchone()

    if not row:
        raise HTTPException(
            status_code=404,
            detail=f"Risk metrics not found for ticker {ticker} with period {period}"
        )

    row_dict = row._asdict()

    return RiskMetricsResponse(
        ticker=ticker,
        period=row_dict.get('period', period),
        calculation_date=row_dict.get('time').date() if row_dict.get('time') else None,
        beta=float(row_dict['beta']) if row_dict.get('beta') else None,
        alpha=float(row_dict['alpha']) if row_dict.get('alpha') else None,
        sharpe_ratio=float(row_dict['sharpe_ratio']) if row_dict.get('sharpe_ratio') else None,
        sortino_ratio=float(row_dict['sortino_ratio']) if row_dict.get('sortino_ratio') else None,
        volatility=float(row_dict['std_dev']) if row_dict.get('std_dev') else None,
        max_drawdown=float(row_dict['max_drawdown']) if row_dict.get('max_drawdown') else None,
        var_95=float(row_dict['var_5']) if row_dict.get('var_5') else None,
        annualized_return=float(row_dict['annualized_return']) if row_dict.get('annualized_return') else None,
        downside_deviation=float(row_dict['downside_deviation']) if row_dict.get('downside_deviation') else None,
        data_points=int(row_dict['data_points']) if row_dict.get('data_points') else None
    )


# ============================================================================
# Technical Indicators Endpoints
# ============================================================================

@app.get("/api/technical/{ticker}", response_model=TechnicalIndicatorsResponse)
async def get_technical_indicators(
    ticker: str,
    session: AsyncSession = Depends(get_session)
):
    """Get latest technical indicators for a ticker.

    Returns moving averages, oscillators (RSI, MACD), and Bollinger Bands.

    Args:
        ticker: Stock ticker symbol.

    Returns:
        Technical indicators for the ticker.

    Raises:
        HTTPException: If ticker not found.
    """
    ticker = ticker.upper()

    query = text("""
        SELECT
            time,
            ticker,
            close,
            sma_20,
            sma_50,
            sma_200,
            ema_12,
            ema_26,
            rsi_14,
            macd,
            macd_signal,
            macd_histogram,
            bb_upper,
            bb_middle,
            bb_lower,
            atr_14,
            adx_14,
            stoch_k,
            stoch_d
        FROM technical_indicators
        WHERE ticker = :ticker
        ORDER BY time DESC
        LIMIT 1
    """)

    result = await session.execute(query, {"ticker": ticker})
    row = result.fetchone()

    if not row:
        raise HTTPException(
            status_code=404,
            detail=f"Technical indicators not found for ticker {ticker}"
        )

    row_dict = row._asdict()

    return TechnicalIndicatorsResponse(
        ticker=ticker,
        date=row_dict.get('time').date() if row_dict.get('time') else None,
        sma_20=float(row_dict['sma_20']) if row_dict.get('sma_20') else None,
        sma_50=float(row_dict['sma_50']) if row_dict.get('sma_50') else None,
        sma_200=float(row_dict['sma_200']) if row_dict.get('sma_200') else None,
        ema_12=float(row_dict['ema_12']) if row_dict.get('ema_12') else None,
        ema_26=float(row_dict['ema_26']) if row_dict.get('ema_26') else None,
        rsi_14=float(row_dict['rsi_14']) if row_dict.get('rsi_14') else None,
        macd=float(row_dict['macd']) if row_dict.get('macd') else None,
        macd_signal=float(row_dict['macd_signal']) if row_dict.get('macd_signal') else None,
        macd_histogram=float(row_dict['macd_histogram']) if row_dict.get('macd_histogram') else None,
        bb_upper=float(row_dict['bb_upper']) if row_dict.get('bb_upper') else None,
        bb_middle=float(row_dict['bb_middle']) if row_dict.get('bb_middle') else None,
        bb_lower=float(row_dict['bb_lower']) if row_dict.get('bb_lower') else None,
        atr_14=float(row_dict['atr_14']) if row_dict.get('atr_14') else None,
        adx_14=float(row_dict['adx_14']) if row_dict.get('adx_14') else None,
        stoch_k=float(row_dict['stoch_k']) if row_dict.get('stoch_k') else None,
        stoch_d=float(row_dict['stoch_d']) if row_dict.get('stoch_d') else None,
        close_price=float(row_dict['close']) if row_dict.get('close') else None
    )


# ============================================================================
# NEWS SENTIMENT ENDPOINTS
# ============================================================================

class NewsArticleResponse(BaseModel):
    """News article with sentiment analysis response."""
    id: int
    title: str
    url: str
    source: str
    published_at: datetime
    summary: Optional[str] = None
    author: Optional[str] = None
    tickers: Optional[List[str]] = None
    sentiment_score: Optional[float] = None  # -100 to +100
    sentiment_label: Optional[str] = None    # Positive, Negative, Neutral
    relevance_score: Optional[float] = None  # 0 to 100
    image_url: Optional[str] = None

    class Config:
        from_attributes = True


class NewsResponse(BaseModel):
    """News articles response."""
    ticker: str
    articles: List[NewsArticleResponse]
    count: int


@app.get("/api/news/{ticker}", response_model=NewsResponse)
async def get_ticker_news(
    ticker: str,
    limit: int = Query(default=30, ge=1, le=100, description="Number of articles to return"),
    days: int = Query(default=30, ge=1, le=365, description="Days of history to fetch"),
    session: AsyncSession = Depends(get_session)
):
    """
    Get news articles with AI sentiment analysis for a ticker.

    Returns articles from the news_articles table with:
    - sentiment_score: -100 (very negative) to +100 (very positive)
    - sentiment_label: Positive, Negative, or Neutral
    - relevance_score: 0-100 how relevant the article is to the ticker
    """
    ticker = ticker.upper()
    logger.info(f"Fetching news for ticker: {ticker}")

    try:
        cutoff_date = datetime.now() - timedelta(days=days)

        # Query news_articles table for articles mentioning this ticker
        query = text("""
            SELECT
                id,
                title,
                url,
                source,
                published_at,
                summary,
                author,
                tickers,
                sentiment_score,
                sentiment_label,
                relevance_score,
                image_url
            FROM news_articles
            WHERE :ticker = ANY(tickers)
              AND published_at >= :cutoff_date
            ORDER BY published_at DESC
            LIMIT :limit
        """)

        result = await session.execute(query, {
            "ticker": ticker,
            "cutoff_date": cutoff_date,
            "limit": limit
        })
        rows = result.fetchall()

        articles = []
        for row in rows:
            row_dict = row._asdict()
            articles.append(NewsArticleResponse(
                id=row_dict['id'],
                title=row_dict['title'],
                url=row_dict['url'],
                source=row_dict['source'],
                published_at=row_dict['published_at'],
                summary=row_dict.get('summary'),
                author=row_dict.get('author'),
                tickers=row_dict.get('tickers'),
                sentiment_score=float(row_dict['sentiment_score']) if row_dict.get('sentiment_score') else None,
                sentiment_label=row_dict.get('sentiment_label'),
                relevance_score=float(row_dict['relevance_score']) if row_dict.get('relevance_score') else None,
                image_url=row_dict.get('image_url')
            ))

        logger.info(f"Found {len(articles)} news articles for {ticker}")

        return NewsResponse(
            ticker=ticker,
            articles=articles,
            count=len(articles)
        )

    except Exception as e:
        logger.error(f"Error fetching news for {ticker}: {e}")
        raise HTTPException(status_code=500, detail=str(e))


# ============================================================================
# BACKTEST ENDPOINTS
# ============================================================================

class RebalanceFrequencyEnum(str, Enum):
    """Rebalance frequency options."""
    daily = "daily"
    weekly = "weekly"
    monthly = "monthly"
    quarterly = "quarterly"


class UniverseEnum(str, Enum):
    """Stock universe options."""
    sp500 = "sp500"
    sp1500 = "sp1500"
    all = "all"


class BacktestConfigRequest(BaseModel):
    """Backtest configuration request model."""
    start_date: date
    end_date: date
    rebalance_frequency: RebalanceFrequencyEnum = RebalanceFrequencyEnum.monthly
    universe: UniverseEnum = UniverseEnum.sp500
    benchmark: str = "SPY"
    transaction_cost_bps: float = Field(default=10.0, ge=0, le=100)
    slippage_bps: float = Field(default=5.0, ge=0, le=100)
    exclude_financials: bool = False
    exclude_utilities: bool = False
    use_smoothed_scores: bool = True


class DecilePerformanceResponse(BaseModel):
    """Performance metrics for a single decile."""
    decile: int
    total_return: float
    annualized_return: float
    sharpe_ratio: Optional[float] = None
    max_drawdown: float
    num_periods: int


class BacktestSummaryResponse(BaseModel):
    """Backtest results summary."""
    config: Dict[str, Any]
    decile_performance: List[DecilePerformanceResponse]
    spread_cagr: float
    top_vs_benchmark: float
    hit_rate: float
    monotonicity_score: float
    information_ratio: float
    top_decile_sharpe: float
    top_decile_max_dd: float
    benchmark: str
    start_date: date
    end_date: date
    num_periods: int


class BacktestJobResponse(BaseModel):
    """Response for async backtest job submission."""
    job_id: str
    status: str
    message: str


# In-memory job storage (in production, use Redis or database)
_backtest_jobs: Dict[str, Dict[str, Any]] = {}


@app.get("/api/backtest/config/default")
async def get_default_backtest_config():
    """Get default backtest configuration.

    Returns:
        Default configuration values for running a backtest.
    """
    return {
        "start_date": (date.today() - timedelta(days=365*5)).isoformat(),
        "end_date": date.today().isoformat(),
        "rebalance_frequency": "monthly",
        "universe": "sp500",
        "benchmark": "SPY",
        "transaction_cost_bps": 10.0,
        "slippage_bps": 5.0,
        "exclude_financials": False,
        "exclude_utilities": False,
        "use_smoothed_scores": True,
    }


@app.post("/api/backtest", response_model=BacktestSummaryResponse)
async def run_backtest(
    config: BacktestConfigRequest,
    session: AsyncSession = Depends(get_session)
):
    """Run a backtest with the specified configuration.

    This endpoint runs a synchronous backtest and returns results.
    For long-running backtests, use the async job endpoint instead.

    Args:
        config: Backtest configuration parameters.

    Returns:
        Backtest summary with decile performance metrics.
    """
    try:
        # Convert request to backtester config
        backtester_config = BacktesterConfig(
            start_date=config.start_date,
            end_date=config.end_date,
            rebalance_frequency=RebalanceFrequency(config.rebalance_frequency.value),
            universe=config.universe.value,
            benchmark=config.benchmark,
            transaction_cost_bps=config.transaction_cost_bps,
            slippage_bps=config.slippage_bps,
            exclude_financials=config.exclude_financials,
            exclude_utilities=config.exclude_utilities,
            use_smoothed_scores=config.use_smoothed_scores,
        )

        # Run backtest
        backtester = ICScoreBacktester(session)
        results = await backtester.run_backtest(backtester_config)

        # Convert results to response format
        decile_perf = []
        for decile in range(1, 11):
            decile_perf.append(DecilePerformanceResponse(
                decile=decile,
                total_return=results.total_return_by_decile.get(decile, 0.0),
                annualized_return=results.annualized_return_by_decile.get(decile, 0.0),
                sharpe_ratio=results.sharpe_ratio_by_decile.get(decile),
                max_drawdown=results.max_drawdown_by_decile.get(decile, 0.0),
                num_periods=len([pr for pr in results.period_results if pr.decile == decile]),
            ))

        return BacktestSummaryResponse(
            config={
                "start_date": config.start_date.isoformat(),
                "end_date": config.end_date.isoformat(),
                "rebalance_frequency": config.rebalance_frequency.value,
                "universe": config.universe.value,
                "benchmark": config.benchmark,
                "transaction_cost_bps": config.transaction_cost_bps,
                "slippage_bps": config.slippage_bps,
                "exclude_financials": config.exclude_financials,
                "exclude_utilities": config.exclude_utilities,
            },
            decile_performance=decile_perf,
            spread_cagr=results.top_bottom_spread,
            top_vs_benchmark=results.top_vs_benchmark,
            hit_rate=results.hit_rate,
            monotonicity_score=results.monotonicity_score,
            information_ratio=results.information_ratio,
            top_decile_sharpe=results.sharpe_ratio_by_decile.get(1, 0.0),
            top_decile_max_dd=results.max_drawdown_by_decile.get(1, 0.0),
            benchmark=config.benchmark,
            start_date=config.start_date,
            end_date=config.end_date,
            num_periods=len(results.period_results) // 10 if results.period_results else 0,
        )

    except Exception as e:
        logger.error(f"Backtest failed: {e}")
        raise HTTPException(status_code=500, detail=f"Backtest failed: {str(e)}")


async def _run_backtest_async(job_id: str, config: BacktestConfigRequest):
    """Background task to run backtest asynchronously."""
    try:
        _backtest_jobs[job_id]["status"] = "running"
        _backtest_jobs[job_id]["started_at"] = datetime.now().isoformat()

        db = get_database()
        async with db.get_session() as session:
            backtester_config = BacktesterConfig(
                start_date=config.start_date,
                end_date=config.end_date,
                rebalance_frequency=RebalanceFrequency(config.rebalance_frequency.value),
                universe=config.universe.value,
                benchmark=config.benchmark,
                transaction_cost_bps=config.transaction_cost_bps,
                slippage_bps=config.slippage_bps,
                exclude_financials=config.exclude_financials,
                exclude_utilities=config.exclude_utilities,
                use_smoothed_scores=config.use_smoothed_scores,
            )

            backtester = ICScoreBacktester(session)
            results = await backtester.run_backtest(backtester_config)

            # Store results
            decile_perf = []
            for decile in range(1, 11):
                decile_perf.append({
                    "decile": decile,
                    "total_return": results.total_return_by_decile.get(decile, 0.0),
                    "annualized_return": results.annualized_return_by_decile.get(decile, 0.0),
                    "sharpe_ratio": results.sharpe_ratio_by_decile.get(decile),
                    "max_drawdown": results.max_drawdown_by_decile.get(decile, 0.0),
                })

            _backtest_jobs[job_id]["result"] = {
                "decile_performance": decile_perf,
                "spread_cagr": results.top_bottom_spread,
                "top_vs_benchmark": results.top_vs_benchmark,
                "hit_rate": results.hit_rate,
                "monotonicity_score": results.monotonicity_score,
                "information_ratio": results.information_ratio,
            }
            _backtest_jobs[job_id]["status"] = "completed"
            _backtest_jobs[job_id]["completed_at"] = datetime.now().isoformat()

    except Exception as e:
        logger.error(f"Async backtest {job_id} failed: {e}")
        _backtest_jobs[job_id]["status"] = "failed"
        _backtest_jobs[job_id]["error"] = str(e)
        _backtest_jobs[job_id]["completed_at"] = datetime.now().isoformat()


@app.post("/api/backtest/jobs", response_model=BacktestJobResponse)
async def submit_backtest_job(
    config: BacktestConfigRequest,
    background_tasks: BackgroundTasks
):
    """Submit a backtest job to run asynchronously.

    Use this for long-running backtests. Poll the status endpoint
    to check when the job is complete.

    Args:
        config: Backtest configuration parameters.

    Returns:
        Job ID and initial status.
    """
    import uuid
    job_id = str(uuid.uuid4())

    _backtest_jobs[job_id] = {
        "id": job_id,
        "status": "pending",
        "config": config.model_dump(),
        "created_at": datetime.now().isoformat(),
        "started_at": None,
        "completed_at": None,
        "result": None,
        "error": None,
    }

    background_tasks.add_task(_run_backtest_async, job_id, config)

    return BacktestJobResponse(
        job_id=job_id,
        status="pending",
        message="Backtest job submitted successfully"
    )


@app.get("/api/backtest/jobs/{job_id}")
async def get_backtest_job_status(job_id: str):
    """Get the status of a backtest job.

    Args:
        job_id: The ID of the backtest job.

    Returns:
        Job status and metadata.
    """
    if job_id not in _backtest_jobs:
        raise HTTPException(status_code=404, detail="Job not found")

    job = _backtest_jobs[job_id]

    response = {
        "job_id": job["id"],
        "status": job["status"],
        "created_at": job["created_at"],
    }

    if job["started_at"]:
        response["started_at"] = job["started_at"]

    if job["completed_at"]:
        response["completed_at"] = job["completed_at"]

    if job["error"]:
        response["error"] = job["error"]

    return response


@app.get("/api/backtest/jobs/{job_id}/result")
async def get_backtest_job_result(job_id: str):
    """Get the result of a completed backtest job.

    Args:
        job_id: The ID of the backtest job.

    Returns:
        Backtest results if job is completed.
    """
    if job_id not in _backtest_jobs:
        raise HTTPException(status_code=404, detail="Job not found")

    job = _backtest_jobs[job_id]

    if job["status"] != "completed":
        raise HTTPException(
            status_code=400,
            detail=f"Job is not completed. Current status: {job['status']}"
        )

    return job["result"]


# Startup event
@app.on_event("startup")
async def startup_event():
    """Run on application startup."""
    logger.info("IC Score API starting up...")
    db = get_database()
    health = await db.health_check()
    logger.info(f"Database health: {health}")


# Shutdown event
@app.on_event("shutdown")
async def shutdown_event():
    """Run on application shutdown."""
    logger.info("IC Score API shutting down...")
    db = get_database()
    await db.close()


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000, log_level="info")
