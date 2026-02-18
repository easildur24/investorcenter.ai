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
            ("risk_metrics", "var_95", "NUMERIC(10,6)"),
            ("risk_metrics", "cvar_95", "NUMERIC(10,6)"),
            (
                "risk_metrics",
                "tracking_error",
                "NUMERIC(10,6)",
            ),
            (
                "risk_metrics",
                "information_ratio",
                "NUMERIC(10,6)",
            ),
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

        # Create tables from Alembic migrations (006-010) that
        # are not in the *.sql files and not in models.py.
        _alembic_stmts = [
            # --- Migration 006: sector_percentiles ---
            (
                "CREATE TABLE IF NOT EXISTS"
                " sector_percentiles ("
                " id UUID PRIMARY KEY"
                " DEFAULT gen_random_uuid(),"
                " sector VARCHAR(50) NOT NULL,"
                " metric_name VARCHAR(50) NOT NULL,"
                " calculated_at DATE NOT NULL"
                " DEFAULT CURRENT_DATE,"
                " min_value NUMERIC(20,4),"
                " p10_value NUMERIC(20,4),"
                " p25_value NUMERIC(20,4),"
                " p50_value NUMERIC(20,4),"
                " p75_value NUMERIC(20,4),"
                " p90_value NUMERIC(20,4),"
                " max_value NUMERIC(20,4),"
                " mean_value NUMERIC(20,4),"
                " std_dev NUMERIC(20,4),"
                " sample_count INTEGER,"
                " created_at TIMESTAMPTZ DEFAULT NOW(),"
                " UNIQUE (sector, metric_name,"
                " calculated_at))"
            ),
            (
                "CREATE MATERIALIZED VIEW"
                " IF NOT EXISTS"
                " mv_latest_sector_percentiles AS"
                " SELECT DISTINCT ON"
                " (sector, metric_name)"
                " id, sector, metric_name,"
                " calculated_at,"
                " min_value, p10_value,"
                " p25_value, p50_value,"
                " p75_value, p90_value,"
                " max_value, mean_value,"
                " std_dev, sample_count"
                " FROM sector_percentiles"
                " ORDER BY sector, metric_name,"
                " calculated_at DESC"
            ),
            # --- Migration 007: lifecycle_classifications ---
            (
                "ALTER TABLE ic_scores"
                " ADD COLUMN IF NOT EXISTS"
                " lifecycle_stage VARCHAR(20)"
            ),
            (
                "ALTER TABLE ic_scores"
                " ADD COLUMN IF NOT EXISTS"
                " raw_score NUMERIC(5,2)"
            ),
            (
                "ALTER TABLE ic_scores"
                " ADD COLUMN IF NOT EXISTS"
                " smoothing_applied BOOLEAN"
                " DEFAULT FALSE"
            ),
            (
                "ALTER TABLE ic_scores"
                " ADD COLUMN IF NOT EXISTS"
                " weights_used JSONB"
            ),
            (
                "ALTER TABLE ic_scores"
                " ADD COLUMN IF NOT EXISTS"
                " sector_rank INTEGER"
            ),
            (
                "ALTER TABLE ic_scores"
                " ADD COLUMN IF NOT EXISTS"
                " sector_total INTEGER"
            ),
            # --- Migration 008: eps_estimates ---
            (
                "CREATE TABLE IF NOT EXISTS"
                " eps_estimates ("
                " id UUID PRIMARY KEY"
                " DEFAULT gen_random_uuid(),"
                " ticker VARCHAR(10) NOT NULL,"
                " fiscal_year INTEGER NOT NULL,"
                " fiscal_quarter INTEGER,"
                " consensus_eps NUMERIC(10,4),"
                " num_analysts INTEGER,"
                " high_estimate NUMERIC(10,4),"
                " low_estimate NUMERIC(10,4),"
                " estimate_30d_ago NUMERIC(10,4),"
                " estimate_60d_ago NUMERIC(10,4),"
                " estimate_90d_ago NUMERIC(10,4),"
                " upgrades_30d INTEGER DEFAULT 0,"
                " downgrades_30d INTEGER DEFAULT 0,"
                " upgrades_60d INTEGER DEFAULT 0,"
                " downgrades_60d INTEGER DEFAULT 0,"
                " upgrades_90d INTEGER DEFAULT 0,"
                " downgrades_90d INTEGER DEFAULT 0,"
                " revision_pct_30d NUMERIC(10,4),"
                " revision_pct_60d NUMERIC(10,4),"
                " revision_pct_90d NUMERIC(10,4),"
                " fetched_at TIMESTAMPTZ,"
                " created_at TIMESTAMPTZ"
                " DEFAULT NOW(),"
                " updated_at TIMESTAMPTZ"
                " DEFAULT NOW(),"
                " UNIQUE (ticker, fiscal_year,"
                " fiscal_quarter))"
            ),
            # --- Migration 009: valuation_history ---
            (
                "CREATE TABLE IF NOT EXISTS"
                " valuation_history ("
                " id UUID PRIMARY KEY"
                " DEFAULT gen_random_uuid(),"
                " ticker VARCHAR(10) NOT NULL,"
                " snapshot_date DATE NOT NULL,"
                " pe_ratio NUMERIC(10,2),"
                " ps_ratio NUMERIC(10,2),"
                " pb_ratio NUMERIC(10,2),"
                " ev_ebitda NUMERIC(10,2),"
                " peg_ratio NUMERIC(10,2),"
                " stock_price NUMERIC(10,2),"
                " market_cap NUMERIC(20,2),"
                " eps_ttm NUMERIC(10,4),"
                " revenue_ttm NUMERIC(20,2),"
                " created_at TIMESTAMPTZ"
                " DEFAULT NOW(),"
                " UNIQUE (ticker, snapshot_date))"
            ),
            (
                "ALTER TABLE ic_scores"
                " ADD COLUMN IF NOT EXISTS"
                " earnings_revisions_score"
                " NUMERIC(5,2)"
            ),
            (
                "ALTER TABLE ic_scores"
                " ADD COLUMN IF NOT EXISTS"
                " historical_value_score"
                " NUMERIC(5,2)"
            ),
            (
                "ALTER TABLE ic_scores"
                " ADD COLUMN IF NOT EXISTS"
                " dividend_quality_score"
                " NUMERIC(5,2)"
            ),
            # --- Migration 010: phase 3 tables ---
            (
                "ALTER TABLE ic_scores"
                " ADD COLUMN IF NOT EXISTS"
                " previous_score NUMERIC(5,2)"
            ),
            (
                "CREATE TABLE IF NOT EXISTS"
                " ic_score_changes ("
                " id UUID PRIMARY KEY"
                " DEFAULT gen_random_uuid(),"
                " ticker VARCHAR(10) NOT NULL,"
                " calculated_at DATE NOT NULL,"
                " previous_score NUMERIC(5,2),"
                " current_score NUMERIC(5,2),"
                " delta NUMERIC(5,2),"
                " factor_changes JSONB,"
                " trigger_events JSONB,"
                " smoothing_applied BOOLEAN"
                " DEFAULT FALSE,"
                " created_at TIMESTAMPTZ"
                " DEFAULT NOW())"
            ),
            (
                "CREATE TABLE IF NOT EXISTS"
                " stock_peers ("
                " id UUID PRIMARY KEY"
                " DEFAULT gen_random_uuid(),"
                " ticker VARCHAR(10) NOT NULL,"
                " peer_ticker VARCHAR(10) NOT NULL,"
                " similarity_score NUMERIC(5,4),"
                " similarity_factors JSONB,"
                " calculated_at DATE"
                " DEFAULT CURRENT_DATE,"
                " created_at TIMESTAMPTZ"
                " DEFAULT NOW(),"
                " UNIQUE (ticker, peer_ticker,"
                " calculated_at))"
            ),
            (
                "CREATE TABLE IF NOT EXISTS"
                " catalyst_events ("
                " id UUID PRIMARY KEY"
                " DEFAULT gen_random_uuid(),"
                " ticker VARCHAR(10) NOT NULL,"
                " event_type VARCHAR(50) NOT NULL,"
                " title VARCHAR(200) NOT NULL,"
                " description TEXT,"
                " event_date DATE,"
                " impact VARCHAR(20),"
                " confidence NUMERIC(3,2),"
                " source VARCHAR(50),"
                " metadata JSONB,"
                " created_at TIMESTAMPTZ"
                " DEFAULT NOW(),"
                " expires_at TIMESTAMPTZ)"
            ),
            (
                "CREATE TABLE IF NOT EXISTS"
                " ic_score_events ("
                " id UUID PRIMARY KEY"
                " DEFAULT gen_random_uuid(),"
                " ticker VARCHAR(10) NOT NULL,"
                " event_type VARCHAR(50) NOT NULL,"
                " event_date DATE NOT NULL,"
                " description TEXT,"
                " impact_direction VARCHAR(10),"
                " impact_magnitude NUMERIC(5,2),"
                " source VARCHAR(50),"
                " metadata JSONB,"
                " created_at TIMESTAMPTZ"
                " DEFAULT NOW())"
            ),
        ]
        for stmt in _alembic_stmts:
            try:
                await conn.execute(
                    text("SAVEPOINT alembic_stmt")
                )
                await conn.execute(text(stmt))
                await conn.execute(
                    text("RELEASE SAVEPOINT alembic_stmt")
                )
            except Exception:
                await conn.execute(
                    text(
                        "ROLLBACK TO SAVEPOINT"
                        " alembic_stmt"
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

    # Nuclear teardown: drop everything in public schema.
    # drop_all_tables() fails when materialized views or
    # foreign key constraints reference migration-only tables.
    async with database.engine.begin() as conn:
        await conn.execute(
            text("DROP SCHEMA public CASCADE")
        )
        await conn.execute(
            text("CREATE SCHEMA public")
        )
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
