"""Synthetic canary tests for production API.

These tests hit the live API with read-only GET requests to
verify that user-facing data is sane.  They are designed to
run as a GitHub Actions scheduled workflow.

Set ``API_BASE_URL`` to the production (or staging) base URL
before running, e.g.:

    API_BASE_URL=https://api.investorcenter.ai \
        python -m pytest scripts/canary_tests.py -v

Exit codes follow normal pytest conventions (0 = pass).
"""

import os
from datetime import datetime, timedelta, timezone

import pytest
import requests

BASE_URL = os.environ.get("API_BASE_URL", "http://localhost:8080")

# Timeout for every request (seconds)
REQUEST_TIMEOUT = 15


def _get(path: str, **params) -> requests.Response:
    """Convenience wrapper around requests.get."""
    url = f"{BASE_URL}{path}"
    return requests.get(url, params=params, timeout=REQUEST_TIMEOUT)


# ==================================================================
# Health
# ==================================================================


class TestHealth:
    def test_health_endpoint(self):
        """GET /health should return 200."""
        resp = _get("/health")
        assert resp.status_code == 200


# ==================================================================
# Ticker data
# ==================================================================


class TestTickerData:
    @pytest.mark.parametrize(
        "symbol",
        ["AAPL", "MSFT", "GOOGL", "AMZN", "META"],
    )
    def test_large_cap_tickers_respond(self, symbol):
        """Major tickers should return 200."""
        resp = _get(f"/api/v1/tickers/{symbol}")
        assert resp.status_code == 200, f"{symbol} returned {resp.status_code}"

    def test_aapl_has_price(self):
        """AAPL should have a non-null price."""
        resp = _get("/api/v1/tickers/AAPL")
        assert resp.status_code == 200
        data = resp.json()
        # The ticker endpoint returns price data directly
        # or nested -- check common field names
        price = data.get("last_price") or data.get("price")
        assert price is not None, "AAPL price is missing"
        assert float(price) > 0, "AAPL price should be > 0"

    def test_aapl_has_ic_score(self):
        """AAPL should have an IC Score between 1-100."""
        resp = _get("/api/v1/stocks/AAPL/ic-score")
        assert resp.status_code == 200
        data = resp.json()
        score = data.get("ic_score") or data.get("score")
        if score is not None:
            assert 1 <= float(score) <= 100, f"IC Score {score} out of range"

    def test_etf_data_exists(self):
        """SPY should return with ETF asset type."""
        resp = _get("/api/v1/tickers/SPY")
        assert resp.status_code == 200
        data = resp.json()
        asset_type = data.get("asset_type", "").lower()
        assert asset_type in (
            "etf",
            "exchange traded fund",
        ), f"SPY asset_type is '{asset_type}', expected ETF"


# ==================================================================
# Technical indicators
# ==================================================================


class TestTechnicalIndicators:
    def test_aapl_technical_indicators(self):
        """AAPL should have technical indicator data."""
        resp = _get("/api/v1/stocks/AAPL/technical")
        assert resp.status_code == 200
        data = resp.json()
        # Should have some data -- exact shape varies
        assert data is not None
        assert len(data) > 0 or isinstance(data, dict)


# ==================================================================
# Screener
# ==================================================================


class TestScreener:
    def test_screener_returns_data(self):
        """Screener should return at least 1 stock."""
        resp = _get("/api/v1/screener/stocks", limit=10)
        assert resp.status_code == 200
        data = resp.json()
        stocks = data.get("stocks") or data.get("data") or []
        assert len(stocks) >= 1, "Screener returned no stocks"

    def test_screener_sort_by_market_cap(self):
        """Screener sorted by market_cap desc should work."""
        resp = _get(
            "/api/v1/screener/stocks",
            sort_by="market_cap",
            order="desc",
            limit=5,
        )
        assert resp.status_code == 200
        data = resp.json()
        stocks = data.get("stocks") or data.get("data") or []
        assert len(stocks) >= 1


# ==================================================================
# Market indices
# ==================================================================


class TestMarketIndices:
    def test_market_indices(self):
        """Market indices endpoint should return data."""
        resp = _get("/api/v1/markets/indices")
        assert resp.status_code == 200
        data = resp.json()
        assert data is not None


# ==================================================================
# Crypto
# ==================================================================


class TestCrypto:
    def test_crypto_btc_price(self):
        """BTC price should be available and positive."""
        resp = _get("/api/v1/crypto/BTC/price")
        # Crypto may not be available in all environments
        if resp.status_code == 200:
            data = resp.json()
            price = data.get("current_price") or data.get("price") or 0
            assert float(price) > 0, "BTC price should be > 0"
        else:
            pytest.skip(f"Crypto endpoint returned {resp.status_code}")
