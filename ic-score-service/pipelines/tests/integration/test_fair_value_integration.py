"""Integration tests for FairValueCalculator.

Requires a real PostgreSQL + TimescaleDB database.
Run with: INTEGRATION_TEST_DB=true pytest ... -v

The FairValueCalculator reads from:
  - ttm_financials (TTM revenue, net_income, FCF, equity, etc.)
  - valuation_ratios (stock_price, ttm_market_cap)
  - risk_metrics (beta for WACC)
  - treasury_rates (10Y rate for risk-free rate)
  - fundamental_metrics_extended (growth rates for DCF; also the
    UPDATE target for fair-value columns)

It writes (UPDATE) to: fundamental_metrics_extended
"""

import os
from datetime import date

import pytest
import pytest_asyncio
from unittest.mock import patch

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
    seed_risk_metrics,
    seed_stock_prices,
    seed_treasury_rates,
    seed_ttm_financials,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


async def _seed_valuation_ratios(session, ticker):
    """Insert a valuation_ratios row so fair value can read
    stock_price and ttm_market_cap."""
    today = date.today()
    await session.execute(
        text(
            "INSERT INTO valuation_ratios"
            " (ticker, calculation_date, stock_price,"
            "  ttm_market_cap)"
            " VALUES (:t, :d, :price, :mcap)"
            " ON CONFLICT (ticker, calculation_date) DO NOTHING"
        ),
        {
            "t": ticker,
            "d": today,
            "price": 185.00,
            "mcap": 2_850_000_000_000,  # ~$2.85T
        },
    )
    await session.commit()


async def _seed_fundamental_metrics_row(session, ticker):
    """Insert a skeleton fundamental_metrics_extended row so the
    calculator's UPDATE has a target row.

    The fair value calculator runs UPDATE ... SET dcf_fair_value = ...
    WHERE ticker = :ticker AND calculation_date = :date, so a row
    must already exist (created by the fundamental_metrics pipeline
    that runs before fair_value in production).
    """
    today = date.today()
    await session.execute(
        text(
            "INSERT INTO fundamental_metrics_extended"
            " (ticker, calculation_date,"
            "  revenue_growth_yoy,"
            "  revenue_growth_3y_cagr,"
            "  revenue_growth_5y_cagr)"
            " VALUES (:t, :d, :g1, :g3, :g5)"
            " ON CONFLICT (ticker, calculation_date) DO NOTHING"
        ),
        {
            "t": ticker,
            "d": today,
            "g1": 8.5,
            "g3": 10.2,
            "g5": 12.0,
        },
    )
    await session.commit()


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest_asyncio.fixture
async def fair_value_calculator(db):
    """Create a FairValueCalculator with test database injected."""
    with patch(
        "pipelines.fair_value_calculator.get_database",
        return_value=db,
    ):
        from pipelines.fair_value_calculator import (
            FairValueCalculator,
        )

        calc = FairValueCalculator()
        yield calc


@pytest_asyncio.fixture
async def seeded_aapl(db):
    """Seed all dependencies for AAPL fair value calculation:
    company, ttm_financials, risk_metrics, treasury_rates,
    valuation_ratios, and a skeleton fundamental_metrics_extended
    row."""
    async with db.session() as session:
        await seed_companies(session, tickers=["AAPL"])
        await seed_ttm_financials(session, "AAPL")
        await seed_risk_metrics(session, "AAPL")
        await seed_treasury_rates(session, days=30)
        await seed_stock_prices(session, "AAPL", days=30)
        await _seed_valuation_ratios(session, "AAPL")
        await _seed_fundamental_metrics_row(session, "AAPL")


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


async def test_fair_value_end_to_end(
    db, seeded_aapl, fair_value_calculator
):
    """Seed all dependencies, run calculator, verify
    dcf_fair_value is written."""
    # calculate_fair_values returns a dict (not DB-dependent)
    result = await fair_value_calculator.calculate_fair_values("AAPL")

    assert result is not None, (
        "calculate_fair_values should return a dict"
    )
    assert result["ticker"] == "AAPL"
    assert result.get("dcf_fair_value") is not None, (
        "dcf_fair_value should be calculated"
    )

    # Also write to DB via process_ticker and verify
    success = await fair_value_calculator.process_ticker("AAPL")
    assert success is True, "process_ticker should return True"

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT dcf_fair_value, graham_number,"
                " epv_fair_value, wacc"
                " FROM fundamental_metrics_extended"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        record = row.fetchone()

    assert record is not None, (
        "fundamental_metrics_extended should have a row for AAPL"
    )
    dcf, graham, epv, wacc = record
    # At least DCF should be written (seed data has positive FCF)
    assert dcf is not None, "dcf_fair_value should be written to DB"


async def test_graham_number_calculated(
    db, seeded_aapl, fair_value_calculator
):
    """Verify graham_number > 0 when EPS and book value are
    positive."""
    result = await fair_value_calculator.calculate_fair_values("AAPL")

    assert result is not None
    graham = result.get("graham_number")
    # Seed data: EPS ~$1.33, equity ~$90B / 15.5B shares ~ $5.8 BVPS
    # Graham = sqrt(22.5 * 1.33 * 5.8) ~ $13.15
    assert graham is not None, "graham_number should be calculated"
    assert float(graham) > 0, (
        f"graham_number={graham} should be > 0"
    )


async def test_wacc_in_valid_range(
    db, seeded_aapl, fair_value_calculator
):
    """WACC should be bounded between 5% and 20% (stored as
    percentage)."""
    result = await fair_value_calculator.calculate_fair_values("AAPL")

    assert result is not None
    wacc = result.get("wacc")
    assert wacc is not None, "wacc should be calculated"
    wacc_float = float(wacc)
    assert 5.0 <= wacc_float <= 20.0, (
        f"wacc={wacc_float}% should be between 5% and 20%"
    )


async def test_fair_value_skips_missing_ttm(
    db, fair_value_calculator
):
    """A ticker without TTM data should be skipped gracefully."""
    # Seed only the company, no TTM financials
    async with db.session() as session:
        await seed_companies(session, tickers=["ZZZZ"])

    result = await fair_value_calculator.calculate_fair_values("ZZZZ")
    assert result is None, (
        "calculate_fair_values should return None "
        "when TTM data is missing"
    )


async def test_dcf_positive(db, seeded_aapl, fair_value_calculator):
    """dcf_fair_value > 0 when FCF > 0 (seed data has positive
    FCF of $19B)."""
    result = await fair_value_calculator.calculate_fair_values("AAPL")

    assert result is not None
    dcf = result.get("dcf_fair_value")
    assert dcf is not None, "dcf_fair_value should be calculated"
    assert float(dcf) > 0, f"dcf_fair_value={dcf} should be > 0"


async def test_fair_value_writes_to_fundamental_metrics(
    db, seeded_aapl, fair_value_calculator
):
    """Verify that process_ticker writes fair value columns into
    the fundamental_metrics_extended table."""
    success = await fair_value_calculator.process_ticker("AAPL")
    assert success is True

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT dcf_fair_value, graham_number,"
                " epv_fair_value, wacc, beta,"
                " cost_of_equity, cost_of_debt"
                " FROM fundamental_metrics_extended"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        record = row.fetchone()

    assert record is not None
    (
        dcf,
        graham,
        epv,
        wacc,
        beta,
        cost_of_equity,
        cost_of_debt,
    ) = record

    # DCF should be populated (positive FCF in seed data)
    assert dcf is not None and float(dcf) > 0
    # WACC components should be populated
    assert wacc is not None and float(wacc) > 0
    assert beta is not None
    assert cost_of_equity is not None and float(cost_of_equity) > 0
    assert cost_of_debt is not None and float(cost_of_debt) > 0
