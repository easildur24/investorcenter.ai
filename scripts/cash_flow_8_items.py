#!/usr/bin/env python3
"""
8 Cash Flow Items - Using YOUR ALGORITHM
TTM = (Latest 3 10-Qs) + (Latest 10-K) - (3 10-Qs before 10-K)
"""

import requests
import json
import sys

class CashFlow8Items:
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

    def calculate_ttm_cash_flow(self, facts_data, concepts, metric_name):
        """Calculate TTM cash flow using YOUR ALGORITHM"""
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
            print(f"  ❌ No data found for {metric_name}")
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
            # YOUR ALGORITHM: (Latest 3 10-Qs) + (Latest 10-K) - (3 10-Qs before 10-K)
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
            
            print(f"  ✅ {metric_name} TTM: ${ttm_result/1000000:.1f}M")
            print(f"    = Latest 3 10-Qs (${latest_3_sum/1000000:.1f}M) + 10-K (${annual_value/1000000:.1f}M) - Before 3 10-Qs (${before_3_sum/1000000:.1f}M)")
            
            return ttm_result
        else:
            print(f"  ❌ {metric_name}: Need 6+ 10-Qs and 1 10-K, found {len(sorted_10qs)} 10-Qs and {1 if latest_10k else 0} 10-K")
            return None

    def get_latest_quarterly_value(self, facts_data, concepts, metric_name):
        """Get latest quarterly value (for Ending Cash)"""
        us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
        
        all_data = []
        for concept in concepts:
            if concept in us_gaap:
                usd_data = us_gaap[concept].get('units', {}).get('USD', [])
                all_data.extend(usd_data)
        
        if all_data:
            latest_item = sorted(all_data, key=lambda x: x.get('end', ''), reverse=True)[0]
            value = latest_item.get('val', 0)
            end_date = latest_item.get('end')
            form = latest_item.get('form', '')
            
            print(f"  ✅ {metric_name}: ${value/1000000:.1f}M (as of {end_date} - {form})")
            return value
        
        print(f"  ❌ No data found for {metric_name}")
        return None

    def calculate_cash_flow_8(self, symbol):
        """Calculate 8 cash flow items"""
        print(f"\n🎯 {symbol} - 8 CASH FLOW ITEMS")
        print("=" * 50)
        
        facts_data = self.get_company_data(symbol)
        if not facts_data:
            return None
        
        results = {}
        
        # 1. Cash from Operations (TTM)
        print("📊 1. CASH FROM OPERATIONS (TTM):")
        results['operating_cf_ttm'] = self.calculate_ttm_cash_flow(
            facts_data, ['NetCashProvidedByUsedInOperatingActivities'], 'Operating Cash Flow'
        )
        
        # 2. Cash from Investing (TTM)
        print("\n📊 2. CASH FROM INVESTING (TTM):")
        results['investing_cf_ttm'] = self.calculate_ttm_cash_flow(
            facts_data, ['NetCashProvidedByUsedInInvestingActivities'], 'Investing Cash Flow'
        )
        
        # 3. Cash from Financing (TTM)
        print("\n📊 3. CASH FROM FINANCING (TTM):")
        results['financing_cf_ttm'] = self.calculate_ttm_cash_flow(
            facts_data, ['NetCashProvidedByUsedInFinancingActivities'], 'Financing Cash Flow'
        )
        
        # 4. Change in Receivables (TTM)
        print("\n📊 4. CHANGE IN RECEIVABLES (TTM):")
        results['change_receivables_ttm'] = self.calculate_ttm_cash_flow(
            facts_data, ['IncreaseDecreaseInAccountsReceivable'], 'Change in Receivables'
        )
        
        # 5. Changes in Working Capital (TTM)
        print("\n📊 5. CHANGES IN WORKING CAPITAL (TTM):")
        results['working_capital_ttm'] = self.calculate_ttm_cash_flow(
            facts_data, ['IncreaseDecreaseInOperatingCapital', 'IncreaseDecreaseInWorkingCapital'], 'Working Capital Change'
        )
        
        # 6. Capital Expenditures (TTM)
        print("\n📊 6. CAPITAL EXPENDITURES (TTM):")
        results['capex_ttm'] = self.calculate_ttm_cash_flow(
            facts_data, 
            ['PaymentsToAcquirePropertyPlantAndEquipment', 'PaymentsToAcquireProductiveAssets', 'CapitalExpenditures'], 
            'Capital Expenditures'
        )
        
        # 7. Ending Cash (Quarterly)
        print("\n📊 7. ENDING CASH (QUARTERLY):")
        results['ending_cash'] = self.get_latest_quarterly_value(
            facts_data, ['CashAndCashEquivalentsAtCarryingValue'], 'Ending Cash'
        )
        
        # 8. Free Cash Flow (Operating CF - CapEx)
        print("\n📊 8. FREE CASH FLOW:")
        if results.get('operating_cf_ttm') and results.get('capex_ttm'):
            results['free_cash_flow'] = results['operating_cf_ttm'] - abs(results['capex_ttm'])
            print(f"  ✅ Free Cash Flow: ${results['free_cash_flow']/1000000:.1f}M (Operating CF - CapEx)")
        else:
            print("  ❌ Could not calculate Free Cash Flow (missing Operating CF or CapEx)")
        
        return results

def main():
    calculator = CashFlow8Items()
    
    symbols = ['TSLA', 'HIMS', 'GOOGL'] if len(sys.argv) == 1 else [sys.argv[1].upper()]
    
    for symbol in symbols:
        results = calculator.calculate_cash_flow_8(symbol)
        
        if results:
            print(f"\n🎯 {symbol} CASH FLOW SUMMARY:")
            print("-" * 40)
            cash_flow_items = [
                'operating_cf_ttm', 'investing_cf_ttm', 'financing_cf_ttm',
                'change_receivables_ttm', 'working_capital_ttm', 'capex_ttm',
                'ending_cash', 'free_cash_flow'
            ]
            
            for item in cash_flow_items:
                if results.get(item):
                    value = results[item] / 1000000000 if abs(results[item]) > 1000000000 else results[item] / 1000000
                    unit = 'B' if abs(results[item]) > 1000000000 else 'M'
                    print(f"{item.replace('_', ' ').title():<25}: ${value:.2f}{unit}")
                else:
                    print(f"{item.replace('_', ' ').title():<25}: NOT FOUND")
        
        print()

if __name__ == "__main__":
    main()

