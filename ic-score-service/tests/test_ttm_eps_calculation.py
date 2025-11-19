#!/usr/bin/env python3
"""Unit tests for TTM EPS calculation methodology.

Tests verify the correct handling of:
1. Annual 10-K EPS (when recent)
2. Cumulative quarterly EPS calculation
3. Q4 calculation from annual - Q3
"""

import pytest
from datetime import date, timedelta
import sys
from pathlib import Path

# Add parent directory to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from pipelines.ttm_financials_calculator_v2 import TTMFinancialsCalculator


class TestTTMEPSCalculation:
    """Test suite for TTM EPS calculation with correct methodology."""

    def setup_method(self):
        """Set up test fixtures."""
        self.calculator = TTMFinancialsCalculator()

    def test_recent_annual_10k_used_directly(self):
        """Test that recent annual 10-K EPS is used directly."""
        # Simulating AAPL FY 2025 scenario
        annual_10k = {
            'ticker': 'AAPL',
            'fiscal_year': 2025,
            'period_end_date': date(2025, 9, 27),
            'filing_date': date.today() - timedelta(days=60),  # Filed 60 days ago (recent)
            'eps_basic': 7.60,
            'eps_diluted': 7.5819,
        }

        quarters = [
            {
                'ticker': 'AAPL',
                'fiscal_year': 2025,
                'fiscal_quarter': 3,
                'period_end_date': date(2025, 6, 28),
                'eps_basic': None,
                'eps_diluted': 5.6906,
            }
        ]

        # Should use annual EPS directly since it was filed recently
        eps_basic, eps_diluted = self.calculator.calculate_ttm_eps(annual_10k, quarters)

        assert eps_basic == 7.60
        assert eps_diluted == 7.5819

    def test_old_annual_10k_not_used(self):
        """Test that old annual 10-K is not used."""
        # Simulating old annual report (filed >3 months ago)
        annual_10k = {
            'ticker': 'AAPL',
            'fiscal_year': 2024,
            'period_end_date': date(2024, 9, 28),
            'filing_date': date.today() - timedelta(days=120),  # Filed 120 days ago (old)
            'eps_basic': 6.22,
            'eps_diluted': 6.2008,
        }

        quarters = [
            {
                'ticker': 'AAPL',
                'fiscal_year': 2025,
                'fiscal_quarter': 2,
                'period_end_date': date(2025, 3, 29),
                'eps_basic': 4.11,
                'eps_diluted': 4.0905,
            }
        ]

        # Should NOT use old annual EPS, will need to calculate from quarters
        # (will return None in this case since we don't have enough quarterly data)
        eps_basic, eps_diluted = self.calculator.calculate_ttm_eps(annual_10k, quarters)

        # Should return None because we don't have enough quarterly data
        assert eps_basic is None
        assert eps_diluted is None

    def test_q4_returns_cumulative_directly(self):
        """Test that Q4 cumulative EPS is returned directly (equals full year)."""
        annual_10k = None  # No annual report

        quarters = [
            {
                'ticker': 'AAPL',
                'fiscal_year': 2024,
                'fiscal_quarter': 4,
                'period_end_date': date(2024, 9, 28),
                'eps_basic': 6.22,
                'eps_diluted': 6.2008,
            }
        ]

        # Q4 cumulative = Full year, return directly
        eps_basic, eps_diluted = self.calculator.calculate_ttm_eps(annual_10k, quarters)

        assert eps_basic == 6.22
        assert eps_diluted == 6.2008

    def test_q3_calculates_ttm_from_cumulative(self):
        """Test TTM calculation from Q3 cumulative + prior Q4."""
        # Scenario: We're at Q3 2025, need to calculate TTM
        # TTM = Q3 2025 (YTD) + Q4 2024
        # Q4 2024 = FY 2024 Annual - Q3 2024 (YTD)

        annual_10k = {
            'ticker': 'AAPL',
            'fiscal_year': 2024,
            'period_end_date': date(2024, 9, 28),
            'filing_date': date.today() - timedelta(days=400),  # Old, won't be used directly
            'eps_basic': 6.22,
            'eps_diluted': 6.2008,
        }

        quarters = [
            # Most recent: Q3 2025 (cumulative YTD)
            {
                'ticker': 'AAPL',
                'fiscal_year': 2025,
                'fiscal_quarter': 3,
                'period_end_date': date(2025, 6, 28),
                'eps_basic': 12.21,
                'eps_diluted': 12.1965,  # Q1+Q2+Q3 cumulative
            },
            # Q3 2024 (cumulative YTD from prior year)
            {
                'ticker': 'AAPL',
                'fiscal_year': 2024,
                'fiscal_quarter': 3,
                'period_end_date': date(2024, 6, 29),
                'eps_basic': 9.40,
                'eps_diluted': 9.4110,  # Q1+Q2+Q3 2024 cumulative
            },
        ]

        # Expected calculation:
        # Q4 2024 = Annual 2024 (6.2008) - Q3 2024 (9.4110) = -3.2102 (This seems wrong!)
        # Wait, that can't be right. Let me reconsider...

        # Actually, if Q3 cumulative is GREATER than annual, something is wrong with our data
        # Let me use realistic numbers based on actual AAPL data:
        # Q3 2025 cumulative: 5.6906 (only 3 quarters)
        # Prior annual 2024: 6.2008 (full year)
        # Prior Q3 2024: 5.1898 (only 3 quarters of 2024)

        # Recalculate with correct data
        quarters = [
            {
                'ticker': 'AAPL',
                'fiscal_year': 2025,
                'fiscal_quarter': 3,
                'period_end_date': date(2025, 6, 28),
                'eps_basic': None,
                'eps_diluted': 5.6906,  # Q1+Q2+Q3 2025 cumulative (YTD)
            },
            {
                'ticker': 'AAPL',
                'fiscal_year': 2024,
                'fiscal_quarter': 3,
                'period_end_date': date(2024, 6, 29),
                'eps_basic': None,
                'eps_diluted': 5.1898,  # Q1+Q2+Q3 2024 cumulative (YTD)
            },
        ]

        # Expected calculation:
        # Q4 2024 = Annual 2024 (6.2008) - Q3 2024 cumulative (5.1898) = 1.0110
        # TTM = Q3 2025 cumulative (5.6906) + Q4 2024 (1.0110) = 6.7016

        eps_basic, eps_diluted = self.calculator.calculate_ttm_eps(annual_10k, quarters)

        # Expected: 5.6906 + (6.2008 - 5.1898) = 5.6906 + 1.0110 = 6.7016
        assert eps_basic is None  # We don't have basic data
        assert eps_diluted is not None
        assert abs(eps_diluted - 6.7016) < 0.01  # Allow small rounding difference

    def test_insufficient_data_returns_none(self):
        """Test that insufficient data returns None."""
        annual_10k = None
        quarters = []

        eps_basic, eps_diluted = self.calculator.calculate_ttm_eps(annual_10k, quarters)

        assert eps_basic is None
        assert eps_diluted is None

    def test_apple_fy2025_scenario(self):
        """Test the actual AAPL FY 2025 scenario from production.

        Based on real data:
        - FY 2025 10-K: EPS = $7.5819 (filed ~2 months ago)
        - Should use this directly
        """
        annual_10k = {
            'ticker': 'AAPL',
            'fiscal_year': 2025,
            'period_end_date': date(2025, 9, 27),
            'filing_date': date.today() - timedelta(days=60),
            'eps_basic': 7.60,
            'eps_diluted': 7.5819,
        }

        quarters = [
            {
                'ticker': 'AAPL',
                'fiscal_year': 2025,
                'fiscal_quarter': 3,
                'period_end_date': date(2025, 6, 28),
                'eps_basic': None,
                'eps_diluted': 5.6906,
            }
        ]

        eps_basic, eps_diluted = self.calculator.calculate_ttm_eps(annual_10k, quarters)

        # Should return annual EPS directly
        assert eps_basic == 7.60
        assert eps_diluted == 7.5819

        # Verify this gives correct P/E
        stock_price = 267.44
        pe_ratio = stock_price / eps_diluted

        # P/E should be around 35.3
        assert abs(pe_ratio - 35.28) < 0.5


if __name__ == '__main__':
    pytest.main([__file__, '-v'])
