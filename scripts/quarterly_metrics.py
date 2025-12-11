#!/usr/bin/env python3
"""
QUARTERLY METRICS - Using YOUR ALGORITHM
For quarterly: Latest 10-Q OR (Latest 10-K - Previous 3 10-Qs)
"""

import requests
import json
import sys
from datetime import datetime

class QuarterlyMetrics:
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

    def calculate_quarterly_metric(self, facts_data, concepts, metric_name):
        """Calculate quarterly metric using YOUR ALGORITHM"""
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
        
        # YOUR QUARTERLY ALGORITHM
        if sorted_10qs and latest_10k:
            # Check if latest filing is 10-Q or 10-K
            all_filings = sorted_10qs + [latest_10k]
            latest_filing = sorted(all_filings, key=lambda x: x.get('end', ''), reverse=True)[0]
            
            if latest_filing.get('form') == '10-Q':
                # Latest is 10-Q - use it directly
                quarterly_value = latest_filing.get('val', 0)
                quarterly_end = latest_filing.get('end')
                frame = latest_filing.get('frame', 'No Frame')
                
                print(f"  ✅ {metric_name} Quarterly: ${quarterly_value/1000000:.1f}M ({frame} ending {quarterly_end})")
                return quarterly_value
                
            elif latest_filing.get('form') == '10-K':
                # Latest is 10-K - calculate Q4: 10-K - Previous 3 10-Qs
                annual_value = latest_filing.get('val', 0)
                annual_end = latest_filing.get('end')
                annual_start = latest_filing.get('start')
                
                # Find 3 10-Qs before this 10-K (within the annual period)
                before_10k_10qs = [q for q in sorted_10qs 
                                  if q.get('end') < annual_end and q.get('end') >= annual_start]
                
                if len(before_10k_10qs) >= 3:
                    before_3_sum = sum(q.get('val', 0) for q in before_10k_10qs[:3])
                    calculated_q4 = annual_value - before_3_sum
                    
                    print(f"  ✅ {metric_name} Quarterly: ${calculated_q4/1000000:.1f}M (Q4 calculated from 10-K)")
                    print(f"    = 10-K (${annual_value/1000000:.1f}M) - Previous 3 10-Qs (${before_3_sum/1000000:.1f}M)")
                    return calculated_q4
        
        print(f"  ❌ Could not calculate quarterly {metric_name}")
        return None

    def calculate_yoy_growth(self, facts_data, concepts, metric_name):
        """Calculate YoY growth using same quarter from previous year"""
        # Get current quarterly value
        current_quarterly = self.calculate_quarterly_metric(facts_data, concepts, f"Current {metric_name}")
        
        if not current_quarterly:
            return None
        
        # Find same quarter from previous year
        us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
        
        # Get all quarterly data
        all_quarterly_data = []
        for concept in concepts:
            if concept in us_gaap:
                usd_data = us_gaap[concept].get('units', {}).get('USD', [])
                quarterly_data = [item for item in usd_data 
                                if item.get('frame') and 'Q' in item.get('frame', '')
                                and item.get('end', '') >= '2020-01-01']
                all_quarterly_data.extend(quarterly_data)
        
        if len(all_quarterly_data) >= 5:  # Need at least 5 quarters for YoY
            sorted_quarters = sorted(all_quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
            
            # Find matching quarter from previous year (approximately 4 quarters back)
            current_end = sorted_quarters[0].get('end')
            current_date = datetime.strptime(current_end, '%Y-%m-%d')
            
            # Look for quarter approximately 1 year ago
            for i, q in enumerate(sorted_quarters[1:], 1):
                q_end = q.get('end')
                q_date = datetime.strptime(q_end, '%Y-%m-%d')
                days_diff = (current_date - q_date).days
                
                # If within 60 days of 1 year (300-420 days range)
                if 300 <= days_diff <= 420:
                    prev_value = q.get('val', 0)
                    
                    if prev_value > 0:
                        growth = ((current_quarterly - prev_value) / prev_value) * 100
                        print(f"  ✅ {metric_name} YoY Growth: {growth:.2f}% ({current_end} vs {q_end})")
                        return growth
                    break
        
        print(f"  ❌ Could not calculate {metric_name} YoY growth")
        return None

    def calculate_quarterly_metrics(self, symbol):
        """Calculate all quarterly metrics"""
        print(f"\n🎯 {symbol} - QUARTERLY METRICS USING YOUR ALGORITHM")
        print("=" * 60)
        
        facts_data = self.get_company_data(symbol)
        if not facts_data:
            return None
        
        results = {}
        
        # Revenue Quarterly
        print("📊 1. REVENUE QUARTERLY:")
        revenue_concepts = ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax']
        results['revenue_quarterly'] = self.calculate_quarterly_metric(facts_data, revenue_concepts, 'Revenue')
        
        # Net Income Quarterly
        print("\n📊 2. NET INCOME QUARTERLY:")
        ni_concepts = ['NetIncomeLoss']
        results['net_income_quarterly'] = self.calculate_quarterly_metric(facts_data, ni_concepts, 'Net Income')
        
        # EBIT Quarterly (using your understanding)
        print("\n📊 3. EBIT QUARTERLY:")
        if results.get('net_income_quarterly'):
            # For quarterly EBIT, we can use operating income directly or calculate from Net Income
            ebit_concepts = ['OperatingIncomeLoss']
            results['ebit_quarterly'] = self.calculate_quarterly_metric(facts_data, ebit_concepts, 'EBIT')
        
        # EBITDA Quarterly
        print("\n📊 4. EBITDA QUARTERLY:")
        if results.get('ebit_quarterly'):
            # Get quarterly depreciation
            depreciation_concepts = ['DepreciationDepletionAndAmortization', 'Depreciation']
            quarterly_depreciation = self.calculate_quarterly_metric(facts_data, depreciation_concepts, 'Depreciation')
            
            # Get quarterly amortization
            amortization_concepts = ['AmortizationOfIntangibleAssets']
            quarterly_amortization = self.calculate_quarterly_metric(facts_data, amortization_concepts, 'Amortization')
            
            if quarterly_depreciation and quarterly_amortization:
                results['ebitda_quarterly'] = results['ebit_quarterly'] + quarterly_depreciation + quarterly_amortization
                print(f"  ✅ EBITDA Quarterly: ${results['ebitda_quarterly']/1000000:.1f}M (EBIT + D&A)")
            elif quarterly_depreciation:
                results['ebitda_quarterly'] = results['ebit_quarterly'] + quarterly_depreciation
                print(f"  ✅ EBITDA Quarterly: ${results['ebitda_quarterly']/1000000:.1f}M (EBIT + Depreciation)")
        
        # YoY Growth Rates
        print("\n📊 5. REVENUE YOY GROWTH:")
        results['revenue_yoy_growth'] = self.calculate_yoy_growth(facts_data, revenue_concepts, 'Revenue')
        
        print("\n📊 6. EPS YOY GROWTH:")
        # For EPS YoY, need to calculate EPS first then compare
        if results.get('net_income_quarterly'):
            # Get shares outstanding
            shares_concepts = ['WeightedAverageNumberOfSharesOutstandingBasic', 'CommonStockSharesOutstanding']
            # This would need similar quarterly calculation for shares
            print("  ⚠️ EPS YoY Growth: Need to implement shares calculation")
        
        print("\n📊 7. EBITDA YOY GROWTH:")
        if results.get('ebitda_quarterly'):
            # Calculate EBITDA YoY using same approach as revenue
            ebitda_concepts = ['OperatingIncomeLoss']  # Will calculate EBITDA from EBIT + D&A
            print("  ⚠️ EBITDA YoY Growth: Need to implement EBITDA YoY comparison")
        
        return results

def main():
    calculator = QuarterlyMetrics()
    
    if len(sys.argv) > 1:
        symbol = sys.argv[1].upper()
    else:
        symbol = 'HIMS'
    
    results = calculator.calculate_quarterly_metrics(symbol)
    
    if results:
        print(f"\n🎯 {symbol} QUARTERLY RESULTS:")
        for key, value in results.items():
            if value:
                if 'growth' in key:
                    print(f"{key}: {value:.2f}%")
                else:
                    print(f"{key}: ${value/1000000:.1f}M")
    else:
        print("❌ Failed to calculate quarterly metrics")

if __name__ == "__main__":
    main()
