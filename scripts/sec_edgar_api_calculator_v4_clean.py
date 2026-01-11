#!/usr/bin/env python3
"""
SEC EDGAR API Calculator v4 - EXACT MATCHES (Clean Version)
"""

import requests
import json
import sys
from datetime import datetime

class SECEdgarCalculatorV4:
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

    def get_exact_ttm_revenue(self, facts_data):
        """Get EXACT TTM revenue - handles both cases: all quarters available or missing Q4"""
        try:
            us_gaap = facts_data.get('facts', {}).get('us-gaap', {})
            
            # Try both revenue concepts and pick the one with most recent data
            revenue_concepts = ['Revenues', 'RevenueFromContractWithCustomerExcludingAssessedTax']
            
            best_concept = None
            best_latest_date = '1900-01-01'
            
            for concept in revenue_concepts:
                if concept in us_gaap:
                    usd_data = us_gaap[concept].get('units', {}).get('USD', [])
                    quarterly_data = [item for item in usd_data 
                                    if item.get('frame') and 'Q' in item.get('frame', '')
                                    and item.get('end', '') >= '2022-01-01']
                    
                    if quarterly_data:
                        latest_quarter = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)[0]
                        latest_date = latest_quarter.get('end', '')
                        
                        if latest_date > best_latest_date:
                            best_latest_date = latest_date
                            best_concept = concept
            
            if not best_concept:
                print("❌ No revenue concept found")
                return None
            
            print(f"  📊 Using {best_concept} (most recent data: {best_latest_date})")
            
            # Get data for the best concept
            usd_data = us_gaap[best_concept].get('units', {}).get('USD', [])
            
            # Get quarterly and annual data
            quarterly_data = [item for item in usd_data 
                            if item.get('frame') and 'Q' in item.get('frame', '')
                            and item.get('end', '') >= '2022-01-01']
            
            annual_data = [item for item in usd_data if item.get('form') == '10-K']
            
            if len(quarterly_data) >= 4:
                sorted_quarters = sorted(quarterly_data, key=lambda x: x.get('end', ''), reverse=True)
                
                print("  🔍 Debug: Recent quarters:")
                for i, q in enumerate(sorted_quarters[:4]):
                    value = q.get('val', 0) / 1000000000
                    end_date = q.get('end')
                    frame = q.get('frame')
                    print(f"    {i+1}. {end_date} = ${value:.2f}B - {frame}")
                
                # Check if we have all 4 recent quarters or need to calculate Q4
                q1_date = sorted_quarters[0].get('end', '')
                q4_date = sorted_quarters[3].get('end', '')
                
                q1_dt = datetime.strptime(q1_date, '%Y-%m-%d')
                q4_dt = datetime.strptime(q4_date, '%Y-%m-%d')
                months_diff = (q1_dt - q4_dt).days / 30.44
                
                print(f"  🔍 Debug: Q1 date: {q1_date}, Q4 date: {q4_date}, months_diff: {months_diff:.1f}")
                
                if months_diff <= 15:  # All 4 quarters are recent
                    # Simple sum of 4 quarters
                    ttm_revenue = sum(q.get('val', 0) for q in sorted_quarters[:4])
                    
                    print(f"  ✅ EXACT TTM REVENUE (4 quarters available):")
                    print(f"    TTM TOTAL: ${ttm_revenue/1000000000:.2f}B")
                    
                    return ttm_revenue
                else:
                    print(f"  🔍 Debug: Quarters too far apart ({months_diff:.1f} months), trying annual calculation...")
                else:
                    # Need to calculate missing Q4 from annual data (Tesla case)
                    if annual_data:
                        sorted_annual = sorted(annual_data, key=lambda x: x.get('end', ''), reverse=True)
                        latest_annual = sorted_annual[0]
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
                            print(f"    Q1: ${q1/1000000000:.2f}B")
                            print(f"    Q2: ${q2/1000000000:.2f}B")
                            print(f"    Q3: ${q3/1000000000:.2f}B")
                            print(f"    Q4 (calculated): ${calculated_q4/1000000000:.2f}B")
                            print(f"    TTM TOTAL: ${ttm_revenue/1000000000:.2f}B")
                            
                            return ttm_revenue
            
            return None
        except Exception as e:
            print(f"❌ Error calculating exact TTM revenue: {e}")
            return None

    def calculate_exact_metrics(self, symbol):
        """Calculate EXACT metrics"""
        print(f"\n🎯 EXACT CALCULATION FOR {symbol}")
        print("=" * 50)
        
        facts_data = self.get_company_data(symbol)
        if not facts_data:
            return False
        
        metrics = {}
        
        print("\n📊 REVENUE TTM - EXACT CALCULATION:")
        revenue_ttm = self.get_exact_ttm_revenue(facts_data)
        if revenue_ttm:
            metrics['revenue_ttm'] = revenue_ttm
            print(f"🎯 SUCCESS: Revenue TTM = ${revenue_ttm/1000000000:.2f}B")
        else:
            print("❌ FAILED to calculate exact revenue TTM")
        
        return metrics

def main():
    calculator = SECEdgarCalculatorV4()
    
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
