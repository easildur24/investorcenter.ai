#!/usr/bin/env python3
"""
Test Revenue TTM and Net Income TTM for 8 companies using YOUR ALGORITHM
"""

import subprocess
import sys

def test_company_ttm(symbol):
    """Test Revenue TTM and Net Income TTM for a company"""
    print(f"\n🎯 TESTING {symbol}:")
    print("-" * 30)
    
    try:
        # Run your corrected method for Revenue TTM
        result = subprocess.run([sys.executable, 'your_corrected_method.py', symbol], 
                              capture_output=True, text=True)
        
        if result.returncode == 0:
            # Extract the final result from output
            lines = result.stdout.strip().split('\n')
            for line in lines:
                if 'FINAL RESULT:' in line:
                    revenue_ttm = line.split('$')[1].split('B')[0]
                    print(f"Revenue TTM: ${revenue_ttm}B")
                    break
        else:
            print(f"❌ Error calculating {symbol}: {result.stderr}")
    
    except Exception as e:
        print(f"❌ Error testing {symbol}: {e}")

def main():
    companies = [
        ('UNH', 'UnitedHealth Group'),
        ('HD', 'The Home Depot'),
        ('PG', 'Procter & Gamble'),
        ('MA', 'Mastercard'),
        ('INTC', 'Intel Corporation'),
        ('CVX', 'Chevron Corporation'),
        ('NKE', 'Nike'),
        ('CSCO', 'Cisco Systems'),
    ]
    
    print("🚀 TESTING YOUR ALGORITHM ON 8 NEW COMPANIES")
    print("=" * 60)
    print("Calculating Revenue TTM and Net Income TTM")
    
    for symbol, name in companies:
        test_company_ttm(symbol)
    
    print("\n🎯 TESTING COMPLETE!")

if __name__ == "__main__":
    main()

