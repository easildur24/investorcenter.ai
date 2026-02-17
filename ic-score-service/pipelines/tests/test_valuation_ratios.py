"""Tests for calculate_ratios method in valuation_ratios_calculator.py.

Phase 1A: No database dependencies. Tests the pure calculate_ratios method
on an instance created with mocked DB and Polygon client.
"""

import os
from datetime import date
from unittest.mock import MagicMock, patch

import pytest

# conftest.py sets LOG_DIR and patches logging.FileHandler before import
with patch(
    "pipelines.valuation_ratios_calculator.get_database"
), patch.dict(
    os.environ, {"POLYGON_API_KEY": "test"}
), patch(
    "pipelines.valuation_ratios_calculator.PolygonClient"
):
    from pipelines.valuation_ratios_calculator import (
        ValuationRatiosCalculator,
    )


@pytest.fixture
def calc():
    """Create a ValuationRatiosCalculator with mocked DB and Polygon."""
    with patch(
        "pipelines.valuation_ratios_calculator.get_database"
    ), patch.dict(
        os.environ, {"POLYGON_API_KEY": "test"}
    ), patch(
        "pipelines.valuation_ratios_calculator.PolygonClient"
    ):
        return ValuationRatiosCalculator()


def _make_price_data(close=150.0):
    return {"close": close, "date": date(2024, 1, 15)}


def _make_ttm_data(
    eps_diluted=6.0,
    net_income=6_000_000,
    shares_outstanding=1_000_000,
    shareholders_equity=1_000_000_000,
    revenue=10_000_000_000,
    ttm_id=1,
):
    return {
        "id": ttm_id,
        "calculation_date": date(2024, 1, 15),
        "ttm_period_start": date(2023, 1, 1),
        "ttm_period_end": date(2023, 12, 31),
        "revenue": revenue,
        "net_income": net_income,
        "eps_diluted": eps_diluted,
        "shares_outstanding": shares_outstanding,
        "shareholders_equity": shareholders_equity,
    }


# =====================================================================
# calculate_ratios - Normal case
# =====================================================================


class TestCalculateRatiosNormal:
    def test_normal_case(self, calc):
        price_data = _make_price_data(close=150.0)
        ttm_data = _make_ttm_data(
            eps_diluted=6.0,
            shares_outstanding=1_000_000,
            shareholders_equity=1_000_000_000,
            revenue=10_000_000_000,
        )

        result = calc.calculate_ratios("AAPL", price_data, ttm_data)

        assert result is not None
        assert result["ticker"] == "AAPL"
        assert result["stock_price"] == 150.0
        # P/E = 150 / 6 = 25
        assert result["ttm_pe_ratio"] == 25.0
        # Market cap = 150 * 1_000_000 = 150_000_000
        assert result["ttm_market_cap"] == 150_000_000
        # Book value per share = 1_000_000_000 / 1_000_000 = 1000
        # P/B = 150 / 1000 = 0.15
        assert result["ttm_pb_ratio"] == 0.15
        # P/S = market_cap / revenue = 150_000_000 / 10_000_000_000 = 0.015
        assert result["ttm_ps_ratio"] == 0.015


# =====================================================================
# calculate_ratios - EPS fallback from net_income
# =====================================================================


class TestCalculateRatiosEpsFallback:
    def test_zero_eps_with_net_income_fallback(self, calc):
        price_data = _make_price_data(close=150.0)
        ttm_data = _make_ttm_data(
            eps_diluted=None,
            net_income=6_000_000,
            shares_outstanding=1_000_000,
        )

        result = calc.calculate_ratios("AAPL", price_data, ttm_data)

        assert result is not None
        # Derived EPS = 6_000_000 / 1_000_000 = 6.0
        # P/E = 150 / 6 = 25
        assert result["ttm_pe_ratio"] == 25.0

    def test_zero_eps_value_with_net_income_fallback(self, calc):
        price_data = _make_price_data(close=150.0)
        ttm_data = _make_ttm_data(
            eps_diluted=0,
            net_income=6_000_000,
            shares_outstanding=1_000_000,
        )

        result = calc.calculate_ratios("AAPL", price_data, ttm_data)

        assert result is not None
        # eps==0, net_income and shares present -> fallback
        assert result["ttm_pe_ratio"] == 25.0


# =====================================================================
# calculate_ratios - Negative EPS
# =====================================================================


class TestCalculateRatiosNegativeEps:
    def test_negative_eps(self, calc):
        price_data = _make_price_data(close=150.0)
        ttm_data = _make_ttm_data(eps_diluted=-5.0)

        result = calc.calculate_ratios("AAPL", price_data, ttm_data)

        assert result is not None
        assert result["ttm_pe_ratio"] is None  # Negative EPS -> no P/E


# =====================================================================
# calculate_ratios - Zero shares with market_cap fallback
# =====================================================================


class TestCalculateRatiosSharesFallback:
    def test_no_shares_with_market_cap_fallback(self, calc):
        price_data = _make_price_data(close=150.0)
        ttm_data = _make_ttm_data(
            shares_outstanding=None,
            eps_diluted=6.0,
            revenue=10_000_000_000,
            shareholders_equity=1_000_000_000,
        )

        result = calc.calculate_ratios(
            "AAPL",
            price_data,
            ttm_data,
            tickers_market_cap=150_000_000.0,
        )

        assert result is not None
        assert result["ttm_market_cap"] == 150_000_000
        # P/E still works because eps_diluted is provided directly
        assert result["ttm_pe_ratio"] == 25.0
        # Derived shares = 150_000_000 / 150 = 1_000_000
        # Book value per share = 1_000_000_000 / 1_000_000 = 1000
        # P/B = 150 / 1000 = 0.15
        assert result["ttm_pb_ratio"] == 0.15
        # P/S = 150_000_000 / 10_000_000_000 = 0.015
        assert result["ttm_ps_ratio"] == 0.015


# =====================================================================
# calculate_ratios - No shares, no market_cap
# =====================================================================


class TestCalculateRatiosNoSharesNoMarketCap:
    def test_no_shares_no_market_cap(self, calc):
        price_data = _make_price_data(close=150.0)
        ttm_data = _make_ttm_data(
            shares_outstanding=None,
            eps_diluted=6.0,
            revenue=10_000_000_000,
            shareholders_equity=1_000_000_000,
        )

        result = calc.calculate_ratios("AAPL", price_data, ttm_data)

        assert result is not None
        # P/E can still work because eps_diluted is directly provided
        assert result["ttm_pe_ratio"] == 25.0
        # P/B requires shares -> None
        assert result["ttm_pb_ratio"] is None
        # P/S requires market_cap -> None
        assert result["ttm_ps_ratio"] is None
        # Market cap is None
        assert result["ttm_market_cap"] is None


# =====================================================================
# calculate_ratios - Zero revenue
# =====================================================================


class TestCalculateRatiosZeroRevenue:
    def test_zero_revenue(self, calc):
        price_data = _make_price_data(close=150.0)
        ttm_data = _make_ttm_data(revenue=0)

        result = calc.calculate_ratios("AAPL", price_data, ttm_data)

        assert result is not None
        assert result["ttm_ps_ratio"] is None  # Zero revenue -> no P/S


# =====================================================================
# calculate_ratios - Zero equity
# =====================================================================


class TestCalculateRatiosZeroEquity:
    def test_zero_equity(self, calc):
        price_data = _make_price_data(close=150.0)
        ttm_data = _make_ttm_data(shareholders_equity=0)

        result = calc.calculate_ratios("AAPL", price_data, ttm_data)

        assert result is not None
        # Zero equity -> book value = 0 -> no P/B
        assert result["ttm_pb_ratio"] is None


# =====================================================================
# calculate_ratios - None revenue
# =====================================================================


class TestCalculateRatiosNoneRevenue:
    def test_none_revenue(self, calc):
        price_data = _make_price_data(close=150.0)
        ttm_data = _make_ttm_data(revenue=None)

        result = calc.calculate_ratios("AAPL", price_data, ttm_data)

        assert result is not None
        assert result["ttm_ps_ratio"] is None
