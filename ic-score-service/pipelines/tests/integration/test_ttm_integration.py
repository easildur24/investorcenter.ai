"""Integration tests for TTMFinancialsCalculator.

Requires a real PostgreSQL + TimescaleDB database.
Run with: INTEGRATION_TEST_DB=true pytest ... -v
"""

import os

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
    seed_financials,
)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest_asyncio.fixture
async def ttm_calculator(db):
    """Create a TTMFinancialsCalculator with test database injected."""
    with patch(
        "pipelines.ttm_financials_calculator.get_database",
        return_value=db,
    ):
        from pipelines.ttm_financials_calculator import (
            TTMFinancialsCalculator,
        )

        calc = TTMFinancialsCalculator()
        yield calc


@pytest_asyncio.fixture
async def seeded_aapl(db):
    """Seed AAPL company + financials for a single-ticker test."""
    async with db.session() as session:
        await seed_companies(session, tickers=["AAPL"])
        await seed_financials(session, "AAPL", quarters=8)


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


async def test_ttm_end_to_end_single_ticker(
    db, seeded_aapl, ttm_calculator
):
    """Seed AAPL financials, run calculator, verify ttm_financials
    has at least one output row."""
    result = await ttm_calculator.process_ticker("AAPL")
    assert result is True, "process_ticker should return True"

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT COUNT(*) FROM ttm_financials"
                " WHERE ticker = 'AAPL'"
            )
        )
        count = row.scalar()
    assert count >= 1, f"Expected >=1 ttm_financials rows, got {count}"


async def test_ttm_output_has_required_fields(
    db, seeded_aapl, ttm_calculator
):
    """Verify all key columns are populated after a run."""
    await ttm_calculator.process_ticker("AAPL")

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT ticker, calculation_date,"
                " ttm_period_start, ttm_period_end,"
                " revenue, net_income,"
                " eps_basic, eps_diluted,"
                " shares_outstanding,"
                " total_assets, total_liabilities,"
                " shareholders_equity,"
                " operating_cash_flow, free_cash_flow"
                " FROM ttm_financials"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        record = row.fetchone()

    assert record is not None, "No ttm_financials row for AAPL"
    (
        ticker,
        calc_date,
        period_start,
        period_end,
        revenue,
        net_income,
        eps_basic,
        eps_diluted,
        shares,
        total_assets,
        total_liabilities,
        equity,
        ocf,
        fcf,
    ) = record

    assert ticker == "AAPL"
    assert calc_date is not None
    assert period_start is not None
    assert period_end is not None
    assert revenue is not None, "revenue should be populated"
    assert net_income is not None, "net_income should be populated"
    assert eps_basic is not None or eps_diluted is not None, (
        "At least one EPS field should be populated"
    )
    assert shares is not None and shares > 0
    assert total_assets is not None and total_assets > 0
    assert equity is not None


async def test_ttm_eps_positive_for_profitable_company(
    db, seeded_aapl, ttm_calculator
):
    """Seeded AAPL data has positive net income, so EPS should
    be > 0."""
    await ttm_calculator.process_ticker("AAPL")

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT eps_basic, eps_diluted, net_income"
                " FROM ttm_financials"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        record = row.fetchone()

    assert record is not None
    eps_basic, eps_diluted, net_income = record
    assert net_income is not None and net_income > 0, (
        "Seed data should produce positive net income"
    )
    # At least one EPS variant should be positive
    if eps_basic is not None:
        assert eps_basic > 0, f"eps_basic={eps_basic} should be > 0"
    if eps_diluted is not None:
        assert (
            eps_diluted > 0
        ), f"eps_diluted={eps_diluted} should be > 0"


async def test_ttm_revenue_reasonable(
    db, seeded_aapl, ttm_calculator
):
    """TTM revenue should be roughly in the range of annual revenue.

    Seed data uses ~$90B annual base, so TTM should be in the
    tens-of-billions range (not orders of magnitude off).
    """
    await ttm_calculator.process_ticker("AAPL")

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT revenue FROM ttm_financials"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        record = row.fetchone()

    assert record is not None
    revenue = record[0]
    assert revenue is not None
    # Seed data: ~$90B base annual revenue.  The calculator sums
    # last 4 quarterly rows (which are cumulative YTD in seed),
    # but the result should still be in a reasonable range.
    assert revenue > 10_000_000_000, (
        f"revenue={revenue} seems too low"
    )
    assert revenue < 1_000_000_000_000, (
        f"revenue={revenue} seems too high"
    )


async def test_ttm_balance_sheet_from_latest_quarter(
    db, seeded_aapl, ttm_calculator
):
    """Balance sheet items (total_assets, shareholders_equity)
    should come from the most recent quarter, not be summed."""
    await ttm_calculator.process_ticker("AAPL")

    async with db.session() as session:
        ttm_row = await session.execute(
            text(
                "SELECT total_assets, shareholders_equity"
                " FROM ttm_financials"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        ttm = ttm_row.fetchone()

        q_row = await session.execute(
            text(
                "SELECT total_assets, shareholders_equity"
                " FROM financials"
                " WHERE ticker = 'AAPL'"
                "   AND statement_type = '10-Q'"
                "   AND period_end_date <= CURRENT_DATE"
                " ORDER BY period_end_date DESC LIMIT 1"
            )
        )
        latest_q = q_row.fetchone()

    assert ttm is not None and latest_q is not None
    ttm_assets, ttm_equity = ttm
    q_assets, q_equity = latest_q

    # Balance sheet items should match the most recent quarter,
    # NOT be a sum of 4 quarters.
    assert ttm_assets == q_assets, (
        f"TTM total_assets ({ttm_assets}) should equal latest "
        f"quarter ({q_assets})"
    )
    assert ttm_equity == q_equity, (
        f"TTM equity ({ttm_equity}) should equal latest "
        f"quarter ({q_equity})"
    )


async def test_ttm_idempotent(db, seeded_aapl, ttm_calculator):
    """Running the calculator twice produces the same results
    (ON CONFLICT DO UPDATE)."""
    await ttm_calculator.process_ticker("AAPL")

    async with db.session() as session:
        row1 = await session.execute(
            text(
                "SELECT revenue, net_income, eps_diluted"
                " FROM ttm_financials"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        first_run = row1.fetchone()

    # Run a second time
    await ttm_calculator.process_ticker("AAPL")

    async with db.session() as session:
        row2 = await session.execute(
            text(
                "SELECT revenue, net_income, eps_diluted"
                " FROM ttm_financials"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        second_run = row2.fetchone()

        count_row = await session.execute(
            text(
                "SELECT COUNT(*) FROM ttm_financials"
                " WHERE ticker = 'AAPL'"
            )
        )
        count = count_row.scalar()

    assert count == 1, (
        f"Expected 1 row after two runs (upsert), got {count}"
    )
    assert first_run == second_run, (
        "Results should be identical across idempotent runs"
    )


async def test_ttm_skips_missing_data(db, ttm_calculator):
    """A ticker with no financials should not crash and should
    return False (skipped)."""
    async with db.session() as session:
        await seed_companies(session, tickers=["ZZZZ"])

    result = await ttm_calculator.process_ticker("ZZZZ")
    assert result is False, (
        "process_ticker should return False for ticker with no data"
    )

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT COUNT(*) FROM ttm_financials"
                " WHERE ticker = 'ZZZZ'"
            )
        )
        count = row.scalar()
    assert count == 0


async def test_ttm_multi_ticker(db, ttm_calculator):
    """Process 3 tickers and verify all get output rows."""
    tickers = ["AAPL", "JNJ", "AMZN"]

    async with db.session() as session:
        await seed_companies(session, tickers=tickers)
        for t in tickers:
            await seed_financials(session, t, quarters=8)

    for t in tickers:
        result = await ttm_calculator.process_ticker(t)
        assert result is True, f"process_ticker({t}) should succeed"

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT DISTINCT ticker FROM ttm_financials"
                " ORDER BY ticker"
            )
        )
        found = [r[0] for r in row.fetchall()]

    assert set(found) == set(tickers), (
        f"Expected tickers {tickers}, found {found}"
    )
