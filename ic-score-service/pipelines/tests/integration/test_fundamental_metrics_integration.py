"""Integration tests for FundamentalMetricsCalculator.

Requires a real PostgreSQL + TimescaleDB database.
Run with: INTEGRATION_TEST_DB=true pytest ... -v

The FundamentalMetricsCalculator reads from:
  - ttm_financials (TTM revenue, net_income, FCF, equity, etc.)
  - stock_prices (current price for EV calculations)
  - valuation_ratios (stock_price, pe_ratio, market_cap)
  - financials (annual 10-K data for growth rate calculations)
  - dividends (dividend history for yield, payout ratio)

It writes to: fundamental_metrics_extended
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
    seed_financials,
    seed_stock_prices,
    seed_ttm_financials,
)


# -------------------------------------------------------------------
# Fixtures
# -------------------------------------------------------------------


@pytest_asyncio.fixture
async def fm_calculator(db):
    """Create a FundamentalMetricsCalculator with test DB injected."""
    with patch(
        "pipelines.fundamental_metrics_calculator.get_database",
        return_value=db,
    ):
        from pipelines.fundamental_metrics_calculator import (
            FundamentalMetricsCalculator,
        )

        calc = FundamentalMetricsCalculator()
        yield calc


@pytest_asyncio.fixture
async def seeded_aapl(db):
    """Seed all dependencies for AAPL fundamental metrics:
    company, financials (for growth rates), ttm_financials,
    and stock_prices (for current price / market cap)."""
    async with db.session() as session:
        await seed_companies(session, tickers=["AAPL"])
        await seed_financials(session, "AAPL", quarters=8)
        await seed_ttm_financials(session, "AAPL")
        await seed_stock_prices(session, "AAPL", days=30)


# -------------------------------------------------------------------
# Tests
# -------------------------------------------------------------------


async def test_fundamental_metrics_end_to_end(
    db, seeded_aapl, fm_calculator
):
    """Seed AAPL with TTM financials and stock prices, run
    calculator, verify rows written to fundamental_metrics_extended.
    """
    success = await fm_calculator.process_ticker("AAPL")
    assert success is True, "process_ticker should return True"

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT COUNT(*) FROM fundamental_metrics_extended"
                " WHERE ticker = 'AAPL'"
            )
        )
        count = row.scalar()

    assert count >= 1, (
        f"Expected >=1 fundamental_metrics_extended rows, "
        f"got {count}"
    )


async def test_roe_calculated(db, seeded_aapl, fm_calculator):
    """Verify ROE = net_income / equity is reasonable.

    Seed data: net_income=$21B, equity=$90B -> ROE ~23.3%.
    ROE should be positive and in a realistic range.
    """
    success = await fm_calculator.process_ticker("AAPL")
    assert success is True

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT roe, roa"
                " FROM fundamental_metrics_extended"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        record = row.fetchone()

    assert record is not None, "No fundamental_metrics row for AAPL"
    roe, roa = record

    # Seed TTM data has positive net income and positive equity
    assert roe is not None, "ROE should be calculated"
    assert float(roe) > 0, f"ROE={roe} should be > 0"
    # Sanity: ROE should be between 1% and 200% for normal companies
    assert float(roe) < 200, f"ROE={roe}% seems unreasonably high"


async def test_profit_margins_valid(
    db, seeded_aapl, fm_calculator
):
    """Verify gross_margin, operating_margin, and net_margin are
    within the expected percentage range (-100% to 100%).

    Seed TTM data: revenue=$95B, gross_profit=$40B (~42%),
    operating_income=$27B (~28%), net_income=$21B (~22%).
    """
    await fm_calculator.process_ticker("AAPL")

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT gross_margin, operating_margin, net_margin"
                " FROM fundamental_metrics_extended"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        record = row.fetchone()

    assert record is not None
    gross_margin, operating_margin, net_margin = record

    # All margins should be populated given seed data
    assert gross_margin is not None, "gross_margin should be set"
    assert operating_margin is not None, (
        "operating_margin should be set"
    )
    assert net_margin is not None, "net_margin should be set"

    # Margins stored as percentages, should be between -100 and 100
    for name, value in [
        ("gross_margin", gross_margin),
        ("operating_margin", operating_margin),
        ("net_margin", net_margin),
    ]:
        v = float(value)
        assert -100 <= v <= 100, (
            f"{name}={v}% outside valid range [-100, 100]"
        )

    # Gross > operating > net for a profitable company
    assert float(gross_margin) > float(operating_margin), (
        "gross_margin should exceed operating_margin"
    )
    assert float(operating_margin) > float(net_margin), (
        "operating_margin should exceed net_margin"
    )


async def test_debt_ratios_calculated(
    db, seeded_aapl, fm_calculator
):
    """Verify debt_to_equity and net_debt_to_ebitda are calculated
    and non-negative when the company carries debt.

    Seed TTM data: short_term_debt=$5B, long_term_debt=$75B,
    equity=$90B -> D/E ~0.89, cash=$28B, OI=$27B ->
    net_debt/EBITDA ~1.93.
    """
    await fm_calculator.process_ticker("AAPL")

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT debt_to_equity, net_debt_to_ebitda"
                " FROM fundamental_metrics_extended"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        record = row.fetchone()

    assert record is not None
    debt_to_equity, net_debt_to_ebitda = record

    assert debt_to_equity is not None, (
        "debt_to_equity should be calculated"
    )
    assert float(debt_to_equity) >= 0, (
        f"debt_to_equity={debt_to_equity} should be >= 0"
    )

    assert net_debt_to_ebitda is not None, (
        "net_debt_to_ebitda should be calculated"
    )
    # Net debt = $80B - $28B = $52B; EBITDA proxy = $27B
    # -> ratio ~1.93, should be positive
    assert float(net_debt_to_ebitda) > 0, (
        f"net_debt_to_ebitda={net_debt_to_ebitda} should be > 0"
    )


async def test_skips_missing_ttm(db, fm_calculator):
    """A ticker with no ttm_financials should produce no output
    and process_ticker should return False."""
    async with db.session() as session:
        await seed_companies(session, tickers=["ZZZZ"])
        # Seed stock prices but NO ttm_financials
        await seed_stock_prices(session, "ZZZZ", days=30)

    result = await fm_calculator.process_ticker("ZZZZ")
    assert result is False, (
        "process_ticker should return False when TTM data missing"
    )

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT COUNT(*)"
                " FROM fundamental_metrics_extended"
                " WHERE ticker = 'ZZZZ'"
            )
        )
        count = row.scalar()
    assert count == 0, (
        "No rows should be written for ticker without TTM data"
    )


async def test_multi_ticker(db, fm_calculator):
    """Seed 3 tickers with TTM data and stock prices, run
    calculator for each, verify all have output rows."""
    tickers = ["AAPL", "JNJ", "AMZN"]

    async with db.session() as session:
        await seed_companies(session, tickers=tickers)
        for t in tickers:
            await seed_financials(session, t, quarters=8)
            await seed_ttm_financials(session, t)
            await seed_stock_prices(session, t, days=30)

    for t in tickers:
        result = await fm_calculator.process_ticker(t)
        assert result is True, (
            f"process_ticker({t}) should succeed"
        )

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT DISTINCT ticker"
                " FROM fundamental_metrics_extended"
                " ORDER BY ticker"
            )
        )
        found = [r[0] for r in row.fetchall()]

    assert set(found) == set(tickers), (
        f"Expected tickers {tickers}, found {found}"
    )


async def test_idempotent_rerun(db, seeded_aapl, fm_calculator):
    """Running the calculator twice should produce the same result
    via ON CONFLICT upsert, with no duplicate rows."""
    await fm_calculator.process_ticker("AAPL")

    async with db.session() as session:
        row1 = await session.execute(
            text(
                "SELECT roe, gross_margin, debt_to_equity,"
                " data_quality_score"
                " FROM fundamental_metrics_extended"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        first_run = row1.fetchone()

    # Second run
    await fm_calculator.process_ticker("AAPL")

    async with db.session() as session:
        row2 = await session.execute(
            text(
                "SELECT roe, gross_margin, debt_to_equity,"
                " data_quality_score"
                " FROM fundamental_metrics_extended"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        second_run = row2.fetchone()

        count_row = await session.execute(
            text(
                "SELECT COUNT(*)"
                " FROM fundamental_metrics_extended"
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


async def test_current_ratio_is_null(
    db, seeded_aapl, fm_calculator
):
    """current_ratio requires current_assets / current_liabilities
    which the calculator does not yet compute (returns None).
    Verify the field is NULL rather than an incorrect value.

    Note: seed TTM data has cash=$28B but the calculator sets
    current_ratio = None because it lacks current assets/liabilities
    breakdown.
    """
    await fm_calculator.process_ticker("AAPL")

    async with db.session() as session:
        row = await session.execute(
            text(
                "SELECT current_ratio, quick_ratio"
                " FROM fundamental_metrics_extended"
                " WHERE ticker = 'AAPL'"
                " ORDER BY calculation_date DESC LIMIT 1"
            )
        )
        record = row.fetchone()

    assert record is not None
    current_ratio, quick_ratio = record

    # The calculator explicitly sets these to None (line 893-894)
    assert current_ratio is None, (
        "current_ratio should be NULL (not yet implemented)"
    )
    assert quick_ratio is None, (
        "quick_ratio should be NULL (not yet implemented)"
    )
