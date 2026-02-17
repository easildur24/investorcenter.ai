"""Tests for pure functions in fair_value_calculator.py.

No database dependencies. Tests only pure calculation functions:
- calculate_cost_of_equity (CAPM)
- estimate_cost_of_debt
- calculate_wacc
- calculate_dcf_fair_value (two-stage DCF)
- calculate_graham_number
- calculate_earnings_power_value (EPV)
"""

import math
from decimal import Decimal

import pytest

from pipelines.fair_value_calculator import (
    calculate_cost_of_equity,
    calculate_dcf_fair_value,
    calculate_earnings_power_value,
    calculate_graham_number,
    calculate_wacc,
    estimate_cost_of_debt,
)


# =====================================================================
# calculate_cost_of_equity (CAPM: Re = Rf + beta * MRP)
# =====================================================================


class TestCalculateCostOfEquity:
    def test_normal_beta_1(self):
        # Re = 0.045 + 1.0 * 0.055 = 0.10
        result = calculate_cost_of_equity(Decimal("1.0"), Decimal("0.045"))
        assert result == Decimal("0.100")

    def test_high_beta(self):
        # Re = 0.045 + 1.5 * 0.055 = 0.1275
        result = calculate_cost_of_equity(Decimal("1.5"), Decimal("0.045"))
        normal = calculate_cost_of_equity(Decimal("1.0"), Decimal("0.045"))
        assert result > normal
        assert result == Decimal("0.1275")

    def test_low_beta(self):
        # Re = 0.045 + 0.5 * 0.055 = 0.0725
        result = calculate_cost_of_equity(Decimal("0.5"), Decimal("0.045"))
        normal = calculate_cost_of_equity(Decimal("1.0"), Decimal("0.045"))
        assert result < normal
        assert result == Decimal("0.0725")

    def test_zero_beta(self):
        # Re = 0.045 + 0 * 0.055 = 0.045 (just risk-free rate)
        result = calculate_cost_of_equity(Decimal("0"), Decimal("0.045"))
        assert result == Decimal("0.045")

    def test_negative_beta(self):
        # Gold stocks can have negative beta
        # Re = 0.045 + (-0.3) * 0.055 = 0.045 - 0.0165 = 0.0285
        result = calculate_cost_of_equity(Decimal("-0.3"), Decimal("0.045"))
        assert result < Decimal("0.045")
        assert result == Decimal("0.0285")


# =====================================================================
# estimate_cost_of_debt
# =====================================================================


class TestEstimateCostOfDebt:
    def test_normal(self):
        # 50M / 1B = 0.05
        result = estimate_cost_of_debt(
            Decimal("50000000"), Decimal("1000000000")
        )
        assert result == Decimal("0.05")

    def test_high_rate_capped(self):
        # 200M / 500M = 0.40, capped at 0.15
        result = estimate_cost_of_debt(
            Decimal("200000000"), Decimal("500000000")
        )
        assert result == Decimal("0.15")

    def test_low_rate_floored(self):
        # 1M / 1B = 0.001, floored at 0.02
        result = estimate_cost_of_debt(
            Decimal("1000000"), Decimal("1000000000")
        )
        assert result == Decimal("0.02")

    def test_none_interest(self):
        result = estimate_cost_of_debt(None, Decimal("1000000000"))
        assert result == Decimal("0.05")

    def test_none_debt(self):
        result = estimate_cost_of_debt(Decimal("50000000"), None)
        assert result == Decimal("0.05")

    def test_zero_debt(self):
        result = estimate_cost_of_debt(Decimal("50000000"), Decimal("0"))
        assert result == Decimal("0.05")


# =====================================================================
# calculate_wacc
# =====================================================================


class TestCalculateWacc:
    def test_all_equity(self):
        # No debt: WACC = (1000/1000 * 0.10) + (0/1000 * Rd * (1-T)) = 0.10
        result = calculate_wacc(
            cost_of_equity=Decimal("0.10"),
            cost_of_debt=Decimal("0.05"),
            total_debt=Decimal("0"),
            market_cap=Decimal("1000"),
        )
        assert result == Decimal("0.10")

    def test_balanced(self):
        # 50/50: WACC = (0.5 * 0.10) + (0.5 * 0.05 * 0.79)
        #       = 0.05 + 0.01975 = 0.06975
        result = calculate_wacc(
            cost_of_equity=Decimal("0.10"),
            cost_of_debt=Decimal("0.05"),
            total_debt=Decimal("500"),
            market_cap=Decimal("500"),
        )
        expected = Decimal("0.06975")
        assert result == expected

    def test_high_leverage(self):
        # 80% debt, 20% equity
        # WACC = (0.2 * 0.12) + (0.8 * 0.06 * 0.79)
        #      = 0.024 + 0.03792 = 0.06192
        result = calculate_wacc(
            cost_of_equity=Decimal("0.12"),
            cost_of_debt=Decimal("0.06"),
            total_debt=Decimal("800"),
            market_cap=Decimal("200"),
        )
        expected = Decimal("0.06192")
        assert result == expected

    def test_floored_at_5_pct(self):
        # Very low cost of equity and debt should floor at 0.05
        result = calculate_wacc(
            cost_of_equity=Decimal("0.01"),
            cost_of_debt=Decimal("0.01"),
            total_debt=Decimal("500"),
            market_cap=Decimal("500"),
        )
        assert result == Decimal("0.05")

    def test_capped_at_20_pct(self):
        # Very high cost of equity should cap at 0.20
        result = calculate_wacc(
            cost_of_equity=Decimal("0.50"),
            cost_of_debt=Decimal("0.15"),
            total_debt=Decimal("0"),
            market_cap=Decimal("1000"),
        )
        assert result == Decimal("0.20")

    def test_zero_total_value(self):
        # Both debt and equity = 0 -> default 0.10
        result = calculate_wacc(
            cost_of_equity=Decimal("0.10"),
            cost_of_debt=Decimal("0.05"),
            total_debt=Decimal("0"),
            market_cap=Decimal("0"),
        )
        assert result == Decimal("0.10")

    def test_tax_shield(self):
        # Verify the debt component uses (1 - tax_rate)
        # With custom tax_rate=0.30:
        # WACC = (0.5 * 0.10) + (0.5 * 0.10 * 0.70)
        #      = 0.05 + 0.035 = 0.085
        result = calculate_wacc(
            cost_of_equity=Decimal("0.10"),
            cost_of_debt=Decimal("0.10"),
            total_debt=Decimal("500"),
            market_cap=Decimal("500"),
            tax_rate=Decimal("0.30"),
        )
        expected = Decimal("0.085")
        assert result == expected


# =====================================================================
# calculate_dcf_fair_value (Two-Stage DCF)
# =====================================================================


class TestCalculateDcfFairValue:
    def test_positive_fcf(self):
        # Realistic inputs: $5B FCF, 10% growth, 2.5% terminal, 10% WACC
        result, details = calculate_dcf_fair_value(
            fcf_ttm=Decimal("5000000000"),
            growth_rate_high=Decimal("0.10"),
            growth_rate_terminal=Decimal("0.025"),
            wacc=Decimal("0.10"),
            shares_outstanding=1000000000,
            net_debt=Decimal("10000000000"),
        )
        assert result is not None
        assert result > Decimal("0")
        assert "error" not in details

    def test_negative_fcf_returns_none(self):
        result, details = calculate_dcf_fair_value(
            fcf_ttm=Decimal("-100000000"),
            growth_rate_high=Decimal("0.10"),
            growth_rate_terminal=Decimal("0.025"),
            wacc=Decimal("0.10"),
            shares_outstanding=1000000000,
            net_debt=Decimal("0"),
        )
        assert result is None
        assert "error" in details

    def test_wacc_equals_terminal_returns_none(self):
        result, details = calculate_dcf_fair_value(
            fcf_ttm=Decimal("5000000000"),
            growth_rate_high=Decimal("0.10"),
            growth_rate_terminal=Decimal("0.10"),
            wacc=Decimal("0.10"),
            shares_outstanding=1000000000,
            net_debt=Decimal("0"),
        )
        assert result is None
        assert "error" in details

    def test_zero_shares_returns_none(self):
        result, details = calculate_dcf_fair_value(
            fcf_ttm=Decimal("5000000000"),
            growth_rate_high=Decimal("0.10"),
            growth_rate_terminal=Decimal("0.025"),
            wacc=Decimal("0.10"),
            shares_outstanding=0,
            net_debt=Decimal("0"),
        )
        assert result is None
        assert "error" in details

    def test_high_growth_rate(self):
        # 30% growth should produce a higher fair value than 10% growth
        result_high, _ = calculate_dcf_fair_value(
            fcf_ttm=Decimal("5000000000"),
            growth_rate_high=Decimal("0.30"),
            growth_rate_terminal=Decimal("0.025"),
            wacc=Decimal("0.10"),
            shares_outstanding=1000000000,
            net_debt=Decimal("0"),
        )
        result_normal, _ = calculate_dcf_fair_value(
            fcf_ttm=Decimal("5000000000"),
            growth_rate_high=Decimal("0.10"),
            growth_rate_terminal=Decimal("0.025"),
            wacc=Decimal("0.10"),
            shares_outstanding=1000000000,
            net_debt=Decimal("0"),
        )
        assert result_high is not None
        assert result_normal is not None
        assert result_high > result_normal

    def test_zero_growth_rate(self):
        # 0% high growth is still valid (no growth DCF)
        result, details = calculate_dcf_fair_value(
            fcf_ttm=Decimal("5000000000"),
            growth_rate_high=Decimal("0.0"),
            growth_rate_terminal=Decimal("0.025"),
            wacc=Decimal("0.10"),
            shares_outstanding=1000000000,
            net_debt=Decimal("0"),
        )
        assert result is not None
        assert result > Decimal("0")

    def test_negative_equity_returns_none(self):
        # Huge net_debt exceeding enterprise value -> negative equity
        result, details = calculate_dcf_fair_value(
            fcf_ttm=Decimal("1000000"),
            growth_rate_high=Decimal("0.05"),
            growth_rate_terminal=Decimal("0.025"),
            wacc=Decimal("0.10"),
            shares_outstanding=1000000,
            net_debt=Decimal("999999999999"),
        )
        assert result is None
        assert "error" in details

    def test_details_dict_structure(self):
        result, details = calculate_dcf_fair_value(
            fcf_ttm=Decimal("5000000000"),
            growth_rate_high=Decimal("0.10"),
            growth_rate_terminal=Decimal("0.025"),
            wacc=Decimal("0.10"),
            shares_outstanding=1000000000,
            net_debt=Decimal("10000000000"),
        )
        assert result is not None
        assert "inputs" in details
        assert "projected_fcf" in details
        assert "terminal_value" in details
        assert "pv_terminal_value" in details
        assert "sum_pv_fcf" in details
        assert "enterprise_value" in details
        assert "equity_value" in details
        assert "fair_value_per_share" in details
        assert len(details["projected_fcf"]) == 10


# =====================================================================
# calculate_graham_number
# =====================================================================


class TestCalculateGrahamNumber:
    def test_normal(self):
        # sqrt(22.5 * 6 * 40) = sqrt(5400) = 73.4847...
        result = calculate_graham_number(Decimal("6"), Decimal("40"))
        assert result is not None
        expected = round(math.sqrt(22.5 * 6 * 40), 2)
        assert float(result) == pytest.approx(expected, abs=0.01)

    def test_negative_eps_returns_none(self):
        result = calculate_graham_number(Decimal("-2"), Decimal("40"))
        assert result is None

    def test_negative_bvps_returns_none(self):
        result = calculate_graham_number(Decimal("6"), Decimal("-10"))
        assert result is None

    def test_zero_eps_returns_none(self):
        result = calculate_graham_number(Decimal("0"), Decimal("40"))
        assert result is None

    def test_none_eps_returns_none(self):
        result = calculate_graham_number(None, Decimal("40"))
        assert result is None

    def test_realistic_aapl(self):
        # AAPL-like: EPS=7.58, BVPS=4.38
        # sqrt(22.5 * 7.58 * 4.38) = sqrt(746.685) ~= 27.33
        result = calculate_graham_number(Decimal("7.58"), Decimal("4.38"))
        assert result is not None
        expected = round(math.sqrt(22.5 * 7.58 * 4.38), 2)
        assert float(result) == pytest.approx(expected, abs=0.01)
        # Reasonable range check
        assert Decimal("20") < result < Decimal("40")


# =====================================================================
# calculate_earnings_power_value (EPV)
# =====================================================================


class TestCalculateEarningsPowerValue:
    def test_normal(self):
        # EBIT=10B, WACC=0.10, net_debt=5B, shares=1B
        # NOPAT = 10B * (1-0.21) = 7.9B
        # EV = 7.9B / 0.10 = 79B
        # Equity = 79B - 5B = 74B
        # Per share = 74B / 1B = 74.0
        result = calculate_earnings_power_value(
            normalized_ebit=Decimal("10000000000"),
            wacc=Decimal("0.10"),
            net_debt=Decimal("5000000000"),
            shares_outstanding=1000000000,
        )
        assert result is not None
        assert float(result) == pytest.approx(74.0, abs=0.01)

    def test_negative_ebit_returns_none(self):
        result = calculate_earnings_power_value(
            normalized_ebit=Decimal("-5000000000"),
            wacc=Decimal("0.10"),
            net_debt=Decimal("5000000000"),
            shares_outstanding=1000000000,
        )
        assert result is None

    def test_overleveraged_returns_none(self):
        # net_debt > enterprise_value -> negative equity
        # NOPAT = 1B * 0.79 = 0.79B, EV = 0.79B / 0.10 = 7.9B
        # Equity = 7.9B - 50B = -42.1B -> None
        result = calculate_earnings_power_value(
            normalized_ebit=Decimal("1000000000"),
            wacc=Decimal("0.10"),
            net_debt=Decimal("50000000000"),
            shares_outstanding=1000000000,
        )
        assert result is None
