#!/usr/bin/env python3
"""Backfill EPS values for existing financial records.

This script calculates and updates eps_basic and eps_diluted for all financial
records that have net_income and shares_outstanding but missing EPS values.

Usage:
    python backfill_eps.py --all          # Backfill all records
    python backfill_eps.py --limit 1000   # Test on 1000 records
    python backfill_eps.py --ticker AAPL  # Single ticker
"""

import argparse
import asyncio
import logging
import os
import sys
from datetime import datetime
from pathlib import Path
from typing import List, Optional

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

from sqlalchemy import text
from tqdm import tqdm

from database.database import get_database

# Setup logging with configurable log directory
LOG_DIR = os.environ.get('LOG_DIR', '/app/logs')
Path(LOG_DIR).mkdir(parents=True, exist_ok=True)

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler(os.path.join(LOG_DIR, 'backfill_eps.log'))
    ]
)
logger = logging.getLogger(__name__)


class EPSBackfiller:
    """Backfill EPS values for existing financial records."""

    def __init__(self):
        """Initialize the backfiller."""
        self.db = get_database()

        # Track progress
        self.processed_count = 0
        self.updated_count = 0
        self.skipped_count = 0
        self.error_count = 0

    async def get_records_to_update(
        self,
        limit: Optional[int] = None,
        ticker: Optional[str] = None
    ) -> List[dict]:
        """Get financial records that need EPS calculated.

        Args:
            limit: Maximum number of records to process.
            ticker: Filter by single ticker.

        Returns:
            List of financial record dicts with id, ticker, net_income, shares_outstanding.
        """
        async with self.db.session() as session:
            where_clauses = [
                "net_income IS NOT NULL",
                "shares_outstanding IS NOT NULL",
                "shares_outstanding > 0",
                "(eps_basic IS NULL OR eps_diluted IS NULL)"
            ]
            params = {"limit": limit or 1000000}

            if ticker:
                where_clauses.append("ticker = :ticker")
                params["ticker"] = ticker.upper()

            query = text(f"""
                SELECT
                    id,
                    ticker,
                    period_end_date,
                    net_income,
                    shares_outstanding,
                    eps_basic,
                    eps_diluted
                FROM financials
                WHERE {' AND '.join(where_clauses)}
                ORDER BY ticker, period_end_date DESC
                LIMIT :limit
            """)

            result = await session.execute(query, params)
            rows = result.fetchall()

            records = []
            for row in rows:
                records.append({
                    'id': row[0],
                    'ticker': row[1],
                    'period_end_date': row[2],
                    'net_income': int(row[3]) if row[3] else None,
                    'shares_outstanding': int(row[4]) if row[4] else None,
                    'eps_basic': row[5],
                    'eps_diluted': row[6],
                })

        logger.info(f"Found {len(records)} records to update")
        return records

    def calculate_eps(self, record: dict) -> dict:
        """Calculate EPS for a financial record.

        Args:
            record: Financial record dict.

        Returns:
            Dict with calculated eps_basic and eps_diluted.
        """
        net_income = record['net_income']
        shares_outstanding = record['shares_outstanding']

        if not net_income or not shares_outstanding or shares_outstanding <= 0:
            return None

        eps = round(net_income / shares_outstanding, 4)

        return {
            'id': record['id'],
            'eps_basic': eps if record['eps_basic'] is None else record['eps_basic'],
            'eps_diluted': eps if record['eps_diluted'] is None else record['eps_diluted'],
        }

    async def update_record(self, eps_data: dict) -> bool:
        """Update financial record with calculated EPS.

        Args:
            eps_data: Dict with id, eps_basic, eps_diluted.

        Returns:
            True if successful, False otherwise.
        """
        try:
            async with self.db.session() as session:
                query = text("""
                    UPDATE financials
                    SET
                        eps_basic = :eps_basic,
                        eps_diluted = :eps_diluted
                    WHERE id = :id
                """)

                await session.execute(query, eps_data)
                await session.commit()

                return True

        except Exception as e:
            logger.error(f"Error updating record {eps_data['id']}: {e}")
            return False

    async def process_batch(self, records: List[dict], batch_size: int = 100):
        """Process records in batches.

        Args:
            records: List of financial records to process.
            batch_size: Number of records per batch.
        """
        progress_bar = tqdm(total=len(records), desc="Backfilling EPS")

        for i in range(0, len(records), batch_size):
            batch = records[i:i + batch_size]

            for record in batch:
                self.processed_count += 1

                # Calculate EPS
                eps_data = self.calculate_eps(record)

                if not eps_data:
                    self.skipped_count += 1
                    progress_bar.update(1)
                    continue

                # Update database
                success = await self.update_record(eps_data)

                if success:
                    self.updated_count += 1
                else:
                    self.error_count += 1

                progress_bar.update(1)
                progress_bar.set_postfix({
                    'updated': self.updated_count,
                    'errors': self.error_count,
                    'skipped': self.skipped_count
                })

        progress_bar.close()

    async def run(
        self,
        limit: Optional[int] = None,
        ticker: Optional[str] = None,
        all_records: bool = False
    ):
        """Run the EPS backfill process.

        Args:
            limit: Limit number of records to process.
            ticker: Process single ticker.
            all_records: Process all records.
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("EPS Backfill Process")
        logger.info("=" * 80)

        # Determine limit
        if all_records:
            limit = None
        elif ticker:
            limit = 10000  # Max records per ticker
        elif limit is None:
            limit = 1000  # Default safe limit

        # Get records to update
        records = await self.get_records_to_update(limit=limit, ticker=ticker)

        if not records:
            logger.info("No records to update")
            return

        logger.info(f"Processing {len(records)} financial records...")

        # Process in batches
        await self.process_batch(records, batch_size=100)

        # Print summary
        duration = datetime.now() - start_time
        logger.info("=" * 80)
        logger.info("Backfill Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Processed: {self.processed_count}")
        logger.info(f"Updated: {self.updated_count}")
        logger.info(f"Skipped: {self.skipped_count}")
        logger.info(f"Errors: {self.error_count}")
        if self.processed_count > 0:
            logger.info(f"Success Rate: {(self.updated_count/self.processed_count*100):.1f}%")


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Backfill EPS values for existing financial records',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Test on 100 records
  python backfill_eps.py --limit 100

  # Process all records
  python backfill_eps.py --all

  # Process single ticker
  python backfill_eps.py --ticker AAPL
        """
    )

    parser.add_argument(
        '--limit',
        type=int,
        help='Limit number of records to process (default: 1000)'
    )
    parser.add_argument(
        '--ticker',
        type=str,
        help='Process single ticker symbol'
    )
    parser.add_argument(
        '--all',
        action='store_true',
        help='Process all records in database'
    )

    args = parser.parse_args()

    # Validate arguments
    if args.ticker and (args.all or args.limit):
        parser.error("--ticker cannot be used with --all or --limit")

    # Run backfill
    backfiller = EPSBackfiller()
    asyncio.run(backfiller.run(
        limit=args.limit,
        ticker=args.ticker,
        all_records=args.all
    ))


if __name__ == '__main__':
    main()
