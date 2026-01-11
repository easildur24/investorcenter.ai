#!/usr/bin/env python3
"""
Mass test Revenue TTM across all supported tickers
"""

from cik_manager import CIKManager
from metrics import MetricsCalculator
import time

def test_all_tickers():
    cik_manager = CIKManager()
    calculator = MetricsCalculator()
    
    # Get all supported tickers
    tickers = sorted(cik_manager.hardcoded_ciks.keys())
    
    print(f"🎯 TESTING {len(tickers)} COMPANIES")
    print("=" * 60)
    
    results = []
    successful = 0
    failed = 0
    
    for i, ticker in enumerate(tickers, 1):
        print(f"\n[{i}/{len(tickers)}] 🧪 TESTING {ticker}...")
        
        try:
            # Get company data
            facts_data = calculator.get_company_data(ticker)
            if not facts_data:
                print(f"  ❌ {ticker}: Failed to fetch data")
                results.append((ticker, "FAILED", "No data"))
                failed += 1
                continue
            
            # Calculate Revenue TTM
            revenue_ttm = calculator.calculate_revenue_ttm(facts_data)
            
            if revenue_ttm:
                revenue_billions = revenue_ttm / 1_000_000_000
                print(f"  ✅ {ticker}: ${revenue_billions:.2f}B")
                results.append((ticker, "SUCCESS", f"${revenue_billions:.2f}B"))
                successful += 1
            else:
                print(f"  ❌ {ticker}: Failed to calculate")
                results.append((ticker, "FAILED", "Calculation failed"))
                failed += 1
                
        except Exception as e:
            print(f"  ❌ {ticker}: Error - {e}")
            results.append((ticker, "ERROR", str(e)))
            failed += 1
        
        # Small delay to be nice to SEC API
        time.sleep(0.1)
    
    # Summary
    print("\n" + "=" * 60)
    print("🎯 FINAL RESULTS:")
    print("=" * 60)
    
    success_rate = (successful / len(tickers)) * 100
    print(f"✅ SUCCESSFUL: {successful}/{len(tickers)} ({success_rate:.1f}%)")
    print(f"❌ FAILED: {failed}/{len(tickers)} ({100-success_rate:.1f}%)")
    
    print(f"\n📊 SUCCESS BREAKDOWN:")
    for ticker, status, value in results:
        if status == "SUCCESS":
            print(f"  ✅ {ticker}: {value}")
    
    if failed > 0:
        print(f"\n❌ FAILURE BREAKDOWN:")
        for ticker, status, error in results:
            if status != "SUCCESS":
                print(f"  ❌ {ticker}: {error}")

if __name__ == "__main__":
    test_all_tickers()
