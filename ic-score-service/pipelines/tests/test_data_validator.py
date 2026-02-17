"""Tests for pipeline output data validators."""

from datetime import date, timedelta

import pytest

from pipelines.utils.data_validator import (
    ValidationResult,
    validate_eps_consistency,
    validate_fair_value,
    validate_financial_record,
    validate_range,
    validate_ttm_financials,
)


# =====================================================================
# validate_range helper
# =====================================================================


class TestValidateRange:
    def test_within_bounds(self):
        assert validate_range(50, "field", 0, 100) is None

    def test_below_minimum(self):
        err = validate_range(-5, "field", 0, 100)
        assert err is not None
        assert "below minimum" in err

    def test_above_maximum(self):
        err = validate_range(150, "field", 0, 100)
        assert err is not None
        assert "above maximum" in err

    def test_none_value(self):
        assert validate_range(None, "field", 0, 100) is None


# =====================================================================
# validate_ttm_financials
# =====================================================================


class TestValidateTtmFinancials:
    def test_valid_complete_data(self):
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
            "ttm_period_start": date(2024, 10, 1),
            "ttm_period_end": date(2025, 9, 30),
            "revenue": 390_000_000_000,
            "net_income": 95_000_000_000,
            "eps_basic": 7.60,
            "eps_diluted": 7.58,
            "shares_outstanding": 15_000_000_000,
        }
        result = validate_ttm_financials(data)
        assert result.is_valid
        assert len(result.errors) == 0
        assert len(result.warnings) == 0

    def test_missing_required_field_ticker(self):
        data = {"calculation_date": date.today()}
        result = validate_ttm_financials(data)
        assert not result.is_valid
        assert any("ticker" in e for e in result.errors)

    def test_missing_required_field_calculation_date(self):
        data = {"ticker": "AAPL"}
        result = validate_ttm_financials(data)
        assert not result.is_valid
        assert any("calculation_date" in e for e in result.errors)

    def test_negative_shares_outstanding(self):
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
            "shares_outstanding": -100,
        }
        result = validate_ttm_financials(data)
        assert not result.is_valid
        assert any("shares_outstanding" in e for e in result.errors)

    def test_period_start_after_end(self):
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
            "ttm_period_start": date(2025, 12, 31),
            "ttm_period_end": date(2025, 1, 1),
        }
        result = validate_ttm_financials(data)
        assert not result.is_valid
        assert any("ttm_period_start" in e for e in result.errors)

    def test_future_calculation_date(self):
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today() + timedelta(days=30),
        }
        result = validate_ttm_financials(data)
        assert not result.is_valid
        assert any("future" in e for e in result.errors)

    def test_eps_sign_mismatch_warning(self):
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
            "eps_basic": 5.0,
            "eps_diluted": -3.0,
        }
        result = validate_ttm_financials(data)
        assert result.is_valid  # Warnings don't fail validation
        assert any("sign mismatch" in w.lower() for w in result.warnings)

    def test_partial_data_still_valid(self):
        """Only required fields, all optional missing."""
        data = {
            "ticker": "MSFT",
            "calculation_date": date.today(),
        }
        result = validate_ttm_financials(data)
        assert result.is_valid


# =====================================================================
# validate_fair_value
# =====================================================================


class TestValidateFairValue:
    def test_valid_fair_value_data(self):
        data = {
            "wacc": 10.5,
            "dcf_fair_value": 250.0,
            "graham_number": 180.0,
            "epv_fair_value": 200.0,
            "beta": 1.2,
            "cost_of_equity": 11.0,
            "dcf_upside_percent": 25.0,
        }
        result = validate_fair_value(data)
        assert result.is_valid
        assert len(result.errors) == 0

    def test_wacc_out_of_bounds_high(self):
        data = {"wacc": 25.0}
        result = validate_fair_value(data)
        assert not result.is_valid
        assert any("wacc" in e for e in result.errors)

    def test_wacc_out_of_bounds_low(self):
        data = {"wacc": 3.0}
        result = validate_fair_value(data)
        assert not result.is_valid
        assert any("wacc" in e for e in result.errors)

    def test_negative_dcf_fair_value(self):
        data = {"dcf_fair_value": -50.0}
        result = validate_fair_value(data)
        assert not result.is_valid
        assert any("dcf_fair_value" in e for e in result.errors)

    def test_unreasonable_beta_warning(self):
        data = {"beta": 10.0}
        result = validate_fair_value(data)
        assert result.is_valid  # Warning, not error
        assert any("beta" in w for w in result.warnings)

    def test_all_none_optional_fields_valid(self):
        """No values calculated — still valid."""
        data = {}
        result = validate_fair_value(data)
        assert result.is_valid

    def test_negative_cost_of_equity(self):
        data = {"cost_of_equity": -2.0}
        result = validate_fair_value(data)
        assert not result.is_valid


# =====================================================================
# validate_financial_record
# =====================================================================


class TestValidateFinancialRecord:
    def test_valid_10k_record(self):
        data = {
            "ticker": "AAPL",
            "period_end_date": date(2025, 9, 27),
            "fiscal_year": 2025,
            "fiscal_quarter": None,
            "statement_type": "10-K",
        }
        result = validate_financial_record(data)
        assert result.is_valid

    def test_valid_10q_record(self):
        data = {
            "ticker": "MSFT",
            "period_end_date": date(2025, 3, 31),
            "fiscal_year": 2025,
            "fiscal_quarter": 3,
            "statement_type": "10-Q",
        }
        result = validate_financial_record(data)
        assert result.is_valid

    def test_invalid_statement_type(self):
        data = {
            "ticker": "AAPL",
            "period_end_date": date(2025, 9, 27),
            "fiscal_year": 2025,
            "statement_type": "8-K",
        }
        result = validate_financial_record(data)
        assert not result.is_valid
        assert any("statement_type" in e for e in result.errors)

    def test_invalid_fiscal_quarter(self):
        data = {
            "ticker": "AAPL",
            "period_end_date": date(2025, 9, 27),
            "fiscal_year": 2025,
            "fiscal_quarter": 5,
            "statement_type": "10-Q",
        }
        result = validate_financial_record(data)
        assert not result.is_valid
        assert any("fiscal_quarter" in e for e in result.errors)


# =====================================================================
# validate_eps_consistency
# =====================================================================


class TestValidateEpsConsistency:
    def test_consistent_positive(self):
        result = validate_eps_consistency(
            eps_basic=7.60,
            eps_diluted=7.58,
            net_income=95_000_000_000,
            shares_outstanding=15_000_000_000,
        )
        assert result.is_valid

    def test_sign_mismatch_with_net_income(self):
        result = validate_eps_consistency(
            eps_basic=-2.50,
            eps_diluted=-2.48,
            net_income=5_000_000_000,  # Positive income, negative EPS
            shares_outstanding=2_000_000_000,
        )
        assert not result.is_valid
        assert any("Sign mismatch" in e for e in result.errors)

    def test_none_values_no_errors(self):
        result = validate_eps_consistency(
            eps_basic=None,
            eps_diluted=None,
            net_income=None,
            shares_outstanding=None,
        )
        assert result.is_valid
        assert len(result.errors) == 0
