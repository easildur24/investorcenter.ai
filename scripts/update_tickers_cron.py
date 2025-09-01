#!/usr/bin/env python3
"""
Cron-friendly Ticker Update Script

This script is designed for periodic execution (daily/weekly) via cron or scheduled tasks.
It fetches the latest ticker data and updates the database with any new listings.

Features:
- Only inserts new tickers (ON CONFLICT DO NOTHING)
- Exits gracefully if database is unavailable
- Logs results for monitoring
- Minimal output unless errors occur

Usage:
    # Add to crontab for daily updates at 6 AM:
    0 6 * * * cd /path/to/investorcenter.ai && python scripts/update_tickers_cron.py >> logs/ticker_updates.log 2>&1

Environment Variables Required:
    DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
"""

import logging
import sys
from datetime import datetime
from pathlib import Path

# Add the us_tickers module to Python path
sys.path.append(str(Path(__file__).parent))

from us_tickers import (get_exchange_listed_tickers, import_stocks_to_database,
                        test_database_connection, transform_for_database)
from us_tickers.database import get_database_stats


def setup_logging() -> None:
    """Setup logging for cron execution."""
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s - %(levelname)s - %(message)s",
        handlers=[
            logging.StreamHandler(
                sys.stdout
            ),  # Output to stdout for cron logging
        ],
    )


def main() -> None:
    """Main cron execution function."""
    setup_logging()
    logger = logging.getLogger(__name__)

    start_time = datetime.now()
    logger.info("=== Ticker Update Job Started ===")

    try:
        # Test database connection
        if not test_database_connection():
            logger.error("Database connection failed - skipping update")
            sys.exit(1)

        # Get current stats
        initial_stats = get_database_stats()
        initial_count = initial_stats.get("total_stocks", 0)
        logger.info(f"Current database has {initial_count} stocks")

        # Fetch latest ticker data (Nasdaq + NYSE only, no ETFs/test issues)
        logger.info("Fetching latest ticker data from exchanges...")
        raw_df = get_exchange_listed_tickers(
            exchanges=("Q", "N"),  # Nasdaq + NYSE
            include_etfs=False,
            include_test_issues=False,
        )
        logger.info(f"Downloaded {len(raw_df)} raw ticker records")

        # Transform for database
        transformed_df = transform_for_database(raw_df)
        logger.info(f"Filtered to {len(transformed_df)} valid tickers")

        if transformed_df.empty:
            logger.warning("No valid tickers found after filtering")
            return

        # Import to database (incremental - only new ones)
        logger.info("Importing new tickers to database...")
        inserted, skipped = import_stocks_to_database(
            transformed_df, batch_size=100
        )

        # Log results
        duration = datetime.now() - start_time
        logger.info(
            f"Update completed in {duration.total_seconds():.1f} seconds"
        )
        logger.info(
            f"Results: {inserted} new, {skipped} existing, {inserted + skipped} total processed"
        )

        if inserted > 0:
            logger.info(
                f"🆕 Added {inserted} new companies to InvestorCenter database"
            )

            # Log new companies by exchange
            final_stats = get_database_stats()
            final_count = final_stats.get("total_stocks", 0)
            logger.info(f"Database now contains {final_count} total stocks")
        else:
            logger.info("📊 Database is up to date - no new stocks found")

        logger.info("=== Ticker Update Job Completed Successfully ===")

    except Exception as e:
        duration = datetime.now() - start_time
        logger.error(
            f"Ticker update failed after {duration.total_seconds():.1f} seconds: {e}"
        )
        logger.error("=== Ticker Update Job Failed ===")
        sys.exit(1)


if __name__ == "__main__":
    main()
