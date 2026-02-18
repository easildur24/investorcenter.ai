"""Tests for the benchmark data ingestion pipeline.

Validates benchmark configuration, Polygon price fetching, daily return
calculation, and database storage with mocked PolygonClient and get_database.
"""

import os
from datetime import datetime, timezone
from unittest.mock import AsyncMock, MagicMock, patch

import pandas as pd
import pytest

# Patch dependencies before importing the pipeline module.
with patch(
    "pipelines.benchmark_data_ingestion.get_database"
), patch.dict(
    os.environ, {"POLYGON_API_KEY": "test"}
), patch(
    "pipelines.benchmark_data_ingestion.PolygonClient"
):
    from pipelines.benchmark_data_ingestion import (
        BenchmarkDataIngestion,
    )


@pytest.fixture
def pipeline():
    """Create a BenchmarkDataIngestion with mocked deps."""
    with patch(
        "pipelines.benchmark_data_ingestion.get_database"
    ) as mock_db, patch.dict(
        os.environ, {"POLYGON_API_KEY": "test"}
    ), patch(
        "pipelines.benchmark_data_ingestion.PolygonClient"
    ) as mock_polygon_cls:
        db_instance = MagicMock()
        mock_db.return_value = db_instance
        polygon_instance = MagicMock()
        mock_polygon_cls.return_value = polygon_instance

        p = BenchmarkDataIngestion()
        p._polygon = polygon_instance  # keep ref for tests
        yield p


# ==================================================================
# BENCHMARKS config
# ==================================================================


class TestBenchmarksConfig:
    def test_benchmarks_not_empty(self):
        assert len(BenchmarkDataIngestion.BENCHMARKS) > 0

    def test_spy_is_first_benchmark(self):
        assert BenchmarkDataIngestion.BENCHMARKS[0]["symbol"] == "SPY"

    def test_each_benchmark_has_required_keys(self):
        for bm in BenchmarkDataIngestion.BENCHMARKS:
            assert "symbol" in bm
            assert "name" in bm
            assert "description" in bm


# ==================================================================
# calculate_daily_return
# ==================================================================


class TestCalculateDailyReturn:
    def test_ascending_prices(self, pipeline):
        df = pd.DataFrame(
            {
                "date": pd.date_range("2024-01-01", periods=5),
                "close": [100.0, 102.0, 101.0, 105.0, 110.0],
            }
        )
        result = pipeline.calculate_daily_return(df)

        assert "daily_return" in result.columns
        # First row should be 0
        assert result.iloc[0]["daily_return"] == 0.0
        # Second row: (102 - 100) / 100 * 100 = 2.0
        assert abs(result.iloc[1]["daily_return"] - 2.0) < 0.01

    def test_single_row(self, pipeline):
        df = pd.DataFrame(
            {
                "date": pd.date_range("2024-01-01", periods=1),
                "close": [100.0],
            }
        )
        result = pipeline.calculate_daily_return(df)
        assert "daily_return" in result.columns

    def test_sorts_by_date(self, pipeline):
        df = pd.DataFrame(
            {
                "date": pd.to_datetime(
                    ["2024-01-03", "2024-01-01", "2024-01-02"]
                ),
                "close": [103.0, 100.0, 102.0],
            }
        )
        result = pipeline.calculate_daily_return(df)
        # After sorting, first close should be 100
        assert result.iloc[0]["close"] == 100.0
        # Return on 2nd row: (102-100)/100*100 = 2.0
        assert abs(result.iloc[1]["daily_return"] - 2.0) < 0.01


# ==================================================================
# get_latest_date
# ==================================================================


class TestGetLatestDate:
    @pytest.mark.asyncio
    async def test_returns_date_when_data_exists(self, pipeline):
        mock_session = AsyncMock()
        mock_result = MagicMock()
        expected_dt = datetime(2024, 6, 1, tzinfo=timezone.utc)
        mock_result.fetchone.return_value = (expected_dt,)
        mock_session.execute.return_value = mock_result

        result = await pipeline.get_latest_date(
            "SPY", mock_session
        )
        assert result == expected_dt

    @pytest.mark.asyncio
    async def test_returns_none_when_no_data(self, pipeline):
        mock_session = AsyncMock()
        mock_result = MagicMock()
        mock_result.fetchone.return_value = (None,)
        mock_session.execute.return_value = mock_result

        result = await pipeline.get_latest_date(
            "SPY", mock_session
        )
        assert result is None


# ==================================================================
# fetch_benchmark_data
# ==================================================================


class TestFetchBenchmarkData:
    @pytest.mark.asyncio
    async def test_success(self, pipeline):
        pipeline.polygon_client.get_daily_prices.return_value = [
            {"c": 450.0, "v": 80000000, "date": "2024-06-01"},
        ]

        result = await pipeline.fetch_benchmark_data(
            "SPY", days=7
        )

        assert result is not None
        assert len(result) == 1

    @pytest.mark.asyncio
    async def test_strips_caret_from_symbol(self, pipeline):
        pipeline.polygon_client.get_daily_prices.return_value = [
            {"c": 5000.0, "v": 0, "date": "2024-06-01"},
        ]

        await pipeline.fetch_benchmark_data("^SPX", days=7)

        call_args = (
            pipeline.polygon_client.get_daily_prices.call_args
        )
        # Symbol passed to Polygon should not have ^
        assert call_args[0][0] == "SPX"

    @pytest.mark.asyncio
    async def test_api_error_returns_none(self, pipeline):
        pipeline.polygon_client.get_daily_prices.side_effect = (
            Exception("timeout")
        )

        result = await pipeline.fetch_benchmark_data(
            "SPY", days=7
        )
        assert result is None

    @pytest.mark.asyncio
    async def test_empty_data_returns_none(self, pipeline):
        pipeline.polygon_client.get_daily_prices.return_value = []

        result = await pipeline.fetch_benchmark_data(
            "SPY", days=7
        )
        assert result is None


# ==================================================================
# store_benchmark_data
# ==================================================================


class TestStoreBenchmarkData:
    @pytest.mark.asyncio
    async def test_empty_prices_returns_false(self, pipeline):
        mock_session = AsyncMock()
        result = await pipeline.store_benchmark_data(
            "SPY", [], mock_session
        )
        assert result is False

    @pytest.mark.asyncio
    async def test_valid_prices_stored(self, pipeline):
        mock_session = AsyncMock()
        prices = [
            {
                "c": 450.0,
                "v": 80000000,
                "date": "2024-06-01",
            },
            {
                "c": 452.0,
                "v": 75000000,
                "date": "2024-06-02",
            },
        ]

        result = await pipeline.store_benchmark_data(
            "SPY", prices, mock_session
        )

        assert result is True
        mock_session.execute.assert_called_once()
        mock_session.commit.assert_called_once()
