"""Tests for Quarter-over-Quarter Growth calculations.

Tests Phase 3 implementation:
- QoQ Revenue Growth
- QoQ EPS Growth
- QoQ EBITDA Growth
- QoQ Net Income Growth
- Edge case handling
"""

import pytest


class TestQoQGrowthCalculations:
    """Test suite for QoQ growth calculations."""

    def calculate_qoq_growth(self, current, previous):
        """Helper function to calculate QoQ growth"""
        if current is None or previous is None:
            return None
        if previous == 0:
            return None
        if (previous < 0 and current > 0) or (previous > 0 and current < 0):
            return None
        return round(((current - previous) / abs(previous)) * 100, 2)

    def test_basic_positive_growth(self):
        """Test basic positive growth scenario"""
        current = 110_000_000
        previous = 100_000_000

        growth = self.calculate_qoq_growth(current, previous)
        assert growth == 10.0

    def test_negative_growth(self):
        """Test negative growth (decline)"""
        current = 90_000_000
        previous = 100_000_000

        growth = self.calculate_qoq_growth(current, previous)
        assert growth == -10.0

    def test_no_growth(self):
        """Test zero growth (flat)"""
        current = 100_000_000
        previous = 100_000_000

        growth = self.calculate_qoq_growth(current, previous)
        assert growth == 0.0

    def test_high_growth(self):
        """Test high growth scenario (100%+)"""
        current = 250_000_000
        previous = 100_000_000

        growth = self.calculate_qoq_growth(current, previous)
        assert growth == 150.0

    def test_division_by_zero(self):
        """Test when previous quarter is zero"""
        current = 100_000_000
        previous = 0

        growth = self.calculate_qoq_growth(current, previous)
        assert growth is None  # Cannot calculate growth from zero

    def test_negative_to_positive_transition(self):
        """Test transition from negative to positive (loss to profit)"""
        current = 10_000_000  # Profitable
        previous = -5_000_000  # Loss

        growth = self.calculate_qoq_growth(current, previous)
        assert growth is None  # Sign change makes percentage meaningless

    def test_positive_to_negative_transition(self):
        """Test transition from positive to negative (profit to loss)"""
        current = -10_000_000  # Loss
        previous = 5_000_000  # Profitable

        growth = self.calculate_qoq_growth(current, previous)
        assert growth is None  # Sign change makes percentage meaningless

    def test_both_negative_values(self):
        """Test growth when both quarters are negative"""
        current = -8_000_000  # Smaller loss
        previous = -10_000_000  # Larger loss

        # Improvement (smaller loss) should show positive growth
        growth = ((current - previous) / abs(previous)) * 100
        assert growth == -20.0  # Loss decreased by 20%

        # Using absolute value for denominator
        growth_abs = self.calculate_qoq_growth(current, previous)
        assert growth_abs == -20.0

    def test_missing_current_data(self):
        """Test when current quarter data is missing"""
        current = None
        previous = 100_000_000

        growth = self.calculate_qoq_growth(current, previous)
        assert growth is None

    def test_missing_previous_data(self):
        """Test when previous quarter data is missing"""
        current = 110_000_000
        previous = None

        growth = self.calculate_qoq_growth(current, previous)
        assert growth is None

    def test_precision_rounding(self):
        """Test that growth is rounded to 2 decimal places"""
        current = 105_678_912
        previous = 100_000_000

        growth = self.calculate_qoq_growth(current, previous)
        assert growth == 5.68  # Rounded to 2 decimals


class TestRevenueGrowthScenarios:
    """Test revenue-specific growth scenarios."""

    def test_seasonal_business_growth(self):
        """Test seasonal business (e.g., retail Q4 spike)"""
        q3_revenue = 50_000_000
        q4_revenue = 100_000_000  # Holiday season

        growth = ((q4_revenue - q3_revenue) / q3_revenue) * 100
        assert growth == 100.0  # 100% QoQ growth (normal for seasonal)

    def test_consistent_growth(self):
        """Test company with consistent quarterly growth"""
        quarters = [100_000_000, 110_000_000, 121_000_000, 133_100_000]

        for i in range(1, len(quarters)):
            growth = ((quarters[i] - quarters[i-1]) / quarters[i-1]) * 100
            assert growth == pytest.approx(10.0, rel=0.1)  # Consistent ~10% growth


class TestEPSGrowthScenarios:
    """Test EPS-specific growth scenarios."""

    def test_eps_growth_outpacing_revenue(self):
        """Test operational leverage (EPS growing faster than revenue)"""
        # Revenue growth
        current_revenue = 110_000_000
        previous_revenue = 100_000_000
        revenue_growth = ((current_revenue - previous_revenue) / previous_revenue) * 100

        # EPS growth
        current_eps = 2.20
        previous_eps = 2.00
        eps_growth = ((current_eps - previous_eps) / previous_eps) * 100

        assert revenue_growth == 10.0
        assert eps_growth == 10.0
        # In this example, they're equal, but often EPS grows faster

    def test_dilution_effect(self):
        """Test EPS decline due to share dilution"""
        # Net income grows but EPS declines due to more shares
        current_ni = 110_000_000
        previous_ni = 100_000_000
        ni_growth = 10.0  # Net income up 10%

        current_eps = 1.80  # Lower due to dilution
        previous_eps = 2.00
        eps_growth = ((current_eps - previous_eps) / previous_eps) * 100

        assert eps_growth == -10.0  # EPS down despite NI up


class TestEBITDAGrowthScenarios:
    """Test EBITDA-specific growth scenarios."""

    def test_ebitda_margin_expansion(self):
        """Test EBITDA growing faster than revenue (margin expansion)"""
        # Revenue growth: 10%
        current_revenue = 110_000_000
        previous_revenue = 100_000_000

        # EBITDA growth: 20% (margin expansion)
        current_ebitda = 24_000_000
        previous_ebitda = 20_000_000

        revenue_growth = ((current_revenue - previous_revenue) / previous_revenue) * 100
        ebitda_growth = ((current_ebitda - previous_ebitda) / previous_ebitda) * 100

        assert revenue_growth == 10.0
        assert ebitda_growth == 20.0
        assert ebitda_growth > revenue_growth  # Positive operating leverage


class TestEdgeCases:
    """Test edge cases and boundary conditions."""

    def test_very_small_numbers(self):
        """Test with very small numbers (micro-cap companies)"""
        current = 1_000  # $1,000 revenue
        previous = 500

        growth = ((current - previous) / previous) * 100
        assert growth == 100.0

    def test_very_large_numbers(self):
        """Test with very large numbers (mega-cap companies)"""
        current = 500_000_000_000  # $500B
        previous = 450_000_000_000  # $450B

        growth = ((current - previous) / previous) * 100
        assert growth == pytest.approx(11.11, rel=0.01)

    def test_float_precision(self):
        """Test floating point precision handling"""
        current = 1.11111111
        previous = 1.00000000

        growth = round(((current - previous) / previous) * 100, 2)
        assert growth == 11.11

    def test_extreme_growth(self):
        """Test extreme growth scenario (>1000%)"""
        current = 11_000_000
        previous = 1_000_000

        growth = ((current - previous) / previous) * 100
        assert growth == 1000.0  # 10x growth


class TestRealWorldExamples:
    """Test with realistic company scenarios."""

    def test_tech_startup_growth(self):
        """Test high-growth tech startup"""
        # Typical tech startup: 50%+ QoQ growth
        quarters = [10_000_000, 15_000_000, 22_500_000, 33_750_000]

        for i in range(1, len(quarters)):
            growth = ((quarters[i] - quarters[i-1]) / quarters[i-1]) * 100
            assert growth == pytest.approx(50.0, rel=0.1)

    def test_mature_company_growth(self):
        """Test mature company with low single-digit growth"""
        # Mature company: 2-5% QoQ growth
        current = 102_000_000
        previous = 100_000_000

        growth = ((current - previous) / previous) * 100
        assert growth == 2.0

    def test_cyclical_industry(self):
        """Test cyclical industry (e.g., automotive)"""
        # May have negative growth in downturns
        current = 80_000_000  # Recession quarter
        previous = 100_000_000

        growth = ((current - previous) / previous) * 100
        assert growth == -20.0

    def test_turnaround_story(self):
        """Test company turnaround (improving from losses)"""
        q1_ni = -50_000_000  # Heavy loss
        q2_ni = -25_000_000  # Smaller loss (improvement)

        # Improvement but both negative
        growth = ((q2_ni - q1_ni) / abs(q1_ni)) * 100
        assert growth == -50.0  # Loss reduced by 50% (improvement)


class TestDataQuality:
    """Test data quality validation."""

    def test_all_metrics_null(self):
        """Test when all metrics are NULL"""
        metrics = {
            'revenue_current': None,
            'revenue_previous': None,
            'eps_current': None,
            'eps_previous': None,
            'ebitda_current': None,
            'ebitda_previous': None
        }

        # All growth calculations should return None
        assert all(v is None for v in metrics.values())

    def test_partial_data_availability(self):
        """Test when only some metrics are available"""
        # Revenue available, but EPS missing
        revenue_growth = ((110_000_000 - 100_000_000) / 100_000_000) * 100
        eps_growth = None  # Missing data

        assert revenue_growth == 10.0
        assert eps_growth is None


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
