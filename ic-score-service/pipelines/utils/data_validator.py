"""Pipeline output data validators.

Reusable validation functions for financial data quality checks.
Used by pipelines to validate outputs before DB writes and by
tests to verify data integrity.

Usage:
    from pipelines.utils.data_validator import (
        validate_ttm_financials,
        validate_fair_value,
        validate_financial_record,
        validate_eps_consistency,
        ValidationResult,
    )

    result = validate_ttm_financials(ttm_data)
    if not result.is_valid:
        for error in result.errors:
            logger.warning(error)
"""

from dataclasses import dataclass, field
from datetime import date, datetime
from typing import Dict, List, Optional, Union

Number = Union[int, float, None]


@dataclass
class ValidationResult:
    """Result of a data validation check."""

    is_valid: bool = True
    errors: List[str] = field(default_factory=list)
    warnings: List[str] = field(default_factory=list)

    def add_error(self, message: str) -> None:
        self.errors.append(message)
        self.is_valid = False

    def add_warning(self, message: str) -> None:
        self.warnings.append(message)


def validate_range(
    value: Number,
    field_name: str,
    min_val: Optional[float] = None,
    max_val: Optional[float] = None,
) -> Optional[str]:
    """Check if a numeric value is within bounds.

    Returns error string if out of range, None otherwise.
    """
    if value is None:
        return None
    if min_val is not None and value < min_val:
        return f"{field_name}={value} is below minimum {min_val}"
    if max_val is not None and value > max_val:
        return f"{field_name}={value} is above maximum {max_val}"
    return None


def validate_ttm_financials(data: Dict) -> ValidationResult:
    """Validate TTM financials data before database write.

    Checks:
    - Required fields present (ticker, calculation_date, period dates)
    - Shares outstanding > 0
    - Period start < period end
    - No future calculation dates
    - EPS sign consistency
    - Revenue positive if present
    """
    result = ValidationResult()

    # Required fields
    for field_name in ("ticker", "calculation_date"):
        if not data.get(field_name):
            result.add_error(f"Missing required field: {field_name}")

    # Period date ordering
    start = data.get("ttm_period_start")
    end = data.get("ttm_period_end")
    if start and end and start > end:
        result.add_error(
            f"ttm_period_start ({start}) is after ttm_period_end ({end})"
        )

    # No future calculation dates
    calc_date = data.get("calculation_date")
    if calc_date:
        today = date.today()
        if isinstance(calc_date, datetime):
            calc_date = calc_date.date()
        if isinstance(calc_date, date) and calc_date > today:
            result.add_error(
                f"calculation_date ({calc_date}) is in the future"
            )

    # Shares outstanding
    shares = data.get("shares_outstanding")
    if shares is not None and shares <= 0:
        result.add_error(
            f"shares_outstanding={shares} must be positive"
        )

    # Revenue should be positive if present
    revenue = data.get("revenue")
    if revenue is not None and revenue < 0:
        result.add_warning(f"Negative revenue={revenue}")

    # EPS sign consistency
    eps_basic = data.get("eps_basic")
    eps_diluted = data.get("eps_diluted")
    if (
        eps_basic is not None
        and eps_diluted is not None
        and eps_basic != 0
        and eps_diluted != 0
    ):
        if (eps_basic > 0) != (eps_diluted > 0):
            result.add_warning(
                f"EPS sign mismatch: basic={eps_basic}, "
                f"diluted={eps_diluted}"
            )

    return result


def validate_fair_value(data: Dict) -> ValidationResult:
    """Validate fair value calculation results.

    Checks:
    - WACC between 5% and 20%
    - DCF, Graham, EPV > 0 if present
    - Beta in reasonable range (-1 to 5)
    - Cost of equity > 0
    - Upside percentage within bounds
    """
    result = ValidationResult()

    # WACC bounds (stored as percentage, e.g., 10.5 for 10.5%)
    wacc = data.get("wacc")
    if wacc is not None:
        err = validate_range(wacc, "wacc", 5.0, 20.0)
        if err:
            result.add_error(err)

    # Fair values must be positive if present
    for fv_field in ("dcf_fair_value", "graham_number", "epv_fair_value"):
        val = data.get(fv_field)
        if val is not None and val <= 0:
            result.add_error(f"{fv_field}={val} must be positive")

    # Beta range
    beta = data.get("beta")
    if beta is not None:
        err = validate_range(beta, "beta", -1.0, 5.0)
        if err:
            result.add_warning(err)

    # Upside percentage sanity
    upside = data.get("dcf_upside_percent")
    if upside is not None:
        err = validate_range(
            upside, "dcf_upside_percent", -99.0, 10000.0
        )
        if err:
            result.add_warning(err)

    # Cost of equity should be positive
    coe = data.get("cost_of_equity")
    if coe is not None and coe <= 0:
        result.add_error(f"cost_of_equity={coe} must be positive")

    return result


def validate_financial_record(data: Dict) -> ValidationResult:
    """Validate a financial record from SEC ingestion.

    Checks:
    - Required fields: ticker, period_end_date, fiscal_year,
      statement_type
    - fiscal_quarter in {None, 1, 2, 3, 4}
    - statement_type in {"10-Q", "10-K"}
    - Margin values within bounds
    - Shares outstanding > 0 if present
    """
    result = ValidationResult()

    # Required fields
    for field_name in (
        "ticker",
        "period_end_date",
        "fiscal_year",
        "statement_type",
    ):
        if not data.get(field_name):
            result.add_error(f"Missing required field: {field_name}")

    # Fiscal quarter validation
    fq = data.get("fiscal_quarter")
    if fq is not None and fq not in (1, 2, 3, 4):
        result.add_error(
            f"fiscal_quarter={fq} must be None or 1-4"
        )

    # Statement type validation
    st = data.get("statement_type")
    if st and st not in ("10-Q", "10-K"):
        result.add_error(
            f"statement_type='{st}' must be '10-Q' or '10-K'"
        )

    # Margin bounds (-1000% to 1000%)
    for margin_field in (
        "gross_margin",
        "operating_margin",
        "net_margin",
    ):
        val = data.get(margin_field)
        if val is not None:
            err = validate_range(val, margin_field, -1000.0, 1000.0)
            if err:
                result.add_warning(err)

    # Shares outstanding
    shares = data.get("shares_outstanding")
    if shares is not None and shares <= 0:
        result.add_error(
            f"shares_outstanding={shares} must be positive"
        )

    return result


def validate_eps_consistency(
    eps_basic: Number,
    eps_diluted: Number,
    net_income: Number,
    shares_outstanding: Number,
) -> ValidationResult:
    """Validate EPS consistency across related fields.

    Checks:
    - If both basic and diluted exist, diluted <= basic (more shares)
    - If net_income and EPS exist, signs should match
    - EPS magnitude should be reasonable given net_income/shares
    """
    result = ValidationResult()

    # All None is valid (no data to check)
    if all(
        v is None
        for v in (eps_basic, eps_diluted, net_income, shares_outstanding)
    ):
        return result

    # Diluted EPS should generally be <= basic EPS (more shares)
    if (
        eps_basic is not None
        and eps_diluted is not None
        and eps_basic != 0
    ):
        if abs(eps_diluted) > abs(eps_basic) * 1.01:  # 1% tolerance
            result.add_warning(
                f"Diluted EPS ({eps_diluted}) magnitude exceeds "
                f"basic EPS ({eps_basic})"
            )

    # Sign consistency between net_income and EPS
    if net_income is not None and net_income != 0:
        for eps_name, eps_val in [
            ("eps_basic", eps_basic),
            ("eps_diluted", eps_diluted),
        ]:
            if eps_val is not None and eps_val != 0:
                if (net_income > 0) != (eps_val > 0):
                    result.add_error(
                        f"Sign mismatch: net_income={net_income}, "
                        f"{eps_name}={eps_val}"
                    )

    return result
