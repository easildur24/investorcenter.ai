"""Tests for technical indicators calculator.

Tests the calculate_technical_indicators() method which uses TA-Lib
to compute RSI, MACD, SMA, EMA, Bollinger Bands, and momentum returns.

Since TA-Lib (C library) may not be installed in the test environment,
we mock the talib module and verify the calculator's logic around the
TA-Lib calls: data validation, result packaging, momentum returns, and
edge-case handling.
"""

import sys
from types import ModuleType
from unittest.mock import MagicMock, patch

import numpy as np
import pandas as pd
import pytest

# -------------------------------------------------------------------
# Create a fake talib module *before* the calculator is imported.
# Each function returns a numpy array matching TA-Lib's interface.
# -------------------------------------------------------------------
_mock_talib = ModuleType("talib")


def _fake_rsi(close, timeperiod=14):
    """Simple RSI approximation for testing."""
    n = len(close)
    result = np.full(n, np.nan)
    if n <= timeperiod:
        return result
    for i in range(timeperiod, n):
        window = close[i - timeperiod : i + 1]
        deltas = np.diff(window)
        gains = np.where(deltas > 0, deltas, 0)
        losses = np.where(deltas < 0, -deltas, 0)
        avg_gain = np.mean(gains)
        avg_loss = np.mean(losses)
        if avg_loss == 0:
            result[i] = 100.0
        else:
            rs = avg_gain / avg_loss
            result[i] = 100.0 - (100.0 / (1.0 + rs))
    return result


def _fake_sma(close, timeperiod=20):
    """Simple moving average for testing."""
    n = len(close)
    result = np.full(n, np.nan)
    for i in range(timeperiod - 1, n):
        result[i] = np.mean(close[i - timeperiod + 1 : i + 1])
    return result


def _fake_ema(close, timeperiod=12):
    """Exponential moving average for testing."""
    n = len(close)
    result = np.full(n, np.nan)
    if n < timeperiod:
        return result
    k = 2.0 / (timeperiod + 1)
    result[timeperiod - 1] = np.mean(close[:timeperiod])
    for i in range(timeperiod, n):
        result[i] = close[i] * k + result[i - 1] * (1 - k)
    return result


def _fake_macd(close, fastperiod=12, slowperiod=26, signalperiod=9):
    """Simplified MACD for testing."""
    fast_ema = _fake_ema(close, fastperiod)
    slow_ema = _fake_ema(close, slowperiod)
    macd_line = fast_ema - slow_ema
    # Signal line = EMA of MACD
    n = len(close)
    signal = np.full(n, np.nan)
    # Find first valid MACD value
    valid_start = None
    for i in range(n):
        if not np.isnan(macd_line[i]):
            valid_start = i
            break
    if valid_start is not None and (n - valid_start) >= signalperiod:
        sig_start = valid_start + signalperiod - 1
        signal[sig_start] = np.mean(
            macd_line[valid_start : sig_start + 1]
        )
        k = 2.0 / (signalperiod + 1)
        for i in range(sig_start + 1, n):
            signal[i] = macd_line[i] * k + signal[i - 1] * (1 - k)
    hist = macd_line - signal
    return macd_line, signal, hist


def _fake_bbands(
    close, timeperiod=20, nbdevup=2, nbdevdn=2, matype=0
):
    """Simplified Bollinger Bands for testing."""
    n = len(close)
    upper = np.full(n, np.nan)
    middle = np.full(n, np.nan)
    lower = np.full(n, np.nan)
    for i in range(timeperiod - 1, n):
        window = close[i - timeperiod + 1 : i + 1]
        m = np.mean(window)
        s = np.std(window)
        middle[i] = m
        upper[i] = m + nbdevup * s
        lower[i] = m - nbdevdn * s
    return upper, middle, lower


_mock_talib.RSI = _fake_rsi
_mock_talib.MACD = _fake_macd
_mock_talib.SMA = _fake_sma
_mock_talib.EMA = _fake_ema
_mock_talib.BBANDS = _fake_bbands

# Install mock talib into sys.modules before importing the calculator
sys.modules.setdefault("talib", _mock_talib)

# conftest.py sets LOG_DIR and patches logging.FileHandler before import
from pipelines.technical_indicators_calculator import (
    TechnicalIndicatorsCalculator,
)


@pytest.fixture
def calc():
    """Create a TechnicalIndicatorsCalculator with mocked DB."""
    with patch(
        "pipelines.technical_indicators_calculator.get_database"
    ):
        return TechnicalIndicatorsCalculator()


def _make_price_df(
    close_prices, *, include_volume=True, start_date="2020-01-01"
):
    """Helper to build a DataFrame matching the expected schema."""
    n = len(close_prices)
    dates = pd.date_range(start_date, periods=n, freq="B")
    data = {
        "date": dates[:n],
        "open": close_prices,
        "high": [p * 1.01 for p in close_prices],
        "low": [p * 0.99 for p in close_prices],
        "close": close_prices,
    }
    if include_volume:
        data["volume"] = [1_000_000] * n
    return pd.DataFrame(data)


# =====================================================================
# Empty / Insufficient Data
# =====================================================================


class TestInsufficientData:
    def test_empty_dataframe(self, calc):
        df = pd.DataFrame()
        result = calc.calculate_technical_indicators(df)
        assert result == {}

    def test_too_few_rows(self, calc):
        """Fewer than 20 rows should return empty dict."""
        df = _make_price_df([100.0] * 10)
        result = calc.calculate_technical_indicators(df)
        assert result == {}

    def test_exactly_19_rows(self, calc):
        """19 rows is still less than 20 -- empty dict."""
        df = _make_price_df([100.0] * 19)
        result = calc.calculate_technical_indicators(df)
        assert result == {}

    def test_missing_close_column(self, calc):
        """DataFrame without 'close' column should return empty dict."""
        df = pd.DataFrame(
            {
                "date": pd.date_range("2020-01-01", periods=30),
                "open": [100.0] * 30,
            }
        )
        result = calc.calculate_technical_indicators(df)
        assert result == {}


# =====================================================================
# RSI Calculation
# =====================================================================


class TestRSI:
    def test_rsi_returned_with_sufficient_data(self, calc):
        """RSI should be calculated when >= 20 data points."""
        prices = list(range(100, 160))  # 60 ascending prices
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert "rsi" in result
        assert result["rsi"] is not None

    def test_rsi_trending_up_is_high(self, calc):
        """Steadily rising prices should produce RSI > 50."""
        prices = [100 + i * 0.5 for i in range(60)]
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert result["rsi"] > 50

    def test_rsi_trending_down_is_low(self, calc):
        """Steadily falling prices should produce RSI < 50."""
        prices = [200 - i * 0.5 for i in range(60)]
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert result["rsi"] < 50

    def test_rsi_bounded_0_100(self, calc):
        """RSI must always be between 0 and 100."""
        np.random.seed(42)
        prices = (100 + np.cumsum(np.random.randn(100))).tolist()
        prices = [max(p, 1.0) for p in prices]
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert 0.0 <= result["rsi"] <= 100.0


# =====================================================================
# MACD Calculation
# =====================================================================


class TestMACD:
    def test_macd_fields_present(self, calc):
        """MACD, signal, and histogram should all be returned."""
        prices = [100 + i * 0.3 for i in range(60)]
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert "macd" in result
        assert "macd_signal" in result
        assert "macd_histogram" in result

    def test_macd_histogram_is_difference(self, calc):
        """MACD histogram = MACD - signal line."""
        np.random.seed(123)
        prices = (
            100 + np.cumsum(np.random.randn(100) * 0.5)
        ).tolist()
        prices = [max(p, 1.0) for p in prices]
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        if (
            result.get("macd") is not None
            and result.get("macd_signal") is not None
            and result.get("macd_histogram") is not None
        ):
            expected_hist = result["macd"] - result["macd_signal"]
            assert (
                abs(result["macd_histogram"] - expected_hist) < 0.001
            )


# =====================================================================
# SMA / EMA Calculations
# =====================================================================


class TestMovingAverages:
    def test_sma_50_with_sufficient_data(self, calc):
        """SMA-50 needs at least 50 data points."""
        prices = [100.0] * 60
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert result.get("sma_50") is not None
        # Constant prices -> SMA should equal the price
        assert abs(result["sma_50"] - 100.0) < 0.01

    def test_sma_200_none_with_insufficient_data(self, calc):
        """SMA-200 should be None when < 200 data points."""
        prices = [100.0] * 50
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert result.get("sma_200") is None

    def test_sma_200_with_sufficient_data(self, calc):
        """SMA-200 should be present with 200+ data points."""
        prices = [100.0] * 250
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert result.get("sma_200") is not None
        assert abs(result["sma_200"] - 100.0) < 0.01

    def test_ema_12_with_sufficient_data(self, calc):
        """EMA-12 should be present with 20+ data points."""
        prices = [50.0] * 30
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert result.get("ema_12") is not None
        assert abs(result["ema_12"] - 50.0) < 0.01

    def test_ema_26_with_sufficient_data(self, calc):
        """EMA-26 should be present with 30+ data points."""
        prices = [75.0] * 40
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert result.get("ema_26") is not None
        assert abs(result["ema_26"] - 75.0) < 0.01


# =====================================================================
# Bollinger Bands
# =====================================================================


class TestBollingerBands:
    def test_bb_middle_equals_sma20(self, calc):
        """BB middle band should equal the 20-day SMA."""
        prices = [100.0] * 30
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert result.get("bb_middle") is not None
        # For constant prices, middle band = price
        assert abs(result["bb_middle"] - 100.0) < 0.01

    def test_bb_bands_converge_for_constant_prices(self, calc):
        """Constant prices => std=0 => all bands equal the price."""
        prices = [100.0] * 30
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert result.get("bb_upper") is not None
        assert result.get("bb_lower") is not None
        assert abs(result["bb_upper"] - 100.0) < 0.01
        assert abs(result["bb_lower"] - 100.0) < 0.01

    def test_bb_upper_above_lower(self, calc):
        """Upper band must be >= lower band."""
        np.random.seed(99)
        prices = (
            100 + np.cumsum(np.random.randn(60))
        ).tolist()
        prices = [max(p, 1.0) for p in prices]
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        if (
            result.get("bb_upper") is not None
            and result.get("bb_lower") is not None
        ):
            assert result["bb_upper"] >= result["bb_lower"]


# =====================================================================
# Volume Moving Average
# =====================================================================


class TestVolumeMA:
    def test_volume_ma_present_with_volume(self, calc):
        """Volume MA-20 should be calculated if volume column exists."""
        prices = [100.0] * 30
        df = _make_price_df(prices, include_volume=True)
        result = calc.calculate_technical_indicators(df)
        assert result.get("volume_ma_20") is not None
        assert abs(result["volume_ma_20"] - 1_000_000) < 1.0

    def test_volume_ma_none_without_volume(self, calc):
        """Volume MA-20 should be None when no volume column."""
        prices = [100.0] * 30
        df = _make_price_df(prices, include_volume=False)
        result = calc.calculate_technical_indicators(df)
        assert result.get("volume_ma_20") is None


# =====================================================================
# Momentum Returns
# =====================================================================


class TestMomentumReturns:
    def test_1m_return_with_21_days(self, calc):
        """1-month return should be present with >= 21 data points."""
        prices = [100.0] * 20 + [110.0] * 5
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert "1m_return" in result
        # 1m_return = ((110/100) - 1) * 100 = 10%
        assert abs(result["1m_return"] - 10.0) < 0.01

    def test_no_long_term_returns_with_short_data(self, calc):
        """3m/6m/12m returns should not exist with < 63 days."""
        prices = [100.0] * 30
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert "3m_return" not in result
        assert "6m_return" not in result
        assert "12m_return" not in result

    def test_current_price_in_result(self, calc):
        """current_price should always be the last close price."""
        prices = [100.0] * 25 + [150.0]
        df = _make_price_df(prices)
        result = calc.calculate_technical_indicators(df)
        assert result.get("current_price") == 150.0

    def test_data_sorted_by_date(self, calc):
        """Calculator should sort by date, so reversed input works."""
        prices = list(range(100, 130))
        df = _make_price_df(prices)
        # Reverse the DataFrame to simulate unsorted input
        df = df.iloc[::-1].reset_index(drop=True)
        result = calc.calculate_technical_indicators(df)
        # After sorting, last close should be the highest
        assert result.get("current_price") == 129
