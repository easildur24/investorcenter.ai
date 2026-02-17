"""Integration tests for IC Score Service database layer.

These tests require a real PostgreSQL database. They are skipped when
INTEGRATION_TEST_DB is not set (e.g., local dev without a database).
In CI, GitHub Actions provides a PostgreSQL service container.
"""

import os
import sys
from datetime import date, datetime
from decimal import Decimal
from pathlib import Path

import pytest
import pytest_asyncio

# Skip entire module if no test DB configured
pytestmark = pytest.mark.skipif(
    os.getenv("INTEGRATION_TEST_DB") != "true",
    reason="INTEGRATION_TEST_DB not set",
)

# Ensure the ic-score-service root is on sys.path
sys.path.insert(0, str(Path(__file__).parent.parent))

from sqlalchemy import text  # noqa: E402

from database.database import Database, DatabaseConfig  # noqa: E402


@pytest_asyncio.fixture(scope="module")
async def db():
    """Create Database instance and all tables once per module."""
    config = DatabaseConfig()
    database = Database(config)
    await database.create_all_tables()
    yield database
    await database.drop_all_tables()
    await database.close()


@pytest_asyncio.fixture(autouse=True)
async def clean_tables(db):
    """Truncate relevant tables before each test."""
    async with db.engine.begin() as conn:
        # Truncate tables in an order that respects FKs (CASCADE)
        await conn.execute(
            text(
                "TRUNCATE ic_scores, companies CASCADE"
            )
        )
    yield


@pytest_asyncio.fixture
async def session(db):
    """Provide a database session for test use."""
    async with db.session_factory() as sess:
        yield sess


# ====================
# Company Tests
# ====================


@pytest.mark.asyncio
async def test_company_insert_and_query(session):
    """Test basic INSERT and SELECT on companies table."""
    await session.execute(
        text(
            "INSERT INTO companies (ticker, name, sector, industry)"
            " VALUES (:t, :n, :s, :i)"
        ),
        {
            "t": "AAPL",
            "n": "Apple Inc.",
            "s": "Technology",
            "i": "Consumer Electronics",
        },
    )
    await session.commit()

    result = await session.execute(
        text("SELECT * FROM companies WHERE ticker = :t"),
        {"t": "AAPL"},
    )
    row = result.fetchone()

    assert row is not None
    data = row._asdict()
    assert data["ticker"] == "AAPL"
    assert data["name"] == "Apple Inc."
    assert data["sector"] == "Technology"
    assert data["industry"] == "Consumer Electronics"


@pytest.mark.asyncio
async def test_company_unique_constraint(session):
    """Test that duplicate tickers are rejected."""
    await session.execute(
        text(
            "INSERT INTO companies (ticker, name, sector)"
            " VALUES (:t, :n, :s)"
        ),
        {"t": "MSFT", "n": "Microsoft", "s": "Technology"},
    )
    await session.commit()

    with pytest.raises(Exception):
        await session.execute(
            text(
                "INSERT INTO companies (ticker, name, sector)"
                " VALUES (:t, :n, :s)"
            ),
            {"t": "MSFT", "n": "Microsoft Dup", "s": "Technology"},
        )
        await session.commit()


@pytest.mark.asyncio
async def test_company_null_optional_fields(session):
    """Test that optional fields accept NULL."""
    await session.execute(
        text(
            "INSERT INTO companies (ticker, name)"
            " VALUES (:t, :n)"
        ),
        {"t": "RARE", "n": "Rare Co"},
    )
    await session.commit()

    result = await session.execute(
        text("SELECT sector, industry FROM companies"
             " WHERE ticker = :t"),
        {"t": "RARE"},
    )
    row = result.fetchone()
    assert row is not None
    data = row._asdict()
    assert data["sector"] is None
    assert data["industry"] is None


# ====================
# IC Score Tests
# ====================


@pytest.mark.asyncio
async def test_ic_score_insert_and_latest_query(session):
    """Test IC Score insert and the 'latest score' query pattern."""
    # Seed a company first
    await session.execute(
        text(
            "INSERT INTO companies (ticker, name, sector)"
            " VALUES (:t, :n, :s)"
        ),
        {"t": "AAPL", "n": "Apple Inc.", "s": "Technology"},
    )

    # Insert an IC Score
    await session.execute(
        text("""
            INSERT INTO ic_scores
                (ticker, date, overall_score, rating,
                 value_score, growth_score, profitability_score,
                 confidence_level, data_completeness)
            VALUES
                (:ticker, :date, :score, :rating,
                 70, 80, 85,
                 'High', 0.95)
        """),
        {
            "ticker": "AAPL",
            "date": date(2025, 1, 15),
            "score": 75.0,
            "rating": "Buy",
        },
    )
    await session.commit()

    # Query using the same pattern as api/main.py
    result = await session.execute(
        text("""
            SELECT * FROM ic_scores
            WHERE ticker = :ticker
            ORDER BY date DESC LIMIT 1
        """),
        {"ticker": "AAPL"},
    )
    row = result.fetchone()

    assert row is not None
    data = row._asdict()
    assert data["ticker"] == "AAPL"
    assert float(data["overall_score"]) == 75.0
    assert data["rating"] == "Buy"
    assert data["confidence_level"] == "High"


@pytest.mark.asyncio
async def test_ic_score_multiple_dates(session):
    """Test multiple scores per ticker, verify latest-first."""
    await session.execute(
        text(
            "INSERT INTO companies (ticker, name, sector)"
            " VALUES (:t, :n, :s)"
        ),
        {"t": "AAPL", "n": "Apple", "s": "Technology"},
    )

    # Insert scores for multiple dates
    for day, score in [
        (date(2025, 1, 13), 72.0),
        (date(2025, 1, 14), 74.0),
        (date(2025, 1, 15), 76.0),
    ]:
        await session.execute(
            text("""
                INSERT INTO ic_scores
                    (ticker, date, overall_score, rating,
                     confidence_level, data_completeness)
                VALUES (:t, :d, :s, 'Buy', 'High', 0.9)
            """),
            {"t": "AAPL", "d": day, "s": score},
        )
    await session.commit()

    # Latest first
    result = await session.execute(
        text("""
            SELECT * FROM ic_scores
            WHERE ticker = :t
            ORDER BY date DESC
        """),
        {"t": "AAPL"},
    )
    rows = result.fetchall()

    assert len(rows) == 3
    # First row should be latest
    assert rows[0]._asdict()["date"] == date(2025, 1, 15)
    assert float(rows[0]._asdict()["overall_score"]) == 76.0


@pytest.mark.asyncio
async def test_ic_score_history_date_range(session):
    """Test date range filtering (used by history endpoint)."""
    await session.execute(
        text(
            "INSERT INTO companies (ticker, name)"
            " VALUES (:t, :n)"
        ),
        {"t": "MSFT", "n": "Microsoft"},
    )

    # Insert 5 days of scores
    for i in range(5):
        await session.execute(
            text("""
                INSERT INTO ic_scores
                    (ticker, date, overall_score, rating,
                     confidence_level, data_completeness)
                VALUES (:t, :d, :s, 'Hold', 'Medium', 0.8)
            """),
            {
                "t": "MSFT",
                "d": date(2025, 1, 11 + i),
                "s": 60.0 + i,
            },
        )
    await session.commit()

    # Query date range Jan 12-14 (3 days)
    result = await session.execute(
        text("""
            SELECT * FROM ic_scores
            WHERE ticker = :t
              AND date >= :start AND date <= :end
            ORDER BY date ASC
        """),
        {
            "t": "MSFT",
            "start": date(2025, 1, 12),
            "end": date(2025, 1, 14),
        },
    )
    rows = result.fetchall()

    assert len(rows) == 3
    assert rows[0]._asdict()["date"] == date(2025, 1, 12)
    assert rows[2]._asdict()["date"] == date(2025, 1, 14)


@pytest.mark.asyncio
async def test_ic_score_top_distinct_on(session):
    """Test DISTINCT ON pattern used by top-scores endpoint."""
    # Insert companies
    for ticker, name in [
        ("AAPL", "Apple"),
        ("MSFT", "Microsoft"),
        ("GOOGL", "Alphabet"),
    ]:
        await session.execute(
            text(
                "INSERT INTO companies (ticker, name)"
                " VALUES (:t, :n)"
            ),
            {"t": ticker, "n": name},
        )

    # Insert multiple days of scores for each
    for ticker, scores in [
        ("AAPL", [(date(2025, 1, 14), 80.0), (date(2025, 1, 15), 85.0)]),
        ("MSFT", [(date(2025, 1, 14), 75.0), (date(2025, 1, 15), 78.0)]),
        ("GOOGL", [(date(2025, 1, 14), 70.0), (date(2025, 1, 15), 72.0)]),
    ]:
        for d, s in scores:
            await session.execute(
                text("""
                    INSERT INTO ic_scores
                        (ticker, date, overall_score, rating,
                         confidence_level, data_completeness)
                    VALUES (:t, :d, :s, 'Buy', 'High', 0.9)
                """),
                {"t": ticker, "d": d, "s": s},
            )
    await session.commit()

    # DISTINCT ON: get latest score per ticker, sorted by score DESC
    result = await session.execute(
        text("""
            WITH latest_scores AS (
                SELECT DISTINCT ON (ticker) *
                FROM ic_scores
                ORDER BY ticker, date DESC
            )
            SELECT * FROM latest_scores
            ORDER BY overall_score DESC
        """)
    )
    rows = result.fetchall()

    assert len(rows) == 3
    # AAPL has highest score (85), then MSFT (78), then GOOGL (72)
    assert rows[0]._asdict()["ticker"] == "AAPL"
    assert float(rows[0]._asdict()["overall_score"]) == 85.0
    assert rows[1]._asdict()["ticker"] == "MSFT"
    assert rows[2]._asdict()["ticker"] == "GOOGL"


@pytest.mark.asyncio
async def test_ic_score_unique_ticker_date(session):
    """Test unique constraint on (ticker, date)."""
    await session.execute(
        text(
            "INSERT INTO companies (ticker, name)"
            " VALUES (:t, :n)"
        ),
        {"t": "AAPL", "n": "Apple"},
    )

    await session.execute(
        text("""
            INSERT INTO ic_scores
                (ticker, date, overall_score, rating,
                 confidence_level, data_completeness)
            VALUES ('AAPL', '2025-01-15', 75.0, 'Buy', 'High', 0.9)
        """)
    )
    await session.commit()

    # Inserting same ticker+date should fail
    with pytest.raises(Exception):
        await session.execute(
            text("""
                INSERT INTO ic_scores
                    (ticker, date, overall_score, rating,
                     confidence_level, data_completeness)
                VALUES ('AAPL', '2025-01-15', 80.0, 'Buy',
                        'High', 0.95)
            """)
        )
        await session.commit()
