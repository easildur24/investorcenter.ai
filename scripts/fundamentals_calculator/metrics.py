#!/usr/bin/env python3
"""
Metrics Calculator - All metric definitions and calculation logic
Each metric knows how to calculate itself using the bulletproof algorithm
"""

import requests
import json
from cik_manager import CIKManager

class MetricsCalculator:
    def __init__(self):
        self.base_url = "https://data.sec.gov/api/xbrl/companyfacts/"
        self.headers = {
            'User-Agent': 'InvestorCenter.ai financial-data-collector contact@investorcenter.ai'
        }
        self.cik_manager = CIKManager()

    def get_company_data(self, ticker):
        """Fetch company data from SEC EDGAR API"""
        cik = self.cik_manager.get_cik(ticker)
        if not cik:
            return None
        
        url = f"{self.base_url}CIK{cik}.json"
        
        try:
            print(f"📡 Calling SEC API: {url}")
            response = requests.get(url, headers=self.headers)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            print(f"❌ Error fetching data for {ticker}: {e}")
            return None

    def calculate_ttm_metric(self, facts_data, concepts, metric_name):
        """
        Calculate TTM metric using THE BULLETPROOF ALGORITHM:
        TTM = (Latest 3 10-Qs) + (Latest 10-K) - (3 10-Qs before 10-K)
        """
        us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
        
        # Get ALL data from all concepts (cross-concept search)
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
        
        # Get 10-Qs (ONLY quarterly frames, no YTD)
        all_10qs = [item for item in all_data 
                   if item.get('form') == '10-Q' 
                   and item.get('frame') and 'Q' in item.get('frame', '')
                   and item.get('end', '') >= '2020-01-01']
        
        # Remove duplicates (same end date and frame)
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
            # THE BULLETPROOF ALGORITHM
            latest_3_10qs = sorted_10qs[:3]
            
            # Find 3 10-Qs right BEFORE the 10-K
            tenk_end_date = latest_10k.get('end')
            before_10k_10qs = [q for q in sorted_10qs if q.get('end') < tenk_end_date]
            before_10k_3 = sorted(before_10k_10qs, key=lambda x: x.get('end', ''), reverse=True)[:3]
            
            # Calculate using THE ALGORITHM
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

    def calculate_revenue_ttm(self, facts_data):
        """
        Revenue TTM Calculation
        SEC Concepts: Try multiple revenue concepts and use the one with most recent data
        """
        print("📊 CALCULATING REVENUE TTM:")
        
        revenue_concepts = [
            'Revenues',
            'RevenueFromContractWithCustomerExcludingAssessedTax'
        ]
        
        return self.calculate_ttm_metric(facts_data, revenue_concepts, 'Revenue')

    def calculate_all_metrics(self, ticker):
        """Calculate all supported metrics for a ticker"""
        print(f"🔍 Fetching SEC data for {ticker}...")
        
        facts_data = self.get_company_data(ticker)
        if not facts_data:
            return None
        
        print(f"✅ Successfully fetched SEC data for {ticker}")
        
        results = {}
        
        # Calculate Revenue TTM (starting with just this one)
        results['revenue_ttm'] = self.calculate_revenue_ttm(facts_data)
        
        # TODO: Add more metrics here as we implement them
        # results['net_income_ttm'] = self.calculate_net_income_ttm(facts_data)
        # results['ebit_ttm'] = self.calculate_ebit_ttm(facts_data)
        # etc.
        
        return results
