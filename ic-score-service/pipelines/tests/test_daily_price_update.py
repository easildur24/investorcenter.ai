"""Tests for the daily price update pipeline.

Validates Polygon.io price fetching, bar transformation, and database
storage logic with mocked asyncpg pool and PolygonClient.
"""

from datetime import datetime
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# ---------------------------------------------------------------------------
# Patch PolygonClient and asyncpg.create_pool before importing the pipeline.
# The module reads POLYGON_API_KEY at import time via PolygonClient.__init__.
# ---------------------------------------------------------------------------

_polygon_patcher = patch(
    "pipelines.daily_price_update.PolygonClient",
    return_value=MagicMock(),
)
_polygon_patcher.start()

from pipelines.daily_price_update import DailyPriceUpdate  # noqa: E402


@pytest.fixture
def mock_polygon():
    """Provide a fresh MagicMock PolygonClient for each test."""
    client = MagicMock()
    with patch(
        "pipelines.daily_price_update.PolygonClient",
        return_value=client,
    ):
        yield client


@pytest.fixture
def mock_pool():
    """Provide a mocked asyncpg connection pool + connection.

    asyncpg's pool.acquire() returns an async context manager
    directly (not a coroutine), so we use MagicMock for the pool
    and configure acquire() to return a context manager stub.
    """
    conn = AsyncMock()

    # Build an async context manager that yields conn
    acm = MagicMock()
    acm.__aenter__ = AsyncMock(return_value=conn)
    acm.__aexit__ = AsyncMock(return_value=False)

    pool = MagicMock()
    pool.acquire.return_value = acm
    pool.close = AsyncMock()
    return pool, conn


def _make_bar(
    ts=1708128000000,
    o=180.0,
    h=185.5,
    l=179.0,
    c=185.50,
    v=45000000,
    vw=182.3,
):
    """Build a single Polygon price bar dict."""
    return {
        "t": ts,
        "o": o,
        "h": h,
        "l": l,
        "c": c,
        "v": v,
        "vw": vw,
    }


# ==================================================================
# Initialisation
# ==================================================================


class TestDailyPriceUpdateInit:
    def test_default_params(self, mock_polygon):
        updater = DailyPriceUpdate()
        assert updater.days == 5
        assert updater.batch_size == 100

    def test_custom_params(self, mock_polygon):
        updater = DailyPriceUpdate(days=10, batch_size=50)
        assert updater.days == 10
        assert updater.batch_size == 50

    def test_date_range_calculated(self, mock_polygon):
        updater = DailyPriceUpdate(days=7)
        assert updater.from_date is not None
        assert updater.to_date is not None
        # from_date should be before to_date
        assert updater.from_date < updater.to_date

    def test_stats_initialised_to_zero(self, mock_polygon):
        updater = DailyPriceUpdate()
        assert updater.processed == 0
        assert updater.success == 0
        assert updater.errors == 0
        assert updater.total_rows_inserted == 0


# ==================================================================
# get_tickers
# ==================================================================


class TestGetTickers:
    @pytest.mark.asyncio
    async def test_returns_symbols(self, mock_pool, mock_polygon):
        pool, conn = mock_pool
        conn.fetch.return_value = [
            {"symbol": "AAPL"},
            {"symbol": "MSFT"},
            {"symbol": "GOOG"},
        ]
        updater = DailyPriceUpdate()
        updater.pool = pool

        tickers = await updater.get_tickers()

        assert tickers == ["AAPL", "MSFT", "GOOG"]
        conn.fetch.assert_called_once()

    @pytest.mark.asyncio
    async def test_respects_limit(self, mock_pool, mock_polygon):
        pool, conn = mock_pool
        conn.fetch.return_value = [{"symbol": "AAPL"}]
        updater = DailyPriceUpdate()
        updater.pool = pool

        await updater.get_tickers(limit=1)

        # The SQL should contain LIMIT when limit is provided
        call_args = conn.fetch.call_args
        assert "LIMIT" in call_args[0][0]


# ==================================================================
# fetch_prices
# ==================================================================


class TestFetchPrices:
    def test_success(self, mock_polygon):
        mock_polygon.get_aggregates.return_value = [_make_bar()]
        updater = DailyPriceUpdate()

        bars = updater.fetch_prices("AAPL")

        assert bars is not None
        assert len(bars) == 1
        mock_polygon.get_aggregates.assert_called_once()

    def test_api_error_returns_none(self, mock_polygon):
        mock_polygon.get_aggregates.side_effect = Exception(
            "API rate limit"
        )
        updater = DailyPriceUpdate()

        bars = updater.fetch_prices("AAPL")

        assert bars is None

    def test_passes_correct_parameters(self, mock_polygon):
        mock_polygon.get_aggregates.return_value = []
        updater = DailyPriceUpdate(days=7)

        updater.fetch_prices("TSLA")

        call_kwargs = mock_polygon.get_aggregates.call_args
        assert call_kwargs[1]["ticker"] == "TSLA"
        assert call_kwargs[1]["timespan"] == "day"
        assert call_kwargs[1]["multiplier"] == 1


# ==================================================================
# store_prices
# ==================================================================


class TestStorePrices:
    @pytest.mark.asyncio
    async def test_empty_bars_returns_zero(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        updater = DailyPriceUpdate()
        updater.pool = pool

        count = await updater.store_prices("AAPL", [])

        assert count == 0

    @pytest.mark.asyncio
    async def test_valid_bars_stored(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        updater = DailyPriceUpdate()
        updater.pool = pool

        bars = [_make_bar(), _make_bar(ts=1708214400000)]
        count = await updater.store_prices("AAPL", bars)

        assert count == 2
        conn.executemany.assert_called_once()

    @pytest.mark.asyncio
    async def test_invalid_bar_skipped(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        updater = DailyPriceUpdate()
        updater.pool = pool

        # Bar missing required 't' key
        bars = [{"o": 100}, _make_bar()]
        count = await updater.store_prices("AAPL", bars)

        # Only the valid bar should be stored
        assert count == 1


# ==================================================================
# process_ticker
# ==================================================================


class TestProcessTicker:
    @pytest.mark.asyncio
    async def test_success_increments_stats(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        mock_polygon.get_aggregates.return_value = [_make_bar()]
        updater = DailyPriceUpdate()
        updater.pool = pool

        result = await updater.process_ticker("AAPL")

        assert result is True
        assert updater.processed == 1
        assert updater.success == 1
        assert updater.errors == 0

    @pytest.mark.asyncio
    async def test_fetch_error_increments_errors(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        mock_polygon.get_aggregates.side_effect = Exception("fail")
        updater = DailyPriceUpdate()
        updater.pool = pool

        result = await updater.process_ticker("AAPL")

        assert result is False
        assert updater.errors == 1

    @pytest.mark.asyncio
    async def test_empty_bars_no_error(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        mock_polygon.get_aggregates.return_value = []
        updater = DailyPriceUpdate()
        updater.pool = pool

        result = await updater.process_ticker("AAPL")

        assert result is True
        assert updater.errors == 0
