#!/usr/bin/env python3
"""
Fetch SIC Codes from Polygon API for stocks missing sector data.

This script fetches ticker details from Polygon.io API and updates the
sic_code column in the tickers table. After running, the backfill_screener_data.py
script can map SIC codes to GICS sectors.

Usage:
    python fetch_sic_codes.py                    # Fetch all missing SIC codes
    python fetch_sic_codes.py --limit 500        # Fetch for 500 tickers only
    python fetch_sic_codes.py --ticker AAPL      # Fetch for single ticker
"""

import argparse
import logging
import os
import sys
import time
from datetime import datetime
from typing import Dict, List, Optional

import psycopg2
import requests

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class SICCodeFetcher:
    """Fetch and store SIC codes from Polygon API."""

    def __init__(self):
        self.db_config = {
            'host': os.environ.get('DB_HOST', 'localhost'),
            'port': int(os.environ.get('DB_PORT', 5432)),
            'database': os.environ.get('DB_NAME', 'investorcenter_db'),
            'user': os.environ.get('DB_USER'),
            'password': os.environ.get('DB_PASSWORD')
        }

        self.api_key = os.environ.get('POLYGON_API_KEY')
        if not self.api_key:
            raise ValueError("POLYGON_API_KEY environment variable required")

        self.conn = None
        self.stats = {
            'tickers_processed': 0,
            'sic_codes_updated': 0,
            'api_calls': 0,
            'errors': 0,
            'no_sic_code': 0
        }

        # Rate limiting - 5 requests per second for free tier
        self.request_interval = 0.2
        self.last_request_time = 0

    def connect_db(self) -> bool:
        """Connect to PostgreSQL database."""
        try:
            self.conn = psycopg2.connect(**self.db_config)
            logger.info(f"Connected to database: {self.db_config['database']}")
            return True
        except Exception as e:
            logger.error(f"Database connection failed: {e}")
            return False

    def rate_limit(self):
        """Ensure we don't exceed API rate limits."""
        current_time = time.time()
        time_since_last = current_time - self.last_request_time

        if time_since_last < self.request_interval:
            time.sleep(self.request_interval - time_since_last)

        self.last_request_time = time.time()

    def fetch_ticker_details(self, ticker: str) -> Optional[Dict]:
        """Fetch ticker details from Polygon API.

        Args:
            ticker: Stock ticker symbol

        Returns:
            Ticker details dict or None
        """
        url = f"https://api.polygon.io/v3/reference/tickers/{ticker}"
        params = {
            'apiKey': self.api_key
        }

        try:
            self.rate_limit()
            self.stats['api_calls'] += 1

            response = requests.get(url, params=params, timeout=10)
            response.raise_for_status()

            data = response.json()
            if data.get('status') == 'OK' and 'results' in data:
                return data['results']
            return None

        except requests.exceptions.RequestException as e:
            logger.debug(f"Failed to fetch details for {ticker}: {e}")
            self.stats['errors'] += 1
            return None

    def get_tickers_to_process(self, limit: Optional[int] = None, ticker: Optional[str] = None) -> List[str]:
        """Get list of tickers missing SIC codes.

        Args:
            limit: Maximum number of tickers
            ticker: Single ticker to process

        Returns:
            List of ticker symbols
        """
        if ticker:
            return [ticker]

        try:
            with self.conn.cursor() as cursor:
                # Get active stocks missing SIC codes
                query = """
                    SELECT symbol FROM (
                        SELECT DISTINCT symbol, market_cap
                        FROM tickers
                        WHERE asset_type = 'stock'
                          AND active = true
                          AND (sic_code IS NULL OR sic_code = '')
                    ) sub
                    ORDER BY market_cap DESC NULLS LAST
                """
                if limit:
                    query += f" LIMIT {limit}"

                cursor.execute(query)
                return [row[0] for row in cursor.fetchall()]

        except Exception as e:
            logger.error(f"Failed to get tickers: {e}")
            return []

    def update_sic_code(self, ticker: str, details: Dict) -> bool:
        """Update SIC code for a ticker.

        Args:
            ticker: Stock ticker symbol
            details: Ticker details from API

        Returns:
            True if updated successfully
        """
        sic_code = details.get('sic_code')
        if not sic_code:
            self.stats['no_sic_code'] += 1
            return False

        try:
            with self.conn.cursor() as cursor:
                cursor.execute("""
                    UPDATE tickers
                    SET sic_code = %s,
                        updated_at = CURRENT_TIMESTAMP
                    WHERE symbol = %s
                """, (sic_code, ticker))

                self.conn.commit()
                self.stats['sic_codes_updated'] += 1
                return True

        except Exception as e:
            logger.error(f"Failed to update SIC code for {ticker}: {e}")
            self.conn.rollback()
            return False

    def run(self, limit: Optional[int] = None, ticker: Optional[str] = None):
        """Run the SIC code fetch process.

        Args:
            limit: Maximum number of tickers to process
            ticker: Single ticker to process
        """
        start_time = datetime.now()
        logger.info("=" * 60)
        logger.info("SIC Code Fetcher")
        logger.info("=" * 60)

        if not self.connect_db():
            sys.exit(1)

        try:
            tickers = self.get_tickers_to_process(limit=limit, ticker=ticker)
            logger.info(f"Processing {len(tickers)} tickers")

            for i, symbol in enumerate(tickers):
                if (i + 1) % 100 == 0:
                    logger.info(f"Progress: {i + 1}/{len(tickers)} - Updated: {self.stats['sic_codes_updated']}")

                details = self.fetch_ticker_details(symbol)
                self.stats['tickers_processed'] += 1

                if details:
                    self.update_sic_code(symbol, details)

            # Print summary
            duration = datetime.now() - start_time
            logger.info("=" * 60)
            logger.info("Fetch Complete")
            logger.info("=" * 60)
            logger.info(f"Duration: {duration}")
            logger.info(f"Tickers processed: {self.stats['tickers_processed']}")
            logger.info(f"API calls: {self.stats['api_calls']}")
            logger.info(f"SIC codes updated: {self.stats['sic_codes_updated']}")
            logger.info(f"No SIC code available: {self.stats['no_sic_code']}")
            logger.info(f"Errors: {self.stats['errors']}")

        finally:
            if self.conn:
                self.conn.close()


def main():
    parser = argparse.ArgumentParser(description='Fetch SIC codes from Polygon API')
    parser.add_argument('--limit', type=int, help='Limit number of tickers')
    parser.add_argument('--ticker', type=str, help='Process single ticker')

    args = parser.parse_args()

    fetcher = SICCodeFetcher()
    fetcher.run(limit=args.limit, ticker=args.ticker)


if __name__ == '__main__':
    main()
