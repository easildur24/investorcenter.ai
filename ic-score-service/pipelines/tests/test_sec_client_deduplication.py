"""Unit tests for SEC Client de-duplication logic.

This test ensures that the SEC client properly de-duplicates financial records
from XBRL data, preventing multiple records for the same fiscal quarter.
"""

import pytest
from pipelines.utils.sec_client import SECClient


class TestSECClientDeduplication:
    """Test de-duplication of SEC financial data."""

    def setup_method(self):
        """Set up test fixtures."""
        self.client = SECClient()

    def test_deduplication_same_fiscal_quarter_different_period_ends(self):
        """Test that records with same fiscal_year+quarter but different period_end_dates are de-duplicated."""
        # Mock company facts data simulating AAPL Q3 2025 10-Q filing
        # This filing contains:
        # - Current quarter data (2025-06-28) - the actual Q3
        # - YTD cumulative data (2025-03-29) - included in same 10-Q
        # - Prior year comparison (2024-06-29) - for comparison
        company_facts = {
            "facts": {
                "us-gaap": {
                    "Revenues": {
                        "units": {
                            "USD": [
                                # Actual Q3 2025 data (most recent period_end)
                                {
                                    "form": "10-Q",
                                    "end": "2025-06-28",
                                    "filed": "2025-08-01",
                                    "fy": 2025,
                                    "fp": "Q3",
                                    "val": 313695000000,
                                },
                                # YTD cumulative data (older period_end, but same fiscal quarter)
                                {
                                    "form": "10-Q",
                                    "end": "2025-03-29",
                                    "filed": "2025-08-01",
                                    "fy": 2025,
                                    "fp": "Q3",
                                    "val": 219659000000,
                                },
                                # Prior year Q3 for comparison (much older)
                                {
                                    "form": "10-Q",
                                    "end": "2024-06-29",
                                    "filed": "2025-08-01",
                                    "fy": 2025,
                                    "fp": "Q3",
                                    "val": 296105000000,
                                },
                            ]
                        }
                    },
                    "NetIncomeLoss": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": "2025-06-28",
                                    "filed": "2025-08-01",
                                    "fy": 2025,
                                    "fp": "Q3",
                                    "val": 84544000000,
                                },
                                {
                                    "form": "10-Q",
                                    "end": "2025-03-29",
                                    "filed": "2025-08-01",
                                    "fy": 2025,
                                    "fp": "Q3",
                                    "val": 61110000000,
                                },
                                {
                                    "form": "10-Q",
                                    "end": "2024-06-29",
                                    "filed": "2025-08-01",
                                    "fy": 2025,
                                    "fp": "Q3",
                                    "val": 79000000000,
                                },
                            ]
                        }
                    },
                }
            }
        }

        # Parse the data
        financials = self.client.parse_financial_data(company_facts, num_periods=10)

        # Should return only 1 record for fiscal quarter Q3 2025
        q3_2025_records = [
            f for f in financials
            if f['fiscal_year'] == 2025 and f['fiscal_quarter'] == 3
        ]

        assert len(q3_2025_records) == 1, (
            f"Expected 1 record for FY2025 Q3, got {len(q3_2025_records)}"
        )

        # Should select the record with the most recent period_end_date (2025-06-28)
        record = q3_2025_records[0]
        assert record['period_end_date'] == "2025-06-28", (
            f"Expected period_end_date='2025-06-28', got '{record['period_end_date']}'"
        )

        # Should have the correct data for that period
        assert record['revenue'] == 313695000000, (
            f"Expected revenue=313695000000, got {record['revenue']}"
        )
        assert record['net_income'] == 84544000000, (
            f"Expected net_income=84544000000, got {record['net_income']}"
        )

    def test_prefer_records_with_revenue(self):
        """Test that records with revenue are preferred over those without."""
        company_facts = {
            "facts": {
                "us-gaap": {
                    # Record 1: Has net_income but NO revenue (older period_end)
                    "NetIncomeLoss": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": "2025-06-30",
                                    "filed": "2025-07-24",
                                    "fy": 2025,
                                    "fp": "Q2",
                                    "val": 599000000,
                                },
                                {
                                    "form": "10-Q",
                                    "end": "2025-03-31",
                                    "filed": "2025-07-24",
                                    "fy": 2025,
                                    "fp": "Q2",
                                    "val": -473000000,
                                },
                            ]
                        }
                    },
                    # Record 2: Has BOTH revenue and net_income (more recent period_end)
                    "Revenues": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": "2025-06-30",
                                    "filed": "2025-07-24",
                                    "fy": 2025,
                                    "fp": "Q2",
                                    "val": 26943000000,
                                },
                            ]
                        }
                    },
                }
            }
        }

        financials = self.client.parse_financial_data(company_facts, num_periods=10)

        q2_2025_records = [
            f for f in financials
            if f['fiscal_year'] == 2025 and f['fiscal_quarter'] == 2
        ]

        # Should return only 1 record
        assert len(q2_2025_records) == 1

        # Should prefer the record with revenue (2025-06-30)
        record = q2_2025_records[0]
        assert record['period_end_date'] == "2025-06-30"
        assert record['revenue'] == 26943000000
        assert record['net_income'] == 599000000

    def test_multiple_quarters_each_deduplicated(self):
        """Test that multiple quarters are each de-duplicated independently."""
        company_facts = {
            "facts": {
                "us-gaap": {
                    "Revenues": {
                        "units": {
                            "USD": [
                                # Q3 2025 - 3 duplicates
                                {"form": "10-Q", "end": "2025-09-30", "filed": "2025-10-23", "fy": 2025, "fp": "Q3", "val": 40634000000},
                                {"form": "10-Q", "end": "2025-06-30", "filed": "2025-10-23", "fy": 2025, "fp": "Q3", "val": 26943000000},
                                {"form": "10-Q", "end": "2025-03-31", "filed": "2025-10-23", "fy": 2025, "fp": "Q3", "val": 20000000000},

                                # Q2 2025 - 2 duplicates
                                {"form": "10-Q", "end": "2025-06-30", "filed": "2025-07-24", "fy": 2025, "fp": "Q2", "val": 26943000000},
                                {"form": "10-Q", "end": "2025-03-31", "filed": "2025-07-24", "fy": 2025, "fp": "Q2", "val": 20000000000},

                                # Q1 2025 - 1 record (no duplicates)
                                {"form": "10-Q", "end": "2025-03-31", "filed": "2025-04-24", "fy": 2025, "fp": "Q1", "val": 20000000000},
                            ]
                        }
                    },
                    "NetIncomeLoss": {
                        "units": {
                            "USD": [
                                {"form": "10-Q", "end": "2025-09-30", "filed": "2025-10-23", "fy": 2025, "fp": "Q3", "val": 12000000},
                                {"form": "10-Q", "end": "2025-06-30", "filed": "2025-10-23", "fy": 2025, "fp": "Q3", "val": 599000000},
                                {"form": "10-Q", "end": "2025-03-31", "filed": "2025-10-23", "fy": 2025, "fp": "Q3", "val": -473000000},

                                {"form": "10-Q", "end": "2025-06-30", "filed": "2025-07-24", "fy": 2025, "fp": "Q2", "val": 126000000},
                                {"form": "10-Q", "end": "2025-03-31", "filed": "2025-07-24", "fy": 2025, "fp": "Q2", "val": -473000000},

                                {"form": "10-Q", "end": "2025-03-31", "filed": "2025-04-24", "fy": 2025, "fp": "Q1", "val": -473000000},
                            ]
                        }
                    },
                }
            }
        }

        financials = self.client.parse_financial_data(company_facts, num_periods=10)

        # Count records per quarter
        quarters = {}
        for f in financials:
            key = (f['fiscal_year'], f['fiscal_quarter'])
            quarters[key] = quarters.get(key, 0) + 1

        # Each quarter should have exactly 1 record
        assert quarters[(2025, 3)] == 1, f"Q3 2025 should have 1 record, got {quarters[(2025, 3)]}"
        assert quarters[(2025, 2)] == 1, f"Q2 2025 should have 1 record, got {quarters[(2025, 2)]}"
        assert quarters[(2025, 1)] == 1, f"Q1 2025 should have 1 record, got {quarters[(2025, 1)]}"

        # Check that we selected the most recent period_end for each
        q3_record = next(f for f in financials if f['fiscal_year'] == 2025 and f['fiscal_quarter'] == 3)
        assert q3_record['period_end_date'] == "2025-09-30"
        assert q3_record['revenue'] == 40634000000

        q2_record = next(f for f in financials if f['fiscal_year'] == 2025 and f['fiscal_quarter'] == 2)
        assert q2_record['period_end_date'] == "2025-06-30"
        assert q2_record['revenue'] == 26943000000

        q1_record = next(f for f in financials if f['fiscal_year'] == 2025 and f['fiscal_quarter'] == 1)
        assert q1_record['period_end_date'] == "2025-03-31"
        assert q1_record['revenue'] == 20000000000

    def test_annual_reports_not_duplicated_with_quarters(self):
        """Test that annual reports (FY) are separate from quarterly reports."""
        company_facts = {
            "facts": {
                "us-gaap": {
                    "Revenues": {
                        "units": {
                            "USD": [
                                # Annual report for FY 2024
                                {"form": "10-K", "end": "2024-12-31", "filed": "2025-02-15", "fy": 2024, "fp": "FY", "val": 100000000000},

                                # Q4 2024
                                {"form": "10-Q", "end": "2024-12-31", "filed": "2025-01-30", "fy": 2024, "fp": "Q4", "val": 30000000000},
                            ]
                        }
                    },
                }
            }
        }

        financials = self.client.parse_financial_data(company_facts, num_periods=10)

        # Should have 2 records: 1 annual (None quarter), 1 quarterly (Q4)
        fy_2024_records = [f for f in financials if f['fiscal_year'] == 2024]

        assert len(fy_2024_records) == 2, f"Expected 2 records for FY 2024 (1 annual, 1 Q4), got {len(fy_2024_records)}"

        annual_record = next((f for f in fy_2024_records if f['fiscal_quarter'] is None), None)
        quarterly_record = next((f for f in fy_2024_records if f['fiscal_quarter'] == 4), None)

        assert annual_record is not None, "Should have annual report record"
        assert quarterly_record is not None, "Should have Q4 record"

        assert annual_record['revenue'] == 100000000000
        assert quarterly_record['revenue'] == 30000000000


    def test_banking_revenue_synthesis(self):
        """Test that banking revenue is synthesized from InterestIncomeExpenseNet + NoninterestIncome."""
        company_facts = {
            "facts": {
                "us-gaap": {
                    # No standard "Revenues" tag - this is typical for banks
                    "InterestIncomeExpenseNet": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": "2025-09-30",
                                    "filed": "2025-10-14",
                                    "fy": 2025,
                                    "fp": "Q3",
                                    "val": 23500000000,
                                },
                            ]
                        }
                    },
                    "NoninterestIncome": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": "2025-09-30",
                                    "filed": "2025-10-14",
                                    "fy": 2025,
                                    "fp": "Q3",
                                    "val": 23600000000,
                                },
                            ]
                        }
                    },
                    "NetIncomeLoss": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": "2025-09-30",
                                    "filed": "2025-10-14",
                                    "fy": 2025,
                                    "fp": "Q3",
                                    "val": 14390000000,
                                },
                            ]
                        }
                    },
                    "NoninterestExpense": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": "2025-09-30",
                                    "filed": "2025-10-14",
                                    "fy": 2025,
                                    "fp": "Q3",
                                    "val": 23700000000,
                                },
                            ]
                        }
                    },
                }
            }
        }

        financials = self.client.parse_financial_data(company_facts, num_periods=10)

        q3_records = [
            f for f in financials
            if f['fiscal_year'] == 2025 and f['fiscal_quarter'] == 3
        ]
        assert len(q3_records) == 1

        record = q3_records[0]

        # Revenue should be synthesized: NII + Noninterest Income
        assert record['revenue'] == 23500000000 + 23600000000, (
            f"Expected revenue={23500000000 + 23600000000}, got {record.get('revenue')}"
        )

        # Original fields should be preserved
        assert record['net_interest_income'] == 23500000000
        assert record['noninterest_income'] == 23600000000

        # Net margin should be calculated from synthesized revenue
        expected_margin = (14390000000 / (23500000000 + 23600000000)) * 100
        assert abs(record['net_margin'] - expected_margin) < 0.01

        # Operating expenses should use NoninterestExpense fallback
        assert record['operating_expenses'] == 23700000000

    def test_insurance_revenue_synthesis(self):
        """Test that insurance revenue is synthesized from PremiumsEarnedNet."""
        company_facts = {
            "facts": {
                "us-gaap": {
                    # No standard "Revenues" tag - typical for insurance companies
                    "PremiumsEarnedNet": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": "2025-06-30",
                                    "filed": "2025-08-01",
                                    "fy": 2025,
                                    "fp": "Q2",
                                    "val": 42000000000,
                                },
                            ]
                        }
                    },
                    "NetIncomeLoss": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": "2025-06-30",
                                    "filed": "2025-08-01",
                                    "fy": 2025,
                                    "fp": "Q2",
                                    "val": 6300000000,
                                },
                            ]
                        }
                    },
                }
            }
        }

        financials = self.client.parse_financial_data(company_facts, num_periods=10)

        q2_records = [
            f for f in financials
            if f['fiscal_year'] == 2025 and f['fiscal_quarter'] == 2
        ]
        assert len(q2_records) == 1

        record = q2_records[0]

        # Revenue should be synthesized from premiums
        assert record['revenue'] == 42000000000, (
            f"Expected revenue=42000000000, got {record.get('revenue')}"
        )

        # Original field preserved
        assert record['premiums_earned'] == 42000000000

        # Net margin calculated from synthesized revenue
        expected_margin = (6300000000 / 42000000000) * 100
        assert abs(record['net_margin'] - expected_margin) < 0.01

    def test_standard_revenue_not_overridden_by_industry_fields(self):
        """Test that when standard Revenues tag exists, industry-specific fields don't override it."""
        company_facts = {
            "facts": {
                "us-gaap": {
                    # Standard revenue IS present
                    "Revenues": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": "2025-06-30",
                                    "filed": "2025-08-01",
                                    "fy": 2025,
                                    "fp": "Q2",
                                    "val": 50000000000,
                                },
                            ]
                        }
                    },
                    # Also has banking-specific fields (some diversified financials report both)
                    "InterestIncomeExpenseNet": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": "2025-06-30",
                                    "filed": "2025-08-01",
                                    "fy": 2025,
                                    "fp": "Q2",
                                    "val": 25000000000,
                                },
                            ]
                        }
                    },
                    "NoninterestIncome": {
                        "units": {
                            "USD": [
                                {
                                    "form": "10-Q",
                                    "end": "2025-06-30",
                                    "filed": "2025-08-01",
                                    "fy": 2025,
                                    "fp": "Q2",
                                    "val": 20000000000,
                                },
                            ]
                        }
                    },
                }
            }
        }

        financials = self.client.parse_financial_data(company_facts, num_periods=10)

        q2_records = [
            f for f in financials
            if f['fiscal_year'] == 2025 and f['fiscal_quarter'] == 2
        ]
        assert len(q2_records) == 1

        record = q2_records[0]

        # Standard Revenues should be used, NOT the synthesized banking sum
        assert record['revenue'] == 50000000000, (
            f"Expected revenue=50000000000 (from Revenues tag), got {record.get('revenue')}"
        )


if __name__ == '__main__':
    pytest.main([__file__, '-v'])
