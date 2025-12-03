#!/usr/bin/env python3
"""
Fetch Dividend Data from Polygon API.

This script fetches dividend history for all stocks and stores in PostgreSQL.
Designed to run as a Kubernetes Job.

Usage:
    python fetch_dividends.py                    # Fetch all missing dividends
    python fetch_dividends.py --limit 100        # Fetch for 100 tickers only
    python fetch_dividends.py --ticker AAPL      # Fetch for single ticker
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
from psycopg2.extras import RealDictCursor, execute_batch

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class DividendFetcher:
    """Fetch and store dividend data from Polygon API."""

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
            'dividends_fetched': 0,
            'dividends_inserted': 0,
            'api_calls': 0,
            'errors': 0
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

    def fetch_dividends(self, ticker: str) -> List[Dict]:
        """Fetch dividend history from Polygon API.

        Args:
            ticker: Stock ticker symbol

        Returns:
            List of dividend records
        """
        url = "https://api.polygon.io/v3/reference/dividends"
        params = {
            'ticker': ticker,
            'limit': 100,
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
            return []

        except requests.exceptions.RequestException as e:
            logger.debug(f"Failed to fetch dividends for {ticker}: {e}")
            self.stats['errors'] += 1
            return []

    def get_tickers_to_process(self, limit: Optional[int] = None, ticker: Optional[str] = None) -> List[str]:
        """Get list of tickers to fetch dividends for.

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
                # Get active stocks not yet in dividends table
                query = """
                    SELECT DISTINCT t.symbol
                    FROM tickers t
                    LEFT JOIN (
                        SELECT DISTINCT symbol FROM dividends
                    ) d ON t.symbol = d.symbol
                    WHERE t.asset_type IN ('stock', 'CS')
                      AND t.active = true
                      AND d.symbol IS NULL
                    ORDER BY t.market_cap DESC NULLS LAST
                """
                if limit:
                    query += f" LIMIT {limit}"

                cursor.execute(query)
                return [row[0] for row in cursor.fetchall()]

        except Exception as e:
            logger.error(f"Failed to get tickers: {e}")
            return []

    def insert_dividends(self, ticker: str, dividends: List[Dict]) -> int:
        """Insert dividend records into database.

        Args:
            ticker: Stock ticker symbol
            dividends: List of dividend records from API

        Returns:
            Number of records inserted
        """
        if not dividends:
            return 0

        try:
            with self.conn.cursor() as cursor:
                records = []
                for div in dividends:
                    records.append((
                        ticker,
                        div.get('ex_dividend_date'),
                        div.get('pay_date'),
                        div.get('record_date'),
                        div.get('declaration_date'),
                        div.get('cash_amount', 0),
                        div.get('currency', 'USD'),
                        div.get('frequency'),
                        div.get('dividend_type', 'CD')
                    ))

                execute_batch(cursor, """
                    INSERT INTO dividends (
                        symbol, ex_date, pay_date, record_date, declaration_date,
                        amount, currency, frequency, type
                    ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                    ON CONFLICT (symbol, ex_date, type) DO UPDATE SET
                        pay_date = EXCLUDED.pay_date,
                        amount = EXCLUDED.amount,
                        updated_at = CURRENT_TIMESTAMP
                """, records, page_size=100)

                self.conn.commit()
                return len(records)

        except Exception as e:
            logger.error(f"Failed to insert dividends for {ticker}: {e}")
            self.conn.rollback()
            return 0

    def run(self, limit: Optional[int] = None, ticker: Optional[str] = None):
        """Run the dividend fetch process.

        Args:
            limit: Maximum number of tickers to process
            ticker: Single ticker to process
        """
        start_time = datetime.now()
        logger.info("=" * 60)
        logger.info("Dividend Data Fetcher")
        logger.info("=" * 60)

        if not self.connect_db():
            sys.exit(1)

        try:
            tickers = self.get_tickers_to_process(limit=limit, ticker=ticker)
            logger.info(f"Processing {len(tickers)} tickers")

            for i, symbol in enumerate(tickers):
                if (i + 1) % 100 == 0:
                    logger.info(f"Progress: {i + 1}/{len(tickers)} - Inserted: {self.stats['dividends_inserted']}")

                dividends = self.fetch_dividends(symbol)
                self.stats['tickers_processed'] += 1
                self.stats['dividends_fetched'] += len(dividends)

                if dividends:
                    inserted = self.insert_dividends(symbol, dividends)
                    self.stats['dividends_inserted'] += inserted

            # Print summary
            duration = datetime.now() - start_time
            logger.info("=" * 60)
            logger.info("Fetch Complete")
            logger.info("=" * 60)
            logger.info(f"Duration: {duration}")
            logger.info(f"Tickers processed: {self.stats['tickers_processed']}")
            logger.info(f"API calls: {self.stats['api_calls']}")
            logger.info(f"Dividends fetched: {self.stats['dividends_fetched']}")
            logger.info(f"Dividends inserted: {self.stats['dividends_inserted']}")
            logger.info(f"Errors: {self.stats['errors']}")

        finally:
            if self.conn:
                self.conn.close()


def main():
    parser = argparse.ArgumentParser(description='Fetch dividend data from Polygon API')
    parser.add_argument('--limit', type=int, help='Limit number of tickers')
    parser.add_argument('--ticker', type=str, help='Process single ticker')

    args = parser.parse_args()

    fetcher = DividendFetcher()
    fetcher.run(limit=args.limit, ticker=args.ticker)


if __name__ == '__main__':
    main()
