#!/usr/bin/env python3
"""Benchmark Data Ingestion Pipeline.

This script fetches benchmark index data (S&P 500 Total Return, etc.) for
risk-adjusted performance calculations. Supports multiple benchmarks:

- ^SPXTR: S&P 500 Total Return Index (preferred for US stocks)
- ^SPX: S&P 500 Price Index (fallback)
- SPY: S&P 500 ETF (alternative fallback)

Usage:
    python benchmark_data_ingestion.py --backfill  # Fetch 3 years historical
    python benchmark_data_ingestion.py             # Daily incremental update
    python benchmark_data_ingestion.py --symbol SPY --days 30  # Custom
"""

import argparse
import asyncio
import logging
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import List, Optional, Dict, Any

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

import pandas as pd
from sqlalchemy import text
from sqlalchemy.dialects.postgresql import insert as pg_insert
from sqlalchemy.ext.asyncio import AsyncSession

from database.database import get_database
from models import BenchmarkReturn
from pipelines.utils.polygon_client import PolygonClient

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('/app/logs/benchmark_data_ingestion.log')
    ]
)
logger = logging.getLogger(__name__)


class BenchmarkDataIngestion:
    """Ingestion pipeline for benchmark index data."""

    # Benchmark symbols to track (in order of preference)
    BENCHMARKS = [
        {
            'symbol': 'SPY',  # S&P 500 ETF (most reliable on Polygon)
            'name': 'S&P 500 ETF',
            'description': 'S&P 500 ETF as proxy for market benchmark'
        },
        {
            'symbol': '^SPX',
            'name': 'S&P 500 Index',
            'description': 'S&P 500 Price Index'
        }
    ]

    def __init__(self):
        """Initialize the ingestion pipeline."""
        self.polygon_client = PolygonClient()
        self.db = get_database()

        # Track progress
        self.processed_count = 0
        self.success_count = 0
        self.error_count = 0

    async def get_latest_date(self, symbol: str, session: AsyncSession) -> Optional[datetime]:
        """Get the latest date for which we have benchmark data.

        Args:
            symbol: Benchmark symbol.
            session: Database session.

        Returns:
            Latest date or None if no data exists.
        """
        query = text("""
            SELECT MAX(time) as latest_date
            FROM benchmark_returns
            WHERE symbol = :symbol
        """)
        result = await session.execute(query, {"symbol": symbol})
        row = result.fetchone()

        if row and row[0]:
            return row[0]
        return None

    def calculate_daily_return(self, prices: pd.DataFrame) -> pd.DataFrame:
        """Calculate daily returns from prices.

        Args:
            prices: DataFrame with 'date' and 'close' columns.

        Returns:
            DataFrame with added 'daily_return' column.
        """
        if len(prices) < 2:
            prices['daily_return'] = None
            return prices

        # Sort by date
        prices = prices.sort_values('date').reset_index(drop=True)

        # Calculate daily percentage return
        prices['daily_return'] = prices['close'].pct_change() * 100

        # First row has no prior data
        prices.loc[0, 'daily_return'] = 0.0

        return prices

    async def store_benchmark_data(
        self,
        symbol: str,
        prices: List[Dict[str, Any]],
        session: AsyncSession
    ) -> bool:
        """Store benchmark data in database.

        Args:
            symbol: Benchmark symbol.
            prices: List of OHLCV price dictionaries.
            session: Database session.

        Returns:
            True if successful.
        """
        try:
            if not prices:
                return False

            # Convert to DataFrame for calculations
            df = pd.DataFrame(prices)
            df.rename(columns={'c': 'close', 'v': 'volume'}, inplace=True)

            # Calculate daily returns
            df = self.calculate_daily_return(df)

            # Prepare records for database
            records = []
            for _, row in df.iterrows():
                # Convert date to datetime with timezone
                price_date = row.get('date')
                if isinstance(price_date, str):
                    price_datetime = datetime.strptime(price_date, '%Y-%m-%d')
                else:
                    price_datetime = datetime.combine(price_date, datetime.min.time())

                # Add timezone info
                price_datetime = price_datetime.replace(tzinfo=timezone.utc)

                record = {
                    'time': price_datetime,
                    'symbol': symbol,
                    'close': float(row['close']),
                    'total_return': float(row['close']),  # For SPY, close = total return
                    'daily_return': float(row['daily_return']) if pd.notna(row['daily_return']) else 0.0,
                    'volume': int(row['volume']) if pd.notna(row['volume']) else None
                }
                records.append(record)

            if not records:
                return False

            # Insert with ON CONFLICT DO UPDATE
            stmt = pg_insert(BenchmarkReturn.__table__).values(records)
            stmt = stmt.on_conflict_do_update(
                index_elements=['time', 'symbol'],
                set_={
                    'close': stmt.excluded.close,
                    'total_return': stmt.excluded.total_return,
                    'daily_return': stmt.excluded.daily_return,
                    'volume': stmt.excluded.volume,
                }
            )

            await session.execute(stmt)
            await session.commit()

            logger.info(f"{symbol}: Stored {len(records)} records")
            return True

        except Exception as e:
            logger.error(f"{symbol}: Error storing data: {e}", exc_info=True)
            await session.rollback()
            return False

    async def fetch_benchmark_data(
        self,
        symbol: str,
        days: int = 7
    ) -> Optional[List[Dict[str, Any]]]:
        """Fetch benchmark data from Polygon.io.

        Args:
            symbol: Benchmark symbol.
            days: Number of days to fetch.

        Returns:
            List of price bars or None if failed.
        """
        try:
            # Polygon expects symbols without ^
            polygon_symbol = symbol.replace('^', '')

            prices = self.polygon_client.get_daily_prices(polygon_symbol, days=days)

            if not prices:
                logger.warning(f"{symbol}: No data returned from Polygon")
                return None

            logger.info(f"{symbol}: Fetched {len(prices)} days from Polygon")
            return prices

        except Exception as e:
            logger.error(f"{symbol}: Error fetching from Polygon: {e}", exc_info=True)
            return None

    async def process_benchmark(
        self,
        symbol: str,
        backfill: bool = False,
        days: Optional[int] = None
    ) -> bool:
        """Process a single benchmark symbol.

        Args:
            symbol: Benchmark symbol.
            backfill: Whether to fetch full historical data (3 years).
            days: Number of days to fetch (overrides backfill).

        Returns:
            True if successful.
        """
        async with self.db.session() as session:
            try:
                # Determine how many days to fetch
                if days is not None:
                    fetch_days = days
                elif backfill:
                    # 3 years = ~1095 days
                    fetch_days = 1150  # Extra buffer for trading days
                else:
                    # Check latest date in database
                    latest_date = await self.get_latest_date(symbol, session)

                    if latest_date:
                        # Fetch from latest date to now
                        days_since_latest = (datetime.now(timezone.utc) - latest_date).days
                        fetch_days = max(7, days_since_latest + 5)  # Add buffer
                        logger.info(f"{symbol}: Latest data is {days_since_latest} days old, fetching {fetch_days} days")
                    else:
                        # No data exists, fetch 3 years
                        fetch_days = 1150
                        logger.info(f"{symbol}: No existing data, fetching {fetch_days} days")

                # Fetch data
                prices = await self.fetch_benchmark_data(symbol, days=fetch_days)

                if not prices:
                    logger.warning(f"{symbol}: Failed to fetch data")
                    return False

                # Store data
                success = await self.store_benchmark_data(symbol, prices, session)

                if success:
                    logger.info(f"{symbol}: Successfully processed {len(prices)} records")
                    return True
                else:
                    logger.error(f"{symbol}: Failed to store data")
                    return False

            except Exception as e:
                logger.error(f"{symbol}: Error processing: {e}", exc_info=True)
                return False

    async def run(
        self,
        symbol: Optional[str] = None,
        backfill: bool = False,
        days: Optional[int] = None
    ):
        """Run the ingestion pipeline.

        Args:
            symbol: Specific benchmark symbol to process (default: all benchmarks).
            backfill: Fetch 3 years of historical data.
            days: Number of days to fetch (overrides backfill).
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("Benchmark Data Ingestion Pipeline")
        logger.info("=" * 80)

        # Determine which symbols to process
        if symbol:
            symbols = [symbol]
        else:
            symbols = [b['symbol'] for b in self.BENCHMARKS]

        logger.info(f"Processing {len(symbols)} benchmark(s): {', '.join(symbols)}")

        if backfill:
            logger.info("Backfill mode: Fetching 3 years of historical data")

        # Process each benchmark
        for sym in symbols:
            self.processed_count += 1
            success = await self.process_benchmark(sym, backfill=backfill, days=days)

            if success:
                self.success_count += 1
            else:
                self.error_count += 1

        # Print summary
        duration = datetime.now() - start_time
        logger.info("=" * 80)
        logger.info("Processing Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Processed: {self.processed_count}")
        logger.info(f"Success: {self.success_count}")
        logger.info(f"Errors: {self.error_count}")

        # Close clients
        self.polygon_client.close()


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Ingest benchmark index data for risk calculations',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Daily incremental update (default)
  python benchmark_data_ingestion.py

  # Backfill 3 years of historical data
  python benchmark_data_ingestion.py --backfill

  # Fetch specific benchmark
  python benchmark_data_ingestion.py --symbol SPY

  # Fetch last 30 days
  python benchmark_data_ingestion.py --days 30
        """
    )

    parser.add_argument(
        '--symbol',
        type=str,
        help='Specific benchmark symbol to process (default: all)'
    )
    parser.add_argument(
        '--backfill',
        action='store_true',
        help='Fetch 3 years of historical data'
    )
    parser.add_argument(
        '--days',
        type=int,
        help='Number of days to fetch (overrides backfill)'
    )

    args = parser.parse_args()

    # Run pipeline
    ingestion = BenchmarkDataIngestion()
    asyncio.run(ingestion.run(
        symbol=args.symbol,
        backfill=args.backfill,
        days=args.days
    ))


if __name__ == '__main__':
    main()
