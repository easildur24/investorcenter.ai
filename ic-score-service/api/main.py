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


# API Endpoints

@app.get("/", include_in_schema=False)
async def root():
    """Root endpoint."""
    return {
        "service": "IC Score API",
        "version": "1.0.0",
        "endpoints": {
            "health": "/health",
            "score": "/api/scores/{ticker}",
            "history": "/api/scores/{ticker}/history",
            "top": "/api/scores/top",
            "screener": "/api/scores/screener"
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
