#!/usr/bin/env python3
"""SEC Insider Trades Ingestion Pipeline.

This script fetches insider trading data from SEC Form 4 filings via RSS feed
and populates the insider_trades table.

Usage:
    python sec_insider_trades_ingestion.py --hours 24     # Last 24 hours
    python sec_insider_trades_ingestion.py --backfill 90  # Last 90 days
"""

import argparse
import asyncio
import logging
import sys
import xml.etree.ElementTree as ET
from datetime import datetime, timedelta
from pathlib import Path
from typing import List, Optional, Dict, Any

sys.path.insert(0, str(Path(__file__).parent.parent))

import requests
from sqlalchemy.dialects.postgresql import insert as pg_insert
from tqdm import tqdm

from database.database import get_database
from models import InsiderTrade

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('/app/logs/sec_insider_trades_ingestion.log')
    ]
)
logger = logging.getLogger(__name__)


class InsiderTradesIngestion:
    """Ingestion pipeline for SEC Form 4 insider trades."""

    RSS_URL = "https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=4&output=atom&count=100"
    USER_AGENT = "InvestorCenter.ai admin@investorcenter.ai"

    def __init__(self):
        """Initialize the ingestion pipeline."""
        self.db = get_database()
        self.session = requests.Session()
        self.session.headers.update({'User-Agent': self.USER_AGENT})

        # Track progress
        self.processed_count = 0
        self.success_count = 0
        self.error_count = 0

    def fetch_form4_filings(self, hours_back: int = 24) -> List[Dict[str, Any]]:
        """Fetch Form 4 filings from SEC RSS feed.

        Args:
            hours_back: How many hours back to fetch filings.

        Returns:
            List of filing dictionaries.
        """
        try:
            response = self.session.get(self.RSS_URL, timeout=30)
            response.raise_for_status()

            # Parse XML
            root = ET.fromstring(response.content)
            ns = {'atom': 'http://www.w3.org/2005/Atom'}

            filings = []
            cutoff_time = datetime.now() - timedelta(hours=hours_back)

            for entry in root.findall('atom:entry', ns):
                # Extract filing data
                title = entry.find('atom:title', ns)
                link = entry.find('atom:link', ns)
                updated = entry.find('atom:updated', ns)

                if title is None or link is None or updated is None:
                    continue

                # Parse timestamp
                updated_time = datetime.fromisoformat(updated.text.replace('Z', '+00:00'))
                if updated_time < cutoff_time:
                    continue

                filing = {
                    'title': title.text,
                    'url': link.get('href'),
                    'updated': updated_time,
                }

                # Extract ticker from title (format: "4 - TICKER - ...")
                parts = title.text.split(' - ')
                if len(parts) >= 2:
                    filing['ticker'] = parts[1].strip()

                filings.append(filing)

            logger.info(f"Fetched {len(filings)} Form 4 filings")
            return filings

        except Exception as e:
            logger.error(f"Error fetching Form 4 filings: {e}")
            return []

    async def store_insider_trades(
        self,
        trades: List[Dict[str, Any]]
    ) -> bool:
        """Store insider trades in database.

        Args:
            trades: List of insider trade dictionaries.

        Returns:
            True if successful.
        """
        if not trades:
            return False

        try:
            async with self.db.session() as session:
                # Insert with ON CONFLICT DO NOTHING
                stmt = pg_insert(InsiderTrade).values(trades)
                stmt = stmt.on_conflict_do_nothing()

                await session.execute(stmt)
                await session.commit()

                logger.info(f"Stored {len(trades)} insider trades")
                return True

        except Exception as e:
            logger.error(f"Error storing insider trades: {e}", exc_info=True)
            return False

    async def run(self, hours: int = 24, backfill_days: int = 0):
        """Run the ingestion pipeline.

        Args:
            hours: Hours back to fetch filings.
            backfill_days: Days to backfill (overrides hours).
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("SEC Insider Trades Ingestion Pipeline")
        logger.info("=" * 80)

        if backfill_days > 0:
            hours = backfill_days * 24

        # Fetch Form 4 filings
        filings = self.fetch_form4_filings(hours_back=hours)

        if not filings:
            logger.warning("No filings to process")
            return

        logger.info(f"Processing {len(filings)} filings...")

        # Note: Full Form 4 XML parsing is complex and requires detailed implementation
        # This is a placeholder that stores basic filing metadata
        # Production implementation should parse full Form 4 XML for transaction details

        trades = []
        for filing in tqdm(filings, desc="Processing filings"):
            # Placeholder: In production, fetch and parse full Form 4 XML
            # For now, store basic metadata
            if 'ticker' in filing:
                trade = {
                    'ticker': filing['ticker'],
                    'filing_date': filing['updated'].date(),
                    'transaction_date': filing['updated'].date(),
                    'insider_name': 'Pending Full Implementation',
                    'shares': 0,
                    'sec_filing_url': filing['url'],
                }
                trades.append(trade)

        # Store trades
        if trades:
            await self.store_insider_trades(trades)

        # Print summary
        duration = datetime.now() - start_time
        logger.info("=" * 80)
        logger.info("Ingestion Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Processed: {len(filings)} filings")
        logger.info(f"Stored: {len(trades)} trades")


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Ingest SEC Form 4 insider trades',
        formatter_class=argparse.RawDescriptionHelpFormatter
    )

    parser.add_argument(
        '--hours',
        type=int,
        default=24,
        help='Fetch filings from last N hours (default: 24)'
    )
    parser.add_argument(
        '--backfill',
        type=int,
        default=0,
        help='Backfill N days of data'
    )

    args = parser.parse_args()

    # Run pipeline
    pipeline = InsiderTradesIngestion()
    asyncio.run(pipeline.run(hours=args.hours, backfill_days=args.backfill))


if __name__ == '__main__':
    main()
