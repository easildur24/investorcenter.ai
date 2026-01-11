#!/usr/bin/env python3
"""
HIMS 4 Core Metrics - Using YOUR CORRECTED ALGORITHM
Target EXACT matches:
- Revenue (TTM): $2.211B
- Net Income (TTM): $133.79M  
- EBIT (TTM): $131.81M
- EBITDA (TTM): $174.28M
"""

import requests
import json

class HIMS4Metrics:
    def __init__(self):
        self.base_url = "https://data.sec.gov/api/xbrl/companyfacts/"
        self.headers = {
            'User-Agent': 'InvestorCenter.ai financial-data-collector contact@investorcenter.ai'
        }
        self.hims_cik = '0001773751'

    def get_hims_data(self):
        """Get HIMS SEC data"""
        url = f"{self.base_url}CIK{self.hims_cik}.json"
        
        try:
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            print(f"❌ Error fetching HIMS data: {e}")
            return None

    def calculate_ttm_metric(self, facts_data, concepts, metric_name):
        """Calculate TTM using YOUR ALGORITHM for any metric"""
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
            
            print(f"  ✅ {metric_name} TTM: ${ttm_result/1000000:.1f}M")
            print(f"    = Latest 3 10-Qs (${latest_3_sum/1000000:.1f}M) + 10-K (${annual_value/1000000:.1f}M) - Before 3 10-Qs (${before_3_sum/1000000:.1f}M)")
            
            return ttm_result
        else:
            print(f"  ❌ {metric_name}: Need 6+ 10-Qs and 1 10-K, found {len(sorted_10qs)} 10-Qs and {1 if latest_10k else 0} 10-K")
            return None

    def calculate_hims_4_metrics(self):
        """Calculate the 4 core metrics for HIMS"""
        print("🏥 HIMS - 4 CORE METRICS USING YOUR ALGORITHM")
        print("=" * 60)
        print("🎯 TARGETS:")
        print("• Revenue (TTM): $2.211B")
        print("• Net Income (TTM): $133.79M")
        print("• EBIT (TTM): $131.81M") 
        print("• EBITDA (TTM): $174.28M")
        print()
        
        facts_data = self.get_hims_data()
        if not facts_data:
            return None
        
        results = {}
        
        # Revenue TTM
        print("📊 1. REVENUE TTM:")
        revenue_concepts = ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax']
        results['revenue_ttm'] = self.calculate_ttm_metric(facts_data, revenue_concepts, 'Revenue')
        
        # Net Income TTM
        print("\n📊 2. NET INCOME TTM:")
        ni_concepts = ['NetIncomeLoss']
        results['net_income_ttm'] = self.calculate_ttm_metric(facts_data, ni_concepts, 'Net Income')
        
        # EBIT TTM (Net Income + Tax + Interest)
        print("\n📊 3. EBIT TTM (Net Income + Tax + Interest):")
        
        # Tax Expense using YOUR ALGORITHM
        tax_concepts = ['IncomeTaxExpenseBenefit']
        tax_ttm = self.calculate_ttm_metric(facts_data, tax_concepts, 'Tax Expense')
        
        # Interest Expense using YOUR ALGORITHM  
        interest_concepts = ['InterestExpense', 'InterestIncomeExpenseNet']
        interest_ttm = self.calculate_ttm_metric(facts_data, interest_concepts, 'Interest Expense')
        
        # Calculate EBIT using your formula
        if results.get('net_income_ttm') and tax_ttm is not None and interest_ttm is not None:
            results['ebit_ttm'] = results['net_income_ttm'] + tax_ttm + interest_ttm
            print(f"  ✅ EBIT TTM: ${results['ebit_ttm']/1000000:.1f}M")
            print(f"    = Net Income (${results['net_income_ttm']/1000000:.1f}M) + Tax (${tax_ttm/1000000:.1f}M) + Interest (${interest_ttm/1000000:.1f}M)")
        else:
            print("  ❌ Could not calculate EBIT (missing components)")
            results['ebit_ttm'] = None
        
        # EBITDA TTM (EBIT + Depreciation)
        print("\n📊 4. EBITDA TTM:")
        depreciation_concepts = ['DepreciationDepletionAndAmortization', 'Depreciation', 'DepreciationAndAmortization']
        depreciation_ttm = self.calculate_ttm_metric(facts_data, depreciation_concepts, 'Depreciation')
        
        if results.get('ebit_ttm') and depreciation_ttm:
            results['ebitda_ttm'] = results['ebit_ttm'] + depreciation_ttm
            print(f"  ✅ EBITDA TTM: ${results['ebitda_ttm']/1000000:.1f}M (EBIT + Depreciation)")
        else:
            print("  ❌ Could not calculate EBITDA (missing EBIT or Depreciation)")
        
        return results

    def show_comparison(self, results):
        """Show comparison with target values"""
        targets = {
            'revenue_ttm': 2211000000,  # $2.211B
            'net_income_ttm': 133790000,  # $133.79M
            'ebit_ttm': 131810000,  # $131.81M
            'ebitda_ttm': 174280000,  # $174.28M
        }
        
        print("\n🎯 COMPARISON WITH TARGETS:")
        print("=" * 60)
        print("Metric                | Our Result    | Target        | Match")
        print("-" * 60)
        
        for key, target in targets.items():
            if key in results and results[key]:
                our_val = results[key]
                diff_pct = abs(our_val - target) / target * 100
                status = "✅ EXACT" if diff_pct < 1 else "✅ CLOSE" if diff_pct < 5 else "❌ OFF"
                
                if key == 'revenue_ttm':
                    print(f"Revenue TTM           | ${our_val/1000000000:.3f}B      | $2.211B       | {status}")
                else:
                    print(f"{key.replace('_', ' ').title():<20} | ${our_val/1000000:.1f}M       | ${target/1000000:.1f}M        | {status}")
            else:
                print(f"{key.replace('_', ' ').title():<20} | NOT CALC      | ${target/1000000:.1f}M        | ❌ FAILED")

def main():
    calculator = HIMS4Metrics()
    results = calculator.calculate_hims_4_metrics()
    
    if results:
        calculator.show_comparison(results)
    else:
        print("❌ Failed to calculate HIMS metrics")

if __name__ == "__main__":
    main()
