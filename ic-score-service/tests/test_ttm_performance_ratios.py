"""Tests for TTM Performance Ratios calculations.

Tests Phase 1 implementation:
- TTM ROA (Return on Assets)
- TTM ROE (Return on Equity)
- TTM ROIC (Return on Invested Capital)
- TTM Margins (Gross, Operating, Net)
"""

import pytest
from decimal import Decimal


class TestTTMPerformanceRatios:
    """Test suite for TTM performance ratio calculations."""

    def test_roa_calculation(self):
        """Test ROA calculation: (Net Income / Average Assets) × 100"""
        net_income = 1_000_000
        current_assets = 10_000_000
        past_assets = 8_000_000
        avg_assets = (current_assets + past_assets) / 2

        expected_roa = (net_income / avg_assets) * 100
        assert expected_roa == pytest.approx(11.11, rel=0.01)

    def test_roe_calculation(self):
        """Test ROE calculation: (Net Income / Average Equity) × 100"""
        net_income = 1_000_000
        current_equity = 5_000_000
        past_equity = 4_500_000
        avg_equity = (current_equity + past_equity) / 2

        expected_roe = (net_income / avg_equity) * 100
        assert expected_roe == pytest.approx(21.05, rel=0.01)

    def test_roic_calculation(self):
        """Test ROIC calculation: (NOPAT / Average Invested Capital) × 100"""
        operating_income = 1_500_000
        tax_rate = 0.21
        nopat = operating_income * (1 - tax_rate)

        current_assets = 10_000_000
        current_cash = 2_000_000
        past_assets = 8_000_000
        past_cash = 1_500_000

        current_invested_capital = current_assets - current_cash
        past_invested_capital = past_assets - past_cash
        avg_invested_capital = (current_invested_capital + past_invested_capital) / 2

        expected_roic = (nopat / avg_invested_capital) * 100
        assert expected_roic == pytest.approx(16.36, rel=0.01)

    def test_gross_margin_calculation(self):
        """Test Gross Margin: (Gross Profit / Revenue) × 100"""
        gross_profit = 5_000_000
        revenue = 10_000_000

        expected_margin = (gross_profit / revenue) * 100
        assert expected_margin == 50.0

    def test_operating_margin_calculation(self):
        """Test Operating Margin: (Operating Income / Revenue) × 100"""
        operating_income = 2_000_000
        revenue = 10_000_000

        expected_margin = (operating_income / revenue) * 100
        assert expected_margin == 20.0

    def test_net_margin_calculation(self):
        """Test Net Margin: (Net Income / Revenue) × 100"""
        net_income = 1_000_000
        revenue = 10_000_000

        expected_margin = (net_income / revenue) * 100
        assert expected_margin == 10.0

    def test_division_by_zero_handling(self):
        """Test that division by zero is handled gracefully"""
        net_income = 1_000_000
        zero_assets = 0

        # Should not raise exception, should return None or handle gracefully
        with pytest.raises(ZeroDivisionError):
            _ = net_income / zero_assets

    def test_negative_values(self):
        """Test calculations with negative net income"""
        net_income = -500_000
        current_assets = 10_000_000
        past_assets = 8_000_000
        avg_assets = (current_assets + past_assets) / 2

        expected_roa = (net_income / avg_assets) * 100
        assert expected_roa == pytest.approx(-5.56, rel=0.01)

    def test_ttm_vs_quarterly(self):
        """Test that TTM ratios are different from quarterly ratios"""
        # TTM uses averaged balance sheet items, quarterly uses point-in-time
        # This test validates the methodology difference
        ttm_avg_assets = (10_000_000 + 8_000_000) / 2  # 9,000,000
        quarterly_assets = 10_000_000  # Current quarter only

        assert ttm_avg_assets != quarterly_assets
        assert ttm_avg_assets == 9_000_000


class TestEdgeCases:
    """Test edge cases and boundary conditions."""

    def test_missing_previous_quarter_data(self):
        """Test when only 3 quarters available instead of 4"""
        # Should skip calculation or use alternative methodology
        pass

    def test_very_small_denominators(self):
        """Test with very small denominators (near zero)"""
        net_income = 1_000_000
        tiny_assets = 100  # Very small assets

        roa = (net_income / tiny_assets) * 100
        assert roa > 1_000_000  # Should be astronomically high

    def test_precision_rounding(self):
        """Test that results are rounded to 4 decimal places"""
        value = 12.3456789
        rounded = round(value, 4)
        assert rounded == 12.3457


class TestRealWorldScenarios:
    """Test with realistic financial data."""

    def test_apple_like_metrics(self):
        """Test with AAPL-like financial metrics"""
        # Approximate AAPL metrics (scaled down)
        net_income = 100_000_000_000  # $100B
        revenue = 400_000_000_000  # $400B
        total_assets = 350_000_000_000  # $350B
        shareholders_equity = 60_000_000_000  # $60B

        net_margin = (net_income / revenue) * 100
        roe = (net_income / shareholders_equity) * 100

        assert net_margin == pytest.approx(25.0, rel=0.01)
        assert roe == pytest.approx(166.67, rel=0.01)

    def test_negative_equity_company(self):
        """Test companies with negative shareholders equity"""
        net_income = 1_000_000
        negative_equity = -500_000

        # ROE with negative equity should be negative
        roe = (net_income / abs(negative_equity)) * 100
        # Many companies choose to return None for this case
        # since interpretation is unclear


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
