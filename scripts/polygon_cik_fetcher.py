#!/usr/bin/env python3
"""
Polygon CIK Fetcher - Updates missing CIK numbers by calling ticker details endpoint
"""

import os
import sys
import time
import logging
import psycopg2
import requests
from datetime import datetime
from typing import Dict, List, Optional
from psycopg2.extras import RealDictCursor

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class PolygonCIKFetcher:
    """Fetches CIK numbers from Polygon ticker details endpoint"""
    
    def __init__(self):
        self.api_key = os.environ.get('POLYGON_API_KEY', 'Q9LhuSPrdj8Fqv9ejYqwXF6AKv7YAsWa')
        self.base_url = "https://api.polygon.io/v3/reference/tickers"
        
        self.db_config = {
            'host': os.environ.get('DB_HOST', 'localhost'),
            'port': int(os.environ.get('DB_PORT', 5432)),
            'database': os.environ.get('DB_NAME', 'investorcenter_db'),
            'user': os.environ.get('DB_USER'),
            'password': os.environ.get('DB_PASSWORD')
        }
        
        self.conn = None
        self.stats = {
            'total_missing': 0,
            'fetched': 0,
            'updated': 0,
            'errors': 0,
            'api_calls': 0
        }
        
        # Rate limiting for paid tier (adjust based on your plan)
        self.requests_per_second = 100  # Paid tier typically allows 100+ req/sec
        self.request_interval = 1.0 / self.requests_per_second
        self.last_request_time = 0
    
    def connect_db(self) -> bool:
        """Connect to PostgreSQL database"""
        try:
            self.conn = psycopg2.connect(**self.db_config)
            logger.info(f"Connected to database: {self.db_config['database']}")
            return True
        except Exception as e:
            logger.error(f"Database connection failed: {e}")
            return False
    
    def get_stocks_without_cik(self, limit: Optional[int] = None) -> List[Dict]:
        """Get stocks that don't have CIK numbers"""
        try:
            with self.conn.cursor(cursor_factory=RealDictCursor) as cursor:
                query = """
                    SELECT 
                        id,
                        symbol,
                        name,
                        exchange,
                        asset_type
                    FROM tickers
                    WHERE 
                        asset_type IN ('stock', 'CS', 'ADRC', 'PFD')
                        AND active = true
                        AND (cik IS NULL OR LENGTH(cik) = 0)
                    ORDER BY 
                        CASE 
                            WHEN exchange IN ('XNAS', 'XNYS', 'XASE') THEN 0
                            ELSE 1
                        END,
                        symbol
                """
                
                if limit:
                    query += f" LIMIT {limit}"
                
                cursor.execute(query)
                stocks = cursor.fetchall()
                
                logger.info(f"Found {len(stocks)} stocks without CIK numbers")
                return stocks
                
        except Exception as e:
            logger.error(f"Failed to get stocks without CIK: {e}")
            return []
    
    def rate_limit(self):
        """Ensure we don't exceed rate limits"""
        current_time = time.time()
        time_since_last = current_time - self.last_request_time
        
        if time_since_last < self.request_interval:
            sleep_time = self.request_interval - time_since_last
            time.sleep(sleep_time)
        
        self.last_request_time = time.time()
    
    def fetch_ticker_details(self, symbol: str) -> Optional[Dict]:
        """Fetch detailed ticker information from Polygon"""
        url = f"{self.base_url}/{symbol}"
        params = {'apiKey': self.api_key}
        
        try:
            self.rate_limit()
            self.stats['api_calls'] += 1
            
            response = requests.get(url, params=params, timeout=10)
            response.raise_for_status()
            
            data = response.json()
            if data.get('status') == 'OK' and 'results' in data:
                return data['results']
            else:
                logger.warning(f"No data found for {symbol}")
                return None
                
        except requests.exceptions.RequestException as e:
            logger.error(f"API request failed for {symbol}: {e}")
            self.stats['errors'] += 1
            return None
    
    def update_ticker_cik(self, ticker_id: int, symbol: str, details: Dict) -> bool:
        """Update ticker with CIK and other details from Polygon"""
        try:
            with self.conn.cursor() as cursor:
                # Extract fields from Polygon response
                cik = details.get('cik', '')
                
                if not cik:
                    logger.debug(f"No CIK found for {symbol}")
                    return False
                
                # Update query with existing columns only
                update_query = """
                    UPDATE tickers
                    SET 
                        cik = %s,
                        composite_figi = %s,
                        share_class_figi = %s,
                        phone_number = %s,
                        address_city = %s,
                        address_state = %s,
                        address_postal = %s,
                        sic_code = %s,
                        sic_description = %s,
                        employees = %s,
                        market_cap = %s,
                        description = %s,
                        logo_url = %s,
                        icon_url = %s,
                        ipo_date = %s,
                        weighted_shares_outstanding = %s,
                        updated_at = CURRENT_TIMESTAMP
                    WHERE id = %s
                """
                
                # Get values with defaults
                address = details.get('address', {})
                branding = details.get('branding', {})
                
                cursor.execute(update_query, (
                    cik,
                    details.get('composite_figi'),
                    details.get('share_class_figi'),
                    details.get('phone_number'),
                    address.get('city'),
                    address.get('state'),
                    address.get('postal_code'),
                    details.get('sic_code'),
                    details.get('sic_description'),
                    details.get('total_employees'),
                    details.get('market_cap'),
                    details.get('description'),
                    branding.get('logo_url'),
                    branding.get('icon_url'),
                    details.get('list_date'),
                    details.get('weighted_shares_outstanding'),
                    ticker_id
                ))
                
                self.conn.commit()
                logger.info(f"Updated {symbol} with CIK: {cik}")
                return True
                
        except Exception as e:
            logger.error(f"Failed to update {symbol}: {e}")
            self.conn.rollback()
            return False
    
    def process_batch(self, stocks: List[Dict]):
        """Process a batch of stocks to fetch their CIK"""
        for i, stock in enumerate(stocks, 1):
            symbol = stock['symbol']
            ticker_id = stock['id']
            
            # Progress logging every 100 stocks
            if i % 100 == 0:
                logger.info(f"Progress: {i}/{len(stocks)} stocks processed")
                logger.info(f"Stats: {self.stats}")
            
            # Fetch ticker details from Polygon
            details = self.fetch_ticker_details(symbol)
            self.stats['fetched'] += 1
            
            if details:
                # Update database with CIK and other details
                if self.update_ticker_cik(ticker_id, symbol, details):
                    self.stats['updated'] += 1
            
            # Brief pause every 1000 requests to be nice to the API
            if self.stats['api_calls'] % 1000 == 0:
                logger.info("Brief pause after 1000 requests...")
                time.sleep(2)
    
    def run(self, limit: Optional[int] = None):
        """Main execution function"""
        start_time = datetime.now()
        logger.info("=== Polygon CIK Fetcher Started ===")
        logger.info(f"Rate limit: {self.requests_per_second} requests/second")
        
        try:
            # Connect to database
            if not self.connect_db():
                logger.error("Failed to connect to database")
                sys.exit(1)
            
            # Get stocks without CIK
            stocks = self.get_stocks_without_cik(limit)
            self.stats['total_missing'] = len(stocks)
            
            if not stocks:
                logger.info("No stocks without CIK found")
                return
            
            logger.info(f"Processing {len(stocks)} stocks...")
            
            # Process all stocks
            self.process_batch(stocks)
            
            # Log statistics
            duration = datetime.now() - start_time
            logger.info(f"Processing completed in {duration.total_seconds():.1f} seconds")
            logger.info(f"Final Statistics:")
            logger.info(f"  Total missing CIK: {self.stats['total_missing']}")
            logger.info(f"  API calls made: {self.stats['api_calls']}")
            logger.info(f"  Tickers fetched: {self.stats['fetched']}")
            logger.info(f"  Tickers updated: {self.stats['updated']}")
            logger.info(f"  Errors: {self.stats['errors']}")
            
            # Calculate success rate
            if self.stats['fetched'] > 0:
                success_rate = (self.stats['updated'] / self.stats['fetched']) * 100
                logger.info(f"  Success rate: {success_rate:.1f}%")
            
            logger.info("=== Polygon CIK Fetcher Completed ===")
            
        except Exception as e:
            logger.error(f"CIK fetcher failed: {e}")
            sys.exit(1)
        finally:
            if self.conn:
                self.conn.close()


if __name__ == "__main__":
    # Allow limit to be passed as argument
    limit = int(sys.argv[1]) if len(sys.argv) > 1 else None
    
    fetcher = PolygonCIKFetcher()
    fetcher.run(limit=limit)