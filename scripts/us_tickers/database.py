"""Database connectivity for US Tickers package."""

import logging
import os
from contextlib import contextmanager
from typing import Generator, Tuple

import pandas as pd
import psycopg2
import psycopg2.extras
from dotenv import load_dotenv

logger = logging.getLogger(__name__)

# Load environment variables
load_dotenv()


class DatabaseConfig:
    """Database configuration from environment variables."""

    def __init__(self) -> None:
        self.host = os.getenv("DB_HOST", "localhost")
        self.port = int(os.getenv("DB_PORT", "5432"))
        self.user = os.getenv("DB_USER", "investorcenter")
        self.password = os.getenv("DB_PASSWORD", "")
        self.database = os.getenv("DB_NAME", "investorcenter_db")
        self.sslmode = os.getenv("DB_SSLMODE", "require")

    @property
    def connection_string(self) -> str:
        """Get PostgreSQL connection string."""
        return (
            f"host={self.host} port={self.port} user={self.user} "
            f"password={self.password} dbname={self.database} "
            f"sslmode={self.sslmode}"
        )


@contextmanager
def get_db_connection() -> (
    Generator[psycopg2.extensions.connection, None, None]
):
    """Get database connection with automatic cleanup."""
    config = DatabaseConfig()
    conn = None

    try:
        logger.info(
            f"Connecting to database: "
            f"{config.user}@{config.host}:{config.port}/{config.database}"
        )
        conn = psycopg2.connect(config.connection_string)
        conn.autocommit = False  # Use transactions
        yield conn

    except psycopg2.Error as e:
        logger.error(f"Database error: {e}")
        if conn:
            conn.rollback()
        raise
    finally:
        if conn:
            conn.close()


def test_database_connection() -> bool:
    """Test database connectivity and schema."""
    try:
        with get_db_connection() as conn:
            with conn.cursor() as cur:
                # Test basic connectivity
                cur.execute("SELECT version()")
                version_result = cur.fetchone()
                if version_result:
                    version = version_result[0]
                    logger.info(f"Connected to PostgreSQL: {version}")
                else:
                    logger.error("Failed to get PostgreSQL version")
                    return False

                # Check if stocks table exists
                cur.execute(
                    """
                    SELECT EXISTS (
                        SELECT FROM information_schema.tables
                        WHERE table_name = 'stocks'
                    )
                """
                )
                table_result = cur.fetchone()
                if not table_result:
                    logger.error("Failed to check if stocks table exists")
                    return False
                table_exists = table_result[0]

                if not table_exists:
                    logger.warning(
                        "Stocks table does not exist. Run migrations first."
                    )
                    logger.warning(
                        "Run: psql -d investorcenter_db "
                        "-f backend/migrations/001_create_stock_tables.sql"
                    )
                    return False

                # Check current stock count
                cur.execute("SELECT COUNT(*) FROM stocks")
                count_result = cur.fetchone()
                if not count_result:
                    logger.error("Failed to get stock count")
                    return False
                count = count_result[0]
                logger.info(f"Current stocks in database: {count}")

                return True

    except Exception as e:
        logger.error(f"Database connection test failed: {e}")
        return False


def import_stocks_to_database(
    stocks_df: pd.DataFrame, batch_size: int = 100
) -> Tuple[int, int]:
    """
    Import stocks to database with incremental updates.

    Args:
        stocks_df: DataFrame with transformed stock data
        batch_size: Number of records to insert per batch

    Returns:
        Tuple of (inserted_count, skipped_count)
    """
    if stocks_df.empty:
        logger.warning("No stocks to import")
        return 0, 0

    inserted_count = 0
    skipped_count = 0

    try:
        with get_db_connection() as conn:
            with conn.cursor() as cur:
                # Prepare INSERT statement with conflict handling
                insert_query = """
                    INSERT INTO stocks (
                        symbol, name, exchange, sector, industry,
                        country, currency, market_cap, description, website
                    )
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                    ON CONFLICT (symbol) DO NOTHING
                """

                # Process in batches
                total_records = len(stocks_df)
                logger.info(
                    f"Importing {total_records} stocks in batches of "
                    f"{batch_size}"
                )

                for i in range(0, total_records, batch_size):
                    batch_df = stocks_df.iloc[i : i + batch_size]
                    batch_data = []

                    for _, row in batch_df.iterrows():
                        values = (
                            row["symbol"],
                            row["name"],
                            row["exchange"],
                            row.get("sector"),
                            row.get("industry"),
                            row["country"],
                            row["currency"],
                            row.get("market_cap"),
                            row.get("description"),
                            row.get("website"),
                        )
                        batch_data.append(values)

                    # Execute batch insert
                    batch_num = (i // batch_size) + 1
                    total_batches = (
                        total_records + batch_size - 1
                    ) // batch_size
                    logger.info(
                        f"Processing batch {batch_num}/{total_batches} "
                        f"({len(batch_data)} records)"
                    )

                    # Get initial count to calculate inserts
                    cur.execute("SELECT COUNT(*) FROM stocks")
                    count_before_result = cur.fetchone()
                    if not count_before_result:
                        raise Exception("Failed to get initial stock count")
                    count_before = count_before_result[0]

                    # Execute batch insert
                    psycopg2.extras.execute_batch(
                        cur, insert_query, batch_data
                    )

                    # Get final count
                    cur.execute("SELECT COUNT(*) FROM stocks")
                    count_after_result = cur.fetchone()
                    if not count_after_result:
                        raise Exception("Failed to get final stock count")
                    count_after = count_after_result[0]

                    batch_inserted = count_after - count_before
                    batch_skipped = len(batch_data) - batch_inserted

                    inserted_count += batch_inserted
                    skipped_count += batch_skipped

                    logger.info(
                        f"Batch {batch_num}: {batch_inserted} inserted, "
                        f"{batch_skipped} skipped"
                    )

                # Commit transaction
                conn.commit()
                logger.info(
                    f"Import completed: {inserted_count} inserted, "
                    f"{skipped_count} skipped"
                )

    except Exception as e:
        logger.error(f"Failed to import stocks: {e}")
        raise

    return inserted_count, skipped_count


def get_existing_symbols() -> set:
    """Get set of existing stock symbols from database."""
    try:
        with get_db_connection() as conn:
            with conn.cursor() as cur:
                cur.execute("SELECT symbol FROM stocks")
                symbols = {row[0] for row in cur.fetchall()}
                logger.info(
                    f"Found {len(symbols)} existing symbols in database"
                )
                return symbols

    except Exception as e:
        logger.error(f"Failed to get existing symbols: {e}")
        return set()


def get_database_stats() -> dict:
    """Get database statistics."""
    try:
        with get_db_connection() as conn:
            with conn.cursor() as cur:
                stats = {}

                # Total stocks
                cur.execute("SELECT COUNT(*) FROM stocks")
                total_result = cur.fetchone()
                if total_result:
                    stats["total_stocks"] = total_result[0]
                else:
                    stats["total_stocks"] = 0

                # Stocks by exchange
                cur.execute(
                    """
                    SELECT exchange, COUNT(*)
                    FROM stocks
                    WHERE exchange IS NOT NULL
                    GROUP BY exchange
                    ORDER BY COUNT(*) DESC
                """
                )
                stats["by_exchange"] = dict(cur.fetchall())

                # Recent additions
                cur.execute(
                    """
                    SELECT COUNT(*) FROM stocks
                    WHERE created_at >= NOW() - INTERVAL '24 hours'
                """
                )
                recent_result = cur.fetchone()
                if recent_result:
                    stats["added_last_24h"] = recent_result[0]
                else:
                    stats["added_last_24h"] = 0

                return stats

    except Exception as e:
        logger.error(f"Failed to get database stats: {e}")
        return {}
