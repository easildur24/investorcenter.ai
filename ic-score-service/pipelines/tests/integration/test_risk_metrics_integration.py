"""Integration tests for RiskMetricsCalculator.

Requires a real PostgreSQL + TimescaleDB database.
Run with: INTEGRATION_TEST_DB=true pytest ... -v

The RiskMetricsCalculator reads from:
  - stock_prices (daily close prices, requires interval='1day')
  - benchmark_returns (SPY close + daily_return)
  - treasury_rates (1-month rate for risk-free rate)

It writes to: risk_metrics (TimescaleDB hypertable)
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
    seed_stock_prices,
    seed_benchmark_returns,
    seed_treasury_rates,
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


async def _seed_risk_prerequisites(session, ticker="AAPL", days=300):
    """Seed all data needed for risk metrics calculation."""
    await seed_companies(session, tickers=[ticker])
    await seed_stock_prices(session, ticker, days=days)
    await seed_benchmark_returns(session, days=days)
    await seed_treasury_rates(session, days=days)


async def _run_risk_calculator(db, ticker="AAPL"):
    """Run the risk metrics calculator for a single ticker.

    Patches get_database so the calculator uses the test DB.
    """
    with patch(
        "pipelines.risk_metrics_calculator.get_database",
        return_value=db,
    ):
        from pipelines.risk_metrics_calculator import (
            RiskMetricsCalculator,
        )

        calculator = RiskMetricsCalculator()
        calculator.db = db
        async with db.session() as session:
            await calculator.process_ticker(ticker, session)


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


async def test_risk_metrics_end_to_end(db, session):
    """Seed 300 days of prices + benchmark + treasury, run
    calculator, verify risk_metrics has output for AAPL."""
    await _seed_risk_prerequisites(session, "AAPL", days=300)
    await _run_risk_calculator(db, "AAPL")

    async with db.session() as s:
        result = await s.execute(
            text(
                "SELECT COUNT(*) FROM risk_metrics"
                " WHERE ticker = 'AAPL'"
            )
        )
        count = result.scalar()

    assert count >= 1, (
        f"Expected at least 1 risk_metrics row for AAPL, got {count}"
    )


async def test_beta_in_valid_range(db, session):
    """Beta should be in a reasonable range (-2 to 5)."""
    await _seed_risk_prerequisites(session, "AAPL", days=300)
    await _run_risk_calculator(db, "AAPL")

    async with db.session() as s:
        result = await s.execute(
            text(
                "SELECT beta FROM risk_metrics"
                " WHERE ticker = 'AAPL'"
                " AND beta IS NOT NULL"
            )
        )
        rows = result.fetchall()

    assert len(rows) > 0, "No risk_metrics rows with beta found"
    for row in rows:
        beta = float(row[0])
        assert -2.0 <= beta <= 5.0, (
            f"Beta {beta} outside expected range [-2, 5]"
        )


async def test_sharpe_ratio_calculated(db, session):
    """Sharpe ratio should be non-None for a valid run."""
    await _seed_risk_prerequisites(session, "AAPL", days=300)
    await _run_risk_calculator(db, "AAPL")

    async with db.session() as s:
        result = await s.execute(
            text(
                "SELECT sharpe_ratio FROM risk_metrics"
                " WHERE ticker = 'AAPL'"
                " AND period = '1Y'"
            )
        )
        row = result.fetchone()

    assert row is not None, "No 1Y risk_metrics row for AAPL"
    assert row[0] is not None, "sharpe_ratio is None"


async def test_max_drawdown_negative(db, session):
    """Maximum drawdown should be <= 0 (it represents a loss)."""
    await _seed_risk_prerequisites(session, "AAPL", days=300)
    await _run_risk_calculator(db, "AAPL")

    async with db.session() as s:
        result = await s.execute(
            text(
                "SELECT max_drawdown FROM risk_metrics"
                " WHERE ticker = 'AAPL'"
                " AND max_drawdown IS NOT NULL"
            )
        )
        rows = result.fetchall()

    assert len(rows) > 0, "No risk_metrics with max_drawdown"
    for row in rows:
        mdd = float(row[0])
        assert mdd <= 0, f"max_drawdown {mdd} should be <= 0"


async def test_requires_min_price_history(db, session):
    """Ticker with < 252 trading days of prices should produce
    no 1Y output (calculator requires 70% of period's trading
    days)."""
    # Seed only 100 calendar days (~70 trading days), well below
    # the 252 trading-day 1Y requirement.
    await _seed_risk_prerequisites(session, "JPM", days=100)
    await _run_risk_calculator(db, "JPM")

    async with db.session() as s:
        result = await s.execute(
            text(
                "SELECT COUNT(*) FROM risk_metrics"
                " WHERE ticker = 'JPM' AND period = '1Y'"
            )
        )
        count = result.scalar()

    assert count == 0, (
        f"Expected 0 risk_metrics for JPM with short history, "
        f"got {count}"
    )


async def test_multi_period(db, session):
    """With enough data, verify 1Y results exist. 3Y requires
    ~756 trading days; with 300 calendar days we only get 1Y."""
    await _seed_risk_prerequisites(session, "AAPL", days=300)
    await _run_risk_calculator(db, "AAPL")

    async with db.session() as s:
        result = await s.execute(
            text(
                "SELECT DISTINCT period FROM risk_metrics"
                " WHERE ticker = 'AAPL'"
                " ORDER BY period"
            )
        )
        periods = [row[0] for row in result.fetchall()]

    assert "1Y" in periods, (
        f"Expected '1Y' in periods, got {periods}"
    )
    # 300 calendar days is not enough for 3Y
    assert "3Y" not in periods, (
        "Did not expect '3Y' with only 300 days of data"
    )
