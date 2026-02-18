"""Tests for the treasury rates ingestion pipeline.

Validates FRED API fetching, observation parsing (including missing '.'
values), series merging, and database storage with mocked requests
and get_database.
"""

import os
from datetime import date
from unittest.mock import AsyncMock, MagicMock, patch

import pandas as pd
import pytest

# Patch get_database and env before importing.
with patch(
    "pipelines.treasury_rates_ingestion.get_database"
), patch.dict(
    os.environ, {"FRED_API_KEY": "test-key"}
):
    from pipelines.treasury_rates_ingestion import (
        TreasuryRatesIngestion,
    )


@pytest.fixture
def pipeline():
    """Create a TreasuryRatesIngestion with mocked DB."""
    with patch(
        "pipelines.treasury_rates_ingestion.get_database"
    ) as mock_db, patch.dict(
        os.environ, {"FRED_API_KEY": "test-key"}
    ):
        db_instance = MagicMock()
        mock_db.return_value = db_instance
        p = TreasuryRatesIngestion(api_key="test-key")
        yield p


def _make_fred_response(observations):
    """Build a FRED API JSON response dict."""
    return {"observations": observations}


# ==================================================================
# FRED_SERIES config
# ==================================================================


class TestFredSeriesConfig:
    def test_has_all_maturities(self):
        series = TreasuryRatesIngestion.FRED_SERIES
        assert "rate_1m" in series
        assert "rate_3m" in series
        assert "rate_6m" in series
        assert "rate_1y" in series
        assert "rate_2y" in series
        assert "rate_10y" in series

    def test_series_ids_are_strings(self):
        for col, series_id in TreasuryRatesIngestion.FRED_SERIES.items():
            assert isinstance(series_id, str)

    def test_dgs3mo_for_3_month(self):
        assert TreasuryRatesIngestion.FRED_SERIES["rate_3m"] == "DGS3MO"


# ==================================================================
# fetch_fred_series
# ==================================================================


class TestFetchFredSeries:
    @patch("pipelines.treasury_rates_ingestion.requests")
    def test_success(self, mock_requests, pipeline):
        mock_response = MagicMock()
        mock_response.json.return_value = _make_fred_response(
            [
                {"date": "2024-06-01", "value": "5.25"},
                {"date": "2024-06-02", "value": "5.30"},
            ]
        )
        mock_response.raise_for_status = MagicMock()
        mock_requests.get.return_value = mock_response

        result = pipeline.fetch_fred_series(
            "DGS3MO", "2024-06-01", "2024-06-02"
        )

        assert result is not None
        assert len(result) == 2
        assert "date" in result.columns
        assert "value" in result.columns

    @patch("pipelines.treasury_rates_ingestion.requests")
    def test_missing_dot_value_becomes_nan(
        self, mock_requests, pipeline
    ):
        """FRED uses '.' for missing values; should be NaN."""
        mock_response = MagicMock()
        mock_response.json.return_value = _make_fred_response(
            [
                {"date": "2024-06-01", "value": "5.25"},
                {"date": "2024-06-02", "value": "."},
            ]
        )
        mock_response.raise_for_status = MagicMock()
        mock_requests.get.return_value = mock_response

        result = pipeline.fetch_fred_series(
            "DGS3MO", "2024-06-01", "2024-06-02"
        )

        assert result is not None
        assert pd.notna(result.iloc[0]["value"])
        assert pd.isna(result.iloc[1]["value"])

    @patch("pipelines.treasury_rates_ingestion.requests")
    def test_api_error_returns_none(
        self, mock_requests, pipeline
    ):
        import requests as req

        mock_requests.exceptions = req.exceptions
        mock_requests.get.side_effect = (
            req.exceptions.ConnectionError("timeout")
        )

        result = pipeline.fetch_fred_series(
            "DGS3MO", "2024-06-01", "2024-06-02"
        )

        assert result is None

    @patch("pipelines.treasury_rates_ingestion.requests")
    def test_no_observations_returns_none(
        self, mock_requests, pipeline
    ):
        mock_response = MagicMock()
        mock_response.json.return_value = {"observations": []}
        mock_response.raise_for_status = MagicMock()
        mock_requests.get.return_value = mock_response

        result = pipeline.fetch_fred_series(
            "DGS3MO", "2024-06-01", "2024-06-02"
        )

        assert result is None


# ==================================================================
# merge_all_series
# ==================================================================


class TestMergeAllSeries:
    def test_single_series(self, pipeline):
        df = pd.DataFrame(
            {
                "date": [date(2024, 6, 1), date(2024, 6, 2)],
                "value": [5.25, 5.30],
            }
        )
        result = pipeline.merge_all_series({"rate_3m": df})

        assert "rate_3m" in result.columns
        assert len(result) == 2

    def test_multiple_series_merged(self, pipeline):
        df_3m = pd.DataFrame(
            {
                "date": [date(2024, 6, 1), date(2024, 6, 2)],
                "value": [5.25, 5.30],
            }
        )
        df_10y = pd.DataFrame(
            {
                "date": [date(2024, 6, 1), date(2024, 6, 2)],
                "value": [4.50, 4.55],
            }
        )
        result = pipeline.merge_all_series(
            {"rate_3m": df_3m, "rate_10y": df_10y}
        )

        assert "rate_3m" in result.columns
        assert "rate_10y" in result.columns
        assert len(result) == 2

    def test_all_none_returns_empty(self, pipeline):
        result = pipeline.merge_all_series(
            {"rate_3m": None, "rate_10y": None}
        )
        assert result.empty

    def test_partial_overlap_outer_join(self, pipeline):
        df_3m = pd.DataFrame(
            {
                "date": [date(2024, 6, 1)],
                "value": [5.25],
            }
        )
        df_10y = pd.DataFrame(
            {
                "date": [date(2024, 6, 2)],
                "value": [4.50],
            }
        )
        result = pipeline.merge_all_series(
            {"rate_3m": df_3m, "rate_10y": df_10y}
        )
        # Outer join means 2 rows
        assert len(result) == 2


# ==================================================================
# store_treasury_rates
# ==================================================================


class TestStoreTreasuryRates:
    @pytest.mark.asyncio
    async def test_empty_dataframe_returns_false(self, pipeline):
        mock_session = AsyncMock()
        result = await pipeline.store_treasury_rates(
            pd.DataFrame(), mock_session
        )
        assert result is False

    @pytest.mark.asyncio
    async def test_valid_data_stored(self, pipeline):
        mock_session = AsyncMock()
        df = pd.DataFrame(
            {
                "date": [date(2024, 6, 1)],
                "rate_1m": [5.20],
                "rate_3m": [5.25],
                "rate_6m": [5.30],
                "rate_1y": [5.00],
                "rate_2y": [4.80],
                "rate_10y": [4.50],
            }
        )

        result = await pipeline.store_treasury_rates(
            df, mock_session
        )

        assert result is True
        mock_session.execute.assert_called_once()
        mock_session.commit.assert_called_once()
