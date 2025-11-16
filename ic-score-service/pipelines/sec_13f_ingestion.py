#!/usr/bin/env python3
"""SEC Form 13F Institutional Holdings Ingestion Pipeline.

This script downloads quarterly 13F bulk files from SEC and populates the
institutional_holdings table with institutional ownership data.

Usage:
    python sec_13f_ingestion.py --quarter 2024Q3      # Specific quarter
    python sec_13f_ingestion.py --backfill 4          # Last 4 quarters
    python sec_13f_ingestion.py --latest              # Latest quarter
"""

import argparse
import asyncio
import csv
import logging
import sys
import time
from datetime import datetime, date, timedelta
from io import StringIO
from pathlib import Path
from typing import List, Optional, Dict, Any

sys.path.insert(0, str(Path(__file__).parent.parent))

import requests
from sqlalchemy import text
from sqlalchemy.dialects.postgresql import insert as pg_insert
from tqdm import tqdm

from database.database import get_database
from models import InstitutionalHolding

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('/app/logs/sec_13f_ingestion.log')
    ]
)
logger = logging.getLogger(__name__)


class SEC13FIngestion:
    """Ingestion pipeline for SEC Form 13F institutional holdings."""

    # SEC 13F bulk data URLs
    BASE_URL = "https://www.sec.gov/files/structureddata/data/form-13f-data-sets"
    USER_AGENT = "InvestorCenter.ai admin@investorcenter.ai"

    # Rate limiting: 10 requests/second
    REQUESTS_PER_SECOND = 10
    MIN_REQUEST_INTERVAL = 1.0 / REQUESTS_PER_SECOND

    def __init__(self):
        """Initialize the ingestion pipeline."""
        self.db = get_database()
        self.session = requests.Session()
        self.session.headers.update({'User-Agent': self.USER_AGENT})
        self.last_request_time = 0.0

        # Track progress
        self.processed_count = 0
        self.success_count = 0
        self.error_count = 0

        # CIK to ticker mapping cache
        self.cik_to_ticker = {}

    def _rate_limit(self):
        """Implement rate limiting for SEC requests."""
        current_time = time.time()
        elapsed = current_time - self.last_request_time

        if elapsed < self.MIN_REQUEST_INTERVAL:
            sleep_time = self.MIN_REQUEST_INTERVAL - elapsed
            time.sleep(sleep_time)

        self.last_request_time = time.time()

    async def load_cik_ticker_mapping(self):
        """Load CIK to ticker mapping from stocks table."""
        async with self.db.session() as session:
            query = text("""
                SELECT cik, ticker
                FROM stocks
                WHERE cik IS NOT NULL
            """)
            result = await session.execute(query)
            rows = result.fetchall()

            for row in rows:
                cik = str(row[0]).zfill(10)  # Pad CIK to 10 digits
                self.cik_to_ticker[cik] = row[1]

        logger.info(f"Loaded {len(self.cik_to_ticker)} CIK to ticker mappings")

    def parse_quarter(self, quarter_str: str) -> date:
        """Parse quarter string (e.g., '2024Q3') to quarter end date.

        Args:
            quarter_str: Quarter string in format YYYYQN.

        Returns:
            Quarter end date.
        """
        year = int(quarter_str[:4])
        quarter = int(quarter_str[-1])

        # Quarter end dates
        quarter_ends = {
            1: date(year, 3, 31),
            2: date(year, 6, 30),
            3: date(year, 9, 30),
            4: date(year, 12, 31),
        }

        return quarter_ends[quarter]

    def get_latest_quarter(self) -> str:
        """Get the latest completed quarter.

        Returns:
            Quarter string (e.g., '2024Q3').
        """
        today = date.today()
        year = today.year
        month = today.month

        # 13F filings are due 45 days after quarter end
        # So we need to go back ~3 months from today
        if month <= 2:
            # In Q1, latest available is likely Q3 of previous year
            return f"{year - 1}Q3"
        elif month <= 5:
            # In Q2, latest available is Q4 of previous year
            return f"{year - 1}Q4"
        elif month <= 8:
            # In Q3, latest available is Q1 of current year
            return f"{year}Q1"
        else:
            # In Q4, latest available is Q2 of current year
            return f"{year}Q2"

    def download_13f_data(self, quarter: str) -> Optional[str]:
        """Download 13F bulk data file for a quarter.

        Args:
            quarter: Quarter string (e.g., '2024Q3').

        Returns:
            CSV data as string, or None if download fails.
        """
        # Construct URL
        # Note: Actual SEC 13F bulk data format may vary
        # This is a placeholder implementation
        year = quarter[:4]
        q = quarter[-1]

        # Example URL pattern (actual may differ)
        url = f"{self.BASE_URL}/{year}q{q}/form13f-{year}q{q}.txt"

        self._rate_limit()

        try:
            logger.info(f"Downloading 13F data for {quarter} from {url}")
            response = self.session.get(url, timeout=60)
            response.raise_for_status()

            logger.info(f"Successfully downloaded 13F data for {quarter}")
            return response.text

        except requests.exceptions.HTTPError as e:
            if e.response.status_code == 404:
                logger.warning(f"13F data not found for {quarter}")
            else:
                logger.error(f"HTTP error downloading {quarter}: {e}")
            return None
        except requests.exceptions.RequestException as e:
            logger.error(f"Request error downloading {quarter}: {e}")
            return None
        except Exception as e:
            logger.exception(f"Unexpected error downloading {quarter}: {e}")
            return None

    def parse_13f_data(
        self,
        csv_data: str,
        quarter_end_date: date
    ) -> List[Dict[str, Any]]:
        """Parse 13F CSV data into structured holdings.

        Args:
            csv_data: Raw CSV data string.
            quarter_end_date: Quarter end date.

        Returns:
            List of holding dictionaries.
        """
        holdings = []

        try:
            # Parse CSV
            reader = csv.DictReader(StringIO(csv_data), delimiter='\t')

            for row in reader:
                # Extract institution CIK and name
                # Note: Actual CSV format may vary
                institution_cik = row.get('FILING_MANAGER_CIK', '').zfill(10)
                institution_name = row.get('FILING_MANAGER_NAME', 'Unknown')

                # Extract company CIK
                company_cik = row.get('ISSUER_CIK', '').zfill(10)

                # Map to ticker
                ticker = self.cik_to_ticker.get(company_cik)
                if not ticker:
                    continue  # Skip if we don't track this stock

                # Extract holdings data
                shares = int(row.get('SHARES', 0))
                market_value = int(row.get('VALUE', 0)) * 1000  # Value in thousands

                # Create holding record
                holding = {
                    'ticker': ticker,
                    'filing_date': quarter_end_date,  # Approximate
                    'quarter_end_date': quarter_end_date,
                    'institution_name': institution_name,
                    'institution_cik': institution_cik,
                    'shares': shares,
                    'market_value': market_value,
                    'position_change': 'Unknown',  # Calculate from previous quarter
                }

                holdings.append(holding)

        except Exception as e:
            logger.error(f"Error parsing 13F data: {e}", exc_info=True)

        logger.info(f"Parsed {len(holdings)} holdings from 13F data")
        return holdings

    async def calculate_quarter_changes(
        self,
        holdings: List[Dict[str, Any]],
        quarter_end_date: date
    ):
        """Calculate quarter-over-quarter changes in holdings.

        Args:
            holdings: List of current quarter holdings.
            quarter_end_date: Current quarter end date.
        """
        # Get previous quarter data
        async with self.db.session() as session:
            for holding in tqdm(holdings, desc="Calculating QoQ changes"):
                ticker = holding['ticker']
                institution_cik = holding['institution_cik']

                query = text("""
                    SELECT shares, market_value
                    FROM institutional_holdings
                    WHERE ticker = :ticker
                      AND institution_cik = :institution_cik
                      AND quarter_end_date < :current_quarter
                    ORDER BY quarter_end_date DESC
                    LIMIT 1
                """)

                result = await session.execute(query, {
                    'ticker': ticker,
                    'institution_cik': institution_cik,
                    'current_quarter': quarter_end_date
                })
                prev_holding = result.fetchone()

                if prev_holding:
                    prev_shares = prev_holding[0]
                    prev_value = prev_holding[1]

                    shares_change = holding['shares'] - prev_shares
                    holding['shares_change'] = shares_change

                    if prev_shares > 0:
                        holding['percent_change'] = (shares_change / prev_shares) * 100
                    else:
                        holding['percent_change'] = None

                    # Determine position change
                    if shares_change > 0:
                        if prev_shares == 0:
                            holding['position_change'] = 'New'
                        else:
                            holding['position_change'] = 'Increased'
                    elif shares_change < 0:
                        if holding['shares'] == 0:
                            holding['position_change'] = 'Sold Out'
                        else:
                            holding['position_change'] = 'Decreased'
                    else:
                        holding['position_change'] = 'Unchanged'
                else:
                    # New position
                    holding['position_change'] = 'New'
                    holding['shares_change'] = holding['shares']
                    holding['percent_change'] = None

    async def store_holdings(self, holdings: List[Dict[str, Any]]) -> bool:
        """Store institutional holdings in database.

        Args:
            holdings: List of holding dictionaries.

        Returns:
            True if successful.
        """
        if not holdings:
            return False

        try:
            async with self.db.session() as session:
                # Insert with ON CONFLICT DO UPDATE
                stmt = pg_insert(InstitutionalHolding).values(holdings)
                stmt = stmt.on_conflict_do_update(
                    index_elements=['ticker', 'quarter_end_date', 'institution_cik'],
                    set_={
                        'shares': stmt.excluded.shares,
                        'market_value': stmt.excluded.market_value,
                        'position_change': stmt.excluded.position_change,
                        'shares_change': stmt.excluded.shares_change,
                        'percent_change': stmt.excluded.percent_change,
                    }
                )

                await session.execute(stmt)
                await session.commit()

                logger.info(f"Stored {len(holdings)} institutional holdings")
                return True

        except Exception as e:
            logger.error(f"Error storing holdings: {e}", exc_info=True)
            return False

    async def process_quarter(self, quarter: str) -> bool:
        """Process 13F data for a quarter.

        Args:
            quarter: Quarter string (e.g., '2024Q3').

        Returns:
            True if successful.
        """
        logger.info(f"Processing 13F data for {quarter}")

        # Parse quarter end date
        quarter_end_date = self.parse_quarter(quarter)

        # Download 13F data
        csv_data = self.download_13f_data(quarter)

        if not csv_data:
            logger.warning(f"No data available for {quarter}")
            return False

        # Parse holdings
        holdings = self.parse_13f_data(csv_data, quarter_end_date)

        if not holdings:
            logger.warning(f"No holdings parsed for {quarter}")
            return False

        # Calculate quarter-over-quarter changes
        await self.calculate_quarter_changes(holdings, quarter_end_date)

        # Store in database
        success = await self.store_holdings(holdings)

        return success

    async def run(
        self,
        quarter: Optional[str] = None,
        backfill_quarters: int = 0,
        latest: bool = False
    ):
        """Run the ingestion pipeline.

        Args:
            quarter: Specific quarter to process.
            backfill_quarters: Number of quarters to backfill.
            latest: Process latest available quarter.
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("SEC Form 13F Institutional Holdings Ingestion Pipeline")
        logger.info("=" * 80)

        # Load CIK to ticker mapping
        await self.load_cik_ticker_mapping()

        # Determine quarters to process
        quarters = []

        if latest or not quarter:
            quarters.append(self.get_latest_quarter())
        elif quarter:
            quarters.append(quarter)

        if backfill_quarters > 0:
            # Generate list of previous quarters
            latest_q = quarters[0] if quarters else self.get_latest_quarter()
            year = int(latest_q[:4])
            q = int(latest_q[-1])

            for i in range(backfill_quarters):
                q -= 1
                if q < 1:
                    q = 4
                    year -= 1
                quarters.append(f"{year}Q{q}")

        logger.info(f"Processing {len(quarters)} quarters: {quarters}")

        # Process each quarter
        for quarter in quarters:
            success = await self.process_quarter(quarter)

            self.processed_count += 1
            if success:
                self.success_count += 1
            else:
                self.error_count += 1

        # Print summary
        duration = datetime.now() - start_time
        logger.info("=" * 80)
        logger.info("Ingestion Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Processed: {self.processed_count} quarters")
        logger.info(f"Success: {self.success_count}")
        logger.info(f"Errors: {self.error_count}")


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Ingest SEC Form 13F institutional holdings',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Process specific quarter
  python sec_13f_ingestion.py --quarter 2024Q3

  # Process latest quarter
  python sec_13f_ingestion.py --latest

  # Backfill last 4 quarters
  python sec_13f_ingestion.py --backfill 4
        """
    )

    parser.add_argument(
        '--quarter',
        type=str,
        help='Specific quarter to process (e.g., 2024Q3)'
    )
    parser.add_argument(
        '--backfill',
        type=int,
        default=0,
        help='Backfill N previous quarters'
    )
    parser.add_argument(
        '--latest',
        action='store_true',
        help='Process latest available quarter'
    )

    args = parser.parse_args()

    # Validate arguments
    if not args.quarter and not args.latest and args.backfill == 0:
        args.latest = True  # Default to latest

    # Run pipeline
    pipeline = SEC13FIngestion()
    asyncio.run(pipeline.run(
        quarter=args.quarter,
        backfill_quarters=args.backfill,
        latest=args.latest
    ))


if __name__ == '__main__':
    main()
