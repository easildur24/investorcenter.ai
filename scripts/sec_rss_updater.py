#!/usr/bin/env python3
"""
SEC RSS Feed Updater - Fetches new SEC filings via RSS feeds
Updates every 10 minutes during business hours
"""

import os
import sys
import time
import logging
import psycopg2
import requests
import feedparser
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Set
from psycopg2.extras import RealDictCursor
import xml.etree.ElementTree as ET

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class SECRSSUpdater:
    """Fetches new SEC filings via RSS feeds and updates database"""
    
    def __init__(self):
        # RSS feed URLs
        self.rss_feeds = {
            'all_filings': 'https://www.sec.gov/Archives/edgar/usgaap.rss.xml',
            'company_filings': 'https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=&company=&dateb=&owner=include&start=0&count=100&output=atom',
            'latest_filings': 'https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&owner=include&output=atom',
            'form_10k': 'https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=10-K&owner=include&output=atom',
            'form_10q': 'https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=10-Q&owner=include&output=atom',
            'form_8k': 'https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=8-K&owner=include&output=atom'
        }
        
        self.db_config = {
            'host': os.environ.get('DB_HOST', 'localhost'),
            'port': int(os.environ.get('DB_PORT', 5432)),
            'database': os.environ.get('DB_NAME', 'investorcenter_db'),
            'user': os.environ.get('DB_USER'),
            'password': os.environ.get('DB_PASSWORD')
        }
        
        self.conn = None
        self.stats = {
            'feeds_checked': 0,
            'new_filings': 0,
            'duplicate_filings': 0,
            'companies_updated': 0,
            'errors': 0
        }
        
        # Rate limiting
        self.request_interval = 0.1  # 10 requests per second max
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
    
    def rate_limit(self):
        """Ensure we don't exceed rate limits"""
        current_time = time.time()
        time_since_last = current_time - self.last_request_time
        
        if time_since_last < self.request_interval:
            sleep_time = self.request_interval - time_since_last
            time.sleep(sleep_time)
        
        self.last_request_time = time.time()
    
    def get_existing_accession_numbers(self) -> Set[str]:
        """Get all existing accession numbers to avoid duplicates"""
        try:
            with self.conn.cursor() as cursor:
                query = """
                    SELECT DISTINCT accession_number 
                    FROM sec_filings 
                    WHERE accession_number IS NOT NULL
                """
                cursor.execute(query)
                return {row[0] for row in cursor.fetchall()}
        except Exception as e:
            logger.error(f"Failed to get existing accession numbers: {e}")
            return set()
    
    def get_tracked_ciks(self) -> Dict[str, int]:
        """Get CIKs of companies we're tracking"""
        try:
            with self.conn.cursor() as cursor:
                query = """
                    SELECT DISTINCT cik, id 
                    FROM tickers 
                    WHERE cik IS NOT NULL 
                    AND LENGTH(cik) > 0
                """
                cursor.execute(query)
                return {row[0]: row[1] for row in cursor.fetchall()}
        except Exception as e:
            logger.error(f"Failed to get tracked CIKs: {e}")
            return {}
    
    def parse_rss_feed(self, feed_url: str) -> List[Dict]:
        """Parse RSS/Atom feed and extract filing information"""
        try:
            self.rate_limit()
            
            # Set user agent as required by SEC
            headers = {
                'User-Agent': 'InvestorCenter/1.0 (contact@investorcenter.ai)'
            }
            
            response = requests.get(feed_url, headers=headers, timeout=30)
            response.raise_for_status()
            
            # Parse the feed
            feed = feedparser.parse(response.text)
            
            filings = []
            for entry in feed.entries:
                filing = self.extract_filing_info(entry)
                if filing:
                    filings.append(filing)
            
            logger.info(f"Parsed {len(filings)} filings from RSS feed")
            return filings
            
        except Exception as e:
            logger.error(f"Failed to parse RSS feed {feed_url}: {e}")
            self.stats['errors'] += 1
            return []
    
    def extract_filing_info(self, entry: Dict) -> Optional[Dict]:
        """Extract filing information from RSS entry"""
        try:
            # Extract basic info
            title = entry.get('title', '')
            link = entry.get('link', '')
            summary = entry.get('summary', '')
            published = entry.get('published', '')
            
            # Parse title for form type and company info
            # Format: "10-K - APPLE INC (0000320193) (Filer)"
            parts = title.split(' - ')
            if len(parts) >= 2:
                form_type = parts[0].strip()
                company_info = parts[1]
                
                # Extract CIK from parentheses
                import re
                cik_match = re.search(r'\((\d{10})\)', company_info)
                if cik_match:
                    cik = cik_match.group(1).lstrip('0')  # Remove leading zeros
                else:
                    # Try to extract from summary or link
                    cik_match = re.search(r'CIK=(\d+)', link)
                    if cik_match:
                        cik = cik_match.group(1)
                    else:
                        return None
                
                # Extract company name
                company_name = company_info.split('(')[0].strip()
                
                # Extract accession number from link
                accession_match = re.search(r'AccessionNumber=(\d{10}-\d{2}-\d{6})', link)
                if not accession_match:
                    # Try alternate format in link
                    accession_match = re.search(r'/(\d{10}-\d{2}-\d{6})/', link)
                
                accession_number = accession_match.group(1) if accession_match else None
                
                # Parse filing date
                try:
                    filing_date = datetime.strptime(published[:10], '%Y-%m-%d').date()
                except:
                    filing_date = datetime.now().date()
                
                return {
                    'cik': cik,
                    'company_name': company_name,
                    'form_type': form_type,
                    'filing_date': filing_date,
                    'accession_number': accession_number,
                    'filing_url': link,
                    'description': summary[:500] if summary else None
                }
            
            return None
            
        except Exception as e:
            logger.debug(f"Failed to extract filing info: {e}")
            return None
    
    def save_new_filing(self, filing: Dict, ticker_id: int) -> Optional[int]:
        """Save new filing to database and return filing ID"""
        try:
            with self.conn.cursor() as cursor:
                # Check if filing already exists
                check_query = """
                    SELECT id FROM sec_filings 
                    WHERE accession_number = %s
                """
                cursor.execute(check_query, (filing['accession_number'],))
                
                existing = cursor.fetchone()
                if existing:
                    self.stats['duplicate_filings'] += 1
                    return None
                
                # Get ticker symbol for logging
                symbol_query = "SELECT symbol FROM tickers WHERE id = %s"
                cursor.execute(symbol_query, (ticker_id,))
                symbol_result = cursor.fetchone()
                symbol = symbol_result[0] if symbol_result else 'UNKNOWN'
                
                # Insert new filing
                insert_query = """
                    INSERT INTO sec_filings (
                        ticker_id, form_type, filing_date, 
                        accession_number, filing_url, description,
                        created_at, updated_at
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s,
                        CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
                    )
                    RETURNING id
                """
                
                cursor.execute(insert_query, (
                    ticker_id,
                    filing['form_type'],
                    filing['filing_date'],
                    filing['accession_number'],
                    filing['filing_url'],
                    filing.get('description')
                ))
                
                filing_id = cursor.fetchone()[0]
                self.conn.commit()
                self.stats['new_filings'] += 1
                logger.info(f"Added new filing: {symbol} {filing['form_type']} from {filing['filing_date']}")
                return filing_id
                
        except Exception as e:
            logger.error(f"Failed to save filing: {e}")
            self.conn.rollback()
            return None
    
    def process_rss_updates(self):
        """Process RSS feeds for new filings"""
        # Get existing accession numbers for deduplication
        existing_accessions = self.get_existing_accession_numbers()
        logger.info(f"Found {len(existing_accessions)} existing filings in database")
        
        # Get tracked companies
        tracked_ciks = self.get_tracked_ciks()
        logger.info(f"Tracking {len(tracked_ciks)} companies")
        
        # Track new filings that need to be downloaded
        new_filing_ids = []
        
        # Process each RSS feed
        for feed_name, feed_url in self.rss_feeds.items():
            logger.info(f"Checking {feed_name} RSS feed...")
            self.stats['feeds_checked'] += 1
            
            filings = self.parse_rss_feed(feed_url)
            
            for filing in filings:
                # Skip if we already have this filing
                if filing['accession_number'] in existing_accessions:
                    self.stats['duplicate_filings'] += 1
                    continue
                
                # Check if we're tracking this company
                cik = filing['cik']
                if cik in tracked_ciks:
                    ticker_id = tracked_ciks[cik]
                    filing_id = self.save_new_filing(filing, ticker_id)
                    if filing_id:
                        existing_accessions.add(filing['accession_number'])
                        new_filing_ids.append(filing_id)
                        
                        # Update company count
                        if ticker_id not in self.stats.get('companies_with_new_filings', set()):
                            if 'companies_with_new_filings' not in self.stats:
                                self.stats['companies_with_new_filings'] = set()
                            self.stats['companies_with_new_filings'].add(ticker_id)
                else:
                    # New company not in our database yet
                    logger.debug(f"Skipping filing for untracked CIK: {cik}")
        
        # Update company count stat
        self.stats['companies_updated'] = len(self.stats.get('companies_with_new_filings', set()))
        
        # Trigger download for new filings
        if new_filing_ids:
            logger.info(f"Triggering download for {len(new_filing_ids)} new filings...")
            self.download_new_filings(new_filing_ids)
    
    def download_new_filings(self, filing_ids: List[int]):
        """Trigger download process for new filings"""
        try:
            # Import the downloader
            from sec_filing_downloader import SECFilingDownloader
            
            # Create downloader instance
            downloader = SECFilingDownloader()
            if not downloader.connect_db():
                logger.error("Downloader failed to connect to database")
                return
            
            # Get filing details
            with self.conn.cursor(cursor_factory=RealDictCursor) as cursor:
                query = """
                    SELECT 
                        f.id,
                        f.ticker_id,
                        f.form_type,
                        f.filing_date,
                        f.accession_number,
                        f.filing_url,
                        t.symbol,
                        t.cik
                    FROM sec_filings f
                    JOIN tickers t ON f.ticker_id = t.id
                    WHERE f.id = ANY(%s)
                    AND f.s3_key IS NULL
                """
                cursor.execute(query, (filing_ids,))
                filings = cursor.fetchall()
            
            # Download each filing
            for filing in filings:
                success = downloader.download_and_upload_filing(filing)
                if success:
                    logger.info(f"Downloaded {filing['symbol']} {filing['form_type']}")
                else:
                    logger.error(f"Failed to download {filing['symbol']} {filing['form_type']}")
            
            downloader.conn.close()
            
        except ImportError:
            logger.warning("Downloader module not available, skipping download")
        except Exception as e:
            logger.error(f"Failed to trigger downloads: {e}")
    
    def run_continuous(self, interval_minutes: int = 10):
        """Run continuously, checking for updates every interval"""
        logger.info(f"Starting continuous RSS updater (interval: {interval_minutes} minutes)")

        while True:
            try:
                start_time = datetime.now()

                # Reconnect to database at the start of each cycle to avoid connection timeouts
                if self.conn:
                    try:
                        self.conn.close()
                    except:
                        pass

                if not self.connect_db():
                    logger.error("Failed to connect to database, retrying in 1 minute...")
                    time.sleep(60)
                    continue

                # Reset stats for this run
                self.stats = {
                    'feeds_checked': 0,
                    'new_filings': 0,
                    'duplicate_filings': 0,
                    'companies_updated': 0,
                    'errors': 0
                }

                # Process RSS updates
                self.process_rss_updates()
                
                # Log statistics
                duration = (datetime.now() - start_time).total_seconds()
                logger.info(f"Update cycle completed in {duration:.1f} seconds")
                logger.info(f"Stats: {self.stats}")
                
                # Wait for next cycle
                logger.info(f"Waiting {interval_minutes} minutes until next update...")
                time.sleep(interval_minutes * 60)
                
            except KeyboardInterrupt:
                logger.info("Stopping RSS updater...")
                break
            except Exception as e:
                logger.error(f"Error in update cycle: {e}")
                time.sleep(60)  # Wait 1 minute before retrying
    
    def run_once(self):
        """Run once and exit"""
        start_time = datetime.now()
        logger.info("=== SEC RSS Updater Started (Single Run) ===")
        
        try:
            # Connect to database
            if not self.connect_db():
                logger.error("Failed to connect to database")
                sys.exit(1)
            
            # Process RSS updates
            self.process_rss_updates()
            
            # Log statistics
            duration = (datetime.now() - start_time).total_seconds()
            logger.info(f"Processing completed in {duration:.1f} seconds")
            logger.info(f"Final Statistics:")
            logger.info(f"  Feeds checked: {self.stats['feeds_checked']}")
            logger.info(f"  New filings: {self.stats['new_filings']}")
            logger.info(f"  Duplicate filings: {self.stats['duplicate_filings']}")
            logger.info(f"  Errors: {self.stats['errors']}")
            
            logger.info("=== SEC RSS Updater Completed ===")
            
        except Exception as e:
            logger.error(f"RSS updater failed: {e}")
            sys.exit(1)
        finally:
            if self.conn:
                self.conn.close()


if __name__ == "__main__":
    import argparse
    
    parser = argparse.ArgumentParser(description='SEC RSS Feed Updater')
    parser.add_argument('--continuous', action='store_true', 
                       help='Run continuously with updates every interval')
    parser.add_argument('--interval', type=int, default=10,
                       help='Update interval in minutes (default: 10)')
    
    args = parser.parse_args()
    
    updater = SECRSSUpdater()
    
    if args.continuous:
        # Connect to database
        if not updater.connect_db():
            logger.error("Failed to connect to database")
            sys.exit(1)
        updater.run_continuous(interval_minutes=args.interval)
    else:
        updater.run_once()