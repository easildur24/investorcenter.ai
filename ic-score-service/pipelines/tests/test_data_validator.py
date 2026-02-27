"""Tests for pipeline output data validators.

Comprehensive unit tests covering all validation functions with valid inputs,
invalid inputs, edge cases (None, empty, boundary values), and error messages.
"""

from datetime import date, datetime, timedelta

import pytest

from pipelines.utils.data_validator import (
    VALID_CONFIDENCE_LEVELS,
    VALID_IC_RATINGS,
    ValidationResult,
    validate_eps_consistency,
    validate_fair_value,
    validate_financial_record,
    validate_ic_score_output,
    validate_pipeline_coverage,
    validate_range,
    validate_risk_metrics_output,
    validate_ttm_financials,
)


# =====================================================================
# ValidationResult dataclass
# =====================================================================


class TestValidationResult:
    """Tests for the ValidationResult dataclass itself."""

    def test_default_is_valid(self):
        result = ValidationResult()
        assert result.is_valid is True
        assert result.errors == []
        assert result.warnings == []

    def test_add_error_marks_invalid(self):
        result = ValidationResult()
        assert result.is_valid is True
        result.add_error("something went wrong")
        assert result.is_valid is False
        assert len(result.errors) == 1
        assert result.errors[0] == "something went wrong"

    def test_add_warning_keeps_valid(self):
        result = ValidationResult()
        result.add_warning("heads up")
        assert result.is_valid is True
        assert len(result.warnings) == 1
        assert result.warnings[0] == "heads up"

    def test_multiple_errors(self):
        result = ValidationResult()
        result.add_error("error1")
        result.add_error("error2")
        assert not result.is_valid
        assert len(result.errors) == 2

    def test_mixed_errors_and_warnings(self):
        result = ValidationResult()
        result.add_warning("warn")
        assert result.is_valid
        result.add_error("err")
        assert not result.is_valid
        assert len(result.errors) == 1
        assert len(result.warnings) == 1


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
        assert "field" in err
        assert "-5" in err

    def test_above_maximum(self):
        err = validate_range(150, "field", 0, 100)
        assert err is not None
        assert "above maximum" in err
        assert "field" in err
        assert "150" in err

    def test_none_value(self):
        assert validate_range(None, "field", 0, 100) is None

    def test_at_exact_minimum(self):
        """Value exactly at minimum should pass."""
        assert validate_range(0, "field", 0, 100) is None

    def test_at_exact_maximum(self):
        """Value exactly at maximum should pass."""
        assert validate_range(100, "field", 0, 100) is None

    def test_min_only(self):
        """Only min_val specified, no max_val."""
        assert validate_range(50, "field", min_val=0) is None
        err = validate_range(-1, "field", min_val=0)
        assert err is not None
        assert "below minimum" in err

    def test_max_only(self):
        """Only max_val specified, no min_val."""
        assert validate_range(50, "field", max_val=100) is None
        err = validate_range(101, "field", max_val=100)
        assert err is not None
        assert "above maximum" in err

    def test_no_bounds(self):
        """Neither min_val nor max_val specified."""
        assert validate_range(999, "field") is None
        assert validate_range(-999, "field") is None

    def test_float_value(self):
        assert validate_range(0.5, "field", 0.0, 1.0) is None
        err = validate_range(1.0001, "field", 0.0, 1.0)
        assert err is not None

    def test_negative_bounds(self):
        assert validate_range(-0.5, "field", -1.0, 0.0) is None
        err = validate_range(0.1, "field", -1.0, 0.0)
        assert err is not None

    def test_field_name_in_error_message(self):
        """The field name should appear in the error string."""
        err = validate_range(-5, "my_custom_field", 0, 100)
        assert "my_custom_field" in err


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

    def test_both_required_fields_missing(self):
        result = validate_ttm_financials({})
        assert not result.is_valid
        assert len(result.errors) >= 2

    def test_empty_string_ticker(self):
        """Empty string should be treated as missing."""
        data = {"ticker": "", "calculation_date": date.today()}
        result = validate_ttm_financials(data)
        assert not result.is_valid
        assert any("ticker" in e for e in result.errors)

    def test_none_ticker(self):
        """None ticker should be treated as missing."""
        data = {"ticker": None, "calculation_date": date.today()}
        result = validate_ttm_financials(data)
        assert not result.is_valid

    def test_negative_shares_outstanding(self):
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
            "shares_outstanding": -100,
        }
        result = validate_ttm_financials(data)
        assert not result.is_valid
        assert any("shares_outstanding" in e for e in result.errors)

    def test_zero_shares_outstanding(self):
        """Zero shares should also be invalid (must be positive)."""
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
            "shares_outstanding": 0,
        }
        result = validate_ttm_financials(data)
        assert not result.is_valid
        assert any("shares_outstanding" in e for e in result.errors)

    def test_shares_outstanding_none_valid(self):
        """None shares_outstanding should be valid (optional field)."""
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
        }
        result = validate_ttm_financials(data)
        assert result.is_valid

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

    def test_period_start_equals_end(self):
        """Same start and end is technically valid (not start > end)."""
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
            "ttm_period_start": date(2025, 6, 15),
            "ttm_period_end": date(2025, 6, 15),
        }
        result = validate_ttm_financials(data)
        assert result.is_valid

    def test_period_dates_missing_is_valid(self):
        """Missing period dates are fine (optional)."""
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
        }
        result = validate_ttm_financials(data)
        assert result.is_valid

    def test_future_calculation_date(self):
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today() + timedelta(days=30),
        }
        result = validate_ttm_financials(data)
        assert not result.is_valid
        assert any("future" in e for e in result.errors)

    def test_today_calculation_date_valid(self):
        """Today's date should be valid (not future)."""
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
        }
        result = validate_ttm_financials(data)
        assert result.is_valid

    def test_datetime_calculation_date_future(self):
        """datetime object (not just date) should also be checked."""
        data = {
            "ticker": "AAPL",
            "calculation_date": datetime.now() + timedelta(days=30),
        }
        result = validate_ttm_financials(data)
        assert not result.is_valid
        assert any("future" in e for e in result.errors)

    def test_negative_revenue_warning(self):
        """Negative revenue should produce a warning, not error."""
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
            "revenue": -1_000_000,
        }
        result = validate_ttm_financials(data)
        assert result.is_valid  # Warning only
        assert any("revenue" in w.lower() for w in result.warnings)

    def test_zero_revenue_valid(self):
        """Zero revenue should not trigger a warning (>= 0 is fine)."""
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
            "revenue": 0,
        }
        result = validate_ttm_financials(data)
        assert result.is_valid
        assert len(result.warnings) == 0

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

    def test_eps_both_positive_no_warning(self):
        """Consistent positive EPS should not warn."""
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
            "eps_basic": 5.0,
            "eps_diluted": 4.9,
        }
        result = validate_ttm_financials(data)
        assert len(result.warnings) == 0

    def test_eps_both_negative_no_warning(self):
        """Consistent negative EPS should not warn."""
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
            "eps_basic": -2.0,
            "eps_diluted": -2.1,
        }
        result = validate_ttm_financials(data)
        assert len(result.warnings) == 0

    def test_eps_one_zero_no_warning(self):
        """If either EPS is zero, sign check should be skipped."""
        data = {
            "ticker": "AAPL",
            "calculation_date": date.today(),
            "eps_basic": 0,
            "eps_diluted": -2.0,
        }
        result = validate_ttm_financials(data)
        # Zero check means the sign comparison is skipped
        assert len(result.warnings) == 0

    def test_partial_data_still_valid(self):
        """Only required fields, all optional missing."""
        data = {
            "ticker": "MSFT",
            "calculation_date": date.today(),
        }
        result = validate_ttm_financials(data)
        assert result.is_valid

    def test_multiple_errors_accumulated(self):
        """Multiple validation failures should accumulate errors."""
        data = {
            # Missing ticker
            # Missing calculation_date
            "shares_outstanding": -100,
            "ttm_period_start": date(2025, 12, 31),
            "ttm_period_end": date(2025, 1, 1),
        }
        result = validate_ttm_financials(data)
        assert not result.is_valid
        assert len(result.errors) >= 3  # ticker, calc_date, shares, period


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

    def test_wacc_at_exact_lower_bound(self):
        """WACC exactly at 5.0 should be valid."""
        data = {"wacc": 5.0}
        result = validate_fair_value(data)
        assert result.is_valid

    def test_wacc_at_exact_upper_bound(self):
        """WACC exactly at 20.0 should be valid."""
        data = {"wacc": 20.0}
        result = validate_fair_value(data)
        assert result.is_valid

    def test_negative_dcf_fair_value(self):
        data = {"dcf_fair_value": -50.0}
        result = validate_fair_value(data)
        assert not result.is_valid
        assert any("dcf_fair_value" in e for e in result.errors)

    def test_zero_dcf_fair_value(self):
        """Zero fair value should also be invalid (must be positive)."""
        data = {"dcf_fair_value": 0}
        result = validate_fair_value(data)
        assert not result.is_valid

    def test_negative_graham_number(self):
        data = {"graham_number": -10.0}
        result = validate_fair_value(data)
        assert not result.is_valid
        assert any("graham_number" in e for e in result.errors)

    def test_negative_epv_fair_value(self):
        data = {"epv_fair_value": -5.0}
        result = validate_fair_value(data)
        assert not result.is_valid
        assert any("epv_fair_value" in e for e in result.errors)

    def test_unreasonable_beta_warning(self):
        data = {"beta": 10.0}
        result = validate_fair_value(data)
        assert result.is_valid  # Warning, not error
        assert any("beta" in w for w in result.warnings)

    def test_negative_beta_within_range(self):
        """Beta of -0.5 is in range [-1, 5]."""
        data = {"beta": -0.5}
        result = validate_fair_value(data)
        assert result.is_valid
        assert len(result.warnings) == 0

    def test_beta_below_negative_one_warning(self):
        """Beta below -1 should trigger warning."""
        data = {"beta": -1.5}
        result = validate_fair_value(data)
        assert result.is_valid  # Warning only
        assert any("beta" in w for w in result.warnings)

    def test_upside_percent_extreme_high_warning(self):
        data = {"dcf_upside_percent": 15000.0}
        result = validate_fair_value(data)
        assert result.is_valid  # Warning only
        assert any("dcf_upside_percent" in w for w in result.warnings)

    def test_upside_percent_extreme_low_warning(self):
        data = {"dcf_upside_percent": -99.5}
        result = validate_fair_value(data)
        assert result.is_valid  # Warning only
        assert any("dcf_upside_percent" in w for w in result.warnings)

    def test_upside_percent_normal(self):
        """Normal upside should not produce warnings."""
        data = {"dcf_upside_percent": 50.0}
        result = validate_fair_value(data)
        assert len(result.warnings) == 0

    def test_all_none_optional_fields_valid(self):
        """No values calculated -- still valid."""
        data = {}
        result = validate_fair_value(data)
        assert result.is_valid

    def test_negative_cost_of_equity(self):
        data = {"cost_of_equity": -2.0}
        result = validate_fair_value(data)
        assert not result.is_valid
        assert any("cost_of_equity" in e for e in result.errors)

    def test_zero_cost_of_equity(self):
        """Zero cost of equity should also be invalid (must be positive)."""
        data = {"cost_of_equity": 0}
        result = validate_fair_value(data)
        assert not result.is_valid

    def test_multiple_fair_value_errors(self):
        """Multiple invalid fair value fields should all produce errors."""
        data = {
            "dcf_fair_value": -10.0,
            "graham_number": -5.0,
            "epv_fair_value": 0,
            "wacc": 50.0,
            "cost_of_equity": -1.0,
        }
        result = validate_fair_value(data)
        assert not result.is_valid
        assert len(result.errors) >= 4


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

    def test_all_valid_quarters(self):
        """Each quarter 1-4 should be valid."""
        for q in (1, 2, 3, 4):
            data = {
                "ticker": "AAPL",
                "period_end_date": date(2025, 3, 31),
                "fiscal_year": 2025,
                "fiscal_quarter": q,
                "statement_type": "10-Q",
            }
            result = validate_financial_record(data)
            assert result.is_valid, f"Quarter {q} should be valid"

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

    def test_fiscal_quarter_zero_invalid(self):
        """Quarter 0 should be invalid."""
        data = {
            "ticker": "AAPL",
            "period_end_date": date(2025, 9, 27),
            "fiscal_year": 2025,
            "fiscal_quarter": 0,
            "statement_type": "10-Q",
        }
        result = validate_financial_record(data)
        assert not result.is_valid

    def test_fiscal_quarter_negative_invalid(self):
        """Negative quarter should be invalid."""
        data = {
            "ticker": "AAPL",
            "period_end_date": date(2025, 9, 27),
            "fiscal_year": 2025,
            "fiscal_quarter": -1,
            "statement_type": "10-Q",
        }
        result = validate_financial_record(data)
        assert not result.is_valid

    def test_missing_all_required_fields(self):
        result = validate_financial_record({})
        assert not result.is_valid
        assert len(result.errors) >= 4

    def test_missing_ticker(self):
        data = {
            "period_end_date": date(2025, 9, 27),
            "fiscal_year": 2025,
            "statement_type": "10-K",
        }
        result = validate_financial_record(data)
        assert not result.is_valid
        assert any("ticker" in e for e in result.errors)

    def test_missing_period_end_date(self):
        data = {
            "ticker": "AAPL",
            "fiscal_year": 2025,
            "statement_type": "10-K",
        }
        result = validate_financial_record(data)
        assert not result.is_valid
        assert any("period_end_date" in e for e in result.errors)

    def test_margin_warning_gross_margin(self):
        """Extreme gross margin should produce a warning."""
        data = {
            "ticker": "AAPL",
            "period_end_date": date(2025, 9, 27),
            "fiscal_year": 2025,
            "statement_type": "10-K",
            "gross_margin": 1500.0,  # Way above 1000
        }
        result = validate_financial_record(data)
        assert result.is_valid  # Margins produce warnings, not errors
        assert any("gross_margin" in w for w in result.warnings)

    def test_margin_within_bounds_no_warning(self):
        """Normal margins should not produce warnings."""
        data = {
            "ticker": "AAPL",
            "period_end_date": date(2025, 9, 27),
            "fiscal_year": 2025,
            "statement_type": "10-K",
            "gross_margin": 45.0,
            "operating_margin": 30.0,
            "net_margin": 25.0,
        }
        result = validate_financial_record(data)
        assert result.is_valid
        assert len(result.warnings) == 0

    def test_negative_margin_extreme_warning(self):
        """Very negative margin below -1000 should trigger warning."""
        data = {
            "ticker": "AAPL",
            "period_end_date": date(2025, 9, 27),
            "fiscal_year": 2025,
            "statement_type": "10-K",
            "operating_margin": -1500.0,
        }
        result = validate_financial_record(data)
        assert any("operating_margin" in w for w in result.warnings)

    def test_shares_outstanding_negative(self):
        data = {
            "ticker": "AAPL",
            "period_end_date": date(2025, 9, 27),
            "fiscal_year": 2025,
            "statement_type": "10-K",
            "shares_outstanding": -1,
        }
        result = validate_financial_record(data)
        assert not result.is_valid
        assert any("shares_outstanding" in e for e in result.errors)

    def test_shares_outstanding_none_valid(self):
        """None shares_outstanding is valid (optional)."""
        data = {
            "ticker": "AAPL",
            "period_end_date": date(2025, 9, 27),
            "fiscal_year": 2025,
            "statement_type": "10-K",
        }
        result = validate_financial_record(data)
        assert result.is_valid


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
        assert len(result.errors) == 0

    def test_consistent_negative(self):
        """Consistently negative values should be valid."""
        result = validate_eps_consistency(
            eps_basic=-2.50,
            eps_diluted=-2.48,
            net_income=-5_000_000_000,
            shares_outstanding=2_000_000_000,
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

    def test_sign_mismatch_positive_eps_negative_income(self):
        """Positive EPS with negative net income should fail."""
        result = validate_eps_consistency(
            eps_basic=3.0,
            eps_diluted=2.9,
            net_income=-500_000_000,
            shares_outstanding=1_000_000_000,
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
        assert len(result.warnings) == 0

    def test_diluted_exceeds_basic_warning(self):
        """Diluted EPS magnitude > basic EPS magnitude is unusual."""
        result = validate_eps_consistency(
            eps_basic=5.0,
            eps_diluted=6.0,  # |6| > |5| * 1.01
            net_income=10_000_000_000,
            shares_outstanding=2_000_000_000,
        )
        assert result.is_valid  # It's a warning
        assert any("Diluted EPS" in w for w in result.warnings)

    def test_diluted_slightly_exceeds_basic_within_tolerance(self):
        """Diluted EPS within 1% tolerance of basic should not warn."""
        result = validate_eps_consistency(
            eps_basic=5.0,
            eps_diluted=5.04,  # |5.04| < |5.0| * 1.01 = 5.05
            net_income=10_000_000_000,
            shares_outstanding=2_000_000_000,
        )
        assert len(result.warnings) == 0

    def test_zero_eps_basic_skips_diluted_check(self):
        """Zero basic EPS should skip the diluted comparison."""
        result = validate_eps_consistency(
            eps_basic=0,
            eps_diluted=-2.0,
            net_income=0,
            shares_outstanding=1_000_000_000,
        )
        assert result.is_valid
        assert len(result.warnings) == 0

    def test_zero_net_income_skips_sign_check(self):
        """Zero net income should skip sign consistency check."""
        result = validate_eps_consistency(
            eps_basic=-1.0,
            eps_diluted=-0.9,
            net_income=0,
            shares_outstanding=1_000_000_000,
        )
        assert result.is_valid

    def test_partial_none_values(self):
        """Some values None, others present should be handled."""
        result = validate_eps_consistency(
            eps_basic=5.0,
            eps_diluted=None,
            net_income=None,
            shares_outstanding=None,
        )
        assert result.is_valid

    def test_only_net_income_and_shares(self):
        """Only net_income and shares provided, no EPS."""
        result = validate_eps_consistency(
            eps_basic=None,
            eps_diluted=None,
            net_income=100_000_000,
            shares_outstanding=10_000_000,
        )
        assert result.is_valid


# =====================================================================
# validate_risk_metrics_output
# =====================================================================


class TestValidateRiskMetricsOutput:
    def test_valid_risk_metrics(self):
        data = {
            "beta": 1.15,
            "sharpe_ratio": 1.5,
            "max_drawdown": -0.18,
            "volatility": 0.22,
            "var_95": -0.028,
        }
        result = validate_risk_metrics_output(data)
        assert result.is_valid

    def test_beta_out_of_range(self):
        data = {"beta": 8.0}
        result = validate_risk_metrics_output(data)
        assert not result.is_valid
        assert any("beta" in e for e in result.errors)

    def test_beta_below_range(self):
        data = {"beta": -3.0}
        result = validate_risk_metrics_output(data)
        assert not result.is_valid
        assert any("beta" in e for e in result.errors)

    def test_beta_at_bounds(self):
        """Beta at exact bounds should be valid."""
        assert validate_risk_metrics_output({"beta": -2.0}).is_valid
        assert validate_risk_metrics_output({"beta": 5.0}).is_valid

    def test_sharpe_ratio_out_of_range(self):
        data = {"sharpe_ratio": 15.0}
        result = validate_risk_metrics_output(data)
        assert not result.is_valid

    def test_sharpe_ratio_below_range(self):
        data = {"sharpe_ratio": -6.0}
        result = validate_risk_metrics_output(data)
        assert not result.is_valid

    def test_max_drawdown_positive_invalid(self):
        data = {"max_drawdown": 0.5}
        result = validate_risk_metrics_output(data)
        assert not result.is_valid

    def test_max_drawdown_zero_valid(self):
        """Max drawdown of 0 is at the boundary and should be valid."""
        data = {"max_drawdown": 0.0}
        result = validate_risk_metrics_output(data)
        assert result.is_valid

    def test_max_drawdown_negative_one_valid(self):
        """Max drawdown of -1.0 means 100% loss -- at boundary."""
        data = {"max_drawdown": -1.0}
        result = validate_risk_metrics_output(data)
        assert result.is_valid

    def test_volatility_out_of_range(self):
        data = {"volatility": 3.0}
        result = validate_risk_metrics_output(data)
        assert not result.is_valid

    def test_volatility_negative(self):
        """Negative volatility should be invalid."""
        data = {"volatility": -0.1}
        result = validate_risk_metrics_output(data)
        assert not result.is_valid

    def test_var95_positive_invalid(self):
        data = {"var_95": 0.05}
        result = validate_risk_metrics_output(data)
        assert not result.is_valid
        assert any("var_95" in e for e in result.errors)

    def test_var95_zero_valid(self):
        """VaR95 of exactly 0 should be valid (<= 0)."""
        data = {"var_95": 0}
        result = validate_risk_metrics_output(data)
        assert result.is_valid

    def test_var95_negative_valid(self):
        data = {"var_95": -0.05}
        result = validate_risk_metrics_output(data)
        assert result.is_valid

    def test_empty_data_valid(self):
        result = validate_risk_metrics_output({})
        assert result.is_valid

    def test_all_at_boundaries(self):
        """All values at their respective boundary extremes."""
        data = {
            "beta": -2.0,
            "sharpe_ratio": -5.0,
            "max_drawdown": -1.0,
            "volatility": 0.0,
            "var_95": 0.0,
        }
        result = validate_risk_metrics_output(data)
        assert result.is_valid


# =====================================================================
# validate_ic_score_output
# =====================================================================


class TestValidateIcScoreOutput:
    def test_valid_ic_score(self):
        data = {
            "overall_score": 75.0,
            "value_score": 80.0,
            "growth_score": 70.0,
            "profitability_score": 85.0,
            "momentum_score": 60.0,
            "stability_score": 72.0,
            "rating": "Buy",
            "confidence_level": "High",
        }
        result = validate_ic_score_output(data)
        assert result.is_valid

    def test_score_out_of_range_high(self):
        data = {"overall_score": 150.0}
        result = validate_ic_score_output(data)
        assert not result.is_valid
        assert any("overall_score" in e for e in result.errors)

    def test_score_out_of_range_low(self):
        data = {"overall_score": 0.0}
        result = validate_ic_score_output(data)
        assert not result.is_valid

    def test_score_at_lower_bound(self):
        """Overall score of exactly 1.0 should be valid."""
        data = {"overall_score": 1.0}
        result = validate_ic_score_output(data)
        assert result.is_valid

    def test_score_at_upper_bound(self):
        """Overall score of exactly 100.0 should be valid."""
        data = {"overall_score": 100.0}
        result = validate_ic_score_output(data)
        assert result.is_valid

    def test_missing_overall_score(self):
        data = {"rating": "Buy"}
        result = validate_ic_score_output(data)
        assert not result.is_valid
        assert any("overall_score" in e for e in result.errors)

    def test_factor_score_out_of_range(self):
        """Factor scores above 100 should fail."""
        data = {"overall_score": 50.0, "value_score": 110.0}
        result = validate_ic_score_output(data)
        assert not result.is_valid
        assert any("value_score" in e for e in result.errors)

    def test_factor_score_negative(self):
        """Negative factor score should fail."""
        data = {"overall_score": 50.0, "growth_score": -5.0}
        result = validate_ic_score_output(data)
        assert not result.is_valid

    def test_factor_score_at_bounds(self):
        """Factor scores at 0 and 100 should be valid."""
        data = {
            "overall_score": 50.0,
            "value_score": 0.0,
            "growth_score": 100.0,
        }
        result = validate_ic_score_output(data)
        assert result.is_valid

    def test_all_valid_ratings(self):
        """Every rating in VALID_IC_RATINGS should pass."""
        for rating in VALID_IC_RATINGS:
            data = {"overall_score": 50.0, "rating": rating}
            result = validate_ic_score_output(data)
            assert result.is_valid, f"Rating '{rating}' should be valid"

    def test_invalid_rating(self):
        data = {"overall_score": 50.0, "rating": "Very Buy"}
        result = validate_ic_score_output(data)
        assert not result.is_valid
        assert any("rating" in e for e in result.errors)

    def test_all_valid_confidence_levels(self):
        """Every confidence level in VALID_CONFIDENCE_LEVELS should pass."""
        for conf in VALID_CONFIDENCE_LEVELS:
            data = {"overall_score": 50.0, "confidence_level": conf}
            result = validate_ic_score_output(data)
            assert result.is_valid, f"Confidence '{conf}' should be valid"

    def test_invalid_confidence_level(self):
        data = {
            "overall_score": 50.0,
            "confidence_level": "Super High",
        }
        result = validate_ic_score_output(data)
        assert not result.is_valid
        assert any("confidence_level" in e for e in result.errors)

    def test_none_rating_valid(self):
        """None rating should be valid (optional field)."""
        data = {"overall_score": 50.0}
        result = validate_ic_score_output(data)
        assert result.is_valid

    def test_none_confidence_valid(self):
        """None confidence should be valid (optional field)."""
        data = {"overall_score": 50.0}
        result = validate_ic_score_output(data)
        assert result.is_valid

    def test_empty_data_missing_overall_score(self):
        result = validate_ic_score_output({})
        assert not result.is_valid
        assert any("overall_score" in e for e in result.errors)


# =====================================================================
# validate_pipeline_coverage
# =====================================================================


class TestValidatePipelineCoverage:
    def test_good_coverage(self):
        result = validate_pipeline_coverage(
            "ttm_financials", expected_count=100, actual_count=90
        )
        assert result.is_valid
        assert any("90.0%" in w for w in result.warnings)

    def test_low_coverage_fails(self):
        result = validate_pipeline_coverage(
            "ttm_financials", expected_count=100, actual_count=50
        )
        assert not result.is_valid
        assert any("50.0%" in e for e in result.errors)

    def test_exact_threshold(self):
        result = validate_pipeline_coverage(
            "ttm_financials",
            expected_count=100,
            actual_count=70,
            min_coverage_pct=70.0,
        )
        assert result.is_valid

    def test_just_below_threshold(self):
        result = validate_pipeline_coverage(
            "ttm_financials",
            expected_count=100,
            actual_count=69,
            min_coverage_pct=70.0,
        )
        assert not result.is_valid

    def test_zero_expected(self):
        result = validate_pipeline_coverage(
            "ttm_financials", expected_count=0, actual_count=0
        )
        assert result.is_valid  # Warning, not error

    def test_negative_expected(self):
        """Negative expected_count should produce warning."""
        result = validate_pipeline_coverage(
            "ttm_financials", expected_count=-5, actual_count=0
        )
        assert result.is_valid  # Warning, not error
        assert any("expected_count" in w for w in result.warnings)

    def test_full_coverage(self):
        result = validate_pipeline_coverage(
            "ttm_financials", expected_count=100, actual_count=100
        )
        assert result.is_valid
        assert any("100.0%" in w for w in result.warnings)

    def test_over_coverage(self):
        """More actual than expected (>100%) should still be valid."""
        result = validate_pipeline_coverage(
            "ttm_financials", expected_count=100, actual_count=110
        )
        assert result.is_valid

    def test_pipeline_name_in_message(self):
        """Pipeline name should appear in error/warning messages."""
        result = validate_pipeline_coverage(
            "my_pipeline", expected_count=100, actual_count=50
        )
        assert any("my_pipeline" in e for e in result.errors)

    def test_custom_min_coverage_pct(self):
        """Custom coverage threshold should work."""
        result = validate_pipeline_coverage(
            "ttm_financials",
            expected_count=100,
            actual_count=95,
            min_coverage_pct=95.0,
        )
        assert result.is_valid

        result = validate_pipeline_coverage(
            "ttm_financials",
            expected_count=100,
            actual_count=94,
            min_coverage_pct=95.0,
        )
        assert not result.is_valid


# =====================================================================
# Constants validation
# =====================================================================


class TestValidatorConstants:
    """Tests for the module-level constants."""

    def test_valid_ic_ratings_not_empty(self):
        assert len(VALID_IC_RATINGS) > 0

    def test_valid_confidence_levels_not_empty(self):
        assert len(VALID_CONFIDENCE_LEVELS) > 0

    def test_expected_ic_ratings(self):
        expected = {"Strong Buy", "Buy", "Hold", "Sell", "Strong Sell"}
        assert VALID_IC_RATINGS == expected

    def test_expected_confidence_levels(self):
        expected = {"Very High", "High", "Medium", "Low", "Very Low"}
        assert VALID_CONFIDENCE_LEVELS == expected
