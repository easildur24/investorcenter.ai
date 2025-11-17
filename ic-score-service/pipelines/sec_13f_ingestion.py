#!/usr/bin/env python3
"""SEC Form 13F Institutional Holdings Ingestion Pipeline.

This script fetches 13F filings from major institutional investors using the
SEC EDGAR API and populates the institutional_holdings table.

Usage:
    python sec_13f_ingestion.py --latest              # Latest quarter from top institutions
    python sec_13f_ingestion.py --limit 50            # Top 50 institutions
    python sec_13f_ingestion.py --cik 0001067983      # Single institution (Berkshire)
"""

import argparse
import asyncio
import logging
import sys
import time
import xml.etree.ElementTree as ET
from datetime import datetime, date
from pathlib import Path
from typing import List, Optional, Dict, Any, Tuple

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
    """Ingestion pipeline for SEC Form 13F institutional holdings using EDGAR API."""

    # SEC EDGAR URLs
    EDGAR_BROWSE_URL = "https://www.sec.gov/cgi-bin/browse-edgar"
    EDGAR_ARCHIVES_URL = "https://www.sec.gov/Archives/edgar/data"
    SEC_TICKERS_URL = "https://www.sec.gov/files/company_tickers.json"

    USER_AGENT = "InvestorCenter.ai research@investorcenter.ai"

    # Rate limiting: 10 requests/second per SEC guidelines
    REQUESTS_PER_SECOND = 10
    MIN_REQUEST_INTERVAL = 1.0 / REQUESTS_PER_SECOND

    # Top institutional investors by AUM (CIK numbers)
    MAJOR_INSTITUTIONS = [
        ('0001067983', 'Berkshire Hathaway'),
        ('0001166559', 'Vanguard Group'),
        ('0001364742', 'BlackRock'),
        ('0000315066', 'State Street Corp'),
        ('0000102909', 'Morgan Stanley'),
        ('0000070858', 'Bank of America'),
        ('0000093751', 'Citigroup'),
        ('0001364180', 'Goldman Sachs'),
        ('0000019617', 'JPMorgan Chase'),
        ('0000718937', 'T. Rowe Price'),
        ('0000039362', 'Fidelity'),
        ('0001000275', 'Wellington Management'),
        ('0001861449', 'Geode Capital Management'),
        ('0001109357', 'Northern Trust'),
        ('0000896159', 'Capital Group'),
    ]

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

        # CUSIP to ticker mapping cache
        self.cusip_to_ticker = {}

    def _rate_limit(self):
        """Implement rate limiting for SEC requests."""
        current_time = time.time()
        elapsed = current_time - self.last_request_time

        if elapsed < self.MIN_REQUEST_INTERVAL:
            sleep_time = self.MIN_REQUEST_INTERVAL - elapsed
            time.sleep(sleep_time)

        self.last_request_time = time.time()

    async def load_cusip_ticker_mapping(self):
        """Load CUSIP to ticker mapping from SEC and database."""
        # First, get from database (stocks we already have)
        async with self.db.session() as session:
            query = text("""
                SELECT cusip, ticker
                FROM stocks
                WHERE cusip IS NOT NULL AND cusip != ''
            """)
            result = await session.execute(query)
            rows = result.fetchall()

            for row in rows:
                cusip = str(row[0]).strip()
                if cusip:
                    self.cusip_to_ticker[cusip] = row[1]

        logger.info(f"Loaded {len(self.cusip_to_ticker)} CUSIP mappings from database")

        # Also fetch from SEC company tickers JSON (has CUSIP in some cases)
        try:
            self._rate_limit()
            response = self.session.get(self.SEC_TICKERS_URL, timeout=30)
            response.raise_for_status()

            data = response.json()
            # SEC tickers JSON doesn't have CUSIP, but we'll keep this for future
            logger.info(f"SEC tickers JSON loaded ({len(data)} companies)")

        except Exception as e:
            logger.warning(f"Could not load SEC tickers JSON: {e}")

    def get_latest_13f_filing(self, institution_cik: str) -> Optional[Dict[str, str]]:
        """Get the latest 13F filing for an institution.

        Args:
            institution_cik: Institution CIK (10 digits).

        Returns:
            Dictionary with filing metadata, or None.
        """
        self._rate_limit()

        try:
            url = f"{self.EDGAR_BROWSE_URL}?action=getcompany&CIK={institution_cik}&type=13F-HR&count=1&output=atom"
            response = self.session.get(url, timeout=30)
            response.raise_for_status()

            # Parse XML response
            root = ET.fromstring(response.content)

            # Find first entry (latest filing)
            # Use full namespace URI in tag name for compatibility
            entry = root.find('{http://www.w3.org/2005/Atom}entry')

            if entry is None:
                logger.warning(f"No 13F filings found for CIK {institution_cik}")
                return None

            # Extract metadata
            content = entry.find('{http://www.w3.org/2005/Atom}content')
            if content is None:
                return None

            # Child elements also use Atom namespace
            accession = content.find('{http://www.w3.org/2005/Atom}accession-number')
            filing_date = content.find('{http://www.w3.org/2005/Atom}filing-date')
            filing_href = content.find('{http://www.w3.org/2005/Atom}filing-href')

            if accession is None or filing_date is None:
                return None

            return {
                'accession_number': accession.text,
                'filing_date': filing_date.text,
                'filing_url': filing_href.text if filing_href is not None else None,
                'cik': institution_cik
            }

        except Exception as e:
            logger.error(f"Error fetching 13F filing for CIK {institution_cik}: {e}")
            return None

    def parse_filing_date_to_quarter(self, filing_date_str: str) -> date:
        """Parse filing date string to quarter end date.

        Args:
            filing_date_str: Filing date in YYYY-MM-DD format.

        Returns:
            Estimated quarter end date.
        """
        filing_date = datetime.strptime(filing_date_str, '%Y-%m-%d').date()

        # 13F filings are due 45 days after quarter end
        # Estimate quarter based on filing month
        month = filing_date.month
        year = filing_date.year

        if month <= 3:
            # Filed in Q1, likely for Q4 of previous year
            return date(year - 1, 12, 31)
        elif month <= 6:
            # Filed in Q2, likely for Q1
            return date(year, 3, 31)
        elif month <= 9:
            # Filed in Q3, likely for Q2
            return date(year, 6, 30)
        else:
            # Filed in Q4, likely for Q3
            return date(year, 9, 30)

    def get_information_table_url(self, accession_number: str, cik: str) -> Optional[str]:
        """Find the information table XML file URL from a 13F filing.

        Args:
            accession_number: Filing accession number.
            cik: Institution CIK.

        Returns:
            URL to information table XML, or None.
        """
        self._rate_limit()

        try:
            # Clean accession number for URL
            accession_clean = accession_number.replace('-', '')

            # Get index page
            index_url = f"{self.EDGAR_ARCHIVES_URL}/{cik}/{accession_clean}/{accession_number}-index.htm"
            response = self.session.get(index_url, timeout=30)
            response.raise_for_status()

            # Look for .xml files (not primary_doc.xml)
            html = response.text

            # Pattern: href="/Archives/edgar/data/CIK/ACCESSION/XXXXX.xml"
            # Find all XML files
            import re
            xml_files = re.findall(r'href="(/Archives/edgar/data/[^"]+\.xml)"', html)

            # Filter out primary_doc.xml and XSLT-transformed versions, find the RAW information table XML
            for xml_file in xml_files:
                if ('primary_doc.xml' not in xml_file.lower() and
                    'xslForm13F_X' not in xml_file):  # Skip XSLT-transformed HTML versions
                    # This is the raw information table XML
                    return f"https://www.sec.gov{xml_file}"

            logger.warning(f"Could not find information table XML in {accession_number}")
            return None

        except Exception as e:
            logger.error(f"Error getting information table URL: {e}")
            return None

    def parse_information_table(
        self,
        xml_url: str,
        institution_cik: str,
        institution_name: str,
        quarter_end_date: date,
        filing_date: date
    ) -> List[Dict[str, Any]]:
        """Parse 13F information table XML.

        Args:
            xml_url: URL to information table XML.
            institution_cik: Institution CIK.
            institution_name: Institution name.
            quarter_end_date: Quarter end date.
            filing_date: Filing date.

        Returns:
            List of holding dictionaries.
        """
        self._rate_limit()

        holdings = []

        try:
            response = self.session.get(xml_url, timeout=30)
            response.raise_for_status()

            # Parse XML
            root = ET.fromstring(response.content)

            # Namespace
            ns = {'': 'http://www.sec.gov/edgar/document/thirteenf/informationtable'}

            # Find all infoTable elements
            for info_table in root.findall('.//infoTable', ns):
                try:
                    # Extract data
                    cusip_elem = info_table.find('cusip', ns)
                    value_elem = info_table.find('value', ns)
                    shares_elem = info_table.find('shrsOrPrnAmt/sshPrnamt', ns)

                    if cusip_elem is None or value_elem is None or shares_elem is None:
                        continue

                    cusip = cusip_elem.text.strip()

                    # Map CUSIP to ticker
                    ticker = self.cusip_to_ticker.get(cusip)
                    if not ticker:
                        # Try without check digit (last character)
                        ticker = self.cusip_to_ticker.get(cusip[:-1]) if len(cusip) > 8 else None

                    if not ticker:
                        # Use CUSIP as ticker temporarily (can backfill later)
                        # CUSIP is 9 chars, ticker field is VARCHAR(10), so it fits
                        ticker = cusip
                        logger.debug(f"No ticker match for CUSIP {cusip}, storing as CUSIP")

                    # Parse values
                    market_value = int(value_elem.text)  # Already in dollars
                    shares = int(shares_elem.text)

                    # Create holding record
                    holding = {
                        'ticker': ticker,
                        'filing_date': filing_date,
                        'quarter_end_date': quarter_end_date,
                        'institution_name': institution_name,
                        'institution_cik': institution_cik,
                        'shares': shares,
                        'market_value': market_value,
                        'position_change': None,  # Will calculate later
                        'shares_change': None,
                        'percent_change': None,
                    }

                    holdings.append(holding)

                except Exception as e:
                    # Skip individual entries that fail to parse
                    continue

        except Exception as e:
            logger.error(f"Error parsing information table from {xml_url}: {e}")

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
        async with self.db.session() as session:
            for holding in holdings:
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

                    shares_change = holding['shares'] - prev_shares
                    holding['shares_change'] = shares_change

                    if prev_shares > 0:
                        holding['percent_change'] = (shares_change / prev_shares) * 100
                    else:
                        holding['percent_change'] = None

                    # Determine position change
                    if shares_change > 0:
                        holding['position_change'] = 'Increased' if prev_shares > 0 else 'New'
                    elif shares_change < 0:
                        holding['position_change'] = 'Sold Out' if holding['shares'] == 0 else 'Decreased'
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

        # De-duplicate holdings by aggregating shares/value for same (ticker, quarter, institution)
        # This happens when institutions have multiple sub-managers holding the same security
        aggregated = {}
        for holding in holdings:
            key = (holding['ticker'], holding['quarter_end_date'], holding['institution_cik'])

            if key in aggregated:
                # Aggregate shares and market value
                aggregated[key]['shares'] += holding['shares']
                aggregated[key]['market_value'] += holding['market_value']
            else:
                # First occurrence - keep all fields
                aggregated[key] = holding.copy()

        deduplicated_holdings = list(aggregated.values())

        if len(holdings) != len(deduplicated_holdings):
            logger.info(
                f"De-duplicated {len(holdings)} holdings to {len(deduplicated_holdings)} "
                f"(aggregated {len(holdings) - len(deduplicated_holdings)} duplicate CUSIPs)"
            )

        # Batch insert to avoid PostgreSQL parameter limit (32767 args)
        # With ~10 fields per record, 1000 records = ~10,000 args (safe)
        batch_size = 1000
        total_stored = 0

        try:
            async with self.db.session() as session:
                for i in range(0, len(deduplicated_holdings), batch_size):
                    batch = deduplicated_holdings[i:i + batch_size]

                    # Insert with ON CONFLICT DO UPDATE
                    stmt = pg_insert(InstitutionalHolding).values(batch)
                    stmt = stmt.on_conflict_do_update(
                        index_elements=['ticker', 'quarter_end_date', 'institution_cik'],
                        set_={
                            'filing_date': stmt.excluded.filing_date,
                            'shares': stmt.excluded.shares,
                            'market_value': stmt.excluded.market_value,
                            'position_change': stmt.excluded.position_change,
                            'shares_change': stmt.excluded.shares_change,
                            'percent_change': stmt.excluded.percent_change,
                        }
                    )

                    await session.execute(stmt)
                    total_stored += len(batch)

                    if len(deduplicated_holdings) > batch_size:
                        logger.info(f"Stored batch {i//batch_size + 1}: {len(batch)} holdings")

                await session.commit()

                logger.info(f"Stored {total_stored} institutional holdings total")
                return True

        except Exception as e:
            logger.error(f"Error storing holdings: {e}", exc_info=True)
            return False

    async def process_institution(
        self,
        institution_cik: str,
        institution_name: str
    ) -> Tuple[int, int]:
        """Process 13F filings for one institution.

        Args:
            institution_cik: Institution CIK.
            institution_name: Institution name.

        Returns:
            Tuple of (holdings_count, error_count).
        """
        logger.info(f"Processing {institution_name} (CIK: {institution_cik})")

        # Get latest 13F filing
        filing = self.get_latest_13f_filing(institution_cik)

        if not filing:
            logger.warning(f"No 13F filing found for {institution_name}")
            return 0, 1

        logger.info(f"Found filing: {filing['accession_number']} dated {filing['filing_date']}")

        # Parse filing date to quarter
        quarter_end_date = self.parse_filing_date_to_quarter(filing['filing_date'])
        filing_date = datetime.strptime(filing['filing_date'], '%Y-%m-%d').date()

        # Get information table URL
        info_table_url = self.get_information_table_url(
            filing['accession_number'],
            institution_cik
        )

        if not info_table_url:
            logger.error(f"Could not find information table for {institution_name}")
            return 0, 1

        # Parse holdings
        holdings = self.parse_information_table(
            info_table_url,
            institution_cik,
            institution_name,
            quarter_end_date,
            filing_date
        )

        if not holdings:
            logger.warning(f"No holdings parsed for {institution_name}")
            return 0, 1

        logger.info(f"Parsed {len(holdings)} holdings for {institution_name}")

        # Calculate changes
        await self.calculate_quarter_changes(holdings, quarter_end_date)

        # Store holdings
        success = await self.store_holdings(holdings)

        return (len(holdings), 0) if success else (0, 1)

    async def run(
        self,
        cik: Optional[str] = None,
        limit: int = 15,
        all_institutions: bool = False
    ):
        """Run the ingestion pipeline.

        Args:
            cik: Specific institution CIK to process.
            limit: Number of major institutions to process.
            all_institutions: Process all major institutions.
        """
        start_time = datetime.now()
        logger.info("=" * 80)
        logger.info("SEC Form 13F Institutional Holdings Ingestion Pipeline")
        logger.info("=" * 80)

        # Load CUSIP to ticker mapping
        await self.load_cusip_ticker_mapping()

        # Determine institutions to process
        institutions = []

        if cik:
            # Single institution
            institution_name = next(
                (name for c, name in self.MAJOR_INSTITUTIONS if c == cik),
                'Unknown'
            )
            institutions = [(cik, institution_name)]
        elif all_institutions:
            institutions = self.MAJOR_INSTITUTIONS
        else:
            institutions = self.MAJOR_INSTITUTIONS[:limit]

        logger.info(f"Processing {len(institutions)} institutions")

        # Process each institution
        total_holdings = 0
        for institution_cik, institution_name in tqdm(institutions, desc="Processing institutions"):
            holdings_count, error_count = await self.process_institution(
                institution_cik,
                institution_name
            )

            self.processed_count += 1
            total_holdings += holdings_count

            if error_count > 0:
                self.error_count += 1
            else:
                self.success_count += 1

        # Print summary
        duration = datetime.now() - start_time
        logger.info("=" * 80)
        logger.info("Ingestion Complete")
        logger.info("=" * 80)
        logger.info(f"Duration: {duration}")
        logger.info(f"Institutions Processed: {self.processed_count}")
        logger.info(f"Success: {self.success_count}")
        logger.info(f"Errors: {self.error_count}")
        logger.info(f"Total Holdings Stored: {total_holdings}")


def main():
    """Main entry point for CLI."""
    parser = argparse.ArgumentParser(
        description='Ingest SEC Form 13F institutional holdings',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Process top 15 institutions (default)
  python sec_13f_ingestion.py --latest

  # Process top 50 institutions
  python sec_13f_ingestion.py --limit 50

  # Process all major institutions
  python sec_13f_ingestion.py --all

  # Process single institution (Berkshire Hathaway)
  python sec_13f_ingestion.py --cik 0001067983
        """
    )

    parser.add_argument(
        '--cik',
        type=str,
        help='Specific institution CIK to process'
    )
    parser.add_argument(
        '--limit',
        type=int,
        default=15,
        help='Number of major institutions to process (default: 15)'
    )
    parser.add_argument(
        '--all',
        action='store_true',
        help='Process all major institutions'
    )
    parser.add_argument(
        '--latest',
        action='store_true',
        help='Process latest filings (same as default)'
    )

    args = parser.parse_args()

    # Run pipeline
    pipeline = SEC13FIngestion()
    asyncio.run(pipeline.run(
        cik=args.cik,
        limit=args.limit,
        all_institutions=args.all
    ))


if __name__ == '__main__':
    main()
