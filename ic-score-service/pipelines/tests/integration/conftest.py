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

    Handles:
    - ``--`` line comments (both standalone and inline)
    - ``$$`` dollar-quoted blocks (PL/pgSQL functions)
    - ``GRANT`` statement filtering
    Returns a list of non-empty SQL statements.
    """
    statements = []
    current = []
    in_dollar_quote = False

    for line in sql.split("\n"):
        stripped = line.strip()

        # Skip pure comment lines and blanks outside $$
        if not in_dollar_quote:
            if stripped.startswith("--") or not stripped:
                continue
            if stripped.upper().startswith("GRANT "):
                continue

        current.append(line)

        # Track $$ dollar-quoting (toggle on each $$)
        count = line.count("$$")
        if count % 2 == 1:
            in_dollar_quote = not in_dollar_quote

        # Check for statement end: ; before any inline --
        # e.g. "HAVING COUNT(*) >= 5;  -- comment"
        if not in_dollar_quote:
            code_part = stripped
            if "--" in code_part:
                code_part = code_part[
                    : code_part.index("--")
                ].rstrip()
            if code_part.endswith(";"):
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
    Uses SAVEPOINTs so a failing statement doesn't abort the whole
    transaction (e.g. ALTER TYPE on a view-referenced column).
    """
    async with engine.begin() as conn:
        # Enable TimescaleDB extension
        await conn.execute(
            text(
                "CREATE EXTENSION IF NOT EXISTS timescaledb"
                " CASCADE"
            )
        )

        # Apply each migration file statement by statement.
        for migration_file in _MIGRATION_FILES:
            sql = migration_file.read_text()
            stmts = _split_sql_statements(sql)
            for stmt in stmts:
                try:
                    await conn.execute(
                        text("SAVEPOINT migration_stmt")
                    )
                    await conn.execute(text(stmt))
                    await conn.execute(
                        text(
                            "RELEASE SAVEPOINT migration_stmt"
                        )
                    )
                except Exception:
                    await conn.execute(
                        text(
                            "ROLLBACK TO SAVEPOINT"
                            " migration_stmt"
                        )
                    )

        # Add columns that exist in production but have no
        # migration file (were added via manual ALTER TABLE).
        _extra_columns = [
            (
                "ttm_financials",
                "avg_shareholders_equity_5q",
                "BIGINT",
            ),
            ("treasury_rates", "rate_5y", "NUMERIC(5,2)"),
            ("risk_metrics", "volatility", "NUMERIC(10,6)"),
        ]
        for tbl, col, dtype in _extra_columns:
            try:
                await conn.execute(
                    text(f"SAVEPOINT extra_col")
                )
                await conn.execute(
                    text(
                        f"ALTER TABLE {tbl}"
                        f" ADD COLUMN IF NOT EXISTS"
                        f" {col} {dtype}"
                    )
                )
                await conn.execute(
                    text("RELEASE SAVEPOINT extra_col")
                )
            except Exception:
                await conn.execute(
                    text(
                        "ROLLBACK TO SAVEPOINT extra_col"
                    )
                )

        # Create hypertables (idempotent)
        for table_name, time_col in _HYPERTABLES:
            try:
                await conn.execute(
                    text("SAVEPOINT hypertable_stmt")
                )
                await conn.execute(
                    text(
                        f"SELECT create_hypertable("
                        f"'{table_name}', '{time_col}',"
                        f" if_not_exists => TRUE,"
                        f" migrate_data => TRUE)"
                    )
                )
                await conn.execute(
                    text("RELEASE SAVEPOINT hypertable_stmt")
                )
            except Exception:
                await conn.execute(
                    text(
                        "ROLLBACK TO SAVEPOINT"
                        " hypertable_stmt"
                    )
                )


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
