"""Tests for IC Score API endpoints.

Uses FastAPI TestClient with database dependency overridden by a mock
session. Each endpoint is tested for success and error paths.

Note: /api/scores/top and /api/scores/screener are defined AFTER
/api/scores/{ticker} in the source, so Starlette matches the path
parameter route first. Tests for the top/screener handler functions
call the async functions directly with a mock session instead.
"""

import sys
from collections import namedtuple
from datetime import date, datetime, timedelta
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from fastapi.testclient import TestClient

# Ensure the ic-score-service root is on sys.path
sys.path.insert(0, str(Path(__file__).parent.parent))

from api.main import (
    app,
    get_ic_score,
    get_score_history,
    get_top_scores,
    screener,
)
from database.database import get_session


# -------------------------------------------------------------------
# Helpers
# -------------------------------------------------------------------


def _make_row(mapping: dict):
    """Create a namedtuple-like object that supports _asdict()."""
    Row = namedtuple("Row", mapping.keys())
    row = Row(**mapping)
    return row


def _ic_score_row(**overrides):
    """Build a complete ic_scores-style row with defaults."""
    defaults = {
        "ticker": "AAPL",
        "date": date(2025, 1, 15),
        "overall_score": 75.0,
        "rating": "Buy",
        "value_score": 70.0,
        "growth_score": 80.0,
        "profitability_score": 85.0,
        "financial_health_score": 72.0,
        "momentum_score": 65.0,
        "analyst_consensus_score": 78.0,
        "insider_activity_score": 60.0,
        "institutional_score": 74.0,
        "news_sentiment_score": 55.0,
        "technical_score": 68.0,
        "sector_percentile": 82.0,
        "confidence_level": "High",
        "data_completeness": 0.95,
    }
    defaults.update(overrides)
    return _make_row(defaults)


def _financial_row(**overrides):
    """Build a financials-style row with reasonable defaults."""
    defaults = {
        "ticker": "AAPL",
        "period_end_date": date(2024, 12, 31),
        "fiscal_year": 2024,
        "fiscal_quarter": None,
        "filing_date": date(2025, 2, 1),
        "revenue": 400_000_000_000,
        "cost_of_revenue": 220_000_000_000,
        "gross_profit": 180_000_000_000,
        "operating_expenses": 50_000_000_000,
        "operating_income": 130_000_000_000,
        "net_income": 100_000_000_000,
        "eps_basic": 6.50,
        "eps_diluted": 6.42,
        "shares_outstanding": 15_500_000_000,
        "total_assets": 350_000_000_000,
        "total_liabilities": 280_000_000_000,
        "shareholders_equity": 70_000_000_000,
        "cash_and_equivalents": 30_000_000_000,
        "short_term_debt": 10_000_000_000,
        "long_term_debt": 100_000_000_000,
        "operating_cash_flow": 110_000_000_000,
        "investing_cash_flow": -20_000_000_000,
        "financing_cash_flow": -90_000_000_000,
        "free_cash_flow": 95_000_000_000,
        "capex": -15_000_000_000,
        "gross_margin": 0.45,
        "operating_margin": 0.325,
        "net_margin": 0.25,
        "roe": 1.43,
        "roa": 0.286,
        "current_ratio": 1.1,
        "debt_to_equity": 1.43,
        "quick_ratio": 0.85,
        "pe_ratio": 30.0,
        "pb_ratio": 45.0,
        "ps_ratio": 8.5,
        "statement_type": "10-K",
        # Prior year fields for YoY calculation
        "prior_revenue": 380_000_000_000,
        "prior_net_income": 95_000_000_000,
        "prior_eps_diluted": 6.10,
        "prior_operating_income": 120_000_000_000,
        "prior_eps": 6.10,
    }
    defaults.update(overrides)
    return _make_row(defaults)


def _risk_row(**overrides):
    """Build a risk_metrics-style row with defaults."""
    defaults = {
        "time": datetime(2025, 1, 15, 12, 0, 0),
        "ticker": "AAPL",
        "period": "1Y",
        "alpha": 5.2,
        "beta": 1.15,
        "sharpe_ratio": 0.85,
        "sortino_ratio": 1.10,
        "std_dev": 22.5,
        "max_drawdown": -15.3,
        "var_5": -3.8,
        "annualized_return": 18.5,
        "downside_deviation": 14.2,
        "data_points": 252,
        "calculation_date": date(2025, 1, 15),
    }
    defaults.update(overrides)
    return _make_row(defaults)


def _technical_row(**overrides):
    """Build a technical_indicators-style row with defaults.

    The SQL query in the endpoint selects these exact columns:
    time, ticker, close, sma_20, sma_50, sma_200, ema_12, ema_26,
    rsi_14, macd, macd_signal, macd_histogram, bb_upper, bb_middle,
    bb_lower, atr_14, adx_14, stoch_k, stoch_d
    """
    defaults = {
        "time": datetime(2025, 1, 15, 12, 0, 0),
        "ticker": "AAPL",
        "close": 195.50,
        "sma_20": 192.0,
        "sma_50": 188.0,
        "sma_200": 175.0,
        "ema_12": 193.5,
        "ema_26": 190.0,
        "rsi_14": 62.0,
        "macd": 1.5,
        "macd_signal": 1.2,
        "macd_histogram": 0.3,
        "bb_upper": 200.0,
        "bb_middle": 192.0,
        "bb_lower": 184.0,
        "atr_14": 3.5,
        "adx_14": 25.0,
        "stoch_k": 72.0,
        "stoch_d": 68.0,
    }
    defaults.update(overrides)
    return _make_row(defaults)


def _news_row(**overrides):
    """Build a news_articles-style row with defaults."""
    defaults = {
        "id": 1,
        "title": "Apple Reports Record Q4 Earnings",
        "url": "https://example.com/article/1",
        "source": "Bloomberg",
        "published_at": datetime(2025, 1, 14, 9, 0, 0),
        "summary": "Apple beat expectations.",
        "author": "John Doe",
        "tickers": ["AAPL", "MSFT"],
        "sentiment_score": 75.0,
        "sentiment_label": "Positive",
        "relevance_score": 90.0,
        "image_url": "https://example.com/img.png",
    }
    defaults.update(overrides)
    return _make_row(defaults)


def _ttm_row(**overrides):
    """Build a ttm_financials-style row with defaults."""
    defaults = {
        "ticker": "AAPL",
        "ttm_period_end": date(2024, 12, 31),
        "calculation_date": date(2025, 1, 15),
        "revenue": 400_000_000_000,
        "cost_of_revenue": 220_000_000_000,
        "gross_profit": 180_000_000_000,
        "operating_expenses": 50_000_000_000,
        "operating_income": 130_000_000_000,
        "net_income": 100_000_000_000,
        "eps_basic": 6.50,
        "eps_diluted": 6.42,
        "shares_outstanding": 15_500_000_000,
        "total_assets": 350_000_000_000,
        "total_liabilities": 280_000_000_000,
        "shareholders_equity": 70_000_000_000,
        "cash_and_equivalents": 30_000_000_000,
        "short_term_debt": 10_000_000_000,
        "long_term_debt": 100_000_000_000,
        "operating_cash_flow": 110_000_000_000,
        "investing_cash_flow": -20_000_000_000,
        "financing_cash_flow": -90_000_000_000,
        "free_cash_flow": 95_000_000_000,
        "capex": -15_000_000_000,
        "prior_revenue": 380_000_000_000,
        "prior_net_income": 95_000_000_000,
        "prior_eps_diluted": 6.10,
        "prior_operating_income": 120_000_000_000,
    }
    defaults.update(overrides)
    return _make_row(defaults)


def _meta_row(**overrides):
    """Build a tickers metadata row."""
    defaults = {"name": "Apple Inc."}
    defaults.update(overrides)
    return _make_row(defaults)


# -------------------------------------------------------------------
# Fixtures
# -------------------------------------------------------------------


@pytest.fixture
def mock_session():
    """Create a mock AsyncSession."""
    return AsyncMock()


@pytest.fixture
def client(mock_session):
    """Create a TestClient with database dependency overridden."""

    async def _mock_get_session():
        yield mock_session

    app.dependency_overrides[get_session] = _mock_get_session
    c = TestClient(app, raise_server_exceptions=False)
    yield c
    app.dependency_overrides.clear()


# =====================================================================
# Root endpoint
# =====================================================================


class TestRootEndpoint:
    def test_root(self, client):
        response = client.get("/")
        assert response.status_code == 200
        data = response.json()
        assert data["service"] == "IC Score API"
        assert "version" in data
        assert "endpoints" in data

    def test_root_contains_expected_endpoints(self, client):
        response = client.get("/")
        data = response.json()
        endpoints = data["endpoints"]
        assert "health" in endpoints
        assert "score" in endpoints
        assert "top" in endpoints
        assert "financials_annual" in endpoints


# =====================================================================
# Health endpoint
# =====================================================================


class TestHealthEndpoint:
    def test_health_check_healthy(self, client):
        """Health check with healthy database."""
        with patch("api.main.get_database") as mock_db:
            mock_db.return_value.health_check = AsyncMock(
                return_value={
                    "status": "healthy",
                    "connected": True,
                    "database": "investorcenter_db",
                    "host": "localhost",
                }
            )
            response = client.get("/health")

        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "healthy"
        assert data["database_connected"] is True
        assert "timestamp" in data

    def test_health_check_unhealthy(self, client):
        """Health check with unhealthy database."""
        with patch("api.main.get_database") as mock_db:
            mock_db.return_value.health_check = AsyncMock(
                return_value={
                    "status": "unhealthy",
                    "connected": False,
                    "error": "Connection refused",
                }
            )
            response = client.get("/health")

        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "unhealthy"
        assert data["database_connected"] is False


# =====================================================================
# GET /api/scores/{ticker}
# =====================================================================


class TestGetICScore:
    def test_score_found(self, client, mock_session):
        row = _ic_score_row()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = row
        mock_session.execute.return_value = mock_result

        response = client.get("/api/scores/AAPL")
        assert response.status_code == 200
        data = response.json()
        assert data["ticker"] == "AAPL"
        assert data["overall_score"] == 75.0
        assert data["rating"] == "Buy"
        assert data["confidence_level"] == "High"
        assert data["data_completeness"] == 0.95

    def test_score_not_found(self, client, mock_session):
        mock_result = MagicMock()
        mock_result.fetchone.return_value = None
        mock_session.execute.return_value = mock_result

        response = client.get("/api/scores/ZZZZ")
        assert response.status_code == 404
        data = response.json()
        assert "not found" in data["detail"].lower()

    def test_ticker_case_insensitive(self, client, mock_session):
        """Lowercase ticker should be uppercased."""
        row = _ic_score_row()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = row
        mock_session.execute.return_value = mock_result

        response = client.get("/api/scores/aapl")
        assert response.status_code == 200
        data = response.json()
        assert data["ticker"] == "AAPL"

    def test_score_all_factor_scores(self, client, mock_session):
        """All factor scores should be included in the response."""
        row = _ic_score_row()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = row
        mock_session.execute.return_value = mock_result

        response = client.get("/api/scores/AAPL")
        data = response.json()
        assert data["value_score"] == 70.0
        assert data["growth_score"] == 80.0
        assert data["profitability_score"] == 85.0
        assert data["financial_health_score"] == 72.0
        assert data["momentum_score"] == 65.0
        assert data["technical_score"] == 68.0


# =====================================================================
# GET /api/scores/{ticker}/history
# =====================================================================


class TestGetScoreHistory:
    def test_history_found(self, client, mock_session):
        rows = [
            _ic_score_row(date=date(2025, 1, 15)),
            _ic_score_row(
                date=date(2025, 1, 14), overall_score=73.0
            ),
        ]
        mock_result = MagicMock()
        mock_result.fetchall.return_value = rows
        mock_session.execute.return_value = mock_result

        response = client.get("/api/scores/AAPL/history?days=7")
        assert response.status_code == 200
        data = response.json()
        assert data["ticker"] == "AAPL"
        assert len(data["scores"]) == 2

    def test_history_not_found(self, client, mock_session):
        mock_result = MagicMock()
        mock_result.fetchall.return_value = []
        mock_session.execute.return_value = mock_result

        response = client.get("/api/scores/ZZZZ/history")
        assert response.status_code == 404

    def test_history_days_validation_too_low(
        self, client, mock_session
    ):
        response = client.get("/api/scores/AAPL/history?days=0")
        assert response.status_code == 422

    def test_history_days_validation_too_high(
        self, client, mock_session
    ):
        response = client.get("/api/scores/AAPL/history?days=400")
        assert response.status_code == 422

    def test_history_default_days(self, client, mock_session):
        """Default days param is 90 when not specified."""
        rows = [_ic_score_row()]
        mock_result = MagicMock()
        mock_result.fetchall.return_value = rows
        mock_session.execute.return_value = mock_result

        response = client.get("/api/scores/AAPL/history")
        assert response.status_code == 200


# =====================================================================
# GET /api/scores/top  (handler tested directly -- route is shadowed)
# =====================================================================


class TestGetTopScores:
    @pytest.mark.asyncio
    async def test_top_scores_returns_list(self):
        """Call get_top_scores handler directly."""
        mock_session = AsyncMock()
        rows = [
            _ic_score_row(ticker="AAPL", overall_score=90.0),
            _ic_score_row(ticker="MSFT", overall_score=88.0),
        ]
        mock_result = MagicMock()
        mock_result.fetchall.return_value = rows
        mock_session.execute.return_value = mock_result

        result = await get_top_scores(
            limit=10,
            sector=None,
            min_score=None,
            min_confidence=None,
            session=mock_session,
        )
        assert result.count == 2
        assert len(result.stocks) == 2

    @pytest.mark.asyncio
    async def test_top_scores_empty_result(self):
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchall.return_value = []
        mock_session.execute.return_value = mock_result

        result = await get_top_scores(
            limit=50,
            sector=None,
            min_score=None,
            min_confidence=None,
            session=mock_session,
        )
        assert result.count == 0
        assert result.stocks == []

    @pytest.mark.asyncio
    async def test_top_scores_with_sector_filter(self):
        mock_session = AsyncMock()
        rows = [_ic_score_row(ticker="AAPL", overall_score=92.0)]
        mock_result = MagicMock()
        mock_result.fetchall.return_value = rows
        mock_session.execute.return_value = mock_result

        result = await get_top_scores(
            limit=50,
            sector="Technology",
            min_score=80.0,
            min_confidence=None,
            session=mock_session,
        )
        assert result.count == 1

    @pytest.mark.asyncio
    async def test_top_scores_with_min_confidence(self):
        mock_session = AsyncMock()
        rows = [_ic_score_row(ticker="AAPL")]
        mock_result = MagicMock()
        mock_result.fetchall.return_value = rows
        mock_session.execute.return_value = mock_result

        result = await get_top_scores(
            limit=50,
            sector=None,
            min_score=None,
            min_confidence="High",
            session=mock_session,
        )
        assert result.count == 1


# =====================================================================
# GET /api/scores/screener (handler tested directly)
# =====================================================================


class TestScreener:
    @pytest.mark.asyncio
    async def test_screener_no_filters(self):
        mock_session = AsyncMock()
        rows = [
            _ic_score_row(ticker="AAPL"),
            _ic_score_row(ticker="MSFT"),
        ]
        mock_result = MagicMock()
        mock_result.fetchall.return_value = rows
        mock_session.execute.return_value = mock_result

        result = await screener(
            min_value=None,
            min_growth=None,
            min_profitability=None,
            min_financial_health=None,
            min_momentum=None,
            min_technical=None,
            min_overall=None,
            sector=None,
            limit=100,
            session=mock_session,
        )
        assert result.count == 2

    @pytest.mark.asyncio
    async def test_screener_with_factor_filters(self):
        mock_session = AsyncMock()
        rows = [_ic_score_row(ticker="AAPL")]
        mock_result = MagicMock()
        mock_result.fetchall.return_value = rows
        mock_session.execute.return_value = mock_result

        result = await screener(
            min_value=70.0,
            min_growth=60.0,
            min_profitability=None,
            min_financial_health=None,
            min_momentum=None,
            min_technical=None,
            min_overall=80.0,
            sector=None,
            limit=100,
            session=mock_session,
        )
        assert result.count == 1

    @pytest.mark.asyncio
    async def test_screener_with_sector(self):
        mock_session = AsyncMock()
        rows = [_ic_score_row(ticker="AAPL")]
        mock_result = MagicMock()
        mock_result.fetchall.return_value = rows
        mock_session.execute.return_value = mock_result

        result = await screener(
            min_value=None,
            min_growth=None,
            min_profitability=None,
            min_financial_health=None,
            min_momentum=None,
            min_technical=None,
            min_overall=None,
            sector="Technology",
            limit=100,
            session=mock_session,
        )
        assert result.count == 1


# =====================================================================
# GET /api/financials/{ticker}/annual
# =====================================================================


class TestAnnualFinancials:
    def test_annual_financials_found(self, client, mock_session):
        fin_row = _financial_row()
        meta = _meta_row()

        mock_result_fin = MagicMock()
        mock_result_fin.fetchall.return_value = [fin_row]

        mock_result_meta = MagicMock()
        mock_result_meta.fetchone.return_value = meta

        mock_session.execute.side_effect = [
            mock_result_fin,
            mock_result_meta,
        ]

        response = client.get("/api/financials/AAPL/annual")
        assert response.status_code == 200
        data = response.json()
        assert data["ticker"] == "AAPL"
        assert data["timeframe"] == "annual"
        assert len(data["periods"]) == 1
        assert data["metadata"]["company_name"] == "Apple Inc."

    def test_annual_financials_not_found(self, client, mock_session):
        mock_result = MagicMock()
        mock_result.fetchall.return_value = []
        mock_session.execute.return_value = mock_result

        response = client.get("/api/financials/ZZZZ/annual")
        assert response.status_code == 404

    def test_annual_yoy_change_calculated(self, client, mock_session):
        """When prior year data exists, YoY changes should appear."""
        fin_row = _financial_row()
        meta = _meta_row()

        mock_result_fin = MagicMock()
        mock_result_fin.fetchall.return_value = [fin_row]

        mock_result_meta = MagicMock()
        mock_result_meta.fetchone.return_value = meta

        mock_session.execute.side_effect = [
            mock_result_fin,
            mock_result_meta,
        ]

        response = client.get("/api/financials/AAPL/annual")
        data = response.json()
        period = data["periods"][0]
        assert period["yoy_change"] is not None
        assert "revenue" in period["yoy_change"]


# =====================================================================
# GET /api/financials/{ticker}/ttm
# =====================================================================


class TestTTMFinancials:
    def test_ttm_financials_found(self, client, mock_session):
        ttm_row = _ttm_row()
        meta = _meta_row()

        mock_result_ttm = MagicMock()
        mock_result_ttm.fetchall.return_value = [ttm_row]

        mock_result_meta = MagicMock()
        mock_result_meta.fetchone.return_value = meta

        mock_session.execute.side_effect = [
            mock_result_ttm,
            mock_result_meta,
        ]

        response = client.get("/api/financials/AAPL/ttm")
        assert response.status_code == 200
        data = response.json()
        assert data["ticker"] == "AAPL"
        assert data["timeframe"] == "ttm"
        assert len(data["periods"]) == 1

    def test_ttm_financials_not_found(self, client, mock_session):
        mock_result = MagicMock()
        mock_result.fetchall.return_value = []
        mock_session.execute.return_value = mock_result

        response = client.get("/api/financials/ZZZZ/ttm")
        assert response.status_code == 404

    def test_ttm_margin_calculation(self, client, mock_session):
        """TTM endpoint should calculate margins from raw data."""
        ttm_row = _ttm_row()
        meta = _meta_row()

        mock_result_ttm = MagicMock()
        mock_result_ttm.fetchall.return_value = [ttm_row]

        mock_result_meta = MagicMock()
        mock_result_meta.fetchone.return_value = meta

        mock_session.execute.side_effect = [
            mock_result_ttm,
            mock_result_meta,
        ]

        response = client.get("/api/financials/AAPL/ttm")
        data = response.json()
        period = data["periods"][0]
        # gross_margin = gross_profit / revenue = 180B / 400B = 0.45
        assert period["gross_margin"] == pytest.approx(0.45, abs=0.01)


# =====================================================================
# GET /api/metrics/{ticker}
# =====================================================================


class TestFinancialMetrics:
    def test_metrics_found(self, client, mock_session):
        row = _financial_row()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = row
        mock_session.execute.return_value = mock_result

        response = client.get("/api/metrics/AAPL")
        assert response.status_code == 200
        data = response.json()
        assert data["ticker"] == "AAPL"
        assert data["gross_margin"] == 0.45
        assert data["net_margin"] == 0.25
        assert data["revenue_growth_yoy"] is not None

    def test_metrics_not_found(self, client, mock_session):
        mock_result = MagicMock()
        mock_result.fetchone.return_value = None
        mock_session.execute.return_value = mock_result

        response = client.get("/api/metrics/ZZZZ")
        assert response.status_code == 404

    def test_metrics_yoy_revenue_growth(self, client, mock_session):
        """Revenue growth = (400B - 380B) / 380B * 100 = 5.26%."""
        row = _financial_row()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = row
        mock_session.execute.return_value = mock_result

        response = client.get("/api/metrics/AAPL")
        data = response.json()
        expected = (
            (400_000_000_000 - 380_000_000_000)
            / 380_000_000_000
            * 100
        )
        assert data["revenue_growth_yoy"] == pytest.approx(
            expected, abs=0.1
        )


# =====================================================================
# GET /api/risk/{ticker}
# =====================================================================


class TestRiskMetrics:
    def test_risk_metrics_found(self, client, mock_session):
        row = _risk_row()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = row
        mock_session.execute.return_value = mock_result

        response = client.get("/api/risk/AAPL?period=1Y")
        assert response.status_code == 200
        data = response.json()
        assert data["ticker"] == "AAPL"
        assert data["beta"] == 1.15
        assert data["sharpe_ratio"] == 0.85
        assert data["max_drawdown"] == -15.3

    def test_risk_metrics_not_found(self, client, mock_session):
        mock_result = MagicMock()
        mock_result.fetchone.return_value = None
        mock_session.execute.return_value = mock_result

        response = client.get("/api/risk/ZZZZ")
        assert response.status_code == 404

    def test_risk_metrics_default_period(self, client, mock_session):
        """Default period should be 1Y."""
        row = _risk_row()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = row
        mock_session.execute.return_value = mock_result

        response = client.get("/api/risk/AAPL")
        assert response.status_code == 200
        data = response.json()
        assert data["period"] == "1Y"

    def test_risk_all_fields_present(self, client, mock_session):
        """Verify all risk metric fields are returned."""
        row = _risk_row()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = row
        mock_session.execute.return_value = mock_result

        response = client.get("/api/risk/AAPL")
        data = response.json()
        assert data["alpha"] == 5.2
        assert data["sortino_ratio"] == 1.10
        assert data["volatility"] == 22.5
        assert data["var_95"] == -3.8
        assert data["annualized_return"] == 18.5
        assert data["data_points"] == 252


# =====================================================================
# GET /api/technical/{ticker} (tested via handler directly)
# =====================================================================


class TestTechnicalIndicators:
    def test_technical_found(self, client, mock_session):
        """Technical indicators returned via HTTP TestClient."""
        row = _technical_row()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = row
        mock_session.execute.return_value = mock_result

        response = client.get("/api/technical/AAPL")
        assert response.status_code == 200
        data = response.json()
        assert data["ticker"] == "AAPL"
        assert data["rsi_14"] == 62.0
        assert data["sma_50"] == 188.0
        assert data["bb_upper"] == 200.0
        assert data["close_price"] == 195.50

    def test_technical_not_found(self, client, mock_session):
        mock_result = MagicMock()
        mock_result.fetchone.return_value = None
        mock_session.execute.return_value = mock_result

        response = client.get("/api/technical/ZZZZ")
        assert response.status_code == 404

    def test_technical_all_fields(self, client, mock_session):
        """Verify all expected indicator fields via HTTP."""
        row = _technical_row()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = row
        mock_session.execute.return_value = mock_result

        response = client.get("/api/technical/AAPL")
        assert response.status_code == 200
        data = response.json()
        assert data["sma_20"] == 192.0
        assert data["sma_200"] == 175.0
        assert data["ema_12"] == 193.5
        assert data["ema_26"] == 190.0
        assert data["macd"] == 1.5
        assert data["macd_signal"] == 1.2
        assert data["macd_histogram"] == 0.3
        assert data["bb_middle"] == 192.0
        assert data["bb_lower"] == 184.0
        assert data["atr_14"] == 3.5
        assert data["adx_14"] == 25.0
        assert data["stoch_k"] == 72.0
        assert data["stoch_d"] == 68.0

    def test_technical_with_null_fields(self, client, mock_session):
        """Technical indicators with some None values."""
        row = _technical_row(
            sma_200=None, atr_14=None, adx_14=None
        )
        mock_result = MagicMock()
        mock_result.fetchone.return_value = row
        mock_session.execute.return_value = mock_result

        response = client.get("/api/technical/AAPL")
        assert response.status_code == 200
        data = response.json()
        assert data["sma_200"] is None
        assert data["atr_14"] is None
        assert data["adx_14"] is None
        assert data["rsi_14"] == 62.0  # Non-null field still works


# =====================================================================
# GET /api/news/{ticker}
# =====================================================================


class TestNewsEndpoint:
    def test_news_found(self, client, mock_session):
        rows = [
            _news_row(id=1),
            _news_row(id=2, title="Apple Launches New Product"),
        ]
        mock_result = MagicMock()
        mock_result.fetchall.return_value = rows
        mock_session.execute.return_value = mock_result

        response = client.get("/api/news/AAPL")
        assert response.status_code == 200
        data = response.json()
        assert data["ticker"] == "AAPL"
        assert data["count"] == 2
        assert len(data["articles"]) == 2

    def test_news_empty(self, client, mock_session):
        """No articles should return empty list, not 404."""
        mock_result = MagicMock()
        mock_result.fetchall.return_value = []
        mock_session.execute.return_value = mock_result

        response = client.get("/api/news/ZZZZ")
        assert response.status_code == 200
        data = response.json()
        assert data["count"] == 0
        assert data["articles"] == []

    def test_news_limit_too_low(self, client, mock_session):
        response = client.get("/api/news/AAPL?limit=0")
        assert response.status_code == 422

    def test_news_limit_too_high(self, client, mock_session):
        response = client.get("/api/news/AAPL?limit=101")
        assert response.status_code == 422

    def test_news_days_validation(self, client, mock_session):
        """days param must be 1-365."""
        response = client.get("/api/news/AAPL?days=0")
        assert response.status_code == 422

        response = client.get("/api/news/AAPL?days=400")
        assert response.status_code == 422


# =====================================================================
# Backtest config endpoint
# =====================================================================


class TestBacktestConfig:
    def test_default_config(self, client):
        response = client.get("/api/backtest/config/default")
        assert response.status_code == 200
        data = response.json()
        assert "start_date" in data
        assert "end_date" in data
        assert data["rebalance_frequency"] == "monthly"
        assert data["universe"] == "sp500"
        assert data["benchmark"] == "SPY"
        assert data["transaction_cost_bps"] == 10.0
        assert data["slippage_bps"] == 5.0

    def test_default_config_excludes(self, client):
        """Default config should not exclude sectors."""
        response = client.get("/api/backtest/config/default")
        data = response.json()
        assert data["exclude_financials"] is False
        assert data["exclude_utilities"] is False


# =====================================================================
# Backtest job status endpoints
# =====================================================================


class TestBacktestJobs:
    def test_job_not_found(self, client):
        response = client.get(
            "/api/backtest/jobs/nonexistent-job-id"
        )
        assert response.status_code == 404

    def test_job_result_not_found(self, client):
        response = client.get(
            "/api/backtest/jobs/nonexistent-job-id/result"
        )
        assert response.status_code == 404

    def test_job_result_not_completed(self, client):
        """Result endpoint should 400 if job is not completed."""
        from api.main import _backtest_jobs

        _backtest_jobs["test-job-123"] = {
            "id": "test-job-123",
            "status": "running",
            "config": {},
            "created_at": datetime.now().isoformat(),
            "started_at": datetime.now().isoformat(),
            "completed_at": None,
            "result": None,
            "error": None,
        }

        response = client.get(
            "/api/backtest/jobs/test-job-123/result"
        )
        assert response.status_code == 400

        # Clean up
        del _backtest_jobs["test-job-123"]

    def test_job_status_returns_metadata(self, client):
        """Job status endpoint should return job metadata."""
        from api.main import _backtest_jobs

        _backtest_jobs["test-job-456"] = {
            "id": "test-job-456",
            "status": "pending",
            "config": {},
            "created_at": "2025-01-15T12:00:00",
            "started_at": None,
            "completed_at": None,
            "result": None,
            "error": None,
        }

        response = client.get(
            "/api/backtest/jobs/test-job-456"
        )
        assert response.status_code == 200
        data = response.json()
        assert data["job_id"] == "test-job-456"
        assert data["status"] == "pending"

        # Clean up
        del _backtest_jobs["test-job-456"]
