#!/usr/bin/env python3
"""
Polygon Volume Data Update Script

This script fetches daily volume and price data from Polygon.io API
and updates the tickers table with the latest trading information.

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
from typing import Dict, List, Optional
from psycopg2.extras import RealDictCursor, execute_batch
from time import sleep

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class PolygonVolumeUpdater:
    """Fetches and updates volume data from Polygon API"""
    
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
        
        self.base_url = "https://api.polygon.io"
        self.conn = None
        self.stats = {
            'processed': 0,
            'updated': 0,
            'failed': 0,
            'api_calls': 0
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
    
    def get_active_tickers(self, limit: int = 1000) -> List[Dict]:
        """Get list of active tickers that need volume updates"""
        try:
            with self.conn.cursor(cursor_factory=RealDictCursor) as cursor:
                # Get tickers that haven't been updated today or have no volume data
                cursor.execute("""
                    SELECT symbol, asset_type, market 
                    FROM tickers 
                    WHERE active = true 
                    AND (
                        last_trade_timestamp IS NULL 
                        OR last_trade_timestamp < CURRENT_DATE
                    )
                    ORDER BY 
                        CASE 
                            WHEN market_cap IS NOT NULL THEN market_cap 
                            ELSE 0 
                        END DESC
                    LIMIT %s
                """, (limit,))
                
                return cursor.fetchall()
        except Exception as e:
            logger.error(f"Failed to get active tickers: {e}")
            return []
    
    def fetch_ticker_snapshot(self, symbol: str, asset_type: str = 'stocks') -> Optional[Dict]:
        """Fetch current snapshot data for a ticker"""
        try:
            # Map asset types to Polygon API endpoints
            endpoint_map = {
                'stock': f'/v2/snapshot/locale/us/markets/stocks/tickers/{symbol}',
                'etf': f'/v2/snapshot/locale/us/markets/stocks/tickers/{symbol}',
                'crypto': f'/v2/snapshot/locale/global/markets/crypto/tickers/X:{symbol}',
                'index': f'/v1/indicators/sma/{symbol}'  # Indices use different endpoint
            }
            
            endpoint = endpoint_map.get(asset_type, endpoint_map['stock'])
            url = f"{self.base_url}{endpoint}"
            
            params = {'apiKey': self.api_key}
            response = requests.get(url, params=params, timeout=10)
            
            self.stats['api_calls'] += 1
            
            if response.status_code == 200:
                data = response.json()
                if data.get('status') == 'OK':
                    return data.get('ticker', {})
            elif response.status_code == 429:
                logger.warning(f"Rate limit hit for {symbol}, waiting...")
                sleep(12)  # Wait for rate limit
                return None
            
            return None
            
        except Exception as e:
            logger.error(f"Failed to fetch snapshot for {symbol}: {e}")
            return None
    
    def fetch_aggregate_bars(self, symbol: str, days: int = 90) -> Optional[Dict]:
        """Fetch aggregate bar data for volume averages"""
        try:
            to_date = datetime.now().strftime('%Y-%m-%d')
            from_date = (datetime.now() - timedelta(days=days)).strftime('%Y-%m-%d')
            
            url = f"{self.base_url}/v2/aggs/ticker/{symbol}/range/1/day/{from_date}/{to_date}"
            params = {
                'apiKey': self.api_key,
                'adjusted': 'true',
                'sort': 'desc',
                'limit': days
            }
            
            response = requests.get(url, params=params, timeout=10)
            self.stats['api_calls'] += 1
            
            if response.status_code == 200:
                data = response.json()
                if data.get('status') == 'OK' and data.get('results'):
                    results = data['results']
                    
                    # Calculate average volumes
                    volumes = [bar['v'] for bar in results if 'v' in bar]
                    
                    return {
                        'avg_volume_30d': sum(volumes[:30]) // min(30, len(volumes)) if volumes else None,
                        'avg_volume_90d': sum(volumes) // len(volumes) if volumes else None,
                        'week_52_high': max([bar['h'] for bar in results if 'h' in bar], default=None),
                        'week_52_low': min([bar['l'] for bar in results if 'l' in bar], default=None)
                    }
            
            return None
            
        except Exception as e:
            logger.error(f"Failed to fetch aggregates for {symbol}: {e}")
            return None
    
    def update_ticker_volume(self, ticker: Dict, snapshot: Dict, aggregates: Optional[Dict]) -> bool:
        """Update ticker with volume and price data"""
        try:
            symbol = ticker['symbol']
            
            with self.conn.cursor() as cursor:
                # Prepare update data
                update_data = {
                    'volume': snapshot.get('day', {}).get('v'),
                    'current_price': snapshot.get('day', {}).get('c'),
                    'previous_close': snapshot.get('prevDay', {}).get('c'),
                    'day_high': snapshot.get('day', {}).get('h'),
                    'day_low': snapshot.get('day', {}).get('l'),
                    'day_open': snapshot.get('day', {}).get('o'),
                    'vwap': snapshot.get('day', {}).get('vw'),
                    'last_trade_timestamp': datetime.now(timezone.utc)
                }
                
                # Add aggregate data if available
                if aggregates:
                    update_data.update({
                        'avg_volume_30d': aggregates.get('avg_volume_30d'),
                        'avg_volume_90d': aggregates.get('avg_volume_90d'),
                        'week_52_high': aggregates.get('week_52_high'),
                        'week_52_low': aggregates.get('week_52_low')
                    })
                
                # Build UPDATE query dynamically
                set_clauses = []
                values = []
                for key, value in update_data.items():
                    if value is not None:
                        set_clauses.append(f"{key} = %s")
                        values.append(value)
                
                if set_clauses:
                    values.append(symbol)  # For WHERE clause
                    query = f"""
                        UPDATE tickers 
                        SET {', '.join(set_clauses)}, updated_at = CURRENT_TIMESTAMP
                        WHERE symbol = %s
                    """
                    
                    cursor.execute(query, values)
                    self.conn.commit()
                    self.stats['updated'] += 1
                    return True
                
            return False
            
        except Exception as e:
            logger.error(f"Failed to update {ticker['symbol']}: {e}")
            self.conn.rollback()
            self.stats['failed'] += 1
            return False
    
    def run(self, batch_size: int = 100):
        """Main execution function"""
        start_time = datetime.now()
        logger.info("=== Polygon Volume Data Update Started ===")
        
        try:
            # Connect to database
            if not self.connect_db():
                logger.error("Failed to connect to database")
                sys.exit(1)
            
            # Get tickers to update
            tickers = self.get_active_tickers(limit=batch_size)
            logger.info(f"Found {len(tickers)} tickers to update")
            
            if not tickers:
                logger.info("No tickers need volume updates")
                return
            
            # Process tickers
            for i, ticker in enumerate(tickers, 1):
                symbol = ticker['symbol']
                asset_type = ticker.get('asset_type', 'stock')
                
                # Rate limiting - 5 requests per minute on free tier
                if self.stats['api_calls'] > 0 and self.stats['api_calls'] % 5 == 0:
                    logger.info("Rate limiting pause (12 seconds)...")
                    sleep(12)
                
                logger.info(f"Processing {i}/{len(tickers)}: {symbol}")
                
                # Fetch snapshot data
                snapshot = self.fetch_ticker_snapshot(symbol, asset_type)
                if not snapshot:
                    self.stats['failed'] += 1
                    continue
                
                # Fetch aggregate data for averages (optional, only for important tickers)
                aggregates = None
                if i <= 20:  # Only fetch aggregates for top 20 tickers to save API calls
                    aggregates = self.fetch_aggregate_bars(symbol)
                
                # Update database
                if self.update_ticker_volume(ticker, snapshot, aggregates):
                    logger.debug(f"Updated volume data for {symbol}")
                
                self.stats['processed'] += 1
            
            # Log statistics
            duration = datetime.now() - start_time
            logger.info(f"Update completed in {duration.total_seconds():.1f} seconds")
            logger.info(f"Statistics: {json.dumps(self.stats, indent=2)}")
            
            if self.stats['updated'] > 0:
                logger.info(f"Updated volume data for {self.stats['updated']} tickers")
            
            logger.info("=== Polygon Volume Data Update Completed ===")
            
        except Exception as e:
            logger.error(f"Volume update failed: {e}")
            sys.exit(1)
        finally:
            if self.conn:
                self.conn.close()


if __name__ == "__main__":
    # Allow batch size to be passed as argument
    batch_size = int(sys.argv[1]) if len(sys.argv) > 1 else 100
    
    updater = PolygonVolumeUpdater()
    updater.run(batch_size=batch_size)