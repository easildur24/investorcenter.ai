"""Tests for the historical price backfill pipeline.

Validates initialization defaults, date range calculation, Polygon
price fetching, database storage, resume/skip logic, and ticker
retrieval with mocked asyncpg pool and PolygonClient.
"""

from datetime import datetime, timedelta
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# Patch PolygonClient before importing the pipeline module.
_polygon_patcher = patch(
    "pipelines.historical_price_backfill.PolygonClient",
    return_value=MagicMock(),
)
_polygon_patcher.start()

from pipelines.historical_price_backfill import (  # noqa: E402
    HistoricalPriceBackfill,
)


@pytest.fixture
def mock_polygon():
    """Provide a fresh MagicMock PolygonClient for each test."""
    client = MagicMock()
    with patch(
        "pipelines.historical_price_backfill.PolygonClient",
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


class TestHistoricalPriceBackfillInit:
    def test_default_params(self, mock_polygon):
        backfill = HistoricalPriceBackfill()
        assert backfill.years == 10
        assert backfill.batch_size == 100
        assert backfill.resume is False

    def test_custom_params(self, mock_polygon):
        backfill = HistoricalPriceBackfill(
            years=15, batch_size=50, resume=True
        )
        assert backfill.years == 15
        assert backfill.batch_size == 50
        assert backfill.resume is True

    def test_date_range_spans_years(self, mock_polygon):
        backfill = HistoricalPriceBackfill(years=5)
        from_dt = datetime.strptime(
            backfill.from_date, "%Y-%m-%d"
        )
        to_dt = datetime.strptime(backfill.to_date, "%Y-%m-%d")
        diff_days = (to_dt - from_dt).days
        # 5 years * 365 = 1825 days, allow some slack
        assert diff_days >= 1820
        assert diff_days <= 1830

    def test_stats_initialised_to_zero(self, mock_polygon):
        backfill = HistoricalPriceBackfill()
        assert backfill.processed == 0
        assert backfill.success == 0
        assert backfill.skipped == 0
        assert backfill.errors == 0
        assert backfill.total_rows_inserted == 0


# ==================================================================
# get_tickers
# ==================================================================


class TestGetTickers:
    @pytest.mark.asyncio
    async def test_returns_symbols(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        conn.fetch.return_value = [
            {"symbol": "AAPL"},
            {"symbol": "MSFT"},
        ]
        backfill = HistoricalPriceBackfill()
        backfill.pool = pool

        tickers = await backfill.get_tickers()

        assert tickers == ["AAPL", "MSFT"]
        conn.fetch.assert_called_once()

    @pytest.mark.asyncio
    async def test_respects_limit(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        conn.fetch.return_value = [{"symbol": "AAPL"}]
        backfill = HistoricalPriceBackfill()
        backfill.pool = pool

        await backfill.get_tickers(limit=1)

        call_args = conn.fetch.call_args
        assert "LIMIT" in call_args[0][0]


# ==================================================================
# fetch_prices
# ==================================================================


class TestFetchPrices:
    def test_success(self, mock_polygon):
        mock_polygon.get_aggregates.return_value = [_make_bar()]
        backfill = HistoricalPriceBackfill()

        bars = backfill.fetch_prices("AAPL")

        assert bars is not None
        assert len(bars) == 1

    def test_api_error_returns_none(self, mock_polygon):
        mock_polygon.get_aggregates.side_effect = Exception(
            "timeout"
        )
        backfill = HistoricalPriceBackfill()

        bars = backfill.fetch_prices("AAPL")

        assert bars is None

    def test_large_limit_for_backfill(self, mock_polygon):
        mock_polygon.get_aggregates.return_value = []
        backfill = HistoricalPriceBackfill()

        backfill.fetch_prices("AAPL")

        call_kwargs = mock_polygon.get_aggregates.call_args[1]
        assert call_kwargs["limit"] == 50000


# ==================================================================
# store_prices
# ==================================================================


class TestStorePrices:
    @pytest.mark.asyncio
    async def test_empty_bars_returns_zero(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        backfill = HistoricalPriceBackfill()
        backfill.pool = pool

        count = await backfill.store_prices("AAPL", [])

        assert count == 0

    @pytest.mark.asyncio
    async def test_valid_bars_stored(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        backfill = HistoricalPriceBackfill()
        backfill.pool = pool

        bars = [_make_bar(), _make_bar(ts=1708214400000)]
        count = await backfill.store_prices("AAPL", bars)

        assert count == 2
        conn.executemany.assert_called_once()


# ==================================================================
# should_skip_ticker (resume logic)
# ==================================================================


class TestShouldSkipTicker:
    @pytest.mark.asyncio
    async def test_no_resume_never_skips(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        backfill = HistoricalPriceBackfill(resume=False)
        backfill.pool = pool

        result = await backfill.should_skip_ticker("AAPL")
        assert result is False

    @pytest.mark.asyncio
    async def test_resume_skips_when_data_old_enough(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        backfill = HistoricalPriceBackfill(
            years=10, resume=True
        )
        backfill.pool = pool

        # Oldest data is 12 years ago (older than requested 10y)
        old_enough = datetime.now() - timedelta(
            days=12 * 365
        )
        conn.fetchrow.return_value = {"oldest": old_enough}

        result = await backfill.should_skip_ticker("AAPL")
        assert result is True

    @pytest.mark.asyncio
    async def test_resume_does_not_skip_when_data_too_recent(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        backfill = HistoricalPriceBackfill(
            years=10, resume=True
        )
        backfill.pool = pool

        # Oldest data is only 2 years ago (not enough for 10y)
        too_recent = datetime.now() - timedelta(days=2 * 365)
        conn.fetchrow.return_value = {"oldest": too_recent}

        result = await backfill.should_skip_ticker("AAPL")
        assert result is False

    @pytest.mark.asyncio
    async def test_resume_does_not_skip_when_no_data(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        backfill = HistoricalPriceBackfill(
            years=10, resume=True
        )
        backfill.pool = pool

        conn.fetchrow.return_value = {"oldest": None}

        result = await backfill.should_skip_ticker("AAPL")
        assert result is False


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
        backfill = HistoricalPriceBackfill()
        backfill.pool = pool

        result = await backfill.process_ticker("AAPL")

        assert result is True
        assert backfill.processed == 1
        assert backfill.success == 1
        assert backfill.errors == 0

    @pytest.mark.asyncio
    async def test_fetch_error_increments_errors(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        mock_polygon.get_aggregates.side_effect = Exception(
            "fail"
        )
        backfill = HistoricalPriceBackfill()
        backfill.pool = pool

        result = await backfill.process_ticker("AAPL")

        assert result is False
        assert backfill.errors == 1

    @pytest.mark.asyncio
    async def test_empty_bars_increments_errors(
        self, mock_pool, mock_polygon
    ):
        pool, conn = mock_pool
        mock_polygon.get_aggregates.return_value = []
        backfill = HistoricalPriceBackfill()
        backfill.pool = pool

        result = await backfill.process_ticker("AAPL")

        # Unlike daily_price_update, backfill treats empty bars
        # as error
        assert result is False
        assert backfill.errors == 1
