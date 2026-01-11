#!/usr/bin/env python3
"""
YOUR EXACT METHOD - No bullshit, just your formula
TTM = (Latest 3 10-Qs) + (Latest 10-K) - (Next 3 10-Qs)
"""

import requests
import json
import sys

class YourExactMethod:
    def __init__(self):
        self.base_url = "https://data.sec.gov/api/xbrl/companyfacts/"
        self.headers = {
            'User-Agent': 'InvestorCenter.ai financial-data-collector contact@investorcenter.ai'
        }
        
        self.cik_mapping = {
            'AAPL': '0000320193',
            'MSFT': '0000789019', 
            'GOOGL': '0001652044',
            'AMZN': '0001018724',
            'TSLA': '0001318605',
            'META': '0001326801',
            'NVDA': '0001045810',
        }

    def get_company_data(self, symbol):
        """Fetch company data from SEC EDGAR API"""
        cik = self.cik_mapping.get(symbol.upper())
        if not cik:
            print(f"❌ CIK not found for {symbol}")
            return None
        
        url = f"{self.base_url}CIK{cik}.json"
        
        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            print(f"❌ Error fetching data for {symbol}: {e}")
            return None

    def calculate_ttm_your_way(self, symbol):
        """Calculate TTM using YOUR EXACT METHOD"""
        print(f"\n🎯 {symbol} - YOUR EXACT METHOD")
        print("=" * 50)
        print("Formula: TTM = (Latest 3 10-Qs) + (Latest 10-K) - (Next 3 10-Qs)")
        print()
        
        facts_data = self.get_company_data(symbol)
        if not facts_data:
            return None
        
        us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
        
        # Try both revenue concepts and combine ALL data
        all_data = []
        for concept in ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax']:
            if concept in us_gaap:
                usd_data = us_gaap[concept].get('units', {}).get('USD', [])
                for item in usd_data:
                    item_copy = item.copy()
                    item_copy['source_concept'] = concept
                    all_data.append(item_copy)
        
        # Get latest 6 10-Qs - ONLY quarterly values (not YTD cumulative)
        all_10qs = [item for item in all_data 
                   if item.get('form') == '10-Q' 
                   and item.get('frame') and 'Q' in item.get('frame', '')  # ONLY quarterly frames
                   and item.get('end', '') >= '2022-01-01']
        
        # Remove duplicates (same end date and frame)
        unique_10qs = {}
        for item in all_10qs:
            key = f"{item.get('end')}_{item.get('frame')}"
            if key not in unique_10qs:
                unique_10qs[key] = item
        
        latest_6_10qs = sorted(unique_10qs.values(), key=lambda x: x.get('end', ''), reverse=True)[:6]
        
        # Get latest 10-K
        all_10ks = [item for item in all_data if item.get('form') == '10-K']
        latest_10k = sorted(all_10ks, key=lambda x: x.get('end', ''), reverse=True)[0] if all_10ks else None
        
        # Check if latest filing is 10-K or 10-Q
        all_filings = all_10qs + all_10ks if latest_10k else all_10qs
        latest_filing = sorted(all_filings, key=lambda x: x.get('end', ''), reverse=True)[0] if all_filings else None
        
        if latest_filing and latest_filing.get('form') == '10-K':
            # Latest is 10-K - use it directly (already TTM)
            annual_value = latest_filing.get('val', 0)
            annual_end = latest_filing.get('end')
            print(f"📊 LATEST FILING IS 10-K:")
            print(f"  {annual_end}: ${annual_value/1000000000:.2f}B (already TTM)")
            print(f"🎯 USING 10-K DIRECTLY")
            return annual_value
        
        elif len(latest_6_10qs) >= 6 and latest_10k:
            print("📊 LATEST 6 10-Qs:")
            first_3_sum = 0
            next_3_sum = 0
            
            for i, q in enumerate(latest_6_10qs):
                value = q.get('val', 0) / 1000000000
                end_date = q.get('end')
                source = q.get('source_concept', 'Unknown')
                
                if i < 3:
                    first_3_sum += q.get('val', 0)
                    print(f"  {i+1}. {end_date}: ${value:.2f}B (from {source}) ✅ ADD")
                else:
                    next_3_sum += q.get('val', 0)
                    print(f"  {i+1}. {end_date}: ${value:.2f}B (from {source}) ❌ SUBTRACT")
            
            print(f"\n📊 LATEST 10-K:")
            annual_value = latest_10k.get('val', 0)
            annual_end = latest_10k.get('end')
            print(f"  {annual_end}: ${annual_value/1000000000:.2f}B ✅ ADD")
            
            print(f"\n🧮 YOUR CALCULATION:")
            print(f"First 3 10-Qs: ${first_3_sum/1000000000:.2f}B")
            print(f"Latest 10-K: ${annual_value/1000000000:.2f}B")
            print(f"Next 3 10-Qs: ${next_3_sum/1000000000:.2f}B")
            
            ttm_result = first_3_sum + annual_value - next_3_sum
            print(f"TTM = ${first_3_sum/1000000000:.2f}B + ${annual_value/1000000000:.2f}B - ${next_3_sum/1000000000:.2f}B")
            print(f"TTM = ${ttm_result/1000000000:.2f}B")
            
            return ttm_result
        else:
            print(f"❌ Need 6 10-Qs and 1 10-K, found {len(latest_6_10qs)} 10-Qs and {1 if latest_10k else 0} 10-K")
            return None

def main():
    calculator = YourExactMethod()
    
    if len(sys.argv) > 1:
        symbol = sys.argv[1].upper()
    else:
        symbol = 'TSLA'
    
    result = calculator.calculate_ttm_your_way(symbol)
    
    if result:
        revenue_b = result / 1000000000
        print(f"\n🎯 FINAL RESULT: ${revenue_b:.2f}B")
        
        if symbol == 'TSLA':
            target = 95.63
            diff = abs(revenue_b - target)
            if diff < 0.1:
                print(f"🎉 EXACT MATCH! (vs ${target:.2f}B target)")
            else:
                print(f"❌ Off by ${diff:.2f}B (vs ${target:.2f}B target)")
    else:
        print("❌ Failed to calculate TTM")

if __name__ == "__main__":
    main()
