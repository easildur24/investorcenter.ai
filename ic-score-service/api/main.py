"""FastAPI service for IC Score API.

This service provides REST API endpoints for accessing IC Scores and related data.
"""

import logging
from datetime import datetime, timedelta, date
from typing import List, Optional

from fastapi import FastAPI, HTTPException, Query, Depends
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
from sqlalchemy import text, select, desc
from sqlalchemy.ext.asyncio import AsyncSession

import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent.parent))

from database.database import get_database, get_session
from models import ICScore, Financial, TechnicalIndicator, NewsArticle

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
    ticker: str
    date: Optional[date] = None

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

    class Config:
        from_attributes = True


# API Endpoints

@app.get("/", include_in_schema=False)
async def root():
    """Root endpoint."""
    return {
        "service": "IC Score API",
        "version": "1.1.0",
        "endpoints": {
            "health": "/health",
            "score": "/api/scores/{ticker}",
            "history": "/api/scores/{ticker}/history",
            "top": "/api/scores/top",
            "screener": "/api/scores/screener",
            "metrics": "/api/metrics/{ticker}",
            "risk": "/api/risk/{ticker}",
            "technical": "/api/technical/{ticker}"
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
