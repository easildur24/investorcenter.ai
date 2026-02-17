"""Tests for TTMFinancialsCalculator.calculate_ttm_eps().

Tests the pure method calculate_ttm_eps which computes Trailing Twelve
Months EPS from annual 10-K and cumulative quarterly 10-Q data.

Methodology under test:
  1. If annual 10-K filed within 90 days: return its EPS directly.
  2. If most recent quarter is Q4: return Q4 EPS directly (cumulative = full year).
  3. For Q1-Q3: TTM = Current Q (YTD cumulative) + Prior Q4,
     where Prior Q4 = Prior Year Annual EPS - Prior Year Same Q EPS (YTD).
  4. Fallback: return (None, None).
"""

from datetime import date, timedelta
from unittest.mock import patch

import pytest

# The class __init__ calls get_database(), so we must mock it at import time.
with patch("pipelines.ttm_financials_calculator.get_database"):
    from pipelines.ttm_financials_calculator import TTMFinancialsCalculator


def _make_annual(
    ticker="TEST",
    fiscal_year=2024,
    period_end_date=None,
    filing_date=None,
    eps_basic=5.00,
    eps_diluted=4.90,
):
    """Helper to build an annual 10-K dict."""
    return {
        "ticker": ticker,
        "fiscal_year": fiscal_year,
        "period_end_date": period_end_date or date(fiscal_year, 12, 31),
        "filing_date": filing_date,
        "eps_basic": eps_basic,
        "eps_diluted": eps_diluted,
    }


def _make_quarter(
    ticker="TEST",
    fiscal_year=2025,
    fiscal_quarter=3,
    period_end_date=None,
    eps_basic=3.50,
    eps_diluted=3.40,
):
    """Helper to build a quarterly 10-Q dict."""
    # Default period_end_date based on quarter
    if period_end_date is None:
        month = fiscal_quarter * 3
        period_end_date = date(fiscal_year, month, 28)
    return {
        "ticker": ticker,
        "fiscal_year": fiscal_year,
        "fiscal_quarter": fiscal_quarter,
        "period_end_date": period_end_date,
        "eps_basic": eps_basic,
        "eps_diluted": eps_diluted,
    }


# =====================================================================
# TestTtmEpsRecentAnnual -- annual 10-K recency logic
# =====================================================================


class TestTtmEpsRecentAnnual:
    """Tests for Strategy 1: use recent annual 10-K if filed <= 90 days."""

    def setup_method(self):
        with patch(
            "pipelines.ttm_financials_calculator.get_database"
        ):
            self.calc = TTMFinancialsCalculator()

    def test_recent_10k_used_directly(self):
        """10-K filed 60 days ago should return annual EPS directly."""
        filing = date.today() - timedelta(days=60)
        annual = _make_annual(
            filing_date=filing,
            eps_basic=7.60,
            eps_diluted=7.58,
        )
        result = self.calc.calculate_ttm_eps(annual, [])
        assert result == (7.60, 7.58)

    def test_boundary_90_days(self):
        """10-K filed exactly 90 days ago should still be used."""
        filing = date.today() - timedelta(days=90)
        annual = _make_annual(
            filing_date=filing,
            eps_basic=4.00,
            eps_diluted=3.95,
        )
        result = self.calc.calculate_ttm_eps(annual, [])
        assert result == (4.00, 3.95)

    def test_91_days_falls_through(self):
        """10-K filed 91 days ago should NOT be used directly."""
        filing = date.today() - timedelta(days=91)
        annual = _make_annual(
            fiscal_year=2024,
            filing_date=filing,
            eps_basic=4.00,
            eps_diluted=3.95,
        )
        # No quarters provided, so falls through to (None, None)
        result = self.calc.calculate_ttm_eps(annual, [])
        assert result == (None, None)

    def test_no_filing_date(self):
        """Annual with filing_date=None should fall through."""
        annual = _make_annual(
            filing_date=None,
            eps_basic=4.00,
            eps_diluted=3.95,
        )
        # No quarters, falls through
        result = self.calc.calculate_ttm_eps(annual, [])
        assert result == (None, None)


# =====================================================================
# TestTtmEpsFromQuarters -- quarterly cumulative calculation logic
# =====================================================================


class TestTtmEpsFromQuarters:
    """Tests for Strategy 2: TTM EPS from cumulative quarterly data."""

    def setup_method(self):
        with patch(
            "pipelines.ttm_financials_calculator.get_database"
        ):
            self.calc = TTMFinancialsCalculator()

    def test_q3_cumulative_realistic_aapl(self):
        """Q3 cumulative calc with realistic AAPL-like numbers.

        TTM = Q3_2025_cumulative + (Annual_2024 - Q3_2024_cumulative)
            = 5.6906 + (6.2008 - 5.1898)
            = 5.6906 + 1.0110
            = 6.7016
        """
        annual = _make_annual(
            ticker="AAPL",
            fiscal_year=2024,
            filing_date=date.today() - timedelta(days=120),
            eps_basic=6.2008,
            eps_diluted=6.2008,
        )
        quarters = [
            _make_quarter(
                ticker="AAPL",
                fiscal_year=2025,
                fiscal_quarter=3,
                period_end_date=date(2025, 6, 28),
                eps_basic=5.6906,
                eps_diluted=5.6906,
            ),
            _make_quarter(
                ticker="AAPL",
                fiscal_year=2024,
                fiscal_quarter=3,
                period_end_date=date(2024, 6, 29),
                eps_basic=5.1898,
                eps_diluted=5.1898,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        assert basic == pytest.approx(6.7016, abs=0.0001)
        assert diluted == pytest.approx(6.7016, abs=0.0001)

    def test_q2_cumulative_calc(self):
        """Q2 cumulative calculation.

        TTM = Q2_2025_cumulative + (Annual_2024 - Q2_2024_cumulative)
            = 3.20 + (10.00 - 4.80)
            = 3.20 + 5.20
            = 8.40
        """
        annual = _make_annual(
            fiscal_year=2024,
            filing_date=date.today() - timedelta(days=200),
            eps_basic=10.00,
            eps_diluted=9.80,
        )
        quarters = [
            _make_quarter(
                fiscal_year=2025,
                fiscal_quarter=2,
                eps_basic=3.20,
                eps_diluted=3.10,
            ),
            _make_quarter(
                fiscal_year=2024,
                fiscal_quarter=2,
                eps_basic=4.80,
                eps_diluted=4.70,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        assert basic == pytest.approx(8.40, abs=0.0001)
        assert diluted == pytest.approx(8.20, abs=0.0001)

    def test_q1_cumulative_calc(self):
        """Q1 cumulative calculation.

        TTM = Q1_2025 + (Annual_2024 - Q1_2024)
            = 1.50 + (6.00 - 1.20)
            = 1.50 + 4.80
            = 6.30
        """
        annual = _make_annual(
            fiscal_year=2024,
            filing_date=date.today() - timedelta(days=300),
            eps_basic=6.00,
            eps_diluted=5.90,
        )
        quarters = [
            _make_quarter(
                fiscal_year=2025,
                fiscal_quarter=1,
                eps_basic=1.50,
                eps_diluted=1.45,
            ),
            _make_quarter(
                fiscal_year=2024,
                fiscal_quarter=1,
                eps_basic=1.20,
                eps_diluted=1.15,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        assert basic == pytest.approx(6.30, abs=0.0001)
        assert diluted == pytest.approx(6.20, abs=0.0001)

    def test_q4_returns_directly(self):
        """Most recent quarter is Q4 -- return Q4 EPS directly."""
        annual = _make_annual(
            fiscal_year=2024,
            filing_date=date.today() - timedelta(days=200),
            eps_basic=10.00,
            eps_diluted=9.80,
        )
        quarters = [
            _make_quarter(
                fiscal_year=2025,
                fiscal_quarter=4,
                eps_basic=12.00,
                eps_diluted=11.80,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        assert basic == 12.00
        assert diluted == 11.80

    def test_missing_prior_year_quarter(self):
        """No matching prior year same-quarter returns None."""
        annual = _make_annual(
            fiscal_year=2024,
            filing_date=date.today() - timedelta(days=200),
            eps_basic=10.00,
            eps_diluted=9.80,
        )
        quarters = [
            _make_quarter(
                fiscal_year=2025,
                fiscal_quarter=3,
                eps_basic=5.00,
                eps_diluted=4.90,
            ),
            # Prior year Q2 instead of Q3 -- no match
            _make_quarter(
                fiscal_year=2024,
                fiscal_quarter=2,
                eps_basic=3.00,
                eps_diluted=2.90,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        assert basic is None
        assert diluted is None

    def test_missing_prior_annual(self):
        """No annual 10-K at all returns None for Q1-Q3."""
        quarters = [
            _make_quarter(
                fiscal_year=2025,
                fiscal_quarter=2,
                eps_basic=3.00,
                eps_diluted=2.90,
            ),
            _make_quarter(
                fiscal_year=2024,
                fiscal_quarter=2,
                eps_basic=2.50,
                eps_diluted=2.40,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(None, quarters)
        assert basic is None
        assert diluted is None

    def test_fiscal_year_two_years_prior(self):
        """Annual from 2 years ago (not prior year) returns None.

        The method checks annual_10k['fiscal_year'] == most_recent_q['fiscal_year'] - 1.
        If the annual is from 2 years prior, the check fails.
        """
        annual = _make_annual(
            fiscal_year=2023,  # Two years before current Q
            filing_date=date.today() - timedelta(days=400),
            eps_basic=8.00,
            eps_diluted=7.90,
        )
        quarters = [
            _make_quarter(
                fiscal_year=2025,
                fiscal_quarter=3,
                eps_basic=5.00,
                eps_diluted=4.90,
            ),
            _make_quarter(
                fiscal_year=2024,
                fiscal_quarter=3,
                eps_basic=4.00,
                eps_diluted=3.90,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        assert basic is None
        assert diluted is None

    def test_both_basic_and_diluted(self):
        """Basic and diluted EPS are calculated independently.

        Basic: 2.00 + (8.00 - 1.80) = 8.20
        Diluted: 1.90 + (7.50 - 1.70) = 7.70
        """
        annual = _make_annual(
            fiscal_year=2024,
            filing_date=date.today() - timedelta(days=200),
            eps_basic=8.00,
            eps_diluted=7.50,
        )
        quarters = [
            _make_quarter(
                fiscal_year=2025,
                fiscal_quarter=1,
                eps_basic=2.00,
                eps_diluted=1.90,
            ),
            _make_quarter(
                fiscal_year=2024,
                fiscal_quarter=1,
                eps_basic=1.80,
                eps_diluted=1.70,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        assert basic == pytest.approx(8.20, abs=0.0001)
        assert diluted == pytest.approx(7.70, abs=0.0001)
        assert basic != diluted


# =====================================================================
# TestTtmEpsEdgeCases -- boundary and degenerate inputs
# =====================================================================


class TestTtmEpsEdgeCases:
    """Edge cases: empty data, negative EPS, zeros, None fields."""

    def setup_method(self):
        with patch(
            "pipelines.ttm_financials_calculator.get_database"
        ):
            self.calc = TTMFinancialsCalculator()

    def test_empty_quarters(self):
        """Empty quarters list returns (None, None)."""
        annual = _make_annual(
            filing_date=date.today() - timedelta(days=200),
        )
        result = self.calc.calculate_ttm_eps(annual, [])
        assert result == (None, None)

    def test_none_annual_no_quarters(self):
        """Both annual=None and quarters=[] returns (None, None)."""
        result = self.calc.calculate_ttm_eps(None, [])
        assert result == (None, None)

    def test_negative_eps(self):
        """Company with losses should produce correct negative TTM.

        TTM = Q2_2025 + (Annual_2024 - Q2_2024)
            = -1.50 + (-4.00 - (-1.20))
            = -1.50 + (-2.80)
            = -4.30
        """
        annual = _make_annual(
            fiscal_year=2024,
            filing_date=date.today() - timedelta(days=200),
            eps_basic=-4.00,
            eps_diluted=-4.10,
        )
        quarters = [
            _make_quarter(
                fiscal_year=2025,
                fiscal_quarter=2,
                eps_basic=-1.50,
                eps_diluted=-1.60,
            ),
            _make_quarter(
                fiscal_year=2024,
                fiscal_quarter=2,
                eps_basic=-1.20,
                eps_diluted=-1.30,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        assert basic == pytest.approx(-4.30, abs=0.0001)
        assert diluted == pytest.approx(-4.40, abs=0.0001)

    def test_zero_eps(self):
        """EPS = 0.0 throughout should return 0.0.

        TTM = 0.0 + (0.0 - 0.0) = 0.0
        """
        annual = _make_annual(
            fiscal_year=2024,
            filing_date=date.today() - timedelta(days=200),
            eps_basic=0.0,
            eps_diluted=0.0,
        )
        quarters = [
            _make_quarter(
                fiscal_year=2025,
                fiscal_quarter=3,
                eps_basic=0.0,
                eps_diluted=0.0,
            ),
            _make_quarter(
                fiscal_year=2024,
                fiscal_quarter=3,
                eps_basic=0.0,
                eps_diluted=0.0,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        assert basic == 0.0
        assert diluted == 0.0

    def test_none_basic_with_diluted(self):
        """basic=None with diluted present: basic=None, diluted calculated.

        Diluted TTM = 2.50 + (9.00 - 2.20) = 9.30
        Basic is None throughout, so result basic is None.
        """
        annual = _make_annual(
            fiscal_year=2024,
            filing_date=date.today() - timedelta(days=200),
            eps_basic=None,
            eps_diluted=9.00,
        )
        quarters = [
            _make_quarter(
                fiscal_year=2025,
                fiscal_quarter=1,
                eps_basic=None,
                eps_diluted=2.50,
            ),
            _make_quarter(
                fiscal_year=2024,
                fiscal_quarter=1,
                eps_basic=None,
                eps_diluted=2.20,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        assert basic is None
        assert diluted == pytest.approx(9.30, abs=0.0001)

    def test_single_quarter_no_match(self):
        """Only 1 quarter with no prior year data returns None.

        The single Q2 needs a prior year Q2 and annual, but neither
        exists in the quarters list.
        """
        annual = _make_annual(
            fiscal_year=2024,
            filing_date=date.today() - timedelta(days=200),
            eps_basic=10.00,
            eps_diluted=9.80,
        )
        quarters = [
            _make_quarter(
                fiscal_year=2025,
                fiscal_quarter=2,
                eps_basic=3.00,
                eps_diluted=2.90,
            ),
            # No prior year Q2 in list
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        assert basic is None
        assert diluted is None


# =====================================================================
# TestTtmEpsRealWorld -- scenarios modeled on real companies
# =====================================================================


class TestTtmEpsRealWorld:
    """Real-world-inspired scenarios with actual company data patterns."""

    def setup_method(self):
        with patch(
            "pipelines.ttm_financials_calculator.get_database"
        ):
            self.calc = TTMFinancialsCalculator()

    def test_apple_fy2025_recent_annual(self):
        """Real AAPL data: recent 10-K within 90 days returns annual EPS.

        Apple FY ends in September. If the 10-K is filed Nov 1 and
        today is within 90 days of that, annual EPS is used directly.
        """
        # Simulate "today" being within 90 days of Nov 1 filing
        filing_date = date.today() - timedelta(days=30)
        annual = _make_annual(
            ticker="AAPL",
            fiscal_year=2025,
            period_end_date=date(2025, 9, 27),
            filing_date=filing_date,
            eps_basic=7.60,
            eps_diluted=7.5819,
        )
        quarters = [
            _make_quarter(
                ticker="AAPL",
                fiscal_year=2025,
                fiscal_quarter=3,
                period_end_date=date(2025, 6, 28),
                eps_basic=5.69,
                eps_diluted=5.6906,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        # Annual is recent, so used directly
        assert basic == 7.60
        assert diluted == 7.5819

    def test_microsoft_non_calendar_fy(self):
        """MSFT FY ends June 30. Q1 starts July.

        Most recent quarter: Q2 FY2025 (period ending Dec 2024)
        Prior annual: FY2024 (period ending Jun 2024)
        Prior Q2 FY2024 (period ending Dec 2023)

        TTM = Q2_FY2025_cumulative + (Annual_FY2024 - Q2_FY2024_cumulative)
            = 6.40 + (11.80 - 5.90) = 6.40 + 5.90 = 12.30
        """
        annual = _make_annual(
            ticker="MSFT",
            fiscal_year=2024,
            period_end_date=date(2024, 6, 30),
            filing_date=date.today() - timedelta(days=200),
            eps_basic=11.80,
            eps_diluted=11.70,
        )
        quarters = [
            _make_quarter(
                ticker="MSFT",
                fiscal_year=2025,
                fiscal_quarter=2,
                period_end_date=date(2024, 12, 31),
                eps_basic=6.40,
                eps_diluted=6.30,
            ),
            _make_quarter(
                ticker="MSFT",
                fiscal_year=2024,
                fiscal_quarter=2,
                period_end_date=date(2023, 12, 31),
                eps_basic=5.90,
                eps_diluted=5.80,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        assert basic == pytest.approx(12.30, abs=0.0001)
        assert diluted == pytest.approx(12.20, abs=0.0001)

    def test_stock_split_scenario(self):
        """Different EPS magnitudes that might indicate a stock split.

        Even with different magnitudes, the math should work correctly.
        Post-split current Q1 EPS = 0.50
        Pre-split prior annual EPS = 20.00
        Pre-split prior Q1 EPS = 4.00

        TTM = 0.50 + (20.00 - 4.00) = 16.50
        (The result is mathematically valid even if economically odd --
         the sanity check for splits is in calculate_ttm_metrics, not here.)
        """
        annual = _make_annual(
            fiscal_year=2024,
            filing_date=date.today() - timedelta(days=200),
            eps_basic=20.00,
            eps_diluted=19.50,
        )
        quarters = [
            _make_quarter(
                fiscal_year=2025,
                fiscal_quarter=1,
                eps_basic=0.50,
                eps_diluted=0.48,
            ),
            _make_quarter(
                fiscal_year=2024,
                fiscal_quarter=1,
                eps_basic=4.00,
                eps_diluted=3.90,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        # Pure math: 0.50 + (20.00 - 4.00) = 16.50
        assert basic == pytest.approx(16.50, abs=0.0001)
        # 0.48 + (19.50 - 3.90) = 16.08
        assert diluted == pytest.approx(16.08, abs=0.0001)

    def test_pre_revenue_company(self):
        """Pre-revenue company: all EPS = 0 or None.

        When annual and all quarters have eps_basic=None and
        eps_diluted=None, the method should return (None, None).
        """
        annual = _make_annual(
            ticker="BIOTECH",
            fiscal_year=2024,
            filing_date=date.today() - timedelta(days=200),
            eps_basic=None,
            eps_diluted=None,
        )
        quarters = [
            _make_quarter(
                ticker="BIOTECH",
                fiscal_year=2025,
                fiscal_quarter=2,
                eps_basic=None,
                eps_diluted=None,
            ),
            _make_quarter(
                ticker="BIOTECH",
                fiscal_year=2024,
                fiscal_quarter=2,
                eps_basic=None,
                eps_diluted=None,
            ),
        ]
        basic, diluted = self.calc.calculate_ttm_eps(annual, quarters)
        assert basic is None
        assert diluted is None
