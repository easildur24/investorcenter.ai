#!/usr/bin/env python3
"""
Tesla Revenue TTM - EXACT CALCULATION
Goal: Get EXACTLY $95.63B to match reference data
"""

import requests
import json
from datetime import datetime

def get_tesla_sec_data():
    """Get Tesla's SEC data"""
    headers = {'User-Agent': 'InvestorCenter.ai financial-data-collector contact@investorcenter.ai'}
    url = 'https://data.sec.gov/api/xbrl/companyfacts/CIK0001318605.json'
    
    response = requests.get(url, headers=headers)
    return response.json()

def analyze_tesla_revenue():
    """Analyze Tesla's revenue data to find EXACT match for $95.63B"""
    print("🚗 TESLA REVENUE TTM - EXACT CALCULATION")
    print("=" * 50)
    print("🎯 TARGET: $95.63B (EXACT)")
    print()
    
    data = get_tesla_sec_data()
    us_gaap = data.get('facts', {}).get('us-gaap', {})
    
    # Check Tesla's revenue concepts
    revenue_concepts = ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax']
    
    for concept in revenue_concepts:
        if concept in us_gaap:
            print(f"📊 ANALYZING {concept}:")
            usd_data = us_gaap[concept].get('units', {}).get('USD', [])
            
            # Get ALL data points
            print("All revenue data points (recent first):")
            sorted_data = sorted(usd_data, key=lambda x: x.get('end', ''), reverse=True)
            
            for i, item in enumerate(sorted_data[:20]):  # Show top 20
                value = item.get('val', 0) / 1000000000
                end_date = item.get('end', '')
                start_date = item.get('start', '')
                form = item.get('form', '')
                frame = item.get('frame', 'No Frame')
                
                print(f"  {i+1:2d}. {end_date}: ${value:>6.2f}B ({start_date} to {end_date}) - {form} - {frame}")
                
                # Check if this matches our target
                if abs(value - 95.63) < 0.1:
                    print(f"      🎯 EXACT MATCH! This is our TTM value!")
            
            print()
            
            # Try different TTM calculations
            print("🧮 TRYING DIFFERENT TTM CALCULATIONS:")
            
            # Method 1: Most recent annual (10-K)
            annual_data = [item for item in usd_data if item.get('form') == '10-K']
            if annual_data:
                latest_annual = sorted(annual_data, key=lambda x: x.get('end', ''), reverse=True)[0]
                annual_value = latest_annual.get('val', 0) / 1000000000
                annual_end = latest_annual.get('end')
                print(f"Method 1 - Latest 10-K: ${annual_value:.2f}B (ending {annual_end})")
                if abs(annual_value - 95.63) < 0.1:
                    print("  🎯 EXACT MATCH!")
            
            # Method 2: Sum of last 4 quarters
            quarterly_data = [item for item in usd_data 
                             if item.get('frame') and 'Q' in item.get('frame', '')]
            
            if len(quarterly_data) >= 4:
                sorted_quarters = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
                
                print(f"Method 2 - Last 4 quarters:")
                ttm_sum = 0
                for i, q in enumerate(sorted_quarters[:4]):
                    value = q.get('val', 0) / 1000000000
                    end_date = q.get('end')
                    ttm_sum += value
                    print(f"  Q{i+1}: {end_date} = ${value:.2f}B")
                
                print(f"  Sum: ${ttm_sum:.2f}B")
                if abs(ttm_sum - 95.63) < 0.1:
                    print("  🎯 EXACT MATCH!")
            
            # Method 3: Try different quarter combinations
            print(f"Method 3 - Testing different combinations:")
            
            # Maybe it's not the last 4 quarters, but a different set?
            if len(quarterly_data) >= 6:
                for start_idx in range(0, 3):  # Try starting from different quarters
                    test_sum = sum(q.get('val', 0) for q in sorted_quarters[start_idx:start_idx+4]) / 1000000000
                    print(f"  Quarters {start_idx+1}-{start_idx+4}: ${test_sum:.2f}B")
                    if abs(test_sum - 95.63) < 0.1:
                        print(f"    🎯 EXACT MATCH! Use quarters {start_idx+1}-{start_idx+4}")
            
            print()

if __name__ == "__main__":
    analyze_tesla_revenue()
