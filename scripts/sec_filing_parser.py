#!/usr/bin/env python3
"""
SEC Filing Parser - Extracts structured data from 10-K and 10-Q filings
"""

import os
import re
import sys
import json
import logging
import psycopg2
import boto3
from datetime import datetime
from typing import Dict, List, Optional, Tuple
from bs4 import BeautifulSoup
from psycopg2.extras import RealDictCursor

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class SECFilingParser:
    """Parses SEC filing documents to extract structured data"""
    
    def __init__(self):
        self.s3_client = boto3.client('s3')
        self.s3_bucket = os.environ.get('S3_BUCKET', 'investorcenter-sec-filings')
        
        self.db_config = {
            'host': os.environ.get('DB_HOST', 'localhost'),
            'port': int(os.environ.get('DB_PORT', 5432)),
            'database': os.environ.get('DB_NAME', 'investorcenter_db'),
            'user': os.environ.get('DB_USER'),
            'password': os.environ.get('DB_PASSWORD')
        }
        
        self.conn = None
        self.stats = {
            'total_filings': 0,
            'parsed': 0,
            'errors': 0,
            'skipped': 0
        }
        
        # Section patterns for 10-K
        self.sections_10k = {
            'business': r'(?:ITEM\s+1[.\s]+BUSINESS|BUSINESS\s+OVERVIEW)',
            'risk_factors': r'(?:ITEM\s+1A[.\s]+RISK\s+FACTORS)',
            'properties': r'(?:ITEM\s+2[.\s]+PROPERTIES)',
            'legal_proceedings': r'(?:ITEM\s+3[.\s]+LEGAL\s+PROCEEDINGS)',
            'market_price': r'(?:ITEM\s+5[.\s]+MARKET)',
            'mda': r'(?:ITEM\s+7[.\s]+MANAGEMENT[\'"]?S?\s+DISCUSSION)',
            'financial_statements': r'(?:ITEM\s+8[.\s]+FINANCIAL\s+STATEMENTS)',
            'controls': r'(?:ITEM\s+9A[.\s]+CONTROLS)',
            'executive_officers': r'(?:ITEM\s+10[.\s]+DIRECTORS.*EXECUTIVE)',
            'compensation': r'(?:ITEM\s+11[.\s]+EXECUTIVE\s+COMPENSATION)',
            'ownership': r'(?:ITEM\s+12[.\s]+SECURITY\s+OWNERSHIP)'
        }
        
        # Section patterns for 10-Q
        self.sections_10q = {
            'financial_statements': r'(?:PART\s+I.*ITEM\s+1[.\s]+FINANCIAL\s+STATEMENTS)',
            'mda': r'(?:ITEM\s+2[.\s]+MANAGEMENT[\'"]?S?\s+DISCUSSION)',
            'controls': r'(?:ITEM\s+4[.\s]+CONTROLS)',
            'legal_proceedings': r'(?:ITEM\s+1[.\s]+LEGAL\s+PROCEEDINGS)',
            'risk_factors': r'(?:ITEM\s+1A[.\s]+RISK\s+FACTORS)',
            'unregistered_sales': r'(?:ITEM\s+2[.\s]+UNREGISTERED\s+SALES)',
            'defaults': r'(?:ITEM\s+3[.\s]+DEFAULTS)',
            'other_information': r'(?:ITEM\s+5[.\s]+OTHER\s+INFORMATION)'
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
    
    def get_unparsed_filings(self, limit: Optional[int] = None) -> List[Dict]:
        """Get filings that haven't been parsed yet"""
        try:
            with self.conn.cursor(cursor_factory=RealDictCursor) as cursor:
                query = """
                    SELECT 
                        f.id,
                        f.ticker_id,
                        f.filing_type as form_type,
                        f.filing_date,
                        f.s3_key,
                        t.symbol,
                        t.name as company_name
                    FROM sec_filings f
                    JOIN tickers t ON f.ticker_id = t.id
                    WHERE 
                        f.s3_key IS NOT NULL
                        AND f.parsed_at IS NULL
                        AND f.filing_type IN ('10-K', '10-Q')
                    ORDER BY f.filing_date DESC
                """
                
                if limit:
                    query += f" LIMIT {limit}"
                
                cursor.execute(query)
                filings = cursor.fetchall()
                
                logger.info(f"Found {len(filings)} unparsed filings")
                return filings
                
        except Exception as e:
            logger.error(f"Failed to get unparsed filings: {e}")
            return []
    
    def download_from_s3(self, s3_key: str) -> Optional[str]:
        """Download filing content from S3"""
        try:
            response = self.s3_client.get_object(
                Bucket=self.s3_bucket,
                Key=s3_key
            )
            
            content = response['Body'].read().decode('utf-8', errors='ignore')
            return content
            
        except Exception as e:
            logger.error(f"Failed to download {s3_key} from S3: {e}")
            return None
    
    def extract_text_from_html(self, html_content: str) -> str:
        """Extract clean text from HTML content"""
        try:
            soup = BeautifulSoup(html_content, 'html.parser')
            
            # Remove script and style elements
            for script in soup(['script', 'style']):
                script.decompose()
            
            # Get text
            text = soup.get_text(separator=' ', strip=True)
            
            # Clean up whitespace
            text = re.sub(r'\s+', ' ', text)
            
            return text
            
        except Exception as e:
            logger.error(f"Failed to extract text from HTML: {e}")
            return ""
    
    def extract_sections(self, text: str, form_type: str) -> Dict[str, str]:
        """Extract sections from filing text"""
        sections = {}
        
        # Choose section patterns based on form type
        if form_type == '10-K':
            patterns = self.sections_10k
        elif form_type == '10-Q':
            patterns = self.sections_10q
        else:
            return sections
        
        # Convert text to uppercase for matching
        text_upper = text.upper()
        
        # Find all section positions
        section_positions = []
        for section_name, pattern in patterns.items():
            matches = re.finditer(pattern, text_upper, re.IGNORECASE)
            for match in matches:
                section_positions.append((match.start(), section_name))
        
        # Sort by position
        section_positions.sort(key=lambda x: x[0])
        
        # Extract text between sections
        for i, (start_pos, section_name) in enumerate(section_positions):
            # Find end position (start of next section or end of document)
            if i < len(section_positions) - 1:
                end_pos = section_positions[i + 1][0]
            else:
                end_pos = len(text)
            
            # Extract section text (limit to 50000 chars for storage)
            section_text = text[start_pos:end_pos][:50000]
            
            # Clean up the text
            section_text = section_text.strip()
            
            if section_text:
                sections[section_name] = section_text
        
        return sections
    
    def extract_financial_metrics(self, text: str) -> Dict[str, any]:
        """Extract key financial metrics from filing text"""
        metrics = {}
        
        # Revenue patterns (enhanced)
        revenue_patterns = [
            r'(?:total\s+)?(?:net\s+)?revenues?\s+(?:were|of|was|are|is)?\s*\$?\s*([\d,]+(?:\.\d+)?)\s*(?:million|billion|thousand)?',
            r'net\s+(?:sales|revenues?)\s+(?:were|of|was|are|is)?\s*\$?\s*([\d,]+(?:\.\d+)?)\s*(?:million|billion|thousand)?',
            r'(?:total\s+)?(?:net\s+)?revenues?[:\s]+\$?\s*([\d,]+(?:\.\d+)?)\s*(?:million|billion|thousand)?',
            r'revenues?\s+for\s+(?:the\s+)?(?:year|quarter)\s+(?:were|was)?\s*\$?\s*([\d,]+(?:\.\d+)?)\s*(?:million|billion|thousand)?'
        ]
        
        # Net income patterns (enhanced)
        income_patterns = [
            r'net\s+(?:income|earnings?)\s+(?:were|of|was|are|is)?\s*\$?\s*([\d,]+(?:\.\d+)?)\s*(?:million|billion|thousand)?',
            r'net\s+(?:income|earnings?)[:\s]+\$?\s*([\d,]+(?:\.\d+)?)\s*(?:million|billion|thousand)?',
            r'net\s+(?:loss|losses)\s+(?:were|of|was|are|is)?\s*\$?\s*\(?([\d,]+(?:\.\d+)?)\)?\s*(?:million|billion|thousand)?',
            r'(?:net\s+)?(?:income|loss)\s+attributable\s+to\s+(?:common\s+)?(?:stockholders|shareholders)\s*[:\s]+\$?\s*\(?([\d,]+(?:\.\d+)?)\)?\s*(?:million|billion|thousand)?'
        ]
        
        # EPS patterns
        eps_patterns = [
            r'(?:basic\s+)?earnings?\s+per\s+(?:common\s+)?share\s*[:\s]+\$?\s*([\d.]+)',
            r'(?:basic\s+)?eps\s*[:\s]+\$?\s*([\d.]+)',
            r'(?:diluted\s+)?earnings?\s+per\s+(?:common\s+)?share\s*[:\s]+\$?\s*([\d.]+)'
        ]
        
        # Total assets patterns
        assets_patterns = [
            r'total\s+assets\s+(?:were|of|was|are|is)?\s*\$?\s*([\d,]+(?:\.\d+)?)\s*(?:million|billion|thousand)?',
            r'total\s+assets[:\s]+\$?\s*([\d,]+(?:\.\d+)?)\s*(?:million|billion|thousand)?'
        ]
        
        # Extract revenue
        for pattern in revenue_patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                value_str = match.group(1).replace(',', '')
                unit = match.group(2) if len(match.groups()) > 1 else None
                try:
                    value = float(value_str)
                    # Convert to actual value based on unit
                    if unit:
                        if 'billion' in unit.lower():
                            value *= 1000000000
                        elif 'million' in unit.lower():
                            value *= 1000000
                        elif 'thousand' in unit.lower():
                            value *= 1000
                    metrics['revenue'] = value
                    break
                except:
                    pass
        
        # Extract net income
        for pattern in income_patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                value_str = match.group(1).replace(',', '')
                unit = match.group(2) if len(match.groups()) > 1 else None
                try:
                    value = float(value_str)
                    # Convert to actual value based on unit
                    if unit:
                        if 'billion' in unit.lower():
                            value *= 1000000000
                        elif 'million' in unit.lower():
                            value *= 1000000
                        elif 'thousand' in unit.lower():
                            value *= 1000
                    # If it's a loss pattern, make it negative
                    if 'loss' in pattern.lower():
                        value = -value
                    metrics['net_income'] = value
                    break
                except:
                    pass
        
        # Extract EPS
        for pattern in eps_patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                try:
                    metrics['eps'] = float(match.group(1))
                    break
                except:
                    pass
        
        # Extract total assets
        for pattern in assets_patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                value_str = match.group(1).replace(',', '')
                unit = match.group(2) if len(match.groups()) > 1 else None
                try:
                    value = float(value_str)
                    # Convert to actual value based on unit
                    if unit:
                        if 'billion' in unit.lower():
                            value *= 1000000000
                        elif 'million' in unit.lower():
                            value *= 1000000
                        elif 'thousand' in unit.lower():
                            value *= 1000
                    metrics['total_assets'] = value
                    break
                except:
                    pass
        
        return metrics
    
    def save_parsed_data(self, filing_id: int, sections: Dict, metrics: Dict) -> bool:
        """Save parsed data to database"""
        try:
            with self.conn.cursor() as cursor:
                # Create parsed data JSON
                parsed_data = {
                    'sections': sections,
                    'metrics': metrics,
                    'parsed_at': datetime.now().isoformat()
                }
                
                # Update filing record
                update_query = """
                    UPDATE sec_filings
                    SET 
                        parsed_data = %s,
                        parsed_at = CURRENT_TIMESTAMP
                    WHERE id = %s
                """
                
                cursor.execute(update_query, (
                    json.dumps(parsed_data),
                    filing_id
                ))
                
                self.conn.commit()
                return True
                
        except Exception as e:
            logger.error(f"Failed to save parsed data for filing {filing_id}: {e}")
            self.conn.rollback()
            return False
    
    def parse_filing(self, filing: Dict) -> bool:
        """Parse a single filing"""
        filing_id = filing['id']
        symbol = filing['symbol']
        form_type = filing['form_type']
        s3_key = filing['s3_key']
        
        logger.info(f"Parsing {symbol} {form_type} filing from {filing['filing_date']}")
        
        try:
            # Download from S3
            html_content = self.download_from_s3(s3_key)
            if not html_content:
                logger.error(f"Failed to download filing {filing_id}")
                self.stats['errors'] += 1
                return False
            
            # Extract text from HTML
            text = self.extract_text_from_html(html_content)
            if not text:
                logger.warning(f"No text extracted from filing {filing_id}")
                self.stats['skipped'] += 1
                return False
            
            # Extract sections
            sections = self.extract_sections(text, form_type)
            logger.info(f"  Extracted {len(sections)} sections")
            
            # Extract financial metrics
            metrics = self.extract_financial_metrics(text)
            if metrics:
                logger.info(f"  Extracted metrics: {metrics}")
            
            # Save to database
            if self.save_parsed_data(filing_id, sections, metrics):
                self.stats['parsed'] += 1
                logger.info(f"  ✓ Successfully parsed and saved")
                return True
            else:
                self.stats['errors'] += 1
                return False
                
        except Exception as e:
            logger.error(f"Failed to parse filing {filing_id}: {e}")
            self.stats['errors'] += 1
            return False
    
    def process_batch(self, filings: List[Dict]):
        """Process a batch of filings"""
        for i, filing in enumerate(filings, 1):
            # Progress logging
            if i % 10 == 0:
                logger.info(f"Progress: {i}/{len(filings)} filings processed")
                logger.info(f"Stats: {self.stats}")
            
            # Parse filing
            self.parse_filing(filing)
    
    def run(self, limit: Optional[int] = None):
        """Main execution function"""
        start_time = datetime.now()
        logger.info("=== SEC Filing Parser Started ===")
        
        try:
            # Connect to database
            if not self.connect_db():
                logger.error("Failed to connect to database")
                sys.exit(1)
            
            # Get unparsed filings
            filings = self.get_unparsed_filings(limit)
            self.stats['total_filings'] = len(filings)
            
            if not filings:
                logger.info("No unparsed filings found")
                return
            
            logger.info(f"Processing {len(filings)} filings...")
            
            # Process all filings
            self.process_batch(filings)
            
            # Log statistics
            duration = datetime.now() - start_time
            logger.info(f"Processing completed in {duration.total_seconds():.1f} seconds")
            logger.info(f"Final Statistics:")
            logger.info(f"  Total filings: {self.stats['total_filings']}")
            logger.info(f"  Successfully parsed: {self.stats['parsed']}")
            logger.info(f"  Skipped: {self.stats['skipped']}")
            logger.info(f"  Errors: {self.stats['errors']}")
            
            # Calculate success rate
            if self.stats['total_filings'] > 0:
                success_rate = (self.stats['parsed'] / self.stats['total_filings']) * 100
                logger.info(f"  Success rate: {success_rate:.1f}%")
            
            logger.info("=== SEC Filing Parser Completed ===")
            
        except Exception as e:
            logger.error(f"Filing parser failed: {e}")
            sys.exit(1)
        finally:
            if self.conn:
                self.conn.close()


if __name__ == "__main__":
    # Allow limit to be passed as argument
    limit = int(sys.argv[1]) if len(sys.argv) > 1 else None
    
    parser = SECFilingParser()
    parser.run(limit=limit)