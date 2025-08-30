#!/usr/bin/env python3
"""
Example usage of the SEC filings downloader for HIMS & Hers Health, Inc.

This script demonstrates various ways to use the SEC filings downloader.
"""

from sec_filings_downloader import SECFilingsDownloader

def main():
    """Demonstrate different ways to use the SEC filings downloader."""
    
    # Initialize the downloader
    downloader = SECFilingsDownloader()
    
    print("=== SEC Filings Downloader Examples ===\n")
    
    # Example 1: Download recent filings metadata only
    print("1. Downloading 5 most recent HIMS filings (metadata only)...")
    downloader.download_filings(
        ticker="HIMS",
        count=5,
        download_documents=False,
        output_dir="examples/hims_recent"
    )
    print("✓ Complete\n")
    
    # Example 2: Download specific form types
    print("2. Downloading HIMS 10-K filings...")
    downloader.download_filings(
        ticker="HIMS",
        form_type="10-K",
        count=3,
        download_documents=False,
        output_dir="examples/hims_10k"
    )
    print("✓ Complete\n")
    
    # Example 3: Download documents as well
    print("3. Downloading HIMS 8-K filings with documents...")
    downloader.download_filings(
        ticker="HIMS",
        form_type="8-K",
        count=2,
        download_documents=True,
        output_dir="examples/hims_8k_with_docs"
    )
    print("✓ Complete\n")
    
    # Example 4: Using the class methods directly
    print("4. Using downloader methods directly...")
    
    # Get filings data
    filings_data = downloader.get_filings_data("HIMS", form_type="10-Q", count=2)
    print(f"Found {len(filings_data)} 10-Q filings")
    
    for filing in filings_data:
        print(f"  - {filing['formType']} filed on {filing['filingDate']}")
    
    print("\n=== All examples completed! ===")
    print("\nCheck the 'examples/' directory for output files.")
    print("Files include:")
    print("  - CSV files with filing metadata")
    print("  - JSON files with raw API responses") 
    print("  - HTML documents (when --download-documents is used)")


if __name__ == "__main__":
    main()
