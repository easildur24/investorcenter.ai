"""Tests for the analyst ratings ingestion pipeline.

Validates Benzinga API fetching, rating parsing, RATING_MAP values,
and database storage logic with mocked requests and get_database.
"""

from datetime import datetime
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# Patch get_database before importing the pipeline module.
with patch("pipelines.analyst_ratings_ingestion.get_database"):
    from pipelines.analyst_ratings_ingestion import (
        AnalystRatingsIngestion,
    )


@pytest.fixture
def pipeline():
    """Create an AnalystRatingsIngestion with mocked DB."""
    with patch(
        "pipelines.analyst_ratings_ingestion.get_database"
    ) as mock_db:
        db_instance = MagicMock()
        mock_db.return_value = db_instance
        p = AnalystRatingsIngestion(api_key="test-key")
        yield p


def _make_raw_rating(
    ticker="AAPL",
    date_str="2024-06-15",
    rating_current="Buy",
    rating_prior=None,
    pt_current=200.0,
    pt_prior=None,
    analyst="John Smith",
    analyst_firm="Goldman Sachs",
    action="Initiated",
):
    """Build a raw Benzinga-style rating dictionary."""
    return {
        "ticker": ticker,
        "date": date_str,
        "rating_current": rating_current,
        "rating_prior": rating_prior,
        "pt_current": pt_current,
        "pt_prior": pt_prior,
        "analyst": analyst,
        "analyst_firm": analyst_firm,
        "action": action,
        "notes": None,
    }


# ==================================================================
# RATING_MAP
# ==================================================================


class TestRatingMap:
    def test_strong_buy_is_5(self):
        assert AnalystRatingsIngestion.RATING_MAP["Strong Buy"] == 5.0

    def test_buy_is_4(self):
        assert AnalystRatingsIngestion.RATING_MAP["Buy"] == 4.0

    def test_hold_is_3(self):
        assert AnalystRatingsIngestion.RATING_MAP["Hold"] == 3.0

    def test_sell_is_2(self):
        assert AnalystRatingsIngestion.RATING_MAP["Sell"] == 2.0

    def test_strong_sell_is_1(self):
        assert (
            AnalystRatingsIngestion.RATING_MAP["Strong Sell"] == 1.0
        )

    def test_all_keys_present(self):
        expected_keys = {
            "Strong Buy",
            "Buy",
            "Outperform",
            "Hold",
            "Neutral",
            "Market Perform",
            "Underperform",
            "Sell",
            "Strong Sell",
        }
        assert set(AnalystRatingsIngestion.RATING_MAP.keys()) == expected_keys


# ==================================================================
# parse_rating
# ==================================================================


class TestParseRating:
    def test_valid_rating_parsed(self, pipeline):
        raw = _make_raw_rating()
        result = pipeline.parse_rating(raw)

        assert result is not None
        assert result["ticker"] == "AAPL"
        assert result["rating"] == "Buy"
        assert result["rating_numeric"] == 4.0
        assert result["price_target"] == 200.0
        assert result["analyst_name"] == "John Smith"
        assert result["analyst_firm"] == "Goldman Sachs"
        assert result["source"] == "Benzinga"

    def test_missing_ticker_returns_none(self, pipeline):
        raw = _make_raw_rating()
        raw["ticker"] = None
        result = pipeline.parse_rating(raw)
        assert result is None

    def test_ticker_uppercased(self, pipeline):
        raw = _make_raw_rating(ticker="aapl")
        result = pipeline.parse_rating(raw)
        assert result["ticker"] == "AAPL"

    def test_upgrade_action_detected(self, pipeline):
        raw = _make_raw_rating(
            rating_current="Buy", rating_prior="Hold"
        )
        result = pipeline.parse_rating(raw)
        assert result["action"] == "Upgraded"

    def test_downgrade_action_detected(self, pipeline):
        raw = _make_raw_rating(
            rating_current="Hold", rating_prior="Buy"
        )
        result = pipeline.parse_rating(raw)
        assert result["action"] == "Downgraded"

    def test_reiterated_action_same_numeric(self, pipeline):
        # Outperform and Buy both map to 4.0
        raw = _make_raw_rating(
            rating_current="Outperform", rating_prior="Buy"
        )
        result = pipeline.parse_rating(raw)
        assert result["action"] == "Reiterated"

    def test_missing_date_uses_today(self, pipeline):
        raw = _make_raw_rating()
        raw["date"] = None
        result = pipeline.parse_rating(raw)
        assert result["rating_date"] == datetime.now().date()

    def test_unknown_rating_numeric_is_none(self, pipeline):
        raw = _make_raw_rating(rating_current="Speculative Buy")
        result = pipeline.parse_rating(raw)
        assert result["rating_numeric"] is None


# ==================================================================
# fetch_ratings_benzinga
# ==================================================================


class TestFetchRatingsBenzinga:
    @patch("pipelines.analyst_ratings_ingestion.requests")
    def test_success(self, mock_requests, pipeline):
        mock_response = MagicMock()
        mock_response.json.return_value = {
            "ratings": [_make_raw_rating()]
        }
        mock_response.raise_for_status = MagicMock()
        mock_requests.get.return_value = mock_response

        result = pipeline.fetch_ratings_benzinga(
            ticker="AAPL", limit=10
        )

        assert len(result) == 1
        mock_requests.get.assert_called_once()

    @patch("pipelines.analyst_ratings_ingestion.requests")
    def test_api_error_returns_empty(
        self, mock_requests, pipeline
    ):
        mock_requests.get.side_effect = (
            Exception("Connection error")
        )
        # Need to set the exception as a requests exception type
        import requests as req

        mock_requests.exceptions = req.exceptions

        result = pipeline.fetch_ratings_benzinga(ticker="AAPL")
        assert result == []

    def test_no_api_key_returns_empty(self):
        with patch(
            "pipelines.analyst_ratings_ingestion.get_database"
        ):
            p = AnalystRatingsIngestion(api_key=None)
            # Force api_key to None
            p.api_key = None
            result = p.fetch_ratings_benzinga()
            assert result == []


# ==================================================================
# store_ratings
# ==================================================================


class TestStoreRatings:
    @pytest.mark.asyncio
    async def test_empty_list_returns_false(self, pipeline):
        result = await pipeline.store_ratings([])
        assert result is False

    @pytest.mark.asyncio
    async def test_valid_ratings_stored(self, pipeline):
        mock_session = AsyncMock()
        pipeline.db.session.return_value.__aenter__ = (
            AsyncMock(return_value=mock_session)
        )
        pipeline.db.session.return_value.__aexit__ = (
            AsyncMock(return_value=False)
        )

        ratings = [
            {
                "ticker": "AAPL",
                "rating_date": datetime(2024, 6, 15).date(),
                "analyst_name": "Test",
                "analyst_firm": "TestFirm",
                "rating": "Buy",
                "rating_numeric": 4.0,
                "price_target": 200.0,
                "prior_rating": None,
                "prior_price_target": None,
                "action": "Initiated",
                "notes": None,
                "source": "Benzinga",
            }
        ]

        result = await pipeline.store_ratings(ratings)
        assert result is True
        mock_session.execute.assert_called_once()
        mock_session.commit.assert_called_once()
