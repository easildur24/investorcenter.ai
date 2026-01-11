#!/usr/bin/env python3
"""
Check Polygon API for latest quarter data to verify our SEC calculations
"""

import requests
import os
import sys

def check_polygon_fundamentals(symbol):
    """Check Polygon API for latest fundamental data"""
    api_key = os.getenv('POLYGON_API_KEY')
    if not api_key:
        print("❌ POLYGON_API_KEY not set")
        return None
    
    # Get TTM fundamentals
    url = f"https://api.polygon.io/vX/reference/financials?ticker={symbol}&timeframe=ttm&limit=1&apikey={api_key}"
    
    try:
        response = requests.get(url)
        response.raise_for_status()
        data = response.json()
        
        if data.get('status') == 'OK' and data.get('results'):
            result = data['results'][0]
            
            print(f"📊 POLYGON DATA FOR {symbol}:")
            print(f"Period: {result.get('start_date')} to {result.get('end_date')}")
            
            # Get revenue
            financials = result.get('financials', {})
            income_statement = financials.get('income_statement', {})
            
            if 'revenues' in income_statement:
                revenue = income_statement['revenues'].get('value', 0) / 1000000000
                print(f"Revenue TTM: ${revenue:.2f}B")
                return revenue
            
        return None
    except Exception as e:
        print(f"❌ Error fetching Polygon data for {symbol}: {e}")
        return None

def main():
    symbols = ['GOOGL', 'META', 'TSLA']
    
    print("🔍 CHECKING POLYGON API FOR LATEST QUARTER DATA")
    print("=" * 60)
    
    for symbol in symbols:
        print(f"\n🔍 {symbol}:")
        polygon_revenue = check_polygon_fundamentals(symbol)
        
        if polygon_revenue:
            print(f"Polygon Revenue TTM: ${polygon_revenue:.2f}B")
            
            # Compare with our V4 calculation
            if symbol == 'TSLA':
                our_revenue = 95.63
                print(f"Our V4 Revenue TTM: ${our_revenue:.2f}B")
                diff = abs(polygon_revenue - our_revenue)
                print(f"Difference: ${diff:.2f}B ({diff/polygon_revenue*100:.1f}%)")
            elif symbol == 'GOOGL':
                our_revenue = 359.71
                print(f"Our V4 Revenue TTM: ${our_revenue:.2f}B")
                diff = abs(polygon_revenue - our_revenue)
                print(f"Difference: ${diff:.2f}B ({diff/polygon_revenue*100:.1f}%)")
            elif symbol == 'META':
                our_revenue = 51.90
                print(f"Our V4 Revenue TTM: ${our_revenue:.2f}B")
                diff = abs(polygon_revenue - our_revenue)
                print(f"Difference: ${diff:.2f}B ({diff/polygon_revenue*100:.1f}%)")
        else:
            print("❌ Could not get Polygon data")

if __name__ == "__main__":
    main()
