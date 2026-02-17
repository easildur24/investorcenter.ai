"""Tests for pure methods in risk_metrics_calculator.py.

Phase 1A: No database dependencies. Tests only pure calculation methods
on an instance created with a mocked database.
"""

from unittest.mock import patch

import numpy as np
import pandas as pd
import pytest

# conftest.py sets LOG_DIR and patches logging.FileHandler before import
with patch("pipelines.risk_metrics_calculator.get_database"):
    from pipelines.risk_metrics_calculator import RiskMetricsCalculator


@pytest.fixture
def calc():
    """Create a RiskMetricsCalculator with mocked DB."""
    with patch("pipelines.risk_metrics_calculator.get_database"):
        return RiskMetricsCalculator()


# =====================================================================
# validate_ticker (static method)
# =====================================================================


class TestValidateTicker:
    def test_valid_simple(self):
        assert RiskMetricsCalculator.validate_ticker("AAPL") is True

    def test_valid_with_dot(self):
        assert RiskMetricsCalculator.validate_ticker("BRK.A") is True

    def test_invalid_lowercase(self):
        # The method calls ticker.upper() so 'abc' becomes 'ABC'
        # which matches the pattern -- validate_ticker normalizes input
        assert RiskMetricsCalculator.validate_ticker("abc") is True

    def test_invalid_empty(self):
        assert RiskMetricsCalculator.validate_ticker("") is False

    def test_invalid_too_long(self):
        # 6 chars without dot is invalid (pattern requires 1-5)
        assert RiskMetricsCalculator.validate_ticker("TOOLNG") is False

    def test_valid_single_char(self):
        assert RiskMetricsCalculator.validate_ticker("A") is True

    def test_valid_five_chars(self):
        assert RiskMetricsCalculator.validate_ticker("ABCDE") is True

    def test_invalid_numbers(self):
        assert RiskMetricsCalculator.validate_ticker("AB1") is False


# =====================================================================
# calculate_daily_returns
# =====================================================================


class TestCalculateDailyReturns:
    def test_normal(self, calc):
        df = pd.DataFrame(
            {
                "date": pd.date_range("2024-01-01", periods=5),
                "close": [100.0, 110.0, 105.0, 115.0, 120.0],
            }
        )
        result = calc.calculate_daily_returns(df)
        assert len(result) == 4  # First row dropped (NaN from pct_change)
        # First return: (110-100)/100 * 100 = 10%
        assert abs(result.iloc[0]["return"] - 10.0) < 0.01

    def test_empty_dataframe(self, calc):
        df = pd.DataFrame(columns=["date", "close"])
        result = calc.calculate_daily_returns(df)
        assert result.empty

    def test_single_row(self, calc):
        df = pd.DataFrame(
            {"date": [pd.Timestamp("2024-01-01")], "close": [100.0]}
        )
        result = calc.calculate_daily_returns(df)
        # Single row -> pct_change produces NaN -> dropped
        assert result.empty


# =====================================================================
# calculate_beta
# =====================================================================


class TestCalculateBeta:
    def test_perfectly_correlated(self, calc):
        np.random.seed(42)
        returns = np.random.randn(100)
        stock = pd.Series(returns)
        benchmark = pd.Series(returns)
        beta = calc.calculate_beta(stock, benchmark)
        assert beta is not None
        assert abs(beta - 1.0) < 0.01

    def test_inverse_correlated(self, calc):
        np.random.seed(42)
        returns = np.random.randn(100)
        stock = pd.Series(-returns)
        benchmark = pd.Series(returns)
        beta = calc.calculate_beta(stock, benchmark)
        assert beta is not None
        assert abs(beta - (-1.0)) < 0.01

    def test_insufficient_data(self, calc):
        stock = pd.Series([1.0, 2.0, 3.0])
        benchmark = pd.Series([1.0, 2.0, 3.0])
        beta = calc.calculate_beta(stock, benchmark)
        assert beta is None  # < 12 points

    def test_zero_benchmark_variance(self, calc):
        stock = pd.Series([1.0] * 20 + [2.0] * 20)
        benchmark = pd.Series([5.0] * 40)  # No variance
        beta = calc.calculate_beta(stock, benchmark)
        assert beta is None  # Division by near-zero variance


# =====================================================================
# calculate_alpha
# =====================================================================


class TestCalculateAlpha:
    def test_positive_alpha(self, calc):
        # Stock returned 15%, benchmark 10%, beta=1.0, rf=4%
        # Alpha = 15 - 4 - 1.0*(10-4) = 15 - 4 - 6 = 5
        alpha = calc.calculate_alpha(15.0, 1.0, 10.0, 4.0)
        assert alpha is not None
        assert abs(alpha - 5.0) < 0.01

    def test_negative_alpha(self, calc):
        # Stock returned 5%, benchmark 10%, beta=1.0, rf=4%
        # Alpha = 5 - 4 - 1.0*(10-4) = 5 - 4 - 6 = -5
        alpha = calc.calculate_alpha(5.0, 1.0, 10.0, 4.0)
        assert alpha is not None
        assert abs(alpha - (-5.0)) < 0.01

    def test_none_beta(self, calc):
        alpha = calc.calculate_alpha(15.0, None, 10.0, 4.0)
        assert alpha is None


# =====================================================================
# calculate_sharpe_ratio
# =====================================================================


class TestCalculateSharpeRatio:
    def test_positive_return_above_rf(self, calc):
        # Return=15%, std=20%, rf=4%
        # Sharpe = (15-4)/20 = 0.55
        sharpe = calc.calculate_sharpe_ratio(15.0, 20.0, 4.0)
        assert sharpe is not None
        assert abs(sharpe - 0.55) < 0.01

    def test_negative_return(self, calc):
        # Return=-5%, std=20%, rf=4%
        # Sharpe = (-5-4)/20 = -0.45
        sharpe = calc.calculate_sharpe_ratio(-5.0, 20.0, 4.0)
        assert sharpe is not None
        assert abs(sharpe - (-0.45)) < 0.01

    def test_zero_std_dev(self, calc):
        sharpe = calc.calculate_sharpe_ratio(15.0, 0.0, 4.0)
        assert sharpe is None


# =====================================================================
# calculate_sortino_ratio
# =====================================================================


class TestCalculateSortinoRatio:
    def test_mixed_returns(self, calc):
        daily_returns = pd.Series(
            [1.0, -0.5, 2.0, -1.0, 0.5, -0.3, 1.5, -0.8]
        )
        sortino, downside_dev = calc.calculate_sortino_ratio(
            10.0, daily_returns, 4.0
        )
        assert sortino is not None
        assert downside_dev is not None
        assert sortino > 0  # Positive return above rf

    def test_all_positive(self, calc):
        daily_returns = pd.Series([1.0, 2.0, 0.5, 1.5, 3.0])
        sortino, downside_dev = calc.calculate_sortino_ratio(
            20.0, daily_returns, 4.0
        )
        # All positive returns -> no downside deviation -> None
        assert sortino is None
        assert downside_dev is None

    def test_all_negative(self, calc):
        daily_returns = pd.Series([-1.0, -2.0, -0.5, -1.5, -3.0])
        sortino, downside_dev = calc.calculate_sortino_ratio(
            -20.0, daily_returns, 4.0
        )
        assert sortino is not None
        assert downside_dev is not None
        assert sortino < 0  # Negative return below rf


# =====================================================================
# calculate_max_drawdown
# =====================================================================


class TestCalculateMaxDrawdown:
    def test_monotonically_increasing(self, calc):
        df = pd.DataFrame(
            {"close": [100.0, 110.0, 120.0, 130.0, 140.0]}
        )
        result = calc.calculate_max_drawdown(df)
        assert result is not None
        assert abs(result - 0.0) < 0.01

    def test_single_50_percent_drop(self, calc):
        df = pd.DataFrame({"close": [100.0, 50.0, 60.0]})
        result = calc.calculate_max_drawdown(df)
        assert result is not None
        assert abs(result - (-50.0)) < 0.01

    def test_empty_dataframe(self, calc):
        df = pd.DataFrame({"close": []})
        result = calc.calculate_max_drawdown(df)
        assert result is None

    def test_single_row(self, calc):
        df = pd.DataFrame({"close": [100.0]})
        result = calc.calculate_max_drawdown(df)
        assert result is None  # len < 2

    def test_multiple_drawdowns(self, calc):
        # Peak 100, drop to 80 (-20%), recover to 120, drop to 90 (-25%)
        df = pd.DataFrame(
            {"close": [100.0, 80.0, 120.0, 90.0]}
        )
        result = calc.calculate_max_drawdown(df)
        assert result is not None
        # Max drawdown is -25% (120 -> 90)
        assert abs(result - (-25.0)) < 0.01


# =====================================================================
# calculate_var_5
# =====================================================================


class TestCalculateVar5:
    def test_normal_data(self, calc):
        np.random.seed(42)
        daily_returns = pd.Series(np.random.randn(100) * 2)
        result = calc.calculate_var_5(daily_returns)
        assert result is not None
        assert result < 0  # VaR should be negative (loss)

    def test_insufficient_data(self, calc):
        daily_returns = pd.Series([1.0, -0.5, 0.3] * 10)  # 30 points < 60
        result = calc.calculate_var_5(daily_returns)
        assert result is None

    def test_exactly_60_points(self, calc):
        np.random.seed(42)
        daily_returns = pd.Series(np.random.randn(60))
        result = calc.calculate_var_5(daily_returns)
        assert result is not None


# =====================================================================
# calculate_annualized_return
# =====================================================================


class TestCalculateAnnualizedReturn:
    def test_normal(self, calc):
        # 100 -> 150 over 1 year = 50%
        result = calc.calculate_annualized_return(100.0, 150.0, 1.0)
        assert abs(result - 50.0) < 0.01

    def test_start_zero(self, calc):
        result = calc.calculate_annualized_return(0.0, 150.0, 1.0)
        assert result == 0.0

    def test_start_negative(self, calc):
        result = calc.calculate_annualized_return(-100.0, 150.0, 1.0)
        assert result == 0.0

    def test_years_zero(self, calc):
        result = calc.calculate_annualized_return(100.0, 150.0, 0.0)
        assert result == 0.0

    def test_years_negative(self, calc):
        result = calc.calculate_annualized_return(100.0, 150.0, -1.0)
        assert result == 0.0

    def test_multi_year(self, calc):
        # 100 -> 200 over 3 years
        # ((200/100)^(1/3) - 1) * 100 = 25.99%
        result = calc.calculate_annualized_return(100.0, 200.0, 3.0)
        assert abs(result - 25.99) < 0.1
