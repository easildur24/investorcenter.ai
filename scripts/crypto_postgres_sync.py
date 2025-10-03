#!/usr/bin/env python3
"""
Sync cryptocurrency metadata from Redis to PostgreSQL

This script:
1. Reads crypto data from Redis (populated by crypto_complete_service.py)
2. Stores/updates ticker metadata in PostgreSQL
3. Maintains historical price snapshots

Run this periodically (e.g., every hour) to keep PostgreSQL in sync.
"""

import redis
import psycopg2
from psycopg2.extras import execute_batch
import json
import logging
from datetime import datetime
import os
import argparse
from typing import List, Dict, Tuple

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class CryptoPostgresSync:
    def __init__(self, db_config: dict = None):
        """Initialize with database configuration"""
        if db_config is None:
            db_config = {
                'host': os.getenv('DB_HOST', 'localhost'),
                'port': os.getenv('DB_PORT', '5433'),
                'user': os.getenv('DB_USER', 'investorcenter'),
                'password': os.getenv('DB_PASSWORD', 'password123'),
                'database': os.getenv('DB_NAME', 'investorcenter_db')
            }

        self.db_config = db_config
        self.redis_client = redis.Redis(
            host='localhost',
            port=6379,
            decode_responses=True
        )
        self.conn = None
        self.cursor = None

    def connect_db(self):
        """Connect to PostgreSQL database"""
        try:
            self.conn = psycopg2.connect(**self.db_config)
            self.cursor = self.conn.cursor()
            logger.info(f"Connected to PostgreSQL: {self.db_config['host']}:{self.db_config['port']}")
            return True
        except Exception as e:
            logger.error(f"Failed to connect to database: {e}")
            return False

    def disconnect_db(self):
        """Close database connection"""
        if self.cursor:
            self.cursor.close()
        if self.conn:
            self.conn.close()

    def get_crypto_data_from_redis(self, limit: int = None) -> List[Dict]:
        """Fetch crypto data from Redis"""
        cryptos = []

        # Get ranked symbols
        symbols = self.redis_client.zrange('crypto:symbols:ranked', 0, -1)

        if limit:
            symbols = symbols[:limit]

        logger.info(f"Fetching {len(symbols)} cryptocurrencies from Redis...")

        for symbol in symbols:
            try:
                data = self.redis_client.get(f'crypto:quote:{symbol}')
                if data:
                    crypto = json.loads(data)
                    cryptos.append(crypto)
            except Exception as e:
                logger.debug(f"Error fetching {symbol}: {e}")

        return cryptos

    def prepare_ticker_data(self, crypto: Dict) -> Tuple:
        """Prepare crypto data for tickers table insertion"""
        return (
            crypto['symbol'],                           # symbol
            crypto['name'],                             # name
            'crypto',                                   # asset_type
            'CRYPTO',                                   # exchange
            'Cryptocurrency',                          # sector
            'Cryptocurrency',                          # industry
            'Global',                                   # country
            'USD',                                      # currency
            int(crypto.get('market_cap', 0)),          # market_cap (bigint)
            f"{crypto['name']} cryptocurrency",        # description
            None,                                       # website
            None,                                       # cik
            None,                                       # ipo_date
            None,                                       # logo_url
            crypto['symbol'],                           # base_currency_symbol
            crypto['name'],                             # base_currency_name
            'USD',                                      # currency_symbol
            'global',                                   # locale
            'crypto',                                   # market
            int(crypto.get('circulating_supply', 0)),  # weighted_shares_outstanding
            True,                                       # active
            None                                        # delisted_date
        )

    def sync_tickers(self, cryptos: List[Dict], batch_size: int = 100):
        """Sync crypto tickers to PostgreSQL"""
        if not cryptos:
            logger.warning("No cryptocurrencies to sync")
            return

        logger.info(f"Syncing {len(cryptos)} cryptocurrencies to PostgreSQL...")

        # Prepare insert query (using only columns that exist in production)
        insert_query = """
            INSERT INTO tickers (
                symbol, name, asset_type, exchange, sector, industry,
                country, currency, market_cap, description, website,
                cik, ipo_date, logo_url, base_currency_symbol,
                base_currency_name, currency_symbol, locale, market,
                weighted_shares_outstanding, active, delisted_date
            ) VALUES (
                %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
            )
            ON CONFLICT (symbol) DO UPDATE SET
                name = EXCLUDED.name,
                asset_type = EXCLUDED.asset_type,
                market_cap = EXCLUDED.market_cap,
                weighted_shares_outstanding = EXCLUDED.weighted_shares_outstanding,
                updated_at = NOW()
        """

        # Prepare data
        ticker_data = []
        for crypto in cryptos:
            try:
                ticker_data.append(self.prepare_ticker_data(crypto))
            except Exception as e:
                logger.debug(f"Error preparing {crypto.get('symbol', '?')}: {e}")

        # Insert in batches
        try:
            execute_batch(self.cursor, insert_query, ticker_data, page_size=batch_size)
            self.conn.commit()
            logger.info(f"✅ Successfully synced {len(ticker_data)} tickers")
        except Exception as e:
            logger.error(f"Error inserting tickers: {e}")
            self.conn.rollback()
            raise

    def store_price_snapshot(self, cryptos: List[Dict], top_n: int = 100):
        """Store price snapshots for top N cryptos (for historical data)"""
        if not cryptos:
            return

        # Only store top N for historical tracking
        top_cryptos = sorted(cryptos, key=lambda x: x.get('rank', 99999))[:top_n]

        logger.info(f"Storing price snapshots for top {len(top_cryptos)} cryptocurrencies...")

        insert_query = """
            INSERT INTO stock_prices (
                symbol, price, volume, change, change_percent, timestamp
            ) VALUES (%s, %s, %s, %s, %s, %s)
        """

        price_data = []
        timestamp = datetime.utcnow()

        for crypto in top_cryptos:
            try:
                # Calculate change amount from percentage
                price = crypto.get('price', 0)
                change_percent = crypto.get('change_24h', 0)
                change_amount = (price * change_percent / 100) if price > 0 else 0

                price_data.append((
                    crypto['symbol'],
                    price,
                    crypto.get('volume_24h', 0),
                    change_amount,
                    change_percent,
                    timestamp
                ))
            except Exception as e:
                logger.debug(f"Error preparing price for {crypto.get('symbol', '?')}: {e}")

        try:
            execute_batch(self.cursor, insert_query, price_data)
            self.conn.commit()
            logger.info(f"✅ Stored {len(price_data)} price snapshots")
        except Exception as e:
            logger.error(f"Error storing price snapshots: {e}")
            self.conn.rollback()

    def get_stats(self) -> Dict:
        """Get statistics about synced data"""
        stats = {}

        try:
            # Count crypto tickers
            self.cursor.execute(
                "SELECT COUNT(*) FROM tickers WHERE asset_type = 'crypto'"
            )
            stats['total_cryptos'] = self.cursor.fetchone()[0]

            # Get top 10 by market cap
            self.cursor.execute("""
                SELECT symbol, name, market_cap
                FROM tickers
                WHERE asset_type = 'crypto' AND market_cap > 0
                ORDER BY market_cap DESC
                LIMIT 10
            """)
            stats['top_10'] = self.cursor.fetchall()

            # Count today's price records
            self.cursor.execute("""
                SELECT COUNT(*)
                FROM stock_prices sp
                JOIN tickers t ON sp.symbol = t.symbol
                WHERE t.asset_type = 'crypto'
                AND DATE(sp.timestamp) = CURRENT_DATE
            """)
            stats['price_snapshots_today'] = self.cursor.fetchone()[0]

        except Exception as e:
            logger.error(f"Error getting stats: {e}")

        return stats

    def run(self, limit: int = None, store_history: bool = True):
        """Main sync process"""
        logger.info("=" * 60)
        logger.info("🔄 CRYPTO POSTGRESQL SYNC")
        logger.info("=" * 60)

        if not self.connect_db():
            return False

        try:
            # Get crypto data from Redis
            cryptos = self.get_crypto_data_from_redis(limit)

            if not cryptos:
                logger.warning("No cryptocurrency data found in Redis")
                logger.info("Make sure crypto_complete_service.py is running")
                return False

            logger.info(f"Found {len(cryptos)} cryptocurrencies in Redis")

            # Sync to PostgreSQL
            self.sync_tickers(cryptos)

            # Store price snapshots for historical data
            if store_history:
                self.store_price_snapshot(cryptos, top_n=100)

            # Show statistics
            stats = self.get_stats()

            logger.info("\n📊 SYNC STATISTICS:")
            logger.info(f"Total cryptos in DB: {stats.get('total_cryptos', 0)}")
            logger.info(f"Price snapshots today: {stats.get('price_snapshots_today', 0)}")

            if stats.get('top_10'):
                logger.info("\nTop 10 by Market Cap:")
                for symbol, name, mcap in stats['top_10']:
                    logger.info(f"  {symbol:6} {name[:30]:30} ${mcap/1e9:>8.2f}B")

            return True

        except Exception as e:
            logger.error(f"Sync failed: {e}")
            return False
        finally:
            self.disconnect_db()


def main():
    parser = argparse.ArgumentParser(description='Sync crypto data to PostgreSQL')
    parser.add_argument(
        '--limit',
        type=int,
        help='Limit number of cryptos to sync (default: all)'
    )
    parser.add_argument(
        '--no-history',
        action='store_true',
        help='Skip storing historical price snapshots'
    )
    parser.add_argument(
        '--test',
        action='store_true',
        help='Test mode - sync only top 10'
    )

    args = parser.parse_args()

    # Configure based on arguments
    limit = 10 if args.test else args.limit
    store_history = not args.no_history and not args.test

    # Run sync
    syncer = CryptoPostgresSync()
    success = syncer.run(limit=limit, store_history=store_history)

    if success:
        logger.info("\n✅ Sync completed successfully!")
    else:
        logger.error("\n❌ Sync failed!")
        exit(1)


if __name__ == "__main__":
    main()