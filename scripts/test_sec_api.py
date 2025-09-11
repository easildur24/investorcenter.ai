#!/usr/bin/env python3
"""
Test script to explore SEC EDGAR API capabilities
"""

import requests
import json
import time
from datetime import datetime, timedelta

# SEC EDGAR API Configuration
BASE_URL = "https://data.sec.gov"
SEC_URL = "https://www.sec.gov"
HEADERS = {
    'User-Agent': 'InvestorCenter AI (contact@investorcenter.ai)',
    'Accept-Encoding': 'gzip, deflate'
}

def test_company_tickers():
    """Test fetching company tickers with CIK mapping"""
    print("Testing Company Tickers Endpoint...")
    url = f"{SEC_URL}/files/company_tickers.json"
    
    try:
        response = requests.get(url, headers=HEADERS)
        response.raise_for_status()
        data = response.json()
        
        # Sample first 5 companies
        print(f"Total companies: {len(data)}")
        for i, (idx, company) in enumerate(data.items()):
            if i >= 5:
                break
            print(f"  {company['ticker']}: CIK={company['cik_str']:010d}, {company['title']}")
        
        # Find Apple as example
        for idx, company in data.items():
            if company['ticker'] == 'AAPL':
                print(f"\nApple found: CIK={company['cik_str']:010d}")
                return company['cik_str']
                
    except Exception as e:
        print(f"Error: {e}")
    
    return None

def test_company_submissions(cik):
    """Test fetching company submissions (filings list)"""
    print(f"\nTesting Submissions for CIK {cik:010d}...")
    url = f"{BASE_URL}/submissions/CIK{cik:010d}.json"
    
    try:
        response = requests.get(url, headers=HEADERS)
        response.raise_for_status()
        data = response.json()
        
        print(f"Company: {data['name']}")
        print(f"SIC: {data.get('sicDescription', 'N/A')}")
        print(f"Fiscal Year End: {data.get('fiscalYearEnd', 'N/A')}")
        
        # Recent filings
        recent_filings = data['filings']['recent']
        print(f"\nTotal recent filings: {len(recent_filings['form'])}")
        
        # Show recent 10-K and 10-Q
        print("\nRecent 10-K and 10-Q filings:")
        for i in range(min(50, len(recent_filings['form']))):
            form_type = recent_filings['form'][i]
            if form_type in ['10-K', '10-Q']:
                filing_date = recent_filings['filingDate'][i]
                report_date = recent_filings['reportDate'][i]
                accession = recent_filings['accessionNumber'][i]
                print(f"  {form_type}: Filed={filing_date}, Period={report_date}, Accession={accession}")
        
        return recent_filings
        
    except Exception as e:
        print(f"Error: {e}")
    
    return None

def test_company_facts(cik):
    """Test fetching company facts (XBRL data)"""
    print(f"\nTesting Company Facts for CIK {cik:010d}...")
    url = f"{BASE_URL}/api/xbrl/companyfacts/CIK{cik:010d}.json"
    
    try:
        response = requests.get(url, headers=HEADERS)
        response.raise_for_status()
        data = response.json()
        
        print(f"Entity: {data['entityName']}")
        
        # Sample available facts
        facts = data.get('facts', {})
        
        # US-GAAP facts
        if 'us-gaap' in facts:
            gaap_facts = facts['us-gaap']
            print(f"\nAvailable US-GAAP facts: {len(gaap_facts)}")
            
            # Show some key metrics if available
            key_metrics = [
                'Revenues',
                'NetIncomeLoss', 
                'EarningsPerShareBasic',
                'Assets',
                'Liabilities',
                'StockholdersEquity'
            ]
            
            for metric in key_metrics:
                if metric in gaap_facts:
                    units = gaap_facts[metric].get('units', {})
                    if 'USD' in units:
                        values = units['USD'][-5:] if len(units['USD']) > 5 else units['USD']
                        print(f"\n{metric}:")
                        for value in values:
                            print(f"  {value.get('fy', 'N/A')}-{value.get('fp', 'N/A')}: ${value.get('val', 0):,.0f}")
                            
    except Exception as e:
        print(f"Error: {e}")

def test_recent_filings():
    """Test fetching recent filings from RSS feed"""
    print("\nTesting Recent Filings RSS Feed...")
    
    # RSS feed for recent 10-K and 10-Q filings
    url = "https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=10-K&count=10&output=atom"
    
    try:
        response = requests.get(url)
        response.raise_for_status()
        
        # Parse RSS/Atom feed (simplified - normally use feedparser)
        content = response.text
        
        # Extract entries (simplified parsing)
        import re
        titles = re.findall(r'<title>(.*?)</title>', content)
        links = re.findall(r'<link.*?href="(.*?)"', content)
        
        print("Recent 10-K filings:")
        for i in range(min(5, len(titles)-1)):  # Skip feed title
            print(f"  {titles[i+1]}")
            if i < len(links)-1:
                print(f"    URL: {links[i+1]}")
                
    except Exception as e:
        print(f"Error: {e}")

def test_filing_document(cik, accession_number):
    """Test fetching actual filing document"""
    print(f"\nTesting Filing Document Retrieval...")
    
    # Format accession number (remove dashes)
    accession_clean = accession_number.replace('-', '')
    
    # Construct filing URL
    filing_url = f"https://www.sec.gov/Archives/edgar/data/{cik}/{accession_clean}/{accession_number}.txt"
    
    print(f"Filing URL: {filing_url}")
    
    try:
        response = requests.get(filing_url, headers=HEADERS)
        response.raise_for_status()
        
        # Get first 1000 characters
        content = response.text[:1000]
        print(f"Filing content preview (first 1000 chars):")
        print(content)
        print(f"\nTotal document size: {len(response.text)} characters")
        
    except Exception as e:
        print(f"Error: {e}")

def main():
    """Run all tests"""
    print("=" * 60)
    print("SEC EDGAR API Test Suite")
    print("=" * 60)
    
    # Test 1: Get company tickers
    apple_cik = test_company_tickers()
    
    if apple_cik:
        # Add delay to respect rate limits
        time.sleep(0.5)
        
        # Test 2: Get company submissions
        submissions = test_company_submissions(apple_cik)
        
        time.sleep(0.5)
        
        # Test 3: Get company facts
        test_company_facts(apple_cik)
        
        time.sleep(0.5)
        
        # Test 4: Get recent filings
        test_recent_filings()
        
        # Test 5: Get actual filing document
        if submissions:
            # Find a recent 10-K
            for i in range(len(submissions['form'])):
                if submissions['form'][i] == '10-K':
                    accession = submissions['accessionNumber'][i]
                    time.sleep(0.5)
                    test_filing_document(apple_cik, accession)
                    break
    
    print("\n" + "=" * 60)
    print("Test Suite Complete!")
    print("=" * 60)

if __name__ == "__main__":
    main()