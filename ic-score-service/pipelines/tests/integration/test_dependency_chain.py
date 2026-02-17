"""Integration tests for pipeline dependency checker.

Requires a real PostgreSQL + TimescaleDB database.
Run with: INTEGRATION_TEST_DB=true pytest ... -v

Tests the dependency_checker module which validates that
upstream pipeline outputs are fresh before downstream
pipelines run.
"""

import os
from datetime import date, datetime, timedelta

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
    seed_benchmark_returns,
    seed_stock_prices,
)


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


async def test_fresh_data_is_ready(db, session):
    """Seed ttm_financials with today's date, then
    check_upstream_freshness('fundamental_metrics') should
    return is_ready=True."""
    await seed_companies(session, tickers=["AAPL"])
    await seed_ttm_financials(session, "AAPL")

    from pipelines.utils.dependency_checker import (
        check_upstream_freshness,
    )

    async with db.session() as s:
        result = await check_upstream_freshness(
            "fundamental_metrics", s, max_age_hours=48
        )

    assert result.is_ready is True, (
        f"Expected is_ready=True, got stale: "
        f"{result.stale_dependencies}"
    )


async def test_stale_data_not_ready(db, session):
    """Seed ttm_financials with a date 90 days ago, then
    check_upstream_freshness('fundamental_metrics',
    max_age_hours=48) should return is_ready=False."""
    await seed_companies(session, tickers=["AAPL"])

    # Insert TTM data with an old created_at timestamp
    old_date = date.today() - timedelta(days=90)
    old_timestamp = datetime.combine(
        old_date, datetime.min.time()
    )
    await session.execute(
        text(
            "INSERT INTO ttm_financials"
            " (ticker, calculation_date,"
            "  ttm_period_start, ttm_period_end,"
            "  revenue, net_income,"
            "  shares_outstanding,"
            "  quarters_included, created_at)"
            " VALUES"
            " (:ticker, :calc_date,"
            "  :start, :end,"
            "  :rev, :ni,"
            "  :shares,"
            "  :quarters, :created_at)"
            " ON CONFLICT (ticker, calculation_date) DO NOTHING"
        ),
        {
            "ticker": "AAPL",
            "calc_date": old_date,
            "start": old_date - timedelta(days=365),
            "end": old_date - timedelta(days=1),
            "rev": 95_000_000_000,
            "ni": 21_000_000_000,
            "shares": 15_500_000_000,
            "quarters": '["Q1","Q2","Q3","Q4"]',
            "created_at": old_timestamp,
        },
    )
    await session.commit()

    from pipelines.utils.dependency_checker import (
        check_upstream_freshness,
    )

    async with db.session() as s:
        result = await check_upstream_freshness(
            "fundamental_metrics", s, max_age_hours=48
        )

    assert result.is_ready is False, (
        "Expected is_ready=False for stale data"
    )
    assert "ttm_financials" in result.stale_dependencies, (
        f"Expected 'ttm_financials' in stale_dependencies, "
        f"got {result.stale_dependencies}"
    )


async def test_empty_table_not_ready(db, session):
    """No data in ttm_financials should cause
    check_upstream_freshness('fundamental_metrics') to return
    is_ready=False with stale_dependencies=['ttm_financials']."""
    from pipelines.utils.dependency_checker import (
        check_upstream_freshness,
    )

    async with db.session() as s:
        result = await check_upstream_freshness(
            "fundamental_metrics", s, max_age_hours=48
        )

    assert result.is_ready is False, (
        "Expected is_ready=False when table is empty"
    )
    assert "ttm_financials" in result.stale_dependencies, (
        f"Expected 'ttm_financials' in stale_dependencies, "
        f"got {result.stale_dependencies}"
    )


async def test_ingestion_always_ready(db, session):
    """Ingestion pipelines have no upstream dependencies, so
    check_upstream_freshness('benchmark_data') should always
    return is_ready=True."""
    from pipelines.utils.dependency_checker import (
        check_upstream_freshness,
    )

    async with db.session() as s:
        result = await check_upstream_freshness(
            "benchmark_data", s, max_age_hours=48
        )

    assert result.is_ready is True, (
        f"Ingestion pipeline should always be ready, "
        f"got stale: {result.stale_dependencies}"
    )
    assert result.stale_dependencies == [], (
        "Ingestion pipeline should have no stale dependencies"
    )


async def test_ic_score_needs_all_upstream(db, session):
    """ic_score_calculator depends on 6 upstream pipelines.
    With only some seeded, check should report the missing
    ones as stale."""
    # Seed only fundamental_metrics_extended (satisfies
    # fundamental_metrics + sector_percentiles + fair_value
    # since they all write to the same table)
    await seed_companies(session, tickers=["AAPL"])

    # Insert a fresh fundamental_metrics_extended row
    today = date.today()
    await session.execute(
        text(
            "INSERT INTO fundamental_metrics_extended"
            " (ticker, calculation_date, revenue_growth_yoy)"
            " VALUES (:t, :d, :g)"
            " ON CONFLICT (ticker, calculation_date) DO NOTHING"
        ),
        {"t": "AAPL", "d": today, "g": 10.0},
    )
    await session.commit()

    # Do NOT seed: technical_indicators, valuation_ratios,
    # risk_metrics -> those should appear as stale

    from pipelines.utils.dependency_checker import (
        check_upstream_freshness,
    )

    async with db.session() as s:
        result = await check_upstream_freshness(
            "ic_score_calculator", s, max_age_hours=48
        )

    assert result.is_ready is False, (
        "ic_score_calculator should not be ready with "
        "incomplete upstream data"
    )
    # technical_indicators, valuation_ratios, and risk_metrics
    # should be stale (their tables are empty)
    for expected_stale in [
        "technical_indicators",
        "valuation_ratios",
        "risk_metrics",
    ]:
        assert expected_stale in result.stale_dependencies, (
            f"Expected '{expected_stale}' in "
            f"stale_dependencies: {result.stale_dependencies}"
        )
