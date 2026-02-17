"""Tests for pure functions in fundamental_metrics_calculator.py.

Phase 1A: No database dependencies. Tests only pure calculation functions.
"""

from decimal import Decimal
from unittest.mock import MagicMock, patch

import pytest

from pipelines.fundamental_metrics_calculator import (
    FundamentalMetricsCalculator,
    calculate_cagr,
    calculate_dividend_growth_rate,
    calculate_ebitda_margin,
    calculate_ev_to_fcf,
    calculate_interest_coverage,
    calculate_net_debt_to_ebitda,
    calculate_payout_ratio,
    calculate_yoy_growth,
    count_consecutive_dividend_growth_years,
)


# =====================================================================
# calculate_ebitda_margin
# =====================================================================


class TestCalculateEbitdaMargin:
    def test_normal(self):
        result = calculate_ebitda_margin(Decimal("1000"), Decimal("5000"))
        assert result == Decimal("20")

    def test_none_ebitda(self):
        assert calculate_ebitda_margin(None, Decimal("5000")) is None

    def test_none_revenue(self):
        assert calculate_ebitda_margin(Decimal("1000"), None) is None

    def test_zero_revenue(self):
        assert calculate_ebitda_margin(Decimal("1000"), Decimal("0")) is None

    def test_negative_margin(self):
        result = calculate_ebitda_margin(Decimal("-500"), Decimal("5000"))
        # Negative EBITDA is falsy for Decimal(0) check in the function
        # The function uses `not ebitda` which is False for negative Decimal
        assert result is not None
        assert result == Decimal("-10")


# =====================================================================
# calculate_ev_to_fcf
# =====================================================================


class TestCalculateEvToFcf:
    def test_normal(self):
        result = calculate_ev_to_fcf(Decimal("1000"), Decimal("100"))
        assert result == Decimal("10")

    def test_none_ev(self):
        assert calculate_ev_to_fcf(None, Decimal("100")) is None

    def test_none_fcf(self):
        assert calculate_ev_to_fcf(Decimal("1000"), None) is None

    def test_zero_fcf(self):
        assert calculate_ev_to_fcf(Decimal("1000"), Decimal("0")) is None

    def test_negative_fcf(self):
        assert calculate_ev_to_fcf(Decimal("1000"), Decimal("-50")) is None


# =====================================================================
# calculate_yoy_growth
# =====================================================================


class TestCalculateYoyGrowth:
    def test_revenue_normal(self):
        result = calculate_yoy_growth(
            Decimal("120"), Decimal("100"), metric_type="revenue"
        )
        assert result == Decimal("20")

    def test_none_current(self):
        assert calculate_yoy_growth(None, Decimal("100")) is None

    def test_none_prior(self):
        assert calculate_yoy_growth(Decimal("120"), None) is None

    def test_zero_prior(self):
        assert (
            calculate_yoy_growth(Decimal("120"), Decimal("0")) is None
        )

    def test_eps_normal(self):
        result = calculate_yoy_growth(
            Decimal("2.5"), Decimal("2"), metric_type="eps"
        )
        assert result == Decimal("25")

    def test_eps_sign_change_loss_to_profit(self):
        result = calculate_yoy_growth(
            Decimal("1"), Decimal("-1"), metric_type="eps"
        )
        assert result is None

    def test_eps_sign_change_profit_to_loss(self):
        result = calculate_yoy_growth(
            Decimal("-1"), Decimal("1"), metric_type="eps"
        )
        assert result is None

    def test_eps_both_negative_loss_narrowing(self):
        # -2 -> -1 is 50% improvement
        result = calculate_yoy_growth(
            Decimal("-1"), Decimal("-2"), metric_type="eps"
        )
        assert result == Decimal("50")

    def test_eps_both_negative_loss_widening(self):
        # -1 -> -2: ((abs(-1) - abs(-2)) / abs(-1)) * 100 = -100%
        result = calculate_yoy_growth(
            Decimal("-2"), Decimal("-1"), metric_type="eps"
        )
        assert result == Decimal("-100")

    def test_fcf_sign_change(self):
        result = calculate_yoy_growth(
            Decimal("100"), Decimal("-50"), metric_type="fcf"
        )
        assert result is None

    def test_revenue_default_metric_type(self):
        # Default metric_type is "revenue" -- no sign-change handling
        result = calculate_yoy_growth(Decimal("120"), Decimal("100"))
        assert result == Decimal("20")


# =====================================================================
# calculate_cagr
# =====================================================================


class TestCalculateCagr:
    def test_normal(self):
        result = calculate_cagr(Decimal("100"), Decimal("150"), 3)
        # (150/100)^(1/3) - 1 = ~14.47%
        assert result is not None
        assert abs(float(result) - 14.4714) < 0.01

    def test_start_zero(self):
        assert calculate_cagr(Decimal("0"), Decimal("150"), 3) is None

    def test_start_negative(self):
        assert calculate_cagr(Decimal("-100"), Decimal("150"), 3) is None

    def test_end_zero(self):
        assert calculate_cagr(Decimal("100"), Decimal("0"), 3) is None

    def test_end_negative(self):
        assert calculate_cagr(Decimal("100"), Decimal("-50"), 3) is None

    def test_years_zero(self):
        assert calculate_cagr(Decimal("100"), Decimal("150"), 0) is None

    def test_years_negative(self):
        assert calculate_cagr(Decimal("100"), Decimal("150"), -1) is None

    def test_one_year_period(self):
        result = calculate_cagr(Decimal("100"), Decimal("150"), 1)
        # (150/100)^(1/1) - 1 = 50%
        assert result is not None
        assert abs(float(result) - 50.0) < 0.01

    def test_none_start(self):
        assert calculate_cagr(None, Decimal("150"), 3) is None

    def test_none_end(self):
        assert calculate_cagr(Decimal("100"), None, 3) is None


# =====================================================================
# calculate_payout_ratio
# =====================================================================


class TestCalculatePayoutRatio:
    def test_normal(self):
        result = calculate_payout_ratio(Decimal("2"), Decimal("5"))
        assert result == Decimal("40")

    def test_none_dividend(self):
        assert calculate_payout_ratio(None, Decimal("5")) is None

    def test_none_eps(self):
        assert calculate_payout_ratio(Decimal("2"), None) is None

    def test_eps_zero(self):
        assert calculate_payout_ratio(Decimal("2"), Decimal("0")) is None

    def test_eps_negative(self):
        assert calculate_payout_ratio(Decimal("2"), Decimal("-3")) is None


# =====================================================================
# calculate_dividend_growth_rate
# =====================================================================


class TestCalculateDividendGrowthRate:
    def test_one_year_growth(self):
        result = calculate_dividend_growth_rate(
            Decimal("2.5"), Decimal("2"), years_diff=1
        )
        assert result == Decimal("25")

    def test_multi_year_cagr(self):
        # 3-year CAGR from 2 to 2.5
        result = calculate_dividend_growth_rate(
            Decimal("2.5"), Decimal("2"), years_diff=3
        )
        assert result is not None
        # Should be CAGR: (2.5/2)^(1/3) - 1 ~= 7.72%
        assert abs(float(result) - 7.7217) < 0.01

    def test_zero_prior(self):
        assert (
            calculate_dividend_growth_rate(
                Decimal("2.5"), Decimal("0"), years_diff=1
            )
            is None
        )

    def test_zero_years_diff(self):
        assert (
            calculate_dividend_growth_rate(
                Decimal("2.5"), Decimal("2"), years_diff=0
            )
            is None
        )

    def test_none_current(self):
        assert (
            calculate_dividend_growth_rate(
                None, Decimal("2"), years_diff=1
            )
            is None
        )

    def test_none_prior(self):
        assert (
            calculate_dividend_growth_rate(
                Decimal("2.5"), None, years_diff=1
            )
            is None
        )


# =====================================================================
# count_consecutive_dividend_growth_years
# =====================================================================


class TestCountConsecutiveDividendGrowthYears:
    def test_empty_list(self):
        assert count_consecutive_dividend_growth_years([]) == 0

    def test_single_item(self):
        assert (
            count_consecutive_dividend_growth_years(
                [{"year": 2023, "amount": Decimal("2")}]
            )
            == 0
        )

    def test_three_years_of_growth(self):
        history = [
            {"year": 2023, "amount": Decimal("3")},
            {"year": 2022, "amount": Decimal("2.5")},
            {"year": 2021, "amount": Decimal("2")},
        ]
        assert count_consecutive_dividend_growth_years(history) == 2

    def test_gap_in_years_stops_counting(self):
        history = [
            {"year": 2023, "amount": Decimal("3")},
            {"year": 2022, "amount": Decimal("2.5")},
            {"year": 2020, "amount": Decimal("2")},  # Gap: 2021 missing
        ]
        assert count_consecutive_dividend_growth_years(history) == 1

    def test_flat_dividend_stops(self):
        history = [
            {"year": 2023, "amount": Decimal("3")},
            {"year": 2022, "amount": Decimal("3")},  # Flat
            {"year": 2021, "amount": Decimal("2")},
        ]
        assert count_consecutive_dividend_growth_years(history) == 0

    def test_held_flat_then_grew(self):
        history = [
            {"year": 2024, "amount": Decimal("4")},
            {"year": 2023, "amount": Decimal("3")},
            {"year": 2022, "amount": Decimal("3")},  # Flat here
            {"year": 2021, "amount": Decimal("2")},
        ]
        # Sorted desc: 2024>2023 (grew), 2023>2022 (flat) -> stops
        assert count_consecutive_dividend_growth_years(history) == 1

    def test_decrease_stops(self):
        history = [
            {"year": 2023, "amount": Decimal("2")},
            {"year": 2022, "amount": Decimal("3")},  # Decreased
            {"year": 2021, "amount": Decimal("2.5")},
        ]
        assert count_consecutive_dividend_growth_years(history) == 0


# =====================================================================
# calculate_interest_coverage
# =====================================================================


class TestCalculateInterestCoverage:
    def test_normal(self):
        result = calculate_interest_coverage(
            Decimal("1000"), Decimal("200")
        )
        assert result == Decimal("5")

    def test_none_operating_income(self):
        assert calculate_interest_coverage(None, Decimal("200")) is None

    def test_none_interest(self):
        assert calculate_interest_coverage(Decimal("1000"), None) is None

    def test_interest_zero(self):
        assert (
            calculate_interest_coverage(Decimal("1000"), Decimal("0"))
            is None
        )

    def test_interest_negative(self):
        assert (
            calculate_interest_coverage(Decimal("1000"), Decimal("-100"))
            is None
        )


# =====================================================================
# calculate_net_debt_to_ebitda
# =====================================================================


class TestCalculateNetDebtToEbitda:
    def test_normal(self):
        result = calculate_net_debt_to_ebitda(
            Decimal("5000"), Decimal("1000"), Decimal("2000")
        )
        # (5000 - 1000) / 2000 = 2.0
        assert result == Decimal("2")

    def test_net_cash_position(self):
        result = calculate_net_debt_to_ebitda(
            Decimal("1000"), Decimal("3000"), Decimal("2000")
        )
        # (1000 - 3000) / 2000 = -1.0 (negative = net cash)
        assert result == Decimal("-1")

    def test_none_total_debt(self):
        assert (
            calculate_net_debt_to_ebitda(
                None, Decimal("1000"), Decimal("2000")
            )
            is None
        )

    def test_none_cash(self):
        assert (
            calculate_net_debt_to_ebitda(
                Decimal("5000"), None, Decimal("2000")
            )
            is None
        )

    def test_none_ebitda(self):
        assert (
            calculate_net_debt_to_ebitda(
                Decimal("5000"), Decimal("1000"), None
            )
            is None
        )

    def test_ebitda_zero(self):
        assert (
            calculate_net_debt_to_ebitda(
                Decimal("5000"), Decimal("1000"), Decimal("0")
            )
            is None
        )

    def test_ebitda_negative(self):
        assert (
            calculate_net_debt_to_ebitda(
                Decimal("5000"), Decimal("1000"), Decimal("-500")
            )
            is None
        )


# =====================================================================
# _calculate_data_quality_score (instance method)
# =====================================================================


class TestCalculateDataQualityScore:
    @patch("pipelines.fundamental_metrics_calculator.get_database")
    def test_full_data(self, mock_db):
        calc = FundamentalMetricsCalculator()
        ttm_data = {
            "revenue": Decimal("1000"),
            "net_income": Decimal("200"),
            "eps_diluted": Decimal("5"),
            "free_cash_flow": Decimal("150"),
            "shareholders_equity": Decimal("3000"),
            "total_assets": Decimal("8000"),
            "operating_income": Decimal("300"),
        }
        growth_rates = {
            "revenue_growth_yoy": Decimal("15"),
            "eps_growth_yoy": Decimal("10"),
            "revenue_growth_3y_cagr": Decimal("12"),
            "revenue_growth_5y_cagr": Decimal("11"),
            "eps_growth_5y_cagr": Decimal("9"),
        }
        result = calc._calculate_data_quality_score(ttm_data, growth_rates)
        # 7 TTM fields + 3 growth fields + 2 (5y rev) + 2 (5y eps) = 14/15
        assert result == Decimal("93.33")

    @patch("pipelines.fundamental_metrics_calculator.get_database")
    def test_minimal_data(self, mock_db):
        calc = FundamentalMetricsCalculator()
        ttm_data = {"revenue": Decimal("1000")}
        growth_rates = {}
        result = calc._calculate_data_quality_score(ttm_data, growth_rates)
        # 1/15 = 6.67%
        assert result == Decimal("6.67")

    @patch("pipelines.fundamental_metrics_calculator.get_database")
    def test_empty_data(self, mock_db):
        calc = FundamentalMetricsCalculator()
        ttm_data = {}
        growth_rates = {}
        result = calc._calculate_data_quality_score(ttm_data, growth_rates)
        assert result == Decimal("0")
