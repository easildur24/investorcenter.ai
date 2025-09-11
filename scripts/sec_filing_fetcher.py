#!/usr/bin/env python3
# Run with: scripts/venv/bin/python scripts/sec_filing_fetcher.py
"""
SEC Filing Fetcher Service

Fetches SEC filings (10-K, 10-Q) using CIK numbers already stored in our database
from Polygon API data.
"""

import os
import sys
import json
import time
import logging
import psycopg2
import requests
from datetime import datetime, timedelta, timezone
from typing import Dict, List, Optional, Tuple
from psycopg2.extras import RealDictCursor

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class SECFilingFetcher:
    """Fetches SEC filings using CIK from database"""
    
    def __init__(self):
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
            'filings_found': 0,
            'filings_saved': 0,
            'errors': 0
        }
        
        # Rate limiting: SEC allows 10 requests per second
        self.last_request_time = 0
        self.min_request_interval = 0.1  # 100ms between requests (10 req/sec)
    
    def connect_db(self) -> bool:
        """Connect to PostgreSQL database"""
        try:
            self.conn = psycopg2.connect(**self.db_config)
            logger.info(f"Connected to database: {self.db_config['database']}")
            return True
        except Exception as e:
            logger.error(f"Database connection failed: {e}")
            return False
    
    def get_stocks_with_cik(self, limit: int = None) -> List[Dict]:
        """Get stocks that have CIK numbers from Polygon data"""
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
                        AND asset_type IN ('stock', 'CS', 'ADRC', 'PFD')  -- Common Stock, ADR, Preferred
                        AND active = true
                    ORDER BY market_cap DESC NULLS LAST
                """
                
                if limit:
                    query += f" LIMIT {limit}"
                
                cursor.execute(query)
                stocks = cursor.fetchall()
                
                logger.info(f"Found {len(stocks)} stocks with CIK numbers")
                return stocks
                
        except Exception as e:
            logger.error(f"Failed to get stocks with CIK: {e}")
            return []
    
    def rate_limit(self):
        """Ensure we don't exceed SEC's rate limit"""
        current_time = time.time()
        time_since_last_request = current_time - self.last_request_time
        
        if time_since_last_request < self.min_request_interval:
            sleep_time = self.min_request_interval - time_since_last_request
            time.sleep(sleep_time)
        
        self.last_request_time = time.time()
    
    def fetch_company_submissions(self, cik: str) -> Optional[Dict]:
        """Fetch list of filings for a company"""
        # Ensure CIK is 10 digits with leading zeros
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
    
    def fetch_company_facts(self, cik: str) -> Optional[Dict]:
        """Fetch XBRL facts (structured financial data)"""
        cik_padded = cik.zfill(10)
        url = f"{self.base_url}/api/xbrl/companyfacts/CIK{cik_padded}.json"
        
        try:
            self.rate_limit()
            response = requests.get(url, headers=self.headers, timeout=30)
            response.raise_for_status()
            
            data = response.json()
            return data
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to fetch facts for CIK {cik}: {e}")
            return None
    
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
                
                logger.debug(f"Saved filing {filing['accession_number']} for {symbol}")
                return True
                
        except Exception as e:
            logger.error(f"Failed to save filing: {e}")
            self.conn.rollback()
            return False
    
    def process_company_filings(self, stock: Dict):
        """Process all filings for a company"""
        symbol = stock['symbol']
        cik = stock['cik']
        ticker_id = stock['ticker_id']
        
        logger.info(f"Processing {symbol} (CIK: {cik})")
        
        # Fetch submissions
        submissions = self.fetch_company_submissions(cik)
        if not submissions:
            self.stats['errors'] += 1
            return
        
        # Extract recent filings
        recent_filings = submissions.get('filings', {}).get('recent', {})
        if not recent_filings:
            logger.warning(f"No recent filings found for {symbol}")
            return
        
        # Process each filing
        forms = recent_filings.get('form', [])
        filing_dates = recent_filings.get('filingDate', [])
        report_dates = recent_filings.get('reportDate', [])
        accession_numbers = recent_filings.get('accessionNumber', [])
        primary_documents = recent_filings.get('primaryDocument', [])
        primary_doc_descriptions = recent_filings.get('primaryDocDescription', [])
        
        filings_saved = 0
        
        for i in range(min(100, len(forms))):  # Process last 100 filings
            # Only process 10-K and 10-Q
            if forms[i] not in ['10-K', '10-Q', '10-K/A', '10-Q/A']:
                continue
            
            filing = {
                'form': forms[i],
                'filing_date': filing_dates[i] if i < len(filing_dates) else None,
                'report_date': report_dates[i] if i < len(report_dates) else None,
                'accession_number': accession_numbers[i] if i < len(accession_numbers) else None,
                'primary_document': primary_documents[i] if i < len(primary_documents) else None,
                'primary_doc_description': primary_doc_descriptions[i] if i < len(primary_doc_descriptions) else None
            }
            
            if self.save_filing_metadata(ticker_id, symbol, cik, filing):
                filings_saved += 1
                self.stats['filings_saved'] += 1
            
            self.stats['filings_found'] += 1
        
        logger.info(f"  Saved {filings_saved} new filings for {symbol}")
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
        logger.info("=== SEC Filing Fetcher Started ===")
        
        try:
            # Connect to database
            if not self.connect_db():
                logger.error("Failed to connect to database")
                sys.exit(1)
            
            # Create tables if they don't exist
            self.ensure_tables_exist()
            
            # Get stocks with CIK
            stocks = self.get_stocks_with_cik(limit)
            
            if not stocks:
                logger.info("No stocks with CIK found")
                return
            
            # Process each company
            for stock in stocks:
                try:
                    self.process_company_filings(stock)
                except Exception as e:
                    logger.error(f"Error processing {stock['symbol']}: {e}")
                    self.stats['errors'] += 1
                    continue
            
            # Log statistics
            duration = datetime.now() - start_time
            logger.info(f"Processing completed in {duration.total_seconds():.1f} seconds")
            logger.info(f"Statistics:")
            logger.info(f"  Companies processed: {self.stats['companies_processed']}")
            logger.info(f"  Filings found: {self.stats['filings_found']}")
            logger.info(f"  Filings saved: {self.stats['filings_saved']}")
            logger.info(f"  Errors: {self.stats['errors']}")
            
            logger.info("=== SEC Filing Fetcher Completed ===")
            
        except Exception as e:
            logger.error(f"Filing fetcher failed: {e}")
            sys.exit(1)
        finally:
            if self.conn:
                self.conn.close()
    
    def ensure_tables_exist(self):
        """Ensure SEC filing tables exist"""
        try:
            with self.conn.cursor() as cursor:
                # Check if tables exist, create if not
                cursor.execute("""
                    CREATE TABLE IF NOT EXISTS sec_filings (
                        id SERIAL PRIMARY KEY,
                        ticker_id INTEGER,
                        symbol VARCHAR(20) NOT NULL,
                        cik VARCHAR(10) NOT NULL,
                        filing_type VARCHAR(20) NOT NULL,
                        filing_date DATE,
                        report_date DATE,
                        accession_number VARCHAR(50) UNIQUE NOT NULL,
                        primary_document VARCHAR(255),
                        primary_doc_description VARCHAR(255),
                        size_bytes INTEGER,
                        is_processed BOOLEAN DEFAULT false,
                        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                    )
                """)
                
                cursor.execute("""
                    CREATE TABLE IF NOT EXISTS sec_sync_status (
                        id SERIAL PRIMARY KEY,
                        symbol VARCHAR(20) UNIQUE NOT NULL,
                        cik VARCHAR(10),
                        last_filing_check TIMESTAMP WITH TIME ZONE,
                        last_successful_sync TIMESTAMP WITH TIME ZONE,
                        total_filings_count INTEGER DEFAULT 0,
                        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                        updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                    )
                """)
                
                self.conn.commit()
                
        except Exception as e:
            logger.error(f"Failed to ensure tables exist: {e}")
            self.conn.rollback()


if __name__ == "__main__":
    # Allow limit to be passed as argument
    # Pass 0 or no argument to process ALL companies
    limit = int(sys.argv[1]) if len(sys.argv) > 1 else None  # Default to ALL companies
    
    if limit == 0:
        limit = None  # 0 means process all
    
    fetcher = SECFilingFetcher()
    fetcher.run(limit=limit)