#!/usr/bin/env python3
"""Historical Price Backfill Script.

Fetches historical daily prices from Polygon.io for all tickers to support
multi-timeframe charts (1D, 5D, 1M, 3M, 6M, YTD, 1Y, 3Y, 5Y, 10Y, MAX).

Usage:
    python historical_price_backfill.py --all                    # All tickers, 10 years
    python historical_price_backfill.py --ticker AAPL            # Single ticker
    python historical_price_backfill.py --years 15               # Custom years
    python historical_price_backfill.py --limit 100              # Test on 100 tickers
    python historical_price_backfill.py --resume                 # Skip tickers with data
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
    handlers=[logging.StreamHandler()]
)
logger = logging.getLogger(__name__)


class HistoricalPriceBackfill:
    """Backfill historical prices from Polygon.io."""

    def __init__(
        self,
        years: int = 10,
        batch_size: int = 100,
        resume: bool = False
    ):
        """Initialize backfill.

        Args:
            years: Number of years of history to fetch (default: 10).
            batch_size: Number of tickers to process in each batch.
            resume: Skip tickers that already have sufficient data.
        """
        self.years = years
        self.batch_size = batch_size
        self.resume = resume
        self.polygon = PolygonClient()
        self.pool: Optional[asyncpg.Pool] = None

        # Calculate date range
        self.to_date = datetime.now().strftime('%Y-%m-%d')
        self.from_date = (datetime.now() - timedelta(days=years * 365)).strftime('%Y-%m-%d')

        # Stats
        self.processed = 0
        self.success = 0
        self.skipped = 0
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
        """Get all tickers from tickers table (stocks + ETFs only).

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
              AND asset_type IN ('stock', 'etf', 'adr', 'fund')
            ORDER BY symbol
        """
        if limit:
            query += f" LIMIT {limit}"

        async with self.pool.acquire() as conn:
            rows = await conn.fetch(query)
            return [row['symbol'] for row in rows]

    async def get_oldest_price_date(self, ticker: str) -> Optional[datetime]:
        """Get the oldest price date for a ticker.

        Args:
            ticker: Stock ticker symbol.

        Returns:
            Oldest price date or None if no data exists.
        """
        query = """
            SELECT MIN(time) as oldest
            FROM stock_prices
            WHERE ticker = $1
        """
        async with self.pool.acquire() as conn:
            row = await conn.fetchrow(query, ticker)
            return row['oldest'] if row else None

    async def should_skip_ticker(self, ticker: str) -> bool:
        """Check if ticker should be skipped (already has sufficient data).

        Args:
            ticker: Stock ticker symbol.

        Returns:
            True if ticker should be skipped.
        """
        if not self.resume:
            return False

        oldest = await self.get_oldest_price_date(ticker)

        # Skip if we already have data going back far enough
        # If oldest is not None, there's at least one row
        if oldest:
            target_oldest = datetime.now() - timedelta(days=self.years * 365)
            if oldest.replace(tzinfo=None) <= target_oldest:
                return True

        return False

    def fetch_prices(self, ticker: str) -> Optional[List[dict]]:
        """Fetch historical prices from Polygon.

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
                limit=50000
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
            Number of rows inserted.
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

        # Use ON CONFLICT to handle duplicates
        query = """
            INSERT INTO stock_prices (time, ticker, open, high, low, close, volume, vwap, interval)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            ON CONFLICT (time, ticker) DO UPDATE SET
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

        # Check if should skip
        if await self.should_skip_ticker(ticker):
            self.skipped += 1
            logger.debug(f"{ticker}: Skipped (already has sufficient data)")
            return True

        # Fetch prices
        bars = self.fetch_prices(ticker)

        if bars is None:
            self.errors += 1
            return False

        if len(bars) == 0:
            logger.warning(f"{ticker}: No price data available from Polygon")
            self.errors += 1
            return False

        # Store prices
        rows_inserted = await self.store_prices(ticker, bars)
        self.total_rows_inserted += rows_inserted
        self.success += 1

        logger.info(f"{ticker}: Inserted {rows_inserted} price records ({len(bars)} bars fetched)")

        return True

    async def run(
        self,
        tickers: Optional[List[str]] = None,
        limit: Optional[int] = None
    ):
        """Run the backfill process.

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
        logger.info(f"Starting historical price backfill")
        logger.info(f"  Date range: {self.from_date} to {self.to_date} ({self.years} years)")
        logger.info(f"  Total tickers: {total_tickers}")
        logger.info(f"  Resume mode: {self.resume}")
        logger.info("")

        # Process in batches
        for i in range(0, total_tickers, self.batch_size):
            batch = all_tickers[i:i + self.batch_size]
            batch_num = i // self.batch_size + 1
            total_batches = (total_tickers + self.batch_size - 1) // self.batch_size

            logger.info(f"Processing batch {batch_num}/{total_batches} ({len(batch)} tickers)")

            for ticker in batch:
                await self.process_ticker(ticker)

            # Progress update
            elapsed = (datetime.now() - start_time).total_seconds()
            rate = self.processed / elapsed if elapsed > 0 else 0
            eta_seconds = (total_tickers - self.processed) / rate if rate > 0 else 0
            eta_minutes = eta_seconds / 60

            logger.info(
                f"Progress: {self.processed}/{total_tickers} "
                f"({100*self.processed/total_tickers:.1f}%) | "
                f"Success: {self.success} | Skipped: {self.skipped} | Errors: {self.errors} | "
                f"Rate: {rate:.1f}/s | ETA: {eta_minutes:.1f} min"
            )

        await self.close()

        # Final summary
        duration = datetime.now() - start_time
        logger.info("")
        logger.info("=" * 60)
        logger.info("Historical Price Backfill Complete")
        logger.info("=" * 60)
        logger.info(f"Duration: {duration}")
        logger.info(f"Processed: {self.processed}")
        logger.info(f"Success: {self.success}")
        logger.info(f"Skipped: {self.skipped}")
        logger.info(f"Errors: {self.errors}")
        logger.info(f"Total rows inserted: {self.total_rows_inserted:,}")


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description='Backfill historical prices from Polygon.io',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Backfill all tickers with 10 years of data
  python historical_price_backfill.py --all

  # Test on 100 tickers first
  python historical_price_backfill.py --limit 100

  # Backfill single ticker
  python historical_price_backfill.py --ticker AAPL

  # Backfill with 15 years of data
  python historical_price_backfill.py --all --years 15

  # Resume mode - skip tickers with sufficient data
  python historical_price_backfill.py --all --resume

Notes:
  - Requires POLYGON_API_KEY environment variable
  - Rate limited to ~5 requests/second
  - Estimated time: ~2-4 hours for 10,000 tickers
  - Use --resume to continue interrupted backfill
        """
    )

    parser.add_argument(
        '--all',
        action='store_true',
        help='Backfill all tickers'
    )
    parser.add_argument(
        '--ticker',
        type=str,
        help='Backfill single ticker'
    )
    parser.add_argument(
        '--years',
        type=int,
        default=10,
        help='Years of history to fetch (default: 10)'
    )
    parser.add_argument(
        '--limit',
        type=int,
        help='Limit number of tickers'
    )
    parser.add_argument(
        '--resume',
        action='store_true',
        help='Skip tickers that already have sufficient data'
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

    # Create backfill instance
    backfill = HistoricalPriceBackfill(
        years=args.years,
        batch_size=args.batch_size,
        resume=args.resume
    )

    # Run
    if args.ticker:
        asyncio.run(backfill.run(tickers=[args.ticker]))
    else:
        asyncio.run(backfill.run(limit=args.limit))


if __name__ == '__main__':
    main()
