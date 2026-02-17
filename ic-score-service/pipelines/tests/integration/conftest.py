"""Shared fixtures for pipeline integration tests.

Sets up a real PostgreSQL + TimescaleDB database with full schema
(models.py tables + migration-only tables like ttm_financials).

Requires:
  INTEGRATION_TEST_DB=true
  DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
"""

import logging
import os
import sys
import tempfile
from pathlib import Path

import pytest
import pytest_asyncio

# --- Patch LOG_DIR and FileHandler before any pipeline import ---
_tmpdir = tempfile.mkdtemp(prefix="ic_integ_test_logs_")
os.environ["LOG_DIR"] = _tmpdir

_original_file_handler = logging.FileHandler


class _SafeFileHandler(logging.StreamHandler):
    """Drop-in replacement that writes to stderr instead of a file."""

    def __init__(
        self, filename=None, mode="a", encoding=None, delay=False
    ):
        super().__init__()


logging.FileHandler = _SafeFileHandler  # type: ignore[misc]

# --- Skip if no integration DB ---
pytestmark = pytest.mark.skipif(
    os.getenv("INTEGRATION_TEST_DB") != "true",
    reason="INTEGRATION_TEST_DB not set",
)

# Ensure ic-score-service root is on sys.path
sys.path.insert(0, str(Path(__file__).parent.parent.parent.parent))

from sqlalchemy import text  # noqa: E402
from sqlalchemy.ext.asyncio import create_async_engine  # noqa: E402
from sqlalchemy.pool import NullPool  # noqa: E402

from database.database import Database, DatabaseConfig  # noqa: E402

# Migration SQL files that create tables NOT in models.py
_MIGRATION_DIR = (
    Path(__file__).parent.parent.parent.parent / "migrations"
)
_MIGRATION_FILES = sorted(
    _MIGRATION_DIR.glob("*.sql"),
    key=lambda p: p.name,
)

# Hypertable definitions
_HYPERTABLES = [
    ("stock_prices", "time"),
    ("technical_indicators", "time"),
    ("risk_metrics", "time"),
    ("benchmark_returns", "time"),
]


def _split_sql_statements(sql: str) -> list:
    """Split a SQL file into individual statements.

    Handles -- comments, multi-line statements, and GRANT filtering.
    Returns a list of non-empty SQL statements.
    """
    statements = []
    current = []

    for line in sql.split("\n"):
        stripped = line.strip()
        # Skip pure comment lines and GRANT statements
        if stripped.startswith("--") or not stripped:
            continue
        upper = stripped.upper()
        if upper.startswith("GRANT "):
            continue

        current.append(line)

        # Statement ends with semicolon
        if stripped.endswith(";"):
            stmt = "\n".join(current).strip()
            if stmt and stmt != ";":
                statements.append(stmt)
            current = []

    # Leftover without trailing semicolon
    if current:
        stmt = "\n".join(current).strip()
        if stmt:
            statements.append(stmt)

    return statements


async def _apply_migrations(engine):
    """Apply SQL migration files and create TimescaleDB hypertables.

    Splits each migration file into individual statements because
    asyncpg cannot execute multiple commands in one prepared statement.
    """
    async with engine.begin() as conn:
        # Enable TimescaleDB extension
        await conn.execute(
            text(
                "CREATE EXTENSION IF NOT EXISTS timescaledb"
                " CASCADE"
            )
        )

        # Apply each migration file statement by statement
        for migration_file in _MIGRATION_FILES:
            sql = migration_file.read_text()
            stmts = _split_sql_statements(sql)
            for stmt in stmts:
                try:
                    await conn.execute(text(stmt))
                except Exception as e:
                    # Tables/indexes may already exist from
                    # create_all_tables() — that's OK
                    msg = str(e)
                    if "already exists" in msg:
                        continue
                    raise

        # Create hypertables (idempotent)
        for table_name, time_col in _HYPERTABLES:
            try:
                await conn.execute(
                    text(
                        f"SELECT create_hypertable("
                        f"'{table_name}', '{time_col}',"
                        f" if_not_exists => TRUE,"
                        f" migrate_data => TRUE)"
                    )
                )
            except Exception:
                pass  # Table may not exist or already hypertable


@pytest_asyncio.fixture(scope="session")
async def db():
    """Create Database instance with full schema.

    Session-scoped: schema is created once, tables are truncated
    between tests via the clean_tables fixture.
    """
    config = DatabaseConfig()
    database = Database(config)

    database._engine = create_async_engine(
        config.url,
        echo=False,
        poolclass=NullPool,
    )
    database._session_factory = None

    # Create tables from models.py (companies, ic_scores, etc.)
    await database.create_all_tables()

    # Apply migration-only tables (ttm_financials, valuation_ratios,
    # fundamental_metrics_extended, dividends, etc.)
    await _apply_migrations(database.engine)

    yield database

    await database.drop_all_tables()
    await database.close()


@pytest_asyncio.fixture(autouse=True)
async def clean_tables(db):
    """Truncate all pipeline-related tables before each test."""
    async with db.engine.begin() as conn:
        # Get all user tables and truncate them
        result = await conn.execute(
            text(
                "SELECT tablename FROM pg_tables"
                " WHERE schemaname = 'public'"
            )
        )
        tables = [row[0] for row in result.fetchall()]

        if tables:
            table_list = ", ".join(tables)
            await conn.execute(
                text(f"TRUNCATE {table_list} CASCADE")
            )
    yield


@pytest_asyncio.fixture
async def session(db):
    """Provide a database session for test use."""
    async with db.session_factory() as sess:
        yield sess
