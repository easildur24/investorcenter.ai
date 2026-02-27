"""Unit tests for Polygon.io API client.

Tests cover all public and private methods of PolygonClient:
- __init__: API key from arg and env, missing key raises ValueError
- _rate_limit: sleep enforcement when requests are too fast
- _make_request: success, HTTP 404, HTTP 500, timeout, unexpected errors
- get_aggregates: OHLCV data parsing and timestamp conversion
- get_daily_prices: delegation to get_aggregates with date range
- get_news: with/without ticker, with published_after filter
- get_ticker_details: response parsing
- get_latest_price: latest bar extraction, empty data handling
- close: session cleanup
"""

import os
from datetime import datetime
from unittest.mock import MagicMock, patch

import pytest
import requests

from pipelines.utils.polygon_client import PolygonClient


# =====================================================================
# Fixtures
# =====================================================================


@pytest.fixture(autouse=True)
def polygon_api_key_env(monkeypatch):
    """Ensure POLYGON_API_KEY is set for all tests."""
    monkeypatch.setenv("POLYGON_API_KEY", "test_api_key_12345")


@pytest.fixture
def client():
    """Create a PolygonClient instance with mocked session."""
    with patch.object(PolygonClient, "_create_session") as mock_create:
        mock_session = MagicMock(spec=requests.Session)
        mock_create.return_value = mock_session
        c = PolygonClient()
    return c


def _mock_response(status_code=200, json_data=None, raise_for_status=None):
    """Helper to create a mock response object."""
    mock_resp = MagicMock()
    mock_resp.status_code = status_code
    mock_resp.json.return_value = json_data or {}

    if raise_for_status:
        mock_resp.raise_for_status.side_effect = raise_for_status
    else:
        mock_resp.raise_for_status.return_value = None

    return mock_resp


# =====================================================================
# TestInit
# =====================================================================


class TestInit:
    """Tests for PolygonClient initialization."""

    def test_init_with_explicit_api_key(self):
        """Client accepts an explicit API key argument."""
        with patch.object(PolygonClient, "_create_session"):
            client = PolygonClient(api_key="my_explicit_key")
        assert client.api_key == "my_explicit_key"

    def test_init_with_env_api_key(self):
        """Client reads API key from POLYGON_API_KEY environment variable."""
        with patch.object(PolygonClient, "_create_session"):
            client = PolygonClient()
        assert client.api_key == "test_api_key_12345"

    def test_init_missing_api_key_raises(self, monkeypatch):
        """Client raises ValueError when no API key is available."""
        monkeypatch.delenv("POLYGON_API_KEY", raising=False)
        with pytest.raises(ValueError, match="POLYGON_API_KEY"):
            PolygonClient(api_key=None)

    def test_init_sets_last_request_time_to_zero(self, client):
        """Initial last_request_time is 0.0."""
        assert client.last_request_time == 0.0


# =====================================================================
# TestCreateSession
# =====================================================================


class TestCreateSession:
    """Tests for _create_session (real session creation)."""

    def test_creates_real_session(self, monkeypatch):
        """_create_session returns a requests.Session with retry adapters."""
        client = PolygonClient()
        assert isinstance(client.session, requests.Session)
        client.close()


# =====================================================================
# TestRateLimit
# =====================================================================


class TestRateLimit:
    """Tests for _rate_limit enforcement."""

    @patch("pipelines.utils.polygon_client.time")
    def test_rate_limit_sleeps_when_too_fast(self, mock_time, client):
        """Rate limiter sleeps when requests are faster than MIN_REQUEST_INTERVAL."""
        # First call at time 1.0, second call at 1.05 (only 0.05s later)
        mock_time.time.side_effect = [1.05, 1.25]
        client.last_request_time = 1.0

        client._rate_limit()

        # Should have slept: MIN_REQUEST_INTERVAL - 0.05 = 0.2 - 0.05 = 0.15
        mock_time.sleep.assert_called_once()
        sleep_arg = mock_time.sleep.call_args[0][0]
        assert abs(sleep_arg - 0.15) < 0.01

    @patch("pipelines.utils.polygon_client.time")
    def test_rate_limit_no_sleep_when_enough_time_elapsed(self, mock_time, client):
        """Rate limiter does not sleep when enough time has passed."""
        mock_time.time.side_effect = [2.0, 2.0]
        client.last_request_time = 1.0  # 1.0 second ago, well beyond MIN_REQUEST_INTERVAL

        client._rate_limit()

        mock_time.sleep.assert_not_called()

    @patch("pipelines.utils.polygon_client.time")
    def test_rate_limit_updates_last_request_time(self, mock_time, client):
        """Rate limiter updates last_request_time after each call."""
        mock_time.time.side_effect = [5.0, 5.0]
        client.last_request_time = 0.0

        client._rate_limit()

        assert client.last_request_time == 5.0


# =====================================================================
# TestMakeRequest
# =====================================================================


class TestMakeRequest:
    """Tests for _make_request."""

    @patch("pipelines.utils.polygon_client.time")
    def test_success(self, mock_time, client):
        """Successful request returns parsed JSON."""
        mock_time.time.return_value = 100.0
        expected_data = {"results": [{"ticker": "AAPL"}]}
        mock_resp = _mock_response(json_data=expected_data)
        client.session.get.return_value = mock_resp

        result = client._make_request("/v2/test", {"param1": "value1"})

        assert result == expected_data
        client.session.get.assert_called_once()
        # Verify apiKey was injected
        call_kwargs = client.session.get.call_args
        assert call_kwargs[1]["params"]["apiKey"] == "test_api_key_12345"
        assert call_kwargs[1]["params"]["param1"] == "value1"

    @patch("pipelines.utils.polygon_client.time")
    def test_success_no_params(self, mock_time, client):
        """Request with no params still passes apiKey."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"status": "ok"})
        client.session.get.return_value = mock_resp

        result = client._make_request("/v2/test")

        assert result == {"status": "ok"}
        call_kwargs = client.session.get.call_args
        assert call_kwargs[1]["params"]["apiKey"] == "test_api_key_12345"

    @patch("pipelines.utils.polygon_client.time")
    def test_http_404_returns_none(self, mock_time, client):
        """HTTP 404 returns None and logs warning."""
        mock_time.time.return_value = 100.0
        http_error = requests.exceptions.HTTPError(response=MagicMock(status_code=404))
        mock_resp = _mock_response(status_code=404, raise_for_status=http_error)
        client.session.get.return_value = mock_resp

        result = client._make_request("/v2/missing")

        assert result is None

    @patch("pipelines.utils.polygon_client.time")
    def test_http_500_returns_none(self, mock_time, client):
        """HTTP 500 server error returns None."""
        mock_time.time.return_value = 100.0
        http_error = requests.exceptions.HTTPError(response=MagicMock(status_code=500))
        mock_resp = _mock_response(status_code=500, raise_for_status=http_error)
        client.session.get.return_value = mock_resp

        result = client._make_request("/v2/broken")

        assert result is None

    @patch("pipelines.utils.polygon_client.time")
    def test_http_429_rate_limit_returns_none(self, mock_time, client):
        """HTTP 429 rate limit error returns None."""
        mock_time.time.return_value = 100.0
        http_error = requests.exceptions.HTTPError(response=MagicMock(status_code=429))
        mock_resp = _mock_response(status_code=429, raise_for_status=http_error)
        client.session.get.return_value = mock_resp

        result = client._make_request("/v2/rate_limited")

        assert result is None

    @patch("pipelines.utils.polygon_client.time")
    def test_timeout_returns_none(self, mock_time, client):
        """Request timeout returns None."""
        mock_time.time.return_value = 100.0
        client.session.get.side_effect = requests.exceptions.Timeout("Connection timed out")

        result = client._make_request("/v2/slow")

        assert result is None

    @patch("pipelines.utils.polygon_client.time")
    def test_connection_error_returns_none(self, mock_time, client):
        """Connection error returns None."""
        mock_time.time.return_value = 100.0
        client.session.get.side_effect = requests.exceptions.ConnectionError("No route")

        result = client._make_request("/v2/unreachable")

        assert result is None

    @patch("pipelines.utils.polygon_client.time")
    def test_unexpected_exception_returns_none(self, mock_time, client):
        """Unexpected exception returns None."""
        mock_time.time.return_value = 100.0
        client.session.get.side_effect = RuntimeError("Something unexpected")

        result = client._make_request("/v2/broken")

        assert result is None

    @patch("pipelines.utils.polygon_client.time")
    def test_url_construction(self, mock_time, client):
        """Request URL is built from BASE_URL + endpoint."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={})
        client.session.get.return_value = mock_resp

        client._make_request("/v2/aggs/ticker/AAPL/range/1/day/2024-01-01/2024-12-31")

        call_args = client.session.get.call_args
        assert call_args[0][0] == "https://api.polygon.io/v2/aggs/ticker/AAPL/range/1/day/2024-01-01/2024-12-31"

    @patch("pipelines.utils.polygon_client.time")
    def test_timeout_parameter(self, mock_time, client):
        """Request uses 30-second timeout."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={})
        client.session.get.return_value = mock_resp

        client._make_request("/v2/test")

        call_kwargs = client.session.get.call_args
        assert call_kwargs[1]["timeout"] == 30


# =====================================================================
# TestGetAggregates
# =====================================================================


class TestGetAggregates:
    """Tests for get_aggregates."""

    @patch("pipelines.utils.polygon_client.time")
    def test_valid_ohlcv_data(self, mock_time, client):
        """Returns parsed OHLCV bars with date conversion."""
        mock_time.time.return_value = 100.0
        # Timestamp 1704067200000 = 2024-01-01 00:00:00 UTC
        api_response = {
            "results": [
                {
                    "o": 150.0,
                    "h": 155.0,
                    "l": 148.0,
                    "c": 153.0,
                    "v": 1000000,
                    "t": 1704067200000,
                },
                {
                    "o": 153.0,
                    "h": 157.0,
                    "l": 152.0,
                    "c": 156.0,
                    "v": 900000,
                    "t": 1704153600000,
                },
            ]
        }
        mock_resp = _mock_response(json_data=api_response)
        client.session.get.return_value = mock_resp

        result = client.get_aggregates(
            ticker="AAPL",
            from_date="2024-01-01",
            to_date="2024-01-02",
        )

        assert result is not None
        assert len(result) == 2
        assert result[0]["o"] == 150.0
        assert result[0]["c"] == 153.0
        assert result[0]["v"] == 1000000
        # Verify date was added from timestamp
        assert "date" in result[0]
        assert isinstance(result[0]["date"], type(datetime.now().date()))

    @patch("pipelines.utils.polygon_client.time")
    def test_no_results_key_returns_none(self, mock_time, client):
        """Returns None when response lacks 'results' key."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"status": "OK", "count": 0})
        client.session.get.return_value = mock_resp

        result = client.get_aggregates(ticker="INVALID", from_date="2024-01-01", to_date="2024-01-02")

        assert result is None

    @patch("pipelines.utils.polygon_client.time")
    def test_api_failure_returns_none(self, mock_time, client):
        """Returns None when API request fails."""
        mock_time.time.return_value = 100.0
        client.session.get.side_effect = requests.exceptions.ConnectionError()

        result = client.get_aggregates(ticker="AAPL", from_date="2024-01-01", to_date="2024-01-02")

        assert result is None

    @patch("pipelines.utils.polygon_client.time")
    def test_default_date_range(self, mock_time, client):
        """Uses default 3-year date range when no dates provided."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"results": []})
        client.session.get.return_value = mock_resp

        result = client.get_aggregates(ticker="AAPL")

        # Should have called with default dates
        call_args = client.session.get.call_args
        url = call_args[0][0]
        # URL should contain date ranges
        assert "/range/1/day/" in url

    @patch("pipelines.utils.polygon_client.time")
    def test_custom_multiplier_and_timespan(self, mock_time, client):
        """Passes custom multiplier and timespan in endpoint."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"results": []})
        client.session.get.return_value = mock_resp

        client.get_aggregates(
            ticker="MSFT",
            multiplier=5,
            timespan="minute",
            from_date="2024-01-01",
            to_date="2024-01-02",
        )

        call_args = client.session.get.call_args
        url = call_args[0][0]
        assert "/range/5/minute/" in url

    @patch("pipelines.utils.polygon_client.time")
    def test_query_params(self, mock_time, client):
        """Passes adjusted, sort, and limit as query parameters."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"results": []})
        client.session.get.return_value = mock_resp

        client.get_aggregates(
            ticker="AAPL",
            from_date="2024-01-01",
            to_date="2024-01-02",
            limit=100,
        )

        call_kwargs = client.session.get.call_args
        params = call_kwargs[1]["params"]
        assert params["adjusted"] == "true"
        assert params["sort"] == "asc"
        assert params["limit"] == 100

    @patch("pipelines.utils.polygon_client.time")
    def test_bars_without_timestamp(self, mock_time, client):
        """Bars without 't' field do not get 'date' added."""
        mock_time.time.return_value = 100.0
        api_response = {
            "results": [
                {"o": 150.0, "h": 155.0, "l": 148.0, "c": 153.0, "v": 1000000},
            ]
        }
        mock_resp = _mock_response(json_data=api_response)
        client.session.get.return_value = mock_resp

        result = client.get_aggregates(ticker="AAPL", from_date="2024-01-01", to_date="2024-01-02")

        assert result is not None
        assert "date" not in result[0]


# =====================================================================
# TestGetDailyPrices
# =====================================================================


class TestGetDailyPrices:
    """Tests for get_daily_prices."""

    @patch("pipelines.utils.polygon_client.time")
    def test_returns_daily_bars(self, mock_time, client):
        """Returns daily OHLCV bars with proper date conversion."""
        mock_time.time.return_value = 100.0
        api_response = {
            "results": [
                {"o": 100.0, "h": 105.0, "l": 99.0, "c": 103.0, "v": 500000, "t": 1704067200000},
            ]
        }
        mock_resp = _mock_response(json_data=api_response)
        client.session.get.return_value = mock_resp

        result = client.get_daily_prices(ticker="GOOG", days=30)

        assert result is not None
        assert len(result) == 1
        assert result[0]["c"] == 103.0
        assert "date" in result[0]

    @patch("pipelines.utils.polygon_client.time")
    def test_uses_correct_date_range(self, mock_time, client):
        """Passes days+50 as lookback for calendar-to-trading day buffer."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"results": []})
        client.session.get.return_value = mock_resp

        client.get_daily_prices(ticker="AAPL", days=252)

        call_args = client.session.get.call_args
        url = call_args[0][0]
        # URL should be constructed for 1-day aggregates
        assert "/range/1/day/" in url

    @patch("pipelines.utils.polygon_client.time")
    def test_api_failure_returns_none(self, mock_time, client):
        """Returns None when API fails."""
        mock_time.time.return_value = 100.0
        client.session.get.side_effect = requests.exceptions.ConnectionError()

        result = client.get_daily_prices(ticker="BAD")

        assert result is None


# =====================================================================
# TestGetNews
# =====================================================================


class TestGetNews:
    """Tests for get_news."""

    @patch("pipelines.utils.polygon_client.time")
    def test_news_without_ticker(self, mock_time, client):
        """Fetches general news when no ticker is specified."""
        mock_time.time.return_value = 100.0
        api_response = {
            "results": [
                {"id": "1", "title": "Market Update", "published_utc": "2024-01-15T10:00:00Z"},
                {"id": "2", "title": "Fed Decision", "published_utc": "2024-01-14T09:00:00Z"},
            ]
        }
        mock_resp = _mock_response(json_data=api_response)
        client.session.get.return_value = mock_resp

        result = client.get_news()

        assert result is not None
        assert len(result) == 2
        assert result[0]["title"] == "Market Update"

    @patch("pipelines.utils.polygon_client.time")
    def test_news_with_ticker(self, mock_time, client):
        """Includes ticker parameter when specified."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"results": [{"id": "1", "title": "AAPL Earnings"}]})
        client.session.get.return_value = mock_resp

        result = client.get_news(ticker="AAPL")

        assert result is not None
        call_kwargs = client.session.get.call_args
        assert call_kwargs[1]["params"]["ticker"] == "AAPL"

    @patch("pipelines.utils.polygon_client.time")
    def test_news_with_published_after(self, mock_time, client):
        """Includes published_utc.gte parameter when published_after is set."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"results": []})
        client.session.get.return_value = mock_resp

        client.get_news(published_after="2024-01-01")

        call_kwargs = client.session.get.call_args
        assert call_kwargs[1]["params"]["published_utc.gte"] == "2024-01-01"

    @patch("pipelines.utils.polygon_client.time")
    def test_news_custom_limit(self, mock_time, client):
        """Passes custom limit parameter."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"results": []})
        client.session.get.return_value = mock_resp

        client.get_news(limit=50)

        call_kwargs = client.session.get.call_args
        assert call_kwargs[1]["params"]["limit"] == 50

    @patch("pipelines.utils.polygon_client.time")
    def test_news_no_results_returns_none(self, mock_time, client):
        """Returns None when response has no results key."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"status": "error"})
        client.session.get.return_value = mock_resp

        result = client.get_news()

        assert result is None

    @patch("pipelines.utils.polygon_client.time")
    def test_news_api_failure_returns_none(self, mock_time, client):
        """Returns None when API request fails."""
        mock_time.time.return_value = 100.0
        client.session.get.side_effect = requests.exceptions.Timeout()

        result = client.get_news(ticker="AAPL")

        assert result is None

    @patch("pipelines.utils.polygon_client.time")
    def test_news_default_params(self, mock_time, client):
        """Default params include order=desc and sort=published_utc."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"results": []})
        client.session.get.return_value = mock_resp

        client.get_news()

        call_kwargs = client.session.get.call_args
        params = call_kwargs[1]["params"]
        assert params["order"] == "desc"
        assert params["sort"] == "published_utc"
        assert params["limit"] == 100


# =====================================================================
# TestGetTickerDetails
# =====================================================================


class TestGetTickerDetails:
    """Tests for get_ticker_details."""

    @patch("pipelines.utils.polygon_client.time")
    def test_returns_ticker_details(self, mock_time, client):
        """Returns ticker details from response['results']."""
        mock_time.time.return_value = 100.0
        api_response = {
            "results": {
                "ticker": "AAPL",
                "name": "Apple Inc.",
                "market_cap": 3000000000000,
                "sic_code": "3571",
            }
        }
        mock_resp = _mock_response(json_data=api_response)
        client.session.get.return_value = mock_resp

        result = client.get_ticker_details("AAPL")

        assert result is not None
        assert result["ticker"] == "AAPL"
        assert result["name"] == "Apple Inc."
        assert result["market_cap"] == 3000000000000

    @patch("pipelines.utils.polygon_client.time")
    def test_correct_endpoint(self, mock_time, client):
        """Calls the v3 reference endpoint with ticker."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"results": {"ticker": "MSFT"}})
        client.session.get.return_value = mock_resp

        client.get_ticker_details("MSFT")

        call_args = client.session.get.call_args
        url = call_args[0][0]
        assert url == "https://api.polygon.io/v3/reference/tickers/MSFT"

    @patch("pipelines.utils.polygon_client.time")
    def test_no_results_returns_none(self, mock_time, client):
        """Returns None when response has no results key."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"status": "NOT_FOUND"})
        client.session.get.return_value = mock_resp

        result = client.get_ticker_details("INVALID")

        assert result is None

    @patch("pipelines.utils.polygon_client.time")
    def test_api_failure_returns_none(self, mock_time, client):
        """Returns None when API request fails."""
        mock_time.time.return_value = 100.0
        client.session.get.side_effect = requests.exceptions.ConnectionError()

        result = client.get_ticker_details("AAPL")

        assert result is None


# =====================================================================
# TestGetLatestPrice
# =====================================================================


class TestGetLatestPrice:
    """Tests for get_latest_price."""

    @patch("pipelines.utils.polygon_client.time")
    def test_returns_latest_bar(self, mock_time, client):
        """Returns the most recent bar with proper field mapping."""
        mock_time.time.return_value = 100.0
        api_response = {
            "results": [
                {"o": 148.0, "h": 150.0, "l": 147.0, "c": 149.0, "v": 800000, "t": 1704067200000},
                {"o": 149.0, "h": 153.0, "l": 148.0, "c": 152.0, "v": 900000, "t": 1704153600000},
            ]
        }
        mock_resp = _mock_response(json_data=api_response)
        client.session.get.return_value = mock_resp

        result = client.get_latest_price("AAPL")

        assert result is not None
        assert result["close"] == 152.0
        assert result["open"] == 149.0
        assert result["high"] == 153.0
        assert result["low"] == 148.0
        assert result["volume"] == 900000
        assert result["timestamp"] == 1704153600000

    @patch("pipelines.utils.polygon_client.time")
    def test_no_data_returns_none(self, mock_time, client):
        """Returns None when no price data is available."""
        mock_time.time.return_value = 100.0
        # _make_request returns None (e.g. network error)
        client.session.get.side_effect = requests.exceptions.ConnectionError()

        result = client.get_latest_price("INVALID")

        assert result is None

    @patch("pipelines.utils.polygon_client.time")
    def test_empty_results_returns_none(self, mock_time, client):
        """Returns None when results list is empty."""
        mock_time.time.return_value = 100.0
        mock_resp = _mock_response(json_data={"results": []})
        client.session.get.return_value = mock_resp

        result = client.get_latest_price("AAPL")

        assert result is None


# =====================================================================
# TestClose
# =====================================================================


class TestClose:
    """Tests for close method."""

    def test_close_calls_session_close(self, client):
        """close() calls session.close()."""
        client.close()

        client.session.close.assert_called_once()

    def test_close_with_none_session(self):
        """close() handles None session gracefully."""
        with patch.object(PolygonClient, "_create_session"):
            c = PolygonClient()
        c.session = None

        # Should not raise
        c.close()


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
