#!/usr/bin/env python3
"""
Revenue TTM and Net Income TTM for multiple companies
Using YOUR BULLETPROOF ALGORITHM
"""

import requests
import json
import sys

class RevenueAndNITTM:
    def __init__(self):
        self.base_url = "https://data.sec.gov/api/xbrl/companyfacts/"
        self.headers = {
            'User-Agent': 'InvestorCenter.ai financial-data-collector contact@investorcenter.ai'
        }
        
        self.cik_mapping = {
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

    def calculate_ttm_your_way(self, facts_data, concepts, metric_name):
        """Calculate TTM using YOUR ALGORITHM"""
        us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
        
        # Get ALL data from all concepts
        all_data = []
        for concept in concepts:
            if concept in us_gaap:
                usd_data = us_gaap[concept].get('units', {}).get('USD', [])
                for item in usd_data:
                    item_copy = item.copy()
                    item_copy['source_concept'] = concept
                    all_data.append(item_copy)
        
        if not all_data:
            return None
        
        # Get 10-Qs (ONLY quarterly frames)
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
            # YOUR ALGORITHM
            latest_3_10qs = sorted_10qs[:3]
            
            # Find 3 10-Qs right BEFORE the 10-K
            tenk_end_date = latest_10k.get('end')
            before_10k_10qs = [q for q in sorted_10qs if q.get('end') < tenk_end_date]
            before_10k_3 = sorted(before_10k_10qs, key=lambda x: x.get('end', ''), reverse=True)[:3]
            
            # Calculate
            latest_3_sum = sum(q.get('val', 0) for q in latest_3_10qs)
            annual_value = latest_10k.get('val', 0)
            before_3_sum = sum(q.get('val', 0) for q in before_10k_3)
            
            ttm_result = latest_3_sum + annual_value - before_3_sum
            return ttm_result
        
        return None

    def test_company(self, symbol, company_name):
        """Test Revenue TTM and Net Income TTM for one company"""
        print(f"\n🎯 {symbol} - {company_name}")
        print("-" * 50)
        
        facts_data = self.get_company_data(symbol)
        if not facts_data:
            print("❌ Failed to get company data")
            return
        
        # Revenue TTM
        revenue_concepts = ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax']
        revenue_ttm = self.calculate_ttm_your_way(facts_data, revenue_concepts, 'Revenue')
        
        # Net Income TTM
        ni_concepts = ['NetIncomeLoss']
        ni_ttm = self.calculate_ttm_your_way(facts_data, ni_concepts, 'Net Income')
        
        # Results
        if revenue_ttm:
            print(f"Revenue TTM: ${revenue_ttm/1000000000:.2f}B")
        else:
            print("Revenue TTM: ❌ FAILED")
        
        if ni_ttm:
            print(f"Net Income TTM: ${ni_ttm/1000000:.1f}M")
        else:
            print("Net Income TTM: ❌ FAILED")

def main():
    calculator = RevenueAndNITTM()
    
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
    
    print("🚀 REVENUE TTM AND NET INCOME TTM - 8 COMPANIES")
    print("=" * 60)
    print("Using YOUR BULLETPROOF ALGORITHM")
    
    for symbol, name in companies:
        calculator.test_company(symbol, name)
    
    print("\n🎯 TESTING COMPLETE!")
    print("Your algorithm tested on 8 additional companies!")

if __name__ == "__main__":
    main()
