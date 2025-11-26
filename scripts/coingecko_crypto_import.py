#!/usr/bin/env python3
"""
CoinGecko Cryptocurrency Import Script

This script fetches cryptocurrency data from CoinGecko API and imports it into the database.
It replaces the Polygon.io crypto imports with comprehensive crypto data from CoinGecko.

Features:
- Fetches top cryptocurrencies by market cap
- Implements rate limiting (30 calls/min for free tier)
- Batch processing with pagination
- Clean symbols (BTC instead of X:BTCUSD)
- Comprehensive metadata including ATH/ATL, supply metrics, etc.
- Upsert logic for database updates

Environment Variables:
    DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME - Database connection
    COINGECKO_API_KEY - Optional API key for higher rate limits
"""

import os
import sys
import json
import time
import logging
import requests
import psycopg2
from datetime import datetime, timezone
from decimal import Decimal
from typing import Dict, List, Optional, Tuple
from psycopg2.extras import RealDictCursor, execute_batch
from urllib.parse import urlencode

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class RateLimiter:
    """Simple rate limiter to respect API limits"""

    def __init__(self, max_calls: int, per_seconds: int):
        self.max_calls = max_calls
        self.per_seconds = per_seconds
        self.calls = []

    def wait_if_needed(self):
        """Wait if necessary to respect rate limits"""
        now = time.time()
        # Remove old calls outside the window
        self.calls = [call_time for call_time in self.calls
                     if now - call_time < self.per_seconds]

        if len(self.calls) >= self.max_calls:
            # Need to wait
            sleep_time = self.per_seconds - (now - self.calls[0]) + 0.1
            if sleep_time > 0:
                logger.info(f"Rate limit reached, sleeping for {sleep_time:.1f} seconds...")
                time.sleep(sleep_time)
                # Clear old calls after sleeping
                now = time.time()
                self.calls = [call_time for call_time in self.calls
                            if now - call_time < self.per_seconds]

        self.calls.append(now)


class CoinGeckoImporter:
    """Handles cryptocurrency imports from CoinGecko API"""

    def __init__(self):
        # API configuration
        self.api_key = os.environ.get('COINGECKO_API_KEY')  # Optional for free tier
        self.base_url = "https://api.coingecko.com/api/v3"

        # Rate limiter: 30 calls per minute for free tier
        self.rate_limiter = RateLimiter(30, 60)

        # Database configuration
        self.db_config = {
            'host': os.environ.get('DB_HOST', 'localhost'),
            'port': int(os.environ.get('DB_PORT', 5432)),
            'database': os.environ.get('DB_NAME', 'investorcenter_db'),
            'user': os.environ.get('DB_USER'),
            'password': os.environ.get('DB_PASSWORD')
        }

        self.conn = None
        self.session = requests.Session()

        # Statistics
        self.stats = {
            'total_fetched': 0,
            'inserted': 0,
            'updated': 0,
            'failed': 0,
            'skipped': 0
        }

    def connect_db(self) -> bool:
        """Connect to PostgreSQL database"""
        try:
            self.conn = psycopg2.connect(**self.db_config)
            logger.info(f"Connected to database: {self.db_config['database']}")
            return True
        except Exception as e:
            logger.error(f"Database connection failed: {e}")
            return False

    def make_api_request(self, endpoint: str, params: Dict = None) -> Optional[Dict]:
        """Make API request with rate limiting and error handling"""
        self.rate_limiter.wait_if_needed()

        url = f"{self.base_url}/{endpoint}"

        # Add API key if available (for higher rate limits)
        if self.api_key and params:
            params['x_cg_demo_api_key'] = self.api_key
        elif self.api_key:
            params = {'x_cg_demo_api_key': self.api_key}

        try:
            response = self.session.get(url, params=params, timeout=30)

            # Handle rate limiting
            if response.status_code == 429:
                retry_after = int(response.headers.get('Retry-After', 60))
                logger.warning(f"Rate limited. Waiting {retry_after} seconds...")
                time.sleep(retry_after)
                return self.make_api_request(endpoint, params)

            response.raise_for_status()
            return response.json()

        except requests.exceptions.RequestException as e:
            logger.error(f"API request failed for {endpoint}: {e}")
            return None

    def fetch_cryptocurrencies(self, limit: int = 3000) -> List[Dict]:
        """Fetch top N cryptocurrencies by market cap"""
        all_coins = []
        per_page = 250  # Maximum allowed by CoinGecko
        pages_needed = (limit + per_page - 1) // per_page

        logger.info(f"Fetching top {limit} cryptocurrencies from CoinGecko...")

        for page in range(1, pages_needed + 1):
            # Calculate how many to fetch on this page
            remaining = limit - len(all_coins)
            if remaining <= 0:
                break

            page_size = min(per_page, remaining)

            params = {
                'vs_currency': 'usd',
                'order': 'market_cap_desc',
                'per_page': page_size,
                'page': page,
                'sparkline': False,
                'price_change_percentage': '24h,7d,30d,1y'
            }

            logger.info(f"Fetching page {page} (coins {(page-1)*per_page + 1} to {(page-1)*per_page + page_size})...")

            coins = self.make_api_request('coins/markets', params)

            if not coins:
                logger.error(f"Failed to fetch page {page}")
                break

            all_coins.extend(coins)
            logger.info(f"  Retrieved {len(coins)} coins (total: {len(all_coins)})")

            # Small delay between pages to be respectful
            if page < pages_needed:
                time.sleep(1)

        self.stats['total_fetched'] = len(all_coins)
        logger.info(f"Successfully fetched {len(all_coins)} cryptocurrencies")
        return all_coins

    def transform_to_db_format(self, coin: Dict) -> Dict:
        """Transform CoinGecko data to match our database schema"""
        # Extract clean symbol (uppercase, no prefixes)
        symbol = coin.get('symbol', '').upper()

        # Handle None values for supply metrics
        def safe_decimal(value):
            if value is None or value == '':
                return None
            try:
                return Decimal(str(value))
            except:
                return None

        # Map CoinGecko fields to our schema
        return {
            'symbol': symbol,
            'name': coin.get('name', ''),
            'asset_type': 'crypto',
            'market': 'crypto',
            'locale': 'global',
            'active': True,  # CoinGecko only returns active coins
            'currency': 'USD',
            'currency_name': 'US Dollar',

            # Market data
            'market_cap': coin.get('market_cap'),
            'market_cap_rank': coin.get('market_cap_rank'),
            'last_price': coin.get('current_price'),
            'volume': coin.get('total_volume'),
            'high_24h': coin.get('high_24h'),
            'low_24h': coin.get('low_24h'),
            'price_change_24h': coin.get('price_change_24h'),
            'price_change_percentage_24h': coin.get('price_change_percentage_24h'),

            # Supply metrics
            'circulating_supply': safe_decimal(coin.get('circulating_supply')),
            'total_supply': safe_decimal(coin.get('total_supply')),
            'max_supply': safe_decimal(coin.get('max_supply')),
            'fully_diluted_valuation': coin.get('fully_diluted_valuation'),

            # All-time high/low (limit percentage to avoid overflow)
            'ath': coin.get('ath'),
            'ath_date': coin.get('ath_date'),
            'ath_change_percentage': min(coin.get('ath_change_percentage', 0), 999999.9999) if coin.get('ath_change_percentage') else None,
            'atl': coin.get('atl'),
            'atl_date': coin.get('atl_date'),
            'atl_change_percentage': min(coin.get('atl_change_percentage', 0), 999999.9999) if coin.get('atl_change_percentage') else None,

            # Additional metadata
            'coingecko_id': coin.get('id'),
            'logo_url': coin.get('image'),
            'last_updated': coin.get('last_updated'),

            # ROI data if available
            'roi_times': coin.get('roi', {}).get('times') if coin.get('roi') else None,
            'roi_currency': coin.get('roi', {}).get('currency') if coin.get('roi') else None,
            'roi_percentage': coin.get('roi', {}).get('percentage') if coin.get('roi') else None
        }

    def create_or_update_crypto_tables(self):
        """Ensure crypto-specific columns exist in the database"""
        try:
            with self.conn.cursor() as cursor:
                # Add crypto-specific columns if they don't exist
                cursor.execute("""
                    -- Add CoinGecko specific columns if they don't exist
                    DO $$
                    BEGIN
                        -- CoinGecko ID
                        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                                     WHERE table_name='tickers' AND column_name='coingecko_id') THEN
                            ALTER TABLE tickers ADD COLUMN coingecko_id VARCHAR(100);
                        END IF;

                        -- Market cap rank
                        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                                     WHERE table_name='tickers' AND column_name='market_cap_rank') THEN
                            ALTER TABLE tickers ADD COLUMN market_cap_rank INTEGER;
                        END IF;

                        -- Supply metrics (if not already present)
                        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                                     WHERE table_name='tickers' AND column_name='total_supply') THEN
                            ALTER TABLE tickers ADD COLUMN total_supply NUMERIC(30,8);
                        END IF;

                        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                                     WHERE table_name='tickers' AND column_name='fully_diluted_valuation') THEN
                            ALTER TABLE tickers ADD COLUMN fully_diluted_valuation BIGINT;
                        END IF;

                        -- All-time high/low
                        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                                     WHERE table_name='tickers' AND column_name='ath') THEN
                            ALTER TABLE tickers ADD COLUMN ath NUMERIC(20,8);
                            ALTER TABLE tickers ADD COLUMN ath_date TIMESTAMP;
                            ALTER TABLE tickers ADD COLUMN ath_change_percentage NUMERIC(10,4);
                        END IF;

                        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                                     WHERE table_name='tickers' AND column_name='atl') THEN
                            ALTER TABLE tickers ADD COLUMN atl NUMERIC(20,8);
                            ALTER TABLE tickers ADD COLUMN atl_date TIMESTAMP;
                            ALTER TABLE tickers ADD COLUMN atl_change_percentage NUMERIC(10,4);
                        END IF;

                        -- 24h price data
                        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                                     WHERE table_name='tickers' AND column_name='high_24h') THEN
                            ALTER TABLE tickers ADD COLUMN high_24h NUMERIC(20,8);
                            ALTER TABLE tickers ADD COLUMN low_24h NUMERIC(20,8);
                            ALTER TABLE tickers ADD COLUMN price_change_24h NUMERIC(20,8);
                            ALTER TABLE tickers ADD COLUMN price_change_percentage_24h NUMERIC(10,4);
                        END IF;

                        -- ROI data
                        IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                                     WHERE table_name='tickers' AND column_name='roi_times') THEN
                            ALTER TABLE tickers ADD COLUMN roi_times NUMERIC(20,8);
                            ALTER TABLE tickers ADD COLUMN roi_currency VARCHAR(10);
                            ALTER TABLE tickers ADD COLUMN roi_percentage NUMERIC(20,4);
                        END IF;
                    END $$;
                """)

                self.conn.commit()
                logger.info("Database schema updated for crypto data")

        except Exception as e:
            logger.error(f"Failed to update database schema: {e}")
            self.conn.rollback()

    def import_to_database(self, coins: List[Dict]) -> Tuple[int, int]:
        """Import cryptocurrencies to database with upsert logic"""
        if not coins:
            return 0, 0

        inserted = 0
        updated = 0

        try:
            with self.conn.cursor(cursor_factory=RealDictCursor) as cursor:
                for coin_data in coins:
                    coin = self.transform_to_db_format(coin_data)

                    # Skip if no symbol
                    if not coin['symbol']:
                        self.stats['skipped'] += 1
                        continue

                    # Check if this crypto ticker exists (using composite key: symbol + asset_type)
                    # With the new schema, we can have both META stock and META crypto
                    cursor.execute(
                        "SELECT id FROM tickers WHERE symbol = %s AND asset_type = 'crypto'",
                        (coin['symbol'],)
                    )
                    existing = cursor.fetchone()

                    if existing:
                        # Update existing crypto ticker
                        cursor.execute("""
                            UPDATE tickers SET
                                name = %s,
                                market = %s,
                                locale = %s,
                                active = %s,
                                currency = %s,
                                currency_name = %s,
                                market_cap = %s,
                                market_cap_rank = %s,
                                last_price = %s,
                                volume = %s,
                                high_24h = %s,
                                low_24h = %s,
                                price_change_24h = %s,
                                price_change_percentage_24h = %s,
                                circulating_supply = %s,
                                total_supply = %s,
                                max_supply = %s,
                                fully_diluted_valuation = %s,
                                ath = %s,
                                ath_date = %s,
                                ath_change_percentage = %s,
                                atl = %s,
                                atl_date = %s,
                                atl_change_percentage = %s,
                                coingecko_id = %s,
                                logo_url = %s,
                                roi_times = %s,
                                roi_currency = %s,
                                roi_percentage = %s,
                                last_updated_utc = %s,
                                updated_at = CURRENT_TIMESTAMP
                            WHERE symbol = %s AND asset_type = 'crypto'
                        """, (
                            coin['name'], coin['market'], coin['locale'],
                            coin['active'], coin['currency'], coin['currency_name'],
                            coin['market_cap'], coin['market_cap_rank'],
                            coin['last_price'], coin['volume'],
                            coin['high_24h'], coin['low_24h'],
                            coin['price_change_24h'], coin['price_change_percentage_24h'],
                            coin['circulating_supply'], coin['total_supply'], coin['max_supply'],
                            coin['fully_diluted_valuation'],
                            coin['ath'], coin['ath_date'], coin['ath_change_percentage'],
                            coin['atl'], coin['atl_date'], coin['atl_change_percentage'],
                            coin['coingecko_id'], coin['logo_url'],
                            coin['roi_times'], coin['roi_currency'], coin['roi_percentage'],
                            coin['last_updated'],
                            coin['symbol']
                        ))
                        updated += 1
                    else:
                        # Insert new ticker
                        cursor.execute("""
                            INSERT INTO tickers (
                                symbol, name, asset_type, market, locale, active,
                                currency, currency_name, market_cap, market_cap_rank,
                                last_price, volume, high_24h, low_24h,
                                price_change_24h, price_change_percentage_24h,
                                circulating_supply, total_supply, max_supply,
                                fully_diluted_valuation, ath, ath_date, ath_change_percentage,
                                atl, atl_date, atl_change_percentage,
                                coingecko_id, logo_url, roi_times, roi_currency, roi_percentage,
                                last_updated_utc, created_at, updated_at
                            ) VALUES (
                                %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                                %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                                %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                                %s, %s, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
                            )
                        """, (
                            coin['symbol'], coin['name'], coin['asset_type'],
                            coin['market'], coin['locale'], coin['active'],
                            coin['currency'], coin['currency_name'],
                            coin['market_cap'], coin['market_cap_rank'],
                            coin['last_price'], coin['volume'],
                            coin['high_24h'], coin['low_24h'],
                            coin['price_change_24h'], coin['price_change_percentage_24h'],
                            coin['circulating_supply'], coin['total_supply'], coin['max_supply'],
                            coin['fully_diluted_valuation'],
                            coin['ath'], coin['ath_date'], coin['ath_change_percentage'],
                            coin['atl'], coin['atl_date'], coin['atl_change_percentage'],
                            coin['coingecko_id'], coin['logo_url'],
                            coin['roi_times'], coin['roi_currency'], coin['roi_percentage'],
                            coin['last_updated']
                        ))
                        inserted += 1

                    # Log progress every 100 coins
                    if (inserted + updated) % 100 == 0:
                        logger.info(f"Progress: {inserted} inserted, {updated} updated")

                self.conn.commit()
                self.stats['inserted'] = inserted
                self.stats['updated'] = updated

                logger.info(f"Database import complete: {inserted} inserted, {updated} updated")
                return inserted, updated

        except Exception as e:
            logger.error(f"Database import failed: {e}")
            self.conn.rollback()
            self.stats['failed'] = len(coins)
            return 0, 0

    def run(self, limit: int = 3000):
        """Main execution function"""
        start_time = datetime.now()
        logger.info("=== CoinGecko Crypto Import Started ===")
        logger.info(f"Target: Import top {limit} cryptocurrencies")

        try:
            # Connect to database
            if not self.connect_db():
                logger.error("Failed to connect to database")
                sys.exit(1)

            # Update database schema for crypto data
            self.create_or_update_crypto_tables()

            # Fetch cryptocurrencies from CoinGecko
            coins = self.fetch_cryptocurrencies(limit)

            if not coins:
                logger.warning("No cryptocurrencies fetched from CoinGecko")
                sys.exit(1)

            # Import to database
            inserted, updated = self.import_to_database(coins)

            # Log summary
            duration = datetime.now() - start_time
            logger.info(f"\n{'='*50}")
            logger.info(f"Import Summary:")
            logger.info(f"  Duration: {duration.total_seconds():.1f} seconds")
            logger.info(f"  Total fetched: {self.stats['total_fetched']}")
            logger.info(f"  Inserted: {self.stats['inserted']}")
            logger.info(f"  Updated: {self.stats['updated']}")
            logger.info(f"  Skipped: {self.stats['skipped']}")
            logger.info(f"  Failed: {self.stats['failed']}")
            logger.info(f"{'='*50}")

            logger.info("=== CoinGecko Crypto Import Completed Successfully ===")

        except Exception as e:
            logger.error(f"Import failed with error: {e}")
            logger.error("=== CoinGecko Crypto Import Failed ===")
            sys.exit(1)
        finally:
            if self.conn:
                self.conn.close()


if __name__ == "__main__":
    # Parse command line arguments
    import argparse

    parser = argparse.ArgumentParser(description='Import cryptocurrency data from CoinGecko')
    parser.add_argument('--limit', type=int, default=3000,
                       help='Number of top cryptocurrencies to import (default: 3000)')
    parser.add_argument('--test', action='store_true',
                       help='Test mode - only import top 10 cryptocurrencies')

    args = parser.parse_args()

    # Override limit for test mode
    if args.test:
        logger.info("Running in TEST mode - will only import top 10 cryptocurrencies")
        args.limit = 10

    # Run the importer
    importer = CoinGeckoImporter()
    importer.run(args.limit)