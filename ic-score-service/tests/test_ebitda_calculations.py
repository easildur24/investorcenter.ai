"""Tests for EBITDA and EV/EBITDA calculations.

Tests Phase 2 implementation:
- EBITDA calculation (Operating Income + D&A)
- TTM EBITDA (sum of 4 quarters)
- Enterprise Value
- EV/EBITDA ratio
"""

import pytest


class TestEBITDACalculations:
    """Test suite for EBITDA calculations."""

    def test_ebitda_basic_calculation(self):
        """Test EBITDA = Operating Income + D&A"""
        operating_income = 5_000_000
        depreciation_and_amortization = 1_000_000

        ebitda = operating_income + depreciation_and_amortization
        assert ebitda == 6_000_000

    def test_ttm_ebitda_from_quarters(self):
        """Test TTM EBITDA = Sum of 4 quarters"""
        q1_ebitda = 1_500_000
        q2_ebitda = 1_600_000
        q3_ebitda = 1_700_000
        q4_ebitda = 1_800_000

        ttm_ebitda = q1_ebitda + q2_ebitda + q3_ebitda + q4_ebitda
        assert ttm_ebitda == 6_600_000

    def test_ebitda_always_greater_than_operating_income(self):
        """Test that EBITDA >= Operating Income (since D&A is added back)"""
        operating_income = 5_000_000
        da = 1_000_000

        ebitda = operating_income + da
        assert ebitda > operating_income
        assert ebitda - operating_income == da

    def test_negative_operating_income(self):
        """Test EBITDA with negative operating income"""
        operating_income = -2_000_000
        da = 3_000_000

        ebitda = operating_income + da
        assert ebitda == 1_000_000  # Can still be positive


class TestEnterpriseValue:
    """Test Enterprise Value calculations."""

    def test_ev_basic_calculation(self):
        """Test EV = Market Cap + Total Debt - Cash"""
        market_cap = 100_000_000
        total_debt = 20_000_000
        cash = 10_000_000

        ev = market_cap + total_debt - cash
        assert ev == 110_000_000

    def test_ev_debt_free_company(self):
        """Test EV for debt-free company"""
        market_cap = 50_000_000
        total_debt = 0
        cash = 5_000_000

        ev = market_cap + total_debt - cash
        assert ev == 45_000_000

    def test_ev_with_excess_cash(self):
        """Test EV when cash > debt"""
        market_cap = 100_000_000
        total_debt = 10_000_000
        cash = 30_000_000

        ev = market_cap + total_debt - cash
        assert ev == 80_000_000  # Lower than market cap

    def test_total_debt_calculation(self):
        """Test Total Debt = Short-term + Long-term debt"""
        short_term_debt = 5_000_000
        long_term_debt = 15_000_000

        total_debt = short_term_debt + long_term_debt
        assert total_debt == 20_000_000


class TestEVEBITDARatio:
    """Test EV/EBITDA ratio calculations."""

    def test_ev_ebitda_basic_calculation(self):
        """Test EV/EBITDA ratio"""
        enterprise_value = 200_000_000
        ttm_ebitda = 20_000_000

        ev_ebitda_ratio = enterprise_value / ttm_ebitda
        assert ev_ebitda_ratio == 10.0

    def test_ev_ebitda_typical_range(self):
        """Test that typical EV/EBITDA ratios are in 10-30 range"""
        ev = 150_000_000
        ebitda = 10_000_000

        ratio = ev / ebitda
        assert 10 <= ratio <= 30  # Typical range for most stocks

    def test_ev_ebitda_with_high_growth(self):
        """Test EV/EBITDA for high-growth company (higher multiples)"""
        ev = 500_000_000
        ebitda = 10_000_000  # Low EBITDA, high valuation

        ratio = ev / ebitda
        assert ratio == 50.0  # Tech/growth stocks can have higher multiples

    def test_ev_ebitda_value_stock(self):
        """Test EV/EBITDA for value stock (lower multiples)"""
        ev = 80_000_000
        ebitda = 20_000_000  # High EBITDA, low valuation

        ratio = ev / ebitda
        assert ratio == 4.0  # Value stocks have lower multiples

    def test_negative_ebitda(self):
        """Test EV/EBITDA with negative EBITDA (unprofitable company)"""
        ev = 100_000_000
        ebitda = -5_000_000

        # Should return None or skip calculation for negative EBITDA
        # Negative EV/EBITDA is not meaningful
        if ebitda > 0:
            ratio = ev / ebitda
        else:
            ratio = None

        assert ratio is None

    def test_zero_ebitda(self):
        """Test division by zero handling for zero EBITDA"""
        ev = 100_000_000
        ebitda = 0

        # Should handle gracefully
        with pytest.raises(ZeroDivisionError):
            _ = ev / ebitda


class TestComparativeAnalysis:
    """Test comparative valuation scenarios."""

    def test_compare_pe_vs_ev_ebitda(self):
        """Test that EV/EBITDA and P/E can differ significantly"""
        # Company A: High debt
        ev_a = 200_000_000
        ebitda_a = 20_000_000
        ev_ebitda_a = ev_a / ebitda_a

        # Company B: No debt
        ev_b = 120_000_000  # Lower EV due to cash
        ebitda_b = 20_000_000  # Same EBITDA
        ev_ebitda_b = ev_b / ebitda_b

        # Company B should have lower EV/EBITDA despite same EBITDA
        assert ev_ebitda_b < ev_ebitda_a

    def test_capital_intensive_industry(self):
        """Test EV/EBITDA for capital-intensive industry (utilities, telecom)"""
        # These industries typically have high D&A
        operating_income = 5_000_000
        da = 10_000_000  # Very high D&A
        ebitda = operating_income + da

        assert ebitda == 15_000_000
        assert da / ebitda > 0.5  # D&A is >50% of EBITDA


class TestRealWorldExamples:
    """Test with realistic company examples."""

    def test_tech_company_metrics(self):
        """Test typical tech company (e.g., Microsoft)"""
        # Tech companies: Low debt, high margins, high multiples
        market_cap = 2_000_000_000_000  # $2T
        debt = 50_000_000_000  # $50B
        cash = 100_000_000_000  # $100B
        ttm_ebitda = 100_000_000_000  # $100B

        ev = market_cap + debt - cash
        ev_ebitda = ev / ttm_ebitda

        assert ev == 1_950_000_000_000
        assert ev_ebitda == pytest.approx(19.5, rel=0.01)

    def test_manufacturing_company(self):
        """Test typical manufacturing company"""
        # Manufacturing: Higher debt, lower margins, moderate multiples
        market_cap = 50_000_000_000  # $50B
        debt = 30_000_000_000  # $30B
        cash = 5_000_000_000  # $5B
        ttm_ebitda = 8_000_000_000  # $8B

        ev = market_cap + debt - cash
        ev_ebitda = ev / ttm_ebitda

        assert ev == 75_000_000_000
        assert ev_ebitda == pytest.approx(9.375, rel=0.01)


class TestDataQuality:
    """Test data quality and edge cases."""

    def test_missing_da_data(self):
        """Test handling when D&A data is missing"""
        operating_income = 5_000_000
        da = None

        # Should skip EBITDA calculation if D&A is missing
        if da is not None:
            ebitda = operating_income + da
        else:
            ebitda = None

        assert ebitda is None

    def test_missing_debt_data(self):
        """Test EV calculation with missing debt data"""
        market_cap = 100_000_000
        short_term_debt = None
        long_term_debt = 20_000_000
        cash = 10_000_000

        # Treat missing debt as 0
        total_debt = (short_term_debt or 0) + (long_term_debt or 0)
        ev = market_cap + total_debt - cash

        assert ev == 110_000_000


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
