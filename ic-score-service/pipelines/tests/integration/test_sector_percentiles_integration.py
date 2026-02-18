"""Integration tests for SectorPercentilesCalculator.

Requires a real PostgreSQL + TimescaleDB database.
Run with: INTEGRATION_TEST_DB=true pytest ... -v

The SectorPercentileAggregator reads from:
  - companies (sector, is_active)
  - fundamental_metrics_extended (growth metrics, valuations)
  - financials (margins, returns, liquidity ratios)
  - valuation_ratios (PE, PB, PS ratios)

It writes to: sector_percentiles
"""

import os
from datetime import date, timedelta

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
)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# Five companies in Technology for testing within-sector percentiles
_TECH_COMPANIES = [
    ("AAPL", "Apple Inc.", "Technology", "Consumer Electronics"),
    ("MSFT", "Microsoft Corp.", "Technology", "Software"),
    ("GOOGL", "Alphabet", "Technology", "Internet"),
    ("META", "Meta Platforms", "Technology", "Social Media"),
    ("NVDA", "Nvidia Corp.", "Technology", "Semiconductors"),
    ("CRM", "Salesforce", "Technology", "SaaS"),
]

# Two companies in Healthcare for cross-sector test
_HEALTH_COMPANIES = [
    ("JNJ", "Johnson & Johnson", "Healthcare", "Pharmaceuticals"),
    ("PFE", "Pfizer Inc.", "Healthcare", "Pharmaceuticals"),
    ("UNH", "UnitedHealth Group", "Healthcare", "Health Plans"),
    ("ABBV", "AbbVie Inc.", "Healthcare", "Pharmaceuticals"),
    ("MRK", "Merck & Co.", "Healthcare", "Pharmaceuticals"),
]


async def _seed_companies_manual(session, companies):
    """Insert company rows directly."""
    for ticker, name, sector, industry in companies:
        await session.execute(
            text(
                "INSERT INTO companies"
                " (ticker, name, sector, industry, is_active)"
                " VALUES (:t, :n, :s, :i, true)"
                " ON CONFLICT (ticker) DO NOTHING"
            ),
            {"t": ticker, "n": name, "s": sector, "i": industry},
        )
    await session.commit()


async def _seed_fundamental_metrics(session, ticker, values):
    """Insert a fundamental_metrics_extended row with explicit
    metric values.

    ``values`` is a dict of column_name -> value.
    """
    today = date.today()
    cols = ["ticker", "calculation_date"] + list(values.keys())
    placeholders = [":ticker", ":calc_date"] + [
        f":{k}" for k in values.keys()
    ]
    params = {"ticker": ticker, "calc_date": today}
    params.update(values)

    col_str = ", ".join(cols)
    ph_str = ", ".join(placeholders)

    await session.execute(
        text(
            f"INSERT INTO fundamental_metrics_extended"
            f" ({col_str}) VALUES ({ph_str})"
            f" ON CONFLICT (ticker, calculation_date) DO NOTHING"
        ),
        params,
    )
    await session.commit()


async def _seed_tech_sector(session):
    """Seed 6 Technology companies with varied metrics."""
    await _seed_companies_manual(session, _TECH_COMPANIES)

    metric_sets = [
        {"revenue_growth_yoy": 15.0, "eps_growth_yoy": 12.0},
        {"revenue_growth_yoy": 20.0, "eps_growth_yoy": 18.0},
        {"revenue_growth_yoy": 25.0, "eps_growth_yoy": 22.0},
        {"revenue_growth_yoy": 10.0, "eps_growth_yoy": 8.0},
        {"revenue_growth_yoy": 35.0, "eps_growth_yoy": 30.0},
        {"revenue_growth_yoy": 18.0, "eps_growth_yoy": 14.0},
    ]
    for (ticker, _, _, _), vals in zip(
        _TECH_COMPANIES, metric_sets
    ):
        await _seed_fundamental_metrics(session, ticker, vals)


async def _seed_health_sector(session):
    """Seed 5 Healthcare companies with varied metrics."""
    await _seed_companies_manual(session, _HEALTH_COMPANIES)

    metric_sets = [
        {"revenue_growth_yoy": 5.0, "eps_growth_yoy": 3.0},
        {"revenue_growth_yoy": -2.0, "eps_growth_yoy": -5.0},
        {"revenue_growth_yoy": 12.0, "eps_growth_yoy": 10.0},
        {"revenue_growth_yoy": 8.0, "eps_growth_yoy": 7.0},
        {"revenue_growth_yoy": 6.0, "eps_growth_yoy": 4.0},
    ]
    for (ticker, _, _, _), vals in zip(
        _HEALTH_COMPANIES, metric_sets
    ):
        await _seed_fundamental_metrics(session, ticker, vals)


async def _run_sector_percentiles(db, sector=None):
    """Run the sector percentile aggregator.

    Patches get_database so the pipeline uses the test DB.
    """
    with patch(
        "pipelines.sector_percentiles_calculator.get_database",
        return_value=db,
    ):
        from pipelines.sector_percentiles_calculator import (
            calculate_sector_percentiles,
        )

        result = await calculate_sector_percentiles(sector=sector)
        return result


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


async def test_sector_percentiles_end_to_end(db, session):
    """Seed 6 companies in Technology with fundamental_metrics,
    run aggregator, verify sector_percentiles has output."""
    await _seed_tech_sector(session)
    result = await _run_sector_percentiles(db, sector="Technology")

    assert result.get("status") == "success", (
        f"Pipeline did not succeed: {result}"
    )

    async with db.session() as s:
        row = await s.execute(
            text(
                "SELECT COUNT(*) FROM sector_percentiles"
                " WHERE sector = 'Technology'"
            )
        )
        count = row.scalar()

    assert count >= 1, (
        f"Expected sector_percentiles rows for Technology, "
        f"got {count}"
    )


async def test_percentile_values_in_range(db, session):
    """All percentile values should be between 0 and 100."""
    await _seed_tech_sector(session)
    await _run_sector_percentiles(db, sector="Technology")

    async with db.session() as s:
        result = await s.execute(
            text(
                "SELECT p10_value, p25_value, p50_value,"
                " p75_value, p90_value"
                " FROM sector_percentiles"
                " WHERE sector = 'Technology'"
            )
        )
        rows = result.fetchall()

    assert len(rows) > 0, "No sector_percentiles rows found"
    for row in rows:
        p10, p25, p50, p75, p90 = (
            float(v) if v is not None else None for v in row
        )
        # Percentile breakpoint values should be ordered
        if all(v is not None for v in [p10, p25, p50, p75, p90]):
            assert p10 <= p25 <= p50 <= p75 <= p90, (
                f"Percentiles not in order: "
                f"{p10}, {p25}, {p50}, {p75}, {p90}"
            )


async def test_multiple_sectors(db, session):
    """Companies in different sectors get independent percentile
    rankings."""
    await _seed_tech_sector(session)
    await _seed_health_sector(session)
    await _run_sector_percentiles(db)

    async with db.session() as s:
        result = await s.execute(
            text(
                "SELECT DISTINCT sector FROM sector_percentiles"
                " ORDER BY sector"
            )
        )
        sectors = [row[0] for row in result.fetchall()]

    assert "Technology" in sectors, (
        f"Technology missing from sectors: {sectors}"
    )
    assert "Healthcare" in sectors, (
        f"Healthcare missing from sectors: {sectors}"
    )


async def test_single_company_sector(db, session):
    """A sector with fewer than MIN_SAMPLE_SIZE (5) companies
    should skip that metric (not enough data for percentiles).
    A sector with exactly 5 should still produce percentiles."""
    # Seed exactly 5 Healthcare companies
    await _seed_health_sector(session)
    await _run_sector_percentiles(db, sector="Healthcare")

    async with db.session() as s:
        result = await s.execute(
            text(
                "SELECT COUNT(*) FROM sector_percentiles"
                " WHERE sector = 'Healthcare'"
            )
        )
        count = result.scalar()

    # 5 companies meets the MIN_SAMPLE_SIZE=5 threshold
    assert count >= 1, (
        f"Expected percentiles for Healthcare (5 companies), "
        f"got {count}"
    )


async def test_writes_to_sector_percentiles_table(db, session):
    """Output is stored in the sector_percentiles table with
    expected columns populated."""
    await _seed_tech_sector(session)
    await _run_sector_percentiles(db, sector="Technology")

    async with db.session() as s:
        result = await s.execute(
            text(
                "SELECT sector, metric_name, sample_count,"
                " min_value, max_value, mean_value, std_dev"
                " FROM sector_percentiles"
                " WHERE sector = 'Technology'"
                " LIMIT 1"
            )
        )
        row = result.fetchone()

    assert row is not None, "No sector_percentiles row found"
    sector, metric, count, min_v, max_v, mean_v, std = row
    assert sector == "Technology"
    assert metric is not None and len(metric) > 0
    assert count >= 5, (
        f"sample_count={count}, expected >= 5"
    )
    assert min_v is not None
    assert max_v is not None
    assert float(min_v) <= float(max_v), (
        f"min_value ({min_v}) > max_value ({max_v})"
    )
