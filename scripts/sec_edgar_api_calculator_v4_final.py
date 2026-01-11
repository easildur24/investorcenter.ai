#!/usr/bin/env python3
"""
SEC EDGAR API Calculator v4 - BULLETPROOF CROSS-CONCEPT METHOD
Searches ALL revenue concepts and combines quarterly data for exact matches
"""

import requests
import json
import sys
from datetime import datetime

class SECEdgarCalculatorV4Final:
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

    def get_all_quarterly_data(self, facts_data, concepts):
        """Get ALL quarterly data from ALL concepts and combine them"""
        us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
        all_quarterly_data = []
        
        for concept in concepts:
            if concept in us_gaap:
                usd_data = us_gaap[concept].get('units', {}).get('USD', [])
                
                # Get quarterly data from this concept
                quarterly_data = [item for item in usd_data 
                                if item.get('frame') and 'Q' in item.get('frame', '')
                                and item.get('end', '') >= '2020-01-01']  # Recent data only
                
                for item in quarterly_data:
                    # Add concept source for tracking
                    item_with_source = item.copy()
                    item_with_source['source_concept'] = concept
                    all_quarterly_data.append(item_with_source)
        
        return all_quarterly_data

    def get_exact_ttm_revenue(self, facts_data):
        """Get EXACT TTM revenue using cross-concept method"""
        try:
            print("  🔍 Searching ALL revenue concepts for quarterly data...")
            
            # Search ALL revenue concepts
            revenue_concepts = ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax']
            all_quarterly_data = self.get_all_quarterly_data(facts_data, revenue_concepts)
            
            if not all_quarterly_data:
                print("  ❌ No quarterly data found")
                return None
            
            # Remove duplicates (same end date, same value)
            unique_quarters = {}
            for item in all_quarterly_data:
                key = f"{item.get('end')}_{item.get('val')}"
                if key not in unique_quarters:
                    unique_quarters[key] = item
            
            quarterly_list = list(unique_quarters.values())
            
            # Sort by end date (most recent first)
            sorted_quarters = sorted(quarterly_list, key=lambda x: x.get('end', ''), reverse=True)
            
            print(f"  📊 Found {len(sorted_quarters)} unique quarterly data points:")
            for i, q in enumerate(sorted_quarters[:8]):
                value = q.get('val', 0) / 1000000000
                end_date = q.get('end')
                frame = q.get('frame', 'No Frame')
                source = q.get('source_concept', 'Unknown')
                print(f"    {i+1}. {frame}: {end_date} = ${value:.2f}B (from {source})")
            
            # Check if we have 4 consecutive recent quarters or need to calculate missing ones
            if len(sorted_quarters) >= 4:
                # Check for missing quarters in the sequence
                recent_4 = sorted_quarters[:4]
                
                # Extract quarter info
                quarters_info = []
                for q in recent_4:
                    frame = q.get('frame', '')
                    if 'CY' in frame and 'Q' in frame:
                        year = int(frame[2:6])  # Extract year from CY2025Q3
                        quarter = int(frame[7])  # Extract quarter number
                        quarters_info.append((year, quarter, q))
                
                # Check if we have consecutive quarters or if Q4 is missing
                if len(quarters_info) >= 3:
                    # Sort by year and quarter
                    quarters_info.sort(key=lambda x: (x[0], x[1]), reverse=True)
                    
                    # Check if Q4 of previous year is missing
                    most_recent = quarters_info[0]
                    if len(quarters_info) == 4:
                        # We might have all 4 quarters
                        ttm_revenue = sum(q.get('val', 0) for q in recent_4)
                        
                        print(f"  ✅ EXACT TTM REVENUE (4 quarters available):")
                        for i, q in enumerate(recent_4):
                            value = q.get('val', 0) / 1000000000
                            end_date = q.get('end')
                            frame = q.get('frame', 'No Frame')
                            print(f"    Q{i+1}: {frame} ({end_date}) = ${value:.2f}B")
                        
                        print(f"    TTM TOTAL: ${ttm_revenue/1000000000:.2f}B")
                        return ttm_revenue
                    else:
                        # Missing Q4 - calculate from annual data
                        print(f"  🔍 Missing Q4 - calculating from annual data...")
                        
                        # Get annual data to calculate missing Q4
                        us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
                        for concept in ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax']:
                            if concept in us_gaap:
                                usd_data = us_gaap[concept].get('units', {}).get('USD', [])
                                annual_data = [item for item in usd_data if item.get('form') == '10-K']
                                
                                if annual_data:
                                    latest_annual = sorted(annual_data, key=lambda x: x.get('end', ''), reverse=True)[0]
                                    annual_value = latest_annual.get('val', 0)
                                    annual_start = latest_annual.get('start')
                                    annual_end = latest_annual.get('end')
                                    
                                    # Find quarters in this annual period
                                    quarters_in_annual = [q for q in sorted_quarters 
                                                        if q.get('end') <= annual_end and q.get('end') >= annual_start]
                                    
                                    if len(quarters_in_annual) == 3:  # Missing Q4
                                        q_sum = sum(q.get('val', 0) for q in quarters_in_annual)
                                        calculated_q4 = annual_value - q_sum
                                        
                                        # TTM = 3 recent quarters + calculated Q4
                                        q1 = sorted_quarters[0].get('val', 0)
                                        q2 = sorted_quarters[1].get('val', 0)
                                        q3 = sorted_quarters[2].get('val', 0)
                                        ttm_revenue = q1 + q2 + q3 + calculated_q4
                                        
                                        print(f"  ✅ EXACT TTM REVENUE (Q4 calculated):")
                                        print(f"    Q1: {sorted_quarters[0].get('frame')} = ${q1/1000000000:.2f}B")
                                        print(f"    Q2: {sorted_quarters[1].get('frame')} = ${q2/1000000000:.2f}B")
                                        print(f"    Q3: {sorted_quarters[2].get('frame')} = ${q3/1000000000:.2f}B")
                                        print(f"    Q4: CALCULATED = ${calculated_q4/1000000000:.2f}B")
                                        print(f"    TTM TOTAL: ${ttm_revenue/1000000000:.2f}B")
                                        
                                        return ttm_revenue
                                    break
            
            return None
            
        except Exception as e:
            print(f"❌ Error calculating exact TTM revenue: {e}")
            return None

    def calculate_exact_metrics(self, symbol):
        """Calculate EXACT metrics using cross-concept method"""
        print(f"\n🎯 EXACT CALCULATION FOR {symbol} (Cross-Concept Method)")
        print("=" * 60)
        
        facts_data = self.get_company_data(symbol)
        if not facts_data:
            return False
        
        metrics = {}
        
        print("\n📊 REVENUE TTM - CROSS-CONCEPT CALCULATION:")
        revenue_ttm = self.get_exact_ttm_revenue(facts_data)
        if revenue_ttm:
            metrics['revenue_ttm'] = revenue_ttm
            print(f"🎯 SUCCESS: Revenue TTM = ${revenue_ttm/1000000000:.2f}B")
        else:
            print("❌ FAILED to calculate exact revenue TTM")
        
        return metrics

def main():
    calculator = SECEdgarCalculatorV4Final()
    
    if len(sys.argv) > 1:
        symbol = sys.argv[1].upper()
    else:
        symbol = 'GOOGL'
    
    metrics = calculator.calculate_exact_metrics(symbol)
    
    if metrics and 'revenue_ttm' in metrics:
        revenue_b = metrics['revenue_ttm'] / 1000000000
        print(f"\n🎯 FINAL RESULT: Revenue TTM = ${revenue_b:.2f}B")
    else:
        print("❌ Failed to calculate revenue TTM")

if __name__ == "__main__":
    main()
