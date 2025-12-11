#!/usr/bin/env python3
"""
YOUR CORRECTED ALGORITHM
TTM = (Latest 3 10-Qs) + (Latest 10-K) - (3 10-Qs right before the 10-K)
"""

import requests
import json
import sys

class YourCorrectedMethod:
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
            'HIMS': '0001773751',
            'UNH': '0000731766',
            'HD': '0000354950',
            'PG': '0000080424',
            'MA': '0001141391',
            'INTC': '0000050863',
            'CVX': '0000093410',
            'NKE': '0000320187',
            'CSCO': '0000858877',
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

    def calculate_ttm_corrected(self, symbol):
        """Calculate TTM using YOUR CORRECTED ALGORITHM"""
        print(f"\n🎯 {symbol} - YOUR CORRECTED ALGORITHM")
        print("=" * 60)
        print("Formula: TTM = (Latest 3 10-Qs) + (Latest 10-K) - (3 10-Qs before 10-K)")
        print()
        
        facts_data = self.get_company_data(symbol)
        if not facts_data:
            return None
        
        us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
        
        # Get ALL data from both revenue concepts
        all_data = []
        for concept in ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax']:
            if concept in us_gaap:
                usd_data = us_gaap[concept].get('units', {}).get('USD', [])
                for item in usd_data:
                    item_copy = item.copy()
                    item_copy['source_concept'] = concept
                    all_data.append(item_copy)
        
        # Get 10-Qs (ONLY quarterly frames, no YTD)
        all_10qs = [item for item in all_data 
                   if item.get('form') == '10-Q' 
                   and item.get('frame') and 'Q' in item.get('frame', '')
                   and item.get('end', '') >= '2020-01-01']
        
        # Remove duplicates
        unique_10qs = {}
        for item in all_10qs:
            key = f"{item.get('end')}_{item.get('frame')}"
            if key not in unique_10qs:
                unique_10qs[key] = item
        
        sorted_10qs = sorted(unique_10qs.values(), key=lambda x: x.get('end', ''), reverse=True)
        
        # Get latest 10-K
        all_10ks = [item for item in all_data if item.get('form') == '10-K']
        latest_10k = sorted(all_10ks, key=lambda x: x.get('end', ''), reverse=True)[0] if all_10ks else None
        
        if len(sorted_10qs) >= 6 and latest_10k:
            # Get latest 3 10-Qs
            latest_3_10qs = sorted_10qs[:3]
            
            # Find 3 10-Qs right BEFORE the 10-K
            tenk_end_date = latest_10k.get('end')
            before_10k_10qs = [q for q in sorted_10qs if q.get('end') < tenk_end_date]
            before_10k_3 = sorted(before_10k_10qs, key=lambda x: x.get('end', ''), reverse=True)[:3]
            
            print("📊 LATEST 3 10-Qs (ADD):")
            latest_3_sum = 0
            for i, q in enumerate(latest_3_10qs):
                value = q.get('val', 0) / 1000000000
                end_date = q.get('end')
                frame = q.get('frame', 'No Frame')
                source = q.get('source_concept', 'Unknown')
                latest_3_sum += q.get('val', 0)
                print(f"  {i+1}. {frame} ({end_date}): ${value:.2f}B")
            
            print(f"\n📊 LATEST 10-K (ADD):")
            annual_value = latest_10k.get('val', 0)
            annual_end = latest_10k.get('end')
            print(f"  {annual_end}: ${annual_value/1000000000:.2f}B")
            
            print(f"\n📊 3 10-Qs BEFORE 10-K (SUBTRACT):")
            before_3_sum = 0
            for i, q in enumerate(before_10k_3):
                value = q.get('val', 0) / 1000000000
                end_date = q.get('end')
                frame = q.get('frame', 'No Frame')
                before_3_sum += q.get('val', 0)
                print(f"  {i+1}. {frame} ({end_date}): ${value:.2f}B")
            
            print(f"\n🧮 YOUR CALCULATION:")
            print(f"Latest 3 10-Qs: ${latest_3_sum/1000000000:.2f}B")
            print(f"Latest 10-K: ${annual_value/1000000000:.2f}B")
            print(f"3 10-Qs before 10-K: ${before_3_sum/1000000000:.2f}B")
            
            ttm_result = latest_3_sum + annual_value - before_3_sum
            print(f"TTM = ${latest_3_sum/1000000000:.2f}B + ${annual_value/1000000000:.2f}B - ${before_3_sum/1000000000:.2f}B")
            print(f"TTM = ${ttm_result/1000000000:.2f}B")
            
            return ttm_result
        else:
            print(f"❌ Need 6+ 10-Qs and 1 10-K, found {len(sorted_10qs)} 10-Qs and {1 if latest_10k else 0} 10-K")
            return None

def main():
    calculator = YourCorrectedMethod()
    
    if len(sys.argv) > 1:
        symbol = sys.argv[1].upper()
    else:
        symbol = 'MSFT'
    
    result = calculator.calculate_ttm_corrected(symbol)
    
    if result:
        revenue_b = result / 1000000000
        print(f"\n🎯 FINAL RESULT: ${revenue_b:.2f}B")
    else:
        print("❌ Failed to calculate TTM")

if __name__ == "__main__":
    main()
