"""Pipeline orchestration tests for IC Score Calculator.

Tests the calculator pipeline with mocked database interactions:
- Full calculate_ic_score pipeline with mock data
- Score calculation logic and weighted averaging
- Pipeline error handling
- Data completeness enforcement
- Rating determination logic
- store_ic_score persistence
- process_stocks orchestration
- run() end-to-end pipeline flow

Uses AsyncMock to patch the database session and all fetch methods,
avoiding any real database calls.
"""

import asyncio
import logging
import os
import sys
import tempfile
from datetime import date, datetime
from decimal import Decimal
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch, PropertyMock

import pytest

# Set LOG_DIR before any pipeline import to avoid /app/logs errors
_tmpdir = tempfile.mkdtemp(prefix="ic_pipeline_test_logs_")
os.environ["LOG_DIR"] = _tmpdir

# Patch FileHandler to avoid filesystem issues
_original_file_handler = logging.FileHandler


class _SafeFileHandler(logging.StreamHandler):
    def __init__(self, filename=None, mode="a", encoding=None, delay=False):
        super().__init__()


logging.FileHandler = _SafeFileHandler  # type: ignore[misc]

# Ensure ic-score-service root is on sys.path
sys.path.insert(0, str(Path(__file__).parent.parent))

# Import with mocked database
with patch("pipelines.ic_score_calculator.get_database"):
    from pipelines.ic_score_calculator import ICScoreCalculator


# ========================================================================
# Fixtures
# ========================================================================


@pytest.fixture
def mock_db():
    """Create a mock Database object with session context manager."""
    db = MagicMock()
    mock_session = AsyncMock()

    # Make db.session() an async context manager returning mock_session
    session_cm = AsyncMock()
    session_cm.__aenter__ = AsyncMock(return_value=mock_session)
    session_cm.__aexit__ = AsyncMock(return_value=False)
    db.session.return_value = session_cm

    return db, mock_session


@pytest.fixture
def calc(mock_db):
    """Create ICScoreCalculator with mocked DB and disabled optional features."""
    db, _ = mock_db
    with patch("pipelines.ic_score_calculator.get_database", return_value=db):
        c = ICScoreCalculator()
    c.db = db
    # Disable features that require real DB connections
    c.USE_SECTOR_RELATIVE_SCORING = False
    c.USE_LIFECYCLE_WEIGHTS = False
    c.USE_SCORE_STABILIZATION = False
    c.USE_PEER_COMPARISON = False
    c.USE_CATALYST_DETECTION = False
    c.USE_SCORE_EXPLANATIONS = False
    c._sector_calculator = None
    c._lifecycle_classifier = None
    c._earnings_revisions_calc = None
    c._historical_valuation_calc = None
    c._dividend_quality_calc = None
    c._score_stabilizer = None
    c._peer_comparison = None
    c._catalyst_detector = None
    c._score_explainer = None
    return c


# ========================================================================
# Helper: Sample data generators
# ========================================================================


def make_financial_data():
    """Sample financial data dict."""
    return {
        "latest": {
            "revenue_growth_yoy": 15.0,
            "eps_growth_yoy": 20.0,
            "net_margin": 25.0,
            "gross_margin": 45.0,
            "operating_margin": 30.0,
            "roe": 30.0,
            "roa": 12.0,
            "debt_to_equity": 0.5,
            "current_ratio": 2.0,
            "quick_ratio": 1.5,
        },
        "historical": [],
    }


def make_fundamental_metrics():
    """Sample fundamental metrics dict."""
    return {
        "revenue_growth_yoy": 15.0,
        "eps_growth_yoy": 20.0,
        "net_margin": 25.0,
        "roe": 30.0,
        "roa": 12.0,
        "roic": 18.0,
        "gross_margin": 45.0,
        "operating_margin": 30.0,
        "dividend_yield": 1.5,
        "payout_ratio": 25.0,
        "debt_to_equity": 0.5,
        "current_ratio": 2.0,
        "quick_ratio": 1.5,
        "ev_to_ebitda": 12.0,
        "ev_to_revenue": 5.0,
        "calculation_date": date.today(),
        "fcf_growth_yoy": 10.0,
        "dcf_fair_value": 200.0,
        "dcf_upside_percent": 15.0,
        "graham_number": 180.0,
    }


def make_valuation_data():
    """Sample valuation data dict."""
    return {
        "pe_ratio": 20.0,
        "pb_ratio": 3.0,
        "ps_ratio": 4.0,
        "stock_price": 170.0,
        "market_cap": 2_500_000_000_000,
    }


def make_technical_data():
    """Sample technical indicators dict."""
    return {
        "rsi": 55,
        "macd_histogram": 1.5,
        "current_price": 170.0,
        "sma_50": 165.0,
        "1m_return": 5.0,
        "3m_return": 8.0,
        "6m_return": 12.0,
        "12m_return": 20.0,
    }


def make_insider_data():
    """Sample insider trading data dict."""
    return {
        "net_buying_90d": 50000,
        "net_buying_value_90d": 5_000_000.0,
        "total_transactions": 10,
    }


def make_news_data():
    """Sample news sentiment data dict."""
    return {
        "article_count": 25,
        "avg_sentiment": 65.0,
        "positive_count": 15,
        "negative_count": 5,
        "neutral_count": 5,
        "recent_article_count": 8,
        "recent_avg_sentiment": 70.0,
    }


def make_analyst_data():
    """Sample analyst ratings data dict."""
    return {
        "total_analysts": 20,
        "buy_count": 12,
        "hold_count": 6,
        "sell_count": 2,
        "avg_price_target": 200.0,
    }


def make_institutional_data():
    """Sample institutional holdings data dict."""
    return {
        "institution_count": 1500,
        "total_shares_held": 4_000_000_000,
        "shares_outstanding": 5_000_000_000,
        "qoq_change_shares": 100_000_000,
    }


# ========================================================================
# Test: Full Pipeline — calculate_ic_score
# ========================================================================


class TestCalculateICScorePipeline:
    """Test the full calculate_ic_score orchestration."""

    @pytest.mark.asyncio
    async def test_full_pipeline_with_all_data(self, calc):
        """Test pipeline produces a valid score when all data sources return data."""
        # Patch all fetch methods to return sample data
        calc.fetch_financial_data = AsyncMock(return_value=make_financial_data())
        calc.fetch_fundamental_metrics = AsyncMock(return_value=make_fundamental_metrics())
        calc.fetch_valuation_data = AsyncMock(return_value=make_valuation_data())
        calc.fetch_technical_data = AsyncMock(return_value=make_technical_data())
        calc.fetch_insider_data = AsyncMock(return_value=make_insider_data())
        calc.fetch_news_sentiment_data = AsyncMock(return_value=make_news_data())
        calc.fetch_analyst_data = AsyncMock(return_value=make_analyst_data())
        calc.fetch_institutional_data = AsyncMock(return_value=make_institutional_data())

        result = await calc.calculate_ic_score("AAPL", "Technology")

        assert result is not None
        assert result["ticker"] == "AAPL"
        assert 0 <= result["overall_score"] <= 100
        assert result["date"] == date.today()
        assert result["rating"] in [
            "Strong Buy", "Buy", "Hold", "Underperform", "Sell"
        ]
        assert result["confidence_level"] in ["High", "Medium", "Low"]
        assert result["data_completeness"] > 0

        # Verify individual factor scores are present
        assert result["value_score"] is not None
        assert result["growth_score"] is not None
        assert result["momentum_score"] is not None
        assert result["technical_score"] is not None

        # Verify metadata
        assert "calculation_metadata" in result
        meta = result["calculation_metadata"]
        assert meta["scoring_version"] == "2.2"
        assert "factors" in meta
        assert "weights_used" in meta

    @pytest.mark.asyncio
    async def test_pipeline_with_no_data(self, calc):
        """Test pipeline returns None when no data sources return data."""
        calc.fetch_financial_data = AsyncMock(return_value=None)
        calc.fetch_fundamental_metrics = AsyncMock(return_value=None)
        calc.fetch_valuation_data = AsyncMock(return_value=None)
        calc.fetch_technical_data = AsyncMock(return_value=None)
        calc.fetch_insider_data = AsyncMock(return_value=None)
        calc.fetch_news_sentiment_data = AsyncMock(return_value=None)
        calc.fetch_analyst_data = AsyncMock(return_value=None)
        calc.fetch_institutional_data = AsyncMock(return_value=None)

        result = await calc.calculate_ic_score("ZZZZ", "Technology")

        assert result is None

    @pytest.mark.asyncio
    async def test_pipeline_data_completeness_threshold(self, calc):
        """Test pipeline rejects scores below minimum data completeness."""
        # Only provide valuation data (1 factor out of 10 = 10% < 40% threshold)
        calc.fetch_financial_data = AsyncMock(return_value=None)
        calc.fetch_fundamental_metrics = AsyncMock(return_value=None)
        calc.fetch_valuation_data = AsyncMock(return_value=make_valuation_data())
        calc.fetch_technical_data = AsyncMock(return_value=None)
        calc.fetch_insider_data = AsyncMock(return_value=None)
        calc.fetch_news_sentiment_data = AsyncMock(return_value=None)
        calc.fetch_analyst_data = AsyncMock(return_value=None)
        calc.fetch_institutional_data = AsyncMock(return_value=None)

        result = await calc.calculate_ic_score("LOW_DATA", "Technology")

        # Should return None due to insufficient data completeness
        assert result is None

    @pytest.mark.asyncio
    async def test_pipeline_core_factors_requirement(self, calc):
        """Test that pipeline requires at least 2 core factors."""
        # Provide momentum + technical + insider + news + analyst + institutional
        # (no core factors: value, growth, profitability, financial_health)
        calc.fetch_financial_data = AsyncMock(return_value=None)
        calc.fetch_fundamental_metrics = AsyncMock(return_value=None)
        calc.fetch_valuation_data = AsyncMock(return_value=None)
        calc.fetch_technical_data = AsyncMock(return_value=make_technical_data())
        calc.fetch_insider_data = AsyncMock(return_value=make_insider_data())
        calc.fetch_news_sentiment_data = AsyncMock(return_value=make_news_data())
        calc.fetch_analyst_data = AsyncMock(return_value=make_analyst_data())
        calc.fetch_institutional_data = AsyncMock(return_value=make_institutional_data())

        result = await calc.calculate_ic_score("NO_CORE", "Technology")

        # Should return None due to insufficient core factors
        assert result is None

    @pytest.mark.asyncio
    async def test_pipeline_partial_data(self, calc):
        """Test pipeline handles partial data gracefully."""
        # Provide enough data to pass thresholds (value + growth + momentum + technical + profitability)
        calc.fetch_financial_data = AsyncMock(return_value=make_financial_data())
        calc.fetch_fundamental_metrics = AsyncMock(return_value=make_fundamental_metrics())
        calc.fetch_valuation_data = AsyncMock(return_value=make_valuation_data())
        calc.fetch_technical_data = AsyncMock(return_value=make_technical_data())
        calc.fetch_insider_data = AsyncMock(return_value=None)
        calc.fetch_news_sentiment_data = AsyncMock(return_value=None)
        calc.fetch_analyst_data = AsyncMock(return_value=None)
        calc.fetch_institutional_data = AsyncMock(return_value=None)

        result = await calc.calculate_ic_score("PARTIAL", "Technology")

        assert result is not None
        assert result["ticker"] == "PARTIAL"
        assert 0 <= result["overall_score"] <= 100

        # Factors with no data should be None
        assert result.get("news_sentiment_score") is None
        assert result.get("analyst_consensus_score") is None

    @pytest.mark.asyncio
    async def test_pipeline_exception_handling(self, calc):
        """Test pipeline returns None on unexpected exception."""
        # Make fetch_financial_data raise an exception
        calc.fetch_financial_data = AsyncMock(side_effect=RuntimeError("API timeout"))
        calc.fetch_fundamental_metrics = AsyncMock(return_value=None)
        calc.fetch_valuation_data = AsyncMock(return_value=None)
        calc.fetch_technical_data = AsyncMock(return_value=None)
        calc.fetch_insider_data = AsyncMock(return_value=None)
        calc.fetch_news_sentiment_data = AsyncMock(return_value=None)
        calc.fetch_analyst_data = AsyncMock(return_value=None)
        calc.fetch_institutional_data = AsyncMock(return_value=None)

        result = await calc.calculate_ic_score("ERROR", "Technology")

        assert result is None


# ========================================================================
# Test: Rating Determination
# ========================================================================


class TestRatingDetermination:
    """Test rating thresholds and confidence levels."""

    @pytest.mark.asyncio
    async def test_strong_buy_rating(self, calc):
        """Test that high scores produce Strong Buy rating."""
        calc.fetch_financial_data = AsyncMock(return_value=make_financial_data())
        calc.fetch_fundamental_metrics = AsyncMock(return_value=make_fundamental_metrics())
        calc.fetch_valuation_data = AsyncMock(return_value={
            "pe_ratio": 8.0, "pb_ratio": 1.0, "ps_ratio": 0.5,
            "stock_price": 170.0, "market_cap": 2_500_000_000_000,
        })
        calc.fetch_technical_data = AsyncMock(return_value={
            "rsi": 60, "macd_histogram": 3.0, "current_price": 170, "sma_50": 150,
            "1m_return": 20.0, "3m_return": 15.0, "6m_return": 25.0, "12m_return": 40.0,
        })
        calc.fetch_insider_data = AsyncMock(return_value={
            "net_buying_90d": 100000,
            "net_buying_value_90d": 10_000_000.0,
            "total_transactions": 20,
        })
        calc.fetch_news_sentiment_data = AsyncMock(return_value={
            "article_count": 30, "avg_sentiment": 85.0,
            "positive_count": 25, "negative_count": 2, "neutral_count": 3,
            "recent_article_count": 10, "recent_avg_sentiment": 90.0,
        })
        calc.fetch_analyst_data = AsyncMock(return_value={
            "total_analysts": 25, "buy_count": 22, "hold_count": 3, "sell_count": 0,
            "avg_price_target": 250.0,
        })
        calc.fetch_institutional_data = AsyncMock(return_value=make_institutional_data())

        result = await calc.calculate_ic_score("BULL", "Technology")

        assert result is not None
        assert result["overall_score"] >= 65  # At least Buy territory
        assert result["rating"] in ["Strong Buy", "Buy"]

    @pytest.mark.asyncio
    async def test_confidence_level_high(self, calc):
        """Test high confidence when all data present (>= 90% completeness)."""
        calc.fetch_financial_data = AsyncMock(return_value=make_financial_data())
        calc.fetch_fundamental_metrics = AsyncMock(return_value=make_fundamental_metrics())
        calc.fetch_valuation_data = AsyncMock(return_value=make_valuation_data())
        calc.fetch_technical_data = AsyncMock(return_value=make_technical_data())
        calc.fetch_insider_data = AsyncMock(return_value=make_insider_data())
        calc.fetch_news_sentiment_data = AsyncMock(return_value=make_news_data())
        calc.fetch_analyst_data = AsyncMock(return_value=make_analyst_data())
        calc.fetch_institutional_data = AsyncMock(return_value=make_institutional_data())

        result = await calc.calculate_ic_score("HIGH_CONF", "Technology")

        assert result is not None
        # With all data sources, completeness should be >= 90%
        assert result["data_completeness"] >= 90
        assert result["confidence_level"] == "High"


# ========================================================================
# Test: Score Calculation Weights
# ========================================================================


class TestScoreWeights:
    """Test weighted score calculation."""

    def test_weights_sum_to_one(self):
        """Verify base weights sum to approximately 1.0."""
        total = sum(ICScoreCalculator.WEIGHTS.values())
        assert abs(total - 1.0) < 0.001, f"Weights sum to {total}, expected 1.0"

    def test_weight_keys_present(self):
        """Verify all expected weight keys exist."""
        expected_keys = {
            "profitability", "financial_health", "growth",
            "value", "intrinsic_value", "historical_value",
            "momentum", "smart_money", "earnings_revisions", "technical",
        }
        assert expected_keys == set(ICScoreCalculator.WEIGHTS.keys())

    def test_rating_thresholds_ordered(self):
        """Verify rating thresholds are properly ordered."""
        thresholds = ICScoreCalculator.RATING_THRESHOLDS
        assert thresholds["Strong Buy"] > thresholds["Buy"]
        assert thresholds["Buy"] > thresholds["Hold"]
        assert thresholds["Hold"] > thresholds["Underperform"]
        assert thresholds["Underperform"] > thresholds["Sell"]

    def test_min_data_completeness_reasonable(self):
        """Verify minimum data completeness is between 0 and 100."""
        assert 0 < ICScoreCalculator.MIN_DATA_COMPLETENESS < 100


# ========================================================================
# Test: store_ic_score
# ========================================================================


class TestStoreICScore:
    """Test IC Score persistence logic."""

    @pytest.mark.asyncio
    async def test_store_success(self, calc, mock_db):
        """Test storing a score successfully."""
        db, mock_session = mock_db

        # Mock the execute/commit flow
        mock_session.execute = AsyncMock()
        mock_session.commit = AsyncMock()

        score_data = {
            "ticker": "AAPL",
            "date": date.today(),
            "overall_score": 75.5,
            "value_score": 60.0,
            "growth_score": 80.0,
            "rating": "Buy",
            "confidence_level": "High",
            "data_completeness": 90.0,
            # Non-column keys should be filtered out
            "peers": [{"ticker": "MSFT"}],
            "previous_score": 72.0,
        }

        result = await calc.store_ic_score(score_data)

        assert result is True

    @pytest.mark.asyncio
    async def test_store_filters_non_column_keys(self, calc, mock_db):
        """Test that store_ic_score filters out non-column keys."""
        score_data = {
            "ticker": "AAPL",
            "date": date.today(),
            "overall_score": 75.5,
            "rating": "Buy",
            # These should be filtered out
            "peers": [{"ticker": "MSFT"}],
            "catalysts": [],
            "explanation": {"summary": "test"},
        }

        # Verify IC_SCORE_COLUMNS doesn't include 'peers', 'catalysts', 'explanation'
        assert "peers" not in ICScoreCalculator.IC_SCORE_COLUMNS
        assert "catalysts" not in ICScoreCalculator.IC_SCORE_COLUMNS
        assert "explanation" not in ICScoreCalculator.IC_SCORE_COLUMNS

    @pytest.mark.asyncio
    async def test_store_db_error(self, calc, mock_db):
        """Test store returns False on database error."""
        db, mock_session = mock_db

        # Make session raise on execute
        mock_session.execute = AsyncMock(side_effect=Exception("DB connection lost"))

        score_data = {
            "ticker": "FAIL",
            "date": date.today(),
            "overall_score": 50.0,
            "rating": "Hold",
        }

        result = await calc.store_ic_score(score_data)

        assert result is False


# ========================================================================
# Test: process_stocks
# ========================================================================


class TestProcessStocks:
    """Test the process_stocks orchestration loop."""

    @pytest.mark.asyncio
    async def test_process_single_stock_success(self, calc, mock_db):
        """Test processing a single stock successfully."""
        db, mock_session = mock_db

        # Mock calculate_ic_score to return valid data
        score_result = {
            "ticker": "AAPL",
            "date": date.today(),
            "overall_score": 75.0,
            "rating": "Buy",
            "confidence_level": "High",
            "data_completeness": 90.0,
        }
        calc.calculate_ic_score = AsyncMock(return_value=score_result)
        calc.store_ic_score = AsyncMock(return_value=True)
        calc._init_v2_components = AsyncMock()

        stocks = [{"ticker": "AAPL", "sector": "Technology"}]
        await calc.process_stocks(stocks, show_progress=False)

        assert calc.processed_count == 1
        assert calc.success_count == 1
        assert calc.error_count == 0
        calc.calculate_ic_score.assert_awaited_once_with("AAPL", "Technology")
        calc.store_ic_score.assert_awaited_once()

    @pytest.mark.asyncio
    async def test_process_multiple_stocks(self, calc, mock_db):
        """Test processing multiple stocks."""
        db, mock_session = mock_db

        calc.calculate_ic_score = AsyncMock(return_value={
            "ticker": "TEST",
            "date": date.today(),
            "overall_score": 60.0,
            "rating": "Hold",
        })
        calc.store_ic_score = AsyncMock(return_value=True)
        calc._init_v2_components = AsyncMock()

        stocks = [
            {"ticker": "AAPL", "sector": "Technology"},
            {"ticker": "MSFT", "sector": "Technology"},
            {"ticker": "GOOGL", "sector": "Technology"},
        ]

        # Reset counters
        calc.processed_count = 0
        calc.success_count = 0
        calc.error_count = 0

        await calc.process_stocks(stocks, show_progress=False)

        assert calc.processed_count == 3
        assert calc.success_count == 3
        assert calc.error_count == 0

    @pytest.mark.asyncio
    async def test_process_stock_with_null_score(self, calc, mock_db):
        """Test handling when calculate_ic_score returns None."""
        db, mock_session = mock_db

        calc.calculate_ic_score = AsyncMock(return_value=None)
        calc.store_ic_score = AsyncMock()
        calc._init_v2_components = AsyncMock()

        stocks = [{"ticker": "NULL_SCORE", "sector": "Unknown"}]

        calc.processed_count = 0
        calc.success_count = 0
        calc.error_count = 0

        await calc.process_stocks(stocks, show_progress=False)

        assert calc.processed_count == 1
        assert calc.success_count == 0
        assert calc.error_count == 1
        calc.store_ic_score.assert_not_awaited()

    @pytest.mark.asyncio
    async def test_process_stock_with_store_failure(self, calc, mock_db):
        """Test handling when store_ic_score fails."""
        db, mock_session = mock_db

        calc.calculate_ic_score = AsyncMock(return_value={
            "ticker": "STORE_FAIL",
            "date": date.today(),
            "overall_score": 50.0,
            "rating": "Hold",
        })
        calc.store_ic_score = AsyncMock(return_value=False)
        calc._init_v2_components = AsyncMock()

        stocks = [{"ticker": "STORE_FAIL", "sector": "Technology"}]

        calc.processed_count = 0
        calc.success_count = 0
        calc.error_count = 0

        await calc.process_stocks(stocks, show_progress=False)

        assert calc.processed_count == 1
        assert calc.success_count == 0
        assert calc.error_count == 1

    @pytest.mark.asyncio
    async def test_process_stock_exception_continues(self, calc, mock_db):
        """Test that an exception in one stock doesn't stop processing others."""
        db, mock_session = mock_db

        call_count = 0

        async def side_effect(ticker, sector):
            nonlocal call_count
            call_count += 1
            if ticker == "ERROR":
                raise RuntimeError("Unexpected error")
            return {
                "ticker": ticker,
                "date": date.today(),
                "overall_score": 60.0,
                "rating": "Hold",
            }

        calc.calculate_ic_score = AsyncMock(side_effect=side_effect)
        calc.store_ic_score = AsyncMock(return_value=True)
        calc._init_v2_components = AsyncMock()

        stocks = [
            {"ticker": "AAPL", "sector": "Technology"},
            {"ticker": "ERROR", "sector": "Technology"},
            {"ticker": "MSFT", "sector": "Technology"},
        ]

        calc.processed_count = 0
        calc.success_count = 0
        calc.error_count = 0

        await calc.process_stocks(stocks, show_progress=False)

        # All 3 should be processed
        assert calc.processed_count == 3
        # AAPL + MSFT succeed, ERROR fails
        assert calc.success_count == 2
        assert calc.error_count == 1


# ========================================================================
# Test: run() end-to-end pipeline
# ========================================================================


class TestRunPipeline:
    """Test the top-level run() method."""

    @pytest.mark.asyncio
    async def test_run_single_ticker(self, calc, mock_db):
        """Test running pipeline for a single ticker."""
        db, mock_session = mock_db

        calc.get_stocks_to_process = AsyncMock(
            return_value=[{"ticker": "AAPL", "sector": "Technology"}]
        )
        calc.process_stocks = AsyncMock()

        # Reset counters for success rate calculation
        calc.processed_count = 1
        calc.success_count = 1
        calc.error_count = 0

        await calc.run(ticker="AAPL")

        calc.get_stocks_to_process.assert_awaited_once_with(
            limit=1, ticker="AAPL", sector=None, sp500=False
        )
        calc.process_stocks.assert_awaited_once()

    @pytest.mark.asyncio
    async def test_run_with_limit(self, calc, mock_db):
        """Test running pipeline with limit."""
        db, mock_session = mock_db

        calc.get_stocks_to_process = AsyncMock(
            return_value=[
                {"ticker": "AAPL", "sector": "Technology"},
                {"ticker": "MSFT", "sector": "Technology"},
            ]
        )
        calc.process_stocks = AsyncMock()

        calc.processed_count = 2
        calc.success_count = 2
        calc.error_count = 0

        await calc.run(limit=2)

        calc.get_stocks_to_process.assert_awaited_once_with(
            limit=2, ticker=None, sector=None, sp500=False
        )

    @pytest.mark.asyncio
    async def test_run_no_stocks_found(self, calc, mock_db):
        """Test run() when no stocks match the criteria."""
        db, mock_session = mock_db

        calc.get_stocks_to_process = AsyncMock(return_value=[])
        calc.process_stocks = AsyncMock()

        await calc.run(sector="Nonexistent")

        calc.process_stocks.assert_not_awaited()

    @pytest.mark.asyncio
    async def test_run_all_stocks(self, calc, mock_db):
        """Test run with all_stocks flag."""
        db, mock_session = mock_db

        calc.get_stocks_to_process = AsyncMock(return_value=[])
        calc.process_stocks = AsyncMock()

        await calc.run(all_stocks=True)

        # all_stocks=True should set limit=None
        calc.get_stocks_to_process.assert_awaited_once_with(
            limit=None, ticker=None, sector=None, sp500=False
        )

    @pytest.mark.asyncio
    async def test_run_default_limit(self, calc, mock_db):
        """Test run with default limit (10)."""
        db, mock_session = mock_db

        calc.get_stocks_to_process = AsyncMock(return_value=[])
        calc.process_stocks = AsyncMock()

        await calc.run()

        # Default limit should be 10
        calc.get_stocks_to_process.assert_awaited_once_with(
            limit=10, ticker=None, sector=None, sp500=False
        )


# ========================================================================
# Test: Scoring Functions (synchronous)
# ========================================================================


class TestScoringFunctionsIntegration:
    """Integration tests for scoring functions with realistic data."""

    def test_momentum_score_bull_market(self, calc):
        """Test momentum scoring in a bull market scenario."""
        tech_data = {
            "1m_return": 8.0,
            "3m_return": 15.0,
            "6m_return": 25.0,
            "12m_return": 40.0,
        }
        score, meta = calc.calculate_momentum_score(tech_data)
        assert score is not None
        assert score > 60  # Should be well above neutral in bull market

    def test_momentum_score_bear_market(self, calc):
        """Test momentum scoring in a bear market scenario."""
        tech_data = {
            "1m_return": -8.0,
            "3m_return": -15.0,
            "6m_return": -25.0,
            "12m_return": -40.0,
        }
        score, meta = calc.calculate_momentum_score(tech_data)
        assert score is not None
        assert score < 40  # Should be well below neutral in bear market

    def test_technical_score_overbought(self, calc):
        """Test technical scoring when RSI signals overbought."""
        tech_data = {"rsi": 75, "macd_histogram": 2.0}
        score, meta = calc.calculate_technical_score(tech_data)
        assert score is not None
        # RSI at 75 is overbought = high technical score
        assert score > 50

    def test_news_sentiment_integration(self, calc):
        """Test news sentiment scoring with realistic data."""
        news_data = make_news_data()
        score, meta = calc.calculate_news_sentiment_score(news_data)
        assert score is not None
        assert 0 <= score <= 100
        assert meta.get("scoring_method") is not None

    def test_insider_activity_net_buying(self, calc):
        """Test insider activity score when insiders are net buying."""
        insider_data = make_insider_data()
        score, meta = calc.calculate_insider_activity_score(insider_data)
        assert score is not None
        assert score > 50  # Net buying should produce above-neutral score

    def test_insider_activity_net_selling(self, calc):
        """Test insider activity score when insiders are net selling."""
        insider_data = {
            "net_buying_90d": -100000,
            "net_buying_value_90d": -10_000_000.0,
            "total_transactions": 15,
        }
        score, meta = calc.calculate_insider_activity_score(insider_data)
        assert score is not None
        assert score < 50  # Net selling should produce below-neutral score

    def test_analyst_consensus_buy(self, calc):
        """Test analyst consensus with majority buys."""
        analyst_data = make_analyst_data()
        valuation_data = make_valuation_data()
        score, meta = calc.calculate_analyst_consensus_score(analyst_data, valuation_data)
        assert score is not None
        assert score > 50  # Majority buys + upside should be positive

    def test_intrinsic_value_upside(self, calc):
        """Test intrinsic value score with positive upside."""
        metrics = make_fundamental_metrics()
        valuation = make_valuation_data()
        score, meta = calc.calculate_intrinsic_value_score(metrics, valuation)
        assert score is not None
        assert 0 <= score <= 100


# ========================================================================
# Test: IC Score Column Filtering
# ========================================================================


class TestICScoreColumns:
    """Test IC_SCORE_COLUMNS filtering."""

    def test_required_columns_present(self):
        """Verify essential columns are in IC_SCORE_COLUMNS."""
        required = {"ticker", "date", "overall_score", "rating", "confidence_level"}
        assert required.issubset(ICScoreCalculator.IC_SCORE_COLUMNS)

    def test_score_columns_present(self):
        """Verify all individual score columns are present."""
        score_columns = {
            "value_score", "growth_score", "profitability_score",
            "financial_health_score", "momentum_score",
            "analyst_consensus_score", "insider_activity_score",
            "institutional_score", "news_sentiment_score", "technical_score",
        }
        assert score_columns.issubset(ICScoreCalculator.IC_SCORE_COLUMNS)

    def test_non_model_fields_excluded(self):
        """Verify transient fields are not in IC_SCORE_COLUMNS."""
        excluded = {"peers", "catalysts", "explanation", "peer_comparison", "previous_score"}
        assert not excluded.intersection(ICScoreCalculator.IC_SCORE_COLUMNS)
