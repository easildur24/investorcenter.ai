#!/usr/bin/env python3
"""SEC Insider Trades Ingestion Pipeline.

This script fetches insider trading data from SEC Form 4 filings via RSS feed,
downloads and parses the Form 4 XML, and populates the insider_trades table.

Usage:
    python sec_insider_trades_ingestion.py --hours 24     # Last 24 hours
    python sec_insider_trades_ingestion.py --backfill 90  # Last 90 days
"""

import argparse
import asyncio
import logging
import os
import re
import sys
import time
import xml.etree.ElementTree as ET
from datetime import datetime, timedelta, date
from decimal import Decimal
from pathlib import Path
from typing import List, Optional, Dict, Any

sys.path.insert(0, str(Path(__file__).parent.parent))

import requests
from sqlalchemy import text
from sqlalchemy.dialects.postgresql import insert as pg_insert
from tqdm import tqdm

from database.database import get_database
from models import InsiderTrade

# Setup logging with configurable log directory
LOG_DIR = os.environ.get('LOG_DIR', '/app/logs')
Path(LOG_DIR).mkdir(parents=True, exist_ok=True)

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler(os.path.join(LOG_DIR, 'sec_insider_trades_ingestion.log'))
    ]
)
logger = logging.getLogger(__name__)


class InsiderTradesIngestion:
    """Ingestion pipeline for SEC Form 4 insider trades."""

    RSS_URL = "https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=4&output=atom&count=100"
    EDGAR_BASE_URL = "https://www.sec.gov/Archives/edgar/data"
    USER_AGENT = "InvestorCenter.ai admin@investorcenter.ai"

    # Rate limiting: 10 requests/second per SEC guidelines
    MIN_REQUEST_INTERVAL = 0.1  # 100ms between requests

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

    def _rate_limit(self):
        """Implement rate limiting for SEC requests."""
        current_time = time.time()
        elapsed = current_time - self.last_request_time
        if elapsed < self.MIN_REQUEST_INTERVAL:
            time.sleep(self.MIN_REQUEST_INTERVAL - elapsed)
        self.last_request_time = time.time()

    def fetch_form4_filings(self, hours_back: int = 24) -> List[Dict[str, Any]]:
        """Fetch Form 4 filings from SEC RSS feed.

        Args:
            hours_back: How many hours back to fetch filings.

        Returns:
            List of filing dictionaries.
        """
        try:
            self._rate_limit()
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

                # Parse timestamp (convert to naive datetime for comparison)
                try:
                    updated_time = datetime.fromisoformat(updated.text.replace('Z', '+00:00')).replace(tzinfo=None)
                except:
                    continue

                if updated_time < cutoff_time:
                    continue

                filing = {
                    'title': title.text,
                    'url': link.get('href'),
                    
                    'updated': updated_time,
                }

                # Extract ticker from title (format: "4 - TICKER - NAME (Issuer)")
                parts = title.text.split(' - ')
                if len(parts) >= 2:
                    ticker = parts[1].strip()
                    # Clean ticker (remove any parenthetical info)
                    ticker = ticker.split('(')[0].strip()
                    if ticker and len(ticker) <= 10:
                        filing['ticker'] = ticker.upper()

                filings.append(filing)

            logger.info(f"Fetched {len(filings)} Form 4 filings from RSS feed")
            return filings

        except Exception as e:
            logger.error(f"Error fetching Form 4 filings: {e}")
            return []

    def parse_form4_xml(self, xml_content: str, filing_url: str) -> List[Dict[str, Any]]:
        """Parse Form 4 XML to extract transaction details.

        Args:
            xml_content: Form 4 XML content.
            filing_url: URL of the filing.

        Returns:
            List of transaction dictionaries.
        """
        transactions = []

        try:
            root = ET.fromstring(xml_content)

            # Extract issuer ticker
            issuer_ticker = None
            issuer_elem = root.find('.//issuer/issuerTradingSymbol')
            if issuer_elem is not None and issuer_elem.text:
                issuer_ticker = issuer_elem.text.strip().upper()

            if not issuer_ticker:
                return []

            # Extract reporting owner info
            owner_name = None
            owner_title = None

            owner_elem = root.find('.//reportingOwner/reportingOwnerId/rptOwnerName')
            if owner_elem is not None and owner_elem.text:
                owner_name = owner_elem.text.strip()

            # Get relationship/title
            relationship = root.find('.//reportingOwner/reportingOwnerRelationship')
            if relationship is not None:
                if relationship.find('isDirector') is not None and relationship.find('isDirector').text == '1':
                    owner_title = 'Director'
                elif relationship.find('isOfficer') is not None and relationship.find('isOfficer').text == '1':
                    title_elem = relationship.find('officerTitle')
                    owner_title = title_elem.text.strip() if title_elem is not None and title_elem.text else 'Officer'
                elif relationship.find('isTenPercentOwner') is not None and relationship.find('isTenPercentOwner').text == '1':
                    owner_title = '10% Owner'
                elif relationship.find('isOther') is not None and relationship.find('isOther').text == '1':
                    owner_title = 'Other'

            if not owner_name:
                return []

            # Parse non-derivative transactions
            for trans in root.findall('.//nonDerivativeTransaction'):
                tx = self._parse_transaction(trans, issuer_ticker, owner_name, owner_title, filing_url, is_derivative=False)
                if tx:
                    transactions.append(tx)

            # Parse derivative transactions
            for trans in root.findall('.//derivativeTransaction'):
                tx = self._parse_transaction(trans, issuer_ticker, owner_name, owner_title, filing_url, is_derivative=True)
                if tx:
                    transactions.append(tx)

        except ET.ParseError as e:
            logger.warning(f"XML parse error: {e}")
        except Exception as e:
            logger.warning(f"Error parsing Form 4: {e}")

        return transactions

    def _parse_transaction(
        self,
        trans_elem: ET.Element,
        ticker: str,
        owner_name: str,
        owner_title: Optional[str],
        filing_url: str,
        is_derivative: bool
    ) -> Optional[Dict[str, Any]]:
        """Parse a single transaction element.

        Args:
            trans_elem: Transaction XML element.
            ticker: Stock ticker.
            owner_name: Insider name.
            owner_title: Insider title.
            filing_url: Filing URL.
            is_derivative: Whether this is a derivative transaction.

        Returns:
            Transaction dictionary or None.
        """
        try:
            # Get transaction date
            trans_date = None
            date_elem = trans_elem.find('.//transactionDate/value')
            if date_elem is not None and date_elem.text:
                try:
                    trans_date = datetime.strptime(date_elem.text.strip(), '%Y-%m-%d').date()
                except:
                    trans_date = date.today()
            else:
                trans_date = date.today()

            # Get transaction code (A=Award, P=Purchase, S=Sale, etc.)
            trans_code = None
            code_elem = trans_elem.find('.//transactionCoding/transactionCode')
            if code_elem is not None and code_elem.text:
                trans_code = code_elem.text.strip()

            # Map transaction codes to types
            trans_type_map = {
                'P': 'Purchase',
                'S': 'Sale',
                'A': 'Award',
                'D': 'Disposition',
                'F': 'Tax Payment',
                'I': 'Discretionary',
                'M': 'Exercise',
                'C': 'Conversion',
                'E': 'Expiration',
                'G': 'Gift',
                'H': 'Expiration',
                'J': 'Other',
                'K': 'Equity Swap',
                'L': 'Small Acquisition',
                'W': 'Will/Inheritance',
                'Z': 'Trust',
            }
            transaction_type = trans_type_map.get(trans_code, f'Other ({trans_code})')

            # Get shares
            shares = 0
            shares_elem = trans_elem.find('.//transactionAmounts/transactionShares/value')
            if shares_elem is not None and shares_elem.text:
                try:
                    shares = int(float(shares_elem.text.strip()))
                except:
                    shares = 0

            if shares == 0:
                return None

            # Get price per share
            price = None
            price_elem = trans_elem.find('.//transactionAmounts/transactionPricePerShare/value')
            if price_elem is not None and price_elem.text:
                try:
                    price = Decimal(price_elem.text.strip())
                except:
                    price = None

            # Get acquired/disposed indicator
            acq_disp = None
            ad_elem = trans_elem.find('.//transactionAmounts/transactionAcquiredDisposedCode/value')
            if ad_elem is not None and ad_elem.text:
                acq_disp = ad_elem.text.strip()

            # Adjust shares sign based on acquired/disposed
            if acq_disp == 'D':  # Disposed
                shares = -abs(shares)
            else:  # Acquired
                shares = abs(shares)

            # Get shares owned after transaction
            shares_after = None
            after_elem = trans_elem.find('.//postTransactionAmounts/sharesOwnedFollowingTransaction/value')
            if after_elem is not None and after_elem.text:
                try:
                    shares_after = int(float(after_elem.text.strip()))
                except:
                    shares_after = None

            # Calculate total value
            total_value = None
            if price and shares:
                total_value = int(abs(price * shares))

            return {
                'ticker': ticker,
                'filing_date': date.today(),
                'transaction_date': trans_date,
                'insider_name': owner_name[:255] if owner_name else 'Unknown',
                'insider_title': owner_title[:255] if owner_title else None,
                'transaction_type': transaction_type,
                'shares': shares,
                'price_per_share': price,
                'total_value': total_value,
                'shares_owned_after': shares_after,
                'is_derivative': is_derivative,
                'form_type': '4',
                'sec_filing_url': filing_url,
            }

        except Exception as e:
            logger.warning(f"Error parsing transaction: {e}")
            return None

    def fetch_form4_details(self, filing_url: str) -> List[Dict[str, Any]]:
        """Fetch and parse Form 4 filing details.

        Args:
            filing_url: URL to the filing index page.

        Returns:
            List of transaction dictionaries.
        """
        try:
            self._rate_limit()

            # Fetch the filing index page
            response = self.session.get(filing_url, timeout=30)
            response.raise_for_status()

            content = response.text

            # Find all XML file links, excluding XSL-transformed versions
            # The actual XML is at paths like: /Archives/edgar/data/CIK/ACCESSION/filename.xml
            # XSL-transformed versions have xslF345X05 in the path - skip those
            xml_url = None

            # Find all href attributes pointing to .xml files
            xml_matches = re.findall(r'href="([^"]*\.xml)"', content, re.IGNORECASE)

            for xml_path in xml_matches:
                # Skip XSL stylesheet transforms (these render as HTML, not valid Form 4 XML)
                if 'xsl' in xml_path.lower():
                    continue

                # Found a raw XML file
                if xml_path.startswith('/'):
                    xml_url = f"https://www.sec.gov{xml_path}"
                elif xml_path.startswith('http'):
                    xml_url = xml_path
                else:
                    # Relative URL - build from filing URL
                    base_url = '/'.join(filing_url.split('/')[:-1])
                    xml_url = f"{base_url}/{xml_path}"
                break

            if not xml_url:
                logger.debug(f"Could not find XML document in {filing_url}")
                return []

            # Fetch the XML document
            self._rate_limit()
            xml_response = self.session.get(xml_url, timeout=30)
            xml_response.raise_for_status()

            # Parse the Form 4 XML
            transactions = self.parse_form4_xml(xml_response.text, filing_url)
            return transactions

        except requests.exceptions.RequestException as e:
            logger.debug(f"Request error for {filing_url}: {e}")
            return []
        except Exception as e:
            logger.debug(f"Error processing {filing_url}: {e}")
            return []

    async def get_known_tickers(self) -> set:
        """Get set of known tickers from database."""
        async with self.db.session() as session:
            result = await session.execute(text("SELECT symbol FROM tickers WHERE active = true"))
            return {row[0] for row in result.fetchall()}

    async def store_insider_trades(self, trades: List[Dict[str, Any]]) -> int:
        """Store insider trades in database.

        Args:
            trades: List of insider trade dictionaries.

        Returns:
            Number of trades stored.
        """
        if not trades:
            return 0

        stored = 0
        try:
            async with self.db.session() as session:
                for trade in trades:
                    try:
                        # Insert with ON CONFLICT DO NOTHING (based on unique constraint)
                        stmt = pg_insert(InsiderTrade).values(trade)
                        stmt = stmt.on_conflict_do_nothing()
                        result = await session.execute(stmt)
                        if result.rowcount > 0:
                            stored += 1
                    except Exception as e:
                        logger.debug(f"Error inserting trade: {e}")
                        continue

                await session.commit()
                return stored

        except Exception as e:
            logger.error(f"Error storing insider trades: {e}", exc_info=True)
            return 0

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

        # Get known tickers
        known_tickers = await self.get_known_tickers()
        logger.info(f"Loaded {len(known_tickers)} known tickers")

        # Fetch Form 4 filings from RSS
        filings = self.fetch_form4_filings(hours_back=hours)

        if not filings:
            logger.warning("No filings found in RSS feed")
            return

        logger.info(f"Processing {len(filings)} filings...")

        all_trades = []

        for filing in tqdm(filings, desc="Processing Form 4 filings"):
            try:
                # Fetch and parse the Form 4 XML
                # NOTE: Ticker is extracted from XML, not RSS title (RSS has company names, not tickers)
                transactions = self.fetch_form4_details(filing['url'])

                for tx in transactions:
                    # Only include if ticker is in our universe
                    ticker = tx.get('ticker')
                    if ticker and ticker in known_tickers:
                        all_trades.append(tx)
                        self.success_count += 1
                    elif ticker:
                        logger.debug(f"Skipping {ticker} - not in known tickers")

                if transactions:
                    self.processed_count += 1

            except Exception as e:
                logger.debug(f"Error processing filing: {e}")
                self.error_count += 1
                continue

        # Store all trades
        stored_count = 0
        if all_trades:
            logger.info(f"Storing {len(all_trades)} insider trades...")
            stored_count = await self.store_insider_trades(all_trades)

        # Print summary
        duration = datetime.now() - start_time
        logger.info("=" * 80)
        logger.info("Ingestion Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Filings processed: {self.processed_count}")
        logger.info(f"Transactions found: {len(all_trades)}")
        logger.info(f"Transactions stored: {stored_count}")
        logger.info(f"Errors: {self.error_count}")


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Ingest SEC Form 4 insider trades',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Fetch last 24 hours of filings
  python sec_insider_trades_ingestion.py --hours 24

  # Backfill last 7 days
  python sec_insider_trades_ingestion.py --backfill 7
        """
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
