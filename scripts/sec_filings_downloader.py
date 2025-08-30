#!/usr/bin/env python3
"""
SEC EDGAR Filings Downloader for HIMS & Hers Health, Inc.

This script downloads SEC filings for HIMS using the SEC's EDGAR Full-Text Search API.
It can download various filing types (10-K, 10-Q, 8-K, etc.) and save them locally.

Usage:
    python sec_filings_downloader.py --ticker HIMS --form-type 10-K --count 10
    python sec_filings_downloader.py --ticker HIMS --all-forms --count 50
    python sec_filings_downloader.py --ticker HIMS --download-documents
"""

import argparse
import json
import os
import time
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional
from urllib.parse import urljoin

import pandas as pd
import requests


class SECFilingsDownloader:
    """Downloads SEC filings using the EDGAR API."""
    
    BASE_URL = "https://www.sec.gov"
    DATA_URL = "https://data.sec.gov"
    
    # Known CIKs for common tickers
    TICKER_TO_CIK = {
        "HIMS": "0001773751",  # Hims & Hers Health, Inc.
    }
    
    def __init__(self, user_agent: str = "InvestorCenter.ai contact@investorcenter.ai"):
        """Initialize the downloader with proper headers."""
        self.headers = {
            "User-Agent": user_agent,
            "Accept-Encoding": "gzip, deflate"
        }
        self.session = requests.Session()
        self.session.headers.update(self.headers)
    
    def get_cik_from_ticker(self, ticker: str) -> Optional[str]:
        """Get CIK from ticker symbol."""
        # First check our known mappings
        if ticker.upper() in self.TICKER_TO_CIK:
            return self.TICKER_TO_CIK[ticker.upper()]
        
        # If not found, try to search for it
        try:
            search_url = f"{self.BASE_URL}/cgi-bin/browse-edgar"
            params = {
                "action": "getcompany",
                "CIK": ticker,
                "type": "10-k",
                "dateb": "",
                "output": "atom"
            }
            
            response = self.session.get(search_url, params=params, timeout=10)
            if response.status_code == 200:
                # Parse the response to extract CIK
                # This is a simplified approach - in production you might want more robust parsing
                content = response.text
                if "CIK=" in content:
                    cik_start = content.find("CIK=") + 4
                    cik_end = content.find("&", cik_start)
                    if cik_end == -1:
                        cik_end = content.find(" ", cik_start)
                    cik = content[cik_start:cik_end].strip()
                    return cik.zfill(10)  # Pad with zeros to 10 digits
        except Exception as e:
            print(f"Warning: Could not automatically find CIK for {ticker}: {e}")
        
        return None
    
    def search_filings(self, 
                      cik: str, 
                      form_type: Optional[str] = None,
                      count: int = 100,
                      start: int = 0) -> Dict:
        """Search for filings using the EDGAR API."""
        # Format CIK properly (pad with zeros to 10 digits)
        formatted_cik = cik.zfill(10)
        
        # Use the SEC data API endpoint
        url = f"{self.DATA_URL}/submissions/CIK{formatted_cik}.json"
        
        try:
            response = self.session.get(url, timeout=30)
            response.raise_for_status()
            
            # Rate limiting - be respectful to SEC servers
            time.sleep(0.1)
            
            data = response.json()
            
            # Extract recent filings
            recent_filings = data.get("filings", {}).get("recent", {})
            
            if not recent_filings:
                return {"filings": []}
            
            # Convert to the expected format
            filings = []
            forms = recent_filings.get("form", [])
            filing_dates = recent_filings.get("filingDate", [])
            accession_numbers = recent_filings.get("accessionNumber", [])
            primary_documents = recent_filings.get("primaryDocument", [])
            primary_doc_descriptions = recent_filings.get("primaryDocDescription", [])
            sizes = recent_filings.get("size", [])
            is_xbrl = recent_filings.get("isXBRL", [])
            is_inline_xbrl = recent_filings.get("isInlineXBRL", [])
            
            # Get company info
            company_name = data.get("name", "")
            
            for i in range(len(forms)):
                if form_type and forms[i] != form_type:
                    continue
                    
                filing = {
                    "formType": forms[i] if i < len(forms) else "",
                    "filingDate": filing_dates[i] if i < len(filing_dates) else "",
                    "accessionNumber": accession_numbers[i] if i < len(accession_numbers) else "",
                    "primaryDocument": primary_documents[i] if i < len(primary_documents) else "",
                    "primaryDocDescription": primary_doc_descriptions[i] if i < len(primary_doc_descriptions) else "",
                    "companyName": company_name,
                    "cik": formatted_cik,
                    "size": sizes[i] if i < len(sizes) else 0,
                    "isXBRL": is_xbrl[i] if i < len(is_xbrl) else False,
                    "isInlineXBRL": is_inline_xbrl[i] if i < len(is_inline_xbrl) else False
                }
                filings.append(filing)
                
                # Apply count limit
                if len(filings) >= count:
                    break
            
            # Apply start offset
            filings = filings[start:start + count]
            
            return {"filings": filings}
            
        except requests.exceptions.RequestException as e:
            print(f"Error fetching filings: {e}")
            return {"filings": []}
    
    def download_filing_document(self, cik: str, accession_number: str, primary_document: str, 
                                output_dir: Path) -> bool:
        """Download the actual filing document."""
        try:
            # Format CIK properly (pad with zeros to 10 digits)
            formatted_cik = cik.zfill(10)
            
            # Remove dashes from accession number for directory structure
            accession_clean = accession_number.replace("-", "")
            
            # Construct the document URL using the correct SEC format
            doc_url = f"{self.BASE_URL}/Archives/edgar/data/{formatted_cik}/{accession_clean}/{primary_document}"
            
            response = self.session.get(doc_url, timeout=30)
            response.raise_for_status()
            
            # Create filename
            filename = f"{accession_number}_{primary_document}"
            filepath = output_dir / filename
            
            # Save the document
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(response.text)
            
            print(f"Downloaded: {filename}")
            
            # Rate limiting
            time.sleep(0.1)
            
            return True
        except Exception as e:
            print(f"Error downloading {primary_document}: {e}")
            return False
    
    def get_filings_data(self, 
                        ticker: str,
                        form_type: Optional[str] = None,
                        count: int = 100) -> List[Dict]:
        """Get filings data for a ticker."""
        # Get CIK
        cik = self.get_cik_from_ticker(ticker)
        if not cik:
            raise ValueError(f"Could not find CIK for ticker {ticker}")
        
        print(f"Using CIK {cik} for ticker {ticker}")
        
        all_filings = []
        fetched = 0
        start = 0
        
        while fetched < count:
            batch_size = min(100, count - fetched)
            
            print(f"Fetching filings {start + 1}-{start + batch_size}...")
            
            data = self.search_filings(cik, form_type, batch_size, start)
            
            if not data or "filings" not in data:
                break
            
            filings = data["filings"]
            if not filings:
                break
            
            all_filings.extend(filings)
            fetched += len(filings)
            start += len(filings)
            
            # If we got fewer than requested, we've reached the end
            if len(filings) < batch_size:
                break
        
        return all_filings[:count]  # Ensure we don't exceed the requested count
    
    def save_filings_metadata(self, filings: List[Dict], output_file: Path):
        """Save filings metadata to CSV."""
        if not filings:
            print("No filings to save.")
            return
        
        # Extract relevant fields
        filings_data = []
        for filing in filings:
            filing_info = {
                "form_type": filing.get("formType"),
                "filing_date": filing.get("filingDate"),
                "accession_number": filing.get("accessionNumber"),
                "primary_document": filing.get("primaryDocument"),
                "primary_doc_description": filing.get("primaryDocDescription"),
                "company_name": filing.get("companyName"),
                "cik": filing.get("cik"),
                "size": filing.get("size"),
                "is_xbrl": filing.get("isXBRL"),
                "is_inline_xbrl": filing.get("isInlineXBRL")
            }
            filings_data.append(filing_info)
        
        # Create DataFrame and save
        df = pd.DataFrame(filings_data)
        df.to_csv(output_file, index=False)
        print(f"Saved metadata for {len(filings_data)} filings to {output_file}")
    
    def download_filings(self, 
                        ticker: str,
                        form_type: Optional[str] = None,
                        count: int = 10,
                        download_documents: bool = False,
                        output_dir: str = "sec_filings"):
        """Main method to download filings for a ticker."""
        # Create output directory
        output_path = Path(output_dir)
        output_path.mkdir(exist_ok=True)
        
        # Create subdirectory for this ticker
        ticker_dir = output_path / ticker.upper()
        ticker_dir.mkdir(exist_ok=True)
        
        print(f"Downloading SEC filings for {ticker.upper()}...")
        if form_type:
            print(f"Form type: {form_type}")
        print(f"Count: {count}")
        print(f"Output directory: {ticker_dir}")
        
        try:
            # Get filings data
            filings = self.get_filings_data(ticker, form_type, count)
            
            if not filings:
                print("No filings found.")
                return
            
            # Save metadata
            metadata_file = ticker_dir / f"{ticker.upper()}_filings_metadata.csv"
            self.save_filings_metadata(filings, metadata_file)
            
            # Save raw JSON data
            json_file = ticker_dir / f"{ticker.upper()}_filings_raw.json"
            with open(json_file, 'w') as f:
                json.dump(filings, f, indent=2)
            print(f"Saved raw data to {json_file}")
            
            # Download actual documents if requested
            if download_documents:
                print("\nDownloading filing documents...")
                docs_dir = ticker_dir / "documents"
                docs_dir.mkdir(exist_ok=True)
                
                # Get CIK for document downloads
                cik = self.get_cik_from_ticker(ticker)
                
                downloaded = 0
                for filing in filings:
                    accession_number = filing.get("accessionNumber")
                    primary_document = filing.get("primaryDocument")
                    
                    if accession_number and primary_document and cik:
                        success = self.download_filing_document(
                            cik, accession_number, primary_document, docs_dir
                        )
                        if success:
                            downloaded += 1
                
                print(f"Successfully downloaded {downloaded} documents to {docs_dir}")
            
            print(f"\nCompleted! Found {len(filings)} filings for {ticker.upper()}")
            
        except Exception as e:
            print(f"Error downloading filings: {e}")


def main():
    """Main CLI function."""
    parser = argparse.ArgumentParser(
        description="Download SEC filings using EDGAR API",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s --ticker HIMS --form-type 10-K --count 5
  %(prog)s --ticker HIMS --all-forms --count 20
  %(prog)s --ticker HIMS --download-documents --count 3
        """
    )
    
    parser.add_argument(
        "--ticker", 
        required=True,
        help="Stock ticker symbol (e.g., HIMS)"
    )
    
    parser.add_argument(
        "--form-type",
        help="Specific form type to download (e.g., 10-K, 10-Q, 8-K)"
    )
    
    parser.add_argument(
        "--all-forms",
        action="store_true",
        help="Download all form types (ignores --form-type)"
    )
    
    parser.add_argument(
        "--count",
        type=int,
        default=10,
        help="Number of filings to download (default: 10)"
    )
    
    parser.add_argument(
        "--download-documents",
        action="store_true",
        help="Download the actual filing documents (not just metadata)"
    )
    
    parser.add_argument(
        "--output-dir",
        default="sec_filings",
        help="Output directory for downloaded files (default: sec_filings)"
    )
    
    parser.add_argument(
        "--user-agent",
        default="InvestorCenter.ai (contact@investorcenter.ai)",
        help="User-Agent string for requests"
    )
    
    args = parser.parse_args()
    
    # Determine form type
    form_type = None if args.all_forms else args.form_type
    
    # Create downloader and run
    downloader = SECFilingsDownloader(user_agent=args.user_agent)
    
    downloader.download_filings(
        ticker=args.ticker,
        form_type=form_type,
        count=args.count,
        download_documents=args.download_documents,
        output_dir=args.output_dir
    )


if __name__ == "__main__":
    main()
