"""Integration tests for post-run data quality validation.

Requires a real PostgreSQL + TimescaleDB database.
Run with: INTEGRATION_TEST_DB=true pytest ... -v

Tests the data_validator module to verify that pipeline
outputs pass quality checks when data is correct and that
validation catches invalid or out-of-range data.
"""

import os
from datetime import date, timedelta

import pytest
import pytest_asyncio

from sqlalchemy import text

pytestmark = [
    pytest.mark.skipif(
        os.getenv("INTEGRATION_TEST_DB") != "true",
        reason="INTEGRATION_TEST_DB not set",
    ),
    pytest.mark.asyncio,
]

from pipelines.tests.integration.seed_data import (
    seed_companies,
    seed_ttm_financials,
)

from pipelines.utils.data_validator import (
    validate_ttm_financials,
    ValidationResult,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _build_ttm_dict(overrides=None):
    """Build a valid TTM financials dict, optionally overriding
    specific fields."""
    today = date.today()
    data = {
        "ticker": "AAPL",
        "calculation_date": today,
        "ttm_period_start": today - timedelta(days=365),
        "ttm_period_end": today - timedelta(days=1),
        "revenue": 95_000_000_000,
        "cost_of_revenue": 55_000_000_000,
        "gross_profit": 40_000_000_000,
        "operating_income": 27_000_000_000,
        "net_income": 21_000_000_000,
        "eps_basic": 1.3548,
        "eps_diluted": 1.3277,
        "shares_outstanding": 15_500_000_000,
        "total_assets": 330_000_000_000,
        "total_liabilities": 240_000_000_000,
        "shareholders_equity": 90_000_000_000,
    }
    if overrides:
        data.update(overrides)
    return data


def _validate_ic_score_range(overall_score):
    """Simple inline validator: IC Score must be 1-100."""
    result = ValidationResult()
    if overall_score is None:
        result.add_error("overall_score is None")
    elif overall_score < 1 or overall_score > 100:
        result.add_error(
            f"overall_score={overall_score} out of range [1, 100]"
        )
    return result


def _validate_pipeline_coverage(
    total_companies, companies_with_data
):
    """Validate that pipeline output covers a sufficient
    fraction of target companies.

    Returns a ValidationResult with is_valid=True if coverage
    >= 50%, otherwise adds an error.
    """
    result = ValidationResult()
    if total_companies == 0:
        result.add_error("No companies in scope")
        return result

    coverage = companies_with_data / total_companies * 100
    result.coverage_percent = coverage  # type: ignore[attr-defined]

    if coverage < 50:
        result.add_warning(
            f"Low coverage: {coverage:.1f}% "
            f"({companies_with_data}/{total_companies})"
        )
    return result


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


async def test_ttm_output_passes_validation(db, session):
    """Seed valid TTM data, run validate_ttm_financials on it,
    verify is_valid=True."""
    data = _build_ttm_dict()
    result = validate_ttm_financials(data)

    assert result.is_valid is True, (
        f"Expected valid TTM data, got errors: {result.errors}"
    )
    assert len(result.errors) == 0


async def test_invalid_ttm_detected(db, session):
    """Insert TTM with revenue=0, shares_outstanding=0, and
    verify validation catches it."""
    data = _build_ttm_dict(
        overrides={
            "shares_outstanding": 0,
        }
    )
    result = validate_ttm_financials(data)

    assert result.is_valid is False, (
        "Expected validation to fail for shares_outstanding=0"
    )
    # Check that the specific error about shares is present
    has_shares_error = any(
        "shares_outstanding" in e for e in result.errors
    )
    assert has_shares_error, (
        f"Expected shares_outstanding error, got: {result.errors}"
    )


async def test_ic_score_output_valid(db, session):
    """Insert ic_score with overall_score=75, verify validation
    passes."""
    result = _validate_ic_score_range(75)

    assert result.is_valid is True, (
        f"Expected valid IC Score, got errors: {result.errors}"
    )


async def test_ic_score_out_of_range(db, session):
    """Insert ic_score with overall_score=150, verify validation
    catches it."""
    result = _validate_ic_score_range(150)

    assert result.is_valid is False, (
        "Expected validation to fail for overall_score=150"
    )
    has_range_error = any(
        "out of range" in e for e in result.errors
    )
    assert has_range_error, (
        f"Expected out-of-range error, got: {result.errors}"
    )


async def test_coverage_check(db, session):
    """Seed 10 companies, TTM for 7 -> coverage is 70%, verify
    validate_pipeline_coverage reports correctly."""
    tickers = [
        "AAPL", "MSFT", "GOOGL", "AMZN", "META",
        "NVDA", "TSLA", "JPM", "JNJ", "XOM",
    ]
    await seed_companies(session, tickers=tickers)

    # Seed TTM for 7 of 10 companies
    for ticker in tickers[:7]:
        await seed_ttm_financials(session, ticker)

    # Verify actual data in DB
    async with db.session() as s:
        total_result = await s.execute(
            text("SELECT COUNT(*) FROM companies")
        )
        total = total_result.scalar()

        ttm_result = await s.execute(
            text(
                "SELECT COUNT(DISTINCT ticker)"
                " FROM ttm_financials"
            )
        )
        with_data = ttm_result.scalar()

    assert total == 10, f"Expected 10 companies, got {total}"
    assert with_data == 7, (
        f"Expected 7 tickers with TTM data, got {with_data}"
    )

    result = _validate_pipeline_coverage(total, with_data)
    coverage = result.coverage_percent  # type: ignore[attr-defined]

    assert abs(coverage - 70.0) < 0.1, (
        f"Expected ~70% coverage, got {coverage}%"
    )
    assert result.is_valid is True, (
        "70% coverage should pass validation (threshold is 50%)"
    )
    assert len(result.warnings) == 0, (
        f"70% coverage should not produce warnings: "
        f"{result.warnings}"
    )
