"""Unit tests for SEC Client utility functions.

Tests cover the four core internal methods of SECClient:
- _parse_fiscal_period: fiscal period string parsing
- _correct_fiscal_years: fiscal year correction for non-calendar FYE
- _correct_shares_outstanding: heuristic correction for shares in millions
- _calculate_metrics: derived financial metric calculations
"""

import pytest
from pipelines.utils.sec_client import SECClient


class TestParseFiscalPeriod:
    """Tests for SECClient._parse_fiscal_period."""

    def setup_method(self):
        self.client = SECClient()

    def test_q1(self):
        """Q1 parses to quarter 1."""
        assert self.client._parse_fiscal_period("Q1") == 1

    def test_q2(self):
        """Q2 parses to quarter 2."""
        assert self.client._parse_fiscal_period("Q2") == 2

    def test_q3(self):
        """Q3 parses to quarter 3."""
        assert self.client._parse_fiscal_period("Q3") == 3

    def test_q4(self):
        """Q4 parses to quarter 4."""
        assert self.client._parse_fiscal_period("Q4") == 4

    def test_fy(self):
        """FY (annual) parses to None."""
        assert self.client._parse_fiscal_period("FY") is None

    def test_q1i_variant(self):
        """Q1I interim variant parses to quarter 1."""
        assert self.client._parse_fiscal_period("Q1I") == 1

    def test_none_input(self):
        """None input returns None."""
        assert self.client._parse_fiscal_period(None) is None


class TestCorrectFiscalYears:
    """Tests for SECClient._correct_fiscal_years.

    This method detects the fiscal year-end (FYE) month from annual filings
    and corrects quarterly fiscal_year values when the company has a
    non-December fiscal year end (e.g., Apple's September FYE).
    """

    def setup_method(self):
        self.client = SECClient()

    def test_calendar_year_no_correction(self):
        """Companies with December FYE should have no fiscal year corrections."""
        financials = [
            # Annual filing ending December (FYE = December)
            {
                "period_end_date": "2024-12-31",
                "fiscal_year": 2024,
                "fiscal_quarter": None,
            },
            # Q1 ending March
            {
                "period_end_date": "2024-03-31",
                "fiscal_year": 2024,
                "fiscal_quarter": 1,
            },
            # Q2 ending June
            {
                "period_end_date": "2024-06-30",
                "fiscal_year": 2024,
                "fiscal_quarter": 2,
            },
            # Q3 ending September
            {
                "period_end_date": "2024-09-30",
                "fiscal_year": 2024,
                "fiscal_quarter": 3,
            },
        ]

        self.client._correct_fiscal_years(financials)

        # All fiscal years should remain unchanged
        assert financials[0]["fiscal_year"] == 2024
        assert financials[1]["fiscal_year"] == 2024
        assert financials[2]["fiscal_year"] == 2024
        assert financials[3]["fiscal_year"] == 2024

    def test_non_calendar_fy_correction(self):
        """AAPL-like company with September FYE: October period belongs to next FY."""
        financials = [
            # Annual filing ending September (FYE = September)
            {
                "period_end_date": "2024-09-28",
                "fiscal_year": 2024,
                "fiscal_quarter": None,
            },
            # Quarterly filing ending in October -- period_end_month (10) > fye_month (9)
            # So correct fiscal year = 2024 + 1 = 2025
            {
                "period_end_date": "2024-10-31",
                "fiscal_year": 2024,
                "fiscal_quarter": 1,
            },
        ]

        self.client._correct_fiscal_years(financials)

        # Annual record should stay the same
        assert financials[0]["fiscal_year"] == 2024
        # October quarter should be corrected to FY 2025
        assert financials[1]["fiscal_year"] == 2025

    def test_no_annual_filings_skips(self):
        """When there are no annual filings, no corrections should be applied."""
        financials = [
            {
                "period_end_date": "2024-10-31",
                "fiscal_year": 2024,
                "fiscal_quarter": 1,
            },
            {
                "period_end_date": "2025-01-31",
                "fiscal_year": 2025,
                "fiscal_quarter": 2,
            },
        ]

        original_fy_values = [f["fiscal_year"] for f in financials]
        self.client._correct_fiscal_years(financials)
        new_fy_values = [f["fiscal_year"] for f in financials]

        assert original_fy_values == new_fy_values

    def test_multiple_quarters_corrected(self):
        """All quarterly records for a non-calendar FYE are corrected consistently."""
        financials = [
            # Annual filing ending June (FYE = June)
            {
                "period_end_date": "2024-06-30",
                "fiscal_year": 2024,
                "fiscal_quarter": None,
            },
            # Q1: period ends September -- month 9 > fye_month 6, so FY = 2024+1 = 2025
            {
                "period_end_date": "2024-09-30",
                "fiscal_year": 2024,
                "fiscal_quarter": 1,
            },
            # Q2: period ends December -- month 12 > fye_month 6, so FY = 2024+1 = 2025
            {
                "period_end_date": "2024-12-31",
                "fiscal_year": 2024,
                "fiscal_quarter": 2,
            },
            # Q3: period ends March -- month 3 <= fye_month 6, so FY = 2025
            {
                "period_end_date": "2025-03-31",
                "fiscal_year": 2025,
                "fiscal_quarter": 3,
            },
        ]

        self.client._correct_fiscal_years(financials)

        # Q1 and Q2 with period_end_month > 6 should become FY 2025
        assert financials[1]["fiscal_year"] == 2025
        assert financials[2]["fiscal_year"] == 2025
        # Q3 with period_end_month (3) <= 6 stays FY 2025
        assert financials[3]["fiscal_year"] == 2025

    def test_annual_records_not_modified(self):
        """Annual records (fiscal_quarter=None) should not have fiscal_year changed."""
        financials = [
            # Annual filing with September FYE
            {
                "period_end_date": "2024-09-28",
                "fiscal_year": 2024,
                "fiscal_quarter": None,
            },
            # Another annual filing
            {
                "period_end_date": "2023-09-30",
                "fiscal_year": 2023,
                "fiscal_quarter": None,
            },
        ]

        self.client._correct_fiscal_years(financials)

        assert financials[0]["fiscal_year"] == 2024
        assert financials[1]["fiscal_year"] == 2023


class TestCorrectSharesOutstanding:
    """Tests for SECClient._correct_shares_outstanding.

    Heuristic: if shares_outstanding < 10,000 but revenue > 1B,
    multiply shares by 1,000,000 (company reported in millions).
    """

    def setup_method(self):
        self.client = SECClient()

    def test_shares_in_millions_corrected(self):
        """Shares < 10,000 with revenue > 1B are multiplied by 1,000,000."""
        financials = [
            {
                "shares_outstanding": 750,
                "revenue": 50_000_000_000,  # 50B
            }
        ]

        self.client._correct_shares_outstanding(financials)

        assert financials[0]["shares_outstanding"] == 750_000_000

    def test_normal_shares_not_corrected(self):
        """Shares >= 10,000 are left unchanged even with high revenue."""
        financials = [
            {
                "shares_outstanding": 15_000_000_000,
                "revenue": 50_000_000_000,
            }
        ]

        self.client._correct_shares_outstanding(financials)

        assert financials[0]["shares_outstanding"] == 15_000_000_000

    def test_no_shares_skipped(self):
        """Records with shares_outstanding=None are skipped without error."""
        financials = [
            {
                "shares_outstanding": None,
                "revenue": 50_000_000_000,
            }
        ]

        self.client._correct_shares_outstanding(financials)

        assert financials[0]["shares_outstanding"] is None

    def test_low_revenue_not_corrected(self):
        """Shares < 10,000 with revenue <= 1B are not corrected (small company)."""
        financials = [
            {
                "shares_outstanding": 500,
                "revenue": 500_000_000,  # 500M, below 1B threshold
            }
        ]

        self.client._correct_shares_outstanding(financials)

        assert financials[0]["shares_outstanding"] == 500


class TestCalculateMetrics:
    """Tests for SECClient._calculate_metrics.

    This method calculates derived financial ratios/metrics in-place.
    """

    def setup_method(self):
        self.client = SECClient()

    def _make_financial(self, **kwargs):
        """Helper to create a financial dict with sensible defaults."""
        base = {
            "revenue": 100_000_000_000,       # 100B
            "cost_of_revenue": 60_000_000_000,  # 60B
            "operating_income": 25_000_000_000,  # 25B
            "net_income": 20_000_000_000,        # 20B
            "total_assets": 400_000_000_000,     # 400B
            "total_liabilities": 250_000_000_000,
            "shareholders_equity": 150_000_000_000,
            "short_term_debt": 10_000_000_000,
            "long_term_debt": 90_000_000_000,
            "operating_cash_flow": 30_000_000_000,
            "capex": -8_000_000_000,  # capex is typically negative
        }
        base.update(kwargs)
        return base

    def test_gross_profit(self):
        """Gross profit = revenue - cost_of_revenue when not already present."""
        financial = self._make_financial()
        # Remove pre-existing gross_profit to force calculation
        financial.pop("gross_profit", None)

        self.client._calculate_metrics(financial)

        # 100B - 60B = 40B
        assert financial["gross_profit"] == 40_000_000_000.0

    def test_free_cash_flow(self):
        """Free cash flow = operating_cash_flow - abs(capex)."""
        financial = self._make_financial()

        self.client._calculate_metrics(financial)

        # 30B - abs(-8B) = 22B
        assert financial["free_cash_flow"] == 22_000_000_000.0

    def test_gross_margin(self):
        """Gross margin = gross_profit / revenue * 100."""
        financial = self._make_financial()
        financial.pop("gross_profit", None)

        self.client._calculate_metrics(financial)

        # gross_profit = 40B, revenue = 100B -> 40%
        assert abs(financial["gross_margin"] - 40.0) < 0.01

    def test_operating_margin(self):
        """Operating margin = operating_income / revenue * 100."""
        financial = self._make_financial()

        self.client._calculate_metrics(financial)

        # 25B / 100B * 100 = 25%
        assert abs(financial["operating_margin"] - 25.0) < 0.01

    def test_net_margin(self):
        """Net margin = net_income / revenue * 100."""
        financial = self._make_financial()

        self.client._calculate_metrics(financial)

        # 20B / 100B * 100 = 20%
        assert abs(financial["net_margin"] - 20.0) < 0.01

    def test_debt_to_equity(self):
        """Debt to equity = (short_term_debt + long_term_debt) / shareholders_equity."""
        financial = self._make_financial()

        self.client._calculate_metrics(financial)

        # (10B + 90B) / 150B = 0.6667
        expected = (10_000_000_000 + 90_000_000_000) / 150_000_000_000
        assert abs(financial["debt_to_equity"] - expected) < 0.0001

    def test_roa(self):
        """ROA = net_income / total_assets * 100."""
        financial = self._make_financial()

        self.client._calculate_metrics(financial)

        # 20B / 400B * 100 = 5%
        assert abs(financial["roa"] - 5.0) < 0.01

    def test_roe(self):
        """ROE = net_income / shareholders_equity * 100."""
        financial = self._make_financial()

        self.client._calculate_metrics(financial)

        # 20B / 150B * 100 = 13.333...%
        expected = (20_000_000_000 / 150_000_000_000) * 100
        assert abs(financial["roe"] - expected) < 0.01

    def test_zero_revenue_safe(self):
        """Zero revenue should not cause division errors; margins should not be set."""
        financial = self._make_financial(revenue=0, cost_of_revenue=0)

        self.client._calculate_metrics(financial)

        # No division by zero error should occur
        assert "gross_margin" not in financial
        assert "operating_margin" not in financial
        assert "net_margin" not in financial


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
