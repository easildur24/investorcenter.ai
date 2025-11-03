#!/usr/bin/env python3
"""
SEC Filing Document Downloader and S3 Uploader

Downloads actual SEC filing documents (HTML/PDF) from EDGAR and stores them in S3.
Uses the metadata already collected in the database.
"""

import os
import sys
import time
import logging
import hashlib
import psycopg2
import requests
import boto3
from datetime import datetime
from typing import Dict, List, Optional, Tuple
from psycopg2.extras import RealDictCursor
from botocore.exceptions import ClientError

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class SECFilingDownloader:
    """Downloads SEC filing documents and uploads to S3"""
    
    def __init__(self):
        self.db_config = {
            'host': os.environ.get('DB_HOST', 'localhost'),
            'port': int(os.environ.get('DB_PORT', 5432)),
            'database': os.environ.get('DB_NAME', 'investorcenter_db'),
            'user': os.environ.get('DB_USER'),
            'password': os.environ.get('DB_PASSWORD')
        }
        
        # S3 configuration
        self.s3_bucket = os.environ.get('S3_BUCKET', 'investorcenter-sec-filings')
        self.aws_region = os.environ.get('AWS_REGION', 'us-east-1')
        
        # Initialize S3 client
        self.s3_client = boto3.client('s3', region_name=self.aws_region)
        
        # SEC EDGAR base URL
        self.edgar_base_url = "https://www.sec.gov/Archives/edgar/data"
        
        # Headers for SEC requests (required by SEC)
        self.headers = {
            'User-Agent': 'InvestorCenter AI (contact@investorcenter.ai)',
            'Accept-Encoding': 'gzip, deflate'
        }
        
        self.conn = None
        self.stats = {
            'total_filings': 0,
            'downloaded': 0,
            'uploaded': 0,
            'skipped': 0,
            'errors': 0
        }
        
        # Rate limiting: SEC allows 10 requests per second
        self.last_request_time = 0
        self.min_request_interval = 0.1  # 100ms between requests
    
    def connect_db(self) -> bool:
        """Connect to PostgreSQL database"""
        try:
            self.conn = psycopg2.connect(**self.db_config)
            logger.info(f"Connected to database: {self.db_config['database']}")
            return True
        except Exception as e:
            logger.error(f"Database connection failed: {e}")
            return False
    
    def ensure_s3_bucket(self):
        """Ensure S3 bucket exists and is configured properly"""
        try:
            # Check if bucket exists
            self.s3_client.head_bucket(Bucket=self.s3_bucket)
            logger.info(f"S3 bucket {self.s3_bucket} exists")
        except ClientError as e:
            error_code = int(e.response['Error']['Code'])
            if error_code == 404:
                # Bucket doesn't exist, create it
                logger.info(f"Creating S3 bucket: {self.s3_bucket}")
                try:
                    if self.aws_region == 'us-east-1':
                        self.s3_client.create_bucket(Bucket=self.s3_bucket)
                    else:
                        self.s3_client.create_bucket(
                            Bucket=self.s3_bucket,
                            CreateBucketConfiguration={'LocationConstraint': self.aws_region}
                        )
                    logger.info(f"Created S3 bucket: {self.s3_bucket}")
                    
                    # Enable versioning
                    self.s3_client.put_bucket_versioning(
                        Bucket=self.s3_bucket,
                        VersioningConfiguration={'Status': 'Enabled'}
                    )
                    
                    # Set lifecycle policy to move old versions to Glacier
                    lifecycle_policy = {
                        'Rules': [{
                            'ID': 'MoveToGlacier',
                            'Status': 'Enabled',
                            'Filter': {'Prefix': ''},
                            'NoncurrentVersionTransitions': [{
                                'NoncurrentDays': 90,
                                'StorageClass': 'GLACIER'
                            }],
                            'NoncurrentVersionExpiration': {
                                'NoncurrentDays': 365
                            }
                        }]
                    }
                    try:
                        self.s3_client.put_bucket_lifecycle_configuration(
                            Bucket=self.s3_bucket,
                            LifecycleConfiguration=lifecycle_policy
                        )
                        logger.info("Set lifecycle policy for bucket")
                    except Exception as policy_error:
                        logger.warning(f"Could not set lifecycle policy: {policy_error}")
                        # Continue even if lifecycle policy fails
                    
                except Exception as create_error:
                    logger.error(f"Failed to create bucket: {create_error}")
                    raise
            else:
                logger.error(f"Error checking bucket: {e}")
                raise
    
    def get_filings_to_download(self, limit: Optional[int] = None) -> List[Dict]:
        """Get filings that haven't been downloaded yet"""
        try:
            with self.conn.cursor(cursor_factory=RealDictCursor) as cursor:
                # Add s3_key column if it doesn't exist
                cursor.execute("""
                    DO $$
                    BEGIN
                        IF NOT EXISTS (
                            SELECT 1 FROM information_schema.columns
                            WHERE table_name = 'sec_filings'
                            AND column_name = 's3_key'
                        ) THEN
                            ALTER TABLE sec_filings
                            ADD COLUMN s3_key VARCHAR(500),
                            ADD COLUMN document_url VARCHAR(500),
                            ADD COLUMN download_date TIMESTAMP WITH TIME ZONE,
                            ADD COLUMN file_size_bytes BIGINT;
                        END IF;

                        IF NOT EXISTS (
                            SELECT 1 FROM information_schema.columns
                            WHERE table_name = 'sec_filings'
                            AND column_name = 'json_s3_key'
                        ) THEN
                            ALTER TABLE sec_filings
                            ADD COLUMN json_s3_key VARCHAR(500),
                            ADD COLUMN json_download_date TIMESTAMP WITH TIME ZONE;
                        END IF;
                    END $$;
                """)
                self.conn.commit()
                
                # Get filings that haven't been downloaded
                query = """
                    SELECT 
                        id,
                        symbol,
                        cik,
                        filing_type,
                        filing_date,
                        accession_number,
                        primary_document
                    FROM sec_filings
                    WHERE 
                        s3_key IS NULL
                        AND primary_document IS NOT NULL
                        AND accession_number IS NOT NULL
                    ORDER BY filing_date DESC
                """
                
                if limit:
                    query += f" LIMIT {limit}"
                
                cursor.execute(query)
                filings = cursor.fetchall()
                
                logger.info(f"Found {len(filings)} filings to download")
                return filings
                
        except Exception as e:
            logger.error(f"Failed to get filings: {e}")
            return []
    
    def construct_edgar_url(self, cik: str, accession_number: str, primary_document: str) -> str:
        """Construct the full EDGAR URL for a filing document"""
        # Remove hyphens from accession number for directory structure
        accession_dir = accession_number.replace('-', '')
        
        # Pad CIK with zeros to 10 digits
        cik_padded = cik.zfill(10)
        
        # Construct URL
        # Format: https://www.sec.gov/Archives/edgar/data/{CIK}/{accession_no_hyphens}/{primary_document}
        url = f"{self.edgar_base_url}/{cik_padded}/{accession_dir}/{primary_document}"
        
        return url
    
    def get_pdf_url(self, cik: str, accession_number: str) -> Optional[str]:
        """Construct URL for PDF version of the filing (if it exists)"""
        # Remove hyphens from accession number
        accession_clean = accession_number.replace('-', '')
        cik_padded = cik.zfill(10)

        # PDF files typically follow this pattern
        pdf_filename = f"{accession_clean}.pdf"
        pdf_url = f"{self.edgar_base_url}/{cik_padded}/{accession_clean}/{pdf_filename}"

        return pdf_url

    def get_company_facts_json(self, cik: str) -> Optional[bytes]:
        """Fetch Company Facts JSON data from SEC API"""
        try:
            cik_padded = cik.zfill(10)
            url = f"https://data.sec.gov/api/xbrl/companyfacts/CIK{cik_padded}.json"

            self.rate_limit()
            response = requests.get(url, headers=self.headers, timeout=30)
            response.raise_for_status()

            return response.content

        except requests.exceptions.RequestException as e:
            logger.debug(f"Company Facts JSON not available for CIK {cik}: {e}")
            return None
    
    def construct_s3_key(self, symbol: str, filing_type: str, filing_date: str, 
                        accession_number: str, file_extension: str = 'html') -> str:
        """Construct S3 key for storing the filing"""
        # Structure: filings/{symbol}/{filing_type}/{year}/{filing_date}_{accession_number}.{extension}
        year = filing_date[:4] if filing_date else 'unknown'
        
        # Clean up filing type (remove special characters)
        clean_type = filing_type.replace('/', '-').replace(' ', '_')
        
        # Create S3 key
        s3_key = f"filings/{symbol}/{clean_type}/{year}/{filing_date}_{accession_number}.{file_extension}"
        
        return s3_key
    
    def rate_limit(self):
        """Ensure we don't exceed SEC's rate limit"""
        current_time = time.time()
        time_since_last_request = current_time - self.last_request_time
        
        if time_since_last_request < self.min_request_interval:
            sleep_time = self.min_request_interval - time_since_last_request
            time.sleep(sleep_time)
        
        self.last_request_time = time.time()
    
    def download_filing(self, url: str) -> Optional[bytes]:
        """Download filing document from SEC EDGAR"""
        try:
            self.rate_limit()
            
            response = requests.get(url, headers=self.headers, timeout=30)
            response.raise_for_status()
            
            return response.content
            
        except requests.exceptions.RequestException as e:
            logger.error(f"Failed to download filing from {url}: {e}")
            return None
    
    def upload_to_s3(self, content: bytes, s3_key: str, metadata: Dict) -> bool:
        """Upload filing document to S3"""
        try:
            # Calculate content hash
            content_hash = hashlib.md5(content).hexdigest()
            
            # Prepare S3 metadata (all values must be strings)
            s3_metadata = {
                'symbol': metadata['symbol'],
                'filing_type': metadata['filing_type'],
                'filing_date': str(metadata['filing_date']),
                'accession_number': metadata['accession_number'],
                'cik': metadata['cik'],
                'content_hash': content_hash
            }
            
            # Determine content type based on file extension
            if s3_key.endswith('.pdf'):
                content_type = 'application/pdf'
            elif s3_key.endswith('.xml'):
                content_type = 'application/xml'
            else:
                content_type = 'text/html'
            
            # Upload to S3
            self.s3_client.put_object(
                Bucket=self.s3_bucket,
                Key=s3_key,
                Body=content,
                ContentType=content_type,
                Metadata=s3_metadata,
                StorageClass='STANDARD_IA'  # Infrequent Access for cost savings
            )
            
            logger.debug(f"Uploaded to S3: {s3_key}")
            return True
            
        except Exception as e:
            logger.error(f"Failed to upload to S3: {e}")
            return False
    
    def update_filing_record(self, filing_id: int, s3_key: str, document_url: str,
                           file_size: int, json_s3_key: str = None) -> bool:
        """Update database record with S3 location"""
        try:
            with self.conn.cursor() as cursor:
                if json_s3_key:
                    cursor.execute("""
                        UPDATE sec_filings
                        SET
                            s3_key = %s,
                            document_url = %s,
                            download_date = CURRENT_TIMESTAMP,
                            file_size_bytes = %s,
                            json_s3_key = %s,
                            json_download_date = CURRENT_TIMESTAMP
                        WHERE id = %s
                    """, (s3_key, document_url, file_size, json_s3_key, filing_id))
                else:
                    cursor.execute("""
                        UPDATE sec_filings
                        SET
                            s3_key = %s,
                            document_url = %s,
                            download_date = CURRENT_TIMESTAMP,
                            file_size_bytes = %s
                        WHERE id = %s
                    """, (s3_key, document_url, file_size, filing_id))

                self.conn.commit()
                return True

        except Exception as e:
            logger.error(f"Failed to update filing record: {e}")
            self.conn.rollback()
            return False
    
    def process_filing(self, filing: Dict) -> bool:
        """Process a single filing: download and upload to S3 (both HTML and PDF if available)"""
        symbol = filing['symbol']
        filing_type = filing['filing_type']
        files_uploaded = []
        
        try:
            logger.info(f"Processing {symbol} {filing_type} from {filing['filing_date']}")
            
            # 1. Download HTML version (primary document)
            html_url = self.construct_edgar_url(
                filing['cik'],
                filing['accession_number'],
                filing['primary_document']
            )
            
            html_content = self.download_filing(html_url)
            if html_content:
                self.stats['downloaded'] += 1
                
                # Determine file extension from primary_document
                file_ext = 'html' if filing['primary_document'].endswith('.htm') else 'html'
                
                # Construct S3 key for HTML
                html_s3_key = self.construct_s3_key(
                    symbol,
                    filing_type,
                    str(filing['filing_date']),
                    filing['accession_number'],
                    file_ext
                )
                
                # Upload HTML to S3
                if self.upload_to_s3(html_content, html_s3_key, filing):
                    self.stats['uploaded'] += 1
                    files_uploaded.append(('html', html_s3_key, len(html_content)))
                    logger.info(f"  ✓ Uploaded HTML version ({len(html_content):,} bytes)")
            
            # 2. Try to download PDF version (may not exist)
            pdf_url = self.get_pdf_url(
                filing['cik'],
                filing['accession_number']
            )
            
            # Try to download PDF (it may not exist)
            try:
                self.rate_limit()
                pdf_response = requests.head(pdf_url, headers=self.headers, timeout=5)
                
                if pdf_response.status_code == 200:
                    # PDF exists, download it
                    pdf_content = self.download_filing(pdf_url)
                    if pdf_content and pdf_content[:4] == b'%PDF':  # Verify it's a PDF
                        self.stats['downloaded'] += 1
                        
                        # Construct S3 key for PDF
                        pdf_s3_key = self.construct_s3_key(
                            symbol,
                            filing_type,
                            str(filing['filing_date']),
                            filing['accession_number'],
                            'pdf'
                        )
                        
                        # Upload PDF to S3
                        if self.upload_to_s3(pdf_content, pdf_s3_key, filing):
                            self.stats['uploaded'] += 1
                            files_uploaded.append(('pdf', pdf_s3_key, len(pdf_content)))
                            logger.info(f"  ✓ Uploaded PDF version ({len(pdf_content):,} bytes)")
                else:
                    logger.debug(f"  No PDF version available (status: {pdf_response.status_code})")
            except Exception as pdf_error:
                logger.debug(f"  PDF not available: {pdf_error}")

            # 3. Download Company Facts JSON (one per company, not per filing)
            # Store with the company symbol for easy retrieval
            try:
                json_content = self.get_company_facts_json(filing['cik'])
                if json_content:
                    self.stats['downloaded'] += 1

                    # Construct S3 key for JSON (company-level, not filing-specific)
                    json_s3_key = f"company-facts/{symbol}/companyfacts_{filing['cik']}.json"

                    # Upload JSON to S3
                    if self.upload_to_s3(json_content, json_s3_key, filing):
                        self.stats['uploaded'] += 1
                        files_uploaded.append(('json', json_s3_key, len(json_content)))
                        logger.info(f"  ✓ Uploaded Company Facts JSON ({len(json_content):,} bytes)")
            except Exception as json_error:
                logger.debug(f"  Company Facts JSON not available: {json_error}")

            # Update database record with primary file info
            if files_uploaded:
                # Use the HTML version as primary
                primary_file = next((f for f in files_uploaded if f[0] == 'html'), files_uploaded[0])
                # Get JSON S3 key if available
                json_file = next((f for f in files_uploaded if f[0] == 'json'), None)
                json_s3_key = json_file[1] if json_file else None

                self.update_filing_record(
                    filing['id'],
                    primary_file[1],  # s3_key
                    html_url,
                    primary_file[2],  # file_size
                    json_s3_key       # json_s3_key
                )
                
                # If we have both HTML and PDF, note it in the log
                if len(files_uploaded) > 1:
                    logger.info(f"  Successfully stored {len(files_uploaded)} versions for {symbol} {filing_type}")
                
                return True
            else:
                self.stats['errors'] += 1
                return False
                
        except Exception as e:
            logger.error(f"Error processing {symbol} {filing_type}: {e}")
            self.stats['errors'] += 1
            return False
    
    def process_batch(self, filings: List[Dict]):
        """Process a batch of filings"""
        for i, filing in enumerate(filings, 1):
            # Progress logging
            if i % 100 == 0:
                logger.info(f"Progress: {i}/{len(filings)} filings processed")
                logger.info(f"Stats: Downloaded={self.stats['downloaded']}, "
                          f"Uploaded={self.stats['uploaded']}, "
                          f"Errors={self.stats['errors']}")
            
            self.process_filing(filing)
            
            # Brief pause every 1000 filings
            if i % 1000 == 0:
                logger.info("Brief pause after 1000 filings...")
                time.sleep(5)
    
    def run(self, limit: Optional[int] = None):
        """Main execution function"""
        start_time = datetime.now()
        logger.info("=== SEC Filing Document Downloader Started ===")
        
        try:
            # Connect to database
            if not self.connect_db():
                logger.error("Failed to connect to database")
                sys.exit(1)
            
            # Ensure S3 bucket exists
            self.ensure_s3_bucket()
            
            # Get filings to download
            filings = self.get_filings_to_download(limit)
            self.stats['total_filings'] = len(filings)
            
            if not filings:
                logger.info("No filings to download")
                return
            
            logger.info(f"Processing {len(filings)} filings...")
            
            # Process all filings
            self.process_batch(filings)
            
            # Log statistics
            duration = datetime.now() - start_time
            logger.info(f"Processing completed in {duration.total_seconds():.1f} seconds")
            logger.info(f"Final Statistics:")
            logger.info(f"  Total filings: {self.stats['total_filings']}")
            logger.info(f"  Downloaded: {self.stats['downloaded']}")
            logger.info(f"  Uploaded to S3: {self.stats['uploaded']}")
            logger.info(f"  Skipped: {self.stats['skipped']}")
            logger.info(f"  Errors: {self.stats['errors']}")
            
            if self.stats['downloaded'] > 0:
                success_rate = (self.stats['uploaded'] / self.stats['downloaded']) * 100
                logger.info(f"  Success rate: {success_rate:.1f}%")
            
            logger.info("=== SEC Filing Document Downloader Completed ===")
            
        except Exception as e:
            logger.error(f"Filing downloader failed: {e}")
            sys.exit(1)
        finally:
            if self.conn:
                self.conn.close()


if __name__ == "__main__":
    # Allow limit to be passed as argument
    limit = int(sys.argv[1]) if len(sys.argv) > 1 else None
    
    downloader = SECFilingDownloader()
    downloader.run(limit=limit)