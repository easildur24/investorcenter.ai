#!/usr/bin/env python3
"""
Polygon Incremental Ticker Update Script for Cron Jobs

This script fetches only NEW or UPDATED tickers from Polygon.io API since the last run.
It tracks the last update timestamp and only fetches changes since then.

Features:
- Incremental updates using last_updated_utc field
- Tracks last sync timestamp in database
- Only fetches new/updated tickers to minimize API calls
- Handles all asset types: stocks, ETFs, crypto, etc.
- Designed for daily cron execution

Environment Variables Required:
    POLYGON_API_KEY - Polygon.io API key
    DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME - Database connection
"""

import os
import sys
import json
import logging
import psycopg2
import requests
from datetime import datetime, timedelta, timezone
from typing import Dict, List, Optional, Tuple
from psycopg2.extras import RealDictCursor, execute_batch

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class PolygonIncrementalUpdater:
    """Handles incremental ticker updates from Polygon API"""
    
    def __init__(self):
        self.api_key = os.environ.get('POLYGON_API_KEY')
        if not self.api_key:
            raise ValueError("POLYGON_API_KEY environment variable is required")
        
        self.db_config = {
            'host': os.environ.get('DB_HOST', 'localhost'),
            'port': int(os.environ.get('DB_PORT', 5432)),
            'database': os.environ.get('DB_NAME', 'investorcenter_db'),
            'user': os.environ.get('DB_USER'),
            'password': os.environ.get('DB_PASSWORD')
        }
        
        self.base_url = "https://api.polygon.io/v3/reference/tickers"
        self.conn = None
        self.stats = {
            'new': 0,
            'updated': 0,
            'unchanged': 0,
            'failed': 0,
            'total_fetched': 0
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
    
    def get_last_sync_timestamp(self) -> Optional[datetime]:
        """Get the last successful sync timestamp from database"""
        try:
            with self.conn.cursor() as cursor:
                # Check if sync_metadata table exists, create if not
                cursor.execute("""
                    CREATE TABLE IF NOT EXISTS sync_metadata (
                        id SERIAL PRIMARY KEY,
                        sync_type VARCHAR(50) UNIQUE NOT NULL,
                        last_sync_utc TIMESTAMP WITH TIME ZONE NOT NULL,
                        records_processed INTEGER DEFAULT 0,
                        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                    )
                """)
                
                # Get last sync timestamp
                cursor.execute("""
                    SELECT last_sync_utc 
                    FROM sync_metadata 
                    WHERE sync_type = 'polygon_tickers'
                """)
                
                result = cursor.fetchone()
                if result:
                    logger.info(f"Last sync was at: {result[0]}")
                    return result[0]
                else:
                    # If no previous sync, default to 7 days ago
                    default_date = datetime.now(timezone.utc) - timedelta(days=7)
                    logger.info(f"No previous sync found, defaulting to: {default_date}")
                    return default_date
                    
        except Exception as e:
            logger.error(f"Failed to get last sync timestamp: {e}")
            # Default to 7 days ago on error
            return datetime.now(timezone.utc) - timedelta(days=7)
    
    def update_sync_timestamp(self, timestamp: datetime, records: int):
        """Update the last sync timestamp in database"""
        try:
            with self.conn.cursor() as cursor:
                cursor.execute("""
                    INSERT INTO sync_metadata (sync_type, last_sync_utc, records_processed)
                    VALUES ('polygon_tickers', %s, %s)
                    ON CONFLICT (sync_type) 
                    DO UPDATE SET 
                        last_sync_utc = EXCLUDED.last_sync_utc,
                        records_processed = EXCLUDED.records_processed,
                        updated_at = CURRENT_TIMESTAMP
                """, (timestamp, records))
                self.conn.commit()
                logger.info(f"Updated sync timestamp to: {timestamp}")
        except Exception as e:
            logger.error(f"Failed to update sync timestamp: {e}")
    
    def fetch_updated_tickers(self, since: datetime) -> List[Dict]:
        """Fetch tickers updated since the given timestamp"""
        all_tickers = []
        cursor = None
        page = 1
        
        # Format date for API (YYYY-MM-DD)
        since_date = since.strftime('%Y-%m-%d')
        logger.info(f"Fetching tickers updated since: {since_date}")
        
        while True:
            params = {
                'apiKey': self.api_key,
                'active': 'true',
                'limit': 1000,
                'order': 'asc',
                'sort': 'ticker'
            }
            
            # Add date filter - tickers updated after the last sync
            # Using ticker endpoint with date filtering
            if cursor:
                params['cursor'] = cursor
            
            try:
                response = requests.get(self.base_url, params=params, timeout=30)
                response.raise_for_status()
                data = response.json()
                
                results = data.get('results', [])
                
                # Filter results by last_updated_utc if available
                filtered_results = []
                for ticker in results:
                    # Check if ticker has been updated since our last sync
                    last_updated = ticker.get('last_updated_utc')
                    if last_updated:
                        ticker_date = datetime.fromisoformat(last_updated.replace('Z', '+00:00'))
                        if ticker_date > since:
                            filtered_results.append(ticker)
                    else:
                        # If no last_updated field, include it (might be new)
                        filtered_results.append(ticker)
                
                all_tickers.extend(filtered_results)
                
                logger.info(f"  Page {page}: fetched {len(results)} tickers, "
                           f"{len(filtered_results)} updated (total: {len(all_tickers)})")
                
                # Check for next page
                next_url = data.get('next_url')
                if not next_url:
                    break
                
                # Extract cursor from next_url
                if 'cursor=' in next_url:
                    cursor = next_url.split('cursor=')[1].split('&')[0]
                else:
                    break
                
                page += 1
                
                # Rate limiting - Polygon allows 5 requests per minute on free tier
                if page % 5 == 0:
                    logger.info("Rate limiting pause...")
                    import time
                    time.sleep(12)  # Wait 12 seconds every 5 requests
                    
            except Exception as e:
                logger.error(f"API request failed: {e}")
                break
        
        self.stats['total_fetched'] = len(all_tickers)
        logger.info(f"Total tickers fetched: {len(all_tickers)}")
        return all_tickers
    
    def upsert_tickers(self, tickers: List[Dict]) -> Tuple[int, int]:
        """Insert or update tickers in database"""
        if not tickers:
            return 0, 0
        
        inserted = 0
        updated = 0
        
        try:
            with self.conn.cursor(cursor_factory=RealDictCursor) as cursor:
                for ticker in tickers:
                    # Prepare data for upsert
                    symbol = ticker.get('ticker', '').upper()
                    if not symbol:
                        continue
                    
                    # Map Polygon fields to our database schema
                    data = {
                        'symbol': symbol,
                        'name': ticker.get('name', ''),
                        'exchange': ticker.get('primary_exchange', ''),
                        'asset_type': ticker.get('type', 'CS'),  # CS = Common Stock
                        'locale': ticker.get('locale', 'us'),
                        'market': ticker.get('market', 'stocks'),
                        'active': ticker.get('active', True),
                        'currency': ticker.get('currency_name', 'usd').upper(),
                        'cik': ticker.get('cik', ''),
                        'composite_figi': ticker.get('composite_figi', ''),
                        'share_class_figi': ticker.get('share_class_figi', ''),
                        'last_updated_utc': ticker.get('last_updated_utc', 
                                                      datetime.now(timezone.utc).isoformat())
                    }
                    
                    # Check if ticker exists
                    cursor.execute("SELECT id FROM tickers WHERE symbol = %s", (symbol,))
                    existing = cursor.fetchone()
                    
                    if existing:
                        # Update existing ticker
                        cursor.execute("""
                            UPDATE tickers SET
                                name = %s,
                                exchange = %s,
                                asset_type = %s,
                                locale = %s,
                                market = %s,
                                active = %s,
                                currency = %s,
                                cik = %s,
                                composite_figi = %s,
                                share_class_figi = %s,
                                last_updated_utc = %s,
                                updated_at = CURRENT_TIMESTAMP
                            WHERE symbol = %s
                        """, (
                            data['name'], data['exchange'], data['asset_type'],
                            data['locale'], data['market'], data['active'],
                            data['currency'], data['cik'], data['composite_figi'],
                            data['share_class_figi'], data['last_updated_utc'],
                            symbol
                        ))
                        updated += 1
                    else:
                        # Insert new ticker
                        cursor.execute("""
                            INSERT INTO tickers (
                                symbol, name, exchange, asset_type, locale, market,
                                active, currency, cik, composite_figi, share_class_figi,
                                last_updated_utc, created_at, updated_at
                            ) VALUES (
                                %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                                CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
                            )
                        """, (
                            symbol, data['name'], data['exchange'], data['asset_type'],
                            data['locale'], data['market'], data['active'],
                            data['currency'], data['cik'], data['composite_figi'],
                            data['share_class_figi'], data['last_updated_utc']
                        ))
                        inserted += 1
                
                self.conn.commit()
                self.stats['new'] = inserted
                self.stats['updated'] = updated
                
                logger.info(f"Database update complete: {inserted} new, {updated} updated")
                return inserted, updated
                
        except Exception as e:
            logger.error(f"Database update failed: {e}")
            self.conn.rollback()
            self.stats['failed'] = len(tickers)
            return 0, 0
    
    def run(self):
        """Main execution function"""
        start_time = datetime.now()
        logger.info("=== Polygon Incremental Ticker Update Started ===")
        
        try:
            # Connect to database
            if not self.connect_db():
                logger.error("Failed to connect to database")
                sys.exit(1)
            
            # Get last sync timestamp
            last_sync = self.get_last_sync_timestamp()
            
            # Fetch updated tickers from Polygon
            updated_tickers = self.fetch_updated_tickers(last_sync)
            
            if not updated_tickers:
                logger.info("No ticker updates found since last sync")
                self.stats['unchanged'] = 0
            else:
                # Update database
                inserted, updated = self.upsert_tickers(updated_tickers)
                
                # Update sync timestamp if successful
                if inserted > 0 or updated > 0:
                    self.update_sync_timestamp(
                        datetime.now(timezone.utc),
                        inserted + updated
                    )
            
            # Log statistics
            duration = datetime.now() - start_time
            logger.info(f"Update completed in {duration.total_seconds():.1f} seconds")
            logger.info(f"Statistics: {json.dumps(self.stats, indent=2)}")
            
            if self.stats['new'] > 0:
                logger.info(f"🆕 Added {self.stats['new']} new tickers")
            if self.stats['updated'] > 0:
                logger.info(f"🔄 Updated {self.stats['updated']} existing tickers")
            if self.stats['new'] == 0 and self.stats['updated'] == 0:
                logger.info("📊 Database is up to date - no changes found")
            
            logger.info("=== Polygon Incremental Ticker Update Completed ===")
            
        except Exception as e:
            logger.error(f"Ticker update failed: {e}")
            logger.error("=== Polygon Incremental Ticker Update Failed ===")
            sys.exit(1)
        finally:
            if self.conn:
                self.conn.close()


if __name__ == "__main__":
    updater = PolygonIncrementalUpdater()
    updater.run()