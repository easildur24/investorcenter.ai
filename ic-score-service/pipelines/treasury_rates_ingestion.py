#!/usr/bin/env python3
"""Treasury Rates Ingestion Pipeline.

This script fetches US Treasury rates from FRED API (Federal Reserve Economic Data)
for risk-free rate calculations in risk metrics.

FRED API series:
- DGS1MO: 1-Month Treasury Constant Maturity Rate
- DGS3MO: 3-Month Treasury Constant Maturity Rate
- DGS6MO: 6-Month Treasury Constant Maturity Rate
- DGS1: 1-Year Treasury Constant Maturity Rate
- DGS2: 2-Year Treasury Constant Maturity Rate
- DGS10: 10-Year Treasury Constant Maturity Rate

API Documentation: https://fred.stlouisfed.org/docs/api/fred/

Usage:
    python treasury_rates_ingestion.py --backfill  # Fetch 3 years historical
    python treasury_rates_ingestion.py             # Daily incremental update
    python treasury_rates_ingestion.py --days 30   # Last 30 days
"""

import argparse
import asyncio
import logging
import os
import sys
from datetime import date, datetime, timedelta
from pathlib import Path
from typing import List, Optional, Dict, Any

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))

import pandas as pd
import requests
from sqlalchemy import text
from sqlalchemy.dialects.postgresql import insert as pg_insert
from sqlalchemy.ext.asyncio import AsyncSession

from database.database import get_database
from models import TreasuryRate

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('/app/logs/treasury_rates_ingestion.log')
    ]
)
logger = logging.getLogger(__name__)


class TreasuryRatesIngestion:
    """Ingestion pipeline for US Treasury rates from FRED API."""

    # FRED API series IDs for different Treasury maturities
    FRED_SERIES = {
        'rate_1m': 'DGS1MO',   # 1-month
        'rate_3m': 'DGS3MO',   # 3-month (most commonly used for risk-free rate)
        'rate_6m': 'DGS6MO',   # 6-month
        'rate_1y': 'DGS1',     # 1-year
        'rate_2y': 'DGS2',     # 2-year
        'rate_10y': 'DGS10'    # 10-year
    }

    def __init__(self, api_key: Optional[str] = None):
        """Initialize the ingestion pipeline.

        Args:
            api_key: FRED API key (defaults to FRED_API_KEY env var).
        """
        self.api_key = api_key or os.getenv('FRED_API_KEY')
        if not self.api_key:
            raise ValueError("FRED_API_KEY environment variable not set")

        self.db = get_database()
        self.base_url = "https://api.stlouisfed.org/fred/series/observations"

        # Track progress
        self.processed_count = 0
        self.success_count = 0
        self.error_count = 0

    async def get_latest_date(self, session: AsyncSession) -> Optional[date]:
        """Get the latest date for which we have Treasury rate data.

        Args:
            session: Database session.

        Returns:
            Latest date or None if no data exists.
        """
        query = text("""
            SELECT MAX(date) as latest_date
            FROM treasury_rates
        """)
        result = await session.execute(query)
        row = result.fetchone()

        if row and row[0]:
            return row[0]
        return None

    def fetch_fred_series(
        self,
        series_id: str,
        start_date: str,
        end_date: str
    ) -> Optional[pd.DataFrame]:
        """Fetch data from FRED API for a single series.

        Args:
            series_id: FRED series ID (e.g., 'DGS3MO').
            start_date: Start date (YYYY-MM-DD).
            end_date: End date (YYYY-MM-DD).

        Returns:
            DataFrame with 'date' and 'value' columns, or None if failed.
        """
        try:
            params = {
                'series_id': series_id,
                'api_key': self.api_key,
                'file_type': 'json',
                'observation_start': start_date,
                'observation_end': end_date
            }

            response = requests.get(self.base_url, params=params, timeout=30)
            response.raise_for_status()

            data = response.json()

            if 'observations' not in data:
                logger.warning(f"{series_id}: No observations in response")
                return None

            observations = data['observations']

            if not observations:
                logger.warning(f"{series_id}: Empty observations list")
                return None

            # Convert to DataFrame
            df = pd.DataFrame(observations)
            df = df[['date', 'value']]

            # Convert date column to datetime
            df['date'] = pd.to_datetime(df['date']).dt.date

            # Replace '.' with NaN (FRED uses '.' for missing values)
            df['value'] = df['value'].replace('.', pd.NA)

            # Convert to float
            df['value'] = pd.to_numeric(df['value'], errors='coerce')

            logger.info(f"{series_id}: Fetched {len(df)} observations")
            return df

        except requests.exceptions.RequestException as e:
            logger.error(f"{series_id}: HTTP error: {e}")
            return None
        except Exception as e:
            logger.error(f"{series_id}: Error fetching data: {e}", exc_info=True)
            return None

    def merge_all_series(
        self,
        series_data: Dict[str, pd.DataFrame]
    ) -> pd.DataFrame:
        """Merge all Treasury rate series into a single DataFrame.

        Args:
            series_data: Dictionary mapping rate column names to DataFrames.

        Returns:
            Merged DataFrame with all rates by date.
        """
        merged = None

        for rate_col, df in series_data.items():
            if df is None or df.empty:
                continue

            # Rename value column to rate column name
            df = df.rename(columns={'value': rate_col})

            if merged is None:
                merged = df
            else:
                merged = merged.merge(df, on='date', how='outer')

        if merged is None:
            return pd.DataFrame()

        # Sort by date
        merged = merged.sort_values('date').reset_index(drop=True)

        return merged

    async def store_treasury_rates(
        self,
        rates_df: pd.DataFrame,
        session: AsyncSession
    ) -> bool:
        """Store Treasury rates in database.

        Args:
            rates_df: DataFrame with dates and rate columns.
            session: Database session.

        Returns:
            True if successful.
        """
        try:
            if rates_df.empty:
                logger.warning("No data to store")
                return False

            # Prepare records for database
            records = []
            for _, row in rates_df.iterrows():
                record = {
                    'date': row['date'],
                    'rate_1m': float(row['rate_1m']) if pd.notna(row.get('rate_1m')) else None,
                    'rate_3m': float(row['rate_3m']) if pd.notna(row.get('rate_3m')) else None,
                    'rate_6m': float(row['rate_6m']) if pd.notna(row.get('rate_6m')) else None,
                    'rate_1y': float(row['rate_1y']) if pd.notna(row.get('rate_1y')) else None,
                    'rate_2y': float(row['rate_2y']) if pd.notna(row.get('rate_2y')) else None,
                    'rate_10y': float(row['rate_10y']) if pd.notna(row.get('rate_10y')) else None,
                }
                records.append(record)

            if not records:
                return False

            # Insert with ON CONFLICT DO UPDATE
            stmt = pg_insert(TreasuryRate.__table__).values(records)
            stmt = stmt.on_conflict_do_update(
                index_elements=['date'],
                set_={
                    'rate_1m': stmt.excluded.rate_1m,
                    'rate_3m': stmt.excluded.rate_3m,
                    'rate_6m': stmt.excluded.rate_6m,
                    'rate_1y': stmt.excluded.rate_1y,
                    'rate_2y': stmt.excluded.rate_2y,
                    'rate_10y': stmt.excluded.rate_10y,
                }
            )

            await session.execute(stmt)
            await session.commit()

            logger.info(f"Stored {len(records)} Treasury rate records")
            return True

        except Exception as e:
            logger.error(f"Error storing data: {e}", exc_info=True)
            await session.rollback()
            return False

    async def run(
        self,
        backfill: bool = False,
        days: Optional[int] = None
    ):
        """Run the ingestion pipeline.

        Args:
            backfill: Fetch 3 years of historical data.
            days: Number of days to fetch (overrides backfill).
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("Treasury Rates Ingestion Pipeline")
        logger.info("=" * 80)

        async with self.db.session() as session:
            try:
                # Determine date range
                end_date = date.today()

                if days is not None:
                    start_date = end_date - timedelta(days=days)
                    logger.info(f"Fetching last {days} days of data")
                elif backfill:
                    # 3 years
                    start_date = end_date - timedelta(days=1095)
                    logger.info("Backfill mode: Fetching 3 years of data")
                else:
                    # Check latest date in database
                    latest_date = await self.get_latest_date(session)

                    if latest_date:
                        # Fetch from latest date to now
                        start_date = latest_date - timedelta(days=5)  # Small overlap
                        days_to_fetch = (end_date - start_date).days
                        logger.info(f"Latest data is from {latest_date}, fetching {days_to_fetch} days")
                    else:
                        # No data exists, fetch 3 years
                        start_date = end_date - timedelta(days=1095)
                        logger.info("No existing data, fetching 3 years")

                # Convert dates to strings
                start_date_str = start_date.strftime('%Y-%m-%d')
                end_date_str = end_date.strftime('%Y-%m-%d')

                logger.info(f"Date range: {start_date_str} to {end_date_str}")

                # Fetch all series
                series_data = {}
                for rate_col, series_id in self.FRED_SERIES.items():
                    df = self.fetch_fred_series(series_id, start_date_str, end_date_str)
                    series_data[rate_col] = df
                    self.processed_count += 1

                    if df is not None and not df.empty:
                        self.success_count += 1
                    else:
                        self.error_count += 1

                # Merge all series
                merged_df = self.merge_all_series(series_data)

                if merged_df.empty:
                    logger.warning("No data fetched from FRED API")
                    return

                logger.info(f"Merged {len(merged_df)} total records")

                # Store in database
                success = await self.store_treasury_rates(merged_df, session)

                if success:
                    logger.info("Successfully stored Treasury rates")
                else:
                    logger.error("Failed to store Treasury rates")

            except Exception as e:
                logger.error(f"Error in pipeline: {e}", exc_info=True)

        # Print summary
        duration = datetime.now() - start_time
        logger.info("=" * 80)
        logger.info("Processing Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Series Processed: {self.processed_count}")
        logger.info(f"Success: {self.success_count}")
        logger.info(f"Errors: {self.error_count}")


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Ingest US Treasury rates from FRED API',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Daily incremental update (default)
  python treasury_rates_ingestion.py

  # Backfill 3 years of historical data
  python treasury_rates_ingestion.py --backfill

  # Fetch last 30 days
  python treasury_rates_ingestion.py --days 30

Environment Variables:
  FRED_API_KEY: FRED API key (required)
                Get your free API key at: https://fred.stlouisfed.org/docs/api/api_key.html
        """
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
    ingestion = TreasuryRatesIngestion()
    asyncio.run(ingestion.run(
        backfill=args.backfill,
        days=args.days
    ))


if __name__ == '__main__':
    main()
