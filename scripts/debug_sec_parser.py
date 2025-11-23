#!/usr/bin/env python3
"""
Debug SEC Filing Parser - Let's see what we're actually extracting
"""

import boto3
import re
import os

def download_recent_filing():
    """Download the most recent AAPL 10-Q to examine"""
    s3_client = boto3.client('s3')
    bucket_name = 'investorcenter-sec-filings'
    
    # Download the most recent 10-Q (Q3 2025)
    key = 'filings/AAPL/10-Q/2025/2025-08-01_0000320193-25-000073.html'
    
    response = s3_client.get_object(Bucket=bucket_name, Key=key)
    content = response['Body'].read().decode('utf-8')
    
    return content

def find_financial_tables(content):
    """Find and extract financial statement tables"""
    print("🔍 Looking for financial statement patterns...")
    
    # Look for consolidated statements of operations
    operations_patterns = [
        r'CONSOLIDATED STATEMENTS OF OPERATIONS.*?</table>',
        r'Consolidated Statements of Operations.*?</table>',
        r'STATEMENTS OF OPERATIONS.*?</table>',
    ]
    
    for i, pattern in enumerate(operations_patterns):
        matches = re.findall(pattern, content, re.IGNORECASE | re.DOTALL)
        if matches:
            print(f"✅ Found operations statement with pattern {i+1}")
            print(f"   Length: {len(matches[0])} characters")
            
            # Save to file for inspection
            with open('/tmp/operations_table.html', 'w') as f:
                f.write(matches[0])
            print("   Saved to /tmp/operations_table.html")
            return matches[0]
    
    print("❌ No operations statement found")
    return None

def extract_revenue_simple(content):
    """Simple revenue extraction"""
    print("\n🔍 Looking for revenue data...")
    
    # Look for Products and Services revenue
    # Apple typically shows: Products, Services, Total net sales
    
    # Pattern 1: Look for "Products" followed by dollar amounts
    products_pattern = r'Products.*?\$\s*([0-9,]+)'
    products_matches = re.findall(products_pattern, content, re.IGNORECASE)
    
    print(f"Products matches: {products_matches[:5]}")  # Show first 5
    
    # Pattern 2: Look for "Services" 
    services_pattern = r'Services.*?\$\s*([0-9,]+)'
    services_matches = re.findall(services_pattern, content, re.IGNORECASE)
    
    print(f"Services matches: {services_matches[:5]}")
    
    # Pattern 3: Look for "Total net sales" or "Net sales"
    total_pattern = r'(?:Total )?[Nn]et sales.*?\$\s*([0-9,]+)'
    total_matches = re.findall(total_pattern, content, re.IGNORECASE)
    
    print(f"Total sales matches: {total_matches[:5]}")

def main():
    print("🚀 Debug SEC Parser for AAPL")
    print("="*50)
    
    # Download recent filing
    print("📥 Downloading recent 10-Q...")
    content = download_recent_filing()
    print(f"✅ Downloaded {len(content)} characters")
    
    # Find financial tables
    operations_table = find_financial_tables(content)
    
    # Extract revenue data
    extract_revenue_simple(content)
    
    # If we found the operations table, analyze it specifically
    if operations_table:
        print("\n🔍 Analyzing operations table specifically...")
        extract_revenue_simple(operations_table)

if __name__ == "__main__":
    main()
