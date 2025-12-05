#!/usr/bin/env python3
"""Daily Stock Price Update Pipeline.

Fetches the latest daily prices from Polygon.io for all tickers to keep
the stock_prices table up to date for technical indicators and risk metrics.

This should run after market close (6 PM ET / 11 PM UTC) on weekdays.

Usage:
    python daily_price_update.py --all                    # All tickers
    python daily_price_update.py --ticker AAPL            # Single ticker
    python daily_price_update.py --limit 100              # Test on 100 tickers
    python daily_price_update.py --days 5                 # Fetch last 5 days
"""

import argparse
import asyncio
import logging
import os
import sys
from datetime import datetime, timedelta
from typing import List, Optional

import asyncpg

# Add parent directory to path for imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from pipelines.utils.polygon_client import PolygonClient

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('/app/logs/daily_price_update.log')
    ]
)
logger = logging.getLogger(__name__)


class DailyPriceUpdate:
    """Update daily prices from Polygon.io."""

    def __init__(self, days: int = 5, batch_size: int = 100):
        """Initialize updater.

        Args:
            days: Number of days to fetch (default: 5 to catch weekends/holidays).
            batch_size: Number of tickers to process in each batch.
        """
        self.days = days
        self.batch_size = batch_size
        self.polygon = PolygonClient()
        self.pool: Optional[asyncpg.Pool] = None

        # Calculate date range (fetch last N days to catch any gaps)
        self.to_date = datetime.now().strftime('%Y-%m-%d')
        self.from_date = (datetime.now() - timedelta(days=days)).strftime('%Y-%m-%d')

        # Stats
        self.processed = 0
        self.success = 0
        self.errors = 0
        self.total_rows_inserted = 0

    async def connect(self):
        """Connect to database."""
        self.pool = await asyncpg.create_pool(
            host=os.getenv('DB_HOST', 'localhost'),
            port=int(os.getenv('DB_PORT', 5432)),
            user=os.getenv('DB_USER', 'investorcenter'),
            password=os.getenv('DB_PASSWORD', 'investorcenter'),
            database=os.getenv('DB_NAME', 'investorcenter_db'),
            min_size=2,
            max_size=10
        )
        logger.info("Connected to database")

    async def close(self):
        """Close connections."""
        if self.pool:
            await self.pool.close()
        self.polygon.close()

    async def get_tickers(self, limit: Optional[int] = None) -> List[str]:
        """Get all tickers from tickers table (high-quality stocks + ETFs only).

        Args:
            limit: Limit number of tickers.

        Returns:
            List of ticker symbols.
        """
        query = """
            SELECT DISTINCT symbol
            FROM tickers
            WHERE symbol IS NOT NULL
              AND active = true
              AND asset_type IN ('CS', 'ETF')
              AND exchange IN ('XNAS', 'XNYS', 'XASE', 'ARCX', 'BATS')
            ORDER BY symbol
        """
        if limit:
            query += f" LIMIT {limit}"

        async with self.pool.acquire() as conn:
            rows = await conn.fetch(query)
            return [row['symbol'] for row in rows]

    def fetch_prices(self, ticker: str) -> Optional[List[dict]]:
        """Fetch recent prices from Polygon.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            List of price bars or None if error.
        """
        try:
            bars = self.polygon.get_aggregates(
                ticker=ticker,
                multiplier=1,
                timespan='day',
                from_date=self.from_date,
                to_date=self.to_date,
                limit=50
            )
            return bars
        except Exception as e:
            logger.error(f"{ticker}: Error fetching prices: {e}")
            return None

    async def store_prices(self, ticker: str, bars: List[dict]) -> int:
        """Store prices in database.

        Args:
            ticker: Stock ticker symbol.
            bars: List of price bars from Polygon.

        Returns:
            Number of rows inserted/updated.
        """
        if not bars:
            return 0

        # Prepare data for insertion
        values = []
        for bar in bars:
            try:
                timestamp = datetime.fromtimestamp(bar['t'] / 1000)
                values.append((
                    timestamp,
                    ticker,
                    float(bar.get('o', 0)),  # open
                    float(bar.get('h', 0)),  # high
                    float(bar.get('l', 0)),  # low
                    float(bar.get('c', 0)),  # close
                    int(bar.get('v', 0)),    # volume
                    float(bar.get('vw', 0)) if bar.get('vw') else None,  # vwap
                    '1day'
                ))
            except (KeyError, ValueError, TypeError) as e:
                logger.warning(f"{ticker}: Skipping invalid bar: {e}")
                continue

        if not values:
            return 0

        # Use ON CONFLICT to handle duplicates (upsert)
        query = """
            INSERT INTO stock_prices (time, ticker, open, high, low, close, volume, vwap, interval)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            ON CONFLICT (ticker, time, interval) DO UPDATE SET
                open = EXCLUDED.open,
                high = EXCLUDED.high,
                low = EXCLUDED.low,
                close = EXCLUDED.close,
                volume = EXCLUDED.volume,
                vwap = EXCLUDED.vwap
        """

        async with self.pool.acquire() as conn:
            await conn.executemany(query, values)

        return len(values)

    async def process_ticker(self, ticker: str) -> bool:
        """Process a single ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            True if successful.
        """
        self.processed += 1

        # Fetch prices
        bars = self.fetch_prices(ticker)

        if bars is None:
            self.errors += 1
            return False

        if len(bars) == 0:
            # No data for this period is not necessarily an error (e.g., new IPO)
            logger.debug(f"{ticker}: No recent price data")
            return True

        # Store prices
        rows_inserted = await self.store_prices(ticker, bars)
        self.total_rows_inserted += rows_inserted
        self.success += 1

        if rows_inserted > 0:
            logger.debug(f"{ticker}: Updated {rows_inserted} price records")

        return True

    async def run(
        self,
        tickers: Optional[List[str]] = None,
        limit: Optional[int] = None
    ):
        """Run the daily update process.

        Args:
            tickers: List of tickers to process (or None for all).
            limit: Limit number of tickers.
        """
        start_time = datetime.now()

        await self.connect()

        # Get tickers to process
        if tickers:
            all_tickers = tickers
        else:
            all_tickers = await self.get_tickers(limit)

        total_tickers = len(all_tickers)
        logger.info(f"Starting daily price update")
        logger.info(f"  Date range: {self.from_date} to {self.to_date} ({self.days} days)")
        logger.info(f"  Total tickers: {total_tickers}")
        logger.info("")

        # Process in batches
        for i in range(0, total_tickers, self.batch_size):
            batch = all_tickers[i:i + self.batch_size]
            batch_num = i // self.batch_size + 1
            total_batches = (total_tickers + self.batch_size - 1) // self.batch_size

            for ticker in batch:
                await self.process_ticker(ticker)

            # Progress update every batch
            elapsed = (datetime.now() - start_time).total_seconds()
            rate = self.processed / elapsed if elapsed > 0 else 0

            logger.info(
                f"Batch {batch_num}/{total_batches} | "
                f"Progress: {self.processed}/{total_tickers} ({100*self.processed/total_tickers:.1f}%) | "
                f"Success: {self.success} | Errors: {self.errors} | "
                f"Rate: {rate:.1f}/s"
            )

        await self.close()

        # Final summary
        duration = datetime.now() - start_time
        logger.info("")
        logger.info("=" * 60)
        logger.info("Daily Price Update Complete")
        logger.info("=" * 60)
        logger.info(f"Duration: {duration}")
        logger.info(f"Processed: {self.processed}")
        logger.info(f"Success: {self.success}")
        logger.info(f"Errors: {self.errors}")
        logger.info(f"Total rows updated: {self.total_rows_inserted:,}")


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description='Update daily stock prices from Polygon.io',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Update all tickers with last 5 days of data
  python daily_price_update.py --all

  # Test on 100 tickers first
  python daily_price_update.py --limit 100

  # Update single ticker
  python daily_price_update.py --ticker AAPL

  # Fetch more days (useful after long weekend)
  python daily_price_update.py --all --days 10

Notes:
  - Requires POLYGON_API_KEY environment variable
  - Rate limited to ~5 requests/second
  - Uses upsert to avoid duplicates
        """
    )

    parser.add_argument(
        '--all',
        action='store_true',
        help='Update all tickers'
    )
    parser.add_argument(
        '--ticker',
        type=str,
        help='Update single ticker'
    )
    parser.add_argument(
        '--days',
        type=int,
        default=5,
        help='Days of history to fetch (default: 5)'
    )
    parser.add_argument(
        '--limit',
        type=int,
        help='Limit number of tickers'
    )
    parser.add_argument(
        '--batch-size',
        type=int,
        default=100,
        help='Batch size for processing (default: 100)'
    )

    args = parser.parse_args()

    if not args.all and not args.ticker and not args.limit:
        parser.print_help()
        sys.exit(1)

    # Create updater instance
    updater = DailyPriceUpdate(
        days=args.days,
        batch_size=args.batch_size
    )

    # Run
    if args.ticker:
        asyncio.run(updater.run(tickers=[args.ticker]))
    else:
        asyncio.run(updater.run(limit=args.limit))


if __name__ == '__main__':
    main()
