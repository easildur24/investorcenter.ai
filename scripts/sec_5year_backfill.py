#!/usr/bin/env python3
"""
SEC 5-Year Historical Data Backfill

Downloads 5 years of SEC filings (10-K and 10-Q) for all companies
with CIK numbers in the database.

This script processes each company and fetches all 10-K and 10-Q filings
from the past 5 years.
"""

import os
import sys
import time
import logging
import psycopg2
import requests
from datetime import datetime, timedelta
from typing import Dict, List, Optional
from psycopg2.extras import RealDictCursor

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class SEC5YearBackfill:
    """Fetches 5 years of SEC filing data"""

    def __init__(self, years: int = 5):
        self.years = years
        self.cutoff_date = datetime.now() - timedelta(days=years * 365)

        self.db_config = {
            'host': os.environ.get('DB_HOST', 'localhost'),
            'port': int(os.environ.get('DB_PORT', 5432)),
            'database': os.environ.get('DB_NAME', 'investorcenter_db'),
            'user': os.environ.get('DB_USER'),
            'password': os.environ.get('DB_PASSWORD')
        }

        self.base_url = "https://data.sec.gov"
        self.headers = {
            'User-Agent': 'InvestorCenter AI (contact@investorcenter.ai)',
            'Accept-Encoding': 'gzip, deflate'
        }

        self.conn = None
        self.stats = {
            'companies_processed': 0,
            'companies_with_cik': 0,
            'filings_found': 0,
            'filings_saved': 0,
            'filings_skipped': 0,
            'errors': 0
        }

        # Rate limiting: SEC allows 10 requests per second
        self.last_request_time = 0
        self.min_request_interval = 0.1  # 100ms

    def connect_db(self) -> bool:
        """Connect to PostgreSQL database"""
        try:
            self.conn = psycopg2.connect(**self.db_config)
            logger.info(f"Connected to database: {self.db_config['database']}")
            return True
        except Exception as e:
            logger.error(f"Database connection failed: {e}")
            return False

    def get_companies_with_cik(self, limit: Optional[int] = None) -> List[Dict]:
        """Get all companies that have CIK numbers"""
        try:
            with self.conn.cursor(cursor_factory=RealDictCursor) as cursor:
                query = """
                    SELECT
                        id as ticker_id,
                        symbol,
                        name,
                        cik,
                        asset_type
                    FROM tickers
                    WHERE
                        cik IS NOT NULL
                        AND cik != ''
                        AND asset_type IN ('stock', 'CS', 'ADRC', 'PFD')
                        AND active = true
                    ORDER BY market_cap DESC NULLS LAST
                """

                if limit:
                    query += f" LIMIT {limit}"

                cursor.execute(query)
                companies = cursor.fetchall()

                logger.info(f"Found {len(companies)} companies with CIK numbers")
                self.stats['companies_with_cik'] = len(companies)
                return companies

        except Exception as e:
            logger.error(f"Failed to get companies: {e}")
            return []

    def rate_limit(self):
        """Ensure we don't exceed SEC's rate limit"""
        current_time = time.time()
        time_since_last = current_time - self.last_request_time

        if time_since_last < self.min_request_interval:
            sleep_time = self.min_request_interval - time_since_last
            time.sleep(sleep_time)

        self.last_request_time = time.time()

    def fetch_company_submissions(self, cik: str) -> Optional[Dict]:
        """Fetch all filings for a company"""
        cik_padded = cik.zfill(10)
        url = f"{self.base_url}/submissions/CIK{cik_padded}.json"

        try:
            self.rate_limit()
            response = requests.get(url, headers=self.headers, timeout=30)
            response.raise_for_status()

            data = response.json()
            return data

        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to fetch submissions for CIK {cik}: {e}")
            return None

    def is_filing_within_range(self, filing_date_str: str) -> bool:
        """Check if filing is within the past N years"""
        try:
            filing_date = datetime.strptime(filing_date_str, '%Y-%m-%d')
            return filing_date >= self.cutoff_date
        except:
            return False

    def save_filing_metadata(self, ticker_id: int, symbol: str, cik: str, filing: Dict) -> bool:
        """Save filing metadata to database"""
        try:
            with self.conn.cursor() as cursor:
                # Check if filing already exists
                cursor.execute("""
                    SELECT id FROM sec_filings
                    WHERE accession_number = %s
                """, (filing['accession_number'],))

                if cursor.fetchone():
                    self.stats['filings_skipped'] += 1
                    return False  # Already exists

                # Insert new filing
                cursor.execute("""
                    INSERT INTO sec_filings (
                        ticker_id, symbol, cik, filing_type,
                        filing_date, report_date, accession_number,
                        primary_document, primary_doc_description,
                        size_bytes, created_at
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, CURRENT_TIMESTAMP
                    )
                    RETURNING id
                """, (
                    ticker_id,
                    symbol,
                    cik,
                    filing['form'],
                    filing['filing_date'],
                    filing.get('report_date'),
                    filing['accession_number'],
                    filing.get('primary_document'),
                    filing.get('primary_doc_description'),
                    filing.get('size', 0)
                ))

                filing_id = cursor.fetchone()[0]
                self.conn.commit()

                logger.debug(f"  Saved {filing['form']} from {filing['filing_date']}")
                self.stats['filings_saved'] += 1
                return True

        except Exception as e:
            logger.error(f"Failed to save filing: {e}")
            self.conn.rollback()
            self.stats['errors'] += 1
            return False

    def process_company(self, company: Dict):
        """Process all 10-K and 10-Q filings for a company within date range"""
        symbol = company['symbol']
        cik = company['cik']
        ticker_id = company['ticker_id']

        logger.info(f"Processing {symbol} (CIK: {cik}) - Looking for filings since {self.cutoff_date.strftime('%Y-%m-%d')}")

        # Fetch submissions
        submissions = self.fetch_company_submissions(cik)
        if not submissions:
            self.stats['errors'] += 1
            return

        # Extract recent filings
        recent_filings = submissions.get('filings', {}).get('recent', {})
        if not recent_filings:
            logger.warning(f"  No filings found for {symbol}")
            return

        # Process each filing
        forms = recent_filings.get('form', [])
        filing_dates = recent_filings.get('filingDate', [])
        report_dates = recent_filings.get('reportDate', [])
        accession_numbers = recent_filings.get('accessionNumber', [])
        primary_documents = recent_filings.get('primaryDocument', [])
        primary_doc_descriptions = recent_filings.get('primaryDocDescription', [])
        sizes = recent_filings.get('size', [])

        filings_saved = 0
        filings_in_range = 0

        for i in range(len(forms)):
            # Only process 10-K and 10-Q (including amendments)
            if forms[i] not in ['10-K', '10-Q', '10-K/A', '10-Q/A']:
                continue

            # Check if filing is within date range
            if i < len(filing_dates) and not self.is_filing_within_range(filing_dates[i]):
                continue

            filings_in_range += 1
            self.stats['filings_found'] += 1

            filing = {
                'form': forms[i],
                'filing_date': filing_dates[i] if i < len(filing_dates) else None,
                'report_date': report_dates[i] if i < len(report_dates) else None,
                'accession_number': accession_numbers[i] if i < len(accession_numbers) else None,
                'primary_document': primary_documents[i] if i < len(primary_documents) else None,
                'primary_doc_description': primary_doc_descriptions[i] if i < len(primary_doc_descriptions) else None,
                'size': sizes[i] if i < len(sizes) else 0
            }

            if self.save_filing_metadata(ticker_id, symbol, cik, filing):
                filings_saved += 1

        logger.info(f"  ✓ {symbol}: Found {filings_in_range} filings in range, saved {filings_saved} new")
        self.stats['companies_processed'] += 1

        # Update sync status
        self.update_sync_status(symbol, cik)

    def update_sync_status(self, symbol: str, cik: str):
        """Update sync status for a company"""
        try:
            with self.conn.cursor() as cursor:
                cursor.execute("""
                    INSERT INTO sec_sync_status (
                        symbol, cik, last_filing_check,
                        last_successful_sync, total_filings_count
                    ) VALUES (
                        %s, %s, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP,
                        (SELECT COUNT(*) FROM sec_filings WHERE symbol = %s)
                    )
                    ON CONFLICT (symbol) DO UPDATE SET
                        cik = EXCLUDED.cik,
                        last_filing_check = CURRENT_TIMESTAMP,
                        last_successful_sync = CURRENT_TIMESTAMP,
                        total_filings_count = (SELECT COUNT(*) FROM sec_filings WHERE symbol = %s),
                        updated_at = CURRENT_TIMESTAMP
                """, (symbol, cik, symbol, symbol))

                self.conn.commit()

        except Exception as e:
            logger.error(f"Failed to update sync status for {symbol}: {e}")
            self.conn.rollback()

    def run(self, limit: Optional[int] = None):
        """Main execution function"""
        start_time = datetime.now()
        logger.info(f"=== SEC {self.years}-Year Historical Backfill Started ===")
        logger.info(f"Fetching 10-K and 10-Q filings since {self.cutoff_date.strftime('%Y-%m-%d')}")

        try:
            # Connect to database
            if not self.connect_db():
                logger.error("Failed to connect to database")
                sys.exit(1)

            # Get all companies with CIK
            companies = self.get_companies_with_cik(limit)

            if not companies:
                logger.info("No companies with CIK found")
                return

            logger.info(f"Processing {len(companies)} companies...")

            # Process each company
            for i, company in enumerate(companies, 1):
                try:
                    self.process_company(company)

                    # Progress logging every 50 companies
                    if i % 50 == 0:
                        logger.info(f"Progress: {i}/{len(companies)} companies processed")
                        logger.info(f"  Stats: Filings found={self.stats['filings_found']}, "
                                  f"Saved={self.stats['filings_saved']}, "
                                  f"Skipped={self.stats['filings_skipped']}")

                    # Brief pause every 100 companies to be respectful
                    if i % 100 == 0:
                        logger.info("Brief pause...")
                        time.sleep(5)

                except Exception as e:
                    logger.error(f"Error processing {company['symbol']}: {e}")
                    self.stats['errors'] += 1
                    continue

            # Log final statistics
            duration = datetime.now() - start_time
            logger.info(f"Processing completed in {duration.total_seconds() / 60:.1f} minutes")
            logger.info(f"=== Final Statistics ===")
            logger.info(f"  Companies with CIK: {self.stats['companies_with_cik']}")
            logger.info(f"  Companies processed: {self.stats['companies_processed']}")
            logger.info(f"  Filings found (in range): {self.stats['filings_found']}")
            logger.info(f"  Filings saved: {self.stats['filings_saved']}")
            logger.info(f"  Filings skipped (duplicates): {self.stats['filings_skipped']}")
            logger.info(f"  Errors: {self.stats['errors']}")

            if self.stats['filings_found'] > 0:
                success_rate = (self.stats['filings_saved'] / self.stats['filings_found']) * 100
                logger.info(f"  Success rate: {success_rate:.1f}%")

            logger.info(f"=== SEC {self.years}-Year Historical Backfill Completed ===")

        except Exception as e:
            logger.error(f"Backfill failed: {e}")
            sys.exit(1)
        finally:
            if self.conn:
                self.conn.close()


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description='SEC 5-Year Historical Data Backfill')
    parser.add_argument('--years', type=int, default=5,
                       help='Number of years to backfill (default: 5)')
    parser.add_argument('--limit', type=int, default=None,
                       help='Limit number of companies to process (for testing)')

    args = parser.parse_args()

    backfill = SEC5YearBackfill(years=args.years)
    backfill.run(limit=args.limit)
